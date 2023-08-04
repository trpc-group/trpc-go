// Package errs provides trpc error code type, which contains errcode errmsg.
// These definitions are multi-language universal.
package errs

import (
	"fmt"
	"io"
	"path"
	"runtime"
	"strconv"
	"strings"
)

var (
	traceable bool               // if traceable is true, the error has a stack trace .
	content   string             // if content is not empty, only print stack information contains it.
	stackSkip = defaultStackSkip // number of stack frames skipped.
)

const (
	defaultStackSkip = 3
)

// SetTraceable controls whether the error has a stack trace.
func SetTraceable(x bool) {
	traceable = x
}

// SetTraceableWithContent controls whether the error has a stack trace.
// When printing the stack information, filter according to the content.
// Avoid outputting a lot of useless information. The stack information
// of other plugins can be filtered out by configuring content as the service name.
func SetTraceableWithContent(c string) {
	traceable = true
	content = c
}

// SetStackSkip supports setting the number of skipped stack frames.
// When encapsulating the New method, you can set stackSkip to 4 (determined
// according to the number of encapsulation layers)
// This function is used to set before the project starts and does not guarantee concurrency safety.
func SetStackSkip(skip int) {
	stackSkip = skip
}

func isOutput(str string) bool {
	return strings.Contains(str, content)
}

// frame represents a program counter inside a stack frame.
// For historical reasons if frame is interpreted as a uintptr
// its value represents the program counter + 1.
type frame uintptr

// pc returns the program counter for this frame;
// multiple frames may have the same PC value.
func (f frame) pc() uintptr { return uintptr(f) - 1 }

// file returns the full path to the file that contains the
// function for this frame's pc.
func (f frame) file() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	file, _ := fn.FileLine(f.pc())
	return file
}

// line returns the line number of source code of the
// function for this frame's pc.
func (f frame) line() int {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return 0
	}
	_, line := fn.FileLine(f.pc())
	return line
}

// name returns the name of this function, if known.
func (f frame) name() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown"
	}
	return fn.Name()
}

// Format formats the frame according to the fmt.Formatter interface.
//
//	%s    source file
//	%d    source line
//	%n    function name
//	%v    equivalent to %s:%d
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+s   function name and path of source file relative to the compile time
//	      GOPATH separated by \n\t (<funcName>\n\t<path>)
//	%+v   equivalent to %+s:%d
func (f frame) Format(s fmt.State, verb rune) {
	switch verb {
	case 's':
		switch {
		case s.Flag('+'):
			io.WriteString(s, f.name())
			io.WriteString(s, "\n\t")
			io.WriteString(s, f.file())
		default:
			io.WriteString(s, path.Base(f.file()))
		}
	case 'd':
		io.WriteString(s, strconv.Itoa(f.line()))
	case 'n':
		io.WriteString(s, funcName(f.name()))
	case 'v':
		f.Format(s, 's')
		io.WriteString(s, ":")
		f.Format(s, 'd')
	}
}

// stackTrace is stack of Frames from innermost (newest) to outermost (oldest).
type stackTrace []frame

// Format formats the stack of Frames according to the fmt.Formatter interface.
//
//	%s	lists source files for each frame in the stack
//	%v	lists the source file and line number for each frame in the stack
//
// Format accepts flags that alter the printing of some verbs, as follows:
//
//	%+v   Prints filename, function, and line number for each frame in the stack.
func (st stackTrace) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			for _, f := range st {
				// filter, only print stack information contains the content.
				if !isOutput(fmt.Sprintf("%+v", f)) {
					continue
				}
				io.WriteString(s, "\n")
				f.Format(s, verb)
			}
		case s.Flag('#'):
			fmt.Fprintf(s, "%#v", []frame(st))
		default:
			st.formatSlice(s, verb)
		}
	case 's':
		st.formatSlice(s, verb)
	}
}

// formatSlice will format this stackTrace into the given buffer as a slice of
// frame, only valid when called with '%s' or '%v'.
func (st stackTrace) formatSlice(s fmt.State, verb rune) {
	io.WriteString(s, "[")
	for i, f := range st {
		if i > 0 {
			io.WriteString(s, " ")
		}
		f.Format(s, verb)
	}
	io.WriteString(s, "]")
}

func callers() stackTrace {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(stackSkip, pcs[:])
	stack := pcs[0:n]
	// convert to errors.stackTrace
	st := make([]frame, len(stack))
	for i := 0; i < len(st); i++ {
		st[i] = frame((stack)[i])
	}
	return st
}

// funcName removes the path prefix component of a function's name reported by func.Name().
func funcName(name string) string {
	i := strings.LastIndex(name, "/")
	name = name[i+1:]
	i = strings.Index(name, ".")
	return name[i+1:]
}
