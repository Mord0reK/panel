# Dokumentacja API

## Autentykacja
Większość endpointów wymaga nagłówka:
`Authorization: Bearer <token>`

### 1. Auth
- `POST /api/setup` - Pierwsza konfiguracja (tylko gdy brak użytkowników). Zwraca token.
- `POST /api/login` - Logowanie. Zwraca token.
- `GET /api/auth/status` - Sprawdzenie statusu sesji i konieczności setupu.

### 2. Serwery
- `GET /api/servers` - Lista wszystkich serwerów.
- `GET /api/servers/:uuid` - Szczegóły serwera i lista jego kontenerów.
- `PUT /api/servers/:uuid/approve` - Zatwierdzenie serwera (wymagane do zbierania metryk).
- `DELETE /api/servers/:uuid` - Usunięcie serwera i wszystkich jego danych.

### 3. Komendy
- `POST /api/servers/:uuid/command` - Wysłanie komendy do agenta.
- `POST /api/servers/:uuid/containers/:id/command` - Komenda dla konkretnego kontenera.

### 4. Metryki Historyczne
- `GET /api/metrics/history/servers/:uuid?range=1h`
  - Zwraca:
  - `host_points`: seria metryk hosta dla zakresu
  - `containers`: serie per-kontener (tylko kontenery z próbkami w zakresie)
  - `points`: alias kompatybilności wskazujący na `host_points`
- `GET /api/metrics/history/servers/:uuid/containers/:id?range=1h`
Dostępne zakresy: `1m, 5m, 15m, 30m, 1h, 6h, 12h, 24h, 7d, 15d, 30d`.

### 5. Live Stream (SSE)
- `GET /api/metrics/live/all` - Stream metryk hosta (CPU, RAM, net i disk I/O w `bytes_per_sec`) dla wszystkich serwerów.
- `GET /api/metrics/live/servers/:uuid` - Stream pełnych metryk live serwera: `host` + `containers` z tego samego ticka.

## WebSocket (Agent)
- `GET /ws/agent` - Endpoint dla agentów.
Wymaga przesłania wiadomości typu `auth` natychmiast po połączeniu.
