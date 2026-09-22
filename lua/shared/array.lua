local info = debug.getinfo(1, "S")
local dir = info.source:match("@?(.*/)")
require(dir .. "base")

local array = {}

-- array.take returns the first n elements
array.take = def("array", "number", "->", "array", function(self, n)
        assert(n <= #self)
        local result = {}
        for i = 1, n do
                result[i] = self[i]
        end
        return result
end)

-- array.drop skips the first n elements
array.drop = def("array", "number", "->", "array", function(self, n)
        assert(n <= #self)
        local result = {}
        for i = n + 1, #self do
                result[#result + 1] = self[i]
        end
        return result
end)

-- array.find returns the index of target
array.find = def("array", "any", "->", "?number", function(self, target)
        for i, v in ipairs(self) do
                if v == target then
                        return i
                end
        end
        return nil
end)

-- array.map applies func to every element of the array
array.map = def("array", "function", "->", "array", function(self, func)
        local result = {}
        for i, v in ipairs(self) do
                result[i] = func(v)
        end
        return result
end)

-- array.any applies func to every element of the array and returns immediately upon evaluating to true
array.any = def("array", "function", "->", "boolean", function(self, func)
        for _, v in ipairs(self) do
                if func(v) then
                        return true
                end
        end
        return false
end)

-- array.contains checks whether expect is in the array
array.contains = function(self, expect)
        return array.any(self, function(actual)
                return actual == expect
        end)
end

array.each = def("array", "function", function(self, func)
        for _, v in ipairs(self) do
                func(v)
        end
end)

array.filter = def("array", "function", "->", "array", function(self, func)
        local result = {}
        for _, v in ipairs(self) do
                if func(v) then
                        table.insert(result, v)
                end
        end
        return result
end)

array.count = function(self, func)
        return #array.filter(self, func)
end

array.reject = def("array", "function", "->", "array", function(self, func)
        local result = {}
        for _, v in ipairs(self) do
                if not func(v) then
                        table.insert(result, v)
                end
        end
        return result
end)

array.zip = def("array", "array", "->", "array", function(a, b)
        local result = {}
        local n = math.min(#a, #b)
        for i = 1, n do
                result[i] = { a[i], b[i] }
        end
        return result
end)

array.enumerate = def("array", "->", "array", function(self)
        local result = {}
        for i, v in ipairs(self) do
                table.insert(result, { i, v })
        end
        return result
end)

return array
