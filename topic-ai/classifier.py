"""Small CPU-only topic suggester; no question text leaves this server."""
import collections,hashlib,json,pathlib,re,time
import numpy as np
import onnxruntime as ort
from tokenizers import Tokenizer
ROOT=pathlib.Path(__file__).resolve().parent
RULES=[
 ('Java Streams',r'\bstreams?\b.{0,30}lambda'),
 ('Java Collections',r'copyonwritearraylist|\bhashset\b'),
 ('Core Java',r'ways.{0,30}create.{0,20}object'),
 ('Java Concurrency',r'\byield\s*\(|\bjoin\s*\('),
 ('Messaging',r'rabbitmq|(?:pull.{0,20}push|push.{0,20}pull).{0,50}(?:messag|mechanism)|message.{0,20}retention'),
 ('AI Tools',r'\bai\b.{0,35}(?:tools?|coding)'),
 ('Observability',r'monitor.{0,30}applications?'),
 ('Production Troubleshooting',r'production.{0,30}issue|troubleshoot'),
 ('Distributed Systems',r'\bcap\b.{0,10}theorem'),
 ('Java Concurrency',r'concurrent\s*hash\s*map|producer.{0,30}consumer.{0,60}(?:without|no)\s+kafka'),
 ('Spring Async',r'enableasync|@async|(?:spring\s*boot|springboot).{0,70}(?:multi.?thread|asynchron|threading)|(?:multi.?thread|asynchron).{0,70}(?:spring\s*boot|springboot)'),
 ('Spring Security',r'spring\s*security|oauth|\bjwt\b|\brbac\b|role.based.access'),
 ('Spring MVC',r'crossorigin|restcontroller|@controller|dispatcherservlet|\bcors\b'),
 ('Spring Core',r'bean.{0,30}(?:life.?cycle|scope)|applicationcontext|@qualifier|@primary|@autowired|dependency.inject'),
 ('Kafka',r'\bkafka\b'),
 ('SQL',r'\bsql\b|\b(?:left|right|inner|outer)\s+join\b|highest\s+salary|\bquery\b.{0,100}(?:employee|salary|table)|\bforeign.key\b|\bprimary.key\b'),
 ('Microservices',r'circuit.breaker|service.discovery|\bmicroservices?\b|\bcqrs\b|saga.pattern'),
 ('Java Collections',r'\bhash\s*map\b|\bhashmap\b|\bhashcode\b|\bhashcollision\b|\barray\s*list\b|\barraylist\b|\blinkedlist\b|\bhashtable\b|\bcomparable\b|\bcomparator\b'),
 ('Java Streams',r'\bflatmap\b|\bparallelstream\b|\bstreams?\b|(?:intermediate|terminal).{0,20}operations'),
 ('Java Functional Interfaces',r'functional\s*interface|\bpredicate\b|\bsupplier\b|\blambda\b'),
 ('Java Concurrency',r'synchroni[sz]|\bthread.?safe\b|\btheadsafe\b|\bmultithreading\b|multi.thread|\bthreads?\b|\bwait\s*\(|\bnotify|\bcompletablefuture\b|\bexecutorservice\b|producer.consumer'),
 ('Java 8 Features',r'default.methods?|\boptional\b|date.time.api|java\s*8.{0,30}features'),
 ('Core Java',r'marker.interface|\bstatic\b|\bmain\s*\(|\bjvm\b|\bgarbage.collection\b|\bequals\s*\('),
 ('Java OOP',r'\bsegregation\b|\bsolid\b|\bpolymorphism\b|\binheritance\b|\bencapsulation\b|\baggregation\b|\bcomposition\b'),
 ('Java Exceptions',r'\bthrows?\b|\bexceptions?\b|try.catch'),
 ('Angular',r'\bangular\b|\bonpush\b|\btrackby\b'),
 ('Testing',r'\bjunit\b|\bmockito\b|unit.test|integration.test|regression.test'),
 ('DevOps',r'\bdocker\b|\bkubernetes\b|\bci.?cd\b'),
 ('Cloud',r'\baws\b|\bazure\b|google.cloud'),
 ('Accounting',r'balance.sheet|cash.flow|journal.entr|financial.statement|income.statement'),
 ('Project Experience',r'(?:tell|describe|explain).{0,45}(?:your\s+project|work\s+experience)|introduce.yourself'),
]
def normalize(text):return re.sub(r'\s+',' ',text.strip().lower())
class Classifier:
 def __init__(self,model_dir=None):
  self.topics=json.loads((ROOT/'topics.json').read_text());self.base_topics=dict(self.topics);self.labels=list(self.topics)
  options=ort.SessionOptions();options.intra_op_num_threads=1;options.inter_op_num_threads=1
  options.execution_mode=ort.ExecutionMode.ORT_SEQUENTIAL
  options.add_session_config_entry('session.intra_op.allow_spinning','0')
  options.add_session_config_entry('session.inter_op.allow_spinning','0')
  options.enable_cpu_mem_arena=False
  options.log_severity_level=3
  model=pathlib.Path(model_dir or ROOT/'model')
  self.session=ort.InferenceSession(str(model/'model.onnx'),sess_options=options,providers=['CPUExecutionProvider'])
  self.tokenizer=Tokenizer.from_file(str(model/'tokenizer.json'))
  self.tokenizer.enable_truncation(max_length=128);self.tokenizer.enable_padding()
  self.prototypes=self.encode(list(self.topics.values()))
  self.manual_topics={};self.example_key=None;self.example_vectors=None;self.example_labels=[];self.example_texts=[]
  self.cache=collections.OrderedDict()
 def encode(self,texts):
  outputs=[]
  for start in range(0,len(texts),4):
   encoded=self.tokenizer.encode_batch(texts[start:start+4])
   feeds={'input_ids':np.asarray([e.ids for e in encoded],dtype=np.int64),'attention_mask':np.asarray([e.attention_mask for e in encoded],dtype=np.int64),'token_type_ids':np.asarray([e.type_ids for e in encoded],dtype=np.int64)}
   tokens=self.session.run(None,{i.name:feeds[i.name] for i in self.session.get_inputs()})[0]
   mask=feeds['attention_mask'][...,None].astype(np.float32)
   vectors=(tokens*mask).sum(axis=1)/np.clip(mask.sum(axis=1),1,None)
   outputs.append(vectors/np.clip(np.linalg.norm(vectors,axis=1,keepdims=True),1e-9,None))
  return np.concatenate(outputs) if outputs else np.empty((0,384),dtype=np.float32)
 def set_examples(self,examples):
  examples=[e for e in examples if isinstance(e,dict) and isinstance(e.get('topic'),str) and 1<=len(e['topic'])<=80 and isinstance(e.get('question'),str) and 1<=len(e['question'])<=2000][:100]
  key=hashlib.sha256(json.dumps(examples,sort_keys=True).encode()).hexdigest()
  if key==self.example_key:return
  topics=dict(self.base_topics)
  for e in examples:topics.setdefault(e['topic'],'Interview questions about '+e['topic']+'.')
  if topics!=self.topics:
   self.topics=topics;self.labels=list(topics);self.prototypes=self.encode(list(topics.values()))
  self.example_texts=[normalize(e['question']) for e in examples];self.example_labels=[e['topic'] for e in examples]
  self.manual_topics={}
  for e in examples:self.manual_topics.setdefault(normalize(e['question']),e['topic'])
  self.example_vectors=self.encode([e['question'] for e in examples]);self.example_key=key;self.cache.clear()
 def suggest(self,questions,examples=()):
  self.set_examples(examples);results=[]
  for question in questions:
   text=normalize(question)
   if text in self.cache:
    results.append(dict(self.cache[text]));self.cache.move_to_end(text);continue
   if text in self.manual_topics:
    result={'topic':self.manual_topics[text],'needsReview':False,'method':'manual-example'}
    results.append(result);continue
   matched=next((topic for topic,pattern in RULES if re.search(pattern,text)),None)
   if matched:result={'topic':matched,'needsReview':bool(re.search(r'\bsegregation\b',text)),'method':'rules'}
   else:
    vector=self.encode([question])[0];scores=self.prototypes@vector
    # Learned manual examples supplement definitions. Avoid identical examples
    # in evaluation so a reviewed question cannot answer its own test.
    if self.example_vectors is not None:
     similarities=self.example_vectors@vector
     for index,score in enumerate(similarities):
      if self.example_texts[index]!=text:
       label=self.labels.index(self.example_labels[index]);scores[label]=max(scores[label],float(score)*.92)
    order=np.argsort(scores)[::-1];best,second=int(order[0]),int(order[1])
    score=float(scores[best]);margin=score-float(scores[second])
    result={'topic':self.labels[best] if score>=.28 else 'Other','needsReview':score<.55 or margin<.06,'method':'model'}
   if len(question)>800:result['needsReview']=True
   self.cache[text]=result
   if len(self.cache)>512:self.cache.popitem(last=False)
   results.append(dict(result))
  return results
