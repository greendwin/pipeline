package pipeline_test

import (
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestCollect(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	nums := sequence(pp, 0, 10)
	sum := pl.Collect(pp, func() int {
		sum := 0
		for v := range nums {
			sum += v
		}
		return sum
	})

	r := checkRead(t, sum)
	assert.Equal(t, r, 45)
}

func TestCollect_DontStuck(t *testing.T) {
	pp := pl.NewPipeline()

	inf := pl.Generate(pp, func(w pl.Writer[int]) {
		k := 0
		for w.Write(k) {
			k += 1
		}
	})

	sum := pl.Collect(pp, func() int {
		sum := 0
		for v := range inf {
			sum += v
		}
		return sum
	})

	checkShutdown(t, pp)
	_ = checkRead(t, sum)
}

func TestCollectErr(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	nums := sequence(pp, 0, 10)
	sum, cherr := pl.CollectErr(pp, func() (int, error) {
		sum := 0
		for v := range nums {
			sum += v
		}
		return sum, nil
	})

	r := checkRead(t, sum)
	assert.Equal(t, r, 45)

	checkPending(t, cherr) // no errors
}

func TestCollectErr_PropagateError(t *testing.T) {
	pp := pl.NewPipeline()

	passError := pl.NewSignal()
	sum, cherr := pl.CollectErr(pp, func() (int, error) {
		passError.Wait()
		return 0, errTest
	})

	checkPending(t, sum)
	checkPending(t, cherr)

	passError.Set()

	err := checkRead(t, cherr)
	assert.Equal(t, err, errTest)

	checkShutdown(t, pp)

	// oneshot never closed
	checkPending(t, sum)
}
