package pipeline_test

import (
	"context"
	"fmt"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestWaitFirst(t *testing.T) {
	for index := range 10 {
		t.Run(fmt.Sprintf("FirstRecv(%d)", index), func(t *testing.T) {
			ctx, cancel := pl.NewPipeline(context.Background())
			defer checkShutdown(t, cancel)

			opts := make([]chan int, 10)
			optsIn := make([]<-chan int, len(opts))
			for k := range opts {
				ch := make(chan int)
				opts[k] = ch
				optsIn[k] = ch
			}

			finished := pl.NewSignal()
			go func() {
				v, ok := pl.WaitFirst(ctx, optsIn...)
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
	ctx, cancel := pl.NewPipeline(context.Background())

	neverSend1 := make(chan int)
	neverSend2 := make(chan int)

	finished := pl.NewSignal()
	go func() {
		_, ok := pl.WaitFirst(ctx, neverSend1, neverSend2)
		assert.False(t, ok)
		finished.Set()
	}()

	checkShutdown(t, cancel)
	checkSignaled(t, finished)
}

func TestWaitFirst_IgnoreClosedChannels(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	willClose1 := make(chan int)
	willClose2 := make(chan int)
	willSend := make(chan int, 1)

	finished := pl.NewSignal()
	go func() {
		v, ok := pl.WaitFirst(ctx, willClose1, willClose2, willSend)
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
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	willClose1 := make(chan int)
	willClose2 := make(chan int)

	finished := pl.NewSignal()
	go func() {
		_, ok := pl.WaitFirst(ctx, willClose1, willClose2)
		assert.False(t, ok)
		finished.Set()
	}()

	close(willClose1)
	checkPending(t, finished)

	close(willClose2)
	checkSignaled(t, finished)
}

func TestFirst(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	opts := make([]chan int, 10)
	optsIn := make([]<-chan int, len(opts))
	for k := range opts {
		ch := make(chan int)
		opts[k] = ch
		optsIn[k] = ch
	}

	res := pl.First(ctx, optsIn...)
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
	})
}
func TestFirst_AllClosed(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	willClose1 := make(chan int)
	willClose2 := make(chan int)

	res := pl.First(ctx, willClose1, willClose2)
	checkPending(t, res)

	close(willClose1)
	checkPending(t, res)

	close(willClose2)
	checkPending(t, res) // still pending, `Oneshot` channels are never closed
}

func TestFanIn(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	seq1 := sequence(ctx, 0, 10)
	seq2 := sequence(ctx, 10, 3)
	seq3 := sequence(ctx, 13, 7)

	merged := pl.FanIn(ctx, seq1, seq2, seq3)

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
	ctx, cancel := pl.NewPipeline(context.Background())

	neverSend1 := make(chan int)
	neverSend2 := make(chan int)
	neverSend3 := make(chan int)

	merged := pl.FanIn(ctx, neverSend1, neverSend2, neverSend3)

	finished := pl.NewSignal()
	go func() {
		_, ok := <-merged
		assert.False(t, ok)
		finished.Set()
	}()

	checkShutdown(t, cancel)
	checkSignaled(t, finished)
}

func TestFanIn_NeverStuckOnSend(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	seq1 := sequence(ctx, 0, 10)
	seq2 := sequence(ctx, 10, 3)
	seq3 := sequence(ctx, 13, 7)

	merged := pl.FanIn(ctx, seq1, seq2, seq3)

	checkShutdown(t, cancel)
	checkSignaled(t, merged)
}
