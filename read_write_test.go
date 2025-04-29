package pipeline_test

import (
	"fmt"
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestWriteValues(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	recv := make(chan int, 10)

	withTimeout(t, "write values", func() {
		for k := range cap(recv) {
			ok := pl.Write(pp, recv, k)
			assert.True(t, ok)
		}
	})
}

func TestWriteDontStuck(t *testing.T) {
	pp := pl.NewPipeline()
	neverRecv := make(chan int)

	pp.Go(func() {
		ok := pl.Write(pp, neverRecv, 42)
		assert.False(t, ok)
	})

	checkShutdown(t, pp)
}

func TestReadValue(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	vals := make(chan int, 10)
	for k := range cap(vals) {
		vals <- k
	}
	close(vals)

	withTimeout(t, "read values", func() {
		for k := range cap(vals) {
			v, ok := pl.Read(pp, vals)
			assert.True(t, ok)
			assert.Equal(t, v, k)
		}

		for range 3 {
			_, ok := pl.Read(pp, vals)
			assert.False(t, ok)
		}
	})
}

func TestReadNeverStuck(t *testing.T) {
	pp := pl.NewPipeline()

	neverSend := make(chan int)

	finished := pl.NewSignal()
	pp.Go(func() {
		_, ok := pl.Read(pp, neverSend)
		assert.False(t, ok)
		finished.Set()
	})

	checkShutdown(t, pp)
	checkSignaled(t, finished)
}

func TestReadErr(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	res := make(chan int)

	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.Run(pp, func() {
		v, err := pl.ReadErr(pp, res, err1, err2)
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
			pp := pl.NewPipeline()
			defer checkShutdown(t, pp)

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

			finished := pl.Run(pp, func() {
				_, err := pl.ReadErr(pp, res, errs...)
				assert.Equal(t, err, errTest)
			})

			checkPending(t, finished)
			willFail <- errTest
			checkSignaled(t, finished)
		})
	}
}

func TestReadErrReportChannelClosed(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	res := make(chan int)

	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.Run(pp, func() {
		_, err := pl.ReadErr(pp, res, err1, err2)
		assert.Equal(t, err, pl.ErrChannelClosed)
	})

	checkPending(t, finished)
	close(res)
	checkSignaled(t, finished)
}

func TestReadErrReportCancelled(t *testing.T) {
	pp := pl.NewPipeline()

	res := make(chan int)
	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.NewSignal()
	go func() {
		_, err := pl.ReadErr(pp, res, err1, err2)
		assert.Equal(t, err, pl.ErrChannelClosed)
		finished.Set()
	}()

	checkPending(t, finished)
	checkShutdown(t, pp)

	checkSignaled(t, finished)
}

func TestReadErrFallbackToRead(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	res := make(chan int)
	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.NewSignal()
	go func() {
		val, err := pl.ReadErr(pp, res, err1, err2)
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
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	res := make(chan int)
	err1 := make(chan error)
	err2 := make(chan error)

	finished := pl.NewSignal()
	go func() {
		_, err := pl.ReadErr(pp, res, err1, err2)
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
