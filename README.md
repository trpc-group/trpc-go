# tRPC-Go framework

The tRPC-Go framework is the Golang version of the company's unified microservices framework, mainly designed as an RPC framework with high performance, pluggability, and easy testing in mind.

## Overall Architecture

![Architecture](.resources/overall.png)

- A server process supports starting multiple service instances and listening to multiple addresses.
- All components are pluggable, with built-in default implementations for basic functions like transport, which can be replaced. Other components need to be implemented by third-party businesses and registered with the framework.
- All interfaces can be mocked, using gomock & mockgen to generate mock code for easy testing.
- Supports any third-party business protocol, just need to implement the business protocol encoding and decoding interface. Default support for trpc and http protocols, switchable at any time, seamless development of CGI and backend servers.
- Provides a trpc command-line tool for generating code templates.

## Plugin Management

- The framework's plugin management design only provides standard interfaces and interface registration capabilities.
- External components are wrapped by third-party businesses as bridges, wrapping system components according to the framework interface, and registering them with the framework.
- When using, just import the wrapper bridge path.
- For specific plugin principles, refer to [plugin](plugin).

## Generation Tool

Refer to [trpc-group/trpc-go-cmdline](https://github.com/trpc-group/trpc-go-cmdline) for installation and usage.

## Service Protocol

- The trpc framework supports any third-party protocol and defaults to trpc and http protocols
- Just specify the protocol field in the configuration file to be equal to http to start a CGI service
- Using the same service description protocol, completely identical code, can switch between trpc and http at any time, achieving truly seamless development of CGI and backend services
- Request data is carried using the http post method and parsed into the method's request structure, specifying the use of pb or json through the http header content-type (application/json or application/pb)
- Third-party custom business protocols can refer to [codec](codec)

## Related Documentation

- [Framework Design Document](https://trpc.group/trpc-go/trpc-wiki)
- [Detailed description of trpc tool](https://trpc.group/trpc-go/trpc-go-cmdline)
- [helloworld Development Guide](examples/helloworld)
- [Third-party protocol implementation demo](https://trpc.group/trpc-go/trpc-codec)

## How to Contribute

Interested students can first take a look at the [Contribution Guide](CONTRIBUTING.md), then look at the unclaimed issues in the Issue, claim tasks for themselves, and contribute to tRPC-Go together.
