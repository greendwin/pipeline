package pipeline_test

import (
	"context"
	"fmt"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestWrite(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	recv := make(chan int, 10)

	withTimeout(t, "write values", func() {
		for k := range cap(recv) {
			err := pl.Write(ctx, recv, k)
			assert.Nil(t, err)
		}
	})
}

func TestWrite_DontStuck(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	neverRecv := make(chan int)

	pl.Go(ctx, func() {
		err := pl.Write(ctx, neverRecv, 42)
		assert.ErrorIs(t, err, context.Canceled)
	})

	checkShutdown(t, cancel)
}

func TestRead(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	vals := make(chan int, 10)
	for k := range cap(vals) {
		vals <- k
	}
	close(vals)

	withTimeout(t, "read values", func() {
		for k := range cap(vals) {
			v, err := pl.Read(ctx, vals)
			assert.Nil(t, err)
			assert.Equal(t, v, k)
		}

		for range 3 {
			_, err := pl.Read(ctx, vals)
			assert.ErrorIs(t, err, context.Canceled)
		}
	})
}

func TestRead_NeverStuck(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	neverSend := make(chan int)

	finished := pl.NewSignal()
	pl.Go(ctx, func() {
		_, err := pl.Read(ctx, neverSend)
		assert.ErrorIs(t, err, context.Canceled)
		finished.Set()
	})

	checkShutdown(t, cancel)
	checkSignaled(t, finished)
}

func TestReadErr(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	res := make(chan int)

	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.Run(ctx, func() {
		v, err := pl.ReadErr(ctx, res, err1, err2)
		assert.Nil(t, err)
		assert.Equal(t, v, 42)
	})

	checkPending(t, finished)
	res <- 42
	checkSignaled(t, finished)
}

func TestReadErr_FinishWithError(t *testing.T) {
	for index := range 10 {
		t.Run(fmt.Sprintf("fail on errs[%d]", index), func(t *testing.T) {
			ctx, cancel := pl.NewPipeline(context.Background())
			defer checkShutdown(t, cancel)

			res := make(chan int)

			var willFail chan error
			errs := make([]<-chan error, 10)
			for k := range errs {
				cherr := make(chan error)
				if k == index {
					willFail = cherr
				}
				errs[k] = cherr
			}

			finished := pl.Run(ctx, func() {
				_, err := pl.ReadErr(ctx, res, errs...)
				assert.Equal(t, err, errTest)
			})

			checkPending(t, finished)
			willFail <- errTest
			checkSignaled(t, finished)
		})
	}
}

func TestReadErr_ReportChannelClosed(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	res := make(chan int)

	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.Run(ctx, func() {
		_, err := pl.ReadErr(ctx, res, err1, err2)
		assert.Equal(t, err, pl.ErrChannelClosed)
	})

	checkPending(t, finished)
	close(res)
	checkSignaled(t, finished)
}

func TestReadErr_ErrorsCanClose(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	res := make(chan int)

	err1 := make(chan error, 1)
	err2 := make(chan error, 1)

	finished := pl.Run(ctx, func() {
		_, err := pl.ReadErr(ctx, res, err1, err2)
		assert.Equal(t, err, errTest)
	})

	close(err1)
	checkPending(t, finished)

	err2 <- errTest
	checkSignaled(t, finished)
}

func TestReadErr_ReportCancelled(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	res := make(chan int)
	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.NewSignal()
	go func() {
		_, err := pl.ReadErr(ctx, res, err1, err2)
		assert.ErrorIs(t, err, context.Canceled)
		finished.Set()
	}()

	checkPending(t, finished)
	checkShutdown(t, cancel)

	checkSignaled(t, finished)
}

func TestReadErr_FallbackToRead(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	res := make(chan int)
	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.NewSignal()
	go func() {
		val, err := pl.ReadErr(ctx, res, err1, err2)
		assert.Nil(t, err)
		assert.Equal(t, val, 42)
		finished.Set()
	}()

	checkPending(t, finished)

	// close both error channels, it still must wait for value
	close(err1)
	close(err2)
	checkPending(t, finished)

	res <- 42
	checkSignaled(t, finished)
}

func TestReadErr_ReportWhanAllClosed(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	res := make(chan int)
	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.NewSignal()
	go func() {
		_, err := pl.ReadErr(ctx, res, err1, err2)
		assert.Equal(t, err, pl.ErrChannelClosed)
		finished.Set()
	}()

	checkPending(t, finished)

	close(err1)
	close(err2)
	checkPending(t, finished)

	close(res)
	checkSignaled(t, finished)
}

func TestReadErr_ReturnErrorCause(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())

	neverSend := make(chan int)
	neverFail := make(chan error)

	finished := pl.NewSignal()
	go func() {
		_, err := pl.ReadErr(ctx, neverSend, neverFail)
		assert.ErrorIs(t, err, errTest)
		finished.Set()
	}()

	checkPending(t, finished)
	cancel(errTest)
	checkSignaled(t, finished)
}
