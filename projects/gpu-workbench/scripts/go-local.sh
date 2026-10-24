#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"
if [ -n "${WORKBENCH_GO:-}" ]; then GO=$WORKBENCH_GO
else GO=$(command -v go); fi
export GOTOOLCHAIN=local CGO_ENABLED=1
export GOCACHE="${GOCACHE:-$ROOT/.cache/build}" GOMODCACHE="${GOMODCACHE:-$ROOT/.cache/mod}"
case "$(uname -s)" in Darwin) export GOOS=darwin;; Linux) export GOOS=linux;; *) echo 'Unsupported host';exit 1;; esac
case "$(uname -m)" in x86_64) export GOARCH=amd64;; arm64|aarch64) export GOARCH=arm64;; *) echo 'Unsupported architecture';exit 1;; esac
exec "$GO" "$@"
