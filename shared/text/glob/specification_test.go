package glob_test

import (
	"strings"
	"testing"

	"local/james-orcales/shared/text/glob"
)

// Test_Compile_Builds_A_Matcher checks a plain pattern matches only itself.
func Test_Compile_Builds_A_Matcher(t *testing.T) {
	pattern, err := glob.Compile("google.com")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if !glob.Match(pattern, "google.com") {
		t.Fatalf("plain pattern must match its own text")
	}
	if glob.Match(pattern, "gobwas.com") {
		t.Fatalf("plain pattern must reject other text")
	}
}

// Test_Wildcards_Match_Runs checks *, **, and ? with no separators.
func Test_Wildcards_Match_Runs(t *testing.T) {
	if !glob.Match(glob.Must_Compile("a*c"), "a12345c") {
		t.Fatalf("* must span a run")
	}
	if !glob.Match(glob.Must_Compile("a?c"), "a1c") {
		t.Fatalf("? must match exactly one character")
	}
	if glob.Match(glob.Must_Compile("a?c"), "ac") {
		t.Fatalf("? must not match zero characters")
	}
}

// Test_Separators_Bound_Wildcards checks * stops at a separator while ** spans it.
func Test_Separators_Bound_Wildcards(t *testing.T) {
	if glob.Match(glob.Must_Compile("a.*", '.'), "a.b.c") {
		t.Fatalf("* must not cross the separator")
	}
	if !glob.Match(glob.Must_Compile("a.**", '.'), "a.b.c") {
		t.Fatalf("** must cross the separator")
	}
}

// Test_Character_Classes_Match_Sets checks class membership, ranges, and negation.
func Test_Character_Classes_Match_Sets(t *testing.T) {
	if !glob.Match(glob.Must_Compile("/{rate,[a-z][a-z][a-z]}*"), "/usd") {
		t.Fatalf("range class must match a lowercase run")
	}
	if !glob.Match(glob.Must_Compile("[!a]*"), "this is a test3") {
		t.Fatalf("negated class must match a non-member first character")
	}
}

// Test_Alternatives_Match_Any_Branch checks a {...} group matches any branch.
func Test_Alternatives_Match_Any_Branch(t *testing.T) {
	pattern := glob.Must_Compile("{abc,def}ghi")
	if !glob.Match(pattern, "defghi") {
		t.Fatalf("alternative branch must match")
	}
	if glob.Match(pattern, "xyzghi") {
		t.Fatalf("non-branch must not match")
	}
}

// Test_Escaping_Matches_Literally checks a backslash escapes a metacharacter.
func Test_Escaping_Matches_Literally(t *testing.T) {
	if !glob.Match(glob.Must_Compile(`\*`), "*") {
		t.Fatalf("escaped asterisk must match a literal asterisk")
	}
	if glob.Match(glob.Must_Compile(`\*`), "a") {
		t.Fatalf("escaped asterisk must not act as a wildcard")
	}
}

// Test_Must_Compile_Panics_On_Bad_Pattern checks the panic contract for Must_Compile.
func Test_Must_Compile_Panics_On_Bad_Pattern(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatalf("Must_Compile must panic on a malformed pattern")
		}
	}()
	// An unterminated character class is malformed, so Compile errors and
	// Must_Compile must turn that into a panic.
	glob.Must_Compile("[abc")
}

// Test_Quote_Meta_Escapes_Metacharacters checks the escaped text compiles to a
// literal matcher for itself.
func Test_Quote_Meta_Escapes_Metacharacters(t *testing.T) {
	raw := `{foo*}`
	quoted := glob.Quote_Meta(raw)
	if quoted != `\{foo\*\}` {
		t.Fatalf("Quote_Meta(%q) = %q", raw, quoted)
	}
	if !glob.Match(glob.Must_Compile(quoted), raw) {
		t.Fatalf("compiled quoted pattern must match the original text literally")
	}
}

// Test_Deep_Nesting_Is_Rejected checks a pattern past the depth cap errors while a
// shallow one compiles.
func Test_Deep_Nesting_Is_Rejected(t *testing.T) {
	over := strings.Repeat("{", glob.PATTERN_DEPTH_MAX+1) + "a" +
		strings.Repeat("}", glob.PATTERN_DEPTH_MAX+1)
	if _, err := glob.Compile(over); err == nil {
		t.Fatalf("nesting past PATTERN_DEPTH_MAX must be rejected")
	}
	if _, err := glob.Compile("{{{a}}}"); err != nil {
		t.Fatalf("shallow nesting must compile: %v", err)
	}
}
