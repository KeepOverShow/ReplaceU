#!/bin/sh
set -eu
GWB_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
. "$GWB_ROOT/scripts/env.sh"
LAB_ROOT=$(CDPATH= cd -- "$ROOT/../../.." && pwd)
mkdir -p "$LAB_ROOT/.vm"
if [ -e /tmp/gwb-vm ] && [ ! -L /tmp/gwb-vm ]; then echo "Unexpected /tmp/gwb-vm directory"; exit 1; fi
ln -sfn "$LAB_ROOT/.vm" /tmp/gwb-vm
exec colima start gwb --runtime containerd --vm-type vz --cpu 4 --memory 6 --disk 30 --kubernetes --kubernetes-version v1.31.11+k3s1 --activate=false --ssh-config=false --mount "$ROOT" --k3s-arg=--disable=traefik,servicelb
