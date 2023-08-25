[TOC]

# 前言

进程启动并不代码服务已经初始化完成，比如需要在启动时进行热加载的服务。
长时间运行的服务，最终可能进入不一致状态，无法对外正常提供服务，除非重启。
类似于 K8s [readiness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes) 和 [liveness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request)，tRPC 也提供了服务的健康检查功能。

# 快速上手

tRPC-Go 的健康检查内置于 `admin` 模块；需在 `trpc_go.yaml` 中开启：
```yaml
server:
  admin:
    port: 11014
```
就可以通过 `curl "http://localhost:11014/is_healthy/"` 来判断服务的状态。HTTP 状态码与服务状态的对应关系如下：

| HTTP 状态码 | 服务状态 |
| :-: | :-: |
| `200` | 健康 |
| `404` | 未知 |
| `503` | 不健康 |

# 详细介绍

「快速上手」一节，我们认为只要 admin 的 `/is_healthy/` 调通，整个服务就是健康的，用户不用关心 server 下面有哪些 service，这适用于大部分默认场景。
对于需要设置特定 service 状态的场景，我们在代码层面提供了 API：
```go
// trpc.go
// GetAdminService gets admin service from server.Server.
func GetAdminService(s *server.Server) (*admin.TrpcAdminServer, error)

// admin/admin.go
// RegisterHealthCheck registers a new service and return two functions, one for unregistering the service and one for
// updating the status of the service.
func (s *TrpcAdminServer) RegisterHealthCheck(serviceName string) (unregister func(), update func(healthcheck.Status), err error)
```
比如，在下面的例子中，
```go
func main() {
	s := trpc.NewServer()
	admin, err := trpc.GetAdminService(s)
	if err != nil { panic(err) }
	
	unregisterXxx, updateXxx, err := admin.RegisterHealthCheck("Xxx")
	if err != nil { panic(err) }
	_, updateYyy, err := admin.RegisterHealthCheck("Yyy")
	if err != nil { panic(err) }
	
	// 当你不再关心 Xxx，希望它不影响整个 server 的状态时，可以调用 unregisterXxx
	// 在 Xxx/Yyy 的实现中，通过调用 updateXxx/updateYyy 更新它们的健康状态
	pb.RegisterXxxService(s, newXxxImpl(unregisterXxx, updateXxx))
	pb.RegisterYyyService(s, newYyyImpl(updateYyy))
	pb.RegisterZzzService(s, newZzzImpl()) // 我们不关心 Zzz
	
	log.Info(s.serve())
}
```
用户有三个 service，但只为 `Xxx` 和 `Yyy` 注册了健康检查。这时用户可以单独获取 service `Xxx` 的状态，通过在 url 后追加 `Xxx` 即可：`curl "http://localhost:11014/is_healthy/Xxx"`。对于未注册的 service `Zzz`，其 HTTP 状态码为 `404`。

因为我们为 `Xxx` 和 `Yyy` 注册了健康检查，整个 server 的状态（即 `curl "http://localhost:11014/is_healthy/"`）将由 `Xxx` 和 `Yyy` 共同决定。只有当 `Xxx` 和 `Yyy` 都是 `healthcheck.Serving` 时，server 的 HTTP 状态码才是 `200`。当 `Xxx` 和 `Yyy` 至少有一个是 `healthcheck.Unknown`（这是使用 `admin.RegisterHealthCheck` 注册的 service 的默认初始状态）时，server 的 HTTP 状态码为 `404`。否则，server 的 HTTP 状态码为 `503`。

简单地说，你只需要记住，只有当所有注册了健康检查的 service 都为 `healthcheck.Serving` 时，整个 server 才是 `200`。

# 和北极星心跳上报配合

`trpc-naming-polaris`(>= v0.3.6) 的心跳上报可以和健康检查配合。

对于未显式注册健康检查的 service，其心跳会在 server 启动后立刻开始，和旧版本北极星行为是一致的。

对于显式注册了健康检查的 service，只有当 service 状态变为 `healthcheck.Serving` 时，才会开始第一次心跳上报。服务运行中，如果 service 状态变为 `healthcheck.NotServing` 或者 `healthcheck.Unknown`，就会停止心跳，直到再次变更为 `healthcheck.Serving` 才恢复（变更的瞬间，会立即发起一次心跳上报）。

