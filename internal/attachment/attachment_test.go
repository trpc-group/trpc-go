package attachment_test

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/attachment"
	"trpc.group/trpc-go/trpc-go/server"
)

func TestGetClientRequestAttachment(t *testing.T) {
	t.Run("nil message", func(t *testing.T) {
		require.Panics(t, func() {
			attachment.GetClientRequestAttachment(nil)
		})
	})
	t.Run("empty message", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		want := attachment.NoopAttachment{}
		if got := attachment.GetClientRequestAttachment(msg); !reflect.DeepEqual(got, want) {
			t.Errorf("GetClientRequestAttachment() = %v, want %v", got, want)
		}
	})
	t.Run("message contains nil attachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: nil})
		want := attachment.NoopAttachment{}
		if got := attachment.GetClientRequestAttachment(msg); !reflect.DeepEqual(got, want) {
			t.Errorf("GetClientRequestAttachment() = %v, want %v", got, want)
		}
	})
	t.Run("message contains non-empty Request attachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		want := bytes.NewReader([]byte("attachment"))
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: &attachment.Attachment{Request: want}})
		if got := attachment.GetClientRequestAttachment(msg); !reflect.DeepEqual(got, want) {
			t.Errorf("GetClientRequestAttachment() = %v, want %v", got, want)
		}
	})
}

func TestGetServerResponseAttachment(t *testing.T) {
	t.Run("nil message", func(t *testing.T) {
		require.Panics(t, func() {
			attachment.GetServerResponseAttachment(nil)
		})
	})
	t.Run("empty message", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		want := attachment.NoopAttachment{}
		if got := attachment.GetServerResponseAttachment(msg); !reflect.DeepEqual(got, want) {
			t.Errorf("GetServerResponseAttachment() = %v, want %v", got, want)
		}
	})
	t.Run("message contains nil attachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: nil})
		want := attachment.NoopAttachment{}
		if got := attachment.GetServerResponseAttachment(msg); !reflect.DeepEqual(got, want) {
			t.Errorf("GetServerResponseAttachment() = %v, want %v", got, want)
		}
	})
	t.Run("message contains non-empty response attachment", func(t *testing.T) {
		msg := trpc.Message(context.Background())
		want := bytes.NewReader([]byte("attachment"))
		msg.WithCommonMeta(codec.CommonMeta{attachment.ServerAttachmentKey{}: &attachment.Attachment{Response: want}})
		if got := attachment.GetServerResponseAttachment(msg); !reflect.DeepEqual(got, want) {
			t.Errorf("GetServerResponseAttachment() = %v, want %v", got, want)
		}
	})
}

func TestSetClientResponseAttachment(t *testing.T) {
	msg := trpc.Message(context.Background())
	var a attachment.Attachment
	msg.WithCommonMeta(codec.CommonMeta{attachment.ClientAttachmentKey{}: &a})
	attachment.SetClientResponseAttachment(msg, []byte("attachment"))
	bts, err := io.ReadAll(a.Response)

	require.Nil(t, err)
	require.Equal(t, []byte("attachment"), bts)
}

func TestSetServerAttachment(t *testing.T) {
	msg := trpc.Message(context.Background())
	attachment.SetServerRequestAttachment(msg, []byte("attachment"))
	bts, err := io.ReadAll(server.GetAttachment(msg).Request())

	require.Nil(t, err)
	require.Equal(t, []byte("attachment"), bts)
}
