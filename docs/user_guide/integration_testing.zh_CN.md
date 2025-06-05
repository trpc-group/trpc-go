## 前言

在集成测试之前，单元测试应该已经完成。集成测试是在单元测试的基础上，将框架的各个模块组装子系统后，测试是否达到或实现相应技术指标及要求。一些模块虽然能够单独地工作，但并不能保证连接起来也能正常的工作。局部反映不出来的问题，在全局上很可能暴露出来。使用集成测试保证框架在开发迭代中整个功能的完整性和正确性。

## 原理 & 实现

### 跨机器的集成测试

#### 测试过程

根据框架/组件的每一个特性，用 [trpc-cli](https://git.woa.com/trpc-go/trpc-cli) 作为触发工具，构建[主调服务](https://git.woa.com/trpc/trpc-plugin-testing/proxy/proxy-go)和[被调服务](https://git.woa.com/trpc/trpc-plugin-testing/feature/feature-go) ，trpc-cli 触发主调服务，主调服务通过调用被调服务，trpc-cli 工具根据返回的结果判断是否符合测试要求。

```text
trpc-cli 解析配置 (test.data.json) → 主调服务 (proxy) → 被调服务 (features)
```

#### 测试范围

集成测试的范围包括框架的主要特性，以及常用的插件，测试用例可以参考[这里](https://iwiki.woa.com/pages/viewpage.action?pageId=517352692)，根据需要后续可以不断的增加测试用例。

#### 触发方式

集成测试的触发方式：定时触发和变更触发。每天定时运行集成测试用例，当框架或者组件的代码发生变动的时候触发。每次触发都需要使用最新的代码。

#### 流水线

整个集成测试都通过蓝盾[流水线](https://devops.woa.com/pipeline/pcgtrpcproject/p-7fdb19384bf348038eee20ea32215369/history)来控制。

#### 测试结果

测试结果在[datatalk](https://beacon.woa.com/datatalk/pg/dashboard/192610)展示。

### 单机上的集成测试

trpc-go 主库的许多特性可以在单机上进行测试，具体的原理和实现见[这里](../../test/README.md)

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
