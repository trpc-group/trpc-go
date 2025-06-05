## 前言

tRPC-Go 框架支持服务的优雅退出，即在接收到退出信号后，能够在指定的超时时间内平滑地关闭所有服务，确保资源的正确释放和服务的顺利退出。在超时时间内，允许当前正在处理的请求继续执行，并阻止新的连接和请求的到达，避免请求中断和数据丢失。在关闭服务的过程中，释放所有相关资源，确保系统资源得到正确管理。在正确配置插件的情况下，通知并更新相关组件的状态。例如，如果使用了北极星服务注册，框架会在服务退出过程中注销北极星上的服务。

## 原理

tRPC-Go 实现优雅退出的原理大致总结如下：

- 信号监听：服务启动时监听 `SIGINT`、`SIGTERM` 和 `SIGSEGV` 信号。与优雅重启不同，优雅退出不能自定义退出信号；
- 接收信号并执行退出逻辑：当服务进程接收到上述信号之一时，开始执行优雅退出逻辑：
  - 遍历并关闭服务：遍历所有 `service`，为每个 `service` 启动一个 `goroutine` 进行关闭操作；
  - 如果服务实现了 `causeCloser` 接口，调用 `CloseCause` 方法，否则调用 `Close` 方法；
  - 取消各个 `service` 在名字服务中的注册；
  - 调用各个插件的 `close` 方法；
  - 停止监听新的连接和请求；
  - 等待 `close_wait_time` 的时间，确保完成退出；
- 如果过程中超过了 `service` 的最大关闭时间，将提前退出，并记录服务退出失败；
- 另外，在优雅重启的过程中，包含了优雅退出的过程，保障了旧服务不会再被使用。

## 实现代码

关于优雅退出和优雅重启的代码实现可以参考 `https://git.woa.com/trpc-go/trpc-go/blob/master/server/serve_unix.go` 。其中涉及 `DefaultServerCloseSIG` 变量处理的部分为优雅退出的逻辑，调用的 `s.tryclose()` 包含了退出的具体实现。需要注意，`DefaultServerGracefulSIG` 相关的部分为优雅重启的逻辑。

## 使用示例

### 配置

请在 `trpc_go.yaml` 文件中确保添加了 `close_wait_time` 和 `max_close_wait_time` 这两项配置：

```yaml
server:  # 服务器配置项
  close_wait_time: 1000  # 关闭服务器时的最短等待时间（以毫秒为单位），以便完成服务注销，框架版本 v0.18.3 之后 1000 为默认值。
  max_close_wait_time: 2000  # 关闭服务器时的最长等待时间（以毫秒为单位），以便完成所有请求的处理，框架版本 v0.18.3 之后 2000 为默认值。
```

为了最优配置效果，我们建议：

- `close_wait_time` 设置为处理请求的可能最大耗时（比如 P999），如果对服务器处理请求的最大耗时不太确定，建议将 `close_wait_time` 设置为 `1000` 毫秒（即 1 秒）。
- 将 `max_close_wait_time` 设置为 `close_wait_time` 的两倍。

注意，因为优雅重启过程中包含优雅退出的逻辑，所以两者都会收到这个配置的影响。

### 优雅退出触发方法

当启动 server 后，首先使用 `ps -ef | grep server_name` 来获取进程的 pid 信息，然后通过 kill 命令向该进程发送 SIGTERM 信号：

```bash
kill -SIGTERM pid
```

或者使用

```bash
pkill -SIGTERM service-name
```

都可以触发优雅退出。

### 注册 onShutdownHooks

`onShutdownHooks` 是 tRPC-Go 服务器中的一个机制，用于在服务器启动关闭之后且所有 `service` 执行关闭之前执行一组预定义的钩子函数。这些钩子函数可以用于执行一些清理操作、资源释放、日志记录等任务，以确保服务器能够优雅地退出。用法如下：

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

### 插件实现 Closer interface

假如插件需要在进程结束时进行资源清理操作，可以额外实现 `plugin.Closer` interface，提供 `Close` 方法，这个方法在进程结束时会被框架内部自动调用（调用的顺序为插件 Setup 的逆序）：

```go
type closablePlugin struct{}

// Type 和 Setup 是 plugin 需要实现的基本方法
func (p *closablePlugin) Type() string {...}
func (p *closablePlugin) Setup(name string, dec Decoder) error {...}

// 插件可以选择额外实现一个 Close 方法，用于进程结束后的资源自动清理
func (p *closablePlugin) Close() error {...}
```
