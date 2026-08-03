// Package invariant exposes eager Always guards and deferred assertion builders. Registration
// owns coverage identity; Ensure owns builder enforcement and individual branch emission.
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
	"math/big"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"unsafe"
)

// ASSERTION_FAILURE_MESSAGE_PREFIX opens every assertion-failure message.
const ASSERTION_FAILURE_MESSAGE_PREFIX = "🚨 Assertion Failure 🚨: "

// Assertion_Failure delays production diagnostics until a failed value is rendered. Explicit
// fields let recovery inspect the failure without parsing its human-readable text.
type Assertion_Failure struct {
	// Identity remains separate so the fixed property is recognizable without formatting.
	Identity string
	// Reason keeps the runtime relation out of the property's stable identity.
	Reason string
	// Value stays unformatted until panic reporting is already unavoidable.
	Value any
}

// Error keeps text construction out of every successful production assertion.
func (failure Assertion_Failure) Error() (message string) {
	return ASSERTION_FAILURE_MESSAGE_PREFIX + failure.Identity + failure.Reason +
		": " + fmt.Sprint(failure.Value)
}

// ELEMENT_MESSAGE_SEPARATOR makes chain coverage keys unambiguous because registration
// rejects it in every message.
const ELEMENT_MESSAGE_SEPARATOR = "\x00"

// ASSERTION_LINKS_MAX is how many expanded links one chain holds, set by the seven-bit ordinal
// counter the builder packs. The bound is part of the source contract, so registration and the
// runtime share one number rather than parallel limits.
const ASSERTION_LINKS_MAX = 127

// ASSERTION_OBSERVATIONS_MAX is how many axes one chain records, set by the observation bits the
// two state words hold after their counters, failure code, and union tag. A guard holds one outcome
// and spends no bit, thus only an axis reaches this. Registration enforces it under every build,
// which is why it sits beside the link bound rather than in the enforcement layout.
const ASSERTION_OBSERVATIONS_MAX = 109

// RANGE_ENUM_CARDINALITY_MAX keeps small registered domains on the fixed-capacity helper that
// states every legal value instead of making Range infer the same finite set.
const RANGE_ENUM_CARDINALITY_MAX = 4

// The boundary witnesses a Range seeds per root. One pair of texts serves static seeding and
// runtime crediting, so both sides rendezvous on identical keys.
const RANGE_MESSAGE_MINIMUM = "The value equals the minimum."

// RANGE_MESSAGE_MAXIMUM is separate because distinct boundaries own distinct branch obligations.
const RANGE_MESSAGE_MAXIMUM = "The value equals the maximum."

// RANGE_GUARD_MINIMUM keeps successful lower-bound reachability visible apart from boundary axes.
const RANGE_GUARD_MINIMUM = "The value is at least its minimum."

// RANGE_GUARD_MAXIMUM keeps successful upper-bound reachability visible apart from boundary axes.
const RANGE_GUARD_MAXIMUM = "The value is at most its maximum."

// ENUM_GUARD_MEMBER makes successful membership a reachability obligation of its own.
const ENUM_GUARD_MEMBER = "The value is an enum member."

// RANGE_MESSAGE_ZERO names the conventional zero witness when it is strictly interior.
const RANGE_MESSAGE_ZERO = "The value is zero."

// RANGE_MESSAGE_ONE names the conventional one witness when it is strictly interior.
const RANGE_MESSAGE_ONE = "The value is one."

// RANGE_MESSAGE_TWO names the conventional two witness when it is strictly interior.
const RANGE_MESSAGE_TWO = "The value is two."

// RANGE_MESSAGE_NEGATIVE_ONE exists only for a signed interval that contains it strictly.
const RANGE_MESSAGE_NEGATIVE_ONE = "The value is negative one."

// Static member identities stay independent of runtime formatting.
func enum_member_message(member_text string) (message string) {
	return "The value equals member " + member_text + "."
}

// Bounds helper expansion so pathological composition cannot make registration consume
// unbounded work; the explicit worklist refuses a forwarding cycle long before this.
const BUNDLE_EXPANSION_STEPS_MAX = 4096

// Bounds the walk up the directory tree searching for a go.mod, so module
// discovery can't loop unboundedly on a pathological path.
const MODULE_SEARCH_DEPTH_MAX = 256

// ASSERTION_KIND_ALWAYS classifies a one-outcome tracker entry for an Always.
const ASSERTION_KIND_ALWAYS Assertion_Kind = 0

// ASSERTION_KIND_SOMETIMES classifies a two-outcome tracker entry for a Sometimes.
const ASSERTION_KIND_SOMETIMES Assertion_Kind = 1

// Bounds the resolver's total work so a constant-reference cycle (`const A = B; const B = A`)
// cannot loop it; a legal bound expression resolves in far fewer steps.
const CONSTANT_RESOLUTION_STEPS_MAX = 4096

// Recorder accumulates assertion observations for one run under registration-owned identities.
type Recorder struct {
	// File_System reads Go source files during AST analysis. Paths are absolute OS paths;
	// lookups strip the leading "/" before calling fs.ReadFile.
	File_System fs.FS

	// Events is the coverage tracker: one entry per registered assertion,
	// keyed by message and credited as observations arrive.
	Events sync.Map
	// Forbidden_Properties counts distinct Range holes because observation panics, so no Events
	// entry exists for the clean-run summary to discover.
	Forbidden_Properties int

	// Assertion_Plans is published once before the suite and read thereafter. Ordinary binaries
	// never populate or consult it; only a recording root pays the key lookup.
	Assertion_Plans map[Plan_Key]*Assertion_Plan

	// Output receives the coverage-gap report and the orphan/bundle diagnostics.
	Output io.Writer
	// Exit ends the process with a status code; the composition tier wires it to os.Exit.
	Exit func(code int)
	// Tty receives the clean-run success summary so it shows even without `go test -v`.
	Tty io.Writer
	// Report_Coverage_Gaps renders the already-sorted flat gap records. Nil selects the
	// Markdown table so pure callers retain the human default without composition wiring.
	Report_Coverage_Gaps Coverage_Gap_Reporter
	// Output_Configuration_Diagnostic prevents a misconfigured report mode from running a
	// suite whose final result could not honor the requested output contract.
	Output_Configuration_Diagnostic string

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

	// Sugar_Package identifies the one package whose unqualified writer calls are the public
	// default surface rather than unrelated local functions.
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
	// coverage and credit it into registered entries so analysis sees what workers found.
	Merge_Fuzz_Coverage func()

	// On_Fatal receives the complete assertion message immediately before an enforcing
	// assertion panics. A composition root can drain an asynchronous log egress before exit.
	On_Fatal func(message string)
}

// Calls the optional composition hook before the assertion site causes the panic.
func recorder_fatal_hook(recorder *Recorder, message string) {
	if recorder == nil {
		return
	}
	if recorder.On_Fatal != nil {
		recorder.On_Fatal(message)
	}
}

// Signed spans the primitive signed integer widths.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Unsigned spans the primitive unsigned integer widths. uintptr is excluded — it carries an
// address, not a quantity, so numeric bounds on it assert nothing about the domain.
type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Float spans both floating-point widths.
type Float interface {
	~float32 | ~float64
}

// Integer spans the primitive integer widths the bare guards accept.
type Integer interface {
	Signed | Unsigned
}

// Namespace names one ensured assertion root without making each message globally unique.
type Namespace string

// Plan_Key names one chain: the root namespace plus the subject type that stands in for the call
// path from that root. A bundle forwards its namespace unchanged, so the namespace alone cannot
// tell two sibling chains apart. The subject type can, because registration rejects a type that
// occurs twice under one root.
//
// The two components stay separate strings rather than one joined string. reflect returns both as
// substrings of linker data, so a comparable struct key resolves the plan with no concatenation and
// no allocation.
type Plan_Key struct {
	// Namespace is the literal the root callsite wrote.
	Namespace Namespace
	// Package is the subject type's import path, from reflect.Type.PkgPath.
	Package string
	// Type is the subject type's name, from reflect.Type.Name.
	Type string
}

// Assertion_Builder is the complete deferred runtime state. Fixed observations make escape and
// allocation unnecessary even at the largest supported chain.
type Assertion_Builder struct {
	// Context holds ordinary namespace bytes or a recording plan. State_A and State_B interpret
	// that pointer as a tagged union: namespace length plus failure for enforcement, or packed
	// observations, ordinal, and failure for recording. The states are mutually exclusive, so
	// carrying both representations through every fluent return would be pure overhead.
	Context unsafe.Pointer
	// State_A holds the namespace length or the first 64 observations selected by Context.
	State_A uintptr
	// State_B holds the union tag, deferred failure, ordinal, and remaining observations.
	State_B uintptr
}

// Assertion_Plan is registration's immutable emission program for one chain.
type Assertion_Plan struct {
	// Recorder owns the plan, so recording builders do not carry a second pointer.
	Recorder *Recorder
	// Identity restores the deferred failure prefix when Context_Data carries the plan. A chain
	// stores its namespace and an inline helper stores its message, which is the whole identity
	// each one has. The package and type of a chain never reach a failure message, thus the
	// plan does not keep them.
	Identity Namespace
	// Links is the exact expanded sequence runtime observations index.
	Links []Assertion_Plan_Link
}

// Assertion_Plan_Link carries the already-resolved identity that Ensure may credit.
type Assertion_Plan_Link struct {
	// Ordinal is the plan position, which every link holds.
	Ordinal uint8
	// Observation is the packed bit position, which only an axis holds. A guard spends no bit,
	// thus this trails Ordinal by the number of guards before it and registration computes it
	// one time rather than the chain deriving it.
	Observation uint8
	// Kind distinguishes one-outcome guards from two-outcome axes.
	Kind Assertion_Kind
	// Entry owns the identity registration resolved before the suite.
	Entry Handle_Entry
}

// Assertion_Kind discriminates a coverage tracker entry as Always or Sometimes.
type Assertion_Kind uint8

// Assertion_Metadata is one coverage tracker entry: which outcomes an assertion observed
// across the run. Seeded at registration, incremented at runtime, scanned by the never-fired
// report.
type Assertion_Metadata struct {
	// Frequency counts true-event observations: an Always or a Sometimes true.
	Frequency atomic.Int64
	// False_Frequency counts false-event observations: a Sometimes false.
	False_Frequency atomic.Int64
	// Kind discriminates the entry: Always or Sometimes.
	Kind Assertion_Kind
	// Message is the complete registration-owned identity used as the tracker key.
	Message string
	// Condition is the source text of the asserted expression, for the gap report.
	Condition string
	// Domain is the Range interval this entry guards, or nil for every other entry. Only a
	// Range's first guard carries one, thus one Range reports one domain row.
	Domain *Assertion_Domain
}

// Assertion_Domain is one Range's declared interval beside the values a run put through it. A gap
// names the branch that never fired. This names the values that did, thus a bound that no real
// value approaches is visible without a second run.
type Assertion_Domain struct {
	// Declared_Minimum is the interval's lower bound as registration read it.
	Declared_Minimum Integer_Value
	// Declared_Maximum is the interval's upper bound as registration read it.
	Declared_Maximum Integer_Value
	// Unsigned selects the key encoding below. A Uint64 bound exceeds the signed range, thus an
	// int64 observation field would narrow it.
	Unsigned bool
	// Observed_Minimum_Key is the smallest observed value, encoded so that the unsigned compare
	// an atomic offers matches the value's own order. It starts at the largest key.
	Observed_Minimum_Key atomic.Uint64
	// Observed_Maximum_Key is the largest observed value, in that same encoding.
	Observed_Maximum_Key atomic.Uint64
	// Observation_Count is how many values reached this Range. Zero means the run never did,
	// and the two keys then hold their seeded extremes rather than a reading.
	Observation_Count atomic.Int64
}

// ASSERTION_DOMAIN_RETRY_MAX bounds one bound's compare-and-swap. A retry means another goroutine
// moved that bound, and a move that still loses to this value is rare, thus this sits far above
// what real contention reaches. Exhausting it drops one bound update and never the count.
const ASSERTION_DOMAIN_RETRY_MAX = 64

// ASSERTION_DOMAIN_SIGN_KEY flips a signed value's sign bit, which maps the signed order onto the
// unsigned order an atomic compares. An unsigned domain needs no flip.
const ASSERTION_DOMAIN_SIGN_KEY = uint64(1) << 63

// Encodes one value so that unsigned comparison of two keys matches the values' own order.
func assertion_domain_key(value uint64, unsigned bool) (key uint64) {
	if unsigned {
		return value
	}
	return value ^ ASSERTION_DOMAIN_SIGN_KEY
}

// Reverses assertion_domain_key into the representation the report prints.
func assertion_domain_value(key uint64, unsigned bool) (value Integer_Value) {
	if unsigned {
		return Integer_Value{Magnitude: key}
	}
	signed := int64(key ^ ASSERTION_DOMAIN_SIGN_KEY)
	if signed < 0 {
		return Integer_Value{Magnitude: uint64(-signed), Negative: true}
	}
	return Integer_Value{Magnitude: uint64(signed)}
}

// Records one value against a Range's declared interval. The two bounds start at the extremes their
// comparison loses to, thus a plain compare-and-swap loop stays correct when several goroutines
// drive one plan and no first-write flag is needed.
func assertion_domain_observe(domain *Assertion_Domain, value uint64) {
	key := assertion_domain_key(value, domain.Unsigned)
	for attempt_index := 0; attempt_index < ASSERTION_DOMAIN_RETRY_MAX; attempt_index++ {
		smallest := domain.Observed_Minimum_Key.Load()
		if smallest <= key {
			break
		}
		if domain.Observed_Minimum_Key.CompareAndSwap(smallest, key) {
			break
		}
	}
	for attempt_index := 0; attempt_index < ASSERTION_DOMAIN_RETRY_MAX; attempt_index++ {
		largest := domain.Observed_Maximum_Key.Load()
		if largest >= key {
			break
		}
		if domain.Observed_Maximum_Key.CompareAndSwap(largest, key) {
			break
		}
	}
	domain.Observation_Count.Add(1)
}

// Handle_Entry is a resolved tracker slot: the seeded metadata and the tracker key, cached so
// Coverage_Sink can persist it without rebuilding the string.
type Handle_Entry struct {
	// Metadata is the seeded tracker entry, nil when registration seeded none.
	Metadata *Assertion_Metadata
	// Key is the tracker key, cached so Coverage_Sink can persist it without rebuilding it.
	Key string
}

// Bumps entry's metadata: Frequency on a true event, False_Frequency on false. A nil-metadata
// entry is skipped. On the 0→1 transition of a branch it fires Coverage_Sink with the entry's
// cached key, so a fuzz worker persists the branch; atomic Add returns the post-increment value,
// so the sink fires once
// per branch.
func recorder_increment_entry(recorder *Recorder, entry Handle_Entry, fired_true bool) {
	if entry.Metadata == nil {
		return
	}
	// Seen-ness is the entire signal — the gap report reads zero versus nonzero — so after
	// the first hit an add would only ping-pong the entry's cache line across workers. The
	// plain load may race another first hit; the add below still elects exactly one sink call.
	if fired_true {
		if entry.Metadata.Frequency.Load() != 0 {
			return
		}
		if entry.Metadata.Frequency.Add(1) == 1 {
			if recorder.Coverage_Sink != nil {
				recorder.Coverage_Sink(entry.Key, true)
			}
		}
		return
	}
	if entry.Metadata.False_Frequency.Load() != 0 {
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
// the NUL chain separator and otherwise-arbitrary bytes; the trailing newline makes the file
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
	branch := line[tab_offset+1:]
	if branch == "T" {
		recorder_merge_increment(recorder, string(key), true)
		return
	}
	if branch != "F" {
		return
	}
	recorder_merge_increment(recorder, string(key), false)
}

// Merge resolves the emitted string key directly; a key with no seeded entry is skipped like any
// runtime increment.
func recorder_merge_increment(recorder *Recorder, key string, fired_true bool) {
	recorder_increment(recorder, key, fired_true)
}

// Recorder_Merge_Fuzz_Coverage_From unions the coverage a fuzz coordinator reads from r
// (the workers' shared file, one Fuzz_Coverage_Line per line) into registered entries: each
// covered (key, branch) marks that branch non-zero. Binary — per-process counts are not summed
// across workers. A malformed or partial trailing line is skipped (a worker killed mid-write
// costs at most its last line); a key with no seeded entry is skipped, like any runtime increment.
func Recorder_Merge_Fuzz_Coverage_From(recorder *Recorder, r io.Reader) {
	var buffer [MERGE_READ_BYTES]byte
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

// Recorder_Register_Packages_For_Analysis parses non-test Go files, seeds individual
// obligations, and publishes the registration-owned emission plan before the suite starts.
//
// Directories default to recorder.Packages_To_Analyze when none are passed; a
// directory may glob, a `*` segment matching one path element and `**` any depth,
// expanded against File_System. Always uses its literal message directly; a builder assertion
// uses its literal namespace, expanded ordinal, and literal or preset-owned message.
func Recorder_Register_Packages_For_Analysis(recorder *Recorder, directories ...string) {
	if len(directories) > 0 {
		recorder.Packages_To_Analyze = directories
	}
	file_set := token.NewFileSet()
	var files []*ast.File
	var test_files []*ast.File
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
			parsed, parsed_tests := recorder_parse_directory(
				recorder.File_System, file_set, expanded)
			files = append(files, parsed...)
			test_files = append(test_files, parsed_tests...)
		}
	}
	// One walk each. Bundle_Index and every Indexed_Function share these two indexes, so
	// building them at both sites would walk every declaration of every file a second time.
	constants := ast_index_constants(files)
	package_types := ast_index_package_types(files)
	index := &Bundle_Index{
		File_System:   recorder.File_System,
		File_Set:      file_set,
		Module_Path:   module_path,
		Module_Root:   module_root,
		Sugar_Package: recorder.Sugar_Package,
		Same_Set: ast_index_functions(
			files, file_set, module_path, module_root, constants, package_types),
		Loaded:           map[string]map[string]Indexed_Function{},
		Loaded_Constants: map[string]map[string]ast.Expr{},
		Constants:        constants,
		Package_Types:    package_types,
	}
	reg := &Registration{
		Planned_Keys:         map[string]bool{},
		Message_Owner:        map[string]bool{},
		Registered_Calls:     map[token.Pos]bool{},
		Namespace_Owner_Path: map[Plan_Key][]token.Pos{},
		Subject_Owner:        map[Plan_Key][]token.Pos{},
		Planned_Inline:       map[string]*Assertion_Plan{},
		Root_Namespace_Owner: map[Namespace]token.Pos{},
		Planned_Assertions:   map[Plan_Key]*Assertion_Plan{},
	}
	recorder_register_assertion_files(recorder, file_set, files, index, reg)
	recorder_check_registration(recorder, file_set, files, test_files, index, reg)
	recorder_commit_planned(recorder, reg)
}

// Runs every fatal registration check in one place. Each reporter is fatal on its own, and the
// commit runs only when none fired, so the published plan stays all-or-nothing.
func recorder_check_registration(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File,
	test_files []*ast.File, index *Bundle_Index, reg *Registration,
) {
	recorder_check_test_assertion_calls(recorder, file_set, test_files, reg)
	recorder_check_bundle_control_flow(recorder, file_set, files, reg)
	recorder_check_bundle_literal_namespaces(recorder, file_set, files, reg)
	recorder_check_duplicate_bundle_namespaces(recorder, file_set, files, reg)
	recorder_check_assertion_bundle_contract(recorder, file_set, files, index, reg)
	recorder_check_unresolved(recorder, reg)
	recorder_check_bundle_cycles(recorder, reg)
	recorder_check_invalid_subjects(recorder, reg)
	recorder_check_repeated_subjects(recorder, reg)
	recorder_check_invalid_identifiers(recorder, reg)
	recorder_check_non_literal_messages(recorder, reg)
	recorder_check_constant_always_conditions(recorder, reg)
	recorder_check_duplicate_messages(recorder, reg)
	recorder_check_unresolved_bounds(recorder, reg)
	recorder_check_invalid_bounds(recorder, reg)
	recorder_report_registration_failure(
		recorder, reg, "invalid assertion chains", reg.Invalid_Chain)
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

// Separates production and test source because tests can validate registration but cannot add
// obligations or submit coverage directly.
func recorder_parse_directory(
	file_system fs.FS, file_set *token.FileSet, directory string,
) (files []*ast.File, test_files []*ast.File) {
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
		SOURCE, read_error := fs.ReadFile(file_system, file_path)
		if read_error != nil {
			return nil
		}
		name := "/" + file_path
		file, parse_error := parser.ParseFile(
			file_set, name, SOURCE, parser.SkipObjectResolution,
		)
		if parse_error == nil {
			if strings.HasSuffix(file_path, "_test.go") {
				test_files = append(test_files, file)
			} else {
				files = append(files, file)
			}
		}
		return nil
	})
	return files, test_files
}

// Test code must reach an assertion through production code. A direct writer can impersonate a
// registered namespace or message and satisfy an obligation without the production callsite.
func recorder_check_test_assertion_calls(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File, reg *Registration,
) {
	if len(reg.Planned) == 0 {
		return
	}
	var violations []string
	for _, file := range files {
		direct_callees := ast_test_assertion_direct_callees(file)
		ast.Inspect(file, func(node ast.Node) (descend bool) {
			switch concrete := node.(type) {
			case *ast.CallExpr:
				name := ast_callee_name(concrete)
				if ast_test_assertion_writer(name) {
					position := recorder_position(file_set, concrete)
					violation := position + "  test source calls " + name
					violations = append(violations, violation)
				}
			case *ast.SelectorExpr:
				if direct_callees[concrete.Pos()] {
					return true
				}
				name := concrete.Sel.Name
				if ast_test_assertion_writer(name) {
					position := recorder_position(file_set, concrete)
					violation := position + "  test source references " + name
					violations = append(violations, violation)
				}
			}
			return true
		})
	}
	recorder_report_registration_failure(
		recorder, reg, "test assertion callsites", violations)
}

func ast_test_assertion_writer(name string) (writer bool) {
	switch name {
	case "Always", "Recorder_Always", "Tree", "Recorder_Tree":
		return true
	}
	return ast_is_invariants_name(name)
}

// Direct call selectors already have a stronger diagnostic. Other selector references can move
// the writer through a function value and evade a call-name scan.
func ast_test_assertion_direct_callees(file *ast.File) (positions map[token.Pos]bool) {
	positions = map[token.Pos]bool{}
	ast.Inspect(file, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if is_call {
			if ast_test_assertion_writer(ast_callee_name(call)) {
				positions[call.Fun.Pos()] = true
			}
		}
		return true
	})
	return positions
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
// import-path map of the file it lives in, so the bundle's own qualified
// sub-calls resolve.
type Indexed_Function struct {
	// Declaration is the discovered function declaration.
	Declaration *ast.FuncDecl
	// Package is the declaring import path. Two packages can declare one bundle name, so bare
	// names cannot identify a bundle, and cannot detect a cycle either.
	Package string
	// Imports maps the file's local names to import paths, so qualified sub-calls resolve.
	Imports map[string]string
	// Constants is the declaration package's static integer namespace.
	Constants map[string]ast.Expr
	// Package_Types maps the declaration package's package-level types to their underlying
	// expressions, so a function-local subject is rejected and a Boolean subject is known.
	Package_Types map[string]ast.Expr
	// Package_Functions keeps bare nested calls anchored to the declaration's package after a
	// cross-package descent; the analyzer root's Same_Set belongs to the caller instead.
	Package_Functions map[string]Indexed_Function
	// Is_Sugar permits the one package's unqualified public writer calls.
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
	// Sugar_Package is the sole package where unqualified Assertions and Always are writers.
	Sugar_Package string
	// Same_Set holds the analyzed packages' functions by bare name (same-package bundles).
	Same_Set map[string]Indexed_Function
	// Loaded caches lazily parsed cross-package functions, keyed by import path then bare name.
	Loaded map[string]map[string]Indexed_Function
	// Loaded_Constants caches those same packages' constants, keyed the same way. A qualified
	// operand names one package, thus a flat bare-name index cannot answer it.
	Loaded_Constants map[string]map[string]ast.Expr
	// Constants maps package constants to their value expressions. A flat bare-name index, like
	// Same_Set, is sufficient because one analysis covers one package tree.
	Constants map[string]ast.Expr
	// Package_Types maps each package-level type declaration to its underlying expression.
	// reflect reports one empty PkgPath and one shared Name for two function-local types that
	// share a name, thus presence here is what lets a chain subject carry an identity. The
	// expression is what tells a Boolean subject from any other.
	Package_Types map[string]ast.Expr
}

// Names each package-level type declaration. A type declared inside a function body is absent,
// which is what lets registration reject it as a chain subject.
func ast_index_package_types(files []*ast.File) (types map[string]ast.Expr) {
	types = map[string]ast.Expr{}
	for _, file := range files {
		for _, declaration := range file.Decls {
			general, is_general := declaration.(*ast.GenDecl)
			if !is_general {
				continue
			}
			if general.Tok != token.TYPE {
				continue
			}
			for _, specification := range general.Specs {
				type_specification, is_type := specification.(*ast.TypeSpec)
				if !is_type {
					continue
				}
				types[type_specification.Name.Name] = type_specification.Type
			}
		}
	}
	return types
}

// Maps each function name to its declaration, its declaring package, and its file's imports, for
// descending *_Invariants bundles. A later definition wins on a name collision.
func ast_index_functions(
	files []*ast.File, file_set *token.FileSet, module_path string, module_root string,
	constants map[string]ast.Expr, package_types map[string]ast.Expr,
) (functions map[string]Indexed_Function) {
	functions = map[string]Indexed_Function{}
	for _, file := range files {
		imports := ast_file_imports(file)
		package_path := ast_file_package_path(file, file_set, module_path, module_root)
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			functions[function.Name.Name] = Indexed_Function{
				Declaration:   function,
				Package:       package_path,
				Imports:       imports,
				Constants:     constants,
				Package_Types: package_types,
			}
		}
	}
	for name, function := range functions {
		function.Package_Functions = functions
		functions[name] = function
	}
	return functions
}

// The one gate into the plan: a key already planned is a fatal collision — two assertions
// sharing a key would silently merge and mask a gap.
// Binds one plan to its recorder and resolves every link's seeded tracker entry.
func recorder_publish_plan(recorder *Recorder, key Plan_Key, plan *Assertion_Plan) {
	plan.Recorder = recorder
	plan.Identity = key.Namespace
	for link_index := range plan.Links {
		link := &plan.Links[link_index]
		value, exists := recorder.Events.Load(link.Entry.Key)
		if !exists {
			continue
		}
		link.Entry.Metadata = value.(*Assertion_Metadata)
	}
	recorder.Assertion_Plans[key] = plan
}

func recorder_plan_seed(
	file_set *token.FileSet, node ast.Node, reg *Registration, key string,
	kind Assertion_Kind, condition string, domain *Assertion_Domain,
) {
	if reg.Planned_Keys[key] {
		reg.Collision = append(reg.Collision,
			recorder_position(file_set, node)+
				"  duplicate message: "+strconv.Quote(key))
		return
	}
	reg.Planned_Keys[key] = true
	reg.Planned = append(reg.Planned,
		Planned_Seed{Key: key, Kind: kind, Condition: condition, Domain: domain})
}

// The all-or-nothing commit: a failed registration leaves Events empty, so a partially seeded
// tracker can never masquerade as a registered run. Metadata.Message holds the full key so the
// gap report, the fuzz merge, and Coverage_Sink all rendezvous on the same string.
func recorder_commit_planned(recorder *Recorder, reg *Registration) {
	if reg.Failed {
		return
	}
	for _, seed := range reg.Planned {
		// The smallest observation loses to the seeded key and the largest wins over it,
		// thus the first value a run puts through the Range replaces both.
		if seed.Domain != nil {
			seed.Domain.Observed_Minimum_Key.Store(^uint64(0))
		}
		recorder.Events.Store(seed.Key, &Assertion_Metadata{
			Kind:      seed.Kind,
			Message:   seed.Key,
			Condition: seed.Condition,
			Domain:    seed.Domain,
		})
	}
	// One map. A chain key always carries a defined type, because a bundle is named for its
	// subject and registration rejects any other subject. An inline key carries only its
	// message, thus the two key spaces cannot meet.
	recorder.Assertion_Plans = map[Plan_Key]*Assertion_Plan{}
	for key, plan := range reg.Planned_Assertions {
		recorder_publish_plan(recorder, key, plan)
	}
	for message, plan := range reg.Planned_Inline {
		recorder_publish_plan(recorder, Plan_Key{Namespace: Namespace(message)}, plan)
	}
	recorder.Forbidden_Properties = reg.Forbidden_Properties
}

func ast_argument(call *ast.CallExpr, index int) (argument ast.Expr) {
	if len(call.Args) <= index {
		return nil
	}
	return call.Args[index]
}

// Indexes the analyzed files' single-name, explicit-value constant declarations by bare name.
// Multi-name and implicit (iota-carrying) specs are deliberately absent: their values depend on
// positional context the flat index cannot carry, so a reference to one is unresolvable and
// fails registration rather than resolving wrongly.
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
				if len(value_specification.Names) != 1 {
					continue
				}
				if len(value_specification.Values) != 1 {
					continue
				}
				name := value_specification.Names[0].Name
				constants[name] = value_specification.Values[0]
			}
		}
	}
	return constants
}

// Integer_Value is exact across the negative int64 minimum and positive uint64 maximum.
type Integer_Value struct {
	// Magnitude stores the absolute value without narrowing through int64.
	Magnitude uint64
	// Negative distinguishes negative values while canonical zero remains nonnegative.
	Negative bool
}

// Constant_Scope resolves one constant operand. A bare name comes from the declaring package. A
// qualified name states its package in the source, thus one shared fact can serve every package
// that names it while a bare name still cannot reach across a boundary.
type Constant_Scope struct {
	// Local is the declaring package's static integer namespace.
	Local map[string]ast.Expr
	// Imports maps the declaring file's local names to import paths.
	Imports map[string]string
	// Index parses an imported package on demand and caches its constants.
	Index *Bundle_Index
}

// Gives the scope one function's operands resolve in: its own package's constants, its own file's
// imports, and the index that parses an imported package on demand.
func indexed_function_constants(
	function Indexed_Function, index *Bundle_Index,
) (scope Constant_Scope) {
	return Constant_Scope{
		Local: function.Constants, Imports: function.Imports, Index: index,
	}
}

// Gives the declaration a qualified operand names, parsing its package the first time.
func constant_scope_qualified(
	scope Constant_Scope, selector *ast.SelectorExpr,
) (declaration ast.Expr, resolved bool) {
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return nil, false
	}
	if scope.Index == nil {
		return nil, false
	}
	import_path, imported := scope.Imports[qualifier.Name]
	if !imported {
		return nil, false
	}
	// The loader fills Loaded_Constants as a side effect, and caches an empty map for a package
	// outside this module, thus a second miss costs one lookup.
	bundle_index_load(scope.Index, import_path)
	declaration, resolved = scope.Index.Loaded_Constants[import_path][selector.Sel.Name]
	return declaration, resolved
}

// Constant_Frame lets registration evaluate constants without recursive source descent.
type Constant_Frame struct {
	// Expression is the AST node this work item resolves.
	Expression ast.Expr
	// Visited marks the operator's second pass after its operands are available.
	Visited bool
}

func constant_resolve(
	constants Constant_Scope, expression ast.Expr,
) (value Integer_Value, ok bool) {
	frames := []Constant_Frame{{Expression: expression}}
	var values []*big.Int
	for steps := 0; len(frames) != 0; steps++ {
		if steps > CONSTANT_RESOLUTION_STEPS_MAX {
			return Integer_Value{}, false
		}
		frame := frames[len(frames)-1]
		frames = frames[:len(frames)-1]
		frames, values, ok = constant_resolve_frame(constants, frame, frames, values)
		if !ok {
			return Integer_Value{}, false
		}
	}
	if len(values) != 1 {
		return Integer_Value{}, false
	}
	return integer_from_big(values[0])
}

func constant_resolve_frame(
	constants Constant_Scope, frame Constant_Frame, frames []Constant_Frame,
	values []*big.Int,
) (next_frames []Constant_Frame, next_values []*big.Int, ok bool) {
	switch concrete := frame.Expression.(type) {
	case *ast.ParenExpr:
		return append(frames, Constant_Frame{Expression: concrete.X}), values, true
	case *ast.BasicLit:
		if concrete.Kind != token.INT {
			return nil, nil, false
		}
		parsed, parsed_ok := new(big.Int).SetString(concrete.Value, 0)
		if !parsed_ok {
			return nil, nil, false
		}
		return frames, append(values, parsed), true
	case *ast.Ident:
		declaration, declared := constants.Local[concrete.Name]
		if !declared {
			return nil, nil, false
		}
		return append(frames, Constant_Frame{Expression: declaration}), values, true
	case *ast.SelectorExpr:
		declaration, declared := constant_scope_qualified(constants, concrete)
		if !declared {
			return nil, nil, false
		}
		return append(frames, Constant_Frame{Expression: declaration}), values, true
	case *ast.CallExpr:
		if len(concrete.Args) != 1 {
			return nil, nil, false
		}
		callee, is_conversion := concrete.Fun.(*ast.Ident)
		if !is_conversion {
			return nil, nil, false
		}
		// A measure over a constant string is a Go constant expression. Reading it as a
		// conversion would push the string itself, which carries no integer.
		if callee.Name == "len" {
			bytes, measured := constant_resolve_bytes(constants, concrete.Args[0])
			if !measured {
				return nil, nil, false
			}
			return frames, append(values, bytes), true
		}
		return append(frames, Constant_Frame{Expression: concrete.Args[0]}), values, true
	case *ast.UnaryExpr:
		return constant_resolve_unary(frame, concrete, frames, values)
	case *ast.BinaryExpr:
		return constant_resolve_binary(frame, concrete, frames, values)
	}
	return nil, nil, false
}

// Measures a constant string operand in bytes. A name resolves through its own package or through
// the package a qualifier states, thus one declared string carries its own bound and no bundle has
// to restate the number.
func constant_resolve_bytes(
	constants Constant_Scope, expression ast.Expr,
) (bytes *big.Int, measured bool) {
	for step_index := 0; step_index < CONSTANT_RESOLUTION_STEPS_MAX; step_index++ {
		switch concrete := expression.(type) {
		case *ast.ParenExpr:
			expression = concrete.X
		case *ast.Ident:
			declaration, declared := constants.Local[concrete.Name]
			if !declared {
				return nil, false
			}
			expression = declaration
		case *ast.SelectorExpr:
			declaration, declared := constant_scope_qualified(constants, concrete)
			if !declared {
				return nil, false
			}
			expression = declaration
		case *ast.BasicLit:
			if concrete.Kind != token.STRING {
				return nil, false
			}
			text, unquote_error := strconv.Unquote(concrete.Value)
			if unquote_error != nil {
				return nil, false
			}
			return big.NewInt(int64(len(text))), true
		default:
			return nil, false
		}
	}
	return nil, false
}

func constant_resolve_unary(
	frame Constant_Frame, unary *ast.UnaryExpr, frames []Constant_Frame, values []*big.Int,
) (next_frames []Constant_Frame, next_values []*big.Int, ok bool) {
	if unary.Op != token.SUB {
		if unary.Op != token.ADD {
			return nil, nil, false
		}
	}
	if !frame.Visited {
		frames = append(frames, Constant_Frame{Expression: frame.Expression, Visited: true})
		return append(frames, Constant_Frame{Expression: unary.X}), values, true
	}
	if unary.Op == token.SUB {
		values[len(values)-1].Neg(values[len(values)-1])
	}
	return frames, values, true
}

func constant_resolve_binary(
	frame Constant_Frame, binary *ast.BinaryExpr, frames []Constant_Frame, values []*big.Int,
) (next_frames []Constant_Frame, next_values []*big.Int, ok bool) {
	if !frame.Visited {
		frames = append(frames, Constant_Frame{Expression: frame.Expression, Visited: true})
		frames = append(frames, Constant_Frame{Expression: binary.Y})
		return append(frames, Constant_Frame{Expression: binary.X}), values, true
	}
	right := values[len(values)-1]
	left := values[len(values)-2]
	values = values[:len(values)-2]
	result := new(big.Int)
	switch binary.Op {
	case token.ADD:
		result.Add(left, right)
	case token.SUB:
		result.Sub(left, right)
	case token.MUL:
		result.Mul(left, right)
	case token.QUO:
		if right.Sign() == 0 {
			return nil, nil, false
		}
		result.Quo(left, right)
	case token.SHL:
		if right.Sign() < 0 {
			return nil, nil, false
		}
		if !right.IsUint64() {
			return nil, nil, false
		}
		if right.Uint64() > 64 {
			return nil, nil, false
		}
		result.Lsh(left, uint(right.Uint64()))
	default:
		return nil, nil, false
	}
	return frames, append(values, result), true
}

func integer_from_big(value *big.Int) (integer Integer_Value, ok bool) {
	integer.Negative = value.Sign() < 0
	magnitude := new(big.Int).Abs(value)
	if magnitude.BitLen() > 64 {
		return Integer_Value{}, false
	}
	integer.Magnitude = magnitude.Uint64()
	if integer.Magnitude == 0 {
		integer.Negative = false
	}
	return integer, true
}

func integer_compare(first Integer_Value, second Integer_Value) (order int) {
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

func integer_text(value Integer_Value) (text string) {
	text = strconv.FormatUint(value.Magnitude, 10)
	if value.Negative {
		return "-" + text
	}
	return text
}

// Reports every bundle the analyzer recognised by name but could not resolve to a
// declaration, then exits. A recognised-but-unresolvable bundle would seed none of
// its elements while the runtime still enforces them, so its coverage obligations
// would vanish unnoticed; failing keeps coverage from being silently dropped — the
// analyzer descends a bundle or refuses it.
func recorder_check_unresolved(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(
		recorder, reg, "unresolved bundles", reg.Unresolved)
}

// Reports every Range/Enum whose bounds or members the resolver could not read. A
// recognised-but-unevaluable argument leaves the witnesses unseeded while the runtime still
// enforces the guard, so its coverage obligations would vanish unnoticed; failing keeps
// coverage from being silently dropped — the analyzer seeds a guard or refuses it.
func recorder_check_unresolved_bounds(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(
		recorder, reg, "unresolved bounds", reg.Unresolved_Bound)
}

// Reports every assertion whose message is not a keyable literal. Registration owns builder
// identity and therefore cannot publish a runtime plan for a dynamic message. Refuse it rather
// than silently erase the corresponding coverage obligation.
func recorder_check_non_literal_messages(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(
		recorder, reg, "non-literal messages", reg.Non_Literal)
}

// A constant-true guard cannot enforce a property. It can only make an unearned reachability event.
func recorder_check_constant_always_conditions(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(
		recorder, reg, "constant Always conditions", reg.Constant_Always)
}

// Reports every collision. Two assertions sharing a key, or two roots sharing a namespace,
// would silently merge into one entry and mask a gap. A duplicate is fatal, never merged.
func recorder_check_duplicate_messages(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(
		recorder, reg, "duplicate messages", reg.Collision)
}

// A root namespace is a composition's whole identity; a malformed one would silently orphan
// every assertion under it.
func recorder_check_invalid_identifiers(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(
		recorder, reg, "invalid identifiers", reg.Invalid_Identifier)
}

// A structurally refused guard — an exclusion outside its interval, an enum without two
// distinct members — panics at runtime on every call, so registration refuses to seed it.
func recorder_check_invalid_bounds(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(
		recorder, reg, "invalid bounds", reg.Invalid_Bound)
}

// Reports each chain whose subject reflect cannot name uniquely. Such a chain would key its plan
// on two empty strings, so two of them would merge in silence.
func recorder_check_invalid_subjects(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(
		recorder, reg, "invalid assertion subjects", reg.Invalid_Subject)
}

// Reports each subject type that occurs twice in one root's expansion.
func recorder_check_repeated_subjects(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(
		recorder, reg, "repeated subject types", reg.Repeated_Subject)
}

// A bundle composition that recurses into itself would forward its identifier forever at
// runtime, thus the descent refuses the cycle statically.
func recorder_check_bundle_cycles(recorder *Recorder, reg *Registration) {
	recorder_report_registration_failure(recorder, reg, "bundle cycles", reg.Cycle)
}

// One failed doctrine poisons the commit. The flag — not Exit — is the gate because an injected
// Exit returns in tests; every reporter after a failed one still runs, so a multi-failure run
// reports everything before dying.
func recorder_report_registration_failure(
	recorder *Recorder, reg *Registration, label string, lines []string,
) {
	if len(lines) == 0 {
		return
	}
	reg.Failed = true
	banner := "🚨 " + strconv.Itoa(len(lines)) + " " + label + " 🚨"
	fmt.Fprintln(recorder.Output, banner)
	for _, line := range lines {
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
	recorder *Recorder, file_set *token.FileSet, files []*ast.File, reg *Registration,
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
	recorder_report_registration_failure(
		recorder, reg, "bundle control-flow statements", violations)
}

// Finds each bundle body that gives a string literal as a nested bundle namespace. The callsite
// that owns the value owns the namespace. A literal in the body moves that identity to the helper
// declaration. Thus two parents of one helper have one child identity, and the coverage of one
// parent can supply the obligations of the other. Only the Namespace parameter of the body keeps
// the identity at the callsite.
// Reports every violation under one banner and exits 1.
func recorder_check_bundle_literal_namespaces(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File, reg *Registration,
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
				if _, named := ast_bundle_namespace(call); !named {
					return true
				}
				violations = append(violations, recorder_position(file_set, call)+
					"  banned: literal namespace inside bundle "+name)
				return true
			})
		}
	}
	recorder_report_registration_failure(
		recorder, reg, "bundle literal namespaces", violations)
}

// Finds one namespace literal at two bundle callsites. The repeat is a fact about the source text,
// thus registration can find it before expansion. One namespace holds one plan, so the two
// callsites would record two independent value streams into one set of obligations.
// Reports every violation under one banner and exits 1.
func recorder_check_duplicate_bundle_namespaces(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File, reg *Registration,
) {
	var violations []string
	// The order of the walk, not the order of this map, controls the report. Thus it stays
	// deterministic.
	registered := map[string]bool{}
	for _, file := range files {
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			if function.Body == nil {
				continue
			}
			violations = append(violations, recorder_duplicate_namespace_violations(
				file_set, function, registered)...)
		}
	}
	recorder_report_registration_failure(
		recorder, reg, "duplicate bundle namespaces", violations)
}

// Adds each namespace literal of the body to registered. Reports each literal that registered has
// already.
func recorder_duplicate_namespace_violations(
	file_set *token.FileSet, function *ast.FuncDecl, registered map[string]bool,
) (violations []string) {
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		namespace, named := ast_bundle_namespace(call)
		if !named {
			return true
		}
		if registered[namespace] {
			violations = append(violations, recorder_position(file_set, call)+
				"  banned: duplicate namespace "+strconv.Quote(namespace))
			return true
		}
		registered[namespace] = true
		return true
	})
	return violations
}

// Gives the compile-time literal namespace of a bundle call. named is false when the call is not a
// bundle, when it has no arguments, or when its last argument is not a literal. The forwarded
// Namespace parameter is one such argument.
func ast_bundle_namespace(call *ast.CallExpr) (namespace string, named bool) {
	if !ast_is_invariants_name(ast_callee_name(call)) {
		return "", false
	}
	if len(call.Args) == 0 {
		return "", false
	}
	return ast_string_literal(call, len(call.Args)-1)
}

// Reports whether name resolves to the predeclared bool through this package's type declarations.
// A defined type can name another defined type, thus the walk follows that chain.
func ast_type_is_boolean(name string, package_types map[string]ast.Expr) (boolean bool) {
	for step_index := 0; step_index < CONSTANT_RESOLUTION_STEPS_MAX; step_index++ {
		if name == "bool" {
			return true
		}
		base, declared := package_types[name]
		if !declared {
			return false
		}
		identifier, is_identifier := base.(*ast.Ident)
		if !is_identifier {
			return false
		}
		name = identifier.Name
	}
	return false
}

// Reports whether function declares name as a type parameter. This scans instead of building the
// whole set, because the usual declaration has no type parameter at all and one membership test
// does not earn a map.
func ast_type_parameter_named(function *ast.FuncDecl, name string) (declared bool) {
	if function == nil {
		return false
	}
	if function.Type.TypeParams == nil {
		return false
	}
	for _, field := range function.Type.TypeParams.List {
		for _, parameter := range field.Names {
			if parameter.Name == name {
				return true
			}
		}
	}
	return false
}

// Generic type parameters can denote primitives even though their identifiers are not builtins;
// resolving them prevents a generic primitive helper from bypassing the helper contract.
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

// Reports whether node is a branching or looping statement banned in a bundle body.
func ast_is_control_flow(node ast.Node) (is_control_flow bool) {
	switch node.(type) {
	case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt,
		*ast.ForStmt, *ast.RangeStmt, *ast.SelectStmt:
		return true
	}
	return false
}

// Planned_Seed is one Events entry a registration pass intends to create; planning defers the
// stores so a failed registration leaves the tracker empty instead of partially seeded.
type Planned_Seed struct {
	// Key is a bare message for Always or namespace, ordinal, and message for a builder link.
	Key string
	// Kind is the entry's assertion kind.
	Kind Assertion_Kind
	// Condition is the source text of the asserted expression, for the gap report.
	Condition string
	// Domain is the Range interval this seed guards, or nil.
	Domain *Assertion_Domain
}

// Registration accumulates everything a registration pass gathers before deciding whether to
// fail: the planned seeds and the diagnostics. Each diagnostic bucket is fatal on its own (see
// the recorder_check_* reporters); the commit runs only when none fired, so Events is
// all-or-nothing.
type Registration struct {
	// Unresolved holds bundles recognised by name but not resolvable to a declaration.
	Unresolved []string
	// Non_Literal holds messages that are not string literals or carry the key separator —
	// either way the static side cannot key them.
	Non_Literal []string
	// Constant_Always holds true guard conditions that cannot enforce a property.
	Constant_Always []string
	// Collision holds duplicate keys and duplicate root identifiers.
	Collision []string
	// Unresolved_Bound holds Range/Enum arguments the constant resolver could not resolve.
	Unresolved_Bound []string
	// Invalid_Identifier holds identifier arguments illegal at their position: a non-literal
	// at a root, an empty or NUL-carrying literal, or a non-forwarded identifier in a bundle.
	Invalid_Identifier []string
	// Invalid_Bound holds structurally refused Range exclusions and Enum member sets.
	Invalid_Bound []string
	// Invalid_Chain holds malformed, empty, split, duplicate, and over-cap builder chains.
	Invalid_Chain []string
	// Invalid_Subject holds chains whose subject reflect cannot name uniquely.
	Invalid_Subject []string
	// Cycle holds bundle compositions that recurse back into themselves.
	Cycle []string
	// Planned holds the Events entries to create when no diagnostic fired.
	Planned []Planned_Seed
	// Planned_Assertions becomes the immutable runtime emission plan after every seed commits.
	Planned_Assertions map[Plan_Key]*Assertion_Plan
	// Planned_Inline becomes the same program for each inline helper, keyed by its message.
	Planned_Inline map[string]*Assertion_Plan
	// Planned_Keys detects key collisions across the whole plan.
	Planned_Keys map[string]bool
	// Registered_Calls holds each eager callsite already registered. One body is reached by the
	// file walk and by every descent into it, thus without this a second visit would report its
	// own guards as duplicates.
	Registered_Calls map[token.Pos]bool
	// Message_Owner holds every message that names an assertion on its own: an Always guard and
	// an inline helper. Their key shapes differ, thus a shared message would not collide in
	// Planned_Keys, and one would silently shadow the other in the report.
	Message_Owner map[string]bool
	// Forbidden_Properties preserves panic-able Range holes outside the coverage plan because a
	// forbidden observation fails instead of earning coverage.
	Forbidden_Properties int
	// Namespace_Owner_Path prevents a second static invocation path from merging independent
	// value streams while repeated descent through the first path stays idempotent.
	Namespace_Owner_Path map[Plan_Key][]token.Pos
	// Subject_Owner records the owner path that first reached each subject type under one root.
	// An identical path is idempotent re-expansion. A different path is a repeat.
	Subject_Owner map[Plan_Key][]token.Pos
	// Root_Namespace_Owner keeps one namespace literal to one root. The subject type separates
	// sibling chains under a root, thus it must not license the same literal at a second root.
	Root_Namespace_Owner map[Namespace]token.Pos
	// Repeated_Subject holds each subject type that occurs twice in one root's expansion.
	Repeated_Subject []string
	// Failed poisons the commit; it is the gate rather than Exit because an injected Exit
	// returns in tests.
	Failed bool
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

// Renders one unresolvable bundle as
// "<site>  unresolved bundle: <name> cannot be analyzed".
func recorder_unresolved_line(file_set *token.FileSet, call *ast.CallExpr) (line string) {
	return recorder_position(file_set, call) + "  unresolved bundle: " +
		ast_callee_name(call) + " cannot be analyzed"
}

// Resolves a bundle call to its declaration using the calling file's imports: a
// bare call hits package_functions (same-package); a qualified pkg.Foo_Invariants resolves
// pkg to an import path and, if it is inside the module, loads that package.
// found is false for an unresolvable bundle (cross-module, missing go.mod, an
// unknown qualifier, or an absent declaration).
func bundle_index_lookup(
	index *Bundle_Index, imports map[string]string,
	package_functions map[string]Indexed_Function, bundle *ast.CallExpr,
) (function Indexed_Function, found bool) {
	qualifier, name := ast_bundle_qualifier(bundle)
	if qualifier == "" {
		same_package, present := package_functions[name]
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
		// Clean collapses the double slash a module rooted at "/" would otherwise produce.
		remainder := strings.TrimPrefix(import_path, index.Module_Path)
		return path.Clean(index.Module_Root + remainder), true
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
	index.Loaded_Constants[import_path] = map[string]ast.Expr{}
	directory, resolved := bundle_index_module_root(index, import_path)
	if !resolved {
		return functions
	}
	files, _ := recorder_parse_directory(index.File_System, index.File_Set, directory)
	constants := ast_index_constants(files)
	index.Loaded_Constants[import_path] = constants
	indexed := ast_index_functions(
		files, index.File_Set, index.Module_Path, index.Module_Root,
		constants, ast_index_package_types(files))
	for name, function := range indexed {
		function.Is_Sugar = import_path == index.Sugar_Package
		function.Package_Functions = functions
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

// Reports whether name is a bundle function name: a *_Invariants (exported) or
// *_invariants (unexported) suffix. Both casings exist because the bundle's name
// follows its type's casing — the free-function-over-a-type rule the linter
// enforces — so the analyzer must accept either or silently drop an unexported
// type's bundle coverage.
func ast_is_invariants_name(name string) (is_bundle bool) {
	return strings.HasSuffix(name, "_Invariants") || strings.HasSuffix(name, "_invariants")
}

// Returns "file:line" for the node's start position.
func recorder_position(file_set *token.FileSet, node ast.Node) (site string) {
	return recorder_token_position(file_set, node.Pos())
}

func recorder_token_position(file_set *token.FileSet, source token.Pos) (site string) {
	position := file_set.Position(source)
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

// Coverage_Gap_Reporter writes one complete gap report in its selected representation.
type Coverage_Gap_Reporter func(output io.Writer, gaps []Coverage_Gap) (err error)

// A Coverage_Gap is one stable flat record for a seeded obligation the run did not exercise.
type Coverage_Gap struct {
	// Section distinguishes branch exploration from assertion reachability.
	Section string `json:"section"`
	// Assertion is the builder namespace or the eager assertion identity.
	Assertion string `json:"assertion"`
	// Package is the import path of the subject a builder keyed its plan on. A namespace
	// descends unchanged, thus it alone cannot say which bundle of a tree lacks evidence. An
	// eager message is empty here, because it owns a globally unique message instead.
	Package string `json:"package"`
	// Type is the subject type a builder keyed its plan on. Two types that share their
	// constants state one property under one namespace by design, and this separates them.
	Type string `json:"type"`
	// Link is the expanded builder ordinal; reachability records leave it null.
	Link *uint8 `json:"link"`
	// Absent is true, false, or reachability according to the absent obligation.
	Absent string `json:"missing"`
	// Property is the builder link identity; reachability records leave it null.
	Property *string `json:"property"`
	// Source is the unquoted Go expression registered for the assertion.
	Source string `json:"source"`
	// Declared is the interval a Range registered, empty in every other section.
	Declared string `json:"declared"`
	// Observed is the interval the run put through that Range, empty when nothing reached it.
	Observed string `json:"observed"`
	// Observations counts the values that reached the Range, null in every other section.
	Observations *int64 `json:"observations"`
}

// Recorder_Analyze_Assertion_Frequency reports every pre-registered assertion
// whose true branch never fired and every Sometimes whose false branch never
// fired — naming each by its message and condition source — then calls Exit(1) when
// any gap exists. It is a no-op in a benchmark or a fuzz worker subprocess; a plain test
// run and the fuzz coordinator both analyze.
func Recorder_Analyze_Assertion_Frequency(recorder *Recorder) {
	if !recorder_output_configuration_valid(recorder) {
		return
	}
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
	// A domain row is context and never a verdict. A run whose every obligation has evidence
	// stays silent, thus the domains ride an existing gap report rather than making one.
	if coverage_gap_count(gaps) == 0 {
		return
	}
	reporter := recorder.Report_Coverage_Gaps
	if reporter == nil {
		reporter = Coverage_Gap_Table_Write
	}
	if report_error := reporter(recorder.Output, gaps); report_error != nil {
		fmt.Fprintln(recorder.Output, "invariant: coverage gap report failed: "+
			report_error.Error())
	}
	recorder.Exit(1)
}

// Rejects a composition-tier output selection before it can produce a differently shaped result.
func recorder_output_configuration_valid(recorder *Recorder) (valid bool) {
	if recorder.Output_Configuration_Diagnostic == "" {
		return true
	}
	fmt.Fprintln(recorder.Output, "invariant: "+recorder.Output_Configuration_Diagnostic)
	recorder.Exit(1)
	return false
}

// Walks the tracker and returns every coverage gap across all seeded assertions, plus one domain
// row for each Range. A Range with no gap still reports its interval, because a bound far wider
// than the values a run reaches is a defect the gap sections cannot show.
func recorder_collect_gaps(recorder *Recorder) (gaps []Coverage_Gap) {
	recorder.Events.Range(func(key, value any) (continue_iteration bool) {
		metadata := value.(*Assertion_Metadata)
		gaps = append(gaps, assertion_metadata_gaps(metadata)...)
		if metadata.Domain != nil {
			gaps = append(gaps, assertion_domain_row(metadata))
		}
		return true
	})
	sort.Slice(gaps, func(left_index int, right_index int) (less bool) {
		return coverage_gap_less(gaps[left_index], gaps[right_index])
	})
	return gaps
}

// Returns the coverage gaps one assertion exhibits. A Sometimes contributes a gap
// per branch it never observed (true and/or false); an Always that never
// fired is a single gap; a fully exercised assertion contributes none.
func assertion_metadata_gaps(metadata *Assertion_Metadata) (gaps []Coverage_Gap) {
	if metadata.Kind == ASSERTION_KIND_SOMETIMES {
		if metadata.Frequency.Load() == 0 {
			gaps = append(gaps, coverage_gap_branch(metadata, "true"))
		}
		if metadata.False_Frequency.Load() == 0 {
			gaps = append(gaps, coverage_gap_branch(metadata, "false"))
		}
		return gaps
	}
	if metadata.Frequency.Load() != 0 {
		return gaps
	}
	assertion, package_path, subject := coverage_gap_identity(metadata.Message)
	return append(gaps, Coverage_Gap{
		Section: "reachability", Assertion: assertion, Package: package_path, Type: subject,
		Absent: "reachability", Source: metadata.Condition,
	})
}

// Builds one Range's domain row. The declared interval always prints. The observed interval prints
// only when a value reached the Range, because the two seeded keys are extremes and not a reading.
func assertion_domain_row(metadata *Assertion_Metadata) (row Coverage_Gap) {
	domain := metadata.Domain
	assertion, package_path, subject := coverage_gap_identity(metadata.Message)
	count := domain.Observation_Count.Load()
	observed := ""
	if count != 0 {
		observed = assertion_domain_interval(
			assertion_domain_value(domain.Observed_Minimum_Key.Load(), domain.Unsigned),
			assertion_domain_value(domain.Observed_Maximum_Key.Load(), domain.Unsigned))
	}
	link := assertion_key_ordinal(metadata.Message)
	return Coverage_Gap{
		Section: "domain", Assertion: assertion, Package: package_path, Type: subject,
		Link: link, Absent: "domain", Source: metadata.Condition,
		Declared: assertion_domain_interval(
			domain.Declared_Minimum, domain.Declared_Maximum),
		Observed: observed, Observations: &count,
	}
}

// Renders one interval the way both bounds read together.
func assertion_domain_interval(
	minimum Integer_Value, maximum Integer_Value,
) (interval string) {
	return integer_text(minimum) + ".." + integer_text(maximum)
}

// Reads the ordinal a builder key carries, or nil when the key is an eager message.
func assertion_key_ordinal(message string) (ordinal *uint8) {
	parts := strings.Split(message, ELEMENT_MESSAGE_SEPARATOR)
	if len(parts) != ASSERTION_KEY_PARTS {
		return nil
	}
	position, position_error := strconv.ParseUint(parts[3], 10, 8)
	if position_error != nil {
		return nil
	}
	link := uint8(position)
	return &link
}

// Splits the registration-owned key into the identity a reader gets. The key has namespace,
// package, type, ordinal, and message. An eager Always cannot contain the separator, thus it is its
// own assertion and owns neither a package nor a subject type.
func coverage_gap_identity(
	message string,
) (assertion string, package_path string, subject string) {
	parts := strings.Split(message, ELEMENT_MESSAGE_SEPARATOR)
	if len(parts) != ASSERTION_KEY_PARTS {
		return message, "", ""
	}
	return parts[0], parts[1], parts[2]
}

// Parses the registration-owned builder identity into the stable reporting schema. A namespace
// descends a tree unchanged, thus two bundles under one root share it. The subject type is what
// separates them, and duplicated bundles across types are what the composition model calls for.
func coverage_gap_branch(metadata *Assertion_Metadata, absent string) (gap Coverage_Gap) {
	unparsed := Coverage_Gap{
		Section: "branch", Assertion: metadata.Message,
		Absent: absent, Source: metadata.Condition,
	}
	parts := strings.Split(metadata.Message, ELEMENT_MESSAGE_SEPARATOR)
	if len(parts) != ASSERTION_KEY_PARTS {
		return unparsed
	}
	ordinal, ordinal_error := strconv.ParseUint(parts[3], 10, 8)
	if ordinal_error != nil {
		return unparsed
	}
	link := uint8(ordinal)
	property := parts[4]
	return Coverage_Gap{
		Section: "branch", Assertion: parts[0], Package: parts[1], Type: parts[2],
		Link: &link, Absent: absent, Property: &property, Source: metadata.Condition,
	}
}

// Orders sections and every visible field so map iteration can never leak into either format.
func coverage_gap_less(left Coverage_Gap, right Coverage_Gap) (less bool) {
	if left.Section != right.Section {
		return left.Section == "branch"
	}
	if left.Assertion != right.Assertion {
		return left.Assertion < right.Assertion
	}
	if left.Package != right.Package {
		return left.Package < right.Package
	}
	if left.Type != right.Type {
		return left.Type < right.Type
	}
	if coverage_gap_link(left) != coverage_gap_link(right) {
		return coverage_gap_link(left) < coverage_gap_link(right)
	}
	if left.Absent != right.Absent {
		return left.Absent < right.Absent
	}
	if coverage_gap_property(left) != coverage_gap_property(right) {
		return coverage_gap_property(left) < coverage_gap_property(right)
	}
	return left.Source < right.Source
}

// Gives null links one stable sentinel without exposing that implementation in the schema.
func coverage_gap_link(gap Coverage_Gap) (link int) {
	if gap.Link == nil {
		return -1
	}
	return int(*gap.Link)
}

// Gives null properties one stable sortable value.
func coverage_gap_property(gap Coverage_Gap) (property string) {
	if gap.Property == nil {
		return ""
	}
	return *gap.Property
}

// Coverage_Gap_Table_Write renders the complete human report as dynamically aligned Markdown.
func Coverage_Gap_Table_Write(output io.Writer, gaps []Coverage_Gap) (err error) {
	var report strings.Builder
	banner := "🚨 " + strconv.Itoa(coverage_gap_count(gaps)) + " coverage gaps 🚨"
	report.WriteString(banner + "\n")
	branch := coverage_gap_section(gaps, "branch")
	if len(branch) > 0 {
		report.WriteString("\n# Branch gaps (" + strconv.Itoa(len(branch)) + ")\n\n")
		coverage_gap_branch_table_write(&report, branch)
	}
	reachability := coverage_gap_section(gaps, "reachability")
	if len(reachability) > 0 {
		report.WriteString("\n# Reachability gaps (" +
			strconv.Itoa(len(reachability)) + ")\n\n")
		coverage_gap_reachability_table_write(&report, reachability)
	}
	domains := coverage_gap_section(gaps, "domain")
	if len(domains) > 0 {
		report.WriteString("\n# Range domains (" + strconv.Itoa(len(domains)) + ")\n\n")
		coverage_gap_domain_table_write(&report, domains)
	}
	report.WriteString("\n" + banner + "\n")
	written, write_error := io.WriteString(output, report.String())
	if write_error != nil {
		return write_error
	}
	if written != report.Len() {
		return io.ErrShortWrite
	}
	return nil
}

// Counts the rows that are an absent obligation. A domain row states what a Range saw and never
// that something is missing, thus it belongs to no banner and to no verdict.
func coverage_gap_count(gaps []Coverage_Gap) (count int) {
	for _, gap := range gaps {
		if gap.Section != "domain" {
			count++
		}
	}
	return count
}

// Writes the domain table, whose Observations column is right aligned as a tally.
func coverage_gap_domain_table_write(report *strings.Builder, gaps []Coverage_Gap) {
	rows := make([][DOMAIN_TABLE_COLUMNS]string, 0, len(gaps))
	widths := [DOMAIN_TABLE_COLUMNS]int{
		len("Assertion"), len("Type"), len("Link"), len("Declared"),
		len("Observed"), len("Count"), len("Source"),
	}
	for _, gap := range gaps {
		link := ""
		if gap.Link != nil {
			link = strconv.Itoa(int(*gap.Link))
		}
		count := ""
		if gap.Observations != nil {
			count = strconv.FormatInt(*gap.Observations, 10)
		}
		row := [DOMAIN_TABLE_COLUMNS]string{
			coverage_gap_table_cell(gap.Assertion),
			coverage_gap_table_cell(gap.Type), link,
			coverage_gap_table_cell(gap.Declared),
			coverage_gap_table_cell(gap.Observed), count,
			coverage_gap_table_cell(gap.Source),
		}
		rows = append(rows, row)
		for column_index := range widths {
			widths[column_index] = integer_maximum(
				widths[column_index], len(row[column_index]))
		}
	}
	coverage_gap_domain_header_write(report, widths)
	for _, row := range rows {
		fmt.Fprintf(report, "| %-*s | %-*s | %*s | %-*s | %-*s | %*s | %-*s |\n",
			widths[0], row[0], widths[1], row[1], widths[2], row[2],
			widths[3], row[3], widths[4], row[4], widths[5], row[5],
			widths[6], row[6])
	}
}

// Keeps the domain header and its alignment markers the same width as every row cell.
func coverage_gap_domain_header_write(
	report *strings.Builder, widths [DOMAIN_TABLE_COLUMNS]int,
) {
	fmt.Fprintf(report, "| %-*s | %-*s | %*s | %-*s | %-*s | %*s | %-*s |\n",
		widths[0], "Assertion", widths[1], "Type", widths[2], "Link",
		widths[3], "Declared", widths[4], "Observed", widths[5], "Count",
		widths[6], "Source")
	report.WriteString("|" + strings.Repeat("-", widths[0]+2))
	report.WriteString("|" + strings.Repeat("-", widths[1]+2))
	report.WriteString("|" + strings.Repeat("-", widths[2]+1) + ":")
	report.WriteString("|" + strings.Repeat("-", widths[3]+2))
	report.WriteString("|" + strings.Repeat("-", widths[4]+2))
	report.WriteString("|" + strings.Repeat("-", widths[5]+1) + ":")
	report.WriteString("|" + strings.Repeat("-", widths[6]+2))
	report.WriteString("|\n")
}

// Selects one report section without changing the collector's stable order.
func coverage_gap_section(gaps []Coverage_Gap, section string) (selected []Coverage_Gap) {
	selected = make([]Coverage_Gap, 0, len(gaps))
	for _, gap := range gaps {
		if gap.Section == section {
			selected = append(selected, gap)
		}
	}
	return selected
}

// Materializes escaped cells once so measuring and writing use identical text.
func coverage_gap_branch_rows(gaps []Coverage_Gap) (rows [][BRANCH_TABLE_COLUMNS]string) {
	rows = make([][BRANCH_TABLE_COLUMNS]string, 0, len(gaps))
	for _, gap := range gaps {
		// An inline helper owns no ordinal, so its link cell stays empty rather than
		// printing the sort sentinel.
		link := ""
		if gap.Link != nil {
			link = strconv.Itoa(int(*gap.Link))
		}
		rows = append(rows, [BRANCH_TABLE_COLUMNS]string{
			coverage_gap_table_cell(gap.Assertion),
			coverage_gap_table_cell(gap.Type),
			link,
			coverage_gap_table_cell(gap.Absent),
			coverage_gap_table_cell(coverage_gap_property(gap)),
			coverage_gap_table_cell(gap.Source),
		})
	}
	return rows
}

// Writes the six-column branch table with Link right aligned as an ordinal.
func coverage_gap_branch_table_write(report *strings.Builder, gaps []Coverage_Gap) {
	rows := coverage_gap_branch_rows(gaps)
	widths := [BRANCH_TABLE_COLUMNS]int{
		len("Assertion"), len("Type"), len("Link"), len("Missing"),
		len("Property"), len("Source"),
	}
	for _, row := range rows {
		for column_index := range widths {
			widths[column_index] = integer_maximum(
				widths[column_index], len(row[column_index]))
		}
	}
	fmt.Fprintf(report, "| %-*s | %-*s | %*s | %-*s | %-*s | %-*s |\n",
		widths[0], "Assertion", widths[1], "Type", widths[2], "Link",
		widths[3], "Missing", widths[4], "Property", widths[5], "Source")
	coverage_gap_branch_separator_write(report, widths)
	for _, row := range rows {
		fmt.Fprintf(report, "| %-*s | %-*s | %*s | %-*s | %-*s | %-*s |\n",
			widths[0], row[0], widths[1], row[1], widths[2], row[2],
			widths[3], row[3], widths[4], row[4], widths[5], row[5])
	}
}

// Keeps Markdown alignment markers the same width as their header and row cells.
func coverage_gap_branch_separator_write(
	report *strings.Builder, widths [BRANCH_TABLE_COLUMNS]int,
) {
	report.WriteString("|" + strings.Repeat("-", widths[0]+2))
	report.WriteString("|" + strings.Repeat("-", widths[1]+2))
	report.WriteString("|" + strings.Repeat("-", widths[2]+1) + ":")
	for _, width := range widths[3:] {
		report.WriteString("|" + strings.Repeat("-", width+2))
	}
	report.WriteString("|\n")
}

// Writes the smaller reachability table because its section already names the absent obligation.
func coverage_gap_reachability_table_write(report *strings.Builder, gaps []Coverage_Gap) {
	rows := make([][REACHABILITY_TABLE_COLUMNS]string, 0, len(gaps))
	widths := [REACHABILITY_TABLE_COLUMNS]int{
		len("Assertion"), len("Type"), len("Source"),
	}
	for _, gap := range gaps {
		row := [REACHABILITY_TABLE_COLUMNS]string{
			coverage_gap_table_cell(gap.Assertion),
			coverage_gap_table_cell(gap.Type),
			coverage_gap_table_cell(gap.Source),
		}
		rows = append(rows, row)
		for column_index := range widths {
			widths[column_index] = integer_maximum(
				widths[column_index], len(row[column_index]))
		}
	}
	fmt.Fprintf(report, "| %-*s | %-*s | %-*s |\n",
		widths[0], "Assertion", widths[1], "Type", widths[2], "Source")
	for _, width := range widths {
		report.WriteString("|" + strings.Repeat("-", width+2))
	}
	report.WriteString("|\n")
	for _, row := range rows {
		fmt.Fprintf(report, "| %-*s | %-*s | %-*s |\n",
			widths[0], row[0], widths[1], row[1], widths[2], row[2])
	}
}

// Escapes structure-significant characters and represents physical lines inside one table cell.
func coverage_gap_table_cell(value string) (escaped string) {
	replacer := strings.NewReplacer(
		"\\", "\\\\", "|", "\\|", "\r\n", "<br>", "\r", "<br>", "\n", "<br>")
	return replacer.Replace(value)
}

// Avoids importing a general-purpose numeric package for one table measurement operation.
func integer_maximum(left int, right int) (maximum int) {
	if left > right {
		return left
	}
	return right
}

// Recorder_Assertion_Summary keeps the enforced subset visible because an undifferentiated total
// cannot distinguish branch exploration from contracts whose violation terminates execution.
func Recorder_Assertion_Summary(recorder *Recorder) (summary string) {
	individual := recorder.Forbidden_Properties
	panic_able := recorder.Forbidden_Properties
	recorder.Events.Range(func(key, value any) (continue_iteration bool) {
		metadata := value.(*Assertion_Metadata)
		switch metadata.Kind {
		case ASSERTION_KIND_ALWAYS:
			individual++
			panic_able++
		default:
			// A Sometimes must witness both its true and its false branch, so it
			// counts twice.
			individual += 2
		}
		return true
	})
	if recorder.Package_Label != "" {
		return fmt.Sprintf(
			"✓ %s: tested %d properties (%d individual, of which %d are panic-able)",
			recorder.Package_Label, individual, individual, panic_able)
	}
	return fmt.Sprintf(
		"✓ tested %d properties (%d individual, of which %d are panic-able)",
		individual, individual, panic_able)
}

// Recorder_Run_Test_Main runs the injected TestMain process. It registers the selected
// directories and runs the suite. It then reports gaps and exits with the suite code.
// A clean run writes its property summary to Tty, or to Output when Tty is nil.
func Recorder_Run_Test_Main(recorder *Recorder, m *testing.M, directories ...string) {
	if !recorder_output_configuration_valid(recorder) {
		return
	}
	Recorder_Register_Packages_For_Analysis(recorder, directories...)
	code := m.Run()
	// A fuzz coordinator merges worker coverage because it never ran the fuzzed body itself
	// (see Coverage / Modes).
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

// Assertion_Registration_Chain is the one expression registration can prove without data flow.
type Assertion_Registration_Chain struct {
	// Root is the Assertions constructor below the links.
	Root *ast.CallExpr
	// Links retain source order so ordinal identity matches runtime expansion.
	Links []*ast.CallExpr
}

// Assertion_Registration_Link is identity-free until a namespace commits it to the global plan.
type Assertion_Registration_Link struct {
	// Message is the literal or preset-owned identity registered for this link.
	Message string
	// Condition preserves the source expression shown in gap diagnostics.
	Condition string
	// Kind selects the one-branch or two-branch coverage obligation.
	Kind Assertion_Kind
	// Forbidden_Properties rides the first Range guard because holes panic before they can own
	// observable coverage entries, while the guard already shares their static domain.
	Forbidden_Properties int
	// Domain rides that same first guard, and for the same reason: one Range owns one interval,
	// thus its report row belongs to one of its links and not to each.
	Domain *Assertion_Domain
}

func recorder_register_assertion_files(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File,
	index *Bundle_Index, reg *Registration,
) {
	for _, file := range files {
		imports := ast_file_imports(file)
		file_package := recorder_file_package(file_set, file, index)
		allow_unqualified := file_package == index.Sugar_Package
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			if function.Body == nil {
				continue
			}
			indexed := Indexed_Function{
				Declaration: function, Package: file_package, Imports: imports,
				Constants: index.Constants, Package_Functions: index.Same_Set,
				Package_Types: index.Package_Types, Is_Sugar: allow_unqualified,
			}
			recorder_register_assertion_function(
				recorder, file_set, indexed, index, reg)
		}
	}
}

// Registration derives package identity from source location because syntax alone cannot
// distinguish the default sugar surface from an unrelated unqualified function.
func recorder_file_package(
	file_set *token.FileSet, file *ast.File, index *Bundle_Index,
) (import_path string) {
	return ast_file_package_path(file, file_set, index.Module_Path, index.Module_Root)
}

// Gives a file's import path from its source location. This must agree exactly with what
// reflect.Type.PkgPath reports for a type declared in that file, because registration and the
// runtime build one Plan_Key from the two sides.
func ast_file_package_path(
	file *ast.File, file_set *token.FileSet, module_path string, module_root string,
) (import_path string) {
	if module_path == "" {
		return ""
	}
	absolute := file_set.Position(file.Pos()).Filename
	relative := strings.TrimPrefix(path.Dir(absolute), module_root)
	relative = strings.TrimPrefix(relative, "/")
	if relative == "" {
		return module_path
	}
	return path.Join(module_path, relative)
}

func recorder_check_assertion_bundle_contract(
	recorder *Recorder, file_set *token.FileSet, files []*ast.File,
	index *Bundle_Index, reg *Registration,
) {
	var violations []string
	for _, file := range files {
		is_sugar := recorder_file_package(file_set, file, index) == index.Sugar_Package
		for _, declaration := range file.Decls {
			function, is_function := declaration.(*ast.FuncDecl)
			if !is_function {
				continue
			}
			if !ast_is_invariants_name(function.Name.Name) {
				continue
			}
			if ast_assertion_namespace_parameter(function) == "" {
				position := recorder_position(file_set, function)
				violations = append(violations, position+
					"  bundle must end with a Namespace parameter")
				continue
			}
			if is_sugar {
				continue
			}
			subject := recorder_assertion_bundle_subject(function)
			if subject == nil {
				continue
			}
			if recorder_type_is_primitive(
				subject, recorder_bundle_type_parameters(function)) {
				position := recorder_position(file_set, function)
				violations = append(violations,
					position+"  primitive bundle subject: "+function.Name.Name)
				continue
			}
			violations = append(violations, recorder_boolean_bundle_violations(
				file_set, function, index.Package_Types)...)
		}
	}
	recorder_report_registration_failure(
		recorder, reg, "invalid bundle contracts", violations)
}

// Gives the bare name of a bundle's subject type, matching what reflect.Type.Name reports. named is
// false for anything reflect cannot name uniquely: an unnamed composite, a pointer to one, a
// generic instantiation, and a subject imported from another package. The house rule puts a bundle
// directly below its type, so a same-package bare name is the only accepted form.
func ast_assertion_subject_type(
	function *ast.FuncDecl, package_types map[string]ast.Expr,
) (name string, named bool) {
	subject := recorder_assertion_bundle_subject(function)
	if subject == nil {
		return "", false
	}
	return ast_defined_type_name(subject, function, package_types)
}

// Reports a Boolean bundle that does not state exactly one Sometimes. A Boolean has two values and
// both are obligations, thus one axis states the whole type. A second link would split one property
// across two entries, and no chain at all would leave both values unwitnessed.
func recorder_boolean_bundle_violations(
	file_set *token.FileSet, function *ast.FuncDecl, package_types map[string]ast.Expr,
) (violations []string) {
	name, named := ast_assertion_subject_type(function, package_types)
	if !named {
		return nil
	}
	if !ast_type_is_boolean(name, package_types) {
		return nil
	}
	links, found := ast_assertion_bundle_links(function)
	position := recorder_position(file_set, function)
	if !found {
		return append(violations, position+
			"  Boolean bundle states no chain: "+function.Name.Name)
	}
	if len(links) != 1 {
		return append(violations, position+
			"  Boolean bundle states more than one link: "+function.Name.Name)
	}
	if ast_assertion_chain_method(links[0]) != "Sometimes" {
		return append(violations, position+
			"  Boolean bundle states a link that is not Sometimes: "+function.Name.Name)
	}
	return nil
}

// Gives the fluent links of the one chain a bundle body opens. found is false when the body opens
// no chain that Ensure terminates.
func ast_assertion_bundle_links(function *ast.FuncDecl) (links []*ast.CallExpr, found bool) {
	if function.Body == nil {
		return nil, false
	}
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		if found {
			return false
		}
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		if ast_assertion_chain_method(call) != "Ensure" {
			return true
		}
		chain, parsed := ast_assertion_chain_from_ensure(call)
		if !parsed {
			return true
		}
		links = chain.Links
		found = true
		return false
	})
	return links, found
}

func recorder_assertion_bundle_subject(function *ast.FuncDecl) (subject ast.Expr) {
	if function.Type.Params == nil {
		return nil
	}
	if len(function.Type.Params.List) == 0 {
		return nil
	}
	return function.Type.Params.List[0].Type
}

func recorder_register_assertion_function(
	recorder *Recorder, file_set *token.FileSet, function Indexed_Function,
	index *Bundle_Index, reg *Registration,
) {
	parameter := ast_assertion_namespace_parameter(function.Declaration)
	is_bundle := ast_is_invariants_name(function.Declaration.Name.Name)
	ensured := recorder_assertion_ensured_roots(function.Declaration)
	ast.Inspect(function.Declaration.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		recorder_register_assertion_always(
			file_set, call, function.Is_Sugar,
			indexed_function_constants(function, index), reg)
		recorder_register_inline_assertion(
			file_set, call, function.Is_Sugar,
			indexed_function_constants(function, index), reg)
		if ast_assertion_chain_method(call) == "Ensure" {
			chain, parsed := ast_assertion_chain_from_ensure(call)
			if !parsed {
				recorder_invalid_chain(file_set, call, reg,
					"Ensure does not terminate one assertion call nest")
				return false
			}
			if is_bundle {
				recorder_validate_assertion_template(
					file_set, chain, parameter,
					indexed_function_constants(function, index), reg)
				return false
			}
			// A chain identifies itself by its subject type, and only a bundle owns
			// one type. An ordinary body states an inline helper instead.
			recorder_invalid_chain(file_set, chain.Root, reg,
				"Tree chain is outside an _Invariants bundle")
			return false
		}
		if ast_assertion_root(call, function.Is_Sugar) {
			if !ensured[call.Pos()] {
				recorder_invalid_chain(file_set, call, reg,
					"Tree chain is not terminated by Ensure")
			}
			return true
		}
		if is_bundle {
			return true
		}
		if !ast_is_invariants_name(ast_callee_name(call)) {
			return true
		}
		recorder_register_assertion_bundle_call(
			file_set, call, function.Imports, index, reg)
		return true
	})
}

// Registers one inline helper call. The message is the whole identity, so it goes through
// recorder_plan_seed exactly as an Always message does — which is also what makes a collision with
// an Always message, or with a second inline helper, fatal without a separate check.
func recorder_register_inline_assertion(
	file_set *token.FileSet, call *ast.CallExpr, allow_unqualified bool,
	constants Constant_Scope, reg *Registration,
) {
	kind, matched := ast_inline_assertion_kind(call, allow_unqualified)
	if !matched {
		return
	}
	if reg.Registered_Calls[call.Pos()] {
		return
	}
	reg.Registered_Calls[call.Pos()] = true
	message, literal := ast_string_literal(call, len(call.Args)-1)
	named := literal
	if message == "" {
		named = false
	}
	if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
		named = false
	}
	if !named {
		position := recorder_position(file_set, call)
		reg.Non_Literal = append(reg.Non_Literal,
			position+"  inline message is not a nonempty NUL-free literal")
		return
	}
	if !recorder_claim_message(file_set, call, message, reg) {
		return
	}
	var links []Assertion_Registration_Link
	valid := true
	switch kind {
	case "sometimes":
		links = append(links, Assertion_Registration_Link{
			Message: message, Condition: ast_condition_text(file_set, call, 0),
			Kind: ASSERTION_KIND_SOMETIMES,
		})
	case "range":
		links, valid = recorder_collect_assertion_range(
			file_set, call, constants, reg, true, nil, false, 1)
	case "range_holed":
		links, valid = recorder_collect_assertion_range(
			file_set, call, constants, reg, true, nil, true, 1)
	case "enum":
		links, valid = recorder_collect_assertion_enum(
			file_set, call, constants, reg, true, nil, 1)
	}
	if !valid {
		return
	}
	if !recorder_assertion_capacity(file_set, call, reg, links, "inline helper") {
		return
	}
	key := Plan_Key{Namespace: Namespace(message)}
	plan := &Assertion_Plan{Identity: key.Namespace}
	observation := 0
	for ordinal_index, link := range links {
		reg.Forbidden_Properties += link.Forbidden_Properties
		encoded := assertion_registration_key(key, uint8(ordinal_index), link.Message)
		recorder_plan_seed(
			file_set, call, reg, encoded, link.Kind, link.Condition, link.Domain)
		plan.Links = append(plan.Links, Assertion_Plan_Link{
			Ordinal: uint8(ordinal_index), Observation: uint8(observation),
			Kind:  link.Kind,
			Entry: Handle_Entry{Key: encoded},
		})
		if link.Kind == ASSERTION_KIND_SOMETIMES {
			observation++
		}
	}
	reg.Planned_Inline[message] = plan
}

// Names the inline helper a call invokes. An inline helper is a free function, so unlike a fluent
// link it carries no receiver to identify it.
func ast_inline_assertion_kind(
	call *ast.CallExpr, allow_unqualified bool,
) (kind string, matched bool) {
	if len(call.Args) == 0 {
		return "", false
	}
	switch {
	case ast_assertion_named_call(call, "Sometimes", allow_unqualified):
		return "sometimes", true
	case ast_assertion_named_call(call, "Range", allow_unqualified):
		return "range", true
	case ast_assertion_named_call(call, "Range_Holed", allow_unqualified):
		return "range_holed", true
	case ast_assertion_named_call(call, "Enum", allow_unqualified):
		return "enum", true
	}
	return "", false
}

func recorder_register_assertion_always(
	file_set *token.FileSet, call *ast.CallExpr, allow_unqualified bool,
	constants Constant_Scope, reg *Registration,
) {
	plain := ast_assertion_named_call(call, "Always", allow_unqualified)
	recorder := ast_assertion_named_call(call, "Recorder_Always", false)
	if !plain {
		if !recorder {
			return
		}
	}
	if recorder {
		if len(call.Args) != 3 {
			return
		}
	} else if len(call.Args) != 2 {
		return
	}
	condition_index := 0
	if recorder {
		condition_index = 1
	}
	condition := ast_argument(call, condition_index)
	constant, resolved := constant_resolve_boolean(constants, condition)
	if resolved {
		if constant {
			position := recorder_position(file_set, call)
			reg.Constant_Always = append(reg.Constant_Always,
				position+"  Always condition is constant true")
		}
	}
	if reg.Registered_Calls[call.Pos()] {
		return
	}
	reg.Registered_Calls[call.Pos()] = true
	message, literal := ast_string_literal(call, condition_index+1)
	if !literal {
		position := recorder_position(file_set, call)
		reg.Non_Literal = append(reg.Non_Literal,
			position+"  Always message is not a NUL-free literal")
		return
	}
	if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
		position := recorder_position(file_set, call)
		reg.Non_Literal = append(reg.Non_Literal,
			position+"  Always message is not a NUL-free literal")
		return
	}
	if !recorder_claim_message(file_set, call, message, reg) {
		return
	}
	recorder_plan_seed(file_set, call, reg, message, ASSERTION_KIND_ALWAYS,
		ast_condition_text(file_set, call, condition_index), nil)
}

// Resolves the Boolean constants that can disguise a true literal without type information.
func constant_resolve_boolean(
	constants Constant_Scope, expression ast.Expr,
) (value bool, resolved bool) {
	negated := false
	for step_index := 0; step_index < CONSTANT_RESOLUTION_STEPS_MAX; step_index++ {
		switch concrete := expression.(type) {
		case *ast.ParenExpr:
			expression = concrete.X
		case *ast.Ident:
			if concrete.Name == "true" {
				return !negated, true
			}
			if concrete.Name == "false" {
				return negated, true
			}
			declaration, declared := constants.Local[concrete.Name]
			if !declared {
				return false, false
			}
			expression = declaration
		case *ast.SelectorExpr:
			declaration, declared := constant_scope_qualified(constants, concrete)
			if !declared {
				return false, false
			}
			expression = declaration
		case *ast.UnaryExpr:
			if concrete.Op != token.NOT {
				return false, false
			}
			negated = !negated
			expression = concrete.X
		default:
			return false, false
		}
	}
	return false, false
}

func recorder_assertion_ensured_roots(
	function *ast.FuncDecl,
) (roots map[token.Pos]bool) {
	roots = map[token.Pos]bool{}
	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		if ast_assertion_chain_method(call) != "Ensure" {
			return true
		}
		chain, parsed := ast_assertion_chain_from_ensure(call)
		if parsed {
			roots[chain.Root.Pos()] = true
		}
		return true
	})
	return roots
}

func ast_assertion_chain_from_ensure(
	ensure *ast.CallExpr,
) (chain Assertion_Registration_Chain, parsed bool) {
	current, receiver := ast_assertion_chain_receiver(ensure)
	if !receiver {
		return chain, false
	}
	var reversed []*ast.CallExpr
	for step_index := 0; step_index < BUNDLE_EXPANSION_STEPS_MAX; step_index++ {
		method := ast_assertion_chain_method(current)
		if !ast_assertion_link_method(method) {
			chain.Root = current
			break
		}
		reversed = append(reversed, current)
		current, receiver = ast_assertion_chain_receiver(current)
		if !receiver {
			return Assertion_Registration_Chain{}, false
		}
	}
	if chain.Root == nil {
		return Assertion_Registration_Chain{}, false
	}
	chain.Links = make([]*ast.CallExpr, len(reversed))
	for index := range reversed {
		chain.Links[index] = reversed[len(reversed)-1-index]
	}
	return chain, true
}

func ast_assertion_chain_method(call *ast.CallExpr) (method string) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return ""
	}
	return selector.Sel.Name
}

func ast_assertion_chain_receiver(
	call *ast.CallExpr,
) (receiver *ast.CallExpr, found bool) {
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return nil, false
	}
	receiver, found = selector.X.(*ast.CallExpr)
	return receiver, found
}

func ast_assertion_link_method(method string) (link bool) {
	if method == "Sometimes" {
		return true
	}
	return ast_assertion_preset_kind(method) != ""
}

func ast_assertion_preset_kind(method string) (kind string) {
	if ast_assertion_integer_method(method, "Range_Holed_") {
		return "range_holed"
	}
	if ast_assertion_integer_method(method, "Range_") {
		return "range"
	}
	if ast_assertion_integer_method(method, "Enum_3_") {
		return "enum_3"
	}
	if ast_assertion_integer_method(method, "Enum_4_") {
		return "enum_4"
	}
	if ast_assertion_integer_method(method, "Enum_") {
		return "enum"
	}
	return ""
}

func ast_assertion_integer_method(method string, prefix string) (matched bool) {
	if !strings.HasPrefix(method, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(method, prefix)
	switch suffix {
	case "Int", "Int8", "Int16", "Int32", "Int64":
		return true
	case "Uint", "Uint8", "Uint16", "Uint32", "Uint64":
		return true
	}
	return false
}

func ast_assertion_unsigned_method(method string) (unsigned bool) {
	return strings.HasSuffix(method, "_Uint") ||
		strings.HasSuffix(method, "_Uint8") ||
		strings.HasSuffix(method, "_Uint16") ||
		strings.HasSuffix(method, "_Uint32") ||
		strings.HasSuffix(method, "_Uint64")
}

// Both roots gained a leading subject argument, which supplies the chain type.
func ast_assertion_root(call *ast.CallExpr, allow_unqualified bool) (root bool) {
	if ast_assertion_named_call(call, "Tree", allow_unqualified) {
		return len(call.Args) == 2
	}
	if !ast_assertion_named_call(call, "Recorder_Tree", false) {
		return false
	}
	return len(call.Args) == 3
}

func ast_assertion_named_call(
	call *ast.CallExpr, name string, allow_unqualified bool,
) (matched bool) {
	if identifier, is_identifier := call.Fun.(*ast.Ident); is_identifier {
		if !allow_unqualified {
			return false
		}
		return identifier.Name == name
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	if selector.Sel.Name != name {
		return false
	}
	qualifier, is_qualifier := selector.X.(*ast.Ident)
	if !is_qualifier {
		return false
	}
	return qualifier.Name == "invariant"
}

func recorder_seed_assertion_root(
	file_set *token.FileSet, chain Assertion_Registration_Chain, function *ast.FuncDecl,
	package_path string, package_types map[string]ast.Expr, constants Constant_Scope,
	reg *Registration, allow_unqualified bool,
) {
	if !ast_assertion_root(chain.Root, allow_unqualified) {
		recorder_invalid_chain(file_set, chain.Root, reg,
			"Ensure must terminate a Tree root")
		return
	}
	namespace_index := len(chain.Root.Args) - 1
	namespace, literal := ast_string_literal(chain.Root, namespace_index)
	if !literal {
		recorder_invalid_assertion_namespace(file_set, chain.Root, reg)
		return
	}
	if namespace == "" {
		recorder_invalid_assertion_namespace(file_set, chain.Root, reg)
		return
	}
	if strings.Contains(namespace, ELEMENT_MESSAGE_SEPARATOR) {
		recorder_invalid_assertion_namespace(file_set, chain.Root, reg)
		return
	}
	subject, named := ast_assertion_chain_subject_type(chain.Root, function, package_types)
	if !named {
		recorder_invalid_subject(file_set, chain.Root, reg)
		return
	}
	owner, claimed := reg.Root_Namespace_Owner[Namespace(namespace)]
	if claimed {
		if owner != chain.Root.Pos() {
			recorder_duplicate_namespace(file_set, chain.Root.Pos(), namespace, reg)
			return
		}
	}
	reg.Root_Namespace_Owner[Namespace(namespace)] = chain.Root.Pos()
	recorder_seed_assertion_chain(
		file_set, chain,
		Plan_Key{Namespace: Namespace(namespace), Package: package_path, Type: subject},
		nil, constants, reg, true)
}

// Names the type of a chain's subject argument by resolving that identifier against the enclosing
// function's parameters. A chain outside a bundle still has a subject, so it still has an identity.
// named is false when the argument is not a plain identifier, when it names no parameter, or when
// the parameter's type is one reflect cannot name uniquely.
func ast_assertion_chain_subject_type(
	root *ast.CallExpr, function *ast.FuncDecl, package_types map[string]ast.Expr,
) (name string, named bool) {
	// Assertions(subject, namespace) and Recorder_Tree(recorder, subject, namespace) both
	// put the subject immediately before the namespace.
	if len(root.Args) < 2 {
		return "", false
	}
	subject := root.Args[len(root.Args)-2]
	// A conversion names its own type, so it needs no parameter lookup. Go infers the type
	// parameter from the converted value, thus reflect reports exactly this name.
	if conversion, is_call := subject.(*ast.CallExpr); is_call {
		return ast_defined_type_name(conversion.Fun, function, package_types)
	}
	argument, is_identifier := subject.(*ast.Ident)
	if !is_identifier {
		return "", false
	}
	if function == nil {
		return "", false
	}
	if function.Type.Params == nil {
		return "", false
	}
	for _, field := range function.Type.Params.List {
		for _, parameter := range field.Names {
			if parameter.Name != argument.Name {
				continue
			}
			return ast_defined_type_name(field.Type, function, package_types)
		}
	}
	return "", false
}

// Gives the bare name of a defined package-level type expression, matching reflect.Type.Name.
// named is false for a builtin, an unnamed composite, a type parameter, a generic instantiation,
// and a type imported from another package.
func ast_defined_type_name(
	expression ast.Expr, function *ast.FuncDecl, package_types map[string]ast.Expr,
) (name string, named bool) {
	if pointer, is_pointer := expression.(*ast.StarExpr); is_pointer {
		expression = pointer.X
	}
	identifier, is_identifier := expression.(*ast.Ident)
	if !is_identifier {
		return "", false
	}
	if recorder_is_builtin_type_name(identifier.Name) {
		return "", false
	}
	if ast_type_parameter_named(function, identifier.Name) {
		return "", false
	}
	if _, declared := package_types[identifier.Name]; !declared {
		return "", false
	}
	return identifier.Name, true
}

func recorder_invalid_assertion_namespace(
	file_set *token.FileSet, root *ast.CallExpr, reg *Registration,
) {
	reg.Invalid_Identifier = append(reg.Invalid_Identifier,
		recorder_position(file_set, root)+
			"  Assertions namespace is not a nonempty NUL-free literal")
}

func recorder_validate_assertion_template(
	file_set *token.FileSet, chain Assertion_Registration_Chain,
	parameter string, constants Constant_Scope, reg *Registration,
) {
	if !ast_assertion_root(chain.Root, true) {
		recorder_invalid_chain(file_set, chain.Root, reg,
			"bundle Ensure must terminate a Tree root")
		return
	}
	argument := ast_argument(chain.Root, len(chain.Root.Args)-1)
	identifier, is_identifier := argument.(*ast.Ident)
	if !is_identifier {
		recorder_invalid_chain(file_set, chain.Root, reg,
			"bundle Tree root must use its trailing namespace parameter")
		return
	}
	if identifier.Name != parameter {
		recorder_invalid_chain(file_set, chain.Root, reg,
			"bundle Tree root must use its trailing namespace parameter")
		return
	}
	recorder_collect_assertion_links(file_set, chain, constants, reg, true)
}

func recorder_seed_assertion_chain(
	file_set *token.FileSet, chain Assertion_Registration_Chain, key Plan_Key,
	owner_path []token.Pos, constants Constant_Scope, reg *Registration, diagnose bool,
) {
	resolved_owner := recorder_namespace_owner_append(owner_path, chain.Root.Pos())
	// SECURITY: One key belongs to one complete static invocation path. A different path would
	// merge independent value streams and let one invocation satisfy another invocation's
	// coverage obligations. Re-expansion of the identical path must stay idempotent.
	registered_owner, seen := reg.Namespace_Owner_Path[key]
	if seen {
		if recorder_namespace_owner_equal(registered_owner, resolved_owner) {
			return
		}
		position := chain.Root.Pos()
		if len(owner_path) != 0 {
			position = owner_path[len(owner_path)-1]
		}
		recorder_duplicate_namespace(file_set, position, string(key.Namespace), reg)
		return
	}
	reg.Namespace_Owner_Path[key] = resolved_owner
	links, valid := recorder_collect_assertion_links(
		file_set, chain, constants, reg, diagnose)
	if !valid {
		return
	}
	if len(links) == 0 {
		recorder_invalid_chain(file_set, chain.Root, reg, "Tree chain has no links")
		return
	}
	if !recorder_assertion_capacity(file_set, chain.Root, reg, links, "Tree") {
		return
	}
	plan := &Assertion_Plan{Identity: key.Namespace}
	observation := 0
	for ordinal_index, link := range links {
		reg.Forbidden_Properties += link.Forbidden_Properties
		encoded := assertion_registration_key(key, uint8(ordinal_index), link.Message)
		recorder_plan_seed(file_set, chain.Links[0], reg, encoded,
			link.Kind, link.Condition, link.Domain)
		plan.Links = append(plan.Links, Assertion_Plan_Link{
			Ordinal: uint8(ordinal_index), Observation: uint8(observation),
			Kind:  link.Kind,
			Entry: Handle_Entry{Key: encoded},
		})
		if link.Kind == ASSERTION_KIND_SOMETIMES {
			observation++
		}
	}
	reg.Planned_Assertions[key] = plan
}

// Claims one bare message for an Always guard or an inline helper. Reports a second claim, which
// would put two unrelated assertions under one name in the gap report.
func recorder_claim_message(
	file_set *token.FileSet, node ast.Node, message string, reg *Registration,
) (claimed bool) {
	if reg.Message_Owner[message] {
		reg.Collision = append(reg.Collision,
			recorder_position(file_set, node)+
				"  duplicate message: "+strconv.Quote(message))
		return false
	}
	reg.Message_Owner[message] = true
	return true
}

// Reports one namespace claimed by a second owner. Both the root claim and the chain-owner guard
// report the same fact, thus the text lives at one site.
func recorder_duplicate_namespace(
	file_set *token.FileSet, position token.Pos, namespace string, reg *Registration,
) {
	reg.Collision = append(reg.Collision,
		recorder_token_position(file_set, position)+
			"  duplicate namespace: "+strconv.Quote(namespace))
}

func recorder_namespace_owner_append(owner []token.Pos, position token.Pos) (next []token.Pos) {
	next = make([]token.Pos, len(owner)+1)
	copy(next, owner)
	next[len(owner)] = position
	return next
}

func recorder_namespace_owner_equal(first []token.Pos, second []token.Pos) (equal bool) {
	if len(first) != len(second) {
		return false
	}
	for position_index := range first {
		if first[position_index] != second[position_index] {
			return false
		}
	}
	return true
}

// MERGE_READ_BYTES is the fuzz-record read window. A stored record is one Base64 key with its
// marker, so this holds many records per read and truncates none that the merge can resolve.
const MERGE_READ_BYTES = 4096

// BRANCH_TABLE_COLUMNS is the branch gap table's width: assertion, type, link, missing, property,
// source. The package stays out of every table, because an import path repeats on each row and the
// subject type alone separates the bundles of one tree.
const BRANCH_TABLE_COLUMNS = 6

// DOMAIN_TABLE_COLUMNS is the Range domain table's width: assertion, type, link, declared,
// observed, count, and source. It carries no polarity, because a domain names no absent branch.
const DOMAIN_TABLE_COLUMNS = 7

// REACHABILITY_TABLE_COLUMNS is the reachability gap table's width: assertion, type, and source.
// Its section names the absent obligation, so it needs no polarity or property column.
const REACHABILITY_TABLE_COLUMNS = 3

// ASSERTION_KEY_PARTS is how many separated elements assertion_registration_key writes: namespace,
// package, type, ordinal, and message.
const ASSERTION_KEY_PARTS = 5

// Joins the identity into the persisted tracker key. Registration builds this once per link, thus
// the concatenation never reaches the record path. The runtime resolves a plan through Plan_Key,
// which joins nothing.
func assertion_registration_key(
	key Plan_Key, ordinal uint8, message string,
) (encoded string) {
	return string(key.Namespace) + ELEMENT_MESSAGE_SEPARATOR + key.Package +
		ELEMENT_MESSAGE_SEPARATOR + key.Type + ELEMENT_MESSAGE_SEPARATOR +
		strconv.Itoa(int(ordinal)) + ELEMENT_MESSAGE_SEPARATOR + message
}

func recorder_collect_assertion_links(
	file_set *token.FileSet, chain Assertion_Registration_Chain,
	constants Constant_Scope, reg *Registration, diagnose bool,
) (links []Assertion_Registration_Link, valid bool) {
	valid = true
	for _, call := range chain.Links {
		method := ast_assertion_chain_method(call)
		if method == "Sometimes" {
			message, literal := ast_string_literal(call, 1)
			if !literal {
				recorder_invalid_sometimes_message(file_set, call, reg, diagnose)
				valid = false
				continue
			}
			if strings.Contains(message, ELEMENT_MESSAGE_SEPARATOR) {
				recorder_invalid_sometimes_message(file_set, call, reg, diagnose)
				valid = false
				continue
			}
			links = append(links, Assertion_Registration_Link{
				Message: message, Condition: ast_condition_text(file_set, call, 0),
				Kind: ASSERTION_KIND_SOMETIMES,
			})
		} else if ast_assertion_preset_kind(method) == "range" {
			var preset_valid bool
			links, preset_valid = recorder_collect_assertion_range(
				file_set, call, constants, reg, diagnose, links, false, 0)
			valid = valid && preset_valid
		} else if ast_assertion_preset_kind(method) == "range_holed" {
			var preset_valid bool
			links, preset_valid = recorder_collect_assertion_range(
				file_set, call, constants, reg, diagnose, links, true, 0)
			valid = valid && preset_valid
		} else if strings.HasPrefix(ast_assertion_preset_kind(method), "enum") {
			var preset_valid bool
			links, preset_valid = recorder_collect_assertion_enum(
				file_set, call, constants, reg, diagnose, links, 0)
			valid = valid && preset_valid
		} else {
			recorder_invalid_chain(file_set, call, reg,
				"Tree contains an unknown link")
			valid = false
		}
		if len(links) > ASSERTION_LINKS_MAX {
			recorder_invalid_chain(file_set, call, reg,
				"Tree exceeds 127 links")
			return nil, false
		}
	}
	return links, valid
}

func recorder_invalid_sometimes_message(
	file_set *token.FileSet, call *ast.CallExpr, reg *Registration, diagnose bool,
) {
	if !diagnose {
		return
	}
	position := recorder_position(file_set, call)
	reg.Non_Literal = append(reg.Non_Literal,
		position+"  Sometimes message is not a NUL-free literal")
}

// Extra_arguments counts the arguments after the preset's own operands. A fluent link has none.
// An inline helper has its message, which must stay out of the hole slots and the arity check.
func recorder_collect_assertion_range(
	file_set *token.FileSet, call *ast.CallExpr, constants Constant_Scope,
	reg *Registration, diagnose bool, links []Assertion_Registration_Link,
	holed bool, extra_arguments int,
) (expanded []Assertion_Registration_Link, valid bool) {
	if !recorder_assertion_range_arity(file_set, call, reg, diagnose, holed, extra_arguments) {
		return links, false
	}
	minimum, minimum_ok := constant_resolve(constants, call.Args[1])
	maximum, maximum_ok := constant_resolve(constants, call.Args[2])
	if !minimum_ok {
		return links, recorder_unresolved_preset(file_set, call, reg, diagnose,
			"Range bounds are not statically resolvable")
	}
	if !maximum_ok {
		return links, recorder_unresolved_preset(file_set, call, reg, diagnose,
			"Range bounds are not statically resolvable")
	}
	if integer_compare(minimum, maximum) > 0 {
		return links, recorder_invalid_preset(file_set, call, reg, diagnose,
			"Range minimum exceeds maximum")
	}
	holes, holes_valid := recorder_assertion_range_holes(
		file_set, call, constants, reg, diagnose, minimum, maximum, extra_arguments)
	if !holes_valid {
		return links, false
	}
	if !recorder_assertion_range_cardinality(
		file_set, call, reg, diagnose, minimum, maximum, holes) {
		return links, false
	}
	condition := ast_condition_text(file_set, call, 0)
	domain := &Assertion_Domain{
		Declared_Minimum: minimum, Declared_Maximum: maximum,
		Unsigned: ast_assertion_unsigned_method(ast_assertion_chain_method(call)),
	}
	expanded = append(links,
		Assertion_Registration_Link{Message: RANGE_GUARD_MINIMUM, Condition: condition,
			Kind: ASSERTION_KIND_ALWAYS, Forbidden_Properties: len(holes),
			Domain: domain},
		Assertion_Registration_Link{Message: RANGE_GUARD_MAXIMUM, Condition: condition,
			Kind: ASSERTION_KIND_ALWAYS})
	if minimum == maximum {
		return expanded, true
	}
	expanded = append(expanded,
		Assertion_Registration_Link{Message: RANGE_MESSAGE_MINIMUM, Condition: condition,
			Kind: ASSERTION_KIND_SOMETIMES},
		Assertion_Registration_Link{Message: RANGE_MESSAGE_MAXIMUM, Condition: condition,
			Kind: ASSERTION_KIND_SOMETIMES})
	expanded = recorder_append_assertion_range_candidate(
		expanded, Integer_Value{}, RANGE_MESSAGE_ZERO, condition, minimum, maximum, holes)
	expanded = recorder_append_assertion_range_candidate(
		expanded, Integer_Value{Magnitude: 1}, RANGE_MESSAGE_ONE,
		condition, minimum, maximum, holes)
	expanded = recorder_append_assertion_range_candidate(
		expanded, Integer_Value{Magnitude: 2}, RANGE_MESSAGE_TWO,
		condition, minimum, maximum, holes)
	if !ast_assertion_unsigned_method(ast_assertion_chain_method(call)) {
		expanded = recorder_append_assertion_range_candidate(
			expanded, Integer_Value{Magnitude: 1, Negative: true},
			RANGE_MESSAGE_NEGATIVE_ONE, condition, minimum, maximum, holes)
	}
	return expanded, true
}

func recorder_assertion_range_arity(
	file_set *token.FileSet, call *ast.CallExpr, reg *Registration,
	diagnose bool, holed bool, extra_arguments int,
) (valid bool) {
	if !holed {
		if len(call.Args) == 3+extra_arguments {
			return true
		}
		return recorder_invalid_preset(file_set, call, reg, diagnose,
			"Range needs exactly value, minimum, and maximum")
	}
	// An inline helper is generic, thus its name carries no width suffix and this test is
	// already false for it. It takes four slots for every width and leans on the duplicate-fill
	// rule instead.
	hole_count := 4
	if ast_assertion_unsigned_method(ast_assertion_chain_method(call)) {
		hole_count = 3
	}
	if len(call.Args) == 3+hole_count+extra_arguments {
		return true
	}
	message := "Range_Holed needs exactly four hole slots"
	if hole_count == 3 {
		message = "Range_Holed needs exactly three hole slots"
	}
	return recorder_invalid_preset(file_set, call, reg, diagnose, message)
}

func recorder_assertion_range_holes(
	file_set *token.FileSet, call *ast.CallExpr, constants Constant_Scope,
	reg *Registration, diagnose bool, minimum Integer_Value, maximum Integer_Value,
	extra_arguments int,
) (holes []Integer_Value, valid bool) {
	padding := false
	for hole_index, argument := range call.Args[3 : len(call.Args)-extra_arguments] {
		hole, resolved := constant_resolve(constants, argument)
		if !resolved {
			return nil, recorder_unresolved_preset(file_set, call, reg, diagnose,
				"Range exclusions are not statically resolvable")
		}
		inside := integer_compare(hole, minimum) > 0
		if integer_compare(hole, maximum) >= 0 {
			inside = false
		}
		if !inside {
			return nil, recorder_invalid_preset(file_set, call, reg, diagnose,
				"Range exclusion is not strictly inside the interval")
		}
		if hole_index == 0 {
			holes = append(holes, hole)
			continue
		}
		comparison := integer_compare(hole, holes[len(holes)-1])
		if comparison < 0 {
			return nil, recorder_invalid_preset(file_set, call, reg, diagnose,
				"Range holes must be ascending")
		}
		if comparison == 0 {
			padding = true
			continue
		}
		if padding {
			return nil, recorder_invalid_preset(file_set, call, reg, diagnose,
				"Range hole duplicates must be final-hole padding")
		}
		holes = append(holes, hole)
	}
	return holes, true
}

func recorder_assertion_range_cardinality(
	file_set *token.FileSet, call *ast.CallExpr, reg *Registration, diagnose bool,
	minimum Integer_Value, maximum Integer_Value, holes []Integer_Value,
) (valid bool) {
	legal_count := assertion_range_legal_count(minimum, maximum, holes)
	if legal_count > RANGE_ENUM_CARDINALITY_MAX {
		return true
	}
	message := "Range covers 1 legal value; use Always instead"
	if legal_count > 1 {
		method := ast_assertion_chain_method(call)
		suffix := strings.TrimPrefix(method, "Range_")
		if strings.HasPrefix(method, "Range_Holed_") {
			suffix = strings.TrimPrefix(method, "Range_Holed_")
		}
		enum := "Enum_" + suffix
		if legal_count > 2 {
			enum = "Enum_" + strconv.Itoa(legal_count) + "_" + suffix
		}
		message = "Range covers " + strconv.Itoa(legal_count) +
			" legal values; use " + enum + " instead"
	}
	return recorder_invalid_preset(file_set, call, reg, diagnose, message)
}

// The count stops after the Enum boundary, so it never increments the maximum integer and never
// walks a large domain. At most four holes occur before the fifth legal value.
func assertion_range_legal_count(
	minimum Integer_Value, maximum Integer_Value, holes []Integer_Value,
) (legal_count int) {
	value := minimum
	for step_index := 0; step_index <= RANGE_ENUM_CARDINALITY_MAX+len(holes); step_index++ {
		if !assertion_integer_contains(holes, value) {
			legal_count++
			if legal_count > RANGE_ENUM_CARDINALITY_MAX {
				return legal_count
			}
		}
		if value == maximum {
			return legal_count
		}
		value = integer_successor(value)
	}
	return RANGE_ENUM_CARDINALITY_MAX + 1
}

func integer_successor(value Integer_Value) (next Integer_Value) {
	if value.Negative {
		if value.Magnitude == 1 {
			return Integer_Value{}
		}
		return Integer_Value{Magnitude: value.Magnitude - 1, Negative: true}
	}
	return Integer_Value{Magnitude: value.Magnitude + 1}
}

func recorder_append_assertion_range_candidate(
	links []Assertion_Registration_Link, value Integer_Value, message string,
	condition string, minimum Integer_Value, maximum Integer_Value, holes []Integer_Value,
) (expanded []Assertion_Registration_Link) {
	if integer_compare(value, minimum) <= 0 {
		return links
	}
	if integer_compare(value, maximum) >= 0 {
		return links
	}
	if assertion_integer_contains(holes, value) {
		return links
	}
	return append(links, Assertion_Registration_Link{
		Message: message, Condition: condition, Kind: ASSERTION_KIND_SOMETIMES,
	})
}

// Extra_arguments counts the arguments after the members. An inline helper has one, its message.
func recorder_collect_assertion_enum(
	file_set *token.FileSet, call *ast.CallExpr, constants Constant_Scope,
	reg *Registration, diagnose bool, links []Assertion_Registration_Link, extra_arguments int,
) (expanded []Assertion_Registration_Link, valid bool) {
	method := ast_assertion_chain_method(call)
	member_count := 2
	if ast_assertion_preset_kind(method) == "enum_3" {
		member_count = 3
	}
	if ast_assertion_preset_kind(method) == "enum_4" {
		member_count = 4
	}
	if len(call.Args) != member_count+1+extra_arguments {
		message := "Enum needs exactly two members"
		if member_count == 3 {
			message = "Enum_3 needs exactly three members"
		}
		if member_count == 4 {
			message = "Enum_4 needs exactly four members"
		}
		return links, recorder_invalid_preset(file_set, call, reg, diagnose,
			message)
	}
	var members []Integer_Value
	for _, argument := range call.Args[1 : len(call.Args)-extra_arguments] {
		member, resolved := constant_resolve(constants, argument)
		if !resolved {
			return links, recorder_unresolved_preset(file_set, call, reg, diagnose,
				"Enum members are not statically resolvable")
		}
		if len(members) != 0 {
			comparison := integer_compare(member, members[len(members)-1])
			if comparison == 0 {
				return links, recorder_invalid_preset(file_set, call, reg, diagnose,
					"Enum members must be exactly distinct")
			}
			if comparison < 0 {
				return links, recorder_invalid_preset(file_set, call, reg, diagnose,
					"Enum members must be ascending")
			}
		}
		members = append(members, member)
	}
	condition := ast_condition_text(file_set, call, 0)
	expanded = append(links, Assertion_Registration_Link{
		Message: ENUM_GUARD_MEMBER, Condition: condition, Kind: ASSERTION_KIND_ALWAYS,
	})
	for _, member := range members {
		expanded = append(expanded, Assertion_Registration_Link{
			Message:   enum_member_message(integer_text(member)),
			Condition: condition, Kind: ASSERTION_KIND_SOMETIMES,
		})
	}
	return expanded, true
}

// Holds one chain to both capacities. Links fill the packed ordinal counter, and axes fill the
// smaller observation bitmap, thus a chain can pass one bound and fail the other.
func recorder_assertion_capacity(
	file_set *token.FileSet, node ast.Node, reg *Registration,
	links []Assertion_Registration_Link, subject string,
) (valid bool) {
	if len(links) > ASSERTION_LINKS_MAX {
		recorder_invalid_chain(file_set, node, reg, subject+" exceeds 127 links")
		return false
	}
	if assertion_axis_count(links) > ASSERTION_OBSERVATIONS_MAX {
		recorder_invalid_chain(file_set, node, reg, subject+" exceeds 109 axes")
		return false
	}
	return true
}

// Counts the links that spend an observation bit. A guard holds one outcome, thus it needs no bit
// and a chain of guards costs the smaller capacity nothing.
func assertion_axis_count(links []Assertion_Registration_Link) (axes int) {
	for _, link := range links {
		if link.Kind == ASSERTION_KIND_SOMETIMES {
			axes++
		}
	}
	return axes
}

func assertion_integer_contains(
	values []Integer_Value, wanted Integer_Value,
) (contains bool) {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func recorder_invalid_preset(
	file_set *token.FileSet, call *ast.CallExpr, reg *Registration,
	diagnose bool, message string,
) (valid bool) {
	if diagnose {
		reg.Invalid_Bound = append(reg.Invalid_Bound,
			recorder_position(file_set, call)+"  "+message)
	}
	return false
}

func recorder_unresolved_preset(
	file_set *token.FileSet, call *ast.CallExpr, reg *Registration,
	diagnose bool, message string,
) (valid bool) {
	if diagnose {
		reg.Unresolved_Bound = append(reg.Unresolved_Bound,
			recorder_position(file_set, call)+"  "+message)
	}
	return false
}

func recorder_invalid_chain(
	file_set *token.FileSet, node ast.Node, reg *Registration, message string,
) {
	reg.Invalid_Chain = append(reg.Invalid_Chain,
		recorder_position(file_set, node)+"  "+message)
}

func ast_assertion_namespace_parameter(function *ast.FuncDecl) (name string) {
	if function.Type.Params == nil {
		return ""
	}
	if len(function.Type.Params.List) == 0 {
		return ""
	}
	last := function.Type.Params.List[len(function.Type.Params.List)-1]
	if len(last.Names) == 0 {
		return ""
	}
	if !ast_assertion_namespace_type(last.Type) {
		return ""
	}
	return last.Names[len(last.Names)-1].Name
}

func ast_assertion_namespace_type(expression ast.Expr) (matched bool) {
	if identifier, is_identifier := expression.(*ast.Ident); is_identifier {
		if identifier.Name == "string" {
			return true
		}
		return identifier.Name == "Namespace"
	}
	selector, is_selector := expression.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	return selector.Sel.Name == "Namespace"
}

func recorder_register_assertion_bundle_call(
	file_set *token.FileSet, call *ast.CallExpr, imports map[string]string,
	index *Bundle_Index, reg *Registration,
) {
	if len(call.Args) == 0 {
		reg.Invalid_Identifier = append(reg.Invalid_Identifier,
			recorder_position(file_set, call)+"  bundle call has no namespace")
		return
	}
	namespace, literal := ast_string_literal(call, len(call.Args)-1)
	if !literal {
		recorder_invalid_bundle_namespace(file_set, call, reg)
		return
	}
	if namespace == "" {
		recorder_invalid_bundle_namespace(file_set, call, reg)
		return
	}
	if strings.Contains(namespace, ELEMENT_MESSAGE_SEPARATOR) {
		recorder_invalid_bundle_namespace(file_set, call, reg)
		return
	}
	function, found := bundle_index_lookup(index, imports, index.Same_Set, call)
	if !found {
		reg.Unresolved = append(reg.Unresolved, recorder_unresolved_line(file_set, call))
		return
	}
	recorder_register_assertion_bundle_instance(
		file_set, function, Namespace(namespace), []token.Pos{call.Pos()}, index, reg)
}

// Reports one chain whose subject reflect cannot name uniquely, joining the recorder_invalid_*
// family so the message text lives at one site.
func recorder_invalid_subject(
	file_set *token.FileSet, root *ast.CallExpr, reg *Registration,
) {
	reg.Invalid_Subject = append(reg.Invalid_Subject,
		recorder_position(file_set, root)+
			"  assertion subject is not a defined package-level type")
}

func recorder_invalid_bundle_namespace(
	file_set *token.FileSet, call *ast.CallExpr, reg *Registration,
) {
	reg.Invalid_Identifier = append(reg.Invalid_Identifier,
		recorder_position(file_set, call)+"  bundle namespace is not a nonempty literal")
}

func recorder_register_assertion_bundle_instance(
	file_set *token.FileSet, function Indexed_Function, namespace Namespace,
	owner_path []token.Pos, index *Bundle_Index, reg *Registration,
) {
	functions := []Indexed_Function{function}
	namespaces := []Namespace{namespace}
	owner_paths := [][]token.Pos{owner_path}
	paths := [][]string{nil}
	for step_index := 0; step_index < BUNDLE_EXPANSION_STEPS_MAX; step_index++ {
		if step_index == len(functions) {
			return
		}
		current := functions[step_index]
		current_namespace := namespaces[step_index]
		current_owner_path := owner_paths[step_index]
		current_path := paths[step_index]
		if recorder_assertion_bundle_cycle(file_set, current, current_path, reg) {
			continue
		}
		if recorder_assertion_subject_repeat(
			file_set, current, current_namespace, current_owner_path, reg) {
			continue
		}
		name := current.Package + "." + current.Declaration.Name.Name
		next_path := append(append([]string{}, current_path...), name)
		enqueue := func(
			nested Indexed_Function, nested_namespace Namespace,
			nested_owner_path []token.Pos,
		) {
			functions = append(functions, nested)
			namespaces = append(namespaces, nested_namespace)
			owner_paths = append(owner_paths, nested_owner_path)
			paths = append(paths, next_path)
		}
		recorder_register_assertion_bundle_body(
			file_set, current, current_namespace, current_owner_path,
			index, reg, enqueue)
	}
	position := recorder_position(file_set, function.Declaration)
	reg.Invalid_Chain = append(reg.Invalid_Chain,
		position+"  helper expansion exceeds 4096 steps")
}

// Compares the package-qualified bundle, not the bare name. Two packages can each declare one
// bundle name, and a bare comparison reports those two as a cycle when one calls the other.
func recorder_assertion_bundle_cycle(
	file_set *token.FileSet, function Indexed_Function, path []string, reg *Registration,
) (cycle bool) {
	name := function.Package + "." + function.Declaration.Name.Name
	for _, ancestor := range path {
		if ancestor != name {
			continue
		}
		position := recorder_position(file_set, function.Declaration)
		reg.Cycle = append(reg.Cycle,
			position+"  bundle cycle: "+strings.Join(append(path, name), " -> "))
		return true
	}
	return false
}

// Rejects one subject type that occurs twice in the expansion of one root. The type stands in for
// the call path, so a second occurrence gives two paths one identity, and the coverage of one
// callsite would satisfy the obligations of the other. A diamond and a repeated sibling both land
// here, and neither is reachable by the ancestor check above.
func recorder_assertion_subject_repeat(
	file_set *token.FileSet, function Indexed_Function, namespace Namespace,
	owner_path []token.Pos, reg *Registration,
) (repeated bool) {
	subject, named := ast_assertion_subject_type(function.Declaration, function.Package_Types)
	if !named {
		return false
	}
	key := Plan_Key{Namespace: namespace, Package: function.Package, Type: subject}
	registered, seen := reg.Subject_Owner[key]
	if !seen {
		reg.Subject_Owner[key] = owner_path
		return false
	}
	if recorder_namespace_owner_equal(registered, owner_path) {
		return false
	}
	position := owner_path[len(owner_path)-1]
	reg.Repeated_Subject = append(reg.Repeated_Subject,
		recorder_token_position(file_set, position)+
			"  repeated subject type "+strconv.Quote(subject)+" under namespace "+
			strconv.Quote(string(namespace)))
	return true
}

func recorder_register_assertion_bundle_body(
	file_set *token.FileSet, function Indexed_Function, namespace Namespace,
	owner_path []token.Pos, index *Bundle_Index, reg *Registration,
	enqueue func(Indexed_Function, Namespace, []token.Pos),
) {
	parameter := ast_assertion_namespace_parameter(function.Declaration)
	ast.Inspect(function.Declaration.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		// An eager guard in this body runs whenever the body runs, thus it owes coverage
		// even when its own package is never analyzed directly.
		recorder_register_assertion_always(
			file_set, call, function.Is_Sugar,
			indexed_function_constants(function, index), reg)
		recorder_register_inline_assertion(
			file_set, call, function.Is_Sugar,
			indexed_function_constants(function, index), reg)
		if ast_assertion_chain_method(call) == "Ensure" {
			chain, parsed := ast_assertion_chain_from_ensure(call)
			if !parsed {
				return false
			}
			// Read the chain's own subject argument, not the declaration's
			// parameter. The runtime keys on whatever the call passes, thus
			// registration must read the same expression or the two sides build
			// different keys.
			subject, named := ast_assertion_chain_subject_type(
				chain.Root, function.Declaration, function.Package_Types)
			if !named {
				recorder_invalid_subject(file_set, chain.Root, reg)
				return false
			}
			recorder_seed_assertion_chain(
				file_set, chain,
				Plan_Key{
					Namespace: namespace, Package: function.Package,
					Type: subject,
				},
				owner_path, indexed_function_constants(function, index),
				reg, false)
			return false
		}
		if !ast_is_invariants_name(ast_callee_name(call)) {
			return true
		}
		nested_namespace, nested_owner_path, resolved :=
			recorder_assertion_nested_namespace(call, parameter, namespace, owner_path)
		if !resolved {
			position := recorder_position(file_set, call)
			reg.Invalid_Identifier = append(reg.Invalid_Identifier,
				position+"  nested bundle namespace is not literal or forwarded")
			return true
		}
		nested, found := bundle_index_lookup(
			index, function.Imports, function.Package_Functions, call)
		if !found {
			unresolved := recorder_unresolved_line(file_set, call)
			reg.Unresolved = append(reg.Unresolved, unresolved)
			return true
		}
		enqueue(nested, nested_namespace, nested_owner_path)
		return true
	})
}

func recorder_assertion_nested_namespace(
	call *ast.CallExpr, parameter string, inherited Namespace, inherited_owner []token.Pos,
) (namespace Namespace, owner_path []token.Pos, resolved bool) {
	if len(call.Args) == 0 {
		return "", nil, false
	}
	index := len(call.Args) - 1
	if literal, is_literal := ast_string_literal(call, index); is_literal {
		if literal == "" {
			return "", nil, false
		}
		if strings.Contains(literal, ELEMENT_MESSAGE_SEPARATOR) {
			return "", nil, false
		}
		return Namespace(literal), []token.Pos{call.Pos()}, true
	}
	identifier, is_identifier := call.Args[index].(*ast.Ident)
	if is_identifier {
		if identifier.Name == parameter {
			owner_path = recorder_namespace_owner_append(inherited_owner, call.Pos())
			return inherited, owner_path, true
		}
	}
	return "", nil, false
}
