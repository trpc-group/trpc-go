# How to mock `Watch`

```go
import (
    "context"
    "testing"

    "git.code.oa.com/trpc-go/trpc-go/config"
    mock "git.code.oa.com/trpc-go/trpc-go/config/mockconfig"
    "github.com/golang/mock/gomock"
    "github.com/stretchr/testify/assert"
)

func TestWatch(t *testing.T) {
    const (
        mockKey = "test-key"
        mockProvider = "test-provider"
    )
    mockChan := make(chan config.Response, 1)
    mockResp := &mockResponse{val: "mock-value"}
    mockChan <- mockResp

    ctrl := gomock.NewController(t)
    kv := mock.NewMockKVConfig(ctrl)
    kv.EXPECT().Name().Return(mockProvider).AnyTimes()
    m := kv.EXPECT().Watch(gomock.Any(), mockKey, gomock.Any()).AnyTimes()
    m.DoAndReturn(func(ctx context.Context, key string, opts ...config.Option) (<-chan config.Response, error) 
        return mockChan, nil
    })
    config.Register(kv)

    got, err := config.Get(mockProvider).Watch(context.TODO(), mockKey)
    assert.Nil(t, err)
    assert.NotNil(t, got)
    assert.Equal(t, mockResp, <-got)
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