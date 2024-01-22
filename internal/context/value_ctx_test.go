package context_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	icontext "trpc.group/trpc-go/trpc-go/internal/context"
)

func TestWithValues(t *testing.T) {
	type testKey struct{}
	testValue := "value"
	ctx := context.WithValue(context.TODO(), testKey{}, testValue)
	ctx1 := icontext.NewContextWithValues(context.TODO(), ctx)
	require.NotNil(t, ctx1.Value(testKey{}))
	type notExist struct{}
	require.Nil(t, ctx1.Value(notExist{}))
}
