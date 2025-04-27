package pipeline

import "time"

// marker type that indicates that channel will never send
type None struct {
	none struct{}
}

// two-state channel: `pending` and `closed`
type Signal <-chan None

func (sig Signal) Wait() {
	<-sig
}

func (sig Signal) WaitFor(d time.Duration) bool {
	select {
	case <-sig:
		return true
	case <-time.After(d):
		return false
	}
}

func NewSignal() SignalMut {
	return make(chan None)
}

// mutable signal that supports `Set` in addition to all `Signal` methods
type SignalMut chan None

func (sig SignalMut) Set() {
	close(sig)
}

func (sig SignalMut) Signal() Signal {
	return chan None(sig)
}

// redeclare that same methods for `mut` version

func (sig SignalMut) Wait() {
	sig.Signal().Wait()
}

func (sig SignalMut) WaitFor(d time.Duration) bool {
	return sig.Signal().WaitFor(d)
}
