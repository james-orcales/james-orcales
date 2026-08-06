
# Main

### Runs Complete Bootstrap

Main validates the host facts and stops invalid input with EXIT_USAGE. Otherwise, it returns the
complete bootstrap runner. The runner uses one shared `io.IO` value and stops with the first
failing step status or EXIT_SUCCESS.

# Runner

### Queues Late Duplicate Retirement

A second IO callback records its failure even after the root submits the first callback's
continuation. The runner does not require both callbacks to arrive in one loop pass.

### Stops After Submitted IO Retires

A terminal status does not stop the runner while a submitted IO operation remains armed. The root
continues its pump until each callback retires, then it can deinitialize the Driver.

# Bootstrap Steps

Bootstrap_Steps constructs the ordered steps from host facts and one shared `io.IO` value. Setup
submits directory, status, file, and process operations through it. It declares no second IO seam,
accepts the longest validated source path, and rejects a completion that does not retire once.

# Order of Operations

The setup binary runs one bootstrap in a fixed order — direnv, dotfiles, fonts, Neovim, fzf,
maddox, m2p, sloc, timeout, Rust, jj, ripgrep, fd, Ghostty — each announced by name, exiting on
the first failure. direnv is first; the Go builds precede the cargo steps; Ghostty downloads last.

# Idempotency

Every binary install skips when the binary at its managed location reports the wanted version and
stays reachable on PATH; otherwise it reinstalls and links in. Ghostty also checks its signature,
Neovim its own way; Install_Command, for this repo's own unversioned commands, gates on PATH alone.

### Accepts A Matching Version

Installed reports true when the binary's --version output starts with the wanted version.

### Rejects A Missing Or Stale Binary

Installed reports false when the binary is absent, its probe fails, or it reports another version.

# Mirror

Mirror plans the sync, writes each pending file through shared IO, and then applies
the macos defaults on darwin. The simulation harness proves the mirror properties.

### Applies Macos Defaults

On darwin the macos defaults commands run through `io.IO.Spawn` after the sync.

### Skips Macos Defaults Off Darwin

On any operating system other than darwin no defaults commands run.

### Narrates The Scan

Mirror names each source directory as the walk reads it, so a large silent tree scan shows it
is advancing rather than looking hung, and reports an up-to-date tree when it writes nothing.

### Probes Each Directory In One Batch

The walk classifies each directory's entries with one Is_Ignored call carrying them all. It does
not start one gitignore subprocess per entry.

### Rejects A Failed Ignore Probe

The walk stops before it writes a file when the gitignore probe returns an error, does not retire
exactly one time, or exits with a nonzero status.

# Install Neovim

Install_Neovim builds the vendored Neovim from `third_party/neovim` and installs it under
`home/.local`, where it lands at `home/.local/bin/nvim`. It does nothing when the checkout's own
nvim is already installed at the wanted release, so a repeat bootstrap does no work.

### Skips Build When Installed From Checkout

When nvim resolves to a path inside the checkout and reports the wanted release, no make runs.

### Builds When The Match Is Outside Repository

An nvim resolving outside the checkout does not count as installed, so the build still runs and its
version is never consulted.

### Configures Prefix Then Installs

make runs against the vendored source twice, in order: first to configure and build with the
install prefix, then to install. Both invocations carry the same prefix; only the second adds the
install goal. Each phase is announced before its make runs.

### Reports Build Failure

A failing make stops the bootstrap before installing and reports a non-zero exit code.

# Install Fonts

Install_Fonts copies vendored Iosevka TTFs into the per-OS user font directory and on Linux
refreshes the font cache after a copy. A repeat with the fonts in place does no work. The
bootstrap omits this work on a host that has no managed font destination.

### Copies Only Missing Fonts

Each vendored face absent from the destination is copied; faces already present are left alone.
Each copy and each skip is logged.

### Skips When All Present

When every face is already in the destination, nothing is copied and the cache is not refreshed.

### Refreshes Cache After Copies

After copying at least one face, a provided cache refresh runs; macOS provides none and is skipped.

### Reports A Copy Failure

A failed font copy makes the run report a non-zero exit code.

# Install Direnv

Install_Direnv builds direnv from the vendored `third_party/direnv` with the Go toolchain, offline
against its committed vendor tree, straight into the bin directory, where the shell hook and every
.envrc find it. It probes the built binary, so a present build at the wanted version is left alone.

### Skips Build When Already Built

When the direnv binary in the bin directory already reports the wanted version, the build is
skipped.

### Builds When Absent

When no direnv at the wanted version is present, the Go toolchain builds it into the bin directory.

### Reports A Build Failure

A failing build reports a non-zero exit code.

# Install Rust

Install_Rust installs the pinned Rust toolchain with rustup into CARGO_HOME, then symlinks cargo,
rustup, and rustc into the PATH directory. It probes rustc at CARGO_HOME directly, so it reinstalls
when the toolchain is missing or the wrong version and otherwise relinks without reinstalling.

### Skips Install When Already Installed

When the rust toolchain at CARGO_HOME reports the wanted version, the install is skipped and the
toolchain is relinked rather than reinstalled.

### Installs Then Links When Absent

When the toolchain is not linked, rustup installs it and then cargo, rustup, and rustc are symlinked
into the link directory.

### Reports An Install Failure

A failing install reports a non-zero exit code.

# Install Fzf

Install_Fzf builds fzf from the vendored `third_party/fzf` with the Go toolchain, offline against
its committed vendor tree, straight into the bin directory. It probes the built binary, so a present
build at the wanted version is left alone rather than recompiled.

### Skips Build When Already Built

When the fzf binary in the bin directory already reports the wanted version, the build is skipped.

### Builds When Absent

When no fzf at the wanted version is present, the Go toolchain builds it into the bin directory.

### Reports A Build Failure

A failing build reports a non-zero exit code.

# Install Command

Install_Command builds a command from this repository — maddox, m2p (markdown_to_pdf), or sloc —
with the Go toolchain straight into `home/.local/bin`, where the build output is the install. Unlike
the vendored tools, its only idempotency check is whether the command already resolves on PATH.

### Skips Build When Already On Path

When the command name already resolves on PATH, no build runs.

### Builds When Absent

When the command name does not resolve on PATH, the Go toolchain builds it into the bin directory.

### Reports A Build Failure

A failing build reports a non-zero exit code.

# Install Jj

Install_Jj builds jj from the vendored `third_party/jj` workspace with cargo, offline against its
committed vendor tree, and installs the binary straight into the bin directory via cargo `--root`,
probing the built binary so a present build at the wanted version is left alone rather than rebuilt.

### Skips Build When Already Built

When the jj binary in the bin directory already reports the wanted version, the build is skipped.

### Builds When Absent

When no jj at the wanted version is present, cargo builds and installs it into the bin directory.

### Reports A Build Failure

A failing build reports a non-zero exit code.

# Install Ripgrep

Install_Ripgrep builds ripgrep from the vendored `third_party/ripgrep` with cargo and the pcre2
feature, offline against its committed vendor tree, installing the rg binary straight into the bin
directory via cargo `--root`; a present build at the wanted version is left alone, not recompiled.

### Skips Build When Already Built

When the rg binary in the bin directory already reports the wanted version, the build is skipped.

### Builds When Absent

When no rg at the wanted version is present, cargo builds and installs it into the bin directory.

### Reports A Build Failure

A failing build reports a non-zero exit code.

# Install Fdcli

Install_Fdcli builds fd from the vendored `third_party/fd` with cargo, offline against its committed
vendor tree, and installs the binary straight into the bin directory (via cargo `--root`). It probes
the built binary, so a present build at the wanted version is left alone rather than recompiled.

### Skips Build When Already Built

When the fd binary in the bin directory already reports the wanted version, the build is skipped.

### Builds When Absent

When no fd at the wanted version is present, cargo builds and installs it into the bin directory.

### Reports A Build Failure

A failing build reports a non-zero exit code.

# Install Ghostty

Install_Ghostty downloads the pinned Ghostty DMG, verifies its SHA256, installs the app, and
symlinks its CLI into the PATH directory. A present install at the wanted version whose code
signature still verifies is relinked, not re-downloaded; off darwin the step does nothing.

### Skips Install When Already Installed

When the installed Ghostty app already reports the wanted version and its code signature verifies,
the download is skipped and the app's CLI is relinked rather than re-downloaded.

### Reinstalls When Signature Is Invalid

An installed app at the wanted version whose code signature no longer verifies is not trusted: the
DMG install runs again, replacing the tampered bundle.

### Pins The Signing Team

The signature check pins Ghostty's signing Team ID, so an app validly signed by another developer
is rejected even though its own signature verifies, forcing a reinstall.

### Installs Then Links When Absent

When no Ghostty at the wanted version is installed, the DMG install runs and then the CLI is
symlinked into the link directory.

### Reports An Install Failure

A failing install stops before linking and reports a non-zero exit code.
