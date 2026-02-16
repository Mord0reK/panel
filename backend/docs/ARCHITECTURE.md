# Architektura Systemu

## Przepływ Danych
1. **Agent** łączy się przez **WebSocket** i przesyła metryki co 1s.
2. **Agent** buduje jeden kanoniczny snapshot (`host + containers`) i używa go zarówno do streamu metrics jak i komendy `stats`.
3. **WebSocketHandler** odbiera snapshoty, aktualizuje status serwera w DB i dodaje metryki hosta oraz kontenerów do **RAM Buffer**.
4. **BulkInserter** co 10s pobiera dane z buforów i wykonuje masowy zapis do tabel `metrics_1s` (kontenery) oraz `host_metrics_1s` (host).
5. **Aggregator** co 10s agreguje dane historyczne kontenerów i hosta na wyższe poziomy (5s, 15s, etc.), usuwając stare dane źródłowe.
6. **REST API** i **SSE** serwują spójne dane live/history z jednego pipeline'u metryk.

## Komponenty
- **AgentHub**: Zarządza aktywnymi połączeniami WebSocket.
- **BufferManager**: Przechowuje ostatnie 60s danych hosta i kontenerów w pamięci.
- **BulkInserter**: Optymalizuje zapisy do bazy SQLite (host + kontenery).
- **Aggregator**: Redukuje rozmiar bazy danych zachowując trend historyczny (host + kontenery).

## Schemat Bazy Danych
System używa SQLite z włączonym trybem WAL dla lepszej wydajności przy jednoczesnych zapisach i odczytach.
Główne tabele: `users`, `servers`, `containers`, oraz 11 tabel metryk o różnej rozdzielczości.
