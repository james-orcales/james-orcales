package heap_test

import (
	"cmp"
	"testing"

	"local/james-orcales/shared/container/heap"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/testify"
)

// Test_Initialization verifies the heap order after Initialize, its idempotence, and an
// empty input.
func Test_Initialization(t *testing.T) {
	t.Parallel()
	elements := []int{5, 2, 9, 1, 7, 3}
	heap.Initialize(elements, smallest_first)
	testify.True(t, ordered(elements, smallest_first),
		"Initialize must establish the heap order")
	heap.Initialize(elements, smallest_first)
	testify.True(t, ordered(elements, smallest_first), "Initialize must be idempotent")
	empty := []int{}
	heap.Initialize(empty, smallest_first)
	testify.Count(t, empty, 0, "Initialize must accept an empty slice")
}

// Test_Insertion verifies that Push grows the slice and keeps the minimum at position zero.
func Test_Insertion(t *testing.T) {
	t.Parallel()
	elements := []int{}
	for _, value := range []int{5, 2, 9, 1, 7} {
		elements = heap.Push(elements, value, smallest_first)
	}
	testify.Count(t, elements, 5, "Push must grow the slice")
	testify.Equal(t, 1, elements[0], "Push must keep the minimum at position zero")
	testify.True(t, ordered(elements, smallest_first), "Push must keep the heap order")
}

// Test_Extraction verifies the Pop order and its equality with Remove at position zero.
func Test_Extraction(t *testing.T) {
	t.Parallel()
	elements := []int{5, 2, 9, 1, 7, 3}
	heap.Initialize(elements, smallest_first)
	testify.True(t, upward_order(drain(elements)),
		"Pop must return the elements from smallest to largest")
	first := []int{5, 2, 9, 1, 7, 3}
	heap.Initialize(first, smallest_first)
	second := clone(first)
	popped_elements, popped := heap.Pop(first, smallest_first)
	removed_elements, removed := heap.Remove(second, 0, smallest_first)
	testify.Equal(t, popped, removed, "Pop must equal Remove at position zero")
	testify.Equal(t, popped_elements, removed_elements,
		"Pop must leave the same elements as Remove at position zero")
}

// Test_Removal verifies that Remove takes one position and keeps the heap order.
func Test_Removal(t *testing.T) {
	t.Parallel()
	elements := []int{5, 2, 9, 1, 7, 3}
	heap.Initialize(elements, smallest_first)
	rest, removed := heap.Remove(elements, 2, smallest_first)
	testify.Count(t, rest, 5, "Remove must shrink the slice")
	testify.True(t, ordered(rest, smallest_first), "Remove must keep the heap order")
	survivors := drain(rest)
	testify.Count(t, survivors, 5, "Remove must keep every other element")
	testify.False(t, contains(survivors, removed),
		"Remove must take the element out of the heap")
}

// Test_Repair verifies that Fix restores the order after a growth and after a decrease.
func Test_Repair(t *testing.T) {
	t.Parallel()
	elements := []int{1, 2, 3, 4, 5, 6}
	elements[0] = 99
	heap.Fix(elements, 0, smallest_first)
	testify.True(t, ordered(elements, smallest_first),
		"Fix must restore the order after a growth")
	elements[5] = 0
	heap.Fix(elements, 5, smallest_first)
	testify.True(t, ordered(elements, smallest_first),
		"Fix must restore the order after a decrease")
	testify.Equal(t, 0, elements[0], "Fix must lift the new minimum to position zero")
}

// Test_Injected_Order verifies that an exchanged comparison gives a maximum heap.
func Test_Injected_Order(t *testing.T) {
	t.Parallel()
	elements := []int{5, 2, 9, 1, 7, 3}
	heap.Initialize(elements, largest_first)
	testify.Equal(t, 9, elements[0],
		"an exchanged comparison must keep the largest element at position zero")
	testify.True(t, ordered(elements, largest_first),
		"an exchanged comparison must give a maximum heap")
}

// Test_Size_Limits verifies the largest admitted heap and the rejection of a full Push.
func Test_Size_Limits(t *testing.T) {
	t.Parallel()
	full := ramp(heap.ELEMENT_COUNT_MAXIMUM)
	heap.Initialize(full, smallest_first)
	testify.True(t, ordered(full, smallest_first),
		"Initialize must accept the largest admitted heap")
	testify.True(t, panicked(func() { heap.Push(full, 0, smallest_first) }),
		"Push must reject a full slice")
}

// Test_Domain_Errors verifies the panic for an empty heap and for an absent position.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	elements := []int{1, 2, 3}
	panic_cases := []func(){
		func() { heap.Pop([]int{}, smallest_first) },
		func() { heap.Remove([]int{}, 0, smallest_first) },
		func() { heap.Remove(elements, 3, smallest_first) },
		func() { heap.Fix(elements, 3, smallest_first) },
		func() { heap.Fix([]int{}, 0, smallest_first) },
	}
	for panic_index, action := range panic_cases {
		testify.True(t, panicked(action), "domain error %d did not panic", panic_index)
	}
}

// Test_Invariant_Domains verifies each special value through a public heap operation.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	cover_position_domains()
	cover_count_domains()
	cover_report_domains()
}

// Reaches both position ends and both interior position sentinels.
func cover_position_domains() {
	full := ramp(heap.ELEMENT_COUNT_MAXIMUM)
	heap.Initialize(full, smallest_first)
	positions := []heap.Position{
		heap.POSITION_MINIMUM, 1, 2, heap.POSITION_MAXIMUM,
	}
	for _, position := range positions {
		heap.Fix(full, position, smallest_first)
	}
	rest, taken := heap.Remove(full, heap.POSITION_MAXIMUM, smallest_first)
	rest, taken = heap.Remove(rest, 2, smallest_first)
	rest, taken = heap.Remove(rest, 1, smallest_first)
	rest, taken = heap.Remove(rest, heap.POSITION_MINIMUM, smallest_first)
	heap.Push(rest, taken, smallest_first)
}

// Reaches both element-count ends and both interior element-count sentinels.
func cover_count_domains() {
	heap.Pop([]int{1}, smallest_first)
	heap.Pop([]int{1, 2}, smallest_first)
	heap.Pop([]int{1, 2, 3}, smallest_first)
	heap.Pop([]int{1, 2, 3, 4}, smallest_first)
	full := ramp(heap.ELEMENT_COUNT_MAXIMUM)
	heap.Initialize(full, smallest_first)
}

// Reaches both branches of the report that states whether an element moved down.
func cover_report_domains() {
	moved := []int{9, 1, 2, 3, 4, 5}
	heap.Fix(moved, 0, smallest_first)
	settled := []int{1, 2, 3, 4, 5, 6}
	heap.Fix(settled, 0, smallest_first)
}

// Compares two integers from smallest to largest.
func smallest_first(left int, right int) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(left, right))
}

// Compares two integers from largest to smallest.
func largest_first(left int, right int) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(right, left))
}

// Reports whether no child precedes its parent.
func ordered(
	elements []int, comparison slices.Comparison_Function[int, int],
) (yes bool) {
	for child_index := 1; child_index < len(elements); child_index++ {
		parent_index := (child_index - 1) / 2
		if comparison(elements[child_index], elements[parent_index]) < 0 {
			return false
		}
	}
	return true
}

// Reports whether the values never decrease.
func upward_order(values []int) (yes bool) {
	for index := 1; index < len(values); index++ {
		if values[index] < values[index-1] {
			return false
		}
	}
	return true
}

// Takes every element out of a heap, smallest first.
func drain(elements []int) (values []int) {
	rest := elements
	for len(rest) > 0 {
		shrunk, minimum := heap.Pop(rest, smallest_first)
		rest = shrunk
		values = append(values, minimum)
	}
	return values
}

// Reports whether the values hold one wanted value.
func contains(values []int, wanted int) (found bool) {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// Makes a decreasing ramp of one count of distinct elements.
func ramp(element_count int) (elements []int) {
	elements = make([]int, element_count)
	value := element_count
	for index := range elements {
		value--
		elements[index] = value
	}
	return elements
}

// Copies a slice so a second operation reads the original elements.
func clone(elements []int) (copied []int) {
	copied = make([]int, len(elements))
	copy(copied, elements)
	return copied
}

// Runs an action and reports whether it caused a panic.
func panicked(action func()) (raised bool) {
	defer func() {
		if recover() != nil {
			raised = true
		}
	}()
	action()
	return false
}
