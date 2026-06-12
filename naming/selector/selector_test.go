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
	"fmt"
	"sync"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/naming/registry"

	"github.com/stretchr/testify/assert"
)

var testNode *registry.Node = &registry.Node{
	ServiceName: "testservice",
	Address:     "testservice.ip.1:16721",
	Network:     "tcp",
}

type testSelector struct {
}

// Select acquire a node.
func (ts *testSelector) Select(serviceName string, opt ...Option) (*registry.Node, error) {
	return testNode, nil
}

// Report reports data.
func (ts *testSelector) Report(node *registry.Node, cost time.Duration, success error) error {
	return nil
}

func TestSelectorRegister(t *testing.T) {
	Register("test-selector", &testSelector{})
	assert.NotNil(t, Get("test-selector"))
	unregisterForTesting("test-selector")
}

func TestSelectorGet(t *testing.T) {
	Register("test-selector", &testSelector{})
	s := Get("test-selector")
	assert.NotNil(t, s)
	unregisterForTesting("test-selector")
	assert.Nil(t, Get("not_exist"))
}

func TestSelectorRegisterAndGetConcurrent(t *testing.T) {
	const goroutines = 8
	const iterations = 100

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		name := fmt.Sprintf("test-selector-concurrent-%d", i)
		wg.Add(2)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				Register(name, &testSelector{})
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = Get(name)
			}
		}()
	}
	wg.Wait()
	for i := 0; i < goroutines; i++ {
		unregisterForTesting(fmt.Sprintf("test-selector-concurrent-%d", i))
	}
}
