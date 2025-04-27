package pipeline_test

import (
	"fmt"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestWaitFirst(t *testing.T) {
	for index := range 10 {
		t.Run(fmt.Sprintf("FirstRecv(%d)", index), func(t *testing.T) {
			pp := pl.NewPipeline()
			defer checkShutdown(t, pp)

			opts := make([]chan int, 10)
			optsIn := make([]<-chan int, len(opts))
			for k := range opts {
				ch := make(chan int)
				opts[k] = ch
				optsIn[k] = ch
			}

			finished := pl.NewSignal()
			go func() {
				v, ok := pl.WaitFirst(pp, optsIn...)
				assert.True(t, ok)
				assert.Equal(t, v, 42)
				finished.Set()
			}()

			checkPending(t, finished)
			opts[index] <- 42
			checkSignaled(t, finished)
		})
	}
}

func TestWaitFirst_NeverStuck(t *testing.T) {
	pp := pl.NewPipeline()

	neverSend1 := make(chan int)
	neverSend2 := make(chan int)

	finished := pl.NewSignal()
	go func() {
		_, ok := pl.WaitFirst(pp, neverSend1, neverSend2)
		assert.False(t, ok)
		finished.Set()
	}()

	checkShutdown(t, pp)
	checkSignaled(t, finished)
}

func TestWaitFirst_IgnoreClosedChannels(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	willClose1 := make(chan int)
	willClose2 := make(chan int)
	willSend := make(chan int, 1)

	finished := pl.NewSignal()
	go func() {
		v, ok := pl.WaitFirst(pp, willClose1, willClose2, willSend)
		assert.True(t, ok)
		assert.Equal(t, v, 42)
		finished.Set()
	}()

	close(willClose1)
	checkPending(t, finished)

	close(willClose2)
	checkPending(t, finished)

	willSend <- 42
	checkSignaled(t, finished)
}

func TestWaitFirst_ReturnNoneWhenAllClosed(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	willClose1 := make(chan int)
	willClose2 := make(chan int)

	finished := pl.NewSignal()
	go func() {
		_, ok := pl.WaitFirst(pp, willClose1, willClose2)
		assert.False(t, ok)
		finished.Set()
	}()

	close(willClose1)
	checkPending(t, finished)

	close(willClose2)
	checkSignaled(t, finished)
}

func TestFirst(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	opts := make([]chan int, 10)
	optsIn := make([]<-chan int, len(opts))
	for k := range opts {
		ch := make(chan int)
		opts[k] = ch
		optsIn[k] = ch
	}

	res := pl.First(pp, optsIn...)
	checkPending(t, res)

	close(opts[4])
	checkPending(t, res)

	close(opts[7])
	checkPending(t, res)

	withTimeout(t, "read channel", func() {
		opts[1] <- 42

		v, ok := <-res
		assert.True(t, ok)
		assert.Equal(t, v, 42)

		// make sure channel is closed
		_, ok = <-res
		assert.False(t, ok)
	})
}
func TestFirst_AllClosed(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	willClose1 := make(chan int)
	willClose2 := make(chan int)

	res := pl.First(pp, willClose1, willClose2)
	checkPending(t, res)

	close(willClose1)
	checkPending(t, res)

	close(willClose2)
	withTimeout(t, "check channel closed", func() {
		// make sure channel is closed without value
		_, ok := <-res
		assert.False(t, ok)
	})
}

func TestFanIn(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	seq1 := sequence(pp, 0, 10)
	seq2 := sequence(pp, 10, 3)
	seq3 := sequence(pp, 13, 7)

	merged := pl.FanIn(pp, seq1, seq2, seq3)

	withTimeout(t, "read merged channel", func() {
		received := make([]bool, 20)
		for v := range merged {
			received[v] = true
		}

		for k, ok := range received {
			assert.True(t, ok, "k=", k)
		}
	})
}

func TestFanIn_NeverStuckOnRecv(t *testing.T) {
	pp := pl.NewPipeline()

	neverSend1 := make(chan int)
	neverSend2 := make(chan int)
	neverSend3 := make(chan int)

	merged := pl.FanIn(pp, neverSend1, neverSend2, neverSend3)

	finished := pl.NewSignal()
	go func() {
		_, ok := <-merged
		assert.False(t, ok)
		finished.Set()
	}()

	checkShutdown(t, pp)
	checkSignaled(t, finished)
}

func TestFanIn_NeverStuckOnSend(t *testing.T) {
	pp := pl.NewPipeline()

	seq1 := sequence(pp, 0, 10)
	seq2 := sequence(pp, 10, 3)
	seq3 := sequence(pp, 13, 7)

	merged := pl.FanIn(pp, seq1, seq2, seq3)

	checkShutdown(t, pp)
	checkSignaled(t, merged)
}
