# Scope

This example shows the scope functionality provided by trpc-go framework. Users can use this feature to call a local-scoped server that is in the same process with the client. In this way, the serialization and network overhead can be skipped.

The client and server are both written in [server/main](server/main.go) to illustrate the usage of scope.

Inside `trpc_go.yaml`, the scope is used for client to decide which server to call:

```yaml
client:  # configuration for client calls.
  scope: "local"  # change to "remote" to compare the performance between "local" and "remote" (you can also choose "all" to first use "local" then fallback to "remote").
  # scope: "remote"
  service:  # configuration for a single backend.
    - name: trpc.test.helloworld.Greeter
      target: ip://127.0.0.1:8000
      # scope: "local"  # per-client service config.
```

* "local": the client can only call the local server.
* "remote": the client can only call the remote server.
* "all": the client can call the local and remote server (first try local, then remote).

The default value is "remote" to keep backward compatibility.

The detailed steps to run the example:

```shell
$ cd examples/features/scope
$ cd server
$ go build .
$ # Use taskset to bind to one core.
$ taskset -c 0 ./server 
2024-10-21 12:36:03.217 DEBUG   maxprocs/maxprocs.go:48 maxprocs: Leaving GOMAXPROCS=1: CPU quota undefined
2024-10-21 12:36:03.217 INFO    server/service.go:203   process: 2005669, trpc service: trpc.test.helloworld.Greeter launch success, tcp: 127.0.0.1:8000, serving ...
2024-10-21 12:36:08.281 INFO    server/main.go:45       QPS: 145369, average cost: 0.01ms
# Press Ctrl+C to exit

$ # Run script to toggle trpc_go.yaml client.scope from "local" to "remote" and run again.
$ ./toggle_scope.sh
YAML configuration toggled in trpc_go.yaml from 'local' to 'remote'
$ taskset -c 0 ./server 
2024-10-21 12:36:18.929 DEBUG   maxprocs/maxprocs.go:48 maxprocs: Leaving GOMAXPROCS=1: CPU quota undefined
2024-10-21 12:36:18.930 INFO    server/service.go:203   process: 2006992, trpc service: trpc.test.helloworld.Greeter launch success, tcp: 127.0.0.1:8000, serving ...
2024-10-21 12:36:36.164 INFO    server/main.go:45       QPS: 21077, average cost: 0.05ms
```

As seen from the log, the QPS can be improved from 21077 to 145369 (↑ 589.7%). (Note: the prevention of serialization and networking contributes largly to the performance gains).
