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

package rpcz

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func testNewSpanIDConcurrentlyIsOk(t *testing.T) {
	const (
		idNum   = 1111
		iterNum = 11
		seed    = 20221111
	)
	gen := newRandomIDGenerator()
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

	expectedGen := newRandomIDGenerator()
	expectedIDs := make([]SpanID, idNum)
	actualIDs := make([]SpanID, idNum)
	for id := range idChan {
		expectedIDs = append(expectedIDs, expectedGen.newSpanID())
		actualIDs = append(actualIDs, id)
	}
	require.ElementsMatch(t, expectedIDs, actualIDs)
}
