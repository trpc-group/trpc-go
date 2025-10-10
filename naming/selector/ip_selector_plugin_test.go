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

package selector_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	_ "trpc.group/trpc-go/trpc-go/naming/selector"
	"trpc.group/trpc-go/trpc-go/plugin"
)

func TestIPSelectorPlugin(t *testing.T) {
	p := plugin.Get("selector", "direct")
	require.NotNil(t, p.Setup("direct", funcDecoder(func(interface{}) error {
		return errors.New("")
	})))
	require.Nil(t, p.Setup("direct", funcDecoder(func(cfg interface{}) error {
		return yaml.Unmarshal([]byte(`
circuitBreaker:
  default:
    enable: true
    statWindow: 60s
    bucketsNum: 12
    sleepWindow: 30s
    requestVolumeThreshold: 10
    errorRateThreshold: 0.5
    continuousErrorThreshold: 10
    requestCountAfterHalfOpen: 10
    successCountAfterHalfOpen: 8
`), cfg)
	})))
	require.Nil(t, p.Setup("direct", funcDecoder(func(cfg interface{}) error {
		return yaml.Unmarshal([]byte(`
circuitBreaker:
  default:
    enable: false
`), cfg)
	})))
}

type funcDecoder func(interface{}) error

func (d funcDecoder) Decode(cfg interface{}) error {
	return d(cfg)
}
