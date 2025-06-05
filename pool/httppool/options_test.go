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

package httppool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithOptions(t *testing.T) {
	opts := &Options{}

	// WithMaxIdleConns
	WithMaxIdleConns(10)(opts)
	assert.Equal(t, 10, opts.MaxIdleConns)

	// WithMaxIdleConnsPerHost
	WithMaxIdleConnsPerHost(5)(opts)
	assert.Equal(t, 5, opts.MaxIdleConnsPerHost)

	// WithMaxConnsPerHost
	WithMaxConnsPerHost(7)(opts)
	assert.Equal(t, 7, opts.MaxConnsPerHost)

	// WithIdleConnTimeout
	WithIdleConnTimeout(time.Second)(opts)
	assert.Equal(t, time.Second, opts.IdleConnTimeout)
}
