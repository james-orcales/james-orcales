// Package heap supplies the Go standard-library binary minimum-heap algorithms through the
// repository naming and assertion boundaries. The caller owns the element storage and injects
// the order, thus this package keeps no state and needs no interface. A comparison that
// exchanges its two operands gives a maximum heap.
package heap

import (
	"local/james-orcales/shared/sim/aver/default"
	"local/james-orcales/shared/slices"
)

// ELEMENT_COUNT_MAXIMUM caps each element slice that crosses this deterministic package
// boundary. It matches the repository slice boundary, so one common element count applies.
const ELEMENT_COUNT_MAXIMUM = slices.SLICE_COUNT_MAXIMUM

// COUNT_MINIMUM is the element count of an empty heap.
const COUNT_MINIMUM = 0

// COUNT_MAXIMUM is the largest admitted element count.
const COUNT_MAXIMUM = ELEMENT_COUNT_MAXIMUM

// POSITION_MINIMUM is the root position of a heap that holds at least one element.
const POSITION_MINIMUM = 0

// POSITION_MAXIMUM is the final position of the largest admitted heap.
const POSITION_MAXIMUM = COUNT_MAXIMUM - 1

// CHILD_COUNT is the child count of one parent in a binary heap.
const CHILD_COUNT = 2

// Position is one element position in a heap.
type Position int

// Position_Invariants states the complete element-position domain.
func Position_Invariants(value Position, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Count is an element count, thus it includes the position after the final element.
type Count int

// Count_Invariants states the complete element-count domain.
func Count_Invariants(value Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Boolean is a true or false report from a heap operation.
type Boolean bool

// Boolean_Invariants states both report results as obligations.
func Boolean_Invariants(value Boolean, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Sometimes(bool(value), "The heap report is true.").
		Ensure()
}

// Enforces the package-wide size boundary at one source root. One generic boundary avoids a
// wrapper type that would discard a caller's named slice type.
func enforce_heap[S ~[]E, E any](elements S) {
	aver.Always(
		len(elements) <= ELEMENT_COUNT_MAXIMUM,
		"A heap boundary admits at most ELEMENT_COUNT_MAXIMUM elements.",
	)
}

// Enforces that a position names an element that the slice holds. The Position domain covers
// the largest admitted heap, thus only the slice states which of those positions exists.
func enforce_position[S ~[]E, E any](elements S, position Position) {
	Position_Invariants(position, "enforce_position.position")
	aver.Always(
		int(position) < len(elements),
		"A heap position names an element that the slice holds.",
	)
}

// Initialize orders an arbitrary element slice into a minimum heap.
func Initialize[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) {
	enforce_heap(elements)
	element_count := len(elements)
	parent_index := element_count/CHILD_COUNT - 1
	for parent_index >= 0 {
		sift_down(elements, Position(parent_index), Count(element_count), comparison)
		parent_index--
	}
}

// Push uses caller-owned spare capacity because hidden growth would violate package allocation
// boundary.
func Push[S ~[]E, E any](
	elements S, element E, comparison slices.Comparison_Function[E, E],
) (result S) {
	defer func() { enforce_heap(result) }()
	enforce_heap(elements)
	aver.Always(
		len(elements) < ELEMENT_COUNT_MAXIMUM,
		"A heap Push keeps room for the new element.",
	)
	aver.Always(
		len(elements) < cap(elements),
		"A heap Push uses room in caller-owned storage.",
	)
	result = elements[:len(elements)+1]
	result[len(elements)] = element
	sift_up(result, Position(len(result)-1), comparison)
	return result
}

// Pop takes the minimum element and returns the shrunk slice together with that element.
func Pop[S ~[]E, E any](
	elements S, comparison slices.Comparison_Function[E, E],
) (result S, minimum E) {
	defer func() { enforce_heap(result) }()
	enforce_heap(elements)
	enforce_position(elements, POSITION_MINIMUM)
	final_index := len(elements) - 1
	elements[POSITION_MINIMUM], elements[final_index] =
		elements[final_index], elements[POSITION_MINIMUM]
	sift_down(elements, POSITION_MINIMUM, Count(final_index), comparison)
	return elements[:final_index], elements[final_index]
}

// Remove takes the element at one position and returns the shrunk slice with that element.
func Remove[S ~[]E, E any](
	elements S, position Position, comparison slices.Comparison_Function[E, E],
) (result S, removed E) {
	defer func() { enforce_heap(result) }()
	Position_Invariants(position, "remove.position")
	enforce_heap(elements)
	enforce_position(elements, position)
	final_index := len(elements) - 1
	if final_index != int(position) {
		elements[int(position)], elements[final_index] =
			elements[final_index], elements[int(position)]
		if !sift_down(elements, position, Count(final_index), comparison) {
			sift_up(elements, position, comparison)
		}
	}
	return elements[:final_index], elements[final_index]
}

// Fix restores the heap order after the element at one position changes.
func Fix[S ~[]E, E any](
	elements S, position Position, comparison slices.Comparison_Function[E, E],
) {
	Position_Invariants(position, "fix.position")
	enforce_heap(elements)
	enforce_position(elements, position)
	if !sift_down(elements, position, Count(len(elements)), comparison) {
		sift_up(elements, position, comparison)
	}
}

// Moves the element at one position toward the root while it precedes its parent.
func sift_up[S ~[]E, E any](
	elements S, position Position, comparison slices.Comparison_Function[E, E],
) {
	Position_Invariants(position, "sift_up.position")
	child_index := int(position)
	for child_index > 0 {
		parent_index := (child_index - 1) / CHILD_COUNT
		if comparison(elements[child_index], elements[parent_index]) >= 0 {
			return
		}
		elements[parent_index], elements[child_index] =
			elements[child_index], elements[parent_index]
		child_index = parent_index
	}
}

// Moves the element at one position away from the root while a child precedes it. The report
// states whether the element moved, thus a caller knows that no upward move can apply. The
// standard library also guards against an index overflow, which the bounded element count
// here makes unreachable.
func sift_down[S ~[]E, E any](
	elements S,
	position Position,
	boundary Count,
	comparison slices.Comparison_Function[E, E],
) (moved Boolean) {
	defer func() { Boolean_Invariants(moved, "sift_down.moved") }()
	Position_Invariants(position, "sift_down.position")
	Count_Invariants(boundary, "sift_down.boundary")
	parent_index := int(position)
	child_index := CHILD_COUNT*parent_index + 1
	for child_index < int(boundary) {
		least_index := child_index
		sibling_index := child_index + 1
		if sibling_index < int(boundary) {
			if comparison(elements[sibling_index], elements[child_index]) < 0 {
				least_index = sibling_index
			}
		}
		if comparison(elements[least_index], elements[parent_index]) >= 0 {
			return moved
		}
		elements[parent_index], elements[least_index] =
			elements[least_index], elements[parent_index]
		parent_index = least_index
		moved = true
		child_index = CHILD_COUNT*parent_index + 1
	}
	return moved
}
