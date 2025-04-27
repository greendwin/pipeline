package pipeline_test

import (
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
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	sum := adder{}

	seq := sequence(pp, 0, 10)
	finished := pl.Process(pp, 1, seq, func(x int) {
		sum.Add(x)
	})

	checkSignaled(t, finished)
	assert.Equal(t, sum.Value(), 45)
}

func TestProcessSpawnWorkers(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	numWorkers := 42

	started := sync.WaitGroup{}
	started.Add(numWorkers)

	input := make(chan int)
	stopProcessing := pl.NewSignal()

	finished := pl.Process(pp, numWorkers, input, func(x int) {
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
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	sum := adder{}

	seq := sequence(pp, 0, 10)
	finished, cherr := pl.ProcessErr(pp, 1, seq, func(x int) error {
		sum.Add(x)
		return nil
	})

	checkSignaled(t, finished)
	assert.Equal(t, sum.Value(), 45)

	checkPending(t, cherr) // no errors
}

func TestProcessErrSpawnWorkers(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	numWorkers := 42

	started := sync.WaitGroup{}
	started.Add(numWorkers)

	input := make(chan int)
	stopProcessing := pl.NewSignal()

	finished, cherr := pl.ProcessErr(pp, numWorkers, input, func(x int) error {
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
	pp := pl.NewPipeline()

	numWorkers := 42

	input := make(chan int)
	doFail := pl.NewSignal()

	finished, cherr := pl.ProcessErr(pp, numWorkers, input, func(x int) error {
		doFail.Wait()
		return testError
	})

	withTimeout(t, "receive error", func() {
		for range numWorkers {
			input <- 42
		}

		doFail.Set()

		err := checkRead(t, cherr)
		assert.Equal(t, err, testError)
	})

	checkPending(t, finished) // nothing was processed, all failed

	// note: multiple errors were emitted simultaneously,
	// make sure that no goroutine was stuck
	checkShutdown(t, pp)
}
