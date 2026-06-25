package vcs_test

import (
	"strings"
	"testing"

	"github.com/james-orcales/james-orcales/lint/internal/vcs"
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
	if !flagged(input, "commit subject is") {
		t.Fatal("an over-long subject must be flagged")
	}
}

// Test_Commits_Conventional_Subjects verifies a non-conventional subject is flagged.
func Test_Commits_Conventional_Subjects(t *testing.T) {
	t.Parallel()
	input := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "did some stuff"},
	}}
	if !flagged(input, "non-conventional commit subject") {
		t.Fatal("a non-conventional subject must be flagged")
	}
}

// Test_Commits_Fixup_Commits verifies a fixup! subject is flagged.
func Test_Commits_Fixup_Commits(t *testing.T) {
	t.Parallel()
	input := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "fixup! feat: thing"},
	}}
	if !flagged(input, "fixup commit on branch") {
		t.Fatal("a fixup commit must be flagged")
	}
}

// Test_Commits_Merge_Commits verifies a non-subtree merge commit is flagged.
func Test_Commits_Merge_Commits(t *testing.T) {
	t.Parallel()
	input := vcs.Check_Input{Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "Merge branch 'feature' into main"},
	}}
	if !flagged(input, "merge commit on branch") {
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
	if flagged(pull, "non-conventional") {
		t.Fatal("a GitHub pull-request merge subject must be exempt")
	}
	octopus := vcs.Check_Input{Non_Merge_Commits: []vcs.Commit{
		{Hash: "abc", Subject: "Merge abc1234 into def5678"},
	}}
	if flagged(octopus, "non-conventional") {
		t.Fatal("a GitHub sha-into-sha merge subject must be exempt")
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
