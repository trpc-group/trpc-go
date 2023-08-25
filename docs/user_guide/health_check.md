# Introduction

When a process starts, the code service may not have finished initializing, such as services that require hot loading during startup.

Long-running services may eventually enter an inconsistent state and be unable to provide services normally to the outside world unless they are restarted.

Similar to K8s [readiness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes) and [liveness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request), tRPC also provides a health check function for services.

# Quick Start

The health check of tRPC-Go is built into the `admin` package and needs to be enabled in `trpc_go.yaml`:
```yaml
server:
  admin:
    port: 11014
```
You can then use `curl "http://localhost:11014/is_healthy/"` to determine the status of the service. The corresponding relationship between HTTP status codes and service status is as follows:

| HTTP status code | Service status |
| :-: | :-: |
| `200` | Healthy |
| `404` | Unknown |
| `503` | Unhealthy |

# Detailed Introduction

In the "Quick Start" section, as long as the `/is_healthy/` of admin is called, the entire service is healthy, and you do not need to care about which services are under the server, which is suitable for most default scenarios. For scenarios that require setting specific service status, we provide an API at the code level:
```go
// trpc.go
// GetAdminService gets admin service from server.Server.
func GetAdminService(s *server.Server) (*admin.TrpcAdminServer, error)

// admin/admin.go
// RegisterHealthCheck registers a new service and return two functions, one for unregistering the service and one for
// updating the status of the service.
func (s *TrpcAdminServer) RegisterHealthCheck(serviceName string) (unregister func(), update func(healthcheck.Status), err error)
```
For example, in the following sample:
```go
func main() {
	s := trpc.NewServer()
	admin, err := trpc.GetAdminService(s)
	if err != nil { panic(err) }
	
	unregisterXxx, updateXxx, err := admin.RegisterHealthCheck("Xxx")
	if err != nil { panic(err) }
	_, updateYyy, err := admin.RegisterHealthCheck("Yyy")
	if err != nil { panic(err) }
	
	// When you no longer care about Xxx and want it to not affect the overall status of the server, you can call unregisterXxx
	// In the implementation of Xxx/Yyy, updateXxx/updateYyy is called to update their health status
	pb.RegisterXxxService(s, newXxxImpl(unregisterXxx, updateXxx))
	pb.RegisterYyyService(s, newYyyImpl(updateYyy))
	pb.RegisterZzzService(s, newZzzImpl()) // We don't care about Zzz
	
	log.Info(s.serve())
}
```
You register three services, but only `Xxx` and `Yyy` have registered health checks. At this time, you can obtain the status of service `Xxx` separately by appending `Xxx` to the URL: `curl "http://localhost:11014/is_healthy/Xxx"`. For the unregistered service `Zzz`, its HTTP status code is `404`.

Because we have registered health checks for `Xxx` and `Yyy`, the status of the entire server (i.e., `curl "http://localhost:11014/is_healthy/"`) will be jointly determined by `Xxx` and `Yyy`. Only when `Xxx` and `Yyy` are both `healthcheck.Serving`, the HTTP status code of the server is `200`. When `Xxx` and `Yyy` are at least one `healthcheck.Unknown` (the default initial state of the service registered using `admin.RegisterHealthCheck`), the HTTP status code of the server is `404`. Otherwise, the HTTP status code of the server is `503`.

In short, you only need to remember that the entire server is `200` only when all registered health check services are `healthcheck.Serving`.

# Cooperate with Polaris heartbeat

`trpc-naming-polaris`(>= v0.3.6)'s heartbeat can cooperate with health check.

For any service that has not explicitly registered for health check, its heartbeat start immediately after server started (same as older version).

For any service that has explicitly registered for health check, only when its status become `healthcheck.Serving`, the first heartbeat starts. If the status changed to `healthcheck.NotServing` or `healthcheck.Unknown`, Polaris heartbeat will be paused until status changed to `healthcheck.Serving` (one heartbeat will be immediately sent upon change).

