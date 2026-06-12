#!/usr/bin/env dash

# bash/zsh: eval "$(./install-golang.sh)"
# fish:     eval (dash ./install-golang.sh)

set +eu

# Send all subprocess chatter (make.bash, tar, go install) to stderr by default;
# the emit_* helpers write the eval payload to FD 3 instead. Otherwise build
# noise pollutes stdout and `eval "$(./install-golang.sh)"` chokes on it.
exec 3>&1 1>&2

INFO()  { printf '\033[34m[INFO]\033[0m %s\n'  "$*" >&2; }
FATAL() { printf '\033[31m[ERROR]\033[0m %s\n' "$*" >&2; exit 1; }

# Emit "export NAME=VALUE" to the real stdout (for eval) and "NAME=VALUE" to GITHUB_ENV.
emit_var() {
        printf 'export %s="%s"\n' "$1" "$2" >&3
        test -n "${GITHUB_ENV:-}" && echo "$1=$2" >> "${GITHUB_ENV}"
}

# Prepend $1 to PATH on the real stdout (eval) and to GITHUB_PATH. Skip if already on PATH.
emit_path() {
        case ":${PATH}:" in *":$1:"*) INFO "PATH already contains $1, skipping"; return ;; esac
        printf 'export PATH="%s:$PATH"\n' "$1" >&3
        test -n "${GITHUB_PATH:-}" && echo "$1" >> "${GITHUB_PATH}"
}

# Parse "X.Y.Z" out of "go version goX.Y.Z os/arch".
go_version() { "$1" version 2>/dev/null | awk '{sub(/^go/,"",$3); print $3}'; }

# Verify $1 is a tarball whose SHA-256 matches "$1.sum".
verify_sha() {
        test -f "$1.sum" || FATAL "missing checksum file: $1.sum"
        expected=$(cat "$1.sum")
        case "$(uname -s)" in
                Darwin) actual=$(shasum -a 256 "$1" | awk '{print $1}') ;;
                Linux)  actual=$(sha256sum "$1"    | awk '{print $1}') ;;
        esac
        if test "${actual}" != "${expected}"; then
                FATAL "checksum mismatch for $1: expected ${expected}, got ${actual}"
        fi
}

case "$(uname -s)-$(uname -m)" in
        Darwin-arm64) GO_OS=darwin GO_ARCH=arm64 ;;
        Linux-x86_64) GO_OS=linux  GO_ARCH=amd64 ;;
        *) FATAL "unsupported platform: $(uname -s)/$(uname -m)" ;;
esac

GIT_ROOT="$(cd "$(dirname "$0")" && pwd)"
GO_SRC_DIR="${GIT_ROOT}/third_party/go"
GO_BIN="${GO_SRC_DIR}/bin/go"
GOROOT="${GO_SRC_DIR}"
GOPATH="${GIT_ROOT}/.local/share/go"
GOBIN="${GIT_ROOT}/.local/bin"

TARGET_VERSION=$(sed -n '1s/^go//p' "${GO_SRC_DIR}/VERSION") || FATAL "failed to read ${GO_SRC_DIR}/VERSION"

BOOTSTRAP_VERSION="1.26.3"
BOOTSTRAP_TARBALL="${GIT_ROOT}/third_party/go${BOOTSTRAP_VERSION}.${GO_OS}-${GO_ARCH}.tar.gz"
BOOTSTRAP_CACHE="${GIT_ROOT}/.local/share/go_bootstrap"
BOOTSTRAP_ROOT="${BOOTSTRAP_CACHE}/go"

GOPLS_VERSION="v0.21.1"

INFO "platform=${GO_OS}/${GO_ARCH} target=go${TARGET_VERSION} bootstrap=go${BOOTSTRAP_VERSION}"

if test "$(go_version "${GO_BIN}")" != "${TARGET_VERSION}"; then
        if test "$(go_version "${BOOTSTRAP_ROOT}/bin/go")" != "${BOOTSTRAP_VERSION}"; then
                test -f "${BOOTSTRAP_TARBALL}" || FATAL "missing bootstrap tarball: ${BOOTSTRAP_TARBALL}"
                verify_sha "${BOOTSTRAP_TARBALL}"
                rm -rf "${BOOTSTRAP_ROOT}"
                mkdir -p "${BOOTSTRAP_CACHE}"
                INFO "extracting bootstrap go ${BOOTSTRAP_VERSION}..."
                tar -xzf "${BOOTSTRAP_TARBALL}" -C "${BOOTSTRAP_CACHE}" || FATAL "extract failed"
        fi
        INFO "building go ${TARGET_VERSION} from ${GO_SRC_DIR}/src..."
        rm -rf "${GO_SRC_DIR}/bin" "${GO_SRC_DIR}/pkg"
        (
                cd "${GO_SRC_DIR}/src" \
                        && GOROOT_BOOTSTRAP="${BOOTSTRAP_ROOT}" GOROOT_FINAL="${GO_SRC_DIR}" GOTOOLCHAIN=local \
                                ./make.bash
        ) || FATAL "make.bash failed"
        test "$(go_version "${GO_BIN}")" = "${TARGET_VERSION}" || FATAL "built go does not report ${TARGET_VERSION}"
fi
INFO "go ready: $("${GO_BIN}" version)"

# go builds into GOROOT/bin, which is not on PATH; expose it (and gofmt) from the
# one directory that is — GOBIN — so `which go` resolves there.
if test "$(which go 2>/dev/null)" != "${GOBIN}/go"; then
        INFO "linking go into ${GOBIN}..."
        mkdir -p "${GOBIN}"
        ln -sf "${GO_BIN}" "${GOBIN}/go"
        ln -sf "${GOROOT}/bin/gofmt" "${GOBIN}/gofmt"
fi

if test -z "${CI:-}"; then
        GOPLS_BIN="${GOBIN}/gopls"
        INSTALLED=$("${GOPLS_BIN}" version 2>/dev/null | awk '/gopls/ {print $NF}')
        if test "${INSTALLED}" != "${GOPLS_VERSION}" || test "$(which gopls 2>/dev/null)" != "${GOPLS_BIN}"; then
                INFO "installing gopls ${GOPLS_VERSION}..."
                GOROOT="${GOROOT}" GOPATH="${GOPATH}" GOBIN="${GOBIN}" GOFLAGS="" GOTOOLCHAIN=local \
                        "${GO_BIN}" install "golang.org/x/tools/gopls@${GOPLS_VERSION}" \
                        || FATAL "gopls install failed"
        fi
        INFO "gopls ready: $("${GOPLS_BIN}" version | head -n1)"
fi

emit_path "${GOBIN}"
emit_var  GOROOT      "${GOROOT}"
emit_var  GOPATH      "${GOPATH}"
emit_var  GOBIN       "${GOBIN}"
emit_var  GOFLAGS     "-mod=vendor"
emit_var  GOTOOLCHAIN "local"

# Pin GOOS/GOARCH in CI so cross-build targets are explicit. Locally, leave unset
# so `go` picks the host.
if test -n "${GITHUB_ENV:-}"; then
        echo "GOOS=${GO_OS}"     >> "${GITHUB_ENV}"
        echo "GOARCH=${GO_ARCH}" >> "${GITHUB_ENV}"
fi

