# awf — "forget workspace"
#
# Inverse of `aw`: forgets the current jj workspace from the shared store
# and deletes its directory. Refuses to run on the main workspace, and
# refuses to delete any directory that wasn't created by `aw` (i.e. whose
# basename isn't one of the NATO names).
function awf
    set -l ws_root (jj workspace root)
    or return 1
    set -l ws_name (basename $ws_root)

    # Refuse to delete the main workspace. Identification trick:
    # main has .jj/repo/ as a real directory (the actual store);
    # secondary workspaces have .jj/repo as a file pointing at the main store.
    if test -d $ws_root/.jj/repo
        echo "refusing to forget main workspace at $ws_root" >&2
        return 1
    end

    # Safety net: only allow deletion if this looks like a workspace `aw`
    # would have created. Protects against running awf in some unrelated
    # secondary jj workspace and nuking it by mistake.
    set -l names alpha bravo charlie delta echo foxtrot \
        golf hotel india juliett kilo lima \
        mike november oscar papa quebec romeo \
        sierra tango uniform victor whiskey xray \
        yankee zulu
    if not contains -- $ws_name $names
        echo "refusing to delete $ws_root: name not in awa preset list" >&2
        return 1
    end

    # Resolve the path to the main workspace before we forget+delete, so we
    # have somewhere to cd to afterward (we can't sit in a directory we just
    # deleted). See aw.fish for an explanation of the .jj/repo pointer file.
    set -l main_repo (realpath $ws_root/.jj/(cat $ws_root/.jj/repo))
    set -l main_root (dirname (dirname $main_repo))

    # Run forget BEFORE cd-ing or deleting — jj needs .jj/repo to resolve
    # the store, which it does relative to the current workspace.
    jj workspace forget $ws_name
    or return 1

    cd $main_root
    rm -rf $ws_root
end
