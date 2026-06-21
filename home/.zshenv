export PATH="$HOME/.local/bin:$PATH"
export PATH="$HOME/code/james-orcales/home/.local/bin:$PATH"

export XDG_DATA_HOME="$HOME/code/james-orcales/home/.local/share"
export XDG_STATE_HOME="$HOME/code/james-orcales/home/.local/state"
export XDG_CACHE_HOME="$HOME/code/james-orcales/home/.local/cache"
export XDG_CONFIG_HOME="$HOME/code/james-orcales/home/.config/"

export GIT_CONFIG_NOSYSTEM=1
export GIT_CONFIG_GLOBAL="$XDG_CONFIG_HOME/git/config"

export DOCKER_CONFIG="$XDG_CONFIG_HOME/docker/"


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
