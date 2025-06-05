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

package frame

import "encoding/binary"

const (
	trpcFrameHeadLen = 16
	// fix import cycle trpc.TrpcMagic_TRPC_MAGIC_VALUE, and trpc.TrpcDataFrameType_TRPC_STREAM_FRAME
	trpcMagicVALUE      = 2352
	trpcStreamFrameType = 1
)

// ContainTRPCStreamHeader checks if the provided byte slice contains a valid TRPC stream header.
func ContainTRPCStreamHeader(bts []byte) bool {
	if len(bts) < trpcFrameHeadLen {
		return false
	}
	magic := binary.BigEndian.Uint16(bts[:2])
	if magic != uint16(trpcMagicVALUE) {
		return false
	}
	return bts[2] == trpcStreamFrameType
}
