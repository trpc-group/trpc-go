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

package trpc

import (
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/rpcz"
)

// TestLoadGlobalConfigNICError tests LoadGlobalConfig error.
func TestLoadGlobalConfigNICError(t *testing.T) {
	err := LoadGlobalConfig("testdata/trpc_go_error.yaml")
	assert.NotNil(t, err)
}

// TestRepairServiceIPWithNicAdminError tests repairServiceIPWithNic for admin error.
func TestRepairServiceIPWithNicAdminError(t *testing.T) {
	conf := &Config{}
	conf.Server.Admin.IP = ""
	conf.Server.Admin.Nic = "ethNoneExist"
	err := repairServiceIPWithNic(conf)
	assert.Contains(t, err.Error(), "can't find admin IP by the NIC")

}

func TestParseEnv(t *testing.T) {
	filePath := "./trpc_go.yaml"
	content := `
global:
  namespace: development
  env_name: ${test}
  container_name: ${container_name}
  local_ip: $local_ip
server:
  app: ${}
  server: ${server
client:
  timeout: ${client_timeout}  
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Errorf("write file err: %v", err)
		return
	}
	defer func() {
		_ = os.Remove(filePath)
	}()

	containerName := `这是容器名称，this is container name.`
	_ = os.Setenv("container_name", containerName)
	localIP := `127.1.2.3`
	_ = os.Setenv("local_ip", localIP)
	timeout := 456
	_ = os.Setenv("client_timeout", strconv.FormatInt(int64(timeout), 10))

	c, err := parseConfigFromFile(filePath)
	if err != nil {
		t.Errorf("parse err: %v", err)
		return
	}
	assert.Equal(t, c.Global.Namespace, "development")
	assert.Equal(t, c.Global.EnvName, "") // env name not set, should be replaced with empty value
	assert.Equal(t, c.Global.ContainerName, containerName)
	assert.Equal(t, c.Global.LocalIP, "$local_ip") // only ${var} instead of $var is valid
	assert.Equal(t, c.Server.App, "")              // empty ${} should be deleted
	assert.Equal(t, c.Server.Server, "${server")   // only ${var} is valid
	assert.Equal(t, c.Client.Timeout, timeout)
}

func TestParseEnvSpecialChar(t *testing.T) {
	filePath := "./trpc_go.yaml"
	content := `
global:
  namespace: development
  env_name: test
  container_name: ${container_name}
  local_ip: ${local_ip}
`
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Errorf("write file err: %v", err)
		return
	}
	defer func() {
		_ = os.Remove(filePath)
	}()

	containerName := `'这是"容器名称": this is container name. !@#$%^&*()_+-='`
	_ = os.Setenv("container_name", containerName)
	actualContainerName := strings.Trim(containerName, `'`) // single quotes
	localIP := `"'127.1.2.3'"`
	_ = os.Setenv("local_ip", localIP)
	actualLocalIP := strings.Trim(localIP, `"`) // double quotes

	c, err := parseConfigFromFile(filePath)
	if err != nil {
		t.Errorf("parse err: %v", err)
		return
	}
	assert.Equal(t, c.Global.Namespace, "development")
	assert.Equal(t, c.Global.EnvName, "test")
	assert.Equal(t, c.Global.ContainerName, actualContainerName)
	assert.Equal(t, c.Global.LocalIP, actualLocalIP)
}

func Test_serverConfigPath(t *testing.T) {
	oldServerConfigPath := ServerConfigPath
	ServerConfigPath = "./testdata/trpc_go.yaml"
	path := serverConfigPath()
	assert.Equal(t, "./testdata/trpc_go.yaml", path)

	ServerConfigPath = "./trpc_go.yaml"
	path = serverConfigPath()
	assert.Equal(t, defaultConfigPath, path)

	ServerConfigPath = oldServerConfigPath
}

func Test_setDefault(t *testing.T) {
	var (
		dstEmpty    string
		dstNotEmpty = "not-empty"
		def         = "default"
	)
	setDefault(&dstEmpty, def)
	setDefault(&dstNotEmpty, def)
	assert.Equal(t, dstEmpty, def)
	assert.NotEqual(t, dstNotEmpty, def)
}

func TestConfigTransport(t *testing.T) {
	t.Run("Server Config", func(t *testing.T) {
		var cfg Config
		require.Nil(t, yaml.Unmarshal([]byte(`
server: 
  transport: test-transport
`), &cfg))
		require.Equal(t, "test-transport", cfg.Server.Transport)
	})
	t.Run("Service Config", func(t *testing.T) {
		var cfg ServiceConfig
		require.Nil(t, yaml.Unmarshal([]byte(`
transport: test-transport
`), &cfg))
		require.Equal(t, "test-transport", cfg.Transport)
	})
	t.Run("Client Config", func(t *testing.T) {
		var cfg ClientConfig
		require.Nil(t, yaml.Unmarshal([]byte(`
transport: test-transport
`), &cfg))
		require.Equal(t, "test-transport", cfg.Transport)
	})
}
func TestConfigStreamFilter(t *testing.T) {
	filterName := "sf1"
	t.Run("server config", func(t *testing.T) {
		var cfg Config
		require.Nil(t, yaml.Unmarshal([]byte(`
server:
  stream_filter:
    - sf1
`), &cfg))
		require.Equal(t, filterName, cfg.Server.StreamFilter[0])
	})
	t.Run("service config", func(t *testing.T) {
		var cfg ServiceConfig
		require.Nil(t, yaml.Unmarshal([]byte(`
stream_filter:
  - sf1
`), &cfg))
		require.Equal(t, filterName, cfg.StreamFilter[0])
	})
	t.Run("client config", func(t *testing.T) {
		var cfg ClientConfig
		require.Nil(t, yaml.Unmarshal([]byte(`
stream_filter:
  - sf1
`), &cfg))
		require.Equal(t, filterName, cfg.StreamFilter[0])
	})

}

func TestRecordWhen(t *testing.T) {
	t.Run("empty record-when", func(t *testing.T) {
		config := &RPCZConfig{}
		require.Nil(t, yaml.Unmarshal(
			[]byte(`
fraction: 1.0
capacity: 10`),
			config,
		))
		r := rpcz.NewRPCZ(config.generate())
		var ids []rpcz.SpanID
		for i := 0; i < 10; i++ {
			s, ender := r.NewChild("")
			ids = append(ids, s.ID())
			ender.End()
		}
		for _, id := range ids {
			_, ok := r.Query(id)
			require.True(t, ok)
		}
	})
	t.Run("unknown node", func(t *testing.T) {
		config := &RecordWhenConfig{}
		err := yaml.Unmarshal(
			[]byte(`
- XOR:
    - __min_request_size: 30
    - __min_response_size: 40
`),
			config,
		)
		require.Contains(t, errs.Msg(err), "unknown node: XOR")
	})
	t.Run("AND node is map type", func(t *testing.T) {
		config := &RecordWhenConfig{}
		err := yaml.Unmarshal(
			[]byte(`
- AND: {__rpc_name: "/trpc.app.server.service/method"}
`),
			config,
		)
		require.Contains(t, errs.Msg(err), "cannot unmarshal !!map into []map[trpc.nodeKind]yaml.Node")
	})
	t.Run("OR node is map type", func(t *testing.T) {
		config := &RecordWhenConfig{}
		err := yaml.Unmarshal(
			[]byte(`
- OR: {__rpc_name: "/trpc.app.server.service/method"}
`),
			config,
		)
		require.Contains(t, errs.Msg(err), "cannot unmarshal !!map into []map[trpc.nodeKind]yaml.Node")
	})

}
func TestRecordWhen_NotNode(t *testing.T) {
	t.Run("NOT node is empty", func(t *testing.T) {
		config := &RPCZConfig{}
		err := yaml.Unmarshal([]byte(`
record_when:
  - NOT:
`),
			config,
		)
		require.ErrorContains(t, err, "value is empty")
	})
	t.Run("NOT node has two children", func(t *testing.T) {
		config := &RecordWhenConfig{}
		err := yaml.Unmarshal(
			[]byte(`
    - NOT: {__rpc_name: "/trpc.app.server.service/method", __min_duration: 1000ms}
    `),
			config,
		)
		require.Contains(t, errs.Msg(err), "the valid number of child node can only be 1")
	})
	t.Run("NOT has a leaf child", func(t *testing.T) {
		config := &RecordWhenConfig{}
		require.Nil(t, yaml.Unmarshal(
			[]byte(`
    - NOT:
        __rpc_name: "/trpc.app.server.service/method"
    `),
			config,
		))
	})
	t.Run("NOT has a internal child", func(t *testing.T) {
		config := &RecordWhenConfig{}
		require.Nil(t, yaml.Unmarshal(
			[]byte(`
- NOT:
    OR:
      - __min_duration: 1000ms
      - __rpc_name: "/trpc.app.server.service/method"
`),
			config,
		))
	})
	t.Run("NOT node is slice type", func(t *testing.T) {
		config := &RecordWhenConfig{}
		err := yaml.Unmarshal(
			[]byte(`
- NOT: 
    - __rpc_name: "/trpc.app.server.service/method"
`),
			config,
		)
		require.Contains(t, errs.Msg(err), "cannot unmarshal !!seq into map[trpc.nodeKind]yaml.Node")
	})
}
func TestRecordWhen_ANDNode(t *testing.T) {
	t.Run("AND node is empty", func(t *testing.T) {
		config := &RPCZConfig{}
		err := yaml.Unmarshal([]byte(`
record_when:
  - AND:
`),
			config,
		)
		require.ErrorContains(t, err, "value is empty")
	})
	t.Run("AND node has two children", func(t *testing.T) {
		config := &RecordWhenConfig{}
		require.Nil(t, yaml.Unmarshal(
			[]byte(`
- AND: 
    - __rpc_name: "/trpc.app.server.service/method" 
    - __min_duration: 1000ms
`),
			config,
		))
	})
	t.Run("AND has a leaf child", func(t *testing.T) {
		config := &RecordWhenConfig{}
		require.Nil(t, yaml.Unmarshal(
			[]byte(`
- AND:
    - __rpc_name: "/trpc.app.server.service/method"
`),
			config,
		))
	})
	t.Run("AND has a internal child", func(t *testing.T) {
		config := &RecordWhenConfig{}
		require.Nil(t, yaml.Unmarshal(
			[]byte(`
- AND:
    - OR:
        - __min_duration: 1000ms
        - __rpc_name: "/trpc.app.server.service/method"
`),
			config,
		))
	})
	t.Run("AND node is map type", func(t *testing.T) {
		config := &RecordWhenConfig{}
		err := yaml.Unmarshal(
			[]byte(`
- AND: 
    __rpc_name: "/trpc.app.server.service/method"
`),
			config,
		)
		require.Contains(t, errs.Msg(err), "cannot unmarshal !!map into []map[trpc.nodeKind]yaml.Node")
	})
}
func TestRPCZ_RecordWhen_ErrorCode(t *testing.T) {
	config := &RPCZConfig{}
	mustYamlUnmarshal(t, []byte(`
fraction: 1.0
capacity: 10
record_when:
  - __sampling_fraction: 1
  - OR:
      - __error_code: 1  # RetServerDecodeFail = 1
      - __error_code: 2  # RetServerEncodeFail = 2
      - __error_message: "service codec"
      - __error_message: "client codec"
  - NOT:
      OR:
        - __error_code: 1
        - __error_message: "service codec"
`), config)

	r := rpcz.NewRPCZ(config.generate())
	var (
		expectedIDs   []rpcz.SpanID
		unexpectedIDs []rpcz.SpanID
	)
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetServerDecodeFail, "service codec"))
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetServerEncodeFail, "service codec"))
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetServerDecodeFail, "client codec"))
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetServerEncodeFail, "client codec"))
		expectedIDs = append(expectedIDs, s.ID())
		ender.End()
	}
	for i, id := range expectedIDs {
		_, ok := r.Query(id)
		require.True(t, ok, i)
	}
	for i, id := range unexpectedIDs {
		_, ok := r.Query(id)
		require.False(t, ok, i)
	}
}
func TestRPC_RecordWhen_CustomAttribute(t *testing.T) {
	config := &RPCZConfig{}
	mustYamlUnmarshal(t, []byte(`
fraction: 1.0
capacity: 10
record_when:
  - __sampling_fraction: 1
  - OR:
      - __has_attribute: (race, elf)
      - __has_attribute: (class, wizard)
  - NOT:
      OR:
        - __has_attribute: (race, dwarf)
        - __has_attribute: (class, warlock)
`), config)

	r := rpcz.NewRPCZ(config.generate())
	var (
		expectedIDs   []rpcz.SpanID
		unexpectedIDs []rpcz.SpanID
	)
	{
		s, ender := r.NewChild("")
		s.SetAttribute("race", "elf")
		s.SetAttribute("class", "wizard")
		expectedIDs = append(expectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute("race", "elf")
		s.SetAttribute("class", "wizard, warlock")
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute("race", "elf, dwarf")
		s.SetAttribute("class", "wizard")
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute("race", "elf, dwarf")
		s.SetAttribute("class", "wizard, warlock")
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	for i, id := range expectedIDs {
		_, ok := r.Query(id)
		require.True(t, ok, i)
	}
	for i, id := range unexpectedIDs {
		_, ok := r.Query(id)
		require.False(t, ok, i)
	}
}
func TestRPC_RecordWhen_InvalidCustomAttribute(t *testing.T) {
	t.Run("miss left parenthesis", func(t *testing.T) {
		config := &RPCZConfig{}
		require.ErrorContains(t, yaml.Unmarshal([]byte(`
record_when:
  - __has_attribute: race, elf)
`), config), "invalid attribute form")
	})
	t.Run("miss right parenthesis", func(t *testing.T) {
		config := &RPCZConfig{}
		require.ErrorContains(t, yaml.Unmarshal([]byte(`
record_when:
  - __has_attribute: (race, elf
`), config), "invalid attribute form")
	})
	t.Run("middle delimiter space", func(t *testing.T) {
		config := &RPCZConfig{}
		require.ErrorContains(t, yaml.Unmarshal([]byte(`
record_when:
  - __has_attribute: (race,elf)
`), config), "invalid attribute form")
	})
	t.Run("middle delimiter comma", func(t *testing.T) {
		config := &RPCZConfig{}
		require.ErrorContains(t, yaml.Unmarshal([]byte(`
record_when:
  - __has_attribute: (race elf)
`), config), "invalid attribute form")
	})
}
func TestRPCZ_RecordWhen_MinDuration(t *testing.T) {
	t.Run("not empty", func(t *testing.T) {
		config := &RPCZConfig{}
		mustYamlUnmarshal(t, []byte(`
fraction: 1.0
capacity: 10
record_when:
  - __error_code: 999 # RetUnknown = 0
  - __min_duration: 100ms
  - __sampling_fraction: 1
`), config)

		r := rpcz.NewRPCZ(config.generate())
		var (
			expectedIDs   []rpcz.SpanID
			unexpectedIDs []rpcz.SpanID
		)
		{
			s, ender := r.NewChild("")
			s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetUnknown, ""))
			// mimic some time-consuming operation.
			time.Sleep(1 * time.Second)
			expectedIDs = append(expectedIDs, s.ID())
			ender.End()
		}
		{
			s, ender := r.NewChild("")
			s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetUnknown, ""))
			// mimic some time-consuming operation.
			time.Sleep(2 * time.Second)
			expectedIDs = append(expectedIDs, s.ID())
			ender.End()
		}
		{
			// don't call ender.End()
			s, _ := r.NewChild("")
			s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
			unexpectedIDs = append(unexpectedIDs, s.ID())
		}
		{
			s, ender := r.NewChild("")
			s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetUnknown, ""))
			// mimic some time-consuming operation.
			time.Sleep(1 * time.Millisecond)
			unexpectedIDs = append(unexpectedIDs, s.ID())
			ender.End()
		}
		{
			s, ender := r.NewChild("")
			s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetUnknown, ""))
			// mimic some time-consuming operation.
			time.Sleep(2 * time.Millisecond)
			unexpectedIDs = append(unexpectedIDs, s.ID())
			ender.End()
		}
		for _, id := range expectedIDs {
			_, ok := r.Query(id)
			require.True(t, ok)
		}
		for _, id := range unexpectedIDs {
			_, ok := r.Query(id)
			require.False(t, ok)
		}
	})
	t.Run("empty", func(t *testing.T) {
		config := &RPCZConfig{}
		mustYamlUnmarshal(t, []byte(`
fraction: 1.0
capacity: 10
record_when:
  - __error_code: 0 # RetOK = 0
  - __sampling_fraction: 1
`), config)

		r := rpcz.NewRPCZ(config.generate())
		var (
			unexpectedID rpcz.SpanID
			expectedID   rpcz.SpanID
		)
		{
			s, ender := r.NewChild("")
			s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetUnknown, ""))
			// mimic some time-consuming operation.
			time.Sleep(1 * time.Millisecond)
			unexpectedID = s.ID()
			ender.End()
		}
		{
			s, ender := r.NewChild("")
			s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
			// mimic some time-consuming operation.
			time.Sleep(1 * time.Millisecond)
			expectedID = s.ID()
			ender.End()
		}
		_, ok := r.Query(unexpectedID)
		require.False(t, ok)

		_, ok = r.Query(expectedID)
		require.True(t, ok)
	})
}
func TestRPCZ_RecordWhen_MinRequestSize(t *testing.T) {
	config := &RPCZConfig{}
	mustYamlUnmarshal(t, []byte(`
fraction: 1.0
capacity: 10
record_when:
  - __sampling_fraction: 1
  - __min_request_size: 30
`), config)

	r := rpcz.NewRPCZ(config.generate())
	t.Run("unset request size", func(t *testing.T) {
		s, ender := r.NewChild("")
		unexpectedID := s.ID()
		ender.End()
		_, ok := r.Query(unexpectedID)
		require.False(t, ok)
	})
	t.Run("request size less than min_request_size", func(t *testing.T) {
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeRequestSize, 29)
		unexpectedID := s.ID()
		ender.End()
		_, ok := r.Query(unexpectedID)
		require.False(t, ok)
	})
	t.Run("request size equals to min_request_size", func(t *testing.T) {
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeRequestSize, 30)
		expectedID := s.ID()
		ender.End()
		_, ok := r.Query(expectedID)
		require.True(t, ok)
	})
	t.Run("request size greater than min_request_size", func(t *testing.T) {
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeRequestSize, 31)
		expectedID := s.ID()
		ender.End()
		_, ok := r.Query(expectedID)
		require.True(t, ok)
	})
}
func TestRPCZ_RecordWhen_MinResponseSize(t *testing.T) {
	config := &RPCZConfig{}
	mustYamlUnmarshal(t, []byte(`
fraction: 1.0
capacity: 10
record_when:
  - __sampling_fraction: 1
  - __min_response_size: 40
`), config)

	r := rpcz.NewRPCZ(config.generate())
	t.Run("unset response size", func(t *testing.T) {
		s, ender := r.NewChild("")
		unexpectedID := s.ID()
		ender.End()
		_, ok := r.Query(unexpectedID)
		require.False(t, ok)
	})
	t.Run("request size less than min_response_size", func(t *testing.T) {
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeResponseSize, 39)
		unexpectedID := s.ID()
		ender.End()
		_, ok := r.Query(unexpectedID)
		require.False(t, ok)
	})
	t.Run("request size equals to min_response_size", func(t *testing.T) {
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeResponseSize, 40)
		expectedID := s.ID()
		ender.End()
		_, ok := r.Query(expectedID)
		require.True(t, ok)
	})
	t.Run("request size greater than min_response_size", func(t *testing.T) {
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeResponseSize, 41)
		expectedID := s.ID()
		ender.End()
		_, ok := r.Query(expectedID)
		require.True(t, ok)
	})
}
func TestRPCZ_RecordWhen_RPCName(t *testing.T) {
	config := &RPCZConfig{}
	mustYamlUnmarshal(t, []byte(`
fraction: 1.0
capacity: 10
record_when:
  - __sampling_fraction: 1
  - __rpc_name: trpc.app.server.service
`), config)

	r := rpcz.NewRPCZ(config.generate())
	t.Run("unset RPCName", func(t *testing.T) {
		s, ender := r.NewChild("")
		unexpectedID := s.ID()
		ender.End()
		_, ok := r.Query(unexpectedID)
		require.False(t, ok)
	})
	t.Run("RPCName does not contain rpc_name", func(t *testing.T) {
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeRPCName, "/xxx.app.server.service/method")
		unexpectedID := s.ID()
		ender.End()
		_, ok := r.Query(unexpectedID)
		require.False(t, ok)
	})
	t.Run("RPCName contains  rpc_name", func(t *testing.T) {
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeRPCName, "/trpc.app.server.service/method")
		expectedID := s.ID()
		ender.End()
		_, ok := r.Query(expectedID)
		require.True(t, ok)
	})
}
func TestRPCZ_RecordWhen_ErrorCodeAndMinDuration(t *testing.T) {
	config := &RPCZConfig{}
	mustYamlUnmarshal(t, []byte(`
fraction: 1.0
capacity: 10
record_when:
  - AND:
      - __error_code: 0 # RetOK = 0
      - __min_duration: 100ms
  - __sampling_fraction: 1
`), config)

	r := rpcz.NewRPCZ(config.generate())
	var (
		expectedIDs   []rpcz.SpanID
		unexpectedIDs []rpcz.SpanID
	)
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
		// mimic some time-consuming operation.
		time.Sleep(1 * time.Second)
		expectedIDs = append(expectedIDs, s.ID())
		ender.End()
	}

	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetUnknown, ""))
		// mimic some time-consuming operation.
		time.Sleep(2 * time.Second)
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
		// mimic some time-consuming operation.
		time.Sleep(1 * time.Millisecond)
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetUnknown, ""))
		// mimic some time-consuming operation.
		time.Sleep(2 * time.Millisecond)
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		// don't call ender.End()
		s, _ := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
		unexpectedIDs = append(unexpectedIDs, s.ID())
	}
	for i, id := range expectedIDs {
		_, ok := r.Query(id)
		require.True(t, ok, i)
	}
	for i, id := range unexpectedIDs {
		_, ok := r.Query(id)
		require.False(t, ok, i)
	}
}

func mustYamlUnmarshal(t *testing.T, in []byte, out interface{}) {
	t.Helper()
	if err := yaml.Unmarshal(in, out); err != nil {
		t.Fatal(err)
	}
}
func TestRepairServiceIdleTime(t *testing.T) {
	t.Run("set by service timeout", func(t *testing.T) {
		var cfg Config
		require.Nil(t, yaml.Unmarshal([]byte(`
server: 
  service:  
    - name: trpc.test.helloworld.Greeter
      ip: 127.0.0.1
      port: 8000
      network: tcp
      protocol: trpc
      timeout: 120000
`), &cfg))
		require.Nil(t, RepairConfig(&cfg))
		require.Equal(t, 120000, cfg.Server.Service[0].Idletime)
	})
	t.Run("set by default", func(t *testing.T) {
		var cfg Config
		require.Nil(t, yaml.Unmarshal([]byte(`
server: 
  service:  
    - name: trpc.test.helloworld.Greeter
      ip: 127.0.0.1
      port: 8000
      network: tcp
      protocol: trpc
      timeout: 500
`), &cfg))
		require.Nil(t, RepairConfig(&cfg))
		require.Equal(t, defaultIdleTimeout, cfg.Server.Service[0].Idletime)
	})
	t.Run("set by config", func(t *testing.T) {
		var cfg Config
		require.Nil(t, yaml.Unmarshal([]byte(`
server: 
  service:  
    - name: trpc.test.helloworld.Greeter
      ip: 127.0.0.1
      port: 8000
      network: tcp
      protocol: trpc
      timeout: 500
      idletime: 1500
`), &cfg))
		require.Nil(t, RepairConfig(&cfg))
		require.Equal(t, 1500, cfg.Server.Service[0].Idletime)
	})
}
