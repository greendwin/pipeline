package pipeline

// oneshot channel, 1-item buffered in most cases
// this channel is never closed, use `Oneshot::Read` or `WaitFirst`
// to read a value in non-stuck manner
type Oneshot[T any] <-chan T

// writer side version that supports writing
type OneshotMut[T any] struct {
	ch chan T
}

func NewOneshot[T any]() OneshotMut[T] {
	return OneshotMut[T]{make(chan T, 1)}
}

// create oneshot-like channel that acts as `Oneshot` on readers side,
// but can be written `groupSize` times
func NewOneshotGroup[T any](groupSize int) OneshotMut[T] {
	return OneshotMut[T]{make(chan T, groupSize)}
}

func (m OneshotMut[T]) Chan() Oneshot[T] {
	return m.ch
}

func (m OneshotMut[T]) Write(val T) {
	select {
	case m.ch <- val:
		// success
	default:
		panic("write must not block: make sure you don't try to write to the same channel multiple times")
	}
}
