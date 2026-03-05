---
name: panel-analysis
description: Procedura głębokiej analizy repozytorium. Używaj ZAWSZE na początku pracy z nowym projektem lub modułem, aby zrozumieć architekturę, przepływ danych (SSE/WebSocket) i strukturę katalogów.
---

# Skill: Panel — Rekonesans przed implementacją

## Cel

Ten skill definiuje standard pracy dla subagenta-analityka. Jego jedynym zadaniem jest przeprowadzenie dokładnego rekonesansu kodu przed przekazaniem raportu do Orchestratora. Nie pisze kodu — tylko analizuje i raportuje.

## Mapa architektury (kontekst stały)

Repozytorium składa się z trzech niezależnych modułów Go/TS:

| Komponent | Ścieżka | Język | Rola |
|-----------|---------|-------|------|
| **Agent** | `/agent/` | Go (`module agent`) | Klient WS — zbiera metryki systemowe i dockerowe, wysyła do backendu co 1s |
| **Backend** | `/backend/` | Go (`module backend`) | Serwer WS + REST API — SQLite, JWT auth, SSE dla frontendu |
| **Frontend** | `/frontend/` | Next.js 16 + React 19 | Dashboard — konsumuje REST + SSE backendu |

### Warstwy backendu (`/backend/internal/`)

```
api/          — handlery HTTP: auth.go, servers.go, metrics.go, commands.go, sse.go, websocket.go
aggregation/  — redukcja punktów danych (SQLite)
auth/         — JWT + hashing
buffer/       — RAM buffer dla metryk live (1m)
config/       — konfiguracja serwera
database/     — SQLite driver (modernc.org/sqlite)
models/       — struktury Go: server.go, container.go, metrics.go, user.go
websocket/    — agent.go (połączenia agentów), protocol.go (typy wiadomości)
```

### Warstwy agenta (`/agent/internal/`)

```
collector/    — CollectSystemMetrics, CollectSystemInfo
config/       — konfiguracja agenta
docker/       — klient Docker API (moby), CollectContainerMetrics, ContainerManager
metrics/      — SnapshotCollector (dedupe CPU delta)
output/       — zapis do JSON
websocket/    — klient WS, AuthInfo, MetricsMessage, Command
```

### Warstwy frontendu (`/frontend/`)

```
app/(app)/dashboard/  — strona główna panelu
app/(app)/servers/    — lista + szczegóły serwerów
app/(app)/settings/   — ustawienia
app/(auth)/           — login / setup
components/           — auth/, charts/, containers/, metrics/, servers/, sidebar/, ui/
hooks/                — useSSE.ts, useLiveAll.ts, useServerMetrics.ts
lib/                  — api.ts (fetch wrappery), backend.ts (URL resolver), formatters.ts
types/index.ts        — WSZYSTKIE typy TS (Server, Container, MetricPoint, SSE events)
```

### Krytyczne powiązania między komponentami

- `agent/internal/websocket` → `backend/internal/websocket/protocol.go` — muszą mieć identyczne typy JSON
- `backend/internal/models/*.go` → `frontend/types/index.ts` — zmiana struktury modelu = zmiana typów TS
- `backend/internal/api/sse.go` → `frontend/hooks/useSSE.ts` / `useLiveAll.ts` — format SSE eventów
- `backend/internal/api/metrics.go` → `frontend/lib/api.ts` — endpointy REST
- **UWAGA:** SSE `/api/metrics/live/servers/[uuid]` zwraca PascalCase (Go struct bezpośrednio), `/api/metrics/live/all` — snake_case. Normalizacja w `frontend/types/index.ts`: `normalizeLiveHost()`, `normalizeLiveContainer()`

---

## Procedura rekonesansu

### Krok 1 — Mapowanie zależności

**Zakaz zgadywania.** Zanim sformułujesz cokolwiek, wykonaj:

**Go (agent lub backend):**
1. Odczytaj `go.mod` docelowego modułu — sprawdź zewnętrzne zależności
2. Sprawdź importy w plikach `.go` dotkniętych zadaniem: szukaj wzorca `"agent/internal/..."` lub `"backend/internal/..."`
3. Zidentyfikuj inne pakiety importujące modyfikowany pakiet (reverse deps)

**Frontend (Next.js/React):**
1. Odczytaj `package.json` — wersje Next.js, React, bibliotek UI
2. Sprawdź importy w modyfikowanych komponentach: skąd importują typy (zawsze `types/index.ts`), hooki, funkcje z `lib/`
3. Jeśli zmiana dotyczy typów — przeszukaj ALL pliki importujące `types/index.ts`

### Krok 2 — Weryfikacja faktyczna (zakaz założeń)

**Agentowi zabrania się zakładać, że nazwa pliku lub katalogu w pełni opisuje jego zawartość.**

Wymagane działania:
- Fizycznie odczytaj kluczowe fragmenty pliku (nazwy funkcji, sygnatury, eksportowane typy)
- Dla pliku Go: sprawdź `package`, eksportowane funkcje/struktury, ich parametry
- Dla komponentu React: sprawdź props interface, co przyjmuje, co renderuje, jakie hooki używa
- Dla hooka: sprawdź co zwraca (typ zwrotny), jakie SSE/API woła
- Dla `types/index.ts`: sprawdź konkretny interface/type zanim powiesz że istnieje lub nie

### Krok 3 — Identyfikacja punktów styku

Szukaj aktywnie cross-komponentowych powiązań:
- Czy zmiana w Go strucie wymaga aktualizacji TypeScript interface?
- Czy zmiana w endpointach REST wymaga aktualizacji `lib/api.ts`?
- Czy zmiana w formacie SSE wymaga aktualizacji hooków i `types/index.ts`?
- Czy zmiana w WS protocol wymaga synchronizacji agent ↔ backend?

---

## Format raportu dla Orchestratora

Po zakończeniu rekonesansu subagent-analityk MUSI zwrócić wyniki dokładnie w poniższym formacie:

```
## Raport rekonesansu: [opis zadania]

### Pliki do bezpośrednich modyfikacji
- `ścieżka/do/pliku.go` : [co konkretnie zmieniać — funkcja X, struct Y]
- `frontend/types/index.ts` : [które interfejsy]
- ...

### Pliki powiązane (wymagają weryfikacji / możliwe breaking changes)
- `ścieżka/do/pliku.go` : [dlaczego powiązane — importuje pakiet X / używa typu Y]
- `frontend/hooks/useSSE.ts` : [jeśli zmieniony format SSE]
- ...

### Ostrzeżenie o powiązaniach cross-komponentowych
[Konkretna lista: np. "Zmiana `HostMetrics` w `backend/internal/models/metrics.go` wymaga aktualizacji `LiveServerHostRaw` w `frontend/types/index.ts` oraz `normalizeLiveHost()` w tym samym pliku"]

### Rekomendacje skillów dla subagentów wykonawczych
- Subagent backend: [nazwa skilla z `.opencode/skills/`]
- Subagent frontend: [nazwa skilla z `.opencode/skills/`]
- Subagent testowy: [nazwa skilla z `.opencode/skills/`]

### Polecenie wykonawcze
[Co Orchestrator powinien zlecić w następnym kroku — konkretne zadania dla subagentów]
```

---

## Zasady bezwzględne

1. **Nie pisz kodu** — ten skill to tylko rekonesans
2. **Nie zakładaj** — sprawdź fizycznie
3. **Raportuj breaking changes jawnie** — nawet potencjalne
4. **Lista plików musi być wyczerpująca** — brakujący plik w raporcie = nieoczekiwany błąd kompilacji lub runtime
5. **Rekomendacja skillów jest obowiązkowa** — Orchestrator potrzebuje tej informacji do delegacji
