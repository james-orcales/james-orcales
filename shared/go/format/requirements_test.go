package format_test

import (
	"testing"

	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/format"
	"local/james-orcales/shared/go/printer"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/testify"
)

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
		Names: format.Name_Storage{
			Values: make(format.Names, format.NAME_SLOT_COUNT),
		},
		Texts: format.Text_Storage{
			Values: make(format.Text, format.NAME_TEXT_SIZE_MAXIMUM),
		},
		Slots: format.Slot_Storage{
			Values: make(format.Slots, format.SLOT_COUNT),
		},
	}
}

type name_case struct {
	Source string
	Form   string
}

// States each source whose declared names the format renames, beside the form it expects.
func name_cases() (cases []name_case) {
	return []name_case{
		{
			Source: "const fooBar = 1\n\nvar held = fooBar\n",
			Form:   "const FOO_BAR = 1\n\nvar held = FOO_BAR\n",
		},
		{
			Source: "var someValue, Other_value = 1, 2\n",
			Form:   "var some_value, Other_Value = 1, 2\n",
		},
		{
			Source: "type HTTPServer struct {\n\tFieldOne int\n\tfieldTwo int\n}\n",
			Form:   "type HTTP_Server struct {\n\tField_One int\n\tfield_two int\n}\n",
		},
		{
			Source: "func DoWork(inputValue int) (resultValue int) {\n" +
				"\tlocalValue := inputValue\n\tresultValue = localValue\n" +
				"\treturn resultValue\n}\n",
			Form: "func Do_Work(input_value int) (result_value int) {\n" +
				"\tlocal_value := input_value\n\tresult_value = local_value\n" +
				"\treturn result_value\n}\n",
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
			Source: "func TestMain(value int) {\n\treturn\n}\n",
			Form:   "func TestMain(value int) {\n\treturn\n}\n",
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
		{
			Source: "func (heldPair Pair) Fold() (sum int) {\n" +
				"\tfor keyA, valueB := range heldPair {\n" +
				"\t\tsum = keyA + valueB\n\t}\n\treturn sum\n}\n",
			Form: "func (held_pair Pair) Fold() (sum int) {\n" +
				"\tfor key_a, value_b := range held_pair {\n" +
				"\t\tsum = key_a + value_b\n\t}\n\treturn sum\n}\n",
		},
	}
}

func test_names(t *testing.T) {
	head := "package one\n\n"
	for _, one := range name_cases() {
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
	testify.Equal(t, printer.Form_Count(len(narrow)), result.Count,
		"a refused rename states the storage it filled")
}

// Caller allocation keeps the printer free of hidden storage growth.
func printer_state() (state *printer.Printer) {
	return &printer.Printer{
		Printer_Fields: printer.Printer_Fields{
			Nodes: printer.Node_Stack_Storage{
				Values: make(printer.Node_Stack, printer.DEPTH_MAXIMUM),
			},
			Widths: printer.Field_Width_Storage{
				Values: make(printer.Field_Widths, printer.FIELD_COUNT_MAXIMUM),
			},
			Results: printer.Result_Token_Storage{
				Values: make(printer.Result_Tokens, printer.RESULT_COUNT_MAXIMUM),
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
			Counts: printer.Count_Storage{
				Values: make(printer.Counts, printer.COUNT_SLOT_COUNT),
			},
			Flags: printer.Flag_Storage{
				Values: make(printer.Flags, printer.FLAG_COUNT),
			},
		},
	}
}

// Caller allocation makes parser bounds part of every format fixture.
func parse_state() (state *ast.Parse_State) {
	return &ast.Parse_State{Parse_State_Fields: ast.Parse_State_Fields{
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
	}}
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
	canonical := "package one\n\nfunc Fold() (sum int) {\n\treturn sum\n}\n"
	testify.Equal(t, canonical, formatted(t, canonical), "the canonical form formats as itself")
	testify.Equal(t, "package one\n\nconst VALUE = 0xFF\n",
		formatted(t, "package one\n\nconst VALUE = 0XFF\n"),
		"a literal the source spelled loosely formats in the canonical form")
	testify.Equal(t, "package one\n\nfunc Fold() {\n\treturn\n}\n",
		formatted(t, "package one\nfunc Fold() {\nreturn\n}\n"),
		"a source the author wrote flat formats with the lines and tabs the form states")
}

func test_clean(t *testing.T) {
	testify.True(t, bool(cleanliness(t, "package one\n")),
		"a source in the canonical form is clean")
	testify.False(t, bool(cleanliness(t, "package one\n\nconst VALUE = 0XFF\n")),
		"a source stating a literal loosely is no clean source")
	testify.False(t, bool(cleanliness(t, "package one\nconst VALUE = 1\n")),
		"a source missing the empty line one declaration opens with is no clean source")
	testify.False(t, bool(cleanliness(t, "package one\n\nconst VALUE = 1")),
		"a source closing on no line feed is no clean source")
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
		testify.Equal(t, printer.Form_Count(width_size), result.Count,
			"a refused format states the storage it filled")
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
	// A source of a few bytes states no file at all, thus the format writes the clause those
	// bytes name and states that no such source is clean.
	for _, text := range []string{"", "p", "pa"} {
		narrow := token.Source(text)
		ast.Parse(tree, narrow)
		narrow_result := format.Format(state, subject, wide, tree, narrow)
		testify.True(t, bool(narrow_result.OK),
			"a format of a source of a few bytes holds its form")
		testify.True(t, narrow_result.Count <= printer.Form_Count(len(text)+9),
			"a source of a few bytes writes the clause those bytes name and no more")
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
	for range 21000 {
		body = append(body, "x++\n"...)
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
	testify.Equal(t, printer.Form_Count(printer.FORM_SIZE_MAXIMUM), result.Count,
		"one format writes at most the widest form")
	// The storage the clean report reads is narrow on purpose: a report the storage refuses
	// answers at the first bytes it writes, thus the widest source costs one form and not two.
	narrow := make([]byte, 4)
	clean := format.Clean(state, subject, narrow, tree, source)
	testify.False(t, bool(clean.OK), "a clean report of a form past the storage is refused")
	testify.False(t, bool(clean.Clean),
		"a source whose form was never written is no clean source")
}

func test_allocation(t *testing.T) {
	held := allocation_fixture{}
	state := format_state()
	subject := printer_state()
	tree := parse_state()
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	source := token.Source("package one\n\ntype Count int\n\nfunc Fold() (sum Count) {\n" +
		"\treturn sum\n}\n")
	ast.Parse(tree, source)
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Format", Call: func() {
			result := format.Format(state, subject, storage, tree, source)
			held.Count, held.Ok = result.Count, bool(result.OK)
		}},
		{Name: "Clean", Call: func() {
			result := format.Clean(state, subject, storage, tree, source)
			held.Clean, held.Ok = result.Clean, bool(result.OK)
		}},
	}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	testify.True(t, bool(held.Ok), "the allocation fixture formats its file")
	testify.True(t, bool(held.Clean), "the allocation fixture states a clean source")
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
			Form: "package one\n\nimport \"mm\"\nimport \"zz\"\n\nimport \"aa\"\n" +
				"import \"bb\"\n",
		},
		{
			Source: "package one\n\nimport \"zz\"\n\nvar one = 1\n",
			Form:   "package one\n\nimport \"zz\"\n\nvar one = 1\n",
		},
	} {
		testify.Equal(t, one.Form, formatted(t, one.Source),
			"the imports of one run stand in the order of the paths they name")
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
	state := format_state()
	subject := printer_state()
	tree := parse_state()
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	declared := token.Source("package one\n\nvar someValue = 1\n\nconst maxSize = 2\n")
	ast.Parse(tree, declared)
	testify.True(t, bool(format.Declare(state, tree, declared)),
		"a table with room takes every name of the file")
	source := token.Source("package one\n\nvar held = someValue + maxSize\n")
	ast.Parse(tree, source)
	result := format.Format(state, subject, storage, tree, source)
	testify.True(t, bool(result.OK), "the format holds the whole form")
	testify.Equal(t, "package one\n\nvar held = some_value + MAX_SIZE\n",
		string(storage[:result.Count]),
		"a name another file declares takes its case in every file that spells it")
	format.Forget(state)
	source = token.Source("package one\n\nvar heldValue = 1\n")
	ast.Parse(tree, source)
	result = format.Format(state, subject, storage, tree, source)
	testify.Equal(t, string(source), string(storage[:result.Count]),
		"a name no declaration took stands as written, though the file declares it")
	format.Forget(state)
	ast.Parse(tree, source)
	result = format.Format(state, subject, storage, tree, source)
	testify.Equal(t, string(source), string(storage[:result.Count]),
		"a forgotten table renames nothing the file does not declare")
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
		testify.True(t, bool(format.Declare(state, tree, narrow)),
			"a source of a few bytes declares nothing")
	}
}
