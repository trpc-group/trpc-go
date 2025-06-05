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
	"fmt"
	"time"

	"trpc.group/trpc-go/trpc-go/naming/circuitbreaker"
	"trpc.group/trpc-go/trpc-go/plugin"
)

func init() {
	plugin.Register("direct", newIPSelectorPlugin())
}

func newIPSelectorPlugin() *ipSelectorPlugin {
	p := ipSelectorPlugin{}
	p.CircuitBreaker.Default.Enable = true
	return &p
}

type ipSelectorPlugin struct {
	CircuitBreaker struct {
		Default struct {
			Enable                    bool           `yaml:"enable"`
			StatWindow                *time.Duration `yaml:"statWindow"`
			BucketsNum                *int           `yaml:"bucketsNum"`
			SleepWindow               *time.Duration `yaml:"sleepWindow"`
			RequestVolumeThreshold    *int           `yaml:"requestVolumeThreshold"`
			ErrorRateThreshold        *float64       `yaml:"errorRateThreshold"`
			ContinuousErrorThreshold  *int           `yaml:"continuousErrorThreshold"`
			RequestCountAfterHalfOpen *int           `yaml:"requestCountAfterHalfOpen"`
			SuccessCountAfterHalfOpen *int           `yaml:"successCountAfterHalfOpen"`
		} `yaml:"default"`
	} `yaml:"circuitBreaker"`
}

// Type returns the type of ipSelectorPlugin "selector".
func (p *ipSelectorPlugin) Type() string {
	return "selector"
}

// Setup setups the ipSelectorPlugin.
func (p *ipSelectorPlugin) Setup(name string, dec plugin.Decoder) error {
	if err := dec.Decode(p); err != nil {
		return fmt.Errorf("failed to setup plugin selector-%s, err: %w", name, err)
	}

	def := &p.CircuitBreaker.Default
	if !def.Enable {
		return nil
	}

	var opts []circuitbreaker.Opt
	if def.StatWindow != nil {
		opts = append(opts, circuitbreaker.WithSlidingWindowInterval(*def.StatWindow))
	}
	if def.BucketsNum != nil {
		opts = append(opts, circuitbreaker.WithSlidingWindowSize(*def.BucketsNum))
	}
	if def.SleepWindow != nil {
		opts = append(opts, circuitbreaker.WithOpenDuration(*def.SleepWindow))
	}
	if def.RequestVolumeThreshold != nil {
		opts = append(opts, circuitbreaker.WithMinRequestsToOpen(*def.RequestVolumeThreshold))
	}
	if def.ErrorRateThreshold != nil {
		opts = append(opts, circuitbreaker.WithErrRateToOpen(*def.ErrorRateThreshold))
	}
	if def.ContinuousErrorThreshold != nil {
		opts = append(opts, circuitbreaker.WithContinuousFailuresToOpen(*def.ContinuousErrorThreshold))
	}
	if def.RequestCountAfterHalfOpen != nil {
		opts = append(opts, circuitbreaker.WithTotalRequestsToClose(*def.RequestCountAfterHalfOpen))
	}
	if def.SuccessCountAfterHalfOpen != nil {
		opts = append(opts, circuitbreaker.WithSuccessRequestsToClose(*def.SuccessCountAfterHalfOpen))
	}
	Register("ip", NewIPSelectorWithCircuitBreaker(circuitbreaker.NewLRUCircuitBreakers(opts...)))
	return nil
}
