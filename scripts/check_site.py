from pathlib import Path
from html.parser import HTMLParser
from urllib.parse import urlsplit,unquote
root=Path(__file__).resolve().parents[1]/'apps/course-site/site'
class Links(HTMLParser):
 def handle_starttag(self,tag,attrs):
  for key,value in attrs:
   if key not in ('href','src') or not value:continue
   u=urlsplit(value)
   if u.scheme or u.netloc or not u.path:continue
   if u.path=='/':continue
   target=root/u.path.removeprefix('/course/') if u.path.startswith('/course/') else self.current.parent/unquote(u.path)
   if u.path.endswith('/'):target=target/'index.html'
   assert target.exists(),(self.current,value)
p=Links()
for f in root.rglob('*.html'):
 p.current=f;p.feed(f.read_text())
print('All local page links resolve')
