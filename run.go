package pipeline

import "context"

func Run(ctx context.Context, cb func()) Signal {
	finished := NewSignal()
	wg := getWaitGroup(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer finished.Set()
		cb()
	}()
	return finished.Chan()
}

func RunErr(ctx context.Context, cb func() error) (Signal, Oneshot[error]) {
	finished := NewSignal()
	cherr := NewOneshot[error]()

	wg := getWaitGroup(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := cb()
		if err != nil {
			// note: `finished` is not triggered in this case
			cherr.Write(err)
			return
		}

		finished.Set()
	}()

	return finished.Chan(), cherr.Chan()
}
