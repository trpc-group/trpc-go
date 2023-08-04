package test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/errs"

	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func Test_newPayload(t *testing.T) {
	var invalidLength int32 = -1
	_, err := newPayload(testpb.PayloadType_COMPRESSIBLE, invalidLength)
	require.EqualError(t, err, fmt.Sprintf("requested a response with invalid length %d", invalidLength))

	_, err = newPayload(testpb.PayloadType_UNCOMPRESSABLE, int32(1))
	require.EqualError(t, err, "PayloadType UNCOMPRESSABLE is not supported")

	_, err = newPayload(testpb.PayloadType_RANDOM, int32(1))
	require.EqualValues(t, retUnsupportedPayload, errs.Code(err))
	require.Contains(t, err.Error(), fmt.Sprintf("unsupported payload type: %d", testpb.PayloadType_RANDOM))

	_, err = newPayload(testpb.PayloadType_COMPRESSIBLE, int32(1))
	require.Nil(t, err)
}
