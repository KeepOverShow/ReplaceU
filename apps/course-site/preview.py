"""Loopback-only preview with the same /course/ prefix as the deployed proxy."""
from server import Handler, ThreadingHTTPServer
class Preview(Handler):
 def do_GET(self):
  if self.path.startswith('/course/'):self.path=self.path[len('/course'):]
  super().do_GET()
 def do_POST(self):
  if self.path.startswith('/course/'):self.path=self.path[len('/course'):]
  super().do_POST()
if __name__=='__main__':ThreadingHTTPServer(('127.0.0.1',18901),Preview).serve_forever()
