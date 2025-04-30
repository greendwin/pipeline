package pipeline

import (
	"context"
	"reflect"
)

func Write[T any](ctx context.Context, out chan<- T, val T) bool {
	select {
	case out <- val:
		return true
	case <-ctx.Done():
		return false
	}
}

func Read[T any](ctx context.Context, in <-chan T) (val T, ok bool) {
	select {
	case val, ok = <-in:
		return
	case <-ctx.Done():
		ok = false // for clarity
		return
	}
}

// read either a channel value or an error from the first triggered channel
//
// returns `ErrChannelClosed` if value channel was closed
// returns `ErrCancelled` if pipeline in shutting down
// fallback to `Read` if all `errs` are closed (return `ErrChannelClosed` if !ok)
func ReadErr[T any](ctx context.Context, in <-chan T, errs ...<-chan error) (T, error) {
	var cases []reflect.SelectCase
	if len(errs) <= 2 {
		cases = make([]reflect.SelectCase, len(errs)+2, 4) // stack allocation on simple cases
	} else {
		cases = make([]reflect.SelectCase, len(errs)+2)
	}

	cases[0] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(in),
	}

	for k, cherr := range errs {
		cases[k+1] = reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(cherr),
		}
	}

	cases[len(cases)-1] = reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	}

	var empty T

	for {
		index, val, ok := reflect.Select(cases)

		if index+1 == len(cases) {
			// `pp.done` was triggered
			return empty, ErrChannelClosed
		}

		if ok {
			if index == 0 {
				return val.Interface().(T), nil
			}

			return empty, val.Interface().(error)
		}

		// results channel was closed
		if index == 0 {
			return empty, ErrChannelClosed
		}

		cases = append(cases[:index], cases[index+1:]...)
	}
}
