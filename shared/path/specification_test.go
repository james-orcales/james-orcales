package path_test

import (
	"runtime"
	"testing"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/path"
	"local/james-orcales/shared/testify"
)

// Test_Clean binds Clean specification leaf before fixture declarations.
func Test_Clean(t *testing.T) {
	test_clean(t)
}

// Test_Components binds Components specification leaf before fixture declarations.
func Test_Components(t *testing.T) {
	test_components(t)
}

// Test_Join binds Join specification leaf before fixture declarations.
func Test_Join(t *testing.T) {
	test_join(t)
}

// Test_Match binds Match specification leaf before fixture declarations.
func Test_Match(t *testing.T) {
	test_match(t)
}

// Test_Bounds binds Bounds specification leaf before fixture declarations.
func Test_Bounds(t *testing.T) {
	test_bounds(t)
}

// Test_Allocation binds Allocation specification leaf before fixture declarations.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

const TEST_STORAGE_SIZE = bytes.SLICE_SIZE_MAXIMUM

const TEST_COMPONENT_COUNT = 2

type clean_case struct {
	Input path.Text
	Want  string
}

type split_case struct {
	Input     path.Text
	Directory string
	File      string
}

type join_case struct {
	Elements path.Elements
	Want     string
}

type match_case struct {
	Pattern path.Text
	Name    path.Text
	Matched bool
	Bad     bool
}

type clean_case_set []clean_case

type split_case_set []split_case

type join_case_set []join_case

type match_case_set []match_case

func clean_cases() (cases clean_case_set) {
	return clean_case_set{
		{Input: "", Want: "."},
		{Input: "abc", Want: "abc"},
		{Input: "abc/def", Want: "abc/def"},
		{Input: "a/b/c", Want: "a/b/c"},
		{Input: ".", Want: "."},
		{Input: "..", Want: ".."},
		{Input: "../..", Want: "../.."},
		{Input: "../../abc", Want: "../../abc"},
		{Input: "/abc", Want: "/abc"},
		{Input: "/", Want: "/"},
		{Input: "abc/", Want: "abc"},
		{Input: "abc/def/", Want: "abc/def"},
		{Input: "a/b/c/", Want: "a/b/c"},
		{Input: "./", Want: "."},
		{Input: "../", Want: ".."},
		{Input: "../../", Want: "../.."},
		{Input: "/abc/", Want: "/abc"},
		{Input: "abc//def//ghi", Want: "abc/def/ghi"},
		{Input: "//abc", Want: "/abc"},
		{Input: "///abc", Want: "/abc"},
		{Input: "//abc//", Want: "/abc"},
		{Input: "abc//", Want: "abc"},
		{Input: "abc/./def", Want: "abc/def"},
		{Input: "/./abc/def", Want: "/abc/def"},
		{Input: "abc/.", Want: "abc"},
		{Input: "abc/def/ghi/../jkl", Want: "abc/def/jkl"},
		{Input: "abc/def/../ghi/../jkl", Want: "abc/jkl"},
		{Input: "abc/def/..", Want: "abc"},
		{Input: "abc/def/../..", Want: "."},
		{Input: "/abc/def/../..", Want: "/"},
		{Input: "abc/def/../../..", Want: ".."},
		{Input: "/abc/def/../../..", Want: "/"},
		{Input: "abc/def/../../../ghi/jkl/../../../mno", Want: "../../mno"},
		{Input: "abc/./../def", Want: "def"},
		{Input: "abc//./../def", Want: "def"},
		{Input: "abc/../../././../def", Want: "../../def"},
	}
}

func split_cases() (cases split_case_set) {
	return split_case_set{
		{Input: "a/b", Directory: "a/", File: "b"},
		{Input: "a/b/", Directory: "a/b/", File: ""},
		{Input: "a/", Directory: "a/", File: ""},
		{Input: "a", Directory: "", File: "a"},
		{Input: "/", Directory: "/", File: ""},
	}
}

func join_cases() (cases join_case_set) {
	return join_case_set{
		{Elements: path.Elements{}, Want: ""},
		{Elements: path.Elements{""}, Want: ""},
		{Elements: path.Elements{"a"}, Want: "a"},
		{Elements: path.Elements{"a", "b"}, Want: "a/b"},
		{Elements: path.Elements{"a", ""}, Want: "a"},
		{Elements: path.Elements{"", "b"}, Want: "b"},
		{Elements: path.Elements{"/", "a"}, Want: "/a"},
		{Elements: path.Elements{"/", ""}, Want: "/"},
		{Elements: path.Elements{"a/", "b"}, Want: "a/b"},
		{Elements: path.Elements{"a/", ""}, Want: "a"},
		{Elements: path.Elements{"", ""}, Want: ""},
		{Elements: path.Elements{"a/b", "../../../xyz"}, Want: "../xyz"},
	}
}

func match_cases_one() (cases match_case_set) {
	return match_case_set{
		{Pattern: "", Name: "", Matched: true},
		{Pattern: "", Name: "a"},
		{Pattern: "abc", Name: "abc", Matched: true},
		{Pattern: "*", Name: "abc", Matched: true},
		{Pattern: "*c", Name: "abc", Matched: true},
		{Pattern: "a*", Name: "a", Matched: true},
		{Pattern: "a*", Name: "abc", Matched: true},
		{Pattern: "a*", Name: "ab/c"},
		{Pattern: "a*/b", Name: "abc/b", Matched: true},
		{Pattern: "a*/b", Name: "a/c/b"},
		{Pattern: "a*b*c*d*e*/f", Name: "axbxcxdxe/f", Matched: true},
		{Pattern: "a*b*c*d*e*/f", Name: "axbxcxdxexxx/f", Matched: true},
		{Pattern: "a*b*c*d*e*/f", Name: "axbxcxdxe/xxx/f"},
		{Pattern: "a*b*c*d*e*/f", Name: "axbxcxdxexxx/fff"},
		{Pattern: "a*b?c*x", Name: "abxbbxdbxebxczzx", Matched: true},
		{Pattern: "a*b?c*x", Name: "abxbbxdbxebxczzy"},
		{Pattern: "ab[c]", Name: "abc", Matched: true},
		{Pattern: "ab[b-d]", Name: "abc", Matched: true},
		{Pattern: "ab[e-g]", Name: "abc"},
		{Pattern: "[a]??", Name: "abc", Matched: true},
		{Pattern: "ab[^c]", Name: "abc"},
		{Pattern: "ab[^b-d]", Name: "abc"},
		{Pattern: "ab[^e-g]", Name: "abc", Matched: true},
	}
}

func match_cases_two() (cases match_case_set) {
	return match_case_set{
		{Pattern: "a\\*b", Name: "a*b", Matched: true},
		{Pattern: "a\\*b", Name: "ab"},
		{Pattern: "a?b", Name: "a☺b", Matched: true},
		{Pattern: "a[^a]b", Name: "a☺b", Matched: true},
		{Pattern: "a???b", Name: "a☺b"},
		{Pattern: "a[^a][^a][^a]b", Name: "a☺b"},
		{Pattern: "[a-ζ]*", Name: "α", Matched: true},
		{Pattern: "[\x01]", Name: "\x01", Matched: true},
		{Pattern: "[\x02]", Name: "\x02", Matched: true},
		{Pattern: "[\U0010ffff]", Name: "\U0010ffff", Matched: true},
		{Pattern: "*[a-ζ]", Name: "A"},
		{Pattern: "a?b", Name: "a/b"},
		{Pattern: "a*b", Name: "a/b"},
		{Pattern: "[\\]a]", Name: "]", Matched: true},
		{Pattern: "[\\-]", Name: "-", Matched: true},
		{Pattern: "[x\\-]", Name: "x", Matched: true},
		{Pattern: "[x\\-]", Name: "-", Matched: true},
		{Pattern: "[x\\-]", Name: "z"},
		{Pattern: "[\\-x]", Name: "x", Matched: true},
		{Pattern: "[\\-x]", Name: "-", Matched: true},
		{Pattern: "[\\-x]", Name: "a"},
	}
}

func match_cases_three() (cases match_case_set) {
	return match_case_set{
		{Pattern: "[]a]", Name: "]", Bad: true},
		{Pattern: "[-]", Name: "-", Bad: true},
		{Pattern: "[x-]", Name: "x", Bad: true},
		{Pattern: "[x-]", Name: "-", Bad: true},
		{Pattern: "[x-]", Name: "z", Bad: true},
		{Pattern: "[-x]", Name: "x", Bad: true},
		{Pattern: "[-x]", Name: "-", Bad: true},
		{Pattern: "[-x]", Name: "a", Bad: true},
		{Pattern: "\\", Name: "a", Bad: true},
		{Pattern: "[a-b-c]", Name: "a", Bad: true},
		{Pattern: "[", Name: "a", Bad: true},
		{Pattern: "[^", Name: "a", Bad: true},
		{Pattern: "[^bc", Name: "a", Bad: true},
		{Pattern: "a[", Name: "a", Bad: true},
		{Pattern: "a[", Name: "ab", Bad: true},
		{Pattern: "a[", Name: "x", Bad: true},
		{Pattern: "a/b[", Name: "x", Bad: true},
		{Pattern: "*x", Name: "xxx", Matched: true},
	}
}

func test_clean(t *testing.T) {
	t.Parallel()
	var storage [TEST_STORAGE_SIZE]byte
	for _, one := range clean_cases() {
		count := path.Clean_Into(storage[:], one.Input)
		testify.Equal(t, one.Want, string(storage[:count]), "Clean_Into(%q)", one.Input)
		count = path.Clean_Into(storage[:], path.Text(one.Want))
		testify.Equal(t, one.Want, string(storage[:count]), "Clean_Into(%q)", one.Want)
	}
}

func test_components(t *testing.T) {
	t.Parallel()
	for _, one := range split_cases() {
		directory, file := path.Split(one.Input)
		testify.Equal(t, one.Directory, string(directory), "Split(%q) directory", one.Input)
		testify.Equal(t, one.File, string(file), "Split(%q) file", one.Input)
	}
	for input, want := range map[path.Text]string{
		"": "", ".": ".", "..": ".", ".a": ".a", "path.go": ".go", "path.pb.go": ".go",
		"a.dir/b":    "",
		"a.dir/b.go": ".go", "a.dir/": "",
	} {
		testify.Equal(t, want, string(path.Extension(input)), "Extension(%q)", input)
	}
	for input, want := range map[path.Text]string{
		"": ".", ".": ".", "/.": ".", "/": "/", "////": "/", "x/": "x",
		"abc": "abc", "abc/def": "def", "a/b/.x": ".x", "a/b/c.": "c.",
		"a/b/c.x": "c.x",
	} {
		testify.Equal(t, want, string(path.Base(input)), "Base(%q)", input)
	}
	var storage [TEST_STORAGE_SIZE]byte
	for input, want := range map[path.Text]string{
		"": ".", ".": ".", "/.": "/", "/": "/", "////": "/", "/foo": "/",
		"x/": "x", "abc": ".", "abc/def": "abc", "abc////def": "abc",
		"a/b/.x": "a/b", "a/b/c.": "a/b", "a/b/c.x": "a/b",
	} {
		count := path.Directory_Into(storage[:], input)
		testify.Equal(t, want, string(storage[:count]), "Directory_Into(%q)", input)
	}
	for input, want := range map[path.Text]bool{
		"": false, "/": true, "/usr/bin/gcc": true, "..": false,
		"/a/../bb": true, ".": false, "./": false, "lala": false,
	} {
		testify.Equal(t, want, bool(path.Is_Absolute(input)), "Is_Absolute(%q)", input)
	}
}

func test_join(t *testing.T) {
	t.Parallel()
	var storage [TEST_STORAGE_SIZE]byte
	for _, one := range join_cases() {
		count := path.Join_Into(storage[:], one.Elements)
		testify.Equal(t, one.Want, string(storage[:count]), "Join_Into(%q)", one.Elements)
	}
}

func test_match(t *testing.T) {
	t.Parallel()
	verify_match_cases(t, match_cases_one())
	verify_match_cases(t, match_cases_two())
	verify_match_cases(t, match_cases_three())
}

func verify_match_cases(t *testing.T, cases match_case_set) {
	for _, one := range cases {
		matched, err := path.Match(one.Pattern, one.Name)
		testify.Equal(t, one.Matched, bool(matched), "Match(%q, %q)", one.Pattern, one.Name)
		if one.Bad {
			testify.Error_Is(
				t, err, path.Error_Bad_Pattern,
				"Match(%q, %q)", one.Pattern, one.Name,
			)
		} else {
			testify.No_Error(t, err, "Match(%q, %q)", one.Pattern, one.Name)
		}
	}
}

func test_bounds(t *testing.T) {
	t.Parallel()
	operating_system_path_size_bound(t)
	path_size_bounds(t)
	match_size_bounds(t)
	element_and_destination_bounds(t)
	invalid_bounds(t)
}

func operating_system_path_size_bound(t *testing.T) {
	expected := 0
	switch runtime.GOOS {
	case "darwin":
		expected = bits.KIBIBYTE_BYTES - path.PATH_TERMINATOR_BYTES
	case "linux":
		expected = 4*bits.KIBIBYTE_BYTES - path.PATH_TERMINATOR_BYTES
	default:
		t.Fatalf("path: unsupported operating system %q", runtime.GOOS)
	}
	testify.Equal(t, expected, path.PATH_SIZE_MAXIMUM, "operating system path size")
}

func path_size_bounds(t *testing.T) {
	var oversized_storage [path.PATH_SIZE_MAXIMUM + 1]byte
	for index := range path.PATH_SIZE_MAXIMUM {
		oversized_storage[index] = 'a'
	}
	maximum := path.Text(string(oversized_storage[:path.PATH_SIZE_MAXIMUM]))
	var storage [TEST_STORAGE_SIZE]byte
	testify.Equal(t, path.PATH_SIZE_MAXIMUM,
		int(path.Clean_Into(storage[:], maximum)), "maximum Clean_Into output")
	directory, file := path.Split(maximum)
	testify.Equal(t, "", string(directory), "maximum unsplit directory")
	testify.Equal(t, string(maximum), string(file), "maximum unsplit file")
	testify.Equal(t, string(maximum), string(path.Base(maximum)), "maximum Base output")
	oversized_storage[0] = '.'
	maximum_extension := path.Text(string(oversized_storage[:path.PATH_SIZE_MAXIMUM]))
	testify.Equal(t, string(maximum_extension), string(path.Extension(maximum_extension)),
		"maximum Extension output")
	oversized_storage[0] = '/'
	maximum_absolute := path.Text(string(oversized_storage[:path.PATH_SIZE_MAXIMUM]))
	testify.True(t, bool(path.Is_Absolute(maximum_absolute)), "maximum absolute path")
	oversized_storage[0] = 'a'
	oversized_storage[path.PATH_SIZE_MAXIMUM-1] = '/'
	maximum_directory := path.Text(string(oversized_storage[:path.PATH_SIZE_MAXIMUM]))
	testify.Equal(t, path.DIRECTORY_SIZE_MAXIMUM,
		int(path.Directory_Into(storage[:], maximum_directory)),
		"maximum Directory_Into output")
	testify.Equal(t, path.PATH_SIZE_MAXIMUM,
		int(path.Join_Into(storage[:], path.Elements{bytes.Text(maximum)})),
		"maximum Join_Into output")
}

func match_size_bounds(t *testing.T) {
	var maximum_storage [path.PATH_SIZE_MAXIMUM]byte
	for index := range maximum_storage {
		maximum_storage[index] = 'a'
	}
	maximum := path.Text(string(maximum_storage[:]))
	matched, match_error := path.Match(maximum, maximum)
	testify.No_Error(t, match_error, "maximum Match input")
	testify.True(t, bool(matched), "maximum Match input")
	matched, match_error = path.Match("a*", maximum)
	testify.No_Error(t, match_error, "maximum Match tail")
	testify.True(t, bool(matched), "maximum Match tail")
	maximum_storage[0] = 'a'
	for index := 1; index < path.PATH_SIZE_MAXIMUM; index++ {
		maximum_storage[index] = '*'
	}
	maximum_star_tail := path.Text(string(maximum_storage[:]))
	matched, match_error = path.Match(maximum_star_tail, "a")
	testify.No_Error(t, match_error, "maximum pattern tail")
	testify.True(t, bool(matched), "maximum pattern tail")
	maximum_storage[0] = '['
	for index := 1; index < path.PATH_SIZE_MAXIMUM; index++ {
		maximum_storage[index] = 'a'
	}
	maximum_class := path.Text(string(maximum_storage[:]))
	_, match_error = path.Match(maximum_class, "a")
	testify.Error_Is(t, match_error, path.Error_Bad_Pattern, "maximum class tail")
	maximum_storage[0] = '['
	maximum_storage[1] = 'a'
	maximum_storage[2] = ']'
	for index := 3; index < path.PATH_SIZE_MAXIMUM; index++ {
		maximum_storage[index] = '?'
	}
	maximum_class_result := path.Text(string(maximum_storage[:]))
	var maximum_name_storage [path.CLASS_MATCH_TAIL_SIZE_MAXIMUM + 1]byte
	for index := range maximum_name_storage {
		maximum_name_storage[index] = 'a'
	}
	matched, match_error = path.Match(
		maximum_class_result, path.Text(string(maximum_name_storage[:])),
	)
	testify.No_Error(t, match_error, "maximum class result tail")
	testify.True(t, bool(matched), "maximum class result tail")
	matched, match_error = path.Match("b*", "a")
	testify.No_Error(t, match_error, "empty validation chunk")
	testify.False(t, bool(matched), "empty validation chunk")
}

func element_and_destination_bounds(t *testing.T) {
	var storage [TEST_STORAGE_SIZE]byte
	testify.Equal(t, 2, int(path.Join_Into(storage[:2], path.Elements{"ab"})),
		"two-byte join destination")
	var maximum_elements [path.ELEMENT_COUNT_MAXIMUM]bytes.Text
	testify.Equal(t, 0,
		int(path.Join_Into(storage[:0], path.Elements(maximum_elements[:]))),
		"maximum empty element count")
	testify.Equal(t, 1, int(path.Clean_Into(storage[:1], "")), "one-byte destination")
	testify.Equal(t, 2, int(path.Clean_Into(storage[:2], "..")), "two-byte destination")
	testify.Equal(t, 1, int(path.Directory_Into(storage[:1], "a")),
		"one-byte directory destination")
	testify.Equal(t, 2, int(path.Directory_Into(storage[:2], "../a")),
		"two-byte directory destination")
}

func invalid_bounds(t *testing.T) {
	var oversized_storage [path.PATH_SIZE_MAXIMUM + 1]byte
	oversized := path.Text(string(oversized_storage[:]))
	var storage [TEST_STORAGE_SIZE]byte
	testify.Panics(t, func() { path.Clean_Into(storage[:], oversized) })
	testify.Panics(t, func() { path.Split(oversized) })
	testify.Panics(t, func() { path.Extension(oversized) })
	testify.Panics(t, func() { path.Base(oversized) })
	testify.Panics(t, func() { path.Is_Absolute(oversized) })
	testify.Panics(t, func() { path.Directory_Into(storage[:], oversized) })
	testify.Panics(t, func() { path.Match(oversized, "name") })
	testify.Panics(t, func() { path.Match("*", oversized) })
	testify.Panics(t, func() {
		path.Join_Into(storage[:], path.Elements{bytes.Text(oversized)})
	})
	testify.Panics(t, func() {
		path.Join_Into(storage[:], path.Elements{
			bytes.Text(oversized[:path.PATH_SIZE_MAXIMUM]), "a",
		})
	})
	var too_many [path.ELEMENT_COUNT_MAXIMUM + 1]bytes.Text
	testify.Panics(t, func() { path.Join_Into(storage[:], path.Elements(too_many[:])) })
	testify.Panics(t, func() { path.Clean_Into(storage[:0], "a") })
	testify.Panics(t, func() {
		path.Join_Into(storage[:1], path.Elements{"a", "b"})
	})
	testify.Panics(t, func() { path.Directory_Into(storage[:0], "a") })
}

type allocation_fixture struct {
	Storage         [TEST_STORAGE_SIZE]byte
	Count           path.Boundary
	Nonempty_Count  path.Nonempty_Count
	Directory_Count path.Directory_Count
	Directory       path.Text
	File            path.Text
	Text            path.Text
	Matched         path.Boolean
	Absolute        path.Boolean
	Error           error
	Components      [TEST_COMPONENT_COUNT]bytes.Text
}

func test_allocation(t *testing.T) {
	fixture := allocation_fixture{}
	checks := []struct {
		Name string
		Call func()
	}{
		{Name: "Clean_Into", Call: func() {
			fixture.Nonempty_Count = path.Clean_Into(fixture.Storage[:], "a/../b")
		}},
		{Name: "Split", Call: func() {
			fixture.Directory, fixture.File = path.Split("a/b")
		}},
		{Name: "Join_Into", Call: func() {
			fixture.Count = path.Join_Into(
				fixture.Storage[:], path.Elements(fixture.Components[:]),
			)
		}},
		{Name: "Extension", Call: func() { fixture.Text = path.Extension("a/b.go") }},
		{Name: "Base", Call: func() { fixture.Text = path.Base("a/b") }},
		{Name: "Is_Absolute", Call: func() { fixture.Absolute = path.Is_Absolute("/a") }},
		{Name: "Directory_Into", Call: func() {
			fixture.Directory_Count = path.Directory_Into(fixture.Storage[:], "a/b")
		}},
		{Name: "Match", Call: func() {
			fixture.Matched, fixture.Error = path.Match("a*", "abc")
		}},
	}
	fixture.Components = [TEST_COMPONENT_COUNT]bytes.Text{"a", "b"}
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) { testify.Zero_Allocation(t, check.Call) })
	}
	if fixture.Count == -1 {
		t.Fatal("operations produced impossible observation")
	}
}
