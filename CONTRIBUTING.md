English | [中文](CONTRIBUTING.zh_CN.md)

# How to Contribute

Thank you for your interest and support in tRPC!

We welcome and appreciate any form of contribution, including but not limited to submitting issues, providing improvement suggestions, improving documentation, fixing bugs, and adding features.
This document aims to provide you with a detailed contribution guide to help you better participate in the project.
Please read this guide carefully before contributing and make sure to follow the rules here.
We look forward to working with you to make this project better together!

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

For change proposals, see Proposing Changes To [tRPC-Go-Proposals](https://github.com/trpc-group/trpc/tree/main/proposal).

## Contributing code

Follow the [GitHub flow](https://docs.github.com/en/get-started/quickstart/github-flow) to [create a GitHub pull request](https://docs.github.com/en/get-started/quickstart/github-flow#create-a-pull-request).
If this is your first time submitting a PR to the tRPC-Go project, you will be reminded in the "Conversation" tab of the PR to sign and submit the [Contributor License Agreement](https://github.com/trpc-group/cla-database/blob/main/Tencent-Contributor-License-Agreement.md).
Only when you have signed the Contributor License Agreement, your submitted PR has the possibility of being accepted.

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
Instead, your name will appear in the change log.

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