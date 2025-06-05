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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/internal/protocol"
	"trpc.group/trpc-go/trpc-go/internal/random"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/overloadctrl"
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

	filter.Register("tjg", filter.NoopServerFilter, filter.NoopClientFilter)
	filter.Register("m007", filter.NoopServerFilter, filter.NoopClientFilter)
	backconfig := &client.BackendConfig{
		ServiceName: "trpc.test.helloworld3", // backend service name
		Namespace:   "Development",
		Target:      "cmlb://1111",
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
		Filter:      []string{"tjg", "m007"},
	}
	err := client.RegisterClientConfig("trpc.test.helloworld3", backconfig)
	assert.NotNil(t, err)
	clientOpts = &client.Options{}
	transportOpts = &transport.RoundTripOptions{}
	require.Nil(t, clientOpts.LoadClientConfig("trpc.test.helloworld3"))
	for _, o := range clientOpts.CallOptions {
		o(transportOpts)
	}
	assert.Equal(t, "tcp", transportOpts.Network)
	assert.Equal(t, trpc.DefaultClientCodec, clientOpts.Codec)
}

func TestConfigNoDiscovery(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName: "trpc.test.helloworld3", // backend service name
		Namespace:   "Development",
		Discovery:   "no-exists",
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
		Filter:      []string{"tjg", "m007"},
	}
	err := client.RegisterClientConfig("trpc.test.nodiscovery", backconfig)
	assert.NotNil(t, err)
	clientOpts := &client.Options{}
	require.Nil(t, clientOpts.LoadClientConfig("trpc.test.nodiscovery"))
}

func TestConfigNoServiceRouter(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName:   "trpc.test.helloworld3", // backend service name
		Namespace:     "Development",
		ServiceRouter: "no-exists",
		Network:       "tcp",
		Timeout:       1000,
		Protocol:      "trpc",
		Filter:        []string{"tjg", "m007"},
	}
	err := client.RegisterClientConfig("trpc.test.noservicerouter", backconfig)
	assert.NotNil(t, err)
	clientOpts := &client.Options{}
	require.Nil(t, clientOpts.LoadClientConfig("trpc.test.noservicerouter"))
}

func TestConfigNoBalance(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName: "trpc.test.helloworld3", // backend service name
		Namespace:   "Development",
		Loadbalance: "no-exists",
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
		Filter:      []string{"tjg", "m007"},
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
	require.Error(t, err)
	require.Contains(t, err.Error(), testOptionsSelectorError.Error())
}

func TestConfigCallerMetadata(t *testing.T) {
	ctx := context.Background()
	req := &codec.Body{}
	rsp := &codec.Body{}
	callerMetadata := map[string]string{
		"key1": "val1",
		"key2": "val2",
	}
	callee := "trpc." + t.Name()
	backconfig := &client.BackendConfig{
		Namespace:      "Development",
		Network:        "tcp",
		Timeout:        1000,
		Protocol:       "trpc",
		CallerMetadata: callerMetadata,
	}
	require.Nil(t, client.RegisterClientConfig(callee, backconfig))
	s := &testOptionsSelector{
		f: func(opts *selector.Options) {
			assert.Equal(t, callerMetadata, opts.SourceMetadata)
		},
	}
	selectorName := "test-options-selector"
	selector.Register(selectorName, s)
	cli := client.New()
	assert.Equal(t, cli, client.DefaultClient)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithCalleeServiceName(callee)
	err := cli.Invoke(ctx, req, rsp,
		client.WithTarget(fmt.Sprintf("%s://trpc.test.client.metadata", selectorName)),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), testOptionsSelectorError.Error())
}

func TestConfigCallerNamespaceEnvSet(t *testing.T) {
	ctx := context.Background()
	req := &codec.Body{}
	rsp := &codec.Body{}
	callerNamespace, callerEnvName, callerSetName := "caller/Development", "caller_env", "caller_set"
	callee := "trpc." + t.Name()
	backconfig := &client.BackendConfig{
		Namespace:       "Development",
		Network:         "tcp",
		Timeout:         1000,
		Protocol:        "trpc",
		CallerNamespace: callerNamespace,
		CallerEnvName:   callerEnvName,
		CallerSetName:   callerSetName,
	}
	require.Nil(t, client.RegisterClientConfig(callee, backconfig))
	s := &testOptionsSelector{
		f: func(opts *selector.Options) {
			assert.Equal(t, callerNamespace, opts.SourceNamespace)
			assert.Equal(t, callerEnvName, opts.SourceEnvName)
			assert.Equal(t, callerSetName, opts.SourceSetName)
		},
	}
	selectorName := "test-options-selector"
	selector.Register(selectorName, s)
	cli := client.New()
	assert.Equal(t, cli, client.DefaultClient)
	ctx, msg := codec.WithNewMessage(ctx)
	msg.WithCalleeServiceName(callee)
	err := cli.Invoke(ctx, req, rsp,
		client.WithTarget(fmt.Sprintf("%s://trpc.test.client.caller", selectorName)),
	)
	require.Error(t, err)
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
		Filter:         []string{"tjg", "m007"},
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
		Filter:      []string{"tjg", "no-exists"},
	}
	err := client.RegisterClientConfig("trpc.test.nofilter", backconfig)
	assert.NotNil(t, err)
	clientOpts := &client.Options{}
	require.Nil(t, clientOpts.LoadClientFilterConfig("trpc.test.nofilter"))
}
func TestConfigDisableFilter(t *testing.T) {
	clientOpts := &client.Options{}
	clientOpts.DisableFilter = true
	require.Nil(t, clientOpts.LoadClientFilterConfig("trpc.test.disablefilter"))
}
func TestConfigFilter(t *testing.T) {
	backconfig := &client.BackendConfig{
		ServiceName: "trpc.test.helloworld3", // backend service name
		Namespace:   "Development",
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
		Filter:      []string{"tjg"},
	}
	filter.Register("tjg", nil, filter.NoopFilter)
	err := client.RegisterClientConfig("trpc.test.filter", backconfig)
	assert.Nil(t, err)
	clientOpts := &client.Options{}
	require.Nil(t, clientOpts.LoadClientFilterConfig("trpc.test.filter"))
}

func TestLoadClientFilterConfigSelectorFilter(t *testing.T) {
	const callee = "test_selector_filter"
	require.Nil(t, client.RegisterClientConfig(callee, &client.BackendConfig{
		Filter: []string{client.DefaultSelectorFilterName},
	}))
	require.Nil(t, (&client.Options{}).LoadClientFilterConfig(callee))
}

func TestLoadClientFilterConfigSelectorFilterRepair(t *testing.T) {
	const callee = "trpc.test.filter.selector"
	backconfig := &client.BackendConfig{
		ServiceName: callee,
		Filter:      []string{client.DefaultSelectorFilterName},
	}
	require.Nil(t, client.RegisterClientConfig(callee, backconfig))

	clientOpts := &client.Options{}
	require.Nil(t, clientOpts.LoadClientFilterConfig(callee))
	require.Equal(t, []string{client.DefaultSelectorFilterName}, clientOpts.FilterNames)
}

func TestRegisterConfigParallel(t *testing.T) {
	safeRand := random.New()
	for i := 0; i < safeRand.Intn(100); i++ {
		t.Run("Parallel", func(t *testing.T) {
			t.Parallel()
			backconfig := &client.BackendConfig{
				ServiceName: "trpc.test.helloworld1", // backend service name
				Target:      "ip://1.1.1.1:2222",     // backend address
				Tag:         "tag1",
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

func TestLoadClientOverloadCtrlCfg(t *testing.T) {
	testClientOC := &overloadctrl.NoopOC{}
	overloadctrl.RegisterClient("test_client_oc",
		func(*overloadctrl.ServiceMethodInfo) overloadctrl.OverloadController {
			return testClientOC
		})

	t.Run("default oc", func(t *testing.T) {
		var cfg client.BackendConfig
		require.Nil(t, yaml.Unmarshal([]byte(`
name: xxx
`), &cfg))
		token, err := cfg.OverloadCtrl.Acquire(context.Background(), "")
		require.Nil(t, err)
		require.Equal(t, overloadctrl.NoopToken{}, token)
	})
	t.Run("wrong format", func(t *testing.T) {
		var cfg client.BackendConfig
		require.NotNil(t, yaml.Unmarshal([]byte(`
overload_ctrl: [1, 2, 3] # invalid format
`), &cfg))
	})
	t.Run("oc not found", func(t *testing.T) {
		var cfg client.BackendConfig
		require.NotNil(t, yaml.Unmarshal([]byte(`
overload_ctrl: not_exist
`), &cfg))
	})
	t.Run("oc found", func(t *testing.T) {
		var cfg client.BackendConfig
		require.Nil(t, yaml.Unmarshal([]byte(`
overload_ctrl: "test_client_oc"
`), &cfg))
		require.Equal(t, testClientOC, cfg.OverloadCtrl.OverloadController)
	})
	t.Run("marshal_unmarshal", func(t *testing.T) {
		ocData := "overload_ctrl: test_client_oc"
		var cfg client.BackendConfig
		require.Nil(t, yaml.Unmarshal([]byte(ocData), &cfg))
		data, err := yaml.Marshal(&cfg)
		require.Nil(t, err)
		require.Contains(t, string(data), ocData)
	})
}

func TestConfig(t *testing.T) {
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
	clientOpts := &client.Options{}
	transportOpts := &transport.RoundTripOptions{}
	require.Nil(t, clientOpts.LoadClientConfig("trpc.test.helloworld3"))
	for _, o := range clientOpts.CallOptions {
		o(transportOpts)
	}
	assert.Equal(t, "tcp", transportOpts.Network)
	assert.Equal(t, trpc.DefaultClientCodec, clientOpts.Codec)
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

func TestReportAnyErrToSelector(t *testing.T) {
	backconfig := &client.BackendConfig{
		ReportAnyErrToSelector: true,
	}
	require.Nil(t, client.RegisterClientConfig("trpc.test.helloworld3", backconfig))
	clientOpts := &client.Options{}
	require.Nil(t, clientOpts.LoadClientConfig("trpc.test.helloworld3"))
}

func TestMethodTimeoutCfg(t *testing.T) {
	backendConfig := client.BackendConfig{}
	require.Nil(t, yaml.Unmarshal([]byte(`
method:
  M0:
    timeout: 1000
  M1: {}
`), &backendConfig))
	require.Len(t, backendConfig.Method, 2)
	m0, ok := backendConfig.Method["M0"]
	require.True(t, ok)
	require.NotNil(t, m0.Timeout)
	require.Equal(t, 1000, *m0.Timeout)
	m1, ok := backendConfig.Method["M1"]
	require.True(t, ok)
	require.Nil(t, m1.Timeout)
}

func TestRegisterWildcardClient(t *testing.T) {
	cfg := client.Config("*")
	t.Cleanup(func() {
		client.RegisterClientConfig("*", cfg)
	})
	client.RegisterClientConfig("*", &client.BackendConfig{
		DisableServiceRouter: true,
	})

	ch := make(chan *client.Options, 1)
	c := client.New()
	ctx, _ := codec.EnsureMessage(context.Background())
	require.Nil(t, c.Invoke(ctx, nil, nil, client.WithFilter(
		func(ctx context.Context, _, _ interface{}, _ filter.ClientHandleFunc) error {
			ch <- client.OptionsFromContext(ctx)
			// Skip next.
			return nil
		})))
	opts := <-ch
	require.True(t, opts.DisableServiceRouter)
}

func TestGetConfig(t *testing.T) {
	client.RegisterConfig(nil) // clean up
	_, err := client.GetConfig(t.Name(), "")
	require.Error(t, err)

	cfg1 := &client.BackendConfig{
		Callee:      t.Name(),
		ServiceName: t.Name(),            // backend service name
		Target:      "ip://1.1.1.1:1111", // backend address
		Network:     "tcp",
		Timeout:     1000,
		Protocol:    "trpc",
	}
	cfg2 := &client.BackendConfig{
		Callee:      t.Name(),
		ServiceName: t.Name() + "/1",     // backend service name
		Target:      "ip://1.1.1.1:2222", // backend address
		Network:     "tcp",
		Timeout:     1200,
		Protocol:    "trpc",
	}
	client.RegisterClientConfig(cfg1.Callee, cfg1)

	cfg, err := client.GetConfig(cfg1.Callee, cfg1.ServiceName)
	require.Nil(t, err)
	require.Equal(t, cfg1, cfg)
	cfg, err = client.GetConfig(cfg1.Callee, "")
	require.Nil(t, err)
	require.Equal(t, cfg1, cfg)
	cfg, err = client.GetConfig(cfg1.Callee, cfg2.ServiceName)
	require.Nil(t, err)
	require.Equal(t, cfg1, cfg)

	client.RegisterClientConfig(cfg2.Callee, cfg2)
	cfg, err = client.GetConfig(cfg1.Callee, cfg1.ServiceName)
	require.Nil(t, err)
	require.Equal(t, cfg1, cfg)
	cfg, err = client.GetConfig(cfg1.Callee, "")
	require.Nil(t, err)
	require.Equal(t, cfg1, cfg)
	cfg, err = client.GetConfig(cfg2.Callee, cfg2.ServiceName)
	require.Nil(t, err)
	require.Equal(t, cfg2, cfg)
	cfg, err = client.GetConfig(cfg2.Callee, "")
	require.Nil(t, err)
	require.Equal(t, cfg1, cfg)

	cfg, err = client.GetConfig(t.Name()+"not-exist", "")
	require.Error(t, err)
	require.Nil(t, cfg)

	cfg, err = client.GetConfig(cfg1.Callee, t.Name()+"not-exist")
	require.Nil(t, err)
	require.Equal(t, cfg2, cfg)
	cfg3 := &client.BackendConfig{
		Protocol: "trpc",
		Target:   "ip://1.1.1.1:3333",
	}
	client.RegisterClientConfig("*", cfg3)
	cfg, err = client.GetConfig(t.Name()+"not-exist", "")
	require.Nil(t, err)
	require.Equal(t, cfg3, cfg)
}

func TestGetOptionsByCalleeAndUserOptions(t *testing.T) {
	client.RegisterConfig(nil)
	defer client.RegisterConfig(nil)
	_, err := client.GetConfig(t.Name(), "")
	require.Error(t, err)

	cfg1 := &client.BackendConfig{
		Callee:      t.Name(),
		ServiceName: t.Name(), // backend service name
		Tag:         "tag1",
		Target:      "ip://1.1.1.1:1111", // backend address
		Network:     protocol.TCP,
		Timeout:     1000,
		Protocol:    protocol.TRPC,
	}
	client.RegisterClientConfig(cfg1.Callee, cfg1)

	ctx, msg := codec.EnsureMessage(context.Background())
	msg.WithCalleeServiceName(t.Name())
	err = client.DefaultClient.Invoke(ctx, nil, nil,
		client.WithServiceName(t.Name()),
		client.WithTag("tag1"),
	)
	require.NotContains(t, err.Error(), "please check for configuration errors")

	err = client.DefaultClient.Invoke(ctx, nil, nil,
		client.WithServiceName(t.Name()),
		client.WithTag("tag2"),
	)
	require.Contains(t, err.Error(), "please check for configuration errors")
}

func TestRegisterConnTypeForNonTRPCService(t *testing.T) {
	const protocol = "http"
	c, s := codec.GetClient(protocol), codec.GetServer(protocol)
	defer func() {
		codec.Register(protocol, s, c)
	}()
	codec.Register(protocol, &fakeCodec{}, &fakeCodec{})
	tests := []struct {
		name    string
		config  string
		success bool
	}{
		{
			name: "conn_type short",
			config: `
name: trpc.test.helloworld.Greeter1  # backend service name.
callee: trpc.test.helloworld.Greeter1  # proto name of the callee service defined in proto stub file.
protocol: http
conn_type: short
`,
			success: true,
		},
		{
			name: "conn_type connpool",
			config: `
name: trpc.test.helloworld.Greeter1  # backend service name.
callee: trpc.test.helloworld.Greeter1  # proto name of the callee service defined in proto stub file.
protocol: http
conn_type: connpool
`,
			success: false,
		},
		{
			name: "conn_type multiplexed",
			config: `
name: trpc.test.helloworld.Greeter1  # backend service name.
callee: trpc.test.helloworld.Greeter1  # proto name of the callee service defined in proto stub file.
protocol: http
conn_type: multiplexed
`,
			success: false,
		},
		{
			name: "conn_type httppool",
			config: `
name: trpc.test.helloworld.Greeter1  # backend service name.
callee: trpc.test.helloworld.Greeter1  # proto name of the callee service defined in proto stub file.
protocol: http
conn_type: httppool
`,
			success: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &client.BackendConfig{}
			require.Nil(t, yaml.Unmarshal([]byte(tt.config), cfg))
			err := client.RegisterClientConfig(t.Name(), cfg)
			require.Equal(t, tt.success, err == nil)
		})
	}
}
