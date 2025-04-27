package pipeline

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
