# setup

`setup` is a one-command machine bootstrap. It takes a freshly cloned copy of this
repository and turns a bare machine into the development environment I actually work in
— shell, editor, fonts, search tools, version control, terminal. The repository is the
source of truth; `setup`'s whole job is to make the machine point at it.

> **The entire design assumes this repository lives at `~/code/james-orcales`.**
> `setup` resolves your home directory and joins `code/james-orcales` onto it, and the
> dotfiles hardcode that same path when they set `XDG_*` and `PATH`. This is deliberate:
> one known location means there is nothing to configure and nothing to discover. The
> paths are identical on every machine, so the environment is reproducible without a
> single per-host setting. Flexibility here would buy nothing but a discovery step that
> can go wrong.

## Principles

**Builds are hermetic.** Every tool is built from sources committed under
`third_party/`, offline — `cargo --offline --locked`, `go build -mod=vendor`, and even
the Go compiler itself is built from a vendored, checksum-verified tarball rather than
downloaded. Cloning the repository gives you the build inputs; the builds don't then go
fetch a mountain of transitive dependencies over the network. Network access is reserved
for a few pinned things — notably the Rust toolchain (via `rustup`) and the prebuilt
Ghostty app. Ghostty is the one deliberate exception to building from source: its Zig
toolchain is still too unstable to pin to, so it's installed from a pinned, prebuilt DMG
instead. Cloning a repo and watching it *not* pull the world is the point.

**Configuration lives in the repository.** The dotfiles set the `XDG_*` variables to
point straight into `home/` — `XDG_CONFIG_HOME` → `home/.config`, and so on. Tools read
their config from version control, not from a `$HOME` that drifts out from under you.
Edit a config, commit it, and every machine that syncs the repo has it.

**The environment layers like Nix.** There is a *global* environment and a *repo-local*
one stacked on top. The global layer is the dotfiles: `home/.zshenv` puts the
user-facing tools (`home/.local/bin` — fish, nvim, jj, rg, fd, …) on `PATH` in every
shell, everywhere. The repo-local layer is direnv: the root `.envrc` adds the language
runtimes, `CARGO_HOME`, `RUSTUP_HOME`, and the repo's own `.local/bin` — and it only
activates inside the repository, inheriting the global layer rather than replacing it.
Step into the repo and you get the build toolchain; step out and you're back to just the
tools.

`setup` is also re-runnable. Each step skips work already done at the right version, so
running it twice converges instead of redoing everything.

## The flow

`setup` runs one ordered bootstrap. **direnv and the dotfiles go first, because the
shell configuration *is* the bootstrap**: they lay down the environment — `PATH`, the
`XDG_*` variables, the direnv hook — that every later step and every future shell runs
inside. The remaining steps install fonts, the editor, and the tool builds, with the one
network-fetched app last.

The dotfiles step is a one-way mirror: it copies `home/` into `$HOME`, writing only the
files that are missing or differ, and never deleting anything. Re-running it is a no-op
once everything matches.

Once the files are in place, opening a shell pulls the environment up by its bootstraps:

```
go run ./setup
  └─ bootstrap (in order):
       direnv → dotfiles → fonts → neovim → fzf → rust → fish → jj → ripgrep → fd → ghostty
       └─ direnv + dotfiles first: the shell config is the bootstrap

new login shell
  ├─ .zshenv    sets XDG_* and the global PATH (home/.local/bin)      ── global layer
  ├─ .zprofile  eval "$(direnv hook zsh)"
  │    └─ .envrc   language runtimes, CARGO_HOME, RUSTUP_HOME,         ── repo-local layer
  │                repo .local/bin   (inherits the global layer)
  └─ .zshrc     exec fish
```

direnv lands in the global `home/.local/bin`, so it's already on `PATH` by the time
`.zprofile` asks the shell to hook it — which is what lets the repo-local layer load at
all.

## What it installs

| Step | Role | Source |
|------|------|--------|
| `direnv` | per-directory environment loader; the repo-local layer | vendored, built with Go |
| `dotfiles` | mirror `home/` configuration into `$HOME` | this repository |
| `fonts` | Iosevka Nerd Font faces | vendored TTFs |
| `neovim` | editor | vendored, built with `make` |
| `fzf` | fuzzy finder | vendored, built with Go |
| `rust` | `cargo` / `rustc` / `rustup` toolchain | installed via `rustup` |
| `fish` | interactive shell | vendored, built with `cargo` |
| `jj` | version control (Jujutsu) | vendored, built with `cargo` |
| `ripgrep` (`rg`) | search | vendored, built with `cargo` (pcre2) |
| `fd` | file finder | vendored, built with `cargo` |
| `ghostty` | terminal emulator (macOS) | pinned, prebuilt DMG |

## Usage

Clone the repository to `~/code/james-orcales`. It brings its own Go, built from the
vendored source, so there's nothing to install first. From the repository root:

```
./install_golang.sh && .local/bin/go run ./setup
```

`install_golang.sh` builds Go from the vendored source and links it into `.local/bin`;
that freshly built `go` then runs `setup`. It's a one-time bootstrap, safe to re-run —
each step skips what's already in place. Once `setup` has installed direnv and the
dotfiles, entering the repo runs `install_golang.sh` for you and puts `go` on `PATH`, so
afterwards it's just `go run ./setup`.

## Assumptions

- The repository is at `~/code/james-orcales`.
- It's run from within the repository, as your normal user (not root).
- The vendored sources under `third_party/` are present — they're committed, so a clean
  clone has them.
