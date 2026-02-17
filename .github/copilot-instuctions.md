# Panel — Server Dashboard + Services Monitor

## Opis projektu

Ten projekt to zintegrowany panel do monitorowania serwerów i usług zewnętrznych. Składa się z trzech komponentów:

| Komponent | Język | Rola |
|-----------|-------|------|
| **Agent** | Go | Instalowany na każdym monitorowanym serwerze. Zbiera statystyki systemowe i wysyła je przez WebSocket do backendu (agent = klient WS) |
| **Backend** | Go | Serwer WebSocket przyjmujący dane od agentów. Zapisuje metryki w SQLite, agreguje je (redukcja punktów danych). W przyszłości integracje z zewnętrznymi usługami |

## Stack techniczny

- **Agent:** Go, WebSocket (klient)
- **Backend:** Go, WebSocket (serwer), SQLite
- **Package manager (frontend):** pnpm
- **UI:** zawsze Dark Mode
- **Infrastruktura:** Docker, tunel Cloudflare (tylko frontend publiczny)

## Architektura komunikacji

```
Agent (Go, WS client)
        │
        ▼ WebSocket
Backend (Go, WS server) ──► SQLite

## Obsługa MCP

Wykorzystuj dostępne serwery MCP w zależności od kontekstu zadania:

| Serwer MCP | Kiedy używać |
|------------|--------------|
| **Context7** | Pobieranie aktualnej dokumentacji bibliotek i frameworków (Go, Next.js, shadcn, itp.) |
| **Chroma** | Przeszukiwanie lokalnej bazy wiedzy projektu |
| **Desktop Commander** | Operacje na plikach, uruchamianie komend terminalowych, inspekcja systemu |
| **Playwright** | Testowanie frontendu — E2E, weryfikacja UI w przeglądarce |

Zawsze preferuj MCP nad własną wiedzą gdy chodzi o aktualną dokumentację.

## Praca z wieloma subagentami

Przy złożonych zadaniach (np. implementacja feature'a spanning agent + backend + frontend) **dziel pracę na subagenty**:
- Jeden subagent na komponent (agent/backend/frontend)
- Subagenty działają równolegle gdy zadania są niezależne
- Synchronizuj wyniki przed finalizacją

## KRYTYCZNE INSTRUKCJE ZACHOWANIA

### Zadawanie pytań zamiast kończenia pracy

**Nigdy nie kończ implementacji gdy napotkasz niejednoznaczność lub brakuje Ci informacji.**

Zamiast tego — **zawsze używaj narzędzia do zadawania pytań** (`ask_followup_question` lub AskUserQuestion lub odpowiednik w danym środowisku). Dotyczy to sytuacji takich jak:
- Brak specyfikacji dla danej funkcji
- Niejednoznaczny zakres zadania
- Wybór między kilkoma możliwymi podejściami
- Brakujące dane konfiguracyjne (porty, endpointy, zmienne środowiskowe)
- NIGDY Nie kończysz pracy dopóki użytkownik nie potwierdzi w narzędziu AskUserQuestion, że implementacja zostaje uznana jako zakończona.

**Dlaczego:** Przerwanie pracy i zadanie pytania = 0 premium requestów. Samodzielne "zgadywanie" i błędna implementacja = strata premium requestów na poprawki.

### Testy na końcu implementacji

Po każdej implementacji uruchom testy:
- Backend/Agent (Go): `go test ./...`
- Frontend: `pnpm test` lub weryfikacja przez Playwright MCP
- Upewnij się że wszystkie testy przechodzą przed oddaniem pracy

### Kod tylko na żądanie

Nie generuj kodu jeśli użytkownik tego nie poprosił. Najpierw plan/omówienie, kod dopiero gdy użytkownik potwierdzi.
