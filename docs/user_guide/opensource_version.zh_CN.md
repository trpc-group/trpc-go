## 前言

随着公司中的使用 tRPC-Go 的业务逐渐增多，对于出海业务，越来越多的合作方也需要使用 tRPC-Go。为了方便公司外部的合作伙伴使用，我们目前已将 tRPC-Go 开源。

开源版本 tRPC-Go 仓库地址：https://github.com/trpc-group/trpc-go

开源版本 tRPC-Go 文档主页：https://trpc.group/docs/languages/go/

## 使用指南

参考开源版本的 tRPC-Go [快速上手](https://trpc.group/docs/languages/go/quick_start/)

## 开源插件

tRPC-Go 框架和插件内外网版本需要一一对应才能使用，也就是说，内网版本的插件只适用于内网 tRPC-Go，不适用于开源版本 tRPC-Go；开源版本插件只适用于开源版本 tRPC-Go，不适用于内网版本 tRPC-Go。

原因是插件在注册的时候，是向指定版本的 tRPC-Go 注册的，例如内网版本的插件，是向 git.code.oa.com/trpc-go/trpc-go/plugin 注册，当用户使用开源版本 tRPC-Go 时，从 trpc.group/trpc-go/trpc-go/plugin 是没法获取注册进来的插件的。

完整的开源 tRPC-Go 插件见 [trpc-ecosystem 仓库](https://github.com/orgs/trpc-ecosystem/repositories?q=&type=all&language=go&sort=)。

## 内外部版本管理

内部目前还有大量的用户，不维护或者推动迁移开源版本是几乎不可能的事情。长期来看，希望能够只维护一个版本，大概率是以开源版本为主（未明确期限）；

目前我们会对 bug，feature 内外进行 cherry-pick 同步，尽可能保证一致；

短期内，内部和外部的 tag 分开管理。

## 开源切换

1. 首先找出 go.mod 里面和 `git.code.oa.com/trpc-go/xxx` 相关的组件，从 go.mod 里面删除，然后把所有的 indirect 依赖也删除（后续 go mod tidy 会自动拉）
2. 然后把项目做全局的查找和替换，将所有 xx.go 文件开头 import 的 `git.code.oa.com/trpc-go/` 修改成 `trpc.group/trpc-go/`，注意这一步不是在 go.mod 里面加 replace 语句，而是全局的字符串替换
3. 执行 go mod tidy，自动拉取仓库

如果最后有些仓库报错，可能是这些仓库没有对应 github 版本，可以再考虑对这些仓库进行开源，示例脚本如下：

```shell
#!/bin/bash

# 定义要替换的旧仓库和新仓库
OLD_REPO="git.code.oa.com/trpc-go/"
NEW_REPO="trpc.group/trpc-go/"

# 步骤 1: 从 go.mod 中删除含有 $OLD_REPO 的行，但是不删除以 'module' 开头的行
grep -E "^module" go.mod > go.mod.tmp
grep -E -v "^module" go.mod | grep -v $OLD_REPO >> go.mod.tmp
mv go.mod.tmp go.mod

# 步骤 2: 全局查找和替换
# 遍历当前目录下的所有 .go 文件，将旧仓库替换为新仓库
# 处理北极星特例
OLD_POLARIS="git.code.oa.com/trpc-go/trpc-naming-polaris"
NEW_POLARIS="trpc.group/trpc-go/trpc-naming-polarismesh"
find . -type f -name "*.go" | xargs sed -i "s|$OLD_POLARIS|$NEW_POLARIS|g"
find . -type f -name "*.go" | xargs sed -i "s|$OLD_REPO|$NEW_REPO|g"

# 步骤 3: 执行 go mod tidy
go mod tidy

echo "操作完成。"
```

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
