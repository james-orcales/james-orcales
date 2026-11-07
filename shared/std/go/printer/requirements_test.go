package printer_test

import "testing"

import "local/james-orcales/shared/std/go/ast"
import "local/james-orcales/shared/std/go/printer"
import "local/james-orcales/shared/std/go/token"
import "local/james-orcales/shared/std/testify"

// Test_Printer binds the Printer specification leaf before fixture declarations.
func Test_Printer(t *testing.T) {
  test_printer(t)
}

// Test_Layout binds the Layout specification leaf before fixture declarations.
func Test_Layout(t *testing.T) {
  test_layout(t)
}

// Test_Elision binds the Elision specification leaf before fixture declarations.
func Test_Elision(t *testing.T) {
  test_elision(t)
}

// Test_Names binds the Names specification leaf before fixture declarations.
func Test_Names(t *testing.T) {
  test_elision_imports(t)
}

// Test_Spread binds the Spread specification leaf before fixture declarations.
func Test_Spread(t *testing.T) {
  test_spread(t)
}

// Test_Bytes binds the Bytes specification leaf before fixture declarations.
func Test_Bytes(t *testing.T) {
  test_bytes(t)
}

// Test_Width binds the Width specification leaf before fixture declarations.
func Test_Width(t *testing.T) {
  test_width(t)
}

// Test_Order binds the Order specification leaf before fixture declarations.
func Test_Order(t *testing.T) {
  test_order(t)
}

// Test_Entry binds the Entry specification leaf before fixture declarations.
func Test_Entry(t *testing.T) {
  test_entry(t)
}

// Test_Returns binds the Returns specification leaf before fixture declarations.
func Test_Returns(t *testing.T) {
  test_returns(t)
}

// Test_Spacing binds the Spacing specification leaf before fixture declarations.
func Test_Spacing(t *testing.T) {
  test_spacing(t)
}

// Test_Alignment binds the Alignment specification leaf before fixture declarations.
func Test_Alignment(t *testing.T) {
  test_alignment(t)
}

// Test_Literals binds the Literals specification leaf before fixture declarations.
func Test_Literals(t *testing.T) {
  test_literals(t)
}

// Test_Trivia binds the Trivia specification leaf before fixture declarations.
func Test_Trivia(t *testing.T) {
  test_trivia(t)
}

// Test_Crowded_Lines binds the Crowded Lines specification leaf before fixture declarations.
func Test_Crowded_Lines(t *testing.T) {
  test_crowded_lines(t)
}

// Test_Lines binds the Lines specification leaf before fixture declarations.
func Test_Lines(t *testing.T) {
  test_lines(t)
}

// Test_Comparison binds the Comparison specification leaf before fixture declarations.
func Test_Comparison(t *testing.T) {
  test_comparison(t)
}

// Test_Refusals binds the Refusals specification leaf before fixture declarations.
func Test_Refusals(t *testing.T) {
  test_refusals(t)
}

// Test_Bounds binds the Bounds specification leaf before fixture declarations.
func Test_Bounds(t *testing.T) {
  test_bounds(t)
}

// Test_Allocation binds the Allocation specification leaf before fixture declarations.
func Test_Allocation(t *testing.T) {
  test_allocation(t)
}

type allocation_fixture struct {
  Count printer.Form_Count
  Ok    printer.Boolean
}

// Caller allocation keeps both runtime paths free of hidden storage growth.
func printer_state() (state *printer.Printer) {
  return &printer.Printer{
    Nodes: printer.Node_Stack_Storage{
      Values: make(printer.Node_Stack, printer.DEPTH_MAXIMUM),
    },
    Pairs: printer.Pair_Storage{
      Functions: make(printer.Pair_Functions, printer.PAIR_COUNT_MAXIMUM),
      Heads:     make(printer.Pair_Heads, printer.PAIR_COUNT_MAXIMUM),
    },
    Aliases: printer.Import_Alias_Storage{
      Values: make(printer.Import_Aliases, printer.IMPORT_COUNT_MAXIMUM),
    },
    Paths: printer.Import_Path_Storage{
      Values: make(printer.Import_Paths, printer.IMPORT_COUNT_MAXIMUM),
    },
    Comments: printer.Comment_Position_Storage{
      Values: make(
        printer.Comment_Positions, printer.COMMENT_COUNT_MAXIMUM,
      ),
    },
    Source: &printer.Source_Storage{
      Values: make(printer.Source_Bytes, token.SOURCE_SIZE_MAXIMUM),
    },
    Form: &printer.Form_Storage{
      Values: make(printer.Form_Bytes, printer.FORM_SIZE_MAXIMUM),
    },
    Counts: printer.Count_Storage{
      Values: make(printer.Counts, printer.COUNT_SLOT_COUNT),
    },
    Flags: printer.Flag_Storage{
      Values: make(printer.Flags, printer.FLAG_COUNT),
    },
  }
}

// Caller allocation makes parser bounds part of every printer fixture.
func parse_state() (state *ast.Parse_State) {
  tree := ast.Syntax_Tree{
    Tokens: ast.Token_Storage{
      Values: make(ast.Tokens, ast.TOKEN_COUNT_MAXIMUM),
    },
    Nodes: ast.Node_Storage{
      Values: make(ast.Nodes, ast.NODE_COUNT_MAXIMUM),
    },
    Parents: ast.Parent_Storage{
      Values: make(ast.Parents, ast.DEPTH_MAXIMUM),
    },
    Last_Children: ast.Last_Child_Storage{
      Values: make(ast.Last_Children, ast.DEPTH_MAXIMUM),
    },
    Earlier_Children: ast.Earlier_Child_Storage{
      Values: make(ast.Earlier_Children, ast.DEPTH_MAXIMUM),
    },
    Errors: ast.Error_Storage{
      Values: make(ast.Errors, ast.ERROR_COUNT_MAXIMUM),
    },
  }
  return &ast.Parse_State{Syntax_Tree: tree}
}

// Prints one source and hands back what the print wrote, which is how every case states the form
// it expects.
func printed(t *testing.T, source string) (form string) {
  t.Helper()
  return print_form(t, source, true)
}

// TEST_STABLE_SIZE_MAXIMUM is the widest form the corpus prints a second time, to read that the
// form the print wrote prints as itself.
const TEST_STABLE_SIZE_MAXIMUM = 4096

// Prints one source and hands back the form, reads that form back through the parse, and where
// the caller asks for it prints the form again and reads that it stands. A file that spends every
// name the storage of dropped names admits writes a name the print of the form it wrote drops,
// thus a test of that storage asks for no print of the form.
// Counts the notes one parse holds, which the form the print writes holds as well. A note the
// author wrote states something of the source, thus a print that dropped one would write a form
// that states less than the source it stands for.
func notes_count(tree ast.Parse_State_Handle) (count int) {
  for index := range ast.TOKEN_COUNT_MAXIMUM {
    one := ast.Syntax_Token_At(&tree.Syntax_Tree, ast.Token_Index(index))
    if one.Kind == token.KIND_END_OF_FILE {
      return count
    }
    if one.Kind == token.KIND_COMMENT {
      count = count + 1
    }
  }
  return count
}

// Counts the declarations one file states, counting neither the comments nor the empty lines the
// author wrote between them. The print orders the declarations of a file and writes each one
// once, thus the count it hands back reads the same for the source and for the form.
func declarations_count(tree ast.Parse_State_Handle) (count int) {
  held := &tree.Syntax_Tree
  file := ast.Syntax_Node_At(held, ast.Index(1))
  child := ast.Index(file.First_Child)
  for range ast.NODE_COUNT_MAXIMUM {
    if child == ast.INDEX_ABSENT {
      return count
    }
    node := ast.Syntax_Node_At(held, child)
    child = ast.Index(node.Next)
    if node.Kind == ast.NODE_COMMENT {
      continue
    }
    if node.Kind == ast.NODE_BLANK {
      continue
    }
    count = count + 1
  }
  return count
}

// Spells the words of every comment one tree holds, dropping the marks each comment opens and
// closes with and every space between its words. A wrap states the words of one comment on lines
// of its own, thus the words alone state what the print kept of the note the author wrote.
func spelled_notes(tree ast.Parse_State_Handle, text string) (words string) {
  source := token.Source(text)
  held_words := make([]byte, 0, len(text))
  for index := range ast.TOKEN_COUNT_MAXIMUM {
    one := ast.Syntax_Token_At(&tree.Syntax_Tree, ast.Token_Index(index))
    if one.Kind == token.KIND_END_OF_FILE {
      return string(held_words)
    }
    if one.Kind != token.KIND_COMMENT {
      continue
    }
    for _, held := range token.Text(source, one) {
      if spelled_space(held) {
        continue
      }
      held_words = append(held_words, held)
    }
  }
  return string(held_words)
}

// Reports whether one byte of a comment states a mark it opens with or a space between its words.
func spelled_space(held byte) (yes bool) {
  if held == 0x20 {
    return true
  }
  if held == 0x09 {
    return true
  }
  if held == 0x0a {
    return true
  }
  if held == 0x0d {
    return true
  }
  if held == 0x2f {
    return true
  }
  return held == 0x2a
}

// Reports whether one run of words spells every letter another one spells, counting each letter
// as many times as it stands. The print states the declarations of a file in the order the rules
// name rather than the order the author wrote them, and a note stands with the declaration it
// opens, thus the read counts the letters the notes spell and never the order they stand in.
func holds_words(held string, words string) (yes bool) {
  counted := [BYTE_COUNT]int{}
  for place := range len(held) {
    counted[held[place]] = counted[held[place]] + 1
  }
  for place := range len(words) {
    counted[words[place]] = counted[words[place]] - 1
    if counted[words[place]] < 0 {
      return false
    }
  }
  return true
}

// BYTE_COUNT is how many values one byte states.
const BYTE_COUNT = 256

// Layout has no trailing whitespace or repeated empty lines. String values retain both.
func holds_bytes(t *testing.T, form string, tree ast.Parse_State_Handle) {
  t.Helper()
  if len(form) == 0 {
    return
  }
  testify.Equal(t, uint8('\n'), form[len(form)-1], "the form closes on one line feed")
  if !(len(form) < 2) {
    testify.True(t, form[len(form)-2] != '\n',
      "the form closes on no empty line")
  }
  literal := make([]bool, len(form))
  for index := range ast.TOKEN_COUNT_MAXIMUM {
    one := ast.Syntax_Token_At(&tree.Syntax_Tree, ast.Token_Index(index))
    if one.Kind == token.KIND_END_OF_FILE {
      break
    }
    if one.Kind == token.KIND_STRING {
      for offset := range len(token.Text(token.Source(form), one)) {
        literal[int(one.Offset)+offset] = true
      }
    }
  }
  held := true
  for index := 1; index < len(form); index++ {
    if literal[index] {
      continue
    }
    if form[index] != '\n' {
      continue
    }
    if form[index-1] == ' ' {
      held = false
    }
    if form[index-1] == '\t' {
      held = false
    }
    if index < 2 {
      continue
    }
    if form[index-1] == '\n' {
      if form[index-2] == '\n' {
        held = false
      }
    }
  }
  testify.True(t, held, "every line of the form closes on a byte a reader reads")
}

func print_form(t *testing.T, source string, stable bool) (form string) {
  t.Helper()
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  ast.Parse(tree, token.Source(source))
  result := printer.Print(subject, storage[:], tree, token.Source(source))
  testify.True(t, bool(result.OK), "the print holds the whole form")
  // A form the print wrote for a tree the parse holds states a file of its own, thus the
  // parse reads it back as it stands.
  form = string(storage[:result.Count])
  // The tree of the source states its failure and the notes it holds before the parse of
  // the form stands in it, thus one tree reads both and the test spends one of them.
  failure, notes := tree.Failure, notes_count(tree)
  held_notes := spelled_notes(tree, source)
  held_declarations := declarations_count(tree)
  again := tree
  ast.Parse(again, token.Source(form))
  testify.Equal(
    t,
    failure,
    again.Failure,
    "the form the print wrote parses as the source it stands for",
  )
  // A tree the parse refused holds forms the canonical form states nothing for, thus only
  // the form of a tree the parse held answers for the bytes its lines close on.
  if failure == 0 {
    holds_bytes(t, form, again)
    // A note the print wraps stands as the run of notes its lines state, thus the
    // form holds the notes the source holds and the lines a wrap opens.
    testify.True(t, !(notes_count(again) < notes),
      "the form holds every note the source holds")
    // A note the print wraps states its words on lines of its own, and a note it
    // holds states them as they stand, thus the words of every note the source
    // holds stand in the notes the form holds, in the order the author wrote them.
    testify.True(
      t,
      holds_words(spelled_notes(again, form), held_notes),
      "the form spells the words of every note the source spells",
    )
    // The print states the declarations of one file in the order the rules name
    // and writes each of them once, thus the form holds the count the source holds.
    testify.Equal(
      t,
      held_declarations,
      declarations_count(again),
      "the form states every declaration the source states, once",
    )
  }
  // A source that stands as its own form states the print of that form already, thus only
  // a source the print changed asks for a print of the form it wrote.
  if !stable {
    return form
  }
  if form == source {
    return form
  }
  // A fixture of thousands of bytes spends a print of its own and states nothing of the
  // stability of the form that the forms a reader writes do not state.
  if len(form) > TEST_STABLE_SIZE_MAXIMUM {
    return form
  }
  // The canonical form of one file is the form the print writes for it, thus a print of
  // that form writes it back byte for byte.
  second := printer.Print(subject, storage[:], again, token.Source(form))
  testify.True(t, bool(second.OK), "the print holds the form it wrote")
  testify.Equal(
    t,
    form,
    string(storage[:second.Count]),
    "the form the print wrote prints as itself",
  )
  return form
}

// Prints through a printer that holds no source or form storage of its own, the way a caller that
// owns one source buffer and one form buffer hands them over.
func lent_print(t *testing.T) {
  subject := printer_state()
  subject.Source = nil
  subject.Form = nil
  tree := parse_state()
  text := "package one\n\nvar a,b = 1,2\n"
  buffer := make([]byte, token.SOURCE_SIZE_MAXIMUM+1)
  source := token.Source(buffer[:copy(buffer, text)])
  storage := make(printer.Form, printer.FORM_SIZE_MAXIMUM)
  ast.Parse(tree, source)
  result := printer.Print(subject, storage, tree, source)
  testify.Equal(
    t,
    "package one\n\nvar a, b = 1, 2\n",
    string(storage[:result.Count]),
    "a printer that holds no storage of its own prints into the buffers it was handed",
  )
  again := "package one\n\nvar c = 3\n"
  source = token.Source(buffer[:copy(buffer, again)])
  ast.Parse(tree, source)
  result = printer.Print(subject, storage, tree, source)
  testify.Equal(
    t,
    again,
    string(storage[:result.Count]),
    "a printer that borrowed the buffers once prints the next source into them",
  )
}

// Reports that one canonical source prints as itself, which is what a caller asks when it wants
// to know whether a file is clean.
func round_trip(t *testing.T, source string) {
  t.Helper()
  testify.Equal(t, source, printed(t, source), "the canonical form prints as itself")
}

func test_printer(t *testing.T) {
  lent_print(t)
  round_trip(t, "package one\n")
  round_trip(t, "package one\n\nimport \"local/example/two\"\n")
  // A name another package of the file already binds stands, thus the import that states it
  // keeps the name the author wrote.
  round_trip(t, "package one\n\nimport \"local/example/two\"\nimport two \"other/two\"\n")
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  source := token.Source("package one\n")
  ast.Parse(tree, source)
  result := printer.Print(subject, storage, tree, source)
  testify.True(t, bool(result.OK), "a fresh printer prints")
  testify.Equal(t, printer.Form_Count(len(source)), result.Count,
    "the print states what it wrote")
  second := printer.Print(subject, storage, tree, source)
  testify.True(t, bool(second.OK), "one printer prints one file after another")
  testify.Equal(t, result.Count, second.Count,
    "the second print writes what the first one did")
}

func test_layout(t *testing.T) {
  // A declaration closes the line it stands on, thus a body that holds one stands on lines
  // of its own however the author wrote it.
  round_trip(t, "package one\n\nfunc Fold() {\n  defer func() { a := 1; _ = a }()\n}\n")
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  defer func() {\n    a := 1\n  }()\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  defer func() { var a = 1 }()\n}\n"),
    "a body that holds a declaration stands on lines of its own",
  )
  test_broken_chain(t)
  test_broken_call(t)
  // A step states one pair of signs behind the value it holds, in either direction.
  round_trip(t, "package one\n\nfunc Fold(one int) {\n  one++\n  one--\n}\n")
  // An empty line ahead of a simple error check states nothing, thus the check stands
  // against the call that bound the error it reads.
  testify.Equal(
    t,
    "package one\n\nfunc F() (err error) {\n  one, err := g()\n"+
      "  if err != nil {\n    return err\n  }\n  _ = one\n  return nil\n}\n",
    printed(
      t,
      "package one\n\nfunc F() (err error) {\n  one, err := g()\n\n"+
        "  if err != nil {\n    return err\n  }\n  _ = one\n  return nil\n}\n",
    ),
    "an empty line ahead of a simple error check states nothing",
  )
  // A list of no fields closes on the line it opened on, and the empty line the author
  // left inside it stands behind the declaration rather than between its braces.
  testify.Equal(
    t,
    "package one\n\ntype S struct{}\n\ntype T int\n",
    printed(t, "package one\n\ntype S struct {\n\n}\n\ntype T int\n"),
    "a list of no fields drops the empty lines it holds",
  )
  test_forms(t)
  test_noted_parts(t)
  // A field that binds a run of names states a comma and a space between two of them.
  round_trip(t, "package one\n\ntype S struct {\n  One, Two int\n  Three    string\n}\n")
  // A function type states the signature a function literal opens with and no body.
  round_trip(t, "package one\n\nvar fold func(one int, two int) (sum int)\n")
  round_trip(t, "package one\n\ntype Fold func(one int) error\n")
  // An assertion reads one type from the value ahead of it, and a type switch states the
  // word type where an assertion states a type.
  round_trip(t, "package one\n\nfunc Fold(one any) {\n  two := one.(int)\n  _ = two\n}\n")
  round_trip(
    t,
    "package one\n\nfunc Fold(one any) {\n  switch one.(type) {\n"+
      "  case int:\n  }\n}\n",
  )
  // A word statement states its word and the value behind it where it binds one.
  round_trip(t, "package one\n\nfunc Fold() {\nloop:\n  g()\n  goto loop\n}\n")
  test_switch_cases(t)
  // A label names the statement behind it and stands one tab shallower than it.
  round_trip(t, "package one\n\nfunc Fold() {\nloop:\n  g()\n}\n")
  // A body that stands on the line its brace opens states a semicolon between the
  // statements it holds.
  round_trip(t, "package one\n\nfunc Fold() {\n  defer func() { one(); two() }()\n}\n")
  // A clause that states no case and no body closes on its braces, the way every other
  // empty body does.
  round_trip(t, "package one\n\nfunc s() {\n  select {}\n}\n")
  round_trip(t, "package one\n\nfunc s() {\n  switch {}\n}\n")
  round_trip(t, "package one\n\nfunc s() {\n  switch a {}\n}\n")
  test_wide_head(t)
  test_wide_tail(t)
  round_trip(t, "package one\n\nconst LIMIT int = 8\n")
  round_trip(t, "package one\n\nvar table [8]int\n")
  round_trip(t, "package one\n\ntype Count int\n")
  round_trip(t, "package one\n\nfunc Fold() (sum int) {\n  return sum\n}\n")
  round_trip(t, "package one\n\nfunc Fold(one int, two int) (sum int) {\n  return one\n}\n")
  round_trip(t, "package one\n\nfunc Fold(one int) (sum int)\n")
  round_trip(t, "package one\n\nfunc Fold(one int) (sum int) // Fold sums.\n")
  round_trip(
    t,
    "package one\n\nfunc Fold() (sum int) {\n  if sum > 0 {\n    return sum\n"+
      "  }\n  return 0\n}\n",
  )
  round_trip(
    t,
    "package one\n\nfunc Fold() (sum int) {\n  for step := range 4 {\n"+
      "    sum = sum + step\n  }\n  return sum\n}\n",
  )
  round_trip(
    t,
    "package one\n\nfunc Fold() {\n  one := fold(1,\n    []int{2, 3})\n"+
      "  _ = one\n}\n",
  )
  round_trip(
    t,
    "package one\n\nfunc Fold() {\n  one := fold(1,\n    []int{\n"+
      "      2,\n      3,\n    })\n  _ = one\n}\n",
  )
  round_trip(
    t,
    "package one\n\nfunc Fold() {\n  one, two := 1,\n    []int{2, 3}\n"+
      "  _, _ = one, two\n}\n",
  )
}

// Prints the parts that carry a note on the line they close: a field, a term, a parameter, and
// an import. A note the author left on such a line keeps it.
func test_noted_parts(t *testing.T) {
  round_trip(t, "package one\n\ntype S struct {\n  One int // a\n  Two string // b\n}\n")
  round_trip(t, "package one\n\ntype R interface {\n  ~int // a\n  ~string // b\n}\n")
  round_trip(t, "package one\n\nfunc f(\n  one int, // a\n  two string, // b\n) {\n}\n")
  round_trip(t, "package one\n\nimport \"one\" // a\n")
}

// Prints the forms a declaration states: the notes a constraint holds, the types a generic binds,
// and the type a field embeds.
func test_forms(t *testing.T) {
  // A file that opens on a note states that note ahead of the package clause it binds.
  round_trip(t, "// One states nothing.\npackage one\n\nvar held = 1\n")
  round_trip(t, "// One states nothing.\n\n// And nothing else.\npackage one\n")
  // A channel names the way it carries, and a select reads one value from it.
  round_trip(t, "package one\n\nfunc f(held chan<- int, one <-chan int) {\n}\n")
  round_trip(
    t,
    "package one\n\nfunc f(ch chan int) {\n  select {\n  case v := <-ch:\n"+
      "    g(v)\n  }\n}\n",
  )
  // A literal that holds literals states the type of its elements once.
  round_trip(t, "package one\n\nvar held = map[string][]Pair{\n  \"one\": {{Left: 1}},\n}\n")
  // A method of a generic type binds the names its receiver states.
  round_trip(
    t,
    "package one\n\nfunc (one *Pair[T]) Method() (left T, err error) {\n"+
      "  return left, nil\n}\n",
  )
  // A call that spreads a value states the three points against that value, and a call
  // that opens on the points keeps the brackets it holds its argument between.
  round_trip(t, "package one\n\nfunc f(held []int) {\n  g(held...)\n}\n")
  round_trip(t, "package one\n\nfunc f(held []int) {\n  g(...held)\n}\n")
  round_trip(t, "package one\n\nfunc f(held []int) {\n  g(one, ...held)\n}\n")
  round_trip(t, "package one\n\nfunc f(held []int) {\n  g(one, held...)\n}\n")
  // A structure the author stated on one line keeps that line, and states a semicolon
  // between two fields of it.
  round_trip(t, "package one\n\nvar held struct{ One int }\n")
  round_trip(t, "package one\n\nvar held struct{ One int; Two string }\n")
  round_trip(t, "package one\n\nvar held struct{ Reader }\n")
  // A constraint holds the notes and the empty lines the author left between its terms.
  round_trip(t, "package one\n\ntype Reader interface {\n  // One.\n  ~int\n\n  ~string\n}\n")
  // A generic names the run of types it binds between brackets.
  round_trip(t, "package one\n\nvar held Pair[int, string]\n")
  round_trip(t, "package one\n\nvar held = Pair[int]{}\n")
  // A field that binds no name of its own states the type it embeds.
  round_trip(t, "package one\n\ntype S struct {\n  Reader\n  Name string\n}\n")
}

func test_elision(t *testing.T) {
  // A note the author left between the type one variable binds and the value that names
  // the same type states neither, thus the print drops the type it would drop without it.
  testify.Equal(
    t,
    "package one\n\nvar x = []int{1} /* note */\n",
    printed(t, "package one\n\nvar x []int/* note */ = []int{1}\n"),
    "a noted type its value names writes no byte",
  )
  testify.Equal(
    t,
    "package one\n\nvar x = []int{1} /* note */\n",
    printed(t, "package one\n\nvar x []int = /* note */ []int{1}\n"),
    "a type the value behind a note names writes no byte",
  )
  testify.Equal(
    t,
    "package one\n\nvar x = []int{1} /* note */\n",
    printed(t, "package one\n\nvar x/* note */ []int = []int{1}\n"),
    "a type behind a noted name writes no byte",
  )
  testify.Equal(
    t,
    "package one\n\nvar x, y = []int{1}, []int{2} /* note */\n",
    printed(t, "package one\n\nvar x, y/* note */ []int = []int{1}, []int{2}\n"),
    "a type behind a noted run of names writes no byte",
  )
  // A note the author left inside the brackets states nothing of the bound they hold, thus
  // the slice closes at its own length however many notes stand between its tokens.
  testify.Equal(
    t,
    "package one\n\nvar whole = a[1:] /* note */\n",
    printed(t, "package one\n\nvar whole = a[1:len(a) /* note */]\n"),
    "a note inside the brackets states nothing of the bound",
  )
  // A slice that closes at the length of the value it reads states that bound nowhere, thus
  // the print writes the brackets and drops it.
  testify.Equal(
    t,
    "package one\n\nvar whole = a[:]\n",
    printed(t, "package one\n\nvar whole = a[:len(a)]\n"),
    "a bound that states the length of the value stands nowhere",
  )
  // A slice that reads a value of any wider form than a name may read otherwise than the
  // value the brackets close, thus its closing bound stands as the author wrote it.
  round_trip(
    t,
    "package one\n\nfunc Fold(pair Pair) {\n"+
      "  one := pair.Left[:len(pair.Left)]\n  _ = one\n}\n",
  )
  round_trip(
    t,
    "package one\n\nfunc Fold(values []int) {\n"+
      "  one := values[1:len(values):cap(values)]\n  _ = one\n}\n",
  )
  round_trip(
    t,
    "package one\n\nfunc Fold(values []int) {\n"+
      "  one := values[1 : len(values)-1]\n  _ = one\n}\n",
  )
  round_trip(
    t,
    "package one\n\nfunc Fold(values []int, held []int) {\n"+
      "  one := values[1:len(held)]\n  _ = one\n}\n",
  )
  round_trip(t, "package one\n\ntype Empty struct{}\n")
  round_trip(t, "package one\n\ntype Marker interface{}\n")
  round_trip(t, "package one\n\nfunc Fold() {\n  one := 1\n  _ = one\n}\n")
  round_trip(t, "package one\n\nfunc Fold() {\n  var one int\n  _ = one\n}\n")
  round_trip(t, "package one\n\nfunc Fold() {\n  var one int = 1\n  _ = one\n}\n")
  // A literal that states no type of its own states elements of a type no form spells, thus
  // the print writes the type its elements name however the literal that holds it stands.
  round_trip(t, "package one\n\nvar table = []Pair{{Left: 1}}\n")
  round_trip(t, "package one\n\nvar table = [][]Pair{{Pair{Left: 1}}}\n")
  round_trip(t, "package one\n\nvar table = []any{Pair{Left: 1}}\n")
  round_trip(t, "package one\n\nvar table = Table{Pair{Left: 1}}\n")
  round_trip(t, "package one\n\nvar table = []*Pair{{Left: 1}}\n")
  round_trip(t, "package one\n\nvar one = &Pair{Left: 1}\n")
  // A variable whose values name the type it binds states that type once, thus the print
  // writes the values alone.
  testify.Equal(
    t,
    "package one\n\nvar held = Pair{}\n",
    printed(t, "package one\n\nvar held Pair = Pair{}\n"),
    "a variable whose value names its type states that type once",
  )
  testify.Equal(
    t,
    "package one\n\nvar held = []int{1}\n",
    printed(t, "package one\n\nvar held []int = []int{1}\n"),
    "a variable whose value names the slice it binds states that type once",
  )
  round_trip(t, "package one\n\nvar held *Pair = &Pair{Left: 1}\n")
  round_trip(t, "package one\n\nvar held Pair = Other{}\n")
  testify.Equal(
    t,
    "package one\n\nvar one, two = Pair{}, Pair{}\n",
    printed(t, "package one\n\nvar one, two Pair = Pair{}, Pair{}\n"),
    "a run of names whose values name the type states that type once",
  )
  test_elision_forms(t)
}

// States the forms the canonical one writes where the author wrote what another form states.
func test_elision_forms(t *testing.T) {
  test_elision_written(t)
}

// States the name an import drops, which is the name the path it states already names.
func test_elision_imports(t *testing.T) {
  // A name the path already states says nothing twice, thus the import drops it and every
  // form that read it names the package the path states. A name another import already
  // binds stands, because dropping it would leave two packages of one name.
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {
      Source: "package one\n\nimport errors \"errors\"\n",
      Form:   "package one\n\nimport \"errors\"\n",
    },
    {
      Source: "package one\n\nimport held \"errors\"\n\n" +
        "func Fold() (err error) {\n  err = held.New(\"one\")\n" +
        "  return err\n}\n",
      Form: "package one\n\nimport \"errors\"\n\n" +
        "func Fold() (err error) {\n  err = errors.New(\"one\")\n" +
        "  return err\n}\n",
    },
    {
      Source: "package one\n\nimport held \"errors\"\nimport text \"strings\"\n",
      Form:   "package one\n\nimport \"errors\"\nimport \"strings\"\n",
    },
    {
      Source: "package one\n\nimport invariant " +
        "\"local/james-orcales/shared/std/sim/aver/default\"\n",
      Form: "package one\n\nimport " +
        "\"local/james-orcales/shared/std/sim/aver/default\"\n",
    },
  } {
    testify.Equal(
      t,
      one.Form,
      printed(t, one.Source),
      "an import states the name its path already names",
    )
  }
  round_trip(
    t,
    "package one\n\nimport \"errors\"\nimport held \"one/errors\"\n\n"+
      "func Fold() (err error) {\n  err = held.New(\"one\")\n  return err\n}\n",
  )
  round_trip(t, "package one\n\nimport \"zz\"\n\nfunc Fold() {\n  zz.One()\n}\n")
  // A note the author left between the sign and the word range states nothing of the names
  // the clause binds, thus the clause drops the blank it binds to nothing all the same.
  testify.Equal(
    t,
    "package one\n\nfunc Nine(two []int) {\n  for one := range two { /*a*/\n"+
      "    _ = one\n  }\n}\n",
    printed(
      t,
      "package one\n\nfunc Nine(two []int) {\n"+
        "  for one, _ :=/*a*/ range two {\n    _ = one\n  }\n}\n",
    ),
    "a noted sign states nothing of the blank a range binds to nothing",
  )
  // A path element that spells no name a file can bind names no package, thus the import
  // keeps the name the author wrote for it.
  round_trip(t, "package one\n\nimport held \"a/b-c\"\n\nfunc Fold() {\n  held.One()\n}\n")
  round_trip(t, "package one\n\nimport held \"a/b.c\"\n\nfunc Fold() {\n  held.One()\n}\n")
  round_trip(t, "package one\n\nimport held \"a/1b\"\n\nfunc Fold() {\n  held.One()\n}\n")
  // The print holds as many dropped names as its storage admits, and a name past them stands
  // as the author wrote it.
  imports := "package one\n"
  for index := range printer.IMPORT_COUNT_MAXIMUM + 1 {
    pair := string([]byte{byte('a' + index/26), byte('a' + index%26)})
    imports = imports + "\nimport x" + pair + " \"p" + pair + "\"\n"
  }
  form := print_form(t, imports, false)
  testify.True(t, !(offset_of(form, "import \"paa\"\n") < 0), "the first name drops")
  testify.True(t, !(offset_of(form, "import xcm \"pcm\"\n") < 0),
    "a name past the storage stands")
}

// A type the values behind it name writes no byte, thus the declaration binds the short sign the
// way one the author wrote without a type does.
func test_elision_short(t *testing.T) {
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  one := []int{1}\n  _ = one\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  var one []int = []int{1}\n"+
      "  _ = one\n}\n"),
    "a variable whose values name its type binds the short sign",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  var one int = 1\n  _ = one\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  var one int = 1\n  _ = one\n}\n"),
    "a variable of a type no value names keeps the word it opens with",
  )
}

// States the forms the canonical one writes where the author wrote what another form states.
func test_elision_written(t *testing.T) {
  test_elision_short(t)
  // A field list of no fields closes on the line it opened on, and a variable of values and
  // no type binds the short sign, thus neither form waits on the author to write it.
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {
      Source: "package one\n\ntype Empty struct {\n}\n",
      Form:   "package one\n\ntype Empty struct{}\n",
    },
    {
      Source: "package one\n\nfunc Fold(values []int) {\n" +
        "  for _ = range values {\n  }\n}\n",
      Form: "package one\n\nfunc Fold(values []int) {\n" +
        "  for range values {\n  }\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold(values []int) {\n" +
        "  for one, _ := range values {\n    _ = one\n  }\n}\n",
      Form: "package one\n\nfunc Fold(values []int) {\n" +
        "  for one := range values {\n    _ = one\n  }\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold(values []int) {\n" +
        "  one := values[1:len(values)]\n  _ = one\n}\n",
      Form: "package one\n\nfunc Fold(values []int) {\n" +
        "  one := values[1:]\n  _ = one\n}\n",
    },
    {
      Source: "package one\n\ntype Marker interface {\n}\n",
      Form:   "package one\n\ntype Marker interface{}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  var one = 1\n  _ = one\n}\n",
      Form:   "package one\n\nfunc Fold() {\n  one := 1\n  _ = one\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  var one, two = 1, 2\n" +
        "  _, _ = one, two\n}\n",
      Form: "package one\n\nfunc Fold() {\n  one, two := 1, 2\n" +
        "  _, _ = one, two\n}\n",
    },
    {
      Source: "package one\n\nvar table = []Pair{Pair{Left: 1}}\n",
      Form:   "package one\n\nvar table = []Pair{{Left: 1}}\n",
    },
    {
      Source: "package one\n\nvar table = map[Pair]Pair{Pair{Left: 1}: " +
        "Pair{Left: 2}}\n",
      Form: "package one\n\nvar table = map[Pair]Pair{{Left: 1}: {Left: 2}}\n",
    },
    {
      Source: "package one\n\nvar table = []*Pair{&Pair{Left: 1}}\n",
      Form:   "package one\n\nvar table = []*Pair{{Left: 1}}\n",
    },
    {
      Source: "package one\n\nvar table = [2]Pair{Pair{Left: 1}, " +
        "Pair{Left: 2}}\n",
      Form: "package one\n\nvar table = [2]Pair{{Left: 1}, {Left: 2}}\n",
    },
  } {
    testify.Equal(
      t,
      one.Form,
      printed(t, one.Source),
      "the canonical form states what the author left unwritten",
    )
  }
}

func test_spread(t *testing.T) {
  round_trip(
    t,
    "package one\n\nfunc Fold(one int,\n  two int,\n) (held bool) {\n"+
      "  return held\n}\n",
  )
  round_trip(t, "package one\n\nvar table = []Pair{\n  {Left: 1},\n  {Left: 2},\n}\n")
  // A call the author opened on the line its parenthesis closes keeps that line, thus only
  // a call whose parenthesis closes its own line closes on a line of its own.
  round_trip(t, "package one\n\nfunc Fold() {\n  Report(one, two,\n    three)\n}\n")
  round_trip(t, "package one\n\nfunc Fold() {\n  Report(\n    one,\n    two,\n  )\n}\n")
  round_trip(
    t,
    "package one\n\nfunc Fold(one int) {\n  switch one {\n  case 1, 2:\n"+
      "    one = 3\n  }\n}\n",
  )
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {
      Source: "package one\n\nfunc Fold(one int,\n  two int) (held bool) {\n" +
        "  return held\n}\n",
      Form: "package one\n\nfunc Fold(one int,\n  two int,\n) (held bool) {\n" +
        "  return held\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  Report(\n    one,\n" +
        "    two)\n}\n",
      Form: "package one\n\nfunc Fold() {\n  Report(\n    one,\n" +
        "    two,\n  )\n}\n",
    },
    {
      Source: "package one\n\nvar table = []Pair{{Left: 1},\n  {Left: 2}}\n",
      Form: "package one\n\nvar table = []Pair{\n  {Left: 1},\n  {Left: 2},\n" +
        "}\n",
    },
    {
      Source: "package one\n\nfunc Fold(one int) {\n  switch one {\n  case 1,\n" +
        "    2:\n    one = 3\n  }\n}\n",
      Form: "package one\n\nfunc Fold(one int) {\n  switch one {\n" +
        "  case 1, 2:\n    one = 3\n  }\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  return\n}\nfunc Step() {\n" +
        "  return\n}\n",
      Form: "package one\n\nfunc Fold() {\n  return\n}\n\nfunc Step() {\n" +
        "  return\n}\n",
    },
  } {
    testify.Equal(
      t,
      one.Form,
      printed(t, one.Source),
      "a form the author broke states one part to a line",
    )
  }
}

func test_width(t *testing.T) {
  // A signature that runs the line past the widest one states one parameter to a line, thus
  // the print reads a line that holds the bracket closing it and nothing else.
  parameters := ""
  for range 18 {
    parameters = parameters + "one int, "
  }
  spread := printed(t, "package one\n\nfunc Fold("+parameters+") {\n}\n")
  testify.Equal(t, spread, printed(t, spread),
    "a signature of many parameters prints as itself")
  // A form that stands inside the widest line reads on one line, thus the print breaks
  // nothing the author closed up.
  round_trip(t, "package one\n\nfunc Fold() {\n  Report(one, two, three)\n}\n")
  round_trip(t, "package one\n\nvar table = []int{1, 2, 3}\n")
  round_trip(t, "package one\n\nfunc Fold(one int, two int) (held bool) {\n"+
    "  return held\n}\n")
  // A note the author laid out states lines of its own: a directive names a tool and a note
  // that opens on spaces counts the columns it stands in, thus the print wraps neither.
  round_trip(
    t,
    "package one\n\n//go:generate echo one two three four five six seven "+
      "eight nine ten eleven twelve thirteen\ntype Count int\n",
  )
  round_trip(
    t,
    "package one\n\n//   one two three four five six seven eight nine ten "+
      "eleven twelve thirteen fourteen\ntype Count int\n",
  )
  // A call whose last argument states a body of its own still reads the line it opens,
  // thus a head past the widest line breaks at every argument the call holds.
  wide_call := "package one\n\nfunc F() {\n  g(" +
    spelled('w', 60) +
    ", " +
    spelled('w', 60) +
    ", func() {\n    h()\n  })\n}\n"
  testify.True(
    t,
    !(offset_of(printed(t, wide_call), "  g(\n    ") < 0),
    "a call that holds a body breaks its arguments where its head runs over",
  )
  // A call that stands as the value of a pair counts the comma the literal writes behind
  // it, thus the line it closes stands inside the widest line.
  pair_call := "package one\n\nvar held = map[string]int{\n  \"one\": g(" +
    spelled('w', 40) +
    ", " +
    spelled('w', 40) +
    "),\n}\n"
  testify.True(
    t,
    !(widest_line(printed(t, pair_call)) > printer.COLUMN_WIDTH_MAXIMUM),
    "a call inside a pair breaks rather than run past the widest line",
  )
  test_wide_forms(t)
  test_width_forms(t)
  // A run of one sign past the widest line breaks at every sign it states, and a grouping
  // inside that run stands in it, whichever binding the sign holds.
  for _, sign := range []string{"==", "+", "*"} {
    run := "package one\n\nvar held = first_value " +
      sign +
      " (second_value " +
      sign +
      " third_value) " +
      sign +
      " fourth_value " +
      sign +
      " fifth_value " +
      sign +
      " sixth_value " +
      sign +
      " seventh_value\n"
    form := printed(t, run)
    testify.True(
      t,
      !(offset_of(form, " "+sign+"\n  (second_value") < 0),
      "the run breaks ahead of the grouping it holds",
    )
    testify.True(
      t,
      !(offset_of(form, " "+sign+"\n  seventh_value\n") < 0),
      "the run breaks at its last sign",
    )
  }
}

// Prints the forms that hold a call wherever a statement holds one, each of them wide enough that
// the print breaks it, and reads the widest line each one closes on.
func test_wide_forms(t *testing.T) {
  wide := spelled('w', 40)
  for _, one := range []string{
    "package one\n\nfunc f() {\n  return g(WIDE, WIDE)\n}\n",
    "package one\n\nfunc f() {\n  switch {\n  case g(WIDE, WIDE):\n  }\n}\n",
    "package one\n\nvar held = []int{g(WIDE, WIDE)}\n",
    "package one\n\nfunc f() {\n  if g(WIDE, WIDE) {\n  }\n}\n",
    "package one\n\nvar held = Pair{One: g(WIDE, WIDE)}\n",
    "package one\n\nvar held = map[string]int{\n  \"one\": g(WIDE, WIDE),\n}\n",
  } {
    held := ""
    for _, value := range []byte(one) {
      held = held + string(rune(value))
    }
    source := ""
    for index := 0; index < len(held); index = index + 1 {
      if !(index+4 > len(held)) {
        if held[index:index+4] == "WIDE" {
          source = source + wide
          index = index + 3
          continue
        }
      }
      source = source + string(held[index])
    }
    testify.True(
      t,
      !(widest_line(printed(t, source)) > printer.COLUMN_WIDTH_MAXIMUM),
      "a form that runs past the widest line breaks rather than run past it",
    )
  }
}

// States the forms the canonical one breaks because they would run past the widest line.
func test_width_forms(t *testing.T) {
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {
      Source: "package one\n\nfunc Fold() {\n  Report(first_value, " +
        "second_value, third_value, fourth_value, fifth_value, " +
        "sixth_value, seventh_value)\n}\n",
      Form: "package one\n\nfunc Fold() {\n  Report(\n    first_value,\n" +
        "    second_value,\n    third_value,\n    fourth_value,\n" +
        "    fifth_value,\n    sixth_value,\n    seventh_value,\n" +
        "  )\n}\n",
    },
    {
      Source: "package one\n\nvar table = []int{100000, 200000, 300000, " +
        "400000, 500000, 600000, 700000, 800000, 900000, 110000, " +
        "120000, 130000}\n",
      Form: "package one\n\nvar table = []int{\n  100000,\n  200000,\n" +
        "  300000,\n  400000,\n  500000,\n  600000,\n  700000,\n" +
        "  800000,\n  900000,\n  110000,\n  120000,\n  130000,\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold(first_value int, second_value int, " +
        "third_value int, fourth_value int, fifth_value int) " +
        "(held bool) {\n  return held\n}\n",
      Form: "package one\n\nfunc Fold(\n  first_value int,\n" +
        "  second_value int,\n  third_value int,\n  fourth_value int,\n" +
        "  fifth_value int,\n) (held bool) {\n  return held\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() (sum int) {\n  sum = " +
        "first_value + second_value + third_value + fourth_value " +
        "+ fifth_value + sixth_value + seventh_value\n" +
        "  return sum\n}\n",
      Form: "package one\n\nfunc Fold() (sum int) {\n  sum = " +
        "first_value +\n    second_value +\n    third_value +\n" +
        "    fourth_value +\n    fifth_value +\n    sixth_value +\n" +
        "    seventh_value\n  return sum\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() (first_value bool, second_value " +
        "error, third_value int, fourth_value string, fifth_value " +
        "int) {\n  return first_value, second_value, third_value, " +
        "fourth_value, fifth_value\n}\n",
      Form: "package one\n\nfunc Fold() (\n  first_value bool,\n" +
        "  second_value error,\n  third_value int,\n" +
        "  fourth_value string,\n  fifth_value int,\n) {\n" +
        "  return first_value, second_value, third_value, " +
        "fourth_value, fifth_value\n}\n",
    },
    {
      Source: "package one\n\n// A note that runs a good deal past the widest " +
        "line the canonical form writes, holding words and more words.\n" +
        "type Count int\n",
      Form: "package one\n\n// A note that runs a good deal past the widest " +
        "line the canonical form writes, holding words and\n// more " +
        "words.\ntype Count int\n",
    },
  } {
    testify.Equal(
      t,
      one.Form,
      printed(t, one.Source),
      "a form that runs past the widest line states one part to a line",
    )
  }
}

func test_order(t *testing.T) {
  test_order_notes(t)
  // A note the author left on the line a type closes keeps that line when the function of
  // invariants moves under that type.
  round_trip(t, "package one\n\ntype A int // one\n\nfunc A_Invariants() {\n}\n")
  round_trip(t, "package one\n\ntype A int\n\nfunc A_Invariants() { // one\n}\n")

  // A function of invariants stands under the type it names however the author ordered them.
  round_trip(
    t,
    "package one\n\ntype A int\n\nfunc A_Invariants() {\n}\n\ntype B int\n\n"+
      "func B_Invariants() {\n}\n",
  )
  // One function of invariants stands under one type, thus a second type of that name takes
  // none of it.
  round_trip(t, "package one\n\ntype A int\n\nfunc A_Invariants() {\n}\n\ntype A int\n")
  // One type takes one function of invariants under it, thus a second function of that name
  // stands where the author wrote it.
  round_trip(
    t,
    "package one\n\ntype A int\n\nfunc A_Invariants() {\n}\n\n"+
      "func A_Invariants() {\n}\n",
  )
  // A function that states the invariants of a type the file never declares names no type
  // to stand under, thus it stands where the author wrote it.
  round_trip(
    t,
    "package one\n\ntype Count int\n\n"+
      "// Count_Invariants states one count.\nfunc Count_Invariants(value Count) {\n}\n",
  )
  round_trip(t, "package one\n\nfunc Held_Invariants(value Held) {\n}\n\n"+
    "func Fold() {\n}\n")
  round_trip(t, "package one\n\ntype Count int\n\nfunc Fold() {\n}\n")
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {
      Source: "package one\n\ntype Count int\n\nfunc Fold() {\n}\n\n" +
        "func Count_Invariants(value Count) {\n}\n",
      Form: "package one\n\ntype Count int\n\n" +
        "func Count_Invariants(value Count) {\n}\n\nfunc Fold() {\n}\n",
    },
    {
      Source: "package one\n\ntype Count int\n\nfunc Fold() {\n}\n\n" +
        "// Count_Invariants states one count.\n" +
        "func Count_Invariants(value Count) {\n}\n",
      Form: "package one\n\ntype Count int\n\n" +
        "// Count_Invariants states one count.\n" +
        "func Count_Invariants(value Count) {\n}\n\nfunc Fold() {\n}\n",
    },
    {
      Source: "package one\n\nfunc Count_Invariants(value Count) {\n}\n\n" +
        "type Count int\n\nfunc Fold() {\n}\n",
      Form: "package one\n\ntype Count int\n\n" +
        "func Count_Invariants(value Count) {\n}\n\nfunc Fold() {\n}\n",
    },
    {
      Source: "package one\n\ntype Count int\n\ntype Mark int\n\n" +
        "func Mark_Invariants(value Mark) {\n}\n\n" +
        "func Count_Invariants(value Count) {\n}\n",
      Form: "package one\n\ntype Count int\n\n" +
        "func Count_Invariants(value Count) {\n}\n\ntype Mark int\n\n" +
        "func Mark_Invariants(value Mark) {\n}\n",
    },
  } {
    testify.Equal(
      t,
      one.Form,
      printed(t, one.Source),
      "a function that states the invariants of a type stands under that type",
    )
  }
}

func test_returns(t *testing.T) {
  // A function that hands nothing back names nothing to write, and a result the signature
  // names as the blank states no name a return can read, thus both stand as written.
  round_trip(
    t,
    "package one\n\nfunc Fold(one int) {\n  if one > 0 {\n    return\n"+
      "  }\n  one++\n}\n",
  )
  round_trip(t, "package one\n\nfunc Fold() (_ bool) {\n  return\n}\n")
  round_trip(t, "package one\n\nfunc Fold() (held bool) {\n  return held\n}\n")
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {
      Source: "package one\n\nfunc Fold(one int) (held bool) {\n" +
        "  if one > 0 {\n    return\n  }\n  return held\n}\n",
      Form: "package one\n\nfunc Fold(one int) (held bool) {\n" +
        "  if one > 0 {\n    return held\n  }\n  return held\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() (held bool, err error) {\n" +
        "  return\n}\n",
      Form: "package one\n\nfunc Fold() (held bool, err error) {\n" +
        "  return held, err\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() (one bool, two bool) {\n  return\n}\n",
      Form: "package one\n\nfunc Fold() (one bool, two bool) {\n" +
        "  return one, two\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() (held bool) {\n" +
        "  step := func() {\n    return\n  }\n  step()\n  return\n}\n",
      Form: "package one\n\nfunc Fold() (held bool) {\n" +
        "  step := func() {\n    return\n  }\n  step()\n  return held\n}\n",
    },
  } {
    testify.Equal(
      t,
      one.Form,
      printed(t, one.Source),
      "a return states the values the signature names",
    )
  }
  // A signature binds as many result names as the storage holds, and one that binds more
  // states a form no return writes, thus its return stands as the author wrote it.
  widest := printer.RESULT_COUNT_MAXIMUM
  for _, count := range []int{widest, widest + 1} {
    results := ""
    written := ""
    for index := range count {
      name := "r" + string([]byte{byte('a' + index)})
      if index > 0 {
        results = results + ", "
        written = written + ","
      }
      results = results + name + " bool"
      written = written + " " + name
    }
    form := printed(t, "package one\n\nfunc Fold() ("+results+") {\n  return\n}\n")
    if count > printer.RESULT_COUNT_MAXIMUM {
      written = ""
    }
    testify.True(
      t,
      !(offset_of(form, " return"+written+"\n") < 0),
      "a return writes the names the storage holds",
    )
  }
}

func test_spacing(t *testing.T) {
  test_clashing_signs(t)
  // Spaces and tabs before a raw literal's newline are part of its value.
  round_trip(t, "package one\n\nconst TEXT = `one \ntwo\t\n`\n")
  testify.Equal(
    t,
    "package one\n\nconst TEXT = `one \ntwo\t\n`\n",
    printed(t, "package one\n\nconst TEXT = `one \r\ntwo\t\r\n`\n"),
    "raw string line endings keep value whitespace while dropping carriage returns",
  )
  // A statement of one word states that word and the value it binds where it binds one.
  round_trip(t, "package one\n\nfunc f() {\n  go g()\n  defer h()\n}\n")
  // An assignment states the binding its sign holds: the names it reads, the names it
  // states itself, the operation it binds, and the channel it sends to.
  round_trip(
    t,
    "package one\n\nfunc f(sum int, held chan int) {\n  sum += one\n"+
      "  sum *= two\n  held <- sum\n}\n",
  )
  // A loop that leaves its condition out writes both semicolons its headers stand between.
  round_trip(t, "package one\n\nfunc f() {\n  for step := 0; ; step++ {\n  }\n}\n")
  // A raw literal holds any byte a file can, thus the print writes the whole byte domain
  // rather than the letters a name spells.
  round_trip(t, "package one\n\nconst TEXT = `\x00\x01\x02\xff`\n")
  for _, one := range []string{
    "  sum = one | two\n",
    "  sum = one & two\n",
    "  sum = one & ^two\n",
    "  sum = one / *pointer\n",
    "  sum = one + +two\n",
    "  sum = one - -two\n",
    "  sum = one + two\n",
    "  sum = one - two\n",
    "  sum = -one\n",
    "  sum = !held\n",
    "  sum = one(two, three)\n",
    "  sum = table[one]\n",
    "  sum = table[one:two]\n",
    "  sum = held.Value\n",
    "  sum = &held\n",
    "  sum = *pointer\n",
    "  sum = one == two\n",
    "  sum = one\n",
    "  sum++\n",
    "  sum, held = one, two\n",
  } {
    round_trip(t, "package one\n\nfunc Fold() {\n"+one+"}\n")
  }
}

func test_alignment(t *testing.T) {
  test_wide_columns(t)
  test_widest_line(t)
  test_trailing_line(t)
  test_broken_element(t)
  test_widest_column(t)
  test_tag_notes(t)
  // A note a pair carries stands in a column of its own rather than closing the run the
  // pair stands in, thus a pair carrying two of them stands in that run as well.
  testify.Equal(
    t,
    "package one\n\nvar six = map[string]int{\n  \"one\":        1, /*a*/ /*b*/\n"+
      "  \"twoooooooo\": 2,\n}\n",
    printed(
      t,
      "package one\n\nvar six = map[string]int{\n  \"one\": 1, /*a*/ /*b*/\n"+
        "  \"twoooooooo\": 2,\n}\n",
    ),
    "a pair carrying two notes stands in the run it opened",
  )
  // A pair the author broke between its key and its value stands on one line where it
  // fits, thus the run it stands in stays open and every value of that run aligns.
  testify.Equal(
    t,
    "package one\n\nvar five = map[string]int{\n  \"one\":        1,\n"+
      "  \"twoooooooo\": 2,\n}\n",
    printed(
      t,
      "package one\n\nvar five = map[string]int{\n  \"one\":\n  1,\n"+
        "  \"twoooooooo\": 2,\n}\n",
    ),
    "a pair the print joined stands in the run it opened",
  )
  // A field binds the names it states on one line, thus a note the author left between
  // them stands on a line of its own behind the field.
  testify.Equal(
    t,
    "package one\n\ntype S struct {\n  A, Bbbb int\n  // note\n}\n",
    printed(t, "package one\n\ntype S struct {\n  A,\n  // note\n  Bbbb int\n}\n"),
    "a field binds its names on one line",
  )
  // A note the author left on a line of its own closes the run of fields it stands in, and
  // the fields behind it align on a width of their own.
  round_trip(t, "package one\n\ntype S struct {\n  A int\n  // note\n  Bbbb int\n}\n")
  round_trip(
    t,
    "package one\n\ntype S struct {\n  A    int // one\n  Bbbb int // two\n"+
      "  C    int\n}\n",
  )
  // A note the author left on the line one field closes keeps that line, thus the fields
  // around it stand in one run and align their types and their tags.
  round_trip(
    t,
    "package one\n\ntype S struct {\n  A  int `one:\"1\"` // one\n"+
      "  Bb int `two:\"2\"` // two\n}\n",
  )
  // A statement that opens with a word of its own carries its note the way every other
  // statement does, thus a run of them stands in one column.
  round_trip(
    t,
    "package one\n\nfunc Fold() {\n  const H = 12345 // b\n"+
      "  const I = 2     // c\n}\n",
  )
  // One character stands in one column however many bytes it spans, thus a run of notes
  // reads as one column whatever the lines it stands on spell.
  round_trip(t, "package one\n\nfunc F() {\n  a = \"ééé\" // one\n"+
    "  bb = 1    // two\n}\n")
  round_trip(t, "package one\n\nvar a = []string{\n  \"ééé\", // one\n"+
    "  \"ab\",  // two\n}\n")
  // A list that carries notes stands one element to a line, thus its narrowest line holds a
  // tab, one byte of an element, and the comma that closes it.
  round_trip(t, "package one\n\nvar a = []int{\n  1, // a\n  2, // b\n}\n")
  // An element wider than a measure answers for states the widest width that measure holds,
  // thus the column its notes open at stands at that width.
  wide_element := "package one\n\nvar a = []int{\n  " +
    spelled('W', printer.WIDTH_MAXIMUM) +
    ", // a\n  B, // b\n}\n"
  testify.True(
    t,
    len(printed(t, wide_element)) > printer.WIDTH_MAXIMUM,
    "a list of one wide element prints that element",
  )
  // A field that states no tag closes at its type, thus the run pads no line past what it
  // writes and the form prints as itself.
  round_trip(t, "package one\n\ntype S struct {\n  One int `tag:\"x\"`\n  Two int\n}\n")
  round_trip(
    t,
    "package one\n\ntype S struct {\n  One int `tag:\"x\"`\n  Two int\n\n"+
      "  // a\n}\n",
  )
  // The narrowest key of a run of pairs spans one byte.
  round_trip(t, "package one\n\nvar a = map[int]int{\n  1: 1,\n  2: 2,\n}\n")
  round_trip(t, "package one\n\ntype Entry struct {\n  Value int\n  Mark  int\n}\n")
  round_trip(t, "package one\n\ntype Entry struct {\n  One   int\n  Three int\n"+
    "  Two   int\n}\n")
  round_trip(t, "package one\n\ntype Entry struct {\n  Value int\n\n  Longer int\n}\n")
  round_trip(t, "package one\n\ntype Entry struct {\n  Value int\n}\n")
  alignment_widths(t)
}

// Builds one word of the width it names out of one letter.
// Prints the run of fields a structure of a thousand fields states. A run reads its widest name
// once, thus the print costs the count of the fields and every field aligns on that name.
func test_wide_run(t *testing.T) {
  fields := "package one\n\ntype S struct {\n"
  for index := range 256 {
    fields = fields + "  Name_" + spelled_count(index) + " int\n"
  }
  wide_fields := printed(t, fields+"}\n")
  testify.True(
    t,
    !(offset_of(wide_fields, "  Name_0   int\n") < 0),
    "the narrow name of a wide run aligns on the widest name of that run",
  )
  testify.True(
    t,
    !(offset_of(wide_fields, " Name_255 int\n") < 0),
    "the widest name of a run states the column the run aligns at",
  )
}

// Reads the widest line one form holds, counting one tab as the columns a reader sees.
func widest_line(form string) (widest int) {
  held := 0
  for _, value := range []byte(form) {
    if value == '\n' {
      held = 0
      continue
    }
    held = held + 1
    if value == '\t' {
      held = held + printer.TAB_WIDTH - 1
    }
    if held > widest {
      widest = held
    }
  }
  return widest
}

// Spells one count in the decimal digits it states.
func spelled_count(value int) (word string) {
  if value == 0 {
    return "0"
  }
  text := ""
  for held := value; held > 0; held = held / 10 {
    text = string(rune('0'+held%10)) + text
  }
  return text
}

func spelled(letter byte, width_size int) (word string) {
  text := make([]byte, width_size)
  for index := range text {
    text[index] = letter
  }
  return string(text)
}

// Reads the byte offset one text first holds another at, or -1 where it holds none.
func offset_of(text string, marker string) (at_offset int) {
  for start := 0; !(start+len(marker) > len(text)); start++ {
    if text[start:start+len(marker)] == marker {
      return start
    }
  }
  return -1
}

// States that every line of one form that holds the marker holds it at one column.
func aligned(t *testing.T, form string, marker string) {
  t.Helper()
  column := -1
  for len(form) > 0 {
    end := offset_of(form, "\n")
    if end < 0 {
      end = len(form)
    }
    at_offset := offset_of(form[:end], marker)
    form = form[min(end+1, len(form)):]
    if at_offset < 0 {
      continue
    }
    if column < 0 {
      column = at_offset
    }
    testify.Equal(t, column, at_offset, "the marker %q stands in one column", marker)
  }
}

// Aligns runs whose parts span one column, two, and the widest width a measure answers with.
func alignment_widths(t *testing.T) {
  wide := printer.WIDTH_MAXIMUM
  fields := "package one\n\ntype Wide struct {\n  " +
    spelled('W', wide) +
    " int `one`\n" +
    "  A X `two`\n  AB XY `three`\n  B " +
    spelled('T', wide) +
    " `four`\n}\n"
  form := printed(t, fields)
  aligned(t, form, "  int")
  aligned(t, form, "`")
  for _, width_size := range []int{1, 2} {
    narrow := "package one\n\ntype Narrow struct {\n  " +
      spelled('A', width_size) +
      " " +
      spelled('X', width_size) +
      " `one`\n  " +
      spelled('B', width_size) +
      " " +
      spelled('Y', width_size) +
      " `two`\n}\n"
    narrow_form := printed(t, narrow)
    aligned(t, narrow_form, "  "+spelled('X', width_size))
    aligned(t, narrow_form, "`")
  }
  keys := "package one\n\nvar table = map[string]int{\n  " +
    spelled('w', wide) +
    ": 1,\n  a: 1,\n  ab: 1,\n}\n"
  // A pair that runs past the widest line stands in no column, thus the narrow keys align
  // among themselves.
  testify.True(
    t,
    !(offset_of(printed(t, keys), "  a:  1,\n  ab: 1,\n") < 0),
    "the narrow keys align apart from the wide one",
  )
  // A field whose type the parser refused writes a type of no width, and the tag behind it
  // still stands a space apart.
  for _, text := range []string{
    "package one\n\ntype Refused struct {\n  A ",
    "package one\n\ntype Refused struct {\n  A @ `tag`\n}\n",
    "package one\n\ntype Refused struct {\n  A, B\n}\n",
    "package one\n\ntype Refused struct {\n  A, B",
    "package one\n\ntype Refused struct {\n  A,",
    "package one\n\ntype Refused struct {\n  A int `one`\n  B @ `two`\n}\n",
  } {
    refused := parse_state()
    source := token.Source(text)
    ast.Parse(refused, source)
    storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
    result := printer.Print(printer_state(), storage, refused, source)
    testify.True(t, !(int(result.Count) > len(storage)), "a refused field prints")
  }
  // A note behind a closed literal reads the two columns the brackets spend,
  // and a note behind a line past the widest width reads that width.
  round_trip(t, "package one\n\nvar fold = f(func() {\n}) // Fold folds nothing.\n")
  round_trip(t, "package one\n\nfunc Fold() {\n  "+spelled('w', wide)+"() // Wide.\n}\n")
}

// Prints the notes the author left behind a bracket and the runs of notes a statement closes.
// Prints the empty lines and the feeds the author left at the edges of one file.
func test_file_edges(t *testing.T) {
  // A file opens on the first byte a declaration writes and closes on one line feed, thus
  // the empty lines ahead of the clause and the feeds behind the last declaration state
  // nothing of the form.
  testify.Equal(
    t,
    "package one\n",
    printed(t, "\n\n\npackage one\n"),
    "a file opens on the first byte a declaration writes",
  )
  testify.Equal(
    t,
    "package one\n\nvar a = 1\n",
    printed(t, "package one\n\nvar a = 1"),
    "a file closes on one line feed",
  )
  testify.Equal(
    t,
    "package one\n",
    printed(t, "package one\n\n\n\n"),
    "a file states no feed behind the one it closes on",
  )
}

func test_noted_runs(t *testing.T) {
  // A note the author left behind the bracket that closes a broken form stands on the line
  // that bracket closes.
  round_trip(t, "package one\n\nvar held = g(\n  2,\n) // two\n")
  test_file_edges(t)
  // The clause that names the package carries its note on the line it closes.
  round_trip(t, "package one // note\n")
  round_trip(t, "package one\n\nfunc Fold() {\n} // one\n\nfunc Hold() {\n} // two\n")
  round_trip(t, "package one // one\n\n// two\nvar a = 1\n")
  // A statement that holds no part of its own carries its note on the line it closes.
  prints_refused(
    t,
    "package one\n\nfunc Fold() {\n  switch x {\n  case 1:\n"+
      "    fallthrough // note\n  case 2:\n  }\n}\n",
    "fallthrough // note",
  )
  // A note the author left between the sign and the value states nothing of the binding,
  // thus the declaration states the sign it binds with.
  testify.Equal(
    t,
    "package one\n\nvar held = g(one, two) /* note */\n",
    printed(t, "package one\n\nvar held = /* note */ g(one, two)\n"),
    "a note behind the sign states nothing of the binding",
  )
  // A note the author left among the names one declaration binds states nothing of the run
  // the commas part, thus the names stand as the author bound them.
  testify.Equal(
    t,
    "package one\n\nvar a, b = 1, 2 /* one */\n",
    printed(t, "package one\n\nvar a /* one */, b = 1, 2\n"),
    "a note among the names states nothing of the run",
  )
  testify.Equal(
    t,
    "package one\n\nvar a, b = 1, 2 /* one */\n",
    printed(t, "package one\n\nvar a, /* one */ b = 1, 2\n"),
    "a note behind the comma states nothing of the run",
  )
  // A part that states nothing writes no byte on the line it opens, thus the form that
  // holds it closes its bracket where that part stood and writes no comma for it.
  testify.Equal(
    t,
    "package one\n\nfunc Fold(one,\n)\n",
    printed(t, "package one\n\nfunc Fold(one,\n  ,\n)\n"),
    "a part that states nothing takes no comma",
  )
  // A note the author closed at the line swallows the bytes behind it, thus a note standing
  // behind one on a line opens a line of its own.
  testify.Equal(
    t,
    "package one\n\nfunc Fold(\n  one int, // one\n  // two\n) (\n  two int,"+
      "\n) {\n  return two\n}\n",
    printed(
      t,
      "package one\n\nfunc Fold(\n  one int, // one\n) ( // two\n  two int,"+
        "\n) {\n  return two\n}\n",
    ),
    "a note behind a note of its line opens a line of its own",
  )
  // A note the author left on the line the bracket of a function type closes keeps that
  // line.
  round_trip(t, "package one\n\ntype F func(\n  one int, // one\n) // two\n")
  // A note the author left on the line one result of a signature closes keeps that line.
  round_trip(
    t,
    "package one\n\nfunc Fold() (\n  one int, // one\n  two int, // two\n)"+
      " {\n  return one, two\n}\n",
  )
  round_trip(t, "package one\n\nfunc Fold() {\n  if x {\n    a = 1\n  } // note\n}\n")
  round_trip(t, "package one\n\nfunc Fold() {\n  if x {\n  } // one\n  if y {\n"+
    "  } // two\n}\n")
  round_trip(t, "package one\n\nfunc Fold() {\n  g(\n    1,\n  ) // one\n"+
    "  bb = 2 // two\n}\n")
  round_trip(
    t,
    "package one\n\ntype S struct {\n  A int\n} // one\n\ntype T struct {\n"+
      "  B int\n} // two\n",
  )
  round_trip(t, "package one\n\ntype S struct {\n  A int\n} // note\n")
  // A field that binds no name carries the tag the author wrote behind the type it embeds.
  round_trip(t, "package one\n\ntype S struct {\n  Pair `json:\"pair\"`\n}\n")
  round_trip(t, "package one\n\ntype S struct {\n  Pair // note\n  A int\n}\n")
  round_trip(t, "package one\n\ntype S struct {\n  *Pair\n  A int `one:\"1\"`\n}\n")
  round_trip(t, "package one\n\ntype R interface {\n  int\n} // note\n")
  round_trip(t, "package one\n\nvar held = []int{\n  1,\n} // note\n")
  test_inner_notes(t)
}

// Prints the notes the author left on lines of their own inside a form.
func test_inner_notes(t *testing.T) {
  // A note the author left on a line of its own inside a form stands at the tabs the parts
  // of that form stand at.
  round_trip(t, "package one\n\nvar held = S{\n  Left: 1,\n  // note\n}\n")
  round_trip(t, "package one\n\nvar held = g(\n  1,\n  // note\n)\n")
  round_trip(t, "package one\n\nfunc Fold(\n  one int,\n  // note\n) {\n}\n")
  round_trip(t, "package one\n\nvar held = g(\n  1,\n\n  // note\n)\n")
  // A literal the author wrote over lines of its own closes the run of notes around it.
  round_trip(t, "package one\n\nfunc Fold() {\n  a = `one\ntwo` // note\n"+
    "  bb = 2 // two\n}\n")
  round_trip(t, "package one\n\nvar held = g(\n  h(\n    1,\n  ), // one\n) // two\n")
  round_trip(t, "package one\n\nfunc Fold() {\n} // note\n")
  round_trip(t, "package one\n\nfunc Fold() { a = 1 } // note\n")
  round_trip(t, "package one\n\ntype S struct{ A int } // note\n")
  round_trip(t, "package one\n\ntype S struct {\n} // note\n")
  round_trip(t, "package one\n\nfunc Fold() {\n  bb = g(\n    2,\n  ) // two\n}\n")
  // A line that carries two notes states each of them as a node of its own, thus the run
  // of notes closes at the second of them.
  round_trip(
    t,
    "package one\n\nfunc Fold() {\n  if x {\n    a = 1\n  } /* one */"+
      " /* two */\n}\n",
  )
  // An empty line and a note of its own line each close the run of notes around them, thus
  // the statements on either side stand in runs of their own.
  round_trip(t, "package one\n\nfunc Fold() {\n  a = 1 // one\n\n  cccc = 3 // three\n}\n")
  round_trip(
    t,
    "package one\n\nfunc Fold() {\n  a = 1 // one\n  // note\n"+
      "  cccc = 3 // three\n}\n",
  )
  // A field that runs over lines closes the run of names around it, thus the fields on
  // either side of it stand in runs of their own.
  round_trip(
    t,
    "package one\n\ntype S struct {\n  A int\n  Bbbb func(\n    one int,\n  )"+
      "\n  C int\n}\n",
  )
  // A statement that runs over lines closes the run of notes around it, thus the statements
  // on either side of it stand in runs of their own.
  round_trip(
    t,
    "package one\n\nfunc Fold() {\n  a = 1 // one\n  bb = g(\n    2,\n  )"+
      " // two\n  cccc = 3 // three\n}\n",
  )
}

// Prints one form carrying a note on each of its lines, and one holding an empty line ahead of
// each. A note and an empty line each stand where the author wrote them, thus every line of one
// form answers for both.
// Prints one call and one literal at the widths that stand on either side of the widest line a
// reader reads, and one note standing at each space of one form. A form inside that width stands
// on one line and one past it states one part to a line, thus both stand in the corpus.
func test_width_places(t *testing.T) {
  name := "aaaaaaaaaaaaaaaaaaaa"
  for count := range 3 {
    held := name
    for range count * 4 {
      held = held + "a"
    }
    printed(t, "package one\n\nvar a = g("+held+", "+held+", "+held+")\n")
    printed(t, "package one\n\nvar a = []int{"+held+", "+held+", "+held+"}\n")
  }
  held := "var a = g(one, two)"
  for index := range len(held) {
    if held[index] != ' ' {
      continue
    }
    printed(t, "package one\n\n"+held[:index]+"  /* note */ "+held[index+1:]+"\n")
  }
}

func test_trivia_places(t *testing.T) {
  lines := []string{
    "func Fold() {", " a = 1", " if x {", "  b = 2", " }", " c = 3", "}",
  }
  for index := range lines {
    shaped := append([]string{}, lines...)
    shaped[index] = shaped[index] + " // note"
    printed(t, "package one\n\n"+joined(shaped))
  }
  for place := range len(lines) + 1 {
    shaped := append([]string{}, lines[:place]...)
    shaped = append(shaped, "")
    shaped = append(shaped, lines[place:]...)
    printed(t, "package one\n\n"+joined(shaped))
  }
}

// A note the author left between the receiver one method binds and the name that method states
// stands on the line the method opens, thus the form holds it.
func test_receiver_note(t *testing.T) {
  printed(t, "package one\n\nfunc (o *T)/*note*/ Fold() {\n}\n")
  printed(t, "package one\n\nfunc (o *T) /*note*/ Fold() {\n}\n")
  printed(t, "package one\n\nfunc (o *T) Fold/*note*/() {\n}\n")
}

// A line that already carries a note stands past the column the run it stands in opens at, thus
// the note behind that one stands one space behind the byte it follows.
func test_two_notes(t *testing.T) {
  printed(t, "package one\n\nfunc Fold() {\n  x = 1 // a\n  y = 2 /*b*/ // c\n}\n")
  printed(t, "package one\n\nfunc Fold() {\n  return/*a*/ five/*b*/\n}\n")
}

// A note or an empty line the author left inside the head of an import states neither the name
// that import binds nor the path it states, thus the print reads both past the trivia.
func test_import_notes(t *testing.T) {
  // A path closing on the byte that parts its elements states an element of no byte,
  // which names no package, thus the name the forms read is the element ahead of it.
  round_trip(t, "package one\n\nimport \"local/example/two/\"\n\n"+
    "func Fold() {\n  two.G()\n}\n")
  round_trip(t, "package one\n\nimport \"local/example/two//\"\n\n"+
    "func Fold() {\n  two.G()\n}\n")
  // A name the path itself states writes nothing of its own, and the forms that read it
  // name the package the path states.
  testify.Equal(
    t,
    "package one\n\nimport \"local/example/two\" /* note */\n\n"+
      "func Fold() {\n  two.G()\n}\n",
    printed(
      t,
      "package one\n\nimport three/* note */ \"local/example/two\"\n\n"+
        "func Fold() {\n  three.G()\n}\n",
    ),
    "a noted import drops the name its path states",
  )
  // Dropping the name would leave two packages of one name, thus the import keeps it.
  testify.Equal(
    t,
    "package one\n\nimport three \"local/example/two\" /* note */\n"+
      "import \"other/two\"\n\nfunc Fold() {\n  three.G()\n  two.H()\n}\n",
    printed(
      t,
      "package one\n\nimport three/* note */ \"local/example/two\"\n"+
        "import \"other/two\"\n\nfunc Fold() {\n  three.G()\n  two.H()\n}\n",
    ),
    "a noted import keeps the name two packages would share",
  )
}

// Joins the lines of one form, each closing on the feed it takes.
func joined(lines []string) (form string) {
  for _, line := range lines {
    form = form + line + "\n"
  }
  return form
}

func test_trivia(t *testing.T) {
  test_notes_ahead(t)
  test_trivia_places(t)
  test_receiver_note(t)
  test_tab_notes(t)
  test_return_notes(t)
  test_note_shapes(t)
  test_import_notes(t)
  test_two_notes(t)
  test_width_places(t)
  // A call the print breaks at its arguments carries its note inside the brackets, thus the
  // statement behind it opens a run of its own and pads no line to the width of the call.
  broken := "package one\n\nfunc Fold() {\n  g(" +
    "Naaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa, Naaaaaaaaaaaaaaaaaaaaaaaaa," +
    " Naaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa) // note\n  g(Na, Naaaaaaaaaaaa, Naaaaa)" +
    " // note\n}\n"
  broken_form := printed(t, broken)
  testify.Equal(
    t,
    broken_form,
    printed(t, broken_form),
    "a statement behind a broken call prints as itself",
  )
  // A statement the print breaks across lines carries its note where that note falls, thus
  // the statements around it stand in a column of their own and the form prints as itself.
  statement := " x := a"
  for range 24 {
    statement = statement + "+a"
  }
  wide := "package one\n\nfunc Fold() {\n" + statement + "  // a\n  g(1) // b\n}\n"
  shaped := printed(t, wide)
  testify.Equal(
    t,
    shaped,
    printed(t, shaped),
    "a broken statement carrying a note prints as itself",
  )
  // An empty line the author left inside a struct stands where the author left it, and only
  // the one standing against the brace that closes the struct closes up.
  round_trip(t, "package one\n\ntype S struct {\n  // b\n\n  One int\n}\n")
  round_trip(t, "package one\n\ntype S struct {\n  One int\n\n  // b\n  Two int\n}\n")
  test_noted_runs(t)
  // A run of notes opens behind its widest line, which the shortest statement spans at a tab
  // and one byte.
  round_trip(t, "package one\n\nfunc Fold() {\n  x // x\n  x // y\n}\n")
  // The comments a line carries inside its forms stand at its end, in the order the author
  // wrote them, up to as many as the storage holds.
  for _, count := range []int{2, printer.COMMENT_COUNT_MAXIMUM} {
    source := "package one\n\nvar a = f(1"
    comments := ""
    for index := range count {
      note := " /* " + string([]byte{byte('a' + index)}) + " */"
      source = source + ", 1" + note
      comments = comments + note
    }
    form := printed(t, source+")\n")
    testify.True(
      t,
      !(offset_of(form, ")"+comments+"\n") < 0),
      "the comments a line carries stand at its end",
    )
  }
  // A note the author closed up against the slashes reads as one word, thus the canonical
  // form opens a space behind them and leaves the directives a tool reads untouched.
  round_trip(t, "package one\n\n// Count names a count.\ntype Count int\n")
  testify.Equal(
    t,
    "package one\n\n// Count names a count.\ntype Count int\n",
    printed(t, "package one\n\n//Count names a count.\ntype Count int\n"),
    "a note opens with a space behind the slashes",
  )
  round_trip(t, "package one\n\n//go:generate echo\ntype Count int\n")
  round_trip(t, "package one\n\n//nolint:all\ntype Count int\n")
  round_trip(t, "package one\n\n//\ntype Count int\n")
  round_trip(t, "package one\n\n//line one.go:1\ntype Count int\n")
  round_trip(t, "package one\n\n//export One\ntype Count int\n")
  round_trip(t, "package one\n\n//nolint\ntype Count int\n")
  round_trip(t, "package one\n\n// Count names a count.\ntype Count int\n")
  round_trip(t, "package one\n\ntype Count int\n\ntype Mark int\n")
  round_trip(t, "package one\n\nfunc Fold() {\n  // One states nothing.\n  return\n}\n")
  round_trip(t, "package one\n\n/* Count names a count. */\ntype Count int\n")
}

func test_crowded_lines(t *testing.T) {
  // A line that carries more comments than the storage holds would lose one, thus the print
  // refuses it rather than dropping what the author wrote.
  crowded := "package one\n\nvar a = f(1"
  for index := range printer.COMMENT_COUNT_MAXIMUM + 1 {
    crowded = crowded + ", 1 /* " + string([]byte{byte('a' + index)}) + " */"
  }
  source := token.Source(crowded + ")\n")
  tree := parse_state()
  ast.Parse(tree, source)
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  refused := printer.Print(printer_state(), storage, tree, source)
  testify.False(t, bool(refused.OK), "a line past the comment storage is refused")
  // A line that carries what the storage holds keeps every comment the author wrote.
  full := "package one\n\nvar a = f(1"
  for index := range printer.COMMENT_COUNT_MAXIMUM {
    full = full + ", 1 /* " + string([]byte{byte('a' + index)}) + " */"
  }
  printed(t, full+")\n")
}

// States the empty lines the canonical form drops and the ones it keeps.
func test_lines(t *testing.T) {
  test_inline_bodies(t)
  test_lines_checked(t)
  // A body opens at its first statement and closes at its last one, thus the empty lines
  // the author left against the braces state nothing and the canonical form drops them.
  test_lines_against(t)
}

// Prints the bodies the author wrote on one line.
func test_inline_bodies(t *testing.T) {
  // A clause closes the line its body closes on, thus a body that holds one stands over
  // lines of its own however the author wrote it.
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  if x { a = 1 }\n}\n",
    printed(t, "package one\nfunc Fold(){if x{a=1}}\n"),
    "a body that holds a clause stands over lines of its own",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  for range 4 { a = 1 }\n}\n",
    printed(t, "package one\nfunc Fold(){for range 4{a=1}}\n"),
    "a body that holds a loop stands over lines of its own",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  if x { a = 1 } else { a = 2 }\n}\n",
    printed(t, "package one\nfunc Fold(){if x{a=1}else{a=2}}\n"),
    "a body that holds a branch stands over lines of its own",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\nLoop:\n  for range 4 { break Loop }\n}\n",
    printed(t, "package one\nfunc Fold(){Loop: for range 4{break Loop}}\n"),
    "a body that holds a label stands over lines of its own",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  type T int\n  var b T\n  _ = b\n}\n",
    printed(t, "package one\nfunc Fold(){type T int;var b T;_=b}\n"),
    "a body that holds a declaration stands over lines of its own",
  )
  // A body of statements that close no line of their own stands on the line it opens on.
  round_trip(t, "package one\n\nfunc Fold() { go g(); defer h() }\n")
  round_trip(t, "package one\n\nfunc Fold() { func() { a = 1 }() }\n")
}

// Prints the empty lines the author left against the braces of a body.
func test_lines_against(t *testing.T) {
  // An empty line the author left inside the form a statement states still stands against
  // the brace the body closes on, thus a body that closes up drops it.
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  g()\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  g()\n\n}\n"),
    "a body that closes up drops the line a statement carried against its brace",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  if x {\n    g()\n  }\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  if x {\n\n    g()\n\n  }\n}\n"),
    "a body of one statement drops the lines against both its braces",
  )
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {
      Source: "package one\n\nfunc Fold() {\n\n  return\n}\n",
      Form:   "package one\n\nfunc Fold() {\n  return\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  if x {\n\n    a = 1" +
        "\n  } /* b */\n}\n",
      Form: "package one\n\nfunc Fold() {\n  if x {\n    a = 1" +
        "\n  } /* b */\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  if x {\n\n    a = 1" +
        "\n  }/* b */ else {\n    a = 2\n  }\n}\n",
      Form: "package one\n\nfunc Fold() {\n  if x {\n    a = 1" +
        "\n  } else { /* b */\n    a = 2\n  }\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  if x {\n    a = 1\n  }\n\n}\n",
      Form:   "package one\n\nfunc Fold() {\n  if x {\n    a = 1\n  }\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  for range 4 {\n  }\n\n}\n",
      Form:   "package one\n\nfunc Fold() {\n  for range 4 {\n  }\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  if x {\n    if y {" +
        "\n    }\n\n  }\n}\n",
      Form: "package one\n\nfunc Fold() {\n  if x {\n    if y {" +
        "\n    }\n  }\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  for i := 0;\n\n    i < 4; i++ {" +
        "\n  }\n}\n",
      Form: "package one\n\nfunc Fold() {\n  for i := 0; i < 4; i++ {\n  }\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  return\n\n}\n",
      Form:   "package one\n\nfunc Fold() {\n  return\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold(held bool) {\n  if held {\n" +
        "\n    return\n\n  }\n}\n",
      Form: "package one\n\nfunc Fold(held bool) {\n  if held {\n" +
        "    return\n  }\n}\n",
    },
  } {
    testify.Equal(
      t,
      one.Form,
      printed(t, one.Source),
      "a body states no empty line against the braces that hold it",
    )
  }
  test_lines_apart(t)
}

// Prints the empty lines the author left against the brackets of a form.
func test_lines_apart(t *testing.T) {
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {
      Source: "package one\n\ntype Pair struct {\n\n  Left int\n\n}\n",
      Form:   "package one\n\ntype Pair struct {\n  Left int\n}\n",
    },
    {
      Source: "package one\n\nvar table = []int{\n\n  1,\n  2,\n\n}\n",
      Form:   "package one\n\nvar table = []int{\n  1,\n  2,\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() {\n  total :=\n\n    1 + 1\n" +
        "  _ = total\n}\n",
      Form: "package one\n\nfunc Fold() {\n  total := 1 + 1\n" +
        "  _ = total\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() (err error) {\n  err = Step()\n\n" +
        "  if err != nil {\n    return err\n  }\n  return err\n}\n",
      Form: "package one\n\nfunc Fold() (err error) {\n  err = Step()\n" +
        "  if err != nil {\n    return err\n  }\n  return err\n}\n",
    },
    {
      Source: "package one\n\nfunc Fold() (err error) {\n" +
        "  value, err := Step()\n\n  if err != nil {\n" +
        "    return err\n  }\n  _ = value\n  return err\n}\n",
      Form: "package one\n\nfunc Fold() (err error) {\n" +
        "  value, err := Step()\n  if err != nil {\n" +
        "    return err\n  }\n  _ = value\n  return err\n}\n",
    },
  } {
    testify.Equal(
      t,
      one.Form,
      printed(t, one.Source),
      "a body states no empty line against the braces that hold it",
    )
  }
}

// Prints the empty lines the author left ahead of a check of an error.
func test_lines_checked(t *testing.T) {
  // A check the author wrote against anything else states no simple check of an error, thus
  // the empty line ahead of it stands where the author put it.
  round_trip(
    t,
    "package one\n\nfunc Fold() (err error) {\n  err = Step()\n\n"+
      "  if err != nil {\n    return err\n  } else {\n    return nil\n  }\n}\n",
  )
  round_trip(
    t,
    "package one\n\nfunc Fold() (err error) {\n  err = Step()\n\n"+
      "  if err == nil {\n    return err\n  }\n  return err\n}\n",
  )
  round_trip(
    t,
    "package one\n\nfunc Fold() (err error) {\n  value := Step()\n\n"+
      "  if err != nil {\n    return err\n  }\n  _ = value\n  return err\n}\n",
  )
}

func test_refusals(t *testing.T) {
  test_storage_widths(t)
  test_arena(t)
  test_hostile(t)
  test_held_spread(t)
  test_stood_tree(t)
  // A source the author left open closes its forms at the end of the file, thus every
  // read the print takes of the run stands inside the tokens the scan wrote.
  texts := []string{
    "package one\n\nfunc F() {\n  g(a, @)\n}\n",
    "package one\n\nvar A = [2]int{1, @}\n",
    "package one\n\nfunc F(a @) {\n}\n",
    "package one\n\nvar a = f(1\n",
    "package one\n\nvar held = a[1:g(2)...*g(3)...]\n",
    "package one\n\nvar held = a[]\n",
    "package one\ntype S struct {\n\n",
    "package one\n\nfunc Fold() {\n  for index := range S{A: 1} {\n    break\n  }\n}\n",
    "package one\n\nfunc f() {\n  switch a {\n  case 1\n  }\n}\n",
    "package one\n\nfunc f() {\n  switch a {\n  case (1\n",
    "package one\ntype S struct {\n  // a\n",
    "package one\n\nvar held = a[1:2:3:4]\n",
    "package one\n\nvar a = f(f(1)\n",
    "package one\n\nvar a = []int{1\n",
    "package one\n\nfunc f() {\n",
  }
  for _, text := range append(texts, refused_sources()...) {
    refused := parse_state()
    source := token.Source(text)
    ast.Parse(refused, source)
    storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
    result := printer.Print(printer_state(), storage, refused, source)
    testify.True(t, !(int(result.Count) > len(storage)),
      "a refused tree prints inside storage")
  }
  subject := printer_state()
  tree := parse_state()
  source := token.Source("package one\n\ntype Count int\n")
  ast.Parse(tree, source)
  storage := make([]byte, 4)
  result := printer.Print(subject, storage, tree, source)
  testify.False(t, bool(result.OK), "a print past the storage is refused")
  testify.Equal(
    t,
    printer.Form_Count(len(storage)),
    result.Count,
    "a refused print states the storage it filled",
  )
  // A caller that hands over storage of a few bytes reads the few bytes the print wrote,
  // which is what a caller that only asks whether a file is clean hands over.
  for _, width_size := range []int{0, 1, 2} {
    narrow := make([]byte, width_size)
    narrow_result := printer.Print(subject, narrow, tree, source)
    testify.False(t, bool(narrow_result.OK), "a print past narrow storage is refused")
    testify.Equal(
      t,
      printer.Form_Count(width_size),
      narrow_result.Count,
      "a refused print states the narrow storage it filled",
    )
  }
  // A source of a few bytes states no file at all, thus the print writes nothing and says
  // so rather than reading a token the source never held.
  for _, text := range []string{"", "p", "pa"} {
    narrow := token.Source(text)
    ast.Parse(tree, narrow)
    wide_storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
    narrow_result := printer.Print(subject, wide_storage, tree, narrow)
    testify.True(t, bool(narrow_result.OK),
      "a print of a source of a few bytes holds its form")
    testify.True(
      t,
      !(narrow_result.Count > printer.Form_Count(len(text)+9)),
      "a source of a few bytes writes the clause those bytes name and no more",
    )
  }
  broken := token.Source("package one\n\nfunc (\n")
  ast.Parse(tree, broken)
  wide := make([]byte, printer.FORM_SIZE_MAXIMUM)
  result = printer.Print(subject, wide, tree, broken)
  testify.True(t, bool(result.OK), "a refused parse prints the tree it holds")
}

// Builds one source of the widest admitted size whose form runs past the widest admitted form,
// which is a body of short statements standing deep inside blocks that indent every one of them.
func wide_source() (source string) {
  body := make([]byte, 0, token.SOURCE_SIZE_MAXIMUM)
  body = append(body, "package one\n\nfunc Fold() {\n"...)
  for range 100 {
    body = append(body, "if true {\n"...)
  }
  // A note the print indents to the depth it stands at grows by that depth, thus a run
  // of short notes inside the deepest block writes a form wider than the storage. One
  // note is one token, thus the run stands well inside the token run one parse admits.
  for range 21000 {
    body = append(body, "//\n"...)
  }
  for range 100 {
    body = append(body, "}\n"...)
  }
  body = append(body, "}\n"...)
  // A declaration behind the form that filled the storage opens where the storage ends.
  body = append(body, "\nfunc Other() {\n}\n"...)
  // A space states no token and closes no line, thus the padding that carries the source to
  // the widest admitted size costs the parse nothing.
  for len(body) < token.SOURCE_SIZE_MAXIMUM {
    body = append(body, ' ')
  }
  return string(body)
}

func test_bounds(t *testing.T) {
  testify.Equal(t, 2097152, printer.FORM_SIZE_MAXIMUM, "one print holds two source widths")
  test_program(t)
  testify.Equal(
    t,
    printer.LAYOUT_STEP_COUNT,
    len(printer.LAYOUT_PROGRAM)/printer.LAYOUT_STEP_SIZE,
    "the step count names every step",
  )
  // The narrowest file opens its first declaration right behind its package clause.
  round_trip(t, "package a\n\nimport \"z\"\n")
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  // A source of the widest admitted size whose form runs past the storage fills the storage
  // and stops there, which is the widest count one print states.
  source := token.Source(wide_source())
  testify.Equal(t, token.SOURCE_SIZE_MAXIMUM, len(source),
    "the source states the widest size")
  ast.Parse(tree, source)
  result := printer.Print(subject, storage, tree, source)
  testify.False(t, bool(result.OK), "a form past the storage is refused")
  testify.Equal(
    t,
    printer.Form_Count(printer.FORM_SIZE_MAXIMUM),
    result.Count,
    "a refused print states the storage it filled",
  )
  testify.Equal(t, 4096, printer.DEPTH_MAXIMUM, "one print walks a tree 4,096 nodes deep")
  // A run of one sign nests one value inside the next, thus a chain of a thousand values
  // nests a thousand levels deep and every one of them stands inside the walk.
  // The chain runs past the widest line, thus its canonical form states one value to a
  // line and the round trip states that form rather than the one line the chain would be.
  deep := "package one\n\nconst TEXT = \"one\""
  for range 1024 {
    deep = deep + " +\n  \"one\""
  }
  round_trip(t, deep+"\n")
  test_deep(t)
  test_wide_run(t)
  // A file of notes alone reads each note once, thus a long run of them prints at once
  // rather than rereading the run for every note it holds.
  notes := ""
  for range 4096 {
    notes = notes + "//\n"
  }
  testify.Equal(
    t,
    "package one\n\n"+notes,
    printed(t, "package one\n"+notes),
    "a run of notes prints as itself",
  )
  // A function that states the invariants of a type stands under that type wherever the
  // author wrote it, and the notes that open it stand with it.
  paired := token.Source(
    "package one\n\n// B states one thing.\nfunc B_Invariants() {\n}" +
      "\n\nfunc A_Invariants() {\n}\n\ntype A int\n\ntype B int\n",
  )
  ast.Parse(tree, paired)
  result = printer.Print(subject, storage, tree, paired)
  testify.Equal(
    t,
    "package one\n\ntype A int\n\nfunc A_Invariants() {\n}\n\ntype B int\n"+
      "\n// B states one thing.\nfunc B_Invariants() {\n}\n",
    string(storage[:result.Count]),
    "a function of invariants stands under the type it names",
  )
  // A body of empty statements that spends every token the run admits closes on the last
  // position a bracket stands at, since the file closes on the semicolon behind that
  // bracket and on its end.
  filled := make([]byte, 0, ast.TOKEN_COUNT_MAXIMUM+64)
  filled = append(filled, "package one\n\nfunc f() {\n"...)
  for range ast.TOKEN_COUNT_MAXIMUM - TEST_FILLED_HEAD_TOKENS - 3 {
    filled = append(filled, ';')
  }
  filled = append(filled, "}\n"...)
  ast.Parse(tree, token.Source(filled))
  result = printer.Print(subject, storage, tree, token.Source(filled))
  testify.True(t, bool(result.OK), "a body that spends the widest run prints")
  testify.Equal(
    t,
    "package one\n\nfunc f() {\n}\n",
    string(storage[:result.Count]),
    "a body of empty statements prints empty",
  )
  test_filled_call(t, subject, tree, storage)
  test_filled_list(t, subject, tree, storage)
}

// Prints one broken list of parameters that spends every token the run admits, whose bracket
// closes on the last position a bracket stands at.
func test_filled_list(
  t *testing.T,
  subject printer.Printer_Handle,
  tree ast.Parse_State_Handle,
  storage []byte,
) {
  spent := make([]byte, 0, ast.TOKEN_COUNT_MAXIMUM+64)
  spent = append(spent, "package one\n\nfunc f() {\n"...)
  for range TEST_FILLED_LIST_FILL {
    spent = append(spent, ';')
  }
  spent = append(spent, "}\n\ntype F func(\n  one int, // a\n)\n"...)
  ast.Parse(tree, token.Source(spent))
  result := printer.Print(subject, storage, tree, token.Source(spent))
  testify.True(t, bool(result.OK), "a list that spends the widest run prints")
  testify.Equal(
    t,
    "package one\n\nfunc f() {\n}\n\ntype F func(\n  one int, // a\n)\n",
    string(storage[:result.Count]),
    "a list that spends the widest run prints as itself",
  )
}

// TEST_FILLED_LIST_FILL is how many empty statements stand ahead of the filled list: the run it
// spends holds the head of eight tokens, the brace that closes the body and the semicolon behind
// it, the eight tokens of the declaration and its parameter, the note that parameter carries,
// that bracket, the semicolon behind it, and the end of the file.
const TEST_FILLED_LIST_FILL = ast.TOKEN_COUNT_MAXIMUM - TEST_FILLED_HEAD_TOKENS - 13

// Prints one broken call that spends every token the run admits, whose bracket closes on the
// last position a bracket stands at.
func test_filled_call(
  t *testing.T,
  subject printer.Printer_Handle,
  tree ast.Parse_State_Handle,
  storage []byte,
) {
  spent := make([]byte, 0, ast.TOKEN_COUNT_MAXIMUM+64)
  spent = append(spent, "package one\n\nfunc f() {\n"...)
  for range TEST_FILLED_CALL_FILL {
    spent = append(spent, ';')
  }
  spent = append(spent, "}\n\nvar a = g(\n  1, // a\n)\n"...)
  ast.Parse(tree, token.Source(spent))
  result := printer.Print(subject, storage, tree, token.Source(spent))
  testify.True(t, bool(result.OK), "a call that spends the widest run prints")
  testify.Equal(
    t,
    "package one\n\nfunc f() {\n}\n\nvar a = g(\n  1, // a\n)\n",
    string(storage[:result.Count]),
    "a call that spends the widest run prints as itself",
  )
}

// TEST_FILLED_CALL_FILL is how many empty statements stand ahead of the filled call: the run it
// spends holds the head of eight tokens, the brace that closes the body and the semicolon behind
// it, the eight tokens of the declaration and its bracket, the note the argument carries, that
// bracket, the semicolon behind it, and the end of the file.
const TEST_FILLED_CALL_FILL = ast.TOKEN_COUNT_MAXIMUM - TEST_FILLED_HEAD_TOKENS - 13

func test_allocation(t *testing.T) {
  held := allocation_fixture{}
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  source := token.Source(
    "package one\n\ntype Count int\n\nfunc Fold() (sum Count) {\n" +
      "  return sum\n}\n",
  )
  ast.Parse(tree, source)
  // A file that states every form the printer writes proves the whole of the print
  // allocates nothing, rather than proving it of the few forms one small file holds.
  wide := parse_state()
  wide_form := token.Source(wide_head() + wide_tail())
  ast.Parse(wide, wide_form)
  checks := []struct {
    Name string
    Call func()
  }{
    {Name: "Print", Call: func() {
      allocation_print(&held, printer.Print(subject, storage, tree, source))
    }},
    {Name: "Print_Wide", Call: func() {
      printer.Print(subject, storage, wide, wide_form)
    }},
  }
  for _, check := range checks {
    t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
  }
  testify.True(t, bool(held.Ok), "the allocation fixture prints its file")
}

func allocation_print(held *allocation_fixture, result printer.Print_Result) {
  held.Count, held.Ok = result.Count, result.OK
}

// States the declarations of one source that holds every form the printer writes.
func wide_head() (source string) {
  return "// Package wide states every form the printer writes.\n" +
    "package wide\n\nimport \"errors\"\n\nimport \"strings\"\n" +
    "\n// COUNT names one count.\nconst COUNT = 1\n" +
    "\n// KIND_ONE names one kind.\nconst KIND_ONE = 0\n" +
    "\n// KIND_TWO names another kind.\nconst KIND_TWO = 1\n" +
    "\nvar table [4]int\n\nvar lookup = map[string]int{\"one\": 1, \"" +
    "two\": 2}\n\nvar channel chan int\n\nvar sender chan<- int\n" +
    "\nvar receiver <-chan int\n\nvar pointer *int\n" +
    "\nvar slice []byte\n\nvar maker = make([]byte, 0, 8)\n" +
    "\n// Pair holds two values.\ntype Pair struct {\n" +
    "  Left  int    `json:\"left\"`\n  Right string `json:\"right\"`" +
    "\n\n  Note string\n}\n\n// Reader states one constraint.\n" +
    "type Reader interface {\n  ~int | ~string\n" +
    "}\n\n// Handler names one function type.\n" +
    "type Handler func(one int, two string) (held bool)\n" +
    "\n// Fold folds one pair.\nfunc Fold[Value ~int](pair Pair, valu" +
    "es []Value) (sum int, err error) {\n  sum = pair.Left*2 + 3\n" +
    "  sum = sum + len(values)/2 - 1\n  sum = sum & ^COUNT\n" +
    "  sum = sum + +COUNT\n  sum = sum - -COUNT\n" +
    "  sum = (sum + 1) * 2\n  head, tail := values[0], values[len(val" +
    "ues)-1]\n  _ = head\n  _ = tail\n  slice := values[1:]\n" +
    "  whole := values[:]\n  front := values[:1]\n" +
    "  middle := values[1 : len(values)-1]\n  _, _, _, _ = slice, who" +
    "le, front, middle\n  sum++\n  sum--\n  if sum > 0 {\n" +
    "    sum = sum + 1 // The sum grows.\n  } else if sum == 0 {\n" +
    "    sum = 1 // The sum opens.\n  } else {\n" +
    "    sum = 0\n  }\n  for step := 0; step < 4; step++ {\n" +
    "    sum = sum + step\n  }\n  for sum > 100 {\n" +
    "    sum = sum - 1\n  }\n  for ; sum < 0; sum++ {\n" +
    "  }\n  for step := 0; ; step++ {\n    sum = sum + step\n" +
    "  }\n  for index := range table {\n    sum = sum + index\n" +
    "  }\n  for key, value := range lookup {\n" +
    "    sum = sum + value + len(key)\n  }\n  switch sum {\n" +
    "  case 0, 1:\n    sum = 2\n  case 2:\n    // A note opens this c" +
    "ase.\n    sum = 3\n  default:\n    sum = 4\n" +
    "  }\n  switch held := any(sum); held.(type) {\n" +
    "  case int:\n    sum = 5\n  default:\n    sum = 6\n" +
    "  }\n  select {\n  case value := <-receiver:\n" +
    "    sum = value\n  case sender <- sum:\n" +
    "    sum = 0\n  default:\n    sum = 1\n  }\n" +
    "Loop:\n  for range 4 {\n    for range 2 {\n" +
    "      break Loop\n    }\n  }\n  defer func() { sum = sum + 1 }()" +
    "\n  go Report(sum)\n  Report(sum)\n  Report(values...)\n" +
    "  pair.Note = strings.TrimSpace(pair.Right)\n" +
    "  err = errors.New(\"held\")\n  return sum, err\n" +
    "}\n\n// Report states one count.\nfunc Report(values ...any) {\n" +
    "  _ = values\n}\n\n// Blanked holds one blank line ahead of its " +
    "brace.\ntype Blanked struct {\n  Value int\n" +
    "}\n\nvar noted func() // A note stands behind this one.\n" +
    "\n// Ranked states one constraint of a term the source approxima" +
    "tes and one it states plainly.\ntype Ranked interface {\n" +
    "  ~int | string\n}\n"
}

// States the bodies of one source that holds every form the printer writes.
func wide_tail() (source string) {
  return "// Package wide states every form the printer writes.\n" +
    "package wide\n\nimport \"errors\"\n\n// Spread states one long c" +
    "all the author broke.\nfunc Spread(\n  one int, two int, three i" +
    "nt,\n) (held bool, err error) {\n  held = one+two > three\n" +
    "  pairs := []Pair{\n    {Left: one, Right: \"one\"}, // The firs" +
    "t pair.\n    {Left: two, Right: \"two\"}, // The second pair.\n" +
    "  }\n  table := map[string]Pair{\n    \"one\": pairs[0],\n" +
    "\n    \"two\": pairs[1],\n  }\n  _ = table\n" +
    "  Report(\n    one,\n    two,\n    three,\n" +
    "  )\n  Report(one, two,\n    three)\n  err = errors.New(\n" +
    "    \"held\",\n  )\n  return held, err\n" +
    "}\n\n// Empty holds nothing.\ntype Empty struct{}\n" +
    "\n// Wrapper embeds one pair.\ntype Wrapper struct {\n" +
    "  Pair\n\n  Note string\n}\n\n// Number states one constraint wi" +
    "thout approximation.\ntype Number interface {\n" +
    "  int | ~string\n}\n\nvar handler func()\n" +
    "\nvar one, two = 1, 2\n\n// Count states one bare result.\n" +
    "func Count() int {\n  return one + two\n" +
    "}\n\n// Wide states one signature the author broke behind its fi" +
    "rst parameter.\nfunc Wide(one int,\n  two int,\n) (held bool) {\n" +
    "  held = one < two\n  if one <\n    two {\n" +
    "    held = false\n  }\n  switch one +\n    two {\n" +
    "  case 3:\n  case 4:\n    held = true\n  }\n" +
    "  for range 2 { // A note stands on the brace line.\n" +
    "    held = !held\n  }\n  empty := Empty{}\n" +
    "  _ = empty\n  deep := map[string]Pair{\n" +
    "    \"one\": {\n      Left:  1,\n      Right: \"one\",\n" +
    "    },\n    \"two\": {Left: 2, Right: \"two\"},\n" +
    "  }\n  _ = deep\n  held = one <\n    two\n" +
    "  block := Wrapper{}\n\n  _ = block\n  return held\n" +
    "}\n\n// Plain states one constraint of a single term.\n" +
    "type Plain interface {\n  int\n}\n\n// Close states the forms th" +
    "at close a line of their own.\nfunc Close(pair Pair, one int, tw" +
    "o int) (held bool) {\n  held = one*2 > two\n" +
    "  held = one * -two\n  empties := []Empty{{}}\n" +
    "  _ = empties\n  if one <\n    two {\n    held = true\n" +
    "  } else if two > one {\n    switch one {\n" +
    "    case 5:\n      held = false\n    }\n" +
    "  }\n  held = strings.\n    Contains(pair.Right, \"one\")\n" +
    "  Report(\n    one,\n\n    // A note stands behind an empty line" +
    ".\n    two,\n  )\n  one++\n\n  return held\n" +
    "}\n"
}

// Prints the declarations of one source that states every form the printer writes, which
// binds the whole of the specification to one file rather than to a case for each leaf.
func test_wide_head(t *testing.T) {
  round_trip(t, wide_head())
}

// Prints the bodies of one source that states every form the printer writes.
func test_wide_tail(t *testing.T) {
  round_trip(t, wide_tail())
}

// Writes one tree the parser never writes: a form standing on a token that is no sign of an
// operation, a form holding nothing at all, and a form holding a note alone. A print answers for
// every tree the arena can hold and not only for the trees one parse writes.
func test_arena(t *testing.T) {
  tree := parse_state()
  source := token.Source("package one\n\nx y\n")
  tree.Tokens.Values[0] = token.Token{Kind: token.KIND_PACKAGE, Offset: 0, Size: 7}
  tree.Tokens.Values[1] = token.Token{Kind: token.KIND_IDENTIFIER, Offset: 8, Size: 3}
  tree.Tokens.Values[2] = token.Token{Kind: token.KIND_IDENTIFIER, Offset: 13, Size: 1}
  tree.Tokens.Values[3] = token.Token{Kind: token.KIND_IDENTIFIER, Offset: 15, Size: 1}
  tree.Tokens.Values[4] = token.Token{Kind: token.KIND_END_OF_FILE, Offset: 16, Size: 0}
  tree.Nodes.Values[0] = ast.Node{Kind: ast.NODE_ERROR}
  tree.Nodes.Values[1] = ast.Node{Kind: ast.NODE_FILE, Token: 0, First_Child: 2}
  tree.Nodes.Values[2] = ast.Node{Kind: ast.NODE_IDENTIFIER, Token: 1, Parent: 1, Next: 3}
  // The sign of the operation is a name, which no parse writes and every print answers for.
  tree.Nodes.Values[3] = ast.Node{
    Kind:        ast.NODE_BINARY,
    Token:       3,
    Parent:      1,
    First_Child: 4,
    Next:        6,
  }
  tree.Nodes.Values[4] = ast.Node{Kind: ast.NODE_IDENTIFIER, Token: 2, Parent: 3, Next: 5}
  tree.Nodes.Values[5] = ast.Node{Kind: ast.NODE_IDENTIFIER, Token: 3, Parent: 3}
  tree.Nodes.Values[6] = ast.Node{Kind: ast.NODE_CALL, Token: 2, Parent: 1, Next: 7}
  tree.Nodes.Values[7] = ast.Node{Kind: ast.NODE_CALL, Token: 2, Parent: 1, First_Child: 8}
  tree.Nodes.Values[8] = ast.Node{Kind: ast.NODE_COMMENT, Token: 2, Parent: 7}
  // The term states no tilde ahead of it, which no parse writes and every print answers for.
  tree.Nodes.Values[7].Next = 9
  tree.Nodes.Values[9] = ast.Node{Kind: ast.NODE_TERM, Token: 2, Parent: 1, First_Child: 10}
  tree.Nodes.Values[10] = ast.Node{Kind: ast.NODE_IDENTIFIER, Token: 2, Parent: 9}
  subject := printer_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  result := printer.Print(subject, storage, tree, source)
  testify.True(t, bool(result.OK), "a print of a hand written tree holds its form")
  testify.True(t, result.Count > 0, "a print of a hand written tree writes the form it holds")
}

// Prints one constant of the stated literal, which is how each case states the form one literal
// takes.
func literal_form(t *testing.T, literal string) (form string) {
  t.Helper()
  return printed(t, "package one\n\nconst VALUE = "+literal+"\n")
}

func test_literals(t *testing.T) {
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {Source: "0XFF", Form: "0xFF"},
    {Source: "0xff", Form: "0xff"},
    {Source: "0X_FF", Form: "0x_FF"},
    {Source: "0B1010", Form: "0b1010"},
    {Source: "0O17", Form: "0o17"},
    {Source: "07", Form: "0o7"},
    {Source: "0755", Form: "0o755"},
    {Source: "00", Form: "0o0"},
    {Source: "0_755", Form: "0o_755"},
    {Source: "0", Form: "0"},
    {Source: "0.5", Form: "0.5"},
    {Source: ".5", Form: ".5"},
    {Source: ".5E10", Form: ".5e10"},
    {Source: ".5i", Form: ".5i"},
    {Source: "`a\rb`", Form: "`ab`"},
    {Source: "`a\r\nb`", Form: "`a\nb`"},
    {Source: "1E6", Form: "1e6"},
    {Source: "0e1", Form: "0e1"},
    {Source: "0E1", Form: "0e1"},
    {Source: "00e1", Form: "00e1"},
    {Source: "0755e1", Form: "0755e1"},
    {Source: "0.5e-3", Form: "0.5e-3"},
    {Source: "0.5E-3", Form: "0.5e-3"},
    {Source: "0X1P-2", Form: "0x1p-2"},
    {Source: "0x1p-2", Form: "0x1p-2"},
    {Source: "0765i", Form: "765i"},
    {Source: "0i", Form: "0i"},
    {Source: "0_0i", Form: "0i"},
    {Source: "10i", Form: "10i"},
    {Source: "1.5E3i", Form: "1.5e3i"},
    {Source: "0755E1i", Form: "0755e1i"},
    {Source: "0.5E1i", Form: "0.5e1i"},
    {Source: "00e1i", Form: "00e1i"},
    {Source: "0E1i", Form: "0e1i"},
    {Source: "0x10i", Form: "0x10i"},
    {Source: "1_000", Form: "1_000"},
  } {
    testify.Equal(
      t,
      "package one\n\nconst VALUE = "+one.Form+"\n",
      literal_form(t, one.Source),
      "the literal prints in the canonical form",
    )
  }
}

// TEST_FILLED_HEAD_TOKENS is how many tokens the head of the filled source holds: package, its
// name, the semicolon behind them, and func, f, both parentheses and the brace.
const TEST_FILLED_HEAD_TOKENS = 8

// Builds the refused sources whose reports every entry must read: the first two codes, the deepest
// code, and refusals at the second token, the third, and the final one of the widest run.
func refused_sources() (sources []string) {
  nested := make([]byte, 0, 2*ast.DEPTH_MAXIMUM+64)
  nested = append(nested, "package one\n\nvar a = "...)
  for range ast.DEPTH_MAXIMUM + 4 {
    nested = append(nested, '(')
  }
  nested = append(nested, 'b')
  for range ast.DEPTH_MAXIMUM + 4 {
    nested = append(nested, ')')
  }
  nested = append(nested, '\n')
  head := "package one\n\nfunc f() {\n"
  filled := make([]byte, len(head), ast.TOKEN_COUNT_MAXIMUM+len(head))
  copy(filled, head)
  for range ast.TOKEN_COUNT_MAXIMUM - TEST_FILLED_HEAD_TOKENS - 1 {
    filled = append(filled, ';')
  }
  return []string{
    "package )\n",
    "package one )\n",
    "package one\n\ntype R interface{\n  Read() int\n}\n",
    // A form the author left open closes on no bracket at all.
    "package one\n\nvar a = g(\n  1,\n",
    "package one\n\nfunc Fold(\n  one int,\n",
    // A field that binds names and states no type stands as the author wrote it.
    "package one\n\ntype S struct {\n  A, B\n}\n",
    string(nested),
    string(filled),
  }
}

// Writes as many tabs as a line opens at.
func tabs(count int) (text string) {
  for range count {
    text = text + "  "
  }
  return text
}

// Prints the forms nested past the deepest indent and the deepest nest, which the print refuses.
func test_deep(t *testing.T) {
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  result := printer.Print_Result{}
  // A line opens at most at the deepest indent, thus a body nested past it is refused, and
  // one nested right to it prints. A form on the deepest line whose parts stand one tab
  // deeper is refused too.
  for _, depth := range []int{printer.INDENT_MAXIMUM, printer.INDENT_MAXIMUM + 1} {
    statement := "f()\n"
    if depth > printer.INDENT_MAXIMUM {
      statement = "f([]int{1})\n"
    }
    nested := "package one\n\nfunc f() {\n"
    for level := 1; level < depth; level = level + 1 {
      nested = nested + tabs(level) + "{\n"
    }
    nested = nested + tabs(depth) + statement
    for level := depth - 1; level > 0; level = level - 1 {
      nested = nested + tabs(level) + "}\n"
    }
    nested = nested + "}\n"
    ast.Parse(tree, token.Source(nested))
    result = printer.Print(subject, storage, tree, token.Source(nested))
    testify.Equal(
      t,
      printer.Boolean(depth == printer.INDENT_MAXIMUM),
      result.OK,
      "a body nested past the deepest indent is refused",
    )
  }
  // A chain of one sign nests one node inside the next, thus a chain deeper than the walk
  // holds is refused.
  chain := "package one\n\nconst TEXT = 1"
  for range printer.DEPTH_MAXIMUM {
    chain = chain + " + 1"
  }
  ast.Parse(tree, token.Source(chain+"\n"))
  result = printer.Print(subject, storage, tree, token.Source(chain+"\n"))
  testify.False(t, bool(result.OK), "a chain nested past the deepest walk is refused")
  // An expression stands at most at the deepest nest, thus an index nested past it is
  // refused.
  calls := "package one\n\nvar a = "
  for range printer.NEST_MAXIMUM + 1 {
    calls = calls + "a["
  }
  calls = calls + "1"
  for range printer.NEST_MAXIMUM + 1 {
    calls = calls + "]"
  }
  ast.Parse(tree, token.Source(calls+"\n"))
  result = printer.Print(subject, storage, tree, token.Source(calls+"\n"))
  testify.False(t, bool(result.OK), "an expression nested past the deepest nest is refused")
  // A sign standing at the deepest nest binds a value standing there too, thus the print
  // reads that nest where it writes the value behind the sign.
  signs := "package one\n\nvar a = "
  for range printer.NEST_MAXIMUM {
    signs = signs + "a["
  }
  signs = signs + "1 + 1"
  for range printer.NEST_MAXIMUM {
    signs = signs + "]"
  }
  ast.Parse(tree, token.Source(signs+"\n"))
  result = printer.Print(subject, storage, tree, token.Source(signs+"\n"))
  testify.False(t, bool(result.OK), "a sign nested past the deepest nest is refused")
}

// Prints sources the author left open or wrote by accident, which a print reads the same way it
// reads any other: every read stands inside the run the scan wrote, and a tree the parse holds
// prints the same form however many times it runs.
func test_hostile(t *testing.T) {
  sample := "package one\n\n// One holds a count.\ntype One int\n\nfunc One_Invariants(" +
    "value One) {\n  if value > 0 {\n    return\n  }\n}\n\nvar table = map[int]int{\n" +
    "  1: 2,\n  3: 4,\n}\n\nfunc Fold(one int, two int) (sum int) {\n  sum = one*2 +" +
    " two\n  for step := range sum {\n    g(step, []int{1, 2}) // a note\n  }\n" +
    "  return sum\n}\n"
  marks := "(){}[],;:=+*&.\n  /\"`1a"
  seed := uint64(1442695040888963407)
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  for range 32 {
    held := []byte(sample)
    for range 4 {
      seed = seed*6364136223846793005 + 1442695040888963407
      place := int(seed>>33) % len(held)
      seed = seed*6364136223846793005 + 1442695040888963407
      held[place] = marks[int(seed>>33)%len(marks)]
    }
    tree := parse_state()
    source := token.Source(held)
    ast.Parse(tree, source)
    result := printer.Print(printer_state(), storage, tree, source)
    testify.True(
      t,
      !(int(result.Count) > len(storage)),
      "a source of anything prints inside storage",
    )
    if tree.Failure != 0 {
      continue
    }
    form := token.Source(storage[:result.Count])
    again := parse_state()
    ast.Parse(again, form)
    testify.Equal(
      t,
      ast.FAILURE_NONE,
      again.Failure,
      "a form the print wrote for a tree the parse holds parses as it stands",
    )
    printed_again := printer.Print(printer_state(), storage, again, form)
    testify.Equal(
      t,
      string(form),
      string(storage[:printed_again.Count]),
      "a tree the parse holds prints one form however many times it runs",
    )
  }
}

// Prints the spread that holds a value, which a source that states one behind the points hangs
// on them. The print writes that value rather than dropping it, thus the form it writes states
// every token the tree holds and prints as itself.
func test_held_spread(t *testing.T) {
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  spread := token.Source("package one\n\nvar held = []int{g(1)...*f(2)}\n")
  ast.Parse(tree, spread)
  result := printer.Print(printer_state(), storage, tree, spread)
  stood := string(storage[:result.Count])
  testify.Equal(
    t,
    "package one\n\nvar held = []int{g(1), ...*f(2)}\n",
    stood,
    "a spread that names what a call reads keeps the brackets of that call",
  )
  ast.Parse(tree, token.Source(stood))
  result = printer.Print(printer_state(), storage, tree, token.Source(stood))
  testify.Equal(
    t,
    stood,
    string(storage[:result.Count]),
    "a spread that holds a value prints as itself",
  )
}

// Reads the program of the layout machine and states the bounds the print reads it at: the steps
// it holds and the widest operand one of them states.
func test_program(t *testing.T) {
  testify.Equal(
    t,
    printer.LAYOUT_STEP_COUNT,
    len(printer.LAYOUT_PROGRAM)/printer.LAYOUT_STEP_SIZE,
    "the step count names every step",
  )
  widest := 0
  for step := range printer.LAYOUT_STEP_COUNT {
    at := step * printer.LAYOUT_STEP_SIZE
    operand := int(printer.LAYOUT_PROGRAM[at+1])*printer.LAYOUT_OPERAND_BASE +
      int(printer.LAYOUT_PROGRAM[at+2])
    widest = max(widest, operand)
  }
  testify.Equal(
    t,
    printer.LAYOUT_OPERAND_MAXIMUM,
    widest,
    "the widest operand of the program states the bound the print reads it at",
  )
}

// Reads the bytes one form states: no line closes on a space or a tab, no two empty lines stand
// together, every note that follows a byte stands behind a space, and the form opens on a byte
// and closes on one line feed.
func test_bytes(t *testing.T) {
  form := printed(
    t,
    "package one\n\n// One holds a count.\ntype One int\n\n"+
      "func Fold(one int, two int) (sum int) {\n  sum = one*2 + two // a note\n\n"+
      "  for step := range sum {\n    sum = sum + step\n  }\n  return sum\n}\n",
  )
  testify.True(t, len(form) > 0, "a file states a form of its own")
  testify.Equal(t, uint8('\n'), form[len(form)-1], "the form closes on one line feed")
  testify.True(t, form[0] != '\n', "the form opens on a byte")
  feeds := 0
  for place := range len(form) {
    held := form[place]
    if held == '\n' {
      feeds = feeds + 1
      testify.True(t, feeds < 3, "no two empty lines stand together")
      continue
    }
    feeds = 0
    if place+1 != len(form) {
      if held == ' ' {
        testify.True(t, form[place+1] != '\n', "no line closes on a space")
      }
      if held == '\t' {
        testify.True(t, form[place+1] != '\n', "no line closes on a tab")
      }
    }
  }
}

// Prints a tree that stands for anything but a file, which states no form the print writes, thus
// the print writes nothing at all for it.
func test_stood_tree(t *testing.T) {
  tree := parse_state()
  source := token.Source("package one\n\nvar a = 1\n")
  ast.Parse(tree, source)
  tree.Nodes.Values[1].Kind = ast.NODE_BLOCK
  empty := printer.Print(printer_state(), make([]byte, printer.FORM_SIZE_MAXIMUM), tree,
    source)
  testify.Equal(t, printer.Form_Count(0), empty.Count,
    "a tree that states no file writes no byte")
}

// Prints the notes the author left ahead of the words a form opens with: behind the word a
// declaration states, and behind the name a type states. A declaration writes every part it
// holds however many notes stand inside it, and a note on the line a brace opens keeps that line.
func test_notes_ahead(t *testing.T) {
  testify.Equal(
    t,
    "package one\n\nvar table = 1 /* d */\n",
    printed(t, "package one\n\nvar /* d */ table = 1\n"),
    "a declaration that opens behind a note writes the names it binds",
  )
  testify.Equal(
    t,
    "package one\n\nvar table = 1 /* d */ /* b */\n",
    printed(t, "package one\n\nvar /* d */ /* b */ table = 1\n"),
    "a declaration that opens behind a run of notes writes the names it binds",
  )
  testify.Equal(
    t,
    "package one\n\nconst C = 1 /* d */\n",
    printed(t, "package one\n\nconst /* d */ C = 1\n"),
    "a constant that opens behind a note writes the name it binds",
  )
  round_trip(t, "package one\n\ntype S struct { // d\n  One int\n}\n")
  // A note the author left behind the sign of an assignment stands between that sign and
  // the value it binds, thus the print reads the sign past it.
  testify.Equal(
    t,
    "package one\n\nfunc F() {\n  sum = one /* d */\n}\n",
    printed(t, "package one\n\nfunc F() {\n  sum = /* d */ one\n}\n"),
    "an assignment reads its sign past the notes behind it",
  )
  // A note the author left inside one statement lands at the end of the line that
  // statement writes, thus it stands in the column of the run around it.
  testify.Equal(
    t,
    "package one\n\nfunc F() (sum int) {\n  sum = one + two /* b */\n"+
      "  return sum      /* d */\n}\n",
    printed(
      t,
      "package one\n\nfunc F() (sum int) {\n  sum  /* b */ = one + two\n"+
        "  return sum /* d */\n}\n",
    ),
    "a note inside a statement stands in the column of the run it lands in",
  )
  // A note that opens a line the source of that line runs on past stands with that
  // source, thus it takes no line of its own.
  testify.Equal(
    t,
    "package one\n\nvar table = []int{1} /* d */ // a\n",
    printed(t, "package one\n\nvar table = []int{\n  /* d */  1 // a\n}\n"),
    "a note that opens a line of source stands with the source behind it",
  )
  test_lines_inside(t)
  // A literal that holds notes alone keeps them between its braces.
  round_trip(t, "package one\n\nvar table = map[string]int{\n  // a\n}\n")
  round_trip(t, "package one\n\nvar held = []int{\n  // a\n}\n")
  // A form that holds a note on the line its brace opens and no part behind it closes on
  // that note.
  round_trip(t, "package one\n\ntype S struct { // a\n}\n")
  round_trip(t, "package one\n\ntype R interface { // a\n}\n")
  round_trip(t, "package one\n\nfunc F() {\n  switch a {\n  case 1: // a\n  }\n}\n")
  round_trip(t, "package one\n\ntype Reader interface { // d\n  ~int\n}\n")
  testify.Equal(
    t,
    "package one\n\ntype S struct { /* d */\n  One int\n}\n",
    printed(t, "package one\n\ntype S /* d */ struct {\n  One int\n}\n"),
    "a note ahead of the word a type states keeps the line its brace opens",
  )
}

// Checks the empty lines one form drops: the ones the author left inside a head, inside an
// expression, or ahead of the brace that opens a body.
func test_lines_inside(t *testing.T) {
  // An empty line between the word a form states and the brace that opens its body
  // states nothing, thus the print drops it.
  testify.Equal(
    t,
    "package one\n\ntype Reader interface {\n  ~int\n}\n",
    printed(t, "package one\n\ntype Reader interface\n\n{\n  ~int\n}\n"),
    "an empty line ahead of the brace one form opens states nothing",
  )
  testify.Equal(
    t,
    "package one\n\ntype S struct {\n  One int\n}\n",
    printed(t, "package one\n\ntype S struct\n\n{\n  One int\n}\n"),
    "an empty line ahead of the brace one structure opens states nothing",
  )
  // A declaration writes its head on one line, thus an empty line the author left inside
  // that head states nothing.
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  g()\n}\n",
    printed(t, "package one\n\nfunc\n\nFold() {\n  g()\n}\n"),
    "an empty line inside the head of a function states nothing",
  )
  testify.Equal(
    t,
    "package one\n\nimport \"local/one/two\"\n\nvar held = two.Value\n",
    printed(t, "package one\n\nimport\n\n\"local/one/two\"\n\nvar held = two.Value\n"),
    "an empty line inside the head of an import states nothing",
  )
  testify.Equal(
    t,
    "package one\n\ntype Pair[T any] struct {\n  Left T\n}\n",
    printed(t, "package one\n\ntype Pair[\n\nT any] struct {\n  Left T\n}\n"),
    "an empty line inside a list of type parameters states nothing",
  )
  // An expression and a statement each write on one line, thus an empty line the author
  // left inside one states nothing.
  testify.Equal(
    t,
    "package one\n\ntype Reader interface {\n  ~int | ~string\n}\n",
    printed(t, "package one\n\ntype Reader interface {\n  ~int |\n\n~string\n}\n"),
    "an empty line inside a union states nothing",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  defer func() {\n    g()\n  }()\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  defer\n\n  func() {\n"+
      "    g()\n  }()\n}\n"),
    "an empty line inside a statement states nothing",
  )
  testify.Equal(
    t,
    "package one\n\nfunc (one *S) Method() {\n}\n",
    printed(t, "package one\n\nfunc (one *\n\nS) Method() {\n}\n"),
    "an empty line inside a pointer type states nothing",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  defer func() {\n    g()\n  }()\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  defer func(\n\n) {\n"+
      "    g()\n  }()\n}\n"),
    "an empty line inside the head of a function literal states nothing",
  )
}

// A note the author left inside the head of a type, or inside the head of the function that
// states its invariants, names nothing of either, thus the pair still stands.
func test_order_notes(t *testing.T) {
  testify.Equal(
    t,
    "package one\n\ntype One int /* note */\n\nfunc One_Invariants(value One) {\n"+
      "  _ = value\n}\n\nvar two = 2\n",
    printed(
      t,
      "package one\n\ntype/* note */ One int\n\nvar two = 2\n\n"+
        "func One_Invariants(value One) {\n  _ = value\n}\n",
    ),
    "a type carrying a note inside its head pairs with the function of its name",
  )
  testify.Equal(
    t,
    "package one\n\ntype One int\n\nfunc One_Invariants(value One) { /* note */\n"+
      "  _ = value\n}\n\nvar two = 2\n",
    printed(
      t,
      "package one\n\ntype One int\n\nvar two = 2\n\n"+
        "func/* note */ One_Invariants(value One) {\n  _ = value\n}\n",
    ),
    "a function carrying a note inside its head pairs with the type of its name",
  )
  // One function of invariants stands under one type, thus a note inside the head of
  // either type of one name leaves the second type standing with none of it.
  testify.Equal(
    t,
    "package one\n\ntype A int /* note */\n\nfunc A_Invariants() {\n}\n\ntype A int\n",
    printed(t, "package one\n\ntype/* note */ A int\n\nfunc A_Invariants() {\n}\n\n"+
      "type A int\n"),
    "a noted head leaves one function under one type",
  )
  testify.Equal(
    t,
    "package one\n\ntype A int\n\nfunc A_Invariants() {\n}\n\ntype A int /* note */\n",
    printed(t, "package one\n\ntype A int\n\nfunc A_Invariants() {\n}\n\n"+
      "type/* note */ A int\n"),
    "a noted head behind the pair takes no function of its own",
  )
  // One type takes one function of invariants under it, thus a second function of that
  // name stands where the author wrote it however either head reads.
  testify.Equal(
    t,
    "package one\n\ntype A int\n\nfunc A_Invariants() { /* note */\n}\n\n"+
      "func A_Invariants() {\n}\n",
    printed(
      t,
      "package one\n\ntype A int\n\nfunc/* note */ A_Invariants() {\n}\n\n"+
        "func A_Invariants() {\n}\n",
    ),
    "a noted function takes the type and the second one stands where it stood",
  )
  testify.Equal(
    t,
    "package one\n\ntype A int\n\nfunc A_Invariants() {\n}\n\n"+
      "func A_Invariants() { /* note */\n}\n",
    printed(
      t,
      "package one\n\ntype A int\n\nfunc A_Invariants() {\n}\n\n"+
        "func/* note */ A_Invariants() {\n}\n",
    ),
    "a noted second function stands where the author wrote it",
  )
}

// A print that fills the storage the caller supplied stops and reports it, thus the count it
// hands back stands inside that storage, a print that reports the form wrote the whole of it,
// and a print into storage that holds the form reports it.
func test_storage_widths(t *testing.T) {
  source := "package one\n\nfunc Fold() {\n  g() // note\n  h()\n}\n"
  state := parse_state()
  ast.Parse(state, token.Source(source))
  held := make([]byte, printer.FORM_SIZE_MAXIMUM)
  whole := printer.Print(printer_state(), held, state, token.Source(source))
  testify.True(t, bool(whole.OK), "the print holds the whole form")
  form := string(held[:whole.Count])
  for size := range len(form) + 2 {
    storage := make([]byte, size)
    one := printer.Print(printer_state(), storage, state, token.Source(source))
    testify.True(t, !(int(one.Count) > size), "the count stands inside the storage")
    testify.Equal(
      t,
      !(size < len(form)),
      bool(one.OK),
      "a print reports the form where the storage holds it",
    )
    if bool(one.OK) {
      testify.Equal(t, form, string(storage[:one.Count]),
        "a print that holds wrote the whole form")
    }
  }
}

// An element that runs the line past the widest one stands over lines of its own once the print
// breaks it, thus it opens no column with the elements around it and the note it carries stands
// one space behind the byte it follows.
func test_broken_element(t *testing.T) {
  // The element stands three columns past the widest line the print writes, counting the
  // byte its line opens at, the key and the bracket ahead of it, and the comma behind it.
  name := spelled(
    'a',
    printer.COLUMN_WIDTH_MAXIMUM-printer.INDENT_WIDTH-
      len("\"k\": {")-len("},")+3,
  )
  testify.Equal(
    t,
    "package one\n\nvar x = map[string][]int{\n  \"k\": { /* n */\n    "+name+
      ",\n  },\n}\n",
    printed(t, "package one\n\nvar x = map[string][]int{\"k\":/* n */ {"+name+"}}\n"),
    "a note inside a broken element stands behind the byte it follows",
  )
}

// A key wider than the widest width one measure answers with states the widest column a run of
// pairs opens at, and a run of one pair pads no value at all, thus that column stands as the
// measure clamps it rather than as a line no reader reads.
func test_widest_column(t *testing.T) {
  // A tab stands in the columns a reader sees, thus a key of tabs spells the widest
  // column a measure answers with in an eighth of the bytes a key of names spends.
  key := spelled('\t', printer.WIDTH_MAXIMUM/TEST_TAB_WIDTH)
  source := "package one\n\nvar wide = map[string]int{`" + key + "`: 1}\n"
  testify.Equal(
    t,
    "package one\n\nvar wide = map[string]int{\n  `"+key+"`: 1,\n}\n",
    printed(t, source),
    "a run of one pair states the column its key spells",
  )
}

// TEST_TAB_WIDTH is how many columns one tab stands in, which a tab inside a string or a note
// spends and which no line of the form opens at.
const TEST_TAB_WIDTH = 8

// A note the author left between the type one field states and the tag behind it states neither,
// thus the print holds that note and writes the tag the field carries.
func test_tag_notes(t *testing.T) {
  testify.Equal(
    t,
    "package one\n\ntype S struct {\n  A  int    `one:\"1\"` /* n */\n"+
      "  Bb string `two:\"2\"`\n}\n",
    printed(
      t,
      "package one\n\ntype S struct {\n  A int/* n */ `one:\"1\"`\n"+
        "  Bb string `two:\"2\"`\n}\n",
    ),
    "a field holds the tag a note stands ahead of",
  )
  testify.Equal(
    t,
    "package one\n\ntype S struct {\n  Int `one:\"1\"` /* n */\n}\n",
    printed(t, "package one\n\ntype S struct {\n  Int/* n */ `one:\"1\"`\n}\n"),
    "a field of no name holds the tag a note stands ahead of",
  )
}

// A tab stands in the columns a reader sees wherever it stands, thus a note that holds one spends
// those columns and the wrap that reads its words reads them the way the print writes them.
func test_tab_notes(t *testing.T) {
  round_trip(t, "package one\n\n// held by a tab\nvar x = 1\n")
  words := ""
  for range 12 {
    words = words + " word"
  }
  printed(t, "package one\n\n//"+words+" and tabs"+words+"\nvar x = 1\n")
}

// Two signs standing together spell a third sign or the mark a comment opens with, thus a sign
// that closes up against the value behind it keeps the space that parts the two.
func test_clashing_signs(t *testing.T) {
  for _, one := range []struct {
    Head string
    Form string
  }{
    {Head: "b < -c", Form: "b < -c"},
    {Head: "b / *c", Form: "b / *c"},
    {Head: "b & ^c", Form: "b & ^c"},
    {Head: "b + +c", Form: "b + +c"},
    {Head: "b - -c", Form: "b - -c"},
    {Head: "b * *c", Form: "b**c"},
  } {
    held := "package one\n\nfunc Fold(a []int, b int, c *int) {\n  _ = a["
    testify.Equal(
      t,
      held+one.Form+"]\n}\n",
      printed(t, held+one.Head+"]\n}\n"),
      "a sign that would spell a second one keeps the space that parts them",
    )
  }
}

// A form standing inside the widest line the canonical form writes stands on the line it opened,
// and one standing past it states one part to a line, thus the widest line names the boundary.
func test_widest_line(t *testing.T) {
  held := "package one\n\nfunc Two() {\n  _ = Fold(one, "
  name := spelled('a',
    printer.COLUMN_WIDTH_MAXIMUM-printer.INDENT_WIDTH-len("_ = Fold(one, )"))
  flat := held + name + ")\n}\n"
  testify.Equal(
    t,
    flat,
    printed(t, flat),
    "a call standing inside the widest line stands on the line it opened",
  )
  broken := held + name + "a)\n}\n"
  testify.Equal(
    t,
    "package one\n\nfunc Two() {\n  _ = Fold(\n    one,\n    "+name+"a,\n  )\n}\n",
    printed(t, broken),
    "a call standing past the widest line states one part to a line",
  )
}

// One character stands in one column however many bytes it spans, and a tab stands in the columns
// a reader sees, thus a run of keys and a run of notes each align on what a reader reads.
func test_wide_columns(t *testing.T) {
  testify.Equal(
    t,
    "package one\n\nvar One = map[string]int{\n  \"éé\":   1,\n  \"bbbb\": 2,\n}\n",
    printed(t, "package one\n\nvar One = map[string]int{\n  \"éé\": 1,\n"+
      "  \"bbbb\": 2,\n}\n"),
    "a run of keys aligns on the columns a reader reads",
  )
  testify.Equal(
    t,
    "package one\n\ntype One struct {\n  Two       int // éé\n"+
      "  ThreeFour string // b\n}\n",
    printed(t, "package one\n\ntype One struct {\n  Two int // éé\n"+
      "  ThreeFour string // b\n}\n"),
    "a run of notes aligns on the columns a reader reads",
  )
}

// A note of no word at all states one, and a word wider than the widest line stands on a line of
// its own, thus neither shape opens a line the print never closes.
func test_note_shapes(t *testing.T) {
  round_trip(t, "package one\n\n//\nvar x = 1\n")
  round_trip(t, "package one\n\n//\n//\nvar x = 1\n")
  round_trip(t, "package one\n\n/**/\nvar x = 1\n")
  held := "https://example.com/"
  for range 12 {
    held = held + "segment/"
  }
  testify.Equal(
    t,
    "package one\n\n// "+held+"\nvar x = 1\n",
    printed(t, "package one\n\n// "+held+"\nvar x = 1\n"),
    "a word wider than the widest line stands on a line of its own",
  )
}

// A return the author left inside a note states nothing of the text it holds, thus the canonical
// form spells that note without it.
func test_return_notes(t *testing.T) {
  testify.Equal(
    t,
    "package one\n\n/* ab */\nvar x = 1\n",
    printed(t, "package one\n\n/* a\rb */\nvar x = 1\n"),
    "a block note drops the return it holds",
  )
  testify.Equal(
    t,
    "package one\n\n// ab\nvar x = 1\n",
    printed(t, "package one\n\n// a\rb\nvar x = 1\n"),
    "a note of one line drops the return it holds",
  )
}

// An empty line the author left behind a run of noted statements closes that run and states
// nothing of the statements it holds, thus those statements stand in the column the run opens.
func test_trailing_line(t *testing.T) {
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  a = 1    // one\n  bbbb = 2 // two\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  a = 1 // one\n  bbbb = 2 // two\n\n"+
      "}\n"),
    "a line behind the run closes it and states nothing of its column",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  a = 1    // one\n  bbbb = 2 // two\n\n"+
      "  c = 3\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  a = 1 // one\n  bbbb = 2 // two\n\n"+
      "  c = 3\n}\n"),
    "a line between two runs closes the run ahead of it",
  )
}

// A chain the author broke ahead of a name it reads opens the line that name stands on, and the
// chain hands that line back once it closed, thus every name of one chain opens at the same tabs
// however many breaks the chain holds.
func test_broken_chain(t *testing.T) {
  round_trip(t, "package one\n\nfunc F() {\n  a.B(x).\n    C(y)\n}\n")
  round_trip(t, "package one\n\nfunc F() {\n  a.B(x).\n    C(y).\n    D(z)\n}\n")
  round_trip(t, "package one\n\nfunc F() {\n  g(a.B(x).\n    C(y))\n}\n")
}

// A call the author broke behind its parenthesis stands broken while the calls inside it open
// their own brackets, thus a call inside one opens where the form around it already broke.
func test_broken_call(t *testing.T) {
  round_trip(t, "package one\n\nfunc F() {\n  g(\n    h(x),\n  )\n}\n")
  round_trip(t, "package one\n\nfunc F() {\n  g(\n    h(\n      x,\n    ),\n  )\n}\n")
  round_trip(t, "package one\n\nfunc F() {\n  g(a,\n    h(x))\n}\n")
  // A value the author wrote on a line of its own opens that line ahead of the call it
  // states, thus the call opens its bracket where the statement already broke.
  round_trip(t, "package one\n\nfunc F(a int, b int) {\n  a, b = one,\n    g(x)\n}\n")
  round_trip(t, "package one\n\nfunc F() (one int, two int) {\n  return three,\n"+
    "    g(x)\n}\n")
  round_trip(
    t,
    "package one\n\nfunc F(a int, b int) {\n  a, b = one,\n    g(\n"+
      "      x,\n      y,\n    )\n}\n",
  )
}

// A file that declares the entry of a program states it ahead of every other function it declares,
// thus a reader opens the file on what the program runs first. The notes above that function move
// with it, and every other declaration stands where the author wrote it.
func test_entry(t *testing.T) {
  testify.Equal(
    t,
    "package one\n\nfunc main() {\n  a()\n}\n\nfunc Fold() {\n  b()\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  b()\n}\n\nfunc main() {\n  a()\n}\n"),
    "one file states main ahead of every other function",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Main() {\n  a()\n}\n\nfunc Fold() {\n  b()\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  b()\n}\n\nfunc Main() {\n  a()\n}\n"),
    "one file states Main ahead of every other function",
  )
  // A file that already states main first states it there still.
  round_trip(t, "package one\n\nfunc main() {\n  a()\n}\n\nfunc Fold() {\n  b()\n}\n")
  // A file that declares no such function stands as the author wrote it.
  round_trip(t, "package one\n\nfunc Fold() {\n  b()\n}\n\nfunc Hold() {\n  a()\n}\n")
  test_main_notes(t)
}

// The notes above the entry of one program move with it, and the declarations of every other kind
// stand where the author wrote them. A function that states the invariants of a type stands under
// that type, thus it opens no place the entry moves to.
func test_main_notes(t *testing.T) {
  testify.Equal(
    t,
    "package one\n\nconst A = 1\n\nfunc main() {\n  a()\n}\n\n"+
      "func Fold() {\n  b()\n}\n",
    printed(
      t,
      "package one\n\nconst A = 1\n\nfunc Fold() {\n  b()\n}\n\n"+
        "func main() {\n  a()\n}\n",
    ),
    "a declaration of another kind stands where the author wrote it",
  )
  testify.Equal(
    t,
    "package one\n\n// Runs.\nfunc main() {\n  a()\n}\n\n"+
      "func Fold() {\n  b()\n}\n",
    printed(t, "package one\n\nfunc Fold() {\n  b()\n}\n\n// Runs.\n"+
      "func main() {\n  a()\n}\n"),
    "the notes above the entry move with it",
  )
  testify.Equal(
    t,
    "package one\n\nfunc main() {\n  a()\n}\n\n// Folds.\n"+
      "func Fold() {\n  b()\n}\n",
    printed(t, "package one\n\n// Folds.\nfunc Fold() {\n  b()\n}\n\n"+
      "func main() {\n  a()\n}\n"),
    "the notes above the place stand with the declaration they open",
  )
  testify.Equal(
    t,
    "package one\n\nfunc main() {\n  a()\n}\n\ntype A int\n\n"+
      "func A_Invariants() {\n  c()\n}\n",
    printed(
      t,
      "package one\n\ntype A int\n\nfunc A_Invariants() {\n  c()\n}\n\n"+
        "func main() {\n  a()\n}\n",
    ),
    "a function standing under a type opens no place the entry moves to",
  )
}

// Prints one source the parse refused and reads that the form holds the text it states. A tree
// the parse refused prints what it holds, because a caller reads more from a partial print than
// from nothing.
func prints_refused(t *testing.T, source string, held string) {
  t.Helper()
  refused := parse_state()
  text := token.Source(source)
  ast.Parse(refused, text)
  testify.True(t, refused.Failure != ast.FAILURE_NONE, "the parse refuses the source")
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  result := printer.Print(printer_state(), storage, refused, text)
  testify.True(
    t,
    !(offset_of(string(storage[:result.Count]), held) < 0),
    "a refused tree prints what it holds",
  )
}

// A default states the statements every other case leaves. The dialect refuses a case that runs
// on into the case behind it, thus that word stands in a tree the parse refused alone, and the
// print writes what such a tree holds.
func test_switch_cases(t *testing.T) {
  round_trip(
    t,
    "package one\n\nfunc Fold() {\n  switch a {\n  case 1:\n    g()\n"+
      "  default:\n    h()\n  }\n}\n",
  )
  prints_refused(
    t,
    "package one\n\nfunc Fold() {\n  switch a {\n  case 1:\n"+
      "    fallthrough\n  default:\n  }\n}\n",
    "fallthrough",
  )
}

// A source that stands in the canonical form reads the same to the byte it closes on, and one
// that parts from it stops the print where the two part. The print writes no byte behind that
// one, thus a caller that asks only the question pays for the answer alone.
func test_comparison(t *testing.T) {
  same := "package one\n\nvar held = 1\n"
  testify.True(t, bool(compared(t, same)),
    "a source in the canonical form reads the same")
  testify.False(
    t,
    bool(compared(t, "package one\n\nvar  held = 1\n")),
    "a source that parts from the form reads otherwise",
  )
  // A source the print writes more of than the source holds parts at its own end.
  testify.False(
    t,
    bool(compared(t, "package one\n\nvar held = 1")),
    "a source that closes early reads otherwise",
  )
  // A compare stops where the two part, thus the form holds the bytes ahead of that byte
  // and no byte behind it.
  subject := printer_state()
  tree := parse_state()
  held := token.Source("package one\n\nvar  held = 1\n")
  ast.Parse(tree, held)
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  printer.Compare(subject, storage, tree, held)
  testify.Equal(
    t,
    "package one\n\nvar ",
    string(storage[:len("package one\n\nvar ")]),
    "the form holds the bytes ahead of the byte the two state otherwise",
  )
  compares_widths(t)
}

// A caller compares whatever source and storage it holds, thus the compare answers for a tree the
// parse refused, for the narrowest storage and the widest, and for every byte a string states.
func compares_widths(t *testing.T) {
  // The reports one compare states run from the clean parse to the deepest refusal, thus
  // these sources state every code and every token place a compare answers for.
  for _, one := range []string{
    "package )\n",
    "package one )\n",
    "package one\n\ntype R interface{\n  Read() int\n}\n",
    refused_sources()[6],
    refused_sources()[7],
    "package one\n\nconst TEXT = `\x00\x01\x02\xff`\n",
    padded_source(),
  } {
    compared(t, one)
  }
  for _, size := range []int{
    printer.FORM_SIZE_MINIMUM,
    printer.FORM_SIZE_MINIMUM + 1,
    printer.FORM_SIZE_MINIMUM + 2,
    printer.FORM_SIZE_MAXIMUM,
  } {
    for _, text := range []string{"", "p", "pa", "package one\n"} {
      tree := parse_state()
      held := token.Source(text)
      ast.Parse(tree, held)
      printer.Compare(printer_state(), make([]byte, size), tree, held)
    }
  }
}

// Builds a source of the widest admitted size that states one clause and nothing else, thus a
// compare of it reads the widest source a caller hands over and costs one clause of work.
func padded_source() (text string) {
  body := make([]byte, 0, token.SOURCE_SIZE_MAXIMUM)
  body = append(body, "package one\n"...)
  for len(body) < token.SOURCE_SIZE_MAXIMUM {
    body = append(body, ' ')
  }
  return string(body)
}

// Compares one source against the form the print writes for it.
func compared(t *testing.T, source string) (same printer.Boolean) {
  t.Helper()
  tree := parse_state()
  held := token.Source(source)
  ast.Parse(tree, held)
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  return printer.Compare(printer_state(), storage, tree, held)
}
