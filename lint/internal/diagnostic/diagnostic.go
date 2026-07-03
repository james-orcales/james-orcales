// Package diagnostic holds the one type every linter rule emits, so a rule can
// live in its own package without importing the impure lint core back for it.
package diagnostic

import (
	"fmt"
	"go/token"
	"strings"
)

// Diagnostic is one rule violation. Position is the offending source
// location; Name and Want are machine-readable rule identity and
// suggested fix; Message is the human-readable line printed to stdout.
// Tier carries the file-check tier for print-time gating: 1 = tier-1
// (always printed; presence anywhere suppresses tier-2 output), 2 =
// tier-2 (printed only when no tier-1 fires globally). Diagnostics
// from non-file tiers (git, stream, cross-file) leave Tier zero — they
// always print and never gate tier-2.
type Diagnostic struct {
	// Position is the offending source location, printed as the clickable
	// file:line:col prefix.
	Position token.Position
	// Name is the machine-readable rule identity, stable for tooling that
	// groups or suppresses by rule.
	Name string
	// Want is the suggested fix, phrased as the desired post-state.
	Want string
	// Message is the human-readable line printed to stdout.
	Message string
	// Tier carries the file-check tier for print-time gating: 1 always
	// prints and suppresses tier-2 globally when present; 2 prints only
	// when no tier-1 fired; non-file tiers leave it 0.
	Tier int
}

// Within_Scope is true when d is inside the user's scope. An empty scope means
// no filter — every diagnostic passes. A git-tier diagnostic uses a synthetic
// `<git:…>` filename that lives under no scope prefix, so it is admitted for any
// non-empty scope by its leading "<" sentinel.
func Within_Scope(d Diagnostic, scope string) (within bool) {
	if scope == "" {
		return true
	}
	if strings.HasPrefix(d.Position.Filename, "<") {
		return true
	}
	if d.Position.Filename == scope {
		return true
	}
	return strings.HasPrefix(d.Position.Filename, scope+"/")
}

// Reportable returns, in input order, the diagnostics that should print: those
// within scope, with tier-two dropped whenever any in-scope tier-one fired
// (tier-two rules may rely on tier-one contracts, so they are noise until the
// tier-one violations are fixed).
func Reportable(diagnostics []Diagnostic, scope string) (reportable []Diagnostic) {
	has_tier_one := false
	for _, d := range diagnostics {
		if !Within_Scope(d, scope) {
			continue
		}
		if d.Tier == 1 {
			has_tier_one = true
			break
		}
	}
	for _, d := range diagnostics {
		if !Within_Scope(d, scope) {
			continue
		}
		if has_tier_one {
			if d.Tier == 2 {
				continue
			}
		}
		reportable = append(reportable, d)
	}
	return reportable
}

// Format renders d as its clickable `position: message` line, the form printed
// to stdout (without the trailing newline).
func Format(d Diagnostic) (line string) {
	return fmt.Sprintf("%s: %s", d.Position, d.Message)
}
