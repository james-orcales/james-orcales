// Package invariant records Always and Sometimes assertions and checks, across a
// run, that every event combination they can produce is actually observed.
// Recorder_Dot_Product groups the elements of one assertion site; Impossible
// declares event combinations that must never occur.
package invariant

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
	"math"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// Assertion_Failure_Message_Prefix opens every assertion-failure message.
const Assertion_Failure_Message_Prefix = "🚨 Assertion Failure 🚨: "

// Skip from an element constructor's recorder_site call out to the user's call
// site, for Get_Caller. The composition tier's chain is fixed by //go:noinline
// frames: Get_Caller closure → recorder_site → Recorder_Always/Sometimes →
// invariant.X sugar → user, so the user frame sits 4 above recorder_site.
const recorder_constructor_skip = 4

// Skip from recorder_dot_product_observe's recorder_site call out to the user's
// invariant.Dot_Product call site, for Get_Caller. The tuple tracker is keyed by
// that site, so it must match the static registration position. One deeper than
// recorder_constructor_skip: the observe frame sits between recorder_site and
// Recorder_Dot_Product.
const recorder_dot_product_skip = 5

// Bounds the bundle-flattening loop in recorder_collect_elements: each step
// either advances one argument cursor or pops a finished scope, so an acyclic
// bundle graph finishes far below this. The cap only stops a pathological (e.g.
// self-referential) *_Invariants graph from making the work depend unboundedly
// on input — TigerStyle forbids that.
const max_bundle_expansion_steps = 4096

// Bounds the walk up the directory tree searching for a go.mod, so module
// discovery can't loop unboundedly on a pathological path.
const max_module_search_depth = 256

// Dot_Element_Kind_Always tags an element whose condition must hold on every call.
const Dot_Element_Kind_Always Dot_Element_Kind = 0

// Dot_Element_Kind_Sometimes tags an element whose condition must be observed
// both true and false across the run.
const Dot_Element_Kind_Sometimes Dot_Element_Kind = 1

// Dot_Element_Kind_Impossible tags a declaration that a set of element events
// must never co-occur.
const Dot_Element_Kind_Impossible Dot_Element_Kind = 2

// Dot_Element_Kind_Boundary tags an element asserting Lo <= X <= Hi (with Lo <
// Hi) and tracking which endpoint X lands on.
const Dot_Element_Kind_Boundary Dot_Element_Kind = 3

// Event_Kind_False is the outcome when a condition is false, and the Lo endpoint
// of a Distinct_Boundary (the first bucket).
const Event_Kind_False Event_Kind = 0

// Event_Kind_True is the outcome when a condition is true, and the Hi endpoint of
// a Distinct_Boundary (the second bucket).
const Event_Kind_True Event_Kind = 1

// Event_Kind_Interior is a Distinct_Boundary value strictly inside (Lo, Hi): it
// satisfies the bound but lands on no endpoint, so it earns no coverage.
const Event_Kind_Interior Event_Kind = 2

// Event_Kind_Outside is a Distinct_Boundary value beyond [Lo, Hi] — a deferred
// violation that fails in Recorder_Dot_Product.
const Event_Kind_Outside Event_Kind = 3

// Event_Kind_Bad_Bounds is a Distinct_Boundary whose endpoints aren't distinct
// (Lo >= Hi, or a NaN endpoint) — a deferred violation enforced in Dot_Product.
const Event_Kind_Bad_Bounds Event_Kind = 4

// Assertion_Kind_Always classifies a per-element tracker entry for an Always.
const Assertion_Kind_Always Assertion_Kind = 0

// Assertion_Kind_Sometimes classifies a per-element tracker entry for a Sometimes.
const Assertion_Kind_Sometimes Assertion_Kind = 1

// Assertion_Kind_Boundary classifies a per-element entry for a Distinct_Boundary.
const Assertion_Kind_Boundary Assertion_Kind = 2

// Assertion_Kind_Tuple classifies a per-tuple entry of a Dot_Product's grid.
const Assertion_Kind_Tuple Assertion_Kind = 3

// Recorder accumulates assertion observations for one run and identifies each
// element by its caller Site.
type Recorder struct {
	// File_System reads Go source files during AST analysis. Paths captured by Get_Caller are
	// absolute OS paths; lookups strip the leading "/" before calling fs.ReadFile.
	File_System fs.FS

	// Get_Caller returns the frame information for the caller at the given skip depth. The
	// composition tier wires this to runtime.Callers. Tests can substitute a no-op or a
	// hardcoded frame.
	Get_Caller func(skip int) (file string, line int)

	Assertions sync.Map

	Output io.Writer
	Exit   func(code int)
	Tty    io.Writer

	Is_Test      bool
	Is_Fuzz      bool
	Is_Benchmark bool

	Packages_To_Analyze []string

	// Site_Root is the absolute workspace root (the directory of go.work, else the
	// git root) that Sites are reported relative to. Discovered during
	// registration and stripped on both the static and runtime sides so the
	// file:line rendezvous stays portable. Empty leaves Sites absolute.
	Site_Root string

	// Sugar_Package is the import path of the recorder-less sugar tier (the
	// package defining Always / Sometimes / … as bare functions). When a bundle
	// resolved from that package is descended, its unqualified calls to those
	// primitives are recognised; empty disables that — bare calls elsewhere are
	// not the invariant primitives and must stay unrecognised.
	Sugar_Package string
}

// Assertion_Kind discriminates a coverage tracker entry: a per-element Always,
// Sometimes, or Boundary, or a per-tuple cell of a Dot_Product's grid.
type Assertion_Kind uint8

// Assertion_Metadata is one coverage tracker entry: how often an element's event
// (or a registered tuple) was observed across the run. Seeded at registration,
// incremented at runtime, scanned by the never-fired report.
type Assertion_Metadata struct {
	Frequency       atomic.Int64
	False_Frequency atomic.Int64
	Kind            Assertion_Kind
	Site            string
	Condition       string
	Tuple_Indices   []int
}

// Dot_Element is a discriminated union: an Always/Sometimes observation, or an
// Impossible declaration. Kind selects which fields carry meaning.
type Dot_Element struct {
	Kind  Dot_Element_Kind
	Event Event_Kind
	Site  string

	Impossibles []Dot_Element_Reference
}

// Event is an alias of Dot_Element so the reference builders carry the Event_
// prefix the banned-methods convention requires for Event_True / Event_False.
type Event = Dot_Element

// Dot_Element_Kind discriminates a Dot_Element: Always, Sometimes, Impossible, or
// Distinct_Boundary.
type Dot_Element_Kind uint8

// Event_Kind is an element's observed outcome: the True/False of an Always or
// Sometimes condition, or which bucket a Distinct_Boundary landed in (its Lo/Hi
// endpoints reuse False/True; Interior / Outside / Bad_Bounds are boundary-only).
type Event_Kind uint8

// Dot_Element_Reference names one element's event by its Site — a coordinate an
// Impossible declares forbidden.
type Dot_Element_Reference struct {
	Site       string
	Event_Kind Event_Kind
}

// Reports the caller's "file:line" at skip, or "" when Get_Caller is unset. The
// file is reported relative to recorder.Site_Root (the absolute path Get_Caller
// returns has that prefix stripped), matching the static registration's relative
// Sites; an empty Site_Root leaves it absolute.
//
//go:noinline
func recorder_site(recorder *Recorder, skip int) (site string) {
	if recorder.Get_Caller == nil {
		return ""
	}
	file, line := recorder.Get_Caller(skip)
	if recorder.Site_Root != "" {
		file = strings.TrimPrefix(file, recorder.Site_Root+"/")
	}
	return file + ":" + strconv.Itoa(line)
}

// Recorder_Always builds an element asserting condition holds on every call.
// Like every element producer it never panics on its own: a false observation
// fails only once the element is consumed by Recorder_Dot_Product, so a bare
// Always outside a Dot_Product enforces nothing.
//
//go:noinline
func Recorder_Always[T ~bool](recorder *Recorder, condition T) (dot_element Dot_Element) {
	event := Event_Kind_False
	if condition {
		event = Event_Kind_True
	}
	return Dot_Element{
		Kind:  Dot_Element_Kind_Always,
		Event: event,
		Site:  recorder_site(recorder, recorder_constructor_skip),
	}
}

// Recorder_Sometimes builds an element asserting condition is observed both true
// and false across the run. Like every element producer it never panics on its
// own — coverage is enforced only when the element is consumed by
// Recorder_Dot_Product; a bare Sometimes tracks nothing.
//
//go:noinline
func Recorder_Sometimes[T ~bool](recorder *Recorder, condition T) (dot_element Dot_Element) {
	event := Event_Kind_False
	if condition {
		event = Event_Kind_True
	}
	return Dot_Element{
		Kind:  Dot_Element_Kind_Sometimes,
		Event: event,
		Site:  recorder_site(recorder, recorder_constructor_skip),
	}
}

// Numeric constrains a Distinct_Boundary's value and endpoints to Go's ordered
// numeric kinds.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// Boundary_Input is the value X and its inclusive endpoints Lo, Hi for a
// Distinct_Boundary. Message-less by design, like the other element producers.
type Boundary_Input[I Numeric] struct {
	X  I
	Lo I
	Hi I
}

// Recorder_Distinct_Boundary builds an element asserting Lo < Hi and Lo <= X <=
// Hi, tracking which endpoint X lands on: the Lo bucket when X == Lo, the Hi
// bucket when X == Hi. Interior values satisfy the bound but contribute no
// coverage — only the endpoints are tracked. Like every element producer it
// never panics on its own: a bad bound or out-of-range X fails only once the
// element reaches Recorder_Dot_Product, so a bare Distinct_Boundary outside a
// Dot_Product is an inert no-op (its dead site surfaces in the never-fired
// report rather than enforcing anything).
//
//go:noinline
func Recorder_Distinct_Boundary[I Numeric](
	recorder *Recorder, input *Boundary_Input[I],
) (dot_element Dot_Element) {
	return Dot_Element{
		Kind:  Dot_Element_Kind_Boundary,
		Event: boundary_input_event(input),
		Site:  recorder_site(recorder, recorder_constructor_skip),
	}
}

// Classifies input into its Event: bad bounds (Lo >= Hi or a NaN endpoint) and
// out-of-range X are deferred-violation outcomes enforced in Recorder_Dot_Product;
// an endpoint reuses False (Lo) / True (Hi); anything else is interior.
func boundary_input_event[I Numeric](input *Boundary_Input[I]) (event Event_Kind) {
	if boundary_input_bounds_invalid(input) {
		return Event_Kind_Bad_Bounds
	}
	if input.X < input.Lo {
		return Event_Kind_Outside
	}
	if input.X > input.Hi {
		return Event_Kind_Outside
	}
	if input.X == input.Lo {
		return Event_Kind_False
	}
	if input.X == input.Hi {
		return Event_Kind_True
	}
	return Event_Kind_Interior
}

// Reports whether the endpoints are unusable: Lo >= Hi, or a NaN float endpoint
// (NaN comparisons are unordered, so the ordering check would silently pass and
// every X would route to the interior sentinel).
func boundary_input_bounds_invalid[I Numeric](input *Boundary_Input[I]) (invalid bool) {
	if numeric_is_nan(input.Lo) {
		return true
	}
	if numeric_is_nan(input.Hi) {
		return true
	}
	return input.Lo >= input.Hi
}

// Reports whether a numeric value is a float NaN; integer kinds are never NaN.
func numeric_is_nan[I Numeric](value I) (is_nan bool) {
	switch concrete := any(value).(type) {
	case float32:
		return math.IsNaN(float64(concrete))
	case float64:
		return math.IsNaN(concrete)
	}
	return false
}

// Impossible declares that the referenced element events must never all co-occur on
// the same call. Build references with Event_True / Event_False.
//
// It globs over the axes you do not name: a carve holds only the axes you pass, and
// the coverage analyzer treats every unnamed axis as a wildcard — it forbids, and
// prunes from the demanded cross-product grid, every tuple matching the named events
// for all values of the other axes (see recorder_carve_matches). So
// Impossible(Event_True(a), Event_True(b)) excludes "a and b both true" across every
// combination of the remaining axes; naming a subset never enumerates full tuples.
func Impossible(impossibles ...Dot_Element_Reference) (dot_element Dot_Element) {
	return Dot_Element{Kind: Dot_Element_Kind_Impossible, Impossibles: impossibles}
}

// Event_True references event at its true outcome, for use in Impossible.
func Event_True(event Event) (reference Dot_Element_Reference) {
	return Dot_Element_Reference{
		Site:       event.Site,
		Event_Kind: Event_Kind_True,
	}
}

// Event_False references event at its false outcome, for use in Impossible.
func Event_False(event Event) (reference Dot_Element_Reference) {
	return Dot_Element_Reference{
		Site:       event.Site,
		Event_Kind: Event_Kind_False,
	}
}

// Recorder_Dot_Product enforces the call's elements: an Always observed false
// fails, and an Impossible whose referenced events all occurred fails.
//
//go:noinline
func Recorder_Dot_Product(recorder *Recorder, dot_elements ...Dot_Element) {
	for _, dot_element := range dot_elements {
		dot_element_enforce(dot_element, dot_elements)
	}
	recorder_dot_product_observe(recorder, dot_elements)
}

// Increments the seeded tracker entry for each observed element and the tuple
// entry for the observed combination. No-op outside a plain `go test` run —
// registration seeds nothing under benchmark or fuzz. //go:noinline fixes its
// stack frame so recorder_dot_product_skip stays correct.
//
//go:noinline
func recorder_dot_product_observe(recorder *Recorder, dot_elements []Dot_Element) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	if recorder.Is_Fuzz {
		return
	}
	callsite := recorder_site(recorder, recorder_dot_product_skip)
	for _, dot_element := range dot_elements {
		if dot_element.Kind == Dot_Element_Kind_Impossible {
			continue
		}
		if dot_element.Event == Event_Kind_Interior {
			continue
		}
		recorder_increment_element(&recorder_increment_element_input{
			Recorder:   recorder,
			Callsite:   callsite,
			Site:       dot_element.Site,
			Fired_True: dot_element.Event == Event_Kind_True,
		})
	}
	tuple_key := recorder_tuple_key(callsite, dot_product_tuple(dot_elements))
	recorder_increment(recorder, tuple_key, true)
}

// Returns the observed tuple of bucket indices for the call's non-Impossible
// elements, in order — the runtime counterpart to the static grid's tuples.
func dot_product_tuple(dot_elements []Dot_Element) (tuple []int) {
	tuple = make([]int, 0, len(dot_elements))
	for _, dot_element := range dot_elements {
		if dot_element.Kind == Dot_Element_Kind_Impossible {
			continue
		}
		tuple = append(tuple, dot_element_bucket(dot_element))
	}
	return tuple
}

// Maps an observed element to its bucket index, mirroring the static grid:
// False / Lo = 0, True / Hi = 1, a Boundary interior = -1 (missing the seeded
// grid, so it earns no coverage). An Always has the single bucket 0 — it survived
// enforcement, so it held true, but its grid has only that one cell.
func dot_element_bucket(dot_element Dot_Element) (bucket int) {
	if dot_element.Kind == Dot_Element_Kind_Always {
		return 0
	}
	if dot_element.Event == Event_Kind_Interior {
		return -1
	}
	if dot_element.Event == Event_Kind_True {
		return 1
	}
	return 0
}

// Input for recorder_increment_element.
type recorder_increment_element_input struct {
	Recorder   *Recorder
	Callsite   string
	Site       string
	Fired_True bool
}

// Credits the per-element entry for an observed element. A bundle element is seeded
// under the caller-qualified key (callsite::from=site); an inline element under its
// bare site. Probing the combined key first, then the bare site, credits whichever
// registration seeded — without the runtime classifying the element, and reusing the
// callsite the tuple already rendezvous on. Touches exactly one entry, and no-ops
// when neither is seeded (an unreachable bucket seeds nothing).
func recorder_increment_element(input *recorder_increment_element_input) {
	combined := recorder_element_key(&recorder_element_key_input{
		Callsite: input.Callsite, Site: input.Site,
	})
	if _, seeded := input.Recorder.Assertions.Load(combined); seeded {
		recorder_increment(input.Recorder, combined, input.Fired_True)
		return
	}
	recorder_increment(input.Recorder, input.Site, input.Fired_True)
}

// Bumps the seeded entry at site: Frequency on a true event, False_Frequency on
// false. A missing entry is skipped — registration seeds only reachable buckets.
func recorder_increment(recorder *Recorder, site string, fired_true bool) {
	value, ok := recorder.Assertions.Load(site)
	if !ok {
		return
	}
	metadata := value.(*Assertion_Metadata)
	if fired_true {
		metadata.Frequency.Add(1)
		return
	}
	metadata.False_Frequency.Add(1)
}

// Fails the call when dot_element states an invariant that was violated: an
// Always observed false, a Boundary with a deferred violation, or an Impossible
// whose forbidden combination occurred.
func dot_element_enforce(dot_element Dot_Element, dot_elements []Dot_Element) {
	if dot_element.Kind == Dot_Element_Kind_Always {
		if dot_element.Event == Event_Kind_False {
			panic(Assertion_Failure_Message_Prefix + "an Always element is false")
		}
	}
	if dot_element.Kind == Dot_Element_Kind_Boundary {
		dot_element_enforce_boundary(dot_element)
	}
	if dot_element.Kind != Dot_Element_Kind_Impossible {
		return
	}
	if dot_element_impossible_violated(dot_element, dot_elements) {
		panic(Assertion_Failure_Message_Prefix + "an Impossible combination occurred")
	}
}

// Fails the call when a Boundary carries a deferred violation: X outside
// [Lo, Hi], or bounds that aren't distinct (Lo >= Hi, or a NaN endpoint).
func dot_element_enforce_boundary(dot_element Dot_Element) {
	if dot_element.Event == Event_Kind_Outside {
		panic(Assertion_Failure_Message_Prefix +
			"a Distinct_Boundary value is outside its bounds")
	}
	if dot_element.Event == Event_Kind_Bad_Bounds {
		panic(Assertion_Failure_Message_Prefix + "a Distinct_Boundary requires Lo < Hi")
	}
}

// Reports whether every event the Impossible names was observed this call — the
// forbidden combination occurred in full. An Impossible with no references
// constrains nothing, so it never fires (rather than firing vacuously every call).
func dot_element_impossible_violated(
	impossible Dot_Element, dot_elements []Dot_Element,
) (violated bool) {
	if len(impossible.Impossibles) == 0 {
		return false
	}
	for _, reference := range impossible.Impossibles {
		if !dot_element_reference_observed(reference, dot_elements) {
			return false
		}
	}
	return true
}

// Reports whether some observation element in the call carries the reference's
// Site and was seen at the event the reference names. Impossible elements are
// skipped — they observe nothing.
func dot_element_reference_observed(
	reference Dot_Element_Reference, dot_elements []Dot_Element,
) (observed bool) {
	for _, dot_element := range dot_elements {
		if dot_element.Kind == Dot_Element_Kind_Impossible {
			continue
		}
		if dot_element.Site != reference.Site {
			continue
		}
		if dot_element.Event != reference.Event_Kind {
			continue
		}
		return true
	}
	return false
}

// Recorder_Register_Packages_For_Analysis parses every non-test .go file under
// the given directories and seeds recorder.Assertions with one entry per element
// bucket and one per non-carved tuple of each invariant.Dot_Product call. That
// seeded set is the expected-coverage space the never-fired report scans after
// the run; literal invariant.X selectors and *_Invariants bundles are recognised.
//
// Directories default to recorder.Packages_To_Analyze when none are passed. Each
// is resolved to an absolute path so the registered Site (an absolute file:line)
// matches the absolute path runtime.Callers reports — the rendezvous is exact.
func Recorder_Register_Packages_For_Analysis(recorder *Recorder, directories ...string) {
	if len(directories) > 0 {
		recorder.Packages_To_Analyze = directories
	}
	file_set := token.NewFileSet()
	var files []*ast.File
	module_path := ""
	module_root := ""
	primary_directory := ""
	for _, directory := range recorder.Packages_To_Analyze {
		absolute, absolute_error := filepath.Abs(directory)
		if absolute_error != nil {
			continue
		}
		if primary_directory == "" {
			primary_directory = absolute
		}
		if module_path == "" {
			module_path, module_root = recorder_module(recorder, absolute)
		}
		if recorder.Site_Root == "" {
			recorder.Site_Root = recorder_site_root(recorder, absolute)
		}
		parsed := recorder_parse_directory(&recorder_parse_directory_input{
			File_System: recorder.File_System,
			File_Set:    file_set,
			Directory:   absolute,
			Site_Root:   recorder.Site_Root,
		})
		files = append(files, parsed...)
	}
	index := &bundle_index{
		File_System:       recorder.File_System,
		File_Set:          file_set,
		Module_Path:       module_path,
		Module_Root:       module_root,
		Site_Root:         recorder.Site_Root,
		Sugar_Package:     recorder.Sugar_Package,
		Workspace_Modules: recorder_workspace_modules(recorder, primary_directory),
		Same_Set:          ast_index_functions(files),
		Loaded:            map[string]map[string]indexed_function{},
	}
	for _, file := range files {
		recorder_register_file(recorder, file_set, file, index)
	}
	recorder_check_bundle_dot_product(recorder, file_set, files)
	recorder_check_orphan_axes(recorder, file_set, files)
}

// Walks up from start_directory for the workspace root: the nearest ancestor
// containing a go.work, else (no go.work anywhere up) the nearest containing a
// .git. Returns "" when neither is found — Sites then stay absolute. Bounded by
// max_module_search_depth; go.work is preferred over .git at every level.
func recorder_site_root(recorder *Recorder, start_directory string) (root string) {
	directory := start_directory
	git_root := ""
	for range max_module_search_depth {
		base := strings.TrimPrefix(directory, "/")
		if recorder_has_entry(recorder, path.Join(base, "go.work")) {
			return directory
		}
		if git_root == "" {
			if recorder_has_entry(recorder, path.Join(base, ".git")) {
				git_root = directory
			}
		}
		parent := path.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return git_root
}

// Reports whether name exists in recorder.File_System.
func recorder_has_entry(recorder *Recorder, name string) (exists bool) {
	_, stat_error := fs.Stat(recorder.File_System, name)
	return stat_error == nil
}

// Walks up from start_directory for a go.mod, returning the module path it
// declares and the absolute directory containing it. Both are "" when none is
// found within max_module_search_depth — cross-package resolution then degrades
// to same-package bundles only.
func recorder_module(
	recorder *Recorder, start_directory string,
) (module_path string, module_root string) {
	directory := start_directory
	for range max_module_search_depth {
		relative := path.Join(strings.TrimPrefix(directory, "/"), "go.mod")
		source, read_error := fs.ReadFile(recorder.File_System, relative)
		if read_error == nil {
			return parse_module_path(source), directory
		}
		parent := path.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return "", ""
}

// Returns the module path declared by a go.mod's `module` directive, or "" when
// absent — a line scan, no golang.org/x/mod dependency.
func parse_module_path(source []byte) (module_path string) {
	for _, line := range strings.Split(string(source), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[0] != "module" {
			continue
		}
		return fields[1]
	}
	return ""
}

// Walks up from start_directory for a go.work, returning its absolute path, or ""
// when none is found within max_module_search_depth. Mirrors recorder_site_root's
// walk so the workspace it finds is the same one Sites are reported relative to.
func recorder_workspace_file(recorder *Recorder, start_directory string) (workspace_file string) {
	directory := start_directory
	for range max_module_search_depth {
		base := strings.TrimPrefix(directory, "/")
		if recorder_has_entry(recorder, path.Join(base, "go.work")) {
			return path.Join(directory, "go.work")
		}
		parent := path.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	return ""
}

// Returns the directories named by a go.work's `use` directives — both the single
// `use ./dir` form and the `use ( ... )` block. A line scan, no golang.org/x/mod
// dependency; other directives (go, toolchain, godebug, replace) are ignored.
func parse_workspace_uses(source []byte) (uses []string) {
	within_block := false
	for _, raw := range strings.Split(string(source), "\n") {
		line := raw
		if comment_offset := strings.Index(line, "//"); comment_offset >= 0 {
			line = line[:comment_offset]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if within_block {
			if line == ")" {
				within_block = false
				continue
			}
			uses = append(uses, line)
			continue
		}
		fields := strings.Fields(line)
		if fields[0] != "use" {
			continue
		}
		if len(fields) >= 2 {
			if fields[1] == "(" {
				within_block = true
				continue
			}
		}
		uses = append(uses, fields[1:]...)
	}
	return uses
}

// Resolves the go.work above start_directory into its member modules: each use
// directory's go.mod gives a (module path, absolute root) pair. nil when there is
// no go.work, so a non-workspace build keeps resolving same-module bundles only.
func recorder_workspace_modules(
	recorder *Recorder, start_directory string,
) (modules []workspace_module) {
	workspace_file := recorder_workspace_file(recorder, start_directory)
	if workspace_file == "" {
		return nil
	}
	source, read_error := fs.ReadFile(
		recorder.File_System, strings.TrimPrefix(workspace_file, "/"),
	)
	if read_error != nil {
		return nil
	}
	workspace_directory := path.Dir(workspace_file)
	for _, use := range parse_workspace_uses(source) {
		root := path.Clean(path.Join(workspace_directory, use))
		module_file := path.Join(strings.TrimPrefix(root, "/"), "go.mod")
		module_source, module_error := fs.ReadFile(recorder.File_System, module_file)
		if module_error != nil {
			continue
		}
		module_path := parse_module_path(module_source)
		if module_path == "" {
			continue
		}
		modules = append(modules, workspace_module{Path: module_path, Root: root})
	}
	return modules
}

// Input for recorder_parse_directory.
type recorder_parse_directory_input struct {
	File_System fs.FS
	File_Set    *token.FileSet
	Directory   string
	Site_Root   string
}

// Parses the non-test .go files directly under the absolute Directory into AST
// files. File_System is rooted at "/", so the leading "/" is stripped to address
// it; the parsed file's name is the absolute path made relative to Site_Root (or
// left absolute when Site_Root is empty) so positions report the same Site the
// runtime observes. Subdirectories are skipped — one directory is one package.
func recorder_parse_directory(input *recorder_parse_directory_input) (files []*ast.File) {
	root := strings.TrimPrefix(input.Directory, "/")
	fs.WalkDir(input.File_System, root, func(
		file_path string, entry fs.DirEntry, walk_error error,
	) (err error) {
		if walk_error != nil {
			return walk_error
		}
		if entry.IsDir() {
			if file_path == root {
				return nil
			}
			return fs.SkipDir
		}
		if !strings.HasSuffix(file_path, ".go") {
			return nil
		}
		if strings.HasSuffix(file_path, "_test.go") {
			return nil
		}
		source, read_error := fs.ReadFile(input.File_System, file_path)
		if read_error != nil {
			return nil
		}
		name := "/" + file_path
		if input.Site_Root != "" {
			name = strings.TrimPrefix(name, input.Site_Root+"/")
		}
		file, parse_error := parser.ParseFile(
			input.File_Set, name, source, parser.SkipObjectResolution,
		)
		if parse_error == nil {
			files = append(files, file)
		}
		return nil
	})
	return files
}

// An indexed_function is a discovered FuncDecl paired with the local-name →
// import-path map of the file it lives in (so the bundle's own qualified
// sub-calls resolve) and whether it lives in the sugar package (so the descent
// recognises its unqualified primitive calls).
type indexed_function struct {
	Declaration *ast.FuncDecl
	Imports     map[string]string
	Is_Sugar    bool
}

// A workspace_module is one module joined by the go.work workspace: its module
// path (from its go.mod) and the absolute directory of that go.mod. Used to resolve
// a *_Invariants bundle that lives in a sibling workspace module.
type workspace_module struct {
	Path string
	Root string
}

// A bundle_index resolves a *_Invariants bundle call to its declaration. Same_Set
// holds the analyzed packages' functions by bare name (same-package bundles); a
// qualified call resolves cross-package within the module via Module_Path /
// Module_Root, or cross-module to a go.work sibling via Workspace_Modules, lazily
// parsing and caching each package in Loaded.
type bundle_index struct {
	File_System       fs.FS
	File_Set          *token.FileSet
	Module_Path       string
	Module_Root       string
	Site_Root         string
	Sugar_Package     string
	Workspace_Modules []workspace_module
	Same_Set          map[string]indexed_function
	Loaded            map[string]map[string]indexed_function
}

// Maps each function name to its declaration and its file's imports, for
// descending *_Invariants bundles. A later definition wins on a name collision.
func ast_index_functions(files []*ast.File) (functions map[string]indexed_function) {
	functions = map[string]indexed_function{}
	for _, file := range files {
		imports := ast_file_imports(file)
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			functions[function.Name.Name] = indexed_function{
				Declaration: function,
				Imports:     imports,
			}
		}
	}
	return functions
}

// Registers every invariant.Dot_Product call in one parsed file. The file's
// import map is threaded down so a qualified cross-package bundle resolves.
func recorder_register_file(
	recorder *Recorder, file_set *token.FileSet, file *ast.File, index *bundle_index,
) {
	imports := ast_file_imports(file)
	for _, declaration := range file.Decls {
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		if function.Body == nil {
			continue
		}
		recorder_register_function(recorder, file_set, function, imports, index)
	}
}

// Collects the function's bindings, then registers every Dot_Product call.
func recorder_register_function(
	recorder *Recorder, file_set *token.FileSet, function *ast.FuncDecl,
	imports map[string]string, index *bundle_index,
) {
	bindings := ast_collect_bindings(function.Body)
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		if ast_invariant_selector(call) != "Dot_Product" {
			return true
		}
		recorder_register_dot_product(
			recorder, file_set, function.Body, call, bindings, imports, index,
		)
		return true
	})
}

// Checks the analyzed files for orphan axis constructors — invariant.Always /
// Sometimes / Distinct_Boundary calls that no Dot_Product consumes (directly, via
// a binding, or via a *_Invariants bundle's append). An orphan enforces nothing
// and seeds no coverage. Reports every orphan to recorder.Output and exits 1.
func recorder_check_orphan_axes(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File,
) {
	var orphans []string
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			if function.Body == nil {
				continue
			}
			legitimate := ast_collect_legitimate_axes(function)
			ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
				call, is_call := node.(*ast.CallExpr)
				if !is_call {
					return true
				}
				if !ast_is_axis_selector(ast_invariant_selector(call)) {
					return true
				}
				if legitimate[call] {
					return true
				}
				orphans = append(orphans, recorder_orphan_line(file_set, call))
				return true
			})
		}
	}
	if len(orphans) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(orphans)) + " orphan axes 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, orphan := range orphans {
		fmt.Fprintln(recorder.Output, orphan)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Checks the analyzed files for a *_Invariants / *_invariants bundle whose body
// calls invariant.Dot_Product — banned, because a bundle returns its elements for a
// caller to consume; consuming them itself would key the assertions to the bundle's
// own site instead of each caller's, defeating the per-callsite identity. Reports
// every violation under one banner and exits 1.
func recorder_check_bundle_dot_product(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File,
) {
	var violations []string
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			if function.Body == nil {
				continue
			}
			if !ast_is_invariants_name(function.Name.Name) {
				continue
			}
			name := function.Name.Name
			ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
				call, is_call := node.(*ast.CallExpr)
				if !is_call {
					return true
				}
				if ast_invariant_selector(call) != "Dot_Product" {
					return true
				}
				violations = append(violations, recorder_position(file_set, call)+
					"  banned: invariant.Dot_Product inside bundle "+name)
				return true
			})
		}
	}
	if len(violations) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(violations)) + " bundle Dot_Product calls 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, violation := range violations {
		fmt.Fprintln(recorder.Output, violation)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Collects, for one function, the axis-constructor calls that are consumed: a
// direct argument of an invariant.Dot_Product call, the RHS of a `name :=`
// binding whose name is such an argument, or — inside a *_Invariants bundle — an
// element passed to append(...). Mirrors v2's ast_collect_legitimate_axis_calls,
// extended for v3 bundles.
func ast_collect_legitimate_axes(function *ast.FuncDecl) (legitimate map[*ast.CallExpr]bool) {
	legitimate = map[*ast.CallExpr]bool{}
	consumed_names := map[string]bool{}
	is_bundle := ast_is_invariants_name(function.Name.Name)
	ast.Inspect(function, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		for _, argument := range ast_consumer_arguments(call, is_bundle) {
			argument_call, is_call_argument := argument.(*ast.CallExpr)
			if is_call_argument {
				legitimate[argument_call] = true
				continue
			}
			argument_ident, is_ident_argument := argument.(*ast.Ident)
			if is_ident_argument {
				consumed_names[argument_ident.Name] = true
			}
		}
		return true
	})
	ast.Inspect(function, func(node ast.Node) (descend bool) {
		assignment, is_assignment := node.(*ast.AssignStmt)
		if !is_assignment {
			return true
		}
		if len(assignment.Lhs) != len(assignment.Rhs) {
			return true
		}
		for index := range assignment.Lhs {
			identifier, is_identifier := assignment.Lhs[index].(*ast.Ident)
			if !is_identifier {
				continue
			}
			if !consumed_names[identifier.Name] {
				continue
			}
			call, is_call := assignment.Rhs[index].(*ast.CallExpr)
			if !is_call {
				continue
			}
			legitimate[call] = true
		}
		return true
	})
	return legitimate
}

// Returns the arguments of call that consume an axis: an invariant.Dot_Product's
// arguments, or — in a bundle — the elements after the first of an append(...).
// nil for any other call.
func ast_consumer_arguments(call *ast.CallExpr, is_bundle bool) (arguments []ast.Expr) {
	if ast_invariant_selector(call) == "Dot_Product" {
		return ast_flatten_consumers(call.Args)
	}
	if !is_bundle {
		return nil
	}
	identifier, is_identifier := call.Fun.(*ast.Ident)
	if !is_identifier {
		return nil
	}
	if identifier.Name != "append" {
		return nil
	}
	if len(call.Args) < 2 {
		return nil
	}
	return call.Args[1:]
}

// Expands a one-level append(...) among expressions into the axis-bearing
// expressions after its accumulator slice, leaving non-append expressions unchanged.
// A Dot_Product fed append(bundle, axis)... consumes that axis exactly as a bundle's
// own append does; without this the axis would read as an orphan. A post-condition
// appends a single axis to a bundle, so one level suffices — no nested-append walk.
// The accumulator (the first append argument) is always a slice, never a bare axis,
// so dropping it loses no axis.
func ast_flatten_consumers(expressions []ast.Expr) (flattened []ast.Expr) {
	for _, expression := range expressions {
		call, is_call := expression.(*ast.CallExpr)
		if !is_call {
			flattened = append(flattened, expression)
			continue
		}
		identifier, is_identifier := call.Fun.(*ast.Ident)
		if !is_identifier {
			flattened = append(flattened, expression)
			continue
		}
		if identifier.Name != "append" {
			flattened = append(flattened, expression)
			continue
		}
		if len(call.Args) < 2 {
			flattened = append(flattened, expression)
			continue
		}
		flattened = append(flattened, call.Args[1:]...)
	}
	return flattened
}

// Reports whether selector names an axis constructor (Always / Sometimes /
// Distinct_Boundary) — the kinds that must be consumed by a Dot_Product.
func ast_is_axis_selector(selector string) (is_axis bool) {
	if selector == "Always" {
		return true
	}
	if selector == "Sometimes" {
		return true
	}
	return selector == "Distinct_Boundary"
}

// Renders one orphan as "<site>  orphan: invariant.<Kind>(<condition>) is never
// passed to a Dot_Product".
func recorder_orphan_line(file_set *token.FileSet, call *ast.CallExpr) (line string) {
	selector := ast_invariant_selector(call)
	condition := ast_condition_text(file_set, call)
	if selector == "Distinct_Boundary" {
		condition = ast_boundary_condition_text(file_set, call)
	}
	return recorder_position(file_set, call) + "  orphan: invariant." + selector +
		"(" + condition + ") is never passed to a Dot_Product"
}

// A registration_axis is one Always/Sometimes element discovered statically: its
// Site (file:line), the source text of its condition, its kind, how many buckets it
// contributes to the tuple grid (Always=1 true; Sometimes=2), and whether it was
// reached by descending a *_Invariants bundle (From_Bundle) — a bundle element's
// per-element entry is keyed by the Dot_Product callsite plus its site, an inline
// element's by its site alone.
type registration_axis struct {
	Site         string
	Condition    string
	Kind         Assertion_Kind
	Bucket_Count int
	From_Bundle  bool
}

// A registration_cell is one coordinate of an Impossible carve: a Dot_Product
// axis position pinned to a bucket index.
type registration_cell struct {
	Position int
	Bucket   int
}

// Resolves the element expressions a Dot_Product consumes. A direct argument list
// (Dot_Product(a, b, Foo_Invariants(x)...)) is returned unchanged. A spread of a
// single local variable (Dot_Product(elems...)) is expanded to that variable's
// feeders — its initializer plus every append(elems, …) — so a bundle reached
// through a binding is descended too, not only the direct spread form. The feeders
// fall back to the original argument when none resolve, leaving an unresolvable
// spread to seed nothing as before.
func recorder_dot_product_arguments(
	body *ast.BlockStmt, call *ast.CallExpr,
) (arguments []ast.Expr) {
	if !call.Ellipsis.IsValid() {
		return call.Args
	}
	if len(call.Args) != 1 {
		return call.Args
	}
	identifier, is_identifier := call.Args[0].(*ast.Ident)
	if !is_identifier {
		return call.Args
	}
	feeders := ast_collect_feeders(body, identifier.Name)
	if len(feeders) == 0 {
		return call.Args
	}
	return feeders
}

// Resolves a Dot_Product's arguments into axes and carve sets (descending any
// *_Invariants bundle), then seeds a per-element entry for each axis and a tuple
// entry for every bucket combination the carves do not forbid.
func recorder_register_dot_product(
	recorder *Recorder, file_set *token.FileSet, body *ast.BlockStmt, call *ast.CallExpr,
	bindings map[string]*ast.CallExpr, imports map[string]string, index *bundle_index,
) {
	arguments := recorder_dot_product_arguments(body, call)
	axes, carves := recorder_collect_elements(file_set, arguments, bindings, imports, index)
	callsite := recorder_position(file_set, call)
	for _, axis := range axes {
		// A bundle element's own site is shared by every callsite that spreads the
		// bundle; qualifying it with the Dot_Product callsite makes each callsite its
		// own coverage entry. An inline element's site is already unique to this call,
		// so it stays bare — and the runtime credits whichever shape was seeded.
		key := axis.Site
		if axis.From_Bundle {
			key = recorder_element_key(&recorder_element_key_input{
				Callsite: callsite, Site: axis.Site,
			})
		}
		recorder.Assertions.LoadOrStore(key, &Assertion_Metadata{
			Kind:      axis.Kind,
			Site:      key,
			Condition: axis.Condition,
		})
	}
	recorder_register_tuples(recorder, callsite, axes, carves)
}

// A collect_frame is one bundle scope mid-flatten: its argument list, the
// bindings that resolve its Impossible references, a cursor over the arguments,
// and the maps that turn this scope's direct axes into flattened positions (so
// its carves can be remapped once the scope is fully walked).
type collect_frame struct {
	Arguments         []ast.Expr
	Bindings          map[string]*ast.CallExpr
	Imports           map[string]string
	Unqualified_Sugar bool
	From_Bundle       bool
	Cursor            int
	Axis_Local        map[*ast.CallExpr]int
	Axis_Global       []int
	Local_Axes        []registration_axis
}

// Resolves a list of element argument expressions into flattened axes and carve
// sets, descending *_Invariants bundles in argument order. Nested bundles flatten
// into one grid; a scope's Impossibles resolve against that scope's own direct
// axes, remapped to flattened positions when the scope completes. An explicit
// scope stack (not recursion) keeps stack depth bounded.
func recorder_collect_elements(
	file_set *token.FileSet, arguments []ast.Expr,
	bindings map[string]*ast.CallExpr, imports map[string]string, index *bundle_index,
) (axes []registration_axis, carves [][]registration_cell) {
	stack := []*collect_frame{{
		Arguments:  arguments,
		Bindings:   bindings,
		Imports:    imports,
		Axis_Local: map[*ast.CallExpr]int{},
	}}
	for step := 0; len(stack) > 0; step++ {
		if step >= max_bundle_expansion_steps {
			break
		}
		frame := stack[len(stack)-1]
		if frame.Cursor == len(frame.Arguments) {
			carves = append(carves, collect_frame_carves(frame)...)
			stack = stack[:len(stack)-1]
			continue
		}
		argument := frame.Arguments[frame.Cursor]
		frame.Cursor++
		bundle, is_bundle := ast_bundle_call(argument)
		if is_bundle {
			child, ok := recorder_bundle_frame(bundle, frame.Imports, index)
			if ok {
				stack = append(stack, child)
			}
			continue
		}
		element_call, resolved := ast_resolve_element(argument, frame.Bindings)
		if !resolved {
			continue
		}
		axis, is_axis := recorder_axis_of(file_set, element_call, frame.Unqualified_Sugar)
		if !is_axis {
			continue
		}
		// An axis collected in a descended bundle scope (non-root frame) belongs to
		// the caller+site identity; the root frame holds the Dot_Product's own
		// arguments, which are inline.
		axis.From_Bundle = frame.From_Bundle
		frame.Axis_Local[element_call] = len(frame.Local_Axes)
		frame.Axis_Global = append(frame.Axis_Global, len(axes))
		frame.Local_Axes = append(frame.Local_Axes, axis)
		axes = append(axes, axis)
	}
	return axes, carves
}

// Builds the scope frame for a *_Invariants bundle call, resolving the bundle's
// declaration through index (same-package or cross-package). ok=false when the
// bundle can't be resolved — it is then skipped. The child frame carries the
// resolved declaration's own file imports so its qualified sub-calls resolve.
func recorder_bundle_frame(
	bundle *ast.CallExpr, imports map[string]string, index *bundle_index,
) (frame *collect_frame, ok bool) {
	function, found := bundle_index_lookup(index, imports, bundle)
	if !found {
		return nil, false
	}
	if function.Declaration.Body == nil {
		return nil, false
	}
	return &collect_frame{
		Arguments:         ast_collect_append_arguments(function.Declaration.Body),
		Bindings:          ast_collect_bindings(function.Declaration.Body),
		Imports:           function.Imports,
		Unqualified_Sugar: function.Is_Sugar,
		From_Bundle:       true,
		Axis_Local:        map[*ast.CallExpr]int{},
	}, true
}

// Resolves a bundle call to its declaration using the calling file's imports: a
// bare call hits Same_Set (same-package); a qualified pkg.Foo_Invariants resolves
// pkg to an import path and, if it is inside the module, loads that package.
// found is false for an unresolvable bundle (cross-module, missing go.mod, an
// unknown qualifier, or an absent declaration).
func bundle_index_lookup(
	index *bundle_index, imports map[string]string, bundle *ast.CallExpr,
) (function indexed_function, found bool) {
	qualifier, name := ast_bundle_qualifier(bundle)
	if qualifier == "" {
		same_package, present := index.Same_Set[name]
		return same_package, present
	}
	import_path, imported := imports[qualifier]
	if !imported {
		return indexed_function{}, false
	}
	cross_package, present := bundle_index_load(index, import_path)[name]
	return cross_package, present
}

// Resolves import_path to the absolute source directory of the module that owns
// it: the primary module first, else a go.work sibling. resolved is false for a
// path in no known module (a truly external dependency). The longest matching
// module path wins, so a nested module shadows its parent.
func bundle_index_module_root(
	index *bundle_index, import_path string,
) (directory string, resolved bool) {
	best_path := ""
	best_root := ""
	consider := func(module_path string, module_root string) {
		if module_path == "" {
			return
		}
		if import_path != module_path {
			if !strings.HasPrefix(import_path, module_path+"/") {
				return
			}
		}
		if len(module_path) <= len(best_path) {
			return
		}
		best_path = module_path
		best_root = module_root
	}
	consider(index.Module_Path, index.Module_Root)
	for _, module := range index.Workspace_Modules {
		consider(module.Path, module.Root)
	}
	if best_path == "" {
		return "", false
	}
	return best_root + strings.TrimPrefix(import_path, best_path), true
}

// Lazily parses the package at import_path and returns its functions by name,
// caching the result. The cache is seeded before parsing so a cyclic import
// resolves to the empty map rather than looping. Returns an empty map for a path
// in no known module — the primary module or a go.work sibling (see
// bundle_index_module_root).
func bundle_index_load(
	index *bundle_index, import_path string,
) (functions map[string]indexed_function) {
	if cached, done := index.Loaded[import_path]; done {
		return cached
	}
	functions = map[string]indexed_function{}
	index.Loaded[import_path] = functions
	directory, resolved := bundle_index_module_root(index, import_path)
	if !resolved {
		return functions
	}
	files := recorder_parse_directory(&recorder_parse_directory_input{
		File_System: index.File_System,
		File_Set:    index.File_Set,
		Directory:   directory,
		Site_Root:   index.Site_Root,
	})
	is_sugar := import_path == index.Sugar_Package
	for name, function := range ast_index_functions(files) {
		function.Is_Sugar = is_sugar
		functions[name] = function
	}
	return functions
}

// Resolves a finished scope's Impossibles into carves over the flattened axis
// positions: ast_resolve_cells yields cells at this scope's local ordinals, which
// the frame's ordinal→global map turns into flattened positions.
func collect_frame_carves(frame *collect_frame) (carves [][]registration_cell) {
	for _, argument := range frame.Arguments {
		element_call, resolved := ast_resolve_element(argument, frame.Bindings)
		if !resolved {
			continue
		}
		if ast_selector(element_call, frame.Unqualified_Sugar) != "Impossible" {
			continue
		}
		cells, ok := ast_resolve_cells(
			element_call, frame.Bindings, frame.Axis_Local, frame.Local_Axes,
			frame.Unqualified_Sugar,
		)
		if !ok {
			continue
		}
		global := make([]registration_cell, len(cells))
		for cell_index, cell := range cells {
			global[cell_index] = registration_cell{
				Position: frame.Axis_Global[cell.Position],
				Bucket:   cell.Bucket,
			}
		}
		carves = append(carves, global)
	}
	return carves
}

// Collects the element expressions that flow into the local named name, in source
// order: the RHS of each `name := …` / `name = …` whose RHS is not an append to
// name, plus the arguments after the first of each `append(name, …)`. Straight-line
// reading only — appends inside control flow, reassignment ordering, and aliasing
// are not analyzed. Lets a Dot_Product spreading name resolve the bundles and
// elements it accumulates.
func ast_collect_feeders(body *ast.BlockStmt, name string) (feeders []ast.Expr) {
	ast.Inspect(body, func(node ast.Node) (descend bool) {
		assignment, is_assignment := node.(*ast.AssignStmt)
		if !is_assignment {
			return true
		}
		if len(assignment.Lhs) != len(assignment.Rhs) {
			return true
		}
		for index := range assignment.Lhs {
			identifier, is_identifier := assignment.Lhs[index].(*ast.Ident)
			if !is_identifier {
				continue
			}
			if identifier.Name != name {
				continue
			}
			right := assignment.Rhs[index]
			if appended, is_append := ast_append_arguments(right, name); is_append {
				feeders = append(feeders, appended...)
				continue
			}
			feeders = append(feeders, right)
		}
		return true
	})
	return feeders
}

// Returns the arguments after the first of an `append(name, …)` call, with
// is_append true only when expression is exactly that — the accumulation step a
// spread variable is built from.
func ast_append_arguments(
	expression ast.Expr, name string,
) (arguments []ast.Expr, is_append bool) {
	call, is_call := expression.(*ast.CallExpr)
	if !is_call {
		return nil, false
	}
	identifier, is_identifier := call.Fun.(*ast.Ident)
	if !is_identifier {
		return nil, false
	}
	if identifier.Name != "append" {
		return nil, false
	}
	if len(call.Args) < 2 {
		return nil, false
	}
	target, is_target := call.Args[0].(*ast.Ident)
	if !is_target {
		return nil, false
	}
	if target.Name != name {
		return nil, false
	}
	return call.Args[1:], true
}

// Collects every element expression appended in the body — each argument after
// the first of every append(...) call — i.e. a bundle's returned dot_elements.
func ast_collect_append_arguments(body *ast.BlockStmt) (arguments []ast.Expr) {
	ast.Inspect(body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		identifier, is_identifier := call.Fun.(*ast.Ident)
		if !is_identifier {
			return true
		}
		if identifier.Name != "append" {
			return true
		}
		if len(call.Args) < 2 {
			return true
		}
		arguments = append(arguments, call.Args[1:]...)
		return true
	})
	return arguments
}

// Reports whether argument is a call to a *_Invariants function (bare or
// qualified) and returns that call.
func ast_bundle_call(argument ast.Expr) (call *ast.CallExpr, is_bundle bool) {
	candidate, is_call := argument.(*ast.CallExpr)
	if !is_call {
		return nil, false
	}
	if !ast_is_invariants_name(ast_callee_name(candidate)) {
		return nil, false
	}
	return candidate, true
}

// Returns the called function's name: the Ident name for a bare call or the Sel
// name for a qualified call; "" otherwise.
func ast_callee_name(call *ast.CallExpr) (name string) {
	if identifier, is_identifier := call.Fun.(*ast.Ident); is_identifier {
		return identifier.Name
	}
	if selector, is_selector := call.Fun.(*ast.SelectorExpr); is_selector {
		return selector.Sel.Name
	}
	return ""
}

// Splits a bundle call into its package qualifier and name: ("", name) for a bare
// Foo_Invariants(), (pkg, name) for a qualified pkg.Foo_Invariants(). Both "" for
// any other call shape.
func ast_bundle_qualifier(call *ast.CallExpr) (qualifier string, name string) {
	if identifier, is_identifier := call.Fun.(*ast.Ident); is_identifier {
		return "", identifier.Name
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return "", ""
	}
	package_identifier, is_package := selector.X.(*ast.Ident)
	if !is_package {
		return "", ""
	}
	return package_identifier.Name, selector.Sel.Name
}

// Maps each of a file's imports to its local name: the explicit alias when
// present, else the import path's last segment. The latter is a heuristic —
// correct when the package's clause name matches its directory basename, which
// holds for the common case but not for, e.g., a package "invariant" in dir "v3".
func ast_file_imports(file *ast.File) (imports map[string]string) {
	imports = map[string]string{}
	for _, specification := range file.Imports {
		import_path := strings.Trim(specification.Path.Value, `"`)
		local := path.Base(import_path)
		if specification.Name != nil {
			local = specification.Name.Name
		}
		imports[local] = import_path
	}
	return imports
}

// Returns the axis for an Always/Sometimes/Distinct_Boundary constructor call;
// is_axis is false for any other call (Impossible, a bundle, a non-invariant
// call). A Boundary is a two-bucket axis (Lo=0, Hi=1) named by its X expression.
func recorder_axis_of(
	file_set *token.FileSet, call *ast.CallExpr, allow_unqualified bool,
) (axis registration_axis, is_axis bool) {
	selector := ast_selector(call, allow_unqualified)
	if selector == "Always" {
		return registration_axis{
			Site:         recorder_position(file_set, call),
			Condition:    ast_condition_text(file_set, call),
			Kind:         Assertion_Kind_Always,
			Bucket_Count: 1,
		}, true
	}
	if selector == "Sometimes" {
		return registration_axis{
			Site:         recorder_position(file_set, call),
			Condition:    ast_condition_text(file_set, call),
			Kind:         Assertion_Kind_Sometimes,
			Bucket_Count: 2,
		}, true
	}
	if selector == "Distinct_Boundary" {
		return registration_axis{
			Site:         recorder_position(file_set, call),
			Condition:    ast_boundary_condition_text(file_set, call),
			Kind:         Assertion_Kind_Boundary,
			Bucket_Count: 2,
		}, true
	}
	// Dedicated string-axis helpers (Sometimes_Has_X / Always_Has_X / Never_Has_X)
	// are single-element constructors the framework owns; each registers like the
	// bare primitive it wraps, sited at the helper's call. The whole call is the
	// condition text so a gap names the property. Never_Has_X is Always(!has_X), so
	// it is an Always axis like Always_Has_X.
	if strings.HasPrefix(selector, "Sometimes_Has_") {
		return registration_axis{
			Site:         recorder_position(file_set, call),
			Condition:    ast_expression_text(file_set, call),
			Kind:         Assertion_Kind_Sometimes,
			Bucket_Count: 2,
		}, true
	}
	if strings.HasPrefix(selector, "Always_Has_") {
		return registration_axis{
			Site:         recorder_position(file_set, call),
			Condition:    ast_expression_text(file_set, call),
			Kind:         Assertion_Kind_Always,
			Bucket_Count: 1,
		}, true
	}
	if strings.HasPrefix(selector, "Never_Has_") {
		return registration_axis{
			Site:         recorder_position(file_set, call),
			Condition:    ast_expression_text(file_set, call),
			Kind:         Assertion_Kind_Always,
			Bucket_Count: 1,
		}, true
	}
	return registration_axis{}, false
}

// Maps each `name := invariant.X(...)` local to its constructor call, so an
// Impossible's Event_True(name) can resolve name back to a Dot_Product axis.
func ast_collect_bindings(body *ast.BlockStmt) (bindings map[string]*ast.CallExpr) {
	bindings = map[string]*ast.CallExpr{}
	ast.Inspect(body, func(node ast.Node) (descend bool) {
		assignment, is_assignment := node.(*ast.AssignStmt)
		if !is_assignment {
			return true
		}
		if len(assignment.Lhs) != 1 {
			return true
		}
		if len(assignment.Rhs) != 1 {
			return true
		}
		identifier, is_identifier := assignment.Lhs[0].(*ast.Ident)
		if !is_identifier {
			return true
		}
		call, is_call := assignment.Rhs[0].(*ast.CallExpr)
		if !is_call {
			return true
		}
		bindings[identifier.Name] = call
		return true
	})
	return bindings
}

// Returns the X in a literal `invariant.X(...)` selector call, or "" otherwise.
func ast_invariant_selector(call *ast.CallExpr) (name string) {
	return ast_selector(call, false)
}

// Returns the invariant primitive a call names: the X of a qualified
// `invariant.X(...)`, or — when allow_unqualified (the call is inside a bundle in
// the sugar package) — a bare `X(...)` whose X is a known primitive. "" otherwise.
func ast_selector(call *ast.CallExpr, allow_unqualified bool) (name string) {
	if selector, is_selector := call.Fun.(*ast.SelectorExpr); is_selector {
		package_identifier, is_identifier := selector.X.(*ast.Ident)
		if !is_identifier {
			return ""
		}
		if package_identifier.Name != "invariant" {
			return ""
		}
		return selector.Sel.Name
	}
	if !allow_unqualified {
		return ""
	}
	identifier, is_identifier := call.Fun.(*ast.Ident)
	if !is_identifier {
		return ""
	}
	if !ast_is_invariant_primitive(identifier.Name) {
		return ""
	}
	return identifier.Name
}

// Reports whether name is an invariant element/reference primitive, the set the
// sugar tier exposes as bare functions and that may appear unqualified inside a
// sugar-package bundle.
func ast_is_invariant_primitive(name string) (is_primitive bool) {
	// The dedicated string-axis helpers are sugar-tier functions too, so they may
	// appear unqualified inside a sugar-package bundle (String_Invariants composes
	// them); recorder_axis_of resolves the prefix to a Sometimes/Always axis.
	if strings.HasPrefix(name, "Sometimes_Has_") {
		return true
	}
	if strings.HasPrefix(name, "Always_Has_") {
		return true
	}
	if strings.HasPrefix(name, "Never_Has_") {
		return true
	}
	switch name {
	case "Always", "Sometimes", "Distinct_Boundary",
		"Impossible", "Event_True", "Event_False":
		return true
	}
	return false
}

// Reports whether name is a bundle function name: a *_Invariants (exported) or
// *_invariants (unexported) suffix. Both casings exist because the bundle's name
// follows its type's casing — the free-function-over-a-type rule the linter
// enforces — so the analyzer must accept either or silently drop an unexported
// type's bundle coverage.
func ast_is_invariants_name(name string) (is_bundle bool) {
	return strings.HasSuffix(name, "_Invariants") || strings.HasSuffix(name, "_invariants")
}

// Resolves a Dot_Product argument to its constructor call: a direct call passes
// through; a bare identifier resolves through the bindings map.
func ast_resolve_element(
	argument ast.Expr, bindings map[string]*ast.CallExpr,
) (call *ast.CallExpr, resolved bool) {
	if direct, is_call := argument.(*ast.CallExpr); is_call {
		return direct, true
	}
	identifier, is_identifier := argument.(*ast.Ident)
	if !is_identifier {
		return nil, false
	}
	bound, has := bindings[identifier.Name]
	if !has {
		return nil, false
	}
	return bound, true
}

// Resolves an Impossible's Event_True/Event_False(local) references into (axis
// position, bucket) cells. ok is false when any reference fails to resolve to a
// Dot_Product axis — that carve is then ignored.
func ast_resolve_cells(
	impossible *ast.CallExpr, bindings map[string]*ast.CallExpr,
	axis_position map[*ast.CallExpr]int, axes []registration_axis, allow_unqualified bool,
) (cells []registration_cell, ok bool) {
	for _, argument := range impossible.Args {
		reference, is_call := argument.(*ast.CallExpr)
		if !is_call {
			return nil, false
		}
		if len(reference.Args) != 1 {
			return nil, false
		}
		identifier, is_identifier := reference.Args[0].(*ast.Ident)
		if !is_identifier {
			return nil, false
		}
		bound, has := bindings[identifier.Name]
		if !has {
			return nil, false
		}
		position, present := axis_position[bound]
		if !present {
			return nil, false
		}
		selector := ast_selector(reference, allow_unqualified)
		bucket := ast_event_bucket(selector, axes[position])
		if bucket < 0 {
			return nil, false
		}
		cells = append(cells, registration_cell{Position: position, Bucket: bucket})
	}
	return cells, true
}

// Maps Event_True/Event_False to a bucket index for the axis: Sometimes is
// false=0 / true=1; Always has only the true bucket at 0. Returns -1 when the
// reference doesn't apply to the axis kind.
func ast_event_bucket(selector string, axis registration_axis) (bucket int) {
	if axis.Kind == Assertion_Kind_Always {
		if selector == "Event_True" {
			return 0
		}
		return -1
	}
	if selector == "Event_False" {
		return 0
	}
	if selector == "Event_True" {
		return 1
	}
	return -1
}

// Returns "file:line" for the node's start position.
func recorder_position(file_set *token.FileSet, node ast.Node) (site string) {
	position := file_set.Position(node.Pos())
	return position.Filename + ":" + strconv.Itoa(position.Line)
}

// Returns the source text of the constructor's first (condition) argument, for
// the never-fired report; "" when the call has no argument.
func ast_condition_text(file_set *token.FileSet, call *ast.CallExpr) (text string) {
	if len(call.Args) == 0 {
		return ""
	}
	return ast_expression_text(file_set, call.Args[0])
}

// Returns the source text of expression, or "" when it can't be printed.
func ast_expression_text(file_set *token.FileSet, expression ast.Expr) (text string) {
	var buffer bytes.Buffer
	if printer.Fprint(&buffer, file_set, expression) != nil {
		return ""
	}
	return buffer.String()
}

// Returns the source text of a Distinct_Boundary's X expression — the value the
// boundary tracks — so the report names the boundary by what it bounds. Falls
// back to the whole argument when the Boundary_Input literal can't be read.
func ast_boundary_condition_text(file_set *token.FileSet, call *ast.CallExpr) (text string) {
	if len(call.Args) == 0 {
		return ""
	}
	composite := ast_boundary_composite(call.Args[0])
	if composite == nil {
		return ast_expression_text(file_set, call.Args[0])
	}
	for _, element := range composite.Elts {
		keyed, is_keyed := element.(*ast.KeyValueExpr)
		if !is_keyed {
			continue
		}
		key, is_identifier := keyed.Key.(*ast.Ident)
		if !is_identifier {
			continue
		}
		if key.Name != "X" {
			continue
		}
		return ast_expression_text(file_set, keyed.Value)
	}
	return ""
}

// Unwraps a Distinct_Boundary argument to its Boundary_Input composite literal,
// tolerating the leading & (a &Boundary_Input{...} pointer). Returns nil when the
// argument isn't a composite literal.
func ast_boundary_composite(argument ast.Expr) (composite *ast.CompositeLit) {
	if unary, is_unary := argument.(*ast.UnaryExpr); is_unary {
		argument = unary.X
	}
	if literal, is_literal := argument.(*ast.CompositeLit); is_literal {
		return literal
	}
	return nil
}

// Seeds one tuple entry per bucket combination of the axes, skipping any tuple a
// carve forbids.
func recorder_register_tuples(
	recorder *Recorder, site string, axes []registration_axis, carves [][]registration_cell,
) {
	if len(axes) == 0 {
		return
	}
	tuple := make([]int, len(axes))
	for more := true; more; more = recorder_tuple_increment(tuple, axes) {
		if recorder_tuple_carved(tuple, carves) {
			continue
		}
		metadata := &Assertion_Metadata{
			Kind:          Assertion_Kind_Tuple,
			Site:          site,
			Tuple_Indices: append([]int(nil), tuple...),
		}
		recorder.Assertions.LoadOrStore(recorder_tuple_key(site, tuple), metadata)
	}
}

// Advances tuple like an odometer over the axes' bucket counts; more is false
// once it wraps past the final combination.
func recorder_tuple_increment(tuple []int, axes []registration_axis) (more bool) {
	for i := len(tuple) - 1; i >= 0; i-- {
		tuple[i]++
		if tuple[i] < axes[i].Bucket_Count {
			return true
		}
		tuple[i] = 0
	}
	return false
}

// Reports whether some carve forbids tuple: a carve matches when tuple equals the
// carve's bucket at every cell position.
func recorder_tuple_carved(tuple []int, carves [][]registration_cell) (carved bool) {
	for _, carve := range carves {
		if recorder_carve_matches(tuple, carve) {
			return true
		}
	}
	return false
}

// Reports whether tuple matches every cell of a single carve.
func recorder_carve_matches(tuple []int, carve []registration_cell) (matches bool) {
	for _, cell := range carve {
		if tuple[cell.Position] != cell.Bucket {
			return false
		}
	}
	return true
}

// Builds the tuple tracker key "<site>:tuple=(i0,i1,...)".
func recorder_tuple_key(site string, tuple []int) (key string) {
	return site + ":tuple=" + recorder_tuple_indices_text(tuple)
}

// Input for recorder_element_key. Callsite and Site are both file:line strings;
// the named fields keep the caller from transposing them.
type recorder_element_key_input struct {
	Callsite string
	Site     string
}

// Builds the per-element tracker key for a bundle element: the Dot_Product callsite
// that spread the bundle, qualified by the element's own site inside the bundle. The
// "::from=" infix cannot occur in a bare "file:line" site, so combined keys never
// collide with bare per-element keys or with ":tuple=" keys.
func recorder_element_key(input *recorder_element_key_input) (key string) {
	return input.Callsite + "::from=" + input.Site
}

// Formats tuple bucket indices as "(i0,i1,...)" for tracker keys and the report.
func recorder_tuple_indices_text(tuple []int) (text string) {
	parts := make([]string, len(tuple))
	for i, index := range tuple {
		parts[i] = strconv.Itoa(index)
	}
	return "(" + strings.Join(parts, ",") + ")"
}

// A coverage_gap is one seeded assertion that the run failed to exercise, paired
// with the reason it counts as a gap (which branch or combination went unseen).
type coverage_gap struct {
	Metadata *Assertion_Metadata
	Reason   string
}

// Recorder_Analyze_Assertion_Frequency reports every pre-registered assertion
// whose true branch never fired and every Sometimes whose false branch never
// fired — naming each by its site and condition source — then calls Exit(1) when
// any gap exists. It is a no-op outside a plain `go test` run, which seeds
// nothing under benchmark or fuzz.
func Recorder_Analyze_Assertion_Frequency(recorder *Recorder) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	if recorder.Is_Fuzz {
		return
	}
	gaps := recorder_collect_gaps(recorder)
	if len(gaps) == 0 {
		return
	}
	recorder_report_gaps(recorder, gaps)
	recorder.Exit(1)
}

// Walks the tracker and returns every coverage gap across all seeded assertions.
func recorder_collect_gaps(recorder *Recorder) (gaps []coverage_gap) {
	recorder.Assertions.Range(func(key, value any) (continue_iteration bool) {
		metadata := value.(*Assertion_Metadata)
		gaps = append(gaps, assertion_metadata_gaps(metadata)...)
		return true
	})
	return gaps
}

// Returns the coverage gaps one assertion exhibits. A Sometimes contributes a gap
// per branch it never observed (true and/or false); a Boundary likewise per
// endpoint (Hi via Frequency, Lo via False_Frequency); an Always or Tuple that
// never fired is a single gap; a fully exercised assertion contributes none.
func assertion_metadata_gaps(metadata *Assertion_Metadata) (gaps []coverage_gap) {
	if metadata.Kind == Assertion_Kind_Sometimes {
		if metadata.Frequency.Load() == 0 {
			gaps = append(gaps, coverage_gap{
				Metadata: metadata, Reason: "true branch never observed",
			})
		}
		if metadata.False_Frequency.Load() == 0 {
			gaps = append(gaps, coverage_gap{
				Metadata: metadata, Reason: "false branch never observed",
			})
		}
		return gaps
	}
	if metadata.Kind == Assertion_Kind_Boundary {
		if metadata.Frequency.Load() == 0 {
			gaps = append(gaps, coverage_gap{
				Metadata: metadata, Reason: "Hi endpoint never observed",
			})
		}
		if metadata.False_Frequency.Load() == 0 {
			gaps = append(gaps, coverage_gap{
				Metadata: metadata, Reason: "Lo endpoint never observed",
			})
		}
		return gaps
	}
	if metadata.Frequency.Load() != 0 {
		return gaps
	}
	if metadata.Kind == Assertion_Kind_Tuple {
		return append(gaps, coverage_gap{Metadata: metadata, Reason: "never observed"})
	}
	return append(gaps, coverage_gap{Metadata: metadata, Reason: "never reached"})
}

// Prints the gaps to recorder.Output in v2's three sections — cross-product,
// branch, reachability — each sorted by site. A banner carrying the gap count
// brackets the report so the verdict survives a top-down or bottom-up skim.
func recorder_report_gaps(recorder *Recorder, gaps []coverage_gap) {
	banner := "🚨 " + strconv.Itoa(len(gaps)) + " coverage gaps 🚨"
	fmt.Fprintln(recorder.Output, banner)
	recorder_report_section(&recorder_report_section_input{
		Output: recorder.Output, Title: "Cross-product gaps", Gaps: gaps,
		Kind: Assertion_Kind_Tuple,
	})
	recorder_report_section(&recorder_report_section_input{
		Output: recorder.Output, Title: "Branch gaps", Gaps: gaps,
		Kind: Assertion_Kind_Sometimes,
	})
	recorder_report_section(&recorder_report_section_input{
		Output: recorder.Output, Title: "Boundary gaps", Gaps: gaps,
		Kind: Assertion_Kind_Boundary,
	})
	recorder_report_section(&recorder_report_section_input{
		Output: recorder.Output, Title: "Reachability gaps", Gaps: gaps,
		Kind: Assertion_Kind_Always,
	})
	fmt.Fprintln(recorder.Output, banner)
}

// Input for recorder_report_section.
type recorder_report_section_input struct {
	Output io.Writer
	Title  string
	Gaps   []coverage_gap
	Kind   Assertion_Kind
}

// Prints, under a markdown heading, the gaps whose assertion is of the given
// kind, sorted by site. Emits nothing when no gap matches, so empty sections
// stay silent.
func recorder_report_section(input *recorder_report_section_input) {
	selected := make([]coverage_gap, 0, len(input.Gaps))
	for _, gap := range input.Gaps {
		if gap.Metadata.Kind == input.Kind {
			selected = append(selected, gap)
		}
	}
	if len(selected) == 0 {
		return
	}
	sort.Slice(selected, func(i, j int) (less bool) {
		return selected[i].Metadata.Site < selected[j].Metadata.Site
	})
	fmt.Fprintln(input.Output)
	fmt.Fprintln(input.Output, "# "+input.Title)
	for _, gap := range selected {
		fmt.Fprintln(input.Output, coverage_gap_line(gap))
	}
}

// Renders one gap as a report line. A tuple names the missing combination; a
// branch or reachability gap names its kind, reason and condition source.
func coverage_gap_line(gap coverage_gap) (line string) {
	metadata := gap.Metadata
	if metadata.Kind == Assertion_Kind_Tuple {
		indices := recorder_tuple_indices_text(metadata.Tuple_Indices)
		return metadata.Site + "  tuple " + indices + " " + gap.Reason
	}
	return metadata.Site + "  " + assertion_kind_name(metadata.Kind) + " — " + gap.Reason +
		": " + strconv.Quote(metadata.Condition)
}

// Returns the report label for a kind: the same word the static pass keys on.
func assertion_kind_name(kind Assertion_Kind) (name string) {
	if kind == Assertion_Kind_Sometimes {
		return "Sometimes"
	}
	if kind == Assertion_Kind_Boundary {
		return "Boundary"
	}
	if kind == Assertion_Kind_Tuple {
		return "Tuple"
	}
	return "Always"
}

// Recorder_Assertion_Summary renders the clean-run banner naming how many
// properties the run tested: per-element entries (Always, Sometimes) are
// individual properties, Tuple entries are combinations, and the Always family
// is the panic-able subset whose violation fails fatally at runtime.
func Recorder_Assertion_Summary(recorder *Recorder) (summary string) {
	individual := 0
	combinations := 0
	panic_able := 0
	recorder.Assertions.Range(func(key, value any) (continue_iteration bool) {
		metadata := value.(*Assertion_Metadata)
		if metadata.Kind == Assertion_Kind_Tuple {
			combinations++
		} else {
			individual++
		}
		if metadata.Kind == Assertion_Kind_Always {
			panic_able++
		}
		if metadata.Kind == Assertion_Kind_Boundary {
			panic_able++
		}
		return true
	})
	return fmt.Sprintf(
		"✓ tested %d properties (%d individual + %d combinations, "+
			"of which %d are panic-able)",
		individual+combinations, individual, combinations, panic_able,
	)
}

// Recorder_Run_Test_Main is the canonical TestMain body: it registers the
// analyzed directories, runs the suite, reports any unexercised assertions, then
// exits with the suite's code. On a clean run — the suite passed and the analysis
// found no gaps — it prints the tested-property summary to Tty (falling back to
// Output) so the line shows even without `go test -v`.
func Recorder_Run_Test_Main(recorder *Recorder, m *testing.M, directories ...string) {
	Recorder_Register_Packages_For_Analysis(recorder, directories...)
	code := m.Run()
	Recorder_Analyze_Assertion_Frequency(recorder)
	if code != 0 {
		recorder.Exit(code)
		return
	}
	summary_output := recorder.Tty
	if summary_output == nil {
		summary_output = recorder.Output
	}
	fmt.Fprintln(summary_output, Recorder_Assertion_Summary(recorder))
	recorder.Exit(code)
}
