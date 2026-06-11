# Building a Pratt Parser in Zig

## Parsing a single digit

Turn this expression into an abstract syntax tree using Pratt parsing. Start with a simple test that
parses a single digit. Tokens are produced by the lexer from an input string, and then the parser
takes these tokens and turns them into nodes. The `ast` struct is simply a reference to the root
node. The lexer works by skipping whitespace and turning single digits into atoms.

In the parser function `parse`, `ast`'s root field is set to the output of the helper function
`parseNode`. `nextToken` and `peekToken` are helper functions to iterate over the token slice.
`parseNode` heap-allocates a node, and that node holds a character of the next token, and then it
returns a pointer to that node.

The original Pratt parser paper says that this parser is initially positioned at the beginning of
the input. It runs the code of the current token, stores the result in a variable called `left`,
advances the input, and repeats the process. If the input is exhausted, then by default the parser
halts and returns a value of `left`. To encode the AST, return the character of the root node, as we
simply expect one node for now.

I called these functions in `main` so I can get `zig build` diagnostics for my LSP — you don't have
to do this. Fix the errors, call `zig test`, and our first test passes.

```zig
// lexer.zig
pub const Token = union(enum) {
    atom: char,
    op: char,
    eof,
};

pub fn next(l: *Lexer) Token {
    var result: Token = .eof;
    while (l.i < l.input.len) {
        const ch = l.input[l.i];
        switch (ch) {
            ' ', '\n', '\t', '\r' => l.i += 1,
            '0'...'9', 'a'...'z' => {
                result = .{ .atom = ch };
                break;
            },
            // operators handled in later sections
            else => std.debug.panic("Unsupported token: {c}", .{ch}),
        }
    }
    l.i += 1;
    return result;
}
```

```zig
// 1_parser.zig
const Node = struct {
    ch: char,
    args: ?Arguments, // always null until we add operators
    // ...
};

const Parser = struct {
    allocator: Allocator,
    tokens: []const Token,
    tok_i: usize,
    root: *Node,

    fn parse(p: *Parser) !void {
        p.root = try p.parse_node(0);
    }

    fn parse_node(p: *Parser, bp_min: u8) !*Node {
        var left_ptr = try p.allocator.create(Node);
        left_ptr.* = switch (p.next_tok()) {
            .atom => |ch| .{ .ch = ch, .args = null },
            // operator cases added later
            .eof => std.debug.panic("Unexpected EOF", .{}),
        };
        // led loop added later
        return left_ptr;
    }

    fn next_tok(p: *Parser) Token {
        p.tok_i += 1;
        return p.tokens[p.tok_i - 1];
    }

    fn peek_tok(p: Parser) Token {
        return p.tokens[p.tok_i];
    }
};
```

## Binary expressions

Moving on to binary expressions. A token is now either an atom or an operator. Nodes now have
optional arguments, which are pointers to their child nodes. Every node is labeled with a token
whose arguments, if any, are its subtrees. This is the distinction between an atom and a binary
expression.

After parsing an atom, instead of returning immediately, we check if the next token is an operator.
If it is, then we parse the right-hand-side argument, heap-allocate another node to hold this infix
expression, and that is what we return instead.

```zig
// 1_parser.zig
const Node = struct {
    ch: char,
    args: ?Arguments,

    const Arguments = struct {
        left: ?*Node,
        middle: ?*Node,
        right: ?*Node,
    };
};
```

```zig
// led: after the atom, if the next token is an operator, build an infix node
while (p.peek_tok() != .eof) {
    const op = p.peek_tok().op;
    if (BindingPower.infix(op)) |bp| {
        if (bp.left < bp_min) break;
        _ = p.next_tok();

        var args: Node.Arguments = .{ .left = left_ptr, .middle = null, .right = undefined };
        const right_ptr = try p.parse_node(bp.right);
        args.right = right_ptr;

        const infix_ptr = try p.allocator.create(Node);
        infix_ptr.* = .{ .ch = op, .args = args };
        left_ptr = infix_ptr;
        continue;
    }
    break;
}
```

## Encoding the AST as a string

To encode these binary expressions, the operator comes first, followed by the left and right
argument, and then we enclose these in parentheses. These are the rules for encoding an AST. The
notable ones are that the order of arguments in the tree is preserved — so, left to right — to be
consistent with prefix notation, wherein the operator precedes its arguments. If we settle for `(+
a, b)`, we may not use `a + b` as well. Lastly, syntactic tokens like parentheses must be present at
the beginning and end of a subtree string.

```zig
// 1_parser.zig
pub fn encode(a: Ast, allocator: Allocator) !string {
    var str = DynamicArray(char){};
    try encode_node(a.root.*, str.writer(allocator).any());
    return try str.toOwnedSlice(allocator);
}

fn encode_node(node: Node, writer: std.io.AnyWriter) !void {
    if (node.args == null) {
        try writer.writeByte(node.ch);
        return;
    }

    try writer.print("({c}", .{node.ch});
    if (node.args.?.left) |left| {
        try writer.writeByte(' ');
        try encode_node(left.*, writer);
    }
    if (node.args.?.middle) |middle| {
        try writer.writeByte(' ');
        try encode_node(middle.*, writer);
    }
    if (node.args.?.right) |right| {
        try writer.writeByte(' ');
        try encode_node(right.*, writer);
    }
    try writer.writeByte(')');
}
```

## Multiple infix expressions: binding powers

Next, we'll parse multiple infix expressions. To determine the priority between two operators, we
use binding powers. Binding powers are properties of operators, and there is a left and a right
binding power. In this case, the left binding power of an add operator is one and the right will be
two. If the left binding power of the current operator is less than the right binding power of the
previous operator, then it will not take priority over the previous expression.

Create a function that returns the binding power of a given operator. Change `parseNode` to take in
a minimum binding power. Function `parse` will start the recursion by giving a minimum binding power
of zero. After parsing an atom, if the next token is an operator, evaluate the binding power of the
next operator. Compare the left binding power of the next operator to the minimum binding power of
the current function scope. If it's greater than the minimum, then we continue parsing by
recursively calling `parseNode`, passing in the right binding power of the next operator.

```zig
// binding_power.zig
left: u8,
right: u8,

pub fn infix(op: char) ?BindingPower {
    return switch (op) {
        '=' => .{ .left = 2, .right = 1 },
        '?' => .{ .left = 4, .right = 3 },
        '+', '-' => .{ .left = 5, .right = 6 },
        '*', '/' => .{ .left = 7, .right = 8 },
        else => null,
    };
}
```

```zig
// 1_parser.zig — recursion starts at minimum binding power 0
fn parse(p: *Parser) !void {
    p.root = try p.parse_node(0);
}

// led: only keep climbing while the operator binds tighter than bp_min
if (BindingPower.infix(op)) |bp| {
    if (bp.left < bp_min) break;
    _ = p.next_tok();
    // ...
    const right_ptr = try p.parse_node(bp.right);
    // ...
}
```

## `nud`, `led`, and parser state

The variable `left` may be consulted by the code of the next token, which will use the value of
`left` as either the translation or value of the left-hand argument, depending on whether it is
translating or interpreting. If a token is preceded by an expression, we call the code denoted by
that token `led`; and without a preceding expression, we call it `nud`. Our `while` loop is a `led`,
and the code before that would be the `nud`.

We shall also change our strategy when asking for a right-hand argument, making a recursive call of
the parser itself rather than of the code of the next token. In making this call, we supply the
binding power associated with the desired argument, which we call the right binding power, whose
value remains fixed as this incarnation of the parser runs. The left binding power is a property of
the current token in the input stream and in general will change each time state Q1 is entered. The
left binding power is the only property of the token not in its semantic code.

To return to Q0, we require right binding power to be less than left binding power. If this test
fails, then by default the parser returns the last value of `left` to whoever called it, which
corresponds to A getting E and AE if A had called the parser that read E. If the test succeeds, the
parser enters state Q0, in which case B gets E instead.

```zig
// 1_parser.zig — nud = code for the first token; led = the while loop
fn parse_node(p: *Parser, bp_min: u8) !*Node {
    var left_ptr = try p.allocator.create(Node);
    left_ptr.* = switch (p.next_tok()) { // nud
        .atom => |ch| .{ .ch = ch, .args = null },
        .op => |op| // ... prefix / parentheses
        .eof => std.debug.panic("Unexpected EOF", .{}),
    };

    while (p.peek_tok() != .eof) { // led
        const op = p.peek_tok().op;
        // postfix / infix handling, guarded by binding power
    }

    return left_ptr;
}
```

## Precedence and associativity

Without changing the logic, adding another operator with the same binding power defaults to left
associativity. Adding multiplication and division is easy: just set their binding powers higher so
they take priority over addition and subtraction. For right associativity, make the left binding
power higher than the right binding power.

This approach is called operator precedence. A number should be associated with each argument
position by means of precedence functions over tokens. These numbers are sometimes called binding
powers. The idea is to assign data types to classes and then totally order the classes. We now
insist that the class of the type at any argument that might participate in an association problem
not be less than the class of the data type of the result of the function taking that argument.
Finally, we adopt the convention that when all four data types in an association are in the same
class, the association is to the left.

```zig
// binding_power.zig
// higher numbers bind tighter: * / take priority over + -
'+', '-' => .{ .left = 5, .right = 6 },
'*', '/' => .{ .left = 7, .right = 8 },
// left > right => right associative (e.g. assignment)
'=' => .{ .left = 2, .right = 1 },
```

## Prefix operators

Let's add a prefix operator to denote the sign of a number, negative or positive. Now the left and
right arguments can be null independent of each other. Give prefix operators the highest priority by
making the right binding power the highest among all operators. Set the left binding power to zero,
since this is a one-sided argument. Update `parseNode` to handle an atom or an operator on the first
token.

```zig
// binding_power.zig
pub fn prefix(op: char) BindingPower {
    return switch (op) {
        '-' => .{ .left = 0, .right = 9 }, // one-sided: left = 0, right is highest
        else => std.debug.panic("Invalid prefix op: {c}", .{op}),
    };
}
```

```zig
// 1_parser.zig — nud: the first token may itself be an operator (prefix)
.op => |op| blk: {
    switch (op) {
        // ...
        else => {
            const bp = BindingPower.prefix(op);
            const right_ptr = try p.parse_node(bp.right);
            const args: Node.Arguments = .{ .left = null, .middle = null, .right = right_ptr };
            break :blk .{ .ch = op, .args = args };
        },
    }
},
```

## Postfix operators

It's the same idea for postfix operators, but we update the infix and postfix functions to return an
optional, since an operator can be either of these two during our `while` loop. A minor mistake
here: postfix should be checked first, before infix. It's fine in our case since there's no overlap
between infix and postfix operators — unlike prefix and infix, which both have the minus operator.

```zig
// binding_power.zig
pub fn postfix(op: char) ?BindingPower {
    return switch (op) {
        '!' => .{ .left = 11, .right = 0 },
        '[' => .{ .left = 13, .right = 0 },
        else => null,
    };
}
```

```zig
// 1_parser.zig — led: check postfix before infix (both return optionals)
if (BindingPower.postfix(op)) |bp| {
    if (bp.left < bp_min) break;
    _ = p.next_tok();

    var args: Node.Arguments = .{ .left = left_ptr, .middle = null, .right = null };
    // ...
    const postfix_ptr = try p.allocator.create(Node);
    postfix_ptr.* = .{ .ch = op, .args = args };
    left_ptr = postfix_ptr;
    continue;
}
```

## Array indexing

Let's now support array indexing, which is a postfix binary operator. Simply add an `if` statement
in the postfix evaluation. After parsing the right argument, we assert that the next token is the
closing bracket.

```zig
// 1_parser.zig — inside the postfix branch
if (op == '[') {
    const right_ptr = try p.parse_node(bp.right);
    assert(p.next_tok().op == ']');
    args.right = right_ptr;
}
```

## Parentheses

Supporting parentheses is the same concept, but this time we always call `parseNode` with a minimum
binding power of zero. This ensures that the expressions inside the parentheses take priority over
everything else. The `nud` of left parenthesis will call the parser and then simply check that the
right parenthesis is present, and advance the input.

```zig
// 1_parser.zig — nud of '(' parses a fresh expression at bp_min = 0
'(' => {
    const new_left = try p.parse_node(0);
    assert(p.next_tok().op == ')');
    break :blk new_left.*;
},
```

## Ternary operators

Adding ternaries takes a little more effort, but it's the same concept. Update the node arguments
with a new middle field. Now, this is a combination of our delimiter examples. Parse the expression
between the question mark and the colon operator with a minimum binding power of zero. Assert the
next token to be the colon, and then parse the right argument.

One fundamental aspect of Pratt parsing is to ensure that all arguments are separated by another
token. Now, this is already intrinsic to binary operators like addition and subtraction. But when it
comes to variable-argument operators like the ternary operator, which has three arguments, we need
to ensure that each argument is delimited by another token. In this case, the colon operator
separates the middle and the last argument.

```zig
// 1_parser.zig — inside the infix branch; '?' adds a middle argument
var args: Node.Arguments = .{ .left = left_ptr, .middle = null, .right = undefined };
if (op == '?') {
    const middle_ptr = try p.parse_node(0);
    assert(p.next_tok().op == ':');
    args.middle = middle_ptr;
}
const right_ptr = try p.parse_node(bp.right);
args.right = right_ptr;
```

## Operator grammars

When a token has more than two arguments, we lose the property of infix notation that the arguments
are delimited. This is a nice property to retain — partly for readability, partly because
complications arise (for example, if minus is to be used as both an infix and a prefix operator).
Left parenthesis also has this property: as an infix, it denotes application; as a prefix, it's a
no-op. Accordingly, we require that all arguments be delimited by at least one token. Such a grammar
Floyd calls an operator grammar.

For example, the `nud` of `if`, when encountered in the context of `if a then b else c`, may call
the parser for `a`, verify that `then` is present, advance, call the parser for `b`, test if `else`
is present, and if so, advance and call the parser a third time.

## Final test

Now let's use the expression from our intro as our final test. These are six properties of Pratt
parsing, which summarize the implementation details of the paper.

```zig
// 1_parser.zig
test "final" {
    var arena = std.heap.ArenaAllocator.init(std.testing.allocator);
    const allocator = arena.allocator();
    defer arena.deinit();

    try test_parse(
        allocator,
        "x[i] = -1! + (2 - 3) * 4 ? z : 0 = y",
        "(= ([ x i) (= (? (+ (- (! 1)) (* (- 2 3) 4)) z 0) y))",
    );
}
```

## Optimization

Okay, now let's have some fun — let's optimize our little Pratt parser. Currently, each node takes
up 40 bytes on 64-bit machines and 20 bytes on 32-bit machines. Let's confirm this by asserting the
size at compile time.

Move the binding power and lexer structs to separate files; we won't be touching these as we try
different implementations of the parser. Create a text file that contains all input from our tests.
In `main.zig`, parse each line 10,000 times using an arena allocator. We accumulate all the memory
through each iteration, and we only free memory on program exit. Let's benchmark our program — the
most important metrics are wall time and peak RSS (peak memory).

```zig
// 1_parser.zig — assert the node size at compile time
const Node = struct {
    ch: char,
    args: ?Arguments,

    comptime {
        if (builtin.cpu.arch == .x86) assert(@sizeOf(Node) == 20);
        if (builtin.cpu.arch == .x86_64) assert(@sizeOf(Node) == 40);
    }
    // ...
};
```

```zig
// main.zig — parse every line of mock.txt 10,000 times under one arena
pub fn main() !void {
    var arena = std.heap.ArenaAllocator.init(std.heap.page_allocator);
    defer arena.deinit();
    const allocator = arena.allocator();

    const max_size = 36;
    var input = try DynamicArray(char).initCapacity(allocator, max_size);

    const mock = try std.fs.cwd().openFile("src/mock.txt", .{});
    var buf = std.io.bufferedReader(mock.reader());
    var reader = buf.reader();

    const iterations = 10000;
    while (reader.streamUntilDelimiter(input.writer(allocator), '\n', null)) {
        for (0..iterations) |_| {
            const ast = try Ast.init(allocator, input.items);
            _ = try ast.encode(allocator);
        }
        input.clearRetainingCapacity();
    } else |err| switch (err) {
        error.EndOfStream => {},
        else => return err,
    }
}
```

### Slices instead of pointers

Using a slice instead of individual pointers for each argument brings down our memory from 40 bytes
down to 24. Just a simple change, and we've reduced our memory usage in half.

```zig
// 2_parser_slice.zig
const Node = struct {
    ch: char,
    args: ?[]const Node,

    comptime {
        if (builtin.cpu.arch == .x86) assert(@sizeOf(Node) == 12);
        if (builtin.cpu.arch == .x86_64) assert(@sizeOf(Node) == 24);
    }
};
```

```zig
// args become a single allocated slice instead of separate pointers
var args = try p.allocator.alloc(Node, args_n);
args[0] = left;
if (middle) |m| args[1] = m;
args[args_n - 1] = right;
left = .{ .ch = op, .args = args };
```

```zig
fn encode_node(node: Node, writer: std.io.AnyWriter) !void {
    if (node.args) |args| {
        try writer.print("({c}", .{node.ch});
        for (args) |a| {
            try writer.writeByte(' ');
            try encode_node(a, writer);
        }
        try writer.writeByte(')');
    } else try writer.writeByte(node.ch);
}
```

### Index-based arrays

Now, instead of using pointers, let's manually manage our heap-allocated nodes in a dynamic array.
To keep track of the children of a node, we need to use another dynamic array. The `args` field is a
dynamic array of indexes, and these are indexes into the `nodes` field. What this does is constrain
our previously-used pointers to four-byte integers as indices. Now each node only takes up eight
bytes of memory on both 32-bit and 64-bit systems.

Benchmarking our optimization against the slice parser: it uses the same amount of memory with worse
performance. We have to remember that, even though each node only takes up eight bytes of memory,
we've also been allocating dynamic arrays to keep track of our nodes. And in my implementation, I
appended all nodes to the `args` dynamic array. This meant that each node was actually taking up 12
bytes of memory, on top of the overhead of creating dynamic arrays across 10,000 iterations.

```zig
// 3_parser_dod.zig
const Node = struct {
    ch: char,
    args_n: u8,
    args_start: Parser.Index,

    comptime {
        if (builtin.mode == .ReleaseFast) assert(@sizeOf(Node) == 8);
        if (builtin.mode == .Debug) assert(@sizeOf(Node) == 12);
    }
};

const Parser = struct {
    allocator: Allocator,
    tokens: []const Token,
    tok_i: usize,
    nodes: DynamicArray(Node), // every node lives here
    args: DynamicArray(Index), // child indices live here

    const Index = union {
        nodes: u32,
        args: u32,
    };

    fn nodes_append(p: *Parser, node: Node) !Index {
        try p.nodes.append(p.allocator, node);
        return .{ .nodes = @intCast(p.nodes.items.len - 1) };
    }

    fn args_append(p: *Parser, i: Index) !Index {
        try p.args.append(p.allocator, i);
        return .{ .args = @intCast(p.args.items.len - 1) };
    }
};
```

### Eliminating the second dynamic array

Let's see if we can keep track of the child nodes without using a second dynamic array. Looking at
the actual indices of the child nodes, we can see that they're all next to each other. So we only
need to keep track of the first argument, and we can calculate an offset for the rest of the
arguments instead of memorizing it.

One caveat: in ternary expressions, if the middle argument is its own subtree instead of an atom,
then the third argument of the ternary will not be next to the first two. So we can't eliminate the
second dynamic array, but we'll only use it to track the arguments of the ternary operator. We only
append the first argument and the last argument, and we can calculate the index of the second
argument just like the rest of the operators.

Now it uses 38% less memory than the slice implementation of the parser, albeit with 10% slower
performance.

```zig
// 4_parser_final.zig
const Node = struct {
    ch: char,
    args_n: u8,
    // if args_n == 3 (ternary): args[args_start.args] = first arg,
    //   first + 1 = second, args[args_start.args + 1] = third
    // else: nodes[args_start.nodes] = first arg, + 1 = second, ...
    args_start: Parser.Index,
};
```

```zig
// only the ternary still needs the args array; every other operator is contiguous
const start_i = try p.nodes_append(left);
var start = start_i;

var args_n: u8 = 2;
if (op == '?') {
    _ = try p.parse_node(0);
    assert(p.next_tok().op == ':');
    args_n += 1;
}
const last_i = try p.parse_node(bp.right);

if (args_n == 3) {
    start = try p.args_append(start_i);
    _ = try p.args_append(last_i);
}
left = Node.init(op, args_n, start);
```

All right — hope this helps!
