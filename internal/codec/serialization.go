package codec

import "trpc.group/trpc-go/trpc-go/codec"

// IsValidSerializationType checks whether t is a valid serialization type.
func IsValidSerializationType(t int) bool {
	const minValidSerializationType = codec.SerializationTypePB
	return t >= minValidSerializationType
}
