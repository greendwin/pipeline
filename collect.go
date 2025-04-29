package pipeline

import "context"

func Collect[T any](ctx context.Context, cb func() T) Oneshot[T] {
	out := NewOneshot[T]()

	wg := getWaitGroup(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		out.Write(cb())
	}()
	return out.Chan()
}

func CollectErr[T any](ctx context.Context, cb func() (T, error)) (Oneshot[T], Oneshot[error]) {
	out := NewOneshot[T]()
	cherr := NewOneshot[error]()

	wg := getWaitGroup(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		v, err := cb()
		if err != nil {
			cherr.Write(err)
			return
		}

		out.Write(v)
	}()

	return out.Chan(), cherr.Chan()
}
