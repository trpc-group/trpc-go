package client

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
)

func TestAttachment(t *testing.T) {
	attm := NewAttachment(bytes.NewReader([]byte("attachment")))
	require.Equal(t, attachment.NoopAttachment{}, attm.Response())

	msg := codec.Message(context.Background())
	setAttachment(msg, &attm.attachment)
	attcher := attachment.GetClientRequestAttachment(msg)
	bts, err := io.ReadAll(attcher)
	require.Nil(t, err)
	require.Equal(t, []byte("attachment"), bts)
}
