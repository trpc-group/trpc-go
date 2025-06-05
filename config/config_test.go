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
	"trpc.group/trpc-go/trpc-go/errs"

	trpc "trpc.group/trpc-go/trpc-go"
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
	t.Run("GetYAML", func(t *testing.T) {
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
	})
	t.Run("GetJson", func(t *testing.T) {
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
	})
	t.Run("GetWithUnmarshal", func(t *testing.T) {
		v := &mockValue{}
		err := config.GetWithUnmarshal("mockJsonKey1", v, "json")
		assert.NotNil(t, err)
	})
	t.Run("GetToml", func(t *testing.T) {
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
	})
	t.Run("GetString", func(t *testing.T) {
		mock := "foo"
		c.Put(context.Background(), "mockString", mock)
		val, err := config.GetString("mockString")
		assert.Nil(t, err)
		assert.Equal(t, mock, val)

		_, err = config.GetString("mockString1")
		assert.NotNil(t, err)
	})
	t.Run("GetInt", func(t *testing.T) {
		mock := 1
		c.Put(context.Background(), "mockInt", fmt.Sprint(mock))
		val, err := config.GetInt("mockInt")
		assert.Nil(t, err)
		assert.Equal(t, mock, val)

		_, err = config.GetInt("mockInt1")
		assert.NotNil(t, err)
	})
	t.Run("Get", func(t *testing.T) {
		c := config.Get("mock")
		assert.NotNil(t, c)
	})
}

// TestGetConfigGetDefault tests getting default value when
// key is absent.
func TestGetConfigGetDefault(t *testing.T) {
	c := &mockKV{
		db: make(map[string]string),
	}
	config.SetGlobalKV(c)
	config.Register(c)
	t.Run("GetStringWithDefault", func(t *testing.T) {
		mock := "foo"
		c.Put(context.Background(), "mockString", mock)
		val := config.GetStringWithDefault("mockString", "otherValue")
		assert.Equal(t, mock, val)

		// key is absent, get default value.
		def := "myDefaultValue"
		val = config.GetStringWithDefault("whatever", def)
		assert.Equal(t, val, def)
	})
	t.Run("GetIntWithDefault", func(t *testing.T) {
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
	})
}

func TestLoadYaml(t *testing.T) {
	err := config.Reload("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.NotNil(t, err)

	c, err := config.Load("../testdata/trpc_go.yaml.1", config.WithCodec("yaml"))
	require.NotNil(t, err)

	c, err = config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.Nil(t, err, "failed to load config")

	out := c.GetString("server.app", "")
	t.Logf("return %+v", out)
	require.Equal(t, out, "test", "app name is wrong")

	buf := c.Bytes()
	require.NotNil(t, buf)
	bytes.Contains(buf, []byte("test"))

	err = config.Reload("../testdata/trpc_go.yaml")
	require.Nil(t, err)

	require.Implements(t, (*config.Config)(nil), c)
}

func TestLoadToml(t *testing.T) {
	rightPath := "../testdata/custom.toml"
	wrongPath := "../testdata/custom.toml.1"

	err := config.Reload(rightPath, config.WithCodec("toml"))
	require.NotNil(t, err)

	c, err := config.Load(wrongPath, config.WithCodec("toml"))
	require.NotNil(t, err, "path not exist")
	t.Logf("load with not exist path, err: %v", err)

	c, err = config.Load(rightPath, config.WithCodec("toml"))
	require.Nil(t, err, "failed to load config")

	out := c.GetString("server.app", "")
	t.Logf("return %s", out)
	require.Equal(t, out, "test", "app name is wrong")

	buf := c.Bytes()
	require.NotNil(t, buf)
	bytes.Contains(buf, []byte("test"))

	obj := struct {
		Server struct {
			App      string
			P        int `toml:"port"`
			Protocol []string
		}
	}{}

	err = c.Unmarshal(&obj)
	require.Nil(t, err, "unmarshal should succ")
	t.Logf("unmarshal struct: %+v", obj)
	require.Equal(t, obj.Server.P, 1000)
	require.Equal(t, len(obj.Server.Protocol), 2)

	err = config.Reload("../testdata/custom.toml", config.WithCodec("toml"))
	require.Nil(t, err)

	require.Implements(t, (*config.Config)(nil), c)
}

func TestLoadUnmarshal(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	out := &trpc.Config{}
	err := c.Unmarshal(out)
	require.Nil(t, err, "failed to load config")
	t.Logf("return %+v", *out)
}

func TestLoadUnmarshalClient(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"))

	out := client.DefaultClientConfig()
	err := c.Unmarshal(&out)
	t.Logf("return %+v %s", out["Test.HelloServer"], err)
	require.Nil(t, err, "failed to load client config")
}

func TestGetString(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	t.Run("key is absent", func(t *testing.T) {
		require.Equal(t, "cc", c.GetString("server.app1", "cc"), "app name is wrong")
		require.Equal(t, "cc", c.GetString("server.admin", "cc"), "app name is wrong")
	})
	t.Run("key is present", func(t *testing.T) {
		require.Equal(t, "test", c.GetString("server.app", "cc"), "app name is wrong")
		require.Equal(t, "9528", c.GetString("server.admin.port", "cc"), "app name is wrong")
	})
}

func TestGetBool(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.False(t, c.GetBool("server.admin_port123", false))
	require.False(t, c.GetBool("server.app", false))
}

func TestGet(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	const defaultValue = 10001
	require.Equal(t, defaultValue, c.Get("server.admin_port123", defaultValue))
}

func TestGetUint(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"))

	t.Run("uint", func(t *testing.T) {
		actual := uint(9528)
		defaultValue := uint(10001)
		require.Equal(t, actual, c.GetUint("server.admin.port", defaultValue))
		require.Equal(t, defaultValue, c.GetUint("server.admin_port123", defaultValue))
		require.Equal(t, defaultValue, c.GetUint("server.app", defaultValue))
	})
	t.Run("uint32", func(t *testing.T) {
		actual := uint32(9528)
		defaultValue := uint32(10001)
		require.Equal(t, actual, c.GetUint32("server.admin.port", defaultValue))
		require.Equal(t, defaultValue, c.GetUint32("server.admin_port123", defaultValue))
		require.Equal(t, defaultValue, c.GetUint32("server.app", defaultValue))
	})
	t.Run("uint64", func(t *testing.T) {
		actual := uint64(9528)
		defaultValue := uint64(10001)
		require.Equal(t, actual, c.GetUint64("server.admin.port", defaultValue))
		require.Equal(t, defaultValue, c.GetUint64("server.admin_port123", defaultValue))
		require.Equal(t, defaultValue, c.GetUint64("server.app", defaultValue))
	})
}

func TestGetInt(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	t.Run("int", func(t *testing.T) {
		actual := 9528
		defaultValue := 10001
		require.Equal(t, actual, c.GetInt("server.admin.port", defaultValue))
		require.Equal(t, defaultValue, c.GetInt("server.admin_port123", defaultValue))
		require.Equal(t, defaultValue, c.GetInt("server.app", defaultValue))
	})
	t.Run("int32", func(t *testing.T) {
		actual := int32(9528)
		defaultValue := int32(10001)
		require.Equal(t, actual, c.GetInt32("server.admin.port", defaultValue))
		require.Equal(t, defaultValue, c.GetInt32("server.admin_port123", defaultValue))
		require.Equal(t, defaultValue, c.GetInt32("server.app", defaultValue))
	})
	t.Run("int64", func(t *testing.T) {
		actual := int64(9528)
		defaultValue := int64(10001)
		require.Equal(t, actual, c.GetInt64("server.admin.port", defaultValue))
		require.Equal(t, defaultValue, c.GetInt64("server.admin_port123", defaultValue))
		require.Equal(t, defaultValue, c.GetInt64("server.app", defaultValue))
	})
}

func TestGetFloat(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	t.Run("float64", func(t *testing.T) {
		actual := float64(9528)
		defaultValue := 1.0
		require.Equal(t, actual, c.GetFloat64("server.admin.port", defaultValue))
		require.Equal(t, defaultValue, c.GetFloat64("server.admin_port123", defaultValue))
		require.Equal(t, defaultValue, c.GetFloat64("server.app", defaultValue))
	})
	t.Run("float32", func(t *testing.T) {
		actual := float32(9528)
		defaultValue := float32(1.0)
		require.Equal(t, actual, c.GetFloat32("server.admin.port", defaultValue))
		require.Equal(t, defaultValue, c.GetFloat32("server.admin_port123", defaultValue))
		require.Equal(t, defaultValue, c.GetFloat32("server.app", defaultValue))
	})
}

func TestIsSet(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"))
	require.True(t, c.IsSet("server.admin.port"))
	require.False(t, c.IsSet("server.admin_port1"))
}

func TestUnmarshal(t *testing.T) {
	c := mustLoad(t, "../testdata/trpc_go.yaml", config.WithCodec("yaml"), config.WithProvider("file"))
	var b struct {
		Server struct {
			App string
		}
	}
	err := c.Unmarshal(&b)
	require.Nil(t, err)
	require.Equal(t, "test", b.Server.App, "failed to read item")
}

func mustLoad(t *testing.T, path string, opts ...config.LoadOption) config.Config {
	t.Helper()

	c, err := config.Load(path, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLoad(t *testing.T) {
	t.Run("nonexistent config path", func(t *testing.T) {
		c, err := config.Load("../testdata/trpc_go.yaml2", config.WithCodec("yaml"), config.WithProvider("file"))
		require.Contains(t, errs.Msg(err), "failed to load")
		require.Nil(t, c)
	})
	t.Run("nonexistent codec ", func(t *testing.T) {
		c, err := config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml1"))
		require.ErrorIs(t, err, config.ErrCodecNotExist)
		require.Nil(t, c)
	})
	t.Run("nonexistent provider", func(t *testing.T) {
		c, err := config.Load("../testdata/trpc_go.yaml", config.WithProvider("etcd"))
		require.ErrorIs(t, err, config.ErrProviderNotExist)
		require.Nil(t, c)
	})
}

func TestProvider(t *testing.T) {
	p := &config.FileProvider{}
	require.Equal(t, "file", p.Name())

	config.RegisterProvider(p)
	require.Equal(t, p, config.GetProvider("file"))
}
