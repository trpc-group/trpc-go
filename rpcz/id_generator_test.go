// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package rpcz

import (
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newRandomIDGenerator(t *testing.T) {
	type args struct {
		seed int64
	}
	tests := []struct {
		name string
		args args
		want *randomIDGenerator
	}{
		{
			name: "seed equals zero",
			args: args{seed: 0},
			want: &randomIDGenerator{randSource: rand.New(rand.NewSource(0))},
		},
		{
			name: "seed greater zero",
			args: args{seed: 20221111},
			want: &randomIDGenerator{randSource: rand.New(rand.NewSource(20221111))},
		},
		{
			name: "seed less zero",
			args: args{seed: -20221111},
			want: &randomIDGenerator{randSource: rand.New(rand.NewSource(-20221111))},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newRandomIDGenerator(tt.args.seed); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newRandomIDGenerator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_randomIDGenerator_newSpanID(t *testing.T) {
	t.Run("testNewSpanIDSequentiallyIsOk", testNewSpanIDSequentiallyIsOk)
	t.Run("testNewSpanIDConcurrentlyIsOk", testNewSpanIDConcurrentlyIsOk)
}

func testNewSpanIDSequentiallyIsOk(t *testing.T) {
	const seed = 20221111
	gen := newRandomIDGenerator(seed)
	rand.Seed(seed)
	for i := 0; i < 1111; i++ {
		require.Equal(t, SpanID(rand.Int63()), gen.newSpanID())
	}
}
func testNewSpanIDConcurrentlyIsOk(t *testing.T) {
	const (
		idNum   = 1111
		iterNum = 11
		seed    = 20221111
	)
	gen := newRandomIDGenerator(seed)
	idChan := make(chan SpanID, idNum)

	var wg sync.WaitGroup
	wg.Add(idNum)
	for i := 0; i < iterNum; i++ {
		for j := 0; j < idNum/iterNum; j++ {
			go func() {
				idChan <- gen.newSpanID()
				wg.Done()
			}()
		}
	}
	wg.Wait()
	close(idChan)

	expectedGen := newRandomIDGenerator(seed)
	expectedIDs := make([]SpanID, idNum)
	actualIDs := make([]SpanID, idNum)
	for id := range idChan {
		expectedIDs = append(expectedIDs, expectedGen.newSpanID())
		actualIDs = append(actualIDs, id)
	}
	require.ElementsMatch(t, expectedIDs, actualIDs)
}
