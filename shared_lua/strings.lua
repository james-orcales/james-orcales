local info = debug.getinfo(1, "S")
local dir = info.source:match("@?(.*/)")
require(dir .. "base")

string.cut = def("string", "string", "->", "string", "string", "boolean", function(s, sep)
        if s == "" or sep == "" then
                return s, "", false
        end
        local i, j = s:find(sep, 1, true)
        if i then
                local left, right = s:sub(1, i - 1), s:sub(j + 1)
                return left, right, true
        else
                return s, "", false
        end
end)

string.split = def("string", "?string", "->", "table", function(s, sep)
        if s == "" or sep == "" then
                return { s }
        end
        sep = sep or " "
        local result = {}
        local pattern = "(.-)" .. sep
        local last_end = 1
        local s_start, s_end, cap = s:find(pattern, 1)
        while s_start do
                table.insert(result, cap)
                last_end = s_end + 1
                s_start, s_end, cap = s:find(pattern, last_end)
        end
        table.insert(result, s:sub(last_end))
        return result
end)

string.count = def("string", "string", "->", "number", function(self, substr)
        if substr == "" then
                return 0
        end
        local result = 0
        local i = 1
        while true do
                local start = self:find(substr, i, true)
                if not start then
                        break
                end
                result = result + 1
                i = start + #substr
        end
        return result
end)

string.lines = def("string", "->", "table", function(s)
        local t = {}
        -- Delegate to lines_iter so both share one definition of what a line is: a trailing
        -- newline terminates the last line rather than adding an empty one.
        for line in string.lines_iter(s) do
                table.insert(t, line)
        end
        return t
end)

string.lines_iter = function(s)
        s = s:gsub("\r\n", "\n"):gsub("\r", "\n")
        local pos = 1
        return function()
                if pos > #s then
                        return nil
                end
                local next_pos = s:find("\n", pos, true)
                local line
                if next_pos then
                        line = s:sub(pos, next_pos - 1)
                        pos = next_pos + 1
                else
                        line = s:sub(pos)
                        pos = #s + 1
                end
                return line
        end
end

string.contains = def("string", "string", "->", "boolean", function(s, sub)
        return s:find(sub, 1, true) ~= nil
end)

string.contains_any = def("string", "table", "->", "boolean", function(s, subs)
        for _, sub in ipairs(subs) do
                if s:find(sub, 1, true) then
                        return true
                end
        end
        return false
end)

string.common_prefix = def("string", "string", "->", "string", function(a, b)
        local len = math.min(#a, #b)
        local i = 1
        while i <= len and a:sub(i, i) == b:sub(i, i) do
                i = i + 1
        end
        return a:sub(1, i - 1)
end)

string.has_prefix = def("string", "string", "->", "boolean", function(s, prefix)
        return s:sub(1, #prefix) == prefix
end)

string.has_suffix = def("string", "string", "->", "boolean", function(s, suffix)
        return s:sub(-#suffix) == suffix
end)

string.trim_left = def("string", "string", "->", "string", function(s, chars)
        return s:match("^[" .. chars .. "]*(.*)$") or s
end)

string.trim_right = def("string", "string", "->", "string", function(s, chars)
        return s:match("^(.-)[" .. chars .. "]*$") or s
end)

string.trim = def("string", "?string", "->", "string", function(s, chars)
        chars = chars or "%s"
        return s:match("^[" .. chars .. "]*(.-)[" .. chars .. "]*$")
end)

string.trim_space = def("string", "->", "string", function(s)
        return s:match("^%s*(.-)%s*$")
end)

string.trim_left_space = def("string", "->", "string", function(s)
        return s:match("^%s*(.*)")
end)

string.trim_right_space = def("string", "->", "string", function(s)
        return s:match("(.-)%s*$")
end)

string.trim_left_null = def("string", "->", "string", function(s)
        return s:match("^\0*(.*)")
end)

string.trim_right_null = def("string", "->", "string", function(s)
        return s:match("(.-)\0*$")
end)

string.trim_null = def("string", "->", "string", function(s)
        return s:match("^\0*(.-)\0*$")
end)

string.expand_tabs = def("string", "?number", "->", "string", function(s, tabsize)
        tabsize = tabsize or 8
        return (s:gsub("\t", string.rep(" ", tabsize)))
end)

string.center_justify = def("string", "number", "?string", "->", "string", function(s, width, fill)
        fill = fill or " "
        local pad = width - #s
        if pad <= 0 then
                return s
        end
        local left = math.floor(pad / 2)
        local right = pad - left
        return string.rep(fill, left) .. s .. string.rep(fill, right)
end)

string.left_justify = def("string", "number", "?string", "->", "string", function(s, width, fill)
        fill = fill or " "
        return s .. string.rep(fill, math.max(width - #s, 0))
end)

string.right_justify = def("string", "number", "?string", "->", "string", function(s, width, fill)
        fill = fill or " "
        return string.rep(fill, math.max(width - #s, 0)) .. s
end)

local capitalize = function(word)
        return word:sub(1, 1):upper() .. word:sub(2):lower()
end

string.to_pascal_case = def("string", "->", "string", function(s)
        local t = {}
        for word in s:gmatch("[%w]+") do
                table.insert(t, capitalize(word))
        end
        return table.concat(t)
end)

string.to_camel_case = def("string", "->", "string", function(s)
        local t = {}
        local first = true
        for word in s:gmatch("[%w]+") do
                if first then
                        table.insert(t, word:lower())
                        first = false
                else
                        table.insert(t, capitalize(word))
                end
        end
        return table.concat(t)
end)

string.to_snake_case = def("string", "->", "string", function(s)
        local t = {}
        for word in s:gmatch("[%w]+") do
                table.insert(t, word:lower())
        end
        return table.concat(t, "_")
end)

string.to_ada_case = def("string", "->", "string", function(s)
        local t = {}
        for word in s:gmatch("[%w]+") do
                table.insert(t, word:upper())
        end
        return table.concat(t, "_")
end)

string.to_kebab_case = def("string", "->", "string", function(s)
        local t = {}
        for word in s:gmatch("[%w]+") do
                table.insert(t, word:lower())
        end
        return table.concat(t, "-")
end)

string.to_upper_kebab_case = def("string", "->", "string", function(s)
        local t = {}
        for word in s:gmatch("[%w]+") do
                table.insert(t, word:upper())
        end
        return table.concat(t, "-")
end)
