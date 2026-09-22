-- test.lua — runs the whole shared_lua suite. Loads the harness, then requires every *_test.lua
-- (each runs its checks on require, accumulating into the global tally), then reports and sets the
-- process exit status. Run from anywhere, e.g. `lua shared_lua/test.lua`.
-- "./" fallback: when this file is run as a bare name (`lua test.lua` from inside shared_lua/) its
-- chunk source has no directory part, so resolve siblings against the current directory instead.
local dir = debug.getinfo(1, "S").source:match("@?(.*/)") or "./"
require(dir .. "testing")
require(dir .. "types_test") -- before base_test: it snapshots globals to prove types.lua is pure
require(dir .. "base_test")
require(dir .. "array_test")
require(dir .. "strings_test")
require(dir .. "json_test")
require(dir .. "sha_test")
tests_finish()
