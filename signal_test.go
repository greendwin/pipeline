package pipeline_test

import (
	"testing"
	"time"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func checkPending[T any](t *testing.T, ch <-chan T) {
	t.Helper()

	select {
	case val, ok := <-ch:
		if !ok {
			t.Fatalf("channel was closed")
		}

		t.Fatalf("channel is not in a pending state (received %v)", val)
	default:
		// success
	}
}

func checkSignaled[T any](t *testing.T, ch <-chan T) {
	t.Helper()

	select {
	case <-ch:
		// success
	case <-time.After(time.Second):
		t.Fatalf("channel is not in a signaled state")
	}
}

func TestSignal(t *testing.T) {
	mut := pl.NewSignal()
	sig := mut.Chan()

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
	}(mut.Chan())

	checkPending(t, finished)
	mut.Set()
	checkSignaled(t, finished)
}

func TestSignalTryWait(t *testing.T) {
	withTimeout(t, "wait signaled", func() {
		sig := pl.NewSignal()
		sig.Set()

		r := sig.TryWait(10 * time.Millisecond)
		assert.True(t, r)
	})

	withTimeout(t, "wait for signal timeout", func() {
		sig := pl.NewSignal()
		r := sig.TryWait(10 * time.Millisecond)
		assert.False(t, r)
	})
}
