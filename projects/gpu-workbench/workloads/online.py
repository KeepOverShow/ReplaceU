"""A real CPU calculation endpoint for host-resource colocation experiments."""
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
from worker import cpu_matrix
import json,time
class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        start=time.perf_counter()
        if self.path=='/healthz':body=b'ok';ctype='text/plain'
        elif self.path=='/compute':body=json.dumps(dict(backend='cpu',checksum=cpu_matrix(32),compute_ms=round((time.perf_counter()-start)*1000,3))).encode();ctype='application/json'
        else:self.send_error(404);return
        self.send_response(200);self.send_header('Content-Type',ctype);self.send_header('Content-Length',str(len(body)));self.end_headers();self.wfile.write(body)
    def log_message(self,*args):pass
if __name__=='__main__':ThreadingHTTPServer(('0.0.0.0',8080),Handler).serve_forever()
