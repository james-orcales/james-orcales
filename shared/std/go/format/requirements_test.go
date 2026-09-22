package format_test

import "testing"

import "local/james-orcales/shared/std/go/ast"
import "local/james-orcales/shared/std/go/format"
import "local/james-orcales/shared/std/go/printer"
import "local/james-orcales/shared/std/go/token"
import "local/james-orcales/shared/std/testify"

// Test_Format binds the Format specification leaf before fixture declarations.
func Test_Format(t *testing.T) {
  test_format(t)
}

// Test_Clean binds the Clean specification leaf before fixture declarations.
func Test_Clean(t *testing.T) {
  test_clean(t)
}

// Test_Imports binds the Imports specification leaf before fixture declarations.
func Test_Imports(t *testing.T) {
  test_imports(t)
}

// Test_Names binds the Names specification leaf before fixture declarations.
func Test_Names(t *testing.T) {
  test_names(t)
}

// Test_Declarations binds the Declarations specification leaf before fixture declarations.
func Test_Declarations(t *testing.T) {
  test_declarations(t)
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
  Clean printer.Boolean
  Ok    bool
}

// Caller allocation keeps sort cursors bounded across both format operations.
func format_state() (state *format.Formatter) {
  return &format.Formatter{
    Places: format.Place_Storage{
      Values: make(format.Places, format.PLACE_SLOT_COUNT),
    },
    Table_Fields: format.Table_Fields{
      Names: format.Name_Storage{
        Values: make(format.Names, format.NAME_SLOT_COUNT),
      },
      Texts: format.Text_Storage{
        Values: make(format.Text, format.NAME_TEXT_SIZE_MAXIMUM),
      },
      Slots: format.Slot_Storage{
        Values: make(format.Slots, format.SLOT_COUNT),
      },
    },
  }
}

type name_case struct {
  Source string
  Form   string
}

func Test_Blank_Identifiers(t *testing.T) {
  source := "package p\n\ntype Value struct {\n  _ int\n}\n\nfunc read(value int, _ int) (result int) {\n  _, result = value, value\n  return result\n}\n"
  testify.Equal(t, source, formatted(t, source))
}

// States each source whose declared names the format renames, beside the form it expects.
func name_cases() (cases []name_case) {
  return append(
    append(field_cases(), walk_name_cases()...),
    []name_case{
      {
        Source: "const fooBar = 1\n\nvar held = fooBar\n",
        Form:   "const FOO_BAR = 1\n\nvar held = FOO_BAR\n",
      },
      {
        Source: "var someValue, Other_value = 1, 2\n",
        Form:   "var some_value, Other_Value = 1, 2\n",
      },
      {
        Source: "type HTTPServer struct {\n  FieldOne int\n  fieldTwo int\n}\n",
        Form:   "type HTTP_Server struct {\n  Field_One int\n  Field_Two int\n}\n",
      },
      {
        Source: "func DoWork(inputValue int) (resultValue int) {\n" +
          "  localValue := inputValue\n  resultValue = localValue\n" +
          "  return resultValue\n}\n",
        Form: "func Do_Work(input_value int) (result_value int) {\n" +
          "  local_value := input_value\n  result_value = local_value\n" +
          "  return result_value\n}\n",
      },
      {
        Source: "var heldValue = \"heldValue\"\n",
        Form:   "var held_value = \"heldValue\"\n",
      },
      {
        Source: "import \"zz\"\n\nvar held = zz.MustCompile\n",
        Form:   "import \"zz\"\n\nvar held = zz.MustCompile\n",
      },
      {
        Source: "func TestMain(value int) {\n  return\n}\n",
        Form:   "func TestMain(value int) {\n  return\n}\n",
      },
      {
        Source: "var foo__bar_ = 1\n",
        Form:   "var foo_bar = 1\n",
      },
      {
        Source: "var heldValue = `heldValue\\` + \"held\\\"heldValue\" + " +
          "'\\'' // heldValue\n",
        Form: "var held_value = `heldValue\\` + \"held\\\"heldValue\" + " +
          "'\\'' // heldValue\n",
      },
      {
        Source: "var heldValue = 1e6 + .5e1 + 0x1Fe /* heldValue */ + 2\n",
        Form:   "var held_value = 1e6 + .5e1 + 0x1Fe + 2 /* heldValue */\n",
      },
      {
        Source: "type Enum_3_Int int\n\ntype User_IDs int\n\ntype V2Layer int\n\n" +
          "type HtTP int\n\ntype Pair[elemType, keyType any] int\n",
        Form: "type Enum_3_Int int\n\ntype User_IDs int\n\ntype V2_Layer int\n\n" +
          "type Ht_TP int\n\ntype Pair[elem_type, key_type any] int\n",
      },
      {
        Source: "const MAX_2 = 1\n\nconst BAD_ = 2\n\nconst X9y = 3\n\n" +
          "var a2B = 4\n",
        Form: "const MAX_2 = 1\n\nconst BAD = 2\n\nconst X9Y = 3\n\n" +
          "var a2_b = 4\n",
      },
    }...,
  )
}

// States each name a receiver and a range clause bind, which the format renames the way it
// renames a name standing alone.
func walk_name_cases() (cases []name_case) {
  return []name_case{
    {
      Source: "func (heldPair Pair) Fold() (sum int) {\n" +
        "  for keyA, valueB := range heldPair {\n" +
        "    sum = keyA + valueB\n  }\n  return sum\n}\n",
      Form: "func (held_pair Pair) Fold() (sum int) {\n" +
        "  for key_a, value_b := range held_pair {\n" +
        "    sum = key_a + value_b\n  }\n  return sum\n}\n",
    },
  }
}

// States each field the format exports, which every package that reads the type spells.
func field_cases() (cases []name_case) {
  return []name_case{
    {
      Source: "type Pair struct {\n  count int\n}\n\n" +
        "func Fold(pair Pair) {\n  g(pair.count)\n}\n",
      Form: "type Pair struct {\n  Count int\n}\n\n" +
        "func Fold(pair Pair) {\n  g(pair.Count)\n}\n",
    },
    {
      Source: "type Pair struct {\n  one, two int\n}\n",
      Form:   "type Pair struct {\n  One, Two int\n}\n",
    },
  }
}

// States each short name the format renames, which is how a search of the table sees a word of
// one letter and of two.
func short_name_cases() (cases []name_case) {
  return []name_case{
    {
      Source: "var aB = 1\n\nvar c = 2\n\nvar dE = 3\n",
      Form:   "var a_b = 1\n\nvar c = 2\n\nvar d_e = 3\n",
    },
  }
}

func test_names(t *testing.T) {
  head := "package one\n\n"
  for _, one := range append(name_cases(), short_name_cases()...) {
    form := formatted(t, head+one.Source)
    testify.Equal(t, head+one.Form, form,
      "a declared name stands in the case the linter asks")
    testify.Equal(t, form, formatted(t, form), "a renamed form formats as itself")
  }
  state := format_state()
  subject := printer_state()
  tree := parse_state()
  source := token.Source(head + "var fooBar = 1\n")
  ast.Parse(tree, source)
  format.Declare(state, tree, source)
  narrow := make([]byte, len(source))
  result := format.Format(state, subject, narrow, tree, source)
  testify.False(t, bool(result.OK), "a rename past the storage is refused")
  testify.Equal(
    t,
    printer.Form_Count(len(narrow)),
    result.Count,
    "a refused rename states the storage it filled",
  )
}

// Caller allocation keeps the printer free of hidden storage growth.
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

// Caller allocation makes parser bounds part of every format fixture.
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

// Formats one source and hands back the form it wrote, which is how each case states the form it
// expects.
func formatted(t *testing.T, source string) (form string) {
  t.Helper()
  state := format_state()
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  held := token.Source(source)
  ast.Parse(tree, held)
  format.Declare(state, tree, held)
  result := format.Format(state, subject, storage, tree, held)
  testify.True(t, bool(result.OK), "the format holds the whole form")
  return string(storage[:result.Count])
}

// Formats through a printer that holds no source or form storage of its own, the way a caller that
// owns one source buffer and one form buffer hands them over.
func lent_format(t *testing.T) {
  subject := printer_state()
  subject.Source = nil
  subject.Form = nil
  tree := parse_state()
  text := "package one\n\nconst VALUE = 0XFF\n"
  buffer := make([]byte, token.SOURCE_SIZE_MAXIMUM+1)
  source := token.Source(buffer[:copy(buffer, text)])
  storage := make(printer.Form, printer.FORM_SIZE_MAXIMUM)
  ast.Parse(tree, source)
  result := format.Format(format_state(), subject, storage, tree, source)
  testify.Equal(
    t,
    "package one\n\nconst VALUE = 0xFF\n",
    string(storage[:result.Count]),
    "a printer that holds no storage of its own formats into the buffers it was handed",
  )
}

// Reports whether one source stands in the canonical form.
func cleanliness(t *testing.T, source string) (clean printer.Boolean) {
  t.Helper()
  state := format_state()
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  held := token.Source(source)
  ast.Parse(tree, held)
  result := format.Clean(state, subject, storage, tree, held)
  testify.True(t, bool(result.OK), "the clean report holds the whole form")
  return result.Clean
}

func test_format(t *testing.T) {
  lent_format(t)
  canonical := "package one\n\nfunc Fold() (sum int) {\n  return sum\n}\n"
  testify.Equal(t, canonical, formatted(t, canonical), "the canonical form formats as itself")
  testify.Equal(
    t,
    "package one\n\nconst VALUE = 0xFF\n",
    formatted(t, "package one\n\nconst VALUE = 0XFF\n"),
    "a literal the source spelled loosely formats in the canonical form",
  )
  testify.Equal(
    t,
    "package one\n\nfunc Fold() {\n  return\n}\n",
    formatted(t, "package one\nfunc Fold() {\nreturn\n}\n"),
    "a source the author wrote flat formats with the lines and tabs the form states",
  )
}

func test_clean(t *testing.T) {
  testify.True(t, bool(cleanliness(t, "package one\n")),
    "a source in the canonical form is clean")
  testify.False(
    t,
    bool(cleanliness(t, "package one\n\nconst VALUE = 0XFF\n")),
    "a source stating a literal loosely is no clean source",
  )
  testify.False(
    t,
    bool(cleanliness(t, "package one\nconst VALUE = 1\n")),
    "a source missing the empty line one declaration opens with is no clean source",
  )
  testify.False(
    t,
    bool(cleanliness(t, "package one\n\nconst VALUE = 1")),
    "a source closing on no line feed is no clean source",
  )
}

func test_refusals(t *testing.T) {
  state := format_state()
  subject := printer_state()
  tree := parse_state()
  source := token.Source("package one\n\ntype Count int\n")
  ast.Parse(tree, source)
  for _, width_size := range []int{0, 1, 2, 4} {
    narrow := make([]byte, width_size)
    result := format.Format(state, subject, narrow, tree, source)
    testify.False(t, bool(result.OK), "a form past the storage is refused")
    testify.Equal(
      t,
      printer.Form_Count(width_size),
      result.Count,
      "a refused format states the storage it filled",
    )
    clean := format.Clean(state, subject, narrow, tree, source)
    testify.False(t, bool(clean.OK), "a clean report past the storage is refused")
    testify.False(t, bool(clean.Clean),
      "a source the format refused is no clean source")
  }
  // A tree the parser refused states the declarations it holds, thus the caller reads a
  // partial form rather than nothing at all.
  broken := token.Source("package one\n\nfunc (\n")
  ast.Parse(tree, broken)
  wide := make([]byte, printer.FORM_SIZE_MAXIMUM)
  result := format.Format(state, subject, wide, tree, broken)
  testify.True(t, bool(result.OK), "a refused parse formats the tree it holds")
  for _, text := range refused_sources() {
    refused := token.Source(text)
    ast.Parse(tree, refused)
    testify.True(
      t,
      bool(format.Declare(state, tree, refused)),
      "a refused parse declares the names its tree holds",
    )
    testify.True(
      t,
      bool(format.Format(state, subject, wide, tree, refused).OK),
      "a refused parse formats inside the widest storage",
    )
    clean := format.Clean(state, subject, wide, tree, refused)
    testify.True(t, bool(clean.OK), "a refused parse reports on its form")
  }
  // A source of a few bytes states no file at all, thus the format writes the clause those
  // bytes name and states that no such source is clean.
  for _, text := range []string{"", "p", "pa"} {
    narrow := token.Source(text)
    ast.Parse(tree, narrow)
    narrow_result := format.Format(state, subject, wide, tree, narrow)
    testify.True(t, bool(narrow_result.OK),
      "a format of a source of a few bytes holds its form")
    testify.True(
      t,
      !(narrow_result.Count > printer.Form_Count(len(text)+9)),
      "a source of a few bytes writes the clause those bytes name and no more",
    )
    clean := format.Clean(state, subject, wide, tree, narrow)
    testify.True(t, bool(clean.OK), "a clean report of a few bytes holds its form")
    testify.False(t, bool(clean.Clean), "a source stating no file is no clean source")
  }
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
  // A space states no token and closes no line, thus the padding that carries the source to
  // the widest admitted size costs the parse nothing.
  for len(body) < token.SOURCE_SIZE_MAXIMUM {
    body = append(body, ' ')
  }
  return string(body)
}

func test_bounds(t *testing.T) {
  state := format_state()
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  source := token.Source(wide_source())
  testify.Equal(t, token.SOURCE_SIZE_MAXIMUM, len(source),
    "the source states the widest size")
  ast.Parse(tree, source)
  result := format.Format(state, subject, storage, tree, source)
  testify.False(t, bool(result.OK), "a form past the widest form is refused")
  testify.Equal(
    t,
    printer.Form_Count(printer.FORM_SIZE_MAXIMUM),
    result.Count,
    "one format writes at most the widest form",
  )
  // The storage the clean report reads is narrow on purpose: a report the storage refuses
  // answers at the first bytes it writes, thus the widest source costs one form and not two.
  narrow := make([]byte, 4)
  clean := format.Clean(state, subject, narrow, tree, source)
  testify.False(t, bool(clean.OK), "a clean report of a form past the storage is refused")
  testify.False(t, bool(clean.Clean),
    "a source whose form was never written is no clean source")
  // The destination is caller storage of any width, thus each width the domain names
  // reaches every step the sort holds. A corpus of one width states nothing about the rest.
  held := token.Source("package one\n\nimport \"zz\"\n\n// note\nvar fooBar = 1\n")
  held_subject := printer_state()
  held_tree := parse_state()
  for _, width_size := range []int{0, 1, 2, printer.FORM_SIZE_MAXIMUM} {
    room := make([]byte, width_size)
    held_state := format_state()
    ast.Parse(held_tree, held)
    format.Declare(held_state, held_tree, held)
    written := format.Format(held_state, held_subject, room, held_tree, held)
    testify.Equal(
      t,
      width_size == printer.FORM_SIZE_MAXIMUM,
      bool(written.OK),
      "only storage the whole form fits holds it",
    )
    format.Clean(held_state, held_subject, room, held_tree, held)
  }
  // The source a declaration reads is caller storage of any width too, thus each width the
  // domain names reaches the steps that take it.
  room := make([]byte, printer.FORM_SIZE_MAXIMUM)
  // One state carries every width, thus the source one declaration already read stands in
  // the storage the next declaration is handed.
  wide_state := format_state()
  for _, one := range []token.Source{
    token.Source(""),
    token.Source("a"),
    token.Source("ab"),
    token.Source(wide_source()),
  } {
    ast.Parse(held_tree, one)
    format.Declare(wide_state, held_tree, one)
    format.Format(wide_state, held_subject, room, held_tree, one)
    format.Clean(wide_state, held_subject, room, held_tree, one)
  }
  shortest_form_cases(t)
  widest_word_case(t)
}

// States the shortest storage each step of the format reads, which is how a floor a step
// declares stands observed and not merely stated.
func shortest_form_cases(t *testing.T) {
  // The shortest file the printer writes is the shortest storage the sort reads, and a name
  // of one letter is the shortest word a search of the table looks for.
  for _, one := range []token.Source{
    token.Source("package a\n"),
    token.Source("package a\n\nvar b = 1\n\nvar cd = 2\n"),
    token.Source("package a\n\nimport \"z\"\n"),
    token.Source("package a\n\nimport \"y\"\nimport \"z\"\n"),
  } {
    held_state := format_state()
    held_subject := printer_state()
    held_tree := parse_state()
    ast.Parse(held_tree, one)
    format.Declare(held_state, held_tree, one)
    short := make([]byte, len(one))
    written := format.Format(held_state, held_subject, short, held_tree, one)
    testify.True(t, bool(written.OK), "a form the source spells fits the source width")
    testify.Equal(
      t,
      string(one),
      string(short[:written.Count]),
      "a source already in the canonical form formats as itself",
    )
  }
  // The shortest declaration the table takes spells a name of two letters, and the rename
  // spells it one byte wider, thus storage of the source width holds every byte but the
  // last and the rename reads the storage it was handed all the same.
  grown := token.Source("package a\n\nvar aB = 1\n")
  grown_state := format_state()
  grown_subject := printer_state()
  grown_tree := parse_state()
  ast.Parse(grown_tree, grown)
  format.Declare(grown_state, grown_tree, grown)
  short := make([]byte, len(grown))
  written := format.Format(grown_state, grown_subject, short, grown_tree, grown)
  testify.False(t, bool(written.OK), "a rename one byte past the storage is refused")
}

// States the widest word a search of the table looks for, which is the widest name one source
// spells with nothing around it but the shortest text a declaration needs.
func widest_word_case(t *testing.T) {
  name := make([]byte, 0, format.WORD_SIZE_MAXIMUM)
  name = append(name, "aB"...)
  for len(name) < format.WORD_SIZE_MAXIMUM {
    name = append(name, 'c')
  }
  source := token.Source("package a\n\nvar " + string(name) + " = 1\n")
  testify.Equal(t, token.SOURCE_SIZE_MAXIMUM, len(source),
    "the source states the widest size")
  state := format_state()
  subject := printer_state()
  tree := parse_state()
  room := make([]byte, printer.FORM_SIZE_MAXIMUM)
  ast.Parse(tree, source)
  format.Declare(state, tree, source)
  written := format.Format(state, subject, room, tree, source)
  testify.True(t, bool(written.OK), "the widest name renames inside the widest form")
}

func test_allocation(t *testing.T) {
  held := allocation_fixture{}
  state := format_state()
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  source := token.Source(
    "package one\n\ntype Count int\n\nfunc Fold() (sum Count) {\n" +
      "  return sum\n}\n",
  )
  ast.Parse(tree, source)
  checks := []struct {
    Name string
    Call func()
  }{
    {Name: "Format", Call: func() {
      allocation_print(&held, format.Format(state, subject, storage, tree, source))
    }},
    {Name: "Clean", Call: func() {
      allocation_clean(&held, format.Clean(state, subject, storage, tree, source))
    }},
  }
  for _, check := range checks {
    t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
  }
  testify.True(t, bool(held.Ok), "the allocation fixture formats its file")
  testify.True(t, bool(held.Clean), "the allocation fixture states a clean source")
}

func allocation_print(held *allocation_fixture, result printer.Print_Result) {
  held.Count, held.Ok = result.Count, bool(result.OK)
}

func allocation_clean(held *allocation_fixture, result format.Clean_Result) {
  held.Clean, held.Ok = result.Clean, bool(result.OK)
}

func test_imports(t *testing.T) {
  for _, one := range []struct {
    Source string
    Form   string
  }{
    {
      Source: "package one\n\nimport \"zz\"\nimport \"aa\"\n",
      Form:   "package one\n\nimport \"aa\"\nimport \"zz\"\n",
    },
    {
      Source: "package one\n\nimport \"aa\"\nimport \"zz\"\n",
      Form:   "package one\n\nimport \"aa\"\nimport \"zz\"\n",
    },
    {
      Source: "package one\n\nimport held \"zz\"\nimport \"aa\"\n",
      Form:   "package one\n\nimport \"aa\"\nimport \"zz\"\n",
    },
    {
      Source: "package one\n\nimport \"zz\" // A note stands behind it.\n" +
        "import \"aa\"\n",
      Form: "package one\n\nimport \"aa\"\n" +
        "import \"zz\" // A note stands behind it.\n",
    },
    {
      Source: "package one\n\n// A note stands above it.\nimport \"zz\"\n" +
        "import \"aa\"\n",
      Form: "package one\n\nimport \"aa\"\n// A note stands above it.\n" +
        "import \"zz\"\n",
    },
    {
      Source: "package one\n\nimport \"zz\"\nimport \"mm\"\n\nimport \"bb\"\n" +
        "import \"aa\"\n",
      Form: "package one\n\nimport \"aa\"\nimport \"bb\"\nimport \"mm\"\n" +
        "import \"zz\"\n",
    },
    {
      Source: "package one\n\nimport \"other.example/held\"\n\n" +
        "import \"unsafe\"\n" +
        "import \"local/james-orcales/shared/std/go/ast\"\nimport \"fmt\"\n",
      Form: "package one\n\nimport \"fmt\"\nimport \"unsafe\"\n\n" +
        "import \"local/james-orcales/shared/std/go/ast\"\n\n" +
        "import \"other.example/held\"\n",
    },
    {
      Source: "package one\n\nimport \"local/james-orcales/shared/std/go/ast\"\n" +
        "import \"errors\"\n",
      Form: "package one\n\nimport \"errors\"\n\n" +
        "import \"local/james-orcales/shared/std/go/ast\"\n",
    },
    {
      Source: "package one\n\nimport \"zz\"\n\nvar one = 1\n",
      Form:   "package one\n\nimport \"zz\"\n\nvar one = 1\n",
    },
  } {
    testify.Equal(
      t,
      one.Form,
      formatted(t, one.Source),
      "the imports of one run stand in the order of the paths they name",
    )
  }
}

// Builds one source that declares a run of long names in the wrong case, each distinct by the
// number that closes it, thus a few of them fill the name table past its storage.
func long_names_source(first int) (source string) {
  body := make([]byte, 0, token.SOURCE_SIZE_MAXIMUM)
  body = append(body, "package one\n\n"...)
  // Each line spells six digits, thus every line stands the same size and no two names clash.
  line_size := len("var aB") + 200 + 6 + len(" = 1\n")
  for index := first; len(body)+line_size < token.SOURCE_SIZE_MAXIMUM; index++ {
    body = append(body, "var aB"...)
    for range 200 {
      body = append(body, 'c')
    }
    divisor := 100000
    for range 6 {
      body = append(body, byte('0'+index/divisor%10))
      divisor = divisor / 10
    }
    body = append(body, " = 1\n"...)
  }
  // A space states no token, thus the padding that carries the source to the widest admitted
  // size declares nothing.
  for len(body) < token.SOURCE_SIZE_MAXIMUM {
    body = append(body, ' ')
  }
  return string(body)
}

func test_declarations(t *testing.T) {
  renames_widths(t)
  state := format_state()
  subject := printer_state()
  tree := parse_state()
  storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
  declared := token.Source("package one\n\nvar someValue = 1\n\nconst maxSize = 2\n")
  ast.Parse(tree, declared)
  testify.True(
    t,
    bool(format.Declare(state, tree, declared)),
    "a table with room takes every name of the file",
  )
  source := token.Source("package one\n\nvar held = someValue + maxSize\n")
  ast.Parse(tree, source)
  result := format.Format(state, subject, storage, tree, source)
  testify.True(t, bool(result.OK), "the format holds the whole form")
  testify.Equal(
    t,
    "package one\n\nvar held = some_value + MAX_SIZE\n",
    string(storage[:result.Count]),
    "a name another file declares takes its case in every file that spells it",
  )
  format.Forget(state)
  source = token.Source("package one\n\nvar heldValue = 1\n")
  ast.Parse(tree, source)
  result = format.Format(state, subject, storage, tree, source)
  testify.Equal(
    t,
    string(source),
    string(storage[:result.Count]),
    "a name no declaration took stands as written, though the file declares it",
  )
  format.Forget(state)
  ast.Parse(tree, source)
  result = format.Format(state, subject, storage, tree, source)
  testify.Equal(
    t,
    string(source),
    string(storage[:result.Count]),
    "a forgotten table renames nothing the file does not declare",
  )
  full := true
  for first := 0; full; first = first + 100000 {
    many := token.Source(long_names_source(first))
    ast.Parse(tree, many)
    full = bool(format.Declare(state, tree, many))
  }
  testify.False(t, full, "a table past its storage reports it")
  format.Forget(state)
  for _, text := range []string{"", "p", "pa"} {
    narrow := token.Source(text)
    ast.Parse(tree, narrow)
    testify.True(
      t,
      bool(format.Declare(state, tree, narrow)),
      "a source of a few bytes declares nothing",
    )
    format.Forget(state)
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
    string(nested),
    string(filled),
  }
}

// A caller renames a form of its own, thus the rename answers for every width a form and a count
// state: the narrowest storage, the widest, and the counts a form of one byte and of two hold.
func renames_widths(t *testing.T) {
  state := format_state()
  // The source one format declared states the names the rename spells, thus the rename
  // answers for a source of one byte and of two the way it answers for a whole file.
  tree := parse_state()
  for _, text := range []string{"", "p", "pa"} {
    held := token.Source(text)
    ast.Parse(tree, held)
    format.Declare(state, tree, held)
    format.Rename(state, make(printer.Form, printer.FORM_SIZE_MINIMUM+1), 1)
  }
  format.Forget(state)
  for _, size := range []int{
    printer.FORM_SIZE_MINIMUM,
    printer.FORM_SIZE_MINIMUM + 1,
    printer.FORM_SIZE_MINIMUM + 2,
    printer.FORM_SIZE_MAXIMUM,
  } {
    storage := make(printer.Form, size)
    for _, count := range []int{0, 1, 2, size} {
      held := format.Rename(state, storage, printer.Form_Count(count))
      testify.True(t, bool(held.OK), "a rename of no name holds the form")
      testify.Equal(
        t,
        printer.Form_Count(count),
        held.Count,
        "a rename of no name hands back the form it read",
      )
    }
  }
}
