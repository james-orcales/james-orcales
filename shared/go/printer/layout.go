// The layout machine of the printer writes one file by running a program of steps over the parse
// tree, the note writer states the comments that stand between those steps, the number writer
// states the literals they hold, the name reader states which import names the file drops, and
// the slice writer states the bounds one value reads at. They stand here because a print reads
// its cursors at every step, and a machine of few units that run every step states the whole
// domain of each cursor rather than a part of it.
package printer

import (
	"local/james-orcales/shared/go/ast"
	"local/james-orcales/shared/go/token"
	"local/james-orcales/shared/sim/aver/default"
)

// Layout_Place is one step of the layout program.
type Layout_Place int32

// Layout_Place_Invariants states every step of the program.
func Layout_Place_Invariants(value Layout_Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), LAYOUT_PLACE_MINIMUM, LAYOUT_PLACE_MAXIMUM).
		Ensure()
}

// LAYOUT_PLACE_MINIMUM is the first step of the program.
const LAYOUT_PLACE_MINIMUM = 0

// LAYOUT_PLACE_MAXIMUM is the last step of the program.
const LAYOUT_PLACE_MAXIMUM = LAYOUT_STEP_COUNT - 1

// LAYOUT_STEP_SIZE is how many bytes of the program one step spans: the action and its operand.
const LAYOUT_STEP_SIZE = 3

// LAYOUT_OPERAND_MINIMUM is the operand a step that states none holds.
const LAYOUT_OPERAND_MINIMUM = 0

// LAYOUT_OPERAND_MAXIMUM is the widest operand one step of the program states, which is the step
// the widest jump names. A test reads the program and states this count.
const LAYOUT_OPERAND_MAXIMUM = 265

// Layout_Operand is what one step of the program states besides the action it names: the step a
// jump names, the rule a call names, the kind a test names, or the word a write states.
type Layout_Operand int32

// Layout_Operand_Invariants states every operand the program holds.
func Layout_Operand_Invariants(value Layout_Operand, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int32(int32(value), LAYOUT_OPERAND_MINIMUM, LAYOUT_OPERAND_MAXIMUM).
		Ensure()
}

// Reads what the step at one place states besides its action. One operand spans two bytes,
// because a byte of a constant string holds seven bits of a step.
func layout_operand(place Layout_Place) (operand Layout_Operand) {
	defer func() { Layout_Operand_Invariants(operand, "layout_operand.operand") }()
	Layout_Place_Invariants(place, "layout_operand.place")
	at := int(place) * LAYOUT_STEP_SIZE
	return Layout_Operand(int(LAYOUT_PROGRAM[at+1])*LAYOUT_OPERAND_BASE +
		int(LAYOUT_PROGRAM[at+2]))
}

// LAYOUT_OPERAND_BASE is the base of the two bytes one operand spans, because a byte of a constant
// string above seven bits would be written as two.
const LAYOUT_OPERAND_BASE = 128

// LAYOUT_NO_OPERAND pads a step whose action reads no operand.
const LAYOUT_NO_OPERAND = "\x00\x00"

// Reads the action the program holds at one place.
func layout_decode(place Layout_Place) (op Layout_Op) {
	defer func() { Layout_Op_Invariants(op, "layout_decode.op") }()
	Layout_Place_Invariants(place, "layout_decode.place")
	return Layout_Op(LAYOUT_PROGRAM[int(place)*LAYOUT_STEP_SIZE])
}

// Runs one rule of the program over the node the walk stands on until it returns, and hands back
// the test the rule ran last. A rule calls another rule over the node the walk stands on, hands it
// the test it ran last, and resumes behind the call with the test the rule it called ran last. The
// indent a rule holds back lives in the run of that rule alone, as any other local does.
func layout_run(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, rule Layout_Rule, held Boolean,
) (holds Boolean) {
	defer func() { Boolean_Invariants(holds, "layout_run.holds") }()
	Printer_Handle_Invariants(subject, "layout_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "layout_run.tree")
	Layout_Rule_Invariants(rule, "layout_run.rule")
	Boolean_Invariants(held, "layout_run.held")
	entry := int(LAYOUT_ENTRIES[int(rule)*2])*LAYOUT_OPERAND_BASE +
		int(LAYOUT_ENTRIES[int(rule)*2+1])
	frame := Layout_Frame{
		Place: Layout_Place(entry), Holds: held,
		File:   File_Walk{Named: false, Commented: false, Spread: false},
		Values: Values_Walk{Named: true, Valued: false, Names: false, Values: false},
		List: List_Walk{Started: false, Opened: false, Broken: false, Tail: false,
			Carries: false, Over: false, Bare: false},
		Parts: Parts_Walk{Names: false, Types: false},
		Block: Block_Walk{Entered: false, Trail: false, Trails: false, Held: false},
	}
	// A rule holds the steps standing between its own entry and the entry behind it.
	limit := LAYOUT_STEP_COUNT
	if int(rule) < LAYOUT_RULE_MAXIMUM {
		limit = int(LAYOUT_ENTRIES[(int(rule)+1)*2])*LAYOUT_OPERAND_BASE +
			int(LAYOUT_ENTRIES[(int(rule)+1)*2+1])
	}
	aver.Always(entry < limit, "A rule of the program holds a step of its own.")
	counts_rule(subject, tree, rule)
	indent := indent_load(subject.Counts.Values)
	column := indent
	stalled, spent := false, 0
	for range LAYOUT_STEP_BUDGET {
		frame.Place, spent = layout_guarded(frame.Place, frame.Holds), spent+1
		aver.Always(spent < LAYOUT_STEP_BUDGET, "A rule returns inside the steps it runs.")
		aver.Always(int(frame.Place) >= entry,
			"A step stands above the step its rule opens at.")
		aver.Always(int(frame.Place) < limit, "A step of one rule stands below its close.")
		switch layout_decode(frame.Place) {
		case LAYOUT_HOLD_FORM:
			// A form the author broke indents its continuations.
			indent = indent_load(subject.Counts.Values)
			flag_store(subject.Flags.Values, FLAG_BROKEN, false)
		case LAYOUT_RESTORE_FORM:
			indent_store(subject.Counts.Values, indent)
			flag_store(subject.Flags.Values, FLAG_BROKEN, false)
		case LAYOUT_LIST_BEGIN:
			column = indent_load(subject.Counts.Values)
		case LAYOUT_LIST_ITEM:
			// A list of type parameters or of results opens its column at its bracket.
			if !bool(frame.List.Started) {
				column = indent_load(subject.Counts.Values)
			}
		case LAYOUT_LIST_END:
			column_indent_store(subject.Counts.Values, column)
		}
		stood := frame.Place
		control := layout_step(subject, tree, &frame)
		if control == LAYOUT_CONTROL_RETURN {
			aver.Always(!stalled, "A step of one rule leaves the place it stood at.")
			return frame.Holds
		}
		if frame.Place == stood {
			stalled = true
		}
		if control == LAYOUT_CONTROL_CALL {
			callee := Layout_Rule(layout_operand(frame.Place - 1))
			frame.Holds = layout_run(subject, tree, callee, frame.Holds)
		}
	}
	return frame.Holds
}

// LAYOUT_STEP_BUDGET caps the steps one rule runs. Every rule steps the walk or returns within a
// bounded run of steps, thus a run that spends the budget misreads the program and stops.
const LAYOUT_STEP_BUDGET = ast.NODE_COUNT_MAXIMUM * 64

// Steps over the guards ahead of one place: a guard runs the step behind it only where the last
// test holds, or only where it failed.
func layout_guarded(place Layout_Place, holds Boolean) (next Layout_Place) {
	defer func() { Layout_Place_Invariants(next, "layout_guarded.next") }()
	Layout_Place_Invariants(place, "layout_guarded.place")
	Boolean_Invariants(holds, "layout_guarded.holds")
	next = place
	for range LAYOUT_STEP_COUNT {
		op := layout_decode(next)
		runs := bool(holds)
		if op == LAYOUT_UNLESS {
			runs = !runs
		}
		if op != LAYOUT_UNLESS {
			if op != LAYOUT_WHEN {
				return next
			}
		}
		next = next + 1
		if !runs {
			next = next + 1
		}
		// A guard stands ahead of the step it guards, thus the step it steps to stands
		// inside the program.
		aver.Always(int(next) < LAYOUT_STEP_COUNT,
			"A guard steps to a step of the program.")
	}
	return next
}

// LAYOUT_CONTROL_CONTINUE runs the next step of the rule.
const LAYOUT_CONTROL_CONTINUE Layout_Control = 0

// LAYOUT_CONTROL_RETURN ends the rule.
const LAYOUT_CONTROL_RETURN Layout_Control = 1

// LAYOUT_CONTROL_CALL runs the rule the step named and resumes behind it.
const LAYOUT_CONTROL_CALL Layout_Control = 2

// Layout_Control is what the rule that runs does once one step of it ran.
type Layout_Control uint8

// Layout_Control_Invariants states the three things a rule does next.
func Layout_Control_Invariants(value Layout_Control, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(
			uint8(value), uint8(LAYOUT_CONTROL_CONTINUE), uint8(LAYOUT_CONTROL_RETURN),
			uint8(LAYOUT_CONTROL_CALL),
		).
		Ensure()
}

// Runs one step of the rule a frame holds and names what the rule does next. The step itself
// writes what the program names outright and moves the walk, and the unit that owns the action
// runs every other one.
func layout_step(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, frame Layout_Frame_Handle,
) (control Layout_Control) {
	defer func() { Layout_Control_Invariants(control, "layout_step.control") }()
	Printer_Handle_Invariants(subject, "layout_step.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "layout_step.tree")
	Layout_Frame_Handle_Invariants(frame, "layout_step.frame")
	op := layout_decode(frame.Place)
	operand := int(layout_operand(frame.Place))
	holds := frame.Holds
	switch layout_group(op) {
	case LAYOUT_GROUP_FILE:
		holds = layout_file(subject, tree, File_Op(op-LAYOUT_FILE), holds, &frame.File)
	case LAYOUT_GROUP_PARTS:
		holds = layout_parts(subject, tree, Parts_Op(op-LAYOUT_DOWN_PART), holds)
	case LAYOUT_GROUP_VALUES:
		holds = layout_values(subject, tree, Values_Op(op-LAYOUT_VALUES_OPEN), holds,
			&frame.Values)
	case LAYOUT_GROUP_LISTS:
		holds = layout_lists(subject, tree, Lists_Op(op-LAYOUT_SIGNATURE_OVER), holds,
			List_Kind(operand), &frame.List)
	case LAYOUT_GROUP_NAMES:
		holds = layout_names(subject, tree, Names_Op(op-LAYOUT_PARTS_OPEN), holds,
			&frame.Parts)
	case LAYOUT_GROUP_BLOCKS:
		holds = layout_blocks(subject, tree, Blocks_Op(op-LAYOUT_SPANS_ONE_LINE), holds,
			&frame.Block)
	default:
		holds = layout_simple(subject, tree, Simple_Op(op), holds)
	}
	// A test of a kind, a keyword, a byte, and the keyword of a value declaration read the
	// operand the program holds, thus the step runs them itself.
	switch op {
	case LAYOUT_IS:
		// A test of a kind reads one class of node, thus the operand names a class the
		// tree states and never a number past them.
		aver.Always(operand <= int(ast.NODE_KIND_MAXIMUM),
			"A test of a kind names a class the tree states.")
		holds = Boolean(int(node_kind(subject, tree)) == operand)
	case LAYOUT_WORD:
		emit_word(subject, Word(operand))
	case LAYOUT_BYTE:
		emit_byte(subject, Symbol(operand))
	case LAYOUT_VALUE_WORD:
		if !bool(flag_load(subject.Flags.Values, FLAG_SHORT)) {
			emit_word(subject, Word(operand))
			emit_space(subject)
		}
	}
	frame.Holds = holds
	return layout_move(frame, op)
}

// Moves a frame to the step it runs next and names what the rule does next: the step a jump
// names, the step behind a call once the rule it names returned, or the step behind any other.
func layout_move(frame Layout_Frame_Handle, op Layout_Op) (control Layout_Control) {
	defer func() { Layout_Control_Invariants(control, "layout_move.control") }()
	Layout_Frame_Handle_Invariants(frame, "layout_move.frame")
	Layout_Op_Invariants(op, "layout_move.op")
	if op == LAYOUT_RETURN {
		return LAYOUT_CONTROL_RETURN
	}
	jumps := op == LAYOUT_JUMP
	if op == LAYOUT_JUMP_UNLESS {
		jumps = !bool(frame.Holds)
	}
	target := Layout_Place(layout_operand(frame.Place))
	frame.Place = frame.Place + 1
	if jumps {
		// The program holds the step a jump names, thus a jump lands on a step and
		// never past the run of them.
		aver.Always(int(target) < LAYOUT_STEP_COUNT,
			"A jump the program states names a step of the program.")
		frame.Place = target
	}
	if op == LAYOUT_CALL {
		// The program holds the rule a call names, thus a call runs a rule of the
		// program and never a number past the run of them.
		aver.Always(int(layout_operand(frame.Place-1)) <= LAYOUT_RULE_MAXIMUM,
			"A call the program states names a rule of the program.")
		return LAYOUT_CONTROL_CALL
	}
	return LAYOUT_CONTROL_CONTINUE
}

// LAYOUT_GROUP_SIMPLE names the actions the step runs itself.
const LAYOUT_GROUP_SIMPLE = 0

// LAYOUT_GROUP_FILE names the actions that walk the declarations of a file.
const LAYOUT_GROUP_FILE = 1

// LAYOUT_GROUP_PARTS names the actions that walk the parts of a form.
const LAYOUT_GROUP_PARTS = 2

// LAYOUT_GROUP_VALUES names the actions that write a value declaration.
const LAYOUT_GROUP_VALUES = 3

// LAYOUT_GROUP_LISTS names the actions that lay out a list.
const LAYOUT_GROUP_LISTS = 4

// LAYOUT_GROUP_NAMES names the actions that write a run of names and a run of types.
const LAYOUT_GROUP_NAMES = 5

// LAYOUT_GROUP_BLOCKS names the actions that write a block.
const LAYOUT_GROUP_BLOCKS = 6

// Layout_Group is the unit that runs one action.
type Layout_Group uint8

// Layout_Group_Invariants states every unit.
func Layout_Group_Invariants(value Layout_Group, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), LAYOUT_GROUP_SIMPLE, LAYOUT_GROUP_BLOCKS).
		Ensure()
}

// Names the unit that runs one action. Each unit owns one run of actions, in the order of the
// units.
func layout_group(op Layout_Op) (group Layout_Group) {
	defer func() { Layout_Group_Invariants(group, "layout_group.group") }()
	Layout_Op_Invariants(op, "layout_group.op")
	group = LAYOUT_GROUP_SIMPLE
	if op >= LAYOUT_FILE {
		group = LAYOUT_GROUP_FILE
	}
	if op >= LAYOUT_DOWN_PART {
		group = LAYOUT_GROUP_PARTS
	}
	if op >= LAYOUT_VALUES_OPEN {
		group = LAYOUT_GROUP_VALUES
	}
	if op >= LAYOUT_SIGNATURE_OVER {
		group = LAYOUT_GROUP_LISTS
	}
	if op >= LAYOUT_PARTS_OPEN {
		group = LAYOUT_GROUP_NAMES
	}
	if op >= LAYOUT_SPANS_ONE_LINE {
		group = LAYOUT_GROUP_BLOCKS
	}
	return group
}

// Runs the actions that move the walk and write what needs no operand, and reports the test the
// action ran. An action of any other kind leaves the last report standing.
func layout_simple(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, op Simple_Op, held Boolean,
) (holds Boolean) {
	defer func() { Boolean_Invariants(holds, "layout_simple.holds") }()
	Printer_Handle_Invariants(subject, "layout_simple.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "layout_simple.tree")
	Simple_Op_Invariants(op, "layout_simple.op")
	Boolean_Invariants(held, "layout_simple.held")
	switch Layout_Op(op) {
	case LAYOUT_DOWN:
		return descend(subject, tree)
	case LAYOUT_NEXT:
		return advance(subject, tree)
	case LAYOUT_UP:
		// Every part of one chain writes before the walk leaves it, thus a step up
		// stands on the last part and drops nothing the author wrote.
		if bool(writes_canonical(subject.Flags.Values)) {
			index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
			aver.Always(ast.Index(ast.Syntax_Node_At(tree, index).Next) == 0,
				"A step up the walk stands on the last part of its chain.")
		}
		ascend(subject)
	case LAYOUT_TOKEN:
		// A signature binds the names its parameters state, thus the count of them
		// reads what the signatures of the file state once the print closed.
		held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
		if ast.Syntax_Node_At(tree, held).Kind == ast.NODE_PARAMETER_NAME {
			subject.Counts.Values[COUNT_PARAMETER_NAME] += 1
		}
		emit_token(subject, tree)
	case LAYOUT_INDENT:
		emit_indent(subject)
	case LAYOUT_LINE:
		close_line(subject, tree)
	case LAYOUT_TRUE:
		return true
	case LAYOUT_INDENT_IN:
		indent_deepen(subject, indent_load(subject.Counts.Values))
	case LAYOUT_TRIVIA_HERE:
		print_trivia(subject, tree)
	}
	return held
}

// LIST_TYPE_PARAMETERS names a list of type parameters.
const LIST_TYPE_PARAMETERS = 0

// LIST_PARAMETERS names a list of parameters.
const LIST_PARAMETERS = 1

// LIST_RESULTS names a list of results.
const LIST_RESULTS = 2

// List_Kind names the list one list action lays out.
type List_Kind uint8

// List_Kind_Invariants states the three lists a signature states.
func List_Kind_Invariants(value List_Kind, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_3_Uint8(uint8(value), LIST_TYPE_PARAMETERS, LIST_PARAMETERS, LIST_RESULTS).
		Ensure()
}

// States what one declaration hands back to the file that holds it: the walk it stepped down, the
// tabs and the nest it opened at, the runs of keys and of notes it opened, and every break it
// took. A form that keeps any of them writes the declaration behind it against a state no
// declaration opens at.
func closes_declaration(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "closes_declaration.subject")
	if !bool(writes_canonical(subject.Flags.Values)) {
		return
	}
	// The report of an open line states what the bytes of the form spell, thus a print that
	// stands at the opening of a line closed the line it wrote.
	written := int(written_load(subject.Counts.Values))
	if written > 0 {
		aver.Always(bool(flag_load(subject.Flags.Values, FLAG_LINE)) ==
			(subject.Form.Values[written-1] == '\n'),
			"The report of an open line states the byte the form closes on.")
	}
	aver.Always(indent_load(subject.Counts.Values) == 0,
		"A declaration closes at the tabs the file opens at.")
	aver.Always(depth_load(subject.Counts.Values) == 1,
		"A declaration closes on the walk the file opened.")
	aver.Always(key_width_load(subject.Counts.Values) == 0,
		"A declaration closes the run of keys it opened.")
	aver.Always(column_load(subject.Counts.Values) == 0,
		"A declaration closes the run of notes it opened.")
	aver.Always(int(written_load(subject.Counts.Values)) >=
		int(declared_load(subject.Counts.Values)),
		"A declaration closes past the byte its run opened at.")
	aver.Always(nest_load(subject.Counts.Values) == NEST_MINIMUM,
		"A declaration closes at the nest the file opens at.")
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_BROKEN)),
		"A declaration closes the break it opened.")
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_TIGHT)),
		"A declaration closes the body it held against its braces.")
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_APART)),
		"A declaration closes the run it stood apart.")
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_SPLIT)),
		"A declaration closes the run of signs it broke.")
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_FLAT)),
		"A declaration closes every measure it took.")
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_NOTELESS)),
		"A declaration closes every read of a width it took.")
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_BODY)),
		"A declaration closes the body mark it opened.")
	aver.Always(comment_count(subject) == COMMENT_COUNT_MINIMUM,
		"A declaration closes carrying no note it still owes.")
	aver.Always(take_line_width(subject.Counts.Values) == 0,
		"A declaration closes the line it stood on.")
	aver.Always(!bool(flag_load(subject.Flags.Values, FLAG_SHORT)),
		"A declaration closes the short sign it stood under.")
}

// Runs the steps that walk the declarations of a file, and reports the test the step ran. A step
// of any other action leaves the last report standing.
func layout_file(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, op File_Op, held Boolean,
	walk File_Walk_Handle,
) (holds Boolean) {
	defer func() { Boolean_Invariants(holds, "layout_file.holds") }()
	Printer_Handle_Invariants(subject, "layout_file.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "layout_file.tree")
	File_Op_Invariants(op, "layout_file.op")
	Boolean_Invariants(held, "layout_file.held")
	File_Walk_Handle_Invariants(walk, "layout_file.walk")
	switch LAYOUT_FILE + Layout_Op(op) {
	case LAYOUT_FILE:
		return Boolean(node_kind(subject, tree) == ast.NODE_FILE)
	case LAYOUT_ALIASES:
		// The tables of names and of pairs state what the file holds, thus the reads of
		// them stand on the file itself.
		aver.Always(node_kind(subject, tree) == ast.NODE_FILE,
			"A read of the tables one file states stands on that file.")
		take_aliases(subject, tree)
		take_pairs(subject, tree)
		take_main(subject, tree)
	case LAYOUT_OPENING:
		walk.Named = Named(print_opening(subject, tree))
	case LAYOUT_NAMED:
		return Boolean(walk.Named)
	case LAYOUT_MOVES:
		moves := notes_move(subject, tree, walk.Run, walk.Moved)
		walk.Run = Notes_Run(node_kind(subject, tree) == ast.NODE_COMMENT)
		walk.Moved = Moved_Notes(moves)
		return moves
	case LAYOUT_OPEN_DECLARATION:
		// The file states the function one program enters at ahead of every other
		// function it declares, thus the print writes it where the first of them opens.
		walk.Spread = Spread_Run(open_main(subject, tree, Boolean(walk.Spread)))
		same := Boolean(declared_kind(subject, tree) == walk.Previous)
		open_declaration(subject, tree, Boolean(walk.Commented), same)
		if !bool(walk.Commented) {
			declared_store(subject.Counts.Values,
				Declaration_Place(written_load(subject.Counts.Values)))
		}
	case LAYOUT_CLOSE_DECLARATION:
		closes_declaration(subject)
		// The print closes one declaration of the file here, thus the count of them
		// reads what the file states once the print closed every one.
		if !bool(trivia_kind(subject, tree)) {
			subject.Counts.Values[COUNT_DECLARED_COUNT] =
				subject.Counts.Values[COUNT_DECLARED_COUNT] + 1
			counts_declaration(subject, tree)
		}
		kind := node_kind(subject, tree)
		walk.Commented = Commented(kind == ast.NODE_COMMENT)
		if !bool(walk.Commented) {
			walk.Spread = Spread_Run(open_spread(subject, Boolean(walk.Spread)))
		}
		if !bool(trivia_kind(subject, tree)) {
			walk.Previous = declared_kind(subject, tree)
		}
		// A type carries the run of invariants that stands under it.
		return Boolean(kind == ast.NODE_TYPE)
	case LAYOUT_INVARIANT_RUN:
		held, moved := print_invariant_run(subject, tree, Boolean(walk.Spread))
		walk.Spread = Spread_Run(held)
		// The type wrote the functions that state its invariants, thus the run stands
		// on as a run of functions.
		if moved {
			walk.Previous = DECLARED_FUNCTION
		}
	case LAYOUT_CLOSE_FILE:
		close_file(subject)
	}
	return held
}

// Runs the steps that walk the parts of a form, and reports the test the step ran. A step of any
// other action leaves the last report standing.
func layout_parts(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, op Parts_Op, held Boolean,
) (holds Boolean) {
	defer func() { Boolean_Invariants(holds, "layout_parts.holds") }()
	Printer_Handle_Invariants(subject, "layout_parts.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "layout_parts.tree")
	Parts_Op_Invariants(op, "layout_parts.op")
	Boolean_Invariants(held, "layout_parts.held")
	switch LAYOUT_DOWN_PART + Layout_Op(op) {
	case LAYOUT_DOWN_PART:
		return descend_part(subject, tree)
	case LAYOUT_NEXT_PART:
		return advance_part(subject, tree)
	case LAYOUT_TAKE_TRIVIA:
		take_trivia(subject, tree, advance(subject, tree))
	case LAYOUT_IMPORT_NAME:
		// A name the path itself states, or one the forms that read it now name by the
		// path, states nothing the import needs, thus the print writes the path alone.
		if node_kind(subject, tree) == ast.NODE_IMPORT_NAME {
			if bool(drops_alias(subject, tree)) {
				counts_dropped_type(subject, tree)
			}
		}
		if !bool(drops_alias(subject, tree)) {
			stood := written_load(subject.Counts.Values)
			emit_token(subject, tree)
			emit_space(subject)
			// A name the import keeps stands ahead of the path it binds, thus it
			// spells bytes of its own and a space behind them.
			if bool(writes_canonical(subject.Flags.Values)) {
				aver.Always(written_load(subject.Counts.Values) > stood+1,
					"A name one import keeps spells bytes of its own.")
			}
		}
	case LAYOUT_PASS:
		print_other_declaration(subject, tree)
	}
	return held
}

// Runs the steps that write a value declaration, and reports the test the step ran. A step of any
// other action leaves the last report standing.
func layout_values(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, op Values_Op, held Boolean,
	walk Values_Walk_Handle,
) (holds Boolean) {
	defer func() { Boolean_Invariants(holds, "layout_values.holds") }()
	Printer_Handle_Invariants(subject, "layout_values.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "layout_values.tree")
	Values_Op_Invariants(op, "layout_values.op")
	Boolean_Invariants(held, "layout_values.held")
	Values_Walk_Handle_Invariants(walk, "layout_values.walk")
	switch LAYOUT_VALUES_OPEN + Layout_Op(op) {
	case LAYOUT_VALUES_OPEN:
		*walk = Values_Walk{Named: true, Valued: false, Names: false, Values: false}
	case LAYOUT_TRIVIA:
		return opens_trivia(subject, tree, walk)
	case LAYOUT_VALUE_CHILD:
		return open_value_child(subject, tree, walk)
	case LAYOUT_EXPRESSION:
		print_expression(subject, tree)
	case LAYOUT_VALUES_NEXT:
		walk.Values = Value_Values(bool(walk.Values) || bool(walk.Valued))
		one := names_one(Boolean(walk.Named), Boolean(walk.Valued))
		walk.Names = Value_Names(bool(walk.Names) || bool(one))
		open := advance(subject, tree)
		ahead := opens_name(subject, tree)
		walk.Named = Value_Named(bool(walk.Named) && bool(open) && bool(ahead))
		return open
	case LAYOUT_TAKE_HERE:
		take_trivia(subject, tree, held)
	}
	return held
}

// Reports whether the trivia the walk stands on closes the declaration it stands in. A note the
// author left behind the word the declaration opens with stands ahead of every name it binds,
// thus the print writes it where it stands and the declaration runs on.
func opens_trivia(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, walk Values_Walk_Handle,
) (closes Boolean) {
	defer func() { Boolean_Invariants(closes, "opens_trivia.closes") }()
	Printer_Handle_Invariants(subject, "opens_trivia.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_trivia.tree")
	Values_Walk_Handle_Invariants(walk, "opens_trivia.walk")
	// A variable whose values name the type it binds states that type once, thus the walk
	// steps over the type the author wrote and the values behind it write it.
	if bool(repeats_bound_type(subject, tree, Boolean(walk.Valued))) {
		drop_bound_type(subject, tree)
		return false
	}
	// The names and the values of one declaration each stand in a run of their own, thus a
	// note the author left among them states nothing of the run it stands in.
	run := Boolean(bool(walk.Names) || bool(walk.Values))
	if bool(run) {
		if !bool(trivia_kind(subject, tree)) {
			return false
		}
		closed, named := names_trivia(subject, tree)
		if bool(closed) {
			return closed
		}
		walk.Named = named
		// The walk stands past the trivia it wrote, thus the form it stands on reads for
		// the type the check ahead of that trivia never stood on.
		if !bool(repeats_bound_type(subject, tree, Boolean(walk.Valued))) {
			return closed
		}
		drop_bound_type(subject, tree)
		return false
	}
	stand := trivia_kind(subject, tree)
	for bool(stand) {
		print_trivia(subject, tree)
		if !bool(advance(subject, tree)) {
			return true
		}
		stand = trivia_kind(subject, tree)
	}
	// The walk steps over every piece of trivia the declaration carries here, thus a walk
	// that still stands names a part the print writes.
	aver.Always(!bool(trivia_kind(subject, tree)),
		"A walk that read the trivia of one declaration stands on a part of it.")
	return false
}

// Reports whether the child the walk stands on states the type one declaration binds and the
// value behind it names that same type. A value that names the type states it once, thus the
// print writes the type the value holds and drops the one the names wear.
func repeats_bound_type(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, valued Boolean,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "repeats_bound_type.yes") }()
	Printer_Handle_Invariants(subject, "repeats_bound_type.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "repeats_bound_type.tree")
	Boolean_Invariants(valued, "repeats_bound_type.valued")
	if bool(valued) {
		return false
	}
	depth := int(depth_load(subject.Counts.Values))
	if depth+2 >= DEPTH_MAXIMUM {
		return false
	}
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the type one run repeats writes no byte of its own.")
	}()
	held := subject.Nodes.Values[depth]
	value := ast.Index(ast.Syntax_Node_At(tree, held).Next)
	if value == ast.INDEX_ABSENT {
		return false
	}
	// A note or an empty line the author left between the type and the value states neither,
	// thus the read steps over the trivia standing between the two.
	subject.Nodes.Values[depth] = value
	enter_named(subject, tree)
	value = subject.Nodes.Values[depth]
	subject.Nodes.Values[depth] = held
	if ast.Syntax_Node_At(tree, value).Kind != ast.NODE_COMPOSITE {
		return false
	}
	// The sign of the declaration stands ahead of its first value, thus the child behind
	// that sign states the type the names wear.
	subject.Nodes.Values[depth] = value
	bound := opens_value(subject, tree)
	subject.Nodes.Values[depth] = held
	if !bool(bound) {
		return false
	}
	head := ast.Index(ast.Syntax_Node_At(tree, value).First_Child)
	if head == ast.INDEX_ABSENT {
		return false
	}
	subject.Nodes.Values[depth] = head
	enter_named(subject, tree)
	head = subject.Nodes.Values[depth]
	subject.Nodes.Values[depth] = held
	// A literal that states no type of its own opens at its brace, thus a child behind that
	// brace states an element and never a type.
	if ast.Syntax_Node_At(tree, head).Token >= ast.Syntax_Node_At(tree, value).Token {
		return false
	}
	subject.Nodes.Values[depth+1] = held
	subject.Nodes.Values[depth+2] = head
	depth_store(subject.Counts.Values, Depth(depth+2))
	yes = same_form(subject, tree)
	// The read pushes the type the names wear and the type the value states, and the compare
	// of the two hands back the walk it was handed.
	aver.Always(int(depth_load(subject.Counts.Values)) == depth+2,
		"A compare of the type one value names hands back the walk it was handed.")
	depth_store(subject.Counts.Values, Depth(depth))
	return yes
}

// Writes what stands ahead of one child of a value declaration: the sign or the comma ahead of a
// value, the comma ahead of a name, and the space ahead of the type. It reports whether the child
// is an expression, which is every child but a name.
func open_value_child(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, walk Values_Walk_Handle,
) (expression Boolean) {
	defer func() { Boolean_Invariants(expression, "open_value_child.expression") }()
	Printer_Handle_Invariants(subject, "open_value_child.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_value_child.tree")
	Values_Walk_Handle_Invariants(walk, "open_value_child.walk")
	walk.Valued = Value_Valued(bool(walk.Valued) || bool(opens_value(subject, tree)))
	behind := Boolean(walk.Names)
	if bool(walk.Valued) {
		behind = Boolean(walk.Values)
		emit_separator(subject, !behind)
		nest_store(subject.Counts.Values, NEST_MINIMUM)
		return true
	}
	if bool(walk.Named) {
		if bool(behind) {
			emit_byte(subject, ',')
			emit_space(subject)
		}
		return false
	}
	emit_space(subject)
	return true
}

// Runs the steps that lay out one list, and reports the test the step ran. A step of any other
// action leaves the last report standing.
func layout_lists(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, op Lists_Op, held Boolean,
	kind List_Kind, list List_Walk_Handle,
) (holds Boolean) {
	defer func() { Boolean_Invariants(holds, "layout_lists.holds") }()
	Printer_Handle_Invariants(subject, "layout_lists.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "layout_lists.tree")
	Lists_Op_Invariants(op, "layout_lists.op")
	Boolean_Invariants(held, "layout_lists.held")
	List_Kind_Invariants(kind, "layout_lists.kind")
	List_Walk_Handle_Invariants(list, "layout_lists.list")
	switch LAYOUT_SIGNATURE_OVER + Layout_Op(op) {
	case LAYOUT_SIGNATURE_OVER:
		list.Over = List_Over(runs_over_signature(subject, tree, held))
	case LAYOUT_RESULTS_OVER:
		list.Over = List_Over(runs_over_results(subject, tree, held))
		list.Bare = List_Bare(bare_result(subject, tree, held))
	case LAYOUT_LIST_BEGIN:
		list.Started, list.Opened, list.Broken = false, false, false
		list.Tail, list.Carries = false, false
	case LAYOUT_LIST_ITEM:
		open_list_item(subject, tree, Boolean(kind == LIST_TYPE_PARAMETERS),
			Boolean(kind == LIST_RESULTS), list)
	case LAYOUT_LIST_END:
		if bool(list.Broken) {
			close_broken(subject, tree, Boolean(list.Tail) || Boolean(list.Carries),
				part_closer(subject, tree))
		}
		close_list_bracket(subject, Boolean(kind == LIST_TYPE_PARAMETERS),
			Boolean(kind == LIST_RESULTS), list)
	case LAYOUT_BODY:
		emit_space(subject)
		// The body one function states closes up against its braces however many
		// statements it holds.
		flag_store(subject.Flags.Values, FLAG_BODY, true)
		print_block(subject, tree)
	}
	return held
}

// Writes what stands ahead of one part of a list: the bracket and the break ahead of the first
// part, and the comma and the break ahead of any other. A list of type parameters opens at its
// bracket and reads no width of its own, and a list of results opens at a space and a parenthesis
// unless it states one bare result.
func open_list_item(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, typed Boolean, results Boolean,
	list List_Walk_Handle,
) {
	Printer_Handle_Invariants(subject, "open_list_item.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_list_item.tree")
	Boolean_Invariants(typed, "open_list_item.typed")
	Boolean_Invariants(results, "open_list_item.results")
	List_Walk_Handle_Invariants(list, "open_list_item.list")
	here := breaks_before(subject, tree)
	if !bool(typed) {
		here = here || Boolean(list.Over)
	}
	if bool(list.Started) {
		open_break(subject, here, Boolean(list.Opened))
		list.Opened = list.Opened || List_Opened(here)
		list.Broken = list.Broken || List_Broken(here)
		print_element_break(subject, tree, here)
		// An item the print breaks to stands on a line of its own, thus the line it
		// opens holds the tabs of the list and nothing the item wrote yet.
		if bool(here) {
			if bool(writes_canonical(subject.Flags.Values)) {
				aver.Always(int(take_line_width(subject.Counts.Values)) ==
					int(indent_load(subject.Counts.Values))*INDENT_WIDTH,
					"An item the print breaks to opens a line of its own.")
			}
		}
		return
	}
	if bool(typed) {
		emit_byte(subject, '[')
	}
	if bool(results) {
		emit_space(subject)
		if !bool(list.Bare) {
			emit_byte(subject, '(')
		}
	}
	if !bool(typed) {
		list.Carries = List_Carries(closes_signature(subject, tree) || Boolean(list.Over))
	}
	list.Tail = List_Tail(closes_list(subject, tree))
	print_first_break(subject, tree, here)
	list.Opened = List_Opened(here)
	list.Broken = List_Broken(here) || List_Broken(list.Tail)
	list.Started = true
}

// Writes the bracket a list of type parameters or of results that holds a part closes with.
func close_list_bracket(
	subject Printer_Handle, typed Boolean, results Boolean, list List_Walk_Handle,
) {
	Printer_Handle_Invariants(subject, "close_list_bracket.subject")
	Boolean_Invariants(typed, "close_list_bracket.typed")
	Boolean_Invariants(results, "close_list_bracket.results")
	List_Walk_Handle_Invariants(list, "close_list_bracket.list")
	if !bool(list.Started) {
		return
	}
	stood := written_load(subject.Counts.Values)
	if bool(typed) {
		emit_byte(subject, ']')
	}
	if bool(results) {
		if !bool(list.Bare) {
			emit_byte(subject, ')')
		}
	}
	// A list closes on the bracket it opened and on no byte besides.
	aver.Always(int(written_load(subject.Counts.Values)) <= int(stood)+1,
		"A close of one list writes the bracket it owes and no more.")
}

// Runs the steps that write a run of names and a run of types, and reports the test the step ran.
// A step of any other action leaves the last report standing.
func layout_names(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, op Names_Op, held Boolean,
	parts Parts_Walk_Handle,
) (holds Boolean) {
	defer func() { Boolean_Invariants(holds, "layout_names.holds") }()
	Printer_Handle_Invariants(subject, "layout_names.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "layout_names.tree")
	Names_Op_Invariants(op, "layout_names.op")
	Boolean_Invariants(held, "layout_names.held")
	Parts_Walk_Handle_Invariants(parts, "layout_names.parts")
	switch LAYOUT_PARTS_OPEN + Layout_Op(op) {
	case LAYOUT_PARTS_OPEN:
		parts.Names, parts.Types = false, false
	case LAYOUT_NAME_MARK, LAYOUT_PARAMETER_MARK:
		if bool(parts.Names) {
			emit_byte(subject, ',')
			emit_space(subject)
		}
		if !bool(parts.Names) {
			if LAYOUT_PARTS_OPEN+Layout_Op(op) == LAYOUT_PARAMETER_MARK {
				emit_byte(subject, '[')
			}
		}
		parts.Names = true
	case LAYOUT_PARAMETERS_CLOSE:
		if bool(parts.Names) {
			emit_byte(subject, ']')
		}
	case LAYOUT_NAMES_SPACE:
		if bool(parts.Names) {
			emit_space(subject)
		}
	case LAYOUT_TYPE_MARK:
		if bool(parts.Types) {
			emit_byte(subject, ',')
			emit_space(subject)
		}
		parts.Types = true
	case LAYOUT_STANDS_LAST:
		return stands_last(subject, tree)
	case LAYOUT_FINISH_PARTS:
		finish_parts(subject, tree)
	}
	return held
}

// Runs the steps that write one block, and reports the test the step ran. A step of any other
// action leaves the last report standing.
func layout_blocks(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, op Blocks_Op, held Boolean,
	block Block_Walk_Handle,
) (holds Boolean) {
	defer func() { Boolean_Invariants(holds, "layout_blocks.holds") }()
	Printer_Handle_Invariants(subject, "layout_blocks.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "layout_blocks.tree")
	Blocks_Op_Invariants(op, "layout_blocks.op")
	Boolean_Invariants(held, "layout_blocks.held")
	Block_Walk_Handle_Invariants(block, "layout_blocks.block")
	switch LAYOUT_SPANS_ONE_LINE + Layout_Op(op) {
	case LAYOUT_SPANS_ONE_LINE:
		return spans_one_line(subject, tree)
	case LAYOUT_INLINE_BLOCK:
		print_inline_block(subject, tree)
	case LAYOUT_BLOCK_OPEN:
		block.Held = Block_Held(open_block_tight(subject, tree))
	case LAYOUT_ENTERED:
		block.Entered = Block_Entered(held)
	case LAYOUT_TRAILS_BRACE:
		return trails_open_brace(subject, tree)
	case LAYOUT_BLOCK_SKIP:
		// An empty line against a brace states nothing, and one counted at a token
		// behind the brace of a block inside falls behind that block.
		if bool(stands_against(subject, tree)) {
			return true
		}
		if bool(closes_block(subject, tree)) {
			if bool(stands_behind(subject, tree)) {
				block.Trail = true
				return true
			}
		}
		// A note the author left on the line the brace closes keeps that line, thus
		// the block holds it and the line it closes leaves it standing.
		if bool(trails_close(subject, tree)) {
			print_trivia(subject, tree)
			return true
		}
		return stands_against(subject, tree)
	case LAYOUT_COLUMN_OPEN:
		// The statements that each carry a note behind them stand in one run, and every
		// note of that run opens at one column. A statement that stands in no run, and
		// one standing behind an empty line, each close the run ahead of them.
		trails := trails_comment(subject, tree) && !spans_form(subject, tree)
		if !bool(trails) {
			column_store(subject.Counts.Values, 0)
		}
		if bool(stands_after_blank(subject)) {
			column_store(subject.Counts.Values, 0)
		}
		if bool(trails) {
			if column_load(subject.Counts.Values) == 0 {
				take_statement_column(subject, tree)
			}
		}
		block.Trails = Block_Trails(trails)
	case LAYOUT_STATEMENT:
		print_block_statement(subject, tree)
	case LAYOUT_COLUMN_CLOSE:
		if !bool(block.Trails) {
			column_store(subject.Counts.Values, 0)
		}
	case LAYOUT_BLOCK_CLOSE:
		close_block(subject, tree, block)
	}
	return held
}

// Reports whether the node the walk stands on states a note the author left on the line the
// brace that closes the form around it stands on.
func trails_close(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_close.yes") }()
	Printer_Handle_Invariants(subject, "trails_close.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "trails_close.tree")
	if node_kind(subject, tree) != ast.NODE_COMMENT {
		return false
	}
	return stands_behind(subject, tree)
}

// Closes one block: the column of its notes, the walk it stepped into, its indent, and the brace
// on a line of its own. The empty line a statement carried behind it stands against the closing
// brace, thus the body closes at its last statement and that line is never written.
func close_block(subject Printer_Handle, tree ast.Syntax_Tree_Handle, block Block_Walk_Handle) {
	Printer_Handle_Invariants(subject, "close_block.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "close_block.tree")
	Block_Walk_Handle_Invariants(block, "close_block.block")
	// Every statement of the body hands back the nest it opened at, thus the body closes at
	// the nest one statement stands at.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(nest_load(subject.Counts.Values) == NEST_MINIMUM,
			"A body closes at the nest its statements stand at.")
	}
	column_store(subject.Counts.Values, 0)
	if bool(block.Entered) {
		ascend(subject)
	}
	indent_shallow(subject, indent_load(subject.Counts.Values))
	flag_store(subject.Flags.Values, FLAG_BLANK, false)
	emit_indent(subject)
	emit_byte(subject, '}')
	// The brace that closes a block stands on a line of its own, at the tabs the block
	// opened at and behind nothing else.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(take_line_width(subject.Counts.Values)) ==
			int(indent_load(subject.Counts.Values))*INDENT_WIDTH+1,
			"A block closes its brace on a line of its own.")

	}
	if bool(block.Trail) {
		flag_store(subject.Flags.Values, FLAG_BLANK, true)
	}
	restore_tight(subject, Boolean(block.Held))
}

// Block_Walk_Handle gives one walk over a block one identity.
type Block_Walk_Handle *Block_Walk

// Block_Walk_Handle_Invariants composes a present walk.
func Block_Walk_Handle_Invariants(value Block_Walk_Handle, namespace aver.Namespace) {
	if value == nil {
		return
	}
	Block_Walk_Invariants(*value, namespace)
}

// The layout program stands below.

// RULE_FILE names rule 0 of the program.
const RULE_FILE = 0

// RULE_DECLARATION names rule 1 of the program.
const RULE_DECLARATION = 1

// RULE_IMPORT names rule 2 of the program.
const RULE_IMPORT = 2

// RULE_NOTE names rule 3 of the program.
const RULE_NOTE = 3

// RULE_CONSTANT names rule 4 of the program.
const RULE_CONSTANT = 4

// RULE_VARIABLE names rule 5 of the program.
const RULE_VARIABLE = 5

// RULE_VALUES names rule 6 of the program.
const RULE_VALUES = 6

// RULE_TYPE names rule 7 of the program.
const RULE_TYPE = 7

// RULE_FUNCTION names rule 8 of the program.
const RULE_FUNCTION = 8

// RULE_RECEIVER names rule 9 of the program.
const RULE_RECEIVER = 9

// RULE_SIGNATURE names rule 10 of the program.
const RULE_SIGNATURE = 10

// RULE_HEAD names rule 11 of the program.
const RULE_HEAD = 11

// RULE_RESULTS names rule 12 of the program.
const RULE_RESULTS = 12

// RULE_TYPE_PARAMETERS names rule 13 of the program.
const RULE_TYPE_PARAMETERS = 13

// RULE_PARAMETER names rule 14 of the program.
const RULE_PARAMETER = 14

// RULE_TYPE_PARAMETER names rule 15 of the program.
const RULE_TYPE_PARAMETER = 15

// RULE_BLOCK names rule 16 of the program.
const RULE_BLOCK = 16

// LAYOUT_RULE_MINIMUM is the first rule.
const LAYOUT_RULE_MINIMUM = RULE_FILE

// LAYOUT_RULE_MAXIMUM is the last rule.
const LAYOUT_RULE_MAXIMUM = RULE_BLOCK

// LAYOUT_STEP_COUNT is how many steps the program holds. Registration reads no length of a
// string, thus the count is spelled out and a test holds it to the program.
const LAYOUT_STEP_COUNT = 267

// FILE_RULE writes the opening of a file and every declaration behind it. A function that states
// the invariants of a type stands under that type, thus the walk writes nothing where the author
// wrote it and the type writes it instead.
const FILE_RULE = string(rune(LAYOUT_FILE)) + LAYOUT_NO_OPERAND + // 0
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x17" + // 1
	string(rune(LAYOUT_ALIASES)) + LAYOUT_NO_OPERAND + // 2
	string(rune(LAYOUT_DOWN)) + LAYOUT_NO_OPERAND + // 3
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x17" + // 4
	string(rune(LAYOUT_OPENING)) + LAYOUT_NO_OPERAND + // 5
	string(rune(LAYOUT_NEXT)) + LAYOUT_NO_OPERAND + // 6
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x15" + // 7
	string(rune(LAYOUT_NAMED)) + LAYOUT_NO_OPERAND + // 8
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x05" + // 9
	string(rune(LAYOUT_MOVES)) + LAYOUT_NO_OPERAND + // 10
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 11
	string(rune(LAYOUT_JUMP)) + "\x00\x12" + // 12
	string(rune(LAYOUT_OPEN_DECLARATION)) + LAYOUT_NO_OPERAND + // 13
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_DECLARATION)) + // 14
	string(rune(LAYOUT_CLOSE_DECLARATION)) + LAYOUT_NO_OPERAND + // 15
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 16
	string(rune(LAYOUT_INVARIANT_RUN)) + LAYOUT_NO_OPERAND + // 17
	string(rune(LAYOUT_NEXT)) + LAYOUT_NO_OPERAND + // 18
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 19
	string(rune(LAYOUT_JUMP)) + "\x00\x0a" + // 20
	string(rune(LAYOUT_CLOSE_FILE)) + LAYOUT_NO_OPERAND + // 21
	string(rune(LAYOUT_UP)) + LAYOUT_NO_OPERAND + // 22
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 23

// DECLARATION_RULE writes one declaration by the rule of its kind.
const DECLARATION_RULE = string(rune(LAYOUT_HOLD_FORM)) + LAYOUT_NO_OPERAND + // 24
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_IMPORT)) + // 25
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x1d" + // 26
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_IMPORT)) + // 27
	string(rune(LAYOUT_JUMP)) + "\x00\x36" + // 28
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_COMMENT)) + // 29
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x21" + // 30
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_NOTE)) + // 31
	string(rune(LAYOUT_JUMP)) + "\x00\x36" + // 32
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_BLANK)) + // 33
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x25" + // 34
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 35
	string(rune(LAYOUT_JUMP)) + "\x00\x36" + // 36
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_CONSTANT)) + // 37
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x29" + // 38
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_CONSTANT)) + // 39
	string(rune(LAYOUT_JUMP)) + "\x00\x36" + // 40
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_VARIABLE)) + // 41
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x2d" + // 42
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_VARIABLE)) + // 43
	string(rune(LAYOUT_JUMP)) + "\x00\x36" + // 44
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_TYPE)) + // 45
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x31" + // 46
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_TYPE)) + // 47
	string(rune(LAYOUT_JUMP)) + "\x00\x36" + // 48
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_FUNCTION)) + // 49
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x35" + // 50
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_FUNCTION)) + // 51
	string(rune(LAYOUT_JUMP)) + "\x00\x36" + // 52
	string(rune(LAYOUT_PASS)) + LAYOUT_NO_OPERAND + // 53
	string(rune(LAYOUT_RESTORE_FORM)) + LAYOUT_NO_OPERAND + // 54
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 55

// IMPORT_RULE writes one import: the word, the name the file binds where it states one, and the
// path.
const IMPORT_RULE = string(rune(LAYOUT_INDENT)) + LAYOUT_NO_OPERAND + // 56
	string(rune(LAYOUT_WORD)) + "\x00" + string(rune(WORD_IMPORT)) + // 57
	string(rune(LAYOUT_BYTE)) + "\x00" + string(rune(' ')) + // 58
	string(rune(LAYOUT_DOWN_PART)) + LAYOUT_NO_OPERAND + // 59
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x47" + // 60
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_IMPORT_NAME)) + // 61
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 62
	string(rune(LAYOUT_IMPORT_NAME)) + LAYOUT_NO_OPERAND + // 63
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 64
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 65
	string(rune(LAYOUT_TOKEN)) + LAYOUT_NO_OPERAND + // 66
	string(rune(LAYOUT_TAKE_TRIVIA)) + LAYOUT_NO_OPERAND + // 67
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 68
	string(rune(LAYOUT_UP)) + LAYOUT_NO_OPERAND + // 69
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND + // 70
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 71
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 72

// NOTE_RULE writes one comment on a line of its own.
const NOTE_RULE = string(rune(LAYOUT_INDENT)) + LAYOUT_NO_OPERAND + // 73
	string(rune(LAYOUT_TOKEN)) + LAYOUT_NO_OPERAND + // 74
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 75
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 76

// CONSTANT_RULE writes one constant declaration.
const CONSTANT_RULE = string(rune(LAYOUT_INDENT)) + LAYOUT_NO_OPERAND + // 77
	string(rune(LAYOUT_VALUE_WORD)) + "\x00" + string(rune(WORD_CONSTANT)) + // 78
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_VALUES)) + // 79
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 80

// VARIABLE_RULE writes one variable declaration.
const VARIABLE_RULE = string(rune(LAYOUT_INDENT)) + LAYOUT_NO_OPERAND + // 81
	string(rune(LAYOUT_VALUE_WORD)) + "\x00" + string(rune(WORD_VARIABLE)) + // 82
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_VALUES)) + // 83
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 84

// VALUES_RULE writes the names of one value declaration, the type they wear, and the values
// behind them, and the trivia behind all of them on the line they close.
const VALUES_RULE = string(rune(LAYOUT_DOWN)) + LAYOUT_NO_OPERAND + // 85
	string(rune(LAYOUT_VALUES_OPEN)) + LAYOUT_NO_OPERAND + // 86
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x63" + // 87
	string(rune(LAYOUT_TRIVIA)) + LAYOUT_NO_OPERAND + // 88
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 89
	string(rune(LAYOUT_JUMP)) + "\x00\x63" + // 90
	string(rune(LAYOUT_VALUE_CHILD)) + LAYOUT_NO_OPERAND + // 91
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 92
	string(rune(LAYOUT_EXPRESSION)) + LAYOUT_NO_OPERAND + // 93
	string(rune(LAYOUT_UNLESS)) + LAYOUT_NO_OPERAND + // 94
	string(rune(LAYOUT_TOKEN)) + LAYOUT_NO_OPERAND + // 95
	string(rune(LAYOUT_VALUES_NEXT)) + LAYOUT_NO_OPERAND + // 96
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 97
	string(rune(LAYOUT_JUMP)) + "\x00\x58" + // 98
	string(rune(LAYOUT_TAKE_HERE)) + LAYOUT_NO_OPERAND + // 99
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 100
	string(rune(LAYOUT_UP)) + LAYOUT_NO_OPERAND + // 101
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 102

// TYPE_RULE writes one type declaration: the word, the name, the parameters it states, and the
// type behind them.
const TYPE_RULE = string(rune(LAYOUT_INDENT)) + LAYOUT_NO_OPERAND + // 103
	string(rune(LAYOUT_WORD)) + "\x00" + string(rune(WORD_TYPE)) + // 104
	string(rune(LAYOUT_BYTE)) + "\x00" + string(rune(' ')) + // 105
	string(rune(LAYOUT_DOWN_PART)) + LAYOUT_NO_OPERAND + // 106
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x7d" + // 107
	string(rune(LAYOUT_TOKEN)) + LAYOUT_NO_OPERAND + // 108
	string(rune(LAYOUT_PARTS_OPEN)) + LAYOUT_NO_OPERAND + // 109
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 110
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x7b" + // 111
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_TYPE_PARAMETER)) + // 112
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x00\x76" + // 113
	string(rune(LAYOUT_PARAMETER_MARK)) + LAYOUT_NO_OPERAND + // 114
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_TYPE_PARAMETER)) + // 115
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 116
	string(rune(LAYOUT_JUMP)) + "\x00\x6f" + // 117
	string(rune(LAYOUT_PARAMETERS_CLOSE)) + LAYOUT_NO_OPERAND + // 118
	string(rune(LAYOUT_BYTE)) + "\x00" + string(rune(' ')) + // 119
	string(rune(LAYOUT_EXPRESSION)) + LAYOUT_NO_OPERAND + // 120
	string(rune(LAYOUT_TAKE_TRIVIA)) + LAYOUT_NO_OPERAND + // 121
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 122
	string(rune(LAYOUT_UP)) + LAYOUT_NO_OPERAND + // 123
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND + // 124
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 125
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 126

// FUNCTION_RULE writes one function declaration: the word, the receiver where it states one, the
// name, the parameters, the results, and the block behind them.
const FUNCTION_RULE = string(rune(LAYOUT_INDENT)) + LAYOUT_NO_OPERAND + // 127
	string(rune(LAYOUT_WORD)) + "\x00" + string(rune(WORD_FUNCTION)) + // 128
	string(rune(LAYOUT_BYTE)) + "\x00" + string(rune(' ')) + // 129
	string(rune(LAYOUT_DOWN_PART)) + LAYOUT_NO_OPERAND + // 130
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x0e" + // 131
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_RECEIVER)) + // 132
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x08" + // 133
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_RECEIVER)) + // 134
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 135
	string(rune(LAYOUT_TOKEN)) + LAYOUT_NO_OPERAND + // 136
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 137
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_SIGNATURE)) + // 138
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 139
	string(rune(LAYOUT_UP)) + LAYOUT_NO_OPERAND + // 140
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND + // 141
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 142
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 143

// RECEIVER_RULE writes the receiver of a method, which stands between the word and the name.
const RECEIVER_RULE = string(rune(LAYOUT_BYTE)) + "\x00" + string(rune('(')) + // 144
	string(rune(LAYOUT_DOWN_PART)) + LAYOUT_NO_OPERAND + // 145
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x16" + // 146
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_PARAMETER)) + // 147
	string(rune(LAYOUT_FINISH_PARTS)) + LAYOUT_NO_OPERAND + // 148
	string(rune(LAYOUT_UP)) + LAYOUT_NO_OPERAND + // 149
	string(rune(LAYOUT_BYTE)) + "\x00" + string(rune(')')) + // 150
	string(rune(LAYOUT_BYTE)) + "\x00" + string(rune(' ')) + // 151
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 152

// SIGNATURE_RULE writes the type parameters, the parameters, the results, and the body of one
// signature.
const SIGNATURE_RULE = string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_HEAD)) + // 153
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x20" + // 154
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_BLOCK)) + // 155
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x20" + // 156
	string(rune(LAYOUT_BODY)) + LAYOUT_NO_OPERAND + // 157
	string(rune(LAYOUT_NEXT)) + LAYOUT_NO_OPERAND + // 158
	string(rune(LAYOUT_JUMP)) + "\x01\x1a" + // 159
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 160

// HEAD_RULE writes the type parameters, the parameters, and the results of one signature, and
// hands back whether the walk stands on a part behind them. A signature that runs the line past
// the widest one states one parameter to a line.
const HEAD_RULE = string(rune(LAYOUT_SIGNATURE_OVER)) + LAYOUT_NO_OPERAND + // 161
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_TYPE_PARAMETERS)) + // 162
	string(rune(LAYOUT_BYTE)) + "\x00" + string(rune('(')) + // 163
	string(rune(LAYOUT_LIST_BEGIN)) + LAYOUT_NO_OPERAND + // 164
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x2d" + // 165
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_PARAMETER)) + // 166
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x2c" + // 167
	string(rune(LAYOUT_LIST_ITEM)) + "\x00" + string(rune(LIST_PARAMETERS)) + // 168
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_PARAMETER)) + // 169
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 170
	string(rune(LAYOUT_JUMP)) + "\x01\x25" + // 171
	string(rune(LAYOUT_TRUE)) + LAYOUT_NO_OPERAND + // 172
	string(rune(LAYOUT_LIST_END)) + "\x00" + string(rune(LIST_PARAMETERS)) + // 173
	string(rune(LAYOUT_BYTE)) + "\x00" + string(rune(')')) + // 174
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_RESULTS)) + // 175
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 176

// RESULTS_RULE writes the results of one signature, and hands back whether the walk stands on a
// part behind them. One bare result needs no parentheses to read, and the canonical form writes
// none; the results read the line they close and break only where they themselves run past the
// widest one.
const RESULTS_RULE = string(rune(LAYOUT_RESULTS_OVER)) + LAYOUT_NO_OPERAND + // 177
	string(rune(LAYOUT_LIST_BEGIN)) + LAYOUT_NO_OPERAND + // 178
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x3b" + // 179
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_RESULT)) + // 180
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x3a" + // 181
	string(rune(LAYOUT_LIST_ITEM)) + "\x00" + string(rune(LIST_RESULTS)) + // 182
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_PARAMETER)) + // 183
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 184
	string(rune(LAYOUT_JUMP)) + "\x01\x33" + // 185
	string(rune(LAYOUT_TRUE)) + LAYOUT_NO_OPERAND + // 186
	string(rune(LAYOUT_LIST_END)) + "\x00" + string(rune(LIST_RESULTS)) + // 187
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 188

// TYPE_PARAMETERS_RULE writes the type parameters of one signature, and hands back whether the
// walk stands on a part behind them. A signature that binds no type parameter writes no brackets.
const TYPE_PARAMETERS_RULE = string(rune(LAYOUT_LIST_BEGIN)) + LAYOUT_NO_OPERAND + // 189
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x46" + // 190
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_TYPE_PARAMETER)) + // 191
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x45" + // 192
	string(rune(LAYOUT_LIST_ITEM)) + "\x00" + string(rune(LIST_TYPE_PARAMETERS)) + // 193
	string(rune(LAYOUT_CALL)) + "\x00" + string(rune(RULE_TYPE_PARAMETER)) + // 194
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 195
	string(rune(LAYOUT_JUMP)) + "\x01\x3e" + // 196
	string(rune(LAYOUT_TRUE)) + LAYOUT_NO_OPERAND + // 197
	string(rune(LAYOUT_LIST_END)) + "\x00" + string(rune(LIST_TYPE_PARAMETERS)) + // 198
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 199

// PARAMETER_RULE writes one parameter or result: the run of names it binds and the type behind
// them. A signature that binds no name states a run of bare types where a named one states a run
// of names, thus the parts behind the names read as a run rather than as one type.
const PARAMETER_RULE = string(rune(LAYOUT_DOWN_PART)) + LAYOUT_NO_OPERAND + // 200
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x5b" + // 201
	string(rune(LAYOUT_PARTS_OPEN)) + LAYOUT_NO_OPERAND + // 202
	string(rune(LAYOUT_IS)) + "\x00" + string(rune(ast.NODE_PARAMETER_NAME)) + // 203
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x53" + // 204
	string(rune(LAYOUT_NAME_MARK)) + LAYOUT_NO_OPERAND + // 205
	string(rune(LAYOUT_TOKEN)) + LAYOUT_NO_OPERAND + // 206
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 207
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 208
	string(rune(LAYOUT_JUMP)) + "\x01\x4b" + // 209
	string(rune(LAYOUT_JUMP)) + "\x01\x59" + // 210
	string(rune(LAYOUT_NAMES_SPACE)) + LAYOUT_NO_OPERAND + // 211
	string(rune(LAYOUT_TYPE_MARK)) + LAYOUT_NO_OPERAND + // 212
	string(rune(LAYOUT_EXPRESSION)) + LAYOUT_NO_OPERAND + // 213
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 214
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 215
	string(rune(LAYOUT_JUMP)) + "\x01\x54" + // 216
	string(rune(LAYOUT_FINISH_PARTS)) + LAYOUT_NO_OPERAND + // 217
	string(rune(LAYOUT_UP)) + LAYOUT_NO_OPERAND + // 218
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 219

// TYPE_PARAMETER_RULE writes one type parameter. The tree names every part of one type parameter
// the same way, thus the parts ahead of the last one state the names and the last one states the
// constraint they bind against.
const TYPE_PARAMETER_RULE = string(rune(LAYOUT_DOWN_PART)) + LAYOUT_NO_OPERAND + // 220
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x6c" + // 221
	string(rune(LAYOUT_PARTS_OPEN)) + LAYOUT_NO_OPERAND + // 222
	string(rune(LAYOUT_STANDS_LAST)) + LAYOUT_NO_OPERAND + // 223
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 224
	string(rune(LAYOUT_JUMP)) + "\x01\x68" + // 225
	string(rune(LAYOUT_NAME_MARK)) + LAYOUT_NO_OPERAND + // 226
	string(rune(LAYOUT_TOKEN)) + LAYOUT_NO_OPERAND + // 227
	string(rune(LAYOUT_NEXT_PART)) + LAYOUT_NO_OPERAND + // 228
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 229
	string(rune(LAYOUT_JUMP)) + "\x01\x5f" + // 230
	string(rune(LAYOUT_JUMP)) + "\x01\x6a" + // 231
	string(rune(LAYOUT_NAMES_SPACE)) + LAYOUT_NO_OPERAND + // 232
	string(rune(LAYOUT_EXPRESSION)) + LAYOUT_NO_OPERAND + // 233
	string(rune(LAYOUT_FINISH_PARTS)) + LAYOUT_NO_OPERAND + // 234
	string(rune(LAYOUT_UP)) + LAYOUT_NO_OPERAND + // 235
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 236

// BLOCK_RULE writes one block: the brace that opens it, its statements one tab deeper, and the
// brace that closes it on a line of its own. A note the author left on the line the brace opens
// keeps that line, thus the print writes it before it closes the line.
const BLOCK_RULE = string(rune(LAYOUT_SPANS_ONE_LINE)) + LAYOUT_NO_OPERAND + // 237
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x71" + // 238
	string(rune(LAYOUT_INLINE_BLOCK)) + LAYOUT_NO_OPERAND + // 239
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND + // 240
	string(rune(LAYOUT_BLOCK_OPEN)) + LAYOUT_NO_OPERAND + // 241
	string(rune(LAYOUT_BYTE)) + "\x00" + string(rune('{')) + // 242
	string(rune(LAYOUT_DOWN)) + LAYOUT_NO_OPERAND + // 243
	string(rune(LAYOUT_ENTERED)) + LAYOUT_NO_OPERAND + // 244
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x02\x07" + // 245
	string(rune(LAYOUT_TRAILS_BRACE)) + LAYOUT_NO_OPERAND + // 246
	string(rune(LAYOUT_JUMP_UNLESS)) + "\x01\x7b" + // 247
	string(rune(LAYOUT_TRIVIA_HERE)) + LAYOUT_NO_OPERAND + // 248
	string(rune(LAYOUT_NEXT)) + LAYOUT_NO_OPERAND + // 249
	string(rune(LAYOUT_JUMP)) + "\x01\x75" + // 250
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 251
	string(rune(LAYOUT_INDENT_IN)) + LAYOUT_NO_OPERAND + // 252
	string(rune(LAYOUT_BLOCK_SKIP)) + LAYOUT_NO_OPERAND + // 253
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 254
	string(rune(LAYOUT_JUMP)) + "\x02\x03" + // 255
	string(rune(LAYOUT_COLUMN_OPEN)) + LAYOUT_NO_OPERAND + // 256
	string(rune(LAYOUT_STATEMENT)) + LAYOUT_NO_OPERAND + // 257
	string(rune(LAYOUT_COLUMN_CLOSE)) + LAYOUT_NO_OPERAND + // 258
	string(rune(LAYOUT_NEXT)) + LAYOUT_NO_OPERAND + // 259
	string(rune(LAYOUT_WHEN)) + LAYOUT_NO_OPERAND + // 260
	string(rune(LAYOUT_JUMP)) + "\x01\x7d" + // 261
	string(rune(LAYOUT_JUMP)) + "\x02\x09" + // 262
	string(rune(LAYOUT_LINE)) + LAYOUT_NO_OPERAND + // 263
	string(rune(LAYOUT_INDENT_IN)) + LAYOUT_NO_OPERAND + // 264
	string(rune(LAYOUT_BLOCK_CLOSE)) + LAYOUT_NO_OPERAND + // 265
	string(rune(LAYOUT_RETURN)) + LAYOUT_NO_OPERAND // 266

// LAYOUT_PROGRAM holds every rule, in the order of the rules.
const LAYOUT_PROGRAM = FILE_RULE +
	DECLARATION_RULE +
	IMPORT_RULE +
	NOTE_RULE +
	CONSTANT_RULE +
	VARIABLE_RULE +
	VALUES_RULE +
	TYPE_RULE +
	FUNCTION_RULE +
	RECEIVER_RULE +
	SIGNATURE_RULE +
	HEAD_RULE +
	RESULTS_RULE +
	TYPE_PARAMETERS_RULE +
	PARAMETER_RULE +
	TYPE_PARAMETER_RULE +
	BLOCK_RULE

// LAYOUT_ENTRIES names the step each rule opens at, in two bytes of seven bits to a
// rule, in the order of the rules.
const LAYOUT_ENTRIES = string(rune(0)) + string(rune(0)) +
	string(rune(0)) + string(rune(24)) +
	string(rune(0)) + string(rune(56)) +
	string(rune(0)) + string(rune(73)) +
	string(rune(0)) + string(rune(77)) +
	string(rune(0)) + string(rune(81)) +
	string(rune(0)) + string(rune(85)) +
	string(rune(0)) + string(rune(103)) +
	string(rune(0)) + string(rune(127)) +
	string(rune(1)) + string(rune(16)) +
	string(rune(1)) + string(rune(25)) +
	string(rune(1)) + string(rune(33)) +
	string(rune(1)) + string(rune(49)) +
	string(rune(1)) + string(rune(61)) +
	string(rune(1)) + string(rune(72)) +
	string(rune(1)) + string(rune(92)) +
	string(rune(1)) + string(rune(109))

// The layout program stands above.

// SIMPLE_OP_MAXIMUM is the last action the step runs itself.
const SIMPLE_OP_MAXIMUM = uint8(LAYOUT_TRIVIA_HERE)

// SIMPLE_OP_MINIMUM is the first action the step runs itself.
const SIMPLE_OP_MINIMUM = uint8(LAYOUT_RETURN)

// Simple_Op is one action the step runs itself.
type Simple_Op uint8

// Simple_Op_Invariants states every action the step runs itself.
func Simple_Op_Invariants(value Simple_Op, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), SIMPLE_OP_MINIMUM, SIMPLE_OP_MAXIMUM).
		Ensure()
}

// FILE_OP_MAXIMUM is the last action of the unit.
const FILE_OP_MAXIMUM = uint8(LAYOUT_CLOSE_FILE - LAYOUT_FILE)

// FILE_OP_MINIMUM is the first action of the unit.
const FILE_OP_MINIMUM = 0

// File_Op is one action of the unit layout_file writes, counted from the first.
type File_Op uint8

// File_Op_Invariants states every action the unit runs.
func File_Op_Invariants(value File_Op, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), FILE_OP_MINIMUM, FILE_OP_MAXIMUM).
		Ensure()
}

// PARTS_OP_MAXIMUM is the last action of the unit.
const PARTS_OP_MAXIMUM = uint8(LAYOUT_PASS - LAYOUT_DOWN_PART)

// PARTS_OP_MINIMUM is the first action of the unit.
const PARTS_OP_MINIMUM = 0

// Parts_Op is one action of the unit layout_parts writes, counted from the first.
type Parts_Op uint8

// Parts_Op_Invariants states every action the unit runs.
func Parts_Op_Invariants(value Parts_Op, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), PARTS_OP_MINIMUM, PARTS_OP_MAXIMUM).
		Ensure()
}

// VALUES_OP_MAXIMUM is the last action of the unit.
const VALUES_OP_MAXIMUM = uint8(LAYOUT_TAKE_HERE - LAYOUT_VALUES_OPEN)

// VALUES_OP_MINIMUM is the first action of the unit.
const VALUES_OP_MINIMUM = 0

// Values_Op is one action of the unit layout_values writes, counted from the first.
type Values_Op uint8

// Values_Op_Invariants states every action the unit runs.
func Values_Op_Invariants(value Values_Op, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), VALUES_OP_MINIMUM, VALUES_OP_MAXIMUM).
		Ensure()
}

// LISTS_OP_MAXIMUM is the last action of the unit.
const LISTS_OP_MAXIMUM = uint8(LAYOUT_BODY - LAYOUT_SIGNATURE_OVER)

// LISTS_OP_MINIMUM is the first action of the unit.
const LISTS_OP_MINIMUM = 0

// Lists_Op is one action of the unit layout_lists writes, counted from the first.
type Lists_Op uint8

// Lists_Op_Invariants states every action the unit runs.
func Lists_Op_Invariants(value Lists_Op, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), LISTS_OP_MINIMUM, LISTS_OP_MAXIMUM).
		Ensure()
}

// NAMES_OP_MAXIMUM is the last action of the unit.
const NAMES_OP_MAXIMUM = uint8(LAYOUT_PARAMETERS_CLOSE - LAYOUT_PARTS_OPEN)

// NAMES_OP_MINIMUM is the first action of the unit.
const NAMES_OP_MINIMUM = 0

// Names_Op is one action of the unit layout_names writes, counted from the first.
type Names_Op uint8

// Names_Op_Invariants states every action the unit runs.
func Names_Op_Invariants(value Names_Op, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), NAMES_OP_MINIMUM, NAMES_OP_MAXIMUM).
		Ensure()
}

// BLOCKS_OP_MAXIMUM is the last action of the unit.
const BLOCKS_OP_MAXIMUM = uint8(LAYOUT_BLOCK_CLOSE - LAYOUT_SPANS_ONE_LINE)

// BLOCKS_OP_MINIMUM is the first action of the unit.
const BLOCKS_OP_MINIMUM = 0

// Blocks_Op is one action of the unit layout_blocks writes, counted from the first.
type Blocks_Op uint8

// Blocks_Op_Invariants states every action the unit runs.
func Blocks_Op_Invariants(value Blocks_Op, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Uint8(uint8(value), BLOCKS_OP_MINIMUM, BLOCKS_OP_MAXIMUM).
		Ensure()
}

// Writes one note in the canonical form, which opens a space behind the slashes so the words read
// as words. A directive states a name a tool reads and no space stands inside it, thus a
// directive stands as the author wrote it.
func emit_note(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_note.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_note.tree")
	// The print writes one note of the file here, thus the count of them reads what the
	// file states once the print closed.
	subject.Counts.Values[COUNT_NOTE] = subject.Counts.Values[COUNT_NOTE] + 1
	// A note that follows a byte of its line stands apart from what it follows, and a note
	// that opens a line stands behind the tabs that line owes, thus neither closes up against
	// a byte a reader reads.
	written := int(written_load(subject.Counts.Values))
	if bool(writes_canonical(subject.Flags.Values)) {
		if written > 0 {
			held := subject.Form.Values[written-1]
			apart := Boolean(held == byte(SYMBOL_SPACE))
			if held == '\t' {
				apart = true
			}
			if held == '\n' {
				apart = true
			}
			aver.Always(bool(apart),
				"A note stands behind a space, a tab, or the line it opens.")
		}
	}
	if bool(wraps_note(subject, tree)) {
		emit_wrapped_note(subject, tree)
		return
	}
	// A note the print writes as it stands states a form of its own: a directive names a
	// tool, a block comment holds text between its own marks, and a note that opens on spaces
	// lays out lines the author counted.
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	stood := int(written_load(subject.Counts.Values))
	dropped := 0
	for index := range len(text) {
		// A return the author left inside a note states nothing of its text.
		if text[index] == '\r' {
			dropped = dropped + 1
			continue
		}
		emit_byte(subject, Symbol(text[index]))
	}
	// A note the print writes as it stands reads as the author wrote it, thus every byte of
	// it but the returns it drops lands in the form wherever the storage holds room.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(int(written_load(subject.Counts.Values))-stood == len(text)-dropped,
			"A note the print writes lands in the form but for the returns it drops.")
	}
}

// Reports whether the note the print stands on wraps at the widest line. A note stating prose on a
// line of its own wraps, because the words it holds read the same on any line. A directive, a note
// standing behind code, a block comment, and a note whose text opens on spaces of its own each
// state a form the author laid out, thus the print writes them as they stand.
func wraps_note(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "wraps_note.yes") }()
	Printer_Handle_Invariants(subject, "wraps_note.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "wraps_note.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	if len(text) < 3 {
		return false
	}
	// Every note the print writes spells the slash it opens on.
	aver.Always(text[0] == '/',
		"A note the print wraps opens on the byte every note opens with.")
	if text[1] != '/' {
		return false
	}
	if bool(states_directive(subject, tree)) {
		return false
	}
	if text[2] == '\t' {
		return false
	}
	// A note whose text opens on a run of spaces states a layout of its own, thus the print
	// keeps every line of it as the author wrote it.
	if len(text) > 3 {
		if text[2] == ' ' {
			if text[3] == ' ' {
				return false
			}
		}
	}
	held := int(indent_load(subject.Counts.Values)) * INDENT_WIDTH
	return Boolean(int(take_line_width(subject.Counts.Values)) == held)
}

// Writes one note, opening a line of its own wherever the words it holds would run past the widest
// line. The print opens every line with the tabs the note opened at and the slashes a note states,
// thus a note of any length reads as one note.
func emit_wrapped_note(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_wrapped_note.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_wrapped_note.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	head := int(take_line_width(subject.Counts.Values)) + len(NOTE_HEAD)
	emit_note_head(subject)
	column := head
	place := 2
	for place < len(text) {
		if text[place] == ' ' {
			place = place + 1
			continue
		}
		end := place
		width := 0
		for end < len(text) {
			if text[end] == ' ' {
				break
			}
			if text[end]&CHARACTER_TAIL_MASK != CHARACTER_TAIL_MARK {
				width = width + 1
			}
			// A tab stands in the columns a reader sees, thus a word that holds
			// one spends those columns the way the print spends them.
			if text[end] == '\t' {
				width = width + TAB_WIDTH - 1
			}
			// A return states nothing of the word it stands in.
			if text[end] == '\r' {
				width = width - 1
			}
			end = end + 1
		}
		if column > head {
			if column+width+1 > COLUMN_WIDTH_MAXIMUM {
				emit_line(subject)
				emit_indent(subject)
				emit_note_head(subject)
				column = head
			}
		}
		if column > head {
			emit_byte(subject, ' ')
			column = column + 1
		}
		for index := place; index < end; index++ {
			if text[index] == '\r' {
				continue
			}
			emit_byte(subject, Symbol(text[index]))
		}
		column = column + width
		place = end
		// A line that holds one word alone holds whatever that word spans, and every
		// other line of the wrap closes inside the widest line.
		if column > head+width {
			aver.Always(column <= COLUMN_WIDTH_MAXIMUM,
				"A line of a wrapped note stands inside the widest line.")
		}
	}
	// The wrap counts the columns it wrote, and the line it closes holds them.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(take_line_width(subject.Counts.Values)) == column,
			"A wrapped note counts the columns its line holds.")
	}
}

// NOTE_HEAD is the text one note opens with, the slashes and the space behind them.
const NOTE_HEAD = "// "

// Writes the slashes one note opens with and the space behind them.
func emit_note_head(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "emit_note_head.subject")
	stood := take_line_width(subject.Counts.Values)
	for index := range len(NOTE_HEAD) {
		emit_byte(subject, Symbol(NOTE_HEAD[index]))
	}
	// A wrap counts the slashes it opens each line with, thus the bytes the print writes
	// here and the width that count reads state one head.
	if bool(writes_canonical(subject.Flags.Values)) {
		spent := int(take_line_width(subject.Counts.Values)) - int(stood)
		aver.Always(spent == len(NOTE_HEAD),
			"The head of one note spends the width the wrap counts for it.")
	}
}

// Reports whether one note states a directive a tool reads rather than a sentence a reader reads.
// A directive names a tool and a word behind a colon, or one of the names the toolchain reserved
// long before that shape settled.
func states_directive(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_directive.yes") }()
	Printer_Handle_Invariants(subject, "states_directive.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "states_directive.tree")
	if bool(names_tool(subject, tree)) {
		return true
	}
	return reserves_name(subject, tree)
}

// Reports whether one note names a tool and a word behind a colon, as go:generate does.
func names_tool(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_tool.yes") }()
	Printer_Handle_Invariants(subject, "names_tool.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "names_tool.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	// A directive stands inside a note, thus the token the read stands on spells the two
	// bytes every note opens with.
	aver.Always(len(text) >= NOTE_MARK_SIZE,
		"A note the read of a directive stands on spells the bytes it opens with.")
	place := NOTE_MARK_SIZE
	for place < len(text) {
		// A tool names itself in lower case and holds a hyphen between its words, thus
		// the name closes at the first byte that states neither.
		letter := text[place] >= 'a'
		if text[place] > 'z' {
			letter = false
		}
		if text[place] == '-' {
			letter = true
		}
		if !letter {
			break
		}
		place = place + 1
	}
	if place == NOTE_MARK_SIZE {
		return false
	}
	if place >= len(text) {
		return false
	}
	if text[place] != ':' {
		return false
	}
	if place+1 >= len(text) {
		return false
	}
	if text[place+1] < 'a' {
		return false
	}
	return Boolean(text[place+1] <= 'z')
}

// Reports whether one note opens with a name the toolchain reserved for a directive of its own.
func reserves_name(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "reserves_name.yes") }()
	Printer_Handle_Invariants(subject, "reserves_name.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "reserves_name.tree")
	if bool(opens_word(subject, tree, Reserved_Name("line "))) {
		return true
	}
	if bool(opens_word(subject, tree, Reserved_Name("export "))) {
		return true
	}
	if bool(opens_word(subject, tree, Reserved_Name("extern "))) {
		return true
	}
	if bool(opens_word(subject, tree, Reserved_Name("sysnb "))) {
		return true
	}
	if bool(opens_word(subject, tree, Reserved_Name("sys "))) {
		return true
	}
	return opens_word(subject, tree, Reserved_Name("nolint"))
}

// Writes one number literal in the form the canonical source states: a base prefix and an
// exponent mark in lower case, and no leading zeros on an imaginary whole number. The digits
// themselves stand as the author wrote them, because case carries meaning in no base.
func emit_number(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_number.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_number.tree")
	stood := int(written_load(subject.Counts.Values))
	emit_number_form(subject, tree)
	// A number states a value and never a sign, thus the byte it opens with spells a digit of
	// the base the source states or the point a fraction below one opens with.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(written_load(subject.Counts.Values)) > stood,
			"A number the print writes lands a byte in the form.")
		held := subject.Form.Values[stood]
		opened := Boolean(held == '.')
		if held >= '0' {
			if held <= '9' {
				opened = true
			}
		}
		aver.Always(bool(opened), "A number opens on a digit of its base or on a point.")
	}
	// Neither the eight base nor the two base states a fraction or an exponent, thus a mark
	// of one standing behind the prefix the print wrote spells a literal no reader reads.
	written := int(written_load(subject.Counts.Values))
	based := Boolean(false)
	if written > stood+1 {
		based = Boolean(subject.Form.Values[stood+1] == 'o')
		if subject.Form.Values[stood+1] == 'b' {
			based = true
		}
	}
	// The canonical form names a base and states an exponent in lower case, thus the byte
	// behind the one the number opens with spells no letter of the upper case.
	if written > stood+1 {
		upper := Boolean(subject.Form.Values[stood+1] >= 'A')
		if subject.Form.Values[stood+1] > 'Z' {
			upper = false
		}
		aver.Always(!bool(upper),
			"A number names the base and the exponent it states in lower case.")
	}
	if bool(based) {
		marked := Boolean(false)
		for place := stood + 2; place < written; place++ {
			switch subject.Form.Values[place] {
			case 'e', 'E', '.', 'p', 'P':
				marked = true
			}
		}
		aver.Always(!bool(marked),
			"A number of a base the print names spells digits of that base alone.")
	}
}

// Writes the bytes of one number literal.
func emit_number_form(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_number_form.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_number_form.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	// A number spells the digits the author wrote, thus the token it stands on spends bytes
	// of the source.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(len(text) > 0, "A number the print writes spells a byte of the source.")
	}
	// A literal of one byte states neither a base prefix nor an exponent.
	if len(text) < 2 {
		emit_digits(subject, tree, EXPONENT_NONE, DIGITS_ALL)
		return
	}
	base := number_base(subject, tree)
	if base != BASE_NONE {
		emit_byte(subject, '0')
		emit_byte(subject, Symbol(base))
		// Only a hexadecimal literal states a power, and no other base states an
		// exponent at all.
		exponent := EXPONENT_NONE
		if base == BASE_HEXADECIMAL {
			exponent = EXPONENT_POWER
		}
		emit_digits(subject, tree, exponent, DIGITS_PAST_BASE)
		return
	}
	// An exponent stands where a whole number cannot, thus a literal that holds one keeps every
	// zero it opens with.
	if bool(holds_exponent(subject, tree)) {
		emit_digits(subject, tree, EXPONENT_DECIMAL, DIGITS_ALL)
		return
	}
	if bool(whole_imaginary(subject, tree)) {
		emit_digits(subject, tree, EXPONENT_NONE, DIGITS_PAST_ZEROS)
		return
	}
	// A whole number the source opens with a zero states the eight base, and the canonical form
	// names that base rather than leaving a reader to count digits.
	if !bool(states_octal(subject, tree)) {
		emit_digits(subject, tree, EXPONENT_NONE, DIGITS_ALL)
		return
	}
	emit_byte(subject, '0')
	emit_byte(subject, Symbol(BASE_OCTAL))
	emit_digits(subject, tree, EXPONENT_NONE, DIGITS_PAST_ZERO)
}

// DIGITS_ALL names a literal whose digits the print writes from its first byte.
const DIGITS_ALL = 0

// DIGITS_PAST_ZERO names a literal of the eight base that opens with a bare zero, which the print
// writes as the prefix it names.
const DIGITS_PAST_ZERO = 1

// DIGITS_PAST_BASE names a literal whose base prefix the print already wrote.
const DIGITS_PAST_BASE = 2

// DIGITS_PAST_ZEROS names an imaginary whole number, whose leading zeros the canonical form
// drops.
const DIGITS_PAST_ZEROS = 3

// Digits is which bytes of one number literal the print writes as digits, which is every byte
// behind the prefix the print writes of its own.
type Digits int

// Digits_Invariants states every prefix a number literal writes ahead of its digits.
func Digits_Invariants(value Digits, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value), DIGITS_ALL, DIGITS_PAST_ZERO, DIGITS_PAST_BASE,
			DIGITS_PAST_ZEROS,
		).
		Ensure()
}

// Names the base one number literal states, in the case the canonical form writes it. Zero names
// a literal that states no base prefix at all.
func number_base(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (base Base) {
	defer func() { Base_Invariants(base, "number_base.base") }()
	Printer_Handle_Invariants(subject, "number_base.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "number_base.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	// A base states the zero it opens on and the letter that names it, thus a literal of one
	// byte alone states none and the read stands on a literal of two bytes at least.
	aver.Always(len(text) >= BASE_MARK_SIZE,
		"A read of the base one number states stands on the bytes that name it.")
	if text[0] != '0' {
		return BASE_NONE
	}
	if text[1] == 'X' {
		return BASE_HEXADECIMAL
	}
	if text[1] == 'x' {
		return BASE_HEXADECIMAL
	}
	if text[1] == 'O' {
		return BASE_OCTAL
	}
	if text[1] == 'o' {
		return BASE_OCTAL
	}
	if text[1] == 'B' {
		return BASE_BINARY
	}
	if text[1] == 'b' {
		return BASE_BINARY
	}
	return BASE_NONE
}

// Reports whether one number literal states an exponent or a fraction, which a whole number
// states neither of.
func holds_exponent(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_exponent.yes") }()
	Printer_Handle_Invariants(subject, "holds_exponent.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_exponent.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	// A number spells digits and the marks its base and its part state, thus the read
	// stands on a number of the tree rather than on any other token.
	numbered := Boolean(one.Kind == token.KIND_INTEGER)
	if one.Kind == token.KIND_FLOAT {
		numbered = true
	}
	if one.Kind == token.KIND_IMAGINARY {
		numbered = true
	}
	aver.Always(bool(numbered), "A read of the exponent one number states stands on a number.")
	for index := range len(text) {
		if text[index] == 'E' {
			return true
		}
		if text[index] == 'e' {
			return true
		}
	}
	return false
}

// Reports whether one number literal states an imaginary whole number, which is the one literal
// the canonical form strips leading zeros from.
func whole_imaginary(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "whole_imaginary.yes") }()
	Printer_Handle_Invariants(subject, "whole_imaginary.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "whole_imaginary.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	// A number spells digits and the marks its base and its part state, thus the read
	// stands on a number of the tree rather than on any other token.
	numbered := Boolean(one.Kind == token.KIND_INTEGER)
	if one.Kind == token.KIND_FLOAT {
		numbered = true
	}
	if one.Kind == token.KIND_IMAGINARY {
		numbered = true
	}
	aver.Always(bool(numbered), "A read of one imaginary number stands on a number.")
	if text[len(text)-1] != 'i' {
		return false
	}
	for index := range len(text) {
		if text[index] == '.' {
			return false
		}
		if text[index] == 'e' {
			return false
		}
	}
	return true
}

// Writes the digits of one number literal from the byte the caller names, with each mark it holds
// written in the case the canonical form states. A mark of zero names a literal that holds none.
func emit_digits(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, exponent Exponent, digits Digits,
) {
	Printer_Handle_Invariants(subject, "emit_digits.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_digits.tree")
	Exponent_Invariants(exponent, "emit_digits.exponent")
	Digits_Invariants(digits, "emit_digits.digits")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	stood := written_load(subject.Counts.Values)
	head := int(digits)
	if digits == DIGITS_PAST_ZEROS {
		// The digits of an imaginary whole number open behind every zero and every
		// underscore the author wrote ahead of them, and a literal of zeros alone states
		// the number zero, which is one digit and never none.
		head = 0
		for head < len(text)-1 {
			if text[head] != '0' {
				if text[head] != '_' {
					break
				}
			}
			head = head + 1
		}
		if head == len(text)-1 {
			emit_byte(subject, '0')
		}
	}
	for index := head; index < len(text); index++ {
		value := Symbol(text[index])
		if exponent != EXPONENT_NONE {
			if value == Symbol(exponent) {
				// The canonical form states an exponent mark in lower case, and the
				// letters stand thirty two apart in the alphabet the source spells.
				value = value + 32
			}
		}
		emit_byte(subject, value)
	}
	// An imaginary whole number states the count it holds, thus the digits it wrote open on
	// no zero unless the number is zero itself, and an exponent mark reads in lower case.
	if bool(writes_canonical(subject.Flags.Values)) {
		form := subject.Form.Values
		written := int(written_load(subject.Counts.Values))
		if digits == DIGITS_PAST_ZEROS {
			if written > int(stood)+2 {
				aver.Always(form[int(stood)] != '0',
					"An imaginary whole number opens on no zero of its own.")
			}
		}
		if exponent != EXPONENT_NONE {
			upper := false
			for place := int(stood); place < written; place++ {
				if form[place] == byte(exponent) {
					upper = true
				}
			}
			aver.Always(!upper,
				"A number the print writes states its exponent in lower case.")
		}
	}
	// A number states the digits it holds, thus the form holds a digit of it at least.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A number the print writes states a digit of its own.")
	}
}

// Writes the empty line one declaration owes the declarations ahead of it. A declaration of
// another kind, and every note that opens one, stand behind an empty line, thus a reader sees
// where one run of declarations closes and the next one opens.
func open_declaration(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, commented Boolean, same Boolean,
) {
	Printer_Handle_Invariants(subject, "open_declaration.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_declaration.tree")
	Boolean_Invariants(commented, "open_declaration.commented")
	Boolean_Invariants(same, "open_declaration.same")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLANK {
		return
	}
	// A note opens the run its declaration stands in, thus the note stands behind the empty
	// line and everything behind that note stands inside the run it opened.
	if bool(commented) {
		return
	}
	// A declaration of the kind the one ahead of it states stands in the same run.
	if kind != ast.NODE_COMMENT {
		if bool(same) {
			return
		}
	}
	if bool(stands_after_blank(subject)) {
		return
	}
	emit_line(subject)
	// A declaration of one kind stands apart from the declaration of another kind ahead of
	// it, thus an empty line stands between the two.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(stands_after_blank(subject)),
			"A declaration of one kind stands an empty line apart from another kind.")
	}
}

// Reports whether one number literal states the eight base by the zero it opens with, which the
// canonical form writes as a base of its own. The number zero states no base, thus a literal of
// one byte states none either.
func states_octal(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_octal.yes") }()
	Printer_Handle_Invariants(subject, "states_octal.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "states_octal.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	// A number spells digits and the marks its base and its part state, thus the read
	// stands on a number of the tree rather than on any other token.
	numbered := Boolean(one.Kind == token.KIND_INTEGER)
	if one.Kind == token.KIND_FLOAT {
		numbered = true
	}
	if one.Kind == token.KIND_IMAGINARY {
		numbered = true
	}
	aver.Always(bool(numbered), "A read of the base one number opens on stands on a number.")
	if len(text) < 2 {
		return false
	}
	if text[0] != '0' {
		return false
	}
	if text[len(text)-1] == 'i' {
		return false
	}
	for index := range len(text) {
		if text[index] == '.' {
			return false
		}
	}
	return true
}

// Reads the imports of the file whose name the print drops. An import states a name the path
// already names, or a name the forms that read it can name by the path instead, thus the print
// writes the path alone. A name another import of the file already binds stands as the author
// wrote it, because two packages of one name name neither.
func take_aliases(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "take_aliases.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_aliases.tree")
	for slot := range IMPORT_COUNT_MAXIMUM {
		subject.Aliases.Values[slot] = 0
	}
	// The table holds the names of one file alone, thus it opens empty however many files
	// the caller prints through one printer.
	aver.Always(alias_count(subject) == 0,
		"A read of the names one file binds opens on an empty table.")
	depth := depth_load(subject.Counts.Values)
	file := subject.Nodes.Values[depth]
	child := ast.Index(ast.Syntax_Node_At(tree, file).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return
		}
		held := ast.Syntax_Node_At(tree, child)
		if held.Kind == ast.NODE_IMPORT {
			subject.Nodes.Values[depth+1] = child
			depth_store(subject.Counts.Values, depth+1)
			hold_alias(subject, tree)
			depth_store(subject.Counts.Values, depth)
		}
		child = ast.Index(held.Next)
	}
}

// Holds the path of the import the walk stands on in the slot one past the import, and reports
// whether the import binds a name to that path.
func enter_path(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "enter_path.yes") }()
	Printer_Handle_Invariants(subject, "enter_path.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "enter_path.tree")
	depth := depth_load(subject.Counts.Values)
	name := ast.Index(ast.Syntax_Node_At(tree, subject.Nodes.Values[depth]).First_Child)
	if name == ast.INDEX_ABSENT {
		return false
	}
	if ast.Syntax_Node_At(tree, name).Kind != ast.NODE_IMPORT_NAME {
		return false
	}
	// A note or an empty line the author left between the name and the path states neither,
	// thus the read steps over the trivia standing between the two.
	path := ast.Index(ast.Syntax_Node_At(tree, name).Next)
	for range ast.NODE_COUNT_MAXIMUM {
		if path == ast.INDEX_ABSENT {
			return false
		}
		kind := ast.Syntax_Node_At(tree, path).Kind
		if kind != ast.NODE_COMMENT {
			if kind != ast.NODE_BLANK {
				break
			}
		}
		path = ast.Index(ast.Syntax_Node_At(tree, path).Next)
	}
	// The walk stands one slot below the import it reads, and the storage holds that slot,
	// thus the read lands inside the walk rather than past it.
	aver.Always(int(depth)+1 < DEPTH_MAXIMUM,
		"A read of one import path lands inside the walk.")
	subject.Nodes.Values[depth+1] = path
	// The path one import binds spells a string, thus the read lands on such a form.
	aver.Always(ast.Syntax_Node_At(tree, path).Kind == ast.NODE_STRING,
		"A read of one import path lands on the string it spells.")
	return true
}

// Holds the name of the import the walk stands on where the print drops that name: a path that
// states a name of its own, and no other import that binds that name.
func hold_alias(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "hold_alias.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "hold_alias.tree")
	depth := depth_load(subject.Counts.Values)
	if !bool(enter_path(subject, tree)) {
		return
	}
	name := ast.Index(ast.Syntax_Node_At(tree, subject.Nodes.Values[depth]).First_Child)
	path := subject.Nodes.Values[depth+1]
	if !bool(names_path(subject, tree)) {
		return
	}
	if bool(clobbers_name(subject, tree)) {
		return
	}
	count := int(alias_count(subject))
	if count >= IMPORT_COUNT_MAXIMUM {
		return
	}
	// The table fills from its first slot and the read of it closes at the first empty one,
	// thus a name lands where no name stands and the read holds every name the file bound.
	aver.Always(subject.Aliases.Values[count] == 0,
		"A name the file binds lands on an empty slot of the table.")
	// An import binds the name the author wrote to the path it states, thus the table holds
	// a name of the source and the string that path spells.
	if bool(writes_canonical(subject.Flags.Values)) {
		held_name := ast.Syntax_Token_At(tree, ast.Syntax_Node_At(tree, name).Token)
		aver.Always(held_name.Kind == token.KIND_IDENTIFIER,
			"A name the table holds spells an identifier of the source.")
		aver.Always(ast.Syntax_Node_At(tree, path).Kind == ast.NODE_STRING,
			"A path the table holds states a string of the tree.")
	}
	subject.Aliases.Values[count] = ast.Syntax_Node_At(tree, name).Token
	subject.Paths.Values[count] = path
}

// ALIAS_COUNT_MINIMUM is the count of a file whose print drops no import name.
const ALIAS_COUNT_MINIMUM = 0

// Alias_Count is how many import names the print drops.
type Alias_Count int

// Alias_Count_Invariants states every count of import names the storage holds.
func Alias_Count_Invariants(value Alias_Count, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(int(value), ALIAS_COUNT_MINIMUM, IMPORT_COUNT_MAXIMUM).
		Ensure()
}

// Counts the import names the print drops, which fill the alias storage up to its first empty
// slot. The first token of a file is the word that opens its package clause and never an import
// name, thus a slot that names that token is empty.
func alias_count(subject Printer_Handle) (count Alias_Count) {
	defer func() { Alias_Count_Invariants(count, "alias_count.count") }()
	Printer_Handle_Invariants(subject, "alias_count.subject")
	for slot := range IMPORT_COUNT_MAXIMUM {
		if subject.Aliases.Values[slot] == 0 {
			return Alias_Count(slot)
		}
	}
	return IMPORT_COUNT_MAXIMUM
}

// Reports whether another import of the file already binds the name the import the walk stands on
// would take once the print drops the name the author wrote for it.
func clobbers_name(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "clobbers_name.yes") }()
	Printer_Handle_Invariants(subject, "clobbers_name.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "clobbers_name.tree")
	depth := depth_load(subject.Counts.Values)
	one := subject.Nodes.Values[depth]
	// The read asks what name one import would take, thus it stands on an import.
	aver.Always(ast.Syntax_Node_At(tree, one).Kind == ast.NODE_IMPORT,
		"A read for a name the file already binds stands on an import.")
	file := ast.Index(ast.Syntax_Node_At(tree, one).Parent)
	child := ast.Index(ast.Syntax_Node_At(tree, file).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return false
		}
		held := ast.Syntax_Node_At(tree, child)
		if held.Kind == ast.NODE_IMPORT {
			if child != one {
				subject.Nodes.Values[depth+1] = child
				depth_store(subject.Counts.Values, depth+1)
				same := binds_same_name(subject, tree)
				depth_store(subject.Counts.Values, depth)
				if bool(same) {
					return true
				}
			}
		}
		child = ast.Index(held.Next)
	}
	return false
}

// Writes the name the path of a dropped import states where the name the walk stands on is one
// the print dropped from an import, and reports whether it wrote it. A name of any other form
// stands as the author wrote it, thus only the head of a selector answers here.
func emit_alias(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "emit_alias.yes") }()
	Printer_Handle_Invariants(subject, "emit_alias.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_alias.tree")
	if node_kind(subject, tree) != ast.NODE_IDENTIFIER {
		return false
	}
	depth := depth_load(subject.Counts.Values)
	one := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth]).Token
	source := token.Source(subject.Source.Values)
	text := token.Text(source, ast.Syntax_Token_At(tree, one))
	for slot := range int(alias_count(subject)) {
		held := token.Text(source, ast.Syntax_Token_At(tree, subject.Aliases.Values[slot]))
		if string(held) != string(text) {
			continue
		}
		// The form read the name its path states, thus the count reads that name, and
		// the name the author wrote stands where the print no longer writes it.
		counts_literal(subject, tree)
		subject.Counts.Values[COUNT_NAME_DROPPED] += 1
		subject.Nodes.Values[depth+1] = subject.Paths.Values[slot]
		depth_store(subject.Counts.Values, depth+1)
		stood := written_load(subject.Counts.Values)
		emit_package_name(subject, tree)
		depth_store(subject.Counts.Values, depth)
		// A form that read an import by the name the author bound names the package the
		// path states, thus the print writes that name in place of the one it dropped.
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(written_load(subject.Counts.Values) > stood,
				"A form that reads a dropped name writes the name of its path.")
			// The print drops a name only where the path states one a file can
			// bind, thus the name it writes here spells such a name.
			opened := subject.Form.Values[int(stood)]
			bound := Boolean(opened == '_')
			if opened >= 'a' {
				if opened <= 'z' {
					bound = true
				}
			}
			if opened >= 'A' {
				if opened <= 'Z' {
					bound = true
				}
			}
			aver.Always(bool(bound),
				"A name read in place of a dropped one opens on a letter.")
		}
		return true
	}
	return false
}

// Writes the name the path the walk stands on states, which is the element that path closes
// with. A path closing at the default or the internal element names the element ahead of that
// one, because such a directory holds what its parent names rather than a package of its own. A
// path of no element states no name, thus the print writes nothing for it.
func emit_package_name(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_package_name.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_package_name.tree")
	node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	source := token.Source(subject.Source.Values)
	text := token.Text(source, ast.Syntax_Token_At(tree, node.Token))
	if len(text) < 3 {
		return
	}
	text = text[1 : len(text)-1]
	for range DEPTH_MAXIMUM {
		opening_count := len(text)
		for opening_count > 0 {
			if text[opening_count-1] == '/' {
				break
			}
			opening_count = opening_count - 1
		}
		held := text[opening_count:]
		// A path closing on the byte that parts its elements states an element of no
		// byte, which names no package, thus the read steps over it the way it steps
		// over the elements a component keeps to itself.
		if len(held) > 0 {
			if string(held) != DEFAULT_TAIL {
				if string(held) != INTERNAL_TAIL {
					text = held
					break
				}
			}
		}
		if opening_count == 0 {
			break
		}
		text = text[:opening_count-1]
	}
	// A path names the package its last element spells, and a path that closes at default or
	// at internal names the element ahead of that one, thus the name spells bytes of its own.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(len(text) > 0, "The name one path states spells bytes of its own.")
		// A name stands for one element of the path, thus no byte that parts two
		// elements stands inside it.
		parted := false
		for index := range len(text) {
			if text[index] == '/' {
				parted = true
			}
		}
		aver.Always(!parted, "The name one path states spells one element of it.")
	}
	for index := range len(text) {
		emit_byte(subject, Symbol(text[index]))
	}
}

// Reports whether the print drops the import name the walk stands on, which the run of names it
// read from the imports of the file states.
func drops_alias(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "drops_alias.yes") }()
	Printer_Handle_Invariants(subject, "drops_alias.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "drops_alias.tree")
	current := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	one := ast.Syntax_Node_At(tree, current).Token
	for slot := range int(alias_count(subject)) {
		if subject.Aliases.Values[slot] == one {
			return true
		}
	}
	return false
}

// Writes the name the import at the top of the walk stack binds past the bytes the print already
// wrote, and the name the path of the import beneath it states behind that, then reports whether
// the two read the same. The read writes nothing it keeps, thus it costs no storage of its own.

// Writes one slice suffix: the value it reads and the bounds it states between colons.
func print_slice_expression(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_slice_expression.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_slice_expression.tree")
	// A slice reads the bounds it holds from the value ahead of it, thus the walk stands on
	// one where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_SLICE_EXPRESSION,
			"A slice the print writes stands on a slice of the tree.")
	}
	subject.Counts.Values[COUNT_FORM] += 1
	subject.Counts.Values[COUNT_SLICE] += 1
	// The tree holds no node for a bound the source left out, thus the print reads the colons
	// from the token run and writes as many as the author wrote.
	written := take_bounds(subject, tree)
	blanks := slice_blanks(subject, tree)
	held := nest_load(subject.Counts.Values)
	if !bool(descend_part(subject, tree)) {
		emit_byte(subject, '[')
		emit_byte(subject, ']')
		return
	}
	nest_store(subject.Counts.Values, NEST_MINIMUM)
	print_expression(subject, tree)
	nest_store(subject.Counts.Values, held)
	emit_byte(subject, '[')
	print_slice_bounds(subject, tree, written, blanks)
	finish_parts(subject, tree)
	ascend(subject)
	emit_byte(subject, ']')
	// A slice holds the bounds it reads between brackets and stands at the nest the value
	// that holds it stood at, thus it closes on that bracket and hands the nest back.
	if bool(writes_canonical(subject.Flags.Values)) {
		place := int(written_load(subject.Counts.Values))
		aver.Always(subject.Form.Values[place-1] == ']',
			"A slice closes on the bracket that holds the bounds it reads.")
		aver.Always(nest_load(subject.Counts.Values) == held,
			"A slice closes at the nest it opened at.")
	}
}

// Writes the bounds of the slice the walk stands in and the colons between them. The walk stands
// on the value the slice reads, and the tree holds no node for a colon, thus the print reads each
// colon from the token run between the brackets of the slice. A colon inside a bracket of its own
// belongs to the form that bracket opens, thus only a colon the slice itself holds counts.
func print_slice_bounds(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, count Slice_Parts, blanks Boolean,
) {
	Printer_Handle_Invariants(subject, "print_slice_bounds.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_slice_bounds.tree")
	Slice_Parts_Invariants(count, "print_slice_bounds.count")
	Boolean_Invariants(blanks, "print_slice_bounds.blanks")
	held, depth := nest_load(subject.Counts.Values), depth_load(subject.Counts.Values)
	// The bounds stand under the form that holds them.
	aver.Always(depth > 0, "The bounds of one slice stand under the form that holds them.")
	depth_store(subject.Counts.Values, depth-1)
	closer := int(closer_position(subject, tree))
	depth_store(subject.Counts.Values, depth)
	opener := int(ast.Syntax_Node_At(tree, subject.Nodes.Values[depth-1]).Token)
	open := advance_part(subject, tree)
	parts, level := 1, 0
	for place := opener + 1; place < closer; place++ {
		if bool(open) {
			enter_leftmost(subject, tree)
			leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
			depth_store(subject.Counts.Values, depth)
			if int(ast.Syntax_Node_At(tree, leftmost).Token) == place {
				nest_store(subject.Counts.Values, Nest(nest_inner(subject, held)))
				stood := Slice_Parts(min(parts, SLICE_PARTS_CAPACITY))
				print_bound(subject, tree, owes_bound(stood, count))
				nest_store(subject.Counts.Values, held)
				parts = parts + 1
				open = advance_part(subject, tree)
			}
		}
		kind := ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind
		step := bracket_step(token.Scanned_Kind(kind))
		if step == BRACKET_OPENS {
			level = level + 1
		}
		if step == BRACKET_CLOSES {
			level = level - 1
		}
		if kind != token.KIND_COLON {
			continue
		}
		if level != 0 {
			continue
		}
		// A space stands only against a bound the source states, thus the bracket and a
		// second colon on either side take none.
		behind := blanks
		if ast.Syntax_Token_At(tree, ast.Token_Index(place-1)).Kind == token.KIND_COLON {
			behind = false
		}
		ahead := blanks
		if ast.Syntax_Token_At(tree, ast.Token_Index(place+1)).Kind == token.KIND_COLON {
			ahead = false
		}
		if place-1 == opener {
			behind = false
		}
		if place+1 == closer {
			ahead = false
		}
		emit_colon(subject, behind, ahead)
	}
	// Every bound a slice states stands between the brackets it opens and closes at, thus
	// the walk over those tokens meets every part the tree hangs under the form.
	aver.Always(!bool(open), "A slice reads every bound the tree hangs under it.")
	// Every bracket the walk stepped into it stepped out of again.
	aver.Always(level == 0, "The tokens of one slice close every bracket they open.")
}

// SLICE_PARTS_VALUE is the value a slice reads alone, which is all a slice of no bound writes.
const SLICE_PARTS_VALUE = 1

// SLICE_PARTS_LOW is the value and one bound.
const SLICE_PARTS_LOW = 2

// SLICE_PARTS_HIGH is the value and two bounds.
const SLICE_PARTS_HIGH = 3

// SLICE_PARTS_CAPACITY is the value and all three bounds a slice states.
const SLICE_PARTS_CAPACITY = 4

// Slice_Parts is how many parts of one slice the print writes: the value it reads and every bound
// it states except a closing bound that states the length of that value.
type Slice_Parts int

// Slice_Parts_Invariants states every count of parts one slice writes.
func Slice_Parts_Invariants(value Slice_Parts, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Int(
			int(value), SLICE_PARTS_VALUE, SLICE_PARTS_LOW, SLICE_PARTS_HIGH,
			SLICE_PARTS_CAPACITY,
		).
		Ensure()
}

// Reads how many parts of the slice the walk stands on the print writes. A slice that closes at
// the length of the value it reads closes where the brackets already close it, thus that bound
// states nothing and the canonical form drops it.
func take_bounds(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (parts Slice_Parts) {
	defer func() { Slice_Parts_Invariants(parts, "take_bounds.parts") }()
	Printer_Handle_Invariants(subject, "take_bounds.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_bounds.tree")
	elide := elides_bound(subject, tree)
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	written := 0
	open := descend(subject, tree)
	for bool(open) {
		// The spread that stands behind a value belongs to that value, thus it states no
		// bound of its own.
		if node_kind(subject, tree) != ast.NODE_ELLIPSIS {
			if !bool(trivia_kind(subject, tree)) {
				written = written + 1
			}
		}
		open = advance(subject, tree)
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	if bool(elide) {
		// The bound the print drops states the length the brackets already close at,
		// thus the slice held a part for it before the read took that part back.
		aver.Always(written > 0, "A slice that drops a bound held that bound.")
		written = written - 1
	}
	// A tree the parse refused holds a slice of any count of parts, and the brackets of one
	// slice admit the value it reads and three bounds, thus the print writes those and no more.
	return Slice_Parts(min(max(written, SLICE_PARTS_VALUE), SLICE_PARTS_CAPACITY))
}

// Reports whether the bound that closes one slice states the length of the value that slice
// reads. A slice of three bounds names a capacity of its own, thus its closing bound stands
// however it reads, and a slice of one colon closes at its own end with no bound at all.
func elides_bound(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "elides_bound.yes") }()
	Printer_Handle_Invariants(subject, "elides_bound.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "elides_bound.tree")
	closer := int(closer_position(subject, tree))
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	opener := int(ast.Syntax_Node_At(tree, index).Token)
	if closer < opener+4 {
		return false
	}
	if !bool(holds_one_colon(subject, tree)) {
		return false
	}
	// A value of one name reads the same twice however the package binds it, and a value of
	// any wider form may read otherwise, thus only a name closes a slice at its own length.
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	// The read steps into the slice and never out of it, thus it stands at the slot it
	// opened or below it, and the walk stands back once it read the token.
	aver.Always(depth_load(subject.Counts.Values) >= held_depth,
		"A read of the value one slice elides stands below the slot it opened.")
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	if int(ast.Syntax_Node_At(tree, leftmost).Token)+1 != opener {
		return false
	}
	return reads_same(subject, tree)
}

// Reports whether the closing bound of the slice the walk stands on reads the length of the one
// name the slice reads. Two forms of one spelling state one value, thus a slice that closes at
// the length of its own value closes at its end.
func reads_same(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "reads_same.yes") }()
	Printer_Handle_Invariants(subject, "reads_same.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "reads_same.tree")
	// The read reads the tokens of one slice and writes none of them, thus the form stands
	// as wide once it closed as it stood when it opened.
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the length one slice closes at writes no byte.")
	}()
	closer := int(closer_position(subject, tree))
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	opener := int(ast.Syntax_Node_At(tree, index).Token)
	// The read opens at the token the slice opens on and closes at the bracket that closes
	// it, thus that bracket stands behind that token.
	aver.Always(closer > opener,
		"A slice closes on a bracket behind the token it opens on.")
	// A note the author left inside the brackets states nothing of the bound they hold, thus
	// the read steps over every note it meets.
	last := closer - 1
	for last > opener {
		if ast.Syntax_Token_At(tree, ast.Token_Index(last)).Kind != token.KIND_COMMENT {
			break
		}
		last = last - 1
	}
	// The length reads one value inside brackets of its own, thus the call closes at the
	// bracket ahead of the one that closes the slice.
	before := ast.Syntax_Token_At(tree, ast.Token_Index(last))
	if before.Kind != token.KIND_PARENTHESIS_RIGHT {
		return false
	}
	depth := 0
	place := last
	for place > opener {
		kind := ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind
		if kind == token.KIND_PARENTHESIS_RIGHT {
			depth = depth + 1
		}
		if kind == token.KIND_PARENTHESIS_LEFT {
			depth = depth - 1
			if depth == 0 {
				break
			}
		}
		place = place - 1
	}
	if place <= opener+1 {
		return false
	}
	source := token.Source(subject.Source.Values)
	called := ast.Syntax_Token_At(tree, ast.Token_Index(place-1))
	if string(token.Text(source, called)) != "len" {
		return false
	}
	if ast.Syntax_Token_At(tree, ast.Token_Index(place-2)).Kind != token.KIND_COLON {
		return false
	}
	one := ast.Syntax_Token_At(tree, ast.Token_Index(opener-1))
	two := ast.Syntax_Token_At(tree, ast.Token_Index(place+1))
	if one.Kind != two.Kind {
		return false
	}
	if string(token.Text(source, one)) != string(token.Text(source, two)) {
		return false
	}
	return Boolean(place+2 == last)
}

// Reports whether the slice the walk stands on holds one colon alone, counting no colon a bracket
// of its own holds.
func holds_one_colon(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_one_colon.yes") }()
	Printer_Handle_Invariants(subject, "holds_one_colon.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_one_colon.tree")
	closer := int(closer_position(subject, tree))
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	opener := int(ast.Syntax_Node_At(tree, index).Token)
	depth := 0
	colons := 0
	for place := opener + 1; place < closer; place = place + 1 {
		kind := ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind
		step := bracket_step(token.Scanned_Kind(kind))
		if step == BRACKET_OPENS {
			depth = depth + 1
		}
		if step == BRACKET_CLOSES {
			depth = depth - 1
		}
		if depth != 0 {
			continue
		}
		if kind == token.KIND_COLON {
			colons = colons + 1
		}
	}
	// A slice parts the bounds it states with a colon, thus a read over the tokens its
	// brackets hold meets one colon at least.
	if closer > opener {
		aver.Always(colons > 0, "A slice parts the bounds it states with a colon.")
	}
	return Boolean(colons == 1)
}

// Reports whether the slice still owes the bound the walk stands on. A slice writes the value it
// reads and every bound it states, and it drops a closing bound the brackets already state, thus
// the bounds it owes close at the count the print read.
func owes_bound(stood Slice_Parts, count Slice_Parts) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "owes_bound.yes") }()
	Slice_Parts_Invariants(stood, "owes_bound.stood")
	Slice_Parts_Invariants(count, "owes_bound.count")
	return Boolean(stood < count)
}

// Writes one colon of a slice and the spaces the canonical form states on either side of it.
func emit_colon(subject Printer_Handle, behind Boolean, ahead Boolean) {
	Printer_Handle_Invariants(subject, "emit_colon.subject")
	Boolean_Invariants(behind, "emit_colon.behind")
	Boolean_Invariants(ahead, "emit_colon.ahead")
	// The colon spends one column of its own and one for each space it stands between.
	stood := take_line_width(subject.Counts.Values)
	held := 1
	if bool(behind) {
		held = held + 1
	}
	if bool(ahead) {
		held = held + 1
	}
	if bool(behind) {
		emit_space(subject)
	}
	emit_byte(subject, ':')
	if bool(ahead) {
		emit_space(subject)
	}
	// A colon that carries a value behind it stands apart from that value, and one that
	// carries none closes the line on the mark itself.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		closed := subject.Form.Values[int(written_load(subject.Counts.Values))-1]
		if bool(ahead) {
			aver.Always(closed == ' ',
				"A colon that carries a value behind it closes on a space.")
		}
		if !bool(ahead) {
			aver.Always(closed == ':',
				"A colon that carries no value closes on itself.")
		}
		if int(take_line_width(subject.Counts.Values)) < WIDTH_MAXIMUM {
			aver.Always(int(take_line_width(subject.Counts.Values))-int(stood) == held,
				"A colon spends one column and the spaces it stands between.")
		}
	}
}

// Reports whether the note the walk stands on names a token past the bracket that closes the
// form the print writes. A note behind that bracket stands on the line the bracket closes, thus
// the flush of the line inside the brackets leaves it held.
func stands_past(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, closer Closer_Position,
) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_past.yes") }()
	Printer_Handle_Invariants(subject, "stands_past.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_past.tree")
	Closer_Position_Invariants(closer, "stands_past.closer")
	if closer == CLOSER_POSITION_NONE {
		return false
	}
	node := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	return Boolean(Closer_Position(node.Token) > closer)
}

// Reads the position of the bracket that closes the form the part the walk stands on stands in.
// The parts of one form stand as the children of the form or as the siblings of one another,
// thus the read runs the tokens behind the part rather than the chain of children.
func part_closer(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (closer Closer_Position) {
	defer func() { Closer_Position_Invariants(closer, "part_closer.closer") }()
	Printer_Handle_Invariants(subject, "part_closer.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "part_closer.tree")
	// A close that holds no note stands on no bracket the read answers for.
	if comment_count(subject) == COMMENT_COUNT_MINIMUM {
		return CLOSER_POSITION_NONE
	}
	depth := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[depth]
	// A note the author left behind the final part runs the read back to the bracket.
	if bool(trivia_kind(subject, tree)) {
		stood := ast.Index(ast.INDEX_ABSENT)
		parent := ast.Index(ast.Syntax_Node_At(tree, held).Parent)
		child := ast.Index(ast.Syntax_Node_At(tree, parent).First_Child)
		for range ast.NODE_COUNT_MAXIMUM {
			if child == ast.INDEX_ABSENT {
				break
			}
			if child == held {
				break
			}
			subject.Nodes.Values[depth] = child
			if !bool(trivia_kind(subject, tree)) {
				stood = child
			}
			child = ast.Index(ast.Syntax_Node_At(tree, child).Next)
		}
		subject.Nodes.Values[depth] = stood
		// A note ahead of every part of the form it hangs under stands past the bracket.
		if stood == ast.INDEX_ABSENT {
			subject.Nodes.Values[depth] = held
			behind := tree.Tokens.Values
			note := int(ast.Syntax_Node_At(tree, held).Token)
			for position := note - 1; position > 0; position-- {
				kind := token.Scanned_Kind(behind[position].Kind)
				if bool(closes_bracket(kind)) {
					return Closer_Position(position)
				}
			}
			return CLOSER_POSITION_NONE
		}
	}
	enter_rightmost(subject, tree)
	// The read steps into the form and never out of it.
	aver.Always(depth_load(subject.Counts.Values) >= depth,
		"A read of the bracket one form closes on stands below the slot it opened.")
	final := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	level := 0
	run := tree.Tokens.Values
	for position := int(final.Token) + 1; position < ast.TOKEN_COUNT_MAXIMUM; position++ {
		kind := token.Scanned_Kind(run[position].Kind)
		if run[position].Kind == token.KIND_END_OF_FILE {
			return CLOSER_POSITION_NONE
		}
		if bool(opens_bracket(kind)) {
			level = level + 1
		}
		if !bool(closes_bracket(kind)) {
			continue
		}
		if level == 0 {
			return Closer_Position(position)
		}
		level = level - 1
	}
	return CLOSER_POSITION_NONE
}

// CLOSER_POSITION_NONE names a flush that holds no note back, which every close of a line states.
const CLOSER_POSITION_NONE = 0

// RUN_CLOSE_ENDED names a run the block that holds it closed.
const RUN_CLOSE_ENDED = 0

// RUN_CLOSE_TRIVIA names a run a note of its own node or an empty line closed.
const RUN_CLOSE_TRIVIA = 1

// RUN_CLOSE_NOTELESS names a run a statement carrying no note closed.
const RUN_CLOSE_NOTELESS = 2

// RUN_CLOSE_SPANS names a run a statement running over lines closed.
const RUN_CLOSE_SPANS = 3

// Run_Close names what closed the run of noted statements one read walks.
type Run_Close uint8

// Run_Close_Invariants states every close one run of noted statements takes.
func Run_Close_Invariants(value Run_Close, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_4_Uint8(uint8(value), RUN_CLOSE_ENDED, RUN_CLOSE_TRIVIA,
			RUN_CLOSE_NOTELESS, RUN_CLOSE_SPANS).
		Ensure()
}

// Reads the column the notes of the run of statements that opens where the walk stands write at.
// Reads the column the notes of the run that opens at the statement the walk stands on write at.
// A statement that carries no note closes the run, thus a run holds the lines a reader reads as
// one table. The measure writes each statement past the bytes the print already wrote, thus it
// costs no storage of its own.
func take_statement_column(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "take_statement_column.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_statement_column.tree")
	stood := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[stood]
	widest := Count(0)
	open := Boolean(true)
	closed := Run_Close(RUN_CLOSE_ENDED)
	for bool(open) {
		// A second note on the line of a statement stands as a node of its own, and an
		// empty line stands as one too, thus either closes the run where it stands.
		if bool(trivia_kind(subject, tree)) {
			closed = RUN_CLOSE_TRIVIA
			break
		}
		if !bool(trails_comment(subject, tree)) {
			closed = RUN_CLOSE_NOTELESS
			break
		}
		// A statement that runs over lines of its own carries its note wherever that
		// note fell, thus it stands in no column with the statements around it.
		if bool(spans_form(subject, tree)) {
			closed = RUN_CLOSE_SPANS
			break
		}
		measured := measure_statement(subject, tree)
		// A statement the print breaks carries its note where that note falls, thus the
		// run closes at it and its width states no column.
		if bool(measured.Spans) {
			closed = RUN_CLOSE_SPANS
			break
		}
		// Every line of a run holds the tab of its block and a byte of the statement it
		// states, thus the run opens its column past those tabs.
		aver.Always(int(measured.Width) >= STATEMENT_WIDTH_MINIMUM,
			"A statement of one run states the width its line holds.")
		widest = max(widest, Count(measured.Width))
		// An empty line behind the statement closes the run it stands in.
		if bool(trails_blank(subject, tree)) {
			closed = RUN_CLOSE_TRIVIA
			break
		}
		open = advance(subject, tree)
		// A note the author left on the line of a statement stands as a node of its own
		// where the statement holds none of it, thus the walk steps over that note alone
		// and reads the statement behind it.
		if bool(open) {
			if bool(trails_line(subject, tree)) {
				open = advance(subject, tree)
			}
		}

	}
	subject.Nodes.Values[depth_load(subject.Counts.Values)] = held
	column_store(subject.Counts.Values, Note_Column(min(int(widest), WIDTH_MAXIMUM)))
	// A run closes at a note of its own node, at an empty line, at a statement carrying no
	// note, at one that runs over lines, or at the end of the block that holds it.
	Run_Close_Invariants(closed, "take_statement_column.closed")
	// The read steps along the run with a step that holds the slot it stood on, thus it
	// hands back the walk it opened.
	aver.Always(depth_load(subject.Counts.Values) == stood,
		"A read of one run of statements holds the slot it opened.")
	// A run opens its column behind the widest line it holds, and every line of a run of
	// statements holds the tab of its block and a byte of the statement it states.
	if column_load(subject.Counts.Values) != 0 {
		aver.Always(int(column_load(subject.Counts.Values)) >= STATEMENT_WIDTH_MINIMUM,
			"A run of notes on statements opens past the tabs of its lines.")
	}
}

// Reports whether the field the walk stands on runs over lines of its own. The tree hangs an
// empty line and a note behind the field they follow, thus the read measures the names and the
// type the field binds and leaves the trivia standing behind them out of the span.
func field_spans(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "field_spans.yes") }()
	Printer_Handle_Invariants(subject, "field_spans.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "field_spans.tree")
	depth := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[depth]
	opening := ast.Syntax_Token_At(tree, ast.Syntax_Node_At(tree, held).Token)
	last := ast.Index(ast.INDEX_ABSENT)
	if bool(descend_quiet(subject, tree)) {
		open := Boolean(true)
		for bool(open) {
			if !bool(trivia_kind(subject, tree)) {
				last = subject.Nodes.Values[depth_load(subject.Counts.Values)]
			}
			open = advance_quiet(subject, tree)
		}
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	if last == ast.INDEX_ABSENT {
		return false
	}
	subject.Nodes.Values[depth] = last
	enter_rightmost(subject, tree)
	// The read steps into the field and never out of it.
	aver.Always(depth_load(subject.Counts.Values) >= depth,
		"A read of the span one field holds stands below the slot it opened.")
	final := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	tail := ast.Syntax_Token_At(tree, final.Token)
	// The tokens of one field stand in the order the source spells them, thus the token the
	// field closes on ends behind the byte the field opens on.
	aver.Always(int(tail.Offset)+int(tail.Size) >= int(opening.Offset),
		"A field closes on a token ending behind the byte it opens on.")

	// A line feed inside the span states a field the author wrote over lines of its own.
	for place := int(opening.Offset); place < int(tail.Offset)+int(tail.Size); place++ {
		if subject.Source.Values[place] == '\n' {
			return true
		}
	}
	return false
}

// Reports whether one token opens a bracketed form.
func opens_bracket(sign token.Scanned_Kind) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_bracket.yes") }()
	token.Scanned_Kind_Invariants(sign, "opens_bracket.sign")
	kind := token.Kind(sign)
	if kind == token.KIND_PARENTHESIS_LEFT {
		return true
	}
	if kind == token.KIND_BRACKET_LEFT {
		return true
	}
	return Boolean(kind == token.KIND_BRACE_LEFT)
}

// Reports whether the empty line the walk stands on stands inside the head of a clause, which is
// the run of tokens between the word the clause opens with and the brace of its body. An empty
// line inside a head states nothing, thus the print drops it wherever the author left it.
func stands_inside_clause(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_inside_clause.yes") }()
	Printer_Handle_Invariants(subject, "stands_inside_clause.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_inside_clause.tree")
	// The read answers for the empty lines one clause holds, thus the walk stands on one.
	aver.Always(node_kind(subject, tree) == ast.NODE_BLANK,
		"A read for a line inside one clause stands on an empty line.")
	held := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	parent := ast.Syntax_Node_At(tree, ast.Index(held.Parent))
	switch parent.Kind {
	case ast.NODE_IF, ast.NODE_FOR, ast.NODE_RANGE:
	case ast.NODE_SWITCH, ast.NODE_TYPE_SWITCH, ast.NODE_SELECT:
	default:
		return false
	}
	child := ast.Index(parent.First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return false
		}
		one := ast.Syntax_Node_At(tree, child)
		// The body of the clause opens at its brace, thus every token ahead of that
		// brace stands in the head the clause states.
		if one.Kind == ast.NODE_BLOCK {
			aver.Always(one.Token > parent.Token,
				"A clause opens its body behind the word the clause states.")
			return Boolean(held.Token < one.Token)
		}
		child = ast.Index(one.Next)
	}
	return false
}

// Reports whether one token closes a bracketed form.
func closes_bracket(sign token.Scanned_Kind) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_bracket.yes") }()
	token.Scanned_Kind_Invariants(sign, "closes_bracket.sign")
	kind := token.Kind(sign)
	if kind == token.KIND_PARENTHESIS_RIGHT {
		return true
	}
	if kind == token.KIND_BRACKET_RIGHT {
		return true
	}
	return Boolean(kind == token.KIND_BRACE_RIGHT)
}

// Reports whether the two forms the top two slots of the walk stack hold state one type, token for
// token. The walk reads both forms through their links and never descends into them, thus the
// stack names them without the print standing on either.
func same_form(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "same_form.yes") }()
	Printer_Handle_Invariants(subject, "same_form.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "same_form.tree")
	depth := depth_load(subject.Counts.Values)
	aver.Always(depth > 0, "A read of two forms stands under the form it compares against.")
	root, two := subject.Nodes.Values[depth-1], subject.Nodes.Values[depth]
	one := root
	// The read compares one form against the form around it, thus it opens on two nodes.
	aver.Always(root != two, "A read of two forms opens on two nodes of the tree.")
	source := token.Source(subject.Source.Values)
	for range ast.NODE_COUNT_MAXIMUM {
		left := ast.Syntax_Node_At(tree, one)
		right := ast.Syntax_Node_At(tree, two)
		if left.Kind != right.Kind {
			return false
		}
		if string(token.Text(source, ast.Syntax_Token_At(tree, left.Token))) !=
			string(token.Text(source, ast.Syntax_Token_At(tree, right.Token))) {
			return false
		}
		if Boolean(left.First_Child == 0) != Boolean(right.First_Child == 0) {
			return false
		}
		if ast.Index(left.First_Child) != ast.INDEX_ABSENT {
			one, two = ast.Index(left.First_Child), ast.Index(right.First_Child)
			continue
		}
		// A form with no child steps to the sibling behind it, past the trivia standing
		// between, or climbs to the first ancestor that holds one.
		next_one, next_two := ast.INDEX_ABSENT, ast.INDEX_ABSENT
		for one != root {
			held := [FORM_SIDE_COUNT]ast.Index{
				ast.Index(ast.Syntax_Node_At(tree, one).Next),
				ast.Index(ast.Syntax_Node_At(tree, two).Next),
			}
			for side := range FORM_SIDE_COUNT {
				for range ast.NODE_COUNT_MAXIMUM {
					if held[side] == ast.INDEX_ABSENT {
						break
					}
					node := ast.Syntax_Node_At(tree, held[side])
					if node.Kind != ast.NODE_COMMENT {
						if node.Kind != ast.NODE_BLANK {
							break
						}
					}
					held[side] = ast.Index(node.Next)
				}
			}
			next_one, next_two = held[0], held[1]
			if next_one != ast.INDEX_ABSENT {
				break
			}
			if next_two != ast.INDEX_ABSENT {
				return false
			}
			one = ast.Index(ast.Syntax_Node_At(tree, one).Parent)
			two = ast.Index(ast.Syntax_Node_At(tree, two).Parent)
		}
		if one == root {
			return true
		}
		if next_two == ast.INDEX_ABSENT {
			return false
		}
		one, two = next_one, next_two
	}
	return false
}

// Reports whether the form the walk stands on holds a part that closes the line it stands on: a
// declaration, or a clause whose own body closes that line.
func holds_closing_part(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_closing_part.yes") }()
	Printer_Handle_Invariants(subject, "holds_closing_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_closing_part.tree")
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	found := Boolean(false)
	open := descend_quiet(subject, tree)
	for bool(open) {
		switch node_kind(subject, tree) {
		case ast.NODE_DECLARATION_STATEMENT, ast.NODE_IF, ast.NODE_FOR, ast.NODE_RANGE:
			found = true
		case ast.NODE_SWITCH, ast.NODE_TYPE_SWITCH, ast.NODE_SELECT, ast.NODE_LABEL:
			found = true
		}
		if bool(found) {
			break
		}
		open = advance_quiet(subject, tree)
	}
	// The read steps into the parts the form holds and never out of the form, thus it stands
	// at the slot it opened or below it, and the walk stands back once it read them.
	aver.Always(depth_load(subject.Counts.Values) >= depth,
		"A read of the parts one form holds stands below the slot it opened.")
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	return found
}

// Reads the position of the bracket that closes the form the walk stands inside, which the parts
// of that form stand one level under.
func form_closer(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (closer Closer_Position) {
	defer func() { Closer_Position_Invariants(closer, "form_closer.closer") }()
	Printer_Handle_Invariants(subject, "form_closer.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "form_closer.tree")
	// A close that holds no note writes every note it holds wherever the bracket stands,
	// thus the read of that bracket stands on the closes that hold one.
	if comment_count(subject) == COMMENT_COUNT_MINIMUM {
		return CLOSER_POSITION_NONE
	}
	depth := depth_load(subject.Counts.Values)
	if depth == 0 {
		return CLOSER_POSITION_NONE
	}
	depth_store(subject.Counts.Values, depth-1)
	held := closer_position(subject, tree)
	depth_store(subject.Counts.Values, depth)
	return held
}

// Reports whether the declaration one slot holds is a function whose name closes with the tail
// every function that states the invariants of a type closes with.
func names_tail(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_tail.yes") }()
	Printer_Handle_Invariants(subject, "names_tail.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "names_tail.tree")
	held := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	if held.Kind != ast.NODE_FUNCTION {
		return false
	}
	// A note or an empty line the author left inside the head names nothing of the
	// declaration, thus the read steps over the trivia ahead of the name.
	name := ast.Index(held.First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if name == ast.INDEX_ABSENT {
			return false
		}
		kind := ast.Syntax_Node_At(tree, name).Kind
		if kind != ast.NODE_COMMENT {
			if kind != ast.NODE_BLANK {
				break
			}
		}
		name = ast.Index(ast.Syntax_Node_At(tree, name).Next)
	}
	named := ast.Syntax_Node_At(tree, name)
	// The read steps along the chain the declaration opens, thus the name it stopped on
	// hangs under that declaration.
	stood := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	aver.Always(ast.Index(named.Parent) == stood,
		"A read of the name one function binds stands on a part of it.")
	if named.Kind != ast.NODE_IDENTIFIER {
		return false
	}
	source := token.Source(subject.Source.Values)
	text := token.Text(source, ast.Syntax_Token_At(tree, named.Token))
	if len(text) <= len(INVARIANTS_TAIL) {
		return false
	}
	return Boolean(string(text[len(text)-len(INVARIANTS_TAIL):]) == INVARIANTS_TAIL)
}

// Reports whether the type the walk stands on is the first type of its name the file declares.
// One function of invariants stands under one type, thus a second type of one name takes none of
// it and a file that declares one name twice writes each function it holds once.
func first_named_type(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "first_named_type.yes") }()
	Printer_Handle_Invariants(subject, "first_named_type.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "first_named_type.tree")
	stood := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	held := ast.Syntax_Node_At(tree, stood)
	// One function of invariants stands under one type, thus the read stands on a type.
	aver.Always(held.Kind == ast.NODE_TYPE,
		"A read for the first type of one name stands on a type.")
	if ast.Index(held.First_Child) == ast.INDEX_ABSENT {
		return true
	}
	source := token.Source(subject.Source.Values)
	// A note or an empty line inside the head names nothing of the type, thus the read
	// steps over the trivia standing ahead of the name.
	child_name := ast.Index(held.First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		kind := ast.Syntax_Node_At(tree, child_name).Kind
		if kind != ast.NODE_COMMENT {
			if kind != ast.NODE_BLANK {
				break
			}
		}
		child_name = ast.Index(ast.Syntax_Node_At(tree, child_name).Next)
		if child_name == ast.INDEX_ABSENT {
			return true
		}
	}
	named := ast.Syntax_Node_At(tree, child_name)
	name := string(token.Text(source, ast.Syntax_Token_At(tree, named.Token)))
	child := ast.Index(ast.Syntax_Node_At(tree, ast.Index(held.Parent)).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return true
		}
		if child == stood {
			return true
		}
		one := ast.Syntax_Node_At(tree, child)
		child = ast.Index(one.Next)
		if one.Kind != ast.NODE_TYPE {
			continue
		}
		child_other := ast.Index(one.First_Child)
		for range ast.NODE_COUNT_MAXIMUM {
			if child_other == ast.INDEX_ABSENT {
				break
			}
			kind := ast.Syntax_Node_At(tree, child_other).Kind
			if kind != ast.NODE_COMMENT {
				if kind != ast.NODE_BLANK {
					break
				}
			}
			child_other = ast.Index(ast.Syntax_Node_At(tree, child_other).Next)
		}
		if child_other == ast.INDEX_ABSENT {
			continue
		}
		other := ast.Syntax_Node_At(tree, child_other)
		if string(token.Text(source, ast.Syntax_Token_At(tree, other.Token))) == name {
			return false
		}
	}
	return true
}

// NOTE_MARK_SIZE is how many bytes the mark one note opens with spends: the two slashes of a note
// the author closed at the line, and the slash and the star of one the author closed by hand.
const NOTE_MARK_SIZE = 2

// Reports whether the function the top slot of the walk stack holds states the invariants of the
// type the slot beneath it holds, which the name of that function says: the name of the type and
// the tail every such function closes with. A declaration that binds no name states no pair.
func names_pair(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_pair.yes") }()
	Printer_Handle_Invariants(subject, "names_pair.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "names_pair.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the type one pair of names states writes no byte of its own.")
	}()
	depth := depth_load(subject.Counts.Values)
	typed := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth-1])
	function := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth])
	if ast.Index(typed.First_Child) == ast.INDEX_ABSENT {
		return false
	}
	if ast.Index(function.First_Child) == ast.INDEX_ABSENT {
		return false
	}
	// The read compares one type against one function, thus the two slots of the walk name
	// two nodes of the tree.
	aver.Always(subject.Nodes.Values[depth-1] != subject.Nodes.Values[depth],
		"A read of one pair names two nodes of the tree.")
	// A note or an empty line inside either head names nothing of the declaration it stands
	// in, thus each read steps over the trivia ahead of the name.
	typed_child, function_child := ast.Index(typed.First_Child), ast.Index(function.First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		kind := ast.Syntax_Node_At(tree, typed_child).Kind
		if kind != ast.NODE_COMMENT {
			if kind != ast.NODE_BLANK {
				break
			}
		}
		typed_child = ast.Index(ast.Syntax_Node_At(tree, typed_child).Next)
		if typed_child == ast.INDEX_ABSENT {
			return false
		}
	}
	for range ast.NODE_COUNT_MAXIMUM {
		kind := ast.Syntax_Node_At(tree, function_child).Kind
		if kind != ast.NODE_COMMENT {
			if kind != ast.NODE_BLANK {
				break
			}
		}
		function_child = ast.Index(ast.Syntax_Node_At(tree, function_child).Next)
		if function_child == ast.INDEX_ABSENT {
			return false
		}
	}
	type_name := ast.Syntax_Node_At(tree, typed_child)
	function_name := ast.Syntax_Node_At(tree, function_child)
	if type_name.Kind != ast.NODE_IDENTIFIER {
		return false
	}
	if function_name.Kind != ast.NODE_IDENTIFIER {
		return false
	}
	source := token.Source(subject.Source.Values)
	named := token.Text(source, ast.Syntax_Token_At(tree, type_name.Token))
	held := token.Text(source, ast.Syntax_Token_At(tree, function_name.Token))
	if len(held) != len(named)+len(INVARIANTS_TAIL) {
		return false
	}
	for index := range len(named) {
		if held[index] != named[index] {
			return false
		}
	}
	return Boolean(string(held[len(named):]) == INVARIANTS_TAIL)
}

// BASE_MARK_SIZE is how many bytes the mark that names the base of one number spends: the zero it
// opens on and the letter behind it.
const BASE_MARK_SIZE = 2

// CHECK_TOKEN_SPAN is how many tokens stand behind the word a simple error check opens with: the
// name it reads, the sign it compares with, the value it compares against, and the brace of its
// body.
const CHECK_TOKEN_SPAN = 4

// Reads that the columns the print stands at state the line the form holds. The print counts the
// columns of each byte as it writes it, thus a cursor that stood back where a measure left it
// would state a line the bytes never spell.
func reads_columns(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "reads_columns.subject")
	if !bool(writes_canonical(subject.Flags.Values)) {
		return
	}
	form := subject.Form.Values
	written := int(written_load(subject.Counts.Values))
	// A line wider than the read admits costs a read of its own for every note it carries,
	// thus the read stands on the lines a reader reads and leaves the widest ones alone.
	place := written
	opened := false
	for range COLUMNS_READ_MAXIMUM {
		if place == 0 {
			opened = true
			break
		}
		if form[place-1] == '\n' {
			opened = true
			break
		}
		place = place - 1
	}
	if !opened {
		return
	}
	// A tab stands in the columns a reader sees, and a byte that continues a character
	// stands in none of its own.
	columns := 0
	for index := place; index < written; index++ {
		if form[index] == '\t' {
			columns = columns + TAB_WIDTH
			continue
		}
		if form[index]&CHARACTER_TAIL_MASK == CHARACTER_TAIL_MARK {
			continue
		}
		columns = columns + 1
	}
	aver.Always(int(take_line_width(subject.Counts.Values)) == min(columns, WIDTH_MAXIMUM),
		"The columns the print stands at state the line the form holds.")
}

// COLUMNS_READ_MAXIMUM is how many bytes the read of the columns one line holds steps back over.
// A line stands inside the widest column a reader reads, and a line wider than that holds a form
// no break of the print shortens, thus the read stands on the lines a reader reads.
const COLUMNS_READ_MAXIMUM = COLUMN_WIDTH_MAXIMUM * 2

// Reports whether the part the print stands at opens the column of a broken form, which every
// part of that form opens at however deep the part before it left the print.
func open_column(here Boolean, opened Boolean) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "open_column.yes") }()
	Boolean_Invariants(here, "open_column.here")
	Boolean_Invariants(opened, "open_column.opened")
	if bool(here) {
		return true
	}
	return opened
}

// Writes one statement of a case body and opens the body where it stands first.
func print_case_body(subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean) {
	Printer_Handle_Invariants(subject, "print_case_body.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_case_body.tree")
	Boolean_Invariants(open, "print_case_body.open")
	if !bool(open) {
		written := written_load(subject.Counts.Values)
		emit_byte(subject, ':')
		// A note the author left on the line the case opens keeps that line.
		stood := open_brace_note(subject, tree)
		close_line(subject, tree)
		// The case closes the values it states on a colon and opens its body on the
		// line behind it, thus the opening writes bytes of its own.
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(written_load(subject.Counts.Values) > written,
				"A case opens its body behind the colon that closes its values.")
		}
		indent_deepen(subject, indent_load(subject.Counts.Values))
		if !bool(stood) {
			return
		}
	}
	print_statement(subject, tree)
}

// Reads the place the part behind the one the walk stands on stands at. Every part behind the
// first stands the same way, thus the walk holds all of them as one place.
// Next_Part_Place names where the part behind one part stands: the part a form opens at or one
// standing behind it. No part behind another opens the head the form reads its parts from.
type Next_Part_Place uint8

// Next_Part_Place_Invariants states both places a part behind another stands at.
func Next_Part_Place_Invariants(value Next_Part_Place, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), uint8(PART_FIRST), uint8(PART_BEHIND)).
		Ensure()
}

// Reads the place the part behind the one the walk stands on stands at.
func next_part_place(place Part_Place) (next Next_Part_Place) {
	defer func() { Next_Part_Place_Invariants(next, "next_part_place.next") }()
	Part_Place_Invariants(place, "next_part_place.place")
	if place == PART_HEAD {
		return PART_FIRST
	}
	return PART_BEHIND
}

// Holds the note the author left on the line one field closes, so that the close of that line
// writes it behind the field rather than on a line of its own. It reports whether a node stands
// behind the notes it held.
func take_trailing_note(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean,
) (next Boolean) {
	defer func() { Boolean_Invariants(next, "take_trailing_note.next") }()
	Printer_Handle_Invariants(subject, "take_trailing_note.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_trailing_note.tree")
	Boolean_Invariants(open, "take_trailing_note.open")
	stood := written_load(subject.Counts.Values)
	stand := open
	for bool(stand) {
		if node_kind(subject, tree) != ast.NODE_COMMENT {
			break
		}
		if bool(stands_alone(subject, tree)) {
			break
		}
		print_trivia(subject, tree)
		stand = advance(subject, tree)
	}
	// A note the print holds stands on the line the field closes, thus the taking of it
	// writes no byte of its own and the close of that line writes it.
	aver.Always(written_load(subject.Counts.Values) == stood,
		"A take of the note one field carries writes no byte of its own.")
	return stand
}

// Holds the note the author left on the line one statement closes, which stands as the sibling of
// that statement where the statement holds no part of its own. The walk stands on the note the
// print took, thus the block steps past it as it steps past the statement it wrote.
func take_statement_note(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "take_statement_note.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_statement_note.tree")
	depth := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[depth]
	stood := written_load(subject.Counts.Values)
	for range COMMENT_COUNT_MAXIMUM {
		if !bool(advance(subject, tree)) {
			break
		}
		if !bool(trails_line(subject, tree)) {
			break
		}
		// A note standing past the bracket that closes the form around the statement
		// stands on the line that bracket closes, thus the statement takes none of it.
		if bool(trails_close(subject, tree)) {
			break
		}
		print_trivia(subject, tree)
		held = subject.Nodes.Values[depth]
	}
	subject.Nodes.Values[depth] = held
	// A note the print holds stands on the line the statement closes, thus the taking of it
	// writes no byte of its own and the close of that line writes it.
	aver.Always(written_load(subject.Counts.Values) == stood,
		"A take of the note one statement carries writes no byte of its own.")
}

// FIELD_CLOSE_NONE names a field that closes the run of fields it stands in at nothing.
const FIELD_CLOSE_NONE = 0

// FIELD_CLOSE_LINE names a run an empty line inside the field closed.
const FIELD_CLOSE_LINE = 1

// Field_Close names what closes the run of fields one field stands in.
type Field_Close uint8

// Field_Close_Invariants states both closes one run of fields takes.
func Field_Close_Invariants(value Field_Close, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Enum_Uint8(uint8(value), FIELD_CLOSE_NONE, FIELD_CLOSE_LINE).
		Ensure()
}

// Writes the names one field binds, parted by the comma and the space that stand between two of
// them. A note the author left between the names
// stands on the line the field closes, thus the walk holds it and reads the name behind it. The
// walk stands on the part behind the names, or on the last name the field binds where it states
// no part at all.
func print_field_names(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_field_names.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_field_names.tree")
	stood := written_load(subject.Counts.Values)
	named := Field_Named(false)
	open := Boolean(true)
	for bool(open) {
		if bool(trivia_kind(subject, tree)) {
			print_trivia(subject, tree)
			open = advance(subject, tree)
			continue
		}
		if node_kind(subject, tree) != ast.NODE_FIELD_NAME {
			break
		}
		open_field_name(subject, named)
		subject.Counts.Values[COUNT_FIELD_NAME] += 1
		emit_token(subject, tree)
		named = true
		open = advance(subject, tree)
	}
	// A field that binds names writes the first of them wherever the print reads one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A field that binds names writes the names it binds.")
	}
}

// Opens one block, reading whether it closes up against its braces, and hands back the tightness
// of the form around it. A body of one statement alone, a body of none, and the body one function
// states each close up; every other block keeps the lines the author wrote.
func open_block_tight(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (held Boolean) {
	defer func() { Boolean_Invariants(held, "open_block_tight.held") }()
	Printer_Handle_Invariants(subject, "open_block_tight.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_block_tight.tree")
	tight := flag_load(subject.Flags.Values, FLAG_BODY) || holds_one_part(subject, tree)
	tight = tight || holds_no_part(subject, tree)
	flag_store(subject.Flags.Values, FLAG_BODY, false)
	return hold_tight(subject, tight)
}

// Reads the trivia the walk stands ahead of into the line the print stands on. A note the author
// left on that line keeps it, thus the line closes holding everything the author wrote on it and
// a note of a line of its own still opens one.
func take_trivia(subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean) {
	Printer_Handle_Invariants(subject, "take_trivia.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_trivia.tree")
	Boolean_Invariants(open, "take_trivia.open")
	stand := open
	for bool(stand) {
		print_trivia(subject, tree)
		stand = advance(subject, tree)
	}
}

// Holds every note the form the walk stands on carries, and leaves the walk where it opened. A
// bound the print drops states nothing of its own, thus the notes the author left inside it
// still stand on the line that form closes.
func take_form_notes(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "take_form_notes.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_form_notes.tree")
	stood := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[stood]
	written := written_load(subject.Counts.Values)
	noted := comment_count(subject)
	for range ast.NODE_COUNT_MAXIMUM {
		if bool(trivia_kind(subject, tree)) {
			print_trivia(subject, tree)
		}
		if bool(descend(subject, tree)) {
			continue
		}
		// The walk climbs out of the form it stepped into until a part stands behind
		// it, and a walk back at the depth it opened at read every part.
		moved := Boolean(false)
		for range DEPTH_MAXIMUM {
			if depth_load(subject.Counts.Values) == stood {
				break
			}
			if bool(advance(subject, tree)) {
				moved = true
				break
			}
			ascend(subject)
		}
		if !bool(moved) {
			break
		}
	}
	depth_store(subject.Counts.Values, stood)
	subject.Nodes.Values[stood] = held
	// A note the print holds stands on the line the form closes, thus the taking of it
	// writes no byte of its own.
	aver.Always(written_load(subject.Counts.Values) == written,
		"A take of the notes one form carries writes no byte of its own.")
	// The take holds each note it met and writes none of them, thus it carries every note
	// it stood with and the ones the form holds behind them.
	aver.Always(comment_count(subject) >= noted,
		"A take of the notes one form carries holds the notes it stood with.")
}

// Reports whether the byte the print wrote last is a space or a tab, which no line of the
// canonical form closes on.
func stands_after_space(subject Printer_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_after_space.yes") }()
	Printer_Handle_Invariants(subject, "stands_after_space.subject")
	written := int(written_load(subject.Counts.Values))
	if written == 0 {
		return false
	}
	form := subject.Form.Values
	held := form[written-1]
	if held == ' ' {
		return true
	}
	return Boolean(held == '\t')
}

// Reports whether the note the walk stands on closes at the line it stands on, which is a note
// the author opened on two slashes. Such a note swallows every byte behind it on its line, thus
// no note stands behind one on the line it closes.
func closes_at_line(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_at_line.yes") }()
	Printer_Handle_Invariants(subject, "closes_at_line.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_at_line.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	if len(text) < NOTE_MARK_SIZE {
		return false
	}
	return Boolean(text[1] == '/')
}

// Reports whether the token before the node the walk stands on binds the names of a run to their
// values, by either sign the source states for it.
func opens_bound(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_bound.yes") }()
	Printer_Handle_Invariants(subject, "opens_bound.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_bound.tree")
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	// The read steps into the form and never out of it, thus it stands at the slot it
	// opened or below it, and the walk stands back once it read the token.
	aver.Always(depth_load(subject.Counts.Values) >= held_depth,
		"A read of the byte one bound opens on stands below the slot it opened.")
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	found := Count(ast.Syntax_Node_At(tree, leftmost).Token)
	position := found
	if position == POSITION_MINIMUM {
		return false
	}
	// A note the author left between the sign and the value states nothing of the binding,
	// thus the read steps over every note it meets.
	place := int(position) - 1
	for place > POSITION_MINIMUM {
		if ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind != token.KIND_COMMENT {
			break
		}
		place = place - 1
	}
	kind := ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind
	if kind == token.KIND_ASSIGN {
		return true
	}
	return kind == token.KIND_DEFINE
}

// States where the brace that closes one body stands: on a line of its own, at the tabs the form
// that holds the body opened at.
func closes_body(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "closes_body.subject")
	if !bool(writes_canonical(subject.Flags.Values)) {
		return
	}
	aver.Always(int(take_line_width(subject.Counts.Values)) ==
		int(indent_load(subject.Counts.Values))*INDENT_WIDTH,
		"A body closes its brace on a line of its own.")
}

// Reports whether the print stands at the opening of a line, which is where the form it wrote
// last closed a line or where it wrote nothing at all.
func opens_form_line(subject Printer_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_form_line.yes") }()
	Printer_Handle_Invariants(subject, "opens_form_line.subject")
	written := int(written_load(subject.Counts.Values))
	if written == 0 {
		return true
	}
	return Boolean(subject.Form.Values[written-1] == '\n')
}

// Reports whether the trivia the walk stands on closes the declaration it stands in. A note the
// author left between the names and the sign that binds them states nothing of the run, thus the
// print holds it and the declaration runs on to the values behind it; a note behind everything
// the declaration states closes it.
func names_trivia(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle,
) (closes Boolean, named Value_Named) {
	defer func() {
		Boolean_Invariants(closes, "names_trivia.closes")
		Value_Named_Invariants(named, "names_trivia.named")
	}()
	Printer_Handle_Invariants(subject, "names_trivia.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "names_trivia.tree")
	if !bool(trivia_kind(subject, tree)) {
		return false, false
	}
	depth := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[depth]
	parted := Boolean(false)
	for range ast.NODE_COUNT_MAXIMUM {
		if !bool(advance_quiet(subject, tree)) {
			break
		}
		if !bool(trivia_kind(subject, tree)) {
			parted = true
			break
		}
	}
	// The read steps along the chain the trivia stands in, thus it holds the slot it opened
	// rather than standing inside a form that chain holds.
	aver.Always(depth_load(subject.Counts.Values) == depth,
		"A read of the trivia among the names holds the slot it opened.")
	subject.Nodes.Values[depth] = held
	if !bool(parted) {
		return true, false
	}
	for bool(trivia_kind(subject, tree)) {
		print_trivia(subject, tree)
		if !bool(advance(subject, tree)) {
			return true, false
		}
	}
	// The names of one run stand behind the commas that part them, and a note standing among
	// them parts none, thus the name behind the note reads as the name of that run.
	return false, Value_Named(opens_name(subject, tree))
}

// Reports whether the line the print stands on opens behind an empty line, which no second empty
// line stands behind.
func stands_after_line(subject Printer_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_after_line.yes") }()
	Printer_Handle_Invariants(subject, "stands_after_line.subject")
	written := int(written_load(subject.Counts.Values))
	if written < 2 {
		return false
	}
	form := subject.Form.Values
	if form[written-1] != '\n' {
		return false
	}
	return Boolean(form[written-2] == '\n')
}

// Reports whether the token before the node the walk stands on opens the value of a declaration.
func opens_value(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_value.yes") }()
	Printer_Handle_Invariants(subject, "opens_value.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_value.tree")
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	// The read steps into the form and never out of it, thus it stands at the slot it
	// opened or below it, and the walk stands back once it read the token.
	aver.Always(depth_load(subject.Counts.Values) >= held_depth,
		"A read of the byte one value opens on stands below the slot it opened.")
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	found := Count(ast.Syntax_Node_At(tree, leftmost).Token)
	position := found
	if position == 0 {
		return false
	}
	// A note the author left between the sign and the value states nothing of the binding,
	// thus the read steps over every note it meets.
	place := int(position) - 1
	for place > POSITION_MINIMUM {
		if ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind != token.KIND_COMMENT {
			break
		}
		place = place - 1
	}
	return ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind == token.KIND_ASSIGN
}

// Reads which of the three the walk stands on at the opening of one file.
func opening_part(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (parted Opening_Part) {
	defer func() { Opening_Part_Invariants(parted, "opening_part.parted") }()
	Printer_Handle_Invariants(subject, "opening_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opening_part.tree")
	if node_kind(subject, tree) == ast.NODE_BLANK {
		return OPENING_PART_LINE
	}
	if node_kind(subject, tree) == ast.NODE_COMMENT {
		return OPENING_PART_NOTE
	}
	return OPENING_PART_NAME
}

// Reports whether the line the print stands on already carries a note. A run opens its column
// behind the widest line it holds, and a line that took a note stands past that column, thus the
// notes behind the first one stand one space behind what they follow.
func carries_note(subject Printer_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "carries_note.yes") }()
	Printer_Handle_Invariants(subject, "carries_note.subject")
	form := subject.Form.Values
	written := int(written_load(subject.Counts.Values))
	// The read walks the line the print stands on backwards, thus it stands inside the form
	// the print wrote and never past the byte it closed on.
	aver.Always(written <= len(form),
		"A read of the line one note stands on opens inside the form.")
	place := written
	opened := false
	for range COLUMNS_READ_MAXIMUM {
		if place == 0 {
			opened = true
			break
		}
		if form[place-1] == '\n' {
			opened = true
			break
		}
		place = place - 1
	}
	// A line the read never opened stands wider than the read admits, and such a line stands
	// past every column a run opens at.
	if !opened {
		return true
	}
	for index := place; index+1 < written; index++ {
		if form[index] != '/' {
			continue
		}
		if form[index+1] != '/' {
			if form[index+1] != '*' {
				continue
			}
		}
		return true
	}
	return false
}

// Opens every counter and every flag one print stands on, and stands the walk on the tree.
func open_counts(subject Printer_Handle, tree ast.Parse_State_Handle) {
	Printer_Handle_Invariants(subject, "open_counts.subject")
	ast.Parse_State_Handle_Invariants(tree, "open_counts.tree")
	// A run of declarations stands its own place before any read of it, thus the print opens
	// every other counter.
	written_store(subject.Counts.Values, 0)
	depth_store(subject.Counts.Values, 0)
	indent_store(subject.Counts.Values, 0)
	key_width_store(subject.Counts.Values, 0)
	column_store(subject.Counts.Values, 0)
	column_indent_store(subject.Counts.Values, 0)
	columns_store(subject.Counts.Values, 0)
	// Every counter opens at none: the read of a closing bracket holds what it found for the
	// file it read, and each count of what the print wrote states what this file holds.
	for slot := range COUNT_SLOT_COUNT {
		subject.Counts.Values[slot] = 0
	}
	split_store(subject, LEVEL_MINIMUM)
	// An expression a declaration or a statement states stands at one, which is the nest
	// the canonical form spaces every sign at.
	nest_store(subject.Counts.Values, NEST_MINIMUM)
	flag_store(subject.Flags.Values, FLAG_FULL, false)
	flag_store(subject.Flags.Values, FLAG_DEEP, false)
	flag_store(subject.Flags.Values, FLAG_LINE, true)
	flag_store(subject.Flags.Values, FLAG_REFUSED, Boolean(tree.Failure != ast.FAILURE_NONE))
	subject.Nodes.Values[0] = TREE_ROOT
	opens_program(subject)
	// A print opens on the file the parse wrote, having written no byte and standing at the
	// opening of the line that file opens on.
	aver.Always(depth_load(subject.Counts.Values) == 0,
		"A print opens on the walk the file stands at.")
	aver.Always(written_load(subject.Counts.Values) == 0,
		"A print opens having written no byte of the form.")
	aver.Always(int(subject.Counts.Values[COUNT_LINE]) == 0,
		"A print opens having closed no line of the form.")
}

// Reports whether the element the walk stands on holds a line feed the print keeps. A break the
// author left between the key one pair states and the value it binds states nothing the print
// keeps, thus only the lines standing inside that value state a pair over lines of its own.
func spans_value(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "spans_value.yes") }()
	Printer_Handle_Invariants(subject, "spans_value.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "spans_value.tree")
	depth := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[depth]
	node := ast.Syntax_Node_At(tree, held)
	if node.Kind != ast.NODE_KEY_VALUE {
		return spans_form(subject, tree)
	}
	child := ast.Index(node.First_Child)
	if child == ast.INDEX_ABSENT {
		return spans_form(subject, tree)
	}
	for range ast.NODE_COUNT_MAXIMUM {
		next := ast.Index(ast.Syntax_Node_At(tree, child).Next)
		if next == ast.INDEX_ABSENT {
			break
		}
		child = next
	}
	// The read stands the walk on the value the pair binds, which is the last part it holds.
	aver.Always(ast.Index(ast.Syntax_Node_At(tree, child).Parent) == held,
		"A read of the value one pair binds stands on a part of that pair.")
	subject.Nodes.Values[depth] = child
	spans := spans_form(subject, tree)
	// The read of the value hands back the walk it was handed.
	aver.Always(depth_load(subject.Counts.Values) == depth,
		"A read of the value one pair binds hands back the walk it was handed.")
	subject.Nodes.Values[depth] = held
	return spans
}

// Reports whether the body the walk stands on holds no part at all, counting the empty lines the
// author left inside it as no part. A field list of no fields states one form the canonical form
// closes on the line it opened on.
func holds_no_part(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_no_part.yes") }()
	Printer_Handle_Invariants(subject, "holds_no_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_no_part.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the parts one form holds none of writes no byte.")
	}()
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	empty := Boolean(true)
	open := descend(subject, tree)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_BLANK {
			empty = false
		}
		open = advance(subject, tree)
	}
	// The read steps into the chain the body holds and never out of the body, thus it stands
	// at the slot it opened or below it, and the walk stands back once it read them.
	aver.Always(depth_load(subject.Counts.Values) >= depth,
		"A read of the parts one body holds stands below the slot it opened.")
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	return empty
}

// Reports whether the node the walk stands on is a comment or an empty line.
func trivia_kind(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trivia_kind.yes") }()
	Printer_Handle_Invariants(subject, "trivia_kind.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "trivia_kind.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_BLANK {
		return true
	}
	return kind == ast.NODE_COMMENT
}

// Counts the comments the line the print stands on carries, which fill the comment storage up to
// its first empty slot. The absent slot holds no comment, thus a slot that names it is empty.
func comment_count(subject Printer_Handle) (count Comment_Count) {
	defer func() { Comment_Count_Invariants(count, "comment_count.count") }()
	Printer_Handle_Invariants(subject, "comment_count.subject")
	for slot := range COMMENT_COUNT_MAXIMUM {
		if subject.Comments.Values[slot] == ast.INDEX_ABSENT {
			return Comment_Count(slot)
		}
	}
	return COMMENT_COUNT_MAXIMUM
}

// Reads the columns the line the print stands on already holds. One character stands in one
// column however many bytes it spans, and a tab stands in the columns a reader sees, thus the read
// counts the bytes that open a character and counts no byte that continues one.
func take_line_width(counts Counts) (width Line_Width) {
	defer func() { Line_Width_Invariants(width, "take_line_width.width") }()
	Counts_Invariants(counts, "take_line_width.counts")
	return columns_load(counts)
}

// Reports whether the body the walk stands on holds one part alone, counting neither the empty
// lines the author left inside it nor a note standing past the bracket it closes on.
func holds_one_part(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_one_part.yes") }()
	Printer_Handle_Invariants(subject, "holds_one_part.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_one_part.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the one part one form holds writes no byte of its own.")
	}()
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	// A note standing past the bracket the body closes on stands outside that body, thus
	// it states no part of it.
	closer := closer_position(subject, tree)
	parts := 0
	open := descend(subject, tree)
	for bool(open) {
		if node_kind(subject, tree) != ast.NODE_BLANK {
			if !bool(stands_past(subject, tree, closer)) {
				parts = parts + 1
			}
		}
		open = advance(subject, tree)
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	// The read walks the chain the body holds and hands back the node it opened on.
	aver.Always(depth_load(subject.Counts.Values) == depth,
		"A read of one part of a body stands at the walk it opened.")
	return Boolean(parts == 1)
}

// Reports whether the empty line the walk stands on stands ahead of a bracket that opens a body.
// The print writes that bracket on the line of the word ahead of it, thus an empty line between
// the two states nothing at all.
func stands_before_brace(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_before_brace.yes") }()
	Printer_Handle_Invariants(subject, "stands_before_brace.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_before_brace.tree")
	// The read answers for one empty line the author left, thus the walk stands on one.
	aver.Always(node_kind(subject, tree) == ast.NODE_BLANK,
		"A read for a line ahead of one brace stands on an empty line.")
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	position := int(ast.Syntax_Node_At(tree, index).Token)
	// A semicolon the line feed itself stands for spans no byte and opens nothing, thus the
	// read steps over it.
	ahead := position
	for ahead < ast.TOKEN_INDEX_MAXIMUM {
		one := ast.Syntax_Token_At(tree, ast.Token_Index(ahead))
		if one.Size != 0 {
			break
		}
		if one.Kind != token.KIND_SEMICOLON {
			break
		}
		ahead = ahead + 1
	}
	opener := ast.Syntax_Token_At(tree, ast.Token_Index(ahead)).Kind
	if opener == token.KIND_END_OF_FILE {
		return false
	}
	return opens_bracket(token.Scanned_Kind(opener))
}

// Reports whether the token before the node the walk stands on opens another name.
func opens_name(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_name.yes") }()
	Printer_Handle_Invariants(subject, "opens_name.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_name.tree")
	// A form names the token it opens with and never the token its own text begins at, thus
	// the test reads the token ahead of the leftmost one the form spans.
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	// The read steps into the form and never out of it, thus it stands at the slot it
	// opened or below it, and the walk stands back once it read the token.
	aver.Always(depth_load(subject.Counts.Values) >= held_depth,
		"A read of the token one name opens with stands below the slot it opened.")
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	found := Count(ast.Syntax_Node_At(tree, leftmost).Token)
	position := found
	if position == 0 {
		return false
	}
	// A note the author left between the comma and the name states nothing of the run, thus
	// the read steps over every note it meets.
	place := int(position) - 1
	for place > POSITION_MINIMUM {
		if ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind != token.KIND_COMMENT {
			break
		}
		place = place - 1
	}
	kind := ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind
	if kind == token.KIND_COMMA {
		return true
	}
	if kind == token.KIND_CONSTANT {
		return true
	}
	return kind == token.KIND_VARIABLE
}

// Steps the walk over the notes and the empty lines standing where it stands. Trivia states
// nothing of the form a read looks for, thus a read that stands on trivia steps to the part
// behind it and a read that stands on a part stands still.
func enter_named(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "enter_named.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "enter_named.tree")
	depth := depth_load(subject.Counts.Values)
	stood := written_load(subject.Counts.Values)
	for range ast.NODE_COUNT_MAXIMUM {
		if !bool(trivia_kind(subject, tree)) {
			break
		}
		next := ast.Index(ast.Syntax_Node_At(tree, subject.Nodes.Values[depth]).Next)
		if next == ast.INDEX_ABSENT {
			break
		}
		subject.Nodes.Values[depth] = next
	}
	// The step reads the tree for the name behind the trivia and writes nothing of its own,
	// thus the form stands as wide once it closed as it stood when it opened.
	aver.Always(written_load(subject.Counts.Values) == stood,
		"A step to the name behind one run of trivia writes no byte.")
}

// FORM_SIDE_COUNT is how many forms one read of two compares: the form the walk stands on and the
// form around it.
const FORM_SIDE_COUNT = 2

// Steps the walk over the type one variable binds, which the values behind it name, and holds
// every note that stood inside it or behind it. A type the print drops states nothing of its own,
// thus it writes no byte and the notes the author left there stand on the line it closes.
func drop_bound_type(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "drop_bound_type.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "drop_bound_type.tree")
	stood := written_load(subject.Counts.Values)
	counts_dropped_type(subject, tree)
	take_form_notes(subject, tree)
	held := Boolean(!bool(advance(subject, tree)))
	for range ast.NODE_COUNT_MAXIMUM {
		if bool(held) {
			break
		}
		if !bool(trivia_kind(subject, tree)) {
			break
		}
		print_trivia(subject, tree)
		held = Boolean(!bool(advance(subject, tree)))
	}
	// The print drops the type where a value behind it names that type, thus a value
	// stands where the walk closes.
	aver.Always(!bool(held),
		"A type the print drops stands ahead of the value that names it.")
	aver.Always(written_load(subject.Counts.Values) == stood,
		"A variable whose value names its type writes no byte for that type.")
}

// Reports whether the empty line the walk stands on stands inside the head of one declaration:
// the run of words, names, types and values it writes ahead of the body it opens. A head stands
// on one line, thus an empty line inside it states nothing at all.
func stands_inside_head(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_inside_head.yes") }()
	Printer_Handle_Invariants(subject, "stands_inside_head.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_inside_head.tree")
	// The read answers for the empty lines one head holds, thus the walk stands on one.
	aver.Always(node_kind(subject, tree) == ast.NODE_BLANK,
		"A read for a line inside one head stands on an empty line.")
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	held := ast.Syntax_Node_At(tree, index)
	// An empty line the declaration closes on stands behind everything the declaration
	// writes, thus it belongs to the file and states the space between two declarations.
	if ast.Index(held.Next) == 0 {
		return false
	}
	switch ast.Syntax_Node_At(tree, ast.Index(held.Parent)).Kind {
	case ast.NODE_IMPORT, ast.NODE_CONSTANT, ast.NODE_VARIABLE, ast.NODE_TYPE:
		return true
	case ast.NODE_FUNCTION, ast.NODE_RECEIVER, ast.NODE_TYPE_PARAMETER:
		return true
	case ast.NODE_PARAMETER, ast.NODE_FIELD, ast.NODE_RESULT:
		return true
	case ast.NODE_UNION, ast.NODE_TERM, ast.NODE_BINARY, ast.NODE_UNARY:
		return true
	case ast.NODE_SELECTOR, ast.NODE_ASSERTION, ast.NODE_SLICE_EXPRESSION:
		return true
	case ast.NODE_DEFER, ast.NODE_GO, ast.NODE_RETURN, ast.NODE_EXPRESSION_STATEMENT:
		return true
	case ast.NODE_ASSIGN, ast.NODE_DEFINE, ast.NODE_OPERATION_ASSIGN, ast.NODE_SEND:
		return true
	case ast.NODE_KEY_VALUE, ast.NODE_PARENTHESIS:
		return true
	case ast.NODE_FUNCTION_LITERAL, ast.NODE_FUNCTION_TYPE:
		return true
	case ast.NODE_POINTER_TYPE, ast.NODE_SLICE_TYPE, ast.NODE_ARRAY_TYPE:
		return true
	case ast.NODE_MAP_TYPE, ast.NODE_CHANNEL_TYPE, ast.NODE_ELLIPSIS:
		return true
	case ast.NODE_CHANNEL_SEND, ast.NODE_CHANNEL_RECEIVE, ast.NODE_GENERIC:
		return true
	}
	return false
}

// Reports whether the empty line the walk stands on stands between the sign of an assignment and
// the value that sign binds, which is a line that states nothing: the value stands on the line
// the sign closes.
func stands_behind_sign(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_behind_sign.yes") }()
	Printer_Handle_Invariants(subject, "stands_behind_sign.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_behind_sign.tree")
	// The read answers for one empty line the author left, thus the walk stands on one.
	aver.Always(node_kind(subject, tree) == ast.NODE_BLANK,
		"A read for a line behind one sign stands on an empty line.")
	if node_kind(subject, tree) != ast.NODE_BLANK {
		return false
	}
	held_depth := depth_load(subject.Counts.Values)
	blank := ast.Syntax_Node_At(tree, subject.Nodes.Values[held_depth]).Token
	enter_assignment(subject, tree)
	if depth_load(subject.Counts.Values) == held_depth {
		return false
	}
	// The values of the assignment open at the first part the sign stands ahead of.
	found := false
	open := descend(subject, tree)
	for bool(open) {
		if bool(opens_bound(subject, tree)) {
			depth := depth_load(subject.Counts.Values)
			enter_leftmost(subject, tree)
			leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
			depth_store(subject.Counts.Values, depth)
			found = ast.Syntax_Node_At(tree, leftmost).Token == blank
			break
		}
		open = advance(subject, tree)
	}
	depth_store(subject.Counts.Values, held_depth)
	return Boolean(found)
}

// States that the print closed every declaration the file holds, once. The rules order the
// declarations of a file and move one under the type it names, thus a count of the ones the print
// closed reads the count the file states where every rule wrote what it took.
func closes_declarations(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "closes_declarations.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_declarations.tree")
	held := 0
	child := ast.Index(ast.Syntax_Node_At(tree, subject.Nodes.Values[0]).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			break
		}
		node := ast.Syntax_Node_At(tree, child)
		child = ast.Index(node.Next)
		if node.Kind == ast.NODE_COMMENT {
			continue
		}
		if node.Kind == ast.NODE_BLANK {
			continue
		}
		// The name the file binds stands as a child of its own and states no
		// declaration the print closes.
		if node.Kind == ast.NODE_IDENTIFIER {
			continue
		}
		held = held + 1
	}
	aver.Always(int(subject.Counts.Values[COUNT_DECLARED_COUNT]) == held,
		"A print closes every declaration the file states, once.")
	// Every declaration closes the line it stands on, and the clause the file opens with
	// closes one of its own, thus the form holds a line for each of them.
	aver.Always(int(subject.Counts.Values[COUNT_LINE]) > held,
		"A print closes a line for the clause and for every declaration behind it.")
}

// States that the print counted one line for each feed the form holds. The count states whether a
// form the print wrote stands over lines of its own, thus a count the bytes never spell would
// close a run of pairs that never ran over lines.
func closes_lines(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "closes_lines.subject")
	feeds := 0
	form := subject.Form.Values
	spaced := Boolean(false)
	returned := Boolean(false)
	for place := range int(written_load(subject.Counts.Values)) {
		if form[place] == '\r' {
			returned = true
		}
		if form[place] != '\n' {
			continue
		}
		feeds = feeds + 1
		if place == 0 {
			continue
		}
		if form[place-1] == ' ' {
			spaced = true
		}
		if form[place-1] == '\t' {
			spaced = true
		}
	}
	aver.Always(int(subject.Counts.Values[COUNT_LINE]) == feeds,
		"A print closes one line for each feed the form holds.")
	// A reader reads no space standing at the end of a line, thus the run that pads one line
	// of a column hands back every byte it spent on the line it closed.
	aver.Always(!bool(spaced), "No line the form holds closes on a space or a tab.")
	// One line feed closes each line of the form, thus no return stands in it.
	aver.Always(!bool(returned), "No line the form holds closes on a carriage return.")
	// The form states the file whole, thus it holds a byte, and every declaration closes
	// the line it stands on, thus it closes on the feed of the last of them.
	written := int(written_load(subject.Counts.Values))
	aver.Always(written > 0, "The form one print closed holds a byte.")
	// One file opens on the clause that names the package it binds, or on a note the author
	// wrote above that clause, thus the form opens on the word the clause states or on the
	// slash one note opens with.
	opening := word_text(WORD_PACKAGE)
	if written > len(opening) {
		matched := 0
		for index := range len(opening) {
			if form[index] == opening[index] {
				matched = matched + 1
			}
		}
		opened := Boolean(matched == len(opening))
		if form[0] == byte(47) {
			opened = true
		}
		aver.Always(bool(opened), "The form opens on the clause or on a note above it.")
	}
	aver.Always(form[written-1] == byte(10),
		"The form closes on the feed of its last line.")
}

// Steps the walk over the notes and the empty lines the author left between the type one field
// states and the tag behind it, holding each of them, and reports whether a part stands where the
// walk now stands. Trivia standing behind the last part of a field belongs to the lines behind
// that field, thus the walk steps over none of it where no tag stands behind it.
func enter_tag(subject Printer_Handle, tree ast.Syntax_Tree_Handle, open Boolean) (stood Boolean) {
	defer func() { Boolean_Invariants(stood, "enter_tag.stood") }()
	Printer_Handle_Invariants(subject, "enter_tag.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "enter_tag.tree")
	Boolean_Invariants(open, "enter_tag.open")
	if !bool(open) {
		return open
	}
	ahead := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	tagged := Boolean(false)
	for range ast.NODE_COUNT_MAXIMUM {
		kind := ast.Syntax_Node_At(tree, ahead).Kind
		if kind != ast.NODE_COMMENT {
			if kind != ast.NODE_BLANK {
				tagged = true
				break
			}
		}
		next := ast.Index(ast.Syntax_Node_At(tree, ahead).Next)
		if next == ast.INDEX_ABSENT {
			break
		}
		ahead = next
	}
	if !bool(tagged) {
		return open
	}
	held := open
	for bool(trivia_kind(subject, tree)) {
		print_trivia(subject, tree)
		held = advance(subject, tree)
	}
	// The read steps over the trivia ahead of the tag, thus a walk that still stands names
	// the tag itself and never the notes that stood ahead of it.
	if bool(held) {
		aver.Always(!bool(trivia_kind(subject, tree)),
			"A walk that read the notes ahead of a tag stands on the tag.")
	}
	return held
}

// Reports whether the declaration the walk stands on is a function that states the invariants of
// a type the file itself declares. A function that answers for a type of another file names no
// type to stand under, thus it stands where the author wrote it.
func states_own_type(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_own_type.yes") }()
	Printer_Handle_Invariants(subject, "states_own_type.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "states_own_type.tree")
	depth := depth_load(subject.Counts.Values)
	one := subject.Nodes.Values[depth]
	node := ast.Syntax_Node_At(tree, one)
	if !bool(holds_pair(subject, tree)) {
		return false
	}
	child := ast.Index(ast.Syntax_Node_At(tree, ast.Index(node.Parent)).First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			return false
		}
		held := ast.Syntax_Node_At(tree, child)
		if held.Kind == ast.NODE_TYPE {
			subject.Nodes.Values[depth+1] = child
			subject.Nodes.Values[depth+2] = one
			depth_store(subject.Counts.Values, depth+2)
			paired := names_pair(subject, tree)
			depth_store(subject.Counts.Values, depth)
			// The read stands two slots below the pair and hands the walk back.
			aver.Always(depth_load(subject.Counts.Values) == depth,
				"A read of the type one pair names stands at its own depth.")
			if bool(paired) {
				return true
			}
		}
		child = ast.Index(held.Next)
	}
	return false
}

// States that the print wrote every part of the tree its rules must write, once: the notes, the
// fields, the parameters and the results, the cases, the statements, the pairs, the terms, and
// the labels. One walk of the arena reads every count, thus the laws cost one pass of the nodes
// rather than a pass for each of them.
func closes_nodes(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_nodes.subject")
	Counts_Invariants(held, "closes_nodes.held")
	counts := subject.Counts.Values
	aver.Always(counts[COUNT_NOTE] == held[COUNT_NOTE],
		"A print writes every note the file states, once.")
	aver.Always(counts[COUNT_FIELD] == held[COUNT_FIELD],
		"A print writes every field the file states, once.")
	aver.Always(counts[COUNT_PARAMETER] == held[COUNT_PARAMETER],
		"A print writes every parameter the file states, once.")
	aver.Always(counts[COUNT_CASE] == held[COUNT_CASE],
		"A print writes every case the file states, once.")
	aver.Always(counts[COUNT_RESULT] == held[COUNT_RESULT],
		"A print writes every result the file states, once.")
	aver.Always(counts[COUNT_DEFAULT] == held[COUNT_DEFAULT],
		"A print writes every default the file states, once.")
	aver.Always(counts[COUNT_STATEMENT] == held[COUNT_STATEMENT],
		"A print writes every statement the file states, once.")
	aver.Always(counts[COUNT_LABEL] == held[COUNT_LABEL],
		"A print writes every label the file states, once.")
	aver.Always(counts[COUNT_FIELD_NAME] == held[COUNT_FIELD_NAME],
		"A print writes every name of a field the file states, once.")
	aver.Always(counts[COUNT_PARAMETER_NAME] == held[COUNT_PARAMETER_NAME],
		"A print writes every name of a parameter the file states, once.")
	aver.Always(counts[COUNT_RECEIVER] == held[COUNT_RECEIVER],
		"A print writes every receiver the file states, once.")
	aver.Always(counts[COUNT_NAME]+counts[COUNT_NAME_DROPPED] == held[COUNT_NAME],
		"A print writes or drops every name the file states, once.")
	aver.Always(counts[COUNT_GROUP] == held[COUNT_GROUP],
		"A print writes every grouping the file states, once.")
	aver.Always(counts[COUNT_UNARY] == held[COUNT_UNARY],
		"A print writes every sign of one value the file states, once.")
	aver.Always(counts[COUNT_CALL]+counts[COUNT_CALL_DROPPED] == held[COUNT_CALL],
		"A print writes or drops every call the file states, once.")
	aver.Always(counts[COUNT_INDEX] == held[COUNT_INDEX],
		"A print writes every read of one slot the file states, once.")
}

// States that the print wrote every form of a kind it never drops, once: the pairs, the terms,
// the unions, the type parameters, the bodies, and the literals, selectors, assertions, slices,
// function literals and ranges a file states.
func closes_forms(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_forms.subject")
	Counts_Invariants(held, "closes_forms.held")
	counts := subject.Counts.Values
	aver.Always(counts[COUNT_PAIR] == held[COUNT_PAIR],
		"A print writes every pair the file states, once.")
	aver.Always(counts[COUNT_TERM] == held[COUNT_TERM],
		"A print writes every term the file states, once.")
	aver.Always(counts[COUNT_UNION] == held[COUNT_UNION],
		"A print writes every union the file states, once.")
	aver.Always(counts[COUNT_TYPE_PARAMETER] == held[COUNT_TYPE_PARAMETER],
		"A print writes every type parameter the file states, once.")
	aver.Always(counts[COUNT_BLOCK] == held[COUNT_BLOCK],
		"A print writes every body the file states, once.")
	aver.Always(counts[COUNT_FORM] == held[COUNT_FORM],
		"A print writes every form of a kind it never drops, once.")
	closes_kinds(subject, held)
	closes_clauses(subject, held)
	closes_words(subject, held)
	closes_bindings(subject, held)
}

// States that the print wrote every form of each kind it never drops, once. A count that holds
// every kind as one reads the same where the print writes one form of a kind twice and drops one
// of another, thus each kind answers for itself.
func closes_kinds(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_kinds.subject")
	Counts_Invariants(held, "closes_kinds.held")
	counts := subject.Counts.Values
	aver.Always(counts[COUNT_OPERATION] == held[COUNT_OPERATION],
		"A print writes every operation the file states, once.")
	aver.Always(counts[COUNT_SELECTOR] == held[COUNT_SELECTOR],
		"A print writes every selector the file states, once.")
	aver.Always(counts[COUNT_ASSERTION] == held[COUNT_ASSERTION],
		"A print writes every assertion the file states, once.")
	aver.Always(counts[COUNT_LITERAL] == held[COUNT_LITERAL],
		"A print writes every literal the file states, once.")
	aver.Always(counts[COUNT_SLICE] == held[COUNT_SLICE],
		"A print writes every slice the file states, once.")
	aver.Always(counts[COUNT_FUNCTION] == held[COUNT_FUNCTION],
		"A print writes every function literal the file states, once.")
	aver.Always(counts[COUNT_RANGE] == held[COUNT_RANGE],
		"A print writes every range the file states, once.")
	// The count of every form of a kind it never drops holds each kind once, thus the parts
	// of that count state it whole.
	parts := counts[COUNT_OPERATION] + counts[COUNT_SELECTOR] + counts[COUNT_ASSERTION]
	parts = parts + counts[COUNT_LITERAL] + counts[COUNT_SLICE] + counts[COUNT_FUNCTION]
	aver.Always(counts[COUNT_FORM] == parts+counts[COUNT_RANGE],
		"The count of the forms one print wrote states the kinds it holds.")
	closes_types(subject, held)
}

// States that the print wrote or dropped every type of the kinds that spell a form of their own,
// once. A type the value behind it names stands dropped, thus the two counts state the file.
func closes_types(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_types.subject")
	Counts_Invariants(held, "closes_types.held")
	counts := subject.Counts.Values
	aver.Always(counts[COUNT_TYPE_FORM]+counts[COUNT_TYPE_DROPPED] == held[COUNT_TYPE_FORM],
		"A print writes or drops every type of a form of its own, once.")
	closes_type_kinds(subject, held)
}

// Counts one run of the rule that writes a parameter, a result, or a type parameter. Each of them
// runs once for the node it writes, thus the counts read what the signatures of the file state
// once the print closed the file.
func counts_rule(subject Printer_Handle, tree ast.Syntax_Tree_Handle, rule Layout_Rule) {
	Printer_Handle_Invariants(subject, "counts_rule.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "counts_rule.tree")
	Layout_Rule_Invariants(rule, "counts_rule.rule")
	if rule == RULE_PARAMETER {
		if node_kind(subject, tree) == ast.NODE_RESULT {
			subject.Counts.Values[COUNT_RESULT] += 1
		}
		if node_kind(subject, tree) != ast.NODE_RESULT {
			subject.Counts.Values[COUNT_PARAMETER] += 1
		}
	}
	if rule == RULE_TYPE_PARAMETER {
		subject.Counts.Values[COUNT_TYPE_PARAMETER] += 1
	}
	if rule == RULE_RECEIVER {
		subject.Counts.Values[COUNT_RECEIVER] += 1
	}
}

// Reports whether the node the walk stands on opens the right side of an assignment, which the
// sign that stands ahead of it states.
func opens_right(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_right.yes") }()
	Printer_Handle_Invariants(subject, "opens_right.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_right.tree")
	held_depth := depth_load(subject.Counts.Values)
	enter_leftmost(subject, tree)
	// The read steps into the form and never out of it, thus it stands at the slot it
	// opened or below it, and the walk stands back once it read the token.
	aver.Always(depth_load(subject.Counts.Values) >= held_depth,
		"A read of the token one right side opens with stands below the slot it opened.")
	leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth_store(subject.Counts.Values, held_depth)
	found := Count(ast.Syntax_Node_At(tree, leftmost).Token)
	position := found
	if position == 0 {
		return false
	}
	// A note the author left behind the sign stands between the sign and the value it binds,
	// thus the read steps over every note it meets.
	behind := int(position) - 1
	for behind > 0 {
		if ast.Syntax_Token_At(tree, ast.Token_Index(behind)).Kind != token.KIND_COMMENT {
			break
		}
		behind = behind - 1
	}
	// An assignment and a send each state their sign ahead of the right side.
	switch ast.Syntax_Token_At(tree, ast.Token_Index(behind)).Kind {
	case token.KIND_ASSIGN, token.KIND_DEFINE, token.KIND_ARROW, token.KIND_PLUS_ASSIGN,
		token.KIND_MINUS_ASSIGN, token.KIND_STAR_ASSIGN, token.KIND_SLASH_ASSIGN,
		token.KIND_PERCENT_ASSIGN, token.KIND_AND_ASSIGN, token.KIND_OR_ASSIGN,
		token.KIND_EXCLUSIVE_OR_ASSIGN, token.KIND_SHIFT_LEFT_ASSIGN,
		token.KIND_SHIFT_RIGHT_ASSIGN, token.KIND_AND_NOT_ASSIGN:
		return true
	}
	return false
}

// Reports whether the empty line the walk stands on stands inside a body that closes up against
// its brackets. A form inside a body states its own answer for the lines against its brackets,
// thus an empty line standing deep inside a statement reads the answer of the body around it
// rather than the answer of the form that holds it.
func closes_tight_body(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "closes_tight_body.yes") }()
	Printer_Handle_Invariants(subject, "closes_tight_body.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_tight_body.tree")
	depth := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[depth]
	node := ast.Index(ast.Syntax_Node_At(tree, held).Parent)
	for range ast.NODE_COUNT_MAXIMUM {
		if node == ast.INDEX_ABSENT {
			return false
		}
		if ast.Syntax_Node_At(tree, node).Kind == ast.NODE_BLOCK {
			break
		}
		node = ast.Index(ast.Syntax_Node_At(tree, node).Parent)
	}
	if node == ast.INDEX_ABSENT {
		return false
	}
	// The body one function states closes up however many statements it holds.
	holder := ast.Index(ast.Syntax_Node_At(tree, node).Parent)
	if holder != ast.INDEX_ABSENT {
		kind := ast.Syntax_Node_At(tree, holder).Kind
		if kind == ast.NODE_FUNCTION {
			return true
		}
		if kind == ast.NODE_FUNCTION_LITERAL {
			return true
		}
	}
	subject.Nodes.Values[depth] = node
	tight := Boolean(bool(holds_one_part(subject, tree)) || bool(holds_no_part(subject, tree)))
	// The read of the parts the body holds hands back the walk it was handed, thus the read
	// stands the node it climbed to back on its own slot.
	aver.Always(depth_load(subject.Counts.Values) == depth,
		"A read of the body around one line holds the slot it opened.")
	subject.Nodes.Values[depth] = held
	return tight
}

// Reports whether an empty line stands behind the statement the walk stands on. The parse hangs
// such a line inside the form the statement closes on, thus the read runs the tokens past the
// notes the statement carries and reads the lines the scanner counted at the token behind them.
func trails_blank(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "trails_blank.yes") }()
	Printer_Handle_Invariants(subject, "trails_blank.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "trails_blank.tree")
	stood := depth_load(subject.Counts.Values)
	enter_rightmost(subject, tree)
	// The read steps into the statement and never out of it, thus it stands at the slot it
	// opened or below it, and the walk stands back once it read the token.
	aver.Always(depth_load(subject.Counts.Values) >= stood,
		"A read of the line behind one statement stands below the slot it opened.")
	final := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	depth_store(subject.Counts.Values, stood)
	place := final.Token + 1
	for range COMMENT_COUNT_MAXIMUM {
		if ast.Syntax_Token_At(tree, place).Kind != token.KIND_COMMENT {
			break
		}
		place = place + 1
	}
	return Boolean(ast.Syntax_Token_At(tree, place).Blanks > 0)
}

// Reports whether one statement states a variable the short sign binds: a run of names, the
// values behind them, and no type at all. A declaration that wears a type states a form the short
// sign cannot hold, and a constant binds no short sign of any kind.
func states_short(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_short.yes") }()
	Printer_Handle_Invariants(subject, "states_short.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "states_short.tree")
	if node_kind(subject, tree) != ast.NODE_VARIABLE {
		return false
	}
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	stood := written_load(subject.Counts.Values)
	short := Boolean(false)
	open := descend(subject, tree)
	for bool(open) {
		// A note or an empty line the author left behind the values states no type,
		// thus the read steps over the trivia the declaration carries.
		if bool(trivia_kind(subject, tree)) {
			open = advance(subject, tree)
			continue
		}
		if bool(opens_value(subject, tree)) {
			short = true
		} else if !bool(opens_name(subject, tree)) {
			// A type the values behind it name writes no byte, thus the form the
			// print writes states no type and the short sign binds it.
			if !bool(repeats_bound_type(subject, tree, false)) {
				short = false
				break
			}
		}
		open = advance(subject, tree)
	}
	// The read walks the parts the declaration holds and reads the tree alone, thus it
	// writes no byte and hands back the walk it opened.
	aver.Always(written_load(subject.Counts.Values) == stood,
		"A read of the sign one declaration binds writes no byte of its own.")
	aver.Always(depth_load(subject.Counts.Values) >= depth,
		"A read of the sign one declaration binds stands below the slot it opened.")
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	return short
}

// Writes one grouping: the parentheses the author wrote and the value between them. Parentheses
// state the binding the signs inside them would otherwise state, thus the value inside stands one
// level shallower than the form around it.
func print_group(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_group.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_group.tree")
	// A grouping holds one value between brackets, thus the walk stands on one where the
	// print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_PARENTHESIS,
			"A grouping the print writes stands on a grouping of the tree.")
	}
	counts_type(subject, tree)
	subject.Counts.Values[COUNT_GROUP] += 1
	held := nest_load(subject.Counts.Values)
	stood := depth_load(subject.Counts.Values)
	inner := Nest(NEST_MINIMUM)
	if held > 1 {
		inner = held - 1
	}
	emit_byte(subject, '(')
	open := descend_part(subject, tree)
	entered := open
	if bool(open) {
		nest_store(subject.Counts.Values, inner)
		print_expression(subject, tree)
		nest_store(subject.Counts.Values, held)
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
	// A grouping closes on the bracket it opened, thus the value it holds stands between the
	// two and the form holds both.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(stands_after_space(subject)),
			"A grouping closes its bracket against the value it holds.")
		// The value between the brackets hands back the walk and the nest it was
		// handed, thus the grouping closes where it opened.
		aver.Always(depth_load(subject.Counts.Values) == stood,
			"A grouping closes at the depth it opened at.")
		aver.Always(nest_load(subject.Counts.Values) == held,
			"A grouping closes at the nest it opened at.")
	}
	emit_byte(subject, ')')
}

// Writes one string literal. A return the author left between the quotes of a raw string states
// nothing of its value, thus the canonical form spells the string without it.
func emit_string(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "emit_string.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "emit_string.tree")
	cursor := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	one := ast.Syntax_Token_At(tree, cursor.Token)
	text := token.Text(token.Source(subject.Source.Values), one)
	stood := int(written_load(subject.Counts.Values))
	dropped := 0
	for index := range len(text) {
		if text[index] == '\r' {
			dropped = dropped + 1
			continue
		}
		emit_byte(subject, Symbol(text[index]))
	}
	// A string stands in the form as the author spelled it, thus every byte of it but the
	// returns it drops lands there wherever the storage the caller supplied holds room.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(int(written_load(subject.Counts.Values))-stood == len(text)-dropped,
			"A string the print writes lands in the form but for the returns it drops.")
	}
	// A string spells its bytes between marks of one kind, thus the form holds that mark at
	// each end of what the print wrote.
	written := int(written_load(subject.Counts.Values))
	if bool(writes_canonical(subject.Flags.Values)) {
		if written > stood+1 {
			opened := subject.Form.Values[stood]
			marked := Boolean(opened == '"')
			if opened == '`' {
				marked = true
			}
			aver.Always(bool(marked),
				"A string opens on a mark of the kind it states.")
			aver.Always(subject.Form.Values[written-1] == opened,
				"A string closes on the mark it opens on.")
		}
	}
}

// Writes the spaces that carry the print from a part of one width to one column, and one space
// where the part already reaches it.
func emit_column(subject Printer_Handle, width Part_Width, column Column_Width) {
	Printer_Handle_Invariants(subject, "emit_column.subject")
	Part_Width_Invariants(width, "emit_column.width")
	Column_Width_Invariants(column, "emit_column.column")
	held := Width(column)
	if held < Width(width) {
		held = Width(width)
	}

	// The padding of a column stands as one run of spaces, thus the print writes its bytes
	// rather than the space that holds two bytes of a line apart.
	stood := int(written_load(subject.Counts.Values))
	for range int(held-Width(width)) + 1 {
		emit_byte(subject, SYMBOL_SPACE)
	}
	// A part that already reaches the column takes one space, thus the padding always
	// carries the line by a space at least.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(stands_after_space(subject)),
			"A column the print pads to closes on a space.")
		// The padding spends one byte for each column it carries the line, thus the
		// part behind it opens at the column the run states.
		aver.Always(int(written_load(subject.Counts.Values))-stood ==
			int(held-Width(width))+1,
			"A column the print pads to spends a byte for each column it carries.")

	}
}

// Moves the walk to the first part of the node it stands on, over trivia and writing nothing,
// thus a measure the print takes leaves the form it wrote untouched.
func descend_quiet(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "descend_quiet.ok") }()
	Printer_Handle_Invariants(subject, "descend_quiet.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "descend_quiet.tree")
	held_depth := depth_load(subject.Counts.Values)
	if !bool(descend(subject, tree)) {
		aver.Always(depth_load(subject.Counts.Values) == held_depth,
			"A quiet step down that finds no part leaves the depth it stood at.")
		return false
	}
	if !bool(trivia_kind(subject, tree)) {
		aver.Always(depth_load(subject.Counts.Values) == held_depth+1,
			"A quiet step down that finds a part stands one slot below the form.")
		return true
	}
	if bool(advance_quiet(subject, tree)) {
		aver.Always(depth_load(subject.Counts.Values) == held_depth+1,
			"A quiet step over the trivia of one form holds the slot it opened.")
		return true
	}
	// A form of trivia alone holds no part, and a walk that reports no part must stand where
	// it stood, thus the step down closes here rather than in a caller that never took it.
	ascend(subject)
	aver.Always(depth_load(subject.Counts.Values) == held_depth,
		"A quiet step back over the trivia of one form leaves the depth it stood at.")
	return false
}

// Moves the walk to the next part of the form it stands in, over trivia and writing nothing.
func advance_quiet(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (ok Boolean) {
	defer func() { Boolean_Invariants(ok, "advance_quiet.ok") }()
	Printer_Handle_Invariants(subject, "advance_quiet.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "advance_quiet.tree")
	held_depth := depth_load(subject.Counts.Values)
	held_node := subject.Nodes.Values[held_depth]
	for bool(advance(subject, tree)) {
		if !bool(trivia_kind(subject, tree)) {
			// The step holds the slot it stood on and stands on a sibling of the
			// node it left, however much trivia it stepped over.
			aver.Always(depth_load(subject.Counts.Values) == held_depth,
				"A quiet step to the next part holds the slot it stood on.")
			aver.Always(subject.Nodes.Values[held_depth] != held_node,
				"A quiet step to the next part leaves the part it stood on.")
			return true
		}
	}
	aver.Always(depth_load(subject.Counts.Values) == held_depth,
		"A quiet step that finds no next part holds the slot it stood on.")
	return false
}

// Opens the braces of one literal at its first element: the brace, the break the author left
// behind it, and the indent the elements answer to.
func open_braces(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, force Boolean,
) (result Literal_Open) {
	defer func() { Literal_Open_Invariants(result, "open_braces.result") }()
	Printer_Handle_Invariants(subject, "open_braces.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_braces.tree")
	Boolean_Invariants(force, "open_braces.force")
	stood := int(written_load(subject.Counts.Values))
	emit_byte(subject, '{')
	// The brace the literal opens on stands in the form, thus the elements it holds stand
	// behind that byte.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(subject.Form.Values[stood] == '{',
			"A literal opens on the brace that holds its elements.")
	}
	indent := indent_load(subject.Counts.Values)
	marked := flag_load(subject.Flags.Values, FLAG_BROKEN)
	opened := open_literal(subject, tree, force)
	return Literal_Open{
		Indent: indent, Marked: Marked_Break(marked), Opened: Opened_Break(opened),
		Broken: Broken_Form(opened || force),
	}
}

// Closes one literal: the line the parts of a broken form stand apart on, the walk the print
// opened, and the braces. A literal of no parts at all opens its brace here, because the print
// writes that brace at its first part and no part stood.
func close_literal(
	subject Printer_Handle,
	tree ast.Syntax_Tree_Handle,
	broken Boolean,
	entered Boolean,
	empty Boolean,
	tail Boolean,
) {
	Printer_Handle_Invariants(subject, "close_literal.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "close_literal.tree")
	Boolean_Invariants(broken, "close_literal.broken")
	Boolean_Invariants(entered, "close_literal.entered")
	Boolean_Invariants(empty, "close_literal.empty")
	Boolean_Invariants(tail, "close_literal.tail")
	stood := written_load(subject.Counts.Values)
	if bool(broken) {
		close_broken(subject, tree, tail, form_closer(subject, tree))
	}
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
	if bool(empty) {
		emit_byte(subject, '{')
		// A literal that holds notes alone keeps them between its braces, thus they stand
		// on the line the brace opens rather than behind the declaration.
		if comment_count(subject) > 0 {
			indent_deepen(subject, indent_load(subject.Counts.Values))
			close_line(subject, tree)
			indent_shallow(subject, indent_load(subject.Counts.Values))
			emit_indent(subject)
		}
	}
	emit_byte(subject, '}')
	// The brace closes the elements the literal holds, thus the last byte it wrote states
	// that brace.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(subject.Form.Values[int(written_load(subject.Counts.Values))-1] == '}',
			"A literal closes on the brace that holds its elements.")
	}
	// A literal of no part at all writes the two braces it stands between and nothing else,
	// thus a reader sees the form close where it opened.
	if bool(empty) {
		if bool(writes_canonical(subject.Flags.Values)) {
			aver.Always(written_load(subject.Counts.Values) > stood+1,
				"A literal of no part writes the braces it stands between.")
		}
	}
}

// States that the bracket one wrapped form opens on stands in the form.
func opens_bracket_byte(subject Printer_Handle, wrap Wrap) {
	Printer_Handle_Invariants(subject, "opens_bracket_byte.subject")
	Wrap_Invariants(wrap, "opens_bracket_byte.wrap")
	if bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		return
	}
	held := byte('(')
	if wrap == WRAP_INDEX {
		held = '['
	}
	aver.Always(subject.Form.Values[int(written_load(subject.Counts.Values))-1] == held,
		"A wrapped form opens on the bracket that holds its parts.")
}

// States that the bracket one wrapped form closes on stands in the form.
func closes_bracket_byte(subject Printer_Handle, wrap Wrap) {
	Printer_Handle_Invariants(subject, "closes_bracket_byte.subject")
	Wrap_Invariants(wrap, "closes_bracket_byte.wrap")
	if bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		return
	}
	held := byte(')')
	if wrap == WRAP_INDEX {
		held = ']'
	}
	aver.Always(subject.Form.Values[int(written_load(subject.Counts.Values))-1] == held,
		"A wrapped form closes on the bracket that holds its parts.")
}

// Reports whether the path standing one slot past the import the walk stands on names a package.
// The name the path states reads the same written as unwritten, thus the print writes it past
// what it already wrote, reads whether it spelled a name a file can bind, and takes it back.
func names_path(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (named Boolean) {
	defer func() { Boolean_Invariants(named, "names_path.named") }()
	Printer_Handle_Invariants(subject, "names_path.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "names_path.tree")
	depth := depth_load(subject.Counts.Values)
	var flags [FLAG_COUNT]Boolean
	copy(flags[:], subject.Flags.Values)
	var comments [COMMENT_COUNT_MAXIMUM]ast.Index
	copy(comments[:], subject.Comments.Values)
	start := written_load(subject.Counts.Values)
	// The trial print writes the name into the form and takes it back, thus every count of
	// what the print wrote stands back with it.
	var counts [COUNT_SLOT_COUNT]Count
	copy(counts[:], subject.Counts.Values)
	held_node := subject.Nodes.Values[depth]
	depth_store(subject.Counts.Values, depth+1)
	emit_package_name(subject, tree)
	depth_store(subject.Counts.Values, depth)
	// The trial print stands one slot past the import and writes the name it reads, thus it
	// leaves the slot the import stands on and the byte count it began at alone.
	aver.Always(subject.Nodes.Values[depth] == held_node,
		"A trial print of one package name leaves the import it stands on.")
	aver.Always(written_load(subject.Counts.Values) >= start,
		"A trial print of one package name writes past the byte count it began at.")
	named = written_load(subject.Counts.Values) > start
	// A file binds the name of a package the way it binds any other name, thus a path element
	// that spells no such name names no package and the import keeps the name it was given.
	for place := int(start); place < int(written_load(subject.Counts.Values)); place++ {
		held := subject.Form.Values[place]
		spelled := Boolean(held == '_')
		if held >= 'a' {
			if held <= 'z' {
				spelled = true
			}
		}
		if held >= 'A' {
			if held <= 'Z' {
				spelled = true
			}
		}
		if place > int(start) {
			if held >= '0' {
				if held <= '9' {
					spelled = true
				}
			}
		}
		if !bool(spelled) {
			named = false
		}
	}
	restore_counts(subject, counts[:])
	restore_flags(subject, flags[:])
	copy(subject.Comments.Values, comments[:])
	return named
}

// Writes one increment or decrement: the value and the two signs behind it. The tree holds one
// node for the whole step, thus its class states which pair of signs the print owes.
func print_step(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_step.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_step.tree")
	// A step states one pair of signs behind the value it holds, thus the walk stands on a
	// step of one kind or the other where the print writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		Step_Kind_Invariants(step_kind(subject, tree), "print_step.stepped")
	}
	value := byte('+')
	if node_kind(subject, tree) == ast.NODE_DECREMENT {
		value = '-'
	}
	if bool(descend_part(subject, tree)) {
		print_expression(subject, tree)
		// The empty line behind a step hangs inside the step, thus the walk steps over
		// it rather than closing the statement without it.
		finish_parts(subject, tree)
		ascend(subject)
	}
	// A step states its signs against the value it counts, thus no space holds the two apart.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(stands_after_space(subject)),
			"A step states its signs against the value it counts.")
	}
	emit_byte(subject, Symbol(value))
	emit_byte(subject, Symbol(value))
	// A step states one pair of signs, thus the two bytes the statement closes on spell the
	// same sign and no reader reads one of them alone.
	if bool(writes_canonical(subject.Flags.Values)) {
		place := int(written_load(subject.Counts.Values))
		aver.Always(subject.Form.Values[place-1] == value,
			"A step closes on the sign it counts its value with.")
		aver.Always(subject.Form.Values[place-2] == value,
			"A step closes on a pair of the sign it counts its value with.")
	}
}

// Writes one label and the statement it names.
func print_label(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_label.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_label.tree")
	// A label names the statement behind it, thus the walk stands on a label where the print
	// writes one.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_LABEL,
			"A label the print writes stands on a label of the tree.")
	}
	subject.Counts.Values[COUNT_LABEL] += 1
	emit_token(subject, tree)
	// A label names the statement behind it, thus the colon closes against that name and no
	// space stands between the two.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(!bool(stands_after_space(subject)),
			"The colon of one label closes against the name it binds.")
	}
	emit_byte(subject, ':')
	// A label states the name it binds and the colon behind it, thus the line it opens
	// closes on that colon.
	if bool(writes_canonical(subject.Flags.Values)) {
		place := int(written_load(subject.Counts.Values))
		aver.Always(subject.Form.Values[place-1] == ':',
			"A label closes on the colon that binds the name it states.")
	}
	if !bool(descend_part(subject, tree)) {
		close_line(subject, tree)
		return
	}
	// The tree names the label twice, thus the name it holds inside stands dropped.
	counts_dropped_type(subject, tree)
	// The tree names the label twice: once on the node and once as the name it binds, thus
	// the statement the label names stands behind that name.
	stand := advance_part(subject, tree)
	close_line(subject, tree)
	if bool(stand) {
		indent_deepen(subject, indent_load(subject.Counts.Values))
		print_statement(subject, tree)
		// The statement closed its line, thus every comment behind it takes a line of
		// its own rather than the end of a line already written.
		drain_trivia(subject, tree, advance(subject, tree))
	}
	if !bool(stand) {
		finish_parts(subject, tree)
	}
	ascend(subject)
}

// Writes one keyed element: the key, the colon, and the value behind it.
func print_key_value(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_key_value.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_key_value.tree")
	walked := depth_load(subject.Counts.Values)
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(node_kind(subject, tree) == ast.NODE_KEY_VALUE,
			"A pair the print writes stands on a pair of the tree.")
	}
	subject.Counts.Values[COUNT_PAIR] += 1
	held := key_width_load(subject.Counts.Values)
	opened_key := descend_part(subject, tree)
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(opened_key), "A pair the print writes opens on a key of the tree.")
	}
	if !bool(opened_key) {
		return
	}
	nest := nest_load(subject.Counts.Values)
	nest_store(subject.Counts.Values, NEST_MINIMUM)
	width := measure_expression(subject, tree)
	nest_store(subject.Counts.Values, nest)
	lines := subject.Counts.Values[COUNT_LINE]
	columns := int(take_line_width(subject.Counts.Values))
	print_expression(subject, tree)
	// A key that stood on one line carries it by the width its measure read.
	read := int(take_line_width(subject.Counts.Values))
	if subject.Counts.Values[COUNT_LINE] == lines {
		if read < WIDTH_MAXIMUM {
			aver.Always(read-columns == int(width),
				"A key of one pair spends the width its measure read.")
		}
	}
	if bool(writes_canonical(subject.Flags.Values)) {
		stood := int(written_load(subject.Counts.Values))
		aver.Always(subject.Form.Values[stood-1] != byte(SYMBOL_SPACE),
			"The colon of one pair closes against its key.")
	}
	emit_byte(subject, ':')
	if held == 0 {
		emit_space(subject)
	}
	if held > 0 {
		emit_column(subject, Part_Width(width), Column_Width(min(int(held), WIDTH_MAXIMUM)))
		tabs := int(indent_load(subject.Counts.Values)) * INDENT_WIDTH
		opened := tabs + max(int(held), int(width)) + 2
		if opened < WIDTH_MAXIMUM {
			if bool(writes_canonical(subject.Flags.Values)) {
				aver.Always(int(take_line_width(subject.Counts.Values)) == opened,
					"A value of one run opens at its column.")
			}
		}
	}
	valued := advance_part(subject, tree)
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(bool(valued), "A pair the print writes binds a value behind its colon.")
	}
	if bool(valued) {
		key_width_store(subject.Counts.Values, 0)
		print_expression(subject, tree)
		key_width_store(subject.Counts.Values, held)
	}
	finish_parts(subject, tree)
	ascend(subject)
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(depth_load(subject.Counts.Values) == walked,
			"A pair closes on the walk it opened.")
		aver.Always(key_width_load(subject.Counts.Values) == held,
			"A pair closes the run of keys it was handed.")
	}
}

// Counts one declaration of the kind it states. A count that holds every kind as one reads the
// same where the print writes one declaration twice and drops one of another kind, thus each
// kind answers for itself.
func counts_declaration(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "counts_declaration.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "counts_declaration.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_IMPORT:
		subject.Counts.Values[COUNT_IMPORT] += 1
	case ast.NODE_CONSTANT:
		subject.Counts.Values[COUNT_CONSTANT] += 1
	case ast.NODE_VARIABLE:
		subject.Counts.Values[COUNT_VARIABLE] += 1
	case ast.NODE_TYPE:
		subject.Counts.Values[COUNT_TYPE] += 1
	case ast.NODE_FUNCTION:
		subject.Counts.Values[COUNT_FUNCTION_DECLARATION] += 1
	}
}

// States that the print wrote every declaration of each kind the file states, once.
func closes_declared_kinds(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_declared_kinds.subject")
	Counts_Invariants(held, "closes_declared_kinds.held")
	counts := subject.Counts.Values
	aver.Always(counts[COUNT_IMPORT] == held[COUNT_IMPORT],
		"A print writes every import the file states, once.")
	aver.Always(counts[COUNT_CONSTANT] == held[COUNT_CONSTANT],
		"A print writes every constant the file states, once.")
	aver.Always(counts[COUNT_VARIABLE] == held[COUNT_VARIABLE],
		"A print writes every variable the file states, once.")
	aver.Always(counts[COUNT_TYPE] == held[COUNT_TYPE],
		"A print writes every type the file states, once.")
	aver.Always(counts[COUNT_FUNCTION_DECLARATION] == held[COUNT_FUNCTION_DECLARATION],
		"A print writes every function the file states, once.")
}

// Counts one clause of the kind it states. Each kind states a form of its own, thus a count that
// holds them all as one would read the same where the print wrote a loop for a check.
func counts_clause(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "counts_clause.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "counts_clause.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_IF:
		subject.Counts.Values[COUNT_IF] += 1
	case ast.NODE_FOR:
		subject.Counts.Values[COUNT_FOR] += 1
	case ast.NODE_SWITCH:
		subject.Counts.Values[COUNT_SWITCH] += 1
	case ast.NODE_TYPE_SWITCH:
		subject.Counts.Values[COUNT_TYPE_SWITCH] += 1
	case ast.NODE_SELECT:
		subject.Counts.Values[COUNT_SELECT] += 1
	}
}

// States that the print wrote every clause of each kind the file states, once.
func closes_clauses(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_clauses.subject")
	Counts_Invariants(held, "closes_clauses.held")
	counts := subject.Counts.Values
	aver.Always(counts[COUNT_IF] == held[COUNT_IF],
		"A print writes every check the file states, once.")
	aver.Always(counts[COUNT_FOR] == held[COUNT_FOR],
		"A print writes every loop the file states, once.")
	aver.Always(counts[COUNT_SWITCH] == held[COUNT_SWITCH],
		"A print writes every switch of values the file states, once.")
	aver.Always(counts[COUNT_TYPE_SWITCH] == held[COUNT_TYPE_SWITCH],
		"A print writes every switch of types the file states, once.")
	aver.Always(counts[COUNT_SELECT] == held[COUNT_SELECT],
		"A print writes every select the file states, once.")
}

// Counts one statement of the word it opens with. Each word closes a run of statements its own
// way, thus a count that holds them all as one would read the same where the print wrote a break
// for a continue.
func counts_word(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "counts_word.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "counts_word.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_RETURN:
		subject.Counts.Values[COUNT_RETURN] += 1
	case ast.NODE_GO:
		subject.Counts.Values[COUNT_GO] += 1
	case ast.NODE_DEFER:
		subject.Counts.Values[COUNT_DEFER] += 1
	case ast.NODE_BREAK:
		subject.Counts.Values[COUNT_BREAK] += 1
	case ast.NODE_CONTINUE:
		subject.Counts.Values[COUNT_CONTINUE] += 1
	case ast.NODE_GOTO:
		subject.Counts.Values[COUNT_GOTO] += 1
	case ast.NODE_FALLTHROUGH:
		subject.Counts.Values[COUNT_FALLTHROUGH] += 1
	}
}

// States that the print wrote every statement of each word the file states, once.
func closes_words(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_words.subject")
	Counts_Invariants(held, "closes_words.held")
	counts := subject.Counts.Values
	aver.Always(counts[COUNT_RETURN] == held[COUNT_RETURN],
		"A print writes every return the file states, once.")
	aver.Always(counts[COUNT_GO] == held[COUNT_GO],
		"A print writes every go statement the file states, once.")
	aver.Always(counts[COUNT_DEFER] == held[COUNT_DEFER],
		"A print writes every defer the file states, once.")
	aver.Always(counts[COUNT_BREAK] == held[COUNT_BREAK],
		"A print writes every break the file states, once.")
	aver.Always(counts[COUNT_CONTINUE] == held[COUNT_CONTINUE],
		"A print writes every continue the file states, once.")
	aver.Always(counts[COUNT_GOTO] == held[COUNT_GOTO],
		"A print writes every goto the file states, once.")
	aver.Always(counts[COUNT_FALLTHROUGH] == held[COUNT_FALLTHROUGH],
		"A print writes every fallthrough the file states, once.")
}

// Counts one binding of the kind it states. Each kind binds its values its own way, thus a count
// that holds them all as one would read the same where the print wrote a send for an assignment.
func counts_binding(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "counts_binding.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "counts_binding.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_ASSIGN:
		subject.Counts.Values[COUNT_ASSIGN] += 1
	case ast.NODE_DEFINE:
		subject.Counts.Values[COUNT_DEFINE] += 1
	case ast.NODE_OPERATION_ASSIGN:
		subject.Counts.Values[COUNT_OPERATION_ASSIGN] += 1
	case ast.NODE_SEND:
		subject.Counts.Values[COUNT_SEND] += 1
	case ast.NODE_INCREMENT:
		subject.Counts.Values[COUNT_INCREMENT] += 1
	case ast.NODE_DECREMENT:
		subject.Counts.Values[COUNT_DECREMENT] += 1
	case ast.NODE_EXPRESSION_STATEMENT:
		subject.Counts.Values[COUNT_EXPRESSION_STATEMENT] += 1
	case ast.NODE_DECLARATION_STATEMENT:
		subject.Counts.Values[COUNT_DECLARATION_STATEMENT] += 1
	}
}

// States that the print wrote every binding of each kind the file states, once.
func closes_bindings(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_bindings.subject")
	Counts_Invariants(held, "closes_bindings.held")
	closes_binding_counts(subject, held)
}

// States each count of the bindings one file holds against the count the print wrote.
func closes_binding_counts(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_binding_counts.subject")
	Counts_Invariants(held, "closes_binding_counts.held")
	counts := subject.Counts.Values
	aver.Always(counts[COUNT_ASSIGN] == held[COUNT_ASSIGN],
		"A print writes every assignment the file states, once.")
	aver.Always(counts[COUNT_DEFINE] == held[COUNT_DEFINE],
		"A print writes every binding of a name the file states, once.")
	aver.Always(counts[COUNT_OPERATION_ASSIGN] == held[COUNT_OPERATION_ASSIGN],
		"A print writes every assignment of an operation the file states, once.")
	aver.Always(counts[COUNT_SEND] == held[COUNT_SEND],
		"A print writes every send the file states, once.")
	aver.Always(counts[COUNT_INCREMENT] == held[COUNT_INCREMENT],
		"A print writes every step up the file states, once.")
	aver.Always(counts[COUNT_DECREMENT] == held[COUNT_DECREMENT],
		"A print writes every step down the file states, once.")
	aver.Always(counts[COUNT_EXPRESSION_STATEMENT] == held[COUNT_EXPRESSION_STATEMENT],
		"A print writes every statement of one expression the file states, once.")
	aver.Always(counts[COUNT_DECLARATION_STATEMENT] == held[COUNT_DECLARATION_STATEMENT],
		"A print writes every statement of one declaration the file states, once.")
}

// Writes the form of one statement, which every kind states its own way.
func print_statement_form(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_statement_form.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_statement_form.tree")
	// A statement opens a form of its own, thus every sign it states spaces the way a
	// statement spaces however deep the form that holds the statement stands.
	nest_store(subject.Counts.Values, NEST_MINIMUM)
	stood := written_load(subject.Counts.Values)
	switch node_kind(subject, tree) {
	case ast.NODE_ASSIGN, ast.NODE_DEFINE, ast.NODE_OPERATION_ASSIGN, ast.NODE_SEND:
		counts_binding(subject, tree)
		print_assignment(subject, tree)
	case ast.NODE_INCREMENT, ast.NODE_DECREMENT:
		counts_binding(subject, tree)
		print_step(subject, tree)
	case ast.NODE_RETURN, ast.NODE_GO, ast.NODE_DEFER, ast.NODE_BREAK,
		ast.NODE_CONTINUE, ast.NODE_GOTO, ast.NODE_FALLTHROUGH:
		counts_word(subject, tree)
		print_word_statement(subject, tree)
	case ast.NODE_IF, ast.NODE_FOR, ast.NODE_SWITCH, ast.NODE_TYPE_SWITCH,
		ast.NODE_SELECT:
		counts_clause(subject, tree)
		print_clause(subject, tree)
	case ast.NODE_CASE, ast.NODE_DEFAULT:
		print_case(subject, tree)
	case ast.NODE_LABEL:
		print_label(subject, tree)
	case ast.NODE_DECLARATION_STATEMENT:
		print_declaration_statement(subject, tree)
	case ast.NODE_EXPRESSION_STATEMENT:
		counts_binding(subject, tree)
		print_inner(subject, tree)
	default:
		print_expression(subject, tree)
	}
	// Every statement of the tree stands as a run of bytes a reader reads, thus a statement
	// that wrote none would leave the block it stands in reading as a block of fewer lines.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A statement the print writes spells a byte of its own.")
	}
}

// Writes one statement that opens with a word: the word the source states and the values behind it.
func print_word_statement(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_word_statement.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_word_statement.tree")
	// A statement of one word states that word and the value it binds where it binds one,
	// thus the walk stands on one of those statements where the print writes one. A return
	// stands apart from the six, because the classes of the tree hold the clauses between.
	if bool(writes_canonical(subject.Flags.Values)) {
		held := node_kind(subject, tree)
		if held != ast.NODE_RETURN {
			Word_Statement_Kind_Invariants(Word_Statement_Kind(held),
				"print_word_statement.worded")
		}
	}
	stood := written_load(subject.Counts.Values)
	emit_token(subject, tree)
	// A statement of one word spells that word, thus it writes bytes of its own however it
	// stands and whatever it binds.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(int(written_load(subject.Counts.Values)) > int(stood),
			"A statement of one word writes the word it states.")
	}
	open := descend_part(subject, tree)
	entered := open
	// A return that names no value hands back the values its signature names, thus the print
	// writes those names where the author wrote none: a reader sees what the function hands
	// back at the line it hands it back on. The names the print writes are the ones the
	// signature bound, and no inner name stands for them, because this dialect shadows none.
	if !bool(open) {
		emit_result_names(subject, tree)
	}
	written := Value_Written(false)
	for bool(open) {
		if bool(written) {
			emit_comma(subject)
		}
		// A value the author wrote on a line of its own keeps that line, one tab deeper
		// than the statement that states it.
		here := Boolean(false)
		if bool(written) {
			here = breaks_before(subject, tree)
		}
		open_continuation(subject, tree, here)
		if !bool(here) {
			emit_space(subject)
		}
		print_expression(subject, tree)
		written = true
		open = advance_part(subject, tree)
	}
	Value_Written_Invariants(written, "print_word_statement.written")
	if bool(entered) {
		finish_parts(subject, tree)
		ascend(subject)
	}
}

// Reports whether one class of node states a type that spells a form of its own, which is every
// class the print writes as a type rather than as the name of one.
func states_type_form(kind ast.Node_Kind) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_type_form.yes") }()
	ast.Node_Kind_Invariants(kind, "states_type_form.kind")
	if bool(states_literal(kind)) {
		return true
	}
	switch kind {
	case ast.NODE_POINTER_TYPE, ast.NODE_SLICE_TYPE, ast.NODE_ELLIPSIS,
		ast.NODE_CHANNEL_TYPE, ast.NODE_CHANNEL_SEND, ast.NODE_CHANNEL_RECEIVE,
		ast.NODE_ARRAY_TYPE, ast.NODE_MAP_TYPE, ast.NODE_FUNCTION_TYPE,
		ast.NODE_STRUCTURE_TYPE, ast.NODE_INTERFACE_TYPE, ast.NODE_GENERIC,
		ast.NODE_CALL, ast.NODE_INDEX, ast.NODE_PARENTHESIS, ast.NODE_UNARY:
		return true
	}
	return false
}

// Reports whether one class of node states a literal the source spells, which the print writes
// as the token it stands on rather than as a form of parts.
func states_literal(kind ast.Node_Kind) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_literal.yes") }()
	ast.Node_Kind_Invariants(kind, "states_literal.kind")
	switch kind {
	case ast.NODE_INTEGER, ast.NODE_FLOAT, ast.NODE_IMAGINARY, ast.NODE_CHARACTER,
		ast.NODE_STRING, ast.NODE_IDENTIFIER, ast.NODE_IMPORT_NAME,
		ast.NODE_CONSTANT_NAME:
		return true
	}
	return false
}

// Counts one form the print writes, where that form states parts of its own rather than a
// literal the source spells.
func counts_type(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "counts_type.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "counts_type.tree")
	kind := node_kind(subject, tree)
	if bool(states_literal(kind)) {
		return
	}
	if bool(states_type_form(kind)) {
		subject.Counts.Values[COUNT_TYPE_FORM] += 1
	}
	counts_type_kind(subject.Counts.Values, kind)
}

// Counts one literal the print writes, which every token of the form passes through.
func counts_literal(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "counts_literal.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "counts_literal.tree")
	if bool(states_literal(node_kind(subject, tree))) {
		subject.Counts.Values[COUNT_TYPE_FORM] += 1
	}
}

// Counts the types the print drops with the one the walk stands on, which is that type and every
// type standing inside it.
func counts_dropped_type(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "counts_dropped_type.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "counts_dropped_type.tree")
	start := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	stood := written_load(subject.Counts.Values)
	held := subject.Counts.Values[COUNT_TYPE_DROPPED]
	place := start
	for range ast.NODE_COUNT_MAXIMUM {
		if bool(states_type_form(ast.Syntax_Node_At(tree, place).Kind)) {
			subject.Counts.Values[COUNT_TYPE_DROPPED] += 1
		}
		if ast.Syntax_Node_At(tree, place).Kind == ast.NODE_CALL {
			subject.Counts.Values[COUNT_CALL_DROPPED] += 1
		}
		if ast.Syntax_Node_At(tree, place).Kind == ast.NODE_IDENTIFIER {
			subject.Counts.Values[COUNT_NAME_DROPPED] += 1
		}
		counts_type_kind(subject.Counts.Values, ast.Syntax_Node_At(tree, place).Kind)
		child := ast.Index(ast.Syntax_Node_At(tree, place).First_Child)
		if child != ast.INDEX_ABSENT {
			place = child
			continue
		}
		// The walk climbs out of the chain it stepped into until a node stands behind
		// it, and a climb back to the type it opened at read every type inside it.
		climbed := start
		for range DEPTH_MAXIMUM {
			if place == start {
				break
			}
			behind := ast.Index(ast.Syntax_Node_At(tree, place).Next)
			if behind != ast.INDEX_ABSENT {
				climbed = behind
				break
			}
			place = ast.Index(ast.Syntax_Node_At(tree, place).Parent)
		}
		place = climbed
		if place == start {
			break
		}
	}
	// The read of a dropped form reads the tree and never the storage, thus it writes no
	// byte, and a form of a kind the count names stands counted with the forms inside it.
	aver.Always(written_load(subject.Counts.Values) == stood,
		"A read of the forms one drop holds writes no byte of its own.")
	if bool(states_type_form(ast.Syntax_Node_At(tree, start).Kind)) {
		aver.Always(subject.Counts.Values[COUNT_TYPE_DROPPED] > held,
			"A drop of one form counts the form it drops.")
	}
}

// Reports whether the statement the walk stands on binds one error: an assignment of any run of
// names whose last name spells the error, which is the run the check behind it reads.
func assigns_error(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "assigns_error.yes") }()
	Printer_Handle_Invariants(subject, "assigns_error.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "assigns_error.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the error one statement binds writes no byte of its own.")
	}()
	kind := node_kind(subject, tree)
	if kind != ast.NODE_ASSIGN {
		if kind != ast.NODE_DEFINE {
			return false
		}
	}
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	first := int(ast.Syntax_Node_At(tree, index).Token)
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	named := Boolean(false)
	open := descend(subject, tree)
	for bool(open) {
		if bool(opens_bound(subject, tree)) {
			// The sign stands behind the last name of the run, thus the name the
			// check reads stands two tokens ahead of the value it binds.
			held_depth := depth_load(subject.Counts.Values)
			enter_leftmost(subject, tree)
			leftmost := subject.Nodes.Values[depth_load(subject.Counts.Values)]
			depth_store(subject.Counts.Values, held_depth)
			position := int(ast.Syntax_Node_At(tree, leftmost).Token) - 2
			if position < first {
				break
			}
			name := ast.Syntax_Token_At(tree, ast.Token_Index(position))
			if string(token.Text(token.Source(subject.Source.Values), name)) != "err" {
				break
			}
			named = Boolean(position == first)
			if !bool(named) {
				mark := ast.Syntax_Token_At(tree, ast.Token_Index(position-1))
				named = Boolean(mark.Kind == token.KIND_COMMA)
			}
			break
		}
		open = advance(subject, tree)
	}
	// The read walks the names the statement binds, thus it stands inside that statement
	// and never above the node it opened on.
	aver.Always(depth_load(subject.Counts.Values) >= depth,
		"A read of the run one statement binds stands inside it.")
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	return named
}

// Reports whether the statement the walk stands on states the simple check of one error: the
// words that read the error against nothing and the block behind them. An initializer, another
// branch, or a condition of any other spelling each state a check the empty line still opens.
func states_check(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "states_check.yes") }()
	Printer_Handle_Invariants(subject, "states_check.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "states_check.tree")
	stood := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == stood,
			"A read of the check one line holds writes no byte of its own.")
	}()
	if node_kind(subject, tree) != ast.NODE_IF {
		return false
	}
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	// Every token the check reads stands behind one that is no end token, thus the run holds
	// each of them.
	position := int(ast.Syntax_Node_At(tree, index).Token)
	// The read looks four tokens past the word the check opens with, thus that word stands
	// no nearer than four tokens to the end of the run.
	aver.Always(position+CHECK_TOKEN_SPAN <= ast.TOKEN_INDEX_MAXIMUM,
		"A check the read reads stands inside the run of tokens.")
	source := token.Source(subject.Source.Values)
	named := ast.Syntax_Token_At(tree, ast.Token_Index(position+1))
	if string(token.Text(source, named)) != "err" {
		return false
	}
	if ast.Syntax_Token_At(tree, ast.Token_Index(position+2)).Kind != token.KIND_NOT_EQUAL {
		return false
	}
	compared := ast.Syntax_Token_At(tree, ast.Token_Index(position+3))
	if string(token.Text(source, compared)) != "nil" {
		return false
	}
	if ast.Syntax_Token_At(tree, ast.Token_Index(position+4)).Kind != token.KIND_BRACE_LEFT {
		return false
	}
	return holds_two_parts(subject, tree)
}

// Reports whether the node the walk stands on holds three parts or more, over the trivia between
// them. A call holds its callee as one part, thus three parts state two arguments.
func holds_three_parts(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "holds_three_parts.yes") }()
	Printer_Handle_Invariants(subject, "holds_three_parts.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "holds_three_parts.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the three parts one form holds writes no byte.")
	}()
	return Boolean(count_parts(subject, tree) == PART_COUNT_THREE)
}

// Reports whether no part stands behind the one the walk stands on, over the trivia between them.
func stands_last(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "stands_last.yes") }()
	Printer_Handle_Invariants(subject, "stands_last.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "stands_last.tree")
	written := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == written,
			"A read of the last part one form holds writes no byte of its own.")
	}()
	depth := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[depth]
	next := advance_quiet(subject, tree)
	subject.Nodes.Values[depth] = held
	return !next
}

// States that the table of rules the program holds reads as one run: each rule opens behind the
// one ahead of it and inside the steps the program states. A rule that opened past the rule
// behind it would run the steps of another rule and close where that one closes.
func opens_program(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "opens_program.subject")
	held := 0
	for rule := range LAYOUT_RULE_MAXIMUM + 1 {
		entry := int(LAYOUT_ENTRIES[rule*2])*LAYOUT_OPERAND_BASE +
			int(LAYOUT_ENTRIES[rule*2+1])
		aver.Always(entry >= held,
			"A rule of the program opens behind the rule ahead of it.")
		aver.Always(entry < LAYOUT_STEP_COUNT,
			"A rule of the program opens on a step of it.")
		closes_rule(subject, Layout_Rule(rule))
		held = entry
	}
	aver.Always(len(LAYOUT_ENTRIES) == (LAYOUT_RULE_MAXIMUM+1)*2,
		"The table of rules states the two bytes each rule opens at.")
	aver.Always(len(LAYOUT_PROGRAM) == LAYOUT_STEP_COUNT*LAYOUT_STEP_SIZE,
		"The program states the bytes each step it holds spends.")
	opens_steps(subject)
	opens_rules(subject)
	opens_reach(subject)
	opens_words(subject)
}

// States that the keywords the print owes read as one table: each one spells its own text, and
// no two of them spell the same. A keyword that spelled another would write the wrong word for
// every declaration that owes it.
func opens_words(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "opens_words.subject")
	for word := range WORD_MAXIMUM + 1 {
		text := word_text(Word(word))
		for other := range WORD_MAXIMUM + 1 {
			if other == word {
				continue
			}
			aver.Always(text != word_text(Word(other)),
				"A keyword the print owes spells a text of its own.")
		}
	}
}

// States that every step of the program stands named: a rule opens at it, or a step ahead of it
// steps to it. A step nothing names states an action the print never runs.
func opens_reach(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "opens_reach.subject")
	var named [LAYOUT_STEP_COUNT]Boolean
	for rule := range LAYOUT_RULE_MAXIMUM + 1 {
		named[int(LAYOUT_ENTRIES[rule*2])*LAYOUT_OPERAND_BASE+
			int(LAYOUT_ENTRIES[rule*2+1])] = true
	}
	for step := range LAYOUT_STEP_COUNT {
		at := step * LAYOUT_STEP_SIZE
		op := Layout_Op(LAYOUT_PROGRAM[at])
		if op == LAYOUT_RETURN {
			continue
		}
		operand := int(LAYOUT_PROGRAM[at+1])*LAYOUT_OPERAND_BASE +
			int(LAYOUT_PROGRAM[at+2])
		// A jump names the step it lands on, a guard names the step it steps over, and
		// every other step names the step behind it.
		if op == LAYOUT_JUMP {
			named[operand] = true
			continue
		}
		if op == LAYOUT_JUMP_UNLESS {
			named[operand] = true
		}
		if op == LAYOUT_WHEN {
			named[step+2] = true
		}
		if op == LAYOUT_UNLESS {
			named[step+2] = true
		}
		if step+1 < LAYOUT_STEP_COUNT {
			named[step+1] = true
		}
	}
	for step := range LAYOUT_STEP_COUNT {
		aver.Always(bool(named[step]),
			"A step of the program stands named by a step or a rule.")
	}
}

// States that every rule of the program stands named: the file rule opens the run, and a call
// names each of the rules behind it. A rule no call names states steps the print never runs.
func opens_rules(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "opens_rules.subject")
	var called [LAYOUT_RULE_MAXIMUM + 1]Boolean
	for step := range LAYOUT_STEP_COUNT {
		at := step * LAYOUT_STEP_SIZE
		if Layout_Op(LAYOUT_PROGRAM[at]) != LAYOUT_CALL {
			continue
		}
		held := int(LAYOUT_PROGRAM[at+1])*LAYOUT_OPERAND_BASE +
			int(LAYOUT_PROGRAM[at+2])
		called[held] = true
	}
	// The print runs the file rule itself, and it runs the rules that write a declaration,
	// a body, a signature, a head and a run of results where a form of Go states one.
	called[RULE_FILE] = true
	called[RULE_DECLARATION] = true
	called[RULE_BLOCK] = true
	called[RULE_SIGNATURE] = true
	called[RULE_HEAD] = true
	called[RULE_RESULTS] = true
	for rule := range LAYOUT_RULE_MAXIMUM + 1 {
		aver.Always(bool(called[rule]),
			"A rule of the program stands named by a call or by the print.")
	}
}

// States that one rule of the program reads as one rule: it closes on a return of its own, and
// every jump it states lands on a step it holds. A rule that jumped into another would run the
// steps of that one and close where it closes.
func closes_rule(subject Printer_Handle, rule Layout_Rule) {
	Printer_Handle_Invariants(subject, "closes_rule.subject")
	Layout_Rule_Invariants(rule, "closes_rule.rule")
	opening := int(LAYOUT_ENTRIES[int(rule)*2])*LAYOUT_OPERAND_BASE +
		int(LAYOUT_ENTRIES[int(rule)*2+1])
	close := LAYOUT_STEP_COUNT
	if int(rule) < LAYOUT_RULE_MAXIMUM {
		close = int(LAYOUT_ENTRIES[(int(rule)+1)*2])*LAYOUT_OPERAND_BASE +
			int(LAYOUT_ENTRIES[(int(rule)+1)*2+1])
	}
	closes := Boolean(false)
	for step := opening; step < close; step++ {
		at := step * LAYOUT_STEP_SIZE
		op := Layout_Op(LAYOUT_PROGRAM[at])
		if op == LAYOUT_RETURN {
			closes = true
		}
		// A guard runs the step behind it or steps over that one, thus a rule that
		// guards its final step would step past the steps it holds.
		if op == LAYOUT_WHEN {
			aver.Always(step+2 < close,
				"A guard one rule states guards a step it holds.")
		}
		if op == LAYOUT_UNLESS {
			aver.Always(step+2 < close,
				"A guard one rule states against a test guards a step it holds.")
		}
		// A call the rule states names another rule, because a rule that called itself
		// would run its own steps until the budget closed the run.
		if op == LAYOUT_CALL {
			held := int(LAYOUT_PROGRAM[at+1])*LAYOUT_OPERAND_BASE +
				int(LAYOUT_PROGRAM[at+2])
			aver.Always(held != int(rule), "A call one rule states names another rule.")
		}
		if op != LAYOUT_JUMP {
			if op != LAYOUT_JUMP_UNLESS {
				continue
			}
		}
		target := int(LAYOUT_PROGRAM[at+1])*LAYOUT_OPERAND_BASE +
			int(LAYOUT_PROGRAM[at+2])
		aver.Always(target >= opening,
			"A jump one rule states lands on a step that rule opened at or behind it.")
		aver.Always(target <= close,
			"A jump one rule states lands inside the steps that rule holds.")
	}
	aver.Always(bool(closes), "A rule of the program closes on a return.")
}

// States that every step the program holds reads as one step: its action stands among the ones
// the machine runs, and the operand it carries stands inside the bound the print reads it at.
func opens_steps(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "opens_steps.subject")
	for step := range LAYOUT_STEP_COUNT {
		at := step * LAYOUT_STEP_SIZE
		aver.Always(int(LAYOUT_PROGRAM[at]) <= int(LAYOUT_OP_MAXIMUM),
			"A step of the program names an action the machine runs.")
		operand := int(LAYOUT_PROGRAM[at+1])*LAYOUT_OPERAND_BASE +
			int(LAYOUT_PROGRAM[at+2])
		aver.Always(operand <= LAYOUT_OPERAND_MAXIMUM,
			"A step of the program carries an operand the print reads.")
		// A call names a rule of the program and a jump names a step of it, thus the
		// steps the machine never ran state the same as the ones it did.
		if Layout_Op(LAYOUT_PROGRAM[at]) == LAYOUT_CALL {
			aver.Always(operand <= LAYOUT_RULE_MAXIMUM,
				"A call the program holds names a rule the program states.")
		}
		if Layout_Op(LAYOUT_PROGRAM[at]) == LAYOUT_JUMP {
			aver.Always(operand < LAYOUT_STEP_COUNT,
				"A jump the program holds names a step the program states.")
		}
		if Layout_Op(LAYOUT_PROGRAM[at]) == LAYOUT_JUMP_UNLESS {
			aver.Always(operand < LAYOUT_STEP_COUNT,
				"A jump one test guards names a step the program states.")
		}
		opens_operand(subject, Layout_Op(LAYOUT_PROGRAM[at]), Layout_Operand(operand))
	}
}

// States that one step of the program carries the operand its action reads: a test of a class
// names a class the tree states, and a step that writes a word names a word the print writes.
func opens_operand(subject Printer_Handle, op Layout_Op, operand Layout_Operand) {
	Printer_Handle_Invariants(subject, "opens_operand.subject")
	Layout_Op_Invariants(op, "opens_operand.op")
	Layout_Operand_Invariants(operand, "opens_operand.operand")
	if op == LAYOUT_IS {
		aver.Always(int(operand) <= int(ast.NODE_KIND_MAXIMUM),
			"A test of a class one step states names a class the tree states.")
	}
	if op == LAYOUT_WORD {
		aver.Always(int(operand) <= int(WORD_MAXIMUM),
			"A word one step writes names a word the print states.")
	}
	if op == LAYOUT_VALUE_WORD {
		aver.Always(int(operand) <= int(WORD_MAXIMUM),
			"A word one value declaration states names a word the print states.")
	}
	// A byte one step writes stands among the signs a reader reads, because the canonical
	// form states its lines in the bytes the source spells and never in the ones it holds.
	if op == LAYOUT_BYTE {
		aver.Always(int(operand) >= int(SYMBOL_SPACE),
			"A byte one step writes stands at the space or past it.")
		aver.Always(int(operand) < SYMBOL_DELETE,
			"A byte one step writes stands inside the signs a reader reads.")
	}
	// A list of a signature states the type parameters, the parameters, or the results.
	if op == LAYOUT_LIST_ITEM {
		aver.Always(int(operand) <= LIST_RESULTS,
			"A list one step opens names a run the signature states.")
	}
}

// Word_Text is the text one keyword the print owes spells.
type Word_Text string

// Word_Text_Invariants states every length one keyword the print writes spells.
func Word_Text_Invariants(value Word_Text, namespace aver.Namespace) {
	aver.Tree(value, namespace).
		Range_Int(len(value), WORD_TEXT_MINIMUM, WORD_TEXT_MAXIMUM).
		Ensure()
}

// WORD_TEXT_MINIMUM is the length of the shortest keyword the print writes, which is var.
const WORD_TEXT_MINIMUM = 3

// WORD_TEXT_MAXIMUM is the length of the longest keyword the print writes, which is interface.
const WORD_TEXT_MAXIMUM = 9

// Reads the text one keyword the print owes spells.
func word_text(word Word) (text Word_Text) {
	defer func() { Word_Text_Invariants(text, "word_text.text") }()
	Word_Invariants(word, "word_text.word")
	switch word {
	case WORD_IMPORT:
		return "import"
	case WORD_CONSTANT:
		return "const"
	case WORD_VARIABLE:
		return "var"
	case WORD_TYPE:
		return "type"
	case WORD_FUNCTION:
		return "func"
	case WORD_STRUCTURE:
		return "struct"
	case WORD_INTERFACE:
		return "interface"
	case WORD_MAP:
		return "map"
	case WORD_CHANNEL:
		return "chan"
	case WORD_RANGE:
		return "range"
	case WORD_ELSE:
		return "else"
	case WORD_DEFAULT:
		return "default"
	}
	return "package"
}

// Writes what stands between one value and the value or the name before it: the assignment sign
// opens the run and a comma holds the rest of it.
func emit_separator(subject Printer_Handle, first Boolean) {
	Printer_Handle_Invariants(subject, "emit_separator.subject")
	Boolean_Invariants(first, "emit_separator.first")
	// A run opens its values on the sign that binds them and parts the rest on a comma,
	// thus what it writes here spends the columns that sign and that comma state.
	stood := take_line_width(subject.Counts.Values)
	held := SEPARATOR_COMMA_WIDTH
	if bool(first) {
		held = SEPARATOR_SIGN_WIDTH
		if bool(flag_load(subject.Flags.Values, FLAG_SHORT)) {
			held = SEPARATOR_SHORT_WIDTH
		}
	}
	if bool(first) {
		emit_space(subject)
		if bool(flag_load(subject.Flags.Values, FLAG_SHORT)) {
			emit_byte(subject, ':')
		}
		emit_byte(subject, '=')
	}
	if !bool(first) {
		emit_comma(subject)
	}
	emit_space(subject)
	reads_separator(subject)
	// What parts two values of one run stands apart from the value behind it, thus the run
	// opens that value on the space this writing closed on.
	if !bool(flag_load(subject.Flags.Values, FLAG_FULL)) {
		aver.Always(bool(stands_after_space(subject)),
			"What parts two values of one run closes on a space.")
		// A line the read saturates states no width of its own, thus the columns it
		// holds state nothing of what the run wrote.
		if int(take_line_width(subject.Counts.Values)) < WIDTH_MAXIMUM {
			if bool(writes_canonical(subject.Flags.Values)) {
				spent := int(take_line_width(subject.Counts.Values)) - int(stood)
				aver.Always(spent == held,
					"What parts two values of one run spends its columns.")
			}
		}
	}
}

// Reads the columns one mark a type opens with spends: the star of a pointer, the brackets of a
// slice, the points of a variadic parameter, and the word and the arrow a channel states.
func prefix_width(prefix Prefix) (width Prefix_Width) {
	defer func() { Prefix_Width_Invariants(width, "prefix_width.width") }()
	Prefix_Invariants(prefix, "prefix_width.prefix")
	switch prefix {
	case PREFIX_SLICE:
		return 2
	case PREFIX_ELLIPSIS:
		return SPREAD_SIZE
	case PREFIX_CHANNEL:
		return Prefix_Width(len(word_text(WORD_CHANNEL)) + 1)
	case PREFIX_SEND:
		return Prefix_Width(len(word_text(WORD_CHANNEL)) + ARROW_SIZE + 1)
	case PREFIX_RECEIVE:
		return Prefix_Width(ARROW_SIZE + len(word_text(WORD_CHANNEL)) + 1)
	}
	return 1
}

// EMPTY_BODY_SIZE is how many bytes the body of no statement spends: the space that parts it
// from the header and the two braces that hold no statement.
const EMPTY_BODY_SIZE = 3



// Counts every node of the tree by the class it states, in one walk of the arena. Each law that
// states what the print wrote reads this tally, thus the laws cost one pass of the nodes rather
// than a pass for each of them.
func closes_tally(subject Printer_Handle, tree ast.Syntax_Tree_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_tally.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "closes_tally.tree")
	Counts_Invariants(held, "closes_tally.held")
	for place := range ast.NODE_COUNT_MAXIMUM {
		node := tree.Nodes.Values[place]
		if node == (ast.Node{}) {
			break
		}
		if bool(states_type_form(node.Kind)) {
			held[COUNT_TYPE_FORM] += 1
		}
		tally_kind(held, node.Kind)
		counts_type_kind(held, node.Kind)
		if node.Kind == ast.NODE_COMMENT {
			continue
		}
		if node.Kind == ast.NODE_BLANK {
			continue
		}
		if ast.Index(node.Parent) == ast.INDEX_ABSENT {
			continue
		}
		if tree.Nodes.Values[node.Parent].Kind == ast.NODE_BLOCK {
			held[COUNT_STATEMENT] += 1
		}
	}
}

// Counts one node of the tree in the slot its class states.
func tally_kind(held Counts, kind ast.Node_Kind) {
	Counts_Invariants(held, "tally_kind.held")
	ast.Node_Kind_Invariants(kind, "tally_kind.kind")
	switch kind {
	case ast.NODE_COMMENT:
		held[COUNT_NOTE] += 1
	case ast.NODE_FIELD:
		held[COUNT_FIELD] += 1
	case ast.NODE_PARAMETER:
		held[COUNT_PARAMETER] += 1
	case ast.NODE_RESULT:
		held[COUNT_RESULT] += 1
	case ast.NODE_CASE:
		held[COUNT_CASE] += 1
	case ast.NODE_DEFAULT:
		held[COUNT_DEFAULT] += 1
	case ast.NODE_LABEL:
		held[COUNT_LABEL] += 1
	case ast.NODE_FIELD_NAME:
		held[COUNT_FIELD_NAME] += 1
	case ast.NODE_PARAMETER_NAME:
		held[COUNT_PARAMETER_NAME] += 1
	case ast.NODE_RECEIVER:
		held[COUNT_RECEIVER] += 1
	case ast.NODE_IDENTIFIER:
		held[COUNT_NAME] += 1
	case ast.NODE_PARENTHESIS:
		held[COUNT_GROUP] += 1
	case ast.NODE_UNARY:
		held[COUNT_UNARY] += 1
	case ast.NODE_CALL:
		held[COUNT_CALL] += 1
	case ast.NODE_INDEX, ast.NODE_GENERIC:
		held[COUNT_INDEX] += 1
	case ast.NODE_IMPORT:
		held[COUNT_IMPORT] += 1
	case ast.NODE_CONSTANT:
		held[COUNT_CONSTANT] += 1
	case ast.NODE_VARIABLE:
		held[COUNT_VARIABLE] += 1
	case ast.NODE_TYPE:
		held[COUNT_TYPE] += 1
	case ast.NODE_FUNCTION:
		held[COUNT_FUNCTION_DECLARATION] += 1
	case ast.NODE_KEY_VALUE:
		held[COUNT_PAIR] += 1
	case ast.NODE_TERM:
		held[COUNT_TERM] += 1
	case ast.NODE_UNION:
		held[COUNT_UNION] += 1
	case ast.NODE_TYPE_PARAMETER:
		held[COUNT_TYPE_PARAMETER] += 1
	case ast.NODE_BLOCK:
		held[COUNT_BLOCK] += 1
	}
	tally_form(held, kind)
}

// Counts one node of the tree in the slot the form it states names: the forms of a kind the print
// never drops, the clauses, the words that close a run of statements, and the bindings.
func tally_form(held Counts, kind ast.Node_Kind) {
	Counts_Invariants(held, "tally_form.held")
	ast.Node_Kind_Invariants(kind, "tally_form.kind")
	switch kind {
	case ast.NODE_COMPOSITE, ast.NODE_SELECTOR, ast.NODE_ASSERTION,
		ast.NODE_SLICE_EXPRESSION, ast.NODE_FUNCTION_LITERAL, ast.NODE_RANGE,
		ast.NODE_BINARY:
		held[COUNT_FORM] += 1
	}
	switch kind {
	case ast.NODE_BINARY:
		held[COUNT_OPERATION] += 1
	case ast.NODE_SELECTOR:
		held[COUNT_SELECTOR] += 1
	case ast.NODE_ASSERTION:
		held[COUNT_ASSERTION] += 1
	case ast.NODE_COMPOSITE:
		held[COUNT_LITERAL] += 1
	case ast.NODE_SLICE_EXPRESSION:
		held[COUNT_SLICE] += 1
	case ast.NODE_FUNCTION_LITERAL:
		held[COUNT_FUNCTION] += 1
	case ast.NODE_RANGE:
		held[COUNT_RANGE] += 1
	case ast.NODE_IF:
		held[COUNT_IF] += 1
	case ast.NODE_FOR:
		held[COUNT_FOR] += 1
	case ast.NODE_SWITCH:
		held[COUNT_SWITCH] += 1
	case ast.NODE_TYPE_SWITCH:
		held[COUNT_TYPE_SWITCH] += 1
	case ast.NODE_SELECT:
		held[COUNT_SELECT] += 1
	}
	tally_word(held, kind)
}

// Counts one node of the tree in the slot the word or the binding it states names.
func tally_word(held Counts, kind ast.Node_Kind) {
	Counts_Invariants(held, "tally_word.held")
	ast.Node_Kind_Invariants(kind, "tally_word.kind")
	switch kind {
	case ast.NODE_RETURN:
		held[COUNT_RETURN] += 1
	case ast.NODE_GO:
		held[COUNT_GO] += 1
	case ast.NODE_DEFER:
		held[COUNT_DEFER] += 1
	case ast.NODE_BREAK:
		held[COUNT_BREAK] += 1
	case ast.NODE_CONTINUE:
		held[COUNT_CONTINUE] += 1
	case ast.NODE_GOTO:
		held[COUNT_GOTO] += 1
	case ast.NODE_FALLTHROUGH:
		held[COUNT_FALLTHROUGH] += 1
	case ast.NODE_ASSIGN:
		held[COUNT_ASSIGN] += 1
	case ast.NODE_DEFINE:
		held[COUNT_DEFINE] += 1
	case ast.NODE_OPERATION_ASSIGN:
		held[COUNT_OPERATION_ASSIGN] += 1
	case ast.NODE_SEND:
		held[COUNT_SEND] += 1
	case ast.NODE_INCREMENT:
		held[COUNT_INCREMENT] += 1
	case ast.NODE_DECREMENT:
		held[COUNT_DECREMENT] += 1
	case ast.NODE_EXPRESSION_STATEMENT:
		held[COUNT_EXPRESSION_STATEMENT] += 1
	case ast.NODE_DECLARATION_STATEMENT:
		held[COUNT_DECLARATION_STATEMENT] += 1
	}
}

// Counts one expression of the kind it states, for the kinds every print writes and never drops.
func counts_expression(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "counts_expression.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "counts_expression.tree")
	switch node_kind(subject, tree) {
	case ast.NODE_UNARY:
		subject.Counts.Values[COUNT_UNARY] += 1
	case ast.NODE_CALL:
		subject.Counts.Values[COUNT_CALL] += 1
	case ast.NODE_INDEX, ast.NODE_GENERIC:
		subject.Counts.Values[COUNT_INDEX] += 1
	}
}

// Counts one node in the slot the type form it states names. The print writes a type or drops it,
// and both roads read the form once, thus one slot answers for the two of them.
func counts_type_kind(counts Counts, kind ast.Node_Kind) {
	Counts_Invariants(counts, "counts_type_kind.counts")
	ast.Node_Kind_Invariants(kind, "counts_type_kind.kind")
	switch kind {
	case ast.NODE_POINTER_TYPE:
		counts[COUNT_POINTER] += 1
	case ast.NODE_SLICE_TYPE:
		counts[COUNT_SLICE_TYPE] += 1
	case ast.NODE_ELLIPSIS:
		counts[COUNT_ELLIPSIS] += 1
	case ast.NODE_CHANNEL_TYPE, ast.NODE_CHANNEL_SEND, ast.NODE_CHANNEL_RECEIVE:
		counts[COUNT_CHANNEL] += 1
	case ast.NODE_ARRAY_TYPE:
		counts[COUNT_ARRAY] += 1
	case ast.NODE_MAP_TYPE:
		counts[COUNT_MAP] += 1
	case ast.NODE_FUNCTION_TYPE:
		counts[COUNT_FUNCTION_TYPE] += 1
	case ast.NODE_STRUCTURE_TYPE:
		counts[COUNT_STRUCTURE] += 1
	case ast.NODE_INTERFACE_TYPE:
		counts[COUNT_INTERFACE] += 1
	}
}

// States that the print read every type of each form the file states, once. The sum of the forms
// reads the same where the print wrote a pointer for a slice, thus each form answers for itself.
func closes_type_kinds(subject Printer_Handle, held Counts) {
	Printer_Handle_Invariants(subject, "closes_type_kinds.subject")
	Counts_Invariants(held, "closes_type_kinds.held")
	counts := subject.Counts.Values
	aver.Always(counts[COUNT_POINTER] == held[COUNT_POINTER],
		"A print reads every pointer type the file states, once.")
	aver.Always(counts[COUNT_SLICE_TYPE] == held[COUNT_SLICE_TYPE],
		"A print reads every slice type the file states, once.")
	aver.Always(counts[COUNT_ELLIPSIS] == held[COUNT_ELLIPSIS],
		"A print reads every spread the file states, once.")
	aver.Always(counts[COUNT_CHANNEL] == held[COUNT_CHANNEL],
		"A print reads every channel type the file states, once.")
	aver.Always(counts[COUNT_ARRAY] == held[COUNT_ARRAY],
		"A print reads every array type the file states, once.")
	aver.Always(counts[COUNT_MAP] == held[COUNT_MAP],
		"A print reads every map type the file states, once.")
	aver.Always(counts[COUNT_FUNCTION_TYPE] == held[COUNT_FUNCTION_TYPE],
		"A print reads every function type the file states, once.")
	aver.Always(counts[COUNT_STRUCTURE] == held[COUNT_STRUCTURE],
		"A print reads every structure type the file states, once.")
	aver.Always(counts[COUNT_INTERFACE] == held[COUNT_INTERFACE],
		"A print reads every interface type the file states, once.")
}

// Writes one statement of a body and counts it. Trivia stands between the statements rather than
// as one of them, thus the count reads what the body states.
func print_block_statement(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "print_block_statement.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_block_statement.tree")
	if !bool(trivia_kind(subject, tree)) {
		subject.Counts.Values[COUNT_STATEMENT] =
			subject.Counts.Values[COUNT_STATEMENT] + 1
	}
	stood := written_load(subject.Counts.Values)
	print_statement(subject, tree)
	// Every statement a body holds spells bytes of its own, thus the form stands wider once
	// the statement closed than it stood when the statement opened.
	if bool(writes_canonical(subject.Flags.Values)) {
		aver.Always(written_load(subject.Counts.Values) > stood,
			"A statement one body holds writes bytes of its own.")
	}
}

// Reads how many parts the form the walk stands on holds, counting up to three.
func count_parts(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (count Part_Count) {
	defer func() { Part_Count_Invariants(count, "count_parts.count") }()
	Printer_Handle_Invariants(subject, "count_parts.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "count_parts.tree")
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	stood := written_load(subject.Counts.Values)
	count = PART_COUNT_NONE
	open := descend_quiet(subject, tree)
	for bool(open) {
		count = count + 1
		if count == PART_COUNT_THREE {
			break
		}
		open = advance_quiet(subject, tree)
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	// A count of the parts one form holds reads the tree and never the storage, thus the
	// form stands as wide once the count closed as it stood when the count opened.
	aver.Always(written_load(subject.Counts.Values) == stood,
		"A count of the parts one form holds writes no byte.")
	return count
}

// Reads the columns the print wrote past the form one measure opened at. One character stands in
// one column however many bytes it spans, and a tab stands in the columns a reader sees.
func measured_columns(subject Printer_Handle, held Counts) (width Line_Width) {
	defer func() { Line_Width_Invariants(width, "measured_columns.width") }()
	Printer_Handle_Invariants(subject, "measured_columns.subject")
	Counts_Invariants(held, "measured_columns.held")
	form := subject.Form.Values
	limit := int(written_load(subject.Counts.Values))
	// A measure reads the bytes the print wrote past the mark it opened at, thus that mark
	// stands at the end of the form or behind it.
	aver.Always(int(held[COUNT_WRITTEN]) <= limit,
		"A measure of the columns one run spends opens inside the form.")
	columns := 0
	for place := int(held[COUNT_WRITTEN]); place < limit; place++ {
		if form[place] == '\t' {
			columns = columns + TAB_WIDTH
			continue
		}
		if form[place]&CHARACTER_TAIL_MASK == CHARACTER_TAIL_MARK {
			continue
		}
		columns = columns + 1
	}
	return Line_Width(min(columns, WIDTH_MAXIMUM))
}

// Reports whether the empty line the walk stands on states nothing the form needs. A body opens
// at its first part and closes at its last one, thus a line standing against the bracket on
// either side states nothing, and so does one standing where a form spells no line of its own.
func drops_blank(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "drops_blank.yes") }()
	Printer_Handle_Invariants(subject, "drops_blank.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "drops_blank.tree")
	if bool(stands_against(subject, tree)) {
		return true
	}
	if bool(stands_before_check(subject, tree)) {
		return true
	}
	if bool(stands_behind_sign(subject, tree)) {
		return true
	}
	if bool(stands_before_brace(subject, tree)) {
		return true
	}
	if bool(stands_inside_head(subject, tree)) {
		return true
	}
	return stands_inside_clause(subject, tree)
}

// Writes one bound of a slice, or reads the bound the slice states nowhere. A bound the form
// behind it names states nothing of its own, thus the print reads the notes and the forms it
// holds and writes none of them.
func print_bound(subject Printer_Handle, tree ast.Syntax_Tree_Handle, owes Boolean) {
	Printer_Handle_Invariants(subject, "print_bound.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_bound.tree")
	Boolean_Invariants(owes, "print_bound.owes")
	if !bool(owes) {
		counts_dropped_type(subject, tree)
		take_form_notes(subject, tree)
		return
	}
	print_expression(subject, tree)
}

// Reports whether the head of the case the walk stands on reads on one line: the word, the values
// behind it, and the colon that closes them, inside the width a short line states. A note the
// author left inside that head names the line it stands on, thus a head that holds one keeps the
// lines the author wrote.
func fits_case(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "fits_case.yes") }()
	Printer_Handle_Invariants(subject, "fits_case.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "fits_case.tree")
	// The read measures what the tree states and hands back every byte a measure spent,
	// thus the form stands as wide once it closed as it stood when it opened.
	held_form := written_load(subject.Counts.Values)
	defer func() {
		aver.Always(written_load(subject.Counts.Values) == held_form,
			"A read of the width one case head spends writes no byte.")
	}()
	index := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	opening := int(ast.Syntax_Node_At(tree, index).Token)
	level := 0
	// The colon that closes the values stands outside every bracket a value opens.
	for place := opening; place < ast.TOKEN_INDEX_MAXIMUM; place = place + 1 {
		kind := ast.Syntax_Token_At(tree, ast.Token_Index(place)).Kind
		// A source the author left open states no colon at all, thus the read closes at
		// the end of the file and the case reads on no line of its own.
		if kind == token.KIND_END_OF_FILE {
			return false
		}
		if kind == token.KIND_COMMENT {
			return false
		}
		step := bracket_step(token.Scanned_Kind(kind))
		if step == BRACKET_OPENS {
			level = level + 1
		}
		if step == BRACKET_CLOSES {
			level = level - 1
		}
		if kind != token.KIND_COLON {
			continue
		}
		if level == 0 {
			break
		}
	}
	held := subject.Nodes.Values[depth_load(subject.Counts.Values)]
	depth := depth_load(subject.Counts.Values)
	// The word one case opens with and the space behind it stand ahead of every value, and
	// the colon closes them, thus the head writes four bytes no value states.
	width := 6
	values := 0
	open := descend_quiet(subject, tree)
	if bool(open) {
		if node_kind(subject, tree) == ast.NODE_EXPRESSION_STATEMENT {
			open = descend_quiet(subject, tree)
			for bool(open) {
				width = width + int(measure_expression(subject, tree)) + 2
				values = values + 1
				open = advance_quiet(subject, tree)
			}
		}
	}
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	if values == 0 {
		return false
	}
	// Each value spends the width it measures and the comma and the space behind it, thus a
	// head of one value stands past the word the case opens with and that value alone.
	aver.Always(width > values*2, "A case head spends the width of every value it names.")
	return Boolean(width-2 <= CASE_WIDTH_MAXIMUM)
}

// Reports whether the line the print stands on holds nothing but the bytes it opens at. A line
// opens at one byte for each depth the walk stands under, thus a line holding those alone stands
// ahead of every form it writes.
func opens_line(subject Printer_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_line.yes") }()
	Printer_Handle_Invariants(subject, "opens_line.subject")
	held := int(indent_load(subject.Counts.Values)) * INDENT_WIDTH
	return Boolean(int(take_line_width(subject.Counts.Values)) == held)
}

// ENTRY_NAME is the name one program enters at.
const ENTRY_NAME = "main"

// ENTRY_NAME_HELD is the name a package states for the same, which a caller of that package
// enters at.
const ENTRY_NAME_HELD = "Main"

// Reports whether the function the walk stands on binds the name one program enters at.
func names_entry(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "names_entry.yes") }()
	Printer_Handle_Invariants(subject, "names_entry.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "names_entry.tree")
	held := ast.Syntax_Node_At(tree, subject.Nodes.Values[depth_load(subject.Counts.Values)])
	if held.Kind != ast.NODE_FUNCTION {
		return false
	}
	// A note or an empty line the author left inside the head names nothing of the
	// declaration, thus the read steps over the trivia ahead of the name.
	name := ast.Index(held.First_Child)
	for range ast.NODE_COUNT_MAXIMUM {
		if name == ast.INDEX_ABSENT {
			return false
		}
		kind := ast.Syntax_Node_At(tree, name).Kind
		if kind != ast.NODE_COMMENT {
			if kind != ast.NODE_BLANK {
				break
			}
		}
		name = ast.Index(ast.Syntax_Node_At(tree, name).Next)
	}
	named := ast.Syntax_Node_At(tree, name)
	// A method states the receiver it binds ahead of the name, thus a function of the name
	// one program enters at states that name as its first part.
	if named.Kind != ast.NODE_IDENTIFIER {
		return false
	}
	source := token.Source(subject.Source.Values)
	text := token.Text(source, ast.Syntax_Token_At(tree, named.Token))
	if string(text) == ENTRY_NAME {
		return true
	}
	return Boolean(string(text) == ENTRY_NAME_HELD)
}

// Reads the function one program enters at and the place the file states it. The first function
// the author wrote opens the run that function moves to, thus the print writes the entry there
// and writes every other declaration where the author wrote it. A file that states the entry
// first already, or states none, names no place to move it to.
func take_main(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "take_main.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "take_main.tree")
	counts := subject.Counts.Values
	counts[COUNT_MAIN] = Count(ast.INDEX_ABSENT)
	counts[COUNT_MAIN_PLACE] = Count(ast.INDEX_ABSENT)
	depth := depth_load(counts)
	file := subject.Nodes.Values[depth]
	child := ast.Index(ast.Syntax_Node_At(tree, file).First_Child)
	start := ast.INDEX_ABSENT
	for range ast.NODE_COUNT_MAXIMUM {
		if child == ast.INDEX_ABSENT {
			break
		}
		held := ast.Syntax_Node_At(tree, child)
		if start == ast.INDEX_ABSENT {
			start = child
		}
		if held.Kind == ast.NODE_COMMENT {
			child = ast.Index(held.Next)
			continue
		}
		subject.Nodes.Values[depth+1] = child
		depth_store(counts, depth+1)
		entry := names_entry(subject, tree)
		opens := opens_function(subject, tree)
		depth_store(counts, depth)
		if bool(opens) {
			if counts[COUNT_MAIN_PLACE] == Count(ast.INDEX_ABSENT) {
				counts[COUNT_MAIN_PLACE] = Count(start)
			}
		}
		if bool(entry) {
			if counts[COUNT_MAIN] == Count(ast.INDEX_ABSENT) {
				counts[COUNT_MAIN] = Count(start)
			}
		}
		start = ast.INDEX_ABSENT
		child = ast.Index(held.Next)
	}
	closes_main(subject)
}

// States what the read of the entry hands back. A file that states the entry where the print
// writes it moves nothing, thus the slot that names the entry stands empty.
func closes_main(subject Printer_Handle) {
	Printer_Handle_Invariants(subject, "closes_main.subject")
	if subject.Counts.Values[COUNT_MAIN] != subject.Counts.Values[COUNT_MAIN_PLACE] {
		return
	}
	subject.Counts.Values[COUNT_MAIN] = Count(ast.INDEX_ABSENT)
}

// Reports whether the declaration the walk stands on stands ahead of every other function the
// file declares rather than where the author wrote it: the function one program enters at, or a
// note that opens it. The place the first function opens carries both, thus the walk writes
// neither where it finds them.
func moves_first(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "moves_first.yes") }()
	Printer_Handle_Invariants(subject, "moves_first.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "moves_first.tree")
	if subject.Counts.Values[COUNT_MAIN] == Count(ast.INDEX_ABSENT) {
		return false
	}
	depth := depth_load(subject.Counts.Values)
	stand := subject.Nodes.Values[depth]
	one := stand
	for range ast.NODE_COUNT_MAXIMUM {
		if ast.Syntax_Node_At(tree, one).Kind != ast.NODE_COMMENT {
			break
		}
		one = ast.Index(ast.Syntax_Node_At(tree, one).Next)
		if one == ast.INDEX_ABSENT {
			return false
		}
	}
	subject.Nodes.Values[depth] = one
	yes = names_entry(subject, tree)
	subject.Nodes.Values[depth] = stand
	return yes
}

// Writes the run the function one program enters at stands in, where the walk stands on the place
// the first function of the file opens. It reports whether it wrote one.
func print_main_run(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (wrote Boolean) {
	defer func() { Boolean_Invariants(wrote, "print_main_run.wrote") }()
	Printer_Handle_Invariants(subject, "print_main_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "print_main_run.tree")
	counts := subject.Counts.Values
	if counts[COUNT_MAIN] == Count(ast.INDEX_ABSENT) {
		return false
	}
	depth := depth_load(counts)
	if subject.Nodes.Values[depth] != ast.Index(counts[COUNT_MAIN_PLACE]) {
		return false
	}
	// The print writes the entry once, thus the place it wrote it at names no place behind
	// it. The slot that names the entry stands as it stood, because the walk still meets the
	// run where the author wrote it and writes none of it there.
	counts[COUNT_MAIN_PLACE] = Count(ast.INDEX_ABSENT)
	stand := subject.Nodes.Values[depth]
	stood := written_load(counts)
	write_main_run(subject, tree)
	subject.Nodes.Values[depth] = stand
	// The run holds the function one program enters at, thus the form holds the bytes of it.
	aver.Always(written_load(counts) > stood,
		"A run that moves the entry of one program writes the bytes of it.")
	return true
}

// Writes each declaration of the run the entry stands in, the notes above it and the function
// itself, the way the walk over the file writes the declarations it meets.
func write_main_run(subject Printer_Handle, tree ast.Syntax_Tree_Handle) {
	Printer_Handle_Invariants(subject, "write_main_run.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "write_main_run.tree")
	counts := subject.Counts.Values
	depth := depth_load(counts)
	node := ast.Index(counts[COUNT_MAIN])
	commented := Boolean(false)
	for range ast.NODE_COUNT_MAXIMUM {
		if node == ast.INDEX_ABSENT {
			return
		}
		subject.Nodes.Values[depth] = node
		kind := node_kind(subject, tree)
		open_declaration(subject, tree, commented, false)
		if !bool(commented) {
			declared_store(counts, Declaration_Place(written_load(counts)))
		}
		print_declaration(subject, tree)
		// The run wrote one declaration of the file here, thus the count of them reads
		// this one the way it reads the ones the file walk wrote.
		if !bool(trivia_kind(subject, tree)) {
			counts[COUNT_DECLARED_COUNT] = counts[COUNT_DECLARED_COUNT] + 1
			counts_declaration(subject, tree)
		}
		commented = Boolean(kind == ast.NODE_COMMENT)
		if !bool(commented) {
			return
		}
		node = ast.Index(ast.Syntax_Node_At(tree, node).Next)
	}
}

// Writes the entry of one program where the place it moves to opens, and hands back the run the
// declaration behind it stands apart from.
func open_main(
	subject Printer_Handle, tree ast.Syntax_Tree_Handle, spread Boolean,
) (held Boolean) {
	defer func() { Boolean_Invariants(held, "open_main.held") }()
	Printer_Handle_Invariants(subject, "open_main.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "open_main.tree")
	Boolean_Invariants(spread, "open_main.spread")
	if !bool(print_main_run(subject, tree)) {
		return spread
	}
	return open_spread(subject, spread)
}

// Reports whether the declaration the walk stands on writes a function of the file where it
// stands: a function the print leaves where the author wrote it, or a type that carries the
// function stating its invariants. The entry of one program stands ahead of every function the
// file writes, thus the first such declaration opens the place it moves to.
func opens_function(subject Printer_Handle, tree ast.Syntax_Tree_Handle) (yes Boolean) {
	defer func() { Boolean_Invariants(yes, "opens_function.yes") }()
	Printer_Handle_Invariants(subject, "opens_function.subject")
	ast.Syntax_Tree_Handle_Invariants(tree, "opens_function.tree")
	kind := node_kind(subject, tree)
	if kind == ast.NODE_FUNCTION {
		return !states_own_type(subject, tree)
	}
	if kind != ast.NODE_TYPE {
		return false
	}
	// The read of the run one type carries steps the walk onto that run, thus the read
	// stands the walk back where it found it.
	depth := depth_load(subject.Counts.Values)
	held := subject.Nodes.Values[depth]
	found := take_invariant_run(subject, tree)
	depth_store(subject.Counts.Values, depth)
	subject.Nodes.Values[depth] = held
	return found
}
