package list_test

import (
	"testing"

	"local/james-orcales/shared/container/list"
	"local/james-orcales/shared/testify"
)

// Test_Construction verifies caller-owned storage and Initialize.
func Test_Construction(t *testing.T) {
	t.Parallel()
	check_list(t, empty_list(), []int{})
	subject := empty_list()
	list.Push_Back(subject, 1)
	check_list(t, subject, []int{1})
	list.Initialize(subject)
	check_list(t, subject, []int{})
}

// Test_Handles verifies POSITION_NONE at both ends and one handle across other operations.
func Test_Handles(t *testing.T) {
	t.Parallel()
	subject := empty_list()
	testify.Equal(t, list.POSITION_NONE, list.Front(subject),
		"Front must return POSITION_NONE for an empty list")
	testify.Equal(t, list.POSITION_NONE, list.Back(subject),
		"Back must return POSITION_NONE for an empty list")
	kept := handle(list.Push_Back(subject, 1))
	list.Push_Front(subject, 0)
	list.Push_Back(subject, 2)
	testify.Equal(t, list.Value(1), list.Value_At(subject, kept),
		"a handle must stay correct across other insertions")
}

// Test_Traversal verifies Front, Back, Next, Previous, and Value_At.
func Test_Traversal(t *testing.T) {
	t.Parallel()
	subject := empty_list()
	first := handle(list.Push_Back(subject, 1))
	second := handle(list.Push_Back(subject, 2))
	testify.Equal(t, first, list.Front(subject), "Front must return the first handle")
	testify.Equal(t, second, list.Back(subject), "Back must return the final handle")
	testify.Equal(t, second, list.Next(subject, first), "Next must walk toward the back")
	testify.Equal(t, first, list.Previous(subject, second),
		"Previous must walk toward the front")
	testify.Equal(t, list.POSITION_NONE, list.Next(subject, second),
		"Next must return POSITION_NONE at the back")
	testify.Equal(t, list.POSITION_NONE, list.Previous(subject, first),
		"Previous must return POSITION_NONE at the front")
	testify.Equal(t, list.Value(2), list.Value_At(subject, second),
		"Value_At must return the stored value")
}

// Test_Insertion verifies Push_Front, Push_Back, Insert_Before, and Insert_After.
func Test_Insertion(t *testing.T) {
	t.Parallel()
	subject := empty_list()
	list.Push_Back(subject, 2)
	mark := handle(list.Push_Front(subject, 1))
	list.Insert_After(subject, 3, mark)
	list.Insert_Before(subject, 0, mark)
	check_list(t, subject, []int{0, 1, 3, 2})
}

// Test_Removal verifies the returned value, the smaller count, and the storage reuse.
func Test_Removal(t *testing.T) {
	t.Parallel()
	subject := empty_list()
	first := list.Push_Back(subject, 1)
	list.Push_Back(subject, 2)
	testify.Equal(t, list.Value(1), list.Remove(subject, handle(first)),
		"Remove must return the value that the handle held")
	check_list(t, subject, []int{2})
	testify.Equal(t, first, list.Push_Back(subject, 3),
		"an insertion must reuse the storage of a removed node")
	check_list(t, subject, []int{2, 3})
}

// Test_Moves verifies each move against both ends and against a mark.
func Test_Moves(t *testing.T) {
	t.Parallel()
	subject := empty_list()
	one := handle(list.Push_Back(subject, 1))
	two := handle(list.Push_Back(subject, 2))
	three := handle(list.Push_Back(subject, 3))
	list.Move_To_Front(subject, three)
	check_list(t, subject, []int{3, 1, 2})
	list.Move_To_Back(subject, three)
	check_list(t, subject, []int{1, 2, 3})
	list.Move_Before(subject, three, one)
	check_list(t, subject, []int{3, 1, 2})
	list.Move_After(subject, three, two)
	check_list(t, subject, []int{1, 2, 3})
}

// Test_Copies verifies Push_Back_List and Push_Front_List, including a copy of one list
// into itself.
func Test_Copies(t *testing.T) {
	t.Parallel()
	source := fill_list(2)
	target := empty_list()
	list.Push_Back(target, 9)
	list.Push_Back_List(target, source)
	check_list(t, target, []int{9, 0, 1})
	list.Push_Front_List(target, source)
	check_list(t, target, []int{0, 1, 9, 0, 1})
	list.Push_Back_List(source, source)
	check_list(t, source, []int{0, 1, 0, 1})
}

// Test_Size_Limits verifies the largest admitted list and the rejection of one more element.
func Test_Size_Limits(t *testing.T) {
	t.Parallel()
	subject := fill_list(list.ELEMENT_COUNT_MAXIMUM)
	testify.Equal(t, list.Count(list.ELEMENT_COUNT_MAXIMUM), list.Element_Count(subject),
		"a list must hold ELEMENT_COUNT_MAXIMUM elements")
	testify.True(t, panicked(func() { list.Push_Back(subject, 0) }),
		"Push_Back must reject a full list")
	testify.True(t, panicked(func() { list.Push_Front(subject, 0) }),
		"Push_Front must reject a full list")
	testify.True(t, panicked(func() { list.Push_Back_List(subject, subject) }),
		"Push_Back_List must reject a copy that does not fit")
}

// Test_Allocation keeps fixed pool meaningful; every public operation must use pool storage
// without hidden collector work.
func Test_Allocation(t *testing.T) {
	state := prepare_list_allocation_state()
	verify_zero_allocation_cases(t, list_read_allocation_cases(state))
	verify_zero_allocation_cases(t, list_insertion_allocation_cases(state))
	verify_zero_allocation_cases(t, list_move_allocation_cases(state))
	testify.Equal(t, list.Count(2), state.Count_Result,
		"Element_Count allocation probe did not run")
	testify.Not_Equal(t, list.POSITION_NONE, state.Position_Result,
		"position allocation probes did not run")
	testify.True(t, state.Element_Position_Result >= list.FIRST_ELEMENT_POSITION,
		"insertion allocation probes did not run")
	testify.Equal(t, list.Value(1), state.Value_Result,
		"value allocation probes did not run")
}

// Test_Domain_Errors verifies the panic for each handle that no live element holds.
func Test_Domain_Errors(t *testing.T) {
	t.Parallel()
	subject := empty_list()
	taken := handle(list.Push_Back(subject, 1))
	list.Remove(subject, taken)
	panic_cases := []func(){
		func() { list.Value_At(subject, list.POSITION_NONE) },
		func() { list.Value_At(subject, list.ROOT_POSITION) },
		func() { list.Value_At(subject, list.FREE_POSITION) },
		func() { list.Value_At(subject, list.Position(list.POSITION_MAXIMUM)) },
		func() { list.Value_At(subject, taken) },
		func() { list.Next(subject, taken) },
		func() { list.Previous(subject, taken) },
		func() { list.Remove(subject, taken) },
		func() { list.Move_To_Front(subject, taken) },
		func() { list.Move_To_Back(subject, taken) },
		func() { list.Insert_Before(subject, 0, taken) },
		func() { list.Insert_After(subject, 0, taken) },
		func() { list.Move_Before(subject, taken, taken) },
		func() { list.Move_After(subject, taken, taken) },
	}
	for panic_index, action := range panic_cases {
		testify.True(t, panicked(action), "domain error %d did not panic", panic_index)
	}
}

// Test_Invariant_Domains verifies each special value through a public list operation.
func Test_Invariant_Domains(t *testing.T) {
	t.Parallel()
	for _, state := range probe_states() {
		drive_every_operation(state)
	}
	cover_value_domains()
}

// Reaches every sentinel required by concrete stored-value boundaries through public calls.
func cover_value_domains() {
	values := []list.Value{
		list.VALUE_MINIMUM, -1, 0, 1, 2, list.VALUE_MAXIMUM,
	}
	for _, value := range values {
		subject := empty_list()
		position := list.Position(list.Push_Back(subject, value))
		list.Value_At(subject, position)
		list.Remove(subject, position)

		subject = empty_list()
		list.Push_Front(subject, value)

		subject = empty_list()
		mark := list.Position(list.Push_Back(subject, 0))
		list.Insert_Before(subject, value, mark)

		subject = empty_list()
		mark = list.Position(list.Push_Back(subject, 0))
		list.Insert_After(subject, value, mark)
	}
}

// Makes one fresh list of one state, so a driven operation never reads a list that an
// earlier operation changed.
type list_factory func() (subject *list.List)

// Names each list state declared domains need, from uninitialized storage to list whose elements
// sit at highest positions of full pool.
func probe_states() (states []list_factory) {
	return []list_factory{
		uninitialized_list,
		empty_list,
		func() (subject *list.List) { return fill_list(1) },
		func() (subject *list.List) { return fill_list(2) },
		func() (subject *list.List) { return fill_list(3) },
		func() (subject *list.List) {
			return fill_list(list.ELEMENT_COUNT_MAXIMUM)
		},
		reuse_list,
		nearly_full_list,
		high_front_list,
		high_pair_list,
		high_reversed_list,
	}
}

// Drives every operation over a fresh list of one state, so each assertion root observes that
// state. A domain error is one of the observations, thus each call absorbs its panic.
func drive_every_operation(state list_factory) {
	for _, position := range probe_positions() {
		drive_every_position_operation(state, position)
	}
	attempt(func() { list.Element_Count(state()) })
	attempt(func() { list.Front(state()) })
	attempt(func() { list.Back(state()) })
	attempt(func() { list.Initialize(state()) })
	attempt(func() { list.Push_Front(state(), 0) })
	attempt(func() { list.Push_Back(state(), 0) })
	attempt(func() { list.Push_Back_List(state(), fill_list(1)) })
	attempt(func() { list.Push_Front_List(state(), fill_list(1)) })
	attempt(func() { list.Push_Back_List(fill_list(1), state()) })
	attempt(func() { list.Push_Front_List(fill_list(1), state()) })
}

// Drives every operation that takes a handle over a fresh list of one state.
func drive_every_position_operation(state list_factory, position list.Position) {
	attempt(func() { list.Next(state(), position) })
	attempt(func() { list.Previous(state(), position) })
	attempt(func() { list.Value_At(state(), position) })
	attempt(func() { list.Remove(state(), position) })
	attempt(func() { list.Move_To_Front(state(), position) })
	attempt(func() { list.Move_To_Back(state(), position) })
	attempt(func() { list.Insert_Before(state(), 0, position) })
	attempt(func() { list.Insert_After(state(), 0, position) })
	attempt(func() { list.Move_Before(state(), position, position) })
	attempt(func() { list.Move_After(state(), position, position) })
}

// Names each handle that the domains need, including POSITION_NONE, both sentinels, both
// interior sentinels, and the highest position of a full pool.
func probe_positions() (positions []list.Position) {
	return []list.Position{
		list.POSITION_NONE,
		list.ROOT_POSITION,
		list.FREE_POSITION,
		list.Position(list.FIRST_ELEMENT_POSITION),
		list.Position(list.FIRST_ELEMENT_POSITION + 1),
		list.Position(list.POSITION_MAXIMUM),
		list.Position(list.POSITION_MAXIMUM - 1),
	}
}

// Makes uninitialized List over caller storage, which holds no sentinel.
func uninitialized_list() (subject *list.List) {
	return &list.List{Nodes: make(list.Nodes, list.NODE_COUNT_MAXIMUM)}
}

// Makes an empty list that holds both sentinels.
func empty_list() (subject *list.List) {
	subject = uninitialized_list()
	list.Initialize(subject)
	return subject
}

// Makes a list whose free chain heads at the first element position, so a later insertion
// reaches the lowest handle that a caller value can occupy.
func reuse_list() (subject *list.List) {
	subject = fill_list(2)
	list.Remove(subject, list.Front(subject))
	return subject
}

// Makes a list that is one element below its limit, so a later insertion grows the pool to
// its final node and reaches the highest handle.
func nearly_full_list() (subject *list.List) {
	return fill_list(list.ELEMENT_COUNT_MAXIMUM - 1)
}

// Makes a list whose two highest handles run from the final one, so a walk toward the front
// reaches the highest handle.
func high_reversed_list() (subject *list.List) {
	subject = high_pair_list()
	list.Move_To_Front(subject, list.Back(subject))
	return subject
}

// Makes a list whose one element sits at the highest position of a full pool.
func high_front_list() (subject *list.List) {
	return drain_below(list.Position(list.POSITION_MAXIMUM))
}

// Makes a list whose two elements sit at the two highest positions of a full pool.
func high_pair_list() (subject *list.List) {
	return drain_below(list.Position(list.POSITION_MAXIMUM - 1))
}

// Fills a list to its element limit, then removes every element below one handle, so the
// remaining handles are the highest ones that the pool holds.
func drain_below(kept list.Position) (subject *list.List) {
	subject = fill_list(list.ELEMENT_COUNT_MAXIMUM)
	position := list.Front(subject)
	for position != kept {
		successor := list.Next(subject, position)
		list.Remove(subject, position)
		position = successor
	}
	return subject
}

// Makes a list that holds one count of increasing values.
func fill_list(element_count int) (subject *list.List) {
	subject = empty_list()
	for value := range element_count {
		list.Push_Back(subject, list.Value(value))
	}
	return subject
}

// Reads one insertion result as a caller handle.
func handle(position list.Element_Position) (converted list.Position) {
	return list.Position(position)
}

// Fails when a list does not hold exactly the wanted values in both walk directions.
func check_list(t *testing.T, subject *list.List, wanted []int) {
	t.Helper()
	testify.Equal(t, list.Count(len(wanted)), list.Element_Count(subject),
		"the element count is wrong")
	forward := []int{}
	position := list.Front(subject)
	for position != list.POSITION_NONE {
		forward = append(forward, int(list.Value_At(subject, position)))
		position = list.Next(subject, position)
	}
	testify.Equal(t, wanted, forward, "the walk toward the back is wrong")
	backward := []int{}
	position = list.Back(subject)
	for position != list.POSITION_NONE {
		backward = append(backward, int(list.Value_At(subject, position)))
		position = list.Previous(subject, position)
	}
	testify.Equal(t, reverse(wanted), backward, "the walk toward the front is wrong")
}

// Copies values in the opposite order.
func reverse(values []int) (reversed []int) {
	reversed = []int{}
	for index := len(values) - 1; index >= 0; index-- {
		reversed = append(reversed, values[index])
	}
	return reversed
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

type allocation_case struct {
	Name string
	Run  func()
}

type list_allocation_state struct {
	Empty                   list.List
	Single                  list.List
	Pair                    list.List
	Source                  list.List
	Subject                 list.List
	First                   list.Position
	Second                  list.Position
	Count_Result            list.Count
	Position_Result         list.Position
	Element_Position_Result list.Element_Position
	Value_Result            list.Value
}

// Snapshots restore each destructive probe without asking pool API for fresh state.
func prepare_list_allocation_state() (state *list_allocation_state) {
	state = &list_allocation_state{}
	state.Empty = *empty_list()
	state.Single = *empty_list()
	state.First = list.Position(list.Push_Back(&state.Single, 1))
	state.Pair = *empty_list()
	state.First = list.Position(list.Push_Back(&state.Pair, 1))
	state.Second = list.Position(list.Push_Back(&state.Pair, 2))
	state.Source = *empty_list()
	list.Push_Back(&state.Source, 1)
	state.Subject = *empty_list()
	return state
}

// Copies list into distinct caller storage so allocation probes keep snapshots stable.
func restore_list(destination *list.List, source *list.List) {
	copy(destination.Nodes, source.Nodes)
	destination.Element_Count = source.Element_Count
}

// Each callback stays named so one failure identifies one regressed read operation.
func list_read_allocation_cases(
	state *list_allocation_state,
) (cases []allocation_case) {
	return []allocation_case{
		{Name: "Initialize", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			list.Initialize(&state.Subject)
		}},
		{Name: "Element_Count", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			state.Count_Result = list.Element_Count(&state.Subject)
		}},
		{Name: "Front", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			state.Position_Result = list.Front(&state.Subject)
		}},
		{Name: "Back", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			state.Position_Result = list.Back(&state.Subject)
		}},
		{Name: "Next", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			state.Position_Result = list.Next(&state.Subject, state.First)
		}},
		{Name: "Previous", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			state.Position_Result = list.Previous(&state.Subject, state.Second)
		}},
		{Name: "Value_At", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			state.Value_Result = list.Value_At(&state.Subject, state.First)
		}},
	}
}

// Each callback stays named so one failure identifies one regressed storage operation.
func list_insertion_allocation_cases(
	state *list_allocation_state,
) (cases []allocation_case) {
	return []allocation_case{
		{Name: "Push_Front", Run: func() {
			restore_list(&state.Subject, &state.Empty)
			state.Element_Position_Result = list.Push_Front(&state.Subject, 1)
		}},
		{Name: "Push_Back", Run: func() {
			restore_list(&state.Subject, &state.Empty)
			state.Element_Position_Result = list.Push_Back(&state.Subject, 1)
		}},
		{Name: "Insert_Before", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			state.Element_Position_Result = list.Insert_Before(
				&state.Subject, 0, state.Second,
			)
		}},
		{Name: "Insert_After", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			state.Element_Position_Result = list.Insert_After(
				&state.Subject, 3, state.First,
			)
		}},
		{Name: "Remove", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			state.Value_Result = list.Remove(&state.Subject, state.First)
		}},
	}
}

// Each callback stays named so one failure identifies one regressed relinking operation.
func list_move_allocation_cases(
	state *list_allocation_state,
) (cases []allocation_case) {
	return []allocation_case{
		{Name: "Move_To_Front", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			list.Move_To_Front(&state.Subject, state.Second)
		}},
		{Name: "Move_To_Back", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			list.Move_To_Back(&state.Subject, state.First)
		}},
		{Name: "Move_Before", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			list.Move_Before(&state.Subject, state.Second, state.First)
		}},
		{Name: "Move_After", Run: func() {
			restore_list(&state.Subject, &state.Pair)
			list.Move_After(&state.Subject, state.First, state.Second)
		}},
		{Name: "Push_Back_List", Run: func() {
			restore_list(&state.Subject, &state.Empty)
			list.Push_Back_List(&state.Subject, &state.Source)
		}},
		{Name: "Push_Front_List", Run: func() {
			restore_list(&state.Subject, &state.Empty)
			list.Push_Front_List(&state.Subject, &state.Source)
		}},
	}
}

// Serial measurement keeps testing allocation counter isolated from parallel tests.
func verify_zero_allocation_cases(t *testing.T, cases []allocation_case) {
	t.Helper()
	for _, one := range cases {
		t.Run(one.Name, func(t *testing.T) {
			testify.Zero_Allocation(
				t, one.Run, "%s must allocate no heap storage", one.Name,
			)
		})
	}
}
