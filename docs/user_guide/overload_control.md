# TRPC-Go Overload Control

## Introduction
RPC framework should provide stability guarantee for services. Stability here refers to, when a large number of requests come in:
- Ensure that the latency of successful requests is stable at a low level, without significant fluctuations or expansion.
- Reject requests beyond processing capacity in a timely manner to prevent link timeouts.
- Avoid OOM caused by too many coroutines or long queues.

There are two different approaches to solving link stability issues. One is a quota-based rate limiting strategy, and the other is server-side adaptive overload protection.
Rate limiting limits the QPS of requests to keep the entire service chain at a lower load level. It has two types: single-machine and distributed, and is a common strategy.
Server-side adaptive overload protection monitors the running state of the service itself and rejects too many requests to stabilize the service at an optimal state.
Usually, server-side adaptive overload protection is sufficient to provide stability guarantees for your service. If you need to limit the traffic of certain types of requests, you can also use the two together.

## Server-side adaptive overload protection
> Please refer to the tRPC proposal [A10_overload_control](https://git.woa.com/trpc/trpc-proposal/blob/master/A10-overload-control.md) for design details.
trpc-go has supported server-side adaptive overload protection since [v0.7.0](https://git.woa.com/trpc-go/trpc-go/blob/v0.7.0/CHANGELOG.md).
Please use it in conjunction with the [trpc-go/trpc-overload-control](https://git.woa.com/trpc-go/trpc-overload-control) algorithm library.
Please use the latest version [v1.4.2](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.4.2/CHANGELOG.md).

tRPC-Go provides overload protection strategies based on three metrics: maximum coroutine scheduling time, coroutine sleep drift, and maximum request time.
Request time is not enabled by default. When the framework detects that a metric exceeds the maximum expected value, it will prioritize and apply flow control to some requests, immediately returning an overload error to ensure that it remains stable near the maximum expected value and guarantee service stability.

Please enable `log monitoring` and `dry-run mode` in the test environment first to ensure that the default coroutine scheduling time performs well.
If your upstream is a tRPC-CPP service, please ensure that its version is greater than [v0.10.0](https://git.woa.com/trpc-cpp/trpc-cpp/blob/v0.10.0/CHANGELOG.md).
Before this version, the CPP Polaris fusing would count overload errors, which did not meet the expectations of overload protection.
To use adaptive overload protection, the service QPS needs to be at least 300. For very low QPS, it is recommended to use quota-based flow control directly,
such as [rate limiter](https://pkg.go.dev/golang.org/x/time/rate) or the Polaris flow control mentioned in this document.

To enable adaptive overload protection, you only need to anonymously import the overload protection package, which will register a default overload protection interceptor named `overload_control`.

```java
import _ "git.code.oa.com/trpc-go/trpc-overload-control/filter"
```
Then add the `overload_control` interceptor to the `trpc_go.yaml` file.
```yaml
server:
  filter: [overload_control]
  service:
    - name: xxx
```
### Priority
Overload protection supports priority function, which allows high-priority requests to pass through first
and rejects low-priority requests when the service is overloaded. The priority of a request is 0 by default (lowest),
and is randomly rejected when overloaded. The following method can be used to mark a request with a priority label.

```aidl
import overloadctrl "git.code.oa.com/trpc-go/trpc-overload-control"
// Set the priority of the request to 128, with a maximum allowed value of 255 (select the correct method to set).
ctx = overloadctrl.SetServerRequestPriority(ctx, 128) // Use this method only when the following method does not work.
ctx = overloadctrl.SetRequestPriority(ctx, 128) // This method must be used when setting priority in client interceptors.
```
It sets the priority in the request's `meta data`, which is passed down with the request chain.
### Plugin Configuration
You can adjust the parameters of overload protection through plugins.
```yaml
plugins:
  # Plugin type，which must be `overload_control`
  overload_control: 
    # Plugin name, which is also the name of the interceptor registered by the plugin.
    # If using `overload_control`, it will override the default interceptor registered.
    # If using a different plugin name, the plugin must be manually registered by calling the `plugin.Register` method in the code.
    overload_control:
      # All configuration options can be left blank, in which case default values will be used.
      # Server-side overload protection
      server:
        # When both whitelist and blacklist are configured, only the whitelist takes effect
        # Whitelist, this interceptor only takes effect on the services/methods in the whitelist.
        whitelist:
          # Service name, msg.CalleeServiceName(). It has two configured methods, method_1 and method_2, in the whitelist. Overload protection does not take effect on other methods.
          service_a:
            # Method name, msg.CalleeMethod(). method_1 is in the whitelist.
            method_1:
            # method_2 is also in the whitelist.
            method_2:
          # All methods under service_b are in the whitelist.
          service_b:
        # Other services are not in the whitelist, so the algorithm does not apply to them.
        # Blacklist only takes effect when a whitelist is not configured, and the algorithm will ignore services/methods in the blacklist.
        blacklist:
          # The two methods, method_1 and method_2, configured under service_x are in the blacklist, while other methods are not.
          service_x:
            # Method name, msg.CalleeMethod(). method_1 is in the blacklist.
            method_1:
            # method_2 is also in the blacklist.
            method_2:
          # All methods under service_y are in the blacklist.  
          service_y:
        dry_run: false # The dry run mode is disabled by default. Enabling it causes the overload protection to always allow requests, allowing the algorithm's state to be observed through logs without affecting the business.
        goroutine_schedule_delay: 3ms # The expected maximum coroutine scheduling time is 3ms by default. If you need to adjust this value, please read the overload protection proposal first.
        sleep_drift: 0ms # The expected maximum coroutine sleep drift is 0 by default, and it is not enabled. If your service does not use the native trpc protocol, or uses udp only (coroutine scheduling time cannot take effect), please set this value to 3ms. If you need to adjust this value, please read the overload protection proposal first.
        request_latency: 0ms # The expected maximum request duration is 0 by default, and it is not enabled. If you need to adjust this value, please read the overload protection proposal first.
        cpu_threshold: 0.75 # The minimum CPU usage (for the entire container) when overload protection is enabled is 75% by default.
        cpu_interval: 1s # Calculate the CPU usage rate in the past 1 second. The higher this value, the slower the overload protection switches between on and off. The default is 1 second.
        log_interval: 0ms # The minimum time interval for overload protection status logging, used for debugging. 0 means logging is not enabled.
```
If the custom plugin name is not `overload_control`, the plugin needs to be registered manually.
```aidl
import "git.code.oa.com/trpc-go/trpc-go/plugin"
import "git.code.oa.com/trpc-go/trpc-overload-control/filter"

plugin.Register(name, filter.NewPlugin(/* options */))
```
The `NewPlugin` method can receive parameters and allow modification of the default overload protection strategy created by the plugin. Yaml configuration can adjust the default strategy based on it.

The plugin will register an interceptor with the same name as the plugin name. To use it, fill in the plugin name in the filter.
Note: please place the overload protection interceptor after the monitoring interceptor so that the overload error will be reported when the monitored service is called.
When a global overload protection strategy needs to be enabled, but a custom strategy is required for a particular service, the service can be added to the blacklist of the global strategy, and an additional interceptor can be added in the service's filter.

### Add An Overload Protection Interceptor Through Code
The `plugin` only provides partial capabilities. To create more precise overload protection strategies through code,
refer to the [RegisterServer](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.2.0/filter/filter.go#L39) method
and various [Opt](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.2.0/options.go#L45) options of
the [server](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.2.0/overloadctrl.go#L28) overload protection library.

## Client-side Adaptive Overload Protection.
> Please refer to the [Priority Overload Protection proposal](https://git.woa.com/cooperyan/trpc-proposal/blob/A12_client_oc/A12-priority.md#%E5%9C%A8%E5%AE%A2%E6%88%B7%E7%AB%AF%E6%8F%90%E5%89%8D%E9%99%90%E6%B5%81) for design details.   
> The version of tRPC-Go must be >= [v0.8.2](https://git.woa.com/trpc-go/trpc-go/blob/v0.8.2/CHANGELOG.md).

Many times, we hope to perceive that the backend service is overloaded at the entrance of the entire request chain, and perform degradation processing in advance to avoid wasting additional resources.
Similar to the Priority Overload Protection for the server in Chapter 2, tRPC-Go also supports client-side Priority Overload Protection. This algorithm is based on the error code returned by the backend.
- `22`: Downstream service overload
- `124`: The downstream has enabled client overload protection and detected overload in its downstream, therefore returning an overload error preemptively.
- `23`: The server-side error of the Polaris's flow control.
- `124`: The client-side error of the Polaris's flow control.

As a basis for decision-making (to add new error codes, please refer to the `throttle_err_codes` in the YAML configuration),
a certain proportion of requests are rejected in advance on the client side to ensure that the server is appropriately overloaded.   
These rejected requests will return error code `124` because they have never left the client,
so a new node can be selected for a confident retry. The algorithm prioritizes rejecting low-priority requests and allowing high-priority requests.

### Enable Client-side Overload Protection.
To enable client-side adaptive overload protection, you need to first anonymously import the client overload protection package.
```java
 import _ "git.code.oa.com/trpc-go/trpc-overload-control/filter"
```


Then add overload protection interceptors for the client in `trpc_go.yaml`.
```yaml
client:
  # Note that the selector interceptor should be configured before the overload protection interceptor!
  # "selector" is the default service discovery interceptor name in the tRPC-Go framework. If not explicitly specified in the filter, it will be automatically added after all interceptors.
  filter: [selector, overload_control]
  service:
    - name: xxx
```
### Plugin Configuration
Do not modify the default configuration unless necessary.  
The parameters of client overload protection can be adjusted in the yaml configuration of the plugin:
```yaml
plugins:
  # Plugin type, which must be `overload_control`
  overload_control:
    # Plugin name, also the interceptor name registered by the plugin
    # If overload_control is used, it will override the default interceptor registered
    # If other plugin names are used, the plugin.Register method must be called manually in the code for registration
    overload_control:
      # All configuration items can be left blank, in which case the default values will be used
      # Client overload protection
      client: 
        # When both whitelist and blacklist are configured, only whitelist will take effect
        whitelist: # Whitelist, this interceptor only works for services/methods in the whitelist
          service_a: # Service name, msg.CalleeServiceName(), with two method_1/2 configured under it in the whitelist, overload protection does not apply to other methods
            method_1: # Method name, msg.CalleeMethod(), method_1 is in the whitelist
            method_2: # method_2 is also in the whitelist
          service_b: # All methods under service_b are in the whitelist
          # Other services are not in the whitelist, and the algorithm does not apply to them
        blacklist: # Blacklist, only takes effect when whitelist is not configured, the algorithm will ignore the services/methods in the blacklist
          service_x: # Two method_1/2 configured under service_x are in the blacklist, while other methods are not in
            method_1: # Method name, msg.CalleeMethod(), method_1 is in the blacklist
            method_2: # method_2 is also in the blacklist
          service_y: # All methods under service_y are in the blacklist
        throttle_err_codes: [] # Additional backend error codes are added as the basis for flow control decisions. For example, 101, 141, etc.
        max_throttle_probability: 0.7 # The maximum flow control ratio, which by default rejects up to 70% of requests, i.e. at least 30% of requests will reach the backend service
        ratio_for_accept: 1.3 # Ensures that the proportion of overloaded requests sent downstream does not exceed about 0.3 = 1.3 - 1 by default, see priority overload protection proposal for details
        ema_factor: 0.8 # In the statistics of the number of requests, the factor of exponential moving average, with a default value of 0.8. Ordinary users should not set this, see the code for details
        ema_interval: 100ms # The time range used to calculate the exponential moving average of the number of requests, with a default value of 100ms. Ordinary users should not set this, see the code for details
        ema_max_idle: 30s # In the statistics of the number of requests, if there is no request within a certain period of time, the number will be reset to 0, with a default value of 30s. Ordinary users should not set this, see the code for details
        log_interval: 0ms # The minimum time interval for overload protection state logs, used for debugging, 0 means no log is enabled.
```
Similar to server-side overload protection, when the custom plugin name is not `overload_control`, users also need to register the plugin manually. This will not be elaborated here.

### Adding A Client Overload Protection Interceptor Through Code.
The plugin only provides partial capabilities. Through code, more refined overload protection strategies can be created.
Please refer to the [RegisterClient](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.3.2/filter/filter.go#L32) method
and various [opt](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.3.2/client/options.go#L11)
in the [client](https://git.woa.com/trpc-go/trpc-overload-control/blob/v1.3.2/client/overloadctrl.go#L26) overload protection library.

## Rate Limiting
tRPC currently provides flow control strategy based on `Polaris`. Please refer to [this proposal](https://git.woa.com/trpc/trpc-proposal/blob/master/A9-polaris-limiter.md#%E5%8C%97%E6%9E%81%E6%98%9F%E9%99%90%E6%B5%81).

### Polaris
The `Polaris` flow control of tRPC-Go is implemented through the [trpc-filter/polaris/limiter](https://git.woa.com/trpc-go/trpc-filter/tree/master/limiter/polaris) plugin.
It is a wrapper for the `Polaris's` [SDK](https://git.woa.com/polaris/polaris-go), which allows tRPC users to easily access `Polaris` flow control.
When the request is limited, the server will return framework error code `23`, and the client will return framework error code `123`.
For detailed `Polaris` flow control capabilities, please refer to the [access rate limiting usage guide](todo).
The following briefly introduces the configuration of the plugin and flow control strategy.

#### tRPC-Go Service Configuration
Anonymously referencing plugins in code:
```java
import _ "git.code.oa.com/trpc-go/trpc-filter/limiter/polaris"
```
Configure trpc_go.yaml:
```yaml
client:
  filter: [polaris_limiter]  # Enable client-side rate limiting.
  service:
    # ...
server:
  filter: [polaris_limiter]  # Enable server-side rate limiting.
  service:
    # ...
plugins:
  limiter:
    polaris: # Cannot be omitted.
      timeout: 1s # Can be omitted. If omitted, a default value of 1 second will be used.
      max_retries: 2 # It can be omitted. If omitted, there will be no retries by default.
      # metrics_provider determines whether to enable the plugin's metric reporting. By default, it is empty and disabled. Note that the Polaris console already provides monitoring capabilities.
      # Currently, only "m007" is supported, selecting other options will result in plugin initialization errors.
      # The link to this metric is located in the "Service Monitoring" -> "TRPC Custom Monitoring" -> "xxx_limiter_polaris_request" section of the 123 platform.
      metrics_provider: m007
```
Please enable client/server rate limiting as needed. Note that in the example, the `polaris_limiter` interceptor is configured for the entire client/server, but you can also configure interceptors for individual services.

#### Configure Rate Limiting Policies In The Polaris Console.
Please refer to [polaris console](http://v2.polaris.woa.com/#/services/list?owner=chrisbchen&isAccurate=1&hostOperator=1&page=1) for more details.

## FAQ
Q1: What is the error code for overload?
- Overload protection: server returns `22`, client returns `124`.
- Polaris rate limiting: server returns `23`, client returns `123`.

