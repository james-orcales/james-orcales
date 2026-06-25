package diagnostic_test

import (
	"go/token"
	"testing"

	"github.com/james-orcales/james-orcales/lint/internal/diagnostic"
)

// Test_Diagnostic_Position verifies Position carries the offending source
// location that prints as the file:line:col prefix.
func Test_Diagnostic_Position(t *testing.T) {
	t.Parallel()
	d := diagnostic.Diagnostic{Position: token.Position{Filename: "a.go", Line: 3, Column: 2}}
	if d.Position.Filename != "a.go" {
		t.Fatal("Position must carry the filename")
	}
	if d.Position.Line != 3 {
		t.Fatal("Position must carry the line")
	}
}

// Test_Diagnostic_Name verifies Name carries the machine-readable rule identity.
func Test_Diagnostic_Name(t *testing.T) {
	t.Parallel()
	d := diagnostic.Diagnostic{Name: "specification"}
	if d.Name != "specification" {
		t.Fatal("Name must carry the rule identity")
	}
}

// Test_Diagnostic_Want verifies Want carries the suggested fix as a post-state.
func Test_Diagnostic_Want(t *testing.T) {
	t.Parallel()
	d := diagnostic.Diagnostic{Want: "add SPECIFICATION.md"}
	if d.Want != "add SPECIFICATION.md" {
		t.Fatal("Want must carry the suggested fix")
	}
}

// Test_Diagnostic_Message verifies Message carries the human-readable line.
func Test_Diagnostic_Message(t *testing.T) {
	t.Parallel()
	d := diagnostic.Diagnostic{Message: "a.go:1 is wrong"}
	if d.Message != "a.go:1 is wrong" {
		t.Fatal("Message must carry the printed line")
	}
}

// Test_Diagnostic_Tier verifies Tier defaults to the non-file zero and records
// the tier-one and tier-two gating values.
func Test_Diagnostic_Tier(t *testing.T) {
	t.Parallel()
	var zero diagnostic.Diagnostic
	if zero.Tier != 0 {
		t.Fatal("a non-file diagnostic must leave Tier zero")
	}
	one := diagnostic.Diagnostic{Tier: 1}
	if one.Tier != 1 {
		t.Fatal("Tier must record the tier-one gating value")
	}
}

// Test_Reporting_Within_Scope verifies the scope filter admits a file under the
// prefix and the synthetic git sentinel, and rejects a file outside.
func Test_Reporting_Within_Scope(t *testing.T) {
	t.Parallel()
	inside := diagnostic.Diagnostic{Position: token.Position{Filename: "pkg/a.go"}}
	if !diagnostic.Within_Scope(inside, "pkg") {
		t.Fatal("a file under the scope prefix must be in scope")
	}
	outside := diagnostic.Diagnostic{Position: token.Position{Filename: "other/a.go"}}
	if diagnostic.Within_Scope(outside, "pkg") {
		t.Fatal("a file outside the scope prefix must be out of scope")
	}
	git := diagnostic.Diagnostic{Position: token.Position{Filename: "<git:log>"}}
	if !diagnostic.Within_Scope(git, "pkg") {
		t.Fatal("a synthetic git diagnostic must always be in scope")
	}
}

// Test_Reporting_Reportable verifies an in-scope tier-one suppresses tier-two and
// out-of-scope diagnostics are dropped.
func Test_Reporting_Reportable(t *testing.T) {
	t.Parallel()
	one := diagnostic.Diagnostic{Position: token.Position{Filename: "pkg/a.go"}, Tier: 1}
	two := diagnostic.Diagnostic{Position: token.Position{Filename: "pkg/b.go"}, Tier: 2}
	away := diagnostic.Diagnostic{Position: token.Position{Filename: "other/c.go"}, Tier: 1}
	got := diagnostic.Reportable([]diagnostic.Diagnostic{one, two, away}, "pkg")
	if len(got) != 1 {
		t.Fatalf("tier-one must suppress tier-two and drop out-of-scope; got %d", len(got))
	}
	if got[0].Tier != 1 {
		t.Fatal("the surviving diagnostic must be the tier-one")
	}
}

// Test_Reporting_Format verifies a diagnostic renders as its clickable
// position and message line.
func Test_Reporting_Format(t *testing.T) {
	t.Parallel()
	d := diagnostic.Diagnostic{
		Position: token.Position{Filename: "a.go", Line: 3, Column: 2},
		Message:  "bad thing",
	}
	if diagnostic.Format(d) != "a.go:3:2: bad thing" {
		t.Fatalf("unexpected line: %q", diagnostic.Format(d))
	}
}
