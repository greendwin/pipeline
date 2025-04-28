package pipeline

// TODO: test me!!!

func Collect[T any](pp *Pipeline, cb func() T) Oneshot[T] {
	out := make(chan T, 1)
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		defer close(out)
		out <- cb()
	}()
	return out
}

func CollectErr[T any](pp *Pipeline, cb func() (T, error)) (Oneshot[T], Oneshot[error]) {
	out := make(chan T, 1)
	cherr := make(chan error, 1)

	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		defer close(out)
		v, err := cb()
		if err != nil {
			cherr <- err
		} else {
			out <- v
		}
	}()

	return out, cherr
}
