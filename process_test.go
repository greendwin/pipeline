package pipeline_test

import (
	"context"
	"sync"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

type adder struct {
	val int
	mu  sync.Mutex
}

func (a *adder) Add(val int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.val += val
}

func (a *adder) Value() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.val
}

func TestProcess(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, ctx, cancel)

	sum := adder{}

	seq := sequence(ctx, 0, 10)
	finished := pl.Process(ctx, 1, seq, func(x int) {
		sum.Add(x)
	})

	checkSignaled(t, finished)
	assert.Equal(t, sum.Value(), 45)
}

func TestProcessSpawnWorkers(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, ctx, cancel)

	numWorkers := 42

	started := sync.WaitGroup{}
	started.Add(numWorkers)

	input := make(chan int)
	stopProcessing := pl.NewSignal()

	finished := pl.Process(ctx, numWorkers, input, func(x int) {
		started.Done()
		stopProcessing.Wait()
	})

	withTimeout(t, "count spawned workers", func() {
		for range numWorkers {
			input <- 42
		}
		started.Wait()
	})

	stopProcessing.Set()
	checkPending(t, finished) // network is waiting for new input values

	close(input)
	checkSignaled(t, finished)
}

func TestProcessErr(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, ctx, cancel)

	sum := adder{}

	seq := sequence(ctx, 0, 10)
	finished, cherr := pl.ProcessErr(ctx, 1, seq, func(x int) error {
		sum.Add(x)
		return nil
	})

	checkSignaled(t, finished)
	assert.Equal(t, sum.Value(), 45)

	checkPending(t, cherr) // no errors
}

func TestProcessErrSpawnWorkers(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, ctx, cancel)

	numWorkers := 42

	started := sync.WaitGroup{}
	started.Add(numWorkers)

	input := make(chan int)
	stopProcessing := pl.NewSignal()

	finished, cherr := pl.ProcessErr(ctx, numWorkers, input, func(x int) error {
		started.Done()
		stopProcessing.Wait()
		return nil
	})

	withTimeout(t, "count spawned workers", func() {
		for range numWorkers {
			input <- 42
		}
		started.Wait()
	})

	stopProcessing.Set()
	checkPending(t, finished) // network is waiting for more input

	close(input)
	checkSignaled(t, finished)

	checkPending(t, cherr) // no errors
}

func TestProcessErrPropagate(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	numWorkers := 42

	input := make(chan int)
	doFail := pl.NewSignal()

	finished, cherr := pl.ProcessErr(ctx, numWorkers, input, func(x int) error {
		doFail.Wait()
		return errTest
	})

	withTimeout(t, "receive error", func() {
		for range numWorkers {
			input <- 42
		}

		doFail.Set()

		err := checkRead(t, cherr)
		assert.Equal(t, err, errTest)
	})

	checkPending(t, finished) // nothing was processed, all failed

	// note: multiple errors were emitted simultaneously,
	// make sure that no goroutine was stuck
	checkShutdown(t, ctx, cancel)
}
