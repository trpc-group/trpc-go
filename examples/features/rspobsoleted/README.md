# RspObsoleted

In some cases, users may want to retrieve the RPC Response struct from the `sync.Pool` and return it to the framework. After the framework uses this struct, it calls the function provided by the user to return the Response struct to the `sync.Pool`.

The framework provides the option `server.WithOnResponseObsoleted` for users to set the release logic that the framework needs to execute after using the struct.

In addition, the Response struct may reference other objects that are also retrieved from the `sync.Pool`, and only a part of the object is referenced. In this case, the user needs to perform the same recycling operation on the object after the Response is returned to the `sync.Pool`.

The framework provides an additional `context.Context` parameter for the function interface to help users complete this function. Users can modify the `msg.CommonMeta` in it to associate the object they want to release with the context of this request, so that they can retrieve the previously placed object from the context in `server.WithOnResponseObsoleted` and perform the corresponding recycling.

[./server/main.go](./server/main.go) provides a complete example for users to refer to. Some users have reported that this optimization can reduce CPU consumption by about 20% in some complex Response struct situations.

## Usage

* Start server.

```shell
go run server/main.go -conf server/trpc_go.yaml 
```

* Start ClientStream client.

```shell
go run client/main.go -conf client/trpc_go.yaml
```
