# Precool

This example demonstrates how to register a precool check for a service and how
to query the precool status through the admin API.

## Usage

* Start server.

```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

The server simulates three startup checks:

1. database connection
2. cache warmup
3. configuration loading

* Check process-level precool status.

```shell
$ curl http://127.0.0.1:11014/cmds/is_precool/
```

* Check service-level precool status.

```shell
$ curl http://127.0.0.1:11014/cmds/is_precool/trpc.examples.precool.Precool
```

Possible `is_precool` values are:

* `proc_success`: precool completed successfully
* `proc_failure`: precool failed
* `proc_ongoing`: precool is still in progress
* `unknown`: service not registered or status unknown

## Explanation

The example registers a service-level precool strategy with
`RegisterServicePrecool`. The strategy returns:

* `precool.Failure` before the first required dependency is ready
* `precool.Ongoing` while later startup checks are still running
* `precool.Success` when all startup checks have completed

This lets external readiness probes distinguish between failed startup, startup
still in progress, and fully ready services.
