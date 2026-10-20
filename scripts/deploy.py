"""Build, check, upload and verify the tracker using the permanent SSH key."""
import argparse
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile

root = Path(__file__).resolve().parent.parent
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--topic-plan', help='Reviewed plan filename inside imports/, applied after deployment')
args = parser.parse_args()
if args.topic_plan and (Path(args.topic_plan).name != args.topic_plan or not (root / 'imports' / args.topic_plan).is_file()):
    parser.error('Topic plan must be an existing filename inside imports/')
key = Path(os.getenv('INTERVIEW_DEPLOY_KEY', '~/.ssh/interview_tracker_deploy')).expanduser()
host = os.getenv('INTERVIEW_DEPLOY_HOST', 'root@51.77.117.66')
port = os.getenv('INTERVIEW_DEPLOY_PORT', '20021')
options = ['-i', str(key), '-o', 'BatchMode=yes', '-o', 'IdentitiesOnly=yes', '-o', 'StrictHostKeyChecking=yes', '-o', 'ConnectTimeout=10']
ssh = ['ssh', *options, '-p', port, host]
def run(command, **kwargs):
    subprocess.run(command, check=True, cwd=root, **kwargs)
if not key.is_file():
    raise SystemExit(f'Deployment key is missing: {key}')
run([*ssh, 'true'])
go = os.getenv('GO_BINARY') or shutil.which('go') or '/home/an/code/little-steps/.tools/go/bin/go'
env = dict(os.environ, CGO_ENABLED='0')
env.setdefault('GOCACHE', str(root / '.tools/go-cache'))
env.setdefault('GOPATH', '/home/an/code/little-steps/.tools/gopath')
run([go, '-C', 'backend', 'test', './...'], env=env)
run([go, '-C', 'backend', 'vet', './...'], env=env)
run(['node', '--check', 'web/app.js'])
run(['node', '--check', 'web/insights.js'])
run([str(root / '.tools/topic-env/bin/python'), 'topic-ai/test-classifier.py'])
with tempfile.TemporaryDirectory(prefix='interview-release-') as temporary:
    stage = Path(temporary)
    run([go, '-C', 'backend', 'test', '-c', '-o', str(stage / 'integration.test')], env=env)
    run([go, '-C', 'backend', 'build', '-buildvcs=false', '-trimpath', '-ldflags=-s -w', '-o', str(stage / 'interview-tracker'), '.'], env=env)
    release = stage / 'release.tar.gz'
    with tarfile.open(release, 'w:gz') as archive:
        archive.add(stage / 'interview-tracker', arcname='interview-tracker')
        archive.add(stage / 'integration.test', arcname='backend/integration.test')
        for folder in ['web', 'deploy', 'topic-ai', 'imports']:
            for path in sorted((root / folder).rglob('*')):
                if path.is_file() and '__pycache__' not in path.parts and path.suffix not in ['.gz', '.pyc']:
                    archive.add(path, arcname=str(path.relative_to(root)))
    run(['scp', *options, '-P', port, str(release), host + ':/root/interview-tracker-release.tar.gz'])
    # The server runner snapshots the current release and rolls it back on failure.
    runner = (root / 'deploy/run-release.py').read_bytes()
    run([*ssh, 'python3 -'], input=runner)
    if args.topic_plan:
        import shlex
        run([*ssh, 'python3 /root/interview-tracker-go-release/imports/apply-reviewed-topics.py ' + shlex.quote(args.topic_plan)])
print('Deployment and verification passed.')
