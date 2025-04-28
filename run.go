package pipeline

func Run(pp *Pipeline, cb func()) Signal {
	finished := NewSignal()
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		defer finished.Set()
		cb()
	}()
	return finished.Chan()
}

func RunErr(pp *Pipeline, cb func() error) (Signal, Oneshot[error]) {
	finished := NewSignal()
	cherr := NewOneshot[error]()

	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
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
