"""Download only the small quantized MiniLM model and tokenizer, pinned to a commit."""
import hashlib,json,pathlib,urllib.request
root=pathlib.Path(__file__).resolve().parent/'model';root.mkdir(exist_ok=True)
repo='sentence-transformers/all-MiniLM-L6-v2'
metadata=json.load(urllib.request.urlopen('https://huggingface.co/api/models/'+repo,timeout=30))
revision=metadata['sha'];manifest={'repository':repo,'revision':revision,'files':{}}
for remote,local in [('onnx/model_quint8_avx2.onnx','model.onnx'),('tokenizer.json','tokenizer.json'),('README.md','MODEL_CARD.md')]:
 target=root/local
 with urllib.request.urlopen(f'https://huggingface.co/{repo}/resolve/{revision}/{remote}',timeout=120) as response, target.with_suffix('.download').open('wb') as output:
  while chunk:=response.read(1024*1024):output.write(chunk)
 target.with_suffix('.download').replace(target)
 manifest['files'][local]=hashlib.sha256(target.read_bytes()).hexdigest()
 print(local,target.stat().st_size,'bytes')
(root/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
