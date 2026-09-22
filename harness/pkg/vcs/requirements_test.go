package vcs_test

import "testing"

import "local/james-orcales/harness/pkg/vcs"
import "local/james-orcales/shared/testify"

// Test_Commits_Subject_Size verifies a subject past the cap is flagged, merge or not, and one at
// the cap is not.
func Test_Commits_Subject_Size(t *testing.T) {
  long := "feat: " + repeat("x", vcs.SUBJECT_SIZE_MAXIMUM)
  testify.Equal(t, vcs.VERDICT_SIZE, vcs.Judge(vcs.Subject(long), false))
  testify.Equal(t, vcs.VERDICT_SIZE, vcs.Judge(vcs.Subject(long), true))
  testify.Equal(t, vcs.VERDICT_NONE, vcs.Judge(vcs.Subject(long[:vcs.SUBJECT_SIZE_MAXIMUM]), false))
}

// Test_Commits_Conventional_Subjects verifies a subject off the conventional shape is flagged,
// and the synthetic merge and exact revert forms are not.
func Test_Commits_Conventional_Subjects(t *testing.T) {
  for _, subject := range []string{"feat: a", "fix(harness): a", "feat!: a", "feat(a/b)!: a"} {
    testify.Equal(t, vcs.VERDICT_NONE, vcs.Judge(vcs.Subject(subject), false), subject)
  }
  for _, subject := range []string{
    "did some stuff",
    "Feat: a",
    "feat:a",
    "feat: ",
    "feat(): a",
    "feat(a: b",
    ": a",
    "Revert \"\"",
    "Revert feat: a",
    "Revert \"feat: a",
    "Revert \"feat: a\" trailing",
  } {
    testify.Equal(t, vcs.VERDICT_CONVENTIONAL, vcs.Judge(vcs.Subject(subject), false), subject)
  }
  exempt := []string{
    "Merge pull request #42 from owner/branch",
    "Merge abc1234 into def5678",
    "Revert \"feat: a\"",
    "Revert \"feat: " + repeat("x", vcs.SUBJECT_SIZE_MAXIMUM) + "\"",
  }
  for _, subject := range exempt {
    testify.Equal(t, vcs.VERDICT_NONE, vcs.Judge(vcs.Subject(subject), false), subject)
  }
  for _, subject := range []string{"Merge pull request #a from b", "Merge abc123 into def5678"} {
    testify.Equal(t, vcs.VERDICT_CONVENTIONAL, vcs.Judge(vcs.Subject(subject), false), subject)
  }
}

// Test_Commits_Fixup_Commits verifies a fixup, squash, or review follow-up subject is flagged,
// and a subject that only mentions a review is not.
func Test_Commits_Fixup_Commits(t *testing.T) {
  for _, subject := range []string{
    "fixup! feat: a",
    "squash! feat: a",
    "fix: Address review comments",
    "fix: apply review nits",
    "fix: cr comments",
    "fix: code review comments",
    "fix: review fixes",
    "fix: review nit",
  } {
    testify.Equal(t, vcs.VERDICT_FIXUP, vcs.Judge(vcs.Subject(subject), false), subject)
  }
  testify.Equal(t, vcs.VERDICT_NONE, vcs.Judge("feat: add the review page", false))
}

// Test_Commits_Merge_Commits verifies a merge commit is flagged, and a git-subtree merge and a
// synthetic merge are not.
func Test_Commits_Merge_Commits(t *testing.T) {
  testify.Equal(t, vcs.VERDICT_MERGE, vcs.Judge("Merge branch 'feature' into main", true))
  testify.Equal(t, vcs.VERDICT_MERGE, vcs.Judge("feat: a", true))
  for _, subject := range []string{
    "Add 'third_party/a/' from commit 'abc'",
    "Merge commit 'abc' as 'third_party/a'",
    "Merge pull request #42 from owner/branch",
  } {
    testify.Equal(t, vcs.VERDICT_NONE, vcs.Judge(vcs.Subject(subject), true), subject)
  }
}

// Repeats one text a count of times.
func repeat(text string, count int) (repeated string) {
  built := make([]byte, 0, len(text)*count)
  for range count {
    built = append(built, text...)
  }
  return string(built)
}
