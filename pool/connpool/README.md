# connection pool

### Connection pool parameters
````go
type Options struct {
	MaxIdle int // The maximum number of idle connections, 0 means no idle, the default value of the framework is 2048.
	MaxActive int // Maximum number of active connections, 0 means no limit.
	Wait bool // Whether to wait when the maximum number of active connections is reached.
	IdleTimeout time.Duration // idle connection timeout time, the default value of the framework is 50s.
	MaxConnLifetime time.Duration // Maximum lifetime of the connection.
	DialTimeout time.Duration // Connection establishment timeout, the default value of the framework is 200ms.
}
````

### Custom connection pool

```go
import "trpc.group/trpc-go/trpc-go/pool/connpool"

func init() {
	connpool.DefaultConnectionPool = connpool.NewConnectionPool(connpool.WithMaxIdle(2048))
}
```