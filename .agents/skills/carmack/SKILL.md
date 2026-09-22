---
name: carmack
description: Load BEFORE you extract a helper function, split a long function, add a conditional that skips work, or write a method that mutates its receiver. Also load it when a bug comes from a caller whose assumed state was wrong, or when execution order across subsystems is unclear. Inline a function that has one caller, consolidate a function that has many callers into one controlled point, do the work always and inhibit the result, and move each block toward purity. Source: John Carmack, 2007 email on inlined code, 2012 article on functional programming in C++, 2014 follow-up (https://cbarrete.com/carmack.html).
---

# Carmack: inline code, then make it pure

The real defect is unexpected dependency and unexpected state mutation. Inlining makes every
mutation visible in sequence. Purity removes the mutation. Use both.

## Organization

Three styles exist:

- A: minor functions first, then the major function that calls them.
- B: the major function first, then the minor functions.
- C: everything inline in the major function, each former subroutine marked by a comment block and a
  bare braced scope.

Prefer C. The reason is process, not speed: avoiding a call does not make code faster by itself.

- Sequential operations appear sequentially. Do not split a function only because it is longer than
  a page or two.
- Put work that runs once per tick in the outermost loop. Do not bury it in a deep call chain that
  a branch can skip.
- A flat sequential function makes execution order obvious. Deep nesting across subsystems silently
  adds a frame of latency.
- A function that does not exist cannot be called as a shortcut that skips required work.
- Use comment blocks and braced scopes so a reader can navigate and an editor can fold.

## Rules

1. A function with one caller: inline it.
2. A function with many callers: consolidate the work into one place, set a flag at each former call
   site, and act on the flag at one controlled point. One check for `health <= 0 && !killed` in the
   think loop beats `KillPlayer()` called from twenty sites.
3. Many variants of one function: replace them with one function that takes defaulted parameters.
4. A function that is almost pure: make it fully pure.
5. A function that must be shared: make its parameters and the function `const`.
6. Minimize control flow and the amount of code under a conditional. Prefer one execution path and
   consistent timing.

## Do it always, then inhibit

Run an operation unconditionally and ignore or suppress its result. Do not skip it conditionally.
A skipped operation often also updated state that something else needed. Worst-case time matters
more than average time, so a variable tick cost is a defect.

Exception: power- or thermal-limited hardware, where unconditional work costs battery and heat.

## Purity

A pure function reads only its parameters and returns its result. It does not read or write global
state, keep internal state, do IO, or mutate its inputs.

- Purity is a continuum. Each step toward it pays, even if the function never arrives.
- A function that touches a few globals: pass them in as parameters. More parameters is a fair
  price. Too many parameters signals a refactor.
- Tangled global state is the worst case. Impose discipline there before you polish leaf math.
- A function is effectively pure if its side effects do not escape.
- Pure code is thread safe, reusable, testable with plain input/output pairs, and easier to analyze.

When many state changes must happen, keep them inline so their full scope is visible. When that
becomes too much, factor blocks into pure functions and keep them from drifting back to impurity.

## Objects

- A non-`const` method is impure. A method on a large object hides what it modifies.
- A constructor can and must be pure.
- A utility object returns a new copy. Provide `Normalized()`, not only `Normalize()`.
- Make large heavyweight objects the unit of grouping. Minimize medium-sized helper objects.
- Prefer explicit loops and discrete `x`/`y`, `width`/`height` variables over small arrays and
  copy-pasted vector code.

## Performance limits

- Mutation in place is fastest. Purity costs copies.
- Do not copy a large buffer to stay pure (a `DrawTriangle()` that returns a new framebuffer).
- Do not append by copying a whole container.
- An output reference is acceptable when return-by-value is unreliable.
- Parallel code already copies, so purity costs less there and pays more. Entities read a shared
  read-only world state and write their updates at the end of the tick.

## Actions

1. For each non-trivial function, list all external state it reads and writes. The list is
   documentation even if nothing else changes.
2. Start each new task by naming the real computation. Pass its inputs to a pure function, then act
   on the result.
3. While debugging, look for hidden state.
4. Make nearly every non-iterator variable `const`.
5. Step through one full tick in a debugger, through every function. It shows the code that nobody
   reads.
