// Package list supplies the Go standard-library doubly linked list through the repository
// naming and assertion boundaries. A list owns a node pool and joins its nodes by position,
// not by pointer, because a pointer node states itself as its own neighbor and no assertion
// tree can hold that cycle. Each operation is a free function, thus a list carries no method
// set.
//
// A list holds its whole pool in one array, thus its size is the same for an empty list and
// for a full one. Make a list behind a pointer and do not copy the value.
package list

import (
	invariant "local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/slices"
)

// ELEMENT_COUNT_MAXIMUM caps each list that crosses this deterministic package boundary. It
// matches the repository slice boundary, so one common element count applies.
const ELEMENT_COUNT_MAXIMUM = slices.SLICE_COUNT_MAXIMUM

// SENTINEL_COUNT is the node count that a list reserves for its two chain heads.
const SENTINEL_COUNT = 2

// COUNT_MINIMUM is the element count of an empty list.
const COUNT_MINIMUM = 0

// COUNT_MAXIMUM is the largest admitted element count.
const COUNT_MAXIMUM = ELEMENT_COUNT_MAXIMUM

// NODE_COUNT_MAXIMUM is the node count of the largest admitted list.
const NODE_COUNT_MAXIMUM = COUNT_MAXIMUM + SENTINEL_COUNT

// POSITION_NONE is the handle that no element holds.
const POSITION_NONE Position = -1

// ROOT_POSITION holds the sentinel of the element ring.
const ROOT_POSITION = 0

// FREE_POSITION holds the sentinel of the free chain.
const FREE_POSITION = 1

// FIRST_ELEMENT_POSITION is the first position that a caller value can occupy.
const FIRST_ELEMENT_POSITION = FREE_POSITION + 1

// POSITION_MINIMUM is POSITION_NONE, which every handle and neighbor field can hold.
const POSITION_MINIMUM = int(POSITION_NONE)

// POSITION_MAXIMUM is the final position of the largest admitted list.
const POSITION_MAXIMUM = NODE_COUNT_MAXIMUM - 1

// Position is one element handle of a list, or POSITION_NONE. Neither sentinel is a handle,
// thus both sentinel positions are holes.
type Position int

// Position_Invariants states the complete element-handle domain.
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), POSITION_MINIMUM, POSITION_MAXIMUM,
			ROOT_POSITION, FREE_POSITION, FREE_POSITION, FREE_POSITION,
		).
		Ensure()
}

// Element_Position is one node that the element ring holds, thus it is never POSITION_NONE
// and never a sentinel.
type Element_Position int

// Element_Position_Invariants states the domain of one live element node.
func Element_Position_Invariants(value Element_Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), FIRST_ELEMENT_POSITION, POSITION_MAXIMUM).
		Ensure()
}

// Mark_Position is the node that an insertion follows. The ring sentinel is the mark of a
// front insertion, thus this domain adds ROOT_POSITION to the live element nodes.
type Mark_Position int

// Mark_Position_Invariants states the domain of one insertion mark.
func Mark_Position_Invariants(value Mark_Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), ROOT_POSITION, POSITION_MAXIMUM,
			FREE_POSITION, FREE_POSITION, FREE_POSITION, FREE_POSITION,
		).
		Ensure()
}

// Successor is the position of the node after one node, or POSITION_NONE.
type Successor int

// Successor_Invariants states the complete neighbor domain.
func Successor_Invariants(value Successor, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Predecessor is the position of the node before one node, or POSITION_NONE.
type Predecessor int

// Predecessor_Invariants states the complete neighbor domain.
func Predecessor_Invariants(value Predecessor, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), POSITION_MINIMUM, POSITION_MAXIMUM).
		Ensure()
}

// Count is the element count of a list, and it excludes both sentinels.
type Count int

// Count_Invariants states the complete element-count domain.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Boolean is a true or false report about one node.
type Boolean bool

// Boolean_Invariants states both report results as obligations.
func Boolean_Invariants(value Boolean, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Sometimes(bool(value), "The node report is true.").
		Ensure()
}

// Element is one node of the pool of a list.
type Element[Value any] struct {
	// Next is the following node of the element ring, or the following node of the free
	// chain while the element ring does not hold this node.
	Next Successor
	// Previous is the preceding node of the element ring. A free node holds POSITION_NONE.
	Previous Predecessor
	// Used states whether the element ring holds this node, thus an operation on a handle
	// that Remove already took causes a panic.
	Used Boolean
	// Value is the value that the caller stores. This package does not read it.
	Value Value
}

// Element_Invariants composes both neighbors and the use report of one node.
func Element_Invariants[Value any](
	element Element[Value], namespace invariant.Namespace,
) {
	Successor_Invariants(element.Next, namespace)
	Predecessor_Invariants(element.Previous, namespace)
	Boolean_Invariants(element.Used, namespace)
}

// List is a doubly linked list. Its zero value is an empty list that is ready to use.
type List[Value any] struct {
	// Nodes holds the ring sentinel at ROOT_POSITION, the free-chain sentinel at
	// FREE_POSITION, and every caller node after them. The pool is one array, because a
	// slice field carries no defined type that an assertion tree can own.
	Nodes [NODE_COUNT_MAXIMUM]Element[Value]
	// Element_Count excludes both sentinels and every free node.
	Element_Count Count
}

// List_Invariants states the element count of one list. The subject is a pointer, because a
// list holds its whole pool and a value parameter would copy that pool at each assertion.
func List_Invariants[Value any](
	subject *List[Value], namespace invariant.Namespace,
) {
	Count_Invariants(subject.Element_Count, namespace)
}

// Initialize empties a list and closes its element ring. The zero List value is already an
// empty list, thus this package states one entry point where the standard library also has a
// New constructor.
func Initialize[Value any](subject *List[Value]) {
	List_Invariants(subject, "initialize.subject")
	subject.Nodes[ROOT_POSITION].Next = ROOT_POSITION
	subject.Nodes[ROOT_POSITION].Previous = ROOT_POSITION
	subject.Nodes[POSITION_MAXIMUM].Next = Successor(POSITION_NONE)
	subject.Nodes[POSITION_MAXIMUM].Previous = Predecessor(POSITION_NONE)
	subject.Nodes[POSITION_MAXIMUM].Used = false
	for position := FREE_POSITION; position < POSITION_MAXIMUM; position++ {
		subject.Nodes[position].Next = Successor(position + 1)
		subject.Nodes[position].Previous = Predecessor(POSITION_NONE)
		subject.Nodes[position].Used = false
	}
	subject.Element_Count = COUNT_MINIMUM
}

// Readies a zero List value at its first use. A zero pool threads no free chain, and the free
// sentinel of a ready list never points at the ring sentinel, thus that link states the
// difference.
func lazy_initialize[Value any](subject *List[Value]) {
	List_Invariants(subject, "lazy_initialize.subject")
	if subject.Nodes[FREE_POSITION].Next == ROOT_POSITION {
		Initialize(subject)
	}
}

// Enforces that a handle names a node that the element ring holds.
func enforce_live[Value any](subject *List[Value], position Position) {
	Position_Invariants(position, "enforce_live.position")
	List_Invariants(subject, "enforce_live.subject")
	invariant.Always(
		int(position) < len(subject.Nodes),
		"A live handle names a node of the pool.",
	)
	invariant.Always(
		bool(subject.Nodes[position].Used),
		"A live handle names a node that the element ring holds.",
	)
}

// Element_Count returns the element count of a list.
func Element_Count[Value any](subject *List[Value]) (count Count) {
	defer func() { Count_Invariants(count, "element_count.count") }()
	List_Invariants(subject, "element_count.subject")
	return subject.Element_Count
}

// Front returns the first handle of a list, or POSITION_NONE for an empty list.
func Front[Value any](subject *List[Value]) (position Position) {
	defer func() { Position_Invariants(position, "front.position") }()
	List_Invariants(subject, "front.subject")
	if subject.Element_Count == COUNT_MINIMUM {
		return POSITION_NONE
	}
	return Position(subject.Nodes[ROOT_POSITION].Next)
}

// Back returns the final handle of a list, or POSITION_NONE for an empty list.
func Back[Value any](subject *List[Value]) (position Position) {
	defer func() { Position_Invariants(position, "back.position") }()
	List_Invariants(subject, "back.subject")
	if subject.Element_Count == COUNT_MINIMUM {
		return POSITION_NONE
	}
	return Position(subject.Nodes[ROOT_POSITION].Previous)
}

// Next returns the following handle, or POSITION_NONE at the back of the list.
func Next[Value any](subject *List[Value], position Position) (successor Position) {
	defer func() { Position_Invariants(successor, "next.successor") }()
	Position_Invariants(position, "next.position")
	List_Invariants(subject, "next.subject")
	enforce_live(subject, position)
	successor = Position(subject.Nodes[position].Next)
	if successor == ROOT_POSITION {
		return POSITION_NONE
	}
	return successor
}

// Previous returns the preceding handle, or POSITION_NONE at the front of the list.
func Previous[Value any](subject *List[Value], position Position) (predecessor Position) {
	defer func() { Position_Invariants(predecessor, "previous.predecessor") }()
	Position_Invariants(position, "previous.position")
	List_Invariants(subject, "previous.subject")
	enforce_live(subject, position)
	predecessor = Position(subject.Nodes[position].Previous)
	if predecessor == ROOT_POSITION {
		return POSITION_NONE
	}
	return predecessor
}

// Value_At returns the value that one handle holds.
func Value_At[Value any](subject *List[Value], position Position) (value Value) {
	Position_Invariants(position, "value_at.position")
	List_Invariants(subject, "value_at.subject")
	enforce_live(subject, position)
	return subject.Nodes[position].Value
}

// Takes one node from the free chain, or grows the pool by one node.
func allocate[Value any](subject *List[Value]) (position Element_Position) {
	defer func() { Element_Position_Invariants(position, "allocate.position") }()
	List_Invariants(subject, "allocate.subject")
	invariant.Always(
		subject.Element_Count < COUNT_MAXIMUM,
		"A list insertion keeps room for the new element.",
	)
	free := subject.Nodes[FREE_POSITION].Next
	subject.Nodes[FREE_POSITION].Next = subject.Nodes[free].Next
	return Element_Position(free)
}

// Puts one node on the free chain, so a later insertion reuses its storage. The pool is the
// whole input, because a released node leaves the element ring and states nothing about the
// count that the ring keeps.
func release[Value any](
	nodes *[NODE_COUNT_MAXIMUM]Element[Value], position Element_Position,
) {
	Element_Position_Invariants(position, "release.position")
	nodes[position].Used = false
	nodes[position].Previous = Predecessor(POSITION_NONE)
	nodes[position].Next = nodes[FREE_POSITION].Next
	nodes[FREE_POSITION].Next = Successor(position)
}

// Joins one node into the element ring after a mark.
func link[Value any](
	subject *List[Value], position Element_Position, mark Mark_Position,
) {
	Element_Position_Invariants(position, "link.position")
	Mark_Position_Invariants(mark, "link.mark")
	List_Invariants(subject, "link.subject")
	successor := subject.Nodes[mark].Next
	subject.Nodes[position].Previous = Predecessor(mark)
	subject.Nodes[position].Next = successor
	subject.Nodes[mark].Next = Successor(position)
	subject.Nodes[successor].Previous = Predecessor(position)
}

// Takes one node out of the element ring and closes the gap.
func unlink[Value any](subject *List[Value], position Element_Position) {
	Element_Position_Invariants(position, "unlink.position")
	List_Invariants(subject, "unlink.subject")
	predecessor := subject.Nodes[position].Previous
	successor := subject.Nodes[position].Next
	subject.Nodes[predecessor].Next = successor
	subject.Nodes[successor].Previous = predecessor
}

// Adds one value after a mark of the element ring and returns its new handle.
func insert[Value any](
	subject *List[Value], value Value, mark Mark_Position,
) (position Element_Position) {
	defer func() { Element_Position_Invariants(position, "insert.position") }()
	Mark_Position_Invariants(mark, "insert.mark")
	List_Invariants(subject, "insert.subject")
	position = allocate(subject)
	subject.Nodes[position].Value = value
	subject.Nodes[position].Used = true
	link(subject, position, mark)
	subject.Element_Count++
	return position
}

// Push_Front adds one value at the front of a list and returns its new handle.
func Push_Front[Value any](subject *List[Value], value Value) (position Element_Position) {
	defer func() { Element_Position_Invariants(position, "push_front.position") }()
	List_Invariants(subject, "push_front.subject")
	lazy_initialize(subject)
	return insert(subject, value, ROOT_POSITION)
}

// Push_Back adds one value at the back of a list and returns its new handle.
func Push_Back[Value any](subject *List[Value], value Value) (position Element_Position) {
	defer func() { Element_Position_Invariants(position, "push_back.position") }()
	List_Invariants(subject, "push_back.subject")
	lazy_initialize(subject)
	mark := Mark_Position(subject.Nodes[ROOT_POSITION].Previous)
	return insert(subject, value, mark)
}

// Insert_Before adds one value before a mark and returns its new handle.
func Insert_Before[Value any](
	subject *List[Value], value Value, mark Position,
) (position Element_Position) {
	defer func() { Element_Position_Invariants(position, "insert_before.position") }()
	Position_Invariants(mark, "insert_before.mark")
	List_Invariants(subject, "insert_before.subject")
	enforce_live(subject, mark)
	before := Mark_Position(subject.Nodes[mark].Previous)
	return insert(subject, value, before)
}

// Insert_After adds one value after a mark and returns its new handle.
func Insert_After[Value any](
	subject *List[Value], value Value, mark Position,
) (position Element_Position) {
	defer func() { Element_Position_Invariants(position, "insert_after.position") }()
	Position_Invariants(mark, "insert_after.mark")
	List_Invariants(subject, "insert_after.subject")
	enforce_live(subject, mark)
	return insert(subject, value, Mark_Position(mark))
}

// Remove takes one handle out of its list and returns the value that the handle held.
func Remove[Value any](subject *List[Value], position Position) (value Value) {
	Position_Invariants(position, "remove.position")
	List_Invariants(subject, "remove.subject")
	enforce_live(subject, position)
	value = subject.Nodes[position].Value
	// The count drops before the two node operations, so each one states the count of the
	// list that it leaves, and an empty result reaches their declared count minimum.
	subject.Element_Count--
	unlink(subject, Element_Position(position))
	release(&subject.Nodes, Element_Position(position))
	return value
}

// Move_To_Front moves one handle to the front of its list.
func Move_To_Front[Value any](subject *List[Value], position Position) {
	Position_Invariants(position, "move_to_front.position")
	List_Invariants(subject, "move_to_front.subject")
	enforce_live(subject, position)
	unlink(subject, Element_Position(position))
	link(subject, Element_Position(position), ROOT_POSITION)
}

// Move_To_Back moves one handle to the back of its list.
func Move_To_Back[Value any](subject *List[Value], position Position) {
	Position_Invariants(position, "move_to_back.position")
	List_Invariants(subject, "move_to_back.subject")
	enforce_live(subject, position)
	mark := Mark_Position(subject.Nodes[ROOT_POSITION].Previous)
	// A node cannot follow itself, thus the node that is already at the back stays there.
	if int(mark) == int(position) {
		return
	}
	unlink(subject, Element_Position(position))
	link(subject, Element_Position(position), mark)
}

// Move_Before moves one handle to the place before a mark.
func Move_Before[Value any](subject *List[Value], position Position, mark Position) {
	Position_Invariants(position, "move_before.position")
	Position_Invariants(mark, "move_before.mark")
	List_Invariants(subject, "move_before.subject")
	enforce_live(subject, position)
	enforce_live(subject, mark)
	before := Mark_Position(subject.Nodes[mark].Previous)
	if int(before) == int(position) {
		return
	}
	unlink(subject, Element_Position(position))
	link(subject, Element_Position(position), before)
}

// Move_After moves one handle to the place after a mark.
func Move_After[Value any](subject *List[Value], position Position, mark Position) {
	Position_Invariants(position, "move_after.position")
	Position_Invariants(mark, "move_after.mark")
	List_Invariants(subject, "move_after.subject")
	enforce_live(subject, position)
	enforce_live(subject, mark)
	if mark == position {
		return
	}
	unlink(subject, Element_Position(position))
	link(subject, Element_Position(position), Mark_Position(mark))
}

// Push_Back_List adds a copy of another list at the back. The other list can be this list.
func Push_Back_List[Value any](subject *List[Value], other *List[Value]) {
	List_Invariants(subject, "push_back_list.subject")
	List_Invariants(other, "push_back_list.other")
	lazy_initialize(subject)
	enforce_copy(subject.Element_Count, other.Element_Count)
	remainder_count := int(other.Element_Count)
	position := Front(other)
	for remainder_count > 0 {
		Push_Back(subject, Value_At(other, position))
		position = Next(other, position)
		remainder_count--
	}
}

// Push_Front_List adds a copy of another list at the front. The other list can be this list.
func Push_Front_List[Value any](subject *List[Value], other *List[Value]) {
	List_Invariants(subject, "push_front_list.subject")
	List_Invariants(other, "push_front_list.other")
	lazy_initialize(subject)
	enforce_copy(subject.Element_Count, other.Element_Count)
	remainder_count := int(other.Element_Count)
	position := Back(other)
	for remainder_count > 0 {
		Push_Front(subject, Value_At(other, position))
		position = Previous(other, position)
		remainder_count--
	}
}

// Enforces that a copy of another list fits. The check runs before the first insertion, so a
// list that cannot hold the whole copy keeps its own elements.
func enforce_copy(subject_count Count, other_count Count) {
	Count_Invariants(subject_count, "enforce_copy.subject_count")
	Count_Invariants(other_count, "enforce_copy.other_count")
	invariant.Always(
		int(subject_count)+int(other_count) <= COUNT_MAXIMUM,
		"A list copy keeps room for every added element.",
	)
}
