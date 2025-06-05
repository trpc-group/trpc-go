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
	Request      io.Reader
	RequestSize  int
	Response     io.Reader
	ResponseSize int
}

// NoopAttachment is an empty attachment.
type NoopAttachment struct{}

// Read implements the io.Reader interface, which always returns (0, io.EOF)
func (a NoopAttachment) Read(_ []byte) (n int, err error) {
	return 0, io.EOF
}

// SizedAttachment is an attachment with size.
type SizedAttachment struct {
	r         io.Reader
	bts       []byte
	size      int64
	ioEnabled bool
}

// ReadAll read all data from SizedAttachment.
// The length of bts is at least Size.
func (a *SizedAttachment) ReadAll(bts []byte) error {
	if a.ioEnabled {
		_, err := io.ReadAtLeast(a.r, bts, int(a.size))
		return err
	}
	copy(bts, a.bts[:a.size])
	return nil
}

// Size returns the size of SizedAttachment.
func (a *SizedAttachment) Size() int64 {
	return a.size
}

// Sizer is the interface that wraps the basic Read method.
// Attachment implements Sizer can reduce memory copy.
type Sizer interface {
	Size() int64
}

// ClientRequestSizedAttachment returns client's Request Attachment with size from msg.
func ClientRequestSizedAttachment(msg codec.Msg) (*SizedAttachment, error) {
	if a, _ := msg.CommonMeta()[ClientAttachmentKey{}].(*Attachment); a != nil {
		if s, ok := a.Request.(Sizer); ok {
			return &SizedAttachment{r: a.Request, ioEnabled: true, size: s.Size()}, nil
		}
		bts, err := io.ReadAll(a.Request)
		if err != nil {
			return nil, err
		}
		return &SizedAttachment{bts: bts, size: int64(len(bts))}, nil
	}
	return &SizedAttachment{}, nil
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

// ServerResponseSizedAttachment returns server's Response Attachment from msg.
func ServerResponseSizedAttachment(msg codec.Msg) (*SizedAttachment, error) {
	if a, _ := msg.CommonMeta()[ServerAttachmentKey{}].(*Attachment); a != nil {
		if s, ok := a.Response.(Sizer); ok {
			return &SizedAttachment{r: a.Response, ioEnabled: true, size: s.Size()}, nil
		}

		bts, err := io.ReadAll(a.Response)
		if err != nil {
			return nil, err
		}
		return &SizedAttachment{bts: bts, size: int64(len(bts))}, nil
	}
	return &SizedAttachment{}, nil
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
