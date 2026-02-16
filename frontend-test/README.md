# API Panel Tester

Prosty frontend w vanilla JavaScript do testowania wszystkich endpointów backendu.

## Jak uruchomić

1. Otwórz plik `index.html` w przeglądarce
2. Upewnij się, że backend działa na `http://localhost:8080`

## Funkcjonalności

### Autoryzacja
- **GET /api/auth/status** - Sprawdzenie statusu autoryzacji
- **POST /api/setup** - Pierwszy setup systemu (utworzenie użytkownika)
- **POST /api/login** - Logowanie użytkownika

### Serwery
- **GET /api/servers** - Lista wszystkich serwerów
- **GET /api/servers/{uuid}** - Szczegóły serwera z kontenerami
- **PUT /api/servers/{uuid}/approve** - Zatwierdzenie serwera
- **DELETE /api/servers/{uuid}** - Usunięcie serwera

### Komendy
- **POST /api/servers/{uuid}/command** - Wysłanie komendy do serwera
- **POST /api/servers/{uuid}/containers/{id}/command** - Wysłanie komendy do kontenera

### Metryki - Historia
- **GET /api/metrics/history/servers/{uuid}** - Historia metryk serwera
- **GET /api/metrics/history/servers/{uuid}/containers/{id}** - Historia metryk kontenera

Dostępne zakresy czasu: 1m, 5m, 15m, 30m, 1h, 6h, 12h, 24h, 7d, 15d, 30d

### Metryki - Live (SSE)
- **GET /api/metrics/live/all** - Metryki wszystkich serwerów w czasie rzeczywistym
- **GET /api/metrics/live/servers/{uuid}** - Metryki konkretnego serwera w czasie rzeczywistym

## Uwagi

- Token JWT jest automatycznie zapisywany w localStorage po setupie/logowaniu
- Wszystkie endpointy chronione wymagają tokenu JWT (oprócz auth endpoints)
- Odpowiedzi API wyświetlają się w panelu po prawej stronie
- SSE połączenia można uruchomić przyciskiem "Start" i zatrzymać przyciskiem "Stop"

## Zmiana adresu API

Jeśli backend działa na innym adresie, zmień wartość `API_BASE` w pliku `script.js`:

```javascript
const API_BASE = 'http://twoj-adres:port';
```
