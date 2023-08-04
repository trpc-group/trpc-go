# Health Checking
The process start doesn't mean the service is available, services that require hot reloading at startup may still in the process of initialization many seconds after the process start. Long-running services may eventually enter an inconsistent state, making it impossible to provide services unless restarted. Like K8s readiness and liveness, tRPC also provides a health check function for services.

## Usage

The health check of tRPC-Go is built into the admin module. You just need to enable the admin module in the trpc_go.yaml file.
```yaml
server:
  admin:
    port: 9988 # whatever
```

You can use curl "http://localhost:9988/is_healthy/"(the suffix / in url is required) to check the status of tRPC-Go service. The mapping between HTTP status codes and service status is as follows:

| status code | server status |
| --- | --- |
| 200 | healthy |
| 404 | unknown |
| 503 | unhealthy |


## Set Your Health Check Logic

For most scenarios, as long as the admin's /is_healthy/ works, the whole service is healthy, and users don't need to care about which services are unavailable

For scenarios that require setting the status of a specific service, trpc provides related APIs.

```go
// trpc.go
// GetAdminService gets admin service from server.Server.
func GetAdminService(s *server.Server) (*admin.TrpcAdminServer, error)
// admin/admin.go
// RegisterHealthCheck registers a new service and return two functions, one for unregistering the service and one for
// updating the status of the service.
func (s *TrpcAdminServer) RegisterHealthCheck(serviceName string) (unregister func(), update func(healthcheck.Status), err error)
```

The example server starts two services, start the server by running:
```shell
go run server/main.go -conf server/trpc_go.yaml # start server
```

Run command below to check the server status:
```shell
# attention: the last / in uri is required!
# see http status code in response header
curl -i localhost:9988/is_healthy/
```

Run command below to check the service status
```shell
# access specified service status via /is_healthy/${server.service.name}
curl -i localhost:9988/is_healthy/foo
```




