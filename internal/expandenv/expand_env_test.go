package expandenv_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	. "trpc.group/trpc-go/trpc-go/internal/expandenv"
)

func TestExpandEnv(t *testing.T) {
	key := "env_key"
	t.Run("no env", func(t *testing.T) {
		require.Equal(t, []byte("abc"), ExpandEnv([]byte("abc")))
	})
	t.Run("${..} is expanded", func(t *testing.T) {
		t.Setenv(key, t.Name())
		require.Equal(t, fmt.Sprintf("head_%s_tail", t.Name()),
			string(ExpandEnv([]byte(fmt.Sprintf("head_${%s}_tail", key)))))
	})
	t.Run("${ is not expanded", func(t *testing.T) {
		require.Equal(t, "head_${_tail",
			string(ExpandEnv([]byte(fmt.Sprintf("head_${_tail")))))
	})
	t.Run("${} is expanded as empty", func(t *testing.T) {
		require.Equal(t, "head__tail",
			string(ExpandEnv([]byte("head_${}_tail"))))
	})
	t.Run("${..} is not expanded if .. contains any space", func(t *testing.T) {
		t.Setenv("key key", t.Name())
		require.Equal(t, "head_${key key}_tail",
			string(ExpandEnv([]byte("head_${key key}_tail"))))
	})
	t.Run("${..} is not expanded if .. contains any new line", func(t *testing.T) {
		t.Setenv("key\nkey", t.Name())
		require.Equal(t, t.Name(), os.Getenv("key\nkey"))
		require.Equal(t, "head_${key\nkey}_tail",
			string(ExpandEnv([]byte("head_${key\nkey}_tail"))))
	})
	t.Run(`${..} is not expanded if .. contains any "`, func(t *testing.T) {
		t.Setenv(`key"key`, t.Name())
		require.Equal(t, t.Name(), os.Getenv(`key"key`))
		require.Equal(t, `head_${key"key}_tail`,
			string(ExpandEnv([]byte(`head_${key"key}_tail`))))
	})
}
