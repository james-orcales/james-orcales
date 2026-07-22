-- array_test.lua — black-box tests for the array module.
local dir = debug.getinfo(1, "S").source:match("@?(.*/)")
require(dir .. "testing") -- defines the global check() / eq_array()
local array = require(dir .. "array")

-- take: first n elements.
check("take 2 of 3", function()
	return eq_array({ 1, 2 }, array.take({ 1, 2, 3 }, 2))
end)

-- drop: skip the first n, reindexed from 1. (Ported code had an off-by-one.)
check("drop 2 of 5", function()
	return eq_array({ 30, 40, 50 }, array.drop({ 10, 20, 30, 40, 50 }, 2))
end)
check("drop 1 of 3", function()
	return eq_array({ 2, 3 }, array.drop({ 1, 2, 3 }, 1))
end)
check("drop 0 is identity", function()
	return eq_array({ 10, 20, 30 }, array.drop({ 10, 20, 30 }, 0))
end)
check("drop all is empty", function()
	return eq_array({}, array.drop({ 10, 20, 30 }, 3))
end)

-- find: index or nil.
check("find hit", function()
	return array.find({ 5, 6, 7 }, 6) == 2
end)
check("find miss", function()
	return array.find({ 5, 6, 7 }, 9) == nil
end)

-- map / any / contains / count.
check("map double", function()
	return eq_array({ 2, 4, 6 }, array.map({ 1, 2, 3 }, function(x)
		return x * 2
	end))
end)
check("any true", function()
	return array.any({ 1, 2, 3 }, function(x)
		return x == 2
	end) == true
end)
check("any false", function()
	return array.any({ 1, 2, 3 }, function(x)
		return x == 9
	end) == false
end)
check("contains true", function()
	return array.contains({ 1, 2, 3 }, 3) == true
end)
check("count evens", function()
	return array.count({ 1, 2, 3, 4 }, function(x)
		return x % 2 == 0
	end) == 2
end)

-- filter / reject: ported code called a nonexistent array.insert.
check("filter evens", function()
	return eq_array({ 2, 4 }, array.filter({ 1, 2, 3, 4, 5 }, function(x)
		return x % 2 == 0
	end))
end)
check("reject evens", function()
	return eq_array({ 1, 3, 5 }, array.reject({ 1, 2, 3, 4, 5 }, function(x)
		return x % 2 == 0
	end))
end)

-- zip: pairs up to the shorter length.
check("zip pairs", function()
	local z = array.zip({ 1, 2 }, { 10, 20 })
	return #z == 2 and z[1][1] == 1 and z[1][2] == 10 and z[2][1] == 2 and z[2][2] == 20
end)

-- enumerate: {index, value} pairs; ported code called the nonexistent array.insert.
check("enumerate pairs", function()
	local e = array.enumerate({ "a", "b" })
	return #e == 2 and e[1][1] == 1 and e[1][2] == "a" and e[2][1] == 2 and e[2][2] == "b"
end)
