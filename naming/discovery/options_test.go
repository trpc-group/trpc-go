//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package discovery

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptions(t *testing.T) {
	opts := &Options{}

	ctx := context.Background()
	WithContext(ctx)(opts)
	assert.Equal(t, opts.Ctx, ctx)

	WithNamespace("ns")(opts)
	assert.Equal(t, opts.Namespace, "ns")
}
