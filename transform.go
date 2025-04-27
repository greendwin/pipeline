package pipeline

import "sync"

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

	closeAfterAll(pp, &wg, out)

	return out
}

func TransformErr[T any, U any](pp *Pipeline, threads int, in <-chan T, cb func(T) (U, error)) (<-chan U, Oneshot[error]) {
	out := make(chan U)
	cherr := make(chan error, threads) // each worker can send one error

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

				r, err := cb(v)
				if err != nil {
					cherr <- err
					return
				}

				if !Write(pp, out, r) {
					return
				}
			}
		})
	}

	closeAfterAll(pp, &wg, out)

	return out, cherr
}

func closeAfterAll[T any](pp *Pipeline, wg *sync.WaitGroup, ch chan T) {
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		defer close(ch)
		wg.Wait()
	}()
}
