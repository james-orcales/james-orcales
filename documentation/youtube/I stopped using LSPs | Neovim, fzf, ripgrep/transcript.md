# I Stopped Using LSPs

Giannis is about to get traded, so I'm wearing my Bucks hat. Yeah, riperoni. Anyway, let's start. Oh
— the slideshow.

> The right tool for the job is often the tool you are already using. Adding new tools has a higher
> cost than many people appreciate.
>
> — John Carmack

In the spirit of this philosophy, I stopped using LSPs. The goal is to reduce dependencies in my
developer setup and use what I already have: Neovim, fuzzy finder, and ripgrep — for two main
reasons: security and portability.

## Why: security and portability

As a professional developer, my developer machine is an extension of the organization's attack
surface. Consequently, developer machines are one of the weak links in an organization's security.
So reducing dependencies means fewer supply-chain attacks, and that improves the security of
everyone involved.

Regarding portability: minimizing the tools I depend on for my workflow means I can switch between
machines and get the same experience, the same efficiency and productivity, without much issue. That
means I can use Darwin, Linux, and even Windows on any given day, and I'd be set up quite quickly.
Right now I only have like two dependencies that are Unix-specific — Ghostty and the fish shell —
which are trivial to replace.

## How Antithesis secures developer machines

Antithesis tackles the security of developer machines in two ways: physical security and software
security.

**Physical security:** they have an on-site arrangement. Developer machines are within the office,
and they can standardize the security of the physical access to machines.

**Software security:** NixOS is installed on all machines. This ensures that system setup is
standardized, because configuration is treated as infrastructure as code. Two benefits: one,
security auditing is straightforward because everything is in one configuration file. Two, every
time you stray from the defaults — you change a setting, you add another dependency — those things
are done explicitly in the configuration file, and that adds a maintenance burden on you, which
incentivizes you to simplify your setup.

## What LSPs actually do for us

Replacing LSPs requires us to take into account these use cases: syntax highlighting, autocomplete
and snippets, API discovery, refactoring, and inline error messages.

## The technical problems with LSPs

Beyond reducing dependencies, another reason I encourage you to migrate away from LSPs is that they
have inherent technical issues. Number one: they're sluggish. Number two: they're too generic.
Number three: they're far too complex for what they actually do. (Well, that's not a technical issue
— that's more of a conclusion.)

**Sluggish.** LSPs are a full-on HTTP server communicating via JSON-RPC. For something that's
constantly running and communicating with your editor, something that's always updating as you make
changes to your codebase, this is quite heavy-handed. And you see how, in large codebases — monolith
codebases — you'd already have performance issues with LSP servers.

**Too generic.** When you're using a language with niche features, LSP just straight up doesn't work
with those features. For example, with Zig, if a function signature has a comptime parameter, LSPs
will straight up not work — it wouldn't have any autocomplete inside that function. Another one is
in Odin: if a function has parametric polymorphism, the LSP will just not work. And when you're used
to having autocomplete and auto-suggestions, and suddenly your LSP isn't working in this function
scope, you get distracted. It takes you away from the zen of coding, because now you're thinking,
"Oh wait, is there an error in my code? Is my LSP bugging out? Did it just crash?" Now you're
thinking about all of these things that are irrelevant to what you're actually solving.

**Too complex for what they do.** Given that LSPs are just as complex a project as compilers, when
you look at these use cases, there's really not that much benefit — the benefits don't outweigh the
costs.

## Syntax highlighting

Syntax highlighting helps us parse the code structure at a quick glance. It also gives us immediate
error feedback: when you're typing and there's an error in your grammar, the syntax highlighting for
the whole file just gets messed up — that's what I mean by error feedback. And LSPs build a more
accurate AST relative to regex and tree-sitter, improving highlighting accuracy.

### The Rob Pike debate

This comment from a Y Combinator thread discussing Rob Pike's opinions on syntax highlighting
articulates my sentiment more accurately. I have to agree:

> Syntax highlighting lets me recognize patterns in code without having to read it, and I actively
> avoid reading details I don't need. To me, that is also what annoyed me about IDEs. I want to
> focus on the shape of the code, not other stuff. When I focus on code, I want that code to be all
> that exists in my mind at that time. But I suspect whether or not you like IDEs is orthogonal to
> how you like your code presented. I also feel like there is an interesting range within developers
> — from those of us who pattern-match, skim, and zoom in on details, to those who read code in
> detail — and I'm not sure those of us on different sides of the spectrum see perceptual load the
> same way.

In other words, he likes to take note of what the code looks like, and that's how he remembers the
content of the codebase. I can relate to this, because back in high school, when I was taking an
exam and trying to remember the lecture, I'd try to imagine my notes — the notes I wrote and how
they looked in my notebook. Was it on the left page? The right page? What did my handwriting look
like? As I recall those details, I then recall what the content was. That's how I can see the merit
in what this person is saying.

On the other side of the fence, we have someone asking for syntax highlighting to be added to the Go
Playground — and take note, this is back in 2012. This thread is infamous for Rob Pike's abrasive
responses. I only included the first one, which is quite tame:

> Gofmt was written to reduce the number of pointless discussions about code formatting. It
> succeeded admirably. I was sad to say, however, that it had no effect whatsoever on the number of
> pointless discussions about syntax highlighting.

If you look at the Go Playground today, it still doesn't have syntax highlighting. Everything is
white — comments, strings, function calls. And you know what? I actually agree with Rob Pike's
decision here, because the playground is meant to show you small snippets of Go source code, and you
really don't need syntax highlighting to understand any of this. Adding syntax highlighting implies
that now we have to add a Go parser written in JavaScript, which is a huge dependency for this
website, and it completely inflates the complexity. Rob Pike, however, probably just doesn't like
syntax highlighting inherently — and on that, I disagree. I'm coming more from an engineering
standpoint: reducing dependencies.

These two examples showcase the spectrum of syntax highlighting. On one side you have Rob Pike: zero
syntax highlighting at all. On the other, you have rainbow vomit: everything is colored, and the
highlighting gives you no sense of structure anymore. And as with all things in life, the
middle-ground approach is always the best one — and I'm here to show you that I think I found it.

### My minimal highlighting setup

What I did is I just have four highlight groups. Technically — no, practically — three. I have
white, gray, yellow, and red. White is source code, yellow is strings, gray is for comments, and
to-do markers (TODO, FIXME, NOTE) inside comments are red. Punctuation pairs are red. That's it.

```lua
local palette = {
	main = {
		["primary"] = "#DFE0DC",
		["primary_dark"] = "#888888",
		["accent"] = "#F6C177",
		["red"] = "#EB6F92",
		["blue"] = "#9CCFD8",
	},
}

palette = palette.main

vim.cmd.colorscheme("quiet")

vim.api.nvim_set_hl(0, "Normal",      { fg = palette.primary      })

vim.api.nvim_set_hl(0, "Comment",            { fg = palette.primary_dark })
vim.api.nvim_set_hl(0, "@comment",           { link = "Comment"          })

vim.api.nvim_set_hl(0, "String",      { fg = palette.accent       })
vim.api.nvim_set_hl(0, "@string",     { link = "String"           })
vim.api.nvim_set_hl(0, "TODO",        { fg = palette.red          })
vim.api.nvim_set_hl(0, "MatchParen",  { fg = palette.red          })
```

Using this for a while, I can confirm that knowing code structure at a quick glance is useful.
However, it does not take much to do this, and it's easy to overdo. That's why only large swaths of
text need highlighting: strings and comments. Also, immediate error feedback is distracting: the
more colors you have, the more colors can get messed up. As you're writing, suddenly your function
calls become green instead of red, and now you're all confused.

### Error feedback in practice

I have a real example here in one of my projects. Let's go to Ghostty, open the Odin project, and
one of my tests — `test.odin`. There we go. In this file I'm doing snapshot tests. I have a function
that returns an object, then I serialize that to a string, and that's what I assert in my snapshot.
So everything is fine — the strings are yellow, still yellow here. And when you get to this point
right here, for this snapshot, everything starts failing. You can see it's changing right in front
of me now: the source code is yellow and the actual strings are white.

During development, I'm not bothered by this, because I have a minimal highlighting setup. So I can
easily adjust to my source code being yellow and my strings being white instead. Now imagine if I
had a bunch of different colors here, and now I'm getting disoriented because my code isn't the way
I expect it to be. That's just one of the productivity gains — and reduction of cognitive load — I
got from using this minimal syntax highlighting.

Going back to Firefox: the answer to less accurate highlighting caused by removing the LSP is to
just use a minimal highlighting setup. It reduces your cognitive overload, and you can focus more on
writing code instead.

## Autocomplete and snippets

Autocomplete and snippets help us write more code and discover APIs. But since discovering APIs is
the next section, we'll focus on writing more code.

### What LSPs offer

LSP communicates with your editor to show autocomplete suggestions on the fly. I'll demonstrate, for
completeness's sake. Let's go back to Ghostty, open up a Golang project. I downloaded my old Neovim
configuration with the LSP setup. Let's go to my logger package, and start with autocomplete. So if
I type `os.` and then press Ctrl-N, the exported functions and variables for the package will show
up, and I can scroll through them. Let's say I want to open a file — `open`, there you go — and I
can type `fd =`, and on the file descriptor I can do `.`, and I can now use the methods for files. I
can change directory on it. You get the idea.

There are also code actions. Let's say I wanted to initialize all fields of this type explicitly: I
press `ca<space>`, and all the available code actions show up. One of them is "fill logger" — there
you can see it — it will initialize all fields.

There are also snippets: if I type `if`, it'll automatically complete the boilerplate code for me
and place me inside insert mode. I don't have it in this setup, but you get the idea. This is the
power of autocomplete and snippets: it allows you to write more code, and write code faster.

### The other side: Ginger Bill

On the other side of the fence, we have Ginger Bill, the creator of the Odin programming language.
He's been programming for 20 years now, at 30 years old. He's never used LSPs, but he has tried IDEs
that have autocomplete, and he found that using these IDEs was a hindrance to his productivity — and
when he stopped using them, he got more productive. Moreover, he recommends to people learning new
languages that they don't use a dedicated IDE like JetBrains or LSPs, and instead learn the actual
language — the grammar and the idioms — because when you're relying on the autocomplete magic, you
don't internalize these things.

### My approach: personalized keymaps

So in my current setup now, the way I automate boilerplate code is I have keymaps that insert text
and immediately place me in insert mode. Let's say I wanted to do `if err != nil`: I press `en` in
normal mode, and there you go, I have the guard check. And let's say I use a function that only
returns an error — let's say changing directory. I can press Shift-`en`, and it will inline the
error declaration. The nice thing is that now I'm creating snippets personalized for my workflow.
It's based on my empirical evidence, my personal experience of what kind of code I usually find
myself writing.

```lua
vim.api.nvim_create_autocmd({ "FileType" }, {
	group = vim.api.nvim_create_augroup("language_specific_macros", { clear = true }),
	callback = function(ev)
		if ev.match == "go" then
			vim.keymap.set("n", "en", "oif err != nil {<CR>}<ESC>O", { noremap = true, silent = true, buffer = ev.buf })
			vim.keymap.set("n", "EN", "Iif <ESC>mzaerr := <ESC>A; err != nil {<CR>}<ESC>`z", { noremap = true, silent = true, buffer = ev.buf })
		end
	end,
})
```

I'll show you my old setup. I also have keymaps that insert log calls using the `slog` package,
because I used `slog` a lot back then. So if I was in insert mode and I pressed `L`, it'd insert an
error log; if I do `Ll`, it's an info log. And you'll notice that in the error log, I automatically
add the error key and the error value.

Embracing this setup, I found myself writing more code and knowing the language, grammar, and idioms
by heart. In sum, autocomplete and snippets are useful for boilerplate code — I can confirm that —
but you can make a lot of useful snippets without using LSPs. You just need to learn how to
personalize your editor.

### AI and typing speed

On another note: AI will replace snippets and code actions. Since they're part of any contemporary
developer setup, prefer them instead. You can think of AI autocomplete as a much more powerful
version of autocomplete and snippets from language servers — and that makes that use case of LSPs
redundant, making it much more attractive to migrate away from this technology.

Lastly, typing speed has never been the bottleneck. At 100-plus words per minute — I've been a
152-words-per-minute typist on QWERTY, and I found myself regressing down to 100–110 once I was
coding for a long time without doing any typing tests in between — just because thinking about what
to write takes more of my time than actually writing code. And writing all these symbols
significantly slows down your typing speed if you're not practiced on that.

Before we move on: drink water, stay hydrated. I'll get my sip in.

## API discovery

Moving on to API discovery, which is the extension of autocomplete. I'll show you how I use
autocomplete for API discovery.

### My old approach

Let's go back to my old setup. What I did beforehand — for the longest time, about two years — is
that when I was exploring packages, when I learned all of these new languages, I would do this
Ctrl-N, and this is how I would explore all of the exported functions, completely basing it off the
function signature and what doc comments were available. Which is completely ass-backwards when
you're trying to learn a language, because when you're learning the standard library, you should be
looking at the implementation details.

When I did want to read the implementation details, I could go to definition, and for the most part
this was sufficient — but it didn't really get me in the flow of just reading the implementation of
the whole package and the different related files. You can see here that for `hostname`, it has
literally one dedicated file for it, so it doesn't incentivize me to read the surrounding context.

I also use this code action which shows all of the implementations of an interface. For example, in
the `io` package, if you look at the `Writer` interface — if I do `ca<space>` here — there's a code
action that shows all the implementations of the `Writer` interface in the standard library. I don't
know why it's not showing up here, but it was an essential part of how I learned the Go standard
library, and that was sufficient for me to learn different languages: Go, Rust, Zig.

### The other side: Mitchell Hashimoto

But let's look at the other side of the fence again, with Mitchell Hashimoto, in this interview on
his worst practices. (Turn on captions — put it that way.)

> **Interviewer:** So tell us, what is your worst practice?
>
> **Mitchell Hashimoto:** I think my worst practice is a little bit more modern. I don't think this
> would have been a worst practice maybe 10 years ago, but it raises eyebrows now when I tell
> people: I use Vim, but I don't use any sort of code intelligence, language server, autocomplete,
> anything. The only thing I use is auto-format on save. And it took me until two years ago to get
> there. I like my editors to do nothing, so that kind of freaks people out. I've gotten a lot of
> people saying, "Oh, you're working so much slower than you could be," or "How can you possibly
> work that way," but that's just how it works.
>
> **Interviewer:** If you have a library that you're trying to use, how do you look up another
> function in the API? How do you find what you're looking for?
>
> **Mitchell Hashimoto:** My learning style is reading the reference manual, at least skimming it
> ahead of time. Maybe that's because I started programming 20, 30 years ago, but it's that. So in
> my mind I sort of have a rough idea — I may not remember the exact arguments, the exact
> capitalization or wording, but I'm like, "Oh yeah, there's a function like `connect_whatever`."
> When I was young, I had a 2-hour-a-week computer time limit, up until I was 18. And the two hours
> — also, if I had to do homework, like type up a paper in Microsoft Word, it counted. So I would
> end up doing a lot of programming by either doing it on paper or doing it in my head, to the point
> where I was like, "Okay, when I get to the computer, I have to just get it out and run it. It has
> to work, because I don't have time to go back and forth."

Shout out to Worst Practices for that interview. Mitchell Hashimoto's learning style is quite
interesting: he first learns the API — the reference manual — and has a general idea of what's
available to him when he's writing code. And this arose from the restrictions placed on him when he
was younger.

I relate to this learning style. When I learn a new technology, I read the reference manual first
and foremost. Let's say NixOS — I first read the official wiki before watching YouTube videos and
reading forums to get a better understanding of the higher-level concepts and how to use it
practically. And then I go back to the reference manual a second, third, fourth time to really
understand how to apply the specific features available.

Again, my old approach and Mitchell Hashimoto's way of learning showcase the spectrum of how to
discover APIs — and, as you can guess, the best approach is always the middle-ground approach. I
think I found that with my new way of exploring libraries in my current setup.

### My new approach: fuzzy finder + ripgrep

Let's change directory into the Go standard library first. Now I take advantage of fuzzy finder and
ripgrep, using regex for everything. Let's say I wanted to explore the `io` package. I do
`s<space>`, which brings up these available regexes that I preconfigured. Let's say I want to see
all of the available functions in the `io` package. Let's remove the preview for now. And here I get
all of the exported functions and methods — you can see at the top the specific regex I used to
filter the text, and then I filtered out the test files.

```lua
local rules = {
	Golang = {
		Function = [[^func +(?:\([a-zA-Z0-9_]+ +\*?[a-zA-Z0-9_]+(?:\[.+\])?\))? *[A-Z][a-zA-Z0-9_]* -- !*test* !*vendor*]],
		Type = [[^type +[A-Z][a-zA-Z0-9_]+ -- !*test* ]],
	},
	Odin = {
		Function = [[^[a-zA-Z0-9_]+ +:: +proc -- !*test* ]],
		Type = [[^\w+ +:: +(?:struct|union|enum|distinct) -- !*test* ]],
	},
	Lua = {
		Function = [[(?:function [a-zA-Z0-9_]+\(|[a-zA-Z0-9_]+ = function\(|= def\()]],
	},
	Rust = {
		-- We don't filter by file extension because Rust API searches often target
		-- individual files, unlike Go or Odin, where the package system makes it
		-- more common to search the entire directory.
		Function_and_Macro = [[(^\s*pub (const )?(unsafe )?fn +[a-zA-Z0-9_#]+|^\s*macro_rules! [a-zA-Z0-9_#]+|^impl )]],
		Type = [[^\s*pub (?:struct|union|enum|trait|type) [a-zA-Z0-9_#]+]],
	},
}
```

The advantage of this over my old approach is that now I can fuzzy-find things I didn't even know
existed. So for example: hey, is there a function that tells me if something is a directory? `IsDir`
— there it is. Most of them are methods. So now I'm thinking, how do I get a `FileMode` type or
object? I search for `FileMode` — there you go, and I get all of the methods available. Same idea
with types: I can look for the exported types here, and let's bring back the preview — I can read
the implementation detail. This addresses another limitation of my old approach: that I'm not easily
exposed to the implementation details of the library. Whereas here, as I'm scrolling through the
entries, I'm already reading the preview of these different types.

For flexibility, I also have the `any` option, which just defaults to an empty regex expression, and
I can search for whatever I want. Let's say I'm interested in whatever `Writer` method there is —
here you go, and now all of the `Writer` methods in `io` are available.

Currently I'm also learning Rust, which is a great way to stress-test this new approach, because now
I'm learning a completely new standard library and language. Let's go to the Rust standard library.
One of the first things I tried learning was how to spawn an external process, so I assumed it was
in the standard library here, and I started searching for a spawn function. There's no spawn
function. I tried `execute` instead. I tried `exec`. And then there's this thing, `output` — and it
takes in a `Command` type. That was pretty much on the money. You can see the power of the fuzzy
finder here, where I wrote a completely unrelated expression — none of these entries have the
substring `exec`, but it showed me exactly what I needed, which is this `output` function instead.

So now I'm starting to look for methods on this `Command` struct. I'll mention that when I'm inside
a file and I start searching for things, all of the entries are scoped to that file. Whereas if I'm
in a directory view and I do a search again, now it's showing all of the available functions for
that directory and all of its subdirectories recursively. In Golang, when I'm inside a Golang
project, everything is always package-scoped — so even when I'm inside a file buffer and I do a
search, it will always show all of the available functions in the whole package, not just the file
itself. I do it this way because in Rust, individual files can be standalone modules.

Going back to what we were doing — looking for methods on the `Command` type — if we start searching
for it again and remove the preview, you can see here all of the methods on the `Command` type. I
    can assume these are the methods on the `Command` type because the way I have it set up, all of
    the entries are sequential. So if we have an `impl Command` before all of these methods, and
    these are indented, then you can assume these are for the `impl Command` block specifically.

```lua
local fzf = require("fzf-lua")

local parse_programming_language = function(path)
	if path:match("%.go$") or path == "go.mod" then
		return "Golang"
	elseif path:match("%.odin$") then
		return "Odin"
	elseif path:match("%.lua$") then
		return "Lua"
	elseif path:match("%.rs$") or path:lower() == "cargo.toml" then
		return "Rust"
	end
	return nil
end

local module_api_search = function()
	local path = vim.api.nvim_buf_get_name(0)
	local operation = fzf.grep

	local programming_language = nil
	if not path:match("^oil://.*") then
		programming_language = parse_programming_language(path)
	else
		local handle = vim.uv.fs_scandir(vim.uv.cwd())
		if handle then
			while true do
				local name, t = vim.uv.fs_scandir_next(handle)
				if not name then
					break
				end
				if t == "file" then
					programming_language = parse_programming_language(name)
					if programming_language then
						break
					end
				end
			end
		end
	end
	if programming_language == nil then
		fzf.live_grep()
		return
	else
		if not path:match("^oil://.*") and (programming_language == "Rust" or programming_language == "Lua") then
			operation = fzf.grep_curbuf
		end
		local items = {}
		for item in pairs(rules[programming_language]) do
			table.insert(items, item)
		end
		table.sort(items)
		table.insert(items, 1, "Any")
		fzf.fzf_exec(items, {
			prompt = string.format("Search Package (%s) > ", programming_language),
			actions = {
				["default"] = function(selected, opts)
					if selected == nil then
						return
					end
					selected = selected[1]
					if selected == "Any" then
						fzf.live_grep()
					else
						operation({
							search = rules[programming_language][selected],
							no_esc = true,
							silent = true,
						})
					end
				end,
			},
		})
	end
end

vim.keymap.set("n", "s<Space>", module_api_search)
```

### Insights

I think the power of this new approach is quite evident, and I've been using it for a while now. My
insights: API discovery through autocomplete is a mediocre developer experience. Using fuzzy finder
and ripgrep enables finding APIs you didn't even know existed. Opening library source files is
streamlined, which leads to reading implementation details more often. And as Ginger Bill and
Mitchell Hashimoto testify, doing this actually helps you remember the APIs and internalize their
idioms.

## Refactoring

Refactoring is one of the biggest selling points for LSPs and dedicated IDEs — especially for
object-oriented programming languages like C++ and Java, where there's a lot of boilerplate. For
Golang specifically, the LSP handles renaming variables and code actions that shift the order of
parameters around. I'll showcase it here.

### What LSPs offer

Let's go back to my project, `go-next-log`. Let's say I want `dst` to be the second parameter
instead. If I do a code action, there's this "move parameter" code action — and you can see all of
the changes it made on the different call sites. It switched the order of the arguments themselves
as well, which is very useful. There's also renaming: for `source`, if I do `gr` and name it `arst`,
everything is renamed. I can also do it here — `grn` — and it's updated on every usage.

These refactoring capabilities of LSPs are a huge quality-of-life improvement, specifically because
the LSP updates all code affected during refactoring, like renaming variables and reordering
function parameters.

### My approach: regex and greppable names

In my current setup, I have no alternative for these features. I only have one for renaming variable
names, which is dependent on regex expressions. I'll show you the workflow. If I wanted to rename
the `dst` variable here in this function, I'd highlight this whole function scope and then press
`sr`, which will do a find-and-replace on `dst`. And you can see all of the `dst` variables are now
changed. I press enter, and there you go. It's simple, but much more capable than people expect.

```lua
-- Global case-INsensitive search and replace, matching the word under cursor, without confirmation
xplat_set("n", "sr", [[:%s/<C-r><C-w>/<C-r><C-w>/gI<Left><Left><Left>]])
-- Visual mode case-sensitive search and replace
xplat_set("v", "sr", [[:s///g<Left><Left><Left>]])
```

The experience can be made nicer by just keeping in mind to use unique variable names that are
easily greppable. This not only improves the renaming experience, but also the grepping of this
variable throughout the whole file. For example, I have a convention that for all logger variables
and identifiers, I always use the identifier `LGR`, which makes it easy to look for all the logger
instances in the file. There you go — there are more than 100 logger usages in here. Imagine if I
instead made this the lowercase `logger` with the full word spelled out. Now when I'm searching for
`logger`, it will clash with the actual `Logger` type — here I'm matching on the capitalized
`Logger` struct also. I can do a case-sensitive search to specifically look for logger variables,
but it's just more involved than having a straight-up unique variable name like this. You
understand?

So again, I have a global search-find-and-replace here with `sr` in normal mode, and this will
replace all loggers in the file. I can also do this in visual mode, which scopes the
find-and-replace to that selection — and then the other loggers are still `LGR`.

### The other side: Karl Zylinski

On the other side of the fence, we have Karl Zylinski, who is a game developer and wrote the first
book on the Odin programming language. This is from the official Odin Discord channel. He says:

> Yeah, go-to-definition doesn't cut it when you're not 100% sure what you're looking for. Not
> having LSP makes me less likely to split things into separate files, because I can't get good
> completion in Sublime for things that are not in the current file. With LSP, you do get that, and
> finding stuff in other files of your project becomes easier. It directly affects my project
> structure.

To generalize: what he's saying is that when you have LSPs and dedicated IDEs, it reduces the
friction in refactoring the codebase. It also reduces the friction in creating a lot of indirection
in your project structure by splitting everything into multiple files — which you just saw in the Go
standard library earlier, which has a single function in the whole file (the `hostname` function for
the `os` package).

### Insights

My insights on my current setup: long and unique variable names are incentivized, improving
readability and especially greppability of the code. It also adds more friction in splitting code
into multiple files — adding indirection in your project structure — making it simpler. There's also
more friction in refactoring, which clarifies when to hack and when to polish.

## Inline error messages

I'll showcase this, but you've already seen it earlier — this is just about having this in your
editor directly. My alternative here is to just use a quickfix list combined with the `makeprg`
feature of Neovim. Inside a Go project, I first set the `makeprg` variable — this is going to get
run whenever you do the `:make` command. So here I set it to `go build scratch.go`. Now let's add an
error to this file, and I do `:make` — let's save it first — and if I do `:make`, there you go, I
get all of these syntax errors.

```lua
vim.api.nvim_create_autocmd({ "FileType" }, {
	desc = "Set errorformat",
	group = vim.api.nvim_create_augroup("set_errorformat", { clear = true }),
	callback = function(ev)
		if ev.match == "odin" then
			vim.opt_local.errorformat = "%f(%l:%v) %m,%-G%.%#"
		elseif ev.match == "go" then
			vim.opt_local.errorformat = "%f:%l:%c: %m"
		elseif ev.match == "rust" then
			vim.opt_local.makeprg = "cargo check"
			vim.opt_local.errorformat = "%-Gerror: could not compile %.%#," --ignore
				.. "%-Gwarning: %.%# generated %.%#," --ignore
				.. "%Eerror[E%n]: %m,%Eerror: %m,%Wwarning: %m,%Inote: %m,"
				.. "%C %#--> %f:%l:%c,%-G%.%#"
		end
	end,
})
```

Now, I've automated this so that when I'm saving the file through my keymap, it adds all of those
errors to my quickfix list. So when I do Ctrl-E, there you go, all of the errors are down below, and
I can sift through them, moving my cursor to the error location directly. There's a bug here where,
since I'm using space indentation, it's not actually going to the exact column — but that's fine.

```lua
vim.keymap.set({ "v", "n", "i" }, "<C-S-E>", function()
	vim.cmd("w")
	local path = vim.api.nvim_buf_get_name(0)
	if path:match("%.go$") or path:match("%.rs") or path:match("%.odin") then
		local global_mp = vim.opt.makeprg:get() or ""
		local local_mp = vim.api.nvim_get_option_value("makeprg", { scope = "local" }) or ""
		if global_mp == "" and local_mp == "" then
			local new_makeprg = vim.fn.input("makeprg is unset. Enter command: ")
			if new_makeprg ~= "" then
				vim.api.nvim_set_option_value("makeprg", new_makeprg, { scope = "global" })
			end
		end
		vim.cmd("silent mak!")
		if #vim.fn.getqflist() > 0 then
			vim.cmd("cope")
			vim.cmd("wincmd w")
		else
			vim.cmd("cclo")
		end
	end
	vim.api.nvim_feedkeys(vim.api.nvim_replace_termcodes("<ESC>", true, false, true), "n", true)
end, { desc = "Save file then check" })

-- Step through the quickfix list, wrapping at both ends
xplat_set("n", "{", function()
	local l = vim.fn.getqflist({ idx = 0 })
	if l.idx == 1 then
		vim.cmd("silent! clast")
	else
		vim.cmd("silent! cprev")
	end
end)

xplat_set("n", "}", function()
	local idx = vim.fn.getqflist({ idx = 0 }).idx
	local len = #vim.fn.getqflist()
	if idx == len then
		vim.cmd("silent! cfirst")
	else
		vim.cmd("silent! cnext")
	end
end)
```

And that's about it. If I needed more information on the errors, I can go to the command line and
specifically build there. In other languages like Odin and Rust, you'd have more information on what
exactly is wrong with the error — but inside the editor, you simply need a good summary, and you
usually know what's wrong.

After using this for a while, I figured it's essentially the same experience, which is a good thing.
I have a slight preference for the new approach, however, since the errors are now scoped to the
lower portion of my screen instead of showing up inline where my cursor is — which is kind of
distracting when I know there are errors in my file but I don't want to acknowledge them at first.
For example, here there are a lot of errors, but it can be simply fixed with this right here — and
now all the errors are gone. Sometimes I just want to ignore that and go on with my day.

## Conclusion

In conclusion, LSP servers are greatly complex for what they bring to the table. Workflows arising
from LSPs are inferior to what's possible when embracing the versatility of a lean toolchain:
personal development environment + fuzzy finder + ripgrep. Removing LSPs not only reduces the
surface area of our development environment, but also amplifies our growth as developers.

The end.
