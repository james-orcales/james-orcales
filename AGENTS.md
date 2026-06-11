## Test-Driven Development

**STRICTLY WRITE THE FAILING TEST FIRST, THEN THE CODE TO MAKE IT PASS. WHITEBOX TESTS ARE BANNED.**

## Comments

Comment aggressively, but only the WHY — never the WHAT. Assert it instead when possible.

The code shows what it does; restating it is noise. A comment earns its place by explaining what
the code cannot: why this approach over the alternative, why a constraint or invariant exists, why
a value is what it is, why something that looks removable is load-bearing.

## Planning

Before finalizing a plan in plan mode, ask the user at least 20 questions.

## Naming

Name a thing for what it is. Not a use case it serves, not a behavior it performs — what it *is*.
`Buffer`, not `RequestBuffer`. `Clock`, not `TimeoutClock`. The narrower name is a lie the moment
a second caller appears.

Use snake_case and Ada_Case.

## Linting

Do not side step the linter. Make the fundamental code change it's asking from you.

