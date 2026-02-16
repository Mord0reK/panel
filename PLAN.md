# Plan implementacji backendu - PEŁNY

---

## 🎯 ETAP 0: Setup projektu

### 0.1 Inicjalizacja
- [ ] Utwórz katalog `backend/`
- [ ] `go mod init backend`
- [ ] Utwórz strukturę folderów:
  ```
  backend/
  ├── cmd/
  │   └── server/
  │       └── main.go
  ├── internal/
  │   ├── config/
  │   ├── database/
  │   ├── models/
  │   ├── auth/
  │   ├── websocket/
  │   ├── api/
  │   ├── aggregation/
  │   └── buffer/
  ├── migrations/
  ├── tests/
  └── go.mod
  ```

### 0.2 Zależności
- [ ] Dodaj dependencies:
  - `github.com/mattn/go-sqlite3` - SQLite driver
  - `github.com/golang-jwt/jwt/v5` - JWT auth
  - `github.com/gorilla/websocket` - WebSocket
  - `github.com/gorilla/mux` - HTTP router
  - `golang.org/x/crypto/bcrypt` - password hashing
  - `github.com/google/uuid` - UUID generation
  - `github.com/stretchr/testify` - testing utilities
- [ ] `go mod tidy`

### 0.3 Config
- [ ] `internal/config/config.go`:
  - Struct `Config` z polami: `Port`, `DatabasePath`, `JWTSecret`
  - Funkcja `Load()` czytająca z env vars lub defaulty
  - Default values: port 8080, db path `./data/backend.db`

---

## 🗄️ ETAP 1: Baza danych i migracje

### 1.1 Schema SQL
- [ ] `migrations/001_initial_schema.sql`:
  - Tabela `users` (id, username, password_hash, created_at)
  - Tabela `servers` (uuid PRIMARY KEY, hostname, approved, cpu_model, cpu_cores, memory_total, platform, kernel, architecture, last_seen, created_at)
  - Tabela `containers` (id, agent_uuid, container_id, name, image, project, service, first_seen, last_seen, UNIQUE(agent_uuid, container_id))
  - Tabela `container_events` (id, agent_uuid, container_id, timestamp, event_type, old_value, new_value)
  - **11 tabel metryk:**
    - `metrics_1s` (agent_uuid, container_id, timestamp, cpu_percent, mem_used, mem_percent, disk_used, disk_percent, net_rx_bytes, net_tx_bytes, PRIMARY KEY(agent_uuid, container_id, timestamp))
    - `metrics_5s` (agent_uuid, container_id, timestamp, cpu_avg, cpu_min, cpu_max, mem_avg, mem_min, mem_max, disk_avg, net_rx_avg, net_rx_min, net_rx_max, net_tx_avg, net_tx_min, net_tx_max, PRIMARY KEY(...))
    - Analogicznie: `metrics_15s`, `metrics_30s`, `metrics_1m`, `metrics_5m`, `metrics_15m`, `metrics_30m`, `metrics_1h`, `metrics_6h`, `metrics_12h`
  - Indeksy:
    - `CREATE INDEX idx_servers_approved ON servers(approved)`
    - `CREATE INDEX idx_containers_agent ON containers(agent_uuid)`
    - `CREATE INDEX idx_events_container ON container_events(agent_uuid, container_id, timestamp)`

### 1.2 Database package
- [ ] `internal/database/db.go`:
  - Funkcja `New(dbPath string) (*sql.DB, error)` - otwiera SQLite connection
  - Funkcja `RunMigrations(db *sql.DB) error` - wykonuje wszystkie migracje z `migrations/`
  - Obsługa `database is locked` - retry logic z exponential backoff
  - Pragma: `PRAGMA journal_mode=WAL` (Write-Ahead Logging dla lepszego concurrency)
  - Pragma: `PRAGMA synchronous=NORMAL` (balance między safety a performance)

### 1.3 Test bazy
- [ ] `tests/database_test.go`:
  - Test inicjalizacji bazy (in-memory SQLite: `:memory:`)
  - Test migracji (sprawdź czy wszystkie tabele istnieją)
  - Test tworzenia i odczytu z każdej tabeli
  - Cleanup po testach

---

## 🔐 ETAP 2: Auth system

### 2.1 User model
- [ ] `internal/models/user.go`:
  - Struct `User` (ID, Username, PasswordHash, CreatedAt)
  - Metoda `Create(db, username, password) error` - hash bcrypt, INSERT
  - Metoda `Authenticate(db, username, password) (*User, error)` - sprawdź hash
  - Metoda `Exists(db) (bool, error)` - sprawdź czy jakikolwiek user istnieje

### 2.2 JWT package
- [ ] `internal/auth/jwt.go`:
  - Funkcja `GenerateToken(userID int, secret string) (string, error)` - tworzy JWT (expires: 7 dni)
  - Funkcja `ValidateToken(tokenString, secret string) (*Claims, error)` - parsuje i waliduje JWT
  - Struct `Claims` (userID, exp, iat)

### 2.3 Auth middleware
- [ ] `internal/auth/middleware.go`:
  - Funkcja `AuthMiddleware(next http.Handler) http.Handler`:
    - Czyta header `Authorization: Bearer <token>`
    - Waliduje token przez `ValidateToken()`
    - Jeśli ok → dodaje userID do request context → next handler
    - Jeśli błąd → 401 Unauthorized

### 2.4 Auth endpoints
- [ ] `internal/api/auth.go`:
  - `GET /api/auth/status` - sprawdza czy user istnieje, czy jest zalogowany
    - Response: `{"setup_required": bool, "authenticated": bool}`
  - `POST /api/setup` - first-time setup (tylko gdy brak userów w DB)
    - Body: `{"username": "...", "password": "..."}`
    - Walidacja: min 3 znaki username, min 8 znaków password
    - Tworzy usera → zwraca JWT token
    - Po utworzeniu wyłącza endpoint (sprawdza czy user istnieje)
  - `POST /api/login` - login
    - Body: `{"username": "...", "password": "..."}`
    - Authenticate → zwraca JWT token
    - Response: `{"token": "..."}`

### 2.5 Test auth
- [ ] `tests/auth_test.go`:
  - Test bcrypt hash/verify
  - Test GenerateToken/ValidateToken
  - Test expired token
  - Test invalid token
  - Test middleware z valid/invalid token
  - Test setup endpoint (create user, próba drugiego usera → 403)
  - Test login endpoint (valid/invalid credentials)

---

## 🔌 ETAP 3: WebSocket server dla agentów

### 3.1 WebSocket handler
- [ ] `internal/websocket/agent.go`:
  - Struct `AgentConnection` (UUID, Conn *websocket.Conn, SendCh chan []byte, CloseCh chan struct{})
  - Struct `AgentHub` (Connections map[string]*AgentConnection, Register chan *AgentConnection, Unregister chan *AgentConnection, mu sync.RWMutex)
  - Funkcja `NewHub() *AgentHub` - tworzy hub
  - Metoda `Hub.Run()` - goroutine zarządzająca połączeniami (register/unregister)
  - Metoda `Hub.SendToAgent(uuid string, message []byte) error` - wysyła komendę do konkretnego agenta

### 3.2 Agent protocol
- [ ] `internal/websocket/protocol.go`:
  - Struct `AgentAuthMessage` (Type="auth", UUID, Info struct{...})
  - Struct `AgentMetricsMessage` (Type="metrics", Timestamp, Containers []ContainerMetrics)
  - Struct `CommandMessage` (Type="command", Action, Target)
  - Funkcja `ParseMessage(data []byte) (interface{}, error)` - rozpoznaje typ wiadomości

### 3.3 WebSocket endpoint
- [ ] `internal/api/websocket.go`:
  - `GET /ws/agent` - WebSocket upgrade
  - Po połączeniu:
    - Czeka na `AgentAuthMessage` (timeout 10s)
    - Waliduje UUID (format check)
    - Sprawdza w DB czy serwer istnieje:
      - Nie istnieje → INSERT z approved=0 → reject metrics
      - Istnieje, approved=0 → UPDATE last_seen → reject metrics
      - Istnieje, approved=1 → UPDATE last_seen → accept metrics
    - Rejestruje w `AgentHub`
    - Read loop: odbiera metryki lub odpowiedzi na komendy
    - Write loop: wysyła komendy z `SendCh`
    - Ping/pong co 30s (heartbeat)
    - Po disconnect: wyrejestruj z hub

### 3.4 Test WebSocket
- [ ] `tests/websocket_test.go`:
  - Test połączenia agenta (mock WebSocket client)
  - Test auth message (valid UUID, invalid UUID)
  - Test approval flow (approved=0 vs approved=1)
  - Test heartbeat (ping/pong)
  - Test disconnect handling
  - Test SendToAgent (wysłanie komendy do konkretnego agenta)

---

## 💾 ETAP 4: RAM buffer i bulk insert

### 4.1 Ring buffer
- [ ] `internal/buffer/ringbuffer.go`:
  - Struct `RingBuffer` (Size int, Data []MetricPoint, WritePos int, mu sync.Mutex)
  - Struct `MetricPoint` (Timestamp int64, CPU float64, MemUsed uint64, MemPercent float64, DiskUsed uint64, DiskPercent float64, NetRx uint64, NetTx uint64)
  - Metoda `Add(point MetricPoint)` - dodaje punkt (thread-safe)
  - Metoda `GetAll() []MetricPoint` - zwraca kopię wszystkich punktów (thread-safe)
  - Metoda `Clear()` - usuwa wszystkie punkty

### 4.2 Buffer manager
- [ ] `internal/buffer/manager.go`:
  - Struct `BufferManager` (Buffers map[string]map[string]*RingBuffer, mu sync.RWMutex)
    - Key1: agent_uuid, Key2: container_id
  - Funkcja `NewManager() *BufferManager`
  - Metoda `GetOrCreate(agentUUID, containerID string) *RingBuffer` - zwraca buffer lub tworzy nowy (60 punktów capacity)
  - Metoda `AddMetric(agentUUID, containerID string, point MetricPoint)`
  - Metoda `GetAllBuffers() map[string]map[string]*RingBuffer` - snapshot wszystkich bufferów (do bulk insert)

### 4.3 Bulk insert worker
- [ ] `internal/buffer/inserter.go`:
  - Funkcja `StartBulkInserter(db *sql.DB, manager *BufferManager) *BulkInserter`
  - Struct `BulkInserter` (db, manager, stopCh chan struct{})
  - Metoda `Run()` goroutine:
    - Co 10s:
      - Pobierz snapshot wszystkich bufferów
      - Dla każdego buffera:
        - Jeśli punktów >= 10:
          - Przygotuj bulk INSERT: `INSERT INTO metrics_1s VALUES (?,?,?,...), (?,?,?,...), ...`
          - Execute transaction
          - Clear buffer (zachowaj ostatnie 60s)
      - Error handling: log, retry logic
  - Metoda `Stop()` - zatrzymuje goroutine

### 4.4 Integracja z WebSocket
- [ ] W `internal/api/websocket.go`:
  - Przy odbiorze `AgentMetricsMessage`:
    - Jeśli server approved=1:
      - Dla każdego kontenera: `bufferManager.AddMetric(agentUUID, containerID, point)`
    - Update `last_seen` w DB dla serwera

### 4.5 Test buffer
- [ ] `tests/buffer_test.go`:
  - Test RingBuffer (add, get, overflow)
  - Test BufferManager (concurrent adds)
  - Test BulkInserter (mock data → sprawdź czy zapisało do DB)
  - Test bulk insert transaction (rollback on error)

---

## 📊 ETAP 5: Agregacja

### 5.1 Aggregation config
- [ ] `internal/aggregation/config.go`:
  - Struct `AggregationLevel` (SourceTable, TargetTable, SourceThreshold time.Duration, AggregationInterval time.Duration, PointsToKeep int)
  - Variable `AggregationLevels []AggregationLevel`:
    ```
    metrics_1s → metrics_5s (> 1min, group by 5s, keep 60)
    metrics_5s → metrics_15s (> 5min, group by 15s, keep 60)
    metrics_15s → metrics_30s (> 15min, group by 30s, keep 60)
    metrics_30s → metrics_1m (> 30min, group by 1m, keep 60)
    metrics_1m → metrics_5m (> 1h, group by 5m, keep 72)
    metrics_5m → metrics_15m (> 6h, group by 15m, keep 48)
    metrics_15m → metrics_30m (> 12h, group by 30m, keep 48)
    metrics_30m → metrics_1h (> 24h, group by 1h, keep 168)
    metrics_1h → metrics_6h (> 7d, group by 6h, keep 60)
    metrics_6h → metrics_12h (> 15d, group by 12h, keep 60)
    ```

### 5.2 Aggregator
- [ ] `internal/aggregation/aggregator.go`:
  - Struct `Aggregator` (db *sql.DB, stopCh chan struct{})
  - Funkcja `NewAggregator(db *sql.DB) *Aggregator`
  - Metoda `Run()` goroutine:
    - Co 10s:
      - Dla każdego poziomu w `AggregationLevels`:
        - Query: `SELECT agent_uuid, container_id, timestamp, ... FROM {source} WHERE timestamp < {threshold} ORDER BY timestamp`
        - Grupuj po intervals (np. po 5s dla metrics_5s)
        - Dla każdej grupy oblicz:
          - AVG, MIN, MAX dla CPU, Memory, Network
          - AVG dla Disk (bez min/max)
        - Bulk INSERT do target table
        - DELETE z source table (WHERE timestamp < threshold)
      - Ostatni poziom (metrics_12h):
        - DELETE WHERE timestamp < (now - 30 dni)
  - Metoda `Stop()` - zatrzymuje goroutine

### 5.3 SQL helpers
- [ ] W `internal/aggregation/aggregator.go`:
  - Funkcja `buildAggregationQuery(level AggregationLevel, threshold int64) string`
  - Funkcja `buildDeleteQuery(table string, threshold int64) string`
  - Funkcja `executeAggregation(tx *sql.Tx, level AggregationLevel, threshold int64) error`

### 5.4 Test agregacji
- [ ] `tests/aggregation_test.go`:
  - Test z fake data:
    - Wstaw 120 punktów do metrics_1s (2 minuty danych)
    - Uruchom agregację
    - Sprawdź czy metrics_5s ma 12 punktów (120s / 5s = 24, ale trzymamy 60s = 12)
    - Sprawdź czy metrics_1s ma tylko ostatnie 60 punktów
    - Sprawdź poprawność obliczeń (avg, min, max)
  - Test dla wszystkich 10 poziomów
  - Test cleanup (metrics_12h > 30 dni → DELETE)

---

## 🌐 ETAP 6: REST API - Servers

### 6.1 Server model
- [ ] `internal/models/server.go`:
  - Struct `Server` (UUID, Hostname, Approved, CPUModel, CPUCores, MemoryTotal, Platform, Kernel, Architecture, LastSeen, CreatedAt)
  - Metoda `GetAll(db) ([]Server, error)` - wszystkie serwery
  - Metoda `GetByUUID(db, uuid) (*Server, error)`
  - Metoda `Approve(db, uuid) error` - SET approved=1
  - Metoda `Delete(db, uuid) error` - DELETE CASCADE (wszystkie kontenery i metryki)
  - Metoda `UpdateLastSeen(db, uuid) error`

### 6.2 Container model
- [ ] `internal/models/container.go`:
  - Struct `Container` (ID, AgentUUID, ContainerID, Name, Image, Project, Service, FirstSeen, LastSeen)
  - Metoda `GetByAgent(db, agentUUID) ([]Container, error)`
  - Metoda `Upsert(db, container) error` - INSERT OR REPLACE

### 6.3 Server endpoints
- [ ] `internal/api/servers.go`:
  - `GET /api/servers` (protected by auth):
    - Query wszystkie serwery
    - Response: `[{"uuid": "...", "hostname": "...", "approved": true, ...}]`
  - `GET /api/servers/:uuid` (protected):
    - Query server + lista kontenerów
    - Response: `{"server": {...}, "containers": [...]}`
  - `PUT /api/servers/:uuid/approve` (protected):
    - SET approved=1
    - Response: `{"success": true}`
  - `DELETE /api/servers/:uuid` (protected):
    - DELETE CASCADE
    - Response: `{"success": true}`

### 6.4 Test server API
- [ ] `tests/api_servers_test.go`:
  - Test GET /api/servers (z i bez auth token)
  - Test GET /api/servers/:uuid (existing, non-existing)
  - Test PUT approve
  - Test DELETE (sprawdź czy usunęło kontenery i metryki)

---

## 🎛️ ETAP 7: REST API - Commands

### 7.1 Command proxy
- [ ] `internal/api/commands.go`:
  - `POST /api/servers/:uuid/command` (protected):
    - Body: `{"action": "stop|start|restart|check-updates|update", "target": "container-name"}`
    - Znajdź połączenie WebSocket w `AgentHub` po UUID
    - Jeśli nie connected → 503 Service Unavailable
    - Wyślij komendę przez `hub.SendToAgent(uuid, message)`
    - Czekaj na odpowiedź (timeout 30s, channel lub sync.Map)
    - Response: `{"success": bool, "result": {...}}`
  - `POST /api/servers/:uuid/containers/:id/command` (protected):
    - Analogicznie, ale z konkretnym container ID

### 7.2 Command handling w WebSocket
- [ ] W `internal/websocket/agent.go`:
  - Dodaj `PendingCommands sync.Map` (commandID → response channel)
  - Przy wysyłaniu komendy:
    - Generuj commandID (UUID)
    - Dodaj do PendingCommands
    - Wyślij command z ID
    - Czekaj na response lub timeout
  - Przy odbiorze response:
    - Znajdź commandID w PendingCommands
    - Wyślij response na channel
    - Usuń z mapy

### 7.3 Test commands
- [ ] `tests/api_commands_test.go`:
  - Test wysyłania komendy (mock WebSocket agent)
  - Test timeout (agent nie odpowiada)
  - Test agent disconnected
  - Test response parsing

---

## 📈 ETAP 8: REST API - Historical metrics

### 8.1 Metrics query
- [ ] `internal/models/metrics.go`:
  - Funkcja `GetHistoricalMetrics(db, agentUUID, containerID, rangeKey string) ([]MetricPoint, error)`:
    - Map `rangeKey` → (table, points, resolution) przez `rangeConfig`
    - Query: `SELECT timestamp, cpu_avg, cpu_min, cpu_max, ... FROM {table} WHERE agent_uuid=? AND container_id=? ORDER BY timestamp DESC LIMIT {points}`
    - Parse results do slice
    - Return w reverse order (oldest first)

### 8.2 Metrics endpoint
- [ ] `internal/api/metrics.go`:
  - `GET /api/metrics/history/servers/:uuid?range=1m` (protected):
    - Sumuj metryki wszystkich kontenerów dla danego serwera
    - Query przez `GetHistoricalMetrics()` dla każdego kontenera
    - Aggregate (suma CPU, suma Memory, etc)
    - Response: `{"points": [{"timestamp": ..., "cpu": ..., "memory": ...}]}`
  - `GET /api/metrics/history/servers/:uuid/containers/:id?range=1m` (protected):
    - Query dla konkretnego kontenera
    - Response: analogicznie

### 8.3 Range validation
- [ ] W `internal/api/metrics.go`:
  - Middleware walidujący `?range` parameter
  - Allowed values: `1m, 5m, 15m, 30m, 1h, 6h, 12h, 24h, 7d, 15d, 30d`
  - Jeśli invalid → 400 Bad Request

### 8.4 Test historical API
- [ ] `tests/api_metrics_test.go`:
  - Test z fake data w różnych tabelach
  - Test każdego range
  - Test agregacji per-server vs per-container
  - Test invalid range parameter
  - Test non-existing server/container

---

## 📡 ETAP 9: SSE (Server-Sent Events)

### 9.1 SSE handler
- [ ] `internal/api/sse.go`:
  - `GET /api/metrics/live/all` (protected):
    - Upgrade do SSE
    - Co 1s:
      - Dla każdego approved servera:
        - Pobierz dane z RAM bufferów (ostatnie punkty)
        - Agreguj per-server (suma wszystkich kontenerów)
      - Format SSE: `data: {"servers": [{"uuid": "...", "cpu": ..., "memory": ...}]}\n\n`
      - Flush response
    - Obsługa disconnect
  - `GET /api/metrics/live/servers/:uuid` (protected):
    - Analogicznie, ale tylko dla jednego serwera
    - Breakdown per-container: `{"server": {...}, "containers": [...]}`

### 9.2 Buffer reader
- [ ] W `internal/buffer/manager.go`:
  - Metoda `GetLatestForServer(agentUUID string) map[string]MetricPoint` - zwraca ostatni punkt dla każdego kontenera
  - Metoda `GetLatestForContainer(agentUUID, containerID string) *MetricPoint`

### 9.3 Test SSE
- [ ] `tests/api_sse_test.go`:
  - Test SSE connection (HTTP client z SSE parsing)
  - Test stream (dodaj dane do buffera → sprawdź czy SSE wysłało)
  - Test disconnect handling
  - Test rate (czy faktycznie co 1s)

---

## 🧪 ETAP 10: Integration tests

### 10.1 End-to-end test
- [ ] `tests/integration_test.go`:
  - Setup: uruchom backend (in-memory DB)
  - Test full flow:
    1. Setup user (POST /api/setup)
    2. Login (POST /api/login)
    3. Connect mock agent (WebSocket)
    4. Send auth message (approved=0)
    5. Approve server (PUT /api/servers/:uuid/approve)
    6. Send metrics (przez WebSocket)
    7. Sprawdź RAM buffer
    8. Poczekaj 11s (bulk insert powinien zadziałać)
    9. Sprawdź metrics_1s w DB
    10. Query historical metrics (GET /api/metrics/history)
    11. Connect SSE (GET /api/metrics/live)
    12. Send command (POST /api/servers/:uuid/command)
    13. Poczekaj na agregację
    14. Sprawdź metrics_5s w DB

### 10.2 Stress test
- [ ] `tests/stress_test.go`:
  - Symuluj 5 agentów × 10 kontenerów = 50 połączeń
  - Wysyłaj metryki co 1s przez 2 minuty
  - Sprawdź:
    - Czy wszystkie metryki zapisały się
    - Czy agregacja działa poprawnie
    - Czy nie ma memory leaks
    - Czy SSE obsługuje wielu klientów

### 10.3 Concurrency test
- [ ] `tests/concurrency_test.go`:
  - Test równoczesnych zapisów do bufferów
  - Test bulk inserter podczas równoczesnych adds
  - Test agregacji podczas bulk inserts
  - Test WebSocket disconnects podczas operacji

---

## 🚀 ETAP 11: Main application

### 11.1 Server setup
- [ ] `cmd/server/main.go`:
  - Load config
  - Initialize database + migracje
  - Create AgentHub, BufferManager, Aggregator, BulkInserter
  - Setup HTTP router (gorilla/mux):
    ```
    /api/setup          → auth.HandleSetup
    /api/login          → auth.HandleLogin
    /api/auth/status    → auth.HandleStatus
    /api/servers        → auth.Protected(servers.HandleList)
    /api/servers/:uuid  → auth.Protected(servers.HandleGet)
    /api/servers/:uuid/approve → auth.Protected(servers.HandleApprove)
    /api/servers/:uuid/command → auth.Protected(commands.HandleCommand)
    /api/metrics/history/... → auth.Protected(metrics.HandleHistory)
    /api/metrics/live/... → auth.Protected(sse.HandleSSE)
    /ws/agent           → websocket.HandleAgent
    ```
  - Start goroutines:
    - `agentHub.Run()`
    - `bulkInserter.Run()`
    - `aggregator.Run()`
  - Start HTTP server
  - Graceful shutdown (SIGINT/SIGTERM)

### 11.2 Graceful shutdown
- [ ] W `main.go`:
  - Catch signals
  - Stop accepting new connections
  - Close WebSocket connections (send close frame)
  - Stop goroutines (BulkInserter, Aggregator)
  - Flush buffers do DB (final bulk insert)
  - Close database
  - Exit

### 11.3 Logging
- [ ] Dodaj structured logging (np. `log/slog` z Go 1.21+):
  - Log level z env var (DEBUG, INFO, WARN, ERROR)
  - Log wszystkie HTTP requests (middleware)
  - Log WebSocket connections/disconnections
  - Log agregacji (ile punktów, ile czasu)
  - Log bulk inserts (ile punktów, errors)

---

## 🐳 ETAP 12: Docker i deployment

### 12.1 Dockerfile
- [ ] `backend/Dockerfile`:
  - Multi-stage build:
    - Stage 1: Build (golang:1.24-alpine, CGO_ENABLED=1 dla SQLite)
    - Stage 2: Runtime (alpine:latest)
  - Expose port 8080
  - Volume: `/data` (dla SQLite DB)
  - Entrypoint: `/app/server`

### 12.2 Docker Compose
- [ ] `backend/docker-compose.yml`:
  ```yaml
  services:
    backend:
      build: .
      ports:
        - "8080:8080"
      volumes:
        - ./data:/data
      environment:
        - PORT=8080
        - DATABASE_PATH=/data/backend.db
        - JWT_SECRET=${JWT_SECRET}
  ```

### 12.3 Env vars
- [ ] `.env.example`:
  - `PORT=8080`
  - `DATABASE_PATH=/data/backend.db`
  - `JWT_SECRET=change-me-in-production`

---

## 📝 ETAP 13: Dokumentacja

### 13.1 README
- [ ] `backend/README.md`:
  - Opis projektu
  - Wymagania (Go 1.24+)
  - Setup instrukcje:
    - `go mod download`
    - `go run cmd/server/main.go`
  - Env vars
  - API endpoints (tabela)
  - WebSocket protocol (przykłady wiadomości)

### 13.2 API docs
- [ ] `backend/docs/API.md`:
  - Wszystkie endpointy z przykładami request/response
  - Auth flow (setup → login → token usage)
  - WebSocket protocol spec
  - SSE format
  - Error codes

### 13.3 Architecture doc
- [ ] `backend/docs/ARCHITECTURE.md`:
  - Diagram komponentów
  - Flow metryk: Agent → WebSocket → RAM Buffer → Bulk Insert → DB → Agregacja
  - Schema bazy (ERD)
  - Agregacja levels (tabela)

---

## ✅ ETAP 14: Finalizacja

### 14.1 Code review checklist
- [ ] Wszystkie testy przechodzą
- [ ] Go vet bez warningów
- [ ] Go fmt na wszystkich plikach
- [ ] Error handling wszędzie
- [ ] Brak race conditions (go test -race)
- [ ] Brak memory leaks (pprof)
- [ ] Graceful shutdown działa
- [ ] Logs są czytelne
- [ ] README kompletny

### 14.2 Performance check
- [ ] Benchmark bulk insert (ile punktów/s)
- [ ] Benchmark agregacji (ile czasu na poziom)
- [ ] Sprawdź memory usage (50 kontenerów × 2 min danych)
- [ ] Sprawdź DB size po 24h symulacji

### 14.3 Deploy test
- [ ] Build Docker image
- [ ] Uruchom przez docker-compose
- [ ] Connect 1 agent (realny)
- [ ] Sprawdź flow przez 10 minut
- [ ] Sprawdź logs
- [ ] Sprawdź metryki

---
