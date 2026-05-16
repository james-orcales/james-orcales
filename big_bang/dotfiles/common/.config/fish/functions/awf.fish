# awf — "forget workspace"
#
# Forgets a workspace and deletes its directory. Must be run from the main
# workspace. Refuses to operate on names not in `aw`'s NATO preset list. All
# logic lives in awf.lua next to this file.
function awf
    set -l fn_dir (status dirname)
    ~/code/big_bang/bin/darwin/lua $fn_dir/awf.lua $argv
end
