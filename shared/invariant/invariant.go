/*
Package invariant provides a property testing framework for Go, designed to encode and enforce
assumptions (invariants) directly within your code.

# Assertion Types

   - **Always** — The property must hold true for *all* inputs or executions. To disprove an Always
   assertion, you only need *one counterexample* where the condition is false.

   - **Sometimes** — The property is expected to be *occasionally true* across runs. To prove a
   Sometimes assertion, you need *one example* where the condition is true. If it never evaluates to
   true, the property is disproven. Observing its absence requires a program context where all
   expected states to be consistently exercised—which is only feasible in testing environments.
*/

package invariant

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"iter"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
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
	// maxAssertionsPerPackage is the maximum number of Sometimes, Always*,
	// and Ensure calls in a single package.
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
	// sync.Map fits the access pattern: writes once per assertion during
	// RegisterPackagesForAnalysis (parallel AST walkers), then read-mostly
	// during the test run. Frequency counters are atomic to allow lock-free
	// concurrent increments across goroutines.
	assertionTracker = sync.Map{}
)

type metadata struct {
	Frequency      atomic.Int64
	FalseFrequency atomic.Int64
	Message        string
	Kind           string
}

// Callers expect this to not panic or be fatal.
var AssertionFailureCallback func(string) = func(msg string) {
	fmt.Fprintf(os.Stderr, "%s\n", msg)
}

// registerAssertion records in the package-global assertion tracker that an
// assertion was evaluated as true at the current source location. It tracks the
// file and line number of the call and increments a counter in
// assertionTracker.
//
// This function is a noop outside of a test environment.
//
// It is concurrency-safe and can be called from multiple goroutines.
//
//go:noinline
func registerAssertion() {
	if !IsRunningUnderGoTest || IsRunningUnderGoBenchmark || IsRunningUnderGoFuzz {
		return
	}
	callers := [1]uintptr{}
	count := runtime.Callers(3, callers[:])
	frame, _ := runtime.CallersFrames(callers[:count]).Next()

	arr := [assertionIDLength]byte{}
	buf := arr[:0]
	buf = append(buf, frame.File...)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, int64(frame.Line), 10)

	id := string(buf)
	if v, ok := assertionTracker.Load(id); ok {
		v.(*metadata).Frequency.Add(1)
	}
}

// registerFalseAssertion records in the package-global assertion tracker that a
// Sometimes assertion was evaluated as false at the current source location.
// This is called from assertionFailureHook to track which assertions have had
// their failure modes exercised.
//
// This function is a noop outside of a test environment.
//
// It is concurrency-safe and can be called from multiple goroutines.
//
//go:noinline
func registerFalseAssertion() {
	if !IsRunningUnderGoTest || IsRunningUnderGoBenchmark || IsRunningUnderGoFuzz {
		return
	}
	callers := [1]uintptr{}
	// Skip 3 frames: registerFalseAssertion -> Sometimes -> caller
	count := runtime.Callers(3, callers[:])
	frame, _ := runtime.CallersFrames(callers[:count]).Next()

	arr := [assertionIDLength]byte{}
	buf := arr[:0]
	buf = append(buf, frame.File...)
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, int64(frame.Line), 10)

	id := string(buf)
	if v, ok := assertionTracker.Load(id); ok {
		v.(*metadata).FalseFrequency.Add(1)
	}
}

func RunTestMain(m *testing.M, dirs ...string) {
	RegisterPackagesForAnalysis(dirs...)
	code := m.Run()
	AnalyzeAssertionFrequency()
	os.Exit(code)
}

// RegisterPackagesForAnalysis ensures that only assertions from the tested directories are tracked
// for frequency analysis. Dirs is relative to the directory of the caller.
//
// NOTE: All assertions must have their last parameter be the message parameter
//
// NOTE: During fuzzing or benchmarking, assertion tracking is disabled because worker processes
// handle assertions in separate memory spaces. Any registrations done there are not synchronized
// back to the parent test runner
//
// NOTE: Tracking assertions during fuzzing can help gauge how thoroughly your fuzzer exercises the
// code. However, since this implementation tracks assertions at the package level—and most fuzz
// tests focus on specific functions—this is generally sufficient. Full assertion tracking is mainly
// useful when simulating the entire program; in that case, using a dedicated script, unit test, or
// custom fuzz harness is more appropriate.
//
// PERF: Create an instrumentation tool that hardcodes assertion source location, drastically
// improving registration performance at runtime. This enables assertion tracking in fuzz testing.
func RegisterPackagesForAnalysis(dirs ...string) {
	Ensure(IsRunningUnderGoTest, "RegisterPackagesForAnalysis is only used in testing environments")
	Ensure(
		len(packagesToAnalyze) == 1 && packagesToAnalyze[0] == ".",
		"packagesToAnalyze was set to the current testing package by default",
	)
	if IsRunningUnderGoBenchmark || IsRunningUnderGoFuzz {
		return
	}
	if len(dirs) > 0 {
		packagesToAnalyze = dirs
	}
	// === Absolute Path Conversion ===
	for i, path := range packagesToAnalyze {
		absPath, err := filepath.Abs(path)
		if err != nil || absPath == "" {
			panic(fmt.Sprintf("Failed to convert package path to absolute path: %s\n", err))
		}
		packagesToAnalyze[i] = absPath
	}

	// ===Collection===
	filesArray := [maxGoFilesPerPackage]string{}
	files := filesArray[:0]
	for _, dir := range packagesToAnalyze {
		before := len(files)
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if d.IsDir() && path != dir {
				return filepath.SkipDir
			}
			if err != nil || filepath.Ext(path) != ".go" {
				return err
			}
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if len(absPath) > len("_test.go") && strings.HasSuffix(absPath, "_test.go") {
				return nil
			}
			files = append(files, absPath)
			return nil
		})
		if err != nil {
			panic(fmt.Sprintf("Collecting all files to find missed invariants: %s\n", err))
		}
		after := len(files)
		Ensure(before < after, "The directory contains go files")
	}
	Ensure(len(files) > 0, "There's at least one file to parse")

	// ===Parsing===
	semaphore := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup

	for _, path := range files {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(filePath string) {
			defer func() { <-semaphore; wg.Done() }()
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
			if err != nil {
				return
			}
			ast.Inspect(node, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				ident, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				if ident.Name != "invariant" {
					return true
				}
				msg := ""
				switch sel.Sel.Name {
				case "Sometimes", "Reachable", "Ensure", "EnsureErrIsNil", "Always", "AlwaysErrIsNil",
					"XSometimes", "XEnsure", "XAlways":
					pos := fset.Position(call.Lparen)
					key := filePath + ":" + strconv.Itoa(pos.Line)
					literal, isLiteral := call.Args[len(call.Args)-1].(*ast.BasicLit)
					if !isLiteral {
						panic("The assertion message is an inlined string literal:" + key)
					}
					Ensure(literal.Kind == token.STRING, "The assertion message is a pure string literal")
					msg = literal.Value[1 : len(literal.Value)-1] // remove quotes

					if (sel.Sel.Name == "Sometimes" || sel.Sel.Name == "XSometimes") && len(call.Args) >= 2 {
						if firstArg, isIdent := call.Args[0].(*ast.Ident); isIdent && firstArg.Name == "true" {
							fmt.Fprintf(os.Stderr,
								"Sometimes(true, ...) at %s — use invariant.Reachable(%s) instead\n",
								key, literal.Value,
							)
							os.Exit(1)
						}
					}

					assertionTracker.Store(key, &metadata{
						Kind:    sel.Sel.Name,
						Message: msg,
					})
				}
				return true
			})
		}(path)
	}

	wg.Wait()
}

// AnalyzeAssertionFrequency scans the given directories for Sometimes, Always*, XAlways* calls that
// have never evaluated to true and returns their source locations and respective messages. To see
// the stats, enable verbosity with `go test ./foo -v`
//
// Usage:
//
//	func TestMain(m *testing.M) {
//		invariant.RunTestMain(m)
//	}
//
// It is critical that you import this package under the name "invariant" as it is hardcoded in the
// analyzer to look for this identifier.
func AnalyzeAssertionFrequency() {
	Ensure(IsRunningUnderGoTest, "AnalyzeAssertionFrequency is only used for testing")
	Ensure(len(packagesToAnalyze) > 0, "At least one package was registered for analysis")
	if IsRunningUnderGoBenchmark || IsRunningUnderGoFuzz {
		return
	}

	longestKindWord := 0
	longestMessageLength := 0
	missed := make([]string, 0)
	assertionTracker.Range(func(k, v any) bool {
		location := k.(string)
		md := v.(*metadata)
		Ensure(location != "", "All assertion records have a location")
		if md.Frequency.Load() == 0 {
			longestKindWord = max(longestKindWord, len(md.Kind))
			longestMessageLength = max(longestMessageLength, len(md.Message))
			missed = append(missed, location)
		}
		return true
	})
	if len(missed) > 0 {
		prefix := ""
		if !INVARIANT_FULL_LOCATION {
			wd, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			prefix = filepath.Dir(wd)
		}
		fmt.Printf("🚨 %d assertions were never true. 🚨\n", len(missed))
		for _, id := range missed {
			md := loadMetadata(id)
			fmt.Printf(
				"\t%*s | %-*s | %s\n",
				longestKindWord, md.Kind,
				longestMessageLength, md.Message,
				strings.TrimPrefix(id, prefix),
			)
		}
		os.Exit(1)
	}

	// === Never-False Analysis for Always/Ensure assertions ===
	//
	// Track Always* and Ensure* assertions that were never evaluated as false.
	// This helps identify assertions whose failure modes were never tested.
	{
		neverFalse := make([]string, 0)
		longestKindWord2 := 0
		longestMessageLength2 := 0

		assertionTracker.Range(func(k, v any) bool {
			location := k.(string)
			md := v.(*metadata)
			switch md.Kind {
			case "Sometimes", "XSometimes":
				if md.FalseFrequency.Load() == 0 {
					longestKindWord2 = max(longestKindWord2, len(md.Kind))
					longestMessageLength2 = max(longestMessageLength2, len(md.Message))
					neverFalse = append(neverFalse, location)
				}
			}
			return true
		})

		if len(neverFalse) > 0 {
			prefix := ""
			if !INVARIANT_FULL_LOCATION {
				wd, err := os.Getwd()
				if err != nil {
					panic(err)
				}
				prefix = filepath.Dir(wd)
			}
			fmt.Printf("🚨 %d assertions were never FALSE. 🚨\n", len(neverFalse))
			for _, id := range neverFalse {
				md := loadMetadata(id)
				fmt.Printf(
					"\t%*s | %-*s | %s\n",
					longestKindWord2, md.Kind,
					longestMessageLength2, md.Message,
					strings.TrimPrefix(id, prefix),
				)
			}
			os.Exit(1)
		}
	}

	// === Analysis ===
	//
	// This is a very simple and “dumb” analysis based solely on absolute assertion frequency.
	// It does not attempt to correlate assertions with specific unit tests or cluster them with
	// each other; any attempt to do so is futile. For that kind of analysis, you’d need
	// instrumentation/static analysis combined with machine learning.
	{
		type scored struct {
			Key   string
			Count int64
		}
		fmt.Printf("Showing up to %d of the least-exercised invariants:\n", leastExercisedInvariantCount)

		longestMessageLength3 := 0
		longestKindWord3 := 0

		h := make([]scored, 0, leastExercisedInvariantCount)
		assertionTracker.Range(func(k, v any) bool {
			key := k.(string)
			assertion := v.(*metadata)
			Ensure(key != "", "Assertion location must not be empty")
			longestMessageLength3 = max(longestMessageLength3, len(assertion.Message))
			longestKindWord3 = max(longestKindWord3, len(assertion.Kind))
			freq := assertion.Frequency.Load()

			if len(h) < leastExercisedInvariantCount {
				h = append(h, scored{Key: key, Count: freq})
				if len(h) == leastExercisedInvariantCount {
					sort.Slice(h, func(i, j int) bool {
						return h[i].Count > h[j].Count
					})
				}
				return true
			}
			if freq < h[0].Count {
				h[0] = scored{Key: key, Count: freq}
				i := 0
				for {
					l, r := 2*i+1, 2*i+2
					if l >= leastExercisedInvariantCount {
						break
					}
					small := l
					if r < leastExercisedInvariantCount && h[r].Count > h[l].Count {
						small = r
					}
					if h[i].Count >= h[small].Count {
						break
					}
					h[i], h[small] = h[small], h[i]
					i = small
				}
			}
			return true
		})

		sort.Slice(h, func(i, j int) bool {
			return h[i].Count < h[j].Count
		})

		for _, v := range h {
			a := loadMetadata(v.Key)
			fmt.Printf(
				"count=%-4d | %-*s | %-*s | %s\n",
				a.Frequency.Load(),
				longestKindWord3, a.Kind,
				longestMessageLength3, a.Message,
				v.Key,
			)
		}
	}
}

// loadMetadata fetches the *metadata for a key, asserting presence.
// Used by AnalyzeAssertionFrequency where the key was just observed during a Range walk.
func loadMetadata(key string) (md *metadata) {
	v, ok := assertionTracker.Load(key)
	Ensure(ok, "metadata is present for key observed during Range")
	return v.(*metadata)
}

// Always calls AssertionFailureCallback if cond is false.
//
// Note 1: Message describes the enforced property with a proper declarative sentence. It's NOT an
// error message.
//
// Note 2: When deferring assertions, wrap them in a closure. Otherwise, cond is evaluated
// immediately. Also, deferring the assertion call without a wrapper closure changes the source
// location for runtime.Callers
//
//	// Correct: deferred assertion evaluates cond later
//	defer func() { invariant.Always(x > 0, "x must be positive") }()
//
//	// Incorrect: cond evaluated immediately
//	defer invariant.Always(x > 0, "x must be positive")
//
// Note 3: The analyzer tracks assertions by source line, so multiple assertions on the same line
// via loops are treated as a single assertion. However, its frequency counter increments on every
// evaluation. To avoid inflating frequency statistics, wrap the loop in a closure that returns a
// single bool, so the assertion is evaluated once for the aggregate condition.
//
//	// Aggregate check over a loop
//	invariant.Always(func() bool {
//	    for _, k := range keys {
//	        if k == forbidden {
//	            return false
//	        }
//	    }
//	    return true
//	}(), "No forbidden keys present")
//
//go:noinline
func Always(cond bool, msg string) {
	if cond {
		registerAssertion()
	} else {
		AssertionFailureCallback(fmt.Sprintf("%s: %s", AssertionFailureMsgPrefix, msg))
	}
}

func AlwaysErrIsNil(err error, msg string) {
	if err != nil {
		msg = fmt.Sprintf("%s: error must be nil. got %q. %s", AssertionFailureMsgPrefix, err, msg)
		AssertionFailureCallback(msg)
	}
	registerAssertion()
}

// Sometimes records that a condition was true at least once throughout the test run. It only has an
// effect in test environments; outside of tests it is a no-op.
//
// Technically, the term "invariant.Sometimes" is a misnomer. It's more accurately described as a
// property check. But I wanted to name this library "invariant" instead of "assert" or "property"
// for better grep-ability... maybe also because it sounds cool.
//
//go:noinline
func Sometimes(cond bool, msg string) (ok bool) {
	if !cond {
		registerFalseAssertion()
		return cond
	}
	registerAssertion()
	return cond
}

// Reachable recordst that a source line was evaluated at least once throughout a test run. This is
// primarily useful for situations wherein using Sometimes is impractical such in switch cases. We
// can't simply embed a invariant.Sometimes(true, "") in each case as the tracker would flag that
// these never evaluated to false.
//
//	 switch action {
//	 case "UPSERT":
//		invariant.Sometimes(true, "Action is UPSERT") // Would get flagged by the tracker
//		changeAction = types.ChangeActionUpsert
//	 case "DELETE":
//		invariant.Reachable("Action is DELETE")
//		changeAction = types.ChangeActionDelete
//	 case "CREATE":
//		changeAction = types.ChangeActionCreate
//	 default:
//	 	return types.Change{}, fmt.Errorf("invalid action: %s", action)
//	 }
func Reachable(msg string) {
	registerAssertion()
}

// Ensure is the same as invariant.Always but it will panic in ALL environments.
//
//go:noinline
func Ensure(cond bool, msg string) {
	if cond {
		registerAssertion()
	} else {
		AssertionFailureCallback(AssertionFailureMsgPrefix + ": " + msg)
		panic(AssertionFailureMsgPrefix + ": " + msg) // in case Callback doesn't panic
	}
}

func EnsureErrIsNil(err error, msg string) {
	if err != nil {
		msg = fmt.Sprintf("%s: error must be nil. got %q. %s", AssertionFailureMsgPrefix, err, msg)
		AssertionFailureCallback(msg)
		panic(msg) // in case Callback doesn't panic
	}
	registerAssertion()
}

// AlwaysWhenReachable is used for non-deterministic code paths that should theoretically never execute
// in production. Unlike Always, it does NOT register the assertion with the tracker, making it invisible
// to test analysis.
//
// This is intended for the "Real" implementation counterparts of Fake test interfaces. Real implementations
// typically depend on external systems (databases, APIs, network calls) that introduce non-determinism.
// Assertions on these code paths cannot be reliably tested because:
//   - External system behavior is non-deterministic
//   - Test environments may not have the resources to trigger all code paths
//   - We want to assert against impossible states in production without failing tests
//
// When the condition is false, it calls AssertionFailureCallback to report the failure in production
// environments. In test environments, this typically panics.
//
// Example usage:
//
//	func (c *ClientReal) ListDevices(ctx context.Context) (Devices, error) {
//	    if c == nil {
//	        // Theoretically impossible if properly constructed, but handle it defensively
//	        invariant.AlwaysWhenReachable(false, "ClientReal should never be nil")
//	        return nil, errors.New("internal error")
//	    }
//	    // ... make API call ...
//	}
//
//go:noinline
func AlwaysWhenReachable(cond bool, msg string) {
	if !cond {
		AssertionFailureCallback(fmt.Sprintf("%s: %s", AssertionFailureMsgPrefix, msg))
	}
}

//go:noinline
func AlwaysErrIsNilWhenReachable(err error, msg string) {
	if err != nil {
		msg = fmt.Sprintf("%s: error must be nil. got %q. %s", AssertionFailureMsgPrefix, err, msg)
		AssertionFailureCallback(msg)
	}
}

//go:noinline
func EnsureWhenReachable(cond bool, msg string) {
	if !cond {
		AssertionFailureCallback(AssertionFailureMsgPrefix + ": " + msg)
		panic(AssertionFailureMsgPrefix + ": " + msg) // in case Callback doesn't panic
	}
}

//go:noinline
func EnsureErrIsNilWhenReachable(err error, msg string) {
	if err != nil {
		msg = fmt.Sprintf("%s: error must be nil. got %q. %s", AssertionFailureMsgPrefix, err, msg)
		AssertionFailureCallback(msg)
		panic(msg) // in case Callback doesn't panic
	}
}

//go:noinline
func Unimplemented(msg string) {
	AssertionFailureCallback(fmt.Sprintf("%s: %s", AssertionFailureMsgPrefix, msg))
	panic(fmt.Sprintf("%s: %s", AssertionFailureMsgPrefix, msg))
}

//go:noinline
func Unreachable(msg string) {
	AssertionFailureCallback(fmt.Sprintf("%s: %s", AssertionFailureMsgPrefix, msg))
	panic(fmt.Sprintf("%s: %s", AssertionFailureMsgPrefix, msg))
}

type _Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Until returns a bounded sequence useful for safely constraining infinite loops.
// If the break condition never evaluates, the loop panics on the final iteration.
// Under "disable_assertions", the loop still runs but runaway loops are undetected.
// If you need an explicit game loop, refer to invariant.GameLoop.
//
// Usage:
//
//	// Instead of this:
//	for {
//		if cond() {
//			break
//		}
//		doWork()
//	}
//	// Do this:
//	for range invariant.Until(10_000) {
//		if cond() {
//			break
//		}
//		doWork()
//	}
//
//go:noinline
func Until[T _Number](limit T) (seq iter.Seq[int]) {
	Ensure(limit > 0, "Loop bound is a positive integer")
	return func(yield func(int) bool) {
		intLimit := int(limit)
		for iteration := range intLimit {
			if iteration == intLimit-1 {
				AssertionFailureCallback("Runaway loop!")
				return
			}
			if !yield(iteration) {
				return
			}
		}
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
func GameLoop() (seq iter.Seq[int]) {
	return func(yield func(int) bool) {
		for iteration := 0; true; iteration++ {
			if !yield(iteration) {
				return
			}
		}
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
func IsAssertionFailure(v any) (ok bool) {
	message, ok := v.(string)
	return ok && strings.HasPrefix(message, AssertionFailureMsgPrefix)
}

func UniqueCount[T any, K comparable](items []T, getKey func(T) K) (count int) {
	if len(items) < 2 {
		return len(items)
	}
	keys := make(map[K]struct{})
	for _, item := range items {
		key := getKey(item)
		if _, exists := keys[key]; !exists {
			keys[key] = struct{}{}
			count++
		}
	}
	return count
}

type RegexMatchInput struct {
	Str     string
	Pattern string
}

func RegexMatch(in RegexMatchInput) (ok bool) {
	matched, _ := regexp.MatchString(in.Pattern, in.Str)
	return matched
}

func All[T any](items []T, cond func(T) bool) (ok bool) {
	for _, item := range items {
		if !cond(item) {
			return false
		}
	}
	return true
}
func None[T any](items []T, cond func(T) bool) (ok bool) {
	for _, item := range items {
		if cond(item) {
			return false
		}
	}
	return true
}

func AllUnique[T any, K comparable](items []T, getKey func(T) K) (ok bool) {
	return UniqueCount(items, getKey) == len(items)
}

func AllIdentical[T any, K comparable](items []T, getKey func(T) K) (ok bool) {
	if len(items) < 2 {
		return true
	}
	base := getKey(items[0])
	for _, item := range items[1:] {
		if getKey(item) != base {
			return false
		}
	}
	return true
}

/*
XAlways evaluates fn and calls AssertionFailureCallback if it returns false. It
is designed for use cases where you want to perform expensive validations that
can be disabled in production builds using the `disable_assertions`
build tag.

	expensiveFn := func() bool { ... }
	// expensiveFn is still evaluated but boolean check is a noop under disable_assertions
	invariant.Always(expensiveFn())


	// expensiveFn itself will be a noop under disable_assertions
	invariant.XAlways(expensiveFn)

Be wary of this if you rely on side effects produced by fn. Rule of thumb would
be to ensure that fn is pure or idempotent.

Lastly, remember to wrap these functions in a closure when deferring them. Refer to invariant.Always
*/
//go:noinline
func XAlways(fn func() bool, msg string) {
	if fn() {
		registerAssertion()
	} else {
		AssertionFailureCallback(fmt.Sprintf("%s: %s\n", AssertionFailureMsgPrefix, msg))
	}
}

//go:noinline
func XAlwaysNil(fn func() any, msg string) {
	if v := fn(); v == nil {
		registerAssertion()
	} else {
		AssertionFailureCallback(fmt.Sprintf("%s: %s: got %v\n", AssertionFailureMsgPrefix, msg, v))
	}
}

func XEnsure(fn func() bool, msg string) {
	if fn() {
		registerAssertion()
	} else {
		failureMsg := fmt.Sprintf("%s: %s\n", AssertionFailureMsgPrefix, msg)
		AssertionFailureCallback(failureMsg)
		panic(failureMsg)
	}
}

//go:noinline
func XSometimes(fn func() bool, msg string) {
	if !IsRunningUnderGoTest || !fn() {
		return
	}
	registerAssertion()
}
