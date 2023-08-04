package restful_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/restful"
)

func TestPattern(t *testing.T) {
	for _, test := range []struct {
		input   string
		wantErr bool
		desc    string
	}{
		{
			input:   "/",
			wantErr: true,
			desc:    "test blank url path",
		},
		{
			input:   "!@#$%^&",
			wantErr: true,
			desc:    "test invalid url path",
		},
		{
			input:   "/foobar/foo/{bar}",
			wantErr: false,
			desc:    "test valid url path",
		},
	} {
		_, err := restful.Parse(test.input)
		require.Equal(t, test.wantErr, err != nil, test.desc)
		if test.wantErr {
			require.Panics(t, func() { restful.Enforce(test.input) }, test.desc)
		} else {
			require.NotPanics(t, func() { restful.Enforce(test.input) }, test.desc)
		}
	}
}
