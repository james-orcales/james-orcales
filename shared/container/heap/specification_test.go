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
	elements := heap.Elements{5, 2, 9, 1, 7, 3}
	heap.Initialize(elements, smallest_first)
	testify.True(t, ordered(elements, smallest_first),
		"Initialize must establish the heap order")
	heap.Initialize(elements, smallest_first)
	testify.True(t, ordered(elements, smallest_first), "Initialize must be idempotent")
	empty := heap.Elements{}
	heap.Initialize(empty, smallest_first)
	testify.Count(t, empty, 0, "Initialize must accept an empty slice")
}

// Test_Insertion keeps caller-owned growth executable because silent growth restores collector
// dependence.
func Test_Insertion(t *testing.T) {
	t.Parallel()
	var storage [INSERTION_ELEMENT_COUNT]heap.Value
	elements := heap.Elements(storage[:0])
	for _, value := range (heap.Elements{5, 2, 9, 1, 7}) {
		elements = heap.Push(elements, value, smallest_first)
	}
	testify.Count(t, elements, 5, "Push must grow the slice")
	testify.Equal(t, heap.Value(1), elements[0], "Push must keep the minimum at position zero")
	testify.True(t, ordered(elements, smallest_first), "Push must keep the heap order")
	without_storage := heap.Elements{}
	testify.True(t, panicked(func() {
		heap.Push(without_storage, 1, smallest_first)
	}), "Push must reject a slice without room in caller storage")
	full_storage := [...]heap.Value{1, 2}
	testify.True(t, panicked(func() {
		heap.Push(heap.Elements(full_storage[:]), 3, smallest_first)
	}), "Push must reject exhausted caller storage below ELEMENT_COUNT_MAXIMUM")
}

// Test_Extraction verifies the Pop order and its equality with Remove at position zero.
func Test_Extraction(t *testing.T) {
	t.Parallel()
	elements := heap.Elements{5, 2, 9, 1, 7, 3}
	heap.Initialize(elements, smallest_first)
	testify.True(t, upward_order(drain(elements)),
		"Pop must return the elements from smallest to largest")
	first := heap.Elements{5, 2, 9, 1, 7, 3}
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
	elements := heap.Elements{5, 2, 9, 1, 7, 3}
	heap.Initialize(elements, smallest_first)
	rest, removed := heap.Remove(elements, 2, smallest_first)
	testify.Count(t, rest, 5, "Remove must shrink the slice")
	testify.True(t, ordered(heap.Elements(rest), smallest_first),
		"Remove must keep the heap order")
	survivors := drain(heap.Elements(rest))
	testify.Count(t, survivors, 5, "Remove must keep every other element")
	testify.False(t, contains(survivors, removed),
		"Remove must take the element out of the heap")
}

// Test_Repair verifies that Fix restores the order after a growth and after a decrease.
func Test_Repair(t *testing.T) {
	t.Parallel()
	elements := heap.Elements{1, 2, 3, 4, 5, 6}
	elements[0] = 99
	heap.Fix(elements, 0, smallest_first)
	testify.True(t, ordered(elements, smallest_first),
		"Fix must restore the order after a growth")
	elements[5] = 0
	heap.Fix(elements, 5, smallest_first)
	testify.True(t, ordered(elements, smallest_first),
		"Fix must restore the order after a decrease")
	testify.Equal(t, heap.Value(0), elements[0],
		"Fix must lift the new minimum to position zero")
}

// Test_Injected_Order verifies that an exchanged comparison gives a maximum heap.
func Test_Injected_Order(t *testing.T) {
	t.Parallel()
	elements := heap.Elements{5, 2, 9, 1, 7, 3}
	heap.Initialize(elements, largest_first)
	testify.Equal(t, heap.Value(9), elements[0],
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

// Test_Allocation keeps collector independence executable across every heap operation;
// caller storage alone must absorb insertion.
func Test_Allocation(t *testing.T) {
	state := prepare_heap_allocation_state()
	for _, one := range heap_allocation_cases(state) {
		t.Run(one.Name, func(t *testing.T) {
			testify.Zero_Allocation(
				t, one.Run, "%s must allocate no heap storage", one.Name,
			)
		})
	}
	testify.Count(t, state.Push_Result, len(state.Push_Storage),
		"Push allocation probe did not run")
	testify.Count(t, state.Pop_Result, len(state.Pop_Storage)-1,
		"Pop allocation probe did not run")
	testify.Equal(t, heap.Value(1), state.Popped,
		"Pop allocation probe returned wrong value")
	testify.Count(t, state.Remove_Result, len(state.Remove_Storage)-1,
		"Remove allocation probe did not run")
	testify.Equal(t, heap.Value(2), state.Removed,
		"Remove allocation probe returned wrong value")
	testify.True(t, ordered(state.Fix_Storage[:], smallest_first),
		"Fix allocation probe did not run")
}

// Test_Domain_Errors verifies the panic for an empty heap and for an absent position.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	elements := heap.Elements{1, 2, 3}
	panic_cases := []func(){
		func() { heap.Pop(heap.Elements{}, smallest_first) },
		func() { heap.Remove(heap.Elements{}, 0, smallest_first) },
		func() { heap.Remove(elements, 3, smallest_first) },
		func() { heap.Fix(elements, 3, smallest_first) },
		func() { heap.Fix(heap.Elements{}, 0, smallest_first) },
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
	cover_value_domains()
}

// INSERTION_ELEMENT_COUNT binds caller storage to values exercised by insertion contract.
const INSERTION_ELEMENT_COUNT = 5

// ALLOCATION_ELEMENT_COUNT keeps each destructive allocation probe above trivial boundaries.
const ALLOCATION_ELEMENT_COUNT = 4

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
	rest, taken = heap.Remove(heap.Elements(rest), 2, smallest_first)
	rest, taken = heap.Remove(heap.Elements(rest), 1, smallest_first)
	rest, taken = heap.Remove(heap.Elements(rest), heap.POSITION_MINIMUM, smallest_first)
	heap.Push(heap.Elements(rest), taken, smallest_first)
}

// Reaches both element-count ends and both interior element-count sentinels.
func cover_count_domains() {
	heap.Pop(heap.Elements{1}, smallest_first)
	heap.Pop(heap.Elements{1, 2}, smallest_first)
	heap.Pop(heap.Elements{1, 2, 3}, smallest_first)
	heap.Pop(heap.Elements{1, 2, 3, 4}, smallest_first)
	heap.Initialize(heap.Elements{1}, smallest_first)
	heap.Initialize(heap.Elements{2, 1}, smallest_first)
	heap.Fix(heap.Elements{1}, 0, smallest_first)
	heap.Fix(heap.Elements{1, 2}, 0, smallest_first)
	full := ramp(heap.ELEMENT_COUNT_MAXIMUM)
	heap.Initialize(full, smallest_first)
	heap.Pop(full, smallest_first)
	push_storage := make(heap.Elements, heap.ELEMENT_COUNT_MAXIMUM)
	heap.Push(push_storage[:heap.ELEMENT_COUNT_MAXIMUM-1], 0, smallest_first)
}

// Reaches both branches of the report that states whether an element moved down.
func cover_report_domains() {
	moved := heap.Elements{9, 1, 2, 3, 4, 5}
	heap.Fix(moved, 0, smallest_first)
	settled := heap.Elements{1, 2, 3, 4, 5, 6}
	heap.Fix(settled, 0, smallest_first)
}

// Reaches every sentinel required by concrete stored-value boundaries through public calls.
func cover_value_domains() {
	values := heap.Elements{
		heap.VALUE_MINIMUM, -1, 0, 1, 2, heap.VALUE_MAXIMUM,
	}
	for _, value := range values {
		storage := make(heap.Elements, 1)
		heap.Push(storage[:0], value, smallest_first)
		heap.Pop(heap.Elements{value}, smallest_first)
		heap.Remove(heap.Elements{value}, 0, smallest_first)
	}
}

// Compares two integers from smallest to largest.
func smallest_first(left heap.Value, right heap.Value) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(left, right))
}

// Compares two integers from largest to smallest.
func largest_first(left heap.Value, right heap.Value) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(right, left))
}

// Reports whether no child precedes its parent.
func ordered(
	elements heap.Elements, comparison heap.Comparison_Function,
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
func upward_order(values heap.Elements) (yes bool) {
	for index := 1; index < len(values); index++ {
		if values[index] < values[index-1] {
			return false
		}
	}
	return true
}

// Takes every element out of a heap, smallest first.
func drain(elements heap.Elements) (values heap.Elements) {
	rest := elements
	for len(rest) > 0 {
		shrunk, minimum := heap.Pop(rest, smallest_first)
		rest = heap.Elements(shrunk)
		values = append(values, minimum)
	}
	return values
}

// Reports whether the values hold one wanted value.
func contains(values heap.Elements, wanted heap.Value) (found bool) {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

// Makes a decreasing ramp of one count of distinct elements.
func ramp(element_count int) (elements heap.Elements) {
	elements = make(heap.Elements, element_count)
	value := heap.Value(element_count)
	for index := range elements {
		value--
		elements[index] = value
	}
	return elements
}

// Copies a slice so a second operation reads the original elements.
func clone(elements heap.Elements) (copied heap.Elements) {
	copied = make(heap.Elements, len(elements))
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

type allocation_case struct {
	Name string
	Run  func()
}

type heap_allocation_state struct {
	Initialize_Storage heap.Elements
	Push_Storage       heap.Elements
	Pop_Storage        heap.Elements
	Remove_Storage     heap.Elements
	Fix_Storage        heap.Elements
	Push_Result        heap.Elements
	Pop_Result         heap.Nonfull_Elements
	Popped             heap.Value
	Remove_Result      heap.Nonfull_Elements
	Removed            heap.Value
}

// Shared state prevents probe setup from entering measured callbacks through closure creation.
func prepare_heap_allocation_state() (state *heap_allocation_state) {
	return &heap_allocation_state{
		Initialize_Storage: make(heap.Elements, ALLOCATION_ELEMENT_COUNT),
		Push_Storage:       make(heap.Elements, ALLOCATION_ELEMENT_COUNT),
		Pop_Storage:        make(heap.Elements, ALLOCATION_ELEMENT_COUNT),
		Remove_Storage:     make(heap.Elements, ALLOCATION_ELEMENT_COUNT),
		Fix_Storage:        make(heap.Elements, ALLOCATION_ELEMENT_COUNT),
	}
}

// Separate callbacks name allocation proof for each operation.
func heap_allocation_cases(state *heap_allocation_state) (cases []allocation_case) {
	return []allocation_case{
		{
			Name: "Initialize",
			Run: func() {
				copy(state.Initialize_Storage, heap.Elements{4, 2, 3, 1})
				heap.Initialize(state.Initialize_Storage, smallest_first)
			},
		},
		{
			Name: "Push",
			Run: func() {
				copy(state.Push_Storage, heap.Elements{1, 2, 3, 0})
				state.Push_Result = heap.Push(
					state.Push_Storage[:3], 0, smallest_first,
				)
			},
		},
		{
			Name: "Pop",
			Run: func() {
				copy(state.Pop_Storage, heap.Elements{1, 2, 3, 4})
				state.Pop_Result, state.Popped = heap.Pop(
					state.Pop_Storage, smallest_first,
				)
			},
		},
		{
			Name: "Remove",
			Run: func() {
				copy(state.Remove_Storage, heap.Elements{1, 2, 3, 4})
				state.Remove_Result, state.Removed = heap.Remove(
					state.Remove_Storage, 1, smallest_first,
				)
			},
		},
		{
			Name: "Fix",
			Run: func() {
				copy(state.Fix_Storage, heap.Elements{9, 2, 3, 4})
				heap.Fix(state.Fix_Storage, 0, smallest_first)
			},
		},
	}
}
