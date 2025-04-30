package pipeline_test

import (
	"context"
	"fmt"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestWriteValues(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	recv := make(chan int, 10)

	withTimeout(t, "write values", func() {
		for k := range cap(recv) {
			ok := pl.Write(ctx, recv, k)
			assert.True(t, ok)
		}
	})
}

func TestWriteDontStuck(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	neverRecv := make(chan int)

	pl.Go(ctx, func() {
		ok := pl.Write(ctx, neverRecv, 42)
		assert.False(t, ok)
	})

	checkShutdown(t, cancel)
}

func TestReadValue(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())
	defer checkShutdown(t, cancel)

	vals := make(chan int, 10)
	for k := range cap(vals) {
		vals <- k
	}
	close(vals)

	withTimeout(t, "read values", func() {
		for k := range cap(vals) {
			v, ok := pl.Read(ctx, vals)
			assert.True(t, ok)
			assert.Equal(t, v, k)
		}

		for range 3 {
			_, ok := pl.Read(ctx, vals)
			assert.False(t, ok)
		}
	})
}

func TestReadNeverStuck(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

	neverSend := make(chan int)

	finished := pl.NewSignal()
	pl.Go(ctx, func() {
		_, ok := pl.Read(ctx, neverSend)
		assert.False(t, ok)
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

func TestReadErrFinishWithError(t *testing.T) {
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

func TestReadErrReportChannelClosed(t *testing.T) {
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

func TestReadErrErrorsCanClose(t *testing.T) {
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

func TestReadErrReportCancelled(t *testing.T) {
	ctx, cancel := pl.NewPipeline(context.Background())

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
	checkShutdown(t, cancel)

	checkSignaled(t, finished)
}

func TestReadErrFallbackToRead(t *testing.T) {
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

func TestReadErrReportWhanAllClosed(t *testing.T) {
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
