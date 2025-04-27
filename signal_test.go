package pipeline_test

import (
	"runtime/debug"
	"testing"
	"time"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
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

func TestSignalWait(t *testing.T) {
	mut := pl.NewSignal()

	finished := pl.NewSignal()
	go func(sig pl.Signal) {
		sig.Wait()
		finished.Set()
	}(mut.Signal())

	checkPending(t, finished)
	mut.Set()
	checkSignaled(t, finished)
}

func TestSignalWaitFor(t *testing.T) {
	withTimeout(t, func() {
		sig := pl.NewSignal()
		sig.Set()

		r := sig.WaitFor(10 * time.Millisecond)
		assert.True(t, r)
	})

	withTimeout(t, func() {
		sig := pl.NewSignal()
		r := sig.WaitFor(10 * time.Millisecond)
		assert.False(t, r)
	})
}
