# tRPC-Go 服务注册

服务注册接口通过与服务注册中心交互，注册节点维护服务健康状态。

## 服务注册接口
```go
// Registry 服务注册接口
type Registry interface {
	Register(service string, opt ...Option) error
	Deregister(service string) error
}
```
自定义实现参考项目内部的实现。

