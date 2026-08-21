package types_test

import (
	"fmt"
	standard_ast "go/ast"
	"go/parser"
	"go/scanner"
	standard_token "go/token"
	standard_types "go/types"
	"testing"

	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/go/types"
	"local/james-orcales/shared/testify"
)

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

// Test_Bodies binds the Bodies specification leaf before fixture declarations.
func Test_Bodies(t *testing.T) {
	test_bodies(t)
}

// Test_Statements binds the Statements specification leaf before fixture declarations.
func Test_Statements(t *testing.T) {
	test_statements(t)
}

// Test_Expressions binds the Expressions specification leaf before fixture declarations.
func Test_Expressions(t *testing.T) {
	test_expressions(t)
}

// Test_Answers binds the Answers specification leaf before fixture declarations.
func Test_Answers(t *testing.T) {
	test_answers(t)
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
// takes that many type slots and one symbol slot.
const TEST_POINTER_RUN = 10

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
	return place_of(true) + sum + mark + read + outside + Count(letter) + Count(len(part)) +
		Count(len(made)) + fresh.Value + Count(number) + folded + stated + LIMIT +
		deep + pointer.Value + Count(total) + kept + Count(len(letters)) + Count(place_of(truth))
}
`

type allocation_fixture struct {
	File   types.File_Index
	Symbol types.Symbol_Index
	Type   types.Type_Index
	Ok     types.Boolean
}

type fixture_file struct {
	Path   string
	Source string
}

// Builds one module and one tree the caller owns, which is how every case here starts.
func module_of() (subject *types.Module, tree *ast.Parse_State) {
	subject = new(types.Module)
	types.Reset(subject)
	return subject, new(ast.Parse_State)
}

// Runs both passes over every file, which is the order the specification states.
func analyse(
	t *testing.T, subject *types.Module, tree *ast.Parse_State, files []fixture_file,
) (indexes []types.File_Index) {
	t.Helper()
	for _, one := range files {
		file, added := types.Add_File(subject, types.Path(one.Path),
			token.Source(one.Source))
		testify.True(t, bool(added), "the file binds to its package")
		indexes = append(indexes, file)
	}
	for slot, one := range files {
		ast.Parse(tree, token.Source(one.Source))
		testify.True(t, bool(types.Declare(subject, indexes[slot], tree)),
			"the file declares: %s", types.Failure_Message(types.Failure(subject)))
	}
	for slot, one := range files {
		ast.Parse(tree, token.Source(one.Source))
		testify.True(t, bool(types.Resolve(subject, indexes[slot], tree)),
			"the file resolves: %s", types.Failure_Message(types.Failure(subject)))
	}
	return indexes
}

// Runs both passes over the two fixture files, which is what most cases here read.
func fixture(t *testing.T) (subject *types.Module, file types.File_Index) {
	t.Helper()
	subject, tree := module_of()
	indexes := analyse(t, subject, tree, []fixture_file{
		{Path: TEST_OTHER_PATH, Source: TEST_OTHER_SOURCE},
		{Path: TEST_PATH, Source: TEST_SOURCE},
		{Path: TEST_PATH, Source: TEST_THIRD_SOURCE},
	})
	return subject, indexes[1]
}

// Names one package member, which is how a case states what it expects to find.
func member_of(
	t *testing.T, subject *types.Module, file types.File_Index, name string,
) (symbol types.Symbol_Index) {
	t.Helper()
	found, ok := types.Lookup(subject, types.Package_Of(subject, file), types.Name(name))
	testify.True(t, bool(ok), "the package states %q", name)
	return found
}

// Reads one link a reader answers as the type slot it names.
func slot_of[Link ~int32](link Link) (index types.Type_Index) {
	return types.Type_Index(link)
}

// Reads one link a reader answers as the member slot it names.
func member_slot_of[Link ~int32](link Link) (index types.Member_Index) {
	return types.Member_Index(link)
}

// Reads one link a reader answers as the symbol slot it names.
func symbol_slot_of[Link ~int32](link Link) (index types.Symbol_Index) {
	return types.Symbol_Index(link)
}

// Names one package member and reads the kind of the type it wears.
func type_kind_of(
	t *testing.T, subject *types.Module, file types.File_Index, name string,
) (kind types.Type_Kind) {
	t.Helper()
	symbol := member_of(t, subject, file, name)
	return types.Type_Kind_Of(subject, types.Symbol_Type_Of(subject, symbol))
}

func test_module(t *testing.T) {
	subject, file := fixture(t)
	testify.Equal(t, types.FAILURE_NONE, types.Failure(subject),
		"a clean module holds no fault")
	testify.Not_Equal(t, types.SYMBOL_ABSENT, member_of(t, subject, file, "Count"),
		"a clean module answers for a name it read")
	types.Reset(subject)
	testify.Equal(t, types.FAILURE_NONE, types.Failure(subject),
		"a reset module holds no fault")
	_, found := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name("int"))
	testify.True(t, bool(found), "a reset module still holds the universe")
	_, gone := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name("Count"))
	testify.False(t, bool(gone), "a reset module holds no name a file stated")
}

func test_files(t *testing.T) {
	subject, tree := module_of()
	indexes := analyse(t, subject, tree, []fixture_file{
		{Path: TEST_PATH, Source: "package one\n\ntype Count int\n"},
		{Path: TEST_PATH, Source: "package one\n\ntype Second Count\n"},
		{Path: TEST_OTHER_PATH, Source: TEST_OTHER_SOURCE},
	})
	testify.Equal(t, types.Package_Of(subject, indexes[0]),
		types.Package_Of(subject, indexes[1]), "two files of one path share one package")
	testify.Not_Equal(t, types.Package_Of(subject, indexes[0]),
		types.Package_Of(subject, indexes[2]), "two paths make two packages")
	testify.Equal(t, token.Source(TEST_OTHER_SOURCE), types.Source_Of(subject, indexes[2]),
		"a file reads back the source the caller owns")
	second := member_of(t, subject, indexes[1], "Second")
	testify.Equal(t, types.TYPE_INT, types.Type_Kind_Of(subject,
		types.Underlying_Of(subject, types.Symbol_Type_Of(subject, second))),
		"a file reads a name another file of its package states")
}

func test_universe(t *testing.T) {
	subject, _ := module_of()
	for _, name := range []string{
		"bool", "byte", "rune", "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "float32", "float64",
		"complex64", "complex128", "error", "any", "comparable",
	} {
		symbol, found := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name(name))
		testify.True(t, bool(found), "the universe states the type %q", name)
		testify.Equal(t, types.SYMBOL_TYPE, types.Symbol_Kind_Of(subject, symbol),
			"a predeclared type names a type")
	}
	for _, name := range []string{
		"len", "cap", "make", "new", "append", "copy", "delete", "panic", "recover",
		"close", "min", "max", "clear", "complex", "real", "imag", "print", "println",
	} {
		symbol, found := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name(name))
		testify.True(t, bool(found), "the universe states the function %q", name)
		testify.Equal(t, types.SYMBOL_BUILTIN, types.Symbol_Kind_Of(subject, symbol),
			"a predeclared function names a builtin")
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
		symbol, found := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name(one.Name))
		testify.True(t, bool(found), "the universe states the value %q", one.Name)
		testify.Equal(t, one.Kind,
			types.Type_Kind_Of(subject, types.Symbol_Type_Of(subject, symbol)),
			"%q wears the type it folds to", one.Name)
	}
	_, absent := types.Lookup(subject, types.PACKAGE_UNIVERSE, types.Name("Total"))
	testify.False(t, bool(absent), "the universe states no name a file declares")
}

func test_declarations(t *testing.T) {
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
		testify.Equal(t, types.Name(one.Name), types.Symbol_Name_Of(subject, symbol),
			"%q reads its own name", one.Name)
		testify.Equal(t, file, types.Symbol_File_Of(subject, symbol),
			"%q names the file that states it", one.Name)
	}
	_, absent := types.Lookup(subject, types.Package_Of(subject, file), types.Name("Read"))
	testify.False(t, bool(absent), "a method binds no package name")
	_, unstated := types.Lookup(subject, types.Package_Of(subject, file), types.Name("Missing"))
	testify.False(t, bool(unstated), "a package states no name no file declares")
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
		testify.Equal(t, one.Kind, type_kind_of(t, subject, file, one.Name),
			"%q names its type", one.Name)
	}
	fixed := types.Symbol_Type_Of(subject, member_of(t, subject, file, "fixed"))
	testify.Equal(t, types.Element_Count(4), types.Type_Element_Count_Of(subject, fixed),
		"an array names the element_count one literal states")
	table := types.Symbol_Type_Of(subject, member_of(t, subject, file, "table"))
	testify.Equal(t, types.ELEMENT_COUNT_UNKNOWN, types.Type_Element_Count_Of(subject, table),
		"an array whose element_count no literal states names none")
	lookup := types.Symbol_Type_Of(subject, member_of(t, subject, file, "lookup"))
	testify.Equal(t, types.TYPE_NAMED, types.Type_Kind_Of(subject,
		slot_of(types.Type_Key_Of(subject, lookup))), "a map names the type of its key")
	testify.Equal(t, types.TYPE_INVALID, type_kind_of(t, subject, file, "third"),
		"a declaration that states no type wears the invalid type")
}

func test_named_types(t *testing.T) {
	subject, file := fixture(t)
	count := types.Symbol_Type_Of(subject, member_of(t, subject, file, "Count"))
	testify.Equal(t, types.TYPE_NAMED, types.Type_Kind_Of(subject, count),
		"a type declaration builds a named type")
	testify.Equal(t, types.Name("Count"), types.Symbol_Name_Of(subject,
		symbol_slot_of(types.Type_Symbol_Of(subject, count))),
		"a named type carries its declaration")
	testify.Equal(t, types.TYPE_INT,
		types.Type_Kind_Of(subject, types.Underlying_Of(subject, count)),
		"a named type walks down to its own")
	entry := types.Underlying_Of(subject,
		types.Symbol_Type_Of(subject, member_of(t, subject, file, "Entry")))
	testify.Equal(t, types.TYPE_STRUCTURE, types.Type_Kind_Of(subject, entry),
		"a struct names a struct")
	member := member_slot_of(types.Type_Members_Of(subject, entry))
	testify.Equal(t, types.Name("Name"), types.Symbol_Name_Of(subject,
		symbol_slot_of(types.Member_Symbol_Of(subject, member))), "a field names itself")
	testify.Equal(t, types.TYPE_NAMED,
		types.Type_Kind_Of(subject, slot_of(types.Member_Type_Of(subject, member))),
		"a field wears the type its declaration states")
	embedded := member_slot_of(types.Member_Next_Of(subject,
		member_slot_of(types.Member_Next_Of(subject, member))))
	testify.Equal(t, types.SYMBOL_ABSENT,
		symbol_slot_of(types.Member_Symbol_Of(subject, embedded)),
		"an embedded field wears no name of its own")
}

func test_answers(t *testing.T) {
	subject, file := fixture(t)
	symbol := member_of(t, subject, file, "Total")
	testify.Equal(t, types.SYMBOL_FUNCTION, types.Symbol_Kind_Of(subject, symbol),
		"a function reads back as a function")
	signature := types.Symbol_Type_Of(subject, symbol)
	testify.Equal(t, types.TYPE_FUNCTION, types.Type_Kind_Of(subject, signature),
		"a function names a signature")
	parameters := slot_of(types.Type_Key_Of(subject, signature))
	testify.Equal(t, types.TYPE_TUPLE, types.Type_Kind_Of(subject, parameters),
		"a signature holds its parameters in a tuple")
	first := member_slot_of(types.Type_Members_Of(subject, parameters))
	testify.Equal(t, types.Name("entries"), types.Symbol_Name_Of(subject,
		symbol_slot_of(types.Member_Symbol_Of(subject, first))), "a parameter names itself")
	second := member_slot_of(types.Member_Next_Of(subject, first))
	testify.Equal(t, types.TYPE_SLICE, types.Type_Kind_Of(subject,
		slot_of(types.Member_Type_Of(subject, second))),
		"a run of arguments reads as a slice")
	results := slot_of(types.Type_Element_Of(subject, signature))
	sum := member_slot_of(types.Type_Members_Of(subject, results))
	testify.Equal(t, types.Name("sum"), types.Symbol_Name_Of(subject,
		symbol_slot_of(types.Member_Symbol_Of(subject, sum))), "a result names itself")
	constraint := types.Underlying_Of(subject,
		types.Symbol_Type_Of(subject, member_of(t, subject, file, "Numeric")))
	testify.Equal(t, types.TYPE_CONSTRAINT, types.Type_Kind_Of(subject, constraint),
		"a constraint names a constraint")
	testify.Not_Equal(t, types.MEMBER_ABSENT,
		member_slot_of(types.Type_Members_Of(subject, constraint)),
		"a constraint states the terms it admits")
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
		testify.Equal(t, one.Kind, types.Type_Kind_Of(subject,
			slot_of(types.Type_Element_Of(subject, index))),
			"the pointer %q names the type it stands over", one.Name)
	}
	for _, one := range []struct {
		Name string
		Kind types.Type_Kind
	}{
		{"truth", types.TYPE_BOOLEAN},
		{"words", types.TYPE_STRING},
	} {
		index := types.Symbol_Type_Of(subject, member_of(t, subject, file, one.Name))
		testify.Equal(t, one.Kind, types.Type_Kind_Of(subject,
			slot_of(types.Type_Key_Of(subject, index))),
			"the map %q names the type of its key", one.Name)
	}
	solo := member_of(t, subject, file, "Solo")
	testify.Equal(t, types.File_Index(2), types.Symbol_File_Of(subject, solo),
		"a symbol names the file that states it")
	for _, name := range []string{"A", "AB"} {
		symbol := member_of(t, subject, file, name)
		testify.Equal(t, types.Name(name), types.Symbol_Name_Of(subject, symbol),
			"the name %q reads back whole", name)
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
		{"package one\n\ntype One struct {\n\tFlag bool\n}\n\n" +
			"type Two struct {\n\tText string\n}\n", 2},
	} {
		subject, tree := module_of()
		file := analyse(t, subject, tree,
			[]fixture_file{{Path: TEST_PATH, Source: one.Source}})[0]
		name := "Pair"
		if one.Head != 1 {
			name = "Two"
		}
		index := types.Underlying_Of(subject,
			types.Symbol_Type_Of(subject, member_of(t, subject, file, name)))
		testify.Equal(t, one.Head, member_slot_of(types.Type_Members_Of(subject, index)),
			"a struct names the first member of its chain")
		if one.Head == 1 {
			testify.Equal(t, types.Member_Index(2),
				member_slot_of(types.Member_Next_Of(subject, one.Head)),
				"a member names the member the arena took after it")
		}
		testify.Equal(t, types.TYPE_BOOLEAN, types.Type_Kind_Of(subject, slot_of(
			types.Member_Type_Of(subject,
				1))), "the first member wears the type it states")
		testify.Equal(t, types.TYPE_STRING, types.Type_Kind_Of(subject, slot_of(
			types.Member_Type_Of(subject, 2))), "the second member wears its own type")
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
	testify.Equal(t, types.Name("two"), types.Symbol_Name_Of(subject, binding),
		"an import wears the name the file states")
	testify.Equal(t, types.Path(TEST_OTHER_PATH),
		types.Path_Of(subject, types.Symbol_Target_Of(subject, binding)),
		"an import names the package its path states")
	testify.Not_Equal(t, types.Package_Of(subject, file),
		types.Symbol_Target_Of(subject, binding), "an import names another package")
}

// Reads every answer at the first slots and the final slot of each arena, because a caller may
// hold any slot and an answer that refuses one is an answer a caller cannot use.
func answers_sweep(t *testing.T, subject *types.Module) {
	t.Helper()
	for _, index := range []types.Type_Index{0, 1, 2, types.TYPE_INDEX_MAXIMUM} {
		testify.True(t,
			slot_of(types.Type_Element_Of(subject, index)) <= types.TYPE_INDEX_MAXIMUM,
			"a type slot names an element inside the arena")
		testify.True(t,
			slot_of(types.Type_Key_Of(subject, index)) <= types.TYPE_INDEX_MAXIMUM,
			"a type slot names a key inside the arena")
		symbol := symbol_slot_of(types.Type_Symbol_Of(subject, index))
		testify.True(t, symbol <= types.SYMBOL_INDEX_MAXIMUM,
			"a type slot names a symbol inside the arena")
		member := member_slot_of(types.Type_Members_Of(subject, index))
		testify.True(t, member <= types.MEMBER_INDEX_MAXIMUM,
			"a type slot names a member inside the arena")
		testify.True(t, types.Underlying_Of(subject, index) <= types.TYPE_INDEX_MAXIMUM,
			"a type slot walks to a type inside the arena")
		testify.True(t,
			types.Type_Element_Count_Of(subject, index) >= types.ELEMENT_COUNT_UNKNOWN,
			"a type slot names a element_count it holds")
		testify.True(t,
			uint8(types.Type_Kind_Of(subject, index)) <= uint8(types.TYPE_TUPLE),
			"a type slot names a kind the module states")
	}
	for _, index := range []types.Symbol_Index{0, 1, 2, types.SYMBOL_INDEX_MAXIMUM} {
		testify.True(t, types.Symbol_Type_Of(subject, index) <= types.TYPE_INDEX_MAXIMUM,
			"a symbol slot names a type inside the arena")
		testify.True(t, types.Symbol_File_Of(subject, index) <= types.FILE_INDEX_MAXIMUM,
			"a symbol slot names a file inside the arena")
		testify.True(t,
			types.Symbol_Target_Of(subject, index) <= types.PACKAGE_INDEX_MAXIMUM,
			"a symbol slot names a package inside the arena")
		testify.True(t,
			len(types.Symbol_Name_Of(subject, index)) <= types.NAME_SIZE_MAXIMUM,
			"a symbol slot wears a name inside the width")
		testify.True(t, uint8(types.Symbol_Kind_Of(subject, index)) <=
			uint8(types.SYMBOL_TYPE_PARAMETER), "a symbol slot names a kind")
	}
	for _, index := range []types.Member_Index{0, 1, 2, types.MEMBER_INDEX_MAXIMUM} {
		testify.True(t,
			slot_of(types.Member_Type_Of(subject, index)) <= types.TYPE_INDEX_MAXIMUM,
			"a member slot names a type inside the arena")
		symbol := symbol_slot_of(types.Member_Symbol_Of(subject, index))
		testify.True(t, symbol <= types.SYMBOL_INDEX_MAXIMUM,
			"a member slot names a symbol inside the arena")
		next := member_slot_of(types.Member_Next_Of(subject, index))
		testify.True(t, next <= types.MEMBER_INDEX_MAXIMUM,
			"a member slot names a member inside the arena")
	}
	for _, index := range []types.File_Index{0, 1, 2, types.FILE_INDEX_MAXIMUM} {
		testify.True(t, types.Package_Of(subject, index) <= types.PACKAGE_INDEX_MAXIMUM,
			"a file slot names a package inside the arena")
		testify.True(t, len(types.Source_Of(subject, index)) <= token.SOURCE_SIZE_MAXIMUM,
			"a file slot names a source inside the width")
	}
	for _, index := range []types.Package_Index{0, 1, 2, types.PACKAGE_INDEX_MAXIMUM} {
		testify.True(t, len(types.Path_Of(subject, index)) <= types.PATH_SIZE_MAXIMUM,
			"a package slot answers to a path inside the width")
		_, found := types.Lookup(subject, index, types.Name("Count"))
		testify.True(t, bool(found) || !bool(found), "a package slot answers a search")
	}
}

func answers_messages(t *testing.T) {
	t.Helper()
	for _, code := range []types.Failure_Code{
		types.FAILURE_NONE, types.FAILURE_FILE_COUNT, types.FAILURE_PACKAGE_COUNT,
		types.FAILURE_SYMBOL_COUNT, types.FAILURE_TYPE_COUNT, types.FAILURE_MEMBER_COUNT,
		types.FAILURE_NAME_SIZE, types.FAILURE_PATH_SIZE, types.FAILURE_NESTING_DEPTH,
		types.FAILURE_UNKNOWN_NAME, types.FAILURE_DUPLICATE_NAME,
		types.FAILURE_UNKNOWN_IMPORT, types.FAILURE_TYPE_CYCLE,
	} {
		text := types.Failure_Message(code)
		testify.True(t, len(text) <= types.MESSAGE_SIZE_MAXIMUM,
			"the code %d reads as a sentence inside the width", code)
		if code == types.FAILURE_NONE {
			testify.Equal(t, types.Message(""), text,
				"a clean module reads no sentence")
			continue
		}
		testify.Not_Equal(t, types.Message(""), text, "a refusal reads a sentence")
	}
}

func test_refusals(t *testing.T) {
	for _, one := range []struct {
		Source string
		Code   types.Failure_Code
	}{
		{"package one\n\nvar value Missing\n", types.FAILURE_UNKNOWN_NAME},
		{"package one\n\ntype Count int\n\ntype Count int\n", types.FAILURE_DUPLICATE_NAME},
		{"package one\n\ntype Loop Loop\n", types.FAILURE_TYPE_CYCLE},
		{"package one\n\nimport far \"local/example/far\"\n\nvar value far.Marker\n",
			types.FAILURE_UNKNOWN_IMPORT},
		{"package one\n\nvar value other.Marker\n", types.FAILURE_UNKNOWN_NAME},
	} {
		subject, tree := module_of()
		file, _ := types.Add_File(subject, TEST_PATH, token.Source(one.Source))
		ast.Parse(tree, token.Source(one.Source))
		declared := types.Declare(subject, file, tree)
		ast.Parse(tree, token.Source(one.Source))
		resolved := types.Resolve(subject, file, tree)
		testify.False(t, bool(declared) && bool(resolved),
			"a refused source fails one pass")
		testify.Equal(t, one.Code, types.Failure(subject), "the refusal names its code")
		testify.Not_Equal(t, types.Message(""), types.Failure_Message(one.Code),
			"a code reads as a sentence")
	}
	subject, tree := module_of()
	broken := token.Source("package one\n\nvar value int\n\nfunc (\n")
	file, _ := types.Add_File(subject, TEST_PATH, broken)
	ast.Parse(tree, broken)
	testify.True(t, bool(types.Declare(subject, file, tree)),
		"a refused parse still declares the names its tree holds")
	testify.Equal(t, types.FAILURE_NONE, types.Failure(subject),
		"a refused parse is the business of the parser and never of the module")
}

func test_bounds(t *testing.T) {
	bounds_files(t)
	bounds_packages(t)
	bounds_widths(t)
	bounds_depth(t)
	bounds_symbols(t)
	bounds_types(t)
	bounds_members(t)
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

// Builds a struct that states one field for each slot a fill needs.
func filled_structure(count int) (source string) {
	buffer := make([]byte, 0, count*16+64)
	buffer = append(buffer, "package fill\n\ntype Wide struct {\n"...)
	for index := range count {
		buffer = append(buffer, fmt.Sprintf("\tField%d int\n", index)...)
	}
	return string(append(buffer, "}\n"...))
}

func bounds_files(t *testing.T) {
	subject, tree := module_of()
	filled := 0
	for filled < types.FILE_COUNT_MAXIMUM-1 {
		_, ok := types.Add_File(subject, TEST_PATH, token.Source("package one\n"))
		testify.True(t, bool(ok), "a file inside the bound binds")
		filled++
	}
	last, ok := types.Add_File(subject, TEST_PATH, token.Source(TEST_OTHER_SOURCE))
	testify.True(t, bool(ok), "the final file binds")
	testify.Equal(t, types.File_Index(types.FILE_COUNT_MAXIMUM-1), last,
		"the final file takes the final slot")
	ast.Parse(tree, token.Source(TEST_OTHER_SOURCE))
	testify.True(t, bool(types.Declare(subject, last, tree)), "the final file declares")
	ast.Parse(tree, token.Source(TEST_OTHER_SOURCE))
	testify.True(t, bool(types.Resolve(subject, last, tree)), "the final file resolves")
	marker, found := types.Lookup(
		subject, types.Package_Of(subject, last), types.Name("Marker"))
	testify.True(t, bool(found), "the final file states its names")
	testify.Equal(t, last, types.Symbol_File_Of(subject, marker),
		"a symbol names the file that states it")
	_, over := types.Add_File(subject, TEST_PATH, token.Source("package one\n"))
	testify.False(t, bool(over), "a file past the bound is refused")
	testify.Equal(t, types.FAILURE_FILE_COUNT, types.Failure(subject),
		"a file past the bound names its code")
}

func bounds_packages(t *testing.T) {
	subject, _ := module_of()
	for index := range types.PACKAGE_COUNT_MAXIMUM - 1 {
		path := types.Path(fmt.Sprintf("local/example/%d", index))
		file, ok := types.Add_File(subject, path, token.Source("package one\n"))
		testify.True(t, bool(ok), "a package inside the bound binds")
		testify.Equal(t, types.Package_Index(index+1), types.Package_Of(subject, file),
			"a path takes the package slot after the last one")
	}
	last := types.Package_Index(types.PACKAGE_COUNT_MAXIMUM - 1)
	testify.Equal(t, types.Path("local/example/254"), types.Path_Of(subject, last),
		"a package reads back the path it answers to")
	_, absent := types.Lookup(subject, last, types.Name("Marker"))
	testify.False(t, bool(absent), "a package with no file states no name")
	bounds_final_package(t, subject)
	_, over := types.Add_File(subject, "local/example/over", token.Source("package one\n"))
	testify.False(t, bool(over), "a package past the bound is refused")
	testify.Equal(t, types.FAILURE_PACKAGE_COUNT, types.Failure(subject),
		"a package past the bound names its code")
}

// Declares one file of the final package, which is where a caller reads the widest package slot
// a module holds.
func bounds_final_package(t *testing.T, subject *types.Module) {
	t.Helper()
	tree := new(ast.Parse_State)
	source := "package last\n\nimport near \"local/example/1\"\n\n" +
		"import far \"local/example/254\"\n\ntype Held int\n"
	file, ok := types.Add_File(subject, "local/example/254", token.Source(source))
	testify.True(t, bool(ok), "the file of the final package binds")
	testify.Equal(t, types.Package_Index(types.PACKAGE_INDEX_MAXIMUM),
		types.Package_Of(subject, file), "the final path names the final package")
	ast.Parse(tree, token.Source(source))
	testify.True(t, bool(types.Declare(subject, file, tree)), "the final package declares")
	ast.Parse(tree, token.Source(source))
	testify.True(t, bool(types.Resolve(subject, file, tree)), "the final package resolves")
	near, found := types.Lookup(subject, types.Package_Of(subject, file), types.Name("Held"))
	testify.True(t, bool(found), "the final package states its names")
	testify.Equal(t, types.File_Index(types.PACKAGE_COUNT_MAXIMUM-1),
		types.Symbol_File_Of(subject, near), "a symbol names the file that states it")
	for _, one := range []struct {
		Name   string
		Target types.Package_Index
	}{
		{"near", 2},
		{"far", types.PACKAGE_INDEX_MAXIMUM},
	} {
		binding := import_of(t, subject, file, one.Name)
		testify.Equal(t, one.Target, types.Symbol_Target_Of(subject, binding),
			"the import %q names the package its path states", one.Name)
	}
}

// Names one import binding of a file, which no package table holds because an import stands in
// one file alone.
func import_of(
	t *testing.T, subject *types.Module, file types.File_Index, name string,
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
		file, ok := types.Add_File(subject, path, token.Source(source))
		testify.True(t, bool(ok), "a path of %d bytes binds", len(path))
		testify.Equal(t, path, types.Path_Of(subject, types.Package_Of(subject, file)),
			"a path of %d bytes reads back", len(path))
	}
	for _, one := range []string{"", "a", "ab"} {
		file, ok := types.Add_File(subject, "local/example/short", token.Source(one))
		testify.True(t, bool(ok), "a source of %d bytes binds", len(one))
		testify.Equal(t, token.Source(one), types.Source_Of(subject, file),
			"a source of %d bytes reads back", len(one))
	}
	testify.Equal(t, token.Source(source), types.Source_Of(subject, 0),
		"the widest source reads back")
	for _, name := range []types.Name{types.Name(""), types.Name("a"), types.Name("ab"),
		types.Name(wide[:types.NAME_SIZE_MAXIMUM])} {
		_, absent := types.Lookup(subject, types.PACKAGE_UNIVERSE, name)
		testify.False(t, bool(absent), "the universe states no name of %d bytes", len(name))
	}
	long := fmt.Sprintf("package fill\n\nvar %s int\n", string(wide[:types.NAME_SIZE_MAXIMUM]))
	file, _ := types.Add_File(subject, "local/example/long", token.Source(long))
	ast.Parse(tree, token.Source(long))
	testify.True(t, bool(types.Declare(subject, file, tree)), "a name at the bound declares")
	symbol, found := types.Lookup(subject, types.Package_Of(subject, file),
		types.Name(wide[:types.NAME_SIZE_MAXIMUM]))
	testify.True(t, bool(found), "a name at the bound binds")
	testify.Equal(t, types.NAME_SIZE_MAXIMUM, len(types.Symbol_Name_Of(subject, symbol)),
		"a name at the bound reads back whole")
	bounds_name_size(t)
}

func bounds_name_size(t *testing.T) {
	subject, tree := module_of()
	wide := make([]byte, types.NAME_SIZE_MAXIMUM+1)
	for index := range wide {
		wide[index] = 'n'
	}
	source := fmt.Sprintf("package fill\n\nvar %s int\n", string(wide))
	file, _ := types.Add_File(subject, TEST_PATH, token.Source(source))
	ast.Parse(tree, token.Source(source))
	testify.False(t, bool(types.Declare(subject, file, tree)),
		"a name past the bound is refused")
	testify.Equal(t, types.FAILURE_NAME_SIZE, types.Failure(subject),
		"a name past the bound names its code")
}

func bounds_depth(t *testing.T) {
	subject, tree := module_of()
	stars := make([]byte, types.DEPTH_MAXIMUM+2)
	for index := range stars {
		stars[index] = '*'
	}
	source := fmt.Sprintf("package fill\n\nvar deep %sint\n", string(stars))
	file, _ := types.Add_File(subject, TEST_PATH, token.Source(source))
	ast.Parse(tree, token.Source(source))
	types.Declare(subject, file, tree)
	ast.Parse(tree, token.Source(source))
	testify.False(t, bool(types.Resolve(subject, file, tree)),
		"a type past the nesting bound is refused")
	testify.Equal(t, types.FAILURE_NESTING_DEPTH, types.Failure(subject),
		"a type past the nesting bound names its code")
}

// Feeds one tail file into a module whose head already stands, which is how a case reads the
// final slot of an arena.
func fill_arena(
	t *testing.T, head []string, tail string,
) (subject *types.Module, file types.File_Index) {
	t.Helper()
	subject, tree := module_of()
	heads := []fixture_file{}
	for _, one := range head {
		heads = append(heads, fixture_file{Path: TEST_PATH, Source: one})
	}
	analyse(t, subject, tree, heads)
	file, _ = types.Add_File(subject, TEST_PATH, token.Source(tail))
	ast.Parse(tree, token.Source(tail))
	types.Declare(subject, file, tree)
	ast.Parse(tree, token.Source(tail))
	types.Resolve(subject, file, tree)
	return subject, file
}

// Builds the head that leaves the symbol arena with the free slot count it names. One declaration
// states many names, thus the head spends two tokens for each symbol it takes.
func symbol_head(t *testing.T, free int) (head []string) {
	t.Helper()
	subject, _ := module_of()
	need := types.SYMBOL_COUNT_MAXIMUM - count_symbols(subject) - free
	spent := 0
	for spent < need {
		count := need - spent
		if count > TEST_NAME_RUN_MAXIMUM {
			count = TEST_NAME_RUN_MAXIMUM
		}
		head = append(head, named_run(spent, count))
		spent = spent + count
	}
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

// Builds the head that leaves the type arena with the free slot count it names. One declaration
// states two pointers, thus the head spends half the symbol slots the types it takes would.
func type_head(t *testing.T, free int) (head string) {
	t.Helper()
	subject, _ := module_of()
	need := types.TYPE_COUNT_MAXIMUM - count_types(subject) - free
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
	return string(buffer)
}

func bounds_symbols(t *testing.T) {
	last := types.Symbol_Index(types.SYMBOL_INDEX_MAXIMUM)
	subject, file := fill_arena(t, symbol_head(t, 2),
		"package fill\n\ntype Solo struct {\n\tOnly int\n}\n")
	testify.Equal(t, types.SYMBOL_FIELD, types.Symbol_Kind_Of(subject, last),
		"the final symbol slot holds the last name a file stated")
	solo := types.Underlying_Of(subject,
		types.Symbol_Type_Of(subject, member_of(t, subject, file, "Solo")))
	testify.Equal(t, last, symbol_slot_of(types.Member_Symbol_Of(subject,
		member_slot_of(types.Type_Members_Of(subject, solo)))),
		"the final field wears the final symbol slot")
	bounds_past_symbol(t)
	bounds_final_symbol(t)
}

func bounds_past_symbol(t *testing.T) {
	subject, file := fill_arena(t, symbol_head(t, 0), "package fill\n\nvar past int\n")
	testify.Equal(t, types.FAILURE_SYMBOL_COUNT, types.Failure(subject),
		"a name past the bound names its code")
	_, absent := types.Lookup(subject, types.Package_Of(subject, file), types.Name("past"))
	testify.False(t, bool(absent), "a name past the bound binds nothing")
}

func bounds_final_symbol(t *testing.T) {
	subject, file := fill_arena(t, symbol_head(t, 1), "package fill\n\ntype Last int\n")
	found, ok := types.Lookup(subject, types.Package_Of(subject, file), types.Name("Last"))
	testify.True(t, bool(ok), "the final name binds")
	testify.Equal(t, types.Symbol_Index(types.SYMBOL_INDEX_MAXIMUM), found,
		"the final name takes the final symbol slot")
	testify.Equal(t, found, symbol_slot_of(types.Type_Symbol_Of(subject,
		types.Symbol_Type_Of(subject, found))),
		"the final named type carries the final symbol slot")
	testify.Equal(t, types.FAILURE_NONE, types.Failure(subject),
		"a module that fills its symbols exactly refuses nothing")
}

func bounds_types(t *testing.T) {
	last := types.Type_Index(types.TYPE_INDEX_MAXIMUM)
	subject, file := fill_arena(t, []string{type_head(t, 2)},
		"package fill\n\nvar pair **int\n\nvar over *int\n")
	testify.Equal(t, types.TYPE_POINTER, types.Type_Kind_Of(subject, last),
		"the final type slot holds the last type a file stated")
	testify.Equal(t, last, slot_of(types.Type_Element_Of(subject, last-1)),
		"a pointer over a pointer names the slot behind it")
	testify.Equal(t, types.FAILURE_TYPE_COUNT, types.Failure(subject),
		"a type past the bound names its code")
	pair, found := types.Lookup(subject, types.Package_Of(subject, file), types.Name("pair"))
	testify.True(t, bool(found), "the final declaration binds its name")
	testify.Equal(t, last-1, types.Symbol_Type_Of(subject, pair),
		"the final declaration wears the type it stated")
	bounds_final_type(t)
	bounds_final_field(t)
	bounds_final_tuple(t)
}

func bounds_final_type(t *testing.T) {
	subject, file := fill_arena(t, []string{type_head(t, 1)}, "package fill\n\nvar solo *int\n")
	testify.Equal(t, types.Type_Index(types.TYPE_INDEX_MAXIMUM),
		types.Symbol_Type_Of(subject, member_of(t, subject, file, "solo")),
		"the final declaration wears the final type slot")
}

func bounds_final_field(t *testing.T) {
	subject, file := fill_arena(t, []string{type_head(t, 3)},
		"package fill\n\ntype Holder struct {\n\tValue *int\n}\n")
	holder := types.Underlying_Of(subject,
		types.Symbol_Type_Of(subject, member_of(t, subject, file, "Holder")))
	testify.Equal(t, types.Type_Index(types.TYPE_INDEX_MAXIMUM),
		slot_of(types.Member_Type_Of(subject,
			member_slot_of(types.Type_Members_Of(subject, holder)))),
		"the final field wears the final type slot")
}

func bounds_final_tuple(t *testing.T) {
	subject, file := fill_arena(t, []string{type_head(t, 2)},
		"package fill\n\nfunc Only(one int) {\n}\n")
	signature := types.Symbol_Type_Of(subject, member_of(t, subject, file, "Only"))
	testify.Equal(t, types.Type_Index(types.TYPE_INDEX_MAXIMUM),
		slot_of(types.Type_Key_Of(subject, signature)),
		"the final signature holds its parameters in the final type slot")
}

func bounds_members(t *testing.T) {
	subject, tree := module_of()
	head := filled_structure(types.MEMBER_COUNT_MAXIMUM - 3)
	analyse(t, subject, tree, []fixture_file{{Path: TEST_PATH, Source: head}})
	tail := "package fill\n\ntype Tail struct {\n\tOne int\n\tTwo int\n\tThree int\n}\n"
	file, _ := types.Add_File(subject, TEST_PATH, token.Source(tail))
	ast.Parse(tree, token.Source(tail))
	types.Declare(subject, file, tree)
	ast.Parse(tree, token.Source(tail))
	types.Resolve(subject, file, tree)
	last := types.Member_Index(types.MEMBER_INDEX_MAXIMUM)
	testify.Equal(t, types.FAILURE_MEMBER_COUNT, types.Failure(subject),
		"a member past the bound names its code")
	testify.Equal(t, last, member_slot_of(types.Member_Next_Of(subject, last-1)),
		"a member chain reaches the final slot")
	testify.Equal(t, types.MEMBER_ABSENT, member_slot_of(types.Member_Next_Of(subject, last)),
		"the final member closes its chain")
	testify.Not_Equal(t, types.SYMBOL_ABSENT,
		symbol_slot_of(types.Member_Symbol_Of(subject, last)),
		"the final member wears the name its field states")
	tail_type := types.Underlying_Of(subject, types.Symbol_Type_Of(subject,
		member_of(t, subject, file, "Tail")))
	testify.Equal(t, last-1, member_slot_of(types.Type_Members_Of(subject, tail_type)),
		"a struct names the first member of its chain")
	bounds_final_head(t)
	testify.Equal(t, types.TYPE_INT, types.Type_Kind_Of(subject,
		slot_of(types.Member_Type_Of(subject, last))),
		"the final member wears the type it states")
	bounds_lengths(t)
}

func bounds_final_head(t *testing.T) {
	subject, tree := module_of()
	head := filled_structure(types.MEMBER_COUNT_MAXIMUM - 2)
	analyse(t, subject, tree, []fixture_file{{Path: TEST_PATH, Source: head}})
	tail := "package fill\n\ntype Tail struct {\n\tOnly int\n}\n"
	file, _ := types.Add_File(subject, TEST_PATH, token.Source(tail))
	ast.Parse(tree, token.Source(tail))
	types.Declare(subject, file, tree)
	ast.Parse(tree, token.Source(tail))
	types.Resolve(subject, file, tree)
	index := types.Underlying_Of(subject,
		types.Symbol_Type_Of(subject, member_of(t, subject, file, "Tail")))
	testify.Equal(t, types.Member_Index(types.MEMBER_INDEX_MAXIMUM),
		member_slot_of(types.Type_Members_Of(subject, index)),
		"the final struct names the final member slot")
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
		testify.Equal(t, one.Element_Count, types.Type_Element_Count_Of(subject, index),
			"the array %q names how many elements it holds", one.Name)
	}
}

func test_allocation(t *testing.T) {
	held := allocation_fixture{}
	subject, file := fixture_module(t)
	symbol := member_of(t, subject, file, "Count")
	path := types.Path(TEST_OTHER_PATH)
	source := token.Source("package two\n")
	sought := types.Name("Count")
	passes, tree := module_of()
	body := new(types.Body)
	fresh, bound := types.Add_File(passes, TEST_PATH, token.Source(TEST_SOURCE))
	testify.True(t, bool(bound), "the file the passes read binds")
	ast.Parse(tree, token.Source(TEST_SOURCE))
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Add_File", Call: func() {
			held.File, held.Ok = types.Add_File(subject, path, source)
		}},
		{Name: "Lookup", Call: func() {
			held.Symbol, held.Ok = types.Lookup(
				subject, types.Package_Of(subject, file), sought)
		}},
		{Name: "Underlying_Of", Call: func() {
			held.Type = types.Underlying_Of(
				subject, types.Symbol_Type_Of(subject, symbol))
		}},
		{Name: "Type_Members_Of", Call: func() {
			held.Type = types.Type_Index(
				member_slot_of(types.Type_Members_Of(subject, held.Type)))
		}},
		{Name: "Declare", Call: func() {
			held.Ok = types.Declare(passes, fresh, tree)
		}},
		{Name: "Resolve", Call: func() {
			held.Ok = types.Resolve(passes, fresh, tree)
		}},
		{Name: "Check", Call: func() {
			held.Ok = types.Check(passes, body, fresh, tree)
		}},
	}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	testify.Equal(t, types.FAILURE_DUPLICATE_NAME, types.Failure(passes),
		"a file the declare pass reads twice states its names twice")
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
	body = new(types.Body)
	files := []fixture_file{
		{Path: TEST_OTHER_PATH, Source: TEST_OTHER_SOURCE},
		{Path: TEST_PATH, Source: TEST_BODY_SOURCE},
	}
	indexes := analyse(t, subject, tree, files)
	for slot, one := range files {
		ast.Parse(tree, token.Source(one.Source))
		testify.True(t, bool(types.Check(subject, body, indexes[slot], tree)),
			"the file checks: %s", types.Failure_Message(types.Failure(subject)))
	}
	ast.Parse(tree, token.Source(TEST_BODY_SOURCE))
	types.Check(subject, body, indexes[1], tree)
	return subject, body, tree
}

// Names the tree slot of one node the fixture states, which is how a case points at the
// expression it means.
func node_of(
	t *testing.T, tree *ast.Parse_State, kind ast.Node_Kind, text string, place int,
) (node ast.Index) {
	t.Helper()
	found := ast.Index(0)
	seen := 0
	walk_nodes(tree, 1, func(index ast.Index) {
		one := ast.Node_At(tree, index)
		if one.Kind != kind {
			return
		}
		if string(token.Text(token.Source(TEST_BODY_SOURCE),
			ast.Token_At(tree, one.Token))) != text {
			return
		}
		if seen == place {
			found = index
		}
		seen = seen + 1
	})
	testify.Not_Equal(t, ast.Index(0), found, "the fixture states %q", text)
	return found
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
	t *testing.T, subject *types.Module, body *types.Body, tree *ast.Parse_State,
	kind ast.Node_Kind, text string, place int,
) (folded types.Type_Kind) {
	t.Helper()
	node := node_of(t, tree, kind, text, place)
	return types.Type_Kind_Of(subject, types.Type_At(body, node))
}

func test_bodies(t *testing.T) {
	subject, body, tree := body_fixture(t)
	testify.Equal(t, types.FAILURE_NONE, types.Failure(subject),
		"a body the module reads holds no fault")
	testify.Equal(t, types.TYPE_UNTYPED_INTEGER,
		folded_kind(t, subject, body, tree, ast.NODE_INTEGER, "0", 0),
		"a literal folds to the type its own spelling states")
	testify.Equal(t, types.TYPE_INVALID, types.Type_Kind_Of(subject,
		types.Type_At(body, ast.Index(ast.NODE_COUNT_MAXIMUM-1))),
		"a slot the pass folded nothing into names the invalid type")
	testify.Equal(t, types.TYPE_INVALID, types.Type_Kind_Of(subject, types.Type_At(body, 0)),
		"the absent slot names the invalid type")
	testify.Equal(t, types.TYPE_INVALID, types.Type_Kind_Of(subject, types.Type_At(body, 1)),
		"the file slot names no type of its own")
	for _, one := range []struct {
		Text string
		Kind types.Type_Kind
	}{
		{"truth", types.TYPE_BOOLEAN},
		{"letters", types.TYPE_STRING},
	} {
		testify.Equal(t, one.Kind, use_kind(t, subject, body, tree, one.Text),
			"the name %q wears the type its declaration states", one.Text)
	}
	bodies_files(t)
	bodies_final_type(t)
}

// Checks the final file of a full module, and one file a refused module holds.
func bodies_files(t *testing.T) {
	t.Helper()
	subject, tree := module_of()
	body := new(types.Body)
	held := types.File_Index(0)
	for index := range types.FILE_COUNT_MAXIMUM {
		source := token.Source(fmt.Sprintf(
			"package one\n\nfunc Only%d() (one int) {\n\treturn 1\n}\n", index))
		file, ok := types.Add_File(subject, TEST_PATH, source)
		testify.True(t, bool(ok), "a file of the fill binds")
		if index != 2 {
			if index != types.FILE_COUNT_MAXIMUM-1 {
				continue
			}
		}
		held = file
		ast.Parse(tree, source)
		types.Declare(subject, file, tree)
		ast.Parse(tree, source)
		testify.True(t, bool(types.Check(subject, body, file, tree)),
			"the file at slot %d checks", file)
	}
	_, over := types.Add_File(subject, TEST_PATH, token.Source("package one\n"))
	testify.False(t, bool(over), "a file past the bound is refused")
	ast.Parse(tree, token.Source("package one\n"))
	testify.False(t, bool(types.Check(subject, body, held, tree)),
		"a module that already refused something reports its bodies refused")
}

// Checks a body whose pointer takes the final type slot, which is what a caller reads back.
func bodies_final_type(t *testing.T) {
	t.Helper()
	tail := "package fill\n\nfunc Only(one int) (two int) {\n\tpointer := &one\n\t" +
		"return *pointer\n}\n"
	subject, file := fill_arena(t, []string{type_head(t, 4)}, tail)
	body := new(types.Body)
	tree := new(ast.Parse_State)
	ast.Parse(tree, token.Source(tail))
	types.Check(subject, body, file, tree)
	found := types.Type_Index(0)
	walk_nodes(tree, 1, func(index ast.Index) {
		if ast.Node_At(tree, index).Kind != ast.NODE_UNARY {
			return
		}
		if found != 0 {
			return
		}
		found = types.Type_At(body, index)
	})
	testify.Equal(t, types.Type_Index(types.TYPE_INDEX_MAXIMUM), found,
		"the address of a value takes the final type slot")
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
		testify.Equal(t, one.Kind, use_kind(t, subject, body, tree, one.Text),
			"the name %q wears the type its statement bound", one.Text)
	}
}

// Reads the kind of the type one name folded to where a body reads it, which is the first slot
// the pass wrote a type into for that name.
func use_kind(
	t *testing.T, subject *types.Module, body *types.Body, tree *ast.Parse_State, text string,
) (folded types.Type_Kind) {
	t.Helper()
	found := types.TYPE_INVALID
	seen := false
	walk_nodes(tree, 1, func(index ast.Index) {
		if bool(seen) {
			return
		}
		one := ast.Node_At(tree, index)
		if one.Kind != ast.NODE_IDENTIFIER {
			return
		}
		if string(token.Text(token.Source(TEST_BODY_SOURCE),
			ast.Token_At(tree, one.Token))) != text {
			return
		}
		if types.Type_At(body, index) == 0 {
			return
		}
		found = types.Type_Kind_Of(subject, types.Type_At(body, index))
		seen = true
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
		testify.Equal(t, one.Folds,
			folded_kind(t, subject, body, tree, one.Kind, one.Text, one.Place),
			"the form %q folds to the type Go states", one.Text)
	}
	expression_calls(t, subject, body, tree)
	expression_selectors(t, subject, body, tree)
}

func expression_calls(
	t *testing.T, subject *types.Module, body *types.Body, tree *ast.Parse_State,
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
		testify.Equal(t, one.Folds,
			folded_kind(t, subject, body, tree, ast.NODE_CALL, "(", one.Place),
			"the call at place %d folds to the type it states", one.Place)
	}
}

func expression_selectors(
	t *testing.T, subject *types.Module, body *types.Body, tree *ast.Parse_State,
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
		testify.Equal(t, one.Folds,
			folded_kind(t, subject, body, tree, ast.NODE_SELECTOR, ".", one.Place),
			"the selector at place %d folds to the type it reads", one.Place)
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
	b *testing.B, subject *types.Module, tree *ast.Parse_State, body *types.Body,
	source token.Source,
) {
	b.Helper()
	types.Reset(subject)
	if _, ok := ast.Parse(tree, source); !bool(ok) {
		b.Fatal("the benchmark source parses")
	}
	file, _ := types.Add_File(subject, TEST_PATH, source)
	types.Declare(subject, file, tree)
	types.Resolve(subject, file, tree)
	types.Check(subject, body, file, tree)
	if types.Failure(subject) != types.FAILURE_NONE {
		b.Fatal(string(types.Failure_Message(types.Failure(subject))))
	}
	if _, found := types.Lookup(subject, types.Package_Of(subject, file),
		types.Name("Fold0")); !bool(found) {
		b.Fatal("the benchmark source states its names")
	}
}

func Benchmark_Analyse_House(b *testing.B) {
	text := benchmark_source()
	source := token.Source(text)
	subject := new(types.Module)
	tree := new(ast.Parse_State)
	body := new(types.Body)
	benchmark_parity(b, subject, tree, body, source)
	b.SetBytes(int64(len(source)))
	b.ReportAllocs()
	for b.Loop() {
		types.Reset(subject)
		ast.Parse(tree, source)
		file, _ := types.Add_File(subject, TEST_PATH, source)
		types.Declare(subject, file, tree)
		types.Resolve(subject, file, tree)
		types.Check(subject, body, file, tree)
	}
}

func Benchmark_Analyse_Standard(b *testing.B) {
	text := benchmark_source()
	b.SetBytes(int64(len(text)))
	b.ReportAllocs()
	for b.Loop() {
		set := standard_token.NewFileSet()
		parsed, err := parser.ParseFile(set, "bench.go", text, 0)
		if err != nil {
			b.Fatal(err)
		}
		info := &standard_types.Info{
			Types: map[standard_ast.Expr]standard_types.TypeAndValue{},
			Defs:  map[*standard_ast.Ident]standard_types.Object{},
		}
		configuration := standard_types.Config{}
		_, err = configuration.Check("bench", set, []*standard_ast.File{parsed}, info)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func Benchmark_Parse_House(b *testing.B) {
	source := token.Source(benchmark_source())
	tree := new(ast.Parse_State)
	if _, ok := ast.Parse(tree, source); !bool(ok) {
		b.Fatal(string(ast.Failure_Message(ast.Failure(tree))))
	}
	b.SetBytes(int64(len(source)))
	b.ReportAllocs()
	for b.Loop() {
		ast.Parse(tree, source)
	}
}

func Benchmark_Parse_Standard(b *testing.B) {
	text := benchmark_source()
	b.SetBytes(int64(len(text)))
	b.ReportAllocs()
	for b.Loop() {
		set := standard_token.NewFileSet()
		_, err := parser.ParseFile(set, "bench.go", text, 0)
		if err != nil {
			b.Fatal(err)
		}
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

func Benchmark_Scan_Standard(b *testing.B) {
	text := benchmark_source()
	standard_set := standard_token.NewFileSet()
	standard_file := standard_set.AddFile("bench.go", standard_set.Base(), len(text))
	standard_scan := scanner.Scanner{}
	standard_scan.Init(standard_file, []byte(text), nil, 0)
	standard_count := 0
	for _, one, _ := standard_scan.Scan(); one != standard_token.EOF; {
		_, one, _ = standard_scan.Scan()
		standard_count = standard_count + 1
	}
	if standard_count < len(text)/BENCHMARK_TOKEN_DENSITY {
		b.Fatal("the standard scan reads the whole source")
	}
	b.SetBytes(int64(len(text)))
	b.ReportAllocs()
	for b.Loop() {
		set := standard_token.NewFileSet()
		file := set.AddFile("bench.go", set.Base(), len(text))
		scanner := scanner.Scanner{}
		scanner.Init(file, []byte(text), nil, 0)
		_, one, _ := scanner.Scan()
		for one != standard_token.EOF {
			_, one, _ = scanner.Scan()
		}
	}
}
