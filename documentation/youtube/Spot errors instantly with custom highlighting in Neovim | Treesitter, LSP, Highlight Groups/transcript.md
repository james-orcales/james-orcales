# Custom Syntax Highlighting in Neovim with Tree-sitter

## Setting the goal

I'm using Gruvbox in Neovim. All keywords here are colored red, and I want to change the `defer`
color to blue so I can see it more easily.

## Creating the query file

In my Neovim config, inside the `after` directory, let's create two new directories. First is
`queries`, and inside that is the target language — which in my case is `zig`. Then I'll create a
file named `highlights.scm`.

At the top of the `.scm` file, we need to add this comment: `; extends`. This tells Neovim to extend
the existing Zig tree-sitter syntax instead of overwriting it.

Next, copy everything as is. In the first string, put whatever you want to have custom syntax
highlighting for. For the `@` symbol, that will be the capture group for your custom syntax
highlighting — you can put anything here. I chose to go with the tree-sitter convention, which is
`keyword.something`. Then you just copy this `#set!` directive at the end with a priority of 200.

In `init.lua`, set the highlighting for the capture group that you used. I've set up some custom
highlight groups, and I'll use my custom blue for my `defer` keywords. Now `defer` should be updated
to blue.

## Highlighting error keywords

Next, I want to change the highlighting for all keywords related to errors. Inspecting the `try`
keyword, we see that tree-sitter already recognizes this as a distinct group of keywords called
exceptions. And if we look into the Neovim highlights under the help tags, we'll see that there is a
specific highlight group for exceptions, and it's Gruvbox red.

Back in `init.lua`, I'll change the highlighting of exceptions to my custom violet highlight group.
But if I go back to my Zig project, we'll see that the `try`s are still not updated to violet.

Inspecting `try` again, we do see custom violet get applied to tree-sitter's `keyword.exception`.
But LSP semantic tokens take priority over tree-sitter nodes, and we can see that LSP does not
distinguish `try` from other keywords as an exception. So the general highlighting for keywords —
Gruvbox red — gets applied instead.

Let's bump the priority of `keyword.exception` in `highlights.scm`. Now the `try`s are violet,
making it so much easier to see the error density of a scope.

## The `catch` keyword

I made the `catch` keyword violet as well. Inspecting `catch`, we still see the LSP highlighting
there, but the custom violet tree-sitter highlighting now takes priority.

## Scoping the bang (`!`) to function declarations

Lastly, I want to make the bang (`!`) in the function return type violet as well. One problem with
the way we do things right now is that tree-sitter will treat all bangs in the file as a
`keyword.exception` — and bangs are also used for negating expressions. So in `!true`, the bang gets
turned violet as well.

We need to scope bangs as `keyword.exception` to function declarations only. We can do that by
looking at the abstract syntax tree built by tree-sitter, through the `:InspectTree` command.
Clicking on the bang, we see that it's part of the function declaration node. All we need to do is
specify the function declaration node before our bang highlight query inside the `.scm` file. Now,
bangs only get turned violet if they're part of a function declaration.

## How I figured this out

Here's how I figured this out on my own with `fzf-lua` (you can use Telescope as well). I searched
for "priority" in the help tags. I then saw the tree-sitter highlight priority. Reading this, all I
    had to figure out was where to place this snippet and how to specify any keyword for it.

Scrolling up to the start of the syntax highlighting section, we see that I need to place this in
`highlights`, and I need to match the nodes to the tree — which we view with the `:InspectTree`
command. Reading further, we see how to match any string throughout the whole file.

All that's left to figure out is where to place this `.scm` file, and we see that Neovim looks for
it under the runtime path. The runtime path includes the `after` directory in our Neovim config, and
it looks for the queries under a subdirectory called `queries`.

Now, to tie it all up: I figured out that I needed to add a subdirectory for the target language
that I wanted to add syntax highlighting for, by reading the tree-sitter contribution docs and
searching under the issues and discussions tab. And this is the post that helped me out the most.

All right — I hope this helps!
