# Agent Monitorujący - Dokumentacja

Agent monitorujący zasoby systemowe i kontenery Docker napisany w języku Go.

## Spis treści

- [Instalacja](#instalacja)
- [Uruchomienie](#uruchomienie)
- [Komendy CLI](#komendy-cli)
- [Struktura danych](#struktura-danych)
- [Konfiguracja](#konfiguracja)

## Instalacja

### Wymagania

- Go 1.24+
- Docker (lokalny socket)

### Budowanie

```bash
cd agent
go build -o agent .
```

## Uruchomienie

### Podstawowe uruchomienie

```bash
./agent
```

Zbiera statystyki i zapisuje do pliku `data/stats.json`.

## Komendy CLI

### stats

Zbiera i zapisuje statystyki do JSON (domyślne zachowanie).

```bash
./agent stats
```

### info

Wyświetla informacje o systemie (bez metryk).

```bash
./agent info
```

Zwracane informacje:
- Hostname
- Platforma i OS
- Wersja kernela
- Architektura
- Informacje o CPU (model, rdzenie, cache)
- Pamięć RAM i swap
- Uptime systemu
- Czas bootowania
- Liczba procesów
- Host ID

### ws

Uruchamia agenta w trybie WebSocket - łączy się z backendem i wysyła metryki w czasie rzeczywistym.

```bash
./agent ws
./agent --backend-url ws://192.168.0.10:8080 ws
```

Wymaga ustawienia `BACKEND_URL` (zmienna środowiskowa) lub flagi `--backend-url`.

### stop

Zatrzymuje kontener.

```bash
./agent stop <nazwa-lub-id>
```

Przykład:
```bash
./agent stop server-agent
./agent stop fb629436cc81
```

### start

Uruchamia zatrzymany kontener.

```bash
./agent start <nazwa-lub-id>
```

### restart

Restartuje kontener.

```bash
./agent restart <nazwa-lub-id>
```

### check-updates

Sprawdza dostępność aktualizacji.

```bash
# Sprawdź wszystkie kontenery i grupy compose
./agent check-updates

# Sprawdź konkretny kontener lub projekt
./agent check-updates beszel-agent
./agent check-updates dashboard
```

Algorytm sprawdzania aktualizacji:
- porównanie lokalnego digestu obrazu z digestem z rejestru (digest-first),
- bez heurystyk opartych o tekst z `docker pull --dry-run`,
- status wyniku: `up_to_date`, `update_available`, `rate_limited`, `local`, `unknown`.

Status `unknown` oznacza, że nie udało się wiarygodnie sprawdzić aktualizacji (np. problem z dostępem do registry, autoryzacją lub manifestem).

Status `rate_limited` oznacza, że rejestr Docker ograniczył liczbę zapytań (np. Docker Hub rate limit).

Status `local` oznacza, że obraz jest zbudowany lokalnie i nie ma odpowiednika w zdalnym rejestrze.

### update

Aktualizuje kontener lub grupę compose.

```bash
# Aktualizuj pojedynczy kontener
./agent update <nazwa-kontenera>

# Aktualizuj całą grupę compose
./agent update <nazwa-projektu>
```

## Struktura danych

## WebSocket (tryb `ws`)

Po połączeniu agent wysyła co 1s wiadomość typu `metrics` z lekkim payloadem realtime.

### `metrics` (co 1s)

- `system` - metryki hosta (CPU, RAM, dysk, sieć)
- `docker.timestamp`
- `docker.containers[]` tylko dla kontenerów `running`:
  - `id`
  - `name`
  - `status`
  - `state`
  - `stats` (cpu, memory, block_io, network, pids)

W payloadzie 1s celowo **nie ma**: `network_info`, `labels`, `compose_groups`, `standalone_containers`.

### Akcje WebSocket (request/response)

- `stats` - pełne metryki (system + pełny Docker payload)
- `info` - informacje statyczne o systemie
- `docker-details` - pełne detale Docker (`network_info`, `labels`, `compose_groups`, `standalone_containers`)
- `start` - uruchomienie kontenera (wymaga `target`)
- `stop` - zatrzymanie kontenera (wymaga `target`)
- `restart` - restart kontenera (wymaga `target`)
- `check-updates` - sprawdzenie aktualizacji (opcjonalny `target` - kontener lub projekt)
- `update` - aktualizacja kontenera lub grupy compose (wymaga `target`)

### Info (system info)

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

### Stats.json

```json
{
  "system": {
    "timestamp": "2026-02-13T12:00:00Z",
    "cpu": {
      "percent": 15.5,
      "count": 20,
      "per_cpu_percent": [15.5]
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
        "interface": "enp5s0",
        "bytes_sent": 109128490,
        "bytes_recv": 3730535397,
        "packets_sent": 694930,
        "packets_recv": 2859444
      }
    ]
  },
  "docker": {
    "timestamp": "2026-02-13T12:00:00Z",
    "containers": [
      {
        "id": "fe560dd35cb2",
        "name": "beszel-agent",
        "image": "henrygd/beszel-agent-nvidia",
        "status": "Up 2 hours",
        "state": "running",
        "created": 1770730707,
        "stats": {
          "cpu": {
            "percent": 0.15,
            "cpu_container": 7113262000,
            "cpu_system": 93360800000000,
            "cpu_user": 0,
            "online_cpus": 20
          },
          "memory": {
            "usage": 59637760,
            "limit": 32673112064,
            "percent": 0.18,
            "cache": 0,
            "rss": 0,
            "swap": 0
          },
          "block_io": {
            "read_bytes": 35725312,
            "write_bytes": 28672
          },
          "network": {
            "rx_bytes": 0,
            "tx_bytes": 0,
            "rx_packets": 0,
            "tx_packets": 0,
            "rx_errors": 0,
            "tx_errors": 0
          },
          "pids": 14
        },
        "network_info": {...},
        "project": "docker",
        "service": "beszel-agent",
        "labels": {
          "com.docker.compose.project": "docker",
          "com.docker.compose.service": "beszel-agent",
          "org.opencontainers.image.version": "0.18.3"
        }
      }
    ],
    "compose_groups": [
      {
        "name": "docker",
        "project": "docker",
        "working_dir": "/home/user/Docker",
        "containers": [...]
      }
    ],
    "standalone_containers": [...]
  }
}
```

### Opis pól

#### System

| Pole | Typ | Opis |
|------|-----|------|
| timestamp | string | Czas zebrania metryk |
| cpu.percent | float64 | Procentowe użycie CPU |
| cpu.count | int | Liczba CPU |
| memory.total | uint64 | Całkowita pamięć RAM |
| memory.available | uint64 | Dostępna pamięć RAM |
| memory.used | uint64 | Używana pamięć RAM |
| memory.percent | float64 | Procent użycia RAM |
| disk[].device | string | Urządzenie dyskowe |
| disk[].mountpoint | string | Punkt montowania |
| disk[].total | uint64 | Całkowity rozmiar |
| disk[].free | uint64 | Wolne miejsce |
| disk[].used | uint64 | Używane miejsce |
| disk[].percent | float64 | Procent użycia |
| network[].interface | string | Nazwa interfejsu |
| network[].bytes_sent | uint64 | Wysłane bajty |
| network[].bytes_recv | uint64 | Odebrane bajty |
| network[].packets_sent | uint64 | Wysłane pakiety |
| network[].packets_recv | uint64 | Odebrane pakiety |

#### Docker

| Pole | Typ | Opis |
|------|-----|------|
| containers[].id | string | ID kontenera (pierwsze 12 znaków) |
| containers[].name | string | Nazwa kontenera |
| containers[].image | string | Obraz Docker |
| containers[].status | string | Status kontenera |
| containers[].state | string | Stan (running, exited, paused) |
| containers[].created | int64 | Timestamp utworzenia |
| containers[].stats | object | Statystyki kontenera |
| containers[].stats.cpu.percent | float64 | Procentowe użycie CPU |
| containers[].stats.cpu.cpu_container | float64 | Użycie CPU kontenera (ns) |
| containers[].stats.cpu.cpu_system | float64 | Użycie CPU systemu (ns) |
| containers[].stats.cpu.cpu_user | float64 | Użycie CPU użytkownika (ns) |
| containers[].stats.cpu.online_cpus | int64 | Liczba dostępnych CPU |
| containers[].stats.memory.usage | uint64 | Użycie pamięci |
| containers[].stats.memory.limit | uint64 | Limit pamięci (0 = brak) |
| containers[].stats.memory.percent | float64 | Procent użycia pamięci |
| containers[].stats.memory.cache | uint64 | Pamięć cache |
| containers[].stats.memory.rss | uint64 | Pamięć RSS |
| containers[].stats.memory.swap | uint64 | Pamięć swap |
| containers[].stats.block_io.read_bytes | uint64 | Odczytane bajty |
| containers[].stats.block_io.write_bytes | uint64 | Zapisane bajty |
| containers[].stats.network.rx_bytes | uint64 | Odebrane bajty |
| containers[].stats.network.tx_bytes | uint64 | Wysłane bajty |
| containers[].stats.network.rx_packets | uint64 | Odebrane pakiety |
| containers[].stats.network.tx_packets | uint64 | Wysłane pakiety |
| containers[].stats.network.rx_errors | uint64 | Błędy odbioru |
| containers[].stats.network.tx_errors | uint64 | Błędy wysyłania |
| containers[].stats.pids | int64 | Liczba procesów |
| containers[].network_info | object | Informacje o sieci |
| containers[].project | string | Projekt compose |
| containers[].service | string | Usługa compose |
| containers[].labels | object | Filtrowane etykiety |
| compose_groups[] | array | Grupy kontenerów compose |
| standalone_containers[] | array | Kontenery bez compose |

## Konfiguracja

### Zmienne środowiskowe

| Zmienna | Opis |
|---------|------|
| `BACKEND_URL` | URL serwera WebSocket dla komendy `ws` (np. `ws://192.168.0.10:8080`) |

### Flagi CLI

| Flaga | Opis |
|-------|------|
| `--backend-url` | URL serwera WebSocket (alternatywa dla `BACKEND_URL`) |

### Filtrowanie danych

Agent automatycznie filtruje:

#### Dysk
- Ignorowane: `/boot/efi`, `/boot`, `/run`, `/run/lock`, `/snap`, `/sys`, `/proc`, `/dev`, `/dev/shm`

#### Sieć (system)
- Ignorowane: `lo`, `virbr*`, `docker0`, `br-*`, `veth*`, `tailscale0`

#### Labels (kontenery)
Zachowane tylko przydatne do aktualizacji:
- `com.docker.compose.project`
- `com.docker.compose.service`
- `com.docker.compose.project.working_dir`
- `com.docker.compose.config-hash`
- `maintainer`
- `org.opencontainers.image.version`
- `org.opencontainers.image.revision`
- `org.opencontainers.image.source`
- `org.opencontainers.image.title`

## Docker Compose

Agent wykrywa kontenery z docker-compose i grupuje je automatycznie na podstawie etykiet:
- `com.docker.compose.project` - nazwa projektu
- `com.docker.compose.service` - nazwa usługi

### Aktualizacja grupy compose

```bash
./agent update nazwa-projektu
```

Polecenie:
1. Wykonuje `docker compose pull` w katalogu projektu
2. Wykonuje `docker compose up -d` aby odtworzyć kontenery

## Integracja z Backendem

Agent jest przygotowany do integracji z backendem przez WebSocket.

- Kanał `metrics` jest zoptymalizowany pod realtime i ma mały payload.
- Szczegółowe dane Docker są dostępne na żądanie przez akcję `docker-details`.
