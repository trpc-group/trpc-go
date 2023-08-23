# How to Contribute

Thank you for your interest and support in tRPC!

We welcome and appreciate any form of contribution, including but not limited to submitting issues, providing improvement suggestions, improving documentation, fixing bugs, and adding features.
This document aims to provide you with a detailed contribution guide to help you better participate in the project.
Please read this guide carefully before contributing and make sure to follow the rules here.
We look forward to working with you to make this project better together!

## Becoming a contributor

The first step is registering as a tRPC-Go contributor.
Before sending your first change to the tRPC-Go project you must have [signed and submitted a CLA (Contributor License Agreement)](to-do).

## Before contributing code

The project welcomes code patches, but to make sure things are well coordinated you should discuss any significant change before starting the work.
It's recommended that you signal your intention to contribute in the issue tracker, either by claiming an [existing one](https://github.com/trpc-group/trpc-go/issues) or by [opening a new issue](https://github.com/trpc-group/trpc-go/issues/new).

### Checking the issue tracker

Whether you already know what contribution to make, or you are searching for an idea, the [issue tracker](https://github.com/trpc-group/trpc-go/issues) is always the first place to go.
Issues are triaged to categorize them and manage the workflow.

Most issues will be marked with one of the following workflow labels:
- **NeedsInvestigation**: The issue is not fully understood and requires analysis to understand the root cause.
- **NeedsDecision**: The issue is relatively well understood, but the tRPC-Go team hasn't yet decided the best way to address it.
  It would be better to wait for a decision before writing code.
  If you are interested in working on an issue in this state, feel free to "ping" maintainers in the issue's comments if some time has passed without a decision.
- **NeedsFix**: The issue is fully understood and code can be written to fix it.

### Opening an issue for any new problem

Excluding very trivial changes, all contributions should be connected to an existing issue.
Feel free to open one and discuss your plans.
This process gives everyone a chance to validate the design, helps prevent duplication of effort, and ensures that the idea fits inside the goals for the language and tools.
It also checks that the design is sound before code is written; the code review tool is not the place for high-level discussions.

When opening an issue, make sure to answer these five questions:
1. What version of tRPC-Go are you using ?
2. What operating system and processor architecture are you using(`go env`)?
3. What did you do?
4. What did you expect to see?
5. What did you see instead?

For change proposals, see Proposing Changes To [tRPC-Go-Proposals](to-do).

## Contributing code

Follow the [GitHub flow](https://docs.github.com/en/get-started/quickstart/github-flow) to [create a GitHub pull request](https://docs.github.com/en/get-started/quickstart/github-flow#create-a-pull-request).

Some things to keep in mind:
- Ensure that your code conforms to the project's code specifications.
  This includes but is not limited to code style, comment specifications, etc. This helps us to maintain the cleanliness and consistency of the project.
- Before submitting a PR, please make sure that you have tested your code locally(`go test ./...`).
  Ensure that the code has no obvious errors and can run normally.
- To update the pull request with new code, just push it to the branch;
  you can either add more commits, or rebase and force-push (both styles are accepted).
- If the request is accepted, all commits will be squashed, and the final commit description will be composed by concatenating the pull request's title and description.
  The individual commits' descriptions will be discarded.
  See following "Write good commit messages" for some suggestions.

### Writing good commit messages

Commit messages in tRPC-Go follow a specific set of conventions, which we discuss in this section.

Here is an example of a good one:


> math: improve Sin, Cos and Tan precision for very large arguments
>
> The existing implementation has poor numerical properties for
> large arguments, so use the McGillicutty algorithm to improve
> accuracy above 1e10.
>
> The algorithm is described at https://wikipedia.org/wiki/McGillicutty_Algorithm
>
> Fixes #159


#### First line

The first line of the change description is conventionally a short one-line summary of the change, prefixed by the primary affected package.

A rule of thumb is that it should be written so to complete the sentence "This change modifies tRPC-Go to _____."
That means it does not start with a capital letter, is not a complete sentence, and actually summarizes the result of the change.

Follow the first line by a blank line.

#### Main content

The rest of the description elaborates and should provide context for the change and explain what it does.
Write in complete sentences with correct punctuation, just like for your comments in tRPC-Go.
Don't use HTML, Markdown, or any other markup language.
Add any relevant information, such as benchmark data if the change affects performance.
The [benchstat](https://godoc.org/golang.org/x/perf/cmd/benchstat) tool is conventionally used to format benchmark data for change descriptions.

#### Referencing issues

The special notation "Fixes #12345" associates the change with issue 12345 in the tRPC-Go issue tracker.
When this change is eventually applied, the issue tracker will automatically mark the issue as fixed.


## Miscellaneous topics

### Copyright headers

Files in the tRPC-Go repository don't list author names, both to avoid clutter and to avoid having to keep the lists up to date.
Instead, your name will appear in the [change log](todo).

New files that you contribute should use the standard copyright header:

```go
// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC-Go source code is licensed under the Apache 2.0 License,
// A copy of the Apache 2.0 License can be found in the LICENSE file.
```

Files in the repository are copyrighted the year they are added.
Do not update the copyright year on files that you change.

---

# 如何贡献

感谢您对 tRPC-Go 的关注和支持！

我们欢迎并感激任何形式的贡献，包括但不限于提交 issue、提供改进建议、改进文档、修复错误和添加功能。
本文档旨在为您提供详细的贡献指南，以帮助您更好地参与项目。
在贡献之前，请仔细阅读本指南并确保遵循这里的规则。
我们期待与您共同努力，使这个项目变得更好！

## 成为贡献者

第一步是注册为 tRPC-Go 贡献者。
向 tRPC-Go 项目发送您的第一个变更之前，您必须签署并提交[贡献者许可协议](to-do)。

## 在贡献代码之前

项目欢迎代码补丁，但为了确保事情得到良好协调，您应该在开始工作之前讨论任何重大变更。
建议您在 issue 跟踪器中表明您的贡献意图，可以通过[认领现有 issue](https://github.com/golang/go/issues)或[创建新 issue](https://github.com/trpc-group/trpc-go/issues/new) 来实现。

### 查看 issue 跟踪器

无论您已经知道要做哪些贡献，还是正在寻找想法，[issue 跟踪器](https://github.com/trpc-group/trpc-go/issues)始终是您的第一个去处。
issue 会被分类以管理工作流程。

大多数 issue 都会被标记为以下工作流标签之一：
- **NeedsInvestigation**：issue 尚未完全理解，需要分析以了解根本原因。
- **NeedsDecision**：issue 相对已经理解得很好，但tRPC-Go团队尚未决定解决 issue 的最佳方法。
  在编写代码之前最好等待决策。
  如果一段时间内没有决策且您有兴趣处理处于这种状态的 issue，请随时在 issue 评论中“ping”维护者。
- **NeedsFix**：issue 已完全理解，可以编写代码进行修复。

### 为任何新问题打开一个 issue

除非是非常细小的变更，否则所有贡献都应与现有 issue 有关。
请随时打开一个 issue 并讨论您的计划。
这个过程让每个人都有机会验证设计，有助于防止工作重复，确保想法符合语言和工具的目标。
在编写代码之前，还可以检查设计是否合理；代码审查工具并非用于高层次的讨论。

在提交 issue 时，请确保回答以下五个问题：
1. 您正在使用哪个版本的tRPC-Go？
2. 您正在使用哪个操作系统和处理器架构（go env）？
3. 您做了什么？
4. 您期望看到什么？
5. 您实际看到的是什么？

关于变更提案，请参阅向 [tRPC-Go-Proposals](to-do) 提议变更。

## 贡献代码

遵循 [GitHub 流程](https://docs.github.com/en/get-started/quickstart/github-flow)来[创建 GitHub PR(Pull Request)](https://docs.github.com/en/get-started/quickstart/github-flow#create-a-pull-request)。

请记住以下几点：

- 确保您的代码符合项目的代码规范。
  这包括但不限于代码风格、注释规范等。这有助于我们维护项目的整洁性和一致性。
- 在提交 PR 之前，请确保您已在本地测试过您的代码。 确保代码没有明显的错误并且可以正常运行。
- 要使用新代码更新拉取请求，只需将其推送到分支； 您可以添加更多提交，也可以 rebase 并 force-push（两种风格都可以接受）。
- 如果请求被接受，所有提交将被压缩，最终提交描述将由 PR 的标题和描述组成。
  单个提交的描述将被丢弃。 请参阅以下“编写良好的提交消息”以获取一些建议。

### 编写良好的提交消息

tRPC-Go 中的提交消息遵循一套特定的约定，我们将在本节中讨论。

以下是一个良好的示例：
> math: improve Sin, Cos and Tan precision for very large arguments
>
> The existing implementation has poor numerical properties for large arguments, so use the McGillicutty algorithm to improve accuracy above 1e10.
>
> The algorithm is described at https://wikipedia.org/wiki/McGillicutty_Algorithm
>
> Fixes #159

#### 第一行

变更描述的第一行通常是一个简短的一行摘要，描述变更的内容，并以主要受影响的包为前缀。

一个经验法则是，它应该被写成完成句子 "This change modifies tRPC-Go to _____." 这意味着它不以大写字母开头，不是一个完整的句子，而且确实概括了变更的结果。

在第一行之后空一行。

#### 主要内容

描述的其余部分应该详细说明，为变更提供上下文并解释它的作用。
像在 tRPC-Go 中的注释一样，使用正确的标点符号写完整的句子。
不要使用 HTML、Markdown 或任何其他标记语言。
添加任何相关信息，例如如果变更影响性能，请添加基准数据。
[benchstat](https://godoc.org/golang.org/x/perf/cmd/benchstat)工具通常用于为变更描述格式化基准数据。

#### 引用 issue

特殊表示法 "Fixes #12345" 将变更与 tRPC-Go issue 跟踪器中的 issue 12345关联。
当此变更最终应用时，issue 跟踪器将自动将该 issue 标记为已修复。

## 其他主题

### 版权声明

tRPC-Go 代码仓库中的文件不列出作者姓名，以避免混乱并避免不断更新列表。
而您的名字将出现在变更日志中。

您贡献的新文件应使用标准版权声明：
```go
// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC-Go source code is licensed under the Apache 2.0 License,
// A copy of the Apache 2.0 License can be found in the LICENSE file.
```

代码仓库中的文件在添加时受版权保护。
在变更文件时，请勿更新版权年份。