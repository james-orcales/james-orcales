// Package callgraph is the callgraph tool's internal library tier. It models a
// program's resolved call graph — which functions call which, and, for calls
// dispatched through injected func-typed fields, which concrete callable a field
// was wired to at a composition root — then answers queries over it: tracing an
// injected dependency from the root that bound it to the leaf that invokes it,
// flagging invokes no wiring reaches, and diffing the resolved edges across
// commits.
//
// It receives normalized facts, extracted from type-checked syntax by package
// main, and never loads or type-checks source itself; that impurity stays in
// package main.
package callgraph

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"slices"
	"strconv"
	"strings"

	"local/james-orcales/shared/cli"
)

// Function is one defined function (or a method satisfying a stdlib interface).
type Function struct {
	// Key identifies the function, qualified by import path so it holds no
	// source position and stays stable across commits — the diff depends on it.
	Key string
	// Is_Root marks a composition root: a main package, a test, or a sanctioned
	// instrumentation or default package, where impure bindings enter the graph.
	Is_Root bool
}

// Field is a func-typed struct field carrying an injected dependency, such as
// time.Clock.Now_Monotonic or io.IO.Read.
type Field struct {
	// Key identifies the field as its owning type's key plus the field name.
	Key string
}

// Call is a resolved edge: From invokes the concrete function To.
type Call struct {
	// From is the caller's function key.
	From string
	// To is the callee's function key.
	To string
}

// Invoke is a call dispatched through a func-typed field. The callee is whatever
// value the field holds, so it stays unresolved until the field's wiring is found.
type Invoke struct {
	// From is the invoking function's key.
	From string
	// Through is the key of the field the call dispatches through.
	Through string
}

// Wire records that at composition root Root, field Field was bound to concrete
// callable Callable — the point an injected dependency enters the graph.
type Wire struct {
	// Root is the function key of the composition root holding the binding.
	Root string
	// Field is the key of the field being wired.
	Field string
	// Callable is the function key the field was bound to.
	Callable string
}

// Flow is a provenance edge: an injected value passes From one function To
// another, handed on as a call argument or stored into a field.
type Flow struct {
	// From is the function key the value departs.
	From string
	// To is the function key the value arrives at.
	To string
}

// Graph is the whole program's facts. Every slice is sorted by key so the graph
// serializes deterministically, which the diff depends on.
type Graph struct {
	// Functions are the program's function nodes.
	Functions []Function
	// Fields are the program's injected func-typed field nodes.
	Fields []Field
	// Calls are the resolved call edges, including those resolution derives from
	// Invokes.
	Calls []Call
	// Invokes are the calls dispatched through fields, pending resolution.
	Invokes []Invoke
	// Wires are the field-to-callable bindings made at composition roots.
	Wires []Wire
	// Flows are the value-provenance edges.
	Flows []Flow
}

// Package is a type-checked package handed to Extract by the loader. Extract
// reads its syntax and type information and performs no loading itself.
type Package struct {
	// Path is the package's import path, the prefix of every key Extract mints
	// for a declaration in the package.
	Path string
	// Is_Root marks the package as a composition root, where wirings are read.
	Is_Root bool
	// File_Set resolves the syntax trees' positions.
	File_Set *token.FileSet
	// Files are the package's parsed syntax trees.
	Files []*ast.File
	// Info holds the resolved type information for Files.
	Info *types.Info
	// Types is the checked package the syntax belongs to.
	Types *types.Package
}

// Chain is an injection chain: the ordered functions an injected dependency
// threads through, from the composition root that wired it to the leaf that
// invokes it.
type Chain struct {
	// Origin is the concrete callable the chain's dependency was wired to.
	Origin string
	// Steps are the function keys in order, root first and leaf last.
	Steps []string
}

// The exit code for a clean run.
const EXIT_SUCCESS = 0

// The exit code for a malformed command line.
const EXIT_USAGE = 2

// Main_Input is the input for Main.
type Main_Input struct {
	// Arguments is the command line, including the program name.
	Arguments []string
	// Output receives the rendered report.
	Output io.Writer
	// Error_Output receives usage and error text.
	Error_Output io.Writer
	// Load finds and type-checks the workspace into extractable packages and
	// reports its module path; the host injects it because it reaches the
	// filesystem.
	Load func() (packages []*Package, module string)
}

// Main parses the command line, loads the workspace through the injected loader,
// and writes each invoked field's injection chain, returning an exit code.
func Main(input *Main_Input) (code int) {
	program := main_program()
	command, parse_err := cli.Program_Parse(&program, input.Arguments)
	if parse_err != nil {
		fmt.Fprintf(input.Error_Output, "callgraph: %v\n\n", parse_err)
		cli.Print_Help(input.Error_Output, program)
		return EXIT_USAGE
	}
	filter := ""
	filters := cli.Get_Option(command.Arguments, "filter").Value.([]string)
	if len(filters) > 0 {
		filter = filters[0]
	}
	packages, module := input.Load()
	prefix := ""
	if module != "" {
		prefix = module + "/"
	}
	graph := Extract(packages)
	fmt.Fprint(input.Output, Render(&Render_Input{Graph: graph, Filter: filter, Strip: prefix}))
	return EXIT_SUCCESS
}

// Declares the callgraph command line: a commandless program whose one optional
// positional filters the chains by substring.
func main_program() (program cli.Program) {
	return cli.New_Single(cli.New_Single_Input{
		Label:       "callgraph",
		Description: "trace injected dependencies from wiring to use site",
		Arguments: []cli.Option{
			cli.New_Variadic[string](cli.New_Variadic_Input{
				Label:       "filter",
				Description: "show only chains whose line contains this substring",
			}),
		},
	})
}

// String renders the chain as "root → … → leaf", matching the doctrine's
// hand-drawn injection diagrams.
func (c Chain) String() (rendered string) {
	return strings.Join(c.Steps, " → ")
}

// Graph_Chain traces field's injected value from the composition root that wired
// it, along the value's flow edges, to the leaf that invokes it. found is false
// when no wiring reaches field — a severed injection chain.
func Graph_Chain(g *Graph, field string) (chain Chain, found bool) {
	wire, wired := graph_wire_for(g, field)
	if !wired {
		return Chain{}, false
	}
	steps := graph_flow_path(g, wire.Root)
	invoke, invoked := graph_invoke_for(g, field)
	if invoked {
		steps = append(steps, invoke.From)
	}
	return Chain{Origin: wire.Callable, Steps: steps}, true
}

// Returns the wiring that binds field, if any.
func graph_wire_for(g *Graph, field string) (wire Wire, found bool) {
	for _, candidate := range g.Wires {
		if candidate.Field == field {
			return candidate, true
		}
	}
	return Wire{}, false
}

// Returns the invoke dispatched through field, if any.
func graph_invoke_for(g *Graph, field string) (invoke Invoke, found bool) {
	for _, candidate := range g.Invokes {
		if candidate.Through == field {
			return candidate, true
		}
	}
	return Invoke{}, false
}

// Follows the value's flow edges forward from start, returning the ordered
// functions it threads through. A visited set stops a cycle from looping forever.
func graph_flow_path(g *Graph, start string) (path []string) {
	visited := map[string]bool{}
	current := start
	for !visited[current] {
		visited[current] = true
		path = append(path, current)
		next, exists := graph_flow_next(g, current)
		if !exists {
			return path
		}
		current = next
	}
	return path
}

// Returns the function the value flows to from current, if any.
func graph_flow_next(g *Graph, current string) (next string, found bool) {
	for _, candidate := range g.Flows {
		if candidate.From == current {
			return candidate.To, true
		}
	}
	return "", false
}

// Graph_Resolve rewrites each invoke into a resolved call, binding it to the
// callable its field was wired to, and appends the derived edges to g.Calls.
func Graph_Resolve(g *Graph) {
	for _, invoke := range g.Invokes {
		wire, wired := graph_wire_for(g, invoke.Through)
		if !wired {
			continue
		}
		g.Calls = append(g.Calls, Call{From: invoke.From, To: wire.Callable})
	}
}

// Graph_Broken reports every field an invoke dispatches through that no wiring
// reaches — a severed injection chain. The result is sorted and deduplicated.
func Graph_Broken(g *Graph) (fields []string) {
	seen := map[string]bool{}
	for _, invoke := range g.Invokes {
		_, wired := graph_wire_for(g, invoke.Through)
		if wired {
			continue
		}
		if seen[invoke.Through] {
			continue
		}
		seen[invoke.Through] = true
		fields = append(fields, invoke.Through)
	}
	slices.Sort(fields)
	return fields
}

// Graph_Diff_Input is the input for Graph_Diff.
type Graph_Diff_Input struct {
	// Before is the earlier graph.
	Before *Graph
	// After is the later graph.
	After *Graph
}

// Graph_Diff compares two resolved graphs and reports the calls present in one
// but not the other. added holds calls new in After; removed holds calls gone
// from Before. Both are sorted.
func Graph_Diff(input *Graph_Diff_Input) (added []Call, removed []Call) {
	before := call_set(input.Before.Calls)
	after := call_set(input.After.Calls)
	added = calls_absent_from(input.After.Calls, before)
	removed = calls_absent_from(input.Before.Calls, after)
	return added, removed
}

// Builds a set of call keys for membership tests.
func call_set(calls []Call) (set map[string]bool) {
	set = map[string]bool{}
	for _, call := range calls {
		set[call_key(call)] = true
	}
	return set
}

// Returns the calls whose key is absent from set, sorted by key.
func calls_absent_from(calls []Call, set map[string]bool) (absent []Call) {
	for _, call := range calls {
		if set[call_key(call)] {
			continue
		}
		absent = append(absent, call)
	}
	slices.SortFunc(absent, func(a Call, b Call) (order int) {
		return strings.Compare(call_key(a), call_key(b))
	})
	return absent
}

// Returns a stable key for a call edge.
func call_key(call Call) (key string) {
	return call.From + "\x00" + call.To
}

// Extract builds a Graph from type-checked packages, minting the facts the
// queries read: functions, injected fields, calls, invokes, wirings, and flows.
// The traversal follows source order, so the graph is deterministic without a
// sort.
func Extract(packages []*Package) (graph *Graph) {
	graph = &Graph{}
	for _, checked := range packages {
		for _, file := range checked.Files {
			extract_file(graph, checked, file)
		}
	}
	return graph
}

// Walks a file's declarations, minting facts for each function.
func extract_file(graph *Graph, checked *Package, file *ast.File) {
	for _, declaration := range file.Decls {
		function, is_function := declaration.(*ast.FuncDecl)
		if !is_function {
			continue
		}
		extract_function(graph, checked, function)
	}
}

// Mints a function node and scans its body for invokes through fields. Methods
// are skipped: the doctrine bans them for a package's own API, so injection
// chains run through free functions.
func extract_function(graph *Graph, checked *Package, function *ast.FuncDecl) {
	if function.Recv != nil {
		return
	}
	from := checked.Path + "." + function.Name.Name
	graph.Functions = append(graph.Functions, Function{Key: from, Is_Root: checked.Is_Root})
	if function.Body == nil {
		return
	}
	closures := 0
	ast.Inspect(function.Body, func(node ast.Node) (recurse bool) {
		extract_invoke(graph, checked, from, node)
		extract_wire(graph, checked, from, node, &closures)
		extract_assign(graph, checked, from, node, &closures)
		extract_flow(graph, checked, from, node)
		extract_call(graph, checked, from, node)
		return true
	})
}

// Mints an invoke fact when node is a call dispatched through a func-typed
// struct field, such as s.Now().
func extract_invoke(graph *Graph, checked *Package, from string, node ast.Node) {
	call, is_call := node.(*ast.CallExpr)
	if !is_call {
		return
	}
	selector, is_selector := call.Fun.(*ast.SelectorExpr)
	if !is_selector {
		return
	}
	selection, selected := checked.Info.Selections[selector]
	if !selected {
		return
	}
	if selection.Kind() != types.FieldVal {
		return
	}
	key, keyed := field_through(selection)
	if !keyed {
		return
	}
	graph.Fields = append(graph.Fields, Field{Key: key})
	graph.Invokes = append(graph.Invokes, Invoke{From: from, Through: key})
}

// Returns the key of the func-typed field a selection dispatches through, and
// whether the selection is such a field at all.
func field_through(selection *types.Selection) (key string, found bool) {
	field, is_variable := selection.Obj().(*types.Var)
	if !is_variable {
		return "", false
	}
	_, is_function := field.Type().Underlying().(*types.Signature)
	if !is_function {
		return "", false
	}
	named, has_named := named_of(selection.Recv())
	if !has_named {
		return "", false
	}
	owner := named.Obj()
	if owner.Pkg() == nil {
		return "", false
	}
	return owner.Pkg().Path() + "." + owner.Name() + "." + field.Name(), true
}

// Returns the named type behind a value or pointer type, if any.
func named_of(typ types.Type) (named *types.Named, found bool) {
	pointer, is_pointer := typ.(*types.Pointer)
	if is_pointer {
		typ = pointer.Elem()
	}
	result, is_named := typ.(*types.Named)
	if !is_named {
		return nil, false
	}
	return result, true
}

// Mints wire facts for a struct literal that binds func-typed fields to
// callables, such as Clock{Now_Realtime: real_now}.
func extract_wire(graph *Graph, checked *Package, root string, node ast.Node, closures *int) {
	literal, is_literal := node.(*ast.CompositeLit)
	if !is_literal {
		return
	}
	named, has_named := named_of(checked.Info.TypeOf(literal))
	if !has_named {
		return
	}
	owner := named.Obj()
	if owner.Pkg() == nil {
		return
	}
	for _, element := range literal.Elts {
		pair, is_pair := element.(*ast.KeyValueExpr)
		if !is_pair {
			continue
		}
		name, is_name := pair.Key.(*ast.Ident)
		if !is_name {
			continue
		}
		field, is_field := checked.Info.Uses[name].(*types.Var)
		if !is_field {
			continue
		}
		_, is_function := field.Type().Underlying().(*types.Signature)
		if !is_function {
			continue
		}
		callable, callable_found := callable_of(checked, root, pair.Value, closures)
		if !callable_found {
			continue
		}
		field_key := owner.Pkg().Path() + "." + owner.Name() + "." + field.Name()
		graph.Fields = append(graph.Fields, Field{Key: field_key})
		graph.Wires = append(graph.Wires,
			Wire{Root: root, Field: field_key, Callable: callable})
	}
}

// Mints wire facts for field assignments that bind a func-typed field to a
// callable, such as loop.Read = func(...) {...} — the form the IO gateways use.
func extract_assign(graph *Graph, checked *Package, root string, node ast.Node, closures *int) {
	assignment, is_assign := node.(*ast.AssignStmt)
	if !is_assign {
		return
	}
	if len(assignment.Lhs) != len(assignment.Rhs) {
		return
	}
	for pair_index := range assignment.Lhs {
		selector, is_selector := assignment.Lhs[pair_index].(*ast.SelectorExpr)
		if !is_selector {
			continue
		}
		selection, selected := checked.Info.Selections[selector]
		if !selected {
			continue
		}
		if selection.Kind() != types.FieldVal {
			continue
		}
		field_key, keyed := field_through(selection)
		if !keyed {
			continue
		}
		callable, found := callable_of(checked, root, assignment.Rhs[pair_index], closures)
		if !found {
			continue
		}
		graph.Fields = append(graph.Fields, Field{Key: field_key})
		graph.Wires = append(graph.Wires,
			Wire{Root: root, Field: field_key, Callable: callable})
	}
}

// Returns the key of the callable an expression denotes: a named or package
// function, or a closure keyed by its enclosing root and ordinal.
func callable_of(
	checked *Package, root string, expression ast.Expr, closures *int,
) (callable string, found bool) {
	switch value := expression.(type) {
	case *ast.Ident:
		return function_object_key(checked.Info.Uses[value])
	case *ast.SelectorExpr:
		return function_object_key(checked.Info.Uses[value.Sel])
	case *ast.FuncLit:
		ordinal := *closures
		*closures = *closures + 1
		return root + "#closure" + strconv.Itoa(ordinal), true
	default:
		return "", false
	}
}

// Returns the key of a function object, or false when the object is not one.
func function_object_key(object types.Object) (key string, found bool) {
	function, is_function := object.(*types.Func)
	if !is_function {
		return "", false
	}
	if function.Pkg() == nil {
		return function.Name(), true
	}
	return function.Pkg().Path() + "." + function.Name(), true
}

// Mints a flow edge when a call hands a func-typed value on to another function,
// threading an injected dependency one step further.
func extract_flow(graph *Graph, checked *Package, from string, node ast.Node) {
	call, is_call := node.(*ast.CallExpr)
	if !is_call {
		return
	}
	callee, callee_found := call_target_key(checked, call.Fun)
	if !callee_found {
		return
	}
	if !call_passes_function(checked, call) {
		return
	}
	graph.Flows = append(graph.Flows, Flow{From: from, To: callee})
}

// Returns the key of the function a call targets, if it is a plain call.
func call_target_key(checked *Package, callee ast.Expr) (key string, found bool) {
	switch target := callee.(type) {
	case *ast.Ident:
		return function_object_key(checked.Info.Uses[target])
	case *ast.SelectorExpr:
		return function_object_key(checked.Info.Uses[target.Sel])
	default:
		return "", false
	}
}

// Reports whether any argument of a call is a func-typed value being handed on.
func call_passes_function(checked *Package, call *ast.CallExpr) (passes bool) {
	for _, argument := range call.Args {
		argument_type := checked.Info.TypeOf(argument)
		if argument_type == nil {
			continue
		}
		_, is_function := argument_type.Underlying().(*types.Signature)
		if is_function {
			return true
		}
	}
	return false
}

// Mints a static call edge for a plain function or method call — the tree's
// structural edges, distinct from the field invokes rendered as annotations.
func extract_call(graph *Graph, checked *Package, from string, node ast.Node) {
	call, is_call := node.(*ast.CallExpr)
	if !is_call {
		return
	}
	callee, found := call_target_key(checked, call.Fun)
	if !found {
		return
	}
	graph.Calls = append(graph.Calls, Call{From: from, To: callee})
}

// A renderer carries the graph and module prefix shared by every tree helper.
type renderer struct {
	// Graph is the graph being drawn.
	Graph *Graph
	// Strip is the module prefix removed from keys, also the first-party marker.
	Strip string
}

// A frame is a pending tree node on the render stack.
type frame struct {
	// Key is the function the node renders.
	Key string
	// Prefix is the indentation printed before the node's connector.
	Prefix string
	// Last marks the node as its parent's final child, choosing └─ over ├─.
	Last bool
}

// Render_Input is the input for Render.
type Render_Input struct {
	// Graph is the graph to render.
	Graph *Graph
	// Filter, when non-empty, keeps only entry-point trees whose text contains it.
	Filter string
	// Strip is the module prefix removed from keys, and the first-party marker.
	Strip string
}

// Render draws the transitive first-party call tree under each entry point. A
// call through an injected field hangs off its caller as a ⟶ annotation naming
// the field and what it was wired to; stdlib callees are not expanded, and a
// callee already drawn elsewhere is shown once and marked "(shown above)" — it
// is a shared callee, not necessarily a cycle. Filter keeps matching trees.
func Render(input *Render_Input) (output string) {
	drawer := &renderer{Graph: input.Graph, Strip: input.Strip}
	var builder strings.Builder
	for _, root := range entry_points(input.Graph) {
		tree := render_tree(drawer, root)
		if !strings.Contains(tree, input.Filter) {
			continue
		}
		builder.WriteString(tree)
		builder.WriteString("\n")
	}
	return builder.String()
}

// Returns the sorted keys of the program's entry points — functions named Main
// or main — where every call tree is rooted.
func entry_points(graph *Graph) (roots []string) {
	seen := map[string]bool{}
	for _, function := range graph.Functions {
		if !is_entry_point(function.Key) {
			continue
		}
		if seen[function.Key] {
			continue
		}
		seen[function.Key] = true
		roots = append(roots, function.Key)
	}
	slices.Sort(roots)
	return roots
}

// Reports whether a function key names an entry point.
func is_entry_point(key string) (entry bool) {
	if strings.HasSuffix(key, ".Main") {
		return true
	}
	if strings.HasSuffix(key, ".main") {
		return true
	}
	return false
}

// Draws the transitive first-party call tree under root, walking iteratively so
// no function recurses by name. A visited set stops a cycle and collapses a
// callee already drawn elsewhere to a single "(shown above)" line.
func render_tree(drawer *renderer, root string) (text string) {
	var builder strings.Builder
	visited := map[string]bool{root: true}
	builder.WriteString(render_short(drawer, root) + render_annotations(drawer, root) + "\n")
	stack := frames_for(render_callees(drawer, root), "")
	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		connector := "├─ "
		extension := "│  "
		if top.Last {
			connector = "└─ "
			extension = "   "
		}
		builder.WriteString(top.Prefix + connector + render_short(drawer, top.Key))
		builder.WriteString(render_annotations(drawer, top.Key))
		if visited[top.Key] {
			builder.WriteString("  (shown above)\n")
			continue
		}
		visited[top.Key] = true
		builder.WriteString("\n")
		children := frames_for(render_callees(drawer, top.Key), top.Prefix+extension)
		stack = append(stack, children...)
	}
	return builder.String()
}

// Wraps callee keys as stack frames in reverse, so a LIFO pop yields display
// order; the last key is tagged for its └─ connector.
func frames_for(keys []string, prefix string) (frames []*frame) {
	for child_index := len(keys) - 1; child_index >= 0; child_index-- {
		frames = append(frames, &frame{
			Key:    keys[child_index],
			Prefix: prefix,
			Last:   child_index == len(keys)-1,
		})
	}
	return frames
}

// Returns parent's sorted, deduplicated first-party callees — the tree's edges.
func render_callees(drawer *renderer, parent string) (callees []string) {
	if drawer.Strip == "" {
		return callees
	}
	seen := map[string]bool{}
	for _, call := range drawer.Graph.Calls {
		if call.From != parent {
			continue
		}
		if !strings.HasPrefix(call.To, drawer.Strip) {
			continue
		}
		if seen[call.To] {
			continue
		}
		seen[call.To] = true
		callees = append(callees, call.To)
	}
	slices.Sort(callees)
	return callees
}

// Returns a function's injected-field annotations: for each field it invokes,
// the field and what it was wired to.
func render_annotations(drawer *renderer, from string) (text string) {
	seen := map[string]bool{}
	for _, invoke := range drawer.Graph.Invokes {
		if invoke.From != from {
			continue
		}
		if seen[invoke.Through] {
			continue
		}
		seen[invoke.Through] = true
		text = text + "  ⟶ " + render_short(drawer, invoke.Through) +
			" [" + render_origin(drawer, invoke.Through) + "]"
	}
	return text
}

// Returns what a field was wired to: a callable's short key, the enclosing
// function for a closure, or "unwired" when nothing binds it.
func render_origin(drawer *renderer, field string) (origin string) {
	wire, found := graph_wire_for(drawer.Graph, field)
	if !found {
		return "unwired"
	}
	callable := render_short(drawer, wire.Callable)
	closure_offset := strings.Index(callable, "#closure")
	if closure_offset < 0 {
		return callable
	}
	return "closure @ " + callable[:closure_offset]
}

// Removes the module prefix from a key so it reads short; a stdlib key, lacking
// the prefix, passes through unchanged.
func render_short(drawer *renderer, key string) (name string) {
	return strings.TrimPrefix(key, drawer.Strip)
}
