package pipeline

func Collect[T any](pp *Pipeline, cb func() T) Oneshot[T] {
	out := NewOneshot[T]()
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		out.Write(cb())
	}()
	return out.Chan()
}

func CollectErr[T any](pp *Pipeline, cb func() (T, error)) (Oneshot[T], Oneshot[error]) {
	out := NewOneshot[T]()
	cherr := NewOneshot[error]()

	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		v, err := cb()
		if err != nil {
			cherr.Write(err)
			return
		}

		out.Write(v)
	}()

	return out.Chan(), cherr.Chan()
}
