// Package glob compiles a glob pattern into a reusable matcher and tests strings
// against it. It is a dependency-injected port of github.com/gobwas/glob (MIT,
// Sergey Kamardin; see LICENSE.mit.kamardin), rewritten to house conventions:
// the upstream Glob interface is gone (the linter bans interfaces), so Compile
// returns a concrete Pattern, and Match is a free function rather than a method.
//
// The pattern syntax is:
//
//	pattern:
//	    { term }
//	term:
//	    `*`         matches any sequence of non-separator characters
//	    `**`        matches any sequence of characters
//	    `?`         matches any single non-separator character
//	    `[` [ `!` ] { character-range } `]`   character class (non-empty)
//	    `{` pattern-list `}`                   pattern alternatives
//	    c           matches character c (c is none of `*?\[{}`)
//	    `\` c       matches character c
//	character-range:
//	    c           matches character c (c is none of `\-]`)
//	    lo `-` hi   matches character c for lo <= c <= hi
//	pattern-list:
//	    pattern { `,` pattern }               comma-separated patterns
//
// The one attacker-reachable risk in a matcher over untrusted patterns is
// unbounded parse/compile/match recursion depth from nested `{...}` groups — a
// stack-overflow denial of service. Parse depth caps the AST depth, which caps
// the compiled-tree and match-time recursion depth, so a single explicit bound
// covers all three: Compile rejects any pattern whose nesting exceeds
// PATTERN_DEPTH_MAX. See PATTERN_DEPTH_MAX.
//
// House layout folds the former runes, syntax, match, and compiler subpackages
// into this single leaf package: the linter's fragmentation rule allots one
// source file per 10000 lines, so all non-test source lives in glob.go. Where
// upstream used an interface the linter bans one, so the parser drives a
// concrete lexer, matchers collapse into the Matcher tagged union keyed by
// Matcher_Kind, and the AST node payload is keyed by Node_Kind; the whole
// shared/text/glob subtree is exempt from the recursion ban so every pass
// recurses into its children exactly as upstream did.
package glob

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Pattern is a compiled glob pattern. The zero value matches nothing useful; use
// Compile to build one.
type Pattern struct {
	// Matcher is the compiled matcher tree Match evaluates. It is a pointer so a
	// compiled Pattern is a word wide and passing it never copies the matcher; it
	// is exported only because the house linter forbids unexported struct fields;
	// callers should treat it as opaque and go through Match.
	Matcher *Matcher
	// Evaluate is the root matcher's kind-specific evaluator, resolved once at
	// compile time. Match calls it directly, so a whole-pattern terminal matcher
	// (Text, Prefix, …) dispatches through one indirect call rather than the
	// Kind switch of matcher_matches — the switch loads Kind and branches on every
	// call, which is pure overhead once the kind is already known. Recursion inside
	// a composite still goes through matcher_matches, whose switch stays the fast
	// path there. Both fields are exported only because the linter forbids
	// unexported struct fields.
	Evaluate func(matcher *Matcher, text string) (matched bool)
}

// Compile parses pattern and compiles it into a Pattern. The optional separators
// are the runes `*` and `?` refuse to cross (typically the path separator); with
// none, `*` behaves like `**`. It returns an error for a malformed pattern or one
// whose `{...}` nesting exceeds PATTERN_DEPTH_MAX.
func Compile(pattern string, separators ...rune) (compiled Pattern, err error) {
	tree, err := Parse(pattern)
	if err != nil {
		return Pattern{}, err
	}
	matcher, err := compile_tree(tree, separators)
	if err != nil {
		return Pattern{}, err
	}
	return Pattern{Matcher: &matcher, Evaluate: select_evaluator(matcher.Kind)}, nil
}

// Must_Compile is Compile without the error return: it panics when Compile would
// fail, for patterns fixed at build time rather than supplied at runtime.
func Must_Compile(pattern string, separators ...rune) (compiled Pattern) {
	compiled, err := Compile(pattern, separators...)
	if err != nil {
		panic(fmt.Sprintf("glob: Must_Compile(%q): %v", pattern, err))
	}
	return compiled
}

// Match reports whether text satisfies the compiled pattern.
func Match(compiled Pattern, text string) (matched bool) {
	return compiled.Evaluate(compiled.Matcher, text)
}

// Quote_Meta returns text with every glob metacharacter backslash-escaped, so the
// result compiles to a pattern that matches text literally. For example
// Quote_Meta(`{foo*}`) returns `\{foo\*\}`.
func Quote_Meta(text string) (quoted string) {
	// A byte loop is correct because every metacharacter is ASCII; worst case
	// every byte is escaped, so twice the input length is always enough.
	buffer := make([]byte, 2*len(text))
	position := 0
	for index := 0; index < len(text); index++ {
		if Special(text[index]) {
			buffer[position] = '\\'
			position++
		}
		buffer[position] = text[index]
		position++
	}
	return string(buffer[0:position])
}

// The parser and lexer below turn a glob pattern into an abstract syntax tree the
// compiler walks. They are a dependency-injected port of the syntax, syntax/ast, and
// syntax/lexer packages from github.com/gobwas/glob (MIT, Sergey Kamardin; see
// LICENSE.mit.kamardin).
//
// Upstream split the front end across three packages wired together by a Lexer
// interface; this port collapses them into one. The house linter bans interfaces
// outside generic constraints, so the parser drives the concrete lexer directly
// rather than through an abstraction, and the three-package seam the interface
// justified no longer earns its keep. For the same reason upstream's Node.Value
// interface{} — a type switch over Text, List, and Range payload structs — became
// typed fields tagged by Node.Kind here: the compiler reads them with a
// `switch node.Kind` rather than a banned type assertion.
//
// The linter also requires every package-level type to be exported and one source
// file per package, so the lexer and its tokens carry exported names and share this
// file even though the compiler consumes only Node and Node_Kind. Treat Lexer, Token, and
// Token_Kind as internal to the front end.

// PATTERN_DEPTH_MAX bounds how deeply {...} alternatives may nest. Parsing a group
// recurses one frame per nesting level, and the compile and match passes that later
// walk the tree recurse to the same depth, so an adversarial pattern such as
// "{{{...}}}" nested millions deep is a stack-overflow denial of service. Parse
// rejects any pattern that nests past this bound instead of recursing into it. 1000
// is far above the depth any real glob needs yet far below where Go's goroutine
// stack is at risk.
const PATTERN_DEPTH_MAX int = 1000

// Special reports whether character is a glob metacharacter — one of * ? \ [ ] { }
// — that a caller must escape to match it literally. The top-level QuoteMeta uses
// it. Comma, `!`, and `-` are deliberately absent: they are metacharacters only
// inside a group or class, not on their own.
func Special(character byte) (special bool) {
	switch character {
	case '*', '?', '\\', '[', ']', '{', '}':
		return true
	default:
		return false
	}
}

// Parse compiles pattern into a parse tree rooted at a NODE_KIND_PATTERN node, or returns
// an error on a malformed class or range, or on a pattern that nests {...}
// alternatives past PATTERN_DEPTH_MAX. That bound is a hard security limit: the
// recursive descent here — and the compile and match passes that later walk the
// tree — recurse one frame per nesting level, so an unbounded pattern would overflow
// the stack. Recursion is intentional and permitted for this parser; only its depth
// is capped.
func Parse(pattern string) (tree *Node, err error) {
	tree, err = parse_lexer(new_lexer(pattern))
	return tree, err
}

// Node_Kind tags a Node with the grammar production it represents, so a consumer reads a
// node's payload by switching on Kind rather than a banned type assertion.
type Node_Kind int

// NODE_KIND_EMPTY is the zero value, mapping to upstream KindNothing: a node that matches
// only the empty string. The parser never emits it; it exists so a consumer can
// represent an empty match.
const NODE_KIND_EMPTY Node_Kind = 0

// NODE_KIND_PATTERN is an ordered sequence of child nodes — the tree root, and each
// alternative inside a {...} group.
const NODE_KIND_PATTERN Node_Kind = 1

// NODE_KIND_LIST is a [...] character class holding its members in Characters: any one of
// those runes, or any rune NOT among them when Negated.
const NODE_KIND_LIST Node_Kind = 2

// NODE_KIND_RANGE is a [a-z] character range spanning Low to High inclusive: any one rune
// in that span, or any rune outside it when Negated.
const NODE_KIND_RANGE Node_Kind = 3

// NODE_KIND_TEXT is a literal run of runes held in Text, matched verbatim.
const NODE_KIND_TEXT Node_Kind = 4

// NODE_KIND_ANY is a single `*`: any run of runes up to the next separator.
const NODE_KIND_ANY Node_Kind = 5

// NODE_KIND_SUPER is `**`: any run of runes, separators included.
const NODE_KIND_SUPER Node_Kind = 6

// NODE_KIND_SINGLE is `?`: exactly one non-separator rune.
const NODE_KIND_SINGLE Node_Kind = 7

// NODE_KIND_ANY_OF is a {...} group whose NODE_KIND_PATTERN children are the alternatives, any
// one of which may match.
const NODE_KIND_ANY_OF Node_Kind = 8

// String names the kind for debugging and Node.String output. Satisfies
// fmt.Stringer.
func (kind Node_Kind) String() (name string) {
	switch kind {
	case NODE_KIND_EMPTY:
		return "Empty"
	case NODE_KIND_PATTERN:
		return "Pattern"
	case NODE_KIND_LIST:
		return "List"
	case NODE_KIND_RANGE:
		return "Range"
	case NODE_KIND_TEXT:
		return "Text"
	case NODE_KIND_ANY:
		return "Any"
	case NODE_KIND_SUPER:
		return "Super"
	case NODE_KIND_SINGLE:
		return "Single"
	case NODE_KIND_ANY_OF:
		return "AnyOf"
	default:
		return ""
	}
}

// Node is one vertex of the parse tree. Kind selects which payload fields carry
// meaning; the rest stay zero. Children and Parent form the tree the compiler walks,
// doubly linked so a walk can ascend as well as descend.
type Node struct {
	// Kind tags which grammar production this node is, and thus which of the payload
	// fields below are meaningful.
	Kind Node_Kind
	// Children are the ordered sub-nodes: a Pattern's sequence, or an AnyOf's
	// alternatives. Empty for leaf kinds.
	Children []*Node
	// Parent is the enclosing node, nil only for the root. It lets the parser ascend
	// out of a nested group without a side stack.
	Parent *Node
	// Text is the literal run for NODE_KIND_TEXT, matched verbatim.
	Text string
	// Characters holds the members of a NODE_KIND_LIST class — the runes the class admits.
	Characters string
	// Negated inverts a NODE_KIND_LIST or NODE_KIND_RANGE so it matches every rune it does NOT
	// cover. Set by a leading `!` inside the brackets.
	Negated bool
	// Low is the inclusive lower bound rune of a NODE_KIND_RANGE.
	Low rune
	// High is the inclusive upper bound rune of a NODE_KIND_RANGE.
	High rune
}

// String renders the node and its subtree as "Kind =payload [child, ...]" for test
// output and go-doc examples; the parse path never calls it. Satisfies fmt.Stringer.
func (node *Node) String() (rendered string) {
	var builder strings.Builder
	builder.WriteString(node.Kind.String())
	payload := node_payload_string(node)
	if payload != "" {
		builder.WriteString(" =")
		builder.WriteString(payload)
	}
	if len(node.Children) > 0 {
		builder.WriteString(" [")
		for child_index, child := range node.Children {
			if child_index > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(child.String())
		}
		builder.WriteString("]")
	}
	return builder.String()
}

// Builds the payload fragment of Node.String for the kinds that carry one; the other
// kinds contribute nothing.
func node_payload_string(node *Node) (payload string) {
	switch node.Kind {
	case NODE_KIND_TEXT:
		return node.Text
	case NODE_KIND_LIST:
		return fmt.Sprintf("negated=%t %q", node.Negated, node.Characters)
	case NODE_KIND_RANGE:
		return fmt.Sprintf("negated=%t %q-%q", node.Negated, node.Low, node.High)
	default:
		return ""
	}
}

// Builds a node of the given kind and inserts the children under it, wiring their
// Parent links. Payload-bearing leaves (Text, List, Range) are written as keyed Node
// literals directly, since their fields vary by kind.
func new_node(kind Node_Kind, children ...*Node) (node *Node) {
	node = &Node{Kind: kind}
	insert(node, children...)
	return node
}

// Appends children to parent and back-links each child to parent, keeping the
// doubly-linked shape the parser relies on to ascend out of a nested group.
func insert(parent *Node, children ...*Node) {
	parent.Children = append(parent.Children, children...)
	for _, child := range children {
		child.Parent = parent
	}
}

// Drives the concrete lexer to build a NODE_KIND_PATTERN-rooted tree. Split from Parse so
// a test can seed a Lexer's buffer directly rather than a source string. depth starts
// at 0; each {...} adds one level, capped by PATTERN_DEPTH_MAX.
func parse_lexer(lexer *Lexer) (tree *Node, err error) {
	root := &Node{Kind: NODE_KIND_PATTERN}
	_, err = parse_pattern(lexer, root, 0)
	if err != nil {
		return nil, err
	}
	return root, nil
}

// Reads tokens into pattern until it reaches a terminator — end of input, a `,`
// separator, or a `}` group close — and returns which terminator stopped it, so a
// caller inside a group knows whether more alternatives follow. The loop exits only
// by returning; its condition is a stand-in for the banned bare `for {}`.
func parse_pattern(lexer *Lexer, pattern *Node, depth int) (terminator Token_Kind, err error) {
	var done bool
	for !done {
		token := lex_next(lexer)
		switch token.Kind {
		case TOKEN_EOF:
			return TOKEN_EOF, nil
		case TOKEN_ERROR:
			return TOKEN_ERROR, errors.New(token.Raw)
		case TOKEN_SEPARATOR:
			return TOKEN_SEPARATOR, nil
		case TOKEN_TERMS_CLOSE:
			return TOKEN_TERMS_CLOSE, nil
		case TOKEN_TEXT:
			insert(pattern, &Node{Kind: NODE_KIND_TEXT, Text: token.Raw})
		case TOKEN_ANY:
			insert(pattern, &Node{Kind: NODE_KIND_ANY})
		case TOKEN_SUPER:
			insert(pattern, &Node{Kind: NODE_KIND_SUPER})
		case TOKEN_SINGLE:
			insert(pattern, &Node{Kind: NODE_KIND_SINGLE})
		case TOKEN_RANGE_OPEN:
			err = parse_range(lexer, pattern)
			if err != nil {
				return TOKEN_ERROR, err
			}
		case TOKEN_TERMS_OPEN:
			err = parse_any_of(lexer, pattern, depth)
			if err != nil {
				return TOKEN_ERROR, err
			}
		default:
			return TOKEN_ERROR, fmt.Errorf(
				"unexpected %s token %q", token.Kind, token.Raw)
		}
	}
	return TOKEN_EOF, nil
}

// Consumes a {...} group after its opening `{`, attaching a NODE_KIND_ANY_OF node to
// parent with one NODE_KIND_PATTERN child per comma-separated alternative. It deepens the
// nesting by one and rejects the pattern once depth passes PATTERN_DEPTH_MAX, the
// recursion-depth security bound. An unterminated group ends without error, matching
// upstream.
func parse_any_of(lexer *Lexer, parent *Node, depth int) (err error) {
	depth++
	if depth > PATTERN_DEPTH_MAX {
		return fmt.Errorf(
			"pattern nests {...} deeper than the limit of %d", PATTERN_DEPTH_MAX)
	}
	any_of := &Node{Kind: NODE_KIND_ANY_OF}
	insert(parent, any_of)
	var done bool
	for !done {
		alternative := &Node{Kind: NODE_KIND_PATTERN}
		insert(any_of, alternative)
		var terminator Token_Kind
		terminator, err = parse_pattern(lexer, alternative, depth)
		if err != nil {
			return err
		}
		if terminator != TOKEN_SEPARATOR {
			done = true
		}
	}
	return nil
}

// Consumes a [...] class after its opening `[`, attaching a NODE_KIND_RANGE or NODE_KIND_LIST
// leaf to pattern. The lexer has already split the body into tokens; this assembles
// them and rejects a class that is neither a clean lo-hi range nor a clean member
// list.
func parse_range(lexer *Lexer, pattern *Node) (err error) {
	var negated bool
	var low rune
	var high rune
	var characters string
	var done bool
	for !done {
		token := lex_next(lexer)
		switch token.Kind {
		case TOKEN_EOF:
			return errors.New("unexpected end of input inside a character class")
		case TOKEN_ERROR:
			return errors.New(token.Raw)
		case TOKEN_NOT:
			negated = true
		case TOKEN_RANGE_LOW:
			low, err = range_bound(token.Raw)
			if err != nil {
				return err
			}
		case TOKEN_RANGE_BETWEEN:
			// The dash is structural; the bounds around it carry the data.
		case TOKEN_RANGE_HIGH:
			high, err = range_bound(token.Raw)
			if err != nil {
				return err
			}
			if high < low {
				return errors.New("character class range ends below its start")
			}
		case TOKEN_TEXT:
			characters = token.Raw
		case TOKEN_RANGE_CLOSE:
			// A class with both bounds is a range; with members it is a list. It
			// must be exactly one, never both and never neither.
			is_range := low != 0 && high != 0
			is_characters := characters != ""
			if is_range == is_characters {
				return errors.New(
					"character class is neither a range nor a member list")
			}
			if is_range {
				insert(pattern, &Node{
					Kind:    NODE_KIND_RANGE,
					Low:     low,
					High:    high,
					Negated: negated,
				})
			} else {
				insert(pattern, &Node{
					Kind:       NODE_KIND_LIST,
					Characters: characters,
					Negated:    negated,
				})
			}
			done = true
		}
	}
	return nil
}

// Decodes token_raw as exactly one rune, the low or high bound of a [a-z] range. The
// lexer can hand over multi-rune raw text for a malformed class, so a raw holding
// more than one rune is rejected.
func range_bound(token_raw string) (bound rune, err error) {
	decoded, width := utf8.DecodeRuneInString(token_raw)
	if len(token_raw) > width {
		return 0, errors.New("a character class range bound must be a single rune")
	}
	return decoded, nil
}

// Token_Kind tags a Token with its lexical category. The lexer is internal to the
// front end; a consumer works with the Node tree, not tokens.
type Token_Kind int

// TOKEN_EOF marks the end of the pattern; the lexer returns it indefinitely once the
// input is exhausted.
const TOKEN_EOF Token_Kind = 0

// TOKEN_ERROR carries a scan failure, its message in Token.Raw.
const TOKEN_ERROR Token_Kind = 1

// TOKEN_TEXT is a literal run of runes, escapes already resolved.
const TOKEN_TEXT Token_Kind = 2

// TOKEN_ANY is a single `*`.
const TOKEN_ANY Token_Kind = 3

// TOKEN_SUPER is `**`.
const TOKEN_SUPER Token_Kind = 4

// TOKEN_SINGLE is `?`.
const TOKEN_SINGLE Token_Kind = 5

// TOKEN_NOT is the `!` that negates a character class.
const TOKEN_NOT Token_Kind = 6

// TOKEN_SEPARATOR is the `,` between alternatives inside a {...} group.
const TOKEN_SEPARATOR Token_Kind = 7

// TOKEN_RANGE_OPEN is the `[` opening a character class.
const TOKEN_RANGE_OPEN Token_Kind = 8

// TOKEN_RANGE_CLOSE is the `]` closing a character class.
const TOKEN_RANGE_CLOSE Token_Kind = 9

// TOKEN_RANGE_LOW is the low bound rune of a [a-z] range.
const TOKEN_RANGE_LOW Token_Kind = 10

// TOKEN_RANGE_HIGH is the high bound rune of a [a-z] range.
const TOKEN_RANGE_HIGH Token_Kind = 11

// TOKEN_RANGE_BETWEEN is the `-` separating the bounds of a [a-z] range.
const TOKEN_RANGE_BETWEEN Token_Kind = 12

// TOKEN_TERMS_OPEN is the `{` opening an alternatives group.
const TOKEN_TERMS_OPEN Token_Kind = 13

// TOKEN_TERMS_CLOSE is the `}` closing an alternatives group.
const TOKEN_TERMS_CLOSE Token_Kind = 14

// String names the token kind for error messages and test output. Satisfies
// fmt.Stringer.
func (kind Token_Kind) String() (name string) {
	switch kind {
	case TOKEN_EOF:
		return "eof"
	case TOKEN_ERROR:
		return "error"
	case TOKEN_TEXT:
		return "text"
	case TOKEN_ANY:
		return "any"
	case TOKEN_SUPER:
		return "super"
	case TOKEN_SINGLE:
		return "single"
	case TOKEN_NOT:
		return "not"
	case TOKEN_SEPARATOR:
		return "separator"
	case TOKEN_RANGE_OPEN:
		return "range_open"
	case TOKEN_RANGE_CLOSE:
		return "range_close"
	case TOKEN_RANGE_LOW:
		return "range_low"
	case TOKEN_RANGE_HIGH:
		return "range_high"
	case TOKEN_RANGE_BETWEEN:
		return "range_between"
	case TOKEN_TERMS_OPEN:
		return "terms_open"
	case TOKEN_TERMS_CLOSE:
		return "terms_close"
	default:
		return "undefined"
	}
}

// Token is one lexical unit: its category and the raw source runes it spans.
type Token struct {
	// Kind is the token's lexical category.
	Kind Token_Kind
	// Raw is the source text the token spans; for TOKEN_ERROR it is the message.
	Raw string
}

// Lexer scans a glob pattern into tokens on demand. It is exported only because the
// linter requires every package-level type to be; the parser is its sole caller.
type Lexer struct {
	// Data is the pattern being scanned.
	Data string
	// Position is the byte offset of the next rune to read in Data.
	Position int
	// Error latches the first scan failure; once set, every read reports it.
	Error error
	// Buffer holds tokens produced ahead of the reader — one fetch of a range emits
	// several — and is drained front to back.
	Buffer []Token
	// Terms_Level counts the open {...} groups, so a `,` or `}` is a separator or a
	// group close only inside a group and literal text outside one.
	Terms_Level int
	// Last_Rune is the most recently read rune, kept so an unread can restore it.
	Last_Rune rune
	// Last_Rune_Size is Last_Rune's byte width, the amount an unread rewinds by.
	Last_Rune_Size int
	// Has_Rune is true when a rune has been unread and awaits re-reading.
	Has_Rune bool
}

// Builds a lexer over source with a small token buffer preallocated for the common
// case where one fetch emits a handful of tokens.
func new_lexer(source string) (state *Lexer) {
	state = &Lexer{Data: source, Buffer: make([]Token, 0, 4)}
	return state
}

// Next returns the next token, draining the buffer first and fetching more when it
// empties. A latched error is reported as a TOKEN_ERROR on every call. A fetch may
// enqueue nothing (a trailing escape yields no text), so the loop repeats until a
// token is available; end of input always eventually enqueues TOKEN_EOF.
func lex_next(state *Lexer) (result Token) {
	if state.Error != nil {
		return Token{Kind: TOKEN_ERROR, Raw: state.Error.Error()}
	}
	for buffer_empty(state) {
		lex_fetch_item(state)
		if state.Error != nil {
			return Token{Kind: TOKEN_ERROR, Raw: state.Error.Error()}
		}
	}
	return buffer_shift(state)
}

// Reports whether the lexer's lookahead buffer is drained.
func buffer_empty(state *Lexer) (empty bool) {
	return len(state.Buffer) == 0
}

// Appends a token to the back of the lexer's lookahead buffer.
func buffer_push(state *Lexer, value Token) {
	state.Buffer = append(state.Buffer, value)
}

// Removes and returns the front token of the lexer's lookahead buffer.
func buffer_shift(state *Lexer) (front Token) {
	front = state.Buffer[0]
	state.Buffer = state.Buffer[1:]
	return front
}

// Reads the rune at the cursor without consuming it, returning rune 0 and width 0 at
// end of input. A malformed UTF-8 sequence latches an error and reports end of
// input. Rune 0 doubles as the end-of-input sentinel, following upstream: a literal
// NUL in a glob pattern is not meaningful.
func lex_peek(state *Lexer) (rune_value rune, width int) {
	if state.Position == len(state.Data) {
		return 0, 0
	}
	rune_value, width = utf8.DecodeRuneInString(state.Data[state.Position:])
	if rune_value == utf8.RuneError {
		lex_error(state, "could not decode a UTF-8 rune")
		return 0, 0
	}
	return rune_value, width
}

// Reads and consumes the next rune, advancing the cursor. A previously unread rune is
// returned first, re-advancing the cursor over it.
func lex_read(state *Lexer) (rune_value rune) {
	if state.Has_Rune {
		state.Has_Rune = false
		lex_seek(state, state.Last_Rune_Size)
		return state.Last_Rune
	}
	var width int
	rune_value, width = lex_peek(state)
	lex_seek(state, width)
	state.Last_Rune = rune_value
	state.Last_Rune_Size = width
	return rune_value
}

// Moves the cursor by width bytes; a negative width rewinds it.
func lex_seek(state *Lexer, width int) {
	state.Position += width
}

// Pushes the last read rune back so the next read returns it again. Two unreads in a
// row are impossible by construction and latch an error.
func lex_unread(state *Lexer) {
	if state.Has_Rune {
		lex_error(state, "could not unread a second rune")
		return
	}
	lex_seek(state, -state.Last_Rune_Size)
	state.Has_Rune = true
}

// Latches message as the lexer's error unless one is already set, so the first
// failure is the one reported.
func lex_error(state *Lexer, message string) {
	if state.Error != nil {
		return
	}
	state.Error = errors.New(message)
}

// Reports whether the lexer is inside at least one open {...} group.
func lex_in_terms(state *Lexer) (inside bool) {
	return state.Terms_Level > 0
}

// Returns the runes that end a text run outside a {...} group: the openers of the
// other constructs.
func text_breakers() (breakers []rune) {
	return []rune{'?', '*', '[', '{'}
}

// Returns the runes that end a text run inside a {...} group: the text breakers plus
// the group's own close and separator.
func terms_breakers() (breakers []rune) {
	return []rune{'?', '*', '[', '{', '}', ','}
}

// Returns the active text breakers for the current context.
func lex_breakers(state *Lexer) (breakers []rune) {
	if lex_in_terms(state) {
		return terms_breakers()
	}
	return text_breakers()
}

// Reads one construct from the cursor and enqueues its token(s). A metacharacter
// becomes its own token; `[` also scans the whole class body; anything else is
// scanned as a text run up to the next breaker.
func lex_fetch_item(state *Lexer) {
	rune_value := lex_read(state)
	switch {
	case rune_value == 0:
		buffer_push(state, Token{Kind: TOKEN_EOF, Raw: ""})
	case rune_value == '{':
		state.Terms_Level++
		buffer_push(state, Token{Kind: TOKEN_TERMS_OPEN, Raw: string(rune_value)})
	case rune_value == ',' && lex_in_terms(state):
		buffer_push(state, Token{Kind: TOKEN_SEPARATOR, Raw: string(rune_value)})
	case rune_value == '}' && lex_in_terms(state):
		buffer_push(state, Token{Kind: TOKEN_TERMS_CLOSE, Raw: string(rune_value)})
		state.Terms_Level--
	case rune_value == '[':
		buffer_push(state, Token{Kind: TOKEN_RANGE_OPEN, Raw: string(rune_value)})
		lex_fetch_range(state)
	case rune_value == '?':
		buffer_push(state, Token{Kind: TOKEN_SINGLE, Raw: string(rune_value)})
	case rune_value == '*':
		lex_fetch_any(state, rune_value)
	default:
		lex_unread(state)
		lex_fetch_text(state, lex_breakers(state))
	}
}

// Enqueues TOKEN_SUPER for `**` or TOKEN_ANY for a lone `*`, given the first `*` is
// already read.
func lex_fetch_any(state *Lexer, first rune) {
	if lex_read(state) == '*' {
		buffer_push(state, Token{Kind: TOKEN_SUPER, Raw: string(first) + string(first)})
		return
	}
	lex_unread(state)
	buffer_push(state, Token{Kind: TOKEN_ANY, Raw: string(first)})
}

// Scans the body of a [...] class after the opening `[` and enqueues its tokens: an
// optional `!`, then either a lo-`-`-hi range or a text run of members, closed by
// `]`. Malformed input latches an error.
func lex_fetch_range(state *Lexer) {
	var want_high bool
	var want_close bool
	var seen_not bool
	var done bool
	for !done {
		rune_value := lex_read(state)
		switch {
		case rune_value == 0:
			lex_error(state, "unexpected end of input inside a character class")
			done = true
		case want_close:
			lex_fetch_range_close(state, rune_value)
			done = true
		case want_high:
			buffer_push(state, Token{Kind: TOKEN_RANGE_HIGH, Raw: string(rune_value)})
			want_close = true
		case !seen_not && rune_value == '!':
			buffer_push(state, Token{Kind: TOKEN_NOT, Raw: string(rune_value)})
			seen_not = true
		default:
			want_high = lex_fetch_range_low_or_text(state, rune_value)
			want_close = !want_high
		}
	}
}

// Handles a class member rune that is neither `!` nor a pending high bound. A `lo-`
// prefix starts a range (enqueue the low bound and the dash, then expect the high
// bound); otherwise the rest is a text run of members up to `]`. Returns whether a
// high bound is now expected.
func lex_fetch_range_low_or_text(state *Lexer, low rune) (want_high bool) {
	next_rune, next_width := lex_peek(state)
	if next_rune == '-' {
		lex_seek(state, next_width)
		buffer_push(state, Token{Kind: TOKEN_RANGE_LOW, Raw: string(low)})
		buffer_push(state, Token{Kind: TOKEN_RANGE_BETWEEN, Raw: string(next_rune)})
		return true
	}
	lex_unread(state)
	lex_fetch_text(state, []rune{']'})
	return false
}

// Enqueues the class-closing `]`, or latches an error when the awaited close rune is
// something else.
func lex_fetch_range_close(state *Lexer, rune_value rune) {
	if rune_value != ']' {
		lex_error(state, "expected a closing bracket to end the character class")
		return
	}
	buffer_push(state, Token{Kind: TOKEN_RANGE_CLOSE, Raw: string(rune_value)})
}

// Scans a run of literal runes into a TOKEN_TEXT, stopping before the next breaker or
// at end of input. A backslash escapes the following rune, so a breaker can be taken
// literally. A run that yields no runes enqueues nothing.
func lex_fetch_text(state *Lexer, breakers []rune) {
	var data []rune
	var escaped bool
	var done bool
	for !done {
		rune_value := lex_read(state)
		if rune_value == 0 {
			done = true
			continue
		}
		if !escaped {
			if rune_value == '\\' {
				escaped = true
				continue
			}
			if slices.Index(breakers, rune_value) != -1 {
				lex_unread(state)
				done = true
				continue
			}
		}
		escaped = false
		data = append(data, rune_value)
	}
	if len(data) > 0 {
		buffer_push(state, Token{Kind: TOKEN_TEXT, Raw: string(data)})
	}
}

// The compiler below turns a syntax abstract syntax tree into a Matcher
// tree, applying the same optimization passes as upstream: it glues adjacent
// fixed-width matchers into a row, folds a run of wildcards into a single width
// bound, factors the common head and tail out of an alternation, and splits a
// sequence around its widest static matcher into a search tree.
//
// It is a dependency-injected port of the compiler subpackage of
// github.com/gobwas/glob (MIT, Sergey Kamardin; see LICENSE.mit.kamardin).
// Upstream modelled matchers as a Matcher interface and AST payloads as an
// interface{} the compiler type-switched over. The house linter bans interfaces
// outside generic constraints, so this port reads the Matcher tagged union
// by its Kind and the Node payload by its Kind instead, and the mutating
// builder methods (EveryOf.Add) become slices assembled once. The whole
// shared/text/glob subtree is exempt from the recursion ban, so every pass
// recurses into its children exactly as upstream did.

// Turns a parsed syntax tree into the Matcher that recognizes the glob, threading
// separators through every wildcard so a '*' stops at the first separator and a
// '?' never spans one. The public Compile is the entry point; the passes
// compile_tree drives are unexported.
func compile_tree(tree *Node, separators []rune) (matcher Matcher, err error) {
	matcher, err = compile(tree, separators)
	if err != nil {
		return Matcher{}, err
	}
	return matcher, nil
}

// Dispatches on the node kind: an alternation and a pattern each drive their own
// multi-child pass, while a leaf builds one matcher the optimizer then tries to
// tighten.
func compile(tree *Node, separators []rune) (matcher Matcher, err error) {
	switch tree.Kind {
	case NODE_KIND_ANY_OF:
		return compile_any_of(tree, separators)
	case NODE_KIND_PATTERN:
		return compile_pattern(tree, separators)
	}
	matcher, err = compile_leaf(tree, separators)
	if err != nil {
		return Matcher{}, err
	}
	return optimize_matcher(matcher), nil
}

// Builds the single matcher a leaf node denotes, before optimization. An unknown
// kind is a malformed tree and is the one error compile surfaces.
func compile_leaf(
	tree *Node, separators []rune,
) (matcher Matcher, err error) {
	switch tree.Kind {
	case NODE_KIND_ANY:
		return New_Any(separators), nil
	case NODE_KIND_SUPER:
		return New_Super(), nil
	case NODE_KIND_SINGLE:
		return New_Single(separators), nil
	case NODE_KIND_EMPTY:
		return New_Empty(), nil
	case NODE_KIND_LIST:
		return New_List([]rune(tree.Characters), tree.Negated), nil
	case NODE_KIND_RANGE:
		return New_Range(&New_Range_Input{
			Low: tree.Low, High: tree.High, Negated: tree.Negated}), nil
	case NODE_KIND_TEXT:
		return New_Text(tree.Text), nil
	}
	return Matcher{}, errors.New("could not compile tree: unknown node type")
}

// Compiles an ordered sequence: an empty sequence matches only the empty string,
// otherwise the children are compiled, minimized into as few matchers as
// possible, and folded into one search tree.
func compile_pattern(
	tree *Node, separators []rune,
) (matcher Matcher, err error) {
	if len(tree.Children) == 0 {
		return New_Empty(), nil
	}
	children, err := compile_tree_children(tree, separators)
	if err != nil {
		return Matcher{}, err
	}
	matcher, err = compile_matchers(minimize_matchers(children))
	if err != nil {
		return Matcher{}, err
	}
	return optimize_matcher(matcher), nil
}

// Compiles an alternation. It first tries to factor a common head and tail out of
// the alternatives (minimize_tree); when that rewrites the tree it compiles the
// rewrite, otherwise it compiles each alternative into an any-of.
func compile_any_of(
	tree *Node, separators []rune,
) (matcher Matcher, err error) {
	minimized := minimize_tree(tree)
	if minimized != nil {
		return compile(minimized, separators)
	}
	children, err := compile_tree_children(tree, separators)
	if err != nil {
		return Matcher{}, err
	}
	return New_Any_Of(children...), nil
}

// Compiles every child of tree in order, optimizing each so the multi-child
// passes see already-tightened matchers.
func compile_tree_children(
	tree *Node, separators []rune,
) (matchers []Matcher, err error) {
	for _, child := range tree.Children {
		compiled, compile_err := compile(child, separators)
		if compile_err != nil {
			return nil, compile_err
		}
		matchers = append(matchers, optimize_matcher(compiled))
	}
	return matchers, nil
}

// Rewrites a matcher into a tighter equivalent where one exists: a separatorless
// wildcard is really a super, a one-child any-of is its child, a single-member
// list is a text, and a text-pivoted tree collapses into an affix. Kinds with no
// rewrite pass through unchanged.
func optimize_matcher(matcher Matcher) (optimized Matcher) {
	switch matcher.Kind {
	case MATCHER_KIND_ANY:
		if len(matcher.Separators) == 0 {
			return New_Super()
		}
	case MATCHER_KIND_ANY_OF:
		if len(matcher.Children) == 1 {
			return matcher.Children[0]
		}
		return matcher
	case MATCHER_KIND_LIST:
		if !matcher.Negated {
			if len(matcher.Runes) == 1 {
				return New_Text(string(matcher.Runes))
			}
		}
		return matcher
	case MATCHER_KIND_BTREE:
		return optimize_btree(matcher)
	}
	return matcher
}

// Optimizes a search tree's two sides first, then collapses the whole tree only
// when its pivot is a plain text. A non-text pivot has no tighter form, so the
// tree stands as built.
func optimize_btree(matcher Matcher) (optimized Matcher) {
	if matcher.Left != nil {
		left := optimize_matcher(*matcher.Left)
		matcher.Left = &left
	}
	if matcher.Right != nil {
		right := optimize_matcher(*matcher.Right)
		matcher.Right = &right
	}
	if matcher.Value == nil {
		return matcher
	}
	if matcher.Value.Kind != MATCHER_KIND_TEXT {
		return matcher
	}
	return optimize_btree_text(matcher)
}

// Collapses a text-pivoted search tree into the affix matcher its sides imply:
// two supers around a text is a contains, a super on one side is a prefix or
// suffix, a matching affix on the empty side is a prefix-suffix, and a wildcard
// on the empty side is a separator-bounded prefix or suffix.
func optimize_btree_text(matcher Matcher) (optimized Matcher) {
	literal := matcher.Value.Literal
	left_nil := matcher.Left == nil
	right_nil := matcher.Right == nil
	left_super := !left_nil && matcher.Left.Kind == MATCHER_KIND_SUPER
	left_prefix := !left_nil && matcher.Left.Kind == MATCHER_KIND_PREFIX
	left_any := !left_nil && matcher.Left.Kind == MATCHER_KIND_ANY
	right_super := !right_nil && matcher.Right.Kind == MATCHER_KIND_SUPER
	right_suffix := !right_nil && matcher.Right.Kind == MATCHER_KIND_SUFFIX
	right_any := !right_nil && matcher.Right.Kind == MATCHER_KIND_ANY
	switch {
	case left_nil && right_nil:
		return New_Text(literal)
	case left_super && right_super:
		return New_Contains(literal, false)
	case left_super && right_nil:
		return New_Suffix(literal)
	case right_super && left_nil:
		return New_Prefix(literal)
	case left_nil && right_suffix:
		return New_Prefix_Suffix(&New_Prefix_Suffix_Input{
			Prefix: literal, Suffix: matcher.Right.Suffix})
	case right_nil && left_prefix:
		return New_Prefix_Suffix(&New_Prefix_Suffix_Input{
			Prefix: matcher.Left.Prefix, Suffix: literal})
	case right_nil && left_any:
		return New_Suffix_Any(literal, matcher.Left.Separators)
	case left_nil && right_any:
		return New_Prefix_Any(literal, matcher.Right.Separators)
	}
	return matcher
}

// Folds a run of matchers into one: a single matcher is itself, an adjacent run
// that glues does so, otherwise the run splits around its widest static matcher
// into a search tree so the engine can pivot on the cheapest exact match. An
// empty run is a caller bug and the one error this pass raises.
func compile_matchers(matchers []Matcher) (matcher Matcher, err error) {
	if len(matchers) == 0 {
		return Matcher{}, errors.New(
			"compile error: need at least one matcher")
	}
	if len(matchers) == 1 {
		return matchers[0], nil
	}
	glued, ok := glue_matchers(matchers)
	if ok {
		return glued, nil
	}
	pivot_index := compile_matchers_pivot_index(matchers)
	if pivot_index == -1 {
		return compile_matchers_no_pivot(matchers)
	}
	return compile_matchers_split(matchers, pivot_index)
}

// Reports the index of the last widest fixed-width matcher, the one to pivot the
// search tree on, or -1 when every matcher is variable-width. The last of equal
// widths wins so ties resolve rightward, as upstream did.
func compile_matchers_pivot_index(matchers []Matcher) (pivot_index int) {
	pivot_index = -1
	widest := RUNE_WIDTH_VARIABLE
	for index, matcher := range matchers {
		width := Rune_Width(matcher)
		if width == RUNE_WIDTH_VARIABLE {
			continue
		}
		if width < widest {
			continue
		}
		widest = width
		pivot_index = index
	}
	return pivot_index
}

// Handles a run with no fixed-width matcher: the head pivots and the whole tail
// becomes its right side, so the search still makes progress one matcher at a
// time.
func compile_matchers_no_pivot(
	matchers []Matcher,
) (matcher Matcher, err error) {
	rest, err := compile_matchers(matchers[1:])
	if err != nil {
		return Matcher{}, err
	}
	return New_Btree(&New_Btree_Input{
		Value: matchers[0], Left: nil, Right: &rest}), nil
}

// Builds the search tree pivoted on the chosen matcher: the matchers before it
// compile into the left side, those after into the right, and an empty side is
// left absent.
func compile_matchers_split(
	matchers []Matcher, pivot_index int,
) (matcher Matcher, err error) {
	value := matchers[pivot_index]
	left_children := matchers[:pivot_index]
	var right_children []Matcher
	if len(matchers) > pivot_index+1 {
		right_children = matchers[pivot_index+1:]
	}
	var left *Matcher
	if len(left_children) > 0 {
		left_compiled, left_err := compile_matchers(left_children)
		if left_err != nil {
			return Matcher{}, left_err
		}
		left = &left_compiled
	}
	var right *Matcher
	if len(right_children) > 0 {
		right_compiled, right_err := compile_matchers(right_children)
		if right_err != nil {
			return Matcher{}, right_err
		}
		right = &right_compiled
	}
	return New_Btree(&New_Btree_Input{
		Value: value, Left: left, Right: right}), nil
}

// Merges an adjacent run into one matcher when it collapses either to a width
// bound (every wildcard agrees on separators) or to a fixed-width row. The
// every-of form is tried first because it subsumes more runs.
func glue_matchers(matchers []Matcher) (matcher Matcher, ok bool) {
	every, every_ok := glue_as_every(matchers)
	if every_ok {
		return every, true
	}
	row, row_ok := glue_as_row(matchers)
	if row_ok {
		return row, true
	}
	return Matcher{}, false
}

// Glues a run of two or more fixed-width matchers into one row whose width is
// their total. A single variable-width member defeats it, since a row's width
// must be known up front.
func glue_as_row(matchers []Matcher) (matcher Matcher, ok bool) {
	if len(matchers) <= 1 {
		return Matcher{}, false
	}
	total_width := 0
	children := make([]Matcher, 0, len(matchers))
	for _, member := range matchers {
		width := Rune_Width(member)
		if width == RUNE_WIDTH_VARIABLE {
			return Matcher{}, false
		}
		children = append(children, member)
		total_width += width
	}
	return New_Row(total_width, children...), true
}

// Glues a run of two or more wildcards that all agree on their separator set into
// a single length-and-separator constraint. A run that mixes separator sets, or
// holds a non-wildcard, does not glue.
func glue_as_every(matchers []Matcher) (matcher Matcher, ok bool) {
	if len(matchers) <= 1 {
		return Matcher{}, false
	}
	facts, classified := glue_every_classify(matchers)
	if !classified {
		return Matcher{}, false
	}
	return glue_every_build(&facts), true
}

// Records which wildcard kinds a run holds and the separator set they must share;
// it fails the moment a member is not a wildcard or a member disagrees on the
// separator set.
func glue_every_classify(
	matchers []Matcher,
) (facts Glue_Every_Facts, ok bool) {
	for index, matcher := range matchers {
		separator, member_ok := glue_every_member_separator(matcher, &facts)
		if !member_ok {
			return facts, false
		}
		if index == 0 {
			facts.Separator = separator
		}
		if !slices.Equal(separator, facts.Separator) {
			return facts, false
		}
	}
	return facts, true
}

// Records one member's kind into facts and reports the separator set it
// constrains the run to. A super constrains nothing (the empty set); a
// non-negated list is not a wildcard and fails the glue.
func glue_every_member_separator(
	matcher Matcher, facts *Glue_Every_Facts,
) (separator []rune, ok bool) {
	switch matcher.Kind {
	case MATCHER_KIND_SUPER:
		facts.Has_Super = true
		return []rune{}, true
	case MATCHER_KIND_ANY:
		facts.Has_Any = true
		return matcher.Separators, true
	case MATCHER_KIND_SINGLE:
		facts.Has_Single = true
		facts.Minimum++
		return matcher.Separators, true
	case MATCHER_KIND_LIST:
		if !matcher.Negated {
			return nil, false
		}
		facts.Has_Single = true
		facts.Minimum++
		return matcher.Runes, true
	}
	return nil, false
}

// Glue_Every_Facts records what glue_every_classify learned about a run of
// wildcards, so glue_every_build can pick the tightest equivalent matcher without
// re-walking the run.
type Glue_Every_Facts struct {
	// Has_Any is set when the run holds a separator-bounded wildcard.
	Has_Any bool
	// Has_Super is set when the run holds an unbounded wildcard.
	Has_Super bool
	// Has_Single is set when the run holds a fixed one-rune matcher.
	Has_Single bool
	// Minimum is the count of one-rune matchers, the run's least width.
	Minimum int
	// Separator is the separator set every wildcard in the run agrees on.
	Separator []rune
}

// Picks the tightest matcher a classified run collapses to: a pure super, a pure
// any, or a bare minimum-width bound where the shape allows, falling back to the
// general composite otherwise.
func glue_every_build(facts *Glue_Every_Facts) (matcher Matcher) {
	if facts.Has_Super {
		if !facts.Has_Any {
			if !facts.Has_Single {
				return New_Super()
			}
		}
	}
	if facts.Has_Any {
		if !facts.Has_Super {
			if !facts.Has_Single {
				return New_Any(facts.Separator)
			}
		}
	}
	wildcard := facts.Has_Any || facts.Has_Super
	if wildcard {
		if facts.Minimum > 0 {
			if len(facts.Separator) == 0 {
				return New_Min(facts.Minimum)
			}
		}
	}
	return glue_every_composite(facts)
}

// Builds the general every-of a mixed run reduces to: a minimum bound for its
// one-rune matchers, a maximum bound too when no wildcard can stretch it, and a
// separator exclusion when the run is separator-bounded.
func glue_every_composite(facts *Glue_Every_Facts) (matcher Matcher) {
	children := make([]Matcher, 0, 3)
	if facts.Minimum > 0 {
		children = append(children, New_Min(facts.Minimum))
		if !facts.Has_Any {
			if !facts.Has_Super {
				children = append(children, New_Max(facts.Minimum))
			}
		}
	}
	if len(facts.Separator) > 0 {
		children = append(
			children, New_Contains(string(facts.Separator), true))
	}
	return New_Every_Of(children...)
}

// Repeatedly collapses the single most valuable gluable span until no span glues,
// so a sequence reaches its fewest matchers. Among gluable spans it prefers a
// wider fixed result, and among those the longer span, since a longer glued span
// removes more matchers.
func minimize_matchers(matchers []Matcher) (minimized []Matcher) {
	matcher_count := len(matchers)
	var best Matcher
	found := false
	best_left_index := 0
	best_right_index := 0
	best_count := 0
	for left_index := 0; left_index < matcher_count; left_index++ {
		for right_index := matcher_count; right_index > left_index; right_index-- {
			glued, ok := glue_matchers(matchers[left_index:right_index])
			if !ok {
				continue
			}
			span_count := right_index - left_index
			swap := !found
			if found {
				best_width := Rune_Width(best)
				glued_width := Rune_Width(glued)
				swap = best_width > RUNE_WIDTH_VARIABLE
				swap = swap && glued_width > RUNE_WIDTH_VARIABLE
				swap = swap && glued_width > best_width
				swap = swap || best_count < span_count
			}
			if !swap {
				continue
			}
			best = glued
			found = true
			best_left_index = left_index
			best_right_index = right_index
			best_count = span_count
		}
	}
	if !found {
		return matchers
	}
	next := make([]Matcher, 0, matcher_count)
	next = append(next, matchers[:best_left_index]...)
	next = append(next, best)
	if best_right_index < matcher_count {
		next = append(next, matchers[best_right_index:]...)
	}
	if len(next) == matcher_count {
		return next
	}
	return minimize_matchers(next)
}

// Rewrites a node into a smaller-but-equivalent tree where a heuristic applies;
// only an alternation has one. A nil result means no rewrite, so the caller
// compiles the node as it stands.
func minimize_tree(tree *Node) (minimized *Node) {
	switch tree.Kind {
	case NODE_KIND_ANY_OF:
		return minimize_tree_any_of(tree)
	}
	return nil
}

// Factors the common leading and trailing children out of an alternation of
// patterns, so "abcX, abcY" becomes "abc(X|Y)". A nil result means the
// alternatives share no common head or tail, or are not all patterns.
func minimize_tree_any_of(tree *Node) (minimized *Node) {
	if !are_of_same_kind(tree.Children, NODE_KIND_PATTERN) {
		return nil
	}
	common_left, common_right := common_children(tree.Children)
	common_left_count := len(common_left)
	common_right_count := len(common_right)
	if common_left_count == 0 {
		if common_right_count == 0 {
			return nil
		}
	}
	var result []*Node
	if common_left_count > 0 {
		result = append(result, &Node{
			Kind: NODE_KIND_PATTERN, Children: common_left})
	}
	var alternatives []*Node
	for _, child := range tree.Children {
		reused := child.Children[common_left_count : len(child.Children)-common_right_count]
		node := &Node{Kind: NODE_KIND_EMPTY}
		if len(reused) > 0 {
			node = &Node{Kind: NODE_KIND_PATTERN, Children: reused}
		}
		alternatives = append_if_unique(alternatives, node)
	}
	result = append(result, minimize_any_of_middle(alternatives)...)
	if common_right_count > 0 {
		result = append(result, &Node{
			Kind: NODE_KIND_PATTERN, Children: common_right})
	}
	return &Node{Kind: NODE_KIND_PATTERN, Children: result}
}

// Wraps the reduced alternatives that sit between the common head and tail: none
// when they all reduced away, the lone survivor bare (unless it reduced to
// nothing), or a fresh alternation of the several.
func minimize_any_of_middle(alternatives []*Node) (middle []*Node) {
	switch {
	case len(alternatives) == 1 && alternatives[0].Kind != NODE_KIND_EMPTY:
		return []*Node{alternatives[0]}
	case len(alternatives) > 1:
		return []*Node{{Kind: NODE_KIND_ANY_OF, Children: alternatives}}
	}
	return nil
}

// Reports whether every node in the slice has the given kind, the precondition
// the alternation-factoring heuristic needs before it may assume each child is a
// pattern.
func are_of_same_kind(nodes []*Node, kind Node_Kind) (same bool) {
	for _, node := range nodes {
		if node.Kind != kind {
			return false
		}
	}
	return true
}

// Appends node unless the slice already holds a structurally equal node, so the
// reduced alternatives carry no duplicates.
func append_if_unique(
	target []*Node, node *Node,
) (result []*Node) {
	for _, member := range target {
		if nodes_equal(&Nodes_Equal_Input{First: member, Second: node}) {
			return target
		}
	}
	return append(target, node)
}

// Finds the children shared by the head and by the tail of a set of pattern
// nodes: the longest run of leading children equal across all of them, and
// likewise the longest trailing run. It anchors the scan on the node with fewest
// children, since no common run can be longer than that.
func common_children(
	nodes []*Node,
) (common_left []*Node, common_right []*Node) {
	if len(nodes) <= 1 {
		return nil, nil
	}
	least_index := least_children(nodes)
	if least_index == -1 {
		return nil, nil
	}
	smallest_node := nodes[least_index]
	smallest_child_count := len(smallest_node.Children)
	common_right = make([]*Node, smallest_child_count)
	last_right_index := smallest_child_count
	stop_left := false
	stop_right := false
	common_count := 0
	left_index := 0
	right_index := smallest_child_count - 1
	for right_index >= 0 {
		if common_count >= smallest_child_count {
			break
		}
		if stop_left {
			if stop_right {
				break
			}
		}
		stop_left, stop_right = common_children_column(&Common_Children_Column_Input{
			Nodes: nodes, Smallest: smallest_node, Least_Index: least_index,
			Left_Index: left_index, Right_Index: right_index,
			Smallest_Child_Count: smallest_child_count,
			Stop_Left:            stop_left, Stop_Right: stop_right,
		})
		if !stop_left {
			common_count++
			common_left = append(common_left, smallest_node.Children[left_index])
		}
		if !stop_right {
			common_count++
			last_right_index = right_index
			common_right[right_index] = smallest_node.Children[right_index]
		}
		left_index++
		right_index--
	}
	common_right = common_right[last_right_index:]
	return common_left, common_right
}

// Common_Children_Column_Input carries one head/tail column pair of the
// common-children scan: the reference node, the full node set, the column indices
// being compared, and the running stop flags the scan threads through.
type Common_Children_Column_Input struct {
	// Nodes is the full set of pattern nodes being compared.
	Nodes []*Node
	// Smallest is the node with fewest children, the scan's reference.
	Smallest *Node
	// Least_Index is Smallest's position in Nodes, skipped in the comparison.
	Least_Index int
	// Left_Index is the leading column under comparison.
	Left_Index int
	// Right_Index is the trailing column of Smallest under comparison.
	Right_Index int
	// Smallest_Child_Count is Smallest's child count, aligning the trailing
	// columns of nodes of different lengths.
	Smallest_Child_Count int
	// Stop_Left is set once the leading run has diverged, and stays set.
	Stop_Left bool
	// Stop_Right is set once the trailing run has diverged, and stays set.
	Stop_Right bool
}

// Tests one head column and one tail column across every node against the
// reference, returning the stop flags updated: a leading mismatch stops the head,
// a trailing mismatch or an overlap with the head stops the tail. Both flags
// latch, so once stopped a run never resumes.
func common_children_column(
	input *Common_Children_Column_Input,
) (stop_left bool, stop_right bool) {
	stop_left = input.Stop_Left
	stop_right = input.Stop_Right
	smallest_left := input.Smallest.Children[input.Left_Index]
	smallest_right := input.Smallest.Children[input.Right_Index]
	for node_index := 0; node_index < len(input.Nodes); node_index++ {
		if stop_left {
			if stop_right {
				break
			}
		}
		if node_index == input.Least_Index {
			continue
		}
		child := input.Nodes[node_index]
		other_left := child.Children[input.Left_Index]
		other_right_index := input.Right_Index + len(child.Children) -
			input.Smallest_Child_Count
		other_right := child.Children[other_right_index]
		left_equal := nodes_equal(
			&Nodes_Equal_Input{First: smallest_left, Second: other_left})
		stop_left = stop_left || !left_equal
		stop_right = stop_right ||
			(!stop_left && input.Right_Index <= input.Left_Index)
		right_equal := nodes_equal(
			&Nodes_Equal_Input{First: smallest_right, Second: other_right})
		stop_right = stop_right || !right_equal
	}
	return stop_left, stop_right
}

// Reports the index of the node with the fewest children, or -1 for an empty set.
// The common-children scan anchors on it because a shared run cannot exceed the
// shortest node's length.
func least_children(nodes []*Node) (least_index int) {
	least_index = -1
	fewest_count := -1
	for node_index, node := range nodes {
		take := least_index == -1
		if !take {
			take = len(node.Children) < fewest_count
		}
		if !take {
			continue
		}
		fewest_count = len(node.Children)
		least_index = node_index
	}
	return least_index
}

// Nodes_Equal_Input pairs the two syntax subtrees nodes_equal compares. The input
// struct exists because two *Node parameters are otherwise swappable at
// the call site.
type Nodes_Equal_Input struct {
	// First is one subtree to compare.
	First *Node
	// Second is the other subtree to compare.
	Second *Node
}

// Reports whether two syntax subtrees are structurally identical, comparing kind
// and every payload field and recursing into children. It ignores the Parent
// back-pointer, which reflect.DeepEqual would wrongly follow and which upstream's
// Node.Equal likewise disregarded.
func nodes_equal(input *Nodes_Equal_Input) (equal bool) {
	first := input.First
	second := input.Second
	if first == nil {
		return second == nil
	}
	if second == nil {
		return false
	}
	if first.Kind != second.Kind {
		return false
	}
	if first.Text != second.Text {
		return false
	}
	if first.Characters != second.Characters {
		return false
	}
	if first.Negated != second.Negated {
		return false
	}
	if first.Low != second.Low {
		return false
	}
	if first.High != second.High {
		return false
	}
	if len(first.Children) != len(second.Children) {
		return false
	}
	for index, child := range first.Children {
		pair := &Nodes_Equal_Input{First: child, Second: second.Children[index]}
		if !nodes_equal(pair) {
			return false
		}
	}
	return true
}

// The matchers below evaluate the leaf and composite matchers a compiled glob is
// built from: each matcher reports whether a whole string matches (matcher_matches) and
// where within a string it could match (Index), the two operations the glob
// engine composes.
//
// It is a dependency-injected port of the match subpackage of
// github.com/gobwas/glob (MIT, Sergey Kamardin; see LICENSE.mit.kamardin).
// Upstream modelled every matcher as an implementation of a Matcher INTERFACE.
// The house linter bans interface declarations, so the twenty implementations
// collapse into ONE tagged-union struct discriminated by Matcher_Kind, and the
// interface's methods become the free functions matcher_matches, Index, and Rune_Width.
// String survives as a method because it satisfies fmt.Stringer, which is
// allowed. The whole shared/text/glob subtree is exempt from the recursion ban,
// so composite kinds recurse into their children exactly as upstream did.

// Matcher_Kind discriminates the Matcher tagged union: it names which matcher a Matcher
// value is, and therefore which of the union's fields carry meaning.
type Matcher_Kind int

// MATCHER_KIND_ANY matches a maximal run of runes containing none of Separators — the
// glob '*' bounded at the next separator (upstream match.Any).
const MATCHER_KIND_ANY Matcher_Kind = 0

// MATCHER_KIND_SUPER matches any string, separators included — the glob '**' (upstream
// match.Super).
const MATCHER_KIND_SUPER Matcher_Kind = 1

// MATCHER_KIND_SINGLE matches exactly one rune that is not a separator — the glob '?'
// (upstream match.Single).
const MATCHER_KIND_SINGLE Matcher_Kind = 2

// MATCHER_KIND_EMPTY matches only the empty string (upstream match.Nothing, renamed
// because the linter rejects the present participle "nothing").
const MATCHER_KIND_EMPTY Matcher_Kind = 3

// MATCHER_KIND_TEXT matches one exact literal string (upstream match.Text).
const MATCHER_KIND_TEXT Matcher_Kind = 4

// MATCHER_KIND_MAX matches any run of at most Limit runes (upstream match.Max).
const MATCHER_KIND_MAX Matcher_Kind = 5

// MATCHER_KIND_MIN matches any run of at least Limit runes (upstream match.Min).
const MATCHER_KIND_MIN Matcher_Kind = 6

// MATCHER_KIND_PREFIX matches any string that begins with Prefix (upstream
// match.Prefix).
const MATCHER_KIND_PREFIX Matcher_Kind = 7

// MATCHER_KIND_SUFFIX matches any string that ends with Suffix (upstream match.Suffix).
const MATCHER_KIND_SUFFIX Matcher_Kind = 8

// MATCHER_KIND_PREFIX_SUFFIX matches any string that begins with Prefix and ends with
// Suffix (upstream match.PrefixSuffix).
const MATCHER_KIND_PREFIX_SUFFIX Matcher_Kind = 9

// MATCHER_KIND_CONTAINS matches when Needle occurs in the string, or, when Negated,
// when it does not (upstream match.Contains).
const MATCHER_KIND_CONTAINS Matcher_Kind = 10

// MATCHER_KIND_RANGE matches exactly one rune inside [Low, High], inverted by Negated
// (upstream match.Range).
const MATCHER_KIND_RANGE Matcher_Kind = 11

// MATCHER_KIND_LIST matches exactly one rune drawn from Runes, inverted by Negated
// (upstream match.List).
const MATCHER_KIND_LIST Matcher_Kind = 12

// MATCHER_KIND_ROW matches a fixed-width sequence of adjacent child matchers whose rune
// widths sum to Rune_Width (upstream match.Row).
const MATCHER_KIND_ROW Matcher_Kind = 13

// MATCHER_KIND_ANY_OF matches when at least one child matches (upstream match.AnyOf).
const MATCHER_KIND_ANY_OF Matcher_Kind = 14

// MATCHER_KIND_EVERY_OF matches when every child matches (upstream match.EveryOf).
const MATCHER_KIND_EVERY_OF Matcher_Kind = 15

// MATCHER_KIND_BTREE matches Value somewhere in the string with Left matching the text
// before it and Right the text after (upstream match.BTree).
const MATCHER_KIND_BTREE Matcher_Kind = 16

// MATCHER_KIND_PREFIX_ANY matches Prefix followed by a run holding no separator
// (upstream match.PrefixAny).
const MATCHER_KIND_PREFIX_ANY Matcher_Kind = 17

// MATCHER_KIND_SUFFIX_ANY matches a run holding no separator followed by Suffix
// (upstream match.SuffixAny).
const MATCHER_KIND_SUFFIX_ANY Matcher_Kind = 18

// RUNE_WIDTH_VARIABLE is the Rune_Width of a matcher whose match length is not
// fixed; it is the sentinel the composers test before assuming a width
// (upstream lenNo).
const RUNE_WIDTH_VARIABLE int = -1

// RUNE_WIDTH_ZERO is the Rune_Width of a matcher that consumes no runes
// (upstream lenZero).
const RUNE_WIDTH_ZERO int = 0

// RUNE_WIDTH_ONE is the Rune_Width of a single-rune matcher (upstream lenOne).
const RUNE_WIDTH_ONE int = 1

// Matcher is the tagged union of every glob matcher. Kind selects the variant;
// each remaining field is read only by the kinds whose doc names it, and left
// zero otherwise. The fields are exported so the compiler package that builds
// these values can also inspect them.
type Matcher struct {
	// Kind selects the union variant and thus which other fields are live.
	Kind Matcher_Kind
	// Separators is the set that bounds a wildcard run: matching stops at the
	// first of these runes. Read by Any, Single, Prefix_Any, Suffix_Any.
	Separators []rune
	// Runes is the membership set a single-rune List matches against.
	Runes []rune
	// Literal is the exact string a Text matcher compares against.
	Literal string
	// Prefix is the required leading substring for Prefix, Prefix_Suffix, and
	// Prefix_Any.
	Prefix string
	// Suffix is the required trailing substring for Suffix, Prefix_Suffix, and
	// Suffix_Any.
	Suffix string
	// Needle is the substring a Contains matcher searches for.
	Needle string
	// Low is the inclusive lower bound of a Range matcher's rune interval.
	Low rune
	// High is the inclusive upper bound of a Range matcher's rune interval.
	High rune
	// Negated inverts the membership test of Contains, Range, and List.
	Negated bool
	// Limit is the rune-count bound of Max (at most) and Min (at least).
	Limit int
	// Rune_Width is the cached rune width every constructor sets: a fixed count for
	// the fixed-width kinds and RUNE_WIDTH_VARIABLE otherwise. Caching it lets the
	// match-time hot paths (Row and Btree) read a child's width instead of
	// recomputing it — and recomputing meant a 200-byte struct copy per call.
	Rune_Width int
	// Segments is the fixed match-length list Index returns for the kinds whose
	// segments do not depend on the input — Text and Row. Precomputing it lets
	// Index hand back this slice instead of allocating one per call, which matters
	// because a Btree calls Index on its Text pivot once per candidate offset. It is
	// read-only; callers never mutate a returned segment list.
	Segments []int
	// Children are the sub-matchers of the composite kinds Any_Of, Every_Of,
	// and Row.
	Children []Matcher
	// Value is the pivot a Btree searches for before testing its two sides.
	Value *Matcher
	// Left matches the text before a Btree's Value; nil requires that side
	// empty.
	Left *Matcher
	// Right matches the text after a Btree's Value; nil requires that side
	// empty.
	Right *Matcher
}

// New_Any builds a matcher for a wildcard run bounded by separators.
func New_Any(separators []rune) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_ANY,
		Separators: separators,
		Rune_Width: RUNE_WIDTH_VARIABLE,
	}
}

// New_Super builds a matcher that accepts every string.
func New_Super() (matcher Matcher) {
	return Matcher{Kind: MATCHER_KIND_SUPER, Rune_Width: RUNE_WIDTH_VARIABLE}
}

// New_Single builds a matcher for one non-separator rune.
func New_Single(separators []rune) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_SINGLE,
		Separators: separators,
		Rune_Width: RUNE_WIDTH_ONE,
	}
}

// New_Empty builds a matcher that accepts only the empty string (upstream
// NewNothing, renamed because the linter rejects the participle "nothing").
func New_Empty() (matcher Matcher) {
	return Matcher{Kind: MATCHER_KIND_EMPTY, Rune_Width: RUNE_WIDTH_ZERO}
}

// New_Text builds a matcher for one exact literal; its rune width is cached so
// composers need not recount it.
func New_Text(literal string) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_TEXT,
		Literal:    literal,
		Rune_Width: utf8.RuneCountInString(literal),
		Segments:   []int{len(literal)},
	}
}

// New_Max builds a matcher for a run of at most limit runes.
func New_Max(limit int) (matcher Matcher) {
	return Matcher{Kind: MATCHER_KIND_MAX, Limit: limit, Rune_Width: RUNE_WIDTH_VARIABLE}
}

// New_Min builds a matcher for a run of at least limit runes.
func New_Min(limit int) (matcher Matcher) {
	return Matcher{Kind: MATCHER_KIND_MIN, Limit: limit, Rune_Width: RUNE_WIDTH_VARIABLE}
}

// New_Prefix builds a matcher for strings beginning with prefix.
func New_Prefix(prefix string) (matcher Matcher) {
	return Matcher{Kind: MATCHER_KIND_PREFIX, Prefix: prefix, Rune_Width: RUNE_WIDTH_VARIABLE}
}

// New_Suffix builds a matcher for strings ending with suffix.
func New_Suffix(suffix string) (matcher Matcher) {
	return Matcher{Kind: MATCHER_KIND_SUFFIX, Suffix: suffix, Rune_Width: RUNE_WIDTH_VARIABLE}
}

// New_Prefix_Suffix_Input carries the two bookend substrings; the input struct
// exists because two string parameters are swappable at the call site.
type New_Prefix_Suffix_Input struct {
	// Prefix is the required leading substring.
	Prefix string
	// Suffix is the required trailing substring.
	Suffix string
}

// New_Prefix_Suffix builds a matcher for strings that both begin with Prefix
// and end with Suffix.
func New_Prefix_Suffix(input *New_Prefix_Suffix_Input) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_PREFIX_SUFFIX,
		Prefix:     input.Prefix,
		Suffix:     input.Suffix,
		Rune_Width: RUNE_WIDTH_VARIABLE,
	}
}

// New_Contains builds a matcher that tests whether needle occurs; negated flips
// the test to require its absence.
func New_Contains(needle string, negated bool) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_CONTAINS,
		Needle:     needle,
		Negated:    negated,
		Rune_Width: RUNE_WIDTH_VARIABLE,
	}
}

// New_Range_Input carries the rune interval; the input struct exists because
// two rune parameters are swappable at the call site.
type New_Range_Input struct {
	// Low is the inclusive lower bound of the interval.
	Low rune
	// High is the inclusive upper bound of the interval.
	High rune
	// Negated inverts membership so the matcher accepts runes outside the
	// interval.
	Negated bool
}

// New_Range builds a matcher for one rune inside [Low, High], inverted by
// Negated.
func New_Range(input *New_Range_Input) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_RANGE,
		Low:        input.Low,
		High:       input.High,
		Negated:    input.Negated,
		Rune_Width: RUNE_WIDTH_ONE,
	}
}

// New_List builds a matcher for one rune drawn from set; negated flips it to
// accept runes outside set.
func New_List(set []rune, negated bool) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_LIST,
		Runes:      set,
		Negated:    negated,
		Rune_Width: RUNE_WIDTH_ONE,
	}
}

// New_Row builds a fixed-width matcher for the given adjacent children whose
// rune widths sum to width.
func New_Row(width int, children ...Matcher) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_ROW,
		Rune_Width: width,
		Children:   children,
		Segments:   []int{width},
	}
}

// New_Any_Of builds a matcher that accepts a string matched by any child.
func New_Any_Of(children ...Matcher) (matcher Matcher) {
	matcher = Matcher{Kind: MATCHER_KIND_ANY_OF, Children: children}
	matcher.Rune_Width = rune_width_any_of(matcher)
	return matcher
}

// New_Every_Of builds a matcher that accepts a string matched by every child.
func New_Every_Of(children ...Matcher) (matcher Matcher) {
	matcher = Matcher{Kind: MATCHER_KIND_EVERY_OF, Children: children}
	matcher.Rune_Width = rune_width_every_of(matcher)
	return matcher
}

// New_Btree_Input carries the pivot and its two side matchers; the input struct
// exists because three Matcher parameters are swappable at the call site.
type New_Btree_Input struct {
	// Value is the pivot matcher the tree searches for.
	Value Matcher
	// Left matches the text before Value; nil requires that side empty.
	Left *Matcher
	// Right matches the text after Value; nil requires that side empty.
	Right *Matcher
}

// New_Btree builds a pivot matcher, caching the total rune width so the search
// window can be pre-trimmed.
func New_Btree(input *New_Btree_Input) (matcher Matcher) {
	value := input.Value
	matcher = Matcher{
		Kind:  MATCHER_KIND_BTREE,
		Value: &value,
		Left:  input.Left,
		Right: input.Right,
	}
	matcher.Rune_Width = btree_total_rune_width(matcher)
	return matcher
}

// New_Prefix_Any builds a matcher for prefix followed by a run that stops at the
// first separator.
func New_Prefix_Any(prefix string, separators []rune) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_PREFIX_ANY,
		Prefix:     prefix,
		Separators: separators,
		Rune_Width: RUNE_WIDTH_VARIABLE,
	}
}

// New_Suffix_Any builds a matcher for a run that stops at the last separator,
// followed by suffix.
func New_Suffix_Any(suffix string, separators []rune) (matcher Matcher) {
	return Matcher{
		Kind:       MATCHER_KIND_SUFFIX_ANY,
		Suffix:     suffix,
		Separators: separators,
		Rune_Width: RUNE_WIDTH_VARIABLE,
	}
}

// Reports whether text is matched in full by matcher, the whole-string test.
// The terminal kinds whose match logic was inline in the switch are their own
// functions so a whole-pattern matcher of that kind can be dispatched straight to
// one via Pattern.Evaluate, skipping the switch. matcher_matches delegates to the
// same functions, so each kind's logic lives in exactly one place.

func match_any(matcher *Matcher, text string) (matched bool) {
	return Index_Any_Runes(text, matcher.Separators) == -1
}

func match_super(matcher *Matcher, text string) (matched bool) {
	return true
}

func match_empty(matcher *Matcher, text string) (matched bool) {
	return text == ""
}

func match_text(matcher *Matcher, text string) (matched bool) {
	return matcher.Literal == text
}

func match_prefix(matcher *Matcher, text string) (matched bool) {
	return strings.HasPrefix(text, matcher.Prefix)
}

func match_suffix(matcher *Matcher, text string) (matched bool) {
	return strings.HasSuffix(text, matcher.Suffix)
}

func match_prefix_suffix(matcher *Matcher, text string) (matched bool) {
	return strings.HasPrefix(text, matcher.Prefix) &&
		strings.HasSuffix(text, matcher.Suffix)
}

func match_contains(matcher *Matcher, text string) (matched bool) {
	return strings.Contains(text, matcher.Needle) != matcher.Negated
}

func match_row(matcher *Matcher, text string) (matched bool) {
	return row_rune_width_ok(matcher, text) && row_match_all(matcher, text)
}

// Maps a matcher's kind to the function that evaluates it, resolved once by
// Compile so Match can call it directly without re-inspecting Kind.
func select_evaluator(
	kind Matcher_Kind,
) (evaluate func(matcher *Matcher, text string) (matched bool)) {
	switch kind {
	case MATCHER_KIND_ANY:
		return match_any
	case MATCHER_KIND_SUPER:
		return match_super
	case MATCHER_KIND_SINGLE:
		return match_single
	case MATCHER_KIND_EMPTY:
		return match_empty
	case MATCHER_KIND_TEXT:
		return match_text
	case MATCHER_KIND_MAX:
		return match_max
	case MATCHER_KIND_MIN:
		return match_min
	case MATCHER_KIND_PREFIX:
		return match_prefix
	case MATCHER_KIND_SUFFIX:
		return match_suffix
	case MATCHER_KIND_PREFIX_SUFFIX:
		return match_prefix_suffix
	case MATCHER_KIND_CONTAINS:
		return match_contains
	case MATCHER_KIND_RANGE:
		return match_range
	case MATCHER_KIND_LIST:
		return match_list
	case MATCHER_KIND_ROW:
		return match_row
	case MATCHER_KIND_ANY_OF:
		return match_any_of
	case MATCHER_KIND_EVERY_OF:
		return match_every_of
	case MATCHER_KIND_BTREE:
		return match_btree
	case MATCHER_KIND_PREFIX_ANY:
		return match_prefix_any
	case MATCHER_KIND_SUFFIX_ANY:
		return match_suffix_any
	}
	return matcher_matches
}

func matcher_matches(matcher *Matcher, text string) (matched bool) {
	switch matcher.Kind {
	case MATCHER_KIND_ANY:
		return match_any(matcher, text)
	case MATCHER_KIND_SUPER:
		return match_super(matcher, text)
	case MATCHER_KIND_SINGLE:
		return match_single(matcher, text)
	case MATCHER_KIND_EMPTY:
		return match_empty(matcher, text)
	case MATCHER_KIND_TEXT:
		return match_text(matcher, text)
	case MATCHER_KIND_MAX:
		return match_max(matcher, text)
	case MATCHER_KIND_MIN:
		return match_min(matcher, text)
	case MATCHER_KIND_PREFIX:
		return match_prefix(matcher, text)
	case MATCHER_KIND_SUFFIX:
		return match_suffix(matcher, text)
	case MATCHER_KIND_PREFIX_SUFFIX:
		return match_prefix_suffix(matcher, text)
	case MATCHER_KIND_CONTAINS:
		return match_contains(matcher, text)
	case MATCHER_KIND_RANGE:
		return match_range(matcher, text)
	case MATCHER_KIND_LIST:
		return match_list(matcher, text)
	case MATCHER_KIND_ROW:
		return match_row(matcher, text)
	case MATCHER_KIND_ANY_OF:
		return match_any_of(matcher, text)
	case MATCHER_KIND_EVERY_OF:
		return match_every_of(matcher, text)
	case MATCHER_KIND_BTREE:
		return match_btree(matcher, text)
	case MATCHER_KIND_PREFIX_ANY:
		return match_prefix_any(matcher, text)
	case MATCHER_KIND_SUFFIX_ANY:
		return match_suffix_any(matcher, text)
	}
	return false
}

func match_single(matcher *Matcher, text string) (matched bool) {
	first, width := utf8.DecodeRuneInString(text)
	if len(text) > width {
		return false
	}
	return slices.Index(matcher.Separators, first) == -1
}

func match_range(matcher *Matcher, text string) (matched bool) {
	first, width := utf8.DecodeRuneInString(text)
	if len(text) > width {
		return false
	}
	in_range := first >= matcher.Low && first <= matcher.High
	return in_range == !matcher.Negated
}

func match_list(matcher *Matcher, text string) (matched bool) {
	first, width := utf8.DecodeRuneInString(text)
	if len(text) > width {
		return false
	}
	in_list := slices.Index(matcher.Runes, first) != -1
	return in_list == !matcher.Negated
}

func match_max(matcher *Matcher, text string) (matched bool) {
	count := 0
	for range text {
		count++
		if count > matcher.Limit {
			return false
		}
	}
	return true
}

func match_min(matcher *Matcher, text string) (matched bool) {
	count := 0
	for range text {
		count++
		if count >= matcher.Limit {
			return true
		}
	}
	return false
}

func match_any_of(matcher *Matcher, text string) (matched bool) {
	for child_index := range matcher.Children {
		if matcher_matches(&matcher.Children[child_index], text) {
			return true
		}
	}
	return false
}

func match_every_of(matcher *Matcher, text string) (matched bool) {
	for child_index := range matcher.Children {
		if !matcher_matches(&matcher.Children[child_index], text) {
			return false
		}
	}
	return true
}

func match_prefix_any(matcher *Matcher, text string) (matched bool) {
	if !strings.HasPrefix(text, matcher.Prefix) {
		return false
	}
	remainder := text[len(matcher.Prefix):]
	return Index_Any_Runes(remainder, matcher.Separators) == -1
}

func match_suffix_any(matcher *Matcher, text string) (matched bool) {
	if !strings.HasSuffix(text, matcher.Suffix) {
		return false
	}
	remainder := text[:len(text)-len(matcher.Suffix)]
	return Index_Any_Runes(remainder, matcher.Separators) == -1
}

// Slides a window across text, at each position matching Value and testing the
// left remainder against the left matcher, then the right remainders against
// the right matcher.
func match_btree(matcher *Matcher, text string) (matched bool) {
	text_size := len(text)
	start, limit := btree_offset_limit(matcher, text_size)
	for start < limit {
		value_start, segments := Index(matcher.Value, text[start:limit])
		if value_start == -1 {
			return false
		}
		left_text := text[:start+value_start]
		left_matched := left_text == ""
		if matcher.Left != nil {
			left_matched = matcher_matches(matcher.Left, left_text)
		}
		if left_matched {
			if match_btree_right(matcher, text, start+value_start, segments) {
				return true
			}
		}
		_, step := utf8.DecodeRuneInString(text[start+value_start:])
		start += value_start + step
	}
	return false
}

// Tries the right matcher against each segment Value reported, longest first, so
// the greediest split is preferred; base is the byte offset where Value's match
// begins.
func match_btree_right(matcher *Matcher, text string, base int, segments []int) (matched bool) {
	text_size := len(text)
	for segment_index := len(segments) - 1; segment_index >= 0; segment_index-- {
		segment := segments[segment_index]
		right_text := ""
		if text_size > base+segment {
			right_text = text[base+segment:]
		}
		right_matched := right_text == ""
		if matcher.Right != nil {
			right_matched = matcher_matches(matcher.Right, right_text)
		}
		if right_matched {
			return true
		}
	}
	return false
}

// Trims the window Value is searched in, using the cached total width and the
// two side widths so an input too short to hold all three parts is rejected
// before any search.
func btree_offset_limit(matcher *Matcher, text_size int) (offset int, limit int) {
	total_width := matcher.Rune_Width
	if total_width != RUNE_WIDTH_VARIABLE {
		if total_width > text_size {
			return 0, 0
		}
	}
	left_width := btree_child_rune_width(matcher.Left)
	right_width := btree_child_rune_width(matcher.Right)
	if left_width >= 0 {
		offset = left_width
	}
	limit = text_size
	if right_width >= 0 {
		limit = text_size - right_width
	}
	return offset, limit
}

// Index reports the first offset in text at which matcher could match, together
// with the segment lengths of the possible matches there, or (-1, nil) when
// matcher cannot match anywhere.
func Index(matcher *Matcher, text string) (offset int, segments []int) {
	switch matcher.Kind {
	case MATCHER_KIND_ANY:
		return index_any(matcher, text)
	case MATCHER_KIND_SUPER:
		return index_super(matcher, text)
	case MATCHER_KIND_SINGLE:
		return index_single(matcher, text)
	case MATCHER_KIND_EMPTY:
		return 0, []int{0}
	case MATCHER_KIND_TEXT:
		return index_text(matcher, text)
	case MATCHER_KIND_MAX:
		return index_max(matcher, text)
	case MATCHER_KIND_MIN:
		return index_min(matcher, text)
	case MATCHER_KIND_PREFIX:
		return index_prefix(matcher, text)
	case MATCHER_KIND_SUFFIX:
		return index_suffix(matcher, text)
	case MATCHER_KIND_PREFIX_SUFFIX:
		return index_prefix_suffix(matcher, text)
	case MATCHER_KIND_CONTAINS:
		return index_contains(matcher, text)
	case MATCHER_KIND_RANGE:
		return index_range(matcher, text)
	case MATCHER_KIND_LIST:
		return index_list(matcher, text)
	case MATCHER_KIND_ROW:
		return index_row(matcher, text)
	case MATCHER_KIND_ANY_OF:
		return index_any_of(matcher, text)
	case MATCHER_KIND_EVERY_OF:
		return index_every_of(matcher, text)
	case MATCHER_KIND_BTREE:
		return -1, nil
	case MATCHER_KIND_PREFIX_ANY:
		return index_prefix_any(matcher, text)
	case MATCHER_KIND_SUFFIX_ANY:
		return index_suffix_any(matcher, text)
	}
	return -1, nil
}

func index_any(matcher *Matcher, text string) (offset int, segments []int) {
	found := Index_Any_Runes(text, matcher.Separators)
	if found == 0 {
		return 0, []int{0}
	}
	if found > 0 {
		text = text[:found]
	}
	segments = make([]int, 0, len(text)+1)
	for byte_offset := range text {
		segments = append(segments, byte_offset)
	}
	segments = append(segments, len(text))
	return 0, segments
}

func index_super(matcher *Matcher, text string) (offset int, segments []int) {
	segments = make([]int, 0, len(text)+1)
	for byte_offset := range text {
		segments = append(segments, byte_offset)
	}
	segments = append(segments, len(text))
	return 0, segments
}

func index_single(matcher *Matcher, text string) (offset int, segments []int) {
	for byte_offset, first := range text {
		if slices.Index(matcher.Separators, first) == -1 {
			return byte_offset, []int{utf8.RuneLen(first)}
		}
	}
	return -1, nil
}

func index_text(matcher *Matcher, text string) (offset int, segments []int) {
	offset = strings.Index(text, matcher.Literal)
	if offset == -1 {
		return -1, nil
	}
	return offset, matcher.Segments
}

func index_max(matcher *Matcher, text string) (offset int, segments []int) {
	segments = make([]int, 0, matcher.Limit+1)
	segments = append(segments, 0)
	count := 0
	for byte_offset, first := range text {
		count++
		if count > matcher.Limit {
			break
		}
		segments = append(segments, byte_offset+utf8.RuneLen(first))
	}
	return 0, segments
}

func index_min(matcher *Matcher, text string) (offset int, segments []int) {
	endpoint_count := len(text) - matcher.Limit + 1
	if endpoint_count <= 0 {
		return -1, nil
	}
	segments = make([]int, 0, endpoint_count)
	count := 0
	for byte_offset, first := range text {
		count++
		if count >= matcher.Limit {
			segments = append(segments, byte_offset+utf8.RuneLen(first))
		}
	}
	if len(segments) == 0 {
		return -1, nil
	}
	return 0, segments
}

func index_prefix(matcher *Matcher, text string) (offset int, segments []int) {
	offset = strings.Index(text, matcher.Prefix)
	if offset == -1 {
		return -1, nil
	}
	prefix_size := len(matcher.Prefix)
	remainder := ""
	if len(text) > offset+prefix_size {
		remainder = text[offset+prefix_size:]
	}
	segments = make([]int, 0, len(remainder)+1)
	segments = append(segments, prefix_size)
	for byte_offset, first := range remainder {
		segments = append(segments, prefix_size+byte_offset+utf8.RuneLen(first))
	}
	return offset, segments
}

func index_suffix(matcher *Matcher, text string) (offset int, segments []int) {
	offset = strings.Index(text, matcher.Suffix)
	if offset == -1 {
		return -1, nil
	}
	return 0, []int{offset + len(matcher.Suffix)}
}

// Reports one match starting at the prefix, whose segment endpoints are every
// suffix occurrence within the tail, in ascending order.
func index_prefix_suffix(matcher *Matcher, text string) (offset int, segments []int) {
	prefix_offset := strings.Index(text, matcher.Prefix)
	if prefix_offset == -1 {
		return -1, nil
	}
	suffix_size := len(matcher.Suffix)
	if suffix_size <= 0 {
		return prefix_offset, []int{len(text) - prefix_offset}
	}
	if len(text)-prefix_offset <= 0 {
		return -1, nil
	}
	segments = make([]int, 0, len(text)-prefix_offset)
	remainder := text[prefix_offset:]
	suffix_offset := strings.LastIndex(remainder, matcher.Suffix)
	for suffix_offset != -1 {
		segments = append(segments, suffix_offset+suffix_size)
		remainder = remainder[:suffix_offset]
		suffix_offset = strings.LastIndex(remainder, matcher.Suffix)
	}
	if len(segments) == 0 {
		return -1, nil
	}
	reverse_segments(segments)
	return prefix_offset, segments
}

func index_contains(matcher *Matcher, text string) (offset int, segments []int) {
	needle_offset := strings.Index(text, matcher.Needle)
	start := 0
	if !matcher.Negated {
		if needle_offset == -1 {
			return -1, nil
		}
		start = needle_offset + len(matcher.Needle)
		if len(text) <= start {
			return 0, []int{start}
		}
		text = text[start:]
	} else if needle_offset != -1 {
		text = text[:needle_offset]
	}
	segments = make([]int, 0, len(text)+1)
	for byte_offset := range text {
		segments = append(segments, start+byte_offset)
	}
	segments = append(segments, start+len(text))
	return 0, segments
}

func index_range(matcher *Matcher, text string) (offset int, segments []int) {
	for byte_offset, first := range text {
		if matcher.Negated != (first >= matcher.Low && first <= matcher.High) {
			return byte_offset, []int{utf8.RuneLen(first)}
		}
	}
	return -1, nil
}

func index_list(matcher *Matcher, text string) (offset int, segments []int) {
	for byte_offset, first := range text {
		if matcher.Negated == (slices.Index(matcher.Runes, first) == -1) {
			return byte_offset, []int{utf8.RuneLen(first)}
		}
	}
	return -1, nil
}

func index_row(matcher *Matcher, text string) (offset int, segments []int) {
	for byte_offset := range text {
		if len(text[byte_offset:]) < matcher.Rune_Width {
			break
		}
		if row_match_all(matcher, text[byte_offset:]) {
			return byte_offset, matcher.Segments
		}
	}
	return -1, nil
}

// Keeps the left-most child match; ties at one offset merge their segment lists
// so every possible match length survives.
func index_any_of(matcher *Matcher, text string) (offset int, segments []int) {
	offset = -1
	segments = make([]int, 0, len(text))
	for child_index := range matcher.Children {
		child_offset, child_segments := Index(&matcher.Children[child_index], text)
		if child_offset == -1 {
			continue
		}
		// A first match, or one that starts earlier, replaces the best so far;
		// the two clauses are split because the linter bans compound ifs.
		adopt := offset == -1
		if !adopt {
			adopt = child_offset < offset
		}
		if adopt {
			offset = child_offset
			segments = append(segments[:0], child_segments...)
			continue
		}
		if child_offset > offset {
			continue
		}
		segments = append_merge(
			&Append_Merge_Input{Target: segments, Source: child_segments})
	}
	if offset == -1 {
		return -1, nil
	}
	return offset, segments
}

// Walks the children left to right, keeping only the segment endpoints on which
// every child so far can agree; an empty agreement set means the conjunction
// cannot match.
func index_every_of(matcher *Matcher, text string) (offset int, segments []int) {
	match_start := 0
	accumulated := 0
	next := make([]int, 0, len(text))
	current := make([]int, 0, len(text))
	remainder := text
	for position := range matcher.Children {
		child_offset, child_segments := Index(&matcher.Children[position], remainder)
		if child_offset == -1 {
			return -1, nil
		}
		if position == 0 {
			current = append(current, child_segments...)
		} else {
			next = next[:0]
			delta := match_start - (child_offset + accumulated)
			for _, base := range current {
				for _, candidate := range child_segments {
					if base+delta == candidate {
						next = append(next, candidate)
					}
				}
			}
			if len(next) == 0 {
				return -1, nil
			}
			current = append(current[:0], next...)
		}
		match_start = child_offset + accumulated
		remainder = text[match_start:]
		accumulated += child_offset
	}
	return match_start, current
}

func index_prefix_any(matcher *Matcher, text string) (offset int, segments []int) {
	offset = strings.Index(text, matcher.Prefix)
	if offset == -1 {
		return -1, nil
	}
	prefix_size := len(matcher.Prefix)
	remainder := text[offset+prefix_size:]
	separator_offset := Index_Any_Runes(remainder, matcher.Separators)
	if separator_offset > -1 {
		remainder = remainder[:separator_offset]
	}
	segments = make([]int, 0, len(remainder)+1)
	segments = append(segments, prefix_size)
	for byte_offset, first := range remainder {
		segments = append(segments, prefix_size+byte_offset+utf8.RuneLen(first))
	}
	return offset, segments
}

func index_suffix_any(matcher *Matcher, text string) (offset int, segments []int) {
	suffix_offset := strings.Index(text, matcher.Suffix)
	if suffix_offset == -1 {
		return -1, nil
	}
	offset = Last_Index_Any_Runes(text[:suffix_offset], matcher.Separators) + 1
	return offset, []int{suffix_offset + len(matcher.Suffix) - offset}
}

// Rune_Width reports the fixed number of runes matcher consumes, or
// RUNE_WIDTH_VARIABLE when that count is not fixed. It is the width the
// composers use to pre-trim search windows (upstream Matcher.Len).
func Rune_Width(matcher Matcher) (width int) {
	switch matcher.Kind {
	case MATCHER_KIND_ANY, MATCHER_KIND_SUPER, MATCHER_KIND_MAX, MATCHER_KIND_MIN,
		MATCHER_KIND_PREFIX, MATCHER_KIND_SUFFIX, MATCHER_KIND_PREFIX_SUFFIX,
		MATCHER_KIND_CONTAINS, MATCHER_KIND_PREFIX_ANY, MATCHER_KIND_SUFFIX_ANY:
		return RUNE_WIDTH_VARIABLE
	case MATCHER_KIND_EMPTY:
		return RUNE_WIDTH_ZERO
	case MATCHER_KIND_SINGLE, MATCHER_KIND_RANGE, MATCHER_KIND_LIST:
		return RUNE_WIDTH_ONE
	case MATCHER_KIND_TEXT, MATCHER_KIND_ROW, MATCHER_KIND_BTREE:
		return matcher.Rune_Width
	case MATCHER_KIND_ANY_OF:
		return rune_width_any_of(matcher)
	case MATCHER_KIND_EVERY_OF:
		return rune_width_every_of(matcher)
	}
	return RUNE_WIDTH_VARIABLE
}

// Fixed only when every child shares one width; a single variable-width child,
// or any disagreement, makes the union variable. The leading-variable reset
// mirrors upstream AnyOf.Len exactly.
func rune_width_any_of(matcher Matcher) (width int) {
	width = RUNE_WIDTH_VARIABLE
	for _, child := range matcher.Children {
		child_width := Rune_Width(child)
		switch {
		case width == RUNE_WIDTH_VARIABLE:
			width = child_width
		case child_width == RUNE_WIDTH_VARIABLE:
			return RUNE_WIDTH_VARIABLE
		case width != child_width:
			return RUNE_WIDTH_VARIABLE
		}
	}
	return width
}

// Preserves upstream EveryOf.Len byte for byte: the width only accumulates once
// it is already positive, so a non-empty conjunction always reports variable
// and an empty one reports zero.
func rune_width_every_of(matcher Matcher) (width int) {
	for _, child := range matcher.Children {
		child_width := Rune_Width(child)
		if width > 0 {
			width += child_width
		} else {
			return RUNE_WIDTH_VARIABLE
		}
	}
	return width
}

func btree_total_rune_width(matcher Matcher) (width int) {
	value_width := Rune_Width(*matcher.Value)
	left_width := btree_child_rune_width(matcher.Left)
	right_width := btree_child_rune_width(matcher.Right)
	if value_width == RUNE_WIDTH_VARIABLE {
		return RUNE_WIDTH_VARIABLE
	}
	if left_width == RUNE_WIDTH_VARIABLE {
		return RUNE_WIDTH_VARIABLE
	}
	if right_width == RUNE_WIDTH_VARIABLE {
		return RUNE_WIDTH_VARIABLE
	}
	return left_width + value_width + right_width
}

// Treats an absent side as zero-width, matching the way upstream left a nil
// child's cached width at its zero value.
func btree_child_rune_width(child *Matcher) (width int) {
	if child == nil {
		return RUNE_WIDTH_ZERO
	}
	return child.Rune_Width
}

// Hands each child in turn the next slice of text as wide as the child's rune
// width, failing on the first child that mismatches or runs out of text.
func row_match_all(matcher *Matcher, text string) (matched bool) {
	start := 0
	for child_index := range matcher.Children {
		child := &matcher.Children[child_index]
		child_width := child.Rune_Width
		rune_count := 0
		last_rune_start := 0
		for byte_offset := range text[start:] {
			last_rune_start = byte_offset
			rune_count++
			if rune_count == child_width {
				break
			}
		}
		if rune_count < child_width {
			return false
		}
		if !matcher_matches(child, text[start:start+last_rune_start+1]) {
			return false
		}
		start += last_rune_start + 1
	}
	return true
}

func row_rune_width_ok(matcher *Matcher, text string) (ok bool) {
	rune_count := 0
	for range text {
		rune_count++
		if rune_count > matcher.Rune_Width {
			return false
		}
	}
	return matcher.Rune_Width == rune_count
}

// Append_Merge_Input pairs the two already-sorted, duplicate-free segment lists
// a merge folds together; the input struct exists because two []int parameters
// are swappable at the call site.
type Append_Merge_Input struct {
	// Target is the first sorted, unique list, and the slice whose backing
	// array the merge reuses.
	Target []int
	// Source is the second sorted, unique list.
	Source []int
}

// Merges two sorted, unique int slices into one sorted, unique slice, reusing
// Target's backing array as upstream appendMerge did.
func append_merge(input *Append_Merge_Input) (merged []int) {
	target_count := len(input.Target)
	source_count := len(input.Source)
	output := make([]int, 0, target_count+source_count)
	x := 0
	y := 0
	for x < target_count || y < source_count {
		if x >= target_count {
			output = append(output, input.Source[y:]...)
			break
		}
		if y >= source_count {
			output = append(output, input.Target[x:]...)
			break
		}
		target_value := input.Target[x]
		source_value := input.Source[y]
		switch {
		case target_value == source_value:
			output = append(output, target_value)
			x++
			y++
		case target_value < source_value:
			output = append(output, target_value)
			x++
		case source_value < target_value:
			output = append(output, source_value)
			y++
		}
	}
	merged = append(input.Target[:0], output...)
	return merged
}

// Reverses the slice in place with a two-pointer swap.
func reverse_segments(segments []int) {
	left := 0
	right := len(segments) - 1
	for left < right {
		segments[left], segments[right] = segments[right], segments[left]
		left++
		right--
	}
}

// String renders matcher as the angle-bracket tree upstream used for debugging;
// it satisfies fmt.Stringer, which is why it survives as a method.
func (matcher Matcher) String() (text string) {
	switch matcher.Kind {
	case MATCHER_KIND_ANY:
		return fmt.Sprintf("<any:![%s]>", string(matcher.Separators))
	case MATCHER_KIND_SUPER:
		return "<super>"
	case MATCHER_KIND_SINGLE:
		return fmt.Sprintf("<single:![%s]>", string(matcher.Separators))
	case MATCHER_KIND_EMPTY:
		return "<nothing>"
	case MATCHER_KIND_TEXT:
		return fmt.Sprintf("<text:`%v`>", matcher.Literal)
	case MATCHER_KIND_MAX:
		return fmt.Sprintf("<max:%d>", matcher.Limit)
	case MATCHER_KIND_MIN:
		return fmt.Sprintf("<min:%d>", matcher.Limit)
	case MATCHER_KIND_PREFIX:
		return fmt.Sprintf("<prefix:%s>", matcher.Prefix)
	case MATCHER_KIND_SUFFIX:
		return fmt.Sprintf("<suffix:%s>", matcher.Suffix)
	case MATCHER_KIND_PREFIX_SUFFIX:
		return fmt.Sprintf("<prefix_suffix:[%s,%s]>", matcher.Prefix, matcher.Suffix)
	case MATCHER_KIND_CONTAINS:
		return string_contains(matcher)
	case MATCHER_KIND_RANGE:
		return string_range(matcher)
	case MATCHER_KIND_LIST:
		return string_list(matcher)
	case MATCHER_KIND_ROW:
		return fmt.Sprintf(
			"<row_%d:[%s]>", matcher.Rune_Width, string_children(matcher.Children))
	case MATCHER_KIND_ANY_OF:
		return fmt.Sprintf("<any_of:[%s]>", string_children(matcher.Children))
	case MATCHER_KIND_EVERY_OF:
		return fmt.Sprintf("<every_of:[%s]>", string_children(matcher.Children))
	case MATCHER_KIND_BTREE:
		return string_btree(matcher)
	case MATCHER_KIND_PREFIX_ANY:
		return fmt.Sprintf(
			"<prefix_any:%s![%s]>", matcher.Prefix, string(matcher.Separators))
	case MATCHER_KIND_SUFFIX_ANY:
		return fmt.Sprintf(
			"<suffix_any:![%s]%s>", string(matcher.Separators), matcher.Suffix)
	}
	return ""
}

func string_contains(matcher Matcher) (text string) {
	negation := ""
	if matcher.Negated {
		negation = "!"
	}
	return fmt.Sprintf("<contains:%s[%s]>", negation, matcher.Needle)
}

func string_range(matcher Matcher) (text string) {
	negation := ""
	if matcher.Negated {
		negation = "!"
	}
	return fmt.Sprintf("<range:%s[%s,%s]>", negation, string(matcher.Low), string(matcher.High))
}

func string_list(matcher Matcher) (text string) {
	negation := ""
	if matcher.Negated {
		negation = "!"
	}
	return fmt.Sprintf("<list:%s[%s]>", negation, string(matcher.Runes))
}

func string_children(children []Matcher) (text string) {
	rendered := make([]string, 0, len(children))
	for _, child := range children {
		rendered = append(rendered, child.String())
	}
	return strings.Join(rendered, ",")
}

func string_btree(matcher Matcher) (text string) {
	left := "<nil>"
	if matcher.Left != nil {
		left = matcher.Left.String()
	}
	right := "<nil>"
	if matcher.Right != nil {
		right = matcher.Right.String()
	}
	return fmt.Sprintf("<btree:[%s<-%s->%s]>", left, matcher.Value.String(), right)
}

// The rune-set searches below are a dependency-injected port of the util/runes and util/strings
// helpers from github.com/gobwas/glob (MIT, Sergey Kamardin); see
// LICENSE.mit.kamardin.
//
// Only the two set searches the standard library lacks survive the port. The
// glob matchers bound a wildcard at the first SEPARATOR they meet, and a
// separator set is ordered by priority, so they need the first rune of the set
// (in set order) that occurs — not the left-most matching position that
// strings.IndexAny / strings.LastIndexAny report. The remaining upstream helpers
// (Equal, IndexRune, and the rest) were exact re-implementations of slices.Equal
// and slices.Index, so callers use those directly instead.

// Index_Any_Runes returns the byte offset in text of the first occurrence of the
// first rune in set — walking set in order — that appears, or -1 when none of the
// set's runes occur.
func Index_Any_Runes(text string, set []rune) (offset int) {
	for _, member := range set {
		offset = strings.IndexRune(text, member)
		if offset != -1 {
			return offset
		}
	}
	return -1
}

// Last_Index_Any_Runes returns the byte offset in text of the last occurrence of
// the first rune in set — walking set in order — that appears, or -1 when none of
// the set's runes occur.
func Last_Index_Any_Runes(text string, set []rune) (offset int) {
	for _, member := range set {
		offset = strings.LastIndex(text, string(member))
		if offset != -1 {
			return offset
		}
	}
	return -1
}

// The rest of this file restores the upstream github.com/gobwas/glob/util/runes
// helpers verbatim in behavior. They operate on []rune (rune-indexed), unlike the
// two string helpers above, and back the ported runes tests and benchmarks. Each
// helper whose two operands share the []rune type takes a keyed *_Input so a call
// site cannot silently transpose them.

// Index_Rune returns the rune index of the first occurrence of needle in source,
// or -1 when needle does not occur. Ports upstream runes.IndexRune.
func Index_Rune(source []rune, needle rune) (index int) {
	for source_index, candidate := range source {
		if candidate == needle {
			return source_index
		}
	}
	return -1
}

// Index_Last_Rune returns the rune index of the last occurrence of needle in
// source, or -1 when needle does not occur. Ports upstream runes.IndexLastRune.
func Index_Last_Rune(source []rune, needle rune) (index int) {
	for source_index := len(source) - 1; source_index >= 0; source_index-- {
		if source[source_index] == needle {
			return source_index
		}
	}
	return -1
}

// Equal_Input pairs the two rune slices whose element-wise equality is tested.
type Equal_Input struct {
	// Left is one of the two slices compared; equality is symmetric in the two.
	Left []rune
	// Right is the other slice compared.
	Right []rune
}

// Equal reports whether Left and Right hold the same runes in the same order.
// Ports upstream runes.Equal.
func Equal(input *Equal_Input) (equal bool) {
	if len(input.Left) != len(input.Right) {
		return false
	}
	for index := 0; index < len(input.Left); index++ {
		if input.Left[index] != input.Right[index] {
			return false
		}
	}
	return true
}

// Index_Runes_Input pairs the rune slice searched with the subsequence sought.
type Index_Runes_Input struct {
	// Source is the rune slice searched.
	Source []rune
	// Needle is the contiguous rune subsequence sought within Source.
	Needle []rune
}

// Index_Runes returns the rune index of the first occurrence of Needle within
// Source, or -1 when Needle does not occur. Ports upstream runes.Index; the
// _Runes suffix keeps it distinct from the matcher Index this package already
// exports.
func Index_Runes(input *Index_Runes_Input) (index int) {
	source := input.Source
	needle := input.Needle
	source_count := len(source)
	needle_count := len(needle)
	switch {
	case needle_count == 0:
		return 0
	case needle_count == 1:
		return Index_Rune(source, needle[0])
	case needle_count == source_count:
		if Equal(&Equal_Input{Left: source, Right: needle}) {
			return 0
		}
		return -1
	case needle_count > source_count:
		return -1
	}
	for start := 0; start < source_count && source_count-start >= needle_count; start++ {
		matched := true
		for element_index := 0; element_index < needle_count; element_index++ {
			if source[start+element_index] != needle[element_index] {
				matched = false
				break
			}
		}
		if matched {
			return start
		}
	}
	return -1
}

// Last_Index_Input pairs the rune slice searched with the subsequence sought
// from the right.
type Last_Index_Input struct {
	// Source is the rune slice searched.
	Source []rune
	// Needle is the contiguous rune subsequence sought within Source.
	Needle []rune
}

// Last_Index returns the rune index of the last occurrence of Needle within
// Source, or -1 when Needle does not occur. An empty Needle reports the length of
// Source. Ports upstream runes.LastIndex.
func Last_Index(input *Last_Index_Input) (index int) {
	source := input.Source
	needle := input.Needle
	source_count := len(source)
	needle_count := len(needle)
	switch {
	case needle_count == 0:
		if source_count == 0 {
			return 0
		}
		return source_count
	case needle_count == 1:
		return Index_Last_Rune(source, needle[0])
	case needle_count == source_count:
		if Equal(&Equal_Input{Left: source, Right: needle}) {
			return 0
		}
		return -1
	case needle_count > source_count:
		return -1
	}
	// Scan candidate windows right-to-left. window_start is the offset of the
	// leftmost rune of the needle-width window ending at start; computing it once
	// keeps the inner compare a forward walk (upstream folded the same offset into
	// each subscript as i-(ln-y-1), which is identical to window_start+element).
	for start := source_count - 1; start >= 0 && start >= needle_count; start-- {
		window_start := start - needle_count + 1
		matched := true
		for element_index := 0; element_index < needle_count; element_index++ {
			if source[window_start+element_index] != needle[element_index] {
				matched = false
				break
			}
		}
		if matched {
			return window_start
		}
	}
	return -1
}

// Index_Any_Input pairs the rune slice searched with the set of runes sought.
type Index_Any_Input struct {
	// Source is the rune slice searched.
	Source []rune
	// Characters is the set of runes any one of which ends the search.
	Characters []rune
}

// Index_Any returns the rune index of the first rune in Source that is also in
// Characters, or -1 when none is present. Ports upstream runes.IndexAny.
func Index_Any(input *Index_Any_Input) (index int) {
	if len(input.Characters) == 0 {
		return -1
	}
	for source_index, candidate := range input.Source {
		for _, wanted := range input.Characters {
			if candidate == wanted {
				return source_index
			}
		}
	}
	return -1
}

// Contains_Input pairs the rune slice searched with the subsequence whose
// presence is tested.
type Contains_Input struct {
	// Source is the rune slice searched.
	Source []rune
	// Needle is the contiguous rune subsequence whose presence is tested.
	Needle []rune
}

// Contains reports whether Needle occurs within Source. Ports upstream
// runes.Contains.
func Contains(input *Contains_Input) (contained bool) {
	return Index_Runes(&Index_Runes_Input{Source: input.Source, Needle: input.Needle}) >= 0
}

// Max returns the greatest rune in source, or 0 when source is empty. Ports
// upstream runes.Max.
func Max(source []rune) (maximum rune) {
	for _, candidate := range source {
		if candidate > maximum {
			maximum = candidate
		}
	}
	return maximum
}

// Min returns the least rune in source, or -1 when source is empty. Ports
// upstream runes.Min.
func Min(source []rune) (minimum rune) {
	minimum = rune(-1)
	for _, candidate := range source {
		if minimum == -1 {
			minimum = candidate
			continue
		}
		if candidate < minimum {
			minimum = candidate
		}
	}
	return minimum
}

// Has_Prefix_Input pairs a rune slice with the leading run tested against it.
type Has_Prefix_Input struct {
	// Source is the rune slice tested.
	Source []rune
	// Prefix is the leading run Source must begin with.
	Prefix []rune
}

// Has_Prefix reports whether Source begins with Prefix. Ports upstream
// runes.HasPrefix.
func Has_Prefix(input *Has_Prefix_Input) (has bool) {
	if len(input.Source) < len(input.Prefix) {
		return false
	}
	head := input.Source[0:len(input.Prefix)]
	return Equal(&Equal_Input{Left: head, Right: input.Prefix})
}

// Has_Suffix_Input pairs a rune slice with the trailing run tested against it.
type Has_Suffix_Input struct {
	// Source is the rune slice tested.
	Source []rune
	// Suffix is the trailing run Source must end with.
	Suffix []rune
}

// Has_Suffix reports whether Source ends with Suffix. Ports upstream
// runes.HasSuffix.
func Has_Suffix(input *Has_Suffix_Input) (has bool) {
	if len(input.Source) < len(input.Suffix) {
		return false
	}
	tail := input.Source[len(input.Source)-len(input.Suffix):]
	return Equal(&Equal_Input{Left: tail, Right: input.Suffix})
}
