package pipeline

import "errors"

func Write[T any](pp *Pipeline, out chan<- T, val T) bool {
	select {
	case out <- val:
		return true
	case <-pp.done:
		return false
	}
}

func Read[T any](pp *Pipeline, in <-chan T) (val T, ok bool) {
	select {
	case val, ok = <-in:
		return
	case <-pp.done:
		ok = false // for clarity
		return
	}
}

var ErrChannelClosed = errors.New("channel closed")

// read either a channel value or an error from the first triggered channel
//
// returns `ErrChannelClosed` if value channel was closed
// returns `ErrCancelled` if pipeline in shutting down
// fallback to `Read` if all `errs` are closed (return `ErrChannelClosed` if !ok)
func ReadErr[T any](pp *Pipeline, in <-chan T, errs ...<-chan error) (val T, err error) {
	var ok bool
	select {
	case <-pp.done.Chan():
		err = ErrChannelClosed
		return

	case val, ok = <-in:
		if !ok {
			err = ErrChannelClosed
		}
		return

	case err, ok = <-First(pp, errs...):
		if ok {
			return
		}

		// all error channels were closed, fall back to `Read`
		val, ok = Read(pp, in)
		if !ok {
			err = ErrChannelClosed
		}
		return
	}
}

// TODO: rewrite `ReadErr` without `First`
