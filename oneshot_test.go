package pipeline_test

import (
	"testing"

	pl "github.com/greendwin/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestOneshot(t *testing.T) {
	ch := pl.NewOneshot[int]()
	checkPending(t, ch.Chan())

	withTimeout(t, "writing to oneshot channel", func() {
		ch.Write(42) // it must not block
	})

	r := checkRead(t, ch.Chan())
	assert.Equal(t, r, 42)
}
