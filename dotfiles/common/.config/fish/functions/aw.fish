# aw — "add workspace"
#
# Creates a fresh sibling jj workspace next to the current one, named after
# the next free NATO letter (alpha, bravo, ...). Symlinks all git-ignored
# files from the main workspace into the new one so things like ./bin,
# ./.go-path, build artifacts, .envrc.local, etc. are shared rather than
# duplicated. Then cds into it and runs `direnv allow`.
#
# Why symlinks: secondary jj workspaces are full checkouts of tracked files,
# but ignored files (build outputs, deps, secrets) aren't replicated. We want
# them to "just work" without rebuilding from scratch in every workspace.
function aw
    # jj workspace root works from any workspace (main or secondary).
    # git rev-parse would fail in secondary workspaces because they have no .git/.
    set -l ws_root (jj workspace root)
    or return 1

    # Find the MAIN workspace's path. We need this because:
    #   1. We want to symlink files from the main workspace, not whichever
    #      secondary we happened to be standing in.
    #   2. New workspaces should be siblings of the main repo, not nested
    #      under a secondary.
    #
    # In jj, the main workspace has .jj/repo/ as a real directory (the store).
    # Secondary workspaces have .jj/repo as a FILE whose contents are a path
    # (relative to .jj/) pointing at the main workspace's .jj/repo directory.
    set -l repo_root
    if test -d $ws_root/.jj/repo
        # We're already in the main workspace.
        set repo_root $ws_root
    else
        # We're in a secondary. Read the pointer file, resolve it to an
        # absolute path, then strip /repo and /.jj to land at the main root.
        set repo_root (dirname (dirname (realpath $ws_root/.jj/(cat $ws_root/.jj/repo))))
    end

    # New workspaces live as siblings of the main repo, e.g.
    # ~/work/bot-management-solution + ~/work/alpha + ~/work/bravo.
    set -l parent (dirname $repo_root)

    # Pool of workspace names. NATO alphabet because they're memorable,
    # ordered, and we'll never realistically need more than 26 at once.
    set -l names alpha bravo charlie delta echo foxtrot \
        golf hotel india juliett kilo lima \
        mike november oscar papa quebec romeo \
        sierra tango uniform victor whiskey xray \
        yankee zulu

    # Pick the first name whose directory doesn't already exist.
    set -l workspace
    for n in $names
        set -l candidate $parent/$n
        if not test -e $candidate
            set workspace $candidate
            break
        end
    end
    if test -z "$workspace"
        echo "all workspace slots taken in $parent" >&2
        return 1
    end

    # Actually create the workspace. jj checks out all tracked files into it.
    jj workspace add $workspace
    or return 1

    # Copy (don't symlink) the main .gitignore so we can append per-workspace
    # ignore rules below without modifying the main repo's file. We accept
    # that this shows as "M .gitignore" in the new workspace's jj status.
    if test -e $repo_root/.gitignore
        cp $repo_root/.gitignore $workspace/.gitignore
    end

    # Symlink every git-ignored path from main into the new workspace.
    #
    #   --others           untracked files
    #   --ignored          ...that are also ignored
    #   --exclude-standard apply .gitignore + .git/info/exclude + global
    #   --directory        collapse a fully-ignored directory to a single entry
    #                      (so we get "bin/" instead of every file under bin/)
    for rel in (git -C $repo_root ls-files --others --ignored --exclude-standard --directory)
        # ls-files --directory adds a trailing slash to directories. Strip it
        # so $repo_root/$rel and $workspace/$rel are clean paths.
        set rel (string trim --right --chars=/ -- $rel)
        if test -z "$rel"; or test "$rel" = .
            continue
        end

        set -l src $repo_root/$rel
        set -l dst $workspace/$rel

        # The destination's parent directory might not exist yet (e.g.
        # dashboard/.next when dashboard/ has no other ignored siblings
        # already created). Create it.
        mkdir -p (dirname $dst)
        or continue

        # Suppress "File exists" errors silently — if some path already
        # exists from `jj workspace add` we just skip it.
        ln -s $src $dst 2>/dev/null

        # Explicitly ignore this exact path. We need this because:
        # The symlinks we just created are SYMLINKS, not directories.
        # .gitignore patterns like "bin/" only match real directories,
        # so a symlink at "bin" pointing to a directory would NOT be
        # ignored by the trailing-slash pattern. By appending "/bin"
        # (no trailing slash) we match the symlink itself by exact path.
        echo "/$rel" >> $workspace/.gitignore
    end

    cd $workspace
    or return 1

    # Trust the workspace's .envrc/.envrc.local (which are symlinked from main).
    direnv allow

    claude
end
