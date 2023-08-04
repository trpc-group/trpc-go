package client

import (
	"io"

	"trpc.group/trpc-go/trpc-go/internal/attachment"
)

// Attachment stores the Attachment of tRPC requests/responses.
type Attachment struct {
	attachment attachment.Attachment
}

// NewAttachment returns a new Attachment whose response Attachment is a NoopAttachment.
func NewAttachment(request io.Reader) *Attachment {
	return &Attachment{attachment: attachment.Attachment{Request: request, Response: attachment.NoopAttachment{}}}
}

// Response returns Response Attachment.
func (a *Attachment) Response() io.Reader {
	return a.attachment.Response
}
