#!/usr/bin/env python3
import json,os,subprocess,time,urllib.request,socket
root=os.path.dirname(os.path.dirname(os.path.abspath(__file__)));os.chdir(root);base='.runtime/prometheus';os.makedirs(base,mode=0o700,exist_ok=True)
credentials=json.load(open('.runtime/credentials.json'))
for team,token in credentials.items():
 path=base+'/'+team+'.token';fd=os.open(path,os.O_WRONLY|os.O_CREAT|os.O_TRUNC,0o600)
 with os.fdopen(fd,'w') as f:f.write(token)
config='global:\n  scrape_interval: 5s\nscrape_configs:\n'
for team in ['alpha','beta']:
 config+='  - job_name: workbench-'+team+'\n    authorization:\n      credentials_file: '+root+'/'+base+'/'+team+'.token\n    static_configs:\n      - targets: ["127.0.0.1:8787"]\n'
config+='  - job_name: koordlet\n    static_configs:\n      - targets: ["127.0.0.1:19316"]\n'
open(base+'/prometheus.yml','w').write(config)
subprocess.run(['.tools/prometheus/promtool','check','config',base+'/prometheus.yml'],check=True)
for port,args,label in [(19316,['.tools/bin/kubectl','--kubeconfig',root+'/.runtime/kube/config','-n','koordinator-system','port-forward','daemonset/koordlet','19316:9316'],'koordlet-forward'),(9090,['.tools/prometheus/prometheus','--config.file='+base+'/prometheus.yml','--storage.tsdb.path='+base+'/data','--storage.tsdb.retention.time=3d','--web.listen-address=127.0.0.1:9090'],'prometheus')]:
 with socket.socket() as s:busy=s.connect_ex(('127.0.0.1',port))==0
 if busy:print(label,'already listening');continue
 with open(base+'/'+label+'.log','a') as log:p=subprocess.Popen(args,stdout=log,stderr=log,stdin=subprocess.DEVNULL,start_new_session=True)
 open(base+'/'+label+'.pid','w').write(str(p.pid))
print('Prometheus: http://127.0.0.1:9090')
