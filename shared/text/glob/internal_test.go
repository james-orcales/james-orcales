package glob

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

// The matches and index_at helpers adapt the pointer-taking evaluator
// (matcher_matches, Index) to the value-holding whitebox tests, which build
// matchers with the New_* constructors: a constructor result is not addressable,
// so the tests hand a value to these helpers and the helpers take the address of
// their own parameter.
func matches(matcher Matcher, text string) (matched bool) {
	return matcher_matches(&matcher, text)
}

func index_at(matcher Matcher, text string) (offset int, segments []int) {
	return Index(&matcher, text)
}

// Front-end lexer and parser tests (whitebox).

// Drives the lexer over pattern and checks it yields exactly items in order. This is
// the shared assertion the ported lexer/lexer_test.go cases run through.
func assert_tokens(t *testing.T, pattern string, items []Token) {
	t.Helper()
	lexer := new_lexer(pattern)
	for item_index, expected := range items {
		actual := lex_next(lexer)
		if actual.Kind != expected.Kind {
			t.Errorf("%q item %d: kind = %s, want %s",
				pattern, item_index, actual.Kind, expected.Kind)
		}
		if actual.Raw != expected.Raw {
			t.Errorf("%q item %d: raw = %q, want %q",
				pattern, item_index, actual.Raw, expected.Raw)
		}
	}
}

// Parses the seeded token stream and checks the resulting tree equals want. This is
// the shared assertion the ported ast/parser_test.go cases run through; upstream fed
// a stub Lexer, so seeding the concrete lexer's buffer preserves the same input.
func assert_tree(t *testing.T, tokens []Token, want *Node) {
	t.Helper()
	result, err := parse_lexer(&Lexer{Buffer: tokens})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(want, result) {
		t.Errorf("\n got %s\nwant %s", result, want)
	}
}

// Test_Lex_Text ports the plain-text lexer cases, including escaped metacharacters
// and a bare comma outside a group.
func Test_Lex_Text(t *testing.T) {
	assert_tokens(t, "", []Token{{Kind: TOKEN_EOF}})
	assert_tokens(t, "hello", []Token{
		{Kind: TOKEN_TEXT, Raw: "hello"}, {Kind: TOKEN_EOF},
	})
	assert_tokens(t, "hello,world", []Token{
		{Kind: TOKEN_TEXT, Raw: "hello,world"}, {Kind: TOKEN_EOF},
	})
	assert_tokens(t, "hello\\,world", []Token{
		{Kind: TOKEN_TEXT, Raw: "hello,world"}, {Kind: TOKEN_EOF},
	})
	assert_tokens(t, "hello\\{world", []Token{
		{Kind: TOKEN_TEXT, Raw: "hello{world"}, {Kind: TOKEN_EOF},
	})
}

// Test_Lex_Wildcards ports the single, any, and super wildcard lexer cases.
func Test_Lex_Wildcards(t *testing.T) {
	assert_tokens(t, "hello?", []Token{
		{Kind: TOKEN_TEXT, Raw: "hello"},
		{Kind: TOKEN_SINGLE, Raw: "?"},
		{Kind: TOKEN_EOF},
	})
	assert_tokens(t, "hellof*", []Token{
		{Kind: TOKEN_TEXT, Raw: "hellof"},
		{Kind: TOKEN_ANY, Raw: "*"},
		{Kind: TOKEN_EOF},
	})
	assert_tokens(t, "hello**", []Token{
		{Kind: TOKEN_TEXT, Raw: "hello"},
		{Kind: TOKEN_SUPER, Raw: "**"},
		{Kind: TOKEN_EOF},
	})
}

// Test_Lex_Class ports the character-class lexer cases: multibyte ranges, negation,
// and member lists.
func Test_Lex_Class(t *testing.T) {
	assert_tokens(t, "[日-語]", []Token{
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_RANGE_LOW, Raw: "日"},
		{Kind: TOKEN_RANGE_BETWEEN, Raw: "-"},
		{Kind: TOKEN_RANGE_HIGH, Raw: "語"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_EOF},
	})
	assert_tokens(t, "[!日-語]", []Token{
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_NOT, Raw: "!"},
		{Kind: TOKEN_RANGE_LOW, Raw: "日"},
		{Kind: TOKEN_RANGE_BETWEEN, Raw: "-"},
		{Kind: TOKEN_RANGE_HIGH, Raw: "語"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_EOF},
	})
	assert_tokens(t, "[日本語]", []Token{
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_TEXT, Raw: "日本語"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_EOF},
	})
	assert_tokens(t, "[!日本語]", []Token{
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_NOT, Raw: "!"},
		{Kind: TOKEN_TEXT, Raw: "日本語"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_EOF},
	})
}

// Test_Lex_Terms ports the alternatives-group lexer cases.
func Test_Lex_Terms(t *testing.T) {
	assert_tokens(t, "{a,b}", []Token{
		{Kind: TOKEN_TERMS_OPEN, Raw: "{"},
		{Kind: TOKEN_TEXT, Raw: "a"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_TEXT, Raw: "b"},
		{Kind: TOKEN_TERMS_CLOSE, Raw: "}"},
		{Kind: TOKEN_EOF},
	})
	assert_tokens(t, "/{z,ab}*", []Token{
		{Kind: TOKEN_TEXT, Raw: "/"},
		{Kind: TOKEN_TERMS_OPEN, Raw: "{"},
		{Kind: TOKEN_TEXT, Raw: "z"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_TEXT, Raw: "ab"},
		{Kind: TOKEN_TERMS_CLOSE, Raw: "}"},
		{Kind: TOKEN_ANY, Raw: "*"},
		{Kind: TOKEN_EOF},
	})
}

// Test_Lex_Rate ports the mixed case that also carries a literal `]` after a closed
// class, proving the trailing bracket lexes as text inside the group.
func Test_Lex_Rate(t *testing.T) {
	assert_tokens(t, "/{rate,[0-9]]}*", []Token{
		{Kind: TOKEN_TEXT, Raw: "/"},
		{Kind: TOKEN_TERMS_OPEN, Raw: "{"},
		{Kind: TOKEN_TEXT, Raw: "rate"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_RANGE_LOW, Raw: "0"},
		{Kind: TOKEN_RANGE_BETWEEN, Raw: "-"},
		{Kind: TOKEN_RANGE_HIGH, Raw: "9"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_TEXT, Raw: "]"},
		{Kind: TOKEN_TERMS_CLOSE, Raw: "}"},
		{Kind: TOKEN_ANY, Raw: "*"},
		{Kind: TOKEN_EOF},
	})
}

// Test_Lex_Nested_Group ports the deepest lexer case: a group holding a negated
// range, wildcards, and an inner group with an escaped member.
func Test_Lex_Nested_Group(t *testing.T) {
	assert_tokens(t, "{[!日-語],*,?,{a,b,\\c}}", []Token{
		{Kind: TOKEN_TERMS_OPEN, Raw: "{"},
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_NOT, Raw: "!"},
		{Kind: TOKEN_RANGE_LOW, Raw: "日"},
		{Kind: TOKEN_RANGE_BETWEEN, Raw: "-"},
		{Kind: TOKEN_RANGE_HIGH, Raw: "語"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_ANY, Raw: "*"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_SINGLE, Raw: "?"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_TERMS_OPEN, Raw: "{"},
		{Kind: TOKEN_TEXT, Raw: "a"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_TEXT, Raw: "b"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_TEXT, Raw: "c"},
		{Kind: TOKEN_TERMS_CLOSE, Raw: "}"},
		{Kind: TOKEN_TERMS_CLOSE, Raw: "}"},
		{Kind: TOKEN_EOF},
	})
}

// Test_Parse_Literals ports the parser cases where a text run surrounds a single
// wildcard, pinning the flat NODE_KIND_PATTERN sequence.
func Test_Parse_Literals(t *testing.T) {
	assert_tree(t,
		[]Token{{Kind: TOKEN_TEXT, Raw: "abc"}, {Kind: TOKEN_EOF}},
		new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "abc"}))
	assert_tree(t, []Token{
		{Kind: TOKEN_TEXT, Raw: "a"}, {Kind: TOKEN_ANY, Raw: "*"},
		{Kind: TOKEN_TEXT, Raw: "c"}, {Kind: TOKEN_EOF},
	}, new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "a"},
		&Node{Kind: NODE_KIND_ANY}, &Node{Kind: NODE_KIND_TEXT, Text: "c"}))
	assert_tree(t, []Token{
		{Kind: TOKEN_TEXT, Raw: "a"}, {Kind: TOKEN_SUPER, Raw: "**"},
		{Kind: TOKEN_TEXT, Raw: "c"}, {Kind: TOKEN_EOF},
	}, new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "a"},
		&Node{Kind: NODE_KIND_SUPER}, &Node{Kind: NODE_KIND_TEXT, Text: "c"}))
	assert_tree(t, []Token{
		{Kind: TOKEN_TEXT, Raw: "a"}, {Kind: TOKEN_SINGLE, Raw: "?"},
		{Kind: TOKEN_TEXT, Raw: "c"}, {Kind: TOKEN_EOF},
	}, new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "a"},
		&Node{Kind: NODE_KIND_SINGLE}, &Node{Kind: NODE_KIND_TEXT, Text: "c"}))
}

// Test_Parse_Class ports the parser cases for a negated range and a bare member list.
func Test_Parse_Class(t *testing.T) {
	assert_tree(t, []Token{
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_NOT, Raw: "!"},
		{Kind: TOKEN_RANGE_LOW, Raw: "a"},
		{Kind: TOKEN_RANGE_BETWEEN, Raw: "-"},
		{Kind: TOKEN_RANGE_HIGH, Raw: "z"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_EOF},
	}, new_node(NODE_KIND_PATTERN,
		&Node{Kind: NODE_KIND_RANGE, Low: 'a', High: 'z', Negated: true}))
	assert_tree(t, []Token{
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_TEXT, Raw: "az"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_EOF},
	}, new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_LIST, Characters: "az"}))
}

// Test_Parse_Terms ports the parser cases for a simple group and a group flanked by
// text and a trailing wildcard.
func Test_Parse_Terms(t *testing.T) {
	assert_tree(t, []Token{
		{Kind: TOKEN_TERMS_OPEN, Raw: "{"},
		{Kind: TOKEN_TEXT, Raw: "a"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_TEXT, Raw: "z"},
		{Kind: TOKEN_TERMS_CLOSE, Raw: "}"},
		{Kind: TOKEN_EOF},
	}, new_node(NODE_KIND_PATTERN, new_node(NODE_KIND_ANY_OF,
		new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "a"}),
		new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "z"}))))
	assert_tree(t, []Token{
		{Kind: TOKEN_TEXT, Raw: "/"},
		{Kind: TOKEN_TERMS_OPEN, Raw: "{"},
		{Kind: TOKEN_TEXT, Raw: "z"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_TEXT, Raw: "ab"},
		{Kind: TOKEN_TERMS_CLOSE, Raw: "}"},
		{Kind: TOKEN_ANY, Raw: "*"},
		{Kind: TOKEN_EOF},
	}, new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "/"},
		new_node(NODE_KIND_ANY_OF,
			new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "z"}),
			new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "ab"})),
		&Node{Kind: NODE_KIND_ANY}))
}

// Test_Parse_Nested ports the deepest parser case: a group whose alternatives are a
// text, an inner group, a single, a range, and a negated list.
func Test_Parse_Nested(t *testing.T) {
	tokens := []Token{
		{Kind: TOKEN_TERMS_OPEN, Raw: "{"},
		{Kind: TOKEN_TEXT, Raw: "a"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_TERMS_OPEN, Raw: "{"},
		{Kind: TOKEN_TEXT, Raw: "x"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_TEXT, Raw: "y"},
		{Kind: TOKEN_TERMS_CLOSE, Raw: "}"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_SINGLE, Raw: "?"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_RANGE_LOW, Raw: "a"},
		{Kind: TOKEN_RANGE_BETWEEN, Raw: "-"},
		{Kind: TOKEN_RANGE_HIGH, Raw: "z"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_SEPARATOR, Raw: ","},
		{Kind: TOKEN_RANGE_OPEN, Raw: "["},
		{Kind: TOKEN_NOT, Raw: "!"},
		{Kind: TOKEN_TEXT, Raw: "qwe"},
		{Kind: TOKEN_RANGE_CLOSE, Raw: "]"},
		{Kind: TOKEN_TERMS_CLOSE, Raw: "}"},
		{Kind: TOKEN_EOF},
	}
	inner := new_node(NODE_KIND_PATTERN, new_node(NODE_KIND_ANY_OF,
		new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "x"}),
		new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "y"})))
	want := new_node(NODE_KIND_PATTERN, new_node(NODE_KIND_ANY_OF,
		new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_TEXT, Text: "a"}),
		inner,
		new_node(NODE_KIND_PATTERN, &Node{Kind: NODE_KIND_SINGLE}),
		new_node(NODE_KIND_PATTERN,
			&Node{Kind: NODE_KIND_RANGE, Low: 'a', High: 'z'}),
		new_node(NODE_KIND_PATTERN,
			&Node{Kind: NODE_KIND_LIST, Characters: "qwe", Negated: true})))
	assert_tree(t, tokens, want)
}

// Parse-tree regression tests (originally the syntax blackbox spec).

// Test_Parse_Text checks a run of ordinary runes parses to a single NODE_KIND_TEXT leaf
// holding the run verbatim.
func Test_Parse_Text(t *testing.T) {
	leaf := only_child(t, parse_ok(t, "abc"))
	if leaf.Kind != NODE_KIND_TEXT {
		t.Fatalf("kind = %s, want Text", leaf.Kind)
	}
	if leaf.Text != "abc" {
		t.Fatalf("text = %q, want %q", leaf.Text, "abc")
	}
}

// Test_Parse_Any checks a lone `*` parses to a NODE_KIND_ANY leaf.
func Test_Parse_Any(t *testing.T) {
	leaf := only_child(t, parse_ok(t, "*"))
	if leaf.Kind != NODE_KIND_ANY {
		t.Fatalf("kind = %s, want Any", leaf.Kind)
	}
}

// Test_Parse_Super checks a `**` parses to a NODE_KIND_SUPER leaf, distinct from a lone
// `*`.
func Test_Parse_Super(t *testing.T) {
	leaf := only_child(t, parse_ok(t, "**"))
	if leaf.Kind != NODE_KIND_SUPER {
		t.Fatalf("kind = %s, want Super", leaf.Kind)
	}
}

// Test_Parse_Single checks a `?` parses to a NODE_KIND_SINGLE leaf.
func Test_Parse_Single(t *testing.T) {
	leaf := only_child(t, parse_ok(t, "?"))
	if leaf.Kind != NODE_KIND_SINGLE {
		t.Fatalf("kind = %s, want Single", leaf.Kind)
	}
}

// Test_Parse_List checks a bracket class of bare members parses to a NODE_KIND_LIST leaf
// carrying the members, not negated.
func Test_Parse_List(t *testing.T) {
	leaf := only_child(t, parse_ok(t, "[az]"))
	if leaf.Kind != NODE_KIND_LIST {
		t.Fatalf("kind = %s, want List", leaf.Kind)
	}
	if leaf.Characters != "az" {
		t.Fatalf("characters = %q, want %q", leaf.Characters, "az")
	}
	if leaf.Negated {
		t.Fatalf("negated = true, want false")
	}
}

// Test_Parse_Range checks a bracket range parses to a NODE_KIND_RANGE leaf with the
// inclusive bounds set and not negated.
func Test_Parse_Range(t *testing.T) {
	leaf := only_child(t, parse_ok(t, "[a-z]"))
	if leaf.Kind != NODE_KIND_RANGE {
		t.Fatalf("kind = %s, want Range", leaf.Kind)
	}
	if leaf.Low != 'a' {
		t.Fatalf("low = %q, want %q", leaf.Low, 'a')
	}
	if leaf.High != 'z' {
		t.Fatalf("high = %q, want %q", leaf.High, 'z')
	}
	if leaf.Negated {
		t.Fatalf("negated = true, want false")
	}
}

// Test_Parse_Range_Negated checks a leading `!` inside the brackets sets Negated on
// the resulting NODE_KIND_RANGE leaf.
func Test_Parse_Range_Negated(t *testing.T) {
	leaf := only_child(t, parse_ok(t, "[!a-z]"))
	if leaf.Kind != NODE_KIND_RANGE {
		t.Fatalf("kind = %s, want Range", leaf.Kind)
	}
	if !leaf.Negated {
		t.Fatalf("negated = false, want true")
	}
}

// Test_Parse_Alternatives checks a group parses to a NODE_KIND_ANY_OF node whose pattern
// children are the comma-separated alternatives in order.
func Test_Parse_Alternatives(t *testing.T) {
	group := only_child(t, parse_ok(t, "{a,b}"))
	if group.Kind != NODE_KIND_ANY_OF {
		t.Fatalf("kind = %s, want AnyOf", group.Kind)
	}
	if len(group.Children) != 2 {
		t.Fatalf("alternatives = %d, want 2", len(group.Children))
	}
	if only_child(t, group.Children[0]).Text != "a" {
		t.Fatalf("first alternative text = %q, want %q", group.Children[0], "a")
	}
	if only_child(t, group.Children[1]).Text != "b" {
		t.Fatalf("second alternative text = %q, want %q", group.Children[1], "b")
	}
}

// Test_Parse_Escape checks a backslash makes the next rune literal, so `a\*b` is one
// text leaf rather than a wildcard.
func Test_Parse_Escape(t *testing.T) {
	leaf := only_child(t, parse_ok(t, "a\\*b"))
	if leaf.Kind != NODE_KIND_TEXT {
		t.Fatalf("kind = %s, want Text", leaf.Kind)
	}
	if leaf.Text != "a*b" {
		t.Fatalf("text = %q, want %q", leaf.Text, "a*b")
	}
}

// Test_Depth_Bound_Rejects_Over_Limit checks a pattern nesting groups one past
// PATTERN_DEPTH_MAX is rejected with an error rather than recursed into.
func Test_Depth_Bound_Rejects_Over_Limit(t *testing.T) {
	over := PATTERN_DEPTH_MAX + 1
	pattern := strings.Repeat("{", over) + strings.Repeat("}", over)
	tree, err := Parse(pattern)
	if err == nil {
		t.Fatalf("nesting %d deep parsed without error", over)
	}
	if tree != nil {
		t.Fatalf("tree = %v, want nil on rejection", tree)
	}
}

// Test_Depth_Bound_Accepts_At_Limit checks a pattern nesting groups exactly
// PATTERN_DEPTH_MAX deep parses without error, so the bound rejects only excess.
func Test_Depth_Bound_Accepts_At_Limit(t *testing.T) {
	limit := PATTERN_DEPTH_MAX
	pattern := strings.Repeat("{", limit) + strings.Repeat("}", limit)
	tree, err := Parse(pattern)
	if err != nil {
		t.Fatalf("nesting %d deep: unexpected error: %v", limit, err)
	}
	if tree == nil {
		t.Fatalf("tree = nil, want a parse tree at the limit")
	}
}

// Test_Special_Characters checks Special reports true for each of the seven
// metacharacters and false for other bytes.
func Test_Special_Characters(t *testing.T) {
	for _, meta := range []byte{'*', '?', '\\', '[', ']', '{', '}'} {
		if !Special(meta) {
			t.Fatalf("Special(%q) = false, want true", meta)
		}
	}
	for _, ordinary := range []byte{'a', ',', '!', '-', '/', '0'} {
		if Special(ordinary) {
			t.Fatalf("Special(%q) = true, want false", ordinary)
		}
	}
}

// Parses pattern, fails the test on error, and asserts the root is a pattern node so
// each leaf test can navigate from a known root.
func parse_ok(t *testing.T, pattern string) (tree *Node) {
	t.Helper()
	tree, err := Parse(pattern)
	if err != nil {
		t.Fatalf("Parse(%q): unexpected error: %v", pattern, err)
	}
	if tree.Kind != NODE_KIND_PATTERN {
		t.Fatalf("root kind = %s, want Pattern", tree.Kind)
	}
	return tree
}

// Asserts node has exactly one child and returns it, the common shape of a
// single-construct pattern.
func only_child(t *testing.T, node *Node) (child *Node) {
	t.Helper()
	if len(node.Children) != 1 {
		t.Fatalf("children = %d, want 1", len(node.Children))
	}
	return node.Children[0]
}

// Matcher Index and whole-string internals (whitebox).

type index_case struct {
	Matcher  Matcher
	Fixture  string
	Offset   int
	Segments []int
}

func run_index_cases(t *testing.T, cases []index_case) {
	t.Helper()
	for case_index, row := range cases {
		offset, segments := index_at(row.Matcher, row.Fixture)
		if offset != row.Offset {
			t.Errorf("#%d offset = %d, want %d", case_index, offset, row.Offset)
		}
		if !slices.Equal(segments, row.Segments) {
			t.Errorf("#%d segments = %v, want %v", case_index, segments, row.Segments)
		}
	}
}

// Test_Any_Index pins Any's index of the leading separator-free run.
func Test_Any_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{
			Matcher:  New_Any([]rune{'.'}),
			Fixture:  "abc",
			Offset:   0,
			Segments: []int{0, 1, 2, 3},
		},
		{
			Matcher:  New_Any([]rune{'.'}),
			Fixture:  "abc.def",
			Offset:   0,
			Segments: []int{0, 1, 2, 3},
		},
	})
}

// Test_Any_Of_Index pins that Any_Of keeps the earliest child match and merges
// ties.
func Test_Any_Of_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{
			Matcher:  New_Any_Of(New_Any(nil), New_Text("b"), New_Text("c")),
			Fixture:  "abc",
			Offset:   0,
			Segments: []int{0, 1, 2, 3},
		},
		{
			Matcher:  New_Any_Of(New_Prefix("b"), New_Suffix("c")),
			Fixture:  "abc",
			Offset:   0,
			Segments: []int{3},
		},
		{
			Matcher: New_Any_Of(
				New_List([]rune("[def]"), false), New_List([]rune("[abc]"), false)),
			Fixture:  "abcdef",
			Offset:   0,
			Segments: []int{1},
		},
	})
}

// Test_Btree_Match pins BTree's whole-string match over several tree shapes.
func Test_Btree_Match(t *testing.T) {
	super := New_Super()
	tree_supers := New_Btree(
		&New_Btree_Input{Value: New_Text("abc"), Left: &super, Right: &super})
	if !matches(tree_supers, "abc") {
		t.Errorf("btree over super sides must match abc")
	}

	single := New_Single(nil)
	tree_singles := New_Btree(
		&New_Btree_Input{Value: New_Text("a"), Left: &single, Right: &single})
	if !matches(tree_singles, "aaa") {
		t.Errorf("btree a between two singles must match aaa")
	}

	tree_open_right := New_Btree(
		&New_Btree_Input{Value: New_Text("b"), Left: &single, Right: nil})
	if matches(tree_open_right, "bbb") {
		t.Errorf("btree b with single left and empty right must not match bbb")
	}

	inner := New_Btree(&New_Btree_Input{Value: New_Single(nil), Left: &super, Right: nil})
	tree_nested := New_Btree(&New_Btree_Input{Value: New_Text("c"), Left: &inner, Right: nil})
	if !matches(tree_nested, "abc") {
		t.Errorf("nested btree must match abc")
	}
}

// Test_Contains_Index pins Contains's index for the present and negated cases.
func Test_Contains_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{
			Matcher:  New_Contains("ab", false),
			Fixture:  "abc",
			Offset:   0,
			Segments: []int{2, 3},
		},
		{
			Matcher:  New_Contains("ab", false),
			Fixture:  "fffabfff",
			Offset:   0,
			Segments: []int{5, 6, 7, 8},
		},
		{Matcher: New_Contains("ab", true), Fixture: "abc", Offset: 0, Segments: []int{0}},
		{
			Matcher:  New_Contains("ab", true),
			Fixture:  "fffabfff",
			Offset:   0,
			Segments: []int{0, 1, 2, 3},
		},
	})
}

// Test_Every_Of_Index pins Every_Of's shared-endpoint intersection.
func Test_Every_Of_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{
			Matcher:  New_Every_Of(New_Any(nil), New_Text("b"), New_Text("c")),
			Fixture:  "dbc",
			Offset:   -1,
			Segments: nil,
		},
		{
			Matcher:  New_Every_Of(New_Any(nil), New_Prefix("b"), New_Suffix("c")),
			Fixture:  "abc",
			Offset:   1,
			Segments: []int{2},
		},
	})
}

// Test_List_Index pins List's index for the present and negated cases.
func Test_List_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{
			Matcher:  New_List([]rune("ab"), false),
			Fixture:  "abc",
			Offset:   0,
			Segments: []int{1},
		},
		{
			Matcher:  New_List([]rune("ab"), true),
			Fixture:  "fffabfff",
			Offset:   0,
			Segments: []int{1},
		},
	})
}

// Test_Index_Max pins Max's index and its cutoff at the limit.
func Test_Index_Max(t *testing.T) {
	run_index_cases(t, []index_case{
		{Matcher: New_Max(3), Fixture: "abc", Offset: 0, Segments: []int{0, 1, 2, 3}},
		{Matcher: New_Max(3), Fixture: "abcdef", Offset: 0, Segments: []int{0, 1, 2, 3}},
	})
}

// Test_Index_Min pins Min's index from the limit onward.
func Test_Index_Min(t *testing.T) {
	run_index_cases(t, []index_case{
		{Matcher: New_Min(1), Fixture: "abc", Offset: 0, Segments: []int{1, 2, 3}},
		{Matcher: New_Min(3), Fixture: "abcd", Offset: 0, Segments: []int{3, 4}},
	})
}

// Test_Empty_Index pins the empty matcher's zero-length segment (upstream
// Nothing).
func Test_Empty_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{Matcher: New_Empty(), Fixture: "abc", Offset: 0, Segments: []int{0}},
		{Matcher: New_Empty(), Fixture: "", Offset: 0, Segments: []int{0}},
	})
}

// Test_Prefix_Index pins Prefix's index and the tail segments it reports.
func Test_Prefix_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{Matcher: New_Prefix("ab"), Fixture: "abc", Offset: 0, Segments: []int{2, 3}},
		{
			Matcher:  New_Prefix("ab"),
			Fixture:  "fffabfff",
			Offset:   3,
			Segments: []int{2, 3, 4, 5},
		},
	})
}

// Test_Prefix_Any_Index pins Prefix_Any's index and its separator-bounded tail.
func Test_Prefix_Any_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{
			Matcher:  New_Prefix_Any("ab", []rune{'.'}),
			Fixture:  "ab",
			Offset:   0,
			Segments: []int{2},
		},
		{
			Matcher:  New_Prefix_Any("ab", []rune{'.'}),
			Fixture:  "abc",
			Offset:   0,
			Segments: []int{2, 3},
		},
		{
			Matcher:  New_Prefix_Any("ab", []rune{'.'}),
			Fixture:  "qw.abcd.efg",
			Offset:   3,
			Segments: []int{2, 3, 4},
		},
	})
}

// Test_Prefix_Suffix_Index pins Prefix_Suffix's index and its suffix endpoints.
func Test_Prefix_Suffix_Index(t *testing.T) {
	matcher_ac := New_Prefix_Suffix(&New_Prefix_Suffix_Input{Prefix: "a", Suffix: "c"})
	matcher_ff := New_Prefix_Suffix(&New_Prefix_Suffix_Input{Prefix: "f", Suffix: "f"})
	matcher_abbc := New_Prefix_Suffix(&New_Prefix_Suffix_Input{Prefix: "ab", Suffix: "bc"})
	run_index_cases(t, []index_case{
		{Matcher: matcher_ac, Fixture: "abc", Offset: 0, Segments: []int{3}},
		{
			Matcher:  matcher_ff,
			Fixture:  "fffabfff",
			Offset:   0,
			Segments: []int{1, 2, 3, 6, 7, 8},
		},
		{Matcher: matcher_abbc, Fixture: "abc", Offset: 0, Segments: []int{3}},
	})
}

// Test_Range_Index pins Range's index for the plain and negated intervals.
func Test_Range_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{
			Matcher:  New_Range(&New_Range_Input{Low: 'a', High: 'z', Negated: false}),
			Fixture:  "abc",
			Offset:   0,
			Segments: []int{1},
		},
		{
			Matcher:  New_Range(&New_Range_Input{Low: 'a', High: 'c', Negated: false}),
			Fixture:  "abcd",
			Offset:   0,
			Segments: []int{1},
		},
		{
			Matcher:  New_Range(&New_Range_Input{Low: 'a', High: 'c', Negated: true}),
			Fixture:  "abcd",
			Offset:   3,
			Segments: []int{1},
		},
	})
}

// Test_Row_Index pins Row's index of a fixed-width child sequence.
func Test_Row_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{
			Matcher:  New_Row(7, New_Text("abc"), New_Text("def"), New_Single(nil)),
			Fixture:  "qweabcdefghij",
			Offset:   3,
			Segments: []int{7},
		},
		{
			Matcher:  New_Row(7, New_Text("abc"), New_Text("def"), New_Single(nil)),
			Fixture:  "abcd",
			Offset:   -1,
			Segments: nil,
		},
	})
}

// Test_Single_Index pins Single's index of the first non-separator rune.
func Test_Single_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{Matcher: New_Single([]rune{'.'}), Fixture: ".abc", Offset: 1, Segments: []int{1}},
		{Matcher: New_Single([]rune{'.'}), Fixture: ".", Offset: -1, Segments: nil},
	})
}

// Test_Suffix_Index pins Suffix's index and its single endpoint.
func Test_Suffix_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{Matcher: New_Suffix("ab"), Fixture: "abc", Offset: 0, Segments: []int{2}},
		{Matcher: New_Suffix("ab"), Fixture: "fffabfff", Offset: 0, Segments: []int{5}},
	})
}

// Test_Suffix_Any_Index pins Suffix_Any's index and its separator-bounded head.
func Test_Suffix_Any_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{
			Matcher:  New_Suffix_Any("ab", []rune{'.'}),
			Fixture:  "ab",
			Offset:   0,
			Segments: []int{2},
		},
		{
			Matcher:  New_Suffix_Any("ab", []rune{'.'}),
			Fixture:  "cab",
			Offset:   0,
			Segments: []int{3},
		},
		{
			Matcher:  New_Suffix_Any("ab", []rune{'.'}),
			Fixture:  "qw.cdab.efg",
			Offset:   3,
			Segments: []int{4},
		},
	})
}

// Test_Super_Index pins Super's index over the whole string.
func Test_Super_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{Matcher: New_Super(), Fixture: "abc", Offset: 0, Segments: []int{0, 1, 2, 3}},
		{Matcher: New_Super(), Fixture: "", Offset: 0, Segments: []int{0}},
	})
}

// Test_Text_Index pins Text's index of its literal and the no-match case.
func Test_Text_Index(t *testing.T) {
	run_index_cases(t, []index_case{
		{Matcher: New_Text("b"), Fixture: "abc", Offset: 1, Segments: []int{1}},
		{Matcher: New_Text("f"), Fixture: "abcd", Offset: -1, Segments: nil},
	})
}

// Test_Append_Merge pins the merge of two sorted, unique segment lists.
func Test_Append_Merge(t *testing.T) {
	first := append_merge(&Append_Merge_Input{Target: []int{0, 6, 7}, Source: []int{0, 1, 3}})
	if !slices.Equal(first, []int{0, 1, 3, 6, 7}) {
		t.Errorf("merge = %v, want [0 1 3 6 7]", first)
	}
	second := append_merge(
		&Append_Merge_Input{Target: []int{0, 1, 3, 6, 7}, Source: []int{0, 1, 10}})
	if !slices.Equal(second, []int{0, 1, 3, 6, 7, 10}) {
		t.Errorf("merge = %v, want [0 1 3 6 7 10]", second)
	}
}

// Matcher whole-string regression tests (originally the match blackbox spec).

// Test_Any_Matches_Runs_Without_Separators checks a separator-free string
// matches and that Index reports the leading run up to the first separator.
func Test_Any_Matches_Runs_Without_Separators(t *testing.T) {
	matcher := New_Any([]rune{'.'})
	if !matches(matcher, "abc") {
		t.Fatalf("a separator-free string must match")
	}
	if matches(matcher, "a.c") {
		t.Fatalf("a string containing a separator must not match")
	}
	offset, segments := index_at(matcher, "ab.c")
	if offset != 0 {
		t.Fatalf("offset = %d, want 0", offset)
	}
	if !slices.Equal(segments, []int{0, 1, 2}) {
		t.Fatalf("segments = %v, want [0 1 2]", segments)
	}
}

// Test_Super_Matches_Every_String checks the '**' matcher accepts strings with
// separators as well as the empty string.
func Test_Super_Matches_Every_String(t *testing.T) {
	matcher := New_Super()
	if !matches(matcher, "a.b/c") {
		t.Fatalf("super must match any non-empty string")
	}
	if !matches(matcher, "") {
		t.Fatalf("super must match the empty string")
	}
}

// Test_Single_Matches_One_Non_Separator_Rune checks '?' accepts a lone
// non-separator rune and rejects separators and longer strings.
func Test_Single_Matches_One_Non_Separator_Rune(t *testing.T) {
	matcher := New_Single([]rune{'.'})
	if !matches(matcher, "a") {
		t.Fatalf("one non-separator rune must match")
	}
	if matches(matcher, ".") {
		t.Fatalf("a separator must not match")
	}
	if matches(matcher, "ab") {
		t.Fatalf("two runes must not match a single")
	}
}

// Test_Empty_Matches_Only_Empty_String checks the renamed Nothing matcher
// accepts the empty string and nothing else.
func Test_Empty_Matches_Only_Empty_String(t *testing.T) {
	matcher := New_Empty()
	if !matches(matcher, "") {
		t.Fatalf("empty matcher must match the empty string")
	}
	if matches(matcher, "a") {
		t.Fatalf("empty matcher must reject a non-empty string")
	}
}

// Test_Text_Matches_Its_Exact_Literal checks a literal matches in full and that
// Index locates it inside a larger string.
func Test_Text_Matches_Its_Exact_Literal(t *testing.T) {
	matcher := New_Text("abc")
	if !matches(matcher, "abc") {
		t.Fatalf("the exact literal must match")
	}
	if matches(matcher, "abcd") {
		t.Fatalf("a superstring must not match a text literal")
	}
	offset, segments := index_at(matcher, "xabc")
	if offset != 1 {
		t.Fatalf("offset = %d, want 1", offset)
	}
	if !slices.Equal(segments, []int{3}) {
		t.Fatalf("segments = %v, want [3]", segments)
	}
}

// Test_Rune_Count_Bounded_Above_By_Max checks Max accepts runs no longer than
// its limit and rejects longer ones.
func Test_Rune_Count_Bounded_Above_By_Max(t *testing.T) {
	matcher := New_Max(2)
	if !matches(matcher, "ab") {
		t.Fatalf("a run at the limit must match")
	}
	if !matches(matcher, "") {
		t.Fatalf("a shorter run must match")
	}
	if matches(matcher, "abc") {
		t.Fatalf("a run past the limit must not match")
	}
}

// Test_Rune_Count_Bounded_Below_By_Min checks Min accepts runs no shorter than
// its limit and rejects shorter ones.
func Test_Rune_Count_Bounded_Below_By_Min(t *testing.T) {
	matcher := New_Min(2)
	if !matches(matcher, "ab") {
		t.Fatalf("a run at the limit must match")
	}
	if matches(matcher, "a") {
		t.Fatalf("a run below the limit must not match")
	}
}

// Test_Prefix_And_Suffix_Bracket_The_String checks the three bookend matchers
// against their leading, trailing, and combined anchors.
func Test_Prefix_And_Suffix_Bracket_The_String(t *testing.T) {
	if !matches(New_Prefix("ab"), "abc") {
		t.Fatalf("prefix must match a string that starts with it")
	}
	if matches(New_Prefix("ab"), "xabc") {
		t.Fatalf("prefix must not match when it is not leading")
	}
	if !matches(New_Suffix("bc"), "abc") {
		t.Fatalf("suffix must match a string that ends with it")
	}
	both := New_Prefix_Suffix(&New_Prefix_Suffix_Input{Prefix: "a", Suffix: "c"})
	if !matches(both, "axc") {
		t.Fatalf("prefix-suffix must match when both bookends are present")
	}
	if matches(both, "axd") {
		t.Fatalf("prefix-suffix must not match a wrong suffix")
	}
}

// Test_Contains_Detects_Substring_Presence checks Contains and its negation.
func Test_Contains_Detects_Substring_Presence(t *testing.T) {
	if !matches(New_Contains("bc", false), "abcd") {
		t.Fatalf("contains must match when the needle is present")
	}
	if matches(New_Contains("bc", false), "axd") {
		t.Fatalf("contains must not match when the needle is absent")
	}
	if !matches(New_Contains("bc", true), "axd") {
		t.Fatalf("negated contains must match when the needle is absent")
	}
	if matches(New_Contains("bc", true), "abcd") {
		t.Fatalf("negated contains must not match when the needle is present")
	}
}

// Test_Range_Tests_Rune_Interval_Membership checks the inclusive interval and
// its negation, and that a multi-rune string is rejected.
func Test_Range_Tests_Rune_Interval_Membership(t *testing.T) {
	inside := New_Range(&New_Range_Input{Low: 'a', High: 'c', Negated: false})
	if !matches(inside, "b") {
		t.Fatalf("a rune inside the interval must match")
	}
	if matches(inside, "d") {
		t.Fatalf("a rune outside the interval must not match")
	}
	if matches(inside, "bb") {
		t.Fatalf("two runes must not match a single-rune range")
	}
	outside := New_Range(&New_Range_Input{Low: 'a', High: 'c', Negated: true})
	if !matches(outside, "d") {
		t.Fatalf("a negated range must match a rune outside the interval")
	}
}

// Test_List_Tests_Rune_Set_Membership checks set membership and its negation.
func Test_List_Tests_Rune_Set_Membership(t *testing.T) {
	inside := New_List([]rune("abc"), false)
	if !matches(inside, "b") {
		t.Fatalf("a rune in the set must match")
	}
	if matches(inside, "d") {
		t.Fatalf("a rune outside the set must not match")
	}
	outside := New_List([]rune("abc"), true)
	if !matches(outside, "d") {
		t.Fatalf("a negated list must match a rune outside the set")
	}
}

// Test_Row_Matches_A_Fixed_Width_Sequence checks a row accepts only strings of
// its exact rune width whose runes its children match in order.
func Test_Row_Matches_A_Fixed_Width_Sequence(t *testing.T) {
	matcher := New_Row(3, New_Text("ab"), New_Single(nil))
	if !matches(matcher, "abc") {
		t.Fatalf("a matching fixed-width string must match")
	}
	if matches(matcher, "abcd") {
		t.Fatalf("a string wider than the row must not match")
	}
	if matches(matcher, "xyz") {
		t.Fatalf("a same-width string the children reject must not match")
	}
}

// Test_Any_Of_Matches_When_One_Child_Matches checks the disjunction accepts a
// string matched by any child and rejects one matched by none.
func Test_Any_Of_Matches_When_One_Child_Matches(t *testing.T) {
	matcher := New_Any_Of(New_Text("ab"), New_Text("cd"))
	if !matches(matcher, "ab") {
		t.Fatalf("a string matched by the first child must match")
	}
	if !matches(matcher, "cd") {
		t.Fatalf("a string matched by the second child must match")
	}
	if matches(matcher, "ef") {
		t.Fatalf("a string matched by no child must not match")
	}
}

// Test_Every_Of_Matches_When_Every_Child_Matches checks the conjunction accepts
// only strings that satisfy all children.
func Test_Every_Of_Matches_When_Every_Child_Matches(t *testing.T) {
	matcher := New_Every_Of(New_Prefix("a"), New_Suffix("c"))
	if !matches(matcher, "abc") {
		t.Fatalf("a string satisfying every child must match")
	}
	if matches(matcher, "abd") {
		t.Fatalf("a string failing one child must not match")
	}
}

// Test_Btree_Splits_Around_Its_Pivot checks the pivot with empty-required sides
// and with a wildcard left side.
func Test_Btree_Splits_Around_Its_Pivot(t *testing.T) {
	both_empty := New_Btree(&New_Btree_Input{Value: New_Text("x")})
	if !matches(both_empty, "x") {
		t.Fatalf("the pivot alone must match when both sides are required empty")
	}
	if matches(both_empty, "ax") {
		t.Fatalf("text before the pivot must fail an empty-required left")
	}
	left := New_Super()
	left_wild := New_Btree(
		&New_Btree_Input{Value: New_Text("x"), Left: &left})
	if !matches(left_wild, "aax") {
		t.Fatalf("a wildcard left must absorb the text before the pivot")
	}
	if matches(left_wild, "xaa") {
		t.Fatalf("text after the pivot must fail an empty-required right")
	}
}

// Test_Prefix_Any_Bounds_A_Run_After_The_Prefix checks the run after the prefix
// must not cross a separator.
func Test_Prefix_Any_Bounds_A_Run_After_The_Prefix(t *testing.T) {
	matcher := New_Prefix_Any("ab", []rune{'.'})
	if !matches(matcher, "abc") {
		t.Fatalf("a separator-free tail must match")
	}
	if matches(matcher, "ab.c") {
		t.Fatalf("a separator in the tail must not match")
	}
	if matches(matcher, "xab") {
		t.Fatalf("a missing prefix must not match")
	}
}

// Test_Suffix_Any_Bounds_A_Run_Before_The_Suffix checks the run before the
// suffix must not cross a separator.
func Test_Suffix_Any_Bounds_A_Run_Before_The_Suffix(t *testing.T) {
	matcher := New_Suffix_Any("bc", []rune{'.'})
	if !matches(matcher, "abc") {
		t.Fatalf("a separator-free head must match")
	}
	if matches(matcher, "a.bc") {
		t.Fatalf("a separator in the head must not match")
	}
	if matches(matcher, "abcd") {
		t.Fatalf("a missing suffix must not match")
	}
}

// Test_Index_Reports_Offset_And_Segments checks Index reports a match position
// with its segment lengths and a negative offset with nil when it cannot match.
func Test_Index_Reports_Offset_And_Segments(t *testing.T) {
	offset, segments := index_at(New_Prefix("ab"), "xxabyy")
	if offset != 2 {
		t.Fatalf("offset = %d, want 2", offset)
	}
	if !slices.Equal(segments, []int{2, 3, 4}) {
		t.Fatalf("segments = %v, want [2 3 4]", segments)
	}
	missing_offset, missing_segments := index_at(New_Text("z"), "abc")
	if missing_offset != -1 {
		t.Fatalf("a non-match must report offset -1, got %d", missing_offset)
	}
	if missing_segments != nil {
		t.Fatalf("a non-match must report nil segments, got %v", missing_segments)
	}
}

// Test_Rune_Width_Distinguishes_Fixed_From_Variable checks fixed-width matchers
// report their rune count and floating ones report the variable sentinel.
func Test_Rune_Width_Distinguishes_Fixed_From_Variable(t *testing.T) {
	if Rune_Width(New_Single(nil)) != RUNE_WIDTH_ONE {
		t.Fatalf("single must have rune width one")
	}
	if Rune_Width(New_Text("abc")) != 3 {
		t.Fatalf("a three-rune text must have rune width three")
	}
	if Rune_Width(New_Empty()) != RUNE_WIDTH_ZERO {
		t.Fatalf("the empty matcher must have rune width zero")
	}
	if Rune_Width(New_Any(nil)) != RUNE_WIDTH_VARIABLE {
		t.Fatalf("a wildcard must have variable rune width")
	}
}

// Compiler pass and full-compile tests (whitebox).

// Test_Common_Children ports upstream TestCommonChildren: it pins the shared
// leading and trailing children common_children factors from a set of patterns.
func Test_Common_Children(t *testing.T) {
	for _, test_case := range common_children_cases() {
		got_left, got_right := common_children(test_case.Nodes)
		left := &node_slice_pair{First: got_left, Second: test_case.Left}
		if !node_slices_equal(left) {
			t.Errorf("%s: left = %v, want %v",
				test_case.Name, got_left, test_case.Left)
		}
		right := &node_slice_pair{First: got_right, Second: test_case.Right}
		if !node_slices_equal(right) {
			t.Errorf("%s: right = %v, want %v",
				test_case.Name, got_right, test_case.Right)
		}
	}
}

// Test_Glue_Matchers ports upstream TestGlueMatchers: it pins the tightest
// length-and-separator matcher a run of adjacent wildcards glues to.
func Test_Glue_Matchers(t *testing.T) {
	for _, test_case := range glue_matchers_cases() {
		got, err := compile_matchers(test_case.Input)
		if err != nil {
			t.Errorf("%s: %v", test_case.Name, err)
			continue
		}
		if got.String() != test_case.Want.String() {
			t.Errorf("%s: got %s, want %s",
				test_case.Name, got.String(), test_case.Want.String())
		}
	}
}

// Test_Compile_Matchers ports upstream TestCompileMatchers: it pins the search
// tree or row a run of matchers folds into.
func Test_Compile_Matchers(t *testing.T) {
	for _, test_case := range compile_matchers_cases() {
		got, err := compile_matchers(test_case.Input)
		if err != nil {
			t.Errorf("%s: %v", test_case.Name, err)
			continue
		}
		if got.String() != test_case.Want.String() {
			t.Errorf("%s: got %s, want %s",
				test_case.Name, got.String(), test_case.Want.String())
		}
	}
}

// Test_Minimize_Matchers ports upstream TestConvertMatchers: it pins the reduced
// matcher list minimize_matchers produces by gluing the best spans.
func Test_Minimize_Matchers(t *testing.T) {
	for _, test_case := range minimize_matchers_cases() {
		got := minimize_matchers(test_case.Input)
		if matchers_render(got) != matchers_render(test_case.Want) {
			t.Errorf("%s: got %s, want %s", test_case.Name,
				matchers_render(got), matchers_render(test_case.Want))
		}
	}
}

// Test_Compiler ports upstream TestCompiler: it pins the exact compiled matcher
// tree for representative patterns, the strongest correctness gate.
func Test_Compiler(t *testing.T) {
	var cases []compiler_case
	cases = append(cases, compiler_cases_leaves()...)
	cases = append(cases, compiler_cases_width()...)
	cases = append(cases, compiler_cases_affix()...)
	cases = append(cases, compiler_cases_alternation()...)
	cases = append(cases, compiler_cases_row()...)
	for _, test_case := range cases {
		got, err := compile_tree(test_case.Tree, test_case.Separators)
		if err != nil {
			t.Errorf("%s: %v", test_case.Name, err)
			continue
		}
		if got.String() != test_case.Want.String() {
			t.Errorf("%s:\n got %s\nwant %s",
				test_case.Name, got.String(), test_case.Want.String())
		}
	}
}

func common_children_cases() (cases []common_children_case) {
	cases = append(cases, common_children_case{
		Name: "single node shares nothing",
		Nodes: []*Node{
			pattern_node(text_node("a"), text_node("z"), text_node("c"))},
	})
	cases = append(cases, common_children_case{
		Name: "one shared head and tail",
		Nodes: []*Node{
			pattern_node(text_node("a"), text_node("z"), text_node("c")),
			pattern_node(text_node("a"), text_node("b"), text_node("c"))},
		Left:  []*Node{text_node("a")},
		Right: []*Node{text_node("c")},
	})
	cases = append(cases, common_children_case{
		Name: "two shared head and tail",
		Nodes: []*Node{
			pattern_node(text_node("a"), text_node("b"),
				text_node("c"), text_node("d")),
			pattern_node(text_node("a"), text_node("b"),
				text_node("c"), text_node("c"), text_node("d"))},
		Left:  []*Node{text_node("a"), text_node("b")},
		Right: []*Node{text_node("c"), text_node("d")},
	})
	cases = append(cases, common_children_case{
		Name: "two head one tail",
		Nodes: []*Node{
			pattern_node(text_node("a"), text_node("b"), text_node("c")),
			pattern_node(text_node("a"), text_node("b"),
				text_node("b"), text_node("c"))},
		Left:  []*Node{text_node("a"), text_node("b")},
		Right: []*Node{text_node("c")},
	})
	cases = append(cases, common_children_case{
		Name: "shared head no tail",
		Nodes: []*Node{
			pattern_node(text_node("a"), text_node("d")),
			pattern_node(text_node("a"), text_node("d")),
			pattern_node(text_node("a"), text_node("e"))},
		Left:  []*Node{text_node("a")},
		Right: []*Node{},
	})
	return cases
}

func glue_matchers_cases() (cases []matcher_case) {
	separators := []rune{'.'}
	cases = append(cases, matcher_case{
		Name:  "super and single glue to min",
		Input: []Matcher{New_Super(), New_Single(nil)},
		Want:  New_Min(1),
	})
	cases = append(cases, matcher_case{
		Name: "any and single keep the separator",
		Input: []Matcher{
			New_Any(separators), New_Single(separators)},
		Want: New_Every_Of(
			New_Min(1), New_Contains(string(separators), true)),
	})
	cases = append(cases, matcher_case{
		Name: "three singles bound both ends",
		Input: []Matcher{
			New_Single(nil), New_Single(nil), New_Single(nil)},
		Want: New_Every_Of(New_Min(3), New_Max(3)),
	})
	cases = append(cases, matcher_case{
		Name: "negated list and any glue to min and contains",
		Input: []Matcher{
			New_List([]rune{'a'}, true), New_Any([]rune{'a'})},
		Want: New_Every_Of(
			New_Min(1), New_Contains("a", true)),
	})
	return cases
}

func compile_matchers_cases() (cases []matcher_case) {
	separators := []rune{'.'}
	negated_range := New_Range(&New_Range_Input{
		Low: 'a', High: 'c', Negated: true})
	member_list := New_List([]rune{'z', 't', 'e'}, false)
	inner := New_Btree(&New_Btree_Input{
		Value: New_Single(separators),
		Left:  matcher_pointer(New_Super())})
	cases = append(cases, matcher_case{
		Name: "widest static child pivots the tree",
		Input: []Matcher{
			New_Super(), New_Single(separators), New_Text("c")},
		Want: New_Btree(&New_Btree_Input{
			Value: New_Text("c"), Left: matcher_pointer(inner)}),
	})
	cases = append(cases, matcher_case{
		Name: "text between two wildcards",
		Input: []Matcher{
			New_Any(nil), New_Text("c"), New_Any(nil)},
		Want: New_Btree(&New_Btree_Input{
			Value: New_Text("c"),
			Left:  matcher_pointer(New_Any(nil)),
			Right: matcher_pointer(New_Any(nil)),
		}),
	})
	cases = append(cases, matcher_case{
		Name: "all fixed width glue to a row",
		Input: []Matcher{
			negated_range, member_list, New_Text("c"), New_Single(nil)},
		Want: New_Row(4,
			negated_range, member_list, New_Text("c"), New_Single(nil)),
	})
	return cases
}

func minimize_matchers_cases() (cases []minimize_case) {
	negated_range := New_Range(&New_Range_Input{
		Low: 'a', High: 'c', Negated: true})
	member_list := New_List([]rune{'z', 't', 'e'}, false)
	cases = append(cases, minimize_case{
		Name: "static run rows and trailing wildcard stays",
		Input: []Matcher{
			negated_range, member_list, New_Text("c"),
			New_Single(nil), New_Any(nil)},
		Want: []Matcher{
			New_Row(4, negated_range, member_list,
				New_Text("c"), New_Single(nil)),
			New_Any(nil)},
	})
	cases = append(cases, minimize_case{
		Name: "static run rows and wildcard run bounds to min",
		Input: []Matcher{
			negated_range, member_list, New_Text("c"),
			New_Single(nil), New_Any(nil),
			New_Single(nil), New_Single(nil), New_Any(nil)},
		Want: []Matcher{
			New_Row(3, negated_range, member_list, New_Text("c")),
			New_Min(3)},
	})
	return cases
}

func compiler_cases_leaves() (cases []compiler_case) {
	separators := []rune{'.'}
	cases = append(cases, compiler_case{
		Name: "text", Tree: pattern_node(text_node("abc")),
		Want: New_Text("abc"),
	})
	cases = append(cases, compiler_case{
		Name: "any with separators", Tree: pattern_node(any_node()),
		Separators: separators, Want: New_Any(separators),
	})
	cases = append(cases, compiler_case{
		Name: "any without separators", Tree: pattern_node(any_node()),
		Want: New_Super(),
	})
	cases = append(cases, compiler_case{
		Name: "super", Tree: pattern_node(super_node()), Want: New_Super(),
	})
	cases = append(cases, compiler_case{
		Name: "single", Tree: pattern_node(single_node()),
		Separators: separators, Want: New_Single(separators),
	})
	cases = append(cases, compiler_case{
		Name: "negated range",
		Tree: pattern_node(&Node{
			Kind: NODE_KIND_RANGE, Low: 'a', High: 'z', Negated: true}),
		Want: New_Range(&New_Range_Input{
			Low: 'a', High: 'z', Negated: true}),
	})
	cases = append(cases, compiler_case{
		Name: "negated list", Tree: pattern_node(list_node("abc", true)),
		Want: New_List([]rune{'a', 'b', 'c'}, true),
	})
	return cases
}

func compiler_cases_width() (cases []compiler_case) {
	separators := []rune{'.'}
	cases = append(cases, compiler_case{
		Name: "any and singles under separators",
		Tree: pattern_node(
			any_node(), single_node(), single_node(), single_node()),
		Separators: separators,
		Want: New_Every_Of(
			New_Min(3), New_Contains(string(separators), true)),
	})
	cases = append(cases, compiler_case{
		Name: "any and singles without separators",
		Tree: pattern_node(
			any_node(), single_node(), single_node(), single_node()),
		Want: New_Min(3),
	})
	cases = append(cases, compiler_case{
		Name:       "row pivots against a wildcard",
		Tree:       pattern_node(any_node(), text_node("abc"), single_node()),
		Separators: separators,
		Want: New_Btree(&New_Btree_Input{
			Value: New_Row(4,
				New_Text("abc"), New_Single(separators)),
			Left: matcher_pointer(New_Any(separators)),
		}),
	})
	cases = append(cases, compiler_case{
		Name: "super and singles around a text",
		Tree: pattern_node(
			super_node(), single_node(), text_node("abc"), single_node()),
		Separators: separators,
		Want: New_Btree(&New_Btree_Input{
			Value: New_Row(5, New_Single(separators),
				New_Text("abc"), New_Single(separators)),
			Left: matcher_pointer(New_Super()),
		}),
	})
	return cases
}

func compiler_cases_affix() (cases []compiler_case) {
	separators := []rune{'.'}
	cases = append(cases, compiler_case{
		Name: "any then text is a suffix",
		Tree: pattern_node(any_node(), text_node("abc")),
		Want: New_Suffix("abc"),
	})
	cases = append(cases, compiler_case{
		Name: "text then any is a prefix",
		Tree: pattern_node(text_node("abc"), any_node()),
		Want: New_Prefix("abc"),
	})
	cases = append(cases, compiler_case{
		Name: "text any text is a prefix suffix",
		Tree: pattern_node(text_node("abc"), any_node(), text_node("def")),
		Want: New_Prefix_Suffix(&New_Prefix_Suffix_Input{
			Prefix: "abc", Suffix: "def"}),
	})
	cases = append(cases, compiler_case{
		Name: "surrounded text is a contains",
		Tree: pattern_node(any_node(), any_node(), any_node(),
			text_node("abc"), any_node(), any_node()),
		Want: New_Contains("abc", false),
	})
	cases = append(cases, compiler_case{
		Name: "surrounded text under separators splits",
		Tree: pattern_node(any_node(), any_node(), any_node(),
			text_node("abc"), any_node(), any_node()),
		Separators: separators,
		Want: New_Btree(&New_Btree_Input{
			Value: New_Text("abc"),
			Left:  matcher_pointer(New_Any(separators)),
			Right: matcher_pointer(New_Any(separators)),
		}),
	})
	cases = append(cases, compiler_case{
		Name: "supers and singles around text bound each side",
		Tree: pattern_node(super_node(), single_node(),
			text_node("abc"), super_node(), single_node()),
		Want: New_Btree(&New_Btree_Input{
			Value: New_Text("abc"),
			Left:  matcher_pointer(New_Min(1)),
			Right: matcher_pointer(New_Min(1)),
		}),
	})
	return cases
}

func compiler_cases_alternation() (cases []compiler_case) {
	separators := []rune{'.'}
	inner := New_Btree(&New_Btree_Input{
		Value: New_Any_Of(New_Text("z"), New_Text("ab")),
		Right: matcher_pointer(New_Super())})
	cases = append(cases, compiler_case{
		Name: "text then alternation then super",
		Tree: pattern_node(text_node("/"),
			any_of_node(text_node("z"), text_node("ab")), super_node()),
		Separators: separators,
		Want: New_Btree(&New_Btree_Input{
			Value: New_Text("/"), Right: matcher_pointer(inner)}),
	})
	cases = append(cases, compiler_case{
		Name: "text pattern again", Tree: pattern_node(text_node("abc")),
		Want: New_Text("abc"),
	})
	cases = append(cases, compiler_case{
		Name: "nested single alternations reduce to text",
		Tree: pattern_node(any_of_node(pattern_node(any_of_node(
			pattern_node(text_node("abc")))))),
		Want: New_Text("abc"),
	})
	cases = append(cases, compiler_case{
		Name: "shared prefix factored out",
		Tree: pattern_node(any_of_node(
			pattern_node(text_node("abc"), single_node()),
			pattern_node(text_node("abc"), list_node("def", false)),
			pattern_node(text_node("abc")),
			pattern_node(text_node("abc")))),
		Want: New_Btree(&New_Btree_Input{
			Value: New_Text("abc"),
			Right: matcher_pointer(New_Any_Of(
				New_Single(nil),
				New_List([]rune{'d', 'e', 'f'}, false),
				New_Empty())),
		}),
	})
	return cases
}

func compiler_cases_row() (cases []compiler_case) {
	range_low := New_Range(&New_Range_Input{Low: 'a', High: 'z'})
	range_high := New_Range(&New_Range_Input{
		Low: 'a', High: 'x', Negated: true})
	cases = append(cases, compiler_case{
		Name: "two ranges then any split into a row and super",
		Tree: pattern_node(
			&Node{Kind: NODE_KIND_RANGE, Low: 'a', High: 'z'},
			&Node{
				Kind: NODE_KIND_RANGE, Low: 'a', High: 'x', Negated: true},
			any_node()),
		Want: New_Btree(&New_Btree_Input{
			Value: New_Row(2, range_low, range_high),
			Right: matcher_pointer(New_Super())}),
	})
	cases = append(cases, compiler_case{
		Name: "alternation of lists between shared literals is a row",
		Tree: pattern_node(any_of_node(
			pattern_node(text_node("abc"),
				list_node("abc", false), text_node("ghi")),
			pattern_node(text_node("abc"),
				list_node("def", false), text_node("ghi")))),
		Want: New_Row(7, New_Text("abc"),
			New_Any_Of(
				New_List([]rune{'a', 'b', 'c'}, false),
				New_List([]rune{'d', 'e', 'f'}, false)),
			New_Text("ghi")),
	})
	return cases
}

// Joins a matcher slice's string forms so two slices can be compared
// structurally by a single string equality.
func matchers_render(matchers []Matcher) (rendered string) {
	parts := make([]string, 0, len(matchers))
	for _, matcher := range matchers {
		parts = append(parts, matcher.String())
	}
	return strings.Join(parts, "|")
}

// Pairs the two node slices node_slices_equal compares. The input struct exists
// because two []*Node parameters are otherwise swappable at the call site.
type node_slice_pair struct {
	// First is one slice to compare.
	First []*Node
	// Second is the other slice to compare.
	Second []*Node
}

func node_slices_equal(pair *node_slice_pair) (same bool) {
	if len(pair.First) != len(pair.Second) {
		return false
	}
	for index, node := range pair.First {
		if !nodes_equal(&Nodes_Equal_Input{First: node, Second: pair.Second[index]}) {
			return false
		}
	}
	return true
}

func matcher_pointer(matcher Matcher) (pointer *Matcher) {
	return &matcher
}

func pattern_node(children ...*Node) (node *Node) {
	return &Node{Kind: NODE_KIND_PATTERN, Children: children}
}

func any_of_node(children ...*Node) (node *Node) {
	return &Node{Kind: NODE_KIND_ANY_OF, Children: children}
}

func text_node(value string) (node *Node) {
	return &Node{Kind: NODE_KIND_TEXT, Text: value}
}

func list_node(characters string, negated bool) (node *Node) {
	return &Node{
		Kind: NODE_KIND_LIST, Characters: characters, Negated: negated}
}

func any_node() (node *Node) {
	return &Node{Kind: NODE_KIND_ANY}
}

func super_node() (node *Node) {
	return &Node{Kind: NODE_KIND_SUPER}
}

func single_node() (node *Node) {
	return &Node{Kind: NODE_KIND_SINGLE}
}

type matcher_case struct {
	// Name labels the case in failure output.
	Name string
	// Input is the matcher run handed to the pass under test.
	Input []Matcher
	// Want is the matcher the pass must produce.
	Want Matcher
}

type minimize_case struct {
	// Name labels the case in failure output.
	Name string
	// Input is the matcher run handed to minimize_matchers.
	Input []Matcher
	// Want is the reduced matcher list minimize_matchers must produce.
	Want []Matcher
}

type compiler_case struct {
	// Name labels the case in failure output.
	Name string
	// Tree is the syntax tree handed to compile_tree.
	Tree *Node
	// Separators is the separator set threaded through the compile.
	Separators []rune
	// Want is the matcher tree compile_tree must produce.
	Want Matcher
}

type common_children_case struct {
	// Name labels the case in failure output.
	Name string
	// Nodes is the pattern set handed to common_children.
	Nodes []*Node
	// Left is the shared leading run common_children must find.
	Left []*Node
	// Right is the shared trailing run common_children must find.
	Right []*Node
}

// Compiler leaf and affix regression tests (originally the compiler blackbox spec).

// Test_Text_Pattern_Compiles_To_Text checks a bare literal compiles to a text
// matcher that accepts only that literal.
func Test_Text_Pattern_Compiles_To_Text(t *testing.T) {
	matcher, err := compile_tree(pattern_node(text_node("abc")), nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_TEXT {
		t.Fatalf("kind = %v, want text", matcher.Kind)
	}
	if !matches(matcher, "abc") {
		t.Fatalf("abc must match")
	}
	if matches(matcher, "abcd") {
		t.Fatalf("abcd must not match")
	}
}

// Test_Any_With_Separators_Stays_Any checks a star under a separator set stays a
// bounded wildcard that stops at a separator.
func Test_Any_With_Separators_Stays_Any(t *testing.T) {
	matcher, err := compile_tree(pattern_node(any_node()), []rune{'.'})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_ANY {
		t.Fatalf("kind = %v, want any", matcher.Kind)
	}
	if !matches(matcher, "abc") {
		t.Fatalf("abc must match")
	}
	if matches(matcher, "a.b") {
		t.Fatalf("a.b must not match across the separator")
	}
}

// Test_Any_With_No_Separators_Becomes_Super checks a star with nothing to stop it
// widens to a super that accepts every string.
func Test_Any_With_No_Separators_Becomes_Super(t *testing.T) {
	matcher, err := compile_tree(pattern_node(any_node()), nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_SUPER {
		t.Fatalf("kind = %v, want super", matcher.Kind)
	}
	if !matches(matcher, "a.b.c") {
		t.Fatalf("super must match across separators")
	}
}

// Test_Single_Compiles_To_Single checks a question mark compiles to a one-rune
// matcher rejecting the empty string and any longer one.
func Test_Single_Compiles_To_Single(t *testing.T) {
	matcher, err := compile_tree(pattern_node(single_node()), []rune{'.'})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_SINGLE {
		t.Fatalf("kind = %v, want single", matcher.Kind)
	}
	if Rune_Width(matcher) != 1 {
		t.Fatalf("width = %d, want 1", Rune_Width(matcher))
	}
	if !matches(matcher, "a") {
		t.Fatalf("a must match")
	}
	if matches(matcher, ".") {
		t.Fatalf("separator must not match")
	}
	if matches(matcher, "ab") {
		t.Fatalf("two runes must not match")
	}
}

// Test_List_Of_One_Character_Becomes_Text checks a one-member class is rewritten
// to the equivalent text matcher.
func Test_List_Of_One_Character_Becomes_Text(t *testing.T) {
	matcher, err := compile_tree(pattern_node(list_node("a", false)), nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_TEXT {
		t.Fatalf("kind = %v, want text", matcher.Kind)
	}
	if !matches(matcher, "a") {
		t.Fatalf("a must match")
	}
	if matches(matcher, "b") {
		t.Fatalf("b must not match")
	}
}

// Test_Range_Compiles_To_Range checks a negated range accepts exactly the runes
// outside its bounds.
func Test_Range_Compiles_To_Range(t *testing.T) {
	tree := pattern_node(
		&Node{Kind: NODE_KIND_RANGE, Low: 'a', High: 'z', Negated: true})
	matcher, err := compile_tree(tree, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_RANGE {
		t.Fatalf("kind = %v, want range", matcher.Kind)
	}
	if !matches(matcher, "0") {
		t.Fatalf("0 is outside a-z so a negated range must match it")
	}
	if matches(matcher, "b") {
		t.Fatalf("b is inside a-z so a negated range must reject it")
	}
}

// Test_Empty_Pattern_Compiles_To_Empty checks a childless pattern matches only
// the empty string.
func Test_Empty_Pattern_Compiles_To_Empty(t *testing.T) {
	matcher, err := compile_tree(pattern_node(), nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_EMPTY {
		t.Fatalf("kind = %v, want empty", matcher.Kind)
	}
	if !matches(matcher, "") {
		t.Fatalf("empty string must match")
	}
	if matches(matcher, "x") {
		t.Fatalf("non-empty string must not match")
	}
}

// Test_Text_Then_Star_Compiles_To_Prefix checks a literal then a star is a prefix
// test.
func Test_Text_Then_Star_Compiles_To_Prefix(t *testing.T) {
	matcher, err := compile_tree(
		pattern_node(text_node("abc"), any_node()), nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_PREFIX {
		t.Fatalf("kind = %v, want prefix", matcher.Kind)
	}
	if !matches(matcher, "abcdef") {
		t.Fatalf("abcdef must match")
	}
	if matches(matcher, "xabc") {
		t.Fatalf("xabc must not match")
	}
}

// Test_Star_Then_Text_Compiles_To_Suffix checks a star then a literal is a suffix
// test.
func Test_Star_Then_Text_Compiles_To_Suffix(t *testing.T) {
	matcher, err := compile_tree(
		pattern_node(any_node(), text_node("abc")), nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_SUFFIX {
		t.Fatalf("kind = %v, want suffix", matcher.Kind)
	}
	if !matches(matcher, "xxabc") {
		t.Fatalf("xxabc must match")
	}
	if matches(matcher, "abcx") {
		t.Fatalf("abcx must not match")
	}
}

// Test_Text_Star_Text_Compiles_To_Prefix_Suffix checks two literals around a star
// require both ends.
func Test_Text_Star_Text_Compiles_To_Prefix_Suffix(t *testing.T) {
	matcher, err := compile_tree(
		pattern_node(text_node("abc"), any_node(), text_node("def")), nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_PREFIX_SUFFIX {
		t.Fatalf("kind = %v, want prefix-suffix", matcher.Kind)
	}
	if !matches(matcher, "abcXXdef") {
		t.Fatalf("abcXXdef must match")
	}
	if matches(matcher, "abc") {
		t.Fatalf("abc alone must not match")
	}
}

// Test_Surrounded_Text_Compiles_To_Contains checks a literal flanked by wildcards
// is a containment test.
func Test_Surrounded_Text_Compiles_To_Contains(t *testing.T) {
	tree := pattern_node(
		any_node(), any_node(), any_node(),
		text_node("abc"), any_node(), any_node())
	matcher, err := compile_tree(tree, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_CONTAINS {
		t.Fatalf("kind = %v, want contains", matcher.Kind)
	}
	if !matches(matcher, "xxabcyy") {
		t.Fatalf("xxabcyy must match")
	}
	if matches(matcher, "xyz") {
		t.Fatalf("xyz must not match")
	}
}

// Test_Any_Then_Singles_Glue_To_Min checks a wildcard then a run of question
// marks glues into one minimum-length bound.
func Test_Any_Then_Singles_Glue_To_Min(t *testing.T) {
	tree := pattern_node(
		any_node(), single_node(), single_node(), single_node())
	matcher, err := compile_tree(tree, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_MIN {
		t.Fatalf("kind = %v, want min", matcher.Kind)
	}
	if !matches(matcher, "abc") {
		t.Fatalf("three runes must meet the minimum")
	}
	if matches(matcher, "ab") {
		t.Fatalf("two runes must fall short of the minimum")
	}
}

// Test_Btree_Splits_Longest_Static_Child checks a sequence with no gluing form
// splits into a search tree pivoted on its widest fixed matcher.
func Test_Btree_Splits_Longest_Static_Child(t *testing.T) {
	tree := pattern_node(any_node(), text_node("abc"), single_node())
	matcher, err := compile_tree(tree, []rune{'.'})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_BTREE {
		t.Fatalf("kind = %v, want btree", matcher.Kind)
	}
	if !matches(matcher, "Zabcd") {
		t.Fatalf("Zabcd must match")
	}
	if matches(matcher, "Zabc") {
		t.Fatalf("Zabc lacks the trailing single and must not match")
	}
}

// Test_Any_Of_Common_Prefix_Factored checks an alternation whose branches share a
// leading literal matches that head once before the branches diverge.
func Test_Any_Of_Common_Prefix_Factored(t *testing.T) {
	tree := pattern_node(any_of_node(
		pattern_node(text_node("abc"), single_node()),
		pattern_node(text_node("abc"), list_node("def", false)),
		pattern_node(text_node("abc")),
		pattern_node(text_node("abc")),
	))
	matcher, err := compile_tree(tree, nil)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if matcher.Kind != MATCHER_KIND_BTREE {
		t.Fatalf("kind = %v, want btree", matcher.Kind)
	}
	if !matches(matcher, "abc") {
		t.Fatalf("the bare-prefix branch must match abc")
	}
	if !matches(matcher, "abcd") {
		t.Fatalf("a one-rune tail must match")
	}
	if matches(matcher, "abcde") {
		t.Fatalf("no branch admits a two-rune tail")
	}
	if matches(matcher, "xabc") {
		t.Fatalf("the prefix is anchored at the start")
	}
}

// Rune-set search regression tests (originally the runes blackbox spec).

// Test_Index_Any_Finds_First_Present_Member checks the search honours set order
// rather than left-most position, and reports -1 when no member is present.
func Test_Index_Any_Finds_First_Present_Member(t *testing.T) {
	// 'b' precedes 'a' in the set, so 'b' at offset 1 wins over 'a' at offset 0:
	// the result follows set order, not the left-most match.
	got := Index_Any_Runes("ab", []rune{'b', 'a'})
	if got != 1 {
		t.Fatalf("offset = %d, want 1", got)
	}
	if Index_Any_Runes("xyz", []rune{'a', 'b'}) != -1 {
		t.Fatalf("absent members must report -1")
	}
}

// Test_Last_Index_Any_Finds_Last_Position checks the last occurrence of the first
// present set member is returned, and -1 when no member is present.
func Test_Last_Index_Any_Finds_Last_Position(t *testing.T) {
	// 'a' occurs at offsets 0 and 2; the last occurrence at 2 is returned.
	got := Last_Index_Any_Runes("aba", []rune{'a'})
	if got != 2 {
		t.Fatalf("offset = %d, want 2", got)
	}
	if Last_Index_Any_Runes("xyz", []rune{'a', 'b'}) != -1 {
		t.Fatalf("absent members must report -1")
	}
}

// Restored runes-helper tests and microbenchmarks, ported faithfully from
// upstream util/runes/runes_test.go, plus the per-matcher and segments/append
// microbenchmarks ported from upstream match/*_test.go. Callers are the free
// functions the merged package exposes (Index renamed Index_Runes, the method
// Index kept as the free Index over a Matcher, New_Nothing renamed New_Empty).

type runes_index_case struct {
	Source string
	Needle string
	Offset int
}

type runes_equal_case struct {
	Left  string
	Right string
	Same  bool
}

// Test_Index_Runes ports upstream runes TestIndex: Index_Runes returns the rune
// index of the first Needle occurrence, exercising the empty, single-rune,
// equal-length, and scanning branches.
func Test_Index_Runes(t *testing.T) {
	cases := []runes_index_case{
		{Source: "", Needle: "", Offset: 0},
		{Source: "", Needle: "a", Offset: -1},
		{Source: "", Needle: "foo", Offset: -1},
		{Source: "fo", Needle: "foo", Offset: -1},
		{Source: "foo", Needle: "foo", Offset: 0},
		{Source: "oofofoofooo", Needle: "f", Offset: 2},
		{Source: "oofofoofooo", Needle: "foo", Offset: 4},
		{Source: "barfoobarfoo", Needle: "foo", Offset: 3},
		{Source: "foo", Needle: "", Offset: 0},
		{Source: "foo", Needle: "o", Offset: 1},
		{Source: "abcABCabc", Needle: "A", Offset: 3},
		{Source: "x", Needle: "a", Offset: -1},
		{Source: "x", Needle: "x", Offset: 0},
		{Source: "abc", Needle: "a", Offset: 0},
		{Source: "abc", Needle: "b", Offset: 1},
		{Source: "abc", Needle: "c", Offset: 2},
		{Source: "abc", Needle: "x", Offset: -1},
	}
	for case_index, row := range cases {
		source := []rune(row.Source)
		needle := []rune(row.Needle)
		got := Index_Runes(&Index_Runes_Input{Source: source, Needle: needle})
		if got != row.Offset {
			t.Errorf("#%d Index_Runes(%q,%q) = %d, want %d",
				case_index, row.Source, row.Needle, got, row.Offset)
		}
	}
}

// Test_Last_Index ports upstream runes TestLastIndex: Last_Index returns the rune
// index of the last Needle occurrence, with an empty Needle reporting the length
// of Source.
func Test_Last_Index(t *testing.T) {
	cases := []runes_index_case{
		{Source: "", Needle: "", Offset: 0},
		{Source: "", Needle: "a", Offset: -1},
		{Source: "", Needle: "foo", Offset: -1},
		{Source: "fo", Needle: "foo", Offset: -1},
		{Source: "foo", Needle: "foo", Offset: 0},
		{Source: "foo", Needle: "f", Offset: 0},
		{Source: "oofofoofooo", Needle: "f", Offset: 7},
		{Source: "oofofoofooo", Needle: "foo", Offset: 7},
		{Source: "barfoobarfoo", Needle: "foo", Offset: 9},
		{Source: "foo", Needle: "", Offset: 3},
		{Source: "foo", Needle: "o", Offset: 2},
		{Source: "abcABCabc", Needle: "A", Offset: 3},
		{Source: "abcABCabc", Needle: "a", Offset: 6},
	}
	for case_index, row := range cases {
		source := []rune(row.Source)
		needle := []rune(row.Needle)
		got := Last_Index(&Last_Index_Input{Source: source, Needle: needle})
		if got != row.Offset {
			t.Errorf("#%d Last_Index(%q,%q) = %d, want %d",
				case_index, row.Source, row.Needle, got, row.Offset)
		}
	}
}

// Test_Index_Any ports upstream runes TestIndexAny: Index_Any returns the rune
// index of the first Source rune that is also a member of the sought set.
func Test_Index_Any(t *testing.T) {
	dots := "1....2....3....4"
	cases := []runes_index_case{
		{Source: "", Needle: "", Offset: -1},
		{Source: "", Needle: "a", Offset: -1},
		{Source: "", Needle: "abc", Offset: -1},
		{Source: "a", Needle: "", Offset: -1},
		{Source: "a", Needle: "a", Offset: 0},
		{Source: "aaa", Needle: "a", Offset: 0},
		{Source: "abc", Needle: "xyz", Offset: -1},
		{Source: "abc", Needle: "xcz", Offset: 2},
		{Source: "a☺b☻c☹d", Needle: "uvw☻xyz", Offset: 3},
		{Source: "aRegExp*", Needle: ".(|)*+?^$[]", Offset: 7},
		{Source: dots + dots + dots, Needle: " ", Offset: -1},
	}
	for case_index, row := range cases {
		source := []rune(row.Source)
		characters := []rune(row.Needle)
		got := Index_Any(&Index_Any_Input{Source: source, Characters: characters})
		if got != row.Offset {
			t.Errorf("#%d Index_Any(%q,%q) = %d, want %d",
				case_index, row.Source, row.Needle, got, row.Offset)
		}
	}
}

// Test_Equal ports upstream runes TestEqual: Equal reports element-wise rune
// equality of two slices.
func Test_Equal(t *testing.T) {
	cases := []runes_equal_case{
		{Left: "a", Right: "a", Same: true},
		{Left: "a", Right: "b", Same: false},
		{Left: "a☺b☻c☹d", Right: "uvw☻xyz", Same: false},
		{Left: "a☺b☻c☹d", Right: "a☺b☻c☹d", Same: true},
	}
	for case_index, row := range cases {
		left := []rune(row.Left)
		right := []rune(row.Right)
		got := Equal(&Equal_Input{Left: left, Right: right})
		if got != row.Same {
			t.Errorf("#%d Equal(%q,%q) = %v, want %v",
				case_index, row.Left, row.Right, got, row.Same)
		}
	}
}

// BENCHMARK_PATTERN is the fixed 36-rune haystack every matcher Index
// microbenchmark scans, ported from upstream match bench_pattern.
const BENCHMARK_PATTERN = "abcdefghijklmnopqrstuvwxyz0123456789"

// SAMPLE_RUNE is the rune whose UTF-8 byte width the width benchmarks resolve,
// ported from upstream match runeToLen.
const SAMPLE_RUNE = 'q'

// Benchmark_Index_Any measures Any's Index over the benchmark pattern.
func Benchmark_Index_Any(b *testing.B) {
	matcher := New_Any([]rune{'.'})
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Any_Parallel measures Any's Index under parallelism.
func Benchmark_Index_Any_Parallel(b *testing.B) {
	matcher := New_Any([]rune{'.'})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Match_Btree measures Btree's whole-string match over a long fixture.
// Upstream drove it with a fakeMatcher test double (Match always true, fixed
// Len); the merged design forbids interfaces and its Matcher is a concrete
// tagged union, so that double is unrepresentable. Concrete Super and Text
// matchers reproduce the same "always match, return immediately" traversal over
// the identical 80-byte fixture.
func Benchmark_Match_Btree(b *testing.B) {
	value := New_Text("ab")
	left := New_Super()
	right := New_Super()
	tree := New_Btree(&New_Btree_Input{Value: value, Left: &left, Right: &right})
	fixture := "abcdefghij" + "abcdefghij" + "abcdefghij" + "abcdefghij" +
		"abcdefghij" + "abcdefghij" + "abcdefghij" + "abcdefghij"
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			matches(tree, fixture)
		}
	})
}

// Benchmark_Index_Contains measures Contains's Index over the benchmark pattern.
func Benchmark_Index_Contains(b *testing.B) {
	matcher := New_Contains(".", true)
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Contains_Parallel measures Contains's Index under parallelism.
func Benchmark_Index_Contains_Parallel(b *testing.B) {
	matcher := New_Contains(".", true)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_List measures List's Index over the benchmark pattern.
func Benchmark_Index_List(b *testing.B) {
	matcher := New_List([]rune("def"), false)
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_List_Parallel measures List's Index under parallelism.
func Benchmark_Index_List_Parallel(b *testing.B) {
	matcher := New_List([]rune("def"), false)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Max measures Max's Index over the benchmark pattern.
func Benchmark_Index_Max(b *testing.B) {
	matcher := New_Max(10)
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Parallel_Max measures Max's Index under parallelism. Named
// with Max last because the house linter requires the extremum word final.
func Benchmark_Index_Parallel_Max(b *testing.B) {
	matcher := New_Max(10)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Min measures Min's Index over the benchmark pattern.
func Benchmark_Index_Min(b *testing.B) {
	matcher := New_Min(10)
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Parallel_Min measures Min's Index under parallelism. Named
// with Min last because the house linter requires the extremum word final.
func Benchmark_Index_Parallel_Min(b *testing.B) {
	matcher := New_Min(10)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Empty measures Empty's Index (upstream Nothing) over the
// benchmark pattern.
func Benchmark_Index_Empty(b *testing.B) {
	matcher := New_Empty()
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Empty_Parallel measures Empty's Index under parallelism.
func Benchmark_Index_Empty_Parallel(b *testing.B) {
	matcher := New_Empty()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Prefix measures Prefix's Index over the benchmark pattern.
func Benchmark_Index_Prefix(b *testing.B) {
	matcher := New_Prefix("qew")
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Prefix_Parallel measures Prefix's Index under parallelism.
func Benchmark_Index_Prefix_Parallel(b *testing.B) {
	matcher := New_Prefix("qew")
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Prefix_Suffix measures Prefix_Suffix's Index over the pattern.
func Benchmark_Index_Prefix_Suffix(b *testing.B) {
	matcher := New_Prefix_Suffix(&New_Prefix_Suffix_Input{Prefix: "qew", Suffix: "sqw"})
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Prefix_Suffix_Parallel measures Prefix_Suffix's Index in
// parallel.
func Benchmark_Index_Prefix_Suffix_Parallel(b *testing.B) {
	matcher := New_Prefix_Suffix(&New_Prefix_Suffix_Input{Prefix: "qew", Suffix: "sqw"})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Range measures Range's Index over the benchmark pattern.
func Benchmark_Index_Range(b *testing.B) {
	matcher := New_Range(&New_Range_Input{Low: '0', High: '9', Negated: false})
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Range_Parallel measures Range's Index under parallelism.
func Benchmark_Index_Range_Parallel(b *testing.B) {
	matcher := New_Range(&New_Range_Input{Low: '0', High: '9', Negated: false})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Row measures Row's Index over the benchmark pattern.
func Benchmark_Index_Row(b *testing.B) {
	matcher := New_Row(7, New_Text("abc"), New_Text("def"), New_Single(nil))
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Row_Parallel measures Row's Index under parallelism.
func Benchmark_Index_Row_Parallel(b *testing.B) {
	matcher := New_Row(7, New_Text("abc"), New_Text("def"), New_Single(nil))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Single measures Single's Index over the benchmark pattern.
func Benchmark_Index_Single(b *testing.B) {
	matcher := New_Single([]rune{'.'})
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Single_Parallel measures Single's Index under parallelism.
func Benchmark_Index_Single_Parallel(b *testing.B) {
	matcher := New_Single([]rune{'.'})
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Suffix measures Suffix's Index over the benchmark pattern.
func Benchmark_Index_Suffix(b *testing.B) {
	matcher := New_Suffix("qwe")
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Suffix_Parallel measures Suffix's Index under parallelism.
func Benchmark_Index_Suffix_Parallel(b *testing.B) {
	matcher := New_Suffix("qwe")
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Super measures Super's Index over the benchmark pattern.
func Benchmark_Index_Super(b *testing.B) {
	matcher := New_Super()
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Super_Parallel measures Super's Index under parallelism.
func Benchmark_Index_Super_Parallel(b *testing.B) {
	matcher := New_Super()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Index_Text measures Text's Index over the benchmark pattern.
func Benchmark_Index_Text(b *testing.B) {
	matcher := New_Text("foo")
	for b.Loop() {
		index_at(matcher, BENCHMARK_PATTERN)
	}
}

// Benchmark_Index_Text_Parallel measures Text's Index under parallelism.
func Benchmark_Index_Text_Parallel(b *testing.B) {
	matcher := New_Text("foo")
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			index_at(matcher, BENCHMARK_PATTERN)
		}
	})
}

// Benchmark_Append_Merge measures append_merge over two sorted segment lists.
func Benchmark_Append_Merge(b *testing.B) {
	target := []int{0, 1, 3, 6, 7}
	source := []int{0, 1, 3}
	for b.Loop() {
		append_merge(&Append_Merge_Input{Target: target, Source: source})
	}
}

// Benchmark_Append_Merge_Parallel measures append_merge under parallelism.
func Benchmark_Append_Merge_Parallel(b *testing.B) {
	target := []int{0, 1, 3, 6, 7}
	source := []int{0, 1, 3}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			append_merge(&Append_Merge_Input{Target: target, Source: source})
		}
	})
}

// Benchmark_Reverse measures reverse_segments over a four-element slice.
func Benchmark_Reverse(b *testing.B) {
	for b.Loop() {
		reverse_segments([]int{1, 2, 3, 4})
	}
}

// Benchmark_Rune_Width_From_Table measures a table lookup of a rune's UTF-8 byte
// width, ported from upstream BenchmarkRuneLenFromTable.
func Benchmark_Rune_Width_From_Table(b *testing.B) {
	table := make([]int, utf8.MaxRune+1)
	for rune_value := 0; rune_value <= utf8.MaxRune; rune_value++ {
		table[rune_value] = utf8.RuneLen(rune(rune_value))
	}
	total := 0
	for b.Loop() {
		total += table[SAMPLE_RUNE]
	}
	if total < 0 {
		b.Fatal("unreachable: width table indexing overflowed")
	}
}

// Benchmark_Rune_Width_From_UTF8 measures computing a rune's UTF-8 byte width
// with utf8.RuneLen, ported from upstream BenchmarkRuneLenFromUTF8.
func Benchmark_Rune_Width_From_UTF8(b *testing.B) {
	total := 0
	for b.Loop() {
		total += utf8.RuneLen(SAMPLE_RUNE)
	}
	if total < 0 {
		b.Fatal("unreachable: rune width computation overflowed")
	}
}

// Measures allocating a capacity-`capacity_count` int slice with make, ported
// from upstream benchMake. The paired upstream benchPool / BenchmarkSegmentsPool
// family is NOT ported: it builds a sync.Pool, and this package is deterministic,
// so the house linter bans `import "sync"` (which is also why the merged package
// dropped the pooled-segments allocator upstream had).
func benchmark_segments_make(capacity_count int, b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			segments := make([]int, 0, capacity_count)
			if cap(segments) < 0 {
				b.Fatal("unreachable: negative capacity")
			}
		}
	})
}

// The capacity value is fused onto the trailing word (Cap1, not _1) because the
// house Ada_Case rule rejects a bare-number name segment. Each maps 1:1 to an
// upstream BenchmarkSegmentsMake_N.

// Benchmark_Segments_Make_Cap1 measures the make path at capacity 1.
func Benchmark_Segments_Make_Cap1(b *testing.B) { benchmark_segments_make(1, b) }

// Benchmark_Segments_Make_Cap2 measures the make path at capacity 2.
func Benchmark_Segments_Make_Cap2(b *testing.B) { benchmark_segments_make(2, b) }

// Benchmark_Segments_Make_Cap4 measures the make path at capacity 4.
func Benchmark_Segments_Make_Cap4(b *testing.B) { benchmark_segments_make(4, b) }

// Benchmark_Segments_Make_Cap8 measures the make path at capacity 8.
func Benchmark_Segments_Make_Cap8(b *testing.B) { benchmark_segments_make(8, b) }

// Benchmark_Segments_Make_Cap16 measures the make path at capacity 16.
func Benchmark_Segments_Make_Cap16(b *testing.B) { benchmark_segments_make(16, b) }

// Benchmark_Segments_Make_Cap32 measures the make path at capacity 32.
func Benchmark_Segments_Make_Cap32(b *testing.B) { benchmark_segments_make(32, b) }

// Benchmark_Segments_Make_Cap64 measures the make path at capacity 64.
func Benchmark_Segments_Make_Cap64(b *testing.B) { benchmark_segments_make(64, b) }

// Benchmark_Segments_Make_Cap128 measures the make path at capacity 128.
func Benchmark_Segments_Make_Cap128(b *testing.B) { benchmark_segments_make(128, b) }

// Benchmark_Segments_Make_Cap256 measures the make path at capacity 256.
func Benchmark_Segments_Make_Cap256(b *testing.B) { benchmark_segments_make(256, b) }

// Benchmark_Last_Index_Runes measures Last_Index over rune slices.
func Benchmark_Last_Index_Runes(b *testing.B) {
	source := []rune("abcdef")
	needle := []rune("cd")
	for b.Loop() {
		Last_Index(&Last_Index_Input{Source: source, Needle: needle})
	}
}

// Benchmark_Last_Index_Strings measures strings.LastIndex for comparison.
func Benchmark_Last_Index_Strings(b *testing.B) {
	for b.Loop() {
		strings.LastIndex("abcdef", "cd")
	}
}

// Benchmark_Index_Any_Runes measures Index_Any over rune slices.
func Benchmark_Index_Any_Runes(b *testing.B) {
	source := []rune("...b...")
	characters := []rune("abc")
	for b.Loop() {
		Index_Any(&Index_Any_Input{Source: source, Characters: characters})
	}
}

// Benchmark_Index_Any_Strings measures strings.IndexAny for comparison.
func Benchmark_Index_Any_Strings(b *testing.B) {
	for b.Loop() {
		strings.IndexAny("...b...", "abc")
	}
}

// Benchmark_Index_Rune_Runes measures Index_Rune over a rune slice.
func Benchmark_Index_Rune_Runes(b *testing.B) {
	source := []rune("...b...")
	for b.Loop() {
		Index_Rune(source, 'b')
	}
}

// Benchmark_Index_Rune_Strings measures strings.IndexRune for comparison.
func Benchmark_Index_Rune_Strings(b *testing.B) {
	for b.Loop() {
		strings.IndexRune("...b...", 'b')
	}
}

// Benchmark_Index_Runes measures Index_Runes over rune slices.
func Benchmark_Index_Runes(b *testing.B) {
	source := []rune("abcdef")
	needle := []rune("cd")
	for b.Loop() {
		Index_Runes(&Index_Runes_Input{Source: source, Needle: needle})
	}
}

// Benchmark_Index_Strings measures strings.Index for comparison.
func Benchmark_Index_Strings(b *testing.B) {
	for b.Loop() {
		strings.Index("abcdef", "cd")
	}
}

// Benchmark_Equal_Runes measures Equal on two identical rune slices.
func Benchmark_Equal_Runes(b *testing.B) {
	left := []rune("abc")
	right := []rune("abc")
	for b.Loop() {
		if Equal(&Equal_Input{Left: left, Right: right}) {
			continue
		}
	}
}

// Benchmark_Equal_Strings measures string equality on two equal strings.
func Benchmark_Equal_Strings(b *testing.B) {
	left := "abc"
	right := "abc"
	for b.Loop() {
		if left == right {
			continue
		}
	}
}

// Benchmark_Not_Equal_Runes measures Equal on two differing rune slices.
func Benchmark_Not_Equal_Runes(b *testing.B) {
	left := []rune("abc")
	right := []rune("abcd")
	for b.Loop() {
		if Equal(&Equal_Input{Left: left, Right: right}) {
			continue
		}
	}
}

// Benchmark_Not_Equal_Strings measures string equality on two differing strings.
func Benchmark_Not_Equal_Strings(b *testing.B) {
	left := "abc"
	right := "abcd"
	for b.Loop() {
		if left == right {
			continue
		}
	}
}
