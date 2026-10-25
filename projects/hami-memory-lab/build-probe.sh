#!/usr/bin/env bash
set -euo pipefail
: "${CUDA_HOME:?Set CUDA_HOME to the matching CUDA toolkit}"
cd "$(dirname "$0")"
mkdir -p .runtime
"${CC:-cc}" -std=c11 -Wall -Wextra -I "$CUDA_HOME/include" probe.c -lcuda -o .runtime/probe
printf '%s\n' 'Built .runtime/probe. This is not a GPU validation result.'
