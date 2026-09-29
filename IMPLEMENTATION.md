# Implementation plan

Detailed build plan derived from `humidity-monitor-plan.md`. Scope: **2–5 battery
nodes**, current state plus statistics, **read primarily on a phone**.

Phases 1, 3 and 5 of the plan below are in the repository and run end to end;
phase 2 is proven on the bench. Phases 4, 6 and 7 are specified but not built.

---

## 1. Shape of the system

```
[ESP32-C3 + SHT41]  --HTTPS POST /api/v1/readings-->  [Go service]  -->  [SQLite]
   2–5 nodes, deep sleep                                   |
   wake → measure → POST → sleep                           +--> alert engine --> Telegram
                                                           |
                                                           +--> Angular PWA (mobile first)
```

One binary. It serves the API, runs the alert ticker and hands out the Angular
build. nginx in front terminates TLS. No container, no external broker, no time
series database — at 5 nodes × 4 readings/hour that is 175 k rows a year, which
SQLite does not notice.

### Decisions that differ from the original plan

| Topic | Original | Here | Why |
|---|---|---|---|
| Ingest payload | single reading | **single *or* array** | a node that buffered while offline flushes in one wake; same handler |
| Reading time | server clock only | server clock **minus `age_s`** | buffered readings keep their real time without an RTC on the node |
| Node token | `token_hash` (unspecified) | 256-bit random, **SHA-256** digest | the token has full entropy already; bcrypt would only add latency to the hottest path |
| Migrations | `golang-migrate` | **embedded `.sql` + a 30-line runner** | one dependency less, migrations ship inside the binary |
| Charts | ng2-charts / ECharts | **inline SVG, zero dependency** | 250 kB initial bundle instead of ~600 kB; matters on a phone over 4G |
| Statistics | on-the-fly `GROUP BY` | same, **with a server-side point cap** | the phone never receives more points than it can draw (480) |
| Migrations dir | `/migrations` | `internal/database/migrations` | `go:embed` cannot reach above its own package |
| UI session | JWT HS256 in `localStorage` | **opaque token, SHA-256 in `sessions`, HttpOnly cookie** | revocable per device, no secret to rotate, not readable by page script |
| Users | single user | **admin + viewer, invite links** | a household shares the dashboard; only admins change sensors |

---

## 2. Repository layout

```
humi/
├── cmd/
│   ├── server/main.go            # flags: -conf config.yml -log <dir>
│   └── nodectl/main.go           # registers a node, prints its token once
├── entity/                       # domain types, JSON contracts
├── impl/core/core.go             # business logic aggregator
├── internal/
│   ├── config/                   # cleanenv YAML
│   ├── database/                 # sqlite, embedded migrations, queries
│   ├── http-server/
│   │   ├── api/                  # router, static file serving
│   │   ├── handlers/             # readings, nodes, stats, service
│   │   └── middleware/           # nodeauth, timeout
│   └── lib/                      # logger, sl, response, clock, token
├── web/                          # Angular 22, standalone + signals
├── firmware/                     # ESP32 sketch
├── deploy/                       # systemd unit, nginx site
├── config.yml
└── Makefile
```

---

## 3. API

Every response is the house envelope: `{data, success, status_message, timestamp}`.

### `POST /api/v1/readings` — node → server

The only endpoint exposed to the internet. `Authorization: Bearer <node token>`,
rate limited per IP, body capped at 64 kB.

```json
{ "rh": 54.2, "temp": 21.8, "vbat": 3.91, "rssi": -67, "fw": "1.0.2" }
```

or, after an outage:

```json
[ { "rh": 54.2, "temp": 21.8, "age_s": 0 },
  { "rh": 55.1, "temp": 21.6, "age_s": 900 } ]
```

`age_s` is seconds before now, clamped to `ingest.max_age_s`. The server owns the
clock; nodes have no RTC and never need one.

Reply carries `interval_s`, so changing a node's cadence on the server is picked
up by the node on its next wake without reflashing:

```json
{ "data": { "stored": 2, "interval_s": 900 }, "success": true }
```

Rejections: `401` unknown token, `400` malformed or out-of-range
(`rh` outside 0–100, `temp` outside −50…100, batch over `max_batch`), `429` rate
limited.

### `GET /api/v1/nodes` — dashboard

Every node with its latest reading and a computed `online` flag
(`now − last_seen ≤ interval_s × ingest.offline_ratio`).

### `GET /api/v1/nodes/{slug}/series?from=&to=&bucket=`

Bucketed aggregate: per bucket `n`, `rh_min|avg|max`, `t_min|avg|max`,
`vbat_min`. Defaults to the last 24 h at a 300 s bucket. If the requested range
would yield more than **480** points the server widens the bucket and reports the
one it used in `bucket_s` — the client plots what it gets and never paginates.

### `GET /api/v1/health`

Row counts; also the readiness probe.

### Sessions, people, settings (phase 5)

Every route below except the first three needs the session cookie; non-GET
requests also need `X-Requested-With: humi` (403 without it) on top of
SameSite=Strict. 401 means no session, 403 means not an admin.

```
POST   /auth/login                   {username, password} → sets cookie      10/min/IP
GET    /auth/invites/{token}         what the link is for                    10/min/IP
POST   /auth/invites/{token}         join {username,password} | reset {password} → signed in
POST   /auth/logout
GET    /auth/me
PUT    /account/password             {current, new}; signs out other devices
GET    /account/sessions             signed-in devices, current marked
DELETE /account/sessions/{id}
GET    /nodes/{slug}/rules

admin:
POST   /nodes                        {slug,name,location,interval_s} → {node, token}
PATCH  /nodes/{slug}                 {name?,location?,interval_s?,enabled?}
DELETE /nodes/{slug}
POST   /nodes/{slug}/token           rotate; the old one dies at once
PUT    /nodes/{slug}/rules           replace the node's rules, one per (metric, op)
GET    /users · PATCH /users/{id} {role} · DELETE /users/{id}
POST   /users/{id}/reset             24 h single-use link
GET    /invites · POST /invites {role} · DELETE /invites/{id}   7 d single-use links
```

The last admin cannot be demoted or removed. `GET /nodes` carries `rh_low` /
`rh_high` from each node's enabled humidity rules; the dashboard colours cards
by them (damp is 10 points above `rh_high`).

### Planned

```
GET    /api/v1/alerts                # history, newest first
```

---

## 4. Data model

`internal/database/migrations/0001_init.sql`, applied at startup and recorded in
`schema_migrations`.

- **`nodes`** — `slug`, `name`, `location`, `token_hash` (unique), `interval_s`,
  `enabled`, `created_at`, `last_seen`
- **`readings`** — `node_id`, `received_at`, `rh`, `temp`, `vbat`, `rssi`;
  index on `(node_id, received_at DESC)`, which is also the series query's path
- **`readings_hourly`** — `WITHOUT ROWID`, PK `(node_id, hour)`, min/avg/max per
  metric. Empty until the retention job of Phase 6 fills it.
- **`rules`** — `node_id` (NULL = all nodes), `metric`, `op`, `threshold`,
  `for_min`, `enabled`
- **`alerts`** — `rule_id`, `node_id`, `fired_at`, `resolved_at`, `value`;
  partial index on open alerts

Pragmas at open: `journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout=5000`,
`foreign_keys=ON`. `MaxOpenConns(1)` — one writer, no `SQLITE_BUSY`, and read
volume here is three requests a minute.

---

## 5. Phases

### Phase 1 — ingest ✅ done

Go service, SQLite, migrations, `POST /readings` behind a token, `GET /health`,
`nodectl` for registering nodes. Verified end to end: single reading, batch with
`age_s`, rejected token.

### Phase 2 — one node on the bench

Breadboard, USB power, no sleep. Prove the whole path before worrying about
current draw.

- SHT41 on I²C, read every 30 s, print to serial
- `WiFi.begin()`, `HTTPClient` POST with the bearer token
- Confirm the row lands with `sqlite3 humi.db 'SELECT * FROM readings'`
- Only then add `esp_deep_sleep_start()`

### Phase 3 — statistics and the dashboard ✅ done

`GET /nodes`, `GET /nodes/{slug}/series`, and the Angular app:

- **Dashboard** — one card per node: humidity as the headline number, temperature
  beside it, a state chip (icon + word + colour, never colour alone), then
  last-seen / battery / RSSI. Battery under 3,4 V turns red. Polls every 60 s.
- **Detail** — 24 h / 7 d / 30 d switcher, two single-measure charts (humidity,
  temperature) each drawing the min–max band as a wash under a 2 px average line,
  crosshair and tooltip on hover and touch, and a table view for the values.
- Never a dual-axis chart. Two measures, two charts.
- Colours are the validated two-hue set (blue `#2a78d6` / orange `#eb6834`
  light, `#3987e5` / `#d95926` dark) — both modes selected, not flipped; passes
  CVD and contrast checks on both surfaces.
- Mobile first: single column under 560 px, 34 px minimum touch targets,
  `env(safe-area-inset-*)`, `theme-color` per scheme.

### Phase 4 — alerts and Telegram

A goroutine on a 60 s ticker in `impl/alerts`, owned by `core`:

1. Load enabled rules, joined to nodes
2. For threshold rules, read the readings of the last `for_min` minutes. Fire
   only when **every** sample in the window breaches — one spike is not an alert.
3. Close with **hysteresis**: a rule that fired at `rh > 65` resolves at
   `rh < 62`. Without the 3 % dead band the alert chatters all night.
4. `offline`: `now − last_seen > interval_s × 3`
5. `vbat`: below 3,4 V, `for_min` of a few hours — a cold node sags under load
   and recovers
6. Dedupe on the open alert row: never notify twice before `resolved_at` is set

Delivery is one `sendMessage` call to the Telegram Bot API. The notifier is an
interface on `core` so a second channel is additive.

Default rules seeded for a new node: `rh > 65` for 60 min, `rh < 30` for 60 min,
`offline`, `vbat < 3.4`.

### Phase 5 — auth and settings ✅ done

- `users`, `sessions`, `invites` (migration `0002`); bcrypt, opaque session
  tokens stored as SHA-256, 30-day sliding expiry refreshed at most hourly
- First admin from `userctl -invite admin`; everyone else by invite link from
  Settings → People & access, shared with the phone's share sheet
- Settings → Sensors: add (token shown once), rename, interval, on/off,
  thresholds, rotate token, delete. Viewers see only their own account page
- Angular: functional `authGuard` / `adminGuard`, an interceptor that adds
  `X-Requested-With` and sends a dead session to sign-in, Signal Forms
- RH comfort bands now come from the rules API; `models.ts` holds no thresholds

### Phase 6 — retention and backup

Daily cron goroutine:

- Readings older than **30 days** → fold into `readings_hourly`, delete the raw
  rows. `Series` picks the table by range: raw under 30 days, hourly above.
- `readings_hourly` older than **2 years** → delete
- `VACUUM` monthly
- `sqlite3 humi.db ".backup '/backup/humi-$(date +%F).db'"` nightly, then off-box

### Phase 7 — the remaining nodes

Assemble, register with `nodectl`, flash the token, place, and let it run a week
before closing the boxes.

---

## 6. Firmware outline

```
wake (timer)
  ├─ restore BSSID, channel and the DHCP lease from RTC memory
  ├─ power the sensor module GPIO high, wait 10 ms
  ├─ SHT41 high-precision measurement (~9 ms)
  ├─ read VBAT from the divider
  ├─ WiFi.config(lease) + WiFi.begin(ssid, pass, channel, bssid)   ← the reconnect shortcut
  ├─ POST, with the buffered readings appended
  │     ok    → clear the buffer, store interval_s from the reply
  │     fail  → retry once, then push into the RTC buffer and give up
  ├─ save BSSID, channel, and the lease if DHCP ran
  └─ esp_deep_sleep(interval_s)
```

RTC memory survives deep sleep and holds: BSSID, channel, the last DHCP lease,
the ring buffer of unsent readings (8 entries is plenty), and a failure counter.
After 5 consecutive failures, drop back to a full scan on the next wake — the AP
may have moved.

**The lease matters more than the BSSID.** Measured on the bench (XIAO ESP32C3,
~85 mA awake): association with a known BSSID takes 0,1–0,2 s, DHCP ~3,3 s.
Reusing the lease brings a wake from ~3,7 s to 0,23–0,47 s typical, with a rare
~5 s outlier when the AP is slow to associate. DHCP runs again every 6 h, and
after any failed connect, so the router keeps seeing the lease renewed. A DHCP
reservation for the node's MAC on the router removes the last conflict risk.

Open questions from the original plan, answered:

- **Interval** — 15 min. Indoor humidity does not move faster than that, and it
  keeps the offline detector at a usable 45 min.
- **Adaptive cadence** — no. It complicates the firmware, breaks even bucket
  spacing in the charts, and the battery already lasts most of a year.
- **OTA** — no. Two to five nodes are reachable with a USB cable; waking to poll
  for updates costs more battery than it saves.

---

## 7. Deployment

- `deploy/humi.service` — systemd, `Restart=always`, runs as its own user
- `deploy/humi.nginx` — TLS via Let's Encrypt, proxy to `127.0.0.1:9820`
- Only `/api/v1/readings` needs to be reachable from outside the LAN. If the
  nodes are all on the home network, publish nothing and reach the UI over
  Tailscale or a VPN — that removes Phase 5 from the critical path entirely.
- `config.yml` holds the Telegram key: `chmod 600`, owned by the service user.

---

## 8. Testing

- Table-driven unit tests for the ingest validator, the bucket-widening rule and
  the alert hysteresis state machine
- Integration tests against a **real SQLite file** in `t.TempDir()`, never a mock
- `go test -race ./...`; `-short` skips the integration set
