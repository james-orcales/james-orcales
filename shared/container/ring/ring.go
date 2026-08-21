// Package ring supplies the Go standard-library circular list through the repository naming
// and assertion boundaries. A pool owns every node and joins its nodes by position, not by
// pointer, because a pointer node states itself as its own neighbor and no assertion tree can
// hold that cycle. Each operation is a free function, thus a ring carries no method set.
//
// One pool holds many rings, because Link and Unlink move nodes between rings and both rings
// must come from one storage. A pool holds its whole storage in one array, thus its size is
// the same for an empty pool and for a full one. Make a pool behind a pointer and do not copy
// the value.
package ring

import (
	"local/james-orcales/shared/invariant/default"
	"local/james-orcales/shared/slices"
)

// ELEMENT_COUNT_MAXIMUM caps each pool that crosses this deterministic package boundary. It
// matches the repository slice boundary, so one common element count applies.
const ELEMENT_COUNT_MAXIMUM = slices.SLICE_COUNT_MAXIMUM

// SENTINEL_COUNT is the node count that a pool reserves for its free chain head.
const SENTINEL_COUNT = 1

// COUNT_MINIMUM is the node count of an empty ring.
const COUNT_MINIMUM = 0

// COUNT_MAXIMUM is the largest admitted node count.
const COUNT_MAXIMUM = ELEMENT_COUNT_MAXIMUM

// NODE_COUNT_MAXIMUM is the node count of a full pool, including its sentinel.
const NODE_COUNT_MAXIMUM = COUNT_MAXIMUM + SENTINEL_COUNT

// POSITION_NONE is the handle of an empty ring.
const POSITION_NONE Position = -1

// FREE_POSITION holds the sentinel of the free chain.
const FREE_POSITION = 0

// FIRST_ELEMENT_POSITION is the first position that a caller value can occupy.
const FIRST_ELEMENT_POSITION = FREE_POSITION + 1

// POSITION_MINIMUM is POSITION_NONE, which every handle and neighbor field can hold.
const POSITION_MINIMUM = int(POSITION_NONE)

// POSITION_MAXIMUM is the final position of a full pool.
const POSITION_MAXIMUM = NODE_COUNT_MAXIMUM - 1

// OFFSET_MINIMUM is the largest backward walk that Move admits.
const OFFSET_MINIMUM = -COUNT_MAXIMUM

// OFFSET_MAXIMUM is the largest forward walk that Move admits.
const OFFSET_MAXIMUM = COUNT_MAXIMUM

// Position is one node handle of a pool, or POSITION_NONE for an empty ring. The free-chain
// sentinel is no handle, thus its position is a hole.
type Position int

// Position_Invariants states the complete node-handle domain.
func Position_Invariants(value Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Holed_Int(
			int(value), POSITION_MINIMUM, POSITION_MAXIMUM,
			FREE_POSITION, FREE_POSITION, FREE_POSITION, FREE_POSITION,
		).
		Ensure()
}

// Element_Position is one node that a ring holds, thus it is never POSITION_NONE and never
// the sentinel. An operation that reads or writes one node takes this handle, and an
// operation that admits an empty ring takes a Position.
type Element_Position int

// Element_Position_Invariants states the domain of one live ring node.
func Element_Position_Invariants(value Element_Position, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), FIRST_ELEMENT_POSITION, POSITION_MAXIMUM).
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

// Count is a node count of one ring or of the free chain.
type Count int

// Count_Invariants states the complete node-count domain.
func Count_Invariants(value Count, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), COUNT_MINIMUM, COUNT_MAXIMUM).
		Ensure()
}

// Offset is a walk length, backward when it is negative and forward when it is positive.
type Offset int

// Offset_Invariants states the complete walk domain.
func Offset_Invariants(value Offset, namespace invariant.Namespace) {
	invariant.Tree(value, namespace).
		Range_Int(int(value), OFFSET_MINIMUM, OFFSET_MAXIMUM).
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

// Visitor_Function reads the value of one node.
type Visitor_Function[Value any] func(value Value)

// Element is one node of the storage of a pool.
type Element[Value any] struct {
	// Next is the following node of its ring, or the following node of the free chain while
	// no ring holds this node.
	Next Successor
	// Previous is the preceding node of its ring. A free node holds POSITION_NONE.
	Previous Predecessor
	// Used states whether a ring holds this node, thus an operation on a handle that Release
	// already took causes a panic.
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

// Pool is the storage of every ring. Its zero value is a ready pool.
type Pool[Value any] struct {
	// Nodes holds the free-chain sentinel at FREE_POSITION and every caller node after it.
	// The storage is one array, because a slice field carries no defined type that an
	// assertion tree can own.
	Nodes [NODE_COUNT_MAXIMUM]Element[Value]
	// Free_Count is the node count that the free chain holds.
	Free_Count Count
}

// Pool_Invariants states the free-chain count of one pool. The subject is a pointer, because
// a pool holds its whole storage and a value parameter would copy it at each assertion.
func Pool_Invariants[Value any](
	subject *Pool[Value], namespace invariant.Namespace,
) {
	Count_Invariants(subject.Free_Count, namespace)
}

// Initialize readies a pool and puts every node on its free chain.
func Initialize[Value any](subject *Pool[Value]) {
	Pool_Invariants(subject, "initialize.subject")
	subject.Nodes[POSITION_MAXIMUM].Next = Successor(POSITION_NONE)
	subject.Nodes[POSITION_MAXIMUM].Previous = Predecessor(POSITION_NONE)
	subject.Nodes[POSITION_MAXIMUM].Used = false
	for position := FREE_POSITION; position < POSITION_MAXIMUM; position++ {
		subject.Nodes[position].Next = Successor(position + 1)
		subject.Nodes[position].Previous = Predecessor(POSITION_NONE)
		subject.Nodes[position].Used = false
	}
	subject.Free_Count = COUNT_MAXIMUM
}

// Readies a zero Pool value at its first use. A zero pool threads no free chain, and the free
// sentinel of a ready pool never points at itself, thus that link states the difference.
func lazy_initialize[Value any](subject *Pool[Value]) {
	Pool_Invariants(subject, "lazy_initialize.subject")
	if subject.Nodes[FREE_POSITION].Next == FREE_POSITION {
		Initialize(subject)
	}
}

// Enforces that a handle names a node that a ring holds. The handle domain already excludes
// the sentinel and every position outside the pool, thus only use remains.
func enforce_live[Value any](subject *Pool[Value], position Element_Position) {
	Element_Position_Invariants(position, "enforce_live.position")
	Pool_Invariants(subject, "enforce_live.subject")
	invariant.Always(
		bool(subject.Nodes[position].Used),
		"A live handle names a node that a ring holds.",
	)
}

// Takes one node off the free chain. The caller owns the free count, because the count of a
// pool that keeps one node back states nothing about the chain that holds it.
func allocate[Value any](
	nodes *[NODE_COUNT_MAXIMUM]Element[Value],
) (position Element_Position) {
	defer func() { Element_Position_Invariants(position, "allocate.position") }()
	vacancy := nodes[FREE_POSITION].Next
	invariant.Always(
		vacancy != Successor(POSITION_NONE),
		"A pool allocation reads one node from its free chain.",
	)
	nodes[FREE_POSITION].Next = nodes[vacancy].Next
	nodes[vacancy].Used = true
	return Element_Position(vacancy)
}

// Puts one node back on the free chain.
func deallocate[Value any](
	nodes *[NODE_COUNT_MAXIMUM]Element[Value], position Element_Position,
) {
	Element_Position_Invariants(position, "deallocate.position")
	nodes[position].Used = false
	nodes[position].Previous = Predecessor(POSITION_NONE)
	nodes[position].Next = nodes[FREE_POSITION].Next
	nodes[FREE_POSITION].Next = Successor(position)
}

// Puts one free-standing node into a ring after a mark.
func splice[Value any](
	nodes *[NODE_COUNT_MAXIMUM]Element[Value],
	position Element_Position,
	mark Element_Position,
) {
	Element_Position_Invariants(position, "splice.position")
	Element_Position_Invariants(mark, "splice.mark")
	successor := nodes[mark].Next
	nodes[position].Previous = Predecessor(mark)
	nodes[position].Next = successor
	nodes[mark].Next = Successor(position)
	nodes[successor].Previous = Predecessor(position)
}

// New takes one count of nodes from the pool and closes them into one ring.
func New[Value any](subject *Pool[Value], count Count) (position Position) {
	defer func() { Position_Invariants(position, "new.position") }()
	Count_Invariants(count, "new.count")
	Pool_Invariants(subject, "new.subject")
	lazy_initialize(subject)
	invariant.Always(
		count <= subject.Free_Count,
		"A new ring takes no more nodes than the pool holds free.",
	)
	if count == COUNT_MINIMUM {
		return POSITION_NONE
	}
	first := allocate(&subject.Nodes)
	subject.Free_Count--
	subject.Nodes[first].Next = Successor(first)
	subject.Nodes[first].Previous = Predecessor(first)
	for remainder_count := count - 1; remainder_count > COUNT_MINIMUM; remainder_count-- {
		added := allocate(&subject.Nodes)
		subject.Free_Count--
		splice(&subject.Nodes, added, Element_Position(subject.Nodes[first].Previous))
	}
	return Position(first)
}

// Next returns the following handle of the ring.
func Next[Value any](
	subject *Pool[Value], position Element_Position,
) (successor Element_Position) {
	defer func() { Element_Position_Invariants(successor, "next.successor") }()
	Element_Position_Invariants(position, "next.position")
	Pool_Invariants(subject, "next.subject")
	enforce_live(subject, position)
	return Element_Position(subject.Nodes[position].Next)
}

// Previous returns the preceding handle of the ring.
func Previous[Value any](
	subject *Pool[Value], position Element_Position,
) (predecessor Element_Position) {
	defer func() { Element_Position_Invariants(predecessor, "previous.predecessor") }()
	Element_Position_Invariants(position, "previous.position")
	Pool_Invariants(subject, "previous.subject")
	enforce_live(subject, position)
	return Element_Position(subject.Nodes[position].Previous)
}

// Value_At returns the value that one handle holds.
func Value_At[Value any](
	subject *Pool[Value], position Element_Position,
) (value Value) {
	Element_Position_Invariants(position, "value_at.position")
	Pool_Invariants(subject, "value_at.subject")
	enforce_live(subject, position)
	return subject.Nodes[position].Value
}

// Set_Value writes the value that one handle holds.
func Set_Value[Value any](
	subject *Pool[Value], position Element_Position, value Value,
) {
	Element_Position_Invariants(position, "set_value.position")
	Pool_Invariants(subject, "set_value.subject")
	enforce_live(subject, position)
	subject.Nodes[position].Value = value
}

// Move walks one offset of nodes and returns the handle that it reaches.
func Move[Value any](
	subject *Pool[Value], position Element_Position, offset Offset,
) (result Element_Position) {
	defer func() { Element_Position_Invariants(result, "move.result") }()
	Element_Position_Invariants(position, "move.position")
	Offset_Invariants(offset, "move.offset")
	Pool_Invariants(subject, "move.subject")
	enforce_live(subject, position)
	result = position
	for backward := offset; backward < 0; backward++ {
		result = Element_Position(subject.Nodes[result].Previous)
	}
	for forward := offset; forward > 0; forward-- {
		result = Element_Position(subject.Nodes[result].Next)
	}
	return result
}

// Element_Count counts the nodes of one ring, and it costs one lap.
func Element_Count[Value any](subject *Pool[Value], position Position) (count Count) {
	defer func() { Count_Invariants(count, "element_count.count") }()
	Position_Invariants(position, "element_count.position")
	Pool_Invariants(subject, "element_count.subject")
	if position == POSITION_NONE {
		return COUNT_MINIMUM
	}
	enforce_live(subject, Element_Position(position))
	count = 1
	for walk := Position(subject.Nodes[position].Next); walk != position; {
		walk = Position(subject.Nodes[walk].Next)
		count++
	}
	return count
}

// For_Each calls one visitor with the value of each node, in forward order.
func For_Each[Value any](
	subject *Pool[Value], position Position, visitor Visitor_Function[Value],
) {
	Position_Invariants(position, "for_each.position")
	Pool_Invariants(subject, "for_each.subject")
	if position == POSITION_NONE {
		return
	}
	enforce_live(subject, Element_Position(position))
	visitor(subject.Nodes[position].Value)
	for walk := Position(subject.Nodes[position].Next); walk != position; {
		visitor(subject.Nodes[walk].Value)
		walk = Position(subject.Nodes[walk].Next)
	}
}

// Link joins the ring of one handle to the ring of another handle, and it returns the handle
// that followed the first one.
func Link[Value any](
	subject *Pool[Value], position Element_Position, other Position,
) (result Element_Position) {
	defer func() { Element_Position_Invariants(result, "link.result") }()
	Element_Position_Invariants(position, "link.position")
	Position_Invariants(other, "link.other")
	Pool_Invariants(subject, "link.subject")
	enforce_live(subject, position)
	result = Element_Position(subject.Nodes[position].Next)
	if other == POSITION_NONE {
		return result
	}
	enforce_live(subject, Element_Position(other))
	final := subject.Nodes[other].Previous
	subject.Nodes[position].Next = Successor(other)
	subject.Nodes[other].Previous = Predecessor(position)
	subject.Nodes[result].Previous = Predecessor(final)
	subject.Nodes[final].Next = Successor(result)
	return result
}

// Unlink takes one count of nodes out of a ring, starting after the handle, and returns the
// handle of the taken part.
func Unlink[Value any](
	subject *Pool[Value], position Element_Position, count Count,
) (result Position) {
	defer func() { Position_Invariants(result, "unlink.result") }()
	Element_Position_Invariants(position, "unlink.position")
	Count_Invariants(count, "unlink.count")
	Pool_Invariants(subject, "unlink.subject")
	enforce_live(subject, position)
	invariant.Always(
		count < Element_Count(subject, Position(position)),
		"An unlinked count leaves at least one node in the ring.",
	)
	if count == COUNT_MINIMUM {
		return POSITION_NONE
	}
	divided := Move(subject, position, Offset(count)+1)
	return Position(Link(subject, position, Position(divided)))
}

// Release returns the nodes of one ring to the pool, so a later New reuses them.
func Release[Value any](subject *Pool[Value], position Position) {
	Position_Invariants(position, "release.position")
	Pool_Invariants(subject, "release.subject")
	if position == POSITION_NONE {
		return
	}
	walk := Element_Position(position)
	enforce_live(subject, walk)
	for remainder_count := Element_Count(subject, position); remainder_count > 0; {
		successor := Element_Position(subject.Nodes[walk].Next)
		deallocate(&subject.Nodes, walk)
		subject.Free_Count++
		walk = successor
		remainder_count--
	}
}
