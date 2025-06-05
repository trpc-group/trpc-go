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

//go:build windows
// +build windows

package client

import (
	"errors"

	"trpc.group/trpc-go/trpc-go/transport"
)

func (cfg *BackendConfig) tnetClientPoolOption() (transport.RoundTripOption, error) {
	return nil, errors.New("tnet does not support windows")
}
