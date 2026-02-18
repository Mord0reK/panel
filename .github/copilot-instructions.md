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

## Obsługa MCP — zasada "MCP first, knowledge never"

**MCP to obowiązkowe narzędzia, nie opcja.** Własna wiedza AI jest zawsze przestarzała. MCP jest zawsze aktualne. Nie ma wyjątków.

### Obowiązkowe triggery MCP

| Sytuacja | Wymagane działanie |
|----------|--------------------|
| Jakakolwiek dokumentacja biblioteki / API / frameworka | **Context7 — zawsze, bez wyjątku** |
| Nie wiem jak wygląda istniejący plik lub funkcja | **Desktop Commander przed napisaniem linii kodu** |
| Napisałem coś we frontendzie | **Playwright — weryfikacja zanim powiesz że skończone** |
| Szukam jak coś jest zrobione w projekcie | **Chroma — przeszukaj bazę wiedzy projektu** |
| Operacje na plikach, terminal, inspekcja systemu | **Desktop Commander** |

**Zakaz zgadywania.** Jeśli nie masz pewności jak wygląda istniejący kod — sprawdź przez Desktop Commander. Jeśli potrzebujesz dokumentacji — użyj Context7. Nigdy nie pisz kodu który może kolidować z tym co już istnieje bez uprzedniego sprawdzenia.

## Praca z wieloma subagentami

Przy złożonych zadaniach (np. implementacja feature'a spanning agent + backend + frontend) **dziel pracę na subagenty**:
- Jeden subagent na komponent (agent/backend/frontend)
- Subagenty działają równolegle gdy zadania są niezależne
- Synchronizuj wyniki przed finalizacją

## KRYTYCZNE INSTRUKCJE ZACHOWANIA

### AskUserQuestion — jedyny kanał komunikacji

**Twardy zakaz: nigdy nie zadajesz pytań w treści wiadomości tekstowej.**

Jedyną dozwoloną formą pytania do użytkownika jest narzędzie `AskUserQuestion`. Tekst w odpowiedzi służy wyłącznie do raportowania — nie do dialogu. Masz binarny wybór: albo użyj narzędzia `AskUserQuestion`, albo kontynuuj pracę. Nie ma trzeciej opcji "napiszę pytanie w tekście i poczekam".

**`AskUserQuestion` jest dozwolone wyłącznie gdy:**
- Decyzja architektoniczna jest nieodwracalna i nie ma dobrego domyślnego wyboru
- Brakuje kredencjałów / sekretów których nie ma w repozytorium
- Zadanie jest sprzeczne z istniejącym kodem i żaden default nie jest oczywisty

**Wszystko inne** — użyj Desktop Commander, sprawdź repo, zaimplementuj, zaraportuj.

### Definicja "zadanie skończone"

Zadanie jest skończone dopiero gdy wszystkie poniższe punkty są spełnione:

1. **Kod napisany** — implementacja gotowa
2. **Testy przeszły** — `go test ./...` (backend/agent) lub `pnpm test` (frontend)
3. **UI zweryfikowane przez Playwright** — jeśli zadanie dotyczy frontendu
4. **Podsumowanie wysłane przez `AskUserQuestion`** — z pytaniem czy użytkownik akceptuje wynik

Nie możesz uznać zadania za skończone po samym napisaniu kodu. Playwright i testy są obowiązkowym warunkiem ukończenia, nie opcją.

### Zadawanie pytań zamiast kończenia pracy

**Nigdy nie kończ implementacji gdy napotkasz niejednoznaczność lub brakuje Ci informacji.**

Zamiast tego — sprawdź przez Desktop Commander / Chroma / Context7. Jeśli nadal nie wiesz — użyj `AskUserQuestion`. Dotyczy to sytuacji takich jak:
- Brak specyfikacji dla danej funkcji
- Niejednoznaczny zakres zadania
- Wybór między kilkoma możliwymi podejściami
- Brakujące dane konfiguracyjne (porty, endpointy, zmienne środowiskowe)

**Dlaczego:** Przerwanie pracy i zadanie pytania = 0 premium requestów. Samodzielne "zgadywanie" i błędna implementacja = strata premium requestów na poprawki.

### Testy na końcu implementacji

Po każdej implementacji uruchom testy:
- Backend/Agent (Go): `go test ./...`
- Frontend: `pnpm test` lub weryfikacja przez Playwright MCP
- Upewnij się że wszystkie testy przechodzą przed oddaniem pracy

### Kod tylko na żądanie

Nie generuj kodu jeśli użytkownik tego nie poprosił. Najpierw plan/omówienie, kod dopiero gdy użytkownik potwierdzi.
