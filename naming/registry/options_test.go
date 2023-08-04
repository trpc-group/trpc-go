package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	opts := &Options{}

	WithAddress("ip://127.0.0.1:8000")(opts)
	assert.Equal(t, opts.Address, "ip://127.0.0.1:8000")
}
