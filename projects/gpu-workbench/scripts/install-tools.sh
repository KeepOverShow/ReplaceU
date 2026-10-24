#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$ROOT"
[ "$(uname -s)" = Darwin ] || { echo 'This bootstrap is for macOS'; exit 1; }
case "$(uname -m)" in x86_64) ARCH=amd64; LIMA_ARCH=x86_64;; arm64) ARCH=arm64; LIMA_ARCH=aarch64;; *) exit 1;; esac
mkdir -p .tools/bin .tools/downloads .tools/prometheus .runtime/kube
if [ ! -x .tools/bin/limactl ]; then
 curl -fL --retry 2 -o .tools/downloads/lima.tar.gz "https://github.com/lima-vm/lima/releases/download/v1.2.1/lima-1.2.1-Darwin-$LIMA_ARCH.tar.gz"
 tar -xzf .tools/downloads/lima.tar.gz -C .tools
fi
if [ ! -x .tools/bin/colima ]; then
 curl -fL --retry 2 -o .tools/bin/colima "https://github.com/abiosoft/colima/releases/download/v0.9.1/colima-Darwin-$LIMA_ARCH"
 chmod +x .tools/bin/colima
fi
if [ ! -x .tools/bin/helm ]; then
 curl -fL --retry 2 -o .tools/downloads/helm.tar.gz "https://get.helm.sh/helm-v3.18.4-darwin-$ARCH.tar.gz"
 tar -xzf .tools/downloads/helm.tar.gz -C .tools/downloads
 cp ".tools/downloads/darwin-$ARCH/helm" .tools/bin/helm
fi
if [ ! -x .tools/bin/kubectl ]; then
 curl -fL --retry 2 -o .tools/bin/kubectl "https://dl.k8s.io/release/v1.31.11/bin/darwin/$ARCH/kubectl"
 chmod +x .tools/bin/kubectl
fi
if [ ! -x .tools/prometheus/prometheus ]; then
 curl -fL --retry 2 -o .tools/downloads/prometheus.tar.gz "https://github.com/prometheus/prometheus/releases/download/v3.5.0/prometheus-3.5.0.darwin-$ARCH.tar.gz"
 tar -xzf .tools/downloads/prometheus.tar.gz -C .tools/prometheus --strip-components=1
fi
echo 'Tools installed under .tools; Go and Python 3 are also required.'
