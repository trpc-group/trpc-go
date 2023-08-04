# tRPC-Go 管理命令模块功能与实现
- Server 接口定义了基础的启动、关闭、路由设置与查询的基础功能
- 外部可传入对 adminConfig 的处理方式来对 admin 服务进行配置，如设置版本号，启用/禁用 TLS, 设置监听端口等
- admin 服务默认监听 9028 端口，未启用 TLS 连接，读写超时时间为 3s。可通过上述的方式自定义

# admin 部分接口解析
- `Run`: 启动 adminServer，监听指定端口，使用传入的参数调整配置，监听端口，接收外部请求
- `Close`: 关闭 adminServer
- `HandleFunc` 注册路由处理函数，不可覆盖

### 
* TLS support
*
