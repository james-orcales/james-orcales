# Implementation-Driven Testing

Throughout this video, you're going to hear birds chirping, cars, and dogs barking. It's 6:00 a.m.,
so people are starting to wake up — the world is starting to make noise. Anyway, a start.

As software integrates deeper with our everyday lives, its complexity grows. To address the modern
tech landscape's deficiencies in software reliability, test-driven development rose to popularity
through the decades. However, this philosophy always felt unnatural to me — like there was something
missing. It tackled software development top-down, while I gravitated toward a ground-up approach.
Implementation-driven testing paved me a clear path forward in developing software. I'll discuss
what it is and examples of how to do it. By the end, we'll see how to reconcile the two
philosophies.

Let me pull up the presentation. Screen, Firefox. There you go.

*Disclaimer: conceptual definitions in this video are my own words unless an external source was
cited explicitly.*

## What is a function?

A function is a set of instructions to be executed by the computer. It may or may not have input
parameters; it may or may not return values. You can categorize functions based on purity, purpose,
among others.

## Functions by purity

There are two: pure and impure functions. A function is impure if it has a side effect or an
external dependency.

A **side effect** is an observable change caused by a function beyond its local scope. Universally,
this would be I/O operations like writing to a file, or mutating a global variable or shared state
like a slice.

An **external dependency** is a resource besides a function's input parameters that can influence
its output. Again, these can be I/O operations or global variables that you use in the computation
of the return value, or against shared state.

Now, depending on your problem domain, resources that we assume are outside the impurity scope can
be considered part of the side effects and external dependencies. For example, if you're doing
low-level programming, heap allocations can be considered side effects or external dependencies,
because if your function creates heap allocations, there is a real risk that your program can crash.

(When you see this terminal icon at the top right, that means right after the current slide there
will be a coding segment.)

### Demo: pure vs. impure functions

Let's go to Neovim — `demo.go`. I've created one pure function and two impure functions here. You
can tell this is a pure function because we only use the input parameters in calculating the return
value, and it's not mutating any global state. `impureAdd` is dependent on `foo` in calculating its
return value; `impureIncrement` mutates the global variable `foo`.

If you look at `func main`, I repeatedly call `pureAdd` with the same input arguments. I'll do the
same with `impureAdd`, while interweaving `impureIncrement` in between the print calls. So when we
run `demo.go`, `pureAdd` stays the same, while the return value of `impureAdd` differs — it
increases.

This phenomenon is what you call determinism. The function `pureAdd` is deterministic in that, given
the same input, it will always return the same output — it will always behave the same. Whereas with
`impureAdd`, even though we gave the same arguments three times, it returned different values. What
you need to know is that the purity of a function has major implications on how difficult it is to
test.

## Functions by purpose, and unit tests

Functions can be categorized by purpose: data transformation, computation, I/O operations,
orchestration, testing/verification. Obviously, we'll be focusing on testing/verification.

A **unit test** is a type of function that verifies the expected behavior or output of some other
function based on predetermined input. The unit (being a function) serves as a specification in
test-driven development.

### Demo: unit tests and the fragility of impure tests

So with the same example, I've created unit tests already. A unit test simply calls the function
you're testing given a predetermined input, and then we have an expected output, and we match the
expected output with what we actually got. If there's a mismatch, then the test suite will fail. I
did the same thing with `impureAdd` here, and we'll comment out `impureIncrement` for now.

When we run `go test` — let's remove the `-count` flag for now... oh, right, because I didn't change
this to demo again. When we run this, the test passes. Now when we do `impureIncrement` in
`TestPureAdd`, the test will fail. This completely unrelated function has mutated global state, and
now our impure-function test is failing. This is the difficulty when it comes to testing impure
functions: you have to stabilize the host environment so you can consistently verify the output it's
creating.

Let's go to this commented command here, `t.Parallel()`. So this first function will run in parallel
now, and when we run the test, it passes again. And if we add `t.Parallel()` to the second test, now
it's failing again. And if we remove this — look, it's cached. So let's add the `-count` flag, so
it's always a cold build — and it's passing again. And if we remove it... it's failing.

## Test-driven development

In test-driven development, you use unit tests to create a specification for the new product feature
you're creating — the idea is you write the unit tests before you even write any implementation
code.

### Demo: writing the test first

Let's simulate this workflow by creating `TestDivide`. Actually, I'll just copy this function,
`add`, and turn it into `divide`. Let's remove `impureIncrement`, and we'll expect the quotient 3.

Okay, let's implement this function now: `func divide(dividend, divisor int) int`, return `dividend
/ divisor`. So now when we run this... oh, `pureDivide` — let's just change this to `divide` for
simplicity. Wait, why did it...? Right, because we had this `impureIncrement`. Okay, so now the test
is passing — our implementation is correct. We expected the quotient 3 out of the inputs.

This is the process that was popularized by test-driven development, and I see the merit in this
workflow. You want to get a feel for the API. You want to create a rough draft of it, and you're not
focusing on the internals, the details.

### README-driven development (Mitchell Hashimoto)

This is actually the same workflow that Mitchell Hashimoto has — he's the founder of HashiCorp. In
one of his interviews (yeah, I watch a lot of chess), he discussed how he starts with creating APIs
like CLI applications. Let me pull up the transcript. And he mentioned, yes, README-driven
development:

> One of the first things I did with every piece of software that I designed, then and now to this
> day, is what some people would call README-driven development. That word certainly didn't exist
> then, but I would actually — and I've talked about this publicly as well — create either a bash
> script or a make file or something, where I just pretended the software already existed. It didn't
> do anything; it just echoed output into the terminal, maybe with some sleeps to make it feel real.
> I did that with Terraform. So I could run through, like, `terraform plan`, `apply` — "I'm running
> some stuff" — and I would just sit there, usually on an airplane at that time in my life when the
> internet was bad. And I would just sit there and pretend, put myself in that environment, run
> these commands, and then gauge how I felt: was it fun to use the software? Was it intuitive what
> the next step was? What sort of flags am I going to need? What sort of functionality do I think
> would exist? And that gave birth to what we would then back up into the technology from there.

So there is some merit to the workflow that test-driven development champions, where you're simply
feeling out the API and not caring much about how it actually works.

## Coverage reports

Common in test-driven development, especially when you're on the job, you'd have coverage reports.
**Coverage reports** are a quantitative metric measuring the total lines of code executed during a
test run. It's often used as a proxy for test comprehensiveness, and oftentimes treated as a key
performance indicator for the reliability of your software.

### Demo: coverage, mocking, and gaming the metric

Let's go to Ghostty again. We already have our unit tests here. It's simple to get the coverage
report for `go test` runs: you simply provide the `-cover` flag. And we have 30% code coverage.
Actually, let me remove the main function. And why do we only have 80% code coverage?

Ah, yes — one thing I forgot to mention during the unit test section is that, in order to test
impure functions properly, you'd have what's called **mocking**, which is our way of stabilizing the
host environment. So if we run the test again, it should fail. There you go. Mocking is simply
creating a dummy variable to represent the host environment. Here, we can't really create a local
variable, but we can do `demo.foo = 0`. So now every time this function runs, `foo` will always
start with 0. And with mocking, you'd probably have a fake database that always returns a successful
response, and another that always returns an error, and then you create unit tests based on those
mocks. I just needed to throw that in there.

So now we should have 100% test coverage, because we're executing all lines of code. Again, test
coverage is simply a quantitative metric of the total lines of code executed during your test run.
So if I added `if false` here, and then a bunch of no-ops — let's create `noop` — and then `if
noop`, and I just pasted a bunch here... even though we didn't change the program logic, and this is
a dead branch that will never run (it'll get compiled out of the binary), if we run the test runner
now, the test coverage got significantly lower.

And this goes the same for the opposite direction. Let's remove the dead branch, so now this will
always get executed. And then let's remove one of the unit tests. Now we have high test coverage,
but we have one feature that's completely untested.

### Why coverage reports are meaningless

So coverage reports are meaningless. They only serve to indicate whether you even have unit tests at
all. In other words, coverage reports can be faked — they can be gamified. They're not objective in
any sense of the word, based on how organizations like to interpret them. When you have developers
who are stressed, who have a lot of workload, and you shame them, shout at them, guilt-trip them
enough, they will inflate these coverage reports to get on with their day and get you off their
back.

Going back to our coverage report, we have 100% test coverage. What happens if I decide to add
another input? This time we'll do a divisor of zero, and we expect the output to return 0 for this
implementation. If we run this, the program crashes — it panics, even though we had 100% coverage
beforehand.

So let's handle this edge case: quotient = `if divisor == 0, return 0`. There you go, and return the
quotient. Now when we run this again, we have 100% test coverage. These are the same metric, but
they have vastly different qualities when it comes to software reliability and correctness.

So let me ask you this: when you have 100% test coverage, how do you move forward? How do you know
which edge cases you haven't tested, other than literally staring at the code, looking at your unit
tests, and figuring out which one you haven't tested? You see, coverage reports are [bleep]. The
program state surrounding each line of code is not accounted for. They simply indicate that you even
have unit tests at all. Worst of all, they're prone to gamification. Remember, this is 100% test
coverage — but we didn't handle the edge case of dividing by zero.

## What TDD is good and bad for

Test-driven development is about writing unit tests before implementation, using them as a
specification to guide product iteration. They're good for drafting APIs, creating smoke tests and
regression tests — but they're bad for polishing APIs, the internals of your system, and enforcing
system properties.

## Properties, invariants, and assertions

A **property** is something expected from an entity. An **invariant** is a type of property that is
never changing. An **assertion** is the enforcement of a condition to hold during program execution.
There are two types: **hard assertions**, which crash the program, and **soft assertions**, which
don't (they typically log instead). **Property testing** is about providing comprehensive test
inputs to APIs while asserting properties within your implementation. Violations encountered during
development and production are accompanied by loud, immediate, and clear feedback.

### Demo: noticing properties and enforcing them

Let's go back to our unit tests and add more test cases. Ghostty — `demo_test`. Let's copy this, and
we'll make the second addend negative; we'll expect it to be 4. And let's do one more — this time
it'll be -4, and the sum should be 2.

As you provide more test cases to your unit test, you'll notice patterns — relationships between
your input and output. These patterns are properties of your system or your data. For example,
whenever the addends have opposing signs (6 and -2, 6 and -4), the sum will always be lower than the
positive addend.

To ensure these patterns we notice are real properties and not just coincidence, we can use
assertions to enforce them. And if they're ever violated, then we get clear feedback and can fix the
issue immediately — because our assumptions during implementation are now invalidated, which
probably means there's a bug floating in our codebase somewhere.

Let's go back to our implementation. Import my library's `invariant` package. And we'll say:
whenever the addends have opposing signs (x > 0 and y < 0), then the sum is less than the positive
addend. The assertion message: "Sum is lower than the positive addend when addends have opposing
signs." Now when we run this, it passes. And if we remove these test cases — I'll comment them out —
oh, I forgot something: we also need to import my library here, and there's a snippet we have to
add, `TestMain` (plus GC setup), and we need to import `os`. Now our test suite is failing, because
we have one assertion that was not evaluated to true throughout the whole test run. And when we add
back one of the test cases, now it's passing again.

Now, if we ever had a bug in our implementation — let's say add 1 here — the assertion will fail.
Let's add 100. The assertion will fail with the assertion message. Now, what if the bug got past the
assertion, and we instead return after this invariant — so it always returns 100? Then our tests
will fail.

What you're noticing here is a two-way contract between the unit tests and the implementation.
Whenever their assumptions are out of sync — maybe the internals changed and we forgot to update the
unit tests — one of them will always fail.

### The two-way contract (TigerBeetle)

And this is the exact testing methodology that TigerBeetle uses. If you don't know TigerBeetle, it's
one of the most reliable software on Earth. Let's look at their blog — "It Takes Two to Contract,"
written by matklad. Essentially, what this blog says is that they have assertions both at the
function definition and the function call site.

So let's bring up `func main` again — I'm not going to execute this, I'm just going to show you.
Let's say I call `pureAdd` here with 6 and -2. What they do is they also assert this in the scope of
the call site. So whenever the internals of this function change and they forgot to update this
caller, there will be an assertion failure somewhere, and they can fix the bug. This is the same
idea as what I'm saying: our unit tests and our implementation are coupled together, and it forces
us to update both at the same time whenever the API changes.

So what I showed you is property testing — or at least my definition of it — which is: you embed
assertions of your system properties in the implementation, and you have unit tests that verify the
output of your API, and this creates a two-way contract between the two sides of your development
environment. Now, whenever they go out of sync because of internal API changes, you have clear
feedback on what issues, bugs, or code need updating.

## The formal definition of property testing

Now, you might say, "Well, James, that just sounds like a more complicated — maybe more explicit —
way of enforcing 100% test coverage. I don't really see the benefit here." And somehow you're right.
Let's take a look at the formal definition of property testing and see if it encompasses my
implementation.

These two definitions are from Antithesis, an autonomous testing company, and an academic paper
titled "Property-Based Testing in Practice" by Goldstein et al., 2024. Pause the video, read these
definitions. What they both have in common is that property testing involves automatically
generating inputs to your test cases. This differs from my implementation, where I manually create
new test inputs for my unit tests.

This automatic generation of input is what we call **fuzzing**, and it's what unlocks the power of
property testing — because now you're feeding this API with a multitude of inputs, and you're
asserting the properties of your system through a wide variety of scenarios, giving you much more
confidence in the correctness of your software.

## From limited property testing to implementation-driven testing

So my library here is simply an emulation of a fraction of the power of property testing, and I like
to call it **limited property testing**, which involves manual creation of inputs. It's easy to
create and maintain, but it only emulates a fraction of a fuzzer's comprehensiveness.

The workflow that naturally arises from limited property testing is called **implementation-driven
testing**. As new assertions are added, the test runner highlights untested conditions, prompting
the creation of additional test cases. This forms a two-way contract between tests and
implementation.

## Types of fuzzers

You're probably saying, "James, I want the real deal — give me full property testing. I want to get
oiled up in property testing." How hard could fuzzing really be? So let's talk about the different
types of fuzzers and rank them by efficiency: dumb fuzzers, specialized fuzzers, and intelligent
fuzzers.

Never mind the terminal icon here — I'm not going to code this. The best way to visualize this is
with a JSON parser. Let me write it out so you can imagine it: `func jsonParse(data string)`, and it
returns an AST.

With a **dumb fuzzer**, it will literally just give it random data, random bytes. This is going to
be very inefficient, because most of the time — 99.9% of the time — you're going to receive an
invalid JSON object.

That's where **specialized fuzzers** come in. Now you're providing higher-level input for this
function — maybe you're providing a token instead, like an open brace, a string key, a JSON array,
nested JSON arrays. But most of the time you're probably still going to provide invalid input,
because the fuzzer doesn't have guidance of the JSON grammar. So you go up another level, and now
you're feeding JSON nodes and modifying a valid JSON input — you're turning this JSON array into a
nested JSON array, turning this string key into an integer key, whatever. Now you're always
providing valid JSON. And we can keep going and going with how to optimize this fuzzer, until you're
basically encoding the edge cases into your fuzzer.

Now imagine doing this for every function, every package, every module in your codebase. It's going
to be high-maintenance, a huge undertaking, and it just doesn't feel like the right way to do
things. It's such a high bar to overcome. There's probably some way we can design this better and
automate it — and that's where **intelligent fuzzers** come in. You'd have a simple dumb fuzzer, but
you empower it with machine learning, and now it analyzes the code paths that are getting evaluated
and associates that with certain inputs, and now your fuzzer will focus on inputs that lead to rarer
branches. What I'm describing here is a completely different software tool, and that's not something
you're probably going to develop in-house. In other words, fuzzers are nice for property testing,
but to actually use them effectively is hard, to say the least.

### Demo: Go's coverage-guided fuzzer

I'll do a simple dumb fuzzer. Let's fuzz our `pureAdd` function here. It's simple, really.
`FuzzPureAdd` — it's that simple. This is Go's native fuzzer; it's coverage-guided, so it looks at
which code branches have evaluated during the fuzz run. But I wouldn't really call it intelligent in
any capacity.

So let's inject a crash into this function: if `sum == 10`, then we'll just crash — panic, "Hello."
Now if you fuzz, it's going to crash immediately. But let's do 100 now — crashes immediately. What
about 1,000? How long does it take to find a sum of 1,000? You see, it has already remembered the
previous edge cases it found, and it's going to store them inside test data. There you go — it got
57, -47, which triggered the crash.

So it has tried 50 million inputs and it still hasn't found a single input that has a sum of 1,000.
So Go's fuzzer is not really that intelligent. Now you have to add a seed corpus for the fuzzer —
something close to your edge cases, like 960-something. And then when we run this, it's going to
panic immediately, because we stayed close to the edge case. But you see how inefficient dumb
fuzzing really is, and how you really need specialized fuzzers if you want to exercise your
assertions effectively.

## Fuzzing: pros and cons

It generates input for a specific API, and this exhaustively exercises system properties. Usually a
mainstream library or tool is available. But the cons: it doesn't verify output correctness, unlike
unit tests — it simply tries to crash the program by finding inputs that would violate your
assertions or hit specific bugs that cause panics. It's limited for impure functions, and it's best
for deterministic output. And new test cases require new fuzzers.

You see, because they're specific to some API, every new function or test case you create, you're
going to have to create a new specialized fuzzer for. To illustrate, let me sketch out a program.
Let's say you have `databaseCallA` and `databaseCallB`, and you want to interweave them with each
other, and you have a fuzzer that creates inputs for all of these. This is just one unit test. What
if you wanted to create a unit test that groups all of `databaseCallA` together, and then the `B`s
    grouped together? This is another unit test with another fuzzer created for it. What if you
    wanted to create another unit test, but it's always going to be `A`? You can see how laborious
    this endeavor is.

You don't have to fuzz every single function in your codebase, but the philosophy of property
testing is that we're exercising the assertions of the whole system — and ideally we're fuzzing
every single function.

## Autonomous testing

So what is the answer? There's got to be some way to achieve this ideal — and there is. It's called
**autonomous testing**. With autonomous testing, you're automating the generation of test cases on
top of input generation. It does both intelligently, using AI — but not ChatGPT or Claude Code;
machine learning. Like I mentioned earlier, it uses machine learning to inspect the code branches
that are getting evaluated and associate that with the input, and then it finds which states lead to
more bugs, more assertion failures.

### How Antithesis does it

You're saying, "James, it sounds nice, it feels like magic, but I don't believe this is real." I
mentioned Antithesis earlier and that they're an autonomous testing company. Let's pull up their
website. They achieve this using a deterministic hypervisor, parallelism, and machine learning.

Their deterministic hypervisor is a virtual machine. They run your software inside it. Now the host
environment of your software is fully deterministic, which means technically your impure functions
are now deterministic too — and with that, we can now consistently verify the behavior of your whole
program. The entire host, the entire environment, is deterministic down to the CPU instructions:
which CPU instructions are executed, how long each is executed, the scheduling of the threads, and
whatever else you can think of that's variable in an operating system.

Then they use machine learning to discover which program or operating-system states lead to more
bugs, more logs, more assertion failures — very rare code branches that are evaluated. And then they
use parallelism to prioritize these special cases and create a myriad of inputs branching off of
that special case. So if you had a bug that occurs once in 10 years, and another that occurs once in
15 years, they can make it so that, the first time they encounter each bug, they can now
consistently make them happen consecutively. So instead of trying to create this unique scenario
once every century, now they can do it at the snap of a finger. That is the power of autonomous
testing.

### The catch: pricing

So I hope you're wowed. But let's talk shop: how expensive is Antithesis? Unfortunately, they've
since removed their pricing plan from the website since I last saw it, and I'll try to search for it
again... yeah, there's just nothing here, they've removed it. So I'm not going to say the exact
amount, but it's in the tens of thousands of dollars per year, and they only support annual plans. I
don't know if it's changed now and they support cheaper plans — you'd have to book a demo directly,
I guess. But yeah, it's going to be too expensive for most organizations.

## TigerBeetle's in-house simulator

Like I said, TigerBeetle is one of the most reliable software in the world — primarily because of
their amazing engineers, but also because of their in-house simulator, just like Antithesis but
specialized for their database. And I just want to show you how many assertions they have in their
codebase. These assertions are actually enabled in production — they don't disable them, which is
crazy, but they manage to maintain their performance metrics, and it shows their confidence in their
product and testing methodology.

Let's clone their repository, `tigerbeetle`. `tokei` first — is there a source directory? Yeah,
`tokei src`. There's 145,000 lines of code, 100,000 in Zig — literal code in Zig. And let's count
`assert`. There's 11,000 assertions there. (I believe they also have "sometimes" assertions
directly, but I don't see where... anyway, it's not showing up.) So they have 11,000 assertions in
their codebase. When you combine rigorous software testing with amazing software architecture, you
get one of the most jaw-dropping software applications in the world.

## My library: golang-snacks/invariant

So autonomous testing is the pinnacle of software testing, but it's hard. Antithesis solves this for
you, but they're expensive. That's why I created my library, **golang-snacks/invariant**, which is
an implementation-driven testing framework. It's 400 lines of code, MIT-licensed, very easy to
vendor — it's meant to be vendored. (Don't use the package manager to download this. Actually, I'll
put it in an `internal` directory, so you can't even use a package manager. I'll remember to do
that.) It includes examples like a backend server, and it serves as a stepping stone toward full
property testing when combined with fuzzing.

So if you want to take on proper fuzzing for your software, then you're basically doing property
testing and then autonomous testing — by introducing assertions into your codebase and dev process,
so that when you're ready to integrate with Antithesis (and Antithesis is ready to give more
accessible pricing), your codebase is ready now: all you have to do is integrate their SDKs and
integrate them into your CI/CD, and now your codebase is getting fuzzed 24/7.

### `always` and `sometimes` assertions

Let's take a look at my library — `invariant/invariant.go`. There are two types of assertions.
You've already seen the first: **always** assertions — the property must hold for all inputs or
executions. To disprove an always assertion, you only need one counterexample where the condition is
false.

The second type is a **sometimes** assertion — the property is expected to be occasionally true
across runs. To prove a sometimes assertion, you need one example where the condition is true; if it
never evaluates to true, the property is disproven. Sometimes assertions are only meaningful in
testing environments, since observing their absence requires multiple executions of the program.

I'll show you `func always` and how it's implemented: it's simply a condition check. If it's true,
we register the assertion in this global assertion tracker (which is only enabled in test
environments); and whenever it's false, we call the assertion-failure callback. With sometimes
assertions, we simply return if the condition is false, and we register it in the tracker if the
condition is true. That's how we detect that a sometimes assertion is at least true once throughout
the whole test run.

```go
//go:noinline
func Always(cond bool, msg string) {
	if cond {
		registerAssertion()
	} else {
		AssertionFailureCallback(fmt.Sprintf("%s: %s", AssertionFailureMsgPrefix, msg))
	}
}

//go:noinline
func Sometimes(cond bool, msg string) (ok bool) {
	if !cond {
		registerFalseAssertion()
		return cond
	}
	registerAssertion()
	return cond
}
```

Now, when you're using this API, it reads as `invariant.Sometimes`, which is actually a misnomer.
(Let me search what "misnomer" means: a misnomer is a wrong or inaccurate name or designation.) Why?
Because an invariant is a constant property, something that never changes — and when we say it only
happens sometimes, then by definition it's not constant. So it's more accurately described as a
conditional property check instead of an invariant. But I think "invariant" sounds cool, so that's
the name I chose.

Before I save this and ruin my build: you can disable the assertions completely if you need a
performance boost, through the `disable_assertions` build tag, and most of the functions are
replaced with a no-op instead.

### The math example

Let's look at the examples — `math`. This is similar to the demo example: the four basic arithmetic
operations, and I'm just asserting the mathematical properties of them. Commutativity for addition,
identity, inverse, subtraction, the zero property (oh wait, no, that's for multiplication),
associativity, yada yada. There's a lot here. And I didn't do it for the divide function — and even
with that, this file already has 29 assertions, which is a lot.

```go
func Add(x, y int) int {
	sum := x + y

	if x > 0 && y > 0 {
		invariant.Always(sum > max(x, y), "Sum is greater than the biggest addend when both addends are positive")
	}

	// Commutative Property
	invariant.Always(sum == x+y, "Addition is commutative")
	invariant.Always(sum == y+x, "Addition is commutative")

	// Identity Property
	if x == 0 {
		invariant.Always(sum == y, "Adding zero to a number should leave it unchanged")
	}
	if y == 0 {
		invariant.Always(sum == x, "Adding zero to a number should leave it unchanged")
	}

	// Inverse property
	if x == -y {
		invariant.Always(sum == 0, "Adding a number and its additive inverse should yield zero")
	}
	if y == -x {
		invariant.Always(sum == 0, "Adding a number and its additive inverse should yield zero")
	}

	return sum
}
```

For divide, I have this cheeky little comment: "Try to fill this one out yourself. You're encouraged
to use AI, though I suspect it won't be much help." Food for thought: if you can't prove basic
mathematical operations, what about your 20 JavaScript microservices? So I tried to make AI
implement this whole file on its own — I fed it the whole library file, `invariant.go` — and it just
couldn't do it. So imagine how well it would do proofing your JavaScript microservices with uncaught
exceptions.

```go
func Divide(dividend, divisor int) (quotient, remainder int) {
	quotient = dividend / divisor
	remainder = dividend % divisor

	// Try to fill this one out yourself. You are encouraged to use AI, though I suspect it won't be of much help ;)
	// Food for thought: If it can't proof basic mathematical operations, what about your 20 javascript microservices?

	return quotient, remainder
}
```

It also has unit tests here. I'll showcase it: when we remove a lot of the inputs here and test it
again — `invariant/examples/math`... there you go, it fails. Why? Because nine assertions were never
true. So let's re-enable those test cases — and now it's passing.

```go
func TestMain(m *testing.M) {
	invariant.RunTestMain(m)
}

func TestAdd(t *testing.T) {
	t.Parallel()

	cases := []struct {
		x, y, expected int
	}{
		{2, 3, 5},
		{2, -3, -1},
		{2, 0, 2},
		{0, 2, 2},
		{7, -7, 0},
		{-7, 7, 0},
	}

	for _, c := range cases {
		got := math.Add(c.x, c.y)
		if got != c.expected {
			t.Errorf("Add(%d, %d) = %d; want %d", c.x, c.y, got, c.expected)
		}
	}
}
```

### The backend server example

Next is a backend server example. This is just a simple TCP server that fails an assertion whenever
I send a specific message. So `always(...)` — this will always fail the assertion. What I'm
showcasing here is overriding the default behavior for assertion failures (which is to crash the
program): instead, we are logging the error and incrementing a global count of the assertion
failures that happened within a certain time period. In this implementation, I set it to every 30
seconds. We have a dedicated goroutine that checks every 30 seconds for assertion failures, and if
there are, we send an email. The email is sent through SMTP — very basic, for Gmail.

```go
func main() {
	invariant.AssertionFailureCallback = func(msg string) {
		atomic.AddInt32(&assertion_failure_count, 1)
		panic(msg)
	}
	// Ensure any leftover assertions are announced.
	defer func() {
		if atomic.LoadInt32(&assertion_failure_count) > 0 {
			send_email()
			atomic.SwapInt32(&assertion_failure_count, 0)
		}
	}()

	// ... accept connections, read messages ...

	ticker := time.Tick(notify_frequency)
	for {
		// Log assertion failures and continue; let all other panics crash the program.
		select {
		case message := <-messages:
			invariant.Always(message != "you gave me up", "Never gonna give you up.")
			database[message] += 1
		case <-ticker:
			if atomic.LoadInt32(&assertion_failure_count) > 0 {
				send_email()
				atomic.SwapInt32(&assertion_failure_count, 0)
			}
		}
	}
}
```

Let's start the TCP server now: `go run` the backend example. And `nc` on port 51094. "You gave me
up. You gave me up. Never gonna give you up." And we'll just wait — it should log when it sends an
email, right here: "assertion failure announced via email." Let's wait for it.

Now I'm going to add another assertion failure — it should have reset the counter, so we should
expect two emails. Let me shut down this first, stop accepting connections. So we should expect two
emails: one that says four assertion failures, and another that says two. What's important to note
is that I explicitly flushed the assertion-failure count when I shut down the server — I have a
defer function that checks one final time if there are any assertions left. This is critical,
because if your server crashes and you didn't announce the assertion failures, that's really bad.

Let's take a look at my email: "Detected four assertion failures in the last 30 seconds." "Detected
two assertion failures in the last 30 seconds." And it was one minute ago.

## Conclusion

So, test-driven development is about drafting APIs, creating quick sanity checks with smoke tests
and regression tests, and output/behavior verification. Implementation-driven testing is great for
polishing API internals, exercising system properties, and actionable feedback on elusive bugs. One
philosophy does not usurp the other — you need both to create a robust testing framework.

With both in place, you get a two-way contract between the high-level test environment and the
low-level implementation environment. Whenever they get out of sync, you get clear, actionable
feedback on the relevant code snippets you need to update. What I love about this approach is that
it encodes the documentation internally: you don't have a lot of context switches with reading the
source implementation, then going to the docs website, then verifying that it adheres to the
specification, yada yada. Now you have a low-level understanding of the system assumptions within
the local scope itself, and that creates a much nicer developer experience — something a lot of
people tend to underestimate, and how it affects the quality of our software.

So I hope you learned something from this video, and you start creating software that you can love.
