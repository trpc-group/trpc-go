// Package attachment provides a internal implementation of tRPC client/server attachment.
package attachment

import (
	"bytes"
	"io"

	"trpc.group/trpc-go/trpc-go/codec"
)

// ClientAttachmentKey is the key of client's Attachment.
type ClientAttachmentKey struct{}

// ServerAttachmentKey is the key of server's Attachment.
type ServerAttachmentKey struct{}

// Attachment stores the attachment in tRPC requests/responses.
type Attachment struct {
	Request  io.Reader
	Response io.Reader
}

// NoopAttachment is a empty attachment.
type NoopAttachment struct{}

// Read implements the io.Reader interface, which always returns (0, io.EOF)
func (a NoopAttachment) Read(_ []byte) (n int, err error) {
	return 0, io.EOF
}

// GetClientRequestAttachment returns client's Request Attachment from msg.
func GetClientRequestAttachment(msg codec.Msg) io.Reader {
	if a, _ := msg.CommonMeta()[ClientAttachmentKey{}].(*Attachment); a != nil {
		return a.Request
	}
	return NoopAttachment{}
}

// GetServerResponseAttachment returns server's Response Attachment from msg.
func GetServerResponseAttachment(msg codec.Msg) io.Reader {
	if a, _ := msg.CommonMeta()[ServerAttachmentKey{}].(*Attachment); a != nil {
		return a.Response
	}
	return NoopAttachment{}
}

// SetClientResponseAttachment sets client's Response attachment to msg.
// If the message does not contain client.Attachment,
// which means that the user has explicitly ignored the att returned by the server.
// For performance reasons, there is no need to set the response attachment into msg.
func SetClientResponseAttachment(msg codec.Msg, attachment []byte) {
	if a, _ := msg.CommonMeta()[ClientAttachmentKey{}].(*Attachment); a != nil {
		a.Response = bytes.NewReader(attachment)
	}
}

// SetServerRequestAttachment sets server's Request Attachment to msg.
func SetServerRequestAttachment(m codec.Msg, attachment []byte) {
	cm := m.CommonMeta()
	if cm == nil {
		cm = make(codec.CommonMeta)
		m.WithCommonMeta(cm)
	}
	cm[ServerAttachmentKey{}] = &Attachment{Request: bytes.NewReader(attachment), Response: NoopAttachment{}}
}
