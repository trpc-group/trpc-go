// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

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
