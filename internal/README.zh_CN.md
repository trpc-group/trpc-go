# tRPC-Go 框架内部数据逻辑

| 类型 | 描述 |
| :----: | :----   |
| env | 环境变量定义 |
| httprule | 解析 RESTful URL |
| packetbuffer | 用于操纵 byte slice |
| rand | 提供协程安全的随机函数 |
| report | 内部异常分支监控上报 |
| ring | 提供并发安全的环形队列 |
| stack | 提供非并发安全的栈实现 |
| writev | 提供 writev 批量发送 Buffer |
