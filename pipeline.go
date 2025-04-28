package pipeline

import (
	"sync"
	"time"
)

// TODO: move to separate file

// TODO: !!! should we close `cherr` channels?
// TODO: another way: should we close `Oneshot` channels? (non errors?)

// 1-buffered channel with single result
type Oneshot[T any] <-chan T

func (ch *Oneshot[T]) Wait() (val T, ok bool) {
	panic("not imeplemented")
}

func (ch *Oneshot[T]) TryWait(d time.Duration) (val T, ok bool) {
	panic("not imeplemented")
}

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
	cherr := make(chan error, 1)
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()

		err := cb()
		if err != nil {
			cherr <- err
		}
	}()
	return cherr
}
