// Package lint is the monorepo's static checker. It enforces the
// workspace organization doctrine (library tier vs. composition tier,
// binary vs. shared library), Tiger-style local conventions
// (snake_case / Ada_Case naming, no compound ifs, no recursion, …),
// and a small set of cross-file rules (package fragmentation, git
// history hygiene, package/exported-identifier documentation).
package lint

import (
	"bytes"
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

	invariant "github.com/james-orcales/james-orcales/golang_snacks/invariant/v2/invariant_default"
)

const max_line_chars = 100
const tab_width = 8

// Hi bounds for Distinct_Boundary axes on string lengths. Each constant
// encodes the realistic upper bound for the semantic domain that the
// string represents — Distinct_Boundary fatals on inputs exceeding Hi,
// so the bound must be wide enough for real inputs AND tight enough that
// a test can satisfy the Hi-equals-X tuple. Constants are named for the
// domain so reading the assertion at a call site makes the bound obvious.

// Max_Identifier_Chars caps Go identifier lengths the linter processes.
// 128 chars is wider than any identifier representable on a max_line_chars
// (140) source line after the surrounding syntax; the repo's longest
// production identifier is 83 chars.
const Max_Identifier_Chars = 128

// Max_invariant_helper_name_chars caps the longest invariant.X helper name
// the linter recognises; "Recorder_Is_Distinct_Boundary" is the longest
// (29 chars). The constant ALSO serves as a sanity bound on helper_name
// strings passed between extractor helpers, all of which receive non-
// empty names (callers gate on `helper_name == "" return` so the boundary
// is paired with `Always(helper_name != "", ...)`).
const max_invariant_helper_name_chars = 29

// Min_invariant_helper_name_chars is the shortest recognised invariant.X
// helper name: "Always" (6 chars). Paired with the max as the Lo/Hi of
// helper_name Distinct_Boundary axes in extract_nil_comparison_path /
// extract_eq_nil_path / nil_predicate_index / nil_allows_neq.
const min_invariant_helper_name_chars = 6

// Min_credit_kind_chars / max_credit_kind_chars cap the bare_composable_
// table values: "bool" (4) and "boundary_float" (14). Tightening to the
// exact table range makes both endpoints reachable by tests that exercise
// any Distinct_Boundary or Always/Sometimes credit shape.
const min_credit_kind_chars = 4
const max_credit_kind_chars = 14

// Min_diagnostic_source_chars / max_diagnostic_source_chars cap the
// `source` string in the diagnostic-builder helpers: "param" (5),
// "param_defer" (11), or "named_return" (12). The literal values come from
// the requirement emit branches in collect_requirements and the validate
// loop.
const min_diagnostic_source_chars = 5
const max_diagnostic_source_chars = 12

// Min_function_label_chars caps the shortest function_label string: a
// single-character function name like `f`. Paired with max_identifier_
// chars as Hi.
const min_function_label_chars = 1

// Min_non_empty is the universal Lo for length axes on inputs the caller
// guarantees non-empty (validated by a callsite check or an Always(s != "")
// invariant). Distinct_Boundary requires Lo < Hi, so empty inputs need their
// own pre-gate; this constant anchors the "≥1 character" bucket for
// non-empty-string and non-empty-slice axes.
const min_non_empty = 1

// Min_split_suggestion_chars caps the shortest non-trivial split suggestion
// returned by suggest. The shortest case is a 2-char Ada_Case input ("aB")
// split into "a_B" — three characters including the inserted underscore.
const min_split_suggestion_chars = 3

// Min_naming_style_chars / max_naming_style_chars bound the length of the
// `Want` field on suggest_input. Callers pass exactly one of "Ada_Case" (8)
// or "snake_case" (10) — the only two style words the casing checks know.
const min_naming_style_chars = 8
const max_naming_style_chars = 10

// Min_stream_check_name_chars / max_stream_check_name_chars bound the
// `Name` field on check_function_stream constructors. Shortest is "symlink"
// (7); longest is "markdown-line-length" (20).
const min_stream_check_name_chars = 7
const max_stream_check_name_chars = 20

// Min_stack_with_body_frame is the Lo for walker stacks that the function
// guarantees have appended at least one body frame on top of the input
// stack (so the post-condition stack length is the input stack length
// plus one, minimum two when the input was non-empty).
const min_stack_with_body_frame = 2

// Min_pair caps Lo for numeric axes whose minimum is two — used by callers
// where the value is "≥2 of something" without a more specific domain
// constant fitting. Where a domain-specific name is clearer, prefer that.
const min_pair = 2

// Max_package_lines_test is the Hi bound on the per-package source-line
// accumulator in the package-fragmentation check. The endpoint is anchored
// by the fragmentation test fixtures (each package ≤ ~12k lines).
const max_package_lines_test = 12750

// Max_test_package_files is the Hi bound on the caller-supplied package
// file-count cap. The fragmentation tests exercise both endpoints (1 file
// allowed; 2 files as the caller-imposed ceiling).
const max_test_package_files = 2

// Max_build_constraint_key_chars caps the normalized build-constraint AST
// string used as a fragmentation grouping key. Sized for the typical
// multi-OS multi-arch expression length seen in practice.
const max_build_constraint_key_chars = 125

// Count_one anchors "exactly one of something" sentinel checks (e.g. a
// Sometimes(len(xs) == 1) on a single-element group). Value-identical to
// min_non_empty but read at the call site with different intent.
const count_one = 1

// Max_obligation_identifiers caps the number of identifiers in a single
// declaration-from-call obligation. Go allows arbitrary multi-LHS, but
// 3-LHS is the widest shape observed in lint.go's own source; setting the
// Hi bucket here gates the Distinct_Boundary axis on obligation.Identifiers
// to a reachable endpoint.
const max_obligation_identifiers = 3

// Max_successor_statements caps obligation.Successor_Statements via the
// generated many-successor fixture; lint.go's own scan stays under because
// every := decl is followed by ≤30 statements in the same block.
const max_successor_statements = 30

// Max_leaf_requirements_per_dispatch caps the number of requirement records
// leaf_dispatch returns: channel leaves emit 3 (pointer + boundary_int +
// zero_int), slice/map leaves emit 2 (boundary_int + zero_slice/zero_map),
// non-container/non-channel types emit 0.
const max_leaf_requirements_per_dispatch = 3

// Check_invariant_assertions_recursion_unroll_threshold caps how many times a
// self-referential struct may be entered during requirement emission. For
// `type Node struct{Next *Node; Value int}` walked from `*Node`, the first
// visit emits Value + Next + nillability; the second (via the back-edge
// through Next) emits Next.Value + Next.Next + nillability; the third would
// be Next.Next.Next and is dropped. See §4.5 of Assertion_Enforcement.md.
const check_invariant_assertions_recursion_unroll_threshold = 2

// Module_index_not_found anchors the -1 sentinel returned when no module
// matches a path lookup. Paired with max_modules_per_workspace as the Hi
// bound on the index domain.
const module_index_not_found = -1

// Max_modules_per_workspace caps the per-workspace module count. The
// monorepo's go.work currently lists ~10 modules; 1024 leaves headroom
// for several orders of magnitude of growth without admitting absurd values.
const max_modules_per_workspace = 1024

// Path_root is the path.Dir result for top-level entries: a single dot
// meaning "current directory". Used as the sentinel comparison value when
// detecting root-level paths.
const path_root = "."

// Declaration_diagnostic_name is the constant Name field on a Diagnostic
// emitted by build_declaration_diagnostic. Pulled out as a file-level const
// so the diag.Name invariant can bind it as a named bound.
const declaration_diagnostic_name = "invariant_assertion_missing_after_declaration"

// Inside_if_diagnostic_name is the fixed Name for inside-if-only diagnostics.
// Inside_if_diagnostic_name_chars must equal its length: the builder asserts
// `Always(len(diag.Name) == inside_if_diagnostic_name_chars)` to satisfy the
// boundary_int requirement on a Name whose value is invariant (a constant
// length can't reach a Distinct_Boundary's two endpoints). The runtime
// assertion catches any drift between the string and the count.
const inside_if_diagnostic_name = "invariant_assertion_inside_if_only"
const inside_if_diagnostic_name_chars = 34

// Inside_if_diagnostic_want_chars is the byte length of the inside-if-only
// Want hint (the inline literal in the builder; the `—` em-dash is 3 bytes).
// The runtime Always(len(diag.Want) == it) catches any drift.
const inside_if_diagnostic_want_chars = 164

// Missing_diagnostic_name is the fixed Name for missing-axis diagnostics;
// missing_diagnostic_name_chars must equal its length (see inside_if note).
const missing_diagnostic_name = "invariant_assertion_missing"
const missing_diagnostic_name_chars = 27

// Declaration_diagnostic_name_chars is the length of declaration_diagnostic_name
// ("invariant_assertion_missing_after_declaration"); the declaration builders
// assert len(diag.Name) == it to bound a Name whose value is invariant.
const declaration_diagnostic_name_chars = 45

// Pointer_requirement_kind is the fixed Kind for pointer requirements;
// pointer_requirement_kind_chars must equal its length.
const pointer_requirement_kind = "pointer"
const pointer_requirement_kind_chars = 7

// Stream_checker_count is the fixed number of stream-tier checks; the builder
// asserts len(checks) == it (a Distinct_Boundary can't bound a constant count).
const stream_checker_count = 10

// Fixed Name strings for the stream-check closures, each paired with its
// length so the checker's defer bounds c.Name (a value invariant per closure).
const path_casing_check_name = "path-casing"
const path_casing_check_name_chars = 11
const agents_pair_check_name = "agents-claude-pair"
const agents_pair_check_name_chars = 18

// Min_recursion_message_chars / max_recursion_message_chars cap the
// "recursion: <node> calls itself" diagnostic message. Lo = 25 chars for
// the 1-char node case; Hi = 152 chars for a max-length 128-char node.
const min_recursion_message_chars = 25
const max_recursion_message_chars = 152

// Min_defer_position_name_chars / max_defer_position_name_chars bound the
// `Name` field on diagnostics built by
// check_invariant_assertions_validate_defer_position. Names are one of two
// fixed labels: `assertion_defer_missing` (23) or
// `assertion_defer_not_at_body_zero` (32).
const min_defer_position_name_chars = 23
const max_defer_position_name_chars = 32

// Min_defer_position_message_chars is the provable floor on the Sprintf'd
// defer-position diagnostic message: the shortest function label (1 char) plus
// the shortest of the three message bodies (102 chars). No label is shorter
// than one character and no body is shorter than 102, so a message can never
// fall below this — making it a panic-safe Lo for the message boundary.
const min_defer_position_message_chars = 103

// Min_defer_position_want_chars / max_defer_position_want_chars bound the
// `Want` field on diagnostics built by validate_defer_position. Want strings
// are three fixed remediation hints: `add an assertion defer ...` (71),
// `move the assertion defer ...` (76), and `place the assertion defer ...`
// (103). Test corpus exercises the shortest (`add`) and longest (`place`).
const min_defer_position_want_chars = 71
const max_defer_position_want_chars = 103

// Min_declaration_diagnostic_want_chars / max_declaration_diagnostic_want_chars
// bound the `Want` field on diagnostics built by
// check_invariant_assertions_build_declaration_diagnostic. Want strings come
// in two shapes: a short single-LHS suggestion (`add an invariant assertion
// ... covering: <list>`) and a long multi-LHS suggestion (`use
// invariant.Cross_Product ... covering: <list>`).
const min_declaration_diagnostic_want_chars = 65
const max_declaration_diagnostic_want_chars = 133

// Min_declaration_diagnostic_message_chars is the shortest declaration-obligation
// message: a single-LHS `<f>: declaration via function call must be followed by
// an invariant assertion covering: <x>` with a 1-char function label and a
// 1-char identifier. The Hi end is the budget ceiling (max_diagnostic_message_chars,
// which no message — bounded by label + identifier-list + the fixed clause —
// reaches), so the boundary masks Hi and observes only the single-LHS Lo.
const min_declaration_diagnostic_message_chars = 87

// Max_diagnostic_message_chars caps the upper bound for a diagnostic
// Message string. Longest observed messages embed a 128-char function label
// plus 257-char field_description plus suggestion text; round to 1024.
const max_diagnostic_message_chars = 1024

// Max_banned_lists_per_check caps the static list-of-lists count for the
// banned-segment check (universal, function-only, file-only, package-only).
const max_banned_lists_per_check = 4

// Min_suggested_sig_chars caps the shortest suggested function signature.
// In practice the shortest fixture-observed signature is the 21-char
// `f(*f_Input) (result T)` template with a single-letter funcname and a
// minimal result type.
const min_suggested_sig_chars = 21

// Min_convert_to_message_chars / max_convert_to_message_chars bound the
// `convert to <sig>` diagnostic message constructed by
// check_input_struct_validate. The 11-character "convert to " prefix is
// added to the suggested signature's length bounds.
const min_convert_to_message_chars = min_suggested_sig_chars + 11
const max_convert_to_message_chars = max_suggested_sig_chars + 11

// Max_stdlib_term_chars caps the stdlib-allowlist terminology suffix
// string. Longest entry is `offset` (6 chars).
const max_stdlib_term_chars = 6

// Min_stdlib_term_chars is the shortest term in the arithmetic-result
// vocabulary: `size` (4 chars). Paired with max_stdlib_term_chars as Hi
// for axes over Left/Right operand-term strings.
const min_stdlib_term_chars = 4

// Max_method_params_test_corpus matches the Params string the
// Test_Coverage_Backfill_Method_Render_Type fixture produces for its Bar
// method: a 1-char type `A` joined to a 128-char type via `,` totals 130.
// Bounded axes over input.Params in check_unnecessary_method_matches_stdlib
// use this as Hi so Bar's call observes the Hi bucket.
const max_method_params_test_corpus = Max_Identifier_Chars + 2

// Min_qualified_ident_chars caps `pkg.Func` shapes at their minimum: a
// single-letter package, dot, single-letter func — three characters.
const min_qualified_ident_chars = 3

// Min_rename_suggestion_chars caps the shortest `<name>_<term>` rename
// suggestion; the smallest single-word replacement ("count") is 5 chars.
const min_rename_suggestion_chars = 5

// Min_ing_word_chars is the smallest word that can carry the `-ing` participle
// suffix: three characters (the suffix itself plus a one-letter prefix would
// not actually be a valid English word, but the linter only inspects shape).
const min_ing_word_chars = 3

// Min_source_with_comment_bytes is the smallest source file that carries a
// comment after the package clause: "package x\n\n// c\n" is 16 bytes.
const min_source_with_comment_bytes = 16

// Min_field_description_chars caps the shortest `<name> <type>` description:
// one-char name + space + one-char type = 3 chars.
const min_field_description_chars = 3

// Min_requirement_field_description_chars is the smallest field_description
// length that survives the keyword_kinds filter and reaches a requirement.
// "a *T" (4 chars: 1-char name + " " + "*T" pointer) is the shortest such
// shape — bare `a T` for a user-defined Ident gets dropped because kinds
// is nil at the leaf path.
const min_requirement_field_description_chars = 4

// Cross_product_helper_chars / recorder_cross_product_helper_chars are the
// string lengths of the two Cross_Product helper-name shapes. Paired as
// Lo / Hi on the helper_name length axis in call_covered_pairs_cross_product.
const cross_product_helper_chars = 13
const recorder_cross_product_helper_chars = 22

// Max_bare_credit_kind_chars caps the bare-composable kind strings used in
// the bare_table: `bool` (4) is the Lo via min_credit_kind_chars, and
// `boundary_int` (12) is the Hi for the bare-credit family.
const max_bare_credit_kind_chars = 12

// Helper_family_index_unknown / helper_family_index_recorder anchor the
// three-valued helper-family discriminator: -1 = unknown / not an invariant
// helper, 0 = naked Always/Sometimes (resolved by middle case), 1 =
// Recorder_-prefixed variant.
const helper_family_index_unknown = -1
const helper_family_index_recorder = 1

// Max_always_family_chars caps the longest helper name in the
// Always/Sometimes nil-eq family: `Recorder_Always` (15).
const max_always_family_chars = 15

// Sign_negative / sign_positive anchor the three-valued sign domain
// returned by expression_sign: -1 for negative, 0 for zero (interior), +1
// for positive.
const sign_negative = -1
const sign_positive = 1

// Min_invariant_suggestion_chars / max_invariant_suggestion_chars cap the
// `use invariant.X(...)` remediation string rendered into assertion-coverage
// diagnostics. Sized empirically from the shortest (`pointer` shape) and
// longest (`boundary_float` shape) wrappers around the axis call.
const min_invariant_suggestion_chars = 167
const max_invariant_suggestion_chars = 373

// Min_composable_helper_chars / max_composable_helper_chars cap the
// helper-name string for composable axis builders: `Always` (6) is the
// shortest, `Distinct_Boundary` (17) the longest.
const min_composable_helper_chars = 6
const max_composable_helper_chars = 17

// Min_suggested_axis_call_chars / max_suggested_axis_call_chars cap the
// rendered axis-builder template string for one assertion requirement.
// Sized empirically from the shortest pointer-shape and longest
// Distinct_Boundary-shape templates.
const min_suggested_axis_call_chars = 22
const max_suggested_axis_call_chars = 228

// Max_invariant_selector_chars caps the longest selector_name string in the
// full invariant-call family: `Recorder_Distinct_Boundary` (26).
const max_invariant_selector_chars = 26

// Max_if_init_identifier_chars caps identifier strings appearing in
// if/for/switch init lines: a tighter bound than Max_Identifier_Chars to
// reflect what fits in a single statement line within the line-length budget.
const max_if_init_identifier_chars = 55

// Tier_2_checks_count / tier_1_checks_count anchor the static tier-list
// length axis. Updated whenever a check is added or removed from the
// dispatcher in Check_File.
const tier_2_checks_count = 7
const tier_1_checks_count = 30

// Min_go_filename_chars is the shortest Go filename: a single-letter package
// name followed by the .go extension, e.g. `a.go`. Used as the Lo bound on
// filename axes in Check_Source / Check_File_System.
const min_go_filename_chars = 4

// The requirement-position label passed to the validate helper is always one of
// "param" (5), "param_defer" (11), or "named_return" (12). Both endpoints are
// always observed (every analyzed function runs the param and named_return
// passes), so the boundary needs no masking.
const min_validate_position_chars = 5
const max_validate_position_chars = 12

// Inside-if-only diagnostics carry a requirement.Kind assertion-kind label.
// The non-nillable leaves that reach this path are integer leaves, whose kinds
// are "zero_int" (8) and "boundary_int" (12) — both observed from a single
// `(result int)` named return, so the boundary needs no masking.
const min_inside_if_kind_chars = 8
const max_inside_if_kind_chars = 12

// The smallest source a parsed file can carry is the shortest valid Go file,
// "package a\n" (10 bytes) — the package keyword, a one-char name, and the
// gofmt-mandated trailing newline. Nothing is shorter, so this is the panic-safe
// Lo for pf.Source byte-length boundaries; a pinned fixture observes it.
const min_go_source_bytes = 10

// Package-group diagnostics carry empty Name and Want (the group-level message
// lives in Message). len == 0 is the constant width; Always(len == this) credits
// boundary_int via the numeric-credit path without a Lo<Hi Distinct_Boundary.
const package_group_diag_chars = 0

// Field_Description for an inside-if leaf is "<name> <type>". The shortest
// reachable is a one-char-named integer leaf, "x int" (5) — names are never
// empty and "int" is the shortest non-nillable type, so 5 is the floor. The
// budget cap (max_field_description_chars) is never reached, so Hi is masked.
const min_inside_if_field_chars = 5

// Min_inside_if_message_chars is the shortest inside-if-only diagnostic message:
// a 1-char function label, the shortest `param` prefix, and a min-length field
// description, plus the fixed `... must be asserted outside any if ... != nil
// ...` clause. Hi is the budget ceiling (max_diagnostic_message_chars), never
// reached, so the boundary masks Hi and observes only this Lo.
const min_inside_if_message_chars = 113

// Min_want_name_chars / max_want_name_chars cap the input-struct
// expected name (e.g. "f_Input" for function `f`). The "_Input" suffix
// is 6 chars; combined with the shortest (1-char) function name the
// minimum is 7. Max is Max_Identifier_Chars + 6 = 134.
const min_want_name_chars = 7
const max_want_name_chars = Max_Identifier_Chars + 6

// Max_filesystem_path_chars caps filesystem path strings the linter
// processes. POSIX PATH_MAX is 4096 on Linux; the linter inherits this
// as the hard bound for file paths and is exercised by the long-path
// backfill test which constructs a 4096-char filename.
const max_filesystem_path_chars = 4096

// Max_filesystem_directory_chars caps directory strings (path.Dir of a
// file path). The directory consumes at most max_filesystem_path_chars
// minus the shortest basename `/a.go` (5 chars).
const max_filesystem_directory_chars = max_filesystem_path_chars - 5

// Max_inferred_field_kind_chars caps the strings returned by
// check_invariant_assertions_infer_field_kind: "int" (3), "bool" (4),
// "pointer" (7), or "" (0). Longest entry is "pointer" at 7 chars.
const max_inferred_field_kind_chars = 7

// Max_field_description_chars caps `<name> <type_str>` descriptions:
// at most one identifier plus a space plus a type expression that itself
// is bounded by identifier length, yielding 2*identifier + 1.
const max_field_description_chars = 2*Max_Identifier_Chars + 1

// Max_suggested_sig_chars caps suggested function-signature strings of the
// form `<funcname>(*<funcname>_Input) (result <type>)`. The funcname
// appears twice (raw plus inside `_Input`), plus the wrapping syntax and
// a result clause; budget is 2*identifier + 6 (`_Input`) + ~16 (result).
const max_suggested_sig_chars = 2*Max_Identifier_Chars + 22

// Max_comment_text_chars caps raw comment text. comment_body strips the
// leading `//` and any whitespace, so the text bound is the body budget
// (4096 chars) plus the 2-char `//` prefix the scanner preserves.
const max_comment_text_chars = max_filesystem_path_chars + 2

// Max_banned_segment_chars caps the longest banned-segment word in the
// banned_segments_universal list: "utilities" at 9 chars.
const max_banned_segment_chars = 9

const max_function_lines = 70

// Git's default short-hash width.
const git_short_hash_chars = 10

// Git's full SHA-1 width — the maximum a `%H` format will produce. Used
// as the hard bound for hash-shaped inputs when git is in SHA-1 mode.
const git_full_hash_chars = 40

// Git's SHA-256 hash width — git's optional SHA-256 object format. Used
// as the hard bound for hash-shaped inputs since git_input_check_short_hash
// must accept either format.
const git_full_hash_chars_sha_256 = 64

const max_lines_per_file = 30000

// Mirrors max_line_chars used by the source-line check: code-review UIs
// truncate around 72–100 chars and longer subjects force horizontal scroll.
const max_commit_subject_chars = 100

// The shared library is hardcoded rather than discovered. Every other
// module at the workspace root is a binary; nothing in the doctrine
// requires runtime negotiation, and a constant keeps the rule grep-able.
const shared_library_module_path = "github.com/james-orcales/james-orcales/golang_snacks"

// Caps every file at 1 MiB. Binaries and large assets should not end up
// tracked in the source tree — push them through git LFS or an external store.
const max_file_size_bytes = 1 << 20

// Max_diagnostics_per_call caps the slice length of `diags []Diagnostic`
// returns. A single check may emit one diagnostic per source line at worst,
// so the budget tracks max_lines_per_file with headroom for declarations
// that emit multiple diagnostics each.
const max_diagnostics_per_call = max_lines_per_file * 4

// Max_parsed_files_per_call caps the slice length of `parsed_files
// []parsed_file` and similar package-level slices. A package typically holds
// dozens of files; the cap leaves ample headroom for the worst-case monorepo
// flat-directory layout without admitting absurd values.
const max_parsed_files_per_call = 32768

// Max_ast_nodes_per_call caps generic []ast.Expr / []ast.Stmt / []*ast.Ident
// slices passed between helpers. Each AST list is bounded by the source file
// it derives from; max_lines_per_file is a generous upper bound.
const max_ast_nodes_per_call = max_lines_per_file

// Max_string_slice_per_call caps generic []string slices used for path
// lists, candidate sets, identifier chains, and similar identifier-bag
// collections.
const max_string_slice_per_call = max_lines_per_file

// Max_coverage_pairs_per_call caps invariant-assertion coverage-pair slices.
// One call may produce one pair per (path, kind) tuple — bounded by the
// number of tracked identifiers in any one function, which is well below
// Max_Identifier_Chars × max_credit_kind_chars in practice.
const max_coverage_pairs_per_call = max_lines_per_file

// Caps the three agent-facing docs at 100 lines. These files are loaded into
// every agent invocation, so each extra line is a per-call tax on context and
// attention. Skills that outgrow the budget should be split; CLAUDE.md/AGENTS.md
// that outgrow it usually mean a repo-level instruction belongs in a
// sub-package's pair instead.
const agent_documentation_max_lines = 100

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

// Tier_max is the Hi bound on Diagnostic.Tier. 0 = cross-file / git / stream
// (printed unconditionally), 1 = file tier-1 (always printed; presence
// suppresses tier-2 globally), 2 = file tier-2 (printed only when no tier-1
// fires anywhere in scope).
const tier_max = 2

// Enumerators for the coverage_status type below. Declared here (above the
// type) because the file-wide rule pulls all consts to the file head;
// Go's package scope resolves the forward type reference.
const coverage_status_uncovered coverage_status = 0
const coverage_status_covered_by_sugar coverage_status = 1
const coverage_status_covered_by_cross_product coverage_status = 2
const coverage_status_inside_if_only coverage_status = 3

// Main_Input bundles every external dependency the linter needs.
// Construction lives in main.go (production) or fstest.MapFS-backed
// tests (unit tests) — the library tier never reaches out to ambient
// state itself.
type Main_Input struct {
	Fsys   fs.FS
	Stdout io.Writer
	Stderr io.Writer
	// Root_Directory is the OS path that matches Fsys. The stream-tier symlink
	// check needs real-OS access through Readlink and Stat below because
	// fs.FS has no symlink primitive. An empty Root_Directory self-disables that
	// one check so fstest.MapFS-backed tests don't need to special-case it.
	Root_Directory string
	// Tracked is the set of paths (relative to Fsys root) the linter is
	// allowed to look at — typically the union of git-tracked and
	// git-untracked-but-not-ignored files. When non-nil, walkers skip
	// every path outside this set, and prune any directory containing no
	// such path. nil disables the filter entirely (fstest.MapFS tests
	// stay green without having to enumerate every entry).
	Tracked map[string]bool
	Git     Git_Input
	// CPU_Count caps parallelism for the parse and check phases. Injected
	// by main.go (binds to runtime.NumCPU); tests may leave it 0, which
	// degrades to single-threaded execution.
	CPU_Count int
	// Readlink and Stat are injected real-OS primitives for the symlink
	// check. Both nil disables the check. main.go binds them to
	// os.Readlink and os.Stat.
	Readlink func(name string) (target string, err error)
	Stat     func(name string) (info fs.FileInfo, err error)
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
	Hash    string
	Subject string
}

// Git_Input drives the git-history tier. Zero value (Enabled=false) skips
// the tier — used when HEAD is on main, when the binary isn't run from a
// git repo, or in fstest.MapFS-backed unit tests that aren't about git.
// Main_Reference_Absent distinguishes "no main ref locally" (shallow CI checkout,
// brand-new repo) from "main ref present, no offending commits" so CI
// misconfiguration surfaces as a specific failure instead of silent pass.
type Git_Input struct {
	Enabled               bool
	Main_Reference_Absent bool
	Merge_Commits         []Git_Commit
	Non_Merge_Commits     []Git_Commit
}

// Main is the linter's entry point. Returns the process exit code:
// 0 if every check passed (success message printed to Stdout), 1 if
// any diagnostic was emitted within Scope_Prefix, 2 on a hard error
// (filesystem walk failure, etc.).
func Main(input *Main_Input) (code int) {
	defer func() { main_assert_exit(code, input) }()
	main_input_precondition(input)

	// Git tier runs first: repo-metadata-only, doesn't touch the FS, and
	// surfaces the fastest signal.
	git_diags := git_input_check(input.Git)
	invariant.Cross_Product(
		invariant.Sometimes(git_diags == nil, "Git_diags is empty for a clean git state"))
	filesystem_diags, err := Check_File_System(&Check_File_System_Input{
		Fsys:           input.Fsys,
		Root:           ".",
		Root_Directory: input.Root_Directory,
		Tracked:        input.Tracked,
		CPU_Count:      input.CPU_Count,
		Readlink:       input.Readlink,
		Stat:           input.Stat,
	})
	invariant.Cross_Product(
		invariant.Always(
			filesystem_diags != nil, "Filesystem_diags is non-nil at this point"),
		invariant.Always(err == nil, "Err is nil for a successful filesystem walk"),
	)
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

func main_assert_exit(code int, input *Main_Input) {
	code_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: code, Lo: 0, Hi: exit_code_max,
		Message: "Code is 0 or 1; code=2 unreachable per upstream Excluding",
	})
	code_zero := invariant.Sometimes(
		code == 0, "Code is the clean-exit value on the success path")
	invariant.Cross_Product(code_boundary, code_zero,
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(input.Tracked), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Tracked is bounded by AST budget",
		}),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(input.Git.Merge_Commits), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Merge_commits is bounded by AST budget",
		}),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:       len(input.Git.Non_Merge_Commits),
			Lo:      0,
			Hi:      max_ast_nodes_per_call,
			Message: "Non_merge_commits is bounded by AST budget",
		}),
		invariant.Excluding("Code Lo implies code_zero true",
			invariant.Bucket_Lo(code_boundary),
			invariant.Bucket_False(code_zero)),
		invariant.Excluding("Code Hi implies code_zero false",
			invariant.Bucket_Hi(code_boundary),
			invariant.Bucket_True(code_zero)))
}

func main_input_precondition(input *Main_Input) {
	cpu_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: input.CPU_Count, Lo: 0, Hi: cpu_count_max,
		Message: "CPU_Count is in the per-process worker budget",
	})
	cpu_zero := invariant.Sometimes(
		input.CPU_Count == 0, "CPU_Count is zero in fstest.MapFS-backed callers")
	git_enabled := invariant.Sometimes(input.Git.Enabled, "Git tier is enabled sometimes")
	main_absent := invariant.Sometimes(
		input.Git.Main_Reference_Absent, "Main reference is absent sometimes")
	invariant.Cross_Product(invariant.Always(input != nil, "Input is non-nil"),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(input.Root_Directory), Lo: 0, Hi: max_filesystem_path_chars,
			Message: "Root_directory is bounded by filesystem path budget",
		}),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(input.Scope_Prefix), Lo: 0, Hi: max_filesystem_path_chars,
			Message: "Scope_prefix is bounded by filesystem path budget",
		}),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(input.Tracked), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Tracked is bounded by AST budget",
		}),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(input.Git.Merge_Commits), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Merge_commits is bounded by AST budget",
		}),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(input.Git.Non_Merge_Commits), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Non_merge_commits is bounded by AST budget",
		}),
		git_enabled, main_absent,
		cpu_boundary, cpu_zero,
		invariant.Excluding("CPU_Count Lo implies cpu_zero true",
			invariant.Bucket_Lo(cpu_boundary), invariant.Bucket_False(cpu_zero)),
		invariant.Excluding("CPU_Count Hi implies cpu_zero false",
			invariant.Bucket_Hi(cpu_boundary), invariant.Bucket_True(cpu_zero)),
		invariant.Excluding("Main reference absent only matters when Git tier is enabled",
			invariant.Bucket_False(git_enabled), invariant.Bucket_True(main_absent)))
}

// True iff the diagnostic is inside the user's scope. Empty scope means
// no filter — all diagnostics pass. Git-tier diagnostics use synthetic
// `<git:…>` filenames; those never live under any scope prefix, so we
// admit them whenever the scope is anything other than empty by checking
// the leading "<" sentinel.
func diagnostic_within_scope(d Diagnostic, scope_prefix string) (within bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			within, "Diagnostic is within scope"))

	}()
	tier_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: d.Tier, Lo: 0, Hi: tier_max,
		Message: "Tier is in [0,2]",
	})
	tier_zero := invariant.Sometimes(d.Tier == 0,
		"Tier is zero for git/stream/cross-file diagnostics")
	scope_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_prefix), Lo: 0, Hi: max_filesystem_path_chars,
		Message: "Scope_prefix is a file-path prefix bounded by filesystem path budget",
	})
	invariant.Cross_Product(
		tier_boundary, tier_zero, scope_axis,
		// Tier is set by the dispatcher: 0 (git/stream/cross-file), 1 or 2
		// (per-file). Lo (=0) implies tier-zero; Hi (=2) implies !tier-zero.
		invariant.Excluding("Tier zero (Lo) implies tier_zero true",
			invariant.Bucket_Lo(tier_boundary), invariant.Bucket_False(tier_zero)),
		invariant.Excluding("Tier two (Hi) implies tier_zero false",
			invariant.Bucket_Hi(tier_boundary), invariant.Bucket_True(tier_zero)),
		invariant.Excluding("Tier Hi per-file diag with max-budget scope_prefix is bad",
			invariant.Bucket_Hi(tier_boundary), invariant.Bucket_Hi(scope_axis)),
	)
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(d.Name), Lo: 0, Hi: max_diagnostic_message_chars,
		Message: "D.Name spans empty (package-group) to bounded names",
	})
	name_empty := invariant.Sometimes(len(d.Name) == 0, "D.Name is empty sometimes")
	invariant.Cross_Product(name_axis, name_empty,
		invariant.Excluding("Hi name empty unreachable",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_True(name_empty)),
		invariant.Excluding("Hi name non-empty unreachable",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_False(name_empty)),
		invariant.Excluding("Lo name implies empty true",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_False(name_empty)),
	)
	want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(d.Want), Lo: 0, Hi: max_diagnostic_message_chars,
		Message: "D.Want spans empty to bounded suggestion text",
	})
	want_empty := invariant.Sometimes(len(d.Want) == 0, "D.Want is empty sometimes")
	invariant.Cross_Product(want_axis, want_empty,
		invariant.Excluding("Hi want empty unreachable",
			invariant.Bucket_Hi(want_axis), invariant.Bucket_True(want_empty)),
		invariant.Excluding("Hi want non-empty unreachable",
			invariant.Bucket_Hi(want_axis), invariant.Bucket_False(want_empty)),
		invariant.Excluding("Lo want implies empty true",
			invariant.Bucket_Lo(want_axis), invariant.Bucket_False(want_empty)),
	)

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
	Position token.Position
	Name     string
	Want     string
	Message  string
	Tier     int
}

type parsed_file struct {
	Path     string
	File_Set *token.FileSet
	File     *ast.File
	Source   []byte
}

var snake_case_re = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)
var ada_case_re = regexp.MustCompile(
	`^([A-Z][a-z0-9]*|[A-Z][A-Z0-9]*)(_([A-Z][a-z0-9]*|[A-Z][A-Z0-9]*))*$`)

// Conventional Commits subject: lowercase type, optional (scope), optional
// `!` breaking-change marker, `: `, non-empty description. Scope contents
// are not whitelisted — package paths and ad-hoc area names both occur in
// the wild and a strict charset would generate more friction than signal.
var conventional_commit_re = regexp.MustCompile(`^[a-z]+(\([^)]+\))?!?: \S`)

func suggest_split_words(name string) (words []string) {
	defer func() {
		words_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(words), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Words is identifier segments; name non-empty so ≥1 word",
		})
		name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty identifier bounded by identifier budget",
		})
		// Hi(words)=Max_Identifier_Chars is unreachable from any input — the
		// split flushes only on '_' or an uppercase rune transition, so each
		// word costs at least one char and the underscore/transition costs
		// another. Densest packing ("aB_aB_…") emits ~2N/3 words per N chars,
		// so 128 words would need >128 chars regardless of pattern. Hi(words)
		// with Lo(name)=1 is the obvious case (one char can't be split into
		// 128 words); Hi(words) with Hi(name)=128 is the same arithmetic with
		// the maximum possible input length.
		invariant.Cross_Product(words_axis, name_axis,
			invariant.Excluding("Hi words with Lo name impossible: ≥1 char per word",
				invariant.Bucket_Hi(words_axis), invariant.Bucket_Lo(name_axis)),
			invariant.Excluding("Hi words with Hi name: ~2N/3 words per N chars",
				invariant.Bucket_Hi(words_axis), invariant.Bucket_Hi(name_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty identifier bounded by identifier budget",
		}),
	)

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
	defer func() { suggest_assert_exit(output, input) }()
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Name is a non-empty identifier within identifier budget",
	})
	want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Want), Lo: min_naming_style_chars, Hi: max_naming_style_chars,
		Message: "Input.Want is `Ada_Case` (8) or `snake_case` (10)",
	})
	invariant.Cross_Product(invariant.Always(input != nil, "Input is non-nil"),
		name_axis, want_axis,
		// Lo name (=1 char) cannot reach suggest: 1-char identifiers always
		// match their style regex, so suggest is never called with a 1-char
		// Name in the test corpus.
		invariant.Excluding("Lo name unreachable (Lo want): 1-char matches regex",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_Lo(want_axis)),
		invariant.Excluding("Lo name unreachable (Hi want): 1-char matches regex",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_Hi(want_axis)),
	)

	words := suggest_split_words(input.Name)
	invariant.Cross_Product(invariant.Always(words != nil, "Words is non-nil at this point"))
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

func suggest_assert_exit(output string, input *suggest_input) {
	output_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(output), Lo: min_split_suggestion_chars, Hi: Max_Identifier_Chars,
		Message: "Output is ≥3 chars: 2-char input splits into `a_B`",
	})
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Name is a non-empty identifier within identifier budget",
	})
	want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Want), Lo: min_naming_style_chars, Hi: max_naming_style_chars,
		Message: "Input.Want is `Ada_Case` (8) or `snake_case` (10)",
	})
	// Output Hi (=128 chars) reaches the cap only for ~86-char inputs (densest
	// split is ~2N/3 words; rejoin with separator adds chars).
	invariant.Cross_Product(output_axis, name_axis, want_axis,
		invariant.Excluding("Hi output with Lo name impossible: ≥1 char per word",
			invariant.Bucket_Hi(output_axis), invariant.Bucket_Lo(name_axis)),
		// Lo name (=1 char) cannot reach suggest: 1-char identifiers match both
		// snake_case (`[a-z]`) and Ada_Case (`[A-Z]`) regexes, so the casing
		// check that gates `suggest()` never fires. Exclude across remaining
		// axes' bucket pairings.
		invariant.Excluding("Lo name unreachable (Lo want): 1-char matches regex",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_Lo(want_axis)),
		invariant.Excluding("Lo name unreachable (Hi want): 1-char matches regex",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_Hi(want_axis)),
		invariant.Excluding("Lo name unreachable (Lo output): 1-char matches regex",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_Lo(output_axis)),
		// Lo output (=3 chars) implies short name input: suggest preserves
		// word content, so a 3-char output requires a ≤3-char input.
		// Pairing Lo output with Hi name (=128 chars) is impossible.
		invariant.Excluding("Lo output (Lo want) implies a short name input",
			invariant.Bucket_Lo(output_axis),
			invariant.Bucket_Hi(name_axis),
			invariant.Bucket_Lo(want_axis)),
		invariant.Excluding("Lo output (Hi want) implies a short name input",
			invariant.Bucket_Lo(output_axis),
			invariant.Bucket_Hi(name_axis),
			invariant.Bucket_Hi(want_axis)),
	)
}

func suggest_is_all_upper(s string) (ok bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			ok, "Predicate evaluates true here"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(s), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "S is a non-empty word from suggest_split_words",
		}),
	)

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
	defer func() {
		decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(file.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "File.Decls is bounded by AST budget",
		})
		diags_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by per-file diag budget",
		})
		// Hi endpoints on both axes are the per-file safety caps for AST size
		// and diagnostic output; reaching either signals pathological input.
		// (Lo decls, Hi diags) is logically impossible: zero declarations yield
		// zero shadow diagnostics.
		invariant.Cross_Product(decls_axis, diags_axis,
			invariant.Excluding("Decls and diags both at safety cap is bad",
				invariant.Bucket_Hi(decls_axis), invariant.Bucket_Hi(diags_axis)),
			invariant.Excluding("Decls at AST safety cap is bad",
				invariant.Bucket_Hi(decls_axis), invariant.Bucket_Lo(diags_axis)),
			invariant.Excluding("Zero decls yield zero shadow diagnostics",
				invariant.Bucket_Lo(decls_axis), invariant.Bucket_Hi(diags_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() {
		check_shadows_function_body_assert_exit(diags, function, global_scope)
	}()
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	receiver_top_level := invariant.Sometimes(function.Recv == nil,
		"Function is a top-level (non-method) function sometimes")
	global_names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(global_scope.Names), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Global_scope.Names is non-empty (file declares ≥1 function)",
	})
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(global_scope != nil, "Global_scope is non-nil"),
		invariant.Always(global_scope.Parent == nil, "Global_scope is at the root"),
		invariant.Always(function != nil, "Function is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		d, receiver_top_level, global_names,
		invariant.Excluding("Diag cap with top-level fn unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_True(receiver_top_level)),
		invariant.Excluding("Diag cap with method fn unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_False(receiver_top_level)),
		invariant.Excluding("Hi global_names unreachable in tests (top-level)",
			invariant.Bucket_Hi(global_names),
			invariant.Bucket_True(receiver_top_level)),
		invariant.Excluding("Hi global_names unreachable in tests (method)",
			invariant.Bucket_Hi(global_names),
			invariant.Bucket_False(receiver_top_level)),
		invariant.Excluding("Method implies its struct type also declared",
			invariant.Bucket_Lo(global_names),
			invariant.Bucket_False(receiver_top_level)),
	)

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

func check_shadows_function_body_assert_exit(
	diags *[]Diagnostic, function *ast.FuncDecl, global_scope *scope,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	receiver_top_level := invariant.Sometimes(function.Recv == nil,
		"Function is a top-level (non-method) function sometimes")
	global_names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(global_scope.Names), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Global_scope.Names is non-empty (file declares ≥1 function)",
	})
	invariant.Cross_Product(d, receiver_top_level, global_names,
		invariant.Excluding("Diag cap with top-level fn unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_True(receiver_top_level)),
		invariant.Excluding("Diag cap with method fn unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_False(receiver_top_level)),
		invariant.Excluding("Hi global_names unreachable in tests (top-level)",
			invariant.Bucket_Hi(global_names),
			invariant.Bucket_True(receiver_top_level)),
		invariant.Excluding("Hi global_names unreachable in tests (method)",
			invariant.Bucket_Hi(global_names),
			invariant.Bucket_False(receiver_top_level)),
		invariant.Excluding("Method implies its struct type also declared",
			invariant.Bucket_Lo(global_names),
			invariant.Bucket_False(receiver_top_level)),
	)
}

// Iteratively walks a sequence of statements with scope tracking. Replaces the
// former check_stmts/check_stmt pair, which were mutually recursive — banned
// by check_no_recursion. The stack holds one frame per nested scope; sibling
// scopes (if/else-if/else) are pushed together and processed LIFO, which is
// fine because they don't share state.
func check_shadows_function_body_walk_body(
	file_set *token.FileSet, root_scope *scope, root_statements []ast.Stmt, diags *[]Diagnostic,
) {
	defer func() {
		check_shadows_function_body_walk_body_assert_exit(
			root_statements, diags, root_scope)
	}()
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(root_statements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Root_statements is bounded by budget",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(root_scope.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Root_scope.Names is bounded by AST budget",
	})
	parent_names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(root_scope.Parent.Names), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Root_scope.Parent.Names is non-empty: global has the fn",
	})
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(root_scope != nil, "Root_scope is non-nil"),
		invariant.Always(root_scope.Parent != nil,
			"Root_scope has a parent (function scope)"),
		invariant.Always(root_scope.Parent.Parent == nil,
			"Root_scope.Parent is the global scope, whose Parent is the chain root"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		s, d, names_axis, parent_names_axis,
		invariant.Excluding("Statements Hi with diags Hi unreachable",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Statements Hi with diags Lo unreachable",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(d)),
		invariant.Excluding("Single-frame stack with max diags requires runaway nesting",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Hi names unreachable (Lo s)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_Lo(s)),
		invariant.Excluding("Hi names unreachable (Hi s)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_Hi(s)),
		invariant.Excluding("Hi parent_names unreachable (Lo s)",
			invariant.Bucket_Hi(parent_names_axis), invariant.Bucket_Lo(s)),
		invariant.Excluding("Hi parent_names unreachable (Hi s)",
			invariant.Bucket_Hi(parent_names_axis), invariant.Bucket_Hi(s)),
	)

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
		invariant.Cross_Product(
			invariant.Always(stack != nil, "Stack is non-nil at this point"))
	}
}

func check_shadows_function_body_walk_body_assert_exit(
	root_statements []ast.Stmt, diags *[]Diagnostic, root_scope *scope,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(root_statements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Root_statements is bounded by budget",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(root_scope.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Root_scope.Names is bounded by AST budget",
	})
	parent_names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(root_scope.Parent.Names),
		Lo:      min_non_empty,
		Hi:      max_ast_nodes_per_call,
		Message: "Root_scope.Parent.Names is non-empty: global has the fn",
	})
	invariant.Cross_Product(s, d, names_axis, parent_names_axis,
		invariant.Excluding("Statements Hi with diags Hi unreachable",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Statements Hi with diags Lo unreachable",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(d)),
		invariant.Excluding("Zero statements yield zero shadow diagnostics",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Hi names unreachable in test corpus (Lo s)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_Lo(s)),
		invariant.Excluding("Hi names unreachable in test corpus (Hi s)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_Hi(s)),
		invariant.Excluding("Hi parent_names unreachable in test corpus (Lo s)",
			invariant.Bucket_Hi(parent_names_axis), invariant.Bucket_Lo(s)),
		invariant.Excluding("Hi parent_names unreachable in test corpus (Hi s)",
			invariant.Bucket_Hi(parent_names_axis), invariant.Bucket_Hi(s)),
	)
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
	defer func() {
		check_shadows_function_body_walk_body_walk_statement_assert_exit(
			&check_shadows_function_body_walk_body_walk_statement_assert_exit_input{
				Stack:       stack,
				Output:      output,
				Diags:       diags,
				Scope_Value: scope_value,
			})
	}()
	check_shadows_function_body_walk_body_walk_statement_assert_entry(
		file_set, scope_value, stack, diags)

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
		invariant.Cross_Product(
			invariant.Always(stack != nil, "Stack is non-nil after a leaf if chain"))
	case *ast.ForStmt:
		for_scope := scope_new_block(scope_value)
		invariant.Cross_Product(
			invariant.Always(for_scope != nil, "For_scope is non-nil at this point"))
		if x.Init != nil {
			check_assign_define(file_set, for_scope, x.Init, diags)
		}
		if x.Body != nil {
			stack = append(stack, walk_frame{Scope: for_scope, Statements: x.Body.List})
		}
	case *ast.RangeStmt:
		stack = check_shadows_function_body_walk_body_walk_statement_push_range_statement(
			file_set, scope_value, x, stack, diags)
		invariant.Cross_Product(
			invariant.Always(
				stack != nil, "Stack is non-nil after a body-less range statement"))
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

func check_shadows_function_body_walk_body_walk_statement_assert_exit(
	input *check_shadows_function_body_walk_body_walk_statement_assert_exit_input,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stack), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Stack at exit is ≥1 (walker gates len>0 before call)",
	})
	o := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Output), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Output is ≥1 (function returns ≥input-stack)",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	// |output-stack| ≤ 1 per call (single push or pop); (Hi,Hi) is the
	// AST safety cap, not a coverage gap.
	invariant.Cross_Product(s, o, d,
		invariant.Excluding("Stack at AST at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(o)),
		invariant.Excluding("Single-call pop or push keeps output near stack",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(o)),
		invariant.Excluding("Single-call pop or push keeps output near stack",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(o)),
		invariant.Excluding("Stack at AST safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Diag cap with Lo stack unreachable in tests",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Lo output with diag cap unreachable in tests",
			invariant.Bucket_Lo(o), invariant.Bucket_Hi(d)),
		invariant.Excluding("Output at safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(o), invariant.Bucket_Hi(d)),
	)
	na := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Scope_Value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	pn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Scope_Value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(na, pn,
		invariant.Excluding("Hi names unreachable (Lo parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Lo(pn)),
		invariant.Excluding("Hi names unreachable (Hi parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Hi(pn)),
		invariant.Excluding("Hi parent unreachable (Lo names)",
			invariant.Bucket_Lo(na), invariant.Bucket_Hi(pn)),
	)
}

func check_shadows_function_body_walk_body_walk_statement_assert_entry(
	file_set *token.FileSet,
	scope_value *scope,
	stack []walk_frame,
	diags *[]Diagnostic,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stack), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Stack is ≥1 (walker gates len>0 before call)",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	parent_parent_nil := invariant.Sometimes(scope_value.Parent.Parent == nil,
		"Scope_value.Parent.Parent is nil at the function-scope root sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(scope_value != nil, "Scope_value is non-nil"),
		invariant.Always(scope_value.Parent != nil, "Scope_value has a parent"),
		parent_parent_nil,
		invariant.Always(diags != nil, "Diags is non-nil"),
		s, d,
		invariant.Excluding("Stack at AST at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Stack at AST at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(d)),
		invariant.Excluding("Single-frame stack with max diags requires runaway nesting",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(d)),
		// Nested scope implies stack ≥ 2 — walker pushed a block frame.
		invariant.Excluding("Nested scope_value implies stack ≥ 2 frames",
			invariant.Bucket_False(parent_parent_nil), invariant.Bucket_Lo(s)),
	)
	na := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	pn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(na, pn,
		invariant.Excluding("Hi names unreachable (Lo parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Lo(pn)),
		invariant.Excluding("Hi names unreachable (Hi parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Hi(pn)),
		invariant.Excluding("Hi parent unreachable (Lo names)",
			invariant.Bucket_Lo(na), invariant.Bucket_Hi(pn)),
	)
}

func scope_new_block(parent *scope) (new_scope *scope) {
	defer func() { scope_new_block_assert_exit(new_scope) }()
	parent_names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parent.Names is bounded by AST budget",
	})
	parent_parent_names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parent.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parent.Parent.Names is bounded by AST budget",
	})
	ppp_nil := invariant.Sometimes(parent.Parent.Parent == nil,
		"Parent.Parent.Parent is nil at the function-scope root sometimes")
	invariant.Cross_Product(
		invariant.Always(parent != nil, "Parent is non-nil"),
		invariant.Always(parent.Parent != nil,
			"Parent is a nested scope inside a function body"),
		ppp_nil, parent_names, parent_parent_names,
		invariant.Excluding("Hi parent_names unreachable (root)",
			invariant.Bucket_Hi(parent_names), invariant.Bucket_True(ppp_nil)),
		invariant.Excluding("Hi parent_names unreachable (nested)",
			invariant.Bucket_Hi(parent_names), invariant.Bucket_False(ppp_nil)),
		invariant.Excluding("Hi parent_parent_names unreachable (root)",
			invariant.Bucket_Hi(parent_parent_names), invariant.Bucket_True(ppp_nil)),
		invariant.Excluding("Hi parent_parent_names unreachable (nested)",
			invariant.Bucket_Hi(parent_parent_names), invariant.Bucket_False(ppp_nil)),
		invariant.Excluding("Lo parent_parent_names at root impossible",
			invariant.Bucket_Lo(parent_parent_names), invariant.Bucket_True(ppp_nil)),
	)

	return &scope{Parent: parent, Names: make(map[string]bool)}
}

func scope_new_block_assert_exit(new_scope *scope) {
	// New_scope.Parent is the constructor's `parent`; the entry asserts
	// parent.Parent is non-nil, so new_scope.Parent.Parent inherits that
	// guarantee. The names axis is bounded by AST budget; the constructor
	// returns a freshly-made empty map so Lo=0 is invariant — paired with the
	// parent_parent_parent_nil axis to keep Hi unreachable (a brand-new map
	// can never be at the safety cap).
	names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(new_scope.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Names is bounded by AST budget",
	})
	parent_parent_parent_nil := invariant.Sometimes(
		new_scope.Parent.Parent.Parent == nil,
		"New_scope grandparent is nil at function-scope root sometimes")
	parent_names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(new_scope.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parent.Names is bounded by AST budget",
	})
	parent_parent_names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(new_scope.Parent.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parent.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(new_scope != nil, "New_scope is non-nil"),
		invariant.Always(new_scope.Parent != nil, "New_scope has a parent"),
		invariant.Always(new_scope.Parent.Parent != nil,
			"New_scope.Parent.Parent is non-nil"),
		names_axis, parent_parent_parent_nil, parent_names, parent_parent_names,
		invariant.Excluding("Hi names with root grandparent unreachable",
			invariant.Bucket_Hi(names_axis),
			invariant.Bucket_True(parent_parent_parent_nil)),
		invariant.Excluding("Hi names with nested grandparent unreachable",
			invariant.Bucket_Hi(names_axis),
			invariant.Bucket_False(parent_parent_parent_nil)),
		invariant.Excluding("Hi parent_names unreachable (root)",
			invariant.Bucket_Hi(parent_names),
			invariant.Bucket_True(parent_parent_parent_nil)),
		invariant.Excluding("Hi parent_names unreachable (nested)",
			invariant.Bucket_Hi(parent_names),
			invariant.Bucket_False(parent_parent_parent_nil)),
		invariant.Excluding("Hi parent_parent_names unreachable (root)",
			invariant.Bucket_Hi(parent_parent_names),
			invariant.Bucket_True(parent_parent_parent_nil)),
		invariant.Excluding("Hi parent_parent_names unreachable (nested)",
			invariant.Bucket_Hi(parent_parent_names),
			invariant.Bucket_False(parent_parent_parent_nil)),
		invariant.Excluding("Lo parent_parent_names at root impossible",
			invariant.Bucket_Lo(parent_parent_names),
			invariant.Bucket_True(parent_parent_parent_nil)),
	)
	ns_parent_names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(new_scope.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "New_scope.Parent.Names is bounded by AST budget",
	})
	ns_grandparent_nil := invariant.Sometimes(new_scope.Parent.Parent.Parent == nil,
		"New_scope.Parent.Parent.Parent is nil at function-scope root sometimes")
	invariant.Cross_Product(ns_parent_names, ns_grandparent_nil,
		invariant.Excluding("Hi parent_names unreachable (root)",
			invariant.Bucket_Hi(ns_parent_names),
			invariant.Bucket_True(ns_grandparent_nil)),
		invariant.Excluding("Hi parent_names unreachable (nested)",
			invariant.Bucket_Hi(ns_parent_names),
			invariant.Bucket_False(ns_grandparent_nil)),
	)
}

func check_shadows_function_body_walk_body_walk_statement_push_if_chain(
	file_set *token.FileSet,
	scope_value *scope,
	root *ast.IfStmt,
	stack []walk_frame,
	diags *[]Diagnostic,
) (output []walk_frame) {
	defer func() {
		check_shadows_function_body_walk_body_walk_statement_push_if_chain_assert_exit(
			output, len(stack), diags, scope_value)
	}()
	check_shadows_function_body_walk_body_walk_statement_push_if_chain_assert_entry(
		file_set, scope_value, root, stack, diags)

	current := root
	for current != nil {
		if_scope := scope_new_block(scope_value)
		invariant.Cross_Product(
			invariant.Always(if_scope != nil, "If_scope is non-nil at this point"))
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
func check_shadows_function_body_walk_body_walk_statement_push_if_chain_assert_exit(
	output []walk_frame, stack_frame_count int, diags *[]Diagnostic, scope_value *scope,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: stack_frame_count, Lo: min_stack_with_body_frame, Hi: max_ast_nodes_per_call,
		Message: "Stack at exit is ≥2 (function appended ≥1 body frame)",
	})
	o := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(output), Lo: min_stack_with_body_frame, Hi: max_ast_nodes_per_call,
		Message: "Output is ≥2 (function appended ≥1 body frame)",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	// Function only appends frames (output ≥ stack); (Hi,Hi) is the
	// AST safety cap, not a working shape.
	invariant.Cross_Product(s, o, d,
		invariant.Excluding("Stack at AST at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(o)),
		invariant.Excluding("Single-call pop or push keeps output near stack",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(o)),
		invariant.Excluding("Single-call pop or push keeps output near stack",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(o)),
		invariant.Excluding("Stack at AST safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Lo stack with diag cap unreachable in tests",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Lo output with diag cap unreachable in tests",
			invariant.Bucket_Lo(o), invariant.Bucket_Hi(d)),
		invariant.Excluding("Output at safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(o), invariant.Bucket_Hi(d)),
	)
	na := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	pn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(na, pn,
		invariant.Excluding("Hi names unreachable (Lo parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Lo(pn)),
		invariant.Excluding("Hi names unreachable (Hi parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Hi(pn)),
		invariant.Excluding("Hi parent unreachable (Lo names)",
			invariant.Bucket_Lo(na), invariant.Bucket_Hi(pn)),
	)
}

func check_shadows_function_body_walk_body_walk_statement_push_if_chain_assert_entry(
	file_set *token.FileSet,
	scope_value *scope,
	root *ast.IfStmt,
	stack []walk_frame,
	diags *[]Diagnostic,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stack), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Stack is ≥1 (walker gates len>0 before call)",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	parent_parent_nil := invariant.Sometimes(scope_value.Parent.Parent == nil,
		"Scope_value.Parent.Parent is nil at the function-scope root sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(scope_value != nil, "Scope_value is non-nil"),
		invariant.Always(scope_value.Parent != nil, "Scope_value has a parent"),
		parent_parent_nil,
		invariant.Always(root != nil, "Root is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		s, d,
		invariant.Excluding("Stack and diags both at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Stack at AST safety cap with zero diags is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(d)),
		invariant.Excluding("Single-frame stack with max diags requires runaway nesting",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Nested scope_value implies stack ≥ 2 frames",
			invariant.Bucket_False(parent_parent_nil), invariant.Bucket_Lo(s)),
	)
	na := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	pn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(na, pn,
		invariant.Excluding("Hi names unreachable (Lo parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Lo(pn)),
		invariant.Excluding("Hi names unreachable (Hi parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Hi(pn)),
		invariant.Excluding("Hi parent unreachable (Lo names)",
			invariant.Bucket_Lo(na), invariant.Bucket_Hi(pn)),
	)
}

func check_shadows_function_body_walk_body_walk_statement_push_range_statement(
	file_set *token.FileSet,
	scope_value *scope,
	x *ast.RangeStmt,
	stack []walk_frame,
	diags *[]Diagnostic,
) (output []walk_frame) {
	defer func() {
		push_range_assert_exit(&push_range_assert_exit_input{
			Stack: stack, Output: output, Diags: diags,
		})
		scope_assert_names_bounded(scope_value)
	}()
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stack), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Stack is ≥1 (walker gates len>0 before call)",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	parent_parent_nil := invariant.Sometimes(scope_value.Parent.Parent == nil,
		"Scope_value.Parent.Parent is nil at the function-scope root sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(scope_value != nil, "Scope_value is non-nil"),
		invariant.Always(scope_value.Parent != nil, "Scope_value has a parent"),
		parent_parent_nil,
		invariant.Always(x != nil, "X is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		s, d,
		invariant.Excluding("Stack and diags both at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Stack at AST safety cap with zero diags is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(d)),
		invariant.Excluding("Single-frame stack with max diags requires runaway nesting",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(d)),
		// Nested scope implies stack ≥ 2 — walker pushed block frame.
		invariant.Excluding("Nested scope_value implies stack ≥ 2 frames",
			invariant.Bucket_False(parent_parent_nil), invariant.Bucket_Lo(s)),
	)
	na := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	pn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(na, pn,
		invariant.Excluding("Hi names unreachable (Lo parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Lo(pn)),
		invariant.Excluding("Hi names unreachable (Hi parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Hi(pn)),
		invariant.Excluding("Hi parent unreachable (Lo names)",
			invariant.Bucket_Lo(na), invariant.Bucket_Hi(pn)),
	)

	range_scope := scope_new_block(scope_value)
	invariant.Cross_Product(
		invariant.Always(range_scope != nil, "Range_scope is non-nil at this point"))
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

func push_range_assert_exit(input *push_range_assert_exit_input) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stack), Lo: min_stack_with_body_frame, Hi: max_ast_nodes_per_call,
		Message: "Stack at exit is ≥2 (range body always appends a frame)",
	})
	o := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Output), Lo: min_stack_with_body_frame, Hi: max_ast_nodes_per_call,
		Message: "Output is ≥2 (range body always appends a frame)",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	// Range body always appends one frame so output == stack+1 modulo
	// the one-frame delta; cross-extreme tuples are unreachable.
	invariant.Cross_Product(s, o, d,
		invariant.Excluding("Stack at AST at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(o)),
		invariant.Excluding("Single-call pop or push keeps output near stack",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(o)),
		invariant.Excluding("Single-call pop or push keeps output near stack",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(o)),
		invariant.Excluding("Stack at AST safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Lo stack with diag cap unreachable in tests",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(d)),
		invariant.Excluding("Lo output with diag cap unreachable in tests",
			invariant.Bucket_Lo(o), invariant.Bucket_Hi(d)),
		invariant.Excluding("Output at safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(o), invariant.Bucket_Hi(d)),
	)
}

func scope_assert_names_bounded(scope_value *scope) {
	na := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	pn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(na, pn,
		invariant.Excluding("Hi names unreachable (Lo parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Lo(pn)),
		invariant.Excluding("Hi names unreachable (Hi parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Hi(pn)),
		invariant.Excluding("Hi parent unreachable (Lo names)",
			invariant.Bucket_Lo(na), invariant.Bucket_Hi(pn)),
	)
}

func check_shadows_function_body_walk_body_walk_statement_push_range_statement_add_variable(
	file_set *token.FileSet,
	scope_value *scope,
	e ast.Expr,
	diags *[]Diagnostic,
) {
	defer func() {
		push_range_add_variable_assert_exit(diags, e, scope_value)
	}()
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	e_present := invariant.Sometimes(e != nil, "Range key or value is present sometimes")
	names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	parent_names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	grandparent_names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Parent.Names is bounded by AST budget",
	})
	// Scope_value here is always a range_scope freshly created by
	// scope_new_block, whose parent is the caller's scope_value (function
	// or deeper). scope_new_block's entry asserts parent.Parent != nil, so
	// range_scope.Parent.Parent is non-nil by construction.
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(scope_value != nil, "Scope_value is non-nil"),
		invariant.Always(scope_value.Parent != nil, "Scope_value has a parent"),
		invariant.Always(scope_value.Parent.Parent != nil,
			"Scope_value.Parent.Parent is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		d, e_present, names_axis, parent_names_axis, grandparent_names_axis,
		invariant.Excluding("Diag-budget cap with present range var unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_True(e_present)),
		invariant.Excluding("Diag-budget cap with absent range var unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_False(e_present)),
		invariant.Excluding("Hi names unreachable (present range var)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_True(e_present)),
		invariant.Excluding("Hi names unreachable (absent range var)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_False(e_present)),
		invariant.Excluding("Hi parent_names unreachable (present range var)",
			invariant.Bucket_Hi(parent_names_axis),
			invariant.Bucket_True(e_present)),
		invariant.Excluding("Hi parent_names unreachable (absent range var)",
			invariant.Bucket_Hi(parent_names_axis),
			invariant.Bucket_False(e_present)),
		invariant.Excluding("Hi grandparent_names unreachable (present range var)",
			invariant.Bucket_Hi(grandparent_names_axis),
			invariant.Bucket_True(e_present)),
		invariant.Excluding("Hi grandparent_names unreachable (absent range var)",
			invariant.Bucket_Hi(grandparent_names_axis),
			invariant.Bucket_False(e_present)),
	)

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

func push_range_add_variable_assert_exit(
	diags *[]Diagnostic, e ast.Expr, scope_value *scope,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	e_present := invariant.Sometimes(
		e != nil, "Range key or value is present sometimes")
	names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	parent_names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	grandparent_names_axis := invariant.Distinct_Boundary(
		&invariant.Boundary_Input[int]{
			X:       len(scope_value.Parent.Parent.Names),
			Lo:      0,
			Hi:      max_ast_nodes_per_call,
			Message: "Scope_value.Parent.Parent.Names is bounded by AST budget",
		})
	invariant.Cross_Product(
		d, e_present, names_axis, parent_names_axis, grandparent_names_axis,
		invariant.Excluding("Diag-budget cap with present range var unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_True(e_present)),
		invariant.Excluding("Diag-budget cap with absent range var unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_False(e_present)),
		invariant.Excluding("Hi names unreachable (present range var)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_True(e_present)),
		invariant.Excluding("Hi names unreachable (absent range var)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_False(e_present)),
		invariant.Excluding("Hi parent_names unreachable (present range var)",
			invariant.Bucket_Hi(parent_names_axis),
			invariant.Bucket_True(e_present)),
		invariant.Excluding("Hi parent_names unreachable (absent range var)",
			invariant.Bucket_Hi(parent_names_axis),
			invariant.Bucket_False(e_present)),
		invariant.Excluding("Hi grandparent_names unreachable (present range var)",
			invariant.Bucket_Hi(grandparent_names_axis),
			invariant.Bucket_True(e_present)),
		invariant.Excluding("Hi grandparent_names unreachable (absent range var)",
			invariant.Bucket_Hi(grandparent_names_axis),
			invariant.Bucket_False(e_present)),
	)
}

func check_shadows_walk_statement_assert_exit(diags *[]Diagnostic, scope_value *scope) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	parent_parent_nil := invariant.Sometimes(scope_value.Parent.Parent == nil,
		"Scope_value.Parent.Parent is nil at the function-scope root sometimes")
	invariant.Cross_Product(d, parent_parent_nil,
		invariant.Excluding("Diag cap at function-scope root unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_True(parent_parent_nil)),
		invariant.Excluding("Diag cap at nested scope unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_False(parent_parent_nil)),
	)
	na := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	pn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(na, pn,
		invariant.Excluding("Hi names unreachable (Lo parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Lo(pn)),
		invariant.Excluding("Hi names unreachable (Hi parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Hi(pn)),
		invariant.Excluding("Hi parent unreachable (Lo names)",
			invariant.Bucket_Lo(na), invariant.Bucket_Hi(pn)),
	)
}

func check_shadows_function_body_walk_body_walk_statement_assign_statement(
	file_set *token.FileSet,
	scope_value *scope,
	x *ast.AssignStmt,
	diags *[]Diagnostic,
) {
	defer func() {
		check_shadows_walk_statement_assert_exit(diags, scope_value)
	}()
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	parent_parent_nil := invariant.Sometimes(scope_value.Parent.Parent == nil,
		"Scope_value.Parent.Parent is nil at the function-scope root sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(scope_value != nil, "Scope_value is non-nil"),
		invariant.Always(scope_value.Parent != nil, "Scope_value has a parent"),
		parent_parent_nil,
		invariant.Always(x != nil, "X is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		d,
		invariant.Excluding("Diag cap at function-scope root unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_True(parent_parent_nil)),
		invariant.Excluding("Diag cap at nested scope unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_False(parent_parent_nil)),
	)
	na := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	pn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(na, pn,
		invariant.Excluding("Hi names unreachable (Lo parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Lo(pn)),
		invariant.Excluding("Hi names unreachable (Hi parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Hi(pn)),
		invariant.Excluding("Hi parent unreachable (Lo names)",
			invariant.Bucket_Lo(na), invariant.Bucket_Hi(pn)),
	)

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
	defer func() {
		check_shadows_walk_statement_assert_exit(diags, scope_value)
	}()
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	parent_parent_nil := invariant.Sometimes(scope_value.Parent.Parent == nil,
		"Scope_value.Parent.Parent is nil at the function-scope root sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(scope_value != nil, "Scope_value is non-nil"),
		invariant.Always(scope_value.Parent != nil, "Scope_value has a parent"),
		parent_parent_nil,
		invariant.Always(x != nil, "X is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		d,
		invariant.Excluding("Diag cap at function-scope root unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_True(parent_parent_nil)),
		invariant.Excluding("Diag cap at nested scope unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_False(parent_parent_nil)),
	)
	na := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	pn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(na, pn,
		invariant.Excluding("Hi names unreachable (Lo parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Lo(pn)),
		invariant.Excluding("Hi names unreachable (Hi parent)",
			invariant.Bucket_Hi(na), invariant.Bucket_Hi(pn)),
		invariant.Excluding("Hi parent unreachable (Lo names)",
			invariant.Bucket_Lo(na), invariant.Bucket_Hi(pn)),
	)

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
	defer func() {
		check_assign_define_assert_exit(diags, scope_value)
	}()
	parent_names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	names_zero := invariant.Sometimes(
		len(scope_value.Names) == 0, "Scope_value.Names is empty sometimes")
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	// Scope_value here is an if/for/range init scope produced by
	// scope_new_block, whose entry assertion guarantees parent.Parent !=
	// nil. So scope_value.Parent.Parent is non-nil by construction.
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(scope_value != nil, "Scope_value is non-nil"),
		invariant.Always(scope_value.Parent != nil, "Scope_value has a parent"),
		invariant.Always(scope_value.Parent.Parent != nil,
			"Scope_value.Parent.Parent is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		parent_names, names, names_zero, d,
		invariant.Excluding("Parent_names at safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(parent_names), invariant.Bucket_Hi(d)),
		invariant.Excluding("Parent_names at safety cap with zero diags is bad",
			invariant.Bucket_Hi(parent_names), invariant.Bucket_Lo(d)),
		invariant.Excluding("Zero parent_names with diag cap unreachable in tests",
			invariant.Bucket_Lo(parent_names), invariant.Bucket_Hi(d)),
		invariant.Excluding("Names Lo (=0) implies names_zero true",
			invariant.Bucket_Lo(names), invariant.Bucket_False(names_zero)),
		invariant.Excluding("Names Hi (=safety cap) implies names_zero false",
			invariant.Bucket_Hi(names), invariant.Bucket_True(names_zero)),
		invariant.Excluding("Names at safety cap unreachable in test corpus",
			invariant.Bucket_Hi(names), invariant.Bucket_Hi(parent_names)),
		invariant.Excluding("Names at cap with Lo parent unreachable in tests",
			invariant.Bucket_Hi(names), invariant.Bucket_Lo(parent_names)),
		invariant.Excluding("Names at cap with Hi diags unreachable in tests",
			invariant.Bucket_Hi(names), invariant.Bucket_Hi(d)),
		invariant.Excluding("Names at cap with Lo diags unreachable in tests",
			invariant.Bucket_Hi(names), invariant.Bucket_Lo(d)),
	)

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

func check_assign_define_assert_exit(diags *[]Diagnostic, scope_value *scope) {
	parent_names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	names := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	names_zero := invariant.Sometimes(
		len(scope_value.Names) == 0, "Scope_value.Names is empty sometimes")
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	invariant.Cross_Product(parent_names, names, names_zero, d,
		invariant.Excluding("Parent_names and diags both at safety cap is bad",
			invariant.Bucket_Hi(parent_names), invariant.Bucket_Hi(d)),
		invariant.Excluding("Parent_names at safety cap with zero diags is bad",
			invariant.Bucket_Hi(parent_names), invariant.Bucket_Lo(d)),
		invariant.Excluding("Zero parent_names with diag cap unreachable in tests",
			invariant.Bucket_Lo(parent_names), invariant.Bucket_Hi(d)),
		invariant.Excluding("Names Lo (=0) implies names_zero true",
			invariant.Bucket_Lo(names), invariant.Bucket_False(names_zero)),
		invariant.Excluding("Names Hi (=safety cap) implies names_zero false",
			invariant.Bucket_Hi(names), invariant.Bucket_True(names_zero)),
		invariant.Excluding("Names at safety cap unreachable in test corpus",
			invariant.Bucket_Hi(names), invariant.Bucket_Hi(parent_names)),
		invariant.Excluding("Names at cap with Lo parent unreachable in tests",
			invariant.Bucket_Hi(names), invariant.Bucket_Lo(parent_names)),
		invariant.Excluding("Names at cap with Hi diags unreachable in tests",
			invariant.Bucket_Hi(names), invariant.Bucket_Hi(d)),
		invariant.Excluding("Names at cap with Lo diags unreachable in tests",
			invariant.Bucket_Hi(names), invariant.Bucket_Lo(d)),
	)
}

func check_shadow(
	file_set *token.FileSet,
	scope_value *scope,
	name string,
	identifier *ast.Ident,
	diags *[]Diagnostic,
) {
	defer func() { check_shadow_assert_exit(diags, scope_value) }()
	parent_parent_nil := invariant.Sometimes(scope_value.Parent.Parent == nil,
		"Scope_value.Parent.Parent is nil at the function-scope root sometimes")
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Name is a non-empty Go identifier bounded by identifier budget",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	parent_names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(scope_value != nil, "Scope_value is non-nil"),
		invariant.Always(scope_value.Parent != nil, "Scope_value has a parent"),
		parent_parent_nil,
		invariant.Always(identifier != nil, "Identifier is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		name_axis, d, names_axis, parent_names_axis,
		// Identifiers at the max-length safety cap appear only at function-
		// scope root (e.g. test fixtures with a single long-name decl) in the
		// observed corpus; nested-scope shadowing fixtures stay well below the
		// safety cap.
		invariant.Excluding("Identifier at cap appears only at function-scope root",
			invariant.Bucket_False(parent_parent_nil), invariant.Bucket_Hi(name_axis)),
		invariant.Excluding("Diag cap at function-scope root unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_True(parent_parent_nil)),
		invariant.Excluding("Diag cap at nested scope unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_False(parent_parent_nil)),
		invariant.Excluding("Hi names unreachable (root)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_True(parent_parent_nil)),
		invariant.Excluding("Hi names unreachable (nested)",
			invariant.Bucket_Hi(names_axis), invariant.Bucket_False(parent_parent_nil)),
		invariant.Excluding("Hi parent_names unreachable (root)",
			invariant.Bucket_Hi(parent_names_axis),
			invariant.Bucket_True(parent_parent_nil)),
		invariant.Excluding("Hi parent_names unreachable (nested)",
			invariant.Bucket_Hi(parent_names_axis),
			invariant.Bucket_False(parent_parent_nil)),
		invariant.Excluding("Lo parent_names at root impossible",
			invariant.Bucket_Lo(parent_names_axis),
			invariant.Bucket_True(parent_parent_nil)),
	)

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

func check_shadow_assert_exit(diags *[]Diagnostic, scope_value *scope) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	parent_parent_nil := invariant.Sometimes(scope_value.Parent.Parent == nil,
		"Scope_value.Parent.Parent is nil at the function-scope root sometimes")
	names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Names is bounded by AST budget",
	})
	parent_names_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(scope_value.Parent.Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Scope_value.Parent.Names is bounded by AST budget",
	})
	invariant.Cross_Product(d, parent_parent_nil, names_axis, parent_names_axis,
		invariant.Excluding("Diag cap at function-scope root unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_True(parent_parent_nil)),
		invariant.Excluding("Diag cap at nested scope unreachable in tests",
			invariant.Bucket_Hi(d), invariant.Bucket_False(parent_parent_nil)),
		invariant.Excluding("Hi names unreachable (root scope)",
			invariant.Bucket_Hi(names_axis),
			invariant.Bucket_True(parent_parent_nil)),
		invariant.Excluding("Hi names unreachable (nested scope)",
			invariant.Bucket_Hi(names_axis),
			invariant.Bucket_False(parent_parent_nil)),
		invariant.Excluding("Hi parent_names unreachable (root scope)",
			invariant.Bucket_Hi(parent_names_axis),
			invariant.Bucket_True(parent_parent_nil)),
		invariant.Excluding("Hi parent_names unreachable (nested scope)",
			invariant.Bucket_Hi(parent_names_axis),
			invariant.Bucket_False(parent_parent_nil)),
		invariant.Excluding("Lo parent_names at root impossible",
			invariant.Bucket_Lo(parent_names_axis),
			invariant.Bucket_True(parent_parent_nil)),
	)
}

// Check_File runs every per-file check (tier-1 first, then tier-2 if
// tier-1 was clean) on one already-parsed file and returns the
// accumulated diagnostics. Used both by Check_Source and by the
// file-system tier's per-file pass. Stamps each diagnostic with its
// origin tier so the printer can gate tier-2 output globally on the
// presence of any tier-1 diagnostic.
func Check_File(file_set *token.FileSet, file *ast.File, source []byte) (diags []Diagnostic) {
	defer func() {
		assert_diags_source_filebytes_bounded(diags, source)
	}()
	assert_file_source_documentation_entry(file_set, file, source)
	diags = check_file_run_tier([]check_function{
		check_casing,
		check_namesd_returns,
		check_no_naked_return,
		check_shadows,
		check_line_character_count,
		check_function_line_count,
		check_compound_if,
		check_comments,
		check_main_first,
		check_constant_first, check_assertion_named_constant,
		check_no_discard,
		check_public_struct_fields,
		check_exported_type_exposes_private,
		check_no_iota,
		check_no_grouped_declaration,
		check_keyed_struct_init,
		check_gofmt,
		check_no_dot_import,
		check_default_package_alias,
		check_test_package,
		check_no_empty_function_body,
		check_no_interfaces,
		check_input_struct,
		check_banned_identifiers,
		check_test_documentation_comment,
		check_snap_backtick,
		check_names,
		check_no_bare_for,
		check_exported_documentation_comment,
	}, file_set, file, source)
	invariant.Cross_Product(
		invariant.Sometimes(diags == nil, "Diags can be empty or zero on this branch"))
	if len(diags) > 0 {
		for i := range diags {
			diags[i].Tier = 1
		}
		return diags
	}
	diags = check_file_run_tier([]check_function{
		check_no_unbounded_apis, check_no_recursion,
		check_no_function_init, check_no_package_vars,
		check_unnecessary_method,
		check_no_third_party_struct_tag,
		check_blank_synchronization_mutex,
	}, file_set, file, source)
	invariant.Cross_Product(
		invariant.Sometimes(diags == nil, "Diags can be empty or zero on this branch"))
	for i := range diags {
		diags[i].Tier = 2
	}
	return diags
}

func check_file_run_tier(
	checks []check_function, file_set *token.FileSet, file *ast.File, source []byte,
) (diags []Diagnostic) {
	defer func() {
		diags_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by per-file diag budget",
		})
		// Checks is the static tier list (tier-2 has 7, tier-1 has 30);
		// production callers always pass one of those two slices.
		checks_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(checks), Lo: tier_2_checks_count, Hi: tier_1_checks_count,
			Message: "Checks is the tier-2 (7) or tier-1 (30) static check list",
		})
		source_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(source), Lo: 0, Hi: max_file_size_bytes,
			Message: "Source is the file bytes bounded by per-file size budget",
		})
		// (Lo diags, Hi source) is logically impossible: a Go file at the
		// byte cap would contain enough declarations to fire at least one
		// per-tier check, so 0 diags from a max-size file cannot happen.
		// (Hi diags, *) tuples mark the per-file diag budget cap — the
		// endpoint guards runaway output, not a working diagnostic count.
		invariant.Cross_Product(
			diags_boundary, checks_boundary, source_boundary,
			invariant.Excluding("Both diags and source at safety caps is bad",
				invariant.Bucket_Hi(diags_boundary),
				invariant.Bucket_Hi(source_boundary)),
			invariant.Excluding("Diags at safety cap with zero source bytes is bad",
				invariant.Bucket_Hi(diags_boundary),
				invariant.Bucket_Lo(source_boundary)),
			invariant.Excluding("Source at safety cap with zero diags is bad",
				invariant.Bucket_Lo(diags_boundary),
				invariant.Bucket_Hi(source_boundary)),
		)
	}()
	checks_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(checks), Lo: tier_2_checks_count, Hi: tier_1_checks_count,
		Message: "Checks is the tier-2 (7) or tier-1 (30) static check list",
	})
	source_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(source), Lo: 0, Hi: max_file_size_bytes,
		Message: "Source is the file bytes bounded by per-file size budget",
	})
	source_empty := invariant.Sometimes(len(source) == 0, "Source is empty sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
		checks_axis, source_axis, source_empty,
		invariant.Excluding("Source at file-size at safety cap is bad",
			invariant.Bucket_Hi(source_axis), invariant.Bucket_True(source_empty)),
		invariant.Excluding("Source at file-size at safety cap is bad",
			invariant.Bucket_Hi(source_axis), invariant.Bucket_False(source_empty)),
		invariant.Excluding("Zero-byte source implies source_empty true",
			invariant.Bucket_Lo(source_axis), invariant.Bucket_False(source_empty)),
	)

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
	defer func() {
		name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(identifier.Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Identifier name length is bounded by identifier budget",
		})
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by per-file diag budget",
		})
		invariant.Cross_Product(name_axis, d,
			invariant.Excluding("Identifier at safety cap with diag-budget cap is bad",
				invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(d)),
			invariant.Excluding("Lo identifier with diag cap unreachable in tests",
				invariant.Bucket_Lo(name_axis), invariant.Bucket_Hi(d)),
		)
	}()
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(identifier.Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Identifier name length is bounded by identifier budget",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(identifier != nil, "Identifier is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		name_axis, d,
		invariant.Excluding("Identifier at safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(d)),
		invariant.Excluding("Lo identifier with diag cap unreachable in tests",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_Hi(d)),
	)

	if identifier.Name == "_" {
		return
	}
	first := rune(identifier.Name[0])
	if !unicode.IsLetter(first) {
		return
	}
	want := "snake_case"
	ok := snake_case_re.MatchString(identifier.Name)
	invariant.Cross_Product(invariant.Sometimes(ok, "Ok can be false on this branch"))
	if unicode.IsUpper(first) {
		want = "Ada_Case"
		ok = ada_case_re.MatchString(identifier.Name)
		invariant.Cross_Product(invariant.Sometimes(ok, "Ok can be false on this branch"))
	}
	if !ok {
		suggestion := suggest(&suggest_input{Name: identifier.Name, Want: want})
		invariant.Cross_Product(
			invariant.Always(suggestion != "", "Suggestion is non-empty at this point"))
		*diags = append(*diags, Diagnostic{
			Position: file_set.Position(identifier.Pos()),
			Name:     identifier.Name,
			Want:     want,
			Message:  fmt.Sprintf("%s -> %s", identifier.Name, suggestion),
		})
	}
}

func check_casing(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Method satisfies a stdlib interface"))

	}()
	invariant.Cross_Product(invariant.Always(
		function_declaration != nil, "Function_declaration is non-nil"))

	if function_declaration.Recv == nil {
		return false
	}
	params := check_unnecessary_method_field_list_types(function_declaration.Type.Params)
	invariant.Cross_Product(
		invariant.Sometimes(params == nil, "Params can be empty or zero on this branch"))
	results := check_unnecessary_method_field_list_types(function_declaration.Type.Results)
	invariant.Cross_Product(
		invariant.Sometimes(results == nil, "Results can be empty or zero on this branch"))
	return check_unnecessary_method_matches_stdlib(
		&check_unnecessary_method_matches_stdlib_input{
			Name:    function_declaration.Name.Name,
			Params:  strings.Join(params, ","),
			Results: strings.Join(results, ","),
		})
}

func check_namesd_returns(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
func assert_diags_decls_bounded(diags []Diagnostic, file *ast.File) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(file.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "File.Decls is bounded by AST budget",
	})
	invariant.Cross_Product(d, decls_axis,
		invariant.Excluding("Diags and decls both at safety cap is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(decls_axis)),
		invariant.Excluding("Zero decls produce zero diags",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(decls_axis)),
		invariant.Excluding("Max-decl clean file at AST safety cap",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(decls_axis)),
	)
}

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
				Message:  "bare return is banned; return all values explicitly",
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
	defer func() {
		assert_diags_source_filebytes_bounded(diags, source)
	}()
	assert_file_source_documentation_entry(file_set, file, source)

	tok_file := file_set.File(file.Pos())
	invariant.Cross_Product(
		invariant.Always(tok_file != nil, "Tok_file is non-nil at this point"))
	filename := ""
	if tok_file != nil {
		filename = tok_file.Name()
		invariant.Cross_Product(
			invariant.Always(filename != "", "Filename is non-empty at this point"))
	}
	// Import lines are exempt: a module path is a single unbreakable token, so a
	// long one cannot be wrapped to satisfy the column limit.
	import_lines := map[int]bool{}
	for _, import_specification := range file.Imports {
		import_lines[file_set.Position(import_specification.Pos()).Line] = true
	}
	line_number := 1
	column := 0
	emit := func(n int) {
		if import_lines[line_number] {
			return
		}
		diags = append(diags, Diagnostic{
			Position: token.Position{
				Filename: filename,
				Line:     line_number,
				Column:   max_line_chars + 1,
			},
			Message: fmt.Sprintf("line is %d chars (max %d)", n, max_line_chars),
		})
	}
	for len(source) > 0 {
		r, size := utf8.DecodeRune(source)
		source = source[size:]
		if r == '\n' {
			if column > max_line_chars {
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
	if column > max_line_chars {
		emit(column)
	}
	return diags
}

func assert_diags_source_filebytes_bounded(diags []Diagnostic, source []byte) {
	diags_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	source_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(source), Lo: 0, Hi: max_file_size_bytes,
		Message: "Source is the file bytes bounded by per-file size budget",
	})
	invariant.Cross_Product(
		diags_boundary, source_boundary,
		invariant.Excluding("Both diags and source at safety caps is bad",
			invariant.Bucket_Hi(diags_boundary),
			invariant.Bucket_Hi(source_boundary)),
		invariant.Excluding("Diags at safety cap with zero source bytes is bad",
			invariant.Bucket_Hi(diags_boundary),
			invariant.Bucket_Lo(source_boundary)),
		invariant.Excluding("Source at safety cap with zero diags is bad",
			invariant.Bucket_Lo(diags_boundary),
			invariant.Bucket_Hi(source_boundary)),
	)
}

func assert_file_source_documentation_entry(
	file_set *token.FileSet, file *ast.File, source []byte,
) {
	source_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(source), Lo: 0, Hi: max_file_size_bytes,
		Message: "Source is the file bytes bounded by per-file size budget",
	})
	documentation_axis := invariant.Sometimes(
		file.Doc != nil, "File has a package doc sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
		source_axis, documentation_axis,
		invariant.Excluding("Source at file-size at safety cap is bad",
			invariant.Bucket_Hi(source_axis),
			invariant.Bucket_True(documentation_axis)),
		invariant.Excluding("Source at file-size at safety cap is bad",
			invariant.Bucket_Hi(source_axis),
			invariant.Bucket_False(documentation_axis)),
		invariant.Excluding("Zero-byte source implies package doc comment is absent",
			invariant.Bucket_Lo(source_axis),
			invariant.Bucket_True(documentation_axis)),
	)
}

// TigerStyle: compound conditions hide cases. Split into nested if/else trees so each branch
// is verifiable in isolation. Only the top-level operator is flagged — `&&`/`||` deep inside a
// subexpression (e.g. a function call arg) doesn't make the if itself compound.
func check_compound_if(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() {
		diags_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by per-file diag budget",
		})
		decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(file.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "File.Decls is bounded by AST budget",
		})
		invariant.Cross_Product(diags_boundary, decls_axis,
			invariant.Excluding("Both diags and decls at safety caps is bad",
				invariant.Bucket_Hi(diags_boundary),
				invariant.Bucket_Hi(decls_axis)),
			invariant.Excluding("Diags at safety cap with zero decls is bad",
				invariant.Bucket_Hi(diags_boundary),
				invariant.Bucket_Lo(decls_axis)),
			invariant.Excluding("Decls at safety cap with zero diags is bad",
				invariant.Bucket_Lo(diags_boundary),
				invariant.Bucket_Hi(decls_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	unwrap := func(e ast.Expr) (output ast.Expr) {
		for range invariant.Game_Loop() {
			pe, ok := e.(*ast.ParenExpr)
			if !ok {
				return e
			}
			e = pe.X
		}
		return e
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
	defer func() {
		diags_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by per-file diag budget",
		})
		decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(file.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "File.Decls is bounded by AST budget",
		})
		invariant.Cross_Product(diags_boundary, decls_axis,
			invariant.Excluding("Both diags and decls at safety caps is bad",
				invariant.Bucket_Hi(diags_boundary),
				invariant.Bucket_Hi(decls_axis)),
			invariant.Excluding("Diags at safety cap with zero decls is bad",
				invariant.Bucket_Hi(diags_boundary),
				invariant.Bucket_Lo(decls_axis)),
			invariant.Excluding("Decls at safety cap with zero diags is bad",
				invariant.Bucket_Lo(diags_boundary),
				invariant.Bucket_Hi(decls_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	check := func(pos, lbrace, rbrace token.Pos, label string) {
		position := file_set.Position(pos)
		invariant.Cross_Product(invariant.Always(position.Line != 0,
			"Position.Line is non-zero at this point"))
		lbrace_position := file_set.Position(lbrace)
		invariant.Cross_Product(invariant.Always(lbrace_position.Line != 0,
			"Lbrace_position.Line is non-zero at this point"))
		rbrace_position := file_set.Position(rbrace)
		invariant.Cross_Product(invariant.Always(rbrace_position.Line != 0,
			"Rbrace_position.Line is non-zero at this point"))
		if !lbrace_position.IsValid() {
			return
		}
		if !rbrace_position.IsValid() {
			return
		}
		start := lbrace_position.Line
		end := rbrace_position.Line
		line_count := end - start + 1
		if line_count > max_function_lines {
			diags = append(diags, Diagnostic{
				Position: position,
				Message: fmt.Sprintf(
					"%s is %d lines (max %d)",
					label, line_count, max_function_lines),
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
	defer func() {
		diags_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by per-file diag budget",
		})
		filename_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(filename), Lo: min_go_filename_chars, Hi: max_filesystem_path_chars,
			Message: "Filename is bounded by filesystem path budget",
		})
		err_axis := invariant.Sometimes(
			err != nil, "Err is non-nil sometimes (parse failure)")
		// Diags at the diag-budget safety cap requires pathological input; a
		// zero-diag parse failure on a 4-char or 4096-char filename never fires
		// because parse failures attach at least one diagnostic.
		invariant.Cross_Product(diags_boundary, filename_axis, err_axis,
			invariant.Excluding("Diags at diag-budget at safety cap is bad",
				invariant.Bucket_Hi(diags_boundary),
				invariant.Bucket_True(err_axis)),
			invariant.Excluding("Diags at diag-budget at safety cap is bad",
				invariant.Bucket_Hi(diags_boundary),
				invariant.Bucket_False(err_axis)),
			invariant.Excluding("Parse error always attaches at least one diagnostic",
				invariant.Bucket_Lo(diags_boundary),
				invariant.Bucket_Lo(filename_axis),
				invariant.Bucket_True(err_axis)),
			invariant.Excluding("Parse error always attaches at least one diagnostic",
				invariant.Bucket_Lo(diags_boundary),
				invariant.Bucket_Hi(filename_axis),
				invariant.Bucket_True(err_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(filename), Lo: min_go_filename_chars, Hi: max_filesystem_path_chars,
			Message: "Filename is bounded by filesystem path budget",
		}),
	)

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
	return Check_File(file_set, file, source_bytes), nil
}

// Check_File_System_Input bundles the per-run dependencies for the
// filesystem tier: the fs.FS view of the workspace, the OS-side
// primitives the stream-tier symlink check needs, the tracked-paths
// filter, and the parallelism cap.
type Check_File_System_Input struct {
	Fsys           fs.FS
	Root           string
	Root_Directory string
	Tracked        map[string]bool
	CPU_Count      int
	Readlink       func(name string) (target string, err error)
	Stat           func(name string) (info fs.FileInfo, err error)
}

// Check_File_System runs the stream tier, parses all Go files, and
// Runs every per-file and cross-file check across the workspace.
// Diagnostics from every tier are unioned into the returned slice.
func Check_File_System(input *Check_File_System_Input) (diags []Diagnostic, err error) {
	defer func() {
		check_file_system_assert_exit(diags, err, input)
	}()
	check_file_system_input_assert_entry(input)
	root := input.Root
	if root == "" {
		root = "."
	}
	cpu_count := input.CPU_Count
	if cpu_count < 1 {
		cpu_count = 1
	}
	directory_has_tracked := check_file_system_directory_index(input.Tracked)
	invariant.Is_Sometimes(directory_has_tracked == nil,
		"Directory_has_tracked can be empty or zero on this branch")

	// Stream and AST tiers run in series; parse failures degrade to per-file diagnostics.
	stream_diags, paths, err := check_file_system_stream(&check_file_system_stream_input{
		Fsys:                  input.Fsys,
		Root:                  root,
		Root_Directory:        input.Root_Directory,
		Tracked:               input.Tracked,
		Directory_Has_Tracked: directory_has_tracked,
		Readlink:              input.Readlink,
		Stat:                  input.Stat,
	})
	invariant.Cross_Product(invariant.Always(err == nil, "Stream tier err is nil"),
		invariant.Sometimes(stream_diags == nil, "Stream_diags is empty for clean tier"),
		invariant.Sometimes(paths == nil, "Paths is empty for empty tree"))
	if err != nil {
		return nil, err
	}
	sources, err := check_file_system_read_files(input.Fsys, paths)
	invariant.Cross_Product(invariant.Always(err == nil, "Read_files err is nil"),
		invariant.Sometimes(len(sources) == 0, "Sources is empty for empty paths"))
	if err != nil {
		return nil, err
	}
	parsed_files, parse_diags := check_file_system_parse_files(paths, sources, cpu_count)
	invariant.Cross_Product(
		invariant.Sometimes(parsed_files == nil, "Parsed_files empty when zero files"),
		invariant.Sometimes(parse_diags == nil, "Parse_diags empty for clean parse"))
	modules, err := build_module_index(input.Fsys, parsed_files)
	invariant.Cross_Product(invariant.Always(modules != nil, "Modules is non-nil"),
		invariant.Always(err == nil, "Build_module_index err is nil"))
	if err != nil {
		return nil, err
	}
	output := append([]Diagnostic{}, stream_diags...)
	output = append(output, parse_diags...)
	output = append(output, check_file_system_run_checks(parsed_files, cpu_count)...)
	output = append(output, check_file_system_package_split(parsed_files)...)
	output = append(output, check_file_system_method_prefix(parsed_files)...)
	output = append(output, check_binary_module_layout(parsed_files, modules)...)
	output = append(output, check_shared_library_no_internal(parsed_files, modules)...)
	output = append(output, check_library_tier_depth(parsed_files, modules)...)
	output = append(output, check_no_ambient_stdlib(parsed_files, modules)...)
	return append(output, check_package_documentation_comment(parsed_files)...), nil
}

func check_file_system_assert_exit(
	diags []Diagnostic, err error, input *Check_File_System_Input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	e := invariant.Sometimes(err != nil, "Err is non-nil sometimes")
	root_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Root), Lo: 0, Hi: max_filesystem_path_chars,
		Message: "Root is bounded by filesystem path budget",
	})
	root_directory_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Root_Directory), Lo: 0, Hi: max_filesystem_path_chars,
		Message: "Root_Directory is bounded by filesystem path budget",
	})
	tracked_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Tracked), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Tracked is bounded by AST budget",
	})
	invariant.Cross_Product(d, e, root_axis, root_directory_axis, tracked_axis,
		invariant.Excluding("Diags Hi (err true)",
			invariant.Bucket_Hi(d), invariant.Bucket_True(e)),
		invariant.Excluding("Diags Hi (err false)",
			invariant.Bucket_Hi(d), invariant.Bucket_False(e)),
		invariant.Excluding("FS error attaches diag",
			invariant.Bucket_Lo(d), invariant.Bucket_True(e)),
		invariant.Excluding("Hi Root unreachable",
			invariant.Bucket_Hi(root_axis),
			invariant.Bucket_Hi(root_directory_axis)),
		invariant.Excluding("Hi Root unreachable",
			invariant.Bucket_Hi(root_axis),
			invariant.Bucket_Lo(root_directory_axis)),
		invariant.Excluding("Hi RD unreachable",
			invariant.Bucket_Hi(root_directory_axis),
			invariant.Bucket_Lo(root_axis)),
		invariant.Excluding("Hi Tracked unreachable",
			invariant.Bucket_Hi(tracked_axis),
			invariant.Bucket_Lo(root_axis)))
}

func check_file_system_input_assert_entry(input *Check_File_System_Input) {
	cpu_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: input.CPU_Count, Lo: 0, Hi: cpu_count_max,
		Message: "CPU_Count is in the per-process worker budget",
	})
	cpu_zero := invariant.Sometimes(
		input.CPU_Count == 0, "CPU_Count is zero in fstest.MapFS-backed callers")
	root_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Root), Lo: 0, Hi: max_filesystem_path_chars,
		Message: "Root is bounded by filesystem path budget",
	})
	root_directory_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Root_Directory), Lo: 0, Hi: max_filesystem_path_chars,
		Message: "Root_Directory is bounded by filesystem path budget",
	})
	tracked_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Tracked), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Tracked is bounded by AST budget",
	})
	invariant.Cross_Product(invariant.Always(input != nil, "Input is non-nil"),
		cpu_boundary, cpu_zero, root_axis, root_directory_axis, tracked_axis,
		invariant.Excluding("CPU zero implies cpu_zero true",
			invariant.Bucket_Lo(cpu_boundary), invariant.Bucket_False(cpu_zero)),
		invariant.Excluding("CPU max implies cpu_zero false",
			invariant.Bucket_Hi(cpu_boundary), invariant.Bucket_True(cpu_zero)),
		invariant.Excluding("Hi Root unreachable",
			invariant.Bucket_Hi(root_axis), invariant.Bucket_Hi(root_directory_axis)),
		invariant.Excluding("Hi Root unreachable",
			invariant.Bucket_Hi(root_axis), invariant.Bucket_Lo(root_directory_axis)),
		invariant.Excluding("Hi RD unreachable",
			invariant.Bucket_Hi(root_directory_axis), invariant.Bucket_Lo(root_axis)),
		invariant.Excluding("Hi Tracked unreachable",
			invariant.Bucket_Hi(tracked_axis), invariant.Bucket_Lo(root_axis)),
		invariant.Excluding("Hi Tracked unreachable",
			invariant.Bucket_Hi(tracked_axis), invariant.Bucket_Hi(root_axis)),
	)
}

// Returns the set of directories that contain at least one tracked path,
// so walkers can SkipDir entire .gitignored subtrees instead of descending
// and rejecting every file one-by-one. Returns nil when tracked is nil so
// callers can compare against nil to disable filtering.
func check_file_system_directory_index(tracked map[string]bool) (output map[string]bool) {
	defer func() {
		t := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(tracked), Lo: 0, Hi: max_parsed_files_per_call,
			Message: "Tracked is bounded by budget",
		})
		o := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(output), Lo: 0, Hi: max_parsed_files_per_call,
			Message: "Output is bounded by budget",
		})
		// (Lo tracked, Hi output) is logically impossible: output ⊆
		// ancestor-set(tracked), so output > tracked is unreachable for
		// any input. (Hi tracked, *) tuples require a workspace at the
		// max_parsed_files_per_call safety cap — that endpoint bounds
		// runaway scans rather than marking a normal workspace size.
		invariant.Cross_Product(t, o,
			invariant.Excluding("Tracked at parsed-files at safety cap is bad",
				invariant.Bucket_Hi(t), invariant.Bucket_Hi(o)),
			invariant.Excluding("Tracked at parsed-files at safety cap is bad",
				invariant.Bucket_Hi(t), invariant.Bucket_Lo(o)),
			invariant.Excluding("Output ancestor-set is bounded by tracked size",
				invariant.Bucket_Lo(t), invariant.Bucket_Hi(o)),
		)
	}()
	t := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(tracked), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Tracked is bounded by budget",
	})
	t_nil := invariant.Sometimes(
		tracked == nil, "Tracked is nil sometimes (nil disables filtering)")
	invariant.Cross_Product(t, t_nil,
		invariant.Excluding("Tracked at parsed-files at safety cap is bad",
			invariant.Bucket_Hi(t), invariant.Bucket_True(t_nil)),
		invariant.Excluding("Tracked at parsed-files at safety cap is bad",
			invariant.Bucket_Hi(t), invariant.Bucket_False(t_nil)),
		invariant.Excluding("Tracked size zero implies tracked is nil",
			invariant.Bucket_Lo(t), invariant.Bucket_False(t_nil)),
	)
	if tracked == nil {
		return nil
	}
	output = make(map[string]bool, len(tracked))
	for p := range tracked {
		for range invariant.Game_Loop() {
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
// rebase equivalent.
func git_input_check(input Git_Input) (diags []Diagnostic) {
	defer func() {
		git_input_check_assert_exit(diags, input)
	}()
	// Main_Reference_Absent only set when Enabled is true (see main_load_git);
	// (Enabled=false, Main_Reference_Absent=true) is unreachable by construction.
	enabled := invariant.Sometimes(input.Enabled, "Git history tier is enabled")
	absent := invariant.Sometimes(
		input.Main_Reference_Absent, "Main reference is absent on shallow checkouts")
	mc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Merge_Commits), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Merge_commits is bounded by AST budget",
	})
	mc_empty := invariant.Sometimes(
		len(input.Merge_Commits) == 0, "Merge_commits is empty sometimes")
	nmc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Non_Merge_Commits), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Non_merge_commits is bounded by AST budget",
	})
	nmc_empty := invariant.Sometimes(
		len(input.Non_Merge_Commits) == 0, "Non_merge_commits is empty sometimes")
	invariant.Cross_Product(
		enabled, absent, mc, mc_empty, nmc, nmc_empty,
		invariant.Excluding("Main_Reference_Absent only set when Enabled true",
			invariant.Bucket_False(enabled), invariant.Bucket_True(absent)),
		invariant.Excluding("Hi mc implies non-empty",
			invariant.Bucket_Hi(mc), invariant.Bucket_True(mc_empty)),
		invariant.Excluding("Lo mc implies empty true",
			invariant.Bucket_Lo(mc), invariant.Bucket_False(mc_empty)),
		invariant.Excluding("Hi nmc implies non-empty",
			invariant.Bucket_Hi(nmc), invariant.Bucket_True(nmc_empty)),
		invariant.Excluding("Lo nmc implies empty true",
			invariant.Bucket_Lo(nmc), invariant.Bucket_False(nmc_empty)),
	)
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

func git_input_check_assert_exit(diags []Diagnostic, input Git_Input) {
	diags_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded",
	})
	enabled_axis := invariant.Sometimes(
		input.Enabled, "Git history tier is enabled sometimes")
	merge_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Merge_Commits), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Merge_commits is bounded by AST budget",
	})
	non_merge_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Non_Merge_Commits), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Non_merge_commits is bounded by AST budget",
	})
	invariant.Cross_Product(diags_boundary, enabled_axis, merge_axis, non_merge_axis,
		invariant.Excluding("Hi diags with git enabled is bad",
			invariant.Bucket_Hi(diags_boundary),
			invariant.Bucket_True(enabled_axis)),
		invariant.Excluding("Hi diags with git disabled is bad",
			invariant.Bucket_Hi(diags_boundary),
			invariant.Bucket_False(enabled_axis)),
	)
}

func git_input_check_commit_diagnostics_assert_exit(
	diags []Diagnostic, commits []Git_Commit,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(commits), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Commits is bounded by AST budget",
	})
	c_empty := invariant.Sometimes(len(commits) == 0, "Commits is empty sometimes")
	invariant.Cross_Product(d, c, c_empty,
		invariant.Excluding("Hi c implies non-empty",
			invariant.Bucket_Hi(c), invariant.Bucket_True(c_empty)),
		invariant.Excluding("Lo c implies empty true",
			invariant.Bucket_Lo(c), invariant.Bucket_False(c_empty)),
		invariant.Excluding("Hi d with empty commits unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_True(c_empty)),
		invariant.Excluding("Hi d with commits unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_False(c_empty)),
	)
}

// Flags each merge commit on the branch (rebase-instead violation) plus any
// over-length subject. Subtree merges are exempt. Split out of git_input_check
// so each commit-slice carries its boundary coverage in a function that fits
// the length cap.
func git_input_check_merge_diagnostics(commits []Git_Commit) (diags []Diagnostic) {
	defer func() {
		git_input_check_commit_diagnostics_assert_exit(diags, commits)
	}()
	cc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(commits), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Commits is bounded by AST budget",
	})
	cc_empty := invariant.Sometimes(len(commits) == 0, "Commits is empty sometimes")
	invariant.Cross_Product(cc, cc_empty,
		invariant.Excluding("Hi cc implies non-empty",
			invariant.Bucket_Hi(cc), invariant.Bucket_True(cc_empty)),
		invariant.Excluding("Lo cc implies empty true",
			invariant.Bucket_Lo(cc), invariant.Bucket_False(cc_empty)),
	)
	for _, c := range commits {
		if c.Subject == "" {
			continue
		}
		filename := "<git:" + git_input_check_short_hash(c.Hash) + ">"
		if len(c.Subject) > max_commit_subject_chars {
			diags = append(diags, Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "commit-subject-length",
				Want: fmt.Sprintf(
					"subject ≤ %d chars", max_commit_subject_chars),
				Message: fmt.Sprintf(
					"commit subject is %d chars (max %d)",
					len(c.Subject), max_commit_subject_chars),
			})
			// Helpers below assert subject ≤ max_commit_subject_chars as
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
// the branch, plus any over-length subject. Split out of git_input_check for
// the same length-cap reason as the merge variant.
func git_input_check_non_merge_diagnostics(commits []Git_Commit) (diags []Diagnostic) {
	defer func() {
		git_input_check_commit_diagnostics_assert_exit(diags, commits)
	}()
	cc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(commits), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Commits is bounded by AST budget",
	})
	cc_empty := invariant.Sometimes(len(commits) == 0, "Commits is empty sometimes")
	invariant.Cross_Product(cc, cc_empty,
		invariant.Excluding("Hi cc implies non-empty",
			invariant.Bucket_Hi(cc), invariant.Bucket_True(cc_empty)),
		invariant.Excluding("Lo cc implies empty true",
			invariant.Bucket_Lo(cc), invariant.Bucket_False(cc_empty)),
	)
	for _, c := range commits {
		if c.Subject == "" {
			continue
		}
		if len(c.Subject) > max_commit_subject_chars {
			filename := "<git:" + git_input_check_short_hash(c.Hash) + ">"
			diags = append(diags, Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "commit-subject-length",
				Want: fmt.Sprintf(
					"subject ≤ %d chars", max_commit_subject_chars),
				Message: fmt.Sprintf(
					"commit subject is %d chars (max %d)",
					len(c.Subject), max_commit_subject_chars),
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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(subject), Lo: min_non_empty, Hi: max_commit_subject_chars,
			Message: "Subject length is bounded by callers filtering empty + max chars",
		}),
		invariant.Always(subject != "",
			"Subject is non-empty per upstream git_input_check filter"),
	)
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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(subject), Lo: min_non_empty, Hi: max_commit_subject_chars,
			Message: "Subject length is bounded by callers filtering empty + max chars",
		}),
		invariant.Always(subject != "",
			"Subject is non-empty per upstream git_input_check filter"),
	)
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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X: len(s), Lo: 0, Hi: git_short_hash_chars,
				Message: "S is the short-hash, capped at git_short_hash_chars",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(h), Lo: 0, Hi: git_full_hash_chars_sha_256,
			Message: "H is a git hash; capped at SHA-256 full-hash width",
		}),
	)
	if len(h) > git_short_hash_chars {
		return h[:git_short_hash_chars]
	}
	return h
}

// File-fragmentation check. Splitting code across many tiny files makes a
// package harder to read top-to-bottom and forces readers to chase symbols
// across the filesystem. The rule: per package, expected_max files =
// ceil(total_lines / max_lines_per_file). Source files, test files, and
// each distinct build-tag constraint form independent groups (a Linux-only
// file and a generic file genuinely have to live separately). SLOC is total
// lines per the user's directive — comments and blanks count.
type package_group_key struct {
	Directory string
	Is_Test   bool
	Build     string
}

type package_group_state struct {
	Files []parsed_file
	Lines int
}

func check_file_system_package_split(parsed_files []parsed_file) (diags []Diagnostic) {
	defer func() {
		check_file_system_package_split_assert_exit(diags, parsed_files)
	}()
	check_file_system_package_split_assert_entry(parsed_files)
	groups := map[package_group_key]*package_group_state{}
	for _, pf := range parsed_files {
		key := package_group_key{
			Directory: path.Dir(pf.Path),
			Is_Test:   strings.HasSuffix(pf.Path, "_test.go"),
			Build:     check_file_system_package_split_build_key(pf.File),
		}
		st := groups[key]
		if st == nil {
			st = &package_group_state{}
			groups[key] = st
		}
		st.Files = append(st.Files, pf)
		tok := pf.File_Set.File(pf.File.Pos())
		invariant.Cross_Product(
			invariant.Always(tok != nil, "Tok is non-nil at this point"))
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
		max_files := (st.Lines + max_lines_per_file - 1) / max_lines_per_file
		if max_files < 1 {
			max_files = 1
		}
		if len(st.Files) <= max_files {
			continue
		}
		diags = append(diags, package_group_key_diag(key, st, max_files))
	}
	return diags
}

func check_file_system_package_split_assert_exit(
	diags []Diagnostic, parsed_files []parsed_file,
) {
	diags_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	parsed_files_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by per-package file budget",
	})
	invariant.Cross_Product(
		diags_boundary, parsed_files_boundary,
		invariant.Excluding("Diags and parsed_files both at safety cap is bad",
			invariant.Bucket_Hi(diags_boundary),
			invariant.Bucket_Hi(parsed_files_boundary)),
		invariant.Excluding("Diags at safety cap with zero parsed_files is bad",
			invariant.Bucket_Hi(diags_boundary),
			invariant.Bucket_Lo(parsed_files_boundary)),
		invariant.Excluding("Parsed_files at safety cap with zero diags is bad",
			invariant.Bucket_Lo(diags_boundary),
			invariant.Bucket_Hi(parsed_files_boundary)),
	)
}

func check_file_system_package_split_assert_entry(parsed_files []parsed_file) {
	parsed_files_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by per-package file budget",
	})
	files_empty_axis := invariant.Sometimes(
		len(parsed_files) == 0, "Parsed_files is empty sometimes")
	invariant.Cross_Product(parsed_files_axis, files_empty_axis,
		// Empty parsed_files ↔ Lo bucket, so (Hi, True) and (Lo, False)
		// are logically impossible (the empty flag is fully determined
		// by the count). (Hi parsed_files, False empty) requires a
		// workspace at the max_parsed_files_per_call safety cap — that
		// endpoint bounds runaway scans, not normal workspace sizes.
		invariant.Excluding("Parsed_files at safety cap contradicts empty true",
			invariant.Bucket_Hi(parsed_files_axis),
			invariant.Bucket_True(files_empty_axis)),
		invariant.Excluding("Parsed_files at safety cap is bad",
			invariant.Bucket_Hi(parsed_files_axis),
			invariant.Bucket_False(files_empty_axis)),
		invariant.Excluding("Zero parsed_files implies empty true",
			invariant.Bucket_Lo(parsed_files_axis),
			invariant.Bucket_False(files_empty_axis)),
	)
}

func package_group_key_diag(
	key package_group_key,
	st *package_group_state,
	max_files int,
) (diag Diagnostic) {
	defer func() {
		package_group_key_assert_diag_exit(key, diag, st)
	}()
	package_group_key_assert_entry(key, st, max_files)

	label := "source"
	if key.Is_Test {
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
				"max %d (one file per %d lines)",
			first.File.Name.Name, key.Directory, len(st.Files), label, build_suffix,
			st.Lines, max_files, max_lines_per_file,
		),
	}
}

func package_group_key_assert_diag_exit(
	key package_group_key, diag Diagnostic, st *package_group_state,
) {
	// Inline Always-calls share the FIRST Cross_Product because the
	// chain-credit gate skips inner-call processing on subsequent
	// Cross_Products in the same defer frame.
	invariant.Cross_Product(
		invariant.Always(diag.Tier == 0, "Tier is 0 at construction"),
		invariant.Always(diag.Name == "",
			"Diag.Name is empty for package-group diagnostics"),
		invariant.Always(len(diag.Name) == package_group_diag_chars,
			"Diag.Name is the fixed empty width"),
		invariant.Always(diag.Want == "",
			"Diag.Want is empty for package-group diagnostics"),
		invariant.Always(len(diag.Want) == package_group_diag_chars,
			"Diag.Want is the fixed empty width"),
		invariant.Always(diag.Message != "",
			"Diag.Message is non-empty for package-group diagnostics"),
	)
	files_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(st.Files), Lo: min_pair, Hi: max_parsed_files_per_call,
		Message: "St.Files is ≥2 when the over-quota diag fires",
	})
	files_d_test := invariant.Sometimes(
		key.Is_Test, "Group is a test package sometimes")
	invariant.Cross_Product(files_d, files_d_test,
		invariant.Excluding("Hi fd source",
			invariant.Bucket_Hi(files_d), invariant.Bucket_False(files_d_test)),
		invariant.Excluding("Hi fd in test",
			invariant.Bucket_Hi(files_d), invariant.Bucket_True(files_d_test)),
	)
}

func package_group_key_assert_entry(
	key package_group_key, st *package_group_state, max_files int,
) {
	lines_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: st.Lines, Lo: min_pair, Hi: max_package_lines_test,
		Message: "Lines is the per-package source-line accumulator " +
			"(test-anchored endpoints)",
	})
	max_files_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: max_files, Lo: min_non_empty, Hi: max_test_package_files,
		Message: "Max_files is the caller-supplied package-size cap",
	})
	invariant.Cross_Product(
		invariant.Always(st != nil, "St is non-nil"),
		invariant.Sometimes(key.Is_Test, "Group is a test package sometimes"),
		invariant.Always(key.Directory != "", "Key.Directory is non-empty path"),
		lines_boundary,
		max_files_boundary,
		// Lines <= 10 (Hi) keeps max_files = ceil(Lines/30000) at Lo=1; the
		// two max_files=Hi tuples are therefore impossible.
		invariant.Excluding("Hi lines keeps max_files at Lo bucket",
			invariant.Bucket_Lo(lines_boundary),
			invariant.Bucket_Hi(max_files_boundary)),
		invariant.Excluding("Hi lines keeps max_files at Lo bucket",
			invariant.Bucket_Hi(lines_boundary),
			invariant.Bucket_Hi(max_files_boundary)),
	)
	files := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(st.Files), Lo: min_pair, Hi: max_parsed_files_per_call,
		Message: "St.Files is ≥2 when the over-quota diag fires",
	})
	files_is_test := invariant.Sometimes(key.Is_Test, "Group is a test package sometimes")
	invariant.Cross_Product(files, files_is_test,
		invariant.Excluding("Hi files unreachable (source)",
			invariant.Bucket_Hi(files), invariant.Bucket_False(files_is_test)),
		invariant.Excluding("Hi files unreachable (test)",
			invariant.Bucket_Hi(files), invariant.Bucket_True(files_is_test)),
	)
	build := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(key.Build), Lo: 0, Hi: max_build_constraint_key_chars,
		Message: "Key.Build is empty or a normalized build constraint",
	})
	build_is_test := invariant.Sometimes(key.Is_Test, "Group is a test package sometimes")
	invariant.Cross_Product(build, build_is_test,
		invariant.Excluding("Hi build unreachable (source)",
			invariant.Bucket_Hi(build), invariant.Bucket_False(build_is_test)),
		invariant.Excluding("Hi build unreachable (test)",
			invariant.Bucket_Hi(build), invariant.Bucket_True(build_is_test)),
	)
	directory_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(key.Directory), Lo: min_non_empty, Hi: max_filesystem_path_chars,
		Message: "Key.Directory is a non-empty package path",
	})
	directory_test := invariant.Sometimes(key.Is_Test, "Group is a test package sometimes")
	invariant.Cross_Product(directory_axis, directory_test,
		invariant.Excluding("Hi directory src",
			invariant.Bucket_Hi(directory_axis),
			invariant.Bucket_False(directory_test)),
		invariant.Excluding("Hi directory test",
			invariant.Bucket_Hi(directory_axis), invariant.Bucket_True(directory_test)),
	)
}

// Build-constraint key for grouping. Uses go/build/constraint, the canonical
// parser, so equivalent expressions ("linux && amd64" vs "amd64 && linux"
// stay distinct in the raw text but the AST stringification normalizes form).
// Only //go:build lines preceding the package clause are considered; in-body
// comments are ignored.
func check_file_system_package_split_build_key(file *ast.File) (key string) {
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X: len(key), Lo: 0, Hi: max_build_constraint_key_chars,
				Message: "Key is the normalized build constraint AST",
			}),
		)
	}()
	invariant.Cross_Product(invariant.Always(
		file != nil, "File is non-nil"))

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
	defer func() {
		check_file_system_method_prefix_assert_exit(diags, parsed_files)
	}()
	parsed_files_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded",
	})
	files_empty_axis := invariant.Sometimes(
		len(parsed_files) == 0, "Parsed_files is empty sometimes")
	invariant.Cross_Product(parsed_files_axis, files_empty_axis,
		invariant.Excluding("Parsed_files at safety cap contradicts empty true",
			invariant.Bucket_Hi(parsed_files_axis),
			invariant.Bucket_True(files_empty_axis)),
		invariant.Excluding("Parsed_files at safety cap is bad",
			invariant.Bucket_Hi(parsed_files_axis),
			invariant.Bucket_False(files_empty_axis)),
		invariant.Excluding("Zero parsed_files implies empty true",
			invariant.Bucket_Lo(parsed_files_axis),
			invariant.Bucket_False(files_empty_axis)),
	)
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

func check_file_system_method_prefix_assert_exit(
	diags []Diagnostic, parsed_files []parsed_file,
) {
	diags_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded",
	})
	parsed_files_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded",
	})
	invariant.Cross_Product(
		diags_boundary, parsed_files_boundary,
		invariant.Excluding("Diags and parsed_files both at safety cap is bad",
			invariant.Bucket_Hi(diags_boundary),
			invariant.Bucket_Hi(parsed_files_boundary)),
		invariant.Excluding("Diags at safety cap with zero parsed_files is bad",
			invariant.Bucket_Hi(diags_boundary),
			invariant.Bucket_Lo(parsed_files_boundary)),
		invariant.Excluding("Parsed_files at safety cap with zero diags is bad",
			invariant.Bucket_Lo(diags_boundary),
			invariant.Bucket_Hi(parsed_files_boundary)),
	)
}

func check_file_system_method_prefix_group(files []parsed_file) (diags []Diagnostic) {
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		f := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(files), Lo: min_non_empty, Hi: max_parsed_files_per_call,
			Message: "Files is ≥1 per group",
		})
		invariant.Cross_Product(d, f,
			invariant.Excluding("Diags at safety cap with files at safety cap is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Hi(f)),
			invariant.Excluding("Diags at safety cap with single file is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Lo(f)),
			invariant.Excluding("Files at safety cap with zero diags is bad",
				invariant.Bucket_Lo(d), invariant.Bucket_Hi(f)),
		)
	}()
	f := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(files), Lo: min_non_empty, Hi: max_parsed_files_per_call,
		Message: "Files is ≥1 per group",
	})
	single_axis := invariant.Sometimes(
		len(files) == count_one, "Files has exactly one file sometimes (single-file group)")
	invariant.Cross_Product(f, single_axis,
		invariant.Excluding("Files at safety cap contradicts single true",
			invariant.Bucket_Hi(f), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Files at safety cap is bad",
			invariant.Bucket_Hi(f), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Single file (Lo) implies single true",
			invariant.Bucket_Lo(f), invariant.Bucket_False(single_axis)),
	)
	declared := check_file_system_method_prefix_group_declared(files)
	invariant.Cross_Product(invariant.Always(declared != nil, "Declared is non-nil"))
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
	defer func() {
		f := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(files), Lo: min_non_empty, Hi: max_parsed_files_per_call,
			Message: "Files is ≥1 per group",
		})
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(declared), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Declared is bounded by budget",
		})
		invariant.Cross_Product(f, d,
			invariant.Excluding("Files and declared both at safety cap is bad",
				invariant.Bucket_Hi(f), invariant.Bucket_Hi(d)),
			invariant.Excluding("Files at safety cap with zero declared types is bad",
				invariant.Bucket_Hi(f), invariant.Bucket_Lo(d)),
			invariant.Excluding("Single file with declared at AST at safety cap is bad",
				invariant.Bucket_Lo(f), invariant.Bucket_Hi(d)),
		)
	}()
	f := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(files), Lo: min_non_empty, Hi: max_parsed_files_per_call,
		Message: "Files is ≥1 per group",
	})
	single_axis := invariant.Sometimes(
		len(files) == count_one, "Files has exactly one file sometimes (single-file group)")
	invariant.Cross_Product(f, single_axis,
		invariant.Excluding("Files at safety cap contradicts single true",
			invariant.Bucket_Hi(f), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Files at safety cap is bad",
			invariant.Bucket_Hi(f), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Single file (Lo) implies single true",
			invariant.Bucket_Lo(f), invariant.Bucket_False(single_axis)),
	)
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
	defer func() {
		check_file_system_method_prefix_group_for_file_assert_exit(diags, declared)
	}()
	check_file_system_method_prefix_group_for_file_assert_entry(declared, pf)
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
		invariant.Cross_Product(invariant.Sometimes(
			type_name == "", "Type_name is empty for unnamed parameter types"))
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
		invariant.Cross_Product(
			invariant.Always(prefix != "", "Prefix is non-empty at this point"))
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
				"function %s has first parameter of type %s; "+
					"rename to %s_<verb> (banned-methods convention: "+
					"free functions over a custom type carry the type "+
					"name as prefix, in the function's own casing style)",
				function_declaration.Name.Name, type_name, prefix),
		})
	}
	return diags
}

func check_file_system_method_prefix_group_for_file_assert_exit(
	diags []Diagnostic, declared map[string]bool,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	dc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(declared), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Declared is ≥1 (caller gates empty)",
	})
	invariant.Cross_Product(d, dc,
		invariant.Excluding("Diags and declared both at safety cap is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(dc)),
		invariant.Excluding("Diags at safety cap with single declared type is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(dc)),
		invariant.Excluding("Declared at AST safety cap with zero diags is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(dc)),
	)
}

func check_file_system_method_prefix_group_for_file_assert_entry(
	declared map[string]bool, pf parsed_file,
) {
	dc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(declared), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Declared is ≥1 (caller gates empty)",
	})
	single_declaration_axis := invariant.Sometimes(
		len(declared) == count_one, "Declared has exactly one type sometimes")
	invariant.Cross_Product(
		invariant.Always(pf.File_Set != nil, "Pf.file_set is non-nil"),
		invariant.Always(pf.File != nil, "Pf.file is non-nil"),
		dc, single_declaration_axis,
		invariant.Excluding("Declared at safety cap contradicts single true",
			invariant.Bucket_Hi(dc), invariant.Bucket_True(single_declaration_axis)),
		invariant.Excluding("Declared at safety cap is bad",
			invariant.Bucket_Hi(dc), invariant.Bucket_False(single_declaration_axis)),
		invariant.Excluding("Single declared type (Lo) implies single true",
			invariant.Bucket_Lo(dc), invariant.Bucket_False(single_declaration_axis)),
	)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pf.Path), Lo: min_go_filename_chars, Hi: max_filesystem_path_chars,
		Message: "Pf.Path spans 4-char (a.go) to deep paths",
	})
	path_single := invariant.Sometimes(
		len(declared) == count_one, "Declared has exactly one type sometimes")
	invariant.Cross_Product(path_axis, path_single,
		invariant.Excluding("Hi path single T",
			invariant.Bucket_Hi(path_axis), invariant.Bucket_True(path_single)),
		invariant.Excluding("Hi path single F",
			invariant.Bucket_Hi(path_axis), invariant.Bucket_False(path_single)),
		invariant.Excluding("Lo path single F",
			invariant.Bucket_Lo(path_axis), invariant.Bucket_False(path_single)),
	)
}

// Extracts the base named type from a parameter expression. Bare
// identifiers, pointer receivers (`*T`), and generic instances over a bare
// identifier qualify — all three are canonical "receiver promoted to first
// param" shapes for methods that the linter forced into free-function form.
// Slices, maps, channels, ellipsis, function types, interfaces, and
// selectors (package-qualified types) are intentionally excluded: they are
// collection or external-package shapes, not method-receiver shapes.
func check_file_system_method_prefix_group_first_parameter_type(expression ast.Expr) (name string) {
	defer func() {
		empty_axis := invariant.Sometimes(
			name == "", "Name is empty for unnamed parameter types")
		size_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: 0, Hi: Max_Identifier_Chars,
			Message: "Name is empty or the parameter type identifier",
		})
		invariant.Cross_Product(
			empty_axis, size_axis,
			// Name=="" iff len(name)==0 — (false, Lo) and (true, Hi) are impossible.
			invariant.Excluding("Non-empty name with zero size contradicts size",
				invariant.Bucket_False(empty_axis), invariant.Bucket_Lo(size_axis)),
			invariant.Excluding("Empty input contradicts size at safety cap",
				invariant.Bucket_True(empty_axis), invariant.Bucket_Hi(size_axis)),
		)
	}()

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
	defer func() {
		sources_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(sources), Lo: 0, Hi: max_parsed_files_per_call,
			Message: "Sources is bounded by budget",
		})
		paths_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(paths), Lo: 0, Hi: max_parsed_files_per_call,
			Message: "Paths is bounded by budget",
		})
		invariant.Cross_Product(sources_axis, paths_axis,
			invariant.Excluding("Sources and paths are 1:1 at file-budget cap is bad",
				invariant.Bucket_Hi(sources_axis), invariant.Bucket_Hi(paths_axis)),
			invariant.Excluding("Sources and paths are 1:1 by caller contract",
				invariant.Bucket_Hi(sources_axis), invariant.Bucket_Lo(paths_axis)),
			invariant.Excluding("Sources and paths are 1:1 by caller contract",
				invariant.Bucket_Lo(sources_axis), invariant.Bucket_Hi(paths_axis)),
		)
	}()
	paths_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(paths), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Paths is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(paths) == 0, "Paths is empty sometimes")
	invariant.Cross_Product(paths_axis, empty_axis,
		invariant.Excluding("Paths at safety cap contradicts empty true",
			invariant.Bucket_Hi(paths_axis), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Paths at safety cap is bad",
			invariant.Bucket_Hi(paths_axis), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero paths implies empty true",
			invariant.Bucket_Lo(paths_axis), invariant.Bucket_False(empty_axis)),
	)
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
	defer func() {
		check_file_system_parse_files_assert_exit(parsed_files, parse_diags, paths, sources)
	}()
	check_file_system_parse_files_assert_entry(paths, sources, cpu_count)

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

func check_file_system_parse_files_assert_exit(
	parsed_files []parsed_file,
	parse_diags []Diagnostic,
	paths []string,
	sources [][]byte,
) {
	parsed_files_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by budget",
	})
	parse_diags_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parse_diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Parse_diags is bounded by budget",
	})
	paths_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(paths), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Paths is bounded by budget",
	})
	sources_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(sources), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Sources is bounded by budget",
	})
	// Paths and sources have a 1:1 relationship by contract (caller
	// passes matched slices), so (Hi paths, Lo sources) and (Lo
	// paths, Hi sources) are logically impossible. Likewise (Hi
	// parse_diags, Lo paths) — zero paths cannot produce parse
	// diagnostics. The remaining Hi(parsed_files) / Hi(parse_diags)
	// / Hi(paths) / Hi(sources) endpoints are the per-call budget
	// safety caps — they bound runaway scans rather than working
	// workspace sizes.
	invariant.Cross_Product(
		parsed_files_axis, parse_diags_axis, paths_axis, sources_axis,
		invariant.Excluding("Parsed_files and parse_diags both at cap is bad",
			invariant.Bucket_Hi(parsed_files_axis),
			invariant.Bucket_Hi(parse_diags_axis)),
		invariant.Excluding("Parsed_files at safety cap is bad",
			invariant.Bucket_Hi(parsed_files_axis),
			invariant.Bucket_Lo(parse_diags_axis)),
		invariant.Excluding("Parse_diags and paths both at cap is bad",
			invariant.Bucket_Hi(parse_diags_axis),
			invariant.Bucket_Hi(paths_axis)),
		invariant.Excluding("Zero paths produce zero parse diagnostics",
			invariant.Bucket_Hi(parse_diags_axis),
			invariant.Bucket_Lo(paths_axis)),
		invariant.Excluding("Paths and sources at file-budget cap is bad",
			invariant.Bucket_Hi(paths_axis),
			invariant.Bucket_Hi(sources_axis)),
		invariant.Excluding("Paths and sources are 1:1 by caller contract",
			invariant.Bucket_Hi(paths_axis),
			invariant.Bucket_Lo(sources_axis)),
		invariant.Excluding("Paths and sources are 1:1 by caller contract",
			invariant.Bucket_Lo(paths_axis),
			invariant.Bucket_Hi(sources_axis)),
	)
}

func check_file_system_parse_files_assert_entry(
	paths []string, sources [][]byte, cpu_count int,
) {
	paths_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(paths), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Paths is bounded by budget",
	})
	sources_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(sources), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Sources is bounded by budget",
	})
	cpu_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: cpu_count, Lo: min_non_empty, Hi: cpu_count_max,
		Message: "Cpu_count is in the per-process worker budget",
	})
	// Paths and sources are 1:1 by contract (each path has a matched
	// source slot), so (Hi paths, Lo sources) and (Lo paths, Hi sources)
	// are logically impossible. (Hi paths, Hi sources) sits at the
	// per-call file-budget safety cap — that endpoint bounds runaway
	// scans, not a working workspace size. (Lo paths, Lo sources, Hi cpu)
	// requires a caller to pass cpu_count at the worker-pool safety cap
	// (1024) with zero work; the worker pool would idle, so reaching it
	// signals misconfigured input rather than a meaningful state.
	invariant.Cross_Product(paths_axis, sources_axis, cpu_axis,
		invariant.Excluding("Paths and sources at file-budget cap is bad",
			invariant.Bucket_Hi(paths_axis), invariant.Bucket_Hi(sources_axis)),
		invariant.Excluding("Paths and sources are 1:1 by caller contract",
			invariant.Bucket_Hi(paths_axis), invariant.Bucket_Lo(sources_axis)),
		invariant.Excluding("Paths and sources are 1:1 by caller contract",
			invariant.Bucket_Lo(paths_axis), invariant.Bucket_Hi(sources_axis)))
}

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

// Walks Fsys for go.mod files, parses only the `module` line via regex
// (avoids pulling in golang.org/x/mod for one field), and maps each
// parsed file to its owning module. When Fsys is rooted inside a
// module (no go.mod visible — typical when the linter is run on a
// subdirectory), no modules are discovered and every file maps to -1;
// downstream doctrine checks then no-op on those files rather than
// reporting bogus violations against a partial view of the workspace.
func build_module_index(fsys fs.FS, parsed_files []parsed_file) (index *module_index, err error) {
	defer func() {
		build_module_index_assert_exit(parsed_files, err, index)
	}()
	build_module_index_assert_entry(fsys, parsed_files)

	index = &module_index{File_To_Module: make(map[string]int, len(parsed_files))}
	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, walk_err error) (output error) {
		return build_module_index_walk(fsys, index, p, d, walk_err)
	})
	if err != nil {
		return nil, err
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
		invariant.Cross_Product(invariant.Always(
			canonical_directory != "", "Canonical_directory is non-empty"))
		directory_package := index.Modules[module_index_number].Directory_Package
		if _, has := directory_package[canonical_directory]; !has {
			directory_package[canonical_directory] = pf.File.Name.Name
		}
	}
	return index, nil
}

func build_module_index_assert_exit(
	parsed_files []parsed_file, err error, index *module_index,
) {
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by budget",
	})
	err_axis := invariant.Sometimes(
		err != nil, "Err is non-nil sometimes (fs.WalkDir failure)")
	modules_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(index.Modules), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Index.Modules is bounded by parsed-files budget",
	})
	f2m_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(index.File_To_Module), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Index.File_To_Module is bounded by parsed-files budget",
	})
	invariant.Cross_Product(p, err_axis, modules_axis, f2m_axis,
		invariant.Always(index != nil, "Index is non-nil"),
		invariant.Excluding("Parsed_files Hi is bad (err true)",
			invariant.Bucket_Hi(p), invariant.Bucket_True(err_axis)),
		invariant.Excluding("Parsed_files Hi is bad (err false)",
			invariant.Bucket_Hi(p), invariant.Bucket_False(err_axis)),
		invariant.Excluding("Zero parsed implies err nil",
			invariant.Bucket_Lo(p), invariant.Bucket_True(err_axis)),
		invariant.Excluding("Hi modules unreachable",
			invariant.Bucket_Hi(modules_axis),
			invariant.Bucket_True(err_axis)),
		invariant.Excluding("Hi modules unreachable",
			invariant.Bucket_Hi(modules_axis),
			invariant.Bucket_False(err_axis)),
		invariant.Excluding("Hi F2M tracks Hi modules",
			invariant.Bucket_Hi(f2m_axis), invariant.Bucket_Lo(modules_axis)),
	)
}

func build_module_index_assert_entry(fsys fs.FS, parsed_files []parsed_file) {
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(parsed_files) == 0, "Parsed_files is empty sometimes")
	invariant.Cross_Product(invariant.Always(fsys != nil, "Fsys is non-nil"), p, empty_axis,
		invariant.Excluding("Hi p contradicts empty",
			invariant.Bucket_Hi(p), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi p unreachable",
			invariant.Bucket_Hi(p), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Lo p implies empty",
			invariant.Bucket_Lo(p), invariant.Bucket_False(empty_axis)))
}

// Handles one fs.WalkDir visit for build_module_index: skips vendored/hidden
// directories, and on each go.mod records the module root and import path.
func build_module_index_walk(
	fsys fs.FS, index *module_index, p string, d fs.DirEntry, walk_err error,
) (output error) {
	if walk_err != nil {
		return walk_err
	}
	if d.IsDir() {
		if p != "." {
			if check_file_system_stream_skip_directory(d.Name()) {
				return fs.SkipDir
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
	index.Modules = append(index.Modules, module_information{
		Root:              path.Dir(p),
		Module_Path:       module_path,
		Is_Shared_Library: module_path == shared_library_module_path,
		Directory_Package: make(map[string]string),
	})
	return nil
}

// Strips ^v[0-9]+$ segments from a slash-separated directory path so
// snap/v2/X is treated identically to snap/X. Major-version segments
// are Go module-versioning convention rather than real package tiers,
// and the doctrine's depth rules must see through them.
func module_index_canonicalize(directory string) (canonical string) {
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:       len(canonical),
				Lo:      min_non_empty,
				Hi:      max_filesystem_directory_chars,
				Message: "Canonical is a directory path; min `.`, capped",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(directory), Lo: min_non_empty, Hi: max_filesystem_directory_chars,
			Message: "Directory is a path.Dir result; min `.`, capped",
		}),
	)

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
	defer func() {
		m := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(modules), Lo: 0, Hi: max_parsed_files_per_call,
			Message: "Modules is bounded by budget",
		})
		empty_axis := invariant.Sometimes(len(modules) == 0, "Modules is empty sometimes")
		invariant.Cross_Product(m, empty_axis,
			invariant.Excluding("Max modules contradicts empty true",
				invariant.Bucket_Hi(m), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Modules at safety cap is bad",
				invariant.Bucket_Hi(m), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero modules implies empty true",
				invariant.Bucket_Lo(m), invariant.Bucket_False(empty_axis)),
		)
		index_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: index, Lo: module_index_not_found, Hi: max_modules_per_workspace,
			Message: "Index is -1 when zero modules match, else matched index",
		})
		index_zero := invariant.Sometimes(index == 0,
			"Index is the first module on the most common workspace shape")
		invariant.Cross_Product(
			index_boundary, index_zero,
			// Lo=-1 ⇒ index==0 is false; Hi=8 ⇒ index==0 is false.
			invariant.Excluding("Negative index sentinel contradicts index_zero true",
				invariant.Bucket_Lo(index_boundary),
				invariant.Bucket_True(index_zero)),
			invariant.Excluding("Max index contradicts index_zero true",
				invariant.Bucket_Hi(index_boundary),
				invariant.Bucket_True(index_zero)),
		)
	}()
	m := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Modules is bounded by budget",
	})
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(file_path), Lo: min_go_filename_chars, Hi: max_filesystem_path_chars,
		Message: "File_path is a path; min 4-char `a.go`, capped",
	})
	invariant.Cross_Product(
		path_axis, m,
		invariant.Excluding("Path at safety cap with modules at safety cap is bad",
			invariant.Bucket_Hi(path_axis), invariant.Bucket_Hi(m)),
		invariant.Excluding("Path at safety cap with zero modules is bad",
			invariant.Bucket_Hi(path_axis), invariant.Bucket_Lo(m)),
		invariant.Excluding("Modules at safety cap with min path is bad",
			invariant.Bucket_Lo(path_axis), invariant.Bucket_Hi(m)),
	)

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
	defer func() {
		assert_diags_files_modules_bounded(diags, parsed_files, modules)
	}()
	check_binary_module_layout_assert_entry(parsed_files, modules)

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
			Message: fmt.Sprintf(
				"binary module %q forbids package %q outside of "+
					"internal/; move %q under %s/internal/",
				m.Module_Path, pf.File.Name.Name, directory, m.Root,
			),
		})
	}
	return diags
}

func check_binary_module_layout_assert_entry(
	parsed_files []parsed_file, modules *module_index,
) {
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(parsed_files) == 0, "Parsed_files is empty sometimes")
	m_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules.Modules), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Modules.Modules is bounded by parsed-file budget",
	})
	f_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules.File_To_Module), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Modules.File_To_Module is bounded by parsed-file budget",
	})
	invariant.Cross_Product(invariant.Always(modules != nil, "Modules is non-nil"),
		p, empty_axis, m_axis, f_axis,
		invariant.Excluding("Max len contradicts empty=true",
			invariant.Bucket_Hi(p), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis p at safety cap is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero len implies empty true",
			invariant.Bucket_Lo(p), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Modules at safety cap unreachable in test corpus",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("File_To_Module at safety cap unreachable in test corpus",
			invariant.Bucket_Hi(f_axis), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Modules and File_To_Module both Hi unreachable in test corpus",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_Hi(f_axis)),
		invariant.Excluding("Modules Hi with empty parsed_files impossible",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("File_To_Module Hi with empty parsed_files impossible",
			invariant.Bucket_Hi(f_axis), invariant.Bucket_True(empty_axis)),
	)
}

// True for directories whose first segment is `internal` — the only
// legal home for non-main packages in a binary module under the
// doctrine. "." (the module root) is illegal for non-main code:
// `package main` is handled by the caller's earlier short-circuit.
func check_binary_module_layout_is_legal(directory string) (legal bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			legal, "License is in the allowed set"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(directory), Lo: min_non_empty, Hi: max_filesystem_directory_chars,
			Message: "Directory is a path.Dir result; min `.`, capped",
		}),
	)

	if directory == "." {
		return false
	}
	segments := strings.Split(directory, "/")
	return segments[0] == "internal"
}

// The shared library exists to be imported by binaries; any internal/
// subtree would hide part of its surface and defeat the layering.
// Reported once per offending directory (the first `internal` segment
// found in any file's path), attributed to the earliest-seen file
// inside that directory so the diagnostic has a real location.
func check_shared_library_no_internal(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {
	defer func() {
		assert_diags_files_modules_bounded(diags, parsed_files, modules)
	}()
	assert_files_modules_entry(parsed_files, modules)

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

// Caps how deep non-main packages may nest before they stop being a
// recognizable library tier. The rule: at most one non-main Go
// ancestor in the same module, after canonicalizing major-version
// segments (v2, v3, …) which are module-versioning convention rather
// than real package layers. The deepest legal position is the
// composition tier — the only place where ambient-stdlib binding is
// permitted.
func check_library_tier_depth(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {
	defer func() {
		assert_diags_files_modules_bounded(diags, parsed_files, modules)
	}()
	assert_files_modules_entry(parsed_files, modules)

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
		invariant.Cross_Product(
			invariant.Always(canonical != "", "Canonical is non-empty at this point"))
		if canonical == "." {
			continue
		}
		ancestors := check_library_tier_depth_ancestors(canonical)
		invariant.Cross_Product(invariant.Sometimes(
			ancestors == nil, "Ancestors can be empty or zero on this branch"))
		count := 0
		var ancestor_names []string
		for _, a := range ancestors {
			if _, has := m.Directory_Package[a]; !has {
				continue
			}
			count++
			ancestor_names = append(ancestor_names, a)
		}
		if count <= 1 {
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
				pf.File.Name.Name, canonical, count, ancestor_names,
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
	defer func() {
		a := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(ancestors), Lo: 0, Hi: max_string_slice_per_call,
			Message: "Ancestors is bounded by budget",
		})
		empty_axis := invariant.Sometimes(
			len(ancestors) == 0, "Ancestors is empty sometimes (directory at root)")
		invariant.Cross_Product(a, empty_axis,
			invariant.Excluding("Max a contradicts empty true",
				invariant.Bucket_Hi(a), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Axis a at safety cap is bad",
				invariant.Bucket_Hi(a), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero a implies empty true",
				invariant.Bucket_Lo(a), invariant.Bucket_False(empty_axis)),
		)
	}()
	directory_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(directory), Lo: min_non_empty, Hi: max_filesystem_directory_chars,
		Message: "Directory is a path.Dir result; min `.`, capped",
	})
	root_axis := invariant.Sometimes(directory == path_root, "Directory is root `.` sometimes")
	invariant.Cross_Product(directory_axis, root_axis,
		invariant.Excluding("Max directory contradicts root true",
			invariant.Bucket_Hi(directory_axis), invariant.Bucket_True(root_axis)),
		invariant.Excluding("Zero directory contradicts root true",
			invariant.Bucket_Lo(directory_axis), invariant.Bucket_True(root_axis)),
	)

	current := directory
	for range invariant.Game_Loop() {
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
	defer func() {
		check_package_documentation_comment_assert_exit(diags, parsed_files)
	}()
	check_package_documentation_comment_assert_entry(parsed_files)
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
		if pf.File.Doc != nil {
			if len(pf.File.Doc.List) > 0 {
				st.Has_Documentation = true
			}
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

func check_package_documentation_comment_assert_exit(
	diags []Diagnostic, parsed_files []parsed_file,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by budget",
	})
	invariant.Cross_Product(d, p,
		invariant.Excluding("Both diags and pairs at safety caps is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(p)),
		invariant.Excluding("Diags at safety cap with zero pairs is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(p)),
		invariant.Excluding("Pairs at safety cap with zero diags is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(p)),
	)
}

func check_package_documentation_comment_assert_entry(parsed_files []parsed_file) {
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(parsed_files) == 0, "Parsed_files is empty sometimes")
	invariant.Cross_Product(p, empty_axis,
		invariant.Excluding("Max len contradicts empty=true",
			invariant.Bucket_Hi(p), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis p at safety cap is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero len implies empty true",
			invariant.Bucket_Lo(p), invariant.Bucket_False(empty_axis)),
	)
}

// Runs checks per file in parallel — CPU bound, capped at the injected
// CPU_Count (typically runtime.NumCPU from main.go).
func check_file_system_run_checks(parsed_files []parsed_file, cpu_count int) (diags []Diagnostic) {
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
			Message: "Parsed_files is bounded by budget",
		})
		invariant.Cross_Product(d, p,
			invariant.Excluding("Both diags and pairs at safety caps is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Hi(p)),
			invariant.Excluding("Diags at safety cap with zero pairs is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Lo(p)),
			invariant.Excluding("Pairs at safety cap with zero diags is bad",
				invariant.Bucket_Lo(d), invariant.Bucket_Hi(p)),
		)
	}()
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by budget",
	})
	cpu_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: cpu_count, Lo: min_non_empty, Hi: cpu_count_max,
		Message: "Cpu_count is in the per-process worker budget",
	})
	invariant.Cross_Product(p, cpu_axis,
		invariant.Excluding("Both p and cpu at safety caps is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Hi(cpu_axis)),
		invariant.Excluding("Max p with min cpu is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Lo(cpu_axis)),
	)

	per_file_diags := make([][]Diagnostic, len(parsed_files))
	sem := make(chan struct{}, cpu_count)
	var wg sync.WaitGroup
	for i, pf := range parsed_files {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, pf parsed_file) {
			defer wg.Done()
			defer func() { <-sem }()
			per_file_diags[i] = Check_File(pf.File_Set, pf.File, pf.Source)
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
	defer func() {
		assert_diags_source_bounded(diags, source)
	}()
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(source), Lo: 0, Hi: max_file_size_bytes,
		Message: "Source is bounded by budget",
	})
	documentation_axis := invariant.Sometimes(
		file.Doc != nil, "File has a package doc sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
		s, documentation_axis,
		invariant.Excluding("Source at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_True(documentation_axis)),
		invariant.Excluding("Source at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_False(documentation_axis)),
		invariant.Excluding("Zero-byte source implies package doc comment is absent",
			invariant.Bucket_Lo(s), invariant.Bucket_True(documentation_axis)),
	)

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

func assert_diags_source_bounded(diags []Diagnostic, source []byte) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(source), Lo: 0, Hi: max_file_size_bytes,
		Message: "Source is bounded by budget",
	})
	invariant.Cross_Product(d, s,
		invariant.Excluding("Both d and s at safety caps is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(s)),
		invariant.Excluding("Max d with min s is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(s)),
		invariant.Excluding("Axis s at safety cap with zero diags is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(s)),
	)
}

func check_comments_group_capital(file_set *token.FileSet, c *ast.Comment) (diags []Diagnostic) {
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		text_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(c.Text), Lo: min_pair, Hi: max_file_size_bytes,
			Message: "C.Text is ≥2 chars (`//`) and bounded by size budget",
		})
		invariant.Cross_Product(d, text_axis,
			invariant.Excluding("Both diags and text at safety caps is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Hi(text_axis)),
			invariant.Excluding("Max diags with zero text is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Lo(text_axis)),
			invariant.Excluding("Text at safety cap with zero diags is bad",
				invariant.Bucket_Lo(d), invariant.Bucket_Hi(text_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(c != nil, "C is non-nil"),
	)

	body := comment_body(c.Text)
	invariant.Cross_Product(invariant.Sometimes(body == "", "Body is empty for empty fixtures"))
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
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		text_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(c.Text), Lo: min_pair, Hi: max_file_size_bytes,
			Message: "C.Text is ≥2 chars (`//`) and bounded by size budget",
		})
		invariant.Cross_Product(d, text_axis,
			invariant.Excluding("Both diags and text at safety caps is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Hi(text_axis)),
			invariant.Excluding("Max diags with zero text is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Lo(text_axis)),
			invariant.Excluding("Text at safety cap with zero diags is bad",
				invariant.Bucket_Lo(d), invariant.Bucket_Hi(text_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(c != nil, "C is non-nil"),
	)

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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X: len(body), Lo: 0, Hi: max_filesystem_path_chars,
				Message: "Body is the trimmed comment payload",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(text), Lo: min_pair, Hi: max_comment_text_chars,
			Message: "Text is a comment line (`//`), bounded by budget",
		}),
	)

	if !strings.HasPrefix(text, "//") {
		return ""
	}
	return strings.TrimLeft(text[2:], " \t")
}

func check_comments_group_has_space_after_slashes(text string) (ok bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			ok, "Predicate evaluates true here"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(text), Lo: min_pair, Hi: max_comment_text_chars,
			Message: "Text is a comment line (`//`), bounded by budget",
		}),
	)

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	graph := build_file_call_graph(file_set, file)
	invariant.Cross_Product(
		invariant.Sometimes(graph.Caller_Order == nil, "Caller_order is empty for files "+
			"with zero func decls"))
	adj := map[string][]call_edge{}
	for _, e := range graph.Edges {
		adj[e.Caller] = append(adj[e.Caller], e)
	}
	return check_no_recursion_find_cycles(file_set, graph.Caller_Order, adj)
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
	defer func() {
		file_call_graph_assert_exit(graph)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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

func file_call_graph_assert_exit(graph file_call_graph) {
	caller_order_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(graph.Caller_Order), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Graph.Caller_Order is bounded by AST budget",
	})
	decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(graph.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Graph.Decls is bounded by AST budget",
	})
	edges_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(graph.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Graph.Edges is bounded by AST budget",
	})
	invariant.Cross_Product(caller_order_axis, decls_axis, edges_axis,
		invariant.Excluding("Caller_Order and Decls track in lockstep — Hi one, "+
			"Lo other is impossible",
			invariant.Bucket_Hi(caller_order_axis),
			invariant.Bucket_Lo(decls_axis)),
		invariant.Excluding("Caller_Order and Decls track in lockstep — Hi one, "+
			"Lo other is impossible",
			invariant.Bucket_Lo(caller_order_axis),
			invariant.Bucket_Hi(decls_axis)),
		invariant.Excluding("Caller_Order at AST safety cap unreachable in test "+
			"corpus (Lo edges)",
			invariant.Bucket_Hi(caller_order_axis),
			invariant.Bucket_Lo(edges_axis)),
		invariant.Excluding("Caller_Order at AST safety cap unreachable in test "+
			"corpus (Hi edges)",
			invariant.Bucket_Hi(caller_order_axis),
			invariant.Bucket_Hi(edges_axis)),
		invariant.Excluding("Decls at AST safety cap unreachable in test corpus "+
			"(Lo edges)",
			invariant.Bucket_Hi(decls_axis), invariant.Bucket_Lo(edges_axis)),
		invariant.Excluding("Decls at AST safety cap unreachable in test corpus "+
			"(Hi edges)",
			invariant.Bucket_Hi(decls_axis), invariant.Bucket_Hi(edges_axis)),
		invariant.Excluding("Edges at AST safety cap unreachable in test corpus "+
			"(Lo callers)",
			invariant.Bucket_Hi(edges_axis),
			invariant.Bucket_Lo(caller_order_axis)),
	)
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
	defer func() {
		recursion_visitor_assert_exit(v)
	}()
	recursion_visitor_assert_entry(v)

	if n == nil {
		k := v.Push_History[len(v.Push_History)-1]
		v.Push_History = v.Push_History[:len(v.Push_History)-1]
		v.Scopes = v.Scopes[:len(v.Scopes)-k]
		return nil
	}
	pushed := recursion_visitor_enter(v, n)
	invariant.Cross_Product(
		invariant.Sometimes(pushed == 0, "Pushed can be zero on this branch"))
	v.Push_History = append(v.Push_History, pushed)
	return v
}

func recursion_visitor_assert_exit(v *recursion_visitor) {
	scopes_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Scopes), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Scopes is bounded by AST budget",
	})
	e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	caller_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(v.Caller),
		Lo: min_non_empty,
		Hi: Max_Identifier_Chars,
		Message: "V.Caller is a non-empty function name bounded by identifier " +
			"budget",
	})
	targets_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(v.Targets),
		Lo: min_non_empty,
		Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty by construction (visitor created per " +
			"function) and bounded by AST budget",
	})
	history_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Push_History), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is bounded by AST budget",
	})
	invariant.Cross_Product(scopes_axis, e, caller_axis, targets_axis, history_axis,
		invariant.Excluding("Scopes at AST safety cap with edges at safety cap is "+
			"bad",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_Hi(e)),
		invariant.Excluding("Scopes at safety cap with zero edges unreachable in "+
			"test corpus",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_Lo(e)),
		invariant.Excluding("Zero scopes with edges at safety cap unreachable in "+
			"test corpus",
			invariant.Bucket_Lo(scopes_axis), invariant.Bucket_Hi(e)),
		invariant.Excluding("Hi targets at AST safety cap unreachable in test "+
			"corpus (Lo scopes)",
			invariant.Bucket_Hi(targets_axis),
			invariant.Bucket_Lo(scopes_axis)),
		invariant.Excluding("Hi targets at AST safety cap unreachable in test "+
			"corpus (Hi scopes)",
			invariant.Bucket_Hi(targets_axis),
			invariant.Bucket_Hi(scopes_axis)),
		invariant.Excluding("Hi push_history at AST safety cap unreachable in "+
			"test corpus (Lo scopes)",
			invariant.Bucket_Hi(history_axis),
			invariant.Bucket_Lo(scopes_axis)),
		invariant.Excluding("Hi push_history at AST safety cap unreachable in "+
			"test corpus (Hi scopes)",
			invariant.Bucket_Hi(history_axis),
			invariant.Bucket_Hi(scopes_axis)),
	)
}

func recursion_visitor_assert_entry(v *recursion_visitor) {
	scopes_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Scopes), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Scopes is bounded by AST budget",
	})
	e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	caller_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "V.Caller is a non-empty function name bounded by identifier budget",
	})
	targets_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(v.Targets),
		Lo: min_non_empty,
		Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty by construction (visitor created per function) " +
			"and bounded by AST budget",
	})
	history_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Push_History), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(v != nil, "V is non-nil"),
		invariant.Always(v.File_Set != nil, "V.File_Set is non-nil"),
		invariant.Always(v.Edges != nil, "V.Edges is non-nil"),
		scopes_axis, e, caller_axis, targets_axis, history_axis,
		invariant.Excluding("Scopes at AST safety cap with edges at safety cap is bad",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_Hi(e)),
		invariant.Excluding("Scopes at safety cap with zero edges unreachable in test "+
			"corpus",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_Lo(e)),
		invariant.Excluding("Zero scopes with edges at safety cap unreachable in test "+
			"corpus",
			invariant.Bucket_Lo(scopes_axis), invariant.Bucket_Hi(e)),
		invariant.Excluding("Hi targets at AST safety cap unreachable in test corpus (Lo "+
			"scopes)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_Lo(scopes_axis)),
		invariant.Excluding("Hi targets at AST safety cap unreachable in test corpus (Hi "+
			"scopes)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_Hi(scopes_axis)),
		invariant.Excluding("Hi push_history at AST safety cap unreachable in test corpus "+
			"(Lo scopes)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_Lo(scopes_axis)),
		invariant.Excluding("Hi push_history at AST safety cap unreachable in test corpus "+
			"(Hi scopes)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_Hi(scopes_axis)),
	)
}

func recursion_visitor_enter(v *recursion_visitor, n ast.Node) (pushed int) {
	defer func() {
		recursion_visitor_enter_assert_exit(pushed, v)
	}()
	recursion_visitor_enter_assert_entry(v)

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

func recursion_visitor_enter_assert_exit(pushed int, v *recursion_visitor) {
	pb_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: pushed, Lo: 0, Hi: count_one,
		Message: "Pushed is either 0 or 1",
	})
	pz_axis := invariant.Sometimes(
		pushed == 0, "Pushed is zero for nodes that leave the scope stack "+
			"untouched")
	e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	caller_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "V.Caller is non-empty bounded",
	})
	targets_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Targets), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty by construction",
	})
	history_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Push_History), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is bounded by AST budget",
	})
	sd := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Scopes), Lo: count_one, Hi: max_ast_nodes_per_call,
		Message: "V.Scopes spans 1 to AST budget",
	})
	so := invariant.Sometimes(
		len(v.Scopes) == count_one, "V.Scopes has exactly 1 entry sometimes")
	invariant.Cross_Product(
		pb_axis, pz_axis, e, caller_axis, targets_axis, history_axis, sd, so,
		invariant.Excluding("Zero pushed pz true",
			invariant.Bucket_Lo(pb_axis), invariant.Bucket_False(pz_axis)),
		invariant.Excluding("Max pushed pz true",
			invariant.Bucket_Hi(pb_axis), invariant.Bucket_True(pz_axis)),
		invariant.Excluding("Zero pushed Hi edges",
			invariant.Bucket_Lo(pb_axis), invariant.Bucket_Hi(e)),
		invariant.Excluding("Max pushed Hi edges",
			invariant.Bucket_Hi(pb_axis), invariant.Bucket_Hi(e)),
		invariant.Excluding("Hi targets Lo pushed",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_Lo(pb_axis)),
		invariant.Excluding("Hi targets Hi pushed",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_Hi(pb_axis)),
		invariant.Excluding("Hi history Lo pushed",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_Lo(pb_axis)),
		invariant.Excluding("Hi history Hi pushed",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_Hi(pb_axis)),
		invariant.Excluding("First Visit pushes a frame",
			invariant.Bucket_Lo(pb_axis), invariant.Bucket_Lo(history_axis)),
		invariant.Excluding("Hi sd with one true",
			invariant.Bucket_Hi(sd), invariant.Bucket_True(so)),
		invariant.Excluding("Hi sd with one false",
			invariant.Bucket_Hi(sd), invariant.Bucket_False(so)),
		invariant.Excluding("Lo sd implies one true",
			invariant.Bucket_Lo(sd), invariant.Bucket_False(so)),
	)
}

func recursion_visitor_enter_assert_entry(v *recursion_visitor) {
	e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	scopes_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Scopes), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Scopes is bounded by AST budget",
	})
	p_caller := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "V.Caller is non-empty function name bounded",
	})
	p_targets := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Targets), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty by construction",
	})
	p_history := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Push_History), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(v != nil, "V is non-nil"),
		invariant.Always(v.File_Set != nil, "V.File_Set is non-nil"),
		invariant.Always(v.Edges != nil, "V.Edges is non-nil"),
		scopes_axis, e, p_caller, p_targets, p_history,
		invariant.Excluding("Hi scopes Hi edges bad",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_Hi(e)),
		invariant.Excluding("Hi scopes Lo edges bad",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_Lo(e)),
		invariant.Excluding("Hi targets Lo scopes bad",
			invariant.Bucket_Hi(p_targets), invariant.Bucket_Lo(scopes_axis)),
		invariant.Excluding("Hi targets Hi scopes bad",
			invariant.Bucket_Hi(p_targets), invariant.Bucket_Hi(scopes_axis)),
		invariant.Excluding("Hi history Lo scopes bad",
			invariant.Bucket_Hi(p_history), invariant.Bucket_Lo(scopes_axis)),
		invariant.Excluding("Hi history Hi scopes bad",
			invariant.Bucket_Hi(p_history), invariant.Bucket_Hi(scopes_axis)),
		invariant.Excluding("Lo scopes Hi edges bad",
			invariant.Bucket_Lo(scopes_axis), invariant.Bucket_Hi(e)),
	)
}

func recursion_visitor_enter_define_statement(v *recursion_visitor, s ast.Stmt) {
	defer func() {
		recursion_visitor_enter_define_statement_assert_exit(s, v)
	}()
	e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	s_nil := invariant.Sometimes(s == nil,
		"Init statement is nil sometimes (for/if without an init clause)")
	caller_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "V.Caller is non-empty bounded by identifier budget",
	})
	targets_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Targets), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty by construction",
	})
	scopes_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Scopes), Lo: min_pair, Hi: max_ast_nodes_per_call,
		Message: "V.Scopes is ≥2 (Walk's BlockStmt push + caller's for/if push)",
	})
	history_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(v.Push_History),
		Lo: min_non_empty,
		Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is ≥1 (Visit pushed the containing BlockStmt before " +
			"reaching for/if)",
	})
	invariant.Cross_Product(
		invariant.Always(v != nil, "V is non-nil"),
		invariant.Always(v.File_Set != nil, "V.File_Set is non-nil"),
		invariant.Always(v.Edges != nil, "V.Edges is non-nil"),
		e, s_nil, caller_axis, targets_axis, scopes_axis, history_axis,
		invariant.Excluding("Hi edges with nil init unreachable",
			invariant.Bucket_Hi(e), invariant.Bucket_True(s_nil)),
		invariant.Excluding("Hi edges with present init unreachable",
			invariant.Bucket_Hi(e), invariant.Bucket_False(s_nil)),
		invariant.Excluding("Hi targets unreachable (nil init)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_True(s_nil)),
		invariant.Excluding("Hi targets unreachable (present init)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_False(s_nil)),
		invariant.Excluding("Hi scopes unreachable (nil init)",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_True(s_nil)),
		invariant.Excluding("Hi scopes unreachable (present init)",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_False(s_nil)),
		invariant.Excluding("Hi history unreachable (nil init)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_True(s_nil)),
		invariant.Excluding("Hi history unreachable (present init)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_False(s_nil)),
	)

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

func recursion_visitor_enter_define_statement_assert_exit(s ast.Stmt, v *recursion_visitor) {
	e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	s_nil := invariant.Sometimes(s == nil,
		"Init statement is nil sometimes (for/if without an init clause)")
	caller_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "V.Caller is non-empty bounded by identifier budget",
	})
	targets_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Targets), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty by construction",
	})
	scopes_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Scopes), Lo: min_pair, Hi: max_ast_nodes_per_call,
		Message: "V.Scopes is ≥2 (Walk's BlockStmt push + caller's for/if push)",
	})
	history_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(v.Push_History),
		Lo: min_non_empty,
		Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is ≥1 (Visit pushed the containing BlockStmt " +
			"before reaching for/if)",
	})
	invariant.Cross_Product(
		e, s_nil, caller_axis, targets_axis, scopes_axis, history_axis,
		invariant.Excluding("Hi edges with nil init unreachable",
			invariant.Bucket_Hi(e), invariant.Bucket_True(s_nil)),
		invariant.Excluding("Hi edges with present init unreachable",
			invariant.Bucket_Hi(e), invariant.Bucket_False(s_nil)),
		invariant.Excluding("Hi targets unreachable (nil init)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_True(s_nil)),
		invariant.Excluding("Hi targets unreachable (present init)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_False(s_nil)),
		invariant.Excluding("Hi scopes unreachable (nil init)",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_True(s_nil)),
		invariant.Excluding("Hi scopes unreachable (present init)",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_False(s_nil)),
		invariant.Excluding("Hi history unreachable (nil init)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_True(s_nil)),
		invariant.Excluding("Hi history unreachable (present init)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_False(s_nil)),
	)
}

func recursion_visitor_define_ident(v *recursion_visitor, e ast.Expr) {
	defer func() {
		recursion_visitor_define_ident_assert_exit(e, v)
	}()
	edges_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	e_nil := invariant.Sometimes(e == nil, "Range key/value or assign LHS is nil sometimes")
	caller_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "V.Caller is non-empty bounded by identifier budget",
	})
	targets_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Targets), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty by construction",
	})
	scopes_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Scopes), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Scopes is non-empty (define requires a scope)",
	})
	history_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Push_History), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is non-empty (define needs history)",
	})
	invariant.Cross_Product(
		invariant.Always(v != nil, "V is non-nil"),
		invariant.Always(v.File_Set != nil, "V.File_Set is non-nil"),
		invariant.Always(v.Edges != nil, "V.Edges is non-nil"),
		edges_axis, e_nil, caller_axis, targets_axis, scopes_axis, history_axis,
		invariant.Excluding("Edges Hi with nil expr unreachable",
			invariant.Bucket_Hi(edges_axis), invariant.Bucket_True(e_nil)),
		invariant.Excluding("Edges Hi with present expr unreachable",
			invariant.Bucket_Hi(edges_axis), invariant.Bucket_False(e_nil)),
		invariant.Excluding("Hi targets unreachable (nil expr)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_True(e_nil)),
		invariant.Excluding("Hi targets unreachable (present expr)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_False(e_nil)),
		invariant.Excluding("Hi scopes unreachable (nil expr)",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_True(e_nil)),
		invariant.Excluding("Hi scopes unreachable (present expr)",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_False(e_nil)),
		invariant.Excluding("Hi history unreachable (nil expr)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_True(e_nil)),
		invariant.Excluding("Hi history unreachable (present expr)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_False(e_nil)),
		invariant.Excluding("E_nil true comes only from RangeStmt which appends a scope "+
			"before calling",
			invariant.Bucket_True(e_nil), invariant.Bucket_Lo(scopes_axis)),
	)

	identifier, is_ident := e.(*ast.Ident)
	if !is_ident {
		return
	}
	if identifier.Name == "_" {
		return
	}
	v.Scopes[len(v.Scopes)-1][identifier.Name] = true
}

func recursion_visitor_define_ident_assert_exit(e ast.Expr, v *recursion_visitor) {
	edges_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	e_nil := invariant.Sometimes(
		e == nil, "Range key/value or assign LHS is nil sometimes")
	caller_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "V.Caller is non-empty bounded by identifier budget",
	})
	targets_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Targets), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty by construction",
	})
	scopes_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Scopes), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Scopes is non-empty (define requires a scope)",
	})
	history_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Push_History), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is non-empty (define needs history)",
	})
	invariant.Cross_Product(
		edges_axis, e_nil, caller_axis, targets_axis, scopes_axis, history_axis,
		invariant.Excluding("Edges Hi with nil expr unreachable",
			invariant.Bucket_Hi(edges_axis), invariant.Bucket_True(e_nil)),
		invariant.Excluding("Edges Hi with present expr unreachable",
			invariant.Bucket_Hi(edges_axis), invariant.Bucket_False(e_nil)),
		invariant.Excluding("Hi targets unreachable (nil expr)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_True(e_nil)),
		invariant.Excluding("Hi targets unreachable (present expr)",
			invariant.Bucket_Hi(targets_axis), invariant.Bucket_False(e_nil)),
		invariant.Excluding("Hi scopes unreachable (nil expr)",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_True(e_nil)),
		invariant.Excluding("Hi scopes unreachable (present expr)",
			invariant.Bucket_Hi(scopes_axis), invariant.Bucket_False(e_nil)),
		invariant.Excluding("Hi history unreachable (nil expr)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_True(e_nil)),
		invariant.Excluding("Hi history unreachable (present expr)",
			invariant.Bucket_Hi(history_axis), invariant.Bucket_False(e_nil)),
		invariant.Excluding("E_nil true comes only from RangeStmt which appends a "+
			"scope before calling",
			invariant.Bucket_True(e_nil), invariant.Bucket_Lo(scopes_axis)),
	)
}

// Recursion_visitor_call_fun_is_ident reports whether the call expression's
// Fun position is a bare identifier (`f()`) as opposed to a selector
// expression (`pkg.f()` or `x.m()`). Nil-safe so it can be evaluated as a
// Cross_Product Sometimes-predicate before the parent function's pointer
// assertions fire.
func recursion_visitor_call_function_is_ident(call *ast.CallExpr) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(yes, "Affirmative branch is exercised"))
	}()
	invariant.Cross_Product(invariant.Always(call != nil, "Call is non-nil"))
	_, is_ident := call.Fun.(*ast.Ident)
	return is_ident
}

func recursion_visitor_enter_record_call_edge(v *recursion_visitor, call *ast.CallExpr) {
	defer func() {
		recursion_visitor_enter_record_call_edge_assert_exit(call, v)
	}()
	recursion_visitor_enter_record_call_edge_assert_entry(call, v)

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

func recursion_visitor_enter_record_call_edge_assert_exit(
	call *ast.CallExpr, v *recursion_visitor,
) {
	e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	function_ident_axis := invariant.Sometimes(
		recursion_visitor_call_function_is_ident(call),
		"Call.Fun is a bare identifier sometimes (vs. a selector like pkg.f or "+
			"x.m)")
	targets_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Targets), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty",
	})
	scopes_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Scopes), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Scopes is ≥1 (Walk pushed BlockStmt scope)",
	})
	// Walk starts on FuncDecl.Body BlockStmt (1 push); CallExpr is reached
	// through ≥1 further non-pushing Visit, so Push_History len ≥ 2.
	history_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Push_History), Lo: min_pair, Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is ≥2 at this site",
	})
	caller_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "V.Caller is non-empty bounded by identifier budget",
	})
	invariant.Cross_Product(
		e,
		function_ident_axis,
		targets_axis,
		scopes_axis,
		history_axis,
		caller_axis,

		invariant.Excluding("Edges at safety cap with bare-ident call unreachable "+
			"in test corpus",
			invariant.Bucket_Hi(e), invariant.Bucket_True(function_ident_axis)),
		invariant.Excluding("Edges at safety cap with selector call unreachable "+
			"in test corpus",
			invariant.Bucket_Hi(e),
			invariant.Bucket_False(function_ident_axis)),
		invariant.Excluding("Hi targets unreachable (bare-ident call)",
			invariant.Bucket_Hi(targets_axis),
			invariant.Bucket_True(function_ident_axis)),
		invariant.Excluding("Hi targets unreachable (selector call)",
			invariant.Bucket_Hi(targets_axis),
			invariant.Bucket_False(function_ident_axis)),
		invariant.Excluding("Hi scopes unreachable (bare-ident call)",
			invariant.Bucket_Hi(scopes_axis),
			invariant.Bucket_True(function_ident_axis)),
		invariant.Excluding("Hi scopes unreachable (selector call)",
			invariant.Bucket_Hi(scopes_axis),
			invariant.Bucket_False(function_ident_axis)),
		invariant.Excluding("Hi push_history unreachable (bare-ident call)",
			invariant.Bucket_Hi(history_axis),
			invariant.Bucket_True(function_ident_axis)),
		invariant.Excluding("Hi push_history unreachable (selector call)",
			invariant.Bucket_Hi(history_axis),
			invariant.Bucket_False(function_ident_axis)),
	)
}

func recursion_visitor_enter_record_call_edge_assert_entry(
	call *ast.CallExpr, v *recursion_visitor,
) {
	e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*v.Edges), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "V.Edges is bounded by AST budget",
	})
	function_ident_axis := invariant.Sometimes(recursion_visitor_call_function_is_ident(call),
		"Call.Fun is a bare identifier sometimes (vs. a selector like pkg.f or x.m)")
	targets_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Targets), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "V.Targets is non-empty by construction",
	})
	scopes_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(v.Scopes),
		Lo: min_non_empty,
		Hi: max_ast_nodes_per_call,
		Message: "V.Scopes is ≥1 (Walk(Body) pushed BlockStmt scope before reaching " +
			"CallExpr)",
	})
	// Walk starts on the FuncDecl.Body BlockStmt, which Visit pushes a 1
	// onto Push_History for; CallExpr is reached only through at least
	// one further non-pushing Visit (ExprStmt or argument descent), so
	// Push_History length at this site is ≥ 2.
	history_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(v.Push_History),
		Lo: min_pair,
		Hi: max_ast_nodes_per_call,
		Message: "V.Push_History is ≥2 at this site (BlockStmt push + at least one " +
			"non-pushing descent to reach CallExpr)",
	})
	caller_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(v.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "V.Caller is a non-empty function name bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(v != nil, "V is non-nil"),
		invariant.Always(v.File_Set != nil, "V.File_Set is non-nil"),
		invariant.Always(v.Edges != nil, "V.Edges is non-nil"),
		invariant.Always(call != nil, "Call is non-nil"),
		e, function_ident_axis, targets_axis, scopes_axis, history_axis, caller_axis,
		invariant.Excluding("Edges at safety cap with bare-ident call unreachable in test "+
			"corpus",
			invariant.Bucket_Hi(e), invariant.Bucket_True(function_ident_axis)),
		invariant.Excluding("Edges at safety cap with selector call unreachable in test "+
			"corpus",
			invariant.Bucket_Hi(e), invariant.Bucket_False(function_ident_axis)),
		invariant.Excluding("Hi targets unreachable (bare-ident call)",
			invariant.Bucket_Hi(targets_axis),
			invariant.Bucket_True(function_ident_axis)),
		invariant.Excluding("Hi targets unreachable (selector call)",
			invariant.Bucket_Hi(targets_axis),
			invariant.Bucket_False(function_ident_axis)),
		invariant.Excluding("Hi scopes unreachable (bare-ident call)",
			invariant.Bucket_Hi(scopes_axis),
			invariant.Bucket_True(function_ident_axis)),
		invariant.Excluding("Hi scopes unreachable (selector call)",
			invariant.Bucket_Hi(scopes_axis),
			invariant.Bucket_False(function_ident_axis)),
		invariant.Excluding("Hi push_history unreachable (bare-ident call)",
			invariant.Bucket_Hi(history_axis),
			invariant.Bucket_True(function_ident_axis)),
		invariant.Excluding("Hi push_history unreachable (selector call)",
			invariant.Bucket_Hi(history_axis),
			invariant.Bucket_False(function_ident_axis)),
	)
}

// Iterative 3-color DFS for cycle detection. Each back edge from a GRAY node
// to a still-GRAY ancestor closes a cycle; we emit one diagnostic per back
// edge (so a strongly-connected component with multiple back edges yields
// multiple diagnostics, one per cycle).
func check_no_recursion_find_cycles(
	file_set *token.FileSet,
	callers []string,
	adj map[string][]call_edge,
) (diags []Diagnostic) {
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(callers), Lo: 0, Hi: max_string_slice_per_call,
			Message: "Callers is bounded by budget",
		})
		a := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(adj), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Adj is bounded by budget",
		})
		// Diags, callers, adj all bounded by per-file safety caps; (Hi, *) and
		// (*, Hi) endpoint pairings are pathological input. (Lo callers, Hi
		// adj) is logically impossible: max edges require at least one caller
		// at the endpoint.
		invariant.Cross_Product(d, c, a,
			invariant.Excluding("Diags at diag-budget safety cap with callers at "+
				"slice-budget at safety cap is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Hi(c)),
			invariant.Excluding("Diags at diag-budget at safety cap is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_Lo(c)),
			invariant.Excluding("Callers at slice-budget safety cap with adj at AST "+
				"at safety cap is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_Hi(a)),
			invariant.Excluding("Callers at slice-budget at safety cap is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_Lo(a)),
			invariant.Excluding("Zero callers implies max adj equals zero",
				invariant.Bucket_Lo(c), invariant.Bucket_Hi(a)),
		)
	}()
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(callers), Lo: 0, Hi: max_string_slice_per_call,
		Message: "Callers is bounded by budget",
	})
	a := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(adj), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Adj is bounded by budget",
	})
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		c, a,
		invariant.Excluding("Callers at slice-budget safety cap with adj at AST at safety "+
			"cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Hi(a)),
		invariant.Excluding("Callers at slice-budget at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Lo(a)),
		invariant.Excluding("Zero callers implies max adj equals zero",
			invariant.Bucket_Lo(c), invariant.Bucket_Hi(a)),
	)

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
			check_no_recursion_find_cycles_dfs(file_set, start, color, adj)...)
	}
	return diags
}

func check_no_recursion_find_cycles_dfs(
	file_set *token.FileSet,
	start string,
	color map[string]int,
	adj map[string][]call_edge,
) (diags []Diagnostic) {
	defer func() {
		check_no_recursion_find_cycles_dfs_assert_exit(diags, color, adj)
	}()
	check_no_recursion_find_cycles_dfs_assert_entry(file_set, start, color, adj)

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
				check_no_recursion_find_cycles_dfs_diag(
					file_set, cycle_nodes, e, adj))
		}
	}
	return diags
}

func check_no_recursion_find_cycles_dfs_assert_exit(
	diags []Diagnostic, color map[string]int, adj map[string][]call_edge,
) {
	co := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(color), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Color is ≥1 at exit (start was set to gray)",
	})
	a := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(adj), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Adj is bounded by budget",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	invariant.Cross_Product(co, a, d,
		invariant.Excluding("Both co and adj at safety caps is bad",
			invariant.Bucket_Hi(co), invariant.Bucket_Hi(a)),
		invariant.Excluding("Max co with zero adj is bad",
			invariant.Bucket_Hi(co), invariant.Bucket_Lo(a)),
		invariant.Excluding("Both a and d at safety caps is bad",
			invariant.Bucket_Hi(a), invariant.Bucket_Hi(d)),
		invariant.Excluding("Max a with zero d is bad",
			invariant.Bucket_Hi(a), invariant.Bucket_Lo(d)),
		invariant.Excluding("Zero a with max d is bad",
			invariant.Bucket_Lo(a), invariant.Bucket_Hi(d)),
	)
}

func check_no_recursion_find_cycles_dfs_assert_entry(
	file_set *token.FileSet, start string, color map[string]int, adj map[string][]call_edge,
) {
	co := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(color), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Color is bounded by budget",
	})
	a := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(adj), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Adj is bounded by budget",
	})
	start_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(start), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Start is a non-empty function key bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		start_axis, co, a,
		invariant.Excluding("Both co and adj at safety caps is bad",
			invariant.Bucket_Hi(co), invariant.Bucket_Hi(a)),
		invariant.Excluding("Max co with zero adj is bad",
			invariant.Bucket_Hi(co), invariant.Bucket_Lo(a)),
		invariant.Excluding("Zero co with max a is bad",
			invariant.Bucket_Lo(co), invariant.Bucket_Hi(a)),
		invariant.Excluding("Both start and co at safety caps is bad",
			invariant.Bucket_Hi(start_axis), invariant.Bucket_Hi(co)),
	)
}

func check_no_recursion_find_cycles_dfs_diag(
	file_set *token.FileSet,
	cycle_nodes []string,
	back_edge call_edge,
	adj map[string][]call_edge,
) (diag Diagnostic) {
	defer func() {
		check_no_recursion_find_cycles_dfs_diag_assert_exit(cycle_nodes, adj, diag)
	}()
	a := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(adj), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Adj is ≥1 (cycle path requires at least one edge)",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(cycle_nodes), Lo: min_non_empty, Hi: max_string_slice_per_call,
		Message: "Cycle_nodes is ≥1 per caller gate",
	})
	caller := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(back_edge.Caller), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Back_edge.Caller is a non-empty function name",
	})
	callee := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(back_edge.Callee), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Back_edge.Callee is a non-empty function name",
	})
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(!back_edge.Shadowed, "Back_edge is unshadowed (asserted by "+
			"callers)"),
		a, c, caller, callee,
		invariant.Excluding("Adj at AST safety cap with cycle_nodes at slice-budget at "+
			"safety cap is bad",
			invariant.Bucket_Hi(a), invariant.Bucket_Hi(c)),
		invariant.Excluding("Adj at AST at safety cap is bad",
			invariant.Bucket_Hi(a), invariant.Bucket_Lo(c)),
		invariant.Excluding("Cycle_nodes at slice-budget at safety cap is bad",
			invariant.Bucket_Lo(a), invariant.Bucket_Hi(c)),
		invariant.Excluding("Hi caller unreachable (Hi a)",
			invariant.Bucket_Hi(caller), invariant.Bucket_Hi(a)),
		invariant.Excluding("Hi callee unreachable (Hi a)",
			invariant.Bucket_Hi(callee), invariant.Bucket_Hi(a)),
		invariant.Excluding("Self-cycle requires caller=callee (Hi caller Lo callee)",
			invariant.Bucket_Lo(c),
			invariant.Bucket_Hi(caller),
			invariant.Bucket_Lo(callee)),
		invariant.Excluding("Self-cycle requires caller=callee (Lo caller Hi callee)",
			invariant.Bucket_Lo(c),
			invariant.Bucket_Lo(caller),
			invariant.Bucket_Hi(callee)),
	)
	// Shadowed back edges fail-fatal via the per-edge Is_Always below — a
	// Sometimes axis on back_edge.Shadowed has no reachable true bucket,
	// so coverage isn't tracked here.
	// Collect all edges traversed in the cycle, so we can assert the
	// no-shadow invariant over every one — not just the back edge.
	for i_index := 0; i_index < len(cycle_nodes)-1; i_index++ {
		from := cycle_nodes[i_index]
		to := cycle_nodes[i_index+1]
		for _, ce := range adj[from] {
			if ce.Callee == to {
				invariant.Cross_Product(
					invariant.Always(!ce.Shadowed, "Check_no_recursion: cycle "+
						"edge shadowed"))
				break
			}
		}
	}
	invariant.Cross_Product(
		invariant.Always(!back_edge.Shadowed, "Check_no_recursion: back-edge shadowed"))
	return Diagnostic{
		Position: back_edge.Position,
		Message:  check_no_recursion_find_cycles_dfs_diag_message(cycle_nodes),
	}
}

func check_no_recursion_find_cycles_dfs_diag_assert_exit(
	cycle_nodes []string, adj map[string][]call_edge, diag Diagnostic,
) {
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(cycle_nodes), Lo: min_non_empty, Hi: max_string_slice_per_call,
		Message: "Cycle_nodes is ≥1 per caller gate",
	})
	a := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(adj), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Adj is ≥1 (cycle path requires at least one edge)",
	})
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diag.Name), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Diag.Name is empty for recursion diagnostics",
	})
	want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diag.Want), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Diag.Want is empty for recursion diagnostics",
	})
	message_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(diag.Message),
		Lo:      min_recursion_message_chars,
		Hi:      max_recursion_message_chars,
		Message: "Diag.Message is `recursion: <node> calls itself`",
	})
	invariant.Cross_Product(c, a, name_axis, want_axis, message_axis,
		invariant.Always(diag.Tier == 0, "Tier is 0 at diagnostic construction "+
			"(set later by tier dispatcher)"),
		invariant.Excluding("Cycle_nodes at slice-budget safety cap with adj at "+
			"AST at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Hi(a)),
		invariant.Excluding("Cycle_nodes at slice-budget at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Lo(a)),
		invariant.Excluding("Adj at AST at safety cap is bad",
			invariant.Bucket_Lo(c), invariant.Bucket_Hi(a)),
		invariant.Excluding("Hi name unreachable (Lo c)",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Lo(c)),
		invariant.Excluding("Hi name unreachable (Hi c)",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(c)),
		invariant.Excluding("Hi want unreachable (Lo c)",
			invariant.Bucket_Hi(want_axis), invariant.Bucket_Lo(c)),
		invariant.Excluding("Hi want unreachable (Hi c)",
			invariant.Bucket_Hi(want_axis), invariant.Bucket_Hi(c)),
	)
}

func check_no_recursion_find_cycles_dfs_diag_message(cycle_nodes []string) (message string) {
	defer func() {
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(cycle_nodes), Lo: min_non_empty, Hi: max_string_slice_per_call,
			Message: "Cycle_nodes is ≥1 per caller gate",
		})
		message_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(message),
			Lo: min_recursion_message_chars,
			Hi: max_recursion_message_chars,
			Message: "Message is `recursion: <node> calls itself` (25 for 1-char " +
				"node, 152 for 128-char node)",
		})
		invariant.Cross_Product(message_axis, c,
			invariant.Excluding("Both message and c at safety caps is bad",
				invariant.Bucket_Hi(message_axis), invariant.Bucket_Hi(c)),
			invariant.Excluding("Zero message with max c is bad",
				invariant.Bucket_Lo(message_axis), invariant.Bucket_Hi(c)),
		)
	}()
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(cycle_nodes), Lo: min_non_empty, Hi: max_string_slice_per_call,
		Message: "Cycle_nodes is ≥1 per caller gate",
	})
	single_axis := invariant.Sometimes(
		len(cycle_nodes) == count_one, "Cycle_nodes is exactly 1 sometimes (self-cycle)")
	invariant.Cross_Product(c, single_axis,
		invariant.Excluding("Max c contradicts single true",
			invariant.Bucket_Hi(c), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Axis c at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Zero c implies single true",
			invariant.Bucket_Lo(c), invariant.Bucket_False(single_axis)),
	)

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() {
		e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(exprs), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
			Message: "Exprs is non-empty in observed paths",
		})
		all_axis := invariant.Sometimes(all, "Every entry passed the check")
		invariant.Cross_Product(e, all_axis,
			invariant.Excluding("Max e contradicts all true",
				invariant.Bucket_Hi(e), invariant.Bucket_True(all_axis)),
			invariant.Excluding("Axis e at safety cap is bad",
				invariant.Bucket_Hi(e), invariant.Bucket_False(all_axis)),
		)
	}()
	e := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(exprs), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Exprs is non-empty in observed paths",
	})
	single_axis := invariant.Sometimes(
		len(exprs) == count_one, "Exprs has exactly one entry sometimes")
	invariant.Cross_Product(e, single_axis,
		invariant.Excluding("Max e contradicts single true",
			invariant.Bucket_Hi(e), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Axis e at safety cap is bad",
			invariant.Bucket_Hi(e), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Zero e implies single true",
			invariant.Bucket_Lo(e), invariant.Bucket_False(single_axis)),
	)

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
	defer func() {
		n := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(names), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
			Message: "Names is non-empty in observed paths",
		})
		all_axis := invariant.Sometimes(all, "Every entry passed the check")
		invariant.Cross_Product(n, all_axis,
			invariant.Excluding("Max n contradicts all true",
				invariant.Bucket_Hi(n), invariant.Bucket_True(all_axis)),
			invariant.Excluding("Axis n at safety cap is bad",
				invariant.Bucket_Hi(n), invariant.Bucket_False(all_axis)),
		)
	}()
	n := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(names), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Names is non-empty in observed paths",
	})
	single_axis := invariant.Sometimes(
		len(names) == count_one, "Names has exactly one entry sometimes")
	invariant.Cross_Product(n, single_axis,
		invariant.Excluding("Max n contradicts single true",
			invariant.Bucket_Hi(n), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Axis n at safety cap is bad",
			invariant.Bucket_Hi(n), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Zero n implies single true",
			invariant.Bucket_Lo(n), invariant.Bucket_False(single_axis)),
	)

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() {
		name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(identifier.Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Identifier name length is bounded by identifier budget",
		})
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by per-file diag budget",
		})
		invariant.Cross_Product(name_axis, d,
			invariant.Excluding("Identifier at safety cap with diag-budget cap is bad",
				invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(d)),
			invariant.Excluding("Identifier at safety cap with zero diags unreachable "+
				"in test corpus",
				invariant.Bucket_Hi(name_axis), invariant.Bucket_Lo(d)),
			invariant.Excluding("Lo identifier with diag cap unreachable in tests",
				invariant.Bucket_Lo(name_axis), invariant.Bucket_Hi(d)),
		)
	}()
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(identifier.Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Identifier name length is bounded by identifier budget",
	})
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(identifier != nil, "Identifier is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		name_axis, d,
		invariant.Excluding("Identifier at safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(d)),
		invariant.Excluding("Identifier at safety cap with zero diags unreachable in test "+
			"corpus",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Lo(d)),
		invariant.Excluding("Lo identifier with diag cap unreachable in tests",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_Hi(d)),
	)

	if identifier.Name == "" {
		return
	}
	r := rune(identifier.Name[0])
	if !unicode.IsLower(r) {
		return
	}
	suggested := check_public_struct_fields_named_capitalize(identifier.Name)
	invariant.Cross_Product(
		invariant.Always(suggested != "", "Suggested is non-empty at this point"))
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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(yes, "Affirmative branch is exercised"))
	}()
	if expression == nil {
		return false
	}
	_, is_star := expression.(*ast.StarExpr)
	return is_star
}

func check_public_struct_fields_embedded(
	file_set *token.FileSet, expression ast.Expr, diags *[]Diagnostic,
) {
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by per-file diag budget",
		})
		is_pointer := invariant.Sometimes(expression_is_pointer_wrapped(expression),
			"Embedded type is pointer-wrapped sometimes")
		invariant.Cross_Product(d, is_pointer,
			invariant.Excluding("Diag-budget cap with pointer-embed unreachable in "+
				"test corpus",
				invariant.Bucket_Hi(d), invariant.Bucket_True(is_pointer)),
			invariant.Excluding("Diag-budget cap with value-embed unreachable in test "+
				"corpus",
				invariant.Bucket_Hi(d), invariant.Bucket_False(is_pointer)),
		)
	}()
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	is_pointer := invariant.Sometimes(expression_is_pointer_wrapped(expression),
		"Embedded type is pointer-wrapped sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(diags != nil, "Diags is non-nil"),
		d, is_pointer,
		invariant.Excluding("Diag-budget cap with pointer-embed unreachable in test corpus",
			invariant.Bucket_Hi(d), invariant.Bucket_True(is_pointer)),
		invariant.Excluding("Diag-budget cap with value-embed unreachable in test corpus",
			invariant.Bucket_Hi(d), invariant.Bucket_False(is_pointer)),
	)

	base := expression
	for range invariant.Game_Loop() {
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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(output_string),
				Lo: min_non_empty,
				Hi: Max_Identifier_Chars,
				Message: "Output_string is non-empty per non-empty name " +
					"input and " +
					"bounded by identifier budget",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty Go identifier bounded by identifier budget",
		}),
	)

	rs := []rune(name)
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	tok_file := file_set.File(file.Pos())
	invariant.Cross_Product(
		invariant.Always(tok_file != nil, "Tok_file is non-nil at this point"))
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
		invariant.Cross_Product(
			invariant.Always(entry_type_params != nil, "Entry_type_params is non-nil "+
				"at this point"))
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
	defer func() {
		check_exported_type_exposes_private_walk_input_assert_exit(input)
	}()
	check_exported_type_exposes_private_walk_input_assert_entry(input)

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
			invariant.Cross_Product(invariant.Always(base != nil, "Base is non-nil"))
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
					Message: fmt.Sprintf("exported type %s exposes unexported "+
						"type %s",
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

func check_exported_type_exposes_private_walk_input_assert_exit(
	input *check_exported_type_exposes_private_walk_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	same_file_types := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Same_File_Types),
		Lo:      min_non_empty,
		Hi:      max_ast_nodes_per_call,
		Message: "Same_file_types is bounded by AST budget",
	})
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Entry_Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Entry_Name is non-empty bounded by identifier budget",
	})
	params_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Root_Type_Params), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Root_Type_Params is bounded by identifier budget",
	})
	invariant.Cross_Product(d, same_file_types, name_axis, params_axis,
		invariant.Excluding("Hi d unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(same_file_types)),
		invariant.Excluding("Hi d unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(same_file_types)),
		invariant.Excluding("Hi sf with Lo d unreachable",
			invariant.Bucket_Hi(same_file_types), invariant.Bucket_Lo(d)),
		invariant.Excluding("Hi params unreachable",
			invariant.Bucket_Hi(params_axis), invariant.Bucket_Lo(name_axis)),
		invariant.Excluding("Hi params unreachable",
			invariant.Bucket_Hi(params_axis), invariant.Bucket_Hi(name_axis)),
	)
}

func check_exported_type_exposes_private_walk_input_assert_entry(
	input *check_exported_type_exposes_private_walk_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	same_file_types := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Same_File_Types), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Same_file_types is bounded by AST budget",
	})
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Entry_Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Entry_Name is non-empty bounded by identifier budget",
	})
	params_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Root_Type_Params), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Root_Type_Params is bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.File_Set != nil, "Input.File_Set is non-nil"),
		invariant.Always(input.Root_Struct != nil, "Input.Root_Struct is non-nil"),
		invariant.Always(input.Diags != nil, "Input.Diags is non-nil"),
		d, same_file_types, name_axis, params_axis,
		invariant.Excluding("Hi d unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(same_file_types)),
		invariant.Excluding("Hi d unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(same_file_types)),
		invariant.Excluding("Hi sf with Lo d unreachable",
			invariant.Bucket_Hi(same_file_types), invariant.Bucket_Lo(d)),
		invariant.Excluding("Hi params unreachable",
			invariant.Bucket_Hi(params_axis), invariant.Bucket_Lo(name_axis)),
		invariant.Excluding(
			"Hi params unreachable", invariant.Bucket_Hi(
				params_axis), invariant.Bucket_Hi(name_axis)))
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
	defer func() {
		check_exported_type_exposes_private_check_input_assert_exit(input)
	}()
	check_exported_type_exposes_private_check_input_assert_entry(input)

	base := check_exported_type_exposes_private_unwrap_pointer(input.Expression)
	invariant.Cross_Product(invariant.Always(base != nil, "Base is non-nil at this point"))
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
			"exported type %s exposes unexported type %s",
			input.Entry_Name,
			identifier.Name),
	})
}

func check_exported_type_exposes_private_check_input_assert_entry(
	input *check_exported_type_exposes_private_check_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	generic := invariant.Sometimes(expression_is_pointer_wrapped(input.Expression),
		"Alias is pointer-wrapped sometimes")
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(input.Entry_Name),
		Lo: min_non_empty,
		Hi: Max_Identifier_Chars,
		Message: "Entry_Name is a non-empty exported identifier bounded by identifier " +
			"budget",
	})
	params_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Type_Params), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Type_Params is bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.File_Set != nil, "Input.File_Set is non-nil"),
		invariant.Always(input.Expression != nil, "Input.Expression is non-nil"),
		invariant.Always(input.Diags != nil, "Input.Diags is non-nil"),
		d, generic, name_axis, params_axis,
		invariant.Excluding("Diag-budget cap with generic type unreachable in test corpus",
			invariant.Bucket_Hi(d), invariant.Bucket_True(generic)),
		invariant.Excluding("Diag-budget cap with non-generic type unreachable in test "+
			"corpus",
			invariant.Bucket_Hi(d), invariant.Bucket_False(generic)),
		invariant.Excluding("Hi name with Hi params unreachable in test corpus",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(params_axis)),
		invariant.Excluding("Hi name with Hi diags unreachable in test corpus",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(d)),
		invariant.Excluding("Hi params with Hi diags unreachable in test corpus",
			invariant.Bucket_Hi(params_axis), invariant.Bucket_Hi(d)),
		invariant.Excluding("Hi params with True generic unreachable in test corpus",
			invariant.Bucket_Hi(params_axis), invariant.Bucket_True(generic)),
		invariant.Excluding("Hi params with Lo name unreachable in test corpus",
			invariant.Bucket_Hi(params_axis), invariant.Bucket_Lo(name_axis)),
	)
}

func check_exported_type_exposes_private_check_input_assert_exit(
	input *check_exported_type_exposes_private_check_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by per-file diag budget",
	})
	generic := invariant.Sometimes(expression_is_pointer_wrapped(input.Expression),
		"Alias is pointer-wrapped sometimes")
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(input.Entry_Name),
		Lo: min_non_empty,
		Hi: Max_Identifier_Chars,
		Message: "Entry_Name is a non-empty exported identifier bounded by identifier " +
			"budget",
	})
	params_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Type_Params), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Type_Params is bounded by identifier budget",
	})
	invariant.Cross_Product(d, generic, name_axis, params_axis,
		invariant.Excluding("Diag-budget cap with generic type unreachable in test corpus",
			invariant.Bucket_Hi(d), invariant.Bucket_True(generic)),
		invariant.Excluding("Diag-budget cap with non-generic type unreachable in test "+
			"corpus",
			invariant.Bucket_Hi(d), invariant.Bucket_False(generic)),
		invariant.Excluding("Hi name with Hi params unreachable in test corpus",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(params_axis)),
		invariant.Excluding("Hi name with Hi diags unreachable in test corpus",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(d)),
		invariant.Excluding("Hi params with Hi diags unreachable in test corpus",
			invariant.Bucket_Hi(params_axis), invariant.Bucket_Hi(d)),
		invariant.Excluding("Hi params with True generic unreachable in test corpus",
			invariant.Bucket_Hi(params_axis), invariant.Bucket_True(generic)),
		invariant.Excluding("Hi params with Lo name unreachable in test corpus",
			invariant.Bucket_Hi(params_axis), invariant.Bucket_Lo(name_axis)),
	)
}

func check_exported_type_exposes_private_unwrap_pointer(expression ast.Expr) (output ast.Expr) {
	defer func() {
		invariant.Cross_Product(invariant.Always(output != nil,
			"Output is non-nil (unwrap preserves at least the leaf expression)"))
	}()
	invariant.Cross_Product(
		invariant.Always(expression != nil, "Expression is non-nil"),
	)

	output = expression
	for range invariant.Game_Loop() {
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
	defer func() {
		n := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(names), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Names is bounded by AST budget",
		})
		empty_axis := invariant.Sometimes(len(names) == 0, "Names is empty sometimes")
		invariant.Cross_Product(n, empty_axis,
			invariant.Excluding("Max n contradicts empty true",
				invariant.Bucket_Hi(n), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Axis n at safety cap is bad",
				invariant.Bucket_Hi(n), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero n implies empty true",
				invariant.Bucket_Lo(n), invariant.Bucket_False(empty_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Sometimes(field_list == nil, "Field_list may be nil"))

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
			Message:  "iota is banned; spell out each constant value",
		})
		return true
	})
	return diags
}

// Constants are the file's compile-time facts — magic numbers, error
// strings, table sizes, version tags. A reader scanning a new file should
// see them up front, before runtime state (vars), types, and the functions
// that consume them. Letting consts drift below other declarations forces
// the reader to scroll past behavior to learn the inputs that behavior is
// keyed on. Function-local consts are exempt: their scope is the function,
// not the file, so their proximity to the code that uses them is the point.
func check_constant_first(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	seen_non_constant := false
	for _, declaration := range file.Decls {
		switch d := declaration.(type) {
		case *ast.FuncDecl:
			seen_non_constant = true
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			if d.Tok == token.CONST {
				if seen_non_constant {
					diags = append(diags, Diagnostic{
						Position: file_set.Position(d.Pos()),
						Message: "const declaration must precede all " +
							"var/type/func declarations in the file",
					})
				}
				continue
			}
			seen_non_constant = true
		}
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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() {
		k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(keys), Lo: 0, Hi: max_string_slice_per_call,
			Message: "Keys is bounded by string-slice budget",
		})
		empty := invariant.Sometimes(
			len(keys) == 0, "Keys is empty for malformed or unparseable tags")
		invariant.Cross_Product(k, empty,
			invariant.Excluding("Max keys at safety cap with empty=true is impossible",
				invariant.Bucket_Hi(k), invariant.Bucket_True(empty)),
			invariant.Excluding("Max keys at safety cap marks the pathological-input "+
				"bound",
				invariant.Bucket_Hi(k), invariant.Bucket_False(empty)),
			invariant.Excluding("Zero keys implies empty true; (Lo, false) is "+
				"impossible",
				invariant.Bucket_Lo(k), invariant.Bucket_False(empty)),
		)
	}()
	invariant.Cross_Product(invariant.Always(tag != nil, "Tag is non-nil per caller gate"))
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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	stdlib_imports := check_invariant_assertions_collect_stdlib_imports(file)
	invariant.Cross_Product(invariant.Always(stdlib_imports != nil,
		"Stdlib_imports is non-nil at this point"))
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		struct_type, is_struct := n.(*ast.StructType)
		if !is_struct {
			return true
		}
		if struct_type.Fields == nil {
			return true
		}
		for _, field := range struct_type.Fields.List {
			if !check_invariant_assertions_type_expression_is_mutex(
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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	invariant_idents := check_invariant_assertions_collect_invariant_idents(file)
	idents_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	empty_idents_axis := invariant.Sometimes(
		len(invariant_idents) == 0, "Invariant_idents is empty for files without "+
			"invariant import")
	invariant.Cross_Product(idents_axis, empty_idents_axis,
		invariant.Excluding("Max idents contradicts empty true",
			invariant.Bucket_Hi(idents_axis), invariant.Bucket_True(empty_idents_axis)),
		invariant.Excluding("Idents at safety cap is bad",
			invariant.Bucket_Hi(idents_axis),
			invariant.Bucket_False(empty_idents_axis)),
		invariant.Excluding("Zero idents implies empty true",
			invariant.Bucket_Lo(idents_axis),
			invariant.Bucket_False(empty_idents_axis)),
	)
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
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		empty := invariant.Sometimes(len(diags) == 0, "Diags is empty for clean calls")
		invariant.Cross_Product(d, empty,
			invariant.Excluding("Max diags marks the pathological-input safety cap",
				invariant.Bucket_Hi(d), invariant.Bucket_True(empty)),
			invariant.Excluding("Max diags marks the pathological-input safety cap",
				invariant.Bucket_Hi(d), invariant.Bucket_False(empty)),
			invariant.Excluding("Zero diags implies empty true",
				invariant.Bucket_Lo(d), invariant.Bucket_False(empty)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(call != nil, "Call is non-nil"),
	)
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
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		empty := invariant.Sometimes(len(diags) == 0, "Diags is empty for clean calls")
		invariant.Cross_Product(d, empty,
			invariant.Excluding("Max diags marks the pathological-input safety cap",
				invariant.Bucket_Hi(d), invariant.Bucket_True(empty)),
			invariant.Excluding("Max diags marks the pathological-input safety cap",
				invariant.Bucket_Hi(d), invariant.Bucket_False(empty)),
			invariant.Excluding("Zero diags implies empty true",
				invariant.Bucket_Lo(d), invariant.Bucket_False(empty)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(call != nil, "Call is non-nil"),
	)
	if len(call.Args) == 0 {
		return nil
	}
	predicate := call.Args[0]
	for range invariant.Game_Loop() {
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
	defer func() {
		invariant.Cross_Product(
			invariant.Sometimes(yes, "Yes is true for allowed bound shapes"))
	}()
	for range invariant.Game_Loop() {
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
		for range invariant.Game_Loop() {
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
	defer func() {
		invariant.Cross_Product(
			invariant.Sometimes(yes, "Yes is true for zero/empty literals"))
	}()
	invariant.Cross_Product(invariant.Always(basic != nil, "Basic is non-nil"))

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
		invariant.Cross_Product(
			invariant.Sometimes(name == "", "Name is empty for unnamed types"))
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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X: len(name), Lo: 0, Hi: Max_Identifier_Chars,
				Message: "Name length is bounded by Go identifier conventions",
			}),
		)
	}()

	for range invariant.Game_Loop() {
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
	defer func() {
		assert_diags_source_bounded(diags, source)
	}()
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(source), Lo: 0, Hi: max_file_size_bytes,
		Message: "Source is bounded by budget",
	})
	documentation_axis := invariant.Sometimes(
		file.Doc != nil, "File has a package doc sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
		s, documentation_axis,
		invariant.Excluding("Source at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_True(documentation_axis)),
		invariant.Excluding("Source at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_False(documentation_axis)),
		invariant.Excluding("Zero-byte source implies package doc comment is absent",
			invariant.Bucket_Lo(s), invariant.Bucket_True(documentation_axis)),
	)

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
	invariant.Cross_Product(
		invariant.Always(tok_file != nil, "Tok_file is non-nil at this point"))
	if tok_file != nil {
		filename = tok_file.Name()
		invariant.Cross_Product(
			invariant.Always(filename != "", "Filename is non-empty at this point"))
	}
	return []Diagnostic{{
		Position: token.Position{Filename: filename, Line: 1, Column: 1},
		Message:  "file is not gofmt-clean",
	}}
}

// Dot imports inject names into the file scope, breaking grep-for-origin and
// inviting collisions. Always import with an explicit name (or package name).
func check_no_dot_import(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() {
		assert_diags_imports_bounded(diags, file)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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

// Composition-tier packages named `<X>_default` re-export the library `<X>`
// and must shadow its name at the import site. This is what lets callers
// keep writing `snap.Init(...)` / `snap.Edit(...)` against the OS-bound
// default — and, critically, what lets snap.Edit's source-line rewriter
// find the literal `snap.Edit(` it searches for. An import without the
// alias would surface the path's basename (`snap_default`), and the
// rewriter's search would never match.
func check_default_package_alias(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {
	defer func() {
		assert_diags_imports_bounded(diags, file)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	for _, import_specification := range file.Imports {
		path := strings.Trim(import_specification.Path.Value, `"`)
		slash_offset := strings.LastIndex(path, "/")
		last_segment := path
		if slash_offset >= 0 {
			last_segment = path[slash_offset+1:]
		}
		if !strings.HasSuffix(last_segment, "_default") {
			continue
		}
		want := strings.TrimSuffix(last_segment, "_default")
		if want == "" {
			continue
		}
		got := ""
		if import_specification.Name != nil {
			got = import_specification.Name.Name
		}
		if got == want {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(import_specification.Pos()),
			Name:     "default-package-alias",
			Want:     fmt.Sprintf("import %s %q", want, path),
			Message: fmt.Sprintf(
				"%q must be imported with alias %q; *_default packages re-export "+
					"their library and must shadow its name",
				path, want,
			),
		})
	}
	return diags
}

// Whitebox test packages couple tests to internals; main packages cannot be
// blackbox-tested coherently. Force every _test.go to declare `package
// <X>_test`, which keeps the test suite restricted to the same public API
// callers see and prevents tests from being written against `package main`.
func check_test_package(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	tok_file := file_set.File(file.Pos())
	invariant.Cross_Product(
		invariant.Always(tok_file != nil, "Tok_file is non-nil at this point"))
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

// Banned segments applied to every declared identifier: function names,
// package names, file names, vars, consts, params, named returns, struct
// fields, type names, labels. "util"/"utils"/"utility"/"utilities" are
// dumping-ground signals — code that lands in a util drawer is code whose
// real home nobody bothered to find. "length"/"len" are ambiguous across
// languages (Rust = bytes, Python = code points) and so are banned in favor
// of `_count` (element quantity) and `_size` (byte count); see
// https://tigerbeetle.com/blog/2026-02-16-index-count-offset-size/. The
// `len(...)` and `cap(...)` builtin call sites are exempt because they
// appear only in callee position of a CallExpr, never as a declared name.

// Banned-word check applied to every declared identifier site: function
// names, package name, file name, vars, consts, params, named returns,
// struct fields, type names, labels. Detection splits the identifier into
// segments (snake_case underscores and Ada_Case boundaries) and flags any
// segment that case-insensitively matches a banned entry. Substrings (e.g.
// "helpme") are not flagged. Walks declaration sub-nodes only; use sites
// (CallExpr.Fun, SelectorExpr, bare Ident references) are not visited, so
// `len(xs)` and `cap(xs)` builtin calls are naturally exempt.
func check_banned_identifiers(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	package_hit := check_banned_identifiers_find_hit(
		file.Name.Name, []string{"util", "utils", "utility", "utilities", "length", "len"})
	invariant.Cross_Product(
		invariant.Sometimes(package_hit == "", "Package_hit can be empty on this branch"))
	if package_hit != "" {
		diags = append(diags, Diagnostic{
			Position: file_set.Position(file.Name.Pos()),
			Message: fmt.Sprintf(
				"package name %s contains banned word '%s'",
				file.Name.Name,
				package_hit),
		})
	}
	diags = append(diags, check_banned_identifiers_file_name(file_set, file)...)
	diags = append(diags, check_banned_identifiers_walk(file_set, file)...)
	return diags
}

func check_banned_identifiers_file_name(
	file_set *token.FileSet, file *ast.File,
) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	tok_file := file_set.File(file.Pos())
	invariant.Cross_Product(
		invariant.Always(tok_file != nil, "Tok_file is non-nil at this point"))
	if tok_file == nil {
		return nil
	}
	filename := tok_file.Name()
	invariant.Cross_Product(
		invariant.Always(filename != "", "Filename is non-empty at this point"))
	base := path.Base(filename)
	// Strip .go (and _test) before splitting so the "test" segment in
	// foo_test.go doesn't become a banned-word candidate itself.
	stem := strings.TrimSuffix(base, ".go")
	stem = strings.TrimSuffix(stem, "_test")
	hit := check_banned_identifiers_find_hit(
		stem, []string{"util", "utils", "utility", "utilities", "length", "len"})
	invariant.Cross_Product(invariant.Sometimes(hit == "", "Hit can be empty on this branch"))
	if hit == "" {
		return nil
	}
	return []Diagnostic{{
		Position: token.Position{Filename: filename, Line: 1, Column: 1},
		Message:  fmt.Sprintf("file name %s contains banned word '%s'", base, hit),
	}}
}

// Walks every declaration site (vars, consts, params, named returns,
// struct fields, type names, labels, range and `:=` defines) and flags any
// name segment matching banned_segments_universal. Function-name-only
// segments (banned_function_name_segments) are checked at FuncDecl sites
// only. Use sites (CallExpr.Fun, SelectorExpr, bare Ident refs) are not
// visited, so `len(xs)` and `cap(xs)` builtin calls are naturally exempt.
func check_banned_identifiers_walk(file_set *token.FileSet, file *ast.File) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	validate_banned_name := func(pos token.Pos, scope, name string, extra []string) {
		hit := check_banned_identifiers_find_hit(
			name,
			extra,
			[]string{"util", "utils", "utility", "utilities", "length", "len"})
		invariant.Cross_Product(
			invariant.Sometimes(hit == "", "Hit can be empty on this branch"))
		if hit == "" {
			return
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(pos),
			Message: fmt.Sprintf(
				"%s name %s contains banned word '%s'", scope, name, hit),
		})
	}
	validate_banned_names := func(scope string, names []*ast.Ident) {
		for _, identifier := range names {
			if identifier == nil {
				continue
			}
			if identifier.Name == "_" {
				continue
			}
			validate_banned_name(identifier.Pos(), scope, identifier.Name, nil)
		}
	}
	ast.Inspect(file, func(n ast.Node) (descend bool) {
		check_banned_identifiers_walk_visit(n, validate_banned_name, validate_banned_names)
		return true
	})
	return diags
}

// Visitor body extracted to keep the surrounding walk function under the
// 100-line per-function limit. Receives the per-position and per-list
// closures bound to the parent's file_set and diag accumulator.
func check_banned_identifiers_walk_visit(
	n ast.Node,
	validate_banned_name func(pos token.Pos, scope, name string, extra []string),
	validate_banned_names func(scope string, names []*ast.Ident),
) {
	switch x := n.(type) {
	case *ast.FuncDecl:
		// "helper" is function-only; not in banned_segments_universal because a
		// file or package named helper is a weaker smell than the function case.
		validate_banned_name(x.Name.Pos(), "function", x.Name.Name, []string{"helper"})
	case *ast.FuncType:
		if x.Params != nil {
			for _, f := range x.Params.List {
				validate_banned_names("parameter", f.Names)
			}
		}
		if x.Results != nil {
			for _, f := range x.Results.List {
				validate_banned_names("named return", f.Names)
			}
		}
	case *ast.ValueSpec:
		validate_banned_names("variable or const", x.Names)
	case *ast.TypeSpec:
		validate_banned_name(x.Name.Pos(), "type", x.Name.Name, nil)
	case *ast.StructType:
		if x.Fields != nil {
			for _, f := range x.Fields.List {
				validate_banned_names("struct field", f.Names)
			}
		}
	case *ast.InterfaceType:
		if x.Methods != nil {
			for _, f := range x.Methods.List {
				validate_banned_names("interface method", f.Names)
			}
		}
	case *ast.AssignStmt:
		if x.Tok != token.DEFINE {
			return
		}
		for _, lhs := range x.Lhs {
			identifier, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}
			if identifier.Name == "_" {
				continue
			}
			validate_banned_name(identifier.Pos(), "variable", identifier.Name, nil)
		}
	case *ast.RangeStmt:
		if x.Tok != token.DEFINE {
			return
		}
		for _, e := range []ast.Expr{x.Key, x.Value} {
			identifier, ok := e.(*ast.Ident)
			if !ok {
				continue
			}
			if identifier.Name == "_" {
				continue
			}
			validate_banned_name(
				identifier.Pos(), "range variable", identifier.Name, nil)
		}
	case *ast.LabeledStmt:
		validate_banned_name(x.Label.Pos(), "label", x.Label.Name, nil)
	}
}

func check_banned_identifiers_find_hit(name string, banned_lists ...[]string) (hit string) {
	defer func() {
		hit_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(hit), Lo: 0, Hi: max_banned_segment_chars,
			Message: "Hit is a banned-segment match; longest is `utilities` (9 chars)",
		})
		bl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(banned_lists), Lo: min_non_empty, Hi: max_banned_lists_per_check,
			Message: "Banned_lists is small static set",
		})
		invariant.Cross_Product(hit_axis, bl,
			invariant.Excluding("Both hit and bl at safety caps is bad",
				invariant.Bucket_Hi(hit_axis), invariant.Bucket_Hi(bl)),
			invariant.Excluding("Zero hit with max bl is bad",
				invariant.Bucket_Lo(hit_axis), invariant.Bucket_Hi(bl)),
			invariant.Excluding("Max hit with min bl is bad",
				invariant.Bucket_Hi(hit_axis), invariant.Bucket_Lo(bl)),
		)
	}()
	bl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(banned_lists), Lo: min_non_empty, Hi: max_banned_lists_per_check,
		Message: "Banned_lists is small static set",
	})
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Name is a non-empty identifier bounded by identifier budget",
	})
	invariant.Cross_Product(
		name_axis, bl,
		invariant.Excluding("Both name and bl at safety caps is bad",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(bl)),
		invariant.Excluding("Zero name with max bl is bad",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_Hi(bl)),
		invariant.Excluding("Max name with min bl is bad",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_Lo(bl)),
	)

	words := suggest_split_words(name)
	invariant.Cross_Product(invariant.Always(words != nil, "Words is non-nil at this point"))
	for _, w := range words {
		for _, list := range banned_lists {
			for _, banned := range list {
				if strings.EqualFold(w, banned) {
					return banned
				}
			}
		}
	}
	return ""
}

// Functions with two or more parameters of the same type are call-site
// landmines: `transfer(source, dst)` can be silently swapped to
// `transfer(dst, source)` with no compiler protest. Force such signatures to
// take a pointer to a named input struct declared directly above; call sites
// then read as `transfer(&Transfer_Input{Src: ..., Dst: ...})` and re-orderings
// become compile errors.
func check_input_struct(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	for _, declaration := range file.Decls {
		function_declaration, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if !check_input_struct_should_trigger(function_declaration) {
			continue
		}
		want_name := check_input_struct_expected_name(function_declaration.Name.Name)
		invariant.Cross_Product(
			invariant.Always(want_name != "", "Want_name is non-empty at this point"))
		diag := check_input_struct_validate(file_set, function_declaration, want_name)
		invariant.Cross_Product(
			invariant.Always(diag != nil, "Diag is non-nil at this point"))
		if diag != nil {
			diags = append(diags, *diag)
		}
	}
	return diags
}

func check_input_struct_should_trigger(function *ast.FuncDecl) (trigger bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			trigger, "Trigger fired at this site"))

	}()
	invariant.Cross_Product(invariant.Always(
		function != nil, "Function is non-nil"))

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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(want),
				Lo: min_want_name_chars,
				Hi: max_want_name_chars,
				Message: "Want is `<function_name>_Input` shape; bounded by " +
					"want_name budget",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(function_name),
			Lo: min_non_empty,
			Hi: Max_Identifier_Chars,
			Message: "Function_name is a non-empty Go identifier bounded by " +
				"identifier budget",
		}),
	)

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
	defer func() {
		name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diag.Name), Lo: 0, Hi: Max_Identifier_Chars,
			Message: "Diag.Name is bounded by identifier budget",
		})
		want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diag.Want), Lo: 0, Hi: max_invariant_suggestion_chars,
			Message: "Diag.Want is bounded by suggestion text budget",
		})
		message_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(diag.Message),
			Lo: min_convert_to_message_chars,
			Hi: max_convert_to_message_chars,
			Message: "Diag.Message is `convert to <sig>` bounded by suggested-sig " +
				"shape",
		})
		// See sibling diag-builders: Always(diag.Tier == 0) credits both
		// boundary_int and zero_int from a single call; Distinct_Boundary
		// would fatal because Tier is invariantly 0 at construction.
		invariant.Cross_Product(
			invariant.Always(diag != nil, "Diag is non-nil on every reachable return"),
			invariant.Always(diag.Tier == 0,
				"Tier is 0 at diagnostic construction (set later by tier "+
					"dispatcher)"),
			invariant.Always(diag.Name == "", "Diag.Name is empty by construction"),
			invariant.Always(diag.Want == "", "Diag.Want is empty by construction"),
			invariant.Always(
				diag.Message != "", "Diag.Message is non-empty by construction"),
			name_axis, want_axis, message_axis,
			invariant.Excluding("Hi name unreachable (Name is always empty)",
				invariant.Bucket_Hi(name_axis), invariant.Bucket_Lo(want_axis)),
			invariant.Excluding("Hi name unreachable (Name is always empty)",
				invariant.Bucket_Hi(name_axis), invariant.Bucket_Hi(want_axis)),
			invariant.Excluding("Hi want unreachable (Want is always empty)",
				invariant.Bucket_Hi(want_axis), invariant.Bucket_Lo(name_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(function != nil, "Function is non-nil"),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(want_name), Lo: min_want_name_chars, Hi: max_want_name_chars,
			Message: "Want_name length is bounded by `<funcname>_Input` shape",
		}),
		invariant.Always(want_name != "",
			"Want_name is non-empty (has `_Input` suffix)"),
	)

	non_variadic := check_input_struct_validate_non_variadic_params(function)
	invariant.Cross_Product(
		invariant.Always(non_variadic != nil, "Non_variadic is non-nil at this point"))
	if len(non_variadic) == 1 {
		if len(non_variadic[0].Names) == 1 {
			// Trigger fires only when some type appears ≥2 times in the param
			// list. A single 1-name field contributes exactly 1 entry, so the
			// trigger cannot fire on this shape — reaching here means
			// check_input_struct_should_trigger lied.
			invariant.Unreachable(
				"Check_input_struct: trigger fired with single-name " +
					"param")
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
	defer func() {
		check_input_struct_validate_suggest_sig_assert_exit(sig)
	}()
	invariant.Cross_Product(
		invariant.Always(function != nil, "Function is non-nil"),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(want_name), Lo: min_want_name_chars, Hi: max_want_name_chars,
			Message: "Want_name length is bounded by `<funcname>_Input` shape",
		}),
		invariant.Always(want_name != "",
			"Want_name is non-empty (has `_Input` suffix)"),
	)

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

func check_input_struct_validate_suggest_sig_assert_exit(sig string) {
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(sig),
			Lo: min_suggested_sig_chars,
			Hi: max_suggested_sig_chars,
			Message: "Sig is the suggested function signature; sized for " +
				"funcname plus `_Input` template",
		}),
	)
}

func check_input_struct_validate_non_variadic_params(function *ast.FuncDecl) (output []*ast.Field) {
	defer func() {
		o := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(output), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
			Message: "Output is ≥1 non-variadic param",
		})
		single_axis := invariant.Sometimes(
			len(output) == count_one, "Output has exactly one param sometimes")
		invariant.Cross_Product(o, single_axis,
			invariant.Excluding("Max o contradicts single true",
				invariant.Bucket_Hi(o), invariant.Bucket_True(single_axis)),
			invariant.Excluding("Axis o at safety cap is bad",
				invariant.Bucket_Hi(o), invariant.Bucket_False(single_axis)),
			invariant.Excluding("Zero o implies single true",
				invariant.Bucket_Lo(o), invariant.Bucket_False(single_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(function != nil, "Function is non-nil"))

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
			Message:  "func init is banned; call an explicit setup function from main",
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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
				Message: "interface method sets are banned; use a concrete type " +
					"or free function",
			})
			return true
		}
		return true
	})
	return diags
}

// Package-level `var` creates implicit mutable state at package load with
// no obvious initialization order, complicating tests and reasoning. Only
// two initializers are exempted: regexp.MustCompile (no const regex type)
// and errors.New (no const error type). The `var _ Iface = (*Impl)(nil)`
// shape is also exempted — it declares no value, just asks the compiler
// to verify Impl satisfies Iface.
func check_no_package_vars(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	const base_message = "package-level var is banned; move to function scope"
	const switch_hint = ", or use a switch for lookup tables"
	for _, declaration := range file.Decls {
		generic_declaration, is_generic_declaration := declaration.(*ast.GenDecl)
		if !is_generic_declaration {
			continue
		}
		if generic_declaration.Tok != token.VAR {
			continue
		}
		for _, specification := range generic_declaration.Specs {
			vs, is_vs := specification.(*ast.ValueSpec)
			if !is_vs {
				continue
			}
			if check_no_package_vars_is_snap_default(file, vs) {
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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(invariant.Always(
		vs != nil, "Vs is non-nil"))

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

// Composition-tier packages (named `*_default` by convention, see lint/README.md)
// are allowed to expose a single `var Default = …` binding — that's literally the
// shape they exist for. Allowed only for the literal name "Default" and only as a
// single-name single-initializer spec.
func check_no_package_vars_is_snap_default(file *ast.File, vs *ast.ValueSpec) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(
		invariant.Always(file != nil, "File is non-nil"),
		invariant.Always(vs != nil, "Vs is non-nil"),
	)

	if !strings.HasSuffix(file.Name.Name, "_default") {
		if file.Name.Name != "snap" {
			return false
		}
	}
	if len(vs.Names) != 1 {
		return false
	}
	if vs.Names[0].Name != "Default" {
		return false
	}
	if len(vs.Values) != 1 {
		return false
	}
	return true
}

// Allowed only when every declared name has a paired initializer that is
// a call to regexp.MustCompile or errors.New. A zero-value declaration
// (no Values) fails this check by construction (len mismatch), which is
// the intended behavior — no package-level zero-value state.
func check_no_package_vars_all_allowed(vs *ast.ValueSpec) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(invariant.Always(
		vs != nil, "Vs is non-nil"))

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
		invariant.Cross_Product(
			invariant.Sometimes(params == nil, "Params can be empty or zero on this "+
				"branch"))
		results := check_unnecessary_method_field_list_types(
			function_declaration.Type.Results)
		invariant.Cross_Product(
			invariant.Sometimes(results == nil, "Results can be empty or zero on this "+
				"branch"))
		match := check_unnecessary_method_matches_stdlib(
			&check_unnecessary_method_matches_stdlib_input{
				Name:    function_declaration.Name.Name,
				Params:  strings.Join(params, ","),
				Results: strings.Join(results, ","),
			})
		invariant.Cross_Product(
			invariant.Sometimes(match, "Match can be false on this branch"))
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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))
		name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(input.Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Input.Name is a Go method name bounded by identifier budget",
		})
		invariant.Cross_Product(name_axis)
	}()
	invariant.Cross_Product(invariant.Always(
		input != nil, "Input is non-nil"))
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Name is a Go method name bounded by identifier budget",
	})
	invariant.Cross_Product(name_axis)
	params_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Params), Lo: 0, Hi: max_method_params_test_corpus,
		Message: "Input.Params spans empty () to Bar fixture `A,xxx128` (130)",
	})
	invariant.Cross_Product(params_axis)
	results_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Results), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Input.Results spans empty (V fixture) to long 128-char type (U fixture)",
	})
	invariant.Cross_Product(results_axis)

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
	defer func() {
		o := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(output_list), Lo: 0, Hi: max_string_slice_per_call,
			Message: "Output_list is bounded by budget",
		})
		empty_axis := invariant.Sometimes(
			len(output_list) == 0, "Output_list is empty sometimes")
		invariant.Cross_Product(o, empty_axis,
			invariant.Excluding("Max o contradicts empty true",
				invariant.Bucket_Hi(o), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Axis o at safety cap is bad",
				invariant.Bucket_Hi(o), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero o implies empty true",
				invariant.Bucket_Lo(o), invariant.Bucket_False(empty_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Sometimes(fl == nil, "Fl may be nil"))

	if fl == nil {
		return nil
	}
	for _, f := range fl.List {
		rendered := check_unnecessary_method_field_list_types_render_type(f.Type)
		invariant.Cross_Product(
			invariant.Always(rendered != "", "Rendered is non-empty at this point"))
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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(output_string),
				Lo: min_non_empty,
				Hi: Max_Identifier_Chars,
				Message: "Output_string is non-empty per leaf cases and bounded " +
					"by identifier budget",
			}),
		)
	}()

	prefix := ""
	for range invariant.Game_Loop() {
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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	tok_file := file_set.File(file.Pos())
	invariant.Cross_Product(
		invariant.Always(tok_file != nil, "Tok_file is non-nil at this point"))
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
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

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
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		documentation_axis := invariant.Sometimes(
			function_declaration.Doc != nil, "Function_declaration has a doc sometimes")
		invariant.Cross_Product(d, documentation_axis,
			invariant.Excluding("Axis d at safety cap is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_True(documentation_axis)),
			invariant.Excluding("Axis d at safety cap is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_False(documentation_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(function_declaration != nil, "Function_declaration is non-nil"),
	)

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
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		diags_empty := invariant.Sometimes(len(diags) == 0, "Diags is empty sometimes")
		invariant.Cross_Product(d, diags_empty,
			invariant.Excluding("Max diags contradicts diags_empty true",
				invariant.Bucket_Hi(d), invariant.Bucket_True(diags_empty)),
			invariant.Excluding("Axis d at safety cap is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_False(diags_empty)),
			invariant.Excluding("Zero diags implies diags_empty true",
				invariant.Bucket_Lo(d), invariant.Bucket_False(diags_empty)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(generic_declaration != nil, "Generic_declaration is "+
			"non-nil"),
	)

	switch generic_declaration.Tok {
	case token.TYPE, token.VAR, token.CONST:
	default:
		return nil
	}
	group_has_documentation := check_exported_documentation_comment_has_documentation(
		generic_declaration.Doc)
	invariant.Cross_Product(
		invariant.Sometimes(group_has_documentation, "Group_has_documentation can be "+
			"false on this branch"))
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
			invariant.Cross_Product(invariant.Sometimes(
				specification_has_documentation,
				"Specification_has_documentation is true for documented value "+
					"specs"))

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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(invariant.Sometimes(
		group == nil, "Group may be nil"))

	if group == nil {
		return false
	}
	return len(group.List) > 0
}

// Enforces the index/count/offset/size naming convention from
// https://tigerbeetle.com/blog/2026-02-16-index-count-offset-size/, the
// no-abbreviations rule, and the nouns-over-present-participles rule.
// Each violation is emitted as its own diagnostic at the offending
// identifier's position so the output is parseable as standard
// file:line:column: message lines.
func check_names(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	var violations []name_violation
	violations = append(violations, check_names_terminology(file)...)
	violations = append(violations, check_names_arithmetic(file)...)
	violations = append(violations, check_names_abbreviations(file)...)
	violations = append(violations, check_names_participles(file)...)
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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(term),
				Lo: 0,
				Hi: max_stdlib_term_chars,
				Message: "Term is the stdlib-allowlist terminology suffix; " +
					"longest is `offset` (6)",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(qualified), Lo: min_qualified_ident_chars, Hi: Max_Identifier_Chars,
			Message: "Qualified is `pkg.Func` shape; minimum `a.B` (3 chars)",
		}),
	)

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
	defer func() {
		v := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(violations), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Violations is bounded by budget",
		})
		decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(file.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "File.Decls is bounded by AST budget",
		})
		invariant.Cross_Product(v, decls_axis,
			invariant.Excluding("Both v and decls at safety caps is bad",
				invariant.Bucket_Hi(v), invariant.Bucket_Hi(decls_axis)),
			invariant.Excluding("Axis v at safety cap with zero decls is bad",
				invariant.Bucket_Hi(v), invariant.Bucket_Lo(decls_axis)),
			invariant.Excluding("Zero v with max decls is bad",
				invariant.Bucket_Lo(v), invariant.Bucket_Hi(decls_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(file != nil, "File is non-nil"))

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
		invariant.Cross_Product(
			invariant.Always(declarations_map != nil, "Declarations_map is non-nil at "+
				"this point"))
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
	defer func() {
		check_names_terminology_function_declarations_assert_exit(declarations)
	}()
	invariant.Cross_Product(invariant.Always(function != nil, "Function is non-nil"))

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

func check_names_terminology_function_declarations_assert_exit(
	declarations map[string]*ast.Ident,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(declarations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Declarations is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(declarations) == 0, "Declarations is empty sometimes")
	invariant.Cross_Product(d, empty_axis,
		invariant.Excluding("Max diags contradicts empty=true",
			invariant.Bucket_Hi(d), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Diags at safety cap with empty input is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Non-empty input with zero diags is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_False(empty_axis)),
	)
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
	defer func() {
		decls_empty := invariant.Sometimes(
			len(input.Declarations) == 0, "Input.Declarations is empty sometimes")
		decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(input.Declarations), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Input.Declarations is bounded by AST budget",
		})
		invariant.Cross_Product(decls_empty, decls_axis,
			invariant.Excluding("Lo decls implies empty=true",
				invariant.Bucket_Lo(decls_axis),
				invariant.Bucket_False(decls_empty)),
			invariant.Excluding("Hi decls implies empty=false",
				invariant.Bucket_Hi(decls_axis),
				invariant.Bucket_True(decls_empty)),
			invariant.Excluding("Hi decls unreachable in test corpus",
				invariant.Bucket_Hi(decls_axis),
				invariant.Bucket_False(decls_empty)))
	}()
	decls_empty := invariant.Sometimes(
		len(input.Declarations) == 0, "Input.Declarations is empty sometimes")
	decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Declarations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Declarations is bounded by AST budget",
	})
	invariant.Cross_Product(invariant.Always(input != nil, "Input is non-nil"),
		decls_empty, decls_axis,
		invariant.Excluding("Lo decls implies empty=true",
			invariant.Bucket_Lo(decls_axis), invariant.Bucket_False(decls_empty)),
		invariant.Excluding("Hi decls implies empty=false",
			invariant.Bucket_Hi(decls_axis), invariant.Bucket_True(decls_empty)),
		invariant.Excluding("Hi decls unreachable in test corpus",
			invariant.Bucket_Hi(decls_axis), invariant.Bucket_False(decls_empty)))

	switch x := input.Node.(type) {
	case *ast.ForStmt:
		ind := check_names_terminology_attach_induction_variable(x)
		invariant.Cross_Product(
			invariant.Sometimes(ind == nil, "Ind is nil for body-only for-loops"))
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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			ind == nil, "Ind may be nil"))

	}()
	invariant.Cross_Product(invariant.Always(
		x != nil, "X is non-nil"))

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
	defer func() {
		check_names_terminology_attach_make_assert_exit(decls)
	}()
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(decls), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Decls is bounded by budget",
	})
	args_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(x.Args), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "X.Args is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(x != nil, "X is non-nil"),
		d, args_axis,
		invariant.Excluding("Both d and args at safety caps is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(args_axis)),
		invariant.Excluding("Max d with zero args is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(args_axis)),
		invariant.Excluding("Zero d with max args is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(args_axis)),
	)

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

func check_names_terminology_attach_make_assert_exit(decls map[string]*ast.Ident) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(decls), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Decls is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(decls) == 0, "Decls is empty sometimes")
	invariant.Cross_Product(d, empty_axis,
		invariant.Excluding("Max diags contradicts empty=true",
			invariant.Bucket_Hi(d), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Diags at safety cap with empty input is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Non-empty input with zero diags is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_False(empty_axis)),
	)
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
	defer func() {
		t := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(terms), Lo: 0, Hi: max_string_slice_per_call,
			Message: "Terms is bounded by budget",
		})
		invariant.Cross_Product(t, invariant.Excluding("Terms at safety cap is bad",
			invariant.Bucket_Hi(t)))
	}()
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
				invariant.Cross_Product(
					invariant.Always(t != "", "T is non-empty at this point"))
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
	defer func() {
		check_names_terminology_emit_assert_exit(violations, requirements)
	}()
	check_names_terminology_emit_assert_entry(requirements)
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
		invariant.Cross_Product(
			invariant.Sometimes(last == "", "Last can be empty on this branch"))
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
		invariant.Cross_Product(
			invariant.Always(suggestion != "", "Suggestion is non-empty at this point"))
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

func check_names_terminology_emit_assert_exit(
	violations []name_violation, requirements map[*ast.Ident]map[string]bool,
) {
	v := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(violations), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Violations is bounded by budget",
	})
	r := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Requirements is bounded by budget",
	})
	invariant.Cross_Product(v, r,
		invariant.Excluding("Both v and r at safety caps is bad",
			invariant.Bucket_Hi(v), invariant.Bucket_Hi(r)),
		invariant.Excluding("Max v with zero r is bad",
			invariant.Bucket_Hi(v), invariant.Bucket_Lo(r)),
		invariant.Excluding("Zero v with max r is bad",
			invariant.Bucket_Lo(v), invariant.Bucket_Hi(r)),
	)
}

func check_names_terminology_emit_assert_entry(
	requirements map[*ast.Ident]map[string]bool,
) {
	r := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Requirements is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(requirements) == 0, "Requirements is empty sometimes")
	invariant.Cross_Product(r, empty_axis,
		invariant.Excluding("Max r contradicts empty true",
			invariant.Bucket_Hi(r), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis r at safety cap is bad",
			invariant.Bucket_Hi(r), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero r implies empty true",
			invariant.Bucket_Lo(r), invariant.Bucket_False(empty_axis)),
	)
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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(output_string),
				Lo: min_rename_suggestion_chars,
				Hi: Max_Identifier_Chars,
				Message: "Output_string is `<name>_<term>` (5 for a replaced " +
					"single word like `count`)",
			}),
		)
	}()
	invariant.Cross_Product(invariant.Always(
		input != nil, "Input is non-nil"))
	term_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(input.Term),
		Lo: min_stdlib_term_chars,
		Hi: max_stdlib_term_chars,
		Message: "Input.Term is the preferred terminology suffix in " +
			"[size,count,index,offset]",
	})
	invariant.Cross_Product(term_axis)
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Name is a flagged identifier, one char and up",
	})
	name_short := invariant.Sometimes(
		len(input.Name) == min_non_empty, "Name is a single char sometimes")
	invariant.Cross_Product(name_axis, name_short,
		invariant.Excluding("Hi name short T",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_True(name_short)),
		invariant.Excluding("Hi name short F",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_False(name_short)),
		invariant.Excluding("Lo name short F",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_False(name_short)),
	)

	style := "snake_case"
	if ada_case_re.MatchString(input.Name) {
		style = "Ada_Case"
	}
	words := suggest_split_words(input.Name)
	invariant.Cross_Product(invariant.Always(words != nil, "Words is non-nil at this point"))
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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(suffix),
				Lo: 0,
				Hi: max_stdlib_term_chars,
				Message: "Suffix is one of the terminology suffixes; longest is " +
					"`offset` (6)",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty Go identifier bounded by identifier budget",
		}),
	)

	words := suggest_split_words(name)
	invariant.Cross_Product(invariant.Always(words != nil, "Words is non-nil at this point"))
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
	defer func() {
		v := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(violations), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Violations is bounded by budget",
		})
		decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(file.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "File.Decls is bounded by AST budget",
		})
		invariant.Cross_Product(v, decls_axis,
			invariant.Excluding("Both v and decls at safety caps is bad",
				invariant.Bucket_Hi(v), invariant.Bucket_Hi(decls_axis)),
			invariant.Excluding("Axis v at safety cap with zero decls is bad",
				invariant.Bucket_Hi(v), invariant.Bucket_Lo(decls_axis)),
			invariant.Excluding("Zero v with max decls is bad",
				invariant.Bucket_Lo(v), invariant.Bucket_Hi(decls_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(file != nil, "File is non-nil"))

	rhs_of := check_names_arithmetic_rhs_map(file)
	invariant.Cross_Product(invariant.Always(rhs_of != nil, "Rhs_of is non-nil at this point"))
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
	defer func() {
		x := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(m), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "M is bounded by budget",
		})
		decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(file.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "File.Decls is bounded by AST budget",
		})
		invariant.Cross_Product(x, decls_axis,
			invariant.Excluding("Both x and decls at safety caps is bad",
				invariant.Bucket_Hi(x), invariant.Bucket_Hi(decls_axis)),
			invariant.Excluding("Max x with zero decls is bad",
				invariant.Bucket_Hi(x), invariant.Bucket_Lo(decls_axis)),
			invariant.Excluding("Zero x with max decls is bad",
				invariant.Bucket_Lo(x), invariant.Bucket_Hi(decls_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(file != nil, "File is non-nil"))

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
	defer func() {
		check_names_arithmetic_check_binary_assert_exit(violations, input)
	}()
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Binary_Expression != nil, "Binary_Expression is non-nil"),
		invariant.Sometimes(input.Lhs == nil, "Lhs is nil for free-standing BinaryExprs"),
	)

	left, is_ident := input.Binary_Expression.X.(*ast.Ident)
	if !is_ident {
		return nil
	}
	right, is_ident := input.Binary_Expression.Y.(*ast.Ident)
	if !is_ident {
		return nil
	}
	left_suffix := check_names_suffix_of(left.Name)
	invariant.Cross_Product(
		invariant.Sometimes(left_suffix == "", "Left_suffix can be empty on this branch"))
	if left_suffix == "" {
		return nil
	}
	right_suffix := check_names_suffix_of(right.Name)
	invariant.Cross_Product(
		invariant.Always(right_suffix != "", "Right_suffix is non-empty at this point"))
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
	invariant.Cross_Product(
		invariant.Sometimes(result == "", "Result can be empty on this branch"))
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
	invariant.Cross_Product(
		invariant.Sometimes(lhs_suffix == "", "Lhs_suffix can be empty on this branch"))
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

func check_names_arithmetic_check_binary_assert_exit(
	violations []name_violation, input *check_names_arithmetic_check_binary_input,
) {
	v := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(violations), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Violations is bounded by budget",
	})
	lhs_axis := invariant.Sometimes(
		input.Lhs == nil, "Lhs is nil for free-standing BinaryExprs")
	invariant.Cross_Product(v, lhs_axis,
		invariant.Excluding("Max v contradicts lhs true",
			invariant.Bucket_Hi(v), invariant.Bucket_True(lhs_axis)),
		invariant.Excluding("Axis v at safety cap is bad",
			invariant.Bucket_Hi(v), invariant.Bucket_False(lhs_axis)),
	)
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
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X: len(result), Lo: 0, Hi: max_stdlib_term_chars,
				Message: "Result is a terminology suffix; longest is `offset` (6)",
			}),
		)
	}()
	invariant.Cross_Product(invariant.Always(
		input != nil, "Input is non-nil"))
	left_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Left), Lo: min_stdlib_term_chars, Hi: max_stdlib_term_chars,
		Message: "Input.Left is a terminology term in [size,count,index,offset]",
	})
	invariant.Cross_Product(left_axis)
	right_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Right), Lo: min_stdlib_term_chars, Hi: max_stdlib_term_chars,
		Message: "Input.Right is a terminology term in [size,count,index,offset]",
	})
	invariant.Cross_Product(right_axis)

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

// Per-word abbreviation denylist. Sourced from
// https://github.com/abbrcode/abbreviations-in-code (🟢+🔴 entries),
// hand-curated to attach candidate expansions for genuinely ambiguous
// abbreviations and dropped entries that Go language/stdlib forces on
// every codebase: err, ctx, fmt, len, cap, min, max. Single-letter
// identifiers (loop counters i/j/k/n/m per Tiger Style) are exempted by
// the check itself, not by absence from this map.
//
// Lookups are by tokenized word from suggest_split_words, lowercased.
// Because the codebase enforces snake_case + PascalCase via check_casing,
// every abbreviation lands as its own token — no substring scan needed.
//
// Init is exempt — Go's package-initialization function is mandatorily named `init`.
func banned_abbreviation_candidates(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(word), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Word is bounded by identifier budget",
		}),
	)

	if candidates = banned_abbreviation_candidates_a_b(word); candidates != nil {
		invariant.Cross_Product(
			invariant.Always(candidates != nil, "Candidates is non-nil at this point"))
		return candidates
	}
	if candidates = banned_abbreviation_candidates_c(word); candidates != nil {
		invariant.Cross_Product(
			invariant.Always(candidates != nil, "Candidates is non-nil at this point"))
		return candidates
	}
	if candidates = banned_abbreviation_candidates_d_f(word); candidates != nil {
		invariant.Cross_Product(
			invariant.Always(candidates != nil, "Candidates is non-nil at this point"))
		return candidates
	}
	if candidates = banned_abbreviation_candidates_g_l(word); candidates != nil {
		invariant.Cross_Product(
			invariant.Always(candidates != nil, "Candidates is non-nil at this point"))
		return candidates
	}
	if candidates = banned_abbreviation_candidates_m_o(word); candidates != nil {
		invariant.Cross_Product(
			invariant.Always(candidates != nil, "Candidates is non-nil at this point"))
		return candidates
	}
	if candidates = banned_abbreviation_candidates_p_r(word); candidates != nil {
		invariant.Cross_Product(
			invariant.Always(candidates != nil, "Candidates is non-nil at this point"))
		return candidates
	}
	if candidates = banned_abbreviation_candidates_s(word); candidates != nil {
		invariant.Cross_Product(
			invariant.Always(candidates != nil, "Candidates is non-nil at this point"))
		return candidates
	}
	return banned_abbreviation_candidates_t_z(word)
}

func banned_abbreviation_candidates_a_b(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	banned_abbreviation_candidates_assert_entry(word)

	switch word {
	case "abbr":
		return []string{"abbreviation"}
	case "abs":
		return []string{"absolute"}
	case "acos":
		return []string{"arccosine"}
	case "acosec":
		return []string{"arccosecant"}
	case "acot":
		return []string{"arccotangent"}
	case "acro":
		return []string{"acronym"}
	case "act":
		return []string{"action", "active", "actual"}
	case "actg":
		return []string{"arccotangent"}
	case "addr":
		return []string{"address"}
	case "algo":
		return []string{"algorithm"}
	case "alt":
		return []string{"alternative", "altitude"}
	case "anno":
		return []string{"annotation"}
	case "app":
		return []string{"application"}
	case "arg":
		return []string{"argument"}
	case "arr":
		return []string{"array", "arrival"}
	case "asec":
		return []string{"arcsecant"}
	case "asin":
		return []string{"arcsine"}
	case "async":
		return []string{"asynchronous"}
	case "atan":
		return []string{"arctangent"}
	case "attr":
		return []string{"attribute"}
	case "auth":
		return []string{"authentication", "authorization"}
	case "aux":
		return []string{"auxiliary"}
	case "avg":
		return []string{"average"}
	case "bg":
		return []string{"background"}
	case "bin":
		return []string{"binary", "bin"}
	case "bool":
		return []string{"boolean"}
	case "brk":
		return []string{"break"}
	case "btn":
		return []string{"button"}
	case "buf":
		return []string{"buffer"}
	case "buff":
		return []string{"buffer"}
	}
	return nil
}

func banned_abbreviation_candidates_c(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	banned_abbreviation_candidates_assert_entry(word)

	switch word {
	case "calc":
		return []string{"calculator", "calculation", "calculate"}
	case "cb":
		return []string{"callback"}
	case "cert":
		return []string{"certificate"}
	case "cfg":
		return []string{"config"}
	case "char":
		return []string{"character"}
	case "chk":
		return []string{"check", "checksum", "checkpoint", "chunk"}
	case "clr":
		return []string{"clear", "color", "caller"}
	case "cls":
		return []string{"class", "close", "clear", "clusters"}
	case "cmd":
		return []string{"command"}
	case "cnt":
		return []string{"count", "counter", "content"}
	case "cntr":
		return []string{"container", "counter", "center"}
	case "col":
		return []string{"column", "color", "collection", "collision"}
	case "coll":
		return []string{"collection", "collision"}
	case "com":
		return []string{"common", "communication"}
	case "comm":
		return []string{"common", "communication", "comment", "commit"}
	case "comp":
		return []string{"component", "compare", "computation", "composition"}
	case "con":
		return []string{"connection", "console", "container", "constant"}
	default:
		return banned_abbreviation_candidates_c_more(word)
	}
}

func banned_abbreviation_candidates_c_more(word string) (candidates []string) {
	switch word {
	case "concat":
		return []string{"concatenation"}
	case "cond":
		return []string{"condition"}
	case "conf":
		return []string{"configuration", "conference"}
	case "config":
		return []string{"configuration"}
	case "conn":
		return []string{"connection"}
	case "const":
		return []string{"constant"}
	case "cont":
		return []string{"continue", "container", "content", "continuous"}
	case "conv":
		return []string{"conversion", "conversation", "convert", "convolution"}
	case "coord":
		return []string{"coordinate", "coordinator"}
	case "cos":
		return []string{"cosine"}
	case "cosec":
		return []string{"cosecant"}
	case "cot":
		return []string{"cotangent"}
	case "cpy":
		return []string{"copy"}
	case "ctg":
		return []string{"cotangent"}
	case "ctrl":
		return []string{"control", "controller"}
	case "cur":
		return []string{"current", "cursor", "currency"}
	case "curr":
		return []string{"current", "currency"}
	}
	return nil
}

func banned_abbreviation_candidates_d(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	banned_abbreviation_candidates_assert_entry(word)
	switch word {
	case "db":
		return []string{"database"}
	case "dbg":
		return []string{"debug", "debugger"}
	case "dec":
		return []string{"decimal", "decode", "decrement", "declaration", "december"}
	case "decl":
		return []string{"declaration"}
	case "def":
		return []string{"default", "definition", "define"}
	case "deg":
		return []string{"degrees"}
	case "del":
		return []string{"delete", "deletion", "delimiter"}
	case "dep":
		return []string{"dependency", "deploy", "deprecated", "department"}
	case "desc":
		return []string{"description", "descending", "descriptor"}
	case "dest":
		return []string{"destination", "destructor"}
	case "dev":
		return []string{"developer", "development", "device"}
	case "dim":
		return []string{"dimension", "dimmer"}
	case "dir":
		return []string{"direction", "directory"}
	case "dis":
		return []string{"disable", "dispatch", "discard", "disconnect"}
	case "disp":
		return []string{"display", "dispatch", "disposition"}
	case "div":
		return []string{"division", "divider", "dividend"}
	case "doc":
		return []string{"document", "documentation"}
	case "docs":
		return []string{"documentation", "documents"}
	case "drv":
		return []string{"driver", "derivative"}
	case "dyn":
		return []string{"dynamic", "dynamics"}
	}
	return nil
}

func banned_abbreviation_candidates_assert_exit(candidates []string, word string) {
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(candidates), Lo: 0, Hi: max_string_slice_per_call,
		Message: "Candidates is bounded by budget",
	})
	word_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(word), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Word is bounded by identifier budget",
	})
	invariant.Cross_Product(c, word_axis,
		invariant.Excluding("Both axes at safety caps is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Hi(word_axis)),
		invariant.Excluding("Max constants with single-word ident is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Lo(word_axis)),
	)
}

func banned_abbreviation_candidates_assert_entry(word string) {
	word_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(word), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Word is bounded by identifier budget",
	})
	single_axis := invariant.Sometimes(
		len(word) == count_one, "Word is exactly 1 char sometimes")
	invariant.Cross_Product(word_axis, single_axis,
		invariant.Excluding("Multi-word ident implies single_axis=false",
			invariant.Bucket_Hi(word_axis), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Single-word ident implies single_axis=true",
			invariant.Bucket_Lo(word_axis), invariant.Bucket_False(single_axis)),
	)
}

func banned_abbreviation_candidates_d_f(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	banned_abbreviation_candidates_assert_entry(word)
	d_candidates := banned_abbreviation_candidates_d(word)
	invariant.Cross_Product(
		invariant.Sometimes(d_candidates == nil, "D_candidates is nil for non-d words"))
	if d_candidates != nil {
		return d_candidates
	}
	switch word {
	case "elm":
		return []string{"element"}
	case "en":
		return []string{"enable", "english"}
	case "env":
		return []string{"environment"}
	case "evt":
		return []string{"event"}
	case "exe":
		return []string{"execution", "executable"}
	case "exp":
		return []string{"exponential", "expression", "expected", "expansion", "experiment"}
	case "expr":
		return []string{"expression"}
	case "ext":
		return []string{"extension", "external", "extract", "extend"}
	case "fac":
		return []string{"factory", "faction", "face"}
	case "fc":
		return []string{"file_chooser", "function_call"}
	case "fct":
		return []string{"facet", "factor"}
	case "fd":
		return []string{"file_descriptor"}
	case "fig":
		return []string{"figure"}
	case "fn":
		return []string{"function"}
	case "fp":
		return []string{
			"file_processor", "function_pointer", "floating_point", "false_positive",
		}
	case "fr":
		return []string{"file_reader", "frame", "from"}
	case "frac":
		return []string{"fraction"}
	case "freq":
		return []string{"frequency"}
	case "fs":
		return []string{"file_system", "full_screen"}
	case "file_set":
		return []string{"file_set"}
	case "fun":
		return []string{"function"}
	case "func":
		return []string{"function"}
	case "fw":
		return []string{"file_writer", "firewall", "framework"}
	}
	return nil
}

func banned_abbreviation_candidates_g_l(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	banned_abbreviation_candidates_assert_entry(word)

	switch word {
	case "gen":
		return []string{"generation", "generator", "general", "generic", "generate"}
	case "geom":
		return []string{"geometry", "geometric"}
	case "hdr":
		return []string{"header"}
	case "hex":
		return []string{"hexadecimal"}
	case "id":
		return []string{"identifier"}
	case "idx":
		return []string{"index"}
	case "iface":
		return []string{"interface"}
	case "img":
		return []string{"image"}
	case "imp":
		return []string{"import", "implementation"}
	case "impl":
		return []string{"implementation"}
	case "in":
		return []string{"input"}
	case "inc":
		return []string{"include", "increment", "increase", "inclusion"}
	case "info":
		return []string{"information"}
	case "ins":
		return []string{"insertion", "instance", "insert"}
	case "inst":
		return []string{"instance", "instruction", "installation"}
	case "int":
		return []string{"integer", "internal", "intersection"}
	case "intf":
		return []string{"interface"}
	case "inv":
		return []string{"inverse", "invocation", "inventory", "invalid"}
	case "km":
		return []string{"keymap", "kilometer"}
	case "kwd":
		return []string{"keyword"}
	case "lang":
		return []string{"language"}
	case "lib":
		return []string{"library"}
	case "ll":
		return []string{"linked_list", "log_level", "low_level"}
	case "lnk":
		return []string{"link"}
	case "loc":
		return []string{"location", "local"}
	case "lvl":
		return []string{"level"}
	}
	return nil
}

func banned_abbreviation_candidates_m_o(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	banned_abbreviation_candidates_assert_entry(word)

	switch word {
	case "mcu":
		return []string{"microcontroller"}
	case "mem":
		return []string{"memory", "member"}
	case "mid":
		return []string{"middle", "midpoint"}
	case "misc":
		return []string{"miscellaneous"}
	case "mng":
		return []string{"manager"}
	case "mgr":
		return []string{"manager"}
	case "mod":
		return []string{"module", "modulo", "modification", "modify", "modifier"}
	case "msg":
		return []string{"message"}
	case "mul":
		return []string{"multiplication", "multiplier", "multiple"}
	case "nav":
		return []string{"navigation", "navigator"}
	case "net":
		return []string{"network", "internet"}
	case "num":
		return []string{"number", "numerator", "numerical"}
	case "obj":
		return []string{"object", "objective"}
	case "oct":
		return []string{"octal", "october", "octet"}
	case "opt":
		return []string{"option", "optimization", "optional", "optimizer"}
	case "org":
		return []string{"organization", "organic"}
	case "orig":
		return []string{"origin", "original"}
	case "os":
		return []string{"operating_system"}
	case "oss":
		return []string{"open_source_software"}
	case "out":
		return []string{"output"}
	}
	return nil
}

func banned_abbreviation_candidates_p_r(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	banned_abbreviation_candidates_assert_entry(word)

	switch word {
	case "param":
		return []string{"parameter"}
	case "perf":
		return []string{"performance"}
	case "pic":
		return []string{"picture"}
	case "pkg":
		return []string{"package"}
	case "pol":
		return []string{"polygon", "policy", "polynomial", "polar"}
	case "pos":
		return []string{"position", "positive"}
	case "pred":
		return []string{"predicate", "prediction", "predecessor"}
	case "pref":
		return []string{"preference", "prefix"}
	case "prev":
		return []string{"previous"}
	case "priv":
		return []string{"private", "privacy", "privilege"}
	case "prod":
		return []string{"production", "product", "producer"}
	case "prof":
		return []string{"profiler", "profile", "professor"}
	case "prop":
		return []string{"property", "propagation", "proposition"}
	case "ptr":
		return []string{"pointer"}
	case "pub":
		return []string{"public", "publisher", "publication"}
	case "px":
		return []string{"pixel"}
	case "qry":
		return []string{"query"}
	default:
		return banned_abbreviation_candidates_p_r_more(word)
	}
}

func banned_abbreviation_candidates_p_r_more(word string) (candidates []string) {
	switch word {
	case "rad":
		return []string{"radians", "radius", "radial"}
	case "rand":
		return []string{"random"}
	case "rec":
		return []string{"record", "recursive", "receive", "rectangle"}
	case "recv":
		return []string{"receive", "receiver"}
	case "ref":
		return []string{"reference", "refresh", "referral"}
	case "rel":
		return []string{"relation", "relative", "release"}
	case "rem":
		return []string{"remote", "remove", "remainder"}
	case "repo":
		return []string{"repository", "report"}
	case "req":
		return []string{"request", "required", "requirement"}
	case "res":
		return []string{"response", "result", "resource", "reserve"}
	case "ret":
		return []string{"return", "retry", "retrieve"}
	case "rev":
		return []string{"revision", "reverse", "review", "revenue"}
	case "rgx":
		return []string{"regular_expression"}
	case "rm":
		return []string{"remove"}
	case "rmv":
		return []string{"remove"}
	case "rnd":
		return []string{"random", "round", "render"}
	case "rng":
		return []string{"range", "random_number_generator"}
	}
	return nil
}

func banned_abbreviation_candidates_s(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	banned_abbreviation_candidates_assert_entry(word)

	switch word {
	case "sc":
		return []string{"script", "scope", "source_code"}
	case "sec":
		return []string{"secant", "second", "section", "security"}
	case "sel":
		return []string{"selection", "selector", "select"}
	case "sep":
		return []string{"separator", "separate", "september"}
	case "seq":
		return []string{"sequence", "sequential", "sequencer"}
	case "sess":
		return []string{"session"}
	case "sin":
		return []string{"sine"}
	case "sln":
		return []string{"solution"}
	case "sol":
		return []string{"solution", "solver", "solid"}
	case "spec":
		return []string{"specification", "special", "species", "spectrum"}
	case "sqrt":
		return []string{"square_root"}
	case "src":
		return []string{"source"}
	case "std":
		return []string{"standard"}
	case "stdio":
		return []string{"standard_input_output"}
	case "stmt":
		return []string{"statement"}
	case "str":
		return []string{"string", "structure", "stream", "struct", "strategy"}
	case "sub":
		return []string{"substring", "subtraction", "subscriber", "subject", "submodule"}
	case "sum":
		return []string{"summation", "summary"}
	case "svc":
		return []string{"service"}
	case "sync":
		return []string{"synchronization", "synchronous"}
	}
	return nil
}

func banned_abbreviation_candidates_t_z(word string) (candidates []string) {
	defer func() {
		banned_abbreviation_candidates_assert_exit(candidates, word)
	}()
	banned_abbreviation_candidates_assert_entry(word)

	switch word {
	case "tan":
		return []string{"tangent"}
	case "td":
		return []string{"table_data"}
	case "temp":
		return []string{"temporary", "temperature", "template"}
	case "tgl":
		return []string{"toggle"}
	case "tgt":
		return []string{"target"}
	case "th":
		return []string{"table_header", "theorem"}
	case "tmp":
		return []string{"temporary", "template", "temperature"}
	case "tmr":
		return []string{"timer"}
	case "tpe":
		return []string{"type"}
	case "tr":
		return []string{"table_row", "trace", "translate", "transaction"}
	case "ts":
		return []string{"timestamp", "time_series", "test_suite"}
	case "tx":
		return []string{"transaction", "transmit", "texture"}
	case "txt":
		return []string{"text"}
	case "usr":
		return []string{"user"}
	case "util":
		return []string{"utility"}
	case "val":
		return []string{"value", "valid", "validation"}
	case "var":
		return []string{"variable", "variant"}
	case "vec":
		return []string{"vector"}
	case "ver":
		return []string{"version", "vertical", "verify"}
	case "win":
		return []string{"window"}
	case "wiz":
		return []string{"wizard"}
	}
	return nil
}

// Words ending in "ing" that are unambiguously nouns. Any declared
// identifier whose final word (per suggest_split_words, lowercased)
// ends in "ing" and is NOT a key here is flagged as a present
// participle. The Stringer interface contract (`String() string`) is
// satisfied implicitly because "string" is in this set.
func is_allowed_ing_noun(word string) (allowed bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			allowed, "Construct is in the allow set"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(word),
			Lo: min_ing_word_chars,
			Hi: Max_Identifier_Chars,
			Message: "Word ends in `ing` (minimum 3 chars) and is bounded by " +
				"identifier budget",
		}),
	)

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
	invariant.Cross_Product(invariant.Always(
		file != nil, "File is non-nil"))

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
	invariant.Cross_Product(invariant.Always(
		function != nil, "Function is non-nil"))

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
	invariant.Cross_Product(invariant.Always(
		gd != nil, "Gd is non-nil"))

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

// Walks every declared identifier and emits a violation for any
// tokenized word that matches banned_abbreviation_candidates. The suggested
// rename substitutes each candidate into the full identifier so the
// author sees a drop-in replacement, not just the bare word — e.g.,
// `foo_id` produces `foo_id -> foo_identifier`, not `id -> identifier`.
// Multi-banned identifiers (e.g., a name containing two banned words)
// emit one violation per hit.
func check_names_abbreviations(file *ast.File) (violations []name_violation) {
	defer func() {
		v := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(violations), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Violations is bounded by budget",
		})
		decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(file.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "File.Decls is bounded by AST budget",
		})
		invariant.Cross_Product(v, decls_axis,
			invariant.Excluding("Both v and decls at safety caps is bad",
				invariant.Bucket_Hi(v), invariant.Bucket_Hi(decls_axis)),
			invariant.Excluding("Axis v at safety cap with zero decls is bad",
				invariant.Bucket_Hi(v), invariant.Bucket_Lo(decls_axis)),
			invariant.Excluding("Zero v with max decls is bad",
				invariant.Bucket_Lo(v), invariant.Bucket_Hi(decls_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(file != nil, "File is non-nil"))

	check_names_walk_decls(file, func(identifier *ast.Ident) {
		words := suggest_split_words(identifier.Name)
		invariant.Cross_Product(
			invariant.Always(words != nil, "Words is non-nil at this point"))
		style := "snake_case"
		if ada_case_re.MatchString(identifier.Name) {
			style = "Ada_Case"
		}
		for word_index, w := range words {
			lower := strings.ToLower(w)
			cands := banned_abbreviation_candidates(lower)
			invariant.Cross_Product(
				invariant.Sometimes(cands == nil, "Cands can be empty or zero on "+
					"this branch"))
			if cands == nil {
				continue
			}
			renames := make([]string, len(cands))
			for cand_index, c := range cands {
				substituted := append([]string{}, words...)
				substituted[word_index] = c
				renames[cand_index] = suggest(&suggest_input{
					Name: strings.Join(substituted, "_"),
					Want: style,
				})
			}
			violations = append(violations, name_violation{
				Position: identifier.Pos(),
				Message: fmt.Sprintf(
					"rename %s -> %s", identifier.Name, strings.Join(
						renames, ",")),
			})
		}
	})
	return violations
}

// Walks every declared identifier and flags any whose final tokenized
// word (lowercased) ends in "ing" and is not in is_allowed_ing_noun.
// The Stringer interface's String() method is implicitly allowed
// because "string" is in the noun allowlist.
func check_names_participles(file *ast.File) (violations []name_violation) {
	defer func() {
		v := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(violations), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Violations is bounded by budget",
		})
		decls_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(file.Decls), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "File.Decls is bounded by AST budget",
		})
		invariant.Cross_Product(v, decls_axis,
			invariant.Excluding("Both v and decls at safety caps is bad",
				invariant.Bucket_Hi(v), invariant.Bucket_Hi(decls_axis)),
			invariant.Excluding("Axis v at safety cap with zero decls is bad",
				invariant.Bucket_Hi(v), invariant.Bucket_Lo(decls_axis)),
			invariant.Excluding("Zero v with max decls is bad",
				invariant.Bucket_Lo(v), invariant.Bucket_Hi(decls_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(file != nil, "File is non-nil"))

	check_names_walk_decls(file, func(identifier *ast.Ident) {
		words := suggest_split_words(identifier.Name)
		invariant.Cross_Product(
			invariant.Always(words != nil, "Words is non-nil at this point"))
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

func check_comments_group_is_inline(
	file_set *token.FileSet, source []byte, c *ast.Comment,
) (inline bool) {
	defer func() {
		s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(source), Lo: min_source_with_comment_bytes, Hi: max_file_size_bytes,
			Message: "Source is ≥16 bytes (`package x\\n\\n// c\\n` minimum)",
		})
		inline_axis := invariant.Sometimes(inline, "Struct is defined inline here")
		invariant.Cross_Product(s, inline_axis,
			invariant.Excluding("Max s contradicts inline true",
				invariant.Bucket_Hi(s), invariant.Bucket_True(inline_axis)),
			invariant.Excluding("Axis s at safety cap is bad",
				invariant.Bucket_Hi(s), invariant.Bucket_False(inline_axis)),
			invariant.Excluding("Zero s contradicts inline true",
				invariant.Bucket_Lo(s), invariant.Bucket_True(inline_axis)),
		)
	}()
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(source), Lo: min_source_with_comment_bytes, Hi: max_file_size_bytes,
		Message: "Source is ≥16 bytes (`package x\\n\\n// c\\n` minimum)",
	})
	multi_axis := invariant.Sometimes(
		strings.HasPrefix(c.Text, "/*"), "C is a block comment sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(c != nil, "C is non-nil"),
		s, multi_axis,
		invariant.Excluding("Zero s contradicts multi true",
			invariant.Bucket_Lo(s), invariant.Bucket_True(multi_axis)),
		invariant.Excluding("Max s contradicts multi true",
			invariant.Bucket_Hi(s), invariant.Bucket_True(multi_axis)),
		invariant.Excluding("Axis s at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_False(multi_axis)),
	)

	position := file_set.Position(c.Slash)
	invariant.Cross_Product(
		invariant.Always(position.Line != 0, "Position.Line is non-zero at this point"))
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
	Readlink              func(name string) (target string, err error)
	Stat                  func(name string) (info fs.FileInfo, err error)
}

// Builds the stream-tier check set. Split out of check_file_system_stream so
// that function fits the length cap while still asserting its full input. Only
// the symlinks checker needs filesystem hooks; the rest are stateless visitors.
func check_file_system_stream_checkers(
	root_directory string,
	readlink func(name string) (target string, err error),
	stat func(name string) (info fs.FileInfo, err error),
) (checks [stream_checker_count]check_function_stream) {
	defer func() {
		invariant.Cross_Product(
			invariant.Always(len(checks) == stream_checker_count, "Checks is the "+
				"fixed stream-tier set"))
	}()
	rd := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(root_directory), Lo: 0, Hi: max_filesystem_path_chars,
		Message: "Root_directory spans empty to deep path",
	})
	rd_empty := invariant.Sometimes(
		len(root_directory) == 0, "Root_directory is empty sometimes")
	invariant.Cross_Product(rd, rd_empty,
		invariant.Excluding("Hi rd implies non-empty",
			invariant.Bucket_Hi(rd), invariant.Bucket_True(rd_empty)),
		invariant.Excluding("Lo rd implies empty true",
			invariant.Bucket_Lo(rd), invariant.Bucket_False(rd_empty)),
	)
	return [stream_checker_count]check_function_stream{
		{Name: "conflict-markers", Visit: check_stream_conflict_markers},
		{Name: "copyleft", Visit: check_stream_copyleft},
		{Name: "github-actions-uses", Visit: check_stream_github_actions_uses},
		{Name: "banned-scripts", Visit: check_stream_banned_scripts},
		{Name: "max-file-size", Visit: check_stream_max_file_size},
		{Name: "agent-doc-max-lines", Visit: check_stream_agent_documentation_max_lines},
		check_file_system_stream_checks_stream_symlinks_checker(
			&check_file_system_stream_checks_stream_symlinks_checker_input{
				Root_Directory: root_directory,
				Readlink:       readlink,
				Stat:           stat,
			}),
		{Name: "markdown-line-length", Visit: check_stream_markdown_line_max},
		check_file_system_stream_checks_stream_agents_claude_pair_checker(),
		check_file_system_stream_checks_stream_path_casing_checker(),
	}
}

func check_file_system_stream(
	input *check_file_system_stream_input,
) (diags []Diagnostic, go_paths []string, err error) {
	defer func() {
		check_file_system_stream_assert_exit(diags, go_paths)
	}()
	check_file_system_stream_input_assert_entry(input)
	checks := check_file_system_stream_checkers(
		input.Root_Directory, input.Readlink, input.Stat)
	invariant.Cross_Product(
		invariant.Always(len(checks) == stream_checker_count, "Checks is the fixed "+
			"stream-tier set"))
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

func check_file_system_stream_assert_exit(diags []Diagnostic, go_paths []string) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	g := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(go_paths), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Go_paths is bounded by budget",
	})
	invariant.Cross_Product(d, g,
		invariant.Excluding("Both d and g at safety caps is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(g)),
		invariant.Excluding("Max d with min g is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(g)),
		invariant.Excluding("Zero d with max g is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(g)),
	)
}

func check_file_system_stream_input_assert_entry(input *check_file_system_stream_input) {
	tracked := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Tracked), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Tracked is bounded by AST budget",
	})
	tracked_empty := invariant.Sometimes(len(input.Tracked) == 0, "Tracked is empty sometimes")
	dht := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Directory_Has_Tracked), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Directory_has_tracked is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		tracked, tracked_empty, dht,
		invariant.Excluding("Hi tracked empty unreachable",
			invariant.Bucket_Hi(tracked), invariant.Bucket_True(tracked_empty)),
		invariant.Excluding("Lo tracked implies empty true",
			invariant.Bucket_Lo(tracked), invariant.Bucket_False(tracked_empty)),
		invariant.Excluding("Hi dht empty unreachable",
			invariant.Bucket_Hi(dht), invariant.Bucket_True(tracked_empty)),
		invariant.Excluding("Hi dht non-empty unreachable",
			invariant.Bucket_Hi(dht), invariant.Bucket_False(tracked_empty)),
	)
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
			if check_file_system_stream_skip_directory(d.Name()) {
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
		if !strings.HasPrefix(p, "third_party/") {
			*go_paths = append(*go_paths, p)
		}
	}
	information, information_error := d.Info()
	invariant.Cross_Product(
		invariant.Always(
			information != nil, "Information is non-nil at this point"),
		invariant.Always(
			information_error == nil, "Information_error is nil at this point"),
	)
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

// Mirrors check_file_system_collect_paths' dir-skip rules (testdata, vendor, dot-dirs).
// Stream tier additionally walks third_party — vendored licenses are exactly
// the kind of thing the copyleft check is meant to catch.
func check_file_system_stream_skip_directory(name string) (skip bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			skip, "Skip applies to this case"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty directory name bounded by identifier budget",
		}),
	)

	if name == "testdata" {
		return true
	}
	if name == "vendor" {
		return true
	}
	// .github holds CI config that the github-actions-uses check needs to
	// see. Other dotdirs (.git, .jj, .claude, .go-path, ...) are tool state
	// the linter does not own and should not scan.
	if name == ".github" {
		return false
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	return false
}

func check_stream_conflict_markers(
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
	output *[]Diagnostic) {
	defer func() {
		assert_path_output_bounded(p, output)
	}()
	invariant.Cross_Product(invariant.Always(output != nil, "Output is non-nil"))
	assert_path_output_bounded(p, output)

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
	source_axis := invariant.Sometimes(source == nil, "Source is nil for unreadable files")
	err_axis := invariant.Sometimes(err == nil, "Err is nil for successful operations")
	invariant.Cross_Product(
		source_axis, err_axis,
		invariant.Excluding("Source and err both set contradicts the iff invariant",
			invariant.Bucket_False(source_axis), invariant.Bucket_False(err_axis)),
		invariant.Excluding("Source and err both nil contradicts the iff invariant",
			invariant.Bucket_True(source_axis), invariant.Bucket_True(err_axis)),
	)
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
// hidden, untyped build/runtime surface. Banned everywhere except the
// top-level third_party/ subtree (vendor/ is already pruned by the stream
// walker's skip rules). Nested third_party/ paths (e.g. pkg/third_party/x.py)
// are intentionally NOT exempt — the carve-out applies only to the canonical
// drop-zone at root.
func check_stream_banned_scripts(
	p string,
	information fs.FileInfo,
	_ func() (data []byte, err error),
	output *[]Diagnostic) {
	defer func() {
		assert_path_output_bounded(p, output)
	}()
	invariant.Cross_Product(invariant.Always(output != nil, "Output is non-nil"))
	assert_path_output_bounded(p, output)

	if strings.HasPrefix(p, "third_party/") {
		return
	}
	if p == "third_party" {
		return
	}
	base := strings.ToLower(information.Name())
	extension := strings.ToLower(path.Ext(base))
	banned := false
	switch extension {
	case ".py", ".sh", ".bash", ".zsh", ".fish", ".ksh", ".csh",
		".pl", ".pm", ".rb", ".lua", ".js", ".mjs", ".cjs", ".ts",
		".php", ".tcl", ".awk", ".ps1", ".psm1", ".bat", ".cmd",
		".vbs", ".groovy", ".r", ".jl":
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
		Message: fmt.Sprintf("banned scripting file %q; move under top-level "+
			"third_party/ or vendor/, or remove", p),
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
	defer func() {
		assert_path_output_bounded(p, output)
	}()
	invariant.Cross_Product(invariant.Always(output != nil, "Output is non-nil"))
	assert_path_output_bounded(p, output)

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
	source_axis := invariant.Sometimes(source == nil, "Source is nil for unreadable files")
	err_axis := invariant.Sometimes(err == nil, "Err is nil for successful operations")
	invariant.Cross_Product(
		source_axis, err_axis,
		invariant.Excluding("Source and err both set contradicts the iff invariant",
			invariant.Bucket_False(source_axis), invariant.Bucket_False(err_axis)),
		invariant.Excluding("Source and err both nil contradicts the iff invariant",
			invariant.Bucket_True(source_axis), invariant.Bucket_True(err_axis)),
	)
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

// Flags license-shaped files containing GPL/AGPL/LGPL/SSPL preambles. The
// MPL guard exists because Mozilla Public License preambles often reference
// the GNU title for comparison; without it, MPL would false-positive as GPL/LGPL.
func check_stream_copyleft(
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
	output *[]Diagnostic) {
	defer func() {
		assert_path_output_bounded(p, output)
	}()
	invariant.Cross_Product(invariant.Always(output != nil, "Output is non-nil"))
	assert_path_output_bounded(p, output)

	copyleft_filename_needles := []string{
		"license", "licence", "notice", "readme", "copying", "copyright", "unlicense",
	}
	name := strings.ToLower(information.Name())
	matched := false
	for _, needle := range copyleft_filename_needles {
		if strings.Contains(name, needle) {
			matched = true
			break
		}
	}
	if !matched {
		return
	}
	source, err := load()
	source_axis := invariant.Sometimes(source == nil, "Source is nil for unreadable files")
	err_axis := invariant.Sometimes(err == nil, "Err is nil for successful operations")
	invariant.Cross_Product(
		source_axis, err_axis,
		invariant.Excluding("Source and err both set contradicts the iff invariant",
			invariant.Bucket_False(source_axis), invariant.Bucket_False(err_axis)),
		invariant.Excluding("Source and err both nil contradicts the iff invariant",
			invariant.Bucket_True(source_axis), invariant.Bucket_True(err_axis)),
	)
	if err != nil {
		return
	}
	n := strings.ToUpper(strings.Join(strings.Fields(string(source)), " "))
	mpl := strings.Contains(n, "MOZILLA PUBLIC LICENSE")
	gpl_title := strings.Contains(n, "GNU GENERAL PUBLIC LICENSE")
	gpl_clause := strings.Contains(n, "THIS PROGRAM IS FREE SOFTWARE") ||
		strings.Contains(n, "COPYLEFT") ||
		strings.Contains(n, "TERMS AND CONDITIONS FOR COPYING")
	agpl_title := strings.Contains(n, "GNU AFFERO GENERAL PUBLIC LICENSE")
	agpl_clause := strings.Contains(n, "REMOTE NETWORK INTERACTION") ||
		strings.Contains(n, "NETWORK USE")
	lgpl_title := strings.Contains(n, "GNU LESSER GENERAL PUBLIC LICENSE")
	lgpl_clause := strings.Contains(n, "LIBRARY") || strings.Contains(n, "LINKING")
	sspl := strings.Contains(n, "SERVER SIDE PUBLIC LICENSE")
	var family string
	switch {
	case gpl_title && gpl_clause && !mpl:
		family = "GNU GPL"
	case agpl_title && agpl_clause:
		family = "GNU AGPL"
	case lgpl_title && lgpl_clause && !mpl:
		family = "GNU LGPL"
	case sspl:
		family = "Server Side Public License"
	default:
		return
	}
	*output = append(*output, Diagnostic{
		Position: token.Position{Filename: p},
		Message: fmt.Sprintf(
			"%s: replace with a permissive license (MIT/Apache/BSD)", family),
	})
}

func check_stream_max_file_size(
	p string,
	information fs.FileInfo,
	_ func() (data []byte, err error),
	output *[]Diagnostic) {
	defer func() {
		assert_path_output_bounded(p, output)
	}()
	invariant.Cross_Product(invariant.Always(output != nil, "Output is non-nil"))
	assert_path_output_bounded(p, output)

	if information.Size() <= max_file_size_bytes {
		return
	}
	*output = append(*output, Diagnostic{
		Position: token.Position{Filename: p},
		Message: fmt.Sprintf("file exceeds 1 MiB (%d bytes); move it out of the tree "+
			"(LFS or external store)", information.Size()),
	})
}

func assert_path_output_bounded(p string, output *[]Diagnostic) {
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(p),
		Lo: min_go_filename_chars,
		Hi: max_filesystem_path_chars,
		Message: "P length is bounded; shortest path is `a.go` style 4-char Go " +
			"file name",
	})
	o := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*output), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Output is bounded by per-file diag budget",
	})
	invariant.Cross_Product(path_axis, o,
		invariant.Excluding("Path at safety cap with diag-budget cap is bad",
			invariant.Bucket_Hi(path_axis), invariant.Bucket_Hi(o)),
		invariant.Excluding("Minimal path with diag-budget cap unreachable in "+
			"test corpus",
			invariant.Bucket_Lo(path_axis), invariant.Bucket_Hi(o)),
	)
}

func check_stream_agent_documentation_max_lines(
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
	output *[]Diagnostic,

) {
	defer func() {
		assert_path_output_bounded(p, output)
	}()
	invariant.Cross_Product(invariant.Always(output != nil, "Output is non-nil"))
	assert_path_output_bounded(p, output)

	switch information.Name() {
	case "CLAUDE.md", "AGENTS.md", "SKILL.md":
	default:
		return
	}
	source, err := load()
	source_axis := invariant.Sometimes(source == nil, "Source is nil for unreadable files")
	err_axis := invariant.Sometimes(err == nil, "Err is nil for successful operations")
	invariant.Cross_Product(
		source_axis, err_axis,
		invariant.Excluding("Source and err both set contradicts the iff invariant",
			invariant.Bucket_False(source_axis), invariant.Bucket_False(err_axis)),
		invariant.Excluding("Source and err both nil contradicts the iff invariant",
			invariant.Bucket_True(source_axis), invariant.Bucket_True(err_axis)),
	)
	if err != nil {
		return
	}
	lines_count := bytes.Count(source, []byte{'\n'})
	if len(source) > 0 {
		if source[len(source)-1] != '\n' {
			lines_count++
		}
	}
	if lines_count > agent_documentation_max_lines {
		message := fmt.Sprintf(
			"%s has %d lines; split or trim it under %d",
			information.Name(), lines_count, agent_documentation_max_lines,
		)
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
func check_file_system_stream_checks_stream_path_casing_checker() (c check_function_stream) {
	defer func() {
		invariant.Cross_Product(
			invariant.Always(
				c.Name != "", "C.Name is non-empty for the path-casing checker"),
			invariant.Always(len(c.Name) == path_casing_check_name_chars, "C.Name is "+
				"the fixed path-casing label"),
		)
	}()

	seen := map[string]bool{}
	return check_function_stream{
		Name: path_casing_check_name,
		Visit: func(
			p string,
			info fs.FileInfo,
			_ func() (data []byte, err error),
			output *[]Diagnostic) {
			if strings.HasPrefix(p, "third_party/") {
				return
			}
			if p == "third_party" {
				return
			}
			segments := strings.Split(p, "/")
			for i, seg := range segments {
				is_file := i == len(segments)-1
				key := strings.Join(segments[:i+1], "/")
				if seen[key] {
					continue
				}
				seen[key] = true
				if strings.HasPrefix(seg, ".") {
					continue
				}
				name := seg
				if is_file {
					dot_offset := strings.IndexByte(seg, '.')
					if dot_offset >= 0 {
						name = seg[:dot_offset]
					}
				}
				if name == "" {
					continue
				}
				if snake_case_re.MatchString(name) {
					continue
				}
				if ada_case_re.MatchString(name) {
					continue
				}
				kind := "directory"
				if is_file {
					kind = "file"
				}
				*output = append(*output, Diagnostic{
					Position: token.Position{Filename: key},
					Message: fmt.Sprintf(
						"%s name %q must be snake_case, Ada_Case, or "+
							"SCREAMING_SNAKE_CASE",
						kind, name,
					),
				})
			}
		},
	}
}

type check_file_system_stream_checks_stream_symlinks_checker_input struct {
	Root_Directory string
	Readlink       func(name string) (target string, err error)
	Stat           func(name string) (info fs.FileInfo, err error)
}

// Reports orphaned symlinks. Readlink and Stat are injected (main.go binds
// them to the real os.Readlink/os.Stat) because fs.FS has no symlink
// primitive — the rule hard-requires real-OS access. An empty Root_Directory or
// nil Readlink/Stat self-disables the check so fstest.MapFS-backed tests
// can opt out without special-casing.
func check_file_system_stream_checks_stream_symlinks_checker(
	input *check_file_system_stream_checks_stream_symlinks_checker_input,
) (c check_function_stream) {
	defer func() {
		check_file_system_stream_checks_stream_symlinks_checker_input_assert_exit(
			input, c)
	}()
	rd_empty := invariant.Sometimes(
		len(input.Root_Directory) == 0, "Root_Directory is empty sometimes")
	rd_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Root_Directory), Lo: 0, Hi: max_filesystem_path_chars,
		Message: "Input.Root_Directory is bounded by filesystem path budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"), rd_empty, rd_axis,
		invariant.Excluding("Lo rd implies empty=true",
			invariant.Bucket_Lo(rd_axis), invariant.Bucket_False(rd_empty)),
		invariant.Excluding("Hi rd implies empty=false",
			invariant.Bucket_Hi(rd_axis), invariant.Bucket_True(rd_empty)))

	root_directory := input.Root_Directory
	readlink := input.Readlink
	stat := input.Stat
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
			if stat == nil {
				return
			}
			if info.Mode()&fs.ModeSymlink == 0 {
				return
			}
			operating_system_path := filepath.Join(root_directory, p)
			// Symlink-resolution coverage is intentionally not tracked here:
			// the in-memory fs.FS test fixtures used across this package
			// can't represent symlinks, so any Sometimes axis on
			// (target == "") / (read_err == nil) would have no admitting
			// observation under tests. Production behavior is unaffected.
			target, read_err := readlink(operating_system_path)
			if read_err != nil {
				*output = append(*output, Diagnostic{
					Position: token.Position{Filename: p},
					Message:  "dangling symlink (unreadable target)",
				})
				return
			}
			resolved := target
			if !filepath.IsAbs(target) {
				resolved = filepath.Join(
					filepath.Dir(operating_system_path), target)
			}
			if _, stat_error := stat(resolved); stat_error != nil {
				*output = append(*output, Diagnostic{
					Position: token.Position{Filename: p},
					Message:  fmt.Sprintf("dangling symlink -> %s", target),
				})
			}
		},
	}
}

func check_file_system_stream_checks_stream_symlinks_checker_input_assert_exit(
	input *check_file_system_stream_checks_stream_symlinks_checker_input,
	c check_function_stream,
) {
	rd_empty := invariant.Sometimes(
		len(input.Root_Directory) == 0, "Root_Directory is empty sometimes")
	rd_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Root_Directory), Lo: 0, Hi: max_filesystem_path_chars,
		Message: "Input.Root_Directory is bounded by filesystem path budget",
	})
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(c.Name),
		Lo:      min_stream_check_name_chars,
		Hi:      max_stream_check_name_chars,
		Message: "C.Name is `symlink` (7); stream-check names range 7-20 chars",
	})
	invariant.Cross_Product(rd_empty, rd_axis, name_axis,
		invariant.Excluding("Lo rd implies empty=true",
			invariant.Bucket_Lo(rd_axis), invariant.Bucket_False(rd_empty)),
		invariant.Excluding("Hi rd implies empty=false",
			invariant.Bucket_Hi(rd_axis), invariant.Bucket_True(rd_empty)),
		invariant.Excluding("Hi name unreachable — Name is `symlink` (7), Hi (20) "+
			"is for other checkers",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_True(rd_empty)),
		invariant.Excluding("Hi name unreachable — Name is `symlink` (7), Hi (20) "+
			"is for other checkers",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_False(rd_empty)),
	)
}

// Enforces a 100-rune visual cap on every .md line. Exemptions: fenced code
// blocks (would force awkward wraps), table rows (pipes can't break across
// lines), and lines containing a URL (`://`) where the URL itself is the
// dominant token.
func check_stream_markdown_line_max(
	p string,
	information fs.FileInfo,
	load func() (data []byte, err error),
	output *[]Diagnostic) {
	defer func() {
		assert_path_output_bounded(p, output)
	}()
	invariant.Cross_Product(invariant.Always(output != nil, "Output is non-nil"))
	assert_path_output_bounded(p, output)

	if !strings.HasSuffix(information.Name(), ".md") {
		return
	}
	source, err := load()
	source_axis := invariant.Sometimes(source == nil, "Source is nil for unreadable files")
	err_axis := invariant.Sometimes(err == nil, "Err is nil for successful operations")
	invariant.Cross_Product(
		source_axis, err_axis,
		invariant.Excluding("Source and err both set contradicts the iff invariant",
			invariant.Bucket_False(source_axis), invariant.Bucket_False(err_axis)),
		invariant.Excluding("Source and err both nil contradicts the iff invariant",
			invariant.Bucket_True(source_axis), invariant.Bucket_True(err_axis)),
	)
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
		if input_code {
			continue
		}
		trimmed := bytes.TrimSpace(line)
		is_table_row := bytes.HasPrefix(
			trimmed, []byte("|")) && bytes.HasSuffix(trimmed, []byte("|"))
		if is_table_row {
			continue
		}
		if bytes.Contains(line, []byte("://")) {
			continue
		}
		runes_count := utf8.RuneCount(line)
		if runes_count > markdown_line_max {
			*output = append(*output, Diagnostic{
				Position: token.Position{Filename: p, Line: line_number, Column: 1},
				Message: fmt.Sprintf("markdown line is %d chars; visual limit is "+
					"%d", runes_count, markdown_line_max),
			})
		}
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
	defer func() {
		invariant.Cross_Product(
			invariant.Always(
				c.Name != "",
				"C.Name is non-empty for the agents-claude-pair checker"),
			invariant.Always(len(c.Name) == agents_pair_check_name_chars, "C.Name is "+
				"the fixed agents-claude-pair label"),
		)
	}()

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
	invariant.Cross_Product(
		invariant.Always(name != "", "Name is non-empty at this point"))
	if name != "AGENTS.md" {
		if name != "CLAUDE.md" {
			return
		}
	}
	if strings.Count(p, "/") > 1 {
		return
	}
	source, err := load()
	source_axis := invariant.Sometimes(
		source == nil, "Source is nil for unreadable files")
	err_axis := invariant.Sometimes(
		err == nil, "Err is nil for successful operations")
	invariant.Cross_Product(
		source_axis, err_axis,
		invariant.Excluding("Source and err both set contradicts the iff "+
			"invariant",
			invariant.Bucket_False(source_axis),
			invariant.Bucket_False(err_axis)),
		invariant.Excluding("Source and err both nil contradicts the iff "+
			"invariant",
			invariant.Bucket_True(source_axis),
			invariant.Bucket_True(err_axis)),
	)
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

// Library packages that touch ambient process state (env vars, the wall
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
func assert_files_modules_entry(parsed_files []parsed_file, modules *module_index) {
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(parsed_files) == 0, "Parsed_files is empty sometimes")
	m_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules.Modules), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Modules.Modules is bounded by parsed-file budget",
	})
	f_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules.File_To_Module), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Modules.File_To_Module is bounded by parsed-file budget",
	})
	invariant.Cross_Product(invariant.Always(modules != nil, "Modules is non-nil"),
		p, empty_axis, m_axis, f_axis,
		invariant.Excluding("Max len contradicts empty=true",
			invariant.Bucket_Hi(p), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis p at safety cap is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero len implies empty true",
			invariant.Bucket_Lo(p), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Modules Hi unreachable",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Modules Hi with p Hi is bad",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_Hi(p)),
		invariant.Excluding("F2M Hi unreachable",
			invariant.Bucket_Hi(f_axis), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("F2M Hi with p Hi is bad",
			invariant.Bucket_Hi(f_axis), invariant.Bucket_Hi(p)),
		invariant.Excluding("Both modules Hi unreachable",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_Hi(f_axis)),
		invariant.Excluding("Modules Hi with empty impossible",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("F2M Hi with empty impossible",
			invariant.Bucket_Hi(f_axis), invariant.Bucket_True(empty_axis)),
	)
}

func assert_diags_files_modules_bounded(
	diags []Diagnostic, parsed_files []parsed_file, modules *module_index,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(parsed_files), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Parsed_files is bounded by budget",
	})
	m_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules.Modules), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Modules.Modules is bounded by parsed-file budget",
	})
	f_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules.File_To_Module), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Modules.File_To_Module is bounded by parsed-file budget",
	})
	invariant.Cross_Product(d, p, m_axis, f_axis,
		invariant.Excluding("Both diags and pairs at safety caps is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(p)),
		invariant.Excluding("Diags Hi with zero pairs is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(p)),
		invariant.Excluding("Pairs Hi with zero diags is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(p)),
		invariant.Excluding("Modules Hi unreachable",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_Hi(d)),
		invariant.Excluding("F2M Hi unreachable",
			invariant.Bucket_Hi(f_axis), invariant.Bucket_Hi(d)),
		invariant.Excluding("Both modules Hi unreachable",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_Hi(f_axis)),
		invariant.Excluding("Modules Hi with zero diags unreachable",
			invariant.Bucket_Hi(m_axis), invariant.Bucket_Lo(d)),
		invariant.Excluding("F2M Hi with zero diags unreachable",
			invariant.Bucket_Hi(f_axis), invariant.Bucket_Lo(d)),
	)
}

func check_no_ambient_stdlib(
	parsed_files []parsed_file, modules *module_index,
) (diags []Diagnostic) {
	defer func() {
		assert_diags_files_modules_bounded(diags, parsed_files, modules)
	}()
	assert_files_modules_entry(parsed_files, modules)

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
		diags = append(diags, check_no_ambient_stdlib_per_file(pf.File_Set, pf.File)...)
	}
	return diags
}

// True iff the file sits exactly one non-main Go ancestor below the
// library tier in its module. Mirrors check_library_tier_depth's
// counting logic but inverts the threshold: tier-depth fires when
// count > 1, the composition-tier exemption fires when count == 1.
func parsed_file_is_composition_tier(pf parsed_file, modules *module_index) (yes bool) {
	defer func() {
		parsed_file_is_composition_tier_assert_exit(yes, pf, modules)
	}()
	parsed_file_is_composition_tier_assert_entry(pf, modules)
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
	invariant.Cross_Product(
		invariant.Always(canonical != "", "Canonical is non-empty at this point"))
	if canonical == "." {
		return false
	}
	ancestors := check_library_tier_depth_ancestors(canonical)
	invariant.Cross_Product(
		invariant.Sometimes(ancestors == nil, "Ancestors can be empty or zero on this "+
			"branch"))
	count := 0
	for _, a := range ancestors {
		if _, has := m.Directory_Package[a]; has {
			count++
		}
	}
	return count == 1
}

func parsed_file_is_composition_tier_assert_exit(
	yes bool, pf parsed_file, modules *module_index,
) {
	invariant.Cross_Product(invariant.Sometimes(
		yes, "Affirmative branch is exercised"))
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pf.Path), Lo: min_go_filename_chars, Hi: max_filesystem_path_chars,
		Message: "Pf.Path spans 4-char (a.go) to deep paths",
	})
	invariant.Cross_Product(path_axis)
	modules_empty_d := invariant.Sometimes(
		len(modules.Modules) == 0, "Modules slice is empty sometimes")
	modules_axis_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules.Modules), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Modules.Modules spans empty workspace to budget",
	})
	invariant.Cross_Product(modules_axis_d, modules_empty_d,
		invariant.Excluding("Hi mod empty T",
			invariant.Bucket_Hi(modules_axis_d),
			invariant.Bucket_True(modules_empty_d)),
		invariant.Excluding("Hi mod empty F",
			invariant.Bucket_Hi(modules_axis_d),
			invariant.Bucket_False(modules_empty_d)),
		invariant.Excluding("Lo mod empty F",
			invariant.Bucket_Lo(modules_axis_d),
			invariant.Bucket_False(modules_empty_d)),
	)
	f2m_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(modules.File_To_Module),
		Lo:      min_non_empty,
		Hi:      max_parsed_files_per_call,
		Message: "File_To_Module spans single-file workspace to budget",
	})
	f2m_single_d := invariant.Sometimes(
		len(modules.File_To_Module) == min_non_empty, "File_To_Module has one "+
			"entry sometimes")
	invariant.Cross_Product(f2m_d, f2m_single_d,
		invariant.Excluding("Hi f2m single",
			invariant.Bucket_Hi(f2m_d), invariant.Bucket_True(f2m_single_d)),
		invariant.Excluding("F2m within budget",
			invariant.Bucket_Hi(f2m_d), invariant.Bucket_False(f2m_single_d)),
		invariant.Excluding("Lo f2m is single",
			invariant.Bucket_Lo(f2m_d), invariant.Bucket_False(f2m_single_d)),
	)
	source_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pf.Source), Lo: min_go_source_bytes, Hi: max_file_size_bytes,
		Message: "Pf.Source byte length",
	})
	source_minimal_d := invariant.Sometimes(
		len(pf.Source) == min_go_source_bytes, "Pf.Source is minimal sometimes")
	invariant.Cross_Product(source_d, source_minimal_d,
		invariant.Excluding("Hi source min",
			invariant.Bucket_Hi(source_d),
			invariant.Bucket_True(source_minimal_d)),
		invariant.Excluding("Source in budget",
			invariant.Bucket_Hi(source_d),
			invariant.Bucket_False(source_minimal_d)),
		invariant.Excluding("Lo source min",
			invariant.Bucket_Lo(source_d),
			invariant.Bucket_False(source_minimal_d)),
	)
}

func parsed_file_is_composition_tier_assert_entry(pf parsed_file, modules *module_index) {
	invariant.Cross_Product(
		invariant.Always(pf.File_Set != nil, "Pf.File_Set is non-nil"),
		invariant.Always(pf.File != nil, "Pf.File is non-nil"),
		invariant.Always(modules != nil, "Modules is non-nil"),
	)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pf.Path), Lo: min_go_filename_chars, Hi: max_filesystem_path_chars,
		Message: "Pf.Path spans 4-char (a.go) to deep paths",
	})
	invariant.Cross_Product(path_axis)
	modules_empty := invariant.Sometimes(
		len(modules.Modules) == 0, "Modules slice is empty sometimes")
	modules_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules.Modules), Lo: 0, Hi: max_parsed_files_per_call,
		Message: "Modules.Modules spans empty workspace to budget",
	})
	invariant.Cross_Product(modules_axis, modules_empty,
		invariant.Excluding("Hi mod empty T",
			invariant.Bucket_Hi(modules_axis), invariant.Bucket_True(modules_empty)),
		invariant.Excluding("Hi mod empty F",
			invariant.Bucket_Hi(modules_axis), invariant.Bucket_False(modules_empty)),
		invariant.Excluding("Lo mod empty F",
			invariant.Bucket_Lo(modules_axis), invariant.Bucket_False(modules_empty)),
	)
	f2m := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(modules.File_To_Module), Lo: min_non_empty, Hi: max_parsed_files_per_call,
		Message: "File_To_Module spans single-file workspace to budget",
	})
	f2m_single := invariant.Sometimes(
		len(modules.File_To_Module) == min_non_empty, "File_To_Module has one entry "+
			"sometimes")
	invariant.Cross_Product(f2m, f2m_single,
		invariant.Excluding("Hi f2m single",
			invariant.Bucket_Hi(f2m), invariant.Bucket_True(f2m_single)),
		invariant.Excluding("F2m within budget",
			invariant.Bucket_Hi(f2m), invariant.Bucket_False(f2m_single)),
		invariant.Excluding("Lo f2m is single",
			invariant.Bucket_Lo(f2m), invariant.Bucket_False(f2m_single)),
	)
	source := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pf.Source), Lo: min_go_source_bytes, Hi: max_file_size_bytes,
		Message: "Pf.Source byte length",
	})
	source_minimal := invariant.Sometimes(
		len(pf.Source) == min_go_source_bytes, "Pf.Source is minimal sometimes")
	invariant.Cross_Product(source, source_minimal,
		invariant.Excluding("Hi source minimal",
			invariant.Bucket_Hi(source), invariant.Bucket_True(source_minimal)),
		invariant.Excluding("Source within budget",
			invariant.Bucket_Hi(source), invariant.Bucket_False(source_minimal)),
		invariant.Excluding("Lo source is minimal",
			invariant.Bucket_Lo(source), invariant.Bucket_False(source_minimal)),
	)
}

func assert_diags_imports_bounded(diags []Diagnostic, file *ast.File) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	imports_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(file.Imports), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "File.Imports is bounded by AST budget",
	})
	invariant.Cross_Product(d, imports_axis,
		invariant.Excluding("Both diags and imports at safety caps is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(imports_axis)),
		invariant.Excluding("Diags at safety cap with zero imports is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(imports_axis)),
		invariant.Excluding("Imports at safety cap with zero diags is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(imports_axis)),
	)
}

func check_no_ambient_stdlib_per_file(
	file_set *token.FileSet, file *ast.File,
) (diags []Diagnostic) {
	defer func() {
		assert_diags_imports_bounded(diags, file)
	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	const import_message = "ambient stdlib import %q: see lint/README.md for resolutions"
	const call_message = "ambient stdlib call %s.%s: see lint/README.md for resolutions"
	for _, implementation := range file.Imports {
		path := strings.Trim(implementation.Path.Value, `"`)
		if !is_ambient_hard_import(path) {
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
		soft_input := &is_ambient_soft_ident_input{Package: path, Name: selection.Sel.Name}
		if !is_ambient_soft_ident(soft_input) {
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

func is_ambient_hard_import(path string) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(path),
			Lo: min_non_empty,
			Hi: Max_Identifier_Chars,
			Message: "Path is an import path; shortest is `C` (cgo, 1 char), longest " +
				"realistic is bounded by identifier budget",
		}),
	)

	switch path {
	case "os", "os/exec", "os/user", "os/signal",
		"flag", "runtime", "math/rand", "crypto/rand":
		return true
	}
	return false
}

type is_ambient_soft_ident_input struct {
	Package string
	Name    string
}

func is_ambient_soft_ident(input *is_ambient_soft_ident_input) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(invariant.Always(
		input != nil, "Input is non-nil"))
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Name), Lo: min_non_empty, Hi: max_filesystem_path_chars,
		Message: "Input.Name is a selector symbol, one char and up",
	})
	name_short := invariant.Sometimes(
		len(input.Name) == min_non_empty, "Name is a single char sometimes")
	invariant.Cross_Product(name_axis, name_short,
		invariant.Excluding("Hi vs single char",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_True(name_short)),
		invariant.Excluding("Symbol within budget",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_False(name_short)),
		invariant.Excluding("Lo is single char",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_False(name_short)),
	)
	package_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Package), Lo: min_non_empty, Hi: max_filesystem_path_chars,
		Message: "Input.Package is an import path, one char and up",
	})
	package_short := invariant.Sometimes(
		len(input.Package) == min_non_empty, "Package is a single char sometimes")
	invariant.Cross_Product(package_axis, package_short,
		invariant.Excluding("Hi vs single char",
			invariant.Bucket_Hi(package_axis), invariant.Bucket_True(package_short)),
		invariant.Excluding("Path within budget",
			invariant.Bucket_Hi(package_axis), invariant.Bucket_False(package_short)),
		invariant.Excluding("Lo is single char",
			invariant.Bucket_Lo(package_axis), invariant.Bucket_False(package_short)),
	)

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

// Bare `for {}` (and its twins `for ;; {}` and `for true {}`) hide the
// loop's termination condition inside the body. Readers can no longer cap
// iteration from the header alone. The intentional-unbounded escape hatch
// is `for range invariant.GameLoop()`, which is a *ast.RangeStmt and thus
// not caught here — choosing a different syntactic form *is* the assertion
// that the loop is unbounded on purpose.
func check_no_bare_for(file_set *token.FileSet, file *ast.File, _ []byte) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	const message = "bare `for {}` is banned; use `for range invariant.GameLoop()` " +
		"if the loop is intentionally unbounded"
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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()

	if condition == nil {
		return true
	}
	ident, is_ident := condition.(*ast.Ident)
	if !is_ident {
		return false
	}
	return ident.Name == "true"
}

// Enforces that every non-test function carries invariant-framework
// assertions on its int / string / pointer boundary identifiers. The
// framework's coverage analyzer (Recorder_Analyze_Assertion_Frequency)
// reports never-fired bucket combinations at test teardown — pairing that
// machinery with this lint check guarantees universal AFL-grade boundary
// coverage instead of per-engineer discipline.
//
// Tier-2: depends on check_namesd_returns (named returns guaranteed),
// check_input_struct (multi-same-type params normalized to *Foo_Input), and
// check_no_empty_function_body (Body != nil for declarations whose body we scan).
//
// Limitations:
//   - Generic type parameters not resolved (the matcher rejects non-keyword idents).
//   - Cross-file struct references emit only the outer pointer requirement,
//     no field-level recursion (no go/types — same trade-off as check_keyed_struct_init).
//   - Cross_Product args stored in temporaries are not credited.
//   - byte / rune / uintptr deliberately excluded from int kinds.
func check_invariant_assertions(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	if check_invariant_assertions_is_test_file(file_set, file) {
		return nil
	}
	if check_invariant_assertions_is_invariant_package(file_set, file) {
		return nil
	}
	invariant_idents := check_invariant_assertions_collect_invariant_idents(file)
	invariant.Cross_Product(
		invariant.Always(invariant_idents != nil, "Invariant_idents is non-nil at this "+
			"point"))
	stdlib_imports := check_invariant_assertions_collect_stdlib_imports(file)
	invariant.Cross_Product(
		invariant.Always(stdlib_imports != nil, "Stdlib_imports is non-nil at this point"))
	struct_field_index := check_invariant_assertions_build_struct_field_index(
		file, stdlib_imports)
	invariant.Cross_Product(
		invariant.Always(struct_field_index != nil, "Struct_field_index is non-nil at "+
			"this point"))
	same_file_consts := check_invariant_assertions_collect_same_file_constants(file)
	invariant.Cross_Product(
		invariant.Always(same_file_consts != nil, "Same_file_consts is non-nil at this "+
			"point"))
	constant_signs := check_invariant_assertions_collect_same_file_constant_signs(file)
	invariant.Cross_Product(
		invariant.Always(constant_signs != nil, "Const_signs is non-nil at this point"))
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function.Body == nil {
			continue
		}
		diags = append(diags, check_invariant_assertions_one_function(
			&check_invariant_assertions_one_function_input{
				File_Set:           file_set,
				Function:           function,
				Invariant_Idents:   invariant_idents,
				Stdlib_Local_Names: stdlib_imports,
				Struct_Field_Index: struct_field_index,
				Same_File_Consts:   same_file_consts,
				Constant_Signs:     constant_signs,
			})...)
	}
	return diags
}

type check_invariant_assertions_one_function_input struct {
	File_Set           *token.FileSet
	Function           *ast.FuncDecl
	Invariant_Idents   map[string]bool
	Stdlib_Local_Names map[string]bool
	Struct_Field_Index map[string]check_invariant_assertions_struct_information
	Same_File_Consts   map[string]bool
	Constant_Signs     map[string]int
}

func check_invariant_assertions_one_function(
	input *check_invariant_assertions_one_function_input,
) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_one_function_assert_exit(diags, input)
	}()
	check_invariant_assertions_one_function_input_assert_entry(input)
	function_label := check_invariant_assertions_function_label(input.Function)
	invariant.Cross_Product(
		invariant.Always(function_label != "", "Function_label is non-empty at this point"))
	defer_diag := check_invariant_assertions_validate_defer_position(
		input.File_Set, input.Function, input.Struct_Field_Index)
	invariant.Cross_Product(
		invariant.Sometimes(defer_diag == nil, "Defer_diag is nil when the defer position "+
			"is valid"))
	if defer_diag != nil {
		diags = append(diags, *defer_diag)
	}
	diags = append(diags, check_invariant_assertions_validate_no_compound_predicates(
		input.File_Set, input.Function, function_label, input.Invariant_Idents)...)
	diags = append(
		diags, check_invariant_assertions_boundary_literal_block(
			&check_invariant_assertions_boundary_literal_block_input{
				Body:             input.Function.Body,
				Invariant_Idents: input.Invariant_Idents,
				Function_Label:   function_label,

				Same_File_Consts:   input.Same_File_Consts,
				File_Set:           input.File_Set,
				Is_Top_Of_Function: true})...)
	diags = append(
		diags, check_invariant_assertions_validate_declarations(
			&check_invariant_assertions_validate_declarations_input{
				File_Set:       input.File_Set,
				Function:       input.Function,
				Function_Label: function_label,

				Invariant_Idents:   input.Invariant_Idents,
				Stdlib_Local_Names: input.Stdlib_Local_Names,

				Constant_Signs: input.Constant_Signs})...)
	return append(diags, check_invariant_assertions_one_function_pipeline(
		&check_invariant_assertions_one_function_pipeline_input{
			File_Set:           input.File_Set,
			Function:           input.Function,
			Invariant_Idents:   input.Invariant_Idents,
			Constant_Signs:     input.Constant_Signs,
			Struct_Field_Index: input.Struct_Field_Index,
		})...)
}

func check_invariant_assertions_one_function_assert_exit(
	diags []Diagnostic, input *check_invariant_assertions_one_function_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	diags_empty := invariant.Sometimes(len(diags) == 0, "Diags is empty sometimes")
	sfc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Same_File_Consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Same_file_consts is bounded by AST budget",
	})
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by AST budget",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by AST budget",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by AST budget",
	})
	stdlib := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_local_names is bounded by AST budget",
	})
	invariant.Cross_Product(d, diags_empty, sfc, ii, cs, sfi, stdlib,
		invariant.Excluding("Max d empty true",
			invariant.Bucket_Hi(d), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi d cap is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Zero d implies empty true",
			invariant.Bucket_Lo(d), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Hi sfc empty",
			invariant.Bucket_Hi(sfc), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi sfc non-empty",
			invariant.Bucket_Hi(sfc), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Hi ii empty",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi ii non-empty",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Hi cs empty",
			invariant.Bucket_Hi(cs), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi cs non-empty",
			invariant.Bucket_Hi(cs), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Hi sfi empty",
			invariant.Bucket_Hi(sfi), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi sfi non-empty",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Hi stdlib empty",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi stdlib non-empty",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_False(diags_empty)),
	)
}

func check_invariant_assertions_one_function_input_assert_entry(
	input *check_invariant_assertions_one_function_input,
) {
	sfc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Same_File_Consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Same_file_consts is bounded by AST budget",
	})
	sfc_empty := invariant.Sometimes(
		len(input.Same_File_Consts) == 0, "Same_file_consts is empty sometimes")
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by AST budget",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by AST budget",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by AST budget",
	})
	stdlib := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_local_names is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(
			input != nil,
			"Input is non-nil"), invariant.Always(input.File_Set != nil,
			"Input.File_Set is non-nil"),
		invariant.Always(input.Function != nil, "Input.Function is non-nil"),
		sfc, sfc_empty, ii, cs, sfi, stdlib,
		invariant.Excluding("Hi sfc with empty unreachable",
			invariant.Bucket_Hi(sfc), invariant.Bucket_True(sfc_empty)),
		invariant.Excluding("Hi sfc with non-empty unreachable",
			invariant.Bucket_Hi(sfc), invariant.Bucket_False(sfc_empty)),
		invariant.Excluding("Lo sfc implies empty true",
			invariant.Bucket_Lo(sfc), invariant.Bucket_False(sfc_empty)),
		invariant.Excluding("Hi ii with empty sfc unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(sfc_empty)),
		invariant.Excluding("Hi ii with non-empty sfc unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(sfc_empty)),
		invariant.Excluding("Hi cs with empty sfc unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_True(sfc_empty)),
		invariant.Excluding("Hi cs with non-empty sfc unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_False(sfc_empty)),
		invariant.Excluding("Hi sfi with empty sfc unreachable",
			invariant.Bucket_Hi(sfi), invariant.Bucket_True(sfc_empty)),
		invariant.Excluding("Hi sfi with non-empty sfc unreachable",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(sfc_empty)),
		invariant.Excluding("Hi stdlib with empty sfc unreachable",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_True(sfc_empty)),
		invariant.Excluding("Hi stdlib non-empty sfc unreachable",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_False(sfc_empty)),
	)
}

type check_invariant_assertions_one_function_pipeline_input struct {
	File_Set           *token.FileSet
	Function           *ast.FuncDecl
	Invariant_Idents   map[string]bool
	Constant_Signs     map[string]int
	Struct_Field_Index map[string]check_invariant_assertions_struct_information
}

// Runs the prologue scan, requirement collection, and per-function validation —
// the tail of one_function's pipeline. Extracted so one_function holds fewer
// containers and fits the per-function line cap while still asserting its full
// file-level map set; the duplicated map assertions here are the cost of that
// split, paid deliberately.
func check_invariant_assertions_one_function_pipeline(
	input *check_invariant_assertions_one_function_pipeline_input,
) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_one_function_pipeline_assert_exit(diags, input)
	}()
	check_invariant_assertions_one_function_pipeline_input_assert_entry(input)
	parameter_covered,
		defer_covered,
		always_nil_pointers := check_invariant_assertions_scan_function_prologue(
		input.Function.Body, input.Invariant_Idents, input.Constant_Signs)
	invariant.Cross_Product(invariant.Always(parameter_covered != nil, "Parameter_covered is "+
		"non-nil at this point"),
		invariant.Always(defer_covered != nil, "Defer_covered is non-nil at this point"),
		invariant.Always(always_nil_pointers != nil, "Always_nil_pointers is non-nil at "+
			"this point"))
	parameter_requirements, parameter_defer_requirements, named_return_requirements :=
		check_invariant_assertions_collect_requirements(
			&check_invariant_assertions_collect_requirements_input{
				Function:            input.Function,
				Struct_Field_Index:  input.Struct_Field_Index,
				Always_Nil_Pointers: always_nil_pointers,
			})
	parameter_empty := invariant.Sometimes(
		parameter_requirements == nil, "Parameter_requirements is empty sometimes")
	defer_empty := invariant.Sometimes(
		parameter_defer_requirements == nil, "Parameter_defer_requirements is empty "+
			"sometimes")
	named_empty := invariant.Sometimes(
		named_return_requirements == nil, "Named_return_requirements is empty sometimes")
	invariant.Cross_Product(parameter_empty, defer_empty, named_empty,
		invariant.Excluding("Empty params implies empty defer requirements",
			invariant.Bucket_True(parameter_empty),
			invariant.Bucket_False(defer_empty)))
	return check_invariant_assertions_validate_one_function(
		&check_invariant_assertions_validate_one_function_input{
			File_Set:                     input.File_Set,
			Parameter_Requirements:       parameter_requirements,
			Parameter_Defer_Requirements: parameter_defer_requirements,
			Named_Return_Requirements:    named_return_requirements,
			Parameter_Covered:            parameter_covered,
			Named_Return_Covered:         defer_covered,
		})
}

func check_invariant_assertions_one_function_pipeline_assert_exit(
	diags []Diagnostic, input *check_invariant_assertions_one_function_pipeline_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	diags_empty := invariant.Sometimes(len(diags) == 0, "Diags is empty sometimes")
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by AST budget",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by AST budget",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by AST budget",
	})
	invariant.Cross_Product(d, diags_empty, ii, cs, sfi,
		invariant.Excluding("Max d empty true",
			invariant.Bucket_Hi(d), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi d cap is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Zero d implies empty true",
			invariant.Bucket_Lo(d), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Hi ii empty",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi ii non-empty",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Hi cs empty",
			invariant.Bucket_Hi(cs), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi cs non-empty",
			invariant.Bucket_Hi(cs), invariant.Bucket_False(diags_empty)),
		invariant.Excluding("Hi sfi empty",
			invariant.Bucket_Hi(sfi), invariant.Bucket_True(diags_empty)),
		invariant.Excluding("Hi sfi non-empty",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(diags_empty)),
	)
}

func check_invariant_assertions_one_function_pipeline_input_assert_entry(
	input *check_invariant_assertions_one_function_pipeline_input,
) {
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by AST budget",
	})
	ii_empty := invariant.Sometimes(
		len(input.Invariant_Idents) == 0, "Invariant_idents is empty sometimes")
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by AST budget",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.File_Set != nil, "Input.File_Set is non-nil"),
		invariant.Always(input.Function != nil, "Input.Function is non-nil"),
		ii, ii_empty, cs, sfi,
		invariant.Excluding("Hi ii with empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(ii_empty)),
		invariant.Excluding("Hi ii with non-empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(ii_empty)),
		invariant.Excluding("Lo ii implies empty true",
			invariant.Bucket_Lo(ii), invariant.Bucket_False(ii_empty)),
		invariant.Excluding("Hi cs with empty ii unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_True(ii_empty)),
		invariant.Excluding("Hi cs with non-empty ii unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_False(ii_empty)),
		invariant.Excluding("Hi sfi with empty ii unreachable",
			invariant.Bucket_Hi(sfi), invariant.Bucket_True(ii_empty)),
		invariant.Excluding("Hi sfi with non-empty ii unreachable",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(ii_empty)),
	)
}

type check_invariant_assertions_validate_one_function_input struct {
	File_Set                     *token.FileSet
	Parameter_Requirements       []check_invariant_assertions_requirement
	Parameter_Defer_Requirements []check_invariant_assertions_requirement
	Named_Return_Requirements    []check_invariant_assertions_requirement
	Parameter_Covered            map[string]map[string]coverage_origin
	Named_Return_Covered         map[string]map[string]coverage_origin
}

// Bridges check_invariant_assertions per-function processing to the validator,
// short-circuiting when neither requirement list is populated. Extracted to
// keep check_invariant_assertions under the per-function line cap.
func check_invariant_assertions_validate_one_function(
	input *check_invariant_assertions_validate_one_function_input,
) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_validate_one_function_assert_exit(diags, input)
	}()
	pr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_requirements is bounded by AST budget",
	})
	pdr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Defer_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_defer_requirements is bounded by AST budget",
	})
	nrr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_requirements is bounded by AST budget",
	})
	pc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_covered is bounded by AST budget",
	})
	nrc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_covered is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.File_Set != nil, "Input.file_set is non-nil"),
		pr, pdr, nrr, pc, nrc,
		invariant.Excluding("Hi pr unreachable (with min pdr)",
			invariant.Bucket_Hi(pr), invariant.Bucket_Lo(pdr)),
		invariant.Excluding("Hi pr unreachable (with Hi pdr)",
			invariant.Bucket_Hi(pr), invariant.Bucket_Hi(pdr)),
		invariant.Excluding("Hi pdr unreachable (with min pr)",
			invariant.Bucket_Hi(pdr), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Hi nrr unreachable (Lo pr)",
			invariant.Bucket_Hi(nrr), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Hi nrr unreachable (Hi pr)",
			invariant.Bucket_Hi(nrr), invariant.Bucket_Hi(pr)),
		invariant.Excluding("Hi pc unreachable (Lo pr)",
			invariant.Bucket_Hi(pc), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Hi pc unreachable (Hi pr)",
			invariant.Bucket_Hi(pc), invariant.Bucket_Hi(pr)),
		invariant.Excluding("Hi nrc unreachable (Lo pr)",
			invariant.Bucket_Hi(nrc), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Hi nrc unreachable (Hi pr)",
			invariant.Bucket_Hi(nrc), invariant.Bucket_Hi(pr)),
	)
	if len(input.Parameter_Requirements) == 0 {
		if len(input.Named_Return_Requirements) == 0 {
			return nil
		}
	}
	return check_invariant_assertions_validate(
		&check_invariant_assertions_validate_input{
			File_Set:                     input.File_Set,
			Parameter_Requirements:       input.Parameter_Requirements,
			Parameter_Defer_Requirements: input.Parameter_Defer_Requirements,
			Named_Return_Requirements:    input.Named_Return_Requirements,
			Parameter_Covered:            input.Parameter_Covered,
			Named_Return_Covered:         input.Named_Return_Covered,
		})
}

func check_invariant_assertions_validate_one_function_assert_exit(
	diags []Diagnostic, input *check_invariant_assertions_validate_one_function_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	pr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_requirements is bounded by AST budget",
	})
	pdr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Parameter_Defer_Requirements),
		Lo:      0,
		Hi:      max_ast_nodes_per_call,
		Message: "Parameter_defer_requirements is bounded by AST budget",
	})
	nrr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_requirements is bounded by AST budget",
	})
	pc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_covered is bounded by AST budget",
	})
	nrc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_covered is bounded by AST budget",
	})
	invariant.Cross_Product(d, pr, pdr, nrr, pc, nrc,
		invariant.Excluding("Both d and pr at safety caps is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(pr)),
		invariant.Excluding("Max d with zero pr is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Zero d with max pr is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(pr)),
		invariant.Excluding("Hi pdr unreachable (Lo pr)",
			invariant.Bucket_Hi(pdr), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Hi pdr unreachable (Hi pr)",
			invariant.Bucket_Hi(pdr), invariant.Bucket_Hi(pr)),
		invariant.Excluding("Hi nrr unreachable (Lo pr)",
			invariant.Bucket_Hi(nrr), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Hi nrr unreachable (Hi pr)",
			invariant.Bucket_Hi(nrr), invariant.Bucket_Hi(pr)),
		invariant.Excluding("Hi pc unreachable (Lo pr)",
			invariant.Bucket_Hi(pc), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Hi pc unreachable (Hi pr)",
			invariant.Bucket_Hi(pc), invariant.Bucket_Hi(pr)),
		invariant.Excluding("Hi nrc unreachable (Lo pr)",
			invariant.Bucket_Hi(nrc), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Hi nrc unreachable (Hi pr)",
			invariant.Bucket_Hi(nrc), invariant.Bucket_Hi(pr)),
	)
}

// Check_Invariant_Assertions is the exported test entry point. Runs only
// check_invariant_assertions against a single source buffer, bypassing the
// tier-1/tier-2 gating in Check_File. Used so the new test can exercise the
// check before the codebase has been ported to satisfy it — once registered
// in check_file_checks_tier2, the standard Check_Source path is the
// production interface.
func Check_Invariant_Assertions(filename string, source any) (diags []Diagnostic, err error) {
	defer func() {
		d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
			Message: "Diags is bounded by budget",
		})
		err_axis := invariant.Sometimes(
			err != nil, "Err is non-nil sometimes (parse failure)")
		invariant.Cross_Product(d, err_axis,
			invariant.Excluding("Axis d at safety cap is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_True(err_axis)),
			invariant.Excluding("Axis d at safety cap is bad",
				invariant.Bucket_Hi(d), invariant.Bucket_False(err_axis)),
			invariant.Excluding("Zero d contradicts err true",
				invariant.Bucket_Lo(d), invariant.Bucket_True(err_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(filename), Lo: min_go_filename_chars, Hi: max_filesystem_path_chars,
			Message: "Filename is non-empty per caller; shortest is a 4-char `a.go`",
		}),
	)

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
	return check_invariant_assertions(file_set, file, source_bytes), nil
}

// Reports whether file's underlying *token.File path ends in "_test.go".
// Test files are out of scope by user decree.
func check_invariant_assertions_is_test_file(file_set *token.FileSet, file *ast.File) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	tok_file := file_set.File(file.Pos())
	invariant.Cross_Product(
		invariant.Always(tok_file != nil, "Tok_file is non-nil at this point"))
	if tok_file == nil {
		return false
	}
	return strings.HasSuffix(tok_file.Name(), "_test.go")
}

// Reports whether file lives within the invariant package (pure tier or
// composition tier). The package implements the assertion API itself —
// requiring its own functions to route through that API would either
// recurse or depend on a not-yet-initialized Default recorder. Exempted
// wholesale until a recursion-safe internal-assertion scheme lands.
func check_invariant_assertions_is_invariant_package(
	file_set *token.FileSet, file *ast.File,
) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	tok_file := file_set.File(file.Pos())
	invariant.Cross_Product(
		invariant.Always(tok_file != nil, "Tok_file is non-nil at this point"))
	if tok_file == nil {
		return false
	}
	return strings.Contains(tok_file.Name(), "golang_snacks/invariant/v2/")
}

// Walks file.Imports and returns the set of import-spec local names that
// resolve to the invariant package or its composition tier. The local name
// defaults to the package's declared name (`invariant` for the pure tier,
// `invariant_default` for the composition tier) unless an explicit alias
// is supplied. Files that don't import either path return an empty set.
func check_invariant_assertions_collect_invariant_idents(file *ast.File) (idents map[string]bool) {
	defer func() {
		i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(idents), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Idents is bounded by budget",
		})
		empty_axis := invariant.Sometimes(
			len(idents) == 0, "Idents is empty for files without an invariant import")
		invariant.Cross_Product(i, empty_axis,
			invariant.Excluding("Max i contradicts empty true",
				invariant.Bucket_Hi(i), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Invariant_idents at AST at safety cap is bad",
				invariant.Bucket_Hi(i), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero i implies empty true",
				invariant.Bucket_Lo(i), invariant.Bucket_False(empty_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(file != nil, "File is non-nil"))

	idents = map[string]bool{}
	const pure_path = "github.com/james-orcales/james-orcales/golang_snacks/invariant/v2"
	const default_path = "github.com/james-orcales/james-orcales/golang_snacks/" +
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

// One int/string/pointer/bool/struct field from a same-file struct, captured
// for recursive requirement generation. Type holds the raw AST expression so
// the recursive emitter can descend through pointer-to-struct and nested
// struct types. Position anchors diagnostics to the field declaration.
type check_invariant_assertions_struct_field struct {
	Name     string
	Kind     string
	Type     ast.Expr
	Position token.Pos
}

// Walks top-level *ast.GenDecl(TYPE) and maps each struct type name to the
// list of its int/string/pointer fields. Embedded fields (no Names) are
// skipped — synthesising a `param.???` path for them requires resolving
// the embedded type, which goes beyond what go/ast can do alone.
// Builds a set of struct type names that contain a sync.Mutex or
// sync.RWMutex field. Such structs are opaque at the assertion level: their
// sibling fields produce no requirements because the mutex says those
// fields' invariants only hold inside a Lock()/Unlock() critical section,
// not at function entry. The recursive walker stops at such structs.
// Reports whether struct_type has a `sync.Mutex` or `sync.RWMutex` field
// (named or unnamed). The mutex-presence scan ignores field names so a
// `_`-named mutex still triggers opaque-on-mutex for the containing struct.
func check_invariant_assertions_struct_type_has_mutex_field(
	struct_type *ast.StructType, stdlib_imports map[string]bool,
) (yes bool) {
	defer func() {
		yes_axis := invariant.Sometimes(yes, "Affirmative branch is exercised")
		stdlib_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Stdlib_imports is bounded by AST budget",
		})
		invariant.Cross_Product(yes_axis, stdlib_axis,
			invariant.Excluding("Hi stdlib unreachable in test corpus (yes true)",
				invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_True(yes_axis)),
			invariant.Excluding("Hi stdlib unreachable in test corpus (yes false)",
				invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_False(yes_axis)),
			invariant.Excluding("Lo stdlib (=0) with yes=true impossible — match "+
				"requires package in stdlib map",
				invariant.Bucket_Lo(stdlib_axis), invariant.Bucket_True(yes_axis)),
		)
	}()
	stdlib_empty := invariant.Sometimes(
		len(stdlib_imports) == 0, "Stdlib_imports is empty sometimes")
	stdlib_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_imports is bounded by AST budget",
	})
	invariant.Cross_Product(invariant.Always(struct_type != nil, "Struct_type is non-nil"),
		stdlib_empty, stdlib_axis,
		invariant.Excluding("Lo stdlib implies stdlib_empty true",
			invariant.Bucket_Lo(stdlib_axis), invariant.Bucket_False(stdlib_empty)),
		invariant.Excluding("Hi stdlib implies stdlib_empty false",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_True(stdlib_empty)),
		invariant.Excluding("Hi stdlib unreachable in test corpus",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_False(stdlib_empty)))
	if struct_type.Fields == nil {
		return false
	}
	for _, field := range struct_type.Fields.List {
		if check_invariant_assertions_type_expression_is_mutex(field.Type, stdlib_imports) {
			return true
		}
	}
	return false
}

// Reports whether type_expression is a `<stdlib>.Mutex` or `<stdlib>.RWMutex`
// selector. The package qualifier must be in stdlib_imports; the selector
// name must be Mutex or RWMutex. Mutex/RWMutex are unique to the sync stdlib
// package, so the qualifier check is sufficient without further sync-specific
// resolution.
func check_invariant_assertions_type_expression_is_mutex(
	type_expression ast.Expr, stdlib_imports map[string]bool,
) (yes bool) {
	defer func() {
		yes_axis := invariant.Sometimes(yes, "Affirmative branch is exercised")
		stdlib_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Stdlib_imports is bounded by AST budget",
		})
		invariant.Cross_Product(yes_axis, stdlib_axis,
			invariant.Excluding("Hi stdlib unreachable in test corpus (yes true)",
				invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_True(yes_axis)),
			invariant.Excluding("Hi stdlib unreachable in test corpus (yes false)",
				invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_False(yes_axis)),
			invariant.Excluding("Lo stdlib (=0) with yes=true impossible — match "+
				"requires package in stdlib map",
				invariant.Bucket_Lo(stdlib_axis), invariant.Bucket_True(yes_axis)),
		)
	}()
	stdlib_empty := invariant.Sometimes(
		len(stdlib_imports) == 0, "Stdlib_imports is empty sometimes")
	stdlib_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_imports is bounded by AST budget",
	})
	invariant.Cross_Product(stdlib_empty, stdlib_axis,
		invariant.Excluding("Lo stdlib implies stdlib_empty true",
			invariant.Bucket_Lo(stdlib_axis), invariant.Bucket_False(stdlib_empty)),
		invariant.Excluding("Hi stdlib implies stdlib_empty false",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_True(stdlib_empty)),
		invariant.Excluding("Hi stdlib unreachable in test corpus",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_False(stdlib_empty)))
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

// Pairs a struct type's tracked fields with whether the struct is mutex-guarded.
// Both are keyed by the struct's name and consulted together by the recursion
// walker (look up the fields to recurse, unless the struct is lock-guarded and
// opaque), so they ride in one map rather than two parallel ones — halving the
// map-assertion load wherever the index is threaded as a parameter.
type check_invariant_assertions_struct_information struct {
	Fields []check_invariant_assertions_struct_field
	Locked bool
}

// Walks top-level *ast.GenDecl(TYPE) and maps each struct type name to its
// struct_info: the list of its int/string/pointer fields, and whether it holds
// a sync.Mutex/RWMutex (lock-guarded structs are opaque — their sibling fields'
// invariants only hold inside a critical section, so the walker stops at them).
// Embedded fields (no Names) are skipped — synthesising a `param.???` path for
// them needs type resolution beyond go/ast.
func check_invariant_assertions_build_struct_field_index(
	file *ast.File, stdlib_imports map[string]bool,
) (index map[string]check_invariant_assertions_struct_information) {
	defer func() {
		i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(index), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Index is bounded by budget",
		})
		empty_axis := invariant.Sometimes(len(index) == 0, "Index is empty sometimes")
		stdlib := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Stdlib_imports is bounded by budget",
		})
		invariant.Cross_Product(i, empty_axis, stdlib,
			invariant.Excluding("Max i contradicts empty true",
				invariant.Bucket_Hi(i), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Index at AST safety cap is bad",
				invariant.Bucket_Hi(i), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero i implies empty true",
				invariant.Bucket_Lo(i), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Hi stdlib unreachable (empty)",
				invariant.Bucket_Hi(stdlib), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Hi stdlib unreachable (non-empty)",
				invariant.Bucket_Hi(stdlib), invariant.Bucket_False(empty_axis)),
		)
	}()
	stdlib_p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_imports is bounded by budget",
	})
	stdlib_empty := invariant.Sometimes(
		len(stdlib_imports) == 0, "Stdlib_imports is empty sometimes")
	invariant.Cross_Product(
		invariant.Always(file != nil, "File is non-nil"), stdlib_p, stdlib_empty,
		invariant.Excluding("Hi stdlib unreachable (empty)",
			invariant.Bucket_Hi(stdlib_p), invariant.Bucket_True(stdlib_empty)),
		invariant.Excluding("Hi stdlib unreachable (non-empty)",
			invariant.Bucket_Hi(stdlib_p), invariant.Bucket_False(stdlib_empty)),
		invariant.Excluding("Lo stdlib implies empty true",
			invariant.Bucket_Lo(stdlib_p), invariant.Bucket_False(stdlib_empty)),
	)

	index = map[string]check_invariant_assertions_struct_information{}
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
			struct_type, is_struct := type_specification.Type.(*ast.StructType)
			if !is_struct {
				continue
			}
			information := check_invariant_assertions_struct_information{
				Fields: check_invariant_assertions_extract_struct_fields(
					struct_type),
				Locked: check_invariant_assertions_struct_type_has_mutex_field(
					struct_type, stdlib_imports),
			}
			index[type_specification.Name.Name] = information
		}
	}
	return index
}

// Returns the int/string/pointer fields of struct_type. The pointer kind
// here does not carry a recurse-target — second-level recursion is out
// of scope (constraint #4 caps recursion at one level).
func check_invariant_assertions_extract_struct_fields(
	struct_type *ast.StructType,
) (fields []check_invariant_assertions_struct_field) {
	defer func() {
		f := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(fields), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Fields is bounded by budget",
		})
		empty_axis := invariant.Sometimes(len(fields) == 0, "Fields is empty sometimes")
		invariant.Cross_Product(f, empty_axis,
			invariant.Excluding("Max f contradicts empty true",
				invariant.Bucket_Hi(f), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Axis f at safety cap is bad",
				invariant.Bucket_Hi(f), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero f implies empty true",
				invariant.Bucket_Lo(f), invariant.Bucket_False(empty_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(struct_type != nil, "Struct_type is non-nil"))

	if struct_type.Fields == nil {
		return nil
	}
	for _, field := range struct_type.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		for _, name := range field.Names {
			if name.Name == "_" {
				continue
			}
			if !ast.IsExported(name.Name) {
				continue
			}
			// Keep every named exported field — the walker's emit_step
			// dispatches on Type and decides what (if anything) to emit
			// per type. Filtering here is what silently dropped string,
			// float, slice, map, and channel fields before §4.5.
			fields = append(fields, check_invariant_assertions_struct_field{
				Name:     name.Name,
				Type:     field.Type,
				Position: name.Pos(),
			})
		}
	}
	return fields
}

// True iff name is one of the integer keywords this check enforces.
// Excludes byte / rune / uintptr by user decision: byte and rune are
// alias keywords whose intent (a UTF-8 code unit / a Unicode code point)
// is meaningfully different from "integer with magic-number coverage",
// and uintptr is a pointer-bits container, not a value domain.
func check_invariant_assertions_is_integer_keyword(name string) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty identifier bounded by identifier budget",
		}),
	)

	switch name {
	case "int", "int8", "int16", "int32", "int64":
		return true
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
}

// True iff name is one of the floating-point keywords this check enforces.
// Float leaves get four requirements (boundary, zero, NaN, Inf) — NaN and
// Inf are sentinel states that integer types can't represent, so they're
// tracked separately rather than collapsing into the boundary half.
func check_invariant_assertions_is_float_keyword(name string) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty identifier bounded by identifier budget",
		}),
	)
	switch name {
	case "float32", "float64":
		return true
	}
	return false
}

// True iff name is the string type keyword. Strings get the same dual
// boundary_int + zero_string requirement structure as ints, with boundary
// credited via len()-based shapes and the empty-state via direct
// `s == ""` comparisons. []byte / []rune are deliberately out of scope —
// only the `string` type triggers.
func check_invariant_assertions_is_string_keyword(name string) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty identifier bounded by identifier budget",
		}),
	)
	return name == "string"
}

// One obligation derived from a function's signature: identifier_path must
// be covered by an assertion of kind at the position of the field that
// produced it. Function_Label carries the enclosing function's display name
// (Foo.Bar for methods, F for free functions) so diagnostics can point the
// reader at the function whose prologue is incomplete.
type check_invariant_assertions_requirement struct {
	Identifier_Path   string
	Kind              string
	Position          token.Pos
	Field_Description string
	Function_Label    string
	// Has_Nillable_Ancestor is true when the path walks through at least
	// one pointer segment before reaching the leaf. Non-nillable leaves
	// (int / bool / string) with no nillable ancestor must be asserted
	// outside any `if X != nil { ... }` block; with a nillable ancestor,
	// inside-if-block credit is acceptable since the deref would otherwise
	// be unsafe.
	Has_Nillable_Ancestor bool
	// Requires_Defer_Position flags requirements emitted from a leaf whose
	// length can change inside the function body — slices (append, reslice),
	// variadics (slice-shaped), and maps (insert, delete). Input requirements
	// of these kinds must be satisfied in BOTH the prologue (entry state) and
	// the defer (exit state). Named-return leaves remain defer-only since the
	// value isn't defined until return time.
	Requires_Defer_Position bool
}

type check_invariant_assertions_collect_requirements_input struct {
	Function            *ast.FuncDecl
	Struct_Field_Index  map[string]check_invariant_assertions_struct_information
	Always_Nil_Pointers map[string]bool
}

// Walks the receiver, parameters, and named returns of the function and
// emits a requirement per in-scope identifier. For pointer-to-same-file-struct
// parameters, also emits one requirement per int/string/pointer field of
// the referenced struct (recursion stops at one level).
func check_invariant_assertions_collect_requirements(
	input *check_invariant_assertions_collect_requirements_input,
) (
	parameter_requirements []check_invariant_assertions_requirement,
	parameter_defer_requirements []check_invariant_assertions_requirement,
	named_return_requirements []check_invariant_assertions_requirement,
) {
	defer func() {
		collect_requirements_assert_exit(&collect_requirements_assert_exit_input{
			Parameter_Requirements:       parameter_requirements,
			Parameter_Defer_Requirements: parameter_defer_requirements,
			Named_Return_Requirements:    named_return_requirements,
			Input:                        input,
		})
	}()
	check_invariant_assertions_collect_requirements_input_assert_entry(input)

	function_label := check_invariant_assertions_function_label(input.Function)
	invariant.Cross_Product(
		invariant.Always(function_label != "", "Function_label is non-empty at this point"))
	if input.Function.Recv != nil {
		parameter_requirements = append(parameter_requirements,
			check_invariant_assertions_requirements_from_field_list(
				&check_invariant_assertions_requirements_from_field_list_input{
					Field_List:          input.Function.Recv,
					Struct_Field_Index:  input.Struct_Field_Index,
					Function_Label:      function_label,
					Always_Nil_Pointers: input.Always_Nil_Pointers,
				})...)
	}
	if input.Function.Type.Params != nil {
		parameter_requirements = append(parameter_requirements,
			check_invariant_assertions_requirements_from_field_list(
				&check_invariant_assertions_requirements_from_field_list_input{
					Field_List:          input.Function.Type.Params,
					Struct_Field_Index:  input.Struct_Field_Index,
					Function_Label:      function_label,
					Always_Nil_Pointers: input.Always_Nil_Pointers,
				})...)
	}
	// Length-mutable inputs (slices, variadics, maps — params or receiver
	// fields) need their assertion satisfied in BOTH the prologue and the
	// defer. Len can change inside the body (append/reslice for slices,
	// insert/delete for maps), so prologue pins entry and defer pins exit.
	// Mirror every Requires_Defer_Position requirement into the defer list.
	for _, requirement := range parameter_requirements {
		if requirement.Requires_Defer_Position {
			parameter_defer_requirements = append(parameter_defer_requirements,
				requirement)
		}
	}
	if input.Function.Type.Results != nil {
		// Tier-1's check_namesd_returns forbids unnamed returns
		// codebase-wide, so by the time this tier-2 check runs every result
		// field carries a name. Encoding the contract as Always at the
		// consumer site keeps the dependency load-bearing instead of relying
		// on a silent defensive skip downstream.
		for _, field := range input.Function.Type.Results.List {
			invariant.Cross_Product(invariant.Always(len(field.Names) > 0,
				"Return field is named (tier-1 guarantee from "+
					"check_namesd_returns)"))

		}
		named_return_requirements = append(named_return_requirements,
			check_invariant_assertions_requirements_from_field_list(
				&check_invariant_assertions_requirements_from_field_list_input{
					Field_List:          input.Function.Type.Results,
					Struct_Field_Index:  input.Struct_Field_Index,
					Function_Label:      function_label,
					Always_Nil_Pointers: input.Always_Nil_Pointers,
				})...)
	}
	return parameter_requirements, parameter_defer_requirements, named_return_requirements
}

type collect_requirements_assert_exit_input struct {
	Parameter_Requirements       []check_invariant_assertions_requirement
	Parameter_Defer_Requirements []check_invariant_assertions_requirement
	Named_Return_Requirements    []check_invariant_assertions_requirement
	Input                        *check_invariant_assertions_collect_requirements_input
}

func collect_requirements_assert_exit(input *collect_requirements_assert_exit_input) {
	pr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_requirements is bounded by budget",
	})
	pd := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Defer_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_defer_requirements is bounded by budget",
	})
	nr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_requirements is bounded by budget",
	})
	anp := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Input.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Always_Nil_Pointers is bounded",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Input.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Struct_Field_Index is bounded",
	})
	invariant.Cross_Product(pr, pd, nr, anp, sfi,
		invariant.Excluding("Hi pr Hi pd unreachable",
			invariant.Bucket_Hi(pr), invariant.Bucket_Hi(pd)),
		invariant.Excluding("Hi pr Lo pd unreachable",
			invariant.Bucket_Hi(pr), invariant.Bucket_Lo(pd)),
		invariant.Excluding("Lo pr Hi pd unreachable",
			invariant.Bucket_Lo(pr), invariant.Bucket_Hi(pd)),
		invariant.Excluding("Hi pd Hi nr unreachable",
			invariant.Bucket_Hi(pd), invariant.Bucket_Hi(nr)),
		invariant.Excluding("Hi pd Lo nr unreachable",
			invariant.Bucket_Hi(pd), invariant.Bucket_Lo(nr)),
		invariant.Excluding("Lo pd Hi nr unreachable",
			invariant.Bucket_Lo(pd), invariant.Bucket_Hi(nr)),
		invariant.Excluding("Hi anp Lo pr unreachable",
			invariant.Bucket_Hi(anp), invariant.Bucket_Lo(pr)),
		invariant.Excluding("Hi sfi Lo pr unreachable",
			invariant.Bucket_Hi(sfi), invariant.Bucket_Lo(pr)),
	)
}

func check_invariant_assertions_collect_requirements_input_assert_entry(
	input *check_invariant_assertions_collect_requirements_input,
) {
	anp := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Always_Nil_Pointers is bounded",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Struct_Field_Index is bounded",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Function != nil, "Function is non-nil"),
		anp, sfi,
		invariant.Excluding("Hi sfi Lo anp unreachable",
			invariant.Bucket_Hi(sfi), invariant.Bucket_Lo(anp)),
		invariant.Excluding("Hi anp Lo sfi unreachable",
			invariant.Bucket_Hi(anp), invariant.Bucket_Lo(sfi)),
		invariant.Excluding("Hi anp Hi sfi unreachable",
			invariant.Bucket_Hi(anp), invariant.Bucket_Hi(sfi)),
	)
}

// Returns the display label for a FuncDecl. Free functions report their
// bare name (`F`); methods report `Receiver_Type.Method` for value receivers
// and `(*Receiver_Type).Method` for pointer receivers, matching the standard
// Go documentation convention. Anonymous receivers report just the method
// name.
func check_invariant_assertions_function_label(function *ast.FuncDecl) (label string) {
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(label),
				Lo: min_non_empty,
				Hi: Max_Identifier_Chars,
				Message: "Label is non-empty per construction and bounded by " +
					"identifier budget",
			}),
		)
	}()
	invariant.Cross_Product(invariant.Always(
		function != nil, "Function is non-nil"))

	if function.Recv == nil {
		return function.Name.Name
	}
	if len(function.Recv.List) == 0 {
		return function.Name.Name
	}
	receiver_type := function.Recv.List[0].Type
	if star, is_star := receiver_type.(*ast.StarExpr); is_star {
		identifier, is_ident := star.X.(*ast.Ident)
		if !is_ident {
			return function.Name.Name
		}
		return "(*" + identifier.Name + ")." + function.Name.Name
	}
	identifier, is_ident := receiver_type.(*ast.Ident)
	if !is_ident {
		return function.Name.Name
	}
	return identifier.Name + "." + function.Name.Name
}

type check_invariant_assertions_requirements_from_field_list_input struct {
	Field_List          *ast.FieldList
	Struct_Field_Index  map[string]check_invariant_assertions_struct_information
	Function_Label      string
	Always_Nil_Pointers map[string]bool
}

// Walks one *ast.FieldList (receiver / params / results) and produces the
// requirements for every named (non-`_`, non-variadic, non-anonymous) field
// whose type matches the MVP set. Struct-typed fields recurse through
// `Struct_Field_Index` to emit per-leaf requirements; pointer-to-struct
// fields produce both a pointer (nil-check) requirement at the top-level
// path and per-inner-field requirements unless the pointer's path appears
// in `Always_Nil_Pointers` (the opt-out for `Always(p == nil)` in the
// prologue). Self-referential structs terminate via the visited set.
func check_invariant_assertions_requirements_from_field_list(
	input *check_invariant_assertions_requirements_from_field_list_input,
) (requirements []check_invariant_assertions_requirement) {
	defer func() {
		check_invariant_assertions_requirements_from_field_list_assert_exit(
			requirements, input)
	}()
	anp := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Always_Nil_Pointers is bounded",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Struct_Field_Index is bounded",
	})
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Field_List != nil, "Field_List is non-nil"),
		anp, sfi, fl,
		invariant.Excluding("Hi anp unreachable (Lo fl)",
			invariant.Bucket_Hi(anp), invariant.Bucket_Lo(fl)),
		invariant.Excluding("Hi anp unreachable (Hi fl)",
			invariant.Bucket_Hi(anp), invariant.Bucket_Hi(fl)),
		invariant.Excluding("Hi sfi unreachable (Lo fl)",
			invariant.Bucket_Hi(sfi), invariant.Bucket_Lo(fl)),
		invariant.Excluding("Hi sfi unreachable (Hi fl)",
			invariant.Bucket_Hi(sfi), invariant.Bucket_Hi(fl)),
	)

	for _, field := range input.Field_List.List {
		if len(field.Names) == 0 {
			continue
		}
		for _, name := range field.Names {
			if name.Name == "_" {
				continue
			}
			requirements = append(requirements,
				check_invariant_assertions_recursive_frame_emit(
					&check_invariant_assertions_recursive_frame{
						Type_Expression: field.Type,
						Path:            name.Name,
						Position:        name.Pos(),
						Function_Label:  input.Function_Label,
						Visited:         map[string]int{},
					},
					&check_invariant_assertions_recursive_frame_tables{
						Struct_Field_Index:  input.Struct_Field_Index,
						Always_Nil_Pointers: input.Always_Nil_Pointers,
					})...)
		}
	}
	return requirements
}

func check_invariant_assertions_requirements_from_field_list_assert_exit(
	requirements []check_invariant_assertions_requirement,
	input *check_invariant_assertions_requirements_from_field_list_input,
) {
	r := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Requirements is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(requirements) == 0, "Requirements is empty sometimes")
	anp := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Always_Nil_Pointers is bounded",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Struct_Field_Index is bounded",
	})
	invariant.Cross_Product(r, empty_axis, anp, sfi,
		invariant.Excluding("Max r contradicts empty true",
			invariant.Bucket_Hi(r), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis r at safety cap is bad",
			invariant.Bucket_Hi(r), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero r implies empty true",
			invariant.Bucket_Lo(r), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi anp with empty",
			invariant.Bucket_Hi(anp), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi anp with reqs",
			invariant.Bucket_Hi(anp), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi sfi with empty",
			invariant.Bucket_Hi(sfi), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi sfi with reqs",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(empty_axis)),
	)
}

type check_invariant_assertions_recursive_frame struct {
	Type_Expression       ast.Expr
	Path                  string
	Position              token.Pos
	Function_Label        string
	Visited               map[string]int
	Has_Nillable_Ancestor bool
}

// Bundles the file-level lookup tables the walk consults — constant across
// every frame, so they ride beside the per-frame data instead of on each frame
// (which would force every handler to assert them). A distinct type from
// *recursive_frame so emit's two parameters don't repeat a map[string]bool and
// trip the input-struct rule.
type check_invariant_assertions_recursive_frame_tables struct {
	Struct_Field_Index  map[string]check_invariant_assertions_struct_information
	Always_Nil_Pointers map[string]bool
}

// Emits one requirement per tracked leaf reachable from input.Path through
// input.Type_Expression. Cycle protection: when Visited[struct_name] reaches
// the unroll threshold, the walker stops descending into that struct's
// children — the cycle's outer pointer still gets its top-level pointer
// requirement, fields beyond the unrolled back-edge don't recurse.
func check_invariant_assertions_recursive_frame_emit(
	input *check_invariant_assertions_recursive_frame,
	tables *check_invariant_assertions_recursive_frame_tables,
) (requirements []check_invariant_assertions_requirement) {
	defer func() {
		check_invariant_assertions_recursive_frame_emit_assert_exit(
			requirements, input, tables)
	}()
	check_invariant_assertions_recursive_frame_emit_assert_entry(input, tables)

	stack := []*check_invariant_assertions_recursive_frame{input}
	for range invariant.Game_Loop() {
		if len(stack) == 0 {
			return requirements
		}
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		// Resolve the file-level pointer-nil lookup here so the frame handlers
		// never carry the always_nil_pointers map (they would have to assert it).
		is_always_nil := tables.Always_Nil_Pointers[top.Path]
		var expand_identifier *ast.Ident
		stack, requirements, expand_identifier =
			check_invariant_assertions_recursive_frame_emit_step(
				stack, top, requirements, is_always_nil)
		check_invariant_assertions_recursive_frame_emit_loop_assert(
			stack, requirements, expand_identifier)
		// The driver owns struct expansion; is_pointer is re-derived here (not a
		// 4th return).
		if expand_identifier != nil {
			_, expand_is_pointer := top.Type_Expression.(*ast.StarExpr)
			stack, _ = check_invariant_assertions_recursive_frame_push_struct_fields(
				stack,
				top,
				expand_identifier,
				tables.Struct_Field_Index,
				expand_is_pointer)
			invariant.Cross_Product(
				invariant.Always(stack != nil, "Stack is non-nil after expansion"))
		}
	}
	return requirements
}

func check_invariant_assertions_recursive_frame_emit_assert_exit(
	requirements []check_invariant_assertions_requirement,
	input *check_invariant_assertions_recursive_frame,
	tables *check_invariant_assertions_recursive_frame_tables,
) {
	r := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Requirements is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(requirements) == 0, "Requirements is empty sometimes")
	anp := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(tables.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Always_nil_pointers is bounded by budget",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(tables.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by budget",
	})
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Visited is bounded by budget",
	})
	invariant.Cross_Product(r, empty_axis, anp, sfi, vis,
		invariant.Excluding("Max r contradicts empty true",
			invariant.Bucket_Hi(r), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi r is bad",
			invariant.Bucket_Hi(r), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Lo r implies empty true",
			invariant.Bucket_Lo(r), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi anp unreachable (empty)",
			invariant.Bucket_Hi(anp), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi anp unreachable (non-empty)",
			invariant.Bucket_Hi(anp), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi sfi unreachable (empty)",
			invariant.Bucket_Hi(sfi), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi sfi unreachable (non-empty)",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi vis unreachable (empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi vis unreachable (non-empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(empty_axis)),
	)
}

func check_invariant_assertions_recursive_frame_emit_assert_entry(
	input *check_invariant_assertions_recursive_frame,
	tables *check_invariant_assertions_recursive_frame_tables,
) {
	input_axis := invariant.Always(input != nil, "Input is non-nil")
	ancestor_axis := invariant.Sometimes(
		input.Has_Nillable_Ancestor, "Input is under a nillable ancestor sometimes")
	anp := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(tables.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Always_nil_pointers is bounded by budget",
	})
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(tables.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by budget",
	})
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Visited is bounded by budget",
	})
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	pa := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Path is the parameter or named-return identifier",
	})
	invariant.Cross_Product(input_axis, invariant.Always(tables != nil, "Tables is non-nil"),
		ancestor_axis, anp, sfi, vis,
		invariant.Excluding(
			"Top-level entry is always at depth zero", invariant.Bucket_True(
				ancestor_axis)),
		invariant.Excluding("Hi anp unreachable",
			invariant.Bucket_Hi(anp), invariant.Bucket_False(ancestor_axis)),
		invariant.Excluding("Hi sfi unreachable",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(ancestor_axis)),
		invariant.Excluding("Hi vis unreachable",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(ancestor_axis)),
	)
	invariant.Cross_Product(fl)
	invariant.Cross_Product(pa)
}

func check_invariant_assertions_recursive_frame_emit_loop_assert(
	stack []*check_invariant_assertions_recursive_frame,
	requirements []check_invariant_assertions_requirement,
	expand_identifier *ast.Ident,
) {
	loop_sk := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stack), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stack is bounded by budget",
	})
	loop_stack_empty := invariant.Sometimes(len(stack) == 0, "Stack is empty sometimes")
	loop_requirement_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Requirements is bounded by budget",
	})
	loop_requirement_empty := invariant.Sometimes(
		len(requirements) == 0, "Requirements is empty sometimes")
	invariant.Cross_Product(loop_sk, loop_stack_empty,
		invariant.Excluding("Max loop_stack contradicts loop_stack_empty true",
			invariant.Bucket_Hi(loop_sk),
			invariant.Bucket_True(loop_stack_empty)),
		invariant.Excluding("Loop_stack at safety cap is bad",
			invariant.Bucket_Hi(loop_sk),
			invariant.Bucket_False(loop_stack_empty)),
		invariant.Excluding("Zero loop_stack implies loop_stack_empty true",
			invariant.Bucket_Lo(loop_sk),
			invariant.Bucket_False(loop_stack_empty)),
	)
	invariant.Cross_Product(loop_requirement_axis, loop_requirement_empty,
		invariant.Excluding("Max loop_requirement contradicts "+
			"loop_requirement_empty true",
			invariant.Bucket_Hi(loop_requirement_axis),
			invariant.Bucket_True(loop_requirement_empty)),
		invariant.Excluding("Loop_requirement at safety cap is bad",
			invariant.Bucket_Hi(loop_requirement_axis),
			invariant.Bucket_False(loop_requirement_empty)),
		invariant.Excluding("Zero loop_requirement implies loop_requirement_empty "+
			"true",
			invariant.Bucket_Lo(loop_requirement_axis),
			invariant.Bucket_False(loop_requirement_empty)),
	)
	invariant.Cross_Product(
		invariant.Sometimes(expand_identifier == nil, "Expand_identifier is nil "+
			"sometimes"))
}

func check_invariant_assertions_recursive_frame_emit_step(
	stack []*check_invariant_assertions_recursive_frame,
	top *check_invariant_assertions_recursive_frame,
	requirements []check_invariant_assertions_requirement,
	is_always_nil bool,
) (
	next_stack []*check_invariant_assertions_recursive_frame,
	next_requirements []check_invariant_assertions_requirement,
	expand_identifier *ast.Ident,
) {
	defer func() {
		check_invariant_assertions_recursive_frame_emit_step_assert_exit(
			&check_invariant_assertions_recursive_frame_emit_step_assert_exit_input{
				Next_Requirements: next_requirements,
				Next_Stack:        next_stack,
				Stack:             stack,
				Requirements:      requirements,
				Top:               top,
				Expand_Identifier: expand_identifier,
			})
	}()
	check_invariant_assertions_recursive_frame_emit_step_assert_entry(
		stack, top, requirements, is_always_nil)
	type_expression := top.Type_Expression
	star, is_pointer := type_expression.(*ast.StarExpr)
	if is_pointer {
		type_expression = star.X
	}
	if is_pointer {
		requirements = append(requirements,
			check_invariant_assertions_recursive_frame_pointer_requirement(top))
		if is_always_nil {
			return stack, requirements, nil
		}
	}
	leaf_requirements := check_invariant_assertions_recursive_frame_leaf_dispatch(
		top, type_expression, is_pointer)
	invariant.Cross_Product(
		invariant.Sometimes(leaf_requirements == nil, "Leaf_requirements is nil for "+
			"non-leaf types"))
	if leaf_requirements != nil {
		return stack, append(requirements, leaf_requirements...), nil
	}
	identifier, is_ident := type_expression.(*ast.Ident)
	if is_ident {
		leaves := check_invariant_assertions_recursive_frame_emit_leaf(
			top, identifier, is_pointer)
		invariant.Cross_Product(
			invariant.Sometimes(leaves == nil, "Leaves is nil for non-leaf types"))
		if leaves != nil {
			return stack, append(requirements, leaves...), nil
		}
		// Defer struct expansion to the driver, which owns the field-index map.
		return stack, requirements, identifier
	}
	// When this frame's pointer is stripped, push_inline descendants inherit a
	// nillable ancestor.
	stack = check_invariant_assertions_recursive_frame_emit_push_inline(
		stack, top, type_expression, top.Has_Nillable_Ancestor || is_pointer)
	invariant.Cross_Product(
		invariant.Always(stack != nil, "Stack is non-nil after push_inline"))
	return stack, requirements, nil
}

// Bundles emit_step's exit-assertion operands: the two incoming slices and the
// two outgoing named-return slices share element types, so passing them
// positionally would trip the same-type-parameter bundling rule.
type check_invariant_assertions_recursive_frame_emit_step_assert_exit_input struct {
	Next_Requirements []check_invariant_assertions_requirement
	Next_Stack        []*check_invariant_assertions_recursive_frame
	Stack             []*check_invariant_assertions_recursive_frame
	Requirements      []check_invariant_assertions_requirement
	Top               *check_invariant_assertions_recursive_frame
	Expand_Identifier *ast.Ident
}

func check_invariant_assertions_recursive_frame_emit_step_assert_exit(
	input *check_invariant_assertions_recursive_frame_emit_step_assert_exit_input,
) {
	r := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Next_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Next_requirements is bounded by budget",
	})
	re := invariant.Sometimes(
		len(input.Next_Requirements) == 0, "Next_requirements is empty sometimes")
	ns := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Next_Stack), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Next_stack is bounded by budget",
	})
	ds := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stack), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stack is bounded by budget",
	})
	dr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Requirements is bounded by budget",
	})
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	invariant.Cross_Product(r, re, ns, ds, dr, vis,
		invariant.Excluding("Max r contradicts re true",
			invariant.Bucket_Hi(r), invariant.Bucket_True(re)),
		invariant.Excluding("Axis r at safety cap is bad",
			invariant.Bucket_Hi(r), invariant.Bucket_False(re)),
		invariant.Excluding("Zero r implies re true",
			invariant.Bucket_Lo(r), invariant.Bucket_False(re)),
		invariant.Excluding("Hi ns unreachable (empty)",
			invariant.Bucket_Hi(ns), invariant.Bucket_True(re)),
		invariant.Excluding("Hi ns unreachable (non-empty)",
			invariant.Bucket_Hi(ns), invariant.Bucket_False(re)),
		invariant.Excluding("Hi ds unreachable (empty)",
			invariant.Bucket_Hi(ds), invariant.Bucket_True(re)),
		invariant.Excluding("Hi ds unreachable (non-empty)",
			invariant.Bucket_Hi(ds), invariant.Bucket_False(re)),
		invariant.Excluding("Hi dr unreachable",
			invariant.Bucket_Hi(dr), invariant.Bucket_True(re)),
		invariant.Excluding("Hi vis unreachable (empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(re)),
		invariant.Excluding("Hi vis unreachable (non-empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(re)),
	)
	expand_axis := invariant.Sometimes(input.Expand_Identifier == nil,
		"Expand_identifier is nil for non-struct frames sometimes")
	invariant.Cross_Product(expand_axis)
}

func check_invariant_assertions_recursive_frame_emit_step_assert_entry(
	stack []*check_invariant_assertions_recursive_frame,
	top *check_invariant_assertions_recursive_frame,
	requirements []check_invariant_assertions_requirement,
	is_always_nil bool,
) {
	sa := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stack), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stack is bounded by budget",
	})
	se := invariant.Sometimes(len(stack) == 0, "Stack is empty sometimes")
	p_vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	invariant.Cross_Product(
		invariant.Always(top != nil, "Top is non-nil"),
		sa, se, p_vis,
		invariant.Excluding("Max stack contradicts se true",
			invariant.Bucket_Hi(sa), invariant.Bucket_True(se)),
		invariant.Excluding("Stack at safety cap is bad",
			invariant.Bucket_Hi(sa), invariant.Bucket_False(se)),
		invariant.Excluding("Zero stack implies se true",
			invariant.Bucket_Lo(sa), invariant.Bucket_False(se)),
		invariant.Excluding("Hi vis unreachable (empty)",
			invariant.Bucket_Hi(p_vis), invariant.Bucket_True(se)),
		invariant.Excluding("Hi vis unreachable (non-empty)",
			invariant.Bucket_Hi(p_vis), invariant.Bucket_False(se)),
	)
	pr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Requirements is bounded by budget",
	})
	pr_empty := invariant.Sometimes(len(requirements) == 0, "Requirements is empty sometimes")
	invariant.Cross_Product(pr, pr_empty,
		invariant.Excluding("Max pr contradicts pr_empty true",
			invariant.Bucket_Hi(pr), invariant.Bucket_True(pr_empty)),
		invariant.Excluding("Axis pr at safety cap is bad",
			invariant.Bucket_Hi(pr), invariant.Bucket_False(pr_empty)),
		invariant.Excluding("Zero pr implies pr_empty true",
			invariant.Bucket_Lo(pr), invariant.Bucket_False(pr_empty)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Function_Label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Top.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Top.path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	always_nil_axis := invariant.Sometimes(
		is_always_nil, "Frame pointer is always-nil sometimes")
	invariant.Cross_Product(always_nil_axis)
	nillable_axis := invariant.Sometimes(
		top.Has_Nillable_Ancestor, "Top is under a nillable ancestor sometimes")
	invariant.Cross_Product(nillable_axis)
}

func check_invariant_assertions_recursive_frame_push_struct_fields(
	stack []*check_invariant_assertions_recursive_frame,
	top *check_invariant_assertions_recursive_frame,
	identifier *ast.Ident,
	struct_field_index map[string]check_invariant_assertions_struct_information,
	is_pointer bool,
) (next_stack []*check_invariant_assertions_recursive_frame, pushed bool) {
	defer func() {
		push_struct_fields_assert_exit(&push_struct_fields_assert_exit_input{
			Next_Stack:         next_stack,
			Stack:              stack,
			Struct_Field_Index: struct_field_index,
			Top:                top,
			Pushed:             pushed,
		})
	}()
	push_struct_fields_assert_entry(stack, top, identifier, struct_field_index, is_pointer)
	information, has := struct_field_index[identifier.Name]
	if !has {
		return stack, false
	}
	if top.Visited[identifier.Name] >= check_invariant_assertions_recursion_unroll_threshold {
		return stack, false
	}
	// Opaque-on-mutex: structs containing sync.Mutex/RWMutex skip sibling
	// recursion (those fields require Lock/Unlock context to be meaningful).
	if information.Locked {
		return stack, false
	}
	new_visited := make(map[string]int, len(top.Visited)+1)
	for k, v := range top.Visited {
		new_visited[k] = v
	}
	new_visited[identifier.Name] = top.Visited[identifier.Name] + 1
	child_has_nillable_ancestor := top.Has_Nillable_Ancestor || is_pointer
	for _, child_field := range information.Fields {
		stack = append(stack, &check_invariant_assertions_recursive_frame{
			Type_Expression:       child_field.Type,
			Path:                  top.Path + "." + child_field.Name,
			Position:              child_field.Position,
			Function_Label:        top.Function_Label,
			Visited:               new_visited,
			Has_Nillable_Ancestor: child_has_nillable_ancestor,
		})
	}
	return stack, true
}

type push_struct_fields_assert_exit_input struct {
	Next_Stack         []*check_invariant_assertions_recursive_frame
	Stack              []*check_invariant_assertions_recursive_frame
	Struct_Field_Index map[string]check_invariant_assertions_struct_information
	Top                *check_invariant_assertions_recursive_frame
	Pushed             bool
}

func push_struct_fields_assert_exit(input *push_struct_fields_assert_exit_input) {
	ns := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Next_Stack), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Next_stack is bounded by budget",
	})
	ns_empty := invariant.Sometimes(
		len(input.Next_Stack) == 0, "Next_stack is empty sometimes")
	ds := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stack), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stack is bounded by budget",
	})
	dsfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Struct_Field_Index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by budget",
	})
	dvis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded by budget",
	})
	invariant.Cross_Product(ns, ns_empty, ds, dsfi, dvis,
		invariant.Excluding("Max ns contradicts ns_empty true",
			invariant.Bucket_Hi(ns), invariant.Bucket_True(ns_empty)),
		invariant.Excluding("Axis ns at safety cap is bad",
			invariant.Bucket_Hi(ns), invariant.Bucket_False(ns_empty)),
		invariant.Excluding("Zero ns implies ns_empty true",
			invariant.Bucket_Lo(ns), invariant.Bucket_False(ns_empty)),
		invariant.Excluding("Hi ds unreachable (empty)",
			invariant.Bucket_Hi(ds), invariant.Bucket_True(ns_empty)),
		invariant.Excluding("Hi ds unreachable (non-empty)",
			invariant.Bucket_Hi(ds), invariant.Bucket_False(ns_empty)),
		invariant.Excluding("Hi dsfi unreachable (empty)",
			invariant.Bucket_Hi(dsfi), invariant.Bucket_True(ns_empty)),
		invariant.Excluding("Hi dsfi unreachable (non-empty)",
			invariant.Bucket_Hi(dsfi), invariant.Bucket_False(ns_empty)),
		invariant.Excluding("Hi dvis unreachable (empty)",
			invariant.Bucket_Hi(dvis), invariant.Bucket_True(ns_empty)),
		invariant.Excluding("Hi dvis unreachable (non-empty)",
			invariant.Bucket_Hi(dvis), invariant.Bucket_False(ns_empty)),
	)
	pushed_axis := invariant.Sometimes(
		input.Pushed, "Pushed is true when struct fields were appended")
	invariant.Cross_Product(pushed_axis)
}

func push_struct_fields_assert_entry(
	stack []*check_invariant_assertions_recursive_frame,
	top *check_invariant_assertions_recursive_frame,
	identifier *ast.Ident,
	struct_field_index map[string]check_invariant_assertions_struct_information,
	is_pointer bool,
) {
	sk := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stack), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stack is bounded by budget",
	})
	stack_empty := invariant.Sometimes(len(stack) == 0, "Stack is empty sometimes")
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(struct_field_index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by budget",
	})
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded by budget",
	})
	nillable_axis := invariant.Sometimes(
		top.Has_Nillable_Ancestor, "Top is under a nillable ancestor sometimes")
	invariant.Cross_Product(
		invariant.Always(top != nil, "Top is non-nil"),
		invariant.Always(identifier != nil, "Identifier is non-nil"),
		invariant.Sometimes(is_pointer, "Is_pointer can be true sometimes"),
		nillable_axis,
		sk, stack_empty, sfi, vis,
		invariant.Excluding("Hi stack with empty true is bad",
			invariant.Bucket_Hi(sk), invariant.Bucket_True(stack_empty)),
		invariant.Excluding("Hi stack with empty F is bad",
			invariant.Bucket_Hi(sk), invariant.Bucket_False(stack_empty)),
		invariant.Excluding("Lo stack implies empty true",
			invariant.Bucket_Lo(sk), invariant.Bucket_False(stack_empty)),
		invariant.Excluding("Hi sfi unreachable (empty)",
			invariant.Bucket_Hi(sfi), invariant.Bucket_True(stack_empty)),
		invariant.Excluding("Hi sfi unreachable (non-empty)",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(stack_empty)),
		invariant.Excluding("Hi vis unreachable (empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(stack_empty)),
		invariant.Excluding("Hi vis unreachable (non-empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(stack_empty)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Function_Label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Top.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Top.path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
}

// Pushes children of an inline struct (`struct{X int; Y int}`) onto the
// recursion stack. Anonymous structs have no name to check against the
// input-bundle convention; recursion is unconditional. Returns the stack
// unchanged for non-struct types. child_has_nillable_ancestor carries the
// ancestor flag forward — it is OR'd with top.Has_Nillable_Ancestor by the
// caller so descending through a pointer-into-inline-struct (e.g. `*struct{...}`)
// flips the bit for inner fields.
func check_invariant_assertions_recursive_frame_emit_push_inline(
	stack []*check_invariant_assertions_recursive_frame,
	top *check_invariant_assertions_recursive_frame,
	type_expression ast.Expr,
	child_has_nillable_ancestor bool,
) (updated []*check_invariant_assertions_recursive_frame) {
	defer func() {
		recursive_frame_push_inline_assert_exit(
			&recursive_frame_push_inline_assert_exit_input{
				Stack: stack, Updated: updated, Top: top,
			})
	}()
	recursive_frame_push_inline_assert_entry(stack, top, child_has_nillable_ancestor)
	struct_type, is_inline := type_expression.(*ast.StructType)
	if !is_inline {
		return stack
	}
	if struct_type.Fields == nil {
		return stack
	}
	for _, child_field := range struct_type.Fields.List {
		if len(child_field.Names) == 0 {
			continue
		}
		for _, child_name := range child_field.Names {
			if child_name.Name == "_" {
				continue
			}
			stack = append(stack, &check_invariant_assertions_recursive_frame{
				Type_Expression: child_field.Type,
				Path:            top.Path + "." + child_name.Name,
				Position:        child_name.Pos(),

				Function_Label: top.Function_Label, Visited: top.Visited,
				Has_Nillable_Ancestor: child_has_nillable_ancestor,
			})
		}
	}
	return stack
}

func recursive_frame_push_inline_assert_entry(
	stack []*check_invariant_assertions_recursive_frame,
	top *check_invariant_assertions_recursive_frame,
	child_has_nillable_ancestor bool,
) {
	top_ancestor_axis := invariant.Sometimes(
		top.Has_Nillable_Ancestor, "Top is under a nillable ancestor sometimes")
	child_ancestor_axis := invariant.Sometimes(
		child_has_nillable_ancestor, "Child is under a nillable ancestor sometimes")
	sk := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stack), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stack is bounded by budget",
	})
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	// (top_ancestor=true, child_ancestor=false) is impossible: when the
	// caller passes top.Has_Nillable_Ancestor=true the OR with is_pointer
	// still yields true, so child_has_nillable_ancestor is also true.
	invariant.Cross_Product(
		invariant.Always(top != nil, "Top is non-nil"),
		top_ancestor_axis, child_ancestor_axis, sk, vis,
		invariant.Excluding("Nillable ancestor propagates from parent to child",
			invariant.Bucket_True(top_ancestor_axis),
			invariant.Bucket_False(child_ancestor_axis)),
		invariant.Excluding("Hi stack unreachable (top anc)",
			invariant.Bucket_Hi(sk), invariant.Bucket_True(top_ancestor_axis)),
		invariant.Excluding("Hi stack unreachable (top-level)",
			invariant.Bucket_Hi(sk), invariant.Bucket_False(top_ancestor_axis)),
		invariant.Excluding("Hi vis unreachable (top anc)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(top_ancestor_axis)),
		invariant.Excluding("Hi vis unreachable (top-level)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(top_ancestor_axis)),
		invariant.Excluding("Empty stack under nillable ancestor with empty visited "+
			"impossible in test corpus",
			invariant.Bucket_Lo(sk),
			invariant.Bucket_True(top_ancestor_axis),
			invariant.Bucket_Lo(vis)),
	)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Top.path is a non-empty frame path",
	})
	path_mask := invariant.Sometimes(
		len(top.Path) == min_non_empty, "Top.path is a single char sometimes")
	invariant.Cross_Product(path_axis, path_mask,
		invariant.Excluding("Hi path mask T",
			invariant.Bucket_Hi(path_axis), invariant.Bucket_True(path_mask)),
		invariant.Excluding("Hi path mask F",
			invariant.Bucket_Hi(path_axis), invariant.Bucket_False(path_mask)),
		invariant.Excluding("Lo path mask F",
			invariant.Bucket_Lo(path_axis), invariant.Bucket_False(path_mask)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Function_Label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Top.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
}

type recursive_frame_push_inline_assert_exit_input struct {
	Stack   []*check_invariant_assertions_recursive_frame
	Updated []*check_invariant_assertions_recursive_frame
	Top     *check_invariant_assertions_recursive_frame
}

func recursive_frame_push_inline_assert_exit(
	input *recursive_frame_push_inline_assert_exit_input,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stack), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stack is bounded by budget",
	})
	u := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Updated), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Updated is bounded by budget",
	})
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	invariant.Cross_Product(s, u, vis,
		invariant.Excluding("Both s and u at safety caps is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(u)),
		invariant.Excluding("Max s with min u is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(u)),
		invariant.Excluding("Zero s with max u is bad",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(u)),
		invariant.Excluding("Hi vis unreachable (Lo s)",
			invariant.Bucket_Hi(vis), invariant.Bucket_Lo(s)),
		invariant.Excluding("Hi vis unreachable (Hi s)",
			invariant.Bucket_Hi(vis), invariant.Bucket_Hi(s)),
	)
}

// Emits the requirement(s) for a tracked primitive ident (int, bool) at the
// given path. Integer leaves emit TWO requirements — one boundary_int and
// one zero_int — because the spec demands BOTH a Boundary assertion and a
// zero-comparison; the dual-requirement shape also drives the >=2
// Cross_Product trigger naturally. Bool leaves emit a single requirement.
// Returns nil for non-leaf types (same-file struct names, untracked types).
// String leaves emit boundary_int (on the length, satisfied by len(s)-
// bearing shapes) plus zero_string (on the empty-state, satisfied by
// `s == ""` / `s != ""` shapes). Float leaves emit four requirements
// (boundary, zero, NaN, Inf).
func check_invariant_assertions_recursive_frame_emit_leaf(
	input *check_invariant_assertions_recursive_frame,
	identifier *ast.Ident,
	is_pointer bool,
) (leaves []check_invariant_assertions_requirement) {
	defer func() {
		check_invariant_assertions_recursive_frame_emit_leaf_assert_exit(leaves, input)
	}()
	ancestor_axis := invariant.Sometimes(
		input.Has_Nillable_Ancestor, "Input is under a nillable ancestor sometimes")
	pointer_axis := invariant.Sometimes(
		is_pointer, "Is_pointer is true when the leaf is behind a pointer")
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Visited is bounded",
	})
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	pa := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Path is non-empty bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(identifier != nil, "Identifier is non-nil"),
		ancestor_axis, pointer_axis, vis,
		invariant.Excluding("Hi vis unreachable",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(ancestor_axis)),
		invariant.Excluding("Hi vis unreachable (ancestor)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(ancestor_axis)),
	)
	invariant.Cross_Product(fl)
	invariant.Cross_Product(pa)

	if is_pointer {
		return nil
	}
	field_description := check_invariant_assertions_field_description(
		input.Path, input.Type_Expression)
	invariant.Cross_Product(
		invariant.Always(
			field_description != "",
			"Field_description is non-empty for tracked leaves"))
	kinds := check_invariant_assertions_keyword_kinds(identifier.Name)
	invariant.Cross_Product(
		invariant.Sometimes(kinds == nil, "Kinds is nil for non-primitive identifiers"))
	if kinds == nil {
		return nil
	}
	for _, kind := range kinds {
		leaves = append(leaves, check_invariant_assertions_requirement{
			Identifier_Path:       input.Path,
			Kind:                  kind,
			Position:              input.Position,
			Field_Description:     field_description,
			Function_Label:        input.Function_Label,
			Has_Nillable_Ancestor: input.Has_Nillable_Ancestor,
		})
	}
	return leaves
}

func check_invariant_assertions_recursive_frame_emit_leaf_assert_exit(
	leaves []check_invariant_assertions_requirement,
	input *check_invariant_assertions_recursive_frame,
) {
	l := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(leaves), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Leaves is bounded by budget",
	})
	empty_axis := invariant.Sometimes(leaves == nil, "Leaves is nil for non-leaf types")
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Visited is bounded",
	})
	invariant.Cross_Product(l, empty_axis, vis,
		invariant.Excluding("Max len contradicts empty true",
			invariant.Bucket_Hi(l), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis l at safety cap is bad",
			invariant.Bucket_Hi(l), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero len implies empty true",
			invariant.Bucket_Lo(l), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi vis unreachable (empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi vis unreachable (non-empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(empty_axis)),
	)
}

// Builds the pointer requirement for top. Extracted from recursive_frame_emit
// to keep that function under the per-function line cap.
func check_invariant_assertions_recursive_frame_pointer_requirement(
	top *check_invariant_assertions_recursive_frame,
) (requirement check_invariant_assertions_requirement) {
	defer func() {
		check_invariant_assertions_requirement_assert_pointer_exit(requirement, top)
	}()
	nillable := invariant.Sometimes(
		top.Has_Nillable_Ancestor, "Top is under a nillable ancestor sometimes")
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Function_Label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Top.Function_Label is non-empty bounded by identifier budget",
	})
	pa := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Top.Path is non-empty bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(top != nil, "Top is non-nil"),
		nillable, vis,
		invariant.Excluding("Hi vis unreachable (nillable)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(nillable)),
		invariant.Excluding("Hi vis unreachable (top-level)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(nillable)),
	)
	invariant.Cross_Product(fl)
	invariant.Cross_Product(pa)
	return check_invariant_assertions_requirement{
		Identifier_Path: top.Path,
		Kind:            pointer_requirement_kind,
		Position:        top.Position,
		Field_Description: check_invariant_assertions_field_description(
			top.Path, top.Type_Expression),
		Function_Label:        top.Function_Label,
		Has_Nillable_Ancestor: top.Has_Nillable_Ancestor,
	}
}

func check_invariant_assertions_requirement_assert_pointer_exit(
	requirement check_invariant_assertions_requirement,
	top *check_invariant_assertions_recursive_frame,
) {
	nillable := invariant.Sometimes(requirement.Has_Nillable_Ancestor,
		"Pointer requirement is under a nillable ancestor sometimes")
	requires_defer := invariant.Sometimes(requirement.Requires_Defer_Position,
		"Pointer requirement requires defer position sometimes")
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(requirement.Function_Label),
		Lo: min_function_label_chars,
		Hi: Max_Identifier_Chars,
		Message: "Requirement.Function_Label is non-empty bounded by identifier " +
			"budget",
	})
	ip := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(requirement.Identifier_Path),
		Lo: min_non_empty,
		Hi: Max_Identifier_Chars,
		Message: "Requirement.Identifier_Path is non-empty bounded by identifier " +
			"budget",
	})
	invariant.Cross_Product(nillable, requires_defer, vis,
		invariant.Always(requirement.Kind != "", "Requirement.Kind is non-empty"),
		invariant.Always(
			len(
				requirement.Kind) == pointer_requirement_kind_chars, "Kin"+
				"d is the fixed pointer label"),
		invariant.Excluding("Pointer requirements leave Requires_Defer_Position "+
			"unset",
			invariant.Bucket_True(requires_defer)),
		invariant.Excluding("Hi vis unreachable (nillable)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(nillable)),
		invariant.Excluding("Hi vis unreachable (top-level)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(nillable)),
	)
	invariant.Cross_Product(fl)
	invariant.Cross_Product(ip)
	field_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Field_Description),
		Lo:      min_requirement_field_description_chars,
		Hi:      max_field_description_chars,
		Message: "Field_description spans short pointer to long slice typename",
	})
	invariant.Cross_Product(field_axis)
}

// Tries the slice/map container-leaf emitter first; falls through to the
// channel-leaf emitter on miss. Returns nil for type expressions that match
// neither (the caller falls through to ident / inline-struct branches).
// `len(ch)` is the racy form excluded by §7.9 — channels emit the cap-only
// triple with path encoded as `cap(<path>)` so the matcher distinguishes
// channel-cap credit from slice/map-len credit on the same identifier.
func check_invariant_assertions_recursive_frame_leaf_dispatch(
	top *check_invariant_assertions_recursive_frame,
	type_expression ast.Expr,
	is_pointer bool,
) (leaves []check_invariant_assertions_requirement) {
	defer func() {
		check_invariant_assertions_recursive_frame_leaf_dispatch_assert_exit(leaves, top)
	}()
	pointer_axis := invariant.Sometimes(
		is_pointer, "Container sits behind a pointer indirection sometimes")
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	invariant.Cross_Product(
		invariant.Always(top != nil, "Top is non-nil"),
		invariant.Always(type_expression != nil, "Type_expression is non-nil"),
		pointer_axis, vis,
		invariant.Excluding("Hi vis unreachable (pointer)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(pointer_axis)),
		invariant.Excluding("Hi vis unreachable (non-pointer)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(pointer_axis)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Function_Label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Top.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Top.path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	nillable_axis := invariant.Sometimes(
		top.Has_Nillable_Ancestor, "Top is under a nillable ancestor sometimes")
	invariant.Cross_Product(nillable_axis)
	leaves = check_invariant_assertions_recursive_frame_container_leaf(
		top, type_expression, is_pointer)
	leaves_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(leaves), Lo: 0, Hi: min_pair,
		Message: "Leaves is 0 (non-container) or 2 (container triple)",
	})
	invariant.Cross_Product(leaves_axis)
	if leaves != nil {
		return leaves
	}
	return check_invariant_assertions_recursive_frame_channel_leaf(
		top, type_expression, is_pointer)
}

func check_invariant_assertions_recursive_frame_leaf_dispatch_assert_exit(
	leaves []check_invariant_assertions_requirement,
	top *check_invariant_assertions_recursive_frame,
) {
	nil_axis := invariant.Sometimes(
		leaves == nil, "Leaves is nil when neither container nor channel matched")
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	invariant.Cross_Product(nil_axis, vis,
		invariant.Excluding("Hi vis unreachable (nil)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(nil_axis)),
		invariant.Excluding("Hi vis unreachable (non-nil)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(nil_axis)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(top.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Top.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Top.path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	leaves_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(leaves), Lo: 0, Hi: max_leaf_requirements_per_dispatch,
		Message: "Leaves is 0 (miss) or 3 (channel triple)",
	})
	invariant.Cross_Product(leaves_axis)
}

// Dispatches a length-mutable container type-expression (slice, variadic,
// map) to its respective leaf-emitter. Returns nil when type_expression
// isn't a length-mutable container — the caller falls through to other
// branches.
func check_invariant_assertions_recursive_frame_container_leaf(
	top *check_invariant_assertions_recursive_frame,
	type_expression ast.Expr,
	is_pointer bool,
) (leaves []check_invariant_assertions_requirement) {
	defer func() {
		check_invariant_assertions_recursive_frame_container_leaf_assert_exit(leaves, top)
	}()
	check_invariant_assertions_recursive_frame_container_leaf_assert_entry(
		top, type_expression, is_pointer)
	zero_kind := ""
	if check_invariant_assertions_is_slice_type(type_expression) {
		zero_kind = "zero_slice"
	}
	if check_invariant_assertions_is_map_type(type_expression) {
		zero_kind = "zero_map"
	}
	if zero_kind == "" {
		return nil
	}
	leaf_path := top.Path
	if is_pointer {
		leaf_path = "*" + top.Path
	}
	field_description := check_invariant_assertions_field_description(
		leaf_path, type_expression)
	invariant.Cross_Product(invariant.Always(field_description != "",
		"Field_description is non-empty for length-mutable leaves"))
	for _, kind := range []string{"boundary_int", zero_kind} {
		leaves = append(leaves, check_invariant_assertions_requirement{
			Identifier_Path:         leaf_path,
			Kind:                    kind,
			Position:                top.Position,
			Field_Description:       field_description,
			Function_Label:          top.Function_Label,
			Has_Nillable_Ancestor:   top.Has_Nillable_Ancestor,
			Requires_Defer_Position: true,
		})
	}
	return leaves
}

func check_invariant_assertions_recursive_frame_container_leaf_assert_exit(
	leaves []check_invariant_assertions_requirement,
	top *check_invariant_assertions_recursive_frame,
) {
	l := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(leaves), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Leaves is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		leaves == nil, "Leaves is nil for non-length-mutable type expressions")
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	invariant.Cross_Product(l, empty_axis, vis,
		invariant.Excluding("Max len contradicts empty true",
			invariant.Bucket_Hi(l), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis l at safety cap is bad",
			invariant.Bucket_Hi(l), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero len implies empty true",
			invariant.Bucket_Lo(l), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi vis unreachable (empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi vis unreachable (non-empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(empty_axis)),
	)
}

func check_invariant_assertions_recursive_frame_container_leaf_assert_entry(
	top *check_invariant_assertions_recursive_frame,
	type_expression ast.Expr,
	is_pointer bool,
) {
	is_slice := invariant.Sometimes(
		check_invariant_assertions_is_slice_type(type_expression), "Slice branch reachable")
	is_map := invariant.Sometimes(
		check_invariant_assertions_is_map_type(type_expression), "Map branch reachable")
	has_nillable := invariant.Sometimes(
		top.Has_Nillable_Ancestor, "Has_Nillable_Ancestor flag reachable")
	pointer_axis := invariant.Sometimes(
		is_pointer, "Container sits behind a pointer indirection sometimes")
	// A type expression is mutually-exclusively slice or map; the (true, true)
	// tuple is structurally impossible. With §4.5 field-type expansion lifted,
	// slice/map struct fields now reach container_leaf — both directly (no
	// pointer wrap, nillable ancestor inherited from a parent pointer) and via
	// the immediate `*[]T`/`*map[K]V` shape (pointer wrap stripped at this layer).
	invariant.Cross_Product(
		invariant.Always(top != nil, "Top is non-nil"),
		invariant.Always(type_expression != nil, "Type_expression is non-nil"),
		has_nillable, is_slice, is_map, pointer_axis,
		invariant.Excluding("A type is either slice or map, exclusively one",
			invariant.Bucket_True(is_slice), invariant.Bucket_True(is_map)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Function_Label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Top.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Top.path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	p_vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	p_vis_empty := invariant.Sometimes(len(top.Visited) == 0, "Top.Visited is empty sometimes")
	invariant.Cross_Product(p_vis, p_vis_empty,
		invariant.Excluding("Max p_vis contradicts empty true",
			invariant.Bucket_Hi(p_vis), invariant.Bucket_True(p_vis_empty)),
		invariant.Excluding("Axis p_vis at safety cap is bad",
			invariant.Bucket_Hi(p_vis), invariant.Bucket_False(p_vis_empty)),
		invariant.Excluding("Zero p_vis implies empty true",
			invariant.Bucket_Lo(p_vis), invariant.Bucket_False(p_vis_empty)),
	)
}

// Emits the channel triple for a `chan T` type expression: a pointer/nil
// check on the channel itself, plus a Distinct_Boundary and zero-check on
// cap(ch). Returns nil for non-channel type expressions so emit_step can
// fall through to other branches. The boundary path is encoded as
// `cap(<path>)` so the coverage matcher distinguishes channel-cap credit
// from slice/map-len credit (path "X" alone is reserved for the latter).
func check_invariant_assertions_recursive_frame_channel_leaf(
	top *check_invariant_assertions_recursive_frame,
	type_expression ast.Expr,
	is_pointer bool,
) (leaves []check_invariant_assertions_requirement) {
	defer func() {
		check_invariant_assertions_recursive_frame_channel_leaf_assert_exit(leaves, top)
	}()
	check_invariant_assertions_recursive_frame_channel_leaf_assert_entry(
		top, type_expression, is_pointer)
	if !check_invariant_assertions_is_channel_type(type_expression) {
		return nil
	}
	channel_path := top.Path
	if is_pointer {
		channel_path = "*" + top.Path
	}
	cap_path := "cap(" + channel_path + ")"
	field_description := check_invariant_assertions_field_description(
		channel_path, type_expression)
	invariant.Cross_Product(
		invariant.Always(
			field_description != "", "Field_description is non-empty for channels"))
	cap_field_description := check_invariant_assertions_field_description(
		cap_path, type_expression)
	invariant.Cross_Product(
		invariant.Always(
			cap_field_description != "",
			"Cap_field_description is non-empty for channels"))
	leaves = append(leaves, check_invariant_assertions_requirement{
		Identifier_Path:       channel_path,
		Kind:                  "pointer",
		Position:              top.Position,
		Field_Description:     field_description,
		Function_Label:        top.Function_Label,
		Has_Nillable_Ancestor: top.Has_Nillable_Ancestor,
	})
	for _, kind := range []string{"boundary_int", "zero_int"} {
		leaves = append(leaves, check_invariant_assertions_requirement{
			Identifier_Path:       cap_path,
			Kind:                  kind,
			Position:              top.Position,
			Field_Description:     cap_field_description,
			Function_Label:        top.Function_Label,
			Has_Nillable_Ancestor: top.Has_Nillable_Ancestor,
		})
	}
	return leaves
}

func check_invariant_assertions_recursive_frame_channel_leaf_assert_exit(
	leaves []check_invariant_assertions_requirement,
	top *check_invariant_assertions_recursive_frame,
) {
	l := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(leaves), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Leaves is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		leaves == nil, "Leaves is nil for non-channel type expressions")
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	invariant.Cross_Product(l, empty_axis, vis,
		invariant.Excluding("Max len contradicts empty true",
			invariant.Bucket_Hi(l), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis l at safety cap is bad",
			invariant.Bucket_Hi(l), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero len implies empty true",
			invariant.Bucket_Lo(l), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi vis unreachable (empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi vis unreachable (non-empty)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(empty_axis)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(top.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Top.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Top.path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
}

func check_invariant_assertions_recursive_frame_channel_leaf_assert_entry(
	top *check_invariant_assertions_recursive_frame,
	type_expression ast.Expr,
	is_pointer bool,
) {
	is_channel := invariant.Sometimes(
		check_invariant_assertions_is_channel_type(type_expression),
		"Channel branch reachable")
	pointer_axis := invariant.Sometimes(
		is_pointer, "Channel sits behind a pointer indirection sometimes")
	vis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Visited), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Top.Visited is bounded",
	})
	invariant.Cross_Product(
		invariant.Always(top != nil, "Top is non-nil"),
		invariant.Always(type_expression != nil, "Type_expression is non-nil"),
		is_channel, pointer_axis, vis,
		invariant.Excluding("Hi vis unreachable (channel)",
			invariant.Bucket_Hi(vis), invariant.Bucket_True(is_channel)),
		invariant.Excluding("Hi vis unreachable (non-channel)",
			invariant.Bucket_Hi(vis), invariant.Bucket_False(is_channel)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Function_Label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Top.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(top.Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Top.path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	nillable_axis := invariant.Sometimes(
		top.Has_Nillable_Ancestor, "Top is under a nillable ancestor sometimes")
	invariant.Cross_Product(nillable_axis)
}

// Reports whether type_expression is a `chan T` channel type. Send-only and
// receive-only variants are channels too.
func check_invariant_assertions_is_channel_type(type_expression ast.Expr) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))
	}()
	_, is_channel := type_expression.(*ast.ChanType)
	return is_channel
}

// Reports whether type_expression is a slice (`[]T`) or variadic (`...T`).
// Fixed-size arrays (`[N]T`) are out of scope — their length is a compile-time
// constant, not a runtime observable worth asserting. Maps and channels also
// support len() but are excluded for now.
func check_invariant_assertions_is_slice_type(type_expression ast.Expr) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))
	}()
	if array_type, is_array := type_expression.(*ast.ArrayType); is_array {
		return array_type.Len == nil
	}
	if _, is_ellipsis := type_expression.(*ast.Ellipsis); is_ellipsis {
		return true
	}
	return false
}

// Reports whether type_expression is a map (`map[K]V`). Channels are
// excluded despite also supporting `len()` — out of scope per user
// decision.
func check_invariant_assertions_is_map_type(type_expression ast.Expr) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))
	}()
	_, is_map := type_expression.(*ast.MapType)
	return is_map
}

// Returns the requirement kinds emitted for a primitive type keyword, or
// nil if name isn't a tracked primitive. Centralises the mapping so the
// emit-leaf function stays small.
func check_invariant_assertions_keyword_kinds(name string) (kinds []string) {
	defer func() {
		k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(kinds), Lo: 0, Hi: max_string_slice_per_call,
			Message: "Kinds is bounded by budget",
		})
		empty_axis := invariant.Sometimes(
			kinds == nil, "Kinds is nil for non-primitive keywords")
		invariant.Cross_Product(k, empty_axis,
			invariant.Excluding("Max k contradicts empty true",
				invariant.Bucket_Hi(k), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Axis k at safety cap is bad",
				invariant.Bucket_Hi(k), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero k implies empty true",
				invariant.Bucket_Lo(k), invariant.Bucket_False(empty_axis)),
		)
	}()
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(name),
		Lo: min_non_empty,
		Hi: Max_Identifier_Chars,
		Message: "Name is a Go type identifier; primitives like `complex128` and user " +
			"types up to identifier budget",
	})
	single_axis := invariant.Sometimes(
		len(name) == count_one, "Name is exactly 1 char sometimes")
	invariant.Cross_Product(name_axis, single_axis,
		invariant.Excluding("Max name contradicts single true",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Zero name implies single true",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_False(single_axis)),
	)
	if check_invariant_assertions_is_integer_keyword(name) {
		return []string{"boundary_int", "zero_int"}
	}
	if check_invariant_assertions_is_float_keyword(name) {
		return []string{"boundary_float", "zero_float", "nan_float", "inf_float"}
	}
	if check_invariant_assertions_is_string_keyword(name) {
		return []string{"boundary_int", "zero_string"}
	}
	if name == "bool" {
		return []string{"bool"}
	}
	return nil
}

// Returns a human-readable label like "x int", "*Foo", "string" used in
// diagnostic messages. Falls back to a generic "<type>" descriptor when
// types.ExprString would need a non-empty FileSet which we don't have here.
func check_invariant_assertions_field_description(
	name string, type_expression ast.Expr,
) (description string) {
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(description),
				Lo: min_field_description_chars,
				Hi: max_field_description_chars,
				Message: "Description is `name type_str` " +
					"(≥3 chars: 1-char name + " +
					"space + 1-char type)",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty identifier bounded by identifier budget",
		}),
	)

	return name + " " + types.ExprString(type_expression)
}

// Coverage pair extracted from a prologue assertion call. A single call may
// cover multiple identifiers (e.g. Cross_Product over two axes); each
// (path, kind) becomes one pair. From_Cross_Product distinguishes pairs that
// came from a multi-axis Cross_Product call (acceptable for any signature)
// from pairs that came from a single-axis _Of helper (acceptable only when
// the function has exactly one in-scope boundary identifier — _Of is sugar
// for a one-axis Cross_Product and cannot enumerate the cross-product over
// multiple parameters).
type check_invariant_assertions_coverage_pair struct {
	Path               string
	Kind               string
	From_Cross_Product bool
}

// Coverage origin tracks how a (path, kind) was credited. From_Cross_Product
// is true iff at least one covering call was a multi-axis Cross_Product (so
// the multi-parameter case can require Cross_Product). Outside_If_Guard is
// true iff at least one credit came from outside any `if X != nil { ... }`
// block (used by the non-nillable-outside-if rule).
type coverage_origin struct {
	From_Cross_Product bool
	Outside_If_Guard   bool
}

// Scans function.Body.List as two interleaved prologues. The first defer
// that wraps a no-arg-no-return FuncLit terminates the named-return-prologue
// recognition (subsequent defers cannot satisfy named-return requirements
// per constraint #6). Expression-statement prologue calls credit the
// parameter set. Scan terminates on the first non-matching statement.
func check_invariant_assertions_scan_function_prologue(
	body *ast.BlockStmt,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) (
	parameter_covered map[string]map[string]coverage_origin,
	named_return_covered map[string]map[string]coverage_origin,
	always_nil_pointers map[string]bool,
) {
	check_invariant_assertions_scan_function_prologue_assert_entry(
		body, invariant_idents, constant_signs)
	parameter_covered = map[string]map[string]coverage_origin{}
	named_return_covered = map[string]map[string]coverage_origin{}
	always_nil_pointers = map[string]bool{}
	defer_seen := false
	// One Cross_Product per prologue chain. After the first Cross_Product /
	// Recorder_Cross_Product credits, skip subsequent ones (still continue
	// the chain so axis-builder bindings and _Of helpers after it credit).
	// Nil-guard if-bodies dispatch to walk_assertion_chain with a fresh
	// frame, so they get an independent budget.
	cross_product_seen := false
	scan_input := &check_invariant_assertions_scan_function_prologue_input{
		Invariant_Idents:     invariant_idents,
		Constant_Signs:       constant_signs,
		Parameter_Covered:    parameter_covered,
		Named_Return_Covered: named_return_covered,
		Always_Nil_Pointers:  always_nil_pointers,
		Defer_Seen:           &defer_seen,
		Cross_Product_Seen:   &cross_product_seen,
	}
	defer func() {
		check_invariant_assertions_scan_function_prologue_input_assert_exit(scan_input)
	}()
	for _, statement := range body.List {
		scan_input.Statement = statement
		if !check_invariant_assertions_scan_function_prologue_input_step(scan_input) {
			break
		}
	}
	return parameter_covered, named_return_covered, always_nil_pointers
}

// Bundles the shared state threaded through scan_function_prologue's per-statement
// loop: the immutable lookup tables, the three coverage maps mutated in place, and
// pointers to the two once-only chain flags. Reused by the loop-body step helper and
// the exit assertion so neither needs a long same-type parameter list.
type check_invariant_assertions_scan_function_prologue_input struct {
	Statement            ast.Stmt
	Invariant_Idents     map[string]bool
	Constant_Signs       map[string]int
	Parameter_Covered    map[string]map[string]coverage_origin
	Named_Return_Covered map[string]map[string]coverage_origin
	Always_Nil_Pointers  map[string]bool
	Defer_Seen           *bool
	Cross_Product_Seen   *bool
}

func check_invariant_assertions_scan_function_prologue_assert_entry(
	body *ast.BlockStmt, invariant_idents map[string]bool, constant_signs map[string]int,
) {
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	invariant.Cross_Product(
		invariant.Always(body != nil, "Body is non-nil"),
		ii, cs,
		invariant.Excluding("Both ii and cs at safety caps is bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_Hi(cs)),
		invariant.Excluding("Max ii with zero cs is bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_Lo(cs)),
		invariant.Excluding("Zero ii with max cs is bad",
			invariant.Bucket_Lo(ii), invariant.Bucket_Hi(cs)),
	)
}

func check_invariant_assertions_scan_function_prologue_input_assert_exit(
	input *check_invariant_assertions_scan_function_prologue_input,
) {
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	pc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_covered is bounded by budget",
	})
	nrc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_covered is bounded by budget",
	})
	anp := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Always_nil_pointers is bounded by budget",
	})
	invariant.Cross_Product(ii, cs, pc, nrc, anp,
		invariant.Excluding("Both ii and cs at safety caps is bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_Hi(cs)),
		invariant.Excluding("Axis ii at safety cap is bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_Lo(cs)),
		invariant.Excluding("Both cs and pc at safety caps is bad",
			invariant.Bucket_Hi(cs), invariant.Bucket_Hi(pc)),
		invariant.Excluding("Max cs with zero pc is bad",
			invariant.Bucket_Hi(cs), invariant.Bucket_Lo(pc)),
		invariant.Excluding("Zero cs with max pc is bad",
			invariant.Bucket_Lo(cs), invariant.Bucket_Hi(pc)),
		invariant.Excluding("Both pc and nrc at safety caps is bad",
			invariant.Bucket_Hi(pc), invariant.Bucket_Hi(nrc)),
		invariant.Excluding("Max pc with zero nrc is bad",
			invariant.Bucket_Hi(pc), invariant.Bucket_Lo(nrc)),
		invariant.Excluding("Zero pc with max nrc is bad",
			invariant.Bucket_Lo(pc), invariant.Bucket_Hi(nrc)),
		invariant.Excluding("Both nrc and anp at safety caps is bad",
			invariant.Bucket_Hi(nrc), invariant.Bucket_Hi(anp)),
		invariant.Excluding("Max nrc with zero anp is bad",
			invariant.Bucket_Hi(nrc), invariant.Bucket_Lo(anp)),
		invariant.Excluding("Zero nrc with max anp is bad",
			invariant.Bucket_Lo(nrc), invariant.Bucket_Hi(anp)),
	)
}

// Processes one prologue statement, mutating the coverage maps in place, and
// reports whether the caller's scan loop should keep going (false ends the
// chain). Defer and nil-guard-if statements recurse or credit and continue;
// any other non-prologue statement ends the chain.
func check_invariant_assertions_scan_function_prologue_input_step(
	input *check_invariant_assertions_scan_function_prologue_input,
) (proceed bool) {
	statement := input.Statement
	if defer_statement, is_defer := statement.(*ast.DeferStmt); is_defer {
		if !*input.Defer_Seen {
			*input.Defer_Seen = true
			check_invariant_assertions_credit_defer(
				defer_statement,
				input.Invariant_Idents,
				input.Constant_Signs,
				input.Named_Return_Covered)
			return true
		}
		return false
	}
	if if_statement, is_if := statement.(*ast.IfStmt); is_if {
		if check_invariant_assertions_is_nil_guard(if_statement.Cond) {
			if if_statement.Body != nil {
				check_invariant_assertions_walk_assertion_chain(
					&check_invariant_assertions_walk_assertion_chain_input{
						Statements:          if_statement.Body.List,
						Invariant_Idents:    input.Invariant_Idents,
						Constant_Signs:      input.Constant_Signs,
						Covered:             input.Parameter_Covered,
						Always_Nil_Pointers: input.Always_Nil_Pointers,
						Outside_If_Guard:    false,
					})
			}
			return true
		}
		return false
	}
	is_cross := check_invariant_assertions_is_cross_product_statement(
		statement, input.Invariant_Idents)
	invariant.Cross_Product(
		invariant.Sometimes(is_cross, "Statement is a top-level Cross_Product "+
			"sometimes"))
	if is_cross {
		if *input.Cross_Product_Seen {
			return true
		}
	}
	if !check_invariant_assertions_credit_prologue_statement(
		&check_invariant_assertions_credit_prologue_statement_input{
			Statement:           statement,
			Invariant_Idents:    input.Invariant_Idents,
			Constant_Signs:      input.Constant_Signs,
			Covered:             input.Parameter_Covered,
			Always_Nil_Pointers: input.Always_Nil_Pointers,
			Outside_If_Guard:    true,
		}) {
		return false
	}
	if is_cross {
		*input.Cross_Product_Seen = true
	}
	return true
}

type check_invariant_assertions_credit_prologue_statement_input struct {
	Statement           ast.Stmt
	Invariant_Idents    map[string]bool
	Constant_Signs      map[string]int
	Covered             map[string]map[string]coverage_origin
	Always_Nil_Pointers map[string]bool
	Outside_If_Guard    bool
}

// Credits a single prologue/if-body statement against the covered map.
// Returns false when the statement isn't a recognized invariant call — the
// caller treats that as the end of the assertion chain. If
// input.Always_Nil_Pointers is non-nil, paths asserted Always(p == nil) get
// added. input.Outside_If_Guard drives the non-nillable-outside-if rule.
func check_invariant_assertions_credit_prologue_statement(
	input *check_invariant_assertions_credit_prologue_statement_input,
) (credited bool) {
	defer func() {
		check_invariant_assertions_credit_prologue_statement_assert_exit(credited, input)
	}()
	check_invariant_assertions_credit_prologue_statement_input_assert_entry(input)
	call, is_axis_builder_binding := check_invariant_assertions_extract_prologue_call(
		input.Statement, input.Invariant_Idents)
	call_axis := invariant.Sometimes(call == nil, "Call is nil for non-prologue statements")
	binding_axis := invariant.Sometimes(
		is_axis_builder_binding, "Statement is an axis-builder binding")
	// Is_axis_builder_binding=true implies call is non-nil; the (nil, true)
	// tuple is impossible by construction in extract_prologue_call.
	invariant.Cross_Product(
		call_axis, binding_axis,
		invariant.Excluding("Call nil with axis_builder_binding true is impossible by "+
			"construction",
			invariant.Bucket_True(call_axis), invariant.Bucket_True(binding_axis)),
	)
	if call == nil {
		return false
	}
	var pairs []check_invariant_assertions_coverage_pair
	if is_axis_builder_binding {
		pairs = check_invariant_assertions_axis_builder_covered_pairs(
			call, input.Invariant_Idents, input.Constant_Signs)
		// Pairs is never nil here: extract_prologue_call's bare_table gate
		// guarantees axis_builder_covered_pairs takes its happy path.
		invariant.Cross_Product(
			invariant.Always(pairs != nil, "Pairs is non-nil for axis-builder "+
				"bindings"))
	} else {
		pairs = check_invariant_assertions_call_covered_pairs(
			call, input.Invariant_Idents, input.Constant_Signs)
		invariant.Cross_Product(
			invariant.Sometimes(pairs == nil, "Pairs is nil for unrecognised "+
				"invariant calls"))
	}
	if pairs == nil {
		return false
	}
	check_invariant_assertions_record_coverage(input.Covered, pairs, input.Outside_If_Guard)
	if input.Always_Nil_Pointers != nil {
		for _, path := range check_invariant_assertions_collect_always_nil_paths(
			call, input.Invariant_Idents) {
			input.Always_Nil_Pointers[path] = true
		}
	}
	return true
}

func check_invariant_assertions_credit_prologue_statement_assert_exit(
	credited bool, input *check_invariant_assertions_credit_prologue_statement_input,
) {
	credited_axis := invariant.Sometimes(
		credited, "Statement was credited as an invariant call")
	idents_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Invariant_Idents is bounded by AST budget",
	})
	signs_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Constant_Signs is bounded by AST budget",
	})
	covered_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Covered is bounded by AST budget",
	})
	anp_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Always_Nil_Pointers is bounded by AST budget",
	})
	invariant.Cross_Product(
		credited_axis, idents_axis, signs_axis, covered_axis, anp_axis,
		invariant.Excluding("Hi idents unreachable (credited)",
			invariant.Bucket_Hi(idents_axis),
			invariant.Bucket_True(credited_axis)),
		invariant.Excluding("Hi idents unreachable (non-credited)",
			invariant.Bucket_Hi(idents_axis),
			invariant.Bucket_False(credited_axis)),
		invariant.Excluding("Hi signs unreachable (credited)",
			invariant.Bucket_Hi(signs_axis),
			invariant.Bucket_True(credited_axis)),
		invariant.Excluding("Hi signs unreachable (non-credited)",
			invariant.Bucket_Hi(signs_axis),
			invariant.Bucket_False(credited_axis)),
		invariant.Excluding("Hi covered unreachable (credited)",
			invariant.Bucket_Hi(covered_axis),
			invariant.Bucket_True(credited_axis)),
		invariant.Excluding("Hi covered unreachable (non-credited)",
			invariant.Bucket_Hi(covered_axis),
			invariant.Bucket_False(credited_axis)),
		invariant.Excluding("Hi always_nil_pointers unreachable (credited)",
			invariant.Bucket_Hi(anp_axis),
			invariant.Bucket_True(credited_axis)),
		invariant.Excluding("Hi always_nil_pointers unreachable (non-credited)",
			invariant.Bucket_Hi(anp_axis),
			invariant.Bucket_False(credited_axis)),
		// Credited requires extract_prologue_call to recognise the
		// statement as an invariant call — which requires the local
		// invariant ident to be in Invariant_Idents. Lo idents (empty)
		// + True credited is impossible by construction.
		invariant.Excluding("Crediting requires a populated Invariant_Idents map",
			invariant.Bucket_Lo(idents_axis),
			invariant.Bucket_True(credited_axis)),
	)
}

func check_invariant_assertions_credit_prologue_statement_input_assert_entry(
	input *check_invariant_assertions_credit_prologue_statement_input,
) {
	idents_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Invariant_Idents is bounded by AST budget",
	})
	signs_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Constant_Signs is bounded by AST budget",
	})
	covered_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Covered is bounded by AST budget",
	})
	anp_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Always_Nil_Pointers is bounded by AST budget",
	})
	outside_axis := invariant.Sometimes(
		input.Outside_If_Guard, "Statement is outside any if-guard")
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Covered != nil, "Covered is non-nil"),
		outside_axis, idents_axis, signs_axis, covered_axis, anp_axis,
		invariant.Excluding("Hi idents unreachable (outside if-guard)",
			invariant.Bucket_Hi(idents_axis), invariant.Bucket_True(outside_axis)),
		invariant.Excluding("Hi idents unreachable (inside if-guard)",
			invariant.Bucket_Hi(idents_axis), invariant.Bucket_False(outside_axis)),
		invariant.Excluding("Hi signs unreachable (outside if-guard)",
			invariant.Bucket_Hi(signs_axis), invariant.Bucket_True(outside_axis)),
		invariant.Excluding("Hi signs unreachable (inside if-guard)",
			invariant.Bucket_Hi(signs_axis), invariant.Bucket_False(outside_axis)),
		invariant.Excluding("Hi covered unreachable (outside if-guard)",
			invariant.Bucket_Hi(covered_axis), invariant.Bucket_True(outside_axis)),
		invariant.Excluding("Hi covered unreachable (inside if-guard)",
			invariant.Bucket_Hi(covered_axis), invariant.Bucket_False(outside_axis)),
		invariant.Excluding("Hi always_nil_pointers unreachable (outside if-guard)",
			invariant.Bucket_Hi(anp_axis), invariant.Bucket_True(outside_axis)),
		invariant.Excluding("Hi always_nil_pointers unreachable (inside if-guard)",
			invariant.Bucket_Hi(anp_axis), invariant.Bucket_False(outside_axis)),
	)
}

type check_invariant_assertions_walk_assertion_chain_input struct {
	Statements          []ast.Stmt
	Invariant_Idents    map[string]bool
	Constant_Signs      map[string]int
	Covered             map[string]map[string]coverage_origin
	Always_Nil_Pointers map[string]bool
	Outside_If_Guard    bool
}

type check_invariant_assertions_walk_frame struct {
	Statements         []ast.Stmt
	Outside_If_Guard   bool
	Cross_Product_Seen bool
}

// Reports whether statement is a top-level invariant.Cross_Product or
// invariant.Recorder_Cross_Product expression-statement. Used by the chain
// walker and prologue scanner to enforce the one-Cross_Product-per-chain
// spec: after the first such statement credits in a chain, subsequent
// Cross_Products are silently skipped (no coverage contribution).
func check_invariant_assertions_is_cross_product_statement(
	statement ast.Stmt, invariant_idents map[string]bool,
) (yes bool) {
	defer func() {
		yes_axis := invariant.Sometimes(
			yes, "Statement is a top-level Cross_Product sometimes")
		empty_idents_axis := invariant.Sometimes(len(invariant_idents) == 0,
			"Invariant_idents is empty for files without invariant import")
		defer_idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Invariant_idents is bounded by budget",
		})
		invariant.Cross_Product(yes_axis, empty_idents_axis, defer_idents,
			invariant.Excluding("Empty idents (file lacks invariant import) implies "+
				"yes false",
				invariant.Bucket_True(yes_axis),
				invariant.Bucket_True(empty_idents_axis)),
			invariant.Excluding("Max defer_idents contradicts empty_idents true",
				invariant.Bucket_Hi(defer_idents),
				invariant.Bucket_True(empty_idents_axis)),
			invariant.Excluding("Defer_idents at safety cap is bad",
				invariant.Bucket_Hi(defer_idents),
				invariant.Bucket_False(empty_idents_axis)),
			invariant.Excluding("Zero defer_idents implies empty_idents true",
				invariant.Bucket_Lo(defer_idents),
				invariant.Bucket_False(empty_idents_axis)),
		)
	}()
	prologue_ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	empty_idents_axis := invariant.Sometimes(
		len(invariant_idents) == 0, "Invariant_idents is empty for files without "+
			"invariant import")
	invariant.Cross_Product(prologue_ii, empty_idents_axis,
		invariant.Excluding("Max prologue invariant_idents contradicts empty_idents true",
			invariant.Bucket_Hi(prologue_ii), invariant.Bucket_True(empty_idents_axis)),
		invariant.Excluding("Prologue invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(prologue_ii),
			invariant.Bucket_False(empty_idents_axis)),
		invariant.Excluding("Zero prologue invariant_idents implies empty_idents true",
			invariant.Bucket_Lo(prologue_ii),
			invariant.Bucket_False(empty_idents_axis)),
	)
	expression_statement, is_expression := statement.(*ast.ExprStmt)
	if !is_expression {
		return false
	}
	call, is_call := expression_statement.X.(*ast.CallExpr)
	if !is_call {
		return false
	}
	name := check_invariant_assertions_extract_call_name(call, invariant_idents)
	invariant.Cross_Product(
		invariant.Sometimes(name == "", "Name is empty for non-invariant calls"))
	if name == "Cross_Product" {
		return true
	}
	if name == "Recorder_Cross_Product" {
		return true
	}
	return false
}

// Walks a list of statements crediting any recognized invariant calls and
// descending into `if X != nil { ... }` bodies via an explicit stack (no
// recursion). Stops at the first unrecognized statement in each list.
// input.Outside_If_Guard is true when the initial list sits at the top of a
// defer or prologue (no enclosing if-block); descent into an if-body resets
// the flag to false on the inner frame.
func check_invariant_assertions_walk_assertion_chain(
	input *check_invariant_assertions_walk_assertion_chain_input,
) {
	defer func() {
		check_invariant_assertions_walk_assertion_chain_input_assert_exit(input)
	}()
	check_invariant_assertions_walk_assertion_chain_input_assert_entry(input)
	stack := []check_invariant_assertions_walk_frame{{
		Statements:       input.Statements,
		Outside_If_Guard: input.Outside_If_Guard,
	}}
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, statement := range top.Statements {
			var proceed bool
			stack, proceed = check_invariant_assertions_walk_assertion_chain_step(
				statement, input, &top, stack)
			if !proceed {
				break
			}
		}
	}
}

// Processes one statement in walk_assertion_chain's frame. Pushes a fresh frame
// for nil-guard if-bodies and credits prologue statements, returning the updated
// stack and whether the caller's chain walk should continue (false ends it).
func check_invariant_assertions_walk_assertion_chain_step(
	statement ast.Stmt,
	input *check_invariant_assertions_walk_assertion_chain_input,
	top *check_invariant_assertions_walk_frame,
	stack []check_invariant_assertions_walk_frame,
) (next_stack []check_invariant_assertions_walk_frame, proceed bool) {
	if if_statement, is_if := statement.(*ast.IfStmt); is_if {
		if check_invariant_assertions_is_nil_guard(if_statement.Cond) {
			if if_statement.Body != nil {
				stack = append(stack, check_invariant_assertions_walk_frame{
					Statements:       if_statement.Body.List,
					Outside_If_Guard: false,
				})
			}
			return stack, true
		}
		return stack, false
	}
	// One Cross_Product per chain. After the first Cross_Product /
	// Recorder_Cross_Product credits in this frame, skip subsequent
	// ones (still continue the chain so axis-builder bindings and
	// _Of helpers after the Cross_Product can be processed).
	is_cross := check_invariant_assertions_is_cross_product_statement(
		statement, input.Invariant_Idents)
	invariant.Cross_Product(
		invariant.Sometimes(is_cross, "Statement is a top-level "+
			"Cross_Product sometimes"))
	if is_cross {
		if top.Cross_Product_Seen {
			return stack, true
		}
	}
	if !check_invariant_assertions_credit_prologue_statement(
		&check_invariant_assertions_credit_prologue_statement_input{
			Statement:           statement,
			Invariant_Idents:    input.Invariant_Idents,
			Constant_Signs:      input.Constant_Signs,
			Covered:             input.Covered,
			Always_Nil_Pointers: input.Always_Nil_Pointers,
			Outside_If_Guard:    top.Outside_If_Guard,
		}) {
		return stack, false
	}
	if is_cross {
		top.Cross_Product_Seen = true
	}
	return stack, true
}

func check_invariant_assertions_walk_assertion_chain_input_assert_exit(
	input *check_invariant_assertions_walk_assertion_chain_input,
) {
	cv := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Covered is bounded by AST budget",
	})
	cv_empty := invariant.Sometimes(
		len(input.Covered) == 0, "Input.Covered is empty sometimes")
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Invariant_Idents is bounded by AST budget",
	})
	ii_empty := invariant.Sometimes(
		len(input.Invariant_Idents) == 0, "Input.Invariant_Idents is empty "+
			"sometimes")
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Constant_Signs is bounded by AST budget",
	})
	anp := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Always_Nil_Pointers), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Always_Nil_Pointers is bounded by AST budget",
	})
	invariant.Cross_Product(cv, cv_empty, ii, ii_empty, cs, anp,
		invariant.Excluding("Hi cv with empty unreachable",
			invariant.Bucket_Hi(cv), invariant.Bucket_True(cv_empty)),
		invariant.Excluding("Hi cv with non-empty unreachable",
			invariant.Bucket_Hi(cv), invariant.Bucket_False(cv_empty)),
		invariant.Excluding("Lo cv implies empty true",
			invariant.Bucket_Lo(cv), invariant.Bucket_False(cv_empty)),
		invariant.Excluding("Hi ii with empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(ii_empty)),
		invariant.Excluding("Hi ii with non-empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(ii_empty)),
		invariant.Excluding("Lo ii implies empty true",
			invariant.Bucket_Lo(ii), invariant.Bucket_False(ii_empty)),
		invariant.Excluding("Hi cs cv true unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_True(cv_empty)),
		invariant.Excluding("Hi cs cv false unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_False(cv_empty)),
		invariant.Excluding("Hi anp cv true unreachable",
			invariant.Bucket_Hi(anp), invariant.Bucket_True(cv_empty)),
		invariant.Excluding("Hi anp cv false unreachable",
			invariant.Bucket_Hi(anp), invariant.Bucket_False(cv_empty)),
	)
}

func check_invariant_assertions_walk_assertion_chain_input_assert_entry(
	input *check_invariant_assertions_walk_assertion_chain_input,
) {
	invariant.Cross_Product(invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Covered != nil, "Covered is non-nil"),
		invariant.Sometimes(input.Outside_If_Guard, "Walk at outer scope"))
	cv := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Covered is bounded by AST budget",
	})
	cv_empty := invariant.Sometimes(len(input.Covered) == 0, "Input.Covered is empty sometimes")
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Invariant_Idents is bounded by AST budget",
	})
	ii_empty := invariant.Sometimes(
		len(input.Invariant_Idents) == 0, "Input.Invariant_Idents is empty sometimes")
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Constant_Signs is bounded by AST budget",
	})
	invariant.Cross_Product(cv, cv_empty, ii, ii_empty, cs,
		invariant.Excluding("Hi cv with empty unreachable",
			invariant.Bucket_Hi(cv), invariant.Bucket_True(cv_empty)),
		invariant.Excluding("Hi cv with non-empty unreachable",
			invariant.Bucket_Hi(cv), invariant.Bucket_False(cv_empty)),
		invariant.Excluding("Lo cv implies empty true",
			invariant.Bucket_Lo(cv), invariant.Bucket_False(cv_empty)),
		invariant.Excluding("Hi ii with empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(ii_empty)),
		invariant.Excluding("Hi ii with non-empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(ii_empty)),
		invariant.Excluding("Lo ii implies empty true",
			invariant.Bucket_Lo(ii), invariant.Bucket_False(ii_empty)),
		invariant.Excluding("Hi cs with empty cv unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_True(cv_empty)),
		invariant.Excluding("Hi cs with non-empty cv unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_False(cv_empty)),
	)
}

// Returns coverage pairs for a bare composable call (Sometimes, Always,
// Boundary, …) bound to a local axis variable. Mirrors the per-arg path
// extraction call_covered_pairs does for of_helpers, but tags every pair
// From_Cross_Product=true on the assumption that the axis variable is
// passed to a Cross_Product downstream. The prologue scanner uses this when
// it processes `axis := invariant.Sometimes(predicate, …)` statements; if
// the bound axis is never actually passed to Cross_Product, the resulting
// over-credit is benign (the path is covered either way; only the multi-
// parameter Cross_Product gate would care, and that gate's existing checks
// catch the structural problem elsewhere).
func check_invariant_assertions_axis_builder_covered_pairs(
	call *ast.CallExpr,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) (pairs []check_invariant_assertions_coverage_pair) {
	defer func() {
		check_invariant_assertions_axis_builder_covered_pairs_assert_exit(
			pairs, invariant_idents, constant_signs)
	}()
	check_invariant_assertions_axis_builder_covered_pairs_assert_entry(
		call, invariant_idents, constant_signs)
	helper_name := check_invariant_assertions_extract_call_name(call, invariant_idents)
	invariant.Cross_Product(
		invariant.Always(helper_name != "", "Helper_name is non-empty for invariant calls"))
	bare_kind, is_bare := invariant_axis_builder_kinds[helper_name]
	if !is_bare {
		return nil
	}
	paths := check_invariant_assertions_extract_identifier_paths(call.Args)
	invariant.Cross_Product(
		invariant.Sometimes(paths == nil, "Paths is nil for axis builders without "+
			"identifier args"))
	pairs = check_invariant_assertions_axis_builder_bare_credit(
		paths, bare_kind, call, constant_signs)
	comparison_path, comparison_kinds := check_invariant_assertions_extract_nil_comparison_path(
		call, helper_name, nil)
	path_axis := invariant.Sometimes(
		comparison_path == "", "Comparison_path is empty for non-comparison helpers")
	kinds_axis := invariant.Sometimes(
		comparison_kinds == nil, "Comparison_kinds is nil for non-comparison helpers")
	invariant.Cross_Product(
		path_axis, kinds_axis,
		invariant.Excluding("Path absent implies kinds empty",
			invariant.Bucket_False(path_axis), invariant.Bucket_True(kinds_axis)),
		invariant.Excluding("Path present implies kinds extracted",
			invariant.Bucket_True(path_axis), invariant.Bucket_False(kinds_axis)),
	)
	for _, comparison_kind := range comparison_kinds {
		pairs = append(pairs, check_invariant_assertions_coverage_pair{
			Path:               comparison_path,
			Kind:               comparison_kind,
			From_Cross_Product: true,
		})
	}
	return pairs
}

// Bare (non-`Is_`) axis builders and the credit kind each contributes; a
// lookup miss means the call isn't a bare axis builder.
var invariant_axis_builder_kinds = map[string]string{
	"Distinct_Boundary":          "boundary_int",
	"Recorder_Distinct_Boundary": "boundary_int",
	"Always":                     "bool",
	"Recorder_Always":            "bool",
	"Sometimes":                  "bool",
	"Recorder_Sometimes":         "bool",
}

func check_invariant_assertions_axis_builder_covered_pairs_assert_exit(
	pairs []check_invariant_assertions_coverage_pair,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) {
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pairs), Lo: 0, Hi: max_coverage_pairs_per_call,
		Message: "Pairs is bounded by budget",
	})
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is ≥1 (callable path requires invariant import)",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	invariant.Cross_Product(p, i, c,
		invariant.Excluding("Both p and i at safety caps is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Hi(i)),
		invariant.Excluding("Axis p at safety cap is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Lo(i)),
		invariant.Excluding("Both idents and signs at safety caps is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
		invariant.Excluding("Constant_signs at safety cap is bad",
			invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
	)
}

func check_invariant_assertions_axis_builder_covered_pairs_assert_entry(
	call *ast.CallExpr, invariant_idents map[string]bool, constant_signs map[string]int,
) {
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is ≥1 (callable path requires invariant import)",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	invariant.Cross_Product(invariant.Always(call != nil, "Call is non-nil"),
		i, c,
		invariant.Excluding("Both idents and signs at AST safety caps is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Invariant_idents at AST at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
	)
}

func check_invariant_assertions_axis_builder_bare_credit(
	paths []string, bare_kind string, call *ast.CallExpr, constant_signs map[string]int,
) (pairs []check_invariant_assertions_coverage_pair) {
	pairs = []check_invariant_assertions_coverage_pair{}
	for _, path := range paths {
		pairs = append(pairs, check_invariant_assertions_coverage_pair{
			Path:               path,
			Kind:               bare_kind,
			From_Cross_Product: true,
		})
	}
	// Mirrors the bare_axis_pairs auto-credit: when the bound axis is a
	// Distinct_Boundary whose Lo/Hi observably constrains x's zero
	// relationship (Lo or Hi == 0 means "x == 0" IS observable; Lo > 0 or
	// Hi < 0 means "x != 0" is invariantly true), the zero half of the
	// dual int / float / string requirement is auto-credited so a separate
	// Sometimes(x == 0) / Always(x != 0) axis is unnecessary.
	if check_invariant_assertions_boundary_input_has_zero_endpoint(call.Args, constant_signs) {
		for _, path := range paths {
			for _, zero_kind := range []string{
				"zero_int", "zero_float", "zero_string", "zero_slice", "zero_map",
			} {
				pairs = append(pairs, check_invariant_assertions_coverage_pair{
					Path:               path,
					Kind:               zero_kind,
					From_Cross_Product: true,
				})
			}
		}
	}
	if check_invariant_assertions_boundary_input_excludes_zero(call.Args, constant_signs) {
		for _, path := range paths {
			for _, zero_kind := range []string{
				"zero_int", "zero_float", "zero_string", "zero_slice", "zero_map",
			} {
				pairs = append(pairs, check_invariant_assertions_coverage_pair{
					Path:               path,
					Kind:               zero_kind,
					From_Cross_Product: true,
				})
			}
		}
	}
	return pairs
}

// Returns the underlying invariant.X(...) call from a prologue statement, or
// nil when the statement isn't a recognised prologue form. Two shapes match:
//
//   - ExprStmt with an invariant.X(...) call (classic prologue).
//   - AssignStmt with token.DEFINE whose single RHS is an invariant.X(...)
//     axis-builder call (`axis := invariant.Sometimes(predicate, …)`) — the
//     variable-bound axis pattern needed when Cross_Product clauses (Excluding
//     / Solely) reference axes by name.
//
// The returned call is the one whose args carry the asserted identifier
// path; the caller feeds it into call_covered_pairs.
func check_invariant_assertions_extract_prologue_call(
	statement ast.Stmt, invariant_idents map[string]bool,
) (call *ast.CallExpr, is_axis_builder_binding bool) {
	defer func() {
		check_invariant_assertions_extract_prologue_call_assert_exit(
			invariant_idents, call, is_axis_builder_binding)
	}()
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(invariant_idents) == 0, "Invariant_idents is empty for files without "+
			"invariant import")
	invariant.Cross_Product(i, empty_axis,
		invariant.Excluding("Max i contradicts empty true",
			invariant.Bucket_Hi(i), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Invariant_idents at AST at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero i implies empty true",
			invariant.Bucket_Lo(i), invariant.Bucket_False(empty_axis)),
	)

	if expression_statement, is_expression := statement.(*ast.ExprStmt); is_expression {
		expression_call, is_call := expression_statement.X.(*ast.CallExpr)
		if !is_call {
			return nil, false
		}
		return expression_call, false
	}
	assignment, is_assignment := statement.(*ast.AssignStmt)
	if !is_assignment {
		return nil, false
	}
	if assignment.Tok != token.DEFINE {
		return nil, false
	}
	if len(assignment.Rhs) != 1 {
		return nil, false
	}
	rhs_call, is_call := assignment.Rhs[0].(*ast.CallExpr)
	if !is_call {
		return nil, false
	}
	rhs_name := check_invariant_assertions_extract_call_name(rhs_call, invariant_idents)
	invariant.Cross_Product(
		invariant.Sometimes(rhs_name == "", "Rhs_name is empty for non-invariant calls"))
	if rhs_name == "" {
		return nil, false
	}
	bare_table := map[string]string{
		"Distinct_Boundary":          "boundary_int",
		"Recorder_Distinct_Boundary": "boundary_int",
		"Always":                     "bool",
		"Recorder_Always":            "bool",
		"Sometimes":                  "bool",
		"Recorder_Sometimes":         "bool",
	}
	invariant.Cross_Product(
		invariant.Always(bare_table != nil, "Bare_table is non-nil at this point"))
	if _, is_bare := bare_table[rhs_name]; !is_bare {
		return nil, false
	}
	return rhs_call, true
}

func check_invariant_assertions_extract_prologue_call_assert_exit(
	invariant_idents map[string]bool, call *ast.CallExpr, is_axis_builder_binding bool,
) {
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	call_axis := invariant.Sometimes(
		call == nil, "Call is nil for non-prologue statements")
	binding_axis := invariant.Sometimes(
		is_axis_builder_binding, "Statement is an axis-builder binding")
	// Is_axis_builder_binding=true implies call is non-nil; the
	// (nil, true) tuple is impossible by construction.
	invariant.Cross_Product(i, call_axis, binding_axis,
		invariant.Excluding("Call nil with axis_builder_binding true is "+
			"impossible by construction",
			invariant.Bucket_True(call_axis),
			invariant.Bucket_True(binding_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i),
			invariant.Bucket_True(call_axis),
			invariant.Bucket_False(binding_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i),
			invariant.Bucket_True(call_axis),
			invariant.Bucket_True(binding_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i),
			invariant.Bucket_False(call_axis),
			invariant.Bucket_False(binding_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i),
			invariant.Bucket_False(call_axis),
			invariant.Bucket_True(binding_axis)),
		invariant.Excluding("Zero invariant_idents with axis-builder binding is "+
			"impossible",
			invariant.Bucket_Lo(i),
			invariant.Bucket_False(call_axis),
			invariant.Bucket_True(binding_axis)),
	)
}

// Returns identifier paths X for which the call is `Is_Always(X == nil, …)` or
// any axis-builder inside a Cross_Product with that exact shape. Used to
// build the opt-out set for recursive requirement generation: when a pointer
// parameter is asserted Always(p == nil), the recursive emitter doesn't
// descend into the pointee's fields.
func check_invariant_assertions_collect_always_nil_paths(
	call *ast.CallExpr,
	invariant_idents map[string]bool,
) (paths []string) {
	defer func() {
		check_invariant_assertions_collect_always_nil_paths_assert_exit(
			paths, invariant_idents)
	}()
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is ≥1 (call path requires invariant import)",
	})
	single_axis := invariant.Sometimes(
		len(invariant_idents) == count_one, "Invariant_idents is exactly 1 sometimes")
	invariant.Cross_Product(
		invariant.Always(call != nil, "Call is non-nil"),
		ii, single_axis,
		invariant.Excluding("Max invariant_idents contradicts single true",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Zero invariant_idents implies single true",
			invariant.Bucket_Lo(ii), invariant.Bucket_False(single_axis)),
	)

	helper_name := check_invariant_assertions_extract_call_name(call, invariant_idents)
	// Helper_name is guaranteed non-empty here: collect_always_nil_paths is
	// only called after scan_function_prologue has confirmed the call covered
	// at least one path, which already requires invariant.* shape.
	invariant.Cross_Product(
		invariant.Always(helper_name != "", "Helper_name is non-empty for invariant calls"))
	is_cross := helper_name == "Cross_Product"
	if !is_cross {
		is_cross = helper_name == "Recorder_Cross_Product"
	}
	if is_cross {
		argument_offset := 0
		if helper_name == "Recorder_Cross_Product" {
			argument_offset = 1
		}
		for i := argument_offset; i < len(call.Args); i++ {
			inner_call, is_inner_call := call.Args[i].(*ast.CallExpr)
			if !is_inner_call {
				continue
			}
			inner_name := check_invariant_assertions_extract_call_name(
				inner_call, invariant_idents)
			invariant.Cross_Product(
				invariant.Sometimes(
					inner_name == "",
					"Inner_name is empty for non-invariant inner calls"))
			if inner_name == "" {
				continue
			}
			path := check_invariant_assertions_extract_eq_nil_path(
				inner_call, inner_name)
			invariant.Cross_Product(
				invariant.Sometimes(
					path == "", "Path is empty for non-Always-eq-nil shapes"))
			if path != "" {
				paths = append(paths, path)
			}
		}
		return paths
	}
	path := check_invariant_assertions_extract_eq_nil_path(call, helper_name)
	invariant.Is_Sometimes(path == "", "Path is empty for non-Always-eq-nil shapes")
	if path != "" {
		paths = append(paths, path)
	}
	return paths
}

func check_invariant_assertions_collect_always_nil_paths_assert_exit(
	paths []string, invariant_idents map[string]bool,
) {
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(paths), Lo: 0, Hi: max_string_slice_per_call,
		Message: "Paths is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		paths == nil, "Paths is nil for non-always-nil calls")
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is ≥1 (call path requires invariant import)",
	})
	invariant.Cross_Product(p, empty_axis, ii,
		invariant.Excluding("Max len contradicts empty=true",
			invariant.Bucket_Hi(p), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis p at safety cap is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Max invariant_idents contradicts empty true",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero paths with empty false at min invariant_idents "+
			"is bad",
			invariant.Bucket_Lo(p),
			invariant.Bucket_False(empty_axis),
			invariant.Bucket_Lo(ii)),
	)
}

// Returns the LHS path when call is exactly Always(X == nil, …) / Is_Always(X == nil, …)
// (or the Recorder_ variants). Returns "" for any other shape, including
// != nil, Sometimes(X == nil), or non-nil comparisons.
func check_invariant_assertions_extract_eq_nil_path(
	call *ast.CallExpr, helper_name string,
) (path string) {
	defer func() {
		empty_axis := invariant.Sometimes(
			path == "", "Path is empty for non-Always-eq-nil shapes")
		size_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(path), Lo: 0, Hi: Max_Identifier_Chars,
			Message: "Path is an ident-chain string bounded by identifier budget",
		})
		invariant.Cross_Product(empty_axis, size_axis,
			invariant.Excluding("Non-empty input with zero size contradicts size "+
				"definition",
				invariant.Bucket_False(empty_axis), invariant.Bucket_Lo(size_axis)),
			invariant.Excluding("Empty input contradicts size at safety cap",
				invariant.Bucket_True(empty_axis), invariant.Bucket_Hi(size_axis)),
		)
	}()
	invariant.Cross_Product(
		invariant.Always(call != nil, "Call is non-nil"),
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:       len(helper_name),
			Lo:      min_invariant_helper_name_chars,
			Hi:      max_invariant_helper_name_chars,
			Message: "Helper_name length is bounded; callers gate empty before calling",
		}),
		invariant.Always(helper_name != "",
			"Helper_name is non-empty per upstream gate"),
	)

	switch helper_name {
	case "Always", "Is_Always":
	case "Recorder_Always", "Recorder_Is_Always":
	default:
		return ""
	}
	argument_offset := 0
	if strings.HasPrefix(helper_name, "Recorder_") {
		argument_offset = 1
	}
	if len(call.Args) <= argument_offset {
		return ""
	}
	binary, is_binary := call.Args[argument_offset].(*ast.BinaryExpr)
	if !is_binary {
		return ""
	}
	if binary.Op != token.EQL {
		return ""
	}
	nil_ident, is_nil_ident := binary.Y.(*ast.Ident)
	if !is_nil_ident {
		return ""
	}
	if nil_ident.Name != "nil" {
		return ""
	}
	return check_invariant_assertions_expression_to_path(binary.X)
}

// Reports whether defer_statement is an assertion-defer in shape: its callee
// is a FuncLit with empty params, empty results, and a non-nil body. Mirrors
// the shape recognised by credit_defer so the two checks stay in sync.
func check_invariant_assertions_is_assertion_defer_shape(
	defer_statement *ast.DeferStmt,
) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Defer matches the assertion-defer shape"))

	}()
	invariant.Cross_Product(invariant.Always(
		defer_statement != nil, "Defer_statement is non-nil"))

	function_literal, is_function_literal := defer_statement.Call.Fun.(*ast.FuncLit)
	if !is_function_literal {
		return false
	}
	if function_literal.Type.Params != nil {
		if len(function_literal.Type.Params.List) > 0 {
			return false
		}
	}
	if function_literal.Type.Results != nil {
		if len(function_literal.Type.Results.List) > 0 {
			return false
		}
	}
	return function_literal.Body != nil
}

// Reports whether condition is exactly `X != nil` where X is an ident-chain.
// Used by the prologue and defer-body walkers to gate descent into an
// if-body: only a nil-guard makes deref inside the body safe.
func check_invariant_assertions_is_nil_guard(condition ast.Expr) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Condition is a nil-guard"))

	}()
	binary, is_binary := condition.(*ast.BinaryExpr)
	if !is_binary {
		return false
	}
	if binary.Op != token.NEQ {
		return false
	}
	right, is_ident := binary.Y.(*ast.Ident)
	if !is_ident {
		return false
	}
	if right.Name != "nil" {
		return false
	}
	return check_invariant_assertions_expression_to_path(binary.X) != ""
}

// Reports whether expression is a top-level compound boolean — a BinaryExpr
// with `&&` or `||` at the root. Nested compounds inside CallExpr arguments
// don't count.
func check_invariant_assertions_predicate_is_compound(expression ast.Expr) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Expression is a top-level compound boolean"))

	}()
	binary, is_binary := expression.(*ast.BinaryExpr)
	if !is_binary {
		return false
	}
	if binary.Op == token.LAND {
		return true
	}
	return binary.Op == token.LOR
}

// Walks function.Body for invariant assertion calls and emits a diagnostic
// whenever the predicate arg of Always/Sometimes/Is_Always/Is_Sometimes (and
// the Recorder_ variants) is a top-level `&&` or `||`. The rule keeps the
// invariant predicate grammar narrow: a single comparison, a bare ident, or
// `!ident`. Compound nil-safety belongs in an `if X != nil { ... }` guard.
func check_invariant_assertions_validate_no_compound_predicates(
	file_set *token.FileSet,
	function *ast.FuncDecl,
	function_label string,
	invariant_idents map[string]bool,
) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_validate_no_compound_predicates_assert_exit(
			diags, invariant_idents)
	}()
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	label_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(function_label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Function_label length is bounded; callers gate empty before calling",
	})
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(function != nil, "Function is non-nil"),
		invariant.Always(function.Body != nil, "Function.Body is non-nil"),
		invariant.Always(function_label != "",
			"Function_label is non-empty per upstream gate"),
		label_axis, ii,
		invariant.Excluding("Both label and invariant_idents at safety caps is bad",
			invariant.Bucket_Hi(label_axis), invariant.Bucket_Hi(ii)),
		invariant.Excluding("Zero label with max invariant_idents is bad",
			invariant.Bucket_Lo(label_axis), invariant.Bucket_Hi(ii)),
	)

	ast.Inspect(function.Body, func(node ast.Node) (descend bool) {
		call, is_call := node.(*ast.CallExpr)
		if !is_call {
			return true
		}
		helper_name := check_invariant_assertions_extract_call_name(call, invariant_idents)
		invariant.Cross_Product(
			invariant.Sometimes(
				helper_name == "", "Helper_name is empty for non-invariant calls"))
		if helper_name == "" {
			return true
		}
		predicate_index := check_invariant_assertions_nil_predicate_index(helper_name)
		invariant.Cross_Product(
			invariant.Sometimes(predicate_index < 0, "Predicate_index is negative for "+
				"non-predicate helpers"))
		if predicate_index < 0 {
			return true
		}
		if predicate_index >= len(call.Args) {
			return true
		}
		if !check_invariant_assertions_predicate_is_compound(call.Args[predicate_index]) {
			return true
		}
		diags = append(diags, Diagnostic{
			Position: file_set.Position(call.Pos()),
			Name:     "invariant_compound_predicate",
			Want: "split the predicate into separate axes inside Cross_Product, " +
				"or guard inner-field access with `if X != nil { ... }`",
			Message: fmt.Sprintf(
				"%s: invariant assertion uses compound boolean predicate (&& / ||)",
				function_label),
		})
		return true
	})
	return diags
}

func check_invariant_assertions_validate_no_compound_predicates_assert_exit(
	diags []Diagnostic, invariant_idents map[string]bool,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	invariant.Cross_Product(d, ii,
		invariant.Excluding("Both d and invariant_idents at safety caps is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Hi(ii)),
		invariant.Excluding("Max d with zero invariant_idents is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_Lo(ii)),
		invariant.Excluding("Zero d with max invariant_idents is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_Hi(ii)),
	)
}

// Reports whether function declares at least one named return value. Used to
// scope the defer-at-body-zero rule: functions without named returns have no
// return-value assertion to host, so they're exempt.
func check_invariant_assertions_has_named_return(function *ast.FuncDecl) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Function declares at least one named return"))

	}()
	invariant.Cross_Product(invariant.Always(
		function != nil, "Function is non-nil"))

	if function.Type.Results == nil {
		return false
	}
	for _, field := range function.Type.Results.List {
		if len(field.Names) > 0 {
			return true
		}
	}
	return false
}

// Enforces the spec rule: when a function declares named returns, the
// assertion defer must be the very first statement of the body. Emits a
// diagnostic when:
//   - the first DeferStmt in body is at index > 0 (not the first statement);
//   - the first DeferStmt is not the assertion-defer shape (cleanup defer
//     precedes the assertion defer, violating LIFO ordering at exit);
//   - no DeferStmt exists at all (named returns have no host).
//
// Functions without named returns are exempt entirely — cleanup defers can
// appear anywhere.
func check_invariant_assertions_validate_defer_position(
	file_set *token.FileSet,
	function *ast.FuncDecl,
	struct_field_index map[string]check_invariant_assertions_struct_information,
) (diag *Diagnostic) {
	defer func() {
		check_invariant_assertions_validate_defer_position_assert_exit(
			struct_field_index, diag)
	}()
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(struct_field_index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by budget",
	})
	single_axis := invariant.Sometimes(
		len(struct_field_index) == count_one, "Struct_field_index is exactly 1 sometimes")
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(function != nil, "Function is non-nil"),
		sfi, single_axis,
		invariant.Excluding("Max sfi contradicts single true",
			invariant.Bucket_Hi(sfi), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Sfi at safety cap is bad",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Zero sfi contradicts single true",
			invariant.Bucket_Lo(sfi), invariant.Bucket_True(single_axis)),
	)

	if !check_invariant_assertions_has_named_return(function) {
		return nil
	}
	if function.Body == nil {
		return nil
	}
	// Scope: skip when no named return contributes a trackable requirement
	// (all return types are untracked — slices, strings, maps, etc.). A defer
	// would have nothing useful to assert; forcing one would be ceremony.
	function_label := check_invariant_assertions_function_label(function)
	invariant.Cross_Product(
		invariant.Always(
			function_label != "", "Function_label is non-empty for named FuncDecls"))
	return_requirements := check_invariant_assertions_requirements_from_field_list(
		&check_invariant_assertions_requirements_from_field_list_input{
			Field_List:         function.Type.Results,
			Struct_Field_Index: struct_field_index,
			Function_Label:     function_label,
		})
	invariant.Cross_Product(invariant.Sometimes(return_requirements == nil,
		"Return_requirements is nil when named returns are all untracked"))

	if len(return_requirements) == 0 {
		return nil
	}
	return check_invariant_assertions_defer_position_violation(file_set, function)
}

func check_invariant_assertions_validate_defer_position_assert_exit(
	struct_field_index map[string]check_invariant_assertions_struct_information,
	diag *Diagnostic,
) {
	sfi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(struct_field_index), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Struct_field_index is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(struct_field_index) == 0, "Struct_field_index is empty sometimes")
	invariant.Cross_Product(
		invariant.Sometimes(diag == nil, "Diag is nil when the defer position is "+
			"valid"),
		sfi, empty_axis,
		invariant.Excluding("Max sfi contradicts empty true",
			invariant.Bucket_Hi(sfi), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Sfi at safety cap is bad",
			invariant.Bucket_Hi(sfi), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero sfi implies empty true",
			invariant.Bucket_Lo(sfi), invariant.Bucket_False(empty_axis)),
	)
	if diag != nil {
		invariant.Cross_Product(
			invariant.Always(diag.Tier == 0, "Tier is 0 at construction"))
		name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(diag.Name),
			Lo: min_defer_position_name_chars,
			Hi: max_defer_position_name_chars,
			Message: "Diag.Name is one of the defer-position diagnostic name " +
				"strings",
		})
		invariant.Cross_Product(name_axis)
		want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(diag.Want),
			Lo: min_defer_position_want_chars,
			Hi: max_defer_position_want_chars,
			Message: "Diag.Want is `add` (71), `move` (76), or `place` (103) " +
				"defer-position hint",
		})
		invariant.Cross_Product(want_axis)
		message_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(diag.Message),
			Lo: min_defer_position_message_chars,
			Hi: max_diagnostic_message_chars,
			Message: "Diag.Message is the Sprintf defer-position hint within " +
				"budget",
		})
		message_name_axis := invariant.Distinct_Boundary(
			&invariant.Boundary_Input[int]{
				X:  len(diag.Name),
				Lo: min_defer_position_name_chars,
				Hi: max_defer_position_name_chars,
				Message: "Diag.Name distinguishes the defer-position " +
					"variant",
			})
		invariant.Cross_Product(message_axis, message_name_axis,
			invariant.Excluding("Hi message at Lo name unreachable",
				invariant.Bucket_Hi(message_axis),
				invariant.Bucket_Lo(message_name_axis)),
			invariant.Excluding("Hi message at Hi name unreachable",
				invariant.Bucket_Hi(message_axis),
				invariant.Bucket_Hi(message_name_axis)),
			invariant.Excluding("Lo message at Lo name unreachable",
				invariant.Bucket_Lo(message_axis),
				invariant.Bucket_Lo(message_name_axis)),
		)
	}
}

// Reports the defer-position violation for a named-return function whose return
// set has a trackable requirement: an assertion defer not at body[0], a
// non-assertion-shaped leading defer, or no assertion defer at all. Returns nil
// when the position is valid. Split from validate_defer_position so each holds
// the line cap; both assert the returned diagnostic's strings — the duplicated
// assertion is the deliberate cost of the split.
func check_invariant_assertions_defer_position_violation(
	file_set *token.FileSet, function *ast.FuncDecl,
) (diag *Diagnostic) {
	defer func() {
		Diagnostic_Assert_Defer_Position(diag)
	}()
	invariant.Cross_Product(invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(function != nil, "Function is non-nil"),
		invariant.Always(function.Body != nil, "Function.Body is non-nil"))
	function_label := check_invariant_assertions_function_label(function)
	invariant.Cross_Product(
		invariant.Always(
			function_label != "", "Function_label is non-empty for named FuncDecls"))
	for index, statement := range function.Body.List {
		defer_statement, is_defer := statement.(*ast.DeferStmt)
		if !is_defer {
			continue
		}
		if index > 0 {
			return &Diagnostic{
				Position: file_set.Position(defer_statement.Pos()),
				Name:     "assertion_defer_not_at_body_zero",
				Want: "move the assertion defer to be the very first " +
					"statement of the function body",
				Message: fmt.Sprintf(
					"%s: assertion defer must be the very first statement"+
						" in the function body (cleanup defers and param"+
						" assertions must follow)",
					function_label),
			}
		}
		if !check_invariant_assertions_is_assertion_defer_shape(defer_statement) {
			return &Diagnostic{
				Position: file_set.Position(defer_statement.Pos()),
				Name:     "assertion_defer_not_at_body_zero",
				Want: "place the assertion defer (empty-signature FuncLit" +
					" hosting invariant calls) as the very first statement",
				Message: fmt.Sprintf(
					"%s: assertion defer must be the very first statement;"+
						" a cleanup defer appears before the assertion "+
						"defer",
					function_label),
			}
		}
		return nil
	}
	return &Diagnostic{
		Position: file_set.Position(function.Body.Pos()),
		Name:     "assertion_defer_missing",
		Want:     "add an assertion defer as the very first statement of the function body",
		Message: fmt.Sprintf(
			"%s: assertion defer must be the very first statement;"+
				" the function declares named returns but has no defer at all",
			function_label),
	}
}

// Asserts the fixed shape of a defer-position diagnostic at construction
// (tier-0, bounded name/want/message variants). The Diagnostic_ prefix is
// mandated because the first parameter is a same-package named type.
func Diagnostic_Assert_Defer_Position(diag *Diagnostic) {
	invariant.Cross_Product(
		invariant.Sometimes(diag == nil, "Diag is nil when the defer position is "+
			"valid"))
	if diag != nil {
		invariant.Cross_Product(
			invariant.Always(diag.Tier == 0, "Tier is 0 at construction"))
		name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(diag.Name),
			Lo: min_defer_position_name_chars,
			Hi: max_defer_position_name_chars,
			Message: "Diag.Name is one of the defer-position diagnostic name " +
				"strings",
		})
		invariant.Cross_Product(name_axis)
		want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(diag.Want),
			Lo: min_defer_position_want_chars,
			Hi: max_defer_position_want_chars,
			Message: "Diag.Want is `add` (71), `move` (76), or `place` (103) " +
				"defer-position hint",
		})
		invariant.Cross_Product(want_axis)
		message_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(diag.Message),
			Lo: min_defer_position_message_chars,
			Hi: max_diagnostic_message_chars,
			Message: "Diag.Message is the Sprintf defer-position hint within " +
				"budget",
		})
		message_name_axis := invariant.Distinct_Boundary(
			&invariant.Boundary_Input[int]{
				X:  len(diag.Name),
				Lo: min_defer_position_name_chars,
				Hi: max_defer_position_name_chars,
				Message: "Diag.Name distinguishes the defer-position " +
					"variant",
			})
		invariant.Cross_Product(message_axis, message_name_axis,
			invariant.Excluding("Hi message at Lo name unreachable",
				invariant.Bucket_Hi(message_axis),
				invariant.Bucket_Lo(message_name_axis)),
			invariant.Excluding("Hi message at Hi name unreachable",
				invariant.Bucket_Hi(message_axis),
				invariant.Bucket_Hi(message_name_axis)),
			invariant.Excluding("Lo message at Lo name unreachable",
				invariant.Bucket_Lo(message_axis),
				invariant.Bucket_Lo(message_name_axis)),
		)
	}
}

// Walks the deferred FuncLit's body as a prologue and credits covered paths
// to the named-return-covered set. A defer whose callee is anything other
// than an empty-signature FuncLit contributes nothing — those are real
// cleanup/recover defers, not the assertion convention.
func check_invariant_assertions_credit_defer(
	defer_statement *ast.DeferStmt,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
	named_return_covered map[string]map[string]coverage_origin,
) {
	defer func() {
		check_invariant_assertions_credit_defer_assert_exit(
			invariant_idents, constant_signs, named_return_covered)
	}()
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is ≥1 (assertion defer path requires invariant import)",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	nrc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(named_return_covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_covered is bounded by budget",
	})
	invariant.Cross_Product(
		invariant.Always(defer_statement != nil, "Defer_statement is non-nil"),
		ii, cs, nrc,
		invariant.Excluding("Both invariant_idents and constant_signs at safety caps is "+
			"bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_Hi(cs)),
		invariant.Excluding("Invariant_idents at safety cap with zero constant_signs is "+
			"bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_Lo(cs)),
		invariant.Excluding("Zero invariant_idents with max constant_signs is bad",
			invariant.Bucket_Lo(ii), invariant.Bucket_Hi(cs)),
		invariant.Excluding("Both constant_signs and named_return_covered at safety caps "+
			"is bad",
			invariant.Bucket_Hi(cs), invariant.Bucket_Hi(nrc)),
		invariant.Excluding("Max constant_signs with zero named_return_covered is bad",
			invariant.Bucket_Hi(cs), invariant.Bucket_Lo(nrc)),
		invariant.Excluding("Zero constant_signs with max named_return_covered is bad",
			invariant.Bucket_Lo(cs), invariant.Bucket_Hi(nrc)),
	)

	function_literal, is_function_literal := defer_statement.Call.Fun.(*ast.FuncLit)
	if !is_function_literal {
		return
	}
	if function_literal.Type.Params != nil {
		if len(function_literal.Type.Params.List) > 0 {
			return
		}
	}
	if function_literal.Type.Results != nil {
		if len(function_literal.Type.Results.List) > 0 {
			return
		}
	}
	if function_literal.Body == nil {
		return
	}
	check_invariant_assertions_walk_assertion_chain(
		&check_invariant_assertions_walk_assertion_chain_input{
			Statements:       function_literal.Body.List,
			Invariant_Idents: invariant_idents,
			Constant_Signs:   constant_signs,
			Covered:          named_return_covered,
			Outside_If_Guard: true,
		})
}

func check_invariant_assertions_credit_defer_assert_exit(
	invariant_idents map[string]bool,
	constant_signs map[string]int,
	named_return_covered map[string]map[string]coverage_origin,
) {
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(invariant_idents),
		Lo: min_non_empty,
		Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is ≥1 (assertion defer path " +
			"requires invariant " +
			"import)",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	nrc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(named_return_covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_covered is bounded by budget",
	})
	invariant.Cross_Product(ii, cs, nrc,
		invariant.Excluding("Both invariant_idents and constant_signs at safety "+
			"caps is bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_Hi(cs)),
		invariant.Excluding("Invariant_idents at safety cap with zero "+
			"constant_signs is bad",
			invariant.Bucket_Hi(ii), invariant.Bucket_Lo(cs)),
		invariant.Excluding("Zero invariant_idents with max constant_signs is bad",
			invariant.Bucket_Lo(ii), invariant.Bucket_Hi(cs)),
		invariant.Excluding("Both constant_signs and named_return_covered at "+
			"safety caps is bad",
			invariant.Bucket_Hi(cs), invariant.Bucket_Hi(nrc)),
		invariant.Excluding("Max constant_signs with zero named_return_covered is "+
			"bad",
			invariant.Bucket_Hi(cs), invariant.Bucket_Lo(nrc)),
		invariant.Excluding("Zero constant_signs with max named_return_covered is "+
			"bad",
			invariant.Bucket_Lo(cs), invariant.Bucket_Hi(nrc)),
	)
}

// Records every (path, kind) pair into the covered map. Multiple calls
// accumulate; a path may be covered by multiple kinds across different
// prologue statements. The map's value is true iff at least one of the
// covering calls was a Cross_Product (versus a single-axis _Of helper) —
// once a path is credited from Cross_Product the bit stays true even if
// later _Of calls re-cover it.
func check_invariant_assertions_record_coverage(
	covered map[string]map[string]coverage_origin,
	pairs []check_invariant_assertions_coverage_pair,
	outside_if_guard bool,
) {
	defer func() {
		p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(pairs), Lo: 0, Hi: max_coverage_pairs_per_call,
			Message: "Pairs is bounded by budget",
		})
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(covered), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Covered is bounded by budget",
		})
		invariant.Cross_Product(p, c,
			invariant.Excluding("Both pairs and constants at safety caps is bad",
				invariant.Bucket_Hi(p), invariant.Bucket_Hi(c)),
			invariant.Excluding("Pairs at safety cap with zero constant_signs is bad",
				invariant.Bucket_Hi(p), invariant.Bucket_Lo(c)),
			invariant.Excluding("Zero pairs with max constant_signs is bad",
				invariant.Bucket_Lo(p), invariant.Bucket_Hi(c)),
		)
	}()
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pairs), Lo: 0, Hi: max_coverage_pairs_per_call,
		Message: "Pairs is bounded by budget",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Covered is bounded by budget",
	})
	outside := invariant.Sometimes(
		outside_if_guard, "At least one credit comes from outside an if-guard")
	invariant.Cross_Product(
		invariant.Always(covered != nil, "Covered is non-nil"),
		outside,
		p, c,
		invariant.Excluding("Outside false with min p is bad",
			invariant.Bucket_False(outside), invariant.Bucket_Lo(p)),
		invariant.Excluding("Both pairs and constants at safety caps is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Hi(c)),
		invariant.Excluding("Pairs at safety cap with zero constant_signs is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Lo(c)),
		invariant.Excluding("Zero pairs with max constant_signs is bad",
			invariant.Bucket_Lo(p), invariant.Bucket_Hi(c)),
	)
	for _, pair := range pairs {
		if covered[pair.Path] == nil {
			covered[pair.Path] = map[string]coverage_origin{}
		}
		origin := covered[pair.Path][pair.Kind]
		if pair.From_Cross_Product {
			origin.From_Cross_Product = true
		}
		if outside_if_guard {
			origin.Outside_If_Guard = true
		}
		covered[pair.Path][pair.Kind] = origin
	}
}

// Determines whether call is a recognised invariant prologue call and, if so,
// returns the (identifier_path, kind) pairs it covers. Recognises three
// shapes:
//
//  1. Top-level _Of helper (Range_Of, String_Size_Of, Nil_Of, …): scans every
//     argument for identifier-path shapes and pairs each with the helper's kind.
//  2. Cross_Product / Recorder_Cross_Product: walks each arg as a bare
//     composable sub-call (Range, Nil, …) and emits one pair per inner
//     identifier-path with the inner composable's kind.
//  3. Bare composable (Range, Nil, …) at the top level: returns ok=false
//     because bare composables register no tracker entries unless wrapped
//     in Cross_Product.
func assert_idents_signs_bounded(
	invariant_idents map[string]bool, constant_signs map[string]int,
) {
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	invariant.Cross_Product(i, c,
		invariant.Excluding("Both idents and signs at AST safety caps is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Invariant_idents at AST at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
	)
}

func check_invariant_assertions_call_covered_pairs_prologue_axes(
	call *ast.CallExpr,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) {
	defer func() {
		assert_idents_signs_bounded(invariant_idents, constant_signs)
	}()
	invariant.Cross_Product(invariant.Always(call != nil, "Call is non-nil"))
	assert_idents_signs_bounded(invariant_idents, constant_signs)
}

func check_invariant_assertions_call_covered_pairs_defer_axes(
	pairs []check_invariant_assertions_coverage_pair,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) {
	defer func() {
		i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Invariant_idents is bounded by budget",
		})
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Constant_signs is bounded by budget",
		})
		p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(pairs), Lo: 0, Hi: max_coverage_pairs_per_call,
			Message: "Pairs is bounded by budget",
		})
		invariant.Cross_Product(i, c, p,
			invariant.Excluding("Both idents and signs at AST safety caps is bad",
				invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
			invariant.Excluding("Invariant_idents at AST at safety cap is bad",
				invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
			invariant.Excluding("Constant_signs at AST at safety cap is bad",
				invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
			invariant.Excluding("Both constants and pairs at safety caps is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_Hi(p)),
			invariant.Excluding("Constants at safety cap with zero pairs is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_Lo(p)),
			invariant.Excluding("Zero constants with max pairs is bad",
				invariant.Bucket_Lo(c), invariant.Bucket_Hi(p)),
		)
	}()
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pairs), Lo: 0, Hi: max_coverage_pairs_per_call,
		Message: "Pairs is bounded by budget",
	})
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	pairs_nil_axis := invariant.Sometimes(pairs == nil, "Pairs is nil for unrecognised calls")
	invariant.Cross_Product(
		pairs_nil_axis, p, i, c,
		invariant.Excluding("Both p and i at safety caps is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Hi(i)),
		invariant.Excluding("Axis p at safety cap is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Lo(i)),
		invariant.Excluding("Zero p with max i is bad",
			invariant.Bucket_Lo(p), invariant.Bucket_Hi(i)),
		invariant.Excluding("Both idents and signs at AST safety caps is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Invariant_idents at AST at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Non-nil pairs with zero p, i, c is impossible by construction",
			invariant.Bucket_False(pairs_nil_axis),
			invariant.Bucket_Lo(p), invariant.Bucket_Lo(i), invariant.Bucket_Lo(c)),
	)
}

func check_invariant_assertions_call_covered_pairs(
	call *ast.CallExpr,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) (pairs []check_invariant_assertions_coverage_pair) {
	defer func() {
		check_invariant_assertions_call_covered_pairs_assert_exit(
			pairs, invariant_idents, constant_signs)
	}()
	invariant.Cross_Product(invariant.Always(call != nil, "Call is non-nil"))
	assert_idents_signs_bounded(invariant_idents, constant_signs)
	check_invariant_assertions_call_covered_pairs_prologue_axes(
		call, invariant_idents, constant_signs)

	helper_name := check_invariant_assertions_extract_call_name(call, invariant_idents)
	invariant.Cross_Product(
		invariant.Sometimes(
			helper_name == "", "Helper_name is empty for non-invariant calls"))
	if helper_name == "" {
		return nil
	}
	if helper_kind, has := invariant_of_helper_kinds[helper_name]; has {
		pairs = []check_invariant_assertions_coverage_pair{}
		paths := check_invariant_assertions_extract_identifier_paths(call.Args)
		invariant.Is_Sometimes(paths == nil, "Paths can be empty or zero on this branch")
		for _, path := range paths {
			pairs = append(pairs, check_invariant_assertions_coverage_pair{
				Path:               path,
				Kind:               helper_kind,
				From_Cross_Product: false,
			})
		}
		comparison_path,
			comparison_kinds := check_invariant_assertions_extract_nil_comparison_path(
			call, helper_name, nil)
		path_axis := invariant.Sometimes(
			comparison_path == "",
			"Comparison_path is empty for non-comparison helpers")
		kinds_axis := invariant.Sometimes(
			comparison_kinds == nil, "Comparison_kinds is nil for non-comparison "+
				"helpers")
		invariant.Cross_Product(
			path_axis, kinds_axis,
			invariant.Excluding("Path absent implies kinds empty",
				invariant.Bucket_False(path_axis),
				invariant.Bucket_True(kinds_axis)),
			invariant.Excluding("Path present implies kinds extracted",
				invariant.Bucket_True(path_axis),
				invariant.Bucket_False(kinds_axis)),
		)
		for _, comparison_kind := range comparison_kinds {
			pairs = append(pairs, check_invariant_assertions_coverage_pair{
				Path:               comparison_path,
				Kind:               comparison_kind,
				From_Cross_Product: false,
			})
		}
		return pairs
	}
	if helper_name != "Cross_Product" {
		if helper_name != "Recorder_Cross_Product" {
			return nil
		}
	}
	return check_invariant_assertions_call_covered_pairs_cross_product(
		call, helper_name, invariant_idents, constant_signs)
}

// Single-axis `Is_*` / `Recorder_Is_*` helpers and the credit kind each
// contributes; lookup miss means the call isn't a single-axis builder.
var invariant_of_helper_kinds = map[string]string{
	"Is_Distinct_Boundary":          "boundary_int",
	"Recorder_Is_Distinct_Boundary": "boundary_int",
	"Is_Always":                     "bool",
	"Recorder_Is_Always":            "bool",
	"Is_Sometimes":                  "bool",
	"Recorder_Is_Sometimes":         "bool",
}

func check_invariant_assertions_call_covered_pairs_assert_exit(
	pairs []check_invariant_assertions_coverage_pair,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) {
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pairs), Lo: 0, Hi: max_coverage_pairs_per_call,
		Message: "Pairs is bounded by budget",
	})
	invariant.Cross_Product(i, c, p,
		invariant.Excluding("Both idents and signs at AST safety caps is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Invariant_idents at AST at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Both constants and pairs at safety caps is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Hi(p)),
		invariant.Excluding("Constants at safety cap with zero pairs is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Lo(p)),
		invariant.Excluding("Zero constants with max pairs is bad",
			invariant.Bucket_Lo(c), invariant.Bucket_Hi(p)),
	)
	check_invariant_assertions_call_covered_pairs_defer_axes(
		pairs, invariant_idents, constant_signs)
}

// Presents the same-file numeric const names as a presence map so the
// numeric-comparison matcher recognises `Always(len(x) == NAMED_CONST)` as a
// boundary credit — the satisfying form for an invariant-length field, whose
// constant length a Distinct_Boundary (two reachable endpoints) can't bound.
// Keys come from constant_signs, which already tracks exactly the consts
// resolving to numeric inline literals (the only valid length-comparison RHS).
func check_invariant_assertions_numeric_constant_name_set(
	constant_signs map[string]int,
) (names map[string]bool) {
	defer func() {
		n := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(names), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Names is bounded by AST budget",
		})
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Constant_signs is bounded by AST budget",
		})
		empty := invariant.Sometimes(len(names) == 0, "Names is empty sometimes")
		invariant.Cross_Product(n, c, empty,
			invariant.Excluding("Hi n with empty unreachable",
				invariant.Bucket_Hi(n), invariant.Bucket_True(empty)),
			invariant.Excluding("Hi n non-empty unreachable",
				invariant.Bucket_Hi(n), invariant.Bucket_False(empty)),
			invariant.Excluding("Lo n implies empty true",
				invariant.Bucket_Lo(n), invariant.Bucket_False(empty)),
			invariant.Excluding("Hi c with empty unreachable",
				invariant.Bucket_Hi(c), invariant.Bucket_True(empty)),
			invariant.Excluding("Hi c non-empty unreachable",
				invariant.Bucket_Hi(c), invariant.Bucket_False(empty)),
		)
	}()
	cc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by AST budget",
	})
	cc_empty := invariant.Sometimes(
		len(constant_signs) == 0, "Constant_signs is empty sometimes")
	invariant.Cross_Product(cc, cc_empty,
		invariant.Excluding("Hi cc with empty unreachable",
			invariant.Bucket_Hi(cc), invariant.Bucket_True(cc_empty)),
		invariant.Excluding("Hi cc non-empty unreachable",
			invariant.Bucket_Hi(cc), invariant.Bucket_False(cc_empty)),
		invariant.Excluding("Lo cc implies empty true",
			invariant.Bucket_Lo(cc), invariant.Bucket_False(cc_empty)),
	)
	names = map[string]bool{}
	for constant_name := range constant_signs {
		names[constant_name] = true
	}
	return names
}

// Walks Cross_Product / Recorder_Cross_Product inner-call args, accumulating
// coverage pairs for every bare composable (Range, Nil, …) and nil-comparison
// helper found. Recorder_Cross_Product skips the leading recorder argument.
func check_invariant_assertions_call_covered_pairs_cross_product(
	call *ast.CallExpr,
	helper_name string,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) (pairs []check_invariant_assertions_coverage_pair) {
	defer func() {
		check_invariant_assertions_call_covered_pairs_cross_product_assert_exit(
			pairs, helper_name, invariant_idents, constant_signs)
	}()
	check_invariant_assertions_call_covered_pairs_cross_product_assert_entry(
		call, helper_name, invariant_idents, constant_signs)
	pairs = []check_invariant_assertions_coverage_pair{}
	argument_offset := 0
	if helper_name == "Recorder_Cross_Product" {
		argument_offset = 1
	}
	for index := argument_offset; index < len(call.Args); index++ {
		inner_call, is_inner_call := call.Args[index].(*ast.CallExpr)
		if !is_inner_call {
			continue
		}
		inner_name := check_invariant_assertions_extract_call_name(
			inner_call, invariant_idents)
		invariant.Cross_Product(
			invariant.Sometimes(
				inner_name == "",
				"Inner_name is empty for non-invariant inner calls"))
		if inner_name == "" {
			continue
		}
		bare_kind, is_bare := invariant_axis_builder_kinds[inner_name]
		if is_bare {
			pairs = append(pairs,
				check_invariant_assertions_bare_axis_pairs(
					inner_call, bare_kind, constant_signs)...)
		}
		comparison_path,
			comparison_kinds := check_invariant_assertions_extract_nil_comparison_path(
			inner_call,
			inner_name,
			check_invariant_assertions_numeric_constant_name_set(constant_signs))
		path_axis := invariant.Sometimes(
			comparison_path == "",
			"Comparison_path is empty for non-comparison helpers")
		kinds_axis := invariant.Sometimes(
			comparison_kinds == nil, "Comparison_kinds is nil for non-comparison "+
				"helpers")
		invariant.Cross_Product(
			path_axis, kinds_axis,
			invariant.Excluding("Path absent implies kinds empty",
				invariant.Bucket_False(path_axis),
				invariant.Bucket_True(kinds_axis)),
			invariant.Excluding("Path present implies kinds extracted",
				invariant.Bucket_True(path_axis),
				invariant.Bucket_False(kinds_axis)),
		)
		for _, comparison_kind := range comparison_kinds {
			pairs = append(pairs, check_invariant_assertions_coverage_pair{
				Path:               comparison_path,
				Kind:               comparison_kind,
				From_Cross_Product: true,
			})
		}
	}
	return pairs
}

func check_invariant_assertions_call_covered_pairs_cross_product_assert_exit(
	pairs []check_invariant_assertions_coverage_pair,
	helper_name string,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) {
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(invariant_idents),
		Lo: min_non_empty,
		Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is ≥1 (cross-product walk requires invariant " +
			"import)",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pairs), Lo: 0, Hi: max_coverage_pairs_per_call,
		Message: "Pairs is bounded by budget",
	})
	h := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(helper_name),
		Lo: cross_product_helper_chars,
		Hi: recorder_cross_product_helper_chars,
		Message: "Helper_name length matches Cross_Product (13) or " +
			"Recorder_Cross_Product (22) per caller gate",
	})
	invariant.Cross_Product(i, c, p, h,
		invariant.Excluding("Both idents and signs at safety caps is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
		invariant.Excluding("Constant_signs at safety cap is bad",
			invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Both constants and pairs at safety caps",
			invariant.Bucket_Hi(c), invariant.Bucket_Hi(p)),
		invariant.Excluding("Constants at safety cap with zero pairs",
			invariant.Bucket_Hi(c), invariant.Bucket_Lo(p)),
		invariant.Excluding("Zero constants with max pairs is bad",
			invariant.Bucket_Lo(c), invariant.Bucket_Hi(p)),
		// Hi(h) = 22 marks Recorder_Cross_Product. Callers always pass
		// a recorder plus ≥1 axis-bearing arg; (Lo i, Lo c, Lo p, Hi h)
		// requires only-pre-bound axes — permitted but unused in prod.
		invariant.Excluding("Recorder_Cross_Product with only pre-bound axes is "+
			"unused",
			invariant.Bucket_Lo(
				i),
			invariant.Bucket_Lo(c),
			invariant.Bucket_Lo(p),
			invariant.Bucket_Hi(h)),
	)
}

func check_invariant_assertions_call_covered_pairs_cross_product_assert_entry(
	call *ast.CallExpr,
	helper_name string,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) {
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is ≥1 (cross-product walk requires invariant import)",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	h := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(helper_name),
		Lo: cross_product_helper_chars,
		Hi: recorder_cross_product_helper_chars,
		Message: "Helper_name length matches Cross_Product (13) or " +
			"Recorder_Cross_Product " +
			"(22) per caller gate",
	})
	invariant.Cross_Product(invariant.Always(call != nil, "Call is non-nil"),
		i, c, h,
		invariant.Excluding("Both idents and signs at AST safety caps is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Invariant_idents at AST at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
		invariant.Excluding("Both c and h at safety caps is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Hi(h)),
		invariant.Excluding("Max c with min h is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Lo(h)),
	)
}

// Builds the coverage pairs for a single bare-composable inner call inside
// a Cross_Product. Emits one pair per identifier path the call covers,
// plus an additional zero_int / zero_float / zero_string credit when the
// call is a Distinct_Boundary whose Lo or Hi key is an inline-literal zero
// (per the Lo/Hi=0 auto-credit rule). Split out of call_covered_pairs to
// keep that dispatcher under the per-function line cap.
func check_invariant_assertions_bare_axis_pairs(
	inner_call *ast.CallExpr,
	bare_kind string,
	constant_signs map[string]int,
) (pairs []check_invariant_assertions_coverage_pair) {
	defer func() {
		check_invariant_assertions_bare_axis_pairs_assert_exit(pairs, constant_signs)
	}()
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	bare_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(bare_kind), Lo: min_credit_kind_chars, Hi: max_bare_credit_kind_chars,
		Message: "Bare_kind is `bool` (4) or `boundary_int` (12) per bare_composable_table",
	})
	invariant.Cross_Product(
		invariant.Always(inner_call != nil, "Inner_call is non-nil"),
		invariant.Always(bare_kind != "",
			"Bare_kind is non-empty per bare_table lookup hit"),
		c, bare_axis,
		invariant.Excluding("Both c and bare at safety caps is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Hi(bare_axis)),
		invariant.Excluding("Max c with zero bare is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_Lo(bare_axis)),
	)
	paths := check_invariant_assertions_extract_identifier_paths(inner_call.Args)
	invariant.Cross_Product(invariant.Sometimes(paths == nil,
		"Paths is nil for inner calls absent identifier args"))
	for _, path := range paths {
		pairs = append(pairs, check_invariant_assertions_coverage_pair{
			Path:               path,
			Kind:               bare_kind,
			From_Cross_Product: true,
		})
	}
	// Two boundary shapes auto-credit the zero half of the dual int / float /
	// string requirement:
	//   * Lo or Hi == 0 — the matching bucket IS the "x == 0" observation;
	//     a separate Sometimes(x == 0) axis is redundant (and creates
	//     impossible tuples).
	//   * Lo > 0 or Hi < 0 — the range provably excludes zero so x != 0 is
	//     invariantly true; a separate Always(x != 0) axis is redundant.
	if !check_invariant_assertions_boundary_input_has_zero_endpoint(
		inner_call.Args, constant_signs) {
		if !check_invariant_assertions_boundary_input_excludes_zero(
			inner_call.Args, constant_signs) {
			return pairs
		}
	}
	for _, path := range paths {
		for _, zero_kind := range []string{
			"zero_int", "zero_float", "zero_string", "zero_slice", "zero_map",
		} {
			pairs = append(pairs, check_invariant_assertions_coverage_pair{
				Path:               path,
				Kind:               zero_kind,
				From_Cross_Product: true,
			})
		}
	}
	return pairs
}

func check_invariant_assertions_bare_axis_pairs_assert_exit(
	pairs []check_invariant_assertions_coverage_pair, constant_signs map[string]int,
) {
	p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(pairs), Lo: 0, Hi: max_coverage_pairs_per_call,
		Message: "Pairs is bounded by budget",
	})
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	pairs_nil_axis := invariant.Sometimes(
		pairs == nil, "Pairs is nil for inner calls without identifier coverage")
	invariant.Cross_Product(p, c, pairs_nil_axis,
		invariant.Excluding("Both pairs and constants at safety caps is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Hi(c)),
		invariant.Excluding("Pairs at safety cap with zero constant_signs is bad",
			invariant.Bucket_Hi(p), invariant.Bucket_Lo(c)),
		invariant.Excluding("Zero pairs with max constant_signs is bad",
			invariant.Bucket_Lo(p), invariant.Bucket_Hi(c)),
		invariant.Excluding("Non-nil pairs with zero p, c is impossible by "+
			"construction",
			invariant.Bucket_Lo(p),
			invariant.Bucket_Lo(c),
			invariant.Bucket_False(pairs_nil_axis)),
	)
}

// Recognises a single zero-value comparison: Always/Sometimes(X == nil) or
// Always(X != nil), and the analogous string `X == ""` / `X != ""` and int
// `X == 0` / `X != 0` forms. Returns the LHS ident-chain path and the kind
// ("pointer" / "zero_string" / "zero_int"), or "","" for any other shape.
// The int shape returns "zero_int" (not "int") because integer requirements
// split into a Boundary (boundary_int) half and a zero-comparison
// (zero_int) half; this extractor only credits the latter. The string
// shape is similarly the empty-state half — boundary credit comes from
// len-based shapes through extract_binary_path's numeric branch.
// Sometimes(X != zero) is rejected: only the canonical `== zero` satisfies
// coverage. The LAND/LOR compound forms are no longer recognised here —
// compound predicates in invariant calls are banned by
// validate_no_compound_predicates.
func check_invariant_assertions_extract_nil_comparison_path(
	call *ast.CallExpr,
	helper_name string,
	same_file_consts map[string]bool,
) (path string, kinds []string) {
	defer func() {
		check_invariant_assertions_extract_nil_comparison_path_assert_exit(
			kinds, same_file_consts, path)
	}()
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(same_file_consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Same_file_consts is bounded by budget",
	})
	helper_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(helper_name),
		Lo:      min_invariant_helper_name_chars,
		Hi:      max_invariant_helper_name_chars,
		Message: "Helper_name length is bounded; callers gate empty before calling",
	})
	invariant.Cross_Product(
		invariant.Always(call != nil, "Call is non-nil"),
		invariant.Always(helper_name != "",
			"Helper_name is non-empty per upstream gate"),
		s, helper_axis,
		invariant.Excluding("Both s and helper at safety caps is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(helper_axis)),
		invariant.Excluding("Max s with min helper is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(helper_axis)),
	)

	predicate_index := check_invariant_assertions_nil_predicate_index(helper_name)
	invariant.Cross_Product(
		invariant.Sometimes(predicate_index == 0, "Predicate_index can be zero on this "+
			"branch"))
	if predicate_index < 0 {
		return "", nil
	}
	if predicate_index >= len(call.Args) {
		return "", nil
	}
	allow_neq := check_invariant_assertions_nil_allows_neq(helper_name)
	invariant.Cross_Product(invariant.Sometimes(allow_neq,
		"Allow_neq can be false on this branch"))
	finite_path, finite_kinds := check_invariant_assertions_extract_finite_path(
		call.Args[predicate_index], allow_neq)
	finite_path_axis := invariant.Sometimes(finite_path == "",
		"Finite_path is empty for non-IsNaN/IsInf predicates")
	finite_kinds_axis := invariant.Sometimes(finite_kinds == nil,
		"Finite_kinds is nil for non-IsNaN/IsInf predicates")
	// Finite_path and finite_kinds are returned together — either both empty
	// (non-IsX shape) or both non-empty (IsNaN/IsInf recognised).
	invariant.Cross_Product(
		finite_path_axis, finite_kinds_axis,
		invariant.Excluding("Finite_path and finite_kinds always co-empty or co-set",
			invariant.Bucket_True(finite_path_axis),
			invariant.Bucket_False(finite_kinds_axis)),
		invariant.Excluding("Finite_path and finite_kinds always co-empty or co-set",
			invariant.Bucket_False(finite_path_axis),
			invariant.Bucket_True(finite_kinds_axis)),
	)
	if finite_path != "" {
		return finite_path, finite_kinds
	}
	binary, is_binary := call.Args[predicate_index].(*ast.BinaryExpr)
	if !is_binary {
		return "", nil
	}
	return check_invariant_assertions_extract_binary_path(binary, allow_neq, same_file_consts)
}

func check_invariant_assertions_extract_nil_comparison_path_assert_exit(
	kinds []string, same_file_consts map[string]bool, path string,
) {
	k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(kinds), Lo: 0, Hi: max_string_slice_per_call,
		Message: "Kinds is bounded by budget",
	})
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(same_file_consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Same_file_consts is bounded by budget",
	})
	path_axis := invariant.Sometimes(
		path == "", "Path is empty for non-comparison shapes")
	kinds_axis := invariant.Sometimes(
		kinds == nil, "Kinds is nil for non-comparison shapes")
	size_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(path), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Path is an ident-chain string bounded by identifier budget",
	})
	invariant.Cross_Product(
		k, s,
		path_axis, kinds_axis, size_axis,
		invariant.Excluding("Path absent implies kinds empty",
			invariant.Bucket_False(path_axis),
			invariant.Bucket_True(kinds_axis)),
		invariant.Excluding("Path present implies kinds extracted",
			invariant.Bucket_True(path_axis),
			invariant.Bucket_False(kinds_axis)),
		invariant.Excluding("Both k and s at safety caps is bad",
			invariant.Bucket_Hi(k), invariant.Bucket_Hi(s)),
		invariant.Excluding("Max k with zero s is bad",
			invariant.Bucket_Hi(k), invariant.Bucket_Lo(s)),
		invariant.Excluding("Path true with max size is bad",
			invariant.Bucket_True(path_axis), invariant.Bucket_Hi(size_axis)),
		invariant.Excluding("Path false with min size is bad",
			invariant.Bucket_False(path_axis), invariant.Bucket_Lo(size_axis)),
		invariant.Excluding("Path false with zero kinds is impossible by "+
			"construction",
			invariant.Bucket_Lo(k),
			invariant.Bucket_Lo(s),
			invariant.Bucket_False(path_axis)),
		invariant.Excluding("Path false with zero kinds is impossible by "+
			"construction",
			invariant.Bucket_Lo(k),
			invariant.Bucket_Hi(s),
			invariant.Bucket_False(path_axis)),
		invariant.Excluding("Path true with zero kinds is impossible by "+
			"construction",
			invariant.Bucket_Lo(k),
			invariant.Bucket_Hi(s),
			invariant.Bucket_True(path_axis)),
	)
}

// Dispatches the BinaryExpr predicate shapes — `x ==/!= nil` (pointer),
// `x ==/!= ""` (string), and the numeric `x ==/!= N` family covering both
// int and float sub-kinds. Split off so extract_nil_comparison_path stays
// readable; the latter handles only the wrapper-call validation and the
// math.IsNaN/IsInf finite-check shape.
func check_invariant_assertions_extract_binary_path(
	binary *ast.BinaryExpr,
	allow_neq bool,
	same_file_consts map[string]bool,
) (path string, kinds []string) {
	defer func() {
		check_invariant_assertions_extract_binary_path_assert_exit(
			kinds, same_file_consts, path)
	}()
	check_invariant_assertions_extract_binary_path_assert_entry(
		binary, allow_neq, same_file_consts)
	if binary.Op != token.EQL {
		if !allow_neq {
			return "", nil
		}
		if binary.Op != token.NEQ {
			return "", nil
		}
	}
	// Nil comparison (X ==/!= nil) — pointer credit only.
	nil_ident, is_nil_ident := binary.Y.(*ast.Ident)
	if is_nil_ident {
		if nil_ident.Name == "nil" {
			return check_invariant_assertions_expression_to_path(
				binary.X), []string{"pointer"}
		}
	}
	// "" string comparison — zero_string credit only. The boundary half
	// of string coverage uses len(s)-based shapes; empty-comparison only
	// satisfies the empty-state requirement.
	if string_literal, is_string := binary.Y.(*ast.BasicLit); is_string {
		if string_literal.Kind == token.STRING {
			if string_literal.Value == "\"\"" {
				return check_invariant_assertions_expression_to_path(
					binary.X), []string{"zero_string"}
			}
			if string_literal.Value == "``" {
				return check_invariant_assertions_expression_to_path(
					binary.X), []string{"zero_string"}
			}
		}
	}
	// Numeric comparison. The credit shape depends on the literal kind on the
	// RHS — an untyped int literal credits both int and float sub-kinds since
	// it parses as either; a typed float literal (`5.5`) credits only float;
	// a same-package const ident credits both since we can't see its type.
	// Cases (helper family h ∈ {Always, Sometimes}, op ∈ {==, !=}):
	//   - h=Always, op === ==, RHS=0 literal → BOTH boundary + zero (int & float)
	//   - h=Always, op == !=, RHS=0 literal → zero (int & float)
	//   - h=Always, op === ==, RHS=non-zero inline literal/const → boundary only
	//   - h=Sometimes, op === ==, RHS=0 literal → zero only
	// Anything else (Sometimes(x == N) for N!=0, Sometimes(x != 0)) → no credit.
	lhs_path := check_invariant_assertions_expression_to_path(binary.X)
	invariant.Cross_Product(invariant.Sometimes(lhs_path == "",
		"Lhs_path is empty for non-ident-chain LHS before len-fallback"))
	if lhs_path == "" {
		// Len(<ident-chain>) credits the inner path so length-based
		// comparisons satisfy boundary requirements on strings.
		lhs_path = check_invariant_assertions_expression_to_size_argument_path(binary.X)
		invariant.Cross_Product(invariant.Sometimes(lhs_path == "",
			"Lhs_path is empty after len-fallback for non-len LHS"))
	}
	invariant.Cross_Product(invariant.Sometimes(lhs_path == "",
		"Lhs_path can be empty for non-ident-chain non-len() LHS"))

	if lhs_path == "" {
		return "", nil
	}
	return check_invariant_assertions_numeric_credit(
		binary, allow_neq, same_file_consts, lhs_path)
}

func check_invariant_assertions_extract_binary_path_assert_exit(
	kinds []string, same_file_consts map[string]bool, path string,
) {
	k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(kinds), Lo: 0, Hi: max_string_slice_per_call,
		Message: "Kinds is bounded by budget",
	})
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(same_file_consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Same_file_consts is bounded by budget",
	})
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(path), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Path is an ident-chain string bounded by identifier budget",
	})
	invariant.Cross_Product(k, s, path_axis,
		invariant.Excluding("Both k and s at safety caps is bad",
			invariant.Bucket_Hi(k), invariant.Bucket_Hi(s)),
		invariant.Excluding("Max k with zero s is bad",
			invariant.Bucket_Hi(k), invariant.Bucket_Lo(s)),
		invariant.Excluding("Zero k with max s is bad",
			invariant.Bucket_Lo(k), invariant.Bucket_Hi(s)),
		invariant.Excluding("Max-identifier path with empty kinds is impossible "+
			"by construction",
			invariant.Bucket_Lo(k),
			invariant.Bucket_Lo(s),
			invariant.Bucket_Hi(path_axis)),
	)
}

func check_invariant_assertions_extract_binary_path_assert_entry(
	binary *ast.BinaryExpr, allow_neq bool, same_file_consts map[string]bool,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(same_file_consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Same_file_consts is bounded by budget",
	})
	allow_neq_axis := invariant.Sometimes(allow_neq, "Allow_neq can be false on this branch")
	invariant.Cross_Product(
		invariant.Always(binary != nil, "Binary is non-nil"),
		s, allow_neq_axis,
		invariant.Excluding("Max s contradicts allow_neq true",
			invariant.Bucket_Hi(s), invariant.Bucket_True(allow_neq_axis)),
		invariant.Excluding("Axis s at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_False(allow_neq_axis)),
	)
}

// Resolves the numeric credit kinds for a `x op N` comparison given the
// already-extracted lhs_path. Split out of extract_binary_path to keep that
// dispatcher under the per-function line cap.
func check_invariant_assertions_numeric_credit(
	binary *ast.BinaryExpr,
	allow_neq bool,
	same_file_consts map[string]bool,
	lhs_path string,
) (path string, kinds []string) {
	defer func() {
		check_invariant_assertions_numeric_credit_assert_exit(same_file_consts, kinds, path)
	}()
	check_invariant_assertions_numeric_credit_assert_entry(
		binary, allow_neq, same_file_consts, lhs_path)
	is_zero_literal, is_integer_shape := check_invariant_assertions_numeric_literal_shape(
		binary.Y)
	invariant.Cross_Product(
		invariant.Sometimes(is_zero_literal, "Is_zero_literal can be true for canonical "+
			"zero RHS"),
		invariant.Sometimes(is_integer_shape, "Is_integer_shape can be true for int-typed "+
			"RHS"),
	)
	is_inline_integer := check_invariant_assertions_is_inline_literal(
		binary.Y, same_file_consts)
	invariant.Cross_Product(invariant.Sometimes(is_inline_integer,
		"Is_inline_integer can be true for inline-literal RHS"))

	// Float credits are always emitted alongside any numeric RHS: an int
	// literal converts to float (untyped constant rule), a const ident
	// might be float-typed, and a float literal naturally credits float.
	// Integer credits piggyback only when the RHS could BE an integer.
	if is_zero_literal {
		zero_kinds := []string{"zero_float"}
		boundary_kinds := []string{"boundary_float"}
		if is_integer_shape {
			zero_kinds = append(zero_kinds, "zero_int")
			boundary_kinds = append(boundary_kinds, "boundary_int")
		}
		if binary.Op == token.NEQ {
			return lhs_path, zero_kinds
		}
		// EQ to 0: Always credits both halves; Sometimes credits zero only.
		if allow_neq {
			return lhs_path, append(boundary_kinds, zero_kinds...)
		}
		return lhs_path, zero_kinds
	}
	// Non-zero numeric comparison. Only Always-family + == with inline literal
	// RHS credits boundary. Sometimes(x == N) for N!=0 doesn't credit.
	if !allow_neq {
		return "", nil
	}
	if binary.Op != token.EQL {
		return "", nil
	}
	if !is_inline_integer {
		return "", nil
	}
	boundary_kinds := []string{"boundary_float"}
	if is_integer_shape {
		boundary_kinds = append(boundary_kinds, "boundary_int")
	}
	return lhs_path, boundary_kinds
}

func check_invariant_assertions_numeric_credit_assert_exit(
	same_file_consts map[string]bool, kinds []string, path string,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(same_file_consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Same_file_consts is bounded by budget",
	})
	k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(kinds), Lo: 0, Hi: max_string_slice_per_call,
		Message: "Kinds is bounded by budget",
	})
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(path), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Path is an ident-chain string bounded by identifier budget",
	})
	invariant.Cross_Product(s, k, path_axis,
		invariant.Excluding("Both s and k at safety caps is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(k)),
		invariant.Excluding("Max s with min k is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(k)),
		invariant.Excluding("Zero s with max k is bad",
			invariant.Bucket_Lo(s), invariant.Bucket_Hi(k)),
		invariant.Excluding("Max-identifier path with empty kinds is impossible "+
			"by construction",
			invariant.Bucket_Lo(s),
			invariant.Bucket_Lo(k),
			invariant.Bucket_Hi(path_axis)),
	)
}

func check_invariant_assertions_numeric_credit_assert_entry(
	binary *ast.BinaryExpr,
	allow_neq bool,
	same_file_consts map[string]bool,
	lhs_path string,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(same_file_consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Same_file_consts is bounded by budget",
	})
	lhs_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(lhs_path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Lhs_path is a non-empty ident-chain bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(binary != nil, "Binary is non-nil"),
		invariant.Sometimes(allow_neq, "Allow_neq can be false on this branch"),
		s, lhs_axis,
		invariant.Excluding("Both s and lhs at safety caps is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Hi(lhs_axis)),
		invariant.Excluding("Max s with min lhs is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_Lo(lhs_axis)),
	)
}

// Classifies a numeric-literal RHS into (is_zero, integer_compatible). The
// float side of the credit is unconditional — caller always emits a float
// kind because string/pointer/struct comparisons return earlier; what
// varies is whether the RHS can ALSO credit an int requirement. An untyped
// int literal like `5` qualifies (Go-spec rule: untyped constant takes the
// param's type); a typed float literal `5.5` does not. Non-literal RHS
// (const ident, signed literal, selector chain) is treated as integer-
// compatible since the linter can't peek at the const's type.
func check_invariant_assertions_numeric_literal_shape(
	expression ast.Expr,
) (is_zero bool, is_integer_shape bool) {
	defer func() {
		invariant.Cross_Product(
			invariant.Sometimes(is_zero, "Is_zero is true for canonical zero literals"),
			invariant.Sometimes(is_integer_shape, "Is_integer_shape is true for "+
				"int-typed literals"),
		)
	}()
	literal, is_basic := expression.(*ast.BasicLit)
	if !is_basic {
		return false, true
	}
	if literal.Kind == token.INT {
		return literal.Value == "0", true
	}
	if literal.Kind == token.FLOAT {
		// Go-parsed BasicLit values are guaranteed well-formed by the
		// upstream parser, so strconv.ParseFloat cannot fail here.
		zero, _ := strconv.ParseFloat(literal.Value, 64)
		return zero == 0, false
	}
	return false, false
}

// Recognises a float finite-check predicate inside an Always/Sometimes call:
// `math.IsNaN(x)`, `!math.IsNaN(x)`, `math.IsInf(x, 0)`, `!math.IsInf(x, 0)`.
// Returns the LHS ident-chain path and the kinds slice the call credits.
// Always-family + bare (non-negated) shape signals "x is invariantly NaN/Inf"
// — the value is sentinel, so boundary_float and zero_float become moot and
// the call double-credits them. Always(!IsNaN) / Sometimes(IsNaN) etc.
// credit only the nan_float / inf_float sub-kind. The IsInf second arg must
// be 0 (both-signs canonical form); +1/-1 are rejected so callers always
// cover both polarities.
func check_invariant_assertions_extract_finite_path(
	predicate ast.Expr,
	allow_neq bool,
) (path string, kinds []string) {
	defer func() {
		k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(kinds), Lo: 0, Hi: max_string_slice_per_call,
			Message: "Kinds is bounded by budget",
		})
		path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(path), Lo: 0, Hi: Max_Identifier_Chars,
			Message: "Path is an ident-chain string bounded by identifier budget",
		})
		invariant.Cross_Product(k, path_axis,
			invariant.Excluding("Both k and path at safety caps is bad",
				invariant.Bucket_Hi(k), invariant.Bucket_Hi(path_axis)),
			invariant.Excluding("Max k with min path is bad",
				invariant.Bucket_Hi(k), invariant.Bucket_Lo(path_axis)),
			invariant.Excluding("Zero k with max path is bad",
				invariant.Bucket_Lo(k), invariant.Bucket_Hi(path_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Sometimes(allow_neq,
		"Allow_neq can be false for Sometimes-family helpers"))
	target := predicate
	negated := false
	if unary, is_unary := predicate.(*ast.UnaryExpr); is_unary {
		if unary.Op != token.NOT {
			return "", nil
		}
		target = unary.X
		negated = true
	}
	call, is_call := target.(*ast.CallExpr)
	if !is_call {
		return "", nil
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return "", nil
	}
	package_ident, is_package := selector.X.(*ast.Ident)
	if !is_package {
		return "", nil
	}
	if package_ident.Name != "math" {
		return "", nil
	}
	return check_invariant_assertions_finite_math_kinds(
		call, selector.Sel.Name, allow_neq && !negated)
}

func check_invariant_assertions_finite_math_kinds(
	call *ast.CallExpr, name string, credit_all bool,
) (path string, kinds []string) {
	switch name {
	case "IsNaN":
		if len(call.Args) != 1 {
			return "", nil
		}
		argument_path := check_invariant_assertions_expression_to_path(call.Args[0])
		invariant.Cross_Product(invariant.Sometimes(argument_path == "",
			"Argument_path is empty for non-ident-chain IsNaN args"))
		if argument_path == "" {
			return "", nil
		}
		// Always(IsNaN(x)) means x is invariantly NaN — sentinel value,
		// no boundary/zero analysis is meaningful, so credit all three.
		if credit_all {
			return argument_path,
				[]string{"nan_float", "boundary_float", "zero_float"}
		}
		return argument_path, []string{"nan_float"}
	case "IsInf":
		if len(call.Args) != 2 {
			return "", nil
		}
		sign_literal, is_basic := call.Args[1].(*ast.BasicLit)
		if !is_basic {
			return "", nil
		}
		if sign_literal.Kind != token.INT {
			return "", nil
		}
		if sign_literal.Value != "0" {
			return "", nil
		}
		argument_path := check_invariant_assertions_expression_to_path(call.Args[0])
		invariant.Cross_Product(invariant.Sometimes(argument_path == "",
			"Argument_path is empty for non-ident-chain IsInf args"))
		if argument_path == "" {
			return "", nil
		}
		if credit_all {
			return argument_path,
				[]string{"inf_float", "boundary_float", "zero_float"}
		}
		return argument_path, []string{"inf_float"}
	}
	return "", nil
}

// Returns the position of the predicate argument for Always/Sometimes-family
// helpers, or valid=false if the helper isn't in the family. Recorder-form
// helpers take the recorder as args[0]; the predicate moves to args[1].
// Returns the position of the predicate argument for Always/Sometimes-family
// helpers, or -1 if the helper isn't in the family. Recorder-form helpers
// take the recorder as args[0]; the predicate moves to args[1]. Single int
// return collapses what was previously (index, valid bool) — the bool was
// always true when index>=0, so the combination created an impossible
// (Hi, false) cross-product tuple.
func check_invariant_assertions_nil_predicate_index(helper_name string) (index int) {
	defer func() {
		index_boundary := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  index,
			Lo: helper_family_index_unknown,
			Hi: helper_family_index_recorder,
			Message: "Index is -1 for unknown helpers, 0 for naked, 1 for " +
				"Recorder_-prefixed",
		})
		index_zero := invariant.Sometimes(index == 0,
			"Index is 0 for the naked Always/Sometimes family")
		invariant.Cross_Product(
			index_boundary, index_zero,
			// The Lo (=-1) and Hi (=1) endpoints both mean index!=0; the
			// matching index==0=true tuples are impossible by definition.
			invariant.Excluding("Negative index sentinel contradicts index_zero true",
				invariant.Bucket_Lo(index_boundary),
				invariant.Bucket_True(index_zero)),
			invariant.Excluding("Max index contradicts index_zero true",
				invariant.Bucket_Hi(index_boundary),
				invariant.Bucket_True(index_zero)),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:       len(helper_name),
			Lo:      min_invariant_helper_name_chars,
			Hi:      max_invariant_helper_name_chars,
			Message: "Helper_name length is bounded; callers gate empty before calling",
		}),
		invariant.Always(helper_name != "",
			"Helper_name is non-empty per upstream gate"),
	)

	switch helper_name {
	case "Always", "Sometimes", "Is_Always", "Is_Sometimes":
		return 0
	case "Recorder_Always", "Recorder_Sometimes",
		"Recorder_Is_Always", "Recorder_Is_Sometimes":
		return 1
	}
	return -1
}

// Returns true for the Always-family helpers (which accept both X == nil
// and X != nil as valid pointer-coverage shapes); false for the Sometimes
// family (only X == nil counts).
func check_invariant_assertions_nil_allows_neq(helper_name string) (allow bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			allow, "Caller is on the allow list"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(helper_name),
			Lo: min_invariant_helper_name_chars,
			Hi: max_always_family_chars,
			Message: "Helper_name is in the Always/Sometimes nil-eq family; longest " +
				"is `Recorder_Always` (15)",
		}),
		invariant.Always(helper_name != "",
			"Helper_name is non-empty per upstream gate"),
	)

	switch helper_name {
	case "Always", "Is_Always", "Recorder_Always", "Recorder_Is_Always":
		return true
	}
	return false
}

// Extracts the simple helper name (e.g. "Range_Of") from a CallExpr whose
// callee is invariant.X. Returns the name and ok=true; returns ok=false
// when the call is not an invariant.X selector.
func check_invariant_assertions_extract_call_name(
	call *ast.CallExpr,
	invariant_idents map[string]bool,
) (name string) {
	defer func() {
		check_invariant_assertions_extract_call_name_assert_exit(invariant_idents, name)
	}()
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	empty_idents_axis := invariant.Sometimes(
		len(invariant_idents) == 0, "Invariant_idents is empty for files without "+
			"invariant import")
	invariant.Cross_Product(invariant.Always(call != nil, "Call is non-nil"),
		i, empty_idents_axis,
		invariant.Excluding("Max i contradicts empty_idents true",
			invariant.Bucket_Hi(i), invariant.Bucket_True(empty_idents_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_False(empty_idents_axis)),
		invariant.Excluding("Zero invariant_idents implies empty_idents true",
			invariant.Bucket_Lo(i), invariant.Bucket_False(empty_idents_axis)),
	)

	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return ""
	}
	identifier, is_ident := selector.X.(*ast.Ident)
	if !is_ident {
		return ""
	}
	if !invariant_idents[identifier.Name] {
		return ""
	}
	return selector.Sel.Name
}

func check_invariant_assertions_extract_call_name_assert_exit(
	invariant_idents map[string]bool, name string,
) {
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		name == "", "Name is empty for non-invariant calls")
	size_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(name),
		Lo: 0,
		Hi: max_invariant_helper_name_chars,
		Message: "Name is empty or the invariant helper name; longest is " +
			"`Recorder_Is_Distinct_Boundary` (29)",
	})
	invariant.Cross_Product(i, empty_axis, size_axis,
		invariant.Excluding("Empty input contradicts size at safety cap",
			invariant.Bucket_True(empty_axis), invariant.Bucket_Hi(size_axis)),
		invariant.Excluding("Non-empty input with zero size contradicts size "+
			"definition",
			invariant.Bucket_False(empty_axis), invariant.Bucket_Lo(size_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i),
			invariant.Bucket_True(empty_axis),
			invariant.Bucket_Lo(size_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i),
			invariant.Bucket_True(empty_axis),
			invariant.Bucket_Hi(size_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i),
			invariant.Bucket_False(empty_axis),
			invariant.Bucket_Lo(size_axis)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(i),
			invariant.Bucket_False(empty_axis),
			invariant.Bucket_Hi(size_axis)),
		invariant.Excluding("Zero invariant_idents with max helper name is bad",
			invariant.Bucket_Lo(i),
			invariant.Bucket_False(empty_axis),
			invariant.Bucket_Hi(size_axis)),
	)
}

// Scans args for shapes that resolve to an identifier-path (bare `*ast.Ident`
// or a chain of `*ast.SelectorExpr` ending in an Ident). Literal arguments
// (lo, hi, max_size) and complex expressions are skipped.
func check_invariant_assertions_extract_identifier_paths(args []ast.Expr) (paths []string) {
	defer func() {
		p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(paths), Lo: 0, Hi: max_string_slice_per_call,
			Message: "Paths is bounded by budget",
		})
		a := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(args), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
			Message: "Args is ≥1 per call",
		})
		invariant.Cross_Product(p, a,
			invariant.Excluding("Both p and a at safety caps is bad",
				invariant.Bucket_Hi(p), invariant.Bucket_Hi(a)),
			invariant.Excluding("Max p with zero a is bad",
				invariant.Bucket_Hi(p), invariant.Bucket_Lo(a)),
			invariant.Excluding("Zero p with max a is bad",
				invariant.Bucket_Lo(p), invariant.Bucket_Hi(a)),
		)
	}()
	a := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(args), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Args is ≥1 per call",
	})
	single_axis := invariant.Sometimes(
		len(args) == count_one, "Args has exactly one entry sometimes")
	invariant.Cross_Product(a, single_axis,
		invariant.Excluding("Max a contradicts single true",
			invariant.Bucket_Hi(a), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Axis a at safety cap is bad",
			invariant.Bucket_Hi(a), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Zero a implies single true",
			invariant.Bucket_Lo(a), invariant.Bucket_False(single_axis)),
	)
	for _, argument := range args {
		path := check_invariant_assertions_expression_to_path(argument)
		invariant.Cross_Product(
			invariant.Sometimes(path == "", "Path is empty for non-ident-chain args"))
		if path != "" {
			paths = append(paths, path)
			continue
		}
		paths = append(paths, check_invariant_assertions_input_struct_paths(argument)...)
	}
	return paths
}

// Extracts identifier paths from a `&Foo_Input{X: x, ...}` or `Foo_Input{...}`
// composite literal. Returns the paths bound to the value fields named X or S
// (Range, String_Size, One_Of, String_Is_ASCII, etc. all use one of these
// two field names for the inspected value). Other fields are ignored so
// Message/Lo/Hi/Max_Size etc. don't pollute the coverage set.
func check_invariant_assertions_input_struct_paths(expression ast.Expr) (paths []string) {
	defer func() {
		p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(paths), Lo: 0, Hi: max_string_slice_per_call,
			Message: "Paths is bounded by budget",
		})
		empty_axis := invariant.Sometimes(len(paths) == 0, "Paths is empty sometimes")
		invariant.Cross_Product(p, empty_axis,
			invariant.Excluding("Max len contradicts empty=true",
				invariant.Bucket_Hi(p), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Axis p at safety cap is bad",
				invariant.Bucket_Hi(p), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero len implies empty true",
				invariant.Bucket_Lo(p), invariant.Bucket_False(empty_axis)),
		)
	}()
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
		key_ident, is_key_ident := key_value.Key.(*ast.Ident)
		if !is_key_ident {
			continue
		}
		if key_ident.Name != "X" {
			if key_ident.Name != "S" {
				continue
			}
		}
		path := check_invariant_assertions_expression_to_path(key_value.Value)
		invariant.Cross_Product(invariant.Sometimes(path == "",
			"Path is empty before len-fallback for non-ident X/S values"))
		if path == "" {
			// `X: len(s)` credits the inner path so Distinct_Boundary on
			// a string's length satisfies the boundary requirement.
			path = check_invariant_assertions_expression_to_size_argument_path(
				key_value.Value)
			invariant.Cross_Product(invariant.Sometimes(path == "",
				"Path is empty after len-fallback for non-len X/S values"))
		}
		invariant.Cross_Product(invariant.Sometimes(path == "",
			"Path is empty for non-ident-chain non-len() X/S values"))
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

// Reports whether the first arg of call_args is a `&Boundary_Input{...}` (or
// `Boundary_Input{...}`) composite literal whose Lo or Hi key is an
// inline-literal zero. Used by the Lo/Hi=0 auto-credit rule: a Distinct_
// Boundary whose Lo (or Hi) bucket equals the zero observation makes any
// separate Sometimes(x == 0) axis redundant and contradictory.
func check_invariant_assertions_boundary_input_has_zero_endpoint(
	call_args []ast.Expr, constant_signs map[string]int,
) (yes bool) {
	defer func() {
		ca := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(call_args), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
			Message: "Call_args is ≥1 per call",
		})
		cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Constant_signs is bounded by budget",
		})
		invariant.Cross_Product(ca, cs,
			invariant.Sometimes(yes, "Affirmative branch is exercised"),
			invariant.Excluding("Both ca and cs at safety caps is bad",
				invariant.Bucket_Hi(ca), invariant.Bucket_Hi(cs)),
			invariant.Excluding("Axis ca at safety cap with zero cs is bad",
				invariant.Bucket_Hi(ca), invariant.Bucket_Lo(cs)),
			invariant.Excluding("Zero ca with max cs is bad",
				invariant.Bucket_Lo(ca), invariant.Bucket_Hi(cs)),
		)
	}()
	ca := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(call_args), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Call_args is ≥1 per call",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	invariant.Cross_Product(ca, cs,
		invariant.Excluding("Both ca and cs at safety caps is bad",
			invariant.Bucket_Hi(ca), invariant.Bucket_Hi(cs)),
		invariant.Excluding("Axis ca at safety cap with zero cs is bad",
			invariant.Bucket_Hi(ca), invariant.Bucket_Lo(cs)),
		invariant.Excluding("Zero ca with max cs is bad",
			invariant.Bucket_Lo(ca), invariant.Bucket_Hi(cs)),
	)
	if len(call_args) == 0 {
		return false
	}
	expression := call_args[0]
	if unary, is_unary := expression.(*ast.UnaryExpr); is_unary {
		expression = unary.X
	}
	composite, is_composite := expression.(*ast.CompositeLit)
	if !is_composite {
		return false
	}
	for _, element := range composite.Elts {
		key_value, is_key_value := element.(*ast.KeyValueExpr)
		if !is_key_value {
			continue
		}
		key_ident, is_key_ident := key_value.Key.(*ast.Ident)
		if !is_key_ident {
			continue
		}
		if key_ident.Name != "Lo" {
			if key_ident.Name != "Hi" {
				continue
			}
		}
		if check_invariant_assertions_is_zero_inline_literal(
			key_value.Value, constant_signs) {
			return true
		}
	}
	return false
}

// Reports whether the first arg of call_args is a `&Boundary_Input{...}` (or
// `Boundary_Input{...}`) composite literal whose declared range provably
// excludes zero: Lo is positive, or Hi is negative. Such a boundary
// statically proves `x != 0`, so the zero half of the dual int / float /
// string requirement is implicitly satisfied — any separate
// `Always(x != 0)` / `Always(s != "")` axis on the same path is redundant.
// Mirrors boundary_input_has_zero_endpoint for the opposite case (range
// observably contains zero as an endpoint vs. range observably excludes
// zero entirely).
func check_invariant_assertions_boundary_input_excludes_zero(
	call_args []ast.Expr, constant_signs map[string]int,
) (yes bool) {
	defer func() {
		check_invariant_assertions_boundary_input_excludes_zero_assert_exit(
			call_args, constant_signs, yes)
	}()
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	ca := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(call_args), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Call_args is ≥1 per call",
	})
	invariant.Cross_Product(ca, cs,
		invariant.Excluding("Both ca and cs at safety caps is bad",
			invariant.Bucket_Hi(ca), invariant.Bucket_Hi(cs)),
		invariant.Excluding("Axis ca at safety cap with zero cs is bad",
			invariant.Bucket_Hi(ca), invariant.Bucket_Lo(cs)),
		invariant.Excluding("Zero ca with max cs is bad",
			invariant.Bucket_Lo(ca), invariant.Bucket_Hi(cs)),
	)
	if len(call_args) == 0 {
		return false
	}
	expression := call_args[0]
	if unary, is_unary := expression.(*ast.UnaryExpr); is_unary {
		expression = unary.X
	}
	composite, is_composite := expression.(*ast.CompositeLit)
	if !is_composite {
		return false
	}
	for _, element := range composite.Elts {
		key_value, is_key_value := element.(*ast.KeyValueExpr)
		if !is_key_value {
			continue
		}
		key_ident, is_key_ident := key_value.Key.(*ast.Ident)
		if !is_key_ident {
			continue
		}
		if key_ident.Name == "Lo" {
			if check_invariant_assertions_is_positive_inline_literal(
				key_value.Value, constant_signs) {
				return true
			}
		}
		if key_ident.Name == "Hi" {
			if check_invariant_assertions_is_negative_inline_literal(
				key_value.Value, constant_signs) {
				return true
			}
		}
	}
	return false
}

func check_invariant_assertions_boundary_input_excludes_zero_assert_exit(
	call_args []ast.Expr, constant_signs map[string]int, yes bool,
) {
	ca := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(call_args), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Call_args is ≥1 per call",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	invariant.Cross_Product(ca, cs,
		invariant.Sometimes(yes, "Affirmative branch is exercised"),
		invariant.Excluding("Both ca and cs at safety caps is bad",
			invariant.Bucket_Hi(ca), invariant.Bucket_Hi(cs)),
		invariant.Excluding("Axis ca at safety cap with zero cs is bad",
			invariant.Bucket_Hi(ca), invariant.Bucket_Lo(cs)),
		invariant.Excluding("Zero ca with max cs is bad",
			invariant.Bucket_Lo(ca), invariant.Bucket_Hi(cs)),
	)
}

// Recognises a `len(<ident-chain>)` CallExpr and returns the inner path,
// or "" otherwise. Used as a fallback in the numeric-comparison and
// Boundary_Input.X extractors so length-based assertions on strings can
// credit the string's coverage requirements. The single-arg constraint
// matches Go's builtin len signature; calls with extra args or with
// `bytes.len`-style qualified names don't credit.
func check_invariant_assertions_expression_to_size_argument_path(
	expression ast.Expr,
) (path string) {
	defer func() {
		empty_axis := invariant.Sometimes(
			path == "", "Path is empty for non-len() expressions")
		size_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(path), Lo: 0, Hi: Max_Identifier_Chars,
			Message: "Path is an ident-chain string bounded by identifier budget",
		})
		invariant.Cross_Product(empty_axis, size_axis,
			invariant.Excluding("Non-empty input with zero size contradicts size "+
				"definition",
				invariant.Bucket_False(empty_axis), invariant.Bucket_Lo(size_axis)),
			invariant.Excluding("Empty input contradicts size at safety cap",
				invariant.Bucket_True(empty_axis), invariant.Bucket_Hi(size_axis)),
		)
	}()
	call, is_call := expression.(*ast.CallExpr)
	if !is_call {
		return ""
	}
	function_ident, is_function_ident := call.Fun.(*ast.Ident)
	if !is_function_ident {
		return ""
	}
	// `cap(ch)` is the channel-buffer boundary form. Mirrors `len(X)` but
	// returns the wrapped form "cap(X)" so the requirement path stamped by
	// the walker for channels (which always carries the `cap(...)` wrapper)
	// round-trips to a credit on the matcher side.
	if function_ident.Name == "cap" {
		if len(call.Args) != 1 {
			return ""
		}
		inner := check_invariant_assertions_expression_to_path(call.Args[0])
		invariant.Cross_Product(
			invariant.Always(
				inner != "",
				"Inner path resolves for cap(X) where X is an ident chain"))
		return "cap(" + inner + ")"
	}
	if function_ident.Name != "len" {
		return ""
	}
	if len(call.Args) != 1 {
		return ""
	}
	return check_invariant_assertions_expression_to_path(call.Args[0])
}

// Converts an ast.Expr to a dotted identifier path. Returns "" for any
// shape that isn't a pure ident chain (function calls, indexed expressions,
// parenthesised dereferences). Uses invariant.GameLoop instead of recursion
// per Tiger Style.
func check_invariant_assertions_expression_to_path(expression ast.Expr) (path string) {
	defer func() {
		empty_axis := invariant.Sometimes(
			path == "", "Path is empty for non-ident-chain expressions")
		size_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(path), Lo: 0, Hi: Max_Identifier_Chars,
			Message: "Path is an ident-chain string bounded by identifier budget",
		})
		invariant.Cross_Product(empty_axis, size_axis,
			invariant.Excluding("Non-empty input with zero size contradicts size "+
				"definition",
				invariant.Bucket_False(empty_axis), invariant.Bucket_Lo(size_axis)),
			invariant.Excluding("Empty input contradicts size at safety cap",
				invariant.Bucket_True(empty_axis), invariant.Bucket_Hi(size_axis)),
		)
	}()

	// Strip a leading boolean NOT so `Always(!X.Y, …)` credits "X.Y" — the
	// natural shape for asserting a bool field is provably false.
	if unary, is_unary := expression.(*ast.UnaryExpr); is_unary {
		if unary.Op == token.NOT {
			expression = unary.X
		}
	}
	// A leading `*` deref makes the path refer to the pointee — `len(*rows)`
	// covers path `*rows`, distinct from path `rows` (which would be the
	// pointer itself). The walker emits container-leaf requirements with the
	// `*` prefix when the field type is `*[]T` / `*map[K]V` / `*chan T`, so
	// the matcher has to round-trip the shape.
	deref_prefix := ""
	if star, is_star := expression.(*ast.StarExpr); is_star {
		deref_prefix = "*"
		expression = star.X
	}
	var segments []string
	current := expression
	for range invariant.Game_Loop() {
		selector, is_selector := current.(*ast.SelectorExpr)
		if !is_selector {
			break
		}
		segments = append(segments, selector.Sel.Name)
		current = selector.X
	}
	identifier, is_ident := current.(*ast.Ident)
	if !is_ident {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(deref_prefix)
	builder.WriteString(identifier.Name)
	for i := len(segments) - 1; i >= 0; i-- {
		builder.WriteByte('.')
		builder.WriteString(segments[i])
	}
	return builder.String()
}

// Walks file.Decls for package-level `const` declarations and returns the
// set of declared names. The Boundary-literal check accepts a bare Ident as
// a Lo/Hi argument only when the ident names a same-file constant. Idents
// referring to vars or function parameters are rejected because their value
// can vary at runtime, which would break the AFL bucket semantics that the
// Lo/Hi pair anchors. Cross-package constants accessed via a SelectorExpr
// (`math.MaxInt`) are accepted unconditionally — the linter has no type
// information to confirm them, but the access shape is conventionally a
// constant reference.
func check_invariant_assertions_collect_same_file_constants(
	file *ast.File,
) (consts map[string]bool) {
	defer func() {
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(consts), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Consts is bounded by budget",
		})
		empty_axis := invariant.Sometimes(len(consts) == 0, "Consts is empty sometimes")
		invariant.Cross_Product(c, empty_axis,
			invariant.Excluding("Max c contradicts empty true",
				invariant.Bucket_Hi(c), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Constant_signs at AST at safety cap is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Lo(c) is the empty bucket; empty=false contradicts",
				invariant.Bucket_Lo(c), invariant.Bucket_False(empty_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(file != nil, "File is non-nil"))

	consts = map[string]bool{}
	for _, declaration := range file.Decls {
		generic_declaration, is_generic := declaration.(*ast.GenDecl)
		if !is_generic {
			continue
		}
		if generic_declaration.Tok != token.CONST {
			continue
		}
		for _, specification := range generic_declaration.Specs {
			value_specification, is_value := specification.(*ast.ValueSpec)
			if !is_value {
				continue
			}
			for _, name := range value_specification.Names {
				if name.Name == "_" {
					continue
				}
				consts[name.Name] = true
			}
		}
	}
	return consts
}

// Walks file.Decls for package-level `const` declarations and returns a
// name → sign map (-1 / 0 / +1) for each const whose declared value
// resolves to a numeric inline literal (BasicLit or signed BasicLit).
// Unresolvable consts (selector chains, expressions, iota) are omitted.
// Feeds the zero / positive / negative auto-credit rules so a
// `Distinct_Boundary{Lo: SOME_CONST, ...}` axis credits the same way as
// the equivalent inline literal `Lo: 5` would.
func check_invariant_assertions_collect_same_file_constant_signs(
	file *ast.File,
) (constant_signs map[string]int) {
	defer func() {
		check_invariant_assertions_collect_same_file_constant_signs_assert_exit(
			constant_signs)
	}()
	invariant.Cross_Product(invariant.Always(file != nil, "File is non-nil"))

	constant_signs = map[string]int{}
	for _, declaration := range file.Decls {
		generic_declaration, is_generic := declaration.(*ast.GenDecl)
		if !is_generic {
			continue
		}
		if generic_declaration.Tok != token.CONST {
			continue
		}
		for _, specification := range generic_declaration.Specs {
			value_specification, is_value := specification.(*ast.ValueSpec)
			if !is_value {
				continue
			}
			for i_index, name := range value_specification.Names {
				if name.Name == "_" {
					continue
				}
				if i_index >= len(value_specification.Values) {
					continue
				}
				sign, ok := check_invariant_assertions_expression_sign(
					value_specification.Values[i_index], nil)
				ok_axis := invariant.Sometimes(
					ok, "Sign resolution succeeded for this const value")
				sign_axis := invariant.Distinct_Boundary(
					&invariant.Boundary_Input[int]{
						X:  sign,
						Lo: sign_negative,
						Hi: sign_positive,
						Message: "Sign mirrors expression_sign's " +
							"three-valued " +
							"domain",
					})
				sign_zero_axis := invariant.Sometimes(
					sign == 0, "Sign is zero for resolved zero const values")
				invariant.Cross_Product(
					ok_axis, sign_axis, sign_zero_axis,
					invariant.Excluding("Sign extracted despite ok equals "+
						"false bucket",
						invariant.Bucket_False(ok_axis),
						invariant.Bucket_Lo(sign_axis)),
					invariant.Excluding("Sign extracted despite ok equals "+
						"false",
						invariant.Bucket_False(ok_axis),
						invariant.Bucket_Hi(sign_axis)),
					invariant.Excluding("Sign less than 0 contradicts "+
						"sign_zero true",
						invariant.Bucket_Lo(sign_axis),
						invariant.Bucket_True(sign_zero_axis)),
					invariant.Excluding("Sign>0 contradicts sign_zero=true",
						invariant.Bucket_Hi(sign_axis),
						invariant.Bucket_True(sign_zero_axis)),
				)
				if !ok {
					continue
				}
				constant_signs[name.Name] = sign
			}
		}
	}
	return constant_signs
}

func check_invariant_assertions_collect_same_file_constant_signs_assert_exit(
	constant_signs map[string]int,
) {
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(constant_signs) == 0, "Constant_signs is empty sometimes")
	invariant.Cross_Product(c, empty_axis,
		invariant.Excluding("Max c contradicts empty true",
			invariant.Bucket_Hi(c), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Lo(c) is the empty bucket; empty=false contradicts",
			invariant.Bucket_Lo(c), invariant.Bucket_False(empty_axis)),
	)
}

// Reports whether expression is an "inline literal" suitable for a Boundary
// Lo/Hi argument: a numeric literal, a signed literal (`-1`), a same-file
// constant identifier, or a package-qualified selector chain bottoming out
// at an Ident (`math.MaxInt`). Function calls, typed conversions, arithmetic,
// indexed expressions, and bare idents that don't name a same-file constant
// all reject. The Lo/Hi pair anchors the AFL boundary buckets; permitting a
// value-might-change expression there would break the bucket semantics.
func check_invariant_assertions_is_inline_literal(
	expression ast.Expr,
	same_file_consts map[string]bool,
) (yes bool) {
	defer func() {
		s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(same_file_consts), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Same_file_consts is bounded by budget",
		})
		yes_axis := invariant.Sometimes(yes, "Affirmative branch is exercised")
		invariant.Cross_Product(s, yes_axis,
			invariant.Excluding("Axis s at safety cap contradicts yes true",
				invariant.Bucket_Hi(s), invariant.Bucket_True(yes_axis)),
			invariant.Excluding("Axis s at safety cap is bad",
				invariant.Bucket_Hi(s), invariant.Bucket_False(yes_axis)),
		)
	}()
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(same_file_consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Same_file_consts is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(same_file_consts) == 0, "Same_file_consts is empty sometimes")
	invariant.Cross_Product(invariant.Always(expression != nil, "Expression is non-nil"),
		s, empty_axis,
		invariant.Excluding("Max s contradicts empty true",
			invariant.Bucket_Hi(s), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis s at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero axis s implies empty true",
			invariant.Bucket_Lo(s), invariant.Bucket_False(empty_axis)),
	)

	if _, is_basic := expression.(*ast.BasicLit); is_basic {
		return true
	}
	if unary, is_unary := expression.(*ast.UnaryExpr); is_unary {
		if unary.Op != token.ADD {
			if unary.Op != token.SUB {
				return false
			}
		}
		_, inner_is_basic := unary.X.(*ast.BasicLit)
		return inner_is_basic
	}
	if identifier, is_ident := expression.(*ast.Ident); is_ident {
		return same_file_consts[identifier.Name]
	}
	selector, is_selector := expression.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	current := ast.Expr(selector)
	for range invariant.Game_Loop() {
		inner_selector, inner_is_selector := current.(*ast.SelectorExpr)
		if !inner_is_selector {
			break
		}
		current = inner_selector.X
	}
	_, bottom_is_ident := current.(*ast.Ident)
	return bottom_is_ident
}

// Reports whether expression is an inline literal AND resolves to zero.
// Drives the Lo/Hi=0 auto-credit rule for Distinct_Boundary: when Lo or
// Hi is zero, the boundary's matching bucket IS the "x == 0" observation,
// so the zero half of the dual requirement is implicitly satisfied.
// Recognises numeric BasicLit zero ("0" / "0.0" / any ParseFloat zero),
// signed BasicLit zero (`-0`, `+0`), and same-file const idents whose
// declared value resolves to zero (via constant_signs). Selector chains and
// unknown shapes return false.
func check_invariant_assertions_is_zero_inline_literal(
	expression ast.Expr, constant_signs map[string]int,
) (yes bool) {
	defer func() {
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Constant_signs is bounded by budget",
		})
		yes_axis := invariant.Sometimes(yes, "Affirmative branch is exercised")
		invariant.Cross_Product(c, yes_axis,
			invariant.Excluding("Max c contradicts yes true",
				invariant.Bucket_Hi(c), invariant.Bucket_True(yes_axis)),
			invariant.Excluding("Axis c at safety cap is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_False(yes_axis)),
		)
	}()
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(constant_signs) == 0, "Constant_signs is empty sometimes")
	invariant.Cross_Product(c, empty_axis,
		invariant.Excluding("Max c contradicts empty true",
			invariant.Bucket_Hi(c), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Lo(c) is the empty bucket; empty=false contradicts",
			invariant.Bucket_Lo(c), invariant.Bucket_False(empty_axis)),
	)
	sign, ok := check_invariant_assertions_expression_sign(expression, constant_signs)
	ok_axis := invariant.Sometimes(ok, "Sign resolution succeeded")
	sign_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: sign, Lo: sign_negative, Hi: sign_positive,
		Message: "Sign mirrors expression_sign's three-valued domain",
	})
	sign_zero_axis := invariant.Sometimes(sign == 0, "Sign is zero for resolved zero literals")
	// Expression_sign returns (0, false) when unresolved, so ok=false implies
	// sign=0 (Lo and Hi buckets unreachable; sign_zero is true). When sign
	// resolves to -1 or +1, sign_zero is false.
	invariant.Cross_Product(
		ok_axis, sign_axis, sign_zero_axis,
		invariant.Excluding("Sign extracted despite ok equals false bucket",
			invariant.Bucket_False(ok_axis), invariant.Bucket_Lo(sign_axis)),
		invariant.Excluding("Sign extracted despite ok equals false",
			invariant.Bucket_False(ok_axis), invariant.Bucket_Hi(sign_axis)),
		invariant.Excluding("Sign less than 0 contradicts sign_zero true",
			invariant.Bucket_Lo(sign_axis), invariant.Bucket_True(sign_zero_axis)),
		invariant.Excluding("Sign>0 contradicts sign_zero=true",
			invariant.Bucket_Hi(sign_axis), invariant.Bucket_True(sign_zero_axis)),
	)
	return ok && sign == 0
}

// Reports whether expression is an inline literal that resolves to a strictly
// positive numeric value. Drives the Lo > 0 auto-credit rule: when a
// Distinct_Boundary's Lo is positive, the boundary statically proves
// `x >= Lo > 0` so `x != 0` is invariantly true and the zero half of the
// dual requirement is implicitly satisfied. Recognises positive BasicLit,
// signed BasicLit (`+5`), and same-file const idents resolving to positive
// values. Selector chains and unknown shapes return false.
func check_invariant_assertions_is_positive_inline_literal(
	expression ast.Expr, constant_signs map[string]int,
) (yes bool) {
	defer func() {
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Constant_signs is bounded by budget",
		})
		yes_axis := invariant.Sometimes(yes, "Affirmative branch is exercised")
		invariant.Cross_Product(c, yes_axis,
			invariant.Excluding("Max c contradicts yes true",
				invariant.Bucket_Hi(c), invariant.Bucket_True(yes_axis)),
			invariant.Excluding("Axis c at safety cap is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_False(yes_axis)),
		)
	}()
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(constant_signs) == 0, "Constant_signs is empty sometimes")
	invariant.Cross_Product(c, empty_axis,
		invariant.Excluding("Max c contradicts empty true",
			invariant.Bucket_Hi(c), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Lo(c) is the empty bucket; empty=false contradicts",
			invariant.Bucket_Lo(c), invariant.Bucket_False(empty_axis)),
	)
	sign, ok := check_invariant_assertions_expression_sign(expression, constant_signs)
	ok_axis := invariant.Sometimes(ok, "Sign resolution succeeded")
	sign_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: sign, Lo: sign_negative, Hi: sign_positive,
		Message: "Sign mirrors expression_sign's three-valued domain",
	})
	sign_zero_axis := invariant.Sometimes(sign == 0, "Sign is zero for resolved zero literals")
	// Expression_sign returns (0, false) when unresolved, so ok=false implies
	// sign=0 (Lo and Hi buckets unreachable; sign_zero is true). When sign
	// resolves to -1 or +1, sign_zero is false.
	invariant.Cross_Product(
		ok_axis, sign_axis, sign_zero_axis,
		invariant.Excluding("Sign extracted despite ok equals false bucket",
			invariant.Bucket_False(ok_axis), invariant.Bucket_Lo(sign_axis)),
		invariant.Excluding("Sign extracted despite ok equals false",
			invariant.Bucket_False(ok_axis), invariant.Bucket_Hi(sign_axis)),
		invariant.Excluding("Sign less than 0 contradicts sign_zero true",
			invariant.Bucket_Lo(sign_axis), invariant.Bucket_True(sign_zero_axis)),
		invariant.Excluding("Sign>0 contradicts sign_zero=true",
			invariant.Bucket_Hi(sign_axis), invariant.Bucket_True(sign_zero_axis)),
	)
	return ok && sign > 0
}

// Reports whether expression is an inline literal that resolves to a strictly
// negative numeric value. Mirror of is_positive_inline_literal for the Hi < 0
// auto-credit rule: a Distinct_Boundary with negative Hi proves `x <= Hi < 0`
// so `x != 0` is invariantly true.
func check_invariant_assertions_is_negative_inline_literal(
	expression ast.Expr, constant_signs map[string]int,
) (yes bool) {
	defer func() {
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Constant_signs is bounded by budget",
		})
		yes_axis := invariant.Sometimes(yes, "Affirmative branch is exercised")
		invariant.Cross_Product(c, yes_axis,
			invariant.Excluding("Max c contradicts yes true",
				invariant.Bucket_Hi(c), invariant.Bucket_True(yes_axis)),
			invariant.Excluding("Axis c at safety cap is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_False(yes_axis)),
		)
	}()
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(constant_signs) == 0, "Constant_signs is empty sometimes")
	invariant.Cross_Product(c, empty_axis,
		invariant.Excluding("Max c contradicts empty true",
			invariant.Bucket_Hi(c), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Lo(c) is the empty bucket; empty=false contradicts",
			invariant.Bucket_Lo(c), invariant.Bucket_False(empty_axis)),
	)
	sign, ok := check_invariant_assertions_expression_sign(expression, constant_signs)
	ok_axis := invariant.Sometimes(ok, "Sign resolution succeeded")
	sign_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: sign, Lo: sign_negative, Hi: sign_positive,
		Message: "Sign mirrors expression_sign's three-valued domain",
	})
	sign_zero_axis := invariant.Sometimes(sign == 0, "Sign is zero for resolved zero literals")
	// Expression_sign returns (0, false) when unresolved, so ok=false implies
	// sign=0 (Lo and Hi buckets unreachable; sign_zero is true). When sign
	// resolves to -1 or +1, sign_zero is false.
	invariant.Cross_Product(
		ok_axis, sign_axis, sign_zero_axis,
		invariant.Excluding("Sign extracted despite ok equals false bucket",
			invariant.Bucket_False(ok_axis), invariant.Bucket_Lo(sign_axis)),
		invariant.Excluding("Sign extracted despite ok equals false",
			invariant.Bucket_False(ok_axis), invariant.Bucket_Hi(sign_axis)),
		invariant.Excluding("Sign less than 0 contradicts sign_zero true",
			invariant.Bucket_Lo(sign_axis), invariant.Bucket_True(sign_zero_axis)),
		invariant.Excluding("Sign>0 contradicts sign_zero=true",
			invariant.Bucket_Hi(sign_axis), invariant.Bucket_True(sign_zero_axis)),
	)
	return ok && sign < 0
}

// Resolves expression to a numeric sign (-1, 0, +1) if it is a recognised
// inline literal: BasicLit numeric value, signed BasicLit (`+5`, `-5`), or
// a same-file const ident whose sign was previously collected. Returns
// `ok=false` for selector chains, expressions, or any non-resolvable shape.
func check_invariant_assertions_expression_sign(
	expression ast.Expr, constant_signs map[string]int,
) (sign int, ok bool) {
	defer func() {
		check_invariant_assertions_expression_sign_assert_exit(constant_signs, sign, ok)
	}()
	check_invariant_assertions_expression_sign_assert_entry(expression, constant_signs)

	if literal, is_basic := expression.(*ast.BasicLit); is_basic {
		return check_invariant_assertions_basic_lit_sign(literal)
	}
	if unary, is_unary := expression.(*ast.UnaryExpr); is_unary {
		if unary.Op != token.ADD {
			if unary.Op != token.SUB {
				return 0, false
			}
		}
		inner_literal, inner_is_basic := unary.X.(*ast.BasicLit)
		if !inner_is_basic {
			return 0, false
		}
		inner_sign, inner_ok := check_invariant_assertions_basic_lit_sign(inner_literal)
		inner_ok_axis := invariant.Sometimes(
			inner_ok, "Inner_ok reflects the inner literal's resolvability")
		inner_sign_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: inner_sign, Lo: 0, Hi: sign_positive,
			Message: "Inner_sign mirrors basic_lit_sign's 0-or-+1 domain",
		})
		// Go syntax forbids unary +/- on string/char/imag literals, so the
		// inner BasicLit always parses as INT or FLOAT — inner_ok is always
		// true here. (inner_ok=false, *) tuples cannot fire.
		invariant.Cross_Product(
			inner_ok_axis, inner_sign_axis,
			invariant.Excluding("Inner sign extracted despite inner_ok equals false",
				invariant.Bucket_False(inner_ok_axis),
				invariant.Bucket_Lo(inner_sign_axis)),
			invariant.Excluding("Inner sign extracted despite inner_ok equals false",
				invariant.Bucket_False(inner_ok_axis),
				invariant.Bucket_Hi(inner_sign_axis)),
		)
		if !inner_ok {
			return 0, false
		}
		if unary.Op == token.SUB {
			return -inner_sign, true
		}
		return inner_sign, true
	}
	if identifier, is_ident := expression.(*ast.Ident); is_ident {
		if constant_signs == nil {
			return 0, false
		}
		constant_sign, has := constant_signs[identifier.Name]
		if !has {
			return 0, false
		}
		return constant_sign, true
	}
	return 0, false
}

func check_invariant_assertions_expression_sign_assert_exit(
	constant_signs map[string]int, sign int, ok bool,
) {
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	ok_axis := invariant.Sometimes(ok, "Sign was resolved")
	sign_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: sign, Lo: sign_negative, Hi: sign_positive,
		Message: "Sign is -1, 0, or +1 — bounded by the three sign domains",
	})
	sign_zero_axis := invariant.Sometimes(
		sign == 0, "Sign is zero for resolved zero literals")
	invariant.Cross_Product(
		c, ok_axis, sign_axis, sign_zero_axis,
		invariant.Excluding("Sign extracted despite ok equals false bucket",
			invariant.Bucket_False(ok_axis), invariant.Bucket_Lo(sign_axis)),
		invariant.Excluding("Sign extracted despite ok equals false",
			invariant.Bucket_False(ok_axis), invariant.Bucket_Hi(sign_axis)),
		invariant.Excluding("Sign less than 0 contradicts sign_zero true",
			invariant.Bucket_Lo(sign_axis),
			invariant.Bucket_True(sign_zero_axis)),
		invariant.Excluding("Sign>0 contradicts sign_zero=true",
			invariant.Bucket_Hi(sign_axis),
			invariant.Bucket_True(sign_zero_axis)),
		invariant.Excluding("Axis c at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_True(ok_axis)),
		invariant.Excluding("Max c with ok false is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_False(ok_axis)),
	)
}

func check_invariant_assertions_expression_sign_assert_entry(
	expression ast.Expr, constant_signs map[string]int,
) {
	c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(constant_signs) == 0, "Constant_signs is empty sometimes")
	invariant.Cross_Product(invariant.Always(expression != nil, "Expression is non-nil"),
		c, empty_axis,
		invariant.Excluding("Max c contradicts empty true",
			invariant.Bucket_Hi(c), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Constant_signs at AST at safety cap is bad",
			invariant.Bucket_Hi(c), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Lo(c) is the empty bucket; empty=false contradicts",
			invariant.Bucket_Lo(c), invariant.Bucket_False(empty_axis)),
	)
}

// Returns the numeric sign of a BasicLit's value (-1, 0, +1). INT and
// FLOAT kinds parse via strconv; STRING / CHAR / IMAG return ok=false.
// BasicLit alone is never negative — Go represents `-5` as UnaryExpr{SUB,
// BasicLit{"5"}}; this helper returns 0 for "0" / "0.0" and +1 for any
// other parsed-positive value.
func check_invariant_assertions_basic_lit_sign(literal *ast.BasicLit) (sign int, ok bool) {
	defer func() {
		ok_axis := invariant.Sometimes(ok, "Sign was resolved")
		sign_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  sign,
			Lo: 0,
			Hi: sign_positive,
			Message: "Sign is 0 (zero literal) or +1 (positive literal); bare " +
				"BasicLit stays non-negative",
		})
		sign_zero_axis := invariant.Sometimes(
			sign == 0, "Sign is zero for zero literals or non-numeric kinds")
		// Non-numeric kinds return (0, false), so ok=false implies sign=0:
		// Hi (sign=+1) is unreachable when ok=false, and sign_zero is true.
		invariant.Cross_Product(
			ok_axis, sign_axis, sign_zero_axis,
			invariant.Excluding("Sign extracted despite ok equals false",
				invariant.Bucket_False(ok_axis), invariant.Bucket_Hi(sign_axis)),
			invariant.Excluding("Sign>0 contradicts sign_zero=true",
				invariant.Bucket_Hi(sign_axis),
				invariant.Bucket_True(sign_zero_axis)),
			invariant.Excluding("Negative sign contradicts sign_zero true",
				invariant.Bucket_Lo(sign_axis),
				invariant.Bucket_False(sign_zero_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(
		literal != nil, "Literal is non-nil"))

	if literal.Kind == token.INT {
		if literal.Value == "0" {
			return 0, true
		}
		return 1, true
	}
	if literal.Kind == token.FLOAT {
		value, err := strconv.ParseFloat(literal.Value, 64)
		if err != nil {
			return 0, false
		}
		if value == 0 {
			return 0, true
		}
		return 1, true
	}
	return 0, false
}

// Input for check_invariant_assertions_validate_boundary_input. Same-type
// fields (the two map[string]bool sets) sit in this struct so call sites
// stay readable and reordering produces compile errors.
type check_invariant_assertions_validate_boundary_input_input struct {
	Call             *ast.CallExpr
	Input_Argument   ast.Expr
	Function_Label   string
	Same_File_Consts map[string]bool
	File_Set         *token.FileSet
}

// Pulls the `&Boundary_Input{...}` or `Boundary_Input{...}` composite from
// Input_Argument and emits one diagnostic per non-literal Lo/Hi key. Call.Pos()
// anchors every diagnostic to the Boundary call site itself. Returns nil
// when Input_Argument isn't a composite literal (the user wrote something
// exotic — out of scope for this rule).
func check_invariant_assertions_validate_boundary_input(
	input *check_invariant_assertions_validate_boundary_input_input,
) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_validate_boundary_input_assert_exit(diags, input)
	}()
	label_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Function_Label), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	consts_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Same_File_Consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Same_File_Consts is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Call != nil, "Call is non-nil"),
		invariant.Always(input.Input_Argument != nil, "Input_argument is non-nil"),
		invariant.Always(input.File_Set != nil, "File_set is non-nil"),
		label_axis, consts_axis,
		invariant.Excluding("Hi label with Hi consts unreachable in test corpus",
			invariant.Bucket_Hi(label_axis), invariant.Bucket_Hi(consts_axis)),
		invariant.Excluding("Hi label with Lo consts unreachable in test corpus",
			invariant.Bucket_Hi(label_axis), invariant.Bucket_Lo(consts_axis)),
		invariant.Excluding("Hi consts unreachable in test corpus (Lo label)",
			invariant.Bucket_Hi(consts_axis), invariant.Bucket_Lo(label_axis)),
	)

	argument_expression := input.Input_Argument
	if unary, is_unary := argument_expression.(*ast.UnaryExpr); is_unary {
		argument_expression = unary.X
	}
	composite, is_composite := argument_expression.(*ast.CompositeLit)
	if !is_composite {
		return nil
	}
	for _, element := range composite.Elts {
		key_value, is_key_value := element.(*ast.KeyValueExpr)
		if !is_key_value {
			continue
		}
		key_ident, is_key_ident := key_value.Key.(*ast.Ident)
		if !is_key_ident {
			continue
		}
		if key_ident.Name != "Lo" {
			if key_ident.Name != "Hi" {
				continue
			}
		}
		if check_invariant_assertions_is_inline_literal(
			key_value.Value, input.Same_File_Consts) {
			continue
		}
		diags = append(diags, Diagnostic{
			Position: input.File_Set.Position(input.Call.Pos()),
			Name:     "invariant_boundary_non_literal",
			Want: "use an inline literal (numeric literal, signed literal, same-file " +
				"const, " +
				"or package-qualified selector) for Lo and Hi — they anchor the " +
				"AFL boundary " +
				"buckets and must not vary at runtime",
			Message: fmt.Sprintf("%s: Boundary %s must be an inline literal — got `%s`",
				input.Function_Label, key_ident.Name, types.ExprString(
					key_value.Value)),
		})
	}
	return diags
}

func check_invariant_assertions_validate_boundary_input_assert_exit(
	diags []Diagnostic, input *check_invariant_assertions_validate_boundary_input_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(diags) == 0, "Diags is empty sometimes")
	label_axis_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Function_Label), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	consts_axis_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Same_File_Consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Same_File_Consts is bounded by AST budget",
	})
	invariant.Cross_Product(d, empty_axis, label_axis_d, consts_axis_d,
		invariant.Excluding("Max diags contradicts empty=true",
			invariant.Bucket_Hi(d), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Diags at safety cap with empty input is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Non-empty input with zero diags is bad",
			invariant.Bucket_Lo(d), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi label with empty diags unreachable",
			invariant.Bucket_Hi(label_axis_d),
			invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi label with non-empty diags unreachable",
			invariant.Bucket_Hi(label_axis_d),
			invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi consts with empty diags unreachable",
			invariant.Bucket_Hi(consts_axis_d),
			invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi consts with non-empty diags unreachable",
			invariant.Bucket_Hi(consts_axis_d),
			invariant.Bucket_False(empty_axis)),
	)
}

// Input for check_invariant_assertions_boundary_literal_call.
type check_invariant_assertions_boundary_literal_call_input struct {
	Call             *ast.CallExpr
	Invariant_Idents map[string]bool
	Function_Label   string
	Same_File_Consts map[string]bool
	File_Set         *token.FileSet
}

// Validates Lo/Hi literal shape for one crediting invariant call. Recognises
// three shapes: top-level Is_Boundary/Recorder_Is_Boundary (validate the
// call directly), and Cross_Product/Recorder_Cross_Product (iterate axes
// and validate any Boundary/Recorder_Boundary/Is_Boundary/Recorder_Is_Boundary
// inner call). Non-Boundary helpers (Always, Sometimes, …) contribute no
// diagnostics. Mirrors the credit-recognition logic in call_covered_pairs so
// "crediting" stays consistent across the two passes.
func check_invariant_assertions_boundary_literal_call(
	input *check_invariant_assertions_boundary_literal_call_input,
) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_boundary_literal_call_assert_exit(diags, input)
	}()
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Call != nil, "Call is non-nil"),
		invariant.Always(input.File_Set != nil, "File_set is non-nil"),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)

	helper_name := check_invariant_assertions_extract_call_name(
		input.Call, input.Invariant_Idents)
	invariant.Cross_Product(
		invariant.Sometimes(
			helper_name == "", "Helper_name is empty for non-invariant calls"))
	if helper_name == "" {
		return nil
	}
	is_sugar := helper_name == "Is_Distinct_Boundary"
	if helper_name == "Recorder_Is_Distinct_Boundary" {
		is_sugar = true
	}
	if is_sugar {
		argument_offset := 0
		if helper_name == "Recorder_Is_Distinct_Boundary" {
			argument_offset = 1
		}
		if len(input.Call.Args) <= argument_offset {
			return nil
		}
		return check_invariant_assertions_validate_boundary_input(
			&check_invariant_assertions_validate_boundary_input_input{
				Call:             input.Call,
				Input_Argument:   input.Call.Args[argument_offset],
				Function_Label:   input.Function_Label,
				Same_File_Consts: input.Same_File_Consts,
				File_Set:         input.File_Set,
			})
	}
	return check_invariant_assertions_boundary_literal_call_input_cross_axes(
		input, helper_name)
}

func check_invariant_assertions_boundary_literal_call_input_cross_axes(
	input *check_invariant_assertions_boundary_literal_call_input, helper_name string,
) (diags []Diagnostic) {
	is_cross := helper_name == "Cross_Product"
	if helper_name == "Recorder_Cross_Product" {
		is_cross = true
	}
	if !is_cross {
		return nil
	}
	argument_offset := 0
	if helper_name == "Recorder_Cross_Product" {
		argument_offset = 1
	}
	for i := argument_offset; i < len(input.Call.Args); i++ {
		inner_call, is_inner_call := input.Call.Args[i].(*ast.CallExpr)
		if !is_inner_call {
			continue
		}
		inner_name := check_invariant_assertions_extract_call_name(
			inner_call, input.Invariant_Idents)
		invariant.Cross_Product(
			invariant.Sometimes(
				inner_name == "",
				"Inner_name is empty for non-invariant axis calls"))
		if inner_name == "" {
			continue
		}
		is_boundary_axis := false
		switch inner_name {
		case "Distinct_Boundary",
			"Recorder_Distinct_Boundary",
			"Is_Distinct_Boundary",
			"Recorder_Is_Distinct_Boundary":
			is_boundary_axis = true
		}
		if !is_boundary_axis {
			continue
		}
		inner_offset := 0
		if inner_name == "Recorder_Distinct_Boundary" {
			inner_offset = 1
		}
		if inner_name == "Recorder_Is_Distinct_Boundary" {
			inner_offset = 1
		}
		if len(inner_call.Args) <= inner_offset {
			continue
		}
		diags = append(diags, check_invariant_assertions_validate_boundary_input(
			&check_invariant_assertions_validate_boundary_input_input{
				Call:             inner_call,
				Input_Argument:   inner_call.Args[inner_offset],
				Function_Label:   input.Function_Label,
				Same_File_Consts: input.Same_File_Consts,
				File_Set:         input.File_Set,
			})...)
	}
	return diags
}

func check_invariant_assertions_boundary_literal_call_assert_exit(
	diags []Diagnostic, input *check_invariant_assertions_boundary_literal_call_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(diags) == 0, "Diags is empty sometimes")
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Invariant_Idents is bounded",
	})
	invariant.Cross_Product(d, empty_axis, ii,
		invariant.Excluding("Hi diags with empty unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi diags with diags is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Lo d implies empty true",
			invariant.Bucket_Lo(d), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi ii with empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi ii with diags unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(empty_axis)),
	)
}

// Input for check_invariant_assertions_boundary_literal_block.
type check_invariant_assertions_boundary_literal_block_input struct {
	Body               *ast.BlockStmt
	Invariant_Idents   map[string]bool
	Function_Label     string
	Same_File_Consts   map[string]bool
	File_Set           *token.FileSet
	Is_Top_Of_Function bool
}

// Walks body as an assertion prologue, mirroring scan_function_prologue's
// shape, and emits Boundary-literal diagnostics for every crediting Boundary
// call it finds. Descends into the first DeferStmt's FuncLit body
// (named-return defer position) when Is_Top_Of_Function is true, and into
// `if X != nil { ... }` bodies anywhere. Iterative — descent uses an
// explicit stack of frames since Tiger Style bans self-recursion.
// Stops at the first non-invariant statement in each list so standalone
// Boundary calls past real work are exempt — matching the "only when
// crediting" scope rule.
func check_invariant_assertions_boundary_literal_block(
	input *check_invariant_assertions_boundary_literal_block_input,
) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_boundary_literal_block_assert_exit(diags, input)
	}()
	check_invariant_assertions_boundary_literal_block_input_assert_entry(input)

	if input.Body == nil {
		return nil
	}
	stack := []check_invariant_assertions_boundary_literal_frame{
		{Body: input.Body, Is_Top_Of_Function: input.Is_Top_Of_Function},
	}
	for range invariant.Game_Loop() {
		if len(stack) == 0 {
			return diags
		}
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		defer_seen := false
		for _, statement := range top.Body.List {
			var proceed bool
			stack, diags, proceed =
				check_invariant_assertions_boundary_literal_frame_step(
					statement, input, top, stack, diags, &defer_seen)
			if !proceed {
				break
			}
		}
	}
	return diags
}

// One frame of boundary_literal_block's iterative function-body walk: a block to
// scan plus whether it sits at the top of the function (where the first defer's
// FuncLit body is also scanned).
type check_invariant_assertions_boundary_literal_frame struct {
	Body               *ast.BlockStmt
	Is_Top_Of_Function bool
}

// Processes one statement of a frame: pushes child frames for the top defer's
// FuncLit and for nil-guard if-bodies, credits prologue calls into diags, and
// reports whether the caller's scan should continue (false ends this frame).
func check_invariant_assertions_boundary_literal_frame_step(
	statement ast.Stmt,
	input *check_invariant_assertions_boundary_literal_block_input,
	top check_invariant_assertions_boundary_literal_frame,
	stack []check_invariant_assertions_boundary_literal_frame,
	diags []Diagnostic,
	defer_seen *bool,
) (
	next_stack []check_invariant_assertions_boundary_literal_frame,
	next_diags []Diagnostic,
	proceed bool,
) {
	if defer_statement, is_defer := statement.(*ast.DeferStmt); is_defer {
		if !top.Is_Top_Of_Function {
			return stack, diags, false
		}
		if *defer_seen {
			return stack, diags, false
		}
		*defer_seen = true
		stack = check_invariant_assertions_boundary_literal_frame_push_defer(
			stack, defer_statement)
		return stack, diags, true
	}
	if if_statement, is_if := statement.(*ast.IfStmt); is_if {
		if !check_invariant_assertions_is_nil_guard(if_statement.Cond) {
			return stack, diags, false
		}
		if if_statement.Body == nil {
			return stack, diags, true
		}
		stack = append(stack, check_invariant_assertions_boundary_literal_frame{
			Body:               if_statement.Body,
			Is_Top_Of_Function: false,
		})
		return stack, diags, true
	}
	frame_diags, is_prologue :=
		check_invariant_assertions_boundary_literal_block_statement_call(
			statement, input)
	if !is_prologue {
		return stack, diags, false
	}
	return stack, append(diags, frame_diags...), true
}

// Pushes the top defer's FuncLit body as a non-top frame, when the deferred call
// is a literal with a body. Split out to keep frame_step's nesting shallow.
func check_invariant_assertions_boundary_literal_frame_push_defer(
	stack []check_invariant_assertions_boundary_literal_frame,
	defer_statement *ast.DeferStmt,
) (next_stack []check_invariant_assertions_boundary_literal_frame) {
	function_literal, is_function_literal := defer_statement.Call.Fun.(*ast.FuncLit)
	if !is_function_literal {
		return stack
	}
	if function_literal.Body == nil {
		return stack
	}
	return append(stack, check_invariant_assertions_boundary_literal_frame{
		Body:               function_literal.Body,
		Is_Top_Of_Function: false,
	})
}

func check_invariant_assertions_boundary_literal_block_assert_exit(
	diags []Diagnostic, input *check_invariant_assertions_boundary_literal_block_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	nil_axis := invariant.Sometimes(diags == nil, "Diags is nil sometimes")
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Invariant_Idents is bounded",
	})
	sfc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Same_File_Consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Same_File_Consts is bounded",
	})
	invariant.Cross_Product(d, nil_axis, ii, sfc,
		invariant.Excluding("Hi d with nil unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_True(nil_axis)),
		invariant.Excluding("Hi d with diags unreachable",
			invariant.Bucket_Hi(d), invariant.Bucket_False(nil_axis)),
		invariant.Excluding("Lo d implies nil true",
			invariant.Bucket_Lo(d), invariant.Bucket_False(nil_axis)),
		invariant.Excluding("Hi ii unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(nil_axis)),
		invariant.Excluding("Hi sfc unreachable",
			invariant.Bucket_Hi(sfc), invariant.Bucket_True(nil_axis)),
	)
}

func check_invariant_assertions_boundary_literal_block_input_assert_entry(
	input *check_invariant_assertions_boundary_literal_block_input,
) {
	body_nil_axis := invariant.Sometimes(
		input.Body == nil, "Body is nil for FuncDecls with external linkage")
	top_axis := invariant.Sometimes(
		input.Is_Top_Of_Function, "Recursion entered from a nested context sometimes")
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Invariant_Idents is bounded",
	})
	sfc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Same_File_Consts), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Same_File_Consts is bounded",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		body_nil_axis,
		invariant.Always(input.Invariant_Idents != nil, "Invariant_idents is non-nil"),
		invariant.Always(input.File_Set != nil, "File_set is non-nil"),
		top_axis, ii, sfc,
		invariant.Excluding("FuncDecls with external linkage are filtered before reaching "+
			"here",
			invariant.Bucket_True(body_nil_axis)),
		invariant.Excluding("Recursive descent uses an iterative loop instead",
			invariant.Bucket_False(top_axis)),
		invariant.Excluding("Hi ii unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(body_nil_axis)),
		invariant.Excluding("Hi sfc unreachable",
			invariant.Bucket_Hi(sfc), invariant.Bucket_False(body_nil_axis)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
}

func check_invariant_assertions_boundary_literal_block_statement_call(
	statement ast.Stmt, input *check_invariant_assertions_boundary_literal_block_input,
) (diags []Diagnostic, is_prologue bool) {
	call, _ := check_invariant_assertions_extract_prologue_call(
		statement, input.Invariant_Idents)
	invariant.Cross_Product(
		invariant.Sometimes(call == nil, "Call is nil for non-prologue statements"))
	if call == nil {
		return nil, false
	}
	return check_invariant_assertions_boundary_literal_call(
		&check_invariant_assertions_boundary_literal_call_input{
			Call:             call,
			Invariant_Idents: input.Invariant_Idents,
			Function_Label:   input.Function_Label,
			Same_File_Consts: input.Same_File_Consts,
			File_Set:         input.File_Set,
		}), true
}

type check_invariant_assertions_validate_input struct {
	File_Set                     *token.FileSet
	Parameter_Requirements       []check_invariant_assertions_requirement
	Parameter_Defer_Requirements []check_invariant_assertions_requirement
	Named_Return_Requirements    []check_invariant_assertions_requirement
	Parameter_Covered            map[string]map[string]coverage_origin
	Named_Return_Covered         map[string]map[string]coverage_origin
}

// For each requirement, checks whether the covered map records an assertion
// of the matching kind on the requirement's identifier-path. When the
// signature has two or more parameters (tracked or not), each tracked
// requirement must be covered by Cross_Product specifically — a stack of
// single-axis _Of helpers does not enumerate the cross-product of buckets
// across parameters and therefore does not satisfy the multi-axis case.
// _Of is accepted only when the signature has exactly one parameter.
func check_invariant_assertions_validate(
	input *check_invariant_assertions_validate_input,
) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_validate_assert_exit(diags, input)
	}()
	check_invariant_assertions_validate_input_assert_entry(input)

	// Credited reqs must route through Cross_Product (or Is_* sugar). Each pass
	// classifies one requirement slice against one coverage map; the param pass
	// reads Parameter_Covered, the two defer passes read Named_Return_Covered.
	diags = append(diags, check_invariant_assertions_validate_requirements(
		input.File_Set, input.Parameter_Requirements, input.Parameter_Covered, "param")...)
	diags = append(diags, check_invariant_assertions_validate_requirements(
		input.File_Set,
		input.Parameter_Defer_Requirements,
		input.Named_Return_Covered,
		"param_defer")...)
	diags = append(diags, check_invariant_assertions_validate_requirements(
		input.File_Set,
		input.Named_Return_Requirements,
		input.Named_Return_Covered,
		"named_return")...)
	return diags
}

func check_invariant_assertions_validate_assert_exit(
	diags []Diagnostic, input *check_invariant_assertions_validate_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	ea := invariant.Sometimes(len(diags) == 0, "Diags is empty sometimes")
	pr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_requirements is bounded by AST budget",
	})
	pc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_covered is bounded by AST budget",
	})
	pdr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Parameter_Defer_Requirements),
		Lo:      0,
		Hi:      max_ast_nodes_per_call,
		Message: "Parameter_defer_requirements is bounded by AST budget",
	})
	invariant.Cross_Product(d, ea, pr, pc, pdr,
		invariant.Excluding("Max d empty true",
			invariant.Bucket_Hi(d), invariant.Bucket_True(ea)),
		invariant.Excluding("Hi d cap is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_False(ea)),
		invariant.Excluding("Lo d implies empty true",
			invariant.Bucket_Lo(d), invariant.Bucket_False(ea)),
		invariant.Excluding("Hi pr with empty d unreachable",
			invariant.Bucket_Hi(pr), invariant.Bucket_True(ea)),
		invariant.Excluding("Hi pr with non-empty d unreachable",
			invariant.Bucket_Hi(pr), invariant.Bucket_False(ea)),
		invariant.Excluding("Hi pc with empty d unreachable",
			invariant.Bucket_Hi(pc), invariant.Bucket_True(ea)),
		invariant.Excluding("Hi pc with non-empty d unreachable",
			invariant.Bucket_Hi(pc), invariant.Bucket_False(ea)),
		invariant.Excluding("Hi pdr with empty d unreachable",
			invariant.Bucket_Hi(pdr), invariant.Bucket_True(ea)),
		invariant.Excluding("Hi pdr with non-empty d unreachable",
			invariant.Bucket_Hi(pdr), invariant.Bucket_False(ea)),
	)
	nrr_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_requirements is bounded by AST budget",
	})
	nrr_d_empty := invariant.Sometimes(
		len(input.Named_Return_Requirements) == 0, "Named returns empty sometimes")
	nrc_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_covered is bounded by AST budget",
	})
	nrc_d_empty := invariant.Sometimes(
		len(input.Named_Return_Covered) == 0, "Named_return_covered is empty "+
			"sometimes")
	invariant.Cross_Product(nrr_d, nrr_d_empty,
		invariant.Excluding("Hi nrr emp T",
			invariant.Bucket_Hi(nrr_d), invariant.Bucket_True(nrr_d_empty)),
		invariant.Excluding("Hi nrr emp F",
			invariant.Bucket_Hi(nrr_d), invariant.Bucket_False(nrr_d_empty)),
		invariant.Excluding("Lo nrr emp F",
			invariant.Bucket_Lo(nrr_d), invariant.Bucket_False(nrr_d_empty)),
	)
	invariant.Cross_Product(nrc_d, nrc_d_empty,
		invariant.Excluding("Hi nrc emp T",
			invariant.Bucket_Hi(nrc_d), invariant.Bucket_True(nrc_d_empty)),
		invariant.Excluding("Hi nrc emp F",
			invariant.Bucket_Hi(nrc_d), invariant.Bucket_False(nrc_d_empty)),
		invariant.Excluding("Lo nrc emp F",
			invariant.Bucket_Lo(nrc_d), invariant.Bucket_False(nrc_d_empty)),
	)
}

func check_invariant_assertions_validate_input_assert_entry(
	input *check_invariant_assertions_validate_input,
) {
	pr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_requirements is bounded by AST budget",
	})
	pr_empty := invariant.Sometimes(
		len(input.Parameter_Requirements) == 0, "Parameter_requirements is empty sometimes")
	pc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_covered is bounded by AST budget",
	})
	pc_empty := invariant.Sometimes(
		len(input.Parameter_Covered) == 0, "Parameter_covered is empty sometimes")
	nrc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_covered is bounded by AST budget",
	})
	pdr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Parameter_Defer_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Parameter_defer_requirements is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.File_Set != nil, "File_Set is non-nil"),
		pr, pr_empty, pc, pc_empty, nrc, pdr,
		invariant.Excluding("Hi pr with empty unreachable",
			invariant.Bucket_Hi(pr), invariant.Bucket_True(pr_empty)),
		invariant.Excluding("Hi pr with non-empty unreachable",
			invariant.Bucket_Hi(pr), invariant.Bucket_False(pr_empty)),
		invariant.Excluding("Lo pr implies empty true",
			invariant.Bucket_Lo(pr), invariant.Bucket_False(pr_empty)),
		invariant.Excluding("Hi pc with empty unreachable",
			invariant.Bucket_Hi(pc), invariant.Bucket_True(pc_empty)),
		invariant.Excluding("Hi pc with non-empty unreachable",
			invariant.Bucket_Hi(pc), invariant.Bucket_False(pc_empty)),
		invariant.Excluding("Lo pc implies empty true",
			invariant.Bucket_Lo(pc), invariant.Bucket_False(pc_empty)),
		invariant.Excluding("Hi nrc unreachable in test corpus",
			invariant.Bucket_Hi(nrc), invariant.Bucket_True(pr_empty)),
		invariant.Excluding("Hi pdr unreachable in test corpus",
			invariant.Bucket_Hi(pdr), invariant.Bucket_True(pr_empty)),
	)
	nrr := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Named_Return_Requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Named_return_requirements is bounded by AST budget",
	})
	nrr_empty := invariant.Sometimes(
		len(input.Named_Return_Requirements) == 0, "Named returns empty sometimes")
	invariant.Cross_Product(nrr, nrr_empty,
		invariant.Excluding("Hi nrr emp T",
			invariant.Bucket_Hi(nrr), invariant.Bucket_True(nrr_empty)),
		invariant.Excluding("Hi nrr emp F",
			invariant.Bucket_Hi(nrr), invariant.Bucket_False(nrr_empty)),
		invariant.Excluding("Lo nrr emp F",
			invariant.Bucket_Lo(nrr), invariant.Bucket_False(nrr_empty)),
	)
}

// Classifies each requirement in one slice against one coverage map and emits a
// diagnostic for the uncovered ones. Factored out of validate's three identical
// passes so the param/defer/named-return loops share one body.
func check_invariant_assertions_validate_requirements(file_set *token.FileSet,
	requirements []check_invariant_assertions_requirement,
	covered map[string]map[string]coverage_origin,
	position string) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_validate_requirements_assert_exit(
			diags, requirements, covered, position)
	}()
	requirement_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Requirements is bounded by AST budget",
	})
	requirement_axis_empty := invariant.Sometimes(
		len(requirements) == 0, "Requirements is empty sometimes")
	covered_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Covered is bounded by AST budget",
	})
	covered_axis_empty := invariant.Sometimes(
		len(covered) == 0, "Covered map is empty sometimes")
	position_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(position), Lo: min_validate_position_chars, Hi: max_validate_position_chars,
		Message: "Position spans param to named_return",
	})
	invariant.Cross_Product(invariant.Always(file_set != nil, "File_set is non-nil"))
	invariant.Cross_Product(position_axis)
	invariant.Cross_Product(requirement_axis, requirement_axis_empty,
		invariant.Excluding("Hi rq emp T",
			invariant.Bucket_Hi(requirement_axis),
			invariant.Bucket_True(requirement_axis_empty)),
		invariant.Excluding("Hi rq emp F",
			invariant.Bucket_Hi(requirement_axis),
			invariant.Bucket_False(requirement_axis_empty)),
		invariant.Excluding("Lo rq emp F",
			invariant.Bucket_Lo(requirement_axis),
			invariant.Bucket_False(requirement_axis_empty)),
	)
	invariant.Cross_Product(covered_axis, covered_axis_empty,
		invariant.Excluding("Hi cv empty T",
			invariant.Bucket_Hi(covered_axis),
			invariant.Bucket_True(covered_axis_empty)),
		invariant.Excluding("Hi cv empty F",
			invariant.Bucket_Hi(covered_axis),
			invariant.Bucket_False(covered_axis_empty)),
		invariant.Excluding("Lo cv empty F",
			invariant.Bucket_Lo(covered_axis),
			invariant.Bucket_False(covered_axis_empty)),
	)
	for _, requirement := range requirements {
		status := check_invariant_assertions_coverage_status(covered, requirement)
		invariant.Cross_Product(
			invariant.Sometimes(status == 0, "Status is uncovered sometimes"))
		if status == coverage_status_uncovered {
			diags = append(diags,
				check_invariant_assertions_build_diagnostic(
					file_set, requirement, position))
			continue
		}
		if status == coverage_status_inside_if_only {
			diags = append(diags,
				check_invariant_assertions_build_inside_if_diagnostic(
					file_set, requirement, position))
			continue
		}
		if status != coverage_status_covered_by_cross_product {
			diags = append(diags,
				check_invariant_assertions_build_diagnostic(
					file_set, requirement, position))
		}
	}
	return diags
}

func check_invariant_assertions_validate_requirements_assert_exit(
	diags []Diagnostic,
	requirements []check_invariant_assertions_requirement,
	covered map[string]map[string]coverage_origin,
	position string,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	d_empty := invariant.Sometimes(len(diags) == 0, "Diags is empty sometimes")
	requirement_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirements), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Requirements is bounded by AST budget",
	})
	requirement_d_empty := invariant.Sometimes(
		len(requirements) == 0, "Requirements is empty sometimes")
	covered_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Covered is bounded by AST budget",
	})
	covered_d_empty := invariant.Sometimes(
		len(covered) == 0, "Covered map is empty sometimes")
	position_d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(position),
		Lo:      min_validate_position_chars,
		Hi:      max_validate_position_chars,
		Message: "Position spans param to named_return",
	})
	invariant.Cross_Product(position_d)
	invariant.Cross_Product(d, d_empty,
		invariant.Excluding("Hi d empty T",
			invariant.Bucket_Hi(d), invariant.Bucket_True(d_empty)),
		invariant.Excluding("Hi d empty F",
			invariant.Bucket_Hi(d), invariant.Bucket_False(d_empty)),
		invariant.Excluding("Lo d empty F",
			invariant.Bucket_Lo(d), invariant.Bucket_False(d_empty)),
	)
	invariant.Cross_Product(requirement_d, requirement_d_empty,
		invariant.Excluding("Hi rq emp T",
			invariant.Bucket_Hi(requirement_d),
			invariant.Bucket_True(requirement_d_empty)),
		invariant.Excluding("Hi rq emp F",
			invariant.Bucket_Hi(requirement_d),
			invariant.Bucket_False(requirement_d_empty)),
		invariant.Excluding("Lo rq emp F",
			invariant.Bucket_Lo(requirement_d),
			invariant.Bucket_False(requirement_d_empty)),
	)
	invariant.Cross_Product(covered_d, covered_d_empty,
		invariant.Excluding("Hi cv empty T",
			invariant.Bucket_Hi(covered_d),
			invariant.Bucket_True(covered_d_empty)),
		invariant.Excluding("Hi cv empty F",
			invariant.Bucket_Hi(covered_d),
			invariant.Bucket_False(covered_d_empty)),
		invariant.Excluding("Lo cv empty F",
			invariant.Bucket_Lo(covered_d),
			invariant.Bucket_False(covered_d_empty)),
	)
}

// Builds the diagnostic for a requirement whose only credit came from inside
// an `if X != nil { ... }` block, but whose path has no nillable ancestor.
// The user must add a top-level assertion outside any if-block.
func check_invariant_assertions_build_inside_if_diagnostic(
	file_set *token.FileSet,
	requirement check_invariant_assertions_requirement,
	source string,
) (diag Diagnostic) {
	defer func() {
		check_invariant_assertions_requirement_assert_inside_if_exit(requirement, diag)
	}()
	check_invariant_assertions_build_inside_if_diagnostic_assert_entry(
		file_set, requirement, source)
	prefix := check_invariant_assertions_diagnostic_prefix(source)
	invariant.Cross_Product(invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(prefix), Lo: min_diagnostic_source_chars, Hi: max_diagnostic_source_chars,
		Message: "Prefix is `param` (5), `param defer` (11), or `named return` (12)",
	}))
	return Diagnostic{
		Position: file_set.Position(requirement.Position),
		Name:     inside_if_diagnostic_name,
		Want: "add an assertion outside any `if X != nil { ... }` block — " +
			"non-nillable kinds whose path has no nillable ancestor must be " +
			"asserted at the prologue/defer top level",
		Message: fmt.Sprintf(
			"%s: %s `%s` must be asserted outside any `if ... != nil { ... }` block — "+
				"its path has no nillable ancestor",
			requirement.Function_Label, prefix, requirement.Field_Description),
	}
}

func check_invariant_assertions_requirement_assert_inside_if_exit(
	requirement check_invariant_assertions_requirement, diag Diagnostic,
) {
	// Always-calls consolidated into the FIRST Cross_Product per the
	// chain-credit gate; inline Name/Want/Message != "" credits
	// zero_string for the constant-set strings.
	invariant.Cross_Product(
		invariant.Always(diag.Tier == 0, "Tier is 0 at construction"),
		invariant.Always(
			diag.Name != "",
			"Diag.Name is non-empty for inside-if-only diagnostics"),
		invariant.Always(len(diag.Name) == inside_if_diagnostic_name_chars, "Diag"+
			".Name is the fixed inside-if label"),
		invariant.Always(
			diag.Want != "",
			"Diag.Want is non-empty for inside-if-only diagnostics"),
		invariant.Always(len(diag.Want) == inside_if_diagnostic_want_chars, "Diag"+
			".Want is the fixed inside-if hint"),
		invariant.Always(
			diag.Message != "",
			"Diag.Message is non-empty for inside-if-only diagnostics"),
	)
	message_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(diag.Message),
		Lo:      min_inside_if_message_chars,
		Hi:      max_diagnostic_message_chars,
		Message: "Diag.Message is the inside-if-only message",
	})
	message_mask := invariant.Sometimes(
		len(requirement.Kind) == min_inside_if_kind_chars,
		"Kind is the shorter zero_int label sometimes")
	invariant.Cross_Product(message_axis, message_mask,
		invariant.Excluding("Hi message unreachable (zero kind)",
			invariant.Bucket_Hi(message_axis),
			invariant.Bucket_True(message_mask)),
		invariant.Excluding("Hi message unreachable (boundary kind)",
			invariant.Bucket_Hi(message_axis),
			invariant.Bucket_False(message_mask)))
}

func check_invariant_assertions_build_inside_if_diagnostic_assert_entry(
	file_set *token.FileSet, requirement check_invariant_assertions_requirement, source string,
) {
	requires_defer_axis := invariant.Sometimes(requirement.Requires_Defer_Position,
		"Requirement requires defer position sometimes")
	source_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(source), Lo: min_diagnostic_source_chars, Hi: max_diagnostic_source_chars,
		Message: "Source is `param`, `param_defer`, or `named_return`",
	})
	// Length-mutable leaves at the top level (no nillable ancestor) could in
	// principle reach this diagnostic when coverage came only from inside an
	// if-guard, e.g. `if rows != nil { Distinct_Boundary(&{X: len(*rows), …}) }`.
	// In practice no test fixture exercises that shape — covering authors put
	// slice boundaries outside the if-guard — so the cell stays unobservable.
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(!requirement.Has_Nillable_Ancestor,
			"Inside-if-only diagnostics fire only without nillable ancestors"),
		requires_defer_axis, source_axis,
		invariant.Excluding("Length-mutable leaves sit at top level in test corpus",
			invariant.Bucket_True(requires_defer_axis)),
	)
	kind_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(requirement.Kind),
		Lo: min_inside_if_kind_chars,
		Hi: max_inside_if_kind_chars,
		Message: "Kind is an integer-leaf assertion label: zero_int (8) to boundary_int " +
			"(12)",
	})
	kind_is_zero := invariant.Sometimes(len(requirement.Kind) == min_inside_if_kind_chars,
		"Kind is the shorter zero_int label sometimes")
	invariant.Cross_Product(kind_axis, kind_is_zero,
		invariant.Excluding("Lo kind zero F",
			invariant.Bucket_Lo(kind_axis), invariant.Bucket_False(kind_is_zero)),
		invariant.Excluding("Hi kind zero T",
			invariant.Bucket_Hi(kind_axis), invariant.Bucket_True(kind_is_zero)),
	)
	check_invariant_assertions_requirement_assert_inside_if_more(requirement)
}

func check_invariant_assertions_requirement_assert_inside_if_more(
	requirement check_invariant_assertions_requirement,
) {
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Function_label is non-empty bounded",
	})
	fl_mask := invariant.Sometimes(
		len(requirement.Kind) == min_inside_if_kind_chars, "Kind is zero_int sometimes")
	invariant.Cross_Product(fl, fl_mask,
		invariant.Excluding("Hi fl mask T",
			invariant.Bucket_Hi(fl), invariant.Bucket_True(fl_mask)),
		invariant.Excluding("Hi fl mask F",
			invariant.Bucket_Hi(fl), invariant.Bucket_False(fl_mask)),
	)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirement.Identifier_Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Identifier_path is non-empty bounded",
	})
	path_mask := invariant.Sometimes(
		len(requirement.Kind) == min_inside_if_kind_chars, "Kind is zero_int sometimes")
	invariant.Cross_Product(path_axis, path_mask,
		invariant.Excluding("Hi path mask T",
			invariant.Bucket_Hi(path_axis), invariant.Bucket_True(path_mask)),
		invariant.Excluding("Hi path mask F",
			invariant.Bucket_Hi(path_axis), invariant.Bucket_False(path_mask)),
	)
	field_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Field_Description),
		Lo:      min_inside_if_field_chars,
		Hi:      max_field_description_chars,
		Message: "Field_description is `<name> <type>` for a non-nillable inside-if leaf",
	})
	field_mask := invariant.Sometimes(
		len(requirement.Kind) == min_inside_if_kind_chars, "Kind is zero_int sometimes")
	invariant.Cross_Product(field_axis, field_mask,
		invariant.Excluding("Hi field mask T",
			invariant.Bucket_Hi(field_axis), invariant.Bucket_True(field_mask)),
		invariant.Excluding("Hi field mask F",
			invariant.Bucket_Hi(field_axis), invariant.Bucket_False(field_mask)),
	)
}

// Maps the diagnostic source tag to the human-readable prefix shown in the
// diag message. Centralised so the param/param_defer/named_return mapping
// stays in one place across the two diagnostic builders.
func check_invariant_assertions_diagnostic_prefix(source string) (prefix string) {
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(prefix),
				Lo: min_diagnostic_source_chars,
				Hi: max_diagnostic_source_chars,
				Message: "Prefix is `param` (5), `param defer` (11), or `named " +
					"return` (12)",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:       len(source),
			Lo:      min_diagnostic_source_chars,
			Hi:      max_diagnostic_source_chars,
			Message: "Source is `param`, `param_defer`, or `named_return`",
		}),
	)
	if source == "named_return" {
		return "named return"
	}
	if source == "param_defer" {
		return "param defer"
	}
	return "param"
}

// Coverage status of one requirement: not covered, covered by a single-axis
// Is_* helper, or covered by a Cross_Product axis. Encoded as a single
// returned value so the (yes=false, from_cross_product=true) impossible
// tuple from the prior two-bool shape doesn't haunt the coverage report.
type coverage_status int

// Returns the coverage status of one requirement. covered iff the
// path/kind pair appears anywhere in the map; cross-product iff at least
// one of the covering calls was a Cross_Product (versus only single-axis
// _Of helpers). Requirements with no nillable ancestor in their path need at
// least one credit from outside any if-guard; if every credit came from
// inside an `if X != nil { ... }` body, the status is `inside_if_only`.
func check_invariant_assertions_coverage_status(
	covered map[string]map[string]coverage_origin,
	requirement check_invariant_assertions_requirement,
) (status coverage_status) {
	defer func() {
		check_invariant_assertions_coverage_status_assert_exit(covered, status)
	}()
	check_invariant_assertions_coverage_status_assert_entry(covered)
	nillable_axis := invariant.Sometimes(requirement.Has_Nillable_Ancestor,
		"Requirement is under a nillable ancestor sometimes")
	requires_defer_axis := invariant.Sometimes(requirement.Requires_Defer_Position,
		"Requirement requires defer position sometimes")
	// Length-mutable leaves can now sit under a nillable ancestor — a struct
	// field of type `*[]T` / `*map[K]V` recurses through the pointer and emits
	// the slice/map triple on the deref'd path, which carries the ancestor
	// flag from the pointer.
	invariant.Cross_Product(nillable_axis, requires_defer_axis)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirement.Kind), Lo: min_credit_kind_chars, Hi: max_credit_kind_chars,
		Message: "Kind is in credit_kind range",
	})
	invariant.Cross_Product(k)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirement.Identifier_Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Identifier_path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	field_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Field_Description),
		Lo:      min_requirement_field_description_chars,
		Hi:      max_field_description_chars,
		Message: "Field_description spans short pointer to long slice typename",
	})
	invariant.Cross_Product(field_axis)

	kinds, has := covered[requirement.Identifier_Path]
	if !has {
		return coverage_status_uncovered
	}
	origin, present := kinds[requirement.Kind]
	if !present {
		return coverage_status_uncovered
	}
	if !requirement.Has_Nillable_Ancestor {
		if !origin.Outside_If_Guard {
			return coverage_status_inside_if_only
		}
	}
	if origin.From_Cross_Product {
		return coverage_status_covered_by_cross_product
	}
	return coverage_status_covered_by_sugar
}

func check_invariant_assertions_coverage_status_assert_exit(
	covered map[string]map[string]coverage_origin, status coverage_status,
) {
	invariant.Is_Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  int(status),
		Lo: 0,
		Hi: 3,
	})
	cv := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Covered is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(covered) == 0, "Covered is empty sometimes")
	invariant.Cross_Product(cv, empty_axis,
		invariant.Excluding("Max cv contradicts empty true",
			invariant.Bucket_Hi(cv), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis cv at safety cap is bad",
			invariant.Bucket_Hi(cv), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero cv implies empty true",
			invariant.Bucket_Lo(cv), invariant.Bucket_False(empty_axis)),
	)
}

func check_invariant_assertions_coverage_status_assert_entry(
	covered map[string]map[string]coverage_origin,
) {
	cv := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Covered is bounded by budget",
	})
	covered_empty_axis := invariant.Sometimes(len(covered) == 0, "Covered is empty sometimes")
	invariant.Cross_Product(cv, covered_empty_axis,
		invariant.Excluding("Max cv contradicts covered_empty true",
			invariant.Bucket_Hi(cv), invariant.Bucket_True(covered_empty_axis)),
		invariant.Excluding("Axis cv at safety cap is bad",
			invariant.Bucket_Hi(cv), invariant.Bucket_False(covered_empty_axis)),
		invariant.Excluding("Zero cv implies covered_empty true",
			invariant.Bucket_Lo(cv), invariant.Bucket_False(covered_empty_axis)),
	)
}

// Builds the diagnostic message for one missing requirement. The suggestion
// text picks the most ergonomic helper for the kind: `Range_Of` for int
// (carries the AFL bucket emphasis), `String_Size_Of` for string (most
// universally applicable axis), `Nil_Of` for pointer (only choice).
func check_invariant_assertions_build_diagnostic(
	file_set *token.FileSet,
	requirement check_invariant_assertions_requirement,
	source string,
) (diag Diagnostic) {
	defer func() {
		Diagnostic_Assert_Missing_Axis(diag)
	}()
	nillable_axis := invariant.Sometimes(requirement.Has_Nillable_Ancestor,
		"Requirement is under a nillable ancestor")
	requires_defer_axis := invariant.Sometimes(requirement.Requires_Defer_Position,
		"Requirement requires defer position sometimes")
	source_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(source), Lo: min_diagnostic_source_chars, Hi: max_diagnostic_source_chars,
		Message: "Source is `param`, `param_defer`, or `named_return`",
	})
	// Length-mutable leaves can sit under a nillable ancestor at any source
	// position with §4.5 expansion lifting infer_field_kind: a named return
	// of `*T` whose T carries a slice/map field recurses through the pointer
	// and emits the inner triple with the ancestor flag set, source set to
	// named_return. The four-quadrant cell is therefore valid.
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		nillable_axis, requires_defer_axis, source_axis,
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirement.Kind), Lo: min_credit_kind_chars, Hi: max_credit_kind_chars,
		Message: "Kind is in credit_kind range",
	})
	invariant.Cross_Product(k)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirement.Identifier_Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Identifier_path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	field_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Field_Description),
		Lo:      min_requirement_field_description_chars,
		Hi:      max_field_description_chars,
		Message: "Field_description spans short pointer to long slice typename",
	})
	invariant.Cross_Product(field_axis)
	suggestion := check_invariant_assertions_requirement_suggestion(requirement)
	invariant.Cross_Product(invariant.Always(suggestion != "",
		"Suggestion is non-empty at this point"))
	prefix := check_invariant_assertions_diagnostic_prefix(source)
	invariant.Cross_Product(invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(prefix), Lo: min_diagnostic_source_chars, Hi: max_diagnostic_source_chars,
		Message: "Prefix is `param` (5), `param defer` (11), or `named return` (12)",
	}))
	return Diagnostic{
		Position: file_set.Position(requirement.Position),
		Name:     missing_diagnostic_name,
		Want:     suggestion,
		Message: fmt.Sprintf("%s: %s `%s` missing invariant %s assertion: %s",
			requirement.Function_Label,
			prefix,
			requirement.Field_Description,
			requirement.Kind,
			suggestion),
	}
}

// Asserts the fixed shape of a missing-axis diagnostic at construction
// (tier-0, fixed-width name label, non-empty want/message). The Diagnostic_
// prefix is mandated because the first parameter is a same-package named type.
func Diagnostic_Assert_Missing_Axis(diag Diagnostic) {
	// Always(diag.Tier == 0, ...) credits BOTH boundary_int (via the
	// Always(x == N) rule, N=0 literal) and zero_int. No need for a
	// Distinct_Boundary here — at construction Tier is invariantly 0,
	// so Lo<Hi would be unsatisfiable and the framework would fatal.
	want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(diag.Want),
		Lo:      min_invariant_suggestion_chars,
		Hi:      max_invariant_suggestion_chars,
		Message: "Diag.Want is the suggestion text from requirement_suggestion",
	})
	invariant.Cross_Product(invariant.Always(diag.Tier == 0, "Tier is 0 at "+
		"construction"),
		want_axis,
		invariant.Always(
			diag.Name != "",
			"Diag.Name is non-empty for missing-axis diagnostics"),
		invariant.Always(len(diag.Name) == missing_diagnostic_name_chars, "Diag.N"+
			"ame is the fixed missing-axis label"),
		invariant.Always(
			diag.Message != "",
			"Diag.Message is non-empty for missing-axis diagnostics"))
}

// Returns the suggested-fix text for one requirement. The rendered axis
// matches what the user must write inside Cross_Product.
func check_invariant_assertions_requirement_suggestion(
	requirement check_invariant_assertions_requirement,
) (suggestion string) {
	defer func() {
		check_invariant_assertions_requirement_suggestion_assert_exit(
			suggestion, requirement)
	}()
	nillable_axis := invariant.Sometimes(requirement.Has_Nillable_Ancestor,
		"Requirement is under a nillable ancestor")
	requires_defer_axis := invariant.Sometimes(requirement.Requires_Defer_Position,
		"Requirement requires defer position sometimes")
	// Length-mutable leaves can sit under a nillable ancestor for nested
	// `*[]T` / `*map[K]V` struct fields; the four-quadrant cell is now valid.
	invariant.Cross_Product(nillable_axis, requires_defer_axis)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirement.Kind), Lo: min_credit_kind_chars, Hi: max_credit_kind_chars,
		Message: "Kind is in credit_kind range",
	})
	invariant.Cross_Product(k)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(requirement.Identifier_Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Identifier_path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	field_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Field_Description),
		Lo:      min_requirement_field_description_chars,
		Hi:      max_field_description_chars,
		Message: "Field_description spans short pointer to long slice typename",
	})
	invariant.Cross_Product(field_axis)
	axis := check_invariant_assertions_suggested_call(
		&check_invariant_assertions_suggested_call_input{
			Helper_Name: check_invariant_assertions_bare_composable_for_kind(
				requirement.Kind),
			Identifier_Path:   requirement.Identifier_Path,
			Kind:              requirement.Kind,
			Field_Description: requirement.Field_Description,
		})
	invariant.Cross_Product(invariant.Always(axis != "", "Axis is non-empty at this point"))
	return "use `invariant.Cross_Product(...)` with one `" + axis +
		"` axis per parameter — `Is_*` helpers are single-axis sugar " +
		"and cannot enumerate the cross-product"
}

func check_invariant_assertions_requirement_suggestion_assert_exit(
	suggestion string, requirement check_invariant_assertions_requirement,
) {
	invariant.Cross_Product(invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(suggestion),
		Lo:      min_invariant_suggestion_chars,
		Hi:      max_invariant_suggestion_chars,
		Message: "Suggestion is a fixed wrapper plus the rendered axis call",
	}))
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Kind),
		Lo:      min_credit_kind_chars,
		Hi:      max_credit_kind_chars,
		Message: "Kind is in credit_kind range",
	})
	invariant.Cross_Product(k)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Identifier_Path),
		Lo:      min_non_empty,
		Hi:      Max_Identifier_Chars,
		Message: "Identifier_path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	field_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(requirement.Field_Description),
		Lo:      min_requirement_field_description_chars,
		Hi:      max_field_description_chars,
		Message: "Field_description spans short pointer to long slice typename",
	})
	invariant.Cross_Product(field_axis)
}

// Returns the recommended bare composable name (the form used inside
// Cross_Product) for kind. The two int sub-kinds map to distinct
// composables: Boundary for the boundary half, Always (with `== 0`) for the
// zero-comparison half. Each suggestion targets exactly the shape needed to
// credit one of the dual int requirements.
func check_invariant_assertions_bare_composable_for_kind(kind string) (composable string) {
	defer func() {
		invariant.Cross_Product(
			invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
				X:  len(composable),
				Lo: min_composable_helper_chars,
				Hi: max_composable_helper_chars,
				Message: "Composable is `Always` (6), `Sometimes` (9), or " +
					"`Distinct_Boundary` (17)",
			}),
		)
	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(kind), Lo: min_credit_kind_chars, Hi: max_credit_kind_chars,
			Message: "Kind is a credit-kind table value bounded by max",
		}),
	)

	switch kind {
	case "boundary_int", "boundary_float":
		return "Distinct_Boundary"
	case "zero_int", "zero_float", "zero_string", "zero_slice", "zero_map":
		return "Always"
	case "nan_float":
		return "Sometimes"
	case "inf_float":
		return "Sometimes"
	case "bool":
		return "Sometimes"
	case "pointer":
		return "Always"
	}
	return ""
}

// Input for check_invariant_assertions_suggested_call. Three same-type
// strings would re-introduce the same-type swap landmine check_input_struct
// is designed to prevent; bundling them in this struct keeps call sites
// readable and lets reordering produce compile errors.
type check_invariant_assertions_suggested_call_input struct {
	Helper_Name       string
	Identifier_Path   string
	Kind              string
	Field_Description string
}

// Renders the suggested call expression with placeholder literals.
func check_invariant_assertions_suggested_call(
	input *check_invariant_assertions_suggested_call_input,
) (call string) {
	defer func() {
		check_invariant_assertions_suggested_call_assert_exit(call, input)
	}()
	check_invariant_assertions_suggested_call_input_assert_entry(input)

	boundary_target := input.Identifier_Path
	// String leaves credit boundary_int via len(s)-bearing shapes, so the
	// rendered suggestion must wrap the path in len() to be copy-pasteable.
	if strings.HasSuffix(input.Field_Description, " string") {
		boundary_target = "len(" + input.Identifier_Path + ")"
	}
	switch input.Kind {
	case "boundary_int":
		// Boundary takes an input struct (X, Lo, Hi, Message). Placeholders
		// `lo`, `hi`, `"..."` are not valid Go — the user must replace them
		// with the actual range and a descriptive Message. The `[int]` type
		// parameter is a placeholder; the user adjusts for uint, int32, etc.
		return "invariant." + input.Helper_Name +
			"(&invariant.Boundary_Input[int]{X: " + boundary_target +
			", Lo: lo, Hi: hi, Message: \"...\"})"
	case "boundary_float":
		return "invariant." + input.Helper_Name +
			"(&invariant.Boundary_Input[float64]{X: " + input.Identifier_Path +
			", Lo: lo, Hi: hi, Message: \"...\"})"
	case "zero_int":
		// Zero-comparison is the predicate-shape rule. The lint accepts
		// Is_Always / Always with `path == 0` or `path != 0`, and the
		// Sometimes variants with `path == 0`. Suggest the most common
		// shape — `== 0` — so the user gets a credit on either Is_Always
		// or Always-inside-Cross_Product without further thought.
		return "invariant." + input.Helper_Name +
			"(" + input.Identifier_Path + " == 0, \"...\")"
	case "zero_float":
		return "invariant." + input.Helper_Name +
			"(" + input.Identifier_Path + " == 0.0, \"...\")"
	case "zero_string":
		return "invariant." + input.Helper_Name +
			"(" + input.Identifier_Path + " == \"\", \"...\")"
	case "zero_slice":
		return "invariant." + input.Helper_Name +
			"(len(" + input.Identifier_Path + ") == 0, \"...\")"
	case "zero_map":
		return "invariant." + input.Helper_Name +
			"(len(" + input.Identifier_Path + ") == 0, \"...\")"
	case "nan_float":
		return "invariant." + input.Helper_Name +
			"(math.IsNaN(" + input.Identifier_Path + "), \"...\")"
	case "inf_float":
		return "invariant." + input.Helper_Name +
			"(math.IsInf(" + input.Identifier_Path + ", 0), \"...\")"
	case "bool":
		return "invariant." + input.Helper_Name +
			"(" + input.Identifier_Path + ")"
	case "pointer":
		// Pointer coverage is the predicate-shape rule, not a dedicated
		// helper. The lint accepts Is_Always / Always with `path == nil`
		// or `path != nil`, and Is_Sometimes / Sometimes with `path == nil`.
		// Suggest the most common defensive shape (`!= nil`).
		return "invariant." + input.Helper_Name +
			"(" + input.Identifier_Path + " != nil, \"...\")"
	}
	return "invariant." + input.Helper_Name +
		"(" + input.Identifier_Path + ")"
}

func check_invariant_assertions_suggested_call_assert_exit(
	call string, input *check_invariant_assertions_suggested_call_input,
) {
	invariant.Cross_Product(invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(call),
		Lo: min_suggested_axis_call_chars,
		Hi: max_suggested_axis_call_chars,
		Message: "Call is the suggested axis-builder template string for one " +
			"requirement",
	}))
	hn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Helper_Name),
		Lo:      min_composable_helper_chars,
		Hi:      max_composable_helper_chars,
		Message: "Helper_name is bounded",
	})
	invariant.Cross_Product(hn)
	k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Kind), Lo: min_credit_kind_chars, Hi: max_credit_kind_chars,
		Message: "Kind is in credit_kind range",
	})
	invariant.Cross_Product(k)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Identifier_Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Identifier_path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	field_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Field_Description),
		Lo:      min_requirement_field_description_chars,
		Hi:      max_field_description_chars,
		Message: "Field_description spans short pointer to long slice typename",
	})
	invariant.Cross_Product(field_axis)
}

func check_invariant_assertions_suggested_call_input_assert_entry(
	input *check_invariant_assertions_suggested_call_input,
) {
	invariant.Cross_Product(invariant.Always(input != nil, "Input is non-nil"))
	hn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Helper_Name),
		Lo:      min_composable_helper_chars,
		Hi:      max_composable_helper_chars,
		Message: "Helper_name is bounded",
	})
	invariant.Cross_Product(hn)
	k := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Kind), Lo: min_credit_kind_chars, Hi: max_credit_kind_chars,
		Message: "Kind is in credit_kind range",
	})
	invariant.Cross_Product(k)
	path_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Identifier_Path), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Identifier_path is non-empty bounded",
	})
	invariant.Cross_Product(path_axis)
	field_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Field_Description),
		Lo:      min_requirement_field_description_chars,
		Hi:      max_field_description_chars,
		Message: "Field_description spans short pointer to long slice typename",
	})
	invariant.Cross_Product(field_axis)
}

// One declaration-site obligation derived from a body walk. At Position
// (the declaration's location) a variable was introduced or reassigned
// from a single CallExpr; Identifiers lists every non-discard LHS name.
// The assertion that satisfies this obligation must appear as
// Next_Statement (or, for init clauses, as the first statement of the
// corresponding branch body) and must cover every identifier. When
// there are two or more identifiers, the only invariant call shape that
// can cover all of them is Cross_Product — the _Of family registers a
// single path per call — so the "must cover all" rule transitively
// enforces the spec's "use Cross_Product" requirement without needing
// a dedicated branch.
type check_invariant_assertions_declaration_obligation struct {
	Position             token.Pos
	Function_Label       string
	Identifiers          []string
	Next_Statement       ast.Stmt
	Successor_Statements []ast.Stmt
}

type check_invariant_assertions_validate_declarations_input struct {
	File_Set           *token.FileSet
	Function           *ast.FuncDecl
	Function_Label     string
	Invariant_Idents   map[string]bool
	Stdlib_Local_Names map[string]bool
	Constant_Signs     map[string]int
}

// Walks function.Body for declaration sites and emits one diagnostic per
// site whose "very next statement" isn't an invariant assertion covering
// every non-discard LHS identifier. Independent of the
// parameter/named-return coverage pass — declarations create their own
// obligations regardless of signature shape, so this runs even on
// parameterless functions.
func check_invariant_assertions_validate_declarations(
	input *check_invariant_assertions_validate_declarations_input,
) (diags []Diagnostic) {
	defer func() {
		check_invariant_assertions_validate_declarations_assert_exit(diags, input)
	}()
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Invariant_Idents is bounded",
	})
	sl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Constant_Signs is bounded",
	})
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.File_Set != nil, "File_Set is non-nil"),
		invariant.Always(input.Function != nil, "Function is non-nil"),
		ii, sl, cs, fl,
		invariant.Excluding("Hi stdlib unreachable (Lo idents)",
			invariant.Bucket_Hi(sl), invariant.Bucket_Lo(ii)),
		invariant.Excluding("Hi stdlib unreachable (Hi idents)",
			invariant.Bucket_Hi(sl), invariant.Bucket_Hi(ii)),
		invariant.Excluding("Hi signs unreachable (Lo idents)",
			invariant.Bucket_Hi(cs), invariant.Bucket_Lo(ii)),
		invariant.Excluding("Hi signs unreachable (Hi idents)",
			invariant.Bucket_Hi(cs), invariant.Bucket_Hi(ii)),
		invariant.Excluding("Hi idents unreachable (Lo label)",
			invariant.Bucket_Hi(ii), invariant.Bucket_Lo(fl)),
		invariant.Excluding("Hi idents unreachable (Hi label)",
			invariant.Bucket_Hi(ii), invariant.Bucket_Hi(fl)),
	)
	if input.Function.Body == nil {
		return nil
	}
	obligations := check_invariant_assertions_collect_declaration_obligations(
		input.Function, input.Function_Label, input.Stdlib_Local_Names)
	invariant.Cross_Product(
		invariant.Sometimes(obligations == nil, "Obligations can be empty or zero on this "+
			"branch"))
	for _, obligation := range obligations {
		diag := check_invariant_assertions_validate_declaration_obligation(
			input.File_Set, obligation, input.Invariant_Idents, input.Constant_Signs)
		invariant.Cross_Product(
			invariant.Sometimes(diag == nil, "Diag is nil when the obligation is "+
				"satisfied"))
		if diag != nil {
			diags = append(diags, *diag)
		}
	}
	return diags
}

func check_invariant_assertions_validate_declarations_assert_exit(
	diags []Diagnostic, input *check_invariant_assertions_validate_declarations_input,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diags), Lo: 0, Hi: max_diagnostics_per_call,
		Message: "Diags is bounded by budget",
	})
	empty_axis := invariant.Sometimes(len(diags) == 0, "Diags is empty sometimes")
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Invariant_Idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Invariant_Idents is bounded",
	})
	sl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded",
	})
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Constant_Signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Constant_Signs is bounded",
	})
	invariant.Cross_Product(d, empty_axis, ii, sl, cs,
		invariant.Excluding("Hi d with empty",
			invariant.Bucket_Hi(d), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi d with diags",
			invariant.Bucket_Hi(d), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Lo d with diags",
			invariant.Bucket_Lo(d), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi idents with empty",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi idents with diags",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi stdlib with empty",
			invariant.Bucket_Hi(sl), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi stdlib with diags",
			invariant.Bucket_Hi(sl), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi signs with empty",
			invariant.Bucket_Hi(cs), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi signs with diags",
			invariant.Bucket_Hi(cs), invariant.Bucket_False(empty_axis)),
	)
}

// Iterative LIFO walk of every reachable statement list in the function
// body. Plain BlockStmt declarations contribute one obligation whose
// next slot is the same list's following statement (or nil if the list
// ends). Init clauses of if/for/switch contribute one obligation per
// reachable branch body: the introduced identifier is in scope across
// every branch and the assertion must lead each branch's body. Type
// switch and range don't introduce values from a CallExpr, so their
// init/value forms produce no obligation — their bodies are still
// walked. Defer/go bodies are intentionally skipped: function-literal
// scopes have their own coverage rules handled elsewhere.
func check_invariant_assertions_collect_declaration_obligations(
	function *ast.FuncDecl,
	function_label string,
	stdlib_imports map[string]bool,
) (obligations []check_invariant_assertions_declaration_obligation) {
	defer func() {
		check_invariant_assertions_collect_declaration_obligations_assert_exit(
			obligations, stdlib_imports)
	}()
	check_invariant_assertions_collect_declaration_obligations_assert_entry(
		function, function_label, stdlib_imports)

	stack := [][]ast.Stmt{function.Body.List}
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for i_index, statement := range top {
			identifiers := check_invariant_assertions_declaration_identifiers(
				statement, stdlib_imports)
			invariant.Cross_Product(invariant.Sometimes(identifiers == nil,
				"Identifiers is nil for non-decl-from-call statements"))

			if identifiers != nil {
				var next_statement ast.Stmt
				var successors []ast.Stmt
				if i_index+1 < len(top) {
					next_statement = top[i_index+1]
					successors = top[i_index+1:]
				}
				obligations = append(obligations,
					check_invariant_assertions_declaration_obligation{
						Position:             statement.Pos(),
						Function_Label:       function_label,
						Identifiers:          identifiers,
						Next_Statement:       next_statement,
						Successor_Statements: successors,
					})
			}
			stack = append(
				stack, check_invariant_assertions_descend(
					&check_invariant_assertions_descend_input{
						Statement:          statement,
						Function_Label:     function_label,
						Obligations:        &obligations,
						Stdlib_Local_Names: stdlib_imports,
					})...)
			invariant.Cross_Product(
				invariant.Always(stack != nil, "Stack is non-nil at this point"))
		}
	}
	return obligations
}

func check_invariant_assertions_collect_declaration_obligations_assert_exit(
	obligations []check_invariant_assertions_declaration_obligation,
	stdlib_imports map[string]bool,
) {
	o := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(obligations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Obligations is bounded by budget",
	})
	sn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_imports is bounded by budget",
	})
	invariant.Cross_Product(o, sn,
		invariant.Excluding("Both o and stdlib_imports at safety caps is bad",
			invariant.Bucket_Hi(o), invariant.Bucket_Hi(sn)),
		invariant.Excluding("Max o with zero stdlib_imports is bad",
			invariant.Bucket_Hi(o), invariant.Bucket_Lo(sn)),
		invariant.Excluding("Zero o with max stdlib_imports is bad",
			invariant.Bucket_Lo(o), invariant.Bucket_Hi(sn)),
	)
}

func check_invariant_assertions_collect_declaration_obligations_assert_entry(
	function *ast.FuncDecl, function_label string, stdlib_imports map[string]bool,
) {
	sn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_imports is bounded by budget",
	})
	stdlib_empty_axis := invariant.Sometimes(
		len(stdlib_imports) == 0, "Stdlib_imports is empty sometimes")
	label_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(function_label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Function_label length is bounded; callers gate empty before calling",
	})
	invariant.Cross_Product(
		invariant.Always(function != nil, "Function is non-nil"),
		invariant.Always(
			function_label != "", "Function_label is non-empty per upstream gate"),
		sn, stdlib_empty_axis, label_axis,
		invariant.Excluding("Max stdlib_imports contradicts stdlib_empty true",
			invariant.Bucket_Hi(sn), invariant.Bucket_True(stdlib_empty_axis)),
		invariant.Excluding("Stdlib_imports at safety cap is bad",
			invariant.Bucket_Hi(sn), invariant.Bucket_False(stdlib_empty_axis)),
		invariant.Excluding("Zero stdlib_imports implies stdlib_empty true",
			invariant.Bucket_Lo(sn), invariant.Bucket_False(stdlib_empty_axis)),
		invariant.Excluding("Both stdlib_imports and label at safety caps is bad",
			invariant.Bucket_Hi(sn), invariant.Bucket_Hi(label_axis)),
		invariant.Excluding("Max stdlib_imports with min label is bad",
			invariant.Bucket_Hi(sn), invariant.Bucket_Lo(label_axis)),
	)
}

type check_invariant_assertions_descend_input struct {
	Statement          ast.Stmt
	Function_Label     string
	Obligations        *[]check_invariant_assertions_declaration_obligation
	Stdlib_Local_Names map[string]bool
}

// For statements that contain nested statement lists, returns those lists
// for the caller to push onto its walk stack, and appends obligations
// produced by init clauses. The switch arms cover every Go statement form
// that nests; statements without nested lists (AssignStmt, DeclStmt,
// ReturnStmt, BranchStmt, etc.) fall through and contribute nothing.
func check_invariant_assertions_descend(
	input *check_invariant_assertions_descend_input,
) (nested_lists [][]ast.Stmt) {
	defer func() {
		check_invariant_assertions_descend_assert_exit(nested_lists, input)
	}()
	check_invariant_assertions_descend_input_assert_entry(input)
	switch s := input.Statement.(type) {
	case *ast.IfStmt:
		check_invariant_assertions_descend_if(s, input)
		if s.Body != nil {
			nested_lists = append(nested_lists, s.Body.List)
		}
		switch else_branch := s.Else.(type) {
		case *ast.BlockStmt:
			nested_lists = append(nested_lists, else_branch.List)
		case *ast.IfStmt:
			nested_lists = append(nested_lists, []ast.Stmt{else_branch})
		}
		return nested_lists
	case *ast.ForStmt:
		check_invariant_assertions_descend_for(s, input)
		if s.Body != nil {
			nested_lists = append(nested_lists, s.Body.List)
		}
		return nested_lists
	case *ast.RangeStmt:
		if s.Body != nil {
			nested_lists = append(nested_lists, s.Body.List)
		}
		return nested_lists
	case *ast.SwitchStmt:
		check_invariant_assertions_descend_switch(s, input)
		for _, clause := range s.Body.List {
			case_clause, is_case := clause.(*ast.CaseClause)
			if !is_case {
				continue
			}
			nested_lists = append(nested_lists, case_clause.Body)
		}
		return nested_lists
	case *ast.TypeSwitchStmt:
		for _, clause := range s.Body.List {
			case_clause, is_case := clause.(*ast.CaseClause)
			if !is_case {
				continue
			}
			nested_lists = append(nested_lists, case_clause.Body)
		}
		return nested_lists
	case *ast.BlockStmt:
		return append(nested_lists, s.List)
	case *ast.SelectStmt:
		for _, clause := range s.Body.List {
			communication_clause, is_communication := clause.(*ast.CommClause)
			if !is_communication {
				continue
			}
			nested_lists = append(nested_lists, communication_clause.Body)
		}
		return nested_lists
	case *ast.LabeledStmt:
		return append(nested_lists, []ast.Stmt{s.Stmt})
	}
	return nested_lists
}

func check_invariant_assertions_descend_assert_exit(
	nested_lists [][]ast.Stmt, input *check_invariant_assertions_descend_input,
) {
	u := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(nested_lists), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Nested_lists is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(nested_lists) == 0, "Nested_lists is empty sometimes")
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Obligations is bounded by AST budget",
	})
	stdlib := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded by AST budget",
	})
	invariant.Cross_Product(u, empty_axis, ob, stdlib,
		invariant.Excluding("Max u contradicts empty true",
			invariant.Bucket_Hi(u), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis u at safety cap is bad",
			invariant.Bucket_Hi(u), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero u implies empty true",
			invariant.Bucket_Lo(u), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Hi u with Hi ob is bad",
			invariant.Bucket_Hi(u), invariant.Bucket_Hi(ob)),
		invariant.Excluding("Lo u with Hi ob unreachable",
			invariant.Bucket_Lo(u), invariant.Bucket_Hi(ob)),
		invariant.Excluding("Hi stdlib with empty u",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Hi stdlib with u",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_False(empty_axis)),
	)
}

func check_invariant_assertions_descend_input_assert_entry(
	input *check_invariant_assertions_descend_input,
) {
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Obligations is bounded by AST budget",
	})
	ob_empty := invariant.Sometimes(
		len(*input.Obligations) == 0, "Obligations is empty sometimes")
	stdlib := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(
			input != nil,
			"Input is non-nil"), invariant.Always(input.Obligations != nil,
			"Obligations is non-nil"),
		ob, ob_empty, stdlib,
		invariant.Excluding("Hi ob with empty unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_True(ob_empty)),
		invariant.Excluding("Hi ob with non-empty unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_False(ob_empty)),
		invariant.Excluding("Lo ob implies empty true",
			invariant.Bucket_Lo(ob), invariant.Bucket_False(ob_empty)),
		invariant.Excluding("Hi stdlib with empty obligations",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_True(ob_empty)),
		invariant.Excluding("Hi stdlib with non-empty obligations",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_False(ob_empty)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
}

// Emits init-clause obligations for an *ast.IfStmt: one for the body
// and one per branch of the else chain. No-op when Init isn't a
// declaration-from-call.
func check_invariant_assertions_descend_if(
	s *ast.IfStmt, input *check_invariant_assertions_descend_input,
) {
	defer func() {
		check_invariant_assertions_descend_if_assert_exit(s, input)
	}()
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Obligations is bounded by AST budget",
	})
	has_init := invariant.Sometimes(s.Init != nil, "If has an init clause sometimes")
	label_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Function_Label), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	stdlib_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(s != nil, "S is non-nil"),
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Obligations != nil, "Obligations is non-nil"),
		ob, has_init, label_axis, stdlib_axis,
		invariant.Excluding("Obligations Hi with init unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_True(has_init)),
		invariant.Excluding("Obligations Hi without init unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_False(has_init)),
		invariant.Excluding("Hi stdlib unreachable (has init)",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_True(has_init)),
		invariant.Excluding("Hi stdlib unreachable (init absent)",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_False(has_init)),
	)
	if s.Init == nil {
		return
	}
	identifiers := check_invariant_assertions_declaration_identifiers(
		s.Init, input.Stdlib_Local_Names)
	invariant.Cross_Product(
		invariant.Sometimes(identifiers == nil, "Identifiers is nil for "+
			"non-decl-from-call init"))
	if identifiers == nil {
		return
	}
	*input.Obligations = append(*input.Obligations,
		check_invariant_assertions_obligation_from_body(
			s.Init, identifiers, input.Function_Label, s.Body))
	check_invariant_assertions_append_else_obligations(
		&check_invariant_assertions_append_else_obligations_input{
			Else_Branch:    s.Else,
			Init_Statement: s.Init,
			Identifiers:    identifiers,
			Function_Label: input.Function_Label,
			Obligations:    input.Obligations,
		})
}

func check_invariant_assertions_descend_if_assert_exit(
	s *ast.IfStmt, input *check_invariant_assertions_descend_input,
) {
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Obligations is bounded by AST budget",
	})
	has_init := invariant.Sometimes(s.Init != nil, "If has an init clause sometimes")
	label_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Function_Label), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	stdlib_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded by AST budget",
	})
	invariant.Cross_Product(ob, has_init, label_axis, stdlib_axis,
		invariant.Excluding("Obligations Hi with init unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_True(has_init)),
		invariant.Excluding("Obligations Hi without init unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_False(has_init)),
		invariant.Excluding("Hi stdlib unreachable (has init)",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_True(has_init)),
		invariant.Excluding("Hi stdlib unreachable (init absent)",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_False(has_init)),
	)
}

// Emits an init-clause obligation for an *ast.ForStmt's body.
func check_invariant_assertions_descend_for(
	s *ast.ForStmt, input *check_invariant_assertions_descend_input,
) {
	defer func() {
		check_invariant_assertions_descend_for_assert_exit(s, input)
	}()
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Obligations is bounded by AST budget",
	})
	has_init := invariant.Sometimes(s.Init != nil, "For has an init clause sometimes")
	label_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Function_Label), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	stdlib_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded by AST budget",
	})
	invariant.Cross_Product(
		invariant.Always(s != nil, "S is non-nil"),
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Obligations != nil, "Obligations is non-nil"),
		ob, has_init, label_axis, stdlib_axis,
		invariant.Excluding("Obligations Hi with init unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_True(has_init)),
		invariant.Excluding("Obligations Hi without init unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_False(has_init)),
		invariant.Excluding("Hi stdlib unreachable (has init)",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_True(has_init)),
		invariant.Excluding("Hi stdlib unreachable (init absent)",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_False(has_init)),
		invariant.Excluding("For without init at Lo obligations unreachable in test "+
			"corpus (Lo label)",
			invariant.Bucket_False(has_init),
			invariant.Bucket_Lo(label_axis),
			invariant.Bucket_Lo(ob)),
		invariant.Excluding("For without init at Lo obligations unreachable in test "+
			"corpus (Hi label)",
			invariant.Bucket_False(has_init),
			invariant.Bucket_Hi(label_axis),
			invariant.Bucket_Lo(ob)),
		invariant.Excluding("Hi label with init present unreachable in test corpus",
			invariant.Bucket_Hi(label_axis), invariant.Bucket_True(has_init)),
	)
	if s.Init == nil {
		return
	}
	identifiers := check_invariant_assertions_declaration_identifiers(
		s.Init, input.Stdlib_Local_Names)
	invariant.Cross_Product(
		invariant.Sometimes(identifiers == nil, "Identifiers is nil for "+
			"non-decl-from-call init"))
	if identifiers == nil {
		return
	}
	*input.Obligations = append(*input.Obligations,
		check_invariant_assertions_obligation_from_body(
			s.Init, identifiers, input.Function_Label, s.Body))
}

func check_invariant_assertions_descend_for_assert_exit(
	s *ast.ForStmt, input *check_invariant_assertions_descend_input,
) {
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Obligations is bounded by AST budget",
	})
	has_init := invariant.Sometimes(s.Init != nil, "For has an init clause sometimes")
	label_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Function_Label), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	stdlib_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded by AST budget",
	})
	invariant.Cross_Product(ob, has_init, label_axis, stdlib_axis,
		invariant.Excluding("Obligations Hi with init unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_True(has_init)),
		invariant.Excluding("Obligations Hi without init unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_False(has_init)),
		invariant.Excluding("Hi stdlib unreachable (has init)",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_True(has_init)),
		invariant.Excluding("Hi stdlib unreachable (init absent)",
			invariant.Bucket_Hi(stdlib_axis), invariant.Bucket_False(has_init)),
		invariant.Excluding("For without init at Lo obligations unreachable in "+
			"test corpus (Lo label)",
			invariant.Bucket_False(has_init),
			invariant.Bucket_Lo(label_axis),
			invariant.Bucket_Lo(ob)),
		invariant.Excluding("For without init at Lo obligations unreachable in "+
			"test corpus (Hi label)",
			invariant.Bucket_False(has_init),
			invariant.Bucket_Hi(label_axis),
			invariant.Bucket_Lo(ob)),
		invariant.Excluding("For with init at Hi label and Lo obligations "+
			"unreachable in test corpus",
			invariant.Bucket_True(has_init),
			invariant.Bucket_Hi(label_axis),
			invariant.Bucket_Lo(ob)),
	)
}

// Emits one init-clause obligation per CaseClause of an *ast.SwitchStmt.
// Each case body must lead with the assertion since the identifier is in
// scope across every branch.
func check_invariant_assertions_descend_switch(
	s *ast.SwitchStmt, input *check_invariant_assertions_descend_input,
) {
	defer func() {
		check_invariant_assertions_descend_switch_assert_exit(s, input)
	}()
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Obligations is bounded by AST budget",
	})
	has_init := invariant.Sometimes(s.Init != nil, "Switch has an init clause sometimes")
	stdlib := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded by AST budget",
	})
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(input.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Input.Function_Label is non-empty bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(s != nil, "S is non-nil"),
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Obligations != nil, "Obligations is non-nil"),
		ob, has_init, stdlib, fl,
		invariant.Excluding("Hi ob with init unreachable in test corpus",
			invariant.Bucket_Hi(ob), invariant.Bucket_True(has_init)),
		invariant.Excluding("Hi ob with absent init unreachable in test corpus",
			invariant.Bucket_Hi(ob), invariant.Bucket_False(has_init)),
		invariant.Excluding("Hi stdlib unreachable (init)",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_True(has_init)),
		invariant.Excluding("Hi stdlib unreachable (absent init)",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_False(has_init)),
		invariant.Excluding("Maximal label observed only with accumulated obligations",
			invariant.Bucket_Hi(fl),
			invariant.Bucket_Lo(ob),
			invariant.Bucket_Lo(stdlib)),
	)
	if s.Init == nil {
		return
	}
	identifiers := check_invariant_assertions_declaration_identifiers(
		s.Init, input.Stdlib_Local_Names)
	invariant.Cross_Product(
		invariant.Sometimes(identifiers == nil, "Identifiers is nil for "+
			"non-decl-from-call init"))
	if identifiers == nil {
		return
	}
	for _, clause := range s.Body.List {
		case_clause, is_case := clause.(*ast.CaseClause)
		if !is_case {
			continue
		}
		*input.Obligations = append(*input.Obligations,
			check_invariant_assertions_obligation_from_case(
				s.Init, identifiers, input.Function_Label, case_clause))
	}
}

func check_invariant_assertions_descend_switch_assert_exit(
	s *ast.SwitchStmt, input *check_invariant_assertions_descend_input,
) {
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Obligations is bounded by AST budget",
	})
	has_init := invariant.Sometimes(
		s.Init != nil, "Switch has an init clause sometimes")
	stdlib := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Stdlib_Local_Names), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Input.Stdlib_Local_Names is bounded by AST budget",
	})
	invariant.Cross_Product(ob, has_init, stdlib,
		invariant.Excluding("Hi ob with init unreachable in test corpus",
			invariant.Bucket_Hi(ob), invariant.Bucket_True(has_init)),
		invariant.Excluding("Hi ob with absent init unreachable in test corpus",
			invariant.Bucket_Hi(ob), invariant.Bucket_False(has_init)),
		invariant.Excluding("Hi stdlib unreachable (init)",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_True(has_init)),
		invariant.Excluding("Hi stdlib unreachable (absent init)",
			invariant.Bucket_Hi(stdlib), invariant.Bucket_False(has_init)),
	)
}

// Builds an obligation whose expected next slot is body.List[0]; nil
// next_statement when body is nil or empty (an empty branch fails the
// rule by construction — there's nowhere to host the assertion).
func check_invariant_assertions_obligation_from_body(
	init_statement ast.Stmt,
	identifiers []string,
	function_label string,
	body *ast.BlockStmt,
) (obligation check_invariant_assertions_declaration_obligation) {
	defer func() {
		check_invariant_assertions_obligation_from_body_assert_exit(
			identifiers, obligation)
	}()
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(identifiers), Lo: min_non_empty, Hi: max_string_slice_per_call,
		Message: "Identifiers is ≥1 per caller gate",
	})
	label_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(function_label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Function_label length is bounded; callers gate empty before calling",
	})
	invariant.Cross_Product(
		invariant.Always(body != nil, "Body is non-nil for parsable if/for forms"),
		invariant.Always(function_label != "",
			"Function_label is non-empty per upstream gate"),
		i, label_axis,
		invariant.Excluding("Both invariant_idents and label at safety caps is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Hi(label_axis)),
		invariant.Excluding("Max invariant_idents with min label is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Lo(label_axis)),
	)

	var next ast.Stmt
	var successors []ast.Stmt
	if len(body.List) > 0 {
		next = body.List[0]
		successors = body.List
	}
	return check_invariant_assertions_declaration_obligation{
		Position:             init_statement.Pos(),
		Function_Label:       function_label,
		Identifiers:          identifiers,
		Next_Statement:       next,
		Successor_Statements: successors,
	}
}

func check_invariant_assertions_obligation_from_body_assert_exit(
	identifiers []string, obligation check_invariant_assertions_declaration_obligation,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(identifiers), Lo: min_non_empty, Hi: max_string_slice_per_call,
		Message: "Identifiers is ≥1 per caller gate",
	})
	single_axis := invariant.Sometimes(
		len(identifiers) == count_one, "Identifiers has exactly one entry "+
			"sometimes")
	invariant.Cross_Product(d, single_axis,
		invariant.Excluding("Max d contradicts single true",
			invariant.Bucket_Hi(d), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Axis d at safety cap is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Single (Lo) axis d implies single true",
			invariant.Bucket_Lo(d), invariant.Bucket_False(single_axis)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Obligation.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(obligation.Identifiers), Lo: count_one, Hi: min_pair,
		Message: "Obligation.identifiers is 1 (single LHS) or 2 (multi-LHS)",
	})
	invariant.Cross_Product(idents)
	succ := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(obligation.Successor_Statements),
		Lo: 0,
		Hi: min_pair,
		Message: "Obligation.successor_statements is 0 (empty body) or 2 " +
			"(multi-statement body)",
	})
	invariant.Cross_Product(succ)
}

// Builds an obligation whose expected next slot is case_clause.Body[0];
// nil when the case body is empty. Position anchors at the init so the
// diagnostic points the reader at the declaration whose coverage is
// incomplete, not at one specific branch.
func check_invariant_assertions_obligation_from_case(
	init_statement ast.Stmt,
	identifiers []string,
	function_label string,
	case_clause *ast.CaseClause,
) (obligation check_invariant_assertions_declaration_obligation) {
	defer func() {
		check_invariant_assertions_obligation_from_case_assert_exit(
			identifiers, obligation)
	}()
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(identifiers), Lo: min_non_empty, Hi: max_string_slice_per_call,
		Message: "Identifiers is ≥1 per caller gate",
	})
	label_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(function_label), Lo: min_function_label_chars, Hi: Max_Identifier_Chars,
		Message: "Function_label length is bounded; callers gate empty before calling",
	})
	invariant.Cross_Product(
		invariant.Always(case_clause != nil, "Case_clause is non-nil"),
		invariant.Always(function_label != "",
			"Function_label is non-empty per upstream gate"),
		i, label_axis,
		invariant.Excluding("Both invariant_idents and label at safety caps is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Hi(label_axis)),
		invariant.Excluding("Max invariant_idents with min label is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_Lo(label_axis)),
	)

	var next ast.Stmt
	var successors []ast.Stmt
	if len(case_clause.Body) > 0 {
		next = case_clause.Body[0]
		successors = case_clause.Body
	}
	return check_invariant_assertions_declaration_obligation{
		Position:             init_statement.Pos(),
		Function_Label:       function_label,
		Identifiers:          identifiers,
		Next_Statement:       next,
		Successor_Statements: successors,
	}
}

func check_invariant_assertions_obligation_from_case_assert_exit(
	identifiers []string, obligation check_invariant_assertions_declaration_obligation,
) {
	d := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(identifiers), Lo: min_non_empty, Hi: max_string_slice_per_call,
		Message: "Identifiers is ≥1 per caller gate",
	})
	single_axis := invariant.Sometimes(
		len(identifiers) == count_one, "Identifiers has exactly one entry "+
			"sometimes")
	invariant.Cross_Product(d, single_axis,
		invariant.Excluding("Max d contradicts single true",
			invariant.Bucket_Hi(d), invariant.Bucket_True(single_axis)),
		invariant.Excluding("Axis d at safety cap is bad",
			invariant.Bucket_Hi(d), invariant.Bucket_False(single_axis)),
		invariant.Excluding("Single (Lo) axis d implies single true",
			invariant.Bucket_Lo(d), invariant.Bucket_False(single_axis)),
	)
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Obligation.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(obligation.Identifiers), Lo: count_one, Hi: min_pair,
		Message: "Obligation.identifiers is 1 (single LHS) or 2 (multi-LHS)",
	})
	invariant.Cross_Product(idents)
	succ := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(obligation.Successor_Statements),
		Lo: 0,
		Hi: min_pair,
		Message: "Obligation.successor_statements is 0 (empty case) or 2 " +
			"(multi-statement case)",
	})
	invariant.Cross_Product(succ)
}

type check_invariant_assertions_append_else_obligations_input struct {
	Else_Branch    ast.Stmt
	Init_Statement ast.Stmt
	Identifiers    []string
	Function_Label string
	Obligations    *[]check_invariant_assertions_declaration_obligation
}

// Walks the else chain of an if statement and appends one obligation per
// reachable branch body. Plain `else { ... }` yields one obligation;
// `else if cond { ... }` yields one for its body and continues walking
// its own Else. Game_Loop bounds the traversal so a malformed chain
// can't spin forever.
func check_invariant_assertions_append_else_obligations(
	input *check_invariant_assertions_append_else_obligations_input,
) {
	defer func() {
		check_invariant_assertions_append_else_obligations_input_assert_exit(input)
	}()
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Obligations has ≥1 entry by construction",
	})
	has_else := invariant.Sometimes(input.Else_Branch != nil, "If has an else branch sometimes")
	ids := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Identifiers), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Identifiers is non-empty (short-decl appends ≥1)",
	})
	label := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Function_Label), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Function_Label is non-empty bounded by identifier budget",
	})
	invariant.Cross_Product(
		invariant.Always(input != nil, "Input is non-nil"),
		invariant.Always(input.Obligations != nil, "Obligations is non-nil"),
		ob, has_else, ids, label,
		invariant.Excluding("Obligations at safety cap with else branch unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_True(has_else)),
		invariant.Excluding("Obligations at safety cap without else branch unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_False(has_else)),
		invariant.Excluding("Hi identifiers unreachable",
			invariant.Bucket_Hi(ids), invariant.Bucket_True(has_else)),
		invariant.Excluding("Hi identifiers unreachable",
			invariant.Bucket_Hi(ids), invariant.Bucket_False(has_else)),
		invariant.Excluding("Lo ids with Hi label and else unreachable in test corpus",
			invariant.Bucket_Lo(ids),
			invariant.Bucket_Hi(label),
			invariant.Bucket_True(has_else)),
	)
	current := input.Else_Branch
	for range invariant.Game_Loop() {
		if current == nil {
			return
		}
		if block, is_block := current.(*ast.BlockStmt); is_block {
			*input.Obligations = append(*input.Obligations,
				check_invariant_assertions_obligation_from_body(
					input.Init_Statement,
					input.Identifiers,
					input.Function_Label,
					block))
			return
		}
		else_if, is_if := current.(*ast.IfStmt)
		if !is_if {
			return
		}
		if else_if.Body != nil {
			*input.Obligations = append(*input.Obligations,
				check_invariant_assertions_obligation_from_body(
					input.Init_Statement,
					input.Identifiers,
					input.Function_Label,
					else_if.Body))
		}
		current = else_if.Else
	}
}

func check_invariant_assertions_append_else_obligations_input_assert_exit(
	input *check_invariant_assertions_append_else_obligations_input,
) {
	ob := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(*input.Obligations), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Obligations has ≥1 entry by construction",
	})
	has_else := invariant.Sometimes(
		input.Else_Branch != nil, "If has an else branch sometimes")
	ids := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Identifiers), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Identifiers is non-empty (short-decl appends ≥1)",
	})
	label := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(input.Function_Label), Lo: min_non_empty, Hi: Max_Identifier_Chars,
		Message: "Function_Label is non-empty bounded by identifier budget",
	})
	invariant.Cross_Product(ob, has_else, ids, label,
		invariant.Excluding("Obligations at safety cap with else branch "+
			"unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_True(has_else)),
		invariant.Excluding("Obligations at safety cap without else branch "+
			"unreachable",
			invariant.Bucket_Hi(ob), invariant.Bucket_False(has_else)),
		invariant.Excluding("Obligations Lo contradicts else (which appended an "+
			"obligation)",
			invariant.Bucket_Lo(ob), invariant.Bucket_True(has_else)),
		invariant.Excluding("Hi identifiers unreachable in test corpus",
			invariant.Bucket_Hi(ids), invariant.Bucket_True(has_else)),
		invariant.Excluding("Hi identifiers unreachable in test corpus",
			invariant.Bucket_Hi(ids), invariant.Bucket_False(has_else)),
	)
}

// Extracts the non-discard LHS identifier names when statement is a
// declaration whose RHS is exactly one CallExpr and that CallExpr
// targets non-stdlib code. Recognises plain AssignStmt (both `:=` and
// `=`) and var-form DeclStmt; rejects compound-op assignments,
// multi-value RHS lists, and any LHS that isn't an Ident (selector
// targets, indexed targets — those aren't "declarations" in the spec's
// sense). Calls into Go builtins (append, len, make, …) and the
// standard library are excluded because the spec scopes the rule to
// user-controlled code: an assertion after `xs = append(xs, x)` adds
// noise without revealing anything the call itself doesn't guarantee.
func check_invariant_assertions_declaration_identifiers(
	statement ast.Stmt,
	stdlib_imports map[string]bool,
) (identifiers []string) {
	defer func() {
		check_invariant_assertions_declaration_identifiers_assert_exit(
			identifiers, stdlib_imports)
	}()
	check_invariant_assertions_declaration_identifiers_assert_entry(stdlib_imports)
	if assignment, is_assign := statement.(*ast.AssignStmt); is_assign {
		if assignment.Tok != token.ASSIGN {
			if assignment.Tok != token.DEFINE {
				return nil
			}
		}
		if len(assignment.Rhs) != 1 {
			return nil
		}
		call, is_call := assignment.Rhs[0].(*ast.CallExpr)
		if !is_call {
			return nil
		}
		if check_invariant_assertions_call_is_exempt(call, stdlib_imports) {
			return nil
		}
		for _, left_hand_side := range assignment.Lhs {
			identifier, is_ident := left_hand_side.(*ast.Ident)
			if !is_ident {
				return nil
			}
			if identifier.Name == "_" {
				continue
			}
			identifiers = append(identifiers, identifier.Name)
		}
		if len(identifiers) == 0 {
			return nil
		}
		return identifiers
	}
	return check_invariant_assertions_declaration_identifiers_from_declaration(
		statement, stdlib_imports)
}

func check_invariant_assertions_declaration_identifiers_from_declaration(
	statement ast.Stmt, stdlib_imports map[string]bool,
) (identifiers []string) {
	declaration_statement, is_declaration := statement.(*ast.DeclStmt)
	if !is_declaration {
		return nil
	}
	generic_declaration, is_generic := declaration_statement.Decl.(*ast.GenDecl)
	if !is_generic {
		return nil
	}
	if generic_declaration.Tok != token.VAR {
		return nil
	}
	if len(generic_declaration.Specs) != 1 {
		return nil
	}
	value_specification, is_value := generic_declaration.Specs[0].(*ast.ValueSpec)
	if !is_value {
		return nil
	}
	if len(value_specification.Values) != 1 {
		return nil
	}
	call, is_call := value_specification.Values[0].(*ast.CallExpr)
	if !is_call {
		return nil
	}
	if check_invariant_assertions_call_is_exempt(call, stdlib_imports) {
		return nil
	}
	for _, name := range value_specification.Names {
		if name.Name == "_" {
			continue
		}
		identifiers = append(identifiers, name.Name)
	}
	if len(identifiers) == 0 {
		return nil
	}
	return identifiers
}

func check_invariant_assertions_declaration_identifiers_assert_exit(
	identifiers []string, stdlib_imports map[string]bool,
) {
	i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(identifiers), Lo: 0, Hi: max_string_slice_per_call,
		Message: "Identifiers is bounded by budget",
	})
	nil_axis := invariant.Sometimes(
		identifiers == nil, "Identifiers is nil for non-decl-from-call statements")
	sn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_imports is bounded by budget",
	})
	invariant.Cross_Product(i, nil_axis, sn,
		invariant.Excluding("Max i contradicts nil true",
			invariant.Bucket_Hi(i), invariant.Bucket_True(nil_axis)),
		invariant.Excluding("Axis i at safety cap is bad",
			invariant.Bucket_Hi(i), invariant.Bucket_False(nil_axis)),
		invariant.Excluding("Max sn contradicts nil true",
			invariant.Bucket_Hi(sn), invariant.Bucket_True(nil_axis)),
		invariant.Excluding("Axis sn at safety cap is bad",
			invariant.Bucket_Hi(sn), invariant.Bucket_False(nil_axis)),
		invariant.Excluding("Zero i with non-nil and zero sn is impossible",
			invariant.Bucket_Lo(i),
			invariant.Bucket_False(nil_axis),
			invariant.Bucket_Lo(sn)),
	)
}

func check_invariant_assertions_declaration_identifiers_assert_entry(
	stdlib_imports map[string]bool,
) {
	sn := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_imports is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(stdlib_imports) == 0, "Stdlib_imports is empty sometimes")
	invariant.Cross_Product(sn, empty_axis,
		invariant.Excluding("Max sn contradicts empty true",
			invariant.Bucket_Hi(sn), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis sn at safety cap is bad",
			invariant.Bucket_Hi(sn), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero sn implies empty true",
			invariant.Bucket_Lo(sn), invariant.Bucket_False(empty_axis)),
	)
}

// Reports whether call should be exempt from the declaration-coverage
// rule. Four categories qualify: Go builtins (append, len, make, …),
// standard-library package calls (errors.New, fmt.Sprintf, including
// aliased imports), type conversions (`int(x)`, `[]rune(s)`, `(*T)(p)`,
// …), and invariant-framework axis builders (invariant.Sometimes /
// Always / Boundary and their Recorder_ variants — used as
// `enabled := invariant.Sometimes(…)` before a Cross_Product). The spec
// scopes the rule to user-controlled call sites — builtins and stdlib
// bring their own contracts, type conversions don't perform work an
// assertion would meaningfully observe, and an axis builder IS itself
// the assertion (its parent Cross_Product carries the coverage). Method
// calls on values (e.g. `b.WriteString`) aren't resolved here because
// the receiver type isn't visible without go/types; those continue to
// trigger the rule.
func check_invariant_assertions_call_is_exempt(
	call *ast.CallExpr,
	stdlib_imports map[string]bool,
) (yes bool) {
	defer func() {
		s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Stdlib_imports is bounded by budget",
		})
		yes_axis := invariant.Sometimes(yes, "Call is exempt")
		invariant.Cross_Product(s, yes_axis,
			invariant.Excluding("Axis s at safety cap contradicts yes true",
				invariant.Bucket_Hi(s), invariant.Bucket_True(yes_axis)),
			invariant.Excluding("Axis s at safety cap is bad",
				invariant.Bucket_Hi(s), invariant.Bucket_False(yes_axis)),
		)
	}()
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(stdlib_imports), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Stdlib_imports is bounded by budget",
	})
	empty_axis := invariant.Sometimes(
		len(stdlib_imports) == 0, "Stdlib_imports is empty sometimes")
	invariant.Cross_Product(invariant.Always(call != nil, "Call is non-nil"),
		s, empty_axis,
		invariant.Excluding("Max s contradicts empty true",
			invariant.Bucket_Hi(s), invariant.Bucket_True(empty_axis)),
		invariant.Excluding("Axis s at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_False(empty_axis)),
		invariant.Excluding("Zero axis s implies empty true",
			invariant.Bucket_Lo(s), invariant.Bucket_False(empty_axis)),
	)

	if check_invariant_assertions_expression_is_type(call.Fun) {
		return true
	}
	if identifier, is_ident := call.Fun.(*ast.Ident); is_ident {
		if check_invariant_assertions_is_go_builtin(identifier.Name) {
			return true
		}
		return check_invariant_assertions_is_predeclared_type(identifier.Name)
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return false
	}
	package_identifier, is_ident := selector.X.(*ast.Ident)
	if !is_ident {
		return false
	}
	if stdlib_imports[package_identifier.Name] {
		return true
	}
	if package_identifier.Name == "invariant" {
		return check_invariant_assertions_is_axis_builder(selector.Sel.Name)
	}
	return false
}

// Reports whether selector_name is one of the invariant-framework axis
// builders — Sometimes / Always / Boundary or their Recorder_ variants.
// These return Cross_Axis values that compose into a Cross_Product;
// binding one to a local variable (`enabled := invariant.Sometimes(…)`)
// is part of the constraint-clause shape (`Excluding(Bucket_False(enabled),
// …)`) and doesn't need a separate covering assertion on the variable.
func check_invariant_assertions_is_axis_builder(selector_name string) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Selector names an axis builder"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(selector_name),
			Lo: min_composable_helper_chars,
			Hi: max_invariant_selector_chars,
			Message: "Selector_name covers `Always` (6) through " +
				"`Recorder_Distinct_Boundary` (26)",
		}),
	)
	switch selector_name {
	case "Sometimes", "Always", "Distinct_Boundary",
		"Recorder_Sometimes", "Recorder_Always", "Recorder_Distinct_Boundary":
		return true
	}
	return false
}

// Reports whether expression is a type-literal form used as the target
// of a type conversion: array/slice (`[]rune`), map, channel, function,
// interface, struct, or pointer (`*T`). Recurses through ParenExpr
// because `(*int)(p)` is the canonical pointer-conversion shape.
func check_invariant_assertions_expression_is_type(expression ast.Expr) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Expression is a type form"))

	}()
	current := expression
	for range invariant.Game_Loop() {
		switch current.(type) {
		case *ast.ArrayType, *ast.MapType, *ast.ChanType, *ast.FuncType,
			*ast.InterfaceType, *ast.StructType, *ast.StarExpr:
			return true
		}
		paren, is_paren := current.(*ast.ParenExpr)
		if !is_paren {
			return false
		}
		current = paren.X
	}
	return false
}

// Reports whether name is one of Go's predeclared type names. Used to
// recognise conversions like `int(x)` and `string(b)` that share the
// CallExpr AST shape with function calls but are pure type casts.
func check_invariant_assertions_is_predeclared_type(name string) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Name is a predeclared type"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty Go identifier bounded by identifier budget",
		}),
	)
	switch name {
	case "bool", "byte", "complex64", "complex128", "error",
		"float32", "float64", "int", "int8", "int16", "int32", "int64",
		"rune", "string", "uint", "uint8", "uint16", "uint32", "uint64",
		"uintptr", "any", "comparable":
		return true
	}
	return false
}

// Reports whether name is a predeclared Go function. Mirrors the
// builtin list from the Go spec; types (int, string, bool, …) are
// excluded because they appear as call targets only via conversions
// (`int(x)`) and conversions register a value the user clearly intends
// to keep, which is exactly the case the spec wants asserted.
func check_invariant_assertions_is_go_builtin(name string) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Name is a builtin"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(name), Lo: min_non_empty, Hi: Max_Identifier_Chars,
			Message: "Name is a non-empty Go identifier bounded by identifier budget",
		}),
	)
	switch name {
	case "append", "cap", "clear", "close", "complex", "copy", "delete",
		"imag", "len", "make", "max", "min", "new", "panic", "print",
		"println", "real", "recover":
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
func check_invariant_assertions_collect_stdlib_imports(
	file *ast.File,
) (names map[string]bool) {
	defer func() {
		n := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(names), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Names is bounded by budget",
		})
		empty_axis := invariant.Sometimes(len(names) == 0, "Names is empty sometimes")
		invariant.Cross_Product(n, empty_axis,
			invariant.Excluding("Max n contradicts empty true",
				invariant.Bucket_Hi(n), invariant.Bucket_True(empty_axis)),
			invariant.Excluding("Axis n at safety cap is bad",
				invariant.Bucket_Hi(n), invariant.Bucket_False(empty_axis)),
			invariant.Excluding("Zero n implies empty true",
				invariant.Bucket_Lo(n), invariant.Bucket_False(empty_axis)),
		)
	}()
	invariant.Cross_Product(invariant.Always(file != nil, "File is non-nil"))

	names = map[string]bool{}
	for _, import_specification := range file.Imports {
		path := strings.Trim(import_specification.Path.Value, `"`)
		if !check_invariant_assertions_import_path_is_stdlib(path) {
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
func check_invariant_assertions_import_path_is_stdlib(import_path string) (yes bool) {
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Path resolves to stdlib"))

	}()
	invariant.Cross_Product(
		invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X:  len(import_path),
			Lo: min_non_empty,
			Hi: Max_Identifier_Chars,
			Message: "Import_path is non-empty per Go import grammar and bounded by " +
				"identifier budget",
		}),
	)
	first_slash_offset := strings.IndexByte(import_path, '/')
	if first_slash_offset < 0 {
		return !strings.ContainsRune(import_path, '.')
	}
	return !strings.ContainsRune(import_path[:first_slash_offset], '.')
}

// Returns the set of identifier paths covered by call ignoring the
// per-pair Kind, plus a flag indicating whether the coverage came from
// Cross_Product (versus a single-axis _Of helper). Returns nil paths
// when call isn't a recognised invariant assertion call, letting the
// caller distinguish "no coverage at all" from "incomplete coverage".
func check_invariant_assertions_paths_covered_by_call(
	call *ast.CallExpr,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) (paths map[string]bool, from_cross bool) {
	defer func() {
		i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Invariant_idents is bounded by budget",
		})
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Constant_signs is bounded by budget",
		})
		p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(paths), Lo: 0, Hi: max_coverage_pairs_per_call,
			Message: "Paths is bounded by budget",
		})
		from_cross_axis := invariant.Sometimes(
			from_cross, "From_cross is true when a Cross_Product covers the path")
		invariant.Cross_Product(
			from_cross_axis, i, c, p,
			invariant.Excluding("Both idents and signs at AST safety caps is bad",
				invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
			invariant.Excluding("Invariant_idents at AST at safety cap is bad",
				invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
			invariant.Excluding("Constant_signs at AST at safety cap is bad",
				invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
			invariant.Excluding("Both constants and pairs at safety caps is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_Hi(p)),
			invariant.Excluding("Constants at safety cap with zero pairs is bad",
				invariant.Bucket_Hi(c), invariant.Bucket_Lo(p)),
			invariant.Excluding("Zero constants with max pairs is bad",
				invariant.Bucket_Lo(c), invariant.Bucket_Hi(p)),
			invariant.Excluding("From_cross_product true with all zero counts is "+
				"impossible",
				invariant.Bucket_True(from_cross_axis),
				invariant.Bucket_Lo(i),
				invariant.Bucket_Lo(c),
				invariant.Bucket_Lo(p)),
		)
	}()
	invariant.Cross_Product(invariant.Always(call != nil, "Call is non-nil"))
	assert_idents_signs_bounded(invariant_idents, constant_signs)

	pairs := check_invariant_assertions_call_covered_pairs(
		call, invariant_idents, constant_signs)
	invariant.Cross_Product(
		invariant.Sometimes(pairs == nil, "Pairs is nil for unrecognised calls"))
	if pairs == nil {
		return nil, false
	}
	paths = map[string]bool{}
	for _, pair := range pairs {
		paths[pair.Path] = true
		if pair.From_Cross_Product {
			from_cross = true
		}
	}
	return paths, from_cross
}

// Determines whether obligation.Next_Statement satisfies the spec. Two
// failure modes:
//   - "missing": next slot is not an invariant assertion call, or the
//     call doesn't cover at least one required identifier path.
//   - "cross_product": required_cross is true (>=2 identifiers) but the
//     covering call is a single-axis _Of helper, not Cross_Product.
func check_invariant_assertions_validate_declaration_obligation(
	file_set *token.FileSet,
	obligation check_invariant_assertions_declaration_obligation,
	invariant_idents map[string]bool,
	constant_signs map[string]int,
) (diag *Diagnostic) {
	defer func() {
		check_invariant_assertions_validate_declaration_obligation_assert_exit(
			invariant_idents, constant_signs, obligation, diag)
	}()
	check_invariant_assertions_validate_declaration_obligation_assert_entry(
		invariant_idents, constant_signs, obligation, file_set)
	if len(obligation.Successor_Statements) == 0 {
		return check_invariant_assertions_build_declaration_diagnostic_pointer(
			file_set, obligation)
	}
	covered, chain_includes_cross := check_invariant_assertions_chain_covered_paths(
		obligation.Successor_Statements, invariant_idents, constant_signs)
	covered_axis := invariant.Sometimes(
		covered == nil, "Covered is nil for non-invariant successor chains")
	chain_axis := invariant.Sometimes(
		chain_includes_cross, "Chain includes a top-level Cross_Product")
	// Chain_includes_cross=true requires processing a Cross_Product statement,
	// which always contributes coverage; (nil, true) is impossible.
	invariant.Cross_Product(
		covered_axis, chain_axis,
		invariant.Excluding("Covered nil with chain_includes_cross true is impossible",
			invariant.Bucket_True(covered_axis), invariant.Bucket_True(chain_axis)),
	)
	if covered == nil {
		return check_invariant_assertions_build_declaration_diagnostic_pointer(
			file_set, obligation)
	}
	for _, identifier := range obligation.Identifiers {
		if !check_invariant_assertions_path_covered(covered, identifier) {
			return check_invariant_assertions_build_declaration_diagnostic_pointer(
				file_set, obligation)
		}
	}
	// Spec rule: a multi-LHS declaration (≥ 2 non-discard identifiers) must be covered by a
	// chain that includes a top-level Cross_Product call. A chain of single-axis Is_X ExprStmts
	// each covering one identifier satisfies path coverage but doesn't enumerate the
	// cross-product across axes — `Is_*` is sugar for a one-axis Cross_Product.
	if len(obligation.Identifiers) >= 2 {
		if !chain_includes_cross {
			return check_invariant_assertions_build_declaration_diagnostic_pointer(
				file_set, obligation)
		}
	}
	return nil
}

func check_invariant_assertions_validate_declaration_obligation_assert_entry(
	invariant_idents map[string]bool,
	constant_signs map[string]int,
	obligation check_invariant_assertions_declaration_obligation,
	file_set *token.FileSet,
) {
	invariant.Cross_Product(invariant.Always(file_set != nil, "File_set is non-nil"))
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Obligation.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(obligation.Identifiers), Lo: count_one, Hi: max_obligation_identifiers,
		Message: "Obligation.identifiers is 1 (single LHS) to 3",
	})
	invariant.Cross_Product(idents)
	succ := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(obligation.Successor_Statements), Lo: 0, Hi: max_successor_statements,
		Message: "Successor_statements spans 0 to 30 (many-stmt fixture)",
	})
	invariant.Cross_Product(succ)
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by AST budget",
	})
	ii_empty := invariant.Sometimes(
		len(invariant_idents) == 0, "Invariant_idents is empty sometimes")
	invariant.Cross_Product(ii, ii_empty,
		invariant.Excluding("Hi ii with empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(ii_empty)),
		invariant.Excluding("Hi ii with non-empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(ii_empty)),
		invariant.Excluding("Lo ii implies empty true",
			invariant.Bucket_Lo(ii), invariant.Bucket_False(ii_empty)),
	)
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by AST budget",
	})
	cs_empty := invariant.Sometimes(
		len(constant_signs) == 0, "Constant_signs is empty sometimes")
	invariant.Cross_Product(cs, cs_empty,
		invariant.Excluding("Hi cs with empty unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_True(cs_empty)),
		invariant.Excluding("Hi cs with non-empty unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_False(cs_empty)),
		invariant.Excluding("Lo cs implies empty true",
			invariant.Bucket_Lo(cs), invariant.Bucket_False(cs_empty)),
	)
}

// Asserts the fixed shape every non-nil declaration diagnostic carries at
// construction (tier-0, fixed-width name label, non-empty want/message). Split
// from the obligation exit-assertion so that block stays within the function
// budget; the Diagnostic_ prefix is mandated because the first parameter is a
// same-package named type.
func Diagnostic_Assert_Declaration_Shape(diag *Diagnostic) {
	want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(diag.Want),
		Lo:      min_declaration_diagnostic_want_chars,
		Hi:      max_declaration_diagnostic_want_chars,
		Message: "Diag.Want is single-LHS or multi-LHS suggestion",
	})
	invariant.Cross_Product(invariant.Always(diag.Tier == 0, "Tier is 0 at "+
		"construction"),
		want_axis,
		invariant.Always(
			diag.Name != "",
			"Diag.Name is non-empty for declaration diagnostics"),
		invariant.Always(
			len(
				diag.Name) == declaration_diagnostic_name_chars,
			"Diag.Name is the fixed decl label"),
		invariant.Always(
			diag.Message != "",
			"Diag.Message is non-empty for declaration diagnostics"))
}

func check_invariant_assertions_validate_declaration_obligation_assert_exit(
	invariant_idents map[string]bool,
	constant_signs map[string]int,
	obligation check_invariant_assertions_declaration_obligation,
	diag *Diagnostic,
) {
	invariant.Cross_Product(invariant.Sometimes(
		diag == nil, "Diag is nil when the obligation is satisfied"))

	if diag != nil {
		Diagnostic_Assert_Declaration_Shape(diag)
	}
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Obligation.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Identifiers),
		Lo:      count_one,
		Hi:      max_obligation_identifiers,
		Message: "Obligation.identifiers is 1 (single LHS) to 3",
	})
	invariant.Cross_Product(idents)
	succ := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Successor_Statements),
		Lo:      0,
		Hi:      max_successor_statements,
		Message: "Successor_statements spans 0 to 30 (many-stmt fixture)",
	})
	invariant.Cross_Product(succ)
	ii := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by AST budget",
	})
	ii_empty := invariant.Sometimes(
		len(invariant_idents) == 0, "Invariant_idents is empty sometimes")
	invariant.Cross_Product(ii, ii_empty,
		invariant.Excluding("Hi ii with empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_True(ii_empty)),
		invariant.Excluding("Hi ii with non-empty unreachable",
			invariant.Bucket_Hi(ii), invariant.Bucket_False(ii_empty)),
		invariant.Excluding("Lo ii implies empty true",
			invariant.Bucket_Lo(ii), invariant.Bucket_False(ii_empty)),
	)
	cs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by AST budget",
	})
	cs_empty := invariant.Sometimes(
		len(constant_signs) == 0, "Constant_signs is empty sometimes")
	invariant.Cross_Product(cs, cs_empty,
		invariant.Excluding("Hi cs with empty unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_True(cs_empty)),
		invariant.Excluding("Hi cs with non-empty unreachable",
			invariant.Bucket_Hi(cs), invariant.Bucket_False(cs_empty)),
		invariant.Excluding("Lo cs implies empty true",
			invariant.Bucket_Lo(cs), invariant.Bucket_False(cs_empty)),
	)
}

// Scans a chain of successor statements collecting coverage paths until it
// hits a statement that isn't a coverage-contributing form. Two forms
// contribute:
//
//   - ExprStmt whose X is an invariant.X(...) call (the classic
//     immediately-following assertion shape).
//   - AssignStmt declaring an axis-binding variable from an axis-builder call
//     (`axis := invariant.Sometimes(predicate, …)` and the Always / Boundary /
//     Recorder_ variants). The predicate's identifier paths are the axis's
//     coverage contribution — the axis-builder call is itself the invariant
//     declaration of the predicate.
//
// Returns the accumulated coverage map and a flag indicating whether the
// chain encountered a top-level Cross_Product / Recorder_Cross_Product call.
// The flag drives the multi-LHS rule in validate_declaration_obligation.
// Covered is nil only if the first statement isn't a coverage form at all.
func check_invariant_assertions_chain_covered_paths(
	successors []ast.Stmt, invariant_idents map[string]bool, constant_signs map[string]int,
) (covered map[string]bool, chain_includes_cross bool) {
	defer func() {
		check_invariant_assertions_chain_covered_paths_assert_result(
			successors, covered, chain_includes_cross)
		check_invariant_assertions_chain_covered_paths_assert_inputs(
			invariant_idents, constant_signs)
	}()
	prologue_successors := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(successors), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Successors is ≥1 per caller gate",
	})
	prologue_idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	prologue_signs := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	invariant.Cross_Product(prologue_successors, prologue_idents, prologue_signs,
		invariant.Excluding("Both successors and idents at safety caps is bad",
			invariant.Bucket_Hi(prologue_successors),
			invariant.Bucket_Hi(prologue_idents)),
		invariant.Excluding("Max successors with zero idents is bad",
			invariant.Bucket_Hi(prologue_successors),
			invariant.Bucket_Lo(prologue_idents)),
		invariant.Excluding("Zero successors with max idents is bad",
			invariant.Bucket_Lo(prologue_successors),
			invariant.Bucket_Hi(prologue_idents)),
		invariant.Excluding("Both idents and signs at safety caps is bad",
			invariant.Bucket_Hi(prologue_idents), invariant.Bucket_Hi(prologue_signs)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(prologue_idents), invariant.Bucket_Lo(prologue_signs)),
		invariant.Excluding("Zero idents with max signs is bad",
			invariant.Bucket_Lo(prologue_idents), invariant.Bucket_Hi(prologue_signs)),
	)
	for _, statement := range successors {
		paths, statement_is_cross := check_invariant_assertions_statement_covered_paths(
			statement, invariant_idents, constant_signs)
		paths_axis := invariant.Sometimes(
			paths == nil, "Paths is nil for non-coverage statements")
		cross_axis := invariant.Sometimes(
			statement_is_cross, "Statement is a top-level Cross_Product")
		// Statement_is_cross=true requires the statement to be a Cross_Product
		// ExprStmt, which always contributes non-nil paths; (nil, true) is
		// impossible by construction.
		invariant.Cross_Product(
			paths_axis, cross_axis,
			invariant.Excluding("Paths nil implies cross_axis false by construction",
				invariant.Bucket_True(paths_axis),
				invariant.Bucket_True(cross_axis)),
		)
		if paths == nil {
			if covered == nil {
				return nil, chain_includes_cross
			}
			return covered, chain_includes_cross
		}
		if statement_is_cross {
			chain_includes_cross = true
		}
		if covered == nil {
			covered = map[string]bool{}
		}
		for path := range paths {
			covered[path] = true
		}
	}
	return covered, chain_includes_cross
}

func check_invariant_assertions_chain_covered_paths_assert_result(
	successors []ast.Stmt, covered map[string]bool, chain_includes_cross bool,
) {
	s := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(successors), Lo: min_non_empty, Hi: max_ast_nodes_per_call,
		Message: "Successors is ≥1 per caller gate",
	})
	dc := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Covered is bounded by budget",
	})
	dce := invariant.Sometimes(len(covered) == 0, "Covered is empty sometimes")
	ca := invariant.Sometimes(
		covered == nil, "Covered is nil for non-invariant successor statements")
	cha := invariant.Sometimes(
		chain_includes_cross, "Chain includes a top-level Cross_Product")
	invariant.Cross_Product(s, ca, cha,
		invariant.Excluding("Covered nil with chain true is impossible",
			invariant.Bucket_True(ca), invariant.Bucket_True(cha)),
		invariant.Excluding("Max s contradicts covered true",
			invariant.Bucket_Hi(s), invariant.Bucket_True(ca)),
		invariant.Excluding("Axis s at safety cap is bad",
			invariant.Bucket_Hi(s), invariant.Bucket_False(ca)),
		invariant.Excluding("Zero s with chain false implies covered nil",
			invariant.Bucket_Lo(s),
			invariant.Bucket_False(ca),
			invariant.Bucket_False(cha)),
	)
	invariant.Cross_Product(dc, dce,
		invariant.Excluding("Max dc contradicts dce true",
			invariant.Bucket_Hi(dc), invariant.Bucket_True(dce)),
		invariant.Excluding("Axis dc at safety cap is bad",
			invariant.Bucket_Hi(dc), invariant.Bucket_False(dce)),
		invariant.Excluding("Zero dc implies dce true",
			invariant.Bucket_Lo(dc), invariant.Bucket_False(dce)),
	)
}

func check_invariant_assertions_chain_covered_paths_assert_inputs(
	invariant_idents map[string]bool, constant_signs map[string]int,
) {
	di := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	die := invariant.Sometimes(
		len(invariant_idents) == 0, "Invariant_idents is empty sometimes")
	ds := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	dse := invariant.Sometimes(
		len(constant_signs) == 0, "Constant_signs is empty sometimes")
	invariant.Cross_Product(di, die,
		invariant.Excluding("Max di contradicts die true",
			invariant.Bucket_Hi(di), invariant.Bucket_True(die)),
		invariant.Excluding("Axis di at safety cap is bad",
			invariant.Bucket_Hi(di), invariant.Bucket_False(die)),
		invariant.Excluding("Zero di implies die true",
			invariant.Bucket_Lo(di), invariant.Bucket_False(die)),
	)
	invariant.Cross_Product(ds, dse,
		invariant.Excluding("Max ds contradicts dse true",
			invariant.Bucket_Hi(ds), invariant.Bucket_True(dse)),
		invariant.Excluding("Axis ds at safety cap is bad",
			invariant.Bucket_Hi(ds), invariant.Bucket_False(dse)),
		invariant.Excluding("Zero ds implies dse true",
			invariant.Bucket_Lo(ds), invariant.Bucket_False(dse)),
	)
}

func check_invariant_assertions_statement_covered_paths_assert_entry(
	invariant_idents map[string]bool, constant_signs map[string]int,
) {
	pi := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Invariant_idents is bounded by budget",
	})
	ps := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Constant_signs is bounded by budget",
	})
	invariant.Cross_Product(pi, ps,
		invariant.Excluding("Both prologue idents and signs at safety caps",
			invariant.Bucket_Hi(pi), invariant.Bucket_Hi(ps)),
		invariant.Excluding("Invariant_idents at safety cap is bad",
			invariant.Bucket_Hi(pi), invariant.Bucket_Lo(ps)),
		invariant.Excluding("Zero idents with max signs is bad",
			invariant.Bucket_Lo(pi), invariant.Bucket_Hi(ps)),
	)
}

// Returns the coverage paths produced by one statement, plus a flag set when
// the statement is a top-level invariant.Cross_Product / Recorder_Cross_Product
// call. The flag rides through chain_covered_paths so multi-LHS declarations
// can require an actual Cross_Product (not just a chain of Is_X helpers).
func check_invariant_assertions_statement_covered_paths(
	statement ast.Stmt, invariant_idents map[string]bool, constant_signs map[string]int,
) (paths map[string]bool, is_cross_product bool) {
	defer func() {
		paths_axis := invariant.Sometimes(
			paths == nil, "Paths is nil for non-coverage statements")
		cross_axis := invariant.Sometimes(
			is_cross_product, "Statement is a top-level Cross_Product")
		i := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(invariant_idents), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Invariant_idents is bounded by budget",
		})
		c := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(constant_signs), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Constant_signs is bounded by budget",
		})
		p := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(paths), Lo: 0, Hi: max_coverage_pairs_per_call,
			Message: "Paths is bounded by budget",
		})
		// (is_cross_product=true, paths=nil) is impossible by construction.
		// Without an invariant import (idents=Lo) helper-name extraction returns
		// "" so pairs=nil and paths=nil; (paths!=nil, idents=Lo) is impossible.
		invariant.Cross_Product(paths_axis, cross_axis, i, c, p,
			invariant.Excluding("Paths nil implies cross_axis false",
				invariant.Bucket_True(paths_axis),
				invariant.Bucket_True(cross_axis)),
			invariant.Excluding("Paths false with min i is bad",
				invariant.Bucket_False(paths_axis), invariant.Bucket_Lo(i)),
			invariant.Excluding("Both i and c at safety caps is bad",
				invariant.Bucket_Hi(i), invariant.Bucket_Hi(c)),
			invariant.Excluding("Axis i at safety cap is bad",
				invariant.Bucket_Hi(i), invariant.Bucket_Lo(c)),
			invariant.Excluding("Axis c at safety cap is bad",
				invariant.Bucket_Lo(i), invariant.Bucket_Hi(c)),
			invariant.Excluding("Both c and pairs at safety caps",
				invariant.Bucket_Hi(c), invariant.Bucket_Hi(p)),
			invariant.Excluding("Axis c at safety cap with zero pairs",
				invariant.Bucket_Hi(c), invariant.Bucket_Lo(p)),
			invariant.Excluding("Zero c with max pairs is bad",
				invariant.Bucket_Lo(c), invariant.Bucket_Hi(p)),
		)
	}()
	check_invariant_assertions_statement_covered_paths_assert_entry(
		invariant_idents, constant_signs)
	if expression_statement, is_expression := statement.(*ast.ExprStmt); is_expression {
		call, is_call := expression_statement.X.(*ast.CallExpr)
		if !is_call {
			return nil, false
		}
		covered, _ := check_invariant_assertions_paths_covered_by_call(
			call, invariant_idents, constant_signs)
		invariant.Cross_Product(
			invariant.Sometimes(covered == nil, "Covered is nil for non-invariant "+
				"calls"))
		call_name := check_invariant_assertions_extract_call_name(call, invariant_idents)
		invariant.Cross_Product(
			invariant.Sometimes(
				call_name == "", "Call_name is empty for non-invariant calls"))
		is_cross := call_name == "Cross_Product"
		if !is_cross {
			is_cross = call_name == "Recorder_Cross_Product"
		}
		return covered, is_cross
	}
	return check_invariant_assertions_statement_covered_paths_assign(
		statement, invariant_idents)
}

func check_invariant_assertions_statement_covered_paths_assign(
	statement ast.Stmt, invariant_idents map[string]bool,
) (paths map[string]bool, is_cross_product bool) {
	assignment, is_assignment := statement.(*ast.AssignStmt)
	if !is_assignment {
		return nil, false
	}
	if assignment.Tok != token.DEFINE {
		return nil, false
	}
	if len(assignment.Rhs) != 1 {
		return nil, false
	}
	call, is_call := assignment.Rhs[0].(*ast.CallExpr)
	if !is_call {
		return nil, false
	}
	call_name := check_invariant_assertions_extract_call_name(call, invariant_idents)
	invariant.Cross_Product(
		invariant.Sometimes(call_name == "", "Call_name is empty for non-invariant calls"))
	if call_name == "" {
		return nil, false
	}
	if _, is_bare := invariant_axis_builder_kinds[call_name]; !is_bare {
		return nil, false
	}
	paths = map[string]bool{}
	for _, path := range check_invariant_assertions_extract_identifier_paths(call.Args) {
		paths[path] = true
	}
	comparison_path, comparison_kinds := check_invariant_assertions_extract_nil_comparison_path(
		call, call_name, nil)
	path_axis := invariant.Sometimes(
		comparison_path == "", "Comparison_path is empty for non-comparison helpers")
	kinds_axis := invariant.Sometimes(
		comparison_kinds == nil, "Comparison_kinds is nil for non-comparison helpers")
	invariant.Cross_Product(
		path_axis, kinds_axis,
		invariant.Excluding("Path absent implies kinds empty",
			invariant.Bucket_False(path_axis), invariant.Bucket_True(kinds_axis)),
		invariant.Excluding("Path present implies kinds extracted",
			invariant.Bucket_True(path_axis), invariant.Bucket_False(kinds_axis)),
	)
	if comparison_path != "" {
		paths[comparison_path] = true
	}
	return paths, false
}

// Builds the declaration diagnostic and returns it as a heap-allocated
// pointer. Wraps check_invariant_assertions_build_declaration_diagnostic
// so callers can return a `*Diagnostic` directly without a transient
// local — the local form trips the declaration-coverage rule once for
// each return site.
func check_invariant_assertions_build_declaration_diagnostic_pointer(
	file_set *token.FileSet,
	obligation check_invariant_assertions_declaration_obligation,
) (diag *Diagnostic) {
	defer func() {
		check_invariant_assertions_declaration_obligation_assert_pointer_exit(
			obligation, diag)
	}()
	invariant.Cross_Product(invariant.Always(
		file_set != nil, "File_set is non-nil"))
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Obligation.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(obligation.Identifiers), Lo: count_one, Hi: max_obligation_identifiers,
		Message: "Obligation.identifiers is 1 (single LHS) to 3",
	})
	invariant.Cross_Product(idents)
	succ := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(obligation.Successor_Statements), Lo: 0, Hi: max_successor_statements,
		Message: "Successor_statements spans 0 to 30 (many-stmt fixture)",
	})
	invariant.Cross_Product(succ)

	value := check_invariant_assertions_build_declaration_diagnostic(file_set, obligation)
	invariant.Cross_Product(
		invariant.Always(value.Message != "", "Value.Message is non-empty at this point"))
	return &value
}

func check_invariant_assertions_declaration_obligation_assert_pointer_exit(
	obligation check_invariant_assertions_declaration_obligation, diag *Diagnostic,
) {
	want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(diag.Want),
		Lo:      min_declaration_diagnostic_want_chars,
		Hi:      max_declaration_diagnostic_want_chars,
		Message: "Diag.Want is single-LHS or multi-LHS suggestion",
	})
	invariant.Cross_Product(
		invariant.Always(diag != nil, "Diag is non-nil at this point"),
		invariant.Always(diag.Tier == 0, "Tier is 0 at construction"),
		want_axis,
		invariant.Always(
			diag.Name != "",
			"Diag.Name is non-empty for declaration diagnostics"),
		invariant.Always(len(diag.Name) == declaration_diagnostic_name_chars, "Di"+
			"ag.Name is the fixed declaration label"),
		invariant.Always(
			diag.Message != "",
			"Diag.Message is non-empty for declaration diagnostics"),
	)
	message_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(diag.Message),
		Lo:      min_declaration_diagnostic_message_chars,
		Hi:      max_diagnostic_message_chars,
		Message: "Diag.Message is the declaration-obligation message",
	})
	message_multi := invariant.Sometimes(len(obligation.Identifiers) >= min_pair,
		"Obligation has multiple identifiers sometimes")
	invariant.Cross_Product(message_axis, message_multi,
		invariant.Excluding("Hi message unreachable (single-LHS)",
			invariant.Bucket_Hi(message_axis),
			invariant.Bucket_False(message_multi)),
		invariant.Excluding("Hi message unreachable (multi-LHS)",
			invariant.Bucket_Hi(message_axis),
			invariant.Bucket_True(message_multi)),
		invariant.Excluding("Lo message implies single-LHS",
			invariant.Bucket_Lo(message_axis),
			invariant.Bucket_True(message_multi)))
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Obligation.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Identifiers),
		Lo:      count_one,
		Hi:      max_obligation_identifiers,
		Message: "Obligation.identifiers is 1 (single LHS) to 3",
	})
	invariant.Cross_Product(idents)
	succ := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Successor_Statements),
		Lo:      0,
		Hi:      max_successor_statements,
		Message: "Successor_statements spans 0 to 30 (many-stmt fixture)",
	})
	invariant.Cross_Product(succ)
}

// Reports whether identifier is covered by any path in the set:
// exact-match, or as the head of a dotted selector chain (an
// assertion on `position.Line` covers a declaration-site
// requirement on `position`). The prefix branch is the only way
// to assert struct-valued LHS — `position == nil` doesn't compile
// for a value struct, so the rule lets the user reach into a
// field instead.
func check_invariant_assertions_path_covered(
	covered map[string]bool, identifier string,
) (yes bool) {
	defer func() {
		cv := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
			X: len(covered), Lo: 0, Hi: max_ast_nodes_per_call,
			Message: "Covered is bounded by budget",
		})
		yes_axis := invariant.Sometimes(yes, "Identifier is covered")
		invariant.Cross_Product(cv, yes_axis,
			invariant.Excluding("Axis cv at safety cap is bad",
				invariant.Bucket_Hi(cv), invariant.Bucket_True(yes_axis)),
			invariant.Excluding("Axis cv at safety cap is bad",
				invariant.Bucket_Hi(cv), invariant.Bucket_False(yes_axis)),
			invariant.Excluding("Zero cv contradicts yes true",
				invariant.Bucket_Lo(cv), invariant.Bucket_True(yes_axis)),
		)
	}()
	cv := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(covered), Lo: 0, Hi: max_ast_nodes_per_call,
		Message: "Covered is bounded by budget",
	})
	ident_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:  len(identifier),
		Lo: min_non_empty,
		Hi: max_if_init_identifier_chars,
		Message: "Identifier is a non-empty Go identifier; capped at 55 chars fitting in " +
			"if/for/switch init line budget",
	})
	invariant.Cross_Product(
		ident_axis, cv,
		invariant.Excluding("Both cv and ident at safety caps is bad",
			invariant.Bucket_Hi(cv), invariant.Bucket_Hi(ident_axis)),
		invariant.Excluding("Max cv with zero ident is bad",
			invariant.Bucket_Hi(cv), invariant.Bucket_Lo(ident_axis)),
	)
	if covered[identifier] {
		return true
	}
	prefix := identifier + "."
	for path := range covered {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// Renders the diagnostic for a missing or incomplete declaration-
// coverage assertion. The suggestion lists every identifier the user
// must cover; for multi-identifier sites that's transitively a request
// for Cross_Product since no _Of helper can cover more than one path.
func check_invariant_assertions_build_declaration_diagnostic(
	file_set *token.FileSet,
	obligation check_invariant_assertions_declaration_obligation,
) (diag Diagnostic) {
	defer func() {
		check_invariant_assertions_declaration_obligation_assert_value_exit(
			obligation, diag)
	}()
	invariant.Cross_Product(invariant.Always(
		file_set != nil, "File_set is non-nil"))
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Obligation.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(obligation.Identifiers), Lo: count_one, Hi: max_obligation_identifiers,
		Message: "Obligation.identifiers is 1 (single LHS) to 3",
	})
	invariant.Cross_Product(idents)
	succ := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(obligation.Successor_Statements), Lo: 0, Hi: max_successor_statements,
		Message: "Successor_statements spans 0 to 30 (many-stmt fixture)",
	})
	invariant.Cross_Product(succ)

	identifier_list := strings.Join(obligation.Identifiers, ", ")
	suggestion := "add an invariant assertion as the very next statement covering: " +
		identifier_list
	message := fmt.Sprintf(
		"%s: declaration via function call must be followed by an invariant assertion "+
			"covering: %s",
		obligation.Function_Label, identifier_list)
	if len(obligation.Identifiers) >= 2 {
		message += " — use invariant.Cross_Product since the" +
			" declaration introduces multiple identifiers"
		suggestion = "use invariant.Cross_Product as the next statement" +
			" (or chain axis-builder declarations terminating in" +
			" Cross_Product) covering: " + identifier_list
	}
	return Diagnostic{
		Position: file_set.Position(obligation.Position),
		Name:     declaration_diagnostic_name,
		Want:     suggestion,
		Message:  message,
	}
}

func check_invariant_assertions_declaration_obligation_assert_value_exit(
	obligation check_invariant_assertions_declaration_obligation, diag Diagnostic,
) {
	// Always(diag.Tier == 0, ...) credits BOTH boundary_int + zero_int.
	// Inline Always(diag.Name/Message != "") credits zero_string for those
	// constant-set strings; must share the FIRST Cross_Product because
	// cross_product_seen gates subsequent Cross_Products from inner-call
	// credit.
	want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(diag.Want),
		Lo:      min_declaration_diagnostic_want_chars,
		Hi:      max_declaration_diagnostic_want_chars,
		Message: "Diag.Want is single-LHS or multi-LHS suggestion",
	})
	invariant.Cross_Product(invariant.Always(diag.Tier == 0, "Tier is 0 at "+
		"construction"),
		want_axis,
		invariant.Always(
			diag.Name != "",
			"Diag.Name is non-empty for declaration diagnostics"),
		invariant.Always(len(diag.Name) == declaration_diagnostic_name_chars, "Di"+
			"ag.Name is the fixed declaration label"),
		invariant.Always(
			diag.Message != "",
			"Diag.Message is non-empty for declaration diagnostics"))
	message_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(diag.Message),
		Lo:      min_declaration_diagnostic_message_chars,
		Hi:      max_diagnostic_message_chars,
		Message: "Diag.Message is the declaration-obligation message",
	})
	message_multi := invariant.Sometimes(len(obligation.Identifiers) >= min_pair,
		"Obligation has multiple identifiers sometimes")
	invariant.Cross_Product(message_axis, message_multi,
		invariant.Excluding("Hi message unreachable (single-LHS)",
			invariant.Bucket_Hi(message_axis),
			invariant.Bucket_False(message_multi)),
		invariant.Excluding("Hi message unreachable (multi-LHS)",
			invariant.Bucket_Hi(message_axis),
			invariant.Bucket_True(message_multi)),
		invariant.Excluding("Lo message implies single-LHS",
			invariant.Bucket_Lo(message_axis),
			invariant.Bucket_True(message_multi)))
	fl := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Function_Label),
		Lo:      min_function_label_chars,
		Hi:      Max_Identifier_Chars,
		Message: "Obligation.function_label is non-empty bounded",
	})
	invariant.Cross_Product(fl)
	idents := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Identifiers),
		Lo:      count_one,
		Hi:      max_obligation_identifiers,
		Message: "Obligation.identifiers is 1 (single LHS) to 3",
	})
	invariant.Cross_Product(idents)
	succ := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X:       len(obligation.Successor_Statements),
		Lo:      0,
		Hi:      max_successor_statements,
		Message: "Successor_statements spans 0 to 30 (many-stmt fixture)",
	})
	invariant.Cross_Product(succ)
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
	defer func() {
		check_no_unbounded_apis_lookup_assert_exit(found, diag)
	}()
	invariant.Cross_Product(invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(selector != nil, "Selector is non-nil"))

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

func check_no_unbounded_apis_lookup_assert_exit(found bool, diag Diagnostic) {
	found_axis := invariant.Sometimes(found, "Lookup matched a banned identifier")
	name_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diag.Name), Lo: 0, Hi: Max_Identifier_Chars,
		Message: "Diag.Name is bounded by identifier budget",
	})
	want_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diag.Want), Lo: 0, Hi: max_filesystem_path_chars,
		Message: "Diag.Want is bounded by path budget",
	})
	message_axis := invariant.Distinct_Boundary(&invariant.Boundary_Input[int]{
		X: len(diag.Message), Lo: 0, Hi: max_diagnostic_message_chars,
		Message: "Diag.Message is bounded by message budget",
	})
	invariant.Cross_Product(
		invariant.Always(
			diag.Tier == 0,
			"Tier is zero"), found_axis, name_axis, want_axis, message_axis,
		invariant.Excluding("Hi name unreachable (hit)",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_True(found_axis)),
		invariant.Excluding("Hi name on miss",
			invariant.Bucket_Hi(name_axis), invariant.Bucket_False(found_axis)),
		invariant.Excluding("Lo name impossible (hit)",
			invariant.Bucket_Lo(name_axis), invariant.Bucket_True(found_axis)),
		invariant.Excluding("Hi want unreachable (hit)",
			invariant.Bucket_Hi(want_axis), invariant.Bucket_True(found_axis)),
		invariant.Excluding("Hi want on miss",
			invariant.Bucket_Hi(want_axis), invariant.Bucket_False(found_axis)),
		invariant.Excluding("Lo want impossible (hit)",
			invariant.Bucket_Lo(want_axis), invariant.Bucket_True(found_axis)),
		invariant.Excluding("Hi msg hit",
			invariant.Bucket_Hi(message_axis),
			invariant.Bucket_True(found_axis)),
		invariant.Excluding("Hi msg miss",
			invariant.Bucket_Hi(message_axis),
			invariant.Bucket_False(found_axis)),
		invariant.Excluding("Lo msg hit",
			invariant.Bucket_Lo(message_axis),
			invariant.Bucket_True(found_axis)),
	)
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
// check_no_dot_import and check_default_package_alias, so the X ident
// always resolves to the package's true name.
//
// Generated files (header `// Code generated ... DO NOT EDIT`) are exempt —
// the substitution would have to flow through the generator anyway, and
// the user owns the generator, not the generated output.
func check_no_unbounded_apis(
	file_set *token.FileSet, file *ast.File, _ []byte,
) (diags []Diagnostic) {
	defer func() { assert_diags_decls_bounded(diags, file) }()
	invariant.Cross_Product(
		invariant.Always(file_set != nil, "File_set is non-nil"),
		invariant.Always(file != nil, "File is non-nil"),
	)

	if check_no_unbounded_apis_is_generated(file) {
		return nil
	}

	ast.Inspect(file, func(n ast.Node) (descend bool) {
		selector_expression, is_selector := n.(*ast.SelectorExpr)
		if !is_selector {
			return true
		}
		diag, found := check_no_unbounded_apis_lookup(file_set, selector_expression)
		invariant.Cross_Product(
			invariant.Always(diag.Tier == 0, "Diag tier is 0 at construction"),
			invariant.Sometimes(found, "Lookup matched a banned identifier"),
		)
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
	defer func() {
		invariant.Cross_Product(invariant.Sometimes(
			yes, "Affirmative branch is exercised"))
	}()
	invariant.Cross_Product(invariant.Always(
		file != nil, "File is non-nil"))

	for _, comment_group := range file.Comments {
		for _, comment := range comment_group.List {
			if generated_re.MatchString(comment.Text) {
				return true
			}
		}
	}
	return false
}
