package pipeline

import (
	"sync"
	"sync/atomic"
)

func Process[T any](pp *Pipeline, threads int, in <-chan T, cb func(T)) Signal {
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

				cb(v)
			}
		})
	}

	return signalAfterAll(pp, &wg, nil)
}

func ProcessErr[T any](pp *Pipeline, threads int, in <-chan T, cb func(T) error) (Signal, Oneshot[error]) {
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

				err := cb(v)
				if err != nil {
					cherr.Write(err)
					hasError.Store(true)
					return
				}
			}
		})
	}

	finished := signalAfterAll(pp, &wg, &hasError)

	return finished, cherr.Chan()
}

func signalAfterAll(pp *Pipeline, wg *sync.WaitGroup, hasError *atomic.Bool) Signal {
	finished := NewSignal()
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		wg.Wait()

		if hasError == nil || !hasError.Load() {
			// note: `finished` never triggered in case of error
			finished.Set()
		}
	}()
	return finished.Chan()
}
