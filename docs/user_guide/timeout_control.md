# 1、Introduction

tRPC-Go's timeout control only occurs when the client calls the service, controlling the time for the call to wait. If the set time is exceeded, the client call immediately returns a timeout failure.

There are mainly three factors that affect timeout control:

- `Link Timeout`: The upstream caller transmits its allowed timeout to the current service through the protocol field. This means that there is only so much timeout I can tolerate. `Please be sure to return data to me within this timeout.` Any data returned after this time is meaningless. This is the `total Link Timeout` when A calls B, as shown in the following figure.
- `Message Timeout`: The longest message processing time of the current service `from receiving a request message to returning a response data`. This is the current service's way of controlling itself from wasting resources. This is `the total timeout for the current request` and is shown in the inside part-B of the following figure.
- `Call Timeout`: The timeout for each rpc request called by the current service to the downstream service. For example, the `individual timeout` for B calling C in the following figure. Typically, a request will call multiple rpcs in a row, such as B calling C, continuing to call D and E in series. This `Call Timeout` controls the independent timeout for each rpc.

When making an rpc call request, the timeout for the rpc call needs to be calculated. The real effective timeout is the minimum value calculated in real-time based on the above three factors. The calculation process is as follows:

- Firstly, calculate and get `the minimum value of the link timeout and message timeout`. For example, if the link timeout is 2s and the message timeout is 1s, then the maximum allowable processing time for the current message is 1s.
- When making an rpc call, calculate `the minimum value of the current message's maximum allowable processing time and the single call timeout`. For example, the single call timeout for B calling C in the figure below is 5s. Therefore, the actual timeout for B calling C is still 1s. As long as the timeout is greater than the current maximum allowable processing time, it is all invalid, and the minimum value will be taken. Another example, if the single call timeout for B calling C is 500ms, then the real timeout for B calling C is exactly 500ms, and this value will be transmitted to C through the protocol field. In the view of service C, the timeout is his link timeout, which will be transmitted and gradually reduced along the entire rpc call chain until it becomes zero, thus eliminating the problem of an infinite loop call.
- Each rpc call actually consumes some time, so `the current maximum allowable processing time for the message needs to be calculated in real-time to determine the remaining time`. For example, if B actually consumes 200ms when calling C, then only 800ms remains as the maximum allowable processing time. At this time, when the second rpc call is start, the minimum value of the remaining message timeout and the individual call timeout needs to be calculated. If the individual call timeout set for B calling D in the following figure is 1s, the actual effective timeout is still 800ms.

# 2, Principle diagram of full-chain timeout control model.
Principle diagram of tRPC-Go's full-chain timeout control model.
![ 'timeout_control.png'](/.resources/user_guide/timeout_control/timeout_control.png)

# 3, Timeout control implementation

tRPC-Go's timeout control is all implemented `based on the context ability`. Context, which means the context of requests, is the first parameter of all RPC interfaces and can set timeouts and cancellations. Therefore, to implement tRPC-Go's timeout control, all RPC calls must carry ctx from the request entry. `It should be noted that timeouts can only be controlled by ctx.`

Context can only control the timeout of each call and cannot control the end of the coroutine. If the business code does not use ctx but uses pure memory-consuming calculations (such as time.Sleep, select, and non-ctx calls, etc.), timeouts cannot be controlled, and the coroutine will be stuck forever and unable to exit.

When the server receives a request, context sets the maximum allowable processing time for the current request based on the timeout field in the protocol and the timeout field configured in the framework. Then it is handed over to the user for use, and the current context will be immediately canceled at the end of the processing function. So when you start a coroutine to process asynchronous logic by yourself, you must not use the context of the request entry and use a new context such as `trpc.BackgroundContext()`.

# 4, Timeout Configuration Example

All tRPC-Go timeout controls can be specified through the configuration file.

Note: the following settings are the timeout configuration for the current service itself and not the upstream's timeout configuration for the current service.

## Link Timeout

The timeout is transmitted through the protocol field from the source service down to subsequent services. Users can configure the flags to turn it on or off.

The link timeout is determined by the upstream client caller. If the client does not set it, there is no link timeout and there is no need to configure it to turn it off.

By default, trpc-client will set the actual timeout of the RPC call to the link timeout. Other clients generally do not set it.

```yaml
server:
  service:
    - name: trpc.app.server.service
      disable_request_timeout: true  # The default value is false. If it is false, the timeout will inherit the upstream service's setting time. If it is configured as true, it will disable this feature, indicating that the timeout transmitted to me by the protocol during the upstream service call will be ignored.
```
In some transaction scenarios, this value can be configured to ensure that all actions either succeed or all fail.

## Message Timeout

Each service can be configured with the maximum processing timeout for all requests when the service starts, which will only take effect when calling downstream services.

If the server's execution of a purely memory-consuming operation (e.g. time.Sleep, select, and non-ctx calls, etc.) causes the processing time of the request to exceed the message timeout, the processing coroutine will not stop immediately.

`It must be noted: timeouts can only be controlled by ctx.`

```yaml
server:
  service:
    - name: trpc.app.server.service
      timeout: 1000  # The unit is ms, each received request allows a maximum execution time of 1000ms, so pay attention to the timeout allocation of all serial rpc calls in the current request, the default is 0, no timeout is set
```

## Call Timeout
Each rpc backend call can configure the maximum timeout of the current call request. If the `WithTimeout Option` is set in the code, `the call timeout is subject to the option in the code, and this configuration does not take effect`. However, the code is not flexible enough. It is recommended not to set `WithTimeout Option` in the code.
`It must be noted: the timeout can only be controlled by ctx.`
```yaml
client:
  service:
    - name: trpc.app.server.service  # The service name of the backend service protocol file, the format is: pbpackagename.pbservicename
      timeout: 500  # The unit is ms, each initiated request allows a timeout of up to 500ms, the default is 0, no timeout is set, that is, infinite waiting
```
Each rpc request will take the minimum value of `link timeout`, `message timeout`, `call timeout` to call the backend, and the remaining time of the current message's longest processing timeout will be calculated in real time.