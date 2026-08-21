package glob_test

import (
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/testify"
	"local/james-orcales/shared/text/glob"
	"local/james-orcales/shared/unicode/utf8"
)

// Test_Compile prevents compiled patterns from owning hidden heap state.
func Test_Compile(t *testing.T) {
	var compiled glob.Compile_Workspace
	pattern := compile_from(t, &compiled, []byte("google.com"), nil)
	var match glob.Match_Workspace
	testify.True(t, match_from(t, pattern, []byte("google.com"), &match))
	testify.False(t, match_from(t, pattern, []byte("gobwas.com"), &match))
}

// Test_Wildcards keeps rune consumption distinct from empty consumption.
func Test_Wildcards(t *testing.T) {
	cases := [...]glob_case{
		{true, "", ""},
		{false, "", "a"},
		{true, "a*c", "a12345c"},
		{true, "a?c", "a1c"},
		{false, "a?c", "ac"},
		{true, "?*?", "ac"},
		{true, "*ä", "åä"},
	}
	match_cases(t, cases[:], nil)
}

// Test_Separators keeps ordinary stars inside one path segment.
func Test_Separators(t *testing.T) {
	cases := [...]glob_case{
		{true, "a.*", "a.b"},
		{false, "a.*", "a.b.c"},
		{true, "a.**", "a.b.c"},
		{true, "a.?.c", "a.b.c"},
		{false, "a.?.c", "a.bb.c"},
	}
	separators := [...]rune{'.'}
	match_cases(t, cases[:], separators[:])
}

// Test_Classes keeps class ranges on decoded Unicode characters.
func Test_Classes(t *testing.T) {
	cases := [...]glob_case{
		{true, "[a-z][!a-x]*", "my"},
		{false, "[a-z][!a-x]*", "ma"},
		{true, "[日-語]", "本"},
		{false, "[!日-語]", "本"},
		{true, "[abc]", "b"},
	}
	match_cases(t, cases[:], nil)
}

// Test_Alternatives keeps nested branches bounded without recursion.
func Test_Alternatives(t *testing.T) {
	cases := [...]glob_case{
		{true, "{abc,def}ghi", "defghi"},
		{false, "{abc,def}ghi", "xyzghi"},
		{true, "{a,ab}{bc,f}", "abc"},
		{true, "*//{,*.}example.com", "http://example.com"},
		{true, "*//{,*.}example.com", "https://www.example.com"},
		{true, "{{a,b},c}", "b"},
	}
	match_cases(t, cases[:], nil)
}

// Test_Escape keeps quote output transactional for direct recompilation.
func Test_Escape(t *testing.T) {
	var output [glob.QUOTED_SIZE_MAXIMUM]byte
	count, status := glob.Quote_Meta_Into(output[:], []byte(`{foo*}`))
	testify.Equal_Values(t, glob.STATUS_OK, status)
	testify.Equal(t, `\{foo\*\}`, string(output[:count]))

	var compiled glob.Compile_Workspace
	pattern := compile_from(t, &compiled, output[:count], nil)
	var match glob.Match_Workspace
	testify.True(t, match_from(t, pattern, []byte(`{foo*}`), &match))
}

// Test_Bounds keeps hostile headers outside parser and matcher storage.
func Test_Bounds(t *testing.T) {
	test_formulas(t)
	_, _, status := glob.Compile(glob.Compile_Input{Source: []byte("a")})
	testify.Equal_Values(t, glob.STATUS_WORKSPACE_INVALID, status)

	var compiled glob.Compile_Workspace
	_, diagnostic, status := glob.Compile(compile_input(&compiled, []byte("[abc"), nil))
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, diagnostic.Code)

	var oversized [glob.PATTERN_SIZE_UNVALIDATED_MAXIMUM]byte
	_, _, status = glob.Compile(compile_input(&compiled, oversized[:], nil))
	testify.Equal_Values(t, glob.STATUS_INPUT_INVALID, status)

	pattern := compile_from(t, &compiled, []byte("*"), nil)
	var match glob.Match_Workspace
	var oversized_text [glob.TEXT_SIZE_UNVALIDATED_MAXIMUM]byte
	_, match_status := glob.Match(match_input(pattern, oversized_text[:], &match))
	testify.Equal_Values(t, glob.STATUS_INPUT_INVALID, match_status)

	var destination [len(`\{foo\*\}`) - utf8.CHARACTER_SIZE_MINIMUM]byte
	for index := range destination {
		destination[index] = 'x'
	}
	count, quote_status := glob.Quote_Meta_Into(destination[:], []byte(`{foo*}`))
	testify.Equal_Values(t, glob.STATUS_OUTPUT_TOO_SMALL, quote_status)
	testify.Equal_Values(t, glob.Output_Count(bytes.SLICE_SIZE_MINIMUM), count)
	testify.Equal(t, "xxxxxxxx", string(destination[:]))

	compile_boundaries(t)
	match_boundaries(t, pattern)
	quote_boundaries(t)
	special_boundaries(t)
}

// Test_Allocation prevents compile, match, and quote storage from escaping.
func Test_Allocation(t *testing.T) {
	source := []byte("{abc,def}*.[a-z]")
	text := []byte("defghi.z")
	var compiled glob.Compile_Workspace
	input := compile_input(&compiled, source, nil)
	var pattern glob.Pattern
	var compile_status glob.Compile_Status
	testify.Zero_Allocation(t, func() {
		pattern, _, compile_status = glob.Compile(input)
	})
	testify.Equal_Values(t, glob.STATUS_OK, compile_status)

	var match glob.Match_Workspace
	match_value := glob.Matched(false)
	match_value_input := match_input(pattern, text, &match)
	var match_status glob.Match_Status
	testify.Zero_Allocation(t, func() {
		match_value, match_status = glob.Match(match_value_input)
	})
	testify.Equal_Values(t, glob.STATUS_OK, match_status)
	testify.True(t, bool(match_value))

	var output [glob.QUOTED_SIZE_MAXIMUM]byte
	var count glob.Output_Count
	var quote_status glob.Quote_Status
	testify.Zero_Allocation(t, func() {
		count, quote_status = glob.Quote_Meta_Into(output[:], source)
	})
	testify.Equal_Values(t, glob.STATUS_OK, quote_status)
	testify.Not_Equal(t, glob.Output_Count(bytes.SLICE_SIZE_MINIMUM), count)
}

// Storage formulas must stay attached to repository text boundaries.
func test_formulas(t *testing.T) {
	testify.Equal(t, bytes.SLICE_SIZE_MAXIMUM, glob.PATTERN_SIZE_MAXIMUM)
	testify.Equal(t, bytes.SLICE_SIZE_MAXIMUM, glob.TEXT_SIZE_MAXIMUM)
	testify.Equal(t,
		glob.PATTERN_SIZE_MAXIMUM+utf8.CHARACTER_SIZE_MINIMUM,
		glob.PATTERN_SIZE_UNVALIDATED_MAXIMUM,
	)
	testify.Equal(t, glob.PATTERN_SIZE_MAXIMUM/len("{}"), glob.PATTERN_DEPTH_MAXIMUM)
	testify.Equal(t,
		glob.PATTERN_SIZE_MAXIMUM+utf8.CHARACTER_SIZE_MINIMUM,
		glob.NODE_COUNT_MAXIMUM,
	)
	testify.Equal(t,
		glob.PATTERN_SIZE_MAXIMUM-len("[]"),
		glob.CLASS_RANGE_COUNT_MAXIMUM,
	)
	testify.Equal(t,
		glob.PATTERN_SIZE_MAXIMUM-len("["),
		glob.CLASS_RANGE_STORAGE_COUNT_MAXIMUM,
	)
	testify.Equal(t,
		glob.PATTERN_SIZE_MAXIMUM+utf8.CHARACTER_SIZE_MINIMUM,
		glob.INSTRUCTION_COUNT_MAXIMUM,
	)
	testify.Equal(t, glob.PATTERN_SIZE_MAXIMUM*len(`\x`), glob.QUOTED_SIZE_MAXIMUM)
}

type glob_case struct {
	Matched bool
	Pattern string
	Text    string
}

func match_cases(t *testing.T, cases []glob_case, separators []rune) {
	t.Helper()
	for index := range cases {
		var compiled glob.Compile_Workspace
		pattern := compile_from(
			t, &compiled, []byte(cases[index].Pattern), separators,
		)
		var match glob.Match_Workspace
		matched := match_from(t, pattern, []byte(cases[index].Text), &match)
		testify.Equal(t, cases[index].Matched, matched)
	}
}

func compile_from(
	t *testing.T,
	workspace *glob.Compile_Workspace,
	source []byte,
	separators []rune,
) (pattern glob.Pattern) {
	t.Helper()
	pattern, _, status := glob.Compile(compile_input(workspace, source, separators))
	testify.Equal_Values(t, glob.STATUS_OK, status)
	return pattern
}

func compile_input(
	workspace *glob.Compile_Workspace,
	source []byte,
	separators []rune,
) (input glob.Compile_Input) {
	input.Source = source
	input.Separators = separators
	input.Workspace.State[glob.WORKSPACE_FIELD] = workspace
	return input
}

func match_from(
	t *testing.T,
	pattern glob.Pattern,
	text []byte,
	workspace *glob.Match_Workspace,
) (matched bool) {
	t.Helper()
	value, status := glob.Match(match_input(pattern, text, workspace))
	testify.Equal_Values(t, glob.STATUS_OK, status)
	return bool(value)
}

func match_input(
	pattern glob.Pattern,
	text []byte,
	workspace *glob.Match_Workspace,
) (input glob.Match_Input) {
	input.Pattern = pattern
	input.Text = text
	input.Workspace.State[glob.WORKSPACE_FIELD] = workspace
	return input
}

func compile_boundaries(t *testing.T) {
	var workspace glob.Compile_Workspace
	patterns := [...][]byte{
		[]byte("ab"),
		[]byte("\\"),
		[]byte("["),
		[]byte("[a"),
		[]byte("[\\"),
		[]byte("a[a]"),
		[]byte("aa[a]"),
		[]byte("**"),
		[]byte("{,"),
	}
	for index := range patterns {
		glob.Compile(compile_input(&workspace, patterns[index], nil))
	}

	var source [glob.PATTERN_SIZE_MAXIMUM]byte
	for index := range source {
		source[index] = 'a'
	}
	source[len(source)-utf8.CHARACTER_SIZE_MINIMUM] = '\\'
	_, diagnostic, status := glob.Compile(
		compile_input(&workspace, source[:], nil),
	)
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)
	testify.Equal_Values(
		t, glob.Diagnostic_Position(glob.PATTERN_SIZE_MAXIMUM),
		diagnostic.Position,
	)
	for index := range source {
		source[index] = 'a'
	}
	source[len(source)-utf8.CHARACTER_SIZE_MINIMUM] = '['
	_, _, status = glob.Compile(compile_input(&workspace, source[:], nil))
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)

	separators_two := [...]rune{'.', '/'}
	pattern, _, status := glob.Compile(
		compile_input(&workspace, nil, separators_two[:]),
	)
	testify.Equal_Values(t, glob.STATUS_OK, status)
	var match glob.Match_Workspace
	testify.True(t, match_from(t, pattern, nil, &match))
	var separators_maximum [glob.SEPARATOR_COUNT_MAXIMUM]rune
	pattern, _, status = glob.Compile(
		compile_input(&workspace, nil, separators_maximum[:]),
	)
	testify.Equal_Values(t, glob.STATUS_OK, status)
	testify.True(t, match_from(t, pattern, nil, &match))
	var separators_oversized [glob.SEPARATOR_COUNT_UNVALIDATED_MAXIMUM]rune
	_, _, status = glob.Compile(
		compile_input(&workspace, nil, separators_oversized[:]),
	)
	testify.Equal_Values(t, glob.STATUS_INPUT_INVALID, status)

	glob.Compile(compile_input(&workspace, nil, nil))

	compile_valid_maximum(t, &workspace)
	compile_nesting_boundaries(t, &workspace)
	compile_class_boundaries(t, &workspace)
	compile_character_boundaries(t, &workspace)
}

func compile_valid_maximum(t *testing.T, workspace *glob.Compile_Workspace) {
	var source [glob.PATTERN_SIZE_MAXIMUM]byte
	for index := range source {
		source[index] = 'a'
	}
	pattern, _, status := glob.Compile(compile_input(workspace, source[:], nil))
	testify.Equal_Values(t, glob.STATUS_OK, status)
	var match glob.Match_Workspace
	testify.False(t, match_from(t, pattern, nil, &match))
}

func compile_nesting_boundaries(t *testing.T, workspace *glob.Compile_Workspace) {
	var nested [glob.PATTERN_SIZE_MAXIMUM]byte
	for index := bytes.SLICE_SIZE_MINIMUM; index < glob.PATTERN_DEPTH_MAXIMUM; index++ {
		nested[index] = '{'
		nested[len(nested)-index-utf8.CHARACTER_SIZE_MINIMUM] = '}'
	}
	_, _, status := glob.Compile(compile_input(workspace, nested[:], nil))
	testify.Equal_Values(t, glob.STATUS_OK, status)

	var exhausted [glob.PATTERN_DEPTH_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM]byte
	for index := bytes.SLICE_SIZE_MINIMUM; index < glob.PATTERN_DEPTH_MAXIMUM; index++ {
		exhausted[index] = '{'
	}
	exhausted[len(exhausted)-utf8.CHARACTER_SIZE_MINIMUM] = 'a'
	_, _, status = glob.Compile(compile_input(workspace, exhausted[:], nil))
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)
	exhausted[len(exhausted)-utf8.CHARACTER_SIZE_MINIMUM] = '*'
	_, _, status = glob.Compile(compile_input(workspace, exhausted[:], nil))
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)

	var too_deep [glob.PATTERN_DEPTH_MAXIMUM + utf8.CHARACTER_SIZE_MINIMUM]byte
	for index := range too_deep {
		too_deep[index] = '{'
	}
	_, _, status = glob.Compile(compile_input(workspace, too_deep[:], nil))
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)

	branch_count := utf8.CHARACTER_SIZE_THREE
	open_count := glob.PATTERN_DEPTH_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM
	branch_source := nested[:open_count+branch_count]
	for index := bytes.SLICE_SIZE_MINIMUM; index < open_count; index++ {
		branch_source[index] = '{'
	}
	for index := open_count; index < len(branch_source); index++ {
		branch_source[index] = ','
	}
	_, _, status = glob.Compile(compile_input(workspace, branch_source, nil))
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)

	branch_at_limit := nested[:glob.PATTERN_DEPTH_MAXIMUM+utf8.CHARACTER_SIZE_MINIMUM]
	for index := bytes.SLICE_SIZE_MINIMUM; index < glob.PATTERN_DEPTH_MAXIMUM; index++ {
		branch_at_limit[index] = '{'
	}
	branch_at_limit[len(branch_at_limit)-utf8.CHARACTER_SIZE_MINIMUM] = ','
	_, _, status = glob.Compile(compile_input(workspace, branch_at_limit, nil))
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)
}

func compile_class_boundaries(t *testing.T, workspace *glob.Compile_Workspace) {
	var source [glob.PATTERN_SIZE_MAXIMUM]byte
	source[bytes.SLICE_SIZE_MINIMUM] = '['
	class_end := len(source) - utf8.CHARACTER_SIZE_MINIMUM
	for index := utf8.CHARACTER_SIZE_MINIMUM; index < class_end; index++ {
		source[index] = 'a'
	}
	source[class_end] = ']'
	pattern, _, status := glob.Compile(compile_input(workspace, source[:], nil))
	testify.Equal_Values(t, glob.STATUS_OK, status)
	var match glob.Match_Workspace
	testify.True(t, match_from(t, pattern, []byte("a"), &match))
	source[class_end] = 'a'
	_, _, status = glob.Compile(compile_input(workspace, source[:], nil))
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)

	open_count := glob.PATTERN_DEPTH_MAXIMUM
	deep_source := source[:open_count+len("[a]")]
	for index := bytes.SLICE_SIZE_MINIMUM; index < open_count; index++ {
		deep_source[index] = '{'
	}
	copy(deep_source[open_count:], []byte("[a]"))
	_, _, status = glob.Compile(compile_input(workspace, deep_source, nil))
	testify.Equal_Values(t, glob.STATUS_SYNTAX_INVALID, status)
}

func compile_character_boundaries(t *testing.T, workspace *glob.Compile_Workspace) {
	var encoded [utf8.CHARACTER_SIZE_MAXIMUM]byte
	encoded_size := utf8.Encode_Character(
		encoded[:], utf8.Character(utf8.DECODED_CHARACTER_MAXIMUM),
	)
	literals := [...][]byte{
		{byte(utf8.DECODED_CHARACTER_MINIMUM)},
		{byte(utf8.CHARACTER_SIZE_MINIMUM)},
		{byte(utf8.CHARACTER_SIZE_TWO)},
		encoded[:encoded_size],
	}
	for index := range literals {
		pattern, _, status := glob.Compile(
			compile_input(workspace, literals[index], nil),
		)
		testify.Equal_Values(t, glob.STATUS_OK, status)
		var match glob.Match_Workspace
		testify.True(t, match_from(t, pattern, literals[index], &match))
	}

	classes := [...][]byte{
		{'[', byte(utf8.DECODED_CHARACTER_MINIMUM), ']'},
		{'[', byte(utf8.CHARACTER_SIZE_MINIMUM), ']'},
		{'[', byte(utf8.CHARACTER_SIZE_TWO), ']'},
	}
	for index := range classes {
		pattern, _, status := glob.Compile(
			compile_input(workspace, classes[index], nil),
		)
		testify.Equal_Values(t, glob.STATUS_OK, status)
		var match glob.Match_Workspace
		member_start := utf8.CHARACTER_SIZE_MINIMUM
		member_end := len(classes[index]) - utf8.CHARACTER_SIZE_MINIMUM
		class_member := classes[index][member_start:member_end]
		testify.True(t, match_from(t, pattern, class_member, &match))
	}

	var class_storage [utf8.CHARACTER_SIZE_MAXIMUM + len("[]")]byte
	class_maximum := class_storage[:int(encoded_size)+len("[]")]
	class_maximum[bytes.SLICE_SIZE_MINIMUM] = '['
	copy(class_maximum[utf8.CHARACTER_SIZE_MINIMUM:], encoded[:encoded_size])
	class_maximum[len(class_maximum)-utf8.CHARACTER_SIZE_MINIMUM] = ']'
	pattern, _, status := glob.Compile(compile_input(workspace, class_maximum, nil))
	testify.Equal_Values(t, glob.STATUS_OK, status)
	var match glob.Match_Workspace
	testify.True(t, match_from(t, pattern, encoded[:encoded_size], &match))
}

func match_boundaries(t *testing.T, pattern glob.Pattern) {
	_, status := glob.Match(match_input(pattern, nil, nil))
	testify.Equal_Values(t, glob.STATUS_WORKSPACE_INVALID, status)

	var workspace glob.Match_Workspace
	_, status = glob.Match(match_input(glob.Pattern{}, nil, &workspace))
	testify.Equal_Values(t, glob.STATUS_PATTERN_INVALID, status)
	match_forged_pattern(t, &workspace)

	var compiled glob.Compile_Workspace
	star := compile_from(t, &compiled, []byte("*"), nil)
	var text [glob.TEXT_SIZE_MAXIMUM]byte
	for index := range text {
		text[index] = 'a'
	}
	testify.True(t, match_from(t, star, text[:], &workspace))

	character_pattern := compile_from(t, &compiled, []byte("?"), nil)
	var encoded [utf8.CHARACTER_SIZE_MAXIMUM]byte
	encoded_size := utf8.Encode_Character(
		encoded[:], utf8.Character(utf8.DECODED_CHARACTER_MAXIMUM),
	)
	texts := [...][]byte{
		{byte(utf8.DECODED_CHARACTER_MINIMUM)},
		{byte(utf8.CHARACTER_SIZE_MINIMUM)},
		{byte(utf8.CHARACTER_SIZE_TWO)},
		encoded[:encoded_size],
	}
	for index := range texts {
		testify.True(t, match_from(t, character_pattern, texts[index], &workspace))
	}

	match_active_state_boundary(t, &compiled, &workspace)
}

func match_forged_pattern(t *testing.T, workspace *glob.Match_Workspace) {
	var compiled glob.Compile_Workspace
	pattern := compile_from(t, &compiled, []byte("a"), nil)
	start := pattern.Control[glob.PATTERN_CONTROL_START]
	compiled.Instruction_Kinds[start] = glob.INSTRUCTION_SPLIT +
		utf8.CHARACTER_SIZE_MINIMUM
	_, status := glob.Match(match_input(pattern, nil, workspace))
	testify.Equal_Values(t, glob.STATUS_PATTERN_INVALID, status)
}

func match_active_state_boundary(
	t *testing.T,
	compiled *glob.Compile_Workspace,
	workspace *glob.Match_Workspace,
) {
	var source [glob.PATTERN_SIZE_MAXIMUM]byte
	position := bytes.SLICE_SIZE_MINIMUM
	source[position] = '{'
	position++
	branch_count := glob.PATTERN_DEPTH_MAXIMUM - utf8.CHARACTER_SIZE_MINIMUM
	for branch := bytes.SLICE_SIZE_MINIMUM; branch < branch_count; branch++ {
		source[position] = '*'
		position++
		if branch+utf8.CHARACTER_SIZE_MINIMUM < branch_count {
			source[position] = ','
			position++
		}
	}
	source[position] = '}'
	position++
	source[position] = '*'
	position++
	testify.Equal(t, len(source), position)
	pattern := compile_from(t, compiled, source[:], nil)
	testify.True(t, match_from(t, pattern, nil, workspace))
	testify.True(t, match_from(t, pattern, []byte("a"), workspace))
}

func quote_boundaries(t *testing.T) {
	var destination [glob.QUOTED_SIZE_MAXIMUM]byte
	for count := bytes.SLICE_SIZE_MINIMUM; count <= utf8.CHARACTER_SIZE_TWO; count++ {
		source := destination[:count]
		written, status := glob.Quote_Meta_Into(destination[:count], source)
		testify.Equal_Values(t, glob.STATUS_OK, status)
		testify.Equal_Values(t, glob.Output_Count(count), written)
	}

	var source [glob.PATTERN_SIZE_MAXIMUM]byte
	for index := range source {
		source[index] = '*'
	}
	written, status := glob.Quote_Meta_Into(destination[:], source[:])
	testify.Equal_Values(t, glob.STATUS_OK, status)
	testify.Equal_Values(t, glob.Output_Count(glob.QUOTED_SIZE_MAXIMUM), written)

	var oversized [glob.PATTERN_SIZE_UNVALIDATED_MAXIMUM]byte
	_, status = glob.Quote_Meta_Into(destination[:], oversized[:])
	testify.Equal_Values(t, glob.STATUS_INPUT_INVALID, status)
}

func special_boundaries(t *testing.T) {
	characters := [...]glob.Character{
		glob.Character(utf8.DECODED_CHARACTER_MINIMUM),
		glob.Character(utf8.CHARACTER_SIZE_MINIMUM),
		glob.Character(utf8.CHARACTER_SIZE_TWO),
		glob.Character(bits.WORD_8_MAXIMUM),
	}
	for index := range characters {
		testify.False(t, bool(glob.Special(characters[index])))
	}
}
