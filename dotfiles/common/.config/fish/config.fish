if status is-interactive
        abbr --add --set-cursor=!! sh sh -c \'!!\'
        abbr --add n nvim .

        #    jj
        abbr --add a    jj
        abbr --add ab   jj bookmark
        abbr --add abc  jj bookmark create
        abbr --add abl  jj bookmark list
        abbr --add abs  jj bookmark set
        abbr --add --set-cursor=!! absp jj bookmark set !! --revision @-
        abbr --add --set-cursor=!! ac jj commit   -m \"!!\"
        abbr --add --set-cursor=!! ad jj describe -m \"!!\"
        abbr --add ae   jj edit
        abbr --add ag   jj git
        abbr --add agp  jj git push
        abbr --add agf  jj git fetch --tracked
        abbr --add ah   jj --help
        abbr --add al   jj log
        abbr --add an   jj new
        abbr --add asp  jj split
        abbr --add ash  jj show
        abbr --add asq  jj squash
        abbr --add ast  jj status
        abbr --add au   jj undo
        abbr --add ar   jj redo

        # Git
        abbr --add gc1 git clone --depth=1 
        abbr --add gc1n git clone --depth=1 --no-single-branch
        abbr --add gcfb git clone --filter=blob:none 

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
end
