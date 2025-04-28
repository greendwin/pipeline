package pipeline

// TODO: test me!!!

func Run(pp *Pipeline, cb func()) Signal {
	finished := NewSignal()
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		defer finished.Set()
		cb()
	}()
	return finished.Signal()
}

func RunErr(pp *Pipeline, cb func() error) (Signal, Oneshot[error]) {
	panic("not implemented")
}
