# Robust

本目录展示了 `"git.woa.com/trpc-go/trpc-robust"` 过载保护插件的使用示例。

其中 `server` 目录下的 `trpc_go.yaml` 需要使用 `trpc-robust` 拦截器并加上相关插件配置以启用过载保护，关键配置如下：

```yaml
server:
  filter:
    - trpc-robust
  service:
    - name: trpc.test.helloworld.Greeter
      port: 8000
      # ...

plugins:
  overload_control:
    trpc-robust:
      server:
        update_every_requests: 100  # 每次处理这么多请求判断一次服务是否过载
        update_duration: 10s  # 每经过这么长时间强制判断一次服务是否过载，为了处理请求量较少的情况
        start_overload_ms: 2  # 认为排队时间超过次数量服务就过载
        point_per_ms: 30  # 超过排队时间阈值 (start_overload_ms) 后每一毫秒对应的负载点数，一般不需要更改
        overload_recover_fail_count: 3  # 从过载状态恢复时，假如排队时间的增加次数超过这个配置，则判断为仍处于过载状态，一般不需要更改
        start_overload_cpu_usage: 0.75  # CPU 利用率高于此值服务才过载，防止 GC STW 导致排队时间错误误判服务过载，取值区间 (0,1)
        cpu_usage_interval: 1s  # CPU 利用率采集的时间范围，一般不需要更改
        report_enabled: true  # 是否上报数据到柔性治理平台
      client:
        overload_error_codes: [22,23]  # 判断下游是否过载的错误码
        start_overload_success_rate: 0.5  # 开始过载的成功率，低于此值认为下游过载，取值区间 (0,1)
        window: 1s  # 统计时间窗口大小
        max_reject_rate: 0.99  # 最大拒绝概率，取值范围 [0,1]，一般不需要更改
        start_working_request: 300  # 在窗口期，请求量少于此值主调过载保护不生效
        report_enabled: true  # 是否上报数据到柔性治理平台
      rank:
        max_rank: 256  # 最大的请求重要程度，一般取默认值即可
```

而 `client` 目录则实现了有如下特征的流量请求（山形流量）：

```go
// /                  peak QPS    +-----------+
// /                             /|           |\
// /                            / |           | \
// /                           /  |           |  \
// / initial QPS +------------+   |           |   +---------------+
// /             |            |   |           |   |               |
// /            initial duration  keep duration   die down duration
// /                          |   |           |   |
// /                   change duration     change duration
```

运行方法：

1. 首先清理环境

```shell
./cleanup.sh # 清理环境
./removelogs.sh # 清理日志文件
```

2. 运行镜像

```shell
./run.sh
```

这一步会运行 `prometheus`,`grafana`,`robust-server`,`robust-client` 这四个镜像。

这几个镜像分别绑核 `0`, `1`, `2,3`, `4-7`，共占 8 核，其中服务端 2 核，客户端 4 核。

这些客户端和服务端推荐按照脚本的方式在容器中进行测试，如果直接运行的话，会因为整体 CPU 利用率不足而无法触发 robust 插件生效。

服务端（`robust-server`）与客户端（`robust-client`）会分别上报数据到配置的 `prometheus` 中，最后可以在 `grafana` 里显示出来。

在 `robust-client` 镜像启动时，它就会自动执行上述特征的流量发送，大概持续两分钟后稳定在一个较低 QPS 上。

然后执行以下命令以关闭服务端的 robust 插件，并重新构造客户端的山形流量：

```shell
./disable_robust.sh && ./tune_restart.sh 
```

同样经过两分钟后，流量稳定在一个较低的 QPS 上。

3. 查看监控

为了方便演示，这里的监控以及展示使用了 `prometheus` 以及 `grafana`，通过端口映射以访问 `grafana` 的 `3000` 端口（如 `http://127.0.0.1:3000`），通过默认账户进行登录，然后导入 dashboard 配置：

[trpc-robust-dashboard.json](/.resources/examples/robust/trpc-robust-dashboard.json)

然后可以观察到类似于下图的数据：

![trpc-robust](/.resources/examples/robust/trpc-robust.png)

主要可以关注到在开启 robust 之后，P99 耗时可以始终维持在较低的水平。
