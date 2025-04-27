package pipeline_test

import (
	"errors"
	"runtime/debug"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func withTimeout(t *testing.T, cb func()) {
	finished := pl.NewSignal()
	go func() {
		cb()
		finished.Set()
	}()
	checkSignaled(t, finished)
}

func checkShutdown(t *testing.T, pp *pl.Pipeline) {
	withTimeout(t, func() {
		pp.Shutdown()
	})
}

func checkRead[T any](t *testing.T, ch <-chan T) T {
	res := make(chan T, 1)
	withTimeout(t, func() {
		v, ok := <-ch
		if !ok {
			t.Fatalf("channel was unexpectedly closed:\n\n%s", debug.Stack())
		}

		res <- v
	})
	return <-res
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
