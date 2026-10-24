#!/bin/sh
set -eu
GWB_ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
. "$GWB_ROOT/scripts/env.sh"
cd "$ROOT"
python3 - <<'PY'
import os,signal,subprocess
try:pid=int(open('.runtime/server.pid').read())
except FileNotFoundError:raise SystemExit('No saved server PID')
cmd=subprocess.run(['ps','-p',str(pid),'-o','command='],capture_output=True,text=True).stdout
if '.runtime/workbench' not in cmd:raise SystemExit('PID no longer belongs to this server; nothing stopped')
os.kill(pid,signal.SIGTERM);print('Stop requested; task state remains in SQLite')
PY
