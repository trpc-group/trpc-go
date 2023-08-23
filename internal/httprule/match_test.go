package httprule_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/internal/httprule"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		toParse      string
		toMatch      string
		wantCaptured map[string]string
		wantErr      bool
		desc         string
	}{
		{
			toParse: "/foobar/{foo}/bar/{baz}",
			toMatch: "/foobar/x/bar/y",
			wantCaptured: map[string]string{
				"foo": "x",
				"baz": "y",
			},
			wantErr: false,
			desc:    "MatchTest01",
		},
		{
			toParse: "/foobar/{foo=x/*}/bar/{baz}",
			toMatch: "/foobar/x/y/bar/z",
			wantCaptured: map[string]string{
				"foo": "x/y",
				"baz": "z",
			},
			wantErr: false,
			desc:    "MatchTest02",
		},
		{
			toParse:      "/foo/bar/**",
			toMatch:      "/foo/bar/x/y/z",
			wantCaptured: map[string]string{},
			wantErr:      false,
			desc:         "MatchTest03",
		},
		{
			toParse:      "/foo/*/bar",
			toMatch:      "/foo/x/bar",
			wantCaptured: map[string]string{},
			wantErr:      false,
			desc:         "MatchTest04",
		},
		{
			toParse: "/a/b/{c}:d:e",
			toMatch: "/a/b/x:y:z:d:e",
			wantCaptured: map[string]string{
				"c": "x:y:z",
			},
			wantErr: false,
			desc:    "MatchTest05",
		},
		{
			toParse:      "/a/b/{c}:d:e",
			toMatch:      "/a/b/anything",
			wantCaptured: nil,
			wantErr:      true,
			desc:         "MatchTest06",
		},
		{
			toParse:      "/a/b/{c}:d:e",
			toMatch:      "/a/b/:d:e",
			wantCaptured: nil,
			wantErr:      true,
			desc:         "MatchTest07",
		},
		{
			toParse:      "/foo/bar/{a=*}",
			toMatch:      "/foo/bar/x/y/z",
			wantCaptured: nil,
			wantErr:      true,
			desc:         "MatchTest08",
		},
		{
			toParse: "/foo/bar/{a=**}",
			toMatch: "/foo/bar/x/y/z",
			wantCaptured: map[string]string{
				"a": "x/y/z",
			},
			wantErr: false,
			desc:    "MatchTest09",
		},
		{
			toParse: "/foo/bar/{a=**}",
			toMatch: "/foo/bar/",
			wantCaptured: map[string]string{
				"a": "",
			},
			wantErr: false,
			desc:    "MatchTest10",
		},
		{
			toParse: "/foo/bar/{a=**}",
			toMatch: "/foo/bar",
			wantCaptured: map[string]string{
				"a": "",
			},
			wantErr: false,
			desc:    "MatchTest11",
		},
	}

	for _, test := range tests {
		tpl, err := httprule.Parse(test.toParse)
		require.Nil(t, err)
		gotCaptured, gotErr := tpl.Match(test.toMatch)
		require.Equal(t, true, reflect.DeepEqual(test.wantCaptured, gotCaptured), test.desc)
		require.Equal(t, test.wantErr, gotErr != nil, test.desc)
	}
}
