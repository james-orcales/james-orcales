/*
Package invariant provides a property testing framework for Go, designed to
encode and enforce assumptions (invariants) directly within your code.

# Assertion Types

   - **Always** — The property must hold true for *all* inputs or executions.
     To disprove an Always assertion, you only need *one counterexample* where
     the condition is false.

   - **Sometimes** — The property is expected to be *occasionally true* across
     runs. To prove a Sometimes assertion, you need *one example* where the
     condition is true. If it never evaluates to true, the property is disproven.
     Observing its absence requires a program context where all expected states
     to be consistently exercised—which is only feasible in testing environments.

# Implementation

In production, assertions panic/crash on violation.
In test environments, invariant activates a global tracker that records every
assertion that evaluated to true.

Before tests run, you register the packages to analyze (usually just the current
test’s target package). The parser locates all assertions within said package
then pre-registers them in the tracker. As tests execute, successful assertions
(true evaluations) are recorded in a package-global tracker keyed by file and line
number. It’s crucial that the package qualifier name remains `invariant`, since
the parser relies on this identifier to detect assertions.

Next, the analyzer cross-references parsed assertions with those observed at
runtime. Any assertion that never evaluated to true (frequency = 0) is reported
as a “missed invariant,” causing the test suite to fail. If all assertions were
exercised, the analyzer prints a summary showing the *least exercised* assertions.

Invariant therefore provides actionable, frequency-based insight into how
thoroughly your properties have been exercised, revealing the true scope and
effectiveness of your testing suite. By tracking which invariants fire and
which remain untested, it exposes gaps that ordinary unit tests alone would
never highlight, giving a far more comprehensive view of software reliability
than coverage metrics or ad-hoc tests can offer. Additionally, by making the
assumptions of your software explicit, it improves developer experience and
accelerates onboarding, helping new contributors understand the intended
behavior and constraints of the system.
*/

package invariant

import (
	"fmt"
	"io"
	"iter"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const (
	// Used to detect panics caused by assertion failures
	//
	//	defer func() {
	//		if err := recover(); err != nil {
	//			if strErr, ok := err.(string); ok && strings.HasPrefix(strErr, invariant.AssertionFailureMsgPrefix) {
	//				// handle assertion failure
	//			}
	//		}
	//	}()
	AssertionFailureMsgPrefix = "🚨 Assertion Failure 🚨"
	// This library minimizes heap allocations by preferring fixed-size backing arrays for
	// slices. These values determine the size of those arrays and should accommodate typical
	// codebases without frequent resizing.
	leastExercisedInvariantCount = 10
	// maxAssertionsPerPackage is the maximum number of Sometimes, XSometimes, Always*,
	// XAlways*, and Ensure calls in a single package.
	maxAssertionsPerPackage = 2048
	maxGoFilesPerPackage    = 1024
	maxFilePath             = 260
	maxFileLines            = 5 // In digits (99,999 lines)
	assertionIDLength       = maxFilePath + 1 + maxFileLines
)

var (
	INVARIANT_FULL_LOCATION    = os.Getenv("INVARIANT_FULL_LOCATION") == "1"
	INVARIANT_STACKTRACE_DEPTH = func() int {
		v := os.Getenv("INVARIANT_STACKTRACE_DEPTH")
		result, err := strconv.Atoi(v)
		if err != nil {
			result = 10
		}
		return result
	}()

	// msg is already prefixed with AssertionFailurePrefix here. If the user msg is empty then
	// it also contains emptyMessageIndicator.
	AssertionFailureHook    = func(msg string) {}
	AssertionFailureIsFatal = false

	IsRunningUnderGoTest = func() bool {
		v := false
		for _, arg := range os.Args {
			if strings.HasPrefix(arg, "-test.") {
				v = true
				break
			}
		}
		return v
	}()

	IsRunningUnderGoFuzz = func() bool {
		v := false
		for _, arg := range os.Args {
			if strings.HasPrefix(arg, "-test.fuzz") {
				v = true
				break
			}
		}
		return v
	}()

	IsRunningUnderGoBenchmark = func() bool {
		v := false
		for _, arg := range os.Args {
			if strings.HasPrefix(arg, "-test.bench") {
				v = true
				break
			}
		}
		return v
	}()

	// packagesToAnalyze defaults to the current testing package.
	packagesToAnalyze = []string{"."}
	// assertionTracker globally tracks true assertions inside packagesToAnalyze.
	assertionTracker        = make(map[string]*metadata, maxAssertionsPerPackage*len(packagesToAnalyze))
	assertionFrequencyMutex = sync.Mutex{}
)

type metadata struct {
	Frequency int
	Message   string
	Kind      string
}

// WARN: Callers rely on this callback to implicitly terminate control flow on failure (via
// panic or os.Exit).
func assertionFailureCallback(msg string) {
	AssertionFailureHook(msg)
	if AssertionFailureIsFatal {
		FprintStackTrace(os.Stderr, 1)
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	} else {
		panic(msg)
	}
}

// Ensure is the same as invariant.Always but it CANNOT be disabled.
//
//go:noinline
func Ensure(cond bool, msg string) {
	if cond {
		registerAssertion()
	} else {
		assertionFailureCallback(msg)
	}
}

// GameLoop provides an explicit infinite loop sequence, similar to `for {}`. This makes infinite
// loops explicit, which is useful if `for {}` is banned in CI/CD or code reviews.
//
// Usage:
//
//	for range invariant.GameLoop() {
//		rl.Render()
//	}
//
// Each iteration yields an increasing integer starting from 0.
func GameLoop() iter.Seq[int] {
	return func(yield func(int) bool) {
		for iteration := 0; true; iteration++ {
			if !yield(iteration) {
				return
			}
		}
	}
}

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
	pcs := make([]uintptr, INVARIANT_STACKTRACE_DEPTH)
	skip := 2 + max(0, callerLocation)

	n := runtime.Callers(skip, pcs[:])
	fs := runtime.CallersFrames(pcs[:n])

	frames := make([]runtime.Frame, INVARIANT_STACKTRACE_DEPTH)
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
		fmt.Fprintf(w,
			"%-*s | %s:%d\n",
			maxFn,
			frame.Function,
			frame.File,
			frame.Line,
		)
	}
}

// Usage:
//
//	defer func() {
//		r := recover()
//		if invariant.IsAssertionFailure(r) {
//			// ...
//		} else {
//			// ...
//		}
//	}()
func IsAssertionFailure(v any) bool {
	message, ok := v.(string)
	return ok && strings.HasPrefix(message, AssertionFailureMsgPrefix)
}
