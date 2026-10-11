"""Install a Go release on Ubuntu as root; keep existing database credentials."""
import os
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
