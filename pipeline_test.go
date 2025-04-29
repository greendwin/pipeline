package pipeline_test

import (
	"context"
	"testing"
	"time"

	pl "github.com/greendwin/pipeline"
)

func withTimeout(t *testing.T, context string, cb func()) {
	t.Helper()

	finished := pl.NewSignal()
	go func() {
		cb()
		finished.Set()
	}()

	if !finished.TryWait(time.Second) {
		t.Fatalf("%s: timeout", context)
	}
}

func checkShutdown(t *testing.T, ctx context.Context, cancel context.CancelFunc) {
	t.Helper()

	withTimeout(t, "pipline shutdown", func() {
		pl.Shutdown(ctx, cancel)
	})
}

func checkRead[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	type result struct {
		val T
		ok  bool
	}

	chres := make(chan result, 1)
	withTimeout(t, "read channel", func() {
		v, ok := <-ch
		chres <- result{v, ok}
	})

	r := <-chres
	if !r.ok {
		t.Fatalf("channel was unexpectedly closed")
	}
	return r.val
}

func TestPipelineShutdown(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	exit := pl.NewSignal()
	pl.Go(ctx, func() {
		<-exit
	})

	shutdownFinished := pl.NewSignal()
	go func() {
		pl.Shutdown(ctx, cancel)
		shutdownFinished.Set()
	}()

	// wait for all spawned goroutines to exit
	checkPending(t, shutdownFinished)
	exit.Set()
	checkSignaled(t, shutdownFinished)
}
