English | [中文](README.zh_CN.md)

# tRPC-Go Framework

[![Go Reference](https://pkg.go.dev/badge/github.com/trpc.group/trpc-go.svg)](https://pkg.go.dev/github.com/trpc.group/trpc-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/trpc.group/trpc-go/trpc-go)](https://goreportcard.com/report/github.com/trpc.group/trpc-go/trpc-go)
[![LICENSE](https://img.shields.io/github/license/trpc.group/trpc-go.svg?style=flat-square)](https://github.com/trpc.group/trpc-go/blob/main/LICENSE)
[![Releases](https://img.shields.io/github/release/trpc.group/trpc-go.svg?style=flat-square)](https://github.com/trpc.group/trpc-go/releases)
[![Docs](https://img.shields.io/badge/docs-latest-green)](http://test.trpc.group.woa.com/docs/)
[![Tests](https://github.com/trpc.group/trpc-go/actions/workflows/prc.yaml/badge.svg)](https://github.com/trpc.group/trpc-go/actions/workflows/prc.yaml)
[![Coverage](https://codecov.io/gh/trpc.group/trpc-go/branch/main/graph/badge.svg)](https://app.codecov.io/gh/trpc.group/trpc-go/tree/main)


tRPC-Go, as the [Go][] language implementation of [tRPC][], is a battle-tested microservices framework that has been extensively validated in production environments. It not only delivers high performance but also offers ease of use and testability.

For more information, please refer to the [quick start guide][quick start] and [detailed documentation][docs].

## Overall Architecture

![Architecture](.resources/overall.png)

tRPC-Go has the following features:

- Multiple services can be started within a single process, listening on multiple addresses.
- All components are pluggable, with default implementations for various basic functionalities that can be replaced. Other components can be implemented by third parties and registered within the framework.
- All interfaces can be mock tested using gomock&mockgen to generate mock code, facilitating testing.
- The framework supports any third-party protocol by implementing the `codec` interfaces for the respective protocol. It defaults to supporting trpc and http protocols and can be switched at any time.
- It provides the [trpc command-line tool][trpc-go-cmdline] for generating code templates.

## Related Documentation

- [quick start guide][quick start] and [detailed documentation][docs]
- readme documents in each directory
- [trpc command-line tool][trpc-go-cmdline]
- [helloworld development guide][helloworld]
- [example documentation for various features][features]

## Ecosystem

- [codec plugins][go-codec]
- [filter plugins][go-filter]
- [database plugins][go-database]
- [more...][ecosystem]

## How to Contribute

If you're interested in contributing, please take a look at the [contribution guidelines][contributing] and check the [unassigned issues][issues] in the repository. Claim a task and let's contribute together to tRPC-Go.

[Go]: https://golang.org
[go-releases]: https://golang.org/doc/devel/release.html
[tRPC]: https://github.com/trpc-group/trpc
[trpc-go-cmdline]: https://github.com/trpc-group/trpc-go-cmdline
[docs]: /docs/README.md
[quick start]: /docs/quick_start.md
[contributing]: CONTRIBUTING.md
[issues]: https://github.com/trpc-group/trpc-go/issues
[go-codec]: https://github.com/trpc-ecosystem/go-codec
[go-filter]: https://github.com/trpc-ecosystem/go-filter
[go-database]: https://github.com/trpc-ecosystem/go-database
[ecosystem]: https://github.com/orgs/trpc-ecosystem/repositories
[helloworld]: /examples/helloworld/
[features]: /examples/features/
