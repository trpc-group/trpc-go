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

package admin

import (
	"time"
)

const (
	defaultListenAddr   = "127.0.0.1:9028" // default listen port.
	defaultUseTLS       = false            // default doesn't use https.
	defaultReadTimeout  = time.Second * 3
	defaultWriteTimeout = time.Second * 60
	defaultSkipServe    = false
)

func loadDefaultConfig() *adminConfig {
	return &adminConfig{
		skipServe:    defaultSkipServe,
		addr:         defaultListenAddr,
		enableTLS:    defaultUseTLS,
		readTimeout:  defaultReadTimeout,
		writeTimeout: defaultWriteTimeout,
	}
}

// adminConfig manages trpc service configuration.
type adminConfig struct {
	addr         string
	enableTLS    bool
	readTimeout  time.Duration
	writeTimeout time.Duration
	version      string
	configPath   string
	skipServe    bool
}

func (a *adminConfig) getAddr() string {
	return a.addr
}
