# fish-like zsh. Plugins are vendored beside this file and sourced by absolute
# path so this works from the repo or from $HOME. %x = file being sourced.
plugin_dir=${${(%):-%x}:A:h}

# Per-host feature branch. `set -q BRANCH; or set -gx BRANCH james` in fish.
export BRANCH="${BRANCH:-james}"

# Force emacs keybindings. Without this, zsh auto-selects VI mode because
# $EDITOR=nvim matches *vi*. Must precede every bindkey below (they attach to
# whichever keymap is selected here).
bindkey -e

# --- history: backbone of both autosuggestions and up-arrow search ---
HISTFILE="${XDG_STATE_HOME:-$HOME/.local/state}/zsh/history"
test -d "${HISTFILE:h}" || mkdir -p "${HISTFILE:h}"
HISTSIZE=100000
SAVEHIST=100000
setopt SHARE_HISTORY        # fish shares history across sessions; match it
setopt EXTENDED_HISTORY
setopt HIST_IGNORE_ALL_DUPS
setopt HIST_IGNORE_SPACE
setopt HIST_REDUCE_BLANKS

# --- directory history: fish-style `cd -`, `cd -2`, and `cd -<Tab>` menu ---
setopt AUTO_PUSHD           # every cd pushes the old dir onto the stack
setopt PUSHD_IGNORE_DUPS    # no duplicate entries in the stack
setopt PUSHD_SILENT         # don't print the stack on every cd

# --- completion: case-insensitive, colored, arrow-key menu ---
autoload -Uz compinit
_zcompdir="${XDG_CACHE_HOME:-$HOME/.cache}/zsh"
test -d "${_zcompdir}" || mkdir -p "${_zcompdir}"
compinit -d "${_zcompdir}/zcompdump"
zstyle ':completion:*' matcher-list 'm:{a-zA-Z}={A-Za-z}'
zstyle ':completion:*' menu select
zstyle ':completion:*' list-colors ${(s.:.)LS_COLORS}   # unquoted on purpose: the (s) flag word-splits
zstyle ':completion:*:descriptions' format '%F{8}%d%f'

# --- prompt: replicates fish_prompt.fish (two blank lines, cwd, git:(branch)) ---
# Branch name, or short hash when detached — matches fish's _git_branch_name.
setopt PROMPT_SUBST
git_prompt() {
        command git rev-parse --is-inside-work-tree &>/dev/null || return
        local ref
        ref=$(command git symbolic-ref --short --quiet HEAD 2>/dev/null) \
                || ref=$(command git rev-parse --short HEAD 2>/dev/null) || return
        print -rn -- " %B%F{blue}git:(%F{red}${ref}%F{blue})%f%b"
}
PROMPT=$'\n\n%B%F{cyan}%~%f%b$(git_prompt)\n%(!.#.$) '

# --- fish-style abbreviations (self-contained, supports --set-cursor via !!) ---
# Fish-syntax expansions (set/and/(cmd)) are translated to zsh (x=/&&/$(cmd)).
# jj naming: every jj abbr starts with `a`; 2nd letter follows the verb.
typeset -A abbreviations=(
        sh    "sh -c '!!'"
        n     'nvim .'

        a     'jj'
        aa    'jj abandon'

        ab    'jj bookmark'
        abc   'jj bookmark create $BRANCH'
        abl   'jj bookmark list'
        abs   'jj bookmark set $BRANCH'
        absp  'jj bookmark set $BRANCH --revision @-'
        abt   'jj bookmark track $BRANCH'

        ac    'jj commit -m "!!"'
        ad    'jj describe -m "!!"'
        adf   'jj diff'
        adfs  'jj diff --stat --from main'
        ae    'jj edit'

        ag    'jj git'
        agp   'jj git push --bookmark james'
        agf   'jj git fetch --tracked'
        agfp  'jj git fetch --tracked && jj git push'
        ah    'jj --help'

        alf   "jj log -r 'fork_point(@|main)..(@|main) | fork_point(@|main) | (@-+ ~ (working_copies() ~ @))' --no-pager"
        alm   "jj log -r '::main' --no-pager --limit=10"
        al    "jj log -r 'all()' --no-pager --limit=10"
        an    'jj new'
        asp   'jj split'

        ash   'jj show --no-pager'
        ashs  'jj show --no-pager --summary'
        ashn  'jj show --name-only'

        as    'jj squash'
        asi   'jj squash --interactive'
        asft  'jj squash -f !! -t'
        asift 'jj squash --interactive -f !! -t'
        asrt  'jj squash -r !! -t'

        ast   'jj status'
        au    'jj undo'
        arv   'jj revert --insert-before @ --revisions'
        ard   'jj redo'
        ars   'jj restore --interactive'
        arb   'jj rebase -s !! -A main'

        awl   'jj workspace list'
        # awsq: snapshot the secondary (jj status -R) so we pick up its on-disk
        # edits, then squash its @ into ours, excluding its per-workspace .gitignore.
        awsq  $'ws=!!; jj status -R $(jj workspace root)/../$ws && jj squash --from $ws@ --into @ \'~.gitignore\''

        gc1   'git clone --depth=1'
        gc1n  'git clone --depth=1 --no-single-branch'
        gcfb  'git clone --filter=blob:none'

        ghpr  'gh pr create --title=@coderabbitai --fill-verbose --base main --head james; gh pr view --web james'

        stdrs 'cd $(rustc --print sysroot)/lib/rustlib/src/rust/library/ && nvim .'
        stdgo 'cd $(go env GOROOT)/src/ && nvim .'

        # fzf one level under a root; ./ entry lets you pick the parent itself.
        goto  'dir=$HOME/!!/; cd "$dir/$({ echo ./; fd --type directory --max-depth 1 --base-directory $dir; } | fzf)" 2>/dev/null && nvim .'
        c     'dir=$HOME/code/; cd "$dir$({ echo ./; fd --type directory --max-depth 1 --base-directory $dir; } | fzf)" 2>/dev/null && nvim .'
        w     'dir=$HOME/work/; cd "$dir$({ echo ./; fd --type directory --max-depth 1 --base-directory $dir; } | fzf)" 2>/dev/null && nvim .'
)
abbr_cursor='!!'
_expand_abbrev() {
        emulate -L zsh
        setopt extended_glob
        local word="${LBUFFER##* }"
        local exp="${abbreviations[$word]}"
        test "${exp}" != "" || return 1
        # Expand only in command position — at line start or right after a
        # separator (&&, ||, ;, |, &, (, {) — like fish's default. Without this a
        # word like `ghpr` after `&&` would be treated as a plain argument.
        local head="${LBUFFER%"$word"}"          # text before the word — kept on expand
        local before="${head%%[[:space:]]#}"     # ...trailing whitespace trimmed for the check
        case "${before[-1]}" in
                "" | ";" | "&" | "|" | "(" | "{" | $'\n') ;;
                *) return 1 ;;
        esac
        case "${exp}" in
                *"${abbr_cursor}"*)
                        LBUFFER="${head}${exp%%${abbr_cursor}*}"    # prefix + text before the marker
                        RBUFFER="${exp#*${abbr_cursor}}${RBUFFER}"  # text after the marker
                        return 0                                     # cursor placed → swallow key
                        ;;
        esac
        LBUFFER="${head}${exp}"
        return 1
}
_abbr_space()  { _expand_abbrev || zle self-insert }
_abbr_accept() { _expand_abbrev || zle accept-line }
zle -N _abbr_space
zle -N _abbr_accept
bindkey ' '  _abbr_space
bindkey '^M' _abbr_accept
bindkey '^J' _abbr_accept

# fish: `bind ctrl-h backward-kill-word`
bindkey '^H' backward-kill-word

# Home / End → start/end of line. Bind every common variant (CSI, SS3 for
# application keypad mode, vt220) so it works regardless of the terminal.
bindkey '^[[H'  beginning-of-line   # Home
bindkey '^[OH'  beginning-of-line   # Home (application mode)
bindkey '^[[1~' beginning-of-line   # Home (vt220/linux)
bindkey '^[[F'  end-of-line         # End
bindkey '^[OF'  end-of-line         # End  (application mode)
bindkey '^[[4~' end-of-line         # End  (vt220/linux)

# Ctrl-C clears the current line in place instead of dropping to a fresh prompt.
# Ctrl-C arrives as SIGINT, not a keystroke, so it can only be caught in a trap.
# Guard on `zle`: only rewrite the buffer while editing at the prompt (via a
# widget, since BUFFER is read-only from a bare trap). A running app, loop, or
# script instead gets a normal interrupt (return 128+signo → abort the command).
TRAPINT() {
        if zle; then
                zle kill-buffer
                zle reset-prompt
                return 0
        fi
        return $(( 128 + $1 ))
}

# --- autosuggestions: fish's grey (fish_color_autosuggestion 808080 = color 244) ---
source "${plugin_dir:?💥}/zsh_autosuggestions.zsh"
ZSH_AUTOSUGGEST_STRATEGY=(history completion)
ZSH_AUTOSUGGEST_HIGHLIGHT_STYLE='fg=244'
# Accept with → (or Ctrl-F / End). Accept one word: Ctrl-→.

# --- up/down: search history by what you've already typed (substring) ---
source "${plugin_dir:?💥}/zsh_history_substring_search.zsh"
HISTORY_SUBSTRING_SEARCH_HIGHLIGHT_FOUND='standout'      # ~ fish_color_search_match --reverse
HISTORY_SUBSTRING_SEARCH_HIGHLIGHT_NOT_FOUND='fg=red'
bindkey '^[[A' history-substring-search-up       # Up
bindkey '^[OA' history-substring-search-up       # Up  (application mode)
bindkey '^[[B' history-substring-search-down     # Down
bindkey '^[OB' history-substring-search-down     # Down (application mode)
