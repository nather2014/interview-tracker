import json,pathlib,resource,time
from classifier import Classifier
root=pathlib.Path(__file__).resolve().parent.parent
plan=json.loads((root/'imports/topic-review-plan.json').read_text())
examples=[{'question':q['question'],'topic':q['newTopic']} for q in plan]
start=time.monotonic();model=Classifier();loaded=time.monotonic()-start
start=time.monotonic();results=model.suggest([q['question'] for q in plan]);elapsed=time.monotonic()-start
wrong=[{'question':q['question'],'expected':q['newTopic'],'actual':s} for q,s in zip(plan,results) if q['newTopic']!=s['topic']]
print('Reviewed questions:',len(plan),'correct:',len(plan)-len(wrong),'load:',round(loaded,2),'s batch:',round(elapsed,2),'s RSS peak:',round(resource.getrusage(resource.RUSAGE_SELF).ru_maxrss/1024,1),'MiB')
for row in wrong:print(json.dumps(row))
assert not wrong
held_out=[('How do we shut off calls to an unhealthy downstream dependency until it recovers?', 'Microservices'),('How can I intercept browser requests from another domain?', 'Spring MVC'),('Explain why an iterator raises an error when its collection changes.', 'Java Collections'),('How do you prepare a balance sheet?', 'Accounting')]
for question,expected in held_out:
 result=model.suggest([question],examples)[0];print('Held-out:',question,result)
 # Ambiguous semantic matches should request review rather than pretend certainty.
 assert result['topic']==expected or result['needsReview']
custom=[{'question':'Explain how we keep widgets ordered during widget ingestion.','topic':'Java Collections'}]
model.set_examples(custom)
result=model.suggest(['Explain keeping incoming widgets ordered during ingestion.'],custom)[0]
print('Manual example learning:',result);assert result['topic']=='Java Collections'
result=model.suggest(['Workers must coordinate updates to shared mutable state.'],examples)[0]
print('Semantic fallback:',result);assert result['method']=='model' and result['topic']=='Java Concurrency'
custom=[{'question':'How does a HashMap handle collisions?','topic':'Custom Collections'}]
result=model.suggest(['How does a HashMap handle collisions?'],custom)[0]
assert result['topic']=='Custom Collections' and result['method']=='manual-example'
print('Classifier tests passed')

new_plan=json.loads((root/'imports/pwc-topic-review-plan.json').read_text())
results=model.suggest([q['question'] for q in new_plan])
for question,result in zip(new_plan,results):
 if question['newTopic']=='Other':
  assert result['needsReview'] or result['topic']=='Other',result
 else:
  assert result['topic']==question['newTopic'],(question['question'],result,question['newTopic'])
print('New PwC topic checks passed:',len(new_plan))
