#!/bin/sh
# Installs or upgrades humi from an unpacked bundle (make bundle). Run as root
# from the bundle directory: sh install.sh
# Never overwrites an existing config.yml or database.
set -eu

case $(uname -m) in
  x86_64)  arch=amd64 ;;
  aarch64) arch=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

id humi >/dev/null 2>&1 || useradd --system --home-dir /opt/humi --shell /usr/sbin/nologin humi
install -d -o humi -g humi /opt/humi /opt/humi/bin /var/log/humi

install -m 755 bin/linux-$arch/humi bin/linux-$arch/nodectl /opt/humi/bin/
rm -rf /opt/humi/web
cp -R web /opt/humi/web

[ -f /opt/humi/config.yml ] || install -m 600 config.yml /opt/humi/config.yml
if [ ! -f /opt/humi/humi.db ] && [ -f humi.db ]; then
  install -m 600 humi.db /opt/humi/humi.db
fi
chown -R humi:humi /opt/humi /var/log/humi

install -m 644 humi.service /etc/systemd/system/humi.service
systemctl daemon-reload
systemctl enable humi >/dev/null
systemctl restart humi
sleep 1
systemctl --no-pager --lines=5 status humi
