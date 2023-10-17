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

package test

import (
	"fmt"
)

// trpcEnv defines the environment of the client/server in end-to-end testing,
// such as transport implementation, synchronous/asynchronous mode, and more.
type trpcEnv struct {
	server *trpcServerEnv
	client *trpcClientEnv
}

var allTRPCEnvs = generateTRPCEnv(generateTRPCServerEnvs(), generateTRPCClientEnvs())

func generateTRPCEnv(se []*trpcServerEnv, ce []*trpcClientEnv) []*trpcEnv {
	var envs []*trpcEnv
	for _, s := range se {
		for _, c := range ce {
			if s.transport == "tnet" && c.multiplexed {
				// tnet's client doesn't support multiplexing
				continue
			}
			envs = append(envs, &trpcEnv{
				server: s,
				client: c,
			})
		}
	}
	return envs
}

// String return the description of trpcEnv.
func (e *trpcEnv) String() string {
	return fmt.Sprintf("%v%v", e.server, e.client)
}

type trpcServerEnv struct {
	network   string
	transport string
	async     bool
}

// String return the description of trpcServerEnv.
func (e *trpcServerEnv) String() string {
	return fmt.Sprintf("network(%s)transport(%s)async(%t)", e.network, e.transport, e.async)
}

func generateTRPCServerEnvs() []*trpcServerEnv {
	var e []*trpcServerEnv
	for _, network := range []string{"tcp", "unix"} {
		for _, transport := range []string{"default", "tnet"} {
			for _, async := range []bool{true, false} {
				e = append(e, &trpcServerEnv{
					network:   network,
					transport: transport,
					async:     async,
				})
			}
		}
	}
	return e
}

type trpcClientEnv struct {
	multiplexed           bool
	disableConnectionPool bool
}

// String return the description of trpcClientEnv.
func (e *trpcClientEnv) String() string {
	return fmt.Sprintf("multiplexed(%t)disableConnectionPool(%t)", e.multiplexed, e.disableConnectionPool)
}

func generateTRPCClientEnvs() []*trpcClientEnv {
	var e []*trpcClientEnv
	for _, multiplexed := range []bool{true, false} {
		for _, connectionPool := range []bool{true, false} {
			e = append(e, &trpcClientEnv{
				multiplexed:           multiplexed,
				disableConnectionPool: connectionPool,
			})
		}
	}
	return e
}

var allHTTPServerEnvs = generateHTTPServerEnv()

type httpServerEnv struct {
	async bool
}

// String return the description of httpServerEnv.
func (e *httpServerEnv) String() string {
	return fmt.Sprintf("async(%t)", e.async)
}

func generateHTTPServerEnv() []*httpServerEnv {
	var e []*httpServerEnv
	for _, async := range []bool{true, false} {
		e = append(e, &httpServerEnv{async: async})
	}
	return e
}

var allHTTPRPCEnvs = generateHTTPRPCEnvs(generateHTTPServerEnv(), generateTRPCClientEnvs())

type httpRPCEnv struct {
	server *httpServerEnv
	client *trpcClientEnv
}

// String return the description of httpRPCEnv.
func (e *httpRPCEnv) String() string {
	return fmt.Sprintf("%v%v", e.server, e.client)
}

func generateHTTPRPCEnvs(se []*httpServerEnv, ce []*trpcClientEnv) []*httpRPCEnv {
	var envs []*httpRPCEnv
	for _, s := range se {
		for _, c := range ce {
			envs = append(envs, &httpRPCEnv{
				server: s,
				client: c,
			})
		}
	}
	return envs
}

var allRESTfulServerEnv = generateRESTfulServerEnv()

type restfulServerEnv struct {
	basedOnFastHTTP bool
	async           bool
}

// String return the description of restfulServerEnv.
func (e *restfulServerEnv) String() string {
	return fmt.Sprintf("basedOnFastHTTP(%t)async(%t)", e.basedOnFastHTTP, e.async)
}

func generateRESTfulServerEnv() []*restfulServerEnv {
	var e []*restfulServerEnv
	for _, basedOnFastHTTP := range []bool{true, false} {
		for _, async := range []bool{true, false} {
			e = append(e, &restfulServerEnv{basedOnFastHTTP: basedOnFastHTTP, async: async})
		}
	}
	return e
}
