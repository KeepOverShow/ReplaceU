#!/usr/bin/env python3
"""Maintain a reverse tunnel configured explicitly by the operator."""
import os,pathlib,subprocess,time,signal,fcntl
root=pathlib.Path(__file__).resolve().parent.parent
runtime=root/'.runtime/showmax'
required=('WORKBENCH_SSH_TARGET','WORKBENCH_SSH_KEY','WORKBENCH_KNOWN_HOSTS')
missing=[name for name in required if not os.environ.get(name)]
if missing:raise SystemExit('Set required environment variables: '+', '.join(missing))
target=os.environ['WORKBENCH_SSH_TARGET']
if target.startswith('-'):raise SystemExit('WORKBENCH_SSH_TARGET must be user@host')
runtime.mkdir(parents=True,exist_ok=True)
lock=open(runtime/'supervisor.lock','w')
try:fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
except BlockingIOError:raise SystemExit('Tunnel supervisor is already running')
(runtime/'supervisor.pid').write_text(str(os.getpid()))
args=['/usr/bin/ssh','-N','-T','-o','BatchMode=yes','-o','IdentitiesOnly=yes','-o','ExitOnForwardFailure=yes','-o','ServerAliveInterval=30','-o','ServerAliveCountMax=3','-o','ConnectTimeout=20','-o','StrictHostKeyChecking=yes','-o','UserKnownHostsFile='+os.environ['WORKBENCH_KNOWN_HOSTS'],'-i',os.environ['WORKBENCH_SSH_KEY'],'-R',os.environ.get('WORKBENCH_SSH_FORWARD','127.0.0.1:18787:127.0.0.1:8787'),target]
child=None;running=True
def stop(*_):
 global running
 running=False
 if child and child.poll() is None:child.terminate()
signal.signal(signal.SIGTERM,stop);signal.signal(signal.SIGINT,stop)
while running:
 print(time.strftime('%Y-%m-%d %H:%M:%S'),'connecting',flush=True)
 child=subprocess.Popen(args,stdin=subprocess.DEVNULL)
 code=child.wait()
 print(time.strftime('%Y-%m-%d %H:%M:%S'),'ssh exited',code,flush=True)
 for _ in range(30):
  if not running:break
  time.sleep(1)
