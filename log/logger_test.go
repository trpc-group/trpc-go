// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package log

import "testing"

func TestLevel_String(t *testing.T) {
	t.Run("debug level", func(t *testing.T) {
		l := LevelDebug
		if got, want := l.String(), "debug"; got != want {
			t.Errorf("l.String() = %s, want %s", got, want)
		}
	})
}
