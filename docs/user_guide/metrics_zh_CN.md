# 前言

metrics 定义了一些常用的指标监控，比如 counter、gauge、timer、histogram，接下来介绍下大致的实现原理、使用。

# 原理 & 实现

## counter

counter 就是个计数器，一般用于统计累积量。

现在的 counter 实现，统计量是用的浮点，为了并发操作时效率更高，希望能通过 CAS 操作代替 Mutex 操作，所以将数值用一个整型和一个浮点型对应的 float64bits 两部分组成。counter 的实际值为整型和浮点部分对应的 float64bits 两部分的和。

ps：后期可能会将类似的统计直接放入各个插件的 sink 中去累积上报。不再在 counter 实现中自己累计。因为有些监控系统需要把原始数据上报到后端，框架内部做累计削弱了扩展性。

## gauge

gauge 就是一个瞬时值，一般用于统计时刻量。

现在的 gauge 实现，统计量用的是浮点，因为是瞬时值，直接使用 CAS 操作更新即可。

ps：后期可能会将类似的统计直接放入各个插件的 sink 中去累积上报。不再在 counter 实现中自己累计。因为有些监控系统需要把原始数据上报到后端，框架内部做累计削弱了扩展性。

## timer

timer 是根据一个操作的开始时间、结束时间，统计某个操作的耗时情况。以 timer.RecordAction(fn) 为例，它会在执行 fn 之前记录一个时间 t，在 fn 执行结束之后计算一下 time.Since(t) 获得耗时信息。

ps：后期可能会将类似的统计直接放入各个插件的 sink 中去累积上报。不再在 counter 实现中自己累计。因为有些监控系统需要把原始数据上报到后端，框架内部做累计削弱了扩展性。

## histogram

histogram 是根据预先划分好的 buckets，将收集到的样本点放入到对应的 bucket 中，这样就方便查看不同区间（bucket 的上下界）的样本数量、平均值、最大值、最小值、和等等。各个区间的具体值由统计策略决定。

histogram 的 sink 实现会将 histogram 转换成平台特定的指标用于展示。

ps：后期可能会将类似的统计直接放入各个插件的 sink 中去累积上报。不再在 counter 实现中自己累计。因为有些监控系统需要把原始数据上报到后端，框架内部做累计削弱了扩展性。

# 使用示例

counter、gauge、timer、histogram 的使用都是一样的，具体到各个监控平台的话，差别只在于导入哪个监控的插件实现包，以及配置文件中该如何配置。

下面针对各个平台的使用、配置给个简单的说明。

## 示例代码


```go
import "git.code.oa.com/trpc-go/trpc-go/metrics"

// example1: counter
metrics.Counter("total.req").Incr()
metrics.IncrCounter("total.req", 1)

// example2: guage
metrics.Gauge("cpu.avg.load").Set(0.7)
metrics.SetGauge("cpu.avg.load", 0.7)

// example3: timer
metrics.Timer("rpc.timecost").RecordAction(func() {
	// do something
})

t := metrics.Timer("rpc.timecost");

// reset timer & do something1
t = t.Reset()
...
t.Record()

// reset timer & do something2
t = t.Reset()
...
t.Record()

// example4: histogram
h := metrics.Histogram("req.timecost.hist", metrics.WithBuckets(...))
h.AddSample(time.Second)
h.AddSample(time.Second * 2)
h.AddSample(time.Second * 3)
...
```

## 配置

日常编码时可以按照示例代码中的示例进行编码，根据运营体系中监控平台的不同，我们还需要导入不同的包实现，并进行正确的配置。

监控请参考格式 trpc-go/trpc-metrics-${name} 进行检索。

