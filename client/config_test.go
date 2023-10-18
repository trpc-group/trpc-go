//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package client_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/internal/rand"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/transport"
)

func TestConfigOptions(t *testing.T) {
	clientOpts := &client.Options{}
	transportOpts := &transport.RoundTripOptions{}

	node := &registry.Node{
		Address:  "127.0.0.1:8080",
		Network:  "udp",
		Protocol: "trpc",
	}
	clientOpts.LoadNodeConfig(node)
	for _, o := range clientOpts.CallOptions {
		o(transportOpts)
	}
	assert.Equal(t, "127.0.0.1:8080", transportOpts.Address)
	assert.Equal(t, "udp", transportOpts.Network)
	assert.Equal(t, trpc.DefaultClientCodec, clientOpts.Codec)

	filter.Register("Monitoring", filter.NoopServerFilter, filter.NoopClientFilter)
	filter.Register("Authentication", filter.NoopServerFilter, filter.NoopClientFilter)
}

func TestConfigNoDiscovery(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName: "trpc.test.helloworld3", // backend service name
		Namespace:   "Development",
		Discovery:   "no-exists",
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
		Filter:      []string{"Monitoring", "Authentication"},
	}
	err := client.RegisterClientConfig("trpc.test.nodiscovery", backconfig)
	assert.NotNil(t, err)
}

func TestConfigNoServiceRouter(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName:   "trpc.test.helloworld3", // backend service name
		Namespace:     "Development",
		ServiceRouter: "no-exists",
		Network:       "tcp",
		Timeout:       1000,
		Protocol:      "trpc",
		Filter:        []string{"Monitoring", "Authentication"},
	}
	err := client.RegisterClientConfig("trpc.test.noservicerouter", backconfig)
	assert.NotNil(t, err)
}

func TestConfigNoBalance(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName: "trpc.test.helloworld3", // backend service name
		Namespace:   "Development",
		Loadbalance: "no-exists",
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
		Filter:      []string{"Monitoring", "Authentication"},
	}
	err := client.RegisterClientConfig("trpc.test.nobalance", backconfig)
	assert.NotNil(t, err)
}

type testOptionsSelector struct {
	f func(*selector.Options)
}

var testOptionsSelectorError = errors.New("test options selector error")

func (s *testOptionsSelector) Select(serviceName string, opt ...selector.Option) (*registry.Node, error) {
	opts := &selector.Options{}
	for _, o := range opt {
		o(opts)
	}
	s.f(opts)
	return nil, testOptionsSelectorError
}

func (s *testOptionsSelector) Report(node *registry.Node, cost time.Duration, err error) error {
	return nil
}

func TestConfigCalleeMetadata(t *testing.T) {
	ctx := context.Background()
	reqBody := codec.Body{}
	rspBody := codec.Body{}
	calleeMetadata := map[string]string{
		"key1": "val1",
		"key2": "val2",
	}
	backconfig := &client.BackendConfig{
		ServiceName:    "trpc.test.helloworld3", // backend service name
		Namespace:      "Development",
		Network:        "tcp",
		Timeout:        1000,
		Protocol:       "trpc",
		CalleeMetadata: calleeMetadata,
	}
	err := client.RegisterClientConfig("trpc.test.client.metadata", backconfig)
	assert.Nil(t, err)

	s := &testOptionsSelector{
		f: func(opts *selector.Options) {
			assert.Equal(t, calleeMetadata, opts.DestinationMetadata)
		},
	}
	selector.Register("test-options-selector", s)
	cli := client.New()
	assert.Equal(t, cli, client.DefaultClient)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithCalleeServiceName("trpc.test.client.metadata")
	err = cli.Invoke(ctx, reqBody, rspBody,
		client.WithTarget("test-options-selector://trpc.test.client.metadata"),
	)
	require.Contains(t, err.Error(), testOptionsSelectorError.Error())
}

func TestConfigNoBreaker(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName:    "trpc.test.helloworld3", // backend service name
		Namespace:      "Development",
		Circuitbreaker: "no-exists",
		Network:        "tcp",
		Timeout:        1000,
		Protocol:       "trpc",
		Filter:         []string{"Monitoring", "Authentication"},
	}
	err := client.RegisterClientConfig("trpc.test.nobreaker", backconfig)
	assert.NotNil(t, err)
}

func TestConfigNoFilter(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName: "trpc.test.helloworld3", // backend service name
		Namespace:   "Development",
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
		Filter:      []string{"Monitoring", "no-exists"},
	}
	err := client.RegisterClientConfig("trpc.test.nofilter", backconfig)
	assert.NotNil(t, err)
}

func TestConfigFilter(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName: "trpc.test.helloworld3", // backend service name
		Namespace:   "Development",
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
		Filter:      []string{"Monitoring"},
	}
	filter.Register("Monitoring", nil, filter.NoopClientFilter)
	err := client.RegisterClientConfig("trpc.test.filter", backconfig)
	assert.Nil(t, err)
}

func TestLoadClientFilterConfigSelectorFilter(t *testing.T) {
	const callee = "test_selector_filter"
	require.Nil(t, client.RegisterClientConfig(callee, &client.BackendConfig{
		Filter: []string{client.DefaultSelectorFilterName},
	}))
}

func TestRegisterConfigParallel(t *testing.T) {
	safeRand := rand.NewSafeRand(time.Now().UnixNano())
	for i := 0; i < safeRand.Intn(100); i++ {
		t.Run("Parallel", func(t *testing.T) {
			t.Parallel()
			backconfig := &client.BackendConfig{
				ServiceName: "trpc.test.helloworld1", // backend service name
				Target:      "ip://1.1.1.1:2222",     // backend address
				Network:     "tcp",
				Timeout:     1000,
				Protocol:    "trpc",
			}
			conf := map[string]*client.BackendConfig{
				"trpc.test.helloworld": backconfig,
			}
			err := client.RegisterConfig(conf)
			assert.Nil(t, err)
			assert.Equal(t, client.DefaultClientConfig(), conf)
		})
	}
}

func TestLoadClientConfig(t *testing.T) {
	err := client.LoadClientConfig("../testdata/trpc_go.yaml")
	assert.Nil(t, err)
}

type testTransport struct{}

func (t *testTransport) RoundTrip(ctx context.Context, req []byte,
	opts ...transport.RoundTripOption) ([]byte, error) {
	return nil, nil
}

func TestConfigTransport(t *testing.T) {
	t.Run("Client Config", func(t *testing.T) {
		tr := &testTransport{}
		transport.RegisterClientTransport("test-transport", tr)
		var cfg client.BackendConfig
		require.Nil(t, yaml.Unmarshal([]byte(`
transport: test-transport
`), &cfg))
		require.Equal(t, "test-transport", cfg.Transport)
		require.Nil(t, client.RegisterClientConfig("trpc.test.hello", &cfg))
	})
}

func TestConfigStreamFilter(t *testing.T) {
	filterName := "sf1"
	cfg := &client.BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
stream_filter:
- sf1
`), cfg))
	require.Equal(t, filterName, cfg.StreamFilter[0])
	// return error if stream filter no registered
	err := client.RegisterClientConfig("trpc.test.hello", cfg)
	assert.NotNil(t, err)

	client.RegisterStreamFilter("sf1", func(ctx context.Context, desc *client.ClientStreamDesc,
		streamer client.Streamer) (client.ClientStream, error) {
		return nil, nil
	})
	require.Nil(t, client.RegisterClientConfig("trpc.test.hello", cfg))
}

func TestConfig(t *testing.T) {
	require.Nil(t, client.RegisterConfig(make(map[string]*client.BackendConfig)))
	c := client.Config("empty")
	assert.Equal(t, "", c.ServiceName)
	assert.Equal(t, "tcp", c.Network)
	assert.Equal(t, "trpc", c.Protocol)

	backconfig := &client.BackendConfig{
		ServiceName: "trpc.test.helloworld1", // backend service name
		Target:      "ip://1.1.1.1:2222",     // backend address
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
	}
	conf := map[string]*client.BackendConfig{
		"trpc.test.helloworld": backconfig,
	}

	err := client.RegisterConfig(conf)
	assert.Nil(t, err)
	assert.Equal(t, client.DefaultClientConfig(), conf)

	require.Nil(t, client.RegisterClientConfig("trpc.test.helloworld2", backconfig))
	assert.Equal(t, "trpc.test.helloworld1", client.Config("trpc.test.helloworld2").ServiceName)

	c = client.Config("no-exist")
	assert.Equal(t, "", c.ServiceName)
	assert.Equal(t, "tcp", c.Network)
	assert.Equal(t, "trpc", c.Protocol)

	c = client.Config("trpc.test.helloworld")
	assert.Equal(t, "trpc.test.helloworld1", c.ServiceName)
	assert.Equal(t, "tcp", c.Network)
	assert.Equal(t, "ip://1.1.1.1:2222", c.Target)
	assert.Equal(t, 1000, c.Timeout)
	assert.Equal(t, "trpc", c.Protocol)

	backconfig = &client.BackendConfig{
		ServiceName: "trpc.test.helloworld1", // backend service name
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
		Compression: 1,
		Password:    "xxx",
		CACert:      "xxx",
	}
	require.Nil(t, client.RegisterClientConfig("trpc.test.helloworld3", backconfig))
}
