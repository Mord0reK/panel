---

## Etap 1 — Fundament (bez UI)
- `lib/backend.ts`
- `lib/api.ts`
- `lib/formatters.ts`
- Wszystkie Route Handlers w `app/api/`
- **Konfiguracja Jest + React Testing Library + Playwright**
- **Testy:** `lib/formatters.test.ts` — unit testy wszystkich funkcji formatujących

**Weryfikacja:** `pnpm build` + `pnpm test` przechodzą

---

## Etap 2 — Auth
- `app/(auth)/layout.tsx`
- `app/(auth)/login/page.tsx`
- `app/(auth)/setup/page.tsx`
- `app/page.tsx`
- **Testy E2E (Playwright):** flow logowania, redirect po setupie, middleware blokuje niezalogowanych

**Weryfikacja:** można się zalogować, Playwright przechodzi

---

## Etap 3 — Layout + Sidebar
- `app/(app)/layout.tsx`
- `components/sidebar/Sidebar.tsx`
- `components/sidebar/ServerNavItem.tsx`
- `components/sidebar/ServerNavList.tsx`
- **Testy:** `ServerNavItem` — czy collapsible otwiera/zamyka, czy aktywna trasa jest podświetlona

**Weryfikacja:** sidebar działa, testy komponentów przechodzą

---

## Etap 4 — Dashboard
- `hooks/useLiveAll.ts`
- `components/servers/ServerCard.tsx`
- `app/(app)/dashboard/page.tsx`
- **Testy:** `useLiveAll` — mock SSE, weryfikacja parsowania eventów i reconnect; `ServerCard` — renderowanie z danymi i bez

**Weryfikacja:** karty aktualizują się live, testy hooków przechodzą

---

## Etap 5 — Widok metryk serwera
- `hooks/useSSE.ts`
- `hooks/useServerMetrics.ts`
- `components/metrics/RangeDropdown.tsx`
- `components/charts/LiveChart.tsx`
- `components/charts/HistoryChart.tsx`
- `components/metrics/MetricsGrid.tsx`
- `components/servers/ServerInfo.tsx`
- `app/(app)/servers/[uuid]/metrics/page.tsx`
- **Testy:** `useServerMetrics` — normalizacja PascalCase → snake_case, bufor 60 punktów; `RangeDropdown` — przełączanie zakresów, grupowanie opcji

**Weryfikacja:** live wykres działa, testy przechodzą

---

## Etap 6 — Widok kontenerów
- `components/containers/ContainersTable.tsx`
- `components/containers/ContainerActions.tsx`
- `app/(app)/servers/[uuid]/containers/page.tsx`
- **Testy:** `ContainerActions` — disabled podczas oczekiwania, obsługa timeout 30s; **E2E (Playwright):** pełny flow start/stop/restart kontenera

**Weryfikacja:** akcje działają, E2E przechodzi

---
