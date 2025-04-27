package pipeline_test

import (
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func squares(pp *pl.Pipeline, count int) <-chan int {
	return pl.Generate(pp, func(wr pl.Writer[int]) {
		for k := range count {
			if !wr.Write(k * k) {
				return
			}
		}
	})
}

func TestGenerate(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	seq := squares(pp, 10)

	withTimeout(t, "read seq", func() {
		index := 0
		for val := range seq {
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
