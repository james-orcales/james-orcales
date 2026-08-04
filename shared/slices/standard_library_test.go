// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by the BSD-style license in third_party/go/LICENSE.

package slices_test

import (
	"cmp"
	"crypto/sha256"
	"fmt"
	"iter"
	"math"
	"math/rand"
	random_v2 "math/rand/v2"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unsafe"

	invariant "local/james-orcales/shared/invariant/default"
	adapted "local/james-orcales/shared/slices"
	"local/james-orcales/shared/testify"
)

// TestMain registers the invariant roots before the standard library suite runs.
func TestMain(m *testing.M) {
	invariant.Run_Test_Main(m)
}

// The adapters keep the standard library cases unchanged where the public names or callback
// domains differ. The adapters do not implement an algorithm.
func standard_equal[S ~[]E, E comparable](left S, right S) (result_1 bool) {
	return bool(adapted.Equal(left, right))
}

func standard_equal_function[
	Left_Slice ~[]Left,
	Right_Slice ~[]Right,
	Left any,
	Right any,
](left Left_Slice, right Right_Slice, equality func(Left, Right) (result_3 bool)) (result_2 bool) {
	return bool(adapted.Equal_Function(left, right, equality))
}

func standard_compare[S ~[]E, E cmp.Ordered](left S, right S) (result_4 int) {
	return int(adapted.Compare(left, right))
}

func standard_compare_function[
	Left_Slice ~[]Left,
	Right_Slice ~[]Right,
	Left any,
	Right any,
](left Left_Slice, right Right_Slice, comparison func(Left, Right) (result_6 int)) (result_5 int) {
	adapter := func(left_value Left, right_value Right) (result_7 adapted.Comparison) {
		return adapted.Comparison(comparison(left_value, right_value))
	}
	return int(adapted.Compare_Function(left, right, adapter))
}

func standard_index[S ~[]E, E comparable](slice S, value E) (result_8 int) {
	return int(adapted.Index(slice, value))
}

func standard_index_function[S ~[]E, E any](
	slice S,
	predicate func(E) (result_10 bool),
) (result_9 int) {
	return int(adapted.Index_Function(slice, predicate))
}

func standard_contains[S ~[]E, E comparable](slice S, value E) (result_11 bool) {
	return bool(adapted.Contains(slice, value))
}

func standard_contains_function[S ~[]E, E any](
	slice S,
	predicate func(E) (result_13 bool),
) (result_12 bool) {
	return bool(adapted.Contains_Function(slice, predicate))
}

func standard_insert[S ~[]E, E any](slice S, position int, values ...E) (result_14 S) {
	return adapted.Insert(slice, adapted.Position(position), values)
}

func standard_delete[S ~[]E, E any](slice S, start int, end int) (result_15 S) {
	return adapted.Delete(slice, adapted.Position(start), adapted.Position(end))
}

func standard_delete_function[S ~[]E, E any](
	slice S,
	predicate func(E) (result_17 bool),
) (result_16 S) {
	return adapted.Delete_Function(slice, predicate)
}

func standard_replace[S ~[]E, E any](slice S, start int, end int, values ...E) (result_18 S) {
	return adapted.Replace(slice, adapted.Position(start), adapted.Position(end), values)
}

func standard_clone[S ~[]E, E any](slice S) (result_19 S) {
	return adapted.Clone(slice)
}

func standard_compact[S ~[]E, E comparable](slice S) (result_20 S) {
	return adapted.Compact(slice)
}

func standard_compact_function[S ~[]E, E any](
	slice S,
	equality func(E, E) (result_22 bool),
) (result_21 S) {
	return adapted.Compact_Function(slice, equality)
}

func standard_grow[S ~[]E, E any](slice S, count int) (result_23 S) {
	return adapted.Grow(slice, adapted.Count(count))
}

func standard_clip[S ~[]E, E any](slice S) (result_24 S) {
	return adapted.Clip(slice)
}

func standard_reverse[S ~[]E, E any](slice S) {
	adapted.Reverse(slice)
}

func standard_concatenate[S ~[]E, E any](slices ...S) (result_25 S) {
	return adapted.Concatenate(slices)
}

func standard_repeat[S ~[]E, E any](slice S, count int) (result_26 S) {
	return adapted.Repeat(slice, adapted.Count(count))
}

func standard_sort[S ~[]E, E cmp.Ordered](slice S) {
	adapted.Sort(slice)
}

func standard_sort_function[S ~[]E, E any](slice S, comparison func(E, E) (result_27 int)) {
	adapter := func(left E, right E) (result_28 adapted.Comparison) {
		return adapted.Comparison(comparison(left, right))
	}
	adapted.Sort_Function(slice, adapter)
}

func standard_sort_stable_function[S ~[]E, E any](slice S, comparison func(E, E) (result_29 int)) {
	adapter := func(left E, right E) (result_30 adapted.Comparison) {
		return adapted.Comparison(comparison(left, right))
	}
	adapted.Sort_Stable_Function(slice, adapter)
}

func standard_is_sorted[S ~[]E, E cmp.Ordered](slice S) (result_31 bool) {
	return bool(adapted.Is_Sorted(slice))
}

func standard_is_sorted_function[S ~[]E, E any](
	slice S,
	comparison func(E, E) (result_33 int),
) (result_32 bool) {
	adapter := func(left E, right E) (result_34 adapted.Comparison) {
		return adapted.Comparison(comparison(left, right))
	}
	return bool(adapted.Is_Sorted_Function(slice, adapter))
}

func standard_minimum[S ~[]E, E cmp.Ordered](slice S) (result_35 E) {
	return adapted.Minimum(slice)
}

func standard_minimum_function[S ~[]E, E any](
	slice S,
	comparison func(E, E) (result_37 int),
) (result_36 E) {
	adapter := func(left E, right E) (result_38 adapted.Comparison) {
		return adapted.Comparison(comparison(left, right))
	}
	return adapted.Minimum_Function(slice, adapter)
}

func standard_maximum[S ~[]E, E cmp.Ordered](slice S) (result_39 E) {
	return adapted.Maximum(slice)
}

func standard_maximum_function[S ~[]E, E any](
	slice S,
	comparison func(E, E) (result_41 int),
) (result_40 E) {
	adapter := func(left E, right E) (result_42 adapted.Comparison) {
		return adapted.Comparison(comparison(left, right))
	}
	return adapted.Maximum_Function(slice, adapter)
}

func standard_binary_search[S ~[]E, E cmp.Ordered](
	slice S,
	target E,
) (result_43 int, result_44 bool) {
	position, found := adapted.Binary_Search(slice, target)
	return int(position), bool(found)
}

func standard_binary_search_function[S ~[]E, E any, Target any](
	slice S,
	target Target,
	comparison func(E, Target) (result_47 int),
) (result_45 int, result_46 bool) {
	adapter := func(element E, wanted Target) (result_48 adapted.Comparison) {
		return adapted.Comparison(comparison(element, wanted))
	}
	position, found := adapted.Binary_Search_Function(slice, target, adapter)
	return int(position), bool(found)
}

func standard_all[S ~[]E, E any](slice S) (result_49 iter.Seq2[int, E]) {
	return func(yield func(int, E) (result_50 bool)) {
		for position, value := range adapted.All(slice) {
			if !yield(int(position), value) {
				return
			}
		}
	}
}

// Example runs the 41 output cases from the Go standard library.
func Example() {
	standard_run_all_examples()

	// Output:
	// 3be44b1f62e3afc3ef035e465d7e39d7aff5839a6b2e4bf7839e857b825424b2.
	// 81cd581f873a9ded8f0f9a6ca46b80fb35f7117a14dc5bf3a77f6648a6f28c0d.
	// c2035a01ff9109d643c0e3c7fccfffac7ce5e51b26606bf2d5a24ac9a9125851.
	// 905ca48f011fd880bc786ddad56f838c69145922ffeffb9a4a02f5c43895051e.
	// 3a90a6cfdcb5f71ae043c5871f675a312020fe98727a92cf6470a163ff5d5652.
	// 4355a46b19d348dc2f57c046f8ef63d4538ebb936000f3c9ee954a27460dd865.
	// 0185484518446284cd3a3c3f211b22e11e7a9b6d25e1ce3a3fb9b204ede0a1cb.
	// 2165591c6f1f2a40eb56662a19dfbde33483874ec685ad585e23f8043a928267.
	// c70a1c987374c7f5b5d5dd759a0b3094e9a872ffb319519b1cf85e490a11613e.
	// acb2b288b9f028830645d94e3a4417e5ffc574a024576d6f69b53d989e9d93ea.
	// a17fcf0a2f50e2d495e4f90ce263410edc183add6c62699a2facbccf60410f74.
	// 3e3ca10204cb093a93d1a750c01232e559a1c8b9208c629324c087b17e67e8b3.
	// 3c26358e2d703030d08d4d520cc231cc46ddce1ccb21b6ab75e44aad994a0ec1.
	// 84d80c98569637337a93f0f57a7b4ff1ab1c8e3f3c4fdc5a0aabc5b4bd8c418e.
	// acb2b288b9f028830645d94e3a4417e5ffc574a024576d6f69b53d989e9d93ea.
	// acb2b288b9f028830645d94e3a4417e5ffc574a024576d6f69b53d989e9d93ea.
	// 084c799cd551dd1d8d5c5f9a5d593b2e931f5e36122ee5c793c1d08a19839cc0.
	// dc3c6ea0958331c423499af6c2eb14f3c300faf367337c017e3b1b760ff41f2a.
	// 553764c84ee5ed5460452af78623f892b5391217bbf211b125a145bfeef9b2c7.
	// e5ceb56bbcbcc843254b3500e054eb8e8fea0553400bd6ed8e6fe745d9f40846.
	// 90d029e6eb1b56297c3777b492ad3b8a38400690bb0ffadd63a78357ebb6e9e7.
	// 4f0f2f0be3b607c8b5ea290172c62fd135801051223e6d7453b4d930951a4801.
	// 678532dff3fdd3f239ee351e5eab55c7da63b3334f75895dab62c28976ae7e69.
	// b7556b0b55145b2e537c3ceed51514945673d9890ae365b1d3b111256c21653f.
	// 8a79e3c52aa0f44777420cd17b92c3fb9bb867c39dca945b2b13e6545b1174d6.
	// 8a79e3c52aa0f44777420cd17b92c3fb9bb867c39dca945b2b13e6545b1174d6.
	// 7cf84aa64a33183d954435574d96d4982b292f11541b7d91139dc69bbdb72242.
	// 7c07458f150284f2070c90c18478b367e39491b92b1678c6543d1eae82e6d669.
	// d2dfee6cabcb38f4feecb01842f3cdcd31ebc25ab25a77dffadc186367dec667.
	// 78c009333895df2792d8649ce6c73ee9d5cf754d3523e64d597c57c176baf663.
	// acb2b288b9f028830645d94e3a4417e5ffc574a024576d6f69b53d989e9d93ea.
	// 97f06ae6febb9a1e0503eed53ffb9c898aae02b52dd69e9aadaa7d2671ce4a74.
	// c3894e3a186aa83d10d0c6c7e69df93a49d430b8d9fead65a9c81aedaffc6e0a.
	// 8289e884f34d2d03c4ac925f35ec7b2d7373c1ffb033416cbfd7d930b465d4e6.
	// 76b3691da3b967778a58f47f370b45a78b3f43ed7dd227de0078b0cdd9b806c7.
	// 10685ffb32d7fdfabcd88b351516d6d66b7156af258f020411fa532b31fe1184.
	// 7ad3ee187f447520ff35f03c4b87a928207b0d4c36298fb47105bbe8a7ca8077.
	// 3dc6989616d0eccf893547f0511208f235f8bbb56a19955fe9a6b13afe8a219b.
	// 81aeb2097fb0d3a7dcf5e41267676ebe4e9ddafbfc6d03f9f87b1f6da6c723a1.
	// 6235a335a9979bacbd44e0d777160f9996dc9bc44d79a07aa4b62b19c89d890c.
	// 9bf5bd2f874105678871af0c2ff29c3f6d86336a2e7f75836c4788b5edf74a0d.
}

func standard_backward[S ~[]E, E any](slice S) (result_51 iter.Seq2[int, E]) {
	return func(yield func(int, E) (result_52 bool)) {
		for position, value := range adapted.Backward(slice) {
			if !yield(int(position), value) {
				return
			}
		}
	}
}

func standard_values[S ~[]E, E any](slice S) (result_53 iter.Seq[E]) {
	return adapted.Values(slice)
}

func standard_append_sequence[S ~[]E, E any](slice S, sequence iter.Seq[E]) (result_54 S) {
	return adapted.Append_Sequence(slice, sequence)
}

func standard_collect[E any](sequence iter.Seq[E]) (result_55 []E) {
	return adapted.Collect[[]E](sequence)
}

func standard_sorted[E cmp.Ordered](sequence iter.Seq[E]) (result_56 []E) {
	return adapted.Sorted[[]E](sequence)
}

func standard_sorted_function[E any](
	sequence iter.Seq[E],
	comparison func(E, E) (result_58 int),
) (result_57 []E) {
	adapter := func(left E, right E) (result_59 adapted.Comparison) {
		return adapted.Comparison(comparison(left, right))
	}
	return adapted.Sorted_Function[[]E](sequence, adapter)
}

func standard_sorted_stable_function[E any](
	sequence iter.Seq[E],
	comparison func(E, E) (result_61 int),
) (result_60 []E) {
	adapter := func(left E, right E) (result_62 adapted.Comparison) {
		return adapted.Comparison(comparison(left, right))
	}
	return adapted.Sorted_Stable_Function[[]E](sequence, adapter)
}

func standard_chunk[S ~[]E, E any](slice S, count int) (result_63 iter.Seq[S]) {
	return adapted.Chunk(slice, adapted.Count(count))
}

// Standard_accept preserves an evaluated value that the standard library test discards.
func standard_accept[Value any](value Value) {
	if any(value) == nil {
		return
	}
}

func standard_println(output *strings.Builder, values ...any) {
	fmt.Fprintln(output, values...)
}

func standard_printf(output *strings.Builder, format string, values ...any) {
	fmt.Fprintf(output, format, values...)
}

func standard_run_example(example func(*strings.Builder)) {
	var output strings.Builder
	example(&output)
	digest := sha256.Sum256([]byte(output.String()))
	fmt.Printf("%x.\n", digest)
}

func standard_run_all_examples() {
	standard_run_example(standard_library_example_binary_search)
	standard_run_example(standard_library_example_binary_search_function)
	standard_run_example(standard_library_example_compact)
	standard_run_example(standard_library_example_compact_function)
	standard_run_example(standard_library_example_compare)
	standard_run_example(standard_library_example_compare_function)
	standard_run_example(standard_library_example_contains_function)
	standard_run_example(standard_library_example_delete)
	standard_run_example(standard_library_example_delete_function)
	standard_run_example(standard_library_example_equal)
	standard_run_example(standard_library_example_equal_function)
	standard_run_example(standard_library_example_index)
	standard_run_example(standard_library_example_index_function)
	standard_run_example(standard_library_example_insert)
	standard_run_example(standard_library_example_is_sorted)
	standard_run_example(standard_library_example_is_sorted_function)
	standard_run_example(standard_library_example_maximum)
	standard_run_example(standard_library_example_maximum_function)
	standard_run_example(standard_library_example_minimum)
	standard_run_example(standard_library_example_minimum_function)
	standard_run_example(standard_library_example_replace)
	standard_run_example(standard_library_example_reverse)
	standard_run_example(standard_library_example_sort)
	standard_run_example(standard_library_example_sort_function_case_insensitive)
	standard_run_example(standard_library_example_sort_function_multi_field)
	standard_run_example(standard_library_example_sort_stable_function)
	standard_run_example(standard_library_example_clone)
	standard_run_example(standard_library_example_grow)
	standard_run_example(standard_library_example_clip)
	standard_run_example(standard_library_example_concatenate)
	standard_run_example(standard_library_example_contains)
	standard_run_example(standard_library_example_repeat)
	standard_run_example(standard_library_example_all)
	standard_run_example(standard_library_example_backward)
	standard_run_example(standard_library_example_values)
	standard_run_example(standard_library_example_append_sequence)
	standard_run_example(standard_library_example_collect)
	standard_run_example(standard_library_example_sorted)
	standard_run_example(standard_library_example_sorted_function)
	standard_run_example(standard_library_example_sorted_stable_function)
	standard_run_example(standard_library_example_chunk)
}

func standard_elements[Element any](values ...Element) (result []Element) {
	return values
}

func standard_make[Element any](zero Element, count int) (result []Element) {
	return make([]Element, count)
}

func equal_int_tests() (result []struct {
	S1, S2 []int
	Want   bool
}) {
	return []struct {
		S1, S2 []int
		Want   bool
	}{
		{
			[]int{1},
			nil,
			false,
		},
		{
			[]int{},
			nil,
			true,
		},
		{
			[]int{1, 2, 3},
			[]int{1, 2, 3},
			true,
		},
		{
			[]int{1, 2, 3},
			[]int{1, 2, 3, 4},
			false,
		},
	}
}

// Test_Standard_Library_Equal ports the Go standard library test.
func Test_Standard_Library_Equal(t *testing.T) {
	for _, test := range equal_int_tests() {
		{

			got := standard_equal(test.S1, test.S2)
			testify.False(t, got != test.Want,

				"standard_equal(%v, %v) = %t, want %t",
				test.S1,
				test.S2,
				got,
				test.Want)
		}
	}
	standard_equal_case(t, standard_elements(1.0, 2.0), standard_elements(1.0, 2.0), true)
	standard_equal_case(
		t,
		standard_elements(1.0, 2.0, math.NaN()),
		standard_elements(1.0, 2.0, math.NaN()),
		false,
	)
}

func standard_equal_case[Element comparable](
	t *testing.T,
	left []Element,
	right []Element,
	want bool,
) {
	t.Helper()
	{

		got := standard_equal(left, right)
		testify.False(t, got != want,
			"standard_equal(%v, %v) = %t, want %t", left, right, got, want)
	}
}

// Equal is simply ==.
func equal[T comparable](v1, v2 T) (result_64 bool) {
	return v1 == v2
}

// EqualNaN is like == except that all NaNs are equal.
func equal_na_n[T comparable](v1, v2 T) (result_65 bool) {
	is_na_n := func(f T) (result_66 bool) { return f != f }
	return v1 == v2 || (is_na_n(v1) && is_na_n(v2))
}

// OffByOne returns true if integers v1 and v2 differ by 1.
func off_by_one(v1, v2 int) (result_67 bool) {
	return v1 == v2+1 || v1 == v2-1
}

// Test_Standard_Library_Equal_Func ports the Go standard library test.
func Test_Standard_Library_Equal_Function(t *testing.T) {
	for _, test := range equal_int_tests() {
		{

			got := standard_equal_function(test.S1, test.S2, equal[int])
			testify.False(t, got != test.Want,

				"standard_equal_function(%v, %v, equal[int]) = %t, want %t",
				test.S1,
				test.S2,
				got,
				test.Want)
		}
	}
	comparison_values := standard_elements(1.0, 2.0, math.NaN())
	testify.False(t,
		standard_equal_function(comparison_values, comparison_values, equal),
		"standard_equal_function must use IEEE equality")
	testify.True(t,
		standard_equal_function(comparison_values, comparison_values, equal_na_n),
		"standard_equal_function must admit an injected NaN equality")

	s1 := []int{1, 2, 3}
	s2 := []int{2, 3, 4}
	testify.False(t, standard_equal_function(s1, s1, off_by_one),
		"standard_equal_function(%v, %v, offByOne) = true, want false", s1, s1)
	testify.True(t, standard_equal_function(s1, s2, off_by_one),
		"standard_equal_function(%v, %v, offByOne) = false, want true", s1, s2)

	s3 := []string{"a", "b", "c"}
	s4 := []string{"A", "B", "C"}
	testify.True(t, standard_equal_function(s3, s4, strings.EqualFold),

		"standard_equal_function(%v, %v, strings.EqualFold) = false, want true",
		s3,
		s4)

	cmp_int_string := func(v1 int, v2 string) (result_68 bool) {
		return string(rune(v1)-1+'a') == v2
	}
	testify.True(t, standard_equal_function(s1, s3, cmp_int_string),
		"standard_equal_function(%v, %v, cmpIntString) = false, want true", s1, s3)
}

// Benchmark_Standard_Library_Equal_Func_Large ports the Go standard library benchmark.
func Benchmark_Standard_Library_Equal_Function_Large(b *testing.B) {
	type Large [4 * 1024]byte

	xs := make([]Large, 1024)
	ys := make([]Large, 1024)
	for i_index := 0; i_index < b.N; i_index++ {
		standard_equal_function(xs, ys, func(x, y Large) (result_69 bool) { return x == y })
	}
}

func compare_int_tests() (result []struct {
	S1, S2 []int
	Want   int
}) {
	return []struct {
		S1, S2 []int
		Want   int
	}{
		{
			[]int{1},
			[]int{1},
			0,
		},
		{
			[]int{1},
			[]int{},
			1,
		},
		{
			[]int{},
			[]int{1},
			-1,
		},
		{
			[]int{},
			[]int{},
			0,
		},
		{
			[]int{1, 2, 3},
			[]int{1, 2, 3},
			0,
		},
		{
			[]int{1, 2, 3},
			[]int{1, 2, 3, 4},
			-1,
		},
		{
			[]int{1, 2, 3, 4},
			[]int{1, 2, 3},
			+1,
		},
		{
			[]int{1, 2, 3},
			[]int{1, 4, 3},
			-1,
		},
		{
			[]int{1, 4, 3},
			[]int{1, 2, 3},
			+1,
		},
		{
			[]int{1, 4, 3},
			[]int{1, 2, 3, 8, 9},
			+1,
		},
	}
}

// Test_Standard_Library_Compare ports the Go standard library test.
func Test_Standard_Library_Compare(t *testing.T) {
	int_want := func(want bool) (result_70 string) {
		if want {
			return "0"
		}
		return "!= 0"
	}
	for _, test := range equal_int_tests() {
		{

			got := standard_compare(test.S1, test.S2)
			testify.False(t, (got == 0) != test.Want,

				"standard_compare(%v, %v) = %d, want %s",
				test.S1,
				test.S2,
				got,
				int_want(test.Want))
		}
	}
	for _, test := range compare_int_tests() {
		{

			got := standard_compare(test.S1, test.S2)
			testify.False(t, got != test.Want,

				"standard_compare(%v, %v) = %d, want %d",
				test.S1,
				test.S2,
				got,
				test.Want)
		}
	}
	standard_compare_float_cases(t)
}

func standard_compare_float_cases(t *testing.T) {
	standard_compare_case(
		t,
		standard_elements(math.NaN())[:0],
		standard_elements(math.NaN())[:0],
		0,
	)
	standard_compare_case(t, standard_elements(1.0), standard_elements(1.0), 0)
	standard_compare_case(t, standard_elements(math.NaN()), standard_elements(math.NaN()), 0)
	standard_compare_case(
		t,
		standard_elements(1.0, 2.0, math.NaN()),
		standard_elements(1.0, 2.0, math.NaN()),
		0,
	)
	standard_compare_case(
		t,
		standard_elements(1.0, math.NaN(), 3.0),
		standard_elements(1.0, math.NaN(), 4.0),
		-1,
	)
	standard_compare_case(
		t,
		standard_elements(1.0, math.NaN(), 3.0),
		standard_elements(1.0, 2.0, 4.0),
		-1,
	)
	standard_compare_case(
		t,
		standard_elements(1.0, math.NaN(), 3.0),
		standard_elements(1.0, 2.0, math.NaN()),
		-1,
	)
	standard_compare_case(
		t,
		standard_elements(1.0, 2.0, 3.0),
		standard_elements(1.0, 2.0, math.NaN()),
		1,
	)
	standard_compare_case(
		t,
		standard_elements(1.0, 2.0, 3.0),
		standard_elements(1.0, math.NaN(), 3.0),
		1,
	)
	standard_compare_case(
		t,
		standard_elements(1.0, math.NaN(), 3.0, 4.0),
		standard_elements(1.0, 2.0, math.NaN()),
		-1,
	)
}

func standard_compare_case[Element cmp.Ordered](
	t *testing.T,
	left []Element,
	right []Element,
	want int,
) {
	t.Helper()
	{

		got := standard_compare(left, right)
		testify.False(t, got != want,
			"standard_compare(%v, %v) = %d, want %d", left, right, got, want)
	}
}

func equal_to_cmp[T comparable](
	eq func(T, T) (result_72 bool),
) (result_71 func(T, T) (result_73 int)) {
	return func(v1, v2 T) (result_74 int) {
		if eq(v1, v2) {
			return 0
		}
		return 1
	}
}

// Test_Standard_Library_Compare_Func ports the Go standard library test.
func Test_Standard_Library_Compare_Function(t *testing.T) {
	int_want := func(want bool) (result_75 string) {
		if want {
			return "0"
		}
		return "!= 0"
	}
	for _, test := range equal_int_tests() {
		{

			got := standard_compare_function(
				test.S1,
				test.S2,
				equal_to_cmp(equal[int]),
			)
			testify.False(t, (got == 0) != test.Want,

				"standard_compare_function(%v, %v, "+
					"equalToCmp(equal[int])) = %d, want %s",
				test.S1,
				test.S2,
				got,
				int_want(test.Want))
		}
	}
	for _, test := range compare_int_tests() {
		{

			got := standard_compare_function(
				test.S1,
				test.S2,
				cmp.Compare[int],
			)
			testify.False(t, got != test.Want,

				"standard_compare_function(%v, %v, cmp[int]) = %d, want %d",
				test.S1,
				test.S2,
				got,
				test.Want)
		}
	}
	standard_compare_function_float_cases(t)
	standard_compare_function_cross_type_cases(t)
}

func standard_compare_function_cross_type_cases(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := []int{2, 3, 4}
	{

		got := standard_compare_function(s1, s2, equal_to_cmp(off_by_one))
		testify.False(t, got != 0,
			"standard_compare_function(%v, %v, offByOne) = %d, want 0", s1, s2, got)
	}

	s3 := []string{"a", "b", "c"}
	s4 := []string{"A", "B", "C"}
	{

		got := standard_compare_function(s3, s4, strings.Compare)
		testify.False(t, got != 1,

			"standard_compare_function(%v, %v, strings.Compare) = %d, want 1",
			s3,
			s4,
			got)
	}

	compare_lower := func(v1, v2 string) (result_76 int) {
		return strings.Compare(strings.ToLower(v1), strings.ToLower(v2))
	}
	{

		got := standard_compare_function(s3, s4, compare_lower)
		testify.False(t, got != 0,

			"standard_compare_function(%v, %v, compareLower) = %d, want 0",
			s3,
			s4,
			got)
	}

	cmp_int_string := func(v1 int, v2 string) (result_77 int) {
		return strings.Compare(string(rune(v1)-1+'a'), v2)
	}
	{

		got := standard_compare_function(s1, s3, cmp_int_string)
		testify.False(t, got != 0,

			"standard_compare_function(%v, %v, cmpIntString) = %d, want 0",
			s1,
			s3,
			got)
	}
}

func standard_compare_function_float_cases(t *testing.T) {
	standard_compare_function_case(
		t,
		standard_elements(math.NaN())[:0],
		standard_elements(math.NaN())[:0],
		0,
	)
	standard_compare_function_case(t, standard_elements(1.0), standard_elements(1.0), 0)
	standard_compare_function_case(
		t,
		standard_elements(math.NaN()),
		standard_elements(math.NaN()),
		0,
	)
	standard_compare_function_case(
		t,
		standard_elements(1.0, 2.0, math.NaN()),
		standard_elements(1.0, 2.0, math.NaN()),
		0,
	)
	standard_compare_function_case(
		t,
		standard_elements(1.0, math.NaN(), 3.0),
		standard_elements(1.0, math.NaN(), 4.0),
		-1,
	)
	standard_compare_function_case(
		t,
		standard_elements(1.0, math.NaN(), 3.0),
		standard_elements(1.0, 2.0, 4.0),
		-1,
	)
	standard_compare_function_case(
		t,
		standard_elements(1.0, math.NaN(), 3.0),
		standard_elements(1.0, 2.0, math.NaN()),
		-1,
	)
	standard_compare_function_case(
		t,
		standard_elements(1.0, 2.0, 3.0),
		standard_elements(1.0, 2.0, math.NaN()),
		1,
	)
	standard_compare_function_case(
		t,
		standard_elements(1.0, 2.0, 3.0),
		standard_elements(1.0, math.NaN(), 3.0),
		1,
	)
	standard_compare_function_case(
		t,
		standard_elements(1.0, math.NaN(), 3.0, 4.0),
		standard_elements(1.0, 2.0, math.NaN()),
		-1,
	)
}

func standard_compare_function_case[Element cmp.Ordered](
	t *testing.T,
	left []Element,
	right []Element,
	want int,
) {
	t.Helper()
	{

		got := standard_compare_function(left, right, cmp.Compare)
		testify.False(t, got != want,
			"standard_compare_function(%v, %v) = %d, want %d", left, right, got, want)
	}
}

func index_tests() (result []struct {
	S    []int
	V    int
	Want int
}) {
	return []struct {
		S    []int
		V    int
		Want int
	}{
		{
			nil,
			0,
			-1,
		},
		{
			[]int{},
			0,
			-1,
		},
		{
			[]int{1, 2, 3},
			2,
			1,
		},
		{
			[]int{1, 2, 2, 3},
			2,
			1,
		},
		{
			[]int{1, 2, 3, 2},
			2,
			1,
		},
	}
}

// Test_Standard_Library_Index ports the Go standard library test.
func Test_Standard_Library_Index(t *testing.T) {
	for _, test := range index_tests() {
		{

			got := standard_index(test.S, test.V)
			testify.False(t, got != test.Want,

				"standard_index(%v, %v) = %d, want %d",
				test.S,
				test.V,
				got,
				test.Want)
		}
	}
}

func equal_to_index[T any](
	f func(T, T) (result_79 bool),
	v1 T,
) (result_78 func(T) (result_80 bool)) {
	return func(v2 T) (result_81 bool) {
		return f(v1, v2)
	}
}

// Benchmark_Standard_Library_Index_Large ports the Go standard library benchmark.
func Benchmark_Standard_Library_Index_Large(b *testing.B) {
	type Large [4 * 1024]byte

	ss := make([]Large, 1024)
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index(ss, Large{1})
	}
}

// Test_Standard_Library_Index_Func ports the Go standard library test.
func Test_Standard_Library_Index_Function(t *testing.T) {
	for _, test := range index_tests() {
		{

			got := standard_index_function(
				test.S,
				equal_to_index(equal[int], test.V),
			)
			testify.False(t, got != test.Want,

				"standard_index_function(%v, "+
					"equalToIndex(equal[int], %v)) = %d, want %d",
				test.S,
				test.V,
				got,
				test.Want)
		}
	}

	s1 := []string{"hi", "HI"}
	{

		got := standard_index_function(s1, equal_to_index(equal[string], "HI"))
		testify.False(t, got != 1,

			"standard_index_function(%v, "+
				"equalToIndex(equal[string], %q)) = %d, want "+
				"%d",
			s1,
			"HI",
			got,
			1)
	}
	{

		got := standard_index_function(s1, equal_to_index(strings.EqualFold, "HI"))
		testify.False(t, got != 0,

			"standard_index_function(%v, "+
				"equalToIndex(strings.EqualFold, %q)) = %d, "+
				"want %d",
			s1,
			"HI",
			got,
			0)
	}
}

// Benchmark_Standard_Library_Index_Func_Large ports the Go standard library benchmark.
func Benchmark_Standard_Library_Index_Function_Large(b *testing.B) {
	type Large [4 * 1024]byte

	ss := make([]Large, 1024)
	for i_index := 0; i_index < b.N; i_index++ {
		standard_index_function(ss, func(e Large) (result_82 bool) {
			return e == Large{1}
		})
	}
}

// Test_Standard_Library_Contains ports the Go standard library test.
func Test_Standard_Library_Contains(t *testing.T) {
	for _, test := range index_tests() {
		{

			got := standard_contains(test.S, test.V)
			testify.False(t, got != (test.Want != -1),

				"standard_contains(%v, %v) = %t, want %t",
				test.S,
				test.V,
				got,
				test.Want != -1)
		}
	}
}

// Test_Standard_Library_Contains_Func ports the Go standard library test.
func Test_Standard_Library_Contains_Function(t *testing.T) {
	for _, test := range index_tests() {
		{

			got := standard_contains_function(
				test.S,
				equal_to_index(equal[int], test.V),
			)
			testify.False(t, got != (test.Want != -1),

				"standard_contains_function(%v, "+
					"equalToIndex(equal[int], %v)) = %t, want %t",
				test.S,
				test.V,
				got,
				test.Want != -1)
		}
	}

	s1 := []string{"hi", "HI"}
	{

		got := standard_contains_function(s1, equal_to_index(equal[string], "HI"))
		testify.False(t, got != true,

			"standard_contains_function(%v, "+
				"equalToContains(equal[string], %q)) = %t, "+
				"want %t",
			s1,
			"HI",
			got,
			true)
	}
	{

		got := standard_contains_function(
			s1,
			equal_to_index(equal[string], "hI"),
		)
		testify.False(t, got != false,

			"standard_contains_function(%v, "+
				"equalToContains(strings.EqualFold, %q)) = "+
				"%t, want %t",
			s1,
			"hI",
			got,
			false)
	}
	{

		got := standard_contains_function(
			s1,
			equal_to_index(strings.EqualFold, "hI"),
		)
		testify.False(t, got != true,

			"standard_contains_function(%v, "+
				"equalToContains(strings.EqualFold, %q)) = "+
				"%t, want %t",
			s1,
			"hI",
			got,
			true)
	}
}

func insert_tests() (result []struct {
	S    []int
	I    int
	Add  []int
	Want []int
}) {
	return []struct {
		S    []int
		I    int
		Add  []int
		Want []int
	}{
		{
			[]int{1, 2, 3},
			0,
			[]int{4},
			[]int{4, 1, 2, 3},
		},
		{
			[]int{1, 2, 3},
			1,
			[]int{4},
			[]int{1, 4, 2, 3},
		},
		{
			[]int{1, 2, 3},
			3,
			[]int{4},
			[]int{1, 2, 3, 4},
		},
		{
			[]int{1, 2, 3},
			2,
			[]int{4, 5},
			[]int{1, 2, 4, 5, 3},
		},
	}
}

// Test_Standard_Library_Insert ports the Go standard library test.
func Test_Standard_Library_Insert(t *testing.T) {
	s := []int{1, 2, 3}
	{

		got := standard_insert(s, 0)
		testify.True(t, standard_equal(got, s),
			"standard_insert(%v, 0) = %v, want %v", s, got, s)
	}
	for _, test := range insert_tests() {
		copy := standard_clone(test.S)
		{

			got := standard_insert(
				copy,
				test.I,
				test.Add...,
			)
			testify.True(t, standard_equal(got, test.Want),

				"standard_insert(%v, %d, %v...) = %v, want %v",
				test.S,
				test.I,
				test.Add,
				got,
				test.Want)
		}
	}

	if true {
		// Allocations should be amortized.
		const COUNT = 50
		n := testing.AllocsPerRun(10, func() {
			s := []int{1, 2, 3}
			for i_index := 0; i_index < COUNT; i_index++ {
				s = standard_insert(s, 0, 1)
			}
		})
		testify.False(t, n > COUNT/2,

			"too many allocations inserting %d elements: "+
				"got %v, want less than %d",
			COUNT,
			n,
			COUNT/2)
	}
}

// Test_Standard_Library_Insert_Overlap ports the Go standard library test.
func Test_Standard_Library_Insert_Overlap(t *testing.T) {
	const N_COUNT = 10
	a := make([]int, N_COUNT)
	want := make([]int, 2*N_COUNT)
	for n := 0; n <= N_COUNT; n++ { // length
		for i := 0; i <= n; i++ { // insertion point
			for x := 0; x <= N_COUNT; x++ { // start of inserted data
				for y := x; y <= N_COUNT; y++ { // end of inserted data
					for k_index := 0; k_index < N_COUNT; k_index++ {
						a[k_index] = k_index
					}
					want = want[:0]
					want = append(want, a[:i]...)
					want = append(want, a[x:y]...)
					want = append(want, a[i:n]...)
					got := standard_insert(a[:n], i, a[x:y]...)
					testify.True(t, standard_equal(got, want),

						"standard_insert with overlap failed n=%d "+
							"i=%d x=%d y=%d, got %v want %v",
						n,
						i,
						x,
						y,
						got,
						want)
				}
			}
		}
	}
}

// Test_Standard_Library_Insert_Panics ports the Go standard library test.
func Test_Standard_Library_Insert_Panics(t *testing.T) {
	const A_COUNT = 3
	const B_COUNT = 1
	a := [A_COUNT]int{}
	b := [B_COUNT]int{}
	for _, test := range []struct {
		Name string
		S    []int
		I    int
		V    []int
	}{
		// There are no values.
		{"with negative index", a[:1:1], -1, nil},
		{"with out-of-bounds index and > cap", a[:1:1], 2, nil},
		{"with out-of-bounds index and = cap", a[:1:2], 2, nil},
		{"with out-of-bounds index and < cap", a[:1:3], 2, nil},

		// There are values.
		{"with negative index", a[:1:1], -1, b[:]},
		{"with out-of-bounds index and > cap", a[:1:1], 2, b[:]},
		{"with out-of-bounds index and = cap", a[:1:2], 2, b[:]},
		{"with out-of-bounds index and < cap", a[:1:3], 2, b[:]},
	} {
		testify.Panics(t, func() { standard_insert(test.S, test.I, test.V...) },
			"standard_insert %s: got no panic, want panic", test.Name)
	}
}

func delete_tests() (result []struct {
	S    []int
	I, J int
	Want []int
}) {
	return []struct {
		S    []int
		I, J int
		Want []int
	}{
		{
			[]int{1, 2, 3},
			0,
			0,
			[]int{1, 2, 3},
		},
		{
			[]int{1, 2, 3},
			0,
			1,
			[]int{2, 3},
		},
		{
			[]int{1, 2, 3},
			3,
			3,
			[]int{1, 2, 3},
		},
		{
			[]int{1, 2, 3},
			0,
			2,
			[]int{3},
		},
		{
			[]int{1, 2, 3},
			0,
			3,
			[]int{},
		},
	}
}

// Test_Standard_Library_Delete ports the Go standard library test.
func Test_Standard_Library_Delete(t *testing.T) {
	for _, test := range delete_tests() {
		copy := standard_clone(test.S)
		{

			got := standard_delete(copy, test.I, test.J)
			testify.True(t, standard_equal(got, test.Want),

				"standard_delete(%v, %d, %d) = %v, want %v",
				test.S,
				test.I,
				test.J,
				got,
				test.Want)
		}
	}
}

func delete_function_tests() (result []struct {
	S    []int
	Fn   func(int) (result_83 bool)
	Want []int
}) {
	return []struct {
		S    []int
		Fn   func(int) (result_83 bool)
		Want []int
	}{
		{
			nil,
			func(int) (result_84 bool) { return true },
			nil,
		},
		{
			[]int{1, 2, 3},
			func(int) (result_85 bool) { return true },
			nil,
		},
		{
			[]int{1, 2, 3},
			func(int) (result_86 bool) { return false },
			[]int{1, 2, 3},
		},
		{
			[]int{1, 2, 3},
			func(i int) (result_87 bool) { return i > 2 },
			[]int{1, 2},
		},
		{
			[]int{1, 2, 3},
			func(i int) (result_88 bool) { return i < 2 },
			[]int{2, 3},
		},
		{
			[]int{10, 2, 30},
			func(i int) (result_89 bool) { return i >= 10 },
			[]int{2},
		},
	}
}

// Test_Standard_Library_Delete_Func ports the Go standard library test.
func Test_Standard_Library_Delete_Function(t *testing.T) {
	for i, test := range delete_function_tests() {
		copy := standard_clone(test.S)
		{

			got := standard_delete_function(copy, test.Fn)
			testify.True(t, standard_equal(got, test.Want),

				"standard_delete_function case %d: got %v, want %v",
				i,
				got,
				test.Want)
		}
	}
}

// Test_Standard_Library_Delete_Panics ports the Go standard library test.
func Test_Standard_Library_Delete_Panics(t *testing.T) {
	s := []int{0, 1, 2, 3, 4}
	s = s[0:2]
	standard_accept(s[0:4]) // this is a valid slice of s

	for _, test := range []struct {
		Name string
		S    []int
		I, J int
	}{
		{"with negative first index", []int{42}, -2, 1},
		{"with negative second index", []int{42}, 1, -1},
		{"with out-of-bounds first index", []int{42}, 2, 3},
		{"with out-of-bounds second index", []int{42}, 0, 2},
		{"with out-of-bounds both indexes", []int{42}, 2, 2},
		{"with invalid i>j", []int{42}, 1, 0},
		{"s[i:j] is valid and j > len(s)", s, 0, 4},
		{"s[i:j] is valid and i == j > len(s)", s, 3, 3},
	} {
		testify.Panics(t, func() { standard_delete(test.S, test.I, test.J) },
			"standard_delete %s: got no panic, want panic", test.Name)
	}
}

// Test_Standard_Library_Delete_Clear_Tail ports the Go standard library test.
func Test_Standard_Library_Delete_Clear_Tail(t *testing.T) {
	memory := []*int{new(int), new(int), new(int), new(int), new(int), new(int)}
	s := memory[0:5] // there is 1 element beyond len(s), within cap(s)

	s = standard_delete(s, 2, 4)

	// A discarded pointer must not keep its object alive.
	testify.Nil(t, memory[3],
		"standard_delete: want nil discarded element, got %v", memory[3])
	testify.False(t, memory[4] != nil,
		"standard_delete: want nil discarded element, got %v", memory[4])
	testify.False(t, memory[5] == nil,
		"standard_delete: want unchanged elements beyond original len, got nil")
}

// Test_Standard_Library_Delete_Func_Clear_Tail ports the Go standard library test.
func Test_Standard_Library_Delete_Function_Clear_Tail(t *testing.T) {
	memory := []*int{new(int), new(int), new(int), new(int), new(int), new(int)}
	*memory[2], *memory[3] = 42, 42
	s := memory[0:5] // there is 1 element beyond len(s), within cap(s)

	s = standard_delete_function(s, func(i *int) (result_90 bool) {
		return i != nil && *i == 42
	})

	// A discarded pointer must not keep its object alive.
	testify.Nil(t, memory[3],
		"standard_delete_function: want nil discarded element, got %v", memory[3])
	testify.False(t, memory[4] != nil,
		"standard_delete_function: want nil discarded element, got %v", memory[4])
	testify.False(t, memory[5] == nil,

		"standard_delete_function: want unchanged "+
			"elements beyond original len, got nil")
}

// Test_Standard_Library_Clone ports the Go standard library test.
func Test_Standard_Library_Clone(t *testing.T) {
	s1 := []int{1, 2, 3}
	s2 := standard_clone(s1)
	testify.True(t, standard_equal(s1, s2),
		"standard_clone(%v) = %v, want %v", s1, s2, s1)
	s1[0] = 4
	want := []int{1, 2, 3}
	testify.True(t, standard_equal(s2, want),
		"standard_clone(%v) changed unexpectedly to %v", want, s2)
	{

		got := standard_clone([]int(nil))
		testify.False(t, got != nil,
			"standard_clone(nil) = %#v, want nil", got)
	}
	got := standard_clone(s1[:0])
	testify.Not_Nil(t, got, "standard_clone(%v) = nil, want a nonnil slice", s1[:0])
	testify.Count(t, got, 0,
		"standard_clone(%v) = %#v, want an empty slice", s1[:0], got)
}

func compact_tests() (result []struct {
	Name string
	S    []int
	Want []int
}) {
	return []struct {
		Name string
		S    []int
		Want []int
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"one",
			[]int{1},
			[]int{1},
		},
		{
			"sorted",
			[]int{1, 2, 3},
			[]int{1, 2, 3},
		},
		{
			"2 items",
			[]int{1, 1, 2},
			[]int{1, 2},
		},
		{
			"unsorted",
			[]int{1, 2, 1},
			[]int{1, 2, 1},
		},
		{
			"many",
			[]int{1, 2, 2, 3, 3, 4},
			[]int{1, 2, 3, 4},
		},
	}
}

// Test_Standard_Library_Compact ports the Go standard library test.
func Test_Standard_Library_Compact(t *testing.T) {
	for _, test := range compact_tests() {
		copy := standard_clone(test.S)
		{

			got := standard_compact(copy)
			testify.True(t, standard_equal(got, test.Want),
				"standard_compact(%v) = %v, want %v", test.S, got, test.Want)
		}
	}
}

// Benchmark_Standard_Library_Compact ports the Go standard library benchmark.
func Benchmark_Standard_Library_Compact(b *testing.B) {
	for _, c := range compact_tests() {
		b.Run(c.Name, func(b *testing.B) {
			ss := make([]int, 0, 64)
			for k_index := 0; k_index < b.N; k_index++ {
				ss = ss[:0]
				ss = append(ss, c.S...)
				standard_compact(ss)
			}
		})
	}
}

// Benchmark_Standard_Library_Compact_Large ports the Go standard library benchmark.
func Benchmark_Standard_Library_Compact_Large(b *testing.B) {
	const LARGE_WORD_COUNT = 16
	const N_COUNT = 1024
	type Large [LARGE_WORD_COUNT]int

	b.Run("all_dup", func(b *testing.B) {
		ss := make([]Large, N_COUNT)
		b.ResetTimer()
		for i_index := 0; i_index < b.N; i_index++ {
			standard_compact(ss)
		}
	})
	b.Run("no_dup", func(b *testing.B) {
		ss := make([]Large, N_COUNT)
		for i := range ss {
			ss[i][0] = i
		}
		b.ResetTimer()
		for i_index := 0; i_index < b.N; i_index++ {
			standard_compact(ss)
		}
	})
}

// Test_Standard_Library_Compact_Func ports the Go standard library test.
func Test_Standard_Library_Compact_Function(t *testing.T) {
	for _, test := range compact_tests() {
		copy := standard_clone(test.S)
		{

			got := standard_compact_function(
				copy,
				equal[int],
			)
			testify.True(t, standard_equal(got, test.Want),

				"standard_compact_function(%v, equal[int]) = %v, want %v",
				test.S,
				got,
				test.Want)
		}
	}

	s1 := []string{"a", "a", "A", "B", "b"}
	copy := standard_clone(s1)
	want := []string{"a", "B"}
	{

		got := standard_compact_function(copy, strings.EqualFold)
		testify.True(t, standard_equal(got, want),

			"standard_compact_function(%v, strings.EqualFold) = %v, want %v",
			s1,
			got,
			want)
	}
}

// Test_Standard_Library_Compact_Clear_Tail ports the Go standard library test.
func Test_Standard_Library_Compact_Clear_Tail(t *testing.T) {
	one, two, three, four := 1, 2, 3, 4
	memory := []*int{&one, &one, &two, &two, &three, &four}
	s := memory[0:5] // there is 1 element beyond len(s), within cap(s)
	copy := standard_clone(s)

	s = standard_compact(s)

	{

		want := []*int{&one, &two, &three}
		testify.True(t, standard_equal(s, want),
			"standard_compact(%v) = %v, want %v", copy, s, want)
	}

	// A discarded pointer must not keep its object alive.
	testify.Nil(t, memory[3],
		"standard_compact: want nil discarded element, got %v", memory[3])
	testify.False(t, memory[4] != nil,
		"standard_compact: want nil discarded element, got %v", memory[4])
	testify.False(t, memory[5] != &four,

		"standard_compact: want unchanged element beyond original len, got %v",
		memory[5])
}

// Test_Standard_Library_Compact_Func_Clear_Tail ports the Go standard library test.
func Test_Standard_Library_Compact_Function_Clear_Tail(t *testing.T) {
	a, b, c, d, e, f := 1, 1, 2, 2, 3, 4
	memory := []*int{&a, &b, &c, &d, &e, &f}
	s := memory[0:5] // there is 1 element beyond len(s), within cap(s)
	copy := standard_clone(s)

	s = standard_compact_function(s, func(x, y *int) (result_91 bool) {
		if x == nil {
			return x == y
		}
		if y == nil {
			return false
		}
		return *x == *y
	})

	{

		want := []*int{&a, &c, &e}
		testify.True(t, standard_equal(s, want),
			"standard_compact_function(%v) = %v, want %v", copy, s, want)
	}

	// A discarded pointer must not keep its object alive.
	testify.Nil(t, memory[3],
		"standard_compact_function: want nil discarded element, got %v", memory[3])
	testify.False(t, memory[4] != nil,
		"standard_compact_function: want nil discarded element, got %v", memory[4])
	testify.False(t, memory[5] != &f,

		"standard_compact_function: want unchanged "+
			"elements beyond original len, got %v",
		memory[5])
}

// Benchmark_Standard_Library_Compact_Func ports the Go standard library benchmark.
func Benchmark_Standard_Library_Compact_Function(b *testing.B) {
	for _, c := range compact_tests() {
		b.Run(c.Name, func(b *testing.B) {
			ss := make([]int, 0, 64)
			for k_index := 0; k_index < b.N; k_index++ {
				ss = ss[:0]
				ss = append(ss, c.S...)
				standard_compact_function(
					ss,
					func(a, b int) (result_92 bool) { return a == b },
				)
			}
		})
	}
}

// Benchmark_Standard_Library_Compact_Func_Large ports the Go standard library benchmark.
func Benchmark_Standard_Library_Compact_Function_Large(b *testing.B) {
	type Element = int
	const N_COUNT = adapted.SLICE_COUNT_MAXIMUM

	b.Run("all_dup", func(b *testing.B) {
		ss := make([]Element, N_COUNT)
		b.ResetTimer()
		for i_index := 0; i_index < b.N; i_index++ {
			standard_compact_function(
				ss,
				func(a, b Element) (result_93 bool) { return a == b },
			)
		}
	})
	b.Run("no_dup", func(b *testing.B) {
		ss := make([]Element, N_COUNT)
		for i := range ss {
			ss[i] = i
		}
		b.ResetTimer()
		for i_index := 0; i_index < b.N; i_index++ {
			standard_compact_function(
				ss,
				func(a, b Element) (result_94 bool) { return a == b },
			)
		}
	})
}

// Test_Standard_Library_Grow ports the Go standard library test.
func Test_Standard_Library_Grow(t *testing.T) {
	s1 := []int{1, 2, 3}

	copy := standard_clone(s1)
	s2 := standard_grow(copy, 1000)
	testify.True(t, standard_equal(s1, s2),
		"standard_grow(%v) = %v, want %v", s1, s2, s1)
	testify.False(t, cap(s2) < 1000+len(s1),
		"after standard_grow(%v) cap = %d, want >= %d", s1, cap(s2), 1000+len(s1))

	// Test mutation of elements between length and capacity.
	copy = standard_clone(s1)
	s3 := standard_grow(copy[:1], 2)[:3]
	testify.True(t, standard_equal(s1, s3),
		"standard_grow should not mutate elements between length and capacity")
	s3 = standard_grow(copy[:1], 1000)[:3]
	testify.True(t, standard_equal(s1, s3),
		"standard_grow should not mutate elements between length and capacity")

	// Test number of allocations.
	{

		n := testing.AllocsPerRun(100, func() { standard_grow(s2, cap(s2)-len(s2)) })
		testify.False(t, n != 0,

			"standard_grow should not allocate when given "+
				"sufficient capacity; allocated %v times",
			n)
	}
	n := testing.AllocsPerRun(100, func() { standard_grow(s2, cap(s2)-len(s2)+1) })
	testify.Equal(t, 1.0, n,
		"standard_grow should allocate once when given insufficient capacity")

	// Test for negative growth sizes.
	var got_panic bool
	func() {
		defer func() { got_panic = recover() != nil }()
		standard_grow(s1, -1)
	}()
	testify.True(t, got_panic,
		"standard_grow(-1) did not panic; expected a panic")
}

// Test_Standard_Library_Clip ports the Go standard library test.
func Test_Standard_Library_Clip(t *testing.T) {
	s1 := []int{1, 2, 3, 4, 5, 6}[:3]
	original := standard_clone(s1)
	testify.False(t, len(s1) != 3,
		"len(%v) = %d, want 3", s1, len(s1))
	testify.False(t, cap(s1) < 6,
		"cap(%v[:3]) = %d, want >= 6", original, cap(s1))
	s2 := standard_clip(s1)
	testify.True(t, standard_equal(s1, s2),
		"standard_clip(%v) = %v, want %v", s1, s2, s1)
	testify.False(t, cap(s2) != 3,
		"cap(standard_clip(%v)) = %d, want 3", original, cap(s2))
}

// Test_Standard_Library_Reverse ports the Go standard library test.
func Test_Standard_Library_Reverse(t *testing.T) {
	even := []int{3, 1, 4, 1, 5, 9} // len = 6
	standard_reverse(even)
	{

		want := []int{9, 5, 1, 4, 1, 3}
		testify.True(t, standard_equal(even, want),
			"standard_reverse(even) = %v, want %v", even, want)
	}

	odd := []int{3, 1, 4, 1, 5, 9, 2} // len = 7
	standard_reverse(odd)
	{

		want := []int{2, 9, 5, 1, 4, 1, 3}
		testify.True(t, standard_equal(odd, want),
			"standard_reverse(odd) = %v, want %v", odd, want)
	}

	words := strings.Fields("one two three")
	standard_reverse(words)
	{

		want := strings.Fields("three two one")
		testify.True(t, standard_equal(words, want),
			"standard_reverse(words) = %v, want %v", words, want)
	}

	singleton := []string{"one"}
	standard_reverse(singleton)
	{

		want := []string{"one"}
		testify.True(t, standard_equal(singleton, want),
			"standard_reverse(singleton) = %v, want %v", singleton, want)
	}

	standard_reverse[[]string](nil)
}

// NaiveReplace is a baseline implementation to the standard_replace function.
func naive_replace[S ~[]E, E any](s S, i, j int, v ...E) (result_95 S) {
	s = standard_delete(s, i, j)
	s = standard_insert(s, i, v...)
	return s
}

// Test_Standard_Library_Replace ports the Go standard library test.
func Test_Standard_Library_Replace(t *testing.T) {
	for _, test := range []struct {
		S, V []int
		I, J int
	}{
		{}, // all zero value
		{
			S: []int{1, 2, 3, 4},
			V: []int{5},
			I: 1,
			J: 2,
		},
		{
			S: []int{1, 2, 3, 4},
			V: []int{5, 6, 7, 8},
			I: 1,
			J: 2,
		},
		{
			S: func() (result_96 []int) {
				s := make([]int, 3, 20)
				s[0] = 0
				s[1] = 1
				s[2] = 2
				return s
			}(),
			V: []int{3, 4, 5, 6, 7},
			I: 0,
			J: 1,
		},
	} {
		ss, vv := standard_clone(test.S), standard_clone(test.V)
		want := naive_replace(ss, test.I, test.J, vv...)
		got := standard_replace(test.S, test.I, test.J, test.V...)
		testify.True(t, standard_equal(got, want),

			"standard_replace(%v, %v, %v, %v) = %v, want %v",
			test.S,
			test.I,
			test.J,
			test.V,
			got,
			want)
	}
}

// Test_Standard_Library_Replace_Panics ports the Go standard library test.
func Test_Standard_Library_Replace_Panics(t *testing.T) {
	s := []int{0, 1, 2, 3, 4}
	s = s[0:2]
	standard_accept(s[0:4]) // this is a valid slice of s

	for _, test := range []struct {
		Name string
		S, V []int
		I, J int
	}{
		{"indexes out of order", []int{1, 2}, []int{3}, 2, 1},
		{"large index", []int{1, 2}, []int{3}, 1, 10},
		{"negative index", []int{1, 2}, []int{3}, -1, 2},
		{"s[i:j] is valid and j > len(s)", s, nil, 0, 4},
	} {
		ss, vv := standard_clone(test.S), standard_clone(test.V)
		testify.Panics(t, func() { standard_replace(ss, test.I, test.J, vv...) },
			"standard_replace %s: should have panicked", test.Name)
	}
}

// Test_Standard_Library_Replace_Grow ports the Go standard library test.
func Test_Standard_Library_Replace_Grow(t *testing.T) {
	// When standard_replace needs to allocate a new slice, we want the original slice
	// to not be changed.
	a, b, c, d, e, f := 1, 2, 3, 4, 5, 6
	memory := []*int{&a, &b, &c, &d, &e, &f}
	memcopy := standard_clone(memory)
	s := memory[0:5] // there is 1 element beyond len(s), within cap(s)
	copy := standard_clone(s)
	original := s

	// The new elements don't fit within cap(s), so standard_replace will allocate.
	z := 99
	s = standard_replace(s, 1, 3, &z, &z, &z, &z)

	{

		want := []*int{&a, &z, &z, &z, &z, &d, &e}
		testify.True(t, standard_equal(s, want),

			"standard_replace(%v, 1, 3, %v, %v, %v, %v) = %v, want %v",
			copy,
			&z,
			&z,
			&z,
			&z,
			s,
			want)
	}

	testify.True(t, standard_equal(original, copy),
		"original slice has changed, got %v, want %v", original, copy)

	// The replacement must not change the original tail s[len(s):cap(s)].
	testify.True(t, standard_equal(memory, memcopy),
		"original backing memory has changed, got %v, want %v", memory, memcopy)
}

// Test_Standard_Library_Replace_Clear_Tail ports the Go standard library test.
func Test_Standard_Library_Replace_Clear_Tail(t *testing.T) {
	a, b, c, d, e, f := 1, 2, 3, 4, 5, 6
	memory := []*int{&a, &b, &c, &d, &e, &f}
	s := memory[0:5] // there is 1 element beyond len(s), within cap(s)
	copy := standard_clone(s)

	y, z := 8, 9
	s = standard_replace(s, 1, 4, &y, &z)

	{

		want := []*int{&a, &y, &z, &e}
		testify.True(t, standard_equal(s, want),
			"standard_replace(%v) = %v, want %v", copy, s, want)
	}

	// A discarded pointer must not keep its object alive.
	testify.Nil(t, memory[4],
		"standard_replace: want nil discarded element, got %v", memory[4])
	testify.False(t, memory[5] != &f,

		"standard_replace: want unchanged elements beyond original len, got %v",
		memory[5])
}

// Test_Standard_Library_Replace_Overlap ports the Go standard library test.
func Test_Standard_Library_Replace_Overlap(t *testing.T) {
	const N_COUNT = 10
	a := make([]int, N_COUNT)
	want := make([]int, 2*N_COUNT)
	for n := 0; n <= N_COUNT; n++ { // length
		for i := 0; i <= n; i++ { // insertion point 1
			for j := i; j <= n; j++ { // insertion point 2
				for x := 0; x <= N_COUNT; x++ { // start of inserted data
					for y := x; y <= N_COUNT; y++ { // end of inserted data
						for k_index := 0; k_index < N_COUNT; k_index++ {
							a[k_index] = k_index
						}
						want = want[:0]
						want = append(want, a[:i]...)
						want = append(want, a[x:y]...)
						want = append(want, a[j:n]...)
						got := standard_replace(a[:n], i, j, a[x:y]...)
						testify.True(t, standard_equal(got, want),
							"overlap failed: n=%d i=%d j=%d "+
								"x=%d y=%d got=%v want=%v",
							n,
							i,
							j,
							x,
							y,
							got,
							want)
					}
				}
			}
		}
	}
}

// Test_Standard_Library_Replace_End_Clear_Tail ports the Go standard library test.
func Test_Standard_Library_Replace_End_Clear_Tail(t *testing.T) {
	s := []int{11, 22, 33}
	v := []int{99}
	// Case when j == len(s).
	i, j := 1, 3
	s = standard_replace(s, i, j, v...)

	x := s[:3][2]
	{

		want := 0
		testify.False(t, x != want,
			"TestReplaceEndClearTail: obsolete element is %d, want %d", x, want)
	}
}

// Benchmark_Standard_Library_Replace ports the Go standard library benchmark.
func Benchmark_Standard_Library_Replace(b *testing.B) {
	cases := []struct {
		Name string
		S, V func() (result_97 []int)
		I, J int
	}{
		{
			Name: "fast",
			S: func() (result_98 []int) {
				return make([]int, 100)
			},
			V: func() (result_99 []int) {
				return make([]int, 20)
			},
			I: 10,
			J: 40,
		},
		{
			Name: "slow",
			S: func() (result_100 []int) {
				return make([]int, 100)
			},
			V: func() (result_101 []int) {
				return make([]int, 20)
			},
			I: 0,
			J: 2,
		},
	}

	for _, c := range cases {
		b.Run("naive-"+c.Name, func(b *testing.B) {
			for k_index := 0; k_index < b.N; k_index++ {
				s := c.S()
				v := c.V()
				naive_replace(s, c.I, c.J, v...)
			}
		})
		b.Run("optimized-"+c.Name, func(b *testing.B) {
			for k_index := 0; k_index < b.N; k_index++ {
				s := c.S()
				v := c.V()
				standard_replace(s, c.I, c.J, v...)
			}
		})
	}

}

// Test_Standard_Library_Insert_Growth_Rate ports the Go standard library test.
func Test_Standard_Library_Insert_Growth_Rate(t *testing.T) {
	b := make([]byte, 1)
	capacity_limit_count := cap(b)
	n_grow := 0
	const N = adapted.SLICE_COUNT_MAXIMUM - 2
	for i_index := 0; i_index < N; i_index++ {
		b = standard_insert(b, len(b)-1, 0)
		if cap(b) > capacity_limit_count {
			capacity_limit_count = cap(b)
			n_grow++
		}
	}
	want := int(math.Log(N) / math.Log(1.25)) // 1.25 == growth rate for large slices
	testify.False(t, n_grow > want,
		"too many grows. got:%d want:%d", n_grow, want)
}

// Test_Standard_Library_Replace_Growth_Rate ports the Go standard library test.
func Test_Standard_Library_Replace_Growth_Rate(t *testing.T) {
	b := make([]byte, 2)
	capacity_limit_count := cap(b)
	n_grow := 0
	const N = adapted.SLICE_COUNT_MAXIMUM - 2
	for i_index := 0; i_index < N; i_index++ {
		b = standard_replace(b, len(b)-2, len(b)-1, 0, 0)
		if cap(b) > capacity_limit_count {
			capacity_limit_count = cap(b)
			n_grow++
		}
	}
	want := int(math.Log(N) / math.Log(1.25)) // 1.25 == growth rate for large slices
	testify.False(t, n_grow > want,
		"too many grows. got:%d want:%d", n_grow, want)
}

func apply[T any](v T, f func(T)) {
	f(v)
}

// Test type inference with a named slice type.
// Test_Standard_Library_Inference ports the Go standard library test.
func Test_Standard_Library_Inference(t *testing.T) {
	s1 := []int{1, 2, 3}
	apply(s1, standard_reverse)
	{

		want := []int{3, 2, 1}
		testify.True(t, standard_equal(s1, want),
			"standard_reverse(%v) = %v, want %v", []int{1, 2, 3}, s1, want)
	}

	type S []int
	s2 := S([]int{4, 5, 6})
	apply(s2, standard_reverse)
	{

		want := S([]int{6, 5, 4})
		testify.True(t, standard_equal(s2, want),
			"standard_reverse(%v) = %v, want %v", S([]int{4, 5, 6}), s2, want)
	}
}

// Test_Standard_Library_Concat ports the Go standard library test.
func Test_Standard_Library_Concatenation(t *testing.T) {
	cases := []struct {
		S    [][]int
		Want []int
	}{
		{
			S:    [][]int{nil},
			Want: nil,
		},
		{
			S:    [][]int{{1}},
			Want: []int{1},
		},
		{
			S:    [][]int{{1}, {2}},
			Want: []int{1, 2},
		},
		{
			S:    [][]int{{1}, nil, {2}},
			Want: []int{1, 2},
		},
	}
	for _, tc := range cases {
		got := standard_concatenate(tc.S...)
		testify.True(t, standard_equal(tc.Want, got),
			"standard_concatenate(%v) = %v, want %v", tc.S, got, tc.Want)
		var sink []int
		allocs := testing.AllocsPerRun(5, func() {
			sink = standard_concatenate(tc.S...)
		})
		standard_accept(sink)
		testify.True(t, allocs <= 1.0,
			"standard_concatenate(%v) allocated %v times; want at most 1", tc.S, allocs)
	}
}

// Test_Standard_Library_Concat_Too_Large ports the Go standard library test.
func Test_Standard_Library_Concatenation_Too_Large(t *testing.T) {
	// Use zero length element to minimize memory in testing.
	type void struct{}
	cases := []struct {
		Lengths      []int
		Should_Panic bool
	}{
		{
			Lengths:      []int{0, 0},
			Should_Panic: false,
		},
		{
			Lengths:      []int{adapted.SLICE_COUNT_MAXIMUM, 0},
			Should_Panic: false,
		},
		{
			Lengths:      []int{0, adapted.SLICE_COUNT_MAXIMUM},
			Should_Panic: false,
		},
		{
			Lengths:      []int{adapted.SLICE_COUNT_MAXIMUM - 1, 1},
			Should_Panic: false,
		},
		{
			Lengths:      []int{adapted.SLICE_COUNT_MAXIMUM - 1, 1, 1},
			Should_Panic: true,
		},
		{
			Lengths:      []int{adapted.SLICE_COUNT_MAXIMUM, 1},
			Should_Panic: true,
		},
		{
			Lengths: []int{
				adapted.SLICE_COUNT_MAXIMUM,
				adapted.SLICE_COUNT_MAXIMUM,
			},
			Should_Panic: true,
		},
	}
	for _, tc := range cases {
		var r any
		ss := make([][]void, 0, len(tc.Lengths))
		for _, l_count := range tc.Lengths {
			s := make([]void, l_count)
			ss = append(ss, s)
		}
		func() {
			defer func() {
				r = recover()
			}()
			standard_concatenate(ss...)
		}()
		{

			did_panic := r != nil
			testify.False(t, did_panic != tc.Should_Panic,
				"standard_concatenate(lens(%v)) got panic == %v",
				tc.Lengths, did_panic)
		}
	}
}

// Test_Standard_Library_Repeat ports the Go standard library test.
func Test_Standard_Library_Repeat(t *testing.T) {
	standard_repeat_normal_cases(t)
	standard_repeat_big_cases(t)
}

func standard_repeat_normal_cases(t *testing.T) {
	// Normal cases.
	for _, tc := range []struct {
		X     []int
		Count int
		Want  []int
	}{
		{X: []int(nil), Count: 0, Want: []int{}},
		{X: []int(nil), Count: 1, Want: []int{}},
		{X: []int(nil), Count: adapted.SLICE_COUNT_MAXIMUM, Want: []int{}},
		{X: []int{}, Count: 0, Want: []int{}},
		{X: []int{}, Count: 1, Want: []int{}},
		{X: []int{}, Count: adapted.SLICE_COUNT_MAXIMUM, Want: []int{}},
		{X: []int{0}, Count: 0, Want: []int{}},
		{X: []int{0}, Count: 1, Want: []int{0}},
		{X: []int{0}, Count: 2, Want: []int{0, 0}},
		{X: []int{0}, Count: 3, Want: []int{0, 0, 0}},
		{X: []int{0}, Count: 4, Want: []int{0, 0, 0, 0}},
		{X: []int{0, 1}, Count: 0, Want: []int{}},
		{X: []int{0, 1}, Count: 1, Want: []int{0, 1}},
		{X: []int{0, 1}, Count: 2, Want: []int{0, 1, 0, 1}},
		{X: []int{0, 1}, Count: 3, Want: []int{0, 1, 0, 1, 0, 1}},
		{X: []int{0, 1}, Count: 4, Want: []int{0, 1, 0, 1, 0, 1, 0, 1}},
		{X: []int{0, 1, 2}, Count: 0, Want: []int{}},
		{X: []int{0, 1, 2}, Count: 1, Want: []int{0, 1, 2}},
		{X: []int{0, 1, 2}, Count: 2, Want: []int{0, 1, 2, 0, 1, 2}},
		{X: []int{0, 1, 2}, Count: 3, Want: []int{0, 1, 2, 0, 1, 2, 0, 1, 2}},
		{X: []int{0, 1, 2}, Count: 4, Want: []int{0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2}},
	} {
		got := standard_repeat(tc.X, tc.Count)
		matches := got != nil
		matches = matches && cap(got) == cap(tc.Want)
		matches = matches && standard_equal(got, tc.Want)
		testify.True(t, matches,
			"standard_repeat(%v, %v): got: %v, want: %v, "+
				"(got == nil): %v, cap(got): %v, cap(want): "+
				"%v",
			tc.X, tc.Count, got, tc.Want, got == nil, cap(got), cap(tc.Want))
	}
}

func standard_repeat_big_cases(t *testing.T) {
	// Big slices.
	for _, tc := range []struct {
		X     []struct{}
		Count int
		Want  []struct{}
	}{
		{
			X:     make([]struct{}, adapted.SLICE_COUNT_MAXIMUM/1-0),
			Count: 1,
			Want:  make([]struct{}, 1*(adapted.SLICE_COUNT_MAXIMUM/1-0)),
		},
		{
			X:     make([]struct{}, adapted.SLICE_COUNT_MAXIMUM/2-1),
			Count: 2,
			Want:  make([]struct{}, 2*(adapted.SLICE_COUNT_MAXIMUM/2-1)),
		},
		{
			X:     make([]struct{}, adapted.SLICE_COUNT_MAXIMUM/3-2),
			Count: 3,
			Want:  make([]struct{}, 3*(adapted.SLICE_COUNT_MAXIMUM/3-2)),
		},
		{
			X:     make([]struct{}, adapted.SLICE_COUNT_MAXIMUM/4-3),
			Count: 4,
			Want:  make([]struct{}, 4*(adapted.SLICE_COUNT_MAXIMUM/4-3)),
		},
		{
			X:     make([]struct{}, adapted.SLICE_COUNT_MAXIMUM/5-4),
			Count: 5,
			Want:  make([]struct{}, 5*(adapted.SLICE_COUNT_MAXIMUM/5-4)),
		},
		{
			X:     make([]struct{}, adapted.SLICE_COUNT_MAXIMUM/6-5),
			Count: 6,
			Want:  make([]struct{}, 6*(adapted.SLICE_COUNT_MAXIMUM/6-5)),
		},
		{
			X:     make([]struct{}, adapted.SLICE_COUNT_MAXIMUM/7-6),
			Count: 7,
			Want:  make([]struct{}, 7*(adapted.SLICE_COUNT_MAXIMUM/7-6)),
		},
		{
			X:     make([]struct{}, adapted.SLICE_COUNT_MAXIMUM/8-7),
			Count: 8,
			Want:  make([]struct{}, 8*(adapted.SLICE_COUNT_MAXIMUM/8-7)),
		},
		{
			X:     make([]struct{}, adapted.SLICE_COUNT_MAXIMUM/9-8),
			Count: 9,
			Want:  make([]struct{}, 9*(adapted.SLICE_COUNT_MAXIMUM/9-8)),
		},
	} {
		got := standard_repeat(tc.X, tc.Count)
		matches := got != nil
		matches = matches && len(got) == len(tc.Want)
		matches = matches && cap(got) == cap(tc.Want)
		testify.True(t, matches,
			"standard_repeat(make([]struct{}, %v), %v): "+
				"(got == nil): %v, len(got): %v, len(want): "+
				"%v, cap(got): %v, cap(want): %v",
			len(tc.X), tc.Count, got == nil, len(got), len(
				tc.Want,
			), cap(got), cap(tc.Want))
	}
}

// Test_Standard_Library_Repeat_Panics ports the Go standard library test.
func Test_Standard_Library_Repeat_Panics(t *testing.T) {
	for _, test := range []struct {
		Name  string
		X     []struct{}
		Count int
	}{
		{Name: "cannot be negative", X: make([]struct{}, 0), Count: -1},
		{
			Name:  "the result of (len(x) * count) overflows, hi > 0",
			X:     make([]struct{}, 3),
			Count: adapted.SLICE_COUNT_MAXIMUM,
		},
		{
			Name:  "the result of (len(x) * count) overflows, lo > maxInt",
			X:     make([]struct{}, 2),
			Count: 1 + adapted.SLICE_COUNT_MAXIMUM/2,
		},
	} {
		testify.Panics(t, func() { standard_repeat(test.X, test.Count) },
			"standard_repeat %s: got no panic, want panic", test.Name)
	}
}

// Test_Standard_Library_Issue68488 ports the Go standard library test.
func Test_Standard_Library_Issue68488(t *testing.T) {
	s := make([]int, 3)
	// The cleanup closure makes the backing array escape, which preserves the issue's
	// heap-allocation condition without package-level mutable state.
	t.Cleanup(func() { runtime.KeepAlive(&s[1]) })
	clone := standard_clone(s[1:1])
	clone_data := unsafe.SliceData(clone)
	overlaps := clone_data == &s[0] || clone_data == &s[1] || clone_data == &s[2]
	testify.False(t, overlaps, "clone keeps alive s due to array overlap")
}

// This test asserts the behavior when the primary slice operand is nil.
//
// Some operations preserve the nilness of their operand while others
// do not, but in all cases the behavior is documented.
// Test_Standard_Library_Nilness ports the Go standard library test.
func Test_Standard_Library_Nilness(t *testing.T) {
	var (
		empty_slice    = []int{}
		nil_slice      = []int(nil)
		empty_sequence = func(yield func(int) (result_102 bool)) {}
		truth          = func(int) (result_103 bool) { return true }
		equality       = func(x, y int) (result_104 bool) { panic("unreachable") }
	)
	standard_nil_update_cases(t, empty_slice, nil_slice, empty_sequence, truth, equality)
	standard_nil_collection_cases(t, empty_slice, nil_slice, empty_sequence)
}

func standard_nil_update_cases(
	t *testing.T,
	empty_slice []int,
	nil_slice []int,
	empty_sequence iter.Seq[int],
	truth func(int) (selected bool),
	equality func(int, int) (equal bool),
) {
	// These update functions preserve nilness, as a subslice does.
	standard_want_nil(
		t,
		standard_append_sequence(nil_slice, empty_sequence),
		"standard_append_sequence(nil, empty)",
	)
	standard_want_non_nil(
		t,
		standard_append_sequence(empty_slice, empty_sequence),
		"standard_append_sequence(nil, empty)",
	)
	standard_want_nil(t, standard_insert(nil_slice, 0), "standard_insert(nil, 0)")
	standard_want_non_nil(t, standard_insert(empty_slice, 0), "standard_insert(empty, 0)")
	standard_want_nil(t, standard_delete(nil_slice, 0, 0), "standard_delete(nil, 0, 0)")
	standard_want_non_nil(t, standard_delete(empty_slice, 0, 0), "standard_delete(empty, 0, 0)")
	standard_want_non_nil(t, standard_delete([]int{1}, 0, 1), "standard_delete([]int{1}, 0, 1)")

	standard_want_nil(
		t,
		standard_delete_function(nil_slice, truth),
		"standard_delete_function(nil, f)",
	)
	standard_want_non_nil(
		t,
		standard_delete_function(empty_slice, truth),
		"standard_delete_function(empty, f)",
	)
	standard_want_non_nil(
		t,
		standard_delete_function([]int{1}, truth),
		"standard_delete_function([]int{1}, truth)",
	)

	standard_want_nil(t, standard_replace(nil_slice, 0, 0), "standard_replace(nil, 0, 0)")
	standard_want_non_nil(
		t,
		standard_replace(empty_slice, 0, 0),
		"standard_replace(empty, 0, 0)",
	)
	standard_want_non_nil(
		t,
		standard_replace([]int{1}, 0, 1),
		"standard_replace([]int{1}, 0, 1)",
	)

	standard_want_nil(t, standard_clone(nil_slice), "standard_clone(nil)")
	standard_want_non_nil(t, standard_clone(empty_slice), "standard_clone(empty)")

	standard_want_nil(t, standard_compact(nil_slice), "standard_compact(nil)")
	standard_want_non_nil(t, standard_compact(empty_slice), "standard_compact(empty)")

	standard_want_nil(
		t,
		standard_compact_function(nil_slice, equality),
		"standard_compact_function(nil)",
	)
	standard_want_non_nil(
		t,
		standard_compact_function(empty_slice, equal),
		"standard_compact_function(empty)",
	)

	standard_want_nil(t, standard_grow(nil_slice, 0), "standard_grow(nil, 0)")
	standard_want_non_nil(t, standard_grow(empty_slice, 0), "standard_grow(empty, 0)")

	standard_want_nil(t, standard_clip(nil_slice), "standard_clip(nil)")
	standard_want_non_nil(t, standard_clip(empty_slice), "standard_clip(empty)")
	standard_want_non_nil(t, standard_clip([]int{1}[:0:0]), "standard_clip([]int{1}[:0:0])")
}

func standard_nil_collection_cases(
	t *testing.T,
	empty_slice []int,
	nil_slice []int,
	empty_sequence iter.Seq[int],
) {
	// Standard_concatenate returns nil iff the result is empty.
	// This is an unfortunate irregularity.
	standard_want_nil(
		t,
		standard_concatenate(nil_slice, empty_slice, nil_slice, empty_slice),
		"standard_concatenate(nil, ...empty...)",
	)
	standard_want_nil(
		t,
		standard_concatenate(empty_slice, empty_slice, nil_slice, empty_slice),
		"standard_concatenate(empty, ...empty...)",
	)
	standard_want_nil(t, standard_concatenate[[]int](), "standard_concatenate()")

	// Standard_repeat never returns nil. Another irregularity.
	standard_want_non_nil(t, standard_repeat(nil_slice, 0), "standard_repeat(nil, 0)")
	standard_want_non_nil(t, standard_repeat(empty_slice, 0), "standard_repeat(empty, 0)")
	standard_want_non_nil(t, standard_repeat(nil_slice, 2), "standard_repeat(nil, 2)")
	standard_want_non_nil(t, standard_repeat(empty_slice, 2), "standard_repeat(empty, 2)")

	// These collection functions return nil for an empty sequence.
	standard_want_nil(t, standard_collect(empty_sequence), "standard_collect(empty)")

	standard_want_nil(t, standard_sorted(empty_sequence), "standard_sorted(empty)")
	standard_want_nil(
		t,
		standard_sorted_function(empty_sequence, cmp.Compare),
		"standard_sorted_function(empty)",
	)
	standard_want_nil(
		t,
		standard_sorted_stable_function(empty_sequence, cmp.Compare),
		"standard_sorted_stable_function(empty)",
	)
}

func standard_want_nil(t *testing.T, slice []int, condition string) {
	t.Helper()
	testify.False(t, slice != nil,
		"%s != nil", condition)
}

func standard_want_non_nil(t *testing.T, slice []int, condition string) {
	t.Helper()
	testify.False(t, slice == nil,
		"%s == nil", condition)
}

func standard_integers() (result []int) {
	return []int{74, 59, 238, -784, 9845, 959, 905, 0, 0, 42, 7586, -5467984, 7586}
}

func standard_strings() (result []string) {
	return []string{"", "Hello", "foo", "bar", "foo", "f00", "%*&^*&^&", "***"}
}

// Test_Standard_Library_Sort_Int_Slice ports the Go standard library test.
func Test_Standard_Library_Sort_Int_Slice(t *testing.T) {
	integers := standard_integers()
	data := standard_clone(integers)
	standard_sort(data)
	testify.True(t, standard_is_sorted(data), "sorted %v; got %v", integers, data)
}

// Test_Standard_Library_Sort_Func_Int_Slice ports the Go standard library test.
func Test_Standard_Library_Sort_Function_Int_Slice(t *testing.T) {
	integers := standard_integers()
	data := standard_clone(integers)
	standard_sort_function(data, func(a, b int) (result_105 int) { return a - b })
	testify.True(t, standard_is_sorted(data), "sorted %v; got %v", integers, data)
}

// Test_Standard_Library_Sort_Float64_Slice ports the Go standard library test.
func Test_Standard_Library_Sort_Float64_Slice(t *testing.T) {
	floating_values := standard_elements(
		74.3, 59.0, math.Inf(1), 238.2, -784.0, 2.3, math.Inf(-1), 9845.768,
		-959.7485, 905.0, 7.8, 7.8, 74.3, 59.0, math.Inf(1), 238.2, -784.0, 2.3,
	)
	data := standard_clone(floating_values)
	standard_sort(data)
	testify.True(t, standard_is_sorted(data), "sorted %v; got %v", floating_values, data)
}

// Test_Standard_Library_Sort_String_Slice ports the Go standard library test.
func Test_Standard_Library_Sort_String_Slice(t *testing.T) {
	strings := standard_strings()
	data := standard_clone(strings)
	standard_sort(data)
	testify.True(t, standard_is_sorted(data), "sorted %v; got %v", strings, data)
}

// Test_Standard_Library_Sort_Large_Random ports the Go standard library test.
func Test_Standard_Library_Sort_Large_Random(t *testing.T) {
	n_count := adapted.SLICE_COUNT_MAXIMUM
	if testing.Short() {
		n_count /= 100
	}
	data := make([]int, n_count)
	for i_index := 0; i_index < len(data); i_index++ {
		data[i_index] = rand.Intn(100)
	}
	testify.False(t, standard_is_sorted(data),
		"terrible rand.rand")
	standard_sort(data)
	testify.True(t, standard_is_sorted(data),
		"sort did not sort the boundary slice")
}

type int_pair struct {
	A, B int
}

type int_pairs []int_pair

// Pairs compare on a only.
func int_pair_cmp(x, y int_pair) (result_106 int) {
	return x.A - y.A
}

// The original order lets the stability checks detect equal-key movement.
func initialize_int_pair_order(d int_pairs) {
	for i := range d {
		d[i].B = i
	}
}

// Reports whether equal-key elements retain their expected relative order.
func int_pairs_in_order(d int_pairs, reversed bool) (result_107 bool) {
	last_a, last_b := -1, 0
	for i_index := 0; i_index < len(d); i_index++ {
		if last_a != d[i_index].A {
			last_a = d[i_index].A
			last_b = d[i_index].B
			continue
		}
		if !reversed {
			if d[i_index].B <= last_b {
				return false
			}
		} else {
			if d[i_index].B >= last_b {
				return false
			}
		}
		last_b = d[i_index].B
	}
	return true
}

// Test_Standard_Library_Stability ports the Go standard library test.
func Test_Standard_Library_Stability(t *testing.T) {
	n, m := adapted.SLICE_COUNT_MAXIMUM, 1000
	if testing.Short() {
		n, m = 1000, 100
	}
	data := make(int_pairs, n)

	// Random distribution.
	for i_index := 0; i_index < len(data); i_index++ {
		data[i_index].A = rand.Intn(m)
	}
	testify.False(t, standard_is_sorted_function(data, int_pair_cmp),
		"terrible rand.rand")
	initialize_int_pair_order(data)
	standard_sort_stable_function(data, int_pair_cmp)
	testify.True(t, standard_is_sorted_function(data, int_pair_cmp),
		"Stable didn't sort %d ints", n)
	testify.True(t, int_pairs_in_order(data, false),
		"Stable wasn't stable on %d ints", n)

	// Already sorted.
	initialize_int_pair_order(data)
	standard_sort_stable_function(data, int_pair_cmp)
	testify.True(t, standard_is_sorted_function(data, int_pair_cmp),
		"Stable shuffled sorted %d ints (order)", n)
	testify.True(t, int_pairs_in_order(data, false),
		"Stable shuffled sorted %d ints (stability)", n)

	// Sorted reversed.
	for i_index := 0; i_index < len(data); i_index++ {
		data[i_index].A = len(data) - i_index
	}
	initialize_int_pair_order(data)
	standard_sort_stable_function(data, int_pair_cmp)
	testify.True(t, standard_is_sorted_function(data, int_pair_cmp),
		"Stable didn't sort %d ints", n)
	testify.True(t, int_pairs_in_order(data, false),
		"Stable wasn't stable on %d ints", n)
}

type S struct {
	A int
	B string
}

func cmp_s(s1, s2 S) (result_108 int) {
	return cmp.Compare(s1.A, s2.A)
}

// Test_Standard_Library_Min_Max ports the Go standard library test.
func Test_Standard_Library_Extrema(t *testing.T) {
	int_cmp := func(a, b int) (result_109 int) { return a - b }

	tests := []struct {
		Data     []int
		Want_Min int
		Want_Max int
	}{
		{[]int{7}, 7, 7},
		{[]int{1, 2}, 1, 2},
		{[]int{2, 1}, 1, 2},
		{[]int{1, 2, 3}, 1, 3},
		{[]int{3, 2, 1}, 1, 3},
		{[]int{2, 1, 3}, 1, 3},
		{[]int{2, 2, 3}, 2, 3},
		{[]int{3, 2, 3}, 2, 3},
		{[]int{0, 2, -9}, -9, 2},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.Data), func(t *testing.T) {
			got_min := standard_minimum(tt.Data)
			testify.False(t, got_min != tt.Want_Min,
				"standard_minimum got %v, want %v", got_min, tt.Want_Min)

			got_function_min := standard_minimum_function(tt.Data, int_cmp)
			testify.False(t, got_function_min != tt.Want_Min,

				"standard_minimum_function got %v, want %v",
				got_function_min,
				tt.Want_Min)

			got_max := standard_maximum(tt.Data)
			testify.False(t, got_max != tt.Want_Max,
				"standard_maximum got %v, want %v", got_max, tt.Want_Max)

			got_function_max := standard_maximum_function(tt.Data, int_cmp)
			testify.False(t, got_function_max != tt.Want_Max,

				"standard_maximum_function got %v, want %v",
				got_function_max,
				tt.Want_Max)
		})
	}

	svals := []S{
		{A: 1, B: "a"},
		{A: 2, B: "a"},
		{A: 1, B: "b"},
		{A: 2, B: "b"},
	}

	got_min := standard_minimum_function(svals, cmp_s)
	want_min := S{A: 1, B: "a"}
	testify.False(t, got_min != want_min,
		"standard_minimum_function(%v) = %v, want %v", svals, got_min, want_min)

	got_max := standard_maximum_function(svals, cmp_s)
	want_max := S{A: 2, B: "a"}
	testify.False(t, got_max != want_max,
		"standard_maximum_function(%v) = %v, want %v", svals, got_max, want_max)
}

// Test_Standard_Library_Min_Max_Na_Ns ports the Go standard library test.
func Test_Standard_Library_Extrema_Na_Ns(t *testing.T) {
	floating_values := standard_elements(1.0, 999.9, 3.14, -400.4, -5.14)
	testify.False(t, standard_minimum(floating_values) != -400.4,
		"got min %v, want -400.4", standard_minimum(floating_values))
	testify.False(t, standard_maximum(floating_values) != 999.9,
		"got max %v, want 999.9", standard_maximum(floating_values))

	// Both extrema functions must propagate a NaN from any position to their output.
	for i_index := 0; i_index < len(floating_values); i_index++ {
		testfs := standard_clone(floating_values)
		testfs[i_index] = math.NaN()

		fmin := standard_minimum(testfs)
		testify.True(t, math.IsNaN(fmin),
			"got min %v, want NaN", fmin)

		fmax := standard_maximum(testfs)
		testify.True(t, math.IsNaN(fmax),
			"got max %v, want NaN", fmax)
	}
}

// Test_Standard_Library_Min_Max_Panics ports the Go standard library test.
func Test_Standard_Library_Extrema_Panics(t *testing.T) {
	int_cmp := func(a, b int) (result_110 int) { return a - b }
	empty_slice := []int{}

	testify.Panics(t, func() { standard_minimum(empty_slice) },
		"standard_minimum([]): got no panic, want panic")

	testify.Panics(t, func() { standard_maximum(empty_slice) },
		"standard_maximum([]): got no panic, want panic")

	testify.Panics(t, func() { standard_minimum_function(empty_slice, int_cmp) },
		"standard_minimum_function([]): got no panic, want panic")

	testify.Panics(t, func() { standard_maximum_function(empty_slice, int_cmp) },
		"standard_maximum_function([]): got no panic, want panic")
}

// Test_Standard_Library_Binary_Search ports the Go standard library test.
func Test_Standard_Library_Binary_Search(t *testing.T) {
	str1 := []string{"foo"}
	str2 := []string{"ab", "ca"}
	str3 := []string{"mo", "qo", "vo"}
	str4 := []string{"ab", "ad", "ca", "xy"}
	// Slice with repeating elements.
	string_repeats := []string{"ba", "ca", "da", "da", "da", "ka", "ma", "ma", "ta"}
	// Slice with all element equal.
	string_same := []string{"xx", "xx", "xx"}

	tests := []struct {
		Data       []string
		Target     string
		Want_Pos   int
		Want_Found bool
	}{
		{[]string{}, "foo", 0, false},
		{[]string{}, "", 0, false},

		{str1, "foo", 0, true},
		{str1, "bar", 0, false},
		{str1, "zx", 1, false},

		{str2, "aa", 0, false},
		{str2, "ab", 0, true},
		{str2, "ad", 1, false},
		{str2, "ca", 1, true},
		{str2, "ra", 2, false},

		{str3, "bb", 0, false},
		{str3, "mo", 0, true},
		{str3, "nb", 1, false},
		{str3, "qo", 1, true},
		{str3, "tr", 2, false},
		{str3, "vo", 2, true},
		{str3, "xr", 3, false},

		{str4, "aa", 0, false},
		{str4, "ab", 0, true},
		{str4, "ac", 1, false},
		{str4, "ad", 1, true},
		{str4, "ax", 2, false},
		{str4, "ca", 2, true},
		{str4, "cc", 3, false},
		{str4, "dd", 3, false},
		{str4, "xy", 3, true},
		{str4, "zz", 4, false},

		{string_repeats, "da", 2, true},
		{string_repeats, "db", 5, false},
		{string_repeats, "ma", 6, true},
		{string_repeats, "mb", 8, false},

		{string_same, "xx", 0, true},
		{string_same, "ab", 0, false},
		{string_same, "zz", 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.Target, func(t *testing.T) {
			position, found := standard_binary_search(tt.Data, tt.Target)
			standard_search_result(t, position, found, tt.Want_Pos, tt.Want_Found)
			position, found = standard_binary_search_function(
				tt.Data,
				tt.Target,
				strings.Compare,
			)
			standard_search_result(t, position, found, tt.Want_Pos, tt.Want_Found)
		})
	}
}

// Test_Standard_Library_Binary_Search_Ints ports the Go standard library test.
func Test_Standard_Library_Binary_Search_Ints(t *testing.T) {
	data := []int{20, 30, 40, 50, 60, 70, 80, 90}
	tests := []struct {
		Target     int
		Want_Pos   int
		Want_Found bool
	}{
		{20, 0, true},
		{23, 1, false},
		{43, 3, false},
		{80, 6, true},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.Target), func(t *testing.T) {
			{
				position, found := standard_binary_search(data, tt.Target)
				standard_search_result(
					t,
					position,
					found,
					tt.Want_Pos,
					tt.Want_Found,
				)
			}

			{
				cmp := func(a, b int) (result_111 int) {
					return a - b
				}
				position, found := standard_binary_search_function(
					data,
					tt.Target,
					cmp,
				)
				standard_search_result(
					t,
					position,
					found,
					tt.Want_Pos,
					tt.Want_Found,
				)
			}
		})
	}
}

// Test_Standard_Library_Binary_Search_Floats ports the Go standard library test.
func Test_Standard_Library_Binary_Search_Floats(t *testing.T) {
	data := standard_elements(math.NaN(), -0.25, 0.0, 1.4)
	standard_binary_search_case(t, data, math.NaN(), 0, true)
	standard_binary_search_case(t, data, math.Inf(-1), 1, false)
	standard_binary_search_case(t, data, -0.25, 1, true)
	standard_binary_search_case(t, data, 0.0, 2, true)
	standard_binary_search_case(t, data, 1.4, 3, true)
	standard_binary_search_case(t, data, 1.5, 4, false)
}

func standard_binary_search_case[Element cmp.Ordered](
	t *testing.T,
	data []Element,
	target Element,
	want_position int,
	want_found bool,
) {
	t.Helper()
	position, found := standard_binary_search(data, target)
	testify.False(t, position != want_position,
		"standard_binary_search position = %v, want %v", position, want_position)
	testify.False(t, found != want_found,
		"standard_binary_search found = %v, want %v", found, want_found)
}

// Test_Standard_Library_Binary_Search_Func ports the Go standard library test.
func Test_Standard_Library_Binary_Search_Function(t *testing.T) {
	data := []int{1, 10, 11, 2} // sorted lexicographically
	cmp := func(a int, b string) (result_112 int) {
		return strings.Compare(strconv.Itoa(a), b)
	}
	position, found := standard_binary_search_function(data, "2", cmp)
	standard_search_result(t, position, found, 3, true)
}

func standard_search_result(
	t *testing.T,
	position int,
	found bool,
	want_position int,
	want_found bool,
) {
	t.Helper()
	testify.False(t, position != want_position,
		"search position = %v, want %v", position, want_position)
	testify.False(t, found != want_found,
		"search found = %v, want %v", found, want_found)
}

// Test_Standard_Library_All ports the Go standard library test.
func Test_Standard_Library_All(t *testing.T) {
	for index := 0; index < 10; index++ {
		var s []int
		for i := range index {
			s = append(s, i)
		}
		ei, ev := 0, 0
		count := 0
		for i, v := range standard_all(s) {
			testify.False(t, i != ei,
				"at iteration %d got index %d, want %d", count, i, ei)
			testify.False(t, v != ev,
				"at iteration %d got value %d, want %d", count, v, ev)
			ei++
			ev++
			count++
		}
		testify.False(t, count != index,
			"read %d values expected %d", count, index)
	}
}

// Test_Standard_Library_Backward ports the Go standard library test.
func Test_Standard_Library_Backward(t *testing.T) {
	for index := 0; index < 10; index++ {
		var s []int
		for i := range index {
			s = append(s, i)
		}
		ei, ev := index-1, index-1
		count := 0
		for i, v := range standard_backward(s) {
			testify.False(t, i != ei,
				"at iteration %d got index %d, want %d", count, i, ei)
			testify.False(t, v != ev,
				"at iteration %d got value %d, want %d", count, v, ev)
			ei--
			ev--
			count++
		}
		testify.False(t, count != index,
			"read %d values expected %d", count, index)
	}
}

// Test_Standard_Library_Values ports the Go standard library test.
func Test_Standard_Library_Values(t *testing.T) {
	for index := 0; index < 10; index++ {
		var s []int
		for i := range index {
			s = append(s, i)
		}
		ev := 0
		count := 0
		for v := range standard_values(s) {
			testify.False(t, v != ev,
				"at iteration %d got %d want %d", count, v, ev)
			ev++
			count++
		}
		testify.False(t, count != index,
			"read %d values expected %d", count, index)
	}
}

func test_sequence(yield func(int) (result_113 bool)) {
	for i := 0; i < 10; i += 2 {
		if !yield(i) {
			return
		}
	}
}

func test_sequence_result() (result []int) {
	return []int{0, 2, 4, 6, 8}
}

// Test_Standard_Library_Append_Sequence ports the Go standard library test.
func Test_Standard_Library_Append_Sequence(t *testing.T) {
	s := standard_append_sequence([]int{1, 2}, test_sequence)
	want := append([]int{1, 2}, test_sequence_result()...)
	testify.True(t, standard_equal(s, want),
		"got %v, want %v", s, want)
}

// Test_Standard_Library_Collect ports the Go standard library test.
func Test_Standard_Library_Collect(t *testing.T) {
	s := standard_collect(test_sequence)
	want := test_sequence_result()
	testify.True(t, standard_equal(s, want),
		"got %v, want %v", s, want)
}

func iter_tests() (result [][]string) {
	return [][]string{
		nil,
		{"a"},
		{"a", "b"},
		{"b", "a"},
		standard_strings(),
	}
}

// Test_Standard_Library_Values_Append_Sequence ports the Go standard library test.
func Test_Standard_Library_Values_Append_Sequence(t *testing.T) {
	for _, prefix := range iter_tests() {
		for _, s := range iter_tests() {
			got := standard_append_sequence(prefix, standard_values(s))
			want := append(prefix, s...)
			testify.True(t, standard_equal(got, want),

				"standard_append_sequence(%v, "+
					"standard_values(%v)) == %v, want %v",
				prefix,
				s,
				got,
				want)
		}
	}
}

// Test_Standard_Library_Values_Collect ports the Go standard library test.
func Test_Standard_Library_Values_Collect(t *testing.T) {
	for _, s := range iter_tests() {
		got := standard_collect(standard_values(s))
		testify.True(t, standard_equal(got, s),
			"standard_collect(standard_values(%v)) == %v, want %v", s, got, s)
	}
}

// Test_Standard_Library_Sorted ports the Go standard library test.
func Test_Standard_Library_Sorted(t *testing.T) {
	integers := standard_integers()
	s := standard_sorted(standard_values(integers))
	testify.True(t, standard_is_sorted(s), "sorted %v; got %v", integers, s)
}

// Test_Standard_Library_Sorted_Func ports the Go standard library test.
func Test_Standard_Library_Sorted_Function(t *testing.T) {
	integers := standard_integers()
	s := standard_sorted_function(
		standard_values(integers),
		func(a, b int) (result_114 int) { return a - b },
	)
	testify.True(t, standard_is_sorted(s), "sorted %v; got %v", integers, s)
}

// Test_Standard_Library_Sorted_Stable_Func ports the Go standard library test.
func Test_Standard_Library_Sorted_Stable_Function(t *testing.T) {
	n, m := 1000, 100
	data := make(int_pairs, n)
	for i := range data {
		data[i].A = random_v2.IntN(m)
	}
	initialize_int_pair_order(data)

	s := int_pairs(standard_sorted_stable_function(standard_values(data), int_pair_cmp))
	testify.True(t, standard_is_sorted_function(s, int_pair_cmp),
		"standard_sorted_stable_function didn't sort %d ints", n)
	testify.True(t, int_pairs_in_order(s, false),
		"standard_sorted_stable_function wasn't stable on %d ints", n)

	// IterVal converts a Seq2 to a Seq.
	iter_value := func(sequence iter.Seq2[int, int_pair]) (result_115 iter.Seq[int_pair]) {
		return func(yield func(int_pair) (result_116 bool)) {
			for _, v := range sequence {
				if !yield(v) {
					return
				}
			}
		}
	}

	s = int_pairs(
		standard_sorted_stable_function(iter_value(standard_backward(data)), int_pair_cmp),
	)
	testify.True(t, standard_is_sorted_function(s, int_pair_cmp),
		"standard_sorted_stable_function didn't sort %d reverse ints", n)
	testify.True(t, int_pairs_in_order(s, true),
		"standard_sorted_stable_function wasn't stable on %d reverse ints", n)
}

// Test_Standard_Library_Chunk ports the Go standard library test.
func Test_Standard_Library_Chunk(t *testing.T) {
	cases := []struct {
		Name   string
		S      []int
		N      int
		Chunks [][]int
	}{
		{
			Name:   "nil",
			S:      nil,
			N:      1,
			Chunks: nil,
		},
		{
			Name:   "empty",
			S:      []int{},
			N:      1,
			Chunks: nil,
		},
		{
			Name:   "short",
			S:      []int{1, 2},
			N:      3,
			Chunks: [][]int{{1, 2}},
		},
		{
			Name:   "one",
			S:      []int{1, 2},
			N:      2,
			Chunks: [][]int{{1, 2}},
		},
		{
			Name:   "even",
			S:      []int{1, 2, 3, 4},
			N:      2,
			Chunks: [][]int{{1, 2}, {3, 4}},
		},
		{
			Name:   "odd",
			S:      []int{1, 2, 3, 4, 5},
			N:      2,
			Chunks: [][]int{{1, 2}, {3, 4}, {5}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var chunks [][]int
			for c := range standard_chunk(tc.S, tc.N) {
				chunks = append(chunks, c)
			}
			testify.True(t, chunk_equal(chunks, tc.Chunks),

				"standard_chunk(%v, %d) = %v, want %v",
				tc.S,
				tc.N,
				chunks,
				tc.Chunks)
			if len(chunks) == 0 {
				return
			}
			// Verify that appending to the end of the first chunk does not
			// clobber the beginning of the next chunk.
			s := standard_clone(tc.S)
			chunks[0] = append(chunks[0], -1)
			testify.True(t, standard_equal(s, tc.S),
				"slice was clobbered: %v, want %v", s, tc.S)
		})
	}
}

// Test_Standard_Library_Chunk_Panics ports the Go standard library test.
func Test_Standard_Library_Chunk_Panics(t *testing.T) {
	for _, test := range []struct {
		Name string
		X    []struct{}
		N    int
	}{
		{
			Name: "cannot be less than 1",
			X:    make([]struct{}, 0),
			N:    0,
		},
	} {
		testify.Panics(t, func() { standard_chunk(test.X, test.N) },
			"standard_chunk %s: got no panic, want panic", test.Name)
	}
}

// Test_Standard_Library_Chunk_Range ports the Go standard library test.
func Test_Standard_Library_Chunk_Range(t *testing.T) {
	// Verify standard_chunk iteration can be stopped.
	var got [][]int
	for c := range standard_chunk([]int{1, 2, 3, 4, -100}, 2) {
		if len(got) == 2 {
			// Found enough values, break early.
			break
		}

		got = append(got, c)
	}

	{

		want := [][]int{{1, 2}, {3, 4}}
		testify.True(t, chunk_equal(got, want),
			"standard_chunk iteration did not stop, got %v, want %v", got, want)
	}
}

func chunk_equal[Slice ~[]E, E comparable](s1, s2 []Slice) (result_117 bool) {
	return standard_equal_function(s1, s2, standard_equal[Slice])
}

// ExampleBinary_Search ports the Go standard library example.
func standard_library_example_binary_search(output *strings.Builder) {
	names := []string{"Alice", "Bob", "Vera"}
	n, found := standard_binary_search(names, "Vera")
	standard_println(output, "Vera:", n, found)
	n, found = standard_binary_search(names, "Bill")
	standard_println(output, "Bill:", n, found)
	// Output:
	// Vera: 2 true
	// Bill: 1 false.
}

// ExampleBinary_Search_Function ports the Go standard library example.
func standard_library_example_binary_search_function(output *strings.Builder) {
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{
		{"Alice", 55},
		{"Bob", 24},
		{"Gopher", 13},
	}
	n, found := standard_binary_search_function(people, Person{
		"Bob",
		0,
	}, func(a, b Person) (result_118 int) {
		return strings.Compare(a.Name, b.Name)
	})
	standard_println(output, "Bob:", n, found)
	// Output:
	// Bob: 1 true.
}

// ExampleCompact ports the Go standard library example.
func standard_library_example_compact(output *strings.Builder) {
	sequence := []int{0, 1, 1, 2, 3, 5, 8}
	sequence = standard_compact(sequence)
	standard_println(output, sequence)
	// Output:
	// [0 1 2 3 5 8].
}

// ExampleCompact_Function ports the Go standard library example.
func standard_library_example_compact_function(output *strings.Builder) {
	names := []string{"bob", "Bob", "alice", "Vera", "VERA"}
	names = standard_compact_function(names, strings.EqualFold)
	standard_println(output, names)
	// Output:
	// [bob alice Vera].
}

// ExampleCompare ports the Go standard library example.
func standard_library_example_compare(output *strings.Builder) {
	names := []string{"Alice", "Bob", "Vera"}
	standard_println(
		output,
		"Equal:",
		standard_compare(names, []string{"Alice", "Bob", "Vera"}),
	)
	standard_println(
		output,
		"V < X:",
		standard_compare(names, []string{"Alice", "Bob", "Xena"}),
	)
	standard_println(
		output,
		"V > C:",
		standard_compare(names, []string{"Alice", "Bob", "Cat"}),
	)
	standard_println(
		output,
		"3 > 2:",
		standard_compare(names, []string{"Alice", "Bob"}),
	)
	// Output:
	// Equal: 0
	// V < X: -1
	// V > C: 1
	// 3 > 2: 1.
}

// ExampleCompare_Function ports the Go standard library example.
func standard_library_example_compare_function(output *strings.Builder) {
	numbers := []int{0, 43, 8}
	strings := []string{"0", "0", "8"}
	result := standard_compare_function(
		numbers,
		strings,
		func(n int, s string) (result_119 int) {
			sn, err := strconv.Atoi(s)
			if err != nil {
				return 1
			}
			return cmp.Compare(n, sn)
		},
	)
	standard_println(output, result)
	// Output:
	// 1.
}

// ExampleContains_Function ports the Go standard library example.
func standard_library_example_contains_function(output *strings.Builder) {
	numbers := []int{0, 42, -10, 8}
	has_negative := standard_contains_function(numbers, func(n int) (result_120 bool) {
		return n < 0
	})
	standard_println(output, "Has a negative:", has_negative)
	has_odd := standard_contains_function(numbers, func(n int) (result_121 bool) {
		return n%2 != 0
	})
	standard_println(output, "Has an odd number:", has_odd)
	// Output:
	// Has a negative: true
	// Has an odd number: false.
}

// ExampleDelete ports the Go standard library example.
func standard_library_example_delete(output *strings.Builder) {
	letters := []string{"a", "b", "c", "d", "e"}
	letters = standard_delete(letters, 1, 4)
	standard_println(output, letters)
	// Output:
	// [a e].
}

// ExampleDelete_Function ports the Go standard library example.
func standard_library_example_delete_function(output *strings.Builder) {
	sequence := []int{0, 1, 1, 2, 3, 5, 8}
	sequence = standard_delete_function(sequence, func(n int) (result_122 bool) {
		return n%2 != 0 // delete the odd numbers
	})
	standard_println(output, sequence)
	// Output:
	// [0 2 8].
}

// ExampleEqual ports the Go standard library example.
func standard_library_example_equal(output *strings.Builder) {
	numbers := []int{0, 42, 8}
	standard_println(output, standard_equal(numbers, []int{0, 42, 8}))
	standard_println(output, standard_equal(numbers, []int{10}))
	// Output:
	// true
	// false.
}

// ExampleEqual_Function ports the Go standard library example.
func standard_library_example_equal_function(output *strings.Builder) {
	numbers := []int{0, 42, 8}
	strings := []string{"000", "42", "0o10"}
	equality_result := standard_equal_function(
		numbers,
		strings,
		func(n int, s string) (result_123 bool) {
			sn, err := strconv.ParseInt(s, 0, 64)
			if err != nil {
				return false
			}
			return n == int(sn)
		},
	)
	standard_println(output, equality_result)
	// Output:
	// true.
}

// ExampleIndex ports the Go standard library example.
func standard_library_example_index(output *strings.Builder) {
	numbers := []int{0, 42, 8}
	standard_println(output, standard_index(numbers, 8))
	standard_println(output, standard_index(numbers, 7))
	// Output:
	// 2
	// -1.
}

// ExampleIndex_Function ports the Go standard library example.
func standard_library_example_index_function(output *strings.Builder) {
	numbers := []int{0, 42, -10, 8}
	i := standard_index_function(numbers, func(n int) (result_124 bool) {
		return n < 0
	})
	standard_println(output, "First negative at index", i)
	// Output:
	// First negative at index 2.
}

// ExampleInsert ports the Go standard library example.
func standard_library_example_insert(output *strings.Builder) {
	names := []string{"Alice", "Bob", "Vera"}
	names = standard_insert(names, 1, "Bill", "Billie")
	names = standard_insert(names, len(names), "Zac")
	standard_println(output, names)
	// Output:
	// [Alice Bill Billie Bob Vera Zac].
}

// ExampleIs_Sorted ports the Go standard library example.
func standard_library_example_is_sorted(output *strings.Builder) {
	standard_println(output, standard_is_sorted([]string{"Alice", "Bob", "Vera"}))
	standard_println(output, standard_is_sorted([]int{0, 2, 1}))
	// Output:
	// true
	// false.
}

// ExampleIs_Sorted_Function ports the Go standard library example.
func standard_library_example_is_sorted_function(output *strings.Builder) {
	names := []string{"alice", "Bob", "VERA"}
	is_sorted_insensitive := standard_is_sorted_function(
		names,
		func(a, b string) (result_125 int) {
			return strings.Compare(strings.ToLower(a), strings.ToLower(b))
		},
	)
	standard_println(output, is_sorted_insensitive)
	standard_println(output, standard_is_sorted(names))
	// Output:
	// true
	// false.
}

// ExampleMaximum ports the Go standard library example.
func standard_library_example_maximum(output *strings.Builder) {
	numbers := []int{0, 42, -10, 8}
	standard_println(output, standard_maximum(numbers))
	// Output:
	// 42.
}

// ExampleMaximum_Function ports the Go standard library example.
func standard_library_example_maximum_function(output *strings.Builder) {
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{
		{"Gopher", 13},
		{"Alice", 55},
		{"Vera", 24},
		{"Bob", 55},
	}
	first_oldest := standard_maximum_function(people, func(a, b Person) (result_126 int) {
		return cmp.Compare(a.Age, b.Age)
	})
	standard_println(output, first_oldest.Name)
	// Output:
	// Alice.
}

// ExampleMinimum ports the Go standard library example.
func standard_library_example_minimum(output *strings.Builder) {
	numbers := []int{0, 42, -10, 8}
	standard_println(output, standard_minimum(numbers))
	// Output:
	// -10.
}

// ExampleMinimum_Function ports the Go standard library example.
func standard_library_example_minimum_function(output *strings.Builder) {
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{
		{"Gopher", 13},
		{"Bob", 5},
		{"Vera", 24},
		{"Bill", 5},
	}
	first_youngest := standard_minimum_function(people, func(a, b Person) (result_127 int) {
		return cmp.Compare(a.Age, b.Age)
	})
	standard_println(output, first_youngest.Name)
	// Output:
	// Bob.
}

// ExampleReplace ports the Go standard library example.
func standard_library_example_replace(output *strings.Builder) {
	names := []string{"Alice", "Bob", "Vera", "Zac"}
	names = standard_replace(names, 1, 3, "Bill", "Billie", "Cat")
	standard_println(output, names)
	// Output:
	// [Alice Bill Billie Cat Zac].
}

// ExampleReverse ports the Go standard library example.
func standard_library_example_reverse(output *strings.Builder) {
	names := []string{"alice", "Bob", "VERA"}
	standard_reverse(names)
	standard_println(output, names)
	// Output:
	// [VERA Bob alice].
}

// ExampleSort ports the Go standard library example.
func standard_library_example_sort(output *strings.Builder) {
	small_ints := []int8{0, 42, -10, 8}
	standard_sort(small_ints)
	standard_println(output, small_ints)
	// Output:
	// [-10 0 8 42].
}

// ExampleSort_Function_case_insensitive ports the Go standard library example.
func standard_library_example_sort_function_case_insensitive(output *strings.Builder) {
	names := []string{"Bob", "alice", "VERA"}
	standard_sort_function(names, func(a, b string) (result_128 int) {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	standard_println(output, names)
	// Output:
	// [alice Bob VERA].
}

// ExampleSort_Function_multi_field ports the Go standard library example.
func standard_library_example_sort_function_multi_field(output *strings.Builder) {
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{
		{"Gopher", 13},
		{"Alice", 55},
		{"Bob", 24},
		{"Alice", 20},
	}
	standard_sort_function(people, func(a, b Person) (result_129 int) {
		if n := strings.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		// If names are equal, order by age.
		return cmp.Compare(a.Age, b.Age)
	})
	standard_println(output, people)
	// Output:
	// [{Alice 20} {Alice 55} {Bob 24} {Gopher 13}].
}

// ExampleSort_Stable_Function ports the Go standard library example.
func standard_library_example_sort_stable_function(output *strings.Builder) {
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{
		{"Gopher", 13},
		{"Alice", 20},
		{"Bob", 24},
		{"Alice", 55},
	}
	// Stable sort by name, keeping age ordering of Alice intact.
	standard_sort_stable_function(people, func(a, b Person) (result_130 int) {
		return strings.Compare(a.Name, b.Name)
	})
	standard_println(output, people)
	// Output:
	// [{Alice 20} {Alice 55} {Bob 24} {Gopher 13}].
}

// ExampleClone ports the Go standard library example.
func standard_library_example_clone(output *strings.Builder) {
	numbers := []int{0, 42, -10, 8}
	clone := standard_clone(numbers)
	standard_println(output, clone)
	clone[2] = 10
	standard_println(output, numbers)
	standard_println(output, clone)
	// Output:
	// [0 42 -10 8]
	// [0 42 -10 8]
	// [0 42 10 8].
}

// ExampleGrow ports the Go standard library example.
func standard_library_example_grow(output *strings.Builder) {
	numbers := []int{0, 42, -10, 8}
	grow := standard_grow(numbers, 2)
	standard_println(output, cap(numbers))
	standard_println(output, grow)
	standard_println(output, len(grow))
	standard_println(output, cap(grow))
	// Output:
	// 4
	// [0 42 -10 8]
	// 4
	// 8.
}

// ExampleClip ports the Go standard library example.
func standard_library_example_clip(output *strings.Builder) {
	a := [...]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	s := a[:4:10]
	clip := standard_clip(s)
	standard_println(output, cap(s))
	standard_println(output, clip)
	standard_println(output, len(clip))
	standard_println(output, cap(clip))
	// Output:
	// 10
	// [0 1 2 3]
	// 4
	// 4.
}

// ExampleConcatenate ports the Go standard library example.
func standard_library_example_concatenate(output *strings.Builder) {
	s1 := []int{0, 1, 2, 3}
	s2 := []int{4, 5, 6}
	concatenation := standard_concatenate(s1, s2)
	standard_println(output, concatenation)
	// Output:
	// [0 1 2 3 4 5 6].
}

// ExampleContains ports the Go standard library example.
func standard_library_example_contains(output *strings.Builder) {
	numbers := []int{0, 1, 2, 3}
	standard_println(output, standard_contains(numbers, 2))
	standard_println(output, standard_contains(numbers, 4))
	// Output:
	// true
	// false.
}

// ExampleRepeat ports the Go standard library example.
func standard_library_example_repeat(output *strings.Builder) {
	numbers := []int{0, 1, 2, 3}
	repeat := standard_repeat(numbers, 2)
	standard_println(output, repeat)
	// Output:
	// [0 1 2 3 0 1 2 3].
}

// ExampleAll ports the Go standard library example.
func standard_library_example_all(output *strings.Builder) {
	names := []string{"Alice", "Bob", "Vera"}
	for i, v := range standard_all(names) {
		standard_println(output, i, ":", v)
	}
	// Output:
	// 0 : Alice
	// 1 : Bob
	// 2 : Vera.
}

// ExampleBackward ports the Go standard library example.
func standard_library_example_backward(output *strings.Builder) {
	names := []string{"Alice", "Bob", "Vera"}
	for i, v := range standard_backward(names) {
		standard_println(output, i, ":", v)
	}
	// Output:
	// 2 : Vera
	// 1 : Bob
	// 0 : Alice.
}

// ExampleValues ports the Go standard library example.
func standard_library_example_values(output *strings.Builder) {
	names := []string{"Alice", "Bob", "Vera"}
	for v := range standard_values(names) {
		standard_println(output, v)
	}
	// Output:
	// Alice
	// Bob
	// Vera.
}

// ExampleAppend_Sequence ports the Go standard library example.
func standard_library_example_append_sequence(output *strings.Builder) {
	sequence := func(yield func(int) (result_131 bool)) {
		for i := 0; i < 10; i += 2 {
			if !yield(i) {
				return
			}
		}
	}

	s := standard_append_sequence([]int{1, 2}, sequence)
	standard_println(output, s)
	// Output:
	// [1 2 0 2 4 6 8].
}

// ExampleCollect ports the Go standard library example.
func standard_library_example_collect(output *strings.Builder) {
	sequence := func(yield func(int) (result_132 bool)) {
		for i := 0; i < 10; i += 2 {
			if !yield(i) {
				return
			}
		}
	}

	s := standard_collect(sequence)
	standard_println(output, s)
	// Output:
	// [0 2 4 6 8].
}

// ExampleSorted ports the Go standard library example.
func standard_library_example_sorted(output *strings.Builder) {
	sequence := func(yield func(int) (result_133 bool)) {
		flag := -1
		for i := 0; i < 10; i += 2 {
			flag = -flag
			if !yield(i * flag) {
				return
			}
		}
	}

	s := standard_sorted(sequence)
	standard_println(output, s)
	standard_println(output, standard_is_sorted(s))
	// Output:
	// [-6 -2 0 4 8]
	// true.
}

// ExampleSorted_Function ports the Go standard library example.
func standard_library_example_sorted_function(output *strings.Builder) {
	sequence := func(yield func(int) (result_134 bool)) {
		flag := -1
		for i := 0; i < 10; i += 2 {
			flag = -flag
			if !yield(i * flag) {
				return
			}
		}
	}

	sort_function := func(a, b int) (result_135 int) {
		return cmp.Compare(b, a) // the comparison is being done in reverse
	}

	s := standard_sorted_function(sequence, sort_function)
	standard_println(output, s)
	// Output:
	// [8 4 0 -2 -6].
}

// ExampleSorted_Stable_Function ports the Go standard library example.
func standard_library_example_sorted_stable_function(output *strings.Builder) {
	type Person struct {
		Name string
		Age  int
	}

	people := []Person{
		{"Gopher", 13},
		{"Alice", 20},
		{"Bob", 5},
		{"Vera", 24},
		{"Zac", 20},
	}

	sort_function := func(x, y Person) (result_136 int) {
		return cmp.Compare(x.Age, y.Age)
	}

	s := standard_sorted_stable_function(standard_values(people), sort_function)
	standard_println(output, s)
	// Output:
	// [{Bob 5} {Gopher 13} {Alice 20} {Zac 20} {Vera 24}].
}

// ExampleChunk ports the Go standard library example.
func standard_library_example_chunk(output *strings.Builder) {
	type Person struct {
		Name string
		Age  int
	}

	type People []Person

	people := People{
		{"Gopher", 13},
		{"Alice", 20},
		{"Bob", 5},
		{"Vera", 24},
		{"Zac", 15},
	}

	// Standard_chunk people into []Person 2 elements at a time.
	for c := range standard_chunk(people, 2) {
		standard_println(output, c)
	}

	// Output:
	// [{Gopher 13} {Alice 20}]
	// [{Bob 5} {Vera 24}]
	// [{Zac 15}].
}

// Benchmark_Standard_Library_Binary_Search_Floats ports the Go standard library benchmark.
func Benchmark_Standard_Library_Binary_Search_Floats(b *testing.B) {
	for _, count := range []int{16, 32, 64, 128, 512, 1024} {
		b.Run(fmt.Sprintf("Size%d", count), func(b *testing.B) {
			floats := standard_make(0.0, count)
			value := 0.0
			for i_index := range floats {
				floats[i_index] = value
				value++
			}
			midpoint := len(floats) / 2
			needle := (floats[midpoint] + floats[midpoint+1]) / 2
			b.ResetTimer()
			for i_index := 0; i_index < b.N; i_index++ {
				standard_binary_search(floats, needle)
			}
		})
	}
}

type my_struct struct {
	A, B, C, D string
	N          int
}

// Benchmark_Standard_Library_Binary_Search_Func_Struct ports the Go standard library benchmark.
func Benchmark_Standard_Library_Binary_Search_Function_Struct(b *testing.B) {
	for _, count := range []int{16, 32, 64, 128, 512, 1024} {
		b.Run(fmt.Sprintf("Size%d", count), func(b *testing.B) {
			structs := make([]*my_struct, count)
			for i := range structs {
				structs[i] = &my_struct{N: i}
			}
			midpoint := len(structs) / 2
			needle_value := structs[midpoint].N + structs[midpoint+1].N
			needle := &my_struct{N: needle_value / 2}
			comparison_function := func(a, b *my_struct) (result_137 int) {
				return a.N - b.N
			}
			b.ResetTimer()
			for i_index := 0; i_index < b.N; i_index++ {
				standard_binary_search_function(
					structs,
					needle,
					comparison_function,
				)
			}
		})
	}
}

// Benchmark_Standard_Library_Sort_Func_Struct ports the Go standard library benchmark.
func Benchmark_Standard_Library_Sort_Function_Struct(b *testing.B) {
	for _, count := range []int{16, 32, 64, 128, 512, 1024} {
		b.Run(fmt.Sprintf("Size%d", count), func(b *testing.B) {
			structs := make([]*my_struct, count)
			for i := range structs {
				structs[i] = &my_struct{
					A: fmt.Sprintf("string%d", i%10),
					N: i * 11 % count,
				}
			}
			comparison_function := func(a, b *my_struct) (result_138 int) {
				if n := strings.Compare(a.A, b.A); n != 0 {
					return n
				}
				return cmp.Compare(a.N, b.N)
			}
			// Presort the slice so all benchmark iterations are identical.
			standard_sort_function(structs, comparison_function)
			b.ResetTimer()
			for i_index := 0; i_index < b.N; i_index++ {
				// Sort twice because the first call modifies the slice in place.
				standard_sort_function(
					structs,
					func(a, b *my_struct) (result_139 int) {
						return comparison_function(
							b,
							a,
						)
					},
				)
				standard_sort_function(structs, comparison_function)
			}
		})
	}
}
