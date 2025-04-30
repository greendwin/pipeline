package pipeline_test

import (
	"context"
	"errors"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

var errTest = errors.New("test")

func TestPipelineGo(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	started := pl.NewSignal()
	pl.Go(ctx, func() {
		started.Set()
	})

	checkSignaled(t, started)
}

func TestPipelineGoErr(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	started := pl.NewSignal()
	exit := pl.NewSignal()

	cherr := pl.GoErr(ctx, func() error {
		started.Set()
		<-exit
		return errTest
	})

	checkSignaled(t, started)
	checkPending(t, cherr)

	exit.Set()

	err := checkRead(t, cherr)
	assert.Equal(t, err, errTest)
}

func TestPipelineShutdownGoErr(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	exit := pl.NewSignal()
	pl.GoErr(ctx, func() error {
		<-exit
		return nil
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
