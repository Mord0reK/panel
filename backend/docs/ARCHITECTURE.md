# Architektura Systemu

## Przepływ Danych

1. **Agent** łączy się przez **WebSocket** i przesyła metryki co 1s.
2. **Agent** buduje jeden kanoniczny snapshot (`host + containers`) i używa go zarówno do streamu metrics jak i komendy `stats`.
3. **WebSocketHandler** odbiera snapshoty, aktualizuje status serwera w DB i dodaje metryki hosta oraz kontenerów do **RAM Buffer**.
4. **BulkInserter** co 10s pobiera punkty opuszczające 60-sekundowy RAM buffer i zapisuje je jako agregaty do `metrics_5s` (kontenery + techniczne serie hosta).
5. **Aggregator** co 10s agreguje dane historyczne na wyższe poziomy (15s, 30s, etc.), usuwając stare dane źródłowe.
6. **REST API** i **SSE** serwują spójne dane live/history z jednego pipeline'u metryk (host live z RAM dla `1m`, host/container history z DB dla `>1m`).

## Komponenty

- **AgentHub**: Zarządza aktywnymi połączeniami WebSocket.
- **BufferManager**: Przechowuje ostatnie 60s danych hosta i kontenerów w pamięci.
- **BulkInserter**: Optymalizuje zapisy do bazy SQLite (kontenery i host).
- **Aggregator**: Redukuje rozmiar bazy danych zachowując trend historyczny kontenerów i hosta.

## Schemat Bazy Danych

System używa SQLite z włączonym trybem WAL dla lepszej wydajności przy jednoczesnych zapisach i odczytach.
Główne tabele: `users`, `servers`, `containers`, `container_events`, oraz 10 tabel metryk o różnej rozdzielczości (`metrics_5s` do `metrics_12h`).

## System Usług Modularnych (Modular Services)

System pozwala na integrację zewnętrznych usług (np. AdGuard Home, Cloudflare) bezpośrednio w panelu.

1. **Interfejs Service**: Każda usługa implementuje standardowy interfejs Go, definiujący meta-dane, schemat konfiguracji oraz obsługę zapytań proxy.
2. **Rejestracja**: Usługi są rejestrowane w centralnym  przy starcie aplikacji.
3. **Konfiguracja**: Ustawienia usług (URL, poświadczenia) są przechowywane w tabeli `service_configs`. Wrażliwe dane są szyfrowane za pomocą **AES-256-GCM**.
4. **Proxy API**: Backend działa jako pośrednik (proxy) dla zapytań z frontendu, automatycznie dołączając wymagane dane uwierzytelniające do zapytań kierowanych do zewnętrznych usług.

## System Usług Modularnych (Modular Services)

System pozwala na integrację zewnętrznych usług (np. AdGuard Home, Cloudflare) bezpośrednio w panelu.

1. **Interfejs Service**: Każda usługa implementuje standardowy interfejs Go, definiujący meta-dane, schemat konfiguracji oraz obsługę zapytań proxy.
2. **Rejestracja**: Usługi są rejestrowane w centralnym `Registry` przy starcie aplikacji.
3. **Konfiguracja**: Ustawienia usług (URL, poświadczenia) są przechowywane w tabeli `service_configs`. Wrażliwe dane są szyfrowane za pomocą **AES-256-GCM**.
4. **Proxy API**: Backend działa jako pośrednik (proxy) dla zapytań z frontendu, automatycznie dołączając wymagane dane uwierzytelniające do zapytań kierowanych do zewnętrznych usług.
