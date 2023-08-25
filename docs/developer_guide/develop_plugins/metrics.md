[TOC]


# Introduction

This article introduces how to develop monitoring plugins, for details, please refer to [m007](https://git.woa.com/trpc-go/trpc-metrics-m007/tree/master) code
Modular monitoring uses the filters of the framework, and it is necessary to understand the framework's [filter](https://git.woa.com/trpc-go/trpc-go/tree/master/filter) and [metrics](https://git.woa.com/trpc-go/trpc-go/tree/master/metrics).

Before reading this article, you need to read [developing filters plugins](./interceptor.md).

# Concept

Using the plugin capability of trpc-go, the overall function includes:

- Modulo report: report the details of the interface after the request, generally including the main caller (reported by the client), the callee (reported by the called server);
- Attribute reporting: monitoring items including cumulative quantity, time quantity and multi-dimension metrics.

The specific details depend on the support of the monitoring platform, because the plugin is used to adapt the framework and monitor the SDK interface.

# Implementation

## Caller&Callee

Register the plugin, pluginName custom, m007Plugin must meet the interface definition.

``` go
const (
   pluginName = "m007"
)
func init() {
   plugin.Register(pluginName, &m007Plugin{})
}
```

Initialize the plugin, if the pluginName value is configured in the configuration file, the framework will call the registered plugin's `Setup` method when initializing.

Mainly do some initialization logic, such as: initialization of dependent monitoring SDK, registration of filter, etc.

``` go
// Parse the configuration and initialize the SDK
... 
// Register the main caller and callee
filter.Register(name, PassiveModuleCallServerFilter, ActiveModuleCallClientFilter)
```

Implement the corresponding filter, see the code for details.

However, it should be noted that the `err` returned by the plugin's interface isn't  the errs of the non-framework.
If the type conversion fails, upload a uniform fixed value.

For other fields, please refer to the requirements of the specific monitoring platform. 
The plugin is uniformly fetched from `msg := trpc.Message(ctx)`. 
If some fields have no value, check whether the codec of the custom protocol has set the corresponding value, and the plugin itself don't understand.

``` go
// ActiveModuleCallClientFilter  caller modulation report filter: call downstream by itself, and report when the downstream returns the packet
func ActiveModuleCallClientFilter(ctx context.Context, req, rsp interface{}, handler filter.HandleFunc) error {
   begin := time.Now()
   err := handler(ctx, req, rsp)
   msg := trpc.Message(ctx)
   activeMsg := new(pcgmonitor.ActiveMsg)
   // self service
   activeMsg.AService = msg.CallerService() 
   ...
   // downstream service
   activeMsg.PApp = msg.CalleeApp()
   ...
   // error code
   ...
   // cost time
   activeMsg.Time = float64(time.Now().Sub(begin) / time.Millisecond)
   // call the monitoring SDK to report 
   pcgmonitor.ReportActive(activeMsg)
   return err
}
``` 

## metrics attribute reporting

Define a specific sink, then register metrics, and put them into the plugin's `Setup` for execution.

``` go
// Register metrics
metrics.RegisterMetricsSink(&M007Sink{})
```

Adapt to the framework interface, and the monitoring items reported using the framework interface will call the Report method of all registered sinks cyclically,
and the specific implementation depends on the support of the monitoring platform itself.

For example: 007's attribute report is a full-strategy report, and the specific strategy of the framework is not discussed here.

``` go
func (m *M007Sink) Report(rec metrics.Record, opts ...metrics.Option) error {
   if len(rec.GetDimensions()) <= 0 {
      // attribute report
      for _, metric := range rec.GetMetrics() {
         pcgmonitor.ReportAttr(metric.Name(), metric.Value()) // 007's attribute full-strategy report
      }
      return nil
   }
   // multi-dimension report
   var dimesions []string
   var statValues []*nmnt.StatValue
   ...
   pcgmonitor.ReportCustom(rec.Name, dimesions, statValues)
   return nil
}

```

# OWNER

## luisyjliu