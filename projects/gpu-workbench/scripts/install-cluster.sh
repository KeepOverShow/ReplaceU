#!/bin/sh
set -eu
GWB_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
. "$GWB_ROOT/scripts/env.sh"
cd "$ROOT"
# Dedicated local kubeconfig. Never fall back to a user's production context.
[ -f "$KUBECONFIG" ] || { echo 'Run scripts/vm-start.sh first'; exit 1; }
kubectl get node colima-gwb >/dev/null
helm repo add koordinator https://koordinator-sh.github.io/charts/
helm repo update koordinator
kubectl create namespace koordinator-system --dry-run=client -o yaml | kubectl apply -f -
kubectl label namespace koordinator-system app.kubernetes.io/managed-by=Helm --overwrite
kubectl annotate namespace koordinator-system meta.helm.sh/release-name=koordinator meta.helm.sh/release-namespace=koordinator-system --overwrite
helm upgrade --install koordinator koordinator/koordinator --version 1.7.0 -n koordinator-system -f deploy/koordinator-sim-values.yaml --post-renderer scripts/helm-post-render.sh --wait --timeout 5m
kubectl apply -f deploy/namespace.yaml
kubectl label node colima-gwb workbench.local/lab=true --overwrite
python3 scripts/sim-devices.py
kubectl -n gpu-workbench create configmap workbench-worker --from-file=worker.py=workloads/worker.py --from-file=online.py=workloads/online.py --dry-run=client -o yaml | kubectl apply -f -
