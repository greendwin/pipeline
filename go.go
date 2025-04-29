package pipeline

import "context"

// spawn goroutine, tracking spawn count
// make sure that it will exit on shutdown
func Go(ctx context.Context, cb func()) {
	wg := getWaitGroup(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		cb()
	}()
}

// spawn goroutine that can fail
func GoErr(ctx context.Context, cb func() error) Oneshot[error] {
	wg := getWaitGroup(ctx)
	cherr := NewOneshot[error]()
	wg.Add(1)
	go func() {
		defer wg.Done()

		err := cb()
		if err != nil {
			cherr.Write(err)
		}
	}()
	return cherr.Chan()
}
