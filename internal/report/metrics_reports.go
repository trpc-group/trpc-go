//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package report reports the statistics of the framework.
package report

import (
	"trpc.group/trpc-go/trpc-go/metrics"
)

// Unified all metrics report inside the framework. Every property starts with "trpc.".
var (
	// -----------------------------server----------------------------- //
	// service starts successfully.
	ServiceStart = metrics.Counter("trpc.ServiceStart")
	// service is not configured with codec, protocol field must be filled in framework configuration.
	ServerCodecEmpty = metrics.Counter("trpc.ServerCodecEmpty")
	// service handle function fails, only happens when encode fails and is not able to reply.
	ServiceHandleFail = metrics.Counter("trpc.ServiceHandleFail")
	// fails to decode request, usually happens when the package is illegal.
	ServiceCodecDecodeFail = metrics.Counter("trpc.ServiceCodecDecodeFail")
	// fails to encode reply, usually happens when there is a bug in the codec plugin.
	ServiceCodecEncodeFail = metrics.Counter("trpc.ServiceCodecEncodeFail")
	// invalid handle rpc name, usually happens when the caller fills an incorrect parameter.
	ServiceHandleRPCNameInvalid = metrics.Counter("trpc.ServiceHandleRpcNameInvalid")
	// fails to unmarshal the body, usually happens when the pb file is not agreed between the caller and callee.
	ServiceCodecUnmarshalFail = metrics.Counter("trpc.ServiceCodecUnmarshalFail")
	// fails to marshal the body.
	ServiceCodecMarshalFail = metrics.Counter("trpc.ServiceCodecMarshalFail")
	// fails to decompress the body, usually happens when the pb file is not agreed between the caller and callee,
	// or the compress method is not aligned.
	ServiceCodecDecompressFail = metrics.Counter("trpc.ServiceCodecDecompressFail")
	// fails to compress the body.
	ServiceCodecCompressFail = metrics.Counter("trpc.ServiceCodecCompressFail")

	// -----------------------------transport----------------------------- //
	// fails to do tcp server transport handle, usually happens when encode fails.
	TCPServerTransportHandleFail = metrics.Counter("trpc.TcpServerTransportHandleFail")
	// fails to do tdp server transport handle, similar to TCPServerTransportHandleFail.
	UDPServerTransportHandleFail = metrics.Counter("trpc.UdpServerTransportHandleFail")
	// tcp server receives EOF, happens normally by the active close of the clients,
	// which occurs typically in the case of long connections.
	TCPServerTransportReadEOF = metrics.Counter("trpc.TcpServerTransportReadEOF")
	// tcp fails to write reply in transport, usually happens when the client has already closed the connection.
	TCPServerTransportWriteFail = metrics.Counter("trpc.TcpServerTransportWriteFail")
	// the request size of tcp server receives.
	TCPServerTransportReceiveSize = metrics.Gauge("trpc.TcpServerTransportReceiveSize")
	// the reply size of tcp server sends.
	TCPServerTransportSendSize = metrics.Gauge("trpc.TcpServerTransportSendSize")
	// udp fails to write reply, usually happens when the caller has already closed the connection
	// because of timeout and is not listening on the port.
	UDPServerTransportWriteFail = metrics.Counter("trpc.UdpServerTransportWriteFail")
	// tcp long connection reaches its idle timeout and releases the connection actively.
	TCPServerTransportIdleTimeout = metrics.Counter("trpc.TcpServerTransportIdleTimeout")
	// tcp server fails to read the frame, usually happens when the package is illegal.
	TCPServerTransportReadFail = metrics.Counter("trpc.TcpServerTransportReadFail")
	// udp server fails to read the frame, usually happens when the package is illegal.
	UDPServerTransportReadFail = metrics.Counter("trpc.UdpServerTransportReadFail")
	// the auxiliary data after udp server has already read for a complete frame.
	UDPServerTransportUnRead = metrics.Counter("trpc.UdpServerTransportUnRead")
	// udp client fails to read the frame, usually happens when the package is illegal.
	UDPClientTransportReadFail = metrics.Counter("trpc.UdpClientTransportReadFail")
	// the auxiliary data after udp client has already read for a complete frame.
	UDPClientTransportUnRead = metrics.Counter("trpc.UdpClientTransportUnRead")
	// request package size received by udp server.
	UDPServerTransportReceiveSize = metrics.Gauge("trpc.UdpServerTransportReceiveSize")
	// response package size sent by udp server.
	UDPServerTransportSendSize = metrics.Gauge("trpc.UdpServerTransportSendSize")
	// receive queue of tcp goroutine pool is full, the requests are overwhelming.
	TCPServerTransportJobQueueFullFail = metrics.Counter("trpc.TcpServerTransportJobQueueFullFail")
	// receive queue of udp goroutine pool is full, the requests are overwhelming.
	UDPServerTransportJobQueueFullFail = metrics.Counter("trpc.UdpServerTransportJobQueueFullFail")
	// TCPServerAsyncGoroutineScheduleDelay is the schedule delay of goroutine pool when async is on.
	// DO NOT change the name, as the overload control algorithm depends on it.
	TCPServerAsyncGoroutineScheduleDelay = metrics.Gauge("trpc.TcpServerAsyncGoroutineScheduleDelay_us")

	// -----------------------------log----------------------------- //
	// log is dropped because the queue is full.
	LogQueueDropNum = metrics.Counter("trpc.LogQueueDropNum")
	// the write size of log.
	LogWriteSize = metrics.Counter("trpc.LogWriteSize")
	// -----------------------------client----------------------------- //
	// the client fails to select the server node, usually happens when the name service is not
	// configured properly, or all the nodes have been fused.
	SelectNodeFail = metrics.Counter("trpc.SelectNodeFail")
	// client has not configured the protocol.
	ClientCodecEmpty = metrics.Counter("trpc.ClientCodecEmpty")
	// fails to load client config, usually happens when the client is not configured properly.
	LoadClientConfigFail = metrics.Counter("trpc.LoadClientConfigFail")
	// fails to load the client filter config, usually happens when client filer array is configured with a
	// filter that does not exist.
	LoadClientFilterConfigFail = metrics.Counter("trpc.LoadClientFilterConfigFail")
	// the request package size of tcp client.
	TCPClientTransportSendSize = metrics.Gauge("trpc.TcpClientTransportSendSize")
	// the response package size of tcp client.
	TCPClientTransportReceiveSize = metrics.Gauge("trpc.TcpClientTransportReceiveSize")
	// the request package size of udp client.
	UDPClientTransportSendSize = metrics.Gauge("trpc.UdpClientTransportSendSize")
	// the response package size of udp client.
	UDPClientTransportReceiveSize = metrics.Gauge("trpc.UdpClientTransportReceiveSize")

	// -----------------------------connection pool----------------------------- //
	// new connections of the connection pool.
	ConnectionPoolGetNewConnection = metrics.Counter("trpc.ConnectionPoolGetNewConnection")
	// fails to get a connection from the connection pool.
	ConnectionPoolGetConnectionErr = metrics.Counter("trpc.ConnectionPoolGetConnectionErr")
	// the remote peer of the connection pool is closed.
	ConnectionPoolRemoteErr = metrics.Counter("trpc.ConnectionPoolRemoteErr")
	// the remote peer of the connection pool returns EOF.
	ConnectionPoolRemoteEOF = metrics.Counter("trpc.ConnectionPoolRemoteEOF")
	// the connection reaches its idle timeout.
	ConnectionPoolIdleTimeout = metrics.Counter("trpc.ConnectionPoolIdleTimeout")
	// the connection exceeds its lifetime.
	ConnectionPoolLifetimeExceed = metrics.Counter("trpc.ConnectionPoolLifetimeExceed")
	// the connection number reaches its limit.
	ConnectionPoolOverLimit = metrics.Counter("trpc.ConnectionPoolOverLimit")

	// -----------------------------multiplexed----------------------------- //
	// fails to reconnect when multiplexed.
	MultiplexedTCPReconnectErr        = metrics.Counter("trpc.MultiplexedReconnectErr")
	MultiplexedTCPReconnectOnReadErr  = metrics.Counter("trpc.MultiplexedReconnectOnReadErr")
	MultiplexedTCPReconnectOnWriteErr = metrics.Counter("trpc.MultiplexedReconnectOnWriteErr")

	// -----------------------------other----------------------------- //
	// panic number of trpc.GoAndWait.
	PanicNum = metrics.Counter("trpc.PanicNum")

	// -----------------------------Admin----------------------------- //
	// AdminPanicNum is the panic number of admin.
	AdminPanicNum = metrics.Counter("trpc.AdminPanicNum")
)
