package pipeline

type Writer[T any] interface {
	Write(T) bool
}

type channel[T any] struct {
	pp *Pipeline
	ch chan T
}

func (ch *channel[T]) Write(val T) bool {
	return Write(ch.pp, ch.ch, val)
}

func Generate[T any](pp *Pipeline, cb func(Writer[T])) <-chan T {
	out := channel[T]{pp, make(chan T)}
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		defer close(out.ch)
		cb(&out)
	}()

	return out.ch
}
