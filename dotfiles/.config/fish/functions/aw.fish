# aw — "add workspace"
#
# Usage:
#   aw            create a sibling workspace using the next free NATO name
#   aw <name>     create a sibling workspace with the given NATO name
#
# Then cds into it and runs `direnv allow`. Build state and ignored files
# start empty in the new workspace — bootstrap them yourself.
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
    set -l names alpha bravo charlie delta edison foxtrot \
        golf hotel india juliett kilo lima \
        mike november oscar papa quebec romeo \
        sierra tango uniform victor whiskey xray \
        yankee zulu

    set -l workspace
    if set -q argv[1]
        # Explicit name: must be in the preset list and not already taken.
        if not contains -- $argv[1] $names
            echo "$argv[1] is not in the NATO preset list" >&2
            return 1
        end
        if test -e $parent/$argv[1]
            echo "$parent/$argv[1] already exists" >&2
            return 1
        end
        set workspace $parent/$argv[1]
    else
        # Auto: first NATO name whose directory doesn't already exist.
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
    end

    # Enumerate gitignored top-level entries in the main workspace BEFORE
    # creating the new workspace, so we don't pick up the new workspace's
    # own .jj/ if it lands inside $repo_root.
    #
    # --directory collapses ignored dirs to a single entry. We skip .jj/
    # because every workspace owns its own .jj/ — symlinking it would alias
    # the new workspace's operation log onto the main's.
    set -l ignored
    for entry in (git -C $repo_root ls-files --others --ignored --exclude-standard --directory)
        set -l rel (string trim --right --chars=/ -- $entry)
        test -z "$rel"; and continue
        test "$rel" = .jj; and continue
        set ignored $ignored $rel
    end

    # Actually create the workspace. jj checks out all tracked files into it.
    jj workspace add $workspace
    or return 1

    # Symlink each ignored entry from the main workspace. Build state, module
    # caches, and vendored binaries are then shared instead of rebuilt per
    # workspace. Symlinks point at absolute paths so they survive cd.
    for rel in $ignored
        ln -s $repo_root/$rel $workspace/$rel
    end

    # Write a workspace-local .gitignore listing the symlinks themselves so
    # jj treats them as ignored here. `awsq` excludes this file when
    # squashing back into main.
    printf '%s\n' $ignored >$workspace/.gitignore

    cd $workspace
    or return 1

    direnv allow
end
