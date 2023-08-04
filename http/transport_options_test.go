package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOptServerTransport(t *testing.T) {
	st := NewServerTransport(
		func() *http.Server { return &http.Server{} },
		WithReusePort(),
		WithEnableH2C())
	require.True(t, st.reusePort)
	require.True(t, st.enableH2C)
}
