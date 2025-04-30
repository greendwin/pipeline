package pipeline_test

import (
	"context"
	"sync"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func sequence(ctx context.Context, start, count int) (<-chan int, pl.Oneshot[error]) {
	return pl.Generate(ctx, func(wr pl.Writer[int]) error {
		for k := range count {
			if err := wr.Write(start + k); err != nil {
				return err
			}
		}
		return nil
	})
}

func TestTransform(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	seq, seqErr := sequence(ctx, 0, 10)
	seqAdd5, seqAdd5Err := pl.Transform(ctx, 1, seq, func(x int) (int, error) {
		return x + 5, nil
	})

	withTimeout(t, "read seq vals", func() {
		index := 0
		for k := range seqAdd5 {
			assert.Equal(t, k, index+5)
			index += 1
		}
		assert.Equal(t, index, 10)
	})

	checkPending(t, seqErr)
	checkPending(t, seqAdd5Err)
}

func TestTransform_SpawnWorkers(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	numWorkers := 42

	started := sync.WaitGroup{}
	started.Add(numWorkers)

	input := make(chan int)
	passResult := pl.NewSignal()

	type res struct{ val int }

	tr, cherr := pl.Transform(ctx, numWorkers, input, func(x int) (res, error) {
		started.Done()
		passResult.Wait()
		return res{x * 2}, nil
	})

	withTimeout(t, "count spawned workers", func() {
		for range numWorkers {
			input <- 42
		}
		started.Wait()
	})

	withTimeout(t, "check processed results", func() {
		close(input)
		checkPending(t, tr) // all workers are locked by processing

		passResult.Set()

		count := 0
		for val := range tr {
			assert.Equal(t, val.val, 84)
			count += 1
		}
		assert.Equal(t, count, numWorkers)
	})

	checkPending(t, cherr)
}

func TestTransform_Propagate(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	numWorkers := 42

	input := make(chan int)
	doFail := pl.NewSignal()

	tr, cherr := pl.Transform(ctx, numWorkers, input, func(x int) (int, error) {
		doFail.Wait()
		return 0, errTest
	})

	withTimeout(t, "receive error", func() {
		for range numWorkers {
			input <- 42
		}

		doFail.Set()

		err := checkRead(t, cherr)
		assert.Equal(t, err, errTest)
	})

	checkPending(t, tr) // nothing was processed, all failed

	// note: multiple errors were emitted simultaneously,
	// make sure that no goroutine was stuck
	checkShutdown(t, cancel)
}
