#!/usr/bin/env bash
set -euo pipefail
: "${DEPLOY_HOST:?Set DEPLOY_HOST to user@server}"
: "${COURSE_DOMAIN:?Set COURSE_DOMAIN}"
[[ "$COURSE_DOMAIN" =~ ^[A-Za-z0-9.-]+$ ]] || exit 2
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
python3 "$ROOT/apps/course-site/build.py"
release=$(mktemp -d)
trap 'rm -rf "$release"' EXIT
cp "$ROOT/apps/course-site/server.py" "$release/"
cp -R "$ROOT/apps/course-site/site" "$release/"
tar -czf "$release/course.tar.gz" -C "$release" server.py site
scp "$release/course.tar.gz" "$DEPLOY_HOST:/tmp/replaceu-course.tar.gz"
# SSH target must have root privileges. Existing service updates keep the process running.
ssh "$DEPLOY_HOST" bash -s -- "$COURSE_DOMAIN" <<'REMOTE'
set -euo pipefail
domain=$1
command -v python3 >/dev/null || { apt-get update; apt-get install -y python3; }
id hami-course >/dev/null 2>&1 || useradd --system --no-create-home --shell /usr/sbin/nologin hami-course
mkdir -p /opt/hami-course/site
stage=$(mktemp -d)
tar -xzf /tmp/replaceu-course.tar.gz -C "$stage"
cp "$stage/server.py" /opt/hami-course/server.py
cp -a "$stage/site/." /opt/hami-course/site/
rm -rf "$stage" /tmp/replaceu-course.tar.gz
printf 'COURSE_ORIGINS=https://%s,https://www.%s\n' "$domain" "$domain" > /etc/replaceu-course.env
if ! systemctl cat hami-course.service >/dev/null 2>&1; then
cat > /etc/systemd/system/hami-course.service <<'UNIT'
[Unit]
Description=ReplaceU course service
After=network.target
[Service]
User=hami-course
WorkingDirectory=/opt/hami-course
EnvironmentFile=/etc/replaceu-course.env
ExecStart=/usr/bin/python3 /opt/hami-course/server.py
Restart=always
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
MemoryMax=192M
TasksMax=48
[Install]
WantedBy=multi-user.target
UNIT
systemctl daemon-reload
systemctl enable --now hami-course
fi
systemctl is-active hami-course
REMOTE
