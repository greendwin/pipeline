package pipeline_test

import (
	"context"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, ctx, cancel)

	squares := pl.Generate(ctx, func(wr pl.Writer[int]) {
		for k := range 10 {
			if !wr.Write(k * k) {
				return
			}
		}
	})

	withTimeout(t, "read seq", func() {
		index := 0
		for val := range squares {
			assert.Equal(t, val, index*index)
			index += 1
		}
	})
}

func TestGenerateDontStuck(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	finished := pl.NewSignal()
	// never receive
	_ = pl.Generate(ctx, func(wr pl.Writer[int]) {
		r := wr.Write(42)
		assert.False(t, r)
		finished.Set()
	})

	checkShutdown(t, ctx, cancel)
	checkSignaled(t, finished)
}

func TestGenerateErr(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, ctx, cancel)

	finished := pl.NewSignal()
	squares, cherr := pl.GenerateErr(ctx, func(wr pl.Writer[int]) error {
		for k := range 10 {
			r := wr.Write(k * k)
			assert.True(t, r)
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

func TestGenerateErrPropagate(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, ctx, cancel)

	partial, cherr := pl.GenerateErr(ctx, func(wr pl.Writer[int]) error {
		for k := range 5 {
			r := wr.Write(k * k)
			assert.True(t, r)
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

func TestGenerateErrDontStuck(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	finished := pl.NewSignal()
	_, cherr := pl.GenerateErr(ctx, func(wr pl.Writer[int]) error {
		r := wr.Write(42)
		assert.False(t, r)
		finished.Set()
		return nil
	})

	// shutdown execution to unblock write
	checkShutdown(t, ctx, cancel)

	checkPending(t, cherr)
	checkSignaled(t, finished)
}
