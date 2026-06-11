export BIG_BANG_GIT_DIR="$HOME/code/james-orcales/big_bang"
# A good reason not to use .local/share is to keep the PATH variable short. I want to avoid symlinks
# and hardcode all variables. A possible alternative is $HOME/big_bang, but that's a decision for
# later.
export BIG_BANG_DATA_DIR="$HOME/.local/share/big_bang"
export BIG_BANG_SHARE="$BIG_BANG_DATA_DIR/share"
export BIG_BANG_BIN="$BIG_BANG_DATA_DIR/bin"
export BIG_BANG_MAN="$BIG_BANG_DATA_DIR/man"
export BIG_BANG_TMP="$BIG_BANG_DATA_DIR/tmp"


export CARGO_HOME="$BIG_BANG_SHARE/rust/.cargo"
export RUSTUP_HOME="$BIG_BANG_SHARE/rust/.rustup"
# for odin
export PATH="/opt/homebrew/opt/llvm@20/bin:$PATH"


export HOMEBREW_NO_AUTO_UPDATE=true
export HOMEBREW_BUNDLE_FILE="$BIG_BANG_DATA_DIR/Brewfile"
export HOMEBREW_CASK_OPTS_REQUIRE_SHA=true


export FZF_DEFAULT_OPTS="          \
        --reverse                          \
        --ansi                             \
        --bind='ctrl-h:backward-kill-word' \
        --bind='shift-down:half-page-down' \
        --bind='shift-up:half-page-up'     \
        --bind='home:first'                \
        --bind='end:last'                  \
        "
export EDITOR=nvim
