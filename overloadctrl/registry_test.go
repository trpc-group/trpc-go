package overloadctrl_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/overloadctrl"
)

func TestRegister(t *testing.T) {
	require.Nil(t, overloadctrl.GetClient("not_exist"))
	require.Nil(t, overloadctrl.GetServer("not_exist"))
	overloadctrl.RegisterClient("test_noop",
		func(info *overloadctrl.ServiceMethodInfo) overloadctrl.OverloadController {
			return overloadctrl.NoopOC{}
		})
	require.NotNil(t, overloadctrl.GetClient("test_noop"))
	overloadctrl.RegisterServer("test_noop",
		func(info *overloadctrl.ServiceMethodInfo) overloadctrl.OverloadController {
			return overloadctrl.NoopOC{}
		})
	require.NotNil(t, overloadctrl.GetServer("test_noop"))
}
