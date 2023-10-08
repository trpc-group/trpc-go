# tRPC-Go 服务路由

服务路由通过在服务发现和负载均衡之间通过服务路由规则对服务节点进行过滤，达到选择特定节点的能力。

## 服务注册接口
```go
// ServiceRouter 服务路由接口
type ServiceRouter interface {
	Filter(serviceName string, nodes []*registry.Node, opt ...Option) ([]*registry.Node, error)
}
```
自定义实现参考项目内部的实现。

