# Agent Monitorujący - Dokumentacja

Agent monitorujący zasoby systemowe i kontenery Docker napisany w języku Go.

## Spis treści

- [Instalacja](#instalacja)
- [Uruchomienie](#uruchomienie)
- [Komendy CLI](#komendy-cli)
- [Tryb WebSocket](#tryb-websocket)
- [Struktury danych](#struktury-danych)
- [Konfiguracja](#konfiguracja)

---

## Instalacja

### Wymagania

- Go 1.24+
- Docker (lokalny socket `/var/run/docker.sock`)

### Budowanie

```bash
cd agent
go build -o agent .
```

---

## Uruchomienie

```bash
./agent          # domyślnie: zbiera metryki i zapisuje do data/stats.json
./agent ws       # tryb WebSocket — łączy się z backendem i streamuje metryki
```

---

## Komendy CLI

| Komenda | Opis |
|---------|------|
| `stats` | Zbiera metryki i zapisuje do `data/stats.json` (domyślne zachowanie) |
| `info` | Wyświetla statyczne informacje o systemie |
| `ws` | Uruchamia agenta w trybie WebSocket |
| `stop <cel>` | Zatrzymuje kontener |
| `compose-stop <projekt>` | Zatrzymuje projekt compose |
| `start <cel>` | Uruchamia zatrzymany kontener |
| `compose-start <projekt>` | Uruchamia projekt compose |
| `restart <cel>` | Restartuje kontener |
| `compose-restart <projekt>` | Restartuje projekt compose |
| `check-updates [cel]` | Sprawdza dostępność aktualizacji obrazów |
| `update <cel>` | Aktualizuje kontener lub grupę compose |

### stats

```bash
./agent stats
```

Zbiera metryki systemu i Dockera, zapisuje do `data/stats.json`.

### info

```bash
./agent info
```

Zwraca statyczne informacje: hostname, platforma, OS, kernel, architektura, CPU (model, rdzenie, cache), RAM, swap, uptime, boot time, liczba procesów, Host ID.

### ws

```bash
./agent ws
./agent --backend-url ws://192.168.0.10:8080/ws/agent ws
```

Wymaga ustawionego `BACKEND_URL` lub flagi `--backend-url`.

### stop / start / restart (wraz z wersjami compose-*)

```bash
./agent stop <nazwa-lub-id>
./agent start <nazwa-lub-id>
./agent restart <nazwa-lub-id>

./agent compose-stop <nazwa-projektu>
./agent compose-start <nazwa-projektu>
./agent compose-restart <nazwa-projektu>
```

Przykłady:
```bash
./agent stop nginx
./agent restart fb629436cc81
./agent compose-restart moj-projekt
```

### check-updates

```bash
./agent check-updates                  # wszystkie kontenery i projekty compose
./agent check-updates nginx            # konkretny kontener
./agent check-updates moj-projekt      # konkretna grupa compose
```

Algorytm oparty o porównanie digestów obrazów (bez `docker pull --dry-run`).

| Status | Opis |
|--------|------|
| `up_to_date` | Obraz jest aktualny |
| `update_available` | Dostępna nowsza wersja |
| `rate_limited` | Docker Hub ograniczył zapytania |
| `local` | Obraz zbudowany lokalnie, brak odpowiednika w rejestrze |
| `unknown` | Nie udało się sprawdzić (problem z registry, autoryzacją lub manifestem) |

### update

```bash
./agent update <nazwa-kontenera>    # pojedynczy kontener
./agent update <nazwa-projektu>     # cała grupa compose
```

Priorytet wyszukiwania: najpierw exact match na nazwę projektu compose, potem na nazwę kontenera.

Dla grupy compose używa bezpośrednio Docker API, nie wymaga CLI `docker-compose`:
1. Równoległe pobranie najnowszych obrazów
2. Porównanie digestów obrazów
3. Sekwencyjne (Stop → Remove → Create → Start) zrekonstruowanie kontenerów, dla których dostępna jest aktualizacja

---

## Tryb WebSocket

### UUID Agenta

UUID jest generowany deterministycznie z `host_id` systemu (SHA256):

```
agent_uuid = hex(SHA256(host_id))
```

Ten sam serwer zawsze generuje ten sam UUID — identyfikuje serwer w backendzie nawet po restarcie agenta.

### Lifecycle połączenia

```
Agent                          Backend
  │                               │
  │──── TCP Connect ─────────────►│
  │──── auth (JSON) ─────────────►│  rejestracja/upsert w DB
  │◄─── auth_response ────────────│  zwykle approved: true (auto-approve)
  │                               │
  │  [jeśli approved: false]      │
  │──── close + wait 60s ────────►│  serwer odrzucony, ponowna próba później
  │──── TCP Connect ─────────────►│
  │──── auth ────────────────────►│
  │──── metrics (co 1s) ─────────►│
  │──── metrics (co 1s) ─────────►│
  │         ...                   │
  │◄─── ping (co 30s) ────────────│
  │──── pong ────────────────────►│
  │                               │
  │  [rozłączenie]                │
  │      oczekiwanie 10s          │
  │──── TCP Connect ─────────────►│  automatyczny reconnect
  │──── auth ────────────────────►│  auth ponawiane po każdym reconnect
```

### Limity i interwały

| Parametr | Wartość |
|----------|---------|
| Interwał metryk | 1s |
| Ping (agent → backend) | co 54s (pongWait * 9/10) |
| Read deadline | 60s (resetowane przy każdym pong) |
| Max rozmiar wiadomości | brak (domyślne 32KB od Gorilla WebSocket) |
| Auth timeout | 30s |
| Opóźnienie reconnect | 10s |

---

## Struktury danych

### WebSocket — wiadomości

#### `auth` (agent → backend)

Wysyłana natychmiast po połączeniu. Zawiera UUID oraz statyczne informacje o serwerze.

```json
{
  "type": "auth",
  "uuid": "a3f2c1d4e5b6...",
  "info": {
    "hostname": "server1",
    "cpu_model": "13th Gen Intel(R) Core(TM) i5-13600KF",
    "cpu_cores": 20,
    "memory_total": 32673112064,
    "platform": "Ubuntu 25.10",
    "kernel": "6.12.62-x64v3-xanmod1",
    "architecture": "x86_64"
  }
}
```

| Pole | Typ | Opis |
|------|-----|------|
| `type` | string | Zawsze `"auth"` |
| `uuid` | string | SHA256 z host_id — unikalny identyfikator serwera |
| `info.hostname` | string | Nazwa hosta |
| `info.cpu_model` | string | Model procesora |
| `info.cpu_cores` | int | Liczba rdzeni logicznych |
| `info.memory_total` | uint64 | Całkowita pamięć RAM (bajty) |
| `info.platform` | string | Nazwa i wersja OS |
| `info.kernel` | string | Wersja kernela |
| `info.architecture` | string | Architektura (np. `x86_64`) |

---

#### `auth_response` (backend → agent)

```json
{
  "type": "auth_response",
  "approved": true
}
```

Jeśli `approved: true` — agent od razu uruchamia pętlę metryk.

Jeśli `approved: false` — agent zamyka połączenie i ponawia próbę po ~60s (nie czeka pasywnie na drugi `auth_response` w tym samym połączeniu).

---

#### `metrics` (agent → backend, co 1s)

Kanoniczna wiadomość wysyłana w pętli co 1 sekundę. Zawiera snapshot hosta i listę uruchomionych kontenerów.

```json
{
  "type": "metrics",
  "timestamp": 1739967600,
  "host": {
    "timestamp": 1739967600,
    "cpu_percent": 12.5,
    "mem_used": 10622971904,
    "mem_percent": 32.5,
    "disk_read_bytes_per_sec": 1048576,
    "disk_write_bytes_per_sec": 524288,
    "net_rx_bytes_per_sec": 204800,
    "net_tx_bytes_per_sec": 102400,
    "disk_read_bytes_total": 987654321,
    "disk_write_bytes_total": 123456789,
    "net_rx_bytes_total": 5000000000,
    "net_tx_bytes_total": 1000000000
  },
  "containers": [
    {
      "container_id": "fb629436cc81",
      "name": "nginx",
      "image": "nginx:latest",
      "project": "webstack",
      "service": "nginx",
      "state": "running",
      "timestamp": 1739967600,
      "cpu_percent": 0.5,
      "mem_used": 52428800,
      "mem_percent": 0.16,
      "disk_used": 10485760,
      "disk_percent": 0.0,
      "net_rx_bytes": 1024000,
      "net_tx_bytes": 512000
    }
  ]
}
```

**Pole `host`:**

| Pole | Typ | Opis |
|------|-----|------|
| `timestamp` | int64 | Unix timestamp snapshotu |
| `cpu_percent` | float64 | Użycie CPU hosta (%) |
| `mem_used` | uint64 | Użyta pamięć RAM (bajty) |
| `mem_percent` | float64 | Użycie RAM (%) |
| `disk_read_bytes_per_sec` | uint64 | Odczyty dysku (bajty/s) — delta od poprzedniego snapshotu |
| `disk_write_bytes_per_sec` | uint64 | Zapisy dysku (bajty/s) — delta |
| `net_rx_bytes_per_sec` | uint64 | Odbiór sieciowy (bajty/s) — delta |
| `net_tx_bytes_per_sec` | uint64 | Wysyłanie sieciowe (bajty/s) — delta |
| `disk_read_bytes_total` | uint64 | Łączne odczyty dysku od startu systemu |
| `disk_write_bytes_total` | uint64 | Łączne zapisy dysku od startu systemu |
| `net_rx_bytes_total` | uint64 | Łączny odbiór sieciowy od startu systemu |
| `net_tx_bytes_total` | uint64 | Łączne wysyłanie sieciowe od startu systemu |

**Pola `containers[]`:**

| Pole | Typ | Opis |
|------|-----|------|
| `container_id` | string | ID kontenera (skrócone) |
| `name` | string | Nazwa kontenera |
| `image` | string | Obraz Docker |
| `project` | string | Projekt compose (pusty jeśli standalone) |
| `service` | string | Usługa compose (pusty jeśli standalone) |
| `state` | string | Stan kontenera (`running`, `exited`, `paused`) |
| `timestamp` | int64 | Unix timestamp snapshotu kontenera |
| `cpu_percent` | float64 | Użycie CPU (%) |
| `mem_used` | uint64 | Użyta pamięć (bajty) |
| `mem_percent` | float64 | Użycie pamięci (%) |
| `disk_used` | uint64 | Użyty dysk (bajty) |
| `disk_percent` | float64 | Użycie dysku (%) |
| `net_rx_bytes` | uint64 | Odebrane bajty (całkowite, nie delta) |
| `net_tx_bytes` | uint64 | Wysłane bajty (całkowite, nie delta) |

> Payload `metrics` zawiera **tylko kontenery w stanie `running`**. Nie zawiera: `network_info`, `labels`, `compose_groups`, `standalone_containers` — te dane dostępne są przez komendę `docker-details`.

---

#### `command` (backend → agent)

Backend może wysłać komendę do agenta w dowolnym momencie.

```json
{
  "type": "command",
  "command_id": "cmd-uuid-1234",
  "action": "restart",
  "target": "nginx",
  "args": {}
}
```

| Pole | Typ | Opis |
|------|-----|------|
| `type` | string | Zawsze `"command"` |
| `command_id` | string | UUID komendy — używany do dopasowania odpowiedzi |
| `action` | string | Akcja do wykonania (patrz tabela niżej) |
| `target` | string | Nazwa lub ID kontenera / projektu (wymagane dla większości akcji) |
| `args` | object | Dodatkowe argumenty dla akcji (opcjonalne) |

**Dostępne akcje:**

| Akcja | Target | Opis |
|-------|--------|------|
| `stats` | — | Pełne metryki (system + Docker) |
| `info` | — | Statyczne informacje o systemie |
| `docker-details` | — | Pełne dane Docker (labels, network_info, compose_groups, standalone) |
| `start` | container | Uruchamia kontener |
| `stop` | container | Zatrzymuje kontener |
| `restart` | container | Restartuje kontener |
| `check-updates` | container / project (opt.) | Sprawdza dostępność aktualizacji |
| `update` | container / project | Aktualizuje kontener lub grupę compose |

---

#### `response` (agent → backend)

Odpowiedź na komendę. `payload` zawiera obiekt z `command_id` i `result`.

```json
{
  "type": "response",
  "command_id": "cmd-uuid-1234",
  "payload": {
    "command_id": "cmd-uuid-1234",
    "result": { ... }
  }
}
```

---

### `info` — struktura odpowiedzi

Zwracana przez komendę `info` lub CLI `./agent info`.

```json
{
  "hostname": "server1",
  "platform": "Ubuntu 25.10",
  "os": "linux",
  "kernel": "6.12.62-x64v3-xanmod1",
  "architecture": "x86_64",
  "cpu": {
    "model_name": "13th Gen Intel(R) Core(TM) i5-13600KF",
    "vendor_id": "GenuineIntel",
    "physical_cores": 14,
    "logical_cores": 20,
    "mhz": 5100,
    "cache_size": 24576
  },
  "memory": {
    "total": 32673112064,
    "swap_total": 24926482432
  },
  "uptime": 7891,
  "boot_time": 1770974407,
  "num_procs": 502,
  "host_id": "9c8ac6ff-576e-4cde-9a39-ac82bf776159"
}
```

| Pole | Typ | Opis |
|------|-----|------|
| `hostname` | string | Nazwa hosta |
| `platform` | string | Nazwa i wersja OS |
| `os` | string | Rodzina OS (`linux`) |
| `kernel` | string | Wersja kernela |
| `architecture` | string | Architektura systemu |
| `cpu.model_name` | string | Model procesora |
| `cpu.vendor_id` | string | Producent CPU |
| `cpu.physical_cores` | int | Fizyczne rdzenie |
| `cpu.logical_cores` | int | Logiczne rdzenie (z hyperthreading) |
| `cpu.mhz` | float64 | Częstotliwość taktowania (MHz) |
| `cpu.cache_size` | int32 | Rozmiar cache L2 (KB) |
| `memory.total` | uint64 | Całkowita pamięć RAM (bajty) |
| `memory.swap_total` | uint64 | Całkowity swap (bajty) |
| `uptime` | uint64 | Czas pracy systemu (sekundy) |
| `boot_time` | uint64 | Unix timestamp ostatniego rozruchu |
| `num_procs` | int | Liczba uruchomionych procesów |
| `host_id` | string | Unikalny identyfikator hosta (podstawa UUID agenta) |

---

### `stats.json` — tryb offline

Zapisywany przez `./agent stats` do `data/stats.json`.

```json
{
  "system": {
    "timestamp": "2026-02-13T12:00:00Z",
    "cpu": {
      "percent": 15.5,
      "count": 20,
      "per_cpu_percent": [12.0, 18.0, 15.5]
    },
    "memory": {
      "total": 32673112064,
      "available": 22050140160,
      "used": 10622971904,
      "percent": 32.5
    },
    "disk": [
      {
        "device": "/dev/nvme0n1p5",
        "mountpoint": "/",
        "total": 689813372928,
        "free": 303142707200,
        "used": 351554801664,
        "percent": 53.7
      }
    ],
    "network": [
      {
        "interface": "eth0",
        "bytes_sent": 1000000,
        "bytes_recv": 5000000,
        "packets_sent": 10000,
        "packets_recv": 50000
      }
    ]
  },
  "docker": {
    "timestamp": "2026-02-13T12:00:00Z",
    "compose_groups": [
      {
        "name": "webstack",
        "project": "webstack",
        "working_dir": "/home/user/webstack",
        "containers": [...]
      }
    ],
    "standalone_containers": [...]
  }
}
```

---

## Konfiguracja

### Zmienne środowiskowe

| Zmienna | Opis |
|---------|------|
| `BACKEND_URL` | URL serwera WebSocket (np. `ws://192.168.0.10:8080/ws/agent`) |

### Flagi CLI

| Flaga | Opis |
|-------|------|
| `--backend-url` | URL serwera WebSocket (nadpisuje `BACKEND_URL`) |

---

## Filtrowanie danych

### Dyski — ignorowane punkty montowania

| Ścieżka |
|---------|
| `/boot/efi` |
| `/boot` |
| `/run` |
| `/run/lock` |
| `/snap` |
| `/sys` |
| `/proc` |
| `/dev` |
| `/dev/shm` |

### Interfejsy sieciowe — ignorowane

| Interfejs / prefiks |
|---------------------|
| `lo` |
| `virbr*` |
| `docker0` |
| `br-*` |
| `veth*` |
| `tailscale0` |

### Labels kontenerów — zachowane

Filtrowane do minimum potrzebnego do grupowania i sprawdzania aktualizacji:

| Label |
|-------|
| `com.docker.compose.project` |
| `com.docker.compose.service` |
| `com.docker.compose.project.working_dir` |
| `com.docker.compose.config-hash` |
| `maintainer` |
| `org.opencontainers.image.version` |
| `org.opencontainers.image.revision` |
| `org.opencontainers.image.source` |
| `org.opencontainers.image.title` |
