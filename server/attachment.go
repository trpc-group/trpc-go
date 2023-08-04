package server

import (
	"io"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
)

// Attachment stores the attachment of tRPC requests/responses.
type Attachment struct {
	attachment *attachment.Attachment
}

// Request returns Request Attachment.
func (a *Attachment) Request() io.Reader {
	return a.attachment.Request
}

// SetResponse sets Response attachment.
func (a *Attachment) SetResponse(attachment io.Reader) {
	a.attachment.Response = attachment
}

// GetAttachment returns Attachment from msg.
// If there is no Attachment in the msg, an empty attachment bound to the msg will be returned.
func GetAttachment(msg codec.Msg) *Attachment {
	cm := msg.CommonMeta()
	if cm == nil {
		cm = make(codec.CommonMeta)
		msg.WithCommonMeta(cm)
	}
	a, _ := cm[attachment.ServerAttachmentKey{}]
	if a == nil {
		a = &attachment.Attachment{Request: attachment.NoopAttachment{}, Response: attachment.NoopAttachment{}}
		cm[attachment.ServerAttachmentKey{}] = a
	}

	return &Attachment{attachment: a.(*attachment.Attachment)}
}
