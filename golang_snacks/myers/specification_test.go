package myers_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/james-orcales/james-orcales/golang_snacks/myers"
	snap "github.com/james-orcales/james-orcales/golang_snacks/snap/snap_default"
)

// Test_Edit_Stringer verifies String renders kind prefixes and escapes quotes.
func Test_Edit_Stringer(t *testing.T) {
	d := myers.Differ{Edits: []myers.Edit{
		{Kind: myers.Edit_Retain, Data: []rune(`a"b`)},
		{Kind: myers.Edit_Delete, Data: []rune("x")},
		{Kind: myers.Edit_Insert, Data: []rune("y")},
	}}
	if !snap.Snapshot_Is_Equal(snap.Init(` "a\"b"-"x"+"y"`), d.String()) {
		t.Error("Snapshot mismatch")
	}
}

// Test_Differ_Construction verifies New copies both texts and leaves edits empty.
func Test_Differ_Construction(t *testing.T) {
	d := myers.New(myers.New_Input{Old: "ab", New: "cd"})
	if string(d.Old) != "ab" {
		t.Error("Old runes mismatch")
	}
	if d.Old_String != "ab" {
		t.Error("Old_String mismatch")
	}
	if string(d.New) != "cd" {
		t.Error("New runes mismatch")
	}
	if d.New_String != "cd" {
		t.Error("New_String mismatch")
	}
	if len(d.Edits) != 0 {
		t.Error("Edits should start empty")
	}
}

// Test_Differ_Reset verifies Reset clears the edits and both texts.
func Test_Differ_Reset(t *testing.T) {
	d := myers.New(myers.New_Input{Old: "ab", New: "cd"})
	myers.Differ_Diff(d)
	myers.Differ_Reset(d)
	if len(d.Edits) != 0 {
		t.Error("Edits should be empty")
	}
	if d.Old_String != "" {
		t.Error("Old_String should be empty")
	}
	if len(d.Old) != 0 {
		t.Error("Old runes should be empty")
	}
}

// Test_Diff_Basics verifies minimal scripts for empty and single-rune inputs.
func Test_Diff_Basics(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		if !snap.Snapshot_Is_Equal(snapshot, myers.Differ_Diff(d)) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("EmptyToEmpty", "", "", snap.Init(``))
	check("EmptyToX", "", "x", snap.Init(`+"x"`))
	check("XToEmpty", "x", "", snap.Init(`-"x"`))
	check("XToX", "x", "x", snap.Init(` "x"`))
	check("XToDoubleX", "x", "xx", snap.Init(` "x"+"x"`))
	check("DoubleXToX", "xx", "x", snap.Init(` "x"-"x"`))
	check("DoubleXToXY", "xx", "xy", snap.Init(` "x"-"x"+"y"`))
	check("XYToDoubleX", "xy", "xx", snap.Init(` "x"-"y"+"x"`))
	check("DoubleXToYX", "xx", "yx", snap.Init(`-"x"+"y" "x"`))
	check("YXToDoubleX", "yx", "xx", snap.Init(`-"y"+"x" "x"`))
	check("XYToXZ", "xy", "xz", snap.Init(` "x"-"y"+"z"`))
}

// Test_Diff_Examples verifies sentences and mixed edits, including invalid
// UTF-8, clean up to a compact script.
func Test_Diff_Examples(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		if !snap.Snapshot_Is_Equal(snapshot, myers.Differ_Diff(d)) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("DogToCatInHat", "The dog in the hat.", "The cat in the hat.",
		snap.Init(` "The "-"dog"+"cat" " in the hat."`))
	check("AddFurryToCat", "The cat in the hat.", "The furry cat in the hat.",
		snap.Init(` "The "+"furry " "cat in the hat."`))
	check("TruncateCatSentence", "The cat in the hat.", "The cat.",
		snap.Init(` "The cat"-" in the hat" "."`))
	check("HappyCatInBlackHat", "The cat in the hat.", "The happy cat in the black hat.",
		snap.Init(` "The "+"happy " "cat in the"+" black" " hat."`))
	check("OxToCatInHat", "The ox in the box.", "The cat in the hat.",
		snap.Init(` "The "-"ox"+"cat" " in the "-"box"+"hat" "."`))
	check("ChaseSceneReplacement", "A dog chased a rat across the yard.",
		"The cat chased the mouse through the garden.",
		snap.Init(`-"A"+"The" " "-"dog"+"cat" " chased "-"a ra" "t"+"he" " "-"acr"+"m" "o"+"u" "s"-"s"+"e through" " the "-"y"+"g" "ard"+"en" "."`))
	check("NumberWordSubstitution", "1 two 3 four five 6 seven",
		"one two three four five six seven",
		snap.Init(`-"1"+"one" " two "-"3"+"three" " four five "-"6"+"six" " seven"`))
	check("ComplexMixedReplacement",
		"AAAAAA111BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBzyyyjCCCCCCCC333DDDDDD",
		"AAAAAAxxxBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBxyyyiCCCCCCCCzzzDDDDDD",
		snap.Init(` "AAAAAA"-"111"+"xxx" "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"-"z"+"x" "yyy"-"j"+"i" "CCCCCCCC"-"333"+"zzz" "DDDDDD"`))
	check("LoremIpsumExpansion",
		"Lorem ipsum dolor sit amet, with some extra words added here, consectetur elit.",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		snap.Init(` "Lorem ipsum dolor sit amet, "-"with some extra words added here, " "consectetur"+" adipiscing" " elit."`))
	check("InvalidUTF8OldToValidNew", "\xff", "a", snap.Init(`-"�"+"a"`))
	check("InsertStartsWithNextRetain", "XABCY", "XABABCYABY",
		snap.Init(` "XAB"+"AB" "CY"+"ABY"`))
}

// Test_Line_Diff_Basics verifies single-line identity, insert, delete, and
// substitution prefixes.
func Test_Line_Diff_Basics(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if strip_line_ends(actual) != strip_line_ends(snapshot.Expected_Output) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("EmptyToEmpty", "", "", snap.Init(`
`))
	check("EmptyToDeletion", "x", "", snap.Init(`
-x`))
	check("EmptyToInsertion", "", "x", snap.Init(`
+x`))
	check("IdentitySingleCharacter", "x", "x", snap.Init(`
 x`))
	check("DeleteDoubleInsertSingle", "xx", "x", snap.Init(`
-xx
+x`))
	check("DeleteSingleInsertDouble", "x", "xx", snap.Init(`
-x
+xx`))
	check("ReplaceXYWithXX", "xy", "xx", snap.Init(`
-xy
+xx`))
	check("ReplaceXXWithXY", "xx", "xy", snap.Init(`
-xx
+xy`))
	check("ReplaceYXWithXX", "yx", "xx", snap.Init(`
-yx
+xx`))
	check("ReplaceXXWithYX", "xx", "yx", snap.Init(`
-xx
+yx`))
	check("ReplaceXYWithXZ", "xy", "xz", snap.Init(`
-xy
+xz`))
	check("BothWithTrailingNewline", "a\nb\n", "a\nc\n", snap.Init(`
 a
-b
+c
 `))
}

// Test_Line_Diff_Blocks verifies inserting, deleting, or replacing a line inside
// a block retains the surrounding lines.
func Test_Line_Diff_Blocks(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if strip_line_ends(actual) != strip_line_ends(snapshot.Expected_Output) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("InsertMiddleLine", "Line 1\n\nLine 2\nLine 3",
		"Line 1\n\nLine 1.5\nLine 2\nLine 3", snap.Init(`
 Line 1

+Line 1.5
 Line 2
 Line 3`))
	check("DeleteMiddleLine", "Header\nBody\nFooter", "Header\n\nFooter", snap.Init(`
 Header
-Body
+
 Footer`))
	check("ReplaceMiddleLine", "Alpha\nBeta\nGamma", "Alpha\nDelta\nGamma", snap.Init(`
 Alpha
-Beta
+Delta
 Gamma`))
}

// Test_Line_Diff_Code verifies edits to source lines render as paired deletions
// and insertions.
func Test_Line_Diff_Code(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if strip_line_ends(actual) != strip_line_ends(snapshot.Expected_Output) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("ReplaceIntegerLiteral", "x := 10", "x := 20", snap.Init(`
-x := 10
+x := 20`))
	check("InvertConditionAndReturn", "if check { return true }",
		"if !check { return false }", snap.Init(`
-if check { return true }
+if !check { return false }`))
	check("RenameFunction", "func old_name() {}", "func new_name() {}", snap.Init(`
-func old_name() {}
+func new_name() {}`))
	check("UpdateCommentStatus", "// TODO: Fix this", "// FIXED: Fixed this", snap.Init(`
-// TODO: Fix this
+// FIXED: Fixed this`))
	check("ChangeImportedModule", "import \"core:fmt\"", "import \"core:log\"", snap.Init(`
-import "core:fmt"
+import "core:log"`))
}

// Test_Line_Diff_Records verifies JSON and YAML record edits retain unchanged
// keys and blank separators.
func Test_Line_Diff_Records(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if strip_line_ends(actual) != strip_line_ends(snapshot.Expected_Output) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("ToggleBooleanField", `{"id": 1, "active": true}`,
		`{"id": 1, "active": false}`, snap.Init(`
-{"id": 1, "active": true}
+{"id": 1, "active": false}`))
	check("AppendArrayElement", `{"items": ["a", "b"]}`,
		`{"items": ["a", "b", "c"]}`, snap.Init(`
-{"items": ["a", "b"]}
+{"items": ["a", "b", "c"]}`))
	check("RenameEnvironment", "name: test\n\nversion: 1.0",
		"name: production\n\nversion: 1.0", snap.Init(`
-name: test
+name: production

 version: 1.0`))
	check("ModifySingleLineInBlock", "Line A\n\nLine B\nLine C\nLine D\nLine E",
		"Line A\n\nLine B\nLine C Modified\nLine D\nLine E", snap.Init(`
 Line A

 Line B
-Line C
+Line C Modified
 Line D
 Line E`))
}

// Test_Line_Diff_Markup verifies nested markup edits retain unchanged tags and
// mark only altered lines.
func Test_Line_Diff_Markup(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if strip_line_ends(actual) != strip_line_ends(snapshot.Expected_Output) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("ModifyHtmlListItemAttribute",
		"<ul>\n\t<li>Item A</li>\n\t<li>Item B</li>\n\n</ul>",
		"<ul>\n\t<li>Item A</li>\n\t<li class=\"active\">Item B</li>\n</ul>", snap.Init(`
 <ul>
 	<li>Item A</li>
-	<li>Item B</li>
-
+	<li class="active">Item B</li>
 </ul>`))
	check("UpdateHtmlTitleText",
		"<div>\n\t<h1>Title</h1>\n\t<p>Paragraph</p>\n\t<footer>Footer</footer>\n</div>",
		"<div>\n\t<h1>Title Updated</h1>\n\t<p>Paragraph</p>\n"+
			"\t<footer>Footer</footer>\n</div>",
		snap.Init(`
 <div>
-	<h1>Title</h1>
+	<h1>Title Updated</h1>
 	<p>Paragraph</p>
 	<footer>Footer</footer>
 </div>`))
}

// Test_Line_Diff_Document verifies a multi-record YAML document with reorderings
// retains the stable keys.
func Test_Line_Diff_Document(t *testing.T) {
	old := "users:\n\n- name: Alice\n  age: 30\n  role: admin\n\n" +
		"- name: Bob\n  age: 25\n  role: user\n\n" +
		"- name: Princess\n  age: 1\n  role: pet"
	new := "users:\n\n- name: Alice\n  age: 31\n  role: admin\n\n" +
		"- name: Bob\n  age: 25\n  role: moderator,\n\n" +
		"- name: Melody\n  age: 2\n  role: pet"
	d := myers.New(myers.New_Input{Old: old, New: new})
	actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
	snapshot := snap.Init(`
 users:

 - name: Alice
-  age: 30
+  age: 31
   role: admin

 - name: Bob
   age: 25
-  role: user
+  role: moderator,

-- name: Princess
-  age: 1
+- name: Melody
+  age: 2
   role: pet`)
	if strip_line_ends(actual) != strip_line_ends(snapshot.Expected_Output) {
		t.Error("Snapshot mismatch")
	}
}

// Test_Line_Diff_Source verifies multi-line function bodies and try blocks diff
// line-for-line.
func Test_Line_Diff_Source(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if strip_line_ends(actual) != strip_line_ends(snapshot.Expected_Output) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("ExpandFunctionBody",
		"function add(a, b) {\n    return a + b\n}",
		"function add(a, b) {\n    let sum = a + b\n    return sum\n}", snap.Init(`
 function add(a, b) {
-    return a + b
+    let sum = a + b
+    return sum
 }`))
	check("UpdateTryExceptLogic",
		"try:\n\tdo_something()\n\nexcept Exception as e:\n\tlog(e)\n\n"+
			"finally:\n\tcleanup()",
		"try:\n\tdo_something_critical()\n\nexcept Exception as e:\n\tlog_error(e)\n\n"+
			"finally:\n\tleak()",
		snap.Init(`
 try:
-	do_something()
+	do_something_critical()

 except Exception as e:
-	log(e)
+	log_error(e)

 finally:
-	cleanup()
+	leak()`))
}

// Test_Line_Diff_Query verifies a multi-clause SQL statement diffs line-by-line.
func Test_Line_Diff_Query(t *testing.T) {
	old := "SELECT id, name, email\n\nFROM users\nWHERE active = 1\nORDER BY name\nLIMIT 10;"
	new := "SELECT id, name, email\n\nFROM users\nWHERE active = 1\n" +
		"ORDER BY created_at DESC\nLIMIT 20;"
	d := myers.New(myers.New_Input{Old: old, New: new})
	actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
	snapshot := snap.Init(`
 SELECT id, name, email

 FROM users
 WHERE active = 1
-ORDER BY name
-LIMIT 10;
+ORDER BY created_at DESC
+LIMIT 20;`)
	if strip_line_ends(actual) != strip_line_ends(snapshot.Expected_Output) {
		t.Error("Snapshot mismatch")
	}
}

// Test_Line_Diff_License verifies a multi-paragraph license rewrite reduces to
// the same block diff a line-based tool produces.
func Test_Line_Diff_License(t *testing.T) {
	old := `MIT License

Copyright (c) 2025 Danzig James Orcales

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software, to deal in the Software without restriction.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND.`
	new := `zlib License

Copyright (c) 2025 Danzig James Orcales. All rights reserved.

This software is provided 'as-is', without any express or implied warranty.

Permission is granted to anyone to use this software for any purpose.`
	d := myers.New(myers.New_Input{Old: old, New: new})
	actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
	// This is the exact same diff as git!
	snapshot := snap.Init(`
-MIT License
+zlib License

-Copyright (c) 2025 Danzig James Orcales
+Copyright (c) 2025 Danzig James Orcales. All rights reserved.

-Permission is hereby granted, free of charge, to any person obtaining a copy
-of this software, to deal in the Software without restriction.
+This software is provided 'as-is', without any express or implied warranty.

-THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND.
+Permission is granted to anyone to use this software for any purpose.`)
	if strip_line_ends(actual) != strip_line_ends(snapshot.Expected_Output) {
		t.Error("Snapshot mismatch")
	}
}

// Test_Algorithm_Diff_Basics verifies empty and single-rune inputs short-circuit.
func Test_Algorithm_Diff_Basics(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		myers.Differ_Algorithm_Diff(d)
		if !snap.Snapshot_Is_Equal(snapshot, d.String()) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("EmptyToEmpty", "", "", snap.Init(``))
	check("EmptyToX", "", "x", snap.Init(`+"x"`))
	check("XToEmpty", "x", "", snap.Init(`-"x"`))
	check("XToX", "x", "x", snap.Init(` "x"`))
	check("XToDoubleX", "x", "xx", snap.Init(` "x"+"x"`))
	check("DoubleXToX", "xx", "x", snap.Init(` "x"-"x"`))
	check("DoubleXToXY", "xx", "xy", snap.Init(` "x"-"x"+"y"`))
	check("XYToDoubleX", "xy", "xx", snap.Init(` "x"-"y"+"x"`))
	check("DoubleXToYX", "xx", "yx", snap.Init(`+"y" "x"-"x"`))
	check("YXToDoubleX", "yx", "xx", snap.Init(`-"y" "x"+"x"`))
	check("XYToXZ", "xy", "xz", snap.Init(` "x"-"y"+"z"`))
}

// Test_Algorithm_Diff_Blog verifies the blog-post sentences diff to a minimal
// single-rune script.
func Test_Algorithm_Diff_Blog(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		myers.Differ_Algorithm_Diff(d)
		if !snap.Snapshot_Is_Equal(snapshot, d.String()) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("DogToCatInHat", "The dog in the hat.", "The cat in the hat.",
		snap.Init(` "The "-"d"-"o"-"g"+"c"+"a"+"t" " in the hat."`))
	check("AddFurryToCat", "The cat in the hat.", "The furry cat in the hat.",
		snap.Init(` "The "+"f"+"u"+"r"+"r"+"y"+" " "cat in the hat."`))
	check("TruncateCatSentence", "The cat in the hat.", "The cat.",
		snap.Init(` "The cat"-" "-"i"-"n"-" "-"t"-"h"-"e"-" "-"h"-"a"-"t" "."`))
	check("HappyCatInBlackHat", "The cat in the hat.", "The happy cat in the black hat.",
		snap.Init(` "The "+"h"+"a"+"p"+"p"+"y"+" " "cat in the "+"b"+"l"+"a"+"c"+"k"+" " "hat."`))
	check("OxToCatInHat", "The ox in the box.", "The cat in the hat.",
		snap.Init(` "The "-"o"-"x"+"c"+"a"+"t" " in the "-"b"-"o"-"x"+"h"+"a"+"t" "."`))
}

// Test_Algorithm_Diff_Custom verifies adversarial mixed edits diff to a minimal
// raw single-rune script.
func Test_Algorithm_Diff_Custom(t *testing.T) {
	check := func(name, old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		myers.Differ_Algorithm_Diff(d)
		if !snap.Snapshot_Is_Equal(snapshot, d.String()) {
			t.Errorf("%s: snapshot mismatch", name)
		}
	}
	check("Meee", "meee", "eeek", snap.Init(`-"m" "eee"+"k"`))
	check("Xyzz", "xyzz", "ikzz", snap.Init(`-"x"-"y"+"i"+"k" "zz"`))
	check("ChaseSceneReplacement", "A dog chased a rat across the yard.",
		"The cat chased the mouse through the garden.",
		snap.Init(`-"A"+"T"+"h"+"e" " "-"d"-"o"-"g"+"c"+"a"+"t" " chased "-"a"-" "-"r"-"a" "t"+"h"+"e" " "-"a"-"c"-"r"+"m" "o"+"u" "s"-"s"+"e" " th"+"r"+"o"+"u"+"g"+"h"+" "+"t"+"h" "e "-"y"+"g" "ard"+"e"+"n" "."`))
	check("NumberWordSubstitution", "1 two 3 four five 6 seven",
		"one two three four five six seven",
		snap.Init(`-"1"+"o"+"n"+"e" " two "-"3"+"t"+"h"+"r"+"e"+"e" " four five "-"6"+"s"+"i"+"x" " seven"`))
	check("ComplexMixedReplacement",
		"AAAAAA111BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBzyyyjCCCCCCCC333DDDDDD",
		"AAAAAAxxxBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBxyyyiCCCCCCCCzzzDDDDDD",
		snap.Init(` "AAAAAA"-"1"-"1"-"1"+"x"+"x"+"x" "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"-"z"+"x" "yyy"-"j"+"i" "CCCCCCCC"-"3"-"3"-"3"+"z"+"z"+"z" "DDDDDD"`))
	check("LoremIpsumExpansion",
		"Lorem ipsum dolor sit amet, with some extra words added here, consectetur elit.",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		snap.Init(` "Lorem ipsum dolor sit amet, "-"w"-"i"-"t"-"h"-" "-"s"-"o"-"m"-"e"-" "-"e"-"x"-"t"-"r"-"a"-" "-"w"-"o"-"r"-"d"-"s"-" "-"a"-"d"-"d"-"e"-"d"-" "-"h"-"e"-"r"-"e"-","-" " "consectetur "+"a"+"d"+"i"+"p"+"i"+"s"+"c"+"i"+"n"+"g"+" " "elit."`))
}

// Test_Find_Common_Prefix_Cases verifies the prefix is order independent and a
// genuine prefix of both inputs across rune families.
func Test_Find_Common_Prefix_Cases(t *testing.T) {
	check := func(a, b string, snapshot snap.Snapshot) {
		t.Helper()
		ra, rb := []rune(a), []rune(b)
		actual := myers.Find_Common_Prefix(myers.Find_Common_Prefix_Input{A: ra, B: rb})
		mirror := myers.Find_Common_Prefix(myers.Find_Common_Prefix_Input{A: rb, B: ra})
		if !snap.Snapshot_Is_Equal(snapshot, string(actual)) {
			t.Error("Snapshot mismatch")
		}
		if !slices.Equal(actual, mirror) {
			t.Error("Prefix changes with flipped arguments")
		}
		if len(actual) == 0 {
			return
		}
		on_a := myers.Runes_Have_Prefix(
			myers.Runes_Have_Prefix_Input{String: ra, Expect: actual},
		)
		on_b := myers.Runes_Have_Prefix(
			myers.Runes_Have_Prefix_Input{String: rb, Expect: actual},
		)
		if !on_a {
			t.Error("Result is not a prefix of A")
		}
		if !on_b {
			t.Error("Result is not a prefix of B")
		}
	}
	check("", "", snap.Init(``))
	check("a", "", snap.Init(``))
	check("", "a", snap.Init(``))
	check("a", "a", snap.Init(`a`))
	check("same", "same", snap.Init(`same`))
	check("same", "emas", snap.Init(``))
	check("long", "longer", snap.Init(`long`))
	check("longer", "long", snap.Init(`long`))
	check("mañana", "ranaña", snap.Init(``))
	check("xeven", "zeven", snap.Init(``))
	check("evenx", "eveny", snap.Init(`even`))
	check("oddx", "oddy", snap.Init(`odd`))
	check("cat in the hat.", "furry cat in the hat.", snap.Init(``))
	check("á", "á", snap.Init(`á`))
	check("á", "a", snap.Init(``))
	check("你好", "你好", snap.Init(`你好`))
	check("再见你好", "朋友你好", snap.Init(``))
	check("😀", "😀", snap.Init(`😀`))
	check("😀✅", "😄✅", snap.Init(``))
	check("𝔘𝔫𝔦𝔠𝔬𝔡𝔢", "𝔘𝔫𝔦𝔠𝔬𝔡𝔢", snap.Init(`𝔘𝔫𝔦𝔠𝔬𝔡𝔢`))
}

// Test_Find_Common_Suffix_Cases verifies the suffix is order independent and a
// genuine suffix of both inputs across rune families.
func Test_Find_Common_Suffix_Cases(t *testing.T) {
	check := func(a, b string, snapshot snap.Snapshot) {
		t.Helper()
		ra, rb := []rune(a), []rune(b)
		actual := myers.Find_Common_Suffix(myers.Find_Common_Suffix_Input{A: ra, B: rb})
		mirror := myers.Find_Common_Suffix(myers.Find_Common_Suffix_Input{A: rb, B: ra})
		if !snap.Snapshot_Is_Equal(snapshot, string(actual)) {
			t.Error("Snapshot mismatch")
		}
		if !slices.Equal(actual, mirror) {
			t.Error("Suffix changes with flipped arguments")
		}
		if len(actual) == 0 {
			return
		}
		on_a := myers.Runes_Have_Suffix(
			myers.Runes_Have_Suffix_Input{String: ra, Expect: actual},
		)
		on_b := myers.Runes_Have_Suffix(
			myers.Runes_Have_Suffix_Input{String: rb, Expect: actual},
		)
		if !on_a {
			t.Error("Result is not a suffix of A")
		}
		if !on_b {
			t.Error("Result is not a suffix of B")
		}
	}
	check("", "", snap.Init(``))
	check("a", "", snap.Init(``))
	check("a", "a", snap.Init(`a`))
	check("same", "same", snap.Init(`same`))
	check("same", "emas", snap.Init(``))
	check("long", "longer", snap.Init(``))
	check("xeven", "zeven", snap.Init(`even`))
	check("evenx", "eveny", snap.Init(``))
	check("oddx", "oddy", snap.Init(``))
	check("cat in the hat.", "furry cat in the hat.", snap.Init(`cat in the hat.`))
	check("á", "á", snap.Init(`á`))
	check("á", "a", snap.Init(``))
	check("你好", "你好", snap.Init(`你好`))
	check("再见你好", "朋友你好", snap.Init(`你好`))
	check("😀", "😀", snap.Init(`😀`))
	check("😀✅", "😄✅", snap.Init(`✅`))
	check("mañana", "ranaña", snap.Init(`a`))
	check("𝔘𝔫𝔦𝔠𝔬𝔡𝔢", "𝔘𝔫𝔦𝔠𝔬𝔡𝔢", snap.Init(`𝔘𝔫𝔦𝔠𝔬𝔡𝔢`))
}

// Test_Find_Common_Run_Cases verifies centered, prefix, and suffix overlaps are
// found above the half-length threshold and a too-short overlap yields nil.
func Test_Find_Common_Run_Cases(t *testing.T) {
	check := func(a, b string, snapshot snap.Snapshot) {
		t.Helper()
		actual := myers.Find_Common_Run(
			myers.Find_Common_Run_Input{A: []rune(a), B: []rune(b)},
		)
		if !snap.Snapshot_Is_Equal(snapshot, string(actual)) {
			t.Error("Snapshot mismatch")
		}
	}
	check("", "", snap.Init(``))
	check("a", "", snap.Init(``))
	check("", "a", snap.Init(``))
	check("a", "a", snap.Init(`a`))
	check("same", "same", snap.Init(`same`))
	check("same", "emas", snap.Init(``))
	check("long", "longer", snap.Init(`long`))
	check("longer", "long", snap.Init(`long`))
	check("substr_is_too_smallzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		"substr_is_too_smalljjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjjj", snap.Init(``))
	check("mañana", "ranaña", snap.Init(`aña`))
	check("ranaña", "mañana", snap.Init(`ana`))
	check("xeven", "zeven", snap.Init(`even`))
	check("xxeven", "zzeven", snap.Init(`even`))
	check("xxxeven", "zzzeven", snap.Init(`even`))
	check("xodd", "zodd", snap.Init(`odd`))
	check("xxodd", "zzodd", snap.Init(`odd`))
	check("xxxodd", "zzzodd", snap.Init(`odd`))
	check("evenx", "eveny", snap.Init(`even`))
	check("oddx", "oddy", snap.Init(`odd`))
	check("cat in the hat.", "furry cat in the hat.", snap.Init(`cat in the hat.`))
	check("á", "á", snap.Init(`á`))
	check("á", "a", snap.Init(``))
	check("你好", "你好", snap.Init(`你好`))
	check("再见你好", "朋友你好", snap.Init(`你好`))
	check("😀", "😀", snap.Init(`😀`))
	check("😀✅", "😄✅", snap.Init(`✅`))
	check("𝔘𝔫𝔦𝔠𝔬𝔡𝔢", "𝔘𝔫𝔦𝔠𝔬𝔡𝔢", snap.Init(`𝔘𝔫𝔦𝔠𝔬𝔡𝔢`))
}

// Test_Runes_Have_Prefix_Predicate verifies the empty and over-long edge cases
// report false.
func Test_Runes_Have_Prefix_Predicate(t *testing.T) {
	check := func(str, expect string, want bool) {
		t.Helper()
		got := myers.Runes_Have_Prefix(myers.Runes_Have_Prefix_Input{
			String: []rune(str), Expect: []rune(expect),
		})
		if got != want {
			t.Errorf("Runes_Have_Prefix(%q, %q) = %v", str, expect, got)
		}
	}
	check("abc", "ab", true)
	check("abc", "abc", true)
	check("abc", "bc", false)
	check("", "a", false)
	check("a", "", false)
	check("ab", "abc", false)
}

// Test_Runes_Have_Suffix_Predicate verifies the empty and over-long edge cases
// report false.
func Test_Runes_Have_Suffix_Predicate(t *testing.T) {
	check := func(str, expect string, want bool) {
		t.Helper()
		got := myers.Runes_Have_Suffix(myers.Runes_Have_Suffix_Input{
			String: []rune(str), Expect: []rune(expect),
		})
		if got != want {
			t.Errorf("Runes_Have_Suffix(%q, %q) = %v", str, expect, got)
		}
	}
	check("abc", "bc", true)
	check("abc", "abc", true)
	check("abc", "ab", false)
	check("", "a", false)
	check("a", "", false)
	check("ab", "abc", false)
}

// Trims trailing spaces from each line. A retained blank line in a line diff
// renders as a single space, but the toolchain strips trailing whitespace from
// source files, so the snapshot literal cannot hold that space; trimming both
// sides keeps the comparison exact for every other line.
func strip_line_ends(text string) (trimmed string) {
	lines := strings.Split(text, "\n")
	for i_index := range lines {
		lines[i_index] = strings.TrimRight(lines[i_index], " ")
	}
	return strings.Join(lines, "\n")
}
