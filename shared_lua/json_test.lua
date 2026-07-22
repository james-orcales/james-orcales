-- json_test.lua — smoke tests for the vendored json module (verbatim upstream, so round-trips
-- rather than exhaustive coverage).
local dir = debug.getinfo(1, "S").source:match("@?(.*/)")
require(dir .. "testing") -- defines the global check() / eq_array()
local json = require(dir .. "json")

check("stringify string", function()
	return json.stringify("hi") == '"hi"'
end)

check("array round-trips", function()
	local back = json.parse(json.stringify({ 10, 20, 30 }))
	return eq_array({ 10, 20, 30 }, back)
end)

check("object parses", function()
	local o = json.parse('{"a":1,"b":2}')
	return o.a == 1 and o.b == 2
end)

check("nested parses", function()
	local t = json.parse('{"nums":[1,2,3],"ok":true}')
	return t.ok == true and eq_array({ 1, 2, 3 }, t.nums)
end)

check("null is the sentinel", function()
	return json.parse("null") == json.null
end)

check("object round-trips", function()
	local back = json.parse(json.stringify({ name = "x", n = 5 }))
	return back.name == "x" and back.n == 5
end)
