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

package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
)

func Test_search(t *testing.T) {
	type args struct {
		unmarshalledData map[string]interface{}
		keys             []string
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "empty keys",
			args: args{
				keys: nil,
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				if !errors.Is(err, ErrConfigNotExist) {
					t.Errorf("received unexpected error got: %+v, want: +%v", err, ErrCodecNotExist)
					return false
				}
				return true
			},
		},
		{
			name: "key doesn't match",
			args: args{
				unmarshalledData: map[string]interface{}{
					"1": []string{"x", "y"},
				},
				keys: []string{"not-1"},
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				if !errors.Is(err, ErrConfigNotExist) {
					t.Errorf("received unexpected error got: %+v, want: +%v", err, ErrCodecNotExist)
					return false
				}
				return true
			},
		},
		{
			name: "value of unmarshalledData isn't map type",
			args: args{
				unmarshalledData: map[string]interface{}{
					"1": []string{"x", "y"},
				},
				keys: []string{"1", "2"},
			},
			want: nil,
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				if !errors.Is(err, ErrConfigNotExist) {
					t.Errorf("received unexpected error got: %+v, want: +%v", err, ErrCodecNotExist)
					return false
				}
				return true
			},
		},
		{
			name: "value of unmarshalledData is map[interface{}]interface{} type",
			args: args{
				unmarshalledData: map[string]interface{}{
					"1": map[interface{}]interface{}{"x": "y"},
				},
				keys: []string{"1", "x"},
			},
			want:    "y",
			wantErr: assert.NoError,
		},
		{
			name: "value of unmarshalledData is map[string]interface{} type",
			args: args{
				unmarshalledData: map[string]interface{}{
					"1": map[string]interface{}{"x": "y"},
				},
				keys: []string{"1", "x"},
			},
			want:    "y",
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := search(tt.args.unmarshalledData, tt.args.keys)
			if !tt.wantErr(t, err, fmt.Sprintf("search(%v, %v)", tt.args.unmarshalledData, tt.args.keys)) {
				return
			}
			assert.Equalf(t, tt.want, got, "search(%v, %v)", tt.args.unmarshalledData, tt.args.keys)
		})
	}
}

func TestTrpcConfig_Load(t *testing.T) {
	t.Run("parse failed", func(t *testing.T) {
		c, err := newTrpcConfig("../testdata/trpc_go.yaml")
		require.Nil(t, err)
		c.decoder = &TomlCodec{}
		err = c.Load()
		require.Contains(t, errs.Msg(err), "failed to parse")
	})
}
func TestYamlCodec_Unmarshal(t *testing.T) {
	t.Run("interface", func(t *testing.T) {
		var tt interface{}
		tt = map[string]interface{}{}
		require.Nil(t, GetCodec("yaml").Unmarshal([]byte("[1, 2]"), &tt))
	})
	t.Run("map[string]interface{}", func(t *testing.T) {
		tt := map[string]interface{}{}
		require.NotNil(t, GetCodec("yaml").Unmarshal([]byte("[1, 2]"), &tt))
	})
}

func TestEnvExpanded(t *testing.T) {
	RegisterProvider(NewEnvProvider(t.Name(), []byte(`
password: ${pwd}
`)))

	t.Setenv("pwd", t.Name())
	cfg, err := DefaultConfigLoader.Load(
		t.Name(),
		WithProvider(t.Name()),
		WithExpandEnv())
	require.Nil(t, err)

	require.Equal(t, t.Name(), cfg.GetString("password", ""))
	require.Contains(t, string(cfg.Bytes()), fmt.Sprintf("password: %s", t.Name()))
}

func TestCodecUnmarshalDstMustBeMap(t *testing.T) {
	filePath := t.TempDir() + "/conf.map"
	require.Nil(t, os.WriteFile(filePath, []byte{}, 0600))
	RegisterCodec(dstMustBeMapCodec{})
	_, err := DefaultConfigLoader.Load(filePath, WithCodec(dstMustBeMapCodec{}.Name()))
	require.Nil(t, err)
}

func NewEnvProvider(name string, data []byte) *EnvProvider {
	return &EnvProvider{
		name: name,
		data: data,
	}
}

type EnvProvider struct {
	name string
	data []byte
}

func (ep *EnvProvider) Name() string {
	return ep.name
}

func (ep *EnvProvider) Read(string) ([]byte, error) {
	return ep.data, nil
}

func (ep *EnvProvider) Watch(cb ProviderCallback) {
	cb("", ep.data)
}

func TestWatch(t *testing.T) {
	p := manualTriggerWatchProvider{}
	var msgs = make(chan WatchMessage)
	SetDefaultWatchHook(func(msg WatchMessage) {
		if msg.Error != nil {
			log.Errorf("config watch error: %+v", msg)
		} else {
			log.Infof("config watch error: %+v", msg)
		}
		msgs <- msg
	})

	RegisterProvider(&p)
	p.Set("key", []byte(`key: value`))
	ops := []LoadOption{WithProvider(p.Name()), WithCodec("yaml"), WithWatch()}
	c1, err := DefaultConfigLoader.Load("key", ops...)
	require.Nilf(t, err, "first load config:%+v", c1)
	require.True(t, c1.IsSet("key"), "first load config key exist")
	require.Equal(t, c1.Get("key", "default"), "value", "first load config get key value")

	var c2 Config
	c2, err = DefaultConfigLoader.Load("key", ops...)
	require.Nil(t, err, "second load config:%+v", c2)
	require.Equal(t, c1, c2, "first and second load config not equal")
	require.True(t, c2.IsSet("key"), "second load config key exist")
	require.Equal(t, c2.Get("key", "default"), "value", "second load config get key value")

	var gw sync.WaitGroup
	gw.Add(1)
	go func() {
		defer gw.Done()
		tt := time.NewTimer(time.Second)
		select {
		case <-msgs:
		case <-tt.C:
			t.Errorf("receive message timeout")
		}
	}()

	p.Set("key", []byte(`:key: value:`))
	gw.Wait()

	var c3 Config
	c3, err = DefaultConfigLoader.Load("key", WithProvider(p.Name()), WithWatchHook(func(msg WatchMessage) {
		msgs <- msg
	}))
	require.Contains(t, errs.Msg(err), "failed to parse")
	require.Nil(t, c3, "update error")

	require.True(t, c2.IsSet("key"), "third load config key exist")
	require.Equal(t, c2.Get("key", "default"), "value", "third load config get key value")

	gw.Add(1)
	go func() {
		defer gw.Done()
		for i := 0; i < 2; i++ {
			tt := time.NewTimer(time.Second)
			select {
			case <-msgs:
			case <-tt.C:
				t.Errorf("receive message timeout number%d ", i)
			}
		}
	}()
	p.Set("key", []byte(`key: value2`))
	gw.Wait()

	require.Truef(t, c2.IsSet("key"), "after update config and get key exist")
	require.Equal(t, c2.Get("key", "default"), "value2", "after update config and config get value")
}

var _ DataProvider = (*manualTriggerWatchProvider)(nil)

type manualTriggerWatchProvider struct {
	values    sync.Map
	callbacks []ProviderCallback
}

func (m *manualTriggerWatchProvider) Name() string {
	return "manual_trigger_watch_provider"
}

func (m *manualTriggerWatchProvider) Read(s string) ([]byte, error) {
	if v, ok := m.values.Load(s); ok {
		return v.([]byte), nil
	}
	return nil, fmt.Errorf("not found config")
}

func (m *manualTriggerWatchProvider) Watch(callback ProviderCallback) {
	m.callbacks = append(m.callbacks, callback)
}

func (m *manualTriggerWatchProvider) Set(key string, v []byte) {
	m.values.Store(key, v)
	for _, callback := range m.callbacks {
		callback(key, v)
	}
}

type dstMustBeMapCodec struct{}

func (c dstMustBeMapCodec) Name() string {
	return "map"
}

func (c dstMustBeMapCodec) Unmarshal(bts []byte, dst interface{}) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr ||
		rv.Elem().Kind() != reflect.Interface ||
		rv.Elem().Elem().Kind() != reflect.Map ||
		rv.Elem().Elem().Type().Key().Kind() != reflect.String ||
		rv.Elem().Elem().Type().Elem().Kind() != reflect.Interface {
		return errors.New("the dst of codec.Unmarshal must be a map")
	}
	return nil
}
