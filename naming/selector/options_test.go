//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package selector

import (
	"context"
	"testing"

	"trpc.group/trpc-go/trpc-go/naming/circuitbreaker"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	opts := &Options{}
	ctx := context.Background()
	WithContext(ctx)(opts)
	WithKey("key")(opts)
	WithReplicas(100)(opts)
	WithSourceSetName("set")(opts)
	WithDestinationSetName("dstSet")(opts)
	d := &discovery.IPDiscovery{}
	WithDiscovery(d)(opts)
	r := &servicerouter.NoopServiceRouter{}
	WithServiceRouter(r)(opts)
	b := loadbalance.NewRandom()
	WithLoadBalancer(b)(opts)
	cb := &circuitbreaker.NoopCircuitBreaker{}
	WithCircuitBreaker(cb)(opts)
	WithDisableServiceRouter()(opts)
	WithDestinationEnvName("dst_env")(opts)
	WithNamespace("test_namespace")(opts)
	WithSourceNamespace("src_namespace")(opts)
	WithEnvKey("env_key")(opts)
	WithSourceServiceName("src_svcname")(opts)
	WithSourceEnvName("src_env")(opts)
	WithSourceMetadata("srcMeta", "value")(opts)
	WithDestinationMetadata("dstMeta", "value")(opts)
	WithEnvTransfer("env_transfer")(opts)
	WithLoadBalanceType("hash")(opts)

	assert.Equal(t, opts.Ctx, ctx)
	assert.Equal(t, opts.SourceSetName, "set")
	assert.Equal(t, opts.Key, "key")
	assert.Equal(t, opts.Replicas, 100)
	assert.Equal(t, opts.CircuitBreaker, cb)
	assert.Equal(t, opts.LoadBalancer, b)
	assert.Equal(t, opts.Discovery, d)
	assert.Equal(t, opts.ServiceRouter, r)
	assert.True(t, opts.DisableServiceRouter)
	assert.Equal(t, opts.DestinationEnvName, "dst_env")
	assert.Equal(t, opts.DestinationSetName, "dstSet")
	assert.Equal(t, opts.Namespace, "test_namespace")
	assert.Equal(t, opts.SourceNamespace, "src_namespace")
	assert.Equal(t, opts.SourceServiceName, "src_svcname")
	assert.Equal(t, opts.EnvKey, "env_key")
	assert.Equal(t, opts.SourceEnvName, "src_env")
	assert.Equal(t, opts.SourceMetadata["srcMeta"], "value")
	assert.Equal(t, opts.DestinationMetadata["dstMeta"], "value")
	assert.Equal(t, opts.EnvTransfer, "env_transfer")
	assert.Len(t, opts.LoadBalanceOptions, 5)
}
