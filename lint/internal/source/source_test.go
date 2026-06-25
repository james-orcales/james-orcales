package source_test

import (
	"testing"

	"github.com/james-orcales/james-orcales/lint/internal/source"
)

// Test_Parse_Glob_Pattern_Negate verifies a leading "!" is stripped into Negate,
// leaving Core identical to the un-negated entry's reduction.
func Test_Parse_Glob_Pattern_Negate(t *testing.T) {
	t.Parallel()
	negated := source.Parse_Glob_Pattern("!pkg/keep")
	if !negated.Negate {
		t.Fatal("a leading ! must set Negate")
	}
	plain := source.Parse_Glob_Pattern("pkg/keep")
	if negated.Core != plain.Core {
		t.Fatalf("Core must ignore the !: got %q want %q", negated.Core, plain.Core)
	}
	if plain.Negate {
		t.Fatal("an entry with no leading ! must not set Negate")
	}
}

// Test_Path_Matches_Glob_Negation_Overrides_Broader_Match verifies a narrower
// negated entry holds one path out of a broader positive entry's release.
func Test_Path_Matches_Glob_Negation_Overrides_Broader_Match(t *testing.T) {
	t.Parallel()
	patterns := []string{"pkg/**", "!pkg/keep"}
	if source.Path_Matches_Glob("pkg/keep", patterns) {
		t.Error("!pkg/keep must veto the broader pkg/** match")
	}
	if !source.Path_Matches_Glob("pkg/other", patterns) {
		t.Error("pkg/** must still match a path the negation does not name")
	}
}

// Test_Path_Matches_Glob_Negation_Wins_Regardless_Of_Order verifies negation
// always wins whether the negated entry is listed before or after the positive
// entry it overrides.
func Test_Path_Matches_Glob_Negation_Wins_Regardless_Of_Order(t *testing.T) {
	t.Parallel()
	negation_first := []string{"!pkg/keep", "pkg/**"}
	negation_last := []string{"pkg/**", "!pkg/keep"}
	if source.Path_Matches_Glob("pkg/keep", negation_first) {
		t.Error("negation listed before the positive entry must still win")
	}
	if source.Path_Matches_Glob("pkg/keep", negation_last) {
		t.Error("negation listed after the positive entry must still win")
	}
}
