package pipeline

import (
	"sync"
)

// pipeline context
// track spawned goroutines and gracefully shutdown full network
type Pipeline struct {
	wg   sync.WaitGroup
	done SignalMut
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		done: NewSignal(),
	}
}

// stop entire pipeline
// make sure that all waiting goroutines are unblocked and exited
func (pp *Pipeline) Shutdown() {
	pp.done.Set()
	pp.wg.Wait()
}

// spawn goroutine, tracking spawn count
// make sure that it will exit on shutdown
func (pp *Pipeline) Go(cb func()) {
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		cb()
	}()
}

// spawn goroutine that can fail
func (pp *Pipeline) GoErr(cb func() error) Oneshot[error] {
	cherr := NewOneshot[error]()
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()

		err := cb()
		if err != nil {
			cherr.Write(err)
		}
	}()
	return cherr.Chan()
}
