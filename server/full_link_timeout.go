package server

import (
	"context"

	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/filter"
)

// mayConvert2FullLinkTimeout infers whether an error is caused by a full-link
// timeout. If so, it returns the full-link timeout error.
func mayConvert2FullLinkTimeout(
	ctx context.Context,
	req interface{},
	next filter.ServerHandleFunc,
) (interface{}, error) {
	rsp, err := next(ctx, req)
	if e, ok := err.(*errs.Error); ok &&
		e.IsTimeout(errs.ErrorTypeFramework) &&
		e.Code != errs.RetClientTimeout {
		e.Code = errs.RetServerFullLinkTimeout
	}
	return rsp, err
}

// mayConvert2NormalTimeout infers whether an error is caused by a server
// timeout. If so, it returns the server timeout error.
func mayConvert2NormalTimeout(
	ctx context.Context,
	req interface{},
	next filter.ServerHandleFunc,
) (interface{}, error) {
	rsp, err := next(ctx, req)
	if e, ok := err.(*errs.Error); ok &&
		e.IsTimeout(errs.ErrorTypeFramework) &&
		e.Code != errs.RetClientTimeout {
		e.Code = errs.RetServerTimeout
	}
	return rsp, err
}
