package pipeline_test

import (
	"context"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	finished := pl.NewSignal()
	squares, cherr := pl.Generate(ctx, func(wr pl.Writer[int]) error {
		for k := range 10 {
			err := wr.Write(k * k)
			assert.Nil(t, err)
		}
		finished.Set()
		return nil
	})

	withTimeout(t, "read seq", func() {
		index := 0
		for val := range squares {
			assert.Equal(t, val, index*index)
			index += 1
		}
	})

	checkPending(t, cherr)
	checkSignaled(t, finished)
}

func TestGenerate_PropagateError(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	partial, cherr := pl.Generate(ctx, func(wr pl.Writer[int]) error {
		for k := range 5 {
			err := wr.Write(k * k)
			assert.Nil(t, err)
		}
		return errTest
	})

	finished := pl.NewSignal()
	go func() {
		index := 0
		for val := range partial {
			assert.Equal(t, val, index*index)
			index += 1
		}
		finished.Set()
	}()

	err := checkRead(t, cherr)
	assert.Equal(t, err, errTest)

	// note: `partial` channel must be closed for this
	checkSignaled(t, finished)
}

func TestGenerate_DontStuck(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	finished := pl.NewSignal()
	_, cherr := pl.Generate(ctx, func(wr pl.Writer[int]) error {
		err := wr.Write(42)
		assert.ErrorIs(t, err, context.Canceled)
		finished.Set()
		return nil
	})

	// shutdown execution to unblock write
	checkShutdown(t, cancel)

	checkPending(t, cherr)
	checkSignaled(t, finished)
}

// TODO: add test for `Cause` propagation
