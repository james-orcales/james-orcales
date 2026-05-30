# Swarm Testing

**Title**:   Will Wilson on Swarm Testing -- Papers We Love SF March 2026
**Source:**  https://www.youtube.com/watch?v=wzfC7Q-xNik

*This is an AI-generated YouTube transcript -> markdown conversion.*

---

## A Story About Spanner

In 2015–2016, Google was in the middle of a giant data-center build-out and made one mistake:
they forgot to lock in contracts with the Asian DRAM manufacturers. A different hyperscaler got
in there and bought all the RAM. Suddenly Google was in a serious memory crunch. A mandate came
down from the very top — everyone had to save a ridiculous amount of memory. It got subdivided
to the SVPs, then the VPs, all the way down the hierarchy, and eventually landed on me, a lowly
peon working on Spanner, the database that all of Google runs on.

My solution was straightforward: save memory by making the user experience worse.

A database is basically a big pile of caches — query caches, block caches, page caches, caches
all the way down. Google has extremely sophisticated RPC tagging: the moment a request hits
the edge of the network, we know exactly which user it belongs to, and that propagates through
every layer down to Spanner. So we could safely put all of a customer's caches on one machine
and still get fair CPU scheduling, isolation, no noisy-neighbor problems.

But just colocating doesn't save memory — the machine is now 4× bigger. What I really wanted
was to *squish them together*. That would have been bad for customer performance, so my plan
was a new kind of cache where users could burst above their fair allocation when no one else
was using it, and as soon as somebody else showed up we'd preferentially evict from the
over-allocated user. If usage patterns are bursty and power-law distributed (some big
customers, many small ones), nobody notices that we've shrunk all of their caches.

We did a lot of simulations and convinced ourselves it would work. The eviction policy was
very complicated. The dynamic reallocation policy was very complicated. The code was gnarly.
But the interface was simple: insert, read, remove — plus a partition key on insert saying
which customer it was for.

## Property-Based Testing — and Why It Missed the Bug

Anytime you see a really complicated implementation behind a really simple interface, you
should think property-based testing. I was a young hot-shot and I knew this. So I wrote a
harness: pick a random number, decide whether to insert / read / remove, pick random values
for everything else, run it. It found bugs. I fixed them. I ran it longer. More bugs. I fixed
those. Eventually I was running it overnight on one of Google's big Borg clusters — billions
and billions of operations — and it found no bugs. My software was perfect.

I rolled it out. There was an outage. (A P1 single-region outage, not a P0 multi-region one,
because Google's SRE team caught it before it took down all of Spanner.)

Where was the bug?

Look at just one figure: the number of items in the cache while the test is running. Reads
don't affect it. I'm doing inserts and removes with 50% probability each — a nice uniform
random distribution. Each insertion increases the count by one, each removal decreases it by
one. That means *the number of items in the cache is a one-dimensional random walk.*

The thing about 1D random walks is that they are exponentially unlikely to get a certain
distance from the origin — a result called the Hoeffding bound. To get distance 30 from the
origin you need on the order of tens of millions of operations in expectation. My cache's
capacity was much larger than 30. So even with billions and billions of operations, I would
never do enough net insertions to fill the cache, never trigger the eviction algorithm, and
never find the segfault in the eviction algorithm.

## Swarm Testing: The Idea

Swarm testing finds that bug — and many others like it — very simply: by *turning off parts
of your program while you are testing it.*

I have three operations: insert, read, remove. There are 7 non-empty subsets of those three.
Each test run, pick a subset uniformly at random and only enable the operations in that
subset for the duration of the run. Two of the seven subsets involve insertions and no
removals. In those runs there's no Hoeffding bound to worry about — you just get insertions,
the cache fills, the bug fires.

This is striking. Randomness is supposed to be good. We're computer scientists; we like
randomness. And yet pure randomness does not find this bug. Something that looks *less*
random finds it.

(The first example in the swarm testing paper is literally my bug — except with a stack
instead of a cache. When I read the paper, I was floored: I had lived this example.)

## Active Suppression

You might object: this sounds awfully special-purpose — isn't this mostly good for resource
exhaustion bugs? The paper shows the answer is no, because huge numbers of features in real
systems exhibit what they call **active suppression**: running some operation hides an entire
class of bugs.

- Filesystem testing: any call to `sync` hides all bugs related to buffering.
- Compiler testing: if your test program contains pointer accesses, you turn off a lot of
  optimizations and never find bugs in those optimizations.
- Distributed-system fault injection: you have to restart servers sometimes or you'll never
  find restart bugs — but if you always restart, you'll never find slow leaks or timer
  overflows.

So you actually have to disable features of your test system sometimes in order to find all
the bugs.

## Passive Suppression — The Truly Generic Concern

What makes swarm testing universally applicable is **passive suppression**: operations that
don't hide bugs themselves, they just *crowd out* the operations that find them.

In the cache example, `read` is like this. If the cache capacity is 30 and your test runs
last about 30 operations, mixing in any reads makes you unlikely to do enough insertions to
hit the bug. With three operations this is mild; with a system that has dozens or hundreds of
features, enabling all of them with uniform probability makes it really unlikely that any
single one gets called enough times in a single run to trigger a problem.

That applies to all testing of all systems. It's totally generic.

## "Doesn't This Cause the Opposite Problem?"

What if a bug requires a weird *combination* of features all enabled together? The math here
just isn't that bad. Usually you don't need that many completely orthogonal features to
surface a bug — five, seven, ten max. If you do thousands of test runs (and your computer
will do thousands of test runs very fast), you have very high probability that some run
enables them all together. And that run is now *more* likely to find the bug, because the
other features are disabled and there's no passive suppression.

The paper makes this concrete with the YAFFS filesystem — which is close to the worst case
for swarm testing, because basically all the bugs live in the `rename` function. Half the
time swarm testing disables `rename`. You'd think this halves the bug-finding rate. Instead,
swarm testing finds *more* bugs than the non-swarm baseline. Why? Some bugs in `rename` are
only reachable if certain other functions also run; some are only reachable if those
functions *don't* run. Without swarm testing you can't get both.

The paper introduces thousands of artificial bugs into a filesystem and a compiler, then for
each one identifies (a) which functions must run to find it, and (b) which functions, if they
ever run, mean you'll never find it. The scary result is that **the same functions are at the
top of both lists**.

In the compiler:

- 33% of all bugs can only be found if the test program contains pointers.
- 41% of all bugs will never be found if the test program contains pointers.

If your generator never makes pointers, you miss 33%. If it always makes pointers (for any
non-trivial test length, this is exponentially likely — random walk again), you miss 41%. The
only way to have your cake and eat it too is to sometimes randomly disable the feature.

That is the whole swarm testing paper. You can now go apply it and your life will get better.

---

# Connections Beyond Testing

The rest of this talk is more vibes-y, less rigorous. There's still math, but more
pointing-and-gesturing math. I'm not 100% certain of everything I'm about to say.

## Playing Mario With a Computer

At Antithesis we like to play Nintendo games with computers. A Nintendo controller is just a
hardware device — one bit per button. You can feed the output of a random number generator
into the controller. You can get smarter: feed in a coverage-guided fuzzer, instrument the
game, keep inputs that take you deeper, throw away the ones that don't, try to extend.

It totally doesn't work. The computer gets stuck on a pipe, bouncing around, while the timer
ticks down.

Why? Look at one button — jump. White noise means 50% of frames the button is held, 50% it's
released. In Mario, as long as you hold jump you keep jumping; release it and you fall and
can't jump again until you land. **The probability of holding the button down for enough
successive frames to clear a tall pipe is exponentially small in the height of the pipe.**

That's worse than the Hoeffding bound. You get a sawtooth pattern of failed jumps. What you
*want* is long strings of jumps followed by long strings of not-jumps. Uniform-random gets you
nowhere; the thing that looks less random gets you somewhere.

This isn't quite swarm testing — we're not disabling jump across a whole episode, but
disabling it across stretches in some fractal way. What's the right way to think about this?

## Fault Injection

Microservices need to keep working when they can't talk to each other. You need to test the
retry logic, the reconnection logic. You don't know the exact moment when a partition
mid-algorithm would cause a bug. This sounds made for randomized testing — so everybody
says "we'll do deterministic simulation testing, we're so cool."

A very common mistake — I have personally seen it in many DST implementations — is to take
every packet between two processes, route it through a virtual network device, and flip a
weighted coin per packet on whether to drop it. The packet-loss distribution that gives you
looks like white noise. That never finds any retry or disconnection bugs. TCP retransmit
saves you. Some networking library saves you. It's like flaky Wi-Fi.

What you want is long stretches where everything works and the system can get into an
interesting state — then a partition where everything drops. The same picture again.

## Load Testing

You're building a service. In real life, all your users are asleep for hours, then you get a
huge spike because you're on the front page of HN. That's why people load-test.

Fewer people make the mistake here than in DST, but you *could* build a load tester that
makes a uniform-random decision at each tick whether to send a request. That isn't load
testing — that's simulating uniform load with a little jitter. What you want is bursty
traffic. Same picture again.

## Financial Markets

Black–Scholes is a very influential option-pricing formula. It tells you the value of an
option in terms of the asset's volatility — more volatility, more chance the option randomly
becomes in-the-money. It won its creators the Nobel Prize and was how basically every bank
priced risk for decades.

It assumes the market is a **martingale**: a memoryless process where what happens Tuesday is
independent of what happened Monday. Brownian motion, random wiggling. The justification was
the efficient market hypothesis: all information is priced in at any given moment, so the
next move is purely driven by new information.

This was a great assumption until it wasn't, and the world economy blew up in 2008 — largely
*because* of Black–Scholes. Two people who get tremendous credit for calling this out
beforehand: Benoit Mandelbrot (father of chaos theory) and Nassim Taleb (prolific Twitter
poster). Their 2006 paper "A Focus on the Exceptions That Prove the Rule" analyzed US equity
price history and just said: the efficient market hypothesis is *empirically wrong*. They
showed the S&P 500 chart, then the S&P 500 chart with the 10 most important days removed —
completely different graphs, which should be impossible under a martingale.

They call the alternative the **fractal market hypothesis**: the price chart has fractal
dimension. Black–Scholes says markets behave like white noise; Taleb and Mandelbrot say they
behave like the bursty picture. Same two diagrams again. I'm losing my mind.

## The Weather

One of my favorite books: the 1951 edition of the *Transactions of the American Society of
Civil Engineers*. On page 770 is "Long-Term Storage Capacity of Reservoirs" by Harold Edwin
Hurst — possibly one of the most important papers ever written.

Hurst was a British civil servant sent to colonial Egypt to administer the empire. He was a
mathy guy and an amateur meteorologist (a brand-new science at the time). He landed in this
arid region and started cataloguing: maybe it rains one day in ten. Then he looked around
and realized **all the existing water infrastructure — dams, reservoirs, irrigation canals —
was way too big**, by orders of magnitude.

Everybody at the time believed weather was a memoryless stochastic process with a uniform
distribution. Hurst gets enormous credit for not just assuming centuries of dam-builders were
stupid. He actually looked at the rainfall data. It looked nothing like white noise — it
looked bursty. And under the bursty model, the dams are the correct size.

He quantified the difference into what we now call the **Hurst exponent**:

- H = 0.5 — the process is a martingale; every step independent.
- H < 0.5 — mean reversion (what people incorrectly believe about lightning strikes).
- H > 0.5 — the **Joseph effect**: seven years of plenty, seven years of famine.

Weather has a very high Hurst exponent. So does traffic to your website. So does the behavior
of network connections in a datacenter. So do financial markets. And so does user behavior:
when people use a cache, they don't send random strings of insertions and deletions — they
send long strings of insertions, then long strings of deletions. That's why production found
my bug very fast even though my test didn't. Production was using a much more realistic
distribution than `/dev/urandom`.

It's tempting to stop here and say: swarm testing works because it forces the operation
time-series to have a higher Hurst exponent than the Linux RNG gives you. Unfortunately, I
don't think that's the full story.

## Fuzzing — No Time Series Here

Suppose you're fuzzing a service parsing protocol messages from the internet. One field is a
uint32. Without knowing anything about the protocol, what value should you put in to find
the most bugs?

Audience: max, 0, −1, 1.

We all have this intuition, and it's correct. AFL has a file called `interesting_numbers` in
its source repo with exactly these values (plus a few you wouldn't expect). A boring number,
in binary, looks like white noise. An interesting number — all zeros, all ones, alternating
patterns — looks bursty.

Now I'm very scared, because **this is not a time series.** There's no Hurst exponent here.
This has nothing to do with long-range dependence in a stochastic process. And yet it's the
same two diagrams.

## Meta-Swarm Testing

I tricked you earlier. I told you swarm testing finds lots of bugs because the odds of
enabling some specific combination of features is actually pretty high. But the odds of
enabling *all* the features together, or just *one*, are governed by the binomial
distribution — and binomial distributions have astonishingly little probability mass in the
tails.

So when you move up a level — from operations to combinations-of-features-enabled — swarm
testing looks like white noise. Meta-swarm testing (sometimes turn everything on or
everything off) restores the bursty picture. And presumably I also need meta-meta-swarm
testing, and meta-meta-meta-swarm testing, in infinite regress. We need to figure out what's
going on and solve it at its root.

---

# Knightian Uncertainty

Francis Knight, an important 1920s economist, has this wonderful quote:

> Uncertainty must be taken in a sense radically distinct from the familiar notion of risk,
> from which it has never been properly separated. The essential fact is that 'risk' means in
> some cases a quantity susceptible of measurement, while at other times it is something
> distinctly not of this character; and there are far-reaching and crucial differences in the
> bearings of the phenomena depending on which of the two is really present and operating.

We use the word *risk* to mean two completely different things. One is risk as the output of
a model — actuarial tables, the odds of dying in a car crash, things statistical models will
tell you. The other is the risk that *the model is wrong*. We don't like thinking about the
second kind. We're very bad at it.

Knight says we should use a different word: **uncertainty**. (Today people often call it
*Knightian uncertainty*, or *model error*, or — per a former secretary of defense —
*unknown unknowns*.)

The first kind of risk is quantifiable. The second is *definitionally* unquantifiable, because
it's the residue left over after you've quantified.

Maybe part of why swarm testing is so good is that it is a way of **systematically and
structurally forcing you to throw away your models**. Whatever distribution you believed your
data had, whatever distribution you believed users would send you — occasionally, swarm
testing just turns it all off and does something completely different and shocking.

Human beings are systematically biased in the statistical models we create, in a pernicious
way. The models we build look like white noise — that's what `/dev/urandom` gives you,
`random.random()`, every stdlib everywhere, every stats class, the whole economy. And it's
not that great. In a micro sense it's very random; in a macro sense it's the most boring
picture ever. Same shade of gray everywhere. No repeating patterns. Same at every zoom level.

The world doesn't look like that. The world looks like the Mandelbrot set — also
unpredictable, but in a completely different way. A super low-entropy picture (the formula
that generates it is very short), full of structure at every scale. If the models we build
prepare us for the first picture and the actual world looks like the second, you might be
okay — depending where in the picture you land.

---

# The Optimal Distribution Conjecture

A ludicrous question: suppose you have a box and you have no idea what's in it. It could be
any computer program. Or your financial portfolio. Or your life. You have an adapter that
maps binary strings into operations on the box. **What is the optimal distribution over all
binary strings for finding interesting behaviors or bugs?**

How could that question possibly have an answer? And yet, after everything above, I feel
like we can say something.

- A purely random string is probably *not* a very good choice. The odds it finds anything
  are just (interesting states / all states), which is a bad ratio.
- You should sometimes generate purely random strings, because all models are wrong —
  including the new model that randomness is bad. But the weight on any individual random
  string should be very low.
- Strings like `00000000`, `11111111`, `1111…10`, the Fibonacci series — these will find
  bugs.

I want a mathematical tool that distinguishes those strings from the random one. And — same
tool — distinguishes the white-noise picture from the Mandelbrot picture.

That tool exists. **Kolmogorov complexity**: for a binary string, the length of the shortest
computer program that prints it. A long random string's shortest program is essentially the
literal string. A structured string's program is much shorter.

**Conjecture: the optimal distribution for finding bugs in any computer program is to sample
from all binary strings with weight inversely proportional to their Kolmogorov complexity.**

If this is true, it explains why swarm testing works: swarm testing is a way of artificially
biasing down the Kolmogorov complexity of your inputs. It's a primitive, caveman way of
doing it. But it does it, and it's freakishly effective.

## Objection: This Is Impossible

You might object that the space of all programs contains an infinite number of monsters
designed to defeat any strategy. That's a true criticism — it's how most impossibility
results in theoretical CS work (diagonalization).

Fine: I amend the conjecture with an asterisk. We're finding bugs in programs **written by
human beings (or human-like entities) for a business purpose.** We are not testing random
piles of spaghetti adversarially designed to defeat our testing strategy. We're testing
programs that were made to accomplish a goal in the world. That's a completely different
domain — and I think it means good strategies exist.

Have we solved software testing? No. Kolmogorov complexity is completely uncomputable —
finding it for a single string isn't guaranteed possible on a Turing machine, let alone
building a probability distribution over all strings. But that doesn't make it useless. If
swarm testing works because it's a fast, crude approximation to this ideal distribution,
maybe there are *other* computable approximations that work even better. At Antithesis we
have research projects directly inspired by this conjecture, and some are yielding very
exciting results.

---

# AI, Because We're in San Francisco

*Universal Artificial Intelligence* by Marcus Hutter is a book from about 25 years ago.
Hutter is possibly the smartest person alive today.

People these days talk about *super intelligence* — an algorithm that, dropped into a
situation, can learn how the world works well enough to take actions that achieve its
objectives. Hutter says that's kid stuff. He wants **universal artificial intelligence**:
something that behaves like a super intelligence not just in *this* world, but in **any
possible world** — drop it into a universe with completely different physics, different
causality, and it would still be optimal.

The insane thing is, he writes down an algorithm. It's hilariously uncomputable, but it's
true and correct. He proves it's Bayes-optimal. The algorithm is a mixture of old-fashioned
reinforcement learning and **Solomonoff induction**, which is basically weaponized Occam's
razor — Occam's razor with gain-of-function applied. You take all the information you've
received so far, construct all possible hypotheses for how the universe works (the true state
machine governing reality), sort them by Kolmogorov complexity, pick the lowest, use it to
take one action, observe what happens, repeat.

It's the most uncomputable thing ever. But again — if that algorithm gives you a universal
AI, maybe there are computable approximations that are still pretty good.

There's somebody in this city very interested in computable approximations to AGI: Jan Leike,
a student of Hutter's, formerly head of alignment research at OpenAI (alignment research is
another name for capabilities research), now at Anthropic working on something even more
secretive.

We used to joke at Antithesis that if we got sufficiently good at testing software, we'd
invent AGI by accident. This now doesn't feel like such a joke. If optimally searching very
complex software programs for bugs, and universal AI, are both deeply connected to the
problem of *fast computable approximations of Kolmogorov complexity* — there's something
interesting there.

---

# Q&A

**Q (Paul):** Seems like there's an obvious connection to quantum mechanics and the
randomness we measure in the wave function.

**Will:** You tell me about it — happy to hear it another time.

---

**Q:** Essentially what we want here is to randomize the randomness. Swarm testing is one
level of that — randomizing over distributions, sometimes pinning probabilities to 0 or 100.
A next step would be to let those probabilities move across regions over time.

**Will:** Yes — that's a technique we've been using for a while: smear swarm testing on and
off over the course of a test run. I don't know if I'm giving away a trade secret, but yes,
we do that, and it works very well. (Restating for the recording: he proposed that
swarm-testing is "randomizing your randomness," and that you can take it a level further by
dialing the amount of swarming up and down over the course of testing. That's meta-swarm
testing — and there are many more levels above it.)

---

**Q:** How does a test look in this case? A unit test has a predictable flow; an integration
test a bit more interaction. If it crashes, that proves a bug — but how do you generalize so
tests benefit from this predictably?

**Will:** Think of it as a generalization of how you do unit tests. A unit test expects a
particular answer: insert 3 into the database, expect to read 3 back. The next level is:
insert N, expect to read N — that's property-based testing. It's one level of relaxing the
constraints, and it immediately gives the test a lot more power. You just have to think of
things you expect to always be true about your software. Treat them as assertions, litter the
code with them, and turn an autonomous testing system loose.

---

**Q:** For a filesystem, selectively turning off operations to no-ops makes test outcomes
hard to reason about.

**Will:** Good clarification. Sometimes the easiest way to implement swarm testing is to
actually turn off parts of your program — especially when that's an acceptable part of the
contract. Other times the practical way is to make your *test generator* never generate those
operations. That can be the API methods being called, or the faults being injected, or
whatever. That's how you can do this non-intrusively to somebody else's software.

---

**Q:** What about Shannon entropy as an approximation for Kolmogorov complexity?

**Will:** Loosely, it's kind of the opposite. There are two ways to talk about Kolmogorov
complexity: as *information gain* and as *compressibility*. As compressibility, it's the
opposite of Shannon entropy. As information gain, it's in the same direction. So — a
sign-convention thing.

---

**Q:** Can you use compression as an approximation of Kolmogorov complexity — try many
compression algorithms and use the lowest?

**Will:** Absolutely, and we do. Before Hutter invented AGI, he ran a compression competition.
He was a big believer in measuring AI by how effectively it can compress things. That
approach has been largely abandoned by history, but if you go to his website he still has
a ton of cool stuff about it. (Audience: "I'd argue we haven't abandoned that, just renamed
it." Will: "Maybe so.")
