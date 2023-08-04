package log_test

import (
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/log"
)

var defaultConfig = []log.OutputConfig{
	{
		Writer:    "console",
		Level:     "debug",
		Formatter: "console",
		FormatConfig: log.FormatConfig{
			TimeFmt: "2006.01.02 15:04:05",
		},
	},
	{
		Writer:    "file",
		Level:     "info",
		Formatter: "json",
		WriteConfig: log.WriteConfig{
			Filename:   "trpc_size.log",
			RollType:   "size",
			MaxAge:     7,
			MaxBackups: 10,
			MaxSize:    100,
		},
		FormatConfig: log.FormatConfig{
			TimeFmt: "2006.01.02 15:04:05",
		},
	},
	{
		Writer:    "file",
		Level:     "info",
		Formatter: "json",
		WriteConfig: log.WriteConfig{
			Filename:   "trpc_time.log",
			RollType:   "time",
			MaxAge:     7,
			MaxBackups: 10,
			MaxSize:    100,
			TimeUnit:   log.Day,
		},
		FormatConfig: log.FormatConfig{
			TimeFmt: "2006-01-02 15:04:05",
		},
	},
}

func TestTimeUnit_Format(t *testing.T) {
	tests := []struct {
		name string
		tr   log.TimeUnit
		want string
	}{
		{"Minute", log.Minute, ".%Y%m%d%H%M"},
		{"Hour", log.Hour, ".%Y%m%d%H"},
		{"Day", log.Day, ".%Y%m%d"},
		{"Month", log.Month, ".%Y%m"},
		{"Year", log.Year, ".%Y"},
		{"default", log.TimeUnit("xxx"), ".%Y%m%d"},
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
		tr   log.TimeUnit
		want time.Duration
	}{
		{"Minute", log.Minute, time.Minute},
		{"Hour", log.Hour, time.Hour},
		{"Day", log.Day, time.Hour * 24},
		{"Month", log.Month, time.Hour * 24 * 30},
		{"Year", log.Year, time.Hour * 24 * 365},
		{"default", log.TimeUnit("xxx"), time.Hour * 24},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.RotationGap(); got != tt.want {
				t.Errorf("TimeUnit.RotationGap() = %v, want %v", got, tt.want)
			}
		})
	}
}
