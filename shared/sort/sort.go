// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by BSD-style license in Go source tree LICENSE file.

// Package sort supplies bounded zero-allocation standard-library sorting and search
// algorithms. Caller owns element and search state storage.
package sort

import (
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/slices"
)

// ELEMENT_COUNT_MAXIMUM matches shared slice boundary, so one count limits all operations.
const ELEMENT_COUNT_MAXIMUM = slices.SLICE_COUNT_MAXIMUM

// COUNT_MINIMUM is empty input count.
const COUNT_MINIMUM = 0

// COUNT_MAXIMUM is largest admitted input count.
const COUNT_MAXIMUM = ELEMENT_COUNT_MAXIMUM

// POSITION_MINIMUM is first input position.
const POSITION_MINIMUM = COUNT_MINIMUM

// POSITION_MAXIMUM includes insertion position after largest admitted input.
const POSITION_MAXIMUM = COUNT_MAXIMUM

// BINARY_PART_COUNT is child count in heap and partition count in quicksort.
const BINARY_PART_COUNT = 2

// MEDIAN_ELEMENT_COUNT is pivot sample count.
const MEDIAN_ELEMENT_COUNT = 3

// QUARTER_COUNT places pivot samples at input quarters.
const QUARTER_COUNT = 4

// INSERTION_SORT_ELEMENT_COUNT_MAXIMUM preserves standard-library small-range threshold.
const INSERTION_SORT_ELEMENT_COUNT_MAXIMUM = MEDIAN_ELEMENT_COUNT * QUARTER_COUNT

// PARTITION_IMBALANCE_DENOMINATOR preserves standard-library one-eighth threshold.
const PARTITION_IMBALANCE_DENOMINATOR = BINARY_PART_COUNT * QUARTER_COUNT

// PARTIAL_INSERTION_STEP_COUNT_MAXIMUM preserves standard-library five-step tuning factor.
const PARTIAL_INSERTION_STEP_COUNT_MAXIMUM = MEDIAN_ELEMENT_COUNT + BINARY_PART_COUNT

// PARTIAL_INSERTION_ELEMENT_COUNT_MINIMUM keeps shifting work on ranges large enough to repay it.
const PARTIAL_INSERTION_ELEMENT_COUNT_MINIMUM = PARTIAL_INSERTION_STEP_COUNT_MAXIMUM *
	BINARY_PART_COUNT *
	PARTIAL_INSERTION_STEP_COUNT_MAXIMUM

// STABLE_INSERTION_BLOCK_COUNT preserves standard-library twenty-element block through arities.
const STABLE_INSERTION_BLOCK_COUNT = QUARTER_COUNT * PARTIAL_INSERTION_STEP_COUNT_MAXIMUM

// PATTERN_BREAK_ELEMENT_COUNT_MINIMUM is first range large enough to scatter around its center.
const PATTERN_BREAK_ELEMENT_COUNT_MINIMUM = BINARY_PART_COUNT * QUARTER_COUNT

// NINTHER_ELEMENT_COUNT_MINIMUM keeps twelve pivot comparisons on ranges that can repay them.
const NINTHER_ELEMENT_COUNT_MINIMUM = PARTIAL_INSERTION_ELEMENT_COUNT_MINIMUM

// PIVOT_COMPARISON_COUNT is comparison count for four median-of-three operations.
const PIVOT_COMPARISON_COUNT = QUARTER_COUNT * MEDIAN_ELEMENT_COUNT

// PIVOT_SWAP_COUNT_MAXIMUM is largest swap count those comparisons can report.
const PIVOT_SWAP_COUNT_MAXIMUM = PIVOT_COMPARISON_COUNT

// PATTERN_RANDOM_LEFT_SHIFT_FIRST is first xorshift coefficient from standard-library PDQ sort.
const PATTERN_RANDOM_LEFT_SHIFT_FIRST = INSERTION_SORT_ELEMENT_COUNT_MAXIMUM + 1

// PATTERN_RANDOM_RIGHT_SHIFT is middle xorshift coefficient from standard-library PDQ sort.
const PATTERN_RANDOM_RIGHT_SHIFT = QUARTER_COUNT + MEDIAN_ELEMENT_COUNT

// PATTERN_RANDOM_LEFT_SHIFT_SECOND is final xorshift coefficient from standard-library PDQ
// sort.
const PATTERN_RANDOM_LEFT_SHIFT_SECOND = STABLE_INSERTION_BLOCK_COUNT - MEDIAN_ELEMENT_COUNT

// PIVOT_HINT_ASCENT marks sampled values that already ascend.
const PIVOT_HINT_ASCENT = slices.ORDERING_GREATER

// PIVOT_HINT_UNKNOWN marks sampled values without one monotone direction.
const PIVOT_HINT_UNKNOWN = slices.ORDERING_EQUAL

// PIVOT_HINT_DESCENT marks sampled values that descend.
const PIVOT_HINT_DESCENT = slices.ORDERING_LESS

// SORT_STACK_COUNT_MAXIMUM bounds pending partitions by machine-word bit count.
const SORT_STACK_COUNT_MAXIMUM = bits.BIT_COUNT_WORD_MAXIMUM

// MERGE_STACK_COUNT_MAXIMUM bounds pending symmetric merges by machine-word bit count.
const MERGE_STACK_COUNT_MAXIMUM = bits.BIT_COUNT_WORD_MAXIMUM

// Count is element count.
type Count int

// Count_Invariants states complete bounded count domain.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Position is element position or insertion position after final element.
type Position int

// Position_Invariants states complete bounded position domain.
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Boolean is true or false operation report.
type Boolean bool

// Boolean_Invariants states both report results as obligations.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The sort report is true.").
		Ensure()
}

// Search_Function reports whether monotone predicate is true at one position.
type Search_Function[State any] func(state State, position Position) (selected Boolean)

// Find_Function compares wanted value against value at one position.
type Find_Function[State any] func(
	state State, position Position,
) (comparison slices.Comparison)

// Sort orders elements without preserving order between equal values.
func Sort[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) {
	enforce_elements(elements, comparison)
	if len(elements) <= 1 {
		return
	}
	pdq_sort(elements, comparison)
}

// Stable orders elements while preserving order between equal values.
func Stable[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) {
	enforce_elements(elements, comparison)
	stable(elements, comparison)
}

// Is_Sorted reports whether elements have injected order.
func Is_Sorted[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) (sorted Boolean) {
	defer func() { Boolean_Invariants(sorted, "is_sorted.sorted") }()
	enforce_elements(elements, comparison)
	for position := len(elements) - 1; position > 0; position-- {
		if comparison(elements[position], elements[position-1]) < 0 {
			return false
		}
	}
	return true
}

// Search returns first position where predicate is true, or count when none is true.
func Search[State any](
	count Count, state State, predicate Search_Function[State],
) (position Position) {
	defer func() { Position_Invariants(position, "search.position") }()
	Count_Invariants(count, "search.count")
	invariant.Always(predicate != nil, "A Search defines its predicate.")
	left, right := 0, int(count)
	for left < right {
		middle := int(uint(left+right) >> 1)
		if !predicate(state, Position(middle)) {
			left = middle + 1
		} else {
			right = middle
		}
	}
	return Position(left)
}

// Find returns first nonpositive comparison position and whether comparison equals zero.
func Find[State any](
	count Count, state State, comparison Find_Function[State],
) (position Position, found Boolean) {
	defer func() {
		Position_Invariants(position, "find.position")
		Boolean_Invariants(found, "find.found")
	}()
	Count_Invariants(count, "find.count")
	invariant.Always(comparison != nil, "A Find defines its comparison.")
	left, right := 0, int(count)
	for left < right {
		middle := int(uint(left+right) >> 1)
		if comparison(state, Position(middle)) > 0 {
			left = middle + 1
		} else {
			right = middle
		}
	}
	position = Position(left)
	if left < int(count) {
		found = comparison(state, position) == 0
	}
	return position, found
}

func enforce_elements[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) {
	invariant.Always(
		len(elements) <= ELEMENT_COUNT_MAXIMUM,
		"A sort input admits at most ELEMENT_COUNT_MAXIMUM elements.",
	)
	invariant.Always(comparison != nil, "A sort input defines its comparison.")
}

func pdq_sort[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) {
	var partitions, predecessors [SORT_STACK_COUNT_MAXIMUM]S
	var limits [SORT_STACK_COUNT_MAXIMUM]int
	stack_count := 1
	partitions[0], limits[0] = elements, int(bits.Bit_Size(bits.Word(len(elements))))
	for stack_count > 0 {
		stack_count--
		current, predecessor, limit :=
			partitions[stack_count], predecessors[stack_count], limits[stack_count]
		was_balanced, was_partitioned := true, true
		for current != nil {
			if len(current) <= INSERTION_SORT_ELEMENT_COUNT_MAXIMUM {
				insertion_sort(current, comparison)
				break
			}
			if limit == 0 {
				heap_sort(current, comparison)
				break
			}
			if !was_balanced {
				break_patterns(current)
				limit--
			}
			pivot, hint := choose_pivot(current, comparison)
			if hint == PIVOT_HINT_DESCENT {
				pivot_index := len(current) - len(pivot)
				reverse_range(current)
				pivot = current[len(current)-1-pivot_index:]
				hint = PIVOT_HINT_ASCENT
			}
			if was_balanced {
				if was_partitioned {
					if hint == PIVOT_HINT_ASCENT {
						if partial_insertion_sort(current, comparison) {
							break
						}
					}
				}
			}
			if predecessor != nil {
				if comparison(predecessor[0], pivot[0]) >= 0 {
					next := partition_equal(current, pivot, comparison)
					predecessor = current[len(current)-len(next)-1:]
					current = next
					continue
				}
			}
			left, right, already_partitioned := partition(current, pivot, comparison)
			was_partitioned = bool(already_partitioned)
			threshold := len(current) / PARTITION_IMBALANCE_DENOMINATOR
			pivot = current[len(left):]
			if len(left) < len(right)+1 {
				was_balanced = len(left) >= threshold
				partitions[stack_count] = left
				predecessors[stack_count] = predecessor
				limits[stack_count] = limit
				stack_count++
				predecessor = pivot
				current = right
			} else {
				was_balanced = len(right)+1 >= threshold
				partitions[stack_count] = right
				predecessors[stack_count] = pivot
				limits[stack_count] = limit
				stack_count++
				current = left
			}
		}
	}
}

func insertion_sort[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) {
	for index := 1; index < len(elements); index++ {
		for position := index; position > 0; position-- {
			if comparison(elements[position], elements[position-1]) >= 0 {
				break
			}
			elements[position], elements[position-1] =
				elements[position-1], elements[position]
		}
	}
}

func heap_sort[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) {
	for root := len(elements)/BINARY_PART_COUNT - 1; root >= 0; root-- {
		sift_down(elements, elements[root:], comparison)
	}
	for final := len(elements) - 1; final > 0; final-- {
		elements[0], elements[final] = elements[final], elements[0]
		heap := elements[:final]
		sift_down(heap, heap, comparison)
	}
}

func sift_down[S ~[]E, E any](
	heap S, root_tail S, comparison slices.Comparison_Function[E, E],
) {
	root := len(heap) - len(root_tail)
	child := BINARY_PART_COUNT*root + 1
	for child < len(heap) {
		if child+1 < len(heap) {
			if comparison(heap[child], heap[child+1]) < 0 {
				child++
			}
		}
		if comparison(heap[root], heap[child]) >= 0 {
			return
		}
		heap[root], heap[child] = heap[child], heap[root]
		root = child
		child = BINARY_PART_COUNT*root + 1
	}
}

func break_patterns[S ~[]E, E any](elements S) {
	if len(elements) < PATTERN_BREAK_ELEMENT_COUNT_MINIMUM {
		return
	}
	random := uint64(len(elements))
	modulus := 1 << bits.Bit_Size(bits.Word(len(elements)))
	first := len(elements)/BINARY_PART_COUNT - MEDIAN_ELEMENT_COUNT/BINARY_PART_COUNT
	for position_index := range MEDIAN_ELEMENT_COUNT {
		random ^= random << PATTERN_RANDOM_LEFT_SHIFT_FIRST
		random ^= random >> PATTERN_RANDOM_RIGHT_SHIFT
		random ^= random << PATTERN_RANDOM_LEFT_SHIFT_SECOND
		other := int(random & uint64(modulus-1))
		if other >= len(elements) {
			other -= len(elements)
		}
		position := first + position_index
		elements[position], elements[other] = elements[other], elements[position]
	}
}

func partition[S ~[]E, E any](
	elements S, pivot_tail S, comparison slices.Comparison_Function[E, E],
) (left S, right S, already_partitioned Boolean) {
	defer func() {
		Boolean_Invariants(already_partitioned, "partition.already_partitioned")
	}()
	pivot := len(elements) - len(pivot_tail)
	elements[0], elements[pivot] = elements[pivot], elements[0]
	left_index, right_index := 1, len(elements)-1
	for left_index <= right_index && comparison(elements[left_index], elements[0]) < 0 {
		left_index++
	}
	for left_index <= right_index && comparison(elements[right_index], elements[0]) >= 0 {
		right_index--
	}
	if left_index > right_index {
		elements[0], elements[right_index] = elements[right_index], elements[0]
		return elements[:right_index], elements[right_index+1:], true
	}
	elements[left_index], elements[right_index] =
		elements[right_index], elements[left_index]
	left_index++
	right_index--
	for left_index <= right_index {
		for left_index <= right_index &&
			comparison(elements[left_index], elements[0]) < 0 {
			left_index++
		}
		for left_index <= right_index &&
			comparison(elements[right_index], elements[0]) >= 0 {
			right_index--
		}
		if left_index > right_index {
			break
		}
		elements[left_index], elements[right_index] =
			elements[right_index], elements[left_index]
		left_index++
		right_index--
	}
	elements[0], elements[right_index] = elements[right_index], elements[0]
	return elements[:right_index], elements[right_index+1:], false
}

func partition_equal[S ~[]E, E any](
	elements S, pivot_tail S, comparison slices.Comparison_Function[E, E],
) (greater S) {
	pivot := len(elements) - len(pivot_tail)
	elements[0], elements[pivot] = elements[pivot], elements[0]
	left_index, right_index := 1, len(elements)-1
	for left_index <= right_index {
		for left_index <= right_index &&
			comparison(elements[0], elements[left_index]) >= 0 {
			left_index++
		}
		for left_index <= right_index &&
			comparison(elements[0], elements[right_index]) < 0 {
			right_index--
		}
		if left_index > right_index {
			break
		}
		elements[left_index], elements[right_index] =
			elements[right_index], elements[left_index]
		left_index++
		right_index--
	}
	return elements[left_index:]
}

func partial_insertion_sort[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) (sorted Boolean) {
	defer func() { Boolean_Invariants(sorted, "partial_insertion_sort.sorted") }()
	position := 1
	for step_index := 0; step_index < PARTIAL_INSERTION_STEP_COUNT_MAXIMUM; step_index++ {
		for position < len(elements) {
			if comparison(elements[position], elements[position-1]) < 0 {
				break
			}
			position++
		}
		if position == len(elements) {
			return true
		}
		if len(elements) < PARTIAL_INSERTION_ELEMENT_COUNT_MINIMUM {
			return false
		}
		elements[position], elements[position-1] =
			elements[position-1], elements[position]
		if position >= BINARY_PART_COUNT {
			for shifted := position - 1; shifted >= 1; shifted-- {
				if comparison(elements[shifted], elements[shifted-1]) >= 0 {
					break
				}
				elements[shifted], elements[shifted-1] =
					elements[shifted-1], elements[shifted]
			}
		}
		if len(elements)-position >= BINARY_PART_COUNT {
			for shifted := position + 1; shifted < len(elements); shifted++ {
				if comparison(elements[shifted], elements[shifted-1]) >= 0 {
					break
				}
				elements[shifted], elements[shifted-1] =
					elements[shifted-1], elements[shifted]
			}
		}
	}
	return false
}

func choose_pivot[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) (pivot S, hint slices.Ordering) {
	defer func() { slices.Ordering_Invariants(hint, "choose_pivot.hint") }()
	// Slice length carries bounded swap count without scalar helper state or closure storage.
	swaps := elements[:0]
	first := elements[len(elements)/QUARTER_COUNT:]
	middle := elements[len(elements)/QUARTER_COUNT*BINARY_PART_COUNT:]
	final := elements[len(elements)/QUARTER_COUNT*MEDIAN_ELEMENT_COUNT:]
	if len(elements) >= PATTERN_BREAK_ELEMENT_COUNT_MINIMUM {
		if len(elements) >= NINTHER_ELEMENT_COUNT_MINIMUM {
			first, swaps = median_adjacent(elements, first, swaps, comparison)
			middle, swaps = median_adjacent(elements, middle, swaps, comparison)
			final, swaps = median_adjacent(elements, final, swaps, comparison)
		}
		middle, swaps = median(elements, first, middle, final, swaps, comparison)
	}
	switch len(swaps) {
	case 0:
		return middle, PIVOT_HINT_ASCENT
	case PIVOT_SWAP_COUNT_MAXIMUM:
		return middle, PIVOT_HINT_DESCENT
	default:
		return middle, PIVOT_HINT_UNKNOWN
	}
}

func median_adjacent[S ~[]E, E any](
	elements S,
	middle S,
	swaps S,
	comparison slices.Comparison_Function[E, E],
) (median_tail S, swaps_after S) {
	position := len(elements) - len(middle)
	return median(
		elements,
		elements[position-1:],
		middle,
		elements[position+1:],
		swaps,
		comparison,
	)
}

func median[S ~[]E, E any](
	elements S,
	first S,
	middle S,
	final S,
	swaps S,
	comparison slices.Comparison_Function[E, E],
) (median_tail S, swaps_after S) {
	first, middle, swaps = order_two(elements, first, middle, swaps, comparison)
	middle, final, swaps = order_two(elements, middle, final, swaps, comparison)
	_, middle, swaps = order_two(elements, first, middle, swaps, comparison)
	return middle, swaps
}

func order_two[S ~[]E, E any](
	elements S,
	first S,
	second S,
	swaps S,
	comparison slices.Comparison_Function[E, E],
) (least S, greatest S, swaps_after S) {
	first_position := len(elements) - len(first)
	second_position := len(elements) - len(second)
	if comparison(elements[second_position], elements[first_position]) < 0 {
		return second, first, elements[:len(swaps)+1]
	}
	return first, second, swaps
}

func reverse_range[S ~[]E, E any](elements S) {
	left, right := 0, len(elements)-1
	for left < right {
		elements[left], elements[right] = elements[right], elements[left]
		left++
		right--
	}
}

func stable[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) {
	first, final := 0, STABLE_INSERTION_BLOCK_COUNT
	for final <= len(elements) {
		insertion_sort(elements[first:final], comparison)
		first = final
		final += STABLE_INSERTION_BLOCK_COUNT
	}
	insertion_sort(elements[first:], comparison)
	for block_count := STABLE_INSERTION_BLOCK_COUNT; block_count < len(elements); {
		merge_blocks(elements, elements[block_count:], comparison)
		block_count *= BINARY_PART_COUNT
	}
}

func merge_blocks[S ~[]E, E any](
	elements S, block_tail S, comparison slices.Comparison_Function[E, E],
) {
	block_count := len(elements) - len(block_tail)
	first := 0
	final := BINARY_PART_COUNT * block_count
	for final <= len(elements) {
		merged := elements[first:final]
		symmetric_merge(merged, merged[block_count:], comparison)
		first = final
		final += BINARY_PART_COUNT * block_count
	}
	middle := first + block_count
	if middle < len(elements) {
		merged := elements[first:]
		symmetric_merge(merged, merged[block_count:], comparison)
	}
}

func symmetric_merge[S ~[]E, E any](
	elements S, middle_tail S, comparison slices.Comparison_Function[E, E],
) {
	var merges [MERGE_STACK_COUNT_MAXIMUM]S
	var middles [MERGE_STACK_COUNT_MAXIMUM]S
	stack_count := 1
	merges[0], middles[0] = elements, middle_tail
	for stack_count > 0 {
		stack_count--
		current := merges[stack_count]
		middle := len(current) - len(middles[stack_count])
		if merge_single(current, current[middle:], comparison) {
			continue
		}
		center := len(current) / BINARY_PART_COUNT
		sum := center + middle
		start, boundary := 0, min(center, middle)
		if middle > center {
			start = sum - len(current)
		}
		pivot := sum - 1
		for start < boundary {
			candidate := int(uint(start+boundary) >> 1)
			if comparison(current[pivot-candidate], current[candidate]) >= 0 {
				start = candidate + 1
			} else {
				boundary = candidate
			}
		}
		end := sum - start
		if start < middle {
			if middle < end {
				rotate(current[start:end], current[middle:end])
			}
		}
		if center < end {
			if end < len(current) {
				right := current[center:]
				right_middle := right[end-center:]
				merges[stack_count], middles[stack_count] = right, right_middle
				stack_count++
			}
		}
		if start > 0 {
			if start < center {
				left := current[:center]
				merges[stack_count], middles[stack_count] = left, left[start:]
				stack_count++
			}
		}
	}
}

func merge_single[S ~[]E, E any](
	elements S, middle_tail S, comparison slices.Comparison_Function[E, E],
) (merged Boolean) {
	defer func() { Boolean_Invariants(merged, "merge_single.merged") }()
	middle := len(elements) - len(middle_tail)
	if middle == 1 {
		left, right := middle, len(elements)
		for left < right {
			center := int(uint(left+right) >> 1)
			if comparison(elements[center], elements[0]) < 0 {
				left = center + 1
			} else {
				right = center
			}
		}
		for position_index := 0; position_index < left-1; position_index++ {
			elements[position_index], elements[position_index+1] =
				elements[position_index+1], elements[position_index]
		}
		return true
	}
	if len(elements)-middle == 1 {
		merge_single_right(elements, middle_tail, comparison)
		return true
	}
	return false
}

func merge_single_right[S ~[]E, E any](
	elements S, middle_tail S, comparison slices.Comparison_Function[E, E],
) {
	middle := len(elements) - len(middle_tail)
	left, right := 0, middle
	for left < right {
		center := int(uint(left+right) >> 1)
		if comparison(elements[middle], elements[center]) >= 0 {
			left = center + 1
		} else {
			right = center
		}
	}
	for position := middle; position > left; position-- {
		elements[position], elements[position-1] =
			elements[position-1], elements[position]
	}
}

func rotate[S ~[]E, E any](elements S, middle_tail S) {
	middle := len(elements) - len(middle_tail)
	left_count := middle
	right_count := len(elements) - middle
	for left_count != right_count {
		if left_count > right_count {
			first := elements[middle-left_count : middle-left_count+right_count]
			second := elements[middle : middle+right_count]
			swap_range(first, second)
			left_count -= right_count
		} else {
			first := elements[middle-left_count : middle]
			second_start := middle + right_count - left_count
			swap_range(first, elements[second_start:second_start+left_count])
			right_count -= left_count
		}
	}
	swap_range(elements[middle-left_count:middle], elements[middle:middle+left_count])
}

func swap_range[S ~[]E, E any](left S, right S) {
	for index := range left {
		left[index], right[index] = right[index], left[index]
	}
}
