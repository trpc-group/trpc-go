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

package selector

import (
	"time"

	"trpc.group/trpc-go/trpc-go/naming/registry"
)

func init() {
	Register("passthrough", NewPassthroughSelector()) // passthrough://temp.sock
	Register("unix", NewPassthroughSelector())        // unix://temp.sock
}

// passthroughSelector is a selector simply passthrough serviceName.
type passthroughSelector struct{}

// NewPassthroughSelector creates a new passthroughSelector.
func NewPassthroughSelector() *passthroughSelector {
	return &passthroughSelector{}
}

// Select implements Selector.Select, passthrough service name.
func (s *passthroughSelector) Select(
	serviceName string, opt ...Option,
) (*registry.Node, error) {
	return &registry.Node{
		ServiceName: serviceName,
		Address:     serviceName,
	}, nil
}

// Report reports nothing.
func (s *passthroughSelector) Report(*registry.Node, time.Duration, error) error {
	return nil
}
