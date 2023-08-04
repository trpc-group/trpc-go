# 为 tRPC-Go 作出贡献

欢迎您 [提出问题](issues) 或 [merge requests](merge_requests)，建议您在为 tRPC-Go 作出贡献前先阅读以下 tRPC-Go 贡献指南。

### 代码规范
 
遵循[Google Golang 代码规范](https://google.github.io/styleguide/go/)。

### Commit Message 编写规范

每次提交，Commit message 都包括三个部分：Header，Body 和 Footer。
 
```html
<type>(<scope>): <subject>
// 空一行
<body>
// 空一行
<footer>
```
 
Header 部分只有一行，包括三个字段：type（必需）、scope（可选）和 subject（必需）。
 
type 用于说明 commit 的类别，只允许使用下面 7 个标识。
 
- feat：新功能（feature）
- fix：修补 bug
- docs：文档（documentation）
- style：格式（不影响代码运行的变动）
- refactor：重构（即不是新增功能，也不是修改 bug 的代码变动）
- test：增加测试
- chore：构建过程或辅助工具的变动
 
scope 用于指定 commit 影响的范围，比如数据层、控制层、视图层等等，或者具体所属 package。
 
subject 是 commit 目的的简短描述。
 
Body 部分是对本次 commit 的详细描述。
 
Footer 部分关闭 Issue。
 
如果当前 commit 针对某个 issue，那么可以在 Footer 部分关闭这个 issue。
 
```html
feat(transport): add transport stream support
 
- support only client stream 
- support only server stream 
- supoort both client and server stream
 
Close #1
```

### 分支管理

tRPC-Go 主仓库一共包含一个 master 分支和多个 release 分支：

release 分支

请勿在 release 分支上提交任何 MR。

master 分支

master 分支作为长期稳定的开发分支，经过测试后会在下一个版本合并到 release 分支。
MR 的目标分支应该是 master 分支。

```html
trpc-go/trpc-go/r0.1
 ↑ 经过测试之后，Create a merge commit 合并进入 release 分支，发布版本
trpc-go/trpc-go/master
 ↑ 开发者提出 MR，Squash and merge 合并进入主仓库 master 分支
your_repo/trpc-go/feature
 ↑ 创建临时特性开发分支
your_repo/trpc-go/master
 ↑ 主仓库 fork 到私人仓库
trpc-go/trpc-go/master
```

### MR 流程规范

对于所有的 MR，我们会运行一些代码检查和测试，一经测试通过，会接受这次 MR，但不会立即将代码合并到 release 分支上，会有一些延迟。

当您准备 MR 时，请确保已经完成以下几个步骤：

1. 将主仓库代码 Fork 到自己名下。
2. 基于您名下的 master 分支创建您的临时开发分支，并在该开发分支上开始编码。
3. 检查您的代码语法及格式，确保完全符合腾讯 Golang 代码规范。
4. 提 MR 之前，首先从主仓库 master 分支 MR 到您的个人开发分支上，保证代码是最新的。
5. 从您的开发分支提一个 MR 到主仓库的 master 分支上，指定相应模块的 owner 为 reviewer。
6. MR 标题详细写明功能点。
7. 经过 CR 完成后，Squash 合并进入主仓库 master 分支，此时开发分支已完成任务可以删除了。
8. 从主仓库 master 分支 Rebase 合并更新到您名下的 master 分支。
9. 重复以上 2~8 步骤，进入下一个特性开发周期。
