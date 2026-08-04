// Package slices supplies the Go standard-library slice algorithms through the repository
// naming and assertion boundaries. This package contains each slice algorithm.
// The package also owns the domain names and callback types that first-party callers use.
// Replacement values use a slice instead of a variadic parameter because a raw variadic
// collection cannot carry a repository domain.
package slices

import (
	"cmp"
	"iter"

	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/math/bits"
)

// SLICE_COUNT_MAXIMUM caps each slice that crosses this deterministic package boundary.
// It matches the repository text and buffer boundary, so one common element count applies.
const SLICE_COUNT_MAXIMUM = 4096

// COUNT_MINIMUM is the smallest element count.
const COUNT_MINIMUM = 0

// COUNT_MAXIMUM is the largest admitted element count.
const COUNT_MAXIMUM = SLICE_COUNT_MAXIMUM

// POSITION_MINIMUM is the first valid slice position.
const POSITION_MINIMUM = COUNT_MINIMUM

// POSITION_MAXIMUM is the position after the largest admitted slice.
const POSITION_MAXIMUM = COUNT_MAXIMUM

// SEARCH_POSITION_MINIMUM is the first ordered-search position.
const SEARCH_POSITION_MINIMUM = POSITION_MINIMUM

// SEARCH_POSITION_MAXIMUM is the position after the largest admitted slice.
const SEARCH_POSITION_MAXIMUM = POSITION_MAXIMUM

// INDEX_NOT_FOUND is the search result for an absent value.
const INDEX_NOT_FOUND Found_Index = -1

// FOUND_INDEX_MINIMUM includes the absent search result.
const FOUND_INDEX_MINIMUM = int(INDEX_NOT_FOUND)

// FOUND_INDEX_MAXIMUM is the final index of the largest admitted slice.
const FOUND_INDEX_MAXIMUM = POSITION_MAXIMUM - 1

// COMPARISON_MINIMUM is the smallest result that an injected comparison can return.
const COMPARISON_MINIMUM = int(bits.INTEGER_64_MINIMUM)

// COMPARISON_MAXIMUM is the largest result that an injected comparison can return.
const COMPARISON_MAXIMUM = int(bits.INTEGER_64_MAXIMUM)

// ORDERING_LESS is the normalized result when the left slice precedes the right slice.
const ORDERING_LESS = -1

// ORDERING_EQUAL is the normalized result when both slices are equal.
const ORDERING_EQUAL = 0

// ORDERING_GREATER is the normalized result when the left slice follows the right slice.
const ORDERING_GREATER = 1

// Boolean is a true or false report from a slice query.
type Boolean bool

// Boolean_Invariants states both query results as obligations.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The query result is true.").
		Ensure()
}

// Position is a valid insertion position, including the position after the final element.
type Position int

// Position_Invariants states the complete insertion-position domain.
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Found_Index is the index of a matching element or INDEX_NOT_FOUND.
type Found_Index int

// Found_Index_Invariants states the complete admitted search-result domain.
func Found_Index_Invariants(value Found_Index, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), FOUND_INDEX_MINIMUM, FOUND_INDEX_MAXIMUM).
		Ensure()
}

// Search_Position is an index or insertion position that an ordered search returns.
type Search_Position int

// Search_Position_Invariants states each admitted ordered-search position.
func Search_Position_Invariants(value Search_Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), SEARCH_POSITION_MINIMUM, SEARCH_POSITION_MAXIMUM).
		Ensure()
}

// Count is an element count or an additional-capacity count.
type Count int

// Count_Invariants states the complete nonnegative count domain.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Comparison is negative, zero, or positive as a left value precedes, equals, or follows
// a right value. The magnitude belongs to the injected comparison and remains unchanged.
type Comparison int

// Comparison_Invariants states the complete machine-integer comparison domain.
func Comparison_Invariants(value Comparison, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COMPARISON_MINIMUM, COMPARISON_MAXIMUM).
		Ensure()
}

// Ordering is the normalized result of an ordered-slice comparison.
type Ordering int

// Ordering_Invariants states the three results that Compare can return.
func Ordering_Invariants(value Ordering, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Enum_3_Int(int(value), ORDERING_LESS, ORDERING_EQUAL, ORDERING_GREATER).
		Ensure()
}

// Equality_Function reports whether a left value and a right value are equal.
type Equality_Function[Left any, Right any] func(left Left, right Right) (equal bool)

// Predicate_Function reports whether one value has a selected property.
type Predicate_Function[Element any] func(element Element) (selected bool)

// Comparison_Function compares a left value and a right value.
type Comparison_Function[Left any, Right any] func(
	left Left, right Right,
) (comparison Comparison)

// Enforces the package-wide size boundary at one source root. One generic boundary avoids a
// capacity wrapper that would discard a caller's named slice type.
func enforce_slice[S ~[]E, E any](slice S) {
	invariant.Always(
		len(slice) <= SLICE_COUNT_MAXIMUM,
		"A slice boundary admits at most SLICE_COUNT_MAXIMUM elements.",
	)
}

// Equal reports whether two comparable slices have equal elements in the same positions.
func Equal[S ~[]E, E comparable](left S, right S) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "equal.equal") }()
	enforce_slice(left)
	enforce_slice(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	equal = true
	return equal
}

// Equal_Function reports whether equality accepts each same-position pair.
func Equal_Function[Left_Slice ~[]Left, Right_Slice ~[]Right, Left any, Right any](
	left Left_Slice,
	right Right_Slice,
	equality Equality_Function[Left, Right],
) (equal Boolean) {
	defer func() { Boolean_Invariants(equal, "equal_function.equal") }()
	enforce_slice(left)
	enforce_slice(right)
	if len(left) != len(right) {
		return false
	}
	for index, left_value := range left {
		if !equality(left_value, right[index]) {
			return false
		}
	}
	equal = true
	return equal
}

// Compare orders two ordered slices lexicographically.
func Compare[S ~[]E, E cmp.Ordered](left S, right S) (ordering Ordering) {
	defer func() { Ordering_Invariants(ordering, "compare.ordering") }()
	enforce_slice(left)
	enforce_slice(right)
	for index, left_value := range left {
		if index >= len(right) {
			return ORDERING_GREATER
		}
		if result := cmp.Compare(left_value, right[index]); result != ORDERING_EQUAL {
			return Ordering(result)
		}
	}
	if len(left) < len(right) {
		return ORDERING_LESS
	}
	ordering = ORDERING_EQUAL
	return ordering
}

// Compare_Function orders two slices lexicographically through comparison.
func Compare_Function[Left_Slice ~[]Left, Right_Slice ~[]Right, Left any, Right any](
	left Left_Slice,
	right Right_Slice,
	comparison_function Comparison_Function[Left, Right],
) (comparison Comparison) {
	defer func() { Comparison_Invariants(comparison, "compare_function.comparison") }()
	enforce_slice(left)
	enforce_slice(right)
	for index, left_value := range left {
		if index >= len(right) {
			return ORDERING_GREATER
		}
		result := comparison_function(left_value, right[index])
		if result != ORDERING_EQUAL {
			return result
		}
	}
	if len(left) < len(right) {
		return ORDERING_LESS
	}
	comparison = ORDERING_EQUAL
	return comparison
}

// Index returns the first index that holds value, or INDEX_NOT_FOUND.
func Index[S ~[]E, E comparable](slice S, value E) (index Found_Index) {
	defer func() { Found_Index_Invariants(index, "index.index") }()
	enforce_slice(slice)
	for candidate := range slice {
		if slice[candidate] == value {
			return Found_Index(candidate)
		}
	}
	index = INDEX_NOT_FOUND
	return index
}

// Index_Function returns the first index that predicate accepts, or INDEX_NOT_FOUND.
func Index_Function[S ~[]E, E any](
	slice S, predicate Predicate_Function[E],
) (index Found_Index) {
	defer func() { Found_Index_Invariants(index, "index_function.index") }()
	enforce_slice(slice)
	for candidate := range slice {
		if predicate(slice[candidate]) {
			return Found_Index(candidate)
		}
	}
	index = INDEX_NOT_FOUND
	return index
}

// Contains reports whether slice holds value.
func Contains[S ~[]E, E comparable](slice S, value E) (contains Boolean) {
	defer func() { Boolean_Invariants(contains, "contains.contains") }()
	contains = Index(slice, value) != INDEX_NOT_FOUND
	return contains
}

// Contains_Function reports whether predicate accepts one element.
func Contains_Function[S ~[]E, E any](
	slice S, predicate Predicate_Function[E],
) (contains Boolean) {
	defer func() { Boolean_Invariants(contains, "contains_function.contains") }()
	contains = Index_Function(slice, predicate) != INDEX_NOT_FOUND
	return contains
}

// Insert adds values at index and returns the modified slice.
func Insert[S ~[]E, Values ~[]E, E any](
	slice S, position Position, values Values,
) (result S) {
	Position_Invariants(position, "insert.position")
	enforce_slice(slice)
	enforce_slice(values)
	invariant.Always(
		position <= Position(len(slice)),
		"An insertion position does not exceed the slice length.",
	)
	index := int(position)
	value_count := len(values)
	if value_count == 0 {
		return slice
	}
	slice_count := len(slice)
	if index == slice_count {
		result = append(slice, values...)
		enforce_slice(result)
		return result
	}
	if slice_count+value_count > cap(slice) {
		result = append(slice[:index], make(S, slice_count+value_count-index)...)
		copy(result[index:], values)
		copy(result[index+value_count:], slice[index:])
		enforce_slice(result)
		return result
	}
	result = slice[:slice_count+value_count]
	if !slices_overlap(values, result[index+value_count:]) {
		copy(result[index+value_count:], result[index:slice_count])
		copy(result[index:], values)
		enforce_slice(result)
		return result
	}
	copy(result[slice_count:], values)
	rotate_right(result[index:], result[slice_count:])
	enforce_slice(result)
	return result
}

// Delete removes the half-open range from start through end.
func Delete[S ~[]E, E any](slice S, start Position, end Position) (result S) {
	Position_Invariants(start, "delete.start")
	Position_Invariants(end, "delete.end")
	enforce_slice(slice)
	invariant.Always(start <= end, "A deletion start does not follow its end.")
	invariant.Always(
		end <= Position(len(slice)),
		"A deletion end does not exceed the slice length.",
	)
	start_index := int(start)
	end_index := int(end)
	if start_index == end_index {
		return slice
	}
	old_count := len(slice)
	result = append(slice[:start_index], slice[end_index:]...)
	clear(slice[len(result):old_count])
	enforce_slice(result)
	return result
}

// Delete_Function removes each element that predicate accepts.
func Delete_Function[S ~[]E, E any](
	slice S, predicate Predicate_Function[E],
) (result S) {
	enforce_slice(slice)
	first := Index_Function(slice, predicate)
	if first == INDEX_NOT_FOUND {
		return slice
	}
	write_index := int(first)
	for read_index := write_index + 1; read_index < len(slice); read_index++ {
		value := slice[read_index]
		if !predicate(value) {
			slice[write_index] = value
			write_index++
		}
	}
	clear(slice[write_index:])
	result = slice[:write_index]
	enforce_slice(result)
	return result
}

// Replace exchanges the half-open range from start through end for values.
func Replace[S ~[]E, Values ~[]E, E any](
	slice S, start Position, end Position, values Values,
) (result S) {
	Position_Invariants(start, "replace.start")
	Position_Invariants(end, "replace.end")
	enforce_slice(slice)
	enforce_slice(values)
	invariant.Always(start <= end, "A replacement start does not follow its end.")
	invariant.Always(
		end <= Position(len(slice)),
		"A replacement end does not exceed the slice length.",
	)
	start_index := int(start)
	end_index := int(end)
	if start_index == end_index {
		return Insert(slice, start, values)
	}
	if end_index == len(slice) {
		result = append(slice[:start_index], values...)
		if len(result) < len(slice) {
			clear(slice[len(result):])
		}
		enforce_slice(result)
		return result
	}
	result_count := start_index + len(values) + len(slice[end_index:])
	if result_count > cap(slice) {
		remainder_count := len(values) + len(slice[end_index:])
		result = append(slice[:start_index], make(S, remainder_count)...)
		copy(result[start_index:], values)
		copy(result[start_index+len(values):], slice[end_index:])
		enforce_slice(result)
		return result
	}
	result = slice[:result_count]
	if start_index+len(values) <= end_index {
		copy(result[start_index:], values)
		copy(result[start_index+len(values):], slice[end_index:])
		clear(slice[result_count:])
		enforce_slice(result)
		return result
	}
	result = replace_expansion(
		result,
		slice,
		slice[:start_index],
		slice[end_index:],
		values,
	)
	enforce_slice(result)
	return result
}

// Clone returns a shallow copy and preserves a nil input.
func Clone[S ~[]E, E any](slice S) (clone S) {
	enforce_slice(slice)
	if slice == nil {
		return nil
	}
	clone = append(S{}, slice...)
	enforce_slice(clone)
	return clone
}

// Compact keeps one element from each consecutive run of equal elements.
func Compact[S ~[]E, E comparable](slice S) (result S) {
	enforce_slice(slice)
	if len(slice) < 2 {
		return slice
	}
	for index := 1; index < len(slice); index++ {
		if slice[index] == slice[index-1] {
			return compact_tail(slice, slice[index:])
		}
	}
	result = slice
	enforce_slice(result)
	return result
}

// Compact_Function keeps the first element from each run that equality joins.
func Compact_Function[S ~[]E, E any](
	slice S, equality Equality_Function[E, E],
) (result S) {
	enforce_slice(slice)
	if len(slice) < 2 {
		return slice
	}
	for index := 1; index < len(slice); index++ {
		if equality(slice[index], slice[index-1]) {
			return compact_function_tail(slice, slice[index:], equality)
		}
	}
	result = slice
	enforce_slice(result)
	return result
}

// Grow reserves capacity for count additional elements.
func Grow[S ~[]E, E any](slice S, count Count) (grown S) {
	Count_Invariants(count, "grow.count")
	enforce_slice(slice)
	additional_count := int(count) - (cap(slice) - len(slice))
	if additional_count > 0 {
		grown = append(slice[:cap(slice)], make([]E, additional_count)...)[:len(slice)]
	} else {
		grown = slice
	}
	enforce_slice(grown)
	return grown
}

// Clip removes unused capacity from slice.
func Clip[S ~[]E, E any](slice S) (clipped S) {
	enforce_slice(slice)
	clipped = slice[:len(slice):len(slice)]
	enforce_slice(clipped)
	return clipped
}

// Reverse reverses slice in place.
func Reverse[S ~[]E, E any](slice S) {
	enforce_slice(slice)
	for left, right := 0, len(slice)-1; left < right; left, right = left+1, right-1 {
		slice[left], slice[right] = slice[right], slice[left]
	}
}

// The expansion needs separate copy orders for each possible overlap shape.
func replace_expansion[S ~[]E, Values ~[]E, E any](
	result S,
	original S,
	prefix S,
	suffix S,
	values Values,
) (expanded S) {
	prefix_count := len(prefix)
	replaced_prefix_count := len(original) - len(suffix)
	if !slices_overlap(result[prefix_count+len(values):], values) {
		copy(result[prefix_count+len(values):], suffix)
		copy(result[prefix_count:], values)
		return result
	}
	overflow_count := len(values) - (replaced_prefix_count - prefix_count)
	if !slices_overlap(result[prefix_count:replaced_prefix_count], values) {
		copy(result[prefix_count:replaced_prefix_count], values[overflow_count:])
		copy(result[len(original):], values[:overflow_count])
		rotate_right(result[prefix_count:], result[len(original):])
		return result
	}
	if !slices_overlap(result[len(original):], values) {
		copy(result[len(original):], values[:overflow_count])
		copy(result[prefix_count:replaced_prefix_count], values[overflow_count:])
		rotate_right(result[prefix_count:], result[len(original):])
		return result
	}
	value_suffix := overlap_suffix(values, suffix)
	tail_start := len(values) - len(value_suffix)
	copy(result[prefix_count:], values)
	copy(result[prefix_count+len(values):], result[prefix_count+tail_start:])
	return result
}

func compact_tail[S ~[]E, E comparable](slice S, tail S) (result S) {
	write_index := len(slice) - len(tail)
	for read_index := 1; read_index < len(tail); read_index++ {
		if tail[read_index] != tail[read_index-1] {
			slice[write_index] = tail[read_index]
			write_index++
		}
	}
	clear(slice[write_index:])
	return slice[:write_index]
}

func compact_function_tail[S ~[]E, E any](
	slice S,
	tail S,
	equality Equality_Function[E, E],
) (result S) {
	write_index := len(slice) - len(tail)
	for read_index := 1; read_index < len(tail); read_index++ {
		if !equality(tail[read_index], tail[read_index-1]) {
			slice[write_index] = tail[read_index]
			write_index++
		}
	}
	clear(slice[write_index:])
	return slice[:write_index]
}

// Element addresses identify overlap without an unsafe package dependency.
func slices_overlap[Left ~[]E, Right ~[]E, E any](
	left Left,
	right Right,
) (overlap Boolean) {
	defer func() { Boolean_Invariants(overlap, "slices_overlap.overlap") }()
	for left_index := range left {
		for right_index := range right {
			if &left[left_index] == &right[right_index] {
				return true
			}
		}
	}
	return false
}

func overlap_suffix[Haystack ~[]E, Needle ~[]E, E any](
	haystack Haystack,
	needle Needle,
) (suffix Haystack) {
	for candidate := range haystack {
		if &haystack[candidate] == &needle[0] {
			return haystack[candidate:]
		}
	}
	panic("The overlapping slice has no start index.")
}

func rotate_right[S ~[]E, E any](slice S, moved S) {
	split := len(slice) - len(moved)
	reverse_elements(slice[:split])
	reverse_elements(slice[split:])
	reverse_elements(slice)
}

func reverse_elements[S ~[]E, E any](slice S) {
	for left, right := 0, len(slice)-1; left < right; left, right = left+1, right-1 {
		slice[left], slice[right] = slice[right], slice[left]
	}
}

// Concatenate returns a new slice that joins the supplied slices.
func Concatenate[Slice_List ~[]S, S ~[]E, E any](slices Slice_List) (result S) {
	enforce_slice(slices)
	total_count := 0
	for _, slice := range slices {
		enforce_slice(slice)
		invariant.Always(
			len(slice) <= SLICE_COUNT_MAXIMUM-total_count,
			"A concatenation result admits at most SLICE_COUNT_MAXIMUM elements.",
		)
		total_count += len(slice)
	}
	result = Grow[S](nil, Count(total_count))
	for _, slice := range slices {
		result = append(result, slice...)
	}
	enforce_slice(result)
	return result
}

// Repeat returns a new slice that contains count copies of slice.
func Repeat[S ~[]E, E any](slice S, count Count) (result S) {
	Count_Invariants(count, "repeat.count")
	enforce_slice(slice)
	if len(slice) > 0 {
		invariant.Always(
			int(count) <= SLICE_COUNT_MAXIMUM/len(slice),
			"A repeat result admits at most SLICE_COUNT_MAXIMUM elements.",
		)
	}
	result_count := len(slice) * int(count)
	result = make(S, result_count)
	copied := copy(result, slice)
	for copied < len(result) {
		copied += copy(result[copied:], result[:copied])
	}
	enforce_slice(result)
	return result
}

// Sort orders slice in ascending order.
func Sort[S ~[]E, E cmp.Ordered](slice S) {
	enforce_slice(slice)
	comparison := func(left E, right E) (ordering Comparison) {
		return Comparison(cmp.Compare(left, right))
	}
	heap_sort(slice, comparison)
}

// Sort_Function orders slice through comparison.
func Sort_Function[S ~[]E, E any](
	slice S, comparison Comparison_Function[E, E],
) {
	enforce_slice(slice)
	heap_sort(slice, comparison)
}

// Sort_Stable_Function orders slice and preserves the input order of equal elements.
func Sort_Stable_Function[S ~[]E, E any](
	slice S, comparison Comparison_Function[E, E],
) {
	enforce_slice(slice)
	stable_merge_sort(slice, comparison)
}

func heap_sort[S ~[]E, E any](slice S, comparison Comparison_Function[E, E]) {
	for root := len(slice)/2 - 1; root >= 0; root-- {
		sift_down(slice, slice[root:], comparison)
	}
	for end := len(slice) - 1; end > 0; end-- {
		slice[0], slice[end] = slice[end], slice[0]
		heap := slice[:end]
		sift_down(heap, heap, comparison)
	}
}

func sift_down[S ~[]E, E any](
	heap S,
	root_tail S,
	comparison Comparison_Function[E, E],
) {
	root := len(heap) - len(root_tail)
	end_count := len(heap)
	for child := root*2 + 1; child < end_count; child = root*2 + 1 {
		if child+1 < end_count {
			if comparison(heap[child], heap[child+1]) < ORDERING_EQUAL {
				child++
			}
		}
		if comparison(heap[root], heap[child]) >= ORDERING_EQUAL {
			return
		}
		heap[root], heap[child] = heap[child], heap[root]
		root = child
	}
}

// One buffer keeps stable sorting at O(n log n) comparisons and movements.
func stable_merge_sort[S ~[]E, E any](
	slice S,
	comparison Comparison_Function[E, E],
) {
	if len(slice) < 2 {
		return
	}
	buffer := make([]E, len(slice))
	current_is_slice := true
	for width := 1; width < len(slice); width *= 2 {
		if current_is_slice {
			merge_pass(slice, buffer, slice[:width], comparison)
		} else {
			merge_pass(buffer, slice, buffer[:width], comparison)
		}
		current_is_slice = !current_is_slice
	}
	if !current_is_slice {
		copy(slice, buffer)
	}
}

func merge_pass[Source ~[]E, Destination ~[]E, Width ~[]E, E any](
	source Source,
	destination Destination,
	width_slice Width,
	comparison Comparison_Function[E, E],
) {
	width_count := len(width_slice)
	for start := 0; start < len(source); start += width_count * 2 {
		middle := min(start+width_count, len(source))
		end := min(start+width_count*2, len(source))
		left := start
		right := middle
		for output := start; output < end; output++ {
			if left < middle {
				if right >= end {
					destination[output] = source[left]
					left++
					continue
				}
				if comparison(source[right], source[left]) >= ORDERING_EQUAL {
					destination[output] = source[left]
					left++
					continue
				}
			}
			destination[output] = source[right]
			right++
		}
	}
}

// Is_Sorted reports whether slice has ascending order.
func Is_Sorted[S ~[]E, E cmp.Ordered](slice S) (sorted Boolean) {
	defer func() { Boolean_Invariants(sorted, "is_sorted.sorted") }()
	enforce_slice(slice)
	for index := len(slice) - 1; index > 0; index-- {
		if cmp.Less(slice[index], slice[index-1]) {
			return false
		}
	}
	sorted = true
	return sorted
}

// Is_Sorted_Function reports whether slice has the order that comparison defines.
func Is_Sorted_Function[S ~[]E, E any](
	slice S, comparison Comparison_Function[E, E],
) (sorted Boolean) {
	defer func() { Boolean_Invariants(sorted, "is_sorted_function.sorted") }()
	enforce_slice(slice)
	for index := len(slice) - 1; index > 0; index-- {
		if comparison(slice[index], slice[index-1]) < ORDERING_EQUAL {
			return false
		}
	}
	sorted = true
	return sorted
}

// Minimum returns the least ordered element and panics for an empty slice.
func Minimum[S ~[]E, E cmp.Ordered](slice S) (minimum E) {
	enforce_slice(slice)
	if len(slice) < 1 {
		panic("Minimum cannot read an empty slice.")
	}
	minimum = slice[0]
	for index := 1; index < len(slice); index++ {
		minimum = min(minimum, slice[index])
	}
	return minimum
}

// Minimum_Function returns the first least element that comparison identifies.
func Minimum_Function[S ~[]E, E any](
	slice S, comparison Comparison_Function[E, E],
) (minimum E) {
	enforce_slice(slice)
	if len(slice) < 1 {
		panic("Minimum_Function cannot read an empty slice.")
	}
	minimum = slice[0]
	for index := 1; index < len(slice); index++ {
		if comparison(slice[index], minimum) < ORDERING_EQUAL {
			minimum = slice[index]
		}
	}
	return minimum
}

// Maximum returns the greatest ordered element and panics for an empty slice.
func Maximum[S ~[]E, E cmp.Ordered](slice S) (maximum E) {
	enforce_slice(slice)
	if len(slice) < 1 {
		panic("Maximum cannot read an empty slice.")
	}
	maximum = slice[0]
	for index := 1; index < len(slice); index++ {
		maximum = max(maximum, slice[index])
	}
	return maximum
}

// Maximum_Function returns the first greatest element that comparison identifies.
func Maximum_Function[S ~[]E, E any](
	slice S, comparison Comparison_Function[E, E],
) (maximum E) {
	enforce_slice(slice)
	if len(slice) < 1 {
		panic("Maximum_Function cannot read an empty slice.")
	}
	maximum = slice[0]
	for index := 1; index < len(slice); index++ {
		if comparison(slice[index], maximum) > ORDERING_EQUAL {
			maximum = slice[index]
		}
	}
	return maximum
}

// Binary_Search returns the first target index or the target insertion index.
func Binary_Search[S ~[]E, E cmp.Ordered](
	slice S, target E,
) (position Search_Position, found Boolean) {
	defer func() {
		Search_Position_Invariants(position, "binary_search.position")
		Boolean_Invariants(found, "binary_search.found")
	}()
	enforce_slice(slice)
	left, right := 0, len(slice)
	for left < right {
		middle := int(uint(left+right) >> 1)
		if cmp.Less(slice[middle], target) {
			left = middle + 1
		} else {
			right = middle
		}
	}
	position = Search_Position(left)
	if left < len(slice) {
		if slice[left] == target {
			found = true
		} else if is_not_a_number(slice[left]) {
			if is_not_a_number(target) {
				found = true
			}
		}
	}
	return position, found
}

// Binary_Search_Function returns the first target index or its insertion index.
func Binary_Search_Function[S ~[]E, E any, Target any](
	slice S,
	target Target,
	comparison Comparison_Function[E, Target],
) (position Search_Position, found Boolean) {
	defer func() {
		Search_Position_Invariants(position, "binary_search_function.position")
		Boolean_Invariants(found, "binary_search_function.found")
	}()
	enforce_slice(slice)
	left, right := 0, len(slice)
	for left < right {
		middle := int(uint(left+right) >> 1)
		if comparison(slice[middle], target) < ORDERING_EQUAL {
			left = middle + 1
		} else {
			right = middle
		}
	}
	position = Search_Position(left)
	if left < len(slice) {
		found = comparison(slice[left], target) == ORDERING_EQUAL
	}
	return position, found
}

func is_not_a_number[Value cmp.Ordered](value Value) (result Boolean) {
	defer func() { Boolean_Invariants(result, "is_not_a_number.result") }()
	return value != value
}

// All returns an iterator that yields each index and value in forward order.
func All[S ~[]E, E any](slice S) (sequence iter.Seq2[Search_Position, E]) {
	enforce_slice(slice)
	sequence = func(yield func(Search_Position, E) (continued bool)) {
		for index, value := range slice {
			if !yield(Search_Position(index), value) {
				return
			}
		}
	}
	return sequence
}

// Backward returns an iterator that yields each index and value in reverse order.
func Backward[S ~[]E, E any](slice S) (sequence iter.Seq2[Search_Position, E]) {
	enforce_slice(slice)
	sequence = func(yield func(Search_Position, E) (continued bool)) {
		for index := len(slice) - 1; index >= 0; index-- {
			if !yield(Search_Position(index), slice[index]) {
				return
			}
		}
	}
	return sequence
}

// Values returns an iterator that yields each element in forward order.
func Values[S ~[]E, E any](slice S) (sequence iter.Seq[E]) {
	enforce_slice(slice)
	sequence = func(yield func(E) (continued bool)) {
		for _, value := range slice {
			if !yield(value) {
				return
			}
		}
	}
	return sequence
}

// Append_Sequence appends each yielded element to slice.
func Append_Sequence[S ~[]E, E any](slice S, sequence iter.Seq[E]) (result S) {
	enforce_slice(slice)
	result = slice
	for value := range sequence {
		invariant.Always(
			len(result) < SLICE_COUNT_MAXIMUM,
			"Append_Sequence admits at most SLICE_COUNT_MAXIMUM result elements.",
		)
		result = append(result, value)
	}
	enforce_slice(result)
	return result
}

// Collect returns a new slice that contains each yielded element.
func Collect[S ~[]E, E any](sequence iter.Seq[E]) (result S) {
	for value := range sequence {
		invariant.Always(
			len(result) < SLICE_COUNT_MAXIMUM,
			"Collect admits at most SLICE_COUNT_MAXIMUM result elements.",
		)
		result = append(result, value)
	}
	enforce_slice(result)
	return result
}

// Sorted collects and sorts the yielded elements in ascending order.
func Sorted[S ~[]E, E cmp.Ordered](sequence iter.Seq[E]) (result S) {
	result = Collect[S](sequence)
	Sort(result)
	return result
}

// Sorted_Function collects and sorts the yielded elements through comparison.
func Sorted_Function[S ~[]E, E any](
	sequence iter.Seq[E], comparison Comparison_Function[E, E],
) (result S) {
	result = Collect[S](sequence)
	Sort_Function(result, comparison)
	return result
}

// Sorted_Stable_Function collects and stably sorts the yielded elements through comparison.
func Sorted_Stable_Function[S ~[]E, E any](
	sequence iter.Seq[E], comparison Comparison_Function[E, E],
) (result S) {
	result = Collect[S](sequence)
	Sort_Stable_Function(result, comparison)
	return result
}

// Chunk returns an iterator over clipped consecutive slices of at most count elements.
func Chunk[S ~[]E, E any](slice S, count Count) (sequence iter.Seq[S]) {
	Count_Invariants(count, "chunk.count")
	enforce_slice(slice)
	invariant.Always(count >= 1, "A chunk count is at least one.")
	sequence = func(yield func(S) (continued bool)) {
		chunk_count := int(count)
		for start := 0; start < len(slice); start += chunk_count {
			end := start + min(chunk_count, len(slice[start:]))
			if !yield(slice[start:end:end]) {
				return
			}
		}
	}
	return sequence
}
