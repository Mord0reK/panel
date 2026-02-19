# API — Dokumentacja

Dokumentacja wszystkich endpointów REST, SSE oraz WebSocket backendu.

## Spis treści

- [Autentykacja](#autentykacja)
- [CORS](#cors)
- [Auth](#auth)
- [Serwery](#serwery)
- [Komendy](#komendy)
- [Metryki historyczne](#metryki-historyczne)
- [Metryki live (SSE)](#metryki-live-sse)
- [WebSocket — Agent](#websocket--agent)

---

## Autentykacja

Wszystkie endpointy z wyjątkiem `/api/setup`, `/api/login`, `/api/auth/status` i `/ws/agent` wymagają nagłówka:

```
Authorization: Bearer <token>
```

Token JWT ważny **7 dni**, podpisany HS256. Zawiera `user_id`. Brak lub nieprawidłowy token → `401 Unauthorized`.

---

## CORS

REST API i preflight (`OPTIONS`) — `Access-Control-Allow-Origin: *` zawsze, niezależnie od konfiguracji.

SSE — `Access-Control-Allow-Origin` ustawiane na wartość zmiennej `CORS_ORIGIN` (domyślnie `*`).

---

## Auth

### `POST /api/setup`

Pierwsze uruchomienie — tworzy konto administratora. Dostępne tylko gdy brak użytkowników w bazie. Kolejne wywołanie → `403 Forbidden`.

**Request:**
```json
{
  "username": "admin",
  "password": "haslo1234"
}
```

Walidacja: `username` min. 3 znaki, `password` min. 8 znaków.

**Response `200`:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

| Kod | Przyczyna |
|-----|-----------|
| `200` | Sukces — zwraca token (auto-login po setupie) |
| `400` | Za krótki username lub password |
| `403` | Setup już wykonany |
| `500` | Błąd bazy |

---

### `POST /api/login`

**Request:**
```json
{
  "username": "admin",
  "password": "haslo1234"
}
```

**Response `200`:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

| Kod | Przyczyna |
|-----|-----------|
| `200` | Sukces |
| `401` | Nieprawidłowe dane logowania |

---

### `GET /api/auth/status`

Sprawdza stan aplikacji i sesji. Nie wymaga tokenu — opcjonalnie przyjmuje nagłówek `Authorization` do weryfikacji.

**Response `200`:**
```json
{
  "setup_required": false,
  "authenticated": true
}
```

| Pole | Opis |
|------|------|
| `setup_required` | `true` gdy brak użytkowników — frontend powinien przekierować na stronę setup |
| `authenticated` | `true` gdy przesłany token jest ważny |

---

## Serwery

Wszystkie endpointy wymagają tokenu JWT.

### `GET /api/servers`

Lista wszystkich serwerów posortowana po hostname.

**Response `200`:**
```json
[
  {
    "uuid": "a3f2c1d4e5b6...",
    "hostname": "server1",
    "approved": true,
    "cpu_model": "Intel Core i5-13600KF",
    "cpu_cores": 20,
    "memory_total": 32673112064,
    "platform": "Ubuntu 25.10",
    "kernel": "6.12.62",
    "architecture": "x86_64",
    "last_seen": "2026-02-19T12:00:00Z",
    "created_at": "2026-01-01T00:00:00Z"
  }
]
```

Zwraca pustą tablicę `[]` gdy brak serwerów.

---

### `GET /api/servers/:uuid`

Szczegóły serwera wraz z listą jego kontenerów.

**Response `200`:**
```json
{
  "server": {
    "uuid": "a3f2c1d4e5b6...",
    "hostname": "server1",
    "approved": true,
    "cpu_model": "Intel Core i5-13600KF",
    "cpu_cores": 20,
    "memory_total": 32673112064,
    "platform": "Ubuntu 25.10",
    "kernel": "6.12.62",
    "architecture": "x86_64",
    "last_seen": "2026-02-19T12:00:00Z",
    "created_at": "2026-01-01T00:00:00Z"
  },
  "containers": [
    {
      "id": 1,
      "agent_uuid": "a3f2c1d4e5b6...",
      "container_id": "fb629436cc81",
      "name": "nginx",
      "image": "nginx:latest",
      "project": "webstack",
      "service": "nginx",
      "first_seen": "2026-01-01T00:00:00Z",
      "last_seen": "2026-02-19T12:00:00Z"
    }
  ]
}
```

| Kod | Przyczyna |
|-----|-----------|
| `200` | Sukces |
| `404` | Serwer nie istnieje |

---

### `PUT /api/servers/:uuid/approve`

Zatwierdza serwer — odblokowuje zbieranie metryk. Jeśli agent jest aktualnie połączony, wysyła do niego `auth_response` z `approved: true` przez WebSocket (push w czasie rzeczywistym).

**Body:** brak

**Response `200`:**
```json
{
  "success": true
}
```

---

### `DELETE /api/servers/:uuid`

Usuwa serwer i wszystkie powiązane dane — kontenery, metryki (CASCADE DELETE w SQLite).

**Response `200`:**
```json
{
  "success": true
}
```

---

## Komendy

Wysyła komendę do agenta przez WebSocket i czeka na odpowiedź (timeout **30s**). Jeśli agent nie jest połączony lub nie odpowie — `timeout` lub błąd.

### `POST /api/servers/:uuid/command`

Komenda dla serwera (głównie akcje systemowe lub Docker na poziomie projektu).

**Request:**
```json
{
  "action": "check-updates",
  "target": "webstack"
}
```

| Pole | Wymagane | Opis |
|------|----------|------|
| `action` | ✅ | Akcja do wykonania (patrz tabela akcji w `agent.md`) |
| `target` | zależy od akcji | Nazwa kontenera lub projektu compose |

**Response `200`:** surowy JSON zwrócony przez agenta (zależy od akcji).

---

### `POST /api/servers/:uuid/containers/:id/command`

Komenda dla konkretnego kontenera. `:id` to `container_id` (skrócone ID Dockera).

**Request:**
```json
{
  "action": "restart"
}
```

**Response `200`:** surowy JSON zwrócony przez agenta.

| Kod | Przyczyna |
|-----|-----------|
| `200` | Sukces — payload z agenta |
| `500` | Timeout (agent nie odpowiedział w 30s) lub inny błąd wewnętrzny |
| `503` | Agent nie jest połączony |

---

## Metryki historyczne

### Dostępne zakresy (`?range=`)

| Wartość | Źródło danych | Rozdzielczość |
|---------|---------------|---------------|
| `1m` | RAM buffer | ~1s (surowe) |
| `5m` | `metrics_5s` | 5s |
| `15m` | `metrics_15s` | 15s |
| `30m` | `metrics_30s` | 30s |
| `1h` | `metrics_1m` | 1 min |
| `6h` | `metrics_5m` | 5 min |
| `12h` | `metrics_15m` | 15 min |
| `24h` | `metrics_30m` | 30 min |
| `7d` | `metrics_1h` | 1 godz |
| `15d` | `metrics_6h` | 6 godz |
| `30d` | `metrics_12h` | 12 godz |

Domyślny zakres gdy brak parametru: `1h`.

---

### `GET /api/metrics/history/servers/:uuid?range=<zakres>`

Historia metryk serwera — host + wszystkie kontenery z danymi w zadanym zakresie.

**Response `200` — zakres `1m` (surowe z RAM):**
```json
{
  "host": {
    "points": [
      {
        "timestamp": 1739967600,
        "cpu": 12.5,
        "mem_used": 10622971904,
        "mem_percent": 32.5,
        "disk_read_bytes_per_sec": 1048576,
        "disk_write_bytes_per_sec": 524288,
        "net_rx_bytes_per_sec": 204800,
        "net_tx_bytes_per_sec": 102400
      }
    ]
  },
  "containers": [
    {
      "container_id": "fb629436cc81",
      "name": "nginx",
      "image": "nginx:latest",
      "project": "webstack",
      "service": "nginx",
      "points": [
        {
          "timestamp": 1739967600,
          "cpu": 0.5,
          "mem_used": 52428800,
          "disk_used": 10485760,
          "net_rx_bytes": 1024000,
          "net_tx_bytes": 512000
        }
      ]
    }
  ]
}
```

**Response `200` — zakres `>1m` (zagregowane z DB):**

Pola `host.points[]` zmieniają strukturę — zawierają wartości avg/min/max:
```json
{
  "host": {
    "points": [
      {
        "timestamp": 1739967600,
        "cpu_avg": 12.5,
        "cpu_min": 8.0,
        "cpu_max": 18.0,
        "mem_used_avg": 10622971904,
        "mem_used_min": 10000000000,
        "mem_used_max": 11000000000,
        "disk_read_bytes_per_sec_avg": 1048576,
        "disk_write_bytes_per_sec_avg": 524288,
        "disk_write_bytes_per_sec_min": 0,
        "disk_write_bytes_per_sec_max": 1048576,
        "net_rx_bytes_per_sec_avg": 204800,
        "net_rx_bytes_per_sec_min": 100000,
        "net_rx_bytes_per_sec_max": 350000,
        "net_tx_bytes_per_sec_avg": 102400,
        "net_tx_bytes_per_sec_min": 50000,
        "net_tx_bytes_per_sec_max": 200000
      }
    ]
  },
  "containers": [
    {
      "container_id": "fb629436cc81",
      "name": "nginx",
      "image": "nginx:latest",
      "project": "webstack",
      "service": "nginx",
      "points": [
        {
          "timestamp": 1739967600,
          "cpu_avg": 0.5,
          "cpu_min": 0.1,
          "cpu_max": 1.2,
          "mem_avg": 52428800,
          "mem_min": 51000000,
          "mem_max": 54000000,
          "net_rx_avg": 1024000,
          "net_rx_min": 800000,
          "net_rx_max": 1200000,
          "net_tx_avg": 512000,
          "net_tx_min": 400000,
          "net_tx_max": 600000
        }
      ]
    }
  ]
}
```

> Kontenery bez danych w zadanym zakresie są pomijane w odpowiedzi.

| Kod | Przyczyna |
|-----|-----------|
| `200` | Sukces |
| `400` | Nieprawidłowy zakres |

---

### `GET /api/metrics/history/servers/:uuid/containers/:id?range=<zakres>`

Historia metryk pojedynczego kontenera. `:id` to `container_id`.

**Response `200` — zakres `1m`:**
```json
{
  "points": [
    {
      "timestamp": 1739967600,
      "cpu": 0.5,
      "mem_used": 52428800,
      "disk_used": 10485760,
      "net_rx_bytes": 1024000,
      "net_tx_bytes": 512000
    }
  ]
}
```

**Response `200` — zakres `>1m`:**
```json
{
  "points": [
    {
      "timestamp": 1739967600,
      "cpu_avg": 0.5,
      "cpu_min": 0.1,
      "cpu_max": 1.2,
      "mem_avg": 52428800,
      "mem_min": 51000000,
      "mem_max": 54000000,
      "net_rx_avg": 1024000,
      "net_rx_min": 800000,
      "net_rx_max": 1200000,
      "net_tx_avg": 512000,
      "net_tx_min": 400000,
      "net_tx_max": 600000
    }
  ]
}
```

---

## Metryki live (SSE)

Strumień `text/event-stream`. Każda wiadomość ma format:
```
data: <json>\n\n
```

Oba endpointy SSE wymagają tokenu JWT. Ticker co **1 sekundę**.

---

### `GET /api/metrics/live/all`

Skrócone metryki wszystkich **zatwierdzonych** serwerów aktualnie połączonych. Przeznaczony do widoku listy serwerów — szybki przegląd stanu wszystkich maszyn.

Serwery bez aktywnych danych w RAM (brak agenta lub agent rozłączony) są pomijane.

**Event `data`:**
```json
{
  "servers": [
    {
      "uuid": "a3f2c1d4e5b6...",
      "hostname": "server1",
      "cpu": 12.5,
      "memory": 10622971904,
      "disk_read_bytes_per_sec": 1048576,
      "disk_write_bytes_per_sec": 524288,
      "net_rx_bytes_per_sec": 204800,
      "net_tx_bytes_per_sec": 102400
    }
  ]
}
```

| Pole | Typ | Opis |
|------|-----|------|
| `uuid` | string | UUID serwera |
| `hostname` | string | Nazwa hosta |
| `cpu` | float64 | Użycie CPU (%) |
| `memory` | uint64 | Użyta pamięć RAM (bajty) |
| `disk_read_bytes_per_sec` | uint64 | Odczyty dysku (bajty/s) |
| `disk_write_bytes_per_sec` | uint64 | Zapisy dysku (bajty/s) |
| `net_rx_bytes_per_sec` | uint64 | Odbiór sieciowy (bajty/s) |
| `net_tx_bytes_per_sec` | uint64 | Wysyłanie sieciowe (bajty/s) |

---

### `GET /api/metrics/live/servers/:uuid`

Pełne metryki live jednego serwera — host + wszystkie kontenery z tego samego ticka buffera. Przeznaczony do widoku szczegółów serwera.

Jeśli brak danych hosta w RAM (agent offline) — tick jest pomijany, brak eventu.

**Event `data`:**
```json
{
  "server_uuid": "a3f2c1d4e5b6...",
  "timestamp": 1739967600,
  "host": {
    "Timestamp": 1739967600,
    "CPU": 12.5,
    "MemUsed": 10622971904,
    "MemPercent": 32.5,
    "DiskReadBytesPerSec": 1048576,
    "DiskWriteBytesPerSec": 524288,
    "NetRxBytesPerSec": 204800,
    "NetTxBytesPerSec": 102400
  },
  "containers": [
    {
      "Timestamp": 1739967600,
      "CPU": 0.5,
      "MemUsed": 52428800,
      "MemPercent": 0.16,
      "DiskUsed": 10485760,
      "DiskPercent": 0.0,
      "NetRx": 1024000,
      "NetTx": 512000
    }
  ]
}
```

> **Uwaga:** Pole `host` w tym endpoincie zwraca klucze w `PascalCase` (Go struct serialized directly), w odróżnieniu od `snake_case` w `/live/all` i endpointach historycznych. Frontend musi obsługiwać obie konwencje lub normalizować po stronie klienta.

---

## WebSocket — Agent

### `GET /ws/agent`

Endpoint wyłącznie dla agentów. Nie wymaga tokenu JWT — autentykacja odbywa się przez wiadomość `auth` po nawiązaniu połączenia.

Szczegółowy opis protokołu WebSocket (typy wiadomości, struktury JSON, lifecycle połączenia) — patrz `agent.md` → sekcja *Tryb WebSocket*.
