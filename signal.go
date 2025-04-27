package pipeline

// marker type that indicates that channel will never send
type None struct {
	none struct{}
}

// two-state channel `opened` and `closed`
type Signal <-chan None

func NewSignal() signalMut {
	return make(chan None)
}

type signalMut chan None

func (sig signalMut) Set() {
	close(sig)
}

func (sig signalMut) Signal() Signal {
	return (chan None)(sig)
}
