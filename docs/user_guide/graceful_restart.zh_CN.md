## 前言

tRPC-Go 框架支持服务优雅重启（热重启），在重启期间，不中断老进程已建立的连接、保证已接受的请求正确处理（包括消费者服务），同时新进程允许建立新的连接并处理连接请求。
在 v0.17.0 之后，tRPC-Go 的 trpc 服务使用 socketPair 方式进行优雅重启，在 v0.19.0 之后，tRPC-Go 的泛 http 服务使用 socketPair 方式进行优雅重启。

## 原理

【旧版】基于 envTrans 的 tRPC-Go 实现热重启的原理大致总结如下：

- 服务启动的时候监听信号 SIGUSR2 作为热重启信号（可自定义）；
- 服务进程在启动的时候，会将启动的每个 serverTransport 的 listenfd 记录下来；
- 当服务进程接收到 SIGUSR2 信号之后，开始执行热重启逻辑；
- 服务进程通过 ForkExec 来创建一个进程副本（不同于 fork），并通过 ProcAttr 传递 listenfd 给子进程，同时通过环境变量来告知子进程当前是热重启模式，也通过环境变量来传递 listenfd 的起始值以及数量；
- 子进程正常启动，在实例化 serverTransport 的时候稍有特殊，会检查当前是否是热重启模式，如果是则继承父进程 listenfd 并重建 listener，反之则走正常启动逻辑；
- 子进程启动之后，立马提供正常服务，父进程得知子进程创建成功之后，择一合适时机退出，如 shutdown tcpconn read、停止 consumer 消费消息、service 上请求都已经正常处理之后再退出；

【新版】基于 socketPair 的 tRPC-Go 实现[热重启](https://git.woa.com/trpc-go/trpc-go/blob/master/internal/graceful/internal/graceful_restart.go)的原理大致总结如下，基于 v0.19.0：

![socketPair_gracefulRestart](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/graceful_restart/socketPair_gracefulRestart.png)

## 实现代码

参考 `https://git.woa.com/trpc-go/trpc-go/blob/master/server/serve_unix.go` 中涉及 `DefaultServerGracefulSIG` 变量处理的部分

参考 [优雅重启](https://git.woa.com/trpc-go/trpc-go/blob/master/internal/graceful/internal/graceful_restart.go)

**注意**：不要混淆，优雅关闭对应实现为 `server.DefaultServerCloseSIG` 的部分。

## 使用示例

### 配置更新

请在 `trpc_go.yaml` 文件中确保添加了 `close_wait_time` 和 `max_close_wait_time` 这两项配置：

```yaml
server:  # 服务器配置项
  read_timeout: 1000  # 读取请求最长处理时间 单位 毫秒
  close_wait_time: 1000  # 关闭服务器时的最短等待时间（以毫秒为单位），以便完成服务注销，框架版本 v0.18.3 之后 1000 为默认值。
  max_close_wait_time: 2000  # 关闭服务器时的最长等待时间（以毫秒为单位），以便完成所有请求的处理，框架版本 v0.18.3 之后 2000 为默认值。
```

为了最优配置效果，我们建议：

- `close_wait_time` 设置为处理请求的可能最大耗时（比如 P999），如果对服务器处理请求的最大耗时不太确定，建议将 `close_wait_time` 设置为 `1000` 毫秒（即 1 秒）。
- 将 `max_close_wait_time` 设置为 `close_wait_time` 的两倍。
- 需要将 `read_timeout` 显式配置出来，建议为 `close_wait_time`，其默认值和 idletime（默认值为 60s） 相同，主要是为了避免大包场景下，读取请求超时，因此此处将 `read_timeout` 调小时在大包或者通信较慢的场景下有风险，服务端读取了一半的包之后触发读超时，然后直接关闭连接，导致客户端收到 171(RetClientReadFrameErr),141(RetClientNetErr) 等错误。

推荐使用 trpc-go 框架的版本 >= v0.18.1，该版本在优雅重启方面进行了全面的优化和完善。

### 热重启触发方法

当启动 server 后，首先使用 `ps -ef | grep server_name` 来获取进程的 pid 信息，然后通过 kill 命令向该进程发送 USR2 信号：

```bash
kill -SIGUSR2 pid
```

上述命令即可完成进程的热重启操作

## 使用自定义 signal

热重启信号默认为 SIGUSR2，用户可以在 `s.Serve` 之前修改这个默认值，比如：

```go
import "git.code.oa.com/trpc-go/trpc-go/server"

func main() {
    server.DefaultServerGracefulSIG = syscall.SIGUSR1
}
```

**注意**：优雅关闭对应的变量为 `server.DefaultServerCloseSIG`，默认已经包含了 `unix.SIGINT, unix.SIGTERM, unix.SIGSEGV`。

### 注册 shutdown hooks

用户可以注册进程结束后需要执行的 hook，此类 hooks 的执行时机是在服务器启动关闭之后且所有 `service` 执行关闭之前，以保证资源的及时清理，用法如下：

```go
import (
    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/server"
)

func main() {
    s := trpc.NewServer()
    s.RegisterOnShutdown(func() { /* Your logic. */ })
    // ...
}
```

### 注册 before graceful restart hooks(版本 v0.19.0)

用户可以注册优雅重启前需要执行的 hook，此类 hooks 的执行时机是在新进程启动前，以保证新进程成功启动，用法如下：

```go
import (
    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/server"
)

func main(){
    s := trpc.NewServer()
    s.RegisterBeforeGracefulRestart(func(){ /* Your logic. */})
}
```

假如插件需要在进程结束时进行资源清理操作，可以额外实现 `plugin.Closer` interface，提供 `Close` 方法，这个方法在进程结束时会被框架内部自动调用（调用的顺序为插件 Setup 的逆序）：

```go
type closablePlugin struct{}

// Type 和 Setup 是 plugin 需要实现的基本方法
func (p *closablePlugin) Type() string {...}
func (p *closablePlugin) Setup(name string, dec Decoder) error {...}

// 插件可以选择额外实现一个 Close 方法，用于进程结束后的资源自动清理
func (p *closablePlugin) Close() error {...}
```

### 支持关闭优雅重启 (版本 v0.20.0)

用户可以随时关闭或者开启优雅重启，[需求来源](https://git.woa.com/trpc-go/trpc-go/issues/1015)，用法如下：

1. 通过配置文件关闭或者开启：

    ```yaml
    global:
      disable_graceful_restart: true # 关闭优雅重启
      # disable_graceful_restart: false # 默认，开启优雅重启
    ```

2. 通过代码关闭或者开启：

```go
import (
    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/server"
)

func main(){
    s := trpc.NewServer()
    s.SetDisableGracefulRestart(true) // 关闭优雅重启，注意，此操作不会将 gracefulRestartHooks 清空
    s.SetDisableGracefulRestart(false) // 默认，开启优雅重启
}
```

### 123 平台使用

当前 123 平台的重启是默认调用 stop.sh 脚本发送 kill 信号，因此用户需要自定义 stop.sh 脚本，将发送 kill 信号改成发送 USR2 信号，具体自定义 stop.sh 脚本可以参考：
[123 平台 iwiki 文档 - 如何自定义监控脚本](https://iwiki.woa.com/p/1628324670#51-%E5%A6%82%E4%BD%95%E8%87%AA%E5%AE%9A%E4%B9%89%E7%9B%91%E6%8E%A7%E8%84%9A%E6%9C%AC%EF%BC%9F)

## FAQ

**Q1：trpc-go 现在是否已经支持了上述提及的热重启能力**

A1: 已支持

---

**Q2：重启期间，老进程是否能正常处理请求**

A2: trpc-go v0.10.0 已支持

---

**Q3：消费者服务，是否支持上述的已接受请求处理完退出**

A3: trpc-go v0.10.0 后已支持

---
**Q4：http 服务是否支持热重启**

A4: 已支持

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
