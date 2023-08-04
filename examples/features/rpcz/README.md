# RPCZ
RPCZ is a tool that monitors the running state of RPC, recording various things that happen in a rpc, such as serialization/deserialization, compression/decompression, and the execution of filter, which can be applied to debug and performance optimization.

Here, this example will show how you can use RPCZ.
## Usage
The configuration of rpcz is divided into basic configuration, advanced configuration, and code configuration.


### 1. Use the Basic configuration.
Basic configuration allows you to configure sampling for all spans.
* Start Service
```shell
$ go run server/main.go -conf server/trpc_go.yaml  -type Basic
```

* Start rpc client with another terminal.
```shell
$ go run client/main.go -conf client/trpc_go.yaml
```
* Query the summary information of multiple recently submitted spans, you can access the following URL, where xxx is the desired number of queries:
```shell
$ curl http://ip:port/cmds/rpcz/spans?num=xxx
```
* For example, executing curl http://ip:port/cmds/rpcz/spans?num=1 will return summary information for the following 1 spans:
```shell
$ curl http://127.0.0.1:9528/cmds/rpcz/spans?num=1
1:
  span: (server, 6748512057923401418)
    time: (Jun 20 09:58:58.827827, Jun 20 09:58:58.828227)
    duration: (0, 399.819µs, 0)
    attributes: (RequestSize, 109),(ResponseSize, 29),(RPCName, /trpc.examples.rpcz.RPCZ/Hello),(Error, success)
```
* Query the detailed information of a specific span, you can access the following URL, where xxx is the span_id obtained from the summary information:
```shell
$ curl http://ip:port/cmds/rpcz/spans/{xxx}
```
* For example, executing curl http://ip:port/cmds/rpcz/spans/6748512057923401418 can be used to query the detailed information of the span with an id of 6748512057923401418.
```shell
$ curl http://127.0.0.1:9528/cmds/rpcz/spans/6748512057923401418
span: (server, 6748512057923401418)
  time: (Jun 20 09:58:58.827827, Jun 20 09:58:58.828227)
  duration: (0, 399.819µs, 0)
  attributes: (RequestSize, 109),(ResponseSize, 29),(RPCName, /trpc.examples.rpcz.RPCZ/Hello),(Error, success)
  span: (DecodeProtocolHead, 6748512057923401418)
    time: (Jun 20 09:58:58.827831, Jun 20 09:58:58.828048)
    duration: (4.091µs, 217.003µs, 178.725µs)
  span: (Decompress, 6748512057923401418)
    time: (Jun 20 09:58:58.828056, Jun 20 09:58:58.828058)
    duration: (229.125µs, 1.259µs, 169.435µs)
  span: (Unmarshal, 6748512057923401418)
    time: (Jun 20 09:58:58.828058, Jun 20 09:58:58.828085)
    duration: (230.73µs, 26.735µs, 142.354µs)
  span: (unknown, 6748512057923401418)
    time: (Jun 20 09:58:58.828087, Jun 20 09:58:58.828155)
    duration: (260.121µs, 67.786µs, 71.912µs)
    span: (HandleFunc, 6748512057923401418)
      time: (Jun 20 09:58:58.828088, Jun 20 09:58:58.828155)
      duration: (970ns, 66.571µs, 245ns)
  span: (Marshal, 6748512057923401418)
    time: (Jun 20 09:58:58.828157, Jun 20 09:58:58.828171)
    duration: (329.61µs, 14.565µs, 55.644µs)
  span: (Compress, 6748512057923401418)
    time: (Jun 20 09:58:58.828172, Jun 20 09:58:58.828172)
    duration: (344.576µs, 258ns, 54.985µs)
  span: (EncodeProtocolHead, 6748512057923401418)
    time: (Jun 20 09:58:58.828172, Jun 20 09:58:58.828215)
    duration: (345.109µs, 42.756µs, 11.954µs)
  span: (SendMessage, 6748512057923401418)
    time: (Jun 20 09:58:58.828216, Jun 20 09:58:58.828227)
    duration: (388.843µs, 10.481µs, 495ns)

```

### 2. Use advanced configuration.
Advanced configuration allows you to sample spans of interest.
* Start Service
```shell
$ go run server/main.go -conf server/trpc_go_rpcz_error.yaml  -type Advanced
```
* Start rpc client with another terminal.
```shell
$ go run client/main.go -conf client/trpc_go.yaml
```
* Query the summary information of multiple recently submitted spans.
```shell
$ curl http://127.0.0.1:9528/cmds/rpcz/spans?num=1
1:
  span: (server, 2465491400181540343)
    time: (Jun 20 09:42:12.060000, Jun 20 09:42:12.060176)
    duration: (0, 176.255µs, 0)
    attributes: (RequestSize, 111),(ResponseSize, 31),(RPCName, /trpc.examples.rpcz.RPCZ/Hello),(Error, type:business, code:21, msg:error msg)
```

### 3. Use code configuration.
Code configuration allows you to perform dynamic sampling on spans.
* Start Service
```shell
$ go run server/main.go -conf server/trpc_go.yaml  -type Code
```
* Start rpc client with another terminal.
```shell
$ go run client/main.go -conf client/trpc_go.yaml
```
* Query the summary information of multiple recently submitted spans.
```shell
$ curl http://127.0.0.1:9528/cmds/rpcz/spans?num=1
1:
  span: (server, 2054474867293077231)
    time: (Jun 20 10:03:48.290265, Jun 20 10:03:48.290710)
    duration: (0, 444.617µs, 0)
    attributes: (RequestSize, 109),(ResponseSize, 45),(RPCName, /trpc.examples.rpcz.RPCZ/Hello),(Error, success)
```