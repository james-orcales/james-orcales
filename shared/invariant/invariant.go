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
	"math/big"
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
// it in every namespace and message. Two separators distinguish chain handles from the flat tuple
// and eager-Always keys that fuzz persistence also carries.
const ELEMENT_MESSAGE_SEPARATOR = "\x00"

// Bounds registration's backwards walks over fluent receivers. A valid chain stops at its root
// far below this because the independently enforced ordinal space has only 255 links; the larger
// bound keeps malformed syntax from making analysis depend unboundedly on source depth.
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

// PRODUCT_FAILURE_PRESET_UPPER defers a typed upper-bound violation until Ensure.
const PRODUCT_FAILURE_PRESET_UPPER = uint8(6)

// PRODUCT_FAILURE_PRESET_LOWER defers a typed lower-bound violation until Ensure.
const PRODUCT_FAILURE_PRESET_LOWER = uint8(7)

// PRODUCT_FAILURE_PRESET_EXCLUDED defers a Range exclusion until Ensure.
const PRODUCT_FAILURE_PRESET_EXCLUDED = uint8(8)

// PRODUCT_FAILURE_PRESET_MEMBER defers an Enum membership violation until Ensure.
const PRODUCT_FAILURE_PRESET_MEMBER = uint8(9)

// PRODUCT_FAILURE_ENUM_DOMAIN prevents an invalid member set from acquiring a runtime shape.
const PRODUCT_FAILURE_ENUM_DOMAIN = uint8(10)

// PRODUCT_FAILURE_RANGE_EXCLUSION prevents a hole from contradicting a demanded boundary axis.
const PRODUCT_FAILURE_RANGE_EXCLUSION = uint8(11)

// One message keeps registration and deferred runtime rejection on the same contract.
const PRODUCT_FAILURE_RANGE_EXCLUSION_MESSAGE = "Range exclusions must be " +
	"strictly inside its boundaries"

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

// DOT_ELEMENT_KIND_GUARD distinguishes a preset's one-bucket reachability link from tuple axes.
const DOT_ELEMENT_KIND_GUARD Engine_Element_Kind = 3

// CHAIN_PRESET_KIND_RANGE identifies a bounded interval expansion.
const CHAIN_PRESET_KIND_RANGE Chain_Preset_Kind = 1

// CHAIN_PRESET_KIND_ENUM identifies a discrete member-set expansion.
const CHAIN_PRESET_KIND_ENUM Chain_Preset_Kind = 2

// CHAIN_INTEGER_KIND_INT identifies the platform int method, which this repository pins to 64 bits.
const CHAIN_INTEGER_KIND_INT Chain_Integer_Kind = 1

// CHAIN_INTEGER_KIND_INT8 identifies the int8 method.
const CHAIN_INTEGER_KIND_INT8 Chain_Integer_Kind = 2

// CHAIN_INTEGER_KIND_INT16 identifies the int16 method.
const CHAIN_INTEGER_KIND_INT16 Chain_Integer_Kind = 3

// CHAIN_INTEGER_KIND_INT32 identifies the int32 method.
const CHAIN_INTEGER_KIND_INT32 Chain_Integer_Kind = 4

// CHAIN_INTEGER_KIND_INT64 identifies the int64 method.
const CHAIN_INTEGER_KIND_INT64 Chain_Integer_Kind = 5

// CHAIN_INTEGER_KIND_UINT identifies the platform uint method.
const CHAIN_INTEGER_KIND_UINT Chain_Integer_Kind = 6

// CHAIN_INTEGER_KIND_UINT8 identifies the uint8 method.
const CHAIN_INTEGER_KIND_UINT8 Chain_Integer_Kind = 7

// CHAIN_INTEGER_KIND_UINT16 identifies the uint16 method.
const CHAIN_INTEGER_KIND_UINT16 Chain_Integer_Kind = 8

// CHAIN_INTEGER_KIND_UINT32 identifies the uint32 method.
const CHAIN_INTEGER_KIND_UINT32 Chain_Integer_Kind = 9

// CHAIN_INTEGER_KIND_UINT64 identifies the uint64 method.
const CHAIN_INTEGER_KIND_UINT64 Chain_Integer_Kind = 10

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

	// Chain_Shapes_Mu serializes publication only; the warmed read path never takes it,
	// because even a read-lock is a write to one shared word, and that word's cache line
	// ping-pongs across every worker on every Dot_Product root.
	Chain_Shapes_Mu sync.Mutex
	// Chain_Shapes publishes the namespace-to-shape map copy-on-write behind an atomic
	// pointer so a warmed root reaches its shape through a plain load. It is keyed by
	// namespace because one namespace names exactly one chain; Product retains the resolved
	// pointer so fluent links do not repeat the map lookup.
	Chain_Shapes atomic.Pointer[map[Namespace]*Chain_Shape]
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

	// Sugar_Package is the import path of the recorder-less sugar tier. When a template
	// resolved from that package is descended, its unqualified invariant calls are recognized;
	// empty keeps identically named calls elsewhere from becoming assertions.
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

// Engine_Element_Kind keeps a chain link's zero value invalid.
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

// Chain_Preset_Kind distinguishes the two typed preset expansions.
type Chain_Preset_Kind uint8

// Chain_Integer_Kind pins the exact primitive method used at the analyzed callsite.
type Chain_Integer_Kind uint8

// Chain_Integer_Value is an exact allocation-free mathematical integer representation.
type Chain_Integer_Value struct {
	// Magnitude carries the absolute value, including every uint64 value.
	Magnitude uint64
	// Negative distinguishes signed negative values without sacrificing uint64 range.
	Negative bool
}

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
	// Guards holds preset bound reachability handles in fluent order.
	Guards []Chain_Guard
	// Presets holds registration's complete typed expansion plan.
	Presets []Chain_Preset
	// Tuples resolves a packed mask directly to its pre-seeded coverage entry.
	// The map prevents a dense array from imposing a width smaller than the chain ordinal.
	Tuples map[Chain_Mask]Handle_Entry
	// Registered distinguishes analyzed chains from enforcement-only foreign chains.
	Registered bool
	// Ensured prevents a discovered shape from growing after its first complete execution.
	Ensured atomic.Bool
}

// Chain_Guard is one preset bound obligation, excluded from tuple coordinates.
type Chain_Guard struct {
	// Ordinal selects the guard result captured by the typed method.
	Ordinal uint8
	// Message names the bound obligation and violation.
	Message string
	// Entry is registration's exact reachability handle.
	Entry Handle_Entry
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

// Chain_Preset is registration's authoritative expansion for one typed method call.
type Chain_Preset struct {
	// Kind distinguishes bounded intervals from member sets.
	Kind Chain_Preset_Kind
	// Integer_Kind pins the concrete exported method.
	Integer_Kind Chain_Integer_Kind
	// Ordinal is the first generated guard link.
	Ordinal uint8
	// Link_Count advances the surrounding fluent chain past the complete expansion.
	Link_Count uint8
	// Axis_Start indexes the first generated axis in Chain_Shape.Axes.
	Axis_Start uint8
	// Axis_Count is the number of generated tuple axes.
	Axis_Count uint8
	// Minimum is the compiled inclusive lower bound.
	Minimum Chain_Integer_Value
	// Maximum is the compiled inclusive upper bound.
	Maximum Chain_Integer_Value
	// Values are exclusions for Range and members for Enum.
	Values []Chain_Integer_Value
	// Axis_Values align positionally with the generated axes.
	Axis_Values []Chain_Integer_Value
}

// Chain_Preset_Axis is one equality axis generated by a typed preset.
type Chain_Preset_Axis struct {
	// Ordinal is the generated Sometimes link position.
	Ordinal uint8
	// Tuple_Position is the axis's coordinate in the surrounding product.
	Tuple_Position uint8
	// Message is the public sibling vocabulary for the generated axis.
	Message string
	// Value is the exact integer whose equality makes the axis true.
	Value Chain_Integer_Value
}

// Chain_Preset_Expansion is the one canonical description registration and discovery publish.
type Chain_Preset_Expansion struct {
	// Valid is false when the expansion would exceed the shared ordinal space.
	Valid bool
	// Preset advances runtime through this complete expansion.
	Preset Chain_Preset
	// Guards are excluded from tuple coordinates but retain reachability.
	Guards []Chain_Guard
	// Axes become ordinary siblings in the surrounding product.
	Axes []Chain_Preset_Axis
	// Links pin every generated guard, axis, and constraint ordinal.
	Links []Chain_Link
	// Rules are the generated mutual-exclusion and saturation carves.
	Rules []Chain_Rule
	// Carves seed the demanded grid minus generated impossible cells.
	Carves [][]Registration_Cell
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
	// Rule_Index resolves an Impossible directly to registration's compiled rule.
	Rule_Index uint8
	// Preset_Index resolves the first generated guard directly to its compiled preset.
	Preset_Index uint8
}

// Chain_Rule is one Impossible compiled to the packed-mask predicate mask&Mask == Want.
type Chain_Rule struct {
	// Mask selects every named axis and globs over all others.
	Mask Chain_Mask
	// Want contains the selected axes' forbidden polarities.
	Want Chain_Mask
	// Message names the constraint when it fires.
	Message string
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
	// Preset_Failure retains the first typed guard violation for Ensure.
	Preset_Failure uint8
	// Preset_Failure_Ordinal identifies the preset that supplied the violation.
	Preset_Failure_Ordinal uint8
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

func chain_enum_domain_valid[Value comparable](members []Value) (valid bool) {
	if len(members) < 2 {
		return false
	}
	first := members[0]
	for _, member := range members[1:] {
		if member != first {
			return true
		}
	}
	return false
}

func product_range[
	Value ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64,
](
	product Product, integer_kind Chain_Integer_Kind,
	value Value, minimum Value, maximum Value, excluded []Value,
) (next Product) {
	minimum_integer := chain_integer_value(minimum)
	maximum_integer := chain_integer_value(maximum)
	for _, hole := range excluded {
		if !chain_integer_between(
			minimum_integer, chain_integer_value(hole), maximum_integer) {
			return product.chain_defer_failure(PRODUCT_FAILURE_RANGE_EXCLUSION)
		}
	}
	return product_preset(
		product, CHAIN_PRESET_KIND_RANGE, integer_kind,
		value, minimum, maximum, excluded)
}

func chain_integer_bounds[
	Value ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64,
](members []Value) (minimum Value, maximum Value) {
	if len(members) > 0 {
		minimum, maximum = members[0], members[0]
	}
	for _, member := range members {
		if member < minimum {
			minimum = member
		}
		if member > maximum {
			maximum = member
		}
	}
	return minimum, maximum
}

// The public methods are concrete, while this free generic helper is monomorphized for direct
// comparisons and keeps the twenty wrappers on one execution path without interface conversion.
func product_preset[
	Value ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64,
](
	product Product, kind Chain_Preset_Kind, integer_kind Chain_Integer_Kind,
	value Value, minimum Value, maximum Value, values []Value,
) (next Product) {
	if product.Failure != 0 {
		return product
	}
	preset, mismatch := chain_preset_resolve(
		product.Shape, kind, integer_kind, product.Ordinal,
		product.Axis_Count, minimum, maximum, values)
	if preset == nil {
		return product.chain_defer_failure(PRODUCT_FAILURE_LINKS)
	}
	if mismatch {
		product.Mismatch = true
	}
	product = product_preset_observe(product, preset, value, minimum, maximum, values)
	product.Ordinal += preset.Link_Count
	product.Axis_Count += preset.Axis_Count
	return product
}

func product_preset_observe[
	Value ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64,
](
	product Product, preset *Chain_Preset,
	value Value, minimum Value, maximum Value, values []Value,
) (next Product) {
	if preset.Kind == CHAIN_PRESET_KIND_RANGE {
		product = product_range_verdict(
			product, preset.Ordinal, value, minimum, maximum, values)
	} else if !chain_integer_slice_contains(values, value) {
		product = product.chain_defer_preset_failure(
			PRODUCT_FAILURE_PRESET_MEMBER, preset.Ordinal)
	}
	if product.Preset_Failure == 0 {
		product.Observations = chain_mask_with(product.Observations, preset.Ordinal)
		product.Observations = chain_mask_with(product.Observations, preset.Ordinal+1)
	}
	value_integer := chain_integer_value(value)
	for axis_index := uint8(0); axis_index < preset.Axis_Count; axis_index++ {
		if preset.Axis_Values[axis_index] != value_integer {
			continue
		}
		axis := product.Shape.Axes[preset.Axis_Start+axis_index]
		product.Observations = chain_mask_with(product.Observations, axis.Ordinal)
	}
	return product
}

func product_range_verdict[
	Value ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64,
](
	product Product, ordinal uint8, value Value, minimum Value, maximum Value, excluded []Value,
) (next Product) {
	if value > maximum {
		return product.chain_defer_preset_failure(PRODUCT_FAILURE_PRESET_UPPER, ordinal)
	}
	if value < minimum {
		return product.chain_defer_preset_failure(PRODUCT_FAILURE_PRESET_LOWER, ordinal)
	}
	if chain_integer_slice_contains(excluded, value) {
		return product.chain_defer_preset_failure(PRODUCT_FAILURE_PRESET_EXCLUDED, ordinal)
	}
	return product
}

func chain_integer_slice_contains[Value comparable](values []Value, value Value) (has bool) {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func (product Product) chain_defer_preset_failure(failure uint8, ordinal uint8) (next Product) {
	if product.Preset_Failure == 0 {
		product.Preset_Failure = failure
		product.Preset_Failure_Ordinal = ordinal
	}
	return product
}

func chain_integer_value[
	Value ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64,
](value Value) (integer Chain_Integer_Value) {
	if value < Value(0) {
		signed := int64(value)
		integer.Negative = true
		integer.Magnitude = uint64(-(signed + 1)) + 1
		return integer
	}
	integer.Magnitude = uint64(value)
	return integer
}

func chain_integer_compare(first Chain_Integer_Value, second Chain_Integer_Value) (order int) {
	if first.Negative != second.Negative {
		if first.Negative {
			return -1
		}
		return 1
	}
	if first.Magnitude == second.Magnitude {
		return 0
	}
	if first.Negative {
		if first.Magnitude > second.Magnitude {
			return -1
		}
		return 1
	}
	if first.Magnitude < second.Magnitude {
		return -1
	}
	return 1
}

func chain_integer_between(
	minimum Chain_Integer_Value, candidate Chain_Integer_Value, maximum Chain_Integer_Value,
) (inside bool) {
	return chain_integer_compare(minimum, candidate) < 0 &&
		chain_integer_compare(candidate, maximum) < 0
}

func chain_integer_values_contains(
	values []Chain_Integer_Value, value Chain_Integer_Value,
) (has bool) {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func chain_preset_resolve[
	Value ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64,
](
	shape *Chain_Shape, kind Chain_Preset_Kind, integer_kind Chain_Integer_Kind,
	ordinal uint8, axis_count uint8, minimum Value, maximum Value, values []Value,
) (preset *Chain_Preset, mismatch bool) {
	if shape.chain_replays() {
		return chain_preset_replay(
			shape, kind, integer_kind, ordinal, axis_count, minimum, maximum, values)
	}
	return chain_preset_discover(
		shape, kind, integer_kind, ordinal, axis_count, minimum, maximum, values)
}

func chain_preset_replay[
	Value ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64,
](
	shape *Chain_Shape, kind Chain_Preset_Kind, integer_kind Chain_Integer_Kind,
	ordinal uint8, axis_count uint8, minimum Value, maximum Value, values []Value,
) (preset *Chain_Preset, mismatch bool) {
	if int(ordinal) >= len(shape.Links) {
		return nil, true
	}
	link := shape.Links[ordinal]
	if link.Kind != DOT_ELEMENT_KIND_GUARD {
		return nil, true
	}
	if int(link.Preset_Index) >= len(shape.Presets) {
		return nil, true
	}
	preset = &shape.Presets[link.Preset_Index]
	if preset.Kind != kind {
		mismatch = true
	}
	if preset.Integer_Kind != integer_kind {
		mismatch = true
	}
	if preset.Ordinal != ordinal {
		mismatch = true
	}
	if preset.Axis_Start != axis_count {
		mismatch = true
	}
	if preset.Minimum != chain_integer_value(minimum) {
		mismatch = true
	}
	if preset.Maximum != chain_integer_value(maximum) {
		mismatch = true
	}
	if len(preset.Values) != len(values) {
		return preset, true
	}
	for value_index, value := range values {
		if preset.Values[value_index] != chain_integer_value(value) {
			mismatch = true
		}
	}
	return preset, mismatch
}

// Foreign discovery copies variadic domain values once because the caller's stack-backed slice
// cannot outlive the method; registered replay never takes this path or performs the copy.
func chain_preset_discover[
	Value ~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64,
](
	shape *Chain_Shape, kind Chain_Preset_Kind, integer_kind Chain_Integer_Kind,
	ordinal uint8, axis_count uint8, minimum Value, maximum Value, values []Value,
) (preset *Chain_Preset, mismatch bool) {
	shape.Mu.Lock()
	defer shape.Mu.Unlock()
	if int(ordinal) < len(shape.Links) {
		return chain_preset_replay(
			shape, kind, integer_kind, ordinal, axis_count, minimum, maximum, values)
	}
	integer_values := make([]Chain_Integer_Value, len(values))
	for value_index, value := range values {
		integer_values[value_index] = chain_integer_value(value)
	}
	expansion := chain_preset_expand(Chain_Preset{
		Kind: kind, Integer_Kind: integer_kind,
		Ordinal: ordinal, Axis_Start: axis_count,
		Minimum: chain_integer_value(minimum), Maximum: chain_integer_value(maximum),
		Values: integer_values,
	}, uint8(len(shape.Rules)))
	if !expansion.Valid {
		return nil, true
	}
	preset_index := uint8(len(shape.Presets))
	expansion.Links[0].Preset_Index = preset_index
	shape.Presets = append(shape.Presets, expansion.Preset)
	shape.Guards = append(shape.Guards, expansion.Guards...)
	for _, axis := range expansion.Axes {
		shape.Axes = append(shape.Axes, Chain_Axis{
			Ordinal: axis.Ordinal, Tuple_Position: axis.Tuple_Position,
			Message: axis.Message,
		})
	}
	shape.Links = append(shape.Links, expansion.Links...)
	shape.Rules = append(shape.Rules, expansion.Rules...)
	return &shape.Presets[preset_index], false
}

func chain_preset_expand(
	preset Chain_Preset, rule_start uint8,
) (expansion Chain_Preset_Expansion) {
	axes := chain_preset_axes(preset)
	saturated := chain_preset_saturated(preset, axes)
	generated_rule_count := len(axes) * (len(axes) - 1) / 2
	if saturated {
		generated_rule_count++
	}
	link_count := 2 + len(axes) + generated_rule_count
	if int(preset.Ordinal)+link_count > CHAIN_LINKS_MAX {
		return expansion
	}
	expansion.Valid = true
	preset.Link_Count = uint8(link_count)
	preset.Axis_Count = uint8(len(axes))
	preset.Axis_Values = make([]Chain_Integer_Value, len(axes))
	expansion.Preset = preset
	expansion.Guards = []Chain_Guard{
		{Ordinal: preset.Ordinal, Message: RANGE_GUARD_UPPER},
		{Ordinal: preset.Ordinal + 1, Message: RANGE_GUARD_LOWER},
	}
	expansion.Links = []Chain_Link{
		{
			Kind:    DOT_ELEMENT_KIND_GUARD,
			Ordinal: preset.Ordinal, Axis_Count: preset.Axis_Start,
		},
		{
			Kind:    DOT_ELEMENT_KIND_GUARD,
			Ordinal: preset.Ordinal + 1, Axis_Count: preset.Axis_Start,
		},
	}
	for axis_index, axis := range axes {
		expansion.Preset.Axis_Values[axis_index] = axis.Value
		expansion.Axes = append(expansion.Axes, axis)
		expansion.Links = append(expansion.Links, Chain_Link{
			Kind: DOT_ELEMENT_KIND_SOMETIMES, Ordinal: axis.Ordinal,
			Axis_Count: preset.Axis_Start + uint8(axis_index), Message: axis.Message,
		})
	}
	chain_preset_append_pairwise(preset, rule_start, axes, &expansion)
	if saturated {
		chain_preset_append_saturation(preset, rule_start, axes, &expansion)
	}
	return expansion
}

func chain_preset_axes(preset Chain_Preset) (axes []Chain_Preset_Axis) {
	values := []struct {
		Integer Chain_Integer_Value
		Message string
	}{
		{Integer: Chain_Integer_Value{}, Message: RANGE_MESSAGE_ZERO},
		{Integer: Chain_Integer_Value{Magnitude: 1}, Message: RANGE_MESSAGE_ONE},
		{Integer: Chain_Integer_Value{Magnitude: 2}, Message: RANGE_MESSAGE_TWO},
		{Integer: Chain_Integer_Value{Magnitude: 1, Negative: true},
			Message: RANGE_MESSAGE_NEGATIVE_ONE},
	}
	if chain_integer_compare(preset.Minimum, preset.Maximum) < 0 {
		axes = chain_preset_append_axis(preset, axes, preset.Minimum, RANGE_MESSAGE_MINIMUM)
		axes = chain_preset_append_axis(preset, axes, preset.Maximum, RANGE_MESSAGE_MAXIMUM)
	}
	for value_index, value := range values {
		if value_index == 3 {
			if !chain_integer_kind_signed(preset.Integer_Kind) {
				continue
			}
		}
		if !chain_integer_between(preset.Minimum, value.Integer, preset.Maximum) {
			continue
		}
		member := chain_integer_values_contains(preset.Values, value.Integer)
		if preset.Kind == CHAIN_PRESET_KIND_RANGE {
			if member {
				continue
			}
		}
		if preset.Kind == CHAIN_PRESET_KIND_ENUM {
			if !member {
				continue
			}
		}
		axes = chain_preset_append_axis(preset, axes, value.Integer, value.Message)
	}
	return axes
}

func chain_preset_append_axis(
	preset Chain_Preset, axes []Chain_Preset_Axis,
	value Chain_Integer_Value, message string,
) (expanded []Chain_Preset_Axis) {
	return append(axes, Chain_Preset_Axis{
		Ordinal:        preset.Ordinal + 2 + uint8(len(axes)),
		Tuple_Position: preset.Axis_Start + uint8(len(axes)),
		Message:        message, Value: value,
	})
}

func chain_integer_kind_signed(kind Chain_Integer_Kind) (signed bool) {
	return kind >= CHAIN_INTEGER_KIND_INT && kind <= CHAIN_INTEGER_KIND_INT64
}

func chain_preset_saturated(
	preset Chain_Preset, axes []Chain_Preset_Axis,
) (saturated bool) {
	if len(axes) == 0 {
		return false
	}
	if preset.Kind == CHAIN_PRESET_KIND_ENUM {
		for _, member := range preset.Values {
			if !chain_preset_axes_contains(axes, member) {
				return false
			}
		}
		return true
	}
	current := preset.Minimum
	for step := 0; step <= len(axes)+len(preset.Values); step++ {
		excluded := chain_integer_values_contains(preset.Values, current)
		witnessed := chain_preset_axes_contains(axes, current)
		if !excluded {
			if !witnessed {
				return false
			}
		}
		if current == preset.Maximum {
			return true
		}
		next, exists := chain_integer_successor(current)
		if !exists {
			return false
		}
		current = next
	}
	return false
}

func chain_preset_axes_contains(axes []Chain_Preset_Axis, value Chain_Integer_Value) (has bool) {
	for _, axis := range axes {
		if axis.Value == value {
			return true
		}
	}
	return false
}

func chain_integer_successor(
	value Chain_Integer_Value,
) (next Chain_Integer_Value, exists bool) {
	if value.Negative {
		if value.Magnitude == 1 {
			return Chain_Integer_Value{}, true
		}
		value.Magnitude--
		return value, true
	}
	if value.Magnitude == ^uint64(0) {
		return next, false
	}
	value.Magnitude++
	return value, true
}

func chain_preset_append_pairwise(
	preset Chain_Preset, rule_start uint8, axes []Chain_Preset_Axis,
	expansion *Chain_Preset_Expansion,
) {
	for first_index := range axes {
		for second_index := first_index + 1; second_index < len(axes); second_index++ {
			cells := []Registration_Cell{
				{Position: int(axes[first_index].Tuple_Position), Bucket: 1},
				{Position: int(axes[second_index].Tuple_Position), Bucket: 1},
			}
			chain_preset_append_rule(preset, rule_start, cells, true, expansion)
		}
	}
}

func chain_preset_append_saturation(
	preset Chain_Preset, rule_start uint8, axes []Chain_Preset_Axis,
	expansion *Chain_Preset_Expansion,
) {
	cells := make([]Registration_Cell, len(axes))
	for axis_index, axis := range axes {
		cells[axis_index] = Registration_Cell{Position: int(axis.Tuple_Position), Bucket: 0}
	}
	chain_preset_append_rule(preset, rule_start, cells, false, expansion)
}

func chain_preset_append_rule(
	preset Chain_Preset, rule_start uint8, cells []Registration_Cell, true_events bool,
	expansion *Chain_Preset_Expansion,
) {
	ordinal := preset.Ordinal + uint8(len(expansion.Links))
	rule_index := rule_start + uint8(len(expansion.Rules))
	message := "Typed preset constraint at link " + strconv.Itoa(int(ordinal)) + "."
	link := Chain_Link{
		Kind: DOT_ELEMENT_KIND_IMPOSSIBLE, Ordinal: ordinal,
		Axis_Count: preset.Axis_Start + uint8(len(expansion.Axes)),
		Message:    message, Reference_Count: uint8(len(cells)), Rule_Index: rule_index,
	}
	var mask Chain_Mask
	var want Chain_Mask
	for cell_index, cell := range cells {
		position := uint8(cell.Position)
		link.References[cell_index] = position
		mask = chain_mask_with(mask, position)
		if true_events {
			link.Reference_Events = chain_mask_with(
				link.Reference_Events, uint8(cell_index))
			want = chain_mask_with(want, position)
		}
	}
	expansion.Links = append(expansion.Links, link)
	expansion.Rules = append(expansion.Rules, Chain_Rule{
		Mask: mask, Want: want, Message: message,
	})
	expansion.Carves = append(expansion.Carves, cells)
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
	case PRODUCT_FAILURE_ENUM_DOMAIN:
		return "Enum requires at least two distinct members"
	case PRODUCT_FAILURE_RANGE_EXCLUSION:
		return PRODUCT_FAILURE_RANGE_EXCLUSION_MESSAGE
	}
	return ""
}

func (product Product) chain_preset_failure_message() (message string) {
	if product.Preset_Failure == 0 {
		return ""
	}
	prefix := string(product.Namespace) + ELEMENT_MESSAGE_SEPARATOR
	switch product.Preset_Failure {
	case PRODUCT_FAILURE_PRESET_UPPER:
		return prefix + RANGE_GUARD_UPPER + "  value exceeds max"
	case PRODUCT_FAILURE_PRESET_LOWER:
		return prefix + RANGE_GUARD_LOWER + "  value below min"
	case PRODUCT_FAILURE_PRESET_EXCLUDED:
		return prefix + RANGE_GUARD_EXCLUDED + "  value is excluded"
	case PRODUCT_FAILURE_PRESET_MEMBER:
		return prefix + RANGE_GUARD_EXCLUDED + "  not a member"
	}
	return "typed preset failed"
}

// Registration and a completed foreign discovery both make the shape immutable; checking the
// latter under its mutex avoids racing two first executions while keeping every replay read-only.
func (shape *Chain_Shape) chain_replays() (replays bool) {
	if shape.Registered {
		return true
	}
	return shape.Ensured.Load()
}

func (product Product) ensure_shape() {
	if failure := product.chain_failure_message(); failure != "" {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + failure)
	}
	if product.Axis_Count == 0 {
		if len(product.Shape.Guards) == 0 {
			panic(ASSERTION_FAILURE_MESSAGE_PREFIX + "Dot_Product has no axes")
		}
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
	if failure := product.chain_preset_failure_message(); failure != "" {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + failure)
	}
}

// Resolving every handle before crediting keeps an unknown key from partially crediting the call.
func (product Product) chain_handle(tuple_mask Chain_Mask) (tuple Handle_Entry) {
	for _, guard := range product.Shape.Guards {
		if guard.Entry.Metadata == nil {
			message := "registered Dot_Product credited unknown guard " +
				strconv.Quote(guard.Message)
			panic(ASSERTION_FAILURE_MESSAGE_PREFIX + message)
		}
	}
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
// The warmed lookup is a plain load of a copy-on-write map: discovery pays a full copy per new
// namespace so the per-call path writes nothing shared.
func recorder_chain_shape(recorder *Recorder, namespace Namespace) (shape *Chain_Shape) {
	shapes := recorder.Chain_Shapes.Load()
	if shapes != nil {
		shape = (*shapes)[namespace]
	}
	if shape != nil {
		return shape
	}
	recorder.Chain_Shapes_Mu.Lock()
	defer recorder.Chain_Shapes_Mu.Unlock()
	shapes = recorder.Chain_Shapes.Load()
	if shapes != nil {
		shape = (*shapes)[namespace]
	}
	if shape != nil {
		return shape
	}
	// A NUL namespace would make the serialized axis key ambiguous. Rejecting it here, before
	// publication, keeps the warmed root scan-free without weakening the guarantee: the shape
	// is never published, so every offending call re-enters this miss path and re-panics.
	if strings.Contains(string(namespace), ELEMENT_MESSAGE_SEPARATOR) {
		panic(ASSERTION_FAILURE_MESSAGE_PREFIX + "Dot_Product namespace contains NUL")
	}
	shape = &Chain_Shape{}
	recorder_chain_shape_publish(recorder, namespace, shape)
	return shape
}

// The caller holds Chain_Shapes_Mu, so two publishers cannot lose each other's entries; a
// reader's load observes either the old complete map or the new one, never a partial insert.
func recorder_chain_shape_publish(recorder *Recorder, namespace Namespace, shape *Chain_Shape) {
	shapes := recorder.Chain_Shapes.Load()
	next := map[Namespace]*Chain_Shape{}
	if shapes != nil {
		for key, value := range *shapes {
			next[key] = value
		}
	}
	next[namespace] = shape
	recorder.Chain_Shapes.Store(&next)
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

// Registered shapes are immutable and can use direct reads without synchronization; only
// first-executed foreign shapes pay the mutex needed to discover their link sequence. Stored
// messages are NUL-free by discovery and registration, so a warmed match proves the incoming
// message clean without a scan; only the mismatch branch and discovery still pay one.
func (shape *Chain_Shape) chain_axis(
	ordinal uint8, axis_count uint8, message string,
) (mismatch bool, failure uint8) {
	if shape.chain_replays() {
		if int(ordinal) >= len(shape.Links) {
			return true, 0
		}
		link := shape.Links[ordinal]
		mismatch = link.Kind != DOT_ELEMENT_KIND_SOMETIMES
		if link.Message != message {
			if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
				return false, PRODUCT_FAILURE_SOMETIMES
			}
			mismatch = true
		}
		if int(axis_count) >= len(shape.Axes) {
			return true, 0
		}
		axis := shape.Axes[axis_count]
		if axis.Ordinal != ordinal {
			mismatch = true
		}
		if axis.Message != message {
			mismatch = true
		}
		return mismatch, 0
	}
	if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
		return false, PRODUCT_FAILURE_SOMETIMES
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
	} else if !shape.Ensured.Load() {
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
	} else if !shape.Ensured.Load() {
		if int(axis_count) == len(shape.Axes) {
			shape.Axes = append(shape.Axes, axis)
		} else {
			mismatch = true
		}
	} else {
		mismatch = true
	}
	return mismatch, 0
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
		Mask: mask, Want: want, Message: link.Message,
	}
	return link, rule, 0
}

// Constraints are resolved while the fluent value still carries the number of preceding axes;
// this makes a forward or non-sibling reference fail before it can erase a demanded obligation.
func (shape *Chain_Shape) chain_rule(
	link Chain_Link, references []Dot_Element_Reference,
) (rule Chain_Rule, mismatch bool, failure uint8) {
	if shape.chain_replays() {
		return shape.chain_rule_replay(
			link.Ordinal, link.Axis_Count, link.Message, references)
	}
	if strings.Contains(link.Message, ELEMENT_MESSAGE_SEPARATOR) {
		return rule, false, PRODUCT_FAILURE_IMPOSSIBLE
	}
	shape.Mu.Lock()
	defer shape.Mu.Unlock()
	link, rule, failure = shape.chain_resolve_rule(link, references)
	if failure != 0 {
		return rule, false, failure
	}
	for _, extant := range shape.Links {
		if extant.Ordinal >= link.Ordinal {
			continue
		}
		if extant.Kind != DOT_ELEMENT_KIND_IMPOSSIBLE {
			continue
		}
		if extant.Message == link.Message {
			return rule, false, PRODUCT_FAILURE_DUPLICATE_RULE
		}
	}
	if int(link.Ordinal) < len(shape.Links) {
		expected := shape.Links[link.Ordinal]
		link.Rule_Index = expected.Rule_Index
		mismatch = !expected.chain_equal(link)
		if int(expected.Rule_Index) >= len(shape.Rules) {
			return rule, true, 0
		}
		return shape.Rules[expected.Rule_Index], mismatch, 0
	}
	if !shape.Ensured.Load() {
		if int(link.Ordinal) == len(shape.Links) {
			link.Rule_Index = uint8(len(shape.Rules))
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

// Registration already proved sibling uniqueness and compiled masks, so warmed replay compares
// only the exact source-level facts whose runtime values could diverge.
func (shape *Chain_Shape) chain_rule_replay(
	ordinal uint8, axis_count uint8, message string, references []Dot_Element_Reference,
) (rule Chain_Rule, mismatch bool, failure uint8) {
	if int(ordinal) >= len(shape.Links) {
		return rule, true, 0
	}
	link := &shape.Links[ordinal]
	if link.Kind != DOT_ELEMENT_KIND_IMPOSSIBLE {
		mismatch = true
	}
	if link.Axis_Count != axis_count {
		mismatch = true
	}
	if link.Message != message {
		if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
			return rule, false, PRODUCT_FAILURE_IMPOSSIBLE
		}
		mismatch = true
	}
	if int(link.Reference_Count) != len(references) {
		return rule, true, 0
	}
	for reference_index, reference := range references {
		axis_position := link.References[reference_index]
		if int(axis_position) >= len(shape.Axes) {
			return rule, true, 0
		}
		expected_message := shape.Axes[axis_position].Message
		if reference.Message != expected_message {
			if strings.Contains(reference.Message, ELEMENT_MESSAGE_SEPARATOR) {
				return rule, false, PRODUCT_FAILURE_REFERENCE
			}
			mismatch = true
		}
		expected_event := chain_mask_has(link.Reference_Events, uint8(reference_index))
		if reference.Event != expected_event {
			mismatch = true
		}
	}
	if int(link.Rule_Index) >= len(shape.Rules) {
		return rule, true, 0
	}
	rule = shape.Rules[link.Rule_Index]
	if rule.Message != link.Message {
		mismatch = true
	}
	return rule, mismatch, 0
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
	if first.Rule_Index != second.Rule_Index {
		return false
	}
	if first.Preset_Index != second.Preset_Index {
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
	if shape.chain_replays() {
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
		shape.Ensured.Store(true)
	}
	return matches
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

// A Handle_Entry is a resolved tracker slot: the seeded metadata and the tracker key, cached so
// Coverage_Sink can persist it without rebuilding the string.
type Handle_Entry struct {
	// Metadata is the seeded tracker entry, nil when registration seeded none.
	Metadata *Assertion_Metadata
	// Key is the tracker key, cached so Coverage_Sink can persist it without rebuilding it.
	Key string
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
			parsed := recorder_parse_directory(recorder.File_System, file_set, expanded)
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

// Parses the non-test .go files directly under the absolute Directory into AST
// files. File_System is rooted at "/", so the leading "/" is stripped to address
// it; the parsed file's name is the absolute path, used only for diagnostics now
// (identity is the message, not the position). Subdirectories are skipped — one
// directory is one package.
func recorder_parse_directory(
	file_system fs.FS, file_set *token.FileSet, directory string,
) (files []*ast.File) {
	root := strings.TrimPrefix(directory, "/")
	fs.WalkDir(file_system, root, func(
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
		SOURCE, read_error := fs.ReadFile(file_system, file_path)
		if read_error != nil {
			return nil
		}
		name := "/" + file_path
		file, parse_error := parser.ParseFile(
			file_set, name, SOURCE, parser.SkipObjectResolution,
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
	// Constants maps package constants to their value expressions. A flat bare-name index, like
	// Same_Set, is sufficient because one analysis covers one package tree.
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

// Reports every typed preset whose bound the evaluator could not resolve, then exits.
// A recognised-but-unevaluable bound leaves the grid unseeded while the runtime still enforces the
// range, so its coverage obligations would vanish unnoticed; failing keeps coverage from being
// silently dropped — the analyzer seeds a Range grid or refuses it.
func recorder_check_unresolved_bounds(recorder *Recorder, unresolved []string) {
	if len(unresolved) == 0 {
		return
	}
	banner := "🚨 " + strconv.Itoa(len(unresolved)) + " unresolved preset bounds 🚨"
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

// Registration_Axis is the analyzer's complete description of one tuple coordinate. Keeping the
// source condition beside registration-owned identity lets analysis explain a missing coordinate
// without making runtime retain source text.
type Registration_Axis struct {
	// Ordinal keeps repeated chain messages distinct.
	Ordinal uint8
	// Tuple_Position is registration's authoritative packed-mask position.
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

// Registration_Guard is one preset reachability link seeded outside tuple coordinates.
type Registration_Guard struct {
	// Ordinal keeps same-named guards from composed presets distinct.
	Ordinal uint8
	// Message is the fixed bound vocabulary.
	Message string
	// Condition is the bound expression shown for a gap.
	Condition string
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
	// Unresolved_Bound holds typed preset links whose bounds the constant evaluator could not
	// resolve, so their grid could not be seeded.
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

func ast_typed_preset_method(
	name string,
) (kind Chain_Preset_Kind, integer_kind Chain_Integer_Kind, matched bool) {
	prefix := ""
	if strings.HasPrefix(name, "Range_") {
		kind = CHAIN_PRESET_KIND_RANGE
		prefix = "Range_"
	}
	if strings.HasPrefix(name, "Enum_") {
		kind = CHAIN_PRESET_KIND_ENUM
		prefix = "Enum_"
	}
	if prefix == "" {
		return kind, integer_kind, false
	}
	integer_kind, matched = ast_chain_integer_kind(strings.TrimPrefix(name, prefix))
	return kind, integer_kind, matched
}

func ast_chain_integer_kind(name string) (kind Chain_Integer_Kind, matched bool) {
	switch name {
	case "Int":
		return CHAIN_INTEGER_KIND_INT, true
	case "Int8":
		return CHAIN_INTEGER_KIND_INT8, true
	case "Int16":
		return CHAIN_INTEGER_KIND_INT16, true
	case "Int32":
		return CHAIN_INTEGER_KIND_INT32, true
	case "Int64":
		return CHAIN_INTEGER_KIND_INT64, true
	case "Uint":
		return CHAIN_INTEGER_KIND_UINT, true
	case "Uint8":
		return CHAIN_INTEGER_KIND_UINT8, true
	case "Uint16":
		return CHAIN_INTEGER_KIND_UINT16, true
	case "Uint32":
		return CHAIN_INTEGER_KIND_UINT32, true
	case "Uint64":
		return CHAIN_INTEGER_KIND_UINT64, true
	}
	return kind, false
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
		if !ast_chain_link_method(method) {
			chain.Root = current
			break
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

func ast_chain_link_method(method string) (link bool) {
	switch method {
	case "Sometimes", "Impossible":
		return true
	}
	_, _, link = ast_typed_preset_method(method)
	return link
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
		recorder, file_set, ensure, namespace, chain.Links, index, reg, allow_unqualified)
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
			if !ast_chain_link_method(method) {
				break
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
	link_calls []*ast.CallExpr, index *Bundle_Index,
	reg *Registration, allow_unqualified bool,
) {
	if len(link_calls) > CHAIN_LINKS_MAX {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, position)+"  Dot_Product exceeds 255 links")
		return
	}
	axes, guards, carves, links, rules, presets, valid := recorder_collect_chain(
		file_set, link_calls, index, reg, allow_unqualified)
	if !valid {
		return
	}
	if len(axes) == 0 {
		if len(guards) == 0 {
			reg.Invalid_Chain = append(reg.Invalid_Chain,
				recorder_position(file_set, position)+"  Dot_Product has no axes")
			return
		}
	}
	recorder_seed_chain(
		recorder, file_set, position, namespace,
		axes, guards, carves, links, rules, presets, reg)
}

func recorder_collect_chain(
	file_set *token.FileSet, calls []*ast.CallExpr, index *Bundle_Index,
	reg *Registration, allow_unqualified bool,
) (
	axes []Registration_Axis, guards []Registration_Guard,
	carves [][]Registration_Cell, links []Chain_Link,
	rules []Chain_Rule, presets []Chain_Preset, valid bool,
) {
	valid = true
	positions := map[string][]int{}
	rule_messages := map[string]bool{}
	ordinal := 0
	for _, call := range calls {
		method := ast_chain_method(call)
		preset_kind, integer_kind, preset_method := ast_typed_preset_method(method)
		if preset_method {
			expansion, ok := recorder_collect_chain_preset(
				file_set, call, preset_kind, integer_kind,
				uint8(ordinal), uint8(len(axes)), uint8(len(rules)), index, reg)
			valid = valid && ok
			if !ok {
				continue
			}
			expansion.Links[0].Preset_Index = uint8(len(presets))
			guards = append(guards, recorder_registration_guards(expansion.Guards)...)
			preset_axes := recorder_registration_axes(expansion.Axes)
			for _, axis := range preset_axes {
				positions[axis.Message] = append(positions[axis.Message], len(axes))
			}
			axes = append(axes, preset_axes...)
			carves = append(carves, expansion.Carves...)
			links = append(links, expansion.Links...)
			rules = append(rules, expansion.Rules...)
			presets = append(presets, expansion.Preset)
			ordinal += int(expansion.Preset.Link_Count)
			continue
		}
		if ordinal >= CHAIN_LINKS_MAX {
			reg.Invalid_Chain = append(reg.Invalid_Chain,
				recorder_position(file_set, call)+"  Dot_Product exceeds 255 links")
			valid = false
			continue
		}
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
			ordinal++
			continue
		}
		cells, link, rule, ok := recorder_collect_chain_rule(
			file_set, call, uint8(ordinal), axes, positions, rule_messages,
			reg, allow_unqualified)
		valid = valid && ok
		if !ok {
			continue
		}
		link.Rule_Index = uint8(len(rules))
		carves = append(carves, cells)
		links = append(links, link)
		rules = append(rules, rule)
		ordinal++
	}
	return axes, guards, carves, links, rules, presets, valid
}

func recorder_registration_guards(guards []Chain_Guard) (registration []Registration_Guard) {
	for _, guard := range guards {
		registration = append(registration, Registration_Guard{
			Ordinal: guard.Ordinal, Message: guard.Message,
			Condition: recorder_preset_guard_condition(guard.Message),
		})
	}
	return registration
}

func recorder_registration_axes(axes []Chain_Preset_Axis) (registration []Registration_Axis) {
	for _, axis := range axes {
		registration = append(registration, Registration_Axis{
			Ordinal: axis.Ordinal, Tuple_Position: axis.Tuple_Position,
			Message: axis.Message, Condition: axis.Message,
			Kind: ASSERTION_KIND_SOMETIMES, Bucket_Count: 2,
		})
	}
	return registration
}

func recorder_collect_chain_preset(
	file_set *token.FileSet, call *ast.CallExpr,
	kind Chain_Preset_Kind, integer_kind Chain_Integer_Kind,
	ordinal uint8, axis_count uint8, rule_count uint8,
	index *Bundle_Index, reg *Registration,
) (expansion Chain_Preset_Expansion, valid bool) {
	minimum, maximum, values, resolved := recorder_collect_preset_domain(
		call, kind, integer_kind, index)
	if !resolved {
		recorder_preset_unresolved(file_set, call, reg)
		return expansion, false
	}
	if kind == CHAIN_PRESET_KIND_ENUM {
		if !chain_enum_domain_valid(values) {
			reg.Invalid_Chain = append(reg.Invalid_Chain,
				recorder_position(file_set, call)+
					"  Enum requires at least two distinct members")
			return expansion, false
		}
	}
	if kind == CHAIN_PRESET_KIND_RANGE {
		for _, hole := range values {
			if !chain_integer_between(minimum, hole, maximum) {
				reg.Invalid_Chain = append(reg.Invalid_Chain,
					recorder_position(file_set, call)+
						"  "+PRODUCT_FAILURE_RANGE_EXCLUSION_MESSAGE)
				return expansion, false
			}
		}
	}
	expansion = chain_preset_expand(Chain_Preset{
		Kind: kind, Integer_Kind: integer_kind,
		Ordinal: ordinal, Axis_Start: axis_count,
		Minimum: minimum, Maximum: maximum, Values: values,
	}, rule_count)
	if !expansion.Valid {
		reg.Invalid_Chain = append(reg.Invalid_Chain,
			recorder_position(file_set, call)+
				"  Dot_Product exceeds 255 expanded links")
		return expansion, false
	}
	return expansion, true
}

func recorder_collect_preset_domain(
	call *ast.CallExpr, kind Chain_Preset_Kind, integer_kind Chain_Integer_Kind,
	index *Bundle_Index,
) (
	minimum Chain_Integer_Value, maximum Chain_Integer_Value,
	values []Chain_Integer_Value, resolved bool,
) {
	if kind == CHAIN_PRESET_KIND_RANGE {
		if len(call.Args) < 3 {
			return minimum, maximum, values, false
		}
		minimum, resolved = recorder_eval_chain_integer(index, call.Args[1], integer_kind)
		if !resolved {
			return minimum, maximum, values, false
		}
		maximum, resolved = recorder_eval_chain_integer(index, call.Args[2], integer_kind)
		if !resolved {
			return minimum, maximum, values, false
		}
		values, resolved = recorder_eval_chain_integers(index, call.Args[3:], integer_kind)
		return minimum, maximum, values, resolved
	}
	values, resolved = recorder_eval_chain_integers(index, call.Args[1:], integer_kind)
	if !resolved {
		return minimum, maximum, values, false
	}
	minimum, maximum = chain_integer_span(values)
	return minimum, maximum, values, true
}

func recorder_eval_chain_integers(
	index *Bundle_Index, expressions []ast.Expr, integer_kind Chain_Integer_Kind,
) (values []Chain_Integer_Value, resolved bool) {
	for _, expression := range expressions {
		value, ok := recorder_eval_chain_integer(index, expression, integer_kind)
		if !ok {
			return nil, false
		}
		values = append(values, value)
	}
	return values, true
}

func recorder_eval_chain_integer(
	index *Bundle_Index, expression ast.Expr, integer_kind Chain_Integer_Kind,
) (integer Chain_Integer_Value, resolved bool) {
	value, ok := recorder_eval_constant(index, expression)
	if !ok {
		return integer, false
	}
	// Go constant division stays rational. The chain domain accepts only integers, so normalize
	// exact results before extraction; otherwise valid arithmetic is rejected and the integer
	// extractors panic.
	value = constant.ToInt(value)
	if value.Kind() != constant.Int {
		return integer, false
	}
	if constant.Sign(value) < 0 {
		if !chain_integer_kind_signed(integer_kind) {
			return integer, false
		}
		signed, exact := constant.Int64Val(value)
		if !exact {
			return integer, false
		}
		integer.Negative = true
		integer.Magnitude = uint64(-(signed + 1)) + 1
		return integer, true
	}
	magnitude, exact := constant.Uint64Val(value)
	if !exact {
		return integer, false
	}
	integer.Magnitude = magnitude
	return integer, true
}

func chain_integer_span(
	values []Chain_Integer_Value,
) (minimum Chain_Integer_Value, maximum Chain_Integer_Value) {
	if len(values) == 0 {
		return minimum, maximum
	}
	minimum, maximum = values[0], values[0]
	for _, value := range values {
		if chain_integer_compare(value, minimum) < 0 {
			minimum = value
		}
		if chain_integer_compare(value, maximum) > 0 {
			maximum = value
		}
	}
	return minimum, maximum
}

func recorder_preset_guard_condition(message string) (condition string) {
	if message == RANGE_GUARD_UPPER {
		return "value <= max"
	}
	return "value >= min"
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
		Mask: mask, Want: want, Message: message}
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
	axes []Registration_Axis, guards []Registration_Guard,
	carves [][]Registration_Cell, links []Chain_Link,
	rules []Chain_Rule, presets []Chain_Preset, reg *Registration,
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
		Guards: make([]Chain_Guard, len(guards)), Presets: presets,
		Tuples: map[Chain_Mask]Handle_Entry{}, Registered: true,
	}
	shape.Ensured.Store(true)
	if recorder.Chain_Entries == nil {
		recorder.Chain_Entries = map[string]*Assertion_Metadata{}
	}
	for guard_index, guard := range guards {
		text := chain_axis_key(Namespace(namespace), guard.Ordinal, guard.Message)
		metadata := &Assertion_Metadata{
			Kind: ASSERTION_KIND_ALWAYS, Message: text, Condition: guard.Condition}
		recorder.Events.Store(text, metadata)
		recorder.Chain_Entries[text] = metadata
		shape.Guards[guard_index] = Chain_Guard{
			Ordinal: guard.Ordinal, Message: guard.Message,
			Entry: Handle_Entry{Metadata: metadata, Key: text},
		}
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
	recorder_chain_shape_publish(recorder, Namespace(namespace), shape)
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
				chain.Links, index, reg, function.Is_Sugar)
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

// Records a typed preset callsite whose bound the evaluator could not resolve, so registration
// fails rather than seeding an incomplete grid the runtime would still enforce.
func recorder_preset_unresolved(
	file_set *token.FileSet, position ast.Node, reg *Registration,
) {
	reg.Unresolved_Bound = append(reg.Unresolved_Bound,
		recorder_position(file_set, position)+
			"  typed preset bound is not a resolvable constant")
}

// Constant_Eval_Frame is one node of the constant-evaluation work stack. Expanded distinguishes
// the first visit that schedules operands from the second visit that combines their values.
type Constant_Eval_Frame struct {
	// Expression is the AST node this frame evaluates.
	Expression ast.Expr
	// Expanded reports whether this operator already pushed its operands for evaluation.
	Expanded bool
}

// Evaluates a typed preset bound expression to an integer constant: an integer literal, a
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
	work := []Constant_Eval_Frame{{Expression: expression}}
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
			work = append(work, Constant_Eval_Frame{Expression: reference})
		case *ast.ParenExpr:
			work = append(work, Constant_Eval_Frame{Expression: node.X})
		case *ast.CallExpr:
			if !ast_integer_conversion(node) {
				return nil, false
			}
			work = append(work, Constant_Eval_Frame{Expression: node.Args[0]})
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

func ast_integer_conversion(call *ast.CallExpr) (conversion bool) {
	if len(call.Args) != 1 {
		return false
	}
	identifier, is_identifier := call.Fun.(*ast.Ident)
	if !is_identifier {
		return false
	}
	switch identifier.Name {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
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
	work []Constant_Eval_Frame, operator ast.Expr, operands ...ast.Expr,
) (expanded []Constant_Eval_Frame) {
	work = append(work, Constant_Eval_Frame{Expression: operator, Expanded: true})
	for i := len(operands) - 1; i >= 0; i-- {
		work = append(work, Constant_Eval_Frame{Expression: operands[i]})
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
		integer_count := constant.ToInt(right)
		if integer_count.Kind() != constant.Int {
			return nil, false
		}
		count, exact := constant.Uint64Val(integer_count)
		if !exact {
			return nil, false
		}
		result := constant.Shift(left, op, uint(count))
		return result, result.Kind() != constant.Unknown
	}
	if recorder_constant_divides(op) {
		left_integer, left_ok := recorder_constant_big_integer(left)
		right_integer, right_ok := recorder_constant_big_integer(right)
		if !left_ok {
			return nil, false
		}
		if !right_ok {
			return nil, false
		}
		if right_integer.Sign() == 0 {
			return nil, false
		}
		result := new(big.Int)
		if op == token.QUO {
			result.Quo(left_integer, right_integer)
		} else {
			result.Rem(left_integer, right_integer)
		}
		return constant.Make(result), true
	}
	if !recorder_constant_is_arithmetic(op) {
		return nil, false
	}
	result := constant.BinaryOp(left, op, right)
	return result, result.Kind() != constant.Unknown
}

// Extracting the integer before quotient or remainder preserves Go's truncation semantics while
// retaining arbitrary precision for bounds whose intermediate products exceed uint64.
func recorder_constant_big_integer(value constant.Value) (integer *big.Int, ok bool) {
	value = constant.ToInt(value)
	if value.Kind() != constant.Int {
		return nil, false
	}
	switch concrete := constant.Val(value).(type) {
	case int64:
		return big.NewInt(concrete), true
	case *big.Int:
		return new(big.Int).Set(concrete), true
	}
	return nil, false
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

// Maps each package-level const's name to its value expression, for evaluating typed preset bounds.
// Only a spec that supplies its own value is indexed; an inherited-value spec (no iota in
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
	files := recorder_parse_directory(index.File_System, index.File_Set, directory)
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
	recorder_report_section(
		recorder.Output, "Branch gaps", gaps, ASSERTION_KIND_SOMETIMES)
	recorder_report_section(
		recorder.Output, "Reachability gaps", gaps, ASSERTION_KIND_ALWAYS)
	fmt.Fprintln(recorder.Output, banner)
}

// Prints, under a markdown heading, the gaps whose assertion is of the given
// kind, sorted by message. Emits nothing when no gap matches, so empty sections
// stay silent.
func recorder_report_section(
	output io.Writer, title string, gaps []Coverage_Gap, kind Assertion_Kind,
) {
	selected := make([]Coverage_Gap, 0, len(gaps))
	for _, gap := range gaps {
		if gap.Metadata.Kind == kind {
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
	fmt.Fprintln(output)
	fmt.Fprintln(output, "# "+title)
	for _, gap := range selected {
		fmt.Fprintln(output, coverage_gap_line(gap))
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
