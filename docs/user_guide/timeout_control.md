English | [中文](timeout_control.zh_CN.md)

# tRPC-Go Timeout Control

## Introduction

When making RPC requests, the framework sets a timeout for waiting for the response. If the specified time is exceeded, the client call will immediately return a timeout failure.

Timeouts are divided into three configurations to provide finer-grained control over timeouts:

- `Fulllink Timeout`: Upstream callers specify the maximum allowed time for the entire request through protocol fields. This means that the upstream caller requests that the response be delivered within a specific time frame. Any response received after this time is considered meaningless. This is illustrated in the diagram below, where A calls B with a `Fulllink Timeout`.

- `Message Timeout`: This is the maximum timeout a current service is configured to spend processing a request from receiving the request message to sending the response data. This is a mechanism for the current service to control its resource utilization. It's depicted as the `Message Timeout` within service B in the diagram.

- `Calling Timeout`: This is the maximum timeout for each RPC request when calling downstream services. For example, in the diagram, when B calls C, it sets a `Calling Timeout`. Typically, a server may involves multiple consecutive RPC calls, as shown when B calls C and then sequentially calls D and E. The `Calling Timeout` controls the timeout for each individual RPC call.

![ 'timeout_control.png'](/.resources-without-git-lfs/user_guide/timeout_control/timeout_control.png)

When making an RPC call, framework will calculate the actual timeout for that specific call. The actual timeout is the minimum of the three timeout configurations mentioned above. The calculation process is as follows:

- First, calculate the minimum of the `Fulllink Timeout` and `Message Timeout`. For example, if the `Fulllink Timeout` is 2 seconds and the `Message Timeout` is 1 second, then the `maximum allowed processing time for the current message` is 1 second.

- When making an RPC call, calculate the minimum of the `maximum allowed processing time for the current message` and the `Calling Timeout` for that specific call. For example, if B->C sets a `Calling Timeout` of 5 seconds, then the actual timeout for B calling C is still 1 second. If B->C sets a `Calling Timeout` of 500 milliseconds, then the actual timeout for B calling C is 500 milliseconds. In this case, the value of 500 milliseconds is also passed to C via protocol fields, and from C's perspective, it becomes its `Fulllink Timeout`. The `Fulllink Timeout` time is passed down the entire RPC call chain, gradually decreasing until it reaches 0, preventing the possibility of infinite loops in calls.

- Since each RPC call consumes some time, the `maximum allowed processing time for the current message` needs to be continuously updated in real-time to account for the remaining time. For example, if B calling C actually takes 200 milliseconds, then the `maximum allowed processing time for the current message` is reduced to 800 milliseconds. When making B->D call, it's necessary to calculate the minimum of the `maximum allowed processing time for the current message` and the `Calling Timeout` for B->D. For example, if B->D sets a `Calling Timeout` of 1 second, the effective timeout for this call is still 800 milliseconds.

## Implementation

tRPC-Go's timeout control is implemented using the `Context`.

`Context` is the first parameter for all RPC interfaces, allowing you to set timeouts and cancellations. Therefore, to make timeout control effective, all RPC calls must carry the `Context` of the request entry. **It's important to note that timeouts can only be controlled through the `Context`**.

`Context` can only control the timeout of each individual call and cannot control the termination of goroutines. If your business code blocks without considering the `Context` (e.g., using `time.Sleep`), timeout control won't be effective, and the goroutine will be blocked indefinitely.

When the Server receives a request, it calculates the `maximum allowed processing time for the current message`, sets a timeout using `context.WithTimeout`, and cancels the current `Context` when the business processing function is complete. Therefore, if you create goroutines to execute asynchronous logic, you must not use the entry request's `Context`. Instead, use a new `Context` such as `trpc.BackgroundContext()`.

## Detailed Configuration

tRPC-Go's timeout control is specified entirely through configuration files.

**Note**: The following settings are all specific to the current service's timeout configuration, which corresponds to server B in the preceding diagram.

### Fulllink Timeout

By default, timeouts are passed down from the source service through protocol fields, and you can configure whether to inherit them or not. `Fulllink Timeout` is determined by the upstream client caller, and trpc Client by default sets the actual RPC timeout to the `Fulllink Timeout`.

```yaml
server:
  service:
    - name: trpc.app.server.service
      disable_request_timeout: true  # Default is false, which means the timeout inherits from the upstream service. Set to true to disable inheriting timeout values passed from the upstream service.
```

If you wish to completely disable timeouts, you can configure this value.

### Message Timeout

Each service can configure the maximum message processing timeout for all incoming requests when starting up.

If your business code blocks without considering the `Context` (e.g., using `time.Sleep`), timeout control won't be effective, and the processing goroutine won't terminate immediately.

```yaml
server:
  service:
    - name: trpc.app.server.service
      timeout: 1000  # In milliseconds, each incoming request allows a maximum execution time of 1000ms. Be mindful of distributing timeout times for all serial RPC calls within the current request. Default is 0, indicating no timeout is set.
```

### Calling Timeout

Each RPC call can configure the maximum timeout for the current request. If your code sets the `WithTimeout` option, the code configuration takes precedence. However, for flexibility, it's recommended to specify the calling timeout directly through the configuration file.

```yaml
client:
  service:
    - name: trpc.app.server.service  # Downstream service name
      timeout: 500  # In milliseconds, each initiated request allows a maximum timeout of 500ms. Default is 0, indicating no timeout is set, meaning it will wait indefinitely.
```