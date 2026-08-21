package heap

import (
	"cmp"
	"testing"

	"local/james-orcales/shared/sim/prng"
	"local/james-orcales/shared/slices"
	"local/james-orcales/shared/testify"
)

// STANDARD_HEAP_ELEMENT_COUNT preserves upstream workload while caller owns fixed storage.
const STANDARD_HEAP_ELEMENT_COUNT = 20

// Compares two integers from smallest to largest.
func smallest_first(left int, right int) (comparison slices.Comparison) {
	return slices.Comparison(cmp.Compare(left, right))
}

// Test_Standard_Library_Initialize_Equal_Elements ports the upstream TestInit0. Initialize
// accepts equal elements, and each Pop then returns that one value.
func Test_Standard_Library_Initialize_Equal_Elements(t *testing.T) {
	t.Parallel()
	elements := []int{}
	for range 20 {
		elements = append(elements, 0)
	}
	Initialize(elements, smallest_first)
	verify_heap(t, elements)
	pop_index := 1
	for len(elements) > 0 {
		rest, minimum := Pop(elements, smallest_first)
		elements = rest
		verify_heap(t, elements)
		testify.Equal(t, 0, minimum, "pop %d returned an unexpected value", pop_index)
		pop_index++
	}
}

// Test_Standard_Library_Initialize_Distinct_Elements ports the upstream TestInit1. Initialize
// accepts a decreasing ramp, and each Pop then returns the next smallest value.
func Test_Standard_Library_Initialize_Distinct_Elements(t *testing.T) {
	t.Parallel()
	elements := []int{}
	for value := 20; value > 0; value-- {
		elements = append(elements, value)
	}
	Initialize(elements, smallest_first)
	verify_heap(t, elements)
	wanted := 1
	for len(elements) > 0 {
		rest, minimum := Pop(elements, smallest_first)
		elements = rest
		verify_heap(t, elements)
		testify.Equal(t, wanted, minimum, "pop %d returned an unexpected value", wanted)
		wanted++
	}
}

// Test_Standard_Library_Push_And_Pop ports the upstream Test. It mixes Initialize, Push, and
// Pop over one heap and requires the smallest value at each extraction.
func Test_Standard_Library_Push_And_Pop(t *testing.T) {
	t.Parallel()
	var storage [STANDARD_HEAP_ELEMENT_COUNT]int
	elements := storage[:0]
	verify_heap(t, elements)
	for value := 20; value > 10; value-- {
		elements = elements[:len(elements)+1]
		elements[len(elements)-1] = value
	}
	Initialize(elements, smallest_first)
	verify_heap(t, elements)
	for value := 10; value > 0; value-- {
		elements = Push(elements, value, smallest_first)
		verify_heap(t, elements)
	}
	wanted := 1
	for len(elements) > 0 {
		rest, minimum := Pop(elements, smallest_first)
		elements = rest
		if wanted < 20 {
			elements = Push(elements, 20+wanted, smallest_first)
		}
		verify_heap(t, elements)
		testify.Equal(t, wanted, minimum, "pop %d returned an unexpected value", wanted)
		wanted++
	}
}

// Test_Standard_Library_Remove_Final_Position ports the upstream TestRemove0. A Remove of the
// final position returns the element that the slice holds there.
func Test_Standard_Library_Remove_Final_Position(t *testing.T) {
	t.Parallel()
	elements := []int{}
	for value := range 10 {
		elements = append(elements, value)
	}
	verify_heap(t, elements)
	for len(elements) > 0 {
		final_index := len(elements) - 1
		rest, removed := Remove(
			elements, Position(final_index), smallest_first,
		)
		elements = rest
		testify.Equal(t, final_index, removed,
			"Remove at position %d returned an unexpected value", final_index)
		verify_heap(t, elements)
	}
}

// Test_Standard_Library_Remove_Root ports the upstream TestRemove1. A Remove of position zero
// drains the heap from smallest to largest.
func Test_Standard_Library_Remove_Root(t *testing.T) {
	t.Parallel()
	elements := []int{}
	for value := range 10 {
		elements = append(elements, value)
	}
	verify_heap(t, elements)
	wanted := 0
	for len(elements) > 0 {
		rest, removed := Remove(elements, POSITION_MINIMUM, smallest_first)
		elements = rest
		testify.Equal(t, wanted, removed,
			"Remove at position zero returned an unexpected value")
		verify_heap(t, elements)
		wanted++
	}
}

// Test_Standard_Library_Remove_Interior ports the upstream TestRemove2. A repeated Remove of
// an interior position takes each element exactly one time.
func Test_Standard_Library_Remove_Interior(t *testing.T) {
	t.Parallel()
	elements := []int{}
	for value := range 10 {
		elements = append(elements, value)
	}
	verify_heap(t, elements)
	seen := map[int]bool{}
	for len(elements) > 0 {
		final_index := len(elements) - 1
		middle_index := final_index / 2
		rest, removed := Remove(
			elements, Position(middle_index), smallest_first,
		)
		elements = rest
		seen[removed] = true
		verify_heap(t, elements)
	}
	testify.Count(t, seen, 10, "Remove must take each element one time")
	for value := range 10 {
		testify.True(t, seen[value], "value %d was never removed", value)
	}
}

// Test_Standard_Library_Fix ports the upstream TestFix. It changes one element at a time,
// then requires Fix to restore the heap order in both directions.
func Test_Standard_Library_Fix(t *testing.T) {
	t.Parallel()
	var storage [STANDARD_HEAP_ELEMENT_COUNT]int
	elements := storage[:0]
	for value := 200; value > 0; value -= 10 {
		elements = Push(elements, value, smallest_first)
	}
	verify_heap(t, elements)
	testify.Equal(t, 10, elements[0], "the root must hold the smallest value")
	elements[0] = 210
	Fix(elements, POSITION_MINIMUM, smallest_first)
	verify_heap(t, elements)
	generator := prng.New(1)
	for round := 100; round > 0; round-- {
		changed_index := prng.Xoshiro_Below(&generator, prng.Bound(len(elements)))
		if round%2 == 0 {
			elements[changed_index] *= 2
		} else {
			elements[changed_index] /= 2
		}
		Fix(elements, Position(changed_index), smallest_first)
		verify_heap(t, elements)
	}
}

// Fails when any child precedes its parent. The upstream verify method walks the tree by
// recursion, which this repository bans, and one pass over each child states the same
// property.
func verify_heap(t *testing.T, elements []int) {
	t.Helper()
	for child_index := 1; child_index < len(elements); child_index++ {
		parent_index := (child_index - 1) / 2
		testify.False(t, elements[child_index] < elements[parent_index],
			"the heap order is broken at position %d", child_index)
	}
}
