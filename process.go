package pipeline

import (
	"context"
	"sync"
	"sync/atomic"
)

func Process[T any](ctx context.Context, threads int, in <-chan T, cb func(T)) Signal {
	var wg sync.WaitGroup
	wg.Add(threads)

	for range threads {
		Go(ctx, func() {
			defer wg.Done()

			for {
				v, ok := Read(ctx, in)
				if !ok {
					return
				}

				cb(v)
			}
		})
	}

	return signalAfterAll(ctx, &wg, nil)
}

func ProcessErr[T any](ctx context.Context, threads int, in <-chan T, cb func(T) error) (Signal, Oneshot[error]) {
	cherr := NewOneshotGroup[error](threads) // each worker can send one error

	var wg sync.WaitGroup
	wg.Add(threads)

	hasError := atomic.Bool{}

	for range threads {
		Go(ctx, func() {
			defer wg.Done()

			for {
				v, ok := Read(ctx, in)
				if !ok {
					return
				}

				err := cb(v)
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
	finished := NewSignal()
	wg.Add(1)
	go func() {
		defer wg.Done()
		wg.Wait()

		if hasError == nil || !hasError.Load() {
			// note: `finished` never triggered in case of error
			finished.Set()
		}
	}()
	return finished.Chan()
}
