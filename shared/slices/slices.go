// Package slices supplies bounded zero-allocation algorithms. Callers own every output buffer.
package slices

import (
	"cmp"
	"unsafe"

	"local/james-orcales/shared/math/bits"
	"local/james-orcales/shared/sim/aver/default"
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
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The query result is true.").
		Ensure()
}

// Position is a valid insertion position, including the position after the final element.
type Position int

// Position_Invariants states the complete insertion-position domain.
func Position_Invariants(value Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Found_Index is the index of a matching element or INDEX_NOT_FOUND.
type Found_Index int

// Found_Index_Invariants states the complete admitted search-result domain.
func Found_Index_Invariants(value Found_Index, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), FOUND_INDEX_MINIMUM, FOUND_INDEX_MAXIMUM).
		Ensure()
}

// Search_Position is an index or insertion position that an ordered search returns.
type Search_Position int

// Search_Position_Invariants states each admitted ordered-search position.
func Search_Position_Invariants(value Search_Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), SEARCH_POSITION_MINIMUM, SEARCH_POSITION_MAXIMUM).
		Ensure()
}

// Count is an element count or an additional-capacity count.
type Count int

// Count_Invariants states the complete nonnegative count domain.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Comparison is negative, zero, or positive as a left value precedes, equals, or follows
// a right value. The magnitude belongs to the injected comparison and remains unchanged.
type Comparison int

// Comparison_Invariants states the complete machine-integer comparison domain.
func Comparison_Invariants(value Comparison, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COMPARISON_MINIMUM, COMPARISON_MAXIMUM).
		Ensure()
}

// Ordering is the normalized result of an ordered-slice comparison.
type Ordering int

// Ordering_Invariants states the three results that Compare can return.
func Ordering_Invariants(value Ordering, namespace aver.Namespace) {
	aver.Tree(value, namespace).
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

// Indexed_Yield_Function receives one position and value until it rejects continuation.
type Indexed_Yield_Function[Element any] func(
	position Search_Position, element Element,
) (continued Boolean)

// Yield_Function receives one value until it rejects continuation.
type Yield_Function[Element any] func(element Element) (continued Boolean)

// Chunk_Yield_Function receives one clipped source view until it rejects continuation.
type Chunk_Yield_Function[Slice ~[]Element, Element any] func(
	chunk Slice,
) (continued Boolean)

// Enforces the package-wide size boundary at one source root. One generic boundary avoids a
// capacity wrapper that would discard a caller's named slice type.
func enforce_slice[S ~[]E, E any](slice S) {
	aver.Always(
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

// Insert_Into writes source with values inserted at position into caller storage.
func Insert_Into[
	Destination ~[]E, Source ~[]E, Values ~[]E, E any,
](
	destination Destination, source Source, position Position, values Values,
) (count Count) {
	defer func() { Count_Invariants(count, "insert_into.count") }()
	Position_Invariants(position, "insert_into.position")
	enforce_slice(destination)
	enforce_slice(source)
	enforce_slice(values)
	aver.Always(
		position <= Position(len(source)),
		"An insertion position does not exceed source count.",
	)
	result_count := len(source) + len(values)
	aver.Always(
		result_count <= SLICE_COUNT_MAXIMUM,
		"An insertion result admits at most SLICE_COUNT_MAXIMUM elements.",
	)
	result, original, same_storage := prepare_result(
		destination, source, Count(result_count),
	)
	index := int(position)
	if len(values) == 0 {
		return Count(result_count)
	}
	if !same_storage {
		aver.Always(
			!slices_overlap(result, values),
			"Separate insertion output does not overlap values.",
		)
	}
	if index == len(original) {
		copy(result[index:], values)
		return Count(result_count)
	}
	if !slices_overlap(values, result[index+len(values):]) {
		copy(result[index+len(values):], original[index:])
		copy(result[index:], values)
		return Count(result_count)
	}
	copy(result[len(original):], values)
	rotate_right(result[index:], result[len(original):])
	return Count(result_count)
}

// Delete_Into writes source without half-open range into caller storage.
func Delete_Into[Destination ~[]E, Source ~[]E, E any](
	destination Destination, source Source, start Position, end Position,
) (count Count) {
	defer func() { Count_Invariants(count, "delete_into.count") }()
	Position_Invariants(start, "delete_into.start")
	Position_Invariants(end, "delete_into.end")
	enforce_slice(destination)
	enforce_slice(source)
	aver.Always(start <= end, "A deletion start does not follow its end.")
	aver.Always(
		end <= Position(len(source)),
		"A deletion end does not exceed source count.",
	)
	result_count := len(source) - (int(end) - int(start))
	result, original, _ := prepare_result(destination, source, Count(result_count))
	copy(result[int(start):], original[int(end):])
	clear(destination[result_count:len(source)])
	return Count(result_count)
}

// Delete_Function_Into writes elements that predicate rejects into caller storage.
func Delete_Function_Into[Destination ~[]E, Source ~[]E, E any](
	destination Destination, source Source, predicate Predicate_Function[E],
) (count Count) {
	defer func() { Count_Invariants(count, "delete_function_into.count") }()
	enforce_slice(destination)
	enforce_slice(source)
	result, original, _ := prepare_result(destination, source, Count(len(source)))
	write_index := 0
	for _, value := range original {
		if !predicate(value) {
			result[write_index] = value
			write_index++
		}
	}
	clear(result[write_index:])
	return Count(write_index)
}

// Replace_Into exchanges source range for values inside caller storage.
func Replace_Into[
	Destination ~[]E, Source ~[]E, Values ~[]E, E any,
](
	destination Destination, source Source, start Position, end Position, values Values,
) (count Count) {
	defer func() { Count_Invariants(count, "replace_into.count") }()
	Position_Invariants(start, "replace_into.start")
	Position_Invariants(end, "replace_into.end")
	enforce_slice(destination)
	enforce_slice(source)
	enforce_slice(values)
	aver.Always(start <= end, "A replacement start does not follow its end.")
	aver.Always(
		end <= Position(len(source)),
		"A replacement end does not exceed source count.",
	)
	if start == end {
		return Insert_Into(destination, source, start, values)
	}
	result_count := int(start) + len(values) + len(source[int(end):])
	aver.Always(
		result_count <= SLICE_COUNT_MAXIMUM,
		"A replacement result admits at most SLICE_COUNT_MAXIMUM elements.",
	)
	result, original, same_storage := prepare_result(
		destination, source, Count(result_count),
	)
	if !same_storage {
		aver.Always(
			!slices_overlap(result, values),
			"Separate replacement output does not overlap values.",
		)
	}
	start_index := int(start)
	end_index := int(end)
	if end_index == len(original) {
		copy(result[start_index:], values)
		clear(destination[result_count:len(original)])
		return Count(result_count)
	}
	if start_index+len(values) <= end_index {
		copy(result[start_index:], values)
		copy(result[start_index+len(values):], original[end_index:])
		clear(destination[result_count:len(original)])
		return Count(result_count)
	}
	replace_expansion(
		result,
		original,
		original[:start_index],
		original[end_index:],
		values,
	)
	return Count(result_count)
}

// Clone_Into copies source into caller storage.
func Clone_Into[Destination ~[]E, Source ~[]E, E any](
	destination Destination, source Source,
) (count Count) {
	defer func() { Count_Invariants(count, "clone_into.count") }()
	enforce_slice(destination)
	enforce_slice(source)
	aver.Always(
		len(source) <= len(destination),
		"Clone destination holds every source element.",
	)
	copy(destination, source)
	return Count(len(source))
}

// Compact_Into writes first element from each consecutive equal run into caller storage.
func Compact_Into[Destination ~[]E, Source ~[]E, E comparable](
	destination Destination, source Source,
) (count Count) {
	defer func() { Count_Invariants(count, "compact_into.count") }()
	enforce_slice(destination)
	enforce_slice(source)
	result, original, _ := prepare_result(destination, source, Count(len(source)))
	if len(original) == 0 {
		return 0
	}
	write_index := 1
	for read_index := 1; read_index < len(original); read_index++ {
		if original[read_index] != original[read_index-1] {
			result[write_index] = original[read_index]
			write_index++
		}
	}
	clear(result[write_index:])
	return Count(write_index)
}

// Compact_Function_Into writes first element from each equality-joined run.
func Compact_Function_Into[Destination ~[]E, Source ~[]E, E any](
	destination Destination, source Source, equality Equality_Function[E, E],
) (count Count) {
	defer func() { Count_Invariants(count, "compact_function_into.count") }()
	enforce_slice(destination)
	enforce_slice(source)
	result, original, _ := prepare_result(destination, source, Count(len(source)))
	if len(original) == 0 {
		return 0
	}
	write_index := 1
	for read_index := 1; read_index < len(original); read_index++ {
		if !equality(original[read_index], original[read_index-1]) {
			result[write_index] = original[read_index]
			write_index++
		}
	}
	clear(result[write_index:])
	return Count(write_index)
}

// Grow_Into copies source into storage that reserves count additional slots.
func Grow_Into[Destination ~[]E, Source ~[]E, E any](
	destination Destination, source Source, count Count,
) (source_count Count) {
	defer func() { Count_Invariants(source_count, "grow_into.source_count") }()
	Count_Invariants(count, "grow_into.count")
	enforce_slice(destination)
	enforce_slice(source)
	aver.Always(
		len(source)+int(count) <= len(destination),
		"Grow destination reserves requested additional slots.",
	)
	copy(destination, source)
	return Count(len(source))
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

// Separate storage must not destroy source before copy. Same-start storage enables in-place edits.
func prepare_result[Destination ~[]E, Source ~[]E, E any](
	destination Destination, source Source, result_count Count,
) (result Destination, original Destination, same_storage Boolean) {
	defer func() { Boolean_Invariants(same_storage, "prepare_result.same_storage") }()
	Count_Invariants(result_count, "prepare_result.result_count")
	required_count := max(int(result_count), len(source))
	aver.Always(
		required_count <= len(destination),
		"Destination holds source and every result element.",
	)
	result = destination[:int(result_count)]
	if len(source) == 0 {
		return result, destination[:0], false
	}
	same_storage = Boolean(&destination[0] == &source[0])
	if same_storage {
		return result, destination[:len(source)], true
	}
	aver.Always(
		!slices_overlap(result, source),
		"Separate destination does not overlap source.",
	)
	copy(destination, source)
	return result, destination[:len(source)], false
}

// The expansion needs separate copy orders for each possible overlap shape.
func replace_expansion[S ~[]E, Values ~[]E, E any](
	result S,
	original S,
	prefix S,
	suffix S,
	values Values,
) {
	prefix_count := len(prefix)
	replaced_prefix_count := len(original) - len(suffix)
	if !slices_overlap(result[prefix_count+len(values):], values) {
		copy(result[prefix_count+len(values):], suffix)
		copy(result[prefix_count:], values)
		return
	}
	overflow_count := len(values) - (replaced_prefix_count - prefix_count)
	if !slices_overlap(result[prefix_count:replaced_prefix_count], values) {
		copy(result[prefix_count:replaced_prefix_count], values[overflow_count:])
		copy(result[len(original):], values[:overflow_count])
		rotate_right(result[prefix_count:], result[len(original):])
		return
	}
	if !slices_overlap(result[len(original):], values) {
		copy(result[len(original):], values[:overflow_count])
		copy(result[prefix_count:replaced_prefix_count], values[overflow_count:])
		rotate_right(result[prefix_count:], result[len(original):])
		return
	}
	value_suffix := overlap_suffix(values, suffix)
	tail_start := len(values) - len(value_suffix)
	copy(result[prefix_count:], values)
	copy(result[prefix_count+len(values):], result[prefix_count+tail_start:])
}

// Zero-size elements own no bytes, so their permitted shared addresses never prove overlap.
func slices_overlap[Left ~[]E, Right ~[]E, E any](
	left Left,
	right Right,
) (overlap Boolean) {
	defer func() { Boolean_Invariants(overlap, "slices_overlap.overlap") }()
	var zero E
	if unsafe.Sizeof(zero) == 0 {
		return false
	}
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
			suffix = haystack[candidate:]
			break
		}
	}
	aver.Always(len(suffix) > 0, "The overlapping slice has a start index.")
	return suffix
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

// Concatenate_Into joins supplied slices inside separate caller storage.
func Concatenate_Into[
	Destination ~[]E, Slice_List ~[]S, S ~[]E, E any,
](destination Destination, slices Slice_List) (count Count) {
	defer func() { Count_Invariants(count, "concatenate_into.count") }()
	enforce_slice(destination)
	enforce_slice(slices)
	total_count := 0
	for _, slice := range slices {
		enforce_slice(slice)
		aver.Always(
			len(slice) <= SLICE_COUNT_MAXIMUM-total_count,
			"A concatenation result admits at most SLICE_COUNT_MAXIMUM elements.",
		)
		total_count += len(slice)
	}
	aver.Always(
		total_count <= len(destination),
		"Concatenation destination holds every result element.",
	)
	for _, slice := range slices {
		aver.Always(
			!slices_overlap(destination[:total_count], slice),
			"Concatenation destination does not overlap source.",
		)
	}
	written := 0
	for _, slice := range slices {
		written += copy(destination[written:], slice)
	}
	return Count(total_count)
}

// Repeat_Into writes count source copies into caller storage.
func Repeat_Into[Destination ~[]E, Source ~[]E, E any](
	destination Destination, source Source, count Count,
) (result_count Count) {
	defer func() { Count_Invariants(result_count, "repeat_into.result_count") }()
	Count_Invariants(count, "repeat_into.count")
	enforce_slice(destination)
	enforce_slice(source)
	if len(source) > 0 {
		aver.Always(
			int(count) <= SLICE_COUNT_MAXIMUM/len(source),
			"A repeat result admits at most SLICE_COUNT_MAXIMUM elements.",
		)
	}
	written_count := len(source) * int(count)
	aver.Always(
		written_count <= len(destination),
		"Repeat destination holds every result element.",
	)
	if written_count == 0 {
		return 0
	}
	copied := copy(destination[:written_count], source)
	for copied < written_count {
		copied += copy(destination[copied:written_count], destination[:copied])
	}
	return Count(written_count)
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

// Strict movement preserves input order for comparison-equal elements.
func stable_merge_sort[S ~[]E, E any](
	slice S,
	comparison Comparison_Function[E, E],
) {
	for index := 1; index < len(slice); index++ {
		value := slice[index]
		position := index
		for position > 0 {
			if comparison(value, slice[position-1]) >= ORDERING_EQUAL {
				break
			}
			slice[position] = slice[position-1]
			position--
		}
		slice[position] = value
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
	aver.Always(len(slice) > 0, "Minimum cannot read an empty slice.")
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
	aver.Always(len(slice) > 0, "Minimum_Function cannot read an empty slice.")
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
	aver.Always(len(slice) > 0, "Maximum cannot read an empty slice.")
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
	aver.Always(len(slice) > 0, "Maximum_Function cannot read an empty slice.")
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
		found = slice[left] == target
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

// All synchronously yields each index and value in forward order.
func All[S ~[]E, E any](
	slice S, yield Indexed_Yield_Function[E],
) (count Count) {
	defer func() { Count_Invariants(count, "all.count") }()
	enforce_slice(slice)
	for index, value := range slice {
		continued := yield(Search_Position(index), value)
		Boolean_Invariants(continued, "all.continued")
		count++
		if !continued {
			return count
		}
	}
	return count
}

// Backward synchronously yields each index and value in reverse order.
func Backward[S ~[]E, E any](
	slice S, yield Indexed_Yield_Function[E],
) (count Count) {
	defer func() { Count_Invariants(count, "backward.count") }()
	enforce_slice(slice)
	for index := len(slice) - 1; index >= 0; index-- {
		continued := yield(Search_Position(index), slice[index])
		Boolean_Invariants(continued, "backward.continued")
		count++
		if !continued {
			return count
		}
	}
	return count
}

// Values synchronously yields each element in forward order.
func Values[S ~[]E, E any](slice S, yield Yield_Function[E]) (count Count) {
	defer func() { Count_Invariants(count, "values.count") }()
	enforce_slice(slice)
	for _, value := range slice {
		continued := yield(value)
		Boolean_Invariants(continued, "values.continued")
		count++
		if !continued {
			return count
		}
	}
	return count
}

// Append_Sequence_Into writes prefix followed by sequence into caller storage.
func Append_Sequence_Into[
	Destination ~[]E, Prefix ~[]E, Sequence ~[]E, E any,
](destination Destination, prefix Prefix, sequence Sequence) (count Count) {
	defer func() { Count_Invariants(count, "append_sequence_into.count") }()
	return Insert_Into(destination, prefix, Position(len(prefix)), sequence)
}

// Collect_Into copies sequence into caller storage.
func Collect_Into[Destination ~[]E, Sequence ~[]E, E any](
	destination Destination, sequence Sequence,
) (count Count) {
	defer func() { Count_Invariants(count, "collect_into.count") }()
	return Clone_Into(destination, sequence)
}

// Sorted_Into copies sequence into caller storage and sorts it ascending.
func Sorted_Into[Destination ~[]E, Sequence ~[]E, E cmp.Ordered](
	destination Destination, sequence Sequence,
) (count Count) {
	defer func() { Count_Invariants(count, "sorted_into.count") }()
	count = Clone_Into(destination, sequence)
	Sort(destination[:int(count)])
	return count
}

// Sorted_Function_Into copies sequence and sorts it through comparison.
func Sorted_Function_Into[Destination ~[]E, Sequence ~[]E, E any](
	destination Destination, sequence Sequence, comparison Comparison_Function[E, E],
) (count Count) {
	defer func() { Count_Invariants(count, "sorted_function_into.count") }()
	count = Clone_Into(destination, sequence)
	Sort_Function(destination[:int(count)], comparison)
	return count
}

// Sorted_Stable_Function_Into copies sequence and stably sorts it through comparison.
func Sorted_Stable_Function_Into[Destination ~[]E, Sequence ~[]E, E any](
	destination Destination, sequence Sequence, comparison Comparison_Function[E, E],
) (count Count) {
	defer func() { Count_Invariants(count, "sorted_stable_function_into.count") }()
	count = Clone_Into(destination, sequence)
	Sort_Stable_Function(destination[:int(count)], comparison)
	return count
}

// Chunk synchronously yields clipped consecutive views of at most count elements.
func Chunk[S ~[]E, E any](
	slice S, count Count, yield Chunk_Yield_Function[S, E],
) (chunk_count Count) {
	defer func() { Count_Invariants(chunk_count, "chunk.chunk_count") }()
	Count_Invariants(count, "chunk.count")
	enforce_slice(slice)
	aver.Always(count >= 1, "A chunk count is at least one.")
	count_int := int(count)
	for start := 0; start < len(slice); start += count_int {
		end := start + min(count_int, len(slice[start:]))
		continued := yield(slice[start:end:end])
		Boolean_Invariants(continued, "chunk.continued")
		chunk_count++
		if !continued {
			return chunk_count
		}
	}
	return chunk_count
}
