package tls_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"trpc.group/trpc-go/trpc-go/internal/tls"
)

func TestGetServerConfig(t *testing.T) {
	_, err := tls.GetServerConfig("../../testdata/ca.pem", "../../testdata/server.crt", "../../testdata/server.key")
	assert.Nil(t, err)
	_, err = tls.GetServerConfig("", "../../testdata/server.crt", "../../testdata/server.key")
	assert.Nil(t, err)
	_, err = tls.GetServerConfig("", "", "")
	assert.NotNil(t, err)
}

func TestGetClientConfig(t *testing.T) {
	_, err := tls.GetClientConfig("localhost", "../../testdata/ca.pem", "../../testdata/client.crt", "../../testdata/client.key")
	assert.Nil(t, err)
	_, err = tls.GetClientConfig("localhost", "none", "../../testdata/client.crt", "../../testdata/client.key")
	assert.Nil(t, err)
	_, err = tls.GetClientConfig("localhost", "../../testdata/ca.pem", "", "")
	assert.Nil(t, err)
	_, err = tls.GetClientConfig("localhost", "root", "", "")
	assert.Nil(t, err)
}
