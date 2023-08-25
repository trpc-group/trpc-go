[TOC]



# Introduction

First of all, welcome everyone to read the tRPC-Go architecture design document. This is a great opportunity to share some of the thoughts behind the tRPC-Go design with everyone. A lot of people may wonder what innovations tRPC-Go has, what advantages it has over external open-source frameworks, and why they should spend a lot of effort to learn a new framework, etc.

This article mainly discusses the architecture design of the unique features of tRPC-Go. The tRPC framework as a whole follows a consistent design across all languages, and the same parts can be viewed in the architecture overview.

Perhaps tRPC-Go is not a star product in the industry, but it should be a good choice for solving problems. The tRPC family provides framework versions in multiple languages, and follows a consistent architecture design at the top level. The framework features and peripheral ecosystem construction also strive to advance synchronously, providing a decent guarantee for meeting the needs of different technology stacks in the company team, supporting peripheral components, and technical support.

"As an arrow piercing through the clouds, meeting thousands of troops and horses", I am fortunate to feel the power of open source collaboration in framework governance. tRPC-Go was born in everyone's discussion, and I hope it will continue to grow in larger discussions within the company.

# Background

In order to help everyone better understand the architecture design of tRPC-Go, this article will cover necessary content as much as possible. This document is based on the tRPC-Go framework v0.3.6. Due to the limited energy of the author, subsequent documents may also become outdated, so we hope everyone can participate together.

The following sections of this article will be organized as follows:

- First, introduce the overall architecture design of tRPC-Go, so that everyone can have a general understanding;
- Then, introduce the server workflow of tRPC-Go, so that everyone can grasp the working principle of the server from a global perspective;
- Then, introduce the client workflow of tRPC-Go, so that everyone can grasp the working principle of the client from a global perspective;
- Then, introduce the performance optimization of tRPC-Go, and inform everyone of some optimization options that can be adjusted;
- Then, I would like to share with everyone the design and optimization points of certain parts, for continuous optimization and iteration in the future.

This is the first article on the architecture design of tRPC-Go, focusing on the framework. Subsequent documents on module design will introduce the collaboration between modules and between modules and the framework in more detail.

# Architecture_Design

## Overall View

The overall architecture design of tRPC-Go is as follows:

![overall](/.resources/developer_guide/architecture_design/overall.png)

The tRPC-Go framework mainly includes several core modules:

- client: provides a concurrent and safe general client implementation, mainly responsible for service discovery, load balancing, route selection, circuit breaking, encoding and decoding, and custom interceptor-related operations, all of which support plug-in extension;
- server: provides a service implementation that supports multiple service startups, registration, deregistration, hot restart, and graceful exit;
- codec: provides encoding and decoding related interfaces, allowing the framework to extend business protocols, serialization methods, data compression methods, etc.;
- config: provides configuration reading related interfaces, supports reading local configuration files, remote configuration center configurations, etc., allows plug-in extension of different formats of configuration files, different configuration centers, and supports reload and watch configuration updates;
- log: provides a common logging interface and zaplog implementation, allowing logging implementation to be extended through plugins, and allowing logs to be output to multiple destinations;
- naming: provides naming service node registration registry, service discovery selector, load balancing load balance, circuit breaking circuit breaker, etc., which is essentially a load balancing implementation based on naming services;
- pool: provides a connection pool implementation, manages idle connections based on stacks, supports periodic checking of connection status and cleaning of connections;
- tracing: provides distributed tracing, currently implemented based on filters and not in the main framework;
- filter: provides the definition of custom interceptors, allowing processing capabilities to be enriched through extension of filters, such as tracing, recovery, model calling, log replay, etc.;
- transport: provides transport layer related definitions and default implementations, supports TCP and UDP transport modes;
- metrics: provides monitoring reporting capabilities, supports common single-dimensional reporting, such as counters, gauges, etc., and also supports multi-dimensional reporting, allowing extension of the Sink interface to interface with different monitoring platforms;
- trpc: provides default trpc protocol, framework configuration, framework version management, and other related information.

## Overall Interaction Process

The overall interaction process of tRPC-Go is as follows:

![interaction_process](/.resources/developer_guide/architecture_design/interaction_process.png)

# Working Principle

## Server

### Startup

The Server startup process generally includes the following steps:

1. Initialize the service instance with `trpc.NewServer()`;
2. Read the framework configuration file (-conf specified) and deserialize it into `trpc.Config`. This configuration includes server, service, client, and various plugin configuration information;
3. Traverse the service list and various plugin configurations in the configuration file to complete the initialization logic;
    1. Start listening for the service, complete the service registration, and cancel all registrations and exit if any one fails;
    2. Complete the initialization of various plugins, and panic and exit the process if any one fails;
    3. Listen for the SIGUSR2 signal and execute the hot restart logic when received;
    4. Listen for signals such as SIGINT, and exit the process normally when received;
4. Register the service with `server.Register(pb.ServiceDesc, serviceImpl)`, which actually registers the mapping relationship between the RPC method name and the processing function;
5. The service has now started normally and is waiting for client connection requests.

### Request Processing

1. The server transport calls Accept to wait for the client to establish a connection;
2. The client initiates a connection request, and the server transport Accept returns a tcpconn connection;
3. The server transport decides whether to process requests on the same connection in series or in parallel based on the current working mode (AsyncMod);
    1. If it is processed in series, one goroutine is used to process each connection, and requests arriving on the connection are processed in order. This is suitable for scenarios where the client does not reuse connections;
    2. If it is processed in parallel, one goroutine is started to process each request on the connection. Although this method implements concurrent processing, it may cause goroutine explosion.
4. The logic of receiving packets begins. The server transport continuously reads requests based on the encoding and decoding protocol, compression method, and serialization method, and encapsulates them into a msg to be processed by the upper layer;
5. After obtaining the msg, find the corresponding registered processing function based on the rpc name inside the msg and call the corresponding processing function;
6. Before calling the corresponding processing function, it actually needs to go through a filterchain. The filterchain executes until the end, which is our registered RPC processing function;
7. Serialize, compress, encode and decode the processing result, and then send it back to the client;

Note that when reading requests from tcpconn, several situations may occur:

- Normal request reading, OK
- Reaching EOF, indicating that the connection on the other end is closed, close it;
- Timeout reading, and exceeding the set connection idle time, close it;
- Data is read, but the unpacking fails, close it;

### Exit

At the exit stage of server, it can be further refined based on different exit scenarios.

#### Normal Exit

When the service receives a signal such as SIGINT, it executes the normal exit logic:

1. Call the close method of each service to close the service logic;
2. Cancel the registration of each service in the name service;
3. Call the close method of each plugin;
4. Exit.

#### Abnormal Exit

1. If a goroutine is started in the business code and panics internally without recovering normally, the service will panic;
2. If the server-side filter "recovery" is introduced in the service and a panic occurs in the business processing goroutine started by the framework, the recovery filter is responsible for capturing it and preventing an abnormal exit.

#### Hot Restart

1. After receiving the SIGUSR2 signal, execute the hot restart logic;
2. The parent process first collects the currently opened listeners, including tcplistener and udp packetconn, and then obtains their fd;
3. The parent process forkexec creates a child process. When creating the child process, it passes the fd through ProcAttr to share `stdin\stdout\stderr` and `tcplistener fd` and `packetconn fd` with the child process, and notifies the child process of the hot restart through environment variables;
4. At this time, the child process starts, and the process startup process will also go through the server startup process. Actually, when starting to listen, it checks the environment variables to discover whether it is in hot restart mode. If it is, it rebuilds the listener through the passed fd. Otherwise, it listens through `net.Listen` or `reuseport.Listen`;
5. After forkexec returns, the parent process continues to execute the subsequent exit process (requests on the established connection are not exited until the processing is completed and the response is sent back);
6. The parent process executes custom transaction cleanup logic, similar to the hook function registered by AtExit (currently not implemented);
7. The parent process exits, and the child process takes over the processing.

## Client

1. When sending a request, first assemble various call parameters;
2. Execute the client filter pre-processing logic;
3. Serialize, compress, and encode the data to be sent;
4. Service discovery finds a group of `ip:port` lists corresponding to the called service name;
5. Find a suitable `ip:port` to initiate the request through load balancing algorithms;
6. Check whether to allow the current request to be initiated through circuit breaker (to avoid pressure on the backend caused by retries and trigger a cascade failure);
7. If everything is OK, prepare to establish a connection to `ip:port`. At this time, it will first check whether there is a corresponding idle connection in the connection pool. If not, it needs to be created through `net.Dial`;
8. After obtaining the connection, start sending data and wait for the response (if it is a connection reuse mode, multiple requests may be sent concurrently on the same connection, and the request response is associated through seqno, which is currently not implemented);
9. After receiving the data, decode, decompress, and deserialize it, and submit it to the upper layer for processing;
10. Execute the client filter post-processing logic.

It should be noted that the client also involves a filterchain logic, which can be extended to a series of functions, such as reporting tracing data and model call data during RPC.

The connection pool used internally by the client is actually referenced in the client transport. The client is a general client, and the differentiation of tcp, udp, and connection pools is managed by the client transport, and the connection pool also periodically checks the availability of connections. More details will be introduced in the subsequent module documentation.

# Summary

Here is a brief summary of the overall architecture design of tRPC-Go, as well as the rough workflow of the client and server, with mentions of the functions of related modules. More information on this part will be provided in the subsequent module design documentation in more detail.

# Owner

zhijiezhang