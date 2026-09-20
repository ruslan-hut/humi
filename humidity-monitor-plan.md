# Мониторинг влажности дома — план проекта

## Общая схема

```
[ESP32-C3 + SHT41] --HTTPS POST--> [Go API] --> [SQLite]
      (2-3 шт,                        |
    батарейки,                        +--> [Telegram alerts]
   deep sleep)                        |
                                      +--> [Angular SPA: графики, настройки]
```

---

## 1. Железо (на каждую точку, ~10-12 €)

| Компонент | Выбор | Примечание |
|---|---|---|
| MCU | ESP32-C3 SuperMini | RISC-V, малое потребление во сне |
| Датчик | SHT41 (I²C) | ±1.8% RH, единицы мкА. Альтернатива — BME280 (+ давление) |
| Питание | Li-ion 18650 + TP4056 с защитой | или 2×AA через повышайку |

**Критично для автономности:**
- Убрать/не брать плату со светодиодом питания и линейным LDO (AMS1117) — они съедают больше самого MCU во сне
- Датчик питать с GPIO, чтобы обесточивать его на время сна
- Целевой средний ток: 100-200 мкА → 18650 хватает на 4-8 месяцев

**Прошивка (Arduino IDE или ESP-IDF):**
1. Wake from deep sleep (таймер, 10-15 мин)
2. Wi-Fi connect (сохранять BSSID + канал в RTC memory — ускоряет реконнект в разы)
3. Замер SHT41
4. HTTPS POST на сервер
5. Deep sleep

Если сервер недоступен — не ретраить бесконечно: 2 попытки, затем сон (иначе батарея сядет за ночь).

---

## 2. Go-сервис

### Стек
- `net/http` + chi/gin (по вкусу)
- `modernc.org/sqlite` (CGO-free) или `mattn/go-sqlite3`
- `golang-migrate` или embedded миграции

### API

```
POST /api/v1/readings          # приём от узлов, auth: Bearer token
GET  /api/v1/nodes             # список узлов + last_seen + состояние
GET  /api/v1/readings?node=&from=&to=&bucket=1h
GET  /api/v1/alerts            # история срабатываний
PUT  /api/v1/nodes/:id/rules   # пороги
```

### Payload от узла

```json
{
  "node": "bedroom",
  "rh": 54.2,
  "temp": 21.8,
  "vbat": 3.91,
  "rssi": -67,
  "fw": "1.0.2"
}
```

Время ставит сервер (`received_at`) — у узла нет RTC и синхронизировать его не нужно.

### Схема БД

```sql
CREATE TABLE nodes (
    id          INTEGER PRIMARY KEY,
    slug        TEXT UNIQUE NOT NULL,   -- "bedroom"
    name        TEXT NOT NULL,          -- "Спальня"
    token_hash  TEXT NOT NULL,
    created_at  INTEGER NOT NULL,
    last_seen   INTEGER
);

CREATE TABLE readings (
    id          INTEGER PRIMARY KEY,
    node_id     INTEGER NOT NULL REFERENCES nodes(id),
    received_at INTEGER NOT NULL,       -- unix
    rh          REAL NOT NULL,
    temp        REAL,
    vbat        REAL,
    rssi        INTEGER
);
CREATE INDEX idx_readings_node_time ON readings(node_id, received_at DESC);

CREATE TABLE rules (
    id         INTEGER PRIMARY KEY,
    node_id    INTEGER REFERENCES nodes(id),  -- NULL = для всех
    metric     TEXT NOT NULL,           -- rh | temp | vbat | offline
    op         TEXT NOT NULL,           -- gt | lt
    threshold  REAL NOT NULL,
    for_min    INTEGER DEFAULT 0,       -- держаться N минут до срабатывания
    enabled    INTEGER DEFAULT 1
);

CREATE TABLE alerts (
    id          INTEGER PRIMARY KEY,
    rule_id     INTEGER NOT NULL REFERENCES rules(id),
    node_id     INTEGER NOT NULL REFERENCES nodes(id),
    fired_at    INTEGER NOT NULL,
    resolved_at INTEGER,
    value       REAL
);
```

**Настройки SQLite:** `PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL; PRAGMA busy_timeout=5000;`
Объём: 3 узла × 4 записи/час × год ≈ 105k строк — для SQLite это ничто.

### Алерты

Фоновая горутина с тикером (раз в минуту):
- **Пороговые:** RH > 65% или < 30% дольше `for_min` → алерт. Гистерезис ~3% при закрытии, иначе будет дребезг
- **Offline:** `last_seen` старше 3× интервала опроса → «узел молчит»
- **Батарея:** `vbat < 3.4 V` → «пора заряжать»
- Дедупликация: не слать повторно, пока алерт не `resolved`

Канал доставки — Telegram Bot API (`sendMessage`), одного HTTP-вызова достаточно.

### Ретеншн

Cron-джоб раз в сутки: записи старше 30 дней сворачивать в часовые агрегаты (min/max/avg), старше года — удалять. Опционально, но пусть будет заложено в схему сразу.

---

## 3. Angular

- Дашборд: карточки узлов (текущие RH/T, батарея, last_seen), цветовая индикация порогов
- Графики: ng2-charts / ECharts, переключатель периода 24ч / 7д / 30д
- Страница правил: CRUD порогов
- Лента алертов
- Auth: простой JWT-логин, одного пользователя достаточно

Без SSR — статика раздаётся тем же Go-сервисом через `embed.FS` или отдельным location в nginx.

---

## 4. Деплой

- systemd-юнит для Go-бинарника
- nginx: TLS (Let's Encrypt) + reverse proxy
- Эндпоинт `/api/v1/readings` смотрит наружу → **обязательно** Bearer-токен на узел + rate limit по IP
- Бэкап: `sqlite3 db .backup` в cron, файл на внешнее хранилище

---

## Структура репозитория

```
humidity-monitor/
├── cmd/server/main.go
├── internal/
│   ├── api/          # handlers, middleware (auth, ratelimit)
│   ├── store/        # sqlite, миграции, запросы
│   ├── alerts/       # движок правил + notifier
│   └── config/
├── migrations/
├── web/              # Angular
├── firmware/         # скетч ESP32
├── docker-compose.yml
└── Makefile
```

---

## Порядок работ

1. Go: приём POST + запись в SQLite + health-check
2. Один узел на макетке, питание от USB — проверить сквозной путь
3. Graph endpoint + минимальный Angular-дашборд
4. Движок алертов + Telegram
5. Сборка узлов на батарейках, замер реального потребления
6. Ретеншн, бэкапы, второй и третий узел

---

## Открытые вопросы

- Интервал опроса: 15 мин — компромисс. Для влажности можно и 30
- Нужна ли адаптивная частота (чаще при резком изменении)? Усложняет прошивку, но продлевает жизнь батареи
- OTA-обновление прошивки: удобно, но требует просыпаться и проверять наличие обновлений — лишний трафик
