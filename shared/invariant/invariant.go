// Package invariant exposes eager Always guards and demanded Dot_Product chains.
// Each chain requires the suite to witness every surviving combination of its
// Sometimes axes; Impossible carves and rejects combinations that cannot occur.
package invariant

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"go/ast"
	"go/constant"
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

// ASSERTION_FAILURE_MESSAGE_PREFIX opens every assertion-failure message.
const ASSERTION_FAILURE_MESSAGE_PREFIX = "🚨 Assertion Failure 🚨: "

// ELEMENT_MESSAGE_SEPARATOR makes structural axis keys unambiguous because registration rejects
// it in every namespace and message. Chain axes use two separators around the ordinal; the private
// Range and Enum engine retains its single-separator flat keys.
const ELEMENT_MESSAGE_SEPARATOR = "\x00"

// Bounds the bundle-flattening loop in recorder_collect_elements: each step
// either advances one argument cursor or pops a finished scope, so an acyclic
// bundle graph finishes far below this. The cap only stops a pathological (e.g.
// self-referential) *_Invariants graph from making the work depend unboundedly
// on input — TigerStyle forbids that.
const BUNDLE_EXPANSION_STEPS_MAX = 4096

// A uint8 ordinal represents the 255 valid link positions without widening Product.
const CHAIN_LINKS_MAX = 255

// Four uint64 words carry every axis representable by the chain's uint8 ordinal. The unused final
// bit is intentional: 255 is the number of valid link positions, not a smaller grid policy.
const CHAIN_MASK_WORDS = 4

// PRODUCT_FAILURE_LINKS preserves the distinct 255-link cap diagnostic until Ensure.
const PRODUCT_FAILURE_LINKS = uint8(1)

// PRODUCT_FAILURE_SOMETIMES distinguishes a malformed axis link from other builder failures.
const PRODUCT_FAILURE_SOMETIMES = uint8(2)

// PRODUCT_FAILURE_IMPOSSIBLE distinguishes a malformed constraint link at Ensure.
const PRODUCT_FAILURE_IMPOSSIBLE = uint8(3)

// PRODUCT_FAILURE_REFERENCE defers every malformed sibling reference to Ensure.
const PRODUCT_FAILURE_REFERENCE = uint8(4)

// PRODUCT_FAILURE_DUPLICATE_RULE keeps duplicate named constraints fatal without eager panic.
const PRODUCT_FAILURE_DUPLICATE_RULE = uint8(5)

// Bounds the walk up the directory tree searching for a go.mod, so module
// discovery can't loop unboundedly on a pathological path.
const MODULE_SEARCH_DEPTH_MAX = 256

// Dot_Element_Kind value 0 is intentionally unassigned: Always left the element algebra
// (it is now an eager guard, see Recorder_Always), and leaving the gap means a zero-value
// Dot_Element{} carries no valid kind — it matches none of the runtime branches rather
// than silently reading as a real kind.

// DOT_ELEMENT_KIND_SOMETIMES tags an element whose condition must be observed
// both true and false across the run.
const DOT_ELEMENT_KIND_SOMETIMES Engine_Element_Kind = 1

// DOT_ELEMENT_KIND_IMPOSSIBLE tags a declaration that a set of element events
// must never co-occur.
const DOT_ELEMENT_KIND_IMPOSSIBLE Engine_Element_Kind = 2

// ASSERTION_KIND_ALWAYS classifies a per-element tracker entry for an Always.
const ASSERTION_KIND_ALWAYS Assertion_Kind = 0

// ASSERTION_KIND_SOMETIMES classifies a per-element tracker entry for a Sometimes.
const ASSERTION_KIND_SOMETIMES Assertion_Kind = 1

// ASSERTION_KIND_TUPLE classifies a per-tuple entry of a Dot_Product's grid.
const ASSERTION_KIND_TUPLE Assertion_Kind = 2

// Recorder accumulates assertion observations for one run and identifies each
// element by its caller Site.
type Recorder struct {
	// File_System reads Go source files during AST analysis. Paths are absolute OS paths;
	// lookups strip the leading "/" before calling fs.ReadFile.
	File_System fs.FS

	// Events is the coverage tracker: one entry per registered element bucket and
	// per Dot_Product tuple, keyed by message and credited as observations arrive.
	Events sync.Map

	// Forbidden holds the grid cells an Impossible carves, keyed by the same
	// namespaced tuple key Events uses. A carved cell is never witnessed (so it is
	// not a coverage entry) but it is a panic-able property — reaching it fails
	// fatally — so the clean-run summary counts it. Seeded per call-site namespace,
	// one entry per cell the carve's glob expands to.
	Forbidden sync.Map

	// Observe_Cache_Mu guards Observe_Cache: the first-observe build takes the
	// write lock; the recording hot path reads under RLock.
	Observe_Cache_Mu sync.RWMutex
	// Observe_Cache memoizes, per Dot_Product message, the metadata pointers and
	// tracker keys its elements and grid cells resolve to, so the recording hot path
	// increments by pointer with no per-call key construction. Built lazily — the keys
	// and Loads happen once. A plain map (not sync.Map) keeps the read allocation-free.
	Observe_Cache map[string]*Observe_Handle

	// Enforce_Cache_Mu guards Enforce_Cache: the first-call build takes the write lock;
	// the enforcement hot path reads under RLock. The sibling of Observe_Cache_Mu.
	Enforce_Cache_Mu sync.RWMutex
	// Enforce_Cache memoizes, per Dot_Product message, each Impossible's references resolved
	// to sibling-axis positions plus its rendered violation message, so enforcement is
	// integer-index and boolean compares — never the per-call O(Impossibles x refs x axes)
	// string scan that resolving references from scratch would be. Enforcement runs in every
	// mode, so this cache is read on every call, unlike Observe_Cache which only the recording
	// modes touch. Built lazily; a plain map keyed by the existing message string reads
	// allocation-free.
	Enforce_Cache map[string]*Enforce_Handle

	// Chain_Shapes_Mu guards discovery because foreign chains have no registration phase to
	// publish an immutable shape before concurrent execution.
	Chain_Shapes_Mu sync.RWMutex
	// Chain_Shapes is keyed by namespace because one namespace names exactly one chain.
	// Product retains the resolved pointer so fluent links do not repeat the map lookup.
	Chain_Shapes map[Namespace]*Chain_Shape
	// Chain_Entries resolves the exact serialized axis keys fuzz workers persist. Registration
	// publishes it before the suite, so merge needs no inverse identity algorithm.
	Chain_Entries map[string]*Assertion_Metadata

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

	// Package_Label is the package's path relative to the module root, derived by
	// Recorder_Register_Packages_For_Analysis and printed in the clean-run summary so
	// the line is identifiable when many packages print to the same terminal. Empty
	// when no go.mod is found or when the registered directory is the module root.
	Package_Label string

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

// Engine_Element survives only because Range and Enum share the original allocation-free grid
// engine; keeping that adapter separate prevents its representation from leaking into Product.
type Engine_Element struct {
	// Kind selects which fields carry meaning: a Sometimes axis or an Impossible declaration.
	Kind Engine_Element_Kind
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

// Engine_Bundle keeps the legacy engine slice allocation-compatible with its preset builders.
type Engine_Bundle = []Engine_Element

// Engine_Element_Kind keeps the preset adapter's zero value invalid.
type Engine_Element_Kind uint8

// Dot_Element_Reference names one element's event by its Message — a coordinate an
// Impossible declares forbidden.
type Dot_Element_Reference struct {
	// Message is the referenced element's own message.
	Message string
	// Event is the outcome of that element this reference names: true for its true event.
	Event bool
}

// Chain_Mask is the packed tuple and rule representation for every axis a chain can contain.
// Arrays remain comparable, so registered tuple resolution is a direct allocation-free map read.
type Chain_Mask [CHAIN_MASK_WORDS]uint64

// Chain_Shape is the immutable execution plan for one registered namespace. A foreign chain builds
// the same plan under Mu on its first execution so shape and enforcement errors behave identically
// even though its coverage entries remain nil.
type Chain_Shape struct {
	// Mu protects discovery of an unregistered, enforcement-only shape.
	Mu sync.Mutex
	// Axes resolves packed mask positions to fluent identities.
	Axes []Chain_Axis
	// Links pins the complete structural sequence shared by a namespace.
	Links []Chain_Link
	// Rules holds every polar carve in declaration order.
	Rules []Chain_Rule
	// Tuples resolves a packed mask directly to its pre-seeded coverage entry.
	// The map prevents a dense array from imposing a width smaller than the chain ordinal.
	Tuples map[Chain_Mask]Handle_Entry
	// Registered distinguishes analyzed chains from enforcement-only foreign chains.
	Registered bool
	// Ensured prevents a discovered shape from growing after its first complete execution.
	Ensured bool
}

// Chain_Axis is registration's complete execution plan for one Sometimes link.
type Chain_Axis struct {
	// Ordinal selects the raw condition captured at this fluent link.
	Ordinal uint8
	// Tuple_Position is registration's exclusive assignment of this axis to the packed grid.
	Tuple_Position uint8
	// Message is retained for runtime sibling-reference resolution.
	Message string
	// Entry is the exact registration-seeded coverage handle Ensure credits.
	Entry Handle_Entry
}

// Chain_Link pins the structural identity of one fluent link. References are retained only for an
// Impossible because polarity and declaration order are part of a namespace's shape.
type Chain_Link struct {
	// Kind distinguishes axis and constraint links.
	Kind Engine_Element_Kind
	// Ordinal prevents a fluent reorder from replaying against an older shape.
	Ordinal uint8
	// Axis_Count resolves references against only the siblings preceding this link.
	Axis_Count uint8
	// Message is either the axis claim or named constraint.
	Message string
	// References retains referenced axis positions in argument order without retaining strings.
	References [CHAIN_LINKS_MAX]uint8
	// Reference_Events packs each argument's polarity by argument position.
	Reference_Events Chain_Mask
	// Reference_Count distinguishes unused fixed-array cells from real coordinates.
	Reference_Count uint8
}

// Chain_Rule is one Impossible compiled to the packed-mask predicate mask&Mask == Want.
type Chain_Rule struct {
	// Mask selects every named axis and globs over all others.
	Mask Chain_Mask
	// Want contains the selected axes' forbidden polarities.
	Want Chain_Mask
	// Message names the constraint when it fires.
	Message string
	// Ordinal maps the compiled rule back to its fluent link.
	Ordinal uint8
}

// Product is the register-sized fluent value for one Dot_Product execution. Value receivers return
// advanced copies; no method takes its address, making an escaping chain unrepresentable by shape.
type Product struct {
	// Recorder receives the complete call only after Ensure accepts it.
	Recorder *Recorder
	// Shape is resolved once at the root and replayed by every value copy.
	Shape *Chain_Shape
	// Namespace is carried because it is part of every axis identity.
	Namespace Namespace
	// Ordinal is the next fluent link position.
	Ordinal uint8
	// Axis_Count counts Sometimes links for sibling-reference validation.
	Axis_Count uint8
	// Observations retains outcomes by fluent ordinal before registration projects them onto
	// its tuple positions at Ensure.
	Observations Chain_Mask
	// Failure retains the first malformed link so only Ensure exposes it.
	Failure uint8
	// Mismatch retains structural divergence so only Ensure exposes it.
	Mismatch bool
}

// Recorder_Always stays outside Product because an eager guard has no second branch to widen a
// demanded grid. It panics immediately when condition is false in every run mode; under a plain
// test run it also credits reachability so an uncalled guard remains visible as a gap.
func Recorder_Always[T ~bool](recorder *Recorder, condition T, message string) {
	if !condition {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + message + "  Always — condition was false")
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
func recorder_sometimes[T ~bool](
	recorder *Recorder, condition T, message string,
) (element Engine_Element) {
	return Engine_Element{
		Kind:    DOT_ELEMENT_KIND_SOMETIMES,
		Event:   bool(condition),
		Message: message,
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
func impossible(impossibles ...Dot_Element_Reference) (element Engine_Element) {
	return Engine_Element{Kind: DOT_ELEMENT_KIND_IMPOSSIBLE, Impossibles: impossibles}
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

// Namespace is a Dot_Product's grid identity — the literal a caller supplies, prefixed onto each
// held axis's own message to form that axis's coverage key. It is its own type, not a bare string,
// because it names a coverage grid rather than carrying arbitrary text; a caller always passes it
// as an inline string literal, which converts to Namespace without ceremony.
type Namespace string

// Recorder_Dot_Product starts one demanded chain under namespace.
func Recorder_Dot_Product(recorder *Recorder, namespace Namespace) (product Product) {
	if strings.Contains(string(namespace), ELEMENT_MESSAGE_SEPARATOR) {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + "Dot_Product namespace contains NUL")
	}
	return Product{
		Recorder:  recorder,
		Shape:     recorder_chain_shape(recorder, namespace),
		Namespace: namespace,
	}
}

// Sometimes only advances the builder; Ensure owns validation and coverage mutation.
func (product Product) Sometimes(condition bool, message string) (next Product) {
	if product.Failure != 0 {
		return product
	}
	if product.Ordinal == CHAIN_LINKS_MAX {
		return product.chain_defer_failure(PRODUCT_FAILURE_LINKS)
	}
	if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
		return product.chain_defer_failure(PRODUCT_FAILURE_SOMETIMES)
	}
	mismatch := product.Shape.chain_axis(&Chain_Axis_Input{
		Ordinal: product.Ordinal, Axis_Count: product.Axis_Count, Message: message,
	})
	if condition {
		product.Observations = chain_mask_with(product.Observations, product.Ordinal)
	}
	product.Ordinal++
	product.Axis_Count++
	if mismatch {
		product.Mismatch = true
	}
	return product
}

// Impossible only advances the builder; Ensure owns validation and enforcement.
func (product Product) Impossible(
	message string, references ...Dot_Element_Reference,
) (next Product) {
	if product.Failure != 0 {
		return product
	}
	if product.Ordinal == CHAIN_LINKS_MAX {
		return product.chain_defer_failure(PRODUCT_FAILURE_LINKS)
	}
	if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
		return product.chain_defer_failure(PRODUCT_FAILURE_IMPOSSIBLE)
	}
	if len(references) == 0 {
		return product.chain_defer_failure(PRODUCT_FAILURE_IMPOSSIBLE)
	}
	if len(references) > CHAIN_LINKS_MAX {
		return product.chain_defer_failure(PRODUCT_FAILURE_IMPOSSIBLE)
	}
	link := Chain_Link{
		Kind: DOT_ELEMENT_KIND_IMPOSSIBLE, Ordinal: product.Ordinal,
		Axis_Count: product.Axis_Count, Message: message,
	}
	if product.Shape.chain_replays() {
		rule, mismatch, failure := product.Shape.chain_rule_registered(link, references)
		if failure != 0 {
			product = product.chain_defer_failure(failure)
		}
		return product.impossible_advance(rule, mismatch)
	}
	rule, mismatch, failure := product.Shape.chain_rule(link, references)
	if failure != 0 {
		product = product.chain_defer_failure(failure)
	}
	return product.impossible_advance(rule, mismatch)
}

// Keeping advancement outside both resolution branches lets the warmed branch's input stay on the
// stack even though first-execution discovery must retain a copy of its references.
func (product Product) impossible_advance(
	rule Chain_Rule, mismatch bool,
) (next Product) {
	if chain_mask_empty(rule.Mask) {
		product = product.chain_defer_failure(PRODUCT_FAILURE_REFERENCE)
	}
	product.Ordinal++
	if mismatch {
		product.Mismatch = true
	}
	return product
}

// Retaining only the category keeps malformed links inert until Ensure owns the diagnostic.
func (product Product) chain_defer_failure(failure uint8) (next Product) {
	if product.Failure == 0 {
		product.Failure = failure
	}
	return product
}

func (product Product) chain_failure_message() (message string) {
	switch product.Failure {
	case PRODUCT_FAILURE_LINKS:
		return "Dot_Product exceeds 255 links"
	case PRODUCT_FAILURE_SOMETIMES:
		return "Sometimes link is invalid"
	case PRODUCT_FAILURE_IMPOSSIBLE:
		return "Impossible link is invalid"
	case PRODUCT_FAILURE_REFERENCE:
		return "Impossible reference does not name one unique preceding axis"
	case PRODUCT_FAILURE_DUPLICATE_RULE:
		return "duplicate Impossible message"
	}
	return ""
}

// Registration and a completed foreign discovery both make the shape immutable; checking the
// latter under its mutex avoids racing two first executions while keeping every replay read-only.
func (shape *Chain_Shape) chain_replays() (replays bool) {
	if shape.Registered {
		return true
	}
	shape.Mu.Lock()
	replays = shape.Ensured
	shape.Mu.Unlock()
	return replays
}

// Ensure validates the complete shape, enforces every carve, and credits the packed tuple.
func (product Product) Ensure() {
	if failure := product.chain_failure_message(); failure != "" {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + failure)
	}
	axis_count := product.Axis_Count
	if axis_count == 0 {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + "Dot_Product has no axes")
	}
	if product.Mismatch {
		message := "Dot_Product shape differs for namespace " +
			strconv.Quote(string(product.Namespace))
		if product.Shape.Registered {
			message = "registered Dot_Product credited unknown axis or shape differs"
		}
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + message)
	}
	if !product.Shape.chain_ensure(product) {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX +
			"Dot_Product shape differs for namespace " +
			strconv.Quote(string(product.Namespace)))
	}
	records := recorder_chain_records(product.Recorder)
	if !product.Shape.Registered {
		records = false
	}
	if len(product.Shape.Rules) == 0 {
		if !records {
			return
		}
	}
	tuple_mask := product.Shape.chain_tuple(product.Observations)
	var violations []string
	for _, rule := range product.Shape.Rules {
		matches := true
		for i_index := range tuple_mask {
			if tuple_mask[i_index]&rule.Mask[i_index] != rule.Want[i_index] {
				matches = false
				break
			}
		}
		if matches {
			violations = append(violations, rule.Message)
		}
	}
	if len(violations) > 0 {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + strings.Join(violations, "\n"))
	}
	if !records {
		return
	}
	tuple := product.chain_handle(tuple_mask)
	if tuple.Metadata == nil {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX +
			"registered Dot_Product credited unknown tuple")
	}
	for i_index := uint8(0); i_index < axis_count; i_index++ {
		axis := product.Shape.Axes[i_index]
		condition := chain_mask_has(product.Observations, axis.Ordinal)
		recorder_increment_entry(product.Recorder, axis.Entry, condition)
	}
	recorder_increment_entry(product.Recorder, tuple, true)
}

// Resolving every handle before crediting keeps an unknown key from partially crediting the call.
func (product Product) chain_handle(tuple_mask Chain_Mask) (tuple Handle_Entry) {
	for _, axis := range product.Shape.Axes {
		if axis.Entry.Metadata == nil {
			message := "registered Dot_Product credited unknown axis " +
				strconv.Quote(axis.Message)
			panic(ASSERTION_FAILURE_MESSAGE_PREFIX + message)
		}
	}
	tuple = product.Shape.Tuples[tuple_mask]
	return tuple
}

// Registration alone assigns tuple positions; runtime carries only link-ordinal observations.
func (shape *Chain_Shape) chain_tuple(observations Chain_Mask) (tuple Chain_Mask) {
	for _, axis := range shape.Axes {
		if chain_mask_has(observations, axis.Ordinal) {
			tuple = chain_mask_with(tuple, axis.Tuple_Position)
		}
	}
	return tuple
}

func chain_mask_with(mask Chain_Mask, position uint8) (next Chain_Mask) {
	mask[position/64] |= uint64(1) << (position % 64)
	return mask
}

func chain_mask_has(mask Chain_Mask, position uint8) (has bool) {
	return mask[position/64]&(uint64(1)<<(position%64)) != 0
}

func chain_mask_empty(mask Chain_Mask) (empty bool) {
	return mask == Chain_Mask{}
}

// Registration publishes shapes before execution, while foreign packages still need the same
// structural enforcement; a per-recorder cache makes both paths converge on one immutable plan.
func recorder_chain_shape(recorder *Recorder, namespace Namespace) (shape *Chain_Shape) {
	recorder.Chain_Shapes_Mu.RLock()
	shape = recorder.Chain_Shapes[namespace]
	recorder.Chain_Shapes_Mu.RUnlock()
	if shape != nil {
		return shape
	}
	recorder.Chain_Shapes_Mu.Lock()
	defer recorder.Chain_Shapes_Mu.Unlock()
	if shape = recorder.Chain_Shapes[namespace]; shape != nil {
		return shape
	}
	shape = &Chain_Shape{}
	if recorder.Chain_Shapes == nil {
		recorder.Chain_Shapes = map[Namespace]*Chain_Shape{}
	}
	recorder.Chain_Shapes[namespace] = shape
	return shape
}

// Coverage is deliberately absent in benchmarks and non-test binaries, but enforcement never is.
func recorder_chain_records(recorder *Recorder) (records bool) {
	if !recorder.Is_Test {
		return false
	}
	return !recorder.Is_Benchmark
}

// Two separators distinguish an axis key from every flat tuple and Always key during fuzz merge.
func chain_axis_key(namespace Namespace, ordinal uint8, message string) (value string) {
	return string(namespace) + ELEMENT_MESSAGE_SEPARATOR +
		strconv.Itoa(int(ordinal)) + ELEMENT_MESSAGE_SEPARATOR + message
}

// Chain_Axis_Input keeps exact shape capture possible without widening Product to retain messages.
type Chain_Axis_Input struct {
	// Ordinal separates fluent position from registration's independently assigned
	// tuple position.
	Ordinal uint8
	// Axis_Count limits sibling resolution to axes preceding this link.
	Axis_Count uint8
	// Message is available only during the call, so shape capture must consume it here.
	Message string
}

// Registered shapes are immutable and can use direct reads without synchronization; only
// first-executed foreign shapes pay the mutex needed to discover their link sequence.
func (shape *Chain_Shape) chain_axis(input *Chain_Axis_Input) (mismatch bool) {
	ordinal := input.Ordinal
	axis_count := input.Axis_Count
	message := input.Message
	if shape.Registered {
		if int(ordinal) >= len(shape.Links) {
			return true
		}
		link := shape.Links[ordinal]
		mismatch = link.Kind != DOT_ELEMENT_KIND_SOMETIMES
		if link.Message != message {
			mismatch = true
		}
		if int(axis_count) >= len(shape.Axes) {
			return true
		}
		axis := shape.Axes[axis_count]
		if axis.Ordinal != ordinal {
			mismatch = true
		}
		if axis.Message != message {
			mismatch = true
		}
		return mismatch
	}
	shape.Mu.Lock()
	defer shape.Mu.Unlock()
	link := Chain_Link{
		Kind: DOT_ELEMENT_KIND_SOMETIMES, Ordinal: ordinal,
		Axis_Count: axis_count, Message: message,
	}
	if int(ordinal) < len(shape.Links) {
		expected := shape.Links[ordinal]
		mismatch = expected.Kind != link.Kind
		if expected.Message != link.Message {
			mismatch = true
		}
	} else if !shape.Ensured {
		if int(ordinal) == len(shape.Links) {
			shape.Links = append(shape.Links, link)
		} else {
			mismatch = true
		}
	} else {
		mismatch = true
	}
	axis := Chain_Axis{
		Ordinal: ordinal, Tuple_Position: axis_count, Message: message,
	}
	if int(axis_count) < len(shape.Axes) {
		expected := shape.Axes[axis_count]
		if !expected.chain_equal(axis) {
			mismatch = true
		}
	} else if !shape.Ensured {
		if int(axis_count) == len(shape.Axes) {
			shape.Axes = append(shape.Axes, axis)
		} else {
			mismatch = true
		}
	} else {
		mismatch = true
	}
	return mismatch
}

// Entry is registration-owned output, not part of the structure a foreign execution replays.
func (first Chain_Axis) chain_equal(second Chain_Axis) (equal bool) {
	if first.Ordinal != second.Ordinal {
		return false
	}
	if first.Tuple_Position != second.Tuple_Position {
		return false
	}
	return first.Message == second.Message
}

// A reference has no ordinal in the public vocabulary, so repeated messages are safe until a rule
// tries to name them; requiring one unique preceding match prevents a carve from changing meaning.
func (shape *Chain_Shape) chain_references(
	axis_count uint8, references []Dot_Element_Reference,
) (
	positions [CHAIN_LINKS_MAX]uint8, events Chain_Mask, failure uint8,
) {
	if int(axis_count) > len(shape.Axes) {
		failure = PRODUCT_FAILURE_REFERENCE
		return positions, events, failure
	}
	var referenced Chain_Mask
	for reference_index, reference := range references {
		if strings.Contains(reference.Message, ELEMENT_MESSAGE_SEPARATOR) {
			failure = PRODUCT_FAILURE_REFERENCE
			return positions, events, failure
		}
		matches := 0
		position := 0
		for i_index := 0; i_index < int(axis_count); i_index++ {
			if shape.Axes[i_index].Message != reference.Message {
				continue
			}
			matches++
			position = i_index
		}
		if matches != 1 {
			failure = PRODUCT_FAILURE_REFERENCE
			return positions, events, failure
		}
		axis_position := uint8(position)
		if chain_mask_has(referenced, axis_position) {
			failure = PRODUCT_FAILURE_REFERENCE
			return positions, events, failure
		}
		referenced = chain_mask_with(referenced, axis_position)
		positions[reference_index] = axis_position
		if reference.Event {
			events = chain_mask_with(events, uint8(reference_index))
		}
	}
	return positions, events, 0
}

func (shape *Chain_Shape) chain_resolve_link(
	link Chain_Link, references []Dot_Element_Reference,
) (resolved Chain_Link, failure uint8) {
	positions, events, failure := shape.chain_references(link.Axis_Count, references)
	if failure != 0 {
		return link, failure
	}
	link.References = positions
	link.Reference_Events = events
	link.Reference_Count = uint8(len(references))
	return link, 0
}

func (shape *Chain_Shape) chain_resolve_rule(
	link Chain_Link, references []Dot_Element_Reference,
) (resolved Chain_Link, rule Chain_Rule, failure uint8) {
	link, failure = shape.chain_resolve_link(link, references)
	if failure != 0 {
		return link, rule, failure
	}
	var mask Chain_Mask
	var want Chain_Mask
	for reference_index := uint8(0); reference_index < link.Reference_Count; reference_index++ {
		axis := shape.Axes[link.References[reference_index]]
		mask = chain_mask_with(mask, axis.Tuple_Position)
		if chain_mask_has(link.Reference_Events, reference_index) {
			want = chain_mask_with(want, axis.Tuple_Position)
		}
	}
	rule = Chain_Rule{
		Mask: mask, Want: want, Message: link.Message, Ordinal: link.Ordinal,
	}
	return link, rule, 0
}

// Constraints are resolved while the fluent value still carries the number of preceding axes;
// this makes a forward or non-sibling reference fail before it can erase a demanded obligation.
func (shape *Chain_Shape) chain_rule(
	link Chain_Link, references []Dot_Element_Reference,
) (rule Chain_Rule, mismatch bool, failure uint8) {
	if shape.Registered {
		return shape.chain_rule_registered(link, references)
	}
	shape.Mu.Lock()
	defer shape.Mu.Unlock()
	link, rule, failure = shape.chain_resolve_rule(link, references)
	if failure != 0 {
		return rule, false, failure
	}
	for _, extant := range shape.Rules {
		if extant.Ordinal >= link.Ordinal {
			continue
		}
		if extant.Message == link.Message {
			return rule, false, PRODUCT_FAILURE_DUPLICATE_RULE
		}
	}
	if int(link.Ordinal) < len(shape.Links) {
		mismatch = !shape.Links[link.Ordinal].chain_equal(link)
		for _, extant := range shape.Rules {
			if extant.Ordinal == link.Ordinal {
				return extant, mismatch, 0
			}
		}
		return rule, true, 0
	}
	if !shape.Ensured {
		if int(link.Ordinal) == len(shape.Links) {
			shape.Links = append(shape.Links, link)
			shape.Rules = append(shape.Rules, rule)
		} else {
			mismatch = true
		}
	} else {
		mismatch = true
	}
	return rule, mismatch, 0
}

// The registered path compares the call's stack-backed references to prebuilt values and never
// retains or boxes them, keeping enforcement allocation-free.
func (shape *Chain_Shape) chain_rule_registered(
	link Chain_Link, references []Dot_Element_Reference,
) (rule Chain_Rule, mismatch bool, failure uint8) {
	link, failure = shape.chain_resolve_link(link, references)
	if failure != 0 {
		return rule, false, failure
	}
	if int(link.Ordinal) >= len(shape.Links) {
		return rule, true, 0
	}
	mismatch = !shape.Links[link.Ordinal].chain_equal(link)
	for _, extant := range shape.Rules {
		if extant.Ordinal == link.Ordinal {
			return extant, mismatch, 0
		}
	}
	return rule, true, 0
}

func (first Chain_Link) chain_equal(second Chain_Link) (equal bool) {
	if first.Kind != second.Kind {
		return false
	}
	if first.Ordinal != second.Ordinal {
		return false
	}
	if first.Axis_Count != second.Axis_Count {
		return false
	}
	if first.Message != second.Message {
		return false
	}
	if first.Reference_Count != second.Reference_Count {
		return false
	}
	for i_index := uint8(0); i_index < first.Reference_Count; i_index++ {
		if first.References[i_index] != second.References[i_index] {
			return false
		}
		if chain_mask_has(first.Reference_Events, i_index) !=
			chain_mask_has(second.Reference_Events, i_index) {
			return false
		}
	}
	return true
}

// Ensuring freezes a discovered shape so a shared namespace cannot merge two products.
func (shape *Chain_Shape) chain_ensure(product Product) (matches bool) {
	axis_count := product.Axis_Count
	if shape.Registered {
		if len(shape.Links) != int(product.Ordinal) {
			return false
		}
		return len(shape.Axes) == int(axis_count)
	}
	shape.Mu.Lock()
	defer shape.Mu.Unlock()
	matches = len(shape.Links) == int(product.Ordinal)
	if len(shape.Axes) != int(axis_count) {
		matches = false
	}
	if matches {
		shape.Ensured = true
	}
	return matches
}

// Recorder_dot_product remains private so Range and Enum can retain their compact engine without
// exposing a second product API. Every axis
// violated on the call is named in one panic, not just the first, so a single run surfaces
// them all. namespace is the grid's identity and is prefixed onto each held axis's own message
// to form that axis's coverage key.
func recorder_dot_product(recorder *Recorder, namespace Namespace, bundle ...Engine_Element) {
	// A Dot_Product with no elements asserts nothing — a no-op grid is always a
	// mistake, so it fails immediately rather than silently recording nothing.
	if len(bundle) == 0 {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + "Dot_Product has no elements")
	}
	message := string(namespace)
	handle := recorder_enforce_handle(recorder, message, bundle)
	var violations []string
	for _, rule := range handle.Rules {
		fired := true
		for _, coordinate := range rule.Coordinates {
			if bundle[coordinate.Index].Event != coordinate.Event {
				fired = false
				break
			}
		}
		if fired {
			violations = append(violations, rule.Message)
		}
	}
	if len(violations) > 0 {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + strings.Join(violations, "\n"))
	}
	recorder_dot_product_observe(recorder, message, bundle)
}

// Integer constrains a Range value to the integer kinds — the bounded-newtype family whose guard
// preamble Range collapses. Floats are excluded: their boundary claims are NaN and the infinities,
// not the integer units, so they stay with Float64_Invariants.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// RANGE_MESSAGE_ZERO is Range's coverage message for the zero boundary. It is exported so the
// static registration seeds the identical key the runtime stamps — the rendezvous every axis
// message relies on. The wording matches the hand-written numeric presets.
const RANGE_MESSAGE_ZERO = "The value is zero."

// RANGE_MESSAGE_ONE is Range's coverage message for the one boundary.
const RANGE_MESSAGE_ONE = "The value is one."

// RANGE_MESSAGE_TWO is Range's coverage message for the two boundary.
const RANGE_MESSAGE_TWO = "The value is two."

// RANGE_MESSAGE_NEGATIVE_ONE is Range's coverage message for the negative-one boundary.
const RANGE_MESSAGE_NEGATIVE_ONE = "The value is negative one."

// RANGE_MESSAGE_MINIMUM is Range's coverage message for the interval's lower edge, the value MIN.
const RANGE_MESSAGE_MINIMUM = "The value is the minimum."

// RANGE_MESSAGE_MAXIMUM is Range's coverage message for the interval's upper edge, the value MAX.
const RANGE_MESSAGE_MAXIMUM = "The value is the maximum."

// RANGE_GUARD_UPPER labels Range's upper-bound guard. Each guard is keyed by the callsite namespace
// (namespace + separator + label), so a shared preset never collides the way a global literal
// message would; exported for the same static/runtime rendezvous as the axes.
const RANGE_GUARD_UPPER = "at most max"

// RANGE_GUARD_LOWER labels Range's lower-bound guard; keyed by namespace like RANGE_GUARD_UPPER.
const RANGE_GUARD_LOWER = "at least min"

// RANGE_GUARD_EXCLUDED labels the enforcement that an excluded (declared-unreachable) in-range
// value never occurs — the guard a holed interval or an enum adds beyond its two bounds.
const RANGE_GUARD_EXCLUDED = "excluded"

// Recorder_Range is the bounded-integer preset: it enforces value ∈ [minimum, maximum] as two
// eager bound guards and self-emits, under namespace, the coverage grid for whichever of
// {0, 1, 2, -1} the interval admits. It collapses the mandated bound preamble and boundary claims
// into one call, keyed by the per-callsite namespace so it reuses across every bounded newtype.
//
// The three magnitudes take their own type parameters, so a caller passes them all positionally
// without the repeated-type ban firing; at a real callsite they are one type, so the conversions
// to Value are identities.
//
// A boundary unit outside the interval is dropped, not witnessed — the guard already forbids it,
// so a Sometimes on it would be an unfillable gap; -1 is dropped for an unsigned value. When the
// interval admits none of the four, no grid is emitted and only the two guards register.
//
// An excluded value is an in-range value the caller declares unreachable — a hole in the interval.
// Each is enforced (reaching it panics like a bound violation) and drops its sentinel axis, and
// once every reachable value is a witnessed axis the all-false cell is carved. The variadic is only
// ranged over, never retained, so the empty and holed calls alike allocate nothing beyond the grid.
func Recorder_Range[Value Integer, Minimum Integer, Maximum Integer](
	recorder *Recorder, value Value, minimum Minimum, maximum Maximum, namespace Namespace,
	excluded ...Value,
) {
	bounds := [2]Value{Value(minimum), Value(maximum)}
	// Enforcement runs in every mode, like Recorder_Always — a bound violation is fatal on the
	// spot, naming the guard it breached under the callsite namespace.
	if value > bounds[1] {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + string(namespace) +
			ELEMENT_MESSAGE_SEPARATOR + RANGE_GUARD_UPPER + "  value exceeds max")
	}
	if value < bounds[0] {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + string(namespace) +
			ELEMENT_MESSAGE_SEPARATOR + RANGE_GUARD_LOWER + "  value below min")
	}
	for _, hole := range excluded {
		if value != hole {
			continue
		}
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + string(namespace) +
			ELEMENT_MESSAGE_SEPARATOR + RANGE_GUARD_EXCLUDED + "  value is excluded")
	}
	recorder_range_credit_guards(recorder, namespace)
	bundle, axis_count := recorder_range_bundle(recorder, value, bounds, excluded)
	// An interval admitting none of the four boundary units has no cell to witness — an empty
	// Dot_Product would panic. The guards alone carry the reachability obligation in that case.
	if axis_count == 0 {
		return
	}
	recorder_dot_product(recorder, namespace, bundle...)
}

// Credits both bound guards' reachability under the recording-mode gate, mirroring Recorder_Always:
// enforcement is unconditional, but coverage is tracked only under a test run.
func recorder_range_credit_guards(recorder *Recorder, namespace Namespace) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	recorder_increment(recorder,
		string(namespace)+ELEMENT_MESSAGE_SEPARATOR+RANGE_GUARD_UPPER, true)
	recorder_increment(recorder,
		string(namespace)+ELEMENT_MESSAGE_SEPARATOR+RANGE_GUARD_LOWER, true)
}

// Builds Recorder_Range's grid: a Sometimes axis for each interval edge (min and max — the whole
// point of a bounded type) and for each of {0, 1, 2, -1} strictly inside the interval, all mutually
// exclusive. A single-value interval (min == max) has no edge axis and no interior, so it seeds an
// empty grid and only the guards carry it. -1 is derived as 0-1 and admitted only for a signed
// value, so an unsigned value's wrap of 0-1 to its maximum can never masquerade as -1.
func recorder_range_bundle[Value Integer](
	recorder *Recorder, value Value, bounds [2]Value, excluded []Value,
) (bundle []Engine_Element, axis_count int) {
	elements, messages := recorder_range_axes(recorder, value, bounds, excluded, false)
	if recorder_range_saturated(bounds, len(messages), excluded) {
		elements = append(elements, recorder_range_all_false(messages))
	}
	return elements, len(messages)
}

// Builds the axes Range and Enum share: a Sometimes for each interval edge and for each of
// {0,1,2,-1} strictly inside, all mutually exclusive. A sentinel is admitted by its membership in
// `set`: for a range `set` is the excluded holes and an in-set sentinel is dropped; for an enum
// `set` is the members and an out-of-set sentinel drops — `recorder_range_holed(c, set) != enum`.
// The all-false carve is the caller's, which alone knows how its reachable count meets the axes.
func recorder_range_axes[Value Integer](
	recorder *Recorder, value Value, bounds [2]Value, set []Value, enum bool,
) (bundle []Engine_Element, messages []string) {
	if bounds[0] < bounds[1] {
		edge := recorder_sometimes(recorder, value == bounds[0], RANGE_MESSAGE_MINIMUM)
		bundle = append(bundle, edge)
		messages = append(messages, RANGE_MESSAGE_MINIMUM)
		edge = recorder_sometimes(recorder, value == bounds[1], RANGE_MESSAGE_MAXIMUM)
		bundle = append(bundle, edge)
		messages = append(messages, RANGE_MESSAGE_MAXIMUM)
	}
	negative_one := Value(0) - Value(1)
	signed := negative_one < Value(0)
	candidates := [4]Value{Value(0), Value(1), Value(2), negative_one}
	interior := [4]bool{
		recorder_range_interior(bounds, candidates[0]),
		recorder_range_interior(bounds, candidates[1]),
		recorder_range_interior(bounds, candidates[2]),
		signed && recorder_range_interior(bounds, candidates[3]),
	}
	units := recorder_range_units()
	for i := range units {
		if !interior[i] {
			continue
		}
		if recorder_range_holed(candidates[i], set) != enum {
			continue
		}
		axis := recorder_sometimes(recorder, value == candidates[i], units[i].Message)
		bundle = append(bundle, axis)
		messages = append(messages, units[i].Message)
	}
	bundle = append(bundle, recorder_range_carves(messages)...)
	return bundle, messages
}

// Recorder_Enum is the discrete-set preset: the reachable values are exactly `members`. It is
// Recorder_Range over the span [min(members), max(members)] with every in-span non-member a hole —
// each member witnessed as that range would, every non-member enforced, and the all-false cell
// carved once every member is an axis. The variadic is only ranged over, so nothing escapes.
func Recorder_Enum[Value Integer](
	recorder *Recorder, value Value, namespace Namespace, members ...Value,
) {
	bounds := recorder_enum_bounds(members)
	member := false
	for _, candidate := range members {
		if value == candidate {
			member = true
		}
	}
	if !member {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + string(namespace) +
			ELEMENT_MESSAGE_SEPARATOR + RANGE_GUARD_EXCLUDED + "  not a member")
	}
	recorder_range_credit_guards(recorder, namespace)
	elements, messages := recorder_range_axes(recorder, value, bounds, members, true)
	// Every member is a witnessed axis exactly when the axis count matches the member count, so
	// a reachable value witnessing none can never occur and the all-false cell is carved.
	if len(members) <= len(messages) {
		elements = append(elements, recorder_range_all_false(messages))
	}
	if len(messages) == 0 {
		return
	}
	recorder_dot_product(recorder, namespace, elements...)
}

// Computes the enum's span — the least and greatest member. An empty member set yields a zero span
// that Recorder_Enum's membership loop rejects for every value.
func recorder_enum_bounds[Value Integer](members []Value) (bounds [2]Value) {
	for i, member := range members {
		if i == 0 {
			bounds = [2]Value{member, member}
			continue
		}
		if member < bounds[0] {
			bounds[0] = member
		}
		if member > bounds[1] {
			bounds[1] = member
		}
	}
	return bounds
}

// Reports whether candidate is one of the excluded holes. Ranges the variadic without retaining it.
func recorder_range_holed[Value Integer](candidate Value, excluded []Value) (holed bool) {
	for _, hole := range excluded {
		if candidate == hole {
			return true
		}
	}
	return false
}

// Reports whether the witnessed axes exhaust the reachable interval — every value in [min,max] that
// is not an excluded hole is a witnessed axis — so the all-false cell (a reachable value matching
// none) can never occur and must be carved. With no holes this is `width < axis_count`.
func recorder_range_saturated[Value Integer](
	bounds [2]Value, axis_count int, excluded []Value,
) (saturated bool) {
	interior_holes := 0
	for _, hole := range excluded {
		if hole <= bounds[0] {
			continue
		}
		if hole >= bounds[1] {
			continue
		}
		interior_holes++
	}
	return bounds[1]-bounds[0]-Value(interior_holes) < Value(axis_count)
}

// Builds the Impossible carving the all-false cell: in a saturated interval no value is none of the
// witnessed axes, so that combination must never be demanded.
func recorder_range_all_false(messages []string) (carve Engine_Element) {
	references := make([]Dot_Element_Reference, 0, len(messages))
	for _, message := range messages {
		references = append(references, Event_False(message))
	}
	return impossible(references...)
}

// Reports whether candidate lies strictly inside the interval bounds, so a sentinel never coincides
// with an edge the min/max axes already witness — the strict test both the runtime grid and the
// static registration apply, so they admit the same axes.
func recorder_range_interior[Value Integer](
	bounds [2]Value, candidate Value,
) (interior bool) {
	return bounds[0] < candidate && candidate < bounds[1]
}

// Builds a mutual-exclusion Impossible over every pair of admitted axis messages: a value is at
// most one boundary unit, so no two of them are ever true together.
func recorder_range_carves(messages []string) (carves []Engine_Element) {
	for i := range messages {
		for j := i + 1; j < len(messages); j++ {
			carves = append(carves,
				impossible(Event_True(messages[i]), Event_True(messages[j])))
		}
	}
	return carves
}

// Returns the cached Enforce_Handle for message, building it on first use. The build resolves
// every Impossible's references to bundle positions once (and panics on a non-sibling typo);
// later calls read it under RLock and enforce with index and boolean compares, no string scan.
// A plain map keyed by the existing message string reads allocation-free. The sibling of
// recorder_observe_handle, but read in every mode — enforcement is not gated on Is_Test.
func recorder_enforce_handle(
	recorder *Recorder, message string, bundle Engine_Bundle) (handle *Enforce_Handle) {
	recorder.Enforce_Cache_Mu.RLock()
	handle = recorder.Enforce_Cache[message]
	recorder.Enforce_Cache_Mu.RUnlock()
	if handle != nil {
		return handle
	}
	recorder.Enforce_Cache_Mu.Lock()
	defer recorder.Enforce_Cache_Mu.Unlock()
	if handle = recorder.Enforce_Cache[message]; handle != nil {
		return handle
	}
	handle = recorder_enforce_handle_build(bundle)
	if recorder.Enforce_Cache == nil {
		recorder.Enforce_Cache = map[string]*Enforce_Handle{}
	}
	recorder.Enforce_Cache[message] = handle
	return handle
}

// Builds an Enforce_Handle: one rule per Impossible that carries references, each reference
// resolved to the bundle position of the sibling axis it names, and the violation message
// pre-rendered. A reference naming no sibling is a typo and panics here — and since a failed build
// stores nothing, the same bad bundle panics on every call, exactly as the old per-call reference
// check did. A reference-less Impossible constrains nothing, so it yields no rule (it never fires,
// matching dot_element_impossible_violated's empty-set case). The handle holds only positions
// (ints) and freshly rendered strings — no pointers into bundle — so bundle stays non-escaping.
func recorder_enforce_handle_build(bundle Engine_Bundle) (handle *Enforce_Handle) {
	handle = &Enforce_Handle{}
	for _, element := range bundle {
		if element.Kind != DOT_ELEMENT_KIND_IMPOSSIBLE {
			continue
		}
		if len(element.Impossibles) == 0 {
			continue
		}
		var coordinates []Reference_Coordinate
		for _, reference := range element.Impossibles {
			index := dot_product_axis_index(bundle, reference.Message)
			if index < 0 {
				panic(ASSERTION_FAILURE_MESSAGE_PREFIX +
					non_sibling_reference_message(reference))
			}
			coordinates = append(coordinates,
				Reference_Coordinate{Index: index, Event: reference.Event})
		}
		handle.Rules = append(handle.Rules, Impossible_Rule{
			Coordinates: coordinates,
			Message:     dot_element_impossible_message(element),
		})
	}
	return handle
}

// Returns the bundle position of the Sometimes axis carrying message, or -1 when none does — a
// linear scan, since a bundle is a handful of elements. Resolves one Impossible reference to the
// sibling it names, so enforcement compares that axis's event by index rather than by string.
func dot_product_axis_index(bundle Engine_Bundle, message string) (index int) {
	for position, element := range bundle {
		if element.Kind != DOT_ELEMENT_KIND_SOMETIMES {
			continue
		}
		if element.Message == message {
			return position
		}
	}
	return -1
}

// Renders the typo panic a non-sibling reference triggers at plan-build time: an Impossible names
// an axis message that no sibling Sometimes carries. A reference can only carve a cell of this
// product's grid and can only fire against an axis observed on this same call, so naming a
// non-sibling is structurally meaningless — caught at once rather than surfacing as an unfillable
// gap.
func non_sibling_reference_message(reference Dot_Element_Reference) (message string) {
	return "Impossible references " + strconv.Quote(reference.Message) +
		", not an axis of this Dot_Product"
}

// An Observe_Handle memoizes, for one Dot_Product message, what its bundle resolves to so the
// recording hot path builds no key per call: the metadata + tracker key for each non-Impossible
// element (in bundle order) and for each grid cell (indexed by the packed bucket tuple).
type Observe_Handle struct {
	// Elements holds one entry per non-Impossible element, in bundle order.
	Elements []Handle_Entry
	// Tuples is indexed by the observed tuple packed big-endian — element 0 is the
	// most significant bit, one per axis (a Sometimes has two buckets), size
	// 1<<len(Elements). A nil-metadata entry is a cell an Impossible carved or never seeded.
	Tuples []Handle_Entry
}

// A Handle_Entry is a resolved tracker slot: the seeded metadata and the tracker key, cached so
// Coverage_Sink can persist it without rebuilding the string.
type Handle_Entry struct {
	// Metadata is the seeded tracker entry, nil when registration seeded none.
	Metadata *Assertion_Metadata
	// Key is the tracker key, cached so Coverage_Sink can persist it without rebuilding it.
	Key string
}

// An Enforce_Handle memoizes, for one Dot_Product message, its Impossible constraints resolved
// against the bundle's shape so enforcement scans no strings per call: one rule per Impossible,
// each carrying its references as bundle positions and its violation message pre-rendered.
type Enforce_Handle struct {
	// Rules holds one resolved Impossible per rule, in bundle order, so a multi-violation
	// panic names them in the same order the from-scratch scan did. A reference-less Impossible
	// constrains nothing and contributes no rule.
	Rules []Impossible_Rule
}

// An Impossible_Rule is one Impossible resolved against the bundle: the forbidden combination as
// bundle positions, plus the panic message it renders when that combination occurs. The message
// is static (the referenced axes' messages and events are fixed), so it is built once here rather
// than per firing.
type Impossible_Rule struct {
	// Coordinates are the referenced sibling axes' positions and the event each is forbidden
	// at; the rule fires when every one currently holds its forbidden event.
	Coordinates []Reference_Coordinate
	// Message is the pre-rendered violation text, identical to dot_element_impossible_message.
	Message string
}

// A Reference_Coordinate is one Impossible reference resolved to a bundle position: the index of
// the sibling axis it names and the event it forbids there. Enforcement fires the rule when
// bundle[Index].Event == Event for every coordinate — integer index and boolean compares, no
// string matching.
type Reference_Coordinate struct {
	// Index is the position of the referenced sibling axis in the bundle.
	Index int
	// Event is the outcome the reference forbids: the rule needs this axis at this event.
	Event bool
}

// Increments the seeded tracker entry for each observed element and the tuple entry for the
// observed combination, through the per-message Observe_Handle so the steady state allocates
// nothing. Records under a plain test, the fuzz coordinator, and a fuzz worker (all carry
// Is_Test); a no-op in a benchmark or a non-test binary, which only enforce.
func recorder_dot_product_observe(
	recorder *Recorder, message string, bundle Engine_Bundle) {
	if !recorder.Is_Test {
		return
	}
	if recorder.Is_Benchmark {
		return
	}
	handle := recorder_observe_handle(recorder, message, bundle)
	axis_index := 0
	packed := 0
	for _, element := range bundle {
		if element.Kind != DOT_ELEMENT_KIND_SOMETIMES {
			continue
		}
		entry := handle.Elements[axis_index]
		axis_index++
		if element.Gated {
			// A gated axis records only when its prerequisite holds — else don't-care —
			// and joins no tuple (an Imply is excluded from the grid).
			if element.Prerequisite {
				recorder_increment_entry(recorder, entry, element.Event)
			}
			continue
		}
		packed <<= 1
		if element.Event {
			packed |= 1
		}
		recorder_increment_entry(recorder, entry, element.Event)
	}
	recorder_increment_entry(recorder, handle.Tuples[packed], true)
}

// Returns the cached Observe_Handle for message, building it on first use. The build resolves
// every element and grid-cell key against the seeded tracker once; later calls read it under
// RLock with no allocation (a plain map keyed by the existing message string boxes nothing).
func recorder_observe_handle(
	recorder *Recorder, message string, bundle Engine_Bundle) (handle *Observe_Handle) {
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
		recorder.Observe_Cache = map[string]*Observe_Handle{}
	}
	recorder.Observe_Cache[message] = handle
	return handle
}

// Builds an Observe_Handle: one element entry per non-Impossible element (keyed prefix +
// separator + own message) and one tuple entry per grid cell, keyed by the projected coordinate
// exactly as registration seeded it. A cell or element registration never seeded resolves to a
// nil-metadata entry, so the runtime skips it.
func recorder_observe_handle_build(
	recorder *Recorder, message string, bundle Engine_Bundle) (handle *Observe_Handle) {
	handle = &Observe_Handle{}
	ungated_count := 0
	for _, element := range bundle {
		if element.Kind != DOT_ELEMENT_KIND_SOMETIMES {
			continue
		}
		key := message + ELEMENT_MESSAGE_SEPARATOR + element.Message
		handle.Elements = append(handle.Elements, recorder_handle_entry(recorder, key))
		if !element.Gated {
			ungated_count++
		}
	}
	handle.Tuples = make([]Handle_Entry, 1<<ungated_count)
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
func recorder_handle_entry(recorder *Recorder, key string) (entry Handle_Entry) {
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
func recorder_increment_entry(recorder *Recorder, entry Handle_Entry, fired_true bool) {
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
		recorder, Handle_Entry{Metadata: value.(*Assertion_Metadata), Key: key}, fired_true)
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
	recorder_merge_increment(recorder, string(key), line[tab_offset+1:] == "T")
}

// Two separators are reserved for chain axes. Registration owns their exact persisted identity, so
// merge resolves the emitted string directly instead of maintaining an inverse identity algorithm.
func recorder_merge_increment(recorder *Recorder, key string, fired_true bool) {
	if strings.Count(key, ELEMENT_MESSAGE_SEPARATOR) != 2 {
		recorder_increment(recorder, key, fired_true)
		return
	}
	metadata := recorder.Chain_Entries[key]
	if metadata == nil {
		return
	}
	recorder_increment_entry(
		recorder, Handle_Entry{Metadata: metadata, Key: key}, fired_true)
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

// Renders an Impossible violation: a header plus one line per co-occurring
// coordinate, each naming the referenced axis by its message and the event observed.
// The Impossible element's own message is empty, so its identity is this set of
// coordinates rather than a single message. Called once per Impossible at plan-build time to
// pre-render each rule's Message, so a firing rule carries its text with no per-call work.
func dot_element_impossible_message(impossible Engine_Element) (message string) {
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

// Recorder_Register_Packages_For_Analysis parses every non-test .go file under
// the given directories and seeds recorder.Events with one entry per element
// bucket and one per non-carved tuple of each invariant.Dot_Product call. That
// seeded set is the expected-coverage space the never-fired report scans after
// the run; literal invariant.X selectors and *_Invariants bundles are recognised.
//
// Directories default to recorder.Packages_To_Analyze when none are passed; a
// directory may glob, a `*` segment matching one path element and `**` any depth,
// expanded against File_System. Each assertion is keyed by its message; a duplicate
// message, or one that is not a string literal, fails registration (see
// recorder_check_duplicate_messages / recorder_check_non_literal_messages).
func Recorder_Register_Packages_For_Analysis(recorder *Recorder, directories ...string) {
	if len(directories) > 0 {
		recorder.Packages_To_Analyze = directories
	}
	file_set := token.NewFileSet()
	var files []*ast.File
	module_path := ""
	module_root := ""
	for _, directory := range recorder.Packages_To_Analyze {
		// A filepath.Abs here would reach the OS for the working directory, which a pure
		// package must not do; Working_Directory is injected so this stays pure.
		absolute := directory
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(recorder.Working_Directory, absolute)
		}
		absolute = filepath.Clean(absolute)
		expanded_directories := recorder_expand_directories(recorder.File_System, absolute)
		for _, expanded := range expanded_directories {
			if module_path == "" {
				module_path, module_root = recorder_module(recorder, expanded)
			}
			if recorder.Package_Label == "" {
				if module_root != "" {
					relative := strings.TrimPrefix(expanded, module_root)
					relative = strings.TrimPrefix(relative, "/")
					if relative != "" {
						recorder.Package_Label = relative
					}
				}
			}
			parsed := recorder_parse_directory(&Recorder_Parse_Directory_Input{
				File_System: recorder.File_System,
				File_Set:    file_set,
				Directory:   expanded,
			})
			files = append(files, parsed...)
		}
	}
	index := &Bundle_Index{
		File_System:   recorder.File_System,
		File_Set:      file_set,
		Module_Path:   module_path,
		Module_Root:   module_root,
		Sugar_Package: recorder.Sugar_Package,
		Same_Set:      ast_index_functions(files),
		Loaded:        map[string]map[string]Indexed_Function{},
		Constants:     ast_index_constants(files),
	}
	reg := &Registration{Seen_Prefix: map[string]bool{}}
	for _, file := range files {
		recorder_register_file(recorder, file_set, file, index, reg)
	}
	recorder_check_bundle_control_flow(recorder, file_set, files)
	recorder_check_primitive_bundles(recorder, file_set, files, index)
	recorder_check_unresolved(recorder, reg.Unresolved)
	recorder_check_non_literal_messages(recorder, reg.Non_Literal)
	recorder_check_duplicate_messages(recorder, reg.Collision)
	recorder_check_unresolved_bounds(recorder, reg.Unresolved_Bound)
	recorder_check_invalid_chains(recorder, reg.Invalid_Chain)
}

// Reports whether name exists in recorder.File_System.
func recorder_has_entry(recorder *Recorder, name string) (exists bool) {
	_, stat_error := fs.Stat(recorder.File_System, name)
	return stat_error == nil
}

// Walks up from start_directory for a go.mod, returning the module path it
// declares and the absolute directory containing it. Both are "" when none is
// found within MODULE_SEARCH_DEPTH_MAX — cross-package resolution then degrades
// to same-package bundles only.
func recorder_module(
	recorder *Recorder, start_directory string,
) (module_path string, module_root string) {
	directory := start_directory
	for range MODULE_SEARCH_DEPTH_MAX {
		relative := path.Join(strings.TrimPrefix(directory, "/"), "go.mod")
		SOURCE, read_error := fs.ReadFile(recorder.File_System, relative)
		if read_error == nil {
			return parse_module_path(SOURCE), directory
		}
		PARENT := path.Dir(directory)
		if PARENT == directory {
			break
		}
		directory = PARENT
	}
	return "", ""
}

// Returns the module path declared by a go.mod's `module` directive, or "" when
// absent — a line scan, no golang.org/x/mod dependency.
func parse_module_path(SOURCE []byte) (module_path string) {
	for _, line := range strings.Split(string(SOURCE), "\n") {
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

// Input for recorder_parse_directory.
type Recorder_Parse_Directory_Input struct {
	// File_System is the filesystem, rooted at "/", the source files are read from.
	File_System fs.FS
	// File_Set is the token file set the parsed positions are recorded in.
	File_Set *token.FileSet
	// Directory is the absolute directory whose non-test .go files are parsed.
	Directory string
}

// Parses the non-test .go files directly under the absolute Directory into AST
// files. File_System is rooted at "/", so the leading "/" is stripped to address
// it; the parsed file's name is the absolute path, used only for diagnostics now
// (identity is the message, not the position). Subdirectories are skipped — one
// directory is one package.
func recorder_parse_directory(input *Recorder_Parse_Directory_Input) (files []*ast.File) {
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
		SOURCE, read_error := fs.ReadFile(input.File_System, file_path)
		if read_error != nil {
			return nil
		}
		name := "/" + file_path
		file, parse_error := parser.ParseFile(
			input.File_Set, name, SOURCE, parser.SkipObjectResolution,
		)
		if parse_error == nil {
			files = append(files, file)
		}
		return nil
	})
	return files
}

// One frontier entry of the directory-glob walk: a directory reached so far and the
// index of the next pattern segment to match against its children.
type Recorder_Glob_State struct {
	// Directory is a directory reached so far in the glob walk.
	Directory string
	// Index is the next pattern segment to match against this directory's children.
	Index int
}

// Expands a directory pattern against the file system into concrete directories. A
// `*` segment matches one path element; a `**` segment matches zero or more, so
// `a/**` covers a and every directory beneath it. A pattern holding no `*` is
// returned unchanged. Results are unique and lexically sorted for a stable seed.
func recorder_expand_directories(file_system fs.FS, pattern string) (directories []string) {
	if !strings.Contains(pattern, "*") {
		return []string{pattern}
	}
	rooted := strings.HasPrefix(pattern, "/")
	segments := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	frontier := []Recorder_Glob_State{{Directory: ".", Index: 0}}
	matched := map[string]bool{}
	for len(frontier) > 0 {
		current := frontier[len(frontier)-1]
		frontier = frontier[:len(frontier)-1]
		if current.Index == len(segments) {
			matched[current.Directory] = true
			continue
		}
		frontier = append(frontier,
			recorder_glob_step(file_system, current, segments[current.Index])...)
	}
	for directory := range matched {
		result := directory
		if rooted {
			result = "/" + directory
		}
		directories = append(directories, result)
	}
	sort.Strings(directories)
	return directories
}

// The frontier entries reached by matching one pattern segment against current's
// children. A `**` also matches in place (zero elements) and stays in play as it
// descends, so it spans any depth.
func recorder_glob_step(
	file_system fs.FS, current Recorder_Glob_State, segment string,
) (next []Recorder_Glob_State) {
	children := recorder_child_directories(file_system, current.Directory)
	if segment == "**" {
		next = append(next, Recorder_Glob_State{
			Directory: current.Directory, Index: current.Index + 1})
		for _, CHILD := range children {
			next = append(next,
				Recorder_Glob_State{Directory: CHILD, Index: current.Index})
		}
		return next
	}
	for _, CHILD := range children {
		matched, _ := path.Match(segment, path.Base(CHILD))
		if !matched {
			continue
		}
		next = append(next, Recorder_Glob_State{Directory: CHILD, Index: current.Index + 1})
	}
	return next
}

// The immediate subdirectories of directory in the file system, as fs paths.
func recorder_child_directories(file_system fs.FS, directory string) (children []string) {
	entries, read_error := fs.ReadDir(file_system, directory)
	if read_error != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		CHILD := entry.Name()
		if directory != "." {
			CHILD = directory + "/" + entry.Name()
		}
		children = append(children, CHILD)
	}
	return children
}

// An Indexed_Function is a discovered FuncDecl paired with the local-name →
// import-path map of the file it lives in (so the bundle's own qualified
// sub-calls resolve) and whether it lives in the sugar package (so the descent
// recognises its unqualified primitive calls).
type Indexed_Function struct {
	// Declaration is the discovered function declaration.
	Declaration *ast.FuncDecl
	// Imports maps the file's local names to import paths, so qualified sub-calls resolve.
	Imports map[string]string
	// Is_Sugar reports whether the function lives in the sugar package.
	Is_Sugar bool
}

// A Bundle_Index resolves a *_Invariants bundle call to its declaration. Same_Set
// holds the analyzed packages' functions by bare name (same-package bundles); a
// qualified call resolves cross-package within the module via Module_Path /
// Module_Root, lazily parsing and caching each package in Loaded. A bundle outside
// this module is unresolvable.
type Bundle_Index struct {
	// File_System is the filesystem the module's packages are parsed from.
	File_System fs.FS
	// File_Set is the token file set cross-package parses are recorded in.
	File_Set *token.FileSet
	// Module_Path is the module's import-path prefix, used to detect in-module qualified calls.
	Module_Path string
	// Module_Root is the module's absolute root directory on File_System.
	Module_Root string
	// Sugar_Package is the import path of the sugar package.
	Sugar_Package string
	// Same_Set holds the analyzed packages' functions by bare name (same-package bundles).
	Same_Set map[string]Indexed_Function
	// Loaded caches lazily parsed cross-package functions, keyed by import path then bare name.
	Loaded map[string]map[string]Indexed_Function
	// Constants maps each analyzed package-level const's name to its value expression, so a
	// Range_Invariants callsite's MIN/MAX arguments can be evaluated to decide which boundary
	// units its interval admits. A flat index by bare name, like Same_Set — one analysis run
	// covers one package tree, so cross-tree name collisions do not arise.
	Constants map[string]ast.Expr
}

// Maps each function name to its declaration and its file's imports, for
// descending *_Invariants bundles. A later definition wins on a name collision.
func ast_index_functions(files []*ast.File) (functions map[string]Indexed_Function) {
	functions = map[string]Indexed_Function{}
	for _, file := range files {
		imports := ast_file_imports(file)
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			functions[function.Name.Name] = Indexed_Function{
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
	recorder *Recorder, file_set *token.FileSet, file *ast.File, index *Bundle_Index,
	reg *Registration,
) {
	imports := ast_file_imports(file)
	allow_unqualified := false
	if index.Sugar_Package != "" {
		file_package := recorder_file_package(file_set, file, index)
		allow_unqualified = file_package == index.Sugar_Package
	}
	for _, declaration := range file.Decls {
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		if function.Body == nil {
			continue
		}
		recorder_register_function(
			recorder, file_set, function, imports, index, reg, allow_unqualified)
	}
}

// Registers each invariant call in the function: a direct Dot_Product with a literal prefix
// seeds a grid; a Dot_Product whose prefix is this function's namespace parameter is a
// grid template, registered at its callsites instead; a _Invariants(v, "lit") call registers
// the called template's grid under "lit"; any other call may be a bare eager Always.
func recorder_register_function(
	recorder *Recorder, file_set *token.FileSet, function *ast.FuncDecl,
	imports map[string]string, index *Bundle_Index, reg *Registration, allow_unqualified bool,
) {
	namespace_parameter := ""
	if ast_is_invariants_name(function.Name.Name) {
		namespace_parameter = ast_namespace_parameter(function)
	}
	ensured_roots := ast_ensured_chain_roots(function)
	constructor := ast_function_returns_product(function)
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		if ast_chain_method(call) == "Ensure" {
			recorder_register_chain(
				recorder, file_set, call, namespace_parameter,
				imports, index, reg, allow_unqualified)
			return true
		}
		if ast_selector(call, allow_unqualified) == "Dot_Product" {
			if len(call.Args) == 1 {
				if ensured_roots[call.Pos()] {
					return true
				}
				if constructor {
					return true
				}
				reg.Invalid_Chain = append(reg.Invalid_Chain,
					recorder_position(file_set, call)+
						"  Dot_Product chain is not terminated by Ensure")
				return true
			}
			recorder_register_dot_product(
				recorder, file_set, call, namespace_parameter, imports, index, reg)
			return true
		}
		// Range_Invariants ends in _Invariants but is not a descendable bundle: its grid
		// is built at runtime from the callsite's MIN/MAX, so it is specialised here from
		// the evaluated bounds, not a fixed template body. A Range under this function's
		// own namespace parameter is a grid template, deferred to its callsites.
		if ast_invariant_selector(call) == "Range_Invariants" {
			recorder_register_range(
				recorder, file_set, call, namespace_parameter, index, reg)
			return true
		}
		// Enum_Invariants, like Range_Invariants, ends in _Invariants but is specialised
		// here from its member constants rather than descended into as a bundle body.
		if ast_invariant_selector(call) == "Enum_Invariants" {
			recorder_register_enum(recorder, file_set, call, index, reg)
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

// The root set distinguishes a complete nested chain from the same root left dangling in an
// expression statement; source positions are stable within the one registration file set.
func ast_ensured_chain_roots(function *ast.FuncDecl) (roots map[token.Pos]bool) {
	roots = map[token.Pos]bool{}
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		if ast_chain_method(call) != "Ensure" {
			return true
		}
		chain, parsed := ast_chain_from_ensure(call)
		if parsed {
			roots[chain.Root.Pos()] = true
		}
		return true
	})
	return roots
}

// Product-returning functions intentionally expose an unensured reusable prefix; every other
// function must terminate its root locally so registration can see the complete demand.
func ast_function_returns_product(function *ast.FuncDecl) (returns bool) {
	if function.Type.Results == nil {
		return false
	}
	for _, result := range function.Type.Results.List {
		if identifier, is_identifier := result.Type.(*ast.Ident); is_identifier {
			if identifier.Name == "Product" {
				return true
			}
		}
		if selector, is_selector := result.Type.(*ast.SelectorExpr); is_selector {
			if selector.Sel.Name == "Product" {
				return true
			}
		}
	}
	return false
}

// Returns the name of a _Invariants function's trailing namespace parameter — the grid identity
// its self-emitted Dot_Product is prefixed by, typed string or Namespace. "" when there is no
// such parameter.
func ast_namespace_parameter(function *ast.FuncDecl) (name string) {
	if function.Type.Params == nil {
		return ""
	}
	fields := function.Type.Params.List
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	if !ast_is_namespace_type(last.Type) {
		return ""
	}
	if len(last.Names) == 0 {
		return ""
	}
	return last.Names[len(last.Names)-1].Name
}

// Reports whether expression is a namespace parameter's type: the bare `string`, the `Namespace`
// defined type (bare in this package), or a qualified `pkg.Namespace` (the type re-exported).
func ast_is_namespace_type(expression ast.Expr) (is_namespace bool) {
	if identifier, ok := expression.(*ast.Ident); ok {
		return identifier.Name == "string" || identifier.Name == "Namespace"
	}
	if selector, ok := expression.(*ast.SelectorExpr); ok {
		return selector.Sel.Name == "Namespace"
	}
	return false
}

// Seeds a reachability entry for a bare eager Always call — invariant.Always or
// Recorder_Always — keyed by its own message. Without this a
// never-reached Always could not be reported, since it never flows through a Dot_Product.
// Calls of any other kind (a Sometimes, a plain function) seed
// nothing: a bare element records nothing and is the caller's responsibility to consume.
// A duplicate Always message is a fatal collision.
func recorder_register_eager_always(
	recorder *Recorder, file_set *token.FileSet, call *ast.CallExpr, reg *Registration,
) {
	axis, is_axis := recorder_axis_of(file_set, call, false, reg)
	if !is_axis {
		return
	}
	if axis.Kind != ASSERTION_KIND_ALWAYS {
		return
	}
	_, loaded := recorder.Events.LoadOrStore(axis.Message, &Assertion_Metadata{
		Kind:      ASSERTION_KIND_ALWAYS,
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

// Reports every Range_Invariants callsite whose bound the evaluator could not resolve, then exits.
// A recognised-but-unevaluable bound leaves the grid unseeded while the runtime still enforces the
// range, so its coverage obligations would vanish unnoticed; failing keeps coverage from being
// silently dropped — the analyzer seeds a Range grid or refuses it.
func recorder_check_unresolved_bounds(recorder *Recorder, unresolved []string) {
	if len(unresolved) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(unresolved)) + " unresolved Range bounds 🚨"
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

// A malformed chain cannot be partially registered because every omitted axis or carve would
// weaken the demanded product; registration reports every structural refusal before exiting.
func recorder_check_invalid_chains(recorder *Recorder, invalid []string) {
	if len(invalid) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(invalid)) + " invalid Dot_Product chains 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range invalid {
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

// Fails registration when a bundle outside the framework's own package takes a
// primitive subject — a builtin, an unnamed slice/map, or any unnamed composite.
// Bundles for primitive types are the framework's presets; user code states a
// primitive inline or wraps it in a custom type. The Sugar_Package, which owns the
// presets, is exempt.
func recorder_check_primitive_bundles(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File, index *Bundle_Index,
) {
	var offenders []string
	for _, file := range files {
		if index.Sugar_Package != "" {
			if recorder_file_package(file_set, file, index) == index.Sugar_Package {
				continue
			}
		}
		offenders = append(offenders,
			recorder_file_primitive_bundles(file_set, file)...)
	}
	if len(offenders) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(offenders)) + " primitive bundles 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range offenders {
		fmt.Fprintln(recorder.Output, line)
	}
	fmt.Fprintln(recorder.Output, banner)
	recorder.Exit(1)
}

// Collects every primitive-subject bundle declared in one file.
func recorder_file_primitive_bundles(
	file_set *token.FileSet, file *ast.File,
) (offenders []string) {
	for _, declaration := range file.Decls {
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		if !ast_is_invariants_name(function.Name.Name) {
			continue
		}
		if ast_namespace_parameter(function) == "" {
			continue
		}
		subject := recorder_bundle_subject(function)
		if subject == nil {
			continue
		}
		if !recorder_type_is_primitive(subject, recorder_bundle_type_parameters(function)) {
			continue
		}
		offenders = append(offenders, recorder_position(file_set, function)+
			"  primitive bundle subject: "+function.Name.Name)
	}
	return offenders
}

// Returns a bundle's subject type — its first parameter's type — or nil when the
// function declares no parameters.
func recorder_bundle_subject(function *ast.FuncDecl) (subject ast.Expr) {
	if function.Type.Params == nil {
		return nil
	}
	if len(function.Type.Params.List) == 0 {
		return nil
	}
	return function.Type.Params.List[0].Type
}

// Returns the bundle's own type-parameter names; a subject naming one of them is a
// generic custom subject, not a primitive.
func recorder_bundle_type_parameters(function *ast.FuncDecl) (names map[string]bool) {
	names = map[string]bool{}
	if function.Type.TypeParams == nil {
		return names
	}
	for _, field := range function.Type.TypeParams.List {
		for _, name := range field.Names {
			names[name.Name] = true
		}
	}
	return names
}

// Reports whether a bundle subject is a primitive: a predeclared builtin, or an
// unnamed composite (slice, map, channel, anonymous struct/interface/func). A bare
// defined-type name, an imported pkg.Type, or a type parameter is a custom subject.
func recorder_type_is_primitive(
	expression ast.Expr, type_parameters map[string]bool,
) (yes bool) {
	core := expression
	star, is_star := core.(*ast.StarExpr)
	if is_star {
		core = star.X
	}
	index, is_index := core.(*ast.IndexExpr)
	if is_index {
		core = index.X
	}
	index_list, is_index_list := core.(*ast.IndexListExpr)
	if is_index_list {
		core = index_list.X
	}
	identifier, is_identifier := core.(*ast.Ident)
	if is_identifier {
		if type_parameters[identifier.Name] {
			return false
		}
		return recorder_is_builtin_type_name(identifier.Name)
	}
	_, is_selector := core.(*ast.SelectorExpr)
	if is_selector {
		return false
	}
	return true
}

// Reports whether name is a Go predeclared type name.
func recorder_is_builtin_type_name(name string) (yes bool) {
	switch name {
	case "string", "bool", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune", "float32", "float64", "complex64", "complex128",
		"error", "any", "comparable":
		return true
	default:
		return false
	}
}

// Returns the import path of file's package, derived from its absolute path against
// the index's module root and path. "" when no module was found.
func recorder_file_package(
	file_set *token.FileSet, file *ast.File, index *Bundle_Index,
) (import_path string) {
	if index.Module_Path == "" {
		return ""
	}
	absolute := file_set.Position(file.Pos()).Filename
	relative := strings.TrimPrefix(path.Dir(absolute), index.Module_Root)
	relative = strings.TrimPrefix(relative, "/")
	if relative == "" {
		return index.Module_Path
	}
	return path.Join(index.Module_Path, relative)
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

// A Registration_Axis is one Always/Sometimes element discovered statically:
// its Message (the element's own literal), the source text of its condition, its kind,
// and how many buckets it contributes to the tuple grid (Always=1 true; Sometimes=2).
// The consuming Dot_Product's message is prefixed onto Message to form the coverage key,
// uniformly for inline and bundle-descended axes alike.
type Registration_Axis struct {
	// Ordinal keeps repeated chain messages distinct; the legacy engine needs no ordinal.
	Ordinal uint8
	// Tuple_Position is assigned only by chain registration; legacy engine axes ignore it.
	Tuple_Position uint8
	// Message is the element's own literal; the Dot_Product prefix forms the coverage key.
	Message string
	// Condition is the source text of the asserted condition.
	Condition string
	// Kind is whether the element is an Always or a Sometimes.
	Kind Assertion_Kind
	// Bucket_Count is how many buckets the axis adds to the tuple grid (Always=1, Sometimes=2).
	Bucket_Count int
	// Gated marks an Imply axis — seeded per-axis but excluded from the tuple grid.
	Gated bool
}

// A Registration_Cell is one coordinate of an Impossible carve: a Dot_Product
// axis position pinned to a bucket index.
type Registration_Cell struct {
	// Position is the Dot_Product axis position this cell pins.
	Position int
	// Bucket is the bucket index the position is pinned to.
	Bucket int
}

// Registration accumulates the diagnostics a registration pass gathers before deciding
// whether to fail: bundles recognised by name but unresolvable, messages that are not
// string literals, and message collisions. Each is fatal on its own (see the
// recorder_check_* reporters). Seen_Prefix tracks Dot_Product messages so two grids
// cannot share one — the global-uniqueness guarantee for prefixes.
type Registration struct {
	// Unresolved holds bundles recognised by name but not resolvable to a declaration.
	Unresolved []string
	// Non_Literal holds messages that are not string literals, which cannot be keyed.
	Non_Literal []string
	// Collision holds Dot_Product messages that collided with an already-seen prefix.
	Collision []string
	// Unresolved_Bound holds Range_Invariants callsites whose MIN or MAX argument the constant
	// evaluator could not resolve, so their grid could not be seeded.
	Unresolved_Bound []string
	// Invalid_Chain holds structural chain errors that would otherwise drop demanded coverage.
	Invalid_Chain []string
	// Seen_Prefix tracks Dot_Product messages so two grids cannot share one prefix.
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

// Registration_Chain holds the already-linearized call nest so seeding never needs variable
// tracking and runtime bit positions follow exactly the source link order.
type Registration_Chain struct {
	// Root identifies where namespace ownership starts.
	Root *ast.CallExpr
	// Links is source ordered so ordinal and mask packing cannot diverge.
	Links []*ast.CallExpr
}

// AST chain selection must accept call receivers while the primitive selector rejects them.
func ast_chain_method(call *ast.CallExpr) (name string) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return ""
	}
	return selector.Sel.Name
}

// A chain is syntactically one nest. Walking receiver calls backward makes split Product variables
// impossible to mis-register as a complete demanded grid.
func ast_chain_from_ensure(ensure *ast.CallExpr) (chain Registration_Chain, ok bool) {
	if ast_chain_method(ensure) != "Ensure" {
		return chain, false
	}
	current, has_receiver := ast_chain_receiver(ensure)
	if !has_receiver {
		return chain, false
	}
	var reversed []*ast.CallExpr
	for step_index := 0; step_index < BUNDLE_EXPANSION_STEPS_MAX; step_index++ {
		method := ast_chain_method(current)
		if method != "Sometimes" {
			if method != "Impossible" {
				chain.Root = current
				break
			}
		}
		reversed = append(reversed, current)
		current, has_receiver = ast_chain_receiver(current)
		if !has_receiver {
			return Registration_Chain{}, false
		}
	}
	if chain.Root == nil {
		return Registration_Chain{}, false
	}
	chain.Links = make([]*ast.CallExpr, len(reversed))
	for i := range reversed {
		chain.Links[i] = reversed[len(reversed)-1-i]
	}
	return chain, true
}

func ast_chain_receiver(call *ast.CallExpr) (receiver *ast.CallExpr, ok bool) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return nil, false
	}
	receiver, is_call := selector.X.(*ast.CallExpr)
	return receiver, is_call
}

// Recorder_register_chain defers template namespaces because only their callsites own identity.
func recorder_register_chain(
	recorder *Recorder, file_set *token.FileSet, ensure *ast.CallExpr,
	namespace_parameter string, imports map[string]string,
	index *Bundle_Index, reg *Registration, allow_unqualified bool,
) {
	chain, parsed := ast_chain_from_ensure(ensure)
	if !parsed {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, ensure)+
				"  Ensure does not terminate one call nest")
		return
	}
	if ast_selector(chain.Root, allow_unqualified) != "Dot_Product" {
		chain, parsed = recorder_expand_chain_constructor(
			file_set, chain, imports, index, reg)
		if !parsed {
			return
		}
	}
	if len(chain.Root.Args) != 1 {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, chain.Root)+
				"  chain Dot_Product root must have one argument")
		return
	}
	namespace, literal := ast_string_literal(chain.Root, 0)
	if !literal {
		if ast_is_template_prefix(chain.Root, namespace_parameter) {
			return
		}
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, chain.Root)+
				"  Dot_Product namespace is not a string literal")
		return
	}
	if strings.Contains(namespace, ELEMENT_MESSAGE_SEPARATOR) {
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, chain.Root)+
				"  Dot_Product namespace is not a NUL-free literal")
		return
	}
	recorder_seed_registration_chain(
		recorder, file_set, ensure, namespace, chain.Links, reg, allow_unqualified)
}

// Constructor expansion is implemented with the same function index used by _Invariants. A direct
// Dot_Product root needs no expansion; unresolved Product roots are fatal instead of losing axes.
func recorder_expand_chain_constructor(
	file_set *token.FileSet, chain Registration_Chain, imports map[string]string,
	index *Bundle_Index, reg *Registration,
) (expanded Registration_Chain, ok bool) {
	constructor, found := bundle_index_lookup(index, imports, chain.Root)
	if !found {
		line := recorder_unresolved_line(file_set, chain.Root)
		reg.Unresolved = append(reg.Unresolved, line)
		return Registration_Chain{}, false
	}
	root, links, found := recorder_product_constructor_chain(constructor.Declaration)
	if !found {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, chain.Root)+
				"  Product constructor has no returned chain")
		return Registration_Chain{}, false
	}
	if len(chain.Root.Args) == 0 {
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, chain.Root)+
				"  Product constructor has no namespace argument")
		return Registration_Chain{}, false
	}
	namespace := chain.Root.Args[len(chain.Root.Args)-1]
	root.Args[0] = namespace
	expanded = Registration_Chain{Root: root, Links: append(links, chain.Links...)}
	return expanded, true
}

// A shallow clone keeps namespace substitution from mutating the shared function index.
func recorder_product_constructor_chain(
	function *ast.FuncDecl,
) (root *ast.CallExpr, links []*ast.CallExpr, found bool) {
	if function.Body == nil {
		return nil, nil, false
	}
	for _, statement := range function.Body.List {
		result, is_return := statement.(*ast.ReturnStmt)
		if !is_return {
			continue
		}
		if len(result.Results) != 1 {
			continue
		}
		call, is_call := result.Results[0].(*ast.CallExpr)
		if !is_call {
			continue
		}
		var reversed []*ast.CallExpr
		for step_index := 0; step_index < BUNDLE_EXPANSION_STEPS_MAX; step_index++ {
			method := ast_chain_method(call)
			if method != "Sometimes" {
				if method != "Impossible" {
					break
				}
			}
			reversed = append(reversed, call)
			call, is_call = ast_chain_receiver(call)
			if !is_call {
				return nil, nil, false
			}
		}
		if ast_selector(call, true) != "Dot_Product" {
			return nil, nil, false
		}
		copy_root := *call
		copy_root.Args = append([]ast.Expr(nil), call.Args...)
		links = make([]*ast.CallExpr, len(reversed))
		for i := range reversed {
			links[i] = reversed[len(reversed)-1-i]
		}
		return &copy_root, links, true
	}
	return nil, nil, false
}

// The collector validates every literal and reference before seeding anything, so a fatal chain
// cannot leave a partial grid that later analysis would mistake for the complete demand.
func recorder_seed_registration_chain(
	recorder *Recorder, file_set *token.FileSet, position ast.Node, namespace string,
	link_calls []*ast.CallExpr, reg *Registration, allow_unqualified bool,
) {
	if len(link_calls) > CHAIN_LINKS_MAX {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, position)+"  Dot_Product exceeds 255 links")
		return
	}
	axes, carves, links, rules, valid := recorder_collect_chain(
		file_set, link_calls, reg, allow_unqualified)
	if !valid {
		return
	}
	if len(axes) == 0 {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, position)+"  Dot_Product has no axes")
		return
	}
	recorder_seed_chain(
		recorder, file_set, position, namespace, axes, carves, links, rules, reg)
}

func recorder_collect_chain(
	file_set *token.FileSet, calls []*ast.CallExpr, reg *Registration, allow_unqualified bool,
) (
	axes []Registration_Axis, carves [][]Registration_Cell,
	links []Chain_Link, rules []Chain_Rule, valid bool,
) {
	valid = true
	positions := map[string][]int{}
	rule_messages := map[string]bool{}
	for ordinal, call := range calls {
		method := ast_chain_method(call)
		if method == "Sometimes" {
			axis, ok := recorder_collect_chain_axis(file_set, call, uint8(ordinal), reg)
			valid = valid && ok
			if !ok {
				continue
			}
			axis.Tuple_Position = uint8(len(axes))
			positions[axis.Message] = append(positions[axis.Message], len(axes))
			axes = append(axes, axis)
			links = append(links, Chain_Link{
				Kind: DOT_ELEMENT_KIND_SOMETIMES, Ordinal: uint8(ordinal),
				Axis_Count: uint8(len(axes) - 1), Message: axis.Message,
			})
			continue
		}
		cells, link, rule, ok := recorder_collect_chain_rule(
			file_set, call, uint8(ordinal), axes, positions, rule_messages,
			reg, allow_unqualified)
		valid = valid && ok
		if !ok {
			continue
		}
		carves = append(carves, cells)
		links = append(links, link)
		rules = append(rules, rule)
	}
	return axes, carves, links, rules, valid
}

func recorder_collect_chain_axis(
	file_set *token.FileSet, call *ast.CallExpr, ordinal uint8, reg *Registration,
) (axis Registration_Axis, valid bool) {
	if len(call.Args) != 2 {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, call)+
				"  Sometimes link must have two arguments")
		return axis, false
	}
	message, literal := ast_string_literal(call, 1)
	if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
		literal = false
	}
	if !literal {
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, call)+
				"  Sometimes message is not a NUL-free literal")
		return axis, false
	}
	return Registration_Axis{
		Ordinal: ordinal, Message: message,
		Condition: ast_condition_text(file_set, call, 0),
		Kind:      ASSERTION_KIND_SOMETIMES, Bucket_Count: 2,
	}, true
}

func recorder_collect_chain_rule(
	file_set *token.FileSet, call *ast.CallExpr, ordinal uint8, axes []Registration_Axis,
	positions map[string][]int, rule_messages map[string]bool,
	reg *Registration, allow_unqualified bool,
) (cells []Registration_Cell, link Chain_Link, rule Chain_Rule, valid bool) {
	if len(call.Args) < 2 {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, call)+
				"  Impossible link must name a rule and references")
		return cells, link, rule, false
	}
	if len(call.Args)-1 > CHAIN_LINKS_MAX {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, call)+
				"  Impossible link has more than 255 references")
		return cells, link, rule, false
	}
	message, literal := ast_string_literal(call, 0)
	if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
		literal = false
	}
	if !literal {
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, call)+
				"  Impossible message is not a NUL-free literal")
		return cells, link, rule, false
	}
	if rule_messages[message] {
		reg.Collision = append(reg.Collision,
			recorder_position(file_set, call)+
				"  duplicate Impossible message: "+strconv.Quote(message))
		return cells, link, rule, false
	}
	rule_messages[message] = true
	axis_count := 0
	for _, matches := range positions {
		axis_count += len(matches)
	}
	referenced := make([]bool, axis_count)
	for _, expression := range call.Args[1:] {
		cell, _, ok := recorder_collect_chain_reference(
			file_set, expression, positions, referenced, reg, allow_unqualified)
		if !ok {
			return cells, link, rule, false
		}
		cells = append(cells, cell)
	}
	var mask Chain_Mask
	var want Chain_Mask
	var events Chain_Mask
	for reference_index, cell := range cells {
		tuple_position := axes[cell.Position].Tuple_Position
		mask = chain_mask_with(mask, tuple_position)
		if cell.Bucket == 1 {
			want = chain_mask_with(want, tuple_position)
			events = chain_mask_with(events, uint8(reference_index))
		}
	}
	link.Kind = DOT_ELEMENT_KIND_IMPOSSIBLE
	link.Ordinal = ordinal
	link.Axis_Count = uint8(axis_count)
	link.Message = message
	link.Reference_Count = uint8(len(cells))
	link.Reference_Events = events
	for reference_index, cell := range cells {
		link.References[reference_index] = uint8(cell.Position)
	}
	rule = Chain_Rule{
		Mask: mask, Want: want, Message: message, Ordinal: ordinal}
	return cells, link, rule, true
}

func recorder_collect_chain_reference(
	file_set *token.FileSet, expression ast.Expr, positions map[string][]int,
	referenced []bool, reg *Registration, allow_unqualified bool,
) (cell Registration_Cell, reference Dot_Element_Reference, valid bool) {
	reference_call, is_call := expression.(*ast.CallExpr)
	if !is_call {
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, expression)+
				"  Impossible reference is not literal")
		return cell, reference, false
	}
	message, literal := ast_string_literal(reference_call, 0)
	if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
		literal = false
	}
	if !literal {
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, reference_call)+
				"  Impossible reference is not a NUL-free literal")
		return cell, reference, false
	}
	matches := positions[message]
	if len(matches) != 1 {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, reference_call)+
				"  Impossible reference does not name one preceding axis")
		return cell, reference, false
	}
	position := matches[0]
	if len(referenced) > position {
		if referenced[position] {
			reg.Invalid_Chain = append(reg.Invalid_Chain,
				recorder_position(file_set, reference_call)+
					"  Impossible repeats an axis reference")
			return cell, reference, false
		}
	}
	referenced[position] = true
	bucket := ast_event_bucket(ast_selector(reference_call, allow_unqualified))
	if bucket < 0 {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, reference_call)+
				"  invalid Impossible polarity")
		return cell, reference, false
	}
	cell = Registration_Cell{Position: position, Bucket: bucket}
	reference = Dot_Element_Reference{Message: message, Event: bucket == 1}
	return cell, reference, true
}

func recorder_seed_chain(
	recorder *Recorder, file_set *token.FileSet, position ast.Node, namespace string,
	axes []Registration_Axis, carves [][]Registration_Cell,
	links []Chain_Link, rules []Chain_Rule, reg *Registration,
) {
	if reg.Seen_Prefix[namespace] {
		reg.Collision = append(reg.Collision,
			recorder_position(file_set, position)+
				"  duplicate Dot_Product namespace: "+strconv.Quote(namespace))
		return
	}
	reg.Seen_Prefix[namespace] = true
	shape := &Chain_Shape{
		Axes: make([]Chain_Axis, len(axes)), Links: links, Rules: rules,
		Tuples: map[Chain_Mask]Handle_Entry{}, Registered: true, Ensured: true,
	}
	if recorder.Chain_Entries == nil {
		recorder.Chain_Entries = map[string]*Assertion_Metadata{}
	}
	for i, axis := range axes {
		text := chain_axis_key(Namespace(namespace), axis.Ordinal, axis.Message)
		metadata := &Assertion_Metadata{
			Kind: axis.Kind, Message: text, Condition: axis.Condition}
		recorder.Events.Store(text, metadata)
		recorder.Chain_Entries[text] = metadata
		shape.Axes[i] = Chain_Axis{
			Ordinal: axis.Ordinal, Tuple_Position: axis.Tuple_Position,
			Message: axis.Message, Entry: Handle_Entry{Metadata: metadata, Key: text},
		}
	}
	recorder_register_tuples(recorder, namespace, axes, carves, shape.Tuples)
	recorder.Chain_Shapes_Mu.Lock()
	if recorder.Chain_Shapes == nil {
		recorder.Chain_Shapes = map[Namespace]*Chain_Shape{}
	}
	recorder.Chain_Shapes[Namespace(namespace)] = shape
	recorder.Chain_Shapes_Mu.Unlock()
}

// Registers a Dot_Product call. A literal prefix seeds a grid from the call's inline axes. A
// prefix that is this function's namespace parameter is a grid template — registered at the
// _Invariants' callsites, not here. Any other non-literal prefix is fatal.
func recorder_register_dot_product(
	recorder *Recorder, file_set *token.FileSet, call *ast.CallExpr, namespace_parameter string,
	imports map[string]string, index *Bundle_Index, reg *Registration,
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
	imports map[string]string, index *Bundle_Index, reg *Registration,
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
	if strings.Contains(namespace, ELEMENT_MESSAGE_SEPARATOR) {
		reg.Non_Literal = append(reg.Non_Literal,
			recorder_position(file_set, call)+
				"  _Invariants namespace is not a NUL-free literal")
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
	template_chains := recorder_template_chains(function.Declaration)
	if len(template_chains) > 0 {
		for _, ensure := range template_chains {
			chain, parsed := ast_chain_from_ensure(ensure)
			if !parsed {
				reg.Invalid_Chain = append(reg.Invalid_Chain,
					recorder_position(file_set, ensure)+
						"  template Ensure is not one call nest")
				continue
			}
			recorder_seed_registration_chain(
				recorder, file_set, call, namespace,
				chain.Links, reg, function.Is_Sugar)
		}
		return
	}
	dot_product, has := recorder_template_dot_product(function.Declaration)
	if has {
		axes, carves := recorder_collect_inline(
			file_set, dot_product.Args[1:], function.Is_Sugar, reg)
		recorder_seed_grid(recorder, file_set, call, namespace, axes, carves, reg)
		return
	}
	// A template whose body is a Range_Invariants under its namespace parameter seeds its
	// bound grid here, under the callsite's literal namespace, from the template's MIN/MAX
	// expressions — the Range mirror of the Dot_Product descent above.
	if range_call, is_range := recorder_template_range(function.Declaration); is_range {
		recorder_seed_range(recorder, file_set, call, namespace,
			[2]ast.Expr{range_call.Args[1], range_call.Args[2]}, range_call.Args[4:],
			index, reg)
	}
}

// A chain template is the ensured product rooted at the function's trailing namespace parameter;
// its definition carries shape while each callsite supplies the actual coverage identity.
func recorder_template_chains(function *ast.FuncDecl) (ensures []*ast.CallExpr) {
	namespace_parameter := ast_namespace_parameter(function)
	if namespace_parameter == "" {
		return nil
	}
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		candidate, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		if ast_chain_method(candidate) != "Ensure" {
			return true
		}
		chain, parsed := ast_chain_from_ensure(candidate)
		if !parsed {
			return true
		}
		if !ast_is_template_prefix(chain.Root, namespace_parameter) {
			return true
		}
		ensures = append(ensures, candidate)
		return true
	})
	return ensures
}

// Bounds the constant-evaluator's stack steps, so a pathological const cycle (a const whose value
// references itself through others) cannot loop the analyzer unboundedly. A real const graph is
// acyclic and finishes far below this.
const CONSTANT_EVAL_STEPS_MAX = 256

// A Range_Unit is one boundary value Recorder_Range may claim: its integer value, the axis message
// it seeds when in range, and the source label its condition renders with.
type Range_Unit struct {
	// Value is the boundary integer (0, 1, 2, or -1).
	Value int64
	// Message is the axis's own message, joined to the namespace to form its coverage key.
	Message string
	// Label is how the value reads in the axis condition text of the gap report.
	Label string
}

// The boundary units Recorder_Range claims, in the exact order its runtime grid emits them, so the
// scan's projected tuple coordinates line up with what the run records.
func recorder_range_units() (units []Range_Unit) {
	return []Range_Unit{
		{Value: 0, Message: RANGE_MESSAGE_ZERO, Label: "0"},
		{Value: 1, Message: RANGE_MESSAGE_ONE, Label: "1"},
		{Value: 2, Message: RANGE_MESSAGE_TWO, Label: "2"},
		{Value: -1, Message: RANGE_MESSAGE_NEGATIVE_ONE, Label: "-1"},
	}
}

// Registers a Range_Invariants callsite. It evaluates the MIN and MAX arguments, seeds the two
// bound guards as namespaced Always reachability entries, and seeds a Sometimes axis for each
// interval edge (min and max) and each of {0, 1, 2, -1} strictly inside — mutually exclusive, so
// only the singleton and all-clear cells survive. The tests mirror Recorder_Range's runtime
// decision, so the scan demands exactly what the run credits. A non-literal namespace, or a bound
// the evaluator cannot resolve, is fatal: the grid could not otherwise be keyed while the runtime
// still enforces it.
func recorder_register_range(
	recorder *Recorder, file_set *token.FileSet, call *ast.CallExpr,
	namespace_parameter string, index *Bundle_Index, reg *Registration,
) {
	// Args are value, MIN, MAX, namespace; the value on Args[0] is not needed to seed the grid.
	if len(call.Args) < 4 {
		return
	}
	namespace, literal := ast_string_literal(call, 3)
	if !literal {
		// A Range under the enclosing bundle's namespace parameter is a grid template —
		// its grid is seeded at each _Invariants callsite's literal namespace, never
		// under the bare parameter, exactly as a self-emitting Dot_Product template is.
		// Any other non-literal namespace is fatal: its coverage could not be keyed.
		if ast_is_range_template(call, namespace_parameter) {
			return
		}
		reg.Non_Literal = append(reg.Non_Literal, recorder_position(file_set, call)+
			"  Range_Invariants namespace is not a string literal")
		return
	}
	recorder_seed_range(recorder, file_set, call, namespace,
		[2]ast.Expr{call.Args[1], call.Args[2]}, call.Args[4:], index, reg)
}

// Reports whether a Range_Invariants call's namespace argument is the enclosing function's
// namespace parameter — the shape of a grid template, registered at the _Invariants' callsites
// rather than here, mirroring ast_is_template_prefix for a Dot_Product's leading prefix.
func ast_is_range_template(call *ast.CallExpr, namespace_parameter string) (is_template bool) {
	if namespace_parameter == "" {
		return false
	}
	if len(call.Args) < 4 {
		return false
	}
	identifier, is_identifier := call.Args[3].(*ast.Ident)
	if !is_identifier {
		return false
	}
	return identifier.Name == namespace_parameter
}

// Seeds a Range grid under namespace: it evaluates the MIN and MAX bound expressions, seeds the two
// bound guards as namespaced Always reachability entries, and seeds a Sometimes axis for each
// interval edge (min and max) and each of {0, 1, 2, -1} strictly inside — mutually exclusive. A
// direct Range callsite passes its own literal namespace; a Range template passes each _Invariants
// callsite's literal namespace with the template's bound expressions. A bound the evaluator cannot
// resolve is fatal, since the grid could not be keyed while the runtime still enforces it.
func recorder_seed_range(
	recorder *Recorder, file_set *token.FileSet, position ast.Node, namespace string,
	expressions [2]ast.Expr, exclusions []ast.Expr, index *Bundle_Index, reg *Registration,
) {
	value_min, ok_min := recorder_eval_constant(index, expressions[0])
	value_max, ok_max := recorder_eval_constant(index, expressions[1])
	if !ok_min {
		recorder_range_unresolved(file_set, position, reg)
		return
	}
	if !ok_max {
		recorder_range_unresolved(file_set, position, reg)
		return
	}
	excluded, resolved := recorder_seed_constants(index, exclusions)
	if !resolved {
		recorder_range_unresolved(file_set, position, reg)
		return
	}
	bounds := [2]constant.Value{value_min, value_max}
	axes := recorder_seed_axes(bounds, excluded, false)
	carves := recorder_range_carve_cells(axes)
	if recorder_constant_saturated(bounds, recorder_range_sometimes_count(axes), excluded) {
		carves = append(carves, recorder_range_all_false_cells(axes))
	}
	recorder_seed_grid(recorder, file_set, position, namespace, axes, carves, reg)
}

// Builds the static mirror of recorder_range_axes: the two bound guards, the min/max edges (unless
// the interval is a point), and each of {0,1,2,-1} strictly inside admitted by its membership in
// `set` — for a range `set` is the excluded holes (dropped), for an enum the members (kept).
func recorder_seed_axes(
	bounds [2]constant.Value, set []constant.Value, enum bool,
) (axes []Registration_Axis) {
	axes = []Registration_Axis{
		{Message: RANGE_GUARD_UPPER, Condition: "value <= max",
			Kind: ASSERTION_KIND_ALWAYS, Bucket_Count: 1},
		{Message: RANGE_GUARD_LOWER, Condition: "value >= min",
			Kind: ASSERTION_KIND_ALWAYS, Bucket_Count: 1},
	}
	if constant.Compare(bounds[0], token.LSS, bounds[1]) {
		axes = append(axes, recorder_range_axis(RANGE_MESSAGE_MINIMUM))
		axes = append(axes, recorder_range_axis(RANGE_MESSAGE_MAXIMUM))
	}
	for _, unit := range recorder_range_units() {
		if !recorder_constant_interior(bounds, unit.Value) {
			continue
		}
		if recorder_constant_holed(unit.Value, set) != enum {
			continue
		}
		axes = append(axes, recorder_range_axis(unit.Message))
	}
	return axes
}

// Evaluates a list of constant expressions (Range exclusions or Enum members). Any expression the
// evaluator cannot resolve makes the whole set unresolved, so the caller fails registration.
func recorder_seed_constants(
	index *Bundle_Index, expressions []ast.Expr,
) (values []constant.Value, resolved bool) {
	for _, expression := range expressions {
		value, ok := recorder_eval_constant(index, expression)
		if !ok {
			return nil, false
		}
		values = append(values, value)
	}
	return values, true
}

// Reports whether candidate equals a value in the set — the static mirror of recorder_range_holed.
func recorder_constant_holed(candidate int64, set []constant.Value) (holed bool) {
	target := constant.MakeInt64(candidate)
	for _, value := range set {
		if constant.Compare(target, token.EQL, value) {
			return true
		}
	}
	return false
}

// Reports whether the reachable interval is saturated — every in-range value that is not an
// excluded hole is a witnessed axis, so the all-false cell can never occur. Mirrors the runtime.
func recorder_constant_saturated(
	bounds [2]constant.Value, axis_count int, excluded []constant.Value,
) (saturated bool) {
	width := constant.BinaryOp(bounds[1], token.SUB, bounds[0])
	holes := 0
	for _, value := range excluded {
		if constant.Compare(value, token.LEQ, bounds[0]) {
			continue
		}
		if constant.Compare(value, token.GEQ, bounds[1]) {
			continue
		}
		holes++
	}
	width = constant.BinaryOp(width, token.SUB, constant.MakeInt64(int64(holes)))
	return constant.Compare(width, token.LSS, constant.MakeInt64(int64(axis_count)))
}

// Registers a direct Enum_Invariants callsite: Args are value, namespace, members. It seeds the
// grid for the discrete member set under the literal namespace. A non-literal namespace is fatal —
// enum templates (a member set under a namespace parameter) are not yet supported.
func recorder_register_enum(
	recorder *Recorder, file_set *token.FileSet, call *ast.CallExpr,
	index *Bundle_Index, reg *Registration,
) {
	if len(call.Args) < 3 {
		return
	}
	namespace, literal := ast_string_literal(call, 1)
	if !literal {
		reg.Non_Literal = append(reg.Non_Literal, recorder_position(file_set, call)+
			"  Enum_Invariants namespace is not a string literal")
		return
	}
	recorder_seed_enum(recorder, file_set, call, namespace, call.Args[2:], index, reg)
}

// Seeds an Enum grid: it evaluates the member constants, spans them into [min, max], and admits the
// same axes a range would while dropping every non-member sentinel. The all-false cell is carved
// once every member is a witnessed axis, so a reachable value witnessing none can never occur.
func recorder_seed_enum(
	recorder *Recorder, file_set *token.FileSet, position ast.Node, namespace string,
	members []ast.Expr, index *Bundle_Index, reg *Registration,
) {
	values, resolved := recorder_seed_constants(index, members)
	if !resolved {
		recorder_range_unresolved(file_set, position, reg)
		return
	}
	bounds := recorder_constant_span(values)
	axes := recorder_seed_axes(bounds, values, true)
	carves := recorder_range_carve_cells(axes)
	if len(values) <= recorder_range_sometimes_count(axes) {
		carves = append(carves, recorder_range_all_false_cells(axes))
	}
	recorder_seed_grid(recorder, file_set, position, namespace, axes, carves, reg)
}

// Spans a member set into its least and greatest value — the static mirror of recorder_enum_bounds.
func recorder_constant_span(values []constant.Value) (bounds [2]constant.Value) {
	for i, value := range values {
		if i == 0 {
			bounds = [2]constant.Value{value, value}
			continue
		}
		if constant.Compare(value, token.LSS, bounds[0]) {
			bounds[0] = value
		}
		if constant.Compare(value, token.GTR, bounds[1]) {
			bounds[1] = value
		}
	}
	return bounds
}

// Counts the Sometimes (coverage) axes of a Range grid — every axis past the two leading guards.
func recorder_range_sometimes_count(axes []Registration_Axis) (count int) {
	for i := range axes {
		if axes[i].Bucket_Count >= 2 {
			count++
		}
	}
	return count
}

// Builds the carve cell pinning every Sometimes axis false — the all-false combination a saturated
// interval can never witness.
func recorder_range_all_false_cells(axes []Registration_Axis) (cells []Registration_Cell) {
	for i := range axes {
		if axes[i].Bucket_Count < 2 {
			continue
		}
		cells = append(cells, Registration_Cell{Position: i, Bucket: 0})
	}
	return cells
}

// Builds one Sometimes coverage axis of a Range grid — an edge or an interior sentinel. The axis's
// own message doubles as its condition text, since the message already reads as the claim.
func recorder_range_axis(message string) (axis Registration_Axis) {
	return Registration_Axis{
		Message: message, Condition: message,
		Kind: ASSERTION_KIND_SOMETIMES, Bucket_Count: 2,
	}
}

// Builds the mutual-exclusion carves over a Range grid's Sometimes axes — every axis past the two
// leading guards, pinned true pairwise, since a value equals at most one edge or sentinel.
func recorder_range_carve_cells(axes []Registration_Axis) (carves [][]Registration_Cell) {
	var positions []int
	for position_index := 2; position_index < len(axes); position_index++ {
		positions = append(positions, position_index)
	}
	for i := range positions {
		for j := i + 1; j < len(positions); j++ {
			carves = append(carves, []Registration_Cell{
				{Position: positions[i], Bucket: 1},
				{Position: positions[j], Bucket: 1},
			})
		}
	}
	return carves
}

// Records a Range_Invariants callsite whose bound the evaluator could not resolve, so registration
// fails rather than seeding an incomplete grid the runtime would still enforce.
func recorder_range_unresolved(
	file_set *token.FileSet, position ast.Node, reg *Registration,
) {
	reg.Unresolved_Bound = append(reg.Unresolved_Bound,
		recorder_position(file_set, position)+
			"  Range_Invariants bound is not a resolvable constant")
}

// Reports whether the integer n lies strictly inside the interval bounds, comparing in arbitrary
// precision so a bound beyond int64 (a uint64 near its ceiling) and a negative n are both handled.
// -1 falls out for an unsigned type whose lower bound is zero without any signedness test.
func recorder_constant_interior(bounds [2]constant.Value, n int64) (interior bool) {
	target := constant.MakeInt64(n)
	return constant.Compare(bounds[0], token.LSS, target) &&
		constant.Compare(target, token.LSS, bounds[1])
}

// A Range_Eval_Frame is one node of the constant-evaluation work stack: an expression to evaluate,
// and whether its operands were already pushed — an operator is visited twice, once to expand its
// children and once to combine their now-evaluated values.
type Range_Eval_Frame struct {
	// Expression is the AST node this frame evaluates.
	Expression ast.Expr
	// Expanded reports whether this operator already pushed its operands for evaluation.
	Expanded bool
}

// Evaluates a Range_Invariants bound expression to an integer constant: an integer literal, a
// reference to a sibling package const, a parenthesised expression, a unary +/-/^, or a binary
// arithmetic/bitwise/shift op over those. ok is false for anything else (a variable, an imported
// selector, a float, a call), which the caller treats as a fatal unresolved bound. No iota.
//
// The walk is an explicit post-order stack, not recursion (which the linter bans): an operator is
// re-pushed marked Expanded after its operands, so it combines their values once those are on the
// value stack. A step cap bounds a pathological const-reference cycle.
func recorder_eval_constant(
	index *Bundle_Index, expression ast.Expr,
) (value constant.Value, ok bool) {
	work := []Range_Eval_Frame{{Expression: expression}}
	var values []constant.Value
	for step := 0; len(work) > 0; step++ {
		if step > CONSTANT_EVAL_STEPS_MAX {
			return nil, false
		}
		frame := work[len(work)-1]
		work = work[:len(work)-1]
		switch node := frame.Expression.(type) {
		case *ast.BasicLit:
			literal, literal_ok := recorder_eval_literal(node)
			if !literal_ok {
				return nil, false
			}
			values = append(values, literal)
		case *ast.Ident:
			reference, defined := index.Constants[node.Name]
			if !defined {
				return nil, false
			}
			work = append(work, Range_Eval_Frame{Expression: reference})
		case *ast.ParenExpr:
			work = append(work, Range_Eval_Frame{Expression: node.X})
		case *ast.UnaryExpr:
			if !frame.Expanded {
				work = recorder_eval_expand(work, node, node.X)
				continue
			}
			top := values[len(values)-1]
			combined, combine_ok := recorder_constant_unary(node.Op, top)
			if !combine_ok {
				return nil, false
			}
			values[len(values)-1] = combined
		case *ast.BinaryExpr:
			if !frame.Expanded {
				work = recorder_eval_expand(work, node, node.X, node.Y)
				continue
			}
			if len(values) < 2 {
				return nil, false
			}
			operands := [2]constant.Value{values[len(values)-2], values[len(values)-1]}
			combined, combine_ok := recorder_constant_binary(node.Op, operands)
			if !combine_ok {
				return nil, false
			}
			values = append(values[:len(values)-2], combined)
		default:
			return nil, false
		}
	}
	if len(values) != 1 {
		return nil, false
	}
	return values[0], true
}

// Evaluates an integer literal; ok is false for a non-integer or unparseable literal.
func recorder_eval_literal(literal *ast.BasicLit) (value constant.Value, ok bool) {
	if literal.Kind != token.INT {
		return nil, false
	}
	evaluated := constant.MakeFromLiteral(literal.Value, token.INT, 0)
	return evaluated, evaluated.Kind() != constant.Unknown
}

// Re-pushes an operator marked Expanded, then its operands in reverse, so each operand evaluates
// before the operator combines them and the first-listed operand ends up first on the value stack.
func recorder_eval_expand(
	work []Range_Eval_Frame, operator ast.Expr, operands ...ast.Expr,
) (expanded []Range_Eval_Frame) {
	work = append(work, Range_Eval_Frame{Expression: operator, Expanded: true})
	for i := len(operands) - 1; i >= 0; i-- {
		work = append(work, Range_Eval_Frame{Expression: operands[i]})
	}
	return work
}

// Applies a unary +, -, or ^ to an integer constant; ok is false for any other operator.
func recorder_constant_unary(
	op token.Token, operand constant.Value,
) (value constant.Value, ok bool) {
	switch op {
	case token.SUB, token.ADD, token.XOR:
		result := constant.UnaryOp(op, operand, 0)
		return result, result.Kind() != constant.Unknown
	}
	return nil, false
}

// Applies a binary arithmetic, bitwise, or shift op to two integer constants. A shift reads its
// count as a uint; a divide or remainder by zero is refused rather than panicking the analyzer; ok
// is false for any other operator.
func recorder_constant_binary(
	op token.Token, operands [2]constant.Value,
) (value constant.Value, ok bool) {
	left := operands[0]
	right := operands[1]
	if recorder_constant_is_shift(op) {
		count, exact := constant.Uint64Val(right)
		if !exact {
			return nil, false
		}
		result := constant.Shift(left, op, uint(count))
		return result, result.Kind() != constant.Unknown
	}
	if !recorder_constant_is_arithmetic(op) {
		return nil, false
	}
	if recorder_constant_divides(op) {
		if constant.Sign(right) == 0 {
			return nil, false
		}
	}
	result := constant.BinaryOp(left, op, right)
	return result, result.Kind() != constant.Unknown
}

// Reports whether op is a shift operator, whose right operand is a bit count rather than a value.
func recorder_constant_is_shift(op token.Token) (yes bool) {
	return op == token.SHL || op == token.SHR
}

// Reports whether op divides, so its zero right operand must be refused before the arithmetic.
func recorder_constant_divides(op token.Token) (yes bool) {
	return op == token.QUO || op == token.REM
}

// Reports whether op is a binary arithmetic or bitwise operator go/constant can apply directly.
func recorder_constant_is_arithmetic(op token.Token) (yes bool) {
	switch op {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
		token.AND, token.OR, token.XOR, token.AND_NOT:
		return true
	}
	return false
}

// Maps each package-level const's name to its value expression, for evaluating Range_Invariants
// bounds. Only a spec that supplies its own value is indexed; an inherited-value spec (no iota in
// this codebase) is skipped, so a bound built on one stays unresolved rather than silently
// miskeyed. A later definition wins on a name collision, matching ast_index_functions.
func ast_index_constants(files []*ast.File) (constants map[string]ast.Expr) {
	constants = map[string]ast.Expr{}
	for _, file := range files {
		for _, declaration := range file.Decls {
			generic, is_generic := declaration.(*ast.GenDecl)
			if !is_generic {
				continue
			}
			if generic.Tok != token.CONST {
				continue
			}
			for _, specification := range generic.Specs {
				value_specification, is_value := specification.(*ast.ValueSpec)
				if !is_value {
					continue
				}
				for i, name := range value_specification.Names {
					if i >= len(value_specification.Values) {
						continue
					}
					constants[name.Name] = value_specification.Values[i]
				}
			}
		}
	}
	return constants
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

// Finds the self-emitted Range_Invariants of a grid template — the call whose namespace argument is
// the function's trailing namespace parameter, so its bound grid is seeded at the _Invariants
// callsites. The Range mirror of recorder_template_dot_product.
func recorder_template_range(
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
		if ast_selector(candidate, true) != "Range_Invariants" {
			return true
		}
		if !ast_is_range_template(candidate, namespace_parameter) {
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
	axes []Registration_Axis, carves [][]Registration_Cell, reg *Registration,
) {
	if reg.Seen_Prefix[prefix] {
		reg.Collision = append(reg.Collision,
			recorder_position(file_set, position)+
				"  duplicate Dot_Product message: "+strconv.Quote(prefix))
		return
	}
	reg.Seen_Prefix[prefix] = true
	for _, axis := range axes {
		key := prefix + ELEMENT_MESSAGE_SEPARATOR + axis.Message
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
	ungated := make([]Registration_Axis, 0, len(axes))
	for _, axis := range axes {
		if !axis.Gated {
			ungated = append(ungated, axis)
		}
	}
	recorder_register_tuples(recorder, prefix, ungated, carves, nil)
}

// Reads a Dot_Product's inline element arguments into axes and carves. A self-emitting
// Dot_Product holds only inline Sometimes/Imply/Impossible — composition is separate _Invariants
// calls, not spreads — so there is no bundle descent. Each Impossible carve resolves its two
// references against the ungated axis positions; the grid excludes gated Imply axes.
func recorder_collect_inline(
	file_set *token.FileSet, arguments []ast.Expr, allow_unqualified bool, reg *Registration,
) (axes []Registration_Axis, carves [][]Registration_Cell) {
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
	index *Bundle_Index, imports map[string]string, bundle *ast.CallExpr,
) (function Indexed_Function, found bool) {
	qualifier, name := ast_bundle_qualifier(bundle)
	if qualifier == "" {
		same_package, present := index.Same_Set[name]
		return same_package, present
	}
	import_path, imported := imports[qualifier]
	if !imported {
		return Indexed_Function{}, false
	}
	cross_package, present := bundle_index_load(index, import_path)[name]
	return cross_package, present
}

// Resolves import_path to its absolute source directory, treating every import path as relative to
// the module path declared in go.mod: import_path must be the module path itself or sit under it,
// and the remainder names the directory under Module_Root. resolved is false for a path outside
// this module (an external dependency). This is pure string-prefix matching — the module path need
// not be a URL, so `module local` resolves `local/shared/foo` to <root>/shared/foo all the same.
func bundle_index_module_root(
	index *Bundle_Index, import_path string,
) (directory string, resolved bool) {
	if index.Module_Path == "" {
		return "", false
	}
	if import_path == index.Module_Path {
		return index.Module_Root, true
	}
	if strings.HasPrefix(import_path, index.Module_Path+"/") {
		return index.Module_Root + strings.TrimPrefix(import_path, index.Module_Path), true
	}
	return "", false
}

// Lazily parses the package at import_path and returns its functions by name,
// caching the result. The cache is seeded before parsing so a cyclic import
// resolves to the empty map rather than looping. Returns an empty map for a path
// outside this module (see bundle_index_module_root).
func bundle_index_load(
	index *Bundle_Index, import_path string,
) (functions map[string]Indexed_Function) {
	if cached, done := index.Loaded[import_path]; done {
		return cached
	}
	functions = map[string]Indexed_Function{}
	index.Loaded[import_path] = functions
	directory, resolved := bundle_index_module_root(index, import_path)
	if !resolved {
		return functions
	}
	files := recorder_parse_directory(&Recorder_Parse_Directory_Input{
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
// second argument. is_axis is false for any other selector.
func ast_axis_signature(
	selector string,
) (kind Assertion_Kind, condition_index int, gated bool, is_axis bool) {
	switch selector {
	case "Always":
		return ASSERTION_KIND_ALWAYS, 0, false, true
	case "Sometimes":
		return ASSERTION_KIND_SOMETIMES, 0, false, true
	case "Imply":
		return ASSERTION_KIND_SOMETIMES, 1, true, true
	case "Recorder_Always":
		return ASSERTION_KIND_ALWAYS, 1, false, true
	case "Recorder_Sometimes":
		return ASSERTION_KIND_SOMETIMES, 1, false, true
	case "Recorder_Imply":
		return ASSERTION_KIND_SOMETIMES, 2, true, true
	}
	return 0, 0, false, false
}

// Returns the axis for an Always/Sometimes constructor call, in either the bare sugar form or
// the explicit Recorder_* form; is_axis is false for any other call (Impossible, a bundle, a
// non-invariant call).
func recorder_axis_of(
	file_set *token.FileSet, call *ast.CallExpr, allow_unqualified bool, reg *Registration,
) (axis Registration_Axis, is_axis bool) {
	selector := ast_selector(call, allow_unqualified)
	if kind, condition_index, gated, ok := ast_axis_signature(selector); ok {
		condition := ast_condition_text(file_set, call, condition_index)
		bucket_count := 2
		if kind == ASSERTION_KIND_ALWAYS {
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
		return Registration_Axis{
			Message:      message,
			Condition:    condition,
			Kind:         kind,
			Bucket_Count: bucket_count,
			Gated:        gated,
		}, true
	}
	return Registration_Axis{}, false
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
	allow_unqualified bool, reg *Registration,
) (cells []Registration_Cell, ok bool) {
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
		cells = append(cells, Registration_Cell{Position: position, Bucket: bucket})
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
	recorder *Recorder, prefix string, axes []Registration_Axis, carves [][]Registration_Cell,
	chain_tuples map[Chain_Mask]Handle_Entry,
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
	var coordinate_axes []Registration_Axis
	if chain_tuples != nil {
		coordinate_axes = make([]Registration_Axis, len(coordinate_positions))
	}
	for j, position := range coordinate_positions {
		axis := axes[position]
		legend[j] = Tuple_Axis{
			Kind: axis.Kind, Condition: axis.Condition, Message: axis.Message}
		if chain_tuples != nil {
			coordinate_axes[j] = axis
		}
	}
	// The odometer still runs the full axis list so an Impossible's carve positions, which
	// index that full list, stay valid; each surviving tuple is then projected onto the
	// varying axes for the stored coordinate and key.
	tuple := make([]int, len(axes))
	for more := true; more; more = recorder_tuple_increment(tuple, axes) {
		projected := make([]int, len(coordinate_positions))
		for j, position := range coordinate_positions {
			projected[j] = tuple[position]
		}
		key := recorder_tuple_key(prefix, projected)
		if recorder_tuple_carved(tuple, carves) {
			// A carved cell is a forbidden combination: never a coverage entry (it
			// must never be witnessed) but a panic-able property the summary counts.
			// The glob expands here — every projected cell a carve matches is one.
			recorder.Forbidden.LoadOrStore(key, struct{}{})
			continue
		}
		metadata := &Assertion_Metadata{
			Kind:          ASSERTION_KIND_TUPLE,
			Message:       prefix,
			Tuple_Indices: projected,
			Axes:          legend,
		}
		value, _ := recorder.Events.LoadOrStore(key, metadata)
		if chain_tuples != nil {
			mask := chain_mask_from_tuple(projected, coordinate_axes)
			chain_tuples[mask] = Handle_Entry{
				Metadata: value.(*Assertion_Metadata), Key: key,
			}
		}
	}
}

func chain_mask_from_tuple(
	tuple []int, coordinate_axes []Registration_Axis,
) (mask Chain_Mask) {
	for i_index, bucket := range tuple {
		if bucket == 1 {
			axis := coordinate_axes[i_index]
			mask = chain_mask_with(mask, axis.Tuple_Position)
		}
	}
	return mask
}

// Advances tuple like an odometer over the axes' bucket counts; more is false
// once it wraps past the final combination.
func recorder_tuple_increment(tuple []int, axes []Registration_Axis) (more bool) {
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
func recorder_tuple_carved(tuple []int, carves [][]Registration_Cell) (carved bool) {
	for _, carve := range carves {
		if recorder_carve_matches(tuple, carve) {
			return true
		}
	}
	return false
}

// Reports whether tuple matches every cell of a single carve.
func recorder_carve_matches(tuple []int, carve []Registration_Cell) (matches bool) {
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

// A Coverage_Gap is one seeded assertion that the run failed to exercise, paired
// with the reason it counts as a gap (which branch or combination went unseen).
type Coverage_Gap struct {
	// Metadata is the seeded assertion that went unexercised.
	Metadata *Assertion_Metadata
	// Reason names why it counts as a gap: which branch or combination went unseen.
	Reason string
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
func recorder_collect_gaps(recorder *Recorder) (gaps []Coverage_Gap) {
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
func assertion_metadata_gaps(metadata *Assertion_Metadata) (gaps []Coverage_Gap) {
	if metadata.Kind == ASSERTION_KIND_SOMETIMES {
		if metadata.Frequency.Load() == 0 {
			gaps = append(gaps, Coverage_Gap{
				Metadata: metadata, Reason: "true branch never observed",
			})
		}
		if metadata.False_Frequency.Load() == 0 {
			gaps = append(gaps, Coverage_Gap{
				Metadata: metadata, Reason: "false branch never observed",
			})
		}
		return gaps
	}
	if metadata.Frequency.Load() != 0 {
		return gaps
	}
	if metadata.Kind == ASSERTION_KIND_TUPLE {
		return append(gaps, Coverage_Gap{Metadata: metadata, Reason: "never observed"})
	}
	return append(gaps, Coverage_Gap{Metadata: metadata, Reason: "never reached"})
}

// Prints the gaps to recorder.Output in v2's three sections — cross-product,
// branch, reachability — each sorted by site. A banner carrying the gap count
// brackets the report so the verdict survives a top-down or bottom-up skim.
func recorder_report_gaps(recorder *Recorder, gaps []Coverage_Gap) {
	banner := "🚨 " + strconv.Itoa(len(gaps)) + " coverage gaps 🚨"
	fmt.Fprintln(recorder.Output, banner)
	recorder_report_cross_product(recorder.Output, gaps)
	recorder_report_section(&Recorder_Report_Section_Input{
		Output: recorder.Output, Title: "Branch gaps", Gaps: gaps,
		Kind: ASSERTION_KIND_SOMETIMES,
	})
	recorder_report_section(&Recorder_Report_Section_Input{
		Output: recorder.Output, Title: "Reachability gaps", Gaps: gaps,
		Kind: ASSERTION_KIND_ALWAYS,
	})
	fmt.Fprintln(recorder.Output, banner)
}

// Input for recorder_report_section.
type Recorder_Report_Section_Input struct {
	// Output is the writer the section is printed to.
	Output io.Writer
	// Title is the markdown heading the section is printed under.
	Title string
	// Gaps is the full gap set; only those matching Kind are printed.
	Gaps []Coverage_Gap
	// Kind selects which assertion kind's gaps this section reports.
	Kind Assertion_Kind
}

// Prints, under a markdown heading, the gaps whose assertion is of the given
// kind, sorted by message. Emits nothing when no gap matches, so empty sections
// stay silent.
func recorder_report_section(input *Recorder_Report_Section_Input) {
	selected := make([]Coverage_Gap, 0, len(input.Gaps))
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
func recorder_report_cross_product(output io.Writer, gaps []Coverage_Gap) {
	by_prefix := map[string][]Coverage_Gap{}
	var prefixes []string
	for _, gap := range gaps {
		if gap.Metadata.Kind != ASSERTION_KIND_TUPLE {
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
func coverage_gap_cell(cell Coverage_Gap) (line string) {
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
	if kind == ASSERTION_KIND_ALWAYS {
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
func coverage_gap_line(gap Coverage_Gap) (line string) {
	metadata := gap.Metadata
	return message_display(metadata.Message) + "  " + assertion_kind_name(metadata.Kind) +
		" — " + gap.Reason + ": " + strconv.Quote(metadata.Condition)
}

// Renders a coverage key for the report: the element separator (the NUL joining a
// Dot_Product prefix to an axis message) shows as " · " so "signup.username␀empty" reads
// as "signup.username · empty". A bare message (an Always, or a grid prefix) is unchanged.
func message_display(message string) (display string) {
	return strings.ReplaceAll(message, ELEMENT_MESSAGE_SEPARATOR, " · ")
}

// Returns the report label for a kind: the same word the static pass keys on.
func assertion_kind_name(kind Assertion_Kind) (name string) {
	if kind == ASSERTION_KIND_SOMETIMES {
		return "Sometimes"
	}
	if kind == ASSERTION_KIND_TUPLE {
		return "Tuple"
	}
	return "Always"
}

// Recorder_Assertion_Summary renders the clean-run banner naming how many
// properties the run tested: an Always is one individual property, a Sometimes is
// two (its true and its false branch are separate obligations); Tuple entries and
// the cells an Impossible carves are combinations; the Always family plus every
// carved cell is the panic-able subset whose violation fails fatally at runtime.
func Recorder_Assertion_Summary(recorder *Recorder) (summary string) {
	individual := 0
	combinations := 0
	panic_able := 0
	recorder.Events.Range(func(key, value any) (continue_iteration bool) {
		metadata := value.(*Assertion_Metadata)
		switch metadata.Kind {
		case ASSERTION_KIND_TUPLE:
			combinations++
		case ASSERTION_KIND_ALWAYS:
			individual++
			panic_able++
		default:
			// A Sometimes — including a gated Imply, recorded as a Sometimes —
			// must witness both its true and its false branch, so it counts twice.
			individual += 2
		}
		return true
	})
	// Each carved cell is a forbidden combination, counted as a panic-able
	// combination: reaching it fails fatally and it lives in the grid.
	recorder.Forbidden.Range(func(key, value any) (continue_iteration bool) {
		combinations++
		panic_able++
		return true
	})
	if recorder.Package_Label != "" {
		return fmt.Sprintf(
			"✓ %s: tested %d properties (%d individual + %d combinations, "+
				"of which %d are panic-able)",
			recorder.Package_Label,
			individual+combinations, individual, combinations, panic_able,
		)
	}
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
