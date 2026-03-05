# Frontend — Dokumentacja

Next.js (App Router) dashboard do monitorowania serwerów. Jedyny publicznie wystawiony komponent stacku — przez tunel Cloudflare.

## Spis treści

- [Stack](#stack)
- [Architektura komunikacji](#architektura-komunikacji)
- [Proxy API](#proxy-api)
- [Autentykacja](#autentykacja)
- [Routing](#routing)
- [Struktura projektu](#struktura-projektu)
- [Layout i Sidebar](#layout-i-sidebar)
- [Widoki](#widoki)
- [Wykresy — ECharts](#wykresy--echarts)
- [SSE — live metryki](#sse--live-metryki)
- [Dropdown zakresów](#dropdown-zakresów)
- [Zmienne środowiskowe](#zmienne-środowiskowe)

---

## Stack

| Technologia | Rola |
|-------------|------|
| Next.js 16 (App Router) | Framework, routing, proxy API |
| TypeScript | Typowanie |
| Tailwind CSS | Stylowanie |
| shadcn/ui | Komponenty UI |
| Apache ECharts (`echarts-for-react`) | Wykresy live i historyczne |
| pnpm | Package manager |

---

## Architektura komunikacji

Frontend **nigdy nie komunikuje się bezpośrednio z backendem**. Wszystkie żądania idą przez Next.js Route Handlers (`/api/*`), które działają jako proxy.

```
Przeglądarka
    │
    │  /api/*  (Next.js Route Handlers)
    ▼
Next.js Server
    │
    │  http://backend:8080/api/*  (wewnętrzna sieć Docker)
    ▼
Backend Go
```

Dzięki temu:
- Token JWT nigdy nie jest dostępny w JS przeglądarki (httpOnly cookie)
- Backend nie musi być publicznie wystawiony
- CORS nie jest problemem — Next.js i backend to ten sam origin z perspektywy przeglądarki

---

## Proxy API

Każdy Route Handler w `app/api/` odczytuje token z httpOnly cookie i przekazuje go do backendu jako nagłówek `Authorization: Bearer <token>`.

### Mapowanie ścieżek

| Ścieżka Next.js | Ścieżka backendu |
|-----------------|-----------------|
| `POST /api/auth/login` | `POST /api/login` |
| `POST /api/auth/setup` | `POST /api/setup` |
| `GET /api/auth/status` | `GET /api/auth/status` |
| `GET /api/servers` | `GET /api/servers` |
| `GET /api/servers/[uuid]` | `GET /api/servers/:uuid` |
| `PATCH /api/servers/[uuid]` | `PATCH /api/servers/:uuid` |
| `PUT /api/servers/[uuid]/approve` | `PUT /api/servers/:uuid/approve` |
| `DELETE /api/servers/[uuid]` | `DELETE /api/servers/:uuid` |
| `POST /api/servers/[uuid]/command` | `POST /api/servers/:uuid/command` |
| `POST /api/servers/[uuid]/containers/[id]/command` | `POST /api/servers/:uuid/containers/:id/command` |
| `POST /api/servers/[uuid]/containers/[id]/check-update` | `POST /api/servers/:uuid/containers/:id/check-update` |
| `POST /api/servers/[uuid]/containers/[id]/update` | `POST /api/servers/:uuid/containers/:id/update` |
| `DELETE /api/servers/[uuid]/containers/[id]` | `DELETE /api/servers/:uuid/containers/:id` |
| `DELETE /api/servers/[uuid]/containers` | `DELETE /api/servers/:uuid/containers` |
| `GET /api/metrics/history/servers/[uuid]` | `GET /api/metrics/history/servers/:uuid` |
| `GET /api/metrics/history/servers/[uuid]/containers/[id]` | `GET /api/metrics/history/servers/:uuid/containers/:id` |
| `GET /api/metrics/live/all` | `GET /api/metrics/live/all` |
| `GET /api/metrics/live/servers/[uuid]` | `GET /api/metrics/live/servers/:uuid` |
| `GET /api/icons` | lokalny endpoint Next.js (ikony z `/public/icons`) |

> Endpointy SSE (`/api/metrics/live/*`) wymagają specjalnej obsługi w Route Handlerze — strumieniowanie response zamiast jednorazowego zwrotu.

### Zmienna środowiskowa backendu

```
BACKEND_URL=http://backend:8080
```

Używana wyłącznie server-side w Route Handlerach. Nigdy nie trafia do klienta.

---

## Autentykacja

### Przechowywanie tokenu — httpOnly cookie

Token JWT nigdy nie jest dostępny w JavaScript przeglądarki. Ustawia go Route Handler `/api/auth/login` po otrzymaniu go z backendu:

```
Set-Cookie: token=<jwt>; HttpOnly; SameSite=Strict; Path=/; Max-Age=604800
```

Każdy kolejny Route Handler odczytuje token z `cookies()` (Next.js server-side) i dołącza go do żądania do backendu.

### Ochrona tras — `proxy.ts`

`proxy.ts` (root projektu) działa server-side przed renderem każdej strony:

- Sprawdza obecność cookie `token`
- Brak tokenu → redirect na `/login`
- Trasy publiczne bez ochrony: `/login`, `/setup`
- Sprawdzenie ważności tokenu: wywołanie `GET /api/auth/status` (real-time w middleware/proxy)

### Flow autentykacji

```
1. Użytkownik wchodzi na dowolną stronę
       │
       ▼
2. proxy.ts — czy jest cookie `token`?
       │ NIE → redirect /login
       │ TAK → kontynuuj
       ▼
3. Strona renderuje się
       │
       ▼
4. Client Component pobiera dane przez /api/* (cookie dołączane automatycznie)
```

### Flow pierwszego uruchomienia

```
GET /api/auth/status → { setup_required: true }
    → redirect /setup
    → POST /api/auth/setup → ustaw cookie → redirect /dashboard
```

---

## Routing

```
app/
├── (auth)/
│   ├── login/
│   │   └── page.tsx          # Strona logowania
│   └── setup/
│       └── page.tsx          # Pierwsza konfiguracja (setup_required: true)
│
└── (app)/
    ├── layout.tsx             # Layout z Sidebarem — chroniony przez proxy
    ├── dashboard/
    │   └── page.tsx          # Lista serwerów z live overview (SSE /live/all)
    └── servers/
        └── [uuid]/
            ├── page.tsx       # redirect → /servers/[uuid]/metrics
            ├── metrics/
            │   └── page.tsx  # Metryki serwera (live + historia)
            ├── containers/
            │   └── page.tsx  # Tabela kontenerów + akcje
            └── logs/
                └── page.tsx  # (placeholder — przyszłość)
```

Route group `(auth)` — bez layoutu z Sidebarem.
Route group `(app)` — z layoutem (Sidebar + topbar).

---

## Struktura projektu

```
frontend/
├── app/
│   ├── (auth)/
│   ├── (app)/
│   └── api/                   # Route Handlers (proxy do backendu)
│       ├── auth/
│       ├── servers/
│       └── metrics/
├── components/
│   ├── ui/                    # shadcn/ui (auto-generowane)
│   ├── sidebar/
│   │   ├── Sidebar.tsx
│   │   ├── ServerNavItem.tsx  # Pojedynczy serwer w sidebarze (collapsible)
│   │   └── ServerNavList.tsx
│   ├── charts/
│   │   ├── LiveChart.tsx      # Wykres live (SSE feed)
│   │   └── HistoryChart.tsx   # Wykres historyczny
│   ├── servers/
│   │   ├── ServerCard.tsx     # Karta serwera na dashboardzie
│   │   └── ServerInfo.tsx     # Informacje statyczne (CPU, RAM, OS)
│   ├── containers/
│   │   ├── ContainersTable.tsx
│   │   └── ContainerActions.tsx
│   └── metrics/
│       ├── RangeDropdown.tsx  # Dropdown wyboru zakresu
│       └── MetricsGrid.tsx    # Grid 4 wykresów (CPU/RAM/Disk/Net)
├── hooks/
│   ├── useSSE.ts              # Hook do obsługi SSE z auto-reconnect
│   ├── useServerMetrics.ts    # Live metryki konkretnego serwera
│   └── useLiveAll.ts          # Live metryki wszystkich serwerów
├── lib/
│   ├── api.ts                 # Klient HTTP (fetch wrapper z obsługą błędów)
│   ├── backend.ts             # Server-side proxy helper (używany w Route Handlerach)
│   └── formatters.ts          # formatBytes(), formatPercent(), formatTimestamp()
├── types/
│   └── index.ts               # Wszystkie TypeScript interfaces (Server, Container, MetricPoint...)
└── proxy.ts              # Auth guard
```

---

## Layout i Sidebar

### Sidebar — struktura

Sidebar jest stały (nie chowany na desktopie). Zawiera sekcję **Serwery** jako collapsible lista.

```
Sidebar
├── Logo / nazwa aplikacji
├── [Sekcja: Serwery]  ← collapsible
│   ├── server1  ← collapsible
│   │   ├── 📊 Metryki        → /servers/[uuid]/metrics
│   │   ├── 📋 Logi           → /servers/[uuid]/logs  (disabled)
│   │   └── 🐳 Kontenery      → /servers/[uuid]/containers
│   ├── server2  ← collapsible
│   │   └── ...
│   └── [niezatwierdzony-serwer]  ← badge "Oczekuje"
└── [przyszłe sekcje — zewnętrzne usługi]
```

Dane listy serwerów do Sidebaru pobierane przez `GET /api/servers` przy montowaniu layoutu, odświeżane co 30s (polling) lub po akcji zatwierdzenia.

Sidebar wyróżnia aktywną trasę (aktywny serwer + aktywna zakładka).

---

## Widoki

### `/dashboard` — Lista serwerów

Siatka kart serwerów. Każda karta pokazuje live dane z SSE `/api/metrics/live/all`:
- Hostname
- CPU % (pasek + liczba)
- RAM usage
- Net RX/TX
- Disk R/W
- Status (online/offline — na podstawie obecności w SSE evencie)

Kliknięcie karty → redirect na `/servers/[uuid]/metrics`.

### `/servers/[uuid]/metrics` — Metryki serwera

**Górna sekcja — info statyczne:**
Hostname, CPU model, liczba rdzeni, RAM total, platforma, kernel, architektura. Dane z `GET /api/servers/[uuid]`. Server Component (bez re-renderów).

**Środkowa sekcja — dropdown zakresu + grid wykresów:**
Cztery wykresy: CPU %, RAM (bajty), Disk R/W (bajty/s), Net RX/TX (bajty/s).

Przy zakresie `Live (1m)` — wykresy zasilane przez SSE (`/api/metrics/live/servers/[uuid]`), dane dopisywane w czasie rzeczywistym, okno przesuwne 60 punktów.

Przy zakresie `>1m` — jednorazowe zapytanie `GET /api/metrics/history/servers/[uuid]?range=<zakres>`, dane statyczne z możliwością ręcznego odświeżenia.

### `/servers/[uuid]/containers` — Kontenery

Tabela kontenerów z `GET /api/servers/[uuid]` (pole `containers`). Kolumny: nazwa, obraz, projekt compose, status, akcje.

Akcje per kontener: `start`, `stop`, `restart` — przez `POST /api/servers/[uuid]/containers/[id]/command`. Przycisk disabled podczas oczekiwania na odpowiedź (timeout 30s z backendu).

---

## Wykresy — ECharts

Biblioteka: `echarts-for-react` (wrapper React dla Apache ECharts).

### LiveChart

Client Component. Przyjmuje nowe punkty przez props/callback z hooka `useServerMetrics`. Używa `chart.setOption()` z `notMerge: false` do dopisywania punktów bez pełnego re-renderu — kluczowe dla wydajności przy 1s interwale.

Okno: ostatnie **60 punktów** (1 minuta). Przy dodaniu nowego punktu najstarszy odpada.

### HistoryChart

Client Component. Inicjalizowany jednorazowo z tablicą punktów historycznych. Przy zakresie `>1m` dane mają strukturę `avg/min/max` — wykresy mogą pokazywać obszar min-max jako wypełnienie (ECharts `areaStyle` z dwoma seriami).

### Wspólna konfiguracja

- Ciemny motyw (custom theme lub `dark` preset ECharts)
- Brak legendy dla pojedynczych serii
- Tooltip z formatowaniem jednostek (bajty → MB/GB, procenty z 2 miejscami po przecinku)
- Responsive — `style={{ width: '100%', height: '200px' }}`
- Oś X: timestamp (format zależny od zakresu — `HH:mm:ss` dla live, `HH:mm` dla 1h, `MM-DD` dla 7d+)

---

## SSE — live metryki

### Hook `useSSE`

Bazowy hook opakowujący `EventSource`.

Autoryzacja SSE jest realizowana server-side przez Route Handlery (`backendStream`), które przekazują token z httpOnly cookie do backendu i streamują odpowiedź do przeglądarki.

Strategia reconnect: przy błędzie lub zamknięciu połączenia — exponential backoff (1s, 2s, 4s, max 30s).

### Hook `useServerMetrics`

Używa `useSSE` na `/api/metrics/live/servers/[uuid]`. Utrzymuje bufor ostatnich 60 punktów per seria (CPU, RAM, disk, net) i udostępnia go komponentom wykresów. Normalizuje PascalCase z backendu na snake_case.

### Hook `useLiveAll`

Używa `useSSE` na `/api/metrics/live/all`. Utrzymuje mapę `uuid → ostatni punkt` i udostępnia ją kartom serwerów na dashboardzie.

> **Uwaga SSE przez proxy:** `EventSource` nie obsługuje custom headers ani cookies cross-origin. Ponieważ Next.js jest tym samym originem co przeglądarka, cookie `token` będzie dołączane automatycznie. Route Handler musi forwardować strumień z backendu do klienta bez buforowania (`Transfer-Encoding: chunked`).

---

## Dropdown zakresów

Pogrupowane opcje w `<Select>` (shadcn/ui):

```
─ Live ─────────────────
  • Live (1m)           ← oznaczony badge "LIVE", zielona kropka

─ Minuty ───────────────
  • 5 minut
  • 15 minut
  • 30 minut

─ Godziny ──────────────
  • 1 godzina
  • 6 godzin
  • 12 godzin
  • 24 godziny

─ Dni ──────────────────
  • 7 dni
  • 15 dni
  • 30 dni
```

| Etykieta | Wartość `?range=` |
|----------|-------------------|
| Live (1m) | `1m` |
| 5 minut | `5m` |
| 15 minut | `15m` |
| 30 minut | `30m` |
| 1 godzina | `1h` |
| 6 godzin | `6h` |
| 12 godzin | `12h` |
| 24 godziny | `24h` |
| 7 dni | `7d` |
| 15 dni | `15d` |
| 30 dni | `30d` |

Zmiana zakresu:
- `1m` → uruchamia SSE, zatrzymuje poprzednie zapytanie historyczne
- `>1m` → zatrzymuje SSE, wysyła zapytanie historyczne, pokazuje spinner podczas ładowania

---

## Zmienne środowiskowe

| Zmienna | Gdzie używana | Opis |
|---------|--------------|------|
| `BACKEND_URL` | Server-side (Route Handlers) | URL backendu wewnątrz Dockera, np. `http://backend:8080` |
| `NODE_ENV` | Server-side (auth routes) | Steruje flagą `secure` dla cookie (`true` w production) |

`BACKEND_URL` **nigdy** nie jest prefiksowana `NEXT_PUBLIC_` — nie może trafić do przeglądarki.
