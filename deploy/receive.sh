#!/bin/sh
# Receives a bundle (make bundle) on stdin and installs it. This is the forced
# command of the CI deploy key in root's authorized_keys, so that key can do
# this and nothing else:
#
#   command="/usr/local/sbin/humi-receive",restrict ssh-ed25519 AAAA... humi-deploy
#
# install.sh keeps this file at /usr/local/sbin/humi-receive, root-owned.
set -eu

BACKUPS=/var/backups/humi
KEEP=5

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

tar xzf - -C "$work"
[ -f "$work/bundle/install.sh" ] || { echo "no bundle on stdin" >&2; exit 1; }

# A copy of the database before the new binary migrates it. Stopped, nothing
# writes while the file and its WAL are copied, so the pair is consistent.
if [ -f /opt/humi/humi.db ]; then
  install -d -m 700 "$BACKUPS"
  systemctl stop humi
  stamp=$(date -u +%Y%m%dT%H%M%SZ)
  cp -p /opt/humi/humi.db "$BACKUPS/humi-$stamp.db"
  if [ -f /opt/humi/humi.db-wal ]; then
    cp -p /opt/humi/humi.db-wal "$BACKUPS/humi-$stamp.db-wal"
  fi
  ls -1t "$BACKUPS"/humi-*.db | tail -n +$((KEEP + 1)) | while read -r old; do
    rm -f "$old" "$old-wal"
  done
  echo "database backed up to $BACKUPS/humi-$stamp.db"
fi

cd "$work/bundle"
sh install.sh

port=$(sed -n 's/^[[:space:]]*port:[[:space:]]*"\{0,1\}\([0-9][0-9]*\).*/\1/p' /opt/humi/config.yml | head -n 1)
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if health=$(curl -fsS "http://127.0.0.1:${port:-9820}/api/v1/health" 2>/dev/null); then
    echo "healthy: $health"
    exit 0
  fi
  sleep 1
done
echo "humi did not come up healthy" >&2
journalctl -u humi --no-pager -n 30 >&2
exit 1
