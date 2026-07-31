package vcs_test

import (
	"strings"
	"testing"

	"local/james-orcales/lint/internal/vcs"
)

// This file dogfoods the doctrine it enforces: each leaf test builds the commit
// input, violates one rule, and asserts Check reports it. The leaf tests are the
// file's leading declarations, in leaf order; the helper follows them.

// Test_Commits_Subject_Size verifies a subject over the character cap is flagged.
func Test_Commits_Subject_Size(t *testing.T) {
	t.Parallel()
	input := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "feat: " + strings.Repeat("x", 200)},
	}}
	if !flagged(input, "The commit subject has") {
		t.Fatal("an over-long subject must be flagged")
	}
}

// Test_Commits_Conventional_Subjects verifies a non-conventional subject is flagged.
func Test_Commits_Conventional_Subjects(t *testing.T) {
	t.Parallel()
	input := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "did some stuff"},
	}}
	if !flagged(input, "is not conventional") {
		t.Fatal("a non-conventional subject must be flagged")
	}
}

// Test_Commits_Fixup_Commits verifies a fixup! subject is flagged.
func Test_Commits_Fixup_Commits(t *testing.T) {
	t.Parallel()
	input := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "fixup! feat: thing"},
	}}
	if !flagged(input, "is a fixup commit") {
		t.Fatal("a fixup commit must be flagged")
	}
}

// Test_Commits_Merge_Commits verifies a non-subtree merge commit is flagged.
func Test_Commits_Merge_Commits(t *testing.T) {
	t.Parallel()
	input := vcs.Check_Input{Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "Merge branch 'feature' into main"},
	}}
	if !flagged(input, "is a merge commit") {
		t.Fatal("a merge commit must be flagged")
	}
}

// Test_Commits_Synthetic_Merge_Exempt verifies a GitHub synthetic-merge subject
// that lands in the non-merge tier on a shallow checkout is not flagged.
func Test_Commits_Synthetic_Merge_Exempt(t *testing.T) {
	t.Parallel()
	pull := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "Merge pull request #42 from owner/branch"},
	}}
	if flagged(pull, "is not conventional") {
		t.Fatal("a GitHub pull-request merge subject must be exempt")
	}
	octopus := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "Merge abc1234 into def5678"},
	}}
	if flagged(octopus, "is not conventional") {
		t.Fatal("a GitHub sha-into-sha merge subject must be exempt")
	}
}

// Test_Commits_Revert_Subject_Exempt verifies Git's generated revert form bypasses both subject
// checks, including when the reverted subject makes the wrapper exceed the ordinary size cap.
func Test_Commits_Revert_Subject_Exempt(t *testing.T) {
	t.Parallel()
	input := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "Revert \"feat: add thing\""},
		{Hash: "def", Subject: "Revert \"feat: " + strings.Repeat("x", 200) + "\""},
	}}
	if diags := vcs.Check(&input); len(diags) != 0 {
		t.Fatalf("exact Git revert subjects must be exempt: %v", diags)
	}
}

// Test_Commits_Malformed_Revert_Subjects verifies the exemption cannot be claimed by an empty,
// unquoted, unterminated, or otherwise approximate Revert prefix.
func Test_Commits_Malformed_Revert_Subjects(t *testing.T) {
	t.Parallel()
	subjects := []string{
		"Revert \"\"",
		"Revert feat: add thing",
		"Revert \"feat: add thing",
		"Revert \"feat: add thing\" trailing",
	}
	for _, subject := range subjects {
		input := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{{
			Hash: "abc", Subject: subject,
		}}}
		if !flagged(input, "is not conventional") {
			t.Fatalf("malformed revert subject must be rejected: %q", subject)
		}
	}
}

// Reports whether Check emits a diagnostic whose message contains fragment for
// the given input.
func flagged(input vcs.Check_Input, fragment string) (found bool) {
	for _, d := range vcs.Check(&input) {
		if strings.Contains(d.Message, fragment) {
			return true
		}
	}
	return false
}
