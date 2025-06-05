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

package graceful

// appendMap appends k1-k2-v to a map[K1]map[K2]V.
func appendMap[K1, K2 comparable, V any](mp map[K1]map[K2]V, k1 K1, k2 K2, v V) map[K1]map[K2]V {
	if mp == nil {
		mp = make(map[K1]map[K2]V)
	}
	if kv, ok := mp[k1]; ok {
		kv[k2] = v
	} else {
		mp[k1] = map[K2]V{k2: v}
	}
	return mp
}

// deleteMap delete k1-k2 from map[K1]map[K2]V.
func deleteMap[K1, K2 comparable, V any](mp map[K1]map[K2]V, k1 K1, k2 K2) {
	delete(mp[k1], k2)
	if len(mp[k1]) == 0 {
		delete(mp, k1)
	}
}
