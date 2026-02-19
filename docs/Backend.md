# Backend — Dokumentacja

Backend systemu monitorowania napisany w Go. Pełni rolę serwera WebSocket dla agentów, zbiera i agreguje metryki, udostępnia REST API i SSE dla frontendu.

## Spis treści

- [Stack](#stack)
- [Architektura](#architektura)
- [Pipeline danych](#pipeline-danych)
- [Komponenty](#komponenty)
- [Baza danych](#baza-danych)
- [Agregacja i retencja](#agregacja-i-retencja)
- [Konfiguracja](#konfiguracja)
- [Uruchomienie](#uruchomienie)

---

## Stack

| Technologia | Wersja | Rola |
|-------------|--------|------|
| Go | 1.24+ | Język backendu |
| SQLite (CGO) | mattn/go-sqlite3 v1.14.34 | Baza danych |
| gorilla/mux | v1.8.1 | Router HTTP |
| gorilla/websocket | v1.5.3 | WebSocket (agenci) |
| golang-jwt/jwt | v5.3.1 | Autentykacja |
| google/uuid | v1.6.0 | Generowanie ID komend |
| golang.org/x/crypto | v0.48.0 | bcrypt (haszowanie haseł) |

---

## Architektura

```
Agenci (WebSocket clients)
        │
        ▼  ws://backend:8080/ws/agent
┌─────────────────────────────────────────────────┐
│  WebSocketHandler                               │
│    ├── auth → upsert serwera w DB               │
│    ├── metrics → BufferManager (RAM)            │
│    └── command response → PendingCommands       │
└──────────────────┬──────────────────────────────┘
                   │
        ┌──────────▼──────────┐
        │   BufferManager     │  RAM: ring-buffer 60s per agent/container
        │   (60s live cache)  │
        └──────┬──────┬───────┘
               │      │
    ┌──────────▼──┐  ┌▼────────────────┐
    │ BulkInserter│  │   SSE Handler   │  → frontend (live stream)
    │  (co 10s)   │  │  (live metrics) │
    └──────┬──────┘  └─────────────────┘
           │
    ┌──────▼──────┐
    │   SQLite    │  WAL mode, max 1 writer
    │  metrics_*  │
    └──────┬──────┘
           │
    ┌──────▼──────┐
    │  Aggregator │  (co 10s) — redukuje rozdzielczość starych danych
    └─────────────┘
           │
    ┌──────▼──────────────────────────┐
    │  REST API + SSE (history)       │  → frontend
    └─────────────────────────────────┘
```

---

## Pipeline danych

### 1. Odbiór metryk (WebSocket → RAM)

`WebSocketHandler.readPump` odbiera każdy pakiet `metrics` od agenta:
- Aktualizuje `last_seen` serwera w DB — **zawsze, niezależnie od statusu zatwierdzenia agenta**
- Dodaje punkt hosta do `BufferManager.HostBuffers[agentUUID]`
- Dodaje punkt każdego kontenera do `BufferManager.Buffers[agentUUID][containerID]`
- Upsertuje metadane kontenera (name, image, project, service) do tabeli `containers`

Bufor to ring-buffer o pojemności **60 punktów** (= 60 sekund przy interwale 1s agenta). Punkt wypchnięty z ring-buffera trafia do kolejki `pending` — czeka na zapis do DB.

> Backend obsługuje też legacy format agentów, gdzie zamiast pola `host` przychodzi stary format `system`. W tym trybie pola `disk_*_bytes_per_sec` i `net_*_bytes_per_sec` są ustawiane na zero.

### 2. Zapis do bazy (BulkInserter)

`BulkInserter` działa co **10 sekund** i:
1. Drainuje kolejki `pending` i `pendingHost` z `BufferManager`
2. Agreguje punkty do 5-sekundowych bucketów (`aggregateTo5s`)
3. Zapisuje wyniki jako `INSERT OR REPLACE INTO metrics_5s` w transakcji
4. Przy błędzie zapisu — requeue punktów z cappingiem do `maxPendingPerContainer = 1200` (≈ 20 minut przy interwale 1s)

Metryki hosta są zapisywane w `metrics_5s` pod specjalnym `container_id = "__host__"`.

> **Uwaga:** `disk_write_avg/min/max` jest obliczane wyłącznie dla hosta (`__host__`). Dla kontenerów te pola mają zawsze wartość 0 — kontenery przesyłają `disk_used` (bajty zajęte przez filesystem), nie przepustowość zapisu.

### 3. Agregacja historyczna (Aggregator)

`Aggregator` działa co **10 sekund** i przechodzi przez wszystkie poziomy agregacji. Dla każdego poziomu:
1. Pobiera z tabeli źródłowej dane starsze niż `SourceThreshold`
2. Agreguje do `AggregationInterval`-owych bucketów (avg/min/max)
3. Zapisuje do tabeli docelowej
4. Usuwa przetworzone wiersze z tabeli źródłowej

### 4. Serwowanie danych (API)

| Zakres | Źródło danych |
|--------|---------------|
| `1m` | RAM buffer (`BufferManager`) — surowe punkty |
| `>1m` | SQLite — zagregowane punkty z odpowiedniej tabeli `metrics_*` |

---

## Komponenty

### AgentHub

Zarządza aktywnymi połączeniami WebSocket agentów.

`Register` i `Unregister` to **kanały** (`chan *AgentConnection`) — rejestracja odbywa się przez wysłanie do kanału, logika przetwarzania jest w pętli `Run()`.

| Metoda / pole | Opis |
|---------------|------|
| `Register` | Kanał do rejestracji nowego połączenia agenta |
| `Unregister` | Kanał do usunięcia połączenia (rozłączenie / błąd) |
| `SendToAgent(uuid, payload)` | Wysyła wiadomość do konkretnego agenta |
| `SetApproved(uuid, bool)` | Pushuje zmianę statusu zatwierdzenia do agenta |
| `GetConnection(uuid)` | Zwraca aktywne połączenie agenta (lub nil) |
| `RequestAgent(agentUUID, action, target)` | Wysyła komendę i synchronicznie czeka na odpowiedź (timeout 30s); używa `PendingCommands` do korelacji |
| `Stop()` | Zatrzymuje pętlę `Run()` |
| `PendingCommands` | `sync.Map` — mapa `command_id → chan []byte` do obsługi response |

### BufferManager

In-memory cache ostatnich 60 sekund metryk.

| Dane | Struktura | Opis |
|------|-----------|------|
| `Buffers` | `map[agentUUID]map[containerID]*RingBuffer` | Metryki kontenerów |
| `HostBuffers` | `map[agentUUID]*HostRingBuffer` | Metryki hosta |
| `pending` | `map[agentUUID]map[containerID][]MetricPoint` | Czeka na zapis do DB |
| `pendingHost` | `map[agentUUID][]HostMetricPoint` | Czeka na zapis do DB |

Po rozłączeniu agenta — ring-buffery są usuwane z RAM (`RemoveAgentBuffers`), ale `pending` NIE jest czyszczone — dane trafią do DB przy następnym flushie BulkInsertera.

### BulkInserter

Zbiorczy zapis do SQLite co 10s. Unika N+1 insertów — batch insert w jednej transakcji na agenta/kontener.

### Aggregator

Kaskadowa redukcja rozdzielczości danych. Działa niezależnie od BulkInsertera.

---

## Baza danych

SQLite z trybem **WAL** (`_journal_mode=WAL`), `_synchronous=NORMAL`, `_busy_timeout=5000`, `_foreign_keys=1`. Pool ograniczony do **1 połączenia** (SQLite nie obsługuje concurrent writers).

Migracje wykonywane automatycznie przy starcie z katalogu `./migrations/` (pliki `.sql` w kolejności alfabetycznej). Każda migracja jest wykonywana przez `execWithRetry` — exponential backoff, maks. 5 prób, obsługa `"database is locked"`. Migracje używają `CREATE TABLE IF NOT EXISTS`, więc są idempotentne (brak tabeli śledzenia wersji).

### Tabele

#### `users`

| Kolumna | Typ | Opis |
|---------|-----|------|
| `id` | INTEGER PK | Auto-increment |
| `username` | TEXT UNIQUE | Nazwa użytkownika |
| `password_hash` | TEXT | bcrypt hash |
| `created_at` | DATETIME | Czas utworzenia |

#### `servers`

| Kolumna | Typ | Opis |
|---------|-----|------|
| `uuid` | TEXT PK | SHA1 z host_id agenta |
| `hostname` | TEXT | Nazwa hosta |
| `approved` | BOOLEAN | Czy serwer zatwierdzony (default: 0) |
| `cpu_model` | TEXT | Model CPU |
| `cpu_cores` | INTEGER | Liczba rdzeni logicznych |
| `memory_total` | INTEGER | RAM (bajty) |
| `platform` | TEXT | OS + wersja |
| `kernel` | TEXT | Wersja kernela |
| `architecture` | TEXT | Architektura |
| `last_seen` | DATETIME | Ostatni kontakt |
| `created_at` | DATETIME | Pierwsze pojawienie się |

#### `containers`

| Kolumna | Typ | Opis |
|---------|-----|------|
| `id` | INTEGER PK | Auto-increment |
| `agent_uuid` | TEXT | FK → `servers.uuid` (CASCADE DELETE) |
| `container_id` | TEXT | ID kontenera Docker |
| `name` | TEXT | Nazwa kontenera |
| `image` | TEXT | Obraz Docker |
| `project` | TEXT | Projekt compose |
| `service` | TEXT | Usługa compose |
| `first_seen` | DATETIME | Pierwsze pojawienie się |
| `last_seen` | DATETIME | Ostatni kontakt |

Unikalny klucz: `(agent_uuid, container_id)`.

#### `container_events`

| Kolumna | Typ | Opis |
|---------|-----|------|
| `id` | INTEGER PK | Auto-increment |
| `agent_uuid` | TEXT | FK → `servers.uuid` |
| `container_id` | TEXT | ID kontenera |
| `timestamp` | DATETIME | Czas zdarzenia |
| `event_type` | TEXT | Typ zdarzenia |
| `old_value` | TEXT | Poprzednia wartość |
| `new_value` | TEXT | Nowa wartość |

#### Tabele metryk (`metrics_5s` … `metrics_12h`)

Wszystkie 10 tabel metryk mają identyczną strukturę. Klucz główny: `(agent_uuid, container_id, timestamp)`.

| Kolumna | Typ | Opis |
|---------|-----|------|
| `agent_uuid` | TEXT | FK → `servers.uuid` (CASCADE DELETE) |
| `container_id` | TEXT | ID kontenera lub `__host__` dla hosta |
| `timestamp` | INTEGER | Unix timestamp bucketu |
| `cpu_avg/min/max` | REAL | Użycie CPU (%) |
| `mem_avg/min/max` | REAL | Pamięć używana (bajty) lub % |
| `disk_avg` | REAL | Dla hosta: odczyt dysku (bajty/s). Dla kontenera: zajęte miejsce na dysku (bajty) |
| `disk_write_avg/min/max` | REAL | Dysk write (bajty/s) — tylko dla hosta; dla kontenerów zawsze 0 |
| `net_rx_avg/min/max` | REAL | Sieć odbiór |
| `net_tx_avg/min/max` | REAL | Sieć wysyłanie |

### Indeksy

| Indeks | Tabela | Kolumny |
|--------|--------|---------|
| `idx_servers_approved` | `servers` | `approved` |
| `idx_containers_agent` | `containers` | `agent_uuid` |
| `idx_events_container` | `container_events` | `agent_uuid, container_id, timestamp` |

---

## Agregacja i retencja

Dane przechodzą przez 9 poziomów agregacji. Źródłowe wiersze są usuwane po przeniesieniu do wyższego poziomu.

| Tabela źródłowa | Tabela docelowa | Dane przenoszone po | Rozdzielczość docelowa |
|-----------------|-----------------|---------------------|------------------------|
| `metrics_5s` | `metrics_15s` | 5 min | 15s |
| `metrics_15s` | `metrics_30s` | 15 min | 30s |
| `metrics_30s` | `metrics_1m` | 30 min | 1 min |
| `metrics_1m` | `metrics_5m` | 1 godz | 5 min |
| `metrics_5m` | `metrics_15m` | 6 godz | 15 min |
| `metrics_15m` | `metrics_30m` | 12 godz | 30 min |
| `metrics_30m` | `metrics_1h` | 24 godz | 1 godz |
| `metrics_1h` | `metrics_6h` | 7 dni | 6 godz |
| `metrics_6h` | `metrics_12h` | 15 dni | 12 godz |

Tabela `metrics_12h` nie ma dalszego poziomu agregacji — dane są jednak usuwane po **30 dniach** przez osobny cleanup uruchamiany razem z Aggregatorem.

### Mapowanie zakresów API na tabele

| Parametr `?range=` | Źródło |
|--------------------|--------|
| `1m` | RAM buffer |
| `5m` | `metrics_5s` |
| `15m` | `metrics_15s` |
| `30m` | `metrics_30s` |
| `1h` | `metrics_1m` |
| `6h` | `metrics_5m` |
| `12h` | `metrics_15m` |
| `24h` | `metrics_30m` |
| `7d` | `metrics_1h` |
| `15d` | `metrics_6h` |
| `30d` | `metrics_12h` |

---

## Konfiguracja

Konfiguracja wyłącznie przez zmienne środowiskowe.

| Zmienna | Domyślna wartość | Opis |
|---------|-----------------|------|
| `PORT` | `8080` | Port HTTP serwera |
| `DATABASE_PATH` | `./data/backend.db` | Ścieżka do pliku SQLite |
| `JWT_SECRET` | `default-secret-change-me` | Sekret JWT — **zmień w produkcji** |
| `CORS_ORIGIN` | `*` | Dozwolony origin dla CORS — **działa wyłącznie dla SSE handlerów**; middleware REST API zawsze zwraca `Access-Control-Allow-Origin: *` |

---

## Uruchomienie

### Lokalnie

```bash
cd backend
go mod download
go run cmd/server/main.go
```

Serwer startuje na porcie `8080`. Pierwsze uruchomienie wymaga konfiguracji przez `POST /api/setup`.

### Docker

```bash
docker compose up -d --build
```

Build dwuetapowy: `golang:1.24-alpine` (CGO + gcc) → `alpine:latest`. CGO wymagane przez `go-sqlite3`.

### Wymagania buildowe

CGO musi być włączone (`CGO_ENABLED=1`). W Alpine wymaga pakietów `gcc` i `musl-dev`.
