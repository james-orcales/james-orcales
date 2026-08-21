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

// VALUE_MINIMUM keeps comparison input inside repository collection scale.
const VALUE_MINIMUM = -ELEMENT_COUNT_MAXIMUM

// VALUE_MAXIMUM keeps comparison input inside repository collection scale.
const VALUE_MAXIMUM = ELEMENT_COUNT_MAXIMUM

// Value is concrete heap value.
type Value int

// Value_Invariants keeps payload inside finite assertion domain.
func Value_Invariants(value Value, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), VALUE_MINIMUM, VALUE_MAXIMUM).
		Ensure()
}

// Elements is caller-owned heap storage.
type Elements []Value

// Elements_Invariants bounds work at every heap boundary.
func Elements_Invariants(elements Elements, namespace aver.Namespace) {
	aver.Tree(elements, namespace).
		Range_Int(len(elements), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// NONEMPTY_COUNT_MINIMUM is smallest storage that holds one position.
const NONEMPTY_COUNT_MINIMUM = COUNT_MINIMUM + 1

// Nonempty_Elements holds at least one position.
type Nonempty_Elements []Value

// Nonempty_Elements_Invariants excludes storage with no position.
func Nonempty_Elements_Invariants(elements Nonempty_Elements, namespace aver.Namespace) {
	aver.Tree(elements, namespace).
		Range_Int(len(elements), NONEMPTY_COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// NONFULL_COUNT_MAXIMUM keeps one position free below largest heap.
const NONFULL_COUNT_MAXIMUM = COUNT_MAXIMUM - 1

// Nonfull_Elements has at least one free admitted position.
type Nonfull_Elements []Value

// Nonfull_Elements_Invariants keeps room below largest admitted heap.
func Nonfull_Elements_Invariants(elements Nonfull_Elements, namespace aver.Namespace) {
	aver.Tree(elements, namespace).
		Range_Int(len(elements), COUNT_MINIMUM, NONFULL_COUNT_MAXIMUM).
		Ensure()
}

// Comparison_Function injects heap order.
type Comparison_Function func(left Value, right Value) (comparison slices.Comparison)

// Enforces package-wide size boundary at one source root.
func enforce_heap(elements Elements) {
	Elements_Invariants(elements, "enforce_heap.elements")
	aver.Always(
		len(elements) <= ELEMENT_COUNT_MAXIMUM,
		"A heap boundary admits at most ELEMENT_COUNT_MAXIMUM elements.",
	)
}

// Enforces that a position names an element that the slice holds. The Position domain covers
// the largest admitted heap, thus only the slice states which of those positions exists.
func enforce_position(elements Elements, position Position) {
	Elements_Invariants(elements, "enforce_position.elements")
	Position_Invariants(position, "enforce_position.position")
	aver.Always(
		int(position) < len(elements),
		"A heap position names an element that the slice holds.",
	)
}

// Initialize orders an arbitrary element slice into a minimum heap.
func Initialize(elements Elements, comparison Comparison_Function) {
	Elements_Invariants(elements, "initialize.elements")
	enforce_heap(elements)
	element_count := len(elements)
	parent_index := element_count/CHILD_COUNT - 1
	for parent_index >= 0 {
		sift_down(
			Nonempty_Elements(elements), Position(parent_index),
			Count(element_count), comparison,
		)
		parent_index--
	}
}

// Push uses caller-owned spare capacity because hidden growth would violate package allocation
// boundary.
func Push(
	elements Elements, element Value, comparison Comparison_Function,
) (result Elements) {
	defer func() {
		Elements_Invariants(result, "push.result")
		enforce_heap(result)
	}()
	Elements_Invariants(elements, "push.elements")
	Value_Invariants(element, "push.element")
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
	sift_up(Nonempty_Elements(result), Position(len(result)-1), comparison)
	return result
}

// Pop takes the minimum element and returns the shrunk slice together with that element.
func Pop(
	elements Elements, comparison Comparison_Function,
) (result Nonfull_Elements, minimum Value) {
	defer func() {
		Nonfull_Elements_Invariants(result, "pop.result")
		Value_Invariants(minimum, "pop.minimum")
	}()
	Elements_Invariants(elements, "pop.elements")
	enforce_heap(elements)
	enforce_position(elements, POSITION_MINIMUM)
	final_index := len(elements) - 1
	elements[POSITION_MINIMUM], elements[final_index] =
		elements[final_index], elements[POSITION_MINIMUM]
	sift_down(Nonempty_Elements(elements), POSITION_MINIMUM, Count(final_index), comparison)
	return Nonfull_Elements(elements[:final_index]), elements[final_index]
}

// Remove takes the element at one position and returns the shrunk slice with that element.
func Remove(
	elements Elements, position Position, comparison Comparison_Function,
) (result Nonfull_Elements, removed Value) {
	defer func() {
		Nonfull_Elements_Invariants(result, "remove.result")
		Value_Invariants(removed, "remove.removed")
	}()
	Elements_Invariants(elements, "remove.elements")
	Position_Invariants(position, "remove.position")
	enforce_heap(elements)
	enforce_position(elements, position)
	final_index := len(elements) - 1
	if final_index != int(position) {
		elements[int(position)], elements[final_index] =
			elements[final_index], elements[int(position)]
		if !sift_down(
			Nonempty_Elements(elements), position, Count(final_index), comparison,
		) {
			sift_up(Nonempty_Elements(elements), position, comparison)
		}
	}
	return Nonfull_Elements(elements[:final_index]), elements[final_index]
}

// Fix restores the heap order after the element at one position changes.
func Fix(elements Elements, position Position, comparison Comparison_Function) {
	Elements_Invariants(elements, "fix.elements")
	Position_Invariants(position, "fix.position")
	enforce_heap(elements)
	enforce_position(elements, position)
	if !sift_down(Nonempty_Elements(elements), position, Count(len(elements)), comparison) {
		sift_up(Nonempty_Elements(elements), position, comparison)
	}
}

// Moves the element at one position toward the root while it precedes its parent.
func sift_up(elements Nonempty_Elements, position Position, comparison Comparison_Function) {
	Nonempty_Elements_Invariants(elements, "sift_up.elements")
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
func sift_down(
	elements Nonempty_Elements,
	position Position,
	boundary Count,
	comparison Comparison_Function,
) (moved Boolean) {
	defer func() { Boolean_Invariants(moved, "sift_down.moved") }()
	Nonempty_Elements_Invariants(elements, "sift_down.elements")
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
