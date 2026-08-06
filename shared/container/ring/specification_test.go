package ring_test

import (
	"testing"

	"local/james-orcales/shared/container/ring"
	"local/james-orcales/shared/testify"
)

// Test_Construction verifies Initialize, the zero Pool value, and each New count.
func Test_Construction(t *testing.T) {
	t.Parallel()
	subject := &ring.Pool[int]{}
	testify.Equal(t, ring.POSITION_NONE, ring.New(subject, 0),
		"New must return POSITION_NONE for a count of zero")
	single := node(ring.New(subject, 1))
	testify.Equal(t, ring.Count(1), ring.Element_Count(subject, handle(single)),
		"New must make a ring of its count")
	testify.Equal(t, single, ring.Next(subject, single),
		"a ring of one node follows itself")
	ready := &ring.Pool[int]{}
	ring.Initialize(ready)
	testify.Equal(t, ring.Count(5), ring.Element_Count(ready, ring.New(ready, 5)),
		"Initialize must ready a pool")
}

// Test_Handles verifies Value_At, Set_Value, and the empty-ring handle.
func Test_Handles(t *testing.T) {
	t.Parallel()
	subject, position := filled_ring(3)
	testify.Equal(t, 0, ring.Value_At(subject, node(position)),
		"Value_At must return the stored value")
	ring.Set_Value(subject, node(position), 9)
	testify.Equal(t, 9, ring.Value_At(subject, node(position)),
		"Set_Value must write the stored value")
	testify.Equal(t, ring.Count(0), ring.Element_Count(subject, ring.POSITION_NONE),
		"an empty ring holds no node")
}

// Test_Traversal verifies Next, Previous, and Element_Count around one ring.
func Test_Traversal(t *testing.T) {
	t.Parallel()
	subject, position := filled_ring(3)
	second := ring.Next(subject, node(position))
	testify.Equal(t, 1, ring.Value_At(subject, second), "Next must walk forward")
	testify.Equal(t, node(position), ring.Previous(subject, second),
		"Previous must walk backward")
	third := ring.Next(subject, second)
	testify.Equal(t, node(position), ring.Next(subject, third),
		"Next must close the ring")
	testify.Equal(t, ring.Count(3), ring.Element_Count(subject, position),
		"Element_Count must count one lap")
}

// Test_Movement verifies Move in both directions and over one full lap.
func Test_Movement(t *testing.T) {
	t.Parallel()
	subject, position := filled_ring(4)
	testify.Equal(t, node(position), ring.Move(subject, node(position), 0),
		"an offset of zero must return its own handle")
	testify.Equal(t, 2, ring.Value_At(subject, ring.Move(subject, node(position), 2)),
		"a positive offset must walk forward")
	testify.Equal(t, 3, ring.Value_At(subject, ring.Move(subject, node(position), -1)),
		"a negative offset must walk backward")
	testify.Equal(t, node(position), ring.Move(subject, node(position), 4),
		"one full lap must return its own handle")
}

// Test_Linking verifies that Link joins two rings and that two handles of one ring divide it.
func Test_Link(t *testing.T) {
	t.Parallel()
	subject, first := filled_ring(2)
	second := node(ring.New(subject, 2))
	ring.Set_Value(subject, second, 8)
	ring.Set_Value(subject, ring.Next(subject, second), 9)
	followed := ring.Link(subject, node(first), handle(second))
	testify.Equal(t, []int{0, 8, 9, 1}, ring_values(subject, first),
		"Link must join the other ring after the handle")
	testify.Equal(t, 1, ring.Value_At(subject, followed),
		"Link must return the handle that followed the first one")
	divided_subject, whole := filled_ring(4)
	third := ring.Move(divided_subject, node(whole), 2)
	divided := ring.Link(divided_subject, node(whole), handle(third))
	testify.Equal(t, []int{0, 2, 3}, ring_values(divided_subject, whole),
		"two handles of one ring must divide that ring")
	testify.Equal(t, []int{1}, ring_values(divided_subject, handle(divided)),
		"Link must return the handle of the divided part")
}

// Test_Unlinking verifies the taken part, the smaller ring, and a count of zero.
func Test_Unlink(t *testing.T) {
	t.Parallel()
	subject, position := filled_ring(4)
	taken := ring.Unlink(subject, node(position), 2)
	testify.Equal(t, []int{0, 3}, ring_values(subject, position),
		"Unlink must leave the other nodes")
	testify.Equal(t, []int{1, 2}, ring_values(subject, taken),
		"Unlink must return the taken nodes")
	testify.Equal(t, ring.POSITION_NONE, ring.Unlink(subject, node(position), 0),
		"a count of zero must return POSITION_NONE")
	testify.Equal(t, []int{0, 3}, ring_values(subject, position),
		"a count of zero must change nothing")
}

// Test_Iteration verifies For_Each in forward order and over an empty ring.
func Test_Iteration(t *testing.T) {
	t.Parallel()
	subject, position := filled_ring(3)
	testify.Equal(t, []int{0, 1, 2}, ring_values(subject, position),
		"For_Each must read every value in forward order")
	testify.Count(t, ring_values(subject, ring.POSITION_NONE), 0,
		"For_Each must read no value of an empty ring")
}

// Test_Release verifies that Release returns nodes to the pool for a later New.
func Test_Release(t *testing.T) {
	t.Parallel()
	subject, position := filled_ring(3)
	ring.Release(subject, position)
	testify.Equal(t, ring.Count(3),
		ring.Element_Count(subject, ring.New(subject, 3)),
		"Release must return the nodes to the pool")
	ring.Release(subject, ring.POSITION_NONE)
}

// Test_Size_Limits verifies the largest admitted ring and the rejection of one more node.
func Test_Size_Limits(t *testing.T) {
	t.Parallel()
	subject := &ring.Pool[int]{}
	position := ring.New(subject, ring.COUNT_MAXIMUM)
	testify.Equal(t, ring.Count(ring.COUNT_MAXIMUM),
		ring.Element_Count(subject, position),
		"a pool must hold ELEMENT_COUNT_MAXIMUM nodes")
	testify.True(t, panicked(func() { ring.New(subject, 1) }),
		"a full pool must reject a new ring")
}

// Test_Domain_Errors verifies the panic for each handle that no live node holds.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	subject, taken := filled_ring(2)
	ring.Release(subject, taken)
	live_subject, live := filled_ring(2)
	panic_cases := []func(){
		func() { ring.Next(subject, node(taken)) },
		func() { ring.Previous(subject, node(taken)) },
		func() { ring.Value_At(subject, node(taken)) },
		func() { ring.Set_Value(subject, node(taken), 0) },
		func() { ring.Move(subject, node(taken), 1) },
		func() { ring.Value_At(subject, ring.FREE_POSITION) },
		func() { ring.Value_At(subject, ring.Element_Position(ring.POSITION_MAXIMUM)) },
		func() { ring.Element_Count(subject, ring.Position(ring.FREE_POSITION)) },
		func() { ring.Unlink(live_subject, node(live), 2) },
	}
	for panic_index, action := range panic_cases {
		testify.True(t, panicked(action), "domain error %d did not panic", panic_index)
	}
}

// Test_Invariant_Domains verifies each special value through a public ring operation.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	for _, state := range probe_states() {
		drive_every_operation(state)
	}
}

// Makes one fresh pool and one handle of one state, so a driven operation never reads a pool
// that an earlier operation changed.
type pool_factory func() (subject *ring.Pool[int], position ring.Position)

// Names each pool state that the declared domains need, from a zero value to a pool whose
// free chain runs from its final node.
func probe_states() (states []pool_factory) {
	return []pool_factory{
		zero_pool,
		ready_pool,
		func() (subject *ring.Pool[int], position ring.Position) {
			return filled_ring(1)
		},
		func() (subject *ring.Pool[int], position ring.Position) {
			return filled_ring(2)
		},
		func() (subject *ring.Pool[int], position ring.Position) {
			return filled_ring(3)
		},
		one_free_pool,
		two_free_pool,
		full_ring,
		released_pool,
	}
}

// Drives every operation over a fresh state, so each assertion root observes that state. A
// domain error is one of the observations, thus each call absorbs its panic.
func drive_every_operation(state pool_factory) {
	_, own := state()
	for _, position := range append(probe_positions(), own) {
		drive_every_position_operation(state, position)
	}
	for _, count := range probe_counts() {
		attempt(func() {
			subject, _ := state()
			ring.New(subject, count)
		})
	}
	attempt(func() {
		subject, _ := state()
		ring.Initialize(subject)
	})
}

// Drives every operation that takes a handle over a fresh state.
func drive_every_position_operation(state pool_factory, position ring.Position) {
	target := node(position)
	attempt(func() {
		subject, _ := state()
		ring.Next(subject, target)
	})
	attempt(func() {
		subject, _ := state()
		ring.Previous(subject, target)
	})
	attempt(func() {
		subject, _ := state()
		ring.Value_At(subject, target)
	})
	attempt(func() {
		subject, _ := state()
		ring.Set_Value(subject, target, 0)
	})
	attempt(func() {
		subject, _ := state()
		ring.Element_Count(subject, position)
	})
	attempt(func() {
		subject, _ := state()
		read_every_value(subject, position)
	})
	attempt(func() {
		subject, _ := state()
		ring.Release(subject, position)
	})
	drive_every_offset_operation(state, target)
	drive_every_count_operation(state, target)
	drive_every_link_operation(state, position)
}

// Drives Move over each offset that the declared domain names.
func drive_every_offset_operation(state pool_factory, target ring.Element_Position) {
	for _, offset := range probe_offsets() {
		attempt(func() {
			subject, _ := state()
			ring.Move(subject, target, offset)
		})
	}
}

// Drives Unlink over each count that the declared domain names.
func drive_every_count_operation(state pool_factory, target ring.Element_Position) {
	for _, count := range probe_counts() {
		attempt(func() {
			subject, _ := state()
			ring.Unlink(subject, target, count)
		})
	}
}

// Drives Link over each probe handle, on both sides of the join.
func drive_every_link_operation(state pool_factory, position ring.Position) {
	for _, other := range probe_positions() {
		attempt(func() {
			subject, _ := state()
			ring.Link(subject, node(position), other)
		})
		attempt(func() {
			subject, own := state()
			ring.Link(subject, node(other), own)
		})
	}
}

// Names each handle that the domains need, including POSITION_NONE, the sentinel, both
// interior sentinels, and both highest positions of a full pool.
func probe_positions() (positions []ring.Position) {
	return []ring.Position{
		ring.POSITION_NONE,
		ring.FREE_POSITION,
		ring.Position(ring.FIRST_ELEMENT_POSITION),
		ring.Position(ring.FIRST_ELEMENT_POSITION + 1),
		ring.Position(ring.POSITION_MAXIMUM - 1),
		ring.Position(ring.POSITION_MAXIMUM),
	}
}

// Names each offset that the declared walk domain needs.
func probe_offsets() (offsets []ring.Offset) {
	return []ring.Offset{
		ring.OFFSET_MINIMUM, -1, 0, 1, 2, ring.OFFSET_MAXIMUM,
	}
}

// Names each count that the declared count domain needs.
func probe_counts() (counts []ring.Count) {
	return []ring.Count{0, 1, 2, ring.COUNT_MAXIMUM}
}

// Makes a zero Pool value, which threads no free chain.
func zero_pool() (subject *ring.Pool[int], position ring.Position) {
	return &ring.Pool[int]{}, ring.POSITION_NONE
}

// Makes a ready pool that holds no ring.
func ready_pool() (subject *ring.Pool[int], position ring.Position) {
	subject = &ring.Pool[int]{}
	ring.Initialize(subject)
	return subject, ring.POSITION_NONE
}

// Makes a pool whose free chain holds one node.
func one_free_pool() (subject *ring.Pool[int], position ring.Position) {
	subject = &ring.Pool[int]{}
	return subject, ring.New(subject, ring.COUNT_MAXIMUM-1)
}

// Makes a pool whose free chain holds two nodes.
func two_free_pool() (subject *ring.Pool[int], position ring.Position) {
	subject = &ring.Pool[int]{}
	return subject, ring.New(subject, ring.COUNT_MAXIMUM-2)
}

// Makes a pool whose one ring holds every node.
func full_ring() (subject *ring.Pool[int], position ring.Position) {
	subject = &ring.Pool[int]{}
	return subject, ring.New(subject, ring.COUNT_MAXIMUM)
}

// Makes a pool whose free chain runs from the final node of the pool, so a later New reaches
// the highest handle.
func released_pool() (subject *ring.Pool[int], position ring.Position) {
	subject, whole := full_ring()
	ring.Release(subject, whole)
	return subject, ring.New(subject, 2)
}

// Makes a pool and one ring of increasing values.
func filled_ring(element_count int) (subject *ring.Pool[int], position ring.Position) {
	subject = &ring.Pool[int]{}
	position = ring.New(subject, ring.Count(element_count))
	walk := node(position)
	for value := range element_count {
		ring.Set_Value(subject, walk, value)
		walk = ring.Next(subject, walk)
	}
	return subject, position
}

// Reads one handle as a live node.
func node(position ring.Position) (target ring.Element_Position) {
	return ring.Element_Position(position)
}

// Reads one live node as a handle that admits an empty ring.
func handle(target ring.Element_Position) (position ring.Position) {
	return ring.Position(target)
}

// Reads every value of one ring in forward order.
func ring_values(subject *ring.Pool[int], position ring.Position) (values []int) {
	values = []int{}
	ring.For_Each(subject, position, func(value int) {
		values = append(values, value)
	})
	return values
}

// Reads every value of one ring and keeps none, so a driven For_Each states its assertions.
func read_every_value(subject *ring.Pool[int], position ring.Position) {
	seen := 0
	ring.For_Each(subject, position, func(value int) {
		seen += value
	})
}

// Runs an action and absorbs a domain panic, because a domain error is an observation.
func attempt(action func()) {
	defer func() {
		recover()
	}()
	action()
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
