package pipeline

import (
	"reflect"
)

func WaitFirst[T any](pp *Pipeline, in ...<-chan T) (T, bool) {
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
	cases[len(in)].Chan = reflect.ValueOf(pp.done)

	for {
		idx, v, ok := reflect.Select(cases)

		if idx+1 == len(cases) {
			// `pp.done` was triggered
			var empty T
			return empty, false
		}

		if ok {
			return v.Interface().(T), true
		}

		if len(cases) == 2 {
			// removing the last case, only `pp.done` remains
			var empty T
			return empty, false
		}

		// remove closed channel
		cases = append(cases[:idx], cases[idx+1:]...)
	}
}

func First[T any](pp *Pipeline, in ...<-chan T) Oneshot[T] {
	out := NewOneshot[T]()
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()

		v, ok := WaitFirst(pp, in...)
		if ok {
			out.Write(v)
		}
	}()

	return out.Chan()
}

func FanIn[T any](pp *Pipeline, in ...<-chan T) <-chan T {
	out := make(chan T)
	pp.wg.Add(1)
	go func() {
		defer pp.wg.Done()
		defer close(out)

		cases := make([]reflect.SelectCase, len(in)+1)
		for k, ch := range in {
			cases[k].Dir = reflect.SelectRecv
			cases[k].Chan = reflect.ValueOf(ch)
		}
		cases[len(in)].Dir = reflect.SelectRecv
		cases[len(in)].Chan = reflect.ValueOf(pp.done)

		for {
			idx, v, ok := reflect.Select(cases)
			if idx+1 == len(cases) {
				// `pp.done` was triggered
				return
			}

			if ok {
				if !Write(pp, out, v.Interface().(T)) {
					return
				}
				continue
			}

			if len(cases) == 2 {
				// last input channel was closed, only `pp.done` left
				return
			}

			// drop closed channel
			cases = append(cases[:idx], cases[idx+1:]...)
		}
	}()

	return out
}
