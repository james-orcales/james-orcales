package slices_test

import (
	"cmp"
	"iter"
	"reflect"
	"testing"

	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/testify"
)

// Test_Equality verifies comparable equality, injected cross-type equality, nil equality,
// and the first unequal element.
func Test_Equality(t *testing.T) {
	t.Parallel()
	testify.True(t, bool(slices.Equal([]int{1, 2}, []int{1, 2})),
		"Equal must accept equal slices")
	testify.False(t, bool(slices.Equal([]int{1, 2}, []int{1, 3})),
		"Equal must reject the first unequal pair")
	testify.True(t, bool(slices.Equal([]int(nil), []int{})),
		"Equal must treat nil and empty slices as equal")
	equal_text := func(number int, text string) (equal bool) {
		return number == len(text)
	}
	testify.True(t,
		bool(slices.Equal_Function([]int{1, 2}, []string{"a", "bb"}, equal_text)),
		"Equal_Function must compare different element types")
}

// Test_Ordering verifies lexicographic order, common-prefix order, and the first result
// from an injected comparison function.
func Test_Ordering(t *testing.T) {
	t.Parallel()
	testify.False(
		t,
		slices.Compare([]int{1, 2}, []int{1, 3}) >= 0,
		"Compare must order the first unequal pair",
	)
	testify.False(
		t,
		slices.Compare([]int{1}, []int{1, 0}) >= 0,
		"Compare must put the shorter common prefix first",
	)
	compare_text := func(number int, text string) (comparison slices.Comparison) {
		return slices.Comparison(cmp.Compare(number, len(text)))
	}
	testify.False(
		t,
		slices.Compare_Function([]int{2}, []string{"a"}, compare_text) <= 0,
		"Compare_Function must return the injected order",
	)
}

// Test_Search verifies the first match, the absent sentinel, and the predicate forms.
func Test_Search(t *testing.T) {
	t.Parallel()
	values := []int{4, 7, 7, 9}
	testify.False(t, slices.Index(values, 7) != 1, "Index must return the first matching index")
	testify.False(
		t,
		slices.Index(values, 8) != slices.INDEX_NOT_FOUND,
		"Index must return INDEX_NOT_FOUND for an absent value",
	)
	is_even := func(value int) (selected bool) { return value%2 == 0 }
	testify.False(
		t,
		slices.Index_Function(values, is_even) != 0,
		"Index_Function must return the first predicate match",
	)
	testify.True(t, bool(slices.Contains(values, 9)),
		"Contains must report a present value")
	testify.True(t, bool(slices.Contains_Function(values, is_even)),
		"Contains_Function must report a predicate match")
}

// Test_Edits verifies insertion, deletion, predicate deletion, replacement, tail clearing,
// and an overlapping insertion input.
func Test_Edits(t *testing.T) {
	t.Parallel()
	inserted := slices.Insert([]int{1, 4}, 1, []int{2, 3})
	assert_slice(t, inserted, []int{1, 2, 3, 4})
	deleted := slices.Delete([]int{1, 2, 3, 4}, 1, 3)
	assert_slice(t, deleted, []int{1, 4})
	delete_even := func(value int) (selected bool) { return value%2 == 0 }
	filtered := slices.Delete_Function([]int{1, 2, 3, 4}, delete_even)
	assert_slice(t, filtered, []int{1, 3})
	replaced := slices.Replace([]int{1, 2, 5}, 1, 2, []int{3, 4})
	assert_slice(t, replaced, []int{1, 3, 4, 5})
	overlap := []int{1, 2, 3}
	overlap = append(overlap, 0, 0, 0)
	overlap = slices.Insert(overlap[:3], 1, overlap[:2])
	assert_slice(t, overlap, []int{1, 1, 2, 2, 3})
	tail := []*int{new(int), new(int), new(int)}
	tail = slices.Delete(tail, 1, 3)
	testify.False(t, tail[:cap(tail)][1] != nil, "Delete must clear the obsolete tail")
}

// Test_Copy_And_Capacity verifies shallow cloning, compaction, reserved capacity, and clipping.
func Test_Copy_And_Capacity(t *testing.T) {
	t.Parallel()
	original := []int{1, 2, 3}
	clone := slices.Clone(original)
	clone[0] = 9
	testify.False(t, original[0] != 1, "Clone must use different slice storage")
	assert_slice(t, slices.Compact([]int{1, 1, 2, 2, 2, 3}), []int{1, 2, 3})
	equal_parity := func(left int, right int) (equal bool) { return left%2 == right%2 }
	assert_slice(t, slices.Compact_Function([]int{1, 3, 2, 4, 5}, equal_parity),
		[]int{1, 2, 5})
	grown := slices.Grow([]int{1, 2}, 5)
	testify.False(
		t,
		cap(grown)-len(grown) < 5,
		"Grow must reserve the requested additional capacity",
	)
	clipped := slices.Clip(grown)
	testify.False(t, cap(clipped) != len(clipped), "Clip must remove unused capacity")
}

// Test_Order_Changes verifies reversal, concatenation, repetition, ordered sorting, injected
// sorting, stable sorting, and sorted reports.
func Test_Order_Changes(t *testing.T) {
	t.Parallel()
	values := []int{3, 1, 2}
	slices.Reverse(values)
	assert_slice(t, values, []int{2, 1, 3})
	assert_slice(t, slices.Concatenate([][]int{{1, 2}, nil, {3}}), []int{1, 2, 3})
	assert_slice(t, slices.Repeat([]int{1, 2}, 3), []int{1, 2, 1, 2, 1, 2})
	slices.Sort(values)
	assert_slice(t, values, []int{1, 2, 3})
	reverse_order := func(left int, right int) (comparison slices.Comparison) {
		return slices.Comparison(cmp.Compare(right, left))
	}
	slices.Sort_Function(values, reverse_order)
	assert_slice(t, values, []int{3, 2, 1})
	testify.True(t, bool(slices.Is_Sorted_Function(values, reverse_order)),
		"Is_Sorted_Function must use the injected order")
	testify.False(t, bool(slices.Is_Sorted(values)),
		"Is_Sorted must reject descending values")
	verify_stable_sort(t)
}

// Test_Extrema verifies ordered extremes and the first equal extreme from each function form.
func Test_Extrema(t *testing.T) {
	t.Parallel()
	values := []int{4, 1, 9, 2}
	testify.False(t, slices.Minimum(values) != 1, "Minimum must return the least value")
	testify.False(t, slices.Maximum(values) != 9, "Maximum must return the greatest value")
	type item struct{ Score int }
	items := []item{{Score: 3}, {Score: 1}, {Score: 1}}
	compare_item := func(left item, right item) (comparison slices.Comparison) {
		return slices.Comparison(cmp.Compare(left.Score, right.Score))
	}
	testify.False(
		t,
		slices.Minimum_Function(items, compare_item) != items[1],
		"Minimum_Function must keep the first equal minimum",
	)
	testify.False(
		t,
		slices.Maximum_Function(items, compare_item) != items[0],
		"Maximum_Function must keep the first equal maximum",
	)
}

// Test_Binary_Search verifies the first match, insertion position, and cross-type comparison.
func Test_Binary_Search(t *testing.T) {
	t.Parallel()
	values := []int{1, 2, 2, 4}
	index, found := slices.Binary_Search(values, 2)
	testify.True(t, bool(found), "Binary_Search must report a present target")
	testify.False(t, index != 1, "Binary_Search must return the first equal index")
	index, found = slices.Binary_Search(values, 3)
	testify.False(t, bool(found), "Binary_Search must report an absent target")
	testify.False(t, index != 3, "Binary_Search must return the insertion index")
	compare_text := func(value int, target string) (comparison slices.Comparison) {
		return slices.Comparison(cmp.Compare(value, len(target)))
	}
	index, found = slices.Binary_Search_Function(values, "xx", compare_text)
	testify.True(t, bool(found),
		"Binary_Search_Function must report the cross-type match")
	testify.False(
		t,
		index != 1,
		"Binary_Search_Function must return the cross-type match index",
	)
}

// Test_Iteration verifies forward, backward, and value-only order, including early stop.
func Test_Iteration(t *testing.T) {
	t.Parallel()
	values := []string{"a", "b", "c"}
	forward := make([]string, 0, 3)
	for index, value := range slices.All(values) {
		forward = append(forward, string(rune('0'+index))+value)
	}
	assert_slice(t, forward, []string{"0a", "1b", "2c"})
	backward := make([]string, 0, 2)
	for index, value := range slices.Backward(values) {
		backward = append(backward, string(rune('0'+index))+value)
		if len(backward) == 2 {
			break
		}
	}
	assert_slice(t, backward, []string{"2c", "1b"})
	assert_slice(t, collect_sequence(slices.Values(values)), values)
}

// Test_Collection verifies append, collection, each collected order, and stable collection.
func Test_Collection(t *testing.T) {
	t.Parallel()
	sequence := slices.Values([]int{3, 1, 2})
	assert_slice(t, slices.Append_Sequence([]int{0}, sequence), []int{0, 3, 1, 2})
	assert_slice(t, slices.Collect[[]int](slices.Values([]int{3, 1, 2})), []int{3, 1, 2})
	assert_slice(t, slices.Sorted[[]int](slices.Values([]int{3, 1, 2})), []int{1, 2, 3})
	reverse_order := func(left int, right int) (comparison slices.Comparison) {
		return slices.Comparison(cmp.Compare(right, left))
	}
	assert_slice(t, slices.Sorted_Function[[]int](
		slices.Values([]int{3, 1, 2}), reverse_order), []int{3, 2, 1})
	verify_stable_collection(t)
}

// Test_Chunks verifies full and final chunks, clipped capacity, empty input, and early stop.
func Test_Chunks(t *testing.T) {
	t.Parallel()
	chunks := make([][]int, 0, 3)
	for chunk := range slices.Chunk([]int{1, 2, 3, 4, 5}, 2) {
		testify.False(t, cap(chunk) != len(chunk), "Chunk must clip each chunk capacity")
		chunks = append(chunks, chunk)
	}
	testify.True(t, reflect.DeepEqual(chunks, [][]int{{1, 2}, {3, 4}, {5}}),
		"Chunk result = %v", chunks)
	{

		next, stop := iter.Pull(slices.Chunk([]int{}, 2))
		testify.False(t, func() (has_value bool) {
			defer stop()
			_, ok := next()
			return ok
		}(), "Chunk must not yield for an empty input")
	}
}

// Test_Nil_Preservation verifies each specified nil or nonnil empty result.
func Test_Nil_Preservation(t *testing.T) {
	t.Parallel()
	var nil_values []int
	testify.False(t, slices.Clone(nil_values) != nil, "Clone must preserve nil")
	testify.False(
		t,
		slices.Insert(nil_values, 0, []int{}) != nil,
		"Insert without values must preserve nil",
	)
	testify.False(
		t,
		slices.Collect[[]int](slices.Values(nil_values)) != nil,
		"Collect must return nil for an empty sequence",
	)
	testify.False(
		t,
		slices.Sorted[[]int](slices.Values(nil_values)) != nil,
		"Sorted must return nil for an empty sequence",
	)
	testify.False(
		t,
		slices.Repeat(nil_values, 0) == nil,
		"Repeat must always return a nonnil slice",
	)
}

// Test_Size_Limits verifies the common input limit, result limit, and collection limit.
func Test_Size_Limits(t *testing.T) {
	t.Parallel()
	too_many := make([]struct{}, slices.SLICE_COUNT_MAXIMUM+1)
	testify.False(
		t,
		panic_text(func() { slices.Equal(too_many, too_many) }) == "",
		"Equal must reject an input above SLICE_COUNT_MAXIMUM",
	)
	full := make([]struct{}, slices.SLICE_COUNT_MAXIMUM)
	testify.False(
		t,
		panic_text(func() { slices.Insert(full, 0, []struct{}{{}}) }) == "",
		"Insert must reject a result above SLICE_COUNT_MAXIMUM",
	)
	sequence := func(yield func(struct{}) (continued bool)) {
		for _, value := range too_many {
			if !yield(value) {
				return
			}
		}
	}
	testify.False(
		t,
		panic_text(func() { slices.Collect[[]struct{}](sequence) }) == "",
		"Collect must reject a result above SLICE_COUNT_MAXIMUM",
	)
}

// Test_Domain_Errors verifies each invalid index, count, and empty-extrema panic.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	panic_cases := []func(){
		func() { slices.Insert([]int{}, 1, []int{1}) },
		func() { slices.Delete([]int{1}, 1, 0) },
		func() { slices.Replace([]int{1}, 0, 2, []int{}) },
		func() { slices.Grow([]int{}, -1) },
		func() { slices.Repeat([]int{1}, -1) },
		func() { slices.Minimum([]int{}) },
		func() { slices.Maximum([]int{}) },
		func() { slices.Chunk([]int{}, 0) },
	}
	for panic_index, action := range panic_cases {
		testify.False(t, panic_text(action) == "",
			"domain error %d did not panic", panic_index)
	}
}

// Test_Invariant_Domains verifies each special value through a public slice operation.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	cover_boolean_domains()
	cover_order_domains()
	cover_comparison_domains()
	cover_count_domains()
	cover_position_domains()
	cover_search_domains()
}

// Reaches both reports from every Boolean query boundary.
func cover_boolean_domains() {
	never_equal := func(left int, right int) (equal bool) { return false }
	slices.Equal_Function([]int{1}, []int{1}, never_equal)
	slices.Contains([]int{1}, 2)
	never := func(value int) (selected bool) { return false }
	slices.Contains_Function([]int{1}, never)
	slices.Is_Sorted([]int{1, 2})
	reverse_order := func(left int, right int) (comparison slices.Comparison) {
		return slices.Comparison(cmp.Compare(right, left))
	}
	slices.Is_Sorted_Function([]int{1, 2}, reverse_order)
	compare_target := func(value int, target int) (comparison slices.Comparison) {
		return slices.Comparison(cmp.Compare(value, target))
	}
	slices.Binary_Search_Function([]int{1}, 2, compare_target)
}

// Reaches the three normalized Compare results.
func cover_order_domains() {
	slices.Compare([]int{1}, []int{2})
	slices.Compare([]int{1}, []int{1})
	slices.Compare([]int{2}, []int{1})
}

// Reaches the full-width bounds and the four interior comparison sentinels.
func cover_comparison_domains() {
	comparisons := []slices.Comparison{
		slices.Comparison(slices.COMPARISON_MINIMUM),
		slices.Comparison(slices.COMPARISON_MAXIMUM),
		0,
		1,
		2,
		-1,
	}
	for _, wanted := range comparisons {
		comparison := func(left int, right int) (result slices.Comparison) { return wanted }
		slices.Compare_Function([]int{0}, []int{0}, comparison)
	}
}

// Reaches all Count sentinels without a proportional allocation.
func cover_count_domains() {
	counts := []slices.Count{0, 1, 2, slices.Count(slices.COUNT_MAXIMUM)}
	for _, count := range counts {
		slices.Grow([]struct{}{}, count)
		slices.Repeat([]struct{}{}, count)
		if count != 0 {
			slices.Chunk([]struct{}{}, count)
		}
	}
}

// Reaches all Position sentinels. The largest position reaches the boundary before the
// standard-library bounds check rejects the empty source slice.
func cover_position_domains() {
	positions := []slices.Position{0, 1, 2}
	for _, position := range positions {
		values := make([]struct{}, int(position))
		slices.Insert(values, position, []struct{}{})
		slices.Delete(values, position, position)
		slices.Replace(values, position, position, []struct{}{})
	}
	panic_text(func() {
		slices.Insert([]struct{}{}, slices.Position(slices.POSITION_MAXIMUM), []struct{}{})
	})
	panic_text(func() {
		slices.Delete([]struct{}{}, slices.Position(slices.POSITION_MAXIMUM),
			slices.Position(slices.POSITION_MAXIMUM))
	})
	panic_text(func() {
		slices.Replace([]struct{}{}, slices.Position(slices.POSITION_MAXIMUM),
			slices.Position(slices.POSITION_MAXIMUM), []struct{}{})
	})
}

// Reaches every search-result and ordered-search-position sentinel.
func cover_search_domains() {
	slices.Index([]bool{true}, true)
	slices.Index([]bool{false, false, true}, true)
	last := make([]bool, slices.SLICE_COUNT_MAXIMUM)
	last[len(last)-1] = true
	slices.Index(last, true)
	never := func(value bool) (selected bool) { return false }
	slices.Index_Function([]bool{}, never)
	is_true := func(value bool) (selected bool) { return value }
	slices.Index_Function([]bool{false, true}, is_true)
	slices.Index_Function([]bool{false, false, true}, is_true)
	slices.Index_Function(last, is_true)
	slices.Binary_Search([]int{}, 0)
	slices.Binary_Search([]int{0, 0}, 1)
	ordered := make([]int, slices.SLICE_COUNT_MAXIMUM)
	slices.Binary_Search(ordered, 1)
	comparison := func(value int, target int) (ordering slices.Comparison) {
		return slices.Comparison(cmp.Compare(value, target))
	}
	slices.Binary_Search_Function([]int{}, 0, comparison)
	slices.Binary_Search_Function([]int{0, 0}, 1, comparison)
	slices.Binary_Search_Function(ordered, 1, comparison)
}

// A stable-sort fixture whose Key supplies order and whose Position exposes stability.
type stable_item struct {
	Key      int
	Position int
}

// Verifies that Sort_Stable_Function preserves the positions of equal keys.
func verify_stable_sort(t *testing.T) {
	t.Helper()
	items := []stable_item{{Key: 2, Position: 0}, {Key: 1, Position: 1},
		{Key: 2, Position: 2}, {Key: 1, Position: 3}}
	compare_item := func(left stable_item, right stable_item) (comparison slices.Comparison) {
		return slices.Comparison(cmp.Compare(left.Key, right.Key))
	}
	slices.Sort_Stable_Function(items, compare_item)
	want := []stable_item{{Key: 1, Position: 1}, {Key: 1, Position: 3},
		{Key: 2, Position: 0}, {Key: 2, Position: 2}}
	assert_slice(t, items, want)
}

// Verifies that Sorted_Stable_Function preserves input order among equal keys.
func verify_stable_collection(t *testing.T) {
	t.Helper()
	items := []stable_item{{Key: 2, Position: 0}, {Key: 1, Position: 1},
		{Key: 2, Position: 2}, {Key: 1, Position: 3}}
	compare_item := func(left stable_item, right stable_item) (comparison slices.Comparison) {
		return slices.Comparison(cmp.Compare(left.Key, right.Key))
	}
	got := slices.Sorted_Stable_Function[[]stable_item](slices.Values(items), compare_item)
	want := []stable_item{{Key: 1, Position: 1}, {Key: 1, Position: 3},
		{Key: 2, Position: 0}, {Key: 2, Position: 2}}
	assert_slice(t, got, want)
}

// Collects a sequence into a test-owned slice without testing the package Collect function.
func collect_sequence[E any](sequence iter.Seq[E]) (values []E) {
	for value := range sequence {
		values = append(values, value)
	}
	return values
}

// Fails when two comparable slices differ.
func assert_slice[S ~[]E, E comparable](t *testing.T, got S, want S) {
	t.Helper()
	testify.True(t, reflect.DeepEqual(got, want),
		"slice = %v, want %v", got, want)
}

// Runs an action and reports whether it caused a panic.
func panic_text(action func()) (message string) {
	defer func() {
		raised := recover()
		if raised != nil {
			message = "panicked"
		}
	}()
	action()
	return ""
}
