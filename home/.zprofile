export PATH="$HOME/.local/bin:$PATH"
if brew --version > /dev/null; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
fi

export MANPATH="$BIG_BANG_MAN:$MANPATH"

if command -v direnv >/dev/null 2>&1; then
        if [ -n "$CLAUDECODE" ]; then
                eval "$(direnv hook zsh)"
                eval "$(DIRENV_LOG_FORMAT= direnv export zsh)"
        fi
fi
