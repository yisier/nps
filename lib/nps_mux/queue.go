package nps_mux

import (
	"errors"
	"io"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

type priorityQueue struct {
	highestChain *bufChain
	middleChain  *bufChain
	lowestChain  *bufChain
	starving     uint8
	stop         bool
	cond         *sync.Cond
}

func (Self *priorityQueue) New() {
	Self.highestChain = new(bufChain)
	Self.highestChain.new(4)
	Self.middleChain = new(bufChain)
	Self.middleChain.new(32)
	Self.lowestChain = new(bufChain)
	Self.lowestChain.new(256)
	locker := new(sync.Mutex)
	Self.cond = sync.NewCond(locker)
}

func (Self *priorityQueue) Push(packager *muxPackager) {
	Self.push(packager)
	Self.cond.Broadcast()
	return
}

func (Self *priorityQueue) push(packager *muxPackager) {
	switch packager.flag {
	case muxPingFlag, muxPingReturn:
		Self.highestChain.pushHead(unsafe.Pointer(packager))
	// the ping package need highest priority
	// prevent ping calculation error
	case muxNewConn, muxNewConnOk, muxNewConnFail:
		// the New conn package need some priority too
		Self.middleChain.pushHead(unsafe.Pointer(packager))
	default:
		Self.lowestChain.pushHead(unsafe.Pointer(packager))
	}
}

const maxStarving uint8 = 8

func (Self *priorityQueue) Pop() (packager *muxPackager) {
	var iter bool
	for {
		packager = Self.TryPop()
		if packager != nil {
			return
		}
		if Self.stop {
			return
		}
		if iter {
			break
			// trying to pop twice
		}
		iter = true
		runtime.Gosched()
	}
	Self.cond.L.Lock()
	defer Self.cond.L.Unlock()
	for packager = Self.TryPop(); packager == nil; {
		if Self.stop {
			return
		}
		Self.cond.Wait()
		// wait for it with no more iter
		packager = Self.TryPop()
	}
	return
}

func (Self *priorityQueue) TryPop() (packager *muxPackager) {
	ptr, ok := Self.highestChain.popTail()
	if ok {
		packager = (*muxPackager)(ptr)
		return
	}
	if Self.starving < maxStarving {
		// not pop too much, lowestChain will wait too long
		ptr, ok = Self.middleChain.popTail()
		if ok {
			packager = (*muxPackager)(ptr)
			Self.starving++
			return
		}
	}
	ptr, ok = Self.lowestChain.popTail()
	if ok {
		packager = (*muxPackager)(ptr)
		if Self.starving > 0 {
			Self.starving = Self.starving / 2
		}
		return
	}
	if Self.starving > 0 {
		ptr, ok = Self.middleChain.popTail()
		if ok {
			packager = (*muxPackager)(ptr)
			Self.starving++
			return
		}
	}
	return
}

func (Self *priorityQueue) Stop() {
	Self.stop = true
	Self.cond.Broadcast()
}

type connQueue struct {
	chain    *bufChain
	starving uint8
	stop     bool
	cond     *sync.Cond
}

func (Self *connQueue) New() {
	Self.chain = new(bufChain)
	Self.chain.new(32)
	locker := new(sync.Mutex)
	Self.cond = sync.NewCond(locker)
}

func (Self *connQueue) Push(connection *conn) {
	Self.chain.pushHead(unsafe.Pointer(connection))
	Self.cond.Broadcast()
	return
}

func (Self *connQueue) Pop() (connection *conn) {
	var iter bool
	for {
		connection = Self.TryPop()
		if connection != nil {
			return
		}
		if Self.stop {
			return
		}
		if iter {
			break
			// trying to pop twice
		}
		iter = true
		runtime.Gosched()
	}
	Self.cond.L.Lock()
	defer Self.cond.L.Unlock()
	for connection = Self.TryPop(); connection == nil; {
		if Self.stop {
			return
		}
		Self.cond.Wait()
		// wait for it with no more iter
		connection = Self.TryPop()
	}
	return
}

func (Self *connQueue) TryPop() (connection *conn) {
	ptr, ok := Self.chain.popTail()
	if ok {
		connection = (*conn)(ptr)
		return
	}
	return
}

func (Self *connQueue) Stop() {
	Self.stop = true
	Self.cond.Broadcast()
}

type listElement struct {
	Buf  []byte
	L    uint16
	Part bool
}

func (Self *listElement) Reset() {
	Self.L = 0
	Self.Buf = nil
	Self.Part = false
}

func newListElement(buf []byte, l uint16, part bool) (element *listElement, err error) {
	if uint16(len(buf)) != l {
		err = errors.New("listElement: buf length not match")
		return
	}
	element = listEle.Get()
	element.Buf = buf
	element.L = l
	element.Part = part
	return
}

type receiveWindowQueue struct {
	lengthWait uint64
	chain      *bufChain
	stopOp     chan struct{}
	readOp     chan struct{}
	// https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	// On non-Linux ARM, the 64-bit functions use instructions unavailable before the ARMv6k core.
	// On ARM, x86-32, and 32-bit MIPS, it is the caller's responsibility
	// to arrange for 64-bit alignment of 64-bit words accessed atomically.
	// The first word in a variable or in an allocated struct, array, or slice can be relied upon to be 64-bit aligned.

	// if there are implicit struct, careful the first word
	timeout time.Time
}

func newReceiveWindowQueue() *receiveWindowQueue {
	queue := receiveWindowQueue{
		chain:  new(bufChain),
		stopOp: make(chan struct{}, 2),
		readOp: make(chan struct{}),
	}
	queue.chain.new(64)
	return &queue
}

func (Self *receiveWindowQueue) Push(element *listElement) {
	var length, wait uint32
	for {
		ptrs := atomic.LoadUint64(&Self.lengthWait)
		length, wait = Self.chain.head.unpack(ptrs)
		length += uint32(element.L)
		if atomic.CompareAndSwapUint64(&Self.lengthWait, ptrs, Self.chain.head.pack(length, 0)) {
			break
		}
		// another goroutine change the length or into wait, make sure
	}
	Self.chain.pushHead(unsafe.Pointer(element))
	if wait == 1 {
		Self.allowPop()
	}
	return
}

func (Self *receiveWindowQueue) Pop() (element *listElement, err error) {
	var length uint32
startPop:
	ptrs := atomic.LoadUint64(&Self.lengthWait)
	length, _ = Self.chain.head.unpack(ptrs)
	if length == 0 {
		if !atomic.CompareAndSwapUint64(&Self.lengthWait, ptrs, Self.chain.head.pack(0, 1)) {
			goto startPop // another goroutine is pushing
		}
		err = Self.waitPush()
		// there is no more data in queue, wait for it
		if err != nil {
			return
		}
		goto startPop // wait finish, trying to Get the New status
	}
	// length is not zero, so try to pop
	for {
		element = Self.TryPop()
		if element != nil {
			return
		}
		// the element counted in length may have been taken by release() when the
		// window was closed concurrently, re-check the length instead of spinning forever
		if length, _ = Self.chain.head.unpack(atomic.LoadUint64(&Self.lengthWait)); length == 0 {
			goto startPop
		}
		runtime.Gosched() // another goroutine is still pushing
	}
}

func (Self *receiveWindowQueue) TryPop() (element *listElement) {
	ptr, ok := Self.chain.popTail()
	if ok {
		element = (*listElement)(ptr)
		atomic.AddUint64(&Self.lengthWait, ^(uint64(element.L)<<dequeueBits - 1))
		return
	}
	return nil
}

func (Self *receiveWindowQueue) allowPop() (closed bool) {
	select {
	case Self.readOp <- struct{}{}:
		return false
	case <-Self.stopOp:
		return true
	}
}

func (Self *receiveWindowQueue) waitPush() (err error) {
	t := Self.timeout.Sub(time.Now())
	if t <= 0 {
		// not Set the timeout, so wait for it without timeout, just like a tcp connection
		select {
		case <-Self.readOp:
			return nil
		case <-Self.stopOp:
			err = io.EOF
			return
		}
	}
	timer := time.NewTimer(t)
	defer timer.Stop()
	select {
	case <-Self.readOp:
		return nil
	case <-Self.stopOp:
		err = io.EOF
		return
	case <-timer.C:
		err = errors.New("mux.queue: read time out")
		return
	}
}

func (Self *receiveWindowQueue) Len() (n uint32) {
	ptrs := atomic.LoadUint64(&Self.lengthWait)
	n, _ = Self.chain.head.unpack(ptrs)
	// just for unpack method use
	return
}

func (Self *receiveWindowQueue) Stop() {
	Self.stopOp <- struct{}{}
	Self.stopOp <- struct{}{}
}

func (Self *receiveWindowQueue) SetTimeOut(t time.Time) {
	Self.timeout = t
}

// https://golang.org/src/sync/poolqueue.go

type bufDequeue struct {
	// headTail packs together a 32-bit head index and a 32-bit
	// tail index. Both are indexes into vals modulo len(vals)-1.
	//
	// tail = index of oldest data in queue
	// head = index of next slot to fill
	//
	// Slots in the range [tail, head) are owned by consumers.
	// A consumer continues to own a slot outside this range until
	// it nils the slot, at which point ownership passes to the
	// producer.
	//
	// The head index is stored in the most-significant bits so
	// that we can atomically add to it and the overflow is
	// harmless.
	headTail uint64

	// vals is a ring buffer of interface{} values stored in this
	// dequeue. The size of this must be a power of 2.
	//
	// A slot is still in use until *both* the tail
	// index has moved beyond it and typ has been Set to nil. This
	// is Set to nil atomically by the consumer and read
	// atomically by the producer.
	vals     []unsafe.Pointer
	starving uint32
}

const dequeueBits = 32

// dequeueLimit is the maximum size of a bufDequeue.
//
// This must be at most (1<<dequeueBits)/2 because detecting fullness
// depends on wrapping around the ring buffer without wrapping around
// the index. We divide by 4 so this fits in an int on 32-bit.
const dequeueLimit = (1 << dequeueBits) / 4

func (d *bufDequeue) unpack(ptrs uint64) (head, tail uint32) {
	const mask = 1<<dequeueBits - 1
	head = uint32((ptrs >> dequeueBits) & mask)
	tail = uint32(ptrs & mask)
	return
}

func (d *bufDequeue) pack(head, tail uint32) uint64 {
	const mask = 1<<dequeueBits - 1
	return (uint64(head) << dequeueBits) |
		uint64(tail&mask)
}

// pushHead adds val at the head of the queue. It returns false if the
// queue is full.
func (d *bufDequeue) pushHead(val unsafe.Pointer) bool {
	var slot *unsafe.Pointer
	var starve uint8
	if atomic.LoadUint32(&d.starving) > 0 {
		runtime.Gosched()
	}
	for {
		ptrs := atomic.LoadUint64(&d.headTail)
		head, tail := d.unpack(ptrs)
		if (tail+uint32(len(d.vals)))&(1<<dequeueBits-1) == head {
			// Queue is full.
			return false
		}
		ptrs2 := d.pack(head+1, tail)
		if atomic.CompareAndSwapUint64(&d.headTail, ptrs, ptrs2) {
			slot = &d.vals[head&uint32(len(d.vals)-1)]
			if starve >= 3 && atomic.LoadUint32(&d.starving) > 0 {
				atomic.StoreUint32(&d.starving, 0)
			}
			break
		}
		starve++
		if starve >= 3 {
			atomic.StoreUint32(&d.starving, 1)
		}
	}
	// The head slot is free, so we own it.
	*slot = val
	return true
}

// popTail removes and returns the element at the tail of the queue.
// It returns false if the queue is empty. It may be called by any
// number of consumers.
func (d *bufDequeue) popTail() (unsafe.Pointer, bool) {
	var val unsafe.Pointer
	var head, tail uint32
	for {
		ptrs := atomic.LoadUint64(&d.headTail)
		head, tail = d.unpack(ptrs)
		if tail == head {
			// Queue is empty.
			return nil, false
		}
		slot := &d.vals[tail&uint32(len(d.vals)-1)]
		val = atomic.LoadPointer(slot)
		if val != nil {
			// We now get a slot.
			if atomic.CompareAndSwapPointer(slot, val, nil) {
				break
				// Tell pushHead that we're done with this slot. Zeroing the
				// slot is also important so we don't leave behind references
				// that could keep this object live longer than necessary.
				//
				// We write to val first and then publish that we're done with
			}
		}
		// Maybe the value was taken by other goroutine or not push yet.
	}
	// At this point pushHead owns the slot.
	if tail < math.MaxUint32 {
		atomic.AddUint64(&d.headTail, 1)
	} else {
		atomic.AddUint64(&d.headTail, ^uint64(math.MaxUint32-1))
	}
	return val, true
}

// bufChain is a dynamically-sized version of bufDequeue.
//
// This is implemented as a doubly-linked list queue of poolDequeues
// where each dequeue is double the size of the previous one. Once a
// dequeue fills up, this allocates a New one and only ever pushes to
// the latest dequeue. Pops happen from the other end of the list and
// once a dequeue is exhausted, it gets removed from the list.
type bufChain struct {
	// head is the bufDequeue to push to. This is only accessed
	// by the producer, so doesn't need to be synchronized.
	head *bufChainElt

	// tail is the bufDequeue to popTail from. This is accessed
	// by consumers, so reads and writes must be atomic.
	tail     *bufChainElt
	newChain uint32
}

type bufChainElt struct {
	bufDequeue

	// next and prev link to the adjacent poolChainElts in this
	// bufChain.
	//
	// next is written atomically by the producer and read
	// atomically by the consumer. It only transitions from nil to
	// non-nil.
	//
	// prev is written atomically by the consumer and read
	// atomically by the producer. It only transitions from
	// non-nil to nil.
	next, prev *bufChainElt
}

func storePoolChainElt(pp **bufChainElt, v *bufChainElt) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(pp)), unsafe.Pointer(v))
}

func loadPoolChainElt(pp **bufChainElt) *bufChainElt {
	return (*bufChainElt)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(pp))))
}

func (c *bufChain) new(initSize int) {
	// Initialize the chain.
	// initSize must be a power of 2
	d := new(bufChainElt)
	d.vals = make([]unsafe.Pointer, initSize)
	storePoolChainElt(&c.head, d)
	storePoolChainElt(&c.tail, d)
}

func (c *bufChain) pushHead(val unsafe.Pointer) {
startPush:
	for {
		if atomic.LoadUint32(&c.newChain) > 0 {
			runtime.Gosched()
		} else {
			break
		}
	}

	d := loadPoolChainElt(&c.head)

	if d.pushHead(val) {
		return
	}

	// The current dequeue is full. Allocate a New one of twice
	// the size.
	if atomic.CompareAndSwapUint32(&c.newChain, 0, 1) {
		newSize := len(d.vals) * 2
		if newSize >= dequeueLimit {
			// Can't make it any bigger.
			newSize = dequeueLimit
		}

		d2 := &bufChainElt{prev: d}
		d2.vals = make([]unsafe.Pointer, newSize)
		d2.pushHead(val)
		storePoolChainElt(&c.head, d2)
		storePoolChainElt(&d.next, d2)
		atomic.StoreUint32(&c.newChain, 0)
		return
	}
	goto startPush
}

func (c *bufChain) popTail() (unsafe.Pointer, bool) {
	d := loadPoolChainElt(&c.tail)
	if d == nil {
		return nil, false
	}

	for {
		// It's important that we load the next pointer
		// *before* popping the tail. In general, d may be
		// transiently empty, but if next is non-nil before
		// the TryPop and the TryPop fails, then d is permanently
		// empty, which is the only condition under which it's
		// safe to drop d from the chain.
		d2 := loadPoolChainElt(&d.next)

		if val, ok := d.popTail(); ok {
			return val, ok
		}

		if d2 == nil {
			// This is the only dequeue. It's empty right
			// now, but could be pushed to in the future.
			return nil, false
		}

		// The tail of the chain has been drained, so move on
		// to the next dequeue. Try to drop it from the chain
		// so the next TryPop doesn't have to look at the empty
		// dequeue again.
		if atomic.CompareAndSwapPointer((*unsafe.Pointer)(unsafe.Pointer(&c.tail)), unsafe.Pointer(d), unsafe.Pointer(d2)) {
			// We won the race. Clear the prev pointer so
			// the garbage collector can collect the empty
			// dequeue and so popHead doesn't back up
			// further than necessary.
			storePoolChainElt(&d2.prev, nil)
		}
		d = d2
	}
}
