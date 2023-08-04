# tRPC-Go 拦截器功能及实现 

框架在 server 业务处理函数 handler 前后，和 client 网络调用函数 call 前后，分别预埋了四个点允许用户自定义任何的逻辑来拦截处理。

## 自定义一个拦截器

### 定义处理逻辑函数

```golang
func ServerFilter() filter.Filter {
    return func(ctx context.Context, req interface{}, next filter.HandleFunc) (interface{}, error) {
        
        //前置逻辑
        
        rsp, err = next(ctx, req, rsp)
        
        //后置逻辑
        
        return rsp, err //返回error给上层
    }
}
```
```golang
func ClientFilter() filter.Filter {
    return func(ctx context.Context, req, rsp interface{}, next filter.HandleFunc) (err error) {
        
        //前置逻辑
        
        err = next(ctx, req, rsp)
        
        //后置逻辑
        
        return err //返回error给上层
    }
}
```

### 注册到框架中
```golang
filter1 := ServerFilter()
filter2 := ClientFilter()

filter.Register("name", filter1, filter2)
```

### 配置文件开启使用
```yaml
server:
 ...
 filter:
  ...
  - name 

client:
 ...
 filter:
  ...
  - name 
```