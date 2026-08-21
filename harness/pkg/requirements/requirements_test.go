package requirements_test

import "testing"

import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/shared/filepath"
import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/testify"

// Test_Coverage_Presence verifies a package without REQUIREMENTS.md is flagged.
func Test_Coverage_Presence(t *testing.T) {
	subject := checker_default()
	output := run(t, subject, contract{Path: "package", Markdown: ABSENT, Test: ABSENT})
	testify.Equal(
		t,
		"REQUIREMENTS.md: The package has no REQUIREMENTS.md. Add REQUIREMENTS.md.\n",
		output,
	)
}

// Test_Coverage_Exemptions verifies package main, third_party, and examples owe no contract.
func Test_Coverage_Exemptions(t *testing.T) {
	subject := checker_default()
	cases := []contract{
		{Path: "package", Impure: true, Markdown: ABSENT, Test: ABSENT},
		{Path: "third_party/package", Markdown: ABSENT, Test: ABSENT},
		{Path: "a/examples", Markdown: ABSENT, Test: ABSENT},
	}
	for _, one := range cases {
		testify.Equal(t, "", run(t, subject, one), one.Path)
	}
}

// Test_Format_Preamble verifies content before the first heading is flagged.
func Test_Format_Preamble(t *testing.T) {
	subject := checker_default()
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "Intro.\n\n# A\n\nBody.\n",
			Test:     "package a\n\nfunc Test_A() {}\n",
		},
	)
	testify.Equal(
		t,
		"REQUIREMENTS.md:1: The content is before the first heading. "+
			"Start the file with a heading.\n",
		output,
	)
}

// Test_Format_Heading_Level verifies a heading other than # and ### is flagged.
func Test_Format_Heading_Level(t *testing.T) {
	subject := checker_default()
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# A\n\nBody.\n\n## B\n\nBody.\n",
			Test:     "package a\n\nfunc Test_A() {}\n",
		},
	)
	testify.Equal(
		t,
		`REQUIREMENTS.md:6: The heading level is not "#" or "###". `+
			`Write a "#" heading or a "###" heading.`+
			"\n",
		output,
	)
}

// Test_Format_Heading_Blank_Lines verifies a heading without a blank line either side is flagged.
func Test_Format_Heading_Blank_Lines(t *testing.T) {
	subject := checker_default()
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "# A\nBody.\n\n# B\n\nBody.\n",
			Test:     "package a\n\nfunc Test_A() {}\n\nfunc Test_B() {}\n",
		},
	)
	testify.Equal(
		t,
		`REQUIREMENTS.md:1: There is no blank line before the heading "A". Write a blank line.`+
			"\n"+
			`REQUIREMENTS.md:1: There is no blank line after the heading "A". Write a blank line.`+
			"\n",
		output,
	)
}

// Test_Format_Heading_Characters verifies a heading word holding a rune that is no letter or
// digit is flagged, and its leaf's test cannot match.
func Test_Format_Heading_Characters(t *testing.T) {
	subject := checker_default()
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# A-B\n\nBody.\n",
			Test:     "package a\n\nfunc Test_A() {}\n",
		},
	)
	testify.Equal(
		t,
		`REQUIREMENTS.md:2: The heading "A-B" has a character that is not a letter or a digit. `+
			"Write only letters and digits.\n"+
			`requirements_test.go:3: The file needs Test_A-B for the leaf "A-B". `+
			"Declare the tests in leaf order at the top of the file.\n",
		output,
	)
}

// Test_Format_Heading_Uniqueness verifies a repeated heading and a ### without a # are flagged.
func Test_Format_Heading_Uniqueness(t *testing.T) {
	subject := checker_default()
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# A\n\n### B\n\nBody.\n\n### B\n\nBody.\n\n# A\n\nBody.\n",
			Test: "package a\n\nfunc Test_A_B() {}\n\n" +
				"func Test_A_B() {}\n\nfunc Test_A() {}\n",
		},
	)
	testify.Equal(
		t,
		`REQUIREMENTS.md:8: The heading "B" is not unique. Write a different heading.`+
			"\n"+
			`REQUIREMENTS.md:12: The heading "A" is not unique. Write a different heading.`+
			"\n",
		output,
	)
	output = run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n### B\n\nBody.\n\n# A\n\nBody.\n",
			Test:     "package a\n\nfunc Test_A() {}\n",
		},
	)
	testify.Equal(
		t,
		`REQUIREMENTS.md:2: The "###" heading "B" has no parent "#" heading. `+
			`Put the heading below a "#" heading.`+
			"\n",
		output,
	)
}

// Test_Format_Section_Not_Empty verifies a leaf without a body line is flagged, a branch intro is
// not, and a leaf's test name is its headings in Ada_Case.
func Test_Format_Section_Not_Empty(t *testing.T) {
	subject := checker_default()
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# A\n\n# B\n\nBody.\n",
			Test:     "package a\n\nfunc Test_A() {}\n\nfunc Test_B() {}\n",
		},
	)
	testify.Equal(
		t,
		`REQUIREMENTS.md:2: The section "A" has no body line. Write a body line.`+"\n",
		output,
	)
	output = run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# a\n\n### two words\n\nBody.\n",
			Test:     "package a\n\nfunc Test_A_Two_Words() {}\n",
		},
	)
	testify.Equal(t, "", output)
}

// Test_Format_Section_Size verifies a fourth body line is flagged.
func Test_Format_Section_Size(t *testing.T) {
	subject := checker_default()
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# A\n\n1\n2\n3\n4\n",
			Test:     "package a\n\nfunc Test_A() {}\n",
		},
	)
	testify.Equal(
		t,
		`REQUIREMENTS.md:7: The section "A" has more than three lines. `+
			"Write a maximum of three lines.\n",
		output,
	)
}

// Test_Format_Section_Contiguity verifies a blank line inside a body is flagged.
func Test_Format_Section_Contiguity(t *testing.T) {
	subject := checker_default()
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# A\n\n1\n\n2\n",
			Test:     "package a\n\nfunc Test_A() {}\n",
		},
	)
	testify.Equal(
		t,
		`REQUIREMENTS.md:6: The section "A" has a blank line between two body lines. `+
			"Remove the blank line.\n",
		output,
	)
}

// Test_Tests_Presence verifies a package with REQUIREMENTS.md and no requirements_test.go is
// flagged.
func Test_Tests_Presence(t *testing.T) {
	subject := checker_default()
	output := run(t, subject, contract{Path: "package", Markdown: CONTRACT, Test: ABSENT})
	testify.Equal(
		t,
		"requirements_test.go: The package has no requirements_test.go. "+
			"Add requirements_test.go.\n",
		output,
	)
}

// Test_Tests_Order verifies each leading declaration slot that is not its leaf's test is flagged
// at the line it holds, and a slot past the declarations at no line.
func Test_Tests_Order(t *testing.T) {
	subject := checker_default()
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# A\n\n### B\n\nBody.\n\n# C\n\nBody.\n\n# D\n\nBody.\n",
			Test:     "package a\n\nconst X = 1\n\nfunc Test_A_B() {}\n",
		},
	)
	testify.Equal(
		t,
		`requirements_test.go:3: The file needs Test_A_B for the leaf "A_B". `+
			"Declare the tests in leaf order at the top of the file.\n"+
			`requirements_test.go:5: The file needs Test_C for the leaf "C". `+
			"Declare the tests in leaf order at the top of the file.\n"+
			`requirements_test.go: The file needs Test_D for the leaf "D". `+
			"Declare the tests in leaf order at the top of the file.\n",
		output,
	)
}

// Test_Resume verifies Next stops for requirements_test.go only once the markdown checks are
// done, and a check without findings ends at STATE_DONE.
func Test_Resume(t *testing.T) {
	subject := checker_default()
	contract_load(subject, contract{Path: "package", Markdown: CONTRACT, Test: CONTRACT_TEST})
	requirements.Begin(subject)
	requirements.Next(subject)
	testify.Equal(t, requirements.STATE_TEST, subject.State)
	subject.Test_Source = token.Source(CONTRACT_TEST)
	requirements.Next(subject)
	testify.Equal(t, requirements.STATE_DONE, subject.State)
}

// Test_Bounds verifies a file past its storage, past its heading table, or holding a heading past
// 128 bytes is flagged, and its check stops.
func Test_Bounds(t *testing.T) {
	subject := checker_default()
	long := make([]byte, requirements.HEADING_SIZE_MAXIMUM+1)
	for index := range long {
		long[index] = 'A'
	}
	output := run(
		t,
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# " + string(long) + "\n\nBody.\n",
			Test:     ABSENT,
		},
	)
	testify.Equal(
		t,
		"REQUIREMENTS.md:2: The heading is longer than 128 bytes. "+
			"Write a shorter heading.\n",
		output,
	)
	small := checker_new(1, requirements.MARKDOWN_STORAGE_SIZE)
	output = run(
		t,
		small,
		contract{
			Path:     "package",
			Markdown: "\n# A\n\nBody.\n\n# B\n\nBody.\n",
			Test:     ABSENT,
		},
	)
	testify.Equal(
		t,
		"REQUIREMENTS.md:6: The file has more headings than its heading table holds. "+
			"Write fewer headings.\n",
		output,
	)
	small = checker_new(requirements.HEADING_COUNT_MAXIMUM, len(CONTRACT))
	output = run(t, small, contract{Path: "package", Markdown: CONTRACT, Test: ABSENT})
	testify.Equal(
		t,
		"REQUIREMENTS.md: The file is larger than its storage. Write a shorter file.\n",
		output,
	)
}

// Test_Allocation verifies a whole check, findings and test comparison included, allocates
// nothing.
func Test_Allocation(t *testing.T) {
	subject := checker_default()
	contract_load(
		subject,
		contract{
			Path:     "package",
			Markdown: "\n# A\n\n### two words\n\n1\n\n2\n3\n4\n",
			Test:     "package a\n\nconst X = 1\n",
		},
	)
	testify.Zero_Allocation(t, func() { allocation_run(subject) })
	testify.Equal(t, requirements.STATE_DONE, subject.State)
}

// ABSENT marks a contract without such a file.
const ABSENT = "\x00"

// CONTRACT is the smallest REQUIREMENTS.md that breaks no rule.
const CONTRACT = "\n# Contract\n\nIt holds.\n"

// CONTRACT_TEST mirrors CONTRACT.
const CONTRACT_TEST = "package a\n\nfunc Test_Contract() {}\n"

type contract struct {
	Path     string
	Impure   bool
	Markdown string
	Test     string
}

func checker_default() (checker *requirements.Checker) {
	return checker_new(requirements.HEADING_COUNT_MAXIMUM, requirements.MARKDOWN_STORAGE_SIZE)
}

func checker_new(heading_count int, markdown_size int) (checker *requirements.Checker) {
	tree := &ast.Parse_State{}
	tree.Tokens.Values = make(ast.Tokens, ast.TOKEN_COUNT_MAXIMUM)
	tree.Nodes.Values = make(ast.Nodes, ast.NODE_COUNT_MAXIMUM)
	tree.Parents.Values = make(ast.Parents, ast.DEPTH_MAXIMUM)
	tree.Last_Children.Values = make(ast.Last_Children, ast.DEPTH_MAXIMUM)
	tree.Earlier_Children.Values = make(ast.Earlier_Children, ast.DEPTH_MAXIMUM)
	checker = &requirements.Checker{}
	checker.Markdown = make(requirements.Markdown_Bytes, markdown_size)
	checker.Parse_State = tree
	checker.Lines = &token.Line_Index{Starts: make(token.Line_Starts, token.LINE_COUNT_MAXIMUM)}
	checker.Headings = make(requirements.Headings, heading_count)
	checker.Leaves = make(requirements.Leaves, heading_count)
	checker.Declarations = make(requirements.Declarations, heading_count)
	checker.Name_Storage = make(requirements.Text_Storage, 0, requirements.NAME_SIZE_MAXIMUM)
	checker.Message_Storage = make(
		requirements.Text_Storage,
		0,
		requirements.MESSAGE_SIZE_MAXIMUM,
	)
	return checker
}

// Loads one contract as the harness would: flags from the directory, and the markdown bytes that
// fit.
func contract_load(checker *requirements.Checker, one contract) {
	checker.Path = filepath.Slice(one.Path)
	checker.Impure = requirements.Flag(one.Impure)
	checker.Has_Markdown = one.Markdown != ABSENT
	checker.Has_Test = one.Test != ABSENT
	checker.Markdown_Count = 0
	if checker.Has_Markdown {
		count := copy(checker.Markdown, one.Markdown)
		checker.Markdown_Count = requirements.Index(count)
	}
	checker.Test_Source = token.Source(one.Test)
}

// Checks one contract to its end and renders each finding as the harness does.
func run(t *testing.T, checker *requirements.Checker, one contract) (output string) {
	t.Helper()
	contract_load(checker, one)
	requirements.Begin(checker)
	for range requirements.STEP_COUNT_MAXIMUM {
		requirements.Next(checker)
		switch checker.State {
		case requirements.STATE_DONE:
			return output
		case requirements.STATE_TEST:
			checker.Test_Source = token.Source(one.Test)
		case requirements.STATE_FINDING:
			output += finding_text(checker)
		}
	}
	t.Fatal("the check never ended")
	return output
}

func finding_text(checker *requirements.Checker) (text string) {
	name := requirements.MARKDOWN_NAME
	if checker.File == requirements.FILE_TEST {
		name = requirements.TEST_NAME
	}
	position := ""
	if checker.Line > 0 {
		position = ":" + itoa(int(checker.Line))
	}
	return name + position + ": " + string(checker.Message) + "\n"
}

func itoa(value int) (text string) {
	text = string(rune('0' + value%10))
	for value /= 10; value > 0; value /= 10 {
		text = string(rune('0'+value%10)) + text
	}
	return text
}

func allocation_run(checker *requirements.Checker) {
	requirements.Begin(checker)
	for range requirements.STEP_COUNT_MAXIMUM {
		requirements.Next(checker)
		if checker.State == requirements.STATE_DONE {
			return
		}
	}
}
