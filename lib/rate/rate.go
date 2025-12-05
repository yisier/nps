package rate

import (
	"context"
	"time"

	xrate "golang.org/x/time/rate"
)

type Rate struct {
	limiter  *xrate.Limiter
	addSize  int64
	stopChan chan struct{}
	NowRate  int64
}

func NewRate(addSize int64) *Rate {
	return &Rate{
		limiter:  xrate.NewLimiter(xrate.Limit(addSize), int(addSize)),
		addSize:  addSize,
		stopChan: make(chan struct{}),
		NowRate:  addSize,
	}
}

func (s *Rate) Start() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.NowRate = s.addSize - (int64(s.limiter.Tokens()))
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
	s.NowRate = s.addSize - (int64(s.limiter.Tokens()))
}
