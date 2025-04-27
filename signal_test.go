package pipeline_test

import (
	"runtime/debug"
	"testing"
	"time"

	pl "github.com/greendwin/pipeline"
)

func checkPending[T any](t *testing.T, ch <-chan T) {
	select {
	case <-ch:
		t.Fatalf("signal is not in a pending state:\n\n%s", debug.Stack())
	default:
		// success
	}
}

func checkSignaled[T any](t *testing.T, ch <-chan T) {
	select {
	case <-ch:
		// success
	case <-time.After(time.Second):
		t.Fatalf("channel is not in a signaled state:\n\n%s", debug.Stack())
	}
}

func TestSignal(t *testing.T) {
	mut := pl.NewSignal()
	sig := mut.Signal()

	checkPending(t, mut)
	checkPending(t, sig)

	mut.Set()
	checkSignaled(t, mut)
	checkSignaled(t, sig)
}
