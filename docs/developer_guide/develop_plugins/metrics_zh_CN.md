# 怎么开发一个 metric 类型的插件

本指南将介绍如何开发一个依赖配置进行加载的 metric 类型的插件。
该插件将上报发起 RPC 时，client 端发送请求到 server 端收到回复的耗时， 以及 server 端收到请求到回复 client 的耗时。
开发该插件需要实现以下三个子功能：

- 实现插件依赖配置进行加载，详细说明请参考 [plugin](/plugin/README_CN.md)
- 实现让监控指标上报到外部平台，详细说明请参考 [metrics](/metrics/README_CN.md)
- 实现在拦截器中上报监控指标，详细说明请参考 [filter](/filter/README_CN.md)

下面以 [trpc-metrics-prometheus](https://github.com/trpc-ecosystem/go-metrics-prometheus) 为例，来介绍相关开发步骤。

## 实现插件依赖配置进行加载

### 1. 确定插件的配置

```yaml
plugins:                                          # 插件配置
  metrics:                                        # 引用metrics
    prometheus:                                   # 启动prometheus
      ip: 0.0.0.0                                 # prometheus绑定地址
      port: 8090                                  # prometheus绑定端口
      path: /metrics                              # metrics路径
      namespace: Development                      # 命名空间
      subsystem: trpc                             # 子系统
      rawmode:   false                            # 原始模式，不会对metrics的特殊字符进行转换 
      enablepush: true                            # 启用push模式，默认不启用
      gateway: http://localhost:9091              # prometheus gateway地址
      password: username:MyPassword               # 设置账号密码， 以冒号分割
      job: job                                    # job名称
      pushinterval: 1                             # push间隔，默认1s上报一次
```

```go
const (
    pluginType = "metrics"
    pluginName = "prometheus"
)

type Config struct {
    IP           string `yaml:"ip"`           // metrics monitoring address.
    Port         int32  `yaml:"port"`         // metrics listens to the port.
    Path         string `yaml:"path"`         // metrics path.
    Namespace    string `yaml:"namespace"`    // formal or test.
    Subsystem    string `yaml:"subsystem"`    // default trpc.
    RawMode      bool   `yaml:"rawmode"`      // by default, the special character in metrics will be converted.
    EnablePush   bool   `yaml:"enablepush"`   // push is not enabled by default.
    Password     string `yaml:"password"`     // account Password.
    Gateway      string `yaml:"gateway"`      // push gateway address.
    PushInterval uint32 `yaml:"pushinterval"` // push interval,default 1s.
    Job          string `yaml:"job"`          // reported task name.
}
```

### 2. 实现 `plugin.Factory` 接口

```go
type Plugin struct {
}

func (p *Plugin) Type() string {
    return pluginType
}

func (p *Plugin) Setup(name string, decoder plugin.Decoder) error {
    cfg := Config{}.Default()
    
    err := decoder.Decode(cfg)
    if err != nil {
        log.Errorf("trpc-metrics-prometheus:conf Decode error:%v", err)
        return err
    }
    go func() {
        err := initMetrics(cfg.IP, cfg.Port, cfg.Path)
        if err != nil {
            log.Errorf("trpc-metrics-prometheus:running:%v", err)
        }
    }()
    
    initSink(cfg)
    
    return nil
}
```

### 3. 调用 `plugin.Register` 把插件自己注册到 `plugin` 包

```go
func init() {
    plugin.Register(pluginName, &Plugin{})
}
```

## 让监控指标上报到外部平台

### 1. 实现 `metrics.Sink` 接口

```go
const (
    sinkName = "prometheus"
)

func (s *Sink) Name() string {
    return sinkName
}

func (s *Sink) Report(rec metrics.Record, opts ...metrics.Option) error {
    if len(rec.GetDimensions()) <= 0 {
        return s.ReportSingleLabel(rec, opts...)
}
    labels := make([]string, 0)
    values := make([]string, 0)
    prefix := rec.GetName()
    
    if len(labels) != len(values) {
        return errLength
    }

    for _, dimension := range rec.GetDimensions() {
        labels = append(labels, dimension.Name)
        values = append(values, dimension.Value)
    }
    for _, m := range rec.GetMetrics() {
        name := s.GetMetricsName(m)
        if prefix != "" {
            name = prefix + "_" + name
        }
        if !checkMetricsValid(name) {
            log.Errorf("metrics %s(%s) is invalid", name, m.Name())
            continue
        }
        s.reportVec(name, m, labels, values)
    }
    return nil
}
```

### 2. 将实现的 Sink 注册到 metrics 包。

```go
func initSink(cfg *Config) {
    defaultPrometheusPusher = push.New(cfg.Gateway, cfg.Job) 
    // set basic auth if set. 
    if len(cfg.Password) > 0 { 
        defaultPrometheusPusher.BasicAuth(basicAuthForPasswordOption(cfg.Password))
    }
    defaultPrometheusSink = &Sink{
        ns:         cfg.Namespace,
        subsystem:  cfg.Subsystem,
        rawMode:    cfg.RawMode,
        enablePush: cfg.EnablePush,
        pusher:     defaultPrometheusPusher
    }
    metrics.RegisterMetricsSink(defaultPrometheusSink)
    // start up pusher if needed.
    if cfg.EnablePush {
    defaultPrometheusPusher.Gatherer(prometheus.DefaultGatherer)
        go pusherRun(cfg, defaultPrometheusPusher)
    }
}
```

## 在拦截器中上报监控指标

### 1. 确定拦截器的配置

```yaml
  filter:
    - prometheus                                   # Add prometheus filter
```

### 2. 实现 `filter.ServerFilter` 和 `filter.ServerFilter` 

```go
func ClientFilter(ctx context.Context, req, rsp interface{}, handler filter.ClientHandleFunc) error {
	begin := time.Now()
	hErr := handler(ctx, req, rsp)
	msg := trpc.Message(ctx)
	labels := getLabels(msg, hErr)
	ms := make([]*metrics.Metrics, 0)
	t := float64(time.Since(begin)) / float64(time.Millisecond)
	ms = append(ms,
		metrics.NewMetrics("time", t, metrics.PolicyHistogram),
		metrics.NewMetrics("requests", 1.0, metrics.PolicySUM))
	metrics.Histogram("ClientFilter_time", clientBounds)
	r := metrics.NewMultiDimensionMetricsX("ClientFilter", labels, ms)
	_ = GetDefaultPrometheusSink().Report(r)
	return hErr
}

func ServerFilter(ctx context.Context, req interface{}, handler filter.ServerHandleFunc) (rsp interface{}, err error) {
	begin := time.Now()
	rsp, err = handler(ctx, req)
	msg := trpc.Message(ctx)
	labels := getLabels(msg, err)
	ms := make([]*metrics.Metrics, 0)
	t := float64(time.Since(begin)) / float64(time.Millisecond)
	ms = append(ms,
		metrics.NewMetrics("time", t, metrics.PolicyHistogram),
		metrics.NewMetrics("requests", 1.0, metrics.PolicySUM))
	metrics.Histogram("ServerFilter_time", serverBounds)
	r := metrics.NewMultiDimensionMetricsX("ServerFilter", labels, ms)
	_ = GetDefaultPrometheusSink().Report(r)
	return rsp, err
}
```

### 3. 将拦截器注册到 `filter` 包

```go
func init() {
    filter.Register(pluginName, ServerFilter, ClientFilter)
}
```
