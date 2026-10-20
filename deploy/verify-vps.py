import json,urllib.request,urllib.error,uuid,time,subprocess
from pathlib import Path
base='http://127.0.0.1:8091'
env=dict(line.split('=',1) for line in Path('/etc/interview-tracker.env').read_text().splitlines() if line and not line.startswith('#'))
def request(path,body=None,method=None,admin=False):
 data=None if body is None else json.dumps(body).encode()
 headers={'Content-Type':'application/json'}
 if admin:headers['Authorization']='Bearer '+env['ADMIN_TOKEN']
 r=urllib.request.Request(base+path,data=data,method=method,headers=headers)
 with urllib.request.urlopen(r,timeout=10) as res:
  raw=res.read();return res.status,json.loads(raw) if raw else None
for n in range(15):
 try:assert request('/api/health')[1]['status']=='ok';break
 except Exception:
  if n==14:raise
  time.sleep(1)
assert urllib.request.urlopen(base+'/').status==200
assert urllib.request.urlopen(base+'/admin.html').status==200
assert urllib.request.urlopen(base+'/insights.html').status==200
assert urllib.request.urlopen(base+'/insights.js').status==200
insights=request('/api/insights')[1]
assert insights['reports'] >= 0 and insights['questions'] >= 0
assert isinstance(insights['topics'],list) and isinstance(insights['months'],list)
name='Deploy verification '+uuid.uuid4().hex
body={'company':name,'role':'Java Developer','experience':'3–5','interviewType':'Online','interviewDate':'2026-01-15','location':'Remote','questions':[{'question':'How does flatMap work?','topic':'Java'}]}
ident=None
try:
 status,result=request('/api/interviews',body);assert status==201;ident=result['id']
 from urllib.parse import urlencode
 args=urlencode({'company':name.lower(),'topic':'java','search':'FLATmap','experience':'3–5','year':2026})
 result=request('/api/interviews?'+args)[1];assert result['total']==1 and result['items'][0]['questions'][0]['topic']=='Java'
 result=request('/api/insights?'+urlencode({'company':name.lower()}))[1]
 assert result['reports']==1 and result['questions']==1 and result['companies']==1
 assert result['topics'][0]['label']=='Java' and result['months']==[{'label':'2026-01','count':1}]
 try:request('/api/admin/session');raise AssertionError('Anonymous admin allowed')
 except urllib.error.HTTPError as e:assert e.code==401
 assert request('/api/admin/session',admin=True)[1]['authenticated']
finally:
 if ident:assert request('/api/admin/interviews/'+str(ident),method='DELETE',admin=True)[0]==204
assert request('/api/interviews?'+urlencode({'company':name}))[1]['total']==0
minimal_id=None
try:
 status,result=request('/api/interviews',{'company':name+' company only'});assert status==201;minimal_id=result['id']
 result=request('/api/interviews?'+urlencode({'company':name+' company only'}))[1]
 assert result['total']==1 and result['items'][0]['interviewDate']=='' and result['items'][0]['questions']==[]
 result=request('/api/topics/suggest',{'questions':['How does flatMap work?','Workers must coordinate updates to shared mutable state.']})[1]
 assert [s['topic'] for s in result['suggestions']]==['Java Streams','Java Concurrency']
 assert result['suggestions'][1]['method']=='model'
 try:request('/api/interviews',{'company':'   '});raise AssertionError('Blank company allowed')
 except urllib.error.HTTPError as e:assert e.code==400
finally:
 if minimal_id:assert request('/api/admin/interviews/'+str(minimal_id),method='DELETE',admin=True)[0]==204
print('Company-only submissions and local AI topic suggestions passed.')
subprocess.run(['/usr/local/sbin/interview-tracker-backup'],check=True)
print('VPS PostgreSQL/API checks passed; temporary report deleted; initial backup created.')
pid=subprocess.check_output(['systemctl','show','interview-tracker','-p','MainPID','--value'],text=True).strip()
for line in Path('/proc/'+pid+'/status').read_text().splitlines():
 if line.startswith(('VmRSS:','Threads:')):print(line)
print(subprocess.check_output(['systemctl','show','interview-tracker','-p','MemoryCurrent','-p','CPUUsageNSec'],text=True).strip())
assert json.loads(urllib.request.urlopen('http://127.0.0.1:8090/api/health').read())['status']=='ok'
print('Little Steps health remains OK.')
