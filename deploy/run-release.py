"""Run on the VPS over SSH; restore the previous release if verification fails."""
import datetime
import os
from pathlib import Path
import shutil
import subprocess
import tarfile

stamp = datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ')
backup = Path('/opt') / ('interview-tracker-rollback-' + stamp)
app = Path('/opt/interview-tracker')
backup.mkdir(mode=0o700)
shutil.copytree(app, backup / 'app', symlinks=True)
configs = [Path('/etc/interview-tracker.env'), Path('/etc/systemd/system/interview-tracker.service'), Path('/etc/systemd/system/interview-topics.service')]
for index, path in enumerate(configs):
    if path.exists():
        shutil.copy2(path, backup / str(index))
stage = Path('/root') / ('interview-tracker-stage-' + stamp)
stage.mkdir(mode=0o700)
with tarfile.open('/root/interview-tracker-release.tar.gz') as archive:
    archive.extractall(stage, filter='data')
# Tests create and clean a separate schema; existing interviews are untouched.
config = dict(line.split('=', 1) for line in configs[0].read_text().splitlines() if line and not line.startswith('#'))
subprocess.run([str(stage / 'backend/integration.test'), '-test.v'], cwd=stage / 'backend', env=dict(os.environ, TEST_DATABASE_URL=config['DATABASE_URL']), check=True)
try:
    subprocess.run(['python3', str(stage / 'deploy/install-vps.py')], check=True)
    subprocess.run(['python3', str(stage / 'deploy/verify-vps.py')], check=True)
except BaseException:
    subprocess.run(['systemctl', 'stop', 'interview-tracker', 'interview-topics'], check=False)
    app.rename(Path('/opt') / ('interview-tracker-failed-' + stamp))
    shutil.copytree(backup / 'app', app, symlinks=True)
    for index, path in enumerate(configs):
        saved = backup / str(index)
        if saved.exists():
            shutil.copy2(saved, path)
        elif path.exists():
            path.unlink()
    subprocess.run(['systemctl', 'daemon-reload'], check=True)
    if (backup / '2').exists():
        subprocess.run(['systemctl', 'restart', 'interview-topics'], check=True)
    else:
        subprocess.run(['systemctl', 'disable', 'interview-topics'], check=False)
    subprocess.run(['systemctl', 'restart', 'interview-tracker'], check=True)
    print('Deployment failed; previous application and configuration restored.', flush=True)
    raise
print('Verified deployment. Rollback files:', backup, flush=True)
