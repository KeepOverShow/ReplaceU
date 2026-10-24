#!/usr/bin/env python3
"""Public teaching service: bounded arithmetic simulations, no CUDA or cluster mutations."""
from http.server import ThreadingHTTPServer, SimpleHTTPRequestHandler
from pathlib import Path
import json, time, urllib.request, urllib.error, os
START=time.monotonic();ROOT=Path(__file__).resolve().parent/'site'

def allocate(used, amount):
    if type(used) is not int or type(amount) is not int or not 0<=used<=4096 or amount not in (1024,2048,3072):
        raise ValueError('invalid memory request')
    accepted=used+amount<=4096
    return dict(simulated=True,limit_mib=4096,before_mib=used,requested_mib=amount,accepted=accepted,after_mib=used+amount if accepted else used,driver_called=accepted)

def schedule(case):
    cases={'fragmented':([5120,5120],4096),'packed':([8192,2048],4096),'shared':([2048,0],2048)}
    if case not in cases:raise ValueError('unknown case')
    used,request=cases[case];choices=[i for i,u in enumerate(used) if 8192-u>=request]
    return dict(simulated=True,capacity_mib=8192,used_mib=used,request_mib=request,total_free_mib=sum(8192-u for u in used),selected_minor=choices[0] if choices else None,state='Allocated' if choices else 'Pending',policy='first-fit teaching model; not the complete Koordinator scoring algorithm')

class Handler(SimpleHTTPRequestHandler):
    def __init__(self,*a,**kw):super().__init__(*a,directory=str(ROOT),**kw)
    def end_headers(self):
        self.send_header('X-Content-Type-Options','nosniff')
        self.send_header('Referrer-Policy','strict-origin-when-cross-origin')
        self.send_header('X-Frame-Options','SAMEORIGIN')
        super().end_headers()
    def reply(self,value,status=200):
        data=json.dumps(value,ensure_ascii=False).encode();self.send_response(status);self.send_header('Content-Type','application/json; charset=utf-8');self.send_header('Cache-Control','no-store');self.send_header('Content-Length',str(len(data)));self.end_headers();self.wfile.write(data)
    def do_GET(self):
        if self.path=='/api/live':
            workbench=False
            try:
                op=urllib.request.build_opener(urllib.request.ProxyHandler({}))
                with op.open('http://127.0.0.1:18787/healthz',timeout=1.5) as r:workbench=r.status==200
            except (OSError,urllib.error.URLError):pass
            return self.reply(dict(course_service='running',uptime_seconds=round(time.monotonic()-START),workbench_reachable=workbench,gpu_mode='simulated',cuda_verified=False,checked_at=int(time.time())))
        if self.path.startswith('/api/'):return self.reply({'error':'not found'},404)
        return super().do_GET()
    def do_POST(self):
        if self.headers.get('Origin') not in (None, *os.getenv('COURSE_ORIGINS','http://127.0.0.1:18900,http://127.0.0.1:18901').split(',')):return self.reply({'error':'origin rejected'},403)
        try:
            size=int(self.headers.get('Content-Length','0'))
            if not 0<size<=1024:raise ValueError('body must be 1..1024 bytes')
            body=json.loads(self.rfile.read(size))
            if not isinstance(body,dict):raise ValueError('object required')
            if self.path=='/api/memory':return self.reply(allocate(body.get('used_mib'),body.get('request_mib')))
            if self.path=='/api/schedule':return self.reply(schedule(body.get('case')))
            return self.reply({'error':'not found'},404)
        except (ValueError,TypeError):return self.reply({'error':'invalid simulation request'},400)

if __name__=='__main__':
    server=ThreadingHTTPServer(('127.0.0.1',int(os.getenv('PORT','18900'))),Handler);server.daemon_threads=True;server.serve_forever()
