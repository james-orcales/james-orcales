package ring

import (
	"testing"

	"local/james-orcales/shared/testify"
)

// Test_Standard_Library_Corner_Cases ports the upstream TestCornerCases. An empty ring, a link
// to an empty ring, and an Unlink of zero nodes each change nothing.
func Test_Standard_Library_Corner_Cases(t *testing.T) {
	t.Parallel()
	subject := ready_pool()
	verify_ring(t, subject, POSITION_NONE, 0, 0)
	single := New(subject, 1)
	verify_ring(t, subject, single, 1, 0)
	Link(subject, node(single), POSITION_NONE)
	verify_ring(t, subject, single, 1, 0)
	Link(subject, node(single), POSITION_NONE)
	verify_ring(t, subject, single, 1, 0)
	Unlink(subject, node(single), 0)
	verify_ring(t, subject, single, 1, 0)
}

// Test_Standard_Library_New ports the upstream TestNew. Each count makes a ring of that count,
// and each value that a caller writes stays in the ring.
func Test_Standard_Library_New(t *testing.T) {
	t.Parallel()
	for element_count := range 10 {
		subject := ready_pool()
		position := New(subject, Count(element_count))
		verify_ring(t, subject, position, Count(element_count), 0)
	}
	for element_count := range 10 {
		subject, position := numbered_ring(element_count)
		verify_ring(t, subject, position,
			Count(element_count), triangle_sum(element_count))
	}
}

// Test_Standard_Library_Link ports the upstream TestLink1. A link of two rings joins them, a
// link to the neighbor handle changes nothing, and a link of one handle to itself divides
// its ring.
func Test_Standard_Library_Link(t *testing.T) {
	t.Parallel()
	subject, first := numbered_ring(1)
	second := New(subject, 1)
	joined := Link(subject, node(first), second)
	verify_ring(t, subject, handle(joined), 2, 1)
	testify.Equal(t, node(first), joined,
		"a link of two rings of one node must return the first handle")
	neighbor := Link(subject, joined, handle(Next(subject, joined)))
	verify_ring(t, subject, handle(neighbor), 2, 1)
	testify.Equal(t, Next(subject, joined), neighbor,
		"a link to the neighbor handle must return that handle")
	divided := Link(subject, neighbor, handle(neighbor))
	verify_ring(t, subject, handle(divided), 1, 1)
	verify_ring(t, subject, handle(neighbor), 1, 0)
}

// Test_Standard_Library_Link_Rings ports the upstream TestLink2. A link joins rings of every
// count, and a link to an empty ring changes nothing.
func Test_Standard_Library_Link_Rings(t *testing.T) {
	t.Parallel()
	subject := ready_pool()
	first := node(New(subject, 1))
	Set_Value(subject, first, 42)
	second := New(subject, 1)
	Set_Value(subject, node(second), 77)
	Link(subject, first, POSITION_NONE)
	verify_ring(t, subject, handle(first), 1, 42)
	Link(subject, first, second)
	verify_ring(t, subject, handle(first), 2, 42+77)
	other_subject, ten := numbered_ring(10)
	Link(other_subject, node(ten), POSITION_NONE)
	verify_ring(t, other_subject, ten, 10, triangle_sum(10))
}

// Test_Standard_Library_Link_Growth ports the upstream TestLink3. Each link adds the whole
// count of the other ring.
func Test_Standard_Library_Link_Growth(t *testing.T) {
	t.Parallel()
	subject := ready_pool()
	position := node(New(subject, 1))
	total_count := Count(1)
	for added_index := 1; added_index < 10; added_index++ {
		added_count := Count(added_index)
		total_count += added_count
		Link(subject, position, New(subject, added_count))
		verify_ring(t, subject, handle(position), total_count, 0)
	}
}

// Test_Standard_Library_Unlink ports the upstream TestUnlink. A count that is not below the
// ring count causes a panic, where the standard library folds that count into one lap.
func Test_Standard_Library_Unlink(t *testing.T) {
	t.Parallel()
	subject, ten := numbered_ring(10)
	whole_sum := triangle_sum(10)
	verify_ring(t, subject, ten, 10, whole_sum)
	moved := Move(subject, node(ten), 6)
	verify_ring(t, subject, handle(moved), 10, whole_sum)
	empty := Unlink(subject, node(ten), 0)
	verify_ring(t, subject, empty, 0, 0)
	single := Unlink(subject, node(ten), 1)
	verify_ring(t, subject, single, 1, 2)
	verify_ring(t, subject, ten, 9, whole_sum-2)
	testify.True(t, raises(func() { Unlink(subject, node(ten), 9) }),
		"Unlink must reject a count that is not below the ring count")
}

// Test_Standard_Library_Link_Unlink ports the upstream TestLinkUnlink. An Unlink and a Link of
// the taken part return the ring to its own count.
func Test_Standard_Library_Link_Unlink(t *testing.T) {
	t.Parallel()
	for element_index := 1; element_index < 4; element_index++ {
		element_count := Count(element_index)
		subject := ready_pool()
		position := node(New(subject, element_count))
		for taken_index := range element_index {
			taken_count := Count(taken_index)
			taken := Unlink(subject, position, taken_count)
			verify_ring(t, subject, taken, taken_count, 0)
			verify_ring(t, subject, handle(position), element_count-taken_count, 0)
			Link(subject, position, taken)
			verify_ring(t, subject, handle(position), element_count, 0)
		}
	}
}

// Test_Standard_Library_Move_Empty_Ring ports the upstream TestMoveEmptyRing. A Move of an
// empty ring causes a panic, where the standard library readies the ring instead.
func Test_Standard_Library_Move_Empty_Ring(t *testing.T) {
	t.Parallel()
	subject := ready_pool()
	testify.True(t, raises(func() { Move(subject, node(POSITION_NONE), 1) }),
		"Move must reject the handle of an empty ring")
}

// Fails when a ring does not hold one count of nodes and one sum of values. The upstream
// verify reads the neighbor fields, which the walks state here.
func verify_ring(
	t *testing.T, subject *Pool, position Position,
	wanted_count Count, wanted_sum int,
) {
	t.Helper()
	testify.Equal(t, wanted_count, Element_Count(subject, position),
		"the node count is wrong")
	seen_count := 0
	seen_sum := 0
	For_Each(subject, position, func(value Value) {
		seen_count++
		seen_sum += int(value)
	})
	testify.Equal(t, int(wanted_count), seen_count, "the forward walk count is wrong")
	testify.Equal(t, wanted_sum, seen_sum, "the forward walk sum is wrong")
	if position == POSITION_NONE {
		return
	}
	verify_connections(t, subject, node(position), wanted_count)
}

// Fails when a backward walk does not undo a forward walk, or when Move does not wrap at the
// ring count.
func verify_connections(
	t *testing.T, subject *Pool, position Element_Position, wanted_count Count,
) {
	t.Helper()
	testify.Equal(t, position, Previous(subject, Next(subject, position)),
		"Previous must undo Next")
	testify.Equal(t, position, Move(subject, position, 0),
		"an offset of zero must stay at its own handle")
	lap := Offset(wanted_count)
	testify.Equal(t, position, Move(subject, position, lap),
		"one forward lap must return its own handle")
	testify.Equal(t, position, Move(subject, position, -lap),
		"one backward lap must return its own handle")
	for extra := range 10 {
		folded := Offset(extra) % lap
		testify.Equal(t,
			Move(subject, position, folded),
			Move(subject, position, lap+Offset(extra)),
			"a forward Move must wrap at the ring count")
		testify.Equal(t,
			Move(subject, position, -folded),
			Move(subject, position, -lap-Offset(extra)),
			"a backward Move must wrap at the ring count")
	}
}

// Makes a ring of one count whose values run from one to that count.
func numbered_ring(element_count int) (subject *Pool, position Position) {
	subject = ready_pool()
	position = New(subject, Count(element_count))
	if position == POSITION_NONE {
		return subject, position
	}
	walk := node(position)
	for value := 1; value <= element_count; value++ {
		Set_Value(subject, walk, Value(value))
		walk = Next(subject, walk)
	}
	return subject, position
}

// Makes empty ring pool over caller-owned storage.
func ready_pool() (subject *Pool) {
	subject = &Pool{Nodes: make(Nodes, NODE_COUNT_MAXIMUM)}
	Initialize(subject)
	return subject
}

// Returns the sum of one to one count.
func triangle_sum(element_count int) (sum int) {
	return (element_count*element_count + element_count) / 2
}

// Reads one handle as a live node.
func node(position Position) (target Element_Position) {
	return Element_Position(position)
}

// Reads one live node as a handle that admits an empty ring.
func handle(target Element_Position) (position Position) {
	return Position(target)
}

// Runs an action and reports whether it caused a panic.
func raises(action func()) (raised bool) {
	defer func() {
		if recover() != nil {
			raised = true
		}
	}()
	action()
	return false
}
