English | [中文](metrics.zh_CN.md)

# How to develop a metric type plugin

This guide will introduce how to develop a metric type plugin that depends on configuration for loading.
The plugin will report the time it takes for the client to send a request to the server and receive a reply when initiating an RPC, as well as the time it takes for the server to receive a request and reply to the client.
To develop this plugin, the following three sub-functions need to be implemented:

- Implement the loading of the plugin's dependencies through configuration. For detailed instructions, please refer to [plugin](/plugin/README.md).
- Implement reporting metrics to an external platform. For detailed instructions, please refer to [metrics](/metrics/README.md).
- Implement reporting metrics in the filter. For detailed instructions, please refer to [filter](/filter/README.md).

The following will use [trpc-metrics-prometheus](https://github.com/trpc-ecosystem/go-metrics-prometheus) as an example to introduce the relevant development steps.

## Implement the loading of the plugin's dependencies through configuration

### 1. Determine the configuration of the plugin

```yaml
plugins:                                          # Plugin configuration
  metrics:                                        # Reference metrics
    prometheus:                                   # Start prometheus
      ip: 0.0.0.0                                 # Promethean binding address
      port: 8090                                  # Promethean binding port
      path: /metrics                              # Metrics path
      namespace: Development                      # Namespace
      subsystem: trpc                             # Subsystem
      rawmode:   false                            # Raw mode, special characters in metrics will not be converted
      enablepush: true                            # Enable push mode, not enabled by default
      gateway: http://localhost:9091              # Prometheus gateway address
      password: username:MyPassword               # Set account password, separated by colons
      job: job                                    # Job name
      pushinterval: 1                             # Push interval, default is 1s
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

### 2. Implement the `plugin.Factory` interface

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

### 3. Call `plugin.Register` to register the plugin with the plugin package

```go
func init() {
    plugin.Register(pluginName, &Plugin{})
}
```

## Implement reporting metrics to an external platform

### 1. Implement the `metrics.Sink` interface

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

### 2. Register the implemented Sink with the metrics package.

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

## Implement reporting metrics in the filters

### 1. Determine the configuration of the filter

```yaml
  filter:
    - prometheus                                   # Add prometheus filter
```

### 2. Implement `filter.ServerFilter` and `filter.ServerFilter`

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

### 3. Register the filter with the `filter` package

```go
func init() {
    filter.Register(pluginName, ServerFilter, ClientFilter)
}
```