local NAMES = {
    alpha=1, bravo=1, charlie=1, delta=1, edison=1, foxtrot=1,
    golf=1, hotel=1, india=1, juliett=1, kilo=1, lima=1,
    mike=1, november=1, oscar=1, papa=1, quebec=1, romeo=1,
    sierra=1, tango=1, uniform=1, victor=1, whiskey=1, xray=1,
    yankee=1, zulu=1,
}

local function die(msg)
    io.stderr:write(msg .. "\n")
    os.exit(1)
end

local function sq(s)
    return "'" .. (s:gsub("'", [['\'']])) .. "'"
end

local function capture(cmd)
    local f = assert(io.popen(cmd, "r"))
    local out = f:read("*a")
    f:close()
    return out
end

local function dirname(p)
    return (p:match("(.*)/[^/]+$")) or "."
end

if not arg[1] or arg[1] == "" then
    die("usage: awf <workspace-name> [workspace-name ...] | awf 'all()'")
end

local ws_root = capture("jj workspace root 2>/dev/null"):gsub("%s+$", "")
if ws_root == "" then
    die("not in a jj repo")
end

if not os.execute("test -d " .. sq(ws_root .. "/.jj/repo")) then
    die("awf must be run from the main workspace")
end

local targets = {}
if arg[1] == "all()" then
    if #arg > 1 then
        die("all() takes no other arguments")
    end
    local out = capture("jj workspace list 2>/dev/null")
    for line in out:gmatch("[^\r\n]+") do
        local name = line:match("^([^:]+):")
        if name and name ~= "default" then
            table.insert(targets, name)
        end
    end
else
    for _, ws_name in ipairs(arg) do
        if not NAMES[ws_name] then
            die("refusing to reset " .. ws_name .. ": not in aw preset list")
        end
        table.insert(targets, ws_name)
    end
end

for _, ws_name in ipairs(targets) do
    local target = dirname(ws_root) .. "/" .. ws_name

    os.execute("jj abandon " .. sq(ws_name .. "@") .. " 2>/dev/null")
    os.execute("jj workspace forget " .. sq(ws_name) .. " 2>/dev/null")
    -- Go's module cache files are mode 0444. chmod -R u+w so rm -rf can delete.
    os.execute("chmod -R u+w " .. sq(target) .. " 2>/dev/null")
    os.execute("rm -rf " .. sq(target))
end
