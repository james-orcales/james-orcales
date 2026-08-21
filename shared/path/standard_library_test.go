// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by BSD-style license in Go source tree LICENSE file.

package path

import (
	"fmt"
	"path"
	"runtime"
	"testing"

	"local/james-orcales/shared/testify"
)

type standard_path_case struct {
	Path   Text
	Result string
}

type standard_split_case struct {
	Path      Text
	Directory string
	File      string
}

type standard_join_case struct {
	Elements Elements
	Path     string
}

type standard_extension_case struct {
	Path      Text
	Extension string
}

type standard_absolute_case struct {
	Path     Text
	Absolute bool
}

type standard_match_case struct {
	Pattern Text
	Name    Text
	Match   bool
	Error   error
}

func standard_clean_cases() (cases []standard_path_case) {
	return []standard_path_case{
		{Path: "", Result: "."},
		{Path: "abc", Result: "abc"},
		{Path: "abc/def", Result: "abc/def"},
		{Path: "a/b/c", Result: "a/b/c"},
		{Path: ".", Result: "."},
		{Path: "..", Result: ".."},
		{Path: "../..", Result: "../.."},
		{Path: "../../abc", Result: "../../abc"},
		{Path: "/abc", Result: "/abc"},
		{Path: "/", Result: "/"},
		{Path: "abc/", Result: "abc"},
		{Path: "abc/def/", Result: "abc/def"},
		{Path: "a/b/c/", Result: "a/b/c"},
		{Path: "./", Result: "."},
		{Path: "../", Result: ".."},
		{Path: "../../", Result: "../.."},
		{Path: "/abc/", Result: "/abc"},
		{Path: "abc//def//ghi", Result: "abc/def/ghi"},
		{Path: "//abc", Result: "/abc"},
		{Path: "///abc", Result: "/abc"},
		{Path: "//abc//", Result: "/abc"},
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
		{Path: "abc/./../def", Result: "def"},
		{Path: "abc//./../def", Result: "def"},
		{Path: "abc/../../././../def", Result: "../../def"},
	}
}

func standard_split_cases() (cases []standard_split_case) {
	return []standard_split_case{
		{Path: "a/b", Directory: "a/", File: "b"},
		{Path: "a/b/", Directory: "a/b/", File: ""},
		{Path: "a/", Directory: "a/", File: ""},
		{Path: "a", Directory: "", File: "a"},
		{Path: "/", Directory: "/", File: ""},
	}
}

func standard_join_cases() (cases []standard_join_case) {
	return []standard_join_case{
		{Elements: Elements{}, Path: ""},
		{Elements: Elements{""}, Path: ""},
		{Elements: Elements{"a"}, Path: "a"},
		{Elements: Elements{"a", "b"}, Path: "a/b"},
		{Elements: Elements{"a", ""}, Path: "a"},
		{Elements: Elements{"", "b"}, Path: "b"},
		{Elements: Elements{"/", "a"}, Path: "/a"},
		{Elements: Elements{"/", ""}, Path: "/"},
		{Elements: Elements{"a/", "b"}, Path: "a/b"},
		{Elements: Elements{"a/", ""}, Path: "a"},
		{Elements: Elements{"", ""}, Path: ""},
	}
}

func standard_extension_cases() (cases []standard_extension_case) {
	return []standard_extension_case{
		{Path: "path.go", Extension: ".go"},
		{Path: "path.pb.go", Extension: ".go"},
		{Path: "a.dir/b", Extension: ""},
		{Path: "a.dir/b.go", Extension: ".go"},
		{Path: "a.dir/", Extension: ""},
	}
}

func standard_base_cases() (cases []standard_path_case) {
	return []standard_path_case{
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

func standard_directory_cases() (cases []standard_path_case) {
	return []standard_path_case{
		{Path: "", Result: "."},
		{Path: ".", Result: "."},
		{Path: "/.", Result: "/"},
		{Path: "/", Result: "/"},
		{Path: "////", Result: "/"},
		{Path: "/foo", Result: "/"},
		{Path: "x/", Result: "x"},
		{Path: "abc", Result: "."},
		{Path: "abc/def", Result: "abc"},
		{Path: "abc////def", Result: "abc"},
		{Path: "a/b/.x", Result: "a/b"},
		{Path: "a/b/c.", Result: "a/b"},
		{Path: "a/b/c.x", Result: "a/b"},
	}
}

func standard_absolute_cases() (cases []standard_absolute_case) {
	return []standard_absolute_case{
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

func standard_match_cases() (cases []standard_match_case) {
	return []standard_match_case{
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
	for _, one := range standard_clean_cases() {
		count := Clean_Into(storage[:], one.Path)
		testify.Equal(t, one.Result, string(storage[:count]), "Clean_Into(%q)", one.Path)
		count = Clean_Into(storage[:], Text(one.Result))
		testify.Equal(t, one.Result, string(storage[:count]), "Clean_Into(%q)", one.Result)
	}
}

// Test_Standard_Library_Clean_Allocation ports upstream TestCleanMallocs.
func Test_Standard_Library_Clean_Allocation(t *testing.T) {
	fixture := struct {
		Storage [PATH_SIZE_MAXIMUM]byte
		Path    Text
		Count   Nonempty_Count
	}{}
	for _, one := range standard_clean_cases() {
		fixture.Path = Text(one.Result)
		testify.Zero_Allocation(t, func() {
			fixture.Count = Clean_Into(fixture.Storage[:], fixture.Path)
		})
	}
	if fixture.Count == 0 {
		t.Fatal("Clean_Into returned impossible zero count")
	}
}

// Test_Standard_Library_Split ports upstream TestSplit.
func Test_Standard_Library_Split(t *testing.T) {
	for _, one := range standard_split_cases() {
		directory, file := Split(one.Path)
		testify.Equal(t, one.Directory, string(directory), "Split(%q) directory", one.Path)
		testify.Equal(t, one.File, string(file), "Split(%q) file", one.Path)
	}
}

// Test_Standard_Library_Join ports upstream TestJoin.
func Test_Standard_Library_Join(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	for _, one := range standard_join_cases() {
		count := Join_Into(storage[:], one.Elements)
		testify.Equal(t, one.Path, string(storage[:count]), "Join_Into(%q)", one.Elements)
	}
}

// Test_Standard_Library_Extension ports upstream TestExt.
func Test_Standard_Library_Extension(t *testing.T) {
	for _, one := range standard_extension_cases() {
		testify.Equal(
			t, one.Extension, string(Extension(one.Path)), "Extension(%q)", one.Path,
		)
	}
}

// Test_Standard_Library_Base ports upstream TestBase.
func Test_Standard_Library_Base(t *testing.T) {
	for _, one := range standard_base_cases() {
		testify.Equal(t, one.Result, string(Base(one.Path)), "Base(%q)", one.Path)
	}
}

// Test_Standard_Library_Directory ports upstream TestDir.
func Test_Standard_Library_Directory(t *testing.T) {
	var storage [PATH_SIZE_MAXIMUM]byte
	for _, one := range standard_directory_cases() {
		count := Directory_Into(storage[:], one.Path)
		testify.Equal(
			t, one.Result, string(storage[:count]), "Directory_Into(%q)", one.Path,
		)
	}
}

// Test_Standard_Library_Is_Absolute ports upstream TestIsAbs.
func Test_Standard_Library_Is_Absolute(t *testing.T) {
	for _, one := range standard_absolute_cases() {
		testify.Equal(
			t, one.Absolute, bool(Is_Absolute(one.Path)), "Is_Absolute(%q)", one.Path,
		)
	}
}

// Test_Standard_Library_Match ports upstream TestMatch.
func Test_Standard_Library_Match(t *testing.T) {
	for _, one := range standard_match_cases() {
		matched, err := Match(one.Pattern, one.Name)
		testify.Equal(t, one.Match, bool(matched), "Match(%q, %q)", one.Pattern, one.Name)
		testify.Equal(t, one.Error, err, "Match(%q, %q) error", one.Pattern, one.Name)
	}
}

// Benchmark_Join pairs upstream workload with caller-owned output.
func Benchmark_Join(b *testing.B) {
	b.Run("Shared", benchmark_join_shared)
	b.Run("Standard_Library", benchmark_join_standard_library)
}

func benchmark_join_shared(b *testing.B) {
	b.ReportAllocs()
	parts := Elements{"one", "two", "three", "four"}
	var destination [PATH_SIZE_MAXIMUM]byte
	var count Boundary
	for b.Loop() {
		count = Join_Into(destination[:], parts)
	}
	b.StopTimer()
	runtime.KeepAlive(count)
}

func benchmark_join_standard_library(b *testing.B) {
	b.ReportAllocs()
	parts := []string{"one", "two", "three", "four"}
	text := parts[0]
	for b.Loop() {
		parts[0] = text
		text = path.Join(parts...)
		text = text[:len(parts[0])]
	}
	b.StopTimer()
	runtime.KeepAlive(text)
}

// Benchmark_Match pairs every upstream match case.
func Benchmark_Match(b *testing.B) {
	for _, one := range standard_match_cases() {
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
					matched, err = path.Match(
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
	cases := standard_match_cases()
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
				matched, err = path.Match(
					string(one.Pattern), string(one.Name),
				)
			}
		}
		b.StopTimer()
		runtime.KeepAlive(matched)
		runtime.KeepAlive(err)
	})
}
