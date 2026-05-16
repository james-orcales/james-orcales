package xdebug

import (
	"encoding/json"
	"fmt"
	"io"
	"path"
	"runtime"
	"strings"
)

const StackTraceDepth = 10

// FprintStackTrace writes a formatted stack trace to the given io.Writer.
//
// It collects up to StackTraceDepth caller program counters starting at `callerLocation`
// (relative to the call site).
//
// Output:
//
//	~/code/golang_snacks git:(main)
//	$ go test ./invariant/examples/02_math/
//	invariant.Always          | /Users/my_username/code/golang_snacks/invariant/assertions_enabled.go:397
//	02_math.Multiply          | /Users/my_username/code/golang_snacks/invariant/examples/02_math/math.go:125
//	02_math_test.TestMultiply | /Users/my_username/code/golang_snacks/invariant/examples/02_math/math_test.go:59
//	testing.tRunner           | /Users/my_username/.local/share/big_bang/share/go/src/testing/testing.go:1934
//	🚨 Assertion Failure 🚨: Fault Injected
//
//	FAIL    github.com/james-orcales/james-orcales/golang_snacks/invariant/examples/02_math        0.330s
//	FAIL
func FprintStackTrace(w io.Writer, callerLocation int) {
	var pcs [StackTraceDepth]uintptr
	skip := 2 + max(0, callerLocation)

	n := runtime.Callers(skip, pcs[:])
	fs := runtime.CallersFrames(pcs[:n])

	var frames [StackTraceDepth]runtime.Frame
	i := 0
	for {
		frame, ok := fs.Next()
		if !ok || i >= len(frames) {
			break
		}
		frame.Function = path.Base(frame.Function)
		frames[i] = frame
		i++
	}

	maxFn := 0
	for j := 0; j < i; j++ {
		n := len(frames[j].Function)
		if n > maxFn {
			maxFn = n
		}
	}

	for j := 0; j < i; j++ {
		frame := frames[j]
		if frame.File == "_testmain.go" {
			continue
		}
		fmt.Fprintf(w,
			"%-*s | %s:%d\n",
			maxFn,
			frame.Function,
			frame.File,
			frame.Line,
		)
	}
}

// PrintJSON is useful for pretty printing structs instead of the "%#v" format specifier which
// writes everything in one line.
func PrintJSON(obj interface{}) {
	text, _ := json.MarshalIndent(obj, "", "\t")
	fmt.Println(string(text))
}

func IntComma(number int) {
	s := fmt.Sprintf("%d", number)
	n := len(s)
	if n <= 3 {
		fmt.Println(number)
		return
	}
	var b strings.Builder
	pre := n % 3
	if pre > 0 {
		b.WriteString(s[:pre])
		if n > 3 {
			b.WriteString(",")
		}
	}
	for i := pre; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteString(",")
		}
	}
	fmt.Println(b.String())
}
