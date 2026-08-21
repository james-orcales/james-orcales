package ast_test

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/testify"
)

// Test_Parse binds the Parse specification leaf before fixture declarations.
func Test_Parse(t *testing.T) {
	test_parse(t)
}

// Test_Nodes binds the Nodes specification leaf before fixture declarations.
func Test_Nodes(t *testing.T) {
	test_nodes(t)
}

// Test_Package_Clause binds the Package Clause leaf before fixture declarations.
func Test_Package_Clause(t *testing.T) {
	test_package_clause(t)
}

// Test_Imports binds the Imports specification leaf before fixture declarations.
func Test_Imports(t *testing.T) {
	test_imports(t)
}

// Test_Constants_And_Variables binds the Constants And Variables leaf before fixtures.
func Test_Constants_And_Variables(t *testing.T) {
	test_constants_and_variables(t)
}

// Test_Types binds the Types specification leaf before fixture declarations.
func Test_Types(t *testing.T) {
	test_types(t)
}

// Test_Functions binds the Functions specification leaf before fixture declarations.
func Test_Functions(t *testing.T) {
	test_functions(t)
}

// Test_Type_Parameters binds the Type Parameters leaf before fixture declarations.
func Test_Type_Parameters(t *testing.T) {
	test_type_parameters(t)
}

// Test_Statements binds the Statements specification leaf before fixture declarations.
func Test_Statements(t *testing.T) {
	test_statements(t)
}

// Test_Expressions binds the Expressions specification leaf before fixture declarations.
func Test_Expressions(t *testing.T) {
	test_expressions(t)
}

// Test_Trivia_Fidelity binds the Fidelity specification leaf before fixture declarations.
func Test_Trivia_Fidelity(t *testing.T) {
	test_comments_fidelity(t)
}

// Test_Trivia_Order binds the Order specification leaf before fixture declarations.
func Test_Trivia_Order(t *testing.T) {
	test_comments_order(t)
}

// Test_Trivia_Attachment binds the Attachment leaf before fixture declarations.
func Test_Trivia_Attachment(t *testing.T) {
	test_comments_attachment(t)
}

// Test_Trivia_Blank_Lines binds the Blank Lines leaf before fixture declarations.
func Test_Trivia_Blank_Lines(t *testing.T) {
	test_blank_lines(t)
}

// Test_Refused_Syntax_Declaration_Groups binds the Declaration Groups leaf before
// fixture declarations.
func Test_Refused_Syntax_Declaration_Groups(t *testing.T) {
	test_declaration_groups(t)
}

// Test_Refused_Syntax_Interface_Methods binds the Interface Methods leaf before
// fixture declarations.
func Test_Refused_Syntax_Interface_Methods(t *testing.T) {
	test_interface_methods(t)
}

// Test_Refused_Syntax_Wide_Identifiers binds the Wide Identifiers leaf before
// fixture declarations.
func Test_Refused_Syntax_Wide_Identifiers(t *testing.T) {
	test_wide_identifiers(t)
}

// Test_Refused_Syntax_Iota binds the Iota specification leaf before fixture declarations.
func Test_Refused_Syntax_Iota(t *testing.T) {
	test_iota(t)
}

// Test_Refused_Syntax_Private_Fields binds the Private Fields leaf before
// fixture declarations.
func Test_Refused_Syntax_Private_Fields(t *testing.T) {
	test_private_fields(t)
}

// Test_Refused_Syntax_Compound_Predicates binds the Compound Predicates leaf before
// fixture declarations.
func Test_Refused_Syntax_Compound_Predicates(t *testing.T) {
	test_compound_predicates(t)
}

// Test_Refused_Syntax_Dot_Imports binds the Dot Imports leaf before fixture declarations.
func Test_Refused_Syntax_Dot_Imports(t *testing.T) {
	test_dot_imports(t)
}

// Test_Refused_Syntax_Blank_Imports binds the Blank Imports leaf before fixture declarations.
func Test_Refused_Syntax_Blank_Imports(t *testing.T) {
	test_blank_imports(t)
}

// Test_Refused_Syntax_Constant_Case binds the Constant Case leaf before fixture declarations.
func Test_Refused_Syntax_Constant_Case(t *testing.T) {
	test_constant_case(t)
}

// Test_Refused_Syntax_Bare_Loops binds the Bare Loops leaf before fixture declarations.
func Test_Refused_Syntax_Bare_Loops(t *testing.T) {
	test_bare_loops(t)
}

// Test_Refused_Syntax_Unnamed_Results binds the Unnamed Results leaf before
// fixture declarations.
func Test_Refused_Syntax_Unnamed_Results(t *testing.T) {
	test_unnamed_results(t)
}

// Test_Refused_Syntax_Type_Aliases binds the Type Aliases leaf before fixture declarations.
func Test_Refused_Syntax_Type_Aliases(t *testing.T) {
	test_type_aliases(t)
}

// Test_Refused_Syntax_Missing_Package_Clause binds the Missing Package Clause leaf
// before fixture declarations.
func Test_Refused_Syntax_Missing_Package_Clause(t *testing.T) {
	test_missing_package_clause(t)
}

// Test_Refused_Syntax_Token_Count binds the Token Count leaf before fixture declarations.
func Test_Refused_Syntax_Token_Count(t *testing.T) {
	test_token_count(t)
}

// Test_Refused_Syntax_Node_Count binds the Node Count leaf before fixture declarations.
func Test_Refused_Syntax_Node_Count(t *testing.T) {
	test_node_count(t)
}

// Test_Refused_Syntax_Nesting_Depth binds the Nesting Depth leaf before fixture declarations.
func Test_Refused_Syntax_Nesting_Depth(t *testing.T) {
	test_nesting_depth(t)
}

// Test_Refused_Syntax_Broken_Syntax binds the Broken Syntax leaf before fixture declarations.
func Test_Refused_Syntax_Broken_Syntax(t *testing.T) {
	test_broken_syntax(t)
}

// Test_Failure_Reports_Codes binds the Codes specification leaf before fixture declarations.
func Test_Failure_Reports_Codes(t *testing.T) {
	test_failures_codes(t)
}

// Test_Failure_Reports_Messages binds the Messages leaf before fixture declarations.
func Test_Failure_Reports_Messages(t *testing.T) {
	test_failures_messages(t)
}

// Test_Failure_Reports_Position binds the Position leaf before fixture declarations.
func Test_Failure_Reports_Position(t *testing.T) {
	test_failures_position(t)
}

// Test_Failure_Reports_Tree binds the Tree specification leaf before fixture declarations.
func Test_Failure_Reports_Tree(t *testing.T) {
	test_errors(t)
}

// Test_Approximate_Forms_Bare_Parameters binds the Bare Parameters leaf before
// fixture declarations.
func Test_Approximate_Forms_Bare_Parameters(t *testing.T) {
	test_bare_parameters(t)
}

// Test_Approximate_Forms_Bracket_Suffixes binds the Bracket Suffixes leaf before
// fixture declarations.
func Test_Approximate_Forms_Bracket_Suffixes(t *testing.T) {
	test_bracket_suffixes(t)
}

// Test_Approximate_Forms_Clause_Literals binds the Clause Literals leaf before
// fixture declarations.
func Test_Approximate_Forms_Clause_Literals(t *testing.T) {
	test_clause_literals(t)
}

// Test_Approximate_Forms_Open_Comments binds the Open Comments leaf before
// fixture declarations.
func Test_Approximate_Forms_Open_Comments(t *testing.T) {
	test_open_comments(t)
}

// Test_Bounds binds the Bounds specification leaf before fixture declarations.
func Test_Bounds(t *testing.T) {
	test_bounds(t)
}

// Test_Allocation binds the Allocation specification leaf before fixture declarations.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

const TEST_HEAD = "package p\n"

const TEST_COMMENT_SIZE = 3

const TEST_SWEEP_COUNT = 64

const TEST_RUN_HEAD_TOKENS = 8

const TEST_SMALL_RUN = 1024

const TEST_TOKEN_SWEEP = 4096

const TEST_BLANK_SATURATION = 300

const TEST_BLANK_MAXIMUM = 255

const TEST_COMMENT_COUNT = 22

const TEST_NODE_SIZE = 20

const TEST_TOKEN_SIZE = 12

const TEST_STATE_SIZE = 4197244

type allocation_fixture struct {
	Root  ast.Root
	Ok    ast.Boolean
	Node  ast.Node
	Token token.Token
}

// Walks the tree in pre-order and reports the kind of every node it meets.
func walk_kinds(
	state *ast.Parse_State, index ast.Index, output []ast.Node_Kind,
) (result []ast.Node_Kind) {
	node := ast.Node_At(state, index)
	output = append(output, node.Kind)
	child := ast.Index(node.First_Child)
	for child != ast.INDEX_ABSENT {
		output = walk_kinds(state, child, output)
		child = ast.Index(ast.Node_At(state, child).Next)
	}
	return output
}

// Parses one source and reports every node kind the tree holds.
func kinds_of(
	t *testing.T, state *ast.Parse_State, text string,
) (kinds []ast.Node_Kind) {
	root, ok := ast.Parse(state, token.Source(text))
	testify.True(t, bool(ok), "the grammar accepts %q", text)
	if ast.Index(root) == ast.INDEX_ABSENT {
		return nil
	}
	return walk_kinds(state, ast.Index(root), nil)
}

// Parses one source that the grammar accepts and states that each kind appears in its tree.
func accepts(t *testing.T, state *ast.Parse_State, text string, want []ast.Node_Kind) {
	kinds := kinds_of(t, state, text)
	for _, one := range want {
		testify.Contains(t, kinds, one, "the tree of %q holds kind %d", text, int(one))
	}
}

func test_parse(t *testing.T) {
	state := new(ast.Parse_State)
	root, ok := ast.Parse(state, token.Source(TEST_HEAD))
	testify.True(t, bool(ok), "the shortest file parses")
	testify.Equal(t, ast.Root(ast.INDEX_FIRST), root, "the file node takes the first slot")
	testify.Equal(t, ast.NODE_FILE, ast.Node_At(state, ast.Index(root)).Kind,
		"the root is a file")
	second, again := ast.Parse(state, token.Source(TEST_HEAD+"var a = 1\n"))
	testify.True(t, bool(again), "one state parses a second source")
	testify.Equal(t, ast.Root(ast.INDEX_FIRST), second,
		"the second parse starts the arena over")
}

func test_nodes(t *testing.T) {
	state := new(ast.Parse_State)
	root, ok := ast.Parse(state, token.Source(TEST_HEAD))
	testify.True(t, bool(ok), "the shortest file parses")
	file := ast.Node_At(state, ast.Index(root))
	testify.Equal(t, ast.INDEX_ABSENT, ast.Index(file.Parent), "the root has no parent")
	name := ast.Index(file.First_Child)
	testify.Equal(t, ast.INDEX_ABSENT, ast.Index(ast.Node_At(state, name).Next),
		"the only child ends the chain")
	child := ast.Node_At(state, name)
	testify.Equal(t, ast.NODE_IDENTIFIER, child.Kind, "the package name is an identifier")
	testify.Equal(t, ast.Index(root), ast.Index(child.Parent),
		"the child names the root as its parent")
	testify.Equal(t, ast.INDEX_ABSENT, ast.Index(child.Next), "the only child ends the chain")
	testify.Equal(t, "p", string(token.Text(token.Source(TEST_HEAD),
		ast.Token_At(state, child.Token))), "the child names its own token")
}

func test_package_clause(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD, []ast.Node_Kind{ast.NODE_FILE, ast.NODE_IDENTIFIER})
	accepts(t, state, "package name_9\n", []ast.Node_Kind{ast.NODE_IDENTIFIER})
}

func test_imports(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD+"import \"io\"\n",
		[]ast.Node_Kind{ast.NODE_IMPORT, ast.NODE_STRING})
	accepts(t, state, TEST_HEAD+"import alias \"io\"\n",
		[]ast.Node_Kind{ast.NODE_IMPORT, ast.NODE_IDENTIFIER})
	accepts(t, state, TEST_HEAD+"import \"io\"\nimport \"os\"\n",
		[]ast.Node_Kind{ast.NODE_IMPORT, ast.NODE_STRING})
	accepts(t, state, TEST_HEAD+"import alias \"io\"\n",
		[]ast.Node_Kind{ast.NODE_IMPORT_NAME})
}

func test_constants_and_variables(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD+"const A = 1\n",
		[]ast.Node_Kind{ast.NODE_CONSTANT, ast.NODE_INTEGER})
	accepts(t, state, TEST_HEAD+"const A Size = 1\n",
		[]ast.Node_Kind{ast.NODE_CONSTANT, ast.NODE_IDENTIFIER})
	accepts(t, state, TEST_HEAD+"var a = 1\n", []ast.Node_Kind{ast.NODE_VARIABLE})
	accepts(t, state, TEST_HEAD+"var a int\n", []ast.Node_Kind{ast.NODE_VARIABLE})
	accepts(t, state, TEST_HEAD+"var a = 1.5\n", []ast.Node_Kind{ast.NODE_FLOAT})
	accepts(t, state, TEST_HEAD+"var a = 1i\n", []ast.Node_Kind{ast.NODE_IMAGINARY})
	accepts(t, state, TEST_HEAD+"var a = 'x'\n", []ast.Node_Kind{ast.NODE_CHARACTER})
}

func test_types(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD+"type T *U\n",
		[]ast.Node_Kind{ast.NODE_TYPE, ast.NODE_POINTER_TYPE})
	accepts(t, state, TEST_HEAD+"type T []U\n", []ast.Node_Kind{ast.NODE_SLICE_TYPE})
	accepts(t, state, TEST_HEAD+"type T [SIZE]U\n", []ast.Node_Kind{ast.NODE_ARRAY_TYPE})
	accepts(t, state, TEST_HEAD+"type T [...]U\n",
		[]ast.Node_Kind{ast.NODE_ARRAY_TYPE, ast.NODE_ELLIPSIS})
	accepts(t, state, TEST_HEAD+"type T map[K]V\n", []ast.Node_Kind{ast.NODE_MAP_TYPE})
	accepts(t, state, TEST_HEAD+"type T chan V\n", []ast.Node_Kind{ast.NODE_CHANNEL_TYPE})
	accepts(t, state, TEST_HEAD+"type T chan<- V\n", []ast.Node_Kind{ast.NODE_CHANNEL_SEND})
	accepts(t, state, TEST_HEAD+"type T <-chan V\n",
		[]ast.Node_Kind{ast.NODE_CHANNEL_RECEIVE})
	accepts(t, state, TEST_HEAD+"type T func(a A) (b B)\n",
		[]ast.Node_Kind{ast.NODE_FUNCTION_TYPE, ast.NODE_PARAMETER})
	accepts(t, state, TEST_HEAD+"type T struct{ A B `json:\"a\"` }\n",
		[]ast.Node_Kind{ast.NODE_STRUCTURE_TYPE, ast.NODE_FIELD, ast.NODE_STRING})
	accepts(t, state, TEST_HEAD+"type T struct{ Embedded }\n",
		[]ast.Node_Kind{ast.NODE_FIELD})
	accepts(t, state, TEST_HEAD+"type T package_name.Other\n",
		[]ast.Node_Kind{ast.NODE_SELECTOR})
	accepts(t, state, TEST_HEAD+"type T Other[A, B]\n", []ast.Node_Kind{ast.NODE_GENERIC})
}

func test_functions(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD+"func f() {\n}\n",
		[]ast.Node_Kind{ast.NODE_FUNCTION, ast.NODE_BLOCK})
	accepts(t, state, TEST_HEAD+"func f(a A, b B) (c C) {\n}\n",
		[]ast.Node_Kind{ast.NODE_PARAMETER})
	accepts(t, state, TEST_HEAD+"func f(a ...A) (c C) {\n}\n",
		[]ast.Node_Kind{ast.NODE_ELLIPSIS})
	accepts(t, state, TEST_HEAD+"func f(subject *Value) {\n}\n",
		[]ast.Node_Kind{ast.NODE_POINTER_TYPE})
	accepts(t, state, TEST_HEAD+"func f(value Value, other Value) {\n}\n",
		[]ast.Node_Kind{ast.NODE_PARAMETER})
	accepts(t, state, TEST_HEAD+"func (subject *Value) String() (text string) {\n}\n",
		[]ast.Node_Kind{ast.NODE_RECEIVER, ast.NODE_FUNCTION})
}

func test_type_parameters(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD+"func f[Value any]() {\n}\n",
		[]ast.Node_Kind{ast.NODE_TYPE_PARAMETER})
	accepts(t, state, TEST_HEAD+"type T[Key comparable] struct{ A B }\n",
		[]ast.Node_Kind{ast.NODE_TYPE_PARAMETER})
	accepts(t, state, TEST_HEAD+"func f[Value ~int]() {\n}\n",
		[]ast.Node_Kind{ast.NODE_TERM})
	accepts(t, state, TEST_HEAD+"func f[Value ~int | ~string]() {\n}\n",
		[]ast.Node_Kind{ast.NODE_UNION})
	accepts(t, state, TEST_HEAD+"type T interface{ ~int | ~string }\n",
		[]ast.Node_Kind{ast.NODE_INTERFACE_TYPE, ast.NODE_UNION})
	accepts(t, state, TEST_HEAD+"func f[Key, Value any]() {\n}\n",
		[]ast.Node_Kind{ast.NODE_TYPE_PARAMETER})
}

func test_statements(t *testing.T) {
	state := new(ast.Parse_State)
	body := TEST_HEAD + "func f() {\n"
	accepts(t, state, statement_source(body, "a := 1"), []ast.Node_Kind{ast.NODE_DEFINE})
	accepts(t, state, statement_source(body, "a = 1"), []ast.Node_Kind{ast.NODE_ASSIGN})
	accepts(t, state, statement_source(body, "a += 1"),
		[]ast.Node_Kind{ast.NODE_OPERATION_ASSIGN})
	accepts(t, state, statement_source(body, "a++"), []ast.Node_Kind{ast.NODE_INCREMENT})
	accepts(t, state, statement_source(body, "a--"), []ast.Node_Kind{ast.NODE_DECREMENT})
	accepts(t, state, statement_source(body, "c <- 1"), []ast.Node_Kind{ast.NODE_SEND})
	accepts(t, state, statement_source(body, "return a"), []ast.Node_Kind{ast.NODE_RETURN})
	accepts(t, state, statement_source(body, "return"), []ast.Node_Kind{ast.NODE_RETURN})
	// A return that names no value parses wherever it stands, because the canonical form
	// writes the values the signature names and a parse that refused one writes nothing.
	accepts(t, state, TEST_HEAD+"func f() (a A) {\nreturn\n}\n",
		[]ast.Node_Kind{ast.NODE_RETURN})
	accepts(t, state, TEST_HEAD+"func f() (a A, b B) {\nif c {\nreturn\n}\nreturn a, b\n}\n",
		[]ast.Node_Kind{ast.NODE_RETURN})
	accepts(t, state, TEST_HEAD+"func f() {\ng(func() (a A) {\nreturn\n})\n}\n",
		[]ast.Node_Kind{ast.NODE_FUNCTION_LITERAL})
	accepts(t, state, statement_source(body, "if a {\nb()\n} else {\nc()\n}"),
		[]ast.Node_Kind{ast.NODE_IF})
	accepts(t, state, statement_source(body, "if a := f(); a {\nb()\n} else if c {\n}"),
		[]ast.Node_Kind{ast.NODE_IF, ast.NODE_DEFINE})
	accepts(t, state, statement_source(body, "for a < b {\n}"), []ast.Node_Kind{ast.NODE_FOR})
	accepts(t, state, statement_source(body, "for i := 0; i < n; i++ {\n}"),
		[]ast.Node_Kind{ast.NODE_FOR})
	accepts(t, state, statement_source(body, "for k := range m {\n}"),
		[]ast.Node_Kind{ast.NODE_RANGE})
	accepts(t, state, statement_source(body, "for range m {\n}"),
		[]ast.Node_Kind{ast.NODE_RANGE})
	accepts(t, state, statement_source(body, "for _, k := range []T{a, b} {\n}"),
		[]ast.Node_Kind{ast.NODE_COMPOSITE})
	accepts(t, state, statement_source(body, "for ; a < b; a++ {\n}"),
		[]ast.Node_Kind{ast.NODE_FOR})
	accepts(t, state, statement_source(body, "var a, b T"),
		[]ast.Node_Kind{ast.NODE_DECLARATION_STATEMENT})
	statement_branches(t, state, body)
}

func test_expressions(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD+"var a = b + c*d\n", []ast.Node_Kind{ast.NODE_BINARY})
	accepts(t, state, TEST_HEAD+"var a = -b\n", []ast.Node_Kind{ast.NODE_UNARY})
	accepts(t, state, TEST_HEAD+"var a = f(b, c)\n", []ast.Node_Kind{ast.NODE_CALL})
	accepts(t, state, TEST_HEAD+"var a = f(b...)\n", []ast.Node_Kind{ast.NODE_ELLIPSIS})
	accepts(t, state, TEST_HEAD+"var a = b[c]\n", []ast.Node_Kind{ast.NODE_INDEX})
	accepts(t, state, TEST_HEAD+"var a = b[c, d]\n", []ast.Node_Kind{ast.NODE_GENERIC})
	accepts(t, state, TEST_HEAD+"var a = \"x\"\n", []ast.Node_Kind{ast.NODE_STRING})
	accepts(t, state, TEST_HEAD+"var a = b[c:d]\n",
		[]ast.Node_Kind{ast.NODE_SLICE_EXPRESSION})
	accepts(t, state, TEST_HEAD+"var a = b[c:d:e]\n",
		[]ast.Node_Kind{ast.NODE_SLICE_EXPRESSION})
	accepts(t, state, TEST_HEAD+"var a = b.c\n", []ast.Node_Kind{ast.NODE_SELECTOR})
	accepts(t, state, TEST_HEAD+"var a = b.(C)\n", []ast.Node_Kind{ast.NODE_ASSERTION})
	accepts(t, state, TEST_HEAD+"var a = (b)\n", []ast.Node_Kind{ast.NODE_PARENTHESIS})
	accepts(t, state, TEST_HEAD+"var a = T{B: c}\n",
		[]ast.Node_Kind{ast.NODE_COMPOSITE, ast.NODE_KEY_VALUE})
	accepts(t, state, TEST_HEAD+"var a = []T{{B: c}}\n", []ast.Node_Kind{ast.NODE_COMPOSITE})
	accepts(t, state, TEST_HEAD+"var a = func(b B) (c C) {\nreturn b\n}\n",
		[]ast.Node_Kind{ast.NODE_FUNCTION_LITERAL})
	accepts(t, state, TEST_HEAD+"var a = make([]T, SIZE)\n",
		[]ast.Node_Kind{ast.NODE_SLICE_TYPE})
	accepts(t, state, TEST_HEAD+"var a = b || c && d == e\n",
		[]ast.Node_Kind{ast.NODE_BINARY})
	accepts(t, state, TEST_HEAD+"var a = b | c ^ d << e &^ f\n",
		[]ast.Node_Kind{ast.NODE_BINARY})
}

// Parses one source the grammar refuses and states the code the refusal must carry.
func refuses(
	t *testing.T, state *ast.Parse_State, text string, code ast.Failure_Code,
) {
	ast.Parse(state, token.Source(text))
	testify.Equal(t, code, ast.Failure(state), "the refusal of %.24q names its code", text)
}

func test_declaration_groups(t *testing.T) {
	state := new(ast.Parse_State)
	group := ast.FAILURE_DECLARATION_GROUP
	refuses(t, state, TEST_HEAD+"const (\nA = 1\n)\n", group)
	refuses(t, state, TEST_HEAD+"var (\na = 1\n)\n", group)
	refuses(t, state, TEST_HEAD+"type (\nT U\n)\n", group)
	refuses(t, state, TEST_HEAD+"func f() {\nconst (\nA = 1\n)\n}\n", group)
	refuses(t, state, TEST_HEAD+"import (\n\"io\"\n)\n", group)
	accepts(t, state, TEST_HEAD+"import \"io\"\n", []ast.Node_Kind{ast.NODE_IMPORT})
}

func test_interface_methods(t *testing.T) {
	state := new(ast.Parse_State)
	method := ast.FAILURE_INTERFACE_METHOD
	for _, one := range []string{
		"type R interface{ Read() int }\n",
		"type R interface {\nRead(p []byte) int\n}\n",
		"type R interface {\nRead() int\nWrite() int\n}\n",
		"type R interface {\nRead() (int, error)\n}\n",
		"type R interface {\nClose()\n}\n",
		"func f(a interface{ Read() int }) {\n}\n",
		"type T struct {\nA interface{ Read() int }\n}\n",
	} {
		refuses(t, state, TEST_HEAD+one, method)
	}
	for _, one := range []string{
		"type R interface{}\n",
		"type R interface{ any }\n",
		"type R interface{ comparable }\n",
		"type R interface{ io.Reader }\n",
		"type R interface{ ~int }\n",
		"type R interface{ int | string }\n",
		"type R interface{ ~int | ~string }\n",
		"func f[T interface{ ~int }]() {\n}\n",
	} {
		accepts(t, state, TEST_HEAD+one, []ast.Node_Kind{ast.NODE_FILE})
	}
	accepts(t, state, TEST_HEAD+"func (subject *Value) String() (text string) {\n}\n",
		[]ast.Node_Kind{ast.NODE_RECEIVER})
}

func test_wide_identifiers(t *testing.T) {
	state := new(ast.Parse_State)
	refuses(t, state, TEST_HEAD+"var é = 1\n", ast.FAILURE_WIDE_IDENTIFIER)
	accepts(t, state, TEST_HEAD+"var a = \"é\"\n", []ast.Node_Kind{ast.NODE_STRING})
	accepts(t, state, TEST_HEAD+"// é\nvar a = 1\n", []ast.Node_Kind{ast.NODE_COMMENT})
}

func test_iota(t *testing.T) {
	state := new(ast.Parse_State)
	refuses(t, state, TEST_HEAD+"const A = iota\n", ast.FAILURE_IOTA)
	refuses(t, state, TEST_HEAD+"func f() {\nb(iota)\n}\n", ast.FAILURE_IOTA)
	accepts(t, state, TEST_HEAD+"const A = 1\n", []ast.Node_Kind{ast.NODE_CONSTANT})
	accepts(t, state, TEST_HEAD+"const A = iotas\n", []ast.Node_Kind{ast.NODE_CONSTANT})
}

func test_private_fields(t *testing.T) {
	state := new(ast.Parse_State)
	private := ast.FAILURE_PRIVATE_FIELD
	refuses(t, state, TEST_HEAD+"type T struct{\nname string\n}\n", private)
	refuses(t, state, TEST_HEAD+"type T struct{\nA, bad string\n}\n", private)
	refuses(t, state, TEST_HEAD+"type T struct{\nembedded\n}\n", private)
	refuses(t, state, TEST_HEAD+"type T struct{\n*embedded\n}\n", private)
	refuses(t, state, TEST_HEAD+"type T struct{\npkg.embedded\n}\n", private)
	accepts(t, state, TEST_HEAD+"type T struct{\nName string\n}\n",
		[]ast.Node_Kind{ast.NODE_FIELD_NAME})
	accepts(t, state, TEST_HEAD+"type T struct{\nEmbedded\n}\n",
		[]ast.Node_Kind{ast.NODE_FIELD})
	accepts(t, state, TEST_HEAD+"type T struct{\npkg.Embedded\n}\n",
		[]ast.Node_Kind{ast.NODE_SELECTOR})
	accepts(t, state, TEST_HEAD+"type T struct{\nGeneric[A]\n}\n",
		[]ast.Node_Kind{ast.NODE_GENERIC})
	accepts(t, state, TEST_HEAD+"type T struct{\nNames [SIZE]byte\n}\n",
		[]ast.Node_Kind{ast.NODE_ARRAY_TYPE, ast.NODE_FIELD_NAME})
	refuses(t, state, TEST_HEAD+"type T struct{\nA[B]# string\n}\n", ast.FAILURE_SYNTAX)
	refuses(t, state, TEST_HEAD+"type T struct{\nA[B]~ string\n}\n", ast.FAILURE_SYNTAX)
	refuses(t, state, TEST_HEAD+"type T struct{\nA[B]", ast.FAILURE_SYNTAX)
	accepts(t, state, TEST_HEAD+"type T struct{\nAb string\n}\n",
		[]ast.Node_Kind{ast.NODE_FIELD_NAME})
}

func test_compound_predicates(t *testing.T) {
	state := new(ast.Parse_State)
	compound := ast.FAILURE_COMPOUND_PREDICATE
	body := TEST_HEAD + "func f() {\n"
	refuses(t, state, statement_source(body, "if a && b {\nc()\n}"), compound)
	refuses(t, state, statement_source(body, "if a || b {\nc()\n}"), compound)
	refuses(t, state, statement_source(body, "if (a && b) {\nc()\n}"), compound)
	refuses(t, state, statement_source(body, "if a {\n} else if b || c {\n}"), compound)
	refuses(t, state, statement_source(body, "if x := f(); a && b {\n}"), compound)
	accepts(t, state, statement_source(body, "if a {\nif b {\nc()\n}\n}"),
		[]ast.Node_Kind{ast.NODE_IF})
	accepts(t, state, statement_source(body, "if f(a && b) {\nc()\n}"),
		[]ast.Node_Kind{ast.NODE_CALL})
	accepts(t, state, statement_source(body, "if a == (b && c) {\nd()\n}"),
		[]ast.Node_Kind{ast.NODE_PARENTHESIS})
	accepts(t, state, statement_source(body, "for a && b {\nc()\n}"),
		[]ast.Node_Kind{ast.NODE_FOR})
	accepts(t, state, statement_source(body, "switch {\ncase a && b:\nc()\n}"),
		[]ast.Node_Kind{ast.NODE_SWITCH})
}

func test_dot_imports(t *testing.T) {
	state := new(ast.Parse_State)
	dot := ast.FAILURE_DOT_IMPORT
	refuses(t, state, TEST_HEAD+"import . \"io\"\n", dot)
	refuses(t, state, TEST_HEAD+"import \"os\"\nimport . \"io\"\n", dot)
	accepts(t, state, TEST_HEAD+"import \"io\"\n", []ast.Node_Kind{ast.NODE_IMPORT})
	accepts(t, state, TEST_HEAD+"var a = b.c\n", []ast.Node_Kind{ast.NODE_SELECTOR})
}

func test_blank_imports(t *testing.T) {
	state := new(ast.Parse_State)
	blank := ast.FAILURE_BLANK_IMPORT
	refuses(t, state, TEST_HEAD+"import _ \"io\"\n", blank)
	refuses(t, state, TEST_HEAD+"import \"os\"\nimport _ \"io\"\n", blank)
	accepts(t, state, TEST_HEAD+"import alias \"io\"\n",
		[]ast.Node_Kind{ast.NODE_IMPORT_NAME})
	accepts(t, state, TEST_HEAD+"var _ = 1\n", []ast.Node_Kind{ast.NODE_VARIABLE})
}

func test_constant_case(t *testing.T) {
	state := new(ast.Parse_State)
	uppercase := ast.FAILURE_CONSTANT_CASE
	accepts(t, state, TEST_HEAD+"const A = 1\n",
		[]ast.Node_Kind{ast.NODE_CONSTANT_NAME})
	accepts(t, state, TEST_HEAD+"const BUFFER_SIZE_MAXIMUM = 1\n",
		[]ast.Node_Kind{ast.NODE_CONSTANT})
	accepts(t, state, TEST_HEAD+"const ENUM_3_INT = 1\n",
		[]ast.Node_Kind{ast.NODE_CONSTANT})
	refuses(t, state, TEST_HEAD+"const a = 1\n", uppercase)
	refuses(t, state, TEST_HEAD+"const Buffer_Size = 1\n", uppercase)
	refuses(t, state, TEST_HEAD+"const bufferSize = 1\n", uppercase)
	refuses(t, state, TEST_HEAD+"const A_ = 1\n", uppercase)
	refuses(t, state, TEST_HEAD+"const A__B = 1\n", uppercase)
	refuses(t, state, TEST_HEAD+"const _A = 1\n", uppercase)
	refuses(t, state, TEST_HEAD+"const A, bad = 1, 2\n", uppercase)
	refuses(t, state, TEST_HEAD+"func f() {\nconst bad = 1\n_ = bad\n}\n", uppercase)
	accepts(t, state, TEST_HEAD+"var bufferSize = 1\n",
		[]ast.Node_Kind{ast.NODE_VARIABLE})
}

func test_bare_loops(t *testing.T) {
	state := new(ast.Parse_State)
	bare := ast.FAILURE_BARE_LOOP
	body := TEST_HEAD + "func f() {\n"
	refuses(t, state, statement_source(body, "for {\nb()\n}"), bare)
	refuses(t, state, statement_source(body, "for ;; {\nb()\n}"), bare)
	refuses(t, state, statement_source(body, "for true {\nb()\n}"), bare)
	refuses(t, state, statement_source(body, "for ; true; {\nb()\n}"), bare)
	accepts(t, state, statement_source(body, "for a < b {\nc()\n}"),
		[]ast.Node_Kind{ast.NODE_FOR})
	accepts(t, state, statement_source(body, "for i := 0; i < n; i++ {\n}"),
		[]ast.Node_Kind{ast.NODE_FOR})
	accepts(t, state, statement_source(body, "for range m {\nb()\n}"),
		[]ast.Node_Kind{ast.NODE_RANGE})
	accepts(t, state, statement_source(body, "for i := 0;; {\nb()\n}"),
		[]ast.Node_Kind{ast.NODE_FOR})
	accepts(t, state, statement_source(body, "for ;;i++ {\nb()\n}"),
		[]ast.Node_Kind{ast.NODE_FOR})
}

func test_unnamed_results(t *testing.T) {
	state := new(ast.Parse_State)
	unnamed := ast.FAILURE_UNNAMED_RESULT
	refuses(t, state, TEST_HEAD+"func f() A {\nreturn a\n}\n", unnamed)
	refuses(t, state, TEST_HEAD+"func f() (A, error) {\nreturn a, b\n}\n", unnamed)
	refuses(t, state, TEST_HEAD+"type T func() A\n", unnamed)
	refuses(t, state, TEST_HEAD+"func f() {\ng(func() A {\nreturn a\n})\n}\n", unnamed)
	accepts(t, state, TEST_HEAD+"func f() (a A) {\nreturn a\n}\n",
		[]ast.Node_Kind{ast.NODE_PARAMETER_NAME})
	accepts(t, state, TEST_HEAD+"func f() (a A, b B) {\nreturn a, b\n}\n",
		[]ast.Node_Kind{ast.NODE_RESULT})
	accepts(t, state, TEST_HEAD+"func f() {\nb()\n}\n",
		[]ast.Node_Kind{ast.NODE_FUNCTION})
	accepts(t, state, TEST_HEAD+"type T func() (a A)\n",
		[]ast.Node_Kind{ast.NODE_RESULT})
}

func test_type_aliases(t *testing.T) {
	state := new(ast.Parse_State)
	alias := ast.FAILURE_TYPE_ALIAS
	refuses(t, state, TEST_HEAD+"type T = U\n", alias)
	refuses(t, state, TEST_HEAD+"type T = pkg.U\n", alias)
	refuses(t, state, TEST_HEAD+"type T[K any] = U[K]\n", alias)
	refuses(t, state, TEST_HEAD+"func f() {\ntype T = U\n_ = a\n}\n", alias)
	accepts(t, state, TEST_HEAD+"type T U\n", []ast.Node_Kind{ast.NODE_TYPE})
	accepts(t, state, TEST_HEAD+"type T pkg.U\n", []ast.Node_Kind{ast.NODE_SELECTOR})
	accepts(t, state, TEST_HEAD+"var a = b\n", []ast.Node_Kind{ast.NODE_VARIABLE})
}

func test_missing_package_clause(t *testing.T) {
	state := new(ast.Parse_State)
	refuses(t, state, "var a = 1\n", ast.FAILURE_PACKAGE_CLAUSE)
	refuses(t, state, "", ast.FAILURE_PACKAGE_CLAUSE)
	accepts(t, state, TEST_HEAD, []ast.Node_Kind{ast.NODE_FILE})
}

func test_token_count(t *testing.T) {
	state := new(ast.Parse_State)
	refuses(t, state, filled_run_source(ast.TOKEN_COUNT_MAXIMUM),
		ast.FAILURE_TOKEN_COUNT)
	accepts(t, state, filled_run_source(TEST_SMALL_RUN), []ast.Node_Kind{ast.NODE_BLOCK})
}

func test_node_count(t *testing.T) {
	state := new(ast.Parse_State)
	refuses(t, state,
		filled_arena_source(ast.NODE_COUNT_MAXIMUM-int(ast.INDEX_FIRST)-1),
		ast.FAILURE_NODE_COUNT)
}

func test_nesting_depth(t *testing.T) {
	state := new(ast.Parse_State)
	refuses(t, state, nested_source(), ast.FAILURE_NESTING_DEPTH)
}

func test_broken_syntax(t *testing.T) {
	state := new(ast.Parse_State)
	for _, one := range []string{
		TEST_HEAD + "var a = )\n", TEST_HEAD + "func {\n}\n",
		TEST_HEAD + "type T\n", TEST_HEAD + "var a = #\n",
	} {
		refuses(t, state, one, ast.FAILURE_SYNTAX)
	}
}

func test_bare_parameters(t *testing.T) {
	state := new(ast.Parse_State)
	testify.Equal(t, []ast.Node_Kind{
		ast.NODE_FILE, ast.NODE_IDENTIFIER, ast.NODE_TYPE, ast.NODE_IDENTIFIER,
		ast.NODE_FUNCTION_TYPE, ast.NODE_PARAMETER, ast.NODE_IDENTIFIER,
		ast.NODE_IDENTIFIER, ast.NODE_RESULT, ast.NODE_PARAMETER_NAME,
		ast.NODE_IDENTIFIER,
	}, kinds_of(t, state, TEST_HEAD+"type T func(A, B) (c C)\n"),
		"two bare names share one parameter node and carry no type")
	accepts(t, state, TEST_HEAD+"type T func(a A, b B) (c C)\n",
		[]ast.Node_Kind{ast.NODE_PARAMETER_NAME})
	accepts(t, state, TEST_HEAD+"type T func(A, other.B, C) (d D)\n",
		[]ast.Node_Kind{ast.NODE_SELECTOR})
}

func test_bracket_suffixes(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD+"var a = b[c]\n", []ast.Node_Kind{ast.NODE_INDEX})
	accepts(t, state, TEST_HEAD+"var a = f[T](b)\n", []ast.Node_Kind{ast.NODE_INDEX})
	accepts(t, state, TEST_HEAD+"var a = f[T, U](b)\n", []ast.Node_Kind{ast.NODE_GENERIC})
	accepts(t, state, TEST_HEAD+"var a = b[c:d]\n",
		[]ast.Node_Kind{ast.NODE_SLICE_EXPRESSION})
}

func test_clause_literals(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD+"func f() {\nfor _, k := range []T{a} {\n}\n}\n",
		[]ast.Node_Kind{ast.NODE_COMPOSITE})
	accepts(t, state, TEST_HEAD+"func f() {\nif a == (T{A: 1}) {\n}\n}\n",
		[]ast.Node_Kind{ast.NODE_COMPOSITE})
	testify.Not_Contains(t,
		kinds_of(t, state, TEST_HEAD+"func f() {\nfor x := T{}; a; {\n}\n}\n"),
		ast.NODE_COMPOSITE, "a bare name in a clause opens a block and never a literal")
}

func test_open_comments(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, TEST_HEAD+"/* open\n", []ast.Node_Kind{ast.NODE_COMMENT})
	accepts(t, state, TEST_HEAD+"/* closed */\nvar a = 1\n",
		[]ast.Node_Kind{ast.NODE_COMMENT})
}

// Holds a comment at every place the grammar admits one, so the fidelity count is a count of
// positions and not of repetitions.
const TEST_COMMENT_SOURCE = "// one file\npackage p // two package\n\n" +
	"// three import\nimport \"io\" // four path\nimport \"os\" // five second\n\n" +
	"/* six general */\nconst A = 1 // seven const\n\n" +
	"// eight type\ntype T struct {\n// nine field\nA B // ten tag\n}\n\n" +
	"// eleven constraint\ntype R interface{\n// twelve element\n~int\n}\n\n" +
	"// thirteen function\nfunc f(\n// fourteen parameter\na A,\n) (c C) { // fifteen brace\n" +
	"// sixteen body\ng(\n// seventeen argument\nb,\n)\nswitch a {\n// eighteen switch\n" +
	"case 1:\n// nineteen case\nh()\n}\nx := T{\n// twenty element\nA: 1,\n}\n" +
	"_ = x\n// twenty one close\n}\n\n// twenty two file end\n"

// Reads the offset of every comment node in a pre-order walk.
func comment_offsets(
	state *ast.Parse_State, index ast.Index, output []int,
) (result []int) {
	node := ast.Node_At(state, index)
	if node.Kind == ast.NODE_COMMENT {
		output = append(output, int(ast.Token_At(state, node.Token).Offset))
	}
	child := ast.Index(node.First_Child)
	for child != ast.INDEX_ABSENT {
		output = comment_offsets(state, child, output)
		child = ast.Index(ast.Node_At(state, child).Next)
	}
	return output
}

// Reads the offset of every comment token the scanner meets.
func scanned_comment_offsets(text string) (offsets []int) {
	scanner := token.Scanner{Source: token.Source(text)}
	for range TEST_TOKEN_SWEEP {
		one := token.Scan(&scanner)
		if one.Kind == token.KIND_COMMENT {
			offsets = append(offsets, int(one.Offset))
		}
		if one.Kind == token.KIND_END_OF_FILE {
			return offsets
		}
	}
	return offsets
}

// Reads the offset and count of every blank run in a pre-order walk.
func blank_runs(
	state *ast.Parse_State, index ast.Index, output []int,
) (result []int) {
	node := ast.Node_At(state, index)
	if node.Kind == ast.NODE_BLANK {
		one := ast.Token_At(state, node.Token)
		output = append(output, int(one.Offset), int(one.Blanks))
	}
	child := ast.Index(node.First_Child)
	for child != ast.INDEX_ABSENT {
		output = blank_runs(state, child, output)
		child = ast.Index(ast.Node_At(state, child).Next)
	}
	return output
}

// Reads the offset and count of every blank run the scanner meets.
func scanned_blank_runs(text string) (runs []int) {
	scanner := token.Scanner{Source: token.Source(text)}
	for range TEST_TOKEN_SWEEP {
		one := token.Scan(&scanner)
		if one.Blanks > 0 {
			runs = append(runs, int(one.Offset), int(one.Blanks))
		}
		if one.Kind == token.KIND_END_OF_FILE {
			return runs
		}
	}
	return runs
}

func test_blank_lines(t *testing.T) {
	state := new(ast.Parse_State)
	source := "\npackage p\n\n\n// Doc.\nconst A = 1\n\nfunc f() {\n\tb()\n\n}\n\n\n\n"
	root, ok := ast.Parse(state, token.Source(source))
	testify.True(t, bool(ok), "the blank line fixture parses")
	testify.Equal(t, scanned_blank_runs(source), blank_runs(state, ast.Index(root), nil),
		"every run of empty lines stands once in the tree with its count")
	testify.Not_Empty(t, blank_runs(state, ast.Index(root), nil),
		"the fixture holds a run of empty lines")
	wide := make([]byte, TEST_BLANK_SATURATION+len(TEST_HEAD))
	for index := range TEST_BLANK_SATURATION {
		wide[index] = '\n'
	}
	copy(wide[TEST_BLANK_SATURATION:], TEST_HEAD)
	saturated, ok_wide := ast.Parse(state, token.Source(wide))
	testify.True(t, bool(ok_wide), "a file behind many empty lines parses")
	testify.Equal(t, TEST_BLANK_MAXIMUM,
		int(ast.Token_At(state, ast.Node_At(state, ast.Index(saturated)).Token).Blanks),
		"a run past the count that fits saturates")
	tight, ok_tight := ast.Parse(state, token.Source(TEST_HEAD+"var a = 1\n"))
	testify.True(t, bool(ok_tight), "a file with no empty line parses")
	testify.Empty(t, blank_runs(state, ast.Index(tight), nil),
		"a file with no empty line holds no blank node")
}

func test_comments_fidelity(t *testing.T) {
	state := new(ast.Parse_State)
	root, ok := ast.Parse(state, token.Source(TEST_COMMENT_SOURCE))
	testify.True(t, bool(ok), "the comment fixture parses")
	scanned := scanned_comment_offsets(TEST_COMMENT_SOURCE)
	testify.Equal(t, TEST_COMMENT_COUNT, len(scanned),
		"the fixture holds a comment at every admitted place")
	testify.Equal(t, scanned, comment_offsets(state, ast.Index(root), nil),
		"every comment the scanner reads stands once in the tree")
}

func test_comments_order(t *testing.T) {
	state := new(ast.Parse_State)
	root, ok := ast.Parse(state, token.Source(TEST_COMMENT_SOURCE))
	testify.True(t, bool(ok), "the comment fixture parses")
	offsets := comment_offsets(state, ast.Index(root), nil)
	for index := 1; index < len(offsets); index++ {
		testify.True(t, offsets[index] > offsets[index-1],
			"comment %d stands after the one before it", index)
	}
}

func test_comments_attachment(t *testing.T) {
	state := new(ast.Parse_State)
	source := "// Doc.\nconst A = 1\n"
	root, ok := ast.Parse(state, token.Source(TEST_HEAD+source))
	testify.True(t, bool(ok), "the attachment fixture parses")
	child := ast.Index(ast.Node_At(state, ast.Index(root)).First_Child)
	kinds := []ast.Node_Kind{}
	for child != ast.INDEX_ABSENT {
		kinds = append(kinds, ast.Node_At(state, child).Kind)
		child = ast.Index(ast.Node_At(state, child).Next)
	}
	testify.Equal(t, []ast.Node_Kind{
		ast.NODE_IDENTIFIER, ast.NODE_COMMENT, ast.NODE_CONSTANT,
	}, kinds, "a doc comment is the sibling ahead of its declaration")
	test_comments(t)
}

func test_comments(t *testing.T) {
	state := new(ast.Parse_State)
	accepts(t, state, "// Package p does nothing.\n"+TEST_HEAD,
		[]ast.Node_Kind{ast.NODE_COMMENT})
	accepts(t, state, TEST_HEAD+"// Doc.\nvar a = 1\n", []ast.Node_Kind{ast.NODE_COMMENT})
	accepts(t, state, TEST_HEAD+"var a = 1 // Trailing.\n",
		[]ast.Node_Kind{ast.NODE_COMMENT})
	accepts(t, state, TEST_HEAD+"/* General. */\nvar a = 1\n",
		[]ast.Node_Kind{ast.NODE_COMMENT})
	accepts(t, state, TEST_HEAD+"func f() {\n// Inside.\na()\n}\n",
		[]ast.Node_Kind{ast.NODE_COMMENT})
}

func test_errors(t *testing.T) {
	state := new(ast.Parse_State)
	broken := []string{
		"", "var a = 1\n", TEST_HEAD + "import 1\n", TEST_HEAD + "func {\n}\n",
		TEST_HEAD + "type T = \n", TEST_HEAD + "var a = #\n",
		TEST_HEAD + "var a = b ~ c\n", TEST_HEAD + "func f() {\nswitch {\ncase\n}\n}\n",
		TEST_HEAD + "type T struct{ a ~int }\n", TEST_HEAD + "\xff\n",
		TEST_HEAD + "var a = b.", TEST_HEAD + "type T struct{ a # }\n",
		TEST_HEAD + "var a = b +", TEST_HEAD + "var a = b # c\n",
		TEST_HEAD + "var a = b c\n", TEST_HEAD + "func f() {\na.",
		TEST_HEAD + "func f() {\na ~ b\n}\n", TEST_HEAD + "func f() {\na # b\n}\n",
		TEST_HEAD + "func f() {\na b\n}\n", TEST_HEAD + "type T map[",
	}
	for _, one := range broken {
		root, ok := ast.Parse(state, token.Source(one))
		testify.False(t, bool(ok), "the grammar rejects %q", one)
		if ast.Index(root) == ast.INDEX_ABSENT {
			continue
		}
		kinds := walk_kinds(state, ast.Index(root), nil)
		testify.Contains(t, kinds, ast.NODE_ERROR,
			"the tree of %q holds an error node", one)
	}
	testify.Equal(t, ast.NODE_ERROR, ast.Node_At(state, ast.INDEX_ABSENT).Kind,
		"the absent slot is loud")
	// A run the source overran leaves no tree, because the descent never started.
	root, _ := ast.Parse(state, token.Source(filled_run_source(ast.TOKEN_COUNT_MAXIMUM)))
	testify.Equal(t, ast.Root(ast.INDEX_ABSENT), root,
		"a source past the token count returns no tree")
}

// Wraps one statement in the shortest file that can hold it.
func statement_source(head string, one string) (text string) {
	return head + one + "\n}\n"
}

// States the statement forms whose count would push one test past its size.
func statement_branches(t *testing.T, state *ast.Parse_State, body string) {
	accepts(t, state, statement_source(body, "switch a {\ncase 1, 2:\nb()\ndefault:\nc()\n}"),
		[]ast.Node_Kind{ast.NODE_SWITCH, ast.NODE_CASE, ast.NODE_DEFAULT})
	accepts(t, state, statement_source(body, "switch a := b.(type) {\ncase C:\nd()\n}"),
		[]ast.Node_Kind{ast.NODE_TYPE_SWITCH})
	accepts(t, state, statement_source(body, "select {\ncase a := <-c:\nb()\n}"),
		[]ast.Node_Kind{ast.NODE_SELECT})
	accepts(t, state, statement_source(body, "go f()"), []ast.Node_Kind{ast.NODE_GO})
	accepts(t, state, statement_source(body, "defer f()"), []ast.Node_Kind{ast.NODE_DEFER})
	accepts(t, state, statement_source(body, "Loop:\nfor a {\nbreak Loop\n}"),
		[]ast.Node_Kind{ast.NODE_LABEL, ast.NODE_BREAK})
	accepts(t, state, statement_source(body, "for a {\ncontinue\n}"),
		[]ast.Node_Kind{ast.NODE_CONTINUE})
	accepts(t, state, statement_source(body, "goto Done"), []ast.Node_Kind{ast.NODE_GOTO})
	accepts(t, state, statement_source(body, "switch a {\ncase 1:\nfallthrough\ncase 2:\n}"),
		[]ast.Node_Kind{ast.NODE_FALLTHROUGH})
	accepts(t, state, statement_source(body, "{\na()\n}"), []ast.Node_Kind{ast.NODE_BLOCK})
	accepts(t, state, statement_source(body, "var a int"),
		[]ast.Node_Kind{ast.NODE_DECLARATION_STATEMENT})
	accepts(t, state, statement_source(body, "type T U"),
		[]ast.Node_Kind{ast.NODE_DECLARATION_STATEMENT})
	accepts(t, state, statement_source(body, "f()"),
		[]ast.Node_Kind{ast.NODE_EXPRESSION_STATEMENT})
	accepts(t, state, statement_source(body, ";"), []ast.Node_Kind{ast.NODE_BLOCK})
}

// Builds a file whose comment count fills the arena to the slot this test names.
func filled_arena_source(comments int) (text string) {
	buffer := make([]byte, len(TEST_HEAD)+comments*TEST_COMMENT_SIZE)
	copy(buffer, TEST_HEAD)
	for count := len(TEST_HEAD); count < len(buffer); count += TEST_COMMENT_SIZE {
		buffer[count] = '/'
		buffer[count+1] = '/'
		buffer[count+2] = '\n'
	}
	return string(buffer)
}

// Builds a file whose body holds the empty statement count this test names.
func filled_run_source(statements int) (text string) {
	head := TEST_HEAD + "func f() {\n"
	buffer := make([]byte, len(head)+statements+2)
	copy(buffer, head)
	for count := len(head); count < len(buffer)-2; count++ {
		buffer[count] = ';'
	}
	buffer[len(buffer)-2] = '}'
	buffer[len(buffer)-1] = '\n'
	return string(buffer)
}

// Builds a source nested past the depth one parse admits.
func nested_source() (text string) {
	buffer := make([]byte, 0, ast.DEPTH_MAXIMUM*2+len(TEST_HEAD)+16)
	buffer = append(buffer, TEST_HEAD...)
	buffer = append(buffer, "var a = "...)
	for range ast.DEPTH_MAXIMUM + 4 {
		buffer = append(buffer, '(')
	}
	buffer = append(buffer, 'b')
	for range ast.DEPTH_MAXIMUM + 4 {
		buffer = append(buffer, ')')
	}
	buffer = append(buffer, '\n')
	return string(buffer)
}

func test_failures_codes(t *testing.T) {
	state := new(ast.Parse_State)
	full := ast.NODE_COUNT_MAXIMUM - int(ast.INDEX_FIRST) - 1
	cases := []struct {
		Code   ast.Failure_Code
		Source string
	}{
		{ast.FAILURE_NONE, TEST_HEAD + "var a = 1\n"},
		{ast.FAILURE_SYNTAX, TEST_HEAD + "var a = )\n"},
		{ast.FAILURE_DECLARATION_GROUP, TEST_HEAD + "const (\nA = 1\n)\n"},
		{ast.FAILURE_DECLARATION_GROUP, TEST_HEAD + "var (\na = 1\n)\n"},
		{ast.FAILURE_DECLARATION_GROUP, TEST_HEAD + "type (\nT U\n)\n"},
		{ast.FAILURE_INTERFACE_METHOD,
			TEST_HEAD + "type R interface{\nRead(p []byte) int\n}\n"},
		{ast.FAILURE_WIDE_IDENTIFIER, TEST_HEAD + "var é = 1\n"},
		{ast.FAILURE_PACKAGE_CLAUSE, "var a = 1\n"},
		{ast.FAILURE_TOKEN_COUNT, filled_run_source(ast.TOKEN_COUNT_MAXIMUM)},
		{ast.FAILURE_NODE_COUNT, filled_arena_source(full)},
		{ast.FAILURE_NESTING_DEPTH, nested_source()},
	}
	for _, one := range cases {
		ast.Parse(state, token.Source(one.Source))
		testify.Equal(t, one.Code, ast.Failure(state),
			"the parse of %.20q names its cause", one.Source)
	}
}

func test_failures_messages(t *testing.T) {
	seen := map[string]bool{}
	for code := range int(ast.FAILURE_NESTING_DEPTH) + 1 {
		text := string(ast.Failure_Message(ast.Failure_Code(code)))
		testify.Not_Empty(t, text, "code %d reads as a sentence", code)
		testify.False(t, seen[text], "code %d reads as a sentence of its own", code)
		seen[text] = true
	}
	testify.Equal(t, "Write one declaration for each name and drop the parentheses.",
		string(ast.Failure_Message(ast.FAILURE_DECLARATION_GROUP)),
		"the group message says what to write")
	testify.Equal(t, "Declare no interface; a generic constraint is the one exception.",
		string(ast.Failure_Message(ast.FAILURE_INTERFACE_METHOD)),
		"the interface message names the one exception")
}

func test_failures_position(t *testing.T) {
	state := new(ast.Parse_State)
	source := TEST_HEAD + "const (\nA = 1\n)\n"
	ast.Parse(state, token.Source(source))
	testify.Equal(t, "(", string(token.Text(token.Source(source),
		ast.Token_At(state, ast.Failure_Token(state)))),
		"the failure names the parenthesis that opened the group")
	// A refusal that a later pass names still points at the token that broke the rule, not at
	// the token the cursor happened to rest on.
	for _, one := range []struct {
		Source string
		Names  string
	}{
		{Source: TEST_HEAD + "const A = iota\n", Names: "iota"},
		{Source: TEST_HEAD + "type T struct{\nname string\n}\n", Names: "name"},
		{Source: TEST_HEAD + "func f() {\nif a && b {\nc()\n}\n}\n", Names: "&&"},
	} {
		ast.Parse(state, token.Source(one.Source))
		testify.Equal(t, one.Names, string(token.Text(token.Source(one.Source),
			ast.Token_At(state, ast.Failure_Token(state)))),
			"the refusal of %.24q names the token that broke the rule", one.Source)
	}
	overrun := filled_run_source(ast.TOKEN_COUNT_MAXIMUM)
	ast.Parse(state, token.Source(overrun))
	testify.Not_Equal(t, token.KIND_ILLEGAL,
		ast.Token_At(state, ast.Failure_Token(state)).Kind,
		"an overrun run still holds the token the failure names")
}

func test_bounds(t *testing.T) {
	state := new(ast.Parse_State)
	arena_bounds(t, state)
	run_bounds(t, state)
	depth_bounds(t, state)
	token_bounds(t, state)
	link_bounds(t, state)
	footprint_bounds(t)
}

// States the storage one parse costs. A node is four four-byte links and one byte of class, and
// a token is two four-byte spans and one byte of class, thus the whole state stays inside four
// mebibytes and a caller can hold one for each goroutine.
func footprint_bounds(t *testing.T) {
	testify.Equal(t, uintptr(TEST_NODE_SIZE), unsafe.Sizeof(ast.Node{}),
		"one node spans its declared width")
	testify.Equal(t, uintptr(TEST_TOKEN_SIZE), unsafe.Sizeof(token.Token{}),
		"one token spans its declared width")
	testify.Equal(t, uintptr(TEST_STATE_SIZE), unsafe.Sizeof(ast.Parse_State{}),
		"one parse state spans its declared width")
}

func arena_bounds(t *testing.T, state *ast.Parse_State) {
	exact := ast.NODE_COUNT_MAXIMUM - int(ast.INDEX_FIRST) - 2
	root, ok := ast.Parse(state, token.Source(filled_arena_source(exact)))
	testify.True(t, bool(ok), "a file that exactly fills the arena parses")
	testify.Equal(t, ast.Root(ast.INDEX_FIRST), root,
		"the filled parse still roots at the first slot")
	testify.Equal(t, ast.NODE_COMMENT,
		ast.Node_At(state, ast.Index(ast.INDEX_MAXIMUM)).Kind,
		"the final slot holds a node")
	_, over := ast.Parse(state, token.Source(filled_arena_source(exact+1)))
	testify.False(t, bool(over), "a file past the arena bound is rejected")
}

func run_bounds(t *testing.T, state *ast.Parse_State) {
	_, ok := ast.Parse(state, token.Source(filled_run_source(1024)))
	testify.True(t, bool(ok), "a body of empty statements writes no node")
	_, over := ast.Parse(state,
		token.Source(filled_run_source(ast.TOKEN_COUNT_MAXIMUM)))
	testify.False(t, bool(over), "a source past the token bound is rejected")
}

func depth_bounds(t *testing.T, state *ast.Parse_State) {
	buffer := make([]byte, 0, ast.DEPTH_MAXIMUM*2+len(TEST_HEAD)+16)
	buffer = append(buffer, TEST_HEAD...)
	buffer = append(buffer, "var a = "...)
	for range ast.DEPTH_MAXIMUM + 4 {
		buffer = append(buffer, '(')
	}
	buffer = append(buffer, 'b')
	for range ast.DEPTH_MAXIMUM + 4 {
		buffer = append(buffer, ')')
	}
	buffer = append(buffer, '\n')
	_, ok := ast.Parse(state, token.Source(buffer))
	testify.False(t, bool(ok), "a source past the depth bound is rejected")
}

func test_allocation(t *testing.T) {
	state := new(ast.Parse_State)
	fixture := allocation_fixture{}
	source := token.Source(TEST_HEAD + "var a = b + c\n")
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Parse", Call: func() {
			fixture.Root, fixture.Ok = ast.Parse(state, source)
		}},
		{Name: "Node_At", Call: func() {
			fixture.Node = ast.Node_At(state, ast.Index(fixture.Root))
		}},
		{Name: "Token_At", Call: func() {
			fixture.Token = ast.Token_At(state, fixture.Node.Token)
		}},
	}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	testify.True(t, bool(fixture.Ok), "the allocation fixture parses")
}

// Builds a source that spans the largest text the scanner admits.
func largest_source() (text string) {
	buffer := make([]byte, token.SOURCE_SIZE_MAXIMUM)
	for index := range buffer {
		buffer[index] = 'a'
	}
	copy(buffer, "package p\n/*")
	copy(buffer[len(buffer)-3:], "*/\n")
	return string(buffer)
}

// Builds a source that is one comment spanning the largest text the scanner admits.
func largest_comment() (text string) {
	buffer := make([]byte, token.SOURCE_SIZE_MAXIMUM)
	for index := range buffer {
		buffer[index] = 'a'
	}
	copy(buffer, "/*")
	copy(buffer[len(buffer)-2:], "*/")
	return string(buffer)
}

// Builds a body of empty statements that fills the token run to its final slot and then ends
// without its closing brace, so the parse rejects at the token that closes the run.
func filled_token_run() (text string) {
	head := TEST_HEAD + "func f() {\n"
	buffer := make([]byte, len(head)+ast.TOKEN_COUNT_MAXIMUM-TEST_RUN_HEAD_TOKENS-1)
	copy(buffer, head)
	for count := len(head); count < len(buffer); count++ {
		buffer[count] = ';'
	}
	return string(buffer)
}

// States the source sizes and the token positions the accessors must see.
func token_bounds(t *testing.T, state *ast.Parse_State) {
	ast.Parse(state, token.Source("a"))
	ast.Parse(state, token.Source("ab"))
	sweep_tokens(t, state, TEST_HEAD+"var a = b ~ \xff\n")
	sweep_tokens(t, state, "a:=b")
	sweep_tokens(t, state, "ab=c")
	sweep_tokens(t, state, largest_source())
	sweep_tokens(t, state, largest_comment())
	root, ok := ast.Parse(state, token.Source(filled_token_run()))
	testify.False(t, bool(ok), "a body with no closing brace is rejected")
	testify.Contains(t, walk_kinds(state, ast.Index(root), nil), ast.NODE_ERROR,
		"the run that ends with no closing brace holds an error node")
	last := ast.Token_Index(ast.TOKEN_COUNT_MAXIMUM - 1)
	testify.Equal(t, token.KIND_END_OF_FILE, ast.Token_At(state, last).Kind,
		"the final run slot holds the end of file")
	testify.Equal(t, last, ast.Failure_Token(state),
		"a run that ends with no closing brace fails at its final token")
	ast.Parse(state, token.Source("package )\n"))
	testify.Equal(t, ast.Token_Index(1), ast.Failure_Token(state),
		"a missing package name fails at the second token")
	ast.Parse(state, token.Source("package p )\n"))
	testify.Equal(t, ast.Token_Index(2), ast.Failure_Token(state),
		"a stray bracket fails at the third token")
}

// Reads every token of one parse back through the accessor.
func sweep_tokens(t *testing.T, state *ast.Parse_State, text string) {
	ast.Parse(state, token.Source(text))
	for index := range TEST_SWEEP_COUNT {
		one := ast.Token_At(state, ast.Token_Index(index))
		if one.Kind == token.KIND_END_OF_FILE {
			return
		}
	}
}

// States the link values that only a filled arena can show.
func link_bounds(t *testing.T, state *ast.Parse_State) {
	exact := ast.NODE_COUNT_MAXIMUM - int(ast.INDEX_FIRST) - 2
	ast.Parse(state, token.Source(filled_arena_source(exact)))
	final := ast.Index(ast.INDEX_MAXIMUM)
	testify.Equal(t, ast.Successor(final), ast.Node_At(state, final-1).Next,
		"the node before the final slot names it as its follower")
	_, over := ast.Parse(state,
		token.Source(filled_arena_source(exact-2)+"var a int\n"))
	testify.False(t, bool(over), "a declaration past the arena bound is rejected")
	testify.Equal(t, ast.Head(final), ast.Node_At(state, final-1).First_Child,
		"the declaration before the final slot opens its chain there")
	testify.Equal(t, ast.Ancestor(final-1), ast.Node_At(state, final).Parent,
		"the final slot names the declaration before it as its parent")
}
