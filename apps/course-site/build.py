# -*- coding: utf-8 -*-
from pathlib import Path
import json,shutil
from html import escape
from shared import CSS,COMMON,js2,js3
ROOT=Path(__file__).resolve().parents[2]; SITE=Path(__file__).parent/'site'
PARTS={'hami':'第一部分 · HAMi 专题','koordinator':'第二部分 · Koordinator 专题','practice':'第三部分 · 联合实践'}
def page(title,body,script=''):
 return '<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>'+escape(title)+' · ReplaceU</title><style>'+CSS+'h3[id]{scroll-margin-top:88px}</style><header><nav><a href="/course/">ReplaceU · 系统学习云计算</a><a href="/course/catalog.html">三部分课程目录</a><a href="https://github.com/KeepOverShow/ReplaceU">项目源码</a></nav></header><main>'+body+'</main><footer>阅读不等于掌握 · 每课完成预测、实验与复述</footer><script>'+COMMON+'\n'+script+'</script></html>'
def build():
 SITE.mkdir(exist_ok=True);(SITE/'lessons').mkdir(exist_ok=True);(SITE/'reference').mkdir(exist_ok=True)
 manifest=json.loads((ROOT/'courses/catalog.json').read_text());cards=''
 for part,label in PARTS.items():
  entries=[d for d in manifest if d['part']==part]
  cards+='<section id="'+part+'"><h2>'+label+'</h2>'
  for d in entries:cards+='<h3><a href="lessons/'+d['slug']+'.html">'+d['number']+' · '+d['title']+'</a></h3><p>'+d['summary']+'</p>'
  cards+='</section>'
 index='<p class="kicker">三条清晰主线</p><h1>先 HAMi，再 Koordinator，<br>最后把知识用于实践。</h1><p>第一部分只讲 HAMi；第二部分只讲 Koordinator；第三部分把前两部分连到实际服务与源码。每课先读 Archify 主图，再判断、操作和复述。</p><nav><a href="#hami">HAMi</a><a href="#koordinator">Koordinator</a><a href="#practice">联合实践</a></nav>'+cards+'<section><h2>服务状态与学习边界</h2><p id="live">读取中…</p><button onclick="live()">刷新</button><p>课程运行在服务器。GPU 实验是明确标记的模拟；实践项目使用真实 Kubernetes/Koordinator 与 CPU 工作负载，不声称完成真实 NVIDIA 隔离验证。</p><a href="/">打开实践工作台</a></section>'
 for name in ['index.html','catalog.html']:(SITE/name).write_text(page('课程目录',index))
 for entry in manifest:
  d=json.loads((ROOT/'courses'/entry['part']/(entry['number']+'.json')).read_text());slug=d['slug'];diagram=d['diagram']
  body='<p class="kicker">'+PARTS[d['part']]+' / '+d['number']+'</p><h1>'+d['title']+'</h1><p>'+d['summary']+'</p>'+d.get('lead','')+'<details class="lesson-toc"><summary>本课目录</summary><nav><a href="#map">知识主图</a><a href="#explain">逐步讲解</a><a href="#check">选择题与解析</a>'+d.get('toc','')+'</nav></details><section id="map"><h2>先沿主图串起知识</h2><a href="../reference/'+diagram+'.html" target="_blank">打开完整 Archify 图 ↗</a><iframe title="'+d['title']+'知识主图" src="../reference/'+diagram+'.html" loading="lazy"></iframe></section>'
  if d.get('map_collapsed'):
   body=body.replace('<section id="map">','<details id="map"><summary>展开 Archify 源码链路图</summary><section>').replace('</iframe></section>','</iframe></section></details>')
  script=d.get('script','')
  if '__MEMORY_CASES__' in script:
   cases=(ROOT/'projects/hami-memory-lab/cases.json').read_text()
   script=script.replace('__MEMORY_CASES__',cases)
  if slug=='practice-01':
   from practice_intro import BODY,SCRIPT
   body=BODY.replace('第 1 课 / 3','第三部分 · 第 1 课').replace('hami-roles.html','practice-flow.html');script=SCRIPT
  else:body+='<section id="explain"><h2>逐步讲解</h2>'+d['body']+'</section>'
  if d.get('memory_lab'):script+=js3
  if slug=='practice-02':script+=js2
  if d['choices']:
   quizzes=[{'question':d.get('question','请选择正确说法：'),'choices':d['choices'],'correct':d['correct'],'feedback':d['feedback']}]+d.get('extra_quizzes',[])
   body+='<section id="check"><h2>选择题与解析</h2>'
   for i,q in enumerate(quizzes):
    quiz_id='checkQuiz'+str(i)
    body+='<div class="quiz-question"><p><b>'+str(i+1)+'.</b> '+escape(q['question'])+'</p><div id="'+quiz_id+'" class="choices">'+''.join('<button type="button">'+escape(c)+'</button>' for c in q['choices'])+'<output aria-live="polite">请选择一个答案。</output></div></div>'
    script+='quiz('+json.dumps(quiz_id)+','+str(q['correct'])+','+json.dumps(q['feedback'],ensure_ascii=False)+');'
   body+='</section>'
  siblings=[x for x in manifest if x['part']==d['part']]
  pos=next(i for i,x in enumerate(siblings) if x['slug']==slug)
  body+='<nav aria-label="课间导航">'+''.join('<a href="'+siblings[i]['slug']+'.html">'+label+'：'+escape(siblings[i]['title'])+'</a>' for i,label in [(pos-1,'上一课'),(pos+1,'下一课')] if 0<=i<len(siblings))+'</nav>'
  body+='<p><a href="/course/catalog.html#'+d['part']+'">返回本部分目录</a></p>'
  (SITE/'lessons'/(slug+'.html')).write_text(page(d['title'],body,script))
 for name in ['practice-flow','hami-runtime','hami-overview','koord-overview','hami-handoff','hami-pod-card','hami-device-protocol','hami-capacity-source','api-list-watch','device-list-and-watch']:
  source=ROOT/'diagrams'/name/'diagram.html'
  if source.exists():shutil.copy2(source,SITE/'reference'/(name+'.html'))
 for reference in (ROOT/'courses/reference').glob('*.html'):
  shutil.copy2(reference,SITE/'reference'/reference.name)
 for old,new in {'0003-hami-roles':'practice-01','0004-gpu-fragmentation':'practice-02','0005-hami-runtime':'hami-03'}.items():
  (SITE/'lessons'/(old+'.html')).write_text('<!doctype html><meta charset="utf-8"><meta http-equiv="refresh" content="0;url='+new+'.html"><a href="'+new+'.html">课程已按三部分重组，继续学习</a>')
 print('Built',len(manifest),'lessons in',len(PARTS),'parts')
if __name__=='__main__':build()
