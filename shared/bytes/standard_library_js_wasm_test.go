// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build js && wasm

package bytes_test

import "testing"

// Test_Issue_65571 applies the upstream regression at this package's maximum
// offset because this port has a bounded Slice domain.
func Test_Issue_65571(Test *testing.T) {
	const SIZE = 4_096
	Slice := make([]byte, SIZE)
	Slice[SIZE-1] = 1
	Position := standard_index_byte(Slice, 1)
	if Position != SIZE-1 {
		Test.Errorf("standard_index_byte(Slice, 1) = %d; want %d", Position, SIZE-1)
	}
}
