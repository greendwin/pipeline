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

// read either channel value or an error from any triggered channel
func ReadErr[T any](pp *Pipeline, in <-chan T, errs ...<-chan error) (T, error) {
	select {
	case val, ok := <-in:
		if !ok {
			return val, ErrChannelClosed
		}
		return val, nil

	case err, ok := <-First(pp, errs...):
		var empty T
		if ok {
			return empty, err
		}
		return empty, ErrChannelClosed
	}
}

// TODO: rewrite `ReadErr` without `First`
// TODO: add tests!
