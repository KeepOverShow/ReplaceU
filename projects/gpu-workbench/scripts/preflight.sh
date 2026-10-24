#!/bin/sh
set -eu
GWB_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
. "$GWB_ROOT/scripts/env.sh"
cd "$ROOT"
MODE=${1:-k8s-sim}
kubectl --request-timeout=10s get nodes
kubectl -n koordinator-system get pods
kubectl get devices
kubectl get nodemetrics,nodeslos
kubectl -n gpu-workbench auth can-i create jobs
if [ "$MODE" = k8s-sim ]; then
  kubectl get device colima-gwb -o jsonpath='{.metadata.labels.workbench\.local/simulated}'
  echo '\nGPU inventory is synthetic. Actual GPU usage is unknown.'
else
  echo 'Real GPU readiness also requires CUDA, NRI, library mapping and limit tests; these are not verified by this script.'
fi
