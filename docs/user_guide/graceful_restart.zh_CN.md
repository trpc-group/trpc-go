[English](graceful_restart.md) | 中文

# 前言

tRPC-Go 框架支持服务优雅重启（热重启），在重启期间，不中断老进程已建立的连接、保证已接受的请求正确处理（包括消费者服务），同时新进程允许建立新的连接并处理连接请求。

# 原理

tRPC-Go 实现热重启的原理大致总结如下：
- 服务启动的时候监听信号 SIGUSR2 作为热重启信号（可自定义）；
- 服务进程在启动的时候，会将启动的每个 servertransport 的 listenfd 记录下来；
- 当服务进程接收到 SIGUSR2 信号之后，开始执行热重启逻辑；
- 服务进程通过 ForkExec 来创建一个进程副本（不同于 fork），并通过 ProcAttr 传递 listenfd 给子进程，同时通过环境变量来告知子进程当前是热重启模式，也通过环境变量来传递 listenfd 的起始值以及数量；
- 子进程正常启动，在实例化 servertransport 的时候稍有特殊，会检查当前是否是热重启模式，如果是则继承父进程 listenfd 并重建 listener，反之则走正常启动逻辑；
- 子进程启动之后，立马提供正常服务，父进程得知子进程创建成功之后，择一合适时机退出，如 shutdown tcpconn read、停止 consumer 消费消息、service 上请求都已经正常处理之后再退出；

# 实现代码

参考 `server/serve_unix.go` 中涉及 `DefaultServerGracefulSIG` 变量处理的部分

# 使用示例

## 热重启触发方法

当启动 server 后，首先使用 `ps -ef | grep server_name` 来获取进程的 pid 信息，然后通过 kill 命令向该进程发送 USR2 信号：

```bash
$ kill -SIGUSR2 pid
```

上述命令即可完成进程的热重启操作

## 使用自定义 signal

热重启信号默认为 SIGUSR2，用户可以在 `s.Serve` 之前修改这个默认值，比如：

```go
import "trpc.group/trpc-go/trpc-go/server"

func main() {
    server.DefaultServerGracefulSIG = syscall.SIGUSR1
}
```

## 注册 shutdown hooks

用户可以注册进程结束后需要执行的 hook，以保证资源的及时清理，用法如下：

```go
import (
  trpc "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/server"
)

func main() {
    s := trpc.NewServer()
    s.RegisterOnShutdown(func() { /* Your logic. */ })
    // ...
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
