if brew --version > /dev/null; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
fi

# Hook direnv unconditionally. fish lives in the direnv-managed .local/bin, so zsh
# must load direnv — and the PATH it exports — here in .zprofile before .zshrc's
# `exec fish` can even find fish. Gating this on $CLAUDECODE left a plain zsh (one
# that never reached fish) with no direnv, so .envrc never loaded and XDG_* unset.
if command -v direnv >/dev/null 2>&1; then
        eval "$(direnv hook zsh)"
        eval "$(DIRENV_LOG_FORMAT= direnv export zsh)"
fi
