-- zlib License
--
-- Copyright (c) 2025 Danzig James Orcales. All rights reserved.
--
-- This software is provided 'as-is', without any express or implied
-- warranty. In no event will the authors be held liable for any damages
-- arising from the use of this software.
--
-- Permission is granted to anyone to use this software for any purpose,
-- including commercial applications, and to alter it and redistribute it
-- freely, subject to the following restrictions:
--
-- 1. The origin of this software must not be misrepresented; you must not
--    claim that you wrote the original software. If you use this software
--    in a product, an acknowledgment in the product documentation would be
--    appreciated but is not required.
-- 2. Altered source versions must be plainly marked as such, and must not be
--    misrepresented as being the original software.
-- 3. This notice may not be removed or altered from any source distribution.

assert(_VERSION == "Lua 5.1")


-- stylua: ignore start
HOME = assert(os.getenv("XDG_CONFIG_HOME") or os.getenv("HOME"))
HOME = HOME .. "/"
BIG_BANG_GIT_DIR  = os.getenv("BIG_BANG_GIT_DIR")
BIG_BANG_DATA_DIR = os.getenv("BIG_BANG_DATA_DIR")
BIG_BANG_SHARE    = os.getenv("BIG_BANG_SHARE")
BIG_BANG_BIN      = os.getenv("BIG_BANG_BIN")
BIG_BANG_MAN      = os.getenv("BIG_BANG_MAN")
BIG_BANG_TMP      = os.getenv("BIG_BANG_TMP")
CARGO_HOME        = os.getenv("CARGO_HOME")
RUSTUP_HOME       = os.getenv("RUSTUP_HOME")
-- stylua: ignore end

-- === HELPER FUNCTIONS ===

function unreachable()
	local info = debug.getinfo(2)
	print(string.format("%s(%d): reached unreachable code", info.source, info.currentline))
	os.exit(1)
end

function unimplemented()
	local info = debug.getinfo(2)
	print(string.format("%s(%d): reached unimplemented code", info.source, info.currentline))
	os.exit(1)
end


-- stylua: ignore start
function INFO(fmt,  ...) print(string.format("%s|INFO |" .. fmt, os.date("!%Y-%m-%dT%H:%M:%SZ"), ...)) end
function WARN(fmt,  ...) print(string.format("%s|WARN |" .. fmt, os.date("!%Y-%m-%dT%H:%M:%SZ"), ...)) end
function ERROR(fmt, ...) print(string.format("%s|ERROR|" .. fmt, os.date("!%Y-%m-%dT%H:%M:%SZ"), ...)) end
function DEBUG(fmt, ...) print(string.format("%s|DEBUG|" .. fmt, os.date("!%Y-%m-%dT%H:%M:%SZ"), ...)) end
-- stylua: ignore end

CURRENT_PROCESS_ENVIRONMENT = (function()
	local env = {}
	local cmd = io.popen("env")
	local output = cmd:read("*a")
	cmd:close()
	for k, v in output:gmatch("([%w_]+)=([^\n]+)") do
		env[k] = v
	end
	return env
end)()
-- NEVER mutate this
ORIGINAL_PROCESS_ENVIRONMENT = (function()
	local copy = {}
	for k, v in pairs(CURRENT_PROCESS_ENVIRONMENT) do
		assert(type(v) ~= "table", "any nested table will make this whole table a shared reference")
		assert(type(v) == "string")
		copy[k] = v
	end
	return copy
end)()
function os.getenv(key)
	assert(type(key) == "string" and key ~= "")
	return CURRENT_PROCESS_ENVIRONMENT[key]
end

-- Run shell commands with pipe or exec semantics
function sh(...)
	local n = select("#", ...)
	assert(n > 0)

	local diff = {}
	for k, v in pairs(CURRENT_PROCESS_ENVIRONMENT) do
		if ORIGINAL_PROCESS_ENVIRONMENT[k] ~= v then
			table.insert(diff, string.format("%s='%s'", k, v:gsub("'", "'")))
		end
	end
	local prefix = (#diff > 0) and table.concat(diff, " ") .. ";" or ""

	local args = { ... }
	local is_piped = args[n] == "|"
	if is_piped then
		args[n] = nil
	end
	local command = prefix .. table.concat(args, " ")

	if is_piped then
		local handle = io.popen(command)
		local output = handle:read("*a")
		handle:close()
		return (output:gsub("\n$", ""))
	else
		return os.execute(command) == 0
	end
end

-- sourcing implies numerous possible side effects. this only cares about env variables
function source(filepath)
	assert(type(filepath) == "string" and filepath ~= "")
	INFO("sourcing %s", filepath)
	assert(operating_system == "Darwin")
	local cmd = string.format("zsh -c 'source %q; env'", filepath)
	for k, v in sh(cmd, "|"):gmatch("([%w_]+)=([^\n]+)") do
		CURRENT_PROCESS_ENVIRONMENT[k] = v
	end
end

function with_file(path, mode, fn, ...)
	assert(type(path) == "string" and path ~= "")
	assert(type(mode) == "string" and mode ~= "")
	assert(type(fn) == "function")

	local handle, open_err = io.open(path, mode)
	if not handle then
		ERROR("opening %s: %s", path, open_err)
		return false, nil
	end
	local results = { pcall(fn, handle, ...) }
	handle:close()

	local ok = results[1]
	if not ok then
		local err = results[2]
		ERROR("executing callback on %s: %s", path, err)
		return false, nil
	end
	return true, unpack(results, 2)
end

function read_file(path)
	local _, content = with_file(path, "r", function(handle)
		return handle:read("*a")
	end)
	assert(content == nil or type(content) == "string")
	return content
end

function write_file(path, content)
	assert(type(path) == "string" and path ~= "")
	assert(type(content) == "string" and content ~= "")
	local _, content = with_file(path, "w", function(handle)
		handle:write(content)
	end)
end

function string.has_prefix(str, prefix)
	assert(type(str) == "string")
	assert(type(prefix) == "string")
	return str:sub(1, #prefix) == prefix
end

function path(...)
	assert(type(HOME) == "string")
	local final, _ = table.concat({ ... }, "/"):gsub("/+", "/")
	assert(not final:match("%s"), "shame on you for using paths with spaces")
	return final
end

-- === END OF HELPER FUNCTIONS ====

-- === PREREQUISITE ===

operating_system = sh("uname", "|")
cpu_architecture = sh("uname -m", "|")
-- Anchor every cwd-dependent operation to the directory of this script so it
-- can be invoked from anywhere (e.g. `./bin/darwin/lua big_bang/bootstrap.lua`
-- from the repo root). Lua 5.1 has no chdir, so we resolve once and pass it
-- explicitly to the few callers that need it.
SCRIPT_DIR = sh(string.format("cd %q && pwd", arg[0]:match("(.*)/") or "."), "|")
assert(
	sh("basename " .. SCRIPT_DIR, "|") == "big_bang"
		and sh("git -C " .. SCRIPT_DIR .. " rev-parse --is-inside-work-tree 2>/dev/null", "|") == "true",
	"Script lives inside the cloned repository"
)
assert(
	(operating_system == "Darwin" and cpu_architecture == "arm64")
		or (
			operating_system == "Linux"
			and cpu_architecture == "x86_64"
			and string.match(read_file("/etc/os-release"), "^ID=debian")
		),
	"System is supported"
)
assert(
	(operating_system == "Darwin" and string.match(sh("ls -l /private/var/select/sh", "|"), "/bin/bash"))
		or (operating_system == "Linux" and string.match(sh("ls -l /bin/sh", "|"), "/bin/dash")),
	"POSIX shell is the default"
)
assert(
	(operating_system == "Darwin" and string.match(os.getenv("SHELL"), "/bin/zsh"))
		or (operating_system == "Linux" and string.match(os.getenv("SHELL"), "/bin/bash")),
	"Interactive shell is the default"
)

-- === END OF PREREQUISITES ===

-- "Why are you hardcoding this here?"
--      The shell config is essential to this bootstrapping so its better to keep its context inside this file.
SHELL_CONFIG = {
	{
		path(HOME, ".zshenv"),
		[[
                ]],
	},
	{
		path(HOME, ".zprofile"),
		[[ 
                ]],
	},
	{
		path(HOME, ".zshrc"),
		[[
                ]],
	},
}

function env_setup()
	INFO("Environment setup")
	if operating_system == "Darwin" then
		for _, config in ipairs(SHELL_CONFIG) do
			assert(#config == 2)
			local path, expect = unpack(config)
			local actual = read_file(path)
			if actual ~= expect then
				INFO("updating " .. path)
				write_file(path, expect)
				source(path)
			end
		end
	elseif operating_system == "Linux" then
		assert(false, "unsupported")
	end
	assert(sh([[ mkdir -p "$BIG_BANG_DATA_DIR" ]]), "Essential directories are created")
	assert(sh([[ mkdir -p "$BIG_BANG_SHARE"    ]]), "Essential directories are created")
	assert(sh([[ mkdir -p "$BIG_BANG_BIN"      ]]), "Essential directories are created")
	assert(sh([[ mkdir -p "$BIG_BANG_TMP"      ]]), "Essential directories are created")
	assert(sh([[ mkdir -p "$BIG_BANG_MAN"      ]]), "Essential directories are created")
	return true
end

-- SSH Keys
-- https://docs.github.com/en/authentication/connecting-to-github-with-ssh/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent?platform=mac
-- TODO: The repo was cloned with https if this script was executed for the very first time. Reassign the origin to the ssh url.
function setup_ssh()
	local config = [[
Host github.com
  AddKeysToAgent yes
  UseKeychain    yes
  IdentityFile   ~/.ssh/id_ed25519
]]
	-- Github only
	local known_hosts = [[
github.com ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl
github.com ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBEmKSENjQEezOmxkZMy7opKgwFB9nkt5YRrYMjNuG5N87uRgg6CLrbo5wAdT/y6v0mKV0U2w0WZ2YB/++Tpockg=
github.com ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQCj7ndNxQowgcQnjshcLrqPEiiphnt+VTTvDP6mHBL9j1aNUkY4Ue1gvwnGLVlOhGeYrnZaMgRK6+PKCUXaDbC7qtbW8gIkhL7aGCsOr/C56SJMy/BCZfxd1nWzAOxSDPgVsmerOBYfNqltV9/hWCqBywINIR+5dIg6JTJ72pcEpEjcYgXkE2YEFXV1JHnsKgbLWNlhScqb2UmyRkQyytRLtL+38TGxkxCflmO+5Z8CSSNY7GidjMIZ7Q4zMjA2n1nGrlTDkzwDCsw+wqFPGQA179cnfGWOWRVruj16z6XyvxvjJwbz0wQZ75XK5tKSb7FNyeIEs4TT4jk+S4dhPeAUC5y+bDYirYgM4GC7uEnztnZyaVWQ7B381AK4Qdrwt51ZqExKbQpTUNn+EjqoTwvqNj4kqx5QUCI0ThS/YkOxJCXmPUWZbhjpCg56i+2aB6CmK2JGhn57K5mj0MNdBXA4/WnwH6XoPWJzK5Nyu2zB3nAZp+S5hpQs+p1vN1/wsjk=
]]
	INFO("SSH setup")
	if read_file(path(HOME, ".ssh/config")) ~= config then
		write_file(path(HOME, ".ssh/config"), config)
	end
	if read_file(path(HOME, ".ssh/known_hosts")) ~= known_hosts then
		write_file(path(HOME, ".ssh/known_hosts"), known_hosts)
	end
	if read_file(path(HOME, ".ssh/id_ed25519")) then
		INFO("Private key already exists")
		return true
	else
		INFO("Generating ssh key: id_ed25519")
		if not sh([[ssh-keygen -t ed25519 -C "dja.orcales@gmail.com"]]) then
			ERROR("failed to generate ssh key")
			return false
		end
		-- TODO: Execute this to enable ssh keys in the current shell immediately.
		-- We can print then `eval "$(bin/lua bootstrap.lua)"` but we'd have to ensure that nothing else is mixed in with stdout.
		-- eval "$(ssh-agent -s)"
		INFO("Execute this to enable the ssh agent in the current shell: %s", [[eval "$(ssh-agent -s)"]])
		sh("ssh-add --apple-use-keychain $HOME/.ssh/id_ed25519")
		sh("pbcopy < $HOME/.ssh/id_ed25519.pub")
		INFO(read_file(path(HOME, ".ssh/id_ed25519.pub")))
		INFO("$HOME/.ssh/id_ed25519.pub has been copied to the clipboard.")
		INFO("Go to https://github.com/settings/keys and add your new key. Press [ENTER] when done.")
		io.read()
		return true
	end
	unreachable()
end

function main()
	setup_ssh()
	assert(env_setup(), "Environment setup is essential")

	-- Delegate Go installation to the workspace-root per-repo installer
	-- (~/code/james-orcales/install_golang.lua). It downloads Go into
	-- <workspace>/bin/go and emits `export` lines on stdout describing
	-- PATH/GOROOT/GOPATH; we merge those into CURRENT_PROCESS_ENVIRONMENT
	-- so the subsequent `go run` resolves correctly via sh()'s env diff.
	local workspace_root = sh(string.format("cd %q/.. && pwd", SCRIPT_DIR), "|")
	local lua_bin = path(workspace_root, "/bin/darwin/lua")
	local installer = path(workspace_root, "/install_golang.lua")
	local exports = sh(string.format("%q %q", lua_bin, installer), "|")
	if exports == nil or exports == "" then
		ERROR("install_golang.lua produced no exports")
		os.exit(1)
	end
	for k, v in exports:gmatch('export%s+([%w_]+)="?([^"\n]+)"?') do
		CURRENT_PROCESS_ENVIRONMENT[k] = v
	end

	print([[=== go run big_bang.go ===]])
	sh("cd " .. SCRIPT_DIR .. " && go run big_bang.go")
	print("=== Bootstrap Finished ===")
end

main()
