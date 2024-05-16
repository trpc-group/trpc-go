package codec

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkRPCNameIsTRPCForm(b *testing.B) {
	rpcNames := []string{
		"/trpc.app.server.service/method",
		"/sdadfasd/xadfasdf/zxcasd/asdfasd/v2",
		"trpc.app.server.service",
		"/trpc.app.server.service",
		"/trpc.app.",
		"/trpc/asdf/asdf",
		"/trpc.asdfasdf/asdfasdf/sdfasdfa/",
		"/trpc.app/method/",
		"/trpc.app/method/hhhhh",
	}
	b.Run("bench regexp", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for j := range rpcNames {
				rpcNameIsTRPCFormRegExp(rpcNames[j])
			}
		}
	})
	b.Run("bench vanilla", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for j := range rpcNames {
				rpcNameIsTRPCForm(rpcNames[j])
			}
		}
	})
}

func TestEnsureEqualSemacticOfTRPCFormChecking(t *testing.T) {
	rpcNames := []string{
		"/trpc.app.server.service/method",
		"/trpc.app.server.service/",
		"/trpc",
		"//",
		"/./",
		"/xx/.",
		"/x./method",
		"/.x/method",
		"/sdadfasd/xadfasdf/zxcasd/asdfasd/v2",
		"trpc.app.server.service",
		"/trpc.app.server.service",
		"/trpc.app.",
		"/trpc/asdf/asdf",
		"/trpc.asdfasdf/asdfasdf/sdfasdfa/",
		"/trpc.app/method/",
		"/trpc.app/method/hhhhh",
	}
	for _, s := range rpcNames {
		v1, v2 := rpcNameIsTRPCFormRegExp(s), rpcNameIsTRPCForm(s)
		require.True(t, v1 == v2, "%s %v %v", s, v1, v2)
	}
}

var r = regexp.MustCompile(`^/[^/.]+\.[^/]+/[^/.]+$`)

func rpcNameIsTRPCFormRegExp(s string) bool {
	return r.MatchString(s)
}
