package pipeline

import (
	"context"
	"errors"
	"reflect"
)

var ErrChannelClosed = errors.New("channel closed")

// return first value ignoring closed channels
// if all `in` channels are closed, return `ErrChannelClosed`
// return `context.Cause(ctx)` in case of cancellation
func WaitFirst[T any](ctx context.Context, in ...<-chan T) (T, error) {
	var cases []reflect.SelectCase
	if len(in) <= 3 {
		cases = make([]reflect.SelectCase, len(in)+1, 4) // stack allocate
	} else {
		cases = make([]reflect.SelectCase, len(in)+1)
	}

	for k, ch := range in {
		cases[k].Dir = reflect.SelectRecv
		cases[k].Chan = reflect.ValueOf(ch)
	}

	cases[len(in)].Dir = reflect.SelectRecv
	cases[len(in)].Chan = reflect.ValueOf(ctx.Done())

	for {
		idx, v, ok := reflect.Select(cases)

		if idx+1 == len(cases) {
			// `ctx.Done()` was triggered
			var empty T
			return empty, context.Cause(ctx)
		}

		if ok {
			return v.Interface().(T), nil
		}

		if len(cases) == 2 {
			// removing the last case, only `ctx.done` remains
			var empty T
			return empty, ErrChannelClosed
		}

		// remove closed channel
		cases = append(cases[:idx], cases[idx+1:]...)
	}
}

func First[T any](ctx context.Context, in ...<-chan T) (Oneshot[T], Oneshot[error]) {
	out := NewOneshot[T]()
	cherr := NewOneshot[error]()

	wg := getWaitGroup(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()

		v, err := WaitFirst(ctx, in...)
		if err != nil {
			cherr.Write(err)
		} else {
			out.Write(v)
		}
	}()

	return out.Chan(), cherr.Chan()
}

func FirstErr(ctx context.Context, errs ...<-chan error) Oneshot[error] {
	cherr := NewOneshot[error]()

	wg := getWaitGroup(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()

		err, waitErr := WaitFirst(ctx, errs...)
		if waitErr != nil {
			cherr.Write(waitErr)
		} else {
			cherr.Write(err)
		}
	}()

	return cherr.Chan()
}

func FanIn[T any](ctx context.Context, in ...<-chan T) (<-chan T, Oneshot[error]) {
	out := make(chan T)
	cherr := NewOneshot[error]()

	wg := getWaitGroup(ctx)
	wg.Add(1)
	go func() {
		defer wg.Done()

		var cases []reflect.SelectCase
		if len(in) <= 3 {
			cases = make([]reflect.SelectCase, len(in)+1, 4)
		} else {
			cases = make([]reflect.SelectCase, len(in)+1)
		}

		for k, ch := range in {
			cases[k].Dir = reflect.SelectRecv
			cases[k].Chan = reflect.ValueOf(ch)
		}
		cases[len(in)].Dir = reflect.SelectRecv
		cases[len(in)].Chan = reflect.ValueOf(ctx.Done())

		for {
			idx, v, ok := reflect.Select(cases)
			if idx+1 == len(cases) {
				// `ctx.Done()` was triggered
				cherr.Write(context.Cause(ctx))
				return
			}

			if ok {
				if !Write(ctx, out, v.Interface().(T)) {
					// `ctx.Done()` was triggered
					cherr.Write(context.Cause(ctx))
					return
				}
				continue
			}

			if len(cases) == 2 {
				// last input channel was closed, so close output channel
				// note: only `ctx.Done()` is left
				close(out)
				return
			}

			// drop closed channel from select
			cases = append(cases[:idx], cases[idx+1:]...)
		}
	}()

	return out, cherr.Chan()
}
