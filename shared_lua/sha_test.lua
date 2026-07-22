-- sha_test.lua — smoke tests for the vendored sha module (verbatim upstream), checked against
-- well-known published digests.
local dir = debug.getinfo(1, "S").source:match("@?(.*/)")
require(dir .. "testing") -- defines the global check() / eq_array()
local sha = require(dir .. "sha")

check("md5 empty", function()
	return sha.md5("") == "d41d8cd98f00b204e9800998ecf8427e"
end)

check("sha1 abc", function()
	return sha.sha1("abc") == "a9993e364706816aba3e25717850c26c9cd0d89d"
end)

check("sha256 empty", function()
	return sha.sha256("") == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
end)

check("sha256 abc", function()
	return sha.sha256("abc") == "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
end)

check("sha512 abc", function()
	return sha.sha512("abc")
		== "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a"
			.. "2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f"
end)
