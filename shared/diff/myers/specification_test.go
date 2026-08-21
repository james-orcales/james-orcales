package myers_test

import (
	"testing"

	"local/james-orcales/shared/diff/myers"
	"local/james-orcales/shared/testify"
)

// Test_Diff_Into_Cases covers minimal rune scripts, escaping, bounds, and allocation.
func Test_Diff_Into_Cases(t *testing.T) {
	workspace := maximum_workspace()
	output := make(myers.Output, myers.DIFF_SIZE_MAXIMUM)
	check := func(old string, new_text string, expected string) {
		t.Helper()
		count, status := myers.Diff_Into(myers.Diff_Input{
			Output:    output,
			Workspace: workspace,
			Old:       myers.Old_Text_Unvalidated(old),
			New:       myers.New_Text_Unvalidated(new_text),
		})
		if status != myers.STATUS_OK {
			t.Fatalf("Diff_Into status = %d, want STATUS_OK", status)
		}
		actual := string(output[:count])
		if actual != expected {
			t.Errorf("Diff_Into(%q, %q) = %q, want %q", old, new_text, actual, expected)
		}
	}
	check("", "", "")
	check("abc", "abc", ` "abc"`)
	check("", "abc", `+"abc"`)
	check("abc", "", `-"abc"`)
	check("abc", "axc", ` "a"-"b"+"x" "c"`)
	check("", `a"b`, `+"a\"b"`)
	check("😀", "😄", `-"😀"+"😄"`)

	check_diff_failures(t, workspace, output)
	check_diff_allocation(t)
}

// Test_Line_Diff_Into_Cases covers line scripts, boundaries, and allocation.
func Test_Line_Diff_Into_Cases(t *testing.T) {
	workspace := maximum_workspace()
	output := make(myers.Line_Output, myers.LINE_DIFF_SIZE_MAXIMUM)
	check := func(old string, new_text string, expected string) {
		t.Helper()
		count, status := myers.Line_Diff_Into(myers.Line_Diff_Input{
			Output:    output,
			Workspace: workspace,
			Old:       myers.Line_Old_Text_Unvalidated(old),
			New:       myers.Line_New_Text_Unvalidated(new_text),
		})
		if status != myers.STATUS_OK {
			t.Fatalf("Line_Diff_Into status = %d, want STATUS_OK", status)
		}
		actual := string(output[:count])
		if actual != expected {
			t.Errorf(
				"Line_Diff_Into(%q, %q) = %q, want %q",
				old, new_text, actual, expected,
			)
		}
	}
	check("", "", "")
	check("a", "a", " a")
	check("", "a\nb", "+a\n+b")
	check("", "ab", "+ab")
	check("a\nb", "", "-a\n-b")
	check("a\nb", "a\nc", " a\n-b\n+c")
	check("a\n", "a\n", " a\n ")

	check_line_diff_failures(t, workspace, output)
	check_line_diff_allocation(t)
}

// Test_Diff_Boundaries_Cases proves character storage bounds through the public API.
func Test_Diff_Boundaries_Cases(t *testing.T) {
	text_maximum := string(make([]byte, myers.TEXT_SIZE_MAXIMUM))
	text_rejected := string(make([]byte, myers.TEXT_SIZE_UNVALIDATED_MAXIMUM))
	for _, output_count := range []int{0, 1, 2, myers.DIFF_SIZE_MAXIMUM} {
		output := make(myers.Output, output_count)
		workspace := maximum_workspace()
		myers.Diff_Into(myers.Diff_Input{
			Output: output, Workspace: workspace,
			Old: myers.Old_Text_Unvalidated(text_maximum),
		})
		myers.Diff_Into(myers.Diff_Input{
			Output: output, Workspace: workspace,
			New: myers.New_Text_Unvalidated(text_rejected),
		})
	}
	for _, storage_count := range []int{0, 1, 2, myers.RUNE_STORAGE_COUNT_REQUIRED} {
		workspace := &myers.Workspace{
			Old_Runes: make(myers.Old_Rune_Storage, storage_count),
			New_Runes: make(myers.New_Rune_Storage, storage_count),
			Matrix:    make(myers.Matrix_Storage, storage_count),
		}
		myers.Diff_Into(myers.Diff_Input{
			Output: make(myers.Output, 2), Workspace: workspace, Old: "a", New: "a",
		})
	}
	for _, pair := range []struct{ Old, New_Text string }{
		{"", ""}, {"a", ""}, {"", "a"}, {"ab", "ab"},
	} {
		old_count := len([]rune(pair.Old))
		new_count := len([]rune(pair.New_Text))
		workspace := &myers.Workspace{
			Old_Runes: make(myers.Old_Rune_Storage, old_count),
			New_Runes: make(myers.New_Rune_Storage, new_count),
			Matrix: make(
				myers.Matrix_Storage, (old_count+1)*(new_count+1),
			),
		}
		myers.Diff_Into(myers.Diff_Input{
			Output:    make(myers.Output, myers.DIFF_SIZE_MAXIMUM),
			Workspace: workspace, Old: myers.Old_Text_Unvalidated(pair.Old),
			New: myers.New_Text_Unvalidated(pair.New_Text),
		})
	}
	myers.Diff_Into(myers.Diff_Input{
		Output:    make(myers.Output, myers.DIFF_SIZE_MAXIMUM),
		Workspace: maximum_workspace(),
		Old:       myers.Old_Text_Unvalidated(text_maximum),
		New:       myers.New_Text_Unvalidated(text_maximum),
	})
}

// Test_Line_Diff_Boundaries_Cases proves line storage bounds through the public API.
func Test_Line_Diff_Boundaries_Cases(t *testing.T) {
	text_maximum := string(make([]byte, myers.TEXT_SIZE_MAXIMUM))
	text_rejected := string(make([]byte, myers.TEXT_SIZE_UNVALIDATED_MAXIMUM))
	for _, output_count := range []int{0, 1, 2, myers.LINE_DIFF_SIZE_MAXIMUM} {
		output := make(myers.Line_Output, output_count)
		workspace := maximum_workspace()
		myers.Line_Diff_Into(myers.Line_Diff_Input{
			Output: output, Workspace: workspace,
			New: myers.Line_New_Text_Unvalidated(text_maximum),
		})
		myers.Line_Diff_Into(myers.Line_Diff_Input{
			Output: output, Workspace: workspace,
			Old: myers.Line_Old_Text_Unvalidated(text_rejected),
		})
	}
	maximum_lines := make([]byte, myers.TEXT_SIZE_MAXIMUM)
	for index := range maximum_lines {
		maximum_lines[index] = '\n'
	}
	for _, input := range []myers.Line_Diff_Input{
		{
			Output:    make(myers.Line_Output, myers.LINE_DIFF_SIZE_MAXIMUM),
			Workspace: maximum_workspace(),
			Old:       myers.Line_Old_Text_Unvalidated(maximum_lines),
		},
		{
			Output:    make(myers.Line_Output, myers.LINE_DIFF_SIZE_MAXIMUM),
			Workspace: maximum_workspace(),
			New:       myers.Line_New_Text_Unvalidated(maximum_lines),
		},
	} {
		myers.Line_Diff_Into(input)
	}
	for _, matrix_count := range []int{0, 1, 2, myers.MATRIX_STORAGE_COUNT_REQUIRED} {
		workspace := &myers.Workspace{
			Old_Runes: make(myers.Old_Rune_Storage, 2),
			New_Runes: make(myers.New_Rune_Storage, 2),
			Matrix:    make(myers.Matrix_Storage, matrix_count),
		}
		myers.Line_Diff_Into(myers.Line_Diff_Input{
			Output: make(myers.Line_Output, 2), Workspace: workspace,
			Old: "a", New: "a",
		})
	}
	for _, pair := range []struct{ Old, New_Text string }{
		{"", ""}, {"a", ""}, {"", "a"}, {"a\nb", "a\nb"},
	} {
		old_count := line_count(pair.Old)
		new_count := line_count(pair.New_Text)
		workspace := &myers.Workspace{
			Old_Runes: make(myers.Old_Rune_Storage, old_count),
			New_Runes: make(myers.New_Rune_Storage, new_count),
			Matrix: make(
				myers.Matrix_Storage, (old_count+1)*(new_count+1),
			),
		}
		myers.Line_Diff_Into(myers.Line_Diff_Input{
			Output:    make(myers.Line_Output, myers.LINE_DIFF_SIZE_MAXIMUM),
			Workspace: workspace, Old: myers.Line_Old_Text_Unvalidated(pair.Old),
			New: myers.Line_New_Text_Unvalidated(pair.New_Text),
		})
	}
}

// Test_Find_Common_Prefix_Cases covers empty, partial, and complete prefixes.
func Test_Find_Common_Prefix_Cases(t *testing.T) {
	check := func(left string, right string, expected string) {
		t.Helper()
		actual := myers.Find_Common_Prefix(myers.Find_Common_Prefix_Input{
			Left: []rune(left), Right: []rune(right),
		})
		if string(actual) != expected {
			t.Errorf("Find_Common_Prefix(%q, %q) = %q", left, right, actual)
		}
	}
	check("", "", "")
	check("abc", "xyz", "")
	check("abc", "abd", "ab")
	check("你好", "你们", "你")
	check("a", "a", "a")
	check("ab", "ab", "ab")
	maximum := make([]rune, myers.RUNE_COUNT_MAXIMUM)
	result := myers.Find_Common_Prefix(myers.Find_Common_Prefix_Input{
		Left: maximum, Right: maximum,
	})
	if len(result) != myers.RUNE_COUNT_MAXIMUM {
		t.Errorf("maximum prefix length = %d", len(result))
	}
}

// Test_Find_Common_Suffix_Cases covers empty, partial, and complete suffixes.
func Test_Find_Common_Suffix_Cases(t *testing.T) {
	check := func(left string, right string, expected string) {
		t.Helper()
		actual := myers.Find_Common_Suffix(myers.Find_Common_Suffix_Input{
			Left: []rune(left), Right: []rune(right),
		})
		if string(actual) != expected {
			t.Errorf("Find_Common_Suffix(%q, %q) = %q", left, right, actual)
		}
	}
	check("", "", "")
	check("abc", "xyz", "")
	check("abc", "xbc", "bc")
	check("再见你好", "朋友你好", "你好")
	check("a", "a", "a")
	check("ab", "ab", "ab")
	maximum := make([]rune, myers.RUNE_COUNT_MAXIMUM)
	result := myers.Find_Common_Suffix(myers.Find_Common_Suffix_Input{
		Left: maximum, Right: maximum,
	})
	if len(result) != myers.RUNE_COUNT_MAXIMUM {
		t.Errorf("maximum suffix length = %d", len(result))
	}
}

// Test_Find_Common_Run_Cases covers qualifying and short shared runs.
func Test_Find_Common_Run_Cases(t *testing.T) {
	check := func(left string, right string, expected string) {
		t.Helper()
		actual := myers.Find_Common_Run(myers.Find_Common_Run_Input{
			Left: []rune(left), Right: []rune(right),
		})
		if string(actual) != expected {
			t.Errorf("Find_Common_Run(%q, %q) = %q", left, right, actual)
		}
	}
	check("", "", "")
	check("abcdef", "zzabc", "abc")
	check("abcdef", "zzab", "")
	check("你好世界", "朋友世界", "世界")
	check("a", "a", "a")
	check("ab", "ab", "ab")
	maximum := make([]rune, myers.RUNE_COUNT_MAXIMUM)
	result := myers.Find_Common_Run(myers.Find_Common_Run_Input{
		Left: maximum, Right: maximum,
	})
	if len(result) != myers.RUNE_COUNT_MAXIMUM {
		t.Errorf("maximum run length = %d", len(result))
	}
}

// Test_Runes_Have_Prefix_Predicate covers true and false predicate paths.
func Test_Runes_Have_Prefix_Predicate(t *testing.T) {
	string_runes := []rune("abc")
	if !myers.Runes_Have_Prefix(myers.Runes_Have_Prefix_Input{
		String: string_runes, Expect: []rune("ab"),
	}) {
		t.Error("matching prefix rejected")
	}
	if myers.Runes_Have_Prefix(myers.Runes_Have_Prefix_Input{
		String: string_runes, Expect: nil,
	}) {
		t.Error("empty prefix accepted")
	}
	if myers.Runes_Have_Prefix(myers.Runes_Have_Prefix_Input{
		String: string_runes, Expect: []rune("abcd"),
	}) {
		t.Error("oversized prefix accepted")
	}
	if myers.Runes_Have_Prefix(myers.Runes_Have_Prefix_Input{
		String: string_runes, Expect: []rune("zz"),
	}) {
		t.Error("different prefix accepted")
	}
	for _, count := range []int{0, 1, 2, myers.RUNE_COUNT_MAXIMUM} {
		value := make([]rune, count)
		myers.Runes_Have_Prefix(myers.Runes_Have_Prefix_Input{
			String: value, Expect: value,
		})
	}
}

// Test_Runes_Have_Suffix_Predicate covers true and false predicate paths.
func Test_Runes_Have_Suffix_Predicate(t *testing.T) {
	string_runes := []rune("abc")
	if !myers.Runes_Have_Suffix(myers.Runes_Have_Suffix_Input{
		String: string_runes, Expect: []rune("bc"),
	}) {
		t.Error("matching suffix rejected")
	}
	if myers.Runes_Have_Suffix(myers.Runes_Have_Suffix_Input{
		String: string_runes, Expect: nil,
	}) {
		t.Error("empty suffix accepted")
	}
	if myers.Runes_Have_Suffix(myers.Runes_Have_Suffix_Input{
		String: string_runes, Expect: []rune("abcd"),
	}) {
		t.Error("oversized suffix accepted")
	}
	if myers.Runes_Have_Suffix(myers.Runes_Have_Suffix_Input{
		String: string_runes, Expect: []rune("zz"),
	}) {
		t.Error("different suffix accepted")
	}
	for _, count := range []int{0, 1, 2, myers.RUNE_COUNT_MAXIMUM} {
		value := make([]rune, count)
		myers.Runes_Have_Suffix(myers.Runes_Have_Suffix_Input{
			String: value, Expect: value,
		})
	}
}

func maximum_workspace() (workspace *myers.Workspace) {
	return &myers.Workspace{
		Old_Runes: make(myers.Old_Rune_Storage, myers.RUNE_STORAGE_COUNT_REQUIRED),
		New_Runes: make(myers.New_Rune_Storage, myers.RUNE_STORAGE_COUNT_REQUIRED),
		Matrix:    make(myers.Matrix_Storage, myers.MATRIX_STORAGE_COUNT_REQUIRED),
	}
}

func check_diff_failures(t *testing.T, workspace *myers.Workspace, output myers.Output) {
	t.Helper()
	_, status := myers.Diff_Into(myers.Diff_Input{
		Output: output, Workspace: workspace,
		Old: myers.Old_Text_Unvalidated(make([]byte, myers.TEXT_SIZE_UNVALIDATED_MAXIMUM)),
	})
	if status != myers.STATUS_INPUT_INVALID {
		t.Errorf("oversized Diff_Into status = %d", status)
	}
	_, status = myers.Diff_Into(myers.Diff_Input{
		Output: output,
		Workspace: &myers.Workspace{
			Old_Runes: make(myers.Old_Rune_Storage, 1),
			New_Runes: make(myers.New_Rune_Storage, 1),
			Matrix:    make(myers.Matrix_Storage, 1),
		},
		Old: "ab", New: "cd",
	})
	if status != myers.STATUS_WORKSPACE_TOO_SMALL {
		t.Errorf("small Diff_Into workspace status = %d", status)
	}
	_, status = myers.Diff_Into(myers.Diff_Input{
		Output: nil, Workspace: workspace, Old: "a", New: "b",
	})
	if status != myers.STATUS_OUTPUT_TOO_SMALL {
		t.Errorf("small Diff_Into output status = %d", status)
	}
}

func check_line_diff_failures(
	t *testing.T, workspace *myers.Workspace, output myers.Line_Output,
) {
	t.Helper()
	_, status := myers.Line_Diff_Into(myers.Line_Diff_Input{
		Output: output, Workspace: workspace,
		New: myers.Line_New_Text_Unvalidated(
			make([]byte, myers.TEXT_SIZE_UNVALIDATED_MAXIMUM),
		),
	})
	if status != myers.STATUS_INPUT_INVALID {
		t.Errorf("oversized Line_Diff_Into status = %d", status)
	}
	_, status = myers.Line_Diff_Into(myers.Line_Diff_Input{
		Output: nil, Workspace: workspace, Old: "a", New: "b",
	})
	if status != myers.STATUS_OUTPUT_TOO_SMALL {
		t.Errorf("small Line_Diff_Into output status = %d", status)
	}
}

func check_diff_allocation(t *testing.T) {
	t.Helper()
	workspace := &myers.Workspace{
		Old_Runes: make(myers.Old_Rune_Storage, 3),
		New_Runes: make(myers.New_Rune_Storage, 3),
		Matrix:    make(myers.Matrix_Storage, 16),
	}
	output := make(myers.Output, 32)
	input := myers.Diff_Input{Output: output, Workspace: workspace, Old: "abc", New: "axc"}
	testify.Zero_Allocation(t, func() {
		_, status := myers.Diff_Into(input)
		if status != myers.STATUS_OK {
			panic("Diff_Into rejected")
		}
	})
}

func check_line_diff_allocation(t *testing.T) {
	t.Helper()
	workspace := &myers.Workspace{
		Old_Runes: make(myers.Old_Rune_Storage, 2),
		New_Runes: make(myers.New_Rune_Storage, 2),
		Matrix:    make(myers.Matrix_Storage, 9),
	}
	output := make(myers.Line_Output, 32)
	input := myers.Line_Diff_Input{
		Output: output, Workspace: workspace, Old: "a\nb", New: "a\nc",
	}
	testify.Zero_Allocation(t, func() {
		_, status := myers.Line_Diff_Into(input)
		if status != myers.STATUS_OK {
			panic("Line_Diff_Into rejected")
		}
	})
}

func line_count(text string) (count int) {
	if text == "" {
		return 0
	}
	count = 1
	for index := 0; index < len(text); index++ {
		if text[index] == '\n' {
			count++
		}
	}
	return count
}
