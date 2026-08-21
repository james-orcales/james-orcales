
# Equality

Equal, Not_Equal, Equal_Values, Not_Equal_Values, Exactly, Same, and Not_Same
report structural, coerced, or pointer equality, reporting a mismatch through t and
returning whether it held.

# Nil And Truth

Nil, Not_Nil, True, False, Zero, and Not_Zero report nil-ness, a boolean, and
whether a value equals the zero value of its type.

# Emptiness And Count

Empty and Not_Empty treat zero values and empty collections as empty. Count reports
whether a value's element count equals a wanted number.

# Collections

Contains and Not_Contains search a slice; Contains_Any and Not_Contains_Any also
span strings and map keys; Subset, Not_Subset, Elements_Match, and Not_Elements_Match
test containment and unordered set-equality.

# Errors

No_Error and Error assert a nil or non-nil error; Equal_Error and Error_Contains
match its message; Error_Is, Not_Error_Is, Error_As, and Not_Error_As wrap the
standard-library error-chain helpers.

# Types

Implements and Not_Implements report whether a value's type satisfies an interface
named by a nil interface pointer; Is_Type and Is_Not_Type compare dynamic types.

# Ordering

Greater, Greater_Or_Equal, Less, and Less_Or_Equal compare two ordered values;
Positive and Negative compare a value against its type's zero.

# Panics

Panics and Not_Panics report whether a function panics; Panics_With_Value and
Panic_With_Message additionally match recovered value or formatted panic message.

# Approximation And Time

In_Delta and In_Epsilon bound the absolute or relative error between fixed-point
numbers, with slice and map variants; Within_Duration and Within_Range bound a
moment against another or a range.

# Regexp And JSON

Regexp and Not_Regexp report whether a pattern matches a string; JSON_Eq reports
whether two JSON documents are semantically equal.

# Allocation

Zero_Allocation warms callback once, measures one execution, and passes only
when callback performs zero heap allocations. Parallel tests cannot call it because
Go allocation measurement temporarily changes GOMAXPROCS.

# Control

Condition reports the result of a caller predicate. Every failed assertion, including
Fail and Fail_Now, reports the failure and aborts the test.

# Predicates

Objects_Are_Equal, Objects_Are_Equal_Values, Is_Nil, Is_Empty, Same_Pointers,
Contains_Element, and Diff_Lists are the pure decisions the assertions build on,
taking no test handle so they can be called and tested directly.

# Files

Asserter_File_Exists, Asserter_No_File_Exists, Asserter_Directory_Exists, and
Asserter_No_Directory_Exists report whether the Asserter's File_System holds a file
or directory at an operating-system path.

# Eventually And Never

Asserter_Eventually and Asserter_Never poll a condition on the Asserter's io loop
and report through t if it never or ever holds before the wait elapses; both are
asynchronous, so the caller drives the loop.
