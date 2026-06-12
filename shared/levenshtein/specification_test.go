package levenshtein_test

import (
	"testing"

	"github.com/james-orcales/james-orcales/shared/levenshtein"
)

// Test_Distance_Cases verifies the edit distance for equality, empties, the three
// single edits, and symmetry.
func Test_Distance_Cases(t *testing.T) {
	check := func(from, to string, want int) {
		t.Helper()
		got := levenshtein.Distance(levenshtein.Distance_Input{From: from, To: to})
		if got != want {
			t.Errorf("Distance(%q, %q) = %d, want %d", from, to, got, want)
		}
	}
	check("abc", "abc", 0)
	check("", "abc", 3)
	check("abc", "", 3)
	check("ab", "abc", 1)  // insertion
	check("abc", "ab", 1)  // deletion
	check("cat", "car", 1) // substitution
	check("priorty", "priority", 1)
	check("kitten", "sitting", 3)
	check("sitting", "kitten", 3) // symmetric
}

// Test_Closest_Cases verifies a near-miss matches, a wild miss and an empty set do
// not, and ties keep the earliest candidate.
func Test_Closest_Cases(t *testing.T) {
	commands := []string{"help", "add", "list", "delete"}

	match, found := levenshtein.Closest(levenshtein.Closest_Input{
		Target: "lst", Candidates: commands,
	})
	if match != "list" {
		t.Errorf("expected list, got %q (found=%v)", match, found)
	}

	_, found = levenshtein.Closest(levenshtein.Closest_Input{
		Target: "zzzzzzzz", Candidates: commands,
	})
	if found {
		t.Error("expected no match for a wild miss")
	}

	_, found = levenshtein.Closest(levenshtein.Closest_Input{
		Target: "anything", Candidates: nil,
	})
	if found {
		t.Error("expected no match for an empty candidate set")
	}

	// Two candidates equally near the target: the earliest wins.
	match, found = levenshtein.Closest(levenshtein.Closest_Input{
		Target: "abcd", Candidates: []string{"abce", "abcf"},
	})
	if match != "abce" {
		t.Errorf("expected abce, got %q (found=%v)", match, found)
	}
}
