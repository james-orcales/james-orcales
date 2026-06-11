# When to Use Soft vs. Hard Assertions

Hey guys — Firefox Slideshow. Today we'll be talking about when to use soft or hard assertions.

Assertions are great for enforcing the assumptions or rules of your system, giving you clear and
actionable feedback upon violation. An assertion is the enforcement of a condition to hold during
program execution. Hard assertions crash the program upon violation; soft assertions don't crash.

In this debate, it usually boils down to: when you're in a long-running context, use soft
assertions; in a short-running context, use hard assertions. And I think this is too reductionist —
there are a lot of factors that go into whether you use hard assertions or not, specifically in
production.

## Case study: TigerBeetle

Let's take a look at TigerBeetle. What is TigerBeetle? TigerBeetle is an online transaction
processing database. It records business transactions in real time. Simply put, it's a long-running
context — it's a database server. This is their website, tigerbeetle.com.

### How TigerBeetle uses assertions

Assertions crash the program upon failure. Assertions are enabled in production, retaining their
crashing behavior, and there are 11,000 assertions in their codebase.

### Why TigerBeetle uses assertions aggressively

Financial transactions have zero margin for error. They have in-house deterministic simulation
testing, which simulates decades of usage for their database, exposing their application to a
multitude of scenarios. They have fault injection, so network failures and disk corruptions happen
at a much higher rate. In essence, they simulate their software in a much harsher environment
compared to production.

This gives them confidence that whatever assertion failures they may have introduced into their
application will get caught during testing. Assertion failures will be rare during production, and
if there are any, then that signals a deficiency in their testing and requires urgent attention.

Lastly, Zig assertions influence compiler optimization. With any real-time processing domain, you
want utmost performance, and in Zig — the programming language they wrote TigerBeetle in —
assertions influence compiler optimization, especially in release mode. (Though not all languages
have this kind of feature.)

### The domain-dependent factors

What you're seeing here is the domain-dependent factors dictating the practicality of hard
assertions in production:

- **Correctness guarantees** — the criticality of the entire program's correctness at runtime. When
  you have an assertion failure, can you recover from it? Can you afford to continue running even
  though some feature of your program is incorrect? With TigerBeetle: no. So hard assertions prevent
  undefined behavior from executing.
- **Testing rigor** — the depth and coverage of your tests. Hard asserts are safe if your tests
  catch almost all bugs; then production failures indicate urgent issues.
- **Performance demands** — the execution-speed requirement. This is only really relevant where
  assertions affect compiler optimizations.

## A formula for hard assertions

So when do you use soft or hard assertions? The three factors determine whether hard assertions are
practical. Treat them multiplicatively: a weakness in any one factor makes hard assertions
impractical. The formula I created is:

> **practicality of hard assertions in production = correctness guarantees × testing rigor ×
> performance demands**

An example: without testing, hard assertions would crash your application frequently. So 1 × 0 × 1 =
0 — you should never use hard assertions in production. With TigerBeetle, our golden boy, every
single factor is a 1; it's maxed out. So should they use hard assertions? Yes, they should — and
they do.

*Disclaimer: this formula and the values provided are arbitrary. They have zero academic foundation
and are intentionally flawed. They simply serve to frame your perspective.*

## Case study: Odin

Next up is Odin. What is Odin? Odin is a systems programming language with manual memory management
— here we refer to its official compiler implementation. This is their website, odinlang.org.

How does Odin use assertions? Assertions crash the program upon failure. Assertions are enabled in
production, and they have 2,000 assertions in the codebase.

Odin's factors: compilers demand absolute correctness, so it's maxed there. Unit testing and fuzz
testing is enough, because compilers are not highly concurrent or stateful like a backend server —
so something like autonomous testing has less value for Odin.

Performance demands: Odin specifically prioritizes compilation speed through high-level
architectural choices, without chasing micro-optimizations. Instead of micro-optimizing the front
end by minimizing its memory consumption, they simplify the type system of the language itself, and
aim to replace the LLVM backend with their own. Compare this to something like Rust, with a rich
type system but significantly slower compilation speed. And something like Zig — I remember this one
pull request where the contributor improved the tokenizer speed by 10%, but they introduced a little
bit more memory (I think 5 to 10% more, in the single-digit megabytes, I don't know), and the pull
request got rejected because the team — Andrew Kelley specifically — was prioritizing minimizing the
memory consumption of the tokenizer. I personally would have accepted the pull request, but I
digress.

So with Odin, we have a 1 in correctness guarantees (they need absolute correctness), 0.66 in
testing rigor, 0.8 in performance demands. So should they hard assert? Maybe, maybe not. But they do
use hard assertions.

And you can notice that both TigerBeetle and Odin need absolute correctness, and they both use hard
assertions in production. So maybe, as long as you need absolute correctness, you should use hard
assertions? Well, let's take a look at Antithesis.

## Case study: Antithesis

Antithesis is an autonomous testing company, providing deterministic simulation testing to any
software application. You guide the Antithesis simulator using the assertion SDK. This is the
website, antithesis.com.

*Clarification: in this discussion, we are reviewing the assertion practices of Antithesis's
customers, not Antithesis's internal practices themselves.*

How does Antithesis implement their assertion SDK? Soft assertions in all environments, with varying
behavior: for development and production, you have no-ops or logs; while in testing, assertions send
metadata to the Antithesis simulator (I think through HTTP requests).

Antithesis customer factors: **correctness guarantees** — since Antithesis's aim is to eliminate all
your bugs, assume absolute correctness regardless of domain. **Testing rigor** — they have access to
deterministic, autonomous testing. **Performance demands** — the SDK supports JavaScript and C++
clients, so it varies, but for our purposes let's consider a high-performance customer, something
like MongoDB. And you can see here, MongoDB.

So, just like our golden boy TigerBeetle, Antithesis has a 1 in every factor. So should they use
hard assertions? Definitely. But do they use hard assertions — or do they recommend their customers
use hard assertions? No.

Antithesis shows that absolute correctness is not the sole factor in whether you should use hard
assertions in production. Unlike TigerBeetle and Odin, the insight is: Antithesis isn't a
long-running service, so soft assertions have little value there (also, how would a compiler even
recover?), and it treats assertions as metadata. Soft assertions let the simulator explore failure
severity instead of aborting. This also demonstrates to customers the full extent of bugs prevented,
by comparing behavior before and after integration.

So with soft assertions, this allows the machine learning algorithm to further categorize assertion
failures by their severity and what kind of bugs they lead to — because not all assertion failures
are equal. Some assertion failures are just typos, or some part of your UI not showing up; some will
crash your server. And as a company, you've got to remember that Antithesis is integrating with
existing codebases. So for their customers to see the value they bring, they need a clear
before-and-after comparison of how their codebase has improved — what kind of bugs were uncovered —
without changing the logic of the program at all.

Essentially, Antithesis's testing recommendations are irrelevant to production environments, and
their customers should still be thinking about whether they should be using hard or soft assertions
in production.

## Case study: Patient

Next is Patient. Patient is a SaaS fintech — a health payment account provider that finances
out-of-pocket healthcare expenses at 0% interest. In other words, Patient fronts all of your
healthcare expenses, and they're allowed to do this because of their sponsor. This allows you to pay
your healthcare expenses periodically, just like credit. This is their website, patient.com.

How does Patient use assertions? Assertion failures return an HTTP 500 response and announce
failures in Slack. Let's look at this link, which is an interview with Isaac:

> In our case, we're writing web servers, and if some assertion fails in one isolated corner of my
> server, I don't want that to result in all of production going down. That would be horrible — I
> would not sleep, that would be terrible. So the way we have it implemented for our use case
> instead is that when an assertion fails, that request completely fails, returns a 500,
> unrecoverable — but everything else in the server keeps operating as expected. Just because
> something went wrong in the code for handling one request doesn't mean I need to shut down
> everything else. So that also really limits the blast radius: yes, you're getting this feedback,
> you're going to know immediately when something goes wrong, but it's just isolated to the code
> that has the issue in it.

Okay, that's more of the setup. And then they also announce failures in Slack. Let's take a look at
that part of the interview:

> On my team, we have a bunch of SQL files, and each of those SQL files returns a count — only a
> number of rows — and each one of those represents an invariant that must always be true of our
> data. And then every day we run those against a read replica. And if one of those queries returns
> any rows, or a non-zero count, then we'll send a message out in Slack and say, "Hey, this
> invariant was broken. We expect this to be true over this whole database, and it wasn't, in this
> one way." I'm very excited about that — we've already caught a bunch of issues using that.

So I assigned these values for Patient: 0.7 for correctness guarantees; 0.6 for testing rigor,
because they don't use autonomous testing (in the video it was mentioned that they don't use any
kind of generative testing or fuzz testing, but Isaac was planning on doing that eventually); and
performance demand 0.7 — I'm giving them the benefit of the doubt, and they're using Go, but maybe
they use JavaScript on the backend, who knows? Should they hard assert? Probably not. And do they
use hard assertions? No.

And again, I'll remind you that these values are arbitrary — you could argue this value could be
higher or lower, whatever. This is only framing your perspective.

## The formula doesn't predict practice

So there's zero correlation between the formula's output and an organization's actual testing
practices. You can see that TigerBeetle has everything at 1 and they use hard assertions, but
Antithesis doesn't use hard assertions. Odin should probably not use hard assertions — or it's
inconclusive, to say the least — but they use hard assertions. With Patient, same idea, but they
don't use hard assertions. You see, the formula doesn't take into account other environments, or the
value of soft assertions in each domain.

## So when do you use soft or hard assertions?

**In development:** use hard assertions while developing locally. Littering your console with
assertion logs is not helpful — you just need to see the first failure with a stack trace.

**In testing:** if you have autonomous testing, use soft assertions — use the metadata of your
program to help guide your autonomous testing simulator. Anything else (unit tests, fuzz tests,
snapshots), use hard assertions.

**In production:** I created a diagram here, and I was too lazy to make a digital equivalent, so I
wrote it on a whiteboard. (Look at that handwriting — I write like a girl.)

- Do you need **absolute correctness** — the entire program needs to be correct, you cannot afford a
  single feature to be wrong, you cannot recover from a failure?
  - **No** (you can recover from some assertion failures) → use **soft assertions**, similar to
    Patient's backend server: return an HTTP 500 for assertion failures.
  - **Yes** → Are you in a **long-running context**?
    - **No** (you're just a CLI tool, similar to Odin) → use **hard assertions** in production.
    - **Yes** (similar to TigerBeetle, a financial database) → Do you have **autonomous testing**?
      - **Yes** → use **hard assertions**.
      - **No** → use **soft assertions**, because you don't have the confidence that your assertions
        were exercised thoroughly during testing.

## A code example: soft assertions in a Go backend

Most likely you'll be using soft assertions in production, so I wanted to give you an example of how
that may look. Let's go to the backend. This is a backend server that uses soft assertions. I used
this in my previous video, "Implementation-Driven Testing," but I modified it a little: now it
panics, and we recover from that panic.

The idea is: we open a TCP server, and we send text to it (the client). The server saves this text
in a database, incrementing every time it's seen. Whenever there's an assertion failure, the server
will panic. To continue running, we recover from this panic here, and we periodically check — in a
separate goroutine — how many assertion failures happened. So if there's any assertion failure
within X amount of time, then we send an email to someone and they need to fix that issue. Here we
check it every 30 seconds. And to send an email, we simply use SMTP here with Google app passwords.
I have my credentials in a separate file — I'm not going to show you that.

It's hard to visualize this, so let's run the program instead. `go run`. Now we're listening. And
netcat on port 49447. Now we can start typing anything. Hello. What? What? And let's do... "you gave
me up." Okay, now we have an assertion failure. Let's do that again. "You gave me up. You gave me
up." There you go — it announced the assertion failures via email.

Let's do another assertion failure, and after this gets announced via email — actually, let's shut
down the server immediately. So what the server will do is, upon shutdown, it will flush out every
assertion failure and do one final check to announce it via email. So we should expect two emails:
one that says three assertion failures happened within the last 30 seconds, and another that says
one assertion failure happened within the last 30 seconds.

I'm going to go to my Gmail account. And there you see: "Detected three assertion failures in the
last 30 seconds." "Detected one assertion failure in the last 30 seconds." Zero minutes ago.

So the code is available on GitHub — I'll put the link down in the description. Hopefully you
learned something from this video, and you can apply soft and hard assertions in your own projects,
and hopefully in your workplace.

The end. Stay hydrated.
