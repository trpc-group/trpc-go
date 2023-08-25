# 前言

本文介绍如何开发监控插件，具体细节可参考 [m007](https://git.woa.com/trpc-go/trpc-metrics-m007/tree/master)代码，模调监控使用的是框架的拦截器能力，需要先了解框架的[filter](https://git.woa.com/trpc-go/trpc-go/tree/master/filter)和[metrics](https://git.woa.com/trpc-go/trpc-go/tree/master/metrics) 。

阅读本篇文章之前，需要先阅读[开发拦截器插件](./interceptor.md)。

# 原理

利用 trpc-go 的插件能力，整体功能包含：

- 模调上报：在请求后上报接口的详细情况，一般包含主调（client 上报）、被调（被调 server 上报）；
- 属性上报：包含累积量、时刻量、多维度的监控项。

具体细节依赖监控平台的支持，插件是用来适配框架和监控 SDK 接口。

# 具体实现

## 主调与被调

注册插件，pluginName 自定义，m007Plugin 要满足接口的定义。

``` go
const (
   pluginName = "m007"
)
func init() {
   plugin.Register(pluginName, &m007Plugin{})
}
```

插件初始化，若配置文件配置 pluginName 值，框架会在初始化时调用注册插件的`Setup`方法。
主要做一些初始化逻辑，比如：依赖监控 SDK 的初始化、filter 的注册等等。

``` go
// 解析配置，SDK初始化
...
// 注册主调、被调
filter.Register(name, PassiveModuleCallServerFilter, ActiveModuleCallClientFilter)
```

实现对应的 filter，具体细节可看代码，
不过要注意，插件从接口返回的 err 读取错误码，非框架的 errs，类型转换失败，此时统一上报固定值。
其他字段具体看特定监控平台要求，插件统一从`msg := trpc.Message(ctx)`里去取，如遇到某些字段没有值，检查自定义协议的 codec 有没有设置对应值，插件本身不理解。

``` go
// ActiveModuleCallClientFilter  主调模调上报拦截器:自身调用下游，下游回包时上报
func ActiveModuleCallClientFilter(ctx context.Context, req, rsp interface{}, handler filter.HandleFunc) error {
   begin := time.Now()
   err := handler(ctx, req, rsp)
   msg := trpc.Message(ctx)
   activeMsg := new(pcgmonitor.ActiveMsg)
   // 自身服务
   activeMsg.AService = msg.CallerService() 
   ...
   // 下游服务
   activeMsg.PApp = msg.CalleeApp()
   ...
   // 错误码
   ...
   // 耗时ms
   activeMsg.Time = float64(time.Now().Sub(begin) / time.Millisecond)
   // 调用监控SDK上报
   pcgmonitor.ReportActive(activeMsg)
   return err
}
```

## metrics 属性上报

定义具体的 sink，然后注册 metrics，放到插件的 setup 内部执行。

``` go
// 注册metrics
metrics.RegisterMetricsSink(&M007Sink{})
```

适配框架接口，使用框架接口上报的监控项会循环调用所有注册的 sink 的`Report`方法，具体实现依赖监控平台本身的支持。比如：007 的属性上报是全策略上报，这里就不区框架具体的策略。

``` go
func (m *M007Sink) Report(rec metrics.Record, opts ...metrics.Option) error {
   if len(rec.GetDimensions()) <= 0 {
      // 属性上报
      for _, metric := range rec.GetMetrics() {
         pcgmonitor.ReportAttr(metric.Name(), metric.Value()) // 007属性全策略上报
      }
      return nil
   }
   // 多维度上报
   var dimesions []string
   var statValues []*nmnt.StatValue
   ...
   pcgmonitor.ReportCustom(rec.Name, dimesions, statValues)
   return nil
}
```

