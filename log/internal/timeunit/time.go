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

// Package timeunit provides a log for the framework and applications.
package timeunit

import (
	"regexp"
	"strings"
	"time"
)

// Some common used timeunit formats.
const (
	// TimeFormatMinute is accurate to the minute.
	TimeFormatMinute = "%Y%m%d%H%M"
	// TimeFormatHour is accurate to the hour.
	TimeFormatHour = "%Y%m%d%H"
	// TimeFormatDay is accurate to the day.
	TimeFormatDay = "%Y%m%d"
	// TimeFormatMonth is accurate to the month.
	TimeFormatMonth = "%Y%m"
	// TimeFormatYear is accurate to the year.
	TimeFormatYear = "%Y"
)

// TimeUnit is the timeunit unit by which files are split, one of minute/hour/day/month/year.
type TimeUnit string

const (
	// Minute splits by the minute.
	Minute = "minute"
	// Hour splits by the hour.
	Hour = "hour"
	// Day splits by the day.
	Day = "day"
	// Month splits by the month.
	Month = "month"
	// Year splits by the year.
	Year = "year"
)

// Format returns a string preceding with `.`. Use TimeFormatDay as default.
func (t TimeUnit) Format() string {
	var timeFmt string
	switch t {
	case "", Day:
		timeFmt = TimeFormatDay
	case Minute:
		timeFmt = TimeFormatMinute
	case Hour:
		timeFmt = TimeFormatHour
	case Month:
		timeFmt = TimeFormatMonth
	case Year:
		timeFmt = TimeFormatYear
	default:
		timeFmt = string(t)
	}
	return "." + timeFmt
}

// RotationGap returns the timeunit.Duration for timeunit unit. Use one day as the default.
func (t TimeUnit) RotationGap() time.Duration {
	switch t {
	case Minute:
		return time.Minute
	case Hour:
		return time.Hour
	case Day:
		return time.Hour * 24
	case Month:
		return time.Hour * 24 * 30
	case Year:
		return time.Hour * 24 * 365
	default:
		return time.Hour * 24
	}
}

const (
	// TimeFormatTag is a placeholder used to represent a date format in filenames.
	// It can be used to identify or replace specific parts of a filename that
	// are intended to include a date format.
	TimeFormatTag = "{time_format}"
)

// Define a mapping that associates time formats with their corresponding regex patterns
var (
	timeFormatPatterns = map[string]string{
		TimeFormatMinute: "\\d{12}", // 12-digit pattern (YYYYMMDDHHMM)
		TimeFormatHour:   "\\d{10}", // 10-digit pattern (YYYYMMDDHH)
		TimeFormatDay:    "\\d{8}",  // 8-digit pattern (YYYYMMDD)
		TimeFormatMonth:  "\\d{6}",  // 6-digit pattern (YYYYMM)
		TimeFormatYear:   "\\d{4}",  // 4-digit pattern (YYYY)
	}
)

// GenerateTimeFormatRegex creates a regex pattern from the file prefix.
// It replaces the time format tag with the specified format.
func GenerateTimeFormatRegex(filePrefix string, timeFormat string) (*regexp.Regexp, error) {
	// Remove the first occurrence of '.' from the time format (e.g., ".%Y%m%d" becomes "%Y%m%d")
	cleanedTimeFormat := strings.Replace(timeFormat, ".", "", 1)

	// Get the corresponding pattern based on cleanedTimeFormat
	if pattern, exists := timeFormatPatterns[cleanedTimeFormat]; exists {
		filePrefix = strings.ReplaceAll(filePrefix, TimeFormatTag, pattern)
	}

	return regexp.Compile(filePrefix)
}

// UpdateFileNameWithTimeFormat updates the filename by replacing the time format tag with the specified time format.
// It returns the updated filename.
func UpdateFileNameWithTimeFormat(originalFilename, timeFormat string) string {
	// Remove the first occurrence of '.' from the time format (e.g., ".%Y%m%d" becomes "%Y%m%d")
	cleanedTimeFormat := strings.Replace(timeFormat, ".", "", 1)

	// Replace the time format tag in the filename with the specified time format
	updatedFilename := strings.ReplaceAll(originalFilename, TimeFormatTag, cleanedTimeFormat)

	return updatedFilename
}

// ContainsTimeFormatTag checks if the given filename contains a time format tag.
// It returns true if the filename contains the time format tag, otherwise false.
func ContainsTimeFormatTag(fileName string) bool {
	// Check if the filename contains the time format tag.
	return strings.Index(fileName, TimeFormatTag) != -1
}
