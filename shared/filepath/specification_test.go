package filepath_test

import (
	"testing"
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/filepath"
	"local/james-orcales/shared/sim/nbio"
	"local/james-orcales/shared/testify"
)

// Test_Clean stays first because specification leaf order is declaration-sensitive.
func Test_Clean(t *testing.T) {
	test_clean(t)
}

// Test_Local_Paths stays ordered because specification mapping rejects reordered leaves.
func Test_Local_Paths(t *testing.T) {
	test_local_paths(t)
}

// Test_Slash_Conversion stays ordered because specification mapping is positional.
func Test_Slash_Conversion(t *testing.T) {
	test_slash_conversion(t)
}

// Test_Components stays ordered because specification mapping is positional.
func Test_Components(t *testing.T) {
	test_components(t)
}

// Test_Join stays ordered because specification mapping is positional.
func Test_Join(t *testing.T) {
	test_join(t)
}

// Test_Absolute_And_Relative stays ordered because specification mapping is positional.
func Test_Absolute_And_Relative(t *testing.T) {
	test_absolute_and_relative(t)
}

// Test_Match stays ordered because specification mapping is positional.
func Test_Match(t *testing.T) {
	test_match(t)
}

// Test_Glob stays ordered because specification mapping is positional.
func Test_Glob(t *testing.T) {
	test_glob(t)
}

// Test_Walk stays ordered because specification mapping is positional.
func Test_Walk(t *testing.T) {
	test_walk(t)
}

// Test_Symbolic_Links stays ordered because specification mapping is positional.
func Test_Symbolic_Links(t *testing.T) {
	test_symbolic_links(t)
}

// Test_Bounds stays ordered because specification mapping is positional.
func Test_Bounds(t *testing.T) {
	test_bounds(t)
}

// Test_Allocation stays ordered because specification mapping is positional.
func Test_Allocation(t *testing.T) {
	test_allocation(t)
}

const TEST_STORAGE_SIZE = bytes.SLICE_SIZE_MAXIMUM

const TEST_COMPONENT_COUNT = 2

type text_case struct {
	Input filepath.Text
	Want  string
}

type clean_case_set []text_case

type relative_case struct {
	Base   filepath.Text
	Target filepath.Text
	Want   string
}

type relative_case_set []relative_case

type match_case struct {
	Pattern filepath.Text
	Name    filepath.Text
	Matched bool
	Bad     bool
}

type match_case_set []match_case

func clean_cases() (cases clean_case_set) {
	return clean_case_set{
		{Input: "abc", Want: "abc"},
		{Input: "abc/def", Want: "abc/def"},
		{Input: "a/b/c", Want: "a/b/c"},
		{Input: ".", Want: "."},
		{Input: "..", Want: ".."},
		{Input: "../..", Want: "../.."},
		{Input: "../../abc", Want: "../../abc"},
		{Input: "/abc", Want: "/abc"},
		{Input: "/", Want: "/"},
		{Input: "", Want: "."},
		{Input: "abc/", Want: "abc"},
		{Input: "abc/def/", Want: "abc/def"},
		{Input: "a/b/c/", Want: "a/b/c"},
		{Input: "./", Want: "."},
		{Input: "../", Want: ".."},
		{Input: "../../", Want: "../.."},
		{Input: "/abc/", Want: "/abc"},
		{Input: "abc//def//ghi", Want: "abc/def/ghi"},
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
		{Input: "/../abc", Want: "/abc"},
		{Input: "a/../b:/../../c", Want: "../c"},
		{Input: "abc/./../def", Want: "def"},
		{Input: "abc//./../def", Want: "def"},
		{Input: "abc/../../././../def", Want: "../../def"},
		{Input: "//abc", Want: "/abc"},
		{Input: "///abc", Want: "/abc"},
		{Input: "//abc//", Want: "/abc"},
	}
}

func relative_cases() (cases relative_case_set) {
	return relative_case_set{
		{Base: "a/b", Target: "a/b", Want: "."},
		{Base: "a/b/.", Target: "a/b", Want: "."},
		{Base: "a/b", Target: "a/b/.", Want: "."},
		{Base: "./a/b", Target: "a/b", Want: "."},
		{Base: "a/b", Target: "./a/b", Want: "."},
		{Base: "ab/cd", Target: "ab/cde", Want: "../cde"},
		{Base: "ab/cd", Target: "ab/c", Want: "../c"},
		{Base: "a/b", Target: "a/b/c/d", Want: "c/d"},
		{Base: "a/b", Target: "a/b/../c", Want: "../c"},
		{Base: "a/b/../c", Target: "a/b", Want: "../b"},
		{Base: "a/b/c", Target: "a/c/d", Want: "../../c/d"},
		{Base: "a/b", Target: "c/d", Want: "../../c/d"},
		{Base: "a/b/c/d", Target: "a/b", Want: "../.."},
		{Base: "a/b/c/d", Target: "a/b/", Want: "../.."},
		{Base: "a/b/c/d/", Target: "a/b", Want: "../.."},
		{Base: "a/b/c/d/", Target: "a/b/", Want: "../.."},
		{Base: "../../a/b", Target: "../../a/b/c/d", Want: "c/d"},
		{Base: "/a/b", Target: "/a/b", Want: "."},
		{Base: "/a/b/.", Target: "/a/b", Want: "."},
		{Base: "/a/b", Target: "/a/b/.", Want: "."},
		{Base: "/ab/cd", Target: "/ab/cde", Want: "../cde"},
		{Base: "/ab/cd", Target: "/ab/c", Want: "../c"},
		{Base: "/a/b", Target: "/a/b/c/d", Want: "c/d"},
		{Base: "/a/b", Target: "/a/b/../c", Want: "../c"},
		{Base: "/a/b/../c", Target: "/a/b", Want: "../b"},
		{Base: "/a/b/c", Target: "/a/c/d", Want: "../../c/d"},
		{Base: "/a/b", Target: "/c/d", Want: "../../c/d"},
		{Base: "/a/b/c/d", Target: "/a/b", Want: "../.."},
		{Base: "/a/b/c/d", Target: "/a/b/", Want: "../.."},
		{Base: "/a/b/c/d/", Target: "/a/b", Want: "../.."},
		{Base: "/a/b/c/d/", Target: "/a/b/", Want: "../.."},
		{Base: "/../../a/b", Target: "/../../a/b/c/d", Want: "c/d"},
		{Base: ".", Target: "a/b", Want: "a/b"},
		{Base: ".", Target: "..", Want: ".."},
		{Base: "", Target: "../../.", Want: "../.."},
	}
}

func match_cases() (cases match_case_set) {
	return match_case_set{
		{Pattern: "", Name: "", Matched: true},
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
		{Pattern: "ab[^c]", Name: "abc"},
		{Pattern: "ab[^b-d]", Name: "abc"},
		{Pattern: "ab[^e-g]", Name: "abc", Matched: true},
		{Pattern: "a\\*b", Name: "a*b", Matched: true},
		{Pattern: "a\\*b", Name: "ab"},
		{Pattern: "a?b", Name: "a☺b", Matched: true},
		{Pattern: "a[^a]b", Name: "a☺b", Matched: true},
		{Pattern: "a???b", Name: "a☺b"},
		{Pattern: "a[^a][^a][^a]b", Name: "a☺b"},
		{Pattern: "[a-ζ]*", Name: "α", Matched: true},
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
		count := filepath.Clean_Into(storage[:], one.Input)
		testify.Equal(t, one.Want, string(storage[:count]), "Clean_Into(%q)", one.Input)
	}
}

func test_local_paths(t *testing.T) {
	t.Parallel()
	for _, value := range []filepath.Text{
		".", "a", "a/../a", "a/", "a/.", "a/./b/./c", "a//b", "a/b", "a/../b", "a/..",
	} {
		testify.True(t, bool(filepath.Is_Local(value)), "Is_Local(%q)", value)
	}
	for _, value := range []filepath.Text{
		"", "..", "../a", "/", "/a", "/a/../..", "a/../../b", "a/../b:/../../c",
	} {
		testify.False(t, bool(filepath.Is_Local(value)), "Is_Local(%q)", value)
	}
	var storage [TEST_STORAGE_SIZE]byte
	for _, value := range []filepath.Text{".", "a", "a/b", "a/b/c", "#a", `a\b:c`} {
		count, err := filepath.Localize_Into(storage[:], value)
		testify.No_Error(t, err, "Localize_Into(%q)", value)
		testify.Equal(t, string(value), string(storage[:count]), "Localize_Into(%q)", value)
	}
	for _, value := range []filepath.Text{
		"", "..", "a/..", "/", "/a", "a\xffb", "a/", "a//b", "a/./b", "a/../b",
		"\x00", "a\x00b",
	} {
		_, err := filepath.Localize_Into(storage[:], value)
		testify.Error_Is(t, err, filepath.Error_Invalid_Path, "Localize_Into(%q)", value)
	}
}

func test_slash_conversion(t *testing.T) {
	t.Parallel()
	var storage [TEST_STORAGE_SIZE]byte
	for _, value := range []filepath.Text{"", "/", "/a/b", "a", "a/b", "a//b"} {
		count := filepath.To_Slash_Into(storage[:], value)
		testify.Equal(
			t, string(value), string(storage[:count]), "To_Slash_Into(%q)", value,
		)
		count = filepath.From_Slash_Into(storage[:], value)
		testify.Equal(
			t, string(value), string(storage[:count]), "From_Slash_Into(%q)", value,
		)
	}
}

func test_components(t *testing.T) {
	t.Parallel()
	var paths [filepath.PATH_COUNT_MAXIMUM]bytes.Text
	count := filepath.Split_List_Into(paths[:], "a:b::c")
	want := [...]string{"a", "b", "", "c"}
	testify.Equal(t, len(want), int(count), "Split_List_Into count")
	for index := range want {
		testify.Equal(t, want[index], string(paths[index]), "Split_List_Into item")
	}
	testify.Equal(t, 0, int(filepath.Split_List_Into(paths[:], "")), "empty path list")
	count = filepath.Split_List_Into(paths[:], ":a:b")
	want = [...]string{"", "a", "b", ""}
	testify.Equal(t, 3, int(count), "leading-empty Split_List_Into count")
	for index := range int(count) {
		testify.Equal(
			t, want[index], string(paths[index]), "leading-empty Split_List_Into item",
		)
	}
	for input, want_parts := range map[filepath.Text][TEST_COMPONENT_COUNT]string{
		"a/b": {"a/", "b"}, "a/b/": {"a/b/", ""}, "a/": {"a/", ""},
		"a": {"", "a"}, "/": {"/", ""},
	} {
		directory, file := filepath.Split(input)
		testify.Equal(t, want_parts[0], string(directory), "Split directory")
		testify.Equal(t, want_parts[1], string(file), "Split file")
	}
	for input, want_value := range map[filepath.Text]string{
		"path.go": ".go", "path.pb.go": ".go", "a.dir/b": "",
		"a.dir/b.go": ".go", "a.dir/": "",
	} {
		testify.Equal(t, want_value, string(filepath.Extension(input)), "Extension")
	}
	for input, want_value := range map[filepath.Text]string{
		"": ".", ".": ".", "/.": ".", "/": "/", "////": "/", "x/": "x",
		"abc": "abc", "abc/def": "def", "a/b/.x": ".x", "a/b/c.": "c.",
		"a/b/c.x": "c.x",
	} {
		testify.Equal(t, want_value, string(filepath.Base(input)), "Base")
	}
	var storage [TEST_STORAGE_SIZE]byte
	for input, want_value := range map[filepath.Text]string{
		"": ".", ".": ".", "/.": "/", "/": "/", "/foo": "/", "x/": "x",
		"abc": ".", "abc/def": "abc", "a/b/.x": "a/b", "a/b/c.": "a/b",
		"a/b/c.x": "a/b", "////": "/",
	} {
		count_directory := filepath.Directory_Into(storage[:], input)
		testify.Equal(t, want_value, string(storage[:count_directory]), "Directory_Into")
	}
	testify.Equal(t, "", string(filepath.Volume_Name("/a/b")), "Volume_Name")
	testify.True(t, bool(filepath.Has_Prefix("a/b", "a/")), "Has_Prefix")
	for input, expected := range map[filepath.Text]bool{
		"": false, "/": true, "/usr/bin/gcc": true, "..": false, "/a/../bb": true,
		".": false, "./": false, "lala": false,
	} {
		testify.Equal(t, expected, bool(filepath.Is_Absolute(input)), "Is_Absolute")
	}
}

func test_join(t *testing.T) {
	t.Parallel()
	var storage [TEST_STORAGE_SIZE]byte
	for _, one := range []struct {
		Elements filepath.Elements
		Want     string
	}{
		{Elements: filepath.Elements{}, Want: ""},
		{Elements: filepath.Elements{""}, Want: ""},
		{Elements: filepath.Elements{"/"}, Want: "/"},
		{Elements: filepath.Elements{"a"}, Want: "a"},
		{Elements: filepath.Elements{"a", "b"}, Want: "a/b"},
		{Elements: filepath.Elements{"a", ""}, Want: "a"},
		{Elements: filepath.Elements{"", "b"}, Want: "b"},
		{Elements: filepath.Elements{"/", "a"}, Want: "/a"},
		{Elements: filepath.Elements{"/", "a/b"}, Want: "/a/b"},
		{Elements: filepath.Elements{"/", ""}, Want: "/"},
		{Elements: filepath.Elements{"/a", "b"}, Want: "/a/b"},
		{Elements: filepath.Elements{"a", "/b"}, Want: "a/b"},
		{Elements: filepath.Elements{"/a", "/b"}, Want: "/a/b"},
		{Elements: filepath.Elements{"a/", "b"}, Want: "a/b"},
		{Elements: filepath.Elements{"a/", ""}, Want: "a"},
		{Elements: filepath.Elements{"", ""}, Want: ""},
		{Elements: filepath.Elements{"/", "a", "b"}, Want: "/a/b"},
		{Elements: filepath.Elements{"//", "a"}, Want: "/a"},
		{Elements: filepath.Elements{"a", "../../b"}, Want: "../b"},
	} {
		count := filepath.Join_Into(storage[:], one.Elements)
		testify.Equal(t, one.Want, string(storage[:count]), "Join_Into(%q)", one.Elements)
	}
}

func test_absolute_and_relative(t *testing.T) {
	t.Parallel()
	var storage [TEST_STORAGE_SIZE]byte
	count, err := filepath.Absolute_Into(storage[:], "/work", "a/../b")
	testify.No_Error(t, err, "relative absolute path")
	testify.Equal(t, "/work/b", string(storage[:count]), "relative absolute path")
	count, err = filepath.Absolute_Into(storage[:], "/work", "/a/../b")
	testify.No_Error(t, err, "already absolute path")
	testify.Equal(t, "/b", string(storage[:count]), "already absolute path")
	for _, one := range relative_cases() {
		count, err = filepath.Relative_Into(storage[:], one.Base, one.Target)
		testify.No_Error(t, err, "Relative_Into(%q, %q)", one.Base, one.Target)
		testify.Equal(
			t, one.Want, string(storage[:count]),
			"Relative_Into(%q, %q)", one.Base, one.Target,
		)
	}
	for _, one := range []struct {
		Base   filepath.Text
		Target filepath.Text
		Error  error
	}{
		{Base: "..", Target: ".", Error: filepath.Error_Base_Above_Root},
		{Base: "..", Target: "a", Error: filepath.Error_Base_Above_Root},
		{Base: "../..", Target: "..", Error: filepath.Error_Base_Above_Root},
		{Base: "../a", Target: "b", Error: filepath.Error_Base_Above_Root},
		{Base: "a", Target: "/a", Error: filepath.Error_Root_Mismatch},
		{Base: "/a", Target: "a", Error: filepath.Error_Root_Mismatch},
	} {
		_, err = filepath.Relative_Into(storage[:], one.Base, one.Target)
		testify.Error_Is(t, err, one.Error, "Relative_Into(%q, %q)", one.Base, one.Target)
	}
}

func test_match(t *testing.T) {
	t.Parallel()
	for _, one := range match_cases() {
		matched, err := filepath.Match(one.Pattern, one.Name)
		testify.Equal(
			t, one.Matched, bool(matched), "Match(%q, %q)", one.Pattern, one.Name,
		)
		if one.Bad {
			testify.Error_Is(t, err, filepath.Error_Bad_Pattern, "bad pattern")
		} else {
			testify.No_Error(t, err, "Match(%q, %q)", one.Pattern, one.Name)
		}
	}
}

// TEST_FILESYSTEM_PATH_CAPACITY bounds public runner fixtures.
const TEST_FILESYSTEM_PATH_CAPACITY = 2

// TEST_FILESYSTEM_ENTRY_CAPACITY matches fixture path width.
const TEST_FILESYSTEM_ENTRY_CAPACITY = TEST_FILESYSTEM_PATH_CAPACITY

const TEST_RUNNER_REARM_MAXIMUM = 32

func test_glob(t *testing.T) {
	t.Parallel()
	storage := nbio.Storage{Status_Procedure: specification_status}
	runner := filepath.Glob_Runner{}
	err := filepath.Glob_Runner_Init(
		&runner, nbio.IO{Storage: storage}, "file", filepath.Glob_Memory{
			Current: filepath.Glob_Current_Paths(specification_paths()),
			Next:    filepath.Glob_Next_Paths(specification_paths()),
			Entries: filepath.Directory_Entries(
				make([]nbio.Directory_Entry, TEST_FILESYSTEM_ENTRY_CAPACITY),
			),
			Directory_Buffer: filepath.Directory_Buffer(
				make(bytes.Slice, nbio.DIRECTORY_BUFFER_SIZE_MINIMUM),
			),
		},
	)
	testify.No_Error(t, err)
	testify.True(t, bool(filepath.Glob_Runner_Stopped(&runner)))
	matches := filepath.Glob_Runner_Matches(&runner)
	testify.Equal(t, 1, len(matches))
	testify.Equal(t, "file", string(matches[0]))
}

type specification_walk_state struct {
	Path string
}

func test_walk(t *testing.T) {
	t.Parallel()
	for _, directory_entry := range []bool{false, true} {
		state := specification_walk_state{}
		runner := filepath.Walk_Runner{}
		memory := filepath.Walk_Memory{
			Queue:    filepath.Walk_Queue_Paths(specification_paths()),
			Children: filepath.Walk_Child_Paths(specification_paths()),
			Entries: filepath.Directory_Entries(
				make([]nbio.Directory_Entry, TEST_FILESYSTEM_ENTRY_CAPACITY),
			),
			Directory_Buffer: filepath.Directory_Buffer(
				make(bytes.Slice, nbio.DIRECTORY_BUFFER_SIZE_MINIMUM),
			),
		}
		var err error
		if directory_entry {
			err = filepath.Walk_Directory_Runner_Init(
				&runner, nbio.IO{Storage: nbio.Storage{
					Status_Procedure: specification_status,
				}}, "file", filepath.Walk_Visitor_State{
					Pointer: unsafe.Pointer(&state),
				},
				specification_walk_visit, memory,
			)
		} else {
			err = filepath.Walk_Runner_Init(
				&runner, nbio.IO{Storage: nbio.Storage{
					Status_Procedure: specification_status,
				}}, "file", filepath.Walk_Visitor_State{
					Pointer: unsafe.Pointer(&state),
				},
				specification_walk_visit, memory,
			)
		}
		testify.No_Error(t, err)
		testify.True(t, bool(filepath.Walk_Runner_Rearm(&runner)))
		testify.False(t, bool(filepath.Walk_Runner_Rearm(&runner)))
		testify.True(t, bool(filepath.Walk_Runner_Stopped(&runner)))
		testify.Equal(t, "file", state.Path)
	}
}

func test_symbolic_links(t *testing.T) {
	t.Parallel()
	storage := nbio.Storage{
		Status_Procedure:    specification_link_status,
		Read_Link_Procedure: specification_link_read,
	}
	var destination [filepath.PATH_SIZE_MAXIMUM]byte
	var remainder [filepath.PATH_SIZE_MAXIMUM]byte
	var link_target [filepath.PATH_SIZE_MAXIMUM]byte
	count, err := filepath.Eval_Symlinks_Into(
		storage, destination[:], remainder[:], link_target[:], "link",
	)
	testify.No_Error(t, err)
	testify.Equal(t, "target", string(destination[:count]))
}

func specification_paths() (paths filepath.Path_Storage) {
	paths = make(filepath.Path_Storage, TEST_FILESYSTEM_PATH_CAPACITY)
	for index := range paths {
		paths[index] = make(filepath.Slice, filepath.PATH_SIZE_MAXIMUM)
	}
	return paths
}

func specification_status(
	_ unsafe.Pointer, _ string,
) (status nbio.File_Status, err error) {
	return nbio.File_Status{Exists: true}, nil
}

func specification_walk_visit(
	visitor_state filepath.Walk_Visitor_State, path bytes.Slice, _ filepath.Walk_Entry,
	visit_err error,
) (err error) {
	if visit_err != nil {
		return visit_err
	}
	state := (*specification_walk_state)(visitor_state.Pointer)
	state.Path = string(path)
	return nil
}

func specification_link_status(
	_ unsafe.Pointer, path string,
) (status nbio.File_Status, err error) {
	if path == "link" {
		return nbio.File_Status{Exists: true, Mode: nbio.FILE_MODE_SYMBOLIC_LINK}, nil
	}
	if path == "target" {
		return nbio.File_Status{Exists: true}, nil
	}
	return nbio.File_Status{}, nil
}

func specification_link_read(
	_ unsafe.Pointer, path string, destination []byte,
) (count int, err error) {
	if path != "link" {
		return 0, filepath.Error_Path_Absent
	}
	return copy(destination, "target"), nil
}

func test_bounds(t *testing.T) {
	t.Parallel()
	public_size_bounds(t)
	absolute_size_bounds(t)
	relative_size_bounds(t)
	collection_size_bounds(t)
	destination_size_bounds(t)
	invalid_input_bounds(t)
}

func public_size_bounds(t *testing.T) {
	var maximum_storage [filepath.PATH_SIZE_MAXIMUM]byte
	for index := range maximum_storage {
		maximum_storage[index] = 'a'
	}
	maximum := filepath.Text(string(maximum_storage[:]))
	var storage [TEST_STORAGE_SIZE]byte
	testify.Equal(t, filepath.PATH_SIZE_MAXIMUM,
		int(filepath.Clean_Into(storage[:], maximum)), "maximum clean")
	testify.True(t, bool(filepath.Is_Local(maximum)), "maximum local path")
	count, err := filepath.Localize_Into(storage[:], maximum)
	testify.No_Error(t, err, "maximum localize")
	testify.Equal(t, filepath.PATH_SIZE_MAXIMUM, int(count), "maximum localize")
	testify.Equal(t, filepath.PATH_SIZE_MAXIMUM,
		int(filepath.To_Slash_Into(storage[:], maximum)), "maximum To_Slash")
	testify.Equal(t, filepath.PATH_SIZE_MAXIMUM,
		int(filepath.From_Slash_Into(storage[:], maximum)), "maximum From_Slash")
	directory, file := filepath.Split(maximum)
	testify.Equal(t, "", string(directory), "maximum split directory")
	testify.Equal(t, string(maximum), string(file), "maximum split file")
	filepath.Split("")
	_, file = filepath.Split("ab")
	testify.Equal(t, "ab", string(file), "two-byte split file")
	testify.Equal(t, filepath.PATH_SIZE_MAXIMUM,
		int(filepath.Join_Into(
			storage[:], filepath.Elements{bytes.Text(maximum)},
		)), "maximum join")
	testify.Equal(t, string(maximum), string(filepath.Base(maximum)), "maximum base")
	testify.Equal(t, "ab", string(filepath.Base("ab")), "two-byte base")
	for _, value := range []filepath.Text{"", ".", ".a"} {
		filepath.Extension(value)
	}
	maximum_storage[0] = '.'
	maximum_extension := filepath.Text(string(maximum_storage[:]))
	testify.Equal(t, string(maximum_extension),
		string(filepath.Extension(maximum_extension)), "maximum extension")
	maximum_storage[0] = 'a'
	maximum_storage[filepath.PATH_SIZE_MAXIMUM-1] = '/'
	maximum_directory := filepath.Text(string(maximum_storage[:]))
	maximum_split_directory_bound(t, maximum_directory)
	testify.Equal(t, filepath.DIRECTORY_SIZE_MAXIMUM,
		int(filepath.Directory_Into(storage[:], maximum_directory)), "maximum directory")
	for _, value := range []filepath.Text{"", "a", "ab", maximum} {
		filepath.Volume_Name(value)
	}
	for _, one := range []struct {
		Value  filepath.Text
		Prefix filepath.Text
	}{
		{Value: "", Prefix: ""},
		{Value: "a", Prefix: "a"},
		{Value: "ab", Prefix: "ab"},
		{Value: maximum, Prefix: maximum},
		{Value: "a", Prefix: "ab"},
	} {
		filepath.Has_Prefix(one.Value, one.Prefix)
	}
	matched, match_error := filepath.Match(maximum, maximum)
	testify.No_Error(t, match_error, "maximum match")
	testify.True(t, bool(matched), "maximum match")
	var tail_storage [filepath.PATH_SIZE_MAXIMUM]byte
	tail_storage[0] = 'a'
	tail_storage[1] = '/'
	for index := 2; index < len(tail_storage); index++ {
		tail_storage[index] = 'a'
	}
	testify.True(t,
		bool(filepath.Is_Local(filepath.Text(string(tail_storage[:])))),
		"maximum component tail")
}

func maximum_split_directory_bound(t *testing.T, maximum filepath.Text) {
	directory, file := filepath.Split(maximum)
	testify.Equal(t, string(maximum), string(directory), "maximum split directory")
	testify.Equal(t, "", string(file), "maximum split empty file")
}

func absolute_size_bounds(t *testing.T) {
	var maximum_storage [filepath.PATH_SIZE_MAXIMUM]byte
	for index := range maximum_storage {
		maximum_storage[index] = 'a'
	}
	maximum_storage[0] = '/'
	maximum := filepath.Text(string(maximum_storage[:]))
	var storage [TEST_STORAGE_SIZE]byte
	for _, working_directory := range []filepath.Text{"", "a", "ab", maximum} {
		count, err := filepath.Absolute_Into(storage[:1], working_directory, "/")
		testify.No_Error(t, err, "absolute ignores working directory")
		testify.Equal(t, 1, int(count), "root absolute count")
	}
	count, err := filepath.Absolute_Into(storage[:2], "/", "/a")
	testify.No_Error(t, err, "two-byte absolute")
	testify.Equal(t, 2, int(count), "two-byte absolute")
	count, err = filepath.Absolute_Into(storage[:], "/", maximum)
	testify.No_Error(t, err, "maximum absolute")
	testify.Equal(t, filepath.PATH_SIZE_MAXIMUM, int(count), "maximum absolute")
	count, err = filepath.Absolute_Into(storage[:0], "", "a")
	testify.Error_Is(t, err, filepath.Error_Working_Directory, "invalid working directory")
	testify.Equal(t, 0, int(count), "invalid absolute count")
	for _, value := range []filepath.Text{"", "a", "ab"} {
		result_count, result_error := filepath.Absolute_Into(storage[:], "/", value)
		testify.No_Error(t, result_error, "absolute input boundary")
		testify.True(t, int(result_count) > 0, "absolute input boundary count")
	}
}

func relative_size_bounds(t *testing.T) {
	var maximum_storage [filepath.PATH_SIZE_MAXIMUM]byte
	for index := range maximum_storage {
		maximum_storage[index] = 'a'
	}
	maximum := filepath.Text(string(maximum_storage[:]))
	var storage [TEST_STORAGE_SIZE]byte
	count, err := filepath.Relative_Into(storage[:], ".", maximum)
	testify.No_Error(t, err, "maximum relative target")
	testify.Equal(t, filepath.PATH_SIZE_MAXIMUM, int(count), "maximum relative output")
	count, err = filepath.Relative_Into(storage[:2], "a", ".")
	testify.No_Error(t, err, "two-byte relative output")
	testify.Equal(t, 2, int(count), "two-byte relative output")
	count, err = filepath.Relative_Into(storage[:1], ".", "a")
	testify.No_Error(t, err, "one-byte relative output")
	testify.Equal(t, 1, int(count), "one-byte relative output")
	count, err = filepath.Relative_Into(storage[:], maximum, "b")
	testify.No_Error(t, err, "maximum relative base")
	testify.Equal(t, "../b", string(storage[:count]), "maximum relative base")
	count, err = filepath.Relative_Into(storage[:], ".", "ab")
	testify.No_Error(t, err, "two-byte relative target")
	testify.Equal(t, "ab", string(storage[:count]), "two-byte relative target")
	count, err = filepath.Relative_Into(storage[:], "/a", "/b")
	testify.No_Error(t, err, "one-byte common boundary")
	testify.Equal(t, "../b", string(storage[:count]), "one-byte common boundary")
	for _, one := range []struct {
		Base   filepath.Text
		Target filepath.Text
	}{
		{Base: "", Target: "a"},
		{Base: "a", Target: ""},
		{Base: "ab", Target: "a"},
	} {
		count, err = filepath.Relative_Into(storage[:], one.Base, one.Target)
		testify.No_Error(t, err, "relative input boundary")
		testify.True(t, int(count) > 0, "relative input boundary count")
	}
	common_prefix_relative_bound(t)
	result_too_large_bound(t)
}

func common_prefix_relative_bound(t *testing.T) {
	var base_storage [filepath.PATH_SIZE_MAXIMUM]byte
	var target_storage [filepath.PATH_SIZE_MAXIMUM]byte
	for index := 0; index < filepath.PATH_SIZE_MAXIMUM-2; index++ {
		base_storage[index] = 'a'
		target_storage[index] = 'a'
	}
	base_storage[filepath.PATH_SIZE_MAXIMUM-2] = '/'
	target_storage[filepath.PATH_SIZE_MAXIMUM-2] = '/'
	base_storage[filepath.PATH_SIZE_MAXIMUM-1] = 'b'
	target_storage[filepath.PATH_SIZE_MAXIMUM-1] = 'c'
	var destination [TEST_STORAGE_SIZE]byte
	count, err := filepath.Relative_Into(
		destination[:], filepath.Text(string(base_storage[:])),
		filepath.Text(string(target_storage[:])),
	)
	testify.No_Error(t, err, "maximum common prefix")
	testify.Equal(t, "../c", string(destination[:count]), "maximum common prefix")
}

func result_too_large_bound(t *testing.T) {
	var base_storage [filepath.PATH_SIZE_MAXIMUM]byte
	for index := range base_storage {
		if index%2 == 0 {
			base_storage[index] = 'a'
		} else {
			base_storage[index] = '/'
		}
	}
	base_storage[len(base_storage)-1] = 'a'
	var target_storage [filepath.PATH_SIZE_MAXIMUM]byte
	for index := range target_storage {
		target_storage[index] = 'b'
	}
	var destination [TEST_STORAGE_SIZE]byte
	count, err := filepath.Relative_Into(
		destination[:0], filepath.Text(string(base_storage[:])),
		filepath.Text(string(target_storage[:])),
	)
	testify.Error_Is(t, err, filepath.Error_Result_Too_Large, "large relative result")
	testify.Equal(t, 0, int(count), "large relative result count")
}

func collection_size_bounds(t *testing.T) {
	var list_storage [filepath.PATH_SIZE_MAXIMUM]byte
	for index := range list_storage {
		list_storage[index] = ':'
	}
	maximum_list := filepath.Text(string(list_storage[:]))
	var paths [filepath.PATH_COUNT_MAXIMUM]bytes.Text
	count := filepath.Split_List_Into(paths[:], maximum_list)
	testify.Equal(t, filepath.PATH_COUNT_MAXIMUM, int(count), "maximum path count")
	var maximum_elements [filepath.ELEMENT_COUNT_MAXIMUM]bytes.Text
	var storage [TEST_STORAGE_SIZE]byte
	testify.Equal(t, 0,
		int(filepath.Join_Into(storage[:0], filepath.Elements(maximum_elements[:]))),
		"maximum empty elements")
	filepath.Join_Into(storage[:1], filepath.Elements{"a"})
	filepath.Join_Into(storage[:2], filepath.Elements{"ab"})
	filepath.Split_List_Into(paths[:0], "")
	filepath.Split_List_Into(paths[:1], "a")
	filepath.Split_List_Into(paths[:2], "a:")
}

func destination_size_bounds(t *testing.T) {
	var storage [TEST_STORAGE_SIZE]byte
	filepath.Clean_Into(storage[:1], "")
	filepath.Clean_Into(storage[:2], "..")
	count, err := filepath.Localize_Into(storage[:0], "")
	testify.Error_Is(t, err, filepath.Error_Invalid_Path, "empty localize destination")
	testify.Equal(t, 0, int(count), "empty localize destination count")
	count, err = filepath.Localize_Into(storage[:1], ".")
	testify.No_Error(t, err, "one-byte localize destination")
	testify.Equal(t, 1, int(count), "one-byte localize destination count")
	count, err = filepath.Localize_Into(storage[:2], "ab")
	testify.No_Error(t, err, "two-byte localize destination")
	testify.Equal(t, 2, int(count), "two-byte localize destination count")
	filepath.To_Slash_Into(storage[:0], "")
	filepath.To_Slash_Into(storage[:1], "a")
	filepath.To_Slash_Into(storage[:2], "ab")
	filepath.From_Slash_Into(storage[:0], "")
	filepath.From_Slash_Into(storage[:1], "a")
	filepath.From_Slash_Into(storage[:2], "ab")
	filepath.Directory_Into(storage[:1], "a")
	filepath.Directory_Into(storage[:2], "../a")
	count, err = filepath.Relative_Into(storage[:0], "/a", "a")
	testify.Error_Is(t, err, filepath.Error_Root_Mismatch, "empty relative destination")
	testify.Equal(t, 0, int(count), "empty relative destination count")
	count, err = filepath.Relative_Into(storage[:1], "a", "a")
	testify.No_Error(t, err, "one-byte relative destination")
	testify.Equal(t, 1, int(count), "one-byte relative destination count")
	count, err = filepath.Relative_Into(storage[:2], "a", "a")
	testify.No_Error(t, err, "two-byte relative destination")
	testify.Equal(t, 1, int(count), "two-byte relative destination count")
	testify.Panics(t, func() { filepath.Relative_Into(storage[:0], "a", "a") })
	testify.Panics(t, func() { filepath.Directory_Into(storage[:0], "") })
	filepath.Directory_Into(storage[:1], "")
	filepath.Directory_Into(storage[:1], "a")
	filepath.Directory_Into(storage[:1], "/a")
}

func invalid_input_bounds(t *testing.T) {
	var oversized_storage [filepath.PATH_SIZE_MAXIMUM + 1]byte
	oversized := filepath.Text(string(oversized_storage[:]))
	var storage [TEST_STORAGE_SIZE]byte
	testify.Panics(t, func() { filepath.Clean_Into(storage[:], oversized) })
	testify.Panics(t, func() { filepath.Is_Local(oversized) })
	testify.Panics(t, func() { filepath.Localize_Into(storage[:], oversized) })
	testify.Panics(t, func() { filepath.To_Slash_Into(storage[:], oversized) })
	testify.Panics(t, func() { filepath.From_Slash_Into(storage[:], oversized) })
	testify.Panics(t, func() { filepath.Split_List_Into(nil, oversized) })
	testify.Panics(t, func() { filepath.Split(oversized) })
	testify.Panics(t, func() {
		filepath.Join_Into(storage[:], filepath.Elements{bytes.Text(oversized)})
	})
	testify.Panics(t, func() { filepath.Extension(oversized) })
	testify.Panics(t, func() { filepath.Is_Absolute(oversized) })
	testify.Panics(t, func() { filepath.Absolute_Into(storage[:], oversized, "/") })
	testify.Panics(t, func() { filepath.Absolute_Into(storage[:], "/", oversized) })
	testify.Panics(t, func() { filepath.Relative_Into(storage[:], oversized, "a") })
	testify.Panics(t, func() { filepath.Relative_Into(storage[:], "a", oversized) })
	testify.Panics(t, func() { filepath.Has_Prefix(oversized, "a") })
	testify.Panics(t, func() { filepath.Has_Prefix("a", oversized) })
	testify.Panics(t, func() { filepath.Base(oversized) })
	testify.Panics(t, func() { filepath.Directory_Into(storage[:], oversized) })
	testify.Panics(t, func() { filepath.Volume_Name(oversized) })
	testify.Panics(t, func() { filepath.Match(oversized, "a") })
	testify.Panics(t, func() { filepath.Match("*", oversized) })
	var too_many [filepath.PATH_COUNT_MAXIMUM + 1]bytes.Text
	testify.Panics(t, func() { filepath.Split_List_Into(too_many[:], "a") })
	testify.Panics(t, func() {
		filepath.Join_Into(storage[:], filepath.Elements(too_many[:]))
	})
	testify.Panics(t, func() { filepath.Clean_Into(storage[:0], "a") })
	testify.Panics(t, func() { filepath.Localize_Into(storage[:0], "a") })
	testify.Panics(t, func() { filepath.To_Slash_Into(storage[:0], "a") })
	testify.Panics(t, func() { filepath.From_Slash_Into(storage[:0], "a") })
	testify.Panics(t, func() { filepath.Split_List_Into(nil, "a") })
	testify.Panics(t, func() {
		filepath.Join_Into(storage[:1], filepath.Elements{"a", "b"})
	})
	testify.Panics(t, func() { filepath.Absolute_Into(storage[:0], "/", "a") })
	testify.Panics(t, func() { filepath.Relative_Into(storage[:0], ".", "a") })
	testify.Panics(t, func() { filepath.Directory_Into(storage[:0], "a") })
}

type allocation_fixture struct {
	Storage        filepath.Slice
	Remainder      filepath.Slice
	Link_Target    filepath.Slice
	Paths          filepath.Paths
	Elements       filepath.Elements
	Clean_Cases    clean_case_set
	Relative_Cases relative_case_set
	Match_Cases    match_case_set
	Count          filepath.Boundary
	Path_Count     filepath.Path_Count
	Nonempty       filepath.Nonempty_Count
	Directory      filepath.Directory_Count
	Text           filepath.Text
	Base           filepath.Base_Text
	Volume         filepath.Volume
	Boolean        filepath.Boolean
	Runner_Stopped bool
	Error          error
	IO             nbio.IO
	Glob           filepath.Glob_Runner
	Glob_Memory    filepath.Glob_Memory
	Matches        filepath.Path_Storage
	Walk           filepath.Walk_Runner
	Walk_State     allocation_walk_state
	Visitor_State  filepath.Walk_Visitor_State
	Walk_Memory    filepath.Walk_Memory
	Filesystem     allocation_filesystem_state
}

type allocation_walk_state struct {
	Count int
}

type allocation_filesystem_state struct {
	Read_Count int
	Open_Error error
	Read_Error error
}

type allocation_check struct {
	Name          string
	Call          func()
	Requires_Stop bool
}

func allocation_fixture_init(fixture *allocation_fixture) {
	fixture.Storage = make(filepath.Slice, TEST_STORAGE_SIZE)
	fixture.Remainder = make(filepath.Slice, TEST_STORAGE_SIZE)
	fixture.Link_Target = make(filepath.Slice, TEST_STORAGE_SIZE)
	fixture.Paths = make(filepath.Paths, filepath.PATH_COUNT_MAXIMUM)
	fixture.Elements = filepath.Elements{"a", "b"}
	fixture.Clean_Cases = clean_cases()
	fixture.Relative_Cases = relative_cases()
	fixture.Match_Cases = match_cases()
	fixture.Visitor_State = filepath.Walk_Visitor_State{
		Pointer: unsafe.Pointer(&fixture.Walk_State),
	}
	fixture.IO = nbio.IO{
		Storage: nbio.Storage{
			State:                           unsafe.Pointer(&fixture.Filesystem),
			Status_Procedure:                allocation_filesystem_status,
			Read_Link_Procedure:             allocation_filesystem_read_link,
			Open_At_Procedure:               allocation_filesystem_open,
			Get_Directory_Entries_Procedure: allocation_filesystem_read_directory,
		},
		Close_Procedure: allocation_filesystem_close,
	}
	fixture.Glob_Memory = filepath.Glob_Memory{
		Current: filepath.Glob_Current_Paths(specification_paths()),
		Next:    filepath.Glob_Next_Paths(specification_paths()),
		Entries: filepath.Directory_Entries(
			make([]nbio.Directory_Entry, TEST_FILESYSTEM_ENTRY_CAPACITY),
		),
		Directory_Buffer: filepath.Directory_Buffer(
			make(bytes.Slice, nbio.DIRECTORY_BUFFER_SIZE_MINIMUM),
		),
	}
	fixture.Walk_Memory = filepath.Walk_Memory{
		Queue:    filepath.Walk_Queue_Paths(specification_paths()),
		Children: filepath.Walk_Child_Paths(specification_paths()),
		Entries: filepath.Directory_Entries(
			make([]nbio.Directory_Entry, TEST_FILESYSTEM_ENTRY_CAPACITY),
		),
		Directory_Buffer: filepath.Directory_Buffer(
			make(bytes.Slice, nbio.DIRECTORY_BUFFER_SIZE_MINIMUM),
		),
	}
}

func allocation_lexical_checks(fixture *allocation_fixture) (checks []allocation_check) {
	checks = allocation_lexical_base_checks(fixture)
	return append(checks, allocation_lexical_corpus_checks(fixture)...)
}

func allocation_lexical_base_checks(fixture *allocation_fixture) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Clean_Into", Call: func() {
			fixture.Nonempty = filepath.Clean_Into(
				bytes.Slice(fixture.Storage), "a/../b",
			)
		}},
		{Name: "Is_Local", Call: func() { fixture.Boolean = filepath.Is_Local("a/b") }},
		{Name: "Localize_Into", Call: func() {
			fixture.Count, fixture.Error = filepath.Localize_Into(
				bytes.Slice(fixture.Storage), "a/b",
			)
		}},
		{Name: "To_Slash_Into", Call: func() {
			fixture.Count = filepath.To_Slash_Into(bytes.Slice(fixture.Storage), "a/b")
		}},
		{Name: "From_Slash_Into", Call: func() {
			fixture.Count = filepath.From_Slash_Into(
				bytes.Slice(fixture.Storage), "a/b",
			)
		}},
		{Name: "Split_List_Into", Call: func() {
			fixture.Path_Count = filepath.Split_List_Into(fixture.Paths[:], "a:b")
		}},
		{Name: "Split", Call: func() { fixture.Text, _ = filepath.Split("a/b") }},
		{Name: "Join_Into", Call: func() {
			fixture.Count = filepath.Join_Into(
				bytes.Slice(fixture.Storage),
				filepath.Elements(fixture.Elements[:]),
			)
		}},
		{Name: "Extension", Call: func() { fixture.Text = filepath.Extension("a.go") }},
		{Name: "Is_Absolute", Call: func() {
			fixture.Boolean = filepath.Is_Absolute("/a")
		}},
		{Name: "Absolute_Into", Call: func() {
			fixture.Count, fixture.Error = filepath.Absolute_Into(
				bytes.Slice(fixture.Storage), "/work", "a",
			)
		}},
		{Name: "Relative_Into", Call: func() {
			fixture.Count, fixture.Error = filepath.Relative_Into(
				bytes.Slice(fixture.Storage), "a", "b",
			)
		}},
		{Name: "Has_Prefix", Call: func() {
			fixture.Boolean = filepath.Has_Prefix("a/b", "a")
		}},
		{Name: "Base", Call: func() { fixture.Base = filepath.Base("a/b") }},
		{Name: "Directory_Into", Call: func() {
			fixture.Directory = filepath.Directory_Into(
				bytes.Slice(fixture.Storage), "a/b",
			)
		}},
		{Name: "Volume_Name", Call: func() {
			fixture.Volume = filepath.Volume_Name("a/b")
		}},
		{Name: "Match", Call: func() {
			fixture.Boolean, fixture.Error = filepath.Match("a*", "ab")
		}},
	}
}

func allocation_lexical_corpus_checks(
	fixture *allocation_fixture,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Clean_Into_Corpus", Call: func() {
			for _, one := range fixture.Clean_Cases {
				fixture.Nonempty = filepath.Clean_Into(
					bytes.Slice(fixture.Storage), one.Input,
				)
			}
		}},
		{Name: "Relative_Into_Corpus", Call: func() {
			for _, one := range fixture.Relative_Cases {
				fixture.Count, fixture.Error = filepath.Relative_Into(
					bytes.Slice(fixture.Storage), one.Base, one.Target,
				)
			}
		}},
		{Name: "Relative_Into_Errors", Call: func() {
			fixture.Count, fixture.Error = filepath.Relative_Into(
				bytes.Slice(fixture.Storage), "..", "a",
			)
			fixture.Count, fixture.Error = filepath.Relative_Into(
				bytes.Slice(fixture.Storage), "a", "/a",
			)
		}},
		{Name: "Localize_Into_Error", Call: func() {
			fixture.Count, fixture.Error = filepath.Localize_Into(
				bytes.Slice(fixture.Storage), "",
			)
		}},
		{Name: "Absolute_Into_Error", Call: func() {
			fixture.Count, fixture.Error = filepath.Absolute_Into(
				bytes.Slice(fixture.Storage), "relative", "a",
			)
		}},
		{Name: "Match_Corpus", Call: func() {
			for _, one := range fixture.Match_Cases {
				fixture.Boolean, fixture.Error = filepath.Match(
					one.Pattern, one.Name,
				)
			}
		}},
	}
}

func allocation_filesystem_checks(
	fixture *allocation_fixture,
) (checks []allocation_check) {
	checks = allocation_filesystem_base_checks(fixture)
	checks = append(checks, allocation_runner_flow_checks(fixture)...)
	return append(checks, allocation_eval_error_checks(fixture)...)
}

func allocation_filesystem_base_checks(
	fixture *allocation_fixture,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Glob_Runner_Init", Call: func() {
			fixture.Error = filepath.Glob_Runner_Init(
				&fixture.Glob, fixture.IO, "file", fixture.Glob_Memory,
			)
		}},
		{Name: "Glob_Runner_Rearm", Call: func() {
			filepath.Glob_Runner_Init(
				&fixture.Glob, fixture.IO, "file", fixture.Glob_Memory,
			)
			fixture.Boolean = filepath.Glob_Runner_Rearm(&fixture.Glob)
		}},
		{Name: "Glob_Runner_Work_Queued", Call: func() {
			fixture.Boolean = filepath.Glob_Runner_Work_Queued(&fixture.Glob)
		}},
		{Name: "Glob_Runner_Stopped", Call: func() {
			fixture.Boolean = filepath.Glob_Runner_Stopped(&fixture.Glob)
		}},
		{Name: "Glob_Runner_Status", Call: func() {
			fixture.Error = filepath.Glob_Runner_Status(&fixture.Glob)
		}},
		{Name: "Glob_Runner_Matches", Call: func() {
			filepath.Glob_Runner_Init(
				&fixture.Glob, fixture.IO, "file", fixture.Glob_Memory,
			)
			fixture.Matches = filepath.Glob_Runner_Matches(&fixture.Glob)
		}},
		{Name: "Walk_Runner_Init", Call: func() {
			fixture.Error = filepath.Walk_Runner_Init(
				&fixture.Walk, fixture.IO, "file",
				fixture.Visitor_State,
				allocation_walk_visit, fixture.Walk_Memory,
			)
		}},
		{Name: "Walk_Directory_Runner_Init", Call: func() {
			fixture.Error = filepath.Walk_Directory_Runner_Init(
				&fixture.Walk, fixture.IO, "file",
				fixture.Visitor_State,
				allocation_walk_visit, fixture.Walk_Memory,
			)
		}},
		{Name: "Walk_Runner_Rearm", Call: func() {
			filepath.Walk_Runner_Init(
				&fixture.Walk, fixture.IO, "file",
				fixture.Visitor_State,
				allocation_walk_visit, fixture.Walk_Memory,
			)
			fixture.Boolean = filepath.Walk_Runner_Rearm(&fixture.Walk)
		}},
		{Name: "Walk_Runner_Work_Queued", Call: func() {
			fixture.Boolean = filepath.Walk_Runner_Work_Queued(&fixture.Walk)
		}},
		{Name: "Walk_Runner_Stopped", Call: func() {
			fixture.Boolean = filepath.Walk_Runner_Stopped(&fixture.Walk)
		}},
		{Name: "Walk_Runner_Status", Call: func() {
			fixture.Error = filepath.Walk_Runner_Status(&fixture.Walk)
		}},
		{Name: "Eval_Symlinks_Into", Call: func() {
			fixture.Count, fixture.Error = filepath.Eval_Symlinks_Into(
				fixture.IO.Storage, fixture.Storage[:filepath.PATH_SIZE_MAXIMUM],
				fixture.Remainder[:filepath.PATH_SIZE_MAXIMUM],
				fixture.Link_Target[:filepath.PATH_SIZE_MAXIMUM], "file",
			)
		}},
	}
}

func allocation_runner_flow_checks(
	fixture *allocation_fixture,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Glob_Runner_Full", Requires_Stop: true, Call: func() {
			allocation_filesystem_reset(&fixture.Filesystem)
			allocation_glob_run(fixture, "dir/*.go")
		}},
		{Name: "Glob_Runner_Open_Error", Requires_Stop: true, Call: func() {
			allocation_filesystem_reset(&fixture.Filesystem)
			fixture.Filesystem.Open_Error = filepath.Error_Path_Absent
			allocation_glob_run(fixture, "dir/*.go")
		}},
		{Name: "Glob_Runner_Read_Error", Requires_Stop: true, Call: func() {
			allocation_filesystem_reset(&fixture.Filesystem)
			fixture.Filesystem.Read_Error = filepath.Error_Path_Absent
			allocation_glob_run(fixture, "dir/*.go")
		}},
		{Name: "Walk_Runner_Full", Requires_Stop: true, Call: func() {
			allocation_filesystem_reset(&fixture.Filesystem)
			allocation_walk_run(fixture, false)
		}},
		{Name: "Walk_Directory_Runner_Full", Requires_Stop: true, Call: func() {
			allocation_filesystem_reset(&fixture.Filesystem)
			allocation_walk_run(fixture, true)
		}},
		{Name: "Walk_Runner_Open_Error", Requires_Stop: true, Call: func() {
			allocation_filesystem_reset(&fixture.Filesystem)
			fixture.Filesystem.Open_Error = filepath.Error_Path_Absent
			allocation_walk_run(fixture, false)
		}},
		{Name: "Walk_Runner_Read_Error", Requires_Stop: true, Call: func() {
			allocation_filesystem_reset(&fixture.Filesystem)
			fixture.Filesystem.Read_Error = filepath.Error_Path_Absent
			allocation_walk_run(fixture, false)
		}},
	}
}

func allocation_eval_error_checks(
	fixture *allocation_fixture,
) (checks []allocation_check) {
	return []allocation_check{
		{Name: "Eval_Symlinks_Into_Missing", Call: func() {
			fixture.Count, fixture.Error = filepath.Eval_Symlinks_Into(
				fixture.IO.Storage, fixture.Storage[:filepath.PATH_SIZE_MAXIMUM],
				fixture.Remainder[:filepath.PATH_SIZE_MAXIMUM],
				fixture.Link_Target[:filepath.PATH_SIZE_MAXIMUM], "missing",
			)
		}},
		{Name: "Eval_Symlinks_Into_Cycle", Call: func() {
			fixture.Count, fixture.Error = filepath.Eval_Symlinks_Into(
				fixture.IO.Storage, fixture.Storage[:filepath.PATH_SIZE_MAXIMUM],
				fixture.Remainder[:filepath.PATH_SIZE_MAXIMUM],
				fixture.Link_Target[:filepath.PATH_SIZE_MAXIMUM], "cycle1",
			)
		}},
	}
}

func test_allocation(t *testing.T) {
	fixture := allocation_fixture{}
	allocation_fixture_init(&fixture)
	checks := allocation_lexical_checks(&fixture)
	checks = append(checks, allocation_filesystem_checks(&fixture)...)
	for _, check := range checks {
		t.Run(check.Name, func(t *testing.T) {
			testify.Zero_Allocation(t, check.Call)
			if check.Requires_Stop {
				testify.True(t, fixture.Runner_Stopped, check.Name)
			}
		})
	}
	if fixture.Count == -1 {
		t.Fatal("operations produced impossible observation")
	}
}

func allocation_walk_visit(
	visitor_state filepath.Walk_Visitor_State, _ bytes.Slice, _ filepath.Walk_Entry,
	visit_err error,
) (err error) {
	if visit_err != nil {
		return visit_err
	}
	state := (*allocation_walk_state)(visitor_state.Pointer)
	state.Count++
	return nil
}

func allocation_filesystem_reset(state *allocation_filesystem_state) {
	*state = allocation_filesystem_state{}
}

func allocation_filesystem_status(
	_ unsafe.Pointer, path string,
) (status nbio.File_Status, err error) {
	switch path {
	case ".", "dir":
		return nbio.File_Status{Exists: true, Mode: nbio.FILE_MODE_DIRECTORY}, nil
	case "file", "dir/file.go", "target":
		return nbio.File_Status{Exists: true}, nil
	case "link", "cycle1", "cycle2":
		return nbio.File_Status{Exists: true, Mode: nbio.FILE_MODE_SYMBOLIC_LINK}, nil
	default:
		return nbio.File_Status{}, nil
	}
}

func allocation_filesystem_read_link(
	_ unsafe.Pointer, path string, destination []byte,
) (count int, err error) {
	switch path {
	case "link":
		return copy(destination, "target"), nil
	case "cycle1":
		return copy(destination, "cycle2"), nil
	case "cycle2":
		return copy(destination, "cycle1"), nil
	default:
		return 0, filepath.Error_Path_Absent
	}
}

func allocation_filesystem_open(
	state unsafe.Pointer, completion *nbio.Completion, _ nbio.File, _ string,
	_ nbio.Open_At_Options, callback nbio.Callback,
) {
	filesystem := (*allocation_filesystem_state)(state)
	filesystem.Read_Count = 0
	completion.Data = 1
	completion.Error = filesystem.Open_Error
	callback(nbio.Completion_Handle(completion))
}

func allocation_filesystem_read_directory(
	state unsafe.Pointer, completion *nbio.Completion, _ nbio.File, _ []byte,
	entries []nbio.Directory_Entry, callback nbio.Callback,
) {
	filesystem := (*allocation_filesystem_state)(state)
	completion.Data = 0
	completion.Error = filesystem.Read_Error
	if completion.Error == nil {
		if filesystem.Read_Count == 0 {
			entries[0] = nbio.Directory_Entry{Name: "file.go"}
			completion.Data = 1
		}
	}
	filesystem.Read_Count++
	callback(nbio.Completion_Handle(completion))
}

func allocation_filesystem_close(
	_ unsafe.Pointer, completion *nbio.Completion, _ nbio.File,
	callback nbio.Callback,
) {
	completion.Data = 0
	completion.Error = nil
	callback(nbio.Completion_Handle(completion))
}

func allocation_glob_run(fixture *allocation_fixture, pattern filepath.Text) {
	fixture.Glob = filepath.Glob_Runner{}
	fixture.Runner_Stopped = false
	fixture.Error = filepath.Glob_Runner_Init(
		&fixture.Glob, fixture.IO, pattern, fixture.Glob_Memory,
	)
	if fixture.Error != nil {
		return
	}
	for range TEST_RUNNER_REARM_MAXIMUM {
		if filepath.Glob_Runner_Stopped(&fixture.Glob) {
			fixture.Runner_Stopped = true
			fixture.Matches = filepath.Glob_Runner_Matches(&fixture.Glob)
			fixture.Error = filepath.Glob_Runner_Status(&fixture.Glob)
			return
		}
		fixture.Boolean = filepath.Glob_Runner_Rearm(&fixture.Glob)
	}
}

func allocation_walk_run(fixture *allocation_fixture, directory_entry bool) {
	fixture.Walk = filepath.Walk_Runner{}
	fixture.Walk_State.Count = 0
	fixture.Runner_Stopped = false
	if directory_entry {
		fixture.Error = filepath.Walk_Directory_Runner_Init(
			&fixture.Walk, fixture.IO, "dir", fixture.Visitor_State,
			allocation_walk_visit, fixture.Walk_Memory,
		)
	} else {
		fixture.Error = filepath.Walk_Runner_Init(
			&fixture.Walk, fixture.IO, "dir", fixture.Visitor_State,
			allocation_walk_visit, fixture.Walk_Memory,
		)
	}
	if fixture.Error != nil {
		return
	}
	for range TEST_RUNNER_REARM_MAXIMUM {
		if filepath.Walk_Runner_Stopped(&fixture.Walk) {
			fixture.Runner_Stopped = true
			fixture.Error = filepath.Walk_Runner_Status(&fixture.Walk)
			return
		}
		fixture.Boolean = filepath.Walk_Runner_Rearm(&fixture.Walk)
	}
}
