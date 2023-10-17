## Admin

This example demonstrates the use of admin commands in tRPC.

## Usage

* Start the server
```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

By default, the framework will not enable the admin capability. To start the admin, you only need to add the admin configuration to the trpc-go configuration file, as shown below.
```yaml
server:
  admin:
    ip: 127.0.0.1                 # the admin listening ip, which can also be configured through network interface card (NIC) settings.
    port: 11014                   # the admin listening port.
    read_timeout: 3000            # maximum time when a request is accepted and the request information is fully read, to prevent slow clients, in milliseconds.
    write_timeout: 60000          # maximum processing time in milliseconds.
```

Registers routes for custom admin commands.
```golang
func testCmds(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("test cmds"))
}

func init() {
// Register custom handler.
	admin.HandleFunc("/testCmds", testCmds)
}
```

* View all admin commands.
```shell
curl http://127.0.0.1:11014/cmds
```

* Trigger execution of custom commands.
```shell
# execute testCmds.
curl http://127.0.0.1:11014/testCmds
```

The client log will be displayed as follows:
```
{"cmds":["/cmds","/version","/cmds/rpcz/spans","/cmds/rpcz/spans/","/debug/pprof/profile","/debug/pprof/symbol","/testCmds","/cmds/loglevel","/cmds/config","/is_healthy/","/debug/pprof/","/debug/pprof/cmdline","/debug/pprof/trace"],"errorcode":0,"message":""}
test cmds%
```

## Explanation

The admin has already integrated the pprof capability by default:

- If admin is enabled, the framework has integrated the http/pprof functionality by default. Do not register again using admin.
- If admin is enabled on the PCG 123 platform, you can view the flame graph on the platform. For more information, please refer to the [tRPC-Go admin commands](/admin/README.md).
