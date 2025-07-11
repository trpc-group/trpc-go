//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Code generated by protoc-gen-secv. DO NOT EDIT.
// source: test.proto

package protocols

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
)

// Validate checks the field values on Empty with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Empty) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Empty with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in EmptyMultiError, or nil if none found.
func (m *Empty) ValidateAll() error {
	return m.validate(true)
}

func (m *Empty) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if len(errors) > 0 {
		return EmptyMultiError(errors)
	}
	return nil
}

// EmptyMultiError is an error wrapping multiple validation errors returned by
// Empty.ValidateAll() if the designated constraints aren't met.
type EmptyMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m EmptyMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m EmptyMultiError) AllErrors() []error { return m }

// EmptyValidationError is the validation error returned by Empty.Validate if
// the designated constraints aren't met.
type EmptyValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e EmptyValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e EmptyValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e EmptyValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e EmptyValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e EmptyValidationError) ErrorName() string { return "EmptyValidationError" }

// Error satisfies the builtin error interface
func (e EmptyValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sEmpty.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = EmptyValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = EmptyValidationError{}

// Validate checks the field values on Payload with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Payload) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Payload with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in PayloadMultiError, or nil if none found.
func (m *Payload) ValidateAll() error {
	return m.validate(true)
}

func (m *Payload) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Type

	// no validation rules for Body

	if len(errors) > 0 {
		return PayloadMultiError(errors)
	}
	return nil
}

// PayloadMultiError is an error wrapping multiple validation errors returned
// by Payload.ValidateAll() if the designated constraints aren't met.
type PayloadMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m PayloadMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m PayloadMultiError) AllErrors() []error { return m }

// PayloadValidationError is the validation error returned by Payload.Validate
// if the designated constraints aren't met.
type PayloadValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e PayloadValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e PayloadValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e PayloadValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e PayloadValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e PayloadValidationError) ErrorName() string { return "PayloadValidationError" }

// Error satisfies the builtin error interface
func (e PayloadValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sPayload.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = PayloadValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = PayloadValidationError{}

// Validate checks the field values on SimpleRequest with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *SimpleRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on SimpleRequest with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in SimpleRequestMultiError, or
// nil if none found.
func (m *SimpleRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *SimpleRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ResponseType

	// no validation rules for ResponseSize

	if all {
		switch v := interface{}(m.GetPayload()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, SimpleRequestValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, SimpleRequestValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetPayload()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return SimpleRequestValidationError{
				field:  "Payload",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if isTsecstr := m._validateTsecstr(m.GetUsername()); !isTsecstr {
		err := SimpleRequestValidationError{
			field:  "Username",
			reason: "value contains invalid strings",
		}
		if !all {
			return err
		}
		errors = append(errors, err)
	}

	// no validation rules for FillUsername

	// no validation rules for FillOauthScope

	if len(errors) > 0 {
		return SimpleRequestMultiError(errors)
	}
	return nil
}

func (m *SimpleRequest) _validateTsecstr(checktsecstr string) bool {

	for _, r := range checktsecstr {

		isLetterInvalid := false
		isNumberInvalid := false
		isSymbolInvalid := false

		if !unicode.IsLetter(r) {
			isLetterInvalid = true
		}

		if !unicode.IsNumber(r) {
			isNumberInvalid = true
		}

		if (string(r) != "=") && (string(r) != "-") &&
			(string(r) != "+") && (string(r) != "/") &&
			(string(r) != "@") && (string(r) != "#") &&
			(string(r) != "_") {
			isSymbolInvalid = true
		}

		if isLetterInvalid && isNumberInvalid && isSymbolInvalid {
			return false
		}

	}

	return true
}

// SimpleRequestMultiError is an error wrapping multiple validation errors
// returned by SimpleRequest.ValidateAll() if the designated constraints
// aren't met.
type SimpleRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m SimpleRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m SimpleRequestMultiError) AllErrors() []error { return m }

// SimpleRequestValidationError is the validation error returned by
// SimpleRequest.Validate if the designated constraints aren't met.
type SimpleRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e SimpleRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e SimpleRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e SimpleRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e SimpleRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e SimpleRequestValidationError) ErrorName() string { return "SimpleRequestValidationError" }

// Error satisfies the builtin error interface
func (e SimpleRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sSimpleRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = SimpleRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = SimpleRequestValidationError{}

// Validate checks the field values on SimpleResponse with the rules defined in
// the proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *SimpleResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on SimpleResponse with the rules defined
// in the proto definition for this message. If any rules are violated, the
// result is a list of violation errors wrapped in SimpleResponseMultiError,
// or nil if none found.
func (m *SimpleResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *SimpleResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetPayload()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, SimpleResponseValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, SimpleResponseValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetPayload()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return SimpleResponseValidationError{
				field:  "Payload",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for Username

	// no validation rules for OauthScope

	if len(errors) > 0 {
		return SimpleResponseMultiError(errors)
	}
	return nil
}

// SimpleResponseMultiError is an error wrapping multiple validation errors
// returned by SimpleResponse.ValidateAll() if the designated constraints
// aren't met.
type SimpleResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m SimpleResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m SimpleResponseMultiError) AllErrors() []error { return m }

// SimpleResponseValidationError is the validation error returned by
// SimpleResponse.Validate if the designated constraints aren't met.
type SimpleResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e SimpleResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e SimpleResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e SimpleResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e SimpleResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e SimpleResponseValidationError) ErrorName() string { return "SimpleResponseValidationError" }

// Error satisfies the builtin error interface
func (e SimpleResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sSimpleResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = SimpleResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = SimpleResponseValidationError{}

// Validate checks the field values on StreamingInputCallRequest with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *StreamingInputCallRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on StreamingInputCallRequest with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// StreamingInputCallRequestMultiError, or nil if none found.
func (m *StreamingInputCallRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *StreamingInputCallRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetPayload()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, StreamingInputCallRequestValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, StreamingInputCallRequestValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetPayload()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return StreamingInputCallRequestValidationError{
				field:  "Payload",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return StreamingInputCallRequestMultiError(errors)
	}
	return nil
}

// StreamingInputCallRequestMultiError is an error wrapping multiple validation
// errors returned by StreamingInputCallRequest.ValidateAll() if the
// designated constraints aren't met.
type StreamingInputCallRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m StreamingInputCallRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m StreamingInputCallRequestMultiError) AllErrors() []error { return m }

// StreamingInputCallRequestValidationError is the validation error returned by
// StreamingInputCallRequest.Validate if the designated constraints aren't met.
type StreamingInputCallRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e StreamingInputCallRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e StreamingInputCallRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e StreamingInputCallRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e StreamingInputCallRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e StreamingInputCallRequestValidationError) ErrorName() string {
	return "StreamingInputCallRequestValidationError"
}

// Error satisfies the builtin error interface
func (e StreamingInputCallRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sStreamingInputCallRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = StreamingInputCallRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = StreamingInputCallRequestValidationError{}

// Validate checks the field values on StreamingInputCallResponse with the
// rules defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *StreamingInputCallResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on StreamingInputCallResponse with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// StreamingInputCallResponseMultiError, or nil if none found.
func (m *StreamingInputCallResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *StreamingInputCallResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for AggregatedPayloadSize

	if len(errors) > 0 {
		return StreamingInputCallResponseMultiError(errors)
	}
	return nil
}

// StreamingInputCallResponseMultiError is an error wrapping multiple
// validation errors returned by StreamingInputCallResponse.ValidateAll() if
// the designated constraints aren't met.
type StreamingInputCallResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m StreamingInputCallResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m StreamingInputCallResponseMultiError) AllErrors() []error { return m }

// StreamingInputCallResponseValidationError is the validation error returned
// by StreamingInputCallResponse.Validate if the designated constraints aren't met.
type StreamingInputCallResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e StreamingInputCallResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e StreamingInputCallResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e StreamingInputCallResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e StreamingInputCallResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e StreamingInputCallResponseValidationError) ErrorName() string {
	return "StreamingInputCallResponseValidationError"
}

// Error satisfies the builtin error interface
func (e StreamingInputCallResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sStreamingInputCallResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = StreamingInputCallResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = StreamingInputCallResponseValidationError{}

// Validate checks the field values on ResponseParameters with the rules
// defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *ResponseParameters) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on ResponseParameters with the rules
// defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// ResponseParametersMultiError, or nil if none found.
func (m *ResponseParameters) ValidateAll() error {
	return m.validate(true)
}

func (m *ResponseParameters) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Size

	if all {
		switch v := interface{}(m.GetInterval()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ResponseParametersValidationError{
					field:  "Interval",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ResponseParametersValidationError{
					field:  "Interval",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetInterval()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ResponseParametersValidationError{
				field:  "Interval",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return ResponseParametersMultiError(errors)
	}
	return nil
}

// ResponseParametersMultiError is an error wrapping multiple validation errors
// returned by ResponseParameters.ValidateAll() if the designated constraints
// aren't met.
type ResponseParametersMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ResponseParametersMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ResponseParametersMultiError) AllErrors() []error { return m }

// ResponseParametersValidationError is the validation error returned by
// ResponseParameters.Validate if the designated constraints aren't met.
type ResponseParametersValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ResponseParametersValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ResponseParametersValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ResponseParametersValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ResponseParametersValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ResponseParametersValidationError) ErrorName() string {
	return "ResponseParametersValidationError"
}

// Error satisfies the builtin error interface
func (e ResponseParametersValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sResponseParameters.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ResponseParametersValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ResponseParametersValidationError{}

// Validate checks the field values on StreamingOutputCallRequest with the
// rules defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *StreamingOutputCallRequest) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on StreamingOutputCallRequest with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// StreamingOutputCallRequestMultiError, or nil if none found.
func (m *StreamingOutputCallRequest) ValidateAll() error {
	return m.validate(true)
}

func (m *StreamingOutputCallRequest) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for ResponseType

	for idx, item := range m.GetResponseParameters() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, StreamingOutputCallRequestValidationError{
						field:  fmt.Sprintf("ResponseParameters[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, StreamingOutputCallRequestValidationError{
						field:  fmt.Sprintf("ResponseParameters[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return StreamingOutputCallRequestValidationError{
					field:  fmt.Sprintf("ResponseParameters[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if all {
		switch v := interface{}(m.GetPayload()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, StreamingOutputCallRequestValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, StreamingOutputCallRequestValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetPayload()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return StreamingOutputCallRequestValidationError{
				field:  "Payload",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return StreamingOutputCallRequestMultiError(errors)
	}
	return nil
}

// StreamingOutputCallRequestMultiError is an error wrapping multiple
// validation errors returned by StreamingOutputCallRequest.ValidateAll() if
// the designated constraints aren't met.
type StreamingOutputCallRequestMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m StreamingOutputCallRequestMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m StreamingOutputCallRequestMultiError) AllErrors() []error { return m }

// StreamingOutputCallRequestValidationError is the validation error returned
// by StreamingOutputCallRequest.Validate if the designated constraints aren't met.
type StreamingOutputCallRequestValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e StreamingOutputCallRequestValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e StreamingOutputCallRequestValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e StreamingOutputCallRequestValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e StreamingOutputCallRequestValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e StreamingOutputCallRequestValidationError) ErrorName() string {
	return "StreamingOutputCallRequestValidationError"
}

// Error satisfies the builtin error interface
func (e StreamingOutputCallRequestValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sStreamingOutputCallRequest.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = StreamingOutputCallRequestValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = StreamingOutputCallRequestValidationError{}

// Validate checks the field values on StreamingOutputCallResponse with the
// rules defined in the proto definition for this message. If any rules are
// violated, the first error encountered is returned, or nil if there are no violations.
func (m *StreamingOutputCallResponse) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on StreamingOutputCallResponse with the
// rules defined in the proto definition for this message. If any rules are
// violated, the result is a list of violation errors wrapped in
// StreamingOutputCallResponseMultiError, or nil if none found.
func (m *StreamingOutputCallResponse) ValidateAll() error {
	return m.validate(true)
}

func (m *StreamingOutputCallResponse) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	if all {
		switch v := interface{}(m.GetPayload()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, StreamingOutputCallResponseValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, StreamingOutputCallResponseValidationError{
					field:  "Payload",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetPayload()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return StreamingOutputCallResponseValidationError{
				field:  "Payload",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	if len(errors) > 0 {
		return StreamingOutputCallResponseMultiError(errors)
	}
	return nil
}

// StreamingOutputCallResponseMultiError is an error wrapping multiple
// validation errors returned by StreamingOutputCallResponse.ValidateAll() if
// the designated constraints aren't met.
type StreamingOutputCallResponseMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m StreamingOutputCallResponseMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m StreamingOutputCallResponseMultiError) AllErrors() []error { return m }

// StreamingOutputCallResponseValidationError is the validation error returned
// by StreamingOutputCallResponse.Validate if the designated constraints
// aren't met.
type StreamingOutputCallResponseValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e StreamingOutputCallResponseValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e StreamingOutputCallResponseValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e StreamingOutputCallResponseValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e StreamingOutputCallResponseValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e StreamingOutputCallResponseValidationError) ErrorName() string {
	return "StreamingOutputCallResponseValidationError"
}

// Error satisfies the builtin error interface
func (e StreamingOutputCallResponseValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sStreamingOutputCallResponse.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = StreamingOutputCallResponseValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = StreamingOutputCallResponseValidationError{}
