---
name: panel-backend-go
description: Zasady dla backendu API w Go. Obsługa SQLite przez database/sql + modernc, migracje goose (embedded), SSE (snake_case/PascalCase) oraz autoryzacja JWT.
---

# Skill: Panel Backend Go

## Mapa Modułu `/backend`

```
module: backend  (go 1.24.4)

cmd/server/           — entrypoint: main.go (gorilla/mux router, wires all handlers)

internal/
  api/
    auth.go           — AuthHandler: POST /api/setup, POST /api/login, GET /api/auth/status
    servers.go        — ServersHandler: GET /api/servers, GET /api/servers/{uuid},
                        PUT /api/servers/{uuid}/approve, PATCH /api/servers/{uuid},
                        DELETE /api/servers/{uuid},
                        DELETE /api/servers/{uuid}/containers/{id},
                        DELETE /api/servers/{uuid}/containers
    metrics.go        — MetricsHandler: GET /api/metrics/history/servers/{uuid}?range=...,
                        GET /api/metrics/history/servers/{uuid}/containers/{id}?range=...
                        Zakresy: 1m (RAM buffer), 5m–30d (SQLite)
    commands.go       — CommandsHandler: POST /api/servers/{uuid}/command
                        POST /api/servers/{uuid}/containers/{id}/command
                        POST /api/servers/{uuid}/containers/{id}/check-update
                        POST /api/servers/{uuid}/containers/{id}/update
                        (wysyła CommandMessage → agent przez WebSocket, czeka 30s)
    sse.go            — SSEHandler: GET /api/metrics/live/all, GET /api/metrics/live/servers/{uuid}
                        *** ASYMETRIA SSE — patrz sekcja poniżej ***
    websocket.go      — WebSocket upgrade dla agentów (gorilla/websocket)

  auth/
    jwt.go            — GenerateToken (HS256, TTL 7 dni), ValidateToken → Claims{UserID int}
    middleware.go     — Middleware(secret) → http.Handler; context key: UserIDKey = "userID"

  buffer/
    manager.go        — BufferManager: RAM ring-buffer dla metryk live (60 punktów = 1 min)
                        AddMetric, AddHostMetric, GetLatestHostForServer,
                        GetLatestForServerAtTimestamp, DrainPendingMetrics
    ringbuffer.go     — RingBuffer[MetricPoint] (per container)
    host_ringbuffer.go— HostRingBuffer[HostMetricPoint]
    inserter.go       — BulkInserter: co 10s drenuje pending i wstawia do SQLite

  database/
    db.go             — New(dbPath) → *sql.DB; driver: modernc.org/sqlite (nie CGO)
                        DSN: WAL mode, synchronous=NORMAL, busy_timeout=5000ms
                        WAŻNE: MaxOpenConns=1 (SQLite single writer)
                        PRAGMA foreign_keys = ON (modernc ignoruje DSN param)
    goose: RunMigrations(db) — embedded FS z /backend/migrations/

  models/
    server.go         — Server{UUID, Hostname, DisplayName, Icon, Status, Approved, Online,
                        CPUModel, CPUCores, CPUThreads, MemoryTotal, Platform, Kernel,
                        Architecture, LastSeen, CreatedAt}
                        Online = computed (offlineThreshold = 30s), nie stored
                        Nowe serwery: auto-approve (approved=true, status=active) w Upsert
                        GetByUUID, GetAll, Upsert, UpdateLastSeen, Approve, UpdateMeta, Delete
    container.go      — Container{AgentUUID, ContainerID, Name, Image, Project, Service,
                        State, Health, Status}
                        GetByAgent, Upsert (ON CONFLICT DO UPDATE)
    metrics.go        — RawMetricPoint, RawHostMetricPoint, HistoricalMetricPoint,
                        HistoricalHostMetricPoint
                        rangeMap: "5m"→metrics_5s … "30d"→metrics_12h
                        HostMainContainerID = "__host__"
                        GetHistoricalMetrics, GetHistoricalHostMetrics
    user.go           — User{ID, Username, PasswordHash}; bcrypt DefaultCost
                        Create (min 8 znaków), Authenticate, Exists

  websocket/
    protocol.go       — Typy wiadomości WS (agent ↔ backend):
                        AgentAuthMessage, AuthResponseMessage,
                        AgentMetricsMessage{Host *HostMetrics, Containers []ContainerMetrics}
                        CommandMessage, CommandResponse
                        ParseMessage(data []byte) → typed struct
    agent.go          — AgentHub (gorilla/mux compatible):
                        Register/Unregister chan, SendToAgent, SetApproved,
                        DisconnectAgent, RequestAgent (commandID + 30s timeout)

  aggregation/
    aggregator.go     — Aggregator.Run() co 10s, ProcessAggregation()
                        Pipeline: metrics_5s → metrics_15s → … → metrics_12h
                        Cleanup: metrics_12h dane >30d usuwane
    config.go         — ContainerAggregationLevels (9 poziomów, patrz poniżej)

migrations/           — SQL zarządzany przez goose (embedded)
  00001_initial_schema.sql   — users, servers, containers, container_events,
                               metrics_5s … metrics_12h (10 tabel)
  00002_disk_used_percent.sql— ADD COLUMN disk_used_percent_{avg,min,max} do wszystkich metrics_*
  00003_cpu_cores_threads.sql— ADD COLUMN cpu_threads do servers
  00004_container_state.sql  — ADD COLUMN state do containers
  00005_container_health.sql — ADD COLUMN health, status do containers

Kluczowe zależności (go.mod):
  github.com/golang-jwt/jwt/v5 v5.3.1
  github.com/gorilla/mux      v1.8.1
  github.com/gorilla/websocket v1.5.3
  github.com/pressly/goose/v3  v3.26.0
  golang.org/x/crypto          v0.48.0   (bcrypt)
  modernc.org/sqlite           v1.46.1   (CGO-free)
```

---

## Zasada: Local Knowledge First

Zanim użyjesz Context7 do dokumentacji `modernc.org/sqlite`, `golang-jwt/jwt`, `goose` czy `gorilla/*`:

1. **Odczytaj najpierw** plik bezpośrednio dotykany zadaniem (Read)
2. **Przeszukaj** repo pod kątem istniejącego wzorca użycia (Grep)
3. Context7 używaj TYLKO wtedy, gdy szukasz API nie używanego jeszcze w projekcie

Przykład: "Jak dodać nowy PRAGMA?" → sprawdź `database/db.go` najpierw, bo tam są już wszystkie PRAGMA.

---

## Zasada: No-Guessing (Baza i Modele)

**Bezwzględny zakaz** modyfikowania zapytań SQL lub schematu bez uprzedniego fizycznego odczytania:

1. **Przed każdą zmianą SQL w `models/*.go`** — odczytaj cały plik modelu
2. **Przed każdą nową migracją** — odczytaj WSZYSTKIE pliki w `migrations/` (stan aktualny schematu)
3. **Przed zmianą struktury tabeli** — zidentyfikuj wszystkie miejsca używające tej tabeli przez Grep

Konkretne pułapki w tym projekcie:
- `disk_used_percent_{avg,min,max}` — kolumny dodane migracją 00002, a NIE w 00001; zapytania w `aggregator.go` i `metrics.go` je uwzględniają
- `cpu_threads` — dodany migracją 00003
- Kolumna `Online` w struktcie `Server` NIE istnieje w SQLite — jest wyliczana runtime przez `isOnline(lastSeen)`
- `GetByUUID` NIE odczytuje `cpu_threads` w klauzuli SELECT — patrz `server.go:37-54` (historyczny brak, może powodować zero dla nowych serwerów przy GetByUUID)
- `modernc.org/sqlite` ignoruje `_foreign_keys=1` w DSN — wymagany `PRAGMA foreign_keys = ON` po `sql.Open` (już w `db.go:42`)
- SQLite single-writer: `MaxOpenConns(1)` ustawione w `db.go:39`; NIE zmieniaj bez analizy

---

## Asymetria SSE (KRYTYCZNE)

Istnieją dwa endpointy SSE — mają **różne formaty JSON**:

### `GET /api/metrics/live/all` (`HandleLiveAll`)
Wysyła **snake_case** (lokalny struct z json tagami):
```json
{
  "servers": [{
    "uuid": "...", "hostname": "...", "cpu": 0.5,
    "memory": 1234, "mem_percent": 45.2, "memory_total": 8000,
    "disk_used_percent": 12.3, "disk_read_bytes_per_sec": 0,
    "disk_write_bytes_per_sec": 0, "net_rx_bytes_per_sec": 0,
    "net_tx_bytes_per_sec": 0
  }]
}
```
Frontend konsumuje przez: `normalizeLiveHost()` w `frontend/types/index.ts`

### `GET /api/metrics/live/servers/{uuid}` (`HandleLiveServer`)
Wysyła **PascalCase** (lokalne structy `hostLive` / `containerLive` z json tagami PascalCase):
```json
{
  "server_uuid": "...", "timestamp": 1234567890,
  "host": {
    "Timestamp": 123, "CPU": 0.5, "MemUsed": 1234, "MemPercent": 45.2,
    "DiskReadBytesPerSec": 0, "DiskWriteBytesPerSec": 0,
    "NetRxBytesPerSec": 0, "NetTxBytesPerSec": 0, "DiskUsedPercent": 12.3
  },
  "containers": [{
    "ContainerID": "abc", "Timestamp": 123, "CPU": 1.2,
    "MemUsed": 512, "MemPercent": 6.0, "DiskUsed": 0,
    "DiskPercent": 0, "NetRx": 0, "NetTx": 0
  }]
}
```
Frontend konsumuje przez: `normalizeLiveContainer()` w `frontend/types/index.ts`

**Reguła synchronizacji:**
Każda zmiana w `api/sse.go` lub `websocket/protocol.go` (struktury `HostMetrics`, `ContainerMetrics`) **WYMAGA natychmiastowego powiadomienia Orchestratora**. Orchestrator decyduje o konieczności aktualizacji:
- `frontend/types/index.ts` (interfejsy + funkcje `normalize*`)
- `agent/internal/websocket/` (strona wysyłająca)

**NIE modyfikuj struktur SSE bez tego powiadomienia** — frontend parsuje pola po nazwie.

---

## Superpowers

Korzystaj ze skilla **Superpowers: Go Expert** (jeśli dostępny) dla:
- Poprawnej obsługi błędów (`fmt.Errorf("context: %w", err)`, nie `errors.New`)
- Goroutine lifecycle (zawsze `stopCh chan struct{}`, `defer ticker.Stop()`)
- Bezpieczeństwa współbieżności (wzorce z `sync.RWMutex` jak w `buffer/manager.go`)
- Unikania data race w testach

---

## Self-Correction: Pętla testowa (3 próby)

Po **każdej** modyfikacji kodu wykonaj z katalogu `/backend/`:

```bash
go test ./...
```

| Próba | Wynik | Akcja |
|-------|-------|-------|
| 1 | FAIL | Analizuj błąd, popraw, próba 2 |
| 2 | FAIL | Analizuj błąd, popraw, próba 3 |
| 3 | FAIL | Przygotuj **Raport błędu testów** dla Orchestratora |

**Format Raportu błędu testów:**
```
## Raport błędu testów (próba 3/3)

### Polecenie
go test ./...

### Pełny output
[wklej cały output]

### Analiza przyczyny
[co powoduje błąd — konkretnie]

### Próby naprawy (1–3)
[co zrobiłeś w każdej próbie]

### Rekomendacja
[co Orchestrator powinien zrobić dalej]
```

**Zakaz:** Subagent backendowy NIE używa `AskUserQuestion`. Eskaluje tylko raportem do Orchestratora.

---

## Instrukcja dodawania migracji

1. Odczytaj `migrations/migrations.go` — sprawdź jak embedded FS jest zadeklarowany
2. Nadaj plik kolejnym numerem: `000XX_opis.sql`
3. Format goose: `-- +goose Up` / `-- +goose Down`
4. SQLite OGRANICZENIA: brak `DROP COLUMN` w starszych wersjach, brak `ADD CONSTRAINT` po CREATE
5. **Nigdy nie modyfikuj istniejących migracji** — tylko addytywne zmiany przez nowy plik
6. Po dodaniu migracji uruchom testy: `go test ./...`
