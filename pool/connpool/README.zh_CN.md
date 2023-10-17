[English](README.md) | 中文

## 背景

客户端请求服务端，如果是以 tcp 协议进行通信的话，需要考虑三次握手建立连接的开销。通常情况下 tcp 通信模式会预先建立好连接，或者发起请求时建立好连接，这里的连接用完之后不会直接 close 掉，而是会被后续复用。  
连接池就是为了实现此功能而进行的一定程度的封装。

## 原理

pool 维护一个 sync.Map 作为连接池，key 为<network, address, protocol>编码，value 为与目标地址建立的连接构成的 ConnectionPool, 其内部以一个链表维护空闲连接。在短连接模式中，transport 层会在 rpc 调用后关闭连接，而在连接池模式中，会把使用完的连接放回连接池，以待下次需要时取出。  
为实现上述目的，连接池需要具备以下功能：
- 提供可用连接，包括创建新连接和复用空闲连接；
- 回收上层使用过的连接作为空闲连接管理；
- 对连接池中空闲连接的管理能力，包括复用连接的选择策略，空闲连接的健康监测等；
- 根据用户配置调整连接池运行参数。

## 设计实现

连接池的整体代码结构如下图所示：
![design_implementation](/.resources/pool/connpool/design_implementation.png)

### 初始化连接池

`NewConnectionPool` 创建一个连接池，支持传入 Option 修改参数，不传则使用默认值初始化。Dial 是默认的创建连接方式，每个 ConnectionPool 会根据自己的 GetOptions 生成 DialOptions, 来建立对应目标的连接。

```go
func NewConnectionPool(opt ...Option) Pool {
  opts := &Options{
    MaxIdle: defaultMaxIdle,
    IdleTimeout: defaultIdleTimeout,
    DialTimeout: defaultDialTimeout,
    Dial: Dial,
  }
  for _, o := range opt {
    o(opts)
  }
  return &pool{
    opts: opts,
    connectionPools: new(sync.Map),
  }
}
```

### 获取连接

通过 pool.Get 可以获取一个连接，参考 client_transport_tcp.go 的实现。

```go
// Get
getOpts := connpool.NewGetOptions()
getOpts.WithContext(ctx)
getOpts.WithFramerBuilder(opts.FramerBuilder)
getOpts.WithDialTLS(opts.TLSCertFile, opts.TLSKeyFile, opts.CACertFile, opts.TLSServerName)
getOpts.WithLocalAddr(opts.LocalAddr)
getOpts.WithDialTimeout(opts.DialTimeout)
getOpts.WithProtocol(opts.Protocol)
conn, err = opts.Pool.Get(opts.Network, opts.Address, getOpts)
```

ConnPool 对外仅暴露 Get 接口，确保连接池状态不会因用户的误操作被破坏。

`Get` 会根据 <network, address, protocol> 获取 ConnectionPool, 如果获取失败需要首先创建，这里做了并发控制，防止 ConnectionPool 被重复建立，核心代码如下所示：

```go
func (p *pool) Get(network string, address string, opts GetOptions) (net.Conn, error) {
  // ...
  key := getNodeKey(network, address, opts.Protocol)
  if v, ok := p.connectionPools.Load(key); ok {
    return v.(*ConnectionPool).Get(ctx)
  }
  // create newPool...
  v, ok := p.connectionPools.LoadOrStore(key, newPool)
  if !ok {
    // init newPool...
    return newPool.Get(ctx)
  }
  return v.(*ConnectionPool).Get(ctx)
}
```

获取到 ConnectionPool 后，尝试获取连接。首先需要获取 token, token 是一个用于并发控制的 ch, 其缓冲长度根据 MaxActive 设置，代表用户可以同时使用 MaxActive 个连接，当活跃连接被归还连接池或关闭时，归还 token. 如果设置 `Wait=True`, 会在获取不到 token 时等待直到超时返回，如果设置 `Wait=False`, 会在获取不到 token 时直接返回 `ErrPoolLimit`。

```go
func (p *ConnectionPool) getToken(ctx context.Context) error {
  if p.MaxActive <= 0 {
    return nil
  }
    
  if p.Wait {
    select {
    case p.token <- struct{}{}:
      return nil
    case <-ctx.Done():
      return ctx.Err()
    }
  } else {
    select {
    case p.token <- struct{}{}:
      return nil
    default:
      return ErrPoolLimit
    }
  }
}

func (p *ConnectionPool) freeToken() {
  if p.MaxActive <= 0 {
    return
  }
  <-p.token
}
```

成功获取 token 后，优先从 idle list 中获取空闲连接，如果失败则新创建连接返回。

### 初始化 ConnectionPool

在 Get 时要进行 ConnectionPool 的初始化，主要分为启动检查协程和根据 MinIdle 预热空闲连接。

#### KeepMinIdles

业务的突发流量可能会导致大量新连接建立，创建连接是一个比较耗时的操作，可能导致请求超时。提前创建部分空闲连接可以起到预热效果。连接池在创建时创建 MinIdle 个连接备用。

#### 检查协程

ConnectionPool 周期性的进行以下检查：

- 空闲连接健康检查
  默认健康检查策略如下图所示，健康检查扫描 idle 链表，如果未通过安全检查则将连接直接关闭，首先检查连接是否正常，然后检查是否到达 IdleTimeout 和 MaxConnLifetime. 可以使用 WithHealthChecker 自定义健康检查策略。
  除周期性的检查空闲连接，在每次从 idle list 获取空闲连接是都会检查，此时将 isFast 设为 true, 只进行连接存活确认：
  ```go
  func (p *ConnectionPool) defaultChecker(pc *PoolConn, isFast bool) bool {
    if pc.isRemoteError(isFast) {
      return false
    }
    if isFast {
      return true
    }
    if p.IdleTimeout > 0 && pc.t.Add(p.IdleTimeout).Before(time.Now()) {
      return false
    }
    if p.MaxConnLifetime > 0 && pc.created.Add(p.MaxConnLifetime).Before(time.Now()) {
      return false
    }
    return true
  }
  ```
  连接池检测连接空闲的时间，通常也要做成可配置化的，目的是为了与 server 端配合（尤其要考虑不同框架的场景），如果配合的不好，也会出问题。比如 pool 空闲连接检测时间是 1min，server 也是 1min，可能会存在这样的情景，就是 server 端密集关闭空闲连接的时候，client 端还没检测到，发送数据的时候发现大量失败，而不得不通过上层重试解决。比较好的做法是，server 空闲连接检测时长设置为 pool 空闲连接检测时长大一些，尽量让 client 端主动关闭连接，避免取出的连接被 server 关闭而不自知。
  
  > 这里其实也有种优化的思路，就是在每次取出一个连接的时候，通过系统调用非阻塞 read 一下，其实是可以判断出连接是否已经对端关闭的，在 Unix/Linux 平台下可用，但是在 windows 平台下遇到点问题，所以 tRPC-Go 中删除了这一个优化点。

- 空闲连接数量检查
  同 KeepMinIdles, 周期性的将空闲连接数补充到 MinIdle 个。
- ConnectionPool 空闲检查
  transport 不会主动关闭 ConnectionPool, 会导致后台检查协程空转。通过设置 poolIdleTimeout, 周期性检查在此时间内用户使用连接数为 0, 来保证长时间未使用的 ConnectionPool 自动关闭。

## 连接的生命周期

MinIdle 是 ConnectionPool 维持的最小空闲连接，在初始化和周期检查中进行补充。
用户获取连接时，首先从空闲连接中获取，若没有空闲连接才会重新创建。当用户完成请求后，将连接归还给 ConnectionPool, 此时有三种可能：
- 当空闲连接超过 MaxIdle 时，根据淘汰策略关闭一个空闲连接；
- 当连接池的 forceClose 设置为 true 时，不归还 ConnectionPool, 直接关闭；
- 加入空闲连接链表。

用户使用连接发生读写错误时，将直接关闭连接。检查连接存活失败后，也会直接关闭：
![life_cycle](/.resources/pool/connpool/life_cycle.png)

## 空闲连接管理策略

连接池有 FIFO 和 LIFO 两种策略进行空闲连接的选择和淘汰，通过 PushIdleConnToTail 控制，应该根据业务的实际特点选择合适的管理策略。

- fifo，保证各个连接均匀使用，但是当调用方请求频率不高，但是恰巧每次能在连接空闲条件命中之前来一个请求，就会导致各个连接无法被释放，此时维持这么多的连接数是多余的。
- lifo, 优先采用栈顶连接，栈底连接不频繁使用会优先淘汰。

```go
func (p *ConnectionPool) addIdleConn(ctx context.Context) error {
  c, _ := p.dial(ctx)
  pc := p.newPoolConn(c)
  if !p.PushIdleConnToTail {
    p.idle.pushHead(pc)
  } else {
    p.idle.pushTail(pc)
  }
}

func (p *ConnectionPool) getIdleConn() *PoolConn {
  for p.idle.head != nil {
    pc := p.idle.head
    p.idle.popHead()
    // ...
  }
}

func (p *ConnectionPool) put(pc *PoolConn, forceClose bool) error {
  if !p.closed && !forceClose {
    if !p.PushIdleConnToTail {
      p.idle.pushHead(pc)
    } else {
      p.idle.pushTail(pc)
    }
    if p.idleSize >= p.MaxIdle {
      pc = p.idle.tail
      p.idle.popTail()
    }
  }
}
```
