package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
)

func Process[T any](ctx context.Context, threads int, in <-chan T, cb func(T) error) (Signal, Oneshot[error]) {
	cherr := NewOneshotGroup[error](threads) // each worker can send one error

	var wg sync.WaitGroup
	wg.Add(threads)

	hasError := atomic.Bool{}

	for range threads {
		Go(ctx, func() {
			defer wg.Done()

			for {
				v, err := Read(ctx, in)
				if err == nil {
					err = cb(v)
				}

				if err != nil {
					cherr.Write(err)
					hasError.Store(true)
					return
				}
			}
		})
	}

	finished := signalAfterAll(ctx, &wg, &hasError)

	return finished, cherr.Chan()
}

func signalAfterAll(ctx context.Context, wg *sync.WaitGroup, hasError *atomic.Bool) Signal {
	pipelineWg := getWaitGroup(ctx)
	finished := NewSignal()
	pipelineWg.Add(1)
	go func() {
		defer pipelineWg.Done()
		wg.Wait()

		if hasError == nil || !hasError.Load() {
			// note: `finished` never triggered in case of error
			finished.Set()
		}
	}()
	return finished.Chan()
}
