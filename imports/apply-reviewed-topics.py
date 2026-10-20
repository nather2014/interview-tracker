"""Apply this reviewed, guarded topic plan on the Interview Tracker VPS."""
import json,pathlib,subprocess,urllib.request,urllib.parse,datetime,sys
root=pathlib.Path(__file__).resolve().parent
plan=json.loads((root/(sys.argv[1] if len(sys.argv)>1 else 'topic-review-plan.json')).read_text())
base='http://127.0.0.1:8091/api'
def fetch(path):return json.load(urllib.request.urlopen(base+path,timeout=20))
def all_reports():
 first=fetch('/interviews?page=0');items=list(first['items'])
 for page in range(1,(first['total']+first['pageSize']-1)//first['pageSize']):items+=fetch('/interviews?page='+str(page))['items']
 return items
before=all_reports()
current={q['id']:q for r in before for q in r['questions']}
for q in plan:
 assert current[q['id']]['question']==q['question'] and current[q['id']]['topic']==q['topic'],'Report changed since review; no updates applied'
subprocess.run(['/usr/local/sbin/interview-tracker-backup'],check=True)
stamp=datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
backup=pathlib.Path('/root')/('interview-topics-before-'+stamp+'.json')
backup.write_text(json.dumps(before,ensure_ascii=False,indent=2));backup.chmod(0o600)
encoded=json.dumps(plan,ensure_ascii=False).replace("'","''")
sql="""BEGIN;
DO $review$
DECLARE item jsonb; stored record;
BEGIN
 FOR item IN SELECT value FROM jsonb_array_elements('"""+encoded+"""'::jsonb) LOOP
  SELECT question,topic,interview_id INTO stored FROM question WHERE id=(item->>'id')::bigint FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION 'Question missing'; END IF;
  IF stored.question IS DISTINCT FROM item->>'question' OR stored.topic IS DISTINCT FROM item->>'topic' OR stored.interview_id IS DISTINCT FROM (item->>'interviewId')::bigint THEN RAISE EXCEPTION 'Question changed since review'; END IF;
  UPDATE question SET topic=item->>'newTopic' WHERE id=(item->>'id')::bigint;
 END LOOP;
END $review$;
COMMIT;
"""
subprocess.run(['runuser','-u','postgres','--','psql','-X','-v','ON_ERROR_STOP=1','-d','interviews'],input=sql,text=True,check=True,capture_output=True)
after=all_reports();by_id={r['id']:r for r in after};new={q['id']:q['newTopic'] for q in plan}
for original in before:
 expected=json.loads(json.dumps(original))
 for q in expected['questions']:
  if q['id'] in new:q['topic']=new[q['id']]
 assert by_id[original['id']]==expected,'Unexpected report differences'
filters=fetch('/filters')
assert set(new.values()).issubset(set(filters['topics']))
for topic in set(new.values()):
 results=fetch('/interviews?'+urllib.parse.urlencode({'topic':topic}))
 assert results['total']>0
 for r in results['items']:
  assert all(q['topic']==topic for q in r['questions'])
print('Updated',sum(q['topic']!=q['newTopic'] for q in plan),'topics across',len({q['interviewId'] for q in plan}),'reports')
print('Verified all question IDs/texts, report details, topic options and filters')
print('Rollback snapshot:',backup)
print('Topics:',', '.join(sorted(set(new.values()))))
