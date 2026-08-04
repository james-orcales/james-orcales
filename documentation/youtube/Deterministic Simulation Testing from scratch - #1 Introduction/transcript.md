This is the introduction to a multi-part series. It's called deterministic simulation testing from
scratch. The objective for this series is to implement single process deterministic simulation
testing for an entire SaaS platform.

Key questions you need to keep in mind for this video. Why do we need to define DST? What does
determinism mean in the context of software engineering? How do you increase the determinism of
your system? What are the different types of DST and their advantages? How would you architect a
SaaS platform such that it can be entirely simulated in one process?

This video is licensed under Creative Commons Attribution. My videos are not monetized or
sponsored. I'm not affiliated with any entities mentioned in this video.

I'd like to start us off by questioning you and your philosophical worldview. Do you believe in
libertarian free will? The claim that when presented an opportunity to make a decision, a choice,
that you can make any decision at that point regardless of what happened beforehand. For example,
if you were asked between to choose between chocolate cake and ice cream for dessert, it doesn't
matter what you had for lunch, what you had for dessert the day before, the week before. It doesn't
matter if you had some childhood memory of you seeing your mom enjoy chocolate cake and now you
have this weird subconscious uh preference for it up until you stretch that um reasoning to the
birth of the universe. Um, your decision is truly random and is made in the spur of the moment.

Next, let's question the foundations of our reality. What is your preferred interpretation of
quantum mechanics? If you're not familiar with this topic, um, as I am, I only know like the gist
of it with a full days of discussion with Claude. Basically, there's different algorithms that
explain or predict the values or outcomes produced by quantum mechanics. And when we say what is
your preferred interpretation of quantum mechanics, what we're really asking is what do you think
is the algorithm that or the mechanism behind quantum mechanics? And to give you an analogy, for
gravity that would be the theory of relativity. These algorithms, what distinguishes them from each
other uh is what assumption, which assumption of our current understanding of physics is getting
violated, and they all have their own unique violations. So it's a pick your poison type of thing.

Despite these differences, however, they all agree on the values or outcomes produced by quantum
mechanics. And that's the interesting part of it all. And what I really want to know from you with
this question is do you think there is a deterministic mechanism behind quantum mechanics, any of
the any of the algorithms proposed, or is it a truly irreducibly random phenomenon and that them
agreeing on the values produced just happenstance.

And question of the decade, how many wipes does it take to know you only needed seven? I'll give
you a hint. It's not eight. It's not always eight.

Ultimately, what I'm really trying to get at here is do we live in a deterministic universe? To be
more precise, determinism here is defined as every event is necessitated by antecedent events and
conditions together with the laws of nature. In other words, determinism is absolute and
unilinear. The entire chain must be of a cause and effect chain. That is what we mean by absolute.
And what we mean by unilinear is that for every cause there is only one effect possible. For every
effect there was only one cause possible.

In the first chain, this represents a an entirely deterministic timeline. You'll see that for every
cause there's one effect. And another thing you should notice is the ellipses at both ends, because
for this to be absolute, well, every, it's going to be an infinite chain, right? Every effect must
have a cause.

The second chain, the second timeline rather, uh represents libertarian free will. The question
mark in the middle, that represents the agent, the human with free will. And you can see the fat
arrow here represents the multiple choices or effects that they can make. So regardless of what
happened before, even if the timeline was entirely deterministic beforehand, multiple effects,
multiple um events could have emerged from this uh agent, and that breaks determinism. Uh firstly
and its unilinear property, which then also breaks its absoluteness since everything coming after
this event is now indeterministic.

The third chain, sorry, the third timeline, I keep calling it a chain, uh represents quantum
mechanics if it was irreducibly random and that there was no deterministic mechanism behind it. So
since one of the lowest layers of our reality is random, then everything on top of it is random,
and things just being a cause and effect deterministic timeline is just an illusion and everything
is just a probabil probability.

Uh, to visualize it better. If you're familiar with that um common scientific fun fact where every
time you punch a wall, there's this very very very very tiny chance that your hand will just phase
through the wall because all of the the atoms will m will miss each other. Uh so this third chain
is, if quantum mechanics is truly random, then basically every time you punch the wall there's, the
universe rolls a dice on whether the atoms will hit each other. Um and most of the time the dice
will say yeah they will hit each other.

These properties of determinism create significant implications on what we are trying to do with
our testing methodology, and we need to address these implications if we are to declare that we are
deter doing deterministic simulation testing. You see, our program then is deterministic if and
only if the universe is deterministic. Even if here in these [snorts] timelines I'm visualizing the
program as deterministic, that when we ask it to do one thing it will only do that one thing. If
any part of our reality is indeterministic, then who the hell are we to to declare that we're
deterministic, that we're doing deterministic simulation testing?

So the question, do we live in a deterministic universe? If the answer is yes, well that makes
everything easy. But that begs the question, why is determinism determinism a defining
characteristic of DST? And if not, does DST manufacture determinism on top of an indeterministic
reality? Or does it simply assume that phenomena relevant to software engineering is sufficiently
deterministic? We don't have an answer, but maybe we don't need it. What if our program was
deterministic regardless of the of the universe's determinism? So it's just a matter of changing
perspective and we are completely ignoring the deterministic properties of the environment and
solely focusing on our program.

Well, up until this point, what I've actually been using as the concept of determinism is what we
call ontological determinism. The claim that reality itself is causally determined, originating
from the Greek word ontology which is the study of what there is. Ontological determinism is the
simple statement of fact that reality itself is by virtue causally determined, or inherently
causally determined. To give you another example, it would be like if I said there is a god, there
is a higher order deity governing us. There's no way for me to prove that. I'm just asserting the
claim on to you as fact, as a mere fact of our reality.

And here the first timeline in the earlier slide uh makes its appearance again. So the entire
universe here is causally determined. It's an entire cause and effect chain. Now using this concept
of determinism, or this definition of determinism uh for our purposes, is [snorts] it should
really, it should already be giving you some funky smell in your gut, because uh for us to claim
deterministic simulation testing we need to prove that the entire universe is deterministic on a
philosophical standpoint. That's quite a high bar to overcome and it signifies that um, or it
signals that uh we're using the wrong term here.

Let's look at other definitions of determinism. And in philosophy, there's another one, is called
epistemic determinism. The claim that the future, however, is knowable given complete knowledge of
the present. It comes from the Latin words episteme and logos, which means, which roughly
translates to knowledge and reason respectively. So when you put those together you are acquiring
knowledge through reason. You are predicting the future with the current information at hand.

Now in a in a completely deterministic universe, yeah that's possible uh and doable if you are a
god. If you are a deity that knows everything. But we are mere mortals, software engineers. Not
only do we have a tiny human brain, but our domain is a tiny computer with limited compute
resources. Even if we had, even if we hypothetically had all of the information present, we don't
have enough compute power to look into the future.

But I digress, because this concept of determinism isn't what deterministic simulation testing is
about. So let me clarify now what deterministic simulation testing is about on a higher level. DST
is solely about uncovering reproducible bugs. A deterministic program, at least a sufficiently
complex one, doesn't mean you can statically list out all of its bugs. Its state space is
combinatorially vast. You need to simulate up to the point where the bug occurs. That's why half of
the engineering effort in DST is uncovering bugs in the first place. Reproducing it once found is
the easy part.

Um, and to put it in simple terms, that's why when you're doing fuzz testing, just simply throwing
random bytes, especially if it's like a grammar um, if it's like a specific grammar, for example, a
JSON parser. If you're just throwing random bytes at that JSON parser, your fuzz testing is
absolute. You're not going to test anything there because most of your input is just going to get
rejected. So you want targeted input that is already valid JSON and then you're fuzzing the grammar
itself, not just some random arbitrary bytes.

We need to look at definitions of determinism that revolve around the idea of localizing the
determinism itself, or something that's about shifting perspective. For that I found Stephen A.
Edwards's quote, which is determinism can be thought of as an abstraction boundary that delineates
where control is passed from a system designer to the implementation. So what you'll notice here is
that his definition is more of a, it's geared towards software engineering. It's not really some
kind of philosophical worldview, and maybe that's the fault of what of our initial efforts.

And he formalized his definition into something that he calls a model of computation. And we're
going to get into some mathematics here, but bear with me. I'm going to make it all digestible
later. This is for those that want the nitty-gritty of it all. And if it helps you, I don't have a
mathematics background as well. [snorts]

Right? So S is the set of all legal system specifications supplied by a designer. C be the set of
all legal choices that can be made in implementing any system. I and O be the sets of inputs and
outputs accounted for by the model of computation. E and B be the sets of environmental inputs and
behaviors not accounted for by the model of computation. A model of computation M is deterministic
if for all specifications, implementation choices, inputs, and environments, there is some function
D given some S, I that produces O.

If you look at the uh equation below, I've highlighted in yellow the deterministic variables and in
red the indeterministic ones. When we are evaluating a program's determinism, we solely look at the
specification, the given input and the produced output, and it should always be the same regardless
of the implementation choices, whether it's in Rust, Go, Python, JavaScript; the environment,
whether it's on a Linux host, Darwin, Windows; um, and the behavior, whether it took 5 seconds, 1
millisecond, or the peak RSS was 50 gigabytes or 5 MB.

These examples that I just mentioned, they aren't always part of the behavior or the environment or
the implementation choices. Maybe it's part, sometimes it's part of the specification that it must
always take 5 seconds for this operation, that it only consumes 500 kilobytes for this system, or
that this particular program is always implemented in X language. These decisions are made by the
programmer. In other words, the the outputs that the model of computation accounts for only depend
on the system specification and the inputs accounted for by the model of computation.
Implementation choices and the environment may only affect the behavior of the system outside of
these outputs.

I have another definition here. It's from Dominik Tornow. He's the founder and CEO of Resonate HQ.
And I I forgot to actually uh mention the credentials of Stephen A. Edwards. He has a PhD in, I
believe, electrical engineering. Um he's an associate professor at Columbia University in their CS
department.

Okay, back to Dominik Tornow. A system's execution unfolds in a series of discrete events called a
trace, governed by the rules of its runtime environment. Each event is categorized as either
observable, external, or non-observable, internal, based on its relevance to us. A system is
deterministic if there exists only one possible trace of external events.

You'll notice that Dominik's definition is geared towards distributed systems, a lot of async
operations where the inter interweaving of events um in unpredictable ways beyond what the human
mind can imagine is the primary source of critical bugs. That's why he has a focus on the event
traces produced by some initial state.

In contrast with Edwards, um Edwards, Edwards's uh definition is more of a, is more mechanistic I'd
say, in that it focuses on which aspects of the program is part of the deterministic boundary,
what's inside it and what's outside of. If I had to put it simply, Edwards's approach is more of a
mental framework that guides how you design your system. Uh it's more of a uh down to the details,
if you will. Dominik Tornow's is higher level. It's simpler. He only categorizes events as either
observable or non-observable, and has only one condition on whether the system is deterministic or
not, which is that it produces only one possible trace of events.

Tornow also formalizes his model. I'm not going to read this. I don't even know how to read it.
Basically, for every initial state, which he calls the seed, there's exactly one possible trace.
They're saying the same thing just in different ways. But the key idea here is that determinism is
relative to which events you choose to observe.

I'm coining this uh definition of determinism. It's called bounded determinism. Determinism defined
relative to a chosen boundary. I've highlighted in yellow the program execution and blue the entry
point, or the, so this would represent the initial state. Looking at the right side, the grayed out
nodes or effects represent the behavior of the program, the side effects if you will, that we
choose to not observe. Yellow would be the output that is part of the specification. And then this
would be the initial state. The blue one is what's part of the spec. And then the grayed out ones
would be the environment.

Now this is actually the incorrect way to visualize this, because since this is all just one
timeline, actually it should be more like this. You are dis discarding some events and you're
choosing to observe some other events. But it's a bit harder to understand what's going on here,
because this isn't, this isn't really what's happening when we're trying to design the system.
We're not just saying like, oh, we're not, we're going to discard this event and then we're going
to discard that event. No, we are discarding categories of events or kinds of events, right?

So for me it's nicer to visualize it as this, where for example this chain would represent the uh
the memory usage of the host. So for our specification we don't care how much memory, how much
memory our program uses. So it's just part of the behavior and the environment, the non-observable
events. In other words, we think of events as layers, and we bring in more layers of determinism to
our program or vice versa.

The nice part about this definition of determinism is that you don't have to sacrifice your
philosophical worldview. The first timeline represents libertarian free will. Again, yellow would
be our program and X is the user. So when we declare that our program is deterministic, it is
independent of the user's free will, the user's indeterministic free will, because to us all that
matters is the specification and the input and the output of the program, or its execution trace.
Now if part of our specification is the user's actual free will, then yeah, the program would be
indeterministic then at that point, right?

And I'm going to go on a tangent here for a little bit. This has some cool philosophical
implications as well, because now you can think of libertarian free will as just the universe
having non-observable events. And when it comes to philosophy and science, observability just means
we can empirically measure it. Right? So this implies that your free will, your decisions directly
affect the decisions of other people in some hidden way, that we're that we're all connected
somehow by the universe. Uh I'm not trying to [laughter] I'm not trying to invent some new
philosophical view here. It's just something that I uh noticed. I don't know if there's like a
real, like this is a real thing already that someone has proposed, but you know it's it's cool.

Examples. Let's make it digestible now. So a C99 compiler. The spec would be the C99 standard. That
would be the external events in Tornow's um definition. You'd have your input, the source file, and
the output, the object file. The internal events, that maps to the implementation choices,
environment, behavior.

For implementation choices, that would be your compiler extensions, division by zero. Now there's a
semantic quirk here with division by zero, because in the standard it would be undefined behavior.
So why would you call it implementation choice? So I think here implementation choices don't refer
to the decision made by the programmer, rather it refers to the consequences of how the program
itself was implemented. So whatever the behavior is of the compiler with division by zero, whether
it was an explicit decision or not, that is part of implementation choices.

Environment, your Instagram username. So if the compiler um miscompiles your program based on your
Instagram username, then it's being maliciously indeterministic to say the least. Behavior,
compilation speed, peak RSS, logs. So that's not part of the specification. Um, and it doesn't
affect whether or not we can declare a compiler implementation as adhering to the spec.

Compilers aren't really that interesting to subject to DST because they're just a pure function.
They're only doing byte transformations from one representation to another.

So let's look at another one, uh an authentication microservice. Now this is where DST shines, and
that is why Dominik Tornow's definition is geared towards the execution trace, because this is a
highly um async service. There's a lot of interweaving of events.

So normally, for the spec you just have OpenAPI. The gate, uh the microservice receives an HTTP
request and it has an HTTP response. Internal events would be, would it cache the response, does it
always do eager fetching. The environment, is it self-hosted, is it on EC2, disk IO. Behavior, the
response time, cache fragmentation, the time itself, like the the measurement of the clock, and
network IO. Uh if you have SLAs, maybe you can put the response time on the spec, like it should
always respond under 50 milliseconds. But you get the idea.

Now you might think for the most part that this determinism boundary for this microservice is all
right, you know it's fine. Uh and that's actually where most systems fail in their implementation.
Think about it, what are the core responsibilities of an authentication microservice? Well, one of
it is validating session lifetimes. Is this credential, is this session expired or. And since that
is a core responsibility of this microservice, it should not only be OpenAPI in the spec.

The time itself, the measurement of the clock itself should be part of the spec. It should be an
observable event. It should not be an internal event. It should not be hardcoded in your internals
that you are always querying the hardware clock through syscalls. It should be dependency injected
in your functions so that you can stub it out in your tests and you can test different scenarios
where your authentication microservice receives expired requests.

And this is how it would look like. So you put your time and network IO. Of course, your
authentication microservice needs to be uh fault tolerant to network failures because it's a
security boundary. Uh this is where uh systems fail for the most part. Instead, it should look like
this. Your spec, your external events for this microservice should be OpenAPI, time, plus network
IO at the very least.

And what this allows is you can now stub out different scenarios in your unit tests, different time
measurements, different network IO fail failures, instead of re relying on integration tests that
are um complicated to set up and most importantly slow to execute, because there now you're doing,
I assume you'd be using Docker, some heavy setups. And in those cases, since they're um very
complex to say the least, you're not doing as much test scenarios there anymore. And one more
thing, it's not easily as fuzzable. But you know, I, we'll get to that more later.

So handling indeterminism. We have some Go code here. If you're not familiar with Go, all you need
to know is that this is a hashmap and that iterating over hashmaps is indeterministic in Go. The Go
runtime intentionally randomizes the order. Now, if you're following the idea of this video, ask
yourself, is this program deterministic or not? If you're following the idea of this video, then
you shouldn't be able to answer that question. You should be, I should be met with the question,
well, is iterating the map part of our observable events? If it's part of our observable events,
our external events, then this program is indeterministic.

So how do you handle indeterminism? Well, one, you can eliminate it from your program entirely. So
we always sort the map before iterating over it. But a second one is seeding. Seeding via forking
the runtime.

This is from a blog post of Polar Signals. All random choices in the runtime use a global random
number generator seeded at startup, including the rand package exposed to the Go application. This
seed is read using the OS specific read random function. There is no way to provide this seed at
startup. So we ended up modifying the Go runtime to read the seed via an environment variable. The
change is less than 10 lines of code, could be less, in a very stable part of the codebase.

Okay. So this term seed has popped up again, and I've already uh mentioned it earlier with Tornow's
definition. Uh the way he defined seed is he refers to it as the entire initial state of the
program. For me I diverge it a little bit and I'll create my own definition of what a seed is. So a
seed is a set, is a set of starting values for all external events. The initial state is a set of
starting values for all external and internal events. So when you're seeding something you are
making it part of the external events of your program.

So here, when we say that Polar Signals seeded the iteration of the map by forking the runtime,
we're saying that they made it part of their deterministic boundary, and now they are able to um
control the behavior of the runtime and how it iterates the map through envir, the environment
variable.

One key idea to note here is that whether it's pure or impure, injected or hard-coded, a function
argument or a CLI flag, it's all part of the initial state or seed. Ideally, however, you'd use
dependency injection and make it a function argument at the very least to make unit testing easy,
but doing otherwise is um completely valid and is dependent on your situation as well.

In DST, the seed is commonly generated from a single integer. The starting values of all external
events are generated from this integer alone. Now, there's a couple of benefits to this. One, it's
very simple to look at, look at an execution trace and reproduce it with a single value. And
another one is that it's very easy to fuzz. However, nothing is stopping you from listing out your
seed in JSON.

One last term that I'd like to clarify before defining single process DST is modeling. A model of a
given concept describes how it produces its execution trace. It should emulate production behavior
as much as possible, or be contrived to induce some test scenario. Examples. This model of time is
frozen at January 13, 2005. This model of time treats CPU compute as instant and only tracks IO
operations and how long they take. This model of time advances nanoseconds every N CPU
instructions.

So what is single process deterministic simulation testing? Single process means the entire system
runs in a single process. Deterministic, a system is deterministic relative to a chosen boundary.
Simulation, modeling external events to exercise a desired program execution. Testing, exercises a
system's runtime within some model of the production environment.

Putting it all together, single process DST exercises a system's runtime by seeding external events
in accordance to some model of the production environment, which may span multiple processes or
hosts, entirely within a single process.

All right, that was a lot. But now everything is clear and I've proposed to you my mental model and
how I understand this entire domain.

Of course, there's different types of DST. Single process and single binary DST, single process and
multi-binary DST, hypervisor based DST.

Single process and single binary DST. Examples of these are databases, TigerBeetle and FoundationDB.
In production, these are multiprocess and single binary deployments. So the same binary, the same
database binary, is deployed on different hosts and they are communicating in a distributed manner.
In testing however, these are single process and single binary. The different nodes are
representable in memory.

Let's take a look at the source code of TigerBeetle. Third party, tiger, TigerBeetle, and let me
zoom in. Uh let's look at the replica type. Yes. Okay. If you're not familiar with Zig, it has first
class comptime or first class metaprogramming features. So this function is returning a type. Okay.
So this is a type declaration basically. And if you look at this type um the different fields that
it holds. So which replica is it? Which node um which node it is. Um I believe it also holds the
clock. It represents the hardware clock of the host, specifically the wall clock. So that will be
the um the calendar time, not the monotonic time. And so when they're testing these, they're just
instantiating this type multiple times and then they're simulating the network communication
between the different nodes in memory. So that's how they're doing single process, single binary
testing for something that is deployed on different hosts in production.

The next one would be a multiprocess and multi-binary deployment. This would be your typical SaaS
platform. Our example here is Resonate HQ. They have their back-end workers communicating to a
central database. In testing, they represent multiple binaries by using an embeddable database as a
backend. So they support SQLite. And if you don't know what embeddable means, it means that you can
import the application itself as a library and you can interface it through the library. So they're
importing SQLite as a library and that's how they're testing it in memory.

And you know this isn't part of DST, but this is more of like a hybrid. And I think it's just cool
to show. So part of their test suite is that uh when they're doing their test scenarios, they're
executing the input against all three backends uh at the same time or sequentially, I don't know.
Uh but the idea is that they're testing all the different backends. So they're doing in-memory
testing plus integration testing all at the same time. Kind of cool. I just wanted to show it.

This is the idea behind testing an entire SaaS platform within a single process, because if you
think about it, a SaaS platform is just a lot of different binaries communicating with each other.
Also, if we can somehow be able to represent these different applications as embeddable libraries,
then we can do them all in a single process. Right? So fuzz testing it instead of just doing plain
integration tests, we'd represent these different applications as interfaces.

So one um approach is that you just put everything behind a runtime dispatch or a static dispatch.
So instead of directly communicating or directly hardcoding the Postgres driver in your internals,
you'd use an interface and you just expose the semantic business operations that you're doing on
the on the database. And that goes for all the other uh backends that you are interfacing with.

And this shape right here, where we are homogenizing everything, is already the same pattern used
in hypervisor based DST. If you look here, the box represents the deterministic hypervisor and all
your applications are hidden behind a Docker container.

So out there there's only one company that does hypervisor based DST, and that's Antithesis. So
they've created this um deterministic hypervisor and then they have their own software explorer
that helps guide it so we can uncover the different bugs living in your system. You can see here,
here in their official visualization um, there, the terminator, that was the box in the previous
visualization, and then there's the workload and the containers of your system.

And this is from their own documentation, handling external dependencies. This is a detailed
discussion of how to handle your system's dependencies for testing in Antithesis. Remember, testing
happens in a hermetic testing environment with no internet access, because your software has always
wanted its own little Faraday cage. Each of your software's dependencies can be provided either as
an actual containerized service, just like you'd use in production, a mock of the service, which
could be prebuilt LocalStack or something you write yourself.

So we're just following their footsteps but doing it at the source code level.

Here are the differences between single process and hypervisor DST respectively. You have semantic
awareness of internals versus you can only see program output. Lightweight, analogous unit tests
versus heavy runtime, as integration tests. In single process DST, you have the exposed OS APIs as
the maximal boundary, aka syscalls. Whereas in hypervisor DST you have CPU instructions.

Consequently, you can model time in single process DST at IO event granularity, or the syscall uh
granularity. In hypervisor based DST, you can model time as CPU instruction granularity. That's why
you'll see in uh TigerBeetle and FoundationDB, they treat CPU compute as instant and they only
track the time that passes on IO operations. That's because you don't have access to the individual
CPU instructions that are getting executed by the CPU. That's not the case with hypervisor based
DST. And that's why Antithesis can also simulate the um how fast your program can execute uh
compute heavy workloads on top of the interle of interweaving of different uh IO events.

Single process only tests simulated paths. Uh so your pure implementation of your code, that's the
only thing that you can uh simulate deterministically. Whereas in hypervisor based DST you can test
both the simulated and the production paths.

When you want single process DST, usually you need a complete rewrite of the entire system,
especially if it's a distributed system, since now you'd need to make all of your binaries have a
pure implementation and to respect a central control event loop. Whereas in hypervisor based DST,
uh it's incremental adoption. You can just plop any program inside their hypervisor and as long as
you can integrate with their platform, they can start fuzzing it and looking for bugs immediately.

Since Antithesis is not only a deterministic hypervisor but an entire testing platform as a
service, they kind of blur the lines a little bit since they inspect your system internals through
their integrations. This is a code from their blog. Lastly, you can add logging or assertions that
help us to distinguish important states.

So logging, that's what they, that's uh this part right here, only sees program output. They inspect
your logs to determine interesting states. So whenever they encounter, or whenever they see a warn
log or an error log, that's some important state that they signal to their fuzzer, or their
software explorer rather. Assertions, these are inside your codebase itself. They have an, they
have an assertion SDK and then you litter the code with these, and that's how you give, that's how
you signal their explorer the semantic properties of your system.

I'd like to mention the advantage of single process DST over hypervisor based DST, which is that
you can encode the properties of your system with the external events that you can expose within
the source code itself.

Scenario. Let's say you are a SaaS platform and you have rejected the idea of rewriting your entire
system for single process DST, and you are relying on Antithesis to uh make time uh for example be
deterministic. So you're going to keep querying the hardware clock internally instead of dependency
injecting it. Now even though Antithesis has an assertion SDK, you cannot easily assert these parts
of your program because now you are dependent on the deterministic environment of the hypervisor.
So now you are reliant on their platform to reliably exercise these these properties.

Another question to answer is, are there any other benefits to having CPU instructions as external
events besides enabling a model of time that covers compute speed? And the answer is yes. Uh the
big 2026, not only do you have to worry about AI hallucination, CPUs hallucinate too. So recently
we've discovered that CPUs can create, can have wrong computations. 1 + 1 equals 3. The official,
the official term for this is uh mercurial cores. In 2021, Google published a paper detailing this
exact phenomenon. The general public doesn't need to worry about this. It's very rare to happen.
It's just that at Google's scale, it's statistically significant and something that they need to
address. And consequently, Meta also uh confirmed this phenomenon with their own published paper.

So for so in uh there's been new research on protocols that can tolerate mercurial cores, and you
know uh having CPU instructions as an external event could be a useful tool in validating the
implementation of these protocols. So you can inject instructions that are doing wrong
computations.

So I've told you a lot of these beautiful things about DST, what you can do with it, how you can
architect it, and the problems that you can solve with it. But I'll make this very clear. DST is
not a silver bullet. You can't model real re reality perfectly. The simulator itself can have bugs
and the test coverage can still have gaps.

This is a testing um partnership I guess between Jepsen and TigerBeetle and Antithesis. This code is
from their uh post analysis mortem I guess. I don't know what to call it. Just a blog. Let's call
it blog.

Occasionally, tests with single bit file corruption in the superblock, write-ahead log, or grid
zones caused TigerBeetle version 0.16.20 to crash on startup. TigerBeetle's internal testing with
developer did not discover this bug because it corrupted entire sectors rather than single bits.
Corrupting a sector caused the checksum to fail and triggered the repair process. The zero padding
assertion was never reached. TigerBeetle revised the developer to introduce single byte errors,
which reproduced the bugs.

So, TigerBeetle, despite their rigorous testing processes, still missed this bug because of a gap in
how they simulated reality. So, that's a clear, very clear example of um DST being not a silver
bullet. Uh there's a lot of other examples in that blog post. I have a link in the description
below.

During the course of this research, Jepsen, TigerBeetle, and Antithesis collaborated to run Jepsen's
TigerBeetle test suite within Antithesis's, say that 10 times fast, Antithesis's environment, taking
advantage of Antithesis's deterministic simulation, oh, the apostrophe s shouldn't be there, fault
injection and time travel debugging capabilities.

If there's one key principle that you should take away from this video and this entire series, it's
end-to-end dependency injection. As long as you keep this in mind, even if you don't watch the rest
of this series, you can arrive to the same architectural decisions based on this principle.

Okay, try to answer these key questions again. Why do we need to define DST? What does determinism
mean in the context of software engineering? How do you increase the determinism of your system?
What are the different types of DST and their advantages? How would you architect an entire SaaS
platform that can be simulated in one process?

The end. In the next video, I'll be discussing how to model time.
