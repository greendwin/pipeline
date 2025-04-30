package pipeline

import (
	"context"
	"sync"
)

func NewPipeline(parent context.Context) (context.Context, context.CancelFunc) {
	wg := &sync.WaitGroup{}
	ctxWg := context.WithValue(parent, waitGroupKey, wg)
	ctx, cancel := context.WithCancel(ctxWg)

	// wait goroutines shutdown on cancel
	return ctx, func() {
		shutdown(wg, cancel)
	}
}

// stop entire pipeline and make sure that all waiting goroutines are unblocked and exited
func shutdown(wg *sync.WaitGroup, cancel context.CancelFunc) {
	cancel()
	wg.Wait()
}

type contextKey int

const waitGroupKey contextKey = 0

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
