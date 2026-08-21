// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by BSD-style license in Go source tree LICENSE file.

package filepath

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"local/james-orcales/shared/bytes"
	"local/james-orcales/shared/simulation/nbio"
	"local/james-orcales/shared/simulation/time"
	"local/james-orcales/shared/testify"
)

type standard_filepath_path_case struct {
	Path   Text
	Result string
}

type standard_filepath_local_case struct {
	Path  Text
	Local bool
}

type standard_filepath_localize_case struct {
	Path Text
	Want string
}

type standard_filepath_split_list_case struct {
	List  Text
	Paths []string
}

type standard_filepath_split_case struct {
	Path      Text
	Directory string
	File      string
}

type standard_filepath_join_case struct {
	Elements Elements
	Path     string
}

type standard_filepath_extension_case struct {
	Path      Text
	Extension string
}

type standard_filepath_absolute_case struct {
	Path     Text
	Absolute bool
}

type standard_filepath_relative_case struct {
	Base   Text
	Target Text
	Want   string
}

type standard_filepath_match_case struct {
	Pattern Text
	Name    Text
	Match   bool
	Error   error
}

func standard_filepath_clean_cases() (cases []standard_filepath_path_case) {
	return []standard_filepath_path_case{
		{Path: "abc", Result: "abc"},
		{Path: "abc/def", Result: "abc/def"},
		{Path: "a/b/c", Result: "a/b/c"},
		{Path: ".", Result: "."},
		{Path: "..", Result: ".."},
		{Path: "../..", Result: "../.."},
		{Path: "../../abc", Result: "../../abc"},
		{Path: "/abc", Result: "/abc"},
		{Path: "/", Result: "/"},
		{Path: "", Result: "."},
		{Path: "abc/", Result: "abc"},
		{Path: "abc/def/", Result: "abc/def"},
		{Path: "a/b/c/", Result: "a/b/c"},
		{Path: "./", Result: "."},
		{Path: "../", Result: ".."},
		{Path: "../../", Result: "../.."},
		{Path: "/abc/", Result: "/abc"},
		{Path: "abc//def//ghi", Result: "abc/def/ghi"},
		{Path: "abc//", Result: "abc"},
		{Path: "abc/./def", Result: "abc/def"},
		{Path: "/./abc/def", Result: "/abc/def"},
		{Path: "abc/.", Result: "abc"},
		{Path: "abc/def/ghi/../jkl", Result: "abc/def/jkl"},
		{Path: "abc/def/../ghi/../jkl", Result: "abc/jkl"},
		{Path: "abc/def/..", Result: "abc"},
		{Path: "abc/def/../..", Result: "."},
		{Path: "/abc/def/../..", Result: "/"},
		{Path: "abc/def/../../..", Result: ".."},
		{Path: "/abc/def/../../..", Result: "/"},
		{Path: "abc/def/../../../ghi/jkl/../../../mno", Result: "../../mno"},
		{Path: "/../abc", Result: "/abc"},
		{Path: "a/../b:/../../c", Result: "../c"},
		{Path: "abc/./../def", Result: "def"},
		{Path: "abc//./../def", Result: "def"},
		{Path: "abc/../../././../def", Result: "../../def"},
		{Path: "//abc", Result: "/abc"},
		{Path: "///abc", Result: "/abc"},
		{Path: "//abc//", Result: "/abc"},
	}
}

func standard_filepath_local_cases() (cases []standard_filepath_local_case) {
	return []standard_filepath_local_case{
		{Path: "", Local: false},
		{Path: ".", Local: true},
		{Path: "..", Local: false},
		{Path: "../a", Local: false},
		{Path: "/", Local: false},
		{Path: "/a", Local: false},
		{Path: "/a/../..", Local: false},
		{Path: "a", Local: true},
		{Path: "a/../a", Local: true},
		{Path: "a/", Local: true},
		{Path: "a/.", Local: true},
		{Path: "a/./b/./c", Local: true},
		{Path: "a/../b:/../../c", Local: false},
	}
}

func standard_filepath_localize_cases() (cases []standard_filepath_localize_case) {
	return []standard_filepath_localize_case{
		{Path: "", Want: ""},
		{Path: ".", Want: "."},
		{Path: "..", Want: ""},
		{Path: "a/..", Want: ""},
		{Path: "/", Want: ""},
		{Path: "/a", Want: ""},
		{Path: "a\xffb", Want: ""},
		{Path: "a/", Want: ""},
		{Path: "a/./b", Want: ""},
		{Path: "\x00", Want: ""},
		{Path: "a", Want: "a"},
		{Path: "a/b/c", Want: "a/b/c"},
		{Path: "#a", Want: "#a"},
		{Path: `a\b:c`, Want: `a\b:c`},
	}
}

func standard_filepath_slash_cases() (cases []standard_filepath_path_case) {
	return []standard_filepath_path_case{
		{Path: "", Result: ""},
		{Path: "/", Result: "/"},
		{Path: "/a/b", Result: "/a/b"},
		{Path: "a//b", Result: "a//b"},
	}
}

func standard_filepath_split_list_cases() (cases []standard_filepath_split_list_case) {
	return []standard_filepath_split_list_case{
		{List: "", Paths: []string{}},
		{List: "a:b", Paths: []string{"a", "b"}},
		{List: ":a:b", Paths: []string{"", "a", "b"}},
	}
}

func standard_filepath_split_cases() (cases []standard_filepath_split_case) {
	return []standard_filepath_split_case{
		{Path: "a/b", Directory: "a/", File: "b"},
		{Path: "a/b/", Directory: "a/b/", File: ""},
		{Path: "a/", Directory: "a/", File: ""},
		{Path: "a", Directory: "", File: "a"},
		{Path: "/", Directory: "/", File: ""},
	}
}

func standard_filepath_join_cases() (cases []standard_filepath_join_case) {
	return []standard_filepath_join_case{
		{Elements: Elements{}, Path: ""},
		{Elements: Elements{""}, Path: ""},
		{Elements: Elements{"/"}, Path: "/"},
		{Elements: Elements{"a"}, Path: "a"},
		{Elements: Elements{"a", "b"}, Path: "a/b"},
		{Elements: Elements{"a", ""}, Path: "a"},
		{Elements: Elements{"", "b"}, Path: "b"},
		{Elements: Elements{"/", "a"}, Path: "/a"},
		{Elements: Elements{"/", "a/b"}, Path: "/a/b"},
		{Elements: Elements{"/", ""}, Path: "/"},
		{Elements: Elements{"/a", "b"}, Path: "/a/b"},
		{Elements: Elements{"a", "/b"}, Path: "a/b"},
		{Elements: Elements{"/a", "/b"}, Path: "/a/b"},
		{Elements: Elements{"a/", "b"}, Path: "a/b"},
		{Elements: Elements{"a/", ""}, Path: "a"},
		{Elements: Elements{"", ""}, Path: ""},
		{Elements: Elements{"/", "a", "b"}, Path: "/a/b"},
		{Elements: Elements{"//", "a"}, Path: "/a"},
	}
}

func standard_filepath_extension_cases() (cases []standard_filepath_extension_case) {
	return []standard_filepath_extension_case{
		{Path: "path.go", Extension: ".go"},
		{Path: "path.pb.go", Extension: ".go"},
		{Path: "a.dir/b", Extension: ""},
		{Path: "a.dir/b.go", Extension: ".go"},
		{Path: "a.dir/", Extension: ""},
	}
}

func standard_filepath_base_cases() (cases []standard_filepath_path_case) {
	return []standard_filepath_path_case{
		{Path: "", Result: "."},
		{Path: ".", Result: "."},
		{Path: "/.", Result: "."},
		{Path: "/", Result: "/"},
		{Path: "////", Result: "/"},
		{Path: "x/", Result: "x"},
		{Path: "abc", Result: "abc"},
		{Path: "abc/def", Result: "def"},
		{Path: "a/b/.x", Result: ".x"},
		{Path: "a/b/c.", Result: "c."},
		{Path: "a/b/c.x", Result: "c.x"},
	}
}

func standard_filepath_directory_cases() (cases []standard_filepath_path_case) {
	return []standard_filepath_path_case{
		{Path: "", Result: "."},
		{Path: ".", Result: "."},
		{Path: "/.", Result: "/"},
		{Path: "/", Result: "/"},
		{Path: "/foo", Result: "/"},
		{Path: "x/", Result: "x"},
		{Path: "abc", Result: "."},
		{Path: "abc/def", Result: "abc"},
		{Path: "a/b/.x", Result: "a/b"},
		{Path: "a/b/c.", Result: "a/b"},
		{Path: "a/b/c.x", Result: "a/b"},
		{Path: "////", Result: "/"},
	}
}

func standard_filepath_absolute_cases() (cases []standard_filepath_absolute_case) {
	return []standard_filepath_absolute_case{
		{Path: "", Absolute: false},
		{Path: "/", Absolute: true},
		{Path: "/usr/bin/gcc", Absolute: true},
		{Path: "..", Absolute: false},
		{Path: "/a/../bb", Absolute: true},
		{Path: ".", Absolute: false},
		{Path: "./", Absolute: false},
		{Path: "lala", Absolute: false},
	}
}

func standard_filepath_absolute_path_cases() (cases []standard_filepath_path_case) {
	return []standard_filepath_path_case{
		{Path: ".", Result: "/tmp/a"},
		{Path: "b", Result: "/tmp/a/b"},
		{Path: "b/", Result: "/tmp/a/b"},
		{Path: "../a", Result: "/tmp/a"},
		{Path: "../a/b", Result: "/tmp/a/b"},
		{Path: "../a/b/./c/../../.././a", Result: "/tmp/a"},
		{Path: "../a/b/./c/../../.././a/", Result: "/tmp/a"},
		{Path: "/tmp", Result: "/tmp"},
		{Path: "/tmp/.", Result: "/tmp"},
		{Path: "/tmp/a/../a/b", Result: "/tmp/a/b"},
		{Path: "/tmp/a/b/c/../../.././a", Result: "/tmp/a"},
		{Path: "/tmp/a/b/c/../../.././a/", Result: "/tmp/a"},
	}
}

func standard_filepath_relative_cases() (cases []standard_filepath_relative_case) {
	return []standard_filepath_relative_case{
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
		{Base: "..", Target: ".", Want: "err"},
		{Base: "..", Target: "a", Want: "err"},
		{Base: "../..", Target: "..", Want: "err"},
		{Base: "a", Target: "/a", Want: "err"},
		{Base: "/a", Target: "a", Want: "err"},
	}
}

func standard_filepath_match_cases() (cases []standard_filepath_match_case) {
	return []standard_filepath_match_case{
		{Pattern: "abc", Name: "abc", Match: true},
		{Pattern: "*", Name: "abc", Match: true},
		{Pattern: "*c", Name: "abc", Match: true},
		{Pattern: "a*", Name: "a", Match: true},
		{Pattern: "a*", Name: "abc", Match: true},
		{Pattern: "a*", Name: "ab/c"},
		{Pattern: "a*/b", Name: "abc/b", Match: true},
		{Pattern: "a*/b", Name: "a/c/b"},
		{Pattern: "a*b*c*d*e*/f", Name: "axbxcxdxe/f", Match: true},
		{Pattern: "a*b*c*d*e*/f", Name: "axbxcxdxexxx/f", Match: true},
		{Pattern: "a*b*c*d*e*/f", Name: "axbxcxdxe/xxx/f"},
		{Pattern: "a*b*c*d*e*/f", Name: "axbxcxdxexxx/fff"},
		{Pattern: "a*b?c*x", Name: "abxbbxdbxebxczzx", Match: true},
		{Pattern: "a*b?c*x", Name: "abxbbxdbxebxczzy"},
		{Pattern: "ab[c]", Name: "abc", Match: true},
		{Pattern: "ab[b-d]", Name: "abc", Match: true},
		{Pattern: "ab[e-g]", Name: "abc"},
		{Pattern: "ab[^c]", Name: "abc"},
		{Pattern: "ab[^b-d]", Name: "abc"},
		{Pattern: "ab[^e-g]", Name: "abc", Match: true},
		{Pattern: "a\\*b", Name: "a*b", Match: true},
		{Pattern: "a\\*b", Name: "ab"},
		{Pattern: "a?b", Name: "a☺b", Match: true},
		{Pattern: "a[^a]b", Name: "a☺b", Match: true},
		{Pattern: "a???b", Name: "a☺b"},
		{Pattern: "a[^a][^a][^a]b", Name: "a☺b"},
		{Pattern: "[a-ζ]*", Name: "α", Match: true},
		{Pattern: "*[a-ζ]", Name: "A"},
		{Pattern: "a?b", Name: "a/b"},
		{Pattern: "a*b", Name: "a/b"},
		{Pattern: "[\\]a]", Name: "]", Match: true},
		{Pattern: "[\\-]", Name: "-", Match: true},
		{Pattern: "[x\\-]", Name: "x", Match: true},
		{Pattern: "[x\\-]", Name: "-", Match: true},
		{Pattern: "[x\\-]", Name: "z"},
		{Pattern: "[\\-x]", Name: "x", Match: true},
		{Pattern: "[\\-x]", Name: "-", Match: true},
		{Pattern: "[\\-x]", Name: "a"},
		{Pattern: "[]a]", Name: "]", Error: Error_Bad_Pattern},
		{Pattern: "[-]", Name: "-", Error: Error_Bad_Pattern},
		{Pattern: "[x-]", Name: "x", Error: Error_Bad_Pattern},
		{Pattern: "[x-]", Name: "-", Error: Error_Bad_Pattern},
		{Pattern: "[x-]", Name: "z", Error: Error_Bad_Pattern},
		{Pattern: "[-x]", Name: "x", Error: Error_Bad_Pattern},
		{Pattern: "[-x]", Name: "-", Error: Error_Bad_Pattern},
		{Pattern: "[-x]", Name: "a", Error: Error_Bad_Pattern},
		{Pattern: "\\", Name: "a", Error: Error_Bad_Pattern},
		{Pattern: "[a-b-c]", Name: "a", Error: Error_Bad_Pattern},
		{Pattern: "[", Name: "a", Error: Error_Bad_Pattern},
		{Pattern: "[^", Name: "a", Error: Error_Bad_Pattern},
		{Pattern: "[^bc", Name: "a", Error: Error_Bad_Pattern},
		{Pattern: "a[", Name: "a", Error: Error_Bad_Pattern},
		{Pattern: "a[", Name: "ab", Error: Error_Bad_Pattern},
		{Pattern: "a[", Name: "x", Error: Error_Bad_Pattern},
		{Pattern: "a/b[", Name: "x", Error: Error_Bad_Pattern},
		{Pattern: "*x", Name: "xxx", Match: true},
	}
}

// Test_Standard_Library_Clean ports upstream TestClean.
func Test_Standard_Library_Clean(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	for _, one := range standard_filepath_clean_cases() {
		count := Clean_Into(storage[:], one.Path)
		testify.Equal(t, one.Result, string(storage[:count]), "Clean_Into(%q)", one.Path)
		count = Clean_Into(storage[:], Text(one.Result))
		testify.Equal(t, one.Result, string(storage[:count]), "Clean_Into(%q)", one.Result)
	}
}

// Test_Standard_Library_Clean_Allocation ports upstream TestClean allocation check.
func Test_Standard_Library_Clean_Allocation(t *testing.T) {
	fixture := struct {
		Storage [PATH_SIZE_MAXIMUM]byte
		Path    Text
		Count   Nonempty_Count
	}{}
	for _, one := range standard_filepath_clean_cases() {
		fixture.Path = Text(one.Result)
		testify.Zero_Allocation(t, func() {
			fixture.Count = Clean_Into(fixture.Storage[:], fixture.Path)
		})
	}
	if fixture.Count == 0 {
		t.Fatal("Clean_Into returned impossible zero count")
	}
}

// Test_Standard_Library_Is_Local ports upstream TestIsLocal.
func Test_Standard_Library_Is_Local(t *testing.T) {
	for _, one := range standard_filepath_local_cases() {
		testify.Equal(t, one.Local, bool(Is_Local(one.Path)), "Is_Local(%q)", one.Path)
	}
}

// Test_Standard_Library_Localize ports upstream TestLocalize Unix cases.
func Test_Standard_Library_Localize(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	for _, one := range standard_filepath_localize_cases() {
		count, err := Localize_Into(storage[:], one.Path)
		if one.Want == "" {
			testify.Error_Is(t, err, Error_Invalid_Path, "Localize_Into(%q)", one.Path)
			testify.Equal(t, 0, int(count), "Localize_Into(%q) count", one.Path)
			continue
		}
		testify.No_Error(t, err, "Localize_Into(%q)", one.Path)
		testify.Equal(t, one.Want, string(storage[:count]), "Localize_Into(%q)", one.Path)
	}
}

// Test_Standard_Library_Slash_Conversion ports upstream TestFromAndToSlash Unix cases.
func Test_Standard_Library_Slash_Conversion(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	for _, one := range standard_filepath_slash_cases() {
		count := From_Slash_Into(storage[:], one.Path)
		testify.Equal(
			t, one.Result, string(storage[:count]), "From_Slash_Into(%q)", one.Path,
		)
		count = To_Slash_Into(storage[:], Text(one.Result))
		testify.Equal(
			t, string(one.Path), string(storage[:count]), "To_Slash_Into(%q)", one.Path,
		)
	}
}

// Test_Standard_Library_Split_List ports upstream TestSplitList Unix cases.
func Test_Standard_Library_Split_List(t *testing.T) {
	var storage [PATH_COUNT_MAXIMUM]bytes.Text
	for _, one := range standard_filepath_split_list_cases() {
		count := Split_List_Into(storage[:], one.List)
		testify.Equal(t, len(one.Paths), int(count), "Split_List_Into(%q) count", one.List)
		for index := range one.Paths {
			testify.Equal(
				t, one.Paths[index], string(storage[index]),
				"Split_List_Into(%q) position %d", one.List, index,
			)
		}
	}
}

// Test_Standard_Library_Split ports upstream TestSplit Unix cases.
func Test_Standard_Library_Split(t *testing.T) {
	for _, one := range standard_filepath_split_cases() {
		directory, file := Split(one.Path)
		testify.Equal(t, one.Directory, string(directory), "Split(%q) directory", one.Path)
		testify.Equal(t, one.File, string(file), "Split(%q) file", one.Path)
	}
}

// Test_Standard_Library_Join ports upstream TestJoin Unix cases.
func Test_Standard_Library_Join(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	for _, one := range standard_filepath_join_cases() {
		count := Join_Into(storage[:], one.Elements)
		testify.Equal(t, one.Path, string(storage[:count]), "Join_Into(%q)", one.Elements)
	}
}

// Test_Standard_Library_Extension ports upstream TestExt.
func Test_Standard_Library_Extension(t *testing.T) {
	for _, one := range standard_filepath_extension_cases() {
		testify.Equal(
			t, one.Extension, string(Extension(one.Path)), "Extension(%q)", one.Path,
		)
	}
}

// Test_Standard_Library_Base ports upstream TestBase Unix cases.
func Test_Standard_Library_Base(t *testing.T) {
	for _, one := range standard_filepath_base_cases() {
		testify.Equal(t, one.Result, string(Base(one.Path)), "Base(%q)", one.Path)
	}
}

// Test_Standard_Library_Directory ports upstream TestDir Unix cases.
func Test_Standard_Library_Directory(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	for _, one := range standard_filepath_directory_cases() {
		count := Directory_Into(storage[:], one.Path)
		testify.Equal(
			t, one.Result, string(storage[:count]), "Directory_Into(%q)", one.Path,
		)
	}
}

// Test_Standard_Library_Is_Absolute ports upstream TestIsAbs Unix cases.
func Test_Standard_Library_Is_Absolute(t *testing.T) {
	for _, one := range standard_filepath_absolute_cases() {
		testify.Equal(
			t, one.Absolute, bool(Is_Absolute(one.Path)),
			"Is_Absolute(%q)", one.Path,
		)
	}
}

// Test_Standard_Library_Absolute ports upstream TestAbs with working directory injected.
func Test_Standard_Library_Absolute(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	for _, one := range standard_filepath_absolute_path_cases() {
		count, err := Absolute_Into(storage[:], "/tmp/a", one.Path)
		testify.No_Error(t, err, "Absolute_Into(%q)", one.Path)
		testify.Equal(t, one.Result, string(storage[:count]), "Absolute_Into(%q)", one.Path)
	}
}

// Test_Standard_Library_Absolute_Empty ports upstream TestAbsEmptyString with working directory
// injected.
func Test_Standard_Library_Absolute_Empty(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	count, err := Absolute_Into(storage[:], "/tmp/a", "")
	testify.No_Error(t, err, "Absolute_Into empty path")
	testify.Equal(t, "/tmp/a", string(storage[:count]), "Absolute_Into empty path")
}

// Test_Standard_Library_Relative ports upstream TestRel Unix cases.
func Test_Standard_Library_Relative(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	for _, one := range standard_filepath_relative_cases() {
		count, err := Relative_Into(storage[:], one.Base, one.Target)
		if one.Want == "err" {
			testify.True(
				t, err != nil, "Relative_Into(%q, %q) error", one.Base, one.Target,
			)
			continue
		}
		testify.No_Error(t, err, "Relative_Into(%q, %q)", one.Base, one.Target)
		testify.Equal(
			t, one.Want, string(storage[:count]), "Relative_Into(%q, %q)",
			one.Base, one.Target,
		)
	}
}

// Test_Standard_Library_Match ports upstream TestMatch Unix cases.
func Test_Standard_Library_Match(t *testing.T) {
	for _, one := range standard_filepath_match_cases() {
		matched, err := Match(one.Pattern, one.Name)
		testify.Equal(t, one.Match, bool(matched), "Match(%q, %q)", one.Pattern, one.Name)
		testify.Equal(t, one.Error, err, "Match(%q, %q) error", one.Pattern, one.Name)
	}
}

// STANDARD_FILESYSTEM_NODE_COUNT counts nodes each largest fixture adds below root.
const STANDARD_FILESYSTEM_NODE_COUNT = 7

// STANDARD_FILESYSTEM_SERIAL_CAPACITY holds one live resource in serial fixture work.
const STANDARD_FILESYSTEM_SERIAL_CAPACITY = 1

// STANDARD_FILESYSTEM_NODE_CAPACITY leaves simulator-reserved half beside fixture nodes.
const STANDARD_FILESYSTEM_NODE_CAPACITY = 2 * STANDARD_FILESYSTEM_NODE_COUNT

// STANDARD_FILESYSTEM_PATH_CAPACITY lets traversal retain every fixture node.
const STANDARD_FILESYSTEM_PATH_CAPACITY = STANDARD_FILESYSTEM_NODE_COUNT

// STANDARD_FILESYSTEM_DESCRIPTOR_CAPACITY holds one serial fixture descriptor.
const STANDARD_FILESYSTEM_DESCRIPTOR_CAPACITY = STANDARD_FILESYSTEM_SERIAL_CAPACITY

// STANDARD_FILESYSTEM_OPERATION_CAPACITY holds one serial fixture operation.
const STANDARD_FILESYSTEM_OPERATION_CAPACITY = STANDARD_FILESYSTEM_DESCRIPTOR_CAPACITY

// STANDARD_FILESYSTEM_TIMELINE_CAPACITY holds one serial fixture completion.
const STANDARD_FILESYSTEM_TIMELINE_CAPACITY = STANDARD_FILESYSTEM_OPERATION_CAPACITY

// STANDARD_FILESYSTEM_EVENT_CAPACITY holds one virtual timer event.
const STANDARD_FILESYSTEM_EVENT_CAPACITY = STANDARD_FILESYSTEM_TIMELINE_CAPACITY

// STANDARD_FILESYSTEM_ENTRY_CAPACITY holds every child in largest fixture directory.
const STANDARD_FILESYSTEM_ENTRY_CAPACITY = STANDARD_FILESYSTEM_NODE_COUNT

// STANDARD_FILESYSTEM_DEADLINE bounds every virtual driver wait.
const STANDARD_FILESYSTEM_DEADLINE = time.MICROSECOND

// Test_Standard_Library_Glob ports upstream TestGlob and TestGlobError through injected storage.
func Test_Standard_Library_Glob(t *testing.T) {
	loop, driver := standard_filesystem_loop(11)
	standard_filesystem_directory(t, loop, driver, "stdlib_glob")
	standard_filesystem_directory(t, loop, driver, "stdlib_glob/filepath")
	standard_filesystem_file(t, loop, driver, "stdlib_glob/match.go")
	standard_filesystem_file(t, loop, driver, "stdlib_glob/match.txt")
	standard_filesystem_file(t, loop, driver, "stdlib_glob/filepath/match.go")
	standard_filesystem_directory(t, loop, driver, "filepath")
	standard_filesystem_file(t, loop, driver, "filepath/match.go")

	tests := []struct {
		Pattern Text
		Want    []string
	}{
		{Pattern: "stdlib_glob/match.go", Want: []string{"stdlib_glob/match.go"}},
		{Pattern: "stdlib_glob/mat?h.go", Want: []string{"stdlib_glob/match.go"}},
		{Pattern: "stdlib_glob/*.go", Want: []string{"stdlib_glob/match.go"}},
		{
			Pattern: "stdlib_glob/*",
			Want: []string{
				"stdlib_glob/filepath", "stdlib_glob/match.go",
				"stdlib_glob/match.txt",
			},
		},
		{
			Pattern: "stdlib_glob/*/match.go",
			Want:    []string{"stdlib_glob/filepath/match.go"},
		},
		{
			Pattern: "../*/match.go",
			Want:    []string{"../filepath/match.go", "../stdlib_glob/match.go"},
		},
		{Pattern: "stdlib_glob/no_match"},
		{Pattern: "stdlib_glob/*/no_match"},
	}
	for _, one := range tests {
		matches, err := standard_glob(t, loop, driver, one.Pattern)
		testify.No_Error(t, err, "Glob(%q)", one.Pattern)
		testify.Equal(t, len(one.Want), len(matches), "Glob(%q) count", one.Pattern)
		for index := range one.Want {
			testify.Equal(
				t, one.Want[index], string(matches[index]), "Glob(%q)", one.Pattern,
			)
		}
	}
	for _, pattern := range []Text{"[]", "stdlib_glob/[]"} {
		_, err := standard_glob(t, loop, driver, pattern)
		testify.Error_Is(t, err, Error_Bad_Pattern, "Glob(%q)", pattern)
	}
	standard_glob_symlinks(t)
	nbio.IO_Deinit(loop)
	nbio.Driver_Deinit(driver)
}

func standard_glob_symlinks(t *testing.T) {
	t.Helper()
	filesystem := standard_link_storage{Nodes: []standard_link_node{
		{Path: "link", Link: "target"},
		{Path: "broken", Link: "missing"},
	}}
	storage := nbio.Storage{
		State: unsafe.Pointer(&filesystem), Status_Procedure: standard_link_status,
		Read_Link_Procedure: standard_link_read,
	}
	for _, path := range []Text{"link", "broken"} {
		matches, err := standard_glob(t, nbio.IO{Storage: storage}, nbio.Driver{}, path)
		testify.No_Error(t, err)
		testify.Equal(t, 1, len(matches))
		testify.Equal(t, string(path), string(matches[0]))
	}
}

type standard_walk_state struct {
	Paths [STANDARD_FILESYSTEM_PATH_CAPACITY]string
	Count int
	Skip  string
	All   string
}

// Test_Standard_Library_Walk ports upstream TestWalk preorder and skip behavior.
func Test_Standard_Library_Walk(t *testing.T) {
	standard_walk_test(t, false)
}

// Test_Standard_Library_Walk_Directory ports upstream TestWalkDir preorder and skip behavior.
func Test_Standard_Library_Walk_Directory(t *testing.T) {
	standard_walk_test(t, true)
}

func standard_walk_test(t *testing.T, directory_entry bool) {
	t.Helper()
	loop, driver := standard_filesystem_loop(17)
	standard_filesystem_directory(t, loop, driver, "stdlib_walk")
	standard_filesystem_directory(t, loop, driver, "stdlib_walk/a")
	standard_filesystem_directory(t, loop, driver, "stdlib_walk/a/sub")
	standard_filesystem_directory(t, loop, driver, "stdlib_walk/b")
	standard_filesystem_file(t, loop, driver, "stdlib_walk/a/one")
	standard_filesystem_file(t, loop, driver, "stdlib_walk/a/sub/two")
	standard_filesystem_file(t, loop, driver, "stdlib_walk/b/three")

	state := standard_walk_state{}
	err := standard_walk(t, loop, driver, &state, directory_entry)
	testify.No_Error(t, err)
	want := []string{
		"stdlib_walk", "stdlib_walk/a", "stdlib_walk/a/one", "stdlib_walk/a/sub",
		"stdlib_walk/a/sub/two", "stdlib_walk/b", "stdlib_walk/b/three",
	}
	testify.Equal(t, len(want), state.Count)
	for index := range want {
		testify.Equal(t, want[index], state.Paths[index], "walk position %d", index)
	}

	state = standard_walk_state{Skip: "stdlib_walk/a"}
	err = standard_walk(t, loop, driver, &state, directory_entry)
	testify.No_Error(t, err)
	testify.Equal(t, 4, state.Count)
	testify.Equal(t, "stdlib_walk/b", state.Paths[2])

	state = standard_walk_state{Skip: "stdlib_walk/a/one"}
	err = standard_walk(t, loop, driver, &state, directory_entry)
	testify.No_Error(t, err)
	testify.Equal(t, 5, state.Count)
	testify.Equal(t, "stdlib_walk/b", state.Paths[3])

	state = standard_walk_state{All: "stdlib_walk/a/one"}
	err = standard_walk(t, loop, driver, &state, directory_entry)
	testify.No_Error(t, err)
	testify.Equal(t, 3, state.Count)

	state = standard_walk_state{}
	err = standard_walk_root(
		t, loop, driver, &state, directory_entry, "stdlib_walk/missing",
	)
	testify.Error_Is(t, err, Error_Path_Absent)
	standard_walk_symlink_root(t, directory_entry)
	nbio.IO_Deinit(loop)
	nbio.Driver_Deinit(driver)
}

func standard_walk_symlink_root(t *testing.T, directory_entry bool) {
	t.Helper()
	filesystem := standard_link_storage{Nodes: []standard_link_node{
		{Path: "link", Link: "target"},
		{Path: "target", Directory: true},
	}}
	storage := nbio.Storage{
		State: unsafe.Pointer(&filesystem), Status_Procedure: standard_link_status,
		Read_Link_Procedure: standard_link_read,
	}
	state := standard_walk_state{}
	err := standard_walk_root(
		t, nbio.IO{Storage: storage}, nbio.Driver{}, &state, directory_entry, "link",
	)
	testify.No_Error(t, err)
	testify.Equal(t, 1, state.Count)
	testify.Equal(t, "link", state.Paths[0])
}

func standard_walk_visit(
	state *standard_walk_state, path bytes.Slice, _ Walk_Entry, visit_err error,
) (err error) {
	if visit_err != nil {
		return visit_err
	}
	value := string(path)
	state.Paths[state.Count] = value
	state.Count++
	if value == state.Skip {
		return Skip_Directory
	}
	if value == state.All {
		return Skip_All
	}
	return nil
}

func standard_walk(
	t *testing.T, loop nbio.IO, driver nbio.Driver, state *standard_walk_state,
	directory_entry bool,
) (err error) {
	return standard_walk_root(t, loop, driver, state, directory_entry, "stdlib_walk")
}

func standard_walk_root(
	t *testing.T, loop nbio.IO, driver nbio.Driver, state *standard_walk_state,
	directory_entry bool, root Text,
) (err error) {
	t.Helper()
	memory := Walk_Memory{
		Queue:    Walk_Queue_Paths(standard_path_storage()),
		Children: Walk_Child_Paths(standard_path_storage()),
		Entries: Directory_Entries(
			make([]nbio.Directory_Entry, STANDARD_FILESYSTEM_ENTRY_CAPACITY),
		),
		Directory_Buffer: Directory_Buffer(
			make(bytes.Slice, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM),
		),
	}
	runner := Walk_Runner[standard_walk_state]{}
	if directory_entry {
		err = Walk_Directory_Runner_Init(
			&runner, loop, root, state, standard_walk_visit, memory,
		)
	} else {
		err = Walk_Runner_Init(
			&runner, loop, root, state, standard_walk_visit, memory,
		)
	}
	if err != nil {
		return err
	}
	standard_walk_drive(t, driver, &runner)
	return Walk_Runner_Status(&runner)
}

func standard_walk_drive[State any](
	t *testing.T, driver nbio.Driver, runner *Walk_Runner[State],
) {
	t.Helper()
	for !bool(Walk_Runner_Stopped(runner)) {
		for Walk_Runner_Rearm(runner) {
		}
		if bool(Walk_Runner_Stopped(runner)) {
			break
		}
		completed, err := nbio.Driver_Run_Until(
			driver, STANDARD_FILESYSTEM_DEADLINE,
			func() (done bool) { return bool(Walk_Runner_Work_Queued(runner)) },
		)
		testify.No_Error(t, err)
		testify.True(t, completed)
	}
}

func standard_glob(
	t *testing.T, loop nbio.IO, driver nbio.Driver, pattern Text,
) (matches Path_Storage, err error) {
	t.Helper()
	runner := Glob_Runner{}
	err = Glob_Runner_Init(&runner, loop, pattern, Glob_Memory{
		Current: Glob_Current_Paths(standard_path_storage()),
		Next:    Glob_Next_Paths(standard_path_storage()),
		Entries: Directory_Entries(
			make([]nbio.Directory_Entry, STANDARD_FILESYSTEM_ENTRY_CAPACITY),
		),
		Directory_Buffer: Directory_Buffer(
			make(bytes.Slice, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM),
		),
	})
	if err != nil {
		return nil, err
	}
	for !bool(Glob_Runner_Stopped(&runner)) {
		for Glob_Runner_Rearm(&runner) {
		}
		if bool(Glob_Runner_Stopped(&runner)) {
			break
		}
		completed, drive_err := nbio.Driver_Run_Until(
			driver, STANDARD_FILESYSTEM_DEADLINE,
			func() (done bool) { return bool(Glob_Runner_Work_Queued(&runner)) },
		)
		testify.No_Error(t, drive_err)
		testify.True(t, completed)
	}
	return Glob_Runner_Matches(&runner), Glob_Runner_Status(&runner)
}

func standard_path_storage() (paths Path_Storage) {
	paths = make(Path_Storage, STANDARD_FILESYSTEM_PATH_CAPACITY)
	for index := range paths {
		paths[index] = make(Slice, PATH_SIZE_MAXIMUM)
	}
	return paths
}

// Walks read no clock, so one view slot satisfies the simulator's bound.
const STANDARD_FILESYSTEM_CLOCK_CAPACITY = 1

func standard_filesystem_loop(seed uint64) (loop nbio.IO, driver nbio.Driver) {
	state := nbio.Sim{}
	queue := [STANDARD_FILESYSTEM_TIMELINE_CAPACITY]*nbio.Completion{}
	events := [STANDARD_FILESYSTEM_EVENT_CAPACITY]nbio.Virtual_Event{}
	clocks := [STANDARD_FILESYSTEM_CLOCK_CAPACITY]nbio.Sim_Clock{}
	nodes := [STANDARD_FILESYSTEM_NODE_CAPACITY]nbio.Sim_Node{}
	descriptors := [STANDARD_FILESYSTEM_DESCRIPTOR_CAPACITY]nbio.Sim_Descriptor{}
	operations := [STANDARD_FILESYSTEM_OPERATION_CAPACITY]nbio.Sim_Operation{}
	return nbio.New_Simulated_IO(&state, seed, time.NANOSECOND, nbio.Sim_Memory{
		Nodes:       nodes[:],
		Descriptors: descriptors[:],
		Operations:  operations[:],
		Queue:       queue[:],
		Events:      events[:],
		Clocks:      clocks[:],
	})
}

func standard_filesystem_directory(
	t *testing.T, loop nbio.IO, driver nbio.Driver, path string,
) {
	t.Helper()
	done := false
	completion := nbio.Completion{}
	nbio.Storage_Mkdir_At(
		loop.Storage, &completion, nbio.DIRECTORY_CURRENT, path, 0o755,
		func(completed *nbio.Completion) {
			testify.No_Error(t, completed.Error, path)
			done = true
		},
	)
	nbio.Driver_Run_Until(
		driver, STANDARD_FILESYSTEM_DEADLINE, func() (ready bool) { return done },
	)
	testify.True(t, done, path)
}

func standard_filesystem_file(
	t *testing.T, loop nbio.IO, driver nbio.Driver, path string,
) {
	t.Helper()
	done := false
	file := nbio.File(-1)
	completion := nbio.Completion{}
	nbio.Storage_Open_At(
		loop.Storage, &completion, nbio.DIRECTORY_CURRENT, path,
		nbio.Open_At_Options{
			Access: nbio.OPEN_WRITE_ONLY, Create: true, Permissions: 0o600,
		},
		func(completed *nbio.Completion) {
			testify.No_Error(t, completed.Error, path)
			file = nbio.File(completed.Data)
			done = true
		},
	)
	nbio.Driver_Run_Until(
		driver, STANDARD_FILESYSTEM_DEADLINE, func() (ready bool) { return done },
	)
	testify.True(t, done, path)
	standard_filesystem_close(t, loop, driver, file)
}

func standard_filesystem_close(
	t *testing.T, loop nbio.IO, driver nbio.Driver, file nbio.File,
) {
	t.Helper()
	done := false
	completion := nbio.Completion{}
	nbio.IO_Close(loop, &completion, file, func(completed *nbio.Completion) {
		testify.No_Error(t, completed.Error)
		done = true
	})
	nbio.Driver_Run_Until(
		driver, STANDARD_FILESYSTEM_DEADLINE, func() (ready bool) { return done },
	)
	testify.True(t, done)
}

type standard_link_node struct {
	Path      string
	Directory bool
	Link      string
}

type standard_link_storage struct {
	Nodes []standard_link_node
}

// Kind of one fixture node as portable mode. Link wins over directory, because lstat reports the
// link itself and never the kind behind it.
func standard_node_mode(node standard_link_node) (mode nbio.File_Mode) {
	if node.Link != "" {
		return nbio.FILE_MODE_SYMBOLIC_LINK
	}
	if node.Directory {
		return nbio.FILE_MODE_DIRECTORY
	}
	return 0
}

// Test_Standard_Library_Eval_Symlinks ports upstream Unix relative, absolute, chained, missing,
// and cyclic link cases through injected storage.
func Test_Standard_Library_Eval_Symlinks(t *testing.T) {
	filesystem := standard_link_storage{Nodes: []standard_link_node{
		{Path: "test", Directory: true},
		{Path: "test/dir", Directory: true},
		{Path: "test/link1", Link: "../test"},
		{Path: "test/link2", Link: "dir"},
		{Path: "test/dir/link3", Link: "../../"},
		{Path: "test/linkabs", Link: "/"},
		{Path: "test/link4", Link: "../test2"},
		{Path: "test2", Link: "test/dir"},
		{Path: "src", Directory: true},
		{Path: "src/pool", Directory: true},
		{Path: "src/pool/test", Directory: true},
		{Path: "src/versions", Directory: true},
		{Path: "src/versions/current", Link: "../../version"},
		{Path: "src/versions/v1", Directory: true},
		{Path: "src/versions/v1/modules", Directory: true},
		{Path: "src/versions/v1/modules/test", Link: "../../../pool/test"},
		{Path: "version", Link: "src/versions/v1"},
		{Path: "chain", Link: "test/link2"},
		{Path: "dangling", Link: "missing"},
		{Path: "cycle1", Link: "cycle2"},
		{Path: "cycle2", Link: "cycle1"},
	}}
	storage := nbio.Storage{
		State: unsafe.Pointer(&filesystem), Status_Procedure: standard_link_status,
		Read_Link_Procedure: standard_link_read,
	}
	tests := []struct {
		Path Text
		Want string
	}{
		{Path: "test", Want: "test"},
		{Path: "test/dir/../..", Want: "."},
		{Path: "test/link1", Want: "test"},
		{Path: "test/link2", Want: "test/dir"},
		{Path: "test/link1/dir", Want: "test/dir"},
		{Path: "test/link2/..", Want: "test"},
		{Path: "test/dir/link3", Want: "."},
		{Path: "test/link2/link3/test", Want: "test"},
		{Path: "test/linkabs", Want: "/"},
		{Path: "test/link4/..", Want: "test"},
		{Path: "src/versions/current/modules/test", Want: "src/pool/test"},
		{Path: "chain", Want: "test/dir"},
		{Path: ".", Want: "."},
	}
	var destination [PATH_SIZE_MAXIMUM]byte
	var scratch [PATH_SIZE_MAXIMUM]byte
	var link_target [PATH_SIZE_MAXIMUM]byte
	for _, one := range tests {
		count, err := Eval_Symlinks_Into(
			storage, destination[:], scratch[:], link_target[:], one.Path,
		)
		testify.No_Error(t, err, "Eval_Symlinks_Into(%q)", one.Path)
		testify.Equal(t, one.Want, string(destination[:count]),
			"Eval_Symlinks_Into(%q)", one.Path)
	}
	_, err := Eval_Symlinks_Into(
		storage, destination[:], scratch[:], link_target[:], "missing",
	)
	testify.Error_Is(t, err, Error_Path_Absent)
	_, err = Eval_Symlinks_Into(
		storage, destination[:], scratch[:], link_target[:], "dangling",
	)
	testify.Error_Is(t, err, Error_Path_Absent)
	_, err = Eval_Symlinks_Into(
		storage, destination[:], scratch[:], link_target[:], "cycle1",
	)
	testify.Error_Is(t, err, Error_Too_Many_Links)
}

func standard_link_status(
	state unsafe.Pointer, path string,
) (status nbio.File_Status, err error) {
	filesystem := (*standard_link_storage)(state)
	for _, node := range filesystem.Nodes {
		if node.Path != path {
			continue
		}
		return nbio.File_Status{Exists: true, Mode: standard_node_mode(node)}, nil
	}
	if path == "." {
		return nbio.File_Status{Exists: true, Mode: nbio.FILE_MODE_DIRECTORY}, nil
	}
	if path == "/" {
		return nbio.File_Status{Exists: true, Mode: nbio.FILE_MODE_DIRECTORY}, nil
	}
	return nbio.File_Status{}, nil
}

func standard_link_read(
	state unsafe.Pointer, path string, destination []byte,
) (count int, err error) {
	filesystem := (*standard_link_storage)(state)
	for _, node := range filesystem.Nodes {
		if node.Path != path {
			continue
		}
		if node.Link == "" {
			return 0, nbio.Not_Symbolic_Link
		}
		return copy(destination, node.Link), nil
	}
	return 0, Error_Path_Absent
}

// Test_Standard_Library_Filesystem_Bounds reaches complete input domains on new exported
// filesystem compositions without driving async work.
func Test_Standard_Library_Filesystem_Bounds(t *testing.T) {
	maximum := make([]byte, PATH_SIZE_MAXIMUM)
	for index := range maximum {
		maximum[index] = 'a'
	}
	values := []Text{"", "a", "aa", Text(string(maximum))}
	capacities := []int{1, 2, FILESYSTEM_PATH_COUNT_MAXIMUM, 1}
	storage := nbio.Storage{Status_Procedure: standard_existing_status}
	for index := range values {
		glob_memory := Glob_Memory{
			Current: Glob_Current_Paths(
				standard_path_storage_count(capacities[index]),
			),
			Next: Glob_Next_Paths(
				standard_path_storage_count(capacities[index]),
			),
			Entries:          Directory_Entries(make([]nbio.Directory_Entry, 1)),
			Directory_Buffer: Directory_Buffer(make([]byte, 1)),
		}
		glob := Glob_Runner{}
		testify.No_Error(t, Glob_Runner_Init(
			&glob, nbio.IO{Storage: storage}, values[index], glob_memory,
		))

		walk_memory := Walk_Memory{
			Queue: Walk_Queue_Paths(
				standard_path_storage_count(capacities[index]),
			),
			Children: Walk_Child_Paths(
				standard_path_storage_count(capacities[index]),
			),
			Entries:          Directory_Entries(make([]nbio.Directory_Entry, 1)),
			Directory_Buffer: Directory_Buffer(make([]byte, 1)),
		}
		walk := Walk_Runner[standard_walk_state]{}
		walk_state := standard_walk_state{}
		testify.No_Error(t, Walk_Runner_Init(
			&walk, nbio.IO{Storage: storage}, values[index], &walk_state,
			standard_walk_visit, walk_memory,
		))

		var destination [PATH_SIZE_MAXIMUM]byte
		var remainder [PATH_SIZE_MAXIMUM]byte
		var link_target [PATH_SIZE_MAXIMUM]byte
		count, err := Eval_Symlinks_Into(
			storage, destination[:], remainder[:], link_target[:], values[index],
		)
		testify.No_Error(t, err)
		if len(values[index]) == 0 {
			testify.Equal(t, 1, int(count))
		} else {
			testify.Equal(t, len(values[index]), int(count))
		}
	}
	standard_glob_state_bounds(values)
	standard_walk_state_bounds(values)
	standard_eval_state_bounds(values)
	standard_path_helper_bounds(values)
	standard_part_tail_bound(t)
	standard_directory_buffer_bound(t)
}

func standard_part_tail_bound(t *testing.T) {
	t.Helper()
	testify.Not_Panics(t, func() {
		leading_separator := make([]byte, PATH_SIZE_MAXIMUM)
		leading_separator[0] = SEPARATOR
		for index := 1; index < len(leading_separator); index++ {
			leading_separator[index] = 'a'
		}
		next_part(Nonempty_Text(string(leading_separator)))
	})
}

func standard_directory_buffer_bound(t *testing.T) {
	t.Helper()
	testify.Panics(t, func() {
		var glob Glob_Runner
		Glob_Runner_Init(&glob, nbio.IO{}, "", Glob_Memory{
			Current: Glob_Current_Paths(standard_path_storage_count(1)),
			Next:    Glob_Next_Paths(standard_path_storage_count(1)),
			Entries: Directory_Entries(
				make([]nbio.Directory_Entry, nbio.DIRECTORY_BUFFER_SIZE_MINIMUM),
			),
			Directory_Buffer: Directory_Buffer(
				make([]byte, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM+1),
			),
		})
	})
}

func standard_path_storage_count(count int) (paths Path_Storage) {
	paths = make(Path_Storage, count)
	if count == 0 {
		return paths
	}
	paths[0] = make(Slice, PATH_SIZE_MAXIMUM)
	return paths
}

func standard_boundary(operation func()) {
	defer func() {
		recover()
	}()
	operation()
}

func standard_glob_state_bounds(values []Text) {
	counts := []int{0, 1, 2, FILESYSTEM_PATH_COUNT_MAXIMUM}
	buffers := []int{0, 1, 2, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM}
	phases := []Glob_Phase{
		GLOB_PHASE_IDLE, GLOB_PHASE_OPEN, GLOB_PHASE_READ, GLOB_PHASE_CLOSE,
	}
	for index := range values {
		runner := Glob_Runner{
			Pattern: values[index],
			Current: Glob_Current_Paths(make(Path_Storage, counts[index])),
			Next:    Glob_Next_Paths(make(Path_Storage, counts[index])),
			Entries: Directory_Entries(
				make([]nbio.Directory_Entry, counts[index]),
			),
			Directory_Buffer: Directory_Buffer(make([]byte, buffers[index])),
			Current_Count:    Glob_Current_Count(counts[index]),
			Next_Count:       Glob_Next_Count(counts[index]),
			Candidate_Index:  Glob_Candidate_Index(counts[index]),
			Segment_Start:    Glob_Segment_Start(len(values[index])),
			Segment_End:      Glob_Segment_End(len(values[index])),
			Segment_Final:    Glob_Segment_Final(index%2 == 1),
			Segment_Meta:     Glob_Segment_Meta(index%2 == 0),
			Phase:            phases[index],
			Work_Ready:       Glob_Work_Ready(index%2 == 1),
			Done:             Glob_Done(index%2 == 0),
		}
		standard_glob_operation_bounds(runner, counts[index])
		standard_glob_init_bounds(runner, counts[index], buffers[index])
	}
	maximum := Glob_Runner{
		Current: Glob_Current_Paths(
			make(Path_Storage, FILESYSTEM_PATH_COUNT_MAXIMUM),
		),
		Current_Count: FILESYSTEM_PATH_COUNT_MAXIMUM,
		Done:          true,
	}
	standard_boundary(func() { Glob_Runner_Matches(&maximum) })
	available := Glob_Runner{
		Loop: nbio.IO{Storage: nbio.Storage{
			Status_Procedure: standard_existing_status,
		}},
		Current:          Glob_Current_Paths(Path_Storage{Slice{}}),
		Current_Count:    1,
		Segment_Meta:     true,
		Directory_Buffer: Directory_Buffer{0},
		Entries:          Directory_Entries{{}},
	}
	standard_boundary(func() { glob_meta_begin(&available) })
}

func standard_glob_operation_bounds(runner Glob_Runner, count int) {
	standard_boundary(func() { value := runner; Glob_Runner_Rearm(&value) })
	standard_boundary(func() { value := runner; Glob_Runner_Work_Queued(&value) })
	standard_boundary(func() { value := runner; Glob_Runner_Stopped(&value) })
	standard_boundary(func() { value := runner; Glob_Runner_Status(&value) })
	standard_boundary(func() { value := runner; Glob_Runner_Matches(&value) })
	standard_boundary(func() { value := runner; glob_completion_apply(&value) })
	standard_boundary(func() { value := runner; glob_meta_begin(&value) })
	standard_boundary(func() { value := runner; glob_directory_read(&value) })
	standard_boundary(func() { value := runner; glob_directory_close(&value) })
	standard_boundary(func() {
		value := runner
		glob_entries_apply(&value, Path_Count(count))
	})
	standard_boundary(func() { value := runner; glob_literal_apply(&value) })
	standard_boundary(func() { value := runner; glob_generation_finish(&value) })
	standard_boundary(func() { value := runner; glob_segment_advance(&value) })
}

func standard_glob_init_bounds(runner Glob_Runner, count int, buffer_size int) {
	memory := Glob_Memory{
		Current: Glob_Current_Paths(standard_path_storage_count(count)),
		Next:    Glob_Next_Paths(standard_path_storage_count(count)),
		Entries: Directory_Entries(make([]nbio.Directory_Entry, count)),
		Directory_Buffer: Directory_Buffer(
			make([]byte, buffer_size),
		),
	}
	standard_boundary(func() {
		Glob_Runner_Init(
			&runner, nbio.IO{Storage: nbio.Storage{
				Status_Procedure: standard_existing_status,
			}}, runner.Pattern, memory,
		)
	})
}

func standard_walk_state_bounds(values []Text) {
	counts := []int{0, 1, 2, FILESYSTEM_PATH_COUNT_MAXIMUM}
	buffers := []int{0, 1, 2, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM}
	phases := []Walk_Phase{
		WALK_PHASE_IDLE, WALK_PHASE_OPEN, WALK_PHASE_READ, WALK_PHASE_CLOSE,
	}
	for index := range values {
		state := standard_walk_state{}
		runner := Walk_Runner[standard_walk_state]{
			Visitor_State: &state, Visitor: standard_walk_visit,
			Queue:    Walk_Queue_Paths(make(Path_Storage, counts[index])),
			Children: Walk_Child_Paths(make(Path_Storage, counts[index])),
			Entries: Directory_Entries(
				make([]nbio.Directory_Entry, counts[index]),
			),
			Directory_Buffer: Directory_Buffer(make([]byte, buffers[index])),
			Queue_Count:      Walk_Queue_Count(counts[index]),
			Children_Count:   Walk_Child_Count(counts[index]),
			Directory_Path_Count: Boundary(
				len(values[index]),
			),
			Phase:      phases[index],
			Work_Ready: Walk_Work_Ready(index%2 == 1),
			Done:       Walk_Done(index%2 == 0),
		}
		standard_walk_operation_bounds(runner, values[index], counts[index])
		standard_walk_init_bounds(runner, values[index], counts[index], buffers[index])
	}
}

func standard_walk_operation_bounds(
	runner Walk_Runner[standard_walk_state], value Text, count int,
) {
	path := Slice(value)
	status := nbio.File_Status{Exists: true, Mode: nbio.FILE_MODE_DIRECTORY}
	standard_boundary(func() { value := runner; Walk_Runner_Rearm(&value) })
	standard_boundary(func() { value := runner; Walk_Runner_Work_Queued(&value) })
	standard_boundary(func() { value := runner; Walk_Runner_Stopped(&value) })
	standard_boundary(func() { value := runner; Walk_Runner_Status(&value) })
	standard_boundary(func() { value := runner; walk_visit_next(&value) })
	standard_boundary(func() {
		value := runner
		walk_visit_result(&value, path, status, Skip_Directory)
	})
	standard_boundary(func() { value := runner; walk_completion_apply(&value) })
	standard_boundary(func() { value := runner; walk_directory_read(&value) })
	standard_boundary(func() { value := runner; walk_directory_close(&value) })
	standard_boundary(func() {
		value := runner
		walk_directory_error(&value, Error_Path_Absent)
	})
	standard_boundary(func() {
		value := runner
		walk_entries_collect(&value, Path_Count(count))
	})
	standard_boundary(func() { value := runner; walk_children_push(&value) })
	standard_boundary(func() {
		value := runner
		walk_queue_siblings_remove(&value, path)
	})
}

func standard_walk_init_bounds(
	runner Walk_Runner[standard_walk_state], root Text, count int,
	buffer_size int,
) {
	memory := Walk_Memory{
		Queue:    Walk_Queue_Paths(standard_path_storage_count(count)),
		Children: Walk_Child_Paths(standard_path_storage_count(count)),
		Entries:  Directory_Entries(make([]nbio.Directory_Entry, count)),
		Directory_Buffer: Directory_Buffer(
			make([]byte, buffer_size),
		),
	}
	standard_boundary(func() {
		value := runner
		Walk_Runner_Init(
			&value, nbio.IO{}, root, value.Visitor_State, standard_walk_visit, memory,
		)
	})
	standard_boundary(func() {
		value := runner
		Walk_Directory_Runner_Init(
			&value, nbio.IO{}, root, value.Visitor_State, standard_walk_visit, memory,
		)
	})
	standard_boundary(func() {
		value := runner
		walk_runner_init(
			&value, nbio.IO{}, root, value.Visitor_State, standard_walk_visit, memory,
		)
	})
}

func standard_eval_state_bounds(values []Text) {
	link_counts := []Eval_Link_Count{0, 1, 2, SYMBOLIC_LINK_COUNT_BOUNDARY}
	for index := range values {
		destination := make(bytes.Slice, PATH_SIZE_MAXIMUM)
		remainder := make(bytes.Slice, PATH_SIZE_MAXIMUM)
		link_target := make(bytes.Slice, PATH_SIZE_MAXIMUM)
		copy(remainder, values[index])
		state := Eval_State{
			Destination:       Eval_Destination(destination),
			Remainder:         Eval_Remainder(remainder),
			Link_Target:       Eval_Link_Target(link_target),
			Remainder_Count:   Eval_Remainder_Count(len(values[index])),
			Destination_Count: Eval_Destination_Count(len(values[index])),
			Start:             Eval_Start(len(values[index])),
			End:               Eval_End(len(values[index])),
			Link_Count:        link_counts[index],
			Done:              Eval_Done(index%2 == 1),
		}
		standard_eval_operation_bounds(state)
		standard_eval_buffer_bounds(values[index])
	}
}

func standard_eval_operation_bounds(state Eval_State) {
	status := nbio.File_Status{Exists: true, Mode: nbio.FILE_MODE_DIRECTORY}
	standard_boundary(func() { value := state; eval_state_step(&value) })
	standard_boundary(func() { value := state; eval_component_find(&value) })
	standard_boundary(func() { value := state; eval_status_apply(&value, status) })
	standard_boundary(func() { value := state; eval_link_apply(&value) })
	standard_boundary(func() { value := state; eval_component_dot(&value) })
	standard_boundary(func() { value := state; eval_component_dot_dot(&value) })
	standard_boundary(func() { value := state; eval_component_append(&value) })
	standard_boundary(func() { value := state; eval_component_remove(&value) })
	standard_boundary(func() { value := state; eval_parent_apply(&value) })
	standard_boundary(func() { value := state; eval_has_component(&value) })
	dot := state
	dot.Remainder[0] = '.'
	dot.Remainder_Count = 1
	dot.Start = 0
	dot.End = 1
	standard_boundary(func() { eval_component_dot(&dot) })
	component := state
	component.Remainder[0] = 'a'
	component.Remainder_Count = 1
	component.Start = 0
	component.End = 0
	standard_boundary(func() { eval_has_component(&component) })
}

func standard_eval_buffer_bounds(value Text) {
	destination := make(bytes.Slice, len(value))
	remainder := make(bytes.Slice, len(value))
	link_target := make(bytes.Slice, len(value))
	storage := nbio.Storage{Status_Procedure: standard_existing_status}
	standard_boundary(func() {
		Eval_Symlinks_Into(
			storage, Slice(destination), Slice(remainder), Slice(link_target), value,
		)
	})
	standard_boundary(func() {
		eval_symlinks_apply(
			storage, Slice(destination), Slice(remainder), Slice(link_target), value,
		)
	})
}

func standard_path_helper_bounds(values []Text) {
	counts := []int{0, 1, 2, FILESYSTEM_PATH_COUNT_MAXIMUM}
	buffers := []int{0, 1, 2, nbio.DIRECTORY_BUFFER_SIZE_MAXIMUM}
	for index := range values {
		value := Slice(values[index])
		storage := standard_path_storage_count(counts[index])
		standard_boundary(func() { path_base(value) })
		standard_boundary(func() { path_parent(value) })
		standard_boundary(func() {
			path_storage_write(storage, Path_Count(counts[index]), value)
		})
		standard_boundary(func() {
			path_storage_write_text(storage, Path_Count(counts[index]), values[index])
		})
		standard_boundary(func() {
			path_storage_join(
				storage, Path_Count(counts[index]), value, values[index],
			)
		})
		standard_boundary(func() { path_storage_slots_validate(storage) })
		standard_boundary(func() { path_storage_sort(storage) })
		standard_boundary(func() {
			directory_buffer_validate(Directory_Buffer(make([]byte, buffers[index])))
		})
		output := standard_path_storage_count(1)
		standard_boundary(func() { path_storage_write(output, 0, value) })
		standard_boundary(func() {
			path_storage_join(output, 0, nil, values[index])
		})
	}
	standard_boundary(func() { path_parent(Slice("/a")) })
	standard_boundary(func() { path_parent(Slice("aa/b")) })
	maximum_parent := make(Slice, PATH_PARENT_SIZE_MAXIMUM)
	for index := range maximum_parent {
		maximum_parent[index] = 'a'
	}
	maximum_path := append(maximum_parent, '/', 'b')
	standard_boundary(func() { path_parent(maximum_path) })
	runner := Walk_Runner[standard_walk_state]{
		Visitor_State: &standard_walk_state{}, Visitor: standard_walk_visit,
		Directory_Status: nbio.File_Status{Exists: true, Mode: nbio.FILE_MODE_DIRECTORY},
	}
	standard_boundary(func() { walk_directory_error(&runner, nil) })
}

func standard_existing_status(
	_ unsafe.Pointer, _ string,
) (status nbio.File_Status, err error) {
	return nbio.File_Status{Exists: true}, nil
}

// Benchmark_Is_Local pairs upstream whole-corpus workload.
func Benchmark_Is_Local(b *testing.B) {
	cases := standard_filepath_local_cases()
	b.Run("Shared", func(b *testing.B) {
		b.ReportAllocs()
		var local Boolean
		for b.Loop() {
			for _, one := range cases {
				local = Is_Local(one.Path)
			}
		}
		b.StopTimer()
		runtime.KeepAlive(local)
	})
	b.Run("Standard_Library", func(b *testing.B) {
		b.ReportAllocs()
		var local bool
		for b.Loop() {
			for _, one := range cases {
				local = filepath.IsLocal(string(one.Path))
			}
		}
		b.StopTimer()
		runtime.KeepAlive(local)
	})
}

// Benchmark_Match pairs every upstream match case.
func Benchmark_Match(b *testing.B) {
	for _, one := range standard_filepath_match_cases() {
		name := fmt.Sprintf("%q_%q", one.Pattern, one.Name)
		b.Run(name, func(b *testing.B) {
			b.Run("Shared", func(b *testing.B) {
				b.ReportAllocs()
				var matched Boolean
				var err error
				for b.Loop() {
					matched, err = Match(one.Pattern, one.Name)
				}
				b.StopTimer()
				runtime.KeepAlive(matched)
				runtime.KeepAlive(err)
			})
			b.Run("Standard_Library", func(b *testing.B) {
				b.ReportAllocs()
				var matched bool
				var err error
				for b.Loop() {
					matched, err = filepath.Match(
						string(one.Pattern), string(one.Name),
					)
				}
				b.StopTimer()
				runtime.KeepAlive(matched)
				runtime.KeepAlive(err)
			})
		})
	}
}

// Benchmark_Match_Corpus weighs every upstream case once per operation.
func Benchmark_Match_Corpus(b *testing.B) {
	cases := standard_filepath_match_cases()
	b.Run("Shared", func(b *testing.B) {
		b.ReportAllocs()
		var matched Boolean
		var err error
		for b.Loop() {
			for _, one := range cases {
				matched, err = Match(one.Pattern, one.Name)
			}
		}
		b.StopTimer()
		runtime.KeepAlive(matched)
		runtime.KeepAlive(err)
	})
	b.Run("Standard_Library", func(b *testing.B) {
		b.ReportAllocs()
		var matched bool
		var err error
		for b.Loop() {
			for _, one := range cases {
				matched, err = filepath.Match(string(one.Pattern), string(one.Name))
			}
		}
		b.StopTimer()
		runtime.KeepAlive(matched)
		runtime.KeepAlive(err)
	})
}
