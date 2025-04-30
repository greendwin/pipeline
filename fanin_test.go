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
				v, err := pl.WaitFirst(ctx, optsIn...)
				assert.Nil(t, err)
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
		_, err := pl.WaitFirst(ctx, neverSend1, neverSend2)
		assert.ErrorIs(t, err, context.Canceled)
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
		v, err := pl.WaitFirst(ctx, willClose1, willClose2, willSend)
		assert.Nil(t, err)
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
		_, err := pl.WaitFirst(ctx, willClose1, willClose2)
		assert.ErrorIs(t, err, pl.ErrChannelClosed)
		finished.Set()
	}()

	close(willClose1)
	checkPending(t, finished)

	close(willClose2)
	checkSignaled(t, finished)
}

func TestWaitFirst_ReturnCauseError(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())

	ch1 := make(chan int)
	ch2 := make(chan int)

	finished := pl.NewSignal()
	go func() {
		_, err := pl.WaitFirst(ctx, ch1, ch2)
		assert.ErrorIs(t, err, errTest)
		finished.Set()
	}()

	checkPending(t, finished)
	cancel(errTest)
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

	res, cherr := pl.First(ctx, optsIn...)
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

	checkPending(t, cherr) // no errors
}
func TestFirst_AllClosed(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	willClose1 := make(chan int)
	willClose2 := make(chan int)

	res, cherr := pl.First(ctx, willClose1, willClose2)
	checkPending(t, res)

	close(willClose1)
	checkPending(t, res)

	close(willClose2)
	checkPending(t, res) // still pending, `Oneshot` channels are never closed

	err := checkRead(t, cherr)
	assert.ErrorIs(t, err, pl.ErrChannelClosed)
}

func TestFirstErr(t *testing.T) {
	for _, first := range [...]bool{true, false} {
		ctx, cancel := pl.NewPipeline(context.Background())
		defer checkShutdown(t, cancel)

		cherr1 := pl.NewOneshot[error]()
		cherr2 := pl.NewOneshot[error]()

		cherr := pl.FirstErr(ctx, cherr1.Chan(), cherr2.Chan())
		checkPending(t, cherr)

		if first {
			cherr1.Write(errTest)
		} else {
			cherr2.Write(errTest)
		}

		err := checkRead(t, cherr)
		assert.ErrorIs(t, err, errTest)
	}
}

func TestFirstErr_AllClosed(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	willClose1 := make(chan error)
	willClose2 := make(chan error)

	cherr := pl.FirstErr(ctx, willClose1, willClose2)

	close(willClose1)
	checkPending(t, cherr) // wait second channel

	close(willClose2)

	err := checkRead(t, cherr)
	assert.ErrorIs(t, err, pl.ErrChannelClosed)
}

func TestFanIn(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	seq1 := sequence(ctx, 0, 10)
	seq2 := sequence(ctx, 10, 3)
	seq3 := sequence(ctx, 13, 7)

	merged, cherr := pl.FanIn(ctx, seq1, seq2, seq3)

	withTimeout(t, "read merged channel", func() {
		received := make([]bool, 20)
		for v := range merged {
			received[v] = true
		}

		for k, ok := range received {
			assert.True(t, ok, "k=", k)
		}
	})

	checkPending(t, cherr) // no errors
}

func TestFanIn_NeverStuckOnRecv(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	neverSend1 := make(chan int)
	neverSend2 := make(chan int)
	neverSend3 := make(chan int)

	merged, cherr := pl.FanIn(ctx, neverSend1, neverSend2, neverSend3)

	checkShutdown(t, cancel)

	checkPending(t, merged)
	err := checkRead(t, cherr)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFanIn_NeverStuckOnSend(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	seq1 := sequence(ctx, 0, 10)
	seq2 := sequence(ctx, 10, 3)
	seq3 := sequence(ctx, 13, 7)

	merged, cherr := pl.FanIn(ctx, seq1, seq2, seq3)

	checkShutdown(t, cancel)

	checkPending(t, merged)
	err := checkRead(t, cherr)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestFanIn_PropagateCause(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())

	seq1 := sequence(ctx, 0, 10)
	seq2 := sequence(ctx, 10, 3)

	merged, cherr := pl.FanIn(ctx, seq1, seq2)

	cancel(errTest)

	err := checkRead(t, cherr)
	assert.ErrorIs(t, err, errTest)

	checkPending(t, merged) // would not close
}
