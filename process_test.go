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
	defer checkShutdown(t, cancel)

	sum := adder{}

	seq, seqErr := sequence(ctx, 0, 10)
	finished, cherr := pl.Process(ctx, 1, seq, func(x int) error {
		sum.Add(x)
		return nil
	})

	checkSignaled(t, finished)
	assert.Equal(t, sum.Value(), 45)

	// no errors
	checkPending(t, seqErr)
	checkPending(t, cherr)
}

func TestProcess_SpawnWorkers(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	numWorkers := 42

	started := sync.WaitGroup{}
	started.Add(numWorkers)

	input := make(chan int)
	stopProcessing := pl.NewSignal()

	finished, cherr := pl.Process(ctx, numWorkers, input, func(x int) error {
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

func TestProcess_Propagate(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	numWorkers := 42

	input := make(chan int)
	doFail := pl.NewSignal()

	finished, cherr := pl.Process(ctx, numWorkers, input, func(x int) error {
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
	checkShutdown(t, cancel)
}
