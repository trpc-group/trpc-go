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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"trpc.group/trpc-go/trpc-go/naming/bannednodes"
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

// IPSelectorTestSuite 测试 suite
type IPSelectorTestSuite struct {
	suite.Suite
	selector Selector
}

// SetupSuite 初始化测试
func (suite *IPSelectorTestSuite) SetupSuite() {
}

// 初始化测试
func (suite *IPSelectorTestSuite) SetupTest() {
	suite.selector = Get("ip")
}

func (suite *IPSelectorTestSuite) TestIPSelectSingleIP() {
	serviceName := "trpc.service.ip.1:1234"
	node, err := suite.selector.Select(serviceName)
	suite.T().Logf("Select return node:{%+v}, err:{%+v}", node, err)

	suite.NoError(err)
	suite.Equal(node.ServiceName, "trpc.service.ip.1:1234")
	suite.Equal(node.Address, "trpc.service.ip.1:1234")
}

func (suite *IPSelectorTestSuite) TestIPSelectMultiIP() {
	serviceName := "trpc.service.ip.1:1234,trpc.service.ip.2:1234"

	node, err := suite.selector.Select(serviceName)
	suite.T().Logf("Select return node:{%+v}, err:{%+v}", node, err)
	suite.NoError(err)
	suite.Equal(node.ServiceName, serviceName)

	node, err = suite.selector.Select(serviceName)
	suite.T().Logf("Select return node:{%+v}, err:{%+v}", node, err)
	suite.NoError(err)
	suite.Equal(node.ServiceName, serviceName)
}

func (suite *IPSelectorTestSuite) TestIPSelectEmpty() {
	serviceName := ""

	node, err := suite.selector.Select(serviceName)
	suite.T().Logf("Select return node:{%+v}, err:{%+v}", node, err)
	suite.Error(err)
	suite.Nil(node, "serviceName is empty")
}

func TestIPSelector(t *testing.T) {
	suite.Run(t, new(IPSelectorTestSuite))
}

func TestIPSelectorSelect(t *testing.T) {
	s := Get("ip")
	n, err := s.Select("trpc.service.ip.1:8888")
	assert.Nil(t, err)
	assert.Equal(t, n.Address, "trpc.service.ip.1:8888")
}

func TestIPSelectorReport(t *testing.T) {
	s := Get("ip")
	n, err := s.Select("trpc.service.ip.1:8888")
	assert.Nil(t, err)
	assert.Equal(t, n.Address, "trpc.service.ip.1:8888")
	assert.Nil(t, s.Report(n, 0, nil))
}

func TestIPSelectorSelectMandatoryBanned(t *testing.T) {
	candidates := "127.0.0.1:8000,127.0.0.1:8001"

	ctx := bannednodes.NewCtx(context.Background(), true)

	s := NewIPSelector()

	_, err := s.Select(candidates, WithContext(ctx))
	require.Nil(t, err)

	_, err = s.Select(candidates, WithContext(ctx))
	require.Nil(t, err)

	_, err = s.Select(candidates, WithContext(ctx))
	require.NotNil(t, err)

	nodes, mandatory, ok := bannednodes.FromCtx(ctx)
	require.True(t, ok)
	require.True(t, mandatory)
	var n int
	nodes.Range(func(*registry.Node) bool {
		n++
		return true
	})
	require.Equal(t, 2, n)
}

func TestIPSelectorSelectOptionalBanned(t *testing.T) {
	candidates := "127.0.0.1:8000,127.0.0.1:8001"

	ctx := bannednodes.NewCtx(context.Background(), false)

	s := NewIPSelector()

	n1, err := s.Select(candidates, WithContext(ctx))
	require.Nil(t, err)

	n2, err := s.Select(candidates, WithContext(ctx))
	require.Nil(t, err)

	require.NotEqual(t, n1.Address, n2.Address)

	_, err = s.Select(candidates, WithContext(ctx))
	require.Nil(t, err)

	nodes, mandatory, ok := bannednodes.FromCtx(ctx)
	require.True(t, ok)
	require.False(t, mandatory)
	var n int
	nodes.Range(func(*registry.Node) bool {
		n++
		return true
	})
	require.Equal(t, 3, n)
}

// BenchmarkIPSelectorSelectOneService benchmark Select 性能
func BenchmarkIPSelectorSelectOneService(b *testing.B) {
	s := Get("ip")
	for i := 0; i < b.N; i++ {
		s.Select("trpc.service.ip.1:8888")
	}
}

// BenchmarkIPSelectorSelectMultiService 测试 Select 性能
func BenchmarkIPSelectorSelectMultiService(b *testing.B) {
	s := Get("ip")
	for i := 0; i < b.N; i++ {
		s.Select("trpc.service.ip.1:8888,trpc.service.ip.1:8886,trpc.service.ip.1:8887")
	}
}
