#!/bin/sh
# Executable scripts pass their root; interactive use starts in the project root.
ROOT=${GWB_ROOT:-$PWD}
if [ ! -f "$ROOT/go.mod" ] || [ ! -f "$ROOT/scripts/env.sh" ]; then
  echo 'Run from the gpu-workbench directory or set GWB_ROOT to its absolute path.' >&2
  return 1
fi
export PATH="$ROOT/.tools/bin:$PATH"
export COLIMA_HOME=/tmp/gwb-vm LIMA_HOME=/tmp/gwb-vm/_lima
export KUBECONFIG="$ROOT/.runtime/kube/config"
export HELM_CACHE_HOME="$ROOT/.cache/helm/cache" HELM_CONFIG_HOME="$ROOT/.cache/helm/config" HELM_DATA_HOME="$ROOT/.cache/helm/data"
