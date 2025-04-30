package pipeline_test

import (
	"context"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestCollect(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	nums := sequence(ctx, 0, 10)
	sum := pl.Collect(ctx, func() int {
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
	ctx, cancel := pl.NewPipeline(context.Background())

	inf := pl.Generate(ctx, func(w pl.Writer[int]) {
		k := 0
		for w.Write(k) {
			k += 1
		}
	})

	sum := pl.Collect(ctx, func() int {
		sum := 0
		for v := range inf {
			sum += v
		}
		return sum
	})

	checkShutdown(t, cancel)
	_ = checkRead(t, sum)
}

func TestCollectErr(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	nums := sequence(ctx, 0, 10)
	sum, cherr := pl.CollectErr(ctx, func() (int, error) {
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
	ctx, cancel := pl.NewPipeline(context.Background())

	passError := pl.NewSignal()
	sum, cherr := pl.CollectErr(ctx, func() (int, error) {
		passError.Wait()
		return 0, errTest
	})

	checkPending(t, sum)
	checkPending(t, cherr)

	passError.Set()

	err := checkRead(t, cherr)
	assert.Equal(t, err, errTest)

	checkShutdown(t, cancel)

	// oneshot never closed
	checkPending(t, sum)
}
