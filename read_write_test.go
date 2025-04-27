package pipeline_test

import (
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestWriteValues(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	recv := make(chan int, 10)

	withTimeout(t, "write values", func() {
		for k := range cap(recv) {
			ok := pl.Write(pp, recv, k)
			assert.True(t, ok)
		}
	})
}

func TestWriteDontStuck(t *testing.T) {
	pp := pl.NewPipeline()
	neverRecv := make(chan int)

	pp.Go(func() {
		ok := pl.Write(pp, neverRecv, 42)
		assert.False(t, ok)
	})

	checkShutdown(t, pp)
}

func TestReadValue(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	vals := make(chan int, 10)
	for k := range cap(vals) {
		vals <- k
	}
	close(vals)

	withTimeout(t, "read values", func() {
		for k := range cap(vals) {
			v, ok := pl.Read(pp, vals)
			assert.True(t, ok)
			assert.Equal(t, v, k)
		}

		for range 3 {
			_, ok := pl.Read(pp, vals)
			assert.False(t, ok)
		}
	})
}

func TestReadNeverStuck(t *testing.T) {
	pp := pl.NewPipeline()

	neverSend := make(chan int)

	finished := pl.NewSignal()
	pp.Go(func() {
		_, ok := pl.Read(pp, neverSend)
		assert.False(t, ok)
		finished.Set()
	})

	checkShutdown(t, pp)
	checkSignaled(t, finished)
}
