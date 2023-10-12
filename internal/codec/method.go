package codec

import "strings"

// MethodFromRPCName returns the method parsed from rpc string.
func MethodFromRPCName(s string) string {
	return s[strings.LastIndex(s, "/")+1:]
}
