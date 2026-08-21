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
	Ok    printer.Boolean
}

// Formats one source and hands back the form it wrote, which is how each case states the form it
// expects.
func formatted(t *testing.T, source string) (form string) {
	t.Helper()
	state := new(format.Formatter)
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	held := token.Source(source)
	ast.Parse(tree, held)
	count, ok := format.Format(state, subject, storage, tree, held)
	testify.True(t, bool(ok), "the format holds the whole form")
	return string(storage[:count])
}

// Reports whether one source stands in the canonical form.
func cleanliness(t *testing.T, source string) (clean printer.Boolean) {
	t.Helper()
	state := new(format.Formatter)
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	held := token.Source(source)
	ast.Parse(tree, held)
	clean, ok := format.Clean(state, subject, storage, tree, held)
	testify.True(t, bool(ok), "the clean report holds the whole form")
	return clean
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
	state := new(format.Formatter)
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	source := token.Source("package one\n\ntype Count int\n")
	ast.Parse(tree, source)
	for _, width_size := range []int{0, 1, 2, 4} {
		narrow := make([]byte, width_size)
		count, ok := format.Format(state, subject, narrow, tree, source)
		testify.False(t, bool(ok), "a form past the storage is refused")
		testify.Equal(t, printer.Form_Count(width_size), count,
			"a refused format states the storage it filled")
		clean, held := format.Clean(state, subject, narrow, tree, source)
		testify.False(t, bool(held), "a clean report past the storage is refused")
		testify.False(t, bool(clean), "a source the format refused is no clean source")
	}
	// A tree the parser refused states the declarations it holds, thus the caller reads a
	// partial form rather than nothing at all.
	broken := token.Source("package one\n\nfunc (\n")
	ast.Parse(tree, broken)
	wide := make([]byte, printer.FORM_SIZE_MAXIMUM)
	_, ok := format.Format(state, subject, wide, tree, broken)
	testify.True(t, bool(ok), "a refused parse formats the tree it holds")
	// A source of a few bytes states no file at all, thus the format writes the clause those
	// bytes name and states that no such source is clean.
	for _, text := range []string{"", "p", "pa"} {
		narrow := token.Source(text)
		ast.Parse(tree, narrow)
		count, held := format.Format(state, subject, wide, tree, narrow)
		testify.True(t, bool(held), "a format of a source of a few bytes holds its form")
		testify.True(t, count <= printer.Form_Count(len(text)+9),
			"a source of a few bytes writes the clause those bytes name and no more")
		clean, stand := format.Clean(state, subject, wide, tree, narrow)
		testify.True(t, bool(stand), "a clean report of a few bytes holds its form")
		testify.False(t, bool(clean), "a source stating no file is no clean source")
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
	state := new(format.Formatter)
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	source := token.Source(wide_source())
	testify.Equal(t, token.SOURCE_SIZE_MAXIMUM, len(source),
		"the source states the widest size")
	ast.Parse(tree, source)
	count, ok := format.Format(state, subject, storage, tree, source)
	testify.False(t, bool(ok), "a form past the widest form is refused")
	testify.Equal(t, printer.Form_Count(printer.FORM_SIZE_MAXIMUM), count,
		"one format writes at most the widest form")
	// The storage the clean report reads is narrow on purpose: a report the storage refuses
	// answers at the first bytes it writes, thus the widest source costs one form and not two.
	narrow := make([]byte, 4)
	clean, held := format.Clean(state, subject, narrow, tree, source)
	testify.False(t, bool(held), "a clean report of a form past the storage is refused")
	testify.False(t, bool(clean), "a source whose form was never written is no clean source")
}

func test_allocation(t *testing.T) {
	held := allocation_fixture{}
	state := new(format.Formatter)
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	source := token.Source("package one\n\ntype Count int\n\nfunc Fold() (sum Count) {\n" +
		"\treturn sum\n}\n")
	ast.Parse(tree, source)
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Format", Call: func() {
			held.Count, held.Ok = format.Format(state, subject, storage, tree, source)
		}},
		{Name: "Clean", Call: func() {
			held.Clean, held.Ok = format.Clean(state, subject, storage, tree, source)
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
			Source: "package one\n\nimport first \"zz\"\nimport \"aa\"\n",
			Form:   "package one\n\nimport \"aa\"\nimport first \"zz\"\n",
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
