"""Install the reviewed search release and topic plan on the VPS."""
import pathlib,shutil,datetime,subprocess,json,urllib.request,urllib.parse,time,sys
stage=pathlib.Path(__file__).resolve().parent
app=pathlib.Path('/opt/interview-tracker')
stamp=datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
backup=pathlib.Path('/opt')/('interview-tracker-search-backup-'+stamp)
shutil.copytree(app,backup)
subprocess.run([sys.executable,str(stage/'apply-reviewed-topics.py')],check=True)
subprocess.run(['runuser','-u','postgres','--','psql','-X','-v','ON_ERROR_STOP=1','-d','interviews','-c','CREATE EXTENSION IF NOT EXISTS pg_trgm WITH SCHEMA public'],check=True,capture_output=True)
try:
 shutil.copy2(stage/'interview-tracker',app/'interview-tracker.new')
 (app/'interview-tracker.new').chmod(0o755)
 (app/'interview-tracker.new').replace(app/'interview-tracker')
 shutil.copytree(stage/'web',app/'web',dirs_exist_ok=True)
 subprocess.run(['systemctl','restart','interview-tracker'],check=True)
 for n in range(20):
  try:
   with urllib.request.urlopen('http://127.0.0.1:8091/api/health',timeout=2) as r:assert json.load(r)['status']=='ok'
   break
  except Exception:
   if n==19:raise
   time.sleep(.5)
 for base in ['http://127.0.0.1:8091','https://interviewtracker.online']:
  for term in ['hashmp','syncronisation','sprng securty','flatmp']:
   url=base+'/api/interviews?'+urllib.parse.urlencode({'search':term})
   with urllib.request.urlopen(url,timeout=20) as r:data=json.load(r)
   assert data['total']>0,term+' returned no matches'
   print('Verified fuzzy query:',term,'reports:',data['total'],base)
  with urllib.request.urlopen(base+'/',timeout=20) as r:assert '20261014-search' in r.read().decode()
 print('Search deployed and verified. Rollback backup:',backup)
except Exception:
 shutil.copy2(backup/'interview-tracker',app/'interview-tracker.restore')
 (app/'interview-tracker.restore').replace(app/'interview-tracker')
 shutil.copytree(backup/'web',app/'web',dirs_exist_ok=True)
 subprocess.run(['systemctl','restart','interview-tracker'],check=True)
 raise
