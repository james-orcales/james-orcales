package naming_test

import "testing"

import "local/james-orcales/harness/pkg/configuration"
import "local/james-orcales/harness/pkg/naming"
import "local/james-orcales/shared/bytes"
import "local/james-orcales/shared/go/ast"
import "local/james-orcales/shared/go/token"
import "local/james-orcales/shared/testify"

// Test_Naming_Name_Style verifies a name splits into words as lint splits it, and a rename keeps
// the case of the name it renames.
func Test_Naming_Name_Style(t *testing.T) {
  subject := subject_new(t)
  expect(
    t,
    subject,
    "package a\n\nvar Http_Id = 1\n",
    "Http_Id",
    "Rename Http_Id -> Http_Identifier.",
  )
  expect(
    t,
    subject,
    "package a\n\nvar HTTP_Id = 1\n",
    "HTTP_Id",
    "Rename HTTP_Id -> HTTP_Identifier.",
  )
  expect(
    t,
    subject,
    "package a\n\nconst ID_MAX = 1\n",
    "ID_MAX",
    "Rename ID_MAX -> Identifier_MAX.",
  )
  expect(
    t,
    subject,
    "package a\n\nvar user_id = 1\n",
    "user_id",
    "Rename user_id -> user_identifier.",
  )
  expect(t, subject, "package a\n\nvar userId = 1\n", "userId", "Rename userId -> user_identifier.")
  expect(t, subject, "package a\n\nvar v2Id = 1\n", "v2Id", "Rename v2Id -> v2_identifier.")
}

// Test_Naming_Full_Words verifies each declared name, the package name, and the file name stem
// state the replacements the table holds, and a used name states none.
func Test_Naming_Full_Words(t *testing.T) {
  subject := subject_new(t)
  expect(t, subject, "package buf\n", "buf", "Rename buf -> buffer.")
  expect(
    t,
    subject,
    "package a\n\nfunc act_now() {}\n",
    "act_now",
    "Rename act_now -> [action_now, active_now, actual_now].",
  )
  declared := []Declared{
    {Source: "package a\n\nfunc (buf *T) F() {}\n", Name: "buf"},
    {Source: "package a\n\nfunc F(a int, buf int) {}\n", Name: "buf"},
    {Source: "package a\n\nfunc F() (buf int) {\n  return 0\n}\n", Name: "buf"},
    {Source: "package a\n\ntype Buf int\n", Name: "Buf"},
    {Source: "package a\n\ntype T struct {\n  A, Buf int\n}\n", Name: "Buf"},
    {Source: "package a\n\nconst A, BUF = 1, 2\n", Name: "BUF"},
    {Source: "package a\n\nvar a, buf int = 1, 2\n", Name: "buf"},
    {Source: "package a\n\nfunc F() {\n  a, buf := 1, 2\n}\n", Name: "buf"},
    {Source: "package a\n\nfunc F() {\n  for _, buf := range a {\n  }\n}\n", Name: "buf"},
    {Source: "package a\n\nfunc F() {\n  if buf := f(); buf {\n  }\n}\n", Name: "buf"},
  }
  for _, entry := range declared {
    rename := "Rename buf -> buffer."
    if entry.Name == "Buf" {
      rename = "Rename Buf -> Buffer."
    }
    if entry.Name == "BUF" {
      rename = "Rename BUF -> Buffer."
    }
    expect(t, subject, entry.Source, entry.Name, rename)
  }
  for _, source := range []string{
    "package a\n\nvar a = buf\n",
    "package a\n\nvar a buf\n",
    "package a\n\nfunc F() {\n  a, b := buf, 2\n}\n",
    "package a\n\nfunc F() {\n  f(func(buf int) {})\n}\n",
    "package a\n\ntype F func(buf int)\n",
    "package a\n\nfunc F() {\n  type buf int\n}\n",
    "package a\n\nfunc F[Buf any]() {}\n",
    "package a\n\nvar _ = 1\n",
  } {
    expect(t, subject, source, "", "")
  }
  subject.Cells[naming.CELL_PATH_SIZE] = naming.State_Cell(copy(subject.Path, "a/cfg_buf_test.go"))
  expect(t, subject, "package a\n", "", "Rename cfg_buf -> configuration_buf.")
}

// Test_Naming_Noun_Names verifies a declared name ending in a participle states it, and a listed
// noun does not.
func Test_Naming_Noun_Names(t *testing.T) {
  subject := subject_new(t)
  message := "The name \"parsing\" is a present participle. Rename it to a noun form."
  expect(t, subject, "package a\n\nvar Parsing = 1\n", "Parsing", message)
  expect(t, subject, "package a\n\nfunc parsing() {}\n", "parsing", message)
  for _, source := range []string{
    "package a\n\nvar heading_string = 1\n",
    "package a\n\nvar parsing_state = 1\n",
    "package parsing\n",
  } {
    expect(t, subject, source, "", "")
  }
}

// Test_Naming_Quantity_Suffixes verifies each piece of evidence asks for its suffix, and a name
// that carries one of them states nothing.
func Test_Naming_Quantity_Suffixes(t *testing.T) {
  subject := subject_new(t)
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  items := len(a)\n}\n",
    "items",
    "The name items has the role count or size. Rename items -> items_count.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  Mark_Size := strings.Index(a, b)\n}\n",
    "Mark_Size",
    "The name Mark_Size has the role offset. Rename Mark_Size -> Mark_Offset.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  runes := utf8.RuneCount(a)\n}\n",
    "runes",
    "The name runes has the role count. Rename runes -> runes_count.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  held := a.b.Cap()\n}\n",
    "held",
    "The name held has the role size. Rename held -> held_size.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  for i := 0; i < n; i++ {\n  }\n}\n",
    "i",
    "The name i has the role index. Rename i -> i_index.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F(n int) {\n  a := make([]byte, n)\n}\n",
    "n",
    "The name n has the role size. Rename n -> n_size.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  var n = 1\n  a := make(map[int]int, n)\n}\n",
    "n",
    "The name n has the role count. Rename n -> n_count.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  size_items := len(a)\n}\n",
    "size_items",
    "The name size_items has the role count or size. Rename size_items -> count_items.",
  )
  for _, source := range []string{
    "package a\n\nfunc F() {\n  item_size := len(a)\n}\n",
    "package a\n\nfunc F() {\n  a, b := len(a), 1\n}\n",
    "package a\n\nfunc F() {\n  _ = len(a)\n}\n",
    "package a\n\nfunc F() {\n  for i := 0; i < n; i += 1 {\n  }\n}\n",
    "package a\n\nfunc F() {\n  for i := a; i < n; i++ {\n  }\n}\n",
    "package a\n\nfunc F() {\n  a := make(T, n)\n}\n",
    "package a\n\nfunc F(n_count int, n_size int) {\n  a, b := make([]int, n_count), make([]byte, n_size)\n}\n",
    "package a\n\nvar items = len(a)\n",
  } {
    expect(t, subject, source, "", "")
  }
}

// Test_Naming_Arithmetic_Suffixes verifies an incoherent sum states its operands, and a sum lint
// admits or of a name of no quantity states nothing.
func Test_Naming_Arithmetic_Suffixes(t *testing.T) {
  subject := subject_new(t)
  expect(
    t,
    subject,
    "package a\n\nvar a = b_offset + c_count\n",
    "b_offset",
    "The arithmetic _offset + _count is not coherent. Use operands with compatible suffixes.",
  )
  expect(
    t,
    subject,
    "package a\n\nvar a = b_index + c_index\n",
    "b_index",
    "The arithmetic _index + _index is not coherent. Use operands with compatible suffixes.",
  )
  for _, source := range []string{
    "package a\n\nvar a = b_offset + 1\n",
    "package a\n\nvar a = b_offset * c_count\n",
    "package a\n\nvar a = b.c_offset + c_count\n",
    "package a\n\nvar a = b_count + c_count\n",
  } {
    expect(t, subject, source, "", "")
  }
}

// Test_Naming_Arithmetic_Results verifies a name := takes from an admitted sum states the result
// it must end in, and one ending in it states nothing.
func Test_Naming_Arithmetic_Results(t *testing.T) {
  subject := subject_new(t)
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  gap := b_index - c_index\n}\n",
    "gap",
    "The arithmetic gap = _index - _index gives the wrong suffix. End the name with _count.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  end_size := b_offset + c_size\n}\n",
    "end_size",
    "The arithmetic end_size = _offset + _size gives the wrong suffix. End the name with _offset.",
  )
  for _, source := range []string{
    "package a\n\nfunc F() {\n  gap_count := b_index - c_index\n}\n",
    "package a\n\nfunc F() {\n  end_offset := c_size + b_offset\n}\n",
    "package a\n\nfunc F() {\n  a_index := b_count + c_index\n}\n",
    "package a\n\nfunc F() {\n  a_size := b_offset - c_offset\n}\n",
    "package a\n\nfunc F() {\n  a_count := b_count - c_count\n}\n",
    "package a\n\nfunc F() {\n  a := b_offset - 1\n}\n",
  } {
    expect(t, subject, source, "", "")
  }
}

// Test_Naming_Extremum_Suffixes verifies a max or min before the last word moves to the end, and
// a last one stands.
func Test_Naming_Extremum_Suffixes(t *testing.T) {
  subject := subject_new(t)
  expect(t, subject, "package a\n\nvar max_line = 1\n", "max_line", "Rename max_line -> line_max.")
  expect(
    t,
    subject,
    "package a\n\nconst RETRY_MIN_COUNT = 1\n",
    "RETRY_MIN_COUNT",
    "Rename RETRY_MIN_COUNT -> RETRY_COUNT_MIN.",
  )
  expect(t, subject, "package a\n\nvar line_max = 1\n", "", "")
}

// Test_Naming_Banned_Words verifies a table word of no replacement states the ban, and a use of
// the builtin states nothing.
func Test_Naming_Banned_Words(t *testing.T) {
  subject := subject_new(t)
  expect(
    t,
    subject,
    "package a\n\nvar name_len = 1\n",
    "name_len",
    "The identifier \"name_len\" contains the banned substring \"len\". Write a different name.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  var Util_Table = 1\n}\n",
    "Util_Table",
    "The identifier \"Util_Table\" contains the banned substring \"util\". Write a different name.",
  )
  expect(t, subject, "package a\n\nvar a = len(b)\n", "", "")
}

// Test_Naming_Import_Aliases verifies an alias holding default in any case states it, whatever
// other import names its package.
func Test_Naming_Import_Aliases(t *testing.T) {
  subject := subject_new(t)
  expect(
    t,
    subject,
    "package a\n\nimport Default_X \"b/c\"\n",
    "Default_X",
    "The import alias \"Default_X\" holds \"default\". Name the alias for the package. "+
      "A default directory declares the parent as that package.",
  )
  expect(
    t,
    subject,
    "package a\n\nimport x_DEFAULT \"b/c\"\nimport \"d/c\"\n",
    "x_DEFAULT",
    "The import alias \"x_DEFAULT\" holds \"default\". Name the alias for the package. "+
      "A default directory declares the parent as that package.",
  )
  expect(t, subject, "package a\n\nimport defaul \"b/c\"\nimport \"d/c\"\n", "", "")
}

// Test_Naming_Unnecessary_Aliases verifies an alias no other import of its package asks for
// states it, and one another import of the package asks for stands.
func Test_Naming_Unnecessary_Aliases(t *testing.T) {
  subject := subject_new(t)
  expect(
    t,
    subject,
    "package a\n\nimport \"b/d\"\nimport x \"b/c\"\n",
    "x",
    "Package import alias \"x\" is unnecessary. Remove alias because no other imported package "+
      "is named \"c\".",
  )
  expect(
    t,
    subject,
    "package a\n\nimport x \"b/c/default\"\n",
    "x",
    "Package import alias \"x\" is unnecessary. Remove alias because no other imported package "+
      "is named \"c\".",
  )
  expect(t, subject, "package a\n\nimport \"d/c/default\"\nimport x \"b/c\"\n", "", "")
  expect(t, subject, "package a\n\nimport x \"b/c\"\nimport y \"d/c\"\n", "", "")
}

// Test_Naming_Shadows verifies each declaration form that hides an outer name states it at the
// inner name, and a name no outer scope declares states nothing.
func Test_Naming_Shadows(t *testing.T) {
  subject := subject_new(t)
  message := "The variable \"value\" shadows a variable in an outer scope. Write a different name."
  for _, source := range []string{
    "package a\n\nfunc F(value int) {\n  {\n    value := 1\n  }\n}\n",
    "package a\n\nfunc F() (value int) {\n  {\n    var value = 1\n  }\n  return 0\n}\n",
    "package a\n\nfunc F() {\n  value := 1\n  if value := f(); value {\n  }\n}\n",
    "package a\n\nfunc F() {\n  value := 1\n  for value := 0; value < 3; value += 1 {\n  }\n}\n",
    "package a\n\nfunc F() {\n  value := 1\n  for _, value := range a {\n  }\n}\n",
    "package a\n\nconst VALUE = 1\n\nvar value = 1\n\nfunc F() {\n  value := 1\n}\n",
    "package a\n\nfunc value() {}\n\nfunc F() {\n  value := 1\n}\n",
  } {
    subject_second(t, subject, source, "value", message)
  }
  subject_second(
    t,
    subject,
    "package a\n\nconst VALUE = 1\n\nfunc F() {\n  const VALUE = 2\n}\n",
    "VALUE",
    "The variable \"VALUE\" shadows a variable in an outer scope. Write a different name.",
  )
  for _, source := range []string{
    "package a\n\nfunc F(value int) {\n  value := 1\n}\n",
    "package a\n\nfunc F() {\n  value := 1\n  value, other := 2, 3\n}\n",
    "package a\n\nfunc F() {\n  {\n    value := 1\n  }\n  value := 2\n}\n",
    "package a\n\nfunc (value T) F() {\n  {\n    value := 1\n  }\n}\n",
    "package a\n\nfunc F() {\n  value := 1\n}\n\nfunc G() {\n  value := 1\n}\n",
  } {
    expect(t, subject, source, "", "")
  }
}

// Test_Naming_Scopes verifies an if chain, a for, and a range each open a scope of their own,
// siblings share none, and a switch, a select, and a closure open none Check reads.
func Test_Naming_Scopes(t *testing.T) {
  subject := subject_new(t)
  message := "The variable \"value\" shadows a variable in an outer scope. Write a different name."
  for _, source := range []string{
    "package a\n\nfunc F() {\n  if f() {\n    value := 1\n  } else {\n    value := 2\n  }\n}\n",
    "package a\n\nfunc F() {\n  if value := f(); value {\n  } else if value := g(); value {\n  }\n}\n",
    "package a\n\nfunc F() {\n  if value := f(); value {\n    value := 1\n  }\n}\n",
    "package a\n\nfunc F() {\n  switch f() {\n  case 1:\n    value := 1\n  }\n  value := 2\n}\n",
    "package a\n\nfunc F(value int) {\n  switch f() {\n  case 1:\n    value := 1\n  }\n}\n",
    "package a\n\nfunc F(value int) {\n  g(func(value int) {})\n}\n",
  } {
    expect(t, subject, source, "", "")
  }
  subject_second(
    t,
    subject,
    "package a\n\nfunc F() {\n  if f() {\n  } else {\n    value := 1\n    {\n      value := 2\n    }\n  }\n}\n",
    "value",
    message,
  )
  subject_second(
    t,
    subject,
    "package a\n\nfunc F() {\n  for value := range a {\n    for _, value := range b {\n    }\n  }\n}\n",
    "value",
    message,
  )
}

// Test_Naming_Records verifies Record lists each name Shadows judged, then the top-level function,
// type, and constant names but a method's and a variable's.
func Test_Naming_Records(t *testing.T) {
  subject := subject_new(t)
  source := "package a\n\nconst LIMIT = 1\n\nvar hidden = 1\n\ntype T int\n\n" +
    "func (receiver T) Method() {}\n\n" +
    "func F(parameter int) {\n  local := 1\n  for index := 0; index < 3; index++ {\n  }\n}\n"
  result := ast.Parse(subject.State, token.Source(source))
  testify.True(t, bool(result.OK), source)
  copy(subject.Source, source)
  testify.False(t, bool(naming.Check(subject.Checker)), source)
  naming.Record(subject.Checker)
  recorded := []string{}
  for _, index := range subject.Records[:subject.Cells[naming.CELL_GRAPH_START]] {
    spelled := subject.State.Tokens.Values[subject.State.Nodes.Values[index].Token]
    recorded = append(recorded, source[spelled.Offset:int(spelled.Offset)+int(spelled.Size)])
  }
  testify.Equal(t, []string{"local", "index", "LIMIT", "T", "F"}, recorded)
  testify.Equal(t, naming.State_Cell(2), subject.Cells[naming.CELL_LOCAL_COUNT])
}

// Test_Naming_Calls verifies Record lists each function and method past the top-level names, each
// followed by the bare names its body calls, a closure's and a go statement's included.
func Test_Naming_Calls(t *testing.T) {
  subject := subject_new(t)
  source := "package a\n\ntype T int\n\nfunc (receiver T) Method() {\n  F()\n}\n\n" +
    "func F() {\n  go G()\n  receiver.Method()\n  H(func() {\n    F()\n  })\n}\n"
  result := ast.Parse(subject.State, token.Source(source))
  testify.True(t, bool(result.OK), source)
  copy(subject.Source, source)
  testify.False(t, bool(naming.Check(subject.Checker)), source)
  naming.Record(subject.Checker)
  graph := subject.Cells[naming.CELL_GRAPH_START]
  recorded := []string{}
  for _, index := range subject.Records[graph:subject.Cells[naming.CELL_RECORD_COUNT]] {
    spelled := subject.State.Tokens.Values[subject.State.Nodes.Values[index].Token]
    recorded = append(recorded, source[spelled.Offset:int(spelled.Offset)+int(spelled.Size)])
  }
  testify.Equal(t, []string{"Method", "F", "F", "G", "H", "F"}, recorded)
}

// Test_Naming_Impurity verifies Record lists each import, states the first clock selector and the
// first transitive one under the name each import binds, and counts the free func Main.
func Test_Naming_Impurity(t *testing.T) {
  subject := subject_new(t)
  source := "package a\n\nimport \"fmt\"\nimport clock \"time\"\nimport \"path/filepath\"\n\n" +
    "type T int\n\nfunc Main() {}\n\nfunc (receiver T) Main() {}\n\n" +
    "func F() {\n  resolve := filepath.Abs\n  clock.Now()\n  fmt.Println()\n}\n"
  result := ast.Parse(subject.State, token.Source(source))
  testify.True(t, bool(result.OK), source)
  copy(subject.Source, source)
  // The alias states its own naming finding, which leaves what Record lists as it stands.
  naming.Check(subject.Checker)
  naming.Record(subject.Checker)
  imports := []string{}
  start := subject.Cells[naming.CELL_IMPORT_START]
  for _, index := range subject.Records[start:subject.Cells[naming.CELL_RECORD_COUNT]] {
    imports = append(imports, node_text(subject, source, naming.State_Cell(index)))
  }
  testify.Equal(t, []string{"\"fmt\"", "clock", "\"path/filepath\""}, imports)
  cells := subject.Cells
  testify.Equal(t, "Now", node_text(subject, source, cells[naming.CELL_SOFT_CALL]))
  testify.Equal(t, "\"time\"", node_text(subject, source, cells[naming.CELL_SOFT_IMPORT]))
  testify.Equal(t, "Abs", node_text(subject, source, cells[naming.CELL_TRANSITIVE_CALL]))
  testify.Equal(
    t,
    "\"path/filepath\"",
    node_text(subject, source, cells[naming.CELL_TRANSITIVE_IMPORT]),
  )
  testify.Equal(t, naming.State_Cell(1), subject.Cells[naming.CELL_MAIN_COUNT])
}

// Test_Naming_Order verifies the stem comes before the file, the file states its first finding,
// and one name states its table word before its participle.
func Test_Naming_Order(t *testing.T) {
  subject := subject_new(t)
  expect(
    t,
    subject,
    "package a\n\nvar parsing = 1\n\nvar buf = 1\n",
    "parsing",
    "The name \"parsing\" is a present participle. Rename it to a noun form.",
  )
  expect(
    t,
    subject,
    "package a\n\nvar buf_parsing = 1\n",
    "buf_parsing",
    "Rename buf_parsing -> buffer_parsing.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  max_buf := len(a)\n}\n",
    "max_buf",
    "Rename max_buf -> max_buffer.",
  )
  expect(
    t,
    subject,
    "package a\n\nfunc F() {\n  max_parsing := len(a)\n}\n",
    "max_parsing",
    "The name max_parsing has the role count or size. Rename max_parsing -> max_parsing_count.",
  )
  subject.Cells[naming.CELL_PATH_SIZE] = naming.State_Cell(copy(subject.Path, "buf.go"))
  expect(t, subject, "package buf\n", "", "Rename buf -> buffer.")
}

// Test_Naming_Bounds verifies a message past the store stands cut, and a byte below a space in a
// replacement or an import path stands as ?.
func Test_Naming_Bounds(t *testing.T) {
  subject := subject_new(t)
  wide := repeat("a_", naming.MESSAGE_SIZE_MAXIMUM) + "buf"
  expect(
    t,
    subject,
    "package a\n\nvar "+wide+" = 1\n",
    wide,
    ("Rename " + wide + " -> ")[:naming.MESSAGE_SIZE_MAXIMUM-1],
  )
  words_parse(t, subject, `{"ab":["x\ny"]}`)
  expect(t, subject, "package a\n\nvar ab = 1\n", "ab", "Rename ab -> x?y.")
  expect(
    t,
    subject,
    "package a\n\nimport x \"b/c\x01\"\n",
    "x",
    "Package import alias \"x\" is unnecessary. Remove alias because no other imported package "+
      "is named \"c?\".",
  )
}

// WORDS is the word table the tests check against.
const WORDS = `{"len":[],"length":[],"util":[],"id":["identifier"],"buf":["buffer"],` +
  `"act":["action","active","actual"],"cfg":["configuration"]}`

// Declared is one source that declares a name the table replaces, and that name.
type Declared struct {
  // Source is the file.
  Source string
  // Name is the name it declares.
  Name string
}

// Subject is one naming check with the parse its source takes.
type Subject struct {
  *naming.Checker
  // State is the parse of the source.
  State *ast.Parse_State
}

// Allocates a naming check over the word table of the tests, at the path a.go.
func subject_new(t *testing.T) (subject Subject) {
  tree := ast.Syntax_Tree{}
  tree.Tokens.Values = make(ast.Tokens, ast.TOKEN_COUNT_MAXIMUM)
  tree.Nodes.Values = make(ast.Nodes, ast.NODE_COUNT_MAXIMUM)
  tree.Parents.Values = make(ast.Parents, ast.DEPTH_MAXIMUM)
  tree.Last_Children.Values = make(ast.Last_Children, ast.DEPTH_MAXIMUM)
  tree.Earlier_Children.Values = make(ast.Earlier_Children, ast.DEPTH_MAXIMUM)
  state := &ast.Parse_State{Syntax_Tree: tree}
  subject = Subject{
    Checker: &naming.Checker{
      Nodes:   naming.Node_View(state.Nodes.Values),
      Tokens:  naming.Token_View(state.Tokens.Values),
      Source:  make(naming.Source_Store, naming.SOURCE_STORAGE_SIZE),
      Path:    make(naming.Path_Store, naming.PATH_SIZE_MAXIMUM),
      Words:   make(naming.Word_Table, naming.WORD_SLOT_COUNT),
      Terms:   make(naming.Terms, ast.NODE_COUNT_MAXIMUM),
      Message: make(naming.Message_Store, naming.MESSAGE_SIZE_MAXIMUM),
      Records: make(naming.Record_Table, naming.RECORD_COUNT_MAXIMUM),
      Names:   make(naming.Scope_Names, ast.NODE_COUNT_MAXIMUM),
      Frames:  make(naming.Frame_Stack, naming.FRAME_COUNT),
      Cells:   make(naming.State_Cells, naming.CELL_COUNT),
    },
    State: state,
  }
  words_parse(t, subject, WORDS)
  subject.Cells[naming.CELL_PATH_SIZE] = naming.State_Cell(copy(subject.Path, "a.go"))
  return subject
}

// Decodes a configuration whose word table is the one given, and builds the table from it.
func words_parse(t *testing.T, subject Subject, words string) {
  decoded := &configuration.Configuration{
    Tokens: make(configuration.Tokens, configuration.TOKEN_COUNT_MAXIMUM),
    Text:   make(configuration.Text, configuration.SOURCE_SIZE_MAXIMUM),
    Values: make(configuration.Values, configuration.VALUE_COUNT_MAXIMUM),
    Cells:  make(configuration.Cells, configuration.CELL_COUNT),
  }
  source := `{"shared_component":"s","instrumentation_packages":[],"ignore":[],` +
    `"opt_out_assertion_mandate_packages":[],"opt_out_recursion_ban":[],` +
    `"opt_out_interface_ban":[],"word_replacements":` +
    words +
    `}`
  fault := configuration.Parse(decoded, configuration.Source(source))
  testify.Equal(t, configuration.FAULT_NONE, fault.Kind)
  subject.Values, subject.Text = naming.Decoded_Values(decoded.Values), naming.Decoded_Text(
    decoded.Text,
  )
  naming.Words_Build(subject.Words, decoded)
}

// Checks one source and expects the finding at the first spelling of the name, or at byte zero
// for no name, with the message; an empty message expects no finding.
func expect(t *testing.T, subject Subject, source string, name string, message string) {
  t.Helper()
  result := ast.Parse(subject.State, token.Source(source))
  testify.True(t, bool(result.OK), source)
  copy(subject.Source, source)
  found := naming.Check(subject.Checker)
  if message == "" {
    testify.False(t, bool(found), source)
    return
  }
  testify.True(t, bool(found), source)
  held := subject.Message[:bytes.Index_Byte(bytes.Slice(subject.Message), 0)]
  testify.Equal(t, message, string(held), source)
  offset := 0
  if name != "" {
    offset = offset_of(source, name)
  }
  testify.Equal(t, offset, int(subject.Cells[naming.CELL_OFFSET]), source)
}

// Checks one source and expects the finding at the second whole spelling of the name.
func subject_second(t *testing.T, subject Subject, source string, name string, message string) {
  t.Helper()
  result := ast.Parse(subject.State, token.Source(source))
  testify.True(t, bool(result.OK), source)
  copy(subject.Source, source)
  testify.True(t, bool(naming.Check(subject.Checker)), source)
  held := subject.Message[:bytes.Index_Byte(bytes.Slice(subject.Message), 0)]
  testify.Equal(t, message, string(held), source)
  first := offset_of(source, name)
  rest := source[first+len(name)-PACKAGE_CLAUSE_SIZE:]
  second := first + len(name) + offset_of(rest, name) - PACKAGE_CLAUSE_SIZE
  testify.Equal(t, second, int(subject.Cells[naming.CELL_OFFSET]), source)
}

// States the text of the token that names one node of the parse.
func node_text(subject Subject, source string, index naming.State_Cell) (text string) {
  held := subject.State.Tokens.Values[subject.State.Nodes.Values[index].Token]
  return source[held.Offset : int(held.Offset)+int(held.Size)]
}

// PACKAGE_CLAUSE_SIZE is the bytes a package clause spends before its name.
const PACKAGE_CLAUSE_SIZE = len("package ")

// States the byte the first whole spelling of a name stands at past the package clause.
func offset_of(source string, name string) (offset int) {
  for index := PACKAGE_CLAUSE_SIZE; index+len(name) < len(source)+1; index++ {
    if source[index:index+len(name)] != name {
      continue
    }
    if identifier_byte(source[index-1]) {
      continue
    }
    if index+len(name) < len(source) {
      if identifier_byte(source[index+len(name)]) {
        continue
      }
    }
    return index
  }
  return -1
}

// Reports whether a byte continues an identifier.
func identifier_byte(symbol byte) (held bool) {
  if symbol == '_' {
    return true
  }
  if symbol-'a' < 26 {
    return true
  }
  if symbol-'A' < 26 {
    return true
  }
  return symbol-'0' < 10
}

// Repeats one text a count of times.
func repeat(text string, count int) (repeated string) {
  built := make([]byte, 0, len(text)*count)
  for range count {
    built = append(built, text...)
  }
  return string(built)
}
