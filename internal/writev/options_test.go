package writev

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithOptions(t *testing.T) {
	opts := &Options{}
	WithBufferSize(128)(opts)
	WithDropFull(true)(opts)
	assert.Equal(t, opts.bufferSize, 128)
	assert.Equal(t, opts.dropFull, true)
}
