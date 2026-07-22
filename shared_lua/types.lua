-- Copyright (c) 2023-2026 James Orcales. All rights reserved.
--
-- This software is provided 'as-is', without any express or implied
-- warranty. In no event will the authors be held liable for any damages
-- arising from the use of this software.
--
-- Permission is granted to anyone to use this software for any purpose,
-- including commercial applications, and to alter it and redistribute it
-- freely, subject to the following restrictions:
--
-- 1. The origin of this software must not be misrepresented; you must not
--    claim that you wrote the original software. If you use this software
--    in a product, an acknowledgment in the product documentation would be
--    appreciated but is not required.
-- 2. Altered source versions must be plainly marked as such, and must not be
--    misrepresented as being the original software.
-- 3. This notice may not be removed or altered from any source distribution.
--
-- A "type" is a value with a check(value) -> ok, fault method. Three forms stand in for a type
-- anywhere the API expects one, so the common case stays terse and the hard case stays possible:
--   * a string token  — a base type, e.g. "number"; "?" prefixes an optional; "|" forms a union
--                        ("?string|number"); "array" means a real Lua sequence.
--   * a function      — a custom predicate, called with the value; it must return truthy to pass.
--   * a types.* value — a combinator: array_of, map_of, tuple, shape, union, optional, enum,
--                       literal, variadic. Combinators nest, and the location of a deep mismatch is
--                       carried in the fault's path (rendered by types.format as "value.a[2].b").

-- unpack moved to table.unpack in Lua 5.2; alias it so the module runs on 5.1/LuaJIT and 5.3/5.4.
local unpack = table.unpack or unpack

local types = {}

-- The metatable that marks a table as a resolved type value (vs. a shape spec or a plain table).
local Type = {}

local is_type = function(value)
        return type(value) == "table" and getmetatable(value) == Type
end
types.is_type = is_type

-- A type value pairs a human-readable name with check(value) -> ok, fault-or-nil.
local new_type = function(name, check)
        return setmetatable({ name = name, check = check }, Type)
end

-- A fault is one violation: what was expected, the offending value, and the path from the checked
-- value down to it (array index or record key at each step; empty for a direct mismatch). An
-- optional `got` field overrides how the actual side renders (used by array_of to report a count).
local fault = function(expected, value, path)
        return { expected = expected, value = value, path = path or {} }
end

-- The longest a value may be rendered in a message before it is truncated.
local VALUE_DISPLAY_MAX = 60

-- Renders a value for a message without trusting it: a hostile __tostring is contained by pcall,
-- strings are quoted, and anything long is truncated so a giant table cannot flood the output.
local display = function(value)
        if value == nil then
                return "nil"
        end
        local text
        if type(value) == "string" then
                text = string.format("%q", value)
        else
                local ok, rendered = pcall(tostring, value)
                text = ok and rendered or "<tostring error>"
        end
        if #text > VALUE_DISPLAY_MAX then
                text = text:sub(1, VALUE_DISPLAY_MAX - 3) .. "..."
        end
        return text
end

local base_type = function(lua_type)
        return new_type(lua_type, function(value)
                if type(value) == lua_type then
                        return true
                end
                return false, fault(lua_type, value)
        end)
end

types.number = base_type("number")
types.string = base_type("string")
types.boolean = base_type("boolean")
types.table = base_type("table")
types["function"] = base_type("function")
types.func = types["function"]
types.thread = base_type("thread")
types.userdata = base_type("userdata")
types.any = new_type("any", function()
        return true
end)

-- A Lua sequence: keys exactly 1..n with no nil holes. #t is unreliable once a hole exists, so
-- count the keys and verify each 1..count is present and every key is a positive integer.
local is_sequence = function(value)
        if type(value) ~= "table" then
                return false
        end
        local count = 0
        for key in pairs(value) do
                if type(key) ~= "number" then
                        return false
                end
                count = count + 1
        end
        for i = 1, count do
                if value[i] == nil then
                        return false
                end
        end
        return true
end

types.sequence = new_type("sequence", function(value)
        if is_sequence(value) then
                return true
        end
        return false, fault("sequence", value)
end)

-- The base-type tokens the string DSL understands. "array" is a real sequence check, not a bare
-- table, so the name stops lying about what it accepts.
local STRING_TYPES = {
        any = types.any,
        number = types.number,
        string = types.string,
        boolean = types.boolean,
        table = types.table,
        array = types.sequence,
        sequence = types.sequence,
        ["function"] = types["function"],
        thread = types.thread,
        userdata = types.userdata,
}

-- Forward declaration: the string parser, the combinators, and resolve() reference one another.
local resolve

local parse_string_type = function(spec)
        local optional = spec:sub(1, 1) == "?"
        if optional then
                spec = spec:sub(2)
        end
        local alternatives = {}
        for token in (spec .. "|"):gmatch("([^|]+)|") do
                local base = STRING_TYPES[token]
                if not base then
                        error("types: unknown type token '" .. token .. "'", 0)
                end
                alternatives[#alternatives + 1] = base
        end
        if #alternatives == 0 then
                error("types: empty type string", 0)
        end
        local result = #alternatives == 1 and alternatives[1] or types.union(unpack(alternatives))
        if optional then
                result = types.optional(result)
        end
        return result
end

-- Normalizes a type expression (string | function | type value) into a type value.
resolve = function(spec)
        if is_type(spec) then
                return spec
        end
        if type(spec) == "string" then
                return parse_string_type(spec)
        end
        if type(spec) == "function" then
                return new_type("predicate", function(value)
                        if spec(value) then
                                return true
                        end
                        return false, fault("predicate", value)
                end)
        end
        error("types: a type must be a string, function, or types.* value, got " .. type(spec), 0)
end
types.resolve = resolve

types.optional = function(spec)
        local inner = resolve(spec)
        return new_type("optional(" .. inner.name .. ")", function(value)
                if value == nil then
                        return true
                end
                return inner.check(value)
        end)
end

types.union = function(...)
        local members, names = {}, {}
        for i = 1, select("#", ...) do
                members[i] = resolve((select(i, ...)))
                names[i] = members[i].name
        end
        local name = table.concat(names, "|")
        return new_type(name, function(value)
                for _, member in ipairs(members) do
                        if member.check(value) then
                                return true
                        end
                end
                return false, fault(name, value)
        end)
end

-- array_of(T) is a dynamic slice ([]T): a sequence of any length whose every element is T.
-- array_of(T, n) is a fixed array ([n]T): a sequence of exactly n elements of T.
types.array_of = function(spec, length)
        if length ~= nil and (type(length) ~= "number" or length < 0 or length % 1 ~= 0) then
                error("array_of: length must be a non-negative integer, got " .. display(length), 2)
        end
        local element = resolve(spec)
        local name = "array_of(" .. element.name .. ")"
        if length ~= nil then
                name = name .. " of length " .. length
        end
        return new_type(name, function(value)
                if not is_sequence(value) then
                        return false, fault(name, value)
                end
                if length ~= nil and #value ~= length then
                        -- Report the actual element count, not the whole table, so a length
                        -- mismatch reads "expected ... of length 3, got length 2".
                        local f = fault(name, value)
                        f.got = "length " .. #value
                        return false, f
                end
                for i = 1, #value do
                        local ok, f = element.check(value[i])
                        if not ok then
                                table.insert(f.path, 1, i)
                                return false, f
                        end
                end
                return true
        end)
end

types.map_of = function(key_spec, value_spec)
        local key_type, value_type = resolve(key_spec), resolve(value_spec)
        local name = "map_of(" .. key_type.name .. ", " .. value_type.name .. ")"
        return new_type(name, function(value)
                if type(value) ~= "table" then
                        return false, fault(name, value)
                end
                for k, v in pairs(value) do
                        local ok, f = key_type.check(k)
                        if not ok then
                                table.insert(f.path, 1, k)
                                return false, f
                        end
                        ok, f = value_type.check(v)
                        if not ok then
                                table.insert(f.path, 1, k)
                                return false, f
                        end
                end
                return true
        end)
end

types.tuple = function(...)
        local elements, names = {}, {}
        for i = 1, select("#", ...) do
                elements[i] = resolve((select(i, ...)))
                names[i] = elements[i].name
        end
        local name = "tuple(" .. table.concat(names, ", ") .. ")"
        return new_type(name, function(value)
                if not is_sequence(value) or #value ~= #elements then
                        return false, fault(name .. " of length " .. #elements, value)
                end
                for i = 1, #elements do
                        local ok, f = elements[i].check(value[i])
                        if not ok then
                                table.insert(f.path, 1, i)
                                return false, f
                        end
                end
                return true
        end)
end

types.shape = function(fields)
        local resolved = {}
        for key, spec in pairs(fields) do
                resolved[key] = resolve(spec)
        end
        return new_type("shape", function(value)
                if type(value) ~= "table" then
                        return false, fault("shape (table)", value)
                end
                for key, field_type in pairs(resolved) do
                        local ok, f = field_type.check(value[key])
                        if not ok then
                                table.insert(f.path, 1, key)
                                return false, f
                        end
                end
                return true
        end)
end

types.enum = function(...)
        local allowed, labels = {}, {}
        for i = 1, select("#", ...) do
                local value = select(i, ...)
                allowed[value] = true
                labels[i] = display(value)
        end
        local name = "enum(" .. table.concat(labels, ", ") .. ")"
        return new_type(name, function(value)
                if value ~= nil and allowed[value] then
                        return true
                end
                return false, fault(name, value)
        end)
end

types.literal = function(expected)
        local name = "literal(" .. display(expected) .. ")"
        return new_type(name, function(value)
                if value == expected then
                        return true
                end
                return false, fault(name, value)
        end)
end

-- Marks the final input slot as variadic: zero or more trailing arguments, each of element type.
-- def() consumes it specially; its own check validates a single value so it stays a usable type.
types.variadic = function(spec)
        local element = resolve(spec)
        local marker = new_type("variadic(" .. element.name .. ")", element.check)
        marker.variadic = true
        marker.element = element
        return marker
end

-- Appends a fault's descent path to a root label: render_path("argument #1", {"port"}) is
-- "argument #1.port"; an identifier key becomes ".key", anything else "[<value>]".
local render_path = function(root, path)
        local out = root
        for _, step in ipairs(path) do
                if type(step) == "string" and step:match("^[%a_][%w_]*$") then
                        out = out .. "." .. step
                else
                        out = out .. "[" .. display(step) .. "]"
                end
        end
        return out
end

-- Renders a fault into a located, humanized string. `root` names the checked value (default
-- "value"); def passes "argument #N" / "return #N". Carries the offending value and its runtime
-- type, not merely the expected type name.
types.format = function(f, root)
        local got = f.got or (f.value == nil and "nil" or (type(f.value) .. " " .. display(f.value)))
        return render_path(root or "value", f.path) .. " expected " .. f.expected .. ", got " .. got
end

-- def() is the function-signature face of the type system: it wraps a function so its argument and
-- return types are checked on every call. Types take any of the three forms above. Optionality is
-- per slot: an optional input may be absent and an optional return may be nil. A violation raises a
-- located error blaming the caller (catchable with pcall). It writes no globals and adds no cost
-- unless called, so consumers that only validate values can ignore it and call check() directly.
--
-- Usage:
--      atoi = types.def("?string", "->", "number", function(str)
--              return tonumber(str or "69")
--      end)
--      atoi("69")   --> 69
--      atoi({})     --> error: def: argument #1 expected string, got table table: 0x...
--
-- Set the global LUA_DISABLE_FUNCTION_SIGNATURE_ASSERTIONS truthy to return the raw function.
types.def = function(...)
        local slot_count = select("#", ...)
        local slots = { ... }
        local callback = slots[slot_count]
        if type(callback) ~= "function" then
                error("def: last argument must be the function, got " .. type(callback), 2)
        end
        if LUA_DISABLE_FUNCTION_SIGNATURE_ASSERTIONS then
                return callback
        end

        -- Split the type slots at "->" into inputs and outputs, resolving each to a type value.
        local inputs, outputs = {}, {}
        local target, seen_arrow = inputs, false
        for i = 1, slot_count - 1 do
                if slots[i] == "->" then
                        if seen_arrow then
                                error("def: a signature has at most one ->", 2)
                        end
                        seen_arrow, target = true, outputs
                else
                        target[#target + 1] = resolve(slots[i])
                end
        end
        if #inputs == 0 and #outputs == 0 then
                error("def: declare at least one input or output type", 2)
        end

        -- A variadic marker may only be the last input; it validates every trailing argument.
        local last = #inputs
        local rest = inputs[last]
        if rest and rest.variadic then
                inputs[last] = nil
        else
                rest = nil
        end

        return function(...)
                local argc = select("#", ...)
                local args = { ... }
                for i = 1, #inputs do
                        local ok, f = inputs[i].check(args[i])
                        if not ok then
                                error("def: " .. types.format(f, "argument #" .. i), 2)
                        end
                end
                if rest then
                        for i = #inputs + 1, argc do
                                local ok, f = rest.element.check(args[i])
                                if not ok then
                                        error("def: " .. types.format(f, "argument #" .. i), 2)
                                end
                        end
                elseif argc > #inputs then
                        error(string.format("def: expected at most %d argument(s), got %d", #inputs, argc), 2)
                end

                -- Call once, capturing the true count so an explicit nil return is not miscounted.
                local collect = function(...)
                        return select("#", ...), { ... }
                end
                local resultc, results = collect(callback(unpack(args, 1, argc)))
                if resultc > #outputs then
                        error(string.format("def: expected at most %d return value(s), got %d", #outputs, resultc), 2)
                end
                for i = 1, #outputs do
                        local ok, f = outputs[i].check(results[i])
                        if not ok then
                                error("def: " .. types.format(f, "return #" .. i), 2)
                        end
                end
                return unpack(results, 1, resultc)
        end
end

return types
