# Replacing Bash with Lua or Dash

Before we start, drink some water. Okay.

The goal is effective shell scripting. The objectives are simplicity, portability, and
reproducibility.

## Objectives

These three objectives overlap with each other — the value of one feeds into the others. Simpler
means better portability and better reproducibility, and vice versa. We don't need to be too
pedantic; these simply serve as a general guideline for us.

- **Simplicity** — how well our solution fits the problem.
- **Portability** — the ease of executing our script.
- **Reproducibility** — the consistency of its behavior across environments.

## Simplicity

Expounding on simplicity: the more we expose ourselves to irrelevant abstractions, the less simple
our solution becomes. For example, using Bash — which is written in 176,000 lines of code — instead
of Lua 5.1, which is 14,000 lines of code, exposes us to functionality unrelated to the problem, and
by virtue, more bugs.

Same idea with shelling out to external commands or libraries instead of maximizing the built-ins of
the language or the runtime; relying on tricky syntax or semantics, which diverts our attention to
using the language effectively rather than solving the actual problem; and creating an abstract
database factory for the day that we support 15 different backends.

## How bash is invoked

There are two ways to invoke bash: the normal way, and through `sh`. `sh`, if you didn't know, is a
symlink wrapper to another shell. On most distributions it points to bash; on others it points to
something like dash instead. Users can also change what shell it points to.

Bash has this feature where it can detect when you invoke it as `sh`, and when that happens, it
enables POSIX mode — it modifies its behavior to not conflict with the POSIX standard. However, this
does not imply that it disables all non-POSIX features. In this discussion, we'll take into account
both of these invocations.

Take a look at the script. We declare an array, do arithmetic evaluation on the elements of the
array — we index the array, take two numbers, add them together, and they should equal the third
element. Assume that `sh` points to bash. All of these are non-POSIX features, but bash will run
this script successfully even in POSIX mode. On Debian, where `sh` points to dash, this script will
fail to run. Oh — there you go.

## Use cases

Use cases to accommodate: build systems, I/O-heavy automation, CPU-heavy tasks, system management.
We'll focus on build systems and I/O-heavy automation. We'll touch on CPU-heavy tasks, but mostly
parsing, and we'll forgo system management altogether.

`ps`, `top`, `mount`, `df`, `du`, batch data processing, numerical computation — all those things
are nothing more than shelling out to an external command, since you'd want to use executables
optimized for these tasks.

So when I say build systems, I'm talking about bootstrapping — a problem that most projects need to
tackle. Dev environment setup: if your project officially supports Linux and Mac, you'd want an
official way to set up the development environment. Other projects use Nix for this —
`configuration.nix`. Package deployments, your release cycle. And I/O-heavy automation: I'm talking
about log processing, data pipelines, streaming.

## Factors to consider

Factors to consider are functionality, supported platforms, and appetite for vendoring. The
importance of the objectives is directly correlated with the gravity of your factors. Functionality:
if you just need a simple ten-line bash script, and your supported platforms are macOS and Windows,
    then this video isn't applicable to you, and you should probably just use Bash instead.

## Evaluating Bash

### As a language

It has tricky syntax and semantics. It's a large superset of POSIX shell — like I said, 176,000
lines of code. It's process-based parallelism, centered on orchestrating CLI utilities (a.k.a.
external commands). Most commonly, you'd be using the coreutils.

This is problematic because of the conflicting implementations of the coreutils. On Linux you have
the GNU coreutils; on macOS you'd have the BSD counterparts. Each implementation introduces
non-standard flags and behaviors that the other might not support. And both inevitably have their
own bugs, simply because they're independent C implementations of the same standard. Because of
Bash's tricky semantics, people are pushed toward the coreutils, which have a relatively saner
interface — but in my opinion, the coreutils are hot garbage regardless.

### Ecosystem

It's pre-installed on most Unix systems. There are fragmented versions across environments. On macOS
you'd have bash version 3.2 — that's the last MIT-licensed version. On Debian and Ubuntu you'd have
5-something. And legacy systems — I'm talking about systems that refuse to upgrade from Debian 10 —
they'd have an older version of Bash.

And like I said before, `/bin/sh` is not guaranteed to symlink to bash. On Ubuntu/Debian that's
dash; on Alpine it's ash; on FreeBSD it's an actual executable — `sh` is not a symlink wrapper
there.

### GPL licensing

It's GPL-licensed, which makes it difficult to vendor. If you're not familiar with licensing: GPL is
a copyleft license. It's a viral license. What GPL does is force software that interacts with it —
be it as a library or as an executable — to be open source. Now, that's not a problem if you want
your software to be open source. But if you're a company with proprietary code, that makes GPL
software hard to interact with.

Now, you might know that the Linux kernel is GPL-licensed, but that's a different kind — it's the
Lesser GPL license, which permits software to interact with it, and yada yada. It's complicated, to
say the least. That's why companies tend to avoid the GPL license.

And it matters that it's difficult to vendor, because vendoring software helps the portability and
reproducibility of our project. We don't have to think about package managers, or where people get
this version of bash, or what version of bash they have. With vendoring, we avoid all those
headaches.

Now, this is some bash code — real bash code that I wrote to bootstrap one of my projects. I'm just
showing this as a refresher of what bash code looks like. This is an unofficial changelog of Bash;
it goes all the way back to version 2.0. You can see how, for as big a language as Bash is, it's
only getting bigger — creating more version fragmentation across the ecosystem. I mean, look at this
— what the hell even is this `printf` nonsense right here?

### How Bash fulfills the objectives

It's definitely not simple. Portability: the greatest thing about Bash is that it's installed
everywhere, making it easy to run bash scripts. Now, whether those scripts run as intended by the
author — that's reproducibility — that's iffy. So Bash is not conducive to effective shell
scripting.

## The options

What are our options, our solutions? Don't use bash; pretend bash is dash; or use bash.

## Solution 1: Don't use bash

Alternatives are Lua 5.1 and Dash. I understand that Lua isn't a shell-y language, but it is a
worthy alternative. I'll show you a code snippet later of how I use Lua to emulate some shell
features.

### Lua: the inspiration

As I mentioned, one of our use cases is build systems. I got inspired to use Lua because of Ginger
Bill, the creator of the Odin programming language. These messages are from the official Odin
Discord channel:

> Most build tools are fundamentally string-manipulation tools, and at the end of the day, a
> scripting language like Lua or Python is a lot nicer to use as a build language.
>
> — Ginger Bill

### Adopting (vendoring) Lua

So how does Lua compare to Bash? Let's talk first about adopting it. You download the source code,
you compile from source, and then you commit both the source and the executable to VCS, usually Git.
This is what I was referring to as vendoring. Lua is one of the easiest software applications to
vendor out there.

If you can get over the portability hump of it not being pre-installed anywhere, and instead vendor
it in your project, then all I need to do is show you that using it is better than Bash. Again, I
got this idea from Ginger Bill. He says, with regard to cross-platform stuff:

> I usually embed Lua into the repo itself, so you can just call it there and then on all platforms.
>
> — Ginger Bill

### Evaluating Lua as a language

Let's get it out of the way: it's one-based indexing. Yes, I actually prefer it this way. Some
people might hate it — it's up to you.

It's a small language: 14,000 lines of code. Compare that to Bash's 176,000 — that's, I believe,
12.5 times smaller. Yes, 12.5 times smaller than Bash.

It's single-threaded concurrency with coroutines. When it comes to parallelism, it's worse than
bash, being single-threaded — however, that does improve its determinism. If you really need
parallelism, you can use Lua Lanes or LuaJIT, or you just embed some bash script inside your Lua
script using `os.execute`, and then you have access to parallelism again.

It's also finished software — the last patch was released back in 2012. And it includes a basic
standard library. Now, Lua has never been a batteries-included language; this is a negative on its
part, but it comes with the trade-off of flexibility. We get to choose what functionality is
introduced to our system, and how.

In networking, we have multiple options, sorted by degree of control: we can use the standard
library, shell out to `curl`, or use LuaSocket.

Now I have a story. I created a bootstrapping script to set up my development environment for macOS.
It was written in Go. I wanted to avoid external commands completely. I used the standard library to
make my HTTP requests, extract archives, do checksums, all that jazz. It was a good learning
experience, but it was a painfully obvious example of over-engineering in hindsight. I ended up
shelling out to `curl` and `tar`, since they were pre-installed on the system and I'd most likely
only be using Linux and macOS for my developer machines anyway. So there's that. There are a lot of
ways to introduce functionality into your system — it doesn't have to be pure Lua or pure Go. You
just have to be aware of what bullets you're biting.

### Lua's ecosystem

It's not pre-installed anywhere, but it supports practically all platforms under the sun. How? Well,
one: the Lua source code is written in ANSI C — that's the C89 standard — and almost all modern C
compilers can compile Lua. So as long as you have access to a C compiler, you can compile Lua from
source.

On top of that, Lua avoids a lot of platform-specific code — and this is related to the deficiency
of Lua's standard library. You have to implement basic convenience APIs like file-path handling,
setting environment variables, and multi-threading, because, remember, a lot of the functionality in
Lua's standard library is just interfacing with the C standard library. And when most people are
only going to be using Linux, macOS, and Windows anyway, this specialization on portability actually
hurts my case. That's why one of my project ideas is creating a standard library for Lua — something
straightforward to vendor.

Moving on — it's also MIT-licensed. This is the complete opposite of GPL licensing; MIT is one of
the most permissive licenses out there. Besides unlicensed software, all you need to do is retain
the copyright attribution to the authors and contributors of the software. And vendoring the source,
the documentation, and let's say three executables for Linux, Windows, and macOS totals 1.66
megabytes. That's a trivial storage cost to your Git history, Git work tree, or whatever. And
open-source libraries are mature.

### Emulating shell features in Lua

This is real Lua code that I wrote to bootstrap another project of mine. Taking a look at the first
function, `sh`: this is basically `os.execute`, but it also emulates shell piping. If you take a
look at the second function, `check_health_odin`, I use `sh` here. Looking at the second invocation,
the last argument is a pipe — what that does is, now `sh` will return the string output of this
command. Then I match that with my expected Odin version. Looking at the first invocation, the last
argument is not a pipe, and now `sh` will return a boolean indicating whether this command was
successful or not.

Another shell feature that I emulate is sourcing — `source .shrc`, `source .bashrc`. There are three
main functionalities we need to implement.

First is mutating the process environment. We copy the process environment and manage it on our own,
since Lua does not have a `setenv` function in the standard library. Then we override `os.getenv`
from the standard library to use our copy of the environment instead.

Next, we need to make the rest of the program aware of our updated environment, and propagate that
to any external commands we execute using the `sh` function we created earlier. Let's extend it to
check which environment variables have changed since program startup, and then prepend that to the
command it's executing, right here.

Finally, sourcing: we source the file and sync with the new environment. We start a new shell,
source the file, print all the environment variables, then iterate over those variables, updating
our current process environment. Sourcing has numerous side effects; this only cares about
environment variables. Other side effects include aliases, variable and function declarations, and
shell options — and there are numerous ways to sync with these side effects. So I'll let you figure
out how to do those things, but in most cases you'll only care about the environment variables.

And I think I skipped over this comment I had — yeah. If you import a library that exposes the
`setenv` API — let's say you're using LuaJIT, or you're using luaposix — you can avoid all of this
nonsense.

### Why Lua 5.1 specifically?

The Neovim documentation is a succinct explanation for this: it's minimal, it's frozen, and it's
targeted by LuaJIT. Later versions beyond 5.1 are essentially different, incompatible dialects.

Lua 5.1 is a complete language. The syntax is frozen. This is great for backwards compatibility.
That's why you'll see a lot of third-party libraries have Lua 5.1 as a minimum supported version.

Then there's LuaJIT — a just-in-time compiler that turns Lua bytecode into native machine code on
the fly, making it one of the fastest runtimes on the planet: 10 times faster than Python. Now, I
avoid performance benchmark metrics in this discussion because it doesn't matter — it really doesn't
matter. But when you're talking about one of the fastest runtimes on the planet, I have to mention
it.

### Lua 5.1 ecosystem (as of 2025)

You've got Penlight, which brings the Python standard library to Lua. It has 6.22 million downloads
— release date was this year, actually. LuaSocket: 6.6 million downloads — that's for your
networking. Last release date 2022. LuaJSON: this is your `jq` replacement, 22 million downloads. So
what I'm trying to show here is that the ecosystem is mature — libraries are stable, and there are
millions of downloads across all of them. By the way, these statistics are from LuaRocks.

### How Lua fulfills the objectives

**Simplicity:** it is 100% simple. **Portability:** very portable, as long as pre-installation is
not a requirement. Arguably it's too portable, which limits its standard library, when 90–99% of you
watching are just using Linux, macOS, Windows. **Reproducibility:** it has high reproducibility
since it's finished software. It's small and easy to vendor — with a couple of caveats. However, the
script can still be run by a Lua version beyond 5.1, or by LuaJIT, which has numerous non-standard
extensions — though guard-checking against these environments can be done with a simple `if`
statement.

### Dash

Dash — the Debian Almquist Shell (Almquist... Almquist, I don't know how to pronounce it). Hardcode
Dash as the interpreter, even if Dash isn't pre-installed on most distributions. If you're
considering Dash, you're looking for a middle-ground solution besides Lua: either your supported
platforms have Dash pre-installed, or you think it's fine to make users install Dash.

**Evaluating Dash as a language:** it's tiny — 2% of POSIX shell, 13,000 lines of code. That is 13
times smaller than Bash. It's borderline finished software. When it comes to ecosystem, it's
Unix-only. It comes pre-installed on macOS, Debian, and Ubuntu. Even if it's not pre-installed on a
lot of distributions, I'd expect these three to make up a large percentage of production
environments and developer machines, so take that into consideration. It's written in C89 again,
like Lua, and is BSD 3-Clause licensed — so this is similar to MIT licensing, very permissive.
Vendoring and compiling from source: same thing, but it's better than Lua since it's pre-installed
on some distributions.

This is the official Git repo for Dash. I say it's borderline finished software because when you
look at the tagged releases and their frequency, they're on an annual basis, and most of them are
just patch updates. So it's slow-moving software.

**How Dash fulfills the objectives:** **Simplicity:** it comes in second behind Lua. Dash is still a
POSIX shell, so it's exposed to the tricky semantics of bash, which incentivizes you to use external
commands like coreutils. **Portability:** it also comes second, behind bash — better than Lua,
because it's pre-installed on some systems. **Reproducibility:** I'd say Dash is the best out of the
three. Despite it not being finished software like Lua, scripts written in Dash are not prone to
misinterpretation by other shells or older versions of Dash.

## Solution 2: Use bash, but pretend it's Dash

One thing you can do is write your scripts using Dash before deploying them as `sh` or bash. So
let's talk about effective Dash scripting instead.

**Dash scripting guidelines.** These sources are for bash, but they're still applicable to Dash:

- ShellHarden and the Google Shell Style Guide, for safe bash scripting.
- The pure-bash-bible — this is a cookbook, a recipe cookbook, for maximizing the shell built-ins so
  you can avoid using external commands.
- And my own guidelines, which synthesize these three and add my own input, often diverging from the
  recommended practices.

Now, I'm not going to discuss these, because it's better that you just read them on your own —
you'll be referring to these style guidelines constantly while you're scripting. Instead, I'll be
talking about the conventions I've developed through experience, except for number one, which is
simply the most important best practice in shell scripting.

### Quote your parameter expansions

We have a string equation here, `1 + 2`, and we want to evaluate it with the `expr` command. In the
first invocation, we don't quote anything, and we get the answer 3 — that's correct. In the second
invocation, we quote the `addition` variable, and now `expr` regurgitates the string equation back
to us. In the third invocation, we don't quote anything, but we set this `IFS` variable to an empty
string — now `expr` regurgitates the string equation back to us.

So what's happening here? You need to be aware of two things.

Number one, parameter expansion. By default, parameter expansion will split up the value it produces
into multiple arguments based on some delimiter. The delimiter is defined by the `IFS` variable,
also known as the internal field separator. Now, you can have multiple characters in there, but by
default it's set to space.

So in our first invocation, we don't quote `addition`. Our string equation is separated by spaces,
so it gets turned into three separate arguments: the first argument is `1`, the second is the plus
sign, and the third is `2`. And `expr` likes it that way — it wants the numbers and operators to be
separate arguments.

Now, when you quote parameter expansion, you disable this word splitting, and now `expr` sees this
string as one single argument. It doesn't know how to parse this, so it just regurgitates the string
back to us. In the third invocation, we set the `IFS` variable to nothing, so even if we don't quote
this parameter expansion, word splitting does not happen — and `expr` sees this as one whole
argument again and regurgitates the string.

*My cat is calling me. She wants me to feed her. I'll be right back. Actually, you know what? I'll
show you. Come here, come here — this is her. Chill out. Look at that baby, look at that baby. Come
here. ... Wash your hands. Okay, crisis averted. Where were we? I think we were done with this
slide. Yeah.*

### Always use long flags

You're probably familiar with this command, `tar xzf`, but I'm willing to bet you don't know what it
actually means, or what these flags stand for. So I'll give you 5 seconds to guess. 5, 4, 3, 2, 1.

Okay — these flags mean: extract a gzip file. So `x`, you're extracting the tar archive; `z`, you're
unwrapping the gz/gzip compression; and `f`, you're specifying the file. `-C` (capital C) actually
means directory — it changes the working directory of tar before it starts unarchiving the tar
archive.

Now imagine this was a different CLI utility you're seeing for the first time. I prefer this second
way of invoking it instead of the first — you can intuit what it's actually doing. With the first
one, you'd have to find the documentation, read it, and figure out what these aliases are actually
doing. And a lot of CLI utilities out there have insane short flags — you'd have a long flag that
starts with the letter S, but the short flag would be a capital I or something. I don't know.
Anyway, it's a horrible experience. Leave the short flags for interactive use on the command line.
For scripts, please use long flags.

And I'm not alone here. This is matklad — he's the creator of rust-analyzer, the Rust LSP. He's now
working at TigerBeetle, and he recently wrote a blog post on using long options in scripts. I highly
recommend reading his blog — it's a high-quality source of information, and I've learned a lot from
him. You'll be exposed to TigerBeetle's best practices and how they develop their fault-tolerant
software. But I digress.

Also, Ginger Bill again — he does not like short flags. So in the Odin compiler, you'll only have
long flags. You see this? There's only one dash. Yes, everything is a long flag — there are no short
flags. Oh yeah, and he also used this as an example: `xvz`. Okay.

### Declare local variables (even though `local` is non-POSIX)

In POSIX shell, you can only do global variable declarations. But if you want to throw in a little
bit of non-POSIX behavior, use the `local` keyword — it's going to save you from a bunch of
headaches and a lot of time. And a lot of shells actually support the `local` keyword — a majority
of the shells that `sh` symlinks to. So bash, zsh, dash, Korn shell, ash: all of those shells
support the `local` keyword.

### Use `test`, avoid brackets

If you didn't know, brackets are not special shell syntax (that's a tongue twister). They're an
alias to the command `test`. The only difference is that they expect their last argument to be a
closing bracket `]`.

So if these are the same command, you can guess what the output of the second command would be.
Spoiler alert: it's not the same. I'll give you 5 seconds. 5, 4, 3, 2, 1. It's a syntax error. We
made a typo — we didn't separate the last bracket with a space. Remember, it has to be a separate
argument. I'll fix the typo in 3, 2, 1. I hope you saw the difference — now it outputs the correct
string.

These helper commands convert your scripts from using the bracket command to the `test` command. The
first one is for Neovim, and the second one uses `sed`, which modifies your file in place — so just
be aware of that.

### Disable errexit

There are a lot of scenarios where errexit will not exit the script. If you don't know what errexit
does: when you have a standalone statement exit with a non-successful status code, it exits the
script right there with a non-successful status code as well. You see here, even if I explicitly
`exit 0` at the end, the script fails with a status code of 1. In the second snippet, we disable
errexit — we don't need to do this, it's disabled by default. Even if `test` fails, it still exits
with a status code of 0.

Now, the sources I mentioned earlier recommend that you enable errexit. In theory it's a good idea,
right? When a command fails, you want the script to stop executing. But since there are so many
scenarios in which that doesn't actually happen, it just creates an inconsistent mental model when
you're scripting. Scenarios like conditionals, pipelines — well, not pipelines, piping — capturing
the status code of the previous command, subshells: in all of these, errexit will not exit the
script.

And from a language-design standpoint, yeah, I get it — it makes sense. You're capturing the status
code in conditionals and pipes, you want to use that, so errexit will not take effect. Also with
capturing the status code of the previous command with `$?`, and in subshells (they have their own
environment, they cannot affect the parent shell) — yeah, that makes sense. But you see how thinking
about all these edge cases where errexit won't fire — for what benefit, right? Just so you don't
have to explicitly exit the script if a command fails? I'd rather you just do explicit error
handling and forget about errexit in the first place.

And what would that look like? Right here, you do some short-circuiting for a command — you exit
explicitly, or you do some guard checking like in Go.

### SCREAMING_SNAKE_CASE for environment variables only

Use `snake_case` unless it's sensitive data or an environment variable that affects an external
command.

This is a GitHub Actions YAML snippet — real production code that I wrote. The first block is where
you declare all your environment variables. The second block is where the bash script goes, to be
executed for that job. You might be asking, "James, if all of these are environment variables, why
would you make `title` lowercase?"

If you're not familiar with GitHub Actions: if you have a variable coming from the GitHub
preprocessor, best practice is to declare it as an environment variable for security reasons. Let's
say someone puts a SQL query in their pull request title — this protects you from SQL injection
attacks, because I think GitHub will escape it. I think. I don't know. All I know is it's best
practice.

Now, do we make this SCREAMING_SNAKE_CASE? Well — is it sensitive data? It's not. Does it affect an
external command? We just use it in a case statement. So for all intents and purposes, this is just
a normal variable that we declared as an environment variable for security reasons.

What about the random API key? Well, it's sensitive data, so we make it SCREAMING_SNAKE_CASE.
`AWS_DEFAULT_REGION` — this is for the AWS CLI, so we don't have a choice here; that's
SCREAMING_SNAKE_CASE.

## Solution 3: Use bash in all its glory (just like Google)

I can't help here, because I really try to avoid bashisms.

## Takeaway

After careful evaluation of Bash, Lua, and Dash with regard to how they contribute to simplicity,
portability, and reproducibility for effective shell scripting, the takeaway is: use Lua, because I
said so.

The end.
