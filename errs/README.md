English | [中文](./README.zh_CN.md)

# 1. Preface

tRPC's error handling mechanism, which functions across different languages, consists of an error code and an error description message. It does not comply with the native practice in Go, which only returns a single string.

To facilitate coding, tRPC-Go provides a wrapper library named errs. It is important to notice that when an RPC interface call fails, `errs.New(code, msg)` is used to return the error code and information, rather than returning the `errors.New(msg)` provided by the Go standard library directly.

```go
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    if failed { // Business logic failure.
        return nil, errs.New(your-int-code, "your business error message") // Fail: Define your own error code, and return the error message to the upstream.
    }
    return &pb.HelloReply{xxx}, nil // Success: Return nil.
}
```

# 2. Definition of Error Codes

In tRPC-Go, there are three types of error code.

- Error Code of the Framework, related to the `framework`
- Error Code of the Callee Framework, related to the `callee framework`
- Error Code of the Business Logic, related to the `business`

## 2.1 Error Code of the Framework

Error Code of the Framework refers to the error code automatically returned by the current framework of its own service, such as timeout when calling downstream services, unserialize failure, etc.

All of them are predefined in the file `trpc.proto`.

- `0~100` covers errors of the server. The framework returns the error code and detailed info to the caller when an error is caused after receiving the requests but before it is handled by the functions of business logics. To the caller, the error code is provided by callee. (See also Section 2.2)
- `101~200` covers errors of the client, which represents errors caused by a failure return by callee services.
- `201~300` covers errors of the streaming.

Here is an example of the Framework's error code in the log files.

```go
type:framework, code:101, msg:xxx timeout
```

## 2.2 Error Code of the Callee Framework

Error codes of the callee framework are those returned by callee services during an RPC call. Although the callee service may not be aware of these errors, it's apparent that they are related to the service rather than the caller. These errors are primarily caused by invalid or incorrect parameters.

If you encounter this type of error, please contact the owner of the callee services for more details. It's also essential to verify that the parameters being passed to the callee service are valid and correct to prevent these errors from occurring.

Here is an example of the Callee Framework's error code in the log files.

```go
type:callee framework, code:12, msg:rpcname:xxx invalid
```

## 2.3 Error Code of the Business Logic

The Error Code of the Business Logic refers to the error code returned by callee services via the `errs.New` function when the caller made an RPC call. It's important to note that this error code, thrown by the business logic, is defined by the developer of the callee services. The meanings of these errors are not related to the framework, and details of them should be consulted with the developers.

tRPC-Go recommends using `errs.New` for business logic errors, rather than defining error codes and outputting errors in the body of the response, as this allows the framework to monitor and report business logic errors. Alternatively, an SDK should be introduced to report errors, which could be inconvenient.

Additionally, in order to differentiate errors, it's suggested to use numbers greater than 10000.

An example of Error Code of the Business Logic is listed as below.

```go
type:business, code:10000, msg:xxx fail
```

# 3. List of Error Codes

**Attention:**

Please note that the error codes listed below only pertain to the framework or the framework of the callee services. The Error Code of the Business Logic is defined by the developers of different RPC services, and the meanings of these errors should be consulted with those developers.

Moreover, error codes can only indicate categories of errors, so it's essential to check the detailed error information to determine the root causes.

| Error Code | Details of the Error                                         |
| ---------- | ------------------------------------------------------------ |
| 0          | Success                                                      |
| 1          | A server decoding error can be caused by misaligned or unsynchronized updates of the protobuf fields between the upstream and downstream services, leading to unpacking failure. Keeping protobuf synchronization and ensuring all services are updated to the latest version of protobuf can solve this problem. |
| 2          | The server has encountered an encoding error, resulting in the failure of serializing the response package. This error can occur due to an issue with setting the PB field, such as inserting binary data with invisible characters into a string field. Please refer to the error information for further details. |
| 11         | The server did not call the corresponding service implementation, and tRPC-Go has no error code related to this error code, as they are defined by other language versions of tRPC. |
| 12         | The server failed to call the appropriate interface implementation due to incorrect function call filling. Please refer to the FAQ section for more information. |
| 21         | The server's business logic processing time has exceeded the maximum or message timeout. Please contact the responsible person of the callee service. |
| 22         | The server is overloaded due to the utilization of a rate-limiting plugin on the callee server, exceeding its capacity threshold. Please contact the owner of the callee service. |
| 23         | Request is rate-limited by the server                        |
| 24         | The server has timed out due to a full RPC call timeout, which implies that the timeout set by the upstream caller is insufficient to enter the business logic of this service in time. |
| 31         | If you encounter a server system error caused by panic, it is likely due to programming bugs like null pointers or array index out of bounds. To address the issue, please contact the owner of the callee service. |
| 41         | Authentication failure, this may be due to issues like `cors` cross domain check failed, `ptlogin` login status checking failed, and `knocklock` checking failed. Please contact the owner of the callee service. |
| 51         | Input parameter validation error.                            |
| 101        | The request timed out when calling on the client due to various reasons. Please refer to the FAQ for more details. |
| 102        | Client's full RPC call timeout. If this error occurs, it means that either the current timeout for initiating the RPC is too short, the upstream did not provide sufficient timeout, or previous RPC calls have exhausted most of their time. |
| 111        | Client connection error. This is typically because the downstream is not listening to the `ipport`, mostly due to downstream startup failure. |
| 121        | Client encoding error, serialization request package failed, which is similar to No.2 listed above |
| 122        | Client decoding error, usually due to misaligned pb, which is similar to No.1 above |
| 123        | Request is rate-limited by client                            |
| 124        | Client overloaded                                            |
| 131        | Client IP routing error. It's usually caused by inputing the incorrect service name or no available instances under the service name |
| 141        | Multiple network errors can cause client issues. Please refer to the FAQ for specific details. |
| 151        | Response parameter validation failed.                        |
| 161        | Upstream caller cancels the request in advance               |
| 171        | Client reading frame data error                              |
| 201        | Client streaming queue full                                  |
| 351        | Client streaming ended.                                      |
| 999        | Unknown errors are often the result of returning errors without error codes, or not using the framework's built-in `errs.New(code, message)` function to return errors. |
| Others     | The columns listed above represent the framework error codes defined by the framework itself. Error codes not included in this list suggest that they are business error codes, which are defined by the business development team. To address these errors, please consult with the owner of the service being called. |

# 4. Technical Details

Below are the code details of the `Error` structure.

```Go
type Error struct { 
    Type int    // Type of the Error Code 1 Error Code of the Framework 2 Error Code of the Business Logic 3 Error Code of the Callee Framework
    Code int32  // Error Code
    Msg  string // Description of the Error
    Desc string // Additional error description. It is mainly used for monitoring purposes, such as TRPC framework errors with a TRPC prefix and HTTP protocol errors with an HTTP prefix. Users can capture these errors by implementing interceptors and change this field to report any prefix for monitoring.
}
```

Error handling process:

1. If a user explicitly returns a business error or framework failure through `errs.New`. Based on its type, the err will be filled into `ret` which indicates a framework error, or `func_ret` which indicates business logic errors
2. When composing and returning the response, the framework checks for errors. If errors are found, the response body will be discarded. Thus, `if the return fails, do not try to return the data through response body again`.
3. If the upstream caller encounters an error during a call, the`framework error` will be directly created and returned to the user. If the call is successful, the framework or business error in the trpc protocol will be resolved, and any `callee framework errors or business errors` will be created and returned to the user.

# 5. FAQ

## 5.1 RPC request returned an error

### 12 rpc name:xxx invalid

- Firstly, it is important to understand that rpcname is the method name in the proto protocol file, with the format `/package.service/method`. It is unrelated to the configuration and is not the servicename in the configuration file.
- Check if the method name `/package.service/method` generated from the called service's protocol file matches the `method name` set by the calling client.
- Check if there is an error in the pb reference, if the called party's service IP address is incorrect, or if you have accidentally called someone else's service.
- Since trpc-go supports reuseport by default, when developing locally, you need to confirm whether multiple different services have started on the same ipport. If multiple different services are started, there will be times when it works normally and times when it fails.
- After `NewServer`, make sure to register the correct pb implementation, for example: `pb.RegisterService(s, &GreeterServerImpl{})`.
- Check if the description of `serviceDesc` in the `pb.go` file generated by the called party's pb generation tool, including `serviceName` and `Func`, is correct.
- Check if an indirect use of the "git.woa.com/polaris/polaris-go" version v0.4.1 has occurred, as this version contains anomalies. It is necessary to upgrade to a version above v0.4.2.

### 31 runtime error: index out of range (or nil pointer)

The downstream service array out-of-bounds or null pointer caused the server to panic. It is a problem with the downstream service, not your problem.

### 161 context canceled

Context cancellation occurs in two situations:

- The context is canceled early due to the client connection being disconnected, usually because there isn't enough time. The upstream client times out and actively disconnects the connection. The current service detects this event and cancels the ongoing network request to avoid doing unnecessary work. This situation is considered normal. This error is common in http servers, especially with external web anomalies. It can be triggered by a user manually refreshing the page and immediately exiting, a client crash, or a bug in the client's webview.
- The context is canceled early due to the rpc function exiting. After the rpc function at the service entry returns, the framework automatically cancels the context. Therefore, you should not continue using the ctx passed in from the request entry in asynchronous goroutines, as the ctx has already been destroyed at this point. For asynchronous calls, do not use the ctx from the request entry; instead, use the asynchronous start API provided by the framework: [`trpc.Go(ctx, timeout, handler)`](https://git.woa.com/trpc-go/trpc-go/blob/master/trpc_util.go#L152).

### 141 EOF

"End of file" error, the peer closed the connection, which could be due to the peer service panicking, or the peer service closing abnormally. It is necessary to have the called service check for the relevant cause.

- 1 If the called party is not a trpc service, it is highly likely due to the idle time of the connection. The default idle time for a trpc-go client's connection is 50 seconds. When the idle time of the called service is less than 50 seconds, the server side actively closes the connection. The trpc-go client will then attempt to reuse a connection that has already been closed, leading to errors. There are three solutions (choose one):
  - 1.1 The called service should increase the idle time of the connection to more than 50 seconds. (The default idle time for a trpc go server is 1 minute, which is greater than 50 seconds and will not cause issues unless the user has set the server idletime arbitrarily.)
  - 1.2 On the trpc-go client side, adjust the idle time of the connection to be less than the idle time of the called party: `connpool.WithIdleTimeout(time.Second)`.

    ```go
    // Example main function calls during initialization.
    connpool.DefaultConnectionPool = connpool.NewConnectionPool(connpool.WithIdleTimeout(time.Second))
    ```

  - 1.3 If the server supports multiplexing of client connections (i.e., multiple sends and receives within a single connection, trpc-go server v0.5.0 and above default supports multiplexing, which is essentially asynchronous server_async, generally requiring the reuse of the server transport logic of the trpc protocol), then the client caller can enable the multiplexing option: `client.WithMultiplexed(true)`.
- 2 If the issue is caused by the called party's service restart, it indicates that there is a problem with the deployment process. The correct procedure for deploying a service is to first remove the IP and port of the service to be deployed from the naming service and wait for a period of time (the cache time varies depending on the naming service, about 30 seconds for Polaris) before starting to delete the old container and rebuild the new one. After the new container starts successfully, add the new IP and port to the naming service.
- 3 If the processing time of the called service is too long and exceeds the server's idletime (default 1 minute), the server will actively close the connection. At this point, the client will get a connection with EOF. The solution can be to increase the server's idletime as per 1.1 above, to be greater than the processing time, or to enable client connection multiplexing as per 1.3 above, provided that the server supports connection multiplexing.
- 4 When the server-side framework version is less than v0.9.5, it will directly close the client's connection for trpc protocol packets exceeding 10MB without returning an error indicating that the packet length is too long to the client. This causes the client to only see a 141 EOF error. After >= v0.9.5, the server-side has optimized this error message [optimization](https://git.woa.com/trpc-go/trpc-go/-/merge_requests/1467). For this error, both the client and server need to manually set `trpc.DefaultMaxFrameSize = xxx` in the code to increase it (both the server and client need to set it).

### 141 tcp client transport ReadFrame: trpc framer: read framer head magic not match

This issue may be caused by network reasons. You can telnet the IP and port to check if the network is accessible. If it's not, adjust the policy accordingly. It's also possible that the Protocol in the test JSON file is not configured correctly.
It's also possible that the protocols of the upstream and downstream services are not aligned, such as sending a trpc request to an http service.

### 141 connection close

The peer's business layer directly closes the connection. This is generally due to a mismatch in the upstream and downstream protocols, such as sending a trpc protocol request to an http server.

### 111/141 connection refused

Different protocols may have different error codes, but the error message is consistent. If the peer does not provide a service on the requested IP:port, a "connection refused" error will occur, indicating that the connection was directly rejected by the peer. This is generally because the downstream service is down or the IP:port is incorrect and not listening on that IP:port. Please ensure that the called service is started normally. This error is very clear, and it is 100% certain that the downstream service is not listening on this IP:port. Do not say the service is normal again; doubt this document, as it is almost certain that the downstream service has been restarted.

### 101 write timeout

A timeout when writing data usually means that the previous RPC has already consumed all the time before the current RPC is called, so there is actually no time to send out this RPC. Please check the service's timeout configuration. For timeout control logic, please refer to the [documentation](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688).

### 101 read timeout

A timeout when reading data usually occurs when the downstream service does not return within the specified time. Please check the service's timeout configuration. For timeout control logic, please refer to the [documentation](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688).

### 101 dial timeout

A timeout when establishing a connection is generally due to network unavailability, similar to a write timeout, or it could be because the downstream service is overloaded and the listening queue is full. Please check the service's timeout configuration. For timeout control logic, please refer to the [documentation](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688).

### 101/141 context deadline exceeded

Different plugins may have different error codes, but "deadline exceeded" means that there is not enough time left, similar to a 101 write timeout.

### 131 client Select

Client addressing error, see [here](https://iwiki.woa.com/p/4008319150#6faq).

### 122 client codec Decode: rsp request id xxx different from req request id

The response ID does not match the request ID; this response is not for the current request.
By default, the trpc go client uses an exclusive connection pool mode. After sending a packet, it will hang and wait for a response, and then put the connection back into the connection pool for reuse next time.
Under normal circumstances, the responses are consistent. Such situations usually occur when there is a bug in the called party's code, where the same request has been responded to multiple times. Since the client only takes one response, it results in the next reuse getting the previous response.

The following two solutions (choose one):

1. The called party should investigate the bug to see if `WriteResponse` or similar interfaces have been called multiple times, causing multiple responses. The tRPC-Go server can only respond automatically through function returns, so this issue will not occur. Other languages such as trpc-cpp and trpc-node provide interfaces for users to respond themselves, so it is very likely that this bug will occur.
2. Change the calling party to use IO multiplexing mode, see here: [tRPC-Go Client Connection Modes](https://iwiki.woa.com/p/435513714), add a client option: `client.WithMultiplexed(true)`. Why not set IO multiplexing as the default? Because in the early stages, for universality, to support all protocols, many private protocols, like HTTP, do not have a request ID, so IO multiplexing cannot be used.

It is recommended to adopt the first solution above, as it is caused by a code bug. The second solution can also solve the problem, but it just hides the issue forever.

### -1 xxx

The error code is not in the list of section 3, which indicates that it is defined by the business itself, and it is necessary to find the corresponding person in charge.

## 5.2 All socket network request error concepts

### EOF

An "End of file" error occurs when the peer closes the connection. It could be due to the peer service panicking or the peer service closing abnormally. The called service needs to investigate the relevant cause.

### reset by peer

The peer sent a reset signal, indicating that the connection has been discarded. This may occur when the peer service is abnormal or under excessive load. The called service needs to investigate the relevant cause.
It is also possible that the protocols of the upstream and downstream services are not aligned, such as sending a TRPC request to an HTTP service.

### broken pipe

The peer has already closed the connection, and if the caller continues to operate the socket without realizing it, a broken pipe error will occur. This error may appear when the peer crashes.
It is also possible that the package is too large, exceeding the size limit of 10M. First, consider the rationality of large packages, then consider setting your own package size limit: `trpc.DefaultMaxFrameSize=1111`.

### connection refused

If the peer does not provide a service on the requested ip:port, a connection refused error will occur, indicating that the connection was directly rejected by the peer. Please ensure that the called service is functioning normally.

## 5.3 Timeout issue: type: framework, code: 101, msg: xxx timeout

### I have clearly set a very long timeout time, so why does it prompt a timeout failure after only a short period of time?

The framework has a limit on the maximum processing time for each request received. The timeout time for each RPC backend call is calculated in real-time based on the current remaining maximum processing time and the call timeout. In this case, it is very likely that during multiple serial RPC calls, the previous RPC has already consumed almost all the time, leaving insufficient time for this RPC.
So, when making multiple RPC calls, you should reasonably allocate the timeout time for each RPC. If each RPC indeed takes a long time, then you should increase the message timeout, or disable the inherited link timeout.

### Why does it always prompt a context cancel error when starting a goroutine to make a network request through Go?

The term 'context' refers to the request context, which is canceled immediately when the current request function exits. Therefore, the goroutine you start using `go` cannot continue to use the `ctx` carried by the request entry and needs to use a new context, such as `trpc.BackgroundContext()`.

### Why do I always get a timeout when sending requests with the trpc-cli tool?

When sending requests with the trpc-cli tool, the default timeout setting is 1 second. Since your service takes a relatively long time, it causes the tool to fail. You can first confirm whether the ipport is correct, then investigate why the service takes so long internally, or increase the timeout time of trpc-cli: `trpc-cli -timeout 5000 -func ...`.

### How to locate a 101 timeout error?

1. First, read and understand the concept of [Timeout Control](https://iwiki.woa.com/pages/viewpage.action?pageId=99485688) to understand the definitions of link timeout and message timeout.
2. Determine if the downstream address is correct, including the environment namespace, service name servicename, and ipport when connecting directly.
3. Confirm whether the downstream service has received the request, whether the processing time is too long, and determine if the network is normal.
4. Timeout issues can be conveniently located using the [trpc-filter/debuglog](https://git.woa.com/trpc-go/trpc-filter/tree/master/debuglog) plugin.
5. Through debuglog logs, you can see the specific duration of each RPC, which can roughly indicate where the problem lies, and determine where the time is mainly consumed.
6. You can use the [tjg call chain](https://git.woa.com/trpc-go/trpc-opentracing-tjg) to troubleshoot execution issues upstream and downstream.
7. If you still can't locate the issue, you can [enable trace logs](https://git.woa.com/trpc-go/trpc-go/tree/master/log) in the downstream service. It is estimated that there might be a mismatch in the protocols between upstream and downstream, causing the downstream to drop packets directly.
8. Ensure that all plugin versions are the latest. There are bugs in the naming service addressing of older versions both upstream and downstream, whether it's Go or C++, they all need to be upgraded and updated.
9. Determine if the network environment is normal by trying on a different machine (or container).
