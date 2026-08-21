package levenshtein_test

import (
	"testing"

	"local/james-orcales/shared/diff/levenshtein"
	"local/james-orcales/shared/strings"
	"local/james-orcales/shared/testify"
)

// Test_Distance_Cases covers equality, edits, symmetry, Unicode, and bounds.
func Test_Distance_Cases(t *testing.T) {
	workspace := levenshtein.Workspace{
		From:     make([]rune, levenshtein.RUNE_COUNT_MAXIMUM),
		To:       make([]rune, levenshtein.RUNE_COUNT_MAXIMUM),
		Previous: make([]int, levenshtein.ROW_COUNT),
		Current:  make([]int, levenshtein.ROW_COUNT),
	}
	check := func(from string, to string, want levenshtein.Distance_Value) {
		t.Helper()
		got, status := levenshtein.Distance(levenshtein.Distance_Input{
			Workspace: &workspace,
			From:      levenshtein.From_Text_Unvalidated(from),
			To:        levenshtein.To_Text_Unvalidated(to),
		})
		if status != levenshtein.STATUS_OK {
			t.Fatalf("Distance status = %d, want STATUS_OK", status)
		}
		if got != want {
			t.Errorf("Distance(%q, %q) = %d, want %d", from, to, got, want)
		}
	}
	check("", "", 0)
	check("a", "a", 0)
	check("a", "b", 1)
	check("ab", "cd", 2)
	check("ab", "abc", 1)
	check("abc", "ab", 1)
	check("kitten", "sitting", 3)
	check("sitting", "kitten", 3)
	check("😀", "😄", 1)
	maximum := string(make([]byte, strings.TEXT_SIZE_MAXIMUM))
	check("", maximum, levenshtein.RUNE_COUNT_MAXIMUM)
	check(maximum, "", levenshtein.RUNE_COUNT_MAXIMUM)

	_, status := levenshtein.Distance(levenshtein.Distance_Input{
		Workspace: &workspace,
		From: levenshtein.From_Text_Unvalidated(
			make([]byte, levenshtein.TEXT_SIZE_UNVALIDATED_MAXIMUM),
		),
		To: "",
	})
	if status != levenshtein.STATUS_INPUT_INVALID {
		t.Errorf("oversized Distance status = %d, want STATUS_INPUT_INVALID", status)
	}

	input := levenshtein.Distance_Input{
		Workspace: &workspace, From: "kitten", To: "sitting",
	}
	testify.Zero_Allocation(t, func() {
		distance, allocation_status := levenshtein.Distance(input)
		if allocation_status != levenshtein.STATUS_OK {
			t.Fatal("distance rejected")
		}
		if distance != 3 {
			t.Fatal("distance changed")
		}
	})
}

// Test_Closest_Cases covers match, miss, tie, empty set, and invalid candidate.
func Test_Closest_Cases(t *testing.T) {
	workspace := levenshtein.Workspace{
		From:     make([]rune, levenshtein.RUNE_COUNT_MAXIMUM),
		To:       make([]rune, levenshtein.RUNE_COUNT_MAXIMUM),
		Previous: make([]int, levenshtein.ROW_COUNT),
		Current:  make([]int, levenshtein.ROW_COUNT),
	}
	commands := []string{"help", "add", "list", "delete"}
	match, found, status := levenshtein.Closest(levenshtein.Closest_Input{
		Workspace: &workspace, Target: "lst", Candidates: commands,
	})
	if status != levenshtein.STATUS_OK {
		t.Errorf("Closest status = %d, want STATUS_OK", status)
	} else if !found {
		t.Error("Closest missed list")
	} else if match != "list" {
		t.Errorf(
			"Closest = (%q, %v, %d), want (list, true, STATUS_OK)",
			match, found, status,
		)
	}

	match, found, status = levenshtein.Closest(levenshtein.Closest_Input{
		Workspace: &workspace, Target: "abcd", Candidates: []string{"abce", "abcf"},
	})
	if status != levenshtein.STATUS_OK {
		t.Errorf("tie Closest status = %d, want STATUS_OK", status)
	} else if !found {
		t.Error("tie Closest missed abce")
	} else if match != "abce" {
		t.Errorf("tie Closest = (%q, %v, %d)", match, found, status)
	}

	_, found, status = levenshtein.Closest(levenshtein.Closest_Input{
		Workspace: &workspace, Target: "zzzzzzzz", Candidates: commands,
	})
	if status != levenshtein.STATUS_OK {
		t.Errorf("miss Closest status = %d, want STATUS_OK", status)
	} else if found {
		t.Errorf("miss Closest = (%v, %d), want (false, STATUS_OK)", found, status)
	}

	_, found, status = levenshtein.Closest(levenshtein.Closest_Input{
		Workspace: &workspace, Target: "", Candidates: nil,
	})
	if status != levenshtein.STATUS_OK {
		t.Errorf("empty Closest status = %d, want STATUS_OK", status)
	} else if found {
		t.Errorf("empty Closest = (%v, %d), want (false, STATUS_OK)", found, status)
	}

	check_closest_boundaries(t, &workspace)
	check_closest_allocation(t, &workspace)
}

func check_closest_boundaries(t *testing.T, workspace *levenshtein.Workspace) {
	t.Helper()
	_, _, status := levenshtein.Closest(levenshtein.Closest_Input{
		Workspace: workspace,
		Target:    "",
		Candidates: []string{
			string(make([]byte, levenshtein.TEXT_SIZE_UNVALIDATED_MAXIMUM)),
		},
	})
	if status != levenshtein.STATUS_INPUT_INVALID {
		t.Errorf("invalid Closest status = %d, want STATUS_INPUT_INVALID", status)
	}

	for _, target := range []string{
		"a", "ab", string(make([]byte, levenshtein.TEXT_SIZE_MAXIMUM)),
	} {
		match, found, exact_status := levenshtein.Closest(levenshtein.Closest_Input{
			Workspace:  workspace,
			Target:     levenshtein.Target_Text_Unvalidated(target),
			Candidates: []string{target},
		})
		if exact_status != levenshtein.STATUS_OK {
			t.Errorf("exact Closest status = %d, want STATUS_OK", exact_status)
		} else if !found {
			t.Errorf("exact Closest missed %d-byte target", len(target))
		} else if string(match) != target {
			t.Errorf(
				"exact Closest returned %d bytes, want %d", len(match), len(target),
			)
		}
	}

	maximum_candidates := make([]string, levenshtein.CANDIDATE_COUNT_MAXIMUM)
	_, found, status := levenshtein.Closest(levenshtein.Closest_Input{
		Workspace: workspace, Target: "", Candidates: maximum_candidates,
	})
	if status != levenshtein.STATUS_OK {
		t.Errorf("maximum candidate status = %d, want STATUS_OK", status)
	} else if !found {
		t.Error("maximum candidate set missed exact empty candidate")
	}

	_, _, status = levenshtein.Closest(levenshtein.Closest_Input{
		Workspace: workspace,
		Target: levenshtein.Target_Text_Unvalidated(
			make([]byte, levenshtein.TEXT_SIZE_UNVALIDATED_MAXIMUM),
		),
		Candidates: nil,
	})
	if status != levenshtein.STATUS_INPUT_INVALID {
		t.Errorf("oversized target status = %d, want STATUS_INPUT_INVALID", status)
	}
}

func check_closest_allocation(t *testing.T, workspace *levenshtein.Workspace) {
	t.Helper()
	closest := levenshtein.Closest_Input{
		Workspace:  workspace,
		Target:     "lst",
		Candidates: []string{"help", "add", "list", "delete"},
	}
	testify.Zero_Allocation(t, func() {
		match, found, status := levenshtein.Closest(closest)
		if status != levenshtein.STATUS_OK {
			t.Fatal("closest rejected")
		}
		if !found {
			t.Fatal("closest missed")
		}
		if match != "list" {
			t.Fatal("closest changed")
		}
	})
}
