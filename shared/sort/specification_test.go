// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by BSD-style license in Go source tree LICENSE file.

package sort_test

import (
	"cmp"
	"testing"

	"local/james-orcales/shared/sim/prng"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/sort"
	"local/james-orcales/shared/testify"
)

// Test_Ordering covers unstable, stable, reverse, and sorted-query contracts.
func Test_Ordering(t *testing.T) {
	t.Parallel()
	verify_unstable_order(t)
	verify_stable_order(t)
	verify_reverse_order(t)
	verify_sorted_query(t)
}

// Test_Search covers first-true, first-equal, and insertion positions.
func Test_Search(t *testing.T) {
	t.Parallel()
	state := search_state{Values: []int{1, 3, 3, 7}, Target: 3}
	position := sort.Search(sort.Count(len(state.Values)), state, at_least_target)
	testify.Equal_Values(t, 1, position, "Search must return first true position")
	position, found := sort.Find(sort.Count(len(state.Values)), state, compare_target)
	testify.Equal_Values(t, 1, position, "Find must return first equal position")
	testify.True(t, bool(found), "Find must report equal position")
	state.Target = 5
	position, found = sort.Find(sort.Count(len(state.Values)), state, compare_target)
	testify.Equal_Values(t, 3, position, "Find must return insertion position")
	testify.False(t, bool(found), "Find must report absent target")
	state.Target = 8
	position = sort.Search(sort.Count(len(state.Values)), state, at_least_target)
	testify.Equal_Values(t, len(state.Values), position,
		"Search must return count when predicate stays false")
}

// Test_Size_Limits covers largest input and rejection above shared boundary.
func Test_Size_Limits(t *testing.T) {
	t.Parallel()
	var maximum [sort.ELEMENT_COUNT_MAXIMUM]int
	for index := range maximum {
		maximum[index] = len(maximum) - index
	}
	sort.Sort(maximum[:], compare_integer)
	testify.True(t, bool(sort.Is_Sorted(maximum[:], compare_integer)),
		"Sort must admit ELEMENT_COUNT_MAXIMUM elements")
	position := sort.Search(sort.COUNT_MAXIMUM, search_state{}, always_false)
	testify.Equal_Values(t, sort.POSITION_MAXIMUM, position,
		"Search must admit COUNT_MAXIMUM")
	var too_many [sort.ELEMENT_COUNT_MAXIMUM + 1]int
	testify.Panics(t, func() { sort.Sort(too_many[:], compare_integer) },
		"Sort must reject oversized elements")
	testify.Panics(t, func() {
		sort.Search(sort.COUNT_MAXIMUM+1, search_state{}, always_false)
	}, "Search must reject oversized count")
}

// Test_Allocation proves every exported operation uses zero heap storage.
func Test_Allocation(t *testing.T) {
	var work_storage [ALLOCATION_ELEMENT_COUNT]int
	var stable_storage [ALLOCATION_ELEMENT_COUNT]stable_item
	fixture := new_allocation_fixture(work_storage[:], stable_storage[:])
	for _, check := range allocation_cases(fixture) {
		t.Run(check.Name, func(t *testing.T) {
			testify.Zero_Allocation(t, check.Call)
		})
	}
	testify.Not_Zero(t, fixture.Observable, "allocation probes must execute")
}

// Test_Domain_Errors covers invalid count and missing injected functions.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	testify.Panics(t, func() {
		sort.Search(sort.Count(-1), search_state{}, always_false)
	}, "Search must reject negative count")
	testify.Panics(t, func() {
		sort.Find(0, search_state{}, nil)
	}, "Find must reject nil comparison")
	testify.Panics(t, func() {
		sort.Sort([]int{1}, nil)
	}, "Sort must reject nil comparison")
}

// Test_Invariant_Domains reaches count, position, and report sentinels through public APIs.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	var maximum [sort.ELEMENT_COUNT_MAXIMUM]int
	var stable_maximum [sort.ELEMENT_COUNT_MAXIMUM]stable_item
	sort.Sort([]int{}, compare_integer)
	sort.Sort([]int{1}, compare_integer)
	sort.Sort([]int{2, 1}, compare_integer)
	sort.Sort(maximum[:], compare_integer)
	sort.Stable([]stable_item{}, compare_stable_item)
	sort.Stable([]stable_item{{Key: 1}}, compare_stable_item)
	sort.Stable([]stable_item{{Key: 2}, {Key: 1}}, compare_stable_item)
	sort.Stable(stable_maximum[:], compare_stable_item)
	sort.Is_Sorted([]int{}, compare_integer)
	sort.Is_Sorted([]int{1}, compare_integer)
	sort.Is_Sorted([]int{1, 2}, compare_integer)
	sort.Is_Sorted(maximum[:], compare_integer)
	sort.Is_Sorted([]int{2, 1}, compare_integer)
	cover_search_domains()
}

const ALLOCATION_ELEMENT_COUNT = sort.STABLE_INSERTION_BLOCK_COUNT*sort.BINARY_PART_COUNT + 1

const ALLOCATION_KEY_COUNT = sort.MEDIAN_ELEMENT_COUNT

const STANDARD_BINARY_PART_COUNT = 2

const STANDARD_SMALL_COUNT = 10 * 10

const STANDARD_POWER = 10

const STANDARD_POWER_COUNT = 1 << STANDARD_POWER

const STANDARD_STORAGE_COUNT = STANDARD_POWER_COUNT + 1

const STANDARD_DISTRIBUTION_COUNT = 5

const STANDARD_MODE_COUNT = STANDARD_DISTRIBUTION_COUNT + 1

const STANDARD_DITHER_MODULUS = STANDARD_DISTRIBUTION_COUNT

const STANDARD_SAWTOOTH = 0

const STANDARD_RANDOM = STANDARD_SAWTOOTH + 1

const STANDARD_STAGGER = STANDARD_RANDOM + 1

const STANDARD_PLATEAU = STANDARD_STAGGER + 1

const STANDARD_SHUFFLE = STANDARD_PLATEAU + 1

const STANDARD_COPY = 0

const STANDARD_REVERSE = STANDARD_COPY + 1

const STANDARD_REVERSE_FIRST = STANDARD_REVERSE + 1

const STANDARD_REVERSE_SECOND = STANDARD_REVERSE_FIRST + 1

const STANDARD_SORTED = STANDARD_REVERSE_SECOND + 1

const STANDARD_DITHER = STANDARD_SORTED + 1

type stable_item struct {
	Key     int
	Ordinal int
}

type search_state struct {
	Values []int
	Target int
}

type allocation_check struct {
	Name string
	Call func()
}

type allocation_fixture struct {
	// Work and Stable view fixed arrays owned by Test_Allocation: slice headers allocate
	// nothing and no fixed array sits in a struct field.
	Work       []int
	Stable     []stable_item
	Search     search_state
	Observable int
}

type standard_search_state struct {
	Target int
}

type counted_integer struct {
	Value int
	Count *int
}

func verify_unstable_order(t *testing.T) {
	t.Helper()
	values := []int{5, 2, 9, 1, 7, 3}
	sort.Sort(values, compare_integer)
	testify.Equal(t, []int{1, 2, 3, 5, 7, 9}, values,
		"Sort must order elements")
	sort.Sort([]int{}, compare_integer)
	one := []int{1}
	sort.Sort(one, compare_integer)
	testify.Equal(t, []int{1}, one, "Sort must preserve singleton")
}

func verify_stable_order(t *testing.T) {
	t.Helper()
	values := []stable_item{
		{Key: 2, Ordinal: 0},
		{Key: 1, Ordinal: 1},
		{Key: 2, Ordinal: 2},
		{Key: 1, Ordinal: 3},
		{Key: 2, Ordinal: 4},
	}
	sort.Stable(values, compare_stable_item)
	want := []stable_item{
		{Key: 1, Ordinal: 1},
		{Key: 1, Ordinal: 3},
		{Key: 2, Ordinal: 0},
		{Key: 2, Ordinal: 2},
		{Key: 2, Ordinal: 4},
	}
	testify.Equal(t, want, values, "Stable must preserve equal-element order")
}

func verify_reverse_order(t *testing.T) {
	t.Helper()
	values := []int{5, 2, 9, 1, 7, 3}
	sort.Sort(values, compare_integer_reverse)
	testify.Equal(t, []int{9, 7, 5, 3, 2, 1}, values,
		"exchanged comparison must reverse order")
}

func verify_sorted_query(t *testing.T) {
	t.Helper()
	testify.True(t, bool(sort.Is_Sorted([]int{1, 2, 2, 3}, compare_integer)),
		"Is_Sorted must accept selected order")
	testify.False(t, bool(sort.Is_Sorted([]int{3, 2, 2, 1}, compare_integer)),
		"Is_Sorted must reject opposite order")
	testify.True(t, bool(sort.Is_Sorted(
		[]int{3, 2, 2, 1}, compare_integer_reverse,
	)), "Is_Sorted must honor exchanged comparison")
}

func cover_search_domains() {
	sort.Search(0, search_state{}, always_false)
	sort.Search(1, search_state{}, always_false)
	sort.Search(2, search_state{}, always_false)
	sort.Search(sort.COUNT_MAXIMUM, search_state{}, always_false)
	sort.Find(0, search_state{}, always_greater)
	sort.Find(1, search_state{}, always_greater)
	sort.Find(2, search_state{}, always_greater)
	sort.Find(sort.COUNT_MAXIMUM, search_state{}, always_greater)
	sort.Find(2, search_state{Values: []int{0, 2}, Target: 2}, compare_target)
}

func compare_integer(left int, right int) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(left, right))
}

func compare_integer_reverse(left int, right int) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(right, left))
}

func compare_stable_item(
	left stable_item, right stable_item,
) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(left.Key, right.Key))
}

func compare_counted_integer(
	left counted_integer, right counted_integer,
) (comparison slices.Comparison) {
	*left.Count++
	return slices.Comparison(cmp.Compare(left.Value, right.Value))
}

func at_least_target(state search_state, position sort.Position) (selected sort.Boolean) {
	return state.Values[position] >= state.Target
}

func always_false(_ search_state, _ sort.Position) (selected sort.Boolean) {
	return false
}

func always_greater(_ search_state, _ sort.Position) (comparison slices.Comparison) {
	return 1
}

func compare_target(
	state search_state, position sort.Position,
) (comparison slices.Comparison) {
	return slices.Comparison(state.Target - state.Values[position])
}

func new_allocation_fixture(
	work []int, stable []stable_item,
) (fixture *allocation_fixture) {
	fixture = &allocation_fixture{
		Work:   work,
		Stable: stable,
		Search: search_state{Values: []int{1, 3, 5, 7}, Target: 5},
	}
	reset_allocation_fixture(fixture)
	return fixture
}

func reset_allocation_fixture(fixture *allocation_fixture) {
	for index := range fixture.Work {
		fixture.Work[index] = len(fixture.Work) - index
		fixture.Stable[index] = stable_item{
			Key:     index%ALLOCATION_KEY_COUNT + 1,
			Ordinal: index,
		}
	}
}

func allocation_cases(fixture *allocation_fixture) (cases []allocation_check) {
	return []allocation_check{
		{Name: "Sort", Call: func() {
			reset_allocation_fixture(fixture)
			sort.Sort(fixture.Work, compare_integer)
			fixture.Observable = fixture.Work[0]
		}},
		{Name: "Stable", Call: func() {
			reset_allocation_fixture(fixture)
			sort.Stable(fixture.Stable, compare_stable_item)
			fixture.Observable = fixture.Stable[0].Key
		}},
		{Name: "Is_Sorted", Call: func() {
			fixture.Observable = boolean_number(
				sort.Is_Sorted(fixture.Work, compare_integer),
			)
		}},
		{Name: "Search", Call: func() {
			fixture.Observable = int(sort.Search(
				sort.Count(len(fixture.Search.Values)),
				fixture.Search,
				at_least_target,
			))
		}},
		{Name: "Find", Call: func() {
			position, found := sort.Find(
				sort.Count(len(fixture.Search.Values)),
				fixture.Search,
				compare_target,
			)
			fixture.Observable = int(position) + boolean_number(found)
		}},
	}
}

func boolean_number(value sort.Boolean) (number int) {
	if value {
		return 1
	}
	return 0
}

// Test_Standard_Library_Largest_Random keeps upstream purpose inside package bound.
func Test_Standard_Library_Largest_Random(t *testing.T) {
	t.Parallel()
	var values [sort.ELEMENT_COUNT_MAXIMUM]int
	generator := prng.New(1)
	for index := range values {
		values[index] = int(prng.Xoshiro_Below(&generator, STANDARD_SMALL_COUNT))
	}
	testify.False(t, bool(sort.Is_Sorted(values[:], compare_integer)),
		"deterministic random fixture must start unordered")
	sort.Sort(values[:], compare_integer)
	testify.True(t, bool(sort.Is_Sorted(values[:], compare_integer)),
		"largest deterministic random fixture must sort")
}

// Test_Standard_Library_Already_Sorted preserves PDQ linear sorted-input path.
func Test_Standard_Library_Already_Sorted(t *testing.T) {
	t.Parallel()
	var values [STANDARD_SMALL_COUNT]counted_integer
	comparison_count := 0
	for index := range values {
		values[index] = counted_integer{Value: index, Count: &comparison_count}
	}
	sort.Sort(values[:], compare_counted_integer)
	testify.True(t,
		comparison_count <= len(values)+sort.PIVOT_COMPARISON_COUNT,
		"sorted input must keep PDQ linear comparison count: %d", comparison_count,
	)
}

// Test_Standard_Library_Break_Patterns protects imbalanced partition path.
func Test_Standard_Library_Break_Patterns(t *testing.T) {
	t.Parallel()
	values := make([]int, STANDARD_SMALL_COUNT/STANDARD_BINARY_PART_COUNT)
	for index := range values {
		values[index] = STANDARD_SMALL_COUNT / STANDARD_BINARY_PART_COUNT
	}
	quarter_count := STANDARD_MODE_COUNT / STANDARD_BINARY_PART_COUNT
	for quarter_index := 1; quarter_index < quarter_count; quarter_index++ {
		values[len(values)/sort.QUARTER_COUNT*quarter_index] = quarter_index
	}
	sort.Sort(values, compare_integer)
	testify.True(t, bool(sort.Is_Sorted(values, compare_integer)),
		"pattern-breaking fixture must sort")
}

// Test_Standard_Library_Bentley_Mc_Ilroy keeps broad upstream pattern matrix.
func Test_Standard_Library_Bentley_Mc_Ilroy(t *testing.T) {
	t.Parallel()
	sizes := [...]int{
		STANDARD_SMALL_COUNT,
		STANDARD_POWER_COUNT - 1,
		STANDARD_POWER_COUNT,
		STANDARD_POWER_COUNT + 1,
	}
	var source [STANDARD_STORAGE_COUNT]int
	var work [STANDARD_STORAGE_COUNT]int
	generator := prng.New(3)
	for _, element_count := range sizes {
		run_standard_sizes(t, source[:element_count], work[:element_count], &generator)
	}
}

// Test_Standard_Library_Stable_Matrix protects equal runs at largest bound.
func Test_Standard_Library_Stable_Matrix(t *testing.T) {
	t.Parallel()
	var values [sort.ELEMENT_COUNT_MAXIMUM]stable_item
	for index := range values {
		values[index] = stable_item{
			Key: index % STANDARD_DITHER_MODULUS, Ordinal: index,
		}
	}
	sort.Stable(values[:], compare_stable_item)
	verify_stable_matrix(t, values[:])
}

// Test_Standard_Library_Search_Exhaustive ports every small first-true position.
func Test_Standard_Library_Search_Exhaustive(t *testing.T) {
	t.Parallel()
	for count := 0; count <= STANDARD_SMALL_COUNT; count++ {
		for target := 0; target <= count; target++ {
			state := standard_search_state{Target: target}
			position := sort.Search(sort.Count(count), state, position_at_least_target)
			testify.Equal_Values(t, target, position,
				"Search count %d target %d returned wrong position", count, target)
		}
	}
}

// Test_Standard_Library_Find_Exhaustive ports every small equality and insertion result.
func Test_Standard_Library_Find_Exhaustive(t *testing.T) {
	t.Parallel()
	for count := 0; count <= STANDARD_SMALL_COUNT; count++ {
		for target := 1; target <= count*STANDARD_BINARY_PART_COUNT+1; target++ {
			verify_standard_find(t, count, target)
		}
	}
}

func run_standard_sizes(
	t *testing.T, source []int, work []int, generator *prng.Xoshiro,
) {
	for modulus := 1; modulus < len(source)*STANDARD_BINARY_PART_COUNT; modulus *= 2 {
		distribution_limit := STANDARD_DISTRIBUTION_COUNT
		for distribution_index := range distribution_limit {
			fill_standard_distribution(source, modulus, distribution_index, generator)
			run_standard_modes(t, source, work)
		}
	}
}

func run_standard_modes(t *testing.T, source []int, work []int) {
	for mode_index := 0; mode_index < STANDARD_MODE_COUNT; mode_index++ {
		fill_standard_mode(source, work, mode_index)
		sort.Sort(work, compare_integer)
		testify.True(t, bool(sort.Is_Sorted(work, compare_integer)),
			"Bentley-McIlroy count %d mode %d did not sort", len(work), mode_index)
	}
}

func fill_standard_distribution(
	values []int, modulus int, distribution int, generator *prng.Xoshiro,
) {
	left, right := 0, 1
	for index := range values {
		switch distribution {
		case STANDARD_SAWTOOTH:
			values[index] = index % modulus
		case STANDARD_RANDOM:
			values[index] = int(prng.Xoshiro_Below(generator, prng.Bound(modulus)))
		case STANDARD_STAGGER:
			values[index] = (index*modulus + index) % len(values)
		case STANDARD_PLATEAU:
			values[index] = min(index, modulus)
		case STANDARD_SHUFFLE:
			left, right = shuffled_values(generator, modulus, left, right)
			values[index] = max(left, right)
		}
	}
}

func shuffled_values(
	generator *prng.Xoshiro, modulus int, left int, right int,
) (next_left int, next_right int) {
	next_left, next_right = left, right
	if prng.Xoshiro_Below(generator, prng.Bound(modulus)) != 0 {
		next_left += STANDARD_BINARY_PART_COUNT
		return next_left, next_right
	}
	next_right += STANDARD_BINARY_PART_COUNT
	return next_left, next_right
}

func fill_standard_mode(source []int, work []int, mode int) {
	switch mode {
	case STANDARD_COPY:
		copy(work, source)
	case STANDARD_REVERSE:
		copy_reverse(work, source, 0, len(source))
	case STANDARD_REVERSE_FIRST:
		copy_reverse(work, source, 0, len(source)/STANDARD_BINARY_PART_COUNT)
		copy(work[len(source)/STANDARD_BINARY_PART_COUNT:],
			source[len(source)/STANDARD_BINARY_PART_COUNT:])
	case STANDARD_REVERSE_SECOND:
		copy(work, source)
		copy_reverse(work, source, len(source)/STANDARD_BINARY_PART_COUNT, len(source))
	case STANDARD_SORTED:
		copy(work, source)
		sort.Sort(work, compare_integer)
	case STANDARD_DITHER:
		for index := range source {
			work[index] = source[index] + index%STANDARD_DITHER_MODULUS
		}
	}
}

func copy_reverse(destination []int, source []int, first int, final int) {
	copy(destination, source)
	for position := first; position < final; position++ {
		destination[position] = source[final-(position-first)-1]
	}
}

func verify_stable_matrix(t *testing.T, values []stable_item) {
	for index := 1; index < len(values); index++ {
		testify.True(t, values[index-1].Key <= values[index].Key,
			"Stable result decreases at position %d", index)
		if values[index-1].Key == values[index].Key {
			testify.True(t, values[index-1].Ordinal < values[index].Ordinal,
				"Stable result exchanges equal values at position %d", index)
		}
	}
}

func position_at_least_target(
	state standard_search_state, position sort.Position,
) (selected sort.Boolean) {
	return int(position) >= state.Target
}

func compare_abstract_target(
	state standard_search_state, position sort.Position,
) (comparison slices.Comparison) {
	element := (int(position) + 1) * STANDARD_BINARY_PART_COUNT
	return slices.Comparison(state.Target - element)
}

func verify_standard_find(t *testing.T, count int, target int) {
	state := standard_search_state{Target: target}
	position, found := sort.Find(sort.Count(count), state, compare_abstract_target)
	want_position := target / STANDARD_BINARY_PART_COUNT
	want_found := target%STANDARD_BINARY_PART_COUNT == 0
	if want_found {
		want_position--
	}
	testify.Equal_Values(t, want_position, position,
		"Find count %d target %d returned wrong position", count, target)
	testify.Equal(t, want_found, bool(found),
		"Find count %d target %d returned wrong report", count, target)
}
