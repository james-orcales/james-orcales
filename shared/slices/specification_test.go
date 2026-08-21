package slices_test

import (
	"cmp"
	"reflect"
	"testing"

	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/testify"
)

// Test_Equality protects nil equivalence and cross-type comparison.
func Test_Equality(t *testing.T) {
	if !slices.Equal([]int{1, 2}, []int{1, 2}) {
		t.Fatal("Equal rejected equal slices")
	}
	if slices.Equal([]int{1, 2}, []int{1, 3}) {
		t.Fatal("Equal accepted unequal slices")
	}
	if !slices.Equal([]int(nil), []int{}) {
		t.Fatal("Equal distinguished nil from empty")
	}
	if !slices.Equal_Function(
		[]int{1, 2}, []string{"a", "bb"}, equal_text_size,
	) {
		t.Fatal("Equal_Function rejected cross-type equality")
	}
}

// Test_Ordering protects normalized lexical order and injected comparison magnitude.
func Test_Ordering(t *testing.T) {
	if slices.Compare([]int{1, 2}, []int{1, 3}) >= 0 {
		t.Fatal("Compare missed first unequal element")
	}
	if slices.Compare([]int{1}, []int{1, 0}) >= 0 {
		t.Fatal("Compare missed shorter common prefix")
	}
	if slices.Compare_Function(
		[]int{2}, []string{"a"}, compare_text_size,
	) <= 0 {
		t.Fatal("Compare_Function changed injected order")
	}
}

// Test_Search protects first-match and absence semantics.
func Test_Search(t *testing.T) {
	values := []int{4, 7, 7, 9}
	if slices.Index(values, 7) != 1 {
		t.Fatal("Index missed first match")
	}
	if slices.Index(values, 8) != slices.INDEX_NOT_FOUND {
		t.Fatal("Index missed absence")
	}
	if slices.Index_Function(values, is_even) != 0 {
		t.Fatal("Index_Function missed first match")
	}
	if !slices.Contains(values, 9) {
		t.Fatal("Contains missed value")
	}
	if !slices.Contains_Function(values, is_even) {
		t.Fatal("Contains_Function missed value")
	}
}

// Test_Edits protects caller ownership, in-place overlap, and pointer clearing.
func Test_Edits(t *testing.T) {
	var destination [TEST_DESTINATION_COUNT]int
	count := slices.Insert_Into(destination[:], []int{1, 4}, 1, []int{2, 3})
	assert_slice(t, destination[:int(count)], []int{1, 2, 3, 4})
	count = slices.Delete_Into(destination[:], []int{1, 2, 3, 4}, 1, 3)
	assert_slice(t, destination[:int(count)], []int{1, 4})
	count = slices.Delete_Function_Into(destination[:], []int{1, 2, 3, 4}, is_even)
	assert_slice(t, destination[:int(count)], []int{1, 3})
	count = slices.Replace_Into(
		destination[:], []int{1, 2, 5}, 1, 2, []int{3, 4},
	)
	assert_slice(t, destination[:int(count)], []int{1, 3, 4, 5})
	verify_overlapping_edits(t)
	verify_delete_clear(t)
}

// Test_Copy_And_Capacity protects copied ownership and explicit reserve checks.
func Test_Copy_And_Capacity(t *testing.T) {
	var destination [TEST_DESTINATION_COUNT]int
	original := []int{1, 2, 3}
	count := slices.Clone_Into(destination[:], original)
	destination[0] = 9
	if original[0] != 1 {
		t.Fatal("Clone_Into aliased separate storage")
	}
	if count != 3 {
		t.Fatal("Clone_Into returned wrong count")
	}
	count = slices.Compact_Into(destination[:], []int{1, 1, 2, 2, 2, 3})
	assert_slice(t, destination[:int(count)], []int{1, 2, 3})
	count = slices.Compact_Function_Into(
		destination[:], []int{1, 3, 2, 4, 5}, equal_parity,
	)
	assert_slice(t, destination[:int(count)], []int{1, 2, 5})
	count = slices.Grow_Into(destination[:], original, 5)
	if count != 3 {
		t.Fatal("Grow_Into changed source count")
	}
	assert_slice(t, destination[:int(count)], original)
	clipped := slices.Clip(destination[:3])
	if cap(clipped) != len(clipped) {
		t.Fatal("Clip retained unused capacity")
	}
}

// Test_Order_Changes protects in-place order and stable equal-element order.
func Test_Order_Changes(t *testing.T) {
	values := []int{3, 1, 2}
	slices.Reverse(values)
	assert_slice(t, values, []int{2, 1, 3})
	var destination [TEST_DESTINATION_COUNT]int
	parts := [][]int{{1, 2}, nil, {3}}
	count := slices.Concatenate_Into(destination[:], parts)
	assert_slice(t, destination[:int(count)], []int{1, 2, 3})
	var zero_destination [TEST_SINGLE_COUNT]struct{}
	var zero_source [TEST_SINGLE_COUNT]struct{}
	zero_parts := [TEST_SINGLE_COUNT][]struct{}{zero_source[:]}
	if slices.Concatenate_Into(zero_destination[:], zero_parts[:]) != 1 {
		t.Fatal("Concatenate_Into rejected separate zero-size storage")
	}
	count = slices.Repeat_Into(destination[:], []int{1, 2}, 3)
	assert_slice(t, destination[:int(count)], []int{1, 2, 1, 2, 1, 2})
	slices.Sort(values)
	assert_slice(t, values, []int{1, 2, 3})
	slices.Sort_Function(values, reverse_order)
	assert_slice(t, values, []int{3, 2, 1})
	if !slices.Is_Sorted_Function(values, reverse_order) {
		t.Fatal("Is_Sorted_Function rejected injected order")
	}
	if slices.Is_Sorted(values) {
		t.Fatal("Is_Sorted accepted descending values")
	}
	verify_stable_sort(t)
}

// Test_Extrema protects first equal extreme selection.
func Test_Extrema(t *testing.T) {
	values := []int{4, 1, 9, 2}
	if slices.Minimum(values) != 1 {
		t.Fatal("Minimum returned wrong value")
	}
	if slices.Maximum(values) != 9 {
		t.Fatal("Maximum returned wrong value")
	}
	items := []stable_item{{Key: 3}, {Key: 1, Position: 1}, {Key: 1, Position: 2}}
	if slices.Minimum_Function(items, compare_stable_item) != items[1] {
		t.Fatal("Minimum_Function replaced first equal minimum")
	}
	if slices.Maximum_Function(items, compare_stable_item) != items[0] {
		t.Fatal("Maximum_Function replaced first equal maximum")
	}
}

// Test_Binary_Search protects first match and insertion-position results.
func Test_Binary_Search(t *testing.T) {
	values := []int{1, 2, 2, 4}
	position, found := slices.Binary_Search(values, 2)
	if !found {
		t.Fatal("Binary_Search missed first match")
	}
	if position != 1 {
		t.Fatal("Binary_Search returned wrong first match")
	}
	position, found = slices.Binary_Search(values, 3)
	if found {
		t.Fatal("Binary_Search missed insertion position")
	}
	if position != 3 {
		t.Fatal("Binary_Search returned wrong insertion position")
	}
	position, found = slices.Binary_Search_Function(values, "xx", compare_text_size)
	if !found {
		t.Fatal("Binary_Search_Function missed cross-type match")
	}
	if position != 1 {
		t.Fatal("Binary_Search_Function returned wrong cross-type match")
	}
}

// Test_Iteration protects synchronous order, counts, and early stop.
func Test_Iteration(t *testing.T) {
	values := []string{"a", "b", "c"}
	var forward [TEST_TRIPLE_COUNT]string
	forward_count := 0
	count := slices.All(values, func(
		position slices.Search_Position, value string,
	) (continued slices.Boolean) {
		forward[forward_count] = string(rune('0'+position)) + value
		forward_count++
		return true
	})
	if count != 3 {
		t.Fatal("All returned wrong count")
	}
	assert_slice(t, forward[:], []string{"0a", "1b", "2c"})
	var backward [TEST_PAIR_COUNT]string
	backward_count := 0
	count = slices.Backward(values, func(
		position slices.Search_Position, value string,
	) (continued slices.Boolean) {
		backward[backward_count] = string(rune('0'+position)) + value
		backward_count++
		return backward_count < len(backward)
	})
	if count != 2 {
		t.Fatal("Backward ignored early stop")
	}
	assert_slice(t, backward[:], []string{"2c", "1b"})
	var collected [TEST_TRIPLE_COUNT]string
	value_count := 0
	count = slices.Values(values, func(value string) (continued slices.Boolean) {
		collected[value_count] = value
		value_count++
		return true
	})
	if count != 3 {
		t.Fatal("Values returned wrong count")
	}
	assert_slice(t, collected[:], values)
}

// Test_Collection protects caller-owned copy and ordering results.
func Test_Collection(t *testing.T) {
	var destination [TEST_DESTINATION_COUNT]int
	count := slices.Append_Sequence_Into(
		destination[:], []int{0}, []int{3, 1, 2},
	)
	assert_slice(t, destination[:int(count)], []int{0, 3, 1, 2})
	count = slices.Collect_Into(destination[:], []int{3, 1, 2})
	assert_slice(t, destination[:int(count)], []int{3, 1, 2})
	count = slices.Sorted_Into(destination[:], []int{3, 1, 2})
	assert_slice(t, destination[:int(count)], []int{1, 2, 3})
	count = slices.Sorted_Function_Into(
		destination[:], []int{3, 1, 2}, reverse_order,
	)
	assert_slice(t, destination[:int(count)], []int{3, 2, 1})
	verify_stable_collection(t)
}

// Test_Chunks protects clipped source views and empty traversal.
func Test_Chunks(t *testing.T) {
	var chunks [TEST_TRIPLE_COUNT][]int
	chunk_index := 0
	count := slices.Chunk(
		[]int{1, 2, 3, 4, 5}, 2,
		func(chunk []int) (continued slices.Boolean) {
			if cap(chunk) != len(chunk) {
				t.Fatal("Chunk retained unused capacity")
			}
			chunks[chunk_index] = chunk
			chunk_index++
			return true
		},
	)
	if count != 3 {
		t.Fatal("Chunk returned wrong count")
	}
	if !reflect.DeepEqual(chunks[:], [][]int{{1, 2}, {3, 4}, {5}}) {
		t.Fatal("Chunk returned wrong views")
	}
	count = slices.Chunk(
		[]int{}, 2,
		func(_ []int) (continued slices.Boolean) {
			t.Fatal("Chunk yielded empty input")
			return true
		},
	)
	if count != 0 {
		t.Fatal("Chunk counted empty input")
	}
}

// Test_Nil_Preservation protects nil input views without owned results.
func Test_Nil_Preservation(t *testing.T) {
	var nil_values []int
	if slices.Clip(nil_values) != nil {
		t.Fatal("Clip changed nil view")
	}
	if slices.Clone_Into(nil_values, nil_values) != 0 {
		t.Fatal("Clone_Into changed empty count")
	}
	if slices.Repeat_Into(nil_values, nil_values, 0) != 0 {
		t.Fatal("Repeat_Into changed empty count")
	}
}

// Test_Allocation proves each exported operation owns no heap storage.
func Test_Allocation(t *testing.T) {
	verify_api_is_zero_allocation(t)
}

// Test_Size_Limits protects package boundary against oversized malicious input.
func Test_Size_Limits(t *testing.T) {
	var too_many [slices.SLICE_COUNT_MAXIMUM + 1]struct{}
	if panic_text(func() { slices.Equal(too_many[:], too_many[:]) }) == "" {
		t.Fatal("Equal accepted oversized input")
	}
	var full [slices.SLICE_COUNT_MAXIMUM]struct{}
	if panic_text(func() {
		slices.Insert_Into(full[:], full[:], 0, []struct{}{{}})
	}) == "" {
		t.Fatal("Insert_Into accepted oversized result")
	}
	var short [TEST_SINGLE_COUNT]int
	if panic_text(func() {
		slices.Clone_Into(short[:], []int{1, 2})
	}) == "" {
		t.Fatal("Clone_Into accepted short destination")
	}
}

// Test_Domain_Errors protects bounded failure for invalid positions and counts.
func Test_Domain_Errors(t *testing.T) {
	var storage [TEST_QUADRUPLE_COUNT]int
	panic_cases := [...]func(){
		func() { slices.Insert_Into(storage[:], []int{}, 1, []int{1}) },
		func() { slices.Delete_Into(storage[:], []int{1}, 1, 0) },
		func() { slices.Replace_Into(storage[:], []int{1}, 0, 2, []int{}) },
		func() { slices.Grow_Into(storage[:], []int{}, -1) },
		func() { slices.Repeat_Into(storage[:], []int{1}, -1) },
		func() { slices.Minimum([]int{}) },
		func() { slices.Maximum([]int{}) },
		func() { slices.Chunk([]int{}, 0, continue_chunk[int]) },
	}
	for index, action := range panic_cases {
		if panic_text(action) == "" {
			t.Fatalf("domain error %d did not panic", index)
		}
	}
}

// Test_Invariant_Domains protects every declared typed sentinel.
func Test_Invariant_Domains(t *testing.T) {
	cover_boolean_domains()
	cover_order_domains()
	cover_comparison_domains()
	cover_count_domains()
	cover_position_domains()
	cover_search_domains()
}

const TEST_DESTINATION_COUNT = 8

const TEST_QUADRUPLE_COUNT = 4

const TEST_TRIPLE_COUNT = 3

const TEST_PAIR_COUNT = 2

const TEST_SINGLE_COUNT = 1

func cover_boolean_domains() {
	slices.Equal_Function([]int{1}, []int{1}, never_equal)
	slices.Contains([]int{1}, 2)
	slices.Contains_Function([]int{1}, never_select)
	slices.Is_Sorted([]int{1, 2})
	slices.Is_Sorted_Function([]int{1, 2}, reverse_order)
	slices.Binary_Search_Function([]int{1}, 2, compare_int)
	slices.All([]int{1, 2}, stop_indexed_yield)
	slices.Backward([]int{1, 2}, stop_indexed_yield)
	slices.Values([]int{1, 2}, stop_yield)
	slices.Chunk([]int{1, 2}, 1, stop_chunk[int])
}

func cover_order_domains() {
	slices.Compare([]int{1}, []int{2})
	slices.Compare([]int{1}, []int{1})
	slices.Compare([]int{2}, []int{1})
}

func cover_comparison_domains() {
	comparisons := [...]slices.Comparison{
		slices.Comparison(slices.COMPARISON_MINIMUM),
		slices.Comparison(slices.COMPARISON_MAXIMUM),
		0,
		1,
		2,
		-1,
	}
	for _, wanted := range comparisons {
		comparison := func(_, _ int) (result slices.Comparison) {
			return wanted
		}
		slices.Compare_Function([]int{0}, []int{0}, comparison)
	}
}

func cover_count_domains() {
	var destination [slices.SLICE_COUNT_MAXIMUM]struct{}
	var one [TEST_SINGLE_COUNT]struct{}
	counts := [...]slices.Count{0, 1, 2, slices.Count(slices.COUNT_MAXIMUM)}
	for _, count := range counts {
		slices.Grow_Into(destination[:], destination[:0], count)
		slices.Repeat_Into(destination[:], one[:], count)
		if count != 0 {
			slices.Chunk(one[:], count, continue_chunk[struct{}])
		}
	}
	slices.Clone_Into(destination[:], destination[:])
	slices.All(destination[:], continue_indexed_yield[struct{}])
	slices.Backward(destination[:], continue_indexed_yield[struct{}])
	slices.Values(destination[:], continue_yield[struct{}])
	slices.All(destination[:0], continue_indexed_yield[struct{}])
	slices.All(destination[:2], continue_indexed_yield[struct{}])
	slices.Backward(destination[:0], continue_indexed_yield[struct{}])
	slices.Values(destination[:0], continue_yield[struct{}])
	slices.Values(destination[:2], continue_yield[struct{}])
	slices.Chunk(destination[:], 1, continue_chunk[struct{}])
	var source_values [slices.SLICE_COUNT_MAXIMUM]bool
	var parts [TEST_SINGLE_COUNT][]bool
	parts[0] = source_values[:]
	var separate [slices.SLICE_COUNT_MAXIMUM]bool
	slices.Concatenate_Into(separate[:], parts[:])
	exercise_output_count_domains()
}

func exercise_output_count_domains() {
	counts := [...]slices.Count{0, 1, 2, slices.Count(slices.COUNT_MAXIMUM)}
	var storage [slices.SLICE_COUNT_MAXIMUM]int
	var separate [slices.SLICE_COUNT_MAXIMUM]int
	var parts [TEST_SINGLE_COUNT][]int
	for _, count := range counts {
		source := storage[:int(count)]
		slices.Append_Sequence_Into(storage[:], source, source[:0])
		slices.Clone_Into(storage[:], source)
		slices.Collect_Into(storage[:], source)
		slices.Grow_Into(storage[:], source, 0)
		slices.Sorted_Into(storage[:], source)
		slices.Sorted_Function_Into(storage[:], source, compare_int)
		slices.Sorted_Stable_Function_Into(storage[:], source, compare_int)
		for index := range source {
			storage[index] = index % 2
		}
		slices.Compact_Into(storage[:], source)
		for index := range source {
			storage[index] = index % 2
		}
		slices.Compact_Function_Into(storage[:], source, never_equal)
		slices.Delete_Function_Into(storage[:], source, never_select)
		parts[0] = source
		slices.Concatenate_Into(separate[:], parts[:])
	}
	slices.Delete_Function_Into(storage[:1], storage[:1], always_select)
}

func cover_position_domains() {
	positions := [...]slices.Position{0, 1, 2}
	var destination [TEST_PAIR_COUNT]struct{}
	for _, position := range positions {
		source := destination[:int(position)]
		slices.Insert_Into(destination[:], source, position, []struct{}{})
		slices.Delete_Into(destination[:], source, position, position)
		slices.Replace_Into(destination[:], source, position, position, []struct{}{})
	}
	var maximum [slices.SLICE_COUNT_MAXIMUM]struct{}
	maximum_position := slices.Position(slices.POSITION_MAXIMUM)
	slices.Insert_Into(maximum[:], maximum[:], maximum_position, []struct{}{})
	slices.Delete_Into(maximum[:], maximum[:], maximum_position, maximum_position)
	slices.Replace_Into(
		maximum[:], maximum[:], maximum_position, maximum_position, []struct{}{},
	)
}

func cover_search_domains() {
	slices.Index([]bool{true}, true)
	slices.Index([]bool{false, false, true}, true)
	var last [slices.SLICE_COUNT_MAXIMUM]bool
	last[len(last)-1] = true
	slices.Index(last[:], true)
	slices.Index_Function([]bool{}, never_select_boolean)
	slices.Index_Function([]bool{false, true}, select_true)
	slices.Index_Function([]bool{false, false, true}, select_true)
	slices.Index_Function(last[:], select_true)
	slices.Binary_Search([]int{}, 0)
	slices.Binary_Search([]int{0, 0}, 1)
	var ordered [slices.SLICE_COUNT_MAXIMUM]int
	slices.Binary_Search(ordered[:], 1)
	slices.Binary_Search_Function([]int{}, 0, compare_int)
	slices.Binary_Search_Function([]int{0, 0}, 1, compare_int)
	slices.Binary_Search_Function(ordered[:], 1, compare_int)
}

func verify_overlapping_edits(t *testing.T) {
	storage := [TEST_DESTINATION_COUNT]int{1, 2, 3}
	source := storage[:3]
	count := slices.Insert_Into(storage[:], source, 1, source[:2])
	assert_slice(t, storage[:int(count)], []int{1, 1, 2, 2, 3})
	storage = [TEST_DESTINATION_COUNT]int{1, 2, 3}
	source = storage[:3]
	count = slices.Insert_Into(storage[:], source, 0, source[1:])
	assert_slice(t, storage[:int(count)], []int{2, 3, 1, 2, 3})
	storage = [TEST_DESTINATION_COUNT]int{1, 2, 3, 4}
	source = storage[:4]
	count = slices.Replace_Into(storage[:], source, 1, 2, source[:3])
	assert_slice(t, storage[:int(count)], []int{1, 1, 2, 3, 3, 4})
}

func verify_delete_clear(t *testing.T) {
	first := 1
	second := 2
	third := 3
	storage := [TEST_TRIPLE_COUNT]*int{&first, &second, &third}
	count := slices.Delete_Into(storage[:], storage[:], 1, 3)
	if count != 1 {
		t.Fatal("Delete_Into returned wrong count")
	}
	if storage[1] != nil {
		t.Fatal("Delete_Into retained first obsolete pointer")
	}
	if storage[2] != nil {
		t.Fatal("Delete_Into retained obsolete pointers")
	}
}

type stable_item struct {
	Key      int
	Position int
}

func verify_stable_sort(t *testing.T) {
	items := []stable_item{{Key: 2, Position: 0}, {Key: 1, Position: 1},
		{Key: 2, Position: 2}, {Key: 1, Position: 3}}
	slices.Sort_Stable_Function(items, compare_stable_item)
	want := []stable_item{{Key: 1, Position: 1}, {Key: 1, Position: 3},
		{Key: 2, Position: 0}, {Key: 2, Position: 2}}
	assert_slice(t, items, want)
}

func verify_stable_collection(t *testing.T) {
	source := []stable_item{{Key: 2, Position: 0}, {Key: 1, Position: 1},
		{Key: 2, Position: 2}, {Key: 1, Position: 3}}
	var destination [TEST_QUADRUPLE_COUNT]stable_item
	count := slices.Sorted_Stable_Function_Into(
		destination[:], source, compare_stable_item,
	)
	want := []stable_item{{Key: 1, Position: 1}, {Key: 1, Position: 3},
		{Key: 2, Position: 0}, {Key: 2, Position: 2}}
	assert_slice(t, destination[:int(count)], want)
}

func equal_text_size(number int, text string) (equal bool) {
	return number == len(text)
}

func compare_text_size(number int, text string) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(number, len(text)))
}

func compare_int(left int, right int) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(left, right))
}

func reverse_order(left int, right int) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(right, left))
}

func compare_stable_item(
	left stable_item, right stable_item,
) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(left.Key, right.Key))
}

func is_even(value int) (selected bool) {
	return value%2 == 0
}

func equal_parity(left int, right int) (equal bool) {
	return left%2 == right%2
}

func never_equal(_, _ int) (equal bool) {
	return false
}

func never_select(_ int) (selected bool) {
	return false
}

func always_select(_ int) (selected bool) {
	return true
}

func never_select_boolean(_ bool) (selected bool) {
	return false
}

func select_true(value bool) (selected bool) {
	return value
}

func stop_indexed_yield(
	_ slices.Search_Position, _ int,
) (continued slices.Boolean) {
	return false
}

func stop_yield(_ int) (continued slices.Boolean) {
	return false
}

func stop_chunk[Element any](_ []Element) (continued slices.Boolean) {
	return false
}

func continue_indexed_yield[Element any](
	_ slices.Search_Position, _ Element,
) (continued slices.Boolean) {
	return true
}

func continue_yield[Element any](_ Element) (continued slices.Boolean) {
	return true
}

func continue_chunk[Element any](_ []Element) (continued slices.Boolean) {
	return true
}

func assert_slice[Slice ~[]Element, Element comparable](
	t *testing.T, got Slice, want Slice,
) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice = %v, want %v", got, want)
	}
}

func panic_text(action func()) (message string) {
	defer func() {
		if recover() != nil {
			message = "panicked"
		}
	}()
	action()
	return ""
}

type zero_allocation_check struct {
	Name string
	Call func()
}

type zero_allocation_fixture struct {
	Destination [TEST_DESTINATION_COUNT]int
	Work        [TEST_QUADRUPLE_COUNT]int
	Source      [TEST_TRIPLE_COUNT]int
	Values      [TEST_PAIR_COUNT]int
	Parts       [TEST_PAIR_COUNT][]int
	Observable  int
}

func verify_api_is_zero_allocation(t *testing.T) {
	fixture := zero_allocation_fixture{
		Source: [TEST_TRIPLE_COUNT]int{3, 1, 2},
		Values: [TEST_PAIR_COUNT]int{7, 8},
	}
	fixture.Parts[0] = fixture.Source[:1]
	fixture.Parts[1] = fixture.Source[1:]
	groups := [][]zero_allocation_check{
		comparison_search_allocation_checks(&fixture),
		edit_copy_allocation_checks(&fixture),
		order_allocation_checks(&fixture),
		extrema_binary_allocation_checks(&fixture),
		iteration_collection_allocation_checks(&fixture),
	}
	for _, group := range groups {
		for _, check := range group {
			t.Run(check.Name, func(t *testing.T) {
				testify.Zero_Allocation(t, check.Call)
			})
		}
	}
	if fixture.Observable == -1 {
		t.Fatal("operations produced impossible observation")
	}
}

func comparison_search_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Equal", Call: func() {
			fixture.Observable = zero_allocation_boolean(
				slices.Equal(fixture.Source[:], fixture.Source[:]),
			)
		}},
		{Name: "Equal_Function", Call: func() {
			fixture.Observable = zero_allocation_boolean(slices.Equal_Function(
				fixture.Source[:], fixture.Source[:], zero_allocation_equal,
			))
		}},
		{Name: "Compare", Call: func() {
			fixture.Observable = int(
				slices.Compare(fixture.Source[:], fixture.Source[:]),
			)
		}},
		{Name: "Compare_Function", Call: func() {
			fixture.Observable = int(slices.Compare_Function(
				fixture.Source[:], fixture.Source[:], zero_allocation_compare,
			))
		}},
		{Name: "Index", Call: func() {
			fixture.Observable = int(slices.Index(fixture.Source[:], 2))
		}},
		{Name: "Index_Function", Call: func() {
			fixture.Observable = int(
				slices.Index_Function(fixture.Source[:], zero_allocation_even),
			)
		}},
		{Name: "Contains", Call: func() {
			fixture.Observable = zero_allocation_boolean(
				slices.Contains(fixture.Source[:], 2),
			)
		}},
		{Name: "Contains_Function", Call: func() {
			fixture.Observable = zero_allocation_boolean(slices.Contains_Function(
				fixture.Source[:], zero_allocation_even,
			))
		}},
	}
}

func edit_copy_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Insert_Into", Call: func() {
			fixture.Observable = int(slices.Insert_Into(
				fixture.Destination[:], fixture.Source[:], 1, fixture.Values[:],
			))
		}},
		{Name: "Delete_Into", Call: func() {
			fixture.Observable = int(slices.Delete_Into(
				fixture.Destination[:], fixture.Source[:], 1, 2,
			))
		}},
		{Name: "Delete_Function_Into", Call: func() {
			fixture.Observable = int(slices.Delete_Function_Into(
				fixture.Destination[:], fixture.Source[:], zero_allocation_even,
			))
		}},
		{Name: "Replace_Into", Call: func() {
			fixture.Observable = int(slices.Replace_Into(
				fixture.Destination[:], fixture.Source[:], 1, 2, fixture.Values[:],
			))
		}},
		{Name: "Clone_Into", Call: func() {
			fixture.Observable = int(
				slices.Clone_Into(fixture.Destination[:], fixture.Source[:]),
			)
		}},
		{Name: "Compact_Into", Call: func() {
			fixture.Observable = int(
				slices.Compact_Into(fixture.Destination[:], fixture.Source[:]),
			)
		}},
		{Name: "Compact_Function_Into", Call: func() {
			fixture.Observable = int(slices.Compact_Function_Into(
				fixture.Destination[:], fixture.Source[:], zero_allocation_equal,
			))
		}},
		{Name: "Grow_Into", Call: func() {
			fixture.Observable = int(
				slices.Grow_Into(fixture.Destination[:], fixture.Source[:], 2),
			)
		}},
		{Name: "Clip", Call: func() {
			fixture.Observable = cap(slices.Clip(fixture.Source[:]))
		}},
	}
}

func order_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Reverse", Call: func() {
			copy(fixture.Work[:], fixture.Source[:])
			slices.Reverse(fixture.Work[:len(fixture.Source)])
			fixture.Observable = fixture.Work[0]
		}},
		{Name: "Concatenate_Into", Call: func() {
			fixture.Observable = int(
				slices.Concatenate_Into(fixture.Destination[:], fixture.Parts[:]),
			)
		}},
		{Name: "Repeat_Into", Call: func() {
			fixture.Observable = int(
				slices.Repeat_Into(fixture.Destination[:], fixture.Source[:], 2),
			)
		}},
		{Name: "Sort", Call: func() {
			copy(fixture.Work[:], fixture.Source[:])
			slices.Sort(fixture.Work[:len(fixture.Source)])
			fixture.Observable = fixture.Work[0]
		}},
		{Name: "Sort_Function", Call: func() {
			copy(fixture.Work[:], fixture.Source[:])
			slices.Sort_Function(
				fixture.Work[:len(fixture.Source)], zero_allocation_compare,
			)
			fixture.Observable = fixture.Work[0]
		}},
		{Name: "Sort_Stable_Function", Call: func() {
			copy(fixture.Work[:], fixture.Source[:])
			slices.Sort_Stable_Function(
				fixture.Work[:len(fixture.Source)], zero_allocation_compare,
			)
			fixture.Observable = fixture.Work[0]
		}},
		{Name: "Is_Sorted", Call: func() {
			fixture.Observable = zero_allocation_boolean(
				slices.Is_Sorted(fixture.Source[:]),
			)
		}},
		{Name: "Is_Sorted_Function", Call: func() {
			fixture.Observable = zero_allocation_boolean(slices.Is_Sorted_Function(
				fixture.Source[:], zero_allocation_compare,
			))
		}},
	}
}

func extrema_binary_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "Minimum", Call: func() {
			fixture.Observable = slices.Minimum(fixture.Source[:])
		}},
		{Name: "Minimum_Function", Call: func() {
			fixture.Observable = slices.Minimum_Function(
				fixture.Source[:], zero_allocation_compare,
			)
		}},
		{Name: "Maximum", Call: func() {
			fixture.Observable = slices.Maximum(fixture.Source[:])
		}},
		{Name: "Maximum_Function", Call: func() {
			fixture.Observable = slices.Maximum_Function(
				fixture.Source[:], zero_allocation_compare,
			)
		}},
		{Name: "Binary_Search", Call: func() {
			position, found := slices.Binary_Search(fixture.Values[:], 8)
			fixture.Observable = int(position) + zero_allocation_boolean(found)
		}},
		{Name: "Binary_Search_Function", Call: func() {
			position, found := slices.Binary_Search_Function(
				fixture.Values[:], 8, zero_allocation_compare,
			)
			fixture.Observable = int(position) + zero_allocation_boolean(found)
		}},
	}
}

func iteration_collection_allocation_checks(
	fixture *zero_allocation_fixture,
) (checks []zero_allocation_check) {
	return []zero_allocation_check{
		{Name: "All", Call: func() {
			fixture.Observable = int(
				slices.All(fixture.Source[:], zero_allocation_indexed_yield),
			)
		}},
		{Name: "Backward", Call: func() {
			fixture.Observable = int(
				slices.Backward(fixture.Source[:], zero_allocation_indexed_yield),
			)
		}},
		{Name: "Values", Call: func() {
			fixture.Observable = int(
				slices.Values(fixture.Source[:], zero_allocation_yield),
			)
		}},
		{Name: "Append_Sequence_Into", Call: func() {
			fixture.Observable = int(slices.Append_Sequence_Into(
				fixture.Destination[:], fixture.Values[:1], fixture.Source[:],
			))
		}},
		{Name: "Collect_Into", Call: func() {
			fixture.Observable = int(
				slices.Collect_Into(fixture.Destination[:], fixture.Source[:]),
			)
		}},
		{Name: "Sorted_Into", Call: func() {
			fixture.Observable = int(
				slices.Sorted_Into(fixture.Destination[:], fixture.Source[:]),
			)
		}},
		{Name: "Sorted_Function_Into", Call: func() {
			fixture.Observable = int(slices.Sorted_Function_Into(
				fixture.Destination[:], fixture.Source[:], zero_allocation_compare,
			))
		}},
		{Name: "Sorted_Stable_Function_Into", Call: func() {
			fixture.Observable = int(slices.Sorted_Stable_Function_Into(
				fixture.Destination[:], fixture.Source[:], zero_allocation_compare,
			))
		}},
		{Name: "Chunk", Call: func() {
			fixture.Observable = int(slices.Chunk(
				fixture.Source[:], 2, zero_allocation_chunk_yield,
			))
		}},
	}
}

func zero_allocation_boolean(value slices.Boolean) (number int) {
	if value {
		return 1
	}
	return 0
}

func zero_allocation_equal(left int, right int) (equal bool) {
	return left == right
}

func zero_allocation_even(value int) (selected bool) {
	return value%2 == 0
}

func zero_allocation_compare(left int, right int) (comparison slices.Comparison) {
	return slices.Comparison(left - right)
}

func zero_allocation_indexed_yield(
	_ slices.Search_Position, _ int,
) (continued slices.Boolean) {
	return true
}

func zero_allocation_yield(_ int) (continued slices.Boolean) {
	return true
}

func zero_allocation_chunk_yield(_ []int) (continued slices.Boolean) {
	return true
}
