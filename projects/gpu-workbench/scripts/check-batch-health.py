#!/usr/bin/env python3
"""Verify BE admission and healthy-device placement against the local lab."""
import json,os,subprocess,time,urllib.request,uuid
op=urllib.request.build_opener(urllib.request.ProxyHandler({}))
token=json.load(open('.runtime/credentials.json'))['alpha'];created=[]
report={'mode':'k8s-sim','real_cuda_verified':False,'checks':[]};run=uuid.uuid4().hex[:8]
os.environ['PATH']=os.path.abspath('.tools/bin')+os.pathsep+os.environ['PATH']
os.environ['KUBECONFIG']=os.path.abspath('.runtime/kube/config')
def api(path,body=None):
 req=urllib.request.Request('http://127.0.0.1:8787'+path,data=json.dumps(body).encode() if body is not None else None,headers={'Authorization':'Bearer '+token,'Content-Type':'application/json','Idempotency-Key':str(uuid.uuid4())})
 with op.open(req,timeout=20) as r:return json.load(r)
def wait(t,states):
 end=time.monotonic()+150
 while time.monotonic()<end:
  v=api('/api/tasks/'+t)
  if v['task']['state'] in states:return v
  if v['task']['state'] in ['Failed','Succeeded','Cancelled']:raise AssertionError(v)
  time.sleep(1)
 raise AssertionError('timed out '+t)
def submit(profile):
 t=api('/api/tasks',dict(name='健康与BE验证-'+profile+'-'+run,memory_mib=512,cores=10,duration_seconds=60,workload='matmul',profile=profile))['task']['id'];created.append(t);return t
try:
 assert api('/api/quota')['active_tasks']==0,'finish other tasks first'
 t=submit('batch');v=wait(t,['Running'])
 pods=json.loads(subprocess.check_output(['kubectl','-n','gpu-workbench','get','pods','-l','workbench.local/task='+t,'-o','json']))['items']
 pod=pods[0];resources=pod['spec']['containers'][0]['resources'];assert 'kubernetes.io/batch-cpu' in resources['requests'];assert pod['spec']['priorityClassName']=='koord-batch'
 report['checks'].append(dict(check='BE admission and execution',passed=True,resources=resources,allocation=v['observation']['allocation']))
 api('/api/tasks/'+t+'/cancel',{});wait(t,['Cancelled'])
 subprocess.run(['python3','scripts/sim-devices.py','--unhealthy','0'],check=True)
 # Wait for the manager to reflect only one healthy GPU in node capacity.
 for _ in range(60):
  node=json.loads(subprocess.check_output(['kubectl','get','node','colima-gwb','-o','json']))
  if node['status']['allocatable'].get('koordinator.sh/gpu-core')=='100':break
  time.sleep(1)
 else:raise AssertionError('healthy capacity was not reconciled')
 t=submit('standard');v=wait(t,['Running']);minor=v['observation']['allocation']['gpu'][0]['minor'];assert minor==1
 report['checks'].append(dict(check='Unhealthy device excluded',passed=True,minor=minor))
 api('/api/tasks/'+t+'/cancel',{});wait(t,['Cancelled']);report['passed']=True
except Exception as e:
 report['passed']=False;report['error']=str(e);raise
finally:
 for t in created:
  try:
   if api('/api/tasks/'+t)['task']['state'] not in ['Succeeded','Failed','Cancelled']:api('/api/tasks/'+t+'/cancel',{})
  except Exception:pass
 subprocess.run(['python3','scripts/sim-devices.py'],check=True)
 path='.runtime/evidence/batch-health-'+run+'.json';json.dump(report,open(path,'w'),ensure_ascii=False,indent=2);print(path,flush=True)
