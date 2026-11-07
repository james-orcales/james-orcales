-- Imported by default
local info = debug.getinfo(1, "S")
local dir = info.source:match("@?(.*/)")
require(dir .. "base")
require(dir .. "strings")
require(dir .. "array")

local M = {
        version = "alpha-2025-11",
}

local mt = {
        __index = function(t, key)
                if key == "json" then
                        local mod = require(dir .. "json")
                        rawset(t, "json", mod)
                        return mod
                elseif key == "sha" then
                        -- This is slow. Use it only to verify your language compiler's checksum during bootstrapping; after
                        -- that, use that language for other scripts.
                        local mod = require(dir .. "sha")
                        rawset(t, "sha", mod)
                        return mod
                end
        end,
}
setmetatable(M, mt)

return M
