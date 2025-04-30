package pipeline_test

import (
	"context"
	"testing"
	"time"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
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

func checkShutdown(t *testing.T, cancel context.CancelFunc) {
	t.Helper()

	withTimeout(t, "pipline shutdown", func() {
		cancel()
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
		cancel()
		shutdownFinished.Set()
	}()

	// wait for all spawned goroutines to exit
	checkPending(t, shutdownFinished)
	exit.Set()
	checkSignaled(t, shutdownFinished)
}

func TestPipelineIsOptional(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	seq := pl.Generate(ctx, func(w pl.Writer[int]) {
		for k := range 100 {
			_ = w.Write(k)
		}
	})

	continueCollect := pl.NewSignal()
	foundStrangeNumber := pl.NewSignal()
	res := pl.Collect(ctx, func() int {
		sum := 0
		for {
			v, ok := pl.Read(ctx, seq)
			if !ok {
				break
			}

			if v == 42 {
				foundStrangeNumber.Set()
				continueCollect.Wait()
				return sum
			}

			sum += v
		}
		return sum
	})

	withTimeout(t, "waiting collection", func() {
		foundStrangeNumber.Wait()
	})

	checkPending(t, res)
	cancel()
	continueCollect.Set()

	v := checkRead(t, res)
	assert.Equal(t, 861, v) // sum from 0 to 42
}
