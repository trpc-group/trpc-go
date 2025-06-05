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

package overloadctrl

import (
	"context"
	"fmt"
)

// Impl 提供了一种基于 yaml 配置的默认实现。
type Impl struct {
	OverloadController        // exported as unit test need it
	Builder            string // exported as server backward compatibility need it
}

// UnmarshalYAML 实现 yaml.Unmarshaler.
func (impl *Impl) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&impl.Builder)
}

// MarshalYAML 实现 yaml.Marshaler.
func (impl Impl) MarshalYAML() (interface{}, error) {
	return impl.Builder, nil
}

// Acquire 实现过载保护接口。
func (impl *Impl) Acquire(ctx context.Context, addr string) (Token, error) {
	if impl.OverloadController == nil {
		return NoopToken{}, nil
	}
	return impl.OverloadController.Acquire(ctx, addr)
}

// Build 构造出实际的过载保护实例。
func (impl *Impl) Build(getBuilder func(string) Builder, smi *ServiceMethodInfo) error {
	if impl.Builder == "" {
		return nil
	}
	newOC := getBuilder(impl.Builder)
	if newOC == nil {
		return fmt.Errorf("overload control builder %s is not found", impl.Builder)
	}
	impl.OverloadController = newOC(smi)
	return nil
}
