package pipeline

import "sync"

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

	return signalAfterAll(pp, &wg)
}

func ProcessErr[T any](pp *Pipeline, threads int, in <-chan T, cb func(T) error) (Signal, Oneshot[error]) {
	cherr := NewOneshotGroup[error](threads) // each worker can send one error

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

				err := cb(v)
				if err != nil {
					cherr.Write(err)
					return
				}
			}
		})
	}

	finished := signalAfterAll(pp, &wg)

	return finished, cherr.Chan()
}

func signalAfterAll(pp *Pipeline, wg *sync.WaitGroup) Signal {
	finished := NewSignal()
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		defer finished.Set()
		wg.Wait()
	}()
	return finished.Signal()
}
