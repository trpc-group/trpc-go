package loadbalance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	opts := &Options{}
	ctx := context.Background()
	WithContext(ctx)(opts)
	WithNamespace("ns")(opts)
	WithInterval(time.Second * 2)(opts)
	WithKey("hash key")(opts)
	WithReplicas(2)(opts)
	WithLoadBalanceType("hash")(opts)

	assert.Equal(t, opts.Ctx, ctx)
	assert.Equal(t, opts.Namespace, "ns")
	assert.Equal(t, opts.Interval, time.Second*2)
	assert.Equal(t, opts.Key, "hash key")
	assert.Equal(t, opts.Replicas, 2)
	assert.Equal(t, opts.LoadBalanceType, "hash")
}
