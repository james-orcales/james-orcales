if status is-interactive
        # Wrap a one-liner in `sh -c` so I can paste shell-isms without rewriting for fish.
        abbr --add --set-cursor=!! sh sh -c \'!!\'
        abbr --add n nvim .

        # ─── jj ───
        # Naming: every jj abbr starts with `a`. Second letter follows the verb
        # (a=abandon, b=bookmark, c=commit, d=describe/diff, e=edit, g=git,
        # h=help, l=log, n=new, p=split, r=rebase/restore/redo/revert,
        # s=show/squash/status, u=undo, w=workspace).
        set -q BRANCH; or set -gx BRANCH james
        abbr --add a    jj
        abbr --add aa   jj abandon

        # bookmarks (`ab*`). $BRANCH is the per-host feature branch name.
        abbr --add ab   jj bookmark
        abbr --add abc  jj bookmark create $BRANCH
        abbr --add abl  jj bookmark list
        abbr --add abs  jj bookmark set $BRANCH
        # `absp` = "set previous" — point branch at @- when I want to push my
        # parent commit but keep iterating on @.
        abbr --add absp jj bookmark set $BRANCH --revision @-
        abbr --add abt  jj bookmark track $BRANCH

        abbr --add ac   --set-cursor=!! jj commit   -m \"!!\"
        abbr --add ad   --set-cursor=!! jj describe -m \"!!\"

        abbr --add adf  jj diff
        abbr --add adfs jj diff --stat --from main
        abbr --add ae   jj edit

        # git interop (`ag*`).
        abbr --add ag   jj  git
        abbr --add agp  jj  git push --bookmark james
        abbr --add agf  jj  git fetch --tracked
        abbr --add agfp 'jj git fetch --tracked && jj git push'

        abbr --add ah   jj --help

        # `al` = local log: only commits between fork-point and @ or main, plus
        # siblings of @ from MY workspace (excludes other workspaces' @).
        abbr --add alf  'jj log -r \'fork_point(@|main)..(@|main) | fork_point(@|main) | (@-+ ~ (working_copies() ~ @))\' --no-pager'
        # `all` = ancestors of main, last 10. Quick orientation when I'm lost.
        abbr --add alm 'jj log -r \'::main\' --no-pager --limit=10'
        abbr --add al  'jj log -r \'all()\' --no-pager --limit=10'
        abbr --add an   jj new
        abbr --add asp  jj split

        # show variants. Default uses --no-pager since I usually want short output.
        abbr --add ash  jj show --no-pager
        abbr --add ashs jj show --no-pager --summary
        abbr --add ashn jj show --name-only

        # squash variants. Default `as` squashes @ into @-.
        abbr --add as   jj squash
        abbr --add asi  jj squash --interactive
        # `asft` / `asift` / `asrt` — explicit -f/-t (or -r/-t for revisions).
        # Cursor lands at !! so I can fill in source/target.
        abbr --add --set-cursor=!! asft  jj squash -f !! -t
        abbr --add --set-cursor=!! asift jj squash --interactive -f !! -t
        abbr --add --set-cursor=!! asrt  jj squash -r !! -t

        abbr --add ast  jj status
        abbr --add au   jj undo
        # `arv` revert: insert as a child of @-, leaving @ untouched.
        abbr --add arv  jj revert --insert-before @ --revisions
        abbr --add ard  jj redo
        abbr --add ars  jj restore --interactive
        abbr --add --set-cursor=!! arb jj rebase -s !! -A main

        # workspaces (`aw*`). aw/awf/arst are functions in functions/.
        #
        # ★ Rule of thumb: when pulling changes in from secondaries, accumulate
        #   them into main's current @. NEVER squash into @- (or anything else
        #   that rewrites a commit a secondary is based on) — every secondary
        #   based on that commit goes stale at once.
        #
        # Gotchas living here, because they bit me:
        #
        # 1. Cross-workspace staleness. When main rewrites @ (squash, describe,
        #    etc.), any secondary whose @ was based on that commit goes stale.
        #    The next jj op in the secondary refuses until update-stale.
        #
        # 2. update-stale recovery. Per jj docs: when the prior op was lost,
        #    update-stale creates a *recovery commit* with the working copy's
        #    on-disk contents, parented onto the current op's WC commit.
        #    On-disk edits aren't lost — they move to a new commit.
        #
        # 3. Divergent twin. The recovery commit shares the original @'s
        #    change_id → divergence (e.g. luv/2 and luv/3). The change_id
        #    alone is now ambiguous in revsets.
        #
        # 4. No syntax for "the divergent twin." There's no @/3-style revset.
        #    To pick the recovery without naming a commit_id you need
        #    `present($cid) ~ $ws@` or `divergent() & ~working_copies()`,
        #    both of which require querying the change_id first.
        #
        # 5. --from $ws@ vs --from $ws@-. Depending on which side of the
        #    divergence $ws@ lands on after update-stale, the real edits may
        #    be at $ws@-. No stable rule — inspect before squashing.
        #
        # 6. jj squash can't create commits. It only moves diff between
        #    existing ones. To "squash into a sibling," pre-create with
        #    `jj new @-` or use `jj duplicate <src> --destination @-`.
        abbr --add awl jj workspace list
        # `awsq` — squash a secondary workspace's changes into @, excluding
        # its .gitignore (which `aw` rewrites with per-workspace symlink rules).
        # `jj status -R` snapshots the workspace before the squash so we pick
        # up its on-disk edits.
        abbr --add --set-cursor=!! awsq 'set -l ws !!; jj status -R (jj workspace root)/../$ws; and jj squash --from $ws@ --into @ \'~.gitignore\''

        # ─── Git ───
        abbr --add gc1 git clone --depth=1
        abbr --add gc1n git clone --depth=1 --no-single-branch
        # blobless clone: full history, fetch blobs lazily. Faster than depth=1
        # when you'll need to look at past file contents.
        abbr --add gcfb git clone --filter=blob:none

        # ─── GitHub ───
        # `ghpr` opens a PR using CodeRabbit's auto-generated title and body,
        # then pops it open in the browser.
        abbr --add ghpr 'gh pr create --title=@coderabbitai --fill-verbose --base main --head james; gh pr view --web james'

        # Jump into language stdlibs for grepping.
        abbr --add stdrs 'cd $(rustc --print sysroot)/lib/rustlib/src/rust/library/; and nvim .'
        abbr --add stdgo 'cd $(go env GOROOT)/src/; and nvim .'

        # `goto X` — fzf one level under ~/X. The `./` entry lets me pick the
        # parent dir itself instead of a child.
        abbr --add --set-cursor=!! goto 'set dir $HOME/!!/;
        cd (echo $dir/(string join \n (echo ./) (fd --type directory --max-depth 1 --base-directory $dir) | fzf)) 2>/dev/null
        and nvim .'

        # Project pickers — same shape as `goto` but rooted at common parents.
        abbr --add c 'set dir $HOME/code/;
        cd (echo $dir(string join \n (echo ./) (fd --type directory --max-depth 1 --base-directory $dir) | fzf)) 2>/dev/null;
        and nvim .'

        abbr --add w 'set dir $HOME/work/;
        cd (echo $dir(string join \n (echo ./) (fd --type directory --max-depth 1 --base-directory $dir) | fzf)) 2>/dev/null;
        and nvim .'

        bind ctrl-h backward-kill-word
        # bind ctrl-w : # i dont know how to erase builtin binds

        direnv hook fish | source
end
