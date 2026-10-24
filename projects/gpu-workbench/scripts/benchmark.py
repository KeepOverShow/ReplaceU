#!/usr/bin/env python3
"""Measure the actual CPU endpoint; no GPU performance claims."""
import argparse,json,time,urllib.request,concurrent.futures,math,os
p=argparse.ArgumentParser();p.add_argument('--label',default='baseline');p.add_argument('--samples',type=int,default=100);p.add_argument('--concurrency',type=int,default=4);a=p.parse_args()
if not 1<=a.samples<=2000 or not 1<=a.concurrency<=16:raise SystemExit('samples 1..2000, concurrency 1..16')
def one(_):
 op=urllib.request.build_opener(urllib.request.ProxyHandler({}));start=time.perf_counter()
 try:
  with op.open('http://127.0.0.1:18080/compute',timeout=15) as r:data=json.load(r)
  return dict(ms=(time.perf_counter()-start)*1000,compute_ms=data['compute_ms'],ok=True)
 except Exception as e:return dict(ms=(time.perf_counter()-start)*1000,ok=False,error=str(e))
start=time.perf_counter()
with concurrent.futures.ThreadPoolExecutor(max_workers=a.concurrency) as pool:results=list(pool.map(one,range(a.samples)))
elapsed=time.perf_counter()-start;lat=sorted(r['ms'] for r in results if r['ok'])
def pct(q):return round(lat[max(0,math.ceil(len(lat)*q)-1)],3) if lat else None
report=dict(label=a.label,backend='real-cpu',gpu_performance_verified=False,samples=a.samples,concurrency=a.concurrency,failures=a.samples-len(lat),elapsed_seconds=elapsed,throughput_per_second=len(lat)/elapsed,p50_ms=pct(.5),p95_ms=pct(.95),p99_ms=pct(.99),raw=results)
os.makedirs('.runtime/evidence',exist_ok=True);path='.runtime/evidence/cpu-'+str(time.time_ns())+'.json';json.dump(report,open(path,'w'),indent=2);print(json.dumps({k:v for k,v in report.items() if k!='raw'},ensure_ascii=False));print(path)
