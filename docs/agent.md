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

- Go 1.21+
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

Zbiera statystyki i zapisuje do pliku `stats.json`.

### Określenie pliku wyjściowego

```bash
./agent custom-stats.json
```

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
- status wyniku: `up_to_date`, `update_available`, `unknown`.

Status `unknown` oznacza, że nie udało się wiarygodnie sprawdzić aktualizacji (np. problem z dostępem do registry, autoryzacją lub manifestem).

### update

Aktualizuje kontener lub grupę compose.

```bash
# Aktualizuj pojedynczy kontener
./agent update <nazwa-kontenera>

# Aktualizuj całą grupę compose
./agent update <nazwa-projektu>
```

## Struktura danych

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
            "online_cpus": 20
          },
          "memory": {
            "usage": 59637760,
            "limit": 32673112064,
            "percent": 0.18
          },
          "block_io": {
            "read_bytes": 35725312,
            "write_bytes": 28672
          },
          "network": {
            "rx_bytes": 0,
            "tx_bytes": 0,
            "rx_packets": 0,
            "tx_packets": 0
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
| containers[].created | int64 | Timestamp创建ania |
| containers[].stats | object | Statystyki kontenera |
| containers[].stats.cpu | object | Statystyki CPU |
| containers[].stats.memory | object | Statystyki pamięci |
| containers[].stats.block_io | object | Statystyki I/O |
| containers[].stats.network | object | Statystyki sieci |
| containers[].stats.pids | int64 | Liczba procesów |
| containers[].network_info | object | Informacje o sieci |
| containers[].project | string | Projekt compose |
| containers[].service | string | Usługa compose |
| containers[].labels | object | Filtrowane etykiety |
| compose_groups[] | array | Grupy kontenerów compose |
| standalone_containers[] | array | Kontenery bez compose |

## Konfiguracja

### Filtrowanie danych

Agent automatycznie filtruje:

#### Dysk
- Ignorowane: `/boot/efi`, `/boot`, `/run`, `/run/lock`, `/snap`, `/sys`, `/proc`, `/dev`, `/dev/shm`

#### Sieć (system)
- Ignorowane: `lo`, `virbr*`, `docker0`, `br-*`, `veth-*`, `tailscale0`

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

Agent jest przygotowany do przyszłej integracji z backendem przez WebSocket. Struktura danych jest zoptymalizowana pod kątem przesyłania w czasie rzeczywistym.
