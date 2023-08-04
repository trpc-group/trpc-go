# tRPC-Go Config Package

trpc-go/config is a componentized config package, it provides an easy way to read multi config content sources and multi config file types.

## Features

- plug-in: could load configuration from multi content sources, such as local file, TConf, Rainbow, etc.

- hot loading: auto loading the new one when configuration changes.

- default setting: if configuration is lost for some reasons, the default value can be used.

## Core Data Structure

- Config: common configuration interface, provides the reading config operation.

- ConfigLoader: config loader interface, by this we can implement config loading strategy.

- Codec: config codec interface, by this we can support multi config type.

- DataProvider: data provider interface, by this we can support multi content sources, such as `file`，`tconf`，`rainbow`.

## How To Use

```go
import (
    "trpc.group/trpc-go/trpc-go/config"
)


// load local config file: config.WithProvider("file")
config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"), config.WithProvider("file"))

// the default data provider is local file.
config.Load("../testdata/trpc_go.yaml", config.WithCodec("yaml"))

// load TConf config file: config.WithProvider("tconf")
c, _ := config.Load("test.yaml", config.WithCodec("yaml"), config.WithProvider("tconf"))

// read bool value
c.GetBool("server.debug", false)

// read string value
c.GetString("server.app", "default")

```

### Concurrency-safely watching remote configuration changes

```go
import (
	"sync/atomic"
    ...
)

type yamlFile struct {
    Server struct {
        App string
    }
}

// see: https://golang.org/pkg/sync/atomic/#Value
var cfg atomic.Value // Concurrency-safe Value

// watching tconf remote config changes by Watch interface in trpc-go/config
c, _ := config.Get("tconf").Watch(context.TODO(), "test.yaml")

go func() {
    for r := range c {
        yf := &yamlFile{}
        fmt.Printf("event: %d, value: %s", r.Event(), r.Value())

        if err := yaml.Unmarshal([]byte(r.Value()), yf); err == nil {
            cfg.Store(yf)
        }

    }
}()

// after init config, we can get newest config object by atomic.Value.Load()
cfg.Load().(*yamlFile)

```

### How To Mock Watching
When writing unit test code, we need create a stub for related function, and replace the original implementation. Other function is in the same way, here is the example of Watch().

```go

import (
	"context"
	"testing"

	"trpc.group/trpc-go/trpc-go/config"
	mock "trpc.group/trpc-go/trpc-go/config/mockconfig"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestWatch(t *testing.T) {
	ctrl := gomock.NewController(t)

    // mock rainbow and its key.
	mockKey := "test.json"
	mockProvider := "rainbow"

    // watching data comes from writing mock channel.
	mockChan := make(chan config.Response, 1)
	mockResp := &mockResponse{val: "mock-value"}
	mockChan <- mockResp

    // mock the process of Watch.
	kv := mock.NewMockKVConfig(ctrl)
	kv.EXPECT().Name().Return(mockProvider).AnyTimes()
	m := kv.EXPECT().Watch(gomock.Any(), mockKey, gomock.Any()).AnyTimes()
	m.DoAndReturn(func(ctx context.Context, key string, opts ...config.Option) (<-chan config.Response, error) {
		return mockChan, nil
	})
    // register kv config.
	config.Register(kv)

	// action
	got, err := config.Get(mockProvider).Watch(context.TODO(), mockKey)
	assert.Nil(t, err)
	assert.NotNil(t, got)

	resp := <-got
	assert.Equal(t, mockResp, resp)
}

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


```

