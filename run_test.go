package pipeline_test

import (
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	processed := pl.NewSignal()
	finished := pl.Run(pp, func() {
		processed.Wait()
	})

	checkPending(t, finished)
	processed.Set()
	checkSignaled(t, finished)
}

func TestRunErr(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	processed := pl.NewSignal()
	finished, cherr := pl.RunErr(pp, func() error {
		processed.Wait()
		return nil
	})

	checkPending(t, finished)
	processed.Set()
	checkSignaled(t, finished)

	checkPending(t, cherr) // no errors
}

func TestRunErr_PropagateError(t *testing.T) {
	pp := pl.NewPipeline()
	defer checkShutdown(t, pp)

	doFail := pl.NewSignal()
	finished, cherr := pl.RunErr(pp, func() error {
		doFail.Wait()
		return errTest
	})

	checkPending(t, finished)
	doFail.Set()

	err := checkRead(t, cherr)
	assert.Equal(t, err, errTest)

	checkPending(t, finished) // don't trigger `finished` on error
}
