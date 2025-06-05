# 路由组件接口

client 后端路由选择器，通过 service name 获取一个节点，内部调用服务发现，负载均衡，熔断隔离

```go
// Selector 路由组件接口
type Selector interface {
    // Select 通过 service name 获取一个后端节点
    Select(serviceName string, opt ...Option) (*registry.Node, error)
    // Report 上报当前请求成功或失败
    Report(node *registry.Node, cost time.Duration, success error) error
}
```
