package rate

import (
	"context"
	"fmt"
	"time"

	xrate "golang.org/x/time/rate"
)

// Rate 使用 golang.org/x/time/rate 实现的字节令牌桶限速器
// 保持原有对外接口：NewRate、Start、Get、ReturnBucket、Stop、NowRate
type Rate struct {
	limiter  *xrate.Limiter
	addSize  int64
	stopChan chan struct{}
	//  暂时表示配置的每秒速率（字节/秒），用于展示
	NowRate int64
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
	// 保留一个 goroutine 以兼容原有 Stop 调用，用于未来扩展（例如监控）
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// 目前将 NowRate 保持为配置速率；可以在此处扩展动态统计
				s.NowRate = s.addSize - (int64(s.limiter.Tokens()))
				//fmt.Printf("token left 2: %f\n", s.limiter.Tokens())
			case <-s.stopChan:
				return
			}
		}
	}()
}

// 停止内部 goroutine
func (s *Rate) Stop() {
	select {
	case <-s.stopChan:
		// already closed
	default:
		close(s.stopChan)
	}
}

// Get 阻塞直到获得 size 字节的令牌
func (s *Rate) Get(size int64) {
	if s.addSize <= 0 {
		return
	}
	if size <= 0 {
		return
	}
	// 使用背景上下文直接等待，必要时可替换为带超时的 Context
	ctx := context.Background()
	// x/time/rate 的 WaitN 接受 int
	_ = s.limiter.WaitN(ctx, int(size))
	s.NowRate = s.addSize - (int64(s.limiter.Tokens()))
	fmt.Printf("get size: %d\n,  token left: %f\n", size, s.limiter.Tokens())
}
