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

package selector

import (
	"testing"

	"trpc.group/trpc-go/trpc-go/naming/registry"

	"github.com/stretchr/testify/assert"
)

func TestTrpcSelectorSelect(t *testing.T) {
	selector := &TrpcSelector{}
	n, err := selector.Select("127.0.0.1:12345")
	assert.Nil(t, err)
	assert.Equal(t, n.Address, "127.0.0.1:12345")
}

func TestTrpcSelectorReport(t *testing.T) {
	selector := &TrpcSelector{}
	n, err := selector.Select("127.0.0.1:12345")

	assert.Nil(t, err)
	assert.Equal(t, n.Address, "127.0.0.1:12345")

	assert.Nil(t, selector.Report(n, 0, nil))
}

func TestTrpcSelectorReportErr(t *testing.T) {
	selector := &TrpcSelector{}
	assert.Equal(t, selector.Report(nil, 0, nil), ErrReportNodeEmpty)
	assert.Equal(t, selector.Report(&registry.Node{}, 0, nil), ErrReportMetaDataEmpty)
	assert.Equal(t, selector.Report(&registry.Node{
		Metadata: make(map[string]interface{}),
	}, 0, nil), ErrReportNoCircuitBreaker)
	assert.Equal(t, selector.Report(&registry.Node{
		Metadata: map[string]interface{}{
			"circuitbreaker": "circuitbreaker",
		},
	}, 0, nil), ErrReportInvalidCircuitBreaker)
}

func TestTrpcSelectorNil(t *testing.T) {
	selector := &TrpcSelector{}

	_, err := selector.Select("")
	assert.NotNil(t, err)

	_, err = selector.Select("service", WithDiscovery(nil))
	assert.NotNil(t, err)

	_, err = selector.Select("service", WithLoadBalancer(nil))
	assert.NotNil(t, err)

	_, err = selector.Select("service", WithCircuitBreaker(nil))
	assert.NotNil(t, err)
}
