import json,os
from http.server import BaseHTTPRequestHandler,HTTPServer
from classifier import Classifier
model=Classifier()
class Handler(BaseHTTPRequestHandler):
 def log_message(self,*args):pass
 def reply(self,status,body):
  data=json.dumps(body).encode();self.send_response(status);self.send_header('Content-Type','application/json');self.send_header('Content-Length',str(len(data)));self.end_headers();self.wfile.write(data)
 def do_GET(self):
  self.reply(200,{'status':'ok','model':'MiniLM-int8','threads':1}) if self.path=='/health' else self.reply(404,{'message':'Not found'})
 def do_POST(self):
  if self.path!='/suggest':self.reply(404,{'message':'Not found'});return
  try:
   size=int(self.headers.get('Content-Length','0'))
   if not 0<size<=1<<20:raise ValueError('Invalid request size')
   body=json.loads(self.rfile.read(size));questions=body['questions'];examples=body.get('examples',[])
   if not isinstance(questions,list) or not 1<=len(questions)<=50 or any(not isinstance(q,str) or not 1<=len(q)<=2000 for q in questions):raise ValueError('Invalid questions')
   if not isinstance(examples,list) or len(examples)>100:raise ValueError('Invalid examples')
   self.reply(200,{'suggestions':model.suggest(questions,examples)})
  except (ValueError,KeyError,TypeError):self.reply(400,{'message':'Invalid suggestion request'})
  except Exception:self.reply(503,{'message':'Topic suggestions unavailable'})
server=HTTPServer(('127.0.0.1',int(os.getenv('TOPIC_PORT','8092'))),Handler)
server.timeout=10
print('MiniLM topic service ready on loopback; one CPU thread',flush=True)
server.serve_forever()
