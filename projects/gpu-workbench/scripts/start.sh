#!/bin/sh
set -eu
GWB_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
. "$GWB_ROOT/scripts/env.sh"
cd "$ROOT"
MODE=${1:-k8s-sim}
if [ ! -x .runtime/workbench ]; then scripts/go-local.sh build -o .runtime/workbench ./cmd/workbench; fi
export WORKBENCH_START_MODE="$MODE"
python3 - <<'PY'
import subprocess,os,time,json,urllib.request
urllib.request.install_opener(urllib.request.build_opener(urllib.request.ProxyHandler({})))
mode=os.environ['WORKBENCH_START_MODE'];args=['.runtime/workbench','-mode',mode,'-data','.runtime']
if mode!='local-demo':args+=['-kubeconfig',os.environ['KUBECONFIG']]
# Reject an already occupied port before launching a second process.
import socket
with socket.socket() as probe:
 probe.settimeout(1)
 if probe.connect_ex(('127.0.0.1',8787))==0:raise SystemExit('Port 8787 is already in use; stop the existing service first')
with open('.runtime/server.log','a') as log:
 p=subprocess.Popen(args,stdout=log,stderr=log,stdin=subprocess.DEVNULL,start_new_session=True)
open('.runtime/server.pid','w').write(str(p.pid))
for i in range(60):
 if p.poll() is not None:raise SystemExit('Server exited; read .runtime/server.log')
 try:
  with urllib.request.urlopen('http://127.0.0.1:8787/healthz',timeout=1) as r:
   if r.status==200:
    open('.runtime/server.pid','w').write(str(p.pid));print('Running: http://127.0.0.1:8787 · '+mode);break
 except Exception:time.sleep(.5)
else:raise SystemExit('Server did not become ready; inspect logs')
PY
