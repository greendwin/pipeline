package pipeline

import (
	"context"
	"sync"
)

type contextKey int

const waitGroupKey contextKey = 0

func NewPipeline(parent context.Context) (context.Context, context.CancelFunc) {
	ctxWg := context.WithValue(parent, waitGroupKey, &sync.WaitGroup{})
	return context.WithCancel(ctxWg)
}

// stop entire pipeline and make sure that all waiting goroutines are unblocked and exited
func Shutdown(ctx context.Context, cancel context.CancelFunc) {
	wg := getWaitGroup(ctx)
	if !wg.IsValid() {
		panic("context must be create with `NewPipeline")
	}
	cancel()
	wg.Wait()
}

func getWaitGroup(ctx context.Context) (opt optWaitGroup) {
	r := ctx.Value(waitGroupKey)
	if r != nil {
		opt.wg = r.(*sync.WaitGroup)
	}
	return
}

// optional wait group, do nothing if context was create without `NewPipeline`
type optWaitGroup struct {
	wg *sync.WaitGroup
}

func (opt optWaitGroup) Add(delta int) {
	if opt.wg != nil {
		opt.wg.Add(delta)
	}
}

func (opt optWaitGroup) Done() {
	if opt.wg != nil {
		opt.wg.Done()
	}
}

func (opt optWaitGroup) Wait() {
	if opt.wg != nil {
		opt.wg.Wait()
	}
}

func (opt optWaitGroup) IsValid() bool {
	return opt.wg != nil
}
