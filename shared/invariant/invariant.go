// Package invariant records Always and Sometimes assertions and checks, across a
// run, that every event combination they can produce is actually observed.
// Recorder_Dot_Product groups the elements of one assertion site; Impossible
// declares event combinations that must never occur.
package invariant

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"io/fs"
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

// Element_Message_Separator joins a Dot_Product's message prefix to a held axis's own
// message to form that axis's coverage key. NUL cannot appear in Go source text or a
// sane message, so it can never occur inside either half — the join is unambiguous, the
// way "::from=" was for the old file:line scheme. recorder_check_non_literal_messages keeps
// messages literal; nothing else reserves NUL.
const Element_Message_Separator = "\x00"

// Bounds the bundle-flattening loop in recorder_collect_elements: each step
// either advances one argument cursor or pops a finished scope, so an acyclic
// bundle graph finishes far below this. The cap only stops a pathological (e.g.
// self-referential) *_Invariants graph from making the work depend unboundedly
// on input — TigerStyle forbids that.
const bundle_expansion_steps_max = 4096

// Bounds the walk up the directory tree searching for a go.mod, so module
// discovery can't loop unboundedly on a pathological path.
const module_search_depth_max = 256

// Dot_Element_Kind value 0 is intentionally unassigned: Always left the element algebra
// (it is now an eager guard, see Recorder_Always), and leaving the gap means a zero-value
// Dot_Element{} carries no valid kind — it matches none of the runtime branches rather
// than silently reading as a real kind.

// Dot_Element_Kind_Sometimes tags an element whose condition must be observed
// both true and false across the run.
const Dot_Element_Kind_Sometimes Dot_Element_Kind = 1

// Dot_Element_Kind_Impossible tags a declaration that a set of element events
// must never co-occur.
const Dot_Element_Kind_Impossible Dot_Element_Kind = 2

// Assertion_Kind_Always classifies a per-element tracker entry for an Always.
const Assertion_Kind_Always Assertion_Kind = 0

// Assertion_Kind_Sometimes classifies a per-element tracker entry for a Sometimes.
const Assertion_Kind_Sometimes Assertion_Kind = 1

// Assertion_Kind_Tuple classifies a per-tuple entry of a Dot_Product's grid.
const Assertion_Kind_Tuple Assertion_Kind = 2

// Recorder accumulates assertion observations for one run and identifies each
// element by its caller Site.
type Recorder struct {
	// File_System reads Go source files during AST analysis. Paths are absolute OS paths;
	// lookups strip the leading "/" before calling fs.ReadFile.
	File_System fs.FS

	// Events is the coverage tracker: one entry per registered element bucket and
	// per Dot_Product tuple, keyed by message and credited as observations arrive.
	Events sync.Map

	// Observe_Cache_Mu guards Observe_Cache: the first-observe build takes the
	// write lock; the recording hot path reads under RLock.
	Observe_Cache_Mu sync.RWMutex
	// Observe_Cache memoizes, per Dot_Product message, the metadata pointers and
	// tracker keys its elements and grid cells resolve to, so the recording hot path
	// increments by pointer with no per-call key construction. Built lazily — the keys
	// and Loads happen once. A plain map (not sync.Map) keeps the read allocation-free.
	Observe_Cache map[string]*observe_handle

	// Output receives the coverage-gap report and the orphan/bundle diagnostics.
	Output io.Writer
	// Exit ends the process with a status code; the composition tier wires it to os.Exit.
	Exit func(code int)
	// Tty receives the clean-run success summary so it shows even without `go test -v`.
	Tty io.Writer

	// Is_Test reports a `go test` run (plain, a `-fuzz` coordinator, or a fuzz worker) — every
	// mode that records coverage. Only a benchmark opts out of recording.
	Is_Test bool
	// Is_Fuzz reports a fuzzing run (coordinator or worker).
	Is_Fuzz bool
	// Is_Fuzz_Worker reports a `-test.fuzzworker` subprocess: it runs the fuzzed body (so it
	// records, and persists each newly-covered key via Coverage_Sink for the coordinator to
	// merge), but it does not analyze — its view of coverage is partial. It always enforces.
	Is_Fuzz_Worker bool
	// Is_Benchmark reports a benchmark run, which records and checks nothing.
	Is_Benchmark bool

	// Packages_To_Analyze are the directories whose source is parsed to seed the
	// expected-coverage space.
	Packages_To_Analyze []string

	// Working_Directory resolves the relative entries of Packages_To_Analyze to
	// absolute paths. The composition tier sets it to the process working directory;
	// empty leaves a relative entry relative.
	Working_Directory string

	// Sugar_Package is the import path of the recorder-less sugar tier (the
	// package defining Always / Sometimes / … as bare functions). When a bundle
	// resolved from that package is descended, its unqualified calls to those
	// primitives are recognised; empty disables that — bare calls elsewhere are
	// not the invariant primitives and must stay unrecognised.
	Sugar_Package string

	// Coverage_Sink, when set, is called the first time each coverage key+branch is observed.
	// A fuzz worker subprocess wires it to persist its exploration to a shared file so the
	// coordinator can merge it (the coordinator never runs the fuzzed body itself). Nil
	// everywhere else — recording then just bumps the in-process counters.
	Coverage_Sink func(key string, fired_true bool)

	// Merge_Fuzz_Coverage, when set, is called by Recorder_Run_Test_Main after the suite runs
	// and before the analysis. A fuzz coordinator wires it to read every worker's persisted
	// coverage and credit it into the registered grid so the analysis sees what workers found.
	Merge_Fuzz_Coverage func()
}

// Assertion_Kind discriminates a coverage tracker entry: a per-element Always or
// Sometimes, or a per-tuple cell of a Dot_Product's grid.
type Assertion_Kind uint8

// Assertion_Metadata is one coverage tracker entry: how often an element's event
// (or a registered tuple) was observed across the run. Seeded at registration,
// incremented at runtime, scanned by the never-fired report.
type Assertion_Metadata struct {
	// Frequency counts true-event observations: an Always/Sometimes true, or a tuple.
	Frequency atomic.Int64
	// False_Frequency counts false-event observations: a Sometimes false.
	False_Frequency atomic.Int64
	// Kind discriminates the entry: Always, Sometimes, or Tuple.
	Kind Assertion_Kind
	// Message is the identity the entry is keyed by — an axis's own message, or for
	// a Tuple the Dot_Product's message prefix.
	Message string
	// Condition is the source text of the asserted expression, for the gap report.
	Condition string
	// Tuple_Indices is the bucket combination a Tuple entry tracks; nil for elements.
	Tuple_Indices []int
	// Axes is a Tuple entry's per-position legend: Axes[i] describes the axis at
	// Tuple_Indices[i]. Without it a bare coordinate cannot be mapped back to the axes
	// it came from — undebuggable once those axes descend from nested bundles. Nil for
	// element entries.
	Axes []Tuple_Axis
}

// Tuple_Axis describes one coordinate position of a Dot_Product grid: the kind,
// condition source, and message of the axis occupying it, so a never-observed cell
// names where each position came from rather than printing a bare bucket index.
type Tuple_Axis struct {
	// Kind is the axis's kind, which decodes a bucket index into its event (a Sometimes
	// 0/1 into false/true, an Always into held).
	Kind Assertion_Kind
	// Condition is the source text of the axis's asserted expression.
	Condition string
	// Message is the axis's own message.
	Message string
}

// Dot_Element is a discriminated union: a Sometimes axis (gated or not), or an Impossible
// declaration. Kind selects which fields carry meaning. An Imply is a gated Sometimes — the
// same kind with Gated set — not a kind of its own.
type Dot_Element struct {
	// Kind selects which fields carry meaning: a Sometimes axis or an Impossible declaration.
	Kind Dot_Element_Kind
	// Event is the observed outcome of a Sometimes axis: true when its condition held.
	Event bool
	// Gated marks a Sometimes whose recording is conditional on Prerequisite (an Imply).
	Gated bool
	// Prerequisite is the gate value of a Gated Sometimes: it records only when this holds,
	// and is don't-care otherwise.
	Prerequisite bool
	// Message is the axis's own message — joined to the consuming Dot_Product's prefix to
	// form the coverage key.
	Message string

	// Impossibles are the forbidden event coordinates an Impossible declares.
	Impossibles []Dot_Element_Reference
}

// Bundle is a slice of Dot_Element — what a _Invariants function returns for a caller to
// spread into a Dot_Product. An alias, so it stays interchangeable with []Dot_Element.
type Bundle = []Dot_Element

// Dot_Element_Kind discriminates a Dot_Element: Always, Sometimes, or Impossible.
type Dot_Element_Kind uint8

// Dot_Element_Reference names one element's event by its Message — a coordinate an
// Impossible declares forbidden.
type Dot_Element_Reference struct {
	// Message is the referenced element's own message.
	Message string
	// Event is the outcome of that element this reference names: true for its true event.
	Event bool
}

// Recorder_Always is an eager guard: it panics immediately when condition is false,
// naming itself by message, in every run mode. Unlike the element producers it is not a
// Dot_Element and is never consumed by Recorder_Dot_Product — there is no inert phase, so
// a constant axis does not masquerade as a cross-product element. Under a plain test run
// it also credits its reachability entry, so an Always the suite never reaches surfaces as
// a coverage gap.
func Recorder_Always[T ~bool](recorder *Recorder, condition T, message string) {
	if !condition {
		panic(Assertion_Failure_Message_Prefix + message + "  Always — condition was false")
	}
	// Enforcement (the panic above) runs in every mode; coverage is credited under a test run
	// or the fuzz coordinator (not a worker), mirroring recorder_dot_product_observe. The
	// reachability entry is seeded statically by recorder_register_eager_always;
	// recorder_increment no-ops when the bare Always was never registered (an Always in a
	// non-analyzed package).
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	recorder_increment(recorder, message, true)
}

// Recorder_Sometimes builds an element asserting condition is observed both true
// and false across the run. Like every element producer it never panics on its
// own — coverage is enforced only when the element is consumed by
// Recorder_Dot_Product; a bare Sometimes tracks nothing. message is the element's
// own identity, prefixed by the consuming Dot_Product's message.
func Recorder_Sometimes[T ~bool](
	recorder *Recorder, condition T, message string,
) (dot_element Dot_Element) {
	return Dot_Element{
		Kind:    Dot_Element_Kind_Sometimes,
		Event:   bool(condition),
		Message: message,
	}
}

// Recorder_Imply builds a gated Sometimes: an axis recorded only on a call where prerequisite
// holds, and don't-care otherwise — a failing prerequisite credits neither branch, so it never
// stands in for the gated false event. The axis is excluded from the grid (the message-less
// prerequisite is not an axis to cross with). condition is evaluated eagerly, before this runs,
// so a condition safe only under the prerequisite must still guard itself (p != nil && p.x):
// the prerequisite gates recording, not evaluation. To gate on several prerequisites, AND them.
func Recorder_Imply[P ~bool, C ~bool](
	recorder *Recorder, prerequisite P, condition C, message string,
) (dot_element Dot_Element) {
	return Dot_Element{
		Kind:         Dot_Element_Kind_Sometimes,
		Event:        bool(condition),
		Gated:        true,
		Prerequisite: bool(prerequisite),
		Message:      message,
	}
}

// Impossible declares that the referenced axis events must never all co-occur on the same call.
// Build references with Event_True / Event_False, naming sibling axes of the same Dot_Product.
//
// It globs over the axes you do not name: the carve holds only the axes you pass, and the
// analyzer treats every unnamed axis as a wildcard — it forbids, and prunes from the demanded
// grid, every tuple matching the named events across all values of the other axes (see
// recorder_carve_matches). So Impossible(Event_True("a"), Event_True("b")) excludes "a and b
// both true" across every combination of the remaining axes.
func Impossible(impossibles ...Dot_Element_Reference) (dot_element Dot_Element) {
	return Dot_Element{Kind: Dot_Element_Kind_Impossible, Impossibles: impossibles}
}

// Event_True references the axis carrying message at its true outcome, for use in Impossible. The
// message names a sibling axis of the consuming Dot_Product (matched by value, like the axis's own
// message); Recorder_Dot_Product panics if it names no sibling.
func Event_True(message string) (reference Dot_Element_Reference) {
	return Dot_Element_Reference{
		Message: message,
		Event:   true,
	}
}

// Event_False references the axis carrying message at its false outcome, for use in Impossible. See
// Event_True for how the message is matched.
func Event_False(message string) (reference Dot_Element_Reference) {
	return Dot_Element_Reference{
		Message: message,
		Event:   false,
	}
}

// Recorder_Dot_Product enforces the call's elements: an Impossible whose referenced events
// all occurred fails. Every axis
// violated on the call is named in one panic, not just the first, so a single run surfaces
// them all. message is the grid's identity and is prefixed onto each held axis's own message
// to form that axis's coverage key.
func Recorder_Dot_Product(recorder *Recorder, message string, bundle ...Dot_Element) {
	dot_product_check_references(bundle)
	var violations []string
	for _, dot_element := range bundle {
		violation := dot_element_violation(dot_element, bundle)
		if violation != "" {
			violations = append(violations, violation)
		}
	}
	if len(violations) > 0 {
		panic(Assertion_Failure_Message_Prefix + strings.Join(violations, "\n"))
	}
	recorder_dot_product_observe(recorder, message, bundle)
}

// Panics when an Impossible names a message that is not an axis of this Dot_Product — one of its
// siblings. A reference can only carve a cell of this product's grid and can only fire against an
// event observed on this same call, so naming a non-sibling is structurally meaningless; catching
// it here, before any recording and on every call, surfaces a typo immediately rather than as an
// unfillable gap — independent of whether the forbidden combination ever occurs. A bundle with no
// Impossible costs nothing.
func dot_product_check_references(bundle Bundle) {
	has_impossible := false
	for _, dot_element := range bundle {
		if dot_element.Kind == Dot_Element_Kind_Impossible {
			has_impossible = true
			break
		}
	}
	if !has_impossible {
		return
	}
	siblings := map[string]bool{}
	for _, dot_element := range bundle {
		if dot_element.Kind == Dot_Element_Kind_Sometimes {
			siblings[dot_element.Message] = true
		}
	}
	for _, dot_element := range bundle {
		if dot_element.Kind != Dot_Element_Kind_Impossible {
			continue
		}
		for _, reference := range dot_element.Impossibles {
			if siblings[reference.Message] {
				continue
			}
			panic(Assertion_Failure_Message_Prefix +
				"Impossible references " + strconv.Quote(reference.Message) +
				", not an axis of this Dot_Product")
		}
	}
}

// An observe_handle memoizes, for one Dot_Product message, what its bundle resolves to so the
// recording hot path builds no key per call: the metadata + tracker key for each non-Impossible
// element (in bundle order) and for each grid cell (indexed by the packed bucket tuple).
type observe_handle struct {
	// Elements holds one entry per non-Impossible element, in bundle order.
	Elements []handle_entry
	// Tuples is indexed by the observed tuple packed big-endian — element 0 is the
	// most significant bit, one per axis (a Sometimes has two buckets), size
	// 1<<len(Elements). A nil-metadata entry is a cell an Impossible carved or never seeded.
	Tuples []handle_entry
}

// A handle_entry is a resolved tracker slot: the seeded metadata and the tracker key, cached so
// Coverage_Sink can persist it without rebuilding the string.
type handle_entry struct {
	// Metadata is the seeded tracker entry, nil when registration seeded none.
	Metadata *Assertion_Metadata
	// Key is the tracker key, cached so Coverage_Sink can persist it without rebuilding it.
	Key string
}

// Increments the seeded tracker entry for each observed element and the tuple entry for the
// observed combination, through the per-message observe_handle so the steady state allocates
// nothing. Records under a plain test, the fuzz coordinator, and a fuzz worker (all carry
// Is_Test); a no-op in a benchmark or a non-test binary, which only enforce.
func recorder_dot_product_observe(
	recorder *Recorder, message string, bundle Bundle,
) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	handle := recorder_observe_handle(recorder, message, bundle)
	axis_index := 0
	packed := 0
	for _, dot_element := range bundle {
		if dot_element.Kind != Dot_Element_Kind_Sometimes {
			continue
		}
		entry := handle.Elements[axis_index]
		axis_index++
		if dot_element.Gated {
			// A gated axis records only when its prerequisite holds — else don't-care —
			// and joins no tuple (an Imply is excluded from the grid).
			if dot_element.Prerequisite {
				recorder_increment_entry(recorder, entry, dot_element.Event)
			}
			continue
		}
		packed <<= 1
		if dot_element.Event {
			packed |= 1
		}
		recorder_increment_entry(recorder, entry, dot_element.Event)
	}
	recorder_increment_entry(recorder, handle.Tuples[packed], true)
}

// Returns the cached observe_handle for message, building it on first use. The build resolves
// every element and grid-cell key against the seeded tracker once; later calls read it under
// RLock with no allocation (a plain map keyed by the existing message string boxes nothing).
func recorder_observe_handle(
	recorder *Recorder, message string, bundle Bundle,
) (handle *observe_handle) {
	recorder.Observe_Cache_Mu.RLock()
	handle = recorder.Observe_Cache[message]
	recorder.Observe_Cache_Mu.RUnlock()
	if handle != nil {
		return handle
	}
	recorder.Observe_Cache_Mu.Lock()
	defer recorder.Observe_Cache_Mu.Unlock()
	if handle = recorder.Observe_Cache[message]; handle != nil {
		return handle
	}
	handle = recorder_observe_handle_build(recorder, message, bundle)
	if recorder.Observe_Cache == nil {
		recorder.Observe_Cache = map[string]*observe_handle{}
	}
	recorder.Observe_Cache[message] = handle
	return handle
}

// Builds an observe_handle: one element entry per non-Impossible element (keyed prefix +
// separator + own message) and one tuple entry per grid cell, keyed by the projected coordinate
// exactly as registration seeded it. A cell or element registration never seeded resolves to a
// nil-metadata entry, so the runtime skips it.
func recorder_observe_handle_build(
	recorder *Recorder, message string, bundle Bundle,
) (handle *observe_handle) {
	handle = &observe_handle{}
	ungated_count := 0
	for _, dot_element := range bundle {
		if dot_element.Kind != Dot_Element_Kind_Sometimes {
			continue
		}
		key := message + Element_Message_Separator + dot_element.Message
		handle.Elements = append(handle.Elements, recorder_handle_entry(recorder, key))
		if !dot_element.Gated {
			ungated_count++
		}
	}
	handle.Tuples = make([]handle_entry, 1<<ungated_count)
	for packed := range handle.Tuples {
		tuple := make([]int, ungated_count)
		for i := range tuple {
			tuple[i] = packed >> (ungated_count - 1 - i) & 1
		}
		tuple_key := recorder_tuple_key(message, tuple)
		handle.Tuples[packed] = recorder_handle_entry(recorder, tuple_key)
	}
	return handle
}

// Resolves key to its seeded tracker metadata (nil when none was seeded), pairing it with the
// key so Coverage_Sink can persist it on first coverage.
func recorder_handle_entry(recorder *Recorder, key string) (entry handle_entry) {
	entry.Key = key
	if value, ok := recorder.Events.Load(key); ok {
		entry.Metadata = value.(*Assertion_Metadata)
	}
	return entry
}

// Bumps entry's metadata: Frequency on a true event, False_Frequency on false. A nil-metadata
// entry (registration seeded none, like a carved cell) is skipped. On the 0→1 transition of a
// branch (its first coverage) it fires Coverage_Sink with the entry's cached key, so a fuzz
// worker persists the cell; atomic Add returns the post-increment value, so the sink fires once
// per branch.
func recorder_increment_entry(recorder *Recorder, entry handle_entry, fired_true bool) {
	if entry.Metadata == nil {
		return
	}
	if fired_true {
		if entry.Metadata.Frequency.Add(1) == 1 {
			if recorder.Coverage_Sink != nil {
				recorder.Coverage_Sink(entry.Key, true)
			}
		}
		return
	}
	if entry.Metadata.False_Frequency.Add(1) == 1 {
		if recorder.Coverage_Sink != nil {
			recorder.Coverage_Sink(entry.Key, false)
		}
	}
}

// Bumps the seeded entry at key — the coordinator's merge path (Recorder_Merge_Fuzz_Coverage_From),
// which holds only string keys, not the runtime's cached handles. A missing entry is skipped.
func recorder_increment(recorder *Recorder, key string, fired_true bool) {
	value, ok := recorder.Events.Load(key)
	if !ok {
		return
	}
	recorder_increment_entry(
		recorder, handle_entry{Metadata: value.(*Assertion_Metadata), Key: key}, fired_true)
}

// Fuzz_Coverage_Line encodes one covered (key, branch) as the line a fuzz worker appends to
// the shared coverage file: base64(key) + "\t" + "T"/"F" + "\n". base64 because a key carries
// the NUL element-separator and otherwise-arbitrary bytes; the trailing newline makes the file
// line-oriented for the coordinator's merge. It is one string, so the worker writes it with a
// single Write under O_APPEND (the lock-free-append requirement).
func Fuzz_Coverage_Line(key string, fired_true bool) (line string) {
	branch := "F"
	if fired_true {
		branch = "T"
	}
	return base64.StdEncoding.EncodeToString([]byte(key)) + "\t" + branch + "\n"
}

func recorder_merge_process_line(recorder *Recorder, line string) {
	tab_offset := strings.IndexByte(line, '\t')
	if tab_offset < 0 {
		return
	}
	key, decode_error := base64.StdEncoding.DecodeString(line[:tab_offset])
	if decode_error != nil {
		return
	}
	recorder_increment(recorder, string(key), line[tab_offset+1:] == "T")
}

// Recorder_Merge_Fuzz_Coverage_From unions the coverage a fuzz coordinator reads from r
// (the workers' shared file, one Fuzz_Coverage_Line per line) into the registered grid: each
// covered (key, branch) marks that branch non-zero. Binary — per-process counts are not summed
// across workers. A malformed or partial trailing line is skipped (a worker killed mid-write
// costs at most its last line); a key with no seeded entry is skipped, like any runtime increment.
func Recorder_Merge_Fuzz_Coverage_From(recorder *Recorder, r io.Reader) {
	var buffer [4096]byte
	var partial string
	var read_error error
	for read_error == nil {
		var n int
		n, read_error = r.Read(buffer[:])
		if n > 0 {
			chunk := partial + string(buffer[:n])
			newline_offset := strings.IndexByte(chunk, '\n')
			for newline_offset >= 0 {
				recorder_merge_process_line(recorder, chunk[:newline_offset])
				chunk = chunk[newline_offset+1:]
				newline_offset = strings.IndexByte(chunk, '\n')
			}
			partial = chunk
		}
	}
}

// Returns the message describing how dot_element violates its invariant on this
// call — an Impossible whose forbidden combination occurred — naming the offending axis by its
// message. Returns "" when the element holds. Always is not an element and never reaches here; it
// enforces eagerly.
func dot_element_violation(
	dot_element Dot_Element, bundle Bundle,
) (violation string) {
	if dot_element.Kind != Dot_Element_Kind_Impossible {
		return ""
	}
	if dot_element_impossible_violated(dot_element, bundle) {
		return dot_element_impossible_message(dot_element)
	}
	return ""
}

// Renders an Impossible violation: a header plus one line per co-occurring
// coordinate, each naming the referenced axis by its message and the event observed.
// The Impossible element's own message is empty, so its identity is this set of
// coordinates rather than a single message.
func dot_element_impossible_message(impossible Dot_Element) (message string) {
	message = "Impossible — forbidden combination occurred:"
	for _, reference := range impossible.Impossibles {
		message += "\n  " + reference.Message + "  " + event_boolean_text(reference.Event)
	}
	return message
}

// Renders an Impossible reference's event as the boolean word it carries.
func event_boolean_text(event bool) (text string) {
	if event {
		return "true"
	}
	return "false"
}

// Reports whether every event the Impossible names was observed this call — the forbidden
// combination occurred in full. An Impossible with no references constrains nothing, so it
// never fires (rather than firing vacuously every call).
func dot_element_impossible_violated(
	impossible Dot_Element, bundle Bundle,
) (violated bool) {
	if len(impossible.Impossibles) == 0 {
		return false
	}
	for _, reference := range impossible.Impossibles {
		if !dot_element_reference_observed(reference, bundle) {
			return false
		}
	}
	return true
}

// Reports whether some Sometimes axis in the call carries the reference's Message and was seen
// at the event the reference names. Impossible elements observe nothing and are skipped.
func dot_element_reference_observed(
	reference Dot_Element_Reference, bundle Bundle,
) (observed bool) {
	for _, dot_element := range bundle {
		if dot_element.Kind != Dot_Element_Kind_Sometimes {
			continue
		}
		if dot_element.Message != reference.Message {
			continue
		}
		if dot_element.Event != reference.Event {
			continue
		}
		return true
	}
	return false
}

// Recorder_Register_Packages_For_Analysis parses every non-test .go file under
// the given directories and seeds recorder.Events with one entry per element
// bucket and one per non-carved tuple of each invariant.Dot_Product call. That
// seeded set is the expected-coverage space the never-fired report scans after
// the run; literal invariant.X selectors and *_Invariants bundles are recognised.
//
// Directories default to recorder.Packages_To_Analyze when none are passed. Each
// assertion is keyed by its message; a duplicate message, or a message that is not a
// string literal, fails registration (see recorder_check_duplicate_messages /
// recorder_check_non_literal_messages).
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
		// A filepath.Abs here would reach the OS for the working directory, which a pure
		// package must not do; Working_Directory is injected so this stays pure.
		absolute := directory
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(recorder.Working_Directory, absolute)
		}
		absolute = filepath.Clean(absolute)
		if primary_directory == "" {
			primary_directory = absolute
		}
		if module_path == "" {
			module_path, module_root = recorder_module(recorder, absolute)
		}
		parsed := recorder_parse_directory(&recorder_parse_directory_input{
			File_System: recorder.File_System,
			File_Set:    file_set,
			Directory:   absolute,
		})
		files = append(files, parsed...)
	}
	index := &bundle_index{
		File_System:       recorder.File_System,
		File_Set:          file_set,
		Module_Path:       module_path,
		Module_Root:       module_root,
		Sugar_Package:     recorder.Sugar_Package,
		Workspace_Modules: recorder_workspace_modules(recorder, primary_directory),
		Same_Set:          ast_index_functions(files),
		Loaded:            map[string]map[string]indexed_function{},
	}
	reg := &registration{Seen_Prefix: map[string]bool{}}
	for _, file := range files {
		recorder_register_file(recorder, file_set, file, index, reg)
	}
	recorder_check_bundle_control_flow(recorder, file_set, files)
	recorder_check_unresolved(recorder, reg.Unresolved)
	recorder_check_non_literal_messages(recorder, reg.Non_Literal)
	recorder_check_duplicate_messages(recorder, reg.Collision)
}

// Reports whether name exists in recorder.File_System.
func recorder_has_entry(recorder *Recorder, name string) (exists bool) {
	_, stat_error := fs.Stat(recorder.File_System, name)
	return stat_error == nil
}

// Walks up from start_directory for a go.mod, returning the module path it
// declares and the absolute directory containing it. Both are "" when none is
// found within module_search_depth_max — cross-package resolution then degrades
// to same-package bundles only.
func recorder_module(
	recorder *Recorder, start_directory string,
) (module_path string, module_root string) {
	directory := start_directory
	for range module_search_depth_max {
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
// when none is found within module_search_depth_max — the workspace whose member
// modules cross-package bundle resolution searches.
func recorder_workspace_file(recorder *Recorder, start_directory string) (workspace_file string) {
	directory := start_directory
	for range module_search_depth_max {
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
}

// Parses the non-test .go files directly under the absolute Directory into AST
// files. File_System is rooted at "/", so the leading "/" is stripped to address
// it; the parsed file's name is the absolute path, used only for diagnostics now
// (identity is the message, not the position). Subdirectories are skipped — one
// directory is one package.
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
	reg *registration,
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
		recorder_register_function(recorder, file_set, function, imports, index, reg)
	}
}

// Registers each invariant call in the function: a direct Dot_Product with a literal prefix
// seeds a grid; a Dot_Product whose prefix is this function's namespace parameter is a
// grid template, registered at its callsites instead; a _Invariants(v, "lit") call registers
// the called template's grid under "lit"; any other call may be a bare eager Always.
func recorder_register_function(
	recorder *Recorder, file_set *token.FileSet, function *ast.FuncDecl,
	imports map[string]string, index *bundle_index, reg *registration,
) {
	namespace_parameter := ""
	if ast_is_invariants_name(function.Name.Name) {
		namespace_parameter = ast_namespace_parameter(function)
	}
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		if ast_invariant_selector(call) == "Dot_Product" {
			recorder_register_dot_product(
				recorder, file_set, call, namespace_parameter, imports, index, reg)
			return true
		}
		if ast_is_invariants_name(ast_callee_name(call)) {
			recorder_register_invariants_callsite(
				recorder, file_set, call, imports, index, reg)
			return true
		}
		// An eager Always never flows through a Dot_Product, so this walk is the only
		// registration that sees it — keyed by its own message like any other.
		recorder_register_eager_always(recorder, file_set, call, reg)
		return true
	})
}

// Returns the name of a _Invariants function's trailing string parameter — the namespace its
// self-emitted Dot_Product is prefixed by. "" when there is no such parameter.
func ast_namespace_parameter(function *ast.FuncDecl) (name string) {
	if function.Type.Params == nil {
		return ""
	}
	fields := function.Type.Params.List
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	identifier, is_string := last.Type.(*ast.Ident)
	if !is_string {
		return ""
	}
	if identifier.Name != "string" {
		return ""
	}
	if len(last.Names) == 0 {
		return ""
	}
	return last.Names[len(last.Names)-1].Name
}

// Seeds a reachability entry for a bare eager Always call — invariant.Always /
// Recorder_Always / Always_Has_X / Never_Has_X — keyed by its own message. Without this a
// never-reached Always could not be reported, since it never flows through a Dot_Product.
// Calls of any other kind (a Sometimes, a plain function) seed
// nothing: a bare element records nothing and is the caller's responsibility to consume.
// A duplicate Always message is a fatal collision.
func recorder_register_eager_always(
	recorder *Recorder, file_set *token.FileSet, call *ast.CallExpr, reg *registration,
) {
	axis, is_axis := recorder_axis_of(file_set, call, false, reg)
	if !is_axis {
		return
	}
	if axis.Kind != Assertion_Kind_Always {
		return
	}
	_, loaded := recorder.Events.LoadOrStore(axis.Message, &Assertion_Metadata{
		Kind:      Assertion_Kind_Always,
		Message:   axis.Message,
		Condition: axis.Condition,
	})
	if loaded {
		reg.Collision = append(reg.Collision,
			recorder_position(file_set, call)+
				"  duplicate message: "+strconv.Quote(axis.Message))
	}
}

// Reports every bundle the analyzer recognised by name but could not resolve to a
// declaration, then exits. A recognised-but-unresolvable bundle would seed none of
// its elements while the runtime still enforces them, so its coverage obligations
// would vanish unnoticed; failing keeps coverage from being silently dropped — the
// analyzer descends a bundle or refuses it.
func recorder_check_unresolved(recorder *Recorder, unresolved []string) {
	if len(unresolved) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(unresolved)) + " unresolved bundles 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range unresolved {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Reports every assertion whose message is not a string literal, then exits. The runtime
// stamps whatever the message expression evaluates to, but the static side cannot key a
// non-literal — so its coverage would never be credited and its gap would vanish. Refuse
// it: a message is a compile-time literal or registration fails.
func recorder_check_non_literal_messages(recorder *Recorder, non_literal []string) {
	if len(non_literal) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(non_literal)) + " non-literal messages 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range non_literal {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Reports every message collision, then exits. Two distinct assertions claiming one
// message — two Dot_Products sharing a prefix, a repeated axis message within one
// Dot_Product, or two Always sharing a message — would silently merge into one entry and
// mask a gap. A duplicate is fatal, never merged.
func recorder_check_duplicate_messages(recorder *Recorder, collisions []string) {
	if len(collisions) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(collisions)) + " duplicate messages 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range collisions {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Checks the analyzed files for a *_Invariants / *_invariants bundle whose body contains a
// branching or looping statement (if, switch, type-switch, for, range, select) — banned,
// because it would make the axes the bundle self-emits depend on runtime values the static scan
// cannot read, silently under-registering coverage. A bundle body must be straight-line.
// Reports every violation under one banner and exits 1.
func recorder_check_bundle_control_flow(
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
				if !ast_is_control_flow(node) {
					return true
				}
				violations = append(violations, recorder_position(file_set, node)+
					"  banned: control flow inside bundle "+name)
				return true
			})
		}
	}
	if len(violations) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(violations)) + " bundle control-flow statements 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, violation := range violations {
		fmt.Fprintln(recorder.Output, violation)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Reports whether node is a branching or looping statement banned in a bundle body.
func ast_is_control_flow(node ast.Node) (is_control_flow bool) {
	switch node.(type) {
	case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt,
		*ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt:
		return true
	}
	return false
}

// A registration_axis is one Always/Sometimes element discovered statically:
// its Message (the element's own literal), the source text of its condition, its kind,
// and how many buckets it contributes to the tuple grid (Always=1 true; Sometimes=2).
// The consuming Dot_Product's message is prefixed onto Message to form the coverage key,
// uniformly for inline and bundle-descended axes alike.
type registration_axis struct {
	Message      string
	Condition    string
	Kind         Assertion_Kind
	Bucket_Count int
	// Gated marks an Imply axis — seeded per-axis but excluded from the tuple grid.
	Gated bool
}

// A registration_cell is one coordinate of an Impossible carve: a Dot_Product
// axis position pinned to a bucket index.
type registration_cell struct {
	Position int
	Bucket   int
}

// Registration accumulates the diagnostics a registration pass gathers before deciding
// whether to fail: bundles recognised by name but unresolvable, messages that are not
// string literals, and message collisions. Each is fatal on its own (see the
// recorder_check_* reporters). Seen_Prefix tracks Dot_Product messages so two grids
// cannot share one — the global-uniqueness guarantee for prefixes.
type registration struct {
	Unresolved  []string
	Non_Literal []string
	Collision   []string
	Seen_Prefix map[string]bool
}

// Returns the unquoted Go string value of the argument at index when it is a string
// literal, mirroring the message the runtime stamps. ok is false when the argument is
// absent, not a *ast.BasicLit, not a STRING, or unquotable — i.e. a variable or a
// concatenation the static side cannot resolve to a key.
func ast_string_literal(call *ast.CallExpr, index int) (value string, ok bool) {
	if len(call.Args) <= index {
		return "", false
	}
	literal, is_literal := call.Args[index].(*ast.BasicLit)
	if !is_literal {
		return "", false
	}
	if literal.Kind != token.STRING {
		return "", false
	}
	unquoted, unquote_error := strconv.Unquote(literal.Value)
	if unquote_error != nil {
		return "", false
	}
	return unquoted, true
}

// Registers a Dot_Product call. A literal prefix seeds a grid from the call's inline axes. A
// prefix that is this function's namespace parameter is a grid template — registered at the
// _Invariants' callsites, not here. Any other non-literal prefix is fatal.
func recorder_register_dot_product(
	recorder *Recorder, file_set *token.FileSet, call *ast.CallExpr, namespace_parameter string,
	imports map[string]string, index *bundle_index, reg *registration,
) {
	prefix, literal := ast_string_literal(call, 0)
	if !literal {
		if ast_is_template_prefix(call, namespace_parameter) {
			return
		}
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, call)+
				"  Dot_Product message is not a string literal")
		return
	}
	axes, carves := recorder_collect_inline(file_set, call.Args[1:], false, reg)
	recorder_seed_grid(recorder, file_set, call, prefix, axes, carves, reg)
}

// Reports whether the Dot_Product's prefix argument is this function's namespace parameter — the
// shape of a grid template, registered at the _Invariants' callsites rather than here.
func ast_is_template_prefix(call *ast.CallExpr, namespace_parameter string) (is_template bool) {
	if namespace_parameter == "" {
		return false
	}
	if len(call.Args) == 0 {
		return false
	}
	identifier, is_identifier := call.Args[0].(*ast.Ident)
	if !is_identifier {
		return false
	}
	return identifier.Name == namespace_parameter
}

// Registers the grid of a _Invariants(v, "lit") callsite: resolves the called template, reads
// the axes of its self-emitted Dot_Product, and seeds them under the literal namespace. A
// non-literal namespace is fatal (its coverage could not be keyed); an unresolvable template is
// fatal. Nested _Invariants calls inside the template body are registered separately by the
// global walk, never flattened into this grid.
func recorder_register_invariants_callsite(
	recorder *Recorder, file_set *token.FileSet, call *ast.CallExpr,
	imports map[string]string, index *bundle_index, reg *registration,
) {
	if len(call.Args) < 2 {
		return
	}
	namespace, literal := ast_string_literal(call, len(call.Args)-1)
	if !literal {
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, call)+
				"  _Invariants namespace is not a string literal")
		return
	}
	function, found := bundle_index_lookup(index, imports, call)
	if !found {
		reg.Unresolved = append(reg.Unresolved, recorder_unresolved_line(file_set, call))
		return
	}
	if function.Declaration.Body == nil {
		reg.Unresolved = append(reg.Unresolved, recorder_unresolved_line(file_set, call))
		return
	}
	dot_product, has := recorder_template_dot_product(function.Declaration)
	if !has {
		return
	}
	axes, carves := recorder_collect_inline(
		file_set, dot_product.Args[1:], function.Is_Sugar, reg)
	recorder_seed_grid(recorder, file_set, call, namespace, axes, carves, reg)
}

// Finds a _Invariants function's self-emitted Dot_Product — the one whose prefix argument is
// the function's namespace parameter. has is false when the function emits no such Dot_Product.
func recorder_template_dot_product(
	function *ast.FuncDecl,
) (call *ast.CallExpr, has bool) {
	namespace_parameter := ast_namespace_parameter(function)
	if namespace_parameter == "" {
		return nil, false
	}
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		if has {
			return false
		}
		candidate, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		if ast_selector(candidate, true) != "Dot_Product" {
			return true
		}
		if !ast_is_template_prefix(candidate, namespace_parameter) {
			return true
		}
		call = candidate
		has = true
		return false
	})
	return call, has
}

// Seeds a grid under prefix: a per-axis entry for every axis (gated or not) and a tuple entry
// for every non-carved combination of the ungated axes. A repeated prefix or a repeated axis
// message within the grid is a duplicate collision.
func recorder_seed_grid(
	recorder *Recorder, file_set *token.FileSet, position ast.Node, prefix string,
	axes []registration_axis, carves [][]registration_cell, reg *registration,
) {
	if reg.Seen_Prefix[prefix] {
		reg.Collision = append(reg.Collision,
			recorder_position(file_set, position)+
				"  duplicate Dot_Product message: "+strconv.Quote(prefix))
		return
	}
	reg.Seen_Prefix[prefix] = true
	for _, axis := range axes {
		key := prefix + Element_Message_Separator + axis.Message
		_, loaded := recorder.Events.LoadOrStore(key, &Assertion_Metadata{
			Kind:      axis.Kind,
			Message:   key,
			Condition: axis.Condition,
		})
		if loaded {
			reg.Collision = append(reg.Collision,
				recorder_position(file_set, position)+
					"  duplicate message: "+strconv.Quote(prefix)+
					" / "+strconv.Quote(axis.Message))
		}
	}
	ungated := make([]registration_axis, 0, len(axes))
	for _, axis := range axes {
		if !axis.Gated {
			ungated = append(ungated, axis)
		}
	}
	recorder_register_tuples(recorder, prefix, ungated, carves)
}

// Reads a Dot_Product's inline element arguments into axes and carves. A self-emitting
// Dot_Product holds only inline Sometimes/Imply/Impossible — composition is separate _Invariants
// calls, not spreads — so there is no bundle descent. Each Impossible carve resolves its two
// references against the ungated axis positions; the grid excludes gated Imply axes.
func recorder_collect_inline(
	file_set *token.FileSet, arguments []ast.Expr, allow_unqualified bool, reg *registration,
) (axes []registration_axis, carves [][]registration_cell) {
	var carve_calls []*ast.CallExpr
	for _, argument := range arguments {
		call, is_call := argument.(*ast.CallExpr)
		if !is_call {
			continue
		}
		if ast_selector(call, allow_unqualified) == "Impossible" {
			carve_calls = append(carve_calls, call)
			continue
		}
		axis, is_axis := recorder_axis_of(file_set, call, allow_unqualified, reg)
		if is_axis {
			axes = append(axes, axis)
		}
	}
	position_of := map[string]int{}
	ungated := 0
	for _, axis := range axes {
		if axis.Gated {
			continue
		}
		position_of[axis.Message] = ungated
		ungated++
	}
	for _, carve_call := range carve_calls {
		cells, ok := ast_resolve_carve(
			file_set, carve_call, position_of, allow_unqualified, reg)
		if ok {
			carves = append(carves, cells)
		}
	}
	return axes, carves
}

// Renders one unresolvable bundle as
// "<site>  unresolved bundle: <name> cannot be analyzed".
func recorder_unresolved_line(file_set *token.FileSet, call *ast.CallExpr) (line string) {
	return recorder_position(file_set, call) + "  unresolved bundle: " +
		ast_callee_name(call) + " cannot be analyzed"
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
	})
	is_sugar := import_path == index.Sugar_Package
	for name, function := range ast_index_functions(files) {
		function.Is_Sugar = is_sugar
		functions[name] = function
	}
	return functions
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

// Maps an axis constructor selector to its assertion kind and the index of its
// condition-bearing argument. The bare sugar forms (Always / Sometimes) carry the condition
// first; the explicit Recorder_* forms lead with the recorder, so the condition rides the
// second argument. is_axis is false for any other selector. The Sometimes_Has_ / Always_Has_ /
// Never_Has_ helpers are absent here: they take no recorder and report the whole call as their
// condition.
func ast_axis_signature(
	selector string,
) (kind Assertion_Kind, condition_index int, gated bool, is_axis bool) {
	switch selector {
	case "Always":
		return Assertion_Kind_Always, 0, false, true
	case "Sometimes":
		return Assertion_Kind_Sometimes, 0, false, true
	case "Imply":
		return Assertion_Kind_Sometimes, 1, true, true
	case "Recorder_Always":
		return Assertion_Kind_Always, 1, false, true
	case "Recorder_Sometimes":
		return Assertion_Kind_Sometimes, 1, false, true
	case "Recorder_Imply":
		return Assertion_Kind_Sometimes, 2, true, true
	}
	return 0, 0, false, false
}

// Returns the axis for an Always/Sometimes constructor call, in either the bare sugar form or
// the explicit Recorder_* form; is_axis is false for any other call (Impossible, a bundle, a
// non-invariant call).
func recorder_axis_of(
	file_set *token.FileSet, call *ast.CallExpr, allow_unqualified bool, reg *registration,
) (axis registration_axis, is_axis bool) {
	selector := ast_selector(call, allow_unqualified)
	if kind, condition_index, gated, ok := ast_axis_signature(selector); ok {
		condition := ast_condition_text(file_set, call, condition_index)
		bucket_count := 2
		if kind == Assertion_Kind_Always {
			bucket_count = 1
		}
		// The message is the argument past the condition; the runtime stamps the same
		// literal. A non-literal cannot be keyed, so it is reported and fails registration.
		message, literal := ast_string_literal(call, condition_index+1)
		if !literal {
			reg.Non_Literal = append(reg.Non_Literal,
				recorder_position(file_set, call)+
					"  "+selector+" message is not a string literal")
		}
		return registration_axis{
			Message:      message,
			Condition:    condition,
			Kind:         kind,
			Bucket_Count: bucket_count,
			Gated:        gated,
		}, true
	}
	// Dedicated string-axis helpers (Sometimes_Has_X / Always_Has_X / Never_Has_X) are
	// single-element constructors the framework owns. They take no message argument; their
	// identity is the helper's own name, which the sugar passes verbatim to the underlying
	// recorder, so the static side derives the same message here without a literal at the
	// call. Never_Has_X is Always(!has_X), an Always axis like Always_Has_X. The whole call
	// is the condition text so a gap names the property.
	if strings.HasPrefix(selector, "Sometimes_Has_") {
		return registration_axis{
			Message:      selector,
			Condition:    ast_expression_text(file_set, call),
			Kind:         Assertion_Kind_Sometimes,
			Bucket_Count: 2,
		}, true
	}
	if strings.HasPrefix(selector, "Always_Has_") {
		return registration_axis{
			Message:      selector,
			Condition:    ast_expression_text(file_set, call),
			Kind:         Assertion_Kind_Always,
			Bucket_Count: 1,
		}, true
	}
	if strings.HasPrefix(selector, "Never_Has_") {
		return registration_axis{
			Message:      selector,
			Condition:    ast_expression_text(file_set, call),
			Kind:         Assertion_Kind_Always,
			Bucket_Count: 1,
		}, true
	}
	return registration_axis{}, false
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
	case "Always", "Sometimes", "Imply", "Dot_Product",
		"Recorder_Always", "Recorder_Sometimes", "Recorder_Imply",
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

// Resolves an Impossible's two Event_True("m")/Event_False("m") references — its two arguments —
// into (ungated grid position, bucket) cells, matching each reference's message literal to an
// ungated axis via position_of. ok is false — the carve skipped — when a reference names no
// ungated axis (the runtime precondition panics on a genuine non-sibling, so it is not a
// registration error) or is non-literal (recorded as a fatal diagnostic).
func ast_resolve_carve(
	file_set *token.FileSet, impossible *ast.CallExpr, position_of map[string]int,
	allow_unqualified bool, reg *registration,
) (cells []registration_cell, ok bool) {
	if len(impossible.Args) == 0 {
		return nil, false
	}
	for _, argument := range impossible.Args {
		reference, is_call := argument.(*ast.CallExpr)
		if !is_call {
			return nil, false
		}
		message, is_literal := ast_string_literal(reference, 0)
		if !is_literal {
			reg.Non_Literal = append(reg.Non_Literal,
				recorder_position(file_set, reference)+
					"  Impossible reference message is not a string literal")
			return nil, false
		}
		position, named := position_of[message]
		if !named {
			return nil, false
		}
		bucket := ast_event_bucket(ast_selector(reference, allow_unqualified))
		if bucket < 0 {
			return nil, false
		}
		cells = append(cells, registration_cell{Position: position, Bucket: bucket})
	}
	return cells, true
}

// Maps Event_True/Event_False to an ungated Sometimes axis bucket: false=0, true=1. Returns -1
// for any other selector.
func ast_event_bucket(selector string) (bucket int) {
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

// Returns the source text of the constructor's condition argument — at condition_index,
// past any leading recorder — for the never-fired report; "" when the call lacks it.
func ast_condition_text(
	file_set *token.FileSet, call *ast.CallExpr, condition_index int,
) (text string) {
	if len(call.Args) <= condition_index {
		return ""
	}
	return ast_expression_text(file_set, call.Args[condition_index])
}

// Returns the source text of expression, or "" when it can't be printed.
func ast_expression_text(file_set *token.FileSet, expression ast.Expr) (text string) {
	var buffer bytes.Buffer
	if printer.Fprint(&buffer, file_set, expression) != nil {
		return ""
	}
	return buffer.String()
}

// Seeds one tuple entry per bucket combination of the varying axes, skipping any tuple a
// carve forbids. Only a multi-bucket axis varies, so only it defines a combination; a
// single-bucket axis (an Always) is constant and carves out no coordinate of its own, so
// it is dropped from the grid — the coordinate carries only the axes that can vary, and the
// dropped Always keeps its coverage in its own per-element reachability entry. An all-Always
// Dot_Product therefore seeds nothing: there is no combination to cover.
func recorder_register_tuples(
	recorder *Recorder, prefix string, axes []registration_axis, carves [][]registration_cell,
) {
	if len(axes) == 0 {
		return
	}
	coordinate_positions := make([]int, 0, len(axes))
	for i, axis := range axes {
		if axis.Bucket_Count >= 2 {
			coordinate_positions = append(coordinate_positions, i)
		}
	}
	if len(coordinate_positions) == 0 {
		return
	}
	// The legend is one per grid, shared by every surviving cell: legend position j is the
	// varying axis the projected coordinate's position j stands for, so the report can name
	// a bare coordinate's positions without the runtime carrying any of this.
	legend := make([]Tuple_Axis, len(coordinate_positions))
	for j, position := range coordinate_positions {
		axis := axes[position]
		legend[j] = Tuple_Axis{
			Kind: axis.Kind, Condition: axis.Condition, Message: axis.Message}
	}
	// The odometer still runs the full axis list so an Impossible's carve positions, which
	// index that full list, stay valid; each surviving tuple is then projected onto the
	// varying axes for the stored coordinate and key.
	tuple := make([]int, len(axes))
	for more := true; more; more = recorder_tuple_increment(tuple, axes) {
		if recorder_tuple_carved(tuple, carves) {
			continue
		}
		projected := make([]int, len(coordinate_positions))
		for j, position := range coordinate_positions {
			projected[j] = tuple[position]
		}
		metadata := &Assertion_Metadata{
			Kind:          Assertion_Kind_Tuple,
			Message:       prefix,
			Tuple_Indices: projected,
			Axes:          legend,
		}
		recorder.Events.LoadOrStore(recorder_tuple_key(prefix, projected), metadata)
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

// Builds the tuple tracker key "<prefix>:tuple=(i0,i1,...)" — the Dot_Product message
// prefix joined to the projected coordinate. The runtime builds the identical key.
func recorder_tuple_key(prefix string, tuple []int) (key string) {
	return prefix + ":tuple=" + recorder_tuple_indices_text(tuple)
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
// fired — naming each by its message and condition source — then calls Exit(1) when
// any gap exists. It is a no-op in a benchmark or a fuzz worker subprocess; a plain test
// run and the fuzz coordinator both analyze.
func Recorder_Analyze_Assertion_Frequency(recorder *Recorder) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	if recorder.Is_Fuzz_Worker {
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
	recorder.Events.Range(func(key, value any) (continue_iteration bool) {
		metadata := value.(*Assertion_Metadata)
		gaps = append(gaps, assertion_metadata_gaps(metadata)...)
		return true
	})
	return gaps
}

// Returns the coverage gaps one assertion exhibits. A Sometimes contributes a gap
// per branch it never observed (true and/or false); an Always or Tuple that never
// fired is a single gap; a fully exercised assertion contributes none.
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
	recorder_report_cross_product(recorder.Output, gaps)
	recorder_report_section(&recorder_report_section_input{
		Output: recorder.Output, Title: "Branch gaps", Gaps: gaps,
		Kind: Assertion_Kind_Sometimes,
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
// kind, sorted by message. Emits nothing when no gap matches, so empty sections
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
	// Two gaps can share a message — a Sometimes missing both branches — so the Reason breaks
	// the tie. Without it the order rides on the tracker's unordered iteration and the report
	// is non-deterministic.
	sort.Slice(selected, func(i, j int) (less bool) {
		if selected[i].Metadata.Message != selected[j].Metadata.Message {
			return selected[i].Metadata.Message < selected[j].Metadata.Message
		}
		return selected[i].Reason < selected[j].Reason
	})
	fmt.Fprintln(input.Output)
	fmt.Fprintln(input.Output, "# "+input.Title)
	for _, gap := range selected {
		fmt.Fprintln(input.Output, coverage_gap_line(gap))
	}
}

// Prints the cross-product gaps grouped by their Dot_Product message prefix: each grid
// prints its axis legend once — every position named by kind, condition, and the axis's
// own message — then one line per never-observed cell, the bare bucket coordinate decoded
// back to each axis's event. A bare coordinate is undebuggable across nested bundles; the
// legend is what maps a position back to the axis it came from. Prefixes sort, and cells
// within a grid sort by their coordinate, so the report is deterministic despite the
// tracker's unordered iteration.
func recorder_report_cross_product(output io.Writer, gaps []coverage_gap) {
	by_prefix := map[string][]coverage_gap{}
	var prefixes []string
	for _, gap := range gaps {
		if gap.Metadata.Kind != Assertion_Kind_Tuple {
			continue
		}
		prefix := gap.Metadata.Message
		if _, seen := by_prefix[prefix]; !seen {
			prefixes = append(prefixes, prefix)
		}
		by_prefix[prefix] = append(by_prefix[prefix], gap)
	}
	if len(prefixes) == 0 {
		return
	}
	sort.Strings(prefixes)
	fmt.Fprintln(output)
	fmt.Fprintln(output, "# Cross-product gaps")
	for _, prefix := range prefixes {
		grid := by_prefix[prefix]
		sort.Slice(grid, func(i, j int) (less bool) {
			return recorder_tuple_indices_text(grid[i].Metadata.Tuple_Indices) <
				recorder_tuple_indices_text(grid[j].Metadata.Tuple_Indices)
		})
		recorder_report_grid_legend(output, prefix, grid[0].Metadata.Axes)
		for _, cell := range grid {
			fmt.Fprintln(output, coverage_gap_cell(cell))
		}
	}
}

// Prints "callsite  grid axes:" then one indented line per coordinate position, the kind
// and quoted condition columns padded to the grid's widest so the sites line up. Prints
// nothing when the legend is absent — a hand-seeded tuple with no axes still renders its
// bare coordinate.
func recorder_report_grid_legend(output io.Writer, prefix string, axes []Tuple_Axis) {
	if len(axes) == 0 {
		return
	}
	kind_width_count, condition_width_count := 0, 0
	for _, axis := range axes {
		kind_width_count = max(kind_width_count, len(assertion_kind_name(axis.Kind)))
		condition_width_count = max(
			condition_width_count, len(strconv.Quote(axis.Condition)))
	}
	fmt.Fprintln(output, prefix+"  grid axes:")
	for i, axis := range axes {
		fmt.Fprintf(output, "  [%d] %-*s %-*s from %s\n",
			i, kind_width_count, assertion_kind_name(axis.Kind),
			condition_width_count, strconv.Quote(axis.Condition), axis.Message)
	}
}

// Renders one never-observed cell: the bare bucket coordinate, then — when the legend is
// present — each position decoded back to its axis's event, so the coordinate reads as
// the combination it stands for rather than a tuple of indices.
func coverage_gap_cell(cell coverage_gap) (line string) {
	metadata := cell.Metadata
	line = metadata.Message + "  tuple " +
		recorder_tuple_indices_text(metadata.Tuple_Indices) + " " + cell.Reason
	if len(metadata.Axes) != len(metadata.Tuple_Indices) {
		return line
	}
	decoded := make([]string, len(metadata.Tuple_Indices))
	for position, index := range metadata.Tuple_Indices {
		decoded[position] = "[" + strconv.Itoa(position) + "]=" +
			assertion_kind_bucket_text(metadata.Axes[position].Kind, index)
	}
	return line + "  ->  " + strings.Join(decoded, " ")
}

// Decodes a bucket index for an axis of the given kind into the event it stands for: a
// Sometimes 0/1 into false/true, an Always into held (its one bucket means the condition held,
// the only outcome an Always records).
func assertion_kind_bucket_text(kind Assertion_Kind, index int) (text string) {
	if kind == Assertion_Kind_Always {
		return "held"
	}
	if index == 1 {
		return "true"
	}
	return "false"
}

// Renders one branch or reachability gap as a report line, naming its kind,
// reason, and condition source. Tuple gaps are rendered by recorder_report_cross_product,
// which carries the per-grid legend this line cannot.
func coverage_gap_line(gap coverage_gap) (line string) {
	metadata := gap.Metadata
	return message_display(metadata.Message) + "  " + assertion_kind_name(metadata.Kind) +
		" — " + gap.Reason + ": " + strconv.Quote(metadata.Condition)
}

// Renders a coverage key for the report: the element separator (the NUL joining a
// Dot_Product prefix to an axis message) shows as " · " so "signup.username␀empty" reads
// as "signup.username · empty". A bare message (an Always, or a grid prefix) is unchanged.
func message_display(message string) (display string) {
	return strings.ReplaceAll(message, Element_Message_Separator, " · ")
}

// Returns the report label for a kind: the same word the static pass keys on.
func assertion_kind_name(kind Assertion_Kind) (name string) {
	if kind == Assertion_Kind_Sometimes {
		return "Sometimes"
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
	recorder.Events.Range(func(key, value any) (continue_iteration bool) {
		metadata := value.(*Assertion_Metadata)
		if metadata.Kind == Assertion_Kind_Tuple {
			combinations++
		} else {
			individual++
		}
		if metadata.Kind == Assertion_Kind_Always {
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
	// A fuzz coordinator merges the workers' persisted coverage before analyzing — it never ran
	// the fuzzed body itself, so without this its grid would be empty (see Coverage / Modes).
	if recorder.Merge_Fuzz_Coverage != nil {
		recorder.Merge_Fuzz_Coverage()
	}
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
