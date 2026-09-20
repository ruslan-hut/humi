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

Hardware for three nodes ordered 20 Sep 2026, due 24 Sep (cells 22–25 Sep).
Next up is **phase 2** — one node on a breadboard over USB, no deep sleep, to
prove SHT41 → WiFi → POST before touching power optimisation.

**The read API is unauthenticated.** The service binds to `127.0.0.1`; keep it
there until phase 5 lands, or reach it over a VPN.
