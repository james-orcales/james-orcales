DEBUG_ENABLED = false
-- DEBUG_ENABLED = true


-- Install here instead of ./bin/ so that we don't have to set ODIN_ROOT
odin_exe = "vendor/Odin/odin"


function DEBUG(msg, caller_location) 
        if not DEBUG_ENABLED then 
                return
        end 
        caller_location = (caller_location or 0) + 2
        local line_number = debug.getinfo(caller_location, "l").currentline
        print(string.format("DEBUG:%s %s", line_number, msg))
end
function INFO (...) print("INFO  " .. table.concat({...}, " ")) end
function WARN (...) print("WARN  " .. table.concat({...}, " ")) end
function ERROR(...) print("ERROR " .. table.concat({...}, " ")) end
function assert(cond, msg)
        caller_location = (caller_location or 0) + 2
        local line_number = debug.getinfo(caller_location, "l").currentline
        if not cond then
                msg = msg or ""
                print(string.format("ASSERTION FAILED:%s | %s", line_number, msg))
                os.exit(1)
        end
end


--- Executes a shell command and emulates shell piping.
-- If the last argument is a pipe symbol `"|"`, it returns the command's string output.
-- Otherwise, it returns a success boolean.
function sh(...)
        local n = select('#', ...)
        if select(n, ...) == '|' then
                local command = table.concat({...}, " ", 1, #{...}-1)
                local handle = io.popen(command)
                local output = handle:read("*a"):match("^(.-)\n?$")
                DEBUG("output: "..output, 1)
                handle:close()
                return output
        else
                local command = table.concat({...}, " ")
                return os.execute(command) == 0
        end
end


function checkhealth_odin(silent)
        if sh('command -v > /dev/null', odin_exe) then
                if sh(odin_exe, 'version', '|') == odin_exe .. " version dev-2025-10:3ad7240" then
                        return true
                else 
                        if not silent then INFO("wrong odin version") end
                end
        end
end


function checkhealth_homebrew(silent)
        if sh('command -v brew > /dev/null') then
                if not silent then INFO("homebrew is already installed") end
                return true
        end
end


function install_homebrew()
        if checkhealth_homebrew() then
                return
        end
        sh([[NONINTERACTIVE=1 /bin/bash -c "$(curl --fail --silent --show-error --location https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"]])
        sh([[brew install llvm@20]])
        assert(checkhealth_homebrew())
end


-- build odin from source to bypass Apple authorization issues
function install_odin()
        if checkhealth_odin() then
                return
        end
        local dir_exists = not sh("mkdir vendor >/dev/null 2>/dev/null") 
        if dir_exists then
                sh("rm -rf vendor/Odin")
        end
        if sh([[
                pushd vendor
                git clone --depth=1 --branch=dev-2025-10 https://github.com/odin-lang/Odin.git/
                pushd Odin
                make release-native
                popd
                popd
        ]])
        then
                assert(checkhealth_odin)
        end
end


function main()
        local cmd = {
                setup = function() 
                        install_homebrew()
                        install_odin()
                end,
                check = function()
                        if not checkhealth_odin() then
                                return
                        end
                        sh(odin_exe, "check", table.concat(arg, " ", 2), "-strict-style -disallow-do -vet-cast -vet-unused -vet-using-param -vet-using-stmt")
                end,
                odin = function()
                        if not checkhealth_odin() then
                                return
                        end
                        sh(odin_exe, table.concat(arg, " ", 2))
                end,
        }
        cmd[arg[1] or "setup"]()
end


main()
