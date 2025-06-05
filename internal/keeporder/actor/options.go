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

package actor

import "time"

// Options specifies the actor's options.
type Options struct {
	IdleGroupTimeout time.Duration
	MaxElementCount  int
}

func (o *Options) fixDefault() {
	if o.IdleGroupTimeout == 0 {
		o.IdleGroupTimeout = defaultIdleGroupTimeout
	}
	if o.MaxElementCount == 0 {
		o.MaxElementCount = defaultMaxElementCount
	}
}
