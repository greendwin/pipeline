package pipeline_test

import (
	"errors"
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

func checkShutdown(t *testing.T, pp *pl.Pipeline) {
	t.Helper()

	withTimeout(t, "pipline shutdown", func() {
		pp.Shutdown()
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

func TestPipelineGo(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	started := pl.NewSignal()
	pp.Go(func() {
		started.Set()
	})

	checkSignaled(t, started)
}

func TestPipelineShutdown(t *testing.T) {
	pp := pl.NewPipeline()

	exit := pl.NewSignal()
	pp.Go(func() {
		<-exit
	})

	shutdownFinished := pl.NewSignal()
	go func() {
		pp.Shutdown()
		shutdownFinished.Set()
	}()

	// wait for all spawned goroutines to exit
	checkPending(t, shutdownFinished)
	exit.Set()
	checkSignaled(t, shutdownFinished)
}

var testError = errors.New("test")

func TestPipelineGoErr(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	started := pl.NewSignal()
	exit := pl.NewSignal()

	cherr := pp.GoErr(func() error {
		started.Set()
		<-exit
		return testError
	})

	checkSignaled(t, started)
	checkPending(t, cherr)

	exit.Set()

	err := checkRead(t, cherr)
	assert.Equal(t, err, testError)
}

func TestPipelineShutdownGoErr(t *testing.T) {
	pp := pl.NewPipeline()

	exit := pl.NewSignal()
	pp.GoErr(func() error {
		<-exit
		return nil
	})

	shutdownFinished := pl.NewSignal()
	go func() {
		pp.Shutdown()
		shutdownFinished.Set()
	}()

	// wait for all spawned goroutines to exit
	checkPending(t, shutdownFinished)
	exit.Set()
	checkSignaled(t, shutdownFinished)
}
