// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package errs

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var initpc = caller()

type X struct{}

// val returns a Frame pointing to itself.
func (x X) val() frame {
	return caller()
}

// ptr returns a Frame pointing to itself.
func (x *X) ptr() frame {
	return caller()
}

func TestFrameFormat(t *testing.T) {
	var tests = []struct {
		frame
		format string
		want   string
	}{{
		initpc,
		"%s",
		"stack_test.go",
	}, {
		initpc,
		"%+s",
		"trpc.group/trpc-go/trpc-go/errs.init\n" +
			"\t.+errs/stack_test.go",
	}, {
		0,
		"%s",
		"unknown",
	}, {
		0,
		"%+s",
		"unknown",
	}, {
		initpc,
		"%d",
		"16",
	}, {
		0,
		"%d",
		"0",
	}, {
		initpc,
		"%n",
		"init",
	}, {
		func() frame {
			var x X
			return x.ptr()
		}(),
		"%n",
		`\(\*X\).ptr`,
	}, {
		func() frame {
			var x X
			return x.val()
		}(),
		"%n",
		"X.val",
	}, {
		0,
		"%n",
		"",
	}, {
		initpc,
		"%v",
		"stack_test.go:16",
	}, {
		initpc,
		"%+v",
		"trpc.group/trpc-go/trpc-go/errs.init\n" +
			"\t.+errs/stack_test.go",
	}, {
		0,
		"%v",
		"unknown:0",
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.frame, tt.format, tt.want)
	}
}

func TestFuncname(t *testing.T) {
	tests := []struct {
		name, want string
	}{
		{"", ""},
		{"runtime.main", "main"},
		{"github.com/pkg/errors.funcname", "funcname"},
		{"funcname", "funcname"},
		{"io.copyBuffer", "copyBuffer"},
		{"main.(*R).Write", "(*R).Write"},
	}

	for _, tt := range tests {
		got := funcName(tt.name)
		want := tt.want
		if got != want {
			t.Errorf("funcname(%q): want: %q, got %q", tt.name, want, got)
		}
	}
}

func getStackTrace() stackTrace {
	const depth = 8
	var pcs [depth]uintptr
	n := runtime.Callers(1, pcs[:])
	stack := pcs[0:n]
	// convert to errors.StackTrace
	st := make([]frame, len(stack))
	for i := 0; i < len(st); i++ {
		st[i] = frame((stack)[i])
	}
	return st
}

func TestStackTraceFormat(t *testing.T) {
	tests := []struct {
		stackTrace
		format string
		want   string
	}{{
		nil,
		"%s",
		`\[\]`,
	}, {
		nil,
		"%v",
		`\[\]`,
	}, {
		nil,
		"%+v",
		"",
	}, {
		nil,
		"%#v",
		`\[\]errs\.frame\(nil\)`,
	}, {
		make(stackTrace, 0),
		"%s",
		`\[\]`,
	}, {
		make(stackTrace, 0),
		"%v",
		`\[\]`,
	}, {
		make(stackTrace, 0),
		"%+v",
		"",
	}, {
		make(stackTrace, 0),
		"%#v",
		`\[\]errs\.frame{}`,
	}, {
		getStackTrace()[:2],
		"%s",
		`\[stack_test.go stack_test.go\]`,
	}, {
		getStackTrace()[:2],
		"%v",
		`\[stack_test.go:126 stack_test.go:178\]`,
	}, {
		getStackTrace()[:2],
		"%+v",
		"\n" +
			"trpc.group/trpc-go/trpc-go/errs.getStackTrace\n" +
			"\t.+errs/stack_test.go:126\n" +
			"trpc.group/trpc-go/trpc-go/errs.TestStackTraceFormat\n" +
			"\t.+errs/stack_test.go:182",
	}, {
		getStackTrace()[:2],
		"%#v",
		`\[\]errs\.frame{stack_test.go:126, stack_test.go:190}`,
	}}

	for i, tt := range tests {
		testFormatRegexp(t, i, tt.stackTrace, tt.format, tt.want)
	}
}

// a version of runtime.Caller that returns a Frame, not a uintptr.
func caller() frame {
	var pcs [3]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	nextFrame, _ := frames.Next()
	return frame(nextFrame.PC)
}

func testFormatRegexp(t *testing.T, n int, arg interface{}, format, want string) {
	t.Helper()
	got := fmt.Sprintf(format, arg)
	gotLines := strings.SplitN(got, "\n", -1)
	wantLines := strings.SplitN(want, "\n", -1)

	if len(wantLines) > len(gotLines) {
		t.Errorf("test %d: wantLines(%d) > gotLines(%d):\n got: %q\nwant: %q", n+1, len(wantLines), len(gotLines), got, want)
		return
	}

	for i, w := range wantLines {
		match, err := regexp.MatchString(w, gotLines[i])
		if err != nil {
			t.Fatal(err)
		}
		if !match {
			t.Errorf("test %d: line %d: fmt.Sprintf(%q, err):\n got: %q\nwant: %q", n+1, i+1, format, got, want)
		}
	}
}
