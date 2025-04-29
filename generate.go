package pipeline

import "context"

type Writer[T any] interface {
	Write(T) bool
}

type channel[T any] struct {
	ctx context.Context
	ch  chan T
}

func (ch *channel[T]) Write(val T) bool {
	return Write(ch.ctx, ch.ch, val)
}

func Generate[T any](ctx context.Context, cb func(Writer[T])) <-chan T {
	out := channel[T]{ctx, make(chan T)}
	wg := getWaitGroup(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(out.ch)
		cb(&out)
	}()

	return out.ch
}

func GenerateErr[T any](ctx context.Context, cb func(Writer[T]) error) (<-chan T, Oneshot[error]) {
	out := channel[T]{ctx, make(chan T)}
	cherr := GoErr(ctx, func() error {
		defer close(out.ch)
		return cb(&out)
	})

	return out.ch, cherr
}
