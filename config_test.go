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
	t.Run("empty MinDuration", func(t *testing.T) {
		var rw RecordWhenConfig
		require.Nil(t, yaml.Unmarshal([]byte(`min_duration:`), &rw))
		require.Nil(t, rw.MinDuration)
		bts, err := yaml.Marshal(rw)
		require.Nil(t, err)
		require.Equal(t, "min_duration: null\nsampling_fraction: 0\n", string(bts))
	})
	t.Run("MinDuration", func(t *testing.T) {
		var rw RecordWhenConfig
		require.Nil(t, yaml.Unmarshal([]byte(`min_duration: 1000ms`), &rw))
		require.Equal(t, *rw.MinDuration, time.Second)
		*rw.MinDuration = time.Minute
		bts, err := yaml.Marshal(rw)
		require.Nil(t, err)
		require.Equal(t, "min_duration: 1m0s\nsampling_fraction: 0\n", string(bts))
	})
	t.Run("empty record-when", func(t *testing.T) {
		config := &RPCZConfig{}
		require.Nil(t, yaml.Unmarshal(
			[]byte(`fraction: 1.0
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
}
func TestRPCZ_RecordWhen_SamplingFraction(t *testing.T) {
	config := &RPCZConfig{}
	require.Nil(t, yaml.Unmarshal(
		[]byte(
			`fraction: 1.0
capacity: 10
record_when:
  error_codes: [0] 
  min_duration: 10ms
  sample_rate: 0
`), config))
	r := rpcz.NewRPCZ(config.generate())
	var unexpectedIDs []rpcz.SpanID
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
		// mimic some time-consuming operation.
		time.Sleep(15 * time.Millisecond)
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
		// mimic some time-consuming operation.
		time.Sleep(20 * time.Millisecond)
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, _ := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
		unexpectedIDs = append(unexpectedIDs, s.ID())
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
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
		// mimic some time-consuming operation.
		time.Sleep(2 * time.Millisecond)
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	for _, id := range unexpectedIDs {
		_, ok := r.Query(id)
		require.False(t, ok)
	}
}
func TestRPCZ_RecordWhen_ErrorCode(t *testing.T) {
	config := &RPCZConfig{}
	require.Nil(t, yaml.Unmarshal(
		[]byte(
			`fraction: 1.0
capacity: 10
record_when:
  error_codes: [1, 2] # RetServerDecodeFail = 1, RetServerEncodeFail = 2
  min_duration: 1s
  sampling_fraction: 1
`), config))
	r := rpcz.NewRPCZ(config.generate())
	var (
		expectedIDs   []rpcz.SpanID
		unexpectedIDs []rpcz.SpanID
	)
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetServerDecodeFail, ""))
		expectedIDs = append(expectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetServerEncodeFail, ""))
		expectedIDs = append(expectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetUnknown, ""))
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		unexpectedIDs = append(unexpectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, nil)
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
}

func TestRPCZ_RecordWhen_MinDuration(t *testing.T) {
	t.Run("not empty", func(t *testing.T) {
		config := &RPCZConfig{}
		require.Nil(t, yaml.Unmarshal(
			[]byte(
				`fraction: 1.0
capacity: 10
record_when:
  error_codes: [0,] 
  min_duration: 100ms
  sampling_fraction: 1
`), config))
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
		require.Nil(t, yaml.Unmarshal(
			[]byte(
				`fraction: 1.0
capacity: 10
record_when:
  error_codes: [0,]
  sampling_fraction: 1
`), config))
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
func TestRPCZ_RecordWhen_ErrorCodeAndMinDuration(t *testing.T) {
	config := &RPCZConfig{}
	require.Nil(t, yaml.Unmarshal(
		[]byte(
			`fraction: 1.0
capacity: 10
record_when:
  error_codes: [0] # RetOK: 0
  min_duration: 100ms
  sampling_fraction: 1
`), config))
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
		expectedIDs = append(expectedIDs, s.ID())
		ender.End()
	}
	{
		s, ender := r.NewChild("")
		s.SetAttribute(rpcz.TRPCAttributeError, errs.NewFrameError(errs.RetOK, ""))
		// mimic some time-consuming operation.
		time.Sleep(1 * time.Millisecond)
		expectedIDs = append(expectedIDs, s.ID())
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
	for _, id := range expectedIDs {
		_, ok := r.Query(id)
		require.True(t, ok)
	}
	for _, id := range unexpectedIDs {
		_, ok := r.Query(id)
		require.False(t, ok)
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
