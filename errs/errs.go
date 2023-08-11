// Package errs provides trpc error code type, which contains errcode errmsg.
// These definitions are multi-language universal.
package errs

import (
	"errors"
	"fmt"
	"io"

	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

// trpc return code.
const (
	// RetOK means success.
	RetOK = trpcpb.TrpcRetCode_TRPC_INVOKE_SUCCESS

	// RetServerDecodeFail is the error code of the server decoding error.
	RetServerDecodeFail = trpcpb.TrpcRetCode_TRPC_SERVER_DECODE_ERR
	// RetServerEncodeFail is the error code of the server encoding error.
	RetServerEncodeFail = trpcpb.TrpcRetCode_TRPC_SERVER_ENCODE_ERR
	// RetServerNoService is the error code that the server does not call the corresponding service implementation.
	RetServerNoService = trpcpb.TrpcRetCode_TRPC_SERVER_NOSERVICE_ERR
	// RetServerNoFunc is the error code that the server does not call the corresponding interface implementation.
	RetServerNoFunc = trpcpb.TrpcRetCode_TRPC_SERVER_NOFUNC_ERR
	// RetServerTimeout is the error code that the request timed out in the server queue.
	RetServerTimeout = trpcpb.TrpcRetCode_TRPC_SERVER_TIMEOUT_ERR
	// RetServerOverload is the error code that the request is overloaded on the server side.
	RetServerOverload = trpcpb.TrpcRetCode_TRPC_SERVER_OVERLOAD_ERR
	// RetServerThrottled is the error code of the server's current limit.
	RetServerThrottled = trpcpb.TrpcRetCode_TRPC_SERVER_LIMITED_ERR
	// RetServerFullLinkTimeout is the server full link timeout error code.
	RetServerFullLinkTimeout = trpcpb.TrpcRetCode_TRPC_SERVER_FULL_LINK_TIMEOUT_ERR
	// RetServerSystemErr is the error code of the server system error.
	RetServerSystemErr = trpcpb.TrpcRetCode_TRPC_SERVER_SYSTEM_ERR
	// RetServerAuthFail is the error code for authentication failure.
	RetServerAuthFail = trpcpb.TrpcRetCode_TRPC_SERVER_AUTH_ERR
	// RetServerValidateFail is the error code for the failure of automatic validation of request parameters.
	RetServerValidateFail = trpcpb.TrpcRetCode_TRPC_SERVER_VALIDATE_ERR

	// RetClientTimeout is the error code that the request timed out on the client side.
	RetClientTimeout = trpcpb.TrpcRetCode_TRPC_CLIENT_INVOKE_TIMEOUT_ERR
	// RetClientFullLinkTimeout is the client full link timeout error code.
	RetClientFullLinkTimeout = trpcpb.TrpcRetCode_TRPC_CLIENT_FULL_LINK_TIMEOUT_ERR
	// RetClientConnectFail is the error code of the client connection error.
	RetClientConnectFail = trpcpb.TrpcRetCode_TRPC_CLIENT_CONNECT_ERR
	// RetClientEncodeFail is the error code of the client encoding error.
	RetClientEncodeFail = trpcpb.TrpcRetCode_TRPC_CLIENT_ENCODE_ERR
	// RetClientDecodeFail is the error code of the client decoding error.
	RetClientDecodeFail = trpcpb.TrpcRetCode_TRPC_CLIENT_DECODE_ERR
	// RetClientThrottled is the error code of the client's current limit.
	RetClientThrottled = trpcpb.TrpcRetCode_TRPC_CLIENT_LIMITED_ERR
	// RetClientOverload is the error code for client overload.
	RetClientOverload = trpcpb.TrpcRetCode_TRPC_CLIENT_OVERLOAD_ERR
	// RetClientRouteErr is the error code for the wrong ip route selected by the client.
	RetClientRouteErr = trpcpb.TrpcRetCode_TRPC_CLIENT_ROUTER_ERR
	// RetClientNetErr is the error code of the client network error.
	RetClientNetErr = trpcpb.TrpcRetCode_TRPC_CLIENT_NETWORK_ERR
	// RetClientValidateFail is the error code for the failure of automatic validation of response parameters.
	RetClientValidateFail = trpcpb.TrpcRetCode_TRPC_CLIENT_VALIDATE_ERR
	// RetClientCanceled is the error code for the upstream caller to cancel the request in advance.
	RetClientCanceled = trpcpb.TrpcRetCode_TRPC_CLIENT_CANCELED_ERR
	// RetClientReadFrameErr is the error code of the client read frame error.
	RetClientReadFrameErr = trpcpb.TrpcRetCode_TRPC_CLIENT_READ_FRAME_ERR
	// RetClientStreamQueueFull is the error code of the client stream queue full.
	RetClientStreamQueueFull = trpcpb.TrpcRetCode_TRPC_STREAM_SERVER_NETWORK_ERR
	// RetClientStreamReadEnd is the error code of the client stream end error while receiving data.
	RetClientStreamReadEnd = trpcpb.TrpcRetCode_TRPC_STREAM_CLIENT_READ_END

	// RetUnknown is the error code for unspecified errors.
	RetUnknown = trpcpb.TrpcRetCode_TRPC_INVOKE_UNKNOWN_ERR
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
	ErrorTypeCalleeFramework = 3 // The error code returned by the client call
	// represents the downstream framework error code.
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
	Code trpcpb.TrpcRetCode
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
			if e.Unwrap() != nil {
				_, _ = fmt.Fprintf(s, "\nCause by %+v", e.Unwrap())
			}
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, e.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", e.Error())
	default:
		_, _ = fmt.Fprintf(s, "%%!%c(errs.Error=%s)", verb, e.Error())
	}
}

// Unwrap support Go 1.13+ error chains.
func (e *Error) Unwrap() error { return e.cause }

// IsTimeout checks whether this error is a timeout error with error type typ.
func (e *Error) IsTimeout(typ int) bool {
	return e.Type == typ &&
		(e.Code == RetClientTimeout ||
			e.Code == RetClientFullLinkTimeout ||
			e.Code == RetServerTimeout ||
			e.Code == RetServerFullLinkTimeout)
}

// ErrCode permits any integer defined in https://go.dev/ref/spec#Numeric_types
type ErrCode interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~int | ~uintptr
}

// New creates an error, which defaults to the business error type to improve business development efficiency.
func New[T ErrCode](code T, msg string) error {
	err := &Error{
		Type: ErrorTypeBusiness,
		Code: trpcpb.TrpcRetCode(code),
		Msg:  msg,
	}
	if traceable {
		err.stack = callers()
	}
	return err
}

// Newf creates an error, the default is the business error type, msg supports format strings.
func Newf[T ErrCode](code T, format string, params ...interface{}) error {
	msg := fmt.Sprintf(format, params...)
	err := &Error{
		Type: ErrorTypeBusiness,
		Code: trpcpb.TrpcRetCode(code),
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
func Wrap[T ErrCode](err error, code T, msg string) error {
	if err == nil {
		return nil
	}
	wrapErr := &Error{
		Type:  ErrorTypeBusiness,
		Code:  trpcpb.TrpcRetCode(code),
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
func Wrapf[T ErrCode](err error, code T, format string, params ...interface{}) error {
	if err == nil {
		return nil
	}
	msg := fmt.Sprintf(format, params...)
	wrapErr := &Error{
		Type:  ErrorTypeBusiness,
		Code:  trpcpb.TrpcRetCode(code),
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
func NewFrameError[T ErrCode](code T, msg string) error {
	err := &Error{
		Type: ErrorTypeFramework,
		Code: trpcpb.TrpcRetCode(code),
		Msg:  msg,
		Desc: "trpc",
	}
	if traceable {
		err.stack = callers()
	}
	return err
}

// WrapFrameError the same as Wrap, except type is ErrorTypeFramework
func WrapFrameError[T ErrCode](err error, code T, msg string) error {
	if err == nil {
		return nil
	}
	wrapErr := &Error{
		Type:  ErrorTypeFramework,
		Code:  trpcpb.TrpcRetCode(code),
		Msg:   msg,
		Desc:  "trpc",
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
func Code(e error) trpcpb.TrpcRetCode {
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
	return err.Code
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
