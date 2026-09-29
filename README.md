# humi

Home humidity and temperature monitoring for 2–5 battery-powered ESP32 nodes.
Go service + SQLite, mobile-first Angular dashboard.

- `IMPLEMENTATION.md` — build plan, API contracts, phases
- `HARDWARE.md` — what to order on amazon.es
- `humidity-monitor-plan.md` — the original sketch this came from

## Run it

```sh
# backend
go run ./cmd/server  -conf config.yml -log .
go run ./cmd/userctl -invite admin                 # prints a one-time link for the first admin

# frontend, proxying /api to :9820
cd web && npm install && npx ng serve
```

Open the link, pick a username and password. Sensors, people and invites are
managed from Settings (the gear) from then on; `nodectl` still works for
registering a node from the shell.

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

Phases 1, 3 and 5 are done: ingest, storage, node state, bucketed series,
dashboard and charts, sign-in with admin/viewer roles, and settings for sensors
(name, interval, thresholds, tokens) and people (invites, reset links, devices).
Alerts (phase 4) are specified, not built: thresholds are stored and colour the
dashboard, but nothing is sent yet.

Hardware received 27 Sep 2026. **Phase 2 is proven on the bench**: one XIAO +
SHT41 on USB posts every 30 s to a server on the LAN and shows up on the
dashboard. Next: deep sleep, battery sense and a µA measurement before
building the rest. Firmware and flashing: `firmware/README.md`.

Everything except `/api/v1/readings` (node token) and `/api/v1/health` needs a
signed-in session: an HttpOnly, SameSite=Strict cookie, 30 days sliding.

## Deploy

Live at https://humi.nomadus.net (Ubuntu, nginx, Let's Encrypt).

```sh
make bundle                                   # bin/humi-bundle.tgz, amd64 + arm64
scp bin/humi-bundle.tgz root@server:/tmp/
# on the server:
cd /tmp && tar xzf humi-bundle.tgz && cd bundle && sudo sh install.sh
```

`install.sh` upgrades in place and never touches an existing `config.yml` or
database. First install only: `deploy/humi.nginx` into `sites-available` and a
certificate from `certbot certonly --nginx`.

The first admin, on the server (`web.base_url` in `/opt/humi/config.yml` makes
the link absolute; add it by hand on a server installed before it existed):

```sh
sudo -u humi /opt/humi/bin/userctl -conf /opt/humi/config.yml -invite admin
sudo -u humi /opt/humi/bin/userctl -conf /opt/humi/config.yml -reset <username>   # locked out
```

**Upgrading from basic auth:** deploy, create the admin with the link above
(basic auth still in front is fine), then drop the `auth_basic` lines from the
live nginx site as in `deploy/humi.nginx` and `nginx -s reload`.
