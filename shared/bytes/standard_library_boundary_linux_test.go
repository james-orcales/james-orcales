// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux

package bytes_test

import (
	"syscall"
	"testing"
)

// Dangerous guard pages expose reads that cross the requested slice. The
// guard pages detect operations that read outside the requested slice.
func dangerous_slice(Test *testing.T) (result_1 []byte) {
	Page_Size := syscall.Getpagesize()
	Mapping, Error := syscall.Mmap(
		0,
		0,
		3*Page_Size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_ANONYMOUS|syscall.MAP_PRIVATE,
	)
	if Error != nil {
		Test.Fatalf("mmap failed: %s", Error)
	}
	Error = syscall.Mprotect(Mapping[:Page_Size], syscall.PROT_NONE)
	if Error != nil {
		Test.Fatalf("low mprotect failed: %s", Error)
	}
	Error = syscall.Mprotect(Mapping[2*Page_Size:], syscall.PROT_NONE)
	if Error != nil {
		Test.Fatalf("high mprotect failed: %s", Error)
	}
	return Mapping[Page_Size : 2*Page_Size]
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Equal_Near_Page_Boundary(Test *testing.T) {
	Slice := dangerous_slice(Test)
	for Index := range Slice {
		Slice[Index] = 'A'
	}
	for Size := 0; Size <= len(Slice); Size++ {
		standard_equal(Slice[:Size], Slice[len(Slice)-Size:])
		standard_equal(Slice[len(Slice)-Size:], Slice[:Size])
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Index_Byte_Near_Page_Boundary(Test *testing.T) {
	Slice := dangerous_slice(Test)
	for Index := range Slice {
		Position := standard_index_byte(Slice[Index:], 1)
		if Position != -1 {
			Test.Fatalf(
				"standard_index_byte(Slice[%d:], 1) = %d; want -1",
				Index,
				Position,
			)
		}
	}
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Index_Near_Page_Boundary(Test *testing.T) {
	Pattern := dangerous_slice(Test)
	if len(Pattern) > 64 {
		Pattern = Pattern[len(Pattern)-64:]
	}
	Slice := dangerous_slice(Test)
	if len(Slice) > 256 {
		Slice = Slice[len(Slice)-256:]
	}
	for Pattern_Index := 1; Pattern_Index < len(Pattern); Pattern_Index++ {
		Pattern[Pattern_Index-1] = 1
		for Index := range Slice {
			Position := standard_index(Slice[Index:], Pattern[:Pattern_Index])
			if Position != -1 {
				Test.Fatalf(
					"standard_index(Slice[%d:], Pattern[:%d]) = %d; want -1",
					Index,
					Pattern_Index,
					Position,
				)
			}
		}
		Pattern[Pattern_Index-1] = 0
	}

	Pattern[len(Pattern)-1] = 1
	for Pattern_Start := range Pattern {
		for Index := range Slice {
			Position := standard_index(Slice[Index:], Pattern[Pattern_Start:])
			if Position != -1 {
				Test.Fatalf(
					"standard_index(Slice[%d:], Pattern[%d:]) = %d; want -1",
					Index,
					Pattern_Start,
					Position,
				)
			}
		}
	}
	Pattern[len(Pattern)-1] = 0
}

// This test keeps upstream standard-library regression coverage in this bounded
// port.
func Test_Count_Near_Page_Boundary(Test *testing.T) {
	Slice := dangerous_slice(Test)
	for Index := range Slice {
		Count := standard_count(Slice[Index:], []byte{1})
		if Count != 0 {
			Test.Fatalf("standard_count(Slice[%d:], {1}) = %d; want 0", Index, Count)
		}
		Count = standard_count(Slice[:Index], []byte{0})
		if Count != Index {
			Test.Fatalf(
				"standard_count(Slice[:%d], {0}) = %d; want %d",
				Index,
				Count,
				Index,
			)
		}
	}
}
