// Package lint is the monorepo's static checker. It enforces the
// workspace organization doctrine (library tier vs. composition tier,
// binary vs. shared library), Tiger-style local conventions
// (snake_case / Ada_Case naming, no compound ifs, no recursion, …),
// and a small set of cross-file rules (package fragmentation, git
// history hygiene, package/exported-identifier documentation).
package lint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const line_chars_max = 100
const tab_width = 8

// Hi bounds for Distinct_Boundary axes on string lengths. Each constant
// encodes the realistic upper bound for the semantic domain that the
// string represents — Distinct_Boundary fatals on inputs exceeding Hi,
// so the bound must be wide enough for real inputs AND tight enough that
// a test can satisfy the Hi-equals-X tuple. Constants are named for the
// domain so reading the assertion at a call site makes the bound obvious.

// Identifier_Chars_Max caps Go identifier lengths the linter processes.
// 128 chars is wider than any identifier representable on a line_chars_max
// (140) source line after the surrounding syntax; the repo's longest
// production identifier is 83 chars.
const Identifier_Chars_Max = 128

// Invariant_helper_name_chars_max caps the longest invariant.X helper name
// the linter recognises; "Recorder_Is_Distinct_Boundary" is the longest
// (29 chars). The constant ALSO serves as a sanity bound on helper_name
// strings passed between extractor helpers, all of which receive non-
// empty names (callers gate on `helper_name == "" return` so the boundary
// is paired with `Always(helper_name != "", ...)`).
const invariant_helper_name_chars_max = 29

// Invariant_helper_name_chars_min is the shortest recognised invariant.X
// helper name: "Always" (6 chars). Paired with the max as the Lo/Hi of
// helper_name Distinct_Boundary axes in extract_nil_comparison_path /
// extract_eq_nil_path / nil_predicate_index / nil_allows_neq.
const invariant_helper_name_chars_min = 6

// Credit_kind_chars_min / credit_kind_chars_max cap the bare_composable_
// table values: "bool" (4) and "boundary_float" (14). Tightening to the
// exact table range makes both endpoints reachable by tests that exercise
// any Distinct_Boundary or Always/Sometimes credit shape.
const credit_kind_chars_min = 4
const credit_kind_chars_max = 14

// Diagnostic_source_chars_min / diagnostic_source_chars_max cap the
// `source` string in the diagnostic-builder helpers: "param" (5),
// "param_defer" (11), or "named_return" (12). The literal values come from
// the requirement emit branches in collect_requirements and the validate
// loop.
const diagnostic_source_chars_min = 5
const diagnostic_source_chars_max = 12

// Function_label_chars_min caps the shortest function_label string: a
// single-character function name like `f`. Paired with
// Identifier_Chars_Max as Hi.
const function_label_chars_min = 1

// Non_empty_min is the universal Lo for length axes on inputs the caller
// guarantees non-empty (validated by a callsite check or an Always(s != "")
// invariant). Distinct_Boundary requires Lo < Hi, so empty inputs need their
// own pre-gate; this constant anchors the "≥1 character" bucket for
// non-empty-string and non-empty-slice axes.
const non_empty_min = 1

// Split_suggestion_chars_min caps the shortest non-trivial split suggestion
// returned by suggest. The shortest case is a 2-char Ada_Case input ("aB")
// split into "a_B" — three characters including the inserted underscore.
const split_suggestion_chars_min = 3

// Naming_style_chars_min / naming_style_chars_max bound the length of the
// `Want` field on suggest_input. Callers pass exactly one of "Ada_Case" (8)
// or "snake_case" (10) — the only two style words the casing checks know.
const naming_style_chars_min = 8
const naming_style_chars_max = 10

// Stream_check_name_chars_min / stream_check_name_chars_max bound the
// `Name` field on check_function_stream constructors. Shortest is "symlink"
// (7); longest is "markdown-line-length" (20).
const stream_check_name_chars_min = 7
const stream_check_name_chars_max = 20

// Stack_with_body_frame_min is the Lo for walker stacks that the function
// guarantees have appended at least one body frame on top of the input
// stack (so the post-condition stack length is the input stack length
// plus one, minimum two when the input was non-empty).
const stack_with_body_frame_min = 2

// Pair_min caps Lo for numeric axes whose minimum is two — used by callers
// where the value is "≥2 of something" without a more specific domain
// constant fitting. Where a domain-specific name is clearer, prefer that.
const pair_min = 2

// Package_lines_test_max is the Hi bound on the per-package source-line
// accumulator in the package-fragmentation check. The endpoint is anchored
// by the fragmentation test fixtures (each package ≤ ~12k lines).
const package_lines_test_max = 12750

// Test_package_files_max is the Hi bound on the caller-supplied package
// file-count cap. The fragmentation tests exercise both endpoints (1 file
// allowed; 2 files as the caller-imposed ceiling).
const test_package_files_max = 2

// Build_constraint_key_chars_max caps the normalized build-constraint AST
// string used as a fragmentation grouping key. Sized for the typical
// multi-OS multi-arch expression length seen in practice.
const build_constraint_key_chars_max = 125

// Count_one anchors "exactly one of something" sentinel checks (e.g. a
// Sometimes(len(xs) == 1) on a single-element group). Value-identical to
// non_empty_min but read at the call site with different intent.
const count_one = 1

// Obligation_identifiers_max caps the number of identifiers in a single
// declaration-from-call obligation. Go allows arbitrary multi-LHS, but
// 3-LHS is the widest shape observed in lint.go's own source; setting the
// Hi bucket here gates the Distinct_Boundary axis on obligation.Identifiers
// to a reachable endpoint.
const obligation_identifiers_max = 3

// Successor_statements_max caps obligation.Successor_Statements via the
// generated many-successor fixture; lint.go's own scan stays under because
// every := decl is followed by ≤30 statements in the same block.
const successor_statements_max = 30

// Leaf_requirements_per_dispatch_max caps the number of requirement records
// leaf_dispatch returns: channel leaves emit 3 (pointer + boundary_int +
// zero_int), slice/map leaves emit 2 (boundary_int + zero_slice/zero_map),
// non-container/non-channel types emit 0.
const leaf_requirements_per_dispatch_max = 3

// Module_index_not_found anchors the -1 sentinel returned when no module
// matches a path lookup. Paired with modules_per_workspace_max as the Hi
// bound on the index domain.
const module_index_not_found = -1

// Modules_per_workspace_max caps the per-workspace module count. The
// monorepo's go.work currently lists ~10 modules; 1024 leaves headroom
// for several orders of magnitude of growth without admitting absurd values.
const modules_per_workspace_max = 1024

// Path_root is the path.Dir result for top-level entries: a single dot
// meaning "current directory". Used as the sentinel comparison value when
// detecting root-level paths.
const path_root = "."

// Declaration_diagnostic_name is the constant Name field on a Diagnostic
// emitted by build_declaration_diagnostic. Pulled out as a file-level const
// so the diag.Name invariant can bind it as a named bound.

// Inside_if_diagnostic_name is the fixed Name for inside-if-only diagnostics.
// Inside_if_diagnostic_name_chars must equal its length: the builder asserts
// `Always(len(diag.Name) == inside_if_diagnostic_name_chars)` to satisfy the
// boundary_int requirement on a Name whose value is invariant (a constant
// length can't reach a Distinct_Boundary's two endpoints). The runtime
// assertion catches any drift between the string and the count.
const inside_if_diagnostic_name_chars = 34

// Inside_if_diagnostic_want_chars is the byte length of the inside-if-only
// Want hint (the inline literal in the builder; the `—` em-dash is 3 bytes).
// The runtime Always(len(diag.Want) == it) catches any drift.
const inside_if_diagnostic_want_chars = 164

// Missing_diagnostic_name is the fixed Name for missing-axis diagnostics;
// missing_diagnostic_name_chars must equal its length (see inside_if note).
const missing_diagnostic_name_chars = 27

// Declaration_diagnostic_name_chars is the length of declaration_diagnostic_name
// assert len(diag.Name) == it to bound a Name whose value is invariant.
const declaration_diagnostic_name_chars = 45

// Pointer_requirement_kind is the fixed Kind for pointer requirements;
// pointer_requirement_kind_chars must equal its length.
const pointer_requirement_kind = "pointer"
const pointer_requirement_kind_chars = 7

// Stream_checker_count is the fixed number of stream-tier checks; the builder
// asserts len(checks) == it (a Distinct_Boundary can't bound a constant count).
const stream_checker_count = 9

// Fixed Name strings for the stream-check closures, each paired with its
// length so the checker's defer bounds c.Name (a value invariant per closure).
const agents_pair_check_name = "agents-claude-pair"
const agents_pair_check_name_chars = 18

// Recursion_message_chars_min / recursion_message_chars_max cap the
// "recursion: <node> calls itself" diagnostic message. Lo = 25 chars for
// the 1-char node case; Hi = 152 chars for a max-length 128-char node.
const recursion_message_chars_min = 25
const recursion_message_chars_max = 152

// Defer_position_name_chars_min / defer_position_name_chars_max bound the
// `Name` field on diagnostics built by
// check_invariant_assertions_validate_defer_position. Names are one of two
// fixed labels: `assertion_defer_missing` (23) or
// `assertion_defer_not_at_body_zero` (32).
const defer_position_name_chars_min = 23
const defer_position_name_chars_max = 32

// Defer_position_message_chars_min is the provable floor on the Sprintf'd
// defer-position diagnostic message: the shortest function label (1 char) plus
// the shortest of the three message bodies (102 chars). No label is shorter
// than one character and no body is shorter than 102, so a message can never
// fall below this — making it a panic-safe Lo for the message boundary.
const defer_position_message_chars_min = 103

// Defer_position_want_chars_min / defer_position_want_chars_max bound the
// `Want` field on diagnostics built by validate_defer_position. Want strings
// are three fixed remediation hints: `add an assertion defer ...` (71),
// `move the assertion defer ...` (76), and `place the assertion defer ...`
// (103). Test corpus exercises the shortest (`add`) and longest (`place`).
const defer_position_want_chars_min = 71
const defer_position_want_chars_max = 103

// Declaration_diagnostic_want_chars_min / declaration_diagnostic_want_chars_max
// bound the `Want` field on diagnostics built by
// check_invariant_assertions_build_declaration_diagnostic. Want strings come
// in two shapes: a short single-LHS suggestion (`add an invariant assertion
// ... covering: <list>`) and a long multi-LHS suggestion (`use
// invariant.Cross_Product ... covering: <list>`).
const declaration_diagnostic_want_chars_min = 65
const declaration_diagnostic_want_chars_max = 133

// Declaration_diagnostic_message_chars_min is the shortest declaration-obligation
// message: a single-LHS `<f>: declaration via function call must be followed by
// an invariant assertion covering: <x>` with a 1-char function label and a
// 1-char identifier. The Hi end is the budget ceiling (diagnostic_message_chars_max,
// which no message — bounded by label + identifier-list + the fixed clause —
// reaches), so the boundary masks Hi and observes only the single-LHS Lo.
const declaration_diagnostic_message_chars_min = 87

// Diagnostic_message_chars_max caps the upper bound for a diagnostic
// Message string. Longest observed messages embed a 128-char function label
// plus 257-char field_description plus suggestion text; round to 1024.
const diagnostic_message_chars_max = 1024

// Banned_lists_per_check_max caps the static list-of-lists count for the
// banned-segment check (universal, function-only, file-only, package-only).
const banned_lists_per_check_max = 4

// Suggested_sig_chars_min caps the shortest suggested function signature.
// In practice the shortest fixture-observed signature is the 21-char
// `f(*f_Input) (result T)` template with a single-letter funcname and a
// minimal result type.
const suggested_sig_chars_min = 21

// Convert_to_message_chars_min / convert_to_message_chars_max bound the
// `convert to <sig>` diagnostic message constructed by
// check_input_struct_validate. The 11-character "convert to " prefix is
// added to the suggested signature's length bounds.
const convert_to_message_chars_min = suggested_sig_chars_min + 11
const convert_to_message_chars_max = suggested_sig_chars_max + 11

// Stdlib_term_chars_max caps the stdlib-allowlist terminology suffix
// string. Longest entry is `offset` (6 chars).
const stdlib_term_chars_max = 6

// Stdlib_term_chars_min is the shortest term in the arithmetic-result
// vocabulary: `size` (4 chars). Paired with stdlib_term_chars_max as Hi
// for axes over Left/Right operand-term strings.
const stdlib_term_chars_min = 4

// Method_params_test_corpus_max matches the Params string the
// Test_Coverage_Backfill_Method_Render_Type fixture produces for its Bar
// method: a 1-char type `A` joined to a 128-char type via `,` totals 130.
// Bounded axes over input.Params in check_unnecessary_method_matches_stdlib
// use this as Hi so Bar's call observes the Hi bucket.
const method_params_test_corpus_max = Identifier_Chars_Max + 2

// Qualified_ident_chars_min caps `pkg.Func` shapes at their minimum: a
// single-letter package, dot, single-letter func — three characters.
const qualified_ident_chars_min = 3

// Rename_suggestion_chars_min caps the shortest `<name>_<term>` rename
// suggestion; the smallest single-word replacement ("count") is 5 chars.
const rename_suggestion_chars_min = 5

// Ing_word_chars_min is the smallest word that can carry the `-ing` participle
// suffix: three characters (the suffix itself plus a one-letter prefix would
// not actually be a valid English word, but the linter only inspects shape).
const ing_word_chars_min = 3

// Source_with_comment_bytes_min is the smallest source file that carries a
// comment after the package clause: "package x\n\n// c\n" is 16 bytes.
const source_with_comment_bytes_min = 16

// Field_description_chars_min caps the shortest `<name> <type>` description:
// one-char name + space + one-char type = 3 chars.
const field_description_chars_min = 3

// Requirement_field_description_chars_min is the smallest field_description
// length that survives the keyword_kinds filter and reaches a requirement.
// "a *T" (4 chars: 1-char name + " " + "*T" pointer) is the shortest such
// shape — bare `a T` for a user-defined Ident gets dropped because kinds
// is nil at the leaf path.
const requirement_field_description_chars_min = 4

// Cross_product_helper_chars / recorder_cross_product_helper_chars are the
// string lengths of the two Cross_Product helper-name shapes. Paired as
// Lo / Hi on the helper_name length axis in call_covered_pairs_cross_product.
const cross_product_helper_chars = 13
const recorder_cross_product_helper_chars = 22

// Bare_credit_kind_chars_max caps the bare-composable kind strings used in
// the bare_table: `bool` (4) is the Lo via credit_kind_chars_min, and
// `boundary_int` (12) is the Hi for the bare-credit family.
const bare_credit_kind_chars_max = 12

// Helper_family_index_unknown / helper_family_index_recorder anchor the
// three-valued helper-family discriminator: -1 = unknown / not an invariant
// helper, 0 = naked Always/Sometimes (resolved by middle case), 1 =
// Recorder_-prefixed variant.
const helper_family_index_unknown = -1
const helper_family_index_recorder = 1

// Always_family_chars_max caps the longest helper name in the
// Always/Sometimes nil-eq family: `Recorder_Always` (15).
const always_family_chars_max = 15

// Sign_negative / sign_positive anchor the three-valued sign domain
// returned by expression_sign: -1 for negative, 0 for zero (interior), +1
// for positive.
const sign_negative = -1
const sign_positive = 1

// Invariant_suggestion_chars_min / invariant_suggestion_chars_max cap the
// `use invariant.X(...)` remediation string rendered into assertion-coverage
// diagnostics. Sized empirically from the shortest (`pointer` shape) and
// longest (`boundary_float` shape) wrappers around the axis call.
const invariant_suggestion_chars_min = 167
const invariant_suggestion_chars_max = 373

// Composable_helper_chars_min / composable_helper_chars_max cap the
// helper-name string for composable axis builders: `Always` (6) is the
// shortest, `Distinct_Boundary` (17) the longest.
const composable_helper_chars_min = 6
const composable_helper_chars_max = 17

// Suggested_axis_call_chars_min / suggested_axis_call_chars_max cap the
// rendered axis-builder template string for one assertion requirement.
// Sized empirically from the shortest pointer-shape and longest
// Distinct_Boundary-shape templates.
const suggested_axis_call_chars_min = 22
const suggested_axis_call_chars_max = 228

// Invariant_selector_chars_max caps the longest selector_name string in the
// full invariant-call family: `Recorder_Distinct_Boundary` (26).
const invariant_selector_chars_max = 26

// If_init_identifier_chars_max caps identifier strings appearing in
// if/for/switch init lines: a tighter bound than Identifier_Chars_Max to
// reflect what fits in a single statement line within the line-length budget.
const if_init_identifier_chars_max = 55

// Tier_2_checks_count / tier_1_checks_count anchor the static tier-list
// length axis. Updated whenever a check is added or removed from the
// dispatcher in Check_File.
const tier_2_checks_count = 6
const tier_1_checks_count = 31

// Go_filename_chars_min is the shortest Go filename: a single-letter package
// name followed by the .go extension, e.g. `a.go`. Used as the Lo bound on
// filename axes in Check_Source / Check_File_System.
const go_filename_chars_min = 4

// The requirement-position label passed to the validate helper is always one of
// "param" (5), "param_defer" (11), or "named_return" (12). Both endpoints are
// always observed (every analyzed function runs the param and named_return
// passes), so the boundary needs no masking.
const validate_position_chars_min = 5
const validate_position_chars_max = 12

// Inside-if-only diagnostics carry a requirement.Kind assertion-kind label.
// The non-nillable leaves that reach this path are integer leaves, whose kinds
// are "zero_int" (8) and "boundary_int" (12) — both observed from a single
// `(result int)` named return, so the boundary needs no masking.
const inside_if_kind_chars_min = 8
const inside_if_kind_chars_max = 12

// The smallest source a parsed file can carry is the shortest valid Go file,
// "package a\n" (10 bytes) — the package keyword, a one-char name, and the
// gofmt-mandated trailing newline. Nothing is shorter, so this is the panic-safe
// Lo for pf.Source byte-length boundaries; a pinned fixture observes it.
const go_source_bytes_min = 10

// Package-group diagnostics carry empty Name and Want (the group-level message
// lives in Message). len == 0 is the constant width; Always(len == this) credits
// boundary_int via the numeric-credit path without a Lo<Hi Distinct_Boundary.
const package_group_diag_chars = 0

// Field_Description for an inside-if leaf is "<name> <type>". The shortest
// reachable is a one-char-named integer leaf, "x int" (5) — names are never
// empty and "int" is the shortest non-nillable type, so 5 is the floor. The
// budget cap (field_description_chars_max) is never reached, so Hi is masked.
const inside_if_field_chars_min = 5

// Inside_if_message_chars_min is the shortest inside-if-only diagnostic message:
// a 1-char function label, the shortest `param` prefix, and a min-length field
// description, plus the fixed `... must be asserted outside any if ... != nil
// ...` clause. Hi is the budget ceiling (diagnostic_message_chars_max), never
// reached, so the boundary masks Hi and observes only this Lo.
const inside_if_message_chars_min = 113

// Want_name_chars_min / want_name_chars_max cap the input-struct
// expected name (e.g. "f_Input" for function `f`). The "_Input" suffix
// is 6 chars; combined with the shortest (1-char) function name the
// minimum is 7. Max is Identifier_Chars_Max + 6 = 134.
const want_name_chars_min = 7
const want_name_chars_max = Identifier_Chars_Max + 6

// Filesystem_path_chars_max caps filesystem path strings the linter
// processes. POSIX PATH_MAX is 4096 on Linux; the linter inherits this
// as the hard bound for file paths and is exercised by the long-path
// backfill test which constructs a 4096-char filename.
const filesystem_path_chars_max = 4096

// Filesystem_directory_chars_max caps directory strings (path.Dir of a
// file path). The directory consumes at most filesystem_path_chars_max
// minus the shortest basename `/a.go` (5 chars).
const filesystem_directory_chars_max = filesystem_path_chars_max - 5

// Inferred_field_kind_chars_max caps the strings returned by
// check_invariant_assertions_infer_field_kind: "int" (3), "bool" (4),
// "pointer" (7), or "" (0). Longest entry is "pointer" at 7 chars.
const inferred_field_kind_chars_max = 7

// Field_description_chars_max caps `<name> <type_str>` descriptions:
// at most one identifier plus a space plus a type expression that itself
// is bounded by identifier length, yielding 2*identifier + 1.
const field_description_chars_max = 2*Identifier_Chars_Max + 1

// Suggested_sig_chars_max caps suggested function-signature strings of the
// form `<funcname>(*<funcname>_Input) (result <type>)`. The funcname
// appears twice (raw plus inside `_Input`), plus the wrapping syntax and
// a result clause; budget is 2*identifier + 6 (`_Input`) + ~16 (result).
const suggested_sig_chars_max = 2*Identifier_Chars_Max + 22

// Comment_text_chars_max caps raw comment text. comment_body strips the
// leading `//` and any whitespace, so the text bound is the body budget
// (4096 chars) plus the 2-char `//` prefix the scanner preserves.
const comment_text_chars_max = filesystem_path_chars_max + 2

// Banned_segment_chars_max caps the longest banned-segment word in the
// banned_segments_universal list: "utilities" at 9 chars.
const banned_segment_chars_max = 9

const function_lines_max = 70

// Git's default short-hash width.
const git_short_hash_chars = 10

// Git's full SHA-1 width — the maximum a `%H` format will produce. Used
// as the hard bound for hash-shaped inputs when git is in SHA-1 mode.
const git_full_hash_chars = 40

// Git's SHA-256 hash width — git's optional SHA-256 object format. Used
// as the hard bound for hash-shaped inputs since git_input_check_short_hash
// must accept either format.
const git_full_hash_chars_sha_256 = 64

const lines_per_file_max = 10000

// Mirrors line_chars_max used by the source-line check: code-review UIs
// truncate around 72–100 chars and longer subjects force horizontal scroll.
const commit_subject_chars_max = 100

// Diagnostics_per_call_max caps the slice length of `diags []Diagnostic`
// returns. A single check may emit one diagnostic per source line at worst,
// so the budget tracks lines_per_file_max with headroom for declarations
// that emit multiple diagnostics each.
const diagnostics_per_call_max = lines_per_file_max * 4

// Parsed_files_per_call_max caps the slice length of `parsed_files
// []parsed_file` and similar package-level slices. A package typically holds
// dozens of files; the cap leaves ample headroom for the worst-case monorepo
// flat-directory layout without admitting absurd values.
const parsed_files_per_call_max = 32768

// Ast_nodes_per_call_max caps generic []ast.Expr / []ast.Stmt / []*ast.Ident
// slices passed between helpers. Each AST list is bounded by the source file
// it derives from; lines_per_file_max is a generous upper bound.
const ast_nodes_per_call_max = lines_per_file_max

// String_slice_per_call_max caps generic []string slices used for path
// lists, candidate sets, identifier chains, and similar identifier-bag
// collections.
const string_slice_per_call_max = lines_per_file_max

// Coverage_pairs_per_call_max caps invariant-assertion coverage-pair slices.
// One call may produce one pair per (path, kind) tuple — bounded by the
// number of tracked identifiers in any one function, which is well below
// Identifier_Chars_Max × credit_kind_chars_max in practice.
const coverage_pairs_per_call_max = lines_per_file_max

// Caps the three agent-facing docs at 100 lines. These files are loaded into
// every agent invocation, so each extra line is a per-call tax on context and
// attention. Skills that outgrow the budget should be split; CLAUDE.md/AGENTS.md
// that outgrow it usually mean a repo-level instruction belongs in a
// sub-package's pair instead.
const agent_documentation_lines_max = 100

// Visual width cap for markdown lines. The 100-rune limit makes prose readable
// in narrow editor splits and side-by-side diffs.
const markdown_line_max = 100

// Exit_code_max is the Hi bound on Main's return code: 0 = clean, 1 =
// diagnostics. Code 2 (hard error) is unreachable per upstream Excluding on
// the stream/read/modules error branches.
const exit_code_max = 1

// Cpu_count_max caps input.CPU_Count to a sane per-process worker budget.
// Servers with more than 1024 CPUs are unreachable in this codebase and the
// linter's fork-join pools would not benefit from going wider.
const cpu_count_max = 1024

// NEVER ADD A THIRD TIER. NEVER ADD A ZERO TIER. tier_max is the Hi bound on
// Diagnostic.Tier and stays 2: there are exactly two tiers. Tier 1 always prints
// and its presence suppresses tier 2 everywhere in scope; tier 2 prints only
// when no tier 1 fired anywhere in scope. That two-state gate is the whole output
// path — a third gating state would rot it. The Tier field's zero value is not a
// tier: never lean on it, never label it "tier 0," and give every new diagnostic
// an explicit tier 1 or tier 2.
const tier_max = 2

// Configuration is the decoded form of the workspace's lint.json, which sits
// beside go.work. It carries the per-workspace policy that would otherwise be
// hard-coded into the checks: the shared module's identity and the global-API
// allowlist.
type Configuration struct {
	// Shared_Module is the workspace-root-relative directory of the workspace's
	// shared library module (e.g. "shared") — the one module every other
	// may import. It drives the shared-vs-binary classification; every other
	// module at the workspace root is treated as a binary. Slash-relative, like
	// the allowlist. Required: a config without it is rejected.
	Shared_Module string `json:"shared_module"`
	// Instrumentation_Packages names the workspace-root-relative directories of
	// write-only instrumentation — assertions, snapshot tooling, telemetry. They
	// may expose a `var Default`, and a pure or deterministic package may import
	// them despite the import bans, since emitting to a write-only side channel
	// cannot feed impurity or nondeterminism back into the importer. Segment-
	// prefix: an entry covers itself and its whole subtree.
	Instrumentation_Packages []string `json:"instrumentation_packages"`
	// Deterministic_Packages names the module top-level directories whose pure
	// packages are held to the deterministic tier on top of purity: no goroutine,
	// channel, select, nor time/context/sync import, and every first-party import
	// must itself be deterministic. An entry covers the pure packages at or under
	// it, so a binary module is named by its bare top-level directory and the
	// shared module's libraries by `shared/*` (all) or `shared/<lib>` (one); the
	// fixed module shape lets the tier auto-apply without listing each package.
	// Impure packages in the subtree (the main package, a default tier) are
	// dropped, not reported. Opt-in; empty lists none. An entry covering no pure
	// package is reported as a coverage gap.
	Deterministic_Packages []string `json:"deterministic_packages"`
	// Word_Replacements drives the vocabulary check: each tokenized, lowercased
	// word maps to its preferred replacements (id -> identifier). An empty list
	// bans the word with no rename suggestion (util, len); an absent key is left
	// alone. Required: a config without it is rejected so the check can never
	// silently go dark.
	Word_Replacements map[string][]string `json:"word_replacements"`
	// Ignore extends the hardcoded global ignore list (Ignored_Directory) with
	// per-workspace entries, as gitignore-style globs: a slash-less entry floats
	// and matches that basename at any depth, an entry with a slash is anchored to
	// the workspace root, a trailing slash binds to directories (and thus their
	// whole subtree), and ** spans path segments while * stays within one. A
	// matching path is dropped from the scan set entirely, so no tier fires on it.
	// Opt-in; empty ignores nothing.
	Ignore []string `json:"ignore"`
}

// Main_Input bundles every external dependency the linter needs.
// Construction lives in main.go (production) or fstest.MapFS-backed
// tests (unit tests) — the library tier never reaches out to impure
// state itself.
type Main_Input struct {
	// Fsys is the workspace tree the linter reads; every scanned path is
	// resolved against it.
	Fsys fs.FS
	// Stdout receives the success line and the per-diagnostic report.
	Stdout io.Writer
	// Stderr receives hard-error messages that abort the run before any
	// diagnostic is printed.
	Stderr io.Writer
	// Root_Directory is the OS path that matches Fsys. The stream-tier symlink
	// check needs real-OS access through Readlink below because fs.FS has no
	// symlink primitive. An empty Root_Directory self-disables that one check so
	// fstest.MapFS-backed tests don't need to special-case it.
	Root_Directory string
	// Tracked is the set of paths (relative to Fsys root) the linter is
	// allowed to look at — typically the union of git-tracked and
	// git-untracked-but-not-ignored files. When non-nil, walkers skip
	// every path outside this set, and prune any directory containing no
	// such path. nil disables the filter entirely (fstest.MapFS tests
	// stay green without having to enumerate every entry).
	Tracked map[string]bool
	// Git carries the commit-history tier's input; the zero value
	// (Enabled false) skips that tier entirely.
	Git Git_Input
	// CPU_Count caps parallelism for the parse and check phases. Injected
	// by main.go (binds to runtime.NumCPU); tests may leave it 0, which
	// degrades to single-threaded execution.
	CPU_Count int
	// Readlink reads a symlink's target for the stream-tier symlink check;
	// nil disables that check. main.go binds it to os.Readlink.
	Readlink func(name string) (target string, err error)
	// Scope_Prefix narrows the set of files diagnostics are emitted for.
	// Files outside this slash-separated prefix (relative to Fsys root)
	// are still walked and parsed — the doctrine checks need the broader
	// module view to compute correctly — but their diagnostics are
	// suppressed from output and don't count toward the exit code.
	// Empty disables the filter.
	Scope_Prefix string
}

// Git_Commit is one commit's identity for the git-history tier:
// the full hash and the subject line of the commit message.
type Git_Commit struct {
	// Hash is the commit's full object name, used to attribute a diagnostic
	// to the offending commit.
	Hash string
	// Subject is the first line of the commit message — the only part the
	// commit-history rules inspect.
	Subject string
}

// Git_Input drives the git-history tier. Zero value (Enabled=false) skips
// the tier — used when HEAD is on main, when the binary isn't run from a
// git repo, or in fstest.MapFS-backed unit tests that aren't about git.
// Main_Reference_Absent distinguishes "no main ref locally" (shallow CI checkout,
// brand-new repo) from "main ref present, no offending commits" so CI
// misconfiguration surfaces as a specific failure instead of silent pass.
type Git_Input struct {
	// Enabled gates the whole tier; the zero value skips git checks so
	// non-git callers and FS-only tests stay clean.
	Enabled bool
	// Main_Reference_Absent flags that no main ref was reachable, turning a
	// silent pass on a shallow checkout into an explicit diagnostic.
	Main_Reference_Absent bool
	// Merge_Commits holds the branch's merge commits, screened for the
	// no-merge-commits rule (subtree merges excepted).
	Merge_Commits []Git_Commit
	// Non_Merge_Commits holds the branch's ordinary commits, screened for
	// the subject-size, conventional-subject, and fixup rules.
	Non_Merge_Commits []Git_Commit
}

// Main is the linter's entry point. Returns the process exit code:
// 0 if every check passed (success message printed to Stdout), 1 if
// any diagnostic was emitted within Scope_Prefix, 2 on a hard error
// (filesystem walk failure, etc.).
func Main(input *Main_Input) (code int) {

	// Main is the single reader of lint.json: the config lives at the Fsys root
	// (beside go.work in production; injected into the MapFS in tests). Reading it
	// here rather than in main.go keeps one config path for both, so tests
	// exercise the real decode instead of bypassing it.
	configuration, configuration_err := read_configuration(input.Fsys)
	if configuration_err != nil {
		fmt.Fprintln(input.Stderr, configuration_err)
		return 2
	}
	// Git tier runs first: it reads only repo metadata, not the FS, for the fastest signal.
	git_diags := Git_Input_Check(input.Git)
	filesystem_diags, err := Check_File_System(&Check_File_System_Input{
		Fsys:                     input.Fsys,
		Root:                     ".",
		Root_Directory:           input.Root_Directory,
		Tracked:                  input.Tracked,
		CPU_Count:                input.CPU_Count,
		Readlink:                 input.Readlink,
		Scope:                    input.Scope_Prefix,
		Instrumentation_Packages: configuration.Instrumentation_Packages,
		Shared_Module:            configuration.Shared_Module,
		Deterministic_Packages:   configuration.Deterministic_Packages,
		Word_Replacements:        configuration.Word_Replacements,
		Ignore:                   configuration.Ignore,
	})
	if err != nil {
		fmt.Fprintln(input.Stderr, err)
		return 2
	}
	all_diags := append(git_diags, filesystem_diags...)
	// Tier-2 checks rely on tier-1 contracts (see check_file_checks_tier2);
	// hold their output until tier-1 is globally clean within the user's
	// scope. The per-file gate inside Check_File already drops tier-2 in
	// any file that has tier-1 issues — this is the cross-file extension
	// of the same rule, applied at print time so detection logic stays
	// unchanged.
	has_tier1 := false
	for _, d := range all_diags {
		if !diagnostic_within_scope(d, input.Scope_Prefix) {
			continue
		}
		if d.Tier == 1 {
			has_tier1 = true
			break
		}
	}
	emitted_count := 0
	for _, d := range all_diags {
		if !diagnostic_within_scope(d, input.Scope_Prefix) {
			continue
		}
		if has_tier1 {
			if d.Tier == 2 {
				continue
			}
		}
		emitted_count++
		fmt.Fprintf(input.Stdout, "%s: %s\n", d.Position, d.Message)
	}
	if emitted_count > 0 {
		return 1
	}
	// AI agents keep checking exit code if there's no explicit success message in output.
	fmt.Fprintln(input.Stdout, "✓ all checks passed")
	return 0
}

// Lint_json_bytes_max caps the bounded read of lint.json. A workspace config is
// a handful of lines plus the word-replacements table; the cap only bounds a
// pathological or accidental huge file so the read never allocates without limit.
const lint_json_bytes_max = 1 << 20

// Reads and decodes lint.json from the root of fsys. The read
// is bounded — fs.ReadFile is unbounded and banned for the same reason os.ReadFile
// is — so a pathological config can't exhaust memory. An absent, unreadable, or
// malformed lint.json is an error: the linter cannot derive shared_module or the
// word-replacements table on its own, so a missing or broken config must fail
// loudly rather than silently degrade the checks.
func read_configuration(fsys fs.FS) (configuration *Configuration, err error) {
	file, open_err := fsys.Open("lint.json")
	if open_err != nil {
		return nil, fmt.Errorf("lint.json is required and must set shared_module "+
			"and word_replacements: %w", open_err)
	}
	defer file.Close()
	buffer := make([]byte, lint_json_bytes_max)
	n, read_err := io.ReadFull(io.LimitReader(file, lint_json_bytes_max), buffer)
	// ReadFull returns ErrUnexpectedEOF for a file shorter than the buffer (the
	// normal case for a small config) and EOF for an empty file; neither is a read
	// failure. Any other error is a real I/O fault and is fatal.
	read_failed := read_err != nil
	if read_err == io.ErrUnexpectedEOF {
		read_failed = false
	}
	if read_err == io.EOF {
		read_failed = false
	}
	if read_failed {
		return nil, fmt.Errorf("cannot read lint.json: %w", read_err)
	}
	return Parse_Configuration(buffer[:n])
}

// Parse_Configuration decodes lint.json. shared_module and word_replacements are
// required — a config without either is rejected so the shared-vs-binary
// classification and the vocabulary check never silently degrade. An unknown
// top-level key is rejected so a typo fails loudly, and malformed or wrong-typed
// JSON is an error; callers treat any of these as a hard failure. The allowlist
// and deterministic-packages lists are optional and default to empty.
func Parse_Configuration(data []byte) (configuration *Configuration, err error) {
	// Decode twice over the same bounded buffer: once as a raw key map to
	// police unknown keys — json.Decoder.DisallowUnknownFields is the usual
	// guard, but json.NewDecoder streams an unbounded reader and is banned
	// here — and once into the typed struct for the values.
	keys := map[string]json.RawMessage{}
	if decode_err := json.Unmarshal(data, &keys); decode_err != nil {
		return nil, decode_err
	}
	known_keys := map[string]bool{
		"shared_module":            true,
		"instrumentation_packages": true,
		"deterministic_packages":   true,
		"word_replacements":        true,
		"ignore":                   true,
	}
	for key := range keys {
		if known_keys[key] {
			continue
		}
		return nil, fmt.Errorf("lint.json: unknown key %q", key)
	}
	configuration = &Configuration{}
	if decode_err := json.Unmarshal(data, configuration); decode_err != nil {
		return nil, decode_err
	}
	if configuration.Shared_Module == "" {
		return nil, fmt.Errorf("lint.json: shared_module is required")
	}
	// Empty (or absent) word_replacements is rejected, not defaulted: the
	// vocabulary check has no built-in table any more, so a missing one would
	// silently disable it rather than fail loudly.
	if len(configuration.Word_Replacements) == 0 {
		return nil, fmt.Errorf("lint.json: word_replacements is required")
	}
	if validate_err := validate_glob_patterns(
		"ignore", configuration.Ignore); validate_err != nil {
		return nil, validate_err
	}
	return configuration, nil
}

// Rejects gitignore-style glob entries the matcher cannot honor, so a broken
// list fails loudly at config load rather than silently matching nothing. field
// names the lint.json key for the error. An empty entry has no path to match; a
// leading "!" is gitignore negation, which our additive lists give no meaning;
// and a segment that path.Match deems malformed (an unterminated "[") would error
// on every comparison.
func validate_glob_patterns(field string, patterns []string) (err error) {
	for _, raw := range patterns {
		where := fmt.Sprintf("lint.json: %s entry %q", field, raw)
		if strings.TrimSpace(raw) == "" {
			return fmt.Errorf("lint.json: %s entry is empty", field)
		}
		if strings.HasPrefix(raw, "!") {
			return fmt.Errorf("%s: negation is unsupported", where)
		}
		for _, segment := range strings.Split(parse_glob_pattern(raw).Core, "/") {
			// ** is the matcher's own segment wildcard, not a path.Match token.
			if segment == "**" {
				continue
			}
			if _, match_err := path.Match(segment, ""); match_err != nil {
				return fmt.Errorf("%s: %w", where, match_err)
			}
		}
	}
	return nil
}

// True iff the diagnostic is inside the user's scope. Empty scope means
// no filter — all diagnostics pass. Git-tier diagnostics use synthetic
// `<git:…>` filenames; those never live under any scope prefix, so we
// admit them whenever the scope is anything other than empty by checking
// the leading "<" sentinel.
func diagnostic_within_scope(d Diagnostic, scope_prefix string) (within bool) {

	if scope_prefix == "" {
		return true
	}
	if strings.HasPrefix(d.Position.Filename, "<") {
		return true
	}
	if d.Position.Filename == scope_prefix {
		return true
	}
	return strings.HasPrefix(d.Position.Filename, scope_prefix+"/")
}

// Diagnostic is one rule violation. Position is the offending source
// location; Name and Want are machine-readable rule identity and
// suggested fix; Message is the human-readable line printed to stdout.
// Tier carries the file-check tier for print-time gating: 1 = tier-1
// (always printed; presence anywhere suppresses tier-2 output), 2 =
// tier-2 (printed only when no tier-1 fires globally). Diagnostics
// from non-file tiers (git, stream, cross-file) leave Tier zero — they
// always print and never gate tier-2.
type Diagnostic struct {
	// Position is the offending source location, printed as the clickable
	// file:line:col prefix.
	Position token.Position
	// Name is the machine-readable rule identity, stable for tooling that
	// groups or suppresses by rule.
	Name string
	// Want is the suggested fix, phrased as the desired post-state.
	Want string
	// Message is the human-readable line printed to stdout.
	Message string
	// Tier carries the file-check tier for print-time gating: 1 always
	// prints and suppresses tier-2 globally when present; 2 prints only
	// when no tier-1 fired; non-file tiers leave it 0.
	Tier int
}

type parsed_file struct {
	Path     string
	File_Set *token.FileSet
	File     *ast.File
	Source   []byte
}

var snake_case_re = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// The `[A-Z][A-Z0-9]*s?` arm admits the conventional Go pluralized-acronym
// word (APIs, IDs, URLs): an all-caps run with a single trailing lowercase
// `s`. Without it, `APIs` would be rejected and the spec heading "Unbounded
// APIs" could not have a conformant Test_<...>_Unbounded_APIs name.
var ada_case_re = regexp.MustCompile(
	`^([A-Z][a-z0-9]*|[A-Z][A-Z0-9]*s?)(_([A-Z][a-z0-9]*|[A-Z][A-Z0-9]*s?))*$`)

// Conventional Commits subject: lowercase type, optional (scope), optional
// `!` breaking-change marker, `: `, non-empty description. Scope contents
// are not whitelisted — package paths and ad-hoc area names both occur in
// the wild and a strict charset would generate more friction than signal.
var conventional_commit_re = regexp.MustCompile(`^[a-z]+(\([^)]+\))?!?: \S`)

func suggest_split_words(name string) (words []string) {
	var current []rune
	runes := []rune(name)
	flush := func() {
		if len(current) > 0 {
			words = append(words, string(current))
			current = nil
		}
	}
	for i, r := range runes {
		if r == '_' {
			flush()
			continue
		}
		if i > 0 {
			if unicode.IsUpper(r) {
				previous := runes[i-1]
				if unicode.IsLower(previous) {
					flush()
				} else if unicode.IsDigit(previous) {
					flush()
				} else if unicode.IsUpper(previous) {
					if i+1 < len(runes) {
						if unicode.IsLower(runes[i+1]) {
							flush()
						}
					}
				}
			}
		}
		current = append(current, r)
	}
	flush()
	return words
}

type suggest_input struct {
	Name string
	Want string
}

func suggest(input *suggest_input) (output string) {

	words := suggest_split_words(input.Name)
	if len(words) == 0 {
		return input.Name
	}
	parts := make([]string, len(words))
	for i, w := range words {
		if input.Want == "snake_case" {
			parts[i] = strings.ToLower(w)
			continue
		}
		if suggest_is_all_upper(w) {
			parts[i] = w
			continue
		}
		rs := []rune(strings.ToLower(w))
		rs[0] = unicode.ToUpper(rs[0])
		parts[i] = string(rs)
	}
	return strings.Join(parts, "_")
}

// Output Hi (=128 chars) reaches the cap only for ~86-char inputs (densest
// split is ~2N/3 words; rejoin with separator adds chars).

func suggest_is_all_upper(s string) (ok bool) {

	has_letter := false
	for _, r := range s {
		if unicode.IsLetter(r) {
			has_letter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return has_letter
}

type check_function = func(
	file_set *token.FileSet, file *ast.File, source []byte,
) (diags []Diagnostic)

// LINT POLICY: do not add checks that force capacity-bounded wrappers over
// raw collections. We explored porting Odin's fixed-slice concept (every
// []T replaced by slice.Fixed[T] with asserted capacity) and abandoned it:
// Go assumes effectively infinite memory — slices, maps, stacks all grow
// on demand — and every stdlib boundary takes raw []T, so the wrapper
// either leaks through Data()/From() accessors (no safety win) or forces
// conversion churn at every call site (no ergonomics). Pursue boundedness
// via orthogonal mechanisms instead: bounded queue depths at IO
// boundaries, bounded worker-pool concurrency, bounded request payloads,
// deadline-bound work units. Bound the rate and backlog of pressure, not
// the heap directly. Note: this rejection is narrowly about capacity
// wrappers over collections; other boundedness lints, e.g. banning
// unbuffered channels, remain on the table.

// LINT POLICY: check_single_caller_callee is abandoned. It required every
// unexported function with one caller to carry the caller's name as a
// prefix, on the theory that the chain signals locality. The type-prefix
// rule (check_file_system_method_prefix) supersedes it for any function whose first
// param is a same-package named type: the type prefix is the load-bearing
// signal, and forcing both prefixes produces verbose names like
// main_git_input_check. For functions outside the type-prefix rule's
// scope, the caller-prefix discipline turned out to be a soft convention
// readers can navigate via grep without lint enforcement.

// Tier 1: independent checks that can run on any well-formed Go file.

type scope struct {
	Parent *scope
	Names  map[string]bool
}

func check_shadows(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	global_names := make(map[string]bool)
	for _, declaration := range file.Decls {
		switch x := declaration.(type) {
		case *ast.FuncDecl:
			global_names[x.Name.Name] = true
		case *ast.GenDecl:
			for _, specification := range x.Specs {
				switch s := specification.(type) {
				case *ast.ValueSpec:
					for _, n := range s.Names {
						global_names[n.Name] = true
					}
				case *ast.TypeSpec:
					global_names[s.Name.Name] = true
				}
			}
		}
	}

	global_scope := &scope{Names: global_names}

	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok {
			check_shadows_function_body(file_set, global_scope, function, &diags)
		}
	}

	return diags
}

func check_shadows_function_body(
	file_set *token.FileSet, global_scope *scope, function *ast.FuncDecl, diags *[]Diagnostic,
) {

	function_scope := &scope{Parent: global_scope, Names: make(map[string]bool)}
	if function.Type.Params != nil {
		for _, f := range function.Type.Params.List {
			for _, nm := range f.Names {
				if nm.Name != "_" {
					function_scope.Names[nm.Name] = true
				}
			}
		}
	}
	if function.Type.Results != nil {
		for _, f := range function.Type.Results.List {
			for _, nm := range f.Names {
				if nm.Name != "" {
					if nm.Name != "_" {
						function_scope.Names[nm.Name] = true
					}
				}
			}
		}
	}
	if function.Body != nil {
		check_shadows_function_body_walk_body(
			file_set, function_scope, function.Body.List, diags)
	}
}

// Iteratively walks a sequence of statements with scope tracking. Replaces the
// former check_stmts/check_stmt pair, which were mutually recursive — banned
// by check_no_recursion. The stack holds one frame per nested scope; sibling
// scopes (if/else-if/else) are pushed together and processed LIFO, which is
// fine because they don't share state.
func check_shadows_function_body_walk_body(
	file_set *token.FileSet, root_scope *scope, root_statements []ast.Stmt, diags *[]Diagnostic,
) {
	stack := []walk_frame{{Scope: root_scope, Statements: root_statements}}
	for len(stack) > 0 {
		top := &stack[len(stack)-1]
		if top.I >= len(top.Statements) {
			stack = stack[:len(stack)-1]
			continue
		}
		statement := top.Statements[top.I]
		top.I++
		current_scope := top.Scope
		stack = check_shadows_function_body_walk_body_walk_statement(
			file_set, current_scope, statement, stack, diags)
	}
}

type walk_frame struct {
	Scope      *scope
	Statements []ast.Stmt
	I          int
}

func check_shadows_function_body_walk_body_walk_statement(
	file_set *token.FileSet,
	scope_value *scope,
	statement ast.Stmt,
	stack []walk_frame,
	diags *[]Diagnostic,
) (output []walk_frame) {
	switch x := statement.(type) {
	case *ast.BlockStmt:
		if x != nil {
			stack = append(stack, walk_frame{
				Scope:      scope_new_block(scope_value),
				Statements: x.List,
			})
		}
	case *ast.IfStmt:
		stack = check_shadows_function_body_walk_body_walk_statement_push_if_chain(
			file_set, scope_value, x, stack, diags)
	case *ast.ForStmt:
		for_scope := scope_new_block(scope_value)
		if x.Init != nil {
			check_assign_define(file_set, for_scope, x.Init, diags)
		}
		if x.Body != nil {
			stack = append(stack, walk_frame{Scope: for_scope, Statements: x.Body.List})
		}
	case *ast.RangeStmt:
		stack = check_shadows_function_body_walk_body_walk_statement_push_range_statement(
			file_set, scope_value, x, stack, diags)
	case *ast.AssignStmt:
		check_shadows_function_body_walk_body_walk_statement_assign_statement(
			file_set, scope_value, x, diags)
	case *ast.DeclStmt:
		check_shadows_function_body_walk_body_walk_statement_declaration_statement(
			file_set, scope_value, x, diags)
	}
	return stack
}

// Bundles walk_statement's exit-assertion operands: the incoming stack and the
// returned output share the []walk_frame element type, so positional parameters
// would trip the same-type-parameter bundling rule.
type check_shadows_function_body_walk_body_walk_statement_assert_exit_input struct {
	Stack       []walk_frame
	Output      []walk_frame
	Diags       *[]Diagnostic
	Scope_Value *scope
}

// |output-stack| ≤ 1 per call (single push or pop); (Hi,Hi) is the
// AST safety cap, not a coverage gap.

func scope_new_block(parent *scope) (new_scope *scope) {

	return &scope{Parent: parent, Names: make(map[string]bool)}
}

// New_scope.Parent is the constructor's `parent`; the entry asserts
// parent.Parent is non-nil, so new_scope.Parent.Parent inherits that
// guarantee. The names axis is bounded by AST budget; the constructor
// returns a freshly-made empty map so Lo=0 is invariant — paired with the
// parent_parent_parent_nil axis to keep Hi unreachable (a brand-new map
// can never be at the safety cap).

func check_shadows_function_body_walk_body_walk_statement_push_if_chain(
	file_set *token.FileSet,
	scope_value *scope,
	root *ast.IfStmt,
	stack []walk_frame,
	diags *[]Diagnostic,
) (output []walk_frame) {
	current := root
	for current != nil {
		if_scope := scope_new_block(scope_value)
		if current.Init != nil {
			check_assign_define(file_set, if_scope, current.Init, diags)
		}
		if current.Body != nil {
			stack = append(stack, walk_frame{
				Scope:      if_scope,
				Statements: current.Body.List,
			})
		}
		if current.Else == nil {
			return stack
		}
		if next, is_if := current.Else.(*ast.IfStmt); is_if {
			current = next
			continue
		}
		if bs, is_block := current.Else.(*ast.BlockStmt); is_block {
			stack = append(stack, walk_frame{
				Scope:      scope_new_block(scope_value),
				Statements: bs.List,
			})
		}
		return stack
	}
	return stack
}

// Takes stack's length as an int rather than the slice itself: the slice would
// share []walk_frame with output, and two same-type parameters must bundle.
// Function only appends frames (output ≥ stack); (Hi,Hi) is the
// AST safety cap, not a working shape.

func check_shadows_function_body_walk_body_walk_statement_push_range_statement(
	file_set *token.FileSet,
	scope_value *scope,
	x *ast.RangeStmt,
	stack []walk_frame,
	diags *[]Diagnostic,
) (output []walk_frame) {
	range_scope := scope_new_block(scope_value)
	check_shadows_function_body_walk_body_walk_statement_push_range_statement_add_variable(
		file_set, range_scope, x.Key, diags)
	check_shadows_function_body_walk_body_walk_statement_push_range_statement_add_variable(
		file_set, range_scope, x.Value, diags)
	if x.Body != nil {
		stack = append(stack, walk_frame{Scope: range_scope, Statements: x.Body.List})
	}
	return stack
}

type push_range_assert_exit_input struct {
	Stack  []walk_frame
	Output []walk_frame
	Diags  *[]Diagnostic
}

// Range body always appends one frame so output == stack+1 modulo
// the one-frame delta; cross-extreme tuples are unreachable.

func check_shadows_function_body_walk_body_walk_statement_push_range_statement_add_variable(
	file_set *token.FileSet,
	scope_value *scope,
	e ast.Expr,
	diags *[]Diagnostic,
) {
	// Scope_value here is always a range_scope freshly created by
	// scope_new_block, whose parent is the caller's scope_value (function
	// or deeper). scope_new_block's entry asserts parent.Parent != nil, so
	// range_scope.Parent.Parent is non-nil by construction.

	if e == nil {
		return
	}
	identifier, is_ident := e.(*ast.Ident)
	if !is_ident {
		return
	}
	if identifier.Name == "_" {
		return
	}
	check_shadow(file_set, scope_value, identifier.Name, identifier, diags)
	scope_value.Names[identifier.Name] = true
}

func check_shadows_function_body_walk_body_walk_statement_assign_statement(
	file_set *token.FileSet,
	scope_value *scope,
	x *ast.AssignStmt,
	diags *[]Diagnostic,
) {

	if x.Tok != token.DEFINE {
		return
	}
	for _, lhs := range x.Lhs {
		if identifier, ok := lhs.(*ast.Ident); ok {
			if identifier.Name != "_" {
				check_shadow(
					file_set, scope_value, identifier.Name, identifier, diags)
				scope_value.Names[identifier.Name] = true
			}
		}
	}
}

func check_shadows_function_body_walk_body_walk_statement_declaration_statement(
	file_set *token.FileSet,
	scope_value *scope,
	x *ast.DeclStmt,
	diags *[]Diagnostic,
) {

	if x.Decl == nil {
		return
	}
	generic_declaration, ok := x.Decl.(*ast.GenDecl)
	if !ok {
		return
	}
	for _, specification := range generic_declaration.Specs {
		vs, is_value := specification.(*ast.ValueSpec)
		if !is_value {
			continue
		}
		for _, nm := range vs.Names {
			if nm.Name != "_" {
				check_shadow(file_set, scope_value, nm.Name, nm, diags)
				scope_value.Names[nm.Name] = true
			}
		}
	}
}

func check_assign_define(
	file_set *token.FileSet, scope_value *scope, statement ast.Stmt, diags *[]Diagnostic,
) {
	// Scope_value here is an if/for/range init scope produced by
	// scope_new_block, whose entry assertion guarantees parent.Parent !=
	// nil. So scope_value.Parent.Parent is non-nil by construction.

	if as, ok := statement.(*ast.AssignStmt); ok {
		if as.Tok == token.DEFINE {
			for _, lhs := range as.Lhs {
				if identifier, is_ident := lhs.(*ast.Ident); is_ident {
					if identifier.Name != "_" {
						check_shadow(
							file_set,
							scope_value,
							identifier.Name,
							identifier,
							diags)
						scope_value.Names[identifier.Name] = true
					}
				}
			}
		}
	}
}

func check_shadow(
	file_set *token.FileSet,
	scope_value *scope,
	name string,
	identifier *ast.Ident,
	diags *[]Diagnostic,
) {

	p := scope_value.Parent
	for p != nil {
		if p.Names[name] {
			*diags = append(*diags, Diagnostic{
				Position: file_set.Position(identifier.Pos()),
				Message: fmt.Sprintf(
					"variable %q shadows outer scope variable", name),
			})
			return
		}
		p = p.Parent
	}
}

// Check_File runs every per-file check (tier-1 first, then tier-2 if
// tier-1 was clean) on one already-parsed file and returns the
// accumulated diagnostics. Used both by Check_Source and by the
// file-system tier's per-file pass. Stamps each diagnostic with its
// origin tier so the printer can gate tier-2 output globally on the
// presence of any tier-1 diagnostic.
func Check_File(
	file_set *token.FileSet, file *ast.File, source []byte, instrumentation []string,
	word_replacements map[string][]string,
) (diags []Diagnostic) {
	diags = check_file_run_tier([]check_function{
		check_casing,
		check_named_returns,
		check_no_naked_return,
		check_shadows,
		check_line_character_count,
		check_function_line_count,
		check_compound_if,
		check_comments,
		check_main_first,
		check_assertion_named_constant,
		check_no_discard,
		check_public_struct_fields,
		check_struct_field_documentation_comment,
		check_exported_type_exposes_private,
		check_no_iota,
		check_no_fallthrough,
		check_no_blank_import,
		check_no_grouped_declaration,
		check_keyed_struct_init,
		check_gofmt,
		check_no_dot_import,
		check_default_package_name,
		check_test_package,
		check_no_empty_function_body,
		check_no_interfaces,
		check_input_struct,
		make_check_names_vocabulary(word_replacements),
		check_test_documentation_comment,
		check_snap_backtick,
		check_names,
		check_no_bare_for,
		check_exported_documentation_comment,
		check_blank_synchronization_mutex,
	}, file_set, file, source)
	if len(diags) > 0 {
		for i := range diags {
			diags[i].Tier = 1
		}
		return diags
	}
	diags = check_file_run_tier([]check_function{
		check_no_unbounded_apis, check_no_recursion,
		check_no_function_init, make_check_no_package_vars(instrumentation),
		check_unnecessary_method,
		check_no_third_party_struct_tag,
	}, file_set, file, source)
	for i := range diags {
		diags[i].Tier = 2
	}
	return diags
}

func check_file_run_tier(
	checks []check_function, file_set *token.FileSet, file *ast.File, source []byte,
) (diags []Diagnostic) {
	per_check := make([][]Diagnostic, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Add(1)
		go func(i int, c check_function) {
			defer wg.Done()
			per_check[i] = c(file_set, file, source)
		}(i, c)
	}
	wg.Wait()
	for _, d := range per_check {
		diags = append(diags, d...)
	}
	return diags
}

func check_casing_ident(file_set *token.FileSet, identifier *ast.Ident, diags *[]Diagnostic) {

	if identifier.Name == "_" {
		return
	}
	first := rune(identifier.Name[0])
	if !unicode.IsLetter(first) {
		return
	}
	want := "snake_case"
	ok := snake_case_re.MatchString(identifier.Name)
	if unicode.IsUpper(first) {
		want = "Ada_Case"
		ok = ada_case_re.MatchString(identifier.Name)
	}
	if !ok {
		suggestion := suggest(&suggest_input{Name: identifier.Name, Want: want})
		*diags = append(*diags, Diagnostic{
			Position: file_set.Position(identifier.Pos()),
			Name:     identifier.Name,
			Want:     want,
			Message:  fmt.Sprintf("%s -> %s", identifier.Name, suggestion),
		})
	}
}

func check_casing(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	check := func(identifier *ast.Ident) {
		check_casing_ident(file_set, identifier, &diags)
	}
	check_field_list := func(fl *ast.FieldList) {
		if fl == nil {
			return
		}
		for _, f := range fl.List {
			for _, n := range f.Names {
				check(n)
			}
		}
	}

	ast.Inspect(file, func(n ast.Node) (descend bool) {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// TestMain is a Go testing-package reserved name; the
			// runner only recognizes that exact spelling.
			if x.Name.Name != "TestMain" {
				if !check_casing_method_satisfies_stdlib(x) {
					check(x.Name)
				}
			}
			check_field_list(x.Recv)
		case *ast.TypeSpec:
			check(x.Name)
		case *ast.ValueSpec:
			for _, name := range x.Names {
				check(name)
			}
		case *ast.FuncType:
			check_field_list(x.Params)
			check_field_list(x.Results)
		case *ast.StructType:
			check_field_list(x.Fields)
		case *ast.InterfaceType:
			check_field_list(x.Methods)
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE {
				for _, lhs := range x.Lhs {
					if identifier, ok := lhs.(*ast.Ident); ok {
						check(identifier)
					}
				}
			}
		}
		return true
	})
	return diags
}

// Reports whether function_declaration is a method whose name + signature
// satisfies a known stdlib interface (fs.FS.Open, fs.ReadFileFS.ReadFile,
// fs.DirEntry.IsDir, …). Stdlib interface names are conventionally
// PascalCase (no underscores) and can't be renamed; check_casing exempts
// them so test fixtures can implement these interfaces without lint
// flagging their method names.
func check_casing_method_satisfies_stdlib(function_declaration *ast.FuncDecl) (yes bool) {

	if function_declaration.Recv == nil {
		return false
	}
	params := check_unnecessary_method_field_list_types(function_declaration.Type.Params)
	results := check_unnecessary_method_field_list_types(function_declaration.Type.Results)
	return check_unnecessary_method_matches_stdlib(
		&check_unnecessary_method_matches_stdlib_input{
			Name:    function_declaration.Name.Name,
			Params:  strings.Join(params, ","),
			Results: strings.Join(results, ","),
		})
}

func check_named_returns(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	ast.Inspect(file, func(n ast.Node) (descend bool) {
		ft, ok := n.(*ast.FuncType)
		if !ok {
			return true
		}
		if ft.Results == nil {
			return true
		}
		for _, f := range ft.Results.List {
			if len(f.Names) == 0 {
				diags = append(diags, Diagnostic{
					Position: file_set.Position(f.Pos()),
					Message: fmt.Sprintf(
						"unnamed return type: %s", types.ExprString(
							f.Type)),
				})
			}
		}
		return true
	})
	return diags
}

// Shared exit-postcondition for per-file checks: diagnostics and declarations
// are each within their per-call budget, and the impossible (diags, decls)
// safety-cap corners are excluded.

// A bare `return` inside a value-returning function silently relies on the
// current values of its named returns; the actual return values vanish from
// the reader's view at the worst possible moment (the function exit), and the
// idiom interacts subtly with defers that mutate the named slots. Once a
// function declares value returns, every `return` must spell its values
// explicitly. Void functions are unaffected — guard-clause `if c { return }`
// patterns remain idiomatic.
//
// Implementation: a single ast.Inspect pass with a stack of enclosing
// *ast.FuncType (pushed on FuncDecl/FuncLit entry, popped on the matching
// nil-exit). At each ReturnStmt, the innermost enclosing function's
// signature decides whether the return is naked. Closures are checked
// against their own signature, not the outer function's, per Go semantics.
func check_no_naked_return(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	var function_stack []*ast.FuncType
	var push_history []bool
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		if n == nil {
			top := len(push_history) - 1
			pushed := push_history[top]
			push_history = push_history[:top]
			if pushed {
				function_stack = function_stack[:len(function_stack)-1]
			}
			return true
		}
		pushed := false
		switch x := n.(type) {
		case *ast.FuncDecl:
			function_stack = append(function_stack, x.Type)
			pushed = true
		case *ast.FuncLit:
			function_stack = append(function_stack, x.Type)
			pushed = true
		case *ast.ReturnStmt:
			if len(function_stack) == 0 {
				break
			}
			if len(x.Results) > 0 {
				break
			}
			signature := function_stack[len(function_stack)-1]
			if signature.Results == nil {
				break
			}
			if len(signature.Results.List) == 0 {
				break
			}
			diags = append(diags, Diagnostic{
				Position: file_set.Position(x.Return),
				Message:  "naked return is banned",
			})
		}
		push_history = append(push_history, pushed)
		return true
	})
	return diags
}

func check_line_character_count(
	file_set *token.FileSet, file *ast.File, source []byte,
) (diags []Diagnostic) {

	tok_file := file_set.File(file.Pos())
	filename := ""
	if tok_file != nil {
		filename = tok_file.Name()
	}
	// Import lines are exempt: a module path is a single unbreakable token, so a
	// long one cannot be wrapped to satisfy the column limit.
	import_lines := map[int]bool{}
	for _, import_specification := range file.Imports {
		import_lines[file_set.Position(import_specification.Pos()).Line] = true
	}
	// Lines spanned by a backtick raw string literal are exempt: the bytes are
	// data the author cannot rewrap (a multi-line snapshot, an embedded
	// template), so the column cap is meaningless there.
	raw_string_lines := raw_string_literal_lines(file_set, file)
	line_number := 1
	column := 0
	emit := func(n int) {
		if import_lines[line_number] {
			return
		}
		if raw_string_lines[line_number] {
			return
		}
		diags = append(diags, Diagnostic{
			Position: token.Position{
				Filename: filename,
				Line:     line_number,
				Column:   line_chars_max + 1,
			},
			Message: fmt.Sprintf("line is %d chars (max %d)", n, line_chars_max),
		})
	}
	for len(source) > 0 {
		r, size := utf8.DecodeRune(source)
		source = source[size:]
		if r == '\n' {
			if column > line_chars_max {
				emit(column)
			}
			line_number++
			column = 0
			continue
		}
		if r == '\t' {
			column += tab_width
			continue
		}
		column++
	}
	if column > line_chars_max {
		emit(column)
	}
	return diags
}

// Returns the set of source lines wholly or partly covered by a backtick raw
// string literal. The column cap exempts these: their content is verbatim data
// the author cannot wrap. A long non-string tail sharing such a line slips by —
// the rare cost of a whole-line exemption.
func raw_string_literal_lines(
	file_set *token.FileSet, file *ast.File,
) (lines map[int]bool) {

	lines = map[int]bool{}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		basic_literal, is_basic_literal := n.(*ast.BasicLit)
		if !is_basic_literal {
			return true
		}
		if basic_literal.Kind != token.STRING {
			return true
		}
		if !strings.HasPrefix(basic_literal.Value, "`") {
			return true
		}
		first := file_set.Position(basic_literal.Pos()).Line
		last := file_set.Position(basic_literal.End()).Line
		for line := first; line <= last; line++ {
			lines[line] = true
		}
		return true
	})
	return lines
}

// TigerStyle: compound conditions hide cases. Split into nested if/else trees so each branch
// is verifiable in isolation. Only the top-level operator is flagged — `&&`/`||` deep inside a
// subexpression (e.g. a function call arg) doesn't make the if itself compound.
func check_compound_if(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	unwrap := func(e ast.Expr) (output ast.Expr) {
		for step := 0; ; step++ {
			pe, ok := e.(*ast.ParenExpr)
			if !ok {
				return e
			}
			e = pe.X
		}
	}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		if_statement, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		be, ok := unwrap(if_statement.Cond).(*ast.BinaryExpr)
		if !ok {
			return true
		}
		is_logical := false
		if be.Op == token.LAND {
			is_logical = true
		}
		if be.Op == token.LOR {
			is_logical = true
		}
		if is_logical {
			diags = append(diags, Diagnostic{
				Position: file_set.Position(if_statement.Cond.Pos()),
				Message: fmt.Sprintf(
					"compound if condition (%s) — "+
						"split into nested ifs", be.Op),
			})
		}
		return true
	})
	return diags
}

func check_function_line_count(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	check := func(pos, lbrace, rbrace token.Pos, label string) {
		position := file_set.Position(pos)
		lbrace_position := file_set.Position(lbrace)
		rbrace_position := file_set.Position(rbrace)
		if !lbrace_position.IsValid() {
			return
		}
		if !rbrace_position.IsValid() {
			return
		}
		start := lbrace_position.Line
		end := rbrace_position.Line
		line_count := end - start + 1
		if line_count > function_lines_max {
			diags = append(diags, Diagnostic{
				Position: position,
				Message: fmt.Sprintf(
					"%s is %d lines (max %d)",
					label, line_count, function_lines_max),
			})
		}
	}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Body != nil {
				check(x.Pos(), x.Body.Lbrace, x.Body.Rbrace, "function")
			}
		case *ast.FuncLit:
			if x.Body != nil {
				check(x.Pos(), x.Body.Lbrace, x.Body.Rbrace, "function literal")
			}
		}
		return true
	})
	return diags
}

// Check_Source parses a single source buffer and returns diagnostics
// from the per-file checks. The filesystem and cross-file doctrine
// tiers are not exercised — callers that need those use
// Check_File_System or Main.
func Check_Source(filename string, source any) (diags []Diagnostic, err error) {
	file_set := token.NewFileSet()
	file, err := parser.ParseFile(
		file_set, filename, source, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var source_bytes []byte
	switch s := source.(type) {
	case []byte:
		source_bytes = s
	case string:
		source_bytes = []byte(s)
	}
	// Check_Source is the single-file API used by tests that don't exercise
	// the var-Default exemption nor the vocabulary table: a nil allowlist allows
	// no package to declare a var Default, and a nil word-replacements table
	// disables the vocabulary check (it has no config to read from). Both are the
	// strict, dependency-free defaults for single-file checks.
	return Check_File(file_set, file, source_bytes, nil, nil), nil
}

// Check_File_System_Input bundles the per-run dependencies for the
// filesystem tier: the fs.FS view of the workspace, the OS-side
// Readlink the stream-tier symlink check needs, the tracked-paths
// filter, and the parallelism cap.
type Check_File_System_Input struct {
	// Fsys is the workspace tree to scan; all paths resolve against it.
	Fsys fs.FS
	// Root is the directory within Fsys to walk from, "." for the whole tree.
	Root string
	// Root_Directory is the matching OS path, needed because fs.FS has no
	// symlink primitive; empty self-disables the symlink check.
	Root_Directory string
	// Tracked restricts the walk to this path set when non-nil; nil scans
	// everything (see Main_Input.Tracked).
	Tracked map[string]bool
	// CPU_Count caps parse and check parallelism; 0 degrades to serial.
	CPU_Count int
	// Readlink reads a symlink's target for the symlink check; nil disables it.
	Readlink func(name string) (target string, err error)
	// Scope is the package argument the linter was pointed at (relative to
	// Fsys root, empty for a whole-workspace run). The SPECIFICATION.md
	// coverage rule is enforced only for packages under this scope so that
	// `lint ./some/pkg` demands the file there, while a scopeless run does
	// not blanket-require it of every package in the tree.
	Scope string
	// Instrumentation_Packages is the lint.json instrumentation list forwarded
	// from Main_Input: write-only packages exempt from the var-Default ban and
	// the purity/determinism import bans. Threaded to the package-var,
	// transitive-purity, and deterministic checks.
	Instrumentation_Packages []string
	// Shared_Module is the shared library module's workspace-root-relative
	// directory, forwarded from Main_Input. It drives shared-vs-binary
	// classification in the module index.
	Shared_Module string
	// Deterministic_Packages is the lint.json deterministic tier list forwarded
	// from Main_Input: workspace-root-relative package directories whose pure
	// packages are held to the deterministic bans. Built into a set once per run
	// and threaded to check_deterministic.
	Deterministic_Packages []string
	// Word_Replacements is the lint.json word_replacements table: each tokenized,
	// lowercased word maps to its preferred expansions (an empty list bans the
	// word outright). Threaded to the vocabulary check via
	// make_check_names_vocabulary. nil disables the check (no config to read),
	// which is what the Check_Source single-file path passes.
	Word_Replacements map[string][]string
	// Ignore is the lint.json ignore list forwarded from Main_Input: gitignore-
	// style globs that trim the tracked scan set, so a matching path is invisible
	// to every tier. Applied once here against Tracked; with no Tracked set (the
	// non-git fallback) it is inert, like every other tracked-set filter.
	Ignore []string
}

// Check_File_System runs the stream tier, parses all Go files, and
// Runs every per-file and cross-file check across the workspace.
// Diagnostics from every tier are unioned into the returned slice.
func Check_File_System(input *Check_File_System_Input) (diags []Diagnostic, err error) {
	root := input.Root
	if root == "" {
		root = "."
	}
	cpu_count := input.CPU_Count
	if cpu_count < 1 {
		cpu_count = 1
	}
	// The lint.json ignore list trims the tracked scan set up front, so every
	// tier below (which keys off Tracked) skips the ignored paths with no
	// per-tier plumbing. This is how `ignore` extends the hardcoded global list.
	tracked := filter_ignored(input.Tracked, input.Ignore)
	directory_has_tracked := check_file_system_directory_index(tracked)

	// Discover every module before parsing: the roots decide which subtrees a
	// scoped run reads (parsing the rest is the work scope skips), and the
	// cross-file checks still resolve imports against the full set. The roots are
	// reused to build the index, so go.mod discovery walks the tree exactly once.
	module_roots, err := discover_module_roots(input.Fsys, directory_has_tracked)
	if err != nil {
		return nil, err
	}
	scan_prefixes := resolve_parse_prefixes(&resolve_parse_prefixes_input{
		Modules: module_roots, Scope: input.Scope, Shared_Module: input.Shared_Module})

	// Stream and AST tiers run in series; parse failures degrade to per-file
	// diagnostics. Scan_Prefixes bounds both: the stream walk skips out-of-scope
	// directories, so the go-path list it returns — and the parse and AST tiers
	// fed from it, where the run spends its time and memory — covers only scope.
	stream_diags, paths, err := check_file_system_stream(&check_file_system_stream_input{
		Fsys:                  input.Fsys,
		Root:                  root,
		Root_Directory:        input.Root_Directory,
		Tracked:               tracked,
		Directory_Has_Tracked: directory_has_tracked,
		Scan_Prefixes:         scan_prefixes,
		Readlink:              input.Readlink,
	})
	if err != nil {
		return nil, err
	}
	sources, err := check_file_system_read_files(input.Fsys, paths)
	if err != nil {
		return nil, err
	}
	parsed_files, parse_diags := check_file_system_parse_files(paths, sources, cpu_count)
	modules := build_module_index(module_roots, parsed_files, input.Shared_Module)
	return check_file_system_doctrine(&check_file_system_doctrine_input{
		Fsys:                     input.Fsys,
		Tracked:                  tracked,
		Parsed_Files:             parsed_files,
		Modules:                  modules,
		CPU_Count:                cpu_count,
		Stream_Diags:             stream_diags,
		Parse_Diags:              parse_diags,
		Instrumentation_Packages: input.Instrumentation_Packages,
		Word_Replacements:        input.Word_Replacements,
		Deterministic_Packages:   input.Deterministic_Packages,
		Scope:                    input.Scope,
		Scan_Prefixes:            scan_prefixes,
	}), nil
}

type check_file_system_doctrine_input struct {
	Fsys                     fs.FS
	Tracked                  map[string]bool
	Parsed_Files             []parsed_file
	Modules                  *module_index
	CPU_Count                int
	Stream_Diags             []Diagnostic
	Parse_Diags              []Diagnostic
	Instrumentation_Packages []string
	Word_Replacements        map[string][]string
	Deterministic_Packages   []string
	Scope                    string
	// Scan_Prefixes is the scope-narrowed parse set (resolve_parse_prefixes): the
	// directory subtrees this run actually parsed, or nil for a whole-workspace
	// run. The deterministic coverage check needs it to tell an out-of-scope entry
	// (a real package this run never parsed) from a genuine stale one.
	Scan_Prefixes []string
}

// Runs the AST and cross-file doctrine tiers over the parsed set and unions their
// diagnostics with the stream and parse diagnostics already collected. Split from
// Check_File_System so each half fits the length cap; the parsed set it receives
// is already scope-narrowed, while Tracked and Modules still span the workspace
// for the path-casing and import-resolution checks that need the full view.
func check_file_system_doctrine(
	input *check_file_system_doctrine_input,
) (output []Diagnostic) {

	parsed_files := input.Parsed_Files
	modules := input.Modules
	output = append([]Diagnostic{}, input.Stream_Diags...)
	output = append(output, input.Parse_Diags...)
	output = append(output, check_path_casing(input.Fsys, input.Tracked)...)
	output = append(output,
		check_file_system_run_checks(parsed_files, input.CPU_Count,
			input.Instrumentation_Packages, input.Word_Replacements)...)
	output = append(output, check_file_system_package_split(parsed_files)...)
	output = append(output, check_file_system_method_prefix(parsed_files)...)
	output = append(output, check_binary_module_layout(parsed_files, modules)...)
	output = append(output, check_binary_module_main_package(parsed_files, modules)...)
	output = append(output,
		check_binary_module_internal_main(parsed_files, modules, input.Tracked)...)
	output = append(output, check_shared_library_no_internal(parsed_files, modules)...)
	output = append(output, check_shared_library_no_main_package(parsed_files, modules)...)
	output = append(output, check_library_tier_depth(parsed_files, modules)...)
	output = append(output,
		check_module_definition_and_location(input.Fsys, modules, input.Tracked)...)
	output = append(output, check_no_impure_stdlib(parsed_files, modules)...)
	output = append(output,
		check_transitive_purity(parsed_files, modules, input.Instrumentation_Packages)...)
	output = append(output, check_deterministic(&check_deterministic_input{
		Parsed_Files:    parsed_files,
		Modules:         modules,
		Packages:        input.Deterministic_Packages,
		Instrumentation: input.Instrumentation_Packages,
		Scan_Prefixes:   input.Scan_Prefixes,
	})...)
	output = append(output, check_time_import_gateway(parsed_files, modules)...)
	output = append(output, check_package_documentation_comment(parsed_files)...)
	return append(output,
		check_specification(input.Fsys, parsed_files, modules, input.Scope)...)
}

// Returns tracked with every path the lint.json ignore globs match dropped, so
// the trimmed set carries the ignore decision to every tier that keys off it.
// Returns tracked unchanged when it is nil (the non-git fallback has nothing to
// trim) or when there are no patterns, so the common no-ignore run allocates
// nothing. A tracked entry is a file, so the matcher tests it with
// key_is_directory false; it still tests every ancestor prefix, which is what
// lets a directory entry match everything beneath it.
func filter_ignored(tracked map[string]bool, ignore []string) (kept map[string]bool) {
	// The non-git fallback has no tracked set to trim, and the common run lists
	// no ignore globs; either way the input passes through unallocated.
	if tracked == nil {
		return tracked
	}
	if len(ignore) == 0 {
		return tracked
	}
	patterns := make([]glob_pattern, 0, len(ignore))
	for _, raw := range ignore {
		patterns = append(patterns, parse_glob_pattern(raw))
	}
	kept = make(map[string]bool, len(tracked))
	for p := range tracked {
		if glob_patterns_match(p, false, patterns) {
			continue
		}
		kept[p] = true
	}
	return kept
}

// Returns the set of directories that contain at least one tracked path,
// so walkers can SkipDir entire .gitignored subtrees instead of descending
// and rejecting every file one-by-one. Returns nil when tracked is nil so
// callers can compare against nil to disable filtering.
func check_file_system_directory_index(tracked map[string]bool) (output map[string]bool) {
	if tracked == nil {
		return nil
	}
	output = make(map[string]bool, len(tracked))
	for p := range tracked {
		for step := 0; ; step++ {
			i := strings.LastIndexByte(p, '/')
			if i < 0 {
				break
			}
			p = p[:i]
			if output[p] {
				break
			}
			output[p] = true
		}
	}
	return output
}

// Returns diagnostics for the git-history tier. Empty when the tier is
// disabled. When Main_Reference_Absent is set, surfaces a single actionable
// diagnostic instead of running per-commit checks — without main locally,
// the .. ranges that drive merge/fixup detection collapse to nothing and
// would silently pass on shallow CI checkouts. Subtree merges are exempted
// from the no-merge-commits rule because `git subtree add/pull` legitimately
// produces a merge commit as its primary mode of operation; there's no
// rebase equivalent. Exported as the git-history seam: the external lint_test
// package drives it directly, since Check_File_System never touches commits.
func Git_Input_Check(input Git_Input) (diags []Diagnostic) {
	// Main_Reference_Absent only set when Enabled is true (see main_load_git);
	// (Enabled=false, Main_Reference_Absent=true) is unreachable by construction.
	if !input.Enabled {
		return nil
	}
	if input.Main_Reference_Absent {
		return []Diagnostic{{
			Position: token.Position{Filename: "<git>"},
			Name:     "git-main-ref",
			Want:     "main ref reachable from HEAD",
			Message: "main ref not found; fetch main or " +
				"set actions/checkout fetch-depth: 0",
		}}
	}
	diags = append(diags, git_input_check_merge_diagnostics(input.Merge_Commits)...)
	diags = append(diags, git_input_check_non_merge_diagnostics(input.Non_Merge_Commits)...)
	return diags
}

// Flags each merge commit on the branch (rebase-instead violation) plus any
// over-length subject. Subtree merges are exempt. Split out of Git_Input_Check
// so each commit-slice carries its boundary coverage in a function that fits
// the length cap.
func git_input_check_merge_diagnostics(commits []Git_Commit) (diags []Diagnostic) {
	for _, c := range commits {
		if c.Subject == "" {
			continue
		}
		filename := "<git:" + git_input_check_short_hash(c.Hash) + ">"
		if len(c.Subject) > commit_subject_chars_max {
			diags = append(diags, Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "commit-subject-length",
				Want: fmt.Sprintf(
					"subject ≤ %d chars", commit_subject_chars_max),
				Message: fmt.Sprintf(
					"commit subject is %d chars (max %d)",
					len(c.Subject), commit_subject_chars_max),
			})
			// Helpers below assert subject ≤ commit_subject_chars_max as
			// a precondition; over-limit subjects are fully diagnosed by
			// the length entry above, so short-circuit before calling them.
			continue
		}
		if git_input_check_is_subtree_merge_subject(c.Subject) {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: token.Position{Filename: filename},
			Name:     "no-merge-commits",
			Want: "rebase onto main: git fetch origin main && " +
				"git rebase origin/main",
			Message: "merge commit on branch: " + c.Subject,
		})
	}
	return diags
}

// Flags fixup commits (autosquash-instead) and non-conventional subjects on
// the branch, plus any over-length subject. Split out of Git_Input_Check for
// the same length-cap reason as the merge variant.
func git_input_check_non_merge_diagnostics(commits []Git_Commit) (diags []Diagnostic) {
	for _, c := range commits {
		if c.Subject == "" {
			continue
		}
		if len(c.Subject) > commit_subject_chars_max {
			filename := "<git:" + git_input_check_short_hash(c.Hash) + ">"
			diags = append(diags, Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "commit-subject-length",
				Want: fmt.Sprintf(
					"subject ≤ %d chars", commit_subject_chars_max),
				Message: fmt.Sprintf(
					"commit subject is %d chars (max %d)",
					len(c.Subject), commit_subject_chars_max),
			})
			continue
		}
		if git_input_check_is_fixup_subject(c.Subject) {
			filename := "<git:" + git_input_check_short_hash(c.Hash) + ">"
			diags = append(diags, Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "no-fixup-commits",
				Want:     "autosquash: git rebase -i --autosquash origin/main",
				Message:  "fixup commit on branch: " + c.Subject,
			})
			// Fixup subjects aren't conventional by construction (e.g.
			// `fixup! feat: foo`); skip the conventional check so they
			// don't double-flag. The autosquash that removes the fixup
			// also removes the violation.
			continue
		}
		if !conventional_commit_re.MatchString(c.Subject) {
			filename := "<git:" + git_input_check_short_hash(c.Hash) + ">"
			diags = append(diags, Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "conventional-commits",
				Want: "subject like: type(scope)?!?: description " +
					"(https://www.conventionalcommits.org/)",
				Message: "non-conventional commit subject: " + c.Subject,
			})
		}
	}
	return diags
}

// Matches the default subjects that `git subtree add` and `git subtree pull`
// produce. Both forms are documented in git-subtree(1) and have remained
// stable for years; commits authored by the porcelain match exactly.
// Hand-authored subtree merges with custom messages aren't recognised and
// will trip the no-merge-commits rule — intentional, since custom-worded
// merges are indistinguishable from regular merges.
func git_input_check_is_subtree_merge_subject(subject string) (yes bool) {
	if strings.HasPrefix(subject, "Add '") {
		if strings.Contains(subject, "' from commit '") {
			return true
		}
	}
	if strings.HasPrefix(subject, "Merge commit '") {
		if strings.Contains(subject, "' as '") {
			return true
		}
	}
	return false
}

// Matches commit subjects that should have been autosquashed before merge.
// Two families: the literal fixup!/squash! prefixes that `git commit --fixup`
// produces, and review-comment phrasings that show up when people address
// feedback in a follow-up commit instead of amending. The phrasing checks
// are conjunctive (verb + noun + "review") so isolated mentions of "review"
// or "comment" in unrelated subjects don't get caught.
func git_input_check_is_fixup_subject(subject string) (yes bool) {
	if strings.HasPrefix(subject, "fixup!") {
		return true
	}
	if strings.HasPrefix(subject, "squash!") {
		return true
	}
	s := strings.ToLower(subject)
	has_review := strings.Contains(s, "review")
	has_address := strings.Contains(s, "address")
	has_apply := strings.Contains(s, "apply")
	has_action := has_address || has_apply
	has_comment := strings.Contains(s, "comment")
	has_feedback := strings.Contains(s, "feedback")
	has_nit := strings.Contains(s, "nit")
	has_target := has_comment || has_feedback || has_nit
	if has_review {
		if has_action {
			if has_target {
				return true
			}
		}
	}
	if strings.Contains(s, "cr comment") {
		return true
	}
	if strings.Contains(s, "code review comment") {
		return true
	}
	if strings.Contains(s, "review fix") {
		return true
	}
	if strings.Contains(s, "review nit") {
		return true
	}
	return false
}

// Truncates a git hash to git_short_hash_chars chars. Pass-through for
// already-short or malformed inputs so test fixtures don't have to supply
// full 40-char hashes.
func git_input_check_short_hash(h string) (s string) {
	if len(h) > git_short_hash_chars {
		return h[:git_short_hash_chars]
	}
	return h
}

// File-fragmentation check. Splitting code across many tiny files makes a
// package harder to read top-to-bottom and forces readers to chase symbols
// across the filesystem. The rule: per package, expected_max files =
// ceil(total_lines / lines_per_file_max). Source files, test files,
// specification_test.go, and each distinct build-tag constraint form
// independent groups (a Linux-only file and a generic file genuinely have
// to live separately). SLOC is total lines per the user's directive —
// comments and blanks count.
type package_group_key struct {
	Directory             string
	Is_Test               bool
	Is_Specification_Test bool
	Build                 string
}

type package_group_state struct {
	Files []parsed_file
	Lines int
}

func check_file_system_package_split(parsed_files []parsed_file) (diags []Diagnostic) {
	groups := map[package_group_key]*package_group_state{}
	for _, pf := range parsed_files {
		key := package_group_key{
			Directory:             path.Dir(pf.Path),
			Is_Test:               strings.HasSuffix(pf.Path, "_test.go"),
			Is_Specification_Test: path.Base(pf.Path) == "specification_test.go",
			Build:                 check_file_system_package_split_build_key(pf.File),
		}
		st := groups[key]
		if st == nil {
			st = &package_group_state{}
			groups[key] = st
		}
		st.Files = append(st.Files, pf)
		tok := pf.File_Set.File(pf.File.Pos())
		if tok != nil {
			st.Lines += tok.LineCount()
		}
	}
	keys := make([]package_group_key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) (less bool) {
		if keys[i].Directory != keys[j].Directory {
			return keys[i].Directory < keys[j].Directory
		}
		if keys[i].Is_Test != keys[j].Is_Test {
			return !keys[i].Is_Test
		}
		return keys[i].Build < keys[j].Build
	})
	for _, key := range keys {
		st := groups[key]
		files_max := (st.Lines + lines_per_file_max - 1) / lines_per_file_max
		if files_max < 1 {
			files_max = 1
		}
		if len(st.Files) == files_max {
			continue
		}
		diags = append(diags, package_group_key_diag(key, st, files_max))
	}
	return diags
}

func package_group_key_diag(
	key package_group_key,
	st *package_group_state,
	files_max int,
) (diag Diagnostic) {

	label := "source"
	if key.Is_Specification_Test {
		label = "specification_test"
	} else if key.Is_Test {
		label = "test"
	}
	build_suffix := ""
	if key.Build != "" {
		build_suffix = fmt.Sprintf(" under build constraint %q", key.Build)
	}
	first := st.Files[0]
	return Diagnostic{
		Position: token.Position{Filename: first.Path, Line: 1, Column: 1},
		Message: fmt.Sprintf(
			"package %s in %s has %d %s files%s totaling %d lines; "+
				"want %d (one file per %d lines)",
			first.File.Name.Name, key.Directory, len(st.Files), label, build_suffix,
			st.Lines, files_max, lines_per_file_max,
		),
	}
}

// Inline Always-calls share the FIRST Cross_Product because the
// chain-credit gate skips inner-call processing on subsequent
// Cross_Products in the same defer frame.

// Build-constraint key for grouping. Uses go/build/constraint, the canonical
// parser, so equivalent expressions ("linux && amd64" vs "amd64 && linux"
// stay distinct in the raw text but the AST stringification normalizes form).
// Only //go:build lines preceding the package clause are considered; in-body
// comments are ignored.
func check_file_system_package_split_build_key(file *ast.File) (key string) {

	for _, g := range file.Comments {
		if g.End() >= file.Package {
			break
		}
		for _, c := range g.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expression, err := constraint.Parse(c.Text)
			if err != nil {
				continue
			}
			return expression.String()
		}
	}
	return ""
}

// Free functions whose first parameter is a named type declared in the same
// package must be named `<Type>_<verb>`. The rule recovers the grouping
// affordance that methods would have provided: check_unnecessary_method bans
// non-stdlib methods and forces the receiver onto the first parameter, but
// without a naming convention the resulting free function loses its visible
// association with the type. This is Odin's convention applied to Go:
// `func entity_update(e Entity)` over `func update(e Entity)`.
//
// Same-package visibility requires cross-file context, so the check runs at
// the FS level rather than as a per-file check_function. Groups parsed_files by
// (Dir, Package_Name, Build) — _test.go files declared as `package foo_test`
// form a distinct group from `package foo` and cannot see foo's unexported
// types, so package name is part of the key. Only bare *ast.Ident and
// generic instances (*ast.IndexExpr / *ast.IndexListExpr over *ast.Ident)
// trigger; pointers, slices, maps, channels, ellipsis, and selectors are
// not unwrapped. Receivers (methods) are skipped — check_unnecessary_method
// already covers them.
func check_file_system_method_prefix(parsed_files []parsed_file) (diags []Diagnostic) {
	type key struct {
		Dir     string
		Package string
		Build   string
	}
	groups := map[key][]parsed_file{}
	for _, pf := range parsed_files {
		k := key{
			Dir:     path.Dir(pf.Path),
			Package: pf.File.Name.Name,
			Build:   check_file_system_package_split_build_key(pf.File),
		}
		groups[k] = append(groups[k], pf)
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) (less bool) {
		if keys[i].Dir != keys[j].Dir {
			return keys[i].Dir < keys[j].Dir
		}
		if keys[i].Package != keys[j].Package {
			return keys[i].Package < keys[j].Package
		}
		return keys[i].Build < keys[j].Build
	})
	for _, k := range keys {
		diags = append(diags, check_file_system_method_prefix_group(groups[k])...)
	}
	return diags
}

func check_file_system_method_prefix_group(files []parsed_file) (diags []Diagnostic) {
	declared := check_file_system_method_prefix_group_declared(files)
	if len(declared) == 0 {
		return nil
	}
	for _, pf := range files {
		diags = append(diags,
			check_file_system_method_prefix_group_for_file(declared, pf)...)
	}
	sort.Slice(diags, func(i, j int) (less bool) {
		if diags[i].Position.Filename != diags[j].Position.Filename {
			return diags[i].Position.Filename < diags[j].Position.Filename
		}
		return diags[i].Position.Line < diags[j].Position.Line
	})
	return diags
}

// Returns the set of type names declared across files; used by the prefix
// check to scope its "free function over a custom type" search.
func check_file_system_method_prefix_group_declared(
	files []parsed_file,
) (declared map[string]bool) {
	declared = map[string]bool{}
	for _, pf := range files {
		for _, declaration := range pf.File.Decls {
			gd, ok := declaration.(*ast.GenDecl)
			if !ok {
				continue
			}
			if gd.Tok != token.TYPE {
				continue
			}
			for _, specification := range gd.Specs {
				type_specification, is_type_specification :=
					specification.(*ast.TypeSpec)
				if !is_type_specification {
					continue
				}
				declared[type_specification.Name.Name] = true
			}
		}
	}
	return declared
}

// Emits per-file prefix-violation diagnostics for the free functions whose
// first parameter type is in `declared`.
func check_file_system_method_prefix_group_for_file(
	declared map[string]bool, pf parsed_file,
) (diags []Diagnostic) {
	for _, declaration := range pf.File.Decls {
		function_declaration, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function_declaration.Recv != nil {
			continue
		}
		if function_declaration.Type.Params == nil {
			continue
		}
		if len(function_declaration.Type.Params.List) == 0 {
			continue
		}
		first_parameter := function_declaration.Type.Params.List[0]
		type_name := check_file_system_method_prefix_group_first_parameter_type(
			first_parameter.Type)
		if type_name == "" {
			continue
		}
		if !declared[type_name] {
			continue
		}
		// Constructor-input exception: type named `<FuncName>_Input` /
		// `<func_name>_input` is named after the function, not vice versa.
		if type_name == function_declaration.Name.Name+"_Input" {
			continue
		}
		if type_name == function_declaration.Name.Name+"_input" {
			continue
		}
		style := "snake_case"
		if unicode.IsUpper(rune(function_declaration.Name.Name[0])) {
			style = "Ada_Case"
		}
		prefix := suggest(&suggest_input{Name: type_name, Want: style})
		if function_declaration.Name.Name == prefix {
			continue
		}
		if strings.HasPrefix(function_declaration.Name.Name, prefix+"_") {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: pf.File_Set.Position(function_declaration.Name.Pos()),
			Name:     function_declaration.Name.Name,
			Want:     prefix + "_<verb>",
			Message: fmt.Sprintf(
				"rename %s -> %s_<verb>",
				function_declaration.Name.Name, prefix),
		})
	}
	return diags
}

// Extracts the base named type from a parameter expression. Bare
// identifiers, pointer receivers (`*T`), and generic instances over a bare
// identifier qualify — all three are canonical "receiver promoted to first
// param" shapes for methods that the linter forced into free-function form.
// Slices, maps, channels, ellipsis, function types, interfaces, and
// selectors (package-qualified types) are intentionally excluded: they are
// collection or external-package shapes, not method-receiver shapes.
func check_file_system_method_prefix_group_first_parameter_type(expression ast.Expr) (name string) {

	if star, is_star := expression.(*ast.StarExpr); is_star {
		expression = star.X
	}
	switch e := expression.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		if identifier, is_ident := e.X.(*ast.Ident); is_ident {
			return identifier.Name
		}
	case *ast.IndexListExpr:
		if identifier, is_ident := e.X.(*ast.Ident); is_ident {
			return identifier.Name
		}
	}
	return ""
}

// Reads every file concurrently — I/O bound, no goroutine cap.
func check_file_system_read_files(fsys fs.FS, paths []string) (sources [][]byte, err error) {
	sources = make([][]byte, len(paths))
	errs := make([]error, len(paths))
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		go func(i int, p string) {
			defer wg.Done()
			sources[i], errs[i] = fs.ReadFile(fsys, p)
		}(i, p)
	}
	wg.Wait()
	for _, e := range errs {
		if e != nil {
			return nil, e
		}
	}
	return sources, nil
}

// Parses files in parallel — CPU bound, capped at NumCPU. Per-file parse
// failures are returned as diagnostics rather than bubbling up as a single
// hard error: a Go file with conflict markers, for instance, can't parse,
// but the stream tier has already flagged the marker — we want both
// diagnostics visible and the rest of the AST tier to keep running on the
// files that did parse.
func check_file_system_parse_files(
	paths []string, sources [][]byte, cpu_count int,
) (parsed_files []parsed_file, parse_diags []Diagnostic) {

	results := make([]parsed_file, len(paths))
	diags := make([]Diagnostic, len(paths))
	had_err := make([]bool, len(paths))
	sem := make(chan struct{}, cpu_count)
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, p string, source []byte) {
			defer wg.Done()
			defer func() { <-sem }()
			file_set := token.NewFileSet()
			file, parse_err := parser.ParseFile(
				file_set, p, source,
				parser.SkipObjectResolution|parser.ParseComments)
			if parse_err != nil {
				diags[i] = Diagnostic{
					Position: token.Position{Filename: p},
					Message:  fmt.Sprintf("parse error: %v", parse_err),
				}
				had_err[i] = true
				return
			}
			results[i] = parsed_file{
				Path: p, File_Set: file_set, File: file, Source: source,
			}
		}(i, p, sources[i])
	}
	wg.Wait()
	for i := range paths {
		if had_err[i] {
			parse_diags = append(parse_diags, diags[i])
			continue
		}
		parsed_files = append(parsed_files, results[i])
	}
	return parsed_files, parse_diags
}

// Paths and sources have a 1:1 relationship by contract (caller
// passes matched slices), so (Hi paths, Lo sources) and (Lo
// paths, Hi sources) are logically impossible. Likewise (Hi
// parse_diags, Lo paths) — zero paths cannot produce parse
// diagnostics. The remaining Hi(parsed_files) / Hi(parse_diags)
// / Hi(paths) / Hi(sources) endpoints are the per-call budget
// safety caps — they bound runaway scans rather than working
// workspace sizes.

// Paths and sources are 1:1 by contract (each path has a matched
// source slot), so (Hi paths, Lo sources) and (Lo paths, Hi sources)
// are logically impossible. (Hi paths, Hi sources) sits at the
// per-call file-budget safety cap — that endpoint bounds runaway
// scans, not a working workspace size. (Lo paths, Lo sources, Hi cpu)
// requires a caller to pass cpu_count at the worker-pool safety cap
// (1024) with zero work; the worker pool would idle, so reaching it
// signals misconfigured input rather than a meaningful state.

// Module identity for a single Go module discovered under Fsys. The
// doctrine's layout rules need three things per file: which module owns
// it, whether that module is the shared imports, and which directories
// contain non-main Go packages (the "Go ancestor" set used by the
// library-tier-depth rule).
type module_information struct {
	Root              string
	Module_Path       string
	Is_Shared_Library bool
	Directory_Package map[string]string
}

// All-doctrine-checks input. Built once after parsing and threaded
// through the directory-level checks so module discovery is paid for
// at most once per Main invocation. Modules is sorted longest-Root
// first so File_To_Module resolution is a linear scan with the
// longest-prefix wins guarantee.
type module_index struct {
	Modules        []module_information
	File_To_Module map[string]int
}

var module_index_module_re = regexp.MustCompile(`(?m)^module\s+(\S+)`)

// Walks Fsys for go.mod files, parsing only the `module` line via regex (avoids
// pulling in golang.org/x/mod for one field), and returns one module_information
// per module discovered — Directory_Package allocated but empty, classification
// and ordering deferred to build_module_index. Split from build_module_index so
// the roots are known before any file is parsed: the scope-to-parse resolver
// needs them to decide which module subtrees to read, and reading the rest is
// the work a scoped run skips. When Fsys is rooted inside a module (no go.mod
// visible — typical when pointed at a subdirectory), no modules are discovered
// and every file later maps to -1, so the doctrine checks no-op on those files
// rather than reporting against a partial view of the workspace.
// directory_has_tracked prunes any directory holding no tracked file, so a
// gitignored stray go.mod — a gopls temp module under tmp/ mirrors a real
// module's path — is never discovered; undiscovered, it cannot shadow the real
// module's root in the longest-prefix import lookup and so break the
// instrumentation directory match. A nil index (the non-git fallback) prunes
// nothing, matching every other tracked-set filter.
func discover_module_roots(
	fsys fs.FS, directory_has_tracked map[string]bool,
) (modules []module_information, err error) {

	scratch := &module_index{}
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, walk_err error) (output error) {
		return build_module_index_walk(fsys, scratch, directory_has_tracked, p, d, walk_err)
	})
	if err != nil {
		return nil, err
	}
	return scratch.Modules, nil
}

// Classifies, orders, and binds parsed files to the modules discovered by
// discover_module_roots. Modules is sorted longest-Root first so File_To_Module
// resolution is a linear longest-prefix scan. A scoped run passes only the
// parsed subset; the resulting index still covers every module's Root (for
// import resolution) but its File_To_Module and Directory_Package describe only
// the files actually parsed — which is all the in-scope checks consult.
func build_module_index(
	modules []module_information, parsed_files []parsed_file, shared_module string,
) (index *module_index) {

	index = &module_index{
		Modules: modules, File_To_Module: make(map[string]int, len(parsed_files))}
	// Classify the shared library by its workspace-root-relative directory (the
	// module Root, e.g. "shared"), matching the slash-relative form used
	// by the rest of lint.json; every other module is a binary. An empty
	// shared_module (e.g. a test that doesn't set one) leaves every module a binary.
	// path.Clean so "./shared/" matches the cleaned module Root; guard the
	// empty case, since path.Clean("") is "." and would wrongly match a root module.
	shared_root := shared_module
	if shared_root != "" {
		shared_root = path.Clean(shared_root)
	}
	for i := range index.Modules {
		index.Modules[i].Is_Shared_Library = index.Modules[i].Root == shared_root
	}
	sort.Slice(index.Modules, func(i, j int) (less bool) {
		return len(index.Modules[i].Root) > len(index.Modules[j].Root)
	})
	for _, pf := range parsed_files {
		index.File_To_Module[pf.Path] = module_index_resolve(pf.Path, index.Modules)
	}
	// Directory_Package excludes test/main files (library-tier-depth rule).
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if pf.File.Name.Name == "main" {
			continue
		}
		module_index_number := index.File_To_Module[pf.Path]
		if module_index_number < 0 {
			continue
		}
		root := index.Modules[module_index_number].Root
		relative := pf.Path
		if root != "." {
			relative = strings.TrimPrefix(pf.Path, root+"/")
		}
		canonical_directory := module_index_canonicalize(path.Dir(relative))
		directory_package := index.Modules[module_index_number].Directory_Package
		if _, has := directory_package[canonical_directory]; !has {
			directory_package[canonical_directory] = pf.File.Name.Name
		}
	}
	return index
}

// Widens a scope argument to the module that must be parsed whole for it. The
// module-level doctrine checks (entry-point, layout, tier depth) reach the same
// verdict only when every file of a module is in view, so a scope pointing inside
// a module parses that whole module; the output filter then narrows diagnostics
// back to the scope. A scope matching no named module falls back to the root
// module when one exists (it owns everything), else to the scope subtree itself —
// files owned by no module resolve to -1 and the module-level checks no-op on
// them. An empty scope (the whole-workspace run) parses the root.
func resolve_scan_root(modules []module_information, scope string) (root string) {

	if scope == "" {
		return "."
	}
	best := ""
	root_module := false
	for i := range modules {
		module_root := modules[i].Root
		if module_root == "." {
			root_module = true
			continue
		}
		owns := scope == module_root
		if !owns {
			owns = strings.HasPrefix(scope, module_root+"/")
		}
		if owns {
			if len(module_root) > len(best) {
				best = module_root
			}
		}
	}
	if best != "" {
		return best
	}
	if root_module {
		return "."
	}
	return scope
}

type resolve_parse_prefixes_input struct {
	Modules       []module_information
	Scope         string
	Shared_Module string
}

// Returns the directory subtrees a scoped run must parse: the scope's own module
// (resolve_scan_root) plus the shared library module. The shared library is the
// one module a first-party file may import, so its packages must be parsed for
// the transitive-purity rule to classify a binary's imports of them — skipping it
// would fail open, the one regression this list exists to bar. A nil result means
// "parse everything" (the whole-workspace run). Prefixes are sorted so the walk
// order is deterministic.
func resolve_parse_prefixes(input *resolve_parse_prefixes_input) (prefixes []string) {

	if input.Scope == "" {
		return nil
	}
	set := map[string]bool{resolve_scan_root(input.Modules, input.Scope): true}
	shared_root := input.Shared_Module
	if shared_root != "" {
		shared_root = path.Clean(shared_root)
		for i := range input.Modules {
			if input.Modules[i].Root == shared_root {
				set[shared_root] = true
				break
			}
		}
	}
	prefixes = make([]string, 0, len(set))
	for prefix := range set {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)
	return prefixes
}

// Reports whether the walk should enter directory dir under a scoped run: it is
// one of the parse prefixes, sits beneath one, or is an ancestor of one (an
// ancestor must be descended through to reach the prefix below it). A nil prefix
// set — the whole-workspace run — admits every directory, as does a "." prefix.
// This narrows the stream walk, and through the go-path list it returns, the AST
// tier with it; the run's time and memory both follow the parse set.
func scan_prefixes_reach(prefixes []string, directory string) (reachable bool) {

	if prefixes == nil {
		return true
	}
	for _, prefix := range prefixes {
		if prefix == "." {
			return true
		}
		if directory == prefix {
			return true
		}
		if strings.HasPrefix(directory, prefix+"/") {
			return true
		}
		if strings.HasPrefix(prefix, directory+"/") {
			return true
		}
	}
	return false
}

// Handles one fs.WalkDir visit for build_module_index: skips vendored/hidden
// directories, and on each go.mod records the module root and import path.
func build_module_index_walk(
	fsys fs.FS, index *module_index, directory_has_tracked map[string]bool,
	p string, d fs.DirEntry, walk_err error,
) (output error) {
	if walk_err != nil {
		return walk_err
	}
	if d.IsDir() {
		if p != "." {
			if Ignored_Directory(p) {
				return fs.SkipDir
			}
			// Prune the same gitignored subtrees every other tier skips, so a stray
			// go.mod outside the tracked set is never discovered as a module.
			if directory_has_tracked != nil {
				if !directory_has_tracked[p] {
					return fs.SkipDir
				}
			}
		}
		return nil
	}
	if d.Name() != "go.mod" {
		return nil
	}
	content, read_err := fs.ReadFile(fsys, p)
	if read_err != nil {
		return read_err
	}
	match := module_index_module_re.FindSubmatch(content)
	if match == nil {
		return nil
	}
	module_path := string(match[1])
	// Is_Shared_Library is left at its zero value here and set in a post-walk
	// pass in build_module_index, where the configured shared module is in scope.
	index.Modules = append(index.Modules, module_information{
		Root:              path.Dir(p),
		Module_Path:       module_path,
		Directory_Package: make(map[string]string),
	})
	return nil
}

// Strips ^v[0-9]+$ segments from a slash-separated directory path so
// snap/v2/X is treated identically to snap/X. Major-version segments
// are Go module-versioning convention rather than real package tiers,
// and the doctrine's depth rules must see through them.
func module_index_canonicalize(directory string) (canonical string) {

	if directory == "." {
		return "."
	}
	segments := strings.Split(directory, "/")
	filtered := make([]string, 0, len(segments))
	for _, s := range segments {
		if module_index_version_re.MatchString(s) {
			continue
		}
		filtered = append(filtered, s)
	}
	if len(filtered) == 0 {
		return "."
	}
	return strings.Join(filtered, "/")
}

var module_index_version_re = regexp.MustCompile(`^v[0-9]+$`)

func module_index_resolve(file_path string, modules []module_information) (index int) {

	for i, module := range modules {
		if module.Root == "." {
			return i
		}
		if file_path == module.Root {
			return i
		}
		if strings.HasPrefix(file_path, module.Root+"/") {
			return i
		}
	}
	return -1
}

// Binary modules confine all non-main source to internal/ so the module
// has no exported surface. Without the rule, an importable package
// could leak out of any binary and become a cross-module dependency
// the doctrine forbids. The check exempts the shared library (which is
// importable by design) and `package main` files (which can sit at any
// depth because Go itself bars importing them).
func check_binary_module_layout(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {

	seen := make(map[string]bool)
	for _, pf := range parsed_files {
		module_index_number := modules.File_To_Module[pf.Path]
		if module_index_number < 0 {
			continue
		}
		m := modules.Modules[module_index_number]
		if m.Is_Shared_Library {
			continue
		}
		if pf.File.Name.Name == "main" {
			continue
		}
		relative := pf.Path
		if m.Root != "." {
			relative = strings.TrimPrefix(pf.Path, m.Root+"/")
		}
		directory := path.Dir(relative)
		if check_binary_module_layout_is_legal(directory) {
			continue
		}
		key := m.Root + "\x00" + directory
		if seen[key] {
			continue
		}
		seen[key] = true
		diags = append(diags, Diagnostic{
			Position: token.Position{Filename: pf.Path, Line: 1, Column: 1},
			Name:     "binary-module-layout",
			Want:     fmt.Sprintf("non-main packages live under %s/internal/", m.Root),
			Message: binary_module_layout_message(&binary_module_layout_message_input{
				Root:      m.Root,
				Directory: directory,
			}),
		})
	}
	return diags
}

type binary_module_layout_message_input struct {
	Root      string
	Directory string
}

func binary_module_layout_message(input *binary_module_layout_message_input) (message string) {
	destination := path.Join(input.Root+"/internal", input.Directory)
	if input.Root == "." {
		destination = "./" + destination
	}
	return fmt.Sprintf("move %s -> %s", input.Directory, destination)
}

// True for directories whose first segment is `internal` — the only
// legal home for non-main packages in a binary module under the
// doctrine. "." (the module root) is illegal for non-main code:
// `package main` is handled by the caller's earlier short-circuit.
func check_binary_module_layout_is_legal(directory string) (legal bool) {

	if directory == "." {
		return false
	}
	segments := strings.Split(directory, "/")
	return segments[0] == "internal"
}

// A binary module exposes exactly one entry point, so its single main
// package lives at the module root. Scattering binaries under cmd/ — the
// GOPATH-era convention — multiplies entry points and invites the
// importable-package leak the layout exists to prevent. A main package
// anywhere but the root is reported; since a directory holds one package,
// pinning every main to the root also caps the module at one.
func check_binary_module_main_package(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {

	seen := make(map[string]bool)
	for _, pf := range parsed_files {
		if pf.File.Name.Name != "main" {
			continue
		}
		module_index_number := modules.File_To_Module[pf.Path]
		if module_index_number < 0 {
			continue
		}
		m := modules.Modules[module_index_number]
		if m.Is_Shared_Library {
			continue
		}
		relative := pf.Path
		if m.Root != "." {
			relative = strings.TrimPrefix(pf.Path, m.Root+"/")
		}
		directory := path.Dir(relative)
		if directory == "." {
			continue
		}
		key := m.Root + "\x00" + directory
		if seen[key] {
			continue
		}
		seen[key] = true
		diags = append(diags, Diagnostic{
			Position: token.Position{Filename: pf.Path, Line: 1, Column: 1},
			Name:     "binary-module-main-package",
			Want:     "the single main package sits at the module root, no cmd/",
			Message: fmt.Sprintf(
				"binary module %q places its main package at %q; the main "+
					"package must sit at the module root, no cmd/ directory",
				m.Module_Path, directory,
			),
		})
	}
	return diags
}

// A binary module exposes its entry point as a single free func Main in its
// top-level internal/ package — the composition tier that package main's
// thin main() delegates to. Pinning it there keeps the real logic out of
// package main (which Go bars from being imported, hence from being tested)
// and gives every binary one auditable seam. Zero such functions means the
// logic has nowhere conformant to live; more than one means the single
// entry point has fractured. The shared library is exempt: it is imported,
// never executed, so it owns no entry point. Modules with no visible go.mod
// resolve to -1 and are skipped, matching the other module-level checks.
// The Tracked filter drops modules whose go.mod is not first-party, mirroring
// check_module_definition_and_location, so a whole-repo run never demands an
// entry point from a toolchain copy; top-level third_party/ is pruned before
// module discovery, so vendored modules never enter the index at all.
func check_binary_module_internal_main(
	parsed_files []parsed_file, modules *module_index, tracked map[string]bool,
) (diags []Diagnostic) {

	counts := make([]int, len(modules.Modules))
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		module_index_number := modules.File_To_Module[pf.Path]
		if module_index_number < 0 {
			continue
		}
		m := modules.Modules[module_index_number]
		relative := pf.Path
		if m.Root != "." {
			relative = strings.TrimPrefix(pf.Path, m.Root+"/")
		}
		if path.Dir(relative) != "internal" {
			continue
		}
		counts[module_index_number] += check_binary_module_internal_main_count(pf.File)
	}
	want := "exactly one func Main in internal/ per binary module"
	for i, m := range modules.Modules {
		if m.Is_Shared_Library {
			continue
		}
		module_file_path := "go.mod"
		if m.Root != "." {
			module_file_path = m.Root + "/go.mod"
		}
		if tracked != nil {
			if !tracked[module_file_path] {
				continue
			}
		}
		position := token.Position{Filename: module_file_path, Line: 1, Column: 1}
		if counts[i] == 0 {
			diags = append(diags, Diagnostic{
				Position: position,
				Name:     "binary-module-internal-main",
				Want:     want,
				Message: fmt.Sprintf(
					"binary module %q declares no func Main in internal/",
					m.Module_Path),
			})
			continue
		}
		if counts[i] > 1 {
			diags = append(diags, Diagnostic{
				Position: position,
				Name:     "binary-module-internal-main",
				Want:     want,
				Message: fmt.Sprintf(
					"binary module %q declares multiple func Main in internal/",
					m.Module_Path),
			})
		}
	}
	return diags
}

// Counts free, top-level functions named Main in one parsed file. A method
// named Main does not count: the entry point is a package-level function the
// thin main() can call directly, not a behavior bound to some type.
func check_binary_module_internal_main_count(file *ast.File) (count int) {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function.Recv != nil {
			continue
		}
		if function.Name.Name != "Main" {
			continue
		}
		count++
	}
	return count
}

// The shared library exists to be imported by binaries; any internal/
// subtree would hide part of its surface and defeat the layering.
// Reported once per offending directory (the first `internal` segment
// found in any file's path), attributed to the earliest-seen file
// inside that directory so the diagnostic has a real location.
func check_shared_library_no_internal(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {

	seen := make(map[string]bool)
	for _, pf := range parsed_files {
		module_index_number := modules.File_To_Module[pf.Path]
		if module_index_number < 0 {
			continue
		}
		m := modules.Modules[module_index_number]
		if !m.Is_Shared_Library {
			continue
		}
		segments := strings.Split(pf.Path, "/")
		for i, s := range segments {
			if s != "internal" {
				continue
			}
			internal_directory := strings.Join(segments[:i+1], "/")
			if seen[internal_directory] {
				break
			}
			seen[internal_directory] = true
			diags = append(diags, Diagnostic{
				Position: token.Position{Filename: pf.Path, Line: 1, Column: 1},
				Name:     "shared-imports-no-internal",
				Want:     "shared library is fully exposed; no internal/ subtree",
				Message: fmt.Sprintf(
					"shared library forbids internal/ directories; remove %q",
					internal_directory),
			})
			break
		}
	}
	return diags
}

// The shared library is imported, never executed, so it declares no
// package main: an entry point belongs in a binary module, and a main
// package here is unreachable anyway — Go bars importing it — so it is
// dead weight the layout forbids outright. Reported once per offending
// directory.
func check_shared_library_no_main_package(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {

	seen := make(map[string]bool)
	for _, pf := range parsed_files {
		if pf.File.Name.Name != "main" {
			continue
		}
		module_index_number := modules.File_To_Module[pf.Path]
		if module_index_number < 0 {
			continue
		}
		m := modules.Modules[module_index_number]
		if !m.Is_Shared_Library {
			continue
		}
		key := m.Root + "\x00" + path.Dir(pf.Path)
		if seen[key] {
			continue
		}
		seen[key] = true
		diags = append(diags, Diagnostic{
			Position: token.Position{Filename: pf.Path, Line: 1, Column: 1},
			Name:     "shared-library-no-main",
			Want:     "shared library declares no package main",
			Message: fmt.Sprintf(
				"shared library %q forbids package main; move the entry "+
					"point to a binary module", m.Module_Path),
		})
	}
	return diags
}

// Caps how deep non-main packages may nest before they stop being a
// recognizable library tier. The rule: at most one non-main Go
// ancestor in the same module, after canonicalizing major-version
// segments (v2, v3, …) which are module-versioning convention rather
// than real package layers. The deepest legal position is the
// composition tier — the only place where impure-stdlib binding is
// permitted.
func check_library_tier_depth(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {

	seen := make(map[string]bool)
	for _, pf := range parsed_files {
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if pf.File.Name.Name == "main" {
			continue
		}
		module_index_number := modules.File_To_Module[pf.Path]
		if module_index_number < 0 {
			continue
		}
		m := modules.Modules[module_index_number]
		relative := pf.Path
		if m.Root != "." {
			relative = strings.TrimPrefix(pf.Path, m.Root+"/")
		}
		canonical := module_index_canonicalize(path.Dir(relative))
		if canonical == "." {
			continue
		}
		ancestor_names := module_information_library_ancestors(m, canonical)
		if len(ancestor_names) <= 1 {
			continue
		}
		key := m.Root + "\x00" + canonical
		if seen[key] {
			continue
		}
		seen[key] = true
		diags = append(diags, Diagnostic{
			Position: token.Position{Filename: pf.Path, Line: 1, Column: 1},
			Name:     "library-tier-depth",
			Want:     "at most one non-main Go ancestor in module (v[0-9]+ skipped)",
			Message: fmt.Sprintf(
				"package %q at %q exceeds library tier; %d non-main ancestors: %v",
				pf.File.Name.Name, canonical, len(ancestor_names), ancestor_names,
			),
		})
	}
	return diags
}

// Returns ancestor directories of `directory` from nearest to module
// root, exclusive of "." itself. invariant.GameLoop annotates the loop
// as intentionally unbounded — path.Dir's fixed point on "." provides
// the real termination.
func check_library_tier_depth_ancestors(directory string) (ancestors []string) {

	current := directory
	for step := 0; ; step++ {
		parent := path.Dir(current)
		if parent == "." {
			break
		}
		if parent == current {
			break
		}
		ancestors = append(ancestors, parent)
		current = parent
	}
	return ancestors
}

// Returns the non-main Go ancestor packages of canonical that count toward
// tier depth. A binary module's top-level internal directory is excluded: all
// its code sits under internal and func Main lives there, so internal is the
// directory the count starts from — the same role a shared module's root
// plays — not a package nested above another. Without the exclusion
// internal/foo/default would count internal as a second ancestor and read as
// nested too deep. Shared modules have no internal directory, so the exclusion
// never affects them.
func module_information_library_ancestors(
	m module_information, canonical string,
) (ancestors []string) {
	for _, a := range check_library_tier_depth_ancestors(canonical) {
		if a == "internal" {
			continue
		}
		if _, has := m.Directory_Package[a]; !has {
			continue
		}
		ancestors = append(ancestors, a)
	}
	return ancestors
}

// A module must sit directly at the workspace root and be registered in
// go.work. Both facts keep module discovery a flat, predictable scan: a
// go.mod nested below a top-level directory would pull files into a module
// the workspace never declared, and an unregistered module would never be
// built or tested even though it carries importable code. Module Location is
// judged from the go.mod's own depth and so always applies; Module Definition
// is judged against go.work and so applies only when that file is present —
// its absence means the linter is scanning a detached subtree where
// registration cannot be decided. The Tracked filter, when set, drops modules
// whose go.mod is not part of the repository (an untracked toolchain or cache
// copy) so a whole-repo run stays focused on first-party code; top-level
// third_party/ is pruned before discovery, so vendored modules never reach
// here.
func check_module_definition_and_location(
	fsys fs.FS, modules *module_index, tracked map[string]bool,
) (diags []Diagnostic) {

	registered, has_workspace := module_workspace_use_set(fsys)
	for _, m := range modules.Modules {
		module_file_path := "go.mod"
		if m.Root != "." {
			module_file_path = m.Root + "/go.mod"
		}
		if tracked != nil {
			if !tracked[module_file_path] {
				continue
			}
		}
		position := token.Position{Filename: module_file_path, Line: 1, Column: 1}
		if strings.Contains(m.Root, "/") {
			diags = append(diags, Diagnostic{
				Position: position,
				Name:     "module-location",
				Want:     "every module located directly at the workspace root",
				Message: fmt.Sprintf(
					"module %q at %q must be located at the repository root",
					m.Module_Path, m.Root),
			})
		}
		if !has_workspace {
			continue
		}
		if registered[m.Root] {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: position,
			Name:     "module-definition",
			Want:     "every go.mod registered via a go.work use directive",
			Message: fmt.Sprintf(
				"module %q at %q is not registered in go.work",
				m.Module_Path, m.Root),
		})
	}
	return diags
}

// Reads the workspace file at the Fsys root and returns the set of module
// roots its use directives name, normalized to the same form as
// module_information.Root. The second result reports whether go.work was
// present at all, letting the caller distinguish an empty workspace from a
// detached scan with no workspace in view.
func module_workspace_use_set(
	fsys fs.FS,
) (registered map[string]bool, present bool) {

	content, err := fs.ReadFile(fsys, "go.work")
	if err != nil {
		return nil, false
	}
	registered = map[string]bool{}
	inside_block := false
	for _, raw_line := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(raw_line)
		if inside_block {
			if line == ")" {
				inside_block = false
				continue
			}
			if line == "" {
				continue
			}
			registered[module_workspace_normalize(line)] = true
			continue
		}
		if line == "use (" {
			inside_block = true
			continue
		}
		if strings.HasPrefix(line, "use ") {
			directive := strings.TrimSpace(strings.TrimPrefix(line, "use "))
			registered[module_workspace_normalize(directive)] = true
		}
	}
	return registered, true
}

// Folds a go.work use path into the directory form module discovery uses for
// a module root: quotes and a leading ./ stripped, a trailing slash removed,
// and the workspace root itself spelled ".".
func module_workspace_normalize(use_path string) (root string) {

	root = strings.Trim(use_path, `"`)
	root = strings.TrimPrefix(root, "./")
	root = strings.TrimSuffix(root, "/")
	if root == "" {
		return "."
	}
	return root
}

// Every non-main, non-_test package must carry a package doc comment in
// at least one of its files. The doc is the comment group attached to
// the `package` clause (ast.File.Doc) and gives `go doc <pkg>` and
// pkg.go.dev a one-paragraph summary of the package's purpose. Without
// it, importers and reviewers have to reconstruct intent from the first
// few declarations.
//
// Granularity is per (directory, package): Go allows the doc to live on
// any one file in the package, so we only flag a package when none of
// its files carries the doc. The diagnostic attaches to the first file
// in the group (sorted by path) so the location is deterministic.
// package main is exempt — it has no doc surface — as are `<X>_test`
// packages, which exist only to host the external test binary.
func check_package_documentation_comment(
	parsed_files []parsed_file,
) (diags []Diagnostic) {
	type key struct {
		Directory string
		Package   string
	}
	type state struct {
		Files             []parsed_file
		Has_Documentation bool
	}
	groups := map[key]*state{}
	for _, pf := range parsed_files {
		if pf.File.Name.Name == "main" {
			continue
		}
		if strings.HasSuffix(pf.File.Name.Name, "_test") {
			continue
		}
		k := key{Directory: path.Dir(pf.Path), Package: pf.File.Name.Name}
		st, has := groups[k]
		if !has {
			st = &state{}
			groups[k] = st
		}
		st.Files = append(st.Files, pf)
		if comment_group_documents(pf.File.Doc) {
			st.Has_Documentation = true
		}
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) (less bool) {
		if keys[i].Directory != keys[j].Directory {
			return keys[i].Directory < keys[j].Directory
		}
		return keys[i].Package < keys[j].Package
	})
	for _, k := range keys {
		st := groups[k]
		if st.Has_Documentation {
			continue
		}
		sort.Slice(st.Files, func(i, j int) (less bool) {
			return st.Files[i].Path < st.Files[j].Path
		})
		first := st.Files[0]
		diags = append(diags, Diagnostic{
			Position: first.File_Set.Position(first.File.Name.Pos()),
			Name:     "package-doc",
			Want:     "// Package " + k.Package + " ...",
			Message:  fmt.Sprintf("package %q is missing a doc comment", k.Package),
		})
	}
	return diags
}

// SPECIFICATION.md doctrine: every pure Go package carries a SPECIFICATION.md whose
// `##` headings each map, in order, to a leading Test_<Heading> function in
// specification_test.go. Enforced here rather than per-file because the
// contract spans three artifacts — the package directory, the markdown, and
// the test file — that no single-file checker sees together. Diagnostics
// attach to paths under the package directory so Main's scope filter limits
// the coverage requirement to whatever package argument the linter was given.
func check_specification(
	fsys fs.FS, parsed_files []parsed_file, index *module_index, scope string,
) (diags []Diagnostic) {
	directories := map[string]bool{}
	has_module := map[string]bool{}
	impure := map[string]bool{}
	for _, pf := range parsed_files {
		directory := path.Dir(pf.Path)
		directories[directory] = true
		if index.File_To_Module[pf.Path] >= 0 {
			has_module[directory] = true
		}
		if parsed_file_is_impure_package(pf, index) {
			impure[directory] = true
		}
	}
	sorted := make([]string, 0, len(directories))
	for directory := range directories {
		sorted = append(sorted, directory)
	}
	sort.Strings(sorted)
	for _, directory := range sorted {
		input := &check_specification_directory_input{
			Fsys: fsys, Directory: directory, Scope: scope,
			Has_Module: has_module[directory], Impure: impure[directory],
		}
		diags = append(diags, check_specification_directory(input)...)
	}
	return diags
}

type check_specification_directory_input struct {
	Fsys      fs.FS
	Directory string
	Scope     string
	// Has_Module is true when at least one file in the directory resolves to a
	// discovered module. The coverage mandate no-ops on module-less directories
	// — the same rule every other doctrine check follows for File_To_Module == -1
	// (see build_module_index) — so transient fixtures and subtrees scanned
	// without their go.mod in view are never required to carry a spec.
	Has_Module bool
	// Impure is true when the directory holds an impure package — `package main`
	// or a `default` package. The contract a SPECIFICATION.md documents is a
	// pure package's; the impure tier is the composition root, exempt from the
	// coverage mandate (an existing file is still format-validated).
	Impure bool
}

func check_specification_directory(
	input *check_specification_directory_input,
) (diags []Diagnostic) {
	specification_path := path.Join(input.Directory, "SPECIFICATION.md")
	// Presence is decided by the real directory listing, not fs.ReadFile: a
	// case-insensitive filesystem resolves SPECIFICATION.md to a differently-cased
	// file, so only an exact, byte-for-byte entry name counts as the spec.
	if !specification_directory_has_exact(input.Fsys, specification_path) {
		// Coverage follows the package argument: an explicit scope demands the
		// file within that subtree, and an empty scope — a whole-workspace run —
		// demands it everywhere. Vendored, example, and impure (package main or
		// `default`) trees are never required to carry one; they host
		// third-party, illustrative, or composition-root code, not the pure
		// package contract a SPECIFICATION.md documents.
		if !input.Has_Module {
			return nil
		}
		if specification_directory_exempt(input.Directory) {
			return nil
		}
		if input.Impure {
			return nil
		}
		covered := input.Scope == ""
		if !covered {
			covered = input.Directory == input.Scope
		}
		if !covered {
			covered = strings.HasPrefix(input.Directory, input.Scope+"/")
		}
		if !covered {
			return nil
		}
		return []Diagnostic{specification_coverage_diag(input.Directory)}
	}
	content, err := fs.ReadFile(input.Fsys, specification_path)
	if err != nil {
		return []Diagnostic{specification_coverage_diag(input.Directory)}
	}
	lines := strings.Split(string(content), "\n")
	leaves, format_diags := check_specification_format(specification_path, lines)
	diags = append(diags, format_diags...)
	return append(diags, check_specification_tests(input.Fsys, input.Directory, leaves)...)
}

// Validates the structural rules a SPECIFICATION.md must obey — no preamble
// before the first heading, a single `##` heading level, unique headings of
// letters-and-digits words, a blank line either side of every heading, a
// contiguous body of one to three lines per section. Line width is not checked
// here: check_stream_markdown_line_max enforces it for every .md file. Returns
// the headings in source order so the test-correspondence rules can use them.
func check_specification_format(
	specification_path string, lines []string,
) (leaves []string, diags []Diagnostic) {
	headings, scan_diags := specification_scan_headings(specification_path, lines)
	diags = append(diags, scan_diags...)
	leaf_lines, names, leaf_diags := specification_leaves(specification_path, headings)
	diags = append(diags, leaf_diags...)
	body_diags := specification_scan_bodies(specification_path, lines, headings, leaf_lines)
	return names, append(diags, body_diags...)
}

// One heading found in a SPECIFICATION.md: its level (2 or 3), 1-based source
// line, the raw text after the marker, and its Ada_Case form.
type specification_heading struct {
	Level int
	Line  int
	Raw   string
	Ada   string
}

func specification_position(path string, line int) (position token.Position) {
	return token.Position{Filename: path, Line: line}
}

// Reports a line's heading level: 3 for "### ", 1 for "# ", 0 otherwise.
func specification_heading_parse(line string) (level int, raw string) {
	if strings.HasPrefix(line, "### ") {
		return 3, strings.TrimPrefix(line, "### ")
	}
	if strings.HasPrefix(line, "# ") {
		return 1, strings.TrimPrefix(line, "# ")
	}
	return 0, ""
}

// Pass one: collect every ## / ### heading and emit the diagnostics that need
// only line context — bad heading levels, content before the first heading,
// blank-line fencing, and non-letter/digit heading words.
func specification_scan_headings(
	specification_path string, lines []string,
) (headings []specification_heading, diags []Diagnostic) {
	seen_heading := false
	for i, line := range lines {
		position := specification_position(specification_path, i+1)
		level, raw := specification_heading_parse(line)
		if level == 0 {
			if strings.HasPrefix(line, "#") {
				diags = append(diags, specification_heading_level_diag(position))
				continue
			}
			if strings.TrimSpace(line) == "" {
				continue
			}
			if !seen_heading {
				diags = append(diags, specification_preamble_diag(position))
			}
			continue
		}
		headings = append(headings, specification_heading{
			Level: level, Line: i + 1, Raw: raw, Ada: specification_ada_case(raw)})
		diags = append(diags, specification_heading_line_diags(position, lines, i, raw)...)
		seen_heading = true
	}
	return headings, diags
}

// The per-heading diagnostics for one heading line: non-letter/digit words and
// blank-line fencing.
func specification_heading_line_diags(
	position token.Position, lines []string, i int, raw string,
) (diags []Diagnostic) {
	if specification_heading_words_invalid(raw) {
		diags = append(diags, specification_heading_words_diag(position, raw))
	}
	return append(diags, check_specification_blank_lines(position, lines, i, raw)...)
}

// State for the tree walk that determines leaves: a ## with no ### child is a
// leaf named Ada(##); each ### is a leaf named Ada(##)_Ada(###). ## names are
// unique file-wide; ### names are unique within their parent ##.
type specification_tree struct {
	Path       string
	Seen_H2    map[string]bool
	Seen_H3    map[string]bool
	Parent     specification_heading
	Has_Child  bool
	Leaf_Lines map[int]bool
	Names      []string
}

// Pass two: walk the headings into the tree, returning the lines that open a
// leaf section, the ordered leaf test-name bases, and the uniqueness diagnostics.
func specification_leaves(
	specification_path string, headings []specification_heading,
) (leaf_lines map[int]bool, names []string, diags []Diagnostic) {
	tree := &specification_tree{
		Path: specification_path, Seen_H2: map[string]bool{},
		Seen_H3: map[string]bool{}, Leaf_Lines: map[int]bool{},
	}
	for _, heading := range headings {
		diags = append(diags, specification_tree_add(tree, heading)...)
	}
	specification_tree_close(tree)
	return tree.Leaf_Lines, tree.Names, diags
}

func specification_tree_add(
	tree *specification_tree, heading specification_heading,
) (diags []Diagnostic) {
	if heading.Level == 3 {
		return specification_tree_child(tree, heading)
	}
	specification_tree_close(tree)
	if tree.Seen_H2[heading.Raw] {
		diags = append(diags, specification_tree_duplicate(tree, heading))
	}
	tree.Seen_H2[heading.Raw] = true
	tree.Parent = heading
	tree.Has_Child = false
	tree.Seen_H3 = map[string]bool{}
	return diags
}

func specification_tree_child(
	tree *specification_tree, heading specification_heading,
) (diags []Diagnostic) {
	position := specification_position(tree.Path, heading.Line)
	if tree.Parent.Line == 0 {
		return []Diagnostic{specification_orphan_diag(position, heading.Raw)}
	}
	tree.Has_Child = true
	if tree.Seen_H3[heading.Raw] {
		diags = append(diags, specification_tree_duplicate(tree, heading))
	}
	tree.Seen_H3[heading.Raw] = true
	tree.Leaf_Lines[heading.Line] = true
	tree.Names = append(tree.Names, tree.Parent.Ada+"_"+heading.Ada)
	return diags
}

// Records the just-finished ## as a leaf when it gained no ### child.
func specification_tree_close(tree *specification_tree) {
	if tree.Parent.Line == 0 {
		return
	}
	if tree.Has_Child {
		return
	}
	tree.Leaf_Lines[tree.Parent.Line] = true
	tree.Names = append(tree.Names, tree.Parent.Ada)
}

func specification_tree_duplicate(
	tree *specification_tree, heading specification_heading,
) (diag Diagnostic) {
	position := specification_position(tree.Path, heading.Line)
	return specification_heading_duplicate_diag(position, heading.Raw)
}

// State for the body pass: the currently open section, its accumulated body line
// count, and whether a blank line has already interrupted that body.
type specification_body struct {
	Path       string
	Leaf_Lines map[int]bool
	Open       specification_heading
	Body       int
	Blank      bool
}

// Pass three: attribute body lines to their opening heading, flagging oversized
// sections, gaps in a section body, and leaf sections with no body. A branch ##
// intro is size- and gap-checked but, not being a leaf, may be empty.
func specification_scan_bodies(
	specification_path string, lines []string,
	headings []specification_heading, leaf_lines map[int]bool,
) (diags []Diagnostic) {
	at := map[int]specification_heading{}
	for _, heading := range headings {
		at[heading.Line] = heading
	}
	state := &specification_body{Path: specification_path, Leaf_Lines: leaf_lines}
	for i, line := range lines {
		heading, is_heading := at[i+1]
		if is_heading {
			diags = append(diags, specification_body_close(state)...)
			state.Open = heading
			state.Body = 0
			state.Blank = false
			continue
		}
		if strings.HasPrefix(line, "#") {
			diags = append(diags, specification_body_close(state)...)
			state.Open = specification_heading{}
			state.Body = 0
			state.Blank = false
			continue
		}
		if strings.TrimSpace(line) == "" {
			if state.Body > 0 {
				state.Blank = true
			}
			continue
		}
		if state.Open.Line == 0 {
			continue
		}
		diags = append(diags, specification_body_line(state, i+1)...)
	}
	return append(diags, specification_body_close(state)...)
}

func specification_body_line(
	state *specification_body, line int,
) (diags []Diagnostic) {
	position := specification_position(state.Path, line)
	raw := state.Open.Raw
	if state.Blank {
		diags = append(diags, specification_section_contiguity_diag(position, raw))
		state.Blank = false
	}
	state.Body++
	if state.Body == 4 {
		diags = append(diags, specification_section_diag(position, raw))
	}
	return diags
}

// Emits the body-required diagnostic when a leaf section closed with no body.
func specification_body_close(state *specification_body) (diags []Diagnostic) {
	if state.Open.Line == 0 {
		return nil
	}
	if !state.Leaf_Lines[state.Open.Line] {
		return nil
	}
	if state.Body != 0 {
		return nil
	}
	position := specification_position(state.Path, state.Open.Line)
	return []Diagnostic{specification_section_body_diag(position, state.Open.Raw)}
}

// True when a heading carries a word with a rune that is neither a letter nor a
// digit. Such a rune survives into the normalized Test_<Heading> name and makes
// it an illegal Go identifier, so the test-correspondence rule could never be
// satisfied for that heading.
func specification_heading_words_invalid(heading string) (invalid bool) {
	for _, word := range strings.Fields(heading) {
		for _, letter := range word {
			if unicode.IsLetter(letter) {
				continue
			}
			if unicode.IsDigit(letter) {
				continue
			}
			return true
		}
	}
	return false
}

func specification_preamble_diag(position token.Position) (diag Diagnostic) {
	return Diagnostic{
		Position: position, Name: "specification",
		Want: "open with a heading",
		Message: fmt.Sprintf("%s:%d content precedes the first heading",
			position.Filename, position.Line),
	}
}

func specification_section_body_diag(
	position token.Position, heading string,
) (diag Diagnostic) {
	return Diagnostic{
		Position: position, Name: "specification",
		Want: "give the section a body line",
		Message: fmt.Sprintf("%s:%d section %q has no body line",
			position.Filename, position.Line, heading),
	}
}

func specification_section_contiguity_diag(
	position token.Position, heading string,
) (diag Diagnostic) {
	return Diagnostic{
		Position: position, Name: "specification",
		Want: "keep the section body contiguous",
		Message: fmt.Sprintf("%s:%d section %q has a blank line between body lines",
			position.Filename, position.Line, heading),
	}
}

func specification_heading_duplicate_diag(
	position token.Position, heading string,
) (diag Diagnostic) {
	return Diagnostic{
		Position: position, Name: "specification",
		Want: "make every heading unique",
		Message: fmt.Sprintf("%s:%d heading %q is duplicated",
			position.Filename, position.Line, heading),
	}
}

func specification_heading_words_diag(
	position token.Position, heading string,
) (diag Diagnostic) {
	return Diagnostic{
		Position: position, Name: "specification",
		Want: "use only letters and digits in headings",
		Message: fmt.Sprintf("%s:%d heading %q must use only letters and digits",
			position.Filename, position.Line, heading),
	}
}

func specification_heading_level_diag(position token.Position) (diag Diagnostic) {
	return Diagnostic{
		Position: position, Name: "specification", Want: "use a # or ### heading",
		Message: fmt.Sprintf("%s:%d uses a heading that is not level # or ###",
			position.Filename, position.Line),
	}
}

func specification_orphan_diag(
	position token.Position, heading string,
) (diag Diagnostic) {
	return Diagnostic{
		Position: position, Name: "specification",
		Want: "nest the subheading under a #",
		Message: fmt.Sprintf("%s:%d ### %q has no parent #",
			position.Filename, position.Line, heading),
	}
}

func specification_section_diag(
	position token.Position, heading string,
) (diag Diagnostic) {
	return Diagnostic{
		Position: position, Name: "specification",
		Want: "limit sections to three lines",
		Message: fmt.Sprintf("%s:%d section %q exceeds three lines",
			position.Filename, position.Line, heading),
	}
}

// Reports whether the directory holds an entry whose name is exactly `name`,
// byte for byte. fs.ReadFile is insufficient on a case-insensitive filesystem
// (it resolves a differently-cased file), so the doctrine's exact-name rule must
// consult the real directory listing.
func specification_directory_has_exact(
	fsys fs.FS, file_path string,
) (present bool) {
	entries, err := fs.ReadDir(fsys, path.Dir(file_path))
	if err != nil {
		return false
	}
	name := path.Base(file_path)
	for _, entry := range entries {
		if entry.Name() == name {
			return true
		}
	}
	return false
}

// A directory is exempt from the coverage mandate when any path segment is
// `third_party` (vendored code in a separate module) or `examples`
// (illustrative, not a real package contract). An existing SPECIFICATION.md in
// such a tree is still format-validated; it just is never required to exist.
func specification_directory_exempt(directory string) (exempt bool) {
	for _, segment := range strings.Split(directory, "/") {
		if segment == "third_party" {
			return true
		}
		if segment == "examples" {
			return true
		}
	}
	return false
}

func specification_coverage_diag(directory string) (diag Diagnostic) {
	return Diagnostic{
		Position: token.Position{Filename: path.Join(directory, "SPECIFICATION.md")},
		Name:     "specification",
		Want:     "add SPECIFICATION.md",
		Message:  fmt.Sprintf("package %q is missing SPECIFICATION.md", directory),
	}
}

func specification_test_file_diag(directory string) (diag Diagnostic) {
	return Diagnostic{
		Position: token.Position{Filename: path.Join(directory, "specification_test.go")},
		Name:     "specification",
		Want:     "add specification_test.go",
		Message:  fmt.Sprintf("package %q is missing specification_test.go", directory),
	}
}

func check_specification_blank_lines(
	position token.Position, lines []string, i int, heading string,
) (diags []Diagnostic) {
	preceded := i > 0
	if preceded {
		preceded = lines[i-1] == ""
	}
	if !preceded {
		diags = append(diags, Diagnostic{
			Position: position, Name: "specification",
			Want: "precede heading with a blank line",
			Message: fmt.Sprintf("%s:%d heading %q is not preceded by a blank line",
				position.Filename, i+1, heading),
		})
	}
	followed := i+1 < len(lines)
	if followed {
		followed = lines[i+1] == ""
	}
	if !followed {
		diags = append(diags, Diagnostic{
			Position: position, Name: "specification",
			Want: "follow heading with a blank line",
			Message: fmt.Sprintf("%s:%d heading %q is not followed by a blank line",
				position.Filename, i+1, heading),
		})
	}
	return diags
}

// Verifies specification_test.go exists and that its leading function
// declarations are exactly Test_<Heading> for each heading, in heading order.
// Comparing by index enforces both the per-heading correspondence and the
// "tests at the very top, in order" rule in one pass: a helper or a misordered
// test shifts the sequence and surfaces as a mismatch at that position.
func check_specification_tests(
	fsys fs.FS, directory string, leaves []string,
) (diags []Diagnostic) {
	test_path := path.Join(directory, "specification_test.go")
	// Exact name first, for the same reason as SPECIFICATION.md: a
	// case-insensitive filesystem would otherwise let a differently-cased file
	// stand in for specification_test.go.
	if !specification_directory_has_exact(fsys, test_path) {
		return []Diagnostic{specification_test_file_diag(directory)}
	}
	functions, ok := check_specification_test_function_names(fsys, test_path)
	if !ok {
		return []Diagnostic{specification_test_file_diag(directory)}
	}
	for i, leaf := range leaves {
		want := "Test_" + leaf
		matched := i < len(functions)
		if matched {
			matched = functions[i] == want
		}
		if matched {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: token.Position{Filename: test_path},
			Name:     "specification",
			Want:     want,
			Message: fmt.Sprintf(
				"%s:%d needs %s for leaf %q (in order, at top)",
				test_path, i+1, want, leaf),
		})
	}
	return diags
}

func check_specification_test_function_names(
	fsys fs.FS, test_path string,
) (functions []string, ok bool) {
	content, err := fs.ReadFile(fsys, test_path)
	if err != nil {
		return nil, false
	}
	file_set := token.NewFileSet()
	file, parse_err := parser.ParseFile(
		file_set, test_path, content, parser.SkipObjectResolution)
	if parse_err != nil {
		return nil, true
	}
	for _, declaration := range file.Decls {
		if generic, is_generic := declaration.(*ast.GenDecl); is_generic {
			if generic.Tok == token.IMPORT {
				continue
			}
			// A var/const/type before the heading tests breaks the "tests at the
			// very top" rule. Surface it as a slot that can never match a
			// Test_<Heading> name, so the ordering check flags it at its position.
			functions = append(functions, generic.Tok.String())
			continue
		}
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		functions = append(functions, function.Name.Name)
	}
	return functions, true
}

// Normalizes a heading to the Ada_Case form used for its test name: each
// space-separated word's first rune is upper-cased and the words are joined
// with underscores ("Test File Name" -> "Test_File_Name").
func specification_ada_case(heading string) (name string) {
	words := strings.Fields(heading)
	for i, word := range words {
		runes := []rune(word)
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, "_")
}

// Runs checks per file in parallel — CPU bound, capped at the injected
// CPU_Count (typically runtime.NumCPU from main.go).
func check_file_system_run_checks(
	parsed_files []parsed_file, cpu_count int, instrumentation []string,
	word_replacements map[string][]string,
) (diags []Diagnostic) {

	per_file_diags := make([][]Diagnostic, len(parsed_files))
	sem := make(chan struct{}, cpu_count)
	var wg sync.WaitGroup
	for i, pf := range parsed_files {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, pf parsed_file) {
			defer wg.Done()
			defer func() { <-sem }()
			per_file_diags[i] = Check_File(
				pf.File_Set, pf.File, pf.Source, instrumentation, word_replacements)
		}(i, pf)
	}
	wg.Wait()
	for _, d := range per_file_diags {
		diags = append(diags, d...)
	}
	return diags
}

// TigerStyle: comments are sentences with a capital letter, ending in `.` or `:`.
// Inline (end-of-line) comments are exempt — they can be phrases.
// Compiler-directive pragmas (e.g. `//go:embed`) are exempt.
var pragma_re = regexp.MustCompile(`^//[a-z][a-z0-9_-]*:`)

func check_comments(file_set *token.FileSet, file *ast.File, source []byte) (diags []Diagnostic) {

	for _, group := range file.Comments {
		if len(group.List) == 0 {
			continue
		}
		first := group.List[0]
		if !strings.HasPrefix(first.Text, "//") {
			continue
		}
		var filtered []*ast.Comment
		for _, c := range group.List {
			if pragma_re.MatchString(c.Text) {
				continue
			}
			filtered = append(filtered, c)
		}
		if len(filtered) == 0 {
			continue
		}
		for _, c := range filtered {
			if !check_comments_group_has_space_after_slashes(c.Text) {
				diags = append(diags, Diagnostic{
					Position: file_set.Position(c.Slash),
					Message:  "comment: missing space after `//`",
				})
			}
		}
		if check_comments_group_is_inline(file_set, source, filtered[0]) {
			continue
		}
		diags = append(diags, check_comments_group_capital(file_set, filtered[0])...)
		diags = append(diags,
			check_comments_group_terminator(file_set, filtered[len(filtered)-1])...)
	}
	return diags
}

func check_comments_group_capital(file_set *token.FileSet, c *ast.Comment) (diags []Diagnostic) {

	body := comment_body(c.Text)
	if body == "" {
		return nil
	}
	r, _ := utf8.DecodeRuneInString(body)
	if !unicode.IsLetter(r) {
		return nil
	}
	if unicode.IsUpper(r) {
		return nil
	}
	return []Diagnostic{{
		Position: file_set.Position(c.Slash),
		Message:  "comment: should start with capital letter",
	}}
}

func check_comments_group_terminator(file_set *token.FileSet, c *ast.Comment) (diags []Diagnostic) {

	body := strings.TrimRight(comment_body(c.Text), " \t")
	if body == "" {
		return nil
	}
	r, _ := utf8.DecodeLastRuneInString(body)
	switch r {
	case '.', ':', '?', '!':
		return nil
	}
	return []Diagnostic{{
		Position: file_set.Position(c.Slash),
		Message:  "comment: should end with `.`, `:`, `?`, or `!`",
	}}
}

func comment_body(text string) (body string) {

	if !strings.HasPrefix(text, "//") {
		return ""
	}
	return strings.TrimLeft(text[2:], " \t")
}

func check_comments_group_has_space_after_slashes(text string) (ok bool) {

	if !strings.HasPrefix(text, "//") {
		return false
	}
	if len(text) == 2 {
		return true
	}
	switch text[2] {
	case ' ', '\t':
		return true
	}
	return false
}

// TigerStyle: recursion makes stack depth depend on input, which is
// adversarially unbounded. Detects direct AND mutual recursion via a per-file
// call graph: nodes are top-level FuncDecls, edges are name-based calls. Any
// cycle in the graph (self-loop or longer) is reported as one diagnostic.
//
// Limitations (will not detect):
//   - Method calls (`x.foo()`) — SelectorExpr, not Ident.
//   - Function values (`g := f; g()`) — aliasing loses the name.
//   - Interface dispatch — undecidable statically.
//   - Package-qualified self-calls (`pkg.F()` from within pkg) — SelectorExpr.
//   - Cross-file / cross-package — needs go/types.
//
// At each cycle's diagnostic, invariant.Ensure asserts that no edge in the
// cycle is shadowed by a local of the same name as the callee. That property
// is enforced by check_shadow (top-level names are in its globalNames set),
// so a shadowed cycle edge means check_shadow missed something — a real
// bug, not a property of the input.
func check_no_recursion(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	graph := build_file_call_graph(file_set, file)
	adj := map[string][]call_edge{}
	for _, e := range graph.Edges {
		adj[e.Caller] = append(adj[e.Caller], e)
	}
	return check_no_recursion_find_cycles(graph.Caller_Order, adj)
}

type file_call_graph struct {
	Caller_Order []string
	Decls        map[string]*ast.FuncDecl
	Edges        []call_edge
}

func build_file_call_graph(
	file_set *token.FileSet,
	file *ast.File,
) (graph file_call_graph) {

	function_names := map[string]bool{}
	graph.Decls = map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		function_declaration, is_function_declaration := declaration.(*ast.FuncDecl)
		if !is_function_declaration {
			continue
		}
		if function_names[function_declaration.Name.Name] {
			continue
		}
		graph.Caller_Order = append(graph.Caller_Order, function_declaration.Name.Name)
		function_names[function_declaration.Name.Name] = true
		graph.Decls[function_declaration.Name.Name] = function_declaration
	}
	for _, declaration := range file.Decls {
		function_declaration, is_function_declaration := declaration.(*ast.FuncDecl)
		if !is_function_declaration {
			continue
		}
		if function_declaration.Body == nil {
			continue
		}
		v := &recursion_visitor{
			File_Set: file_set,
			Caller:   function_declaration.Name.Name,
			Targets:  function_names,
			Edges:    &graph.Edges,
		}
		ast.Walk(v, function_declaration.Body)
	}
	return graph
}

type call_edge struct {
	Caller   string
	Callee   string
	Position token.Position
	Shadowed bool
}

type recursion_visitor struct {
	File_Set *token.FileSet
	Caller   string
	Targets  map[string]bool
	Edges    *[]call_edge
	// Scopes[i] holds names defined in scope level i. Pushed on entering
	// scope-introducing nodes (BlockStmt, IfStmt, ForStmt, RangeStmt, FuncLit)
	// and popped on exit.
	Scopes []map[string]bool
	// Push_history records how many scopes each Visit(non-nil) pushed, so the
	// matching Visit(nil) can pop the right number.
	Push_History []int
}

// Visit is the ast.Visitor entry point: pushes a fresh scope frame
// when n introduces one (Block, If, For, Range, FuncLit) and records
// any same-package call edge encountered.
func (v *recursion_visitor) Visit(n ast.Node) (next ast.Visitor) {

	if n == nil {
		k := v.Push_History[len(v.Push_History)-1]
		v.Push_History = v.Push_History[:len(v.Push_History)-1]
		v.Scopes = v.Scopes[:len(v.Scopes)-k]
		return nil
	}
	pushed := recursion_visitor_enter(v, n)
	v.Push_History = append(v.Push_History, pushed)
	return v
}

func recursion_visitor_enter(v *recursion_visitor, n ast.Node) (pushed int) {

	switch x := n.(type) {
	case *ast.BlockStmt:
		v.Scopes = append(v.Scopes, map[string]bool{})
		return 1
	case *ast.FuncLit:
		v.Scopes = append(v.Scopes, map[string]bool{})
		return 1
	case *ast.IfStmt:
		v.Scopes = append(v.Scopes, map[string]bool{})
		recursion_visitor_enter_define_statement(v, x.Init)
		return 1
	case *ast.ForStmt:
		v.Scopes = append(v.Scopes, map[string]bool{})
		recursion_visitor_enter_define_statement(v, x.Init)
		return 1
	case *ast.RangeStmt:
		v.Scopes = append(v.Scopes, map[string]bool{})
		recursion_visitor_define_ident(v, x.Key)
		recursion_visitor_define_ident(v, x.Value)
		return 1
	case *ast.AssignStmt:
		if x.Tok == token.DEFINE {
			for _, lhs := range x.Lhs {
				recursion_visitor_define_ident(v, lhs)
			}
		}
		return 0
	case *ast.CallExpr:
		recursion_visitor_enter_record_call_edge(v, x)
		return 0
	}
	return 0
}

func recursion_visitor_enter_define_statement(v *recursion_visitor, s ast.Stmt) {

	as, is_assign := s.(*ast.AssignStmt)
	if !is_assign {
		return
	}
	if as.Tok != token.DEFINE {
		return
	}
	for _, lhs := range as.Lhs {
		recursion_visitor_define_ident(v, lhs)
	}
}

func recursion_visitor_define_ident(v *recursion_visitor, e ast.Expr) {

	identifier, is_ident := e.(*ast.Ident)
	if !is_ident {
		return
	}
	if identifier.Name == "_" {
		return
	}
	v.Scopes[len(v.Scopes)-1][identifier.Name] = true
}

// Recursion_visitor_call_fun_is_ident reports whether the call expression's
// Fun position is a bare identifier (`f()`) as opposed to a selector
// expression (`pkg.f()` or `x.m()`). Nil-safe so it can be evaluated as a
// Cross_Product Sometimes-predicate before the parent function's pointer
// assertions fire.
func recursion_visitor_call_function_is_ident(call *ast.CallExpr) (yes bool) {
	_, is_ident := call.Fun.(*ast.Ident)
	return is_ident
}

func recursion_visitor_enter_record_call_edge(v *recursion_visitor, call *ast.CallExpr) {

	identifier, is_ident := call.Fun.(*ast.Ident)
	if !is_ident {
		return
	}
	if !v.Targets[identifier.Name] {
		return
	}
	shadowed := false
	for _, s := range v.Scopes {
		if s[identifier.Name] {
			shadowed = true
			break
		}
	}
	*v.Edges = append(*v.Edges, call_edge{
		Caller:   v.Caller,
		Callee:   identifier.Name,
		Position: v.File_Set.Position(call.Pos()),
		Shadowed: shadowed,
	})
}

// Walk starts on FuncDecl.Body BlockStmt (1 push); CallExpr is reached
// through ≥1 further non-pushing Visit, so Push_History len ≥ 2.

// Walk starts on the FuncDecl.Body BlockStmt, which Visit pushes a 1
// onto Push_History for; CallExpr is reached only through at least
// one further non-pushing Visit (ExprStmt or argument descent), so
// Push_History length at this site is ≥ 2.

// Iterative 3-color DFS for cycle detection. Each back edge from a GRAY node
// to a still-GRAY ancestor closes a cycle; we emit one diagnostic per back
// edge (so a strongly-connected component with multiple back edges yields
// multiple diagnostics, one per cycle).
func check_no_recursion_find_cycles(
	callers []string,
	adj map[string][]call_edge,
) (diags []Diagnostic) {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := map[string]int{}
	for _, start := range callers {
		if color[start] != white {
			continue
		}
		diags = append(diags,
			check_no_recursion_find_cycles_dfs(start, color, adj)...)
	}
	return diags
}

func check_no_recursion_find_cycles_dfs(
	start string,
	color map[string]int,
	adj map[string][]call_edge,
) (diags []Diagnostic) {

	const (
		white = 0
		gray  = 1
		black = 2
	)
	type dfs_frame struct {
		Node string
		Iter int
	}
	path := []string{start}
	on_path := map[string]int{start: 0}
	color[start] = gray
	stack := []dfs_frame{{Node: start}}
	for len(stack) > 0 {
		top := &stack[len(stack)-1]
		edges := adj[top.Node]
		if top.Iter >= len(edges) {
			color[top.Node] = black
			delete(on_path, top.Node)
			path = path[:len(path)-1]
			stack = stack[:len(stack)-1]
			continue
		}
		e := edges[top.Iter]
		top.Iter++
		switch color[e.Callee] {
		case white:
			color[e.Callee] = gray
			on_path[e.Callee] = len(path)
			path = append(path, e.Callee)
			stack = append(stack, dfs_frame{Node: e.Callee})
		case gray:
			cycle_start := on_path[e.Callee]
			cycle_nodes := append([]string{}, path[cycle_start:]...)
			diags = append(diags,
				check_no_recursion_find_cycles_dfs_diag(cycle_nodes, e))
		}
	}
	return diags
}

func check_no_recursion_find_cycles_dfs_diag(
	cycle_nodes []string,
	back_edge call_edge,
) (diag Diagnostic) {
	return Diagnostic{
		Position: back_edge.Position,
		Message:  check_no_recursion_find_cycles_dfs_diag_message(cycle_nodes),
	}
}

func check_no_recursion_find_cycles_dfs_diag_message(cycle_nodes []string) (message string) {

	if len(cycle_nodes) == 1 {
		return fmt.Sprintf("recursion: %s calls itself", cycle_nodes[0])
	}
	var sb strings.Builder
	sb.WriteString("recursion: cycle ")
	for _, n := range cycle_nodes {
		sb.WriteString(n)
		sb.WriteString(" → ")
	}
	sb.WriteString(cycle_nodes[0])
	return sb.String()
}

// The entry point is what readers look for first — burying it under helpers
// forces a scan of the whole file to find where execution starts.
func check_main_first(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	var first_function *ast.FuncDecl
	for _, declaration := range file.Decls {
		function_declaration, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if first_function == nil {
			first_function = function_declaration
			continue
		}
		if function_declaration.Name.Name == "main" {
			diags = append(diags, Diagnostic{
				Position: file_set.Position(function_declaration.Pos()),
				Message:  "func main should be declared first in the file",
			})
		}
		if function_declaration.Name.Name == "Main" {
			diags = append(diags, Diagnostic{
				Position: file_set.Position(function_declaration.Pos()),
				Message:  "func Main should be declared first in the file",
			})
		}
		if function_declaration.Name.Name == "TestMain" {
			diags = append(diags, Diagnostic{
				Position: file_set.Position(function_declaration.Pos()),
				Message:  "func TestMain should be declared first in the file",
			})
		}
	}
	return diags
}

// A bare `_ =` or `_, _, _ :=` hides every return value of the RHS — most
// often a silently-dropped error. Mixed forms like `_, x := f()` are allowed
// because at least one return is named and the `_` is genuine selection.
// `var _ Iface = (*Impl)(nil)` is allowed: the explicit type makes it a
// compile-time interface-satisfaction assertion, not a value discard.
func check_no_discard(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	const message = "discard: _ = ... hides the value; name it or drop the assignment"
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if check_no_discard_all_blank_exprs(x.Lhs) {
				diags = append(diags, Diagnostic{
					Position: file_set.Position(x.Pos()),
					Message:  message,
				})
			}
		case *ast.GenDecl:
			if x.Tok != token.VAR {
				return true
			}
			for _, specification := range x.Specs {
				vs, ok := specification.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if vs.Type != nil {
					continue
				}
				if len(vs.Values) == 0 {
					continue
				}
				if !check_no_discard_all_blank_idents(vs.Names) {
					continue
				}
				diags = append(diags, Diagnostic{
					Position: file_set.Position(vs.Pos()),
					Message:  message,
				})
			}
		}
		return true
	})
	return diags
}

func check_no_discard_all_blank_exprs(exprs []ast.Expr) (all bool) {

	if len(exprs) == 0 {
		return false
	}
	for _, expression := range exprs {
		identifier, ok := expression.(*ast.Ident)
		if !ok {
			return false
		}
		if identifier.Name != "_" {
			return false
		}
	}
	return true
}

func check_no_discard_all_blank_idents(names []*ast.Ident) (all bool) {

	if len(names) == 0 {
		return false
	}
	for _, name := range names {
		if name.Name != "_" {
			return false
		}
	}
	return true
}

// Unexported struct fields hide state from cross-package callers and force
// awkward getter/setter accessors. Embedded fields' implicit name is the
// rightmost ident of the type expression (stripping `*` and package qualifier).
func check_public_struct_fields(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	ast.Inspect(file, func(n ast.Node) (descend bool) {
		st, ok := n.(*ast.StructType)
		if !ok {
			return true
		}
		if st.Fields == nil {
			return true
		}
		for _, f := range st.Fields.List {
			if len(f.Names) == 0 {
				check_public_struct_fields_embedded(file_set, f.Type, &diags)
				continue
			}
			for _, name := range f.Names {
				check_public_struct_fields_named(file_set, name, &diags)
			}
		}
		return true
	})
	return diags
}

func check_public_struct_fields_named(
	file_set *token.FileSet, identifier *ast.Ident, diags *[]Diagnostic,
) {

	if identifier.Name == "" {
		return
	}
	r := rune(identifier.Name[0])
	if !unicode.IsLower(r) {
		return
	}
	suggested := check_public_struct_fields_named_capitalize(identifier.Name)
	*diags = append(*diags, Diagnostic{
		Position: file_set.Position(identifier.Pos()),
		Message:  fmt.Sprintf("rename %s -> %s", identifier.Name, suggested),
	})
}

// Expression_is_pointer_wrapped reports whether the type expression is a
// `*T` star-wrapped form, nil-safe so it can be used as a Cross_Product
// Sometimes-predicate evaluated before the parent function's pointer
// assertions fire.
func expression_is_pointer_wrapped(expression ast.Expr) (yes bool) {
	if expression == nil {
		return false
	}
	_, is_star := expression.(*ast.StarExpr)
	return is_star
}

func check_public_struct_fields_embedded(
	file_set *token.FileSet, expression ast.Expr, diags *[]Diagnostic,
) {

	base := expression
	for step := 0; ; step++ {
		star, is_star := base.(*ast.StarExpr)
		if !is_star {
			break
		}
		base = star.X
	}
	switch x := base.(type) {
	case *ast.Ident:
		check_public_struct_fields_named(file_set, x, diags)
	case *ast.SelectorExpr:
		check_public_struct_fields_named(file_set, x.Sel, diags)
	}
}

func check_public_struct_fields_named_capitalize(name string) (output_string string) {

	rs := []rune(name)
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}

// Every field of an exported package-level struct must carry its own doc
// comment. The struct-level comment documents the type as a whole; a reader
// scanning a single field otherwise sees its name and type but no statement
// of what it holds or why it exists. The check covers every file-scope
// exported struct, package main included — an exported field is a reader's
// surface whether or not the package is importable. Only _test.go fixtures
// are exempt, since they deliberately model violations. Embedded fields are
// skipped because their meaning is their type, and blank-named fields because
// they exist only for padding or a compile-time assertion. A trailing line
// comment binds to ast.Field.Comment rather than Doc, so it does not satisfy
// the rule: the doc must lead the field.
func check_struct_field_documentation_comment(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	tok_file := file_set.File(file.Pos())
	if tok_file == nil {
		return nil
	}
	if strings.HasSuffix(tok_file.Name(), "_test.go") {
		return nil
	}
	for _, declaration := range file.Decls {
		generic_declaration, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		if generic_declaration.Tok != token.TYPE {
			continue
		}
		for _, specification := range generic_declaration.Specs {
			type_specification, is_type := specification.(*ast.TypeSpec)
			if !is_type {
				continue
			}
			if !ast.IsExported(type_specification.Name.Name) {
				continue
			}
			struct_type, is_struct := type_specification.Type.(*ast.StructType)
			if !is_struct {
				continue
			}
			check_struct_field_documentation_comment_fields(
				&check_struct_field_documentation_comment_fields_input{
					File_Set:    file_set,
					Struct_Name: type_specification.Name.Name,
					Struct_Type: struct_type,
					Diags:       &diags,
				})
		}
	}
	return diags
}

type check_struct_field_documentation_comment_fields_input struct {
	File_Set    *token.FileSet
	Struct_Name string
	Struct_Type *ast.StructType
	Diags       *[]Diagnostic
}

func check_struct_field_documentation_comment_fields(
	input *check_struct_field_documentation_comment_fields_input,
) {

	if input.Struct_Type.Fields == nil {
		return
	}
	for _, field := range input.Struct_Type.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		if field.Doc != nil {
			continue
		}
		for _, name := range field.Names {
			if name.Name == "_" {
				continue
			}
			*input.Diags = append(*input.Diags, Diagnostic{
				Position: input.File_Set.Position(name.Pos()),
				Name:     "struct-field-doc",
				Want:     "doc comment leading every exported-struct field",
				Message: fmt.Sprintf(
					"field %s.%s lacks a doc comment",
					input.Struct_Name, name.Name),
			})
		}
	}
}

// An exported struct field or alias whose type resolves to an unexported
// named identifier produces an opaque slot in the exported surface:
// cross-package consumers receive the value but cannot construct, name,
// or pattern against the private parts, forcing a getter/setter shim or a
// type-assertion dance. The walk visits every exported struct type at file
// scope, unwraps leading `*` from each field's type expression, and flags
// any field whose base identifier is unexported. Same-file struct targets
// are walked transitively (visited set guards self-reference and mutual
// recursion). Exported aliases (`type Foo = bar`) get the same treatment
// against their RHS. Builtins (int, string, error, any, …) and in-scope
// generic type parameters are excluded. Qualified selectors (`pkg.Name`),
// container element types (slice/map/chan), function-typed fields, and
// anonymous structs are out of scope by design — the goal is the direct
// named-type position, which is where the leak typically lives. Test files
// (_test.go) are exempt: fixtures legitimately reach into package
// internals.
func check_exported_type_exposes_private(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	tok_file := file_set.File(file.Pos())
	if tok_file == nil {
		return nil
	}
	if strings.HasSuffix(tok_file.Name(), "_test.go") {
		return nil
	}

	same_file_types := check_exported_type_exposes_private_collect_types(file)
	for _, type_specification := range same_file_types {
		if !ast.IsExported(type_specification.Name.Name) {
			continue
		}
		entry_type_params := check_exported_type_exposes_private_type_params(
			type_specification.TypeParams)
		entry_name := type_specification.Name.Name
		if type_specification.Assign != token.NoPos {
			check_exported_type_exposes_private_check(
				&check_exported_type_exposes_private_check_input{
					File_Set:    file_set,
					Entry_Name:  entry_name,
					Expression:  type_specification.Type,
					Type_Params: entry_type_params,
					Diags:       &diags,
				})
			continue
		}
		struct_type, ok := type_specification.Type.(*ast.StructType)
		if !ok {
			continue
		}
		check_exported_type_exposes_private_walk(
			&check_exported_type_exposes_private_walk_input{
				File_Set:         file_set,
				Entry_Name:       entry_name,
				Root_Struct:      struct_type,
				Root_Type_Params: entry_type_params,
				Same_File_Types:  same_file_types,
				Diags:            &diags,
			})
	}
	return diags
}

func check_exported_type_exposes_private_collect_types(
	file *ast.File,
) (same_file_types map[string]*ast.TypeSpec) {
	same_file_types = map[string]*ast.TypeSpec{}
	for _, declaration := range file.Decls {
		generic_declaration, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		if generic_declaration.Tok != token.TYPE {
			continue
		}
		for _, specification := range generic_declaration.Specs {
			type_specification,
				is_type_specification := specification.(*ast.TypeSpec)
			if !is_type_specification {
				continue
			}
			same_file_types[type_specification.Name.Name] = type_specification
		}
	}
	return same_file_types
}

type check_exported_type_exposes_private_walk_input struct {
	File_Set         *token.FileSet
	Entry_Name       string
	Root_Struct      *ast.StructType
	Root_Type_Params map[string]bool
	Same_File_Types  map[string]*ast.TypeSpec
	Diags            *[]Diagnostic
}

// Iterative DFS over an exported struct type's transitive same-file struct
// fields. Recursion is banned in this package (check_no_recursion), so the
// walk pushes frames onto an explicit stack. Visited targets are tracked
// by type-spec name; cycle-safe by construction.
func check_exported_type_exposes_private_walk(
	input *check_exported_type_exposes_private_walk_input,
) {

	visited := map[string]bool{input.Entry_Name: true}
	stack := []exposed_type_frame{{input.Root_Struct, input.Root_Type_Params}}
	for len(stack) > 0 {
		top := len(stack) - 1
		current := stack[top]
		stack = stack[:top]
		if current.Struct_Type.Fields == nil {
			continue
		}
		for _, field := range current.Struct_Type.Fields.List {
			base := check_exported_type_exposes_private_unwrap_pointer(field.Type)
			identifier, ok := base.(*ast.Ident)
			if !ok {
				continue
			}
			if current.Type_Params[identifier.Name] {
				continue
			}
			if !ast.IsExported(identifier.Name) {
				switch identifier.Name {
				case "int", "int8", "int16", "int32", "int64",
					"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
					"float32", "float64", "complex64", "complex128",
					"bool",
					"string",
					"byte",
					"rune",
					"error",
					"any",
					"comparable":
					continue
				}
				*input.Diags = append(*input.Diags, Diagnostic{
					Position: input.File_Set.Position(identifier.Pos()),
					Message: fmt.Sprintf(
						"public type %s contains private type %s",
						input.Entry_Name, identifier.Name),
				})
				continue
			}
			target, ok := input.Same_File_Types[identifier.Name]
			if !ok {
				continue
			}
			if visited[target.Name.Name] {
				continue
			}
			target_struct, ok := target.Type.(*ast.StructType)
			if !ok {
				continue
			}
			visited[target.Name.Name] = true
			// Generic struct types only appear via IndexExpr/IndexListExpr, not
			// bare Ident.
			// The walk only recurses on bare-Ident targets, so target.TypeParams
			// is unreachable here.
			stack = append(stack, exposed_type_frame{
				Struct_Type: target_struct, Type_Params: current.Type_Params,
			})
		}
	}
}

type exposed_type_frame struct {
	Struct_Type *ast.StructType
	Type_Params map[string]bool
}

type check_exported_type_exposes_private_check_input struct {
	File_Set    *token.FileSet
	Entry_Name  string
	Expression  ast.Expr
	Type_Params map[string]bool
	Diags       *[]Diagnostic
}

// Alias check: `type Foo = bar` reveals bar through Foo's exported name even
// though no struct field is involved. Pointer-wrapped aliases (`type Foo = *bar`)
// are unwrapped the same way as field positions.
func check_exported_type_exposes_private_check(
	input *check_exported_type_exposes_private_check_input,
) {

	base := check_exported_type_exposes_private_unwrap_pointer(input.Expression)
	identifier, ok := base.(*ast.Ident)
	if !ok {
		return
	}
	if input.Type_Params[identifier.Name] {
		return
	}
	if ast.IsExported(identifier.Name) {
		return
	}
	switch identifier.Name {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"bool", "string", "byte", "rune", "error", "any", "comparable":
		return
	}
	*input.Diags = append(*input.Diags, Diagnostic{
		Position: input.File_Set.Position(identifier.Pos()),
		Message: fmt.Sprintf(
			"public type %s contains private type %s",
			input.Entry_Name,
			identifier.Name),
	})
}

func check_exported_type_exposes_private_unwrap_pointer(expression ast.Expr) (output ast.Expr) {

	output = expression
	for step := 0; ; step++ {
		star, is_star := output.(*ast.StarExpr)
		if !is_star {
			break
		}
		output = star.X
	}
	return output
}

func check_exported_type_exposes_private_type_params(
	field_list *ast.FieldList,
) (names map[string]bool) {

	names = map[string]bool{}
	if field_list == nil {
		return names
	}
	for _, field := range field_list.List {
		for _, name := range field.Names {
			names[name.Name] = true
		}
	}
	return names
}

// The iota identifier silently couples a constant's value to its position in
// the const block; reordering rows changes meaning without changing any
// expression. Spelling each value out makes order an editorial choice instead
// of a semantic one.
func check_no_iota(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	ast.Inspect(file, func(n ast.Node) (descend bool) {
		identifier, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		if identifier.Name != "iota" {
			return true
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(identifier.Pos()),
			Message:  "iota is banned",
		})
		return true
	})
	return diags
}

// Fallthrough makes a case silently run the next one — a control jump that is
// easy to miss and easy to leave dangling after an edit. Spell out the shared
// logic in each case instead.
func check_no_fallthrough(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	ast.Inspect(file, func(n ast.Node) (descend bool) {
		branch, ok := n.(*ast.BranchStmt)
		if !ok {
			return true
		}
		if branch.Tok != token.FALLTHROUGH {
			return true
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(branch.Pos()),
			Message:  "fallthrough is banned",
		})
		return true
	})
	return diags
}

// A blank import runs a package's init for its side effects alone, smuggling
// behavior in past the import list where no caller names it. Depend on the
// package explicitly instead.
func check_no_blank_import(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	for _, import_specification := range file.Imports {
		if import_specification.Name == nil {
			continue
		}
		if import_specification.Name.Name != "_" {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(import_specification.Pos()),
			Message:  "blank import is banned",
		})
	}
	return diags
}

// Parenthesized var/const/type groups put the visual weight on the block
// boundary rather than on each name, and smear unrelated bindings under one
// keyword — the reader has to scan into the block to learn what's being
// declared. Forcing one declaration per keyword anchors the eye on the
// identifier and keeps diffs honest about which name actually changed.
// import (...) is exempt: gofmt owns import block formatting and rewriting
// every import to its own line fights the formatter.
func check_no_grouped_declaration(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	for _, declaration := range file.Decls {
		generic_declaration, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		if generic_declaration.Tok == token.IMPORT {
			continue
		}
		if !generic_declaration.Lparen.IsValid() {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(generic_declaration.Pos()),
			Message:  "grouped declaration banned; split into one per line",
		})
	}
	return diags
}

// Struct tags drive reflection-based bindings that hide a contract inside a
// string literal: the field's behaviour at run time depends on unparsed text
// the compiler never inspects. Stdlib keys (json, xml, asn1) are the three
// the standard library itself consumes and so are permitted; everything else
// (yaml, validate, gorm, mapstructure, …) is third-party reflection and is
// banned. Walks every *ast.Field in the file; ast.Inspect already recurses
// into nested struct types and anonymous composites.
func check_no_third_party_struct_tag(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	stdlib_keys := map[string]bool{"json": true, "xml": true, "asn1": true}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		field, is_field := n.(*ast.Field)
		if !is_field {
			return true
		}
		if field.Tag == nil {
			return true
		}
		keys := check_no_third_party_struct_tag_parse_keys(field.Tag)
		for _, key := range keys {
			if stdlib_keys[key] {
				continue
			}
			diags = append(diags, Diagnostic{
				Position: file_set.Position(field.Tag.Pos()),
				Message: fmt.Sprintf("struct tag key %q is not stdlib; only "+
					"json, xml, and asn1 are permitted", key),
			})
		}
		return true
	})
	return diags
}

// Parses the keys out of a struct tag literal as it appears in source —
// either a raw-string (`json:"name"`) or interpreted-string ("json:\"name\"")
// form. Returns the keys in declaration order. Tag value contents are
// ignored; only the key tokens to the left of each colon are extracted.
// Mirrors stdlib reflect.StructTag.Lookup parsing without that helper's
// per-key API.
func check_no_third_party_struct_tag_parse_keys(tag *ast.BasicLit) (keys []string) {
	raw, err := strconv.Unquote(tag.Value)
	if err != nil {
		return nil
	}
	for len(raw) > 0 {
		for len(raw) > 0 {
			if raw[0] != ' ' {
				if raw[0] != '\t' {
					break
				}
			}
			raw = raw[1:]
		}
		if len(raw) == 0 {
			break
		}
		colon_offset := strings.IndexByte(raw, ':')
		if colon_offset <= 0 {
			break
		}
		keys = append(keys, raw[:colon_offset])
		rest := raw[colon_offset+1:]
		if len(rest) == 0 {
			break
		}
		if rest[0] != '"' {
			break
		}
		end := 1
		for end < len(rest) {
			if rest[end] == '\\' {
				if end+1 < len(rest) {
					end += 2
					continue
				}
			}
			if rest[end] == '"' {
				end++
				break
			}
			end++
		}
		if end > len(rest) {
			break
		}
		raw = rest[end:]
	}
	return keys
}

// Flags any struct field whose only name is `_` and whose type resolves to a
// stdlib `sync.Mutex` / `sync.RWMutex`. The blank form provides no usable
// receiver to call Lock on, so the only practical effect is to disable
// opaque-on-mutex sibling-field recursion at the assertion layer — that is
// not the legitimate use of a mutex. Callers wanting copy-prevention should
// use the `noCopy` idiom; callers wanting an actual lock should name the
// field.
func check_blank_synchronization_mutex(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	stdlib_imports := collect_stdlib_imports(file)
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		struct_type, is_struct := n.(*ast.StructType)
		if !is_struct {
			return true
		}
		if struct_type.Fields == nil {
			return true
		}
		for _, field := range struct_type.Fields.List {
			if !type_expression_is_mutex(
				field.Type, stdlib_imports) {
				continue
			}
			for _, name := range field.Names {
				if name.Name != "_" {
					continue
				}
				diags = append(diags, Diagnostic{
					Position: file_set.Position(name.Pos()),
					Message: "blank-named sync mutex has no usable Lock " +
						"receiver; " +
						"use the noCopy idiom for non-copy semantics or " +
						"name the field to lock it",
				})
			}
		}
		return true
	})
	return diags
}

// Required-shape invariant assertions carry numeric bounds — Distinct_Boundary's
// Lo/Hi, and the RHS of `Always(x op N)` / `Sometimes(x op N)` comparisons. An
// inline literal at one of those positions has no name: the reader can't tell
// which budget it represents, the linter can't grep for related sites, and
// identical literals scattered through the codebase drift independently. The
// rule: every bound is either a single named identifier (or selector chain),
// or a zero-state sentinel (literal 0, "", or nil) — never an inline numeric
// literal, signed literal, arithmetic expression, or typed conversion.
func check_assertion_named_constant(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	invariant_idents := collect_invariant_idents(file)
	if len(invariant_idents) == 0 {
		return nil
	}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		call, is_call := n.(*ast.CallExpr)
		if !is_call {
			return true
		}
		selector, is_selector := call.Fun.(*ast.SelectorExpr)
		if !is_selector {
			return true
		}
		identifier, is_identifier := selector.X.(*ast.Ident)
		if !is_identifier {
			return true
		}
		if !invariant_idents[identifier.Name] {
			return true
		}
		name := selector.Sel.Name
		if name == "Distinct_Boundary" {
			diags = append(diags,
				check_assertion_named_constant_distinct_boundary(file_set, call)...)
			return true
		}
		if name == "Always" {
			diags = append(diags,
				check_assertion_named_constant_predicate(file_set, call)...)
			return true
		}
		if name == "Sometimes" {
			diags = append(diags,
				check_assertion_named_constant_predicate(file_set, call)...)
			return true
		}
		return true
	})
	return diags
}

func check_assertion_named_constant_distinct_boundary(
	file_set *token.FileSet, call *ast.CallExpr,
) (diags []Diagnostic) {
	if len(call.Args) == 0 {
		return nil
	}
	expression := call.Args[0]
	if unary, is_unary := expression.(*ast.UnaryExpr); is_unary {
		expression = unary.X
	}
	composite, is_composite := expression.(*ast.CompositeLit)
	if !is_composite {
		return nil
	}
	for _, element := range composite.Elts {
		key_value, is_key_value := element.(*ast.KeyValueExpr)
		if !is_key_value {
			continue
		}
		key_identifier, is_key_identifier := key_value.Key.(*ast.Ident)
		if !is_key_identifier {
			continue
		}
		if key_identifier.Name != "Lo" {
			if key_identifier.Name != "Hi" {
				continue
			}
		}
		if assertion_bound_is_valid(key_value.Value) {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(key_value.Value.Pos()),
			Message: "assertion bound must be a file-level named constant, not an " +
				"inline literal",
		})
	}
	return diags
}

func check_assertion_named_constant_predicate(
	file_set *token.FileSet, call *ast.CallExpr,
) (diags []Diagnostic) {
	if len(call.Args) == 0 {
		return nil
	}
	predicate := call.Args[0]
	for step := 0; ; step++ {
		paren, is_paren := predicate.(*ast.ParenExpr)
		if !is_paren {
			break
		}
		predicate = paren.X
	}
	binary, is_binary := predicate.(*ast.BinaryExpr)
	if !is_binary {
		return nil
	}
	op := binary.Op
	is_comparison := false
	if op == token.EQL {
		is_comparison = true
	}
	if op == token.NEQ {
		is_comparison = true
	}
	if op == token.LSS {
		is_comparison = true
	}
	if op == token.GTR {
		is_comparison = true
	}
	if op == token.LEQ {
		is_comparison = true
	}
	if op == token.GEQ {
		is_comparison = true
	}
	if !is_comparison {
		return nil
	}
	if assertion_bound_is_valid(binary.Y) {
		return nil
	}
	return []Diagnostic{{
		Position: file_set.Position(binary.Y.Pos()),
		Message: "assertion bound must be a file-level named constant, not an inline " +
			"literal",
	}}
}

// Assertion_bound_is_valid reports whether expression is an allowed shape at
// the bound position of a required-shape invariant assertion: a single
// identifier, a selector chain bottoming out at an identifier, the literal
// nil, or a zero-state sentinel (literal 0, 0.0, signed zero, or empty
// string).
func assertion_bound_is_valid(expression ast.Expr) (yes bool) {
	for step := 0; ; step++ {
		paren, is_paren := expression.(*ast.ParenExpr)
		if !is_paren {
			break
		}
		expression = paren.X
	}
	if identifier, is_identifier := expression.(*ast.Ident); is_identifier {
		return identifier.Name != "_"
	}
	if selector, is_selector := expression.(*ast.SelectorExpr); is_selector {
		current := ast.Expr(selector)
		for step := 0; ; step++ {
			inner_selector, is_inner_selector := current.(*ast.SelectorExpr)
			if !is_inner_selector {
				break
			}
			current = inner_selector.X
		}
		_, bottom_is_identifier := current.(*ast.Ident)
		return bottom_is_identifier
	}
	if basic, is_basic := expression.(*ast.BasicLit); is_basic {
		return basic_lit_is_zero_or_empty(basic)
	}
	if unary, is_unary := expression.(*ast.UnaryExpr); is_unary {
		if unary.Op != token.ADD {
			if unary.Op != token.SUB {
				return false
			}
		}
		inner_basic, inner_is_basic := unary.X.(*ast.BasicLit)
		if !inner_is_basic {
			return false
		}
		return basic_lit_is_zero_or_empty(inner_basic)
	}
	return false
}

func basic_lit_is_zero_or_empty(basic *ast.BasicLit) (yes bool) {

	if basic.Kind == token.STRING {
		if basic.Value == `""` {
			return true
		}
		if basic.Value == "``" {
			return true
		}
		return false
	}
	if basic.Kind == token.INT {
		return basic.Value == "0"
	}
	if basic.Kind == token.FLOAT {
		f, err := strconv.ParseFloat(basic.Value, 64)
		if err != nil {
			return false
		}
		return f == 0
	}
	return false
}

// Positional struct literals break silently when fields are added or reordered.
// Without go/types we can only be certain about same-file struct declarations;
// cross-file and cross-package literals are skipped to keep false positives at
// zero. The full check would require type information.
func check_keyed_struct_init(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	struct_names := map[string]bool{}
	for _, declaration := range file.Decls {
		generic_declaration, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		if generic_declaration.Tok != token.TYPE {
			continue
		}
		for _, specification := range generic_declaration.Specs {
			type_specification, is_type_specification := specification.(*ast.TypeSpec)
			if !is_type_specification {
				continue
			}
			_, is_struct := type_specification.Type.(*ast.StructType)
			if !is_struct {
				continue
			}
			struct_names[type_specification.Name.Name] = true
		}
	}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		name := check_keyed_struct_init_type_ident(lit.Type)
		if name == "" {
			return true
		}
		if !struct_names[name] {
			return true
		}
		if len(lit.Elts) == 0 {
			return true
		}
		for _, e := range lit.Elts {
			_, is_kv := e.(*ast.KeyValueExpr)
			if is_kv {
				continue
			}
			diags = append(diags, Diagnostic{
				Position: file_set.Position(lit.Pos()),
				Message:  fmt.Sprintf("%s literal must use keyed fields", name),
			})
			return true
		}
		return true
	})
	return diags
}

func check_keyed_struct_init_type_ident(expression ast.Expr) (name string) {

	for step := 0; ; step++ {
		star, is_star := expression.(*ast.StarExpr)
		if !is_star {
			break
		}
		expression = star.X
	}
	identifier, is_ident := expression.(*ast.Ident)
	if !is_ident {
		return ""
	}
	return identifier.Name
}

// The gofmt tool is the canonical Go formatter; deviating from it creates
// noise in diffs and pulls editor cursors around. We emit one diagnostic per
// file — localizing hunks would re-implement gofmt's diff logic for no real
// gain over `gofmt -w`.
func check_gofmt(file_set *token.FileSet, file *ast.File, source []byte) (diags []Diagnostic) {

	if len(source) == 0 {
		return nil
	}
	formatted, err := format.Source(source)
	if err != nil {
		return nil
	}
	if bytes.Equal(formatted, source) {
		return nil
	}
	filename := ""
	tok_file := file_set.File(file.Pos())
	if tok_file != nil {
		filename = tok_file.Name()
	}
	return []Diagnostic{{
		Position: token.Position{Filename: filename, Line: 1, Column: 1},
		Message:  "file is not gofmt-clean",
	}}
}

// Dot imports inject names into the file scope, breaking grep-for-origin and
// inviting collisions. Always import with an explicit name (or package name).
func check_no_dot_import(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	for _, import_specification := range file.Imports {
		if import_specification.Name == nil {
			continue
		}
		if import_specification.Name.Name != "." {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(import_specification.Pos()),
			Message:  "dot import is banned",
		})
	}
	return diags
}

// A composition-tier package lives in a directory named `default` nested under
// its library and re-exports it. It must declare the package clause of its
// parent directory — `foo/default` declares `package foo` — so that importing
// `foo/default` binds to `foo` and shadows the library, letting callers read as
// if no split had happened (and letting snap.Edit's source-line rewriter find
// the literal `snap.Edit(` it searches for). The directory name `default` is a
// Go keyword and so cannot itself be a package name; the parent name is the
// natural and required choice.
func check_default_package_name(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	tok_file := file_set.File(file.Pos())
	if tok_file == nil {
		return nil
	}
	directory := path.Dir(tok_file.Name())
	if path.Base(directory) != "default" {
		return nil
	}
	parent := path.Base(path.Dir(directory))
	if parent == "." {
		return nil
	}
	if parent == "/" {
		return nil
	}
	// Strip _test so an external test package (`package foo_test`) in the
	// default directory is judged by its base name, not rejected outright.
	if strings.TrimSuffix(file.Name.Name, "_test") == parent {
		return nil
	}
	diags = append(diags, Diagnostic{
		Position: file_set.Position(file.Name.Pos()),
		Name:     "default-package-name",
		Want:     fmt.Sprintf("package %s", parent),
		Message: fmt.Sprintf(
			"default package must declare 'package %s', not 'package %s'; it "+
				"shadows the library it re-exports",
			parent, file.Name.Name),
	})
	return diags
}

// Whitebox test packages couple tests to internals; main packages cannot be
// blackbox-tested coherently. Force every _test.go to declare `package
// <X>_test`, which keeps the test suite restricted to the same public API
// callers see and prevents tests from being written against `package main`.
func check_test_package(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	tok_file := file_set.File(file.Pos())
	if tok_file == nil {
		return nil
	}
	if !strings.HasSuffix(tok_file.Name(), "_test.go") {
		return nil
	}
	name := file.Name.Name
	flag := false
	if name == "main" {
		flag = true
	}
	if name == "main_test" {
		flag = true
	}
	if !strings.HasSuffix(name, "_test") {
		flag = true
	}
	if !flag {
		return nil
	}
	return []Diagnostic{{
		Position: file_set.Position(file.Name.Pos()),
		Message: fmt.Sprintf("test file must declare 'package <X>_test'; got 'package "+
			"%s'", name),
	}}
}

// Flags any tokenized word in a declared name that appears in the
// word_replacements table. This is the single home for two related
// naming rules sharing one table: abbreviations get a `rename x -> ...`
// suggestion built from their candidate expansions, and banned words (no
// candidate) get an `identifier "x" contains banned substring "y"` diagnostic.
// One table means the flagged-word list lives in exactly one place.
//
// The package name and file name are checked directly; declared identifiers are
// walked by check_names_walk_decls (function names, receivers, params, named
// returns, type names, struct fields, and body var/const/`:=`/range defines).
// Use sites are not visited, so the `len(xs)` and `cap(xs)` builtins are exempt.
// Func-type signature names and closure parameters are deliberately not walked:
// those names are documentation-only and idiomatic abbreviations there (e.g.
// `info fs.FileInfo`) are not the target. "helper" is banned only in function
// names — see check_names_vocabulary_function_ban.
func check_names_vocabulary(
	file_set *token.FileSet, file *ast.File, table map[string][]string,
) (diags []Diagnostic) {

	diags = append(diags,
		check_names_vocabulary_at(
			file_set.Position(file.Name.Pos()), file.Name.Name, table)...)
	diags = append(diags, check_names_vocabulary_file_name(file_set, file, table)...)
	diags = append(diags, check_names_vocabulary_function_ban(file_set, file)...)
	check_names_walk_decls(file, func(identifier *ast.Ident) {
		position := file_set.Position(identifier.Pos())
		diags = append(diags,
			check_names_vocabulary_at(position, identifier.Name, table)...)
	})
	return diags
}

// Binds the word-replacements table (decoded from lint.json's word_replacements)
// into a check_function, mirroring
// make_check_no_package_vars. Threading the table rather than reaching for a
// package global keeps the check pure and lets tests drive it from a fixture.
func make_check_names_vocabulary(table map[string][]string) (checker check_function) {
	return func(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
		return check_names_vocabulary(file_set, file, table)
	}
}

// Flags "helper" in a function name. The ban is function-name-only — a file or
// package named helper is a weaker smell than a function whose name hides what
// it does — so it rides outside the universal word_replacements table,
// which applies at every declaration site.
func check_names_vocabulary_function_ban(
	file_set *token.FileSet, file *ast.File,
) (diags []Diagnostic) {

	ast.Inspect(file, func(n ast.Node) (descend bool) {
		function, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		for _, word := range suggest_split_words(function.Name.Name) {
			if strings.EqualFold(word, "helper") {
				diags = append(diags, Diagnostic{
					Position: file_set.Position(function.Name.Pos()),
					Message: fmt.Sprintf(
						`identifier %q contains banned substring "helper"`,
						function.Name.Name),
				})
				return true
			}
		}
		return true
	})
	return diags
}

// Resolves the file's basename to the stem used for word-splitting: the .go
// suffix (and a _test suffix) are stripped first so the "test" segment in
// foo_test.go is not itself treated as a flagged word. The position is
// synthetic (line 1, column 1) because a file name has no token in the source.
func check_names_vocabulary_file_name(
	file_set *token.FileSet, file *ast.File, table map[string][]string,
) (diags []Diagnostic) {

	tok_file := file_set.File(file.Pos())
	if tok_file == nil {
		return nil
	}
	filename := tok_file.Name()
	stem := strings.TrimSuffix(path.Base(filename), ".go")
	stem = strings.TrimSuffix(stem, "_test")
	return check_names_vocabulary_at(
		token.Position{Filename: filename, Line: 1, Column: 1}, stem, table)
}

// Emits one diagnostic per flagged word found in name. A word with
// candidate expansions yields a rename suggestion; a banned word (empty
// candidate list) yields the banned-substring diagnostic. Blank "_" is skipped.
func check_names_vocabulary_at(
	position token.Position, name string, table map[string][]string,
) (diags []Diagnostic) {

	if name == "_" {
		return nil
	}
	words := suggest_split_words(name)
	style := "snake_case"
	if ada_case_re.MatchString(name) {
		style = "Ada_Case"
	}
	for word_index, w := range words {
		lower := strings.ToLower(w)
		candidates := word_replacements_for(table, lower)
		if candidates == nil {
			continue
		}
		message := check_names_vocabulary_message(&check_names_vocabulary_message_input{
			Name: name, Word: lower, Words: words,
			Word_Index: word_index, Candidates: candidates, Style: style,
		})
		diags = append(diags, Diagnostic{Position: position, Message: message})
	}
	return diags
}

type check_names_vocabulary_message_input struct {
	Name       string
	Word       string
	Words      []string
	Word_Index int
	Candidates []string
	Style      string
}

// Renders the diagnostic text. No candidates means the word is banned outright.
// One candidate renders a bare `rename x -> y`; several render
// `rename x -> [a, b, c]`. Each candidate is substituted into the offending
// word slot so the author sees a drop-in replacement, not just the bare word —
// e.g. `foo_id` produces `foo_identifier`, not `id -> identifier`.
func check_names_vocabulary_message(input *check_names_vocabulary_message_input) (message string) {

	if len(input.Candidates) == 0 {
		return fmt.Sprintf(
			"identifier %q contains banned substring %q", input.Name, input.Word)
	}
	renames := make([]string, len(input.Candidates))
	for candidate_index, candidate := range input.Candidates {
		substituted := append([]string{}, input.Words...)
		substituted[input.Word_Index] = candidate
		renames[candidate_index] = suggest(&suggest_input{
			Name: strings.Join(substituted, "_"), Want: input.Style})
	}
	if len(renames) == 1 {
		return fmt.Sprintf("rename %s -> %s", input.Name, renames[0])
	}
	return fmt.Sprintf("rename %s -> [%s]", input.Name, strings.Join(renames, ", "))
}

// Functions with two or more parameters of the same type are call-site
// landmines: `transfer(source, dst)` can be silently swapped to
// `transfer(dst, source)` with no compiler protest. Force such signatures to
// take a pointer to a named input struct declared directly above; call sites
// then read as `transfer(&Transfer_Input{Src: ..., Dst: ...})` and re-orderings
// become compile errors.
func check_input_struct(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	for _, declaration := range file.Decls {
		function_declaration, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if !check_input_struct_should_trigger(function_declaration) {
			continue
		}
		want_name := check_input_struct_expected_name(function_declaration.Name.Name)
		diag := check_input_struct_validate(file_set, function_declaration, want_name)
		if diag != nil {
			diags = append(diags, *diag)
		}
	}
	return diags
}

func check_input_struct_should_trigger(function *ast.FuncDecl) (trigger bool) {

	if function.Type.Params == nil {
		return false
	}
	counts := map[string]int{}
	for _, f := range function.Type.Params.List {
		if _, is_variadic := f.Type.(*ast.Ellipsis); is_variadic {
			continue
		}
		key := types.ExprString(f.Type)
		name_count := len(f.Names)
		if name_count == 0 {
			name_count = 1
		}
		counts[key] += name_count
		if counts[key] >= 2 {
			return true
		}
	}
	return false
}

func check_input_struct_expected_name(function_name string) (want string) {

	suffix := "_Input"
	if len(function_name) > 0 {
		if unicode.IsLower(rune(function_name[0])) {
			suffix = "_input"
		}
	}
	return function_name + suffix
}

func check_input_struct_validate(
	file_set *token.FileSet,
	function *ast.FuncDecl,
	want_name string,
) (diag *Diagnostic) {
	non_variadic := check_input_struct_validate_non_variadic_params(function)
	if len(non_variadic) == 1 {
		if len(non_variadic[0].Names) == 1 {
			// Trigger fires only when some type appears ≥2 times in the param
			// list. A single 1-name field contributes exactly 1 entry, so the
			// trigger cannot fire on this shape — reaching here means
			// check_input_struct_should_trigger lied.
		}
	}
	return &Diagnostic{
		Position: file_set.Position(function.Pos()),
		Message: "convert to " + check_input_struct_validate_suggest_sig(
			function, want_name),
	}
}

func check_input_struct_validate_suggest_sig(
	function *ast.FuncDecl, want_name string,
) (sig string) {

	var sb strings.Builder
	sb.WriteString(function.Name.Name)
	sb.WriteString("(*")
	sb.WriteString(want_name)
	if function.Type.Params != nil {
		for _, f := range function.Type.Params.List {
			if _, is_variadic := f.Type.(*ast.Ellipsis); !is_variadic {
				continue
			}
			type_string := types.ExprString(f.Type)
			if len(f.Names) == 0 {
				sb.WriteString(", ")
				sb.WriteString(type_string)
				continue
			}
			for _, n := range f.Names {
				sb.WriteString(", ")
				sb.WriteString(n.Name)
				sb.WriteString(" ")
				sb.WriteString(type_string)
			}
		}
	}
	sb.WriteString(")")
	if function.Type.Results == nil {
		return sb.String()
	}
	if len(function.Type.Results.List) == 0 {
		return sb.String()
	}
	sb.WriteString(" (")
	first := true
	for _, f := range function.Type.Results.List {
		type_string := types.ExprString(f.Type)
		if len(f.Names) == 0 {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(type_string)
			first = false
			continue
		}
		for _, n := range f.Names {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(n.Name)
			sb.WriteString(" ")
			sb.WriteString(type_string)
			first = false
		}
	}
	sb.WriteString(")")
	return sb.String()
}

func check_input_struct_validate_non_variadic_params(function *ast.FuncDecl) (output []*ast.Field) {

	if function.Type.Params == nil {
		return nil
	}
	for _, f := range function.Type.Params.List {
		if _, is_variadic := f.Type.(*ast.Ellipsis); is_variadic {
			continue
		}
		output = append(output, f)
	}
	return output
}

// Empty-body functions are dead weight: either the function is unfinished, or
// it's a marker method satisfying an interface — both are better expressed
// explicitly (a panic with a TODO, or moving the marker to a typed sentinel).
// Interface method signatures have Body == nil and are unaffected.
func check_no_empty_function_body(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	for _, declaration := range file.Decls {
		function_declaration, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function_declaration.Body == nil {
			continue
		}
		if len(function_declaration.Body.List) > 0 {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(function_declaration.Pos()),
			Message: fmt.Sprintf(
				"func %s has an empty body", function_declaration.Name.Name),
		})
	}
	return diags
}

// `func init` runs implicitly at package load, scattering startup logic across
// files in an order that depends on filename sort. An explicit, named
// initialization function called from `main` keeps control flow visible.
func check_no_function_init(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	for _, declaration := range file.Decls {
		function_declaration, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function_declaration.Recv != nil {
			continue
		}
		if function_declaration.Name.Name != "init" {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(function_declaration.Pos()),
			Message:  "func init is banned; expose a func Init() instead",
		})
	}
	return diags
}

// Interface method sets are banned. Methods exist to make a concrete type fit
// a contract; once the contract concept is removed, every method that does
// not satisfy a stdlib interface is just dressed-up free-function syntax.
// Type-element interfaces (generic constraints built from unions and
// approximations like `~int | ~int64`) carry no method set and are allowed.
// `any` / bare `interface{}` is allowed as the empty interface.
//
// Detection rule: any *ast.InterfaceType whose Methods.List contains at least
// one *ast.Field with a non-empty Names slice (a method element). Embedded
// interface names (Names empty, Type is Ident/SelectorExpr) are not flagged
// here because at the AST level they are indistinguishable from
// type-set constraints; they fall out naturally once the underlying
// method-set interfaces are removed.
func check_no_interfaces(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	ast.Inspect(file, func(n ast.Node) (recurse bool) {
		interface_type, ok := n.(*ast.InterfaceType)
		if !ok {
			return true
		}
		if interface_type.Methods == nil {
			return true
		}
		for _, f := range interface_type.Methods.List {
			if len(f.Names) == 0 {
				continue
			}
			diags = append(diags, Diagnostic{
				Position: file_set.Position(interface_type.Pos()),
				Message:  "interface declarations are banned (except for generics)",
			})
			return true
		}
		return true
	})
	return diags
}

// Reports whether the workspace-root-relative directory is a listed instrumentation
// package or sits in one's subtree. Segment-prefix so a family entry
// (shared/invariant) covers its versions and default tier; entries are path-cleaned
// so "./pkg/" and "pkg" name the same directory.
func instrumentation_match(directory string, packages []string) (yes bool) {
	for _, entry := range packages {
		clean := path.Clean(entry)
		if directory == clean {
			return true
		}
		if strings.HasPrefix(directory, clean+"/") {
			return true
		}
	}
	return false
}

// Reports whether the import resolves to a first-party package at or under a listed
// instrumentation package. Instrumentation is write-only — emitting to it cannot
// feed impurity or nondeterminism back into the importer — so pure and
// deterministic packages may import it despite the import bans.
func import_path_is_instrumentation(
	import_path string, modules *module_index, packages []string,
) (yes bool) {
	module_index_number := module_index_for_import_path(import_path, modules)
	if module_index_number < 0 {
		return false
	}
	m := modules.Modules[module_index_number]
	return instrumentation_match(import_path_workspace_directory(import_path, m), packages)
}

// Binds the instrumentation list into the package-var check. The check_function
// signature carries no config of its own, so the list (needed by the var-Default
// exemption) is captured in a closure built per run.
func make_check_no_package_vars(instrumentation []string) (checker check_function) {
	return func(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
		return check_no_package_vars(file_set, file, instrumentation)
	}
}

// Package-level `var` creates implicit mutable state at package load with
// no obvious initialization order, complicating tests and reasoning. Only
// two initializers are exempted: regexp.MustCompile (no const regex type)
// and errors.New (no const error type). The `var _ Iface = (*Impl)(nil)`
// shape is also exempted — it declares no value, just asks the compiler
// to verify Impl satisfies Iface.
func check_no_package_vars(
	file_set *token.FileSet, file *ast.File, instrumentation []string,
) (diags []Diagnostic) {

	const base_message = "package-level var is banned" +
		" (except for regexp.MustCompile and errors.New)"
	const switch_hint = ", or use a switch for lookup tables"
	for _, declaration := range file.Decls {
		generic_declaration, is_generic_declaration := declaration.(*ast.GenDecl)
		if !is_generic_declaration {
			continue
		}
		if generic_declaration.Tok != token.VAR {
			continue
		}
		// A //go:embed directive can only attach to a package-level var, so such
		// a var has no function-scope alternative and is exempt.
		if check_no_package_vars_is_embed(generic_declaration) {
			continue
		}
		for _, specification := range generic_declaration.Specs {
			vs, is_vs := specification.(*ast.ValueSpec)
			if !is_vs {
				continue
			}
			if check_no_package_vars_is_default(file_set, file, vs, instrumentation) {
				continue
			}
			if check_no_package_vars_all_allowed(vs) {
				continue
			}
			message := base_message
			if check_no_package_vars_is_map_or_slice_literal(vs) {
				message += switch_hint
			}
			diags = append(diags, Diagnostic{
				Position: file_set.Position(vs.Pos()),
				Message:  message,
			})
		}
	}
	return diags
}

// Detects the literal-table shape (`var T = map[K]V{...}` or
// `var T = []E{...}`) so the diagnostic can nudge the user toward a
// switch. Switches are zero-allocation, refuse to compile on missing
// cases when paired with exhaustiveness tooling, and surface the
// table's logic at the call site instead of behind an identifier.
func check_no_package_vars_is_map_or_slice_literal(vs *ast.ValueSpec) (yes bool) {

	if len(vs.Values) == 0 {
		return false
	}
	for _, v := range vs.Values {
		lit, ok := v.(*ast.CompositeLit)
		if !ok {
			return false
		}
		switch lit.Type.(type) {
		case *ast.MapType, *ast.ArrayType:
			continue
		default:
			return false
		}
	}
	return true
}

// Composition-tier packages are allowed to expose a single `var Default = …`
// binding — that's literally the shape they exist for. The package's directory
// (workspace-root-relative) must be at or under a listed instrumentation_packages
// entry; the literal `default/` directory name confers nothing on its own. Allowed
// only for the literal name "Default" and only as a single-name
// single-initializer spec.
func check_no_package_vars_is_default(
	file_set *token.FileSet, file *ast.File, vs *ast.ValueSpec, instrumentation []string,
) (yes bool) {

	tok_file := file_set.File(file.Pos())
	if tok_file == nil {
		return false
	}
	if !instrumentation_match(path.Dir(tok_file.Name()), instrumentation) {
		return false
	}
	if len(vs.Names) != 1 {
		return false
	}
	if vs.Names[0].Name != "Default" {
		return false
	}
	return len(vs.Values) == 1
}

// Allowed only when every declared name has a paired initializer that is
// a call to regexp.MustCompile or errors.New. A zero-value declaration
// (no Values) fails this check by construction (len mismatch), which is
// the intended behavior — no package-level zero-value state.
// Reports whether the var declaration carries a //go:embed directive in its doc
// comment. Such a var must live at package scope (the directive is rejected
// elsewhere), so the package-var ban does not apply.
func check_no_package_vars_is_embed(declaration *ast.GenDecl) (embedded bool) {
	if declaration.Doc == nil {
		return false
	}
	for _, comment := range declaration.Doc.List {
		if strings.HasPrefix(comment.Text, "//go:embed ") {
			return true
		}
	}
	return false
}

func check_no_package_vars_all_allowed(vs *ast.ValueSpec) (yes bool) {

	if len(vs.Values) == 0 {
		return false
	}
	if len(vs.Values) != len(vs.Names) {
		return false
	}
	for _, v := range vs.Values {
		call, ok := v.(*ast.CallExpr)
		if !ok {
			return false
		}
		selector_expression, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		package_identifier, ok := selector_expression.X.(*ast.Ident)
		if !ok {
			return false
		}
		qualified := package_identifier.Name + "." + selector_expression.Sel.Name
		switch qualified {
		case "regexp.MustCompile", "errors.New":
		default:
			return false
		}
	}
	return true
}

// Receiver methods whose name+signature does not match a known stdlib
// interface method are dressed-up free functions. With user-defined interfaces
// banned (check_no_interfaces), the only legitimate satisfaction targets are
// stdlib interfaces; their methods form a small fixed set whose signatures
// can be matched syntactically. Third-party interface satisfaction is not
// accommodated — convert to a free function whose first parameter is the
// former receiver.
//
// Matching is by joined rendered type strings: each param/result list becomes
// a comma-separated string ("[]byte" or "int,error" or ""), and the lookup
// is a switch keyed on method name. `any` and `interface{}` both render as
// "any" (the empty interface). Pointers, slices, ellipsis, and qualified
// types render directly from the AST.
func check_unnecessary_method(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	for _, declaration := range file.Decls {
		function_declaration, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function_declaration.Recv == nil {
			continue
		}
		params := check_unnecessary_method_field_list_types(
			function_declaration.Type.Params)
		results := check_unnecessary_method_field_list_types(
			function_declaration.Type.Results)
		match := check_unnecessary_method_matches_stdlib(
			&check_unnecessary_method_matches_stdlib_input{
				Name:    function_declaration.Name.Name,
				Params:  strings.Join(params, ","),
				Results: strings.Join(results, ","),
			})
		if match {
			continue
		}
		message := fmt.Sprintf(
			"method %s does not satisfy any stdlib interface; "+
				"convert to a free function with the receiver as the first "+
				"parameter",
			function_declaration.Name.Name,
		)
		diags = append(diags, Diagnostic{
			Position: file_set.Position(function_declaration.Name.Pos()),
			Message:  message,
		})
	}
	return diags
}

type check_unnecessary_method_matches_stdlib_input struct {
	Name    string
	Params  string
	Results string
}

func check_unnecessary_method_matches_stdlib(
	input *check_unnecessary_method_matches_stdlib_input,
) (yes bool) {

	return check_unnecessary_method_matches_stdlib_input_signature(input)
}

func check_unnecessary_method_matches_stdlib_input_signature(
	input *check_unnecessary_method_matches_stdlib_input,
) (yes bool) {
	switch input.Name {
	case "Error", "String", "GoString":
		return input.Params == "" && input.Results == "string"
	case "Read", "Write":
		return input.Params == "[]byte" && input.Results == "int,error"
	case "Close":
		return input.Params == "" && input.Results == "error"
	case "Seek":
		return input.Params == "int64,int" && input.Results == "int64,error"
	case "WriteTo":
		return input.Params == "io.Writer" && input.Results == "int64,error"
	case "ReadFrom":
		return input.Params == "io.Reader" && input.Results == "int64,error"
	case "Len":
		return input.Params == "" && input.Results == "int"
	case "Less":
		return input.Params == "int,int" && input.Results == "bool"
	case "Swap":
		return input.Params == "int,int" && input.Results == ""
	case "MarshalJSON", "MarshalText", "MarshalBinary":
		return input.Params == "" && input.Results == "[]byte,error"
	case "UnmarshalJSON", "UnmarshalText", "UnmarshalBinary":
		return input.Params == "[]byte" && input.Results == "error"
	case "Format":
		return input.Params == "fmt.State,rune" && input.Results == ""
	case "Set":
		return input.Params == "string" && input.Results == "error"
	case "Scan":
		return input.Params == "any" && input.Results == "error"
	case "Visit":
		return input.Params == "ast.Node" && input.Results == "ast.Visitor"
	case "Open":
		return input.Params == "string" && input.Results == "fs.File,error"
	case "ReadFile":
		return input.Params == "string" && input.Results == "[]byte,error"
	case "ReadDir":
		return input.Params == "string" && input.Results == "[]fs.DirEntry,error"
	case "Stat":
		switch input.Params {
		case "":
			return input.Results == "fs.FileInfo,error"
		case "string":
			return input.Results == "fs.FileInfo,error"
		}
		return false
	case "Name":
		return input.Params == "" && input.Results == "string"
	case "Size":
		return input.Params == "" && input.Results == "int64"
	case "Mode":
		return input.Params == "" && input.Results == "fs.FileMode"
	case "ModTime":
		return input.Params == "" && input.Results == "time.Time"
	case "IsDir":
		return input.Params == "" && input.Results == "bool"
	case "Sys":
		return input.Params == "" && input.Results == "any"
	case "Type":
		return input.Params == "" && input.Results == "fs.FileMode"
	case "Info":
		return input.Params == "" && input.Results == "fs.FileInfo,error"
	}
	return false
}

// Flattens a FieldList into one rendered type string per declared name. A
// field with no names contributes a single entry (e.g., `(string)` →
// ["string"]), while a field with N names contributes N entries (e.g.,
// `(a, b int)` → ["int", "int"]).
func check_unnecessary_method_field_list_types(fl *ast.FieldList) (output_list []string) {

	if fl == nil {
		return nil
	}
	for _, f := range fl.List {
		rendered := check_unnecessary_method_field_list_types_render_type(f.Type)
		count := len(f.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			output_list = append(output_list, rendered)
		}
	}
	return output_list
}

// Renders an ast.Expr representing a type into a canonical string. The outer
// loop strips type prefixes (`*`, `[]`, `...`) onto a string accumulator
// without recursion; the inner switch handles base cases. Anything outside
// this set returns a sentinel that cannot match a stdlib table entry, so
// unusual signatures correctly fall through to the "not stdlib" diagnostic.
func check_unnecessary_method_field_list_types_render_type(
	expression ast.Expr,
) (output_string string) {

	prefix := ""
	for step := 0; ; step++ {
		stripped := false
		switch e := expression.(type) {
		case *ast.StarExpr:
			prefix += "*"
			expression = e.X
			stripped = true
		case *ast.ArrayType:
			if e.Len != nil {
				return "<unknown>"
			}
			prefix += "[]"
			expression = e.Elt
			stripped = true
		case *ast.Ellipsis:
			prefix += "..."
			expression = e.Elt
			stripped = true
		}
		if !stripped {
			break
		}
	}
	switch e := expression.(type) {
	case *ast.Ident:
		return prefix + e.Name
	case *ast.SelectorExpr:
		package_identifier, ok := e.X.(*ast.Ident)
		if !ok {
			return "<unknown>"
		}
		return prefix + package_identifier.Name + "." + e.Sel.Name
	case *ast.InterfaceType:
		if e.Methods == nil {
			return prefix + "any"
		}
		if len(e.Methods.List) == 0 {
			return prefix + "any"
		}
	}
	return "<unknown>"
}

// Snap.Init / snap.Edit carry snapshot literals — the canonical form is
// multi-line text. Double-quoted strings force \n escapes that destroy
// readability and turn whitespace edits into character-level diffs. Forcing a
// backticked raw string keeps the snapshot literal looking like the data it
// represents. Only flags the first arg when it is itself a string literal;
// variables and other expressions are unaffected (no type info here to track).
func check_snap_backtick(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	ast.Inspect(file, func(n ast.Node) (descend bool) {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector_expression, is_selector_expression := call.Fun.(*ast.SelectorExpr)
		if !is_selector_expression {
			return true
		}
		ident, is_ident := selector_expression.X.(*ast.Ident)
		if !is_ident {
			return true
		}
		if ident.Name != "snap" {
			return true
		}
		method := selector_expression.Sel.Name
		if method != "Init" {
			if method != "Edit" {
				return true
			}
		}
		if len(call.Args) == 0 {
			return true
		}
		lit, is_lit := call.Args[0].(*ast.BasicLit)
		if !is_lit {
			return true
		}
		if lit.Kind != token.STRING {
			return true
		}
		if strings.HasPrefix(lit.Value, "`") {
			return true
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(lit.Pos()),
			Message: fmt.Sprintf(
				"snap.%s must use a backticked raw string literal", method),
		})
		return true
	})
	return diags
}

// Tests are the executable specification of the system; a future reader hits
// them when they need to know what behavior is contractually promised. A bare
// Test_Foo with no doc forces them to reconstruct intent from the assertions.
// TestMain is exempt — it's a runner mandated by the testing package, not a
// behavioral test.
func check_test_documentation_comment(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	tok_file := file_set.File(file.Pos())
	if tok_file == nil {
		return nil
	}
	if !strings.HasSuffix(tok_file.Name(), "_test.go") {
		return nil
	}
	for _, declaration := range file.Decls {
		function_declaration, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function_declaration.Recv != nil {
			continue
		}
		if !strings.HasPrefix(function_declaration.Name.Name, "Test") {
			continue
		}
		if function_declaration.Name.Name == "TestMain" {
			continue
		}
		if function_declaration.Doc != nil {
			if len(function_declaration.Doc.List) > 0 {
				continue
			}
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(function_declaration.Pos()),
			Message: fmt.Sprintf(
				"test %s is missing a doc comment", function_declaration.Name.Name),
		})
	}
	return diags
}

// Every exported top-level identifier must carry a doc comment, the way
// Go's own conventions teach `go doc` to surface a meaningful summary for
// every importable name. Without the rule, an exported symbol is a
// promise to other modules with no human-readable contract attached.
//
// Scope: top-level FuncDecls, TypeSpecs, and ValueSpecs whose declared
// name is exported per ast.IsExported. Methods (FuncDecl with Recv) are
// included — every method that survived check_unnecessary_method
// satisfies a stdlib interface and so is part of the type's public
// shape. For grouped GenDecls, a doc on the containing block applies to
// every spec inside (matching the Go parser, which hangs a single
// leading comment on the GenDecl rather than the spec); a spec with its
// own Doc satisfies the rule independently.
//
// Exemptions: package main (exports nothing reachable from outside) and
// `_test` packages (the test-doc rule covers Test_ functions; remaining
// names in test packages are internal to the test binary).
func check_exported_documentation_comment(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	if file.Name.Name == "main" {
		return nil
	}
	if strings.HasSuffix(file.Name.Name, "_test") {
		return nil
	}
	for _, declaration := range file.Decls {
		switch d := declaration.(type) {
		case *ast.FuncDecl:
			diags = append(diags,
				check_exported_documentation_comment_function(file_set, d)...)
		case *ast.GenDecl:
			diags = append(diags,
				check_exported_documentation_comment_generic(file_set, d)...)
		}
	}
	return diags
}

func check_exported_documentation_comment_function(
	file_set *token.FileSet, function_declaration *ast.FuncDecl,
) (diags []Diagnostic) {

	if !ast.IsExported(function_declaration.Name.Name) {
		return nil
	}
	if check_exported_documentation_comment_has_documentation(function_declaration.Doc) {
		return nil
	}
	label := "func"
	if function_declaration.Recv != nil {
		label = "method"
	}
	return []Diagnostic{{
		Position: file_set.Position(function_declaration.Name.Pos()),
		Name:     "exported-doc",
		Want:     "doc comment on exported " + label,
		Message: fmt.Sprintf(
			"exported %s %s is missing a doc comment",
			label,
			function_declaration.Name.Name),
	}}
}

func check_exported_documentation_comment_generic(
	file_set *token.FileSet, generic_declaration *ast.GenDecl,
) (diags []Diagnostic) {

	switch generic_declaration.Tok {
	case token.TYPE, token.VAR, token.CONST:
	default:
		return nil
	}
	group_has_documentation := check_exported_documentation_comment_has_documentation(
		generic_declaration.Doc)
	kind := "var"
	switch generic_declaration.Tok {
	case token.TYPE:
		kind = "type"
	case token.CONST:
		kind = "const"
	}
	diags = check_exported_documentation_comment_generic_specs(
		file_set, generic_declaration, group_has_documentation, kind)
	return diags
}

func check_exported_documentation_comment_generic_specs(
	file_set *token.FileSet,
	generic_declaration *ast.GenDecl,
	group_has_documentation bool,
	kind string,
) (diags []Diagnostic) {
	for _, specification := range generic_declaration.Specs {
		switch s := specification.(type) {
		case *ast.TypeSpec:
			if !ast.IsExported(s.Name.Name) {
				continue
			}
			if group_has_documentation {
				continue
			}
			if check_exported_documentation_comment_has_documentation(s.Doc) {
				continue
			}
			diags = append(diags, Diagnostic{
				Position: file_set.Position(s.Name.Pos()),
				Name:     "exported-doc",
				Want:     "doc comment on exported type",
				Message: fmt.Sprintf(
					"exported type %s is missing a doc comment", s.Name.Name),
			})
		case *ast.ValueSpec:
			specification_has_documentation :=
				check_exported_documentation_comment_has_documentation(s.Doc)

			for _, name := range s.Names {
				if !ast.IsExported(name.Name) {
					continue
				}
				if group_has_documentation {
					continue
				}
				if specification_has_documentation {
					continue
				}
				diags = append(diags, Diagnostic{
					Position: file_set.Position(name.Pos()),
					Name:     "exported-doc",
					Want:     "doc comment on exported " + kind,
					Message: fmt.Sprintf("exported %s %s is missing a doc "+
						"comment", kind, name.Name),
				})
			}
		}
	}
	return diags
}

func check_exported_documentation_comment_has_documentation(group *ast.CommentGroup) (yes bool) {

	return comment_group_documents(group)
}

// True iff the comment group supplies prose documentation. CommentGroup.Text
// strips Go directives (//go:embed, //go:noinline, //line, …), so a group of
// directives alone yields the empty string and does not document a declaration.
func comment_group_documents(group *ast.CommentGroup) (yes bool) {

	if group == nil {
		return false
	}
	return strings.TrimSpace(group.Text()) != ""
}

// Enforces the index/count/offset/size naming convention from
// https://tigerbeetle.com/blog/2026-02-16-index-count-offset-size/, the
// no-abbreviations rule, and the nouns-over-present-participles rule.
// Each violation is emitted as its own diagnostic at the offending
// identifier's position so the output is parseable as standard
// file:line:column: message lines.
func check_names(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {

	var violations []name_violation
	violations = append(violations, check_names_terminology(file)...)
	violations = append(violations, check_names_arithmetic(file)...)
	violations = append(violations, check_names_participles(file)...)
	violations = append(violations, check_names_extremum(file)...)
	for _, v := range violations {
		diags = append(diags,
			Diagnostic{Position: file_set.Position(v.Position), Message: v.Message})
	}
	sort.Slice(diags, func(i, j int) (less bool) {
		if diags[i].Position.Line != diags[j].Position.Line {
			return diags[i].Position.Line < diags[j].Position.Line
		}
		return diags[i].Position.Column < diags[j].Position.Column
	})
	return diags
}

// One violation against a name-rule, attached to the source position of
// the offending construct (usually the declaring ident; for arithmetic
// invariants, the BinaryExpr itself).
type name_violation struct {
	Position token.Pos
	Message  string
}

// Stdlib allowlist: callee → required suffix. Curated; missing entries
// return "" rather than a guess-from-the-name, because stdlib's own naming
// is sometimes contrary to the blog's vocabulary (e.g., `strings.Index`
// returns a byte position which the blog calls `_offset`, not `_index`).
func check_names_terminology_attach_callee_term_stdlib_required(qualified string) (term string) {

	switch qualified {
	case "strings.Index", "strings.IndexByte", "strings.LastIndex",
		"bytes.Index", "bytes.IndexByte", "bytes.LastIndex":
		return "offset"
	case "strings.Count", "bytes.Count",
		"utf8.RuneCount", "utf8.RuneCountInString":
		return "count"
	case "binary.Size", "unsafe.Sizeof":
		return "size"
	}
	return ""
}

// Walks the file per top-level FuncDecl, collecting term requirements per
// declared ident from evidence-bearing AST nodes (C-style for-loop
// induction, len/cap and stdlib-allowlist call results, make-args), then
// emits one violation line per declared ident whose name lacks the
// required suffix. Returns lines in source order so the per-file group is
// stable across runs.
//
// Only evidence sources that are unambiguous without type information are
// handled. RangeStmt key and IndexExpr are deliberately omitted: both AST
// shapes cover slices/arrays AND maps interchangeably (`for k := range m`
// or `m[k]` look identical in the AST whether the receiver is a slice or
// a map). For slices the key/index slot is a position and would correctly
// take `_index`; for maps it's a lookup key, not a position, and `_index`
// is wrong. Without go/types we can't distinguish the two, so the rule
// would over-trigger on every map iteration and lookup. Leaving these
// unchecked is the conservative call — a soundness-over-coverage tradeoff.
func check_names_terminology(file *ast.File) (violations []name_violation) {

	for _, declaration := range file.Decls {
		function_declaration, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function_declaration.Body == nil {
			continue
		}
		declarations_map := check_names_terminology_function_declarations(
			function_declaration)
		requirements := map[*ast.Ident]map[string]bool{}
		require := func(identifier *ast.Ident, term string) {
			if identifier == nil {
				return
			}
			if requirements[identifier] == nil {
				requirements[identifier] = map[string]bool{}
			}
			requirements[identifier][term] = true
		}
		ast.Inspect(function_declaration.Body, func(n ast.Node) (descend bool) {
			check_names_terminology_attach(&check_names_terminology_attach_input{
				Node: n, Declarations: declarations_map, Require: require,
			})
			return true
		})
		violations = append(violations, check_names_terminology_emit(requirements)...)
	}
	return violations
}

// Builds a flat name → declaring ident map for a function. Includes params,
// named returns, and every var/const/`:=`/range/for-init declaration in
// the body. First declaration wins on collision — sufficient for the
// project's non-shadowing standard (see check_shadow).
func check_names_terminology_function_declarations(
	function *ast.FuncDecl,
) (declarations map[string]*ast.Ident) {

	declarations = map[string]*ast.Ident{}
	add := func(identifier *ast.Ident) {
		if identifier == nil {
			return
		}
		if identifier.Name == "_" {
			return
		}
		if declarations[identifier.Name] != nil {
			return
		}
		declarations[identifier.Name] = identifier
	}
	if function.Type.Params != nil {
		for _, f := range function.Type.Params.List {
			for _, identifier := range f.Names {
				add(identifier)
			}
		}
	}
	if function.Type.Results != nil {
		for _, f := range function.Type.Results.List {
			for _, identifier := range f.Names {
				add(identifier)
			}
		}
	}
	ast.Inspect(function.Body, func(n ast.Node) (descend bool) {
		switch x := n.(type) {
		case *ast.ValueSpec:
			for _, identifier := range x.Names {
				add(identifier)
			}
		case *ast.AssignStmt:
			if x.Tok != token.DEFINE {
				return true
			}
			for _, lhs := range x.Lhs {
				if identifier, is_ident := lhs.(*ast.Ident); is_ident {
					add(identifier)
				}
			}
		case *ast.RangeStmt:
			if x.Tok != token.DEFINE {
				return true
			}
			if identifier, is_ident := x.Key.(*ast.Ident); is_ident {
				add(identifier)
			}
			if identifier, is_ident := x.Value.(*ast.Ident); is_ident {
				add(identifier)
			}
		}
		return true
	})
	return declarations
}

type check_names_terminology_attach_input struct {
	Node         ast.Node
	Declarations map[string]*ast.Ident
	Require      func(identifier *ast.Ident, term string)
}

// Attaches term requirements to declaring idents based on the AST pattern
// at Node. Three evidence categories are handled in one body to keep the
// single-caller-callee chain flat:
//
//   - C-style for-loop induction (`for i := 0; i < N; i++`) → _index
//   - AssignStmt with RHS = call to len/cap (→ _count or _size) or to an
//     allowlisted stdlib symbol (→ per table) → suffix on LHS
//   - CallExpr to make(<sliceType>/<mapType>/<chanType>, n[, m]) → suffix
//     on n,m (byte element type → _size, otherwise _count).
func check_names_terminology_attach(input *check_names_terminology_attach_input) {

	switch x := input.Node.(type) {
	case *ast.ForStmt:
		ind := check_names_terminology_attach_induction_variable(x)
		if ind != nil {
			input.Require(ind, "index")
		}
	case *ast.AssignStmt:
		if x.Tok != token.DEFINE {
			return
		}
		if len(x.Lhs) != 1 {
			return
		}
		if len(x.Rhs) != 1 {
			return
		}
		lhs, is_ident := x.Lhs[0].(*ast.Ident)
		if !is_ident {
			return
		}
		if lhs.Name == "_" {
			return
		}
		call, is_call := x.Rhs[0].(*ast.CallExpr)
		if !is_call {
			return
		}
		for _, t := range check_names_terminology_attach_callee_term(call.Fun) {
			input.Require(lhs, t)
		}
	case *ast.CallExpr:
		check_names_terminology_attach_make(x, input.Declarations, input.Require)
	}
}

// Returns the induction ident of a strictly-shaped `for i := 0; i < N; i++`
// loop, or nil if the loop doesn't match. The shape match is intentionally
// strict — any deviation skips the attachment to avoid false positives on
// loops that use an int counter for non-index purposes (state machines,
// timeouts, etc.).
func check_names_terminology_attach_induction_variable(x *ast.ForStmt) (ind *ast.Ident) {

	if x.Init == nil {
		return nil
	}
	if x.Cond == nil {
		return nil
	}
	if x.Post == nil {
		return nil
	}
	init_statement, is_assign := x.Init.(*ast.AssignStmt)
	if !is_assign {
		return nil
	}
	if init_statement.Tok != token.DEFINE {
		return nil
	}
	if len(init_statement.Lhs) != 1 {
		return nil
	}
	if len(init_statement.Rhs) != 1 {
		return nil
	}
	candidate, is_ident := init_statement.Lhs[0].(*ast.Ident)
	if !is_ident {
		return nil
	}
	if candidate.Name == "_" {
		return nil
	}
	if _, is_lit := init_statement.Rhs[0].(*ast.BasicLit); !is_lit {
		return nil
	}
	condition, is_binary := x.Cond.(*ast.BinaryExpr)
	if !is_binary {
		return nil
	}
	if condition.Op != token.LSS {
		return nil
	}
	left, is_ident := condition.X.(*ast.Ident)
	if !is_ident {
		return nil
	}
	if left.Name != candidate.Name {
		return nil
	}
	post, is_increment_decrement := x.Post.(*ast.IncDecStmt)
	if !is_increment_decrement {
		return nil
	}
	post_identifier, is_ident := post.X.(*ast.Ident)
	if !is_ident {
		return nil
	}
	if post_identifier.Name != candidate.Name {
		return nil
	}
	return candidate
}

// Handles make(<type>, n[, m]) — attaches a count or size requirement to
// each bare-ident length/capacity argument. Byte-element slice → size;
// other slice, map, or chan → count. A make whose first arg is an opaque
// named type (alias for a slice/map/chan) is left alone because the kind
// can't be seen without types.
func check_names_terminology_attach_make(
	x *ast.CallExpr,
	decls map[string]*ast.Ident,
	require func(identifier *ast.Ident, term string),
) {

	callee, is_ident := x.Fun.(*ast.Ident)
	if !is_ident {
		return
	}
	if callee.Name != "make" {
		return
	}
	if len(x.Args) < 2 {
		return
	}
	term := ""
	switch t := x.Args[0].(type) {
	case *ast.ArrayType:
		term = "count"
		if elem, is_ident := t.Elt.(*ast.Ident); is_ident {
			if elem.Name == "byte" {
				term = "size"
			}
		}
	case *ast.MapType:
		term = "count"
	case *ast.ChanType:
		term = "count"
	}
	if term == "" {
		return
	}
	for _, argument := range x.Args[1:] {
		identifier, is_argument_identifier := argument.(*ast.Ident)
		if !is_argument_identifier {
			continue
		}
		if identifier.Name == "_" {
			continue
		}
		require(decls[identifier.Name], term)
	}
}

// Returns the required terminology terms for the given callee expression.
// Empty slice means no rule applies. Handles three patterns:
//
//	bare ident — len, cap (→ count, size dual-accept)
//	selector  — stdlib allowlist lookup (e.g., strings.Index → offset)
//	method tail — any `.Len()` / `.Cap()` receiver (→ size, per the blog's
//	  convention that container Len returns byte count)
//
// The dual-accept for len/cap reflects the lexical limit: without types
// we can't prove whether x is byte-element-typed in `v := len(x)`, so
// both _count and _size stay legal and the diagnostic reminds the user
// to pick _size when the count is in bytes.
func check_names_terminology_attach_callee_term(function_expression ast.Expr) (terms []string) {
	switch f := function_expression.(type) {
	case *ast.Ident:
		if f.Name == "len" {
			return []string{"count", "size"}
		}
		if f.Name == "cap" {
			return []string{"count", "size"}
		}
	case *ast.SelectorExpr:
		receiver, is_ident := f.X.(*ast.Ident)
		if is_ident {
			key := receiver.Name + "." + f.Sel.Name
			if t := check_names_terminology_attach_callee_term_stdlib_required(
				key); t != "" {
				return []string{t}
			}
		}
		if f.Sel.Name == "Len" {
			return []string{"size"}
		}
		if f.Sel.Name == "Cap" {
			return []string{"size"}
		}
	}
	return nil
}

// Walks the collected requirements and emits one violation line per
// declaring ident whose name lacks any acceptable suffix. Lines are
// sorted by declaring ident position for stable output.
func check_names_terminology_emit(
	requirements map[*ast.Ident]map[string]bool,
) (violations []name_violation) {
	type entry struct {
		Name     string
		Terms    []string
		Position token.Pos
	}
	entries := make([]entry, 0, len(requirements))
	for identifier, terms := range requirements {
		term_list := []string{}
		// Stable order: index, count, size, offset — reads naturally in
		// "count or size" for the len/cap dual-suffix case.
		for _, t := range []string{"index", "count", "size", "offset"} {
			if terms[t] {
				term_list = append(term_list, t)
			}
		}
		last := check_names_suffix_of(identifier.Name)
		matched := false
		for _, t := range term_list {
			if last == t {
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		entries = append(entries,
			entry{Name: identifier.Name, Terms: term_list, Position: identifier.Pos()})
	}
	sort.Slice(entries, func(i, j int) (less bool) {
		return entries[i].Position < entries[j].Position
	})
	for _, e := range entries {
		category := strings.Join(e.Terms, " or ")
		preferred := e.Terms[0]
		suggestion := check_names_terminology_emit_rename(
			&check_names_terminology_emit_rename_input{
				Name: e.Name, Term: preferred,
			})
		violations = append(violations, name_violation{
			Position: e.Position,
			Message: fmt.Sprintf(
				"naming convention: %s (used as %s) → rename to %s",
				e.Name,
				category,
				suggestion),
		})
	}
	return violations
}

type check_names_terminology_emit_rename_input struct {
	Name string
	Term string
}

// Returns a renamed identifier carrying the required terminology word. If
// the name already contains one of the terminology words (index, count,
// offset, size, length, len) as a segment, that segment is replaced
// in-place; otherwise the term is appended. Casing is preserved via the
// existing snake_case/Ada_Case detection in suggest.
func check_names_terminology_emit_rename(
	input *check_names_terminology_emit_rename_input,
) (output_string string) {

	style := "snake_case"
	if ada_case_re.MatchString(input.Name) {
		style = "Ada_Case"
	}
	words := suggest_split_words(input.Name)
	terminology := map[string]bool{
		"index": true, "count": true, "offset": true, "size": true,
		"length": true, "len": true,
	}
	replaced := false
	for i, w := range words {
		if !terminology[strings.ToLower(w)] {
			continue
		}
		words[i] = input.Term
		replaced = true
		break
	}
	if !replaced {
		words = append(words, input.Term)
	}
	return suggest(&suggest_input{Name: strings.Join(words, "_"), Want: style})
}

// Returns the lowercased trailing segment of name if it matches one of the
// four positive terminology words (index, count, offset, size); otherwise
// returns "". Tier-3 arithmetic uses this to recognize suffixed operands.
func check_names_suffix_of(name string) (suffix string) {

	words := suggest_split_words(name)
	if len(words) == 0 {
		return ""
	}
	last := strings.ToLower(words[len(words)-1])
	switch last {
	case "index", "count", "offset", "size":
		return last
	}
	return ""
}

// Tier 3: arithmetic-invariant check. Walks `*ast.BinaryExpr` ADD/SUB
// nodes. When both operands are bare idents that carry recognized
// suffixes, validates the combination per the result table; mismatched
// combinations are flagged ("_offset + _count is incoherent"). When the
// binary expression is the sole RHS of a `:=` assignment to a bare ident, the
// LHS's suffix is also validated against the result type.
//
// Conservative: only fires when *both* operands carry recognized
// suffixes. Mixed (one suffixed, one bare) is silently accepted —
// otherwise the rule would fire on every `len(x) + 1` style expression.
func check_names_arithmetic(file *ast.File) (violations []name_violation) {

	rhs_of := check_names_arithmetic_rhs_map(file)
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		binary_expression, is_binary_expression := n.(*ast.BinaryExpr)
		if !is_binary_expression {
			return true
		}
		if binary_expression.Op != token.ADD {
			if binary_expression.Op != token.SUB {
				return true
			}
		}
		violations = append(violations,
			check_names_arithmetic_check_binary(
				&check_names_arithmetic_check_binary_input{
					Binary_Expression: binary_expression,
					Lhs:               rhs_of[binary_expression],
				})...)
		return true
	})
	return violations
}

// Maps each BinaryExpr that is the sole RHS of a `:=` assignment to its
// LHS ident, so the arithmetic walker can validate the assignment target's
// suffix against the computed result type without needing parent context
// inside ast.Inspect.
func check_names_arithmetic_rhs_map(file *ast.File) (m map[*ast.BinaryExpr]*ast.Ident) {

	m = map[*ast.BinaryExpr]*ast.Ident{}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		assign, is_assign := n.(*ast.AssignStmt)
		if !is_assign {
			return true
		}
		if assign.Tok != token.DEFINE {
			return true
		}
		if len(assign.Lhs) != 1 {
			return true
		}
		if len(assign.Rhs) != 1 {
			return true
		}
		binary_expression, is_binary_expression := assign.Rhs[0].(*ast.BinaryExpr)
		if !is_binary_expression {
			return true
		}
		lhs, is_ident := assign.Lhs[0].(*ast.Ident)
		if !is_ident {
			return true
		}
		if lhs.Name == "_" {
			return true
		}
		m[binary_expression] = lhs
		return true
	})
	return m
}

type check_names_arithmetic_check_binary_input struct {
	Binary_Expression *ast.BinaryExpr
	Lhs               *ast.Ident
}

// Validates one BinaryExpr ADD/SUB site. Returns 0–2 violations:
// one for an incoherent operand combination, one for an LHS-suffix
// mismatch (the latter only if the operand combo is coherent and Lhs is
// non-nil). Skips silently when either operand is unsuffixed.
func check_names_arithmetic_check_binary(
	input *check_names_arithmetic_check_binary_input,
) (violations []name_violation) {

	left, is_ident := input.Binary_Expression.X.(*ast.Ident)
	if !is_ident {
		return nil
	}
	right, is_ident := input.Binary_Expression.Y.(*ast.Ident)
	if !is_ident {
		return nil
	}
	left_suffix := check_names_suffix_of(left.Name)
	if left_suffix == "" {
		return nil
	}
	right_suffix := check_names_suffix_of(right.Name)
	if right_suffix == "" {
		return nil
	}
	op_string := "+"
	if input.Binary_Expression.Op == token.SUB {
		op_string = "-"
	}
	result := check_names_arithmetic_check_binary_result(
		&check_names_arithmetic_check_binary_result_input{
			Left: left_suffix, Op: input.Binary_Expression.Op, Right: right_suffix,
		})
	if result == "" {
		violations = append(violations, name_violation{
			Position: input.Binary_Expression.Pos(),
			Message: fmt.Sprintf(
				"arithmetic: _%s %s _%s is incoherent",
				left_suffix,
				op_string,
				right_suffix),
		})
		return violations
	}
	if input.Lhs == nil {
		return nil
	}
	lhs_suffix := check_names_suffix_of(input.Lhs.Name)
	if lhs_suffix == result {
		return nil
	}
	violations = append(violations, name_violation{
		Position: input.Lhs.Pos(),
		Message: fmt.Sprintf(
			"arithmetic: %s = _%s %s _%s; must end in _%s",
			input.Lhs.Name, left_suffix, op_string, right_suffix, result,
		),
	})
	return violations
}

type check_names_arithmetic_check_binary_result_input struct {
	Left  string
	Op    token.Token
	Right string
}

// Returns the result-type suffix for a binary op on suffixed operands per
// the blog's invariant table. Empty result means the combination is
// incoherent (caller flags it). ADD is treated as commutative, so
// `_offset + _size` and `_size + _offset` both produce `_offset`.
func check_names_arithmetic_check_binary_result(
	input *check_names_arithmetic_check_binary_result_input,
) (result string) {

	if input.Op == token.SUB {
		if input.Left == input.Right {
			switch input.Left {
			case "index":
				return "count"
			case "offset":
				return "size"
			case "count":
				return "count"
			case "size":
				return "size"
			}
		}
		return ""
	}
	// ADD: normalize operand order so we only enumerate canonical pairs.
	a, b := input.Left, input.Right
	if a > b {
		a, b = b, a
	}
	if a == b {
		switch a {
		case "count":
			return "count"
		case "size":
			return "size"
		}
		return ""
	}
	if a == "count" {
		if b == "index" {
			return "index"
		}
	}
	if a == "offset" {
		if b == "size" {
			return "offset"
		}
	}
	return ""
}

// Looks word up in the per-word denylist decoded
// from lint.json's word_replacements (Configuration.Word_Replacements). Two
// kinds of entry share one lookup:
//
//   - Abbreviations carry one or more expansion candidates (id -> identifier).
//     Sourced from https://github.com/abbrcode/abbreviations-in-code (🟢+🔴
//     entries), hand-curated for genuinely ambiguous abbreviations. Entries Go
//     language/stdlib forces on every codebase are dropped: err, ctx, fmt, cap.
//   - Bans carry an empty (non-nil) candidate list because no mechanical rename
//     exists — the fix is to name the thing for what it is: util/utils/utility/
//     utilities (dumping-ground signals) and len/length (ambiguous across
//     languages — Rust = bytes, Python = code points; see
//     https://tigerbeetle.com/blog/2026-02-16-index-count-offset-size/). "helper"
//     is a third ban but only in function names, so it lives outside this
//     universal table — see check_names_vocabulary_function_ban.
//
// The nil-vs-empty distinction is load-bearing and survives the JSON round-trip:
// an absent key returns nil ("not in the table", skipped); a present key with an
// empty array returns a non-nil empty slice ("banned, no candidate") and yields
// the banned-substring diagnostic. Callers branch on len(candidates) vs nil.
//
// Lookups are by tokenized word from suggest_split_words, lowercased. Because
// the codebase enforces snake_case + PascalCase via check_casing, every word
// lands as its own token — no substring scan needed. Use sites are never
// visited, so the len and cap builtins stay legal. Single-letter loop counters
// (i/j/k/n/m per Tiger Style) are exempted by the check itself. Init is exempt —
// Go's package-initialization function is mandatorily named `init`.
func word_replacements_for(table map[string][]string, word string) (candidates []string) {
	candidates, present := table[word]
	if !present {
		return nil
	}
	// A present key whose JSON value is null decodes to a nil slice; normalize it
	// to the non-nil empty slice so the "present → banned" branch stays distinct
	// from "absent → not in the table".
	if candidates == nil {
		return []string{}
	}
	return candidates
}

// Words ending in "ing" that are unambiguously nouns. Any declared
// identifier whose final word (per suggest_split_words, lowercased)
// ends in "ing" and is NOT a key here is flagged as a present
// participle. The Stringer interface contract (`String() string`) is
// satisfied implicitly because "string" is in this set.
func is_allowed_ing_noun(word string) (allowed bool) {

	switch word {
	case "string", "ring", "thing", "king", "wing":
		return true
	case "sibling", "darling", "herring", "awning", "morning":
		return true
	case "evening", "ceiling", "lightning", "building", "dwelling":
		return true
	case "housing", "opening", "landing", "crossing", "parking":
		return true
	case "lining", "casing", "padding", "packaging", "clothing":
		return true
	case "bedding", "plumbing", "wiring", "tubing", "lighting":
		return true
	case "icing", "dressing", "stuffing", "frosting", "topping":
		return true
	case "filling", "coating", "seasoning", "helping", "serving":
		return true
	case "savings", "earnings", "holdings", "winnings", "belongings":
		return true
	case "offering", "meaning", "warning", "greeting", "blessing":
		return true
	case "heading", "ending", "beginning", "finding", "reading":
		return true
	case "saying", "feeling", "hearing", "meeting", "gathering":
		return true
	case "briefing", "screening", "sighting", "posting", "listing":
		return true
	case "mapping", "binding", "encoding", "setting", "grouping":
		return true
	case "ordering", "pairing", "spacing", "timing", "sizing":
		return true
	case "drawing", "painting", "carving", "etching", "engraving":
		return true
	case "recording":
		return true
	}
	return false
}

// Walks every declaration site in file and invokes visit for each
// declared identifier: top-level FuncDecl names (including methods),
// method receivers, params, named returns, TypeSpec names, struct
// field names, ValueSpec names (top-level and inside func bodies),
// AssignStmt LHS with DEFINE, and RangeStmt key/value with DEFINE.
// Blank "_" is skipped. Use-sites are never visited.
func check_names_walk_decls(file *ast.File, visit func(identifier *ast.Ident)) {

	emit := func(identifier *ast.Ident) {
		if identifier == nil {
			return
		}
		if identifier.Name == "_" {
			return
		}
		visit(identifier)
	}
	for _, declaration := range file.Decls {
		switch x := declaration.(type) {
		case *ast.FuncDecl:
			check_names_walk_declarations_function(x, emit)
		case *ast.GenDecl:
			check_names_walk_declarations_generic(x, emit)
		}
	}
}

func check_names_walk_declarations_function(
	function *ast.FuncDecl, emit func(identifier *ast.Ident)) {

	emit(function.Name)
	if function.Recv != nil {
		for _, f := range function.Recv.List {
			for _, identifier := range f.Names {
				emit(identifier)
			}
		}
	}
	if function.Type.Params != nil {
		for _, f := range function.Type.Params.List {
			for _, identifier := range f.Names {
				emit(identifier)
			}
		}
	}
	if function.Type.Results != nil {
		for _, f := range function.Type.Results.List {
			for _, identifier := range f.Names {
				emit(identifier)
			}
		}
	}
	if function.Body == nil {
		return
	}
	ast.Inspect(function.Body, func(n ast.Node) (descend bool) {
		check_names_walk_decls_body(n, emit)
		return true
	})
}

func check_names_walk_decls_body(n ast.Node, emit func(identifier *ast.Ident)) {
	switch y := n.(type) {
	case *ast.AssignStmt:
		if y.Tok != token.DEFINE {
			return
		}
		for _, lhs := range y.Lhs {
			if identifier, ok := lhs.(*ast.Ident); ok {
				emit(identifier)
			}
		}
	case *ast.RangeStmt:
		if y.Tok != token.DEFINE {
			return
		}
		if identifier, ok := y.Key.(*ast.Ident); ok {
			emit(identifier)
		}
		if identifier, ok := y.Value.(*ast.Ident); ok {
			emit(identifier)
		}
	case *ast.ValueSpec:
		for _, identifier := range y.Names {
			emit(identifier)
		}
	}
}

func check_names_walk_declarations_generic(gd *ast.GenDecl, emit func(identifier *ast.Ident)) {

	for _, specification := range gd.Specs {
		switch s := specification.(type) {
		case *ast.ValueSpec:
			for _, identifier := range s.Names {
				emit(identifier)
			}
		case *ast.TypeSpec:
			emit(s.Name)
			st, ok := s.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if st.Fields == nil {
				continue
			}
			for _, f := range st.Fields.List {
				for _, identifier := range f.Names {
					emit(identifier)
				}
			}
		}
	}
}

// Walks every declared identifier and flags any whose final tokenized
// word (lowercased) ends in "ing" and is not in is_allowed_ing_noun.
// The Stringer interface's String() method is implicitly allowed
// because "string" is in the noun allowlist.
func check_names_participles(file *ast.File) (violations []name_violation) {

	check_names_walk_decls(file, func(identifier *ast.Ident) {
		words := suggest_split_words(identifier.Name)
		if len(words) == 0 {
			return
		}
		last := strings.ToLower(words[len(words)-1])
		if !strings.HasSuffix(last, "ing") {
			return
		}
		if is_allowed_ing_noun(last) {
			return
		}
		violations = append(violations, name_violation{
			Position: identifier.Pos(),
			Message: fmt.Sprintf(
				"present participle %q → rename to a noun form", last),
		})
	})
	return violations
}

// Flags any declared identifier whose tokenized words include "max" or "min"
// anywhere but the final position. Extrema read as suffixes (line_max,
// retry_min); a leading or interior max/min is banned. Rides inside check_names
// beside the abbreviation and participle passes.
func check_names_extremum(file *ast.File) (violations []name_violation) {

	check_names_walk_decls(file, func(identifier *ast.Ident) {
		words := suggest_split_words(identifier.Name)
		style := "snake_case"
		if ada_case_re.MatchString(identifier.Name) {
			style = "Ada_Case"
		}
		for word_index, w := range words {
			lower := strings.ToLower(w)
			if lower != "max" {
				if lower != "min" {
					continue
				}
			}
			if word_index == len(words)-1 {
				continue
			}
			reordered := append([]string{}, words[:word_index]...)
			reordered = append(reordered, words[word_index+1:]...)
			reordered = append(reordered, w)
			violations = append(violations, name_violation{
				Position: identifier.Pos(),
				Message: fmt.Sprintf("rename %s -> %s", identifier.Name,
					suggest(&suggest_input{
						Name: strings.Join(reordered, "_"), Want: style})),
			})
		}
	})
	return violations
}

func check_comments_group_is_inline(
	file_set *token.FileSet, source []byte, c *ast.Comment,
) (inline bool) {

	position := file_set.Position(c.Slash)
	if position.Offset == 0 {
		return false
	}
	i := position.Offset - 1
	for i >= 0 {
		if source[i] == '\n' {
			return false
		}
		switch source[i] {
		case ' ', '\t':
			i--
			continue
		}
		return true
	}
	return false
}

// Stream-tier checks run against every file in the tree, regardless of
// extension. They cover invariants that an AST-only walker cannot reach:
// conflict markers, license shape, oversized files, dangling symlinks,
// markdown line length, SKILL.md size, and AGENTS.md ↔ CLAUDE.md drift.
//
// The whole tier runs ahead of the AST tier. If any stream check fires,
// every stream diagnostic is reported and the AST tier is suppressed —
// otherwise a conflict marker in a Go file surfaces as an opaque parse
// error instead of the actual problem.
type check_function_stream struct {
	Name  string
	Visit func(
		p string,
		info fs.FileInfo,
		load func() (data []byte, err error),
		output *[]Diagnostic)
	Finalize func(out *[]Diagnostic)
}

type check_file_system_stream_input struct {
	Fsys                  fs.FS
	Root                  string
	Root_Directory        string
	Tracked               map[string]bool
	Directory_Has_Tracked map[string]bool
	// Scan_Prefixes bounds the walk to the module subtrees a scoped run examines;
	// nil walks the whole tree. Every stream check positions its diagnostic at the
	// visited path, so an out-of-scope finding is dropped by the scope filter at
	// print time regardless — pruning the directory just skips the wasted reads.
	Scan_Prefixes []string
	Readlink      func(name string) (target string, err error)
}

func check_file_system_stream(
	input *check_file_system_stream_input,
) (diags []Diagnostic, go_paths []string, err error) {
	// Only the symlinks checker needs configuration — the tracked sets and the OS
	// Readlink seam; the rest are stateless visitors.
	checks := [stream_checker_count]check_function_stream{
		{Name: "conflict-markers", Visit: check_stream_conflict_markers},
		{Name: "github-actions-uses", Visit: check_stream_github_actions_uses},
		{Name: "banned-scripts", Visit: check_stream_banned_scripts},
		{Name: "banned-archives", Visit: check_stream_banned_archives},
		{Name: "agent-doc-max-lines", Visit: check_stream_agent_documentation_lines_max},
		check_file_system_stream_checks_stream_symlinks_checker(
			&check_file_system_stream_checks_stream_symlinks_checker_input{
				Root_Directory:        input.Root_Directory,
				Tracked:               input.Tracked,
				Directory_Has_Tracked: input.Directory_Has_Tracked,
				Readlink:              input.Readlink,
			}),
		{Name: "markdown-line-length", Visit: check_stream_markdown_line_max},
		{Name: "trailing-whitespace", Visit: check_stream_markdown_trailing_whitespace},
		check_file_system_stream_checks_stream_agents_claude_pair_checker(),
	}
	per_check := make([][]Diagnostic, len(checks))
	err = fs.WalkDir(input.Fsys, input.Root,
		func(p string, d fs.DirEntry, walk_err error) (output error) {
			return check_file_system_stream_walk(
				p, d, walk_err, input, checks, per_check, &go_paths)
		})
	if err != nil {
		return nil, nil, err
	}
	for i, c := range checks {
		if c.Finalize != nil {
			c.Finalize(&per_check[i])
		}
	}
	for i, c := range checks {
		for _, d := range per_check[i] {
			d.Message = fmt.Sprintf("%s: %s", c.Name, d.Message)
			diags = append(diags, d)
		}
	}
	return diags, go_paths, nil
}

// Handles one fs.WalkDir visit for check_file_system_stream: applies the
// directory-skip rules, records .go paths, and runs every stream checker over
// tracked files (each checker reads source lazily through the shared loader).
func check_file_system_stream_walk(
	p string, d fs.DirEntry, walk_err error,
	input *check_file_system_stream_input,
	checks [stream_checker_count]check_function_stream,
	per_check [][]Diagnostic, go_paths *[]string,
) (output error) {
	if walk_err != nil {
		return walk_err
	}
	if d.IsDir() {
		if p != input.Root {
			if Ignored_Directory(p) {
				return fs.SkipDir
			}
			if !scan_prefixes_reach(input.Scan_Prefixes, p) {
				return fs.SkipDir
			}
		}
		if input.Directory_Has_Tracked != nil {
			if p != input.Root {
				if !input.Directory_Has_Tracked[p] {
					return fs.SkipDir
				}
			}
		}
		return nil
	}
	if input.Tracked != nil {
		if !input.Tracked[p] {
			return nil
		}
	}
	if path.Ext(p) == ".go" {
		*go_paths = append(*go_paths, p)
	}
	information, information_error := d.Info()
	if information_error != nil {
		return information_error
	}
	var (
		source   []byte
		read_err error
		loaded   bool
	)
	load := func() (data []byte, err error) {
		if !loaded {
			source, read_err = fs.ReadFile(input.Fsys, p)
			loaded = true
		}
		return source, read_err
	}
	for i, c := range checks {
		c.Visit(p, information, load, &per_check[i])
	}
	return nil
}

// Ignored_Directory reports whether a directory a walker reaches should be
// pruned outright — the one global ignore list, shared by every tier and by
// main's tracked-file walk so the rule cannot drift between them. The argument
// is the slash path from the scan root, so the top-level third_party/ drop-zone
// is matched exactly (a nested pkg/third_party/ is first-party code and stays
// linted) while vendor, .git, and .jj match at any depth: Go's vendor/ nests by
// convention, and tool-state dirs surface inside worktrees and submodules.
// Both spellings of the drop-zone are pruned: the directory on disk is named
// third-party/ (hyphen), so matching only the underscore left its 70k-file
// vendored corpus to be walked on every run and trimmed only later, by the
// ignore list, after the directory syscalls were already spent.
// Gitignored paths are not handled here; main prunes them via the Tracked set.
func Ignored_Directory(relative string) (ignored bool) {

	if relative == "third_party" {
		return true
	}
	if relative == "third-party" {
		return true
	}
	base := relative[strings.LastIndexByte(relative, '/')+1:]
	return base == "vendor" || base == ".git" || base == ".jj"
}

func check_stream_conflict_markers(
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
	output *[]Diagnostic) {

	// Line starts emitted by every common VCS merge tool. The size gate skips
	// minified bundles and binary blobs where the scan would dominate runtime
	// and yield moimportsy false positives.
	conflict_marker_prefixes := [][]byte{
		[]byte("<<<<<<<"),
		[]byte(">>>>>>>"),
		[]byte("%%%%%%%"),
		[]byte("+++++++"),
		[]byte("\\\\\\\\\\\\\\"),
	}
	if information.Size() > 1<<20 {
		return
	}
	source, err := load()
	if err != nil {
		return
	}
	line_number := 1
	for i := 0; i < len(source); {
		for _, m := range conflict_marker_prefixes {
			if bytes.HasPrefix(source[i:], m) {
				*output = append(*output, Diagnostic{
					Position: token.Position{
						Filename: p,
						Line:     line_number,
						Column:   1,
					},
					Message: "resolve the conflict and remove the marker",
				})
				break
			}
		}
		newline_offset := bytes.IndexByte(source[i:], '\n')
		if newline_offset < 0 {
			break
		}
		i += newline_offset + 1
		line_number++
	}
}

// Scripting-language files dilute the Go-first stance of this repo and add
// hidden, untyped build/runtime surface. The top-level third_party/ drop-zone
// and every vendor/ tree are pruned upstream by Ignored_Directory, so they
// never reach here; a nested third_party/ (e.g. pkg/third_party/x.py) is
// first-party code and IS flagged.
func check_stream_banned_scripts(
	p string,
	information fs.FileInfo,
	_ func() (data []byte, err error),
	output *[]Diagnostic) {

	base := strings.ToLower(information.Name())
	extension := strings.ToLower(path.Ext(base))
	banned := false
	switch extension {
	case ".py", ".sh", ".bash", ".zsh", ".fish", ".ksh", ".csh", ".pl", ".pm", ".rb", ".lua",
		".tcl", ".awk", ".ps1", ".psm1", ".bat", ".cmd", ".vbs", ".groovy", ".r",
		".jl":
		banned = true
	}
	switch base {
	case "makefile", "gnumakefile", "rakefile", "gemfile",
		"pipfile", "justfile", "taskfile":
		banned = true
	}
	if !banned {
		return
	}
	*output = append(*output, Diagnostic{
		Position: token.Position{Filename: p, Line: 1, Column: 1},
		Message:  fmt.Sprintf("rewrite %q as a go script", p),
	})
}

// An .xz file forces decompression the Go stdlib cannot do: there is no
// compress/xz, so reading one shells out to external tar, whose GNU and BSD
// builds diverge in flags and behavior. gzip is read directly by compress/gzip
// and archive/tar, so it is the portable choice. The top-level third_party/
// drop-zone and every vendor/ tree are pruned upstream by Ignored_Directory.
func check_stream_banned_archives(
	p string,
	information fs.FileInfo,
	_ func() (data []byte, err error),
	output *[]Diagnostic) {

	if strings.ToLower(path.Ext(information.Name())) != ".xz" {
		return
	}
	*output = append(*output, Diagnostic{
		Position: token.Position{Filename: p, Line: 1, Column: 1},
		Message:  ".xz files are banned; use .gz or .zip instead",
	})
}

// Flags any `uses:` line in a GitHub Actions workflow. Third-party actions
// are an unaudited supply-chain surface: every `uses: owner/repo@ref` pins
// remote code that runs in CI with repo credentials. The rule is absolute —
// rewrite the step with `run:` and inline shell instead.
func check_stream_github_actions_uses(
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
	output *[]Diagnostic) {

	if !strings.HasPrefix(p, ".github/workflows/") {
		return
	}
	extension := strings.ToLower(path.Ext(p))
	if extension != ".yml" {
		if extension != ".yaml" {
			return
		}
	}
	source, err := load()
	if err != nil {
		return
	}
	line_number := 1
	line_start := 0
	for i := 0; i <= len(source); i++ {
		if i < len(source) {
			if source[i] != '\n' {
				continue
			}
		}
		line := source[line_start:i]
		trimmed := bytes.TrimLeft(line, " \t")
		// YAML list items prefix the first key with "- ", e.g.
		// `  - uses: actions/checkout@v4`. Strip an optional leading
		// dash+space so both list-head and aligned-key forms match.
		if bytes.HasPrefix(trimmed, []byte("- ")) {
			trimmed = bytes.TrimLeft(trimmed[2:], " \t")
		}
		if bytes.HasPrefix(trimmed, []byte("uses:")) {
			*output = append(*output, Diagnostic{
				Position: token.Position{Filename: p, Line: line_number, Column: 1},
				Message: "third-party github action banned; replace `uses:` with " +
					"an inline `run:` step",
			})
		}
		line_number++
		line_start = i + 1
	}
}

func check_stream_agent_documentation_lines_max(
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
	output *[]Diagnostic,

) {

	switch information.Name() {
	case "CLAUDE.md", "AGENTS.md", "SKILL.md":
	default:
		return
	}
	source, err := load()
	if err != nil {
		return
	}
	lines_count := bytes.Count(source, []byte{'\n'})
	if len(source) > 0 {
		if source[len(source)-1] != '\n' {
			lines_count++
		}
	}
	if lines_count > agent_documentation_lines_max {
		message := fmt.Sprintf(
			"%s has %d lines; split or trim it under %d",
			information.Name(), lines_count, agent_documentation_lines_max,
		)
		// A skill that outgrows the budget should shed prose, not just shrink:
		// the cure is to move steps into a script the agent runs, so SKILL.md
		// earns its own directive rather than the generic split-or-trim line.
		if information.Name() == "SKILL.md" {
			message = fmt.Sprintf(
				"SKILL.md is capped at %d lines. "+
					"Prefer procedural scripting over prose.",
				agent_documentation_lines_max,
			)
		}
		*output = append(*output, Diagnostic{
			Position: token.Position{Filename: p, Line: 1, Column: 1},
			Message:  message,
		})
	}
}

// Enforces snake_case, Ada_Case, or SCREAMING_SNAKE_CASE on every directory
// name and every file stem. The ada_case_re alternation already accepts
// SCREAMING segments (`[A-Z][A-Z0-9]*`), so two regexes cover all three styles.
//
// Exemptions:
//   - Hidden entries (segment begins with `.`) — these are tool/OS
//     conventions outside our naming policy.
//   - Path components under top-level `third_party/` — vendored code keeps
//     its upstream naming.
//
// Each segment is reported at most once across the run; the file walk
// visits a dir's name once on entry, then re-visits it as a prefix of
// every contained file, which would otherwise produce N duplicate diags.
// Enforces the path-casing rule absolutely: every path segment, split on its
// dots, is snake_case, Ada_Case, or SCREAMING_SNAKE_CASE. The only exemptions
// are the .git directory and gitignored paths — both already encoded in tracked
// (git ls-files --exclude-standard). When tracked is nil (git unavailable) it
// walks the whole tree, the .git directory aside, so the rule still binds.
func check_path_casing(
	fsys fs.FS, tracked map[string]bool,
) (diags []Diagnostic) {

	seen := map[string]bool{}
	for _, p := range check_path_casing_paths(fsys, tracked) {
		if p == ".git" {
			continue
		}
		if strings.HasPrefix(p, ".git/") {
			continue
		}
		segments := strings.Split(p, "/")
		if path_casing_vendored(segments) {
			continue
		}
		for i, seg := range segments {
			key := strings.Join(segments[:i+1], "/")
			if seen[key] {
				continue
			}
			seen[key] = true
			if path_casing_segment_ok(seg) {
				continue
			}
			suggestion := path_casing_suggest(seg)
			diags = append(diags, Diagnostic{
				Position: token.Position{Filename: key},
				Message:  fmt.Sprintf("rename %s -> %s", seg, suggestion),
			})
		}
	}
	return diags
}

// Reports whether the path lies in a vendored tree — any third_party or vendor
// directory — whose file names upstream chose and the workspace does not own.
func path_casing_vendored(segments []string) (yes bool) {

	for _, seg := range segments {
		switch seg {
		case "third_party", "vendor":
			return true
		}
	}
	return false
}

// Judges a path segment by its dot-separated components independently: the FQDN
// naming scheme dots apart cased components (TIGER_STYLE.Index_Count.md), and
// every one must hold — the extension included, which is lowercase and so
// trivially snake_case.
func path_casing_segment_ok(seg string) (ok bool) {

	for _, component := range strings.Split(seg, ".") {
		if component == "" {
			// A leading, trailing, or doubled dot — a dotfile's leading dot,
			// say — yields an empty component with nothing to case.
			continue
		}
		if snake_case_re.MatchString(component) {
			continue
		}
		if ada_case_re.MatchString(component) {
			continue
		}
		return false
	}
	return true
}

// Produces the corrected form of a path segment by splitting on dots, applying
// the appropriate casing style to each component (Ada_Case when the component
// starts with an uppercase letter, snake_case otherwise), and rejoining.
// Hyphens are converted to underscores before the style pass so that kebab
// components are handled correctly by suggest's word splitter.
func path_casing_suggest(seg string) (output string) {
	components := strings.Split(seg, ".")
	for i, c := range components {
		if c == "" {
			continue
		}
		style := "snake_case"
		if unicode.IsUpper(rune(c[0])) {
			style = "Ada_Case"
		}
		components[i] = suggest(&suggest_input{
			Name: strings.ReplaceAll(c, "-", "_"),
			Want: style,
		})
	}
	return strings.Join(components, ".")
}

// Returns the sorted paths the rule must check. With a tracked set it is exactly
// that set (gitignore already applied); with none it walks the tree, pruning
// .git so git's internals never enter the check.
func check_path_casing_paths(fsys fs.FS, tracked map[string]bool) (paths []string) {

	if tracked != nil {
		paths = make([]string, 0, len(tracked))
		for p := range tracked {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		return paths
	}
	walk_err := fs.WalkDir(fsys, ".",
		func(p string, d fs.DirEntry, entry_err error) (output error) {
			if entry_err != nil {
				return nil
			}
			if d.IsDir() {
				if Ignored_Directory(p) {
					return fs.SkipDir
				}
				return nil
			}
			paths = append(paths, p)
			return nil
		})
	if walk_err != nil {
		return nil
	}
	sort.Strings(paths)
	return paths
}

// A glob_pattern is a lint.json ignore entry parsed into the three facts the
// matcher needs: Core is the gitignore pattern reduced to a form
// glob_match can run against a full prefix (an unanchored, slash-less entry is
// rewritten with a leading **/ so "weird.md" matches at any depth); Anchored
// records whether the original entry was tied to the workspace root (it held a
// slash) rather than floating; Directory_Only records a trailing slash, which in
// gitignore binds the entry to directories.
type glob_pattern struct {
	Core           string
	Anchored       bool
	Directory_Only bool
}

// Reduces a raw lint.json entry to a glob_pattern. A trailing slash is
// gitignore's directory marker; a leading or interior slash anchors the entry to
// the root; a slash-less entry floats, which we model as **/ + entry so the same
// prefix matcher serves both. Assumes the entry already passed
// validate_glob_patterns, so it cannot be empty or negated.
func parse_glob_pattern(raw string) (parsed glob_pattern) {
	parsed.Directory_Only = strings.HasSuffix(raw, "/")
	trimmed := strings.TrimSuffix(raw, "/")
	had_leading_slash := strings.HasPrefix(trimmed, "/")
	trimmed = strings.TrimPrefix(trimmed, "/")
	parsed.Anchored = had_leading_slash || strings.Contains(trimmed, "/")
	parsed.Core = trimmed
	if !parsed.Anchored {
		parsed.Core = "**/" + trimmed
	}
	return parsed
}

// Reports whether key — or any of its ancestor directories — matches one of the
// gitignore-style patterns. gitignore excludes a path when the path or any
// ancestor directory matches, so we test every prefix of key; each ancestor is a
// directory, and the leaf's directory status is key_is_directory. A
// Directory_Only entry is skipped at prefixes that are files, which is what makes
// a trailing-slash entry bind to directories and their subtree but not to a
// same-named file.
func glob_patterns_match(
	key string, key_is_directory bool, patterns []glob_pattern,
) (found bool) {

	if len(patterns) == 0 {
		return false
	}
	segments := strings.Split(key, "/")
	for i := range segments {
		prefix := strings.Join(segments[:i+1], "/")
		prefix_is_directory := i < len(segments)-1 || key_is_directory
		for _, p := range patterns {
			// A trailing-slash entry binds to directories, so it must skip a
			// prefix that is a file (gitignore's directory-only semantics).
			if p.Directory_Only {
				if !prefix_is_directory {
					continue
				}
			}
			// The entry passed parse-time validation, so glob_match cannot return
			// ErrBadPattern here; a non-match is the only other outcome.
			matched, _ := glob_match(&glob_match_input{Pattern: p.Core, Path: prefix})
			if matched {
				return true
			}
		}
	}
	return false
}

type glob_match_input struct {
	Pattern string
	Path    string
}

// Reports whether Path matches the doublestar Pattern. A ** segment matches zero
// or more whole path segments; every other segment is matched against the
// corresponding path segment by path.Match, so *, ?, and [...] keep their
// single-segment meaning (none crosses a slash) and a malformed segment surfaces
// as path.Match's ErrBadPattern.
func glob_match(input *glob_match_input) (matched bool, err error) {
	return glob_match_segments(&glob_match_segments_input{
		Pattern: strings.Split(input.Pattern, "/"),
		Name:    strings.Split(input.Path, "/"),
	})
}

type glob_match_segments_input struct {
	Pattern []string
	Name    []string
}

// Matches the Pattern segments against the Name segments with a two-pointer scan
// that backtracks across **, the segment-level analogue of wildcard matching. A
// ** is remembered as a resume point and first tried as matching zero segments;
// on a later mismatch the scan returns to it and lets the ** swallow one more
// name segment, which is how a single ** spans an unknown depth. Trailing **s
// match the empty remainder, which is why dir/** also matches dir itself.
func glob_match_segments(input *glob_match_segments_input) (matched bool, err error) {
	pattern := input.Pattern
	name := input.Name
	pattern_index := 0
	name_index := 0
	// The resume index sits just after the most recent **; -1 means no ** is
	// available to backtrack to. star_name_index records how much of name that **
	// has been charged with so far.
	star_pattern_index := -1
	star_name_index := 0
	for name_index < len(name) {
		if pattern_index < len(pattern) {
			if pattern[pattern_index] == "**" {
				star_pattern_index = pattern_index + 1
				star_name_index = name_index
				pattern_index++
				continue
			}
			ok, match_err := path.Match(pattern[pattern_index], name[name_index])
			if match_err != nil {
				return false, match_err
			}
			if ok {
				pattern_index++
				name_index++
				continue
			}
		}
		// No literal segment matched here, so the only way forward is to charge
		// the last ** with one more name segment; absent a **, the match fails.
		if star_pattern_index < 0 {
			return false, nil
		}
		pattern_index = star_pattern_index
		star_name_index++
		name_index = star_name_index
	}
	// Name is exhausted; the match holds only if every leftover pattern segment
	// is a ** standing for the empty remainder.
	for pattern_index < len(pattern) {
		if pattern[pattern_index] != "**" {
			return false, nil
		}
		pattern_index++
	}
	return true, nil
}

type check_file_system_stream_checks_stream_symlinks_checker_input struct {
	Root_Directory        string
	Tracked               map[string]bool
	Directory_Has_Tracked map[string]bool
	Readlink              func(name string) (target string, err error)
}

// Reports a tracked symlink that does not resolve to a tracked target. The walk
// only visits tracked files, so every symlink reaching here is already tracked —
// the rule applies to vcs-tracked symlinks alone. Membership, not on-disk
// existence, is the test: a target outside the tracked set — missing, gitignored,
// vendored, or escaping the repo — is banned even when it exists, because no
// fresh checkout would carry it; a directory target is allowed when it holds
// tracked files. Readlink is injected (main.go binds os.Readlink) because fs.FS
// has no symlink primitive. A nil Tracked set (the non-git fallback), an empty
// Root_Directory, or a nil Readlink self-disables the check.
func check_file_system_stream_checks_stream_symlinks_checker(
	input *check_file_system_stream_checks_stream_symlinks_checker_input,
) (c check_function_stream) {
	root_directory := input.Root_Directory
	tracked := input.Tracked
	directory_has_tracked := input.Directory_Has_Tracked
	readlink := input.Readlink
	return check_function_stream{
		Name: "symlink",
		Visit: func(
			p string,
			info fs.FileInfo,
			_ func() (data []byte, err error),
			output *[]Diagnostic) {
			if root_directory == "" {
				return
			}
			if readlink == nil {
				return
			}
			if tracked == nil {
				return
			}
			if info.Mode()&fs.ModeSymlink == 0 {
				return
			}
			operating_system_path := filepath.Join(root_directory, p)
			target, read_err := readlink(operating_system_path)
			if read_err != nil {
				*output = append(*output, Diagnostic{
					Position: token.Position{Filename: p},
					Message:  "unreadable symlink target",
				})
				return
			}
			resolved := target
			if !filepath.IsAbs(target) {
				resolved = filepath.Join(
					filepath.Dir(operating_system_path), target)
			}
			relative, relative_err := filepath.Rel(root_directory, resolved)
			target_path := filepath.ToSlash(relative)
			// A relative_err means resolved can't be expressed under the root (a
			// different volume) — outside the tracked tree, so it falls through to
			// the untracked diagnostic. Self-reference is ruled out first: the
			// symlink's own path is tracked, so a link to itself would otherwise
			// pass the membership test below.
			if relative_err == nil {
				if target_path == p {
					*output = append(*output, Diagnostic{
						Position: token.Position{Filename: p},
						Message:  "symlink targets itself",
					})
					return
				}
				if tracked[target_path] {
					return
				}
				if directory_has_tracked[target_path] {
					return
				}
			}
			*output = append(*output, Diagnostic{
				Position: token.Position{Filename: p},
				Message:  fmt.Sprintf("untracked symlink target -> %s", target),
			})
		},
	}
}

// Enforces a 100-rune visual cap on every .md line. Outside fenced code, two
// rendering exemptions apply: table rows (pipes can't break across lines) and
// lines whose width is dominated by a URL (`://`). Inside a fenced block both
// rationales evaporate — the line is literal code — so no exemption applies and
// every line is held to the cap.
// Display width approximates what a fixed-width terminal renders, the property
// "visual limit" always meant but rune count only approximated: a tab advances
// to the next eight-column stop, an East Asian wide or fullwidth rune occupies
// two columns, a nonspacing or enclosing mark occupies none, every other rune
// one. The wide ranges are a stdlib-only stand-in for the exhaustive East Asian
// Width property (golang.org/x/text/width), which this package forgoes to stay
// dependency-free.
func display_width(text string) (width int) {
	for _, glyph := range text {
		switch {
		case glyph == '\t':
			width += 8 - width%8
		case unicode.Is(unicode.Mn, glyph), unicode.Is(unicode.Me, glyph):
		case display_glyph_wide(glyph):
			width += 2
		default:
			width++
		}
	}
	return width
}

func display_glyph_wide(glyph rune) (wide bool) {
	spans := [...][2]rune{
		{0x1100, 0x115F},   // Hangul Jamo
		{0x2E80, 0x303E},   // CJK radicals, Kangxi, CJK symbols
		{0x3041, 0x33FF},   // Hiragana through CJK compatibility
		{0x3400, 0x4DBF},   // CJK extension A
		{0x4E00, 0x9FFF},   // CJK unified ideographs
		{0xA000, 0xA4CF},   // Yi
		{0xAC00, 0xD7A3},   // Hangul syllables
		{0xF900, 0xFAFF},   // CJK compatibility ideographs
		{0xFE30, 0xFE4F},   // CJK compatibility forms
		{0xFF00, 0xFF60},   // Fullwidth forms
		{0xFFE0, 0xFFE6},   // Fullwidth signs
		{0x1F300, 0x1FAFF}, // Emoji and pictographs
		{0x20000, 0x3FFFD}, // CJK extension B and beyond
	}
	for _, span := range spans {
		if glyph < span[0] {
			continue
		}
		if glyph > span[1] {
			continue
		}
		return true
	}
	return false
}

func check_stream_markdown_line_max(
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
	output *[]Diagnostic) {

	if !strings.HasSuffix(information.Name(), ".md") {
		return
	}
	source, err := load()
	if err != nil {
		return
	}
	input_code := false
	line_number := 0
	for i := 0; i < len(source); {
		line_number++
		newline_offset := bytes.IndexByte(source[i:], '\n')
		var line []byte
		if newline_offset < 0 {
			line = source[i:]
			i = len(source)
		} else {
			line = source[i : i+newline_offset]
			i += newline_offset + 1
		}
		if bytes.HasPrefix(line, []byte("```")) {
			input_code = !input_code
			continue
		}
		trimmed := bytes.TrimSpace(line)
		is_table_row := bytes.HasPrefix(
			trimmed, []byte("|")) && bytes.HasSuffix(trimmed, []byte("|"))
		// The table-row and URL exemptions are markdown-rendering allowances: a
		// real table row cannot wrap and a prose URL has no fold point. Inside a
		// fenced block the line is literal code, where neither rationale holds, so
		// the exemptions are gated off and every code line is held to the cap.
		if !input_code {
			if is_table_row {
				continue
			}
			if bytes.Contains(line, []byte("://")) {
				continue
			}
		}
		columns := display_width(string(line))
		if columns > markdown_line_max {
			*output = append(*output, Diagnostic{
				Position: token.Position{Filename: p, Line: line_number, Column: 1},
				Message: fmt.Sprintf(
					"markdown line is %d columns; visual limit is %d",
					columns, markdown_line_max),
			})
		}
	}
}

func check_stream_markdown_trailing_whitespace(
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
	output *[]Diagnostic) {

	if !strings.HasSuffix(information.Name(), ".md") {
		return
	}
	source, err := load()
	if err != nil {
		return
	}
	input_code := false
	line_number := 0
	for i := 0; i < len(source); {
		line_number++
		newline_offset := bytes.IndexByte(source[i:], '\n')
		var line []byte
		if newline_offset < 0 {
			line = source[i:]
			i = len(source)
		} else {
			line = source[i : i+newline_offset]
			i += newline_offset + 1
		}
		if bytes.HasPrefix(line, []byte("```")) {
			input_code = !input_code
			continue
		}
		// Code may rely on trailing whitespace; only prose is held to the rule.
		if input_code {
			continue
		}
		if len(line) == len(bytes.TrimRight(line, " \t")) {
			continue
		}
		*output = append(*output, Diagnostic{
			Position: token.Position{Filename: p, Line: line_number, Column: 1},
			Message:  "markdown line has trailing whitespace",
		})
	}
}

// Enforces that every directory containing AGENTS.md or CLAUDE.md contains
// both, byte-identical. The pair is one source of truth split across two
// filenames so any agent harness reading either sees the same instructions;
// drift is the failure mode.
//
// Scope: root + one level deep. Anything deeper is per-package context that
// doesn't need a paired sibling.
func check_file_system_stream_checks_stream_agents_claude_pair_checker() (c check_function_stream) {

	pairs := map[string]*agents_claude_pair{}
	return check_function_stream{
		Name: agents_pair_check_name,
		Visit: func(
			p string,
			info fs.FileInfo,
			load func() (data []byte, err error),
			_ *[]Diagnostic) {
			agents_claude_pair_visit(pairs, p, info, load)
		},
		Finalize: func(output *[]Diagnostic) {
			agents_claude_pair_finalize(pairs, output)
		},
	}
}

type agents_claude_pair struct {
	Agents     []byte
	Claude     []byte
	Has_Agents bool
	Has_Claude bool
}

func agents_claude_pair_visit(
	pairs map[string]*agents_claude_pair,
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
) {
	name := information.Name()
	if name != "AGENTS.md" {
		if name != "CLAUDE.md" {
			return
		}
	}
	if strings.Count(p, "/") > 1 {
		return
	}
	source, err := load()
	if err != nil {
		return
	}
	directory := path.Dir(p)
	pp, ok := pairs[directory]
	if !ok {
		pp = &agents_claude_pair{}
		pairs[directory] = pp
	}
	if name == "AGENTS.md" {
		pp.Has_Agents = true
		pp.Agents = append(pp.Agents[:0], source...)
	} else {
		pp.Has_Claude = true
		pp.Claude = append(pp.Claude[:0], source...)
	}
}

func agents_claude_pair_finalize(
	pairs map[string]*agents_claude_pair, output *[]Diagnostic,
) {
	dirs := make([]string, 0, len(pairs))
	for d := range pairs {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, d := range dirs {
		pp := pairs[d]
		switch {
		case !pp.Has_Agents:
			*output = append(*output, Diagnostic{
				Position: token.Position{Filename: d},
				Message: "AGENTS.md is missing; it must mirror " +
					"CLAUDE.md byte-for-byte",
			})
		case !pp.Has_Claude:
			*output = append(*output, Diagnostic{
				Position: token.Position{Filename: d},
				Message: "CLAUDE.md is missing; it must mirror " +
					"AGENTS.md byte-for-byte",
			})
		case !bytes.Equal(pp.Agents, pp.Claude):
			*output = append(*output, Diagnostic{
				Position: token.Position{Filename: d},
				Message: "AGENTS.md and CLAUDE.md differ; make " +
					"them byte-identical",
			})
		}
	}
}

// Library packages that touch impure process state (env vars, the wall
// clock, the default HTTP transport, /dev/urandom, …) cannot be substituted
// for in tests or rewired by callers. Force every such read to happen in
// package main, where the program is allowed to bind to the real world,
// and have libraries receive the dependency as a parameter instead.
//
// Exemptions: package main (composition root), _test.go files (tests
// legitimately call time.Now, t.TempDir, etc.), and composition-tier
// packages (one logical depth below the library tier in the same
// module). The composition tier is where a library is permitted to
// wire its default to the real world — that binding has to live
// somewhere, and the doctrine reserves exactly this position for it.

func check_no_impure_stdlib(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {

	for _, pf := range parsed_files {
		if pf.File.Name.Name == "main" {
			continue
		}
		if strings.HasSuffix(pf.Path, "_test.go") {
			continue
		}
		if parsed_file_is_composition_tier(pf, modules) {
			continue
		}
		diags = append(diags, check_no_impure_stdlib_per_file(pf.File_Set, pf.File)...)
	}
	return diags
}

// True iff the file sits exactly one non-main Go ancestor below the
// library tier in its module. Mirrors check_library_tier_depth's
// counting logic but inverts the threshold: tier-depth fires when
// count > 1, the composition-tier exemption fires when count == 1.
func parsed_file_is_composition_tier(pf parsed_file, modules *module_index) (yes bool) {
	module_index_number := modules.File_To_Module[pf.Path]
	if module_index_number < 0 {
		return false
	}
	m := modules.Modules[module_index_number]
	relative := pf.Path
	if m.Root != "." {
		relative = strings.TrimPrefix(pf.Path, m.Root+"/")
	}
	canonical := module_index_canonicalize(path.Dir(relative))
	if canonical == "." {
		return false
	}
	return len(module_information_library_ancestors(m, canonical)) == 1
}

func check_no_impure_stdlib_per_file(
	file_set *token.FileSet, file *ast.File,
) (diags []Diagnostic) {

	const import_message = "impure stdlib import %q: see lint/README.md for resolutions"
	const call_message = "impure stdlib call %s.%s: see lint/README.md for resolutions"
	for _, implementation := range file.Imports {
		path := strings.Trim(implementation.Path.Value, `"`)
		if !is_impure_hard_import(path) {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(implementation.Pos()),
			Message:  fmt.Sprintf(import_message, path),
		})
	}
	local_to_path := make(map[string]string, len(file.Imports))
	for _, implementation := range file.Imports {
		path := strings.Trim(implementation.Path.Value, `"`)
		name := ""
		switch {
		case implementation.Name != nil:
			name = implementation.Name.Name
		default:
			slash_offset := strings.LastIndex(path, "/")
			name = path[slash_offset+1:]
		}
		if name == "_" {
			continue
		}
		if name == "." {
			continue
		}
		local_to_path[name] = path
	}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		selection, is_selection := n.(*ast.SelectorExpr)
		if !is_selection {
			return true
		}
		ident, is_ident := selection.X.(*ast.Ident)
		if !is_ident {
			return true
		}
		path, has := local_to_path[ident.Name]
		if !has {
			return true
		}
		soft_input := &is_impure_soft_ident_input{Package: path, Name: selection.Sel.Name}
		if !is_impure_soft_ident(soft_input) {
			return true
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(selection.Pos()),
			Message:  fmt.Sprintf(call_message, path, selection.Sel.Name),
		})
		return true
	})
	return diags
}

func is_impure_hard_import(path string) (yes bool) {

	switch path {
	case "os", "os/exec", "os/user", "os/signal",
		"flag", "runtime", "math/rand", "crypto/rand":
		return true
	}
	return false
}

type is_impure_soft_ident_input struct {
	Package string
	Name    string
}

func is_impure_soft_ident(input *is_impure_soft_ident_input) (yes bool) {

	switch input.Package {
	case "time":
		switch input.Name {
		case "Now", "Since", "Until", "Sleep",
			"After", "Tick", "NewTimer", "NewTicker":
			return true
		}
	case "fmt":
		switch input.Name {
		case "Print", "Println", "Printf":
			return true
		}
	case "net/http":
		switch input.Name {
		case "Get", "Post", "PostForm", "Head",
			"DefaultClient", "DefaultTransport", "DefaultServeMux",
			"Handle", "HandleFunc", "ListenAndServe", "ListenAndServeTLS":
			return true
		}
	case "net":
		switch input.Name {
		case "Dial", "DialTimeout",
			"LookupHost", "LookupIP", "LookupAddr", "LookupCNAME",
			"LookupMX", "LookupNS", "LookupTXT", "LookupSRV", "LookupPort",
			"DefaultResolver":
			return true
		}
	}
	return false
}

// A pure package's purity must hold across its whole dependency closure, not
// just its own file. The Impure Stdlib rule already runs on every library-tier
// package, so a pure package importing only pure first-party packages stays
// pure by induction — no call graph is needed. This rule supplies the missing
// edge: a pure package may not import a first-party package that is itself
// impure (a `default` package, or one nested a Go ancestor below the library
// tier), nor call a curated stdlib API that reaches the impure set only
// transitively. Unlike
// Impure Stdlib, the ban binds the pure package's _test.go files too; direct
// leaf use (os.Getenv, time.Now) in tests remains a matter for Impure Stdlib.
func check_transitive_purity(
	parsed_files []parsed_file, modules *module_index, instrumentation []string,
) (diags []Diagnostic) {

	for _, pf := range parsed_files {
		if parsed_file_is_impure_package(pf, modules) {
			continue
		}
		diags = append(diags, check_transitive_purity_per_file(
			pf.File_Set, pf.File, modules, instrumentation)...)
	}
	return diags
}

// True iff the file belongs to an impure package: package main, or a `default`
// package or one a Go ancestor below the library tier. Those are the very
// packages a pure package may not depend on, so they are exempt as callers too
// — including their _test.go files, classified by directory since tests carry
// no entry in Directory_Package. A file owned by no module (index -1) is left
// to the downstream no-op convention every other doctrine check follows.
func parsed_file_is_impure_package(pf parsed_file, modules *module_index) (yes bool) {

	base := strings.TrimSuffix(pf.File.Name.Name, "_test")
	if base == "main" {
		return true
	}
	module_index_number := modules.File_To_Module[pf.Path]
	if module_index_number < 0 {
		return false
	}
	m := modules.Modules[module_index_number]
	relative := pf.Path
	if m.Root != "." {
		relative = strings.TrimPrefix(pf.Path, m.Root+"/")
	}
	canonical := module_index_canonicalize(path.Dir(relative))
	return directory_is_impure(canonical, m)
}

// True iff the module-relative directory holds an impure package: a `default`
// directory (the naming convention for an impure global binding, see
// check_default_package_name) or a package sitting exactly one non-main Go
// ancestor below the library tier.
func directory_is_impure(canonical string, m module_information) (yes bool) {

	if canonical == "." {
		return false
	}
	last := canonical
	slash_offset := strings.LastIndex(canonical, "/")
	if slash_offset >= 0 {
		last = canonical[slash_offset+1:]
	}
	if last == "default" {
		return true
	}
	return len(module_information_library_ancestors(m, canonical)) == 1
}

// Flags the two routes impurity launders into a pure file: an import of an
// impure first-party package, and a call to a curated stdlib API that reaches
// the impure set only transitively.
func check_transitive_purity_per_file(
	file_set *token.FileSet, file *ast.File, modules *module_index, instrumentation []string,
) (diags []Diagnostic) {

	const import_message = "impure dependency %q: a pure package imports only pure packages"
	const call_message = "impure transitive call %s.%s: a pure package calls only pure APIs"
	local_to_path := make(map[string]string, len(file.Imports))
	for _, implementation := range file.Imports {
		import_path := strings.Trim(implementation.Path.Value, `"`)
		if import_path_is_impure_first_party(import_path, modules) {
			if !import_path_is_instrumentation(import_path, modules, instrumentation) {
				diags = append(diags, Diagnostic{
					Position: file_set.Position(implementation.Pos()),
					Name:     "transitive-purity",
					Want:     "import only pure first-party packages",
					Message:  fmt.Sprintf(import_message, import_path),
				})
			}
		}
		name := import_local_name(implementation, import_path)
		if name == "_" {
			continue
		}
		if name == "." {
			continue
		}
		local_to_path[name] = import_path
	}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		selection, is_selection := n.(*ast.SelectorExpr)
		if !is_selection {
			return true
		}
		ident, is_ident := selection.X.(*ast.Ident)
		if !is_ident {
			return true
		}
		import_path, has := local_to_path[ident.Name]
		if !has {
			return true
		}
		curated := &is_transitive_stdlib_ident_input{
			Package: import_path, Name: selection.Sel.Name}
		if !is_transitive_stdlib_ident(curated) {
			return true
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(selection.Pos()),
			Name:     "transitive-purity",
			Want:     "call only pure stdlib APIs",
			Message:  fmt.Sprintf(call_message, import_path, selection.Sel.Name),
		})
		return true
	})
	return diags
}

// Resolves the in-file identifier an import binds: its explicit alias, or the
// final path segment when none is given.
func import_local_name(implementation *ast.ImportSpec, import_path string) (name string) {

	if implementation.Name != nil {
		return implementation.Name.Name
	}
	slash_offset := strings.LastIndex(import_path, "/")
	return import_path[slash_offset+1:]
}

// True iff the import path resolves to a first-party package that is itself
// impure (a `default` package, or one a Go ancestor below the library tier). A stdlib
// or third-party path is owned by no module and so is never first-party here.
func import_path_is_impure_first_party(import_path string, modules *module_index) (yes bool) {

	module_index_number := module_index_for_import_path(import_path, modules)
	if module_index_number < 0 {
		return false
	}
	m := modules.Modules[module_index_number]
	relative := strings.TrimPrefix(import_path, m.Module_Path)
	relative = strings.TrimPrefix(relative, "/")
	if relative == "" {
		relative = "."
	}
	canonical := module_index_canonicalize(relative)
	return directory_is_impure(canonical, m)
}

// Returns the index of the module whose path is the longest prefix of the
// import path, or -1 for a stdlib/third-party path owned by no module.
func module_index_for_import_path(import_path string, modules *module_index) (index int) {

	index = -1
	for i := range modules.Modules {
		m := modules.Modules[i]
		if m.Module_Path == "" {
			continue
		}
		under := &import_path_under_module_input{
			Import_Path: import_path, Module_Path: m.Module_Path}
		if !import_path_under_module(under) {
			continue
		}
		if index < 0 {
			index = i
			continue
		}
		if len(m.Module_Path) > len(modules.Modules[index].Module_Path) {
			index = i
		}
	}
	return index
}

type import_path_under_module_input struct {
	// Import_Path is the candidate package path under test.
	Import_Path string
	// Module_Path is the module's declared import prefix.
	Module_Path string
}

// True iff the import path names the module itself or a package within it.
func import_path_under_module(input *import_path_under_module_input) (yes bool) {

	if input.Import_Path == input.Module_Path {
		return true
	}
	return strings.HasPrefix(input.Import_Path, input.Module_Path+"/")
}

type is_transitive_stdlib_ident_input struct {
	// Package is the imported stdlib path the selector reads from.
	Package string
	// Name is the selected identifier called on that package.
	Name string
}

// True iff the stdlib selector reaches the impure set only transitively. The
// set is curated and grown by hand rather than computed, so the rule never
// descends into stdlib sources. Each entry names an API whose package is not
// itself a hard-banned import yet whose body touches an impure effect — the
// process, the wall clock, the filesystem, the OS trust store, or the network.
// Pure siblings (filepath.Join, context.WithCancel) are deliberately absent, as
// are the observability writes (log.Print) the doctrine exempts.
func is_transitive_stdlib_ident(input *is_transitive_stdlib_ident_input) (yes bool) {

	switch input.Package {
	case "log":
		switch input.Name {
		// The log.Fatal family calls os.Exit, terminating the process.
		case "Fatal", "Fatalf", "Fatalln":
			return true
		}
	case "context":
		switch input.Name {
		// WithTimeout and WithDeadline read the wall clock to set the deadline.
		case "WithTimeout", "WithDeadline":
			return true
		}
	case "path/filepath":
		switch input.Name {
		// These resolve against the real filesystem or working directory.
		case "Abs", "Walk", "WalkDir", "Glob", "EvalSymlinks":
			return true
		}
	case "crypto/x509":
		switch input.Name {
		// SystemCertPool reads the host's trust store off disk.
		case "SystemCertPool":
			return true
		}
	case "net/smtp":
		switch input.Name {
		// SendMail dials a remote server, like the net/http senders.
		case "SendMail":
			return true
		}
	}
	return false
}

// Bare `for {}` (and its twins `for ;; {}` and `for true {}`) hide the
// loop's termination condition inside the body. Readers can no longer cap
// iteration from the header alone. The intentional-unbounded escape hatch
// is `for range invariant.GameLoop()`, which is a *ast.RangeStmt and thus
// not caught here — choosing a different syntactic form *is* the assertion
// that the loop is unbounded on purpose.
func check_no_bare_for(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	const message = "bare `for {}` is banned" +
		" if the loop is intentionally unbounded"
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		f, is_for := n.(*ast.ForStmt)
		if !is_for {
			return true
		}
		if f.Init != nil {
			return true
		}
		if f.Post != nil {
			return true
		}
		if !is_bare_for_condition(f.Cond) {
			return true
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(f.Pos()),
			Message:  message,
		})
		return true
	})
	return diags
}

// True for Cond shapes that don't actually constrain iteration: nil (bare
// `for {}` or `for ;; {}`) and the literal identifier `true` (`for true {}`).
// Any other expression is treated as a real condition — Tier B's job to
// scrutinize further.
func is_bare_for_condition(condition ast.Expr) (yes bool) {

	if condition == nil {
		return true
	}
	ident, is_ident := condition.(*ast.Ident)
	if !is_ident {
		return false
	}
	return ident.Name == "true"
}

// Walks file.Imports and returns the set of import-spec local names that
// resolve to the invariant package or its composition tier. The local name
// defaults to the package's declared name (`invariant` for the pure tier,
// `invariant_default` for the composition tier) unless an explicit alias
// is supplied. Files that don't import either path return an empty set.
func collect_invariant_idents(file *ast.File) (idents map[string]bool) {
	idents = map[string]bool{}
	const pure_path = "github.com/james-orcales/james-orcales/shared/invariant/v2"
	const default_path = "github.com/james-orcales/james-orcales/shared/" +
		"invariant/v2/invariant_default"
	for _, import_specification := range file.Imports {
		path := strings.Trim(import_specification.Path.Value, `"`)
		if path != pure_path {
			if path != default_path {
				continue
			}
		}
		if import_specification.Name != nil {
			idents[import_specification.Name.Name] = true
			continue
		}
		if path == default_path {
			idents["invariant_default"] = true
			continue
		}
		idents["invariant"] = true
	}
	return idents
}

// Reports whether type_expression is a `<stdlib>.Mutex` or `<stdlib>.RWMutex`
// selector. The package qualifier must be in stdlib_imports; the selector
// name must be Mutex or RWMutex. Mutex/RWMutex are unique to the sync stdlib
// package, so the qualifier check is sufficient without further sync-specific
// resolution.
func type_expression_is_mutex(
	type_expression ast.Expr, stdlib_imports map[string]bool,
) (yes bool) {
	selector, is_selector := type_expression.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	package_ident, is_package := selector.X.(*ast.Ident)
	if !is_package {
		return false
	}
	if !stdlib_imports[package_ident.Name] {
		return false
	}
	if selector.Sel.Name == "Mutex" {
		return true
	}
	if selector.Sel.Name == "RWMutex" {
		return true
	}
	return false
}

// Walks file.Imports and returns the set of local names that resolve to
// a stdlib import path. The local name is the import's explicit alias
// when present; otherwise the last `/`-separated segment of the import
// path, matching Go's default. Dot imports and blank imports contribute
// nothing — dot imports are forbidden by check_no_dot_import, and blank
// imports don't introduce a usable local name.
func collect_stdlib_imports(
	file *ast.File,
) (names map[string]bool) {

	names = map[string]bool{}
	for _, import_specification := range file.Imports {
		path := strings.Trim(import_specification.Path.Value, `"`)
		if !import_path_is_stdlib(path) {
			continue
		}
		local := ""
		if import_specification.Name != nil {
			local = import_specification.Name.Name
		} else {
			last_slash := strings.LastIndexByte(path, '/')
			if last_slash < 0 {
				local = path
			} else {
				local = path[last_slash+1:]
			}
		}
		if local == "_" {
			continue
		}
		if local == "." {
			continue
		}
		names[local] = true
	}
	return names
}

// Reports whether import_path is part of the Go standard library. The
// heuristic: stdlib paths never contain a `.` in their first
// `/`-separated segment, while third-party paths do (github.com/…,
// golang.org/x/…, gopkg.in/…). Matches the convention `go list std`
// follows.
func import_path_is_stdlib(import_path string) (yes bool) {
	first_slash_offset := strings.IndexByte(import_path, '/')
	if first_slash_offset < 0 {
		return !strings.ContainsRune(import_path, '.')
	}
	return !strings.ContainsRune(import_path[:first_slash_offset], '.')
}

// Switch-based lookup over the v1 ban list, returning the diagnostic the
// caller should emit for a banned `pkg.Identifier` selector. Encoded as a
// switch rather than a slice so the data lives in code (no package-level
// var, no rebuilt slice per call), and packaged as a Diagnostic return so
// the caller doesn't juggle category / substitution strings whose static
// shape can't anchor Lo/Hi coverage.
//
// Takes file_set so the diagnostic carries the selector's source position.
// Returns (Diagnostic{}, false) on a miss (X is not an Ident, or the
// pkg.Identifier doesn't match the switch).
func check_no_unbounded_apis_lookup(
	file_set *token.FileSet, selector *ast.SelectorExpr,
) (diag Diagnostic, found bool) {

	package_identifier, is_ident := selector.X.(*ast.Ident)
	if !is_ident {
		return Diagnostic{}, false
	}
	qualified := package_identifier.Name + "." + selector.Sel.Name

	category, substitution, banned := check_no_unbounded_apis_classify_stdlib(qualified)
	if !banned {
		category, substitution, banned = check_no_unbounded_apis_classify_network(qualified)
	}
	if !banned {
		return Diagnostic{}, false
	}
	return Diagnostic{
		Position: file_set.Position(selector.Pos()),
		Name:     qualified,
		Want:     substitution,
		Message: fmt.Sprintf(
			"%s: unbounded API '%s'; use %s instead",
			category,
			qualified,
			substitution),
	}, true
}

// Classifies stdlib read/decode/decompression/allocation APIs. Returns the
// diagnostic category and suggested substitution, or banned=false when the
// qualified name is not one this half recognises (the caller then tries the
// net/ioutil half).
func check_no_unbounded_apis_classify_stdlib(
	qualified string,
) (category, substitution string, banned bool) {
	switch qualified {
	case "io.ReadAll":
		return "unbounded-read", "io.ReadFull(r, buf) with a bounded buf", true
	case "io.Copy", "io.CopyBuffer":
		return "unbounded-read", "io.CopyN(dst, src, N)", true
	case "os.ReadFile":
		return "unbounded-read",
			"os.Open + io.ReadFull(io.LimitReader(f, N), buf)", true
	case "os.ReadDir":
		return "unbounded-read", "a bounded-slice helper that caps the result", true
	case "bufio.NewScanner", "bufio.NewReader":
		return "unbounded-read", "r.Read(buf) with a fixed buf", true
	case "json.NewDecoder":
		return "unbounded-decode", "json.Unmarshal over a bounded []byte", true
	case "xml.NewDecoder":
		return "unbounded-decode", "xml.Unmarshal over a bounded []byte", true
	case "gob.NewDecoder":
		return "unbounded-decode",
			"bounded read into []byte, then decode manually", true
	case "csv.NewReader":
		return "unbounded-decode", "bounded read + manual parse", true
	case "gzip.NewReader",
		"flate.NewReader",
		"zlib.NewReader",
		"bzip2.NewReader",
		"lzw.NewReader":
		return "unbounded-decompression",
			"wrap the decompressed reader in io.LimitReader", true
	case "zip.NewReader", "zip.OpenReader":
		return "unbounded-decompression",
			"check UncompressedSize64 against a literal cap before reading", true
	case "tar.NewReader":
		return "unbounded-decompression",
			"check Header.Size against a literal cap before reading", true
	case "bytes.NewBuffer", "bytes.NewBufferString":
		return "unbounded-allocation",
			"a fixed []byte with explicit length tracking", true
	default:
		return "", "", false
	}
}

// Classifies net/http and deprecated ioutil APIs. Companion to classify_stdlib;
// returns banned=false when the qualified name is not one of these.
func check_no_unbounded_apis_classify_network(
	qualified string,
) (category, substitution string, banned bool) {
	switch qualified {
	case "http.Get":
		return "unbounded-http", "(&http.Client{Timeout: N}).Get(...)", true
	case "http.Post":
		return "unbounded-http", "(&http.Client{Timeout: N}).Post(...)", true
	case "http.PostForm":
		return "unbounded-http", "(&http.Client{Timeout: N}).PostForm(...)", true
	case "http.Head":
		return "unbounded-http", "(&http.Client{Timeout: N}).Head(...)", true
	case "http.ListenAndServe", "http.ListenAndServeTLS":
		return "unbounded-http",
			"explicit http.Server with timeouts and MaxHeaderBytes set", true
	case "http.DefaultClient":
		return "unbounded-http", "an explicit http.Client with Timeout set", true
	case "http.DefaultServeMux":
		return "unbounded-http", "an explicit http.ServeMux", true
	case "http.DefaultTransport":
		return "unbounded-http", "an explicit http.Transport", true
	case "ioutil.ReadAll":
		return "deprecated-ioutil", "io.ReadFull(r, buf) with a bounded buf", true
	case "ioutil.ReadFile":
		return "deprecated-ioutil",
			"os.Open + io.ReadFull(io.LimitReader(f, N), buf)", true
	case "ioutil.ReadDir":
		return "deprecated-ioutil", "a bounded-slice helper", true
	case "ioutil.WriteFile":
		return "deprecated-ioutil", "os.WriteFile", true
	case "ioutil.TempFile":
		return "deprecated-ioutil", "os.CreateTemp", true
	case "ioutil.TempDir":
		return "deprecated-ioutil", "os.MkdirTemp", true
	case "ioutil.NopCloser":
		return "deprecated-ioutil", "io.NopCloser", true
	case "ioutil.Discard":
		return "deprecated-ioutil", "io.Discard", true
	default:
		return "", "", false
	}
}

// Anchored single-line match for the Go-tooling generated-source header.
// See cmd/go/internal/generate/generate.go in the Go toolchain — the
// trailing period is optional in the wild, hence `\.?`.
var generated_re = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.?$`)

// Flags every selector expression whose `pkg.Identifier` pair appears in
// banned_unbounded_apis. The walk inspects bare *ast.SelectorExpr nodes —
// call sites (`io.ReadAll(r)`) and value references (`http.DefaultClient`)
// are both caught. Canonical package idents are guaranteed by tier-1's
// check_no_dot_import and check_default_package_name, so the X ident
// always resolves to the package's true name.
//
// Generated files (header `// Code generated ... DO NOT EDIT`) are exempt —
// the substitution would have to flow through the generator anyway, and
// the user owns the generator, not the generated output.
func check_no_unbounded_apis(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {

	if check_no_unbounded_apis_is_generated(file) {
		return nil
	}

	ast.Inspect(file, func(n ast.Node) (descend bool) {
		selector_expression, is_selector := n.(*ast.SelectorExpr)
		if !is_selector {
			return true
		}
		diag, found := check_no_unbounded_apis_lookup(file_set, selector_expression)
		if !found {
			return true
		}
		diags = append(diags, diag)
		return true
	})
	return diags
}

// Reports whether the file carries the `// Code generated ... DO NOT EDIT`
// header anywhere in its comment groups. The convention is line-oriented
// and matched verbatim by generated_re.
func check_no_unbounded_apis_is_generated(file *ast.File) (yes bool) {

	for _, comment_group := range file.Comments {
		for _, comment := range comment_group.List {
			if generated_re.MatchString(comment.Text) {
				return true
			}
		}
	}
	return false
}

// Bundles check_deterministic's two string-slice lists — the deterministic packages
// and the instrumentation exemptions — which would otherwise repeat a parameter
// type.
type check_deterministic_input struct {
	// Parsed_Files is every parsed file in the workspace.
	Parsed_Files []parsed_file
	// Modules is the resolved module index.
	Modules *module_index
	// Packages is lint.json's deterministic_packages.
	Packages []string
	// Instrumentation is lint.json's instrumentation_packages: write-only imports a
	// deterministic package may make despite the induction.
	Instrumentation []string
	// Scan_Prefixes is the scope-narrowed parse set, or nil for a whole-workspace
	// run. The coverage check is bounded by it so a scoped run does not flag an
	// entry for a module it never parsed.
	Scan_Prefixes []string
}

// Enforces the opt-in deterministic tier: a deterministic_packages entry names a
// module's top-level directory and the tier auto-applies to that module's pure
// packages, since the fixed module shape lets purity stand in for an explicit
// listing. A covered package is held, atop purity, to bans on every construct
// whose result is decided outside the program — a goroutine, a channel, a
// select, or a time/context/sync import — and may import only other
// deterministic first-party packages. Impure packages in the subtree (the main
// package, a default tier) are not deterministic, so expansion drops them rather
// than reporting them. The bans bind a covered package's _test.go files too.
func check_deterministic(input *check_deterministic_input) (diags []Diagnostic) {

	pure := deterministic_pure_directories(input.Parsed_Files, input.Modules)

	// Expand each entry to the pure package directories at or under it, so a
	// module's top-level directory covers its packages without listing each. The
	// trailing /* of the shared/* form is stripped to the parent directory it
	// names; an entry is otherwise path-cleaned so "./pkg/" and "pkg" name the one
	// directory the parsed files are keyed by. The expansion runs before the
	// checks so the import induction tests against the concrete covered
	// directories, not the coarse entry, which would falsely flag a covered import.
	covered := map[string]bool{}
	matched := map[string]bool{}
	for _, entry := range input.Packages {
		base := strings.TrimSuffix(path.Clean(entry), "/*")
		for directory := range pure {
			under := directory == base
			if !under {
				under = strings.HasPrefix(directory, base+"/")
			}
			if !under {
				continue
			}
			covered[directory] = true
			matched[path.Clean(entry)] = true
		}
	}
	for _, pf := range input.Parsed_Files {
		if !covered[path.Dir(pf.Path)] {
			continue
		}
		diags = append(diags, check_deterministic_constructs(pf.File_Set, pf.File)...)
		diags = append(diags, check_deterministic_imports(
			pf.File_Set, pf.File, input.Modules, covered, input.Instrumentation)...)
	}
	return append(diags, check_deterministic_coverage(&check_deterministic_coverage_input{
		Packages:      input.Packages,
		Matched:       matched,
		Scan_Prefixes: input.Scan_Prefixes,
	})...)
}

// Returns the set of package directories — keyed as path.Dir gives a parsed
// file's path — whose package is pure, the only candidates the deterministic
// tier may cover. A directory is impure if any of its files is an impure package
// (main, a default tier, or a package below the library tier), so the pure set is
// every package directory minus those.
func deterministic_pure_directories(
	parsed_files []parsed_file, modules *module_index,
) (pure map[string]bool) {

	impure := map[string]bool{}
	pure = map[string]bool{}
	for _, pf := range parsed_files {
		directory := path.Dir(pf.Path)
		if parsed_file_is_impure_package(pf, modules) {
			impure[directory] = true
			continue
		}
		pure[directory] = true
	}
	// Files in one directory share a package, so a directory is wholly pure or
	// wholly impure; the subtraction only guards the rare read where an external
	// _test package's pure clause lands beside its impure source.
	for directory := range impure {
		delete(pure, directory)
	}
	return pure
}

// Bundles check_deterministic_coverage's inputs: the entry list and the scan
// prefixes both being string slices repeat a type, which the input-struct rule folds.
type check_deterministic_coverage_input struct {
	// Packages is lint.json's deterministic_packages, reported verbatim on a gap.
	Packages []string
	// Matched marks, by cleaned entry, which entries covered a pure package.
	Matched map[string]bool
	// Scan_Prefixes is the scope-narrowed parse set, nil for a whole-workspace run;
	// an entry outside it was never parsed and so is not judged.
	Scan_Prefixes []string
}

// Reports any deterministic_packages entry that covered no pure package. A typo,
// a stale path, or a directory holding nothing pure would otherwise opt nothing
// into the tier and pass silently — the exact coverage gap the tier exists to
// close. An entry outside the scan prefixes is skipped: a scoped run never parsed
// its module, so its emptiness is an artifact of scope, not a real gap, and a
// full run (nil prefixes, which scan_prefixes_reach admits everywhere) judges it.
func check_deterministic_coverage(
	input *check_deterministic_coverage_input,
) (diags []Diagnostic) {

	for _, entry := range input.Packages {
		if input.Matched[path.Clean(entry)] {
			continue
		}
		base := strings.TrimSuffix(path.Clean(entry), "/*")
		if !scan_prefixes_reach(input.Scan_Prefixes, base) {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: token.Position{Filename: "<lint.json>"},
			Name:     "deterministic",
			Want:     "every deterministic_packages entry covers a pure package",
			Message: fmt.Sprintf(
				"deterministic_packages: no pure package found at %q", entry),
			Tier: 1,
		})
	}
	return diags
}

// Flags the nondeterministic control constructs a deterministic package may not
// contain: a goroutine (the kernel decides its interleaving), any channel use —
// type, send, or receive — and a select, whose ready-case choice the runtime
// randomizes.
func check_deterministic_constructs(
	file_set *token.FileSet, file *ast.File,
) (diags []Diagnostic) {

	report := func(position token.Position, tail string) {
		diags = append(diags, Diagnostic{
			Position: position,
			Name:     "deterministic",
			Message:  "deterministic package " + tail,
			Tier:     1,
		})
	}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		switch node := n.(type) {
		case *ast.GoStmt:
			report(file_set.Position(node.Pos()), "must not start a goroutine")
		case *ast.SelectStmt:
			report(file_set.Position(node.Pos()), "must not use select")
		case *ast.ChanType:
			report(file_set.Position(node.Pos()), "must not use a channel")
		case *ast.SendStmt:
			report(file_set.Position(node.Pos()), "must not use a channel")
		case *ast.UnaryExpr:
			if node.Op == token.ARROW {
				report(file_set.Position(node.Pos()), "must not use a channel")
			}
		}
		return true
	})
	return diags
}

// Flags the imports a deterministic package may not make: time and context
// launder ambient wall-clock time and cancellation past the injected clock, sync
// and sync/atomic guard a concurrency it does not have, and any first-party
// package that is not itself deterministic breaks the induction. An
// instrumentation package is exempt, as it is for transitive purity — a
// write-only side channel feeds no nondeterminism back into the importer.
func check_deterministic_imports(
	file_set *token.FileSet, file *ast.File, modules *module_index, set map[string]bool,
	instrumentation []string,
) (diags []Diagnostic) {

	for _, implementation := range file.Imports {
		import_path := strings.Trim(implementation.Path.Value, `"`)
		if is_nondeterministic_import(import_path) {
			diags = append(diags, Diagnostic{
				Position: file_set.Position(implementation.Pos()),
				Name:     "deterministic",
				Message: fmt.Sprintf(
					"deterministic package must not import %q", import_path),
				Tier: 1,
			})
			continue
		}
		if !import_path_is_nondeterministic_first_party(import_path, modules, set) {
			continue
		}
		if import_path_is_instrumentation(import_path, modules, instrumentation) {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(implementation.Pos()),
			Name:     "deterministic",
			Message: fmt.Sprintf(
				"deterministic package must import only deterministic packages: %q",
				import_path),
			Tier: 1,
		})
	}
	return diags
}

// True iff the import path is a stdlib package a deterministic package may not
// use: time and context launder the wall clock and cancellation, sync and
// sync/atomic exist only to coordinate a concurrency it does not have.
func is_nondeterministic_import(path string) (yes bool) {

	switch path {
	case "time", "context", "sync", "sync/atomic":
		return true
	}
	return false
}

// True iff the import path resolves to a first-party package that is not itself
// in the deterministic set. A stdlib or third-party path is owned by no module,
// so it is never flagged here — stdlib is policed by is_nondeterministic_import,
// third-party is out of scope, the same blind spot transitive purity carries.
func import_path_is_nondeterministic_first_party(
	import_path string, modules *module_index, set map[string]bool,
) (yes bool) {

	module_index_number := module_index_for_import_path(import_path, modules)
	if module_index_number < 0 {
		return false
	}
	m := modules.Modules[module_index_number]
	return !set[import_path_workspace_directory(import_path, m)]
}

// Maps a first-party import path to the workspace-root-relative package directory
// the deterministic set is keyed by: strip the module path to the in-module
// subpath, then re-root it under the module's workspace directory, mirroring the
// form path.Dir gives a parsed file so set membership matches.
func import_path_workspace_directory(
	import_path string, m module_information,
) (directory string) {

	relative := strings.TrimPrefix(import_path, m.Module_Path)
	relative = strings.TrimPrefix(relative, "/")
	if m.Root == "." {
		if relative == "" {
			return "."
		}
		return relative
	}
	if relative == "" {
		return m.Root
	}
	return m.Root + "/" + relative
}

// Stdlib time is the one ambient source of real wall-clock time. Funneling every
// read through a single gateway keeps the injected Clock the only way the rest of
// the shared module sees the clock, so within the shared library importing stdlib
// "time" is allowed only in the time/default gateway; every other package injects
// a Clock. Binary modules are out of scope — separate tools with their own needs.
func check_time_import_gateway(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {

	gateway := module_index_time_gateway(modules)
	if gateway == "" {
		return nil
	}
	for _, pf := range parsed_files {
		module_index_number := modules.File_To_Module[pf.Path]
		if module_index_number < 0 {
			continue
		}
		if !modules.Modules[module_index_number].Is_Shared_Library {
			continue
		}
		if path.Dir(pf.Path) == gateway {
			continue
		}
		for _, implementation := range pf.File.Imports {
			if strings.Trim(implementation.Path.Value, `"`) != "time" {
				continue
			}
			diags = append(diags, Diagnostic{
				Position: pf.File_Set.Position(implementation.Pos()),
				Name:     "stdlib-time",
				Want:     "import the time/default gateway and inject a Clock",
				Message: fmt.Sprintf(
					"stdlib time may be imported only by %q; inject the Clock",
					gateway),
				Tier: 1,
			})
		}
	}
	return diags
}

// Returns the workspace-relative directory of the shared module's stdlib-time
// gateway (its time/default), or "" when no module is the shared library.
func module_index_time_gateway(modules *module_index) (gateway string) {

	for _, m := range modules.Modules {
		if !m.Is_Shared_Library {
			continue
		}
		if m.Root == "." {
			return "time/default"
		}
		return m.Root + "/time/default"
	}
	return ""
}
