package discovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	opts := &Options{}

	ctx := context.Background()
	WithContext(ctx)(opts)
	assert.Equal(t, opts.Ctx, ctx)

	WithNamespace("ns")(opts)
	assert.Equal(t, opts.Namespace, "ns")
}
