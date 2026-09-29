# humi

Home humidity and temperature monitoring for 2–5 battery-powered ESP32 nodes.
Go service + SQLite, mobile-first Angular dashboard.

- `IMPLEMENTATION.md` — build plan, API contracts, phases
- `HARDWARE.md` — what to order on amazon.es
- `humidity-monitor-plan.md` — the original sketch this came from

## Run it

```sh
# backend
go run ./cmd/nodectl -slug bedroom -name Bedroom   # prints the ingest token once
go run ./cmd/server  -conf config.yml -log .

# frontend, proxying /api to :9820
cd web && npm install && npx ng serve
```

Or build once and let the Go binary serve the UI:

```sh
make all
make run          # config.yml already points web.dir at web/dist/web/browser
```

## Post a reading

```sh
curl -X POST localhost:9820/api/v1/readings \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"rh":54.2,"temp":21.8,"vbat":3.91,"rssi":-67}'
```

## Status

Phases 1 and 3 are done: ingest, storage, node state, bucketed series, dashboard
and charts. Alerts (phase 4) and UI auth (phase 5) are specified, not built.

Hardware received 27 Sep 2026. **Phase 2 is proven on the bench**: one XIAO +
SHT41 on USB posts every 30 s to a server on the LAN and shows up on the
dashboard. Next: deep sleep, battery sense and a µA measurement before
building the rest. Firmware and flashing: `firmware/README.md`.

**The read API is unauthenticated in the service itself.** It binds to
`127.0.0.1`; in production nginx puts basic auth in front of everything except
`/api/v1/readings` until phase 5 lands.

## Deploy

Live at https://humi.nomadus.net (Ubuntu, nginx, Let's Encrypt).

```sh
make bundle                                   # bin/humi-bundle.tgz, amd64 + arm64
scp bin/humi-bundle.tgz root@server:/tmp/
# on the server:
cd /tmp && tar xzf humi-bundle.tgz && cd bundle && sudo sh install.sh
```

`install.sh` upgrades in place and never touches an existing `config.yml` or
database. First install only: `deploy/humi.nginx` into `sites-available`, a
certificate from `certbot certonly --nginx`, and `/etc/nginx/humi.htpasswd`.
Register a node on the server:

```sh
sudo -u humi /opt/humi/bin/nodectl -conf /opt/humi/config.yml -slug bedroom -name Bedroom -interval 900
```
