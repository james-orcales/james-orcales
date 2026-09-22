-- base_test.lua — tests for the base prelude: the def() type system, plus sh and HOST_SYSTEM_INFO.
-- base defines its helpers (def, types, sh, ...) as globals, so requiring it makes them available.
-- Violations now raise a catchable error (not os.exit), so negative cases use pcall.
local dir = debug.getinfo(1, "S").source:match("@?(.*/)")
require(dir .. "testing") -- defines the global check() / eq_array()
require(dir .. "base") -- defines def, types, sh, HOST_SYSTEM_INFO, logging globals

-- Builds a def whose inputs are the given type specs and whose body just returns true, so a test
-- can focus on the input signature: typed(good) == true, and pcall(typed, bad) fails.
local accept = function(...)
	local signature = { ... }
	signature[#signature + 1] = "->"
	signature[#signature + 1] = "boolean"
	signature[#signature + 1] = function()
		return true
	end
	return def(unpack(signature))
end

-- === def: happy path (base types, optionals, unions, arity) ===

check("def identity round-trips a typed value", function()
	local identity = def("number", "->", "number", function(x)
		return x
	end)
	return identity(5) == 5
end)
check("def optional input may be absent", function()
	local atoi = def("?string", "->", "number", function(str)
		return tonumber(str or "69")
	end)
	return atoi("42") == 42 and atoi() == 69
end)
check("def any accepts every type", function()
	local id = def("any", "->", "boolean", function(v)
		return v ~= nil
	end)
	return id(1) == true and id("s") == true and id({}) == true
end)
check("def optional return may be nil", function()
	local maybe = def("boolean", "->", "?number", function(present)
		if present then
			return 7
		end
		return nil
	end)
	return maybe(true) == 7 and maybe(false) == nil
end)
check("def checks multiple return values positionally", function()
	local split = def("string", "->", "string", "boolean", function(s)
		return s, #s > 0
	end)
	local a, b = split("hi")
	return a == "hi" and b == true
end)

-- === Diagnostics: index, expected, actual type, offending value ===

check("violation names index, expected type, actual type, and value", function()
	local typed = def("number", "->", "number", function(x)
		return x
	end)
	local ok, err = pcall(typed, "eighty")
	return not ok
		and err:find("argument #1", 1, true)
		and err:find("expected number", 1, true)
		and err:find("got string", 1, true)
		and err:find('"eighty"', 1, true)
end)
check("return violation is labelled as a return", function()
	local lies = def("boolean", "->", "number", function()
		return "not a number"
	end)
	local ok, err = pcall(lies, true)
	return not ok and err:find("return #1", 1, true) and err:find("expected number", 1, true)
end)
check("nil for a required slot reports got nil", function()
	local typed = accept("number")
	local ok, err = pcall(typed, nil)
	return not ok and err:find("got nil", 1, true)
end)

-- === Arity ===

check("too many arguments is rejected", function()
	return not pcall(accept("number"), 1, 2)
end)
check("too many return values is rejected", function()
	local two = def("boolean", "->", "number", function()
		return 1, 2
	end)
	return not pcall(two, true)
end)
check("an explicit trailing nil past arity is rejected", function()
	-- Guards the deliberate strictness: select("#") preserves the trailing nil, so argc exceeds
	-- the declared arity and the call is refused rather than silently dropping the slot.
	local typed = accept("number")
	return not pcall(typed, 1, nil)
end)

-- === Combinators: union ===

check("union accepts any member, rejects outsiders", function()
	local typed = accept("number|string")
	return typed(1) == true and typed("x") == true and (not pcall(typed, {}))
end)

-- === Combinators: sequence via the "array" token ===

check("array token requires a real sequence", function()
	local typed = accept("array")
	return typed({ 1, 2, 3 }) == true and typed({}) == true and (not pcall(typed, { x = 1 }))
end)

-- === Combinators: array_of with element path ===

check("array_of checks elements and locates the failure", function()
	local typed = accept(types.array_of("number"))
	if not (typed({ 1, 2, 3 }) == true) then
		return false
	end
	local ok, err = pcall(typed, { 1, 2, "x", 4 })
	return not ok and err:find("argument #1[3]", 1, true) and err:find("expected number", 1, true)
end)

-- === Combinators: array_of with a fixed length ([N]T) ===

check("array_of without a length stays a dynamic slice", function()
	local typed = accept(types.array_of("number"))
	return typed({ 1, 2, 3, 4, 5 }) == true and typed({}) == true
end)
check("array_of with a length accepts the exact count", function()
	local typed = accept(types.array_of("number", 3))
	return typed({ 1, 2, 3 }) == true
end)
check("array_of with a length rejects the wrong count", function()
	local typed = accept(types.array_of("number", 3))
	local ok, err = pcall(typed, { 1, 2 })
	return not ok and err:find("length 3", 1, true) and err:find("got length 2", 1, true)
end)
check("array_of with a length still checks elements", function()
	local typed = accept(types.array_of("number", 3))
	local ok, err = pcall(typed, { 1, "x", 3 })
	return not ok and err:find("argument #1[2]", 1, true) and err:find("expected number", 1, true)
end)
check("array_of with length 0 is the empty array", function()
	local typed = accept(types.array_of("string", 0))
	return typed({}) == true and (not pcall(typed, { "x" }))
end)
check("array_of rejects a non-integer length at construction", function()
	return not pcall(types.array_of, "number", -1)
end)

-- === Combinators: map_of ===

check("map_of checks keys and values", function()
	local typed = accept(types.map_of("string", "number"))
	return typed({ a = 1, b = 2 }) == true and (not pcall(typed, { a = "x" }))
end)

-- === Combinators: tuple ===

check("tuple checks fixed-arity heterogeneous sequences", function()
	local typed = accept(types.tuple("number", "string"))
	return typed({ 1, "x" }) == true
		and (not pcall(typed, { 1, 2 })) -- second is not a string
		and (not pcall(typed, { 1 })) -- wrong length
end)

-- === Combinators: shape with required and optional fields, named path ===

check("shape validates named fields including optionals", function()
	local typed = accept(types.shape({ host = "string", port = "number", tag = types.optional("string") }))
	return typed({ host = "h", port = 80 }) == true and typed({ host = "h", port = 80, tag = "x" }) == true
end)
check("shape reports the failing field by name", function()
	local typed = accept(types.shape({ port = "number" }))
	local ok, err = pcall(typed, { port = "eighty" })
	return not ok and err:find("argument #1.port", 1, true)
end)

-- === Combinators: enum and literal ===

check("enum accepts only listed values", function()
	local typed = accept(types.enum("r", "g", "b"))
	return typed("g") == true and (not pcall(typed, "x"))
end)
check("literal accepts only the exact value", function()
	local typed = accept(types.literal(42))
	return typed(42) == true and (not pcall(typed, 43))
end)

-- === Custom predicate functions ===

check("a predicate function is a custom type", function()
	local positive = function(x)
		return type(x) == "number" and x > 0
	end
	local typed = accept(positive)
	return typed(5) == true and (not pcall(typed, -1)) and (not pcall(typed, "x"))
end)

-- === Variadic input ===

check("variadic checks every trailing argument", function()
	local typed = accept("string", types.variadic("number"))
	return typed("a", 1, 2, 3) == true -- all trailing are numbers
		and typed("a") == true -- zero trailing is fine
		and (not pcall(typed, "a", 1, "x")) -- third argument is not a number
end)
check("variadic is rejected in a non-last input slot", function()
	local ok, err = pcall(def, types.variadic("number"), "string", "->", "boolean", function()
		return true
	end)
	return not ok and err:find("variadic", 1, true) and err:find("last input", 1, true)
end)
check("variadic is rejected in an output slot", function()
	local ok, err = pcall(def, "string", "->", types.variadic("number"), function()
		return 1
	end)
	return not ok and err:find("variadic", 1, true) and err:find("last input", 1, true)
end)

-- === Nested combinators keep a deep path ===

check("nested combinators report a full descent path", function()
	local typed = accept(types.array_of(types.shape({ id = "number" })))
	local ok, err = pcall(typed, { { id = 1 }, { id = "bad" } })
	return not ok and err:find("argument #1[2].id", 1, true)
end)

-- === Production toggle ===

check("disabling signatures returns the raw, unchecked function", function()
	types.disable_signatures(true)
	local raw = def("number", "->", "number", function(x)
		return x
	end)
	local unchecked = raw("not a number") == "not a number"
	types.disable_signatures(false)
	return unchecked
end)

-- === sh and host info (unchanged base behavior) ===

check("sh captures stdout", function()
	return sh("echo", "hi", "|") == "hi"
end)
check("sh reports success", function()
	return sh("true") == true
end)
check("sh reports failure", function()
	return sh("false") == false
end)
check("host system info populated", function()
	return type(HOST_SYSTEM_INFO.operating_system) == "string"
		and #HOST_SYSTEM_INFO.operating_system > 0
		and type(HOST_SYSTEM_INFO.cpu_architecture) == "string"
		and #HOST_SYSTEM_INFO.cpu_architecture > 0
end)
