-- testing.lua — a tiny plain-Lua test harness (Lua 5.1), no external framework.
--
-- The helpers are GLOBAL and this file is idempotent (guarded by `if not check`). That is
-- deliberate: base.lua's relative-require idiom yields a different require-name for a module
-- depending on the load path ("shared_lua/testing" vs "./shared_lua/testing"), so a returned-table
-- singleton would be duplicated and the tally split. Globals — the idiom base.lua already uses —
-- are shared no matter which name loaded this file, and the guard keeps the counters from resetting
-- when a second require re-runs the chunk.
--
-- check() takes a THUNK, not a pre-evaluated condition: pcall turns a genuine Lua error (e.g.
-- calling a nil field) into a reported FAIL instead of aborting the run. It cannot catch
-- base.assert's os.exit(1) — that hard-kills the process by design — so tests exercise the positive
-- path and never deliberately trip a library assertion.
if not check then
	local passed, failed = 0, 0

	check = function(name, thunk)
		local ok, result = pcall(thunk)
		if ok and result then
			passed = passed + 1
		else
			local why = ok and "returned false/nil" or ("error: " .. tostring(result))
			failed = failed + 1
			io.stderr:write("FAIL: " .. tostring(name) .. " (" .. why .. ")\n")
		end
	end

	-- Element-wise equality for the flat 1-based arrays the array/strings helpers return.
	eq_array = function(a, b)
		if type(a) ~= "table" or type(b) ~= "table" then
			return false
		end
		if #a ~= #b then
			return false
		end
		for i = 1, #a do
			if a[i] ~= b[i] then
				return false
			end
		end
		return true
	end

	tests_finish = function()
		print(string.format("shared_lua: %d passed, %d failed", passed, failed))
		if failed > 0 then
			os.exit(1)
		end
	end
end
