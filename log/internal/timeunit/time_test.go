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

package timeunit

import (
	"testing"
	"time"
)

func TestTimeUnit_Format(t *testing.T) {
	tests := []struct {
		name string
		tr   TimeUnit
		want string
	}{
		{"Minute", Minute, ".%Y%m%d%H%M"},
		{"Hour", Hour, ".%Y%m%d%H"},
		{"Day", Day, ".%Y%m%d"},
		{"Month", Month, ".%Y%m"},
		{"Year", Year, ".%Y"},
		{"strftime format", "%Y-%m-%d-%H-%M", ".%Y-%m-%d-%H-%M"},
		{"default", TimeUnit(""), ".%Y%m%d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.Format(); got != tt.want {
				t.Errorf("TimeUnit.Format() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeUnit_RotationGap(t *testing.T) {
	tests := []struct {
		name string
		tr   TimeUnit
		want time.Duration
	}{
		{"Minute", Minute, time.Minute},
		{"Hour", Hour, time.Hour},
		{"Day", Day, time.Hour * 24},
		{"Month", Month, time.Hour * 24 * 30},
		{"Year", Year, time.Hour * 24 * 365},
		{"default", TimeUnit("xxx"), time.Hour * 24},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.RotationGap(); got != tt.want {
				t.Errorf("TimeUnit.RotationGap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUpdateFileNameWithTimeFormat(t *testing.T) {
	type args struct {
		originalFilename string
		timeFormat       string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Minute", args{"trpc_{time_format}.log", TimeFormatMinute}, "trpc_%Y%m%d%H%M.log"},
		{"Hour", args{"trpc_{time_format}.log", TimeFormatHour}, "trpc_%Y%m%d%H.log"},
		{"Day", args{"trpc_{time_format}.log", TimeFormatDay}, "trpc_%Y%m%d.log"},
		{"Month", args{"trpc_{time_format}.log", TimeFormatMonth}, "trpc_%Y%m.log"},
		{"Year", args{"trpc_{time_format}.log", TimeFormatYear}, "trpc_%Y.log"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UpdateFileNameWithTimeFormat(tt.args.originalFilename, tt.args.timeFormat); got != tt.want {
				t.Errorf("UpdateFileNameWithTimeFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsTimeFormatTag(t *testing.T) {
	type args struct {
		fileName string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"true", args{"trpc_{time_format}.log"}, true},
		{"false", args{"trpc.log"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ContainsTimeFormatTag(tt.args.fileName); got != tt.want {
				t.Errorf("ContainsTimeFormatTag() = %v, want %v", got, tt.want)
			}
		})
	}
}
