-- strings_test.lua — black-box tests for the strings module.
-- strings extends the global `string` table, so methods are callable as ("x"):method(...).
local dir = debug.getinfo(1, "S").source:match("@?(.*/)")
require(dir .. "testing") -- defines the global check() / eq_array()
require(dir .. "strings")

-- cut: split on the first literal separator; third value reports whether it was found.
check("cut found", function()
	local left, right, found = ("key=value"):cut("=")
	return left == "key" and right == "value" and found == true
end)
check("cut not found", function()
	local left, right, found = ("nosep"):cut("=")
	return left == "nosep" and right == "" and found == false
end)

-- split: ported code inserted into an undefined global `t`.
check("split comma", function()
	return eq_array({ "a", "b", "c" }, ("a,b,c"):split(","))
end)
check("split default space", function()
	return eq_array({ "a", "b", "c" }, ("a b c"):split())
end)
check("split trailing empty", function()
	return eq_array({ "a", "" }, ("a,"):split(","))
end)

-- count: ported code called a nonexistent :substr and had a broken guard.
check("count single", function()
	return ("banana"):count("a") == 3
end)
check("count multi non-overlapping", function()
	return ("banana"):count("na") == 2
end)
check("count non-overlapping aa", function()
	return ("aaaa"):count("aa") == 2
end)
check("count absent", function()
	return ("abc"):count("z") == 0
end)

-- trim family (the real trim is the pattern-class one; the duplicate broken copy is removed).
check("trim default whitespace", function()
	return ("  hi  "):trim() == "hi"
end)
check("trim chars", function()
	return ("xxhixx"):trim("x") == "hi"
end)
check("trim_space", function()
	return ("\t hi \n"):trim_space() == "hi"
end)
check("trim_left", function()
	return ("xxhi"):trim_left("x") == "hi"
end)
check("trim_right", function()
	return ("hixx"):trim_right("x") == "hi"
end)

-- prefix / suffix / contains / common_prefix.
check("has_prefix", function()
	return ("foobar"):has_prefix("foo") == true and ("foobar"):has_prefix("bar") == false
end)
check("has_suffix", function()
	return ("foobar"):has_suffix("bar") == true
end)
check("contains", function()
	return ("hello"):contains("ell") == true and ("hello"):contains("z") == false
end)
check("common_prefix", function()
	return ("flower"):common_prefix("flow") == "flow"
end)

-- lines: a newline terminates a line; a trailing newline does not add an empty final line
-- (matching the lines_iter companion). Empty input yields no lines.
check("lines no trailing newline", function()
	return eq_array({ "a", "b", "c" }, ("a\nb\nc"):lines())
end)
check("lines trailing newline", function()
	return eq_array({ "a", "b" }, ("a\nb\n"):lines())
end)
check("lines normalizes crlf", function()
	return eq_array({ "a", "b" }, ("a\r\nb"):lines())
end)
check("lines empty is empty", function()
	return eq_array({}, (""):lines())
end)
check("lines agrees with lines_iter", function()
	local iter = {}
	for line in ("p\nq\n\nr"):lines_iter() do
		table.insert(iter, line)
	end
	return eq_array(iter, ("p\nq\n\nr"):lines())
end)

-- justify.
check("left_justify", function()
	return ("hi"):left_justify(4) == "hi  "
end)
check("right_justify", function()
	return ("hi"):right_justify(4) == "  hi"
end)
check("center_justify", function()
	return ("hi"):center_justify(6) == "  hi  "
end)

-- case converters.
check("to_snake_case", function()
	return ("hello world"):to_snake_case() == "hello_world"
end)
check("to_ada_case", function()
	return ("hello world"):to_ada_case() == "HELLO_WORLD"
end)
check("to_pascal_case", function()
	return ("hello world"):to_pascal_case() == "HelloWorld"
end)
check("to_camel_case", function()
	return ("hello world"):to_camel_case() == "helloWorld"
end)
check("to_kebab_case", function()
	return ("hello world"):to_kebab_case() == "hello-world"
end)
