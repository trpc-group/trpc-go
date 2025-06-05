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

// Package errs provides trpc error code type, which contains errcode errmsg.
// These definitions are multi-language universal.
package errs

import (
	"errors"
	"fmt"
	"io"

	"trpc.group/trpc-go/trpc-go/internal/protocol"
)

// trpc return code.
const (
	// RetOK means success.
	RetOK = 0

	// RetServerDecodeFail is the error code of the server decoding error.
	RetServerDecodeFail = 1
	// RetServerEncodeFail is the error code of the server encoding error.
	RetServerEncodeFail = 2
	// RetServerNoService is the error code that the server does not call the corresponding service implementation.
	RetServerNoService = 11
	// RetServerNoFunc is the error code that the server does not call the corresponding interface implementation.
	RetServerNoFunc = 12
	// RetServerTimeout is the error code that the request timed out in the server queue.
	RetServerTimeout = 21
	// RetServerOverload is the error code that the request is overloaded on the server side.
	RetServerOverload = 22
	// RetServerThrottled is the error code of the server's current limit.
	RetServerThrottled = 23
	// RetServerFullLinkTimeout is the server full link timeout error code.
	RetServerFullLinkTimeout = 24
	// RetServerSystemErr is the error code of the server system error.
	RetServerSystemErr = 31
	// RetServerAuthFail is the error code for authentication failure.
	RetServerAuthFail = 41
	// RetServerValidateFail is the error code for the failure of automatic validation of request parameters.
	RetServerValidateFail = 51

	// RetClientTimeout is the error code that the request timed out on the client side.
	RetClientTimeout = 101
	// RetClientFullLinkTimeout is the client full link timeout error code.
	RetClientFullLinkTimeout = 102
	// RetClientConnectFail is the error code of the client connection error.
	RetClientConnectFail = 111
	// RetClientEncodeFail is the error code of the client encoding error.
	RetClientEncodeFail = 121
	// RetClientDecodeFail is the error code of the client decoding error.
	RetClientDecodeFail = 122
	// RetClientThrottled is the error code of the client's current limit.
	RetClientThrottled = 123
	// RetClientOverload is the error code for client overload.
	RetClientOverload = 124
	// RetClientRouteErr is the error code for the wrong ip route selected by the client.
	RetClientRouteErr = 131
	// RetClientNetErr is the error code of the client network error.
	RetClientNetErr = 141
	// RetClientValidateFail is the error code for the failure of automatic validation of response parameters.
	RetClientValidateFail = 151
	// RetClientCanceled is the error code for the upstream caller to cancel the request in advance.
	RetClientCanceled = 161
	// RetClientReadFrameErr is the error code of the client read frame error.
	RetClientReadFrameErr = 171
	// RetClientStreamQueueFull is the error code of the client stream queue full.
	RetClientStreamQueueFull = 201
	// RetClientStreamReadEnd is the error code of the client stream end error while receiving data.
	RetClientStreamReadEnd = 351
	// RetClientStreamInitErr is the error code of the client stream init error.
	RetClientStreamInitErr = 361

	// RetInvalidArgument indicates client specified an invalid argument.
	RetInvalidArgument = 400
	// RetNotFound means some requested entity (e.g., file or directory) was not found.
	RetNotFound = 404

	// RetUnknown is the error code for unspecified errors.
	RetUnknown = 999
)

// Err frame error value.
var (
	// ErrOK means success.
	ErrOK error

	// ErrServerNoService is an error that the server does not call the corresponding service implementation.
	ErrServerNoService = NewFrameError(RetServerNoService, "server router no service")
	// ErrServerNoFunc is an error that the server does not call the corresponding interface implementation.
	ErrServerNoFunc = NewFrameError(RetServerNoFunc, "server router no rpc method")
	// ErrServerTimeout is the error that the request timed out in the server queue.
	ErrServerTimeout = NewFrameError(RetServerTimeout, "server message handle timeout")
	// ErrServerOverload is an error that the request is overloaded on the server side.
	ErrServerOverload = NewFrameError(RetServerOverload, "server overload")
	// ErrServerRoutinePoolBusy is an error that the request is overloaded on the server side.
	ErrServerRoutinePoolBusy = NewFrameError(RetServerOverload, "server goroutine pool too small")
	// ErrServerClose is a server system error.
	ErrServerClose = NewFrameError(RetServerSystemErr, "server close")

	// ErrServerNoResponse is a server-side unresponsive error.
	ErrServerNoResponse = NewFrameError(RetOK, "server no response")
	// ErrClientNoResponse is the error of the client not responding.
	ErrClientNoResponse = NewFrameError(RetOK, "client no response")

	// ErrUnknown is an unknown error.
	ErrUnknown = NewFrameError(RetUnknown, "unknown error")
)

// ErrorType is the error code type, including framework error code and business error code.
const (
	ErrorTypeFramework       = 1
	ErrorTypeBusiness        = 2
	ErrorTypeCalleeFramework = 3 // The Error code and Msg come from the downstream framework.
)

func typeDesc(t int) string {
	switch t {
	case ErrorTypeFramework:
		return "framework"
	case ErrorTypeCalleeFramework:
		return "callee framework"
	default:
		return "business"
	}
}

const (
	// Success is the success prompt string.
	Success = "success"
)

// Error is the error code structure which contains error code type and error message.
type Error struct {
	Type int
	Code int32
	Msg  string
	Desc string

	cause error      // internal error, form the error chain.
	stack stackTrace // call stack, if the error chain already has a stack, it will not be set.
}

// Error implements the error interface and returns the error description.
func (e *Error) Error() string {
	if e == nil {
		return Success
	}

	if e.cause != nil {
		return fmt.Sprintf("type:%s, code:%d, msg:%s, caused by %s",
			typeDesc(e.Type), e.Code, e.Msg, e.cause.Error())
	}
	return fmt.Sprintf("type:%s, code:%d, msg:%s", typeDesc(e.Type), e.Code, e.Msg)
}

// Format implements the fmt.Formatter interface.
func (e *Error) Format(s fmt.State, verb rune) {
	var stackTrace stackTrace
	defer func() {
		if stackTrace != nil {
			stackTrace.Format(s, verb)
		}
	}()
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "type:%s, code:%d, msg:%s", typeDesc(e.Type), e.Code, e.Msg)
			if e.stack != nil {
				stackTrace = e.stack
			}
			if e.cause != nil {
				_, _ = fmt.Fprintf(s, "\nCause by %+v", e.cause)
			}
			return
		}
		_, _ = io.WriteString(s, e.Error())
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	default:
		_, _ = fmt.Fprintf(s, "%%!%c(errs.Error=%s)", verb, e.Error())
	}
}

// Unwrap supports Go 1.13+ error chains.
func (e *Error) Unwrap() error {
	// Check nil error to avoid panic.
	if e == nil {
		return nil
	}
	return e.cause
}

// IsTimeout checks whether this error is a timeout error with error type typ.
func (e *Error) IsTimeout(typ int) bool {
	return e.Type == typ &&
		(e.Code == RetClientTimeout ||
			e.Code == RetClientFullLinkTimeout ||
			e.Code == RetServerTimeout ||
			e.Code == RetServerFullLinkTimeout)
}

// New creates an error, which defaults to the business error type to improve business development efficiency.
func New(code int, msg string) error {
	err := &Error{
		Type: ErrorTypeBusiness,
		Code: int32(code),
		Msg:  msg,
	}
	if traceable {
		err.stack = callers()
	}
	return err
}

// Newf creates an error, the default is the business error type, msg supports format strings.
func Newf(code int, format string, params ...interface{}) error {
	msg := fmt.Sprintf(format, params...)
	err := &Error{
		Type: ErrorTypeBusiness,
		Code: int32(code),
		Msg:  msg,
	}
	if traceable {
		err.stack = callers()
	}
	return err
}

// Wrap creates a new error contains input error.
// only add stack when traceable is true and the input type is not Error, this will ensure that there is no multiple
// stacks in the error chain.
func Wrap(err error, code int, msg string) error {
	if err == nil {
		return nil
	}
	wrapErr := &Error{
		Type:  ErrorTypeBusiness,
		Code:  int32(code),
		Msg:   msg,
		cause: err,
	}
	var e *Error
	// the error chain does not contain item which type is Error, add stack.
	if traceable && !errors.As(err, &e) {
		wrapErr.stack = callers()
	}
	return wrapErr
}

// Wrapf the same as Wrap, msg supports format strings.
func Wrapf(err error, code int, format string, params ...interface{}) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, params...)
	wrapErr := &Error{
		Type:  ErrorTypeBusiness,
		Code:  int32(code),
		Msg:   msg,
		cause: err,
	}
	var e *Error
	// the error chain does not contain item which type is Error, add stack.
	if traceable && !errors.As(err, &e) {
		wrapErr.stack = callers()
	}
	return wrapErr
}

// NewFrameError creates a frame error.
func NewFrameError(code int, msg string) error {
	err := &Error{
		Type: ErrorTypeFramework,
		Code: int32(code),
		Msg:  msg,
		Desc: protocol.TRPC,
	}
	if traceable {
		err.stack = callers()
	}
	return err
}

// NewCalleeFrameError creates a callee frame error.
func NewCalleeFrameError(code int, msg string) error {
	err := &Error{
		Type: ErrorTypeCalleeFramework,
		Code: int32(code),
		Msg:  msg,
		Desc: protocol.TRPC,
	}
	if traceable {
		err.stack = callers()
	}
	return err
}

// WrapFrameError the same as Wrap, except type is ErrorTypeFramework
func WrapFrameError(err error, code int, msg string) error {
	if err == nil {
		return nil
	}
	wrapErr := &Error{
		Type:  ErrorTypeFramework,
		Code:  int32(code),
		Msg:   msg,
		Desc:  protocol.TRPC,
		cause: err,
	}
	var e *Error
	// the error chain does not contain item which type is Error, add stack.
	if traceable && !errors.As(err, &e) {
		wrapErr.stack = callers()
	}
	return wrapErr
}

// Code gets the error code through error.
func Code(e error) int {
	if e == nil {
		return RetOK
	}

	// Doing type assertion first has a slight performance boost over just using errors.As
	// because of avoiding reflect when the assertion is probably true.
	err, ok := e.(*Error)
	if !ok && !errors.As(e, &err) {
		return RetUnknown
	}
	if err == nil {
		return RetOK
	}
	return int(err.Code)
}

// Msg gets error msg through error.
func Msg(e error) string {
	if e == nil {
		return Success
	}
	err, ok := e.(*Error)
	if !ok && !errors.As(e, &err) {
		return e.Error()
	}
	if err == (*Error)(nil) {
		return Success
	}
	// For cases of error chains, err.Error() will print the entire chain,
	// including the current error and the nested error messages, in an appropriate format.
	if err.Unwrap() != nil {
		return err.Error()
	}
	return err.Msg
}

// Cause returns the internal error.
// Deprecated: use Unwrap instead.
func (e *Error) Cause() error { return e.Unwrap() }
