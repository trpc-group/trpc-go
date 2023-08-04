# Plugin

Plugins are the bridge that connects the framework core and external service governance components. On one hand, plugins need to implement the plugin according to the framework standard interface, register it to the framework core, and complete the plugin instantiation. On the other hand, plugins need to call the SDK/API of the external service governance service to implement service governance functions such as service discovery, load balancing, monitoring, and call chains.

## Usage

- plugin import

    ```go
    import (
    	_ "trpc.group/trpc-go/trpc-go/examples/features/plugin"
    )
    ```

- plugin config

  Plugin config in yaml file, example:

  ```yaml
  custom:
    custom:
      test: test
      test_obj:
        key1: value1
        key2: false
        key3: 1234
  ```

* Start server.

```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Start client.

```shell
$ go run client/main.go -conf client/trpc_go.yaml
```

* Server output

```
2023-05-10 11:20:13.046 INFO    plugin/custom_plugin.go:48      [plugin] init customPlugin success, config: {test {value1 false 1234}}
2023-05-10 11:20:13.047 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=16: CPU quota undefined
2023-05-10 11:20:13.047 INFO    server/service.go:164   process:25080, trpc service:trpc.test.helloworld.Greeter launch success, tcp:127.0.0.1:9091, serving ...
2023-05-10 11:20:20.307 INFO    server/main.go:31       [Plugin] trpc-go-server SayHello, req.msg:client
2023-05-10 11:20:20.307 INFO    plugin/custom_plugin.go:55      [plugin] call key1 : value1, key2 : false, key3 : 1234
```
  
  


