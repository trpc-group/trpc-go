package servicerouter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	opts := &Options{}

	ctx := context.Background()
	WithContext(ctx)(opts)
	WithNamespace("ns")(opts)
	WithSourceNamespace("sns")(opts)
	WithSourceServiceName("sname")(opts)
	WithSourceEnvName("envname")(opts)
	WithEnvTransfer("envtransfer")(opts)
	WithEnvKey("env")(opts)
	WithDisableServiceRouter()(opts)
	WithDestinationEnvName("dst_env")(opts)
	WithSourceSetName("setname")(opts)
	WithDestinationSetName("dstSetName")(opts)
	WithSourceMetadata("srcMeta", "value")(opts)
	WithDestinationMetadata("dstMeta", "value")(opts)

	assert.Equal(t, opts.Ctx, ctx)
	assert.Equal(t, opts.Namespace, "ns")
	assert.Equal(t, opts.SourceNamespace, "sns")
	assert.Equal(t, opts.SourceServiceName, "sname")
	assert.Equal(t, opts.SourceEnvName, "envname")
	assert.Equal(t, opts.SourceSetName, "setname")
	assert.Equal(t, opts.EnvTransfer, "envtransfer")
	assert.Equal(t, opts.EnvKey, "env")
	assert.True(t, opts.DisableServiceRouter)
	assert.Equal(t, opts.DestinationEnvName, "dst_env")
	assert.Equal(t, opts.DestinationSetName, "dstSetName")
	assert.Equal(t, opts.SourceMetadata["srcMeta"], "value")
	assert.Equal(t, opts.DestinationMetadata["dstMeta"], "value")
}
