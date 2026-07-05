if brew --version > /dev/null; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
fi

# Hook direnv unconditionally. fish lives in the direnv-managed .local/bin, so zsh
# must load direnv — and the PATH it exports — here in .zprofile before .zshrc's
# `exec fish` can even find fish. Gating this on $CLAUDECODE left a plain zsh (one
# that never reached fish) with no direnv, so .envrc never loaded and XDG_* unset.
if command -v direnv >/dev/null 2>&1; then
        # Strip direnv's SIGINT-trap juggling from its hook: the `trap - SIGINT` it
        # runs every precmd resets the handler to default, which deletes .zshrc's
        # TRAPINT (the Ctrl-C line-clear) — in zsh a TRAP<sig> function and
        # `trap ... sig` are the same slot. Without this, Ctrl-C only clears in a
        # shell that skipped .zprofile (e.g. `exec zsh`).
        eval "$(direnv hook zsh | grep --invert-match 'trap.*SIGINT')"
        eval "$(DIRENV_LOG_FORMAT= direnv export zsh)"
fi
