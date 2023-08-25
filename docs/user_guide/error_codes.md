[TOC]



## 1. Preface

tRPC's error handling mechanism, which functions across different languages, consists of an error code and an error description message. It does not comply with the native practice in Go, which only returns a single string.

To facilitate coding, tRPC-Go provides a wrapper library named errs. It is important to notice that when an RPC interface call fails, `errs.New(code, msg)` is used to return the error code and information, rather than returning the `errors.New(msg)` provided by the Go standard library directly.

```
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    if failed { // 业务逻辑失败
        return nil, errs.New(your-int-code, "your business error message") // 失败 自己定义错误码，错误信息返回给上游
    }
    return &pb.HelloRepley{xxx}, nil // 成功返回 nil
}
```


## 2. Definition of Error Codes

In tRPC-Go, there are three types of error code.

- Error Code of the Framework, related to the `framework`

- Error Code of the Callee Framework, related to the `callee framework`

- Error Code of the Business Logic, related to the `business`

### Error Code of the Framework

Error Code of the Framework refers to the error code automatically returned by the current framework of its own service, such as timeout when calling downstream services, unserialize failure, etc.  

All of them are predefined in the file `trpc.proto`.
- `0~100` covers errors of the server. The framework returns the error code and detailed info to the caller when an error is caused after receiving the requests but before it is handled by the functions of business logics. To the caller, the error code is provided by callee. (See also Section 2.2)
- `101~200` covers errors of the client, which represents errors caused by a failure return by callee services.
- `201~300` covers errors of the streaming. 

Here is an example of the Framework's error code in the log files.
```
type:framework, code:101, msg:xxx timeout
```


### Error Code of the Callee Framework

Error codes of the callee framework are those returned by callee services during an RPC call. Although the callee service may not be aware of these errors, it's apparent that they are related to the service rather than the caller. These errors are primarily caused by invalid or incorrect parameters.

If you encounter this type of error, please contact the owner of the callee services for more details. It's also essential to verify that the parameters being passed to the callee service are valid and correct to prevent these errors from occurring.

Here is an example of the Callee Framework's error code in the log files.

```
type:callee framework, code:12, msg:rpcname:xxx invalid
```


### Error Code of the Business Logic

The Error Code of the Business Logic refers to the error code returned by callee services via the `errs.New` function when the caller made an RPC call. It's important to note that this error code, thrown by the business logic, is defined by the developer of the callee services. The meanings of these errors are not related to the framework, and details of them should be consulted with the developers.

tRPC-Go recommends using `errs.New` for business logic errors, rather than defining error codes and outputting errors in the body of the response, as this allows the framework to monitor and report business logic errors. Alternatively, an SDK should be introduced to report errors, which could be inconvenient.

Additionally, in order to differentiate errors, it's suggested to use numbers greater than 10000.

An example of Error Code of the Business Logic is listed as below.
```
type:business, code:10000, msg:xxx fail
```


## 3. List of Error Codes

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



## 4. Technical Details

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
2. When composing and returning the response, the framework checks for errors. If errors are found, the response body will be discarded. Thus, `if the return fails, do not try to return the data through reponse body again`.
3. If the upstream caller encounters an error during a call, the` framework error` will be directly created and returned to the user. If the call is successful, the framework or business error in the trpc protocol will be resolved, and any `callee framework errors or business errors` will be created and returned to the user.