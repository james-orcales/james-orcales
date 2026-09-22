package types_test

import "fmt"
import "testing"

import "local/james-orcales/shared/std/go/ast"
import "local/james-orcales/shared/std/go/token"
import "local/james-orcales/shared/std/go/types"
import "local/james-orcales/shared/std/testify"

// Test_Module binds the Module specification leaf before fixture declarations.
func Test_Module(t *testing.T) {
  test_module(t)
}

// Test_Files binds the Files specification leaf before fixture declarations.
func Test_Files(t *testing.T) {
  test_files(t)
}

// Test_Universe binds the Universe specification leaf before fixture declarations.
func Test_Universe(t *testing.T) {
  test_universe(t)
}

// Test_Declarations binds the Declarations specification leaf before fixture declarations.
func Test_Declarations(t *testing.T) {
  test_declarations(t)
}

// Test_Type_Expressions binds the Type Expressions leaf before fixture declarations.
func Test_Type_Expressions(t *testing.T) {
  test_type_expressions(t)
}

// Test_Named_Types binds the Named Types specification leaf before fixture declarations.
func Test_Named_Types(t *testing.T) {
  test_named_types(t)
}

// Test_Type_Resolution binds the Type Resolution leaf before fixture declarations.
func Test_Type_Resolution(t *testing.T) {
  test_type_resolution(t)
}

// Test_Exported_Types binds the Exported Types leaf before fixture declarations.
func Test_Exported_Types(t *testing.T) {
  test_exported_types(t)
}

// Test_Exported_Struct_Fields binds the Exported Struct Fields leaf before fixture declarations.
func Test_Exported_Struct_Fields(t *testing.T) {
  test_exported_struct_fields(t)
}

// Test_Struct_Field_Type_Encapsulation binds the Struct Field Type Encapsulation leaf before
// fixture declarations.
func Test_Struct_Field_Type_Encapsulation(t *testing.T) {
  test_struct_field_type_encapsulation(t)
}

// Test_Bodies binds the Bodies specification leaf before fixture declarations.
func Test_Bodies(t *testing.T) {
  test_bodies(t)
}

// Test_Statements binds the Statements specification leaf before fixture declarations.
func Test_Statements(t *testing.T) {
  test_statements(t)
}

// Test_Hidden_Names binds the Hidden Names specification leaf before fixture declarations.
func Test_Hidden_Names(t *testing.T) {
  test_hidden_names(t)
}

// Test_Expressions binds the Expressions specification leaf before fixture declarations.
func Test_Expressions(t *testing.T) {
  test_expressions(t)
}

// Test_Answers binds the Answers specification leaf before fixture declarations.
func Test_Answers(t *testing.T) {
  test_answers(t)
}

// Test_Invariants binds the Invariants specification leaf before fixture declarations.
func Test_Invariants(t *testing.T) {
  test_invariant_pairs(t)
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

const TEST_PATH = "local/example/one"

// TEST_NAME_RUN_MAXIMUM caps the names one fill file states, because the token run of one file
// holds two tokens for each name and one run is bounded.
const TEST_NAME_RUN_MAXIMUM = 30000

// TEST_POINTER_RUN is how many pointers one fill declaration stands over, thus one declaration
// takes that many type slots and one symbol slot. A long run fills the type storage with few
// declarations, and it stays inside the depth one check admits.
const TEST_POINTER_RUN = 50

// TEST_TYPE_FREE is how many type slots one type head leaves free, which is the most one case
// spends on its tail.
const TEST_TYPE_FREE = 4

const TEST_OTHER_PATH = "local/example/two"

const TEST_OTHER_SOURCE = "package two\n\ntype Marker int\n\nconst MARK Marker = 1\n"

const TEST_THIRD_SOURCE = "package one\n\n// A names one byte of a name.\ntype A bool\n\n" +
  "type AB string\n\ntype Solo struct {\n\tOnly Count\n}\n\n" +
  "type Twin struct {\n\tLeft Count\n\tRight Count\n}\n"

const TEST_SOURCE = `package one

import two "local/example/two"

const LIMIT Count = 8

var table [LIMIT]Entry

var fixed [4]Count

// pointer stands over the type a field of the entry states.
var pointer *Entry

var flag *bool

var text *string

var truth map[bool]Count

var words map[string]Count

var slice []Entry

var lookup map[Count]Entry

var sender chan<- Count

var receiver <-chan Count

var pipe chan Count

var call func(one Count) (sum Count)

var wrapped (Count)

var outside two.Marker

var instance Holder[Count]

var first, second Count

var third = LIMIT

var fourth, fifth = LIMIT, LIMIT

type Count int

// Entry states the fields a caller reads back.
type Entry struct {
	Name  Count
	Value Count
	Base
}

type Base struct {
	Mark Count
}

type Numeric interface {
	~int | ~int64
}

type Plain interface {
	Count
}

type Holder[Item Numeric] struct {
	Value Item
}

func Total(entries []Entry, rest ...Count) (sum Count) {
	return sum
}

func Widest[Item Numeric](values []Item) (widest Item) {
	return values[0]
}

func (subject *Entry) Read() (value Count) {
	return subject.Value
}
`

const TEST_BODY_SOURCE = `package one

import two "local/example/two"

type Count int

func place_of(truth bool) (place Count) {
	return 1
}

type Held struct {
	Value Count
	Base
}

type Base struct {
	Mark Count
}

func (subject *Held) Read() (value Count) {
	return subject.Value
}

var shared Held

var flagged bool

var written string

var boxed any

func Fold(entries []Held, lookup map[Count]Held) (sum Count) {
	total := 0
	kept := shared.Value
	truth := flagged
	letters := written
	for place, one := range entries {
		total = total + int(one.Value) + place
	}
	held := Held{Value: 1}
	pointer := &held
	mark := held.Mark
	read := held.Read()
	outside := two.MARK
	text := "abc"
	letter := text[0]
	part := entries[1:2]
	count := len(entries)
	made := make([]Held, 4)
	fresh := new(Held)
	call := func(one Count) (two Count) {
		return one
	}
	folded := call(3)
	number := int64(count)
	found, ok := lookup[1]
	var stated Count = 5
	const LIMIT Count = 6
	deep := (mark)
	if ok {
		total = total + count
	}
	switch found.Value {
	case 1:
		total = total + 1
	}
	asserted := boxed.(Count)
	switch kind := boxed.(type) {
	case Count:
		total = total + int(kind)
	}
	grouped := new((Count))
	total = total + int(asserted) + int(*grouped)
	return place_of(true) + sum + mark + read + outside + Count(letter) + Count(len(part)) +
		Count(len(made)) + fresh.Value + Count(number) + folded + stated + LIMIT +
		deep + pointer.Value + Count(total) + kept + Count(len(letters)) + Count(place_of(truth))
}
`

type allocation_fixture struct {
  File          types.File_Index
  Symbol        types.Symbol_Index
  Type          types.Type_Index
  Ok            types.Boolean
  Failure       types.Failure_Code
  Message       types.Message
  Path          types.Path
  Package       types.Package_Index
  Source        token.Source
  Symbol_Kind   types.Symbol_Kind
  Name          types.Name
  Type_Kind     types.Type_Kind
  Type_Element  types.Type_Element
  Type_Key      types.Type_Key
  Element_Count types.Element_Count
  Type_Symbol   types.Type_Symbol
  Member_Head   types.Member_Head
  Member_Type   types.Member_Type
  Member_Symbol types.Member_Symbol
  Member_Next   types.Member_Successor
}

type fixture_file struct {
  Path   string
  Source string
}

// Caller allocation makes every module bound explicit in fixtures.
func module_state() (subject *types.Module) {
  return &types.Module{
    Files:      make(types.File_Storage, types.FILE_COUNT_MAXIMUM),
    Packages:   make(types.Package_Storage, types.PACKAGE_COUNT_MAXIMUM),
    Symbols:    make(types.Symbol_Storage, types.SYMBOL_COUNT_MAXIMUM),
    Types:      make(types.Type_Storage, types.TYPE_COUNT_MAXIMUM),
    Members:    make(types.Member_Storage, types.MEMBER_COUNT_MAXIMUM),
    Buckets:    make(types.Bucket_Storage, types.BUCKET_COUNT),
    Nodes:      make(types.Node_Storage, types.DEPTH_MAXIMUM),
    Parameters: make(types.Parameter_Storage, types.TYPE_PARAMETER_MAXIMUM),
    Owners:     make(types.Owner_Storage, types.DEPTH_MAXIMUM),
    Words:      make(types.Word_Storage, types.UNIVERSE_WORD_SIZE),
    Counts:     make(types.Count_Storage, types.COUNT_SLOT_COUNT),
  }
}

// Caller allocation makes every body bound explicit in fixtures.
func body_state() (body *types.Body) {
  return &types.Body{
    Types:  make(types.Body_Type_Storage, ast.NODE_COUNT_MAXIMUM),
    Locals: make(types.Local_Storage, types.LOCAL_COUNT_MAXIMUM),
    Staged: make(types.Staged_Storage, types.STAGED_NAME_MAXIMUM),
  }
}

// Caller allocation makes every parser bound explicit in fixtures.
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

// Builds one module and one tree the caller owns, which is how every case here starts.
func module_of() (subject *types.Module, tree *ast.Parse_State) {
  subject = module_state()
  types.Reset(subject)
  return subject, parse_state()
}

// Runs both passes over every file, which is the order the specification states.
func analyse(
  t *testing.T,
  subject *types.Module,
  tree *ast.Parse_State,
  files []fixture_file,
) (indexes []types.File_Index) {
  t.Helper()
  for _, one := range files {
    result := types.Add_File(subject, types.Path(one.Path), token.Source(one.Source))
    testify.True(t, bool(result.OK), "the file binds to its package")
    indexes = append(indexes, result.File)
  }
  trees := []*ast.Parse_State{tree}
  for slot, one := range files {
    if slot > 0 {
      trees = append(trees, parse_state())
    }
    ast.Parse(trees[slot], token.Source(one.Source))
    testify.True(
      t,
      bool(types.Declare(subject, indexes[slot], trees[slot])),
      "the file declares: %s",
      types.Failure_Message(types.Failure(subject)),
    )
  }
  for slot := range files {
    testify.True(
      t,
      bool(types.Resolve(subject, indexes[slot], trees[slot])),
      "the file resolves: %s",
      types.Failure_Message(types.Failure(subject)),
    )
  }
  return indexes
}

// Runs both passes over the two fixture files, which is what most cases here read.
func fixture(t *testing.T) (subject *types.Module, file types.File_Index) {
  t.Helper()
  subject, tree := module_of()
  indexes := analyse(
    t,
    subject,
    tree,
    []fixture_file{
      {Path: TEST_OTHER_PATH, Source: TEST_OTHER_SOURCE},
      {Path: TEST_PATH, Source: TEST_SOURCE},
      {Path: TEST_PATH, Source: TEST_THIRD_SOURCE},
    },
  )
  return subject, indexes[1]
}

// Names one package member, which is how a case states what it expects to find.
func member_of(
  t *testing.T,
  subject *types.Module,
  file types.File_Index,
  name string,
) (symbol types.Symbol_Index) {
  t.Helper()
  result := types.Lookup(subject, types.Package_Of(subject, file), types.Name(name))
  testify.True(t, bool(result.Found), "the package states %q", name)
  return result.Symbol
}

// Names one package member and reads the kind of the type it wears.
func type_kind_of(
  t *testing.T,
  subject *types.Module,
  file types.File_Index,
  name string,
) (kind types.Type_Kind) {
  t.Helper()
  symbol := member_of(t, subject, file, name)
  return types.Type_Kind_Of(subject, types.Symbol_Type_Of(subject, symbol))
}

func test_module(t *testing.T) {
  subject, file := fixture(t)
  testify.Equal(t, types.FAILURE_NONE, types.Failure(subject),
    "a clean module holds no fault")
  testify.Not_Equal(
    t,
    types.SYMBOL_ABSENT,
    member_of(t, subject, file, "Count"),
    "a clean module answers for a name it read",
  )
  types.Reset(subject)
  testify.Equal(t, types.FAILURE_NONE, types.Failure(subject),
    "a reset module holds no fault")
  found := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name("int"))
  testify.True(t, bool(found.Found), "a reset module still holds the universe")
  gone := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name("Count"))
  testify.False(t, bool(gone.Found), "a reset module holds no name a file stated")
}

func test_files(t *testing.T) {
  subject, tree := module_of()
  indexes := analyse(
    t,
    subject,
    tree,
    []fixture_file{
      {Path: TEST_PATH, Source: "package one\n\ntype Count int\n"},
      {Path: TEST_PATH, Source: "package one\n\ntype Second Count\n"},
      {Path: TEST_OTHER_PATH, Source: TEST_OTHER_SOURCE},
    },
  )
  testify.Equal(
    t,
    types.Package_Of(subject, indexes[0]),
    types.Package_Of(subject, indexes[1]),
    "two files of one path share one package",
  )
  testify.Not_Equal(
    t,
    types.Package_Of(subject, indexes[0]),
    types.Package_Of(subject, indexes[2]),
    "two paths make two packages",
  )
  testify.Equal(
    t,
    token.Source(TEST_OTHER_SOURCE),
    types.Source_Of(subject, indexes[2]),
    "a file reads back the source the caller owns",
  )
  second := member_of(t, subject, indexes[1], "Second")
  testify.Equal(
    t,
    types.TYPE_INT,
    types.Type_Kind_Of(subject,
      types.Underlying_Of(subject, types.Symbol_Type_Of(subject, second))),
    "a file reads a name another file of its package states",
  )
}

func test_universe(t *testing.T) {
  subject, _ := module_of()
  for _, name := range []string{
    "bool",
    "byte",
    "rune",
    "string",
    "int",
    "int8",
    "int16",
    "int32",
    "int64",
    "uint",
    "uint8",
    "uint16",
    "uint32",
    "uint64",
    "uintptr",
    "float32",
    "float64",
    "complex64",
    "complex128",
    "error",
    "any",
    "comparable",
  } {
    result := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name(name))
    testify.True(t, bool(result.Found), "the universe states the type %q", name)
    testify.Equal(
      t,
      types.SYMBOL_TYPE,
      types.Symbol_Kind_Of(subject, result.Symbol),
      "a predeclared type names a type",
    )
  }
  for _, name := range []string{
    "len",
    "cap",
    "make",
    "new",
    "append",
    "copy",
    "delete",
    "panic",
    "recover",
    "close",
    "min",
    "max",
    "clear",
    "complex",
    "real",
    "imag",
    "print",
    "println",
  } {
    result := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name(name))
    testify.True(t, bool(result.Found), "the universe states the function %q", name)
    testify.Equal(
      t,
      types.SYMBOL_BUILTIN,
      types.Symbol_Kind_Of(subject, result.Symbol),
      "a predeclared function names a builtin",
    )
  }
  for _, one := range []struct {
    Name string
    Kind types.Type_Kind
  }{
    {"true", types.TYPE_UNTYPED_BOOLEAN},
    {"false", types.TYPE_UNTYPED_BOOLEAN},
    {"iota", types.TYPE_UNTYPED_INTEGER},
    {"nil", types.TYPE_UNTYPED_NIL},
  } {
    result := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name(one.Name))
    testify.True(t, bool(result.Found), "the universe states the value %q", one.Name)
    testify.Equal(
      t,
      one.Kind,
      types.Type_Kind_Of(subject, types.Symbol_Type_Of(subject, result.Symbol)),
      "%q wears the type it folds to",
      one.Name,
    )
  }
  absent := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name("Total"))
  testify.False(t, bool(absent.Found), "the universe states no name a file declares")
}

func test_declarations(t *testing.T) {
  declared_tail_import(t)
  subject, file := fixture(t)
  for _, one := range []struct {
    Name string
    Kind types.Symbol_Kind
  }{
    {"LIMIT", types.SYMBOL_CONSTANT},
    {"table", types.SYMBOL_VARIABLE},
    {"first", types.SYMBOL_VARIABLE},
    {"second", types.SYMBOL_VARIABLE},
    {"third", types.SYMBOL_VARIABLE},
    {"fourth", types.SYMBOL_VARIABLE},
    {"fifth", types.SYMBOL_VARIABLE},
    {"Count", types.SYMBOL_TYPE},
    {"Entry", types.SYMBOL_TYPE},
    {"Holder", types.SYMBOL_TYPE},
    {"Total", types.SYMBOL_FUNCTION},
    {"Widest", types.SYMBOL_FUNCTION},
  } {
    symbol := member_of(t, subject, file, one.Name)
    testify.Equal(t, one.Kind, types.Symbol_Kind_Of(subject, symbol),
      "%q names its kind", one.Name)
    testify.Equal(
      t,
      types.Name(one.Name),
      types.Symbol_Name_Of(subject, symbol),
      "%q reads its own name",
      one.Name,
    )
    testify.Equal(
      t,
      file,
      types.Symbol_File_Of(subject, symbol),
      "%q names the file that states it",
      one.Name,
    )
  }
  absent := types.Lookup(subject, types.Package_Of(subject, file), types.Name("Read"))
  testify.False(t, bool(absent.Found), "a method binds no package name")
  unstated := types.Lookup(subject, types.Package_Of(subject, file), types.Name("Missing"))
  testify.False(t, bool(unstated.Found), "a package states no name no file declares")
}

func test_type_expressions(t *testing.T) {
  subject, file := fixture(t)
  for _, one := range []struct {
    Name string
    Kind types.Type_Kind
  }{
    {"pointer", types.TYPE_POINTER},
    {"slice", types.TYPE_SLICE},
    {"table", types.TYPE_ARRAY},
    {"fixed", types.TYPE_ARRAY},
    {"lookup", types.TYPE_MAP},
    {"pipe", types.TYPE_CHANNEL},
    {"sender", types.TYPE_CHANNEL},
    {"receiver", types.TYPE_CHANNEL},
    {"call", types.TYPE_FUNCTION},
    {"wrapped", types.TYPE_NAMED},
    {"outside", types.TYPE_NAMED},
    {"instance", types.TYPE_NAMED},
    {"first", types.TYPE_NAMED},
    {"second", types.TYPE_NAMED},
  } {
    testify.Equal(
      t,
      one.Kind,
      type_kind_of(t, subject, file, one.Name),
      "%q names its type",
      one.Name,
    )
  }
  fixed := types.Symbol_Type_Of(subject, member_of(t, subject, file, "fixed"))
  testify.Equal(
    t,
    types.Element_Count(4),
    types.Type_Element_Count_Of(subject, fixed),
    "an array names the element_count one literal states",
  )
  table := types.Symbol_Type_Of(subject, member_of(t, subject, file, "table"))
  testify.Equal(
    t,
    types.ELEMENT_COUNT_UNKNOWN,
    types.Type_Element_Count_Of(subject, table),
    "an array whose element_count no literal states names none",
  )
  lookup := types.Symbol_Type_Of(subject, member_of(t, subject, file, "lookup"))
  testify.Equal(
    t,
    types.TYPE_NAMED,
    types.Type_Kind_Of(subject, types.Type_Index(types.Type_Key_Of(subject, lookup))),
    "a map names the type of its key",
  )
  testify.Equal(
    t,
    types.TYPE_INVALID,
    type_kind_of(t, subject, file, "third"),
    "a declaration that states no type wears the invalid type",
  )
}

func test_named_types(t *testing.T) {
  subject, file := fixture(t)
  count := types.Symbol_Type_Of(subject, member_of(t, subject, file, "Count"))
  testify.Equal(
    t,
    types.TYPE_NAMED,
    types.Type_Kind_Of(subject, count),
    "a type declaration builds a named type",
  )
  testify.Equal(
    t,
    types.Name("Count"),
    types.Symbol_Name_Of(subject,
      types.Symbol_Index(types.Type_Symbol_Of(subject, count))),
    "a named type carries its declaration",
  )
  testify.Equal(
    t,
    types.TYPE_INT,
    types.Type_Kind_Of(subject, types.Underlying_Of(subject, count)),
    "a named type walks down to its own",
  )
  entry := types.Underlying_Of(
    subject,
    types.Symbol_Type_Of(subject, member_of(t, subject, file, "Entry")),
  )
  testify.Equal(
    t,
    types.TYPE_STRUCTURE,
    types.Type_Kind_Of(subject, entry),
    "a struct names a struct",
  )
  member := types.Member_Index(types.Type_Members_Of(subject, entry))
  testify.Equal(
    t,
    types.Name("Name"),
    types.Symbol_Name_Of(subject,
      types.Symbol_Index(types.Member_Symbol_Of(subject, member))),
    "a field names itself",
  )
  testify.Equal(
    t,
    types.TYPE_NAMED,
    types.Type_Kind_Of(subject,
      types.Type_Index(types.Member_Type_Of(subject, member))),
    "a field wears the type its declaration states",
  )
  embedded := types.Member_Index(
    types.Member_Next_Of(subject,
      types.Member_Index(types.Member_Next_Of(subject, member))),
  )

  testify.Equal(
    t,
    types.SYMBOL_ABSENT,
    types.Symbol_Index(types.Member_Symbol_Of(subject, embedded)),
    "an embedded field wears no name of its own",
  )
  named_grouped_fields(t)
  named_grouped_parameters(t)
}

// Reads the signature one grouped parameter list states, which takes one member for each name it
// spells and binds each name in the body.
func named_grouped_parameters(t *testing.T) {
  source := "package one\n\nfunc Pair(left, right *int) {\n\tread := right\n}\n"
  subject, tree := module_of()
  file := analyse(t, subject, tree, []fixture_file{{Path: TEST_PATH, Source: source}})[0]
  signature := types.Symbol_Type_Of(subject, member_of(t, subject, file, "Pair"))
  parameters := types.Type_Index(types.Type_Key_Of(subject, signature))
  left := types.Member_Index(types.Type_Members_Of(subject, parameters))
  right := types.Member_Index(types.Member_Next_Of(subject, left))
  for _, one := range []struct {
    Member types.Member_Index
    Name   types.Name
  }{{left, types.Name("left")}, {right, types.Name("right")}} {
    testify.Equal(
      t,
      one.Name,
      types.Symbol_Name_Of(subject,
        types.Symbol_Index(types.Member_Symbol_Of(subject, one.Member))),
      "a grouped parameter takes a member for the name %q",
      one.Name,
    )
    testify.Equal(
      t,
      types.TYPE_POINTER,
      types.Type_Kind_Of(subject,
        types.Type_Index(types.Member_Type_Of(subject, one.Member))),
      "the parameter %q wears the type its group states",
      one.Name,
    )
  }
  testify.Equal(
    t,
    types.TYPE_POINTER,
    checked_kind(t, source, "right", 0),
    "the body binds every name of a grouped parameter",
  )
}

// Reads the struct one grouped field states, which takes one member for each name it spells.
func named_grouped_fields(t *testing.T) {
  subject, tree := module_of()
  source := "package fill\n\ntype Pair struct {\n\tLeft, Right *int\n}\n"
  file := analyse(t, subject, tree, []fixture_file{{Path: TEST_PATH, Source: source}})[0]
  pair := types.Underlying_Of(
    subject,
    types.Symbol_Type_Of(subject, member_of(t, subject, file, "Pair")),
  )
  left := types.Member_Index(types.Type_Members_Of(subject, pair))
  right := types.Member_Index(types.Member_Next_Of(subject, left))
  for _, one := range []struct {
    Member types.Member_Index
    Name   types.Name
  }{{left, types.Name("Left")}, {right, types.Name("Right")}} {
    testify.Equal(
      t,
      one.Name,
      types.Symbol_Name_Of(subject,
        types.Symbol_Index(types.Member_Symbol_Of(subject, one.Member))),
      "a grouped field takes a member for the name %q",
      one.Name,
    )
    testify.Equal(
      t,
      types.TYPE_POINTER,
      types.Type_Kind_Of(subject,
        types.Type_Index(types.Member_Type_Of(subject, one.Member))),
      "the name %q wears the type its group states",
      one.Name,
    )
  }
  testify.Equal(
    t,
    types.MEMBER_ABSENT,
    types.Member_Index(types.Member_Next_Of(subject, right)),
    "the final name of a group closes the chain",
  )
}

func test_answers(t *testing.T) {
  subject, file := fixture(t)
  symbol := member_of(t, subject, file, "Total")
  testify.Equal(
    t,
    types.SYMBOL_FUNCTION,
    types.Symbol_Kind_Of(subject, symbol),
    "a function reads back as a function",
  )
  signature := types.Symbol_Type_Of(subject, symbol)
  testify.Equal(
    t,
    types.TYPE_FUNCTION,
    types.Type_Kind_Of(subject, signature),
    "a function names a signature",
  )
  parameters := types.Type_Index(types.Type_Key_Of(subject, signature))
  testify.Equal(
    t,
    types.TYPE_TUPLE,
    types.Type_Kind_Of(subject, parameters),
    "a signature holds its parameters in a tuple",
  )
  first := types.Member_Index(types.Type_Members_Of(subject, parameters))
  testify.Equal(
    t,
    types.Name("entries"),
    types.Symbol_Name_Of(subject,
      types.Symbol_Index(types.Member_Symbol_Of(subject, first))),
    "a parameter names itself",
  )
  second := types.Member_Index(types.Member_Next_Of(subject, first))
  testify.Equal(
    t,
    types.TYPE_SLICE,
    types.Type_Kind_Of(subject,
      types.Type_Index(types.Member_Type_Of(subject, second))),
    "a run of arguments reads as a slice",
  )
  results := types.Type_Index(types.Type_Element_Of(subject, signature))
  sum := types.Member_Index(types.Type_Members_Of(subject, results))
  testify.Equal(
    t,
    types.Name("sum"),
    types.Symbol_Name_Of(subject,
      types.Symbol_Index(types.Member_Symbol_Of(subject, sum))),
    "a result names itself",
  )
  constraint := types.Underlying_Of(
    subject,
    types.Symbol_Type_Of(subject, member_of(t, subject, file, "Numeric")),
  )
  testify.Equal(
    t,
    types.TYPE_CONSTRAINT,
    types.Type_Kind_Of(subject, constraint),
    "a constraint names a constraint",
  )
  testify.Not_Equal(
    t,
    types.MEMBER_ABSENT,
    types.Member_Index(types.Type_Members_Of(subject, constraint)),
    "a constraint states the terms it admits",
  )
  answers_imports(t, subject, file)
  answers_links(t, subject, file)
  answers_heads(t)
  answers_sweep(t, subject)
  answers_messages(t)
}

// Reads the links one type holds to another, which is how a caller walks a type it was handed.
func answers_links(t *testing.T, subject *types.Module, file types.File_Index) {
  t.Helper()
  for _, one := range []struct {
    Name string
    Kind types.Type_Kind
  }{
    {"flag", types.TYPE_BOOLEAN},
    {"text", types.TYPE_STRING},
  } {
    index := types.Symbol_Type_Of(subject, member_of(t, subject, file, one.Name))
    testify.Equal(
      t,
      one.Kind,
      types.Type_Kind_Of(subject,
        types.Type_Index(types.Type_Element_Of(subject, index))),
      "the pointer %q names the type it stands over",
      one.Name,
    )
  }
  for _, one := range []struct {
    Name string
    Kind types.Type_Kind
  }{
    {"truth", types.TYPE_BOOLEAN},
    {"words", types.TYPE_STRING},
  } {
    index := types.Symbol_Type_Of(subject, member_of(t, subject, file, one.Name))
    testify.Equal(
      t,
      one.Kind,
      types.Type_Kind_Of(subject,
        types.Type_Index(types.Type_Key_Of(subject, index))),
      "the map %q names the type of its key",
      one.Name,
    )
  }
  solo := member_of(t, subject, file, "Solo")
  testify.Equal(
    t,
    types.File_Index(2),
    types.Symbol_File_Of(subject, solo),
    "a symbol names the file that states it",
  )
  for _, name := range []string{"A", "AB"} {
    symbol := member_of(t, subject, file, name)
    testify.Equal(
      t,
      types.Name(name),
      types.Symbol_Name_Of(subject, symbol),
      "the name %q reads back whole",
      name,
    )
  }
}

// Reads the first member of a struct in two modules, because one module states one first slot
// and a head that stands behind it needs a module of its own.
func answers_heads(t *testing.T) {
  t.Helper()
  for _, one := range []struct {
    Source string
    Head   types.Member_Index
  }{
    {"package one\n\ntype Pair struct {\n\tFlag bool\n\tText string\n}\n", 1},
    {
      "package one\n\ntype One struct {\n\tFlag bool\n}\n\n" +
        "type Two struct {\n\tText string\n}\n",
      2,
    },
  } {
    subject, tree := module_of()
    file := analyse(t, subject, tree,
      []fixture_file{{Path: TEST_PATH, Source: one.Source}})[0]
    name := "Pair"
    if one.Head != 1 {
      name = "Two"
    }
    index := types.Underlying_Of(
      subject,
      types.Symbol_Type_Of(subject, member_of(t, subject, file, name)),
    )
    testify.Equal(
      t,
      one.Head,
      types.Member_Index(types.Type_Members_Of(subject, index)),
      "a struct names the first member of its chain",
    )
    if one.Head == 1 {
      testify.Equal(
        t,
        types.Member_Index(2),
        types.Member_Index(types.Member_Next_Of(subject, one.Head)),
        "a member names the member the arena took after it",
      )
    }
    testify.Equal(
      t,
      types.TYPE_BOOLEAN,
      types.Type_Kind_Of(subject, types.Type_Index(
        types.Member_Type_Of(subject,
          1),
      )),
      "the first member wears the type it states",
    )
    testify.Equal(
      t,
      types.TYPE_STRING,
      types.Type_Kind_Of(subject, types.Type_Index(
        types.Member_Type_Of(subject, 2),
      )),
      "the second member wears its own type",
    )
  }
}

func answers_imports(t *testing.T, subject *types.Module, file types.File_Index) {
  t.Helper()
  binding := types.Symbol_Index(0)
  for index := range types.Symbol_Index(count_symbols(subject)) {
    if types.Symbol_Kind_Of(subject, index) != types.SYMBOL_IMPORT {
      continue
    }
    binding = index
  }
  testify.Not_Equal(t, types.SYMBOL_ABSENT, binding, "the file states one import")
  testify.Equal(
    t,
    types.Name("two"),
    types.Symbol_Name_Of(subject, binding),
    "an import wears the name the file states",
  )
  testify.Equal(
    t,
    types.Path(TEST_OTHER_PATH),
    types.Path_Of(subject, types.Symbol_Target_Of(subject, binding)),
    "an import names the package its path states",
  )
  testify.Not_Equal(
    t,
    types.Package_Of(subject, file),
    types.Symbol_Target_Of(subject, binding),
    "an import names another package",
  )
}

// Reads every answer at the first slots and the final slot of each arena, because a caller may
// hold any slot and an answer that refuses one is an answer a caller cannot use.
func answers_sweep(t *testing.T, subject *types.Module) {
  t.Helper()
  for _, index := range []types.Type_Index{0, 1, 2, types.TYPE_INDEX_MAXIMUM} {
    testify.True(
      t,
      !(types.Type_Index(types.Type_Element_Of(subject, index)) >
        types.TYPE_INDEX_MAXIMUM),
      "a type slot names an element inside the arena",
    )
    testify.True(
      t,
      !(types.Type_Index(types.Type_Key_Of(subject, index)) >
        types.TYPE_INDEX_MAXIMUM),
      "a type slot names a key inside the arena",
    )
    symbol := types.Symbol_Index(types.Type_Symbol_Of(subject, index))
    testify.True(
      t,
      !(symbol > types.SYMBOL_INDEX_MAXIMUM),
      "a type slot names a symbol inside the arena",
    )
    member := types.Member_Index(types.Type_Members_Of(subject, index))
    testify.True(
      t,
      !(member > types.MEMBER_INDEX_MAXIMUM),
      "a type slot names a member inside the arena",
    )
    testify.True(
      t,
      !(types.Underlying_Of(subject, index) > types.TYPE_INDEX_MAXIMUM),
      "a type slot walks to a type inside the arena",
    )
    testify.True(
      t,
      !(types.Type_Element_Count_Of(subject, index) < types.ELEMENT_COUNT_UNKNOWN),
      "a type slot names a element_count it holds",
    )
    testify.True(
      t,
      !(uint8(types.Type_Kind_Of(subject, index)) > uint8(types.TYPE_TUPLE)),
      "a type slot names a kind the module states",
    )
  }
  for _, index := range []types.Symbol_Index{0, 1, 2, types.SYMBOL_INDEX_MAXIMUM} {
    testify.True(
      t,
      !(types.Symbol_Type_Of(subject, index) > types.TYPE_INDEX_MAXIMUM),
      "a symbol slot names a type inside the arena",
    )
    testify.True(
      t,
      !(types.Symbol_File_Of(subject, index) > types.FILE_INDEX_MAXIMUM),
      "a symbol slot names a file inside the arena",
    )
    testify.True(
      t,
      !(types.Symbol_Target_Of(subject, index) > types.PACKAGE_INDEX_MAXIMUM),
      "a symbol slot names a package inside the arena",
    )
    testify.True(
      t,
      !(len(types.Symbol_Name_Of(subject, index)) > types.NAME_SIZE_MAXIMUM),
      "a symbol slot wears a name inside the width",
    )
    testify.True(
      t,
      !(uint8(types.Symbol_Kind_Of(subject, index)) >
        uint8(types.SYMBOL_TYPE_PARAMETER)),
      "a symbol slot names a kind",
    )
  }
  for _, index := range []types.Member_Index{0, 1, 2, types.MEMBER_INDEX_MAXIMUM} {
    testify.True(
      t,
      !(types.Type_Index(types.Member_Type_Of(subject, index)) >
        types.TYPE_INDEX_MAXIMUM),
      "a member slot names a type inside the arena",
    )
    symbol := types.Symbol_Index(types.Member_Symbol_Of(subject, index))
    testify.True(
      t,
      !(symbol > types.SYMBOL_INDEX_MAXIMUM),
      "a member slot names a symbol inside the arena",
    )
    next := types.Member_Index(types.Member_Next_Of(subject, index))
    testify.True(
      t,
      !(next > types.MEMBER_INDEX_MAXIMUM),
      "a member slot names a member inside the arena",
    )
  }
  for _, index := range []types.File_Index{0, 1, 2, types.FILE_INDEX_MAXIMUM} {
    testify.True(
      t,
      !(types.Package_Of(subject, index) > types.PACKAGE_INDEX_MAXIMUM),
      "a file slot names a package inside the arena",
    )
    testify.True(
      t,
      !(len(types.Source_Of(subject, index)) > token.SOURCE_SIZE_MAXIMUM),
      "a file slot names a source inside the width",
    )
  }
  for _, index := range []types.Package_Index{0, 1, 2, types.PACKAGE_INDEX_MAXIMUM} {
    testify.True(
      t,
      !(len(types.Path_Of(subject, index)) > types.PATH_SIZE_MAXIMUM),
      "a package slot answers to a path inside the width",
    )
    result := types.Lookup(subject, index, types.Name("Count"))
    testify.True(t, bool(result.Found) != !bool(result.Found),
      "a package slot answers a search")
    report := types.Unpaired(subject, index)
    testify.True(
      t,
      !(report.Symbol < types.PACKAGE_SYMBOL_FIRST),
      "a package slot answers for the functions it pairs",
    )
  }
}

func answers_messages(t *testing.T) {
  t.Helper()
  for _, code := range []types.Failure_Code{
    types.FAILURE_NONE,
    types.FAILURE_FILE_COUNT,
    types.FAILURE_PACKAGE_COUNT,
    types.FAILURE_SYMBOL_COUNT,
    types.FAILURE_TYPE_COUNT,
    types.FAILURE_MEMBER_COUNT,
    types.FAILURE_NAME_SIZE,
    types.FAILURE_PATH_SIZE,
    types.FAILURE_NESTING_DEPTH,
    types.FAILURE_UNKNOWN_NAME,
    types.FAILURE_DUPLICATE_NAME,
    types.FAILURE_UNKNOWN_IMPORT,
    types.FAILURE_TYPE_CYCLE,
    types.FAILURE_SHADOWED_NAME,
    types.FAILURE_UNEXPORTED_TYPE,
    types.FAILURE_NOT_TYPE,
  } {
    text := types.Failure_Message(code)
    testify.True(
      t,
      !(len(text) > types.MESSAGE_SIZE_MAXIMUM),
      "the code %d reads as a sentence inside the width",
      code,
    )
    if code == types.FAILURE_NONE {
      testify.Equal(t, types.Message(""), text,
        "a clean module reads no sentence")
      continue
    }
    testify.Not_Equal(t, types.Message(""), text, "a refusal reads a sentence")
  }
}

func test_refusals(t *testing.T) {
  refused_module_passes(t)
  for _, one := range []struct {
    Source string
    Code   types.Failure_Code
  }{
    {"package one\n\nvar value Missing\n", types.FAILURE_UNKNOWN_NAME},
    {"package one\n\ntype Count int\n\ntype Count int\n", types.FAILURE_DUPLICATE_NAME},
    {"package one\n\ntype Loop Loop\n", types.FAILURE_TYPE_CYCLE},
    {
      "package one\n\nimport far \"local/example/far\"\n\nvar value far.Marker\n",
      types.FAILURE_UNKNOWN_IMPORT,
    },
    {"package one\n\nvar value other.Marker\n", types.FAILURE_UNKNOWN_NAME},
  } {
    subject, tree := module_of()
    file := types.Add_File(subject, TEST_PATH, token.Source(one.Source)).File
    ast.Parse(tree, token.Source(one.Source))
    declared := types.Declare(subject, file, tree)
    ast.Parse(tree, token.Source(one.Source))
    resolved := types.Resolve(subject, file, tree)
    if declared {
      testify.False(t, bool(resolved), "a refused source fails one pass")
    }
    testify.Equal(t, one.Code, types.Failure(subject), "the refusal names its code")
    testify.Not_Equal(
      t,
      types.Message(""),
      types.Failure_Message(one.Code),
      "a code reads as a sentence",
    )
  }
  subject, tree := module_of()
  broken := token.Source("package one\n\nvar value int\n\nfunc (\n")
  file := types.Add_File(subject, TEST_PATH, broken).File
  ast.Parse(tree, broken)
  testify.True(
    t,
    bool(types.Declare(subject, file, tree)),
    "a refused parse still declares the names its tree holds",
  )
  testify.Equal(
    t,
    types.FAILURE_NONE,
    types.Failure(subject),
    "a refused parse is the business of the parser and never of the module",
  )
  for _, text := range refused_sources() {
    held, parsed := module_of()
    source := token.Source(text)
    index := types.Add_File(held, TEST_PATH, source).File
    ast.Parse(parsed, source)
    types.Declare(held, index, parsed)
    types.Resolve(held, index, parsed)
    types.Check(held, body_state(), index, parsed)
    testify.Equal(
      t,
      types.FAILURE_NONE,
      types.Failure(held),
      "every pass reads a refused tree and leaves the refusal to the parser",
    )
  }
}

func test_bounds(t *testing.T) {
  bounds_files(t)
  bounds_packages(t)
  bounds_widths(t)
  bounds_depth(t)
  bounds_symbols(t)
  bounds_types(t)
  bounds_members(t)
  bounds_locals(t)
  bounds_staged(t)
}

// Checks one body source and reads the kind the identifier the text names folds to at a place.
func checked_kind(t *testing.T, source string, text string, place int) (kind types.Type_Kind) {
  t.Helper()
  subject, tree := module_of()
  file := analyse(t, subject, tree, []fixture_file{{Path: TEST_PATH, Source: source}})[0]
  body := body_state()
  ast.Parse(tree, token.Source(source))
  testify.True(t, bool(types.Check(subject, body, file, tree)), "the body checks")
  found := ast.Index(0)
  seen := 0
  walk_nodes(tree, 1, func(index ast.Index) {
    node_match(tree, index, ast.NODE_IDENTIFIER, source, text, place, &seen, &found)
  })
  testify.Not_Equal(t, ast.Index(0), found, "the source states %q", text)
  return types.Type_Kind_Of(subject, types.Type_At(body, found))
}

// Fills the local storage of one body and opens a block past it. The names bound stand, and a
// name past the bound binds nothing.
func bounds_locals(t *testing.T) {
  buffer := []byte("package one\n\nfunc Wide(")
  for index := range types.LOCAL_COUNT_MAXIMUM {
    if index > 0 {
      buffer = append(buffer, ", "...)
    }
    buffer = append(buffer, fmt.Sprintf("l%d", index)...)
  }
  buffer = append(buffer, " int) {\n\tvar past int\n\t{\n\t\tpast = l0\n\t}\n}\n"...)
  source := string(buffer)
  testify.Equal(
    t,
    types.TYPE_INT,
    checked_kind(t, source, "l0", 0),
    "a name inside the local bound stands in a block past it",
  )
  testify.Equal(
    t,
    types.TYPE_INVALID,
    checked_kind(t, source, "past", 1),
    "a name past the local bound binds nothing",
  )
}

// Binds the most names one short declaration stages, each to the value at its own place.
func bounds_staged(t *testing.T) {
  names := []byte{}
  values := []byte{}
  for index := range types.STAGED_NAME_MAXIMUM {
    if index > 0 {
      names = append(names, ", "...)
      values = append(values, ", "...)
    }
    names = append(names, fmt.Sprintf("s%d", index)...)
    values = append(values, fmt.Sprintf("\"%d\"", index)...)
  }
  last := fmt.Sprintf("s%d", types.STAGED_NAME_MAXIMUM-1)
  source := "package one\n\nfunc Many() {\n\t" +
    string(names) +
    " := " +
    string(values) +
    "\n\tlast := " +
    last +
    "\n}\n"
  testify.Equal(
    t,
    types.TYPE_UNTYPED_STRING,
    checked_kind(t, source, last, 1),
    "the final staged name wears the value at the final place",
  )
}

// Reads how many type slots one module spent, which is the first slot no fold wrote.
func count_types(subject *types.Module) (count int) {
  for index := 1; index < types.TYPE_COUNT_MAXIMUM; index++ {
    if types.Type_Kind_Of(subject, types.Type_Index(index)) == types.TYPE_INVALID {
      return index
    }
  }
  return types.TYPE_COUNT_MAXIMUM
}

// Reads how many symbol slots one module spent.
func count_symbols(subject *types.Module) (count int) {
  for index := 1; index < types.SYMBOL_COUNT_MAXIMUM; index++ {
    if types.Symbol_Kind_Of(subject,
      types.Symbol_Index(index)) == types.SYMBOL_UNKNOWN {
      return index
    }
  }
  return types.SYMBOL_COUNT_MAXIMUM
}

// Builds a source that states one variable for each slot a fill needs.
func filled_source(prefix string, one_type string, count int) (source string) {
  buffer := make([]byte, 0, count*20+16)
  buffer = append(buffer, "package fill\n"...)
  for index := range count {
    buffer = append(buffer, fmt.Sprintf("var %s%d %s\n", prefix, index, one_type)...)
  }
  return string(buffer)
}

// Builds a struct that states one field for each member slot a fill needs. One line names every
// field, thus the parser takes the names as one run.
func filled_structure(count int) (source string) {
  buffer := make([]byte, 0, count*8+64)
  buffer = append(buffer, "package fill\n\ntype Wide struct {\n\t"...)
  for index := range count {
    if index > 0 {
      buffer = append(buffer, ", "...)
    }
    buffer = append(buffer, fmt.Sprintf("F%d", index)...)
  }
  return string(append(buffer, " int\n}\n"...))
}

func bounds_files(t *testing.T) {
  subject, tree := module_of()
  filled := 0
  for filled < types.FILE_COUNT_MAXIMUM-1 {
    result := types.Add_File(subject, TEST_PATH, token.Source("package one\n"))
    testify.True(t, bool(result.OK), "a file inside the bound binds")
    filled++
  }
  last_result := types.Add_File(subject, TEST_PATH, token.Source(TEST_OTHER_SOURCE))
  last := last_result.File
  testify.True(t, bool(last_result.OK), "the final file binds")
  testify.Equal(
    t,
    types.File_Index(types.FILE_COUNT_MAXIMUM-1),
    last,
    "the final file takes the final slot",
  )
  ast.Parse(tree, token.Source(TEST_OTHER_SOURCE))
  testify.True(t, bool(types.Declare(subject, last, tree)), "the final file declares")
  ast.Parse(tree, token.Source(TEST_OTHER_SOURCE))
  testify.True(t, bool(types.Resolve(subject, last, tree)), "the final file resolves")
  marker_result := types.Lookup(
    subject, types.Package_Of(subject, last), types.Name("Marker"),
  )
  testify.True(t, bool(marker_result.Found), "the final file states its names")
  testify.Equal(
    t,
    last,
    types.Symbol_File_Of(subject, marker_result.Symbol),
    "a symbol names the file that states it",
  )
  over := types.Add_File(subject, TEST_PATH, token.Source("package one\n"))
  testify.False(t, bool(over.OK), "a file past the bound is refused")
  testify.Equal(
    t,
    types.FAILURE_FILE_COUNT,
    types.Failure(subject),
    "a file past the bound names its code",
  )
}

func bounds_packages(t *testing.T) {
  subject, _ := module_of()
  for index := range types.PACKAGE_COUNT_MAXIMUM - 1 {
    path := types.Path(fmt.Sprintf("local/example/%d", index))
    result := types.Add_File(subject, path, token.Source("package one\n"))
    testify.True(t, bool(result.OK), "a package inside the bound binds")
    testify.Equal(
      t,
      types.Package_Index(index+1),
      types.Package_Of(subject, result.File),
      "a path takes the package slot after the last one",
    )
  }
  last := types.Package_Index(types.PACKAGE_COUNT_MAXIMUM - 1)
  testify.Equal(
    t,
    types.Path("local/example/254"),
    types.Path_Of(subject, last),
    "a package reads back the path it answers to",
  )
  absent := types.Lookup(subject, last, types.Name("Marker"))
  testify.False(t, bool(absent.Found), "a package with no file states no name")
  bounds_final_package(t, subject)
  over := types.Add_File(subject, "local/example/over", token.Source("package one\n"))
  testify.False(t, bool(over.OK), "a package past the bound is refused")
  testify.Equal(
    t,
    types.FAILURE_PACKAGE_COUNT,
    types.Failure(subject),
    "a package past the bound names its code",
  )
}

// Declares one file of the final package, which is where a caller reads the widest package slot
// a module holds.
func bounds_final_package(t *testing.T, subject *types.Module) {
  t.Helper()
  tree := parse_state()
  source := "package last\n\nimport near \"local/example/1\"\n\n" +
    "import far \"local/example/254\"\n\ntype Held int\n"
  file_result := types.Add_File(subject, "local/example/254", token.Source(source))
  file := file_result.File
  testify.True(t, bool(file_result.OK), "the file of the final package binds")
  testify.Equal(
    t,
    types.Package_Index(types.PACKAGE_INDEX_MAXIMUM),
    types.Package_Of(subject, file),
    "the final path names the final package",
  )
  ast.Parse(tree, token.Source(source))
  testify.True(t, bool(types.Declare(subject, file, tree)), "the final package declares")
  ast.Parse(tree, token.Source(source))
  testify.True(t, bool(types.Resolve(subject, file, tree)), "the final package resolves")
  near_result := types.Lookup(subject, types.Package_Of(subject, file), types.Name("Held"))
  testify.True(t, bool(near_result.Found), "the final package states its names")
  testify.Equal(
    t,
    types.File_Index(types.PACKAGE_COUNT_MAXIMUM-1),
    types.Symbol_File_Of(subject, near_result.Symbol),
    "a symbol names the file that states it",
  )
  for _, one := range []struct {
    Name   string
    Target types.Package_Index
  }{
    {"near", 2},
    {"far", types.PACKAGE_INDEX_MAXIMUM},
  } {
    binding := import_of(t, subject, file, one.Name)
    testify.Equal(
      t,
      one.Target,
      types.Symbol_Target_Of(subject, binding),
      "the import %q names the package its path states",
      one.Name,
    )
  }
}

// Names one import binding of a file, which no package table holds because an import stands in
// one file alone.
func import_of(
  t *testing.T,
  subject *types.Module,
  file types.File_Index,
  name string,
) (symbol types.Symbol_Index) {
  t.Helper()
  for index := range types.Symbol_Index(count_symbols(subject)) {
    if types.Symbol_Kind_Of(subject, index) != types.SYMBOL_IMPORT {
      continue
    }
    if types.Symbol_File_Of(subject, index) != file {
      continue
    }
    if string(types.Symbol_Name_Of(subject, index)) != name {
      continue
    }
    return index
  }
  testify.Fail(t, "the file states no import named "+name)
  return types.SYMBOL_ABSENT
}

func bounds_widths(t *testing.T) {
  subject, tree := module_of()
  wide := make([]byte, types.PATH_SIZE_MAXIMUM)
  for index := range wide {
    wide[index] = 'p'
  }
  source := make([]byte, token.SOURCE_SIZE_MAXIMUM)
  for index := range source {
    source[index] = '\n'
  }
  for _, path := range []types.Path{"", "a", "ab", types.Path(wide)} {
    result := types.Add_File(subject, path, token.Source(source))
    testify.True(t, bool(result.OK), "a path of %d bytes binds", len(path))
    testify.Equal(
      t,
      path,
      types.Path_Of(subject, types.Package_Of(subject, result.File)),
      "a path of %d bytes reads back",
      len(path),
    )
  }
  for _, one := range []string{"", "a", "ab"} {
    result := types.Add_File(subject, "local/example/short", token.Source(one))
    testify.True(t, bool(result.OK), "a source of %d bytes binds", len(one))
    testify.Equal(
      t,
      token.Source(one),
      types.Source_Of(subject, result.File),
      "a source of %d bytes reads back",
      len(one),
    )
  }
  testify.Equal(
    t,
    token.Source(source),
    types.Source_Of(subject, 0),
    "the widest source reads back",
  )
  for _, name := range []types.Name{
    types.Name(""),
    types.Name("a"),
    types.Name("ab"),
    types.Name(wide[:types.NAME_SIZE_MAXIMUM]),
  } {
    result := types.Lookup(subject, types.PACKAGE_UNIVERSE, name)
    testify.False(t, bool(result.Found),
      "the universe states no name of %d bytes", len(name))
  }
  long := fmt.Sprintf("package fill\n\nvar %s int\n", string(wide[:types.NAME_SIZE_MAXIMUM]))
  file := types.Add_File(subject, "local/example/long", token.Source(long)).File
  ast.Parse(tree, token.Source(long))
  testify.True(t, bool(types.Declare(subject, file, tree)), "a name at the bound declares")
  result := types.Lookup(
    subject,
    types.Package_Of(subject, file),
    types.Name(wide[:types.NAME_SIZE_MAXIMUM]),
  )
  testify.True(t, bool(result.Found), "a name at the bound binds")
  testify.Equal(
    t,
    types.NAME_SIZE_MAXIMUM,
    len(types.Symbol_Name_Of(subject, result.Symbol)),
    "a name at the bound reads back whole",
  )
  bounds_name_size(t)
}

func bounds_name_size(t *testing.T) {
  subject, tree := module_of()
  wide := make([]byte, types.NAME_SIZE_MAXIMUM+1)
  for index := range wide {
    wide[index] = 'n'
  }
  source := fmt.Sprintf("package fill\n\nvar %s int\n", string(wide))
  file := types.Add_File(subject, TEST_PATH, token.Source(source)).File
  ast.Parse(tree, token.Source(source))
  testify.False(t, bool(types.Declare(subject, file, tree)),
    "a name past the bound is refused")
  testify.Equal(
    t,
    types.FAILURE_NAME_SIZE,
    types.Failure(subject),
    "a name past the bound names its code",
  )
}

func bounds_depth(t *testing.T) {
  subject, tree := module_of()
  stars := make([]byte, types.DEPTH_MAXIMUM+2)
  for index := range stars {
    stars[index] = '*'
  }
  source := fmt.Sprintf("package fill\n\nvar deep %sint\n", string(stars))
  file := types.Add_File(subject, TEST_PATH, token.Source(source)).File
  ast.Parse(tree, token.Source(source))
  types.Declare(subject, file, tree)
  ast.Parse(tree, token.Source(source))
  testify.False(
    t,
    bool(types.Resolve(subject, file, tree)),
    "a type past the nesting bound is refused",
  )
  testify.Equal(
    t,
    types.FAILURE_NESTING_DEPTH,
    types.Failure(subject),
    "a type past the nesting bound names its code",
  )
}

// Copies every storage of one module into storage of its own, thus many cases read one head.
func clone_module(from *types.Module) (subject *types.Module) {
  subject = module_state()
  copy(subject.Files, from.Files)
  copy(subject.Packages, from.Packages)
  copy(subject.Symbols, from.Symbols)
  copy(subject.Types, from.Types)
  copy(subject.Members, from.Members)
  copy(subject.Buckets, from.Buckets)
  copy(subject.Nodes, from.Nodes)
  copy(subject.Parameters, from.Parameters)
  copy(subject.Owners, from.Owners)
  copy(subject.Words, from.Words)
  copy(subject.Counts, from.Counts)
  return subject
}

// Feeds tail files one after another into a copy of a module whose head already stands, which is
// how a case reads the final slot of an arena. The file is the last tail.
func fill_arena(
  head *types.Module,
  tails ...string,
) (subject *types.Module, file types.File_Index) {
  subject = clone_module(head)
  tree := parse_state()
  for _, tail := range tails {
    file = types.Add_File(subject, TEST_PATH, token.Source(tail)).File
    ast.Parse(tree, token.Source(tail))
    types.Declare(subject, file, tree)
    types.Resolve(subject, file, tree)
  }
  return subject, file
}

// Builds the module that leaves the symbol arena with two free slots. One declaration states many
// names, thus the head spends two tokens for each symbol it takes.
func symbol_head(t *testing.T) (head *types.Module) {
  t.Helper()
  head, tree := module_of()
  need := types.SYMBOL_COUNT_MAXIMUM - count_symbols(head) - 2
  files := []fixture_file{}
  spent := 0
  for spent < need {
    count := min(need-spent, TEST_NAME_RUN_MAXIMUM)
    one := fixture_file{Path: TEST_PATH, Source: named_run(spent, count)}
    files = append(files, one)
    spent = spent + count
  }
  analyse(t, head, tree, files)
  return head
}

// Builds one file that states one variable declaration of many names.
func named_run(first int, count int) (source string) {
  buffer := make([]byte, 0, count*12+32)
  buffer = append(buffer, "package fill\n\nvar "...)
  for index := range count {
    if index > 0 {
      buffer = append(buffer, ", "...)
    }
    buffer = append(buffer, fmt.Sprintf("value%d", first+index)...)
  }
  return string(append(buffer, " int\n"...))
}

// Builds the module that leaves the type arena with TEST_TYPE_FREE free slots. One declaration
// states a run of pointers, thus the head spends few symbol slots for the types it takes.
func type_head(t *testing.T) (head *types.Module) {
  t.Helper()
  head, tree := module_of()
  need := types.TYPE_COUNT_MAXIMUM - count_types(head) - TEST_TYPE_FREE
  buffer := make([]byte, 0, need*4+64)
  buffer = append(buffer, "package fill\n"...)
  stars := make([]byte, TEST_POINTER_RUN)
  for index := range stars {
    stars[index] = '*'
  }
  for index := range need / TEST_POINTER_RUN {
    buffer = append(buffer, fmt.Sprintf("var run%d %sint\n", index, string(stars))...)
  }
  for index := range need % TEST_POINTER_RUN {
    buffer = append(buffer, fmt.Sprintf("var solitary%d *int\n", index)...)
  }
  analyse(t, head, tree, []fixture_file{{Path: TEST_PATH, Source: string(buffer)}})
  return head
}

// Builds the file that leaves the type arena of a head with the free slot count it names. Each
// pointer takes one slot.
func type_pad(free int) (source string) {
  buffer := []byte("package fill\n")
  for index := range TEST_TYPE_FREE - free {
    buffer = append(buffer, fmt.Sprintf("var pad%d *int\n", index)...)
  }
  return string(buffer)
}

// Builds the module that leaves the member arena with two free slots. One grouped field states a
// name for each member it takes.
func member_head(t *testing.T) (head *types.Module) {
  t.Helper()
  head, tree := module_of()
  analyse(
    t,
    head,
    tree,
    []fixture_file{
      {Path: TEST_PATH, Source: filled_structure(types.MEMBER_COUNT_MAXIMUM - 3)},
    },
  )
  return head
}

func bounds_symbols(t *testing.T) {
  head := symbol_head(t)
  last := types.Symbol_Index(types.SYMBOL_INDEX_MAXIMUM)
  subject, file := fill_arena(head, "package fill\n\ntype Solo struct {\n\tOnly int\n}\n")
  testify.Equal(
    t,
    types.SYMBOL_FIELD,
    types.Symbol_Kind_Of(subject, last),
    "the final symbol slot holds the last name a file stated",
  )
  solo := types.Underlying_Of(
    subject,
    types.Symbol_Type_Of(subject, member_of(t, subject, file, "Solo")),
  )
  testify.Equal(
    t,
    last,
    types.Symbol_Index(
      types.Member_Symbol_Of(subject,
        types.Member_Index(types.Type_Members_Of(subject, solo))),
    ),

    "the final field wears the final symbol slot",
  )
  bounds_past_symbol(t, subject)
  bounds_final_symbol(t, head)
}

func bounds_past_symbol(t *testing.T, full *types.Module) {
  subject, file := fill_arena(full, "package fill\n\nvar past int\n")
  testify.Equal(
    t,
    types.FAILURE_SYMBOL_COUNT,
    types.Failure(subject),
    "a name past the bound names its code",
  )
  absent := types.Lookup(subject, types.Package_Of(subject, file), types.Name("past"))
  testify.False(t, bool(absent.Found), "a name past the bound binds nothing")
}

func bounds_final_symbol(t *testing.T, head *types.Module) {
  subject, file := fill_arena(
    head,
    "package fill\n\nvar pad int\n",
    "package fill\n\ntype Last int\n",
  )
  result := types.Lookup(subject, types.Package_Of(subject, file), types.Name("Last"))
  testify.True(t, bool(result.Found), "the final name binds")
  testify.Equal(
    t,
    types.Symbol_Index(types.SYMBOL_INDEX_MAXIMUM),
    result.Symbol,
    "the final name takes the final symbol slot",
  )
  testify.Equal(
    t,
    result.Symbol,
    types.Symbol_Index(types.Type_Symbol_Of(subject,
      types.Symbol_Type_Of(subject, result.Symbol))),

    "the final named type carries the final symbol slot",
  )
  testify.Equal(
    t,
    types.FAILURE_NONE,
    types.Failure(subject),
    "a module that fills its symbols exactly refuses nothing",
  )
}

func bounds_types(t *testing.T) {
  head := type_head(t)
  last := types.Type_Index(types.TYPE_INDEX_MAXIMUM)
  subject, file := fill_arena(
    head,
    type_pad(2),
    "package fill\n\nvar pair **int\n\nvar over *int\n",
  )
  testify.Equal(
    t,
    types.TYPE_POINTER,
    types.Type_Kind_Of(subject, last),
    "the final type slot holds the last type a file stated",
  )
  testify.Equal(
    t,
    last,
    types.Type_Index(types.Type_Element_Of(subject, last-1)),
    "a pointer over a pointer names the slot behind it",
  )
  testify.Equal(
    t,
    types.FAILURE_TYPE_COUNT,
    types.Failure(subject),
    "a type past the bound names its code",
  )
  pair := types.Lookup(subject, types.Package_Of(subject, file), types.Name("pair"))
  testify.True(t, bool(pair.Found), "the final declaration binds its name")
  testify.Equal(
    t,
    last-1,
    types.Symbol_Type_Of(subject, pair.Symbol),
    "the final declaration wears the type it stated",
  )
  bounds_final_type(t, head)
  bounds_final_field(t, head)
  bounds_final_tuple(t, head)
}

func bounds_final_type(t *testing.T, head *types.Module) {
  subject, file := fill_arena(head, type_pad(1), "package fill\n\nvar solo *int\n")
  testify.Equal(
    t,
    types.Type_Index(types.TYPE_INDEX_MAXIMUM),
    types.Symbol_Type_Of(subject, member_of(t, subject, file, "solo")),
    "the final declaration wears the final type slot",
  )
}

func bounds_final_field(t *testing.T, head *types.Module) {
  subject, file := fill_arena(
    head,
    type_pad(3),
    "package fill\n\ntype Holder struct {\n\tValue *int\n}\n",
  )
  holder := types.Underlying_Of(
    subject,
    types.Symbol_Type_Of(subject, member_of(t, subject, file, "Holder")),
  )
  testify.Equal(
    t,
    types.Type_Index(types.TYPE_INDEX_MAXIMUM),
    types.Type_Index(
      types.Member_Type_Of(subject,
        types.Member_Index(types.Type_Members_Of(subject, holder))),
    ),

    "the final field wears the final type slot",
  )
}

func bounds_final_tuple(t *testing.T, head *types.Module) {
  subject, file := fill_arena(head, type_pad(2), "package fill\n\nfunc Only(one int) {\n}\n")
  signature := types.Symbol_Type_Of(subject, member_of(t, subject, file, "Only"))
  testify.Equal(
    t,
    types.Type_Index(types.TYPE_INDEX_MAXIMUM),
    types.Type_Index(types.Type_Key_Of(subject, signature)),
    "the final signature holds its parameters in the final type slot",
  )
}

func bounds_members(t *testing.T) {
  head := member_head(t)
  subject, file := fill_arena(
    head,
    "package fill\n\ntype Tail struct {\n\tOne int\n\tTwo int\n\tThree int\n}\n",
  )
  last := types.Member_Index(types.MEMBER_INDEX_MAXIMUM)
  testify.Equal(
    t,
    types.FAILURE_MEMBER_COUNT,
    types.Failure(subject),
    "a member past the bound names its code",
  )
  testify.Equal(
    t,
    last,
    types.Member_Index(types.Member_Next_Of(subject, last-1)),
    "a member chain reaches the final slot",
  )
  testify.Equal(
    t,
    types.MEMBER_ABSENT,
    types.Member_Index(types.Member_Next_Of(subject, last)),
    "the final member closes its chain",
  )
  testify.Not_Equal(
    t,
    types.SYMBOL_ABSENT,
    types.Symbol_Index(types.Member_Symbol_Of(subject, last)),
    "the final member wears the name its field states",
  )
  tail_type := types.Underlying_Of(
    subject,
    types.Symbol_Type_Of(subject,
      member_of(t, subject, file, "Tail")),
  )
  testify.Equal(
    t,
    last-1,
    types.Member_Index(types.Type_Members_Of(subject, tail_type)),
    "a struct names the first member of its chain",
  )
  bounds_final_head(t, head)
  testify.Equal(
    t,
    types.TYPE_INT,
    types.Type_Kind_Of(subject,
      types.Type_Index(types.Member_Type_Of(subject, last))),
    "the final member wears the type it states",
  )
  bounds_lengths(t)
}

func bounds_final_head(t *testing.T, head *types.Module) {
  subject, file := fill_arena(
    head,
    "package fill\n\nfunc Pad(int) {\n}\n",
    "package fill\n\ntype Tail struct {\n\tOnly int\n}\n",
  )
  index := types.Underlying_Of(
    subject,
    types.Symbol_Type_Of(subject, member_of(t, subject, file, "Tail")),
  )
  testify.Equal(
    t,
    types.Member_Index(types.MEMBER_INDEX_MAXIMUM),
    types.Member_Index(types.Type_Members_Of(subject, index)),
    "the final struct names the final member slot",
  )
}

func bounds_lengths(t *testing.T) {
  subject, tree := module_of()
  source := "package fill\n\nvar none [0]int\n\nvar one [1]int\n\nvar two [2]int\n\n" +
    "var wide [1099511627776]int\n\nvar past [1099511627777]int\n"
  file := analyse(t, subject, tree, []fixture_file{{Path: TEST_PATH, Source: source}})[0]
  for _, one := range []struct {
    Name          string
    Element_Count types.Element_Count
  }{
    {"none", 0},
    {"one", 1},
    {"two", 2},
    {"wide", types.ELEMENT_COUNT_MAXIMUM},
    {"past", types.ELEMENT_COUNT_UNKNOWN},
  } {
    index := types.Symbol_Type_Of(subject, member_of(t, subject, file, one.Name))
    testify.Equal(
      t,
      one.Element_Count,
      types.Type_Element_Count_Of(subject, index),
      "the array %q names how many elements it holds",
      one.Name,
    )
  }
}

type allocation_check struct {
  Name string
  Call func()
}

type allocation_state struct {
  Held        allocation_fixture
  Subject     *types.Module
  File        types.File_Index
  Symbol      types.Symbol_Index
  Type        types.Type_Index
  Member      types.Member_Index
  Owner       types.Package_Index
  Passes      *types.Module
  Parse_State *ast.Parse_State
  Body        *types.Body
  Fresh       types.File_Index
  Pass_Source token.Source
}

func test_allocation(t *testing.T) {
  state := allocation_state_of(t)
  allocation_module_checks(t, state)
  allocation_symbol_checks(t, state)
  allocation_type_checks(t, state)
  allocation_pass_checks(t, state)
}

// Builds shared allocation fixtures outside every measured callback.
func allocation_state_of(t *testing.T) (state allocation_state) {
  state.Subject, state.File = fixture_module(t)
  state.Symbol = member_of(t, state.Subject, state.File, "Count")
  state.Type = types.Symbol_Type_Of(state.Subject, state.Symbol)
  state.Member = types.Member_Index(types.Type_Members_Of(state.Subject, state.Type))
  state.Owner = types.Package_Of(state.Subject, state.File)
  state.Pass_Source = token.Source(
    "package one\n\ntype Count int\n\nfunc Fold() (sum Count) { return sum }\n",
  )
  state.Passes, state.Parse_State = module_of()
  state.Body = body_state()
  result := types.Add_File(state.Passes, TEST_PATH, state.Pass_Source)
  state.Fresh, state.Held.Ok = result.File, result.OK
  testify.True(t, bool(state.Held.Ok), "the file the passes read binds")
  ast.Parse(state.Parse_State, state.Pass_Source)
  return state
}

// Runs one concrete list so closure setup stays outside measured calls.
func allocation_checks(t *testing.T, checks []allocation_check) {
  for _, check := range checks {
    t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
  }
}

// Measures module lifecycle, file answers, and lookup.
func allocation_module_checks(t *testing.T, state allocation_state) {
  reset_subject := module_state()
  path := types.Path(TEST_OTHER_PATH)
  source := token.Source("package two\n")
  sought := types.Name("Count")
  allocation_checks(
    t,
    []allocation_check{
      {Name: "Reset", Call: func() { types.Reset(reset_subject) }},
      {Name: "Failure", Call: func() {
        state.Held.Failure = types.Failure(state.Subject)
      }},
      {Name: "Failure_Message", Call: func() {
        state.Held.Message = types.Failure_Message(types.FAILURE_UNKNOWN_NAME)
      }},
      {Name: "Add_File", Call: func() {
        allocation_file(&state.Held, types.Add_File(state.Subject, path, source))
      }},
      {Name: "Path_Of", Call: func() {
        state.Held.Path = types.Path_Of(state.Subject, state.Owner)
      }},
      {Name: "Package_Of", Call: func() {
        state.Held.Package = types.Package_Of(state.Subject, state.File)
      }},
      {Name: "Source_Of", Call: func() {
        state.Held.Source = types.Source_Of(state.Subject, state.File)
      }},
      {Name: "Lookup", Call: func() {
        allocation_lookup(&state.Held, types.Lookup(
          state.Subject, state.Owner, sought,
        ))
      }},
    },
  )
}

// Measures every symbol answer.
func allocation_symbol_checks(t *testing.T, state allocation_state) {
  allocation_checks(t, []allocation_check{
    {Name: "Symbol_Kind_Of", Call: func() {
      state.Held.Symbol_Kind = types.Symbol_Kind_Of(state.Subject, state.Symbol)
    }},
    {Name: "Symbol_Name_Of", Call: func() {
      state.Held.Name = types.Symbol_Name_Of(state.Subject, state.Symbol)
    }},
    {Name: "Symbol_File_Of", Call: func() {
      state.Held.File = types.Symbol_File_Of(state.Subject, state.Symbol)
    }},
    {Name: "Symbol_Type_Of", Call: func() {
      state.Held.Type = types.Symbol_Type_Of(state.Subject, state.Symbol)
    }},
    {Name: "Symbol_Target_Of", Call: func() {
      state.Held.Package = types.Symbol_Target_Of(state.Subject, state.Symbol)
    }},
  })
}

// Measures every type and member answer.
func allocation_type_checks(t *testing.T, state allocation_state) {
  allocation_checks(t, []allocation_check{
    {Name: "Type_Kind_Of", Call: func() {
      state.Held.Type_Kind = types.Type_Kind_Of(state.Subject, state.Type)
    }},
    {Name: "Type_Element_Of", Call: func() {
      state.Held.Type_Element = types.Type_Element_Of(state.Subject, state.Type)
    }},
    {Name: "Type_Key_Of", Call: func() {
      state.Held.Type_Key = types.Type_Key_Of(state.Subject, state.Type)
    }},
    {Name: "Type_Element_Count_Of", Call: func() {
      state.Held.Element_Count = types.Type_Element_Count_Of(
        state.Subject, state.Type,
      )
    }},
    {Name: "Type_Symbol_Of", Call: func() {
      state.Held.Type_Symbol = types.Type_Symbol_Of(state.Subject, state.Type)
    }},
    {Name: "Underlying_Of", Call: func() {
      state.Held.Type = types.Underlying_Of(state.Subject, state.Type)
    }},
    {Name: "Type_Members_Of", Call: func() {
      state.Held.Member_Head = types.Type_Members_Of(state.Subject, state.Type)
    }},
    {Name: "Member_Type_Of", Call: func() {
      state.Held.Member_Type = types.Member_Type_Of(state.Subject, state.Member)
    }},
    {Name: "Member_Symbol_Of", Call: func() {
      state.Held.Member_Symbol = types.Member_Symbol_Of(
        state.Subject, state.Member,
      )
    }},
    {Name: "Member_Next_Of", Call: func() {
      state.Held.Member_Next = types.Member_Next_Of(state.Subject, state.Member)
    }},
  })
}

// Measures each pass after rebuilding its required state inside measured call.
func allocation_pass_checks(t *testing.T, state allocation_state) {
  allocation_checks(
    t,
    []allocation_check{
      {Name: "Declare", Call: func() { allocation_declare(&state) }},
      {Name: "Resolve", Call: func() { allocation_resolve(&state) }},
      {Name: "Check", Call: func() { allocation_check_pass(&state) }},
      {Name: "Type_At", Call: func() {
        state.Held.Type = types.Type_At(state.Body, 0)
      }},
    },
  )
  testify.True(t, bool(state.Held.Ok), "the allocation fixture completes each pass")
}

func allocation_declare(state *allocation_state) {
  types.Reset(state.Passes)
  state.Fresh = types.Add_File(state.Passes, TEST_PATH, state.Pass_Source).File
  ast.Parse(state.Parse_State, state.Pass_Source)
  state.Held.Ok = types.Declare(
    state.Passes, state.Fresh, state.Parse_State,
  )
}

func allocation_resolve(state *allocation_state) {
  types.Reset(state.Passes)
  state.Fresh = types.Add_File(state.Passes, TEST_PATH, state.Pass_Source).File
  ast.Parse(state.Parse_State, state.Pass_Source)
  types.Declare(state.Passes, state.Fresh, state.Parse_State)
  ast.Parse(state.Parse_State, state.Pass_Source)
  state.Held.Ok = types.Resolve(
    state.Passes, state.Fresh, state.Parse_State,
  )
}

func allocation_check_pass(state *allocation_state) {
  types.Reset(state.Passes)
  state.Fresh = types.Add_File(state.Passes, TEST_PATH, state.Pass_Source).File
  ast.Parse(state.Parse_State, state.Pass_Source)
  types.Declare(state.Passes, state.Fresh, state.Parse_State)
  ast.Parse(state.Parse_State, state.Pass_Source)
  types.Resolve(state.Passes, state.Fresh, state.Parse_State)
  ast.Parse(state.Parse_State, state.Pass_Source)
  state.Held.Ok = types.Check(
    state.Passes, state.Body, state.Fresh, state.Parse_State,
  )
}

func allocation_file(held *allocation_fixture, result types.File_Result) {
  held.File, held.Ok = result.File, result.OK
}

func allocation_lookup(held *allocation_fixture, result types.Lookup_Result) {
  held.Symbol, held.Ok = result.Symbol, result.Found
}

// Builds the fixture module without shadowing the value one case holds.
func fixture_module(t *testing.T) (subject *types.Module, file types.File_Index) {
  t.Helper()
  return fixture(t)
}

// Runs every pass over the body fixture, which is what a caller does with a module it checks.
func body_fixture(t *testing.T) (subject *types.Module, body *types.Body, tree *ast.Parse_State) {
  t.Helper()
  subject, tree = module_of()
  body = body_state()
  files := []fixture_file{
    {Path: TEST_OTHER_PATH, Source: TEST_OTHER_SOURCE},
    {Path: TEST_PATH, Source: TEST_BODY_SOURCE},
  }
  indexes := analyse(t, subject, tree, files)
  for slot, one := range files {
    ast.Parse(tree, token.Source(one.Source))
    testify.True(
      t,
      bool(types.Check(subject, body, indexes[slot], tree)),
      "the file checks: %s",
      types.Failure_Message(types.Failure(subject)),
    )
  }
  ast.Parse(tree, token.Source(TEST_BODY_SOURCE))
  types.Check(subject, body, indexes[1], tree)
  return subject, body, tree
}

// Names the tree slot of one node the fixture states, which is how a case points at the
// expression it means.
func node_of(
  t *testing.T,
  tree *ast.Parse_State,
  kind ast.Node_Kind,
  text string,
  place int,
) (node ast.Index) {
  t.Helper()
  found := ast.Index(0)
  seen := 0
  walk_nodes(tree, 1, func(index ast.Index) {
    node_match(tree, index, kind, TEST_BODY_SOURCE, text, place, &seen, &found)
  })
  testify.Not_Equal(t, ast.Index(0), found, "the fixture states %q", text)
  return found
}

func node_match(
  tree *ast.Parse_State,
  index ast.Index,
  kind ast.Node_Kind,
  source string,
  text string,
  place int,
  seen *int,
  found *ast.Index,
) {
  one := ast.Node_At(tree, index)
  if one.Kind != kind {
    return
  }
  if string(token.Text(token.Source(source), ast.Token_At(tree, one.Token))) != text {
    return
  }
  if *seen == place {
    *found = index
  }
  *seen = *seen + 1
}

// Walks every slot of one tree, which is how a case finds the node it means.
func walk_nodes(tree *ast.Parse_State, index ast.Index, visit func(index ast.Index)) {
  if index == 0 {
    return
  }
  visit(index)
  one := ast.Node_At(tree, index)
  walk_nodes(tree, ast.Index(one.First_Child), visit)
  walk_nodes(tree, ast.Index(one.Next), visit)
}

// Reads the kind of the type one node folded to.
func folded_kind(
  t *testing.T,
  subject *types.Module,
  body *types.Body,
  tree *ast.Parse_State,
  kind ast.Node_Kind,
  text string,
  place int,
) (folded types.Type_Kind) {
  t.Helper()
  node := node_of(t, tree, kind, text, place)
  return types.Type_Kind_Of(subject, types.Type_At(body, node))
}

func test_bodies(t *testing.T) {
  subject, body, tree := body_fixture(t)
  testify.Equal(
    t,
    types.FAILURE_NONE,
    types.Failure(subject),
    "a body the module reads holds no fault",
  )
  testify.Equal(
    t,
    types.TYPE_UNTYPED_INTEGER,
    folded_kind(t, subject, body, tree, ast.NODE_INTEGER, "0", 0),
    "a literal folds to the type its own spelling states",
  )
  testify.Equal(
    t,
    types.TYPE_INVALID,
    types.Type_Kind_Of(subject,
      types.Type_At(body, ast.Index(ast.NODE_COUNT_MAXIMUM-1))),
    "a slot the pass folded nothing into names the invalid type",
  )
  testify.Equal(
    t,
    types.TYPE_INVALID,
    types.Type_Kind_Of(subject, types.Type_At(body, 0)),
    "the absent slot names the invalid type",
  )
  testify.Equal(
    t,
    types.TYPE_INVALID,
    types.Type_Kind_Of(subject, types.Type_At(body, 1)),
    "the file slot names no type of its own",
  )
  for _, one := range []struct {
    Text string
    Kind types.Type_Kind
  }{
    {"truth", types.TYPE_BOOLEAN},
    {"letters", types.TYPE_STRING},
  } {
    testify.Equal(
      t,
      one.Kind,
      use_kind(t, subject, body, tree, one.Text),
      "the name %q wears the type its declaration states",
      one.Text,
    )
  }
  bodies_files(t)
  bodies_final_type(t)
}

// Checks the final file of a full module, and one file a refused module holds.
func bodies_files(t *testing.T) {
  t.Helper()
  subject, tree := module_of()
  body := body_state()
  held := types.File_Index(0)
  for index := range types.FILE_COUNT_MAXIMUM {
    source := token.Source(
      fmt.Sprintf(
        "package one\n\nfunc Only%d() (one int) {\n\treturn 1\n}\n", index,
      ),
    )
    result := types.Add_File(subject, TEST_PATH, source)
    file := result.File
    testify.True(t, bool(result.OK), "a file of the fill binds")
    if index != 2 {
      if index != types.FILE_COUNT_MAXIMUM-1 {
        continue
      }
    }
    held = file
    ast.Parse(tree, source)
    types.Declare(subject, file, tree)
    ast.Parse(tree, source)
    testify.True(
      t,
      bool(types.Check(subject, body, file, tree)),
      "the file at slot %d checks",
      file,
    )
  }
  over := types.Add_File(subject, TEST_PATH, token.Source("package one\n"))
  testify.False(t, bool(over.OK), "a file past the bound is refused")
  ast.Parse(tree, token.Source("package one\n"))
  testify.False(
    t,
    bool(types.Check(subject, body, held, tree)),
    "a module that already refused something reports its bodies refused",
  )
}

// Checks a body whose pointer takes the final type slot, which is what a caller reads back.
func bodies_final_type(t *testing.T) {
  t.Helper()
  tail := "package fill\n\nfunc Only(one int) (two int) {\n\tpointer := &one\n\t" +
    "return *pointer\n}\n"
  subject, file := fill_arena(type_head(t), tail)
  body := body_state()
  tree := parse_state()
  ast.Parse(tree, token.Source(tail))
  types.Check(subject, body, file, tree)
  found := types.Type_Index(0)
  walk_nodes(tree, 1, func(index ast.Index) {
    unary_type(tree, body, index, &found)
  })
  testify.Equal(
    t,
    types.Type_Index(types.TYPE_INDEX_MAXIMUM),
    found,
    "the address of a value takes the final type slot",
  )
}

func unary_type(tree *ast.Parse_State, body *types.Body, index ast.Index, found *types.Type_Index) {
  if ast.Node_At(tree, index).Kind != ast.NODE_UNARY {
    return
  }
  if *found != 0 {
    return
  }
  *found = types.Type_At(body, index)
}

func test_statements(t *testing.T) {
  subject, body, tree := body_fixture(t)
  for _, one := range []struct {
    Text string
    Kind types.Type_Kind
  }{
    {"total", types.TYPE_UNTYPED_INTEGER},
    {"place", types.TYPE_INT},
    {"one", types.TYPE_NAMED},
    {"ok", types.TYPE_UNTYPED_BOOLEAN},
    {"found", types.TYPE_NAMED},
    {"stated", types.TYPE_NAMED},
    {"LIMIT", types.TYPE_NAMED},
    {"held", types.TYPE_NAMED},
    {"pointer", types.TYPE_POINTER},
    {"entries", types.TYPE_SLICE},
    {"lookup", types.TYPE_MAP},
    {"letter", types.TYPE_UINT_8},
    {"part", types.TYPE_SLICE},
    {"made", types.TYPE_SLICE},
    {"fresh", types.TYPE_POINTER},
    {"call", types.TYPE_FUNCTION},
    {"number", types.TYPE_INT_64},
    {"deep", types.TYPE_NAMED},
    {"outside", types.TYPE_NAMED},
  } {
    testify.Equal(
      t,
      one.Kind,
      use_kind(t, subject, body, tree, one.Text),
      "the name %q wears the type its statement bound",
      one.Text,
    )
  }
}

// Reads the kind of the type one name folded to where a body reads it, which is the first slot
// the pass wrote a type into for that name.
func use_kind(
  t *testing.T,
  subject *types.Module,
  body *types.Body,
  tree *ast.Parse_State,
  text string,
) (folded types.Type_Kind) {
  t.Helper()
  found := types.TYPE_INVALID
  seen := false
  walk_nodes(tree, 1, func(index ast.Index) {
    identifier_type(subject, body, tree, index, text, &found, &seen)
  })
  testify.True(t, seen, "the fixture reads the name %q", text)
  return found
}

func test_expressions(t *testing.T) {
  subject, body, tree := body_fixture(t)
  for _, one := range []struct {
    Kind  ast.Node_Kind
    Text  string
    Place int
    Folds types.Type_Kind
  }{
    {ast.NODE_STRING, "\"abc\"", 0, types.TYPE_UNTYPED_STRING},
    {ast.NODE_UNARY, "&", 0, types.TYPE_POINTER},
    {ast.NODE_INDEX, "[", 0, types.TYPE_UINT_8},
    {ast.NODE_SLICE_EXPRESSION, "[", 0, types.TYPE_SLICE},
    {ast.NODE_COMPOSITE, "{", 0, types.TYPE_NAMED},
    {ast.NODE_FUNCTION_LITERAL, "func", 0, types.TYPE_FUNCTION},
    {ast.NODE_PARENTHESIS, "(", 0, types.TYPE_NAMED},
  } {
    testify.Equal(
      t,
      one.Folds,
      folded_kind(t, subject, body, tree, one.Kind, one.Text, one.Place),
      "the form %q folds to the type Go states",
      one.Text,
    )
  }
  expression_calls(t, subject, body, tree)
  expression_selectors(t, subject, body, tree)
  for _, one := range []struct {
    Text  string
    Place int
    Folds types.Type_Kind
  }{
    {"asserted", 1, types.TYPE_NAMED},
    {"kind", 1, types.TYPE_NAMED},
    {"grouped", 1, types.TYPE_POINTER},
  } {
    kind := ast.NODE_IDENTIFIER
    folded := folded_kind(t, subject, body, tree, kind, one.Text, one.Place)
    testify.Equal(t, one.Folds, folded, "%q wears the type Go states", one.Text)
  }
  for _, width_size := range []int{1, 2, types.NAME_SIZE_MAXIMUM} {
    wide_names(t, width_size)
  }
}

// TEST_WIDE_PATH is the package the program of wide names imports.
const TEST_WIDE_PATH = "local/example/wide"

// Checks one program whose every name spans the width it states: a local, a field, a field an
// embedded struct carries, an import, a member of the package it names, a package value, and a
// function that calls itself.
func identifier_type(
  subject *types.Module,
  body *types.Body,
  tree *ast.Parse_State,
  index ast.Index,
  text string,
  found *types.Type_Kind,
  seen *bool,
) {
  if *seen {
    return
  }
  one := ast.Node_At(tree, index)
  if one.Kind != ast.NODE_IDENTIFIER {
    return
  }
  if string(token.Text(token.Source(TEST_BODY_SOURCE), ast.Token_At(tree, one.Token))) != text {
    return
  }
  if types.Type_At(body, index) == 0 {
    return
  }
  *found = types.Type_Kind_Of(subject, types.Type_At(body, index))
  *seen = true
}

func repeated_letter(letter byte, width_size int) (text string) {
  name := make([]byte, width_size)
  for index := range name {
    name[index] = letter
  }
  return string(name)
}

func wide_names(t *testing.T, width_size int) {
  spelled := func(letter byte) (text string) { return repeated_letter(letter, width_size) }
  local, field, carried := spelled('l'), spelled('F'), spelled('G')
  imported, member, value, called := spelled('i'), spelled('M'), spelled('p'), spelled('c')
  other := "package wide\n\nvar " + member + " int\n"
  source := "package one\n\nimport " +
    imported +
    " \"" +
    TEST_WIDE_PATH +
    "\"\n\n" +
    "type Inner struct {\n\t" +
    carried +
    " int\n}\n\n" +
    "type Outer struct {\n\t" +
    field +
    " int\n\tInner\n}\n\n" +
    "var " +
    value +
    " Outer\n\n" +
    "func " +
    called +
    "(" +
    local +
    " Outer) (result int) {\n\treturn " +
    local +
    "." +
    field +
    " + " +
    local +
    "." +
    carried +
    " + " +
    imported +
    "." +
    member +
    " + " +
    value +
    "." +
    field +
    " + " +
    called +
    "(" +
    local +
    ")\n}\n"
  subject, tree := module_of()
  files := []fixture_file{
    {Path: TEST_WIDE_PATH, Source: other},
    {Path: TEST_PATH, Source: source},
  }
  indexes := analyse(t, subject, tree, files)
  for slot, one := range files {
    ast.Parse(tree, token.Source(one.Source))
    testify.True(
      t,
      bool(types.Check(subject, body_state(), indexes[slot], tree)),
      "a program whose names span %d bytes checks",
      width_size,
    )
  }
}

func expression_calls(
  t *testing.T,
  subject *types.Module,
  body *types.Body,
  tree *ast.Parse_State,
) {
  t.Helper()
  for _, one := range []struct {
    Place int
    Folds types.Type_Kind
  }{
    {0, types.TYPE_INT},
    {1, types.TYPE_NAMED},
    {2, types.TYPE_INT},
    {3, types.TYPE_SLICE},
    {4, types.TYPE_POINTER},
    {5, types.TYPE_NAMED},
    {6, types.TYPE_INT_64},
  } {
    testify.Equal(
      t,
      one.Folds,
      folded_kind(t, subject, body, tree, ast.NODE_CALL, "(", one.Place),
      "the call at place %d folds to the type it states",
      one.Place,
    )
  }
}

func expression_selectors(
  t *testing.T,
  subject *types.Module,
  body *types.Body,
  tree *ast.Parse_State,
) {
  t.Helper()
  for _, one := range []struct {
    Place int
    Folds types.Type_Kind
  }{
    {0, types.TYPE_NAMED},
    {1, types.TYPE_NAMED},
    {2, types.TYPE_NAMED},
    {3, types.TYPE_NAMED},
    {4, types.TYPE_FUNCTION},
    {5, types.TYPE_NAMED},
    {6, types.TYPE_NAMED},
    {7, types.TYPE_NAMED},
  } {
    testify.Equal(
      t,
      one.Folds,
      folded_kind(t, subject, body, tree, ast.NODE_SELECTOR, ".", one.Place),
      "the selector at place %d folds to the type it reads",
      one.Place,
    )
  }
}

// BENCHMARK_TOKEN_DENSITY is the widest byte count one token of the benchmark source spends, thus
// a scan that reads the whole source states at least one token for each stretch of that width.
const BENCHMARK_TOKEN_DENSITY = 8

// BENCHMARK_BLOCK_COUNT is how many declaration blocks the benchmark source states. It stands
// over one file a caller writes by hand, thus the measurement reads a module and not a snippet.
const BENCHMARK_BLOCK_COUNT = 100

// Builds the source every benchmark reads: one type, one struct, and one function for each block,
// which is the shape a caller feeds this module.
func benchmark_source() (source string) {
  buffer := make([]byte, 0, BENCHMARK_BLOCK_COUNT*400)
  buffer = append(buffer, "package bench\n"...)
  for index := range BENCHMARK_BLOCK_COUNT {
    buffer = append(buffer, fmt.Sprintf(`
type Count%d int

type Entry%d struct {
	Value Count%d
	Mark  Count%d
}

func Fold%d(entry Entry%d, scale Count%d) (sum Count%d) {
	total := entry.Value
	for step := range 4 {
		total = total + Count%d(step)
	}
	if scale > 0 {
		total = total + scale
	}
	return total + entry.Mark
}
`, index, index, index, index, index, index, index, index, index)...)
  }
  return string(buffer)
}

// Proves one house pass read the whole source, thus a measurement never states the speed of a
// pass that stopped early.
func benchmark_parity(
  b *testing.B,
  subject *types.Module,
  tree *ast.Parse_State,
  body *types.Body,
  source token.Source,
) {
  b.Helper()
  types.Reset(subject)
  if result := ast.Parse(tree, source); result.Failure != ast.FAILURE_NONE {
    b.Fatal("the benchmark source parses")
  }
  file := types.Add_File(subject, TEST_PATH, source).File
  types.Declare(subject, file, tree)
  types.Resolve(subject, file, tree)
  types.Check(subject, body, file, tree)
  if types.Failure(subject) != types.FAILURE_NONE {
    b.Fatal(string(types.Failure_Message(types.Failure(subject))))
  }
  if result := types.Lookup(subject, types.Package_Of(subject, file),
    types.Name("Fold0")); !bool(result.Found) {
    b.Fatal("the benchmark source states its names")
  }
}

func Benchmark_Analyse_House(b *testing.B) {
  text := benchmark_source()
  source := token.Source(text)
  subject := module_state()
  tree := parse_state()
  body := body_state()
  benchmark_parity(b, subject, tree, body, source)
  b.SetBytes(int64(len(source)))
  b.ReportAllocs()
  for b.Loop() {
    types.Reset(subject)
    ast.Parse(tree, source)
    file := types.Add_File(subject, TEST_PATH, source).File
    types.Declare(subject, file, tree)
    types.Resolve(subject, file, tree)
    types.Check(subject, body, file, tree)
  }
}

func Benchmark_Parse_House(b *testing.B) {
  source := token.Source(benchmark_source())
  tree := parse_state()
  if result := ast.Parse(tree, source); result.Failure != ast.FAILURE_NONE {
    b.Fatal(string(ast.Failure_Message(result.Failure)))
  }
  b.SetBytes(int64(len(source)))
  b.ReportAllocs()
  for b.Loop() {
    ast.Parse(tree, source)
  }
}

func Benchmark_Scan_House(b *testing.B) {
  source := token.Source(benchmark_source())
  scanned := token.Scanner{Source: source}
  count := 0
  for one := token.Scan(&scanned); one.Kind != token.KIND_END_OF_FILE; {
    one = token.Scan(&scanned)
    count = count + 1
  }
  if count < len(source)/BENCHMARK_TOKEN_DENSITY {
    b.Fatal("the scan reads the whole source")
  }
  b.SetBytes(int64(len(source)))
  b.ReportAllocs()
  for b.Loop() {
    scanner := token.Scanner{Source: source}
    one := token.Scan(&scanner)
    for one.Kind != token.KIND_END_OF_FILE {
      one = token.Scan(&scanner)
    }
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
    "var a = 1\n",
    "package one\n\ntype R interface{\n\tRead() int\n}\n",
    string(nested),
    string(filled),
  }
}

// TEST_BODY_PATH is the package the body fixture stands in when it shares a module with the
// declaration fixtures, apart from the names they state.
const TEST_BODY_PATH = "local/example/body"

// Binds an import that states no name of its own, which takes the segment after the final slash
// of its path.
func declared_tail_import(t *testing.T) {
  subject, tree := module_of()
  files := analyse(
    t,
    subject,
    tree,
    []fixture_file{
      {Path: TEST_OTHER_PATH, Source: TEST_OTHER_SOURCE},
      {Path: TEST_PATH, Source: "package one\n\nimport \"local/example/two\"\n"},
    },
  )
  binding := import_of(t, subject, files[1], "two")
  target := types.Symbol_Target_Of(subject, binding)
  testify.Equal(
    t,
    types.Package_Of(subject, files[0]),
    target,
    "an import with no name of its own names the package its path states",
  )
}

// TEST_MISSING_PATH is the package a refused module reads a missing name in, which refuses a
// module that already refused once more.
const TEST_MISSING_PATH = "local/example/missing"

// TEST_CYCLE_PATH is the package the cycle a refused module opens with stands in, apart from the
// names the fixture files state.
const TEST_CYCLE_PATH = "local/example/cycle"

// Refuses a module by filling its file slots, which the files already bound survive.
func refuse_files(t *testing.T, subject *types.Module) {
  t.Helper()
  for bool(types.Add_File(subject, TEST_PATH, token.Source("package one\n")).OK) {
  }
}

// Refuses a module by filling its package slots, which the files already bound survive.
func refuse_packages(t *testing.T, subject *types.Module) {
  t.Helper()
  for index := 0; ; index++ {
    path := types.Path(fmt.Sprintf("local/example/fill%d", index))
    if !bool(types.Add_File(subject, path, token.Source("package fill\n")).OK) {
      return
    }
  }
}

// Refuses a module with a type that names itself, in a package of its own.
func refuse_cycle(t *testing.T, subject *types.Module) {
  t.Helper()
  source := token.Source("package cycle\n\ntype Loop Loop\n")
  file := types.Add_File(subject, TEST_CYCLE_PATH, source).File
  tree := parse_state()
  ast.Parse(tree, source)
  types.Declare(subject, file, tree)
  types.Resolve(subject, file, tree)
}

// Refuses one module by checking a body that binds a name a block around it binds.
func refuse_shadow(t *testing.T, subject *types.Module) {
  t.Helper()
  source := token.Source(
    "package shadow\n\nfunc F() {\n  held := 1\n" +
      "  {\n    held := 2\n    _ = held\n  }\n  _ = held\n}\n",
  )
  file := types.Add_File(subject, TEST_SHADOW_PATH, source).File
  tree := parse_state()
  ast.Parse(tree, source)
  types.Declare(subject, file, tree)
  ast.Parse(tree, source)
  types.Resolve(subject, file, tree)
  ast.Parse(tree, source)
  types.Check(subject, body_state(), file, tree)
}

// TEST_SHADOW_PATH names the package one shadowed name stands in.
const TEST_SHADOW_PATH = "local/example/shadow"

// Refuses one module with a type expression that names a function, which is the last code a
// refusal states.
func refuse_value_type(t *testing.T, subject *types.Module) {
  t.Helper()
  source := token.Source("package value\n\nfunc F() {}\n\nvar held F\n")
  file := types.Add_File(subject, TEST_VALUE_PATH, source).File
  tree := parse_state()
  ast.Parse(tree, source)
  types.Declare(subject, file, tree)
  types.Resolve(subject, file, tree)
}

// TEST_VALUE_PATH names the package one type expression that names a value stands in.
const TEST_VALUE_PATH = "local/example/value"

// Runs every pass over the fixture files of a module that already refused, once for each report a
// module keeps at its bounds: the first two codes and the last. The first cause stands through
// every later pass, because those passes read state the refusal already left half built.
func refused_module_passes(t *testing.T) {
  for _, one := range []struct {
    Code   types.Failure_Code
    Refuse func(*testing.T, *types.Module)
  }{
    {types.FAILURE_FILE_COUNT, refuse_files},
    {types.FAILURE_PACKAGE_COUNT, refuse_packages},
    {types.FAILURE_SHADOWED_NAME, refuse_shadow},
    {types.FAILURE_NOT_TYPE, refuse_value_type},
  } {
    subject, tree := module_of()
    files := []fixture_file{
      {Path: TEST_OTHER_PATH, Source: TEST_OTHER_SOURCE},
      {Path: TEST_PATH, Source: TEST_SOURCE},
      {Path: TEST_PATH, Source: TEST_THIRD_SOURCE},
      {Path: TEST_BODY_PATH, Source: TEST_BODY_SOURCE},
      {Path: TEST_MISSING_PATH, Source: "package missing\n\nvar value Missing\n"},
    }
    indexes := []types.File_Index{}
    for _, file := range files {
      path := types.Path(file.Path)
      result := types.Add_File(subject, path, token.Source(file.Source))
      indexes = append(indexes, result.File)
    }
    one.Refuse(t, subject)
    testify.Equal(t, one.Code, types.Failure(subject), "the module refuses at a bound")
    for slot, file := range files {
      ast.Parse(tree, token.Source(file.Source))
      types.Declare(subject, indexes[slot], tree)
    }
    for slot, file := range files {
      ast.Parse(tree, token.Source(file.Source))
      types.Resolve(subject, indexes[slot], tree)
      ast.Parse(tree, token.Source(file.Source))
      types.Check(subject, body_state(), indexes[slot], tree)
    }
    testify.Equal(
      t,
      one.Code,
      types.Failure(subject),
      "the first cause stands through every later pass",
    )
    reads_refused_module(t, subject, indexes[1])
    types.Reset(subject)
    testify.Equal(
      t,
      types.FAILURE_NONE,
      types.Failure(subject),
      "a reset module stands again, whatever the module before it refused",
    )
  }
}

// Reads a module that refused through every accessor a caller holds, which answers the way it
// answers for a module that stands, and binds one more file past the refusal.
func reads_refused_module(t *testing.T, subject *types.Module, file types.File_Index) {
  t.Helper()
  owner := types.Package_Of(subject, file)
  testify.Equal(
    t,
    types.Path(TEST_PATH),
    types.Path_Of(subject, owner),
    "a refused module still names the path of a package",
  )
  testify.True(
    t,
    !(types.Unpaired(subject, owner).Symbol < types.PACKAGE_SYMBOL_FIRST),
    "a refused module still answers for the functions it pairs",
  )
  testify.Equal(
    t,
    token.Source(TEST_SOURCE),
    types.Source_Of(subject, file),
    "a refused module still holds a source",
  )
  count := types.Lookup(subject, owner, types.Name("Count"))
  testify.True(t, bool(count.Found), "a refused module still binds the names it declared")
  symbol := count.Symbol
  testify.Equal(
    t,
    types.SYMBOL_TYPE,
    types.Symbol_Kind_Of(subject, symbol),
    "a refused module still reads the kind of a symbol",
  )
  testify.Equal(
    t,
    types.Name("Count"),
    types.Symbol_Name_Of(subject, symbol),
    "a refused module still reads the name of a symbol",
  )
  testify.Equal(
    t,
    file,
    types.Symbol_File_Of(subject, symbol),
    "a refused module still reads the file of a symbol",
  )
  types.Symbol_Target_Of(subject, symbol)
  named := types.Symbol_Type_Of(subject, symbol)
  testify.Equal(
    t,
    types.TYPE_NAMED,
    types.Type_Kind_Of(subject, named),
    "a refused module still reads the kind of a type",
  )
  testify.Equal(
    t,
    symbol,
    types.Symbol_Index(types.Type_Symbol_Of(subject, named)),
    "a refused module still reads the declaration of a type",
  )
  named_entry := types.Lookup(subject, owner, types.Name("Entry")).Symbol
  entry := types.Underlying_Of(subject, types.Symbol_Type_Of(subject, named_entry))
  member := types.Member_Index(types.Type_Members_Of(subject, entry))
  types.Member_Type_Of(subject, member)
  types.Member_Symbol_Of(subject, member)
  types.Member_Next_Of(subject, member)
  types.Type_Element_Of(subject, named)
  types.Type_Key_Of(subject, named)
  types.Type_Element_Count_Of(subject, named)
  types.Add_File(subject, TEST_PATH, token.Source("package one\n"))
}

// A function that states the invariants of a type stands directly below that type, thus a package
// that declares no type of that name states a function standing below nothing. The read names the
// first such function, and names none where every one of them pairs with a type of the package.
func test_invariant_pairs(t *testing.T) {
  subject, tree := module_of()
  held := analyse(
    t,
    subject,
    tree,
    []fixture_file{
      {
        Path: TEST_PATH,
        Source: "package one\n\ntype Pair int\n\n" +
          "func Pair_Invariants() {\n}\n",
      },
    },
  )
  owner := types.Package_Of(subject, held[0])
  testify.False(
    t,
    bool(types.Unpaired(subject, owner).Found),
    "a function that names a type of the package pairs with it",
  )
  other, second := module_of()
  stood := analyse(
    t,
    other,
    second,
    []fixture_file{
      {Path: TEST_PATH, Source: "package one\n\nfunc Held_Invariants() {\n}\n"},
    },
  )
  report := types.Unpaired(other, types.Package_Of(other, stood[0]))
  testify.True(
    t,
    bool(report.Found),
    "a function that names no type of the package stands below nothing",
  )
  testify.Equal(
    t,
    types.Name("Held_Invariants"),
    types.Symbol_Name_Of(other, types.Symbol_Index(report.Symbol)),
    "the report names that function",
  )
  pairs_second_symbol(t)
  pairs_final_symbol(t)
}

// The report names the function wherever the package states it: the slot behind the first name a
// file binds, and the last slot the arena holds.
func pairs_second_symbol(t *testing.T) {
  subject, tree := module_of()
  held := analyse(
    t,
    subject,
    tree,
    []fixture_file{
      {Path: TEST_PATH, Source: "package one\n\nvar pad int\n\n" +
        "func Held_Invariants() {\n}\n"},
    },
  )
  report := types.Unpaired(subject, types.Package_Of(subject, held[0]))
  testify.Equal(
    t,
    types.Name("Held_Invariants"),
    types.Symbol_Name_Of(subject, types.Symbol_Index(report.Symbol)),
    "a function behind one name reads back from the slot behind it",
  )
}

// Reads the report of a module whose final symbol slot holds such a function.
func pairs_final_symbol(t *testing.T) {
  subject, file := fill_arena(
    symbol_head(t),
    "package fill\n\nvar pad int\n",
    "package fill\n\nfunc Gone_Invariants() {\n}\n",
  )
  report := types.Unpaired(subject, types.Package_Of(subject, file))
  testify.Equal(
    t,
    types.Unpaired_Slot(types.SYMBOL_INDEX_MAXIMUM),
    report.Symbol,
    "a function in the final slot reads back from it",
  )
}

// A name one block binds that a block around it already binds hides that binding, thus the check
// refuses it. A second binding of one name in the one block states an assignment, and two blocks
// that stand beside each other hide nothing from each other.
func test_hidden_names(t *testing.T) {
  for _, one := range []string{
    "package one\n\nfunc F() {\n  held := 1\n  {\n    held := 2\n" +
      "    _ = held\n  }\n  _ = held\n}\n",
    "package one\n\nfunc F(held int) {\n  {\n    held := 2\n    _ = held\n  }\n}\n",
    "package one\n\nfunc F() {\n  held := 1\n  for step := range 2 {\n" +
      "    held := step\n    _ = held\n  }\n  _ = held\n}\n",
  } {
    testify.Equal(
      t,
      types.FAILURE_SHADOWED_NAME,
      checked_failure(t, one),
      "a name a block around it binds stands nowhere",
    )
  }
  for _, one := range []string{
    "package one\n\nfunc F() {\n  held, other := g()\n  held, third := g()\n" +
      "  _, _, _ = held, other, third\n}\n",
    "package one\n\nfunc F() {\n  {\n    held := 1\n    _ = held\n  }\n" +
      "  {\n    held := 2\n    _ = held\n  }\n}\n",
  } {
    testify.Equal(
      t,
      types.FAILURE_NONE,
      checked_failure(t, one),
      "a binding that hides nothing stands",
    )
  }
}

// Checks one body source and reads why the module refused, or that it stands.
func checked_failure(t *testing.T, source string) (code types.Failure_Code) {
  t.Helper()
  subject, tree := module_of()
  file := types.Add_File(subject, TEST_PATH, token.Source(source)).File
  ast.Parse(tree, token.Source(source))
  types.Declare(subject, file, tree)
  ast.Parse(tree, token.Source(source))
  types.Resolve(subject, file, tree)
  ast.Parse(tree, token.Source(source))
  types.Check(subject, body_state(), file, tree)
  return types.Failure(subject)
}

// Runs both passes over every file and reads why the module refused, or that it stands.
func resolved_failure(t *testing.T, files []fixture_file) (code types.Failure_Code) {
  t.Helper()
  subject := module_state()
  types.Reset(subject)
  indexes := []types.File_Index{}
  for _, one := range files {
    result := types.Add_File(subject, types.Path(one.Path), token.Source(one.Source))
    indexes = append(indexes, result.File)
  }
  trees := []*ast.Parse_State{}
  for slot, one := range files {
    trees = append(trees, parse_state())
    ast.Parse(trees[slot], token.Source(one.Source))
    types.Declare(subject, indexes[slot], trees[slot])
  }
  for slot := range files {
    types.Resolve(subject, indexes[slot], trees[slot])
  }
  return types.Failure(subject)
}

// Reads the failure one source of the first package refuses with, or that it stands.
func source_failure(t *testing.T, source string) (code types.Failure_Code) {
  t.Helper()
  return resolved_failure(
    t,
    []fixture_file{
      {Path: TEST_OTHER_PATH, Source: TEST_OTHER_SOURCE},
      {Path: TEST_PATH, Source: source},
    },
  )
}

// Asserts every source refuses with one code.
func refuses_with(t *testing.T, code types.Failure_Code, sources []string, message string) {
  t.Helper()
  for _, one := range sources {
    testify.Equal(t, code, source_failure(t, one), "%s: %q", message, one)
  }
}

// A name a type expression states names a type: a predeclared one, one its package declares, a
// type parameter in scope, or one an imported package declares. A constant, a variable, a
// function, and a builtin name no type.
func test_type_resolution(t *testing.T) {
  refuses_with(
    t,
    types.FAILURE_NOT_TYPE,
    []string{
      "package one\n\nfunc F() {}\n\nvar value F\n",
      "package one\n\nconst LIMIT = 1\n\ntype Box struct {\n\tSize LIMIT\n}\n",
      "package one\n\nvar held int\n\nfunc F(one held) {}\n",
      "package one\n\nfunc F() (one len) {\n\treturn one\n}\n",
      "package one\n\nimport two \"local/example/two\"\n\nvar value two.MARK\n",
      "package one\n\nvar value []*map[string]nil\n",
    },
    "a name that states no type stands in no type expression",
  )
  refuses_with(
    t,
    types.FAILURE_NONE,
    []string{
      "package one\n\ntype Count int\n\nvar value Count\n",
      "package one\n\nvar value int\n\nvar fault error\n\nvar some any\n",
      "package one\n\ntype Box[Item any] struct {\n\tValue Item\n}\n",
      "package one\n\nimport two \"local/example/two\"\n\nvar value two.Marker\n",
      "package one\n\nfunc F[Item comparable](one Item) (two Item) {\n\treturn one\n}\n",
    },
    "a name that states a type stands in a type expression",
  )
}

// A type the package declares is exported. A type a function body states stands inside that body
// alone, thus no declaration of it reaches another package.
func test_exported_types(t *testing.T) {
  refuses_with(
    t,
    types.FAILURE_UNEXPORTED_TYPE,
    []string{
      "package one\n\ntype count int\n",
      "package one\n\ntype _ int\n",
      "package one\n\ntype (\n\tLeft int\n\tright int\n)\n",
      "package one\n\ntype box[Item any] struct {\n\tValue Item\n}\n",
    },
    "a package type is exported",
  )
  refuses_with(
    t,
    types.FAILURE_NONE,
    []string{
      "package one\n\nfunc F() {\n\ttype held int\n}\n",
    },
    "an exported type or a local type stands",
  )
}

// An exported struct exposes no field of an unexported type, through any struct its package
// declares. No package declares an unexported type, thus the refusal of that declaration is the
// refusal of the field that reaches it.
func test_exported_struct_fields(t *testing.T) {
  refuses_with(
    t,
    types.FAILURE_UNEXPORTED_TYPE,
    []string{
      "package one\n\ntype Outer struct {\n\tInner inner\n}\n\ntype inner struct {\n\tValue int\n}\n",
      "package one\n\ntype Outer struct {\n\tInner *Middle\n}\n\n" +
        "type Middle struct {\n\tDeep []deep\n}\n\ntype deep int\n",
      "package one\n\ntype Outer struct {\n\tMiddle\n}\n\n" +
        "type Middle struct {\n\tDeep map[string]*deep\n}\n\ntype deep int\n",
    },
    "an exported struct reaches no unexported type",
  )
  refuses_with(
    t,
    types.FAILURE_NONE,
    []string{
      "package one\n\ntype Outer struct {\n\tInner *Middle\n}\n\n" +
        "type Middle struct {\n\tDeep Deep\n}\n\ntype Deep int\n",
    },
    "an exported struct that reaches exported types stands",
  )
}

// The type of a field of an exported struct is exported. A predeclared type, a type parameter,
// and a type another package exports each name a type a caller spells.
func test_struct_field_type_encapsulation(t *testing.T) {
  refuses_with(
    t,
    types.FAILURE_UNEXPORTED_TYPE,
    []string{
      "package one\n\ntype Outer struct {\n\tHeld held\n}\n\ntype held int\n",
      "package one\n\ntype Outer struct {\n\tHeld *held\n}\n\ntype held int\n",
    },
    "an exported struct holds no field of an unexported type",
  )
  refuses_with(
    t,
    types.FAILURE_NONE,
    []string{
      "package one\n\ntype Outer struct {\n\tCount int\n\tText string\n\tFault error\n}\n",
      "package one\n\ntype Outer[Item any] struct {\n\tHeld Item\n}\n",
      "package one\n\nimport two \"local/example/two\"\n\n" +
        "type Outer struct {\n\tMark two.Marker\n}\n",
    },
    "a field of a type a caller spells stands",
  )
}
