// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package multiplexed

import "io"

// FrameParser is the interface to parse a single frame.
type FrameParser interface {
	// Parse parses vid and frame from io.ReadCloser. rc.Close must be called before Parse return.
	Parse(rc io.Reader) (vid uint32, buf []byte, err error)
}
