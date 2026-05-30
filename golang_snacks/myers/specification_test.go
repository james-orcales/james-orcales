package myers_test

import (
	"fmt"
	"slices"
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

// Test_Diff_Single verifies minimal scripts for empty and single-rune inputs.
func Test_Diff_Single(t *testing.T) {
	check := func(old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		if !snap.Snapshot_Is_Equal(snapshot, myers.Differ_Diff(d)) {
			t.Error("Snapshot mismatch")
		}
	}
	check("", "", snap.Init(``))
	check("", "x", snap.Init(`+"x"`))
	check("x", "", snap.Init(`-"x"`))
	check("x", "x", snap.Init(` "x"`))
	check("x", "xx", snap.Init(` "x"+"x"`))
	check("xx", "x", snap.Init(` "x"-"x"`))
	check("xx", "xy", snap.Init(` "x"-"x"+"y"`))
	check("xy", "xx", snap.Init(` "x"-"y"+"x"`))
	check("xx", "yx", snap.Init(`-"x"+"y" "x"`))
	check("yx", "xx", snap.Init(`-"y"+"x" "x"`))
	check("xy", "xz", snap.Init(` "x"-"y"+"z"`))
}

// Test_Diff_Examples verifies sentences and mixed edits clean up to compact runs.
func Test_Diff_Examples(t *testing.T) {
	check := func(old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		if !snap.Snapshot_Is_Equal(snapshot, myers.Differ_Diff(d)) {
			t.Error("Snapshot mismatch")
		}
	}
	check("The dog in the hat.", "The cat in the hat.",
		snap.Init(` "The "-"dog"+"cat" " in the hat."`))
	check("The cat in the hat.", "The furry cat in the hat.",
		snap.Init(` "The "+"furry " "cat in the hat."`))
	check("The cat in the hat.", "The cat.",
		snap.Init(` "The cat"-" in the hat" "."`))
	check("The cat in the hat.", "The happy cat in the black hat.",
		snap.Init(` "The "+"happy " "cat in the"+" black" " hat."`))
	check("The ox in the box.", "The cat in the hat.",
		snap.Init(` "The "-"ox"+"cat" " in the "-"box"+"hat" "."`))
	check("1 two 3 four five 6 seven", "one two three four five six seven",
		snap.Init(`-"1"+"one" " two "-"3"+"three" " four five "-"6"+"six" " seven"`))
	check("XABCY", "XABABCYABY", snap.Init(` "XAB"+"AB" "CY"+"ABY"`))
}

// Test_Line_Diff_Single_Line verifies single-line identity, insert, delete, and
// substitution prefixes.
func Test_Line_Diff_Single_Line(t *testing.T) {
	check := func(old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if !snap.Snapshot_Is_Equal(snapshot, actual) {
			t.Error("Snapshot mismatch")
		}
	}
	check("", "", snap.Init(`
`))
	check("x", "", snap.Init(`
-x`))
	check("", "x", snap.Init(`
+x`))
	check("x", "x", snap.Init(`
 x`))
	check("xx", "x", snap.Init(`
-xx
+x`))
	check("xy", "xx", snap.Init(`
-xy
+xx`))
	check("a\nb\n", "a\nc\n", snap.Init(`
 a
-b
+c
 `))
}

// Test_Line_Diff_Multiline verifies inserting and deleting a line inside a block
// retains the surrounding lines.
func Test_Line_Diff_Multiline(t *testing.T) {
	check := func(old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if !snap.Snapshot_Is_Equal(snapshot, actual) {
			t.Error("Snapshot mismatch")
		}
	}
	check("Line 1\n\nLine 2\nLine 3", "Line 1\n\nLine 1.5\nLine 2\nLine 3", snap.Init(`
 Line 1

+Line 1.5
 Line 2
 Line 3`))
	check("Header\nBody\nFooter", "Header\n\nFooter", snap.Init(`
 Header
-Body
+
 Footer`))
	check("Alpha\nBeta\nGamma", "Alpha\nDelta\nGamma", snap.Init(`
 Alpha
-Beta
+Delta
 Gamma`))
}

// Test_Line_Diff_Code verifies edits to source lines render as paired deletions
// and insertions.
func Test_Line_Diff_Code(t *testing.T) {
	check := func(old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if !snap.Snapshot_Is_Equal(snapshot, actual) {
			t.Error("Snapshot mismatch")
		}
	}
	check("x := 10", "x := 20", snap.Init(`
-x := 10
+x := 20`))
	check("if check { return true }", "if !check { return false }", snap.Init(`
-if check { return true }
+if !check { return false }`))
	check("func old_name() {}", "func new_name() {}", snap.Init(`
-func old_name() {}
+func new_name() {}`))
	check("import \"core:fmt\"", "import \"core:log\"", snap.Init(`
-import "core:fmt"
+import "core:log"`))
}

// Test_Line_Diff_Structured_Data verifies record edits retain unchanged keys and
// mark only altered lines.
func Test_Line_Diff_Structured_Data(t *testing.T) {
	check := func(old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if !snap.Snapshot_Is_Equal(snapshot, actual) {
			t.Error("Snapshot mismatch")
		}
	}
	check(`{"id": 1, "active": true}`, `{"id": 1, "active": false}`, snap.Init(`
-{"id": 1, "active": true}
+{"id": 1, "active": false}`))
	check(`{"items": ["a", "b"]}`, `{"items": ["a", "b", "c"]}`, snap.Init(`
-{"items": ["a", "b"]}
+{"items": ["a", "b", "c"]}`))
	check("name: test\n\nversion: 1.0", "name: production\n\nversion: 1.0", snap.Init(`
-name: test
+name: production

 version: 1.0`))
}

// Test_Line_Diff_Markup verifies nested markup edits retain unchanged tags and
// mark only altered lines.
func Test_Line_Diff_Markup(t *testing.T) {
	check := func(old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
		if !snap.Snapshot_Is_Equal(snapshot, actual) {
			t.Error("Snapshot mismatch")
		}
	}
	check(
		"<ul>\n\t<li>Item A</li>\n\t<li>Item B</li>\n\n</ul>",
		"<ul>\n\t<li>Item A</li>\n\t<li class=\"active\">Item B</li>\n</ul>",
		snap.Init(`
 <ul>
 	<li>Item A</li>
-	<li>Item B</li>
-
+	<li class="active">Item B</li>
 </ul>`))
	check(
		"<div>\n\t<h1>Title</h1>\n\t<p>Paragraph</p>\n</div>",
		"<div>\n\t<h1>Title Updated</h1>\n\t<p>Paragraph</p>\n</div>",
		snap.Init(`
 <div>
-	<h1>Title</h1>
+	<h1>Title Updated</h1>
 	<p>Paragraph</p>
 </div>`))
}

// Test_Line_Diff_License verifies a whole-license rewrite reduces to the same
// block diff a line-based tool produces.
func Test_Line_Diff_License(t *testing.T) {
	old := "MIT License\n\nCopyright (c) 2025 Danzig James Orcales\n\n" +
		"Permission is hereby granted.\n\nThe above notice is included."
	new := "zlib License\n\nCopyright (c) 2025 Danzig James Orcales. All rights reserved.\n\n" +
		"Permission is granted to anyone.\n\nThe origin must not be misrepresented."
	d := myers.New(myers.New_Input{Old: old, New: new})
	actual := fmt.Sprint("\n", myers.Differ_Line_Diff(d))
	snapshot := snap.Init(`
-MIT License
+zlib License

-Copyright (c) 2025 Danzig James Orcales
+Copyright (c) 2025 Danzig James Orcales. All rights reserved.

-Permission is hereby granted.
+Permission is granted to anyone.

-The above notice is included.
+The origin must not be misrepresented.`)
	if !snap.Snapshot_Is_Equal(snapshot, actual) {
		t.Error("Snapshot mismatch")
	}
}

// Test_Algorithm_Diff_Single verifies empty and single-rune inputs short-circuit.
func Test_Algorithm_Diff_Single(t *testing.T) {
	check := func(old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		myers.Differ_Algorithm_Diff(d)
		if !snap.Snapshot_Is_Equal(snapshot, d.String()) {
			t.Error("Snapshot mismatch")
		}
	}
	check("", "", snap.Init(``))
	check("", "x", snap.Init(`+"x"`))
	check("x", "", snap.Init(`-"x"`))
	check("x", "x", snap.Init(` "x"`))
	check("x", "xx", snap.Init(` "x"+"x"`))
	check("xx", "x", snap.Init(` "x"-"x"`))
	check("xy", "xz", snap.Init(` "x"-"y"+"z"`))
	check("xx", "yx", snap.Init(`+"y" "x"-"x"`))
	check("yx", "xx", snap.Init(`-"y" "x"+"x"`))
}

// Test_Algorithm_Diff_Examples verifies sentences produce a minimal rune script.
func Test_Algorithm_Diff_Examples(t *testing.T) {
	check := func(old, new string, snapshot snap.Snapshot) {
		t.Helper()
		d := myers.New(myers.New_Input{Old: old, New: new})
		myers.Differ_Algorithm_Diff(d)
		if !snap.Snapshot_Is_Equal(snapshot, d.String()) {
			t.Error("Snapshot mismatch")
		}
	}
	check("The dog in the hat.", "The cat in the hat.",
		snap.Init(` "The "-"d"-"o"-"g"+"c"+"a"+"t" " in the hat."`))
	check("meee", "eeek", snap.Init(`-"m" "eee"+"k"`))
	check("xyzz", "ikzz", snap.Init(`-"x"-"y"+"i"+"k" "zz"`))
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
		if !on_a {
			t.Error("Result is not a prefix of A")
		}
		on_b := myers.Runes_Have_Prefix(
			myers.Runes_Have_Prefix_Input{String: rb, Expect: actual},
		)
		if !on_b {
			t.Error("Result is not a prefix of B")
		}
	}
	check("", "", snap.Init(``))
	check("a", "", snap.Init(``))
	check("same", "same", snap.Init(`same`))
	check("same", "emas", snap.Init(``))
	check("long", "longer", snap.Init(`long`))
	check("evenx", "eveny", snap.Init(`even`))
	check("á", "a", snap.Init(``))
	check("你好", "你好", snap.Init(`你好`))
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
		if !on_a {
			t.Error("Result is not a suffix of A")
		}
		on_b := myers.Runes_Have_Suffix(
			myers.Runes_Have_Suffix_Input{String: rb, Expect: actual},
		)
		if !on_b {
			t.Error("Result is not a suffix of B")
		}
	}
	check("", "", snap.Init(``))
	check("a", "", snap.Init(``))
	check("same", "same", snap.Init(`same`))
	check("same", "emas", snap.Init(``))
	check("xeven", "zeven", snap.Init(`even`))
	check("evenx", "eveny", snap.Init(``))
	check("á", "a", snap.Init(``))
	check("再见你好", "朋友你好", snap.Init(`你好`))
	check("😀✅", "😄✅", snap.Init(`✅`))
	check("mañana", "ranaña", snap.Init(`a`))
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
	check("a", "a", snap.Init(`a`))
	check("same", "emas", snap.Init(``))
	check("longer", "long", snap.Init(`long`))
	check("mañana", "ranaña", snap.Init(`aña`))
	check("xeven", "zeven", snap.Init(`even`))
	check("xxxeven", "zzzeven", snap.Init(`even`))
	check("xodd", "zodd", snap.Init(`odd`))
	check("evenxxx", "evenyyy", snap.Init(`even`))
	check("你好", "你好", snap.Init(`你好`))
	check("😀✅", "😄✅", snap.Init(`✅`))
	check("aXXXXXXXXXXXXXXXX", "aYYYYYYYYYYYYYYYY", snap.Init(``))
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
