// Package vcs enforces the version-control-history doctrine: the rules governing
// every commit on a branch — subject size and conventional shape, and the fixup
// and merge commits that must be squashed or rebased away before landing. The
// caller runs git and hands over the parsed commits, so this package is a pure,
// deterministic function of its input.
package vcs

import (
	"fmt"
	"go/token"
	"regexp"
	"strings"

	"local/james-orcales/lint/internal/diagnostic"
)

// Git's default short-hash width, used to shorten a hash in a diagnostic filename.
const SHORT_HASH_CHARS = 10

// The subject cap: code-review UIs truncate around 72–100 chars and a longer
// subject forces horizontal scroll.
const SUBJECT_CHARS_MAX = 100

// Conventional Commits subject: lowercase type, optional (scope), optional `!`
// breaking-change marker, `: `, non-empty description. Scope contents are not
// whitelisted — package paths and ad-hoc area names both occur in the wild and a
// strict charset would generate more friction than signal.
var conventional_subject_re = regexp.MustCompile(`^[a-z]+(\([^)]+\))?!?: \S`)

// GitHub's synthetic merge subjects. On a shallow checkout the merge's second
// parent is pruned, so `git log --no-merges` lets the merge through as an
// ordinary commit; its capitalized subject would then trip the conventional
// check, so it is exempt.
var github_synthetic_merge_re = regexp.MustCompile(
	`^Merge [0-9a-f]{7,64} into [0-9a-f]{7,64}$|^Merge pull request #\d+ from \S`)

// Git wraps the reverted subject in exactly one outer quoted form. Recognizing the subject shape,
// rather than a capitalized prefix, keeps malformed hand-written approximations under both rules.
var revert_subject_re = regexp.MustCompile(`^Revert ".+"$`)

// Commit is one commit's identity for the history tier: the full hash and the
// subject line of its message.
type Commit struct {
	// Hash is the commit's full object name, used to attribute a diagnostic to
	// the offending commit.
	Hash string
	// Subject is the first line of the commit message — the only part the
	// history rules inspect.
	Subject string
}

// Check_Input is the branch's commits, partitioned into merges and non-merges as
// the caller's git query already screened them.
type Check_Input struct {
	// Merge_Commits are screened for the no-merge-commits rule, subtree merges
	// excepted.
	Merge_Commits []Commit
	// Non_Merge_Commits are screened for the subject-size, conventional-subject,
	// and fixup rules.
	Non_Merge_Commits []Commit
}

// Check enforces the commit-history doctrine over the branch's commits.
func Check(input *Check_Input) (diags []diagnostic.Diagnostic) {
	diags = append(diags, merge_diagnostics(input.Merge_Commits)...)
	diags = append(diags, non_merge_diagnostics(input.Non_Merge_Commits)...)
	return diags
}

// Flags each merge commit on the branch (rebase-instead violation) plus any
// over-length subject. Subtree merges are exempt. Split from Check so each
// commit slice is handled in a function that fits the length cap.
func merge_diagnostics(commits []Commit) (diags []diagnostic.Diagnostic) {
	for _, c := range commits {
		if c.Subject == "" {
			continue
		}
		filename := "<git:" + short_hash(c.Hash) + ">"
		if len(c.Subject) > SUBJECT_CHARS_MAX {
			diags = append(diags, diagnostic.Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "commit-subject-length",
				Want:     fmt.Sprintf("subject ≤ %d chars", SUBJECT_CHARS_MAX),
				Message: fmt.Sprintf(
					"commit subject is %d chars (max %d)",
					len(c.Subject), SUBJECT_CHARS_MAX),
			})
			// The subtree-merge check assumes a bounded subject; the length entry
			// above fully diagnoses an over-limit one, so short-circuit here.
			continue
		}
		if is_subtree_merge_subject(c.Subject) {
			continue
		}
		diags = append(diags, diagnostic.Diagnostic{
			Position: token.Position{Filename: filename},
			Name:     "no-merge-commits",
			Want: "rebase onto main: git fetch origin main && " +
				"git rebase origin/main",
			Message: "merge commit on branch: " + c.Subject,
		})
	}
	return diags
}

// Flags fixup commits (autosquash-instead) and non-conventional subjects on the
// branch, plus any over-length subject. Split from Check for the same length-cap
// reason as the merge variant.
func non_merge_diagnostics(commits []Commit) (diags []diagnostic.Diagnostic) {
	for _, c := range commits {
		if c.Subject == "" {
			continue
		}
		if github_synthetic_merge_re.MatchString(c.Subject) {
			continue
		}
		if revert_subject_re.MatchString(c.Subject) {
			continue
		}
		if len(c.Subject) > SUBJECT_CHARS_MAX {
			filename := "<git:" + short_hash(c.Hash) + ">"
			diags = append(diags, diagnostic.Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "commit-subject-length",
				Want:     fmt.Sprintf("subject ≤ %d chars", SUBJECT_CHARS_MAX),
				Message: fmt.Sprintf(
					"commit subject is %d chars (max %d)",
					len(c.Subject), SUBJECT_CHARS_MAX),
			})
			continue
		}
		if is_fixup_subject(c.Subject) {
			filename := "<git:" + short_hash(c.Hash) + ">"
			diags = append(diags, diagnostic.Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "no-fixup-commits",
				Want:     "autosquash: git rebase -i --autosquash origin/main",
				Message:  "fixup commit on branch: " + c.Subject,
			})
			// A fixup subject is not conventional by construction (e.g.
			// `fixup! feat: foo`); skip the conventional check so it does not
			// double-flag. The autosquash that removes the fixup removes this too.
			continue
		}
		if !conventional_subject_re.MatchString(c.Subject) {
			filename := "<git:" + short_hash(c.Hash) + ">"
			diags = append(diags, diagnostic.Diagnostic{
				Position: token.Position{Filename: filename},
				Name:     "conventional-commits",
				Want: "subject like: type(scope)?!?: description " +
					"(https://www.conventionalcommits.org/)",
				Message: "non-conventional commit subject: " + c.Subject,
			})
		}
	}
	return diags
}

// Matches the default subjects that `git subtree add` and `git subtree pull`
// produce. Both forms are documented in git-subtree(1) and have remained stable
// for years; commits authored by the porcelain match exactly. Hand-authored
// subtree merges with custom messages aren't recognised and trip the
// no-merge-commits rule — intentional, since a custom-worded merge is
// indistinguishable from a regular one.
func is_subtree_merge_subject(subject string) (yes bool) {
	if strings.HasPrefix(subject, "Add '") {
		if strings.Contains(subject, "' from commit '") {
			return true
		}
	}
	if strings.HasPrefix(subject, "Merge commit '") {
		if strings.Contains(subject, "' as '") {
			return true
		}
	}
	return false
}

// Matches commit subjects that should have been autosquashed before merge. Two
// families: the literal fixup!/squash! prefixes that `git commit --fixup`
// produces, and review-comment phrasings that show up when people address
// feedback in a follow-up commit instead of amending. The phrasing checks are
// conjunctive (verb + noun + "review") so isolated mentions of "review" or
// "comment" in unrelated subjects don't get caught.
func is_fixup_subject(subject string) (yes bool) {
	if strings.HasPrefix(subject, "fixup!") {
		return true
	}
	if strings.HasPrefix(subject, "squash!") {
		return true
	}
	s := strings.ToLower(subject)
	has_review := strings.Contains(s, "review")
	has_address := strings.Contains(s, "address")
	has_apply := strings.Contains(s, "apply")
	has_action := has_address || has_apply
	has_comment := strings.Contains(s, "comment")
	has_feedback := strings.Contains(s, "feedback")
	has_nit := strings.Contains(s, "nit")
	has_target := has_comment || has_feedback || has_nit
	if has_review {
		if has_action {
			if has_target {
				return true
			}
		}
	}
	if strings.Contains(s, "cr comment") {
		return true
	}
	if strings.Contains(s, "code review comment") {
		return true
	}
	if strings.Contains(s, "review fix") {
		return true
	}
	if strings.Contains(s, "review nit") {
		return true
	}
	return false
}

// Truncates a git hash to SHORT_HASH_CHARS. Pass-through for an already-short or
// malformed input so a test fixture need not supply a full 40-char hash.
func short_hash(h string) (s string) {
	if len(h) > SHORT_HASH_CHARS {
		return h[:SHORT_HASH_CHARS]
	}
	return h
}
