package pipeline_test

import (
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	squares := pl.Generate(pp, func(wr pl.Writer[int]) {
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
	pp := pl.NewPipeline()

	finished := pl.NewSignal()
	// never receive
	_ = pl.Generate(pp, func(wr pl.Writer[int]) {
		r := wr.Write(42)
		assert.False(t, r)
		finished.Set()
	})

	checkShutdown(t, pp)
	checkSignaled(t, finished)
}

func TestGenerateErr(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	finished := pl.NewSignal()
	squares, cherr := pl.GenerateErr(pp, func(wr pl.Writer[int]) error {
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
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	partial, cherr := pl.GenerateErr(pp, func(wr pl.Writer[int]) error {
		for k := range 5 {
			r := wr.Write(k * k)
			assert.True(t, r)
		}
		return testError
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
	assert.Equal(t, err, testError)

	// note: `partial` channel must be closed for this
	checkSignaled(t, finished)
}

func TestGenerateErrDontStuck(t *testing.T) {
	pp := pl.NewPipeline()

	finished := pl.NewSignal()
	_, cherr := pl.GenerateErr(pp, func(wr pl.Writer[int]) error {
		r := wr.Write(42)
		assert.False(t, r)
		finished.Set()
		return nil
	})

	// shutdown execution to unblock write
	checkShutdown(t, pp)

	checkPending(t, cherr)
	checkSignaled(t, finished)
}
