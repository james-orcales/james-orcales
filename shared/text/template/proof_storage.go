package template

import "local/james-orcales/shared/sim/aver/default"

// Fields_Proof_Stored separates traversal proof from transient slice headers.
type Fields_Proof_Stored interface{}

// Fields_Proof_Stored_Invariants fixes borrowed-field representation.
func Fields_Proof_Stored_Invariants(value Fields_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Fields_Storage_Value)
	aver.Always(valid == (value != nil), "Fields have expected storage type.")
}

// Evaluation_Frame_Delivery_Pointer carries one caller-owned suspended frame.
type Evaluation_Frame_Delivery_Pointer interface{}

// Evaluation_Frame_Delivery_Pointer_Invariants fixes delivery-frame representation.
func Evaluation_Frame_Delivery_Pointer_Invariants(
	value Evaluation_Frame_Delivery_Pointer, _ aver.Namespace,
) {
	_, valid := value.(Evaluation_Frame_Pointer)
	aver.Always(valid == (value != nil), "Evaluation delivery frame has expected pointer type.")
}

// Text_Span_Start_Proof_Stored separates literal proof from transient boundaries.
type Text_Span_Start_Proof_Stored interface{}

// Text_Span_Start_Proof_Stored_Invariants fixes literal-start representation.
func Text_Span_Start_Proof_Stored_Invariants(value Text_Span_Start_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Text_Span_Stored_Start)
	aver.Always(valid == (value != nil), "Text span start has expected storage type.")
}

// Text_Span_End_Proof_Stored separates literal proof from transient boundaries.
type Text_Span_End_Proof_Stored interface{}

// Text_Span_End_Proof_Stored_Invariants fixes literal-end representation.
func Text_Span_End_Proof_Stored_Invariants(value Text_Span_End_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Text_Span_Stored_End)
	aver.Always(valid == (value != nil), "Text span end has expected storage type.")
}

// Parse_Failure_Status_Proof_Stored separates refusal proof from transient statuses.
type Parse_Failure_Status_Proof_Stored interface{}

// Parse_Failure_Status_Proof_Stored_Invariants fixes parser-refusal representation.
func Parse_Failure_Status_Proof_Stored_Invariants(
	value Parse_Failure_Status_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Parse_Failure_Status_Validated_Value)
	aver.Always(valid == (value != nil), "Parse failure status has expected storage type.")
}

// Classified_Node_Kind_Proof_Stored separates classification proof from transient kinds.
type Classified_Node_Kind_Proof_Stored interface{}

// Classified_Node_Kind_Proof_Stored_Invariants fixes node-kind representation.
func Classified_Node_Kind_Proof_Stored_Invariants(
	value Classified_Node_Kind_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Classified_Node_Kind_Value)
	aver.Always(valid == (value != nil), "Classified node kind has expected storage type.")
}

// Generated_Start_Proof_Stored separates output proof from transient boundaries.
type Generated_Start_Proof_Stored interface{}

// Generated_Start_Proof_Stored_Invariants fixes generated-start representation.
func Generated_Start_Proof_Stored_Invariants(value Generated_Start_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Generated_Start_Validated_Value)
	aver.Always(valid == (value != nil), "Generated start has expected storage type.")
}

// Formatted_Integer_Text_Proof_Stored separates conversion proof from transient text.
type Formatted_Integer_Text_Proof_Stored interface{}

// Formatted_Integer_Text_Proof_Stored_Invariants fixes formatted-text representation.
func Formatted_Integer_Text_Proof_Stored_Invariants(
	value Formatted_Integer_Text_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Formatted_Integer_Text_Validated_Value)
	aver.Always(valid == (value != nil), "Formatted integer text has expected storage type.")
}

// Variable_Chain_Proof_Stored separates lexer proof from transient source slices.
type Variable_Chain_Proof_Stored interface{}

// Variable_Chain_Proof_Stored_Invariants fixes variable-chain representation.
func Variable_Chain_Proof_Stored_Invariants(value Variable_Chain_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Variable_Chain_Validated_Value)
	aver.Always(valid == (value != nil), "Variable chain has expected storage type.")
}

// Parse_Function_Name_Proof_Stored separates policy proof from transient source slices.
type Parse_Function_Name_Proof_Stored interface{}

// Parse_Function_Name_Proof_Stored_Invariants fixes parser-name representation.
func Parse_Function_Name_Proof_Stored_Invariants(
	value Parse_Function_Name_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Parse_Function_Name_Validated_Value)
	aver.Always(valid == (value != nil), "Parse function name has expected storage type.")
}

// Function_Name_Proof_Stored separates callback proof from transient source slices.
type Function_Name_Proof_Stored interface{}

// Function_Name_Proof_Stored_Invariants fixes callback-name representation.
func Function_Name_Proof_Stored_Invariants(value Function_Name_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Function_Name_Validated_Value)
	aver.Always(valid == (value != nil), "Function name has expected storage type.")
}

// Node_Reference_Proof_Stored separates location proof from transient arena slots.
type Node_Reference_Proof_Stored interface{}

// Node_Reference_Proof_Stored_Invariants fixes location-reference representation.
func Node_Reference_Proof_Stored_Invariants(value Node_Reference_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Node_Reference)
	aver.Always(valid == (value != nil), "Node reference has expected storage type.")
}

// Node_Allocation_Proof_Stored separates allocation proof from transient node fields.
type Node_Allocation_Proof_Stored interface{}

// Node_Allocation_Proof_Stored_Invariants fixes allocation-record representation.
func Node_Allocation_Proof_Stored_Invariants(
	value Node_Allocation_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Node)
	aver.Always(valid == (value != nil), "Node allocation has expected storage type.")
}

// Execution_Failure_Position_Proof_Stored separates diagnostic proof from transient positions.
type Execution_Failure_Position_Proof_Stored interface{}

// Execution_Failure_Position_Proof_Stored_Invariants fixes diagnostic position representation.
func Execution_Failure_Position_Proof_Stored_Invariants(
	value Execution_Failure_Position_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Execution_Failure_Position_Value)
	aver.Always(
		valid == (value != nil), "Execution failure position has expected storage type.",
	)
}

// Action_Scan_End_Proof_Stored separates scan proof from transient boundaries.
type Action_Scan_End_Proof_Stored interface{}

// Action_Scan_End_Proof_Stored_Invariants fixes scan boundary representation.
func Action_Scan_End_Proof_Stored_Invariants(
	value Action_Scan_End_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Action_Scan_End_Validated_Value)
	aver.Always(valid == (value != nil), "Action scan end has expected storage type.")
}

// Classified_Token_Kind_Proof_Stored separates classification proof from transient kinds.
type Classified_Token_Kind_Proof_Stored interface{}

// Classified_Token_Kind_Proof_Stored_Invariants fixes token-kind representation.
func Classified_Token_Kind_Proof_Stored_Invariants(
	value Classified_Token_Kind_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Classified_Token_Kind_Value)
	aver.Always(valid == (value != nil), "Classified token kind has expected storage type.")
}

// Number_Text_Proof_Stored separates numeric proof from transient source slices.
type Number_Text_Proof_Stored interface{}

// Number_Text_Proof_Stored_Invariants fixes numeric-text representation.
func Number_Text_Proof_Stored_Invariants(value Number_Text_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Number_Text_Validated_Value)
	aver.Always(valid == (value != nil), "Number text has expected storage type.")
}

// Failure_Status_Proof_Stored separates refusal proof from transient statuses.
type Failure_Status_Proof_Stored interface{}

// Failure_Status_Proof_Stored_Invariants fixes execution refusal representation.
func Failure_Status_Proof_Stored_Invariants(value Failure_Status_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Failure_Status_Validated_Value)
	aver.Always(valid == (value != nil), "Failure status has expected storage type.")
}

// Parenthesis_Depth_Proof_Stored separates parser proof from transient depth.
type Parenthesis_Depth_Proof_Stored interface{}

// Parenthesis_Depth_Proof_Stored_Invariants fixes parenthesis-depth representation.
func Parenthesis_Depth_Proof_Stored_Invariants(
	value Parenthesis_Depth_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Parenthesis_Depth_Tracked_Value)
	aver.Always(valid == (value != nil), "Parenthesis depth has expected storage type.")
}

// Pipeline_Boundary_Proof_Stored separates token proof from transient boundaries.
type Pipeline_Boundary_Proof_Stored interface{}

// Pipeline_Boundary_Proof_Stored_Invariants fixes pipeline-boundary representation.
func Pipeline_Boundary_Proof_Stored_Invariants(
	value Pipeline_Boundary_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Pipeline_Boundary_Validated_Value)
	aver.Always(valid == (value != nil), "Pipeline boundary has expected storage type.")
}

// Parsed_Float_Proof_Stored separates conversion proof from transient encodings.
type Parsed_Float_Proof_Stored interface{}

// Parsed_Float_Proof_Stored_Invariants fixes binary64 representation.
func Parsed_Float_Proof_Stored_Invariants(value Parsed_Float_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Parsed_Float_Validated_Value)
	aver.Always(valid == (value != nil), "Parsed float has expected storage type.")
}

// Pipeline_Root_Proof_Stored separates root proof shape from transient references.
type Pipeline_Root_Proof_Stored interface{}

// Pipeline_Root_Proof_Stored_Invariants accepts absent output or one typed reference.
func Pipeline_Root_Proof_Stored_Invariants(value Pipeline_Root_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Pipeline_Root_Validated_Value)
	aver.Always(valid == (value != nil), "Pipeline root has expected storage type.")
}

// Pipeline_Term_Proof_Stored separates term proof shape from transient references.
type Pipeline_Term_Proof_Stored interface{}

// Pipeline_Term_Proof_Stored_Invariants accepts absent output or one typed reference.
func Pipeline_Term_Proof_Stored_Invariants(value Pipeline_Term_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Pipeline_Term_Validated_Value)
	aver.Always(valid == (value != nil), "Pipeline term has expected storage type.")
}

// Template_Reference_Proof_Stored separates definition proof shape from transient references.
type Template_Reference_Proof_Stored interface{}

// Template_Reference_Proof_Stored_Invariants accepts absent output or one typed reference.
func Template_Reference_Proof_Stored_Invariants(
	value Template_Reference_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Template_Reference_Validated_Value)
	aver.Always(valid == (value != nil), "Template reference has expected storage type.")
}

// Declaration_Reference_Proof_Stored separates binding proof shape from transient references.
type Declaration_Reference_Proof_Stored interface{}

// Declaration_Reference_Proof_Stored_Invariants accepts absent output or one typed reference.
func Declaration_Reference_Proof_Stored_Invariants(
	value Declaration_Reference_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Declaration_Reference_Validated_Value)
	aver.Always(valid == (value != nil), "Declaration reference has expected storage type.")
}

// Argument_Count_Proof_Stored separates callback proof shape from transient progress.
type Argument_Count_Proof_Stored interface{}

// Argument_Count_Proof_Stored_Invariants accepts absent output or one typed count.
func Argument_Count_Proof_Stored_Invariants(
	value Argument_Count_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Argument_Count_Tracked_Value)
	aver.Always(valid == (value != nil), "Argument count has expected storage type.")
}

// Pipeline_Declarations_Fields carries raw parser declaration scratch.
type Pipeline_Declarations_Fields struct {
	// First retains the required declaration token.
	First Pipeline_First_Declaration_Storage
	// Second retains an optional range value declaration.
	Second Pipeline_Second_Declaration_Storage
}

// Pipeline_Declarations_Fields_Invariants bounds hostile declaration scratch.
func Pipeline_Declarations_Fields_Invariants(
	value Pipeline_Declarations_Fields, namespace aver.Namespace,
) {
	Pipeline_First_Declaration_Storage_Invariants(value.First, namespace)
	Pipeline_Second_Declaration_Storage_Invariants(value.Second, namespace)
}

// Pipeline_First_Declaration_Storage owns one required scratch token.
type Pipeline_First_Declaration_Storage struct {
	// Value prevents a stale scalar proof from escaping parser storage.
	Value Pipeline_First_Declaration_Stored
}

// Pipeline_First_Declaration_Storage_Invariants fixes first-token ownership.
func Pipeline_First_Declaration_Storage_Invariants(
	value Pipeline_First_Declaration_Storage, namespace aver.Namespace,
) {
	Pipeline_First_Declaration_Stored_Invariants(value.Value, namespace)
}

// Pipeline_First_Declaration_Stored separates scratch shape from the required token.
type Pipeline_First_Declaration_Stored interface{}

// Pipeline_First_Declaration_Stored_Invariants fixes first-token representation.
func Pipeline_First_Declaration_Stored_Invariants(
	value Pipeline_First_Declaration_Stored, _ aver.Namespace,
) {
	_, valid := value.(Pipeline_First_Declaration)
	aver.Always(valid == (value != nil), "First declaration has expected storage type.")
}

// Pipeline_Second_Declaration_Storage owns one optional scratch token.
type Pipeline_Second_Declaration_Storage struct {
	// Value prevents a stale scalar proof from escaping parser storage.
	Value Pipeline_Second_Declaration_Stored
}

// Pipeline_Second_Declaration_Storage_Invariants fixes second-token ownership.
func Pipeline_Second_Declaration_Storage_Invariants(
	value Pipeline_Second_Declaration_Storage, namespace aver.Namespace,
) {
	Pipeline_Second_Declaration_Stored_Invariants(value.Value, namespace)
}

// Pipeline_Second_Declaration_Stored separates scratch shape from an optional token.
type Pipeline_Second_Declaration_Stored interface{}

// Pipeline_Second_Declaration_Stored_Invariants fixes second-token representation.
func Pipeline_Second_Declaration_Stored_Invariants(
	value Pipeline_Second_Declaration_Stored, _ aver.Namespace,
) {
	_, valid := value.(Pipeline_Second_Declaration)
	aver.Always(valid == (value != nil), "Second declaration has expected storage type.")
}

// Declaration_References_Fields carries raw evaluator binding references.
type Declaration_References_Fields struct {
	// First retains the range index identity.
	First Declaration_First_Reference_Storage
	// Second retains the range value identity.
	Second Declaration_Second_Reference_Storage
}

// Declaration_References_Fields_Invariants bounds hostile binding references.
func Declaration_References_Fields_Invariants(
	value Declaration_References_Fields, namespace aver.Namespace,
) {
	Declaration_First_Reference_Storage_Invariants(value.First, namespace)
	Declaration_Second_Reference_Storage_Invariants(value.Second, namespace)
}

// Declaration_First_Reference_Storage owns one optional index binding.
type Declaration_First_Reference_Storage struct {
	// Value prevents stale reference proof from escaping evaluator storage.
	Value Declaration_First_Reference_Stored
}

// Declaration_First_Reference_Storage_Invariants fixes index-binding ownership.
func Declaration_First_Reference_Storage_Invariants(
	value Declaration_First_Reference_Storage, namespace aver.Namespace,
) {
	Declaration_First_Reference_Stored_Invariants(value.Value, namespace)
}

// Declaration_First_Reference_Stored separates binding shape from an optional index.
type Declaration_First_Reference_Stored interface{}

// Declaration_First_Reference_Stored_Invariants fixes index-binding representation.
func Declaration_First_Reference_Stored_Invariants(
	value Declaration_First_Reference_Stored, _ aver.Namespace,
) {
	_, valid := value.(Declaration_First_Reference)
	aver.Always(
		valid == (value != nil), "First declaration reference has expected storage type.",
	)
}

// Declaration_Second_Reference_Storage owns one optional value binding.
type Declaration_Second_Reference_Storage struct {
	// Value prevents stale reference proof from escaping evaluator storage.
	Value Declaration_Second_Reference_Stored
}

// Declaration_Second_Reference_Storage_Invariants fixes value-binding ownership.
func Declaration_Second_Reference_Storage_Invariants(
	value Declaration_Second_Reference_Storage, namespace aver.Namespace,
) {
	Declaration_Second_Reference_Stored_Invariants(value.Value, namespace)
}

// Declaration_Second_Reference_Stored separates binding shape from an optional value.
type Declaration_Second_Reference_Stored interface{}

// Declaration_Second_Reference_Stored_Invariants fixes value-binding representation.
func Declaration_Second_Reference_Stored_Invariants(
	value Declaration_Second_Reference_Stored, _ aver.Namespace,
) {
	_, valid := value.(Declaration_Second_Reference)
	aver.Always(
		valid == (value != nil), "Second declaration reference has expected storage type.",
	)
}

// Identifier_Search_Fields carries raw lexer search boundaries.
type Identifier_Search_Fields struct {
	// Position retains candidate character start.
	Position Identifier_Search_Position_Storage
	// End retains action content termination.
	End Identifier_Search_End_Storage
}

// Identifier_Search_Fields_Invariants bounds hostile lexer search boundaries.
func Identifier_Search_Fields_Invariants(
	value Identifier_Search_Fields, namespace aver.Namespace,
) {
	Identifier_Search_Position_Storage_Invariants(value.Position, namespace)
	Identifier_Search_End_Storage_Invariants(value.End, namespace)
}

// Identifier_Search_Position_Storage owns transient lexer progress.
type Identifier_Search_Position_Storage struct {
	// Value prevents a stale scalar proof from escaping lexer storage.
	Value Identifier_Search_Position_Stored
}

// Identifier_Search_Position_Storage_Invariants fixes progress ownership.
func Identifier_Search_Position_Storage_Invariants(
	value Identifier_Search_Position_Storage, namespace aver.Namespace,
) {
	Identifier_Search_Position_Stored_Invariants(value.Value, namespace)
}

// Identifier_Search_Position_Stored separates search shape from transient progress.
type Identifier_Search_Position_Stored interface{}

// Identifier_Search_Position_Stored_Invariants fixes progress representation.
func Identifier_Search_Position_Stored_Invariants(
	value Identifier_Search_Position_Stored, _ aver.Namespace,
) {
	_, valid := value.(Identifier_Search_Position)
	aver.Always(
		valid == (value != nil), "Identifier search position has expected storage type.",
	)
}

// Identifier_Search_End_Storage owns one transient lexer bound.
type Identifier_Search_End_Storage struct {
	// Value prevents a stale scalar proof from escaping lexer storage.
	Value Identifier_Search_End_Stored
}

// Identifier_Search_End_Storage_Invariants fixes termination ownership.
func Identifier_Search_End_Storage_Invariants(
	value Identifier_Search_End_Storage, namespace aver.Namespace,
) {
	Identifier_Search_End_Stored_Invariants(value.Value, namespace)
}

// Identifier_Search_End_Stored separates search shape from its transient bound.
type Identifier_Search_End_Stored interface{}

// Identifier_Search_End_Stored_Invariants fixes termination representation.
func Identifier_Search_End_Stored_Invariants(
	value Identifier_Search_End_Stored, _ aver.Namespace,
) {
	_, valid := value.(Identifier_Search_End)
	aver.Always(valid == (value != nil), "Identifier search end has expected storage type.")
}

// Arguments_Proof_Stored separates callback proof shape from transient arguments.
type Arguments_Proof_Stored interface{}

// Arguments_Proof_Stored_Invariants accepts absent output or typed arguments.
func Arguments_Proof_Stored_Invariants(value Arguments_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Arguments_Staged_Value)
	aver.Always(valid == (value != nil), "Arguments have expected storage type.")
}

// Functions_Proof_Stored separates registry proof shape from transient functions.
type Functions_Proof_Stored interface{}

// Functions_Proof_Stored_Invariants accepts absent output or one typed registry.
func Functions_Proof_Stored_Invariants(value Functions_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Functions_Validated_Value)
	aver.Always(valid == (value != nil), "Functions have expected storage type.")
}

// Node_Link_Proof_Stored separates link proof shape from transient references.
type Node_Link_Proof_Stored interface{}

// Node_Link_Proof_Stored_Invariants accepts absent output or one typed reference.
func Node_Link_Proof_Stored_Invariants(value Node_Link_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Node_Link_Validated_Value)
	aver.Always(valid == (value != nil), "Node link has expected storage type.")
}

// Output_Count_Proof_Stored separates writer proof shape from transient progress.
type Output_Count_Proof_Stored interface{}

// Output_Count_Proof_Stored_Invariants accepts absent output or one typed count.
func Output_Count_Proof_Stored_Invariants(value Output_Count_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Output_Count_Staged_Value)
	aver.Always(valid == (value != nil), "Output count has expected storage type.")
}

// Source_Cursor_Stored separates source proof shape from transient positions.
type Source_Cursor_Stored interface{}

// Source_Cursor_Stored_Invariants accepts absent output or one typed position.
func Source_Cursor_Stored_Invariants(value Source_Cursor_Stored, _ aver.Namespace) {
	_, valid := value.(Source_Cursor_Validated_Value)
	aver.Always(valid == (value != nil), "Source cursor has expected storage type.")
}

// Number_Position_Stored separates numeric proof shape from transient positions.
type Number_Position_Stored interface{}

// Number_Position_Stored_Invariants accepts absent output or one typed position.
func Number_Position_Stored_Invariants(value Number_Position_Stored, _ aver.Namespace) {
	_, valid := value.(Number_Position_Validated_Value)
	aver.Always(valid == (value != nil), "Number position has expected storage type.")
}

// Quoted_Position_Stored separates quoted proof shape from transient positions.
type Quoted_Position_Stored interface{}

// Quoted_Position_Stored_Invariants accepts absent output or one typed position.
func Quoted_Position_Stored_Invariants(value Quoted_Position_Stored, _ aver.Namespace) {
	_, valid := value.(Quoted_Position_Validated_Value)
	aver.Always(valid == (value != nil), "Quoted position has expected storage type.")
}

// Syntax_Variable_Count_Stored separates stack shape from transient population.
type Syntax_Variable_Count_Stored interface{}

// Syntax_Variable_Count_Stored_Invariants accepts absent zero or one typed count.
func Syntax_Variable_Count_Stored_Invariants(
	value Syntax_Variable_Count_Stored, _ aver.Namespace,
) {
	_, valid := value.(Syntax_Variable_Count_Storage_Value)
	aver.Always(valid == (value != nil), "Syntax variable count has expected storage type.")
}

// Control_Depth_Stored separates branch proof shape from transient depth.
type Control_Depth_Stored interface{}

// Control_Depth_Stored_Invariants accepts absent output or one typed depth.
func Control_Depth_Stored_Invariants(value Control_Depth_Stored, _ aver.Namespace) {
	_, valid := value.(Control_Depth_Tracked_Value)
	aver.Always(valid == (value != nil), "Control depth has expected storage type.")
}

// Declaration_References_Stored separates binding shape from transient references.
type Declaration_References_Stored interface{}

// Declaration_References_Stored_Invariants fixes declaration representation.
func Declaration_References_Stored_Invariants(
	value Declaration_References_Stored, _ aver.Namespace,
) {
	_, valid := value.(Declaration_References)
	aver.Always(valid == (value != nil), "Declarations have expected storage type.")
}

// Frame_Storage_Fields carries raw mutable traversal storage.
type Frame_Storage_Fields struct {
	// Value keeps mutation inside traversal storage.
	Value Frame_Stored
}

// Frame_Storage_Fields_Invariants bounds hostile traversal storage.
func Frame_Storage_Fields_Invariants(value Frame_Storage_Fields, namespace aver.Namespace) {
	Frame_Stored_Invariants(value.Value, namespace)
}

// Frame_State_Stored separates traversal shape from transient state.
type Frame_State_Stored interface{}

// Frame_State_Stored_Invariants fixes traversal-frame representation.
func Frame_State_Stored_Invariants(value Frame_State_Stored, _ aver.Namespace) {
	_, valid := value.(Frame_Stored)
	aver.Always(valid == (value != nil), "Traversal frame has expected storage type.")
}

// Value_Format_Frame_Storage_Fields carries raw mutable formatter storage.
type Value_Format_Frame_Storage_Fields struct {
	// Value keeps mutation inside formatter storage.
	Value Value_Format_Frame_Stored
}

// Value_Format_Frame_Storage_Fields_Invariants bounds hostile formatter storage.
func Value_Format_Frame_Storage_Fields_Invariants(
	value Value_Format_Frame_Storage_Fields, namespace aver.Namespace,
) {
	Value_Format_Frame_Stored_Invariants(value.Value, namespace)
}

// Value_Format_Frame_State_Stored separates formatter shape from transient state.
type Value_Format_Frame_State_Stored interface{}

// Value_Format_Frame_State_Stored_Invariants fixes formatter-frame representation.
func Value_Format_Frame_State_Stored_Invariants(
	value Value_Format_Frame_State_Stored, _ aver.Namespace,
) {
	_, valid := value.(Value_Format_Frame_Stored)
	aver.Always(valid == (value != nil), "Formatter frame has expected storage type.")
}

// Control_Frame_Storage_Fields carries raw mutable branch storage.
type Control_Frame_Storage_Fields struct {
	// Value keeps mutation inside the frame stack.
	Value Control_Frame_Stored
}

// Control_Frame_Storage_Fields_Invariants bounds hostile branch storage.
func Control_Frame_Storage_Fields_Invariants(
	value Control_Frame_Storage_Fields, namespace aver.Namespace,
) {
	Control_Frame_Stored_Invariants(value.Value, namespace)
}

// Control_Frame_State_Stored separates branch shape from transient state.
type Control_Frame_State_Stored interface{}

// Control_Frame_State_Stored_Invariants fixes branch-frame representation.
func Control_Frame_State_Stored_Invariants(value Control_Frame_State_Stored, _ aver.Namespace) {
	_, valid := value.(Control_Frame_Stored)
	aver.Always(valid == (value != nil), "Control frame has expected storage type.")
}

// Evaluation_Frame_Fields carries raw mutable evaluator storage.
type Evaluation_Frame_Fields struct {
	// Nodes retain caller parser records while parent evaluation waits.
	Nodes Evaluation_Nodes
	// Values retain dot and command head while child evaluation runs.
	Values Evaluation_Values
	// References retain next sibling work.
	References Evaluation_References
	// States retain pipeline value and declarations.
	States Evaluation_States
	// Argument_Bases isolate nested callback argument regions.
	Argument_Bases Evaluation_Argument_Bases
	// Control retains iterative evaluator state.
	Control Evaluation_Control
}

// Evaluation_Frame_Fields_Invariants bounds hostile evaluator storage.
func Evaluation_Frame_Fields_Invariants(
	value Evaluation_Frame_Fields, namespace aver.Namespace,
) {
	Evaluation_Nodes_Invariants(value.Nodes, namespace)
	Evaluation_Values_Invariants(value.Values, namespace)
	Evaluation_References_Invariants(value.References, namespace)
	Evaluation_States_Invariants(value.States, namespace)
	Evaluation_Argument_Bases_Invariants(value.Argument_Bases, namespace)
	Evaluation_Control_Invariants(value.Control, namespace)
}

// Evaluation_References_Stored separates evaluator shape from transient references.
type Evaluation_References_Stored interface{}

// Evaluation_References_Stored_Invariants fixes evaluator-reference representation.
func Evaluation_References_Stored_Invariants(
	value Evaluation_References_Stored, _ aver.Namespace,
) {
	_, valid := value.(Evaluation_References)
	aver.Always(valid == (value != nil), "Evaluation references have expected storage type.")
}

// Evaluation_Argument_Bases_Stored separates evaluator shape from transient stack state.
type Evaluation_Argument_Bases_Stored interface{}

// Evaluation_Argument_Bases_Stored_Invariants fixes argument-base representation.
func Evaluation_Argument_Bases_Stored_Invariants(
	value Evaluation_Argument_Bases_Stored, _ aver.Namespace,
) {
	_, valid := value.(Evaluation_Argument_Bases)
	aver.Always(valid == (value != nil), "Evaluation argument base has expected storage type.")
}

// Evaluation_Control_Stored separates evaluator shape from transient control state.
type Evaluation_Control_Stored interface{}

// Evaluation_Control_Stored_Invariants fixes evaluator-control representation.
func Evaluation_Control_Stored_Invariants(value Evaluation_Control_Stored, _ aver.Namespace) {
	_, valid := value.(Evaluation_Control)
	aver.Always(valid == (value != nil), "Evaluation control has expected storage type.")
}

// Configuration_Fields carries raw parser policy storage.
type Configuration_Fields struct {
	// Left retains the validated custom marker or empty default marker.
	Left Opening_Delimiter_Storage
	// Right retains the validated custom marker or empty default marker.
	Right Right_Delimiter_Storage
	// Mode retains validated parse flags.
	Mode Mode_Storage
	// Function retains caller name policy.
	Function Parse_Function_Storage
}

// Configuration_Fields_Invariants bounds hostile parser policy storage.
func Configuration_Fields_Invariants(value Configuration_Fields, namespace aver.Namespace) {
	Opening_Delimiter_Storage_Invariants(value.Left, namespace)
	Right_Delimiter_Storage_Invariants(value.Right, namespace)
	Mode_Storage_Invariants(value.Mode, namespace)
	Parse_Function_Storage_Invariants(value.Function, namespace)
}

// Configuration_Left_Stored separates policy shape from delimiter contents.
type Configuration_Left_Stored interface{}

// Configuration_Left_Stored_Invariants fixes opening-delimiter representation.
func Configuration_Left_Stored_Invariants(value Configuration_Left_Stored, _ aver.Namespace) {
	_, valid := value.(Opening_Delimiter_Storage)
	aver.Always(valid == (value != nil), "Configuration left delimiter has expected type.")
}

// Configuration_Right_Stored separates policy shape from delimiter contents.
type Configuration_Right_Stored interface{}

// Configuration_Right_Stored_Invariants fixes closing-delimiter representation.
func Configuration_Right_Stored_Invariants(value Configuration_Right_Stored, _ aver.Namespace) {
	_, valid := value.(Right_Delimiter_Storage)
	aver.Always(valid == (value != nil), "Configuration right delimiter has expected type.")
}

// Configuration_Mode_Stored separates policy shape from transient flags.
type Configuration_Mode_Stored interface{}

// Configuration_Mode_Stored_Invariants fixes mode representation.
func Configuration_Mode_Stored_Invariants(value Configuration_Mode_Stored, _ aver.Namespace) {
	_, valid := value.(Mode_Storage)
	aver.Always(valid == (value != nil), "Configuration mode has expected type.")
}

// Program_Validated_Fields carries raw validated-program proof storage.
type Program_Validated_Fields struct {
	// Source retains checked borrowed input.
	Source Source
	// Document retains checked arena metadata.
	Document Program_Validated_Document
	// Syntax retains checked caller storage.
	Syntax Syntax_Storage
}

// Program_Validated_Fields_Invariants bounds hostile program proof storage.
func Program_Validated_Fields_Invariants(
	value Program_Validated_Fields, namespace aver.Namespace,
) {
	Source_Invariants(value.Source, namespace)
	Program_Validated_Document_Invariants(value.Document, namespace)
	Syntax_Storage_Invariants(value.Syntax, namespace)
}

// Program_Document_Stored separates metadata proof shape from transient counts.
type Program_Document_Stored interface{}

// Program_Document_Stored_Invariants fixes validated metadata representation.
func Program_Document_Stored_Invariants(value Program_Document_Stored, _ aver.Namespace) {
	_, valid := value.(Program_Validated_Document)
	aver.Always(valid == (value != nil), "Program document has expected storage type.")
}

// Token_Validated_Fields carries raw lexer proof storage.
type Token_Validated_Fields struct {
	// Value cannot share identity with unvalidated input.
	Value Token_Validated_Value
}

// Token_Validated_Fields_Invariants bounds hostile lexer proof storage.
func Token_Validated_Fields_Invariants(value Token_Validated_Fields, namespace aver.Namespace) {
	Token_Validated_Value_Invariants(value.Value, namespace)
}

// Token_Proof_Stored separates lexer proof shape from its transient span.
type Token_Proof_Stored interface{}

// Token_Proof_Stored_Invariants fixes lexer proof representation.
func Token_Proof_Stored_Invariants(value Token_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Token_Validated_Value)
	aver.Always(valid == (value != nil), "Validated token has expected storage type.")
}

// Pipeline_Frame_Storage_Fields carries raw mutable frame storage.
type Pipeline_Frame_Storage_Fields struct {
	// Value keeps mutation inside the frame stack.
	Value Pipeline_Frame_Stored
}

// Pipeline_Frame_Storage_Fields_Invariants bounds hostile frame storage.
func Pipeline_Frame_Storage_Fields_Invariants(
	value Pipeline_Frame_Storage_Fields, namespace aver.Namespace,
) {
	Pipeline_Frame_Stored_Invariants(value.Value, namespace)
}

// Pipeline_Frame_State_Stored separates frame shape from transient references.
type Pipeline_Frame_State_Stored interface{}

// Pipeline_Frame_State_Stored_Invariants fixes pipeline-frame representation.
func Pipeline_Frame_State_Stored_Invariants(
	value Pipeline_Frame_State_Stored, _ aver.Namespace,
) {
	_, valid := value.(Pipeline_Frame_Stored)
	aver.Always(valid == (value != nil), "Pipeline frame has expected storage type.")
}

// List_State_Storage_Fields carries raw mutable list storage.
type List_State_Storage_Fields struct {
	// Value keeps mutation inside parser state.
	Value List_State_Stored
}

// List_State_Storage_Fields_Invariants bounds hostile list storage.
func List_State_Storage_Fields_Invariants(
	value List_State_Storage_Fields, namespace aver.Namespace,
) {
	List_State_Stored_Invariants(value.Value, namespace)
}

// List_State_Value_Stored separates list shape from transient references.
type List_State_Value_Stored interface{}

// List_State_Value_Stored_Invariants fixes list-state representation.
func List_State_Value_Stored_Invariants(value List_State_Value_Stored, _ aver.Namespace) {
	_, valid := value.(List_State_Stored)
	aver.Always(valid == (value != nil), "List state has expected storage type.")
}

// Number_Span_Validated_Fields carries raw numeric-span proof storage.
type Number_Span_Validated_Fields struct {
	// Value keeps token proof attached to the span.
	Value Number_Span_Stored
}

// Number_Span_Validated_Fields_Invariants bounds hostile numeric proof storage.
func Number_Span_Validated_Fields_Invariants(
	value Number_Span_Validated_Fields, namespace aver.Namespace,
) {
	Number_Span_Stored_Invariants(value.Value, namespace)
}

// Number_Span_Proof_Stored separates numeric proof shape from transient boundaries.
type Number_Span_Proof_Stored interface{}

// Number_Span_Proof_Stored_Invariants fixes numeric-span representation.
func Number_Span_Proof_Stored_Invariants(value Number_Span_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Number_Span_Stored)
	aver.Always(valid == (value != nil), "Number span has expected storage type.")
}

// Quoted_Span_Validated_Fields carries raw quoted-span proof storage.
type Quoted_Span_Validated_Fields struct {
	// Value keeps source proof attached to the span.
	Value Quoted_Span_Stored
}

// Quoted_Span_Validated_Fields_Invariants bounds hostile quoted proof storage.
func Quoted_Span_Validated_Fields_Invariants(
	value Quoted_Span_Validated_Fields, namespace aver.Namespace,
) {
	Quoted_Span_Stored_Invariants(value.Value, namespace)
}

// Quoted_Span_Proof_Stored separates quoted proof shape from transient boundaries.
type Quoted_Span_Proof_Stored interface{}

// Quoted_Span_Proof_Stored_Invariants fixes quoted-span representation.
func Quoted_Span_Proof_Stored_Invariants(value Quoted_Span_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Quoted_Span_Stored)
	aver.Always(valid == (value != nil), "Quoted span has expected storage type.")
}

// Action_Span_Validated_Fields carries raw action-span proof storage.
type Action_Span_Validated_Fields struct {
	// Value keeps delimiter proof attached to the span.
	Value Action_Span_Stored
}

// Action_Span_Validated_Fields_Invariants bounds hostile action proof storage.
func Action_Span_Validated_Fields_Invariants(
	value Action_Span_Validated_Fields, namespace aver.Namespace,
) {
	Action_Span_Stored_Invariants(value.Value, namespace)
}

// Action_Span_Proof_Stored separates action proof shape from transient boundaries.
type Action_Span_Proof_Stored interface{}

// Action_Span_Proof_Stored_Invariants fixes action-span representation.
func Action_Span_Proof_Stored_Invariants(value Action_Span_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Action_Span_Stored)
	aver.Always(valid == (value != nil), "Action span has expected storage type.")
}

// Lexeme_Validated_Fields carries raw source-span proof storage.
type Lexeme_Validated_Fields struct {
	// Value cannot share identity with unvalidated input.
	Value Lexeme_Stored
}

// Lexeme_Validated_Fields_Invariants bounds hostile lexeme proof storage.
func Lexeme_Validated_Fields_Invariants(
	value Lexeme_Validated_Fields, namespace aver.Namespace,
) {
	Lexeme_Stored_Invariants(value.Value, namespace)
}

// Lexeme_Proof_Stored separates lexeme proof shape from transient boundaries.
type Lexeme_Proof_Stored interface{}

// Lexeme_Proof_Stored_Invariants fixes lexeme representation.
func Lexeme_Proof_Stored_Invariants(value Lexeme_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Lexeme_Stored)
	aver.Always(valid == (value != nil), "Lexeme has expected storage type.")
}

// Identifier_Span_Validated_Fields carries raw identifier proof storage.
type Identifier_Span_Validated_Fields struct {
	// Value cannot share identity with unvalidated input.
	Value Identifier_Span_Stored
}

// Identifier_Span_Validated_Fields_Invariants bounds hostile identifier proof storage.
func Identifier_Span_Validated_Fields_Invariants(
	value Identifier_Span_Validated_Fields, namespace aver.Namespace,
) {
	Identifier_Span_Stored_Invariants(value.Value, namespace)
}

// Identifier_Span_Proof_Stored separates identifier proof shape from transient boundaries.
type Identifier_Span_Proof_Stored interface{}

// Identifier_Span_Proof_Stored_Invariants fixes identifier-span representation.
func Identifier_Span_Proof_Stored_Invariants(
	value Identifier_Span_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Identifier_Span_Stored)
	aver.Always(valid == (value != nil), "Identifier span has expected storage type.")
}

// Text_Span_Validated_Fields carries raw text proof storage.
type Text_Span_Validated_Fields struct {
	// Value keeps parser proof attached to the span.
	Value Text_Span_Stored
}

// Text_Span_Validated_Fields_Invariants bounds hostile text proof storage.
func Text_Span_Validated_Fields_Invariants(
	value Text_Span_Validated_Fields, namespace aver.Namespace,
) {
	Text_Span_Stored_Invariants(value.Value, namespace)
}

// Text_Span_Proof_Stored separates text proof shape from transient boundaries.
type Text_Span_Proof_Stored interface{}

// Text_Span_Proof_Stored_Invariants fixes text-span representation.
func Text_Span_Proof_Stored_Invariants(value Text_Span_Proof_Stored, _ aver.Namespace) {
	_, valid := value.(Text_Span_Stored)
	aver.Always(valid == (value != nil), "Text span has expected storage type.")
}

// Action_Scan_Span_Validated_Fields carries raw scan proof storage.
type Action_Scan_Span_Validated_Fields struct {
	// Value keeps source proof attached to the span.
	Value Action_Scan_Span_Stored
}

// Action_Scan_Span_Validated_Fields_Invariants bounds hostile scan proof storage.
func Action_Scan_Span_Validated_Fields_Invariants(
	value Action_Scan_Span_Validated_Fields, namespace aver.Namespace,
) {
	Action_Scan_Span_Stored_Invariants(value.Value, namespace)
}

// Action_Scan_Span_Proof_Stored separates scan proof shape from transient boundaries.
type Action_Scan_Span_Proof_Stored interface{}

// Action_Scan_Span_Proof_Stored_Invariants fixes scan-span representation.
func Action_Scan_Span_Proof_Stored_Invariants(
	value Action_Scan_Span_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Action_Scan_Span_Stored)
	aver.Always(valid == (value != nil), "Action scan span has expected storage type.")
}

// Template_Search_Validated_Fields carries raw lookup proof storage.
type Template_Search_Validated_Fields struct {
	// Value keeps traversal proof attached to the result.
	Value Template_Search_Stored
}

// Template_Search_Validated_Fields_Invariants bounds hostile lookup proof storage.
func Template_Search_Validated_Fields_Invariants(
	value Template_Search_Validated_Fields, namespace aver.Namespace,
) {
	Template_Search_Stored_Invariants(value.Value, namespace)
}

// Template_Search_Proof_Stored separates lookup proof shape from transient references.
type Template_Search_Proof_Stored interface{}

// Template_Search_Proof_Stored_Invariants fixes lookup representation.
func Template_Search_Proof_Stored_Invariants(
	value Template_Search_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Template_Search_Stored)
	aver.Always(valid == (value != nil), "Template search has expected storage type.")
}

// Scope_Restore_Validated_Fields carries raw scope-restoration proof storage.
type Scope_Restore_Validated_Fields struct {
	// Value keeps workspace proof attached to restoration state.
	Value Scope_Restore_Stored
}

// Scope_Restore_Validated_Fields_Invariants bounds hostile restoration proof storage.
func Scope_Restore_Validated_Fields_Invariants(
	value Scope_Restore_Validated_Fields, namespace aver.Namespace,
) {
	Scope_Restore_Stored_Invariants(value.Value, namespace)
}

// Scope_Restore_Proof_Stored separates restoration proof shape from transient cursors.
type Scope_Restore_Proof_Stored interface{}

// Scope_Restore_Proof_Stored_Invariants fixes restoration representation.
func Scope_Restore_Proof_Stored_Invariants(
	value Scope_Restore_Proof_Stored, _ aver.Namespace,
) {
	_, valid := value.(Scope_Restore_Stored)
	aver.Always(valid == (value != nil), "Scope restoration has expected storage type.")
}

// Template_State_Fields carries raw definition-chain storage.
type Template_State_Fields struct {
	// First begins definition sibling chain.
	First Template_First_Storage
	// Last links next definition.
	Last Template_Last_Storage
	// Count bounds definition chain length.
	Count Template_Count_Storage
}

// Template_State_Fields_Invariants bounds hostile definition-chain storage.
func Template_State_Fields_Invariants(value Template_State_Fields, namespace aver.Namespace) {
	Template_First_Storage_Invariants(value.First, namespace)
	Template_Last_Storage_Invariants(value.Last, namespace)
	Template_Count_Storage_Invariants(value.Count, namespace)
}

// Template_First_Stored separates chain shape from transient references.
type Template_First_Stored interface{}

// Template_First_Stored_Invariants fixes first-reference representation.
func Template_First_Stored_Invariants(value Template_First_Stored, _ aver.Namespace) {
	_, valid := value.(Template_First_Storage)
	aver.Always(valid == (value != nil), "Template first reference has expected storage type.")
}

// Template_Last_Stored separates chain shape from transient references.
type Template_Last_Stored interface{}

// Template_Last_Stored_Invariants fixes last-reference representation.
func Template_Last_Stored_Invariants(value Template_Last_Stored, _ aver.Namespace) {
	_, valid := value.(Template_Last_Storage)
	aver.Always(valid == (value != nil), "Template last reference has expected storage type.")
}

// Template_Count_Stored separates chain shape from transient population.
type Template_Count_Stored interface{}

// Template_Count_Stored_Invariants fixes definition-count representation.
func Template_Count_Stored_Invariants(value Template_Count_Stored, _ aver.Namespace) {
	_, valid := value.(Template_Count_Storage)
	aver.Always(valid == (value != nil), "Template count has expected storage type.")
}
