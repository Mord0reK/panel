# Backend Panel Sterowania

Backend dla systemu monitorowania kontenerów Docker, napisany w Go.

## Funkcje
- Obsługa agentów przez WebSocket
- Gromadzenie metryk w czasie rzeczywistym
- Agregacja danych historycznych (SQLite)
- Autentykacja JWT
- Live stream metryk przez SSE (Server-Sent Events)
- Proxy komend do agentów (restart, stop, etc.)

## Wymagania
- Go 1.24+
- SQLite (CGO enabled)
- Docker (opcjonalnie)

## Setup Lokalny
1. `cd backend`
2. `go mod download`
3. `go run cmd/server/main.go`

Serwer uruchomi się na porcie `8080`. Pierwsze uruchomienie wymaga konfiguracji przez `/api/setup`.

## Docker
```bash
docker-compose up --build
```

## Dokumentacja API
Szczegółowa dokumentacja znajduje się w folderze `docs/`.
- [API.md](./docs/API.md) - Endpointy REST i WebSocket
- [ARCHITECTURE.md](./docs/ARCHITECTURE.md) - Architektura systemu
