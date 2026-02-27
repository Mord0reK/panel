---
description: "Triggeruj ZAWSZE, gdy zadanie dotyczy zbierania metryk (CPU, RAM, Docker), modyfikacji kodu w `/agent/internal` lub protokołu komunikacji WebSocket po stronie klienta."
---

# Skill: Panel — Agent Go (Metryki, Docker, WebSocket)

## Cel

Ten skill definiuje standard pracy dla subagenta modyfikującego komponent **Agent** (`/agent/`). Agent to klient WebSocket napisany w Go, który co 1 sekundę zbiera metryki systemowe i dockerowe, a następnie wysyła je do backendu.

---

## Mapa modułu agenta (`module agent`)

```
agent/
├── main.go                      — punkt wejścia, obsługa komend CLI, pętla WS
└── internal/
    ├── collector/
    │   └── metrics.go           — CollectSystemMetrics, CollectSystemInfo
    ├── config/
    │   └── (konfiguracja)       — IsIgnoredMount, IsIgnoredNetworkInterface
    ├── docker/
    │   ├── container.go         — CollectContainerMetrics, CollectRealtimeContainerMetrics,
    │   │                          NewDockerClient, FindContainerByName, GroupContainersByCompose
    │   └── manager.go           — ContainerManager (start/stop/restart/update/check-updates)
    ├── metrics/
    │   └── snapshot.go          — SnapshotCollector, Collect (dedupe CPU delta + rate calculation)
    ├── output/
    │   └── (zapis JSON)
    └── websocket/
        └── client.go            — Client, AuthInfo, MetricsMessage, Command, AuthMessage
```

---

## Kluczowe sygnatury — zakaz zgadywania

Przed każdą modyfikacją **fizycznie odczytaj** odpowiedni plik. Nigdy nie zakładaj, że pamiętasz strukturę.

### `agent/internal/collector/metrics.go`

```go
func CollectSystemMetrics(ctx context.Context) (*SystemMetrics, error)
func CollectSystemInfo(ctx context.Context) (*SystemInfo, error)

type SystemMetrics struct {
    Timestamp time.Time      `json:"timestamp"`
    CPU       CPUStats       `json:"cpu"`
    Memory    MemoryStats    `json:"memory"`
    Disk      []DiskStats    `json:"disk"`
    Network   []NetworkStats `json:"network"`
}

// UWAGA: User/System/Idle mają json:"-" — nie są serializowane do JSON
type CPUStats struct {
    Percent       float64   `json:"percent"`
    Cores         int       `json:"cores"`
    Threads       int       `json:"threads"`
    PerCPUPercent []float64 `json:"per_cpu_percent"`
    User          float64   `json:"-"`
    System        float64   `json:"-"`
    Idle          float64   `json:"-"`
}
```

### `agent/internal/metrics/snapshot.go`

```go
func NewSnapshotCollector() *SnapshotCollector
func (c *SnapshotCollector) Collect(ctx context.Context, dockerCli *client.Client) (*Snapshot, error)

// HostSnapshot — wysyłany jako "host" w MetricsMessage
type HostSnapshot struct {
    Timestamp            int64   `json:"timestamp"`
    CPU                  float64 `json:"cpu_percent"`
    MemUsed              uint64  `json:"mem_used"`
    MemPercent           float64 `json:"mem_percent"`
    MemoryTotal          uint64  `json:"memory_total"`
    DiskReadBytesPerSec  uint64  `json:"disk_read_bytes_per_sec"`
    DiskWriteBytesPerSec uint64  `json:"disk_write_bytes_per_sec"`
    NetRxBytesPerSec     uint64  `json:"net_rx_bytes_per_sec"`
    NetTxBytesPerSec     uint64  `json:"net_tx_bytes_per_sec"`
    DiskUsedPercent      float64 `json:"disk_used_percent"`
    // Poniższe pola NIE są parsowane przez backend (brak w protocol.go HostMetrics):
    DiskReadBytesTotal   uint64  `json:"disk_read_bytes_total"`
    DiskWriteBytesTotal  uint64  `json:"disk_write_bytes_total"`
    NetRxBytesTotal      uint64  `json:"net_rx_bytes_total"`
    NetTxBytesTotal      uint64  `json:"net_tx_bytes_total"`
}
```

### `agent/internal/docker/container.go`

```go
func NewDockerClient() (*client.Client, error)
func CollectContainerMetrics(ctx context.Context, cli *client.Client) (*ContainerMetrics, error)
func CollectRealtimeContainerMetrics(ctx context.Context, cli *client.Client) (*RealtimeContainerMetrics, error)
func FindContainerByName(ctx context.Context, cli *client.Client, name string) (string, error)
func GroupContainersByCompose(containers []ContainerInfo) GroupedContainers

// RealtimeContainerInfo — element Snapshot.Containers
type RealtimeContainerInfo struct {
    ContainerID string  `json:"container_id"`
    Name        string  `json:"name"`
    Image       string  `json:"image"`
    Project     string  `json:"project"`
    Service     string  `json:"service"`
    State       string  `json:"state"`
    Timestamp   int64   `json:"timestamp"`
    CPU         float64 `json:"cpu_percent"`
    MemUsed     uint64  `json:"mem_used"`
    MemPercent  float64 `json:"mem_percent"`
    DiskUsed    uint64  `json:"disk_used"`
    DiskPercent float64 `json:"disk_percent"`
    NetRx       uint64  `json:"net_rx_bytes"`
    NetTx       uint64  `json:"net_tx_bytes"`
}
```

### `agent/internal/websocket/client.go`

```go
func NewClient(url string) *Client
func (c *Client) Connect(ctx context.Context) error
func (c *Client) SendMessage(msgType string, data interface{}) error
func (c *Client) SendAuth(uuid string, info AuthInfo) error
func (c *Client) WaitForAuthResponse(ctx context.Context) (bool, error)
func (c *Client) Listen(ctx context.Context, handler func(Command) error) error
func (c *Client) Close()
func (c *Client) IsConnected() bool
func (c *Client) Reconnect(ctx context.Context) error

type MetricsMessage struct {
    Type       string      `json:"type"`
    Timestamp  int64       `json:"timestamp"`
    Host       interface{} `json:"host,omitempty"`
    Containers interface{} `json:"containers,omitempty"`
}

type AuthInfo struct {
    Hostname     string `json:"hostname"`
    CPUModel     string `json:"cpu_model"`
    CPUCores     int    `json:"cpu_cores"`
    CPUThreads   int    `json:"cpu_threads"`  // zdefiniowane, ale NIE wypełniane w main.go — patrz asymetrie
    MemoryTotal  uint64 `json:"memory_total"`
    Platform     string `json:"platform"`
    Kernel       string `json:"kernel"`
    Architecture string `json:"architecture"`
}

type Command struct {
    Type      string          `json:"type"`
    CommandID string          `json:"command_id"`
    Action    string          `json:"action"`
    Target    string          `json:"target,omitempty"`
    Args      json.RawMessage `json:"args,omitempty"`
}
```

---

## Zależności zewnętrzne (`go.mod`)

| Pakiet | Wersja | Użycie |
|--------|--------|--------|
| `github.com/shirou/gopsutil/v4` | v4.26.1 | CPU, disk, host, mem, net |
| `github.com/moby/moby/client` | v0.2.2 | Docker API |
| `github.com/moby/moby/api` | v1.53.0 | Typy Docker API |
| `github.com/gorilla/websocket` | v1.5.3 | WS klient |

---

## Zasada Local Knowledge First

Zanim użyjesz Context7 do sprawdzania dokumentacji bibliotek zewnętrznych (`gopsutil`, `moby/client`, `gorilla/websocket`), najpierw sprawdź **bezpłatnymi narzędziami odczytu (Read tool)**:

1. Czy plik który modyfikujesz już importuje potrzebny podpakiet? (odczytaj importy)
2. Czy jest już precedens użycia tej funkcji w innym pliku w `/agent/internal/`? (Grep po projekcie)
3. Czy sygnatura jest oczywista z istniejącego kodu?

**Context7 jest wymagany tylko gdy:** żaden z powyższych kroków nie dał odpowiedzi — tzn. wchodzisz w nowy podpakiet bez precedensu w projekcie lub sygnatura jest niejednoznaczna.

---

## Zasada No-Guessing — procedura dodawania nowej statystyki

Gdy zadanie wymaga dodania nowego pola do metryk:

1. **Odczytaj fizycznie** `agent/internal/collector/metrics.go` — istniejące struktury
2. **Odczytaj fizycznie** `agent/internal/metrics/snapshot.go` — `HostSnapshot`, logika `Collect()`
3. **Local Knowledge First** — sprawdź czy gopsutil już jest używany dla podobnej metryki (Grep)
4. **Context7 jeśli potrzebne** — tylko gdy krok 3 nie dał odpowiedzi
5. **Sprawdź synchronizację WS** (patrz sekcja poniżej)
6. Dopiero potem pisz kod

---

## Synchronizacja protokołu WS — KRYTYCZNE

**Każda zmiana struktury wiadomości WS wymaga sprawdzenia `backend/internal/websocket/protocol.go`.**

### Mapa powiązań JSON

| Agent (klient) | Backend (serwer) | JSON key |
|----------------|------------------|----------|
| `websocket.AuthInfo` | `protocol.AgentInfo` | `info` w `auth` msg |
| `metrics.HostSnapshot` | `protocol.HostMetrics` | `host` w `metrics` msg |
| `docker.RealtimeContainerInfo` | `protocol.ContainerMetrics` | element `containers[]` |
| `websocket.Command` | `protocol.CommandMessage` | `command` msg |
| `websocket.MetricsMessage.Type` | `protocol.MsgTypeMetrics = "metrics"` | string stały |

### Znane asymetrie (stan aktualny)

| Pole | Agent | Backend | Skutek |
|------|-------|---------|--------|
| `cpu_threads` (auth) | `AuthInfo.CPUThreads` — **zdefiniowane, nie wypełniane** w `main.go` | `AgentInfo.CPUThreads` — zdefiniowane | Backend zawsze odbiera `0`; serwer rejestruje się z `cpu_threads=0` |
| `disk_*_bytes_total`, `net_*_bytes_total` | `HostSnapshot` — 4 pola `*Total` | `protocol.HostMetrics` — brak tych pól | Backend ignoruje te pola przy parsowaniu |

**Kontekst `CPUThreads`:** W `main.go` funkcja `runWebSocket()` konstruuje `AuthInfo` bez ustawiania `CPUThreads`, mimo że pole istnieje w obu strukturach. Jeśli zadanie dotyczy uzupełnienia tego pola — użyj `sysInfo.CPU.LogicalCores` z `CollectSystemInfo`.

### Procedura przy zmianie struktury wiadomości

Jeśli modyfikujesz JAKIKOLWIEK typ w `agent/internal/websocket/client.go` lub `agent/internal/metrics/snapshot.go`, który jest serializowany do JSON:

1. Otwórz `backend/internal/websocket/protocol.go`
2. Znajdź odpowiadający typ po stronie backendu
3. Zsynchronizuj pola i JSON tagi
4. Sprawdź czy `backend/internal/models/` wymaga aktualizacji
5. Sprawdź czy `frontend/types/index.ts` wymaga aktualizacji (przez skill `panel-analysis`)

---

## Pętla metryk — architektura runtime

```
main.go:runWebSocket()
  └─► metrics.NewSnapshotCollector()
  └─► go sendMetricsLoop(ctx, wsClient, dockerCli, snapshotCollector)
        └─► ticker każde 1s
              └─► snapshotCollector.Collect(ctx, dockerCli)
                    ├─► collector.CollectSystemMetrics(ctx)
                    ├─► gopsutil cpu.TimesWithContext()       — delta CPU%
                    ├─► collectNetworkTotals()                — delta net bytes/s
                    ├─► collectDiskIOTotals()                 — delta disk bytes/s
                    └─► docker.CollectRealtimeContainerMetrics()
              └─► wsClient.SendMessage("metrics", MetricsMessage{...})
```

**Uwaga:** `SnapshotCollector` jest stateful. Przy pierwszym `Collect()` wartości `*PerSec` wynoszą 0 — brak poprzedniego punktu odniesienia. Zachowanie zamierzone.

---

## Obsługa komend z backendu

| Action | Target | Opis |
|--------|--------|------|
| `stats` | — | Snapshot metryk |
| `info` | — | `CollectSystemInfo` |
| `docker-details` | — | `CollectContainerMetrics` |
| `stop` / `start` / `restart` | nazwa kontenera | `ContainerManager` |
| `compose-stop` / `compose-start` / `compose-restart` | nazwa projektu | `ContainerManager` |
| `check-updates` | kontener lub projekt | `CheckForUpdates` / `CheckComposeUpdates` |
| `update` | kontener lub projekt | `UpdateContainer` / `UpdateComposeGroup` |

---

## Stabilność — zasada `context.WithTimeout` dla operacji Docker

Operacje w `manager.go` są wywoływane z handlera komend WS — mogą blokować goroutine handlera. Operacje sieciowe (`DistributionInspect`, `ImagePull`) mogą trwać dziesiątki sekund.

### Wymagane timeouty przy dodawaniu nowych operacji

| Rodzaj operacji | Zalecany timeout |
|----------------|-----------------|
| `DistributionInspect` (zapytanie do registry) | `30s` |
| `ImagePull` (pobieranie obrazu) | `120s` |
| `ContainerList`, `ContainerInspect`, `ImageInspect` | `15s` |
| `ContainerStop`, `ContainerStart`, `ContainerRestart` | `30s` (Go context) + Docker-level timeout `10s` |

### Wzorzec użycia

```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
result, err := m.cli.DistributionInspect(ctx, image, ...)
```

**Uwaga:** Istniejący kod `manager.go` nie wszędzie stosuje ten wzorzec. Nie poprawiaj istniejącego kodu bez wyraźnego polecenia — stosuj regułę tylko przy **dodawaniu nowych operacji**.

---

## Self-Correction — obowiązkowe testy

### Po każdej zmianie kodu agenta

```bash
# Z katalogu /agent/
go test ./...

# Lub z korzenia repozytorium
go test ./agent/...

# Weryfikacja kompilacji
go build ./...
```

### Procedura naprawy błędów testów (3 próby samodzielnie)

**Próba 1:** Przeczytaj output błędu → zidentyfikuj plik i linię → napraw → uruchom testy.

**Próba 2:** Fizycznie odczytaj plik z błędem (Read tool) → sprawdź importy i sygnatury → napraw → uruchom testy.

**Próba 3:** Local Knowledge First (Grep) + Context7 jeśli potrzebne → napraw → uruchom testy.

### Raportowanie po wyczerpaniu 3 prób

Jeśli po 3 próbach testy nadal nie przechodzą, subagent **zatrzymuje pracę** i przygotowuje raport dla **Orchestratora**:

```
## Raport błędu testów: [nazwa zadania]

### Stan po 3 próbach naprawy
- Testy: FAIL
- Komenda: `go test ./...` (z `/agent/`)

### Output błędu (ostatnia próba)
[pełny output z go test]

### Próby naprawy
1. [Próba 1 — co sprawdzono, co zmieniono, jaki efekt]
2. [Próba 2 — co sprawdzono, co zmieniono, jaki efekt]
3. [Próba 3 — co sprawdzono, co zmieniono, jaki efekt]

### Hipoteza przyczyny
[Techniczna ocena dlaczego problem nie zostaje rozwiązany]

### Rekomendacja dla Orchestratora
[Sugerowana strategia lub pytanie do użytkownika przez AskUserQuestion]
```

**Orchestrator decyduje:** czy spróbować innej strategii, czy przekazać raport użytkownikowi przez `AskUserQuestion`. Subagent nie kontaktuje się bezpośrednio z użytkownikiem.

---

## Zasady bezwzględne

1. **Local Knowledge First** — sprawdź kod projektu (Read/Grep) przed sięgnięciem po Context7
2. **Context7 przy nowym API** — gdy Local Knowledge First nie wystarczy dla `gopsutil`, `moby/client`, `gorilla/websocket`
3. **Fizyczny odczyt przed modyfikacją** — nie modyfikuj struktury bez uprzedniego odczytu pliku
4. **Synchronizacja WS jest obowiązkowa** — zmiana w typach wiadomości = sprawdzenie `backend/internal/websocket/protocol.go`
5. **Testy po każdej zmianie** — `go test ./...` to warunek konieczny, nie opcja
6. **3 próby self-correction** — wyczerpaj własne próby przed eskalacją do Orchestratora
7. **Eskalacja do Orchestratora, nie do użytkownika** — subagent raportuje Orchestratorowi szczegółowy raport techniczny; Orchestrator decyduje o dalszym kroku
8. **`context.WithTimeout` dla nowych operacji Docker** — każda nowa operacja API w `manager.go` musi mieć odpowiedni timeout
9. **Raportuj breaking changes** — jeśli zmiana wymaga modyfikacji backendu lub frontendu, jawnie zaznacz to w raporcie do Orchestratora
