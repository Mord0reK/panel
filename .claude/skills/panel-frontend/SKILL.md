---
name: panel-frontend
description: Standardy UI (Next.js, Tailwind 4, Shadcn). Krytyczne zasady normalizacji danych PascalCase z SSE na camelCase, wymóg Dark Mode oraz obowiązkowa weryfikacja zmian testami Playwright.
---

# Skill: Panel Frontend

## Mapa Modułu `/frontend`

```
Next.js 16 (App Router) + React 19 + TypeScript 5
Tailwind CSS 4 (CSS-first, bez tailwind.config.ts — konfiguracja w globals.css)
Shadcn UI (via `shadcn` package + komponenty w /components/ui/)
Lucide React ^0.575.0
ECharts + echarts-for-react (wykresy metryk)

app/
  (app)/              — chroniony middleware (JWT cookie)
  (auth)/             — /login, /setup (bez auth)
  api/                — Next.js Route Handlers (proxy do backendu Go)
  layout.tsx          — <html className="dark"> — DARK MODE ZAWSZE WŁĄCZONY
  globals.css         — @custom-variant dark (&:is(.dark *)); definicja tokenów CSS

components/
  auth/               — LoginForm, SetupForm
  charts/             — wykresy ECharts (metryki hosta i kontenerów)
  containers/         — tabele i karty kontenerów
  metrics/            — widżety metryk (CPU, RAM, Disk, Net)
  servers/            — ServerCard, ServerInfo, ServerIconDisplay, EditServerModal
  sidebar/            — nawigacja
  ui/                 — Shadcn primitives (Button, Card, Dialog, Sidebar, …)

hooks/
  useSSE.ts           — bazowy hook SSE (EventSource wrapper)
  useLiveAll.ts       — SSE /api/metrics/live/all → Map<uuid, LiveServerSnapshot>
  useServerMetrics.ts — SSE /api/metrics/live/servers/[uuid] + normalizacja PascalCase
  use-mobile.ts       — media query hook

types/index.ts        — JEDYNE źródło typów + funkcje normalizujące

e2e/                  — testy Playwright (auth.spec.ts)
```

---

## Zasada: Dark Mode Only (NIE MA Light Mode)

`<html className="dark">` jest **hardcoded** w `app/layout.tsx`. Klasa `.dark` jest permanentnie obecna — nie ma przełącznika.

**Dozwolone palety Tailwind** (zgodne z estetyką projektu):
- Tła: `bg-zinc-950`, `bg-zinc-900`, `bg-zinc-800`, `bg-background`
- Karty/panele: `bg-zinc-900/80`, `bg-zinc-950`, `bg-card`
- Borders: `border-zinc-800`, `border-zinc-700`, `border-border`
- Tekst główny: `text-zinc-100`, `text-zinc-200`, `text-foreground`
- Tekst pomocniczy: `text-zinc-400`, `text-zinc-500`, `text-muted-foreground`
- Akcentowe: emerald (online/OK), red (alert), amber (warning), blue/cyan (metryki)

**Zakaz:** `bg-white`, `bg-gray-100`, `text-black`, `dark:*` (variant dark jest zbędny — tryb ciemny jest jedynym).

Zamiast `dark:bg-zinc-800` → po prostu `bg-zinc-800`.

---

## KRYTYCZNE: Normalizacja Danych SSE

Backend Go ma **asymetrię formatu JSON** między dwoma SSE endpointami:

### `/api/metrics/live/all` → snake_case (OK, brak normalizacji)
```typescript
// LiveServerSnapshot — pola bezpośrednio używalne w JSX
snapshot.cpu          // ✅
snapshot.mem_percent  // ✅
```

### `/api/metrics/live/servers/[uuid]` → PascalCase (WYMAGA normalizacji)
```typescript
// LiveServerEvent.host — LiveServerHostRaw — NIGDY nie używaj bezpośrednio!
event.host.CPU         // ❌ ZAKAZ w JSX
event.host.MemUsed     // ❌ ZAKAZ w JSX

// Po normalizacji → LiveServerHost — snake_case, bezpieczne
const host = normalizeLiveHost(event.host)
host.cpu               // ✅
host.mem_used          // ✅
```

### Reguła obowiązkowa

Każdy nowy komponent lub hook pobierający dane z `/api/metrics/live/servers/[uuid]` **MUSI**:

1. Importować funkcje z `@/types`:
   ```typescript
   import { normalizeLiveHost, normalizeLiveContainer } from '@/types'
   import type { LiveServerHostRaw, LiveServerHost } from '@/types'
   ```

2. Wywołać normalizację przed przekazaniem danych do state/JSX:
   ```typescript
   const host: LiveServerHost = normalizeLiveHost(event.host)
   const containers = event.containers.map(normalizeLiveContainer)
   ```

3. Używać znormalizowanych typów w propsach komponentów:
   ```typescript
   // Props komponentu
   interface MetricWidgetProps {
     host: LiveServerHost          // ✅ znormalizowany
     // host: LiveServerHostRaw    // ❌ surowy PascalCase
   }
   ```

**Wzorzec referencyjny:** `hooks/useServerMetrics.ts` — kompletna implementacja z pre-fill historii i sliding window.

---

## Zasada: Local Knowledge First

Zanim sięgniesz po Context7 dla Lucide Icons / Shadcn / Tailwind / ECharts:

1. **Sprawdź istniejące komponenty** w `/frontend/components/` — zachowaj spójność wzorców
2. **Odczytaj** `types/index.ts` przed dodaniem nowych typów (unikaj duplikatów)
3. **Sprawdź** istniejący hook SSE (`useSSE.ts`, `useLiveAll.ts`) przed pisaniem nowego

Context7 używaj TYLKO gdy potrzebujesz API nieużywanego jeszcze w projekcie.

Przykład: "Jak użyć ikony Lucide?" → sprawdź `ServerCard.tsx` gdzie `import { CpuIcon } from 'lucide-react'`.

---

## Weryfikacja Playwright

Po każdej zmianie dotyczącej logicznych ścieżek użytkownika (formularze, nawigacja, przekierowania auth) uruchom testy E2E:

```bash
# Z katalogu /frontend
pnpm exec playwright test
```

Testy E2E: `frontend/e2e/auth.spec.ts` — weryfikują middleware auth, formularze login/setup, `data-testid` atrybuty.

**Wymóg `data-testid`:** Każdy interaktywny element formularza MUSI mieć atrybut `data-testid`, żeby testy mogły go zlokalizować. Wzorzec: `data-testid="login-username"`, `data-testid="setup-submit"`.

Konfiguracja: `playwright.config.ts` — baseURL `http://localhost:3000`, `pnpm dev` jako webServer, testDir `./e2e`.

---

## Self-Correction: Pętla testowa (3 próby)

Po każdej modyfikacji wykonaj z katalogu `/frontend/`:

```bash
pnpm test                        # Jest unit tests
pnpm exec playwright test        # E2E (gdy zmieniono UI / formularze)
```

| Próba | Wynik | Akcja |
|-------|-------|-------|
| 1 | FAIL | Analizuj błąd, popraw, próba 2 |
| 2 | FAIL | Analizuj błąd, popraw, próba 3 |
| 3 | FAIL | Przygotuj **Raport błędu testów** dla Orchestratora |

**Format Raportu błędu testów:**
```
## Raport błędu testów (próba 3/3)

### Polecenie
[pnpm test / pnpm exec playwright test]

### Pełny output
[wklej cały output]

### Analiza przyczyny
[co powoduje błąd — konkretnie]

### Próby naprawy (1–3)
[co zrobiłeś w każdej próbie]

### Rekomendacja
[co Orchestrator powinien zrobić dalej]
```

**Zakaz:** Subagent frontendowy NIE używa `AskUserQuestion`. Eskaluje tylko raportem do Orchestratora.

---

## Kluczowe wzorce kodu

### Dodawanie nowego komponentu z SSE `/api/metrics/live/servers/[uuid]`

```typescript
'use client'
import { useServerMetrics } from '@/hooks/useServerMetrics'

interface Props { uuid: string }

export function MyMetricWidget({ uuid }: Props) {
  const { hostPoints, connected } = useServerMetrics(uuid)
  const latest = hostPoints[hostPoints.length - 1]

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-4">
      <p className="text-zinc-400 text-xs uppercase tracking-wider">CPU</p>
      <p className="font-mono text-xl font-bold text-blue-400">
        {latest ? latest.cpu.toFixed(1) : '—'}%
      </p>
    </div>
  )
}
```

### Sygnały statusu (spójne z projektem)

```typescript
// Online/Offline
const status = online
  ? 'bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20'
  : 'bg-zinc-800 text-zinc-500'

// CPU load
const cpuColor = cpu >= 80 ? 'text-red-400' : cpu >= 60 ? 'text-amber-400' : 'text-blue-400'

// RAM
const memColor = mem >= 85 ? 'text-red-400' : mem >= 70 ? 'text-amber-400' : 'text-emerald-400'
```
