#!/usr/bin/env python3
"""Publish or alter synthetic device inventory ONLY on a labelled local lab node."""
import argparse,json,subprocess
p=argparse.ArgumentParser();p.add_argument('--node',default='colima-gwb');p.add_argument('--unhealthy',type=int,choices=[0,1]);p.add_argument('--memory-mib',type=int,default=8192);a=p.parse_args()
def k(*args,**kw):return subprocess.run(['kubectl',*args],check=True,capture_output=True,text=True,**kw)
node=json.loads(k('get','node',a.node,'-o','json').stdout)
if node['metadata'].get('labels',{}).get('workbench.local/lab')!='true':raise SystemExit('Refusing to modify a node without workbench.local/lab=true')
if not 1024<=a.memory_mib<=32768:raise SystemExit('memory must be 1024..32768 MiB')
devices=[]
for i in range(2):
 devices.append(dict(type='gpu',id='SIM-GPU-'+str(i),minor=i,health=i!=a.unhealthy,resources={'koordinator.sh/gpu-core':'100','koordinator.sh/gpu-memory':str(a.memory_mib)+'Mi','koordinator.sh/gpu-memory-ratio':'100'},topology=dict(socketID=0,nodeID=0,pcieID='SIM-PCIE-'+str(i),busID='0000:00:0'+str(i)+'.0')))
cr={'apiVersion':'scheduling.koordinator.sh/v1alpha1','kind':'Device','metadata':{'name':a.node,'labels':{'workbench.local/simulated':'true','workbench.local/lab':'true'},'annotations':{'workbench.local/source':'synthetic inventory; no physical GPU'}},'spec':{'devices':devices}}
print(k('apply','-f','-',input=json.dumps(cr)).stdout.strip())
print('Published simulated inventory; actual GPU usage and CUDA isolation remain unverified.')
