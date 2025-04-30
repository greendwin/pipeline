package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
)

func Transform[T any, U any](ctx context.Context, threads int, in <-chan T, cb func(T) (U, error)) (<-chan U, Oneshot[error]) {
	out := make(chan U)
	cherr := NewOneshotGroup[error](threads) // each worker can send one error

	var wg sync.WaitGroup
	wg.Add(threads)

	hasError := atomic.Bool{}

	for range threads {
		Go(ctx, func() {
			defer wg.Done()

			var err error
			for {
				var v T
				v, err = Read(ctx, in)
				if err != nil {
					break
				}

				var r U
				r, err = cb(v)
				if err != nil {
					break
				}

				if err = Write(ctx, out, r); err != nil {
					break
				}
			}

			if err != nil {
				cherr.Write(err)
				hasError.Store(true)
			}
		})
	}

	closeAfterAll(ctx, &wg, &hasError, out)

	return out, cherr.Chan()
}

func closeAfterAll[T any](ctx context.Context, wg *sync.WaitGroup, hasError *atomic.Bool, ch chan T) {
	pipelineWg := getWaitGroup(ctx)
	pipelineWg.Add(1)
	go func() {
		defer pipelineWg.Done()
		wg.Wait()

		if hasError == nil || !hasError.Load() {
			close(ch) // don't close channel on error
		}
	}()
}
