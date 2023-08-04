package multiplexed

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestPoolOptions test the configuration items of the multiplexed pool.
func TestPoolOptions(t *testing.T) {
	opts := &PoolOptions{}
	WithConnectNumber(50000)(opts)
	WithQueueSize(20000)(opts)
	WithDropFull(true)(opts)
	assert.Equal(t, opts.connectNumberPerHost, 50000)
	assert.Equal(t, opts.sendQueueSize, 20000)
	assert.Equal(t, opts.dropFull, true)
}
