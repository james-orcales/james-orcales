---
name: invariant
description: >
  Load BEFORE you write an `_Invariants` function, add an assertion, or resolve an invariant
  coverage gap in this repository. An `_Invariants` function holds the set of invariants for one
  type, and it can call other `_Invariants` functions. Each type must occur one time in one
  chain, and each chain is separate from the others. State a property that two types share
  through the same package-level constants. A large quantity of diagnostics is the usual result,
  thus the Quickstart gives the steps that start the work.
---

# Many diagnostics are usual

The framework is not easy, and it gives a large quantity of diagnostics. That quantity is the
usual result.

# _Invariants

An `_Invariants` function holds the set of invariants for one type. It can call other
`_Invariants` functions. Each type in one chain must occur one time. Each chain is separate from
the others, thus one type can be part of many chains.

# Composition

You will find yourself declaring types with duplicated `_Invariants` declarations. To state a
property that they share, use the same package-level constants in each of them.

# Reuse constants

The framework resolves each constant expression at compile time. Reuse constants as much as
possible, especially from other packages, referencing them by variable instead of inlining the
literal. Then one change to a load-bearing constant goes to each place that uses its assumptions.

# Quickstart

When you cannot start, do these steps first:

- Correct each bound that is not applicable, especially one set to the minimum or the maximum
value of the primitive type.
- Push each `if` up and each `for` down. A branch that moves up the stack decreases the quantity
of diagnostics.
- Remove each unused struct field and each unused function parameter.
- Replace a large struct in a parameter or an embedded position when the callsite uses only some
of its fields.
- Return no value where you can, because a returned value requires a deferred assertion.
- Delete dead code.

# Reading diagnostics

The report separates two kinds of gap, and each one has a different correction:

- An axis that ran and holds one polarity must have a different **value**. Its bound is too wide,
or no input gets to the other branch.
- An axis that never ran must have a different **path**. No test gets to that code.

Each Range also reports its declared interval against the interval that the run observed. An
observed interval far inside the declared one names a type that is too broad, and no absent
branch can show that.

The report starts with one row for each namespace that holds a gap, the largest first. Above 40
gaps it goes to a file. `INVARIANT_OUTPUT=table` or `INVARIANT_OUTPUT=json` selects the form.

# Handling unvalidated input

At each boundary that takes unvalidated input:

1. Make two types for the data, `Foo_Unvalidated` and `Foo`.
2. The two types' invariants should NOT share the same constants.
3. Write one function that changes the first type into the second.
4. Centralize graceful error handling in that function.

```go
func Foo_Validate(foo_unvalidated Foo_Unvalidated) (foo Foo, err Foo_Validation_Error)
```

# Gotchas

- A Range must have at least five legal values. For two to four, use the matching Enum. For one,
use a direct `Always` equality.
- A bundle for a defined Boolean type states exactly one `Sometimes`. Any other link, a second
link, or no chain at all is a fatal error.
- A test must not call an assertion writer or an invariant helper. Get to the assertion through a
registered production entry point.
- A type alias is a fatal error. A defined type holds an identity, and a second name for a type
must have one.
- A constant condition in `Always` is a fatal error, because it states no property.
- One `Always` message and one namespace literal each occur one time in the full registration.
- A bundle body is straight-line. A branch or a loop makes its assertion set conditional.
- In a bundle body, a bundle call uses the `Namespace` parameter. A string literal there is a fatal
  error.
