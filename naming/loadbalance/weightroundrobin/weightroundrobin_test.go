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

package weightroundrobin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

func TestWrrSmoothBalancing(t *testing.T) {
	wrr := NewWeightRoundRobin(0)
	// weight: a: 5, b: 1, c: 1
	// list shound be: a, a, b, a, c, a, a
	tests := []int{0, 0, 1, 0, 2, 0, 0}
	for i := 0; i < 7; i++ {
		n, err := wrr.Select("test1", list1)
		assert.Nil(t, err)
		assert.Equal(t, list1[tests[i]], n)
	}
}

func TestWrrListLengthChange(t *testing.T) {
	wrr := NewWeightRoundRobin(defaultUpdateRate)
	n1, err := wrr.Select("test1", list1)
	assert.Nil(t, err)
	assert.Equal(t, n1, list1[0])

	tests := []int{3, 0, 1, 2, 3, 0, 3, 1, 2, 0, 3}
	for i := 0; i < 11; i++ {
		n, err := wrr.Select("test1", list2)
		assert.Nil(t, err)
		assert.Equal(t, list2[tests[i]], n)
	}
}

func TestWrrInterval(t *testing.T) {
	wrr := NewWeightRoundRobin(time.Second * 1)

	n1, err := wrr.Select("test1", list1)
	assert.Nil(t, err)
	assert.Equal(t, n1, list1[0])
	tests := []int{0, 1, 0, 2, 0, 0}
	for i := 0; i < 6; i++ {
		n, err := wrr.Select("test1", list3)
		assert.Nil(t, err)
		assert.Equal(t, list1[tests[i]], n)
	}

	time.Sleep(time.Second)

	tests = []int{0, 1, 0, 2, 0, 1, 0}
	for i := 0; i < 6; i++ {
		n, err := wrr.Select("test1", list3)
		assert.Nil(t, err)
		assert.Equal(t, list3[tests[i]], n)
	}
}

func TestWrrDifferentService(t *testing.T) {
	wrr := NewWeightRoundRobin(defaultUpdateRate)
	// weight: a: 5, b: 1, c: 1
	// list shound be: a, a, b, a, c, a, a
	tests := []int{0, 0, 1, 0, 2, 0, 0}
	for i := 0; i < 7; i++ {
		n, err := wrr.Select("test1", list1)
		assert.Nil(t, err)
		assert.Equal(t, list1[tests[i]], n)
	}

	tests = []int{3, 0, 1, 2, 3, 0, 3, 1, 2, 0, 3}
	for i := 0; i < 11; i++ {
		n, err := wrr.Select("test2", list2)
		assert.Nil(t, err)
		assert.Equal(t, list2[tests[i]], n)
	}
}

var list1 = []*registry.Node{
	{
		Address: "list1.ip.1:8080",
		Weight:  5,
		Metadata: map[string]interface{}{
			"weight": 5,
		},
	},
	{
		Address: "list1.ip.3:8080",
		Weight:  1,
		Metadata: map[string]interface{}{
			"weight": 1,
		},
	},
	{
		Address: "list1.ip.4:8080",
		Weight:  1,
		Metadata: map[string]interface{}{
			"weight": 1,
		},
	},
}

var list2 = []*registry.Node{
	{
		Address: "list2.ip.5:8080",
		Weight:  3,
		Metadata: map[string]interface{}{
			"weight": 3,
		},
	},
	{
		Address: "list2.ip.6:8080",
		Weight:  2,
		Metadata: map[string]interface{}{
			"weight": 2,
		},
	},
	{
		Address: "list2.ip.7:8080",
		Weight:  2,
		Metadata: map[string]interface{}{
			"weight": 2,
		},
	},
	{
		Address: "list2.ip.8:8080",
		Weight:  4,
		Metadata: map[string]interface{}{
			"weight": 4,
		},
	},
}

var list3 = []*registry.Node{
	{
		Address: "list3.ip.1:8080",
		Weight:  4,
		Metadata: map[string]interface{}{
			"weight": 4,
		},
	},
	{
		Address: "list3.ip.3:8080",
		Weight:  2,
		Metadata: map[string]interface{}{
			"weight": 2,
		},
	},
	{
		Address: "list3.ip.4:8080",
		Weight:  1, Metadata: map[string]interface{}{
			"weight": 1,
		},
	},
}
