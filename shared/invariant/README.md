# invariant

`shared/invariant` is the pure assertion engine. `shared/invariant/default` supplies the
OS-backed recorder and the package-level sugar used by application code.

## Eager guards

`Always(condition, message)` is deliberately bare and eager. A false condition panics at
its callsite; registration seeds its literal message so an unreached guard is also visible as a
coverage gap.

## Deferred assertion chains

An assertion set is one fluent expression:

```go
invariant.Assertions("queue").
    Sometimes(queue.Empty(), "The queue is empty.").
    Sometimes(queue.Full(), "The queue is full.").
    Ensure()
```

`Sometimes` records neither coverage nor failure itself. It captures one boolean by fluent
ordinal; `Ensure` resolves the registration-owned plan and credits exactly one of the axis's
independent true or false obligations. Repeated messages are safe because identity is namespace,
ordinal, and message.

There is no cross product. The suite witnesses every individual branch, not every combination, and
there are no tuple cells, carves, polarity references, combination gaps, or legends.

## Integer domains

Every concrete integer width has chain methods for bounded and enumerated domains:

```go
invariant.Assertions(namespace).
    Range_Int(value, Minimum, Maximum, holes...).
    Ensure()

invariant.Assertions(namespace).
    Enum_Uint8(value, State_New, State_Ready, State_Done).
    Ensure()
```

Range enforces both bounds and strictly-interior holes at `Ensure`. Its coverage includes successful
lower and upper guards, both interval boundaries when distinct, and eligible interior 0, 1, 2, and
-1 witnesses. A boundary cannot be excluded.

Enum enforces membership at `Ensure`, requires at least two distinct members, seeds one successful
membership guard, and gives each distinct member an independent true/false axis.

## Type-owned helpers

A defined type keeps its contract beside the type in a trailing-namespace helper:

```go
type Token string

func Token_Invariants(token Token, namespace invariant.Namespace) {
    invariant.Assertions(namespace).
        Sometimes(len(token) == 0, "The token is empty.").
        Ensure()
}
```

Registration instantiates that template at each literal callsite namespace. A chain is one unsplit,
nonempty call nest ending in `Ensure`; returning a half-built builder is unsupported.

## Coverage lifecycle

Registration parses the selected packages before the suite and seeds every individual obligation.
It also publishes the immutable ordered handles runtime emission consumes; runtime never rebuilds an
identity from a message. A failed `Ensure` preflights the whole plan and credits nothing.

Plain tests and fuzz processes record. Benchmarks and ordinary binaries enforce without recording,
and ordinary binaries do not consult registration plans. The `noassert` build tag compiles the same
surface with inert bodies for explicit unchecked builds.
