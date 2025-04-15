English | [中文](flatbuffers.zh_CN.md)

# Background

[flatbuffers](https://flatbuffers.dev/) introduction: A serialization library launched by Google, mainly used in gaming and mobile scenarios. Its function is similar to Protobuf, and its main advantages are:

- Access to serialized data is incredibly fast (after serialization, data can be accessed without deserialization; Unmarshal only extracts the byte slice, and accessing fields is similar to virtual table mechanism i.e., looking up the offset and locating the data). In fact, both Marshal and Unmarshal operations of Flatbuffers are lightweight, and the actual serialization step is deferred until construction. Therefore, construction takes up a large proportion of the total time.
- Since it can access fields without deserialization, it is suitable for cases where only a small number of fields need to be accessed, such as when only a few fields of a large message are required. Protobuf needs to deserialize the entire message to access these fields, while Flatbuffers does not.
- Efficient use of memory, without frequent memory allocation: This is mainly compared with protobuf. When protobuf serializes and deserializes, it needs to allocate memory to store intermediate results. However, after initial construction, Flatbuffers does not need to allocate memory again during serialization and deserialization.
- Performance tests have also shown that Flatbuffers outperforms Protobuf when dealing with large amounts of data.

Summary:
Pushing all operations to the construction stage makes the Marshal and Unmarshal operations very lightweight.  

According to benchmark tests, the proportion of time taken is:  
- For Protobuf, the construction stage accounts for about 20% (including construction, Marshal, and Unmarshal).  
- For Flatbuffers, it accounts for 90%.  

Drawbacks:

- Modifying a constructed Flatbuffer is more cumbersome.
- The API for constructing data is somewhat difficult to use.

# Principle

![flatbuffers](/.resources_without_git_lfs/user_guide/server/flatbuffers/flatbuffers.png)

# Example
Firstly, install the latest version of the [trpc-cmdline](https://github.com/trpc-group/trpc-cmdline) tool.

Next, use the tool to generate flatbuffers corresponding stub code, which currently supports single-send and single-receive, server/client streaming, bidirectional streaming, etc.

We will walk through all the steps using a simple example.

First, define an IDL file. Its syntax can be learned from the flatbuffers official website. The overall structure is very similar to protobuf. An example is shown below:

```idl
namespace trpc.testapp.greeter; // Equivalent to the "package" in protobuf.

// Equivalent to the "go_package" statement in protobuf.
// Note: "attribute" is a standard syntax in flatbuffers, and the "go_package=xxx" syntax is a custom support implemented by trpc-cmdline.
attribute "go_package=github.com/trpcprotocol/testapp/greeter";

table HelloReply { // "table" is equivalent to "message" in protobuf.
  Message:string;
}

table HelloRequest {
  Message:string;
}

rpc_service Greeter {
  SayHello(HelloRequest):HelloReply; // Single-send and single-receive.
  SayHelloStreamClient(HelloRequest):HelloReply (streaming: "client"); // Client streaming.
  SayHelloStreamServer(HelloRequest):HelloReply (streaming: "server"); // Server streaming.
  SayHelloStreamBidi(HelloRequest):HelloReply (streaming: "bidi"); // Bidirectional streaming.
}

// Example with two services.
rpc_service Greeter2 {
  SayHello(HelloRequest):HelloReply;
  SayHelloStreamClient(HelloRequest):HelloReply (streaming: "client");
  SayHelloStreamServer(HelloRequest):HelloReply (streaming: "server");
  SayHelloStreamBidi(HelloRequest):HelloReply (streaming: "bidi");
}
```
The meaning of the "go_package" field is similar to the corresponding field in protobuf,see https://developers.google.com/protocol-buffers/docs/reference/go-generated#package

As shown in the above link. Note that in protobuf, the "package" and "go_package" fields are unrelated.

*There is no correlation between the Go import path and the package specifier in the .proto file. The latter is only relevant to the protobuf namespace, while the former is only relevant to the Go namespace.*

However, due to the limitations of flatc, the last segment of the namespace in the flatbuffers IDL file must be the same as the last segment of the "go_package" field. At least, the two bold sections below must be the same:

- namespace trpc.testapp.greeter;
- attribute "go_package=github.com/trpcprotocol/testapp/greeter";

Then, use the following command to generate the corresponding stub code:

```sh
$ trpc create --fbs greeter.fbs -o out-greeter --mod github.com/testapp/testgreeter
```
The "--fbs" option specifies the file name of the flatbuffers file (with relative path), "-o" specifies the output path, "--mod" specifies the package content in the generated go.mod file. If "--mod" is not specified, it will look for the go.mod file in the current directory and use the package content in that file as the "--mod" content, which represents the module path identifier of the server itself. This is different from the "go_package" in the IDL file, which represents the module path identifier of the stub code.

The directory structure of the generated code is as follows:

```sh
├── cmd/client/main.go # Client code.
├── go.mod
├── go.sum
├── greeter_2.go       # Server implementation for the second service.
├── greeter_2_test.go  # Testing for the server of the second service.
├── greeter.go         # Server implementation for the first service.
├── greeter_test.go    # Testing for the server of the first service.
├── main.go            # Service startup code.
├── stub/github.com/trpcprotocol/testapp/greeter # Stub code files.
└── trpc_go.yaml       # Configuration files.
```
In one terminal, compile and run the server:
```sh
$ go build      # Compile
$ ./testgreeter # Run
```
In another terminal, run the client:
```sh
$ go run cmd/client/main.go
```
Then, you can view the messages sent between the two terminals in their respective logs.

The main.go file for starting the server is shown below:
```go
package main
import (
    "flag"
    
    _ "trpc.group/trpc-go/trpc-filter/debuglog"
    _ "trpc.group/trpc-go/trpc-filter/recovery"
    trpc "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/log"
    fb "github.com/trpcprotocol/testapp/greeter"
)
func main() {
    flag.Parse()
    s := trpc.NewServer()
    // If there are multiple services, the service name must be explicitly written as the first parameter, otherwise the stream will have issues.
    fb.RegisterGreeterService(s.Service("trpc.testapp.greeter.Greeter"), &greeterServiceImpl{})
    fb.RegisterGreeter2Service(s.Service("trpc.testapp.greeter.Greeter2"), &greeter2ServiceImpl{})
    if err := s.Serve(); err != nil {
        log.Fatal(err)
    }
}
```
The overall content is basically the same as the generated file for Protobuf, and the only thing to note is that "serverFBBuilderInitialSize" is used to set the initial size of "flatbuffers.NewBuilder" when constructing "rsp" in the server stub code for the service. Its default value is 1024. It is recommended to set the size to exactly the size required to construct all the data in order to achieve optimal performance. However, if the data size is variable, setting this size will become a burden. Therefore, it is recommended to keep the default value of 1024 until it becomes a performance bottleneck.

Here is an example of server-side logic implementation:
```go
func (s *greeterServiceImpl) SayHello(ctx context.Context, req *fb.HelloRequest) (*flatbuffers.Builder, error) {
    // Flatbuffers processing logic for single request and response scenario (for reference only, please modify as needed).
    log.Debugf("Simple server receive %v", req)
    // Replace "Message" with the name of the field you want to operate on.
    v := req.Message() // Get Message field of request.
    var m string
    if v == nil {
        m = "Unknown"
    } else {
        m = string(v)
    }
    // Example of adding a field:
    // Replace "String" in "CreateString" with the type of field you want to operate on.
    // Replace "Message" in "AddMessage" with the name of the field you want to operate on.
    idx := b.CreateString("welcome " + m) // Create a string in Flatbuffers.
    b := &flatbuffers.Builder{}
    fb.HelloReplyStart(b)
    fb.HelloReplyAddMessage(b, idx)
    b.Finish(fb.HelloReplyEnd(b))
    return b, nil
}
```
Here is a detailed explanation of each step in the construction process:
```go
// Import the package containing the stub code.
import fb "github.com/trpcprotocol/testapp/greeter"
// Start by creating a *flatbuffers.Builder.
b := flatbuffers.NewBuilder(0) 
// To populate a field in a struct:
// First create an object of the type that the field represents.
// For example, if the field is of type String, 
// you can call b.CreateString("a string") to create the string.
// This method returns the index of the string in the flatbuffer.
i := b.CreateString("GreeterSayHello")
// To construct a HelloRequest struct:
// Call the XXXXStart method provided in the stub code to indicate the start of constructing this struct.
// The corresponding end method is fb.HelloRequestEnd.
fb.HelloRequestStart(b)
// If the field to be populated is called message, you can call fb.HelloRequestAddMessage(b, i) to construct the message field by passing the builder and the index of the previously constructed string.
// Other fields can be constructed in a similar manner.
fb.HelloRequestAddMessage(b, i)
// Call the XXXEnd method when the struct is complete.
// This method will return the index of the struct in the flatbuffer.
// Then call b.Finish to complete the construction of the flatbuffer.
b.Finish(fb.HelloRequestEnd(b))
```
It's evident that the Flatbuffers construction API is significantly difficult to use, especially when constructing nested structures.

To access a specific field in a received message, simply access it as follows:

```go
req.Message() // Access the message field in req.
```

# 性能对比
![performanceComparison](/.resources_without_git_lfs/user_guide/server/flatbuffers/performanceComparison.png)
The load testing environment consisted of two machines with 8 cores, 2.5 GHz CPU, and 16 GB memory.
- We implemented a client-side loop packet tool that can send packages serialized with either Protobuf or Flatbuffers.
- We fixed the number of goroutines at 500 and tested for 50 seconds each time.
- Each point on the graph represents the mean of three alternating tests of Flatbuffers and Protobuf (no standard deviation is shown because we found that the three values did not differ significantly).
- The horizontal axis represents the number of fields, with each element in the vector treated as a separate field covering all basic field types.
- The left y-axis represents QPS, and the right y-axis represents the p99 latency at different field numbers.
- From this table, it can be seen that when there are no map fields, flatbuffers' performance is better than protobuf when the total number of fields increases.
- The reason flatbuffers' performance is worse when there are fewer fields is that the initial builder in flatbuffers initializes the byte slice size uniformly to 1024, so even with fewer fields, this large space still needs to be allocated (protobuf does not do this), resulting in worse performance. This can be alleviated by adjusting the initial byte slice size beforehand, but this would add some burden to business. Therefore, a uniform initial size of 1024 was set during load testing.

![performanceComparison2](/.resources_without_git_lfs/user_guide/server/flatbuffers/performanceComparison2.png)

- Protobuf has poor performance in map serialization and deserialization, as seen in the graph.
- Since flatbuffers does not have a map type, it uses a vector of key-value pairs as a replacement, with the key-value types consistent with the key-value types in protobuf's map.
- As can be seen, when the number of fields increases, flatbuffers' performance improves more significantly.

![performanceComparison3](/.resources_without_git_lfs/user_guide/server/flatbuffers/performanceComparison3.png)

- From the graph, it can be seen that when the total number of fields is high, flatbuffers' performance is better than protobuf, especially when a map is present.
- The horizontal axis is the number of fields without map. For the line with map, each point corresponds to a larger horizontal axis value.
- These field numbers correspond to packet sizes in ascending order:

| Whether there is a map | Serialization method |  |  |  |  |  |  |
| --- | --- | --- | --- | --- | --- | --- | --- |
| N | flatbuffers | 284 | 708 | 1124 | 1964 | 3644 | 7243 |
| N | protobuf | 167 | 519 | 871 | 1573 | 2973 | 5834 |
| Y | flatbuffers | 292 | 1084 | 1900 | 3540 | 6819 | 13619 |
| Y | protobuf | 167 | 659 | 1171 | 2192 | 4232 | 8494 |


# FAQ
##  Q1: How to generate stub code when other files are included in the .fbs file?

Refer to the following usage examples on https://github.com/trpc-group/trpc-cmdline/tree/main/testcase/flatbuffers:

- 2-multi-fb-same-namespace: Multiple .fbs files with the same namespace are in the same directory (namespace in flatbuffers is equivalent to the package statement in protobuf), and one of the main files includes the other .fbs files.
- 3-multi-fb-diff-namespace: Multiple .fbs files are in the same directory with different namespaces. For example, the main file defining RPC references types in different namespaces.
- 4.1-multi-fb-same-namespace-diff-dir: Multiple .fbs files have the same namespace but are in different directories. The main file helloworld.fbs uses a relative path when including other files, and --fbsdir is not needed to specify the search path, as shown in run4.1.sh.
- 4.2-multi-fb-same-namespace-diff-dir: Except that the include statement in helloworld.fbs uses only the file name, everything else is the same as 4.1. To run this example correctly, specify the search path using --fbsdir, as shown in run4.2.sh:
  ```sh
  trpc create --fbsdir testcase/flatbuffers/4.2-multi-fb-same-namespace-diff-dir/request \
            --fbsdir testcase/flatbuffers/4.2-multi-fb-same-namespace-diff-dir/response \
            --fbs testcase/flatbuffers/4.2-multi-fb-same-namespace-diff-dir/helloworld.fbs \
            -o out-4-2 \
            --mod github.com/testapp/testserver42
  ```
  Therefore, to simplify the command line parameters as much as possible, it is recommended to write the relative path of the file in the include statement (if they are not in the same folder).
- 5-multi-fb-diff-gopkg: Multiple .fbs files with include relationships, and their go_package is different. Note: due to the limitation of flatc, it currently does not support two files with the same namespace but different go_package, and requires that the last segment of namespace and go_package in a file must be the same (for example, trpc.testapp.testserver and github.com/testapp/testserver have the same last segment "testserver").
