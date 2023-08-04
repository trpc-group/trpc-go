// Package codec provides some common codec-related functions.
package codec

import "trpc.group/trpc-go/trpc-go/codec"

// IsValidCompressType checks whether t is a valid Compress type.
func IsValidCompressType(t int) bool {
	const minValidCompressType = codec.CompressTypeNoop
	return t >= minValidCompressType
}
