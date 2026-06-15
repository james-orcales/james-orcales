
# Order of Operations

The setup binary runs one bootstrap in a fixed order — direnv, dotfiles, fonts, Neovim, fzf, Rust,
fish, jj, ripgrep, fd, Ghostty — each announced by name, exiting on the first failure. direnv is
first; Ghostty, the one network download, is last, after the compile-bound cargo steps.

# Idempotency

Every binary install obeys one rule: skip when the binary at its managed location reports the wanted
version and stays reachable on PATH; otherwise reinstall and link it in. The cargo and Go builds
share the Installed probe; Ghostty also checks its code signature, and Neovim checks its own way.

### Accepts A Matching Version

Installed reports true when the binary's --version output starts with the wanted version.

### Rejects A Missing Or Stale Binary

Installed reports false when the binary is absent or reports a different version.

# Plan

Plan returns the writes that bring the home directory in line with the source dotfiles.

### Empty Source Yields No Writes

A source tree with no files produces nothing to sync.

### Missing Destination Is Created

A source file absent from the home directory is written to its mirrored path.

### Identical Destination Is Skipped

A source file whose home copy already holds the same bytes is left untouched.

### Differing Destination Is Overwritten

A source file whose home copy differs is rewritten with the source bytes.

### Nested Path Mirrors Under Home

A nested source file maps to the same relative path joined under the home directory.

### Ignored Path Is Skipped

A source path the gitignore predicate marks ignored is not synced, and an ignored directory is
pruned so its contents are never read.

# Main

Main plans the sync and writes each pending file through the injected writer.

### Writes Planned Files

Each planned file's contents reach the writer at its mirrored path, reporting success.

### Skips Identical Destination

A home directory already matching the source yields no writes on a repeat run.

### Reports Write Failure

A writer error makes the run report a non-zero exit code.

### Applies Macos Defaults

On darwin the macos defaults commands run through the injected runner after the sync.

### Skips Macos Defaults Off Darwin

On any operating system other than darwin no defaults commands run.

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

Install_Fonts copies the vendored Iosevka TTFs from `third_party/iosevka_nerd_font_mono` into the
per-OS user font directory, copying only the faces not already there, and on Linux refreshes the
font cache when it copies any. A repeat bootstrap with the fonts in place does no work.

### Skips Without A Font Directory

An empty font directory — an operating system with no known user font location — does nothing.

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

# Install Fish

Install_Fish builds fish from the vendored `third_party/fish-shell` with cargo, offline against its
committed vendor tree, and symlinks fish, fish_indent, and fish_key_reader into the PATH directory.
It probes the built binary, so a present build is relinked rather than recompiled.

### Skips Build When Already Built

When the fish binary under CARGO_HOME already reports the wanted version, the build is skipped and
the binaries are relinked rather than recompiled.

### Builds Then Links When Absent

When no fish at the wanted version is present, cargo builds and installs it and then the binaries
are symlinked into the link directory.

### Reports A Build Failure

A failing build stops before linking and reports a non-zero exit code.

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

# Install Jj

Install_Jj builds jj from the vendored `third_party/jj` workspace with cargo, offline against its
committed vendor tree, and symlinks the binary into the PATH directory. It probes the built binary,
so a present build is relinked rather than recompiled when only the symlink is missing.

### Skips Build When Already Built

When the jj binary under CARGO_HOME already reports the wanted version, the build is skipped and the
binary is relinked rather than recompiled.

### Builds Then Links When Absent

When no jj at the wanted version is present, cargo builds and installs it and then the binary is
symlinked into the link directory.

### Reports A Build Failure

A failing build stops before linking and reports a non-zero exit code.

# Install Ripgrep

Install_Ripgrep builds ripgrep from the vendored `third_party/ripgrep` with cargo and the pcre2
feature, offline against its committed vendor tree, and symlinks the rg binary into the PATH
directory. It probes the built binary, so a present build is relinked rather than recompiled.

### Skips Build When Already Built

When the rg binary under CARGO_HOME already reports the wanted version, the build is skipped and the
binary is relinked rather than recompiled.

### Builds Then Links When Absent

When no rg at the wanted version is present, cargo builds and installs it and then the binary is
symlinked into the link directory.

### Reports A Build Failure

A failing build stops before linking and reports a non-zero exit code.

# Install Fdcli

Install_Fdcli builds fd from the vendored `third_party/fd` with cargo, offline against its committed
vendor tree, and symlinks the binary into the PATH directory. It probes the built binary, so a
present build is relinked rather than recompiled.

### Skips Build When Already Built

When the fd binary under CARGO_HOME already reports the wanted version, the build is skipped and the
binary is relinked rather than recompiled.

### Builds Then Links When Absent

When no fd at the wanted version is present, cargo builds and installs it and then the binary is
symlinked into the link directory.

### Reports A Build Failure

A failing build stops before linking and reports a non-zero exit code.

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
