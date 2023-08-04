package config_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/config"

	"trpc.group/trpc-go/trpc-go"
)

type mockResponse struct {
	val string
}

func (r *mockResponse) Value() string {
	return r.val
}

func (r *mockResponse) MetaData() map[string]string {
	return nil
}

func (r *mockResponse) Event() config.EventType {
	return config.EventTypeNull
}

type mockKV struct {
	mu sync.RWMutex
	db map[string]string
}

// Put mocks putting key and value into kv storage.
func (kv *mockKV) Put(ctx context.Context, key, val string, opts ...config.Option) error {
	kv.mu.Lock()
	kv.db[key] = val
	kv.mu.Unlock()
	return nil
}

// Get mocks getting value from kv storage by key.
func (kv *mockKV) Get(ctx context.Context, key string, opts ...config.Option) (config.Response, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if val, ok := kv.db[key]; ok {
		v := &mockResponse{val: val}
		return v, nil
	}

	return nil, fmt.Errorf("invalid key")
}

// Watch makes mockKV satisfy the KV interface, this method
// is empty.
func (kv *mockKV) Watch(ctx context.Context, key string, opts ...config.Option) (<-chan config.Response, error) {
	return nil, nil
}

func (kv *mockKV) Name() string {
	return "mock"
}

// Del makes mockKV satisfy the KV interface, this method
// is empty.
func (kv *mockKV) Del(ctx context.Context, key string, opts ...config.Option) error {
	return nil
}

type mockValue struct {
	Age  int
	Name string
}

func TestGlobalKV(t *testing.T) {
	kv := config.GlobalKV()

	mock := "foo"

	err := kv.Put(context.Background(), "mockString", mock)
	assert.Nil(t, err)

	val, err := kv.Get(context.Background(), "mockString")
	assert.NotNil(t, err)
	assert.Nil(t, val)

	err = kv.Del(context.Background(), "mockString")
	assert.Nil(t, err)

	config.SetGlobalKV(&mockKV{})
	assert.NotNil(t, config.GlobalKV())
}

func TestGetConfigInfo(t *testing.T) {
	c := &mockKV{
		db: make(map[string]string),
	}
	config.SetGlobalKV(c)
	config.Register(c)

	{
		tmp := `
age: 20
name: 'foo'
`
		err := c.Put(context.Background(), "mockYAMLKey", tmp)
		assert.Nil(t, err)

		v := &mockValue{}
		err = config.GetYAML("mockYAMLKey", v)
		assert.Nil(t, err)
		assert.Equal(t, 20, v.Age)
		assert.Equal(t, "foo", v.Name)
		err = config.GetYAMLWithProvider("mockYAMLKey", v, "mock")
		assert.Nil(t, err)
		assert.Equal(t, 20, v.Age)
		assert.Equal(t, "foo", v.Name)
		err = config.GetYAMLWithProvider("mockYAMLKey", v, "mockNotExist")
		assert.NotNil(t, err)
	}

	// Test GetJson
	{
		tmp := &mockValue{
			Age:  20,
			Name: "foo",
		}
		tmpStr, err := json.Marshal(tmp)
		assert.Nil(t, err)
		err = c.Put(context.Background(), "mockJsonKey", string(tmpStr))
		assert.Nil(t, err)

		v := &mockValue{}
		err = config.GetJSON("mockJsonKey", v)
		assert.Nil(t, err)
		assert.Equal(t, 20, v.Age)
		assert.Equal(t, "foo", v.Name)
		err = config.GetJSONWithProvider("mockJsonKey", v, "mock")
		assert.Nil(t, err)
		assert.Equal(t, 20, v.Age)
		assert.Equal(t, "foo", v.Name)
		err = config.GetJSONWithProvider("mockJsonKey", v, "mockNotExist")
		assert.NotNil(t, err)

		codec := &config.JSONCodec{}
		out := make(map[string]string)
		codec.Unmarshal(tmpStr, &out)
	}

	// Test GetWithUnmarshal
	{

		v := &mockValue{}
		err := config.GetWithUnmarshal("mockJsonKey1", v, "json")
		assert.NotNil(t, err)
	}

	// Test GetToml
	{
		tmp := `
age = 20
name = "foo"
`
		err := c.Put(context.Background(), "mockTomlKey", tmp)
		assert.Nil(t, err)

		v := &mockValue{}
		err = config.GetTOML("mockTomlKey", v)
		assert.Nil(t, err)
		assert.Equal(t, 20, v.Age)
		assert.Equal(t, "foo", v.Name)
		err = config.GetTOMLWithProvider("mockTomlKey", v, "mock")
		assert.Nil(t, err)
		assert.Equal(t, 20, v.Age)
		assert.Equal(t, "foo", v.Name)
		err = config.GetTOMLWithProvider("mockTomlKey", v, "mockNotExist")
		assert.NotNil(t, err)
	}

	// Test GetString
	{
		mock := "foo"
		c.Put(context.Background(), "mockString", mock)
		val, err := config.GetString("mockString")
		assert.Nil(t, err)
		assert.Equal(t, mock, val)
		_, err = config.GetString("mockString1")
		assert.NotNil(t, err)
	}
	{
		mock := 1
		c.Put(context.Background(), "mockInt", fmt.Sprint(mock))
		val, err := config.GetInt("mockInt")
		assert.Nil(t, err)
		assert.Equal(t, mock, val)
		_, err = config.GetInt("mockInt1")
		assert.NotNil(t, err)
	}

	// Test Get
	{
		c := config.Get("mock")
		assert.NotNil(t, c)
	}
}

// TestGetConfigGetDefault tests getting default value when
// key is absent.
func TestGetConfigGetDefault(t *testing.T) {
	c := &mockKV{
		db: make(map[string]string),
	}
	config.SetGlobalKV(c)
	config.Register(c)

	// Test GetStringWithDefault
	{
		// get key successfully.
		mock := "foo"
		c.Put(context.Background(), "mockString", mock)
		val := config.GetStringWithDefault("mockString", "otherValue")
		assert.Equal(t, mock, val)

		// key is absent, get default value.
		def := "myDefaultValue"
		val = config.GetStringWithDefault("whatever", def)
		assert.Equal(t, val, def)
	}
	// Test GetIntWithDefault
	{
		// get key successfully.
		mockint := 555
		c.Put(context.Background(), "mockInt", fmt.Sprint(mockint))
		val := config.GetIntWithDefault("mockInt", 123)
		assert.Equal(t, mockint, val)

		// key is absent, get default value.
		def := 888
		val = config.GetIntWithDefault("whatever", def)
		assert.Equal(t, val, def)

		// key exists, but fail to transfers to target type,
		// get default value.
		mockstr := "foo"
		c.Put(context.Background(), "whatever", mockstr)
		val = config.GetIntWithDefault("whatever", def)
		assert.Equal(t, val, def)
	}
}

func TestLoadYaml(t *testing.T) {
	require := require.New(t)
	err := config.Reload("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.NotNil(err)

	c, err := config.Load("../testdata/trpc_go.yaml.1", config.WithCodec("yaml"))
	require.NotNil(err)

	c, err = config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")
	// out := &T{}
	out := c.GetString("server.app", "")
	t.Logf("return %+v", out)
	require.Equal(out, "test", "app name is wrong")

	buf := c.Bytes()
	require.NotNil(buf)
	bytes.Contains(buf, []byte("test"))

	err = config.Reload("../testdata/trpc_go.yaml")
	require.Nil(err)

	require.Implements((*config.Config)(nil), c)
}

func TestLoadToml(t *testing.T) {
	require := require.New(t)
	rightPath := "../testdata/custom.toml"
	wrongPath := "../testdata/custom.toml.1"

	err := config.Reload(rightPath, config.WithCodec("toml"))
	require.NotNil(err)

	c, err := config.Load(wrongPath, config.WithCodec("toml"))
	require.NotNil(err, "path not exist")
	t.Logf("load with not exist path, err:%v", err)

	c, err = config.Load(rightPath, config.WithCodec("toml"))
	require.Nil(err, "failed to load config")
	//out := &T{}
	out := c.GetString("server.app", "")
	t.Logf("return %s", out)
	require.Equal(out, "test", "app name is wrong")

	buf := c.Bytes()
	require.NotNil(buf)
	bytes.Contains(buf, []byte("test"))

	obj := struct {
		Server struct {
			App      string
			P        int `toml:"port"`
			Protocol []string
		}
	}{}

	err = c.Unmarshal(&obj)
	require.Nil(err, "unmarshal should succ")
	t.Logf("unmarshal struct:%+v", obj)
	require.Equal(obj.Server.P, 1000)
	require.Equal(len(obj.Server.Protocol), 2)

	err = config.Reload("../testdata/custom.toml", config.WithCodec("toml"))
	require.Nil(err)

	require.Implements((*config.Config)(nil), c)
}

func TestLoadUnmarshal(t *testing.T) {
	require := require.New(t)
	config, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")

	out := &trpc.Config{}
	err = config.Unmarshal(out)

	require.Nil(err, "failed to load config")
	t.Logf("return %+v", *out)
}

func TestLoadUnmarshalClient(t *testing.T) {
	require := require.New(t)
	config, err := config.Load("../testdata/client.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")

	out := client.DefaultClientConfig()
	err = config.Unmarshal(&out)
	t.Logf("return %+v %s", out["Test.HelloServer"], err)
	require.Nil(err, "failed to load client config")
}

func TestGetString(t *testing.T) {
	require := require.New(t)
	c, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")

	out := c.GetString("server.app", "cc")
	t.Logf("return %+v", out)
	require.Equal("test", out, "app name is wrong")

	out = c.GetString("server.app1", "cc")
	t.Logf("return %+v", out)
	require.Equal("cc", out, "app name is wrong")

	out = c.GetString("server.admin.port", "cc")
	t.Logf("return %+v", out)
	require.Equal("9528", out, "app name is wrong")

	out = c.GetString("server.admin", "cc")
	t.Logf("return %+v", out)
	require.Equal("cc", out, "app name is wrong")
}

func TestGetBool(t *testing.T) {
	require := require.New(t)
	c, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")

	out := c.GetBool("server.admin_port123", false)
	t.Logf("return %+v", out)
	require.Equal(false, out)

	out = c.GetBool("server.app", false)
	t.Logf("return %+v", out)
	require.Equal(false, out)
}

func TestGet(t *testing.T) {
	require := require.New(t)
	c, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")

	out := c.Get("server.admin_port123", 10001)
	t.Logf("return %+v", out)
	require.Equal(10001, out)
}

func TestGetUint(t *testing.T) {
	require := require.New(t)
	c, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")

	{
		actual := uint(9528)
		dft := uint(10001)

		out := c.GetUint("server.admin.port", dft)
		t.Logf("return %+v", out)
		require.Equal(actual, out)

		out = c.GetUint("server.admin_port123", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

		out = c.GetUint("server.app", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)
	}

	{
		actual := uint32(9528)
		dft := uint32(10001)

		out := c.GetUint32("server.admin.port", dft)
		t.Logf("return %+v", out)
		require.Equal(actual, out)

		out = c.GetUint32("server.admin_port123", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

		out = c.GetUint32("server.app", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)
	}

	{
		actual := uint64(9528)
		dft := uint64(10001)

		out := c.GetUint64("server.admin.port", dft)
		t.Logf("return %+v", out)
		require.Equal(actual, out)

		out = c.GetUint64("server.admin_port123", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

		out = c.GetUint64("server.app", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)
	}

}

func TestGetInt(t *testing.T) {
	require := require.New(t)
	c, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")

	{
		actual := 9528
		dft := 10001

		out := c.GetInt("server.admin.port", dft)
		t.Logf("return %+v", out)
		require.Equal(actual, out)

		out = c.GetInt("server.admin_port123", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

		out = c.GetInt("server.app", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)
	}

	{
		actual := int32(9528)
		dft := int32(10001)

		out := c.GetInt32("server.admin.port", dft)
		t.Logf("return %+v", out)
		require.Equal(actual, out)

		out = c.GetInt32("server.admin_port123", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

		out = c.GetInt32("server.app", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)
	}

	{
		actual := int64(9528)
		dft := int64(10001)

		out := c.GetInt64("server.admin.port", dft)
		t.Logf("return %+v", out)
		require.Equal(actual, out)

		out = c.GetInt64("server.admin_port123", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

		out = c.GetInt64("server.app", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)
	}

}

func TestGetFloat(t *testing.T) {
	require := require.New(t)
	c, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")

	{
		actual := float64(9528)
		dft := float64(1.0)

		out := c.GetFloat64("server.admin.port", dft)
		t.Logf("return %+v", out)
		require.Equal(actual, out)

		out = c.GetFloat64("server.admin_port123", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

		out = c.GetFloat64("server.app", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

	}

	{
		actual := float32(9528)
		dft := float32(1.0)

		out := c.GetFloat32("server.admin.port", dft)
		t.Logf("return %+v", out)
		require.Equal(actual, out)

		out = c.GetFloat32("server.admin_port123", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

		out = c.GetFloat32("server.app", dft)
		t.Logf("return %+v", out)
		require.Equal(dft, out)

	}
}

func TestIsSet(t *testing.T) {
	require := require.New(t)
	c, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(err, "failed to load config")

	out := c.IsSet("server.admin.port")
	require.Equal(true, out)
	out = c.IsSet("server.admin_port1")
	require.Equal(false, out)
}

func TestUnmarshal(t *testing.T) {
	require := require.New(t)
	c, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"), config.WithProvider("file"))
	require.Nil(err, "failed to load config")
	var b struct {
		Server struct {
			App string
		}
	}
	err = c.Unmarshal(&b)
	require.Nil(err)
	require.Equal("test", b.Server.App, "failed to read item")
}

func TestLoad(t *testing.T) {
	c, err := config.Load("../testdata/trpc_go.yaml2", config.WithCodec("yaml"), config.WithProvider("file"))
	assert.NotNil(t, err)
	assert.Nil(t, c)

	c, err = config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml1"))
	assert.NotNil(t, err)
	assert.Nil(t, c)

	c, err = config.Load("../testdata/trpc_go.yaml", config.WithProvider("etcd"))
	assert.NotNil(t, err)
	assert.Nil(t, c)
}

func TestProvider(t *testing.T) {
	require := require.New(t)
	p := &config.FileProvider{}
	require.Equal("file", p.Name())
	config.RegisterProvider(p)
	pp := config.GetProvider("file")
	require.Equal(p, pp)
}
