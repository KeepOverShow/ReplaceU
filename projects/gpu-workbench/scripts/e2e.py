#!/usr/bin/env python3
"""End-to-end evidence from the running local lab, with no fake pass results."""
import json,time,uuid,urllib.request,urllib.error,os,subprocess
op=urllib.request.build_opener(urllib.request.ProxyHandler({}));base='http://127.0.0.1:8787'
tokens=json.load(open('.runtime/credentials.json'));run=uuid.uuid4().hex[:8];created=[];evidence={'run':run,'mode':'k8s-sim','real_cuda_verified':False,'checks':[],'tasks':{}}
def api(path,body=None,team='alpha',key=None):
 h={'Authorization':'Bearer '+tokens[team],'Content-Type':'application/json'}
 if key:h['Idempotency-Key']=key
 req=urllib.request.Request(base+path,data=None if body is None else json.dumps(body).encode(),headers=h)
 with op.open(req,timeout=18) as r:return json.load(r)
def record(name,detail):
 evidence['checks'].append({'check':name,'passed':True,'detail':detail});print('PASS',name,flush=True)
def submit(name,memory=1024,cores=25,duration=20,workload='matmul',profile='standard',key=None):
 req=dict(name=name,memory_mib=memory,cores=cores,duration_seconds=duration,workload=workload,profile=profile)
 result=api('/api/tasks',req,key=key or run+'-'+uuid.uuid4().hex);t=result['task'];created.append(t['id']);return t,req

def wait(id,states,timeout=150):
 until=time.monotonic()+timeout;last=None
 while time.monotonic()<until:
  last=api('/api/tasks/'+id)
  if last['task']['state'] in states:return last
  if last['task']['state'] in ('Failed','Succeeded','Cancelled'):raise AssertionError('unexpected terminal '+json.dumps(last,ensure_ascii=False))
  time.sleep(1)
 raise AssertionError('timeout '+json.dumps(last,ensure_ascii=False))
def cancel(id):api('/api/tasks/'+id+'/cancel',{})
def save_task(id):
 evidence['tasks'][id]=api('/api/tasks/'+id)
 evidence['tasks'][id]['events']=api('/api/tasks/'+id+'/events')['events']
 try:evidence['tasks'][id]['logs']=api('/api/tasks/'+id+'/logs')['logs']
 except Exception as e:evidence['tasks'][id]['logs_error']=str(e)
try:
 info=api('/api/info');assert info['mode']=='k8s-sim';assert api('/api/quota')['active_tasks']==0,'finish or cancel existing tasks first'
 task,req=submit('闭环成功-'+run,duration=8,key=run+'-idem');again=api('/api/tasks',req,key=run+'-idem');assert again['replayed'] and again['task']['id']==task['id'];record('幂等提交',task['id'])
 try:api('/api/tasks/'+task['id'],team='beta');raise AssertionError('cross team read accepted')
 except urllib.error.HTTPError as e:assert e.code==404
 record('团队读取隔离','beta receives 404')
 done=wait(task['id'],['Succeeded']);save_task(task['id']);assert done['observation']['allocation'];assert '"real_cuda_validation": false' in evidence['tasks'][task['id']]['logs'];record('真实 Job 成功及分配注解',done['task']['job_uid'])
 task,_=submit('模拟显存超限-'+run,workload='memory');wait(task['id'],['Failed']);save_task(task['id']);assert 'simulated_limit_rejected' in evidence['tasks'][task['id']]['logs'];record('模拟显存超限拒绝',42)
 # Pin both large jobs to different GPUs by requesting > half of each card.
 a,_=submit('碎片-A-'+run,memory=5120,cores=60,duration=180);b,_=submit('碎片-B-'+run,memory=5120,cores=60,duration=180)
 oa=wait(a['id'],['Running']);ob=wait(b['id'],['Running']);ma=oa['observation']['allocation']['gpu'][0]['minor'];mb=ob['observation']['allocation']['gpu'][0]['minor'];assert ma!=mb
 c,_=submit('碎片等待-'+run,memory=4096,cores=25,duration=12);pending=wait(c['id'],['Pending']);time.sleep(4);pending=api('/api/tasks/'+c['id']);assert pending['task']['state']=='Pending';record('每卡碎片阻塞',pending['task']['reason']);save_task(c['id'])
 cancel(a['id']);wait(a['id'],['Cancelled']);wait(c['id'],['Running','Succeeded']);record('取消后重新调度',c['id']);cancel(b['id']);wait(b['id'],['Cancelled']);wait(c['id'],['Succeeded']);save_task(a['id']);save_task(b['id']);save_task(c['id'])
 # Five simultaneous shares across two GPUs prove that at least one card is shared.
 shares=[submit('共享-%d-'%i+run,memory=512,cores=10,duration=120)[0] for i in range(5)]
 obs=[wait(t['id'],['Running']) for t in shares];minors=[o['observation']['allocation']['gpu'][0]['minor'] for o in obs];assert len(set(minors))<len(minors);record('同卡共享调度',minors)
 for t in shares:cancel(t['id'])
 for t in shares:wait(t['id'],['Cancelled']);save_task(t['id'])
 q=api('/api/quota');assert q['active_tasks']==q['used_memory_mib']==q['used_cores']==0;record('取消后的配额回收',q)
 # A real restart must retain the execution UID and must not create an extra Job.
 task,_=submit('重启恢复-'+run,duration=120);before=wait(task['id'],['Running']);uid=before['task']['job_uid'];subprocess.run(['scripts/stop.sh'],check=True)
 for i in range(30):
  try:op.open(base+'/healthz',timeout=1);time.sleep(.5)
  except Exception:break
 subprocess.run(['scripts/start.sh','k8s-sim'],check=True);after=wait(task['id'],['Running']);assert after['task']['job_uid']==uid;assert api('/api/quota')['active_tasks']==1
 record('进程重启恢复同一 Job',uid);cancel(task['id']);wait(task['id'],['Cancelled']);save_task(task['id'])
 evidence['passed']=True
except Exception as e:
 evidence['passed']=False;evidence['error']=str(e);raise
finally:
 for id in created:
  try:
   t=api('/api/tasks/'+id)['task']
   if t['state'] not in ('Succeeded','Failed','Cancelled'):cancel(id)
  except Exception:pass
 os.makedirs('.runtime/evidence',exist_ok=True);path='.runtime/evidence/e2e-'+run+'.json';json.dump(evidence,open(path,'w'),ensure_ascii=False,indent=2);print('Evidence:',path,flush=True)
