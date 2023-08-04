package server_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
	"trpc.group/trpc-go/trpc-go/server"
)

func TestAttachment(t *testing.T) {
	msg := trpc.Message(context.Background())
	attm := server.GetAttachment(msg)
	require.Equal(t, attachment.NoopAttachment{}, attm.Request())

	attm.SetResponse(bytes.NewReader([]byte("attachment")))
	responseAttm := attachment.GetServerResponseAttachment(msg)
	bts, err := io.ReadAll(responseAttm)
	require.Nil(t, err)
	require.Equal(t, []byte("attachment"), bts)
}
