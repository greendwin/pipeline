package pipeline

import (
	"sync"
	"sync/atomic"
)

func Transform[T any, U any](pp *Pipeline, threads int, in <-chan T, cb func(T) U) <-chan U {
	out := make(chan U)

	var wg sync.WaitGroup
	wg.Add(threads)

	for range threads {
		pp.Go(func() {
			defer wg.Done()

			for {
				v, ok := Read(pp, in)
				if !ok {
					return
				}

				if !Write(pp, out, cb(v)) {
					return
				}
			}
		})
	}

	closeAfterAll(pp, &wg, nil, out)

	return out
}

func TransformErr[T any, U any](pp *Pipeline, threads int, in <-chan T, cb func(T) (U, error)) (<-chan U, Oneshot[error]) {
	out := make(chan U)
	cherr := NewOneshotGroup[error](threads) // each worker can send one error

	var wg sync.WaitGroup
	wg.Add(threads)

	hasError := atomic.Bool{}

	for range threads {
		pp.Go(func() {
			defer wg.Done()

			for {
				v, ok := Read(pp, in)
				if !ok {
					return
				}

				r, err := cb(v)
				if err != nil {
					cherr.Write(err)
					hasError.Store(true)
					return
				}

				if !Write(pp, out, r) {
					return
				}
			}
		})
	}

	closeAfterAll(pp, &wg, &hasError, out)

	return out, cherr.Chan()
}

func closeAfterAll[T any](pp *Pipeline, wg *sync.WaitGroup, hasError *atomic.Bool, ch chan T) {
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		wg.Wait()

		if hasError == nil || !hasError.Load() {
			close(ch) // don't close channel on error
		}
	}()
}
