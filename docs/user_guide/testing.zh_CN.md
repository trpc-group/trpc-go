# tRPC-Go Testing
[api-testing](https://github.com/LinuxSuRen/api-testing) 是一个在无需编码即可帮助开发者和测试人员探索 API 的工具。这个开源的工具提供了两种使用方法：命令行和 Web 界面。支持的协议包括：`HTTP`、`gRPC`、`tRPC`.

这篇教程将告诉你如何使用这个工具的 `tRPC` 功能。

## 安装
它提供了多种安装方式，包括：Docker, Helm chart, Kubernetes Operator 等。考虑到 Docker 可能是最简单的一种方式，下面将会以 Docker 为例做介绍：

```shell
# 从 v0.0.14 开始支持 tRPC 功能
docker run -p 8080:8080 linuxsuren/api-testing:master
```

## 使用
当你的容器启动后，就可以通过下面的地址来访问了：

`http://localhost:8080`

![image](https://github.com/trpc-group/trpc-go/assets/1450685/fa9c66fc-ec5b-4a70-9466-f6d923aac229)
![image](https://github.com/trpc-group/trpc-go/assets/1450685/03347a59-51a4-43ed-aa44-7b0dec340b90)
![image](https://github.com/trpc-group/trpc-go/assets/1450685/12d1c7b0-bffd-4a48-adc4-5ed4a156afef)
![image](https://github.com/trpc-group/trpc-go/assets/1450685/0c264a5e-dec0-400a-a420-55af79dbedbd)

## 更多内容
[api-testing](https://github.com/linuxsuren/api-testing) 支持多种后端存储，默认会以本地文件的方式进行保存，适合体验的场景。但它也支持将数据保存到关系型数据库、Git 仓库中。欢迎查看官方文档了解更多内容。
