#!/usr/bin/env sh
# Pre-flight for local linting: explains the two Go-version mismatches that
# have repeatedly been "fixed" by editing the go directive in go.mod, and names
# the actual remedy instead. Run by .husky/pre-commit and `task doctor`.
#
# The go directive is the module's minimum language version. Its three-part
# form (e.g. `go 1.26.0`) is required by dependencies and rewritten by
# `go mod tidy`; hand-editing it never fixes a tool that is too old and, at
# worst, changes how CI selects its toolchain.
set -eu

fail() {
  printf '%s\n' "error: $1" >&2
  printf '%s\n' "       $2" >&2
  printf '%s\n' "       Do not edit the go directive in go.mod to work around this." >&2
  exit 1
}

# Compare two X.Y[.Z] versions: returns 0 when $1 >= $2.
version_ge() {
  [ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}

want="$(sed -nE 's/^go ([0-9]+\.[0-9]+(\.[0-9]+)?)[[:space:]]*$/\1/p' go.mod | head -n1)"
[ -n "$want" ] || fail "could not read the go directive from go.mod" "expected a line like 'go 1.26.0'"

# 1. The local go must be able to build the module. With the default
#    GOTOOLCHAIN=auto (or any '+auto' value) the go command downloads a newer
#    toolchain by itself, so only a pinned/local setting can be too old.
toolchain="${GOTOOLCHAIN:-$(go env GOTOOLCHAIN 2>/dev/null || echo auto)}"
case "$toolchain" in
  auto|*+auto) ;;
  *)
    have="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
    if [ -n "$have" ] && ! version_ge "$have" "$want"; then
      fail "local go is $have but go.mod requires go >= $want (GOTOOLCHAIN=$toolchain)" \
           "upgrade Go (e.g. 'brew upgrade go') or unset GOTOOLCHAIN so it can auto-download the required toolchain."
    fi
    ;;
esac

# 2. golangci-lint refuses to run when the Go it was *built with* is older than
#    the go directive: "the Go language version (go1.X) used to build
#    golangci-lint is lower than the targeted Go version (1.Y)". That is a
#    stale linter binary, not a go.mod problem.
if command -v golangci-lint >/dev/null 2>&1; then
  built="$(golangci-lint version 2>/dev/null | sed -nE 's/.*built with go([0-9]+\.[0-9]+(\.[0-9]+)?).*/\1/p' | head -n1)"
  if [ -n "$built" ] && ! version_ge "$built" "$want"; then
    fail "golangci-lint was built with go$built, older than the go.mod target ($want); it will refuse to lint" \
         "upgrade the linter (e.g. 'brew upgrade golangci-lint') so it is built with go >= $want."
  fi
fi
