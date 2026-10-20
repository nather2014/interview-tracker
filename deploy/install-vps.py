"""Install a Go release on Ubuntu as root; keep existing database credentials."""
import os
import datetime
import time
import json
import urllib.request
import pathlib
import secrets
import shutil
import subprocess

if os.geteuid() != 0:
    raise SystemExit('Run as root')
release = pathlib.Path('/root/interview-tracker-release.tar.gz')
if not release.is_file():
    raise SystemExit('Upload /root/interview-tracker-release.tar.gz first')
if subprocess.run(['id', 'interviews'], capture_output=True).returncode:
    subprocess.run(['useradd', '--system', '--home-dir', '/opt/interview-tracker', '--shell', '/usr/sbin/nologin', 'interviews'], check=True)
app = pathlib.Path('/opt/interview-tracker')
app.mkdir(exist_ok=True)
env = pathlib.Path('/etc/interview-tracker.env')

def sql(query):
    return subprocess.run(['runuser', '-u', 'postgres', '--', 'psql', '-At', '-v', 'ON_ERROR_STOP=1'], input=query, text=True, capture_output=True, check=True).stdout.strip()

if not env.exists():
    if sql("SELECT 1 FROM pg_roles WHERE rolname='interviews'") or sql("SELECT 1 FROM pg_database WHERE datname='interviews'"):
        raise SystemExit('Existing database or role: configure environment before installing')
    password = secrets.token_hex(32)
    sql("CREATE ROLE interviews LOGIN PASSWORD '" + password + "'")
    sql("CREATE DATABASE interviews OWNER interviews TEMPLATE template0 ENCODING 'UTF8'")
    env.write_text('DATABASE_URL=postgres://interviews:' + password + '@127.0.0.1:5432/interviews?sslmode=disable\nADMIN_TOKEN=' + secrets.token_hex(32) + '\nLISTEN_ADDR=127.0.0.1:8091\nWEB_DIR=/opt/interview-tracker/web\nGOMEMLIMIT=64MiB\nGOMAXPROCS=1\n')
    env.chmod(0o600)
stage = pathlib.Path('/root/interview-tracker-go-release')
stage.mkdir(exist_ok=True)
subprocess.run(['tar', '-xzf', str(release), '-C', str(stage)], check=True)
# Keep a complete rollback release before changing either service.
if (app / 'interview-tracker').exists():
    stamp = datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%SZ')
    shutil.copytree(app, pathlib.Path('/opt') / ('interview-tracker-backup-' + stamp))
    subprocess.run(['/usr/local/sbin/interview-tracker-backup'], check=True)
if (stage / 'topic-ai').is_dir():
    if subprocess.run(['python3', '-c', 'import ensurepip'], capture_output=True).returncode:
        subprocess.run(['apt-get', 'update', '-qq'], check=True)
        subprocess.run(['apt-get', 'install', '-y', 'python3-venv'], check=True)
    shutil.copytree(stage / 'topic-ai', app / 'topic-ai', dirs_exist_ok=True)
    topic_python = app / 'topic-venv/bin/python'
    if not topic_python.exists():
        subprocess.run(['python3', '-m', 'venv', str(app / 'topic-venv')], check=True)
    subprocess.run([str(topic_python), '-m', 'pip', 'install', '-r', str(app / 'topic-ai/requirements.txt')], check=True)
    pathlib.Path('/etc/systemd/system/interview-topics.service').write_text('''[Unit]
Description=Interview Tracker local MiniLM topic classifier
After=network.target

[Service]
User=interviews
Group=interviews
WorkingDirectory=/opt/interview-tracker/topic-ai
ExecStart=/opt/interview-tracker/topic-venv/bin/python server.py
Environment=OMP_NUM_THREADS=1
Environment=OPENBLAS_NUM_THREADS=1
Environment=TOKENIZERS_PARALLELISM=false
Environment=HF_HUB_OFFLINE=1
Environment=PYTHONDONTWRITEBYTECODE=1
Restart=on-failure
RestartSec=5
MemoryHigh=192M
MemoryMax=256M
CPUQuota=50%
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true

[Install]
WantedBy=multi-user.target
''')
    subprocess.run(['systemctl', 'daemon-reload'], check=True)
    subprocess.run(['systemctl', 'enable', 'interview-topics'], check=True)
    subprocess.run(['systemctl', 'restart', 'interview-topics'], check=True)
    for attempt in range(30):
        try:
            with urllib.request.urlopen('http://127.0.0.1:8092/health', timeout=2) as response:
                assert json.load(response)['status'] == 'ok'
            break
        except Exception:
            if attempt == 29:
                raise
            time.sleep(1)
    lines = [line for line in env.read_text().splitlines() if not line.startswith('TOPIC_SERVICE_URL=')]
    env.write_text('\n'.join(lines) + '\nTOPIC_SERVICE_URL=http://127.0.0.1:8092\n')
    env.chmod(0o600)
target = app / 'interview-tracker'
if target.exists():
    shutil.copy2(target, app / 'interview-tracker.previous')
shutil.copy2(stage / 'interview-tracker', app / 'interview-tracker.new')
(app / 'interview-tracker.new').chmod(0o755)
os.replace(app / 'interview-tracker.new', target)
shutil.copytree(stage / 'web', app / 'web', dirs_exist_ok=True)
pathlib.Path('/etc/systemd/system/interview-tracker.service').write_text('''[Unit]
Description=Interview Tracker Go API
After=network.target postgresql.service
Requires=postgresql.service

[Service]
User=interviews
Group=interviews
WorkingDirectory=/opt/interview-tracker
EnvironmentFile=/etc/interview-tracker.env
ExecStart=/opt/interview-tracker/interview-tracker
Restart=on-failure
RestartSec=5
MemoryHigh=96M
MemoryMax=128M
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
UMask=0077

[Install]
WantedBy=multi-user.target
''')
backup = pathlib.Path('/usr/local/sbin/interview-tracker-backup')
backup.write_text('''#!/bin/sh
set -eu
umask 077
mkdir -p /var/backups/interview-tracker
runuser -u postgres -- pg_dump -d interviews -Fc > "/var/backups/interview-tracker/interviews-$(date -u +%Y%m%dT%H%M%SZ).dump"
''')
backup.chmod(0o700)
pathlib.Path('/etc/cron.d/interview-tracker-backup').write_text('45 2 * * * root /usr/local/sbin/interview-tracker-backup\n')
subprocess.run(['systemctl', 'daemon-reload'], check=True)
subprocess.run(['systemctl', 'enable', 'interview-tracker'], check=True)
subprocess.run(['systemctl', 'restart', 'interview-tracker'], check=True)
print('Installed service on 127.0.0.1:8091; admin token remains in protected server environment.')
