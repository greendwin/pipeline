package pipeline

func Write[T any](pp *Pipeline, out chan<- T, val T) bool {
	select {
	case out <- val:
		return true
	case <-pp.done:
		return false
	}
}

func Read[T any](pp *Pipeline, in <-chan T) (v T, ok bool) {
	select {
	case v, ok = <-in:
		return v, ok
	case <-pp.done:
		var empty T
		return empty, false
	}
}
