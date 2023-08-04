# 连接池

### 连接池参数
```go
type Options struct {
	MaxIdle         int           // 最大闲置连接数量，0 代表不做闲置，框架默认值 2048
	MaxActive       int           // 最大活跃连接数量，0 代表不做限制
	Wait            bool          // 活跃连接达到最大数量时，是否等待
	IdleTimeout     time.Duration // 空闲连接超时时间，框架默认值 50s
	MaxConnLifetime time.Duration // 连接的最大生命周期
	DialTimeout     time.Duration // 建立连接超时时间，框架默认值 200ms
}
```

### 自定义连接池

```go
import "trpc.group/trpc-go/trpc-go/pool/connpool"

func init() {
    connpool.DefaultConnectionPool = connpool.NewConnectionPool(connpool.WithMaxIdle(2048))
}
```
