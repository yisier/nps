package rate

import (
	"context"
	"sync/atomic"
	"time"

	xrate "golang.org/x/time/rate"
)

type Rate struct {
	limiter  *xrate.Limiter
	addSize  int64
	stopChan chan struct{}
	consumed int64
	NowRate  int64
}

func NewRate(addSize int64) *Rate {
	return &Rate{
		limiter:  xrate.NewLimiter(xrate.Limit(addSize), int(addSize)),
		addSize:  addSize,
		stopChan: make(chan struct{}),
	}
}

func (s *Rate) Start() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.NowRate = atomic.SwapInt64(&s.consumed, 0)
			case <-s.stopChan:
				return
			}
		}
	}()
}

func (s *Rate) Stop() {
	select {
	case <-s.stopChan:
	default:
		close(s.stopChan)
	}
}

func (s *Rate) Get(size int64) {
	if s.addSize <= 0 {
		return
	}
	if size <= 0 {
		return
	}
	ctx := context.Background()
	_ = s.limiter.WaitN(ctx, int(size))
	atomic.AddInt64(&s.consumed, size)
}
