package printer_test

import (
	"testing"

	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/printer"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/testify"
)

// Test_Printer binds the Printer specification leaf before fixture declarations.
func Test_Printer(t *testing.T) {
	test_printer(t)
}

// Test_Layout binds the Layout specification leaf before fixture declarations.
func Test_Layout(t *testing.T) {
	test_layout(t)
}

// Test_Spacing binds the Spacing specification leaf before fixture declarations.
func Test_Spacing(t *testing.T) {
	test_spacing(t)
}

// Test_Alignment binds the Alignment specification leaf before fixture declarations.
func Test_Alignment(t *testing.T) {
	test_alignment(t)
}

// Test_Trivia binds the Trivia specification leaf before fixture declarations.
func Test_Trivia(t *testing.T) {
	test_trivia(t)
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

// Prints one source and hands back what the print wrote, which is how every case states the form
// it expects.
func printed(t *testing.T, source string) (form string) {
	t.Helper()
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	ast.Parse(tree, token.Source(source))
	count, ok := printer.Print(subject, storage[:], tree, token.Source(source))
	testify.True(t, bool(ok), "the print holds the whole form")
	return string(storage[:count])
}

// Reports that one canonical source prints as itself, which is what a caller asks when it wants
// to know whether a file is clean.
func round_trip(t *testing.T, source string) {
	t.Helper()
	testify.Equal(t, source, printed(t, source), "the canonical form prints as itself")
}

func test_printer(t *testing.T) {
	round_trip(t, "package one\n")
	round_trip(t, "package one\n\nimport \"local/example/two\"\n")
	round_trip(t, "package one\n\nimport two \"local/example/two\"\n")
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	source := token.Source("package one\n")
	ast.Parse(tree, source)
	count, ok := printer.Print(subject, storage, tree, source)
	testify.True(t, bool(ok), "a fresh printer prints")
	testify.Equal(t, printer.Form_Count(len(source)), count, "the print states what it wrote")
	second, again := printer.Print(subject, storage, tree, source)
	testify.True(t, bool(again), "one printer prints one file after another")
	testify.Equal(t, count, second, "the second print writes what the first one did")
}

func test_layout(t *testing.T) {
	test_wide_head(t)
	test_wide_tail(t)
	round_trip(t, "package one\n\nconst LIMIT int = 8\n")
	round_trip(t, "package one\n\nvar table [8]int\n")
	round_trip(t, "package one\n\ntype Count int\n")
	round_trip(t, "package one\n\nfunc Fold() (sum int) {\n\treturn sum\n}\n")
	round_trip(t, "package one\n\nfunc Fold(one int, two int) (sum int) {\n\treturn one\n}\n")
	round_trip(t, "package one\n\nfunc Fold() (sum int) {\n\tif sum > 0 {\n\t\treturn sum\n"+
		"\t}\n\treturn 0\n}\n")
	round_trip(t, "package one\n\nfunc Fold() (sum int) {\n\tfor step := range 4 {\n"+
		"\t\tsum = sum + step\n\t}\n\treturn sum\n}\n")
}

func test_spacing(t *testing.T) {
	// A raw literal holds any byte a file can, thus the print writes the whole byte domain
	// rather than the letters a name spells.
	round_trip(t, "package one\n\nconst TEXT = `\x00\x01\x02\xff`\n")
	for _, one := range []string{
		"\tsum = one || two\n",
		"\tsum = one && two\n",
		"\tsum = one & ^two\n",
		"\tsum = one / *pointer\n",
		"\tsum = one + +two\n",
		"\tsum = one - -two\n",
		"\tsum = one + two\n",
		"\tsum = one - two\n",
		"\tsum = -one\n",
		"\tsum = !held\n",
		"\tsum = one(two, three)\n",
		"\tsum = table[one]\n",
		"\tsum = table[one:two]\n",
		"\tsum = held.Value\n",
		"\tsum = &held\n",
		"\tsum = *pointer\n",
		"\tsum = one == two\n",
		"\tsum = one\n",
		"\tsum++\n",
		"\tsum, held = one, two\n",
	} {
		round_trip(t, "package one\n\nfunc Fold() {\n"+one+"}\n")
	}
}

func test_alignment(t *testing.T) {
	round_trip(t, "package one\n\ntype Entry struct {\n\tValue int\n\tMark  int\n}\n")
	round_trip(t, "package one\n\ntype Entry struct {\n\tOne   int\n\tThree int\n"+
		"\tTwo   int\n}\n")
	round_trip(t, "package one\n\ntype Entry struct {\n\tValue int\n\n\tLonger int\n}\n")
	round_trip(t, "package one\n\ntype Entry struct {\n\tValue int\n}\n")
}

func test_trivia(t *testing.T) {
	round_trip(t, "package one\n\n// Count names a count.\ntype Count int\n")
	round_trip(t, "package one\n\ntype Count int\n\ntype Mark int\n")
	round_trip(t, "package one\n\nfunc Fold() {\n\t// One states nothing.\n\treturn\n}\n")
	round_trip(t, "package one\n\n/* Count names a count. */\ntype Count int\n")
}

func test_refusals(t *testing.T) {
	test_arena(t)
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	source := token.Source("package one\n\ntype Count int\n")
	ast.Parse(tree, source)
	storage := make([]byte, 4)
	count, ok := printer.Print(subject, storage, tree, source)
	testify.False(t, bool(ok), "a print past the storage is refused")
	testify.Equal(t, printer.Form_Count(len(storage)), count,
		"a refused print states the storage it filled")
	// A caller that hands over storage of a few bytes reads the few bytes the print wrote,
	// which is what a caller that only asks whether a file is clean hands over.
	for _, width_size := range []int{0, 1, 2} {
		narrow := make([]byte, width_size)
		written, held := printer.Print(subject, narrow, tree, source)
		testify.False(t, bool(held), "a print past narrow storage is refused")
		testify.Equal(t, printer.Form_Count(width_size), written,
			"a refused print states the narrow storage it filled")
	}
	// A source of a few bytes states no file at all, thus the print writes nothing and says
	// so rather than reading a token the source never held.
	for _, text := range []string{"", "p", "pa"} {
		narrow := token.Source(text)
		ast.Parse(tree, narrow)
		wide_storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
		written, held := printer.Print(subject, wide_storage, tree, narrow)
		testify.True(t, bool(held), "a print of a source of a few bytes holds its form")
		testify.True(t, written <= printer.Form_Count(len(text)+9),
			"a source of a few bytes writes the clause those bytes name and no more")
	}
	broken := token.Source("package one\n\nfunc (\n")
	ast.Parse(tree, broken)
	wide := make([]byte, printer.FORM_SIZE_MAXIMUM)
	_, held := printer.Print(subject, wide, tree, broken)
	testify.True(t, bool(held), "a refused parse prints the tree it holds")
}

// Builds one source of the widest admitted size whose form runs past the widest admitted form,
// which is a body of short statements standing deep inside blocks that indent every one of them.
func wide_source() (source string) {
	body := make([]byte, 0, token.SOURCE_SIZE_MAXIMUM)
	body = append(body, "package one\n\nfunc Fold() {\n"...)
	for range 60 {
		body = append(body, "if true {\n"...)
	}
	// A run holds one token for the name, one for the step, and one for the line it closes,
	// thus forty thousand steps stand well inside the token run one parse admits.
	for range 40000 {
		body = append(body, "x++\n"...)
	}
	for range 60 {
		body = append(body, "}\n"...)
	}
	body = append(body, "}\n"...)
	// An empty line states no token at all, thus the padding that carries the source to the
	// widest admitted size costs the parse nothing.
	for len(body) < token.SOURCE_SIZE_MAXIMUM {
		body = append(body, '\n')
	}
	return string(body)
}

func test_bounds(t *testing.T) {
	testify.Equal(t, 2097152, printer.FORM_SIZE_MAXIMUM, "one print holds two source widths")
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	// A source of the widest admitted size whose form runs past the storage fills the storage
	// and stops there, which is the widest count one print states.
	source := token.Source(wide_source())
	testify.Equal(t, token.SOURCE_SIZE_MAXIMUM, len(source),
		"the source states the widest size")
	ast.Parse(tree, source)
	count, ok := printer.Print(subject, storage, tree, source)
	testify.False(t, bool(ok), "a form past the storage is refused")
	testify.Equal(t, printer.Form_Count(printer.FORM_SIZE_MAXIMUM), count,
		"a refused print states the storage it filled")
	testify.Equal(t, ast.NODE_COUNT_MAXIMUM, printer.DEPTH_MAXIMUM,
		"one print walks every node one parse holds")
	// A run of one sign nests one value inside the next, thus a chain of a thousand values
	// nests a thousand levels deep and every one of them stands inside the walk.
	deep := "package one\n\nconst TEXT = \"one\""
	for range 1024 {
		deep = deep + " + \"one\""
	}
	round_trip(t, deep+"\n")
}

func test_allocation(t *testing.T) {
	held := allocation_fixture{}
	subject := new(printer.Printer)
	tree := new(ast.Parse_State)
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	source := token.Source("package one\n\ntype Count int\n\nfunc Fold() (sum Count) {\n" +
		"\treturn sum\n}\n")
	ast.Parse(tree, source)
	// A file that states every form the printer writes proves the whole of the print
	// allocates nothing, rather than proving it of the few forms one small file holds.
	wide := new(ast.Parse_State)
	wide_form := token.Source(wide_head() + wide_tail())
	ast.Parse(wide, wide_form)
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Print", Call: func() {
			held.Count, held.Ok = printer.Print(subject, storage, tree, source)
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
		"\tLeft  int    `json:\"left\"`\n\tRight string `json:\"right\"`" +
		"\n\n\tNote string\n}\n\n// Reader states one constraint.\n" +
		"type Reader interface {\n\t~int | ~string\n" +
		"}\n\n// Handler names one function type.\n" +
		"type Handler func(one int, two string) (held bool)\n" +
		"\n// Fold folds one pair.\nfunc Fold[Value ~int](pair Pair, valu" +
		"es []Value) (sum int, err error) {\n\tsum = pair.Left*2 + 3\n" +
		"\tsum = sum + len(values)/2 - 1\n\tsum = sum & ^COUNT\n" +
		"\tsum = sum + +COUNT\n\tsum = sum - -COUNT\n" +
		"\tsum = (sum + 1) * 2\n\thead, tail := values[0], values[len(val" +
		"ues)-1]\n\t_ = head\n\t_ = tail\n\tslice := values[1:]\n" +
		"\twhole := values[:]\n\tfront := values[:1]\n" +
		"\tmiddle := values[1 : len(values)-1]\n\t_, _, _, _ = slice, who" +
		"le, front, middle\n\tsum++\n\tsum--\n\tif sum > 0 {\n" +
		"\t\tsum = sum + 1 // The sum grows.\n\t} else if sum == 0 {\n" +
		"\t\tsum = 1 // The sum opens.\n\t} else {\n" +
		"\t\tsum = 0\n\t}\n\tfor step := 0; step < 4; step++ {\n" +
		"\t\tsum = sum + step\n\t}\n\tfor sum > 100 {\n" +
		"\t\tsum = sum - 1\n\t}\n\tfor ; sum < 0; sum++ {\n" +
		"\t}\n\tfor index := range table {\n\t\tsum = sum + index\n" +
		"\t}\n\tfor key, value := range lookup {\n" +
		"\t\tsum = sum + value + len(key)\n\t}\n\tswitch sum {\n" +
		"\tcase 0, 1:\n\t\tsum = 2\n\tcase 2:\n\t\t// A note opens this c" +
		"ase.\n\t\tsum = 3\n\tdefault:\n\t\tsum = 4\n" +
		"\t}\n\tswitch held := any(sum); held.(type) {\n" +
		"\tcase int:\n\t\tsum = 5\n\tdefault:\n\t\tsum = 6\n" +
		"\t}\n\tselect {\n\tcase value := <-receiver:\n" +
		"\t\tsum = value\n\tcase sender <- sum:\n" +
		"\t\tsum = 0\n\tdefault:\n\t\tsum = 1\n\t}\n" +
		"Loop:\n\tfor range 4 {\n\t\tfor range 2 {\n" +
		"\t\t\tbreak Loop\n\t\t}\n\t}\n\tdefer func() { sum = sum + 1 }()" +
		"\n\tgo Report(sum)\n\tReport(sum)\n\tReport(values...)\n" +
		"\tpair.Note = strings.TrimSpace(pair.Right)\n" +
		"\terr = errors.New(\"held\")\n\treturn sum, err\n" +
		"}\n\n// Report states one count.\nfunc Report(values ...any) {\n" +
		"\t_ = values\n}\n\n// Blanked holds one blank line ahead of its " +
		"brace.\ntype Blanked struct {\n\tValue int\n" +
		"}\n\nvar noted func() // A note stands behind this one.\n" +
		"\n// Ranked states one constraint of a term the source approxima" +
		"tes and one it states plainly.\ntype Ranked interface {\n" +
		"\t~int | string\n}\n"
}

// States the bodies of one source that holds every form the printer writes.
func wide_tail() (source string) {
	return "// Package wide states every form the printer writes.\n" +
		"package wide\n\nimport \"errors\"\n\n// Spread states one long c" +
		"all the author broke.\nfunc Spread(\n\tone int, two int, three i" +
		"nt,\n) (held bool, err error) {\n\theld = one+two > three\n" +
		"\tpairs := []Pair{\n\t\t{Left: one, Right: \"one\"}, // The firs" +
		"t pair.\n\t\t{Left: two, Right: \"two\"}, // The second pair.\n" +
		"\t}\n\ttable := map[string]Pair{\n\t\t\"one\": pairs[0],\n" +
		"\n\t\t\"two\": pairs[1],\n\t}\n\t_ = table\n" +
		"\tReport(\n\t\tone,\n\t\ttwo,\n\t\tthree,\n" +
		"\t)\n\tReport(one, two,\n\t\tthree)\n\terr = errors.New(\n" +
		"\t\t\"held\",\n\t)\n\treturn held, err\n" +
		"}\n\n// Empty holds nothing.\ntype Empty struct{}\n" +
		"\n// Wrapper embeds one pair.\ntype Wrapper struct {\n" +
		"\tPair\n\n\tNote string\n}\n\n// Number states one constraint wi" +
		"thout approximation.\ntype Number interface {\n" +
		"\tint | ~string\n}\n\nvar handler func()\n" +
		"\nvar one, two = 1, 2\n\n// Count states one bare result.\n" +
		"func Count() int {\n\treturn one + two\n" +
		"}\n\n// Wide states one signature the author broke behind its fi" +
		"rst parameter.\nfunc Wide(one int,\n\ttwo int) (held bool) {\n" +
		"\theld = one < two\n\tif one <\n\t\ttwo {\n" +
		"\t\theld = false\n\t}\n\tswitch one +\n\t\ttwo {\n" +
		"\tcase 3:\n\tcase 4:\n\t\theld = true\n\t}\n" +
		"\tfor range 2 { // A note stands on the brace line.\n" +
		"\t\theld = !held\n\t}\n\tempty := Empty{}\n" +
		"\t_ = empty\n\tdeep := map[string]Pair{\n" +
		"\t\t\"one\": {\n\t\t\tLeft:  1,\n\t\t\tRight: \"one\",\n" +
		"\t\t},\n\t\t\"two\": {Left: 2, Right: \"two\"},\n" +
		"\t}\n\t_ = deep\n\theld = held ||\n\t\tone > two\n" +
		"\tblock := Wrapper{}\n\n\t_ = block\n\treturn held\n" +
		"}\n\n// Plain states one constraint of a single term.\n" +
		"type Plain interface {\n\tint\n}\n\n// Close states the forms th" +
		"at close a line of their own.\nfunc Close(pair Pair, one int, tw" +
		"o int) (held bool) {\n\theld = one*2 > two\n" +
		"\theld = one * -two\n\tempties := []Empty{{}}\n" +
		"\t_ = empties\n\tif one <\n\t\ttwo {\n\t\theld = true\n" +
		"\t} else if two > one {\n\t\tswitch one {\n" +
		"\t\tcase 5:\n\t\t\theld = false\n\t\t}\n" +
		"\t}\n\theld = strings.\n\t\tContains(pair.Right, \"one\")\n" +
		"\tReport(\n\t\tone,\n\n\t\t// A note stands behind an empty line" +
		".\n\t\ttwo,\n\t)\n\tone++\n\n\treturn held\n" +
		"}\n\n// Trail closes its body on an empty line, which the canoni" +
		"cal form keeps.\nfunc Trail(one int) {\n" +
		"\tone++\n\n}\n\n// Open opens its body on an empty line, which t" +
		"he canonical form keeps.\nfunc Open(one int) {\n" +
		"\n\tone++\n}\n"
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
	tree := new(ast.Parse_State)
	source := token.Source("package one\n\nx y\n")
	tree.Tokens[0] = token.Token{Kind: token.KIND_PACKAGE, Offset: 0, Size: 7}
	tree.Tokens[1] = token.Token{Kind: token.KIND_IDENTIFIER, Offset: 8, Size: 3}
	tree.Tokens[2] = token.Token{Kind: token.KIND_IDENTIFIER, Offset: 13, Size: 1}
	tree.Tokens[3] = token.Token{Kind: token.KIND_IDENTIFIER, Offset: 15, Size: 1}
	tree.Tokens[4] = token.Token{Kind: token.KIND_END_OF_FILE, Offset: 16, Size: 0}
	tree.Token_Cursors[ast.CURSOR_TOKEN_COUNT] = 5
	tree.Nodes[0] = ast.Node{Kind: ast.NODE_ERROR}
	tree.Nodes[1] = ast.Node{Kind: ast.NODE_FILE, Token: 0, First_Child: 2}
	tree.Nodes[2] = ast.Node{Kind: ast.NODE_IDENTIFIER, Token: 1, Parent: 1, Next: 3}
	// The sign of the operation is a name, which no parse writes and every print answers for.
	tree.Nodes[3] = ast.Node{
		Kind: ast.NODE_BINARY, Token: 3, Parent: 1, First_Child: 4, Next: 6,
	}
	tree.Nodes[4] = ast.Node{Kind: ast.NODE_IDENTIFIER, Token: 2, Parent: 3, Next: 5}
	tree.Nodes[5] = ast.Node{Kind: ast.NODE_IDENTIFIER, Token: 3, Parent: 3}
	tree.Nodes[6] = ast.Node{Kind: ast.NODE_CALL, Token: 2, Parent: 1, Next: 7}
	tree.Nodes[7] = ast.Node{Kind: ast.NODE_CALL, Token: 2, Parent: 1, First_Child: 8}
	tree.Nodes[8] = ast.Node{Kind: ast.NODE_COMMENT, Token: 2, Parent: 7}
	// The term states no tilde ahead of it, which no parse writes and every print answers for.
	tree.Nodes[7].Next = 9
	tree.Nodes[9] = ast.Node{Kind: ast.NODE_TERM, Token: 2, Parent: 1, First_Child: 10}
	tree.Nodes[10] = ast.Node{Kind: ast.NODE_IDENTIFIER, Token: 2, Parent: 9}
	tree.Node_Cursors[ast.CURSOR_NODE_COUNT] = 11
	subject := new(printer.Printer)
	storage := make([]byte, printer.FORM_SIZE_MAXIMUM)
	count, ok := printer.Print(subject, storage, tree, source)
	testify.True(t, bool(ok), "a print of a hand written tree holds its form")
	testify.True(t, count > 0, "a print of a hand written tree writes the form it holds")
}
