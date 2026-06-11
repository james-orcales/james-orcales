export PATH="$HOME/.local/bin:$PATH"
if brew --version > /dev/null; then
        eval "$(/opt/homebrew/bin/brew shellenv)"
fi

# Place path exports in .zprofile - https://stackoverflow.com/a/34244862 Zsh on Arch [and OSX]
# sources /etc/profile – which overwrites and exports PATH – after having sourced $HOME/.zshenv
export PATH="$BIG_BANG_SHARE/nvim/nvim-macos-arm64/bin:$PATH"
export PATH="$CARGO_HOME/bin:$PATH"
# Put BIG_BANG_BIN last for it to take priority.
export PATH="$BIG_BANG_BIN:$PATH"


export MANPATH="$BIG_BANG_MAN:$MANPATH"

if command -v direnv >/dev/null 2>&1; then
        if [ -n "$CLAUDECODE" ]; then
                eval "$(direnv hook zsh)"
                eval "$(DIRENV_LOG_FORMAT= direnv export zsh)"
        fi
fi
