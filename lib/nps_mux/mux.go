package nps_mux

import (
	"errors"
	"io"
	"log"
	"math"
	"net"
	"os"
	"sync/atomic"
	"time"
)

const (
	muxPingFlag uint8 = iota
	muxNewConnOk
	muxNewConnFail
	muxNewMsg
	muxNewMsgPart
	muxMsgSendOk
	muxNewConn
	muxConnClose
	muxPingReturn
	muxPing            int32 = -1
	maximumSegmentSize       = poolSizeWindow
	maximumWindowSize        = 1 << 27 // 1<<31-1 TCP slide window size is very large,
	// we use 128M, reduce memory usage
)

type Mux struct {
	latency uint64 // we store latency in bits, but it's float64
	net.Listener
	conn               net.Conn
	connMap            *connMap
	newConnCh          chan *conn
	id                 int32
	closeChan          chan struct{}
	isClose            atomic.Bool
	counter            *latencyCounter
	bw                 *bandwidth
	pingCh             chan []byte
	pingCheckTime      uint32 // we check the ping per 5s
	pingCheckThreshold uint32
	connType           string
	writeQueue         priorityQueue
	newConnQueue       connQueue
}

func NewMux(c net.Conn, connType string, pingCheckThreshold int) *Mux {
	//c.(*net.TCPConn).SetReadBuffer(0)
	//c.(*net.TCPConn).SetWriteBuffer(0)
	fd, err := getConnFd(c)
	if err != nil {
		log.Println(err)
	}
	var checkThreshold uint32
	if pingCheckThreshold <= 0 {
		if connType == "kcp" {
			checkThreshold = 20
		} else {
			checkThreshold = 60
		}
	} else {
		checkThreshold = uint32(pingCheckThreshold)
	}
	m := &Mux{
		conn:               c,
		connMap:            NewConnMap(),
		id:                 0,
		closeChan:          make(chan struct{}, 1),
		newConnCh:          make(chan *conn),
		bw:                 NewBandwidth(fd),
		connType:           connType,
		pingCh:             make(chan []byte),
		pingCheckThreshold: checkThreshold,
		counter:            newLatencyCounter(),
	}
	m.writeQueue.New()
	m.newConnQueue.New()
	//read session by flag
	m.readSession()
	//ping
	m.ping()
	m.writeSession()
	return m
}

// IsClose returns whether the mux connection has been closed.
func (s *Mux) IsClose() bool { return s.isClose.Load() }

func (s *Mux) NewConn() (*conn, error) {
	if s.isClose.Load() {
		return nil, errors.New("the mux has closed")
	}
	conn := NewConn(s.getId(), s)
	//it must be Set before send
	s.connMap.Set(conn.connId, conn)
	s.sendInfo(muxNewConn, conn.connId, nil)
	//Set a timer timeout 120 second
	timer := time.NewTimer(time.Minute * 2)
	defer timer.Stop()
	select {
	case <-conn.connStatusOkCh:
		return conn, nil
	case <-timer.C:
	}
	return nil, errors.New("create connection fail，the server refused the connection")
}

func (s *Mux) Accept() (net.Conn, error) {
	if s.isClose.Load() {
		return nil, errors.New("accpet error,the mux has closed")
	}
	conn := <-s.newConnCh
	if conn == nil {
		return nil, errors.New("accpet error,the conn has closed")
	}
	return conn, nil
}

func (s *Mux) Addr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *Mux) sendInfo(flag uint8, id int32, data interface{}) {
	if s.isClose.Load() {
		return
	}
	var err error
	pack := muxPack.Get()
	err = pack.Set(flag, id, data)
	if err != nil {
		muxPack.Put(pack)
		log.Println("mux: New Pack err", err)
		_ = s.Close()
		return
	}
	s.writeQueue.Push(pack)
	return
}

func (s *Mux) writeSession() {
	go func() {
		for {
			if s.isClose.Load() {
				break
			}
			pack := s.writeQueue.Pop()
			if s.isClose.Load() {
				break
			}
			//if pack.flag == muxNewMsg || pack.flag == muxNewMsgPart {
			//	if pack.length >= 100 {
			//		log.Println("write session id", pack.id, "\n", string(pack.content[:100]))
			//	} else {
			//		log.Println("write session id", pack.id, "\n", string(pack.content[:pack.length]))
			//	}
			//}
			err := pack.Pack(s.conn)
			muxPack.Put(pack)
			if err != nil {
				log.Println("mux: Pack err", err)
				_ = s.Close()
				break
			}
		}
	}()
}

func (s *Mux) ping() {
	go func() {
		now, _ := time.Now().UTC().MarshalText()
		s.sendInfo(muxPingFlag, muxPing, now)
		// send the ping flag and Get the latency first
		ticker := time.NewTicker(time.Second * 5)
		defer ticker.Stop()
		for {
			if s.isClose.Load() {
				break
			}
			select {
			case <-ticker.C:
			}
			if atomic.LoadUint32(&s.pingCheckTime) > s.pingCheckThreshold {
				log.Println("mux: ping time out, checktime", s.pingCheckTime, "threshold", s.pingCheckThreshold)
				_ = s.Close()
				// more than limit times not receive the ping return package,
				// mux conn is damaged, maybe a packet drop, close it
				break
			}
			now, _ = time.Now().UTC().MarshalText()
			s.sendInfo(muxPingFlag, muxPing, now)
			atomic.AddUint32(&s.pingCheckTime, 1)
		}
		return
	}()

	go func() {
		var now time.Time
		var data []byte
		for {
			if s.isClose.Load() {
				break
			}
			select {
			case data = <-s.pingCh:
				atomic.StoreUint32(&s.pingCheckTime, 0)
			case <-s.closeChan:
				break
			}
			_ = now.UnmarshalText(data)
			latency := time.Now().UTC().Sub(now).Seconds()
			if latency > 0 {
				atomic.StoreUint64(&s.latency, math.Float64bits(s.counter.Latency(latency)))
				// convert float64 to bits, store it atomic
				//log.Println("ping", math.Float64frombits(atomic.LoadUint64(&s.latency)))
			}
			if cap(data) > 0 && !s.isClose.Load() {
				windowBuff.Put(data)
			}
		}
	}()
}

func (s *Mux) readSession() {
	go func() {
		var connection *conn
		for {
			if s.isClose.Load() {
				break
			}
			connection = s.newConnQueue.Pop()
			if s.isClose.Load() {
				break // make sure that is closed
			}
			s.connMap.Set(connection.connId, connection) //it has been Set before send ok
			s.newConnCh <- connection
			s.sendInfo(muxNewConnOk, connection.connId, nil)
		}
	}()
	go func() {
		var pack *muxPackager
		var l uint16
		var err error
		for {
			if s.isClose.Load() {
				return
			}
			pack = muxPack.Get()
			s.bw.StartRead()
			if l, err = pack.UnPack(s.conn); err != nil {
				log.Println("mux: read session unpack from connection err", err)
				_ = s.Close()
				break
			}
			s.bw.SetCopySize(l)
			//if pack.flag == muxNewMsg || pack.flag == muxNewMsgPart {
			//	if pack.length >= 100 {
			//		log.Printf("read session id %d pointer %p\n%v", pack.id, pack.content, string(pack.content[:100]))
			//	} else {
			//		log.Printf("read session id %d pointer %p\n%v", pack.id, pack.content, string(pack.content[:pack.length]))
			//	}
			//}
			switch pack.flag {
			case muxNewConn: //New connection
				connection := NewConn(pack.id, s)
				s.newConnQueue.Push(connection)
				continue
			case muxPingFlag: //ping
				s.sendInfo(muxPingReturn, muxPing, pack.content)
				windowBuff.Put(pack.content)
				continue
			case muxPingReturn:
				s.pingCh <- pack.content
				continue
			}
			if connection, ok := s.connMap.Get(pack.id); ok && !connection.isClose {
				switch pack.flag {
				case muxNewMsg, muxNewMsgPart: //New msg from remote connection
					err = s.newMsg(connection, pack)
					if err != nil {
						log.Println("mux: read session connection New msg err", err)
						_ = connection.Close()
					}
					continue
				case muxNewConnOk: //connection ok
					connection.connStatusOkCh <- struct{}{}
					continue
				case muxNewConnFail:
					connection.connStatusFailCh <- struct{}{}
					continue
				case muxMsgSendOk:
					if connection.isClose {
						continue
					}
					connection.sendWindow.SetSize(pack.window)
					continue
				case muxConnClose: //close the connection
					connection.closingFlag = true
					connection.receiveWindow.Stop() // close signal to receive window
					continue
				}
			} else if pack.flag == muxConnClose {
				continue
			}
			muxPack.Put(pack)
		}
	}()
}

func (s *Mux) newMsg(connection *conn, pack *muxPackager) (err error) {
	if connection.isClose {
		err = io.ErrClosedPipe
		return
	}
	//insert into queue
	if pack.flag == muxNewMsgPart {
		err = connection.receiveWindow.Write(pack.content, pack.length, true, pack.id)
	}
	if pack.flag == muxNewMsg {
		err = connection.receiveWindow.Write(pack.content, pack.length, false, pack.id)
	}
	return
}

func (s *Mux) Close() (err error) {
	if s.isClose.Load() {
		return errors.New("the mux has closed")
	}
	s.isClose.Store(true)
	log.Println("close mux")
	s.connMap.Close()
	//s.connMap = nil
	s.closeChan <- struct{}{}
	close(s.newConnCh)
	// while target host close socket without finish steps, conn.Close method maybe blocked
	// and tcp status change to CLOSE WAIT or TIME WAIT, so we close it in other goroutine
	_ = s.conn.SetDeadline(time.Now().Add(time.Second * 5))
	go s.conn.Close()
	s.release()
	return
}

func (s *Mux) release() {
	for {
		pack := s.writeQueue.TryPop()
		if pack == nil {
			break
		}
		if pack.basePackager.content != nil {
			windowBuff.Put(pack.basePackager.content)
		}
		muxPack.Put(pack)
	}
	for {
		connection := s.newConnQueue.TryPop()
		if connection == nil {
			break
		}
		connection = nil
	}
	s.writeQueue.Stop()
	s.newConnQueue.Stop()
}

// Get New connId as unique flag
func (s *Mux) getId() (id int32) {
	//Avoid going beyond the scope
	if (math.MaxInt32 - s.id) < 10000 {
		atomic.StoreInt32(&s.id, 0)
	}
	id = atomic.AddInt32(&s.id, 1)
	if _, ok := s.connMap.Get(id); ok {
		return s.getId()
	}
	return
}

type bandwidth struct {
	readBandwidth uint64 // store in bits, but it's float64
	readStart     time.Time
	lastReadStart time.Time
	bufLength     uint32
	fd            *os.File
	calcThreshold uint32
}

func NewBandwidth(fd *os.File) *bandwidth {
	return &bandwidth{fd: fd}
}

func (Self *bandwidth) StartRead() {
	if Self.readStart.IsZero() {
		Self.readStart = time.Now()
	}
	if Self.bufLength >= Self.calcThreshold {
		Self.lastReadStart, Self.readStart = Self.readStart, time.Now()
		Self.calcBandWidth()
	}
}

func (Self *bandwidth) SetCopySize(n uint16) {
	Self.bufLength += uint32(n)
}

func (Self *bandwidth) calcBandWidth() {
	t := Self.readStart.Sub(Self.lastReadStart)
	bufferSize, err := sysGetSock(Self.fd)
	if err != nil {
		log.Println(err)
		Self.bufLength = 0
		return
	}
	if Self.bufLength >= uint32(bufferSize) {
		atomic.StoreUint64(&Self.readBandwidth, math.Float64bits(float64(Self.bufLength)/t.Seconds()))
		// calculate the whole socket buffer, the time meaning to fill the buffer
	} else {
		Self.calcThreshold = uint32(bufferSize)
	}
	// socket buffer size is bigger than bufLength, so we don't calculate it
	Self.bufLength = 0
}

func (Self *bandwidth) Get() (bw float64) {
	// The zero value, 0 for numeric types
	bw = math.Float64frombits(atomic.LoadUint64(&Self.readBandwidth))
	if bw <= 0 {
		bw = 0
	}
	return
}

const counterBits = 4
const counterMask = 1<<counterBits - 1

func newLatencyCounter() *latencyCounter {
	return &latencyCounter{
		buf:     make([]float64, 1<<counterBits, 1<<counterBits),
		headMin: 0,
	}
}

type latencyCounter struct {
	buf []float64 //buf is a fixed length ring buffer,
	// if buffer is full, New value will replace the oldest one.
	headMin uint8 //head indicate the head in ring buffer,
	// in meaning, slot in list will be replaced;
	// min indicate this slot value is minimal in list.

	// we delineate the effective range with three times the minimum latency
	// average of effective latency for all current data as a mux latency
}

func (Self *latencyCounter) unpack(idxs uint8) (head, min uint8) {
	head = (idxs >> counterBits) & counterMask
	// we Set head is 4 bits
	min = idxs & counterMask
	return
}

func (Self *latencyCounter) pack(head, min uint8) uint8 {
	return head<<counterBits |
		min&counterMask
}

func (Self *latencyCounter) add(value float64) {
	head, min := Self.unpack(Self.headMin)
	Self.buf[head] = value
	if head == min {
		min = Self.minimal()
		//if head equals min, means the min slot already be replaced,
		// so we need to find another minimal value in the list,
		// and change the min indicator
	}
	if Self.buf[min] > value {
		min = head
	}
	head++
	Self.headMin = Self.pack(head, min)
}

func (Self *latencyCounter) minimal() (min uint8) {
	var val float64
	var i uint8
	for i = 0; i < counterMask; i++ {
		if Self.buf[i] > 0 {
			if val > Self.buf[i] {
				val = Self.buf[i]
				min = i
			}
		}
	}
	return
}

func (Self *latencyCounter) Latency(value float64) (latency float64) {
	Self.add(value)
	latency = Self.countSuccess()
	return
}

const lossRatio = 3

func (Self *latencyCounter) countSuccess() (successRate float64) {
	var i, success uint8
	_, min := Self.unpack(Self.headMin)
	for i = 0; i < counterMask; i++ {
		if Self.buf[i] <= lossRatio*Self.buf[min] && Self.buf[i] > 0 {
			success++
			successRate += Self.buf[i]
		}
	}
	// counting all the data in the ring buf, except zero
	successRate = successRate / float64(success)
	return
}
