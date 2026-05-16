#!/usr/bin/env lua

-- Execution instructions:
--
-- bash/zsh
-- eval "$(./bin/{os}/lua install_golang.lua)"
--
-- fish
-- eval (./bin/{os}/lua install_golang.lua)

setmetatable(_G, {
	__newindex = function(_, k)
		error("global write: " .. k, 2)
	end,
})

local INFO = function(msg)
	io.stderr:write("\27[34mINF\27[0m " .. msg .. "\n")
end
local ERROR = function(msg)
	io.stderr:write("\27[31mERR\27[0m " .. msg .. "\n")
end

local crash = function(msg)
	ERROR(msg)
	os.exit(1)
end

local capture = function(cmd)
	local p = io.popen(cmd)
	if not p then
		return nil
	end
	local out = p:read("*a")
	local ok = p:close()
	return out, ok
end

local run = function(cmd)
	local ok = os.execute(cmd)
	return ok == 0 or ok == true
end

local is_executable = function(path)
	return run(string.format("test -x %q", path))
end

local path_contains = function(entry)
	local PATH = os.getenv("PATH") or ""
	for p in string.gmatch(PATH, "([^:]+)") do
		if p == entry then
			return true
		end
	end
	return false
end

local emit_exports = function(install_dir, git_root)
	if path_contains(install_dir .. "/go/bin") then
		INFO("PATH already contains " .. install_dir .. "/go/bin, skipping")
	else
		io.write(string.format('export PATH="%s/go/bin:$PATH"\n', install_dir))
	end
	if path_contains(install_dir) then
		INFO("PATH already contains " .. install_dir .. ", skipping")
	else
		io.write(string.format('export PATH="%s:$PATH"\n', install_dir))
	end
	io.write(string.format('export GOROOT="%s/go"\n', install_dir))
	io.write(string.format('export GOPATH="%s/.go-path"\n', git_root))
	io.write(string.format('export GOBIN="%s/bin"\n', git_root))
	io.write('export GOFLAGS="-mod=vendor"\n')

	local github_env = os.getenv("GITHUB_ENV")
	local github_path = os.getenv("GITHUB_PATH")
	if github_env and github_env ~= "" and github_path and github_path ~= "" then
		local fp = assert(io.open(github_path, "a"))
		if not path_contains(install_dir .. "/go/bin") then
			fp:write(install_dir .. "/go/bin\n")
		end
		if not path_contains(install_dir) then
			fp:write(install_dir .. "\n")
		end
		fp:close()
		local fe = assert(io.open(github_env, "a"))
		fe:write(string.format("GOROOT=%s/go\n", install_dir))
		fe:write(string.format("GOPATH=%s/.go-path\n", git_root))
		fe:write(string.format("GOBIN=%s/bin\n", git_root))
		fe:write("GOFLAGS=-mod=vendor\n")
		fe:write("GOOS=linux\n")
		fe:write("GOARCH=amd64\n")
		fe:close()
	end
end

local ensure_gopls = function(install_dir, git_root, gopls_version)
	if os.getenv("CI") and os.getenv("CI") ~= "" then
		INFO("CI detected, skipping gopls install")
		return
	end
	local gopls_bin = install_dir .. "/gopls"
	if is_executable(gopls_bin) then
		local out = capture(string.format("%q version 2>/dev/null", gopls_bin)) or ""
		local installed
		for line in string.gmatch(out, "[^\n]+") do
			if string.find(line, "gopls", 1, true) then
				installed = string.match(line, "(%S+)%s*$")
				break
			end
		end
		if installed == gopls_version then
			INFO("gopls " .. gopls_version .. " already installed at " .. gopls_bin)
			return
		end
		INFO("gopls version mismatch: want " .. gopls_version .. ", have " .. (installed or "unknown"))
	end
	INFO("installing gopls " .. gopls_version .. "...")
	local cmd = string.format(
		'GOROOT=%q GOPATH=%q GOBIN=%q GOFLAGS="" %q install golang.org/x/tools/gopls@%s',
		install_dir .. "/go",
		git_root .. "/.go-path",
		install_dir,
		install_dir .. "/go/bin/go",
		gopls_version
	)
	if not run(cmd) then
		crash("failed to install gopls")
	end
	local ver = capture(string.format("%q version | head -n1", gopls_bin)) or ""
	INFO("installed: " .. (string.match(ver, "[^\n]+") or ""))
end

local parse_go_version = function(out)
	return string.match(out or "", "go version go(%S+)")
end

local main = function()
	local go_version = "1.25.7"
	local gopls_version = "v0.21.1"

	local detected_os = (capture("uname -s") or ""):gsub("%s+$", "")
	local go_os
	if detected_os == "Darwin" then
		go_os = "darwin"
	elseif detected_os == "Linux" then
		go_os = "linux"
	else
		crash("unsupported OS: " .. detected_os)
	end

	local detected_arch = (capture("uname -m") or ""):gsub("%s+$", "")
	local go_arch, go_sha256
	local platform = go_os .. "-" .. detected_arch
	if platform == "darwin-arm64" then
		go_arch = "arm64"
		go_sha256 = "ff18369ffad05c57d5bed888b660b31385f3c913670a83ef557cdfd98ea9ae1b"
	elseif platform == "linux-x86_64" then
		go_arch = "amd64"
		go_sha256 = "12e6d6a191091ae27dc31f6efc630e3a3b8ba409baf3573d955b196fdf086005"
	else
		crash("unsupported platform: " .. detected_os .. "/" .. detected_arch)
	end

	INFO("detected platform: " .. go_os .. "/" .. detected_arch)

	local go_tarball = string.format("go%s.%s-%s.tar.gz", go_version, go_os, go_arch)
	local go_download_url = "https://go.dev/dl/" .. go_tarball

	local script_path = arg and arg[0] or "./install_golang.lua"
	local git_root = (capture(string.format('cd "$(dirname %q)" && pwd', script_path)) or ""):gsub("%s+$", "")
	INFO("repo root: " .. git_root)

	local go_install_dir = git_root .. "/bin"
	INFO("install directory: " .. go_install_dir)
	INFO("target version: go" .. go_version)

	if is_executable(go_install_dir .. "/go/bin/go") then
		INFO("found existing Go binary at " .. go_install_dir .. "/go/bin/go")
		local out, ok = capture(string.format("%q version", go_install_dir .. "/go/bin/go"))
		if not ok then
			crash("failed to get installed Go version")
		end
		local installed = parse_go_version(out)
		INFO("installed version: " .. (installed or "unknown"))
		if installed == go_version then
			INFO("go " .. go_version .. " is already installed at " .. go_install_dir .. "/go")
			ensure_gopls(go_install_dir, git_root, gopls_version)
			emit_exports(go_install_dir, git_root)
			return
		end
		INFO("version mismatch: want " .. go_version .. ", have " .. (installed or "unknown"))
		INFO("removing " .. go_install_dir .. "/go...")
		run(string.format("rm -rf %q", go_install_dir .. "/go"))
	end

	local tmpdir = (capture("mktemp -d") or ""):gsub("%s+$", "")
	if tmpdir == "" then
		crash("failed to create temporary directory")
	end

	local tarball_path = tmpdir .. "/" .. go_tarball

	INFO("downloading " .. go_download_url .. "...")
	local curl_cmd = string.format(
		"curl --fail --show-error --location --retry 3 --retry-delay 2 --output %q %q",
		tarball_path,
		go_download_url
	)
	if not run(curl_cmd) then
		run(string.format("rm -rf %q", tmpdir))
		crash("failed to download " .. go_download_url)
	end

	INFO("download complete: " .. tarball_path)
	INFO("verifying SHA256 checksum...")
	local shasum_cmd
	if go_os == "darwin" then
		shasum_cmd = string.format("shasum -a 256 %q", tarball_path)
	else
		shasum_cmd = string.format("sha256sum %q", tarball_path)
	end
	local shasum_out, shasum_ok = capture(shasum_cmd)
	if not shasum_ok then
		run(string.format("rm -rf %q", tmpdir))
		crash("failed to compute SHA256 checksum")
	end
	local actual_sha256 = string.match(shasum_out or "", "^(%S+)")
	if actual_sha256 ~= go_sha256 then
		ERROR("checksum mismatch")
		ERROR("  expected: " .. go_sha256)
		ERROR("  actual:   " .. (actual_sha256 or "unknown"))
		run(string.format("rm -rf %q", tmpdir))
		os.exit(1)
	end
	INFO("checksum verified: " .. actual_sha256)

	if not run(string.format("mkdir -p %q", go_install_dir)) then
		run(string.format("rm -rf %q", tmpdir))
		crash("failed to create " .. go_install_dir)
	end

	INFO("extracting to " .. go_install_dir .. "/go...")
	if not run(string.format("tar --extract --gzip --file %q --directory %q", tarball_path, go_install_dir)) then
		run(string.format("rm -rf %q", tmpdir))
		crash("failed to extract tarball")
	end

	run(string.format("rm -rf %q", tmpdir))

	local installed_out, installed_ok = capture(string.format("%q version", go_install_dir .. "/go/bin/go"))
	if not installed_ok then
		crash("failed to verify installed Go")
	end
	local installed_version = parse_go_version(installed_out)
	if installed_version ~= go_version then
		ERROR("installed version does not match expected")
		ERROR("  expected: " .. go_version)
		ERROR("  actual:   " .. (installed_version or "unknown"))
		os.exit(1)
	end
	INFO("verified: " .. (string.match(installed_out or "", "[^\n]+") or ""))

	ensure_gopls(go_install_dir, git_root, gopls_version)
	emit_exports(go_install_dir, git_root)
end

main()
