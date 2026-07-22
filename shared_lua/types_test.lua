-- types_test.lua — tests the type system as an EMBEDDER sees it: required on its own, with no base,
-- no globals, and no side effects. This file must be required before base_test (which intentionally
-- installs globals), so the purity snapshot below is meaningful.
local dir = debug.getinfo(1, "S").source:match("@?(.*/)")
require(dir .. "testing") -- defines the global check() / eq_array()

-- Snapshot global names before loading types, to prove the module writes none.
local globals_before = {}
for k in pairs(_G) do
	globals_before[k] = true
end

local types = require(dir .. "types")

-- === The embedding contract: pure, global-free ===

check("requiring types.lua writes no globals", function()
	for k in pairs(_G) do
		if not globals_before[k] then
			return false
		end
	end
	return true
end)
check("types.lua installs none of base's globals", function()
	return rawget(_G, "def") == nil
		and rawget(_G, "types") == nil
		and rawget(_G, "sh") == nil
		and rawget(_G, "HOST_SYSTEM_INFO") == nil
end)
check("the module exposes the combinator API", function()
	return type(types) == "table"
		and types.is_type(types.number)
		and type(types.resolve) == "function"
		and type(types.format) == "function"
		and type(types.array_of) == "function"
		and type(types.shape) == "function"
end)

-- === check() works standalone, without def() ===

check("check returns true on a match", function()
	return types.number.check(5) == true
end)
check("check returns a fault on a mismatch", function()
	local ok, f = types.number.check("x")
	return ok == false and f.expected == "number" and f.value == "x" and type(f.path) == "table"
end)
check("the string DSL resolves without base", function()
	local T = types.resolve("?number|string")
	return T.check(nil) == true and T.check(7) == true and T.check("s") == true and (T.check({}) == false)
end)

-- === types.format: located, humanized messages ===

check("format renders a located message rooted at value", function()
	local Shape = types.shape({ decision = types.enum("allow", "block") })
	local ok, f = Shape.check({ decision = "maybe" })
	local msg = types.format(f)
	return ok == false
		and msg:find("value.decision", 1, true)
		and msg:find("expected enum", 1, true)
		and msg:find('"maybe"', 1, true)
end)
check("format defaults root to value and accepts a custom root", function()
	local ok, f = types.array_of("number").check({ 1, "x" })
	return not ok and types.format(f):find("value[2]", 1, true) and types.format(f, "arg"):find("arg[2]", 1, true)
end)
check("format tolerates a hostile __tostring on the offending value", function()
	local evil = setmetatable({}, {
		__tostring = function()
			error("boom")
		end,
	})
	local ok, f = types.number.check(evil)
	local rendered = types.format(f)
	return not ok and type(rendered) == "string" and rendered:find("tostring error", 1, true)
end)

-- === The proposal's actual use case (untrusted validator response) ===

check("the ResponseShape example validates and locates faults", function()
	local ResponseShape = types.shape({
		decision = types.enum("allow", "block", "redirect", "not_matched"),
		status_code = "?number",
		request_id = "?string",
		response_html = "?string",
		headers = types.optional(types.map_of("string", "string")),
		cookies = types.optional(types.array_of("table")),
	})
	local good = ResponseShape.check({ decision = "block", status_code = 403, headers = { ["x-id"] = "y" } })
	local bad, f = ResponseShape.check({ decision = "nope" })
	return good == true and bad == false and f.path[1] == "decision" and types.format(f):find("decision", 1, true)
end)

-- === def is part of the module and works without base ===

check("types.def wraps and checks a function standalone", function()
	local identity = types.def("number", "->", "number", function(x)
		return x
	end)
	return identity(5) == 5 and (not pcall(identity, "x"))
end)
check("types.def is the same value base re-exposes", function()
	-- base.lua only re-publishes it as a global; the definition belongs to types.
	return rawget(_G, "def") == nil or _G.def == types.def
end)
