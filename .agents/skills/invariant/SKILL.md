---
name: invariant
description: >
  Load BEFORE you write an `_Invariants` bundle, add an assertion, or resolve an invariant
  coverage gap in this repository. Minimize the assertion tree before you add types or bundles.
  A bundle names each bundle it composes and never learns what composed it, the reverse of
  inheritance. The types under one root make a tree, never a DAG. A bundle is an identity, and
  registration never shares one. A constant is a fact that two distinct types both name.
---

# The composition model of shared/invariant

The full rules are in `shared/invariant/SPECIFICATION.md`.

## Test

Run each Go test suite with `-timeout=10s`. Always aim to REDUCE the coverage gaps.

## Choose the smallest assertion tree first

The harness checks the tree that the code declares. It does not prove that each node is
necessary. Each named helper that states an invariant creates another proof boundary.

Do not add a type or a bundle only because a coverage gap exists. A gap can identify a missing
production witness, a boundary that is too broad, an unnecessary helper, or the wrong state
owner.

Before you add a type or a bundle:

1. Classify the change as a storage fact, a semantic identity, or a coverage obligation.
2. State each shared storage fact through the same constants. Do not add a type for a bound or a
primitive width.
3. Identify the function that owns the state change. Keep a value local when it has no separate
domain identity.
4. Remove a named helper when it only reports a local predicate to that owner.
5. Narrow a boundary when real values cannot reach its declared property.
6. Add another type only when an unavoidable root needs another position or another value set.

Do not confuse proof boundaries with domain identities. Two function namespaces need separate
evidence, but that fact does not require two types. First remove unnecessary boundaries. Then
model the tree that remains.

## A parent knows its children, a child knows no parent

Object-oriented inheritance points knowledge upward. A subtype names its base type, and the base
type never learns which types extend it. The set of implementors stays open, thus a base type
cannot state what its subtypes hold.

An assertion points knowledge downward. A bundle names each bundle it composes, and a composed
bundle never names what composed it. The set is closed and each root knows it in full.

Two consequences follow, and both are the reason for the rest of this document:

1. A root states its whole tree. The entrypoint holds the widest type, thus its bundle reaches
every obligation below it. Depth gives precision, not breadth.
2. A bundle carries no context. It cannot know its position, thus it cannot state anything about
the situation that composed it.

The namespace shows the same direction. One root writes it one time, and each bundle sends it down
unchanged.

## The shape is a tree, never a DAG

A bundle carries no context, thus its type is its position and nothing else. One type at two
positions under one root gives one name to two obligations, and evidence from one position
satisfies the other. Registration rejects that shape.

A diamond is the usual form: two branches that both reach one leaf type. A repeated sibling and a
cycle fail for the same reason.

The constraint applies to each root, not to the program. One type can sit in many bundles and many
trees. The type must be unique in each chain that contains it.

## A bundle is an identity, a constant is a fact

Object-oriented code carries structure and vocabulary in one act. Thus reuse is cheap, and a new
name is expensive. Here the two are separate, and the costs are the reverse.

A bundle that calls a second bundle does not borrow it. The call puts that bundle at one position
and makes one obligation there. Two callers of one bundle state that their two situations are one
obligation, and evidence from one satisfies the other. That statement is usually false. Thus a
bundle composes by position, and registration never shares one.

A constant holds no identity. Any number of types can name one constant at no cost.

**Duplicate the bundle. Share the constants.** Reuse of a bundle is a coverage decision, never a
decision about code economy.

## Duplicate types only for unavoidable positions

One type cannot sit at two positions under one root. When an unavoidable root contains two
distinct positions for one value set, declare separate types. Do not duplicate a type only because
two functions report separate coverage gaps.

State the shared property through the constants. Each type keeps its own bundle, and each of those
bundles names the same bounds. When two types share a full value set, make one constant name the
other.

The duplication then costs one bundle for each type. The fact stays at one site, thus one edit
keeps every type correct.
