// Package invariant provides a property testing framework for Go: encode and enforce
// invariants directly in code, then track which assertions ever fired during the test
// suite to surface untested branches.
//
// This package is the pure library tier. All dependencies arrive as fields on a
// Recorder: source filesystem, runtime.Callers wrapper, exit hook, output sink,
// failure callback. The pure tier never imports os or runtime. For an OS-bound
// default ready to drop into tests, import the sibling composition-tier package
// github.com/james-orcales/james-orcales/golang_snacks/invariant/v2/invariant_default.
//
// Assertion kinds:
//
//   - Always — must hold true for every evaluation. One counter-example fails the test.
//   - Sometimes — must evaluate to both true and false across the test run. Observing
//     only one of those states fails the test.
//   - Ensure — like Always but panics in every environment, not just under tests.
package invariant

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"iter"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"unicode"
	"unicode/utf8"
)

// Assertion_Failure_Message_Prefix is prepended to every failure message emitted by
// the recorder. Used by Is_Assertion_Failure to detect recovered panics that came
// from this package.
const Assertion_Failure_Message_Prefix = "🚨 Assertion Failure 🚨"

// Framework_Precondition_Violation_Prefix is prepended to every panic raised
// by recorder_framework_panic_unless. Distinct from Assertion_Failure_Message_Prefix
// so callers can distinguish a framework-internal precondition violation
// (the user passed nonsensical parameters to the library, or the library's
// own state is corrupt) from a user-domain assertion failure (the user's
// program-level Always / Ensure / etc. fired false).
const Framework_Precondition_Violation_Prefix = "🛑 invariant framework precondition violated 🛑"

// Recorder holds every dependency the pure library tier needs. Construct one
// directly for tests; for production use the sibling invariant_default package
// which builds an OS-bound singleton.
type Recorder struct {
	// Output receives diagnostic output: failure messages, never-fired reports.
	Output io.Writer

	// Tty, when non-nil, receives the post-run coverage summary on a clean
	// successful test run. The composition tier wires this to /dev/tty (when
	// openable) so the line survives `go test` without `-v` — the test binary's
	// stdout/stderr are captured by go test on pass, but a tty write bypasses
	// the capture. nil falls back to Output.
	Tty io.Writer

	// File_System reads Go source files during AST analysis. Paths captured by
	// Get_Caller are absolute OS paths; lookups strip the leading "/" before
	// calling fs.ReadFile.
	File_System fs.FS

	// Get_Caller returns the frame information for the caller at the given skip
	// depth. The composition tier wires this to runtime.Callers. Tests can
	// substitute a no-op or a hardcoded frame.
	Get_Caller func(skip int) (frame_information Frame_Information, err error)

	// Exit terminates the test process with the given code on user-domain
	// assertion failures when Fatal_Failures is set. Composition tier wires
	// this to os.Exit. Tests substitute a recording stub.
	Exit func(code int)

	// Framework_Exit terminates the process when a framework-internal
	// precondition is violated (library misuse, corrupt state). Separate
	// from Exit so the framework's own failure mode is isolated from the
	// user's Fatal_Failures / Exit configuration — a user of
	// invariant_default setting Default.Exit to a custom handler for their
	// own assertions does NOT redirect framework-internal violations.
	// Composition tier wires this to os.Exit. Pure-tier tests stub it (or
	// leave nil to fall back to a panic, which is also recoverable).
	Framework_Exit func(code int)

	// Fatal_Failures, when true, makes Ensure-class assertions terminate the
	// process via Exit(1) instead of panicking. Use this when running in
	// environments where a panic would be swallowed or where a structured
	// exit-code signal is preferred (CI runs, long-lived services, etc.).
	Fatal_Failures bool

	// Full_Location, when true, prints absolute file paths in reports.
	Full_Location bool

	// Stacktrace_Depth bounds the number of frames captured for failure context.
	Stacktrace_Depth int

	// Working_Directory is the path prefix to trim from report output when
	// Full_Location is false. Empty disables trimming.
	Working_Directory string

	// Is_Test, Is_Fuzz, Is_Benchmark are externally-supplied flags describing
	// the runtime environment. Composition tier derives these from os.Args.
	Is_Test      bool
	Is_Fuzz      bool
	Is_Benchmark bool

	// Failure_Callback receives every assertion-failure message before any
	// further action (panic for Ensure-kind, return for Always-kind). When nil,
	// the recorder writes to Output.
	Failure_Callback func(message string)

	// Packages_To_Analyze lists absolute directory paths whose Go source files
	// are walked for invariant.X call-site registration.
	Packages_To_Analyze []string

	// Assertions stores pre-registered assertion metadata keyed by
	// "absolute_file_path:line". Read-mostly after registration; concurrent
	// increments are atomic.
	Assertions sync.Map

	// Axis_Handle_Pool produces *Axis_Handle values used for pointer-identity
	// matching of Bucket_References to their source axes inside a single
	// Cross_Product call. Construction wires it via
	//   sync.Pool{New: func() any { return new(Axis_Handle) }}.
	// Recorder_Always / Recorder_Sometimes / Recorder_Distinct_Boundary acquire one
	// handle per axis; recorder_cross_product_at releases every record's
	// handle back to the pool at the end of the call. Pool reuse across calls
	// (within the same Recorder) is safe because uniqueness is required only
	// within one call, not across calls.
	Axis_Handle_Pool sync.Pool
}

// Frame_Information is the runtime-source-location pair captured by Get_Caller.
type Frame_Information struct {
	File string
	Line int
}

// Assertion_Metadata tracks per-call-site state. Frequency counts true
// evaluations; False_Frequency counts false evaluations (only Sometimes
// distinguishes the two). Site is the file:line of the registering call
// stripped of any tuple suffix — the report groups gaps by Site. For
// Cross entries Cross_Buckets carries the per-axis decomposition of this
// tuple and Tuple_Indices the numeric tuple coordinate; both are nil for
// scalar Is_* sugar entries that key by file:line alone.
type Assertion_Metadata struct {
	Frequency       atomic.Int64
	False_Frequency atomic.Int64
	Message         string
	Kind            string
	Site            string
	Cross_Buckets   []Cross_Bucket
	Tuple_Indices   []int
}

// Cross_Bucket captures one axis's chosen bucket at a Cross tuple — the
// triple (helper kind, value expression, bucket name) plus the axis's
// string-literal description. The report uses these to render the axis
// list once per site instead of repeating it on every tuple line.
type Cross_Bucket struct {
	Composable_Kind  string
	Value_Expression string
	Bucket_Name      string
	Axis_Message     string
}

// Regex_Match_Input is the input for Regex_Match.
type Regex_Match_Input struct {
	String  string
	Pattern string
}

// Cross_Axis carries one helper's per-call bucket classification. Composable
// helpers (Always, Sometimes, Boundary, …) return it; Cross_Product consumes
// it. A record that is never passed to Cross_Product is a no-op for coverage
// — bare composables only enforce their hard-bound checks. Message carries
// the per-axis description (the string literal passed at the call site) so
// coverage reports can name what each axis is tracking.
//
// Handle is the per-call identity stamp produced by Axis_Handle_Pool and read
// by Bucket_False / Bucket_True / Bucket_Lo / Bucket_Hi when an author writes
// a constraint cell against this axis. Two axes within one Cross_Product
// call always carry distinct handles; the same handle pointer may be reused
// across Cross_Product calls via the pool. nil for Excluding records, which
// are not axes.
//
// Constraint is populated only for Excluding records — the list of bucket-
// cells that constitute the clause. nil for axis records (Always /
// Sometimes / Boundary).
type Cross_Axis struct {
	Axis_Kind    string
	Bucket_Index int
	Bucket_Name  string
	Message      string
	Handle       *Axis_Handle
	Constraint   []Bucket_Reference
}

// Axis_Handle is the per-axis identity token. The pointer itself is the
// entire information content — callers never construct one directly; they
// receive one stamped on a Cross_Axis via Recorder_Always / Recorder_Sometimes
// / Recorder_Distinct_Boundary and reference it indirectly through Bucket_False
// / Bucket_True / Bucket_Lo / Bucket_Hi. The struct is exported only because
// it appears in the exported Cross_Axis / Bucket_Reference field types; treat
// it as opaque.
//
// The byte field is load-bearing: a zero-size struct{} type lets the Go
// runtime collapse all new(T) allocations to the same address (spec:
// "Two distinct zero-size variables may have the same address in memory"),
// which would break pointer identity across axes within one Cross_Product
// call. One byte forces unique heap addresses without otherwise mattering.
type Axis_Handle struct {
	_ byte
}

// Bucket_Reference names one cell within a Cross_Product axis. Constructed by
// Bucket_False / Bucket_True / Bucket_Lo / Bucket_Hi; consumed by Excluding.
// The Handle pointer is matched against the axes passed to Cross_Product to
// resolve which axis position this cell refers to.
type Bucket_Reference struct {
	Handle       *Axis_Handle
	Bucket_Index int
}

// Integer_Like constrains a generic to any of Go's built-in integer kinds.
type Integer_Like interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Numeric widens Integer_Like to also accept float kinds. Used by Boundary
// so a single helper covers both integer and floating-point endpoint
// observation.
type Numeric interface {
	Integer_Like | ~float32 | ~float64
}

// recorder_caller_location returns "file:line:0" for the frame skip levels
// above the Get_Caller closure, or "" when the location can't be determined.
// Column is always 0 — runtime.Callers does not surface column information.
func recorder_caller_location(r *Recorder, skip int) (location string) {
	if r.Get_Caller == nil {
		return ""
	}
	frame, err := r.Get_Caller(skip)
	if err != nil || frame.File == "" {
		return ""
	}
	return frame.File + ":" + strconv.Itoa(frame.Line) + ":0"
}

// Routes a failure message through Failure_Callback when set, otherwise writes
// to Output. The pure-tier primitives call this before panicking or returning
// so test code can intercept the message.
func recorder_emit_failure(r *Recorder, message string) {

	if r.Failure_Callback != nil {
		r.Failure_Callback(message)
		return
	}
	fmt.Fprintln(r.Output, message)
}

// Emits the failure message and terminates the assertion: when Fatal_Failures
// is set, calls Exit(1) and returns; otherwise panics with the formatted
// message. Used by every Ensure-class primitive (Ensure, Ensure_Nil_Error,
// Unreachable, Unimplemented, …) so the panic-vs-exit decision lives in one
// place.
func recorder_fail_fatal(r *Recorder, formatted string) {

	recorder_emit_failure(r, formatted)
	if r.Fatal_Failures {
		r.Exit(1)
		return
	}
	panic(formatted)
}

// Asserts a precondition the framework itself relies on. Routes violations
// through Framework_Exit, NOT through the user-domain failure path
// (Failure_Callback, Fatal_Failures, Exit). Framework-internal violations
// terminate via os.Exit in production (composition tier wires
// Framework_Exit accordingly) and don't honor any user configuration. Falls
// back to a panic when Framework_Exit is nil (pure-tier tests can either
// stub it for clean capture or rely on the panic for defer-recover).
func recorder_framework_panic_unless(r *Recorder, condition bool, message string) {

	if condition {
		return
	}
	formatted := Framework_Precondition_Violation_Prefix + ": " + message
	if r.Framework_Exit != nil {
		// Exit doesn't carry the message anywhere visible, so surface it
		// via Output before terminating.
		if r.Output != nil {
			fmt.Fprintln(r.Output, formatted)
		}
		r.Framework_Exit(1)
		return
	}
	// The panic value carries the message; no emit needed — emitting here
	// would duplicate the message for any recover-and-reprint handler.
	panic(formatted)
}

// Records that an assertion evaluated true at the caller's source location.
// No-op outside test environments. The skip parameter is forwarded to
// Get_Caller — callers pick the value matching their call depth.
//
//go:noinline
func recorder_register_assertion(r *Recorder, skip int) {

	if !r.Is_Test {
		return
	}
	if r.Is_Benchmark {
		return
	}
	if r.Is_Fuzz {
		return
	}
	frame_information, err := r.Get_Caller(skip)
	if err != nil {
		return
	}
	if frame_information.File == "" {
		return
	}
	key := frame_information.File + ":" + strconv.Itoa(frame_information.Line)
	value, ok := r.Assertions.Load(key)
	if !ok {
		return
	}
	value.(*Assertion_Metadata).Frequency.Add(1)
}

// Records that a Sometimes assertion evaluated false at the caller's source
// location.
//
//go:noinline
func recorder_register_false_assertion(r *Recorder, skip int) {

	if !r.Is_Test {
		return
	}
	if r.Is_Benchmark {
		return
	}
	if r.Is_Fuzz {
		return
	}
	frame_information, err := r.Get_Caller(skip)
	if err != nil {
		return
	}
	if frame_information.File == "" {
		return
	}
	key := frame_information.File + ":" + strconv.Itoa(frame_information.Line)
	value, ok := r.Assertions.Load(key)
	if !ok {
		return
	}
	value.(*Assertion_Metadata).False_Frequency.Add(1)
}

// Recorder_Run_Test_Main is the canonical TestMain body for tests that want
// assertion tracking. It registers the analyzed packages, runs the test suite,
// reports any unexercised assertions, then exits with the test suite's code.
// On a clean run (m.Run() returned 0 and Analyze_Assertion_Frequency found
// nothing) it prints the count of registered assertion tuples to r.Output —
// every entry in r.Assertions was tested.
func Recorder_Run_Test_Main(r *Recorder, m *testing.M, dirs ...string) {

	// Fatal_Failures terminates via Exit(1); Failure_Callback intercepts the
	// message before panic. Setting both leaves the failure semantics
	// ambiguous (which one wins?), so refuse the combination up-front rather
	// than picking silently.
	recorder_framework_panic_unless(r, !(r.Fatal_Failures && r.Failure_Callback != nil),
		"Fatal_Failures and Failure_Callback are mutually exclusive")
	Recorder_Register_Packages_For_Analysis(r, dirs...)
	code := m.Run()
	Recorder_Analyze_Assertion_Frequency(r)
	if code == 0 {
		// Tty bypasses `go test`'s stdout/stderr capture so this line shows
		// even without -v. Falls back to Output when no tty is available.
		summary_output := r.Tty
		if summary_output == nil {
			summary_output = r.Output
		}
		individual, combinations, panic_able := recorder_count_assertions(r)
		fmt.Fprintf(summary_output,
			"✓ tested %d properties (%d individual + %d combinations, of which %d are panic-able)\n",
			individual+combinations, individual, combinations, panic_able)
	}
	r.Exit(code)
}

// Returns the split assertion count: `individual` counts standalone
// observations (Is_* sugar entries plus per-axis Cross_Axis_* entries —
// each is a single property), `combinations` counts entries that came from
// the Cartesian-product expansion of a Cross_Product call (Kind="Cross",
// including Is_Boundary's degenerate one-axis registration). Sum = total
// properties tested. `panic_able` is the subset whose contract is "must
// hold or fail-fatal at runtime" — see recorder_metadata_is_panic_able
// for the exact Kind enumeration.
func recorder_count_assertions(r *Recorder) (individual int, combinations int, panic_able int) {

	r.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*Assertion_Metadata)
		if metadata.Kind == "Cross" {
			combinations++
		} else {
			individual++
		}
		if recorder_metadata_is_panic_able(metadata.Kind) {
			panic_able++
		}
		return true
	})
	return individual, combinations, panic_able
}

// Reports whether an Assertion_Metadata Kind denotes a property whose
// violation triggers a fail-fatal at runtime — the Always family.
// Coverage-only kinds (Sometimes, Cross tuple trackers) return false:
// a never-fired Sometimes branch fails the suite at end-of-run analysis,
// but does not panic mid-execution the way an Always-class violation does.
// Registry-faithful: an Is_Distinct_Boundary site (registered as Cross
// tuples, no Cross_Axis_Always entries) contributes 0, even though the
// underlying Distinct_Boundary primitive performs fail-fatal bound checks
// at runtime.
func recorder_metadata_is_panic_able(kind string) (yes bool) {

	switch kind {
	case "Cross_Axis_Always",
		"Is_Always", "Recorder_Is_Always",
		"Is_Always_Nil_Error", "Recorder_Is_Always_Nil_Error":
		return true
	}
	return false
}

// Recorder_Register_Packages_For_Analysis walks the named directories' Go source
// files (via r.File_System) and pre-registers every literal invariant.X call
// site in r.Assertions so subsequent runtime firings have something to count
// against. Dirs is empty by default, meaning the recorder's existing
// Packages_To_Analyze is used unchanged.
//
// NOTE: Every primitive's last argument must be a string-literal message.
// Sprintf-shaped expressions are rejected.
//
// NOTE: During fuzz or benchmark runs the worker processes don't share memory
// with the parent, so registration is skipped.
func Recorder_Register_Packages_For_Analysis(r *Recorder, dirs ...string) {

	// Framework-internal preconditions: recorder_framework_panic_unless so violations
	// surface as Go panics regardless of the recorder's Fatal_Failures /
	// Failure_Callback configuration. Internal violations are framework bugs
	// / misuse, not user-domain assertion failures, and should not be routed
	// through the user-configurable failure path.
	recorder_framework_panic_unless(r, r.Is_Test, "Register_Packages_For_Analysis runs under tests")
	if r.Is_Benchmark {
		return
	}
	if r.Is_Fuzz {
		return
	}
	if len(dirs) > 0 {
		r.Packages_To_Analyze = dirs
	}
	for index, path := range r.Packages_To_Analyze {
		absolute_path, err := filepath.Abs(path)
		if err != nil {
			panic(fmt.Sprintf("Failed to convert package path to absolute: %s", err))
		}
		recorder_framework_panic_unless(r, absolute_path != "", "Package path resolves to an absolute path")
		r.Packages_To_Analyze[index] = absolute_path
	}
	files := recorder_register_packages_for_analysis_collect(r)
	recorder_framework_panic_unless(r, len(files) > 0, "At least one Go source file was discovered")
	recorder_register_packages_for_analysis_parse(r, files)
}

// Walks every directory in r.Packages_To_Analyze and returns the absolute
// paths of every non-test .go file found. Strips the leading "/" before
// walking via r.File_System.
func recorder_register_packages_for_analysis_collect(r *Recorder) (files []string) {

	files = make([]string, 0)
	for _, directory := range r.Packages_To_Analyze {
		before_count := len(files)
		root := strings.TrimPrefix(directory, "/")
		err := fs.WalkDir(r.File_System, root, func(path string, d fs.DirEntry, err error) (returned error) {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if path == root {
					return nil
				}
				return fs.SkipDir
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			files = append(files, "/"+path)
			return nil
		})
		if err != nil {
			panic(fmt.Sprintf("Collecting files for analysis: %s", err))
		}
		after_count := len(files)
		recorder_framework_panic_unless(r, before_count < after_count, "Directory contains at least one Go source file")
	}
	return files
}

// Parses every file in parallel and registers each invariant.X call site it
// finds. Panics raised by a worker (e.g. from
// recorder_framework_panic_unless when an assertion site has a non-literal
// message) are captured and re-raised on the caller's goroutine after all
// workers finish — without this forwarding, a worker panic would crash the
// whole process instead of reaching the caller's defer/recover.
func recorder_register_packages_for_analysis_parse(r *Recorder, files []string) {

	var wait_group sync.WaitGroup
	var first_panic_mu sync.Mutex
	var first_panic any
	for _, file_path := range files {
		wait_group.Add(1)
		go func(path string) {
			defer wait_group.Done()
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				first_panic_mu.Lock()
				defer first_panic_mu.Unlock()
				if first_panic == nil {
					first_panic = recovered
				}
			}()
			recorder_register_packages_for_analysis_parse_one(r, path)
		}(file_path)
	}
	wait_group.Wait()
	if first_panic != nil {
		panic(first_panic)
	}
}

// Parses one file and registers each invariant.X call site found inside it.
// Walks per-FuncDecl so each function's local axis-variable bindings can be
// resolved when a Cross_Product call's args (or its Excluding cells)
// reference an axis by name (`enabled := invariant.Sometimes(…)`) rather than
// inlining the axis call.
func recorder_register_packages_for_analysis_parse_one(r *Recorder, file_path string) {

	content, err := fs.ReadFile(r.File_System, strings.TrimPrefix(file_path, "/"))
	if err != nil {
		return
	}
	file_set := token.NewFileSet()
	node, err := parser.ParseFile(file_set, file_path, content, parser.ParseComments)
	if err != nil {
		return
	}
	for _, decl := range node.Decls {
		func_decl, is_func := decl.(*ast.FuncDecl)
		if !is_func || func_decl.Body == nil {
			// Top-level decls (vars, consts, type-bound funcs without a body)
			// can still embed invariant.X calls in initializers; walk them
			// with empty maps.
			ast.Inspect(decl, func(n ast.Node) (descend bool) {
				recorder_register_packages_for_analysis_visit(r,
					&recorder_register_packages_for_analysis_visit_input{
						File_Set: file_set, File_Path: file_path, Node: n,
					})
				return true
			})
			continue
		}
		bindings := ast_collect_axis_bindings(func_decl.Body)
		legitimate_constraints := ast_collect_legitimate_constraint_calls(func_decl)
		legitimate_axis_calls := ast_collect_legitimate_axis_calls(func_decl)
		ast.Inspect(func_decl, func(n ast.Node) (descend bool) {
			recorder_register_packages_for_analysis_visit(r,
				&recorder_register_packages_for_analysis_visit_input{
					File_Set:               file_set,
					File_Path:              file_path,
					Node:                   n,
					Bindings:               bindings,
					Legitimate_Constraints: legitimate_constraints,
					Legitimate_Axis_Calls:  legitimate_axis_calls,
				})
			return true
		})
	}
}

// ast_collect_legitimate_constraint_calls walks the function body and
// records every invariant.Excluding call that appears as a direct argument
// of an invariant.Cross_Product / Recorder_Cross_Product call. Used by
// recorder_register_packages_for_analysis_visit to detect stray Excluding
// calls — ones written outside any Cross_Product argument list — which are
// a misuse and framework-panic at registration.
func ast_collect_legitimate_constraint_calls(func_decl *ast.FuncDecl) (legitimate map[*ast.CallExpr]bool) {

	legitimate = map[*ast.CallExpr]bool{}
	ast.Inspect(func_decl, func(n ast.Node) (descend bool) {
		call, is_call := n.(*ast.CallExpr)
		if !is_call {
			return true
		}
		selector, is_selector := call.Fun.(*ast.SelectorExpr)
		if !is_selector {
			return true
		}
		identifier, is_ident := selector.X.(*ast.Ident)
		if !is_ident {
			return true
		}
		if identifier.Name != "invariant" {
			return true
		}
		if selector.Sel.Name != "Cross_Product" && selector.Sel.Name != "Recorder_Cross_Product" {
			return true
		}
		for _, arg := range call.Args {
			arg_call, is_arg_call := arg.(*ast.CallExpr)
			if !is_arg_call {
				continue
			}
			legitimate[arg_call] = true
		}
		return true
	})
	return legitimate
}

// ast_collect_legitimate_axis_calls walks the function body and records
// every axis-constructor call — invariant.Sometimes, invariant.Always,
// invariant.Distinct_Boundary (and Recorder_ variants) — that is reachable
// from a Cross_Product argument list, either as a direct argument or as the
// RHS of an axis-builder binding whose LHS identifier appears as a
// Cross_Product argument. Axis-constructor calls outside this set are
// orphans — the returned Cross_Axis is dropped on the floor without ever
// reaching r.Assertions, so the assertion contributes nothing at runtime.
// recorder_register_packages_for_analysis_visit uses the set to panic on
// orphan axis calls at registration, mirroring the Excluding-outside-
// Cross_Product check.
//
// Two-pass: the first pass collects Ident names that appear as Cross_Product
// arguments AND every CallExpr that appears directly as a Cross_Product
// argument. The second pass walks all axis-builder assignments (`x :=
// invariant.X(…)`) and marks the RHS as legitimate when the LHS name is in
// the Cross_Product-arg set. Two assignments to the same name (one in a
// deferred FuncLit, one in the outer body — both consumed by their
// respective Cross_Products) both clear, because every axis-builder call
// whose LHS is referenced is in the set, not just the last write.
func ast_collect_legitimate_axis_calls(
	func_decl *ast.FuncDecl,
) (legitimate map[*ast.CallExpr]bool) {

	legitimate = map[*ast.CallExpr]bool{}
	cross_product_arg_names := map[string]bool{}
	ast.Inspect(func_decl, func(n ast.Node) (descend bool) {
		call, is_call := n.(*ast.CallExpr)
		if !is_call {
			return true
		}
		selector, is_selector := call.Fun.(*ast.SelectorExpr)
		if !is_selector {
			return true
		}
		identifier, is_ident := selector.X.(*ast.Ident)
		if !is_ident {
			return true
		}
		if identifier.Name != "invariant" {
			return true
		}
		if selector.Sel.Name != "Cross_Product" && selector.Sel.Name != "Recorder_Cross_Product" {
			return true
		}
		for _, arg := range call.Args {
			arg_call, is_arg_call := arg.(*ast.CallExpr)
			if is_arg_call {
				legitimate[arg_call] = true
				continue
			}
			arg_ident, is_arg_ident := arg.(*ast.Ident)
			if !is_arg_ident {
				continue
			}
			cross_product_arg_names[arg_ident.Name] = true
		}
		return true
	})
	ast.Inspect(func_decl, func(n ast.Node) (descend bool) {
		assign, is_assign := n.(*ast.AssignStmt)
		if !is_assign {
			return true
		}
		if len(assign.Lhs) != len(assign.Rhs) {
			return true
		}
		for i := range assign.Lhs {
			lhs_ident, is_lhs_ident := assign.Lhs[i].(*ast.Ident)
			if !is_lhs_ident {
				continue
			}
			if !cross_product_arg_names[lhs_ident.Name] {
				continue
			}
			rhs_call, is_rhs_call := assign.Rhs[i].(*ast.CallExpr)
			if !is_rhs_call {
				continue
			}
			legitimate[rhs_call] = true
		}
		return true
	})
	return legitimate
}

// Reports whether selector_name is an axis-constructor name. Axis
// constructors return a Cross_Axis that has no effect unless consumed by a
// Cross_Product (either directly as an argument or via an axis-builder
// binding referenced from one). recorder_register_packages_for_analysis_visit
// panics on these when they appear outside any Cross_Product's argument
// list — same role as the Excluding-outside-Cross_Product check.
func recorder_register_packages_for_analysis_is_axis_constructor(selector_name string) (yes bool) {

	switch selector_name {
	case "Sometimes", "Always", "Distinct_Boundary",
		"Recorder_Sometimes", "Recorder_Always", "Recorder_Distinct_Boundary":
		return true
	}
	return false
}

// ast_collect_axis_bindings walks one function body collecting
// { ident_name → underlying CallExpr } assignments. Used by
// recorder_register_packages_for_analysis_register_cross to resolve
// variable-bound axis args (`enabled := invariant.Sometimes(…)`) inside a
// Cross_Product call. Both `:=` and `=` are honored; the last assignment
// wins for re-bound names, which mirrors Go's scoping for the cases this
// feature targets (one axis bound, then referenced in the same scope).
func ast_collect_axis_bindings(body *ast.BlockStmt) (bindings map[string]*ast.CallExpr) {

	bindings = map[string]*ast.CallExpr{}
	ast.Inspect(body, func(n ast.Node) (descend bool) {
		assign, is_assign := n.(*ast.AssignStmt)
		if !is_assign {
			return true
		}
		if len(assign.Lhs) != len(assign.Rhs) {
			return true
		}
		for i := range assign.Lhs {
			ident, is_ident := assign.Lhs[i].(*ast.Ident)
			if !is_ident {
				continue
			}
			call, is_call := assign.Rhs[i].(*ast.CallExpr)
			if !is_call {
				continue
			}
			bindings[ident.Name] = call
		}
		return true
	})
	return bindings
}

// Inspects one AST node; if it's an invariant.X(...) call, registers it in
// r.Assertions. Dispatches Cross_Product to a tuple registrar that inspects
// each axis-helper sub-call; dispatches single-axis Is_Boundary to the
// of-path that fans the helper's bucket set into per-bucket entries. The
// bindings map is the per-function axis-variable map produced by
// ast_collect_axis_bindings; nil for top-level decls.
type recorder_register_packages_for_analysis_visit_input struct {
	File_Set               *token.FileSet
	File_Path              string
	Node                   ast.Node
	Bindings               map[string]*ast.CallExpr
	Legitimate_Constraints map[*ast.CallExpr]bool
	Legitimate_Axis_Calls  map[*ast.CallExpr]bool
}

func recorder_register_packages_for_analysis_visit(
	r *Recorder,
	input *recorder_register_packages_for_analysis_visit_input,
) {

	call, ok := input.Node.(*ast.CallExpr)
	if !ok {
		return
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	identifier, ok := selector.X.(*ast.Ident)
	if !ok {
		return
	}
	if identifier.Name != "invariant" {
		return
	}
	file_set := input.File_Set
	file_path := input.File_Path
	switch selector.Sel.Name {
	case "Cross_Product", "Recorder_Cross_Product":
		recorder_register_packages_for_analysis_register_cross(r, file_set, file_path, call, input.Bindings)
		return
	case "Is_Distinct_Boundary", "Recorder_Is_Distinct_Boundary":
		recorder_register_packages_for_analysis_register_of(r, file_set, file_path, call)
		return
	case "Excluding":
		// 🛑 DO NOT replace this framework-panic with a silent return for
		// "tolerance" of stray Excluding calls. The whole point of this
		// detector is that an Excluding clause outside a Cross_Product is
		// dead — it shapes no never-fired report and enforces nothing at
		// runtime. Silently accepting it lets authors ship constraints
		// that do nothing. Read Test_Register_Framework_Panics_On_Excluding_Outside_Cross_Product
		// before "fixing" this.
		if input.Legitimate_Constraints[call] {
			return
		}
		position := file_set.Position(call.Lparen)
		location := file_path + ":" + strconv.Itoa(position.Line)
		recorder_framework_panic_unless(r, false, fmt.Sprintf(
			"invariant.%s at %s: clause appears outside any Cross_Product / Recorder_Cross_Product "+
				"call — Excluding is only meaningful as a direct argument of a Cross_Product",
			selector.Sel.Name, location))
		return
	}
	if recorder_register_packages_for_analysis_is_axis_constructor(selector.Sel.Name) {
		// 🛑 DO NOT replace this framework-panic with a silent return for
		// "tolerance" of stray axis constructors. A bare
		// Sometimes / Always / Distinct_Boundary call past any Cross_Product
		// consumer is dead — Recorder_X returns a Cross_Axis whose Handle
		// is acquired but never released, and no Assertion_Metadata entry is
		// ever stored, so the assertion contributes nothing at runtime.
		// Silently accepting it lets authors ship apparent assertions that
		// do nothing — same failure mode as Excluding-outside-Cross_Product.
		// Read Test_Register_Framework_Panics_On_Orphan_Axis_Constructor
		// before "fixing" this.
		if input.Legitimate_Axis_Calls[call] {
			return
		}
		position := file_set.Position(call.Lparen)
		location := file_path + ":" + strconv.Itoa(position.Line)
		recorder_framework_panic_unless(r, false, fmt.Sprintf(
			"invariant.%s at %s: axis constructor appears outside any Cross_Product / "+
				"Recorder_Cross_Product call — the returned Cross_Axis is dropped, "+
				"the Handle is never released, and no Assertion_Metadata is registered. "+
				"Pass the axis as a Cross_Product argument, bind it to a name that appears "+
				"as a Cross_Product argument, or use the Is_%s sugar that registers as a "+
				"single-axis assertion on its own",
			selector.Sel.Name, location, strings.TrimPrefix(selector.Sel.Name, "Recorder_")))
		return
	}
	if !recorder_register_packages_for_analysis_is_tracked_kind(selector.Sel.Name) {
		return
	}
	position := file_set.Position(call.Lparen)
	location := file_path + ":" + strconv.Itoa(position.Line)
	message, has_literal_message := ast_extract_string_literal_message(call)
	recorder_framework_panic_unless(r, has_literal_message, fmt.Sprintf(
		"invariant.%s at %s: last argument must be a string literal or `+`-concatenation of string literals — "+
			"the coverage tracker registers assertions by their literal message at AST time, "+
			"so non-literal messages (fmt.Sprintf, variables, call results) can't appear in the never-fired report",
		selector.Sel.Name, location))
	violation, valid := ast_validate_invariant_message(message)
	recorder_framework_panic_unless(r, valid, fmt.Sprintf(
		"invariant.%s at %s: message %q %s",
		selector.Sel.Name, location, message, violation))
	r.Assertions.Store(location, &Assertion_Metadata{
		Kind:    selector.Sel.Name,
		Message: message,
		Site:    location,
	})
}

// Extracts the last argument of call as a string-literal expression — either
// a single string literal or a `+`-concatenation chain whose every leaf is a
// string literal. Returns ok=false when the call has no arguments or its
// last argument can't be folded (e.g., contains a variable or a call).
func ast_extract_string_literal_message(call *ast.CallExpr) (message string, ok bool) {

	if len(call.Args) == 0 {
		return "", false
	}
	return ast_fold_string_literal_expression(call.Args[len(call.Args)-1])
}

// Folds an AST expression into its string-literal value. Accepts a single
// *ast.BasicLit of token.STRING (interpreted or raw), a *ast.BinaryExpr
// with Op == token.ADD whose operands recursively fold, or a *ast.ParenExpr
// whose child folds. Anything else returns ok=false — the registrar
// translates that into the framework-precondition panic that names the
// literal-or-concatenation requirement. strconv.Unquote handles escape
// sequences and raw-string backticks correctly; the legacy Value[1:len-1]
// trick was wrong for both.
func ast_fold_string_literal_expression(expression ast.Expr) (folded string, ok bool) {

	switch node := expression.(type) {
	case *ast.BasicLit:
		if node.Kind != token.STRING {
			return "", false
		}
		unquoted, err := strconv.Unquote(node.Value)
		if err != nil {
			return "", false
		}
		return unquoted, true
	case *ast.BinaryExpr:
		if node.Op != token.ADD {
			return "", false
		}
		left, left_ok := ast_fold_string_literal_expression(node.X)
		if !left_ok {
			return "", false
		}
		right, right_ok := ast_fold_string_literal_expression(node.Y)
		if !right_ok {
			return "", false
		}
		return left + right, true
	case *ast.ParenExpr:
		return ast_fold_string_literal_expression(node.X)
	}
	return "", false
}

// Matches a heuristic negative token as a whole ASCII word — `not`, `never`,
// the common modal contractions, and the `fail`/`broken`/`invalid`/`illegal`
// family. `(?i)` makes the match case-insensitive; `\b` is RE2's ASCII
// word boundary so `Notebook` doesn't trip `not` and `negative` doesn't
// trip `no`. Apostrophe is a non-word char, so `\bcan't\b` matches the
// 5-char token between outer word boundaries even though there's an inner
// boundary at the apostrophe.
//
//	the rest of the constants in this file.
//
//nolint:gochecknoglobals — read-only compiled regex; package-level matches
var banned_negative_word_pattern = regexp.MustCompile(
	`(?i)\b(not|never|no|none|cannot|can't|don't|doesn't|didn't|isn't|` +
		`wasn't|aren't|weren't|won't|shouldn't|wouldn't|couldn't|hasn't|haven't|hadn't|` +
		`fail|fails|failed|broken|invalid|illegal)\b`)

// Checks an extracted invariant message against the positive-statement
// rules the registrar enforces: starts with an uppercase letter, at least
// 3 whitespace-separated words, at least 10 runes long, contains no
// heuristic negative tokens. The word and length floors keep authors from
// satisfying the capital-letter rule with empty stubs like "OK" or "A B
// C" that carry no claim about what holds. Returns ok=false with a
// violation description naming the specific rule (and the offending
// token, for the negative-word rule) so the framework panic guides the
// author toward a rewrite.
func ast_validate_invariant_message(message string) (violation string, ok bool) {

	if message == "" {
		return "must not be empty", false
	}
	first, _ := utf8.DecodeRuneInString(message)
	if !unicode.IsUpper(first) {
		return "must start with an uppercase letter", false
	}
	if word_count := len(strings.Fields(message)); word_count < 3 {
		return fmt.Sprintf("must contain at least 3 words (got %d)", word_count), false
	}
	if rune_count := utf8.RuneCountInString(message); rune_count < 10 {
		return fmt.Sprintf("must be at least 10 characters long (got %d)", rune_count), false
	}
	if found := banned_negative_word_pattern.FindString(message); found != "" {
		return fmt.Sprintf(
			"should exclude the negative token %q — state the invariant as the property that holds, not the failure mode",
			found), false
	}
	return "", true
}

// Reports whether selector_name is one of the assertion primitive names worth
// tracking. Matches both the composition-tier convenience names and the
// pure-tier explicit names (Recorder_Is_Always, Recorder_Is_Sometimes, …).
func recorder_register_packages_for_analysis_is_tracked_kind(selector_name string) (tracked bool) {

	switch selector_name {
	case "Is_Sometimes", "Is_Always", "Is_Always_Nil_Error",
		"Recorder_Is_Sometimes", "Recorder_Is_Always", "Recorder_Is_Always_Nil_Error":
		return true
	}
	return false
}

// Pre-registers a tracker entry per tuple in the cross-product of the
// Cross_Product call's axis arguments, after filtering tuples forbidden by
// any Excluding clause. Axis args may be inline composable calls
// (`invariant.Sometimes(…)`) or identifiers bound to such calls in the
// enclosing function (`enabled := invariant.Sometimes(…); … enabled, …`).
// If any axis arg can't be resolved, the entire site is skipped — same
// behavior as today for the inline-only path.
//
// Excluding clauses are parsed separately. A clause whose cells reference
// an axis not present in this call, or whose Bucket_X selector mismatches
// the resolved axis kind, framework-panics at registration via
// recorder_framework_panic_unless. So does a configuration that filters
// every tuple in the product (zero entries to register would leave the
// site invisible to the never-fired report).
func recorder_register_packages_for_analysis_register_cross(
	r *Recorder,
	file_set *token.FileSet,
	file_path string,
	call *ast.CallExpr,
	bindings map[string]*ast.CallExpr,
) {

	selector := call.Fun.(*ast.SelectorExpr)
	argument_offset := 0
	if strings.HasPrefix(selector.Sel.Name, "Recorder_") {
		argument_offset = 1
	}
	position := file_set.Position(call.Lparen)
	site := file_path + ":" + strconv.Itoa(position.Line)

	axis_calls := []*ast.CallExpr{}
	constraint_clauses := []*ast.CallExpr{}
	for i := argument_offset; i < len(call.Args); i++ {
		resolved, ok := ast_resolve_to_invariant_call(call.Args[i], bindings)
		if !ok {
			return
		}
		clause_kind, _ := ast_invariant_selector_kind(resolved)
		if clause_kind == "Excluding" {
			constraint_clauses = append(constraint_clauses, resolved)
			continue
		}
		axis_calls = append(axis_calls, resolved)
	}
	if len(axis_calls) == 0 {
		return
	}

	bucket_sets := make([][]string, 0, len(axis_calls))
	composable_kinds := make([]string, 0, len(axis_calls))
	value_expressions := make([]string, 0, len(axis_calls))
	messages := make([]string, 0, len(axis_calls))
	for _, axis_call := range axis_calls {
		axis_position := file_set.Position(axis_call.Lparen)
		axis_location := file_path + ":" + strconv.Itoa(axis_position.Line)
		buckets, value_expression, message, kind, parse_ok := ast_composable_buckets(r, axis_location, axis_call)
		if !parse_ok {
			return
		}
		bucket_sets = append(bucket_sets, buckets)
		composable_kinds = append(composable_kinds, kind)
		value_expressions = append(value_expressions, value_expression)
		messages = append(messages, message)
	}

	excluding_cells := [][]ast_constraint_cell{}
	for _, clause := range constraint_clauses {
		clause_location := file_path + ":" + strconv.Itoa(file_set.Position(clause.Lparen).Line)
		cells := ast_resolve_constraint_cells(r, clause_location, "Excluding", clause, axis_calls, composable_kinds, bindings)
		// An Excluding clause whose cells all name the SAME bucket on a Boundary
		// axis defeats the point of the boundary: a boundary's purpose is that
		// BOTH endpoints are observable in the verified space, and
		// Excluding(Bucket_Hi(b)) — or any number of redundant duplicates of it
		// — declares one endpoint unreachable. The honest assertion is a
		// tighter Lo/Hi (so both endpoints are reachable) or an Always against
		// the actual bound. The check covers single-cell clauses and
		// duplicate-cell clauses alike: regardless of the cell count, if every
		// cell names the same (axis, bucket) on a Distinct_Boundary, the shape
		// is banned.
		if len(cells) >= 1 {
			first := cells[0]
			all_same_boundary_endpoint := composable_kinds[first.Axis_Position] == "Distinct_Boundary"
			if all_same_boundary_endpoint {
				for _, cell := range cells[1:] {
					if cell.Axis_Position != first.Axis_Position {
						all_same_boundary_endpoint = false
						break
					}
					if cell.Bucket_Index != first.Bucket_Index {
						all_same_boundary_endpoint = false
						break
					}
				}
			}
			if all_same_boundary_endpoint {
				recorder_framework_panic_unless(r, false, fmt.Sprintf(
					"invariant.Excluding at %s: Excluding naming only a single Boundary endpoint "+
						"(Bucket_Lo / Bucket_Hi), regardless of cell count, is banned — a boundary "+
						"requires both endpoints to be observable. Either pick tighter Lo/Hi so both "+
						"endpoints are reachable, or replace the Boundary with an Always asserting "+
						"the bound directly.",
					clause_location))
			}
		}
		excluding_cells = append(excluding_cells, cells)
	}

	recorder_register_cross_product(&recorder_register_cross_product_input{
		Recorder:          r,
		Site:              site,
		Bucket_Sets:       bucket_sets,
		Composable_Kinds:  composable_kinds,
		Value_Expressions: value_expressions,
		Messages:          messages,
		Excluding_Cells:   excluding_cells,
	})
}

// ast_constraint_cell is the AST-time resolution of one cell in an Excluding
// clause — a (position-in-axes, bucket-index) pair. Position binds to the
// axis order in the parent Cross_Product call.
type ast_constraint_cell struct {
	Axis_Position int
	Bucket_Index  int
}

// ast_resolve_to_invariant_call resolves arg to an invariant.X(...) CallExpr.
// Inline calls pass through; identifier args are looked up in bindings. Any
// other shape (function call result, struct field access, etc.) returns ok=false
// — matching today's "skip the whole site" behavior for unparseable args.
func ast_resolve_to_invariant_call(arg ast.Expr, bindings map[string]*ast.CallExpr) (call *ast.CallExpr, ok bool) {

	if cexpr, is_call := arg.(*ast.CallExpr); is_call {
		return cexpr, true
	}
	ident, is_ident := arg.(*ast.Ident)
	if !is_ident {
		return nil, false
	}
	bound, has := bindings[ident.Name]
	if !has {
		return nil, false
	}
	return bound, true
}

// ast_invariant_selector_kind returns the kind (helper name with Recorder_
// stripped) of an invariant.X CallExpr — "Sometimes" for invariant.Sometimes,
// invariant.Recorder_Sometimes, etc. ok=false when the call isn't of the
// invariant.X shape.
func ast_invariant_selector_kind(call *ast.CallExpr) (kind string, ok bool) {

	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return "", false
	}
	identifier, is_identifier := selector.X.(*ast.Ident)
	if !is_identifier {
		return "", false
	}
	if identifier.Name != "invariant" {
		return "", false
	}
	return strings.TrimPrefix(selector.Sel.Name, "Recorder_"), true
}

// ast_resolve_constraint_cells walks the Excluding clause and turns each
// Bucket_X(axis_var) cell into a (axis_position, bucket_index) pair. The
// clause's first argument is the message string and is skipped by the cell
// walk. Framework-panics on every silence-point case: malformed cell shape,
// foreign axis reference, Bucket_X kind that doesn't match the axis kind.
//
// 🛑 DO NOT downgrade the framework-panic paths in this function to silent
// returns. Each one corresponds to a Test_Register_Framework_Panics_On_*
// case that exists precisely to keep this loud — silently disabling a
// Cross_Product site is the entire failure mode the constraint feature was
// designed to prevent. Read the failing test before "tolerating" anything
// here.
func ast_resolve_constraint_cells(
	r *Recorder,
	clause_location string,
	clause_kind string,
	clause *ast.CallExpr,
	axis_calls []*ast.CallExpr,
	composable_kinds []string,
	bindings map[string]*ast.CallExpr,
) (cells []ast_constraint_cell) {

	recorder_framework_panic_unless(r, len(clause.Args) >= 1, fmt.Sprintf(
		"invariant.%s at %s: clause has no arguments; the first argument must be the message "+
			"string literal followed by Bucket_X(axis) cells",
		clause_kind, clause_location))
	message, message_is_literal := ast_fold_string_literal_expression(clause.Args[0])
	recorder_framework_panic_unless(r, message_is_literal, fmt.Sprintf(
		"invariant.%s at %s: the first argument must be a string literal (or `+`-concatenation "+
			"of string literals) — non-literal messages (fmt.Sprintf, variables, call results) "+
			"can't appear in the never-fired report",
		clause_kind, clause_location))
	violation, valid := ast_validate_invariant_message(message)
	recorder_framework_panic_unless(r, valid, fmt.Sprintf(
		"invariant.%s at %s: message %q %s",
		clause_kind, clause_location, message, violation))
	for _, arg := range clause.Args[1:] {
		cell_call, is_call := arg.(*ast.CallExpr)
		recorder_framework_panic_unless(r, is_call, fmt.Sprintf(
			"invariant.%s at %s: each argument must be Bucket_False(axis) / Bucket_True(axis) / "+
				"Bucket_Lo(axis) / Bucket_Hi(axis); got a non-call expression",
			clause_kind, clause_location))
		cell_selector_name, kind_ok := ast_invariant_selector_kind(cell_call)
		recorder_framework_panic_unless(r, kind_ok, fmt.Sprintf(
			"invariant.%s at %s: each argument must be invariant.Bucket_X(axis); the cell call is not "+
				"an invariant.X invocation",
			clause_kind, clause_location))
		recorder_framework_panic_unless(r,
			cell_selector_name == "Bucket_False" || cell_selector_name == "Bucket_True" ||
				cell_selector_name == "Bucket_Lo" || cell_selector_name == "Bucket_Hi",
			fmt.Sprintf(
				"invariant.%s at %s: each argument must use Bucket_False / Bucket_True / Bucket_Lo / "+
					"Bucket_Hi; got %q",
				clause_kind, clause_location, cell_selector_name))
		recorder_framework_panic_unless(r, len(cell_call.Args) == 1, fmt.Sprintf(
			"invariant.%s at %s: %s takes exactly one axis argument; got %d",
			clause_kind, clause_location, cell_selector_name, len(cell_call.Args)))

		receiver_ident, is_ident := cell_call.Args[0].(*ast.Ident)
		recorder_framework_panic_unless(r, is_ident, fmt.Sprintf(
			"invariant.%s at %s: %s argument must be a bare axis identifier (e.g., enabled); "+
				"got a more complex expression — bind the axis to a local variable first",
			clause_kind, clause_location, cell_selector_name))

		bound_call, bound_ok := bindings[receiver_ident.Name]
		recorder_framework_panic_unless(r, bound_ok, fmt.Sprintf(
			"invariant.%s at %s: %s(%s) references an identifier with no axis binding in scope",
			clause_kind, clause_location, cell_selector_name, receiver_ident.Name))

		axis_position := -1
		for i, axis_call := range axis_calls {
			if axis_call == bound_call {
				axis_position = i
				break
			}
		}
		recorder_framework_panic_unless(r, axis_position >= 0, fmt.Sprintf(
			"invariant.%s at %s: %s(%s) references an axis variable not present among the parent "+
				"Cross_Product axes (foreign-axis reference)",
			clause_kind, clause_location, cell_selector_name, receiver_ident.Name))

		axis_kind := composable_kinds[axis_position]
		bucket_index := ast_bucket_index_for_selector(cell_selector_name, axis_kind)
		recorder_framework_panic_unless(r, bucket_index >= 0, fmt.Sprintf(
			"invariant.%s at %s: %s does not apply to a %s axis (use Bucket_False / Bucket_True for "+
				"Sometimes, Bucket_Lo / Bucket_Hi for Boundary)",
			clause_kind, clause_location, cell_selector_name, axis_kind))

		cells = append(cells, ast_constraint_cell{
			Axis_Position: axis_position,
			Bucket_Index:  bucket_index,
		})
	}
	return cells
}

// ast_bucket_index_for_selector maps a Bucket_X selector name to its bucket
// index given the axis kind. Returns -1 when the selector doesn't apply to
// the axis kind (Bucket_Lo on a Sometimes axis, etc.); the caller turns -1
// into a framework-precondition violation.
func ast_bucket_index_for_selector(selector_name string, axis_kind string) (bucket_index int) {

	switch axis_kind {
	case "Sometimes":
		switch selector_name {
		case "Bucket_False":
			return 0
		case "Bucket_True":
			return 1
		}
	case "Distinct_Boundary":
		switch selector_name {
		case "Bucket_Lo":
			return 0
		case "Bucket_Hi":
			return 1
		}
	}
	return -1
}

// Input for recorder_register_cross_product. The struct exists because
// check_input_struct flags multi-same-type params.
//
// Excluding_Cells carries the AST-resolved constraint cells for the site.
// Each inner slice is one clause; cells inside a clause are conjuncted (a
// tuple matches the clause only when every cell matches). Multiple clauses
// are disjunctive: each Excluding clause forbids an independent cell.
type recorder_register_cross_product_input struct {
	Recorder          *Recorder
	Site              string
	Bucket_Sets       [][]string
	Composable_Kinds  []string
	Value_Expressions []string
	Messages          []string
	Excluding_Cells   [][]ast_constraint_cell
}

// Enumerates the cross-product of bucket_sets and stores one tracker entry
// per tuple admitted by the Excluding filter. When the filter rejects every
// tuple, framework-panics: a Cross_Product whose every cell is unreachable
// has nothing to verify, which is almost always a bug in the constraints
// rather than a genuine site.
//
// 🛑 DO NOT replace the "zero admitted tuples" framework-panic with a
// silent return. A site that registers nothing is invisible to the
// never-fired report and the runtime check has nothing to compare against
// — the entire point of constraint clauses (loud failure on misuse) is
// undermined. Read Test_Register_Framework_Panics_On_All_Cells_Filtered.
func recorder_register_cross_product(input *recorder_register_cross_product_input) {

	indices := make([]int, len(input.Bucket_Sets))
	reachable_buckets := make([]map[int]bool, len(input.Bucket_Sets))
	for i := range reachable_buckets {
		reachable_buckets[i] = map[int]bool{}
	}
	registered := 0
	for range Game_Loop() {
		if recorder_register_cross_product_tuple(indices, input) {
			registered++
			for axis_index, bucket_index := range indices {
				reachable_buckets[axis_index][bucket_index] = true
			}
		}
		if !ast_increment_tuple(indices, input.Bucket_Sets) {
			break
		}
	}
	recorder_framework_panic_unless(input.Recorder, registered > 0, fmt.Sprintf(
		"Cross_Product at %s: every tuple in the product was filtered by Excluding "+
			"constraints — the site has no cells left to verify. Either remove a constraint or "+
			"remove the Cross_Product call.", input.Site))
	recorder_register_cross_product_axes(input, reachable_buckets)
}

// Registers one tracker entry per axis in the Cross_Product call (in addition
// to the per-tuple entries). Each axis is an independent property; counting
// only tuples would collapse a five-axis Always Cross_Product into one
// "tested" credit. Boundary axes decompose into four sub-properties: two
// Always (X >= Lo and X <= Hi — the bound enforcement) and two Sometimes
// (X == Lo and X == Hi — the endpoint observations). When an Excluding
// clause makes an axis bucket unreachable, the corresponding sub-property
// is either omitted (Boundary lo_hit/hi_hit) or registered as Cross_Axis_Always
// on the reachable branch (Sometimes axes — demanding the unreachable branch
// would be a permanent false-positive gap).
func recorder_register_cross_product_axes(
	input *recorder_register_cross_product_input,
	reachable_buckets []map[int]bool,
) {
	for axis_index, kind := range input.Composable_Kinds {
		base_key := input.Site + ":axis=" + strconv.Itoa(axis_index)
		bucket := Cross_Bucket{
			Composable_Kind:  kind,
			Value_Expression: input.Value_Expressions[axis_index],
			Axis_Message:     input.Messages[axis_index],
		}
		switch kind {
		case "Always":
			axis_bucket := bucket
			axis_bucket.Bucket_Name = "true"
			input.Recorder.Assertions.Store(base_key, &Assertion_Metadata{
				Kind:          "Cross_Axis_Always",
				Site:          input.Site,
				Cross_Buckets: []Cross_Bucket{axis_bucket},
			})
		case "Sometimes":
			true_reachable := reachable_buckets[axis_index][1]
			false_reachable := reachable_buckets[axis_index][0]
			axis_bucket := bucket
			axis_kind := "Cross_Axis_Sometimes"
			if true_reachable {
				if !false_reachable {
					axis_kind = "Cross_Axis_Always"
					axis_bucket.Bucket_Name = "true"
				}
			}
			if false_reachable {
				if !true_reachable {
					axis_kind = "Cross_Axis_Always"
					axis_bucket.Bucket_Name = "false"
				}
			}
			input.Recorder.Assertions.Store(base_key, &Assertion_Metadata{
				Kind:          axis_kind,
				Site:          input.Site,
				Cross_Buckets: []Cross_Bucket{axis_bucket},
			})
		case "Distinct_Boundary":
			// lo_ge and hi_le: bound enforcement, always registered (held
			// whenever the axis fires at all; Boundary would have panicked
			// on out-of-range).
			lo_ge_bucket := bucket
			lo_ge_bucket.Bucket_Name = "lo_ge"
			input.Recorder.Assertions.Store(base_key+":property=lo_ge", &Assertion_Metadata{
				Kind:          "Cross_Axis_Always",
				Site:          input.Site,
				Cross_Buckets: []Cross_Bucket{lo_ge_bucket},
			})
			hi_le_bucket := bucket
			hi_le_bucket.Bucket_Name = "hi_le"
			input.Recorder.Assertions.Store(base_key+":property=hi_le", &Assertion_Metadata{
				Kind:          "Cross_Axis_Always",
				Site:          input.Site,
				Cross_Buckets: []Cross_Bucket{hi_le_bucket},
			})
			// lo_hit and hi_hit: endpoint observation, registered only when
			// the endpoint is reachable. Excluding that forbids an endpoint
			// suppresses the corresponding hit-property. Sometimes-shape
			// requires both branches (true = at this endpoint, false =
			// elsewhere); when the opposite endpoint is unreachable no
			// observation can falsify the property, so the registration
			// degenerates to Cross_Axis_Always (single-branch coverage).
			lo_reachable := reachable_buckets[axis_index][0]
			hi_reachable := reachable_buckets[axis_index][1]
			if lo_reachable {
				lo_hit_bucket := bucket
				lo_hit_bucket.Bucket_Name = "lo_hit"
				lo_hit_kind := "Cross_Axis_Sometimes"
				if !hi_reachable {
					lo_hit_kind = "Cross_Axis_Always"
				}
				input.Recorder.Assertions.Store(base_key+":property=lo_hit", &Assertion_Metadata{
					Kind:          lo_hit_kind,
					Site:          input.Site,
					Cross_Buckets: []Cross_Bucket{lo_hit_bucket},
				})
			}
			if hi_reachable {
				hi_hit_bucket := bucket
				hi_hit_bucket.Bucket_Name = "hi_hit"
				hi_hit_kind := "Cross_Axis_Sometimes"
				if !lo_reachable {
					hi_hit_kind = "Cross_Axis_Always"
				}
				input.Recorder.Assertions.Store(base_key+":property=hi_hit", &Assertion_Metadata{
					Kind:          hi_hit_kind,
					Site:          input.Site,
					Cross_Buckets: []Cross_Bucket{hi_hit_bucket},
				})
			}
		}
	}
}

// Stores one tracker entry for the current tuple when admitted by the
// constraint filter. Returns admitted=true if the tuple was stored. The
// long-promised "constraint-filter early-return" comment finally earns its
// keep here.
func recorder_register_cross_product_tuple(indices []int, input *recorder_register_cross_product_input) (admitted bool) {

	if !ast_tuple_admitted_by_constraints(indices, input.Excluding_Cells) {
		return false
	}
	var key_builder strings.Builder
	key_builder.WriteString(input.Site)
	key_builder.WriteString(":tuple=(")
	buckets := make([]Cross_Bucket, len(input.Bucket_Sets))
	for i, index := range indices {
		if i > 0 {
			key_builder.WriteByte(',')
		}
		key_builder.WriteString(strconv.Itoa(index))
		buckets[i] = Cross_Bucket{
			Composable_Kind:  input.Composable_Kinds[i],
			Value_Expression: input.Value_Expressions[i],
			Bucket_Name:      input.Bucket_Sets[i][index],
			Axis_Message:     input.Messages[i],
		}
	}
	key_builder.WriteByte(')')
	tuple_indices := make([]int, len(indices))
	copy(tuple_indices, indices)
	input.Recorder.Assertions.Store(key_builder.String(), &Assertion_Metadata{
		Kind:          "Cross",
		Site:          input.Site,
		Cross_Buckets: buckets,
		Tuple_Indices: tuple_indices,
	})
	return true
}

// ast_tuple_admitted_by_constraints reports whether the indices tuple
// survives the Excluding filter. Conjunction within a clause; disjunction
// between clauses — any Excluding clause matching → forbidden.
func ast_tuple_admitted_by_constraints(indices []int, excluding_cells [][]ast_constraint_cell) (admitted bool) {

	for _, clause := range excluding_cells {
		if ast_clause_matches_tuple(clause, indices) {
			return false
		}
	}
	return true
}

// ast_clause_matches_tuple reports whether every cell in the clause matches
// the tuple at its named axis_position. Cells whose axis_position falls
// outside indices are treated as non-matching (defensive against malformed
// input that AST resolution should have caught).
func ast_clause_matches_tuple(cells []ast_constraint_cell, indices []int) (matched bool) {

	for _, cell := range cells {
		if cell.Axis_Position < 0 || cell.Axis_Position >= len(indices) {
			return false
		}
		if indices[cell.Axis_Position] != cell.Bucket_Index {
			return false
		}
	}
	return true
}

// Treats a single-axis composable Is_* call (today: Is_Boundary) as a
// degenerate one-element Cross_Product and registers per-bucket entries at
// the call site.
func recorder_register_packages_for_analysis_register_of(
	r *Recorder,
	file_set *token.FileSet,
	file_path string,
	call *ast.CallExpr,
) {

	position := file_set.Position(call.Lparen)
	site := file_path + ":" + strconv.Itoa(position.Line)
	buckets, value_expression, message_literal, kind, ok := ast_composable_buckets(r, site, call)
	if !ok {
		return
	}
	for index, bucket_name := range buckets {
		key := site + ":tuple=(" + strconv.Itoa(index) + ")"
		r.Assertions.Store(key, &Assertion_Metadata{
			Kind: "Cross",
			Site: site,
			Cross_Buckets: []Cross_Bucket{{
				Composable_Kind:  kind,
				Value_Expression: value_expression,
				Bucket_Name:      bucket_name,
				Axis_Message:     message_literal,
			}},
			Tuple_Indices: []int{index},
		})
	}
}

// Extracts the bucket-name list and helper kind for one composable helper
// call. Also returns the source-text rendering of the value expression
// (e.g. "input.Enabled" or "subject") and the literal message from the axis
// call so coverage reports can show which variable each axis was tracking
// and the author's description of what the axis means. Returns ok=false if
// the call isn't a known composable shape. When ok=true, the message is
// validated against the positive-invariant rules — a violation, or a
// Message field that exists but can't be folded to a string literal, is
// a framework-precondition panic via r so the registration site fails
// loudly rather than silently dropping the message.
func ast_composable_buckets(r *Recorder, location string, call *ast.CallExpr) (buckets []string, value_expression string, message string, kind string, ok bool) {

	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return nil, "", "", "", false
	}
	identifier, is_identifier := selector.X.(*ast.Ident)
	if !is_identifier {
		return nil, "", "", "", false
	}
	if identifier.Name != "invariant" {
		return nil, "", "", "", false
	}
	name := selector.Sel.Name
	argument_offset := 0
	if strings.HasPrefix(name, "Recorder_") {
		argument_offset = 1
	}
	kind = strings.TrimPrefix(name, "Recorder_")
	kind = strings.TrimPrefix(kind, "Is_")
	if len(call.Args) > argument_offset {
		value_expression = ast_extract_axis_value_expression(call, argument_offset)
	}
	message, has_message, foldable := ast_extract_axis_message(call, argument_offset)
	buckets, kind, ok = ast_composable_buckets_dispatch(call, kind, argument_offset)
	if !ok {
		return nil, "", "", "", false
	}
	recorder_framework_panic_unless(r, !has_message || foldable, fmt.Sprintf(
		"invariant.%s axis at %s: message must be a string literal or `+`-concatenation of string literals — "+
			"non-literal expressions (variables, calls, fmt.Sprintf) can't appear in the coverage report",
		name, location))
	if has_message {
		violation, valid := ast_validate_invariant_message(message)
		recorder_framework_panic_unless(r, valid, fmt.Sprintf(
			"invariant.%s axis at %s: message %q %s",
			name, location, message, violation))
	}
	return buckets, value_expression, message, kind, true
}

// Extracts the value-expression source for an axis call. For positional
// shape, returns the source of the first non-recorder arg. For input-struct
// shape (`&X_Input{X: input.Enabled, ...}`), returns the source of the X or
// S field. Falls back to ast_expression_source of the raw arg when neither
// pattern matches.
func ast_extract_axis_value_expression(call *ast.CallExpr, argument_offset int) (source string) {

	composite := ast_extract_struct_argument(call, argument_offset)
	if composite != nil {
		for _, element := range composite.Elts {
			key_value, is_key_value := element.(*ast.KeyValueExpr)
			if !is_key_value {
				continue
			}
			key_ident, is_key_ident := key_value.Key.(*ast.Ident)
			if !is_key_ident {
				continue
			}
			if key_ident.Name != "X" {
				if key_ident.Name != "S" {
					continue
				}
			}
			return ast_expression_source(key_value.Value)
		}
		return "<expr>"
	}
	return ast_expression_source(call.Args[argument_offset])
}

// Extracts the literal message string from an axis call. For positional
// axes (Always, Sometimes), the message is the last argument; for input-
// struct axes (Boundary), the message is the Message field of the
// composite literal at args[argument_offset]. has_message distinguishes
// "no Message supplied" (allowed) from "Message supplied but couldn't be
// folded" (foldable=false → caller panics); foldable=true with
// has_message=false signals a clean absence. The folded message itself
// flows back as the first return — string-literal `+`-concatenation is
// accepted via ast_fold_string_literal_expression.
func ast_extract_axis_message(call *ast.CallExpr, argument_offset int) (message string, has_message bool, foldable bool) {

	composite := ast_extract_struct_argument(call, argument_offset)
	if composite != nil {
		for _, element := range composite.Elts {
			kv, is_kv := element.(*ast.KeyValueExpr)
			if !is_kv {
				continue
			}
			key_ident, is_key_ident := kv.Key.(*ast.Ident)
			if !is_key_ident {
				continue
			}
			if key_ident.Name != "Message" {
				continue
			}
			folded, ok := ast_fold_string_literal_expression(kv.Value)
			return folded, true, ok
		}
		return "", false, true
	}
	if len(call.Args) <= argument_offset {
		return "", false, true
	}
	folded, ok := ast_extract_string_literal_message(call)
	return folded, true, ok
}

// Extracts the input-struct composite literal from call.Args[argument_offset]
// when the axis is invoked as `Helper(&X_Input{...})` (production shape).
// Returns nil when the call uses the positional shape (the AST-analysis
// test fixtures' pseudo-form) or when the arg isn't a composite literal.
func ast_extract_struct_argument(call *ast.CallExpr, argument_offset int) (composite *ast.CompositeLit) {

	if argument_offset >= len(call.Args) {
		return nil
	}
	argument := call.Args[argument_offset]
	if unary, is_unary := argument.(*ast.UnaryExpr); is_unary {
		argument = unary.X
	}
	composite, is_composite := argument.(*ast.CompositeLit)
	if !is_composite {
		return nil
	}
	return composite
}

// Converts an ast.Expr to a dotted source path (Ident or SelectorExpr
// chain). Returns "<expr>" for shapes that aren't pure identifier paths
// (calls, indexed expressions, etc.). Iterative per Tiger Style.
func ast_expression_source(expression ast.Expr) (source string) {

	var segments []string
	current := expression
	for range Game_Loop() {
		selector, is_selector := current.(*ast.SelectorExpr)
		if !is_selector {
			break
		}
		segments = append(segments, selector.Sel.Name)
		current = selector.X
	}
	identifier, is_identifier := current.(*ast.Ident)
	if !is_identifier {
		return "<expr>"
	}
	var b strings.Builder
	b.WriteString(identifier.Name)
	for i := len(segments) - 1; i >= 0; i-- {
		b.WriteByte('.')
		b.WriteString(segments[i])
	}
	return b.String()
}

// Dispatches bucket enumeration by helper kind. Kept separate so
// ast_composable_buckets stays under the line cap. Each axis case accepts
// both the positional shape (used by AST-analysis tests with fake source)
// and the input-struct shape (used by production code, where the framework
// helpers actually live behind a *X_Input struct to satisfy the
// check_input_struct rule).
func ast_composable_buckets_dispatch(
	call *ast.CallExpr,
	kind string,
	argument_offset int,
) (buckets []string, output_kind string, ok bool) {

	switch kind {
	case "Always":
		return []string{"true"}, kind, true
	case "Sometimes":
		return []string{"false", "true"}, kind, true
	case "Distinct_Boundary":
		// Role-based bucket set: the helper identity alone determines
		// cardinality and naming. The Lo / Hi argument expressions are
		// never read at AST time — pre-registration knows only that a
		// Boundary site has two endpoints worth tracking.
		return []string{"Lo", "Hi"}, kind, true
	}
	return nil, "", false
}

// Increments the cross-product index tuple in the manner of a multi-base
// counter. Returns false when the tuple has rolled past the last cell
// (i.e., enumeration is complete).
func ast_increment_tuple(indices []int, bucket_sets [][]string) (continued bool) {

	for i := len(indices) - 1; i >= 0; i-- {
		indices[i]++
		if indices[i] < len(bucket_sets[i]) {
			return true
		}
		indices[i] = 0
	}
	return false
}

// Recorder_Analyze_Assertion_Frequency reports any pre-registered assertion
// that never fired (Frequency == 0) or any Sometimes assertion whose false
// branch never fired (False_Frequency == 0). On any such failure r.Exit(1)
// is invoked.
func Recorder_Analyze_Assertion_Frequency(r *Recorder) {

	recorder_framework_panic_unless(r, r.Is_Test, "Analyze_Assertion_Frequency runs under tests")
	if r.Is_Benchmark {
		return
	}
	if r.Is_Fuzz {
		return
	}
	missed := recorder_analyze_assertion_frequency_collect_never_fired(r)
	never_false := recorder_analyze_assertion_frequency_collect_never_false(r)
	if len(missed) == 0 {
		if len(never_false) == 0 {
			return
		}
	}
	recorder_analyze_assertion_frequency_report(&recorder_analyze_assertion_frequency_report_input{
		Recorder:    r,
		Missed:      missed,
		Never_False: never_false,
	})
	r.Exit(1)
}

// Walks the tracker for entries whose Frequency is still zero — i.e. the
// assertion's true branch was never observed.
func recorder_analyze_assertion_frequency_collect_never_fired(r *Recorder) (missed []string) {

	missed = make([]string, 0)
	r.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*Assertion_Metadata)
		if metadata.Frequency.Load() == 0 {
			missed = append(missed, key.(string))
		}
		return true
	})
	return missed
}

// Walks the tracker for Sometimes entries whose False_Frequency is zero — i.e.
// the assertion's false branch was never observed.
func recorder_analyze_assertion_frequency_collect_never_false(r *Recorder) (never_false []string) {

	never_false = make([]string, 0)
	r.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*Assertion_Metadata)
		if !recorder_metadata_tracks_false_branch(metadata.Kind) {
			return true
		}
		if metadata.False_Frequency.Load() == 0 {
			never_false = append(never_false, key.(string))
		}
		return true
	})
	return never_false
}

// Reports whether a metadata Kind requires the false branch to fire at
// least once for coverage to pass. Bare Sometimes and Is_Sometimes both
// track both branches; Cross_Axis_Sometimes (axis-level coverage inside
// Cross_Product) likewise. Always-kinds — including Cross_Axis_Always
// even when it came from a Sometimes axis with one reachable branch —
// don't have a false branch in the coverage sense.
func recorder_metadata_tracks_false_branch(kind string) (tracks bool) {

	if kind == "Sometimes" {
		return true
	}
	if kind == "X_Sometimes" {
		return true
	}
	return kind == "Cross_Axis_Sometimes"
}

// Input for recorder_analyze_assertion_frequency_report.
type recorder_analyze_assertion_frequency_report_input struct {
	Recorder    *Recorder
	Missed      []string
	Never_False []string
}

// Holds the missed/never-false entries split by category. Cross_By_Site
// groups cross-product tuples that share a call site so the printer can
// render each site once with its axes named and a list of missing tuples
// — collapsing the worst case (single Cross with all buckets missing)
// from N lines to one block.
type recorder_report_partition struct {
	Cross_By_Site      map[string][]*Assertion_Metadata
	Branch_Missed      []*Assertion_Metadata
	Branch_Never_False []*Assertion_Metadata
	Reachability       []*Assertion_Metadata
}

// Prints missed/never-false assertions to input.Recorder.Output. The report
// groups gaps into three categories — cross-product, branch (Is_Sometimes),
// reachability (Is_Always) — each as its own markdown-style section.
// Within Cross-product, tuples that share a site collapse into one block
// listing the axes once and the missing tuples below; sites and tuples sort
// by file path then numeric line / tuple index, never lexicographically.
func recorder_analyze_assertion_frequency_report(input *recorder_analyze_assertion_frequency_report_input) {

	r := input.Recorder
	prefix := ""
	if !r.Full_Location {
		prefix = r.Working_Directory
	}
	partition := recorder_report_partition_entries(&recorder_report_partition_entries_input{
		Recorder:    r,
		Missed:      input.Missed,
		Never_False: input.Never_False,
	})
	total := len(input.Missed) + len(input.Never_False)
	site_set := make(map[string]struct{})
	for site := range partition.Cross_By_Site {
		site_set[site] = struct{}{}
	}
	for _, metadata := range partition.Branch_Missed {
		site_set[metadata.Site] = struct{}{}
	}
	for _, metadata := range partition.Branch_Never_False {
		site_set[metadata.Site] = struct{}{}
	}
	for _, metadata := range partition.Reachability {
		site_set[metadata.Site] = struct{}{}
	}
	// Banner renders once into a local so the top header and the final-line
	// sentinel stay in lockstep. AI agents (and humans skimming long output)
	// read the last line first; printing the gap count at both ends means
	// the verdict survives whether the reader scans top-down or bottom-up.
	banner := fmt.Sprintf("TEST SUITE FAILED: 🚨 %d coverage %s at %d %s 🚨",
		total, recorder_report_pluralize(&recorder_report_pluralize_input{
			N: total, Singular: "gap", Plural: "gaps",
		}),
		len(site_set), recorder_report_pluralize(&recorder_report_pluralize_input{
			N: len(site_set), Singular: "site", Plural: "sites",
		}))
	fmt.Fprintln(r.Output, banner)
	site_totals := recorder_report_site_totals(r)
	if len(partition.Cross_By_Site) > 0 {
		fmt.Fprintln(r.Output)
		recorder_report_print_cross(r.Output, prefix, &partition, site_totals)
	}
	branch_present := len(partition.Branch_Missed) > 0
	if !branch_present {
		branch_present = len(partition.Branch_Never_False) > 0
	}
	if branch_present {
		fmt.Fprintln(r.Output)
		recorder_report_print_branch(r.Output, prefix, &partition)
	}
	if len(partition.Reachability) > 0 {
		fmt.Fprintln(r.Output)
		recorder_report_print_reachability(r.Output, prefix, &partition)
	}

	fmt.Fprintln(r.Output, banner)
}

// Input for recorder_report_partition_entries.
type recorder_report_partition_entries_input struct {
	Recorder    *Recorder
	Missed      []string
	Never_False []string
}

// Splits the missed/never-false identifier lists into the four buckets the
// report renders. Missed identifiers route by Kind: "Cross" → grouped by
// Site; Is_Always-family → Reachability; everything else (Is_Sometimes,
// X_Sometimes, walk-check Sometimes) → Branch_Missed. Never_False entries
// always land in Branch_Never_False — only Sometimes-family assertions
// track a false-frequency, so they're the only ones that can show up here.
func recorder_report_partition_entries(
	input *recorder_report_partition_entries_input,
) (partition recorder_report_partition) {

	partition.Cross_By_Site = make(map[string][]*Assertion_Metadata)
	for _, identifier := range input.Missed {
		value, _ := input.Recorder.Assertions.Load(identifier)
		metadata := value.(*Assertion_Metadata)
		switch {
		case metadata.Kind == "Cross":
			partition.Cross_By_Site[metadata.Site] = append(partition.Cross_By_Site[metadata.Site], metadata)
		case strings.Contains(metadata.Kind, "Always"):
			partition.Reachability = append(partition.Reachability, metadata)
		default:
			partition.Branch_Missed = append(partition.Branch_Missed, metadata)
		}
	}
	for _, identifier := range input.Never_False {
		value, _ := input.Recorder.Assertions.Load(identifier)
		metadata := value.(*Assertion_Metadata)
		partition.Branch_Never_False = append(partition.Branch_Never_False, metadata)
	}
	return partition
}

// Counts how many Cross tracker entries each Site owns. Used as the
// denominator in "N of M tuples never observed" so the report communicates
// gap saturation (all-N-missing reads differently from one-of-N-missing)
// without re-deriving cross-product cardinality from bucket-set sizes.
func recorder_report_site_totals(r *Recorder) (counts map[string]int) {

	counts = make(map[string]int)
	r.Assertions.Range(func(key, value any) (continue_iter bool) {
		metadata := value.(*Assertion_Metadata)
		if metadata.Kind == "Cross" {
			counts[metadata.Site]++
		}
		return true
	})
	return counts
}

// Input for recorder_report_site_less.
type recorder_report_site_less_input struct {
	A string
	B string
}

// Orders Site strings ("file.go:1234") by file path, then by line number
// numerically. The lexicographic default would put "lint.go:10" before
// "lint.go:2" — confusing for a top-to-bottom read of the report.
func recorder_report_site_less(input *recorder_report_site_less_input) (less bool) {

	a_file, a_line := recorder_report_split_site(input.A)
	b_file, b_line := recorder_report_split_site(input.B)
	if a_file != b_file {
		return a_file < b_file
	}
	return a_line < b_line
}

// Input for recorder_report_pluralize.
type recorder_report_pluralize_input struct {
	N        int
	Singular string
	Plural   string
}

func recorder_report_pluralize(input *recorder_report_pluralize_input) (word string) {

	if input.N == 1 {
		return input.Singular
	}
	return input.Plural
}

func recorder_report_split_site(site string) (file string, line int) {

	colon := strings.LastIndexByte(site, ':')
	if colon < 0 {
		return site, 0
	}
	parsed, err := strconv.Atoi(site[colon+1:])
	if err != nil {
		return site, 0
	}
	return site[:colon], parsed
}

// Input for recorder_report_tuple_less.
type recorder_report_tuple_less_input struct {
	A []int
	B []int
}

// Compares two tuple-index slices lexicographically using numeric (not
// string) ordering, so (0,2) sorts before (0,10).
func recorder_report_tuple_less(input *recorder_report_tuple_less_input) (less bool) {

	for i := 0; i < len(input.A) && i < len(input.B); i++ {
		if input.A[i] != input.B[i] {
			return input.A[i] < input.B[i]
		}
	}
	return len(input.A) < len(input.B)
}

// Emits the cross-product section: one block per site, sorted numerically
// by file then line. Each block names the axes once and then lists the
// missing buckets/tuples — collapsing the long worst-case (all buckets of
// a saturated single-axis site missing) into one human-readable line.
func recorder_report_print_cross(
	output io.Writer,
	prefix string,
	partition *recorder_report_partition,
	site_totals map[string]int,
) {

	fmt.Fprintln(output, "# Cross-product gaps")
	sites := make([]string, 0, len(partition.Cross_By_Site))
	for site := range partition.Cross_By_Site {
		sites = append(sites, site)
	}
	sort.Slice(sites, func(i, j int) (less bool) {
		return recorder_report_site_less(&recorder_report_site_less_input{A: sites[i], B: sites[j]})
	})
	for _, site := range sites {
		tuples := partition.Cross_By_Site[site]
		sort.Slice(tuples, func(i, j int) (less bool) {
			return recorder_report_tuple_less(&recorder_report_tuple_less_input{
				A: tuples[i].Tuple_Indices, B: tuples[j].Tuple_Indices,
			})
		})
		fmt.Fprintln(output)
		recorder_report_print_cross_site(&recorder_report_print_cross_site_input{
			Output:     output,
			Prefix:     prefix,
			Site:       site,
			Tuples:     tuples,
			Site_Total: site_totals[site],
		})
	}
}

// Input for recorder_report_print_cross_site.
type recorder_report_print_cross_site_input struct {
	Output     io.Writer
	Prefix     string
	Site       string
	Tuples     []*Assertion_Metadata
	Site_Total int
}

func recorder_report_print_cross_site(input *recorder_report_print_cross_site_input) {

	rendered_site := strings.TrimPrefix(input.Site, input.Prefix)
	axes := input.Tuples[0].Cross_Buckets
	if len(axes) == 1 {
		recorder_report_print_cross_single_axis(&recorder_report_print_cross_single_axis_input{
			Output:        input.Output,
			Rendered_Site: rendered_site,
			Axis:          axes[0],
			Tuples:        input.Tuples,
			Site_Total:    input.Site_Total,
		})
		return
	}
	fmt.Fprintf(input.Output, "%s  Cross_Product — %d of %d tuples never observed\n",
		rendered_site, len(input.Tuples), input.Site_Total)
	fmt.Fprintln(input.Output, "  axes:")
	for _, axis := range axes {
		if axis.Axis_Message != "" {
			fmt.Fprintf(input.Output, "    %s %s  %q\n",
				axis.Composable_Kind, axis.Value_Expression, axis.Axis_Message)
		} else {
			fmt.Fprintf(input.Output, "    %s %s\n", axis.Composable_Kind, axis.Value_Expression)
		}
	}
	fmt.Fprintln(input.Output, "  missing tuples:")
	for _, tuple := range input.Tuples {
		var b strings.Builder
		b.WriteByte('(')
		for i, bucket := range tuple.Cross_Buckets {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(bucket.Bucket_Name)
		}
		b.WriteByte(')')
		fmt.Fprintf(input.Output, "    %s\n", b.String())
	}
}

// Input for recorder_report_print_cross_single_axis.
type recorder_report_print_cross_single_axis_input struct {
	Output        io.Writer
	Rendered_Site string
	Axis          Cross_Bucket
	Tuples        []*Assertion_Metadata
	Site_Total    int
}

func recorder_report_print_cross_single_axis(input *recorder_report_print_cross_single_axis_input) {

	fmt.Fprintf(input.Output, "%s  Cross_Product — %d of %d %s(%s) buckets never observed\n",
		input.Rendered_Site, len(input.Tuples), input.Site_Total,
		input.Axis.Composable_Kind, input.Axis.Value_Expression)
	if input.Axis.Axis_Message != "" {
		fmt.Fprintf(input.Output, "  annotation: %q\n", input.Axis.Axis_Message)
	}
	var b strings.Builder
	for i, tuple := range input.Tuples {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(tuple.Cross_Buckets[0].Bucket_Name)
	}
	fmt.Fprintf(input.Output, "  missing: %s\n", b.String())
}

func recorder_report_print_branch(
	output io.Writer,
	prefix string,
	partition *recorder_report_partition,
) {

	fmt.Fprintln(output, "# Branch gaps")
	sort.Slice(partition.Branch_Missed, func(i, j int) (less bool) {
		return recorder_report_site_less(&recorder_report_site_less_input{
			A: partition.Branch_Missed[i].Site, B: partition.Branch_Missed[j].Site,
		})
	})
	for _, metadata := range partition.Branch_Missed {
		rendered := strings.TrimPrefix(metadata.Site, prefix)
		fmt.Fprintf(output, "%s  %s — true branch unobserved: %q\n",
			rendered, metadata.Kind, metadata.Message)
	}
	sort.Slice(partition.Branch_Never_False, func(i, j int) (less bool) {
		return recorder_report_site_less(&recorder_report_site_less_input{
			A: partition.Branch_Never_False[i].Site, B: partition.Branch_Never_False[j].Site,
		})
	})
	for _, metadata := range partition.Branch_Never_False {
		rendered := strings.TrimPrefix(metadata.Site, prefix)
		fmt.Fprintf(output, "%s  %s — false branch unobserved: %q\n",
			rendered, metadata.Kind, metadata.Message)
	}
}

func recorder_report_print_reachability(
	output io.Writer,
	prefix string,
	partition *recorder_report_partition,
) {

	fmt.Fprintln(output, "# Reachability gaps")
	sort.Slice(partition.Reachability, func(i, j int) (less bool) {
		return recorder_report_site_less(&recorder_report_site_less_input{
			A: partition.Reachability[i].Site, B: partition.Reachability[j].Site,
		})
	})
	for _, metadata := range partition.Reachability {
		rendered := strings.TrimPrefix(metadata.Site, prefix)
		fmt.Fprintf(output, "%s  %s — line never reached: %q\n",
			rendered, metadata.Kind, metadata.Message)
	}
}

// Recorder_Always classifies predicate as a Cross_Axis with a single
// "true" bucket. When predicate is false, fail-fatals immediately with the
// passed message. Used inside Recorder_Cross_Product to make "this predicate
// must hold" composable with other axes.
//
//go:noinline
func Recorder_Always[B ~bool](r *Recorder, predicate B, message string) (record Cross_Axis) {

	if !predicate {
		location := recorder_caller_location(r, 3)
		msg := Assertion_Failure_Message_Prefix + ": " + message
		if location != "" {
			msg = location + " " + msg
		}
		recorder_fail_fatal(r, msg)
	}
	return Cross_Axis{
		Axis_Kind: "Always", Bucket_Index: 0, Bucket_Name: "true", Message: message,
		Handle: recorder_acquire_axis_handle(r),
	}
}

// Recorder_Sometimes classifies predicate as a Cross_Axis with two buckets,
// "false" and "true". Records both branches across the test run; the
// never-fired report surfaces the un-exercised branch annotated with message.
//
//go:noinline
func Recorder_Sometimes[B ~bool](r *Recorder, predicate B, message string) (record Cross_Axis) {

	handle := recorder_acquire_axis_handle(r)
	if predicate {
		return Cross_Axis{
			Axis_Kind: "Sometimes", Bucket_Index: 1, Bucket_Name: "true", Message: message,
			Handle: handle,
		}
	}
	return Cross_Axis{
		Axis_Kind: "Sometimes", Bucket_Index: 0, Bucket_Name: "false", Message: message,
		Handle: handle,
	}
}

// Input for Recorder_Distinct_Boundary.
type Boundary_Input[I Numeric] struct {
	X       I
	Lo      I
	Hi      I
	Message string
}

// Recorder_Distinct_Boundary enforces Lo < Hi (endpoints must be distinct)
// and Lo <= X <= Hi, then classifies X into one of two buckets — "Lo" at
// Bucket_Index 0 when X == Lo, "Hi" at Bucket_Index 1 when X == Hi.
// Interior values pass the bound check but return a Bucket_Index of -1 so
// the Cross_Product key lookup misses (interior observations satisfy the
// bound but contribute no coverage — only endpoints are tracked). Both
// Lo>=Hi and out-of-range X fail-fatal immediately as user-domain
// assertion failures; no Cross_Axis is returned.
//
//go:noinline
func Recorder_Distinct_Boundary[I Numeric](r *Recorder, input *Boundary_Input[I]) (record Cross_Axis) {

	// Float NaN endpoints make the Lo<Hi check meaningless (NaN comparisons
	// are unordered per IEEE-754), so the subsequent ordering check would
	// silently pass and the bucket math would route into the "between"
	// sentinel for every X. Surface this as a user-domain assertion failure
	// up front. The type switch over `any(input.Lo)` is the canonical way
	// to peek at a generic value's concrete numeric type without breaking
	// the Numeric constraint for the integer instantiations.
	switch lo := any(input.Lo).(type) {
	case float32:
		hi := any(input.Hi).(float32)
		if math.IsNaN(float64(lo)) || math.IsNaN(float64(hi)) {
			location := recorder_caller_location(r, 3)
			msg := fmt.Sprintf("%s: Distinct_Boundary requires Lo and Hi to be finite; got Lo=%v, Hi=%v",
				Assertion_Failure_Message_Prefix, input.Lo, input.Hi)
			if location != "" {
				msg = location + " " + msg
			}
			recorder_fail_fatal(r, msg)
		}
	case float64:
		hi := any(input.Hi).(float64)
		if math.IsNaN(lo) || math.IsNaN(hi) {
			location := recorder_caller_location(r, 3)
			msg := fmt.Sprintf("%s: Distinct_Boundary requires Lo and Hi to be finite; got Lo=%v, Hi=%v",
				Assertion_Failure_Message_Prefix, input.Lo, input.Hi)
			if location != "" {
				msg = location + " " + msg
			}
			recorder_fail_fatal(r, msg)
		}
	}
	if input.Lo >= input.Hi {
		location := recorder_caller_location(r, 3)
		msg := fmt.Sprintf("%s: Distinct_Boundary requires Lo < Hi; got Lo=%v, Hi=%v",
			Assertion_Failure_Message_Prefix, input.Lo, input.Hi)
		if location != "" {
			msg = location + " " + msg
		}
		recorder_fail_fatal(r, msg)
	}
	if input.X < input.Lo || input.X > input.Hi {
		location := recorder_caller_location(r, 3)
		msg := fmt.Sprintf("%s: %s. got %v, want [%v, %v]",
			Assertion_Failure_Message_Prefix, input.Message, input.X, input.Lo, input.Hi)
		if location != "" {
			msg = location + " " + msg
		}
		recorder_fail_fatal(r, msg)
	}
	handle := recorder_acquire_axis_handle(r)
	if input.X == input.Lo {
		return Cross_Axis{
			Axis_Kind: "Distinct_Boundary", Bucket_Index: 0, Bucket_Name: "Lo", Message: input.Message,
			Handle: handle,
		}
	}
	if input.X == input.Hi {
		return Cross_Axis{
			Axis_Kind: "Distinct_Boundary", Bucket_Index: 1, Bucket_Name: "Hi", Message: input.Message,
			Handle: handle,
		}
	}
	return Cross_Axis{
		Axis_Kind: "Distinct_Boundary", Bucket_Index: -1, Bucket_Name: "between", Message: input.Message,
		Handle: handle,
	}
}

// Recorder_Is_Distinct_Boundary is the single-axis sugar for Recorder_Distinct_Boundary: the
// Always bound is enforced via the composable helper, then the returned
// Cross_Axis is routed through recorder_cross_product_at so the matching
// per-bucket tracker entry (pre-registered by the AST analyzer via the
// of-path) increments. Interior values pass the Always check but emit the
// -1 sentinel that misses the lookup — no coverage credit for non-endpoints.
//
//go:noinline
func Recorder_Is_Distinct_Boundary[I Numeric](r *Recorder, input *Boundary_Input[I]) {

	recorder_cross_product_at(r, 4, Recorder_Distinct_Boundary(r, input))
}

// Recorder_Is_Always enforces condition under tests and in production. When
// condition is false, emits the failure message and terminates (panic, or
// Exit(1) if Fatal_Failures is set). On true, increments the tracker so
// never-fired assertions surface in the coverage report.
//
//go:noinline
func Recorder_Is_Always(r *Recorder, condition bool, message string) {

	if condition {
		recorder_register_assertion(r, 4)
		return
	}
	location := recorder_caller_location(r, 3)
	msg := Assertion_Failure_Message_Prefix + ": " + message
	if location != "" {
		msg = location + " " + msg
	}
	recorder_fail_fatal(r, msg)
}

// Recorder_Is_Always_Nil_Error is Always for the err == nil property.
//
//go:noinline
func Recorder_Is_Always_Nil_Error(r *Recorder, err error, message string) {

	if err == nil {
		recorder_register_assertion(r, 4)
		return
	}
	location := recorder_caller_location(r, 3)
	msg := fmt.Sprintf("%s: error must be nil. got %q. %s",
		Assertion_Failure_Message_Prefix, err, message)
	if location != "" {
		msg = location + " " + msg
	}
	recorder_fail_fatal(r, msg)
}

// Recorder_Is_Sometimes records that condition was true at least once during the
// test run. Returns the evaluated condition so callers can branch on it.
//
//go:noinline
func Recorder_Is_Sometimes(r *Recorder, condition bool, message string) (firing_true bool) {

	if condition {
		recorder_register_assertion(r, 4)
		return true
	}
	recorder_register_false_assertion(r, 4)
	return false
}

// Recorder_Is_Always_When_Reachable would be Always without tracker
// registration, intended for non-deterministic code paths whose failure
// modes can't be reliably exercised under tests (network I/O, etc.). Kept
// commented out: a library compliant with the pure implementation
// shouldn't need this — every reachable path's failure modes belong in a
// regular Always.

// Recorder_Unimplemented terminates the process with an assertion-failure
// message.
//
//go:noinline
func Recorder_Unimplemented(r *Recorder, message string) {

	recorder_fail_fatal(r, fmt.Sprintf("%s: %s", Assertion_Failure_Message_Prefix, message))
}

// Recorder_Unreachable terminates the process with an assertion-failure
// message.
//
//go:noinline
func Recorder_Unreachable(r *Recorder, message string) {

	recorder_fail_fatal(r, fmt.Sprintf("%s: %s", Assertion_Failure_Message_Prefix, message))
}

// recorder_acquire_axis_handle returns a fresh Axis_Handle pointer for a
// single Cross_Product call's axis. Production-tier recorders wire
// Axis_Handle_Pool with New_Axis_Handle_Pool so Get() always returns a
// non-nil handle (pooled or freshly allocated via Pool.New). Zero-valued
// Recorders (e.g. unit-test fixtures that don't bother configuring the
// pool) hit the fallthrough at the bottom — Pool.Get() returns nil, the
// type assertion fails, and `new(Axis_Handle)` allocates directly.
func recorder_acquire_axis_handle(r *Recorder) (handle *Axis_Handle) {
	if pooled, ok := r.Axis_Handle_Pool.Get().(*Axis_Handle); ok && pooled != nil {
		return pooled
	}
	return new(Axis_Handle)
}

// New_Axis_Handle_Pool returns a sync.Pool wired with a New func that
// allocates Axis_Handle instances. Composition-tier code uses this to
// configure Recorder.Axis_Handle_Pool. Callers should treat the returned
// handles as opaque (see Axis_Handle's doc). The returned Pool must not
// be copied after first use (standard sync.Pool contract).
func New_Axis_Handle_Pool() (pool sync.Pool) {

	return sync.Pool{New: func() (handle any) {
		return new(Axis_Handle)
	}}
}

// recorder_release_axis_handle returns a handle to the pool for reuse. Called
// at the end of recorder_cross_product_at once the constraint check and the
// tracker increment are complete — handles must not be reused while still
// referenced by an in-flight Cross_Product call's Bucket_References.
func recorder_release_axis_handle(r *Recorder, handle *Axis_Handle) {
	if handle == nil {
		return
	}
	r.Axis_Handle_Pool.Put(handle)
}

// Bucket_False names the false bucket of a Sometimes axis. Used as a cell in
// Excluding clauses passed to Cross_Product. Pure packaging — the returned
// Bucket_Reference carries the axis's Handle and the bucket index (0 for
// the false bucket). AST registration verifies that the receiver is actually
// a Sometimes axis; mismatch is a framework precondition violation caught at
// Recorder_Register_Packages_For_Analysis time.
func Bucket_False(axis Cross_Axis) (reference Bucket_Reference) {
	return Bucket_Reference{Handle: axis.Handle, Bucket_Index: 0}
}

// Bucket_True names the true bucket of a Sometimes axis. See Bucket_False.
func Bucket_True(axis Cross_Axis) (reference Bucket_Reference) {
	return Bucket_Reference{Handle: axis.Handle, Bucket_Index: 1}
}

// Bucket_Lo names the lower endpoint of a Boundary axis. See Bucket_False.
func Bucket_Lo(axis Cross_Axis) (reference Bucket_Reference) {
	return Bucket_Reference{Handle: axis.Handle, Bucket_Index: 0}
}

// Bucket_Hi names the upper endpoint of a Boundary axis. See Bucket_False.
func Bucket_Hi(axis Cross_Axis) (reference Bucket_Reference) {
	return Bucket_Reference{Handle: axis.Handle, Bucket_Index: 1}
}

// Excluding declares the named cell impossible. The message names the reason
// the cell cannot fire (a logical contract, a 1:1 input invariant, the
// safety-cap nature of a budget endpoint) — the same role an axis Message
// plays for Always / Sometimes / Distinct_Boundary. The cell itself is the
// conjunction of every listed Bucket_Reference; cells referencing axes not
// in the parent Cross_Product call are framework precondition violations.
// Two effects:
//
//  1. At AST pre-registration, the cell is removed from the never-fired
//     tracker so it never appears as missing coverage.
//  2. At runtime (test and production), an observation matching every cell
//     in any Excluding clause fail-fatals through recorder_fail_fatal; the
//     message surfaces in the violation report so the author sees which
//     impossibility was disproven.
//
// Multiple Excluding clauses inside one Cross_Product are independent
// impossibilities (disjunction at the clause level, conjunction within).
//
// 🛑 Do NOT reintroduce a mirror API ("Solely", "Only", "Admit", etc.) that
// inverts Excluding into a positive admit-set. The mirror form was removed
// deliberately: every Solely(X) is expressible as the equivalent set of
// Excluding(¬X) clauses, and offering both gave authors a second way to
// spell the same constraint. Worse, the existence of an admit-set encouraged
// "shape the test surface to what the fixtures happen to cover" rather than
// "prove which cells are logically impossible" — the former is a coverage
// cheat, the latter is the entire point of constraint clauses.
//
// And: a Cross_Product call carrying many Excluding clauses is itself a
// smell. If you find yourself listing five or more Excluding lines, the
// product is probably wrong (too many axes for one site, or axes that
// aren't actually independent). Decompose the call before adding another
// Excluding to make a fixture pass.
func Excluding(message string, references ...Bucket_Reference) (record Cross_Axis) {
	return Cross_Axis{Axis_Kind: "Excluding", Message: message, Constraint: references}
}

// Recorder_Cross_Product fires one composite Sometimes per bucket-tuple in the
// cross-product of records. Pre-registered tracker entries (one per tuple)
// are seeded at AST-analysis time so a never-fired tuple shows up in the
// coverage report.
//
//go:noinline
func Recorder_Cross_Product(r *Recorder, records ...Cross_Axis) {

	recorder_cross_product_at(r, 4, records...)
}

// Implements Recorder_Cross_Product with an explicit skip so the sugar
// wrappers (Recorder_Is_Distinct_Boundary, …) can share the same machinery and
// stack-walk arithmetic stays in one place.
//
//go:noinline
func recorder_cross_product_at(r *Recorder, skip int, records ...Cross_Axis) {

	defer recorder_release_record_handles(r, records)

	// 🛑 Aliased Handles are a user-code bug, not a framework one. Two
	// axis records sharing a *Axis_Handle means the same axis value was
	// passed twice — silently corrupts Excluding matching because every
	// cell referring to that Handle resolves to whichever record we scan
	// first. Detect at the source. TBD: gate behind r.Is_Test || r.Is_Fuzz
	// for production hot-paths. Overhead is small — a record list of size
	// 8 is 28 pointer compares — but if it shows up in a profile, the gate
	// is the obvious knob.
	for i, a := range records {
		if a.Handle == nil {
			continue
		}
		for j := i + 1; j < len(records); j++ {
			if records[j].Handle != a.Handle {
				continue
			}
			site := ""
			if r.Get_Caller != nil {
				if frame_information, err := r.Get_Caller(skip); err == nil && frame_information.File != "" {
					site = frame_information.File + ":" + strconv.Itoa(frame_information.Line)
				}
			}
			recorder_framework_panic_unless(r, false, fmt.Sprintf(
				"Cross_Product at %s: aliased axis handles — axis %q (#%d) and "+
					"axis %q (#%d) share the same Handle. Each Always / Sometimes / "+
					"Distinct_Boundary call must yield a unique Handle. This can be a "+
					"user-code aliasing bug (an axis variable passed twice) OR a "+
					"framework one (sync.Pool returned a duplicate under concurrent "+
					"goroutines releasing into the same Recorder.Axis_Handle_Pool).",
				site, a.Message, i, records[j].Message, j))
			return
		}
	}

	// 🛑 The constraint check runs ABOVE the test-time gates. Excluding
	// declarations are user-domain invariants — identical in severity to
	// Is_Always — and MUST fail-fatal in production builds, not just under
	// tests. DO NOT push this check below the Is_Test / Is_Benchmark /
	// Is_Fuzz gates: doing so silently disables production enforcement,
	// which is the entire failure mode this feature was built to prevent.
	// If you think this can be moved, read invariant_test.go's
	// Test_Cross_Product_Constraint_Enforces_In_Production_Mode case first.
	if violation, violated := records_check_constraints(r, records); violated {
		recorder_fail_fatal(r, violation)
		return
	}

	if !r.Is_Test {
		return
	}
	if r.Is_Benchmark {
		return
	}
	if r.Is_Fuzz {
		return
	}
	frame_information, err := r.Get_Caller(skip)
	if err != nil {
		return
	}
	if frame_information.File == "" {
		return
	}
	site := frame_information.File + ":" + strconv.Itoa(frame_information.Line)
	recorder_cross_product_increment_axes(r, site, records)
	var key_builder strings.Builder
	key_builder.WriteString(site)
	key_builder.WriteString(":tuple=(")
	first_axis := true
	for _, record := range records {
		if record.Axis_Kind == "Excluding" {
			continue
		}
		if !first_axis {
			key_builder.WriteByte(',')
		}
		first_axis = false
		key_builder.WriteString(strconv.Itoa(record.Bucket_Index))
	}
	key_builder.WriteByte(')')
	value, ok := r.Assertions.Load(key_builder.String())
	if !ok {
		return
	}
	value.(*Assertion_Metadata).Frequency.Add(1)
}

// Increments the per-axis tracker entries registered by
// recorder_register_cross_product_axes. Each axis record contributes to its
// own entries independent of the tuple-level counting:
//   - Always axis: bumps <site>:axis=N Frequency (record reaching here is
//     by definition the true branch — Always panics on false).
//   - Sometimes axis: bumps Frequency on bucket "true", False_Frequency on
//     "false". When AST registration determined only one branch is reachable,
//     the entry was stored as Cross_Axis_Always; the runtime still writes to
//     the same key, and the never-false collector skips Cross_Axis_Always.
//   - Boundary axis: lo_ge and hi_le always tick Frequency (the bound held
//     to reach this point). lo_hit and hi_hit are Sometimes-shape: hit
//     bumps Frequency, miss bumps False_Frequency. When Bucket_Index is the
//     between-sentinel (-1) neither hit-property changes.
func recorder_cross_product_increment_axes(r *Recorder, site string, records []Cross_Axis) {

	axis_index := 0
	for _, record := range records {
		if record.Axis_Kind == "Excluding" {
			continue
		}
		base_key := site + ":axis=" + strconv.Itoa(axis_index)
		switch record.Axis_Kind {
		case "Always":
			recorder_cross_product_increment_axis_entry(r, base_key, true)
		case "Sometimes":
			recorder_cross_product_increment_axis_entry(r, base_key, record.Bucket_Name == "true")
		case "Distinct_Boundary":
			recorder_cross_product_increment_axis_entry(r, base_key+":property=lo_ge", true)
			recorder_cross_product_increment_axis_entry(r, base_key+":property=hi_le", true)
			if record.Bucket_Index != -1 {
				recorder_cross_product_increment_axis_entry(r, base_key+":property=lo_hit", record.Bucket_Index == 0)
				recorder_cross_product_increment_axis_entry(r, base_key+":property=hi_hit", record.Bucket_Index == 1)
			}
		}
		axis_index++
	}
}

// Bumps the tracker entry at key. The increment routes by the entry's
// registered Kind:
//   - Cross_Axis_Always: always bump Frequency. Either it's an Always axis
//     (whose only bucket is "true") or a Sometimes axis whose other branch
//     was constraint-unreachable at registration time (so the observed
//     bucket is necessarily the one reachable branch — constraints panic
//     otherwise before we reach here).
//   - Cross_Axis_Sometimes: bump Frequency on fired_true, else False_Frequency.
//
// Missing entries are silently skipped — registration determined the bucket
// was unreachable so no entry was created (Boundary lo_hit / hi_hit on
// forbidden endpoints).
func recorder_cross_product_increment_axis_entry(r *Recorder, key string, fired_true bool) {

	value, ok := r.Assertions.Load(key)
	if !ok {
		return
	}
	metadata := value.(*Assertion_Metadata)
	if metadata.Kind == "Cross_Axis_Always" {
		metadata.Frequency.Add(1)
		return
	}
	if fired_true {
		metadata.Frequency.Add(1)
		return
	}
	metadata.False_Frequency.Add(1)
}

// records_check_constraints runs the Excluding runtime check over the
// observed records. Returns a non-empty violation message and violated=true
// when an Excluding clause matches the observed tuple. Otherwise returns ok.
//
// Constraint cells reference axes by Handle pointer-identity. A cell whose
// Handle doesn't appear in records is a framework precondition violation
// (foreign-axis reference); AST registration normally catches this at test
// time, but the runtime defends in case production code hits an unregistered
// site.
//
// Boundary-axis interior caveat: a Boundary observation with X strictly
// between Lo and Hi produces Bucket_Index = -1 (the "between" state). The
// tracker silently misses on such observations — the key contains "-1"
// which matches no pre-registered tuple, so no Frequency increment occurs
// and no never-fired report mentions the observation. The constraint check
// preserves that silence: a Bucket_Reference can only name index 0 (Lo) or
// 1 (Hi), so an interior observation never matches an Excluding cell.
func records_check_constraints(r *Recorder, records []Cross_Axis) (violation_message string, violated bool) {

	for _, record := range records {
		if record.Axis_Kind != "Excluding" {
			continue
		}
		for _, cell := range record.Constraint {
			if records_axis_with_handle_exists(cell.Handle, records) {
				continue
			}
			// 🛑 DO NOT replace this framework-panic with a silent skip. An
			// unresolved Handle means the author wrote a constraint cell
			// that references an axis not present in the parent
			// Cross_Product call. Silencing this hides the bug; the panic
			// surfaces it. See the matching
			// Test_Register_Framework_Panics_On_Excluding_Refs_Foreign_Axis
			// guard — both the AST and the runtime catch the same shape.
			recorder_framework_panic_unless(r, false, fmt.Sprintf(
				"Cross_Product %s clause references an axis handle not present in the records — "+
					"either the constraint refers to an axis variable not passed to the parent "+
					"Cross_Product call (foreign-axis reference) or AST registration silently "+
					"skipped its check. Investigate; do not catch.", record.Axis_Kind))
			return "", false
		}
	}

	for _, record := range records {
		if record.Axis_Kind != "Excluding" {
			continue
		}
		if records_constraint_clause_matches(record.Constraint, records) {
			return format_constraint_violation("Excluding", record.Message, record.Constraint, records), true
		}
	}
	return "", false
}

// records_constraint_clause_matches reports whether every cell in the clause
// matches its corresponding axis in records (same Handle, same Bucket_Index).
// An unresolved Handle should already have been caught upstream by
// records_check_constraints; if one slips through (e.g. via a programmatically
// constructed clause), the cell is treated as non-matching.
func records_constraint_clause_matches(cells []Bucket_Reference, records []Cross_Axis) (matched bool) {

	for _, cell := range cells {
		cell_matched := false
		for _, record := range records {
			if record.Axis_Kind == "Excluding" {
				continue
			}
			if record.Handle != cell.Handle {
				continue
			}
			if record.Bucket_Index != cell.Bucket_Index {
				return false
			}
			cell_matched = true
			break
		}
		if !cell_matched {
			return false
		}
	}
	return true
}

// records_axis_with_handle_exists reports whether any axis record in records
// carries the given Handle. Constraint records (Excluding) are skipped.
func records_axis_with_handle_exists(handle *Axis_Handle, records []Cross_Axis) (exists bool) {

	for _, record := range records {
		if record.Axis_Kind == "Excluding" {
			continue
		}
		if record.Handle == handle {
			return true
		}
	}
	return false
}

// format_constraint_violation renders a fail-fatal message that names the
// violated Excluding clause's reason (its message), every axis at its
// observed bucket, and the cells the clause names.
func format_constraint_violation(kind, clause_message string, cells []Bucket_Reference, records []Cross_Axis) (message string) {

	var b strings.Builder
	b.WriteString(Assertion_Failure_Message_Prefix)
	b.WriteString(": Cross_Product ")
	b.WriteString(kind)
	b.WriteString(" constraint violated")
	if clause_message != "" {
		fmt.Fprintf(&b, " — %q", clause_message)
	}
	b.WriteString(" — observed cell:")
	for _, record := range records {
		if record.Axis_Kind == "Excluding" {
			continue
		}
		fmt.Fprintf(&b, "\n  axis %q at bucket %q", record.Message, record.Bucket_Name)
	}
	if cells != nil {
		b.WriteString("\n(matched the Excluding clause naming: ")
		for i, cell := range cells {
			if i > 0 {
				b.WriteString(", ")
			}
			for _, record := range records {
				if record.Handle == cell.Handle {
					fmt.Fprintf(&b, "%q at %q", record.Message, record.Bucket_Name)
					break
				}
			}
		}
		b.WriteString(")")
	}
	return b.String()
}

// recorder_release_record_handles returns every axis-record Handle in records
// to the Recorder's pool. Constraint records (Excluding) carry no Handle so
// they're skipped naturally by the nil guard in recorder_release_axis_handle.
func recorder_release_record_handles(r *Recorder, records []Cross_Axis) {

	for _, record := range records {
		recorder_release_axis_handle(r, record.Handle)
	}
}

// Until returns a bounded sequence useful for safely capping infinite loops.
// On the final iteration the loop emits a runaway warning via the default
// recorder behavior would — here we just panic so the caller's stack survives.
//
//go:noinline
func Until[T Integer_Like](limit T) (sequence iter.Seq[int]) {
	if limit <= 0 {
		panic(fmt.Sprintf("%s: %s", Assertion_Failure_Message_Prefix, "Loop bound is positive"))
	}
	return func(yield func(int) (continue_iteration bool)) {
		integer_limit := int(limit)
		for iteration := range integer_limit {
			if iteration == integer_limit-1 {
				panic(fmt.Sprintf("%s: %s", Assertion_Failure_Message_Prefix, "Runaway loop"))
			}
			if !yield(iteration) {
				return
			}
		}
	}
}

// Game_Loop yields an infinite sequence of increasing integers. Use it when
// `for {}` is banned by CI/CD or code review conventions.
func Game_Loop() (sequence iter.Seq[int]) {
	return func(yield func(int) (continue_iteration bool)) {
		for iteration := 0; true; iteration++ {
			if !yield(iteration) {
				return
			}
		}
	}
}

// Is_Assertion_Failure reports whether a recovered panic value originated from
// this package. The message may be prefixed with a "file:line:col " location
// string, so we search for the sentinel rather than requiring a prefix.
func Is_Assertion_Failure(value any) (yes bool) {

	message, ok := value.(string)
	if !ok {
		return false
	}
	return strings.Contains(message, Assertion_Failure_Message_Prefix)
}

// Has_Whitespace reports whether s contains at least one unicode.IsSpace rune.
// Used inside Always/Sometimes inside Cross_Product to satisfy the
// has_whitespace sub-kind of the string coverage rule.
func Has_Whitespace(s string) (yes bool) {

	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

// Has_Special_Control reports whether s contains at least one unicode.IsControl
// rune (non-printable control characters: \0, \n, \r, etc.). Note that this
// overlaps with Has_Whitespace for \n, \t, and friends; the spec keeps both
// sub-kinds distinct so callers can tighten one without the other.
func Has_Special_Control(s string) (yes bool) {

	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// Has_Null_Byte reports whether s contains a 0x00 byte. Strings crossing
// FFI / syscall boundaries that expect C-style nul-terminated buffers must
// not contain embedded nulls — this predicate is the cheap pre-check.
func Has_Null_Byte(s string) (yes bool) {

	return strings.IndexByte(s, 0) >= 0
}
