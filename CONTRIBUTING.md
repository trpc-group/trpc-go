# 为 tRPC-Go 作出贡献

欢迎您 [提出问题](issues) 或 [merge requests](merge_requests)，建议您在为 tRPC-Go 作出贡献前先阅读以下 tRPC-Go 贡献指南。

### 代码规范
 
必须遵循 [腾讯 Golang 代码规范](https://git.woa.com/standards/go)。

### 提交日志编写规范

术语对照：

* 合并请求：Merge Request，简称 MR

当用户提交合并请求之后，提交日志实际上有两种：

1. 点入工蜂合并请求页面左上角显示的标题（title）以及描述（description）
2. 在一个合并请求下后续不断追加的各个 commit

目前 tRPC-Go 的开发采取压缩合并（Squash and Merge）的方式，因此上述第一种日志会作为最终的提交信息在主干上保留下来，这一种日志也是本规范所重点讨论的。

而对于第二种日志，建议贡献者只在每次提交时书写一行简要信息即可（无需加句号）。

对于第一种提交日志，用户需要在工蜂合并请求页面上点击编辑（edit）按钮进行修改，分为以下几个部分：

1. 标题（title）
2. 描述（description）

#### 标题规范

* 简要描述合并请求修改内容，尽量不超过 76 个半角字符
* 格式：`软件包：变更结果` 比如 `admin: check error before setting header in test`
  * 软件包：以主要受影响的软件包为前缀，跟随一个半角冒号和空格 `:␣`
    * 参考同目录内相似提交的标题
    * 多个包的协同修改可以使用 Shell 风格的表达式展开 `{pkgA,pkgB,pkgC}:␣`
    * 大规模修改（如批量格式化）可以使用“all”或使用“lsc”（Large-scale change）
  * 变更结果：内容应可将这句话填空使其通顺：“这个变更修改软件包以_________”
    * 使用一般现在时动词开头，中文不使用“了”字，同时由于不是完整句，第一个词无需大写，末尾不需要有句号/句点
    * 使用范围准确的动词，如尽量描述具体执行的动作，如“添加”、“修改”、“删除”等，同时避免使用“修复”、“解决”等表示愿望的动词
    * 当空间允许，并且目的简单时，可以在标题内包括对应内容。如：“降低 CPU 请求量以减少资源浪费”

**注意**：
* 如果你无法用一句话概括这次变更，这可能意味着你需要将提交拆分为更小单位
* 当 MR 还没有开发好，可以在开头加上 `[WIP]` 以告知 reviewer 不要 review，开发完成后及时移除该标志

标题示例：

```markdown
internal: define and internalize protocol name constants
admin: check error before setting header in test
robust: reject probability should be reverted
client: support setting of caller metadata through client config
test: assert additional error code for e2e test
plugin: avoid using reference to loop iterator variable
codec: message should be put back to pool
docs: specify that version is trpc framework version
lsc: cherry-pick to opensource
{rpcz, server}: add docs about how to inject root span for custom transport
```

#### 描述规范

描述大致可以分为正文和脚注：

##### 正文

正文可以分为三部分：

1. 背景：
  * 说明本次合并请求的背景和目的，应允许任意 Reviewer 在不依赖其他信息的情况下，理解变更并进行 CR
    * 举例来说，如果一次变更涉及性能改进，则应该提供变更前后的对比数据和测试方法，便于 Reviewer 判断和检验
  * 如有必要，提供相关文档，bug 等链接。注意链接对读者（而非仅 Reviewer）应长期可见
2. 变更目标：
  * 本合并请求期望解决的问题是什么
3. 变更内容：
  * 详细列出代码变更内容，和标题进行呼应
  * 设计到用户接口的变更或者重要变更，需要重点强调说明

**注意**：

* 正文开头不要逐字句地把标题完全重复一遍
* 长段落需要在中间换行，每行尽量不超过 76 个半角字符
* 正文内部完整句子的结尾要有句号/句点，段落结尾添加一个空行
* 如果你在编写正文时发现内容在逻辑上无法被标题覆盖，这可能意味着你的修改文不对题，应当拆分成更多的合并请求进行提交
* 在变更本身非常简单的情况下，以上正文可以进行缩减，只使用单一段落的几句话来完成

正文示例：

```markdown
Currently, if you want to include caller metadata information during selection,
you can only use client.WithCallerMetadata, as there is no way to append this
information through configuration. However, callee metadata can be modified
through configuration.

To align with callee metadata, this MR now supports setting caller metadata
through the configuration file.

Detailed changes: added CallerMetadata configuration and updated the README.
```

##### 脚注

目前工蜂在合入时会自动追加关联的 TAPD 单的脚注，因此贡献者不需要在脚注中手动填写相关信息，贡献者需要注意添加的脚注只有一条：

* 对本合并请求所解决的 issue 进行关闭：`close #xxx`
  * 注意 `close` 保持全小写状态
  * `close` 后面有一个空格
  * `#xxx` 中的 `xxx` 是相关 issue 的编号

脚注示例：

```markdown
close #947
```

如果没有关联的 issue，脚注留空即可。

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
5. 从您的开发分支提一个 MR 到主仓库的 master 分支上。
6. 参考上面提到的规范写好 MR 的标题和描述，MR 创建时可以忽略 TAPD（除非是确定已知的 TAPD 任务），MR 提交后，PMC 成员会辅助 TAPD 的创建/关联。
7. 经过 CR 完成后，Squash 合并进入主仓库 master 分支，此时开发分支已完成任务可以删除了。
8. 从主仓库 master 分支 Rebase 合并更新到您名下的 master 分支。
9. 重复以上 2~8 步骤，进入下一个特性开发周期。

## 试用 MR

提交 MR 后需要经过评审以及验证才能够合入，为了降低风险，推荐用户先用 replace 的方法对分支引入的特性/修复的 bug 进行验证，流程如下：

1. 将以下内容加入到用户自己仓库的 `go.mod` 中（假设提交 MR 的 fork 仓库为 `git.woa.com/somename/trpc-go`（通过 URL 链接提取出来类似的部分），分支名为 `somebranch`，这两个信息可以从工蜂 MR 界面里大标题的下方拿到）：
```shell
replace git.code.oa.com/trpc-go/trpc-go => git.woa.com/somename/trpc-go somebranch
```
2. 执行 `go mod tidy`，上述 `somebranch` 会自动更新为对应的 commit id
