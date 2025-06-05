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

package log

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
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

func TestConfig_LogName(t *testing.T) {
	yamlData := `
writer: "console"
writer_config:
  log_path: "/var/log/app.log"
formatter: "json"
formatter_config:
  message_key: S
level: "info"
caller_skip: 2
enable_color: true
logger_name: "test"
`
	var config OutputConfig
	err := yaml.Unmarshal([]byte(yamlData), &config)
	assert.NoError(t, err)
	assert.Equal(t, "console", config.Writer)
	assert.Equal(t, "json", config.Formatter)
	assert.Equal(t, "info", config.Level)
	assert.Equal(t, 2, config.CallerSkip)
	assert.Equal(t, true, config.EnableColor)
	assert.Equal(t, "test", config.LoggerName)
}
