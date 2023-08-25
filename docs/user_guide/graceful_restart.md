# Introduction

The tRPC-Go framework supports graceful restart (hot restart). During the restart process, established connections are not interrupted to ensure that accepted requests are handled correctly (including consumer services), while allowing the new process to establish a new connection and handle new requests.

# Implementation

tRPC-Go's implementation of graceful restart can be described as follows:
- The service listens on the SIGUSR2 signal as the graceful restart signal (customizable).
- The service process keeps track of each server transport's listenfd.
- When the process accepts the SIGUSR2 signal, it starts the graceful restart logic.
- The process uses ForkExec (instead of Fork) to create a new process with a new binary and uses ProcAttr to pass listenfd to the child process. The child process checks the environment variable to know if it is a graceful restart and obtains the initial value and amount for listenfd.
- The child process starts normally but checks for graceful restart when initializing server transport. If it is gracefully restarting, it inherits from the parent's listenfd and reconstructs the listener; otherwise, it starts normally.
- After the child process starts, it immediately starts serving. The parent process exits at a suitable time (e.g., shutdown tcpconn read, consumer stopped, service requested all handled) after the child process started successfully.

# Implementation Details

Refer to `server/serve_unix.go` and search for `DefaultServerGracefulSIG` for implementation details.

# Usage Example

## Trigger graceful restart

After the server is started, use `ps -ef | grep server_name` to obtain the pid, then use `kill` to send `SIGUSR2` to the process.

```bash
$ kill -SIGUSR2 pid
```

The command above will trigger graceful restart.

## Use custom signal

By default, the signal for graceful restart is SIGUSR2. You can modify the signal before `s.Serve`, for example:

```go
import "git.code.oa.com/trpc-go/trpc-go/server"

func main() {
	server.DefaultServerGracefulSIG = syscall.SIGUSR1
}
```

## Register shutdown hooks

You can register hooks to run on process exit for resource cleanup, for example:

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

If your plugin needs to do cleanup on process exit, you can implement the `plugin.Closer` interface. The `Close` method will be called by the framework on exit (reverse order of plugins setup):

```go
type closablePlugin struct{}

// Type and Setup are a plugin's required methods
func (p *closablePlugin) Type() string {...}
func (p *closablePlugin) Setup(name string, dec Decoder) error {...}

// Plugins can optionally implement a Close method, for resource cleanup on process exit
func (p *closablePlugin) Close() error {...}
```

# FAQ

