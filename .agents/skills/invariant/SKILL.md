---
name: invariant
description: >
  Load BEFORE you write an `_Invariants` bundle or add an assertion in this repository. An
  assertion composes from the top down, not from the bottom up. The types under one root make a
  tree, never a DAG, thus two types with the same properties are intentional. A bundle is an
  identity, and registration never shares one. A constant is a fact, and it carries the property
  that two such types share.
---

# The composition model of shared/invariant

The full rules are in `shared/invariant/SPECIFICATION.md`.

## Composition runs from the top down

Object-oriented code composes from the bottom up. You make small parts, then you assemble them,
and the whole is the sum of its parts.

An assertion goes the other way. The entrypoint holds the widest type. Thus its bundle states the
whole program at one time, and each more specific invariant sits below it. Depth gives precision,
not breadth. A deep function states one narrow leaf, and the entrypoint states the full tree.

The namespace shows this direction. One root writes the namespace one time, and each bundle sends
it down unchanged.

## The shape is a tree, never a DAG

A type names its position in the descent. Thus one type at two positions under one root gives one
name to two obligations, and evidence from one position satisfies the other. Registration rejects
that shape.

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

## Two types with the same properties are intentional

One type cannot sit at two positions under one root. Thus you will declare several types that hold
the same properties, only to give each position its own name. The harness demands this, and it is
correct.

State the shared property through the constants. Each type keeps its own bundle, and each of those
bundles names the same bounds. When two types share a full value set, make one constant name the
other.

The duplication then costs one bundle for each type. The fact stays at one site, thus one edit
keeps every type correct.
