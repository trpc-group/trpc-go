package client

import "trpc.group/trpc-go/trpc-go/errs"

// mayConvert2FullLinkTimeout infers whether an error is caused by a full-link
// timeout. If so, it returns the full-link timeout error.
func mayConvert2FullLinkTimeout(err error) error {
	if e, ok := err.(*errs.Error); ok && e.IsTimeout(errs.ErrorTypeFramework) {
		e.Code = errs.RetClientFullLinkTimeout
	}
	return err
}
