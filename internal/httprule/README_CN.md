# 关于 HttpRule

要支持 RESTful API，其难点在于如何将 Proto Message 里面的各个字段映射到 HTTP 请求/响应中。这种映射关系当然不是原生就存在的。

所以，我们需要定义一种规则，来规定具体如何映射，这种规则，就是 ***HttpRule*** ：

在 pb 文件中，我们通过 Option 选项：**trpc.api.http** 来指定 HttpRule，映射时 rpc 请求 message 里的“叶子”字段分**三种**情况处理：

1. 字段被 HttpRule 的 url path 引用：HttpRule 的 url path 引用了 rpc 请求 message 中的一个或多个字段，则 rpc 请求 message 的这些字段就通过 url path 传递。但这些字段必须是原生基础类型的非数组字段，不支持消息类型的字段，也不支持数组字段。

2. 字段被 HttpRule 的 body 引用：HttpRule 的 body 里指明了映射的字段，则 rpc 请求 message 的这些字段就通过 http 请求 body 传递。

3. 其他字段：其他字段都会自动成为 url 查询参数，而且如果是 repeated 字段，则支持同一个 url 查询参数多次查询。

**补充**：

1. 如果 HttpRule 的 body 里未指明字段，用 "*" 来定义，则没有被 url path 绑定的每个请求 message 字段都通过 http 请求的 body 传递。  

2. 如果 HttpRule 的 body 为空，则没有被 url path 绑定的每个请求 message 字段都会自动成为 url 查询参数。

## 关于 httprule 包

易见，HttpRule 的处理难点在于 url path 匹配。

首先，RESTful 请求的 url path 需要用一个模板统一起来：

```Go
 Template = "/" Segments [ Verb ] ;
 Segments = Segment { "/" Segment } ;
 Segment  = "*" | "**" | LITERAL | Variable ;
 Variable = "{" FieldPath [ "=" Segments ] "}" ;
 FieldPath = IDENT { "." IDENT } ;
 Verb     = ":" LITERAL ;
```

即 HttpRule 里的 url path 都必须按照这个模板。

```httprule```包提供 ```Parse``` 方法可以将任意 HttpRule 里指定的 url path 解析为模板 ```PathTemplate``` 类型。

```PathTemplate``` 类提供一个 ```Match``` 方法，可以从真实的 http 请求 url path 里匹配到变量的值。

***举例一：***

   如果 pb Option **trpc.api.http** 中指定的 HttpRule url path 为：```/foobar/{foo}/bar/{baz}```，其中 ```foo``` 和 ```baz``` 为变量。

   解析到模板：

   ```Go
      tpl, _ := httprule.Parse("/foobar/{foo}/bar/{baz}")
   ```

   匹配一个 http 请求中的 url path：

   ```Go
      captured, _ := tpl.Match("/foobar/x/bar/y")
   ```

   则：

   ```Go
      reflect.DeepEqual(captured, map[string]string{"foo":"x", "baz":"y"})
   ```

***举例二：***

   如果 pb Option **trpc.api.http** 中指定的 HttpRule url path 为：```/foobar/{foo=x/*}```，其中 ```foo``` 为变量。

   解析到模板：

   ```Go
      tpl, _ := httprule.Parse("/foobar/{foo=x/*}")
   ```

   匹配一个 http 请求中的 url path：

   ```Go
      captured, _ := tpl.Match("/foobar/x/y")
   ```

   则：

   ```Go
      reflect.DeepEqual(captured, map[string]string{"foo":"x/y"})
   ```
