package list

import (
	"testing"

	"local/james-orcales/shared/testify"
)

// Test_Standard_Library_List ports the upstream TestList over one element. It drives the
// single-element list through each move and each removal.
func Test_Standard_Library_List(t *testing.T) {
	t.Parallel()
	subject := ready_list()
	check_values(t, subject, []int{})
	only := Position(Push_Front(subject, 1))
	check_values(t, subject, []int{1})
	Move_To_Front(subject, only)
	check_values(t, subject, []int{1})
	Move_To_Back(subject, only)
	check_values(t, subject, []int{1})
	Remove(subject, only)
	check_values(t, subject, []int{})
}

// Test_Standard_Library_List_Order ports the ordering part of the upstream TestList. It
// inserts at both ends and beside a mark, then removes from the middle.
func Test_Standard_Library_List_Order(t *testing.T) {
	t.Parallel()
	subject := ready_list()
	second := Position(Push_Front(subject, 2))
	first := Position(Push_Front(subject, 1))
	third := Position(Push_Back(subject, 3))
	fourth := Position(Push_Back(subject, 4))
	check_values(t, subject, []int{1, 2, 3, 4})
	Remove(subject, second)
	check_values(t, subject, []int{1, 3, 4})
	Move_To_Front(subject, third)
	check_values(t, subject, []int{3, 1, 4})
	Move_To_Front(subject, first)
	Move_To_Back(subject, third)
	check_values(t, subject, []int{1, 4, 3})
	Move_To_Front(subject, third)
	check_values(t, subject, []int{3, 1, 4})
	Move_To_Front(subject, third)
	check_values(t, subject, []int{3, 1, 4})
	Move_To_Back(subject, third)
	check_values(t, subject, []int{1, 4, 3})
	Move_To_Back(subject, third)
	check_values(t, subject, []int{1, 4, 3})
	Insert_Before(subject, 2, fourth)
	check_values(t, subject, []int{1, 2, 4, 3})
	Insert_After(subject, 5, fourth)
	check_values(t, subject, []int{1, 2, 4, 5, 3})
}

// Test_Standard_Library_Extension ports the upstream TestExtending. It copies one list into
// another at each end, and it copies one list into itself.
func Test_Standard_Library_Extension(t *testing.T) {
	t.Parallel()
	source := ready_list()
	Push_Back(source, 1)
	Push_Back(source, 2)
	Push_Back(source, 3)
	other := ready_list()
	Push_Back(other, 4)
	Push_Back(other, 5)
	target := ready_list()
	Push_Back_List(target, source)
	check_values(t, target, []int{1, 2, 3})
	Push_Back_List(target, other)
	check_values(t, target, []int{1, 2, 3, 4, 5})
	front := ready_list()
	Push_Front_List(front, other)
	check_values(t, front, []int{4, 5})
	Push_Front_List(front, source)
	check_values(t, front, []int{1, 2, 3, 4, 5})
	check_values(t, source, []int{1, 2, 3})
	check_values(t, other, []int{4, 5})
}

// Test_Standard_Library_Extension_Self ports the self-copy part of the upstream TestExtending,
// and it copies an empty list, which changes nothing.
func Test_Standard_Library_Extension_Self(t *testing.T) {
	t.Parallel()
	subject := ready_list()
	Push_Back(subject, 1)
	Push_Back(subject, 2)
	Push_Back(subject, 3)
	Push_Back_List(subject, subject)
	check_values(t, subject, []int{1, 2, 3, 1, 2, 3})
	front := ready_list()
	Push_Back_List(front, subject)
	Push_Front_List(front, front)
	check_values(t, front, []int{1, 2, 3, 1, 2, 3, 1, 2, 3, 1, 2, 3})
	empty := ready_list()
	Push_Back_List(subject, empty)
	check_values(t, subject, []int{1, 2, 3, 1, 2, 3})
	Push_Front_List(subject, empty)
	check_values(t, subject, []int{1, 2, 3, 1, 2, 3})
}

// Test_Standard_Library_Remove ports the upstream TestRemove. A second Remove of one handle
// causes a panic, where the standard library changes nothing, because a handle that Remove
// took names no element.
func Test_Standard_Library_Remove(t *testing.T) {
	t.Parallel()
	subject := ready_list()
	first := Position(Push_Back(subject, 1))
	Push_Back(subject, 2)
	testify.Equal(t, 1, Remove(subject, first), "Remove must return the value")
	check_values(t, subject, []int{2})
	testify.True(t, raises(func() { Remove(subject, first) }),
		"a second Remove of one handle must cause a panic")
	check_values(t, subject, []int{2})
}

// Test_Standard_Library_Removed_Handle ports the upstream TestIssue6349. A handle that Remove
// took names no element, thus each walk from it causes a panic.
func Test_Standard_Library_Removed_Handle(t *testing.T) {
	t.Parallel()
	subject := ready_list()
	Push_Back(subject, 1)
	Push_Back(subject, 2)
	front := Front(subject)
	testify.Equal(t, 1, Remove(subject, front), "Remove must return the value")
	testify.True(t, raises(func() { Next(subject, front) }),
		"Next must reject a handle that Remove took")
	testify.True(t, raises(func() { Previous(subject, front) }),
		"Previous must reject a handle that Remove took")
}

// Test_Standard_Library_Move ports the upstream TestMove. A move onto its own mark changes
// nothing, and each other move keeps every element.
func Test_Standard_Library_Move(t *testing.T) {
	t.Parallel()
	subject := ready_list()
	first := Position(Push_Back(subject, 1))
	second := Position(Push_Back(subject, 2))
	third := Position(Push_Back(subject, 3))
	fourth := Position(Push_Back(subject, 4))
	Move_After(subject, third, third)
	check_values(t, subject, []int{1, 2, 3, 4})
	Move_Before(subject, second, second)
	check_values(t, subject, []int{1, 2, 3, 4})
	Move_After(subject, third, second)
	check_values(t, subject, []int{1, 2, 3, 4})
	Move_Before(subject, second, third)
	check_values(t, subject, []int{1, 2, 3, 4})
	Move_Before(subject, second, fourth)
	check_values(t, subject, []int{1, 3, 2, 4})
	Move_Before(subject, fourth, first)
	check_values(t, subject, []int{4, 1, 3, 2})
	Move_After(subject, fourth, first)
	check_values(t, subject, []int{1, 4, 3, 2})
	Move_After(subject, third, second)
	check_values(t, subject, []int{1, 4, 2, 3})
}

// Test_Standard_Library_Zero_List ports the upstream TestZeroList. Each insertion readies a
// zero List value.
func Test_Standard_Library_Zero_List(t *testing.T) {
	t.Parallel()
	front := &List[int]{}
	Push_Front(front, 1)
	check_values(t, front, []int{1})
	back := &List[int]{}
	Push_Back(back, 1)
	check_values(t, back, []int{1})
	copied_front := &List[int]{}
	Push_Front_List(copied_front, front)
	check_values(t, copied_front, []int{1})
	copied_back := &List[int]{}
	Push_Back_List(copied_back, back)
	check_values(t, copied_back, []int{1})
}

// Test_Standard_Library_Unknown_Mark ports the upstream TestInsertBeforeUnknownMark,
// TestInsertAfterUnknownMark, and TestMoveUnknownMark. Each unknown mark causes a panic,
// where the standard library changes nothing, and the list keeps its elements.
func Test_Standard_Library_Unknown_Mark(t *testing.T) {
	t.Parallel()
	subject := ready_list()
	Push_Back(subject, 1)
	Push_Back(subject, 2)
	Push_Back(subject, 3)
	unknown := Position(POSITION_MAXIMUM)
	own := Front(subject)
	testify.True(t, raises(func() { Insert_Before(subject, 1, unknown) }),
		"Insert_Before must reject an unknown mark")
	testify.True(t, raises(func() { Insert_After(subject, 1, unknown) }),
		"Insert_After must reject an unknown mark")
	testify.True(t, raises(func() { Move_After(subject, own, unknown) }),
		"Move_After must reject an unknown mark")
	testify.True(t, raises(func() { Move_Before(subject, own, unknown) }),
		"Move_Before must reject an unknown mark")
	check_values(t, subject, []int{1, 2, 3})
}

// Makes an empty list, which the standard library makes with its New constructor.
func ready_list() (subject *List[int]) {
	subject = &List[int]{}
	Initialize(subject)
	return subject
}

// Fails when a list does not hold exactly the wanted values in both walk directions. The
// upstream checkListPointers reads the neighbor fields, which the walk states here.
func check_values(t *testing.T, subject *List[int], wanted []int) {
	t.Helper()
	testify.Equal(t, Count(len(wanted)), Element_Count(subject),
		"the element count is wrong")
	forward := []int{}
	position := Front(subject)
	for position != POSITION_NONE {
		forward = append(forward, Value_At(subject, position))
		position = Next(subject, position)
	}
	testify.Equal(t, wanted, forward, "the walk toward the back is wrong")
	backward := []int{}
	position = Back(subject)
	for position != POSITION_NONE {
		backward = append(backward, Value_At(subject, position))
		position = Previous(subject, position)
	}
	testify.Equal(t, backward_values(wanted), backward,
		"the walk toward the front is wrong")
}

// Copies values in the opposite order.
func backward_values(values []int) (reversed []int) {
	reversed = []int{}
	for index := len(values) - 1; index >= 0; index-- {
		reversed = append(reversed, values[index])
	}
	return reversed
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
