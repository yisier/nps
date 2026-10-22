package nps_mux

import (
	"io"
	"sync/atomic"
	"testing"
	"time"
)

// A reader that has already seen an element counted in length must not spin
// forever when release() takes that element first because the conn was closed.
func TestReceiveWindowQueuePopAfterConcurrentRelease(t *testing.T) {
	q := newReceiveWindowQueue()
	// the reader sees one element in the queue...
	atomic.StoreUint64(&q.lengthWait, q.chain.head.pack(1, 0))
	done := make(chan error, 1)
	go func() {
		_, err := q.Pop()
		done <- err
	}()
	time.Sleep(10 * time.Millisecond)
	// ...but the window is closed and release() drains that element first
	q.Stop()
	atomic.StoreUint64(&q.lengthWait, 0)
	select {
	case err := <-done:
		if err != io.EOF {
			t.Fatalf("Pop returned %v, want io.EOF", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Pop is still spinning after the queue was stopped and drained")
	}
}
