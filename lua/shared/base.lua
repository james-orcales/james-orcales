-- Module "base" layers CLI conveniences (globals, os/shell, logging) over the dependency-free
-- `types` module, which it requires. It defines globals and is imported by other lua-core modules.
assert(_VERSION == "Lua 5.1", "lua-core will only support lua-5.1.5")

local dir = debug.getinfo(1, "S").source:match("@?(.*/)")
-- Expose the type system and its def() decorator as globals so base's callers reach them without a
-- second require. def lives in `types` (it is part of the type system); base only re-publishes it.
-- The types module writes no globals and has no side effects, so it stays safe to embed on its own.
types = require(dir .. "types")
def = types.def

-- === ASSERTIONS ===

assert = function(cond, msg)
        local msg = msg or "<empty>"
        if not cond then
                print(debug.traceback("", 2))
                print("assertion failed: " .. msg)
                os.exit(1)
        end
end

sometimes = def("boolean", "string", function(cond, msg)
        -- noop
end)

unreachable = def("boolean", "string", function(cond, msg)
        local msg = msg or "<empty>"
        print(debug.traceback("", 2))
        print("reached unreachable code: " .. msg)
        os.exit(1)
end)

unimplemented = def("boolean", "string", function(cond, msg)
        local msg = msg or "<empty>"
        print(debug.traceback("", 2))
        print("reached unimplemented code: " .. msg)
        os.exit(1)
end)

-- === SYSTEM ===

--- Executes a shell command and emulates shell piping.
-- If the last argument is a pipe symbol `"|"`, it returns the command's string output.
-- Otherwise, it returns a success boolean.
sh = function(...)
        local n = select("#", ...)
        assert(n > 0)
        if select(n, ...) == "|" then
                local command = table.concat({ ... }, " ", 1, #{ ... } - 1)
                local handle = io.popen(command)
                local output = handle:read("*a"):match("^(.-)\n?$")
                handle:close()
                return output
        else
                local command = table.concat({ ... }, " ")
                return os.execute(command) == 0
        end
end

HOST_SYSTEM_INFO = {
        operating_system = sh("uname", "|"),
        cpu_architecture = sh("uname -m", "|"),
}

-- === LOGGING ===

eprint = function(...)
        io.stderr:write(table.concat({ ... }, " "), "\n")
end

fatal = function(...)
        eprint(...)
        os.exit(1)
end

fatalf = function(...)
        io.stderr:write(string.format(...) .. "\n")
        os.exit(1)
end

-- stylua: ignore start
DEBUG = function(msg, caller_location)
        if not DEBUG_ENABLED then
                return
        end
        caller_location = (caller_location or 0) + 2
        local line_number = debug.getinfo(caller_location, "l").currentline
        print(string.format("DEBUG:%s %s", line_number, msg))
end
INFO  = function(...) print("INFO  " .. table.concat({...}, " ")) end
WARN  = function(...) print("WARN  " .. table.concat({...}, " ")) end
ERROR = function(...) print("ERROR " .. table.concat({...}, " ")) end
-- stylua: ignore end
