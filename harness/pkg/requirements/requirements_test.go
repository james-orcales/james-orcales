package requirements_test

import "testing"

import "local/james-orcales/harness/pkg/requirements"
import "local/james-orcales/shared/std/go/ast"
import "local/james-orcales/shared/std/go/token"
import "local/james-orcales/shared/std/testify"
import "local/james-orcales/shared/std/unicode/ucd"
import "local/james-orcales/shared/std/unicode/utf8"

// Test_Coverage_Presence verifies a package without REQUIREMENTS.md is flagged.
func Test_Coverage_Presence(t *testing.T) {
  subject := checker_new()
  output := run(t, subject, contract{Path: "package", Markdown: ABSENT, Test: ABSENT})
  testify.Equal(
    t,
    "REQUIREMENTS.md: The package has no REQUIREMENTS.md. Add REQUIREMENTS.md.\n",
    output,
  )
  // A package holding the test file and no contract: the contract is what the package
  // owes, thus a test file standing alone answers none of that.
  output = run(t, subject, contract{Path: "package", Markdown: ABSENT, Test: CONTRACT_TEST})
  testify.Equal(
    t,
    "REQUIREMENTS.md: The package has no REQUIREMENTS.md. Add REQUIREMENTS.md.\n",
    output,
  )
}

// Test_Coverage_Exemptions verifies package main, third_party, and examples owe no contract.
func Test_Coverage_Exemptions(t *testing.T) {
  subject := checker_new()
  cases := []contract{
    {Path: "package", Impure: true, Markdown: ABSENT, Test: ABSENT},
    {Path: "third_party/package", Markdown: ABSENT, Test: ABSENT},
    {Path: "a/examples", Markdown: ABSENT, Test: ABSENT},
  }
  for _, one := range cases {
    testify.Equal(t, "", run(t, subject, one), one.Path)
  }
  // A package that owes no contract and holds one: the exemption names what a package must
  // start, not what a contract it started may hold, thus the check states its faults.
  testify.Equal(
    t,
    "REQUIREMENTS.md:1: The content is before the first heading. "+
      "Start the file with a heading.\n"+
      `REQUIREMENTS.md:3: The heading level is not "#" or "###". `+
      `Write a "#" heading or a "###" heading.`+
      "\n"+
      "requirements_test.go: The package has no requirements_test.go. "+
      "Add requirements_test.go.\n",
    run(
      t,
      subject,
      contract{
        Path:     "package",
        Impure:   true,
        Markdown: "Intro.\n\n## B\n",
        Test:     ABSENT,
      },
    ),
  )
  // A segment that merely opens with an exempt name is a segment of its own, thus the
  // package under it owes its contract the way any other package does.
  for _, path := range [EXEMPT_LIKE_COUNT]string{"third_partyx/a", "xexamples/a"} {
    testify.Equal(
      t,
      "REQUIREMENTS.md: The package has no REQUIREMENTS.md. "+
        "Add REQUIREMENTS.md.\n",
      run(t, subject, contract{Path: path, Markdown: ABSENT, Test: ABSENT}),
      path,
    )
  }
}

// Test_Format_Preamble verifies content before the first heading is flagged.
func Test_Format_Preamble(t *testing.T) {
  subject := checker_new()
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
  // A file of no bytes, and one of a blank line: neither states a heading and neither
  // states content, thus a package answers its contract with no section at all.
  for _, markdown := range []string{"", "\n"} {
    output = run(
      t,
      subject,
      contract{
        Path:     "package",
        Markdown: markdown,
        Test:     "package a\n",
      },
    )
    testify.Equal(
      t,
      "REQUIREMENTS.md: The file states no heading. Write a heading.\n",
      output,
    )
  }
}

// Test_Format_Heading_Level verifies a heading other than # and ### is flagged.
func Test_Format_Heading_Level(t *testing.T) {
  subject := checker_new()
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
  subject := checker_new()
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
  // A contract that ends on its heading, with no newline after it: nothing follows the
  // heading, thus no blank line does either.
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A",
      Test:     "package a\n\nfunc Test_A() {}\n",
    },
  )
  testify.Equal(
    t,
    `REQUIREMENTS.md:2: There is no blank line after the heading "A". `+
      "Write a blank line.\n"+
      `REQUIREMENTS.md:2: The section "A" has no body line. `+
      "Write a body line.\n",
    output,
  )
  heading_return(t, subject)
  // A contract whose last section carries no body line: the file ends where the next heading
  // would stand, thus the section closes at the end rather than goes unstated.
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\n\nBody.\n\n# B\n",
      Test:     "package a\n\nfunc Test_A() {}\n\nfunc Test_B() {}\n",
    },
  )
  testify.Equal(
    t,
    `REQUIREMENTS.md:6: The section "B" has no body line. Write a body line.`+"\n",
    output,
  )
}

// Test_Format_Heading_Characters verifies a heading word holding a rune that is no letter or
// digit is flagged, and its leaf's test cannot match.
func Test_Format_Heading_Characters(t *testing.T) {
  subject := checker_new()
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
    `REQUIREMENTS.md:2: The heading "A-B" is not spelled with ASCII letters and `+
      "digits. Write a heading of ASCII letters and digits.\n"+
      `requirements_test.go:3: The file needs Test_A-B for the leaf "A-B". `+
      "Declare the tests in leaf order at the top of the file.\n",
    output,
  )
  // A digit is a character a heading word holds, thus a heading spelled with one stands
  // rather than reads as a character that is neither a letter nor a digit.
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A1 B2\n\nBody.\n",
      Test:     "package a\n\nfunc Test_A1_B2() {}\n",
    },
  )
  testify.Equal(t, "", output)
  heading_case(t, subject)
}

// Test_Format_Heading_Uniqueness verifies a repeated heading and a ### without a # are flagged.
func Test_Format_Heading_Uniqueness(t *testing.T) {
  subject := checker_new()
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
  // A leaf name is unique under its own parent, thus the same name under a second parent
  // stands rather than reads as a repeat of the first.
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\n\n### B\n\nBody.\n\n# C\n\n### B\n\nBody.\n",
      Test:     "package a\n\nfunc Test_A_B() {}\n\nfunc Test_C_B() {}\n",
    },
  )
  testify.Equal(t, "", output)
  // A "#" leaf of two words and a "###" leaf under a "#" of one fold to one test name,
  // thus the second is flagged although no two headings share their text.
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\n\n### B\n\nBody.\n\n# A B\n\nBody.\n",
      Test:     "package a\n\nfunc Test_A_B() {}\n",
    },
  )
  testify.Equal(
    t,
    `REQUIREMENTS.md:8: The heading "A B" states the test name an earlier heading `+
      "states. Write a different heading.\n"+
      `requirements_test.go: The file needs Test_A_B for the leaf "A_B". `+
      "Declare the tests in leaf order at the top of the file.\n",
    output,
  )
}

// Test_Format_Section_Not_Empty verifies a leaf without a body line is flagged, a branch intro is
// not, and a leaf's test name is its headings in Ada_Case.
func Test_Format_Section_Not_Empty(t *testing.T) {
  subject := checker_new()
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
  // A "###" section is a leaf of its own, thus one carrying no body line is flagged the
  // way a "#" with no child is, and the "#" above it holds children so it owes none.
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\n\n### B\n\n# C\n\nBody.\n",
      Test:     "package a\n\nfunc Test_A_B() {}\n\nfunc Test_C() {}\n",
    },
  )
  testify.Equal(
    t,
    `REQUIREMENTS.md:4: The section "B" has no body line. Write a body line.`+"\n",
    output,
  )
}

// Test_Format_Section_Size verifies a fourth body line is flagged.
func Test_Format_Section_Size(t *testing.T) {
  subject := checker_new()
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
  // Three lines is the most a section holds, thus a section of exactly three stands
  // rather than reads as one line past the bound.
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\n\n1\n2\n3\n",
      Test:     "package a\n\nfunc Test_A() {}\n",
    },
  )
  testify.Equal(t, "", output)
}

// Test_Format_Section_Contiguity verifies a blank line inside a body is flagged.
func Test_Format_Section_Contiguity(t *testing.T) {
  subject := checker_new()
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

// Test_Format_Conflict_Markers verifies a line that opens on a conflict marker is flagged, and
// one that holds a marker past its first byte is not.
func Test_Format_Conflict_Markers(t *testing.T) {
  subject := checker_new()
  test := "package a\n\nfunc Test_A() {}\n"
  for _, marker := range []string{"<<<<<<<", ">>>>>>>", "%%%%%%%", "+++++++", "\\\\\\\\\\\\\\"} {
    markdown := "\n# A\n\nBody.\n" + marker + " side\n"
    output := run(t, subject, contract{Path: "package", Markdown: markdown, Test: test})
    testify.Equal(
      t,
      "REQUIREMENTS.md:5: The file has a conflict marker. "+
        "Resolve the conflict and remove the marker.\n",
      output,
      marker,
    )
  }
  markdown := "\n# A\n\nBody <<<<<<< side.\n"
  testify.Equal(t, "", run(t, subject, contract{Path: "package", Markdown: markdown, Test: test}))
}

// Test_Tests_Presence verifies a package with REQUIREMENTS.md and no requirements_test.go is
// flagged.
func Test_Tests_Presence(t *testing.T) {
  subject := checker_new()
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
  subject := checker_new()
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
  // A test file of more lines than one index holds places none of its declarations, thus
  // each slot it states names the file alone rather than a line the index never took.
  blanks := make([]byte, 0, token.LINE_COUNT_MAXIMUM+len("package a\n\nfunc Test_B() {}\n"))
  blanks = append(blanks, "package a\n\nfunc Test_B() {}\n"...)
  for range token.LINE_COUNT_MAXIMUM {
    blanks = append(blanks, '\n')
  }
  output = run(
    t,
    subject,
    contract{Path: "package", Markdown: "\n# A\n\nBody.\n", Test: string(blanks)},
  )
  testify.Equal(
    t,
    `requirements_test.go: The file needs Test_A for the leaf "A". `+
      "Declare the tests in leaf order at the top of the file.\n",
    output,
  )
  // A method carries its receiver before its name, thus the name the slot states is the
  // first identifier past that receiver rather than the receiver itself.
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\n\nBody.\n",
      Test:     "package a\n\nfunc (s S) Test_A() {}\n\ntype S struct{}\n",
    },
  )
  testify.Equal(t, "", output)
  tests_quoted(t, subject)
}

// Test_Tests_Refused verifies a test file the parser refuses leaves the order check unrun, since
// a refused file states no declaration a slot could hold.
func Test_Tests_Refused(t *testing.T) {
  subject := checker_new()
  output := run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\n\nBody.\n",
      Test:     "package a\n\nfunc f( {}\n",
    },
  )
  testify.Equal(t, "", output)
  // The same contract with a test file the parser takes: the order check runs, thus the
  // leaf it answers none of is flagged.
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\n\nBody.\n",
      Test:     "package a\n\nvar wrong = 1\n",
    },
  )
  testify.Equal(
    t,
    `requirements_test.go:3: The file needs Test_A for the leaf "A". `+
      "Declare the tests in leaf order at the top of the file.\n",
    output,
  )
}

// Test_Tests_Rewritten verifies a requirements_test.go the caller rewrote goes unchecked, and the
// check states that a rerun checks it.
func Test_Tests_Rewritten(t *testing.T) {
  output := run(
    t,
    checker_new(),
    contract{
      Path:      "package",
      Markdown:  "\n# A\n\nBody.\n",
      Test:      "package a\n\nvar wrong = 1\n",
      Rewritten: true,
    },
  )
  testify.Equal(
    t,
    "requirements_test.go: The file was rewritten by this run. Run the check again for its tests.\n",
    output,
  )
}

// Test_Tests_Declarations verifies Collect reads each leading declaration's name and line out of a
// parse the caller made, and parses nothing itself.
func Test_Tests_Declarations(t *testing.T) {
  subject := checker_new()
  syntax := syntax_new()
  source := token.Source("package a\n\nfunc Test_A() {}\n\nfunc (r R) Test_B() {}\n")
  state := ast.Parse_State{Syntax_Tree: *syntax.Syntax_Tree}
  testify.True(t, bool(ast.Parse(&state, source).OK))
  collected := requirements.Collect(
    syntax.Syntax_Tree,
    syntax.Lines.Starts,
    requirements.Parsed_Source(source),
    subject.Declarations,
  )
  testify.Equal(t, requirements.Declaration_Count(2), collected)
  first, second := subject.Declarations[0], subject.Declarations[1]
  testify.Equal(t, "Test_A", string(source[first.Start:first.End]))
  testify.Equal(t, requirements.Declaration_Line(3), first.Line)
  testify.Equal(t, "Test_B", string(source[second.Start:second.End]))
  testify.Equal(t, requirements.Declaration_Line(5), second.Line)
}

// Test_Resume verifies Next stops for requirements_test.go only once the markdown checks are
// done, and a check without findings ends at STATE_DONE.
func Test_Resume(t *testing.T) {
  subject := checker_new()
  contract_load(subject, contract{Path: "package", Markdown: CONTRACT, Test: CONTRACT_TEST})
  requirements.Begin(subject)
  requirements.Next(subject)
  testify.Equal(t, requirements.STATE_TEST, state_of(subject))
  test_load(subject, contract{Test: CONTRACT_TEST})
  requirements.Next(subject)
  testify.Equal(t, requirements.STATE_DONE, state_of(subject))
}

// Test_Rewrite verifies the rewrite puts a blank line on each side of every heading and takes
// the blank lines out from between two body lines, and leaves alone what it cannot check whole.
func Test_Rewrite(t *testing.T) {
  subject := checker_new()
  cases := [][REWRITE_PAIR_COUNT]string{
    {"# A\nBody.\n\n \nMore.\n# B\nText.", "\n# A\n\nBody.\nMore.\n\n# B\n\nText."},
    {"# A\r\nBody.\r\n\r\nMore.\r\n", "\r\n# A\r\n\r\nBody.\r\nMore.\r\n"},
    {"\n# A", "\n# A\n"},
    {"\n# A\r", "\n# A\r\n"},
  }
  for _, one := range cases {
    written, _ := rewritten(t, subject, one[0])
    testify.Equal(t, one[1], written, one[0])
    // A rewritten contract states none of what the rewrite answers, thus a second one
    // changes nothing.
    written, _ = rewritten(t, subject, one[1])
    testify.Equal(t, "", written, one[1])
  }
  // The findings the rewrite answers go unstated, and the one it leaves names the line the
  // rewritten contract holds it at.
  written, output := rewritten(t, subject, "# A\nOne.\nTwo.\n\nThree.\nFour.\n")
  testify.Equal(t, "\n# A\n\nOne.\nTwo.\nThree.\nFour.\n", written)
  testify.Equal(
    t,
    "REQUIREMENTS.md:7: The section \"A\" has more than three lines. "+
      "Write a maximum of three lines.\n"+
      "requirements_test.go: The package has no requirements_test.go. Add requirements_test.go.\n",
    output,
  )
  // A heading past its bound stops the check there, thus the rewrite leaves the contract as
  // the check will state it.
  long := make([]byte, requirements.HEADING_SIZE_MAXIMUM+1)
  for index := range long {
    long[index] = 'A'
  }
  oversized := "# " + string(long) + "\nBody."
  written, _ = rewritten(t, subject, oversized)
  testify.Equal(t, "", written)
}

// Test_Bounds verifies a file past its storage, past its heading table, or holding a heading past
// 128 bytes is flagged, and its check stops.
func Test_Bounds(t *testing.T) {
  subject := checker_new()
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
  // One heading past the table, thus the table fills on the last one.
  section := "# A\n\nBody.\n\n"
  crowded := make([]byte, 0, len("\n")+(requirements.HEADING_COUNT_MAXIMUM+1)*len(section))
  crowded = append(crowded, '\n')
  for range requirements.HEADING_COUNT_MAXIMUM + 1 {
    crowded = append(crowded, section...)
  }
  output = run(t, subject, contract{Path: "package", Markdown: string(crowded), Test: ABSENT})
  last := len("\n") + requirements.HEADING_COUNT_MAXIMUM*4 + 1
  testify.Equal(
    t,
    "REQUIREMENTS.md:"+
      itoa(last)+
      ": "+
      "The file has more headings than its heading table holds. "+
      "Write fewer headings.\n",
    output,
  )
  // A file that fills its storage is one byte short of its whole self at least.
  full := make([]byte, requirements.MARKDOWN_STORAGE_SIZE)
  for index := range full {
    full[index] = '\n'
  }
  output = run(t, subject, contract{Path: "package", Markdown: string(full), Test: ABSENT})
  testify.Equal(
    t,
    "REQUIREMENTS.md: The file is larger than its storage. Write a shorter file.\n",
    output,
  )
  // One declaration past the table, thus the table fills on the last one and the leaf its
  // leading slot answers still stands.
  declaration, head := "var a = 0\n", "package a\n\nfunc Test_A() {}\n"
  markdown := "\n# A\n\nBody.\n"
  many := make([]byte, 0, len(head)+(requirements.HEADING_COUNT_MAXIMUM+1)*len(declaration))
  many = append(many, head...)
  for range requirements.HEADING_COUNT_MAXIMUM + 1 {
    many = append(many, declaration...)
  }
  output = run(t, subject, contract{Path: "package", Markdown: markdown, Test: string(many)})
  testify.Equal(t, "", output)
  bounds_upper(t)
}

// Test_Allocation verifies a whole check, findings and test comparison included, allocates
// nothing.
func Test_Allocation(t *testing.T) {
  subject := checker_new()
  contract_load(
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\n\n### two words\n\n1\n\n2\n3\n4\n",
      Test:     "package a\n\nconst X = 1\n",
    },
  )
  testify.Zero_Allocation(t, func() { allocation_run(subject) })
  testify.Equal(t, requirements.STATE_DONE, state_of(subject))
  // A rune past ASCII takes the Unicode tables for its class and its upper case.
  contract_load(
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# überblick zwei\n\nBody.\n\n# a·b\n\nBody.\n",
      Test:     "package a\n\nfunc Test_X() {}\n",
    },
  )
  testify.Zero_Allocation(t, func() { allocation_run(subject) })
  testify.Equal(t, requirements.STATE_DONE, state_of(subject))
  contract_load(subject, contract{Path: "package", Markdown: ABSENT, Test: ABSENT})
  testify.Zero_Allocation(t, func() { allocation_run(subject) })
  testify.Equal(t, requirements.STATE_DONE, state_of(subject))
}

func state_of(checker *requirements.Checker) (state requirements.Number) {
  return checker.Numbers.Values[requirements.NUMBER_STATE]
}

// EXEMPT_LIKE_COUNT is how many paths stand named like an exempt segment without being one.
const EXEMPT_LIKE_COUNT = 2

// ABSENT marks a contract without such a file.
const ABSENT = "\x00"

// CONTRACT is the smallest REQUIREMENTS.md that breaks no rule.
const CONTRACT = "\n# Contract\n\nIt holds.\n"

// CONTRACT_TEST mirrors CONTRACT.
const CONTRACT_TEST = "package a\n\nfunc Test_Contract() {}\n"

// REWRITE_PAIR_COUNT is a contract and what the rewrite makes of it.
const REWRITE_PAIR_COUNT = 2

type contract struct {
  Path      string
  Impure    bool
  Markdown  string
  Test      string
  Rewritten bool
}

func syntax_new() (syntax requirements.Syntax) {
  state := &ast.Syntax_Tree{}
  state.Tokens.Values = make(ast.Tokens, ast.TOKEN_COUNT_MAXIMUM)
  state.Nodes.Values = make(ast.Nodes, ast.NODE_COUNT_MAXIMUM)
  state.Parents.Values = make(ast.Parents, ast.DEPTH_MAXIMUM)
  state.Last_Children.Values = make(ast.Last_Children, ast.DEPTH_MAXIMUM)
  state.Earlier_Children.Values = make(ast.Earlier_Children, ast.DEPTH_MAXIMUM)
  state.Errors.Values = make(ast.Errors, ast.ERROR_COUNT_MAXIMUM)
  starts := make(token.Line_Starts, token.LINE_COUNT_MAXIMUM)
  return requirements.Syntax{Syntax_Tree: state, Lines: &token.Line_Index{Starts: starts}}
}

// A contract a host ended on a carriage return alone, and one that is a heading and a return
// and nothing else: a return stands as a byte of its own line rather than half an ending, thus
// each pass stops at the end of the file rather than reads past it.
func heading_return(t *testing.T, subject *requirements.Checker) {
  t.Helper()
  output := run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\r\n# A\r\n\r",
      Test:     "package a\n\nfunc Test_A() {}\n",
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
      Markdown: "# \r",
      Test:     "package a\n\nfunc Test_A() {}\n",
    },
  )
  testify.Equal(
    t,
    `REQUIREMENTS.md:1: The heading "" is not spelled with ASCII letters and `+
      "digits. Write a heading of ASCII letters and digits.\n"+
      `REQUIREMENTS.md:1: There is no blank line before the heading "". `+
      "Write a blank line.\n"+
      `REQUIREMENTS.md:1: There is no blank line after the heading "". `+
      "Write a blank line.\n"+
      `REQUIREMENTS.md:1: The section "" has no body line. `+
      "Write a body line.\n"+
      `requirements_test.go:3: The file needs Test_ for the leaf "". `+
      "Declare the tests in leaf order at the top of the file.\n",
    output,
  )
}

func checker_new() (checker *requirements.Checker) {
  heading_count := requirements.HEADING_COUNT_MAXIMUM
  tree := &ast.Parse_State{}
  tree.Tokens.Values = make(ast.Tokens, ast.TOKEN_COUNT_MAXIMUM)
  tree.Nodes.Values = make(ast.Nodes, ast.NODE_COUNT_MAXIMUM)
  tree.Parents.Values = make(ast.Parents, ast.DEPTH_MAXIMUM)
  tree.Last_Children.Values = make(ast.Last_Children, ast.DEPTH_MAXIMUM)
  tree.Earlier_Children.Values = make(ast.Earlier_Children, ast.DEPTH_MAXIMUM)
  tree.Errors.Values = make(ast.Errors, ast.ERROR_COUNT_MAXIMUM)
  checker = &requirements.Checker{}
  checker.Markdown = make(requirements.Markdown_Bytes, requirements.MARKDOWN_STORAGE_SIZE)
  checker.Rewritten = make(requirements.Rewritten_Bytes, requirements.MARKDOWN_STORAGE_SIZE)
  checker.Test = make(requirements.Test_Bytes, requirements.TEST_STORAGE_SIZE)
  checker.Path = make(requirements.Path_Bytes, requirements.PATH_STORAGE_SIZE)
  checker.Numbers.Values = make(requirements.Numbers, requirements.NUMBER_SLOT_COUNT)
  checker.Flags.Values = make(requirements.Flags, requirements.FLAG_SLOT_COUNT)
  checker.Headings = make(requirements.Headings, heading_count)
  checker.Leaves = make(requirements.Leaves, heading_count)
  checker.Declarations = make(requirements.Declarations, heading_count)
  checker.Name = make(requirements.Name_Bytes, requirements.NAME_SIZE_MAXIMUM)
  checker.Prior = make(requirements.Prior_Bytes, requirements.NAME_SIZE_MAXIMUM)
  checker.Message = make(requirements.Message_Bytes, requirements.MESSAGE_SIZE_MAXIMUM)
  checker.Events = make(requirements.Events, requirements.EVENT_COUNT_MAXIMUM)
  checker.Lines = make(requirements.Rewritten_Lines, requirements.LINE_TABLE_SIZE)
  return checker
}

// Loads one contract as the harness would: flags from the directory, and the markdown bytes that
// fit.
func contract_load(checker *requirements.Checker, one contract) {
  requirements.Load_Path(checker, requirements.Path_Count(copy(checker.Path, one.Path)))
  flags := checker.Flags.Values
  flags[requirements.FLAG_IMPURE] = requirements.Flag(one.Impure)
  flags[requirements.FLAG_REWRITE_WANTED] = false
  flags[requirements.FLAG_HAS_MARKDOWN] = one.Markdown != ABSENT
  flags[requirements.FLAG_HAS_TEST] = one.Test != ABSENT
  requirements.Load(checker, 0)
  if flags[requirements.FLAG_HAS_MARKDOWN] {
    loaded := copy(checker.Markdown, one.Markdown)
    requirements.Load(checker, requirements.Markdown_Count(loaded))
  }
  test_load(checker, one)
}

// Loads requirements_test.go with the declarations of the caller's one parse of it, as the harness
// does: none for a file the parser refused, and none the check places for one a fix rewrote.
func test_load(checker *requirements.Checker, one contract) {
  count := copy(checker.Test, one.Test)
  requirements.Load_Test(checker, requirements.Test_Count(count))
  syntax := syntax_new()
  state := ast.Parse_State{Syntax_Tree: *syntax.Syntax_Tree}
  source := token.Source(checker.Test[:count])
  read, collected := requirements.TEST_REFUSED, requirements.Declaration_Count(0)
  if bool(ast.Parse(&state, source).OK) {
    read = requirements.TEST_PARSED
    collected = requirements.Collect(
      syntax.Syntax_Tree,
      syntax.Lines.Starts,
      requirements.Parsed_Source(source),
      checker.Declarations,
    )
  }
  if one.Rewritten {
    read = requirements.TEST_REWRITTEN
  }
  requirements.Load_Tests(checker, collected, read)
}

// Checks one contract under a fix to its end: the rewrite the check asks the caller to write, or
// none, and each finding rendered as the harness does.
func rewritten(
  t *testing.T,
  checker *requirements.Checker,
  markdown string,
) (written string, output string) {
  t.Helper()
  contract_load(checker, contract{Path: "package", Markdown: markdown, Test: ABSENT})
  checker.Flags.Values[requirements.FLAG_REWRITE_WANTED] = true
  requirements.Begin(checker)
  for range requirements.STEP_COUNT_MAXIMUM {
    requirements.Next(checker)
    numbers := checker.Numbers.Values
    switch numbers[requirements.NUMBER_STATE] {
    case requirements.STATE_DONE:
      return written, output
    case requirements.STATE_REWRITE:
      written = string(checker.Rewritten[:numbers[requirements.NUMBER_REWRITE_COUNT]])
    case requirements.STATE_FINDING:
      output += finding_text(checker)
    }
  }
  t.Fatal("the check never ended")
  return written, output
}

// Checks one contract to its end and renders each finding as the harness does.
func run(t *testing.T, checker *requirements.Checker, one contract) (output string) {
  t.Helper()
  contract_load(checker, one)
  requirements.Begin(checker)
  for range requirements.STEP_COUNT_MAXIMUM {
    requirements.Next(checker)
    switch checker.Numbers.Values[requirements.NUMBER_STATE] {
    case requirements.STATE_DONE:
      return output
    case requirements.STATE_TEST:
      test_load(checker, one)
    case requirements.STATE_FINDING:
      output += finding_text(checker)
    }
  }
  t.Fatal("the check never ended")
  return output
}

func finding_text(checker *requirements.Checker) (text string) {
  numbers := checker.Numbers.Values
  name := requirements.MARKDOWN_NAME
  if requirements.File(numbers[requirements.NUMBER_FILE]) == requirements.FILE_TEST {
    name = requirements.TEST_NAME
  }
  position := ""
  if numbers[requirements.NUMBER_LINE] > 0 {
    position = ":" + itoa(int(numbers[requirements.NUMBER_LINE]))
  }
  message := checker.Message[:numbers[requirements.NUMBER_MESSAGE_COUNT]]
  return name + position + ": " + string(message) + "\n"
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
    if checker.Numbers.Values[requirements.NUMBER_STATE] == requirements.STATE_DONE {
      return
    }
  }
}

// A heading spelled in runes past ASCII, one spelled with a capital inside a word, and one
// spelled as an acronym: the first two build a name this toolchain cannot carry, the last does.
func heading_case(t *testing.T, subject *requirements.Checker) {
  t.Helper()
  output := run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# \u00fcberblick\n\nBody.\n",
      Test:     "package a\n",
    },
  )
  testify.Equal(
    t,
    `REQUIREMENTS.md:2: The heading "überblick" is not spelled with ASCII letters `+
      "and digits. Write a heading of ASCII letters and digits.\n"+
      `requirements_test.go: The file needs Test_Überblick for the leaf `+
      `"Überblick". Declare the tests in leaf order at the top of the file.`+
      "\n",
    output,
  )
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# abC\n\nBody.\n",
      Test:     "package a\n",
    },
  )
  testify.Equal(
    t,
    `REQUIREMENTS.md:2: The heading "abC" carries a capital after a lower case `+
      "letter. Start the word with the capital, or spell the word in "+
      "capitals.\n"+
      `requirements_test.go: The file needs Test_AbC for the leaf "AbC". `+
      "Declare the tests in leaf order at the top of the file.\n",
    output,
  )
  output = run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# JSON\n\nBody.\n",
      Test:     "package a\n\nfunc Test_JSON() {}\n",
    },
  )
  testify.Equal(t, "", output)
}

// A heading a host wrote with a byte below a space builds a name holding that byte, thus the
// message quotes the name the way it quotes the heading rather than hands a terminal a byte that
// steers it.
func tests_quoted(t *testing.T, subject *requirements.Checker) {
  t.Helper()
  output := run(
    t,
    subject,
    contract{
      Path:     "package",
      Markdown: "\n# A\x1bB\n\nBody.\n",
      Test:     "package a\n",
    },
  )
  testify.Equal(
    t,
    `REQUIREMENTS.md:2: The heading "A?B" is not spelled with ASCII letters and `+
      "digits. Write a heading of ASCII letters and digits.\n"+
      `requirements_test.go: The file needs Test_A?B for the leaf "A?B". `+
      "Declare the tests in leaf order at the top of the file.\n",
    output,
  )
}

// Every character the tables map to an upper case widens by one byte at most, and only one past
// ASCII widens at all: the name bound spends one byte per word on that and no more.
func bounds_upper(t *testing.T) {
  t.Helper()
  last := utf8.Character(utf8.RUNE_MAX)
  for character := utf8.Character(utf8.CHARACTER_ONE_MAXIMUM + 1); character < last+1; character++ {
    size := utf8.Character_Size(character)
    if size == utf8.CHARACTER_SIZE_INVALID {
      continue
    }
    upper := utf8.Character(ucd.To_Upper(ucd.Character(character)))
    growth := int(utf8.Character_Size(upper)) - int(size)
    if growth > requirements.UPPER_GROWTH_MAXIMUM {
      t.Fatalf("%U widens by %d bytes in upper case", character, growth)
    }
  }
}
