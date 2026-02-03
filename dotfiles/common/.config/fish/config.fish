if status is-interactive
        abbr --add --set-cursor=!! sh sh -c \'!!\'
        abbr --add n nvim .

        #    jj
        set -q BRANCH; or set -gx BRANCH james
        abbr --add a    jj
        abbr --add aa   jj abandon
        abbr --add ab   jj bookmark
        abbr --add abc  jj bookmark create $BRANCH
        abbr --add abl  jj bookmark list
        abbr --add abs  jj bookmark set $BRANCH
        abbr --add absp jj bookmark set $BRANCH --revision @-
        abbr --add abt  jj bookmark track $BRANCH
        abbr --add ac   --set-cursor=!! jj commit   -m \"!!\"
        abbr --add ad   --set-cursor=!! jj describe -m \"!!\"
        abbr --add adf  jj diff
        abbr --add adfs jj diff --stat --from main
        abbr --add ae   jj edit
        abbr --add ag   jj  git
        abbr --add agp  jj  git push --bookmark james
        abbr --add agf  jj  git fetch --tracked
        abbr --add agfp 'jj git fetch --tracked && jj git push'
        abbr --add ah   jj --help
        abbr --add al  'jj log -r \'fork_point(@|main)..(@|main) | fork_point(@|main)\' --no-pager'
        abbr --add all 'jj log -r \'::main\' --no-pager --limit=10'
        abbr --add an   jj new
        abbr --add asp  jj split
        abbr --add ash  jj show --no-pager
        abbr --add ashs jj show --no-pager --summary
        abbr --add ashn jj show --name-only
        abbr --add as   jj squash
        abbr --add asi  jj squash --interactive
        abbr --add --set-cursor=!! asft  jj squash -f !! -t
        abbr --add --set-cursor=!! asift jj squash --interactive -f !! -t
        abbr --add --set-cursor=!! asrt  jj squash -r !! -t
        abbr --add ast  jj status
        abbr --add au   jj undo
        abbr --add arv  jj revert --insert-before @ --revisions 
        abbr --add ard  jj redo
        abbr --add ars  jj restore --interactive
        abbr --add --set-cursor=!! arb jj rebase -s !! -A main
        abbr --add awl jj workspace list

        # Git
        abbr --add gc1 git clone --depth=1 
        abbr --add gc1n git clone --depth=1 --no-single-branch
        abbr --add gcfb git clone --filter=blob:none 

        # GitHub
        abbr --add ghpr 'gh pr create --title=@coderabbitai --fill-verbose --base main --head james; gh pr view --web james'

        abbr --add stdrs 'cd $(rustc --print sysroot)/lib/rustlib/src/rust/library/; and nvim .'
        abbr --add stdgo 'cd $(go env GOROOT)/src/; and nvim .'

        abbr --add --set-cursor=!! goto 'set dir $HOME/!!/; 
        cd (echo $dir/(string join \n (echo ./) (fd --type directory --max-depth 1 --base-directory $dir) | fzf)) 2>/dev/null
        and nvim .'

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
