package codec

import "strings"

// MethodFromRPCName returns the method parsed from rpc string.
// Reference:
// https://git.woa.com/trpc/trpc-proposal/blob/master/A15-metrics-rules.md
func MethodFromRPCName(s string) string {
	return s[strings.LastIndex(s, "/")+1:]
}
