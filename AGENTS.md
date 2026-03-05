# SYSTEM PROMPT: Karta Projektu i Protokoły Agenta

## 1. INICJALIZACJA I EKSPLORACJA (Zawsze przed startem)
1. **Zarządzanie skillami:** Użyj `skill list` (TYLKO OPENCODE) lub dedykowanego narzędzia środowiska (np. w GitHub Copilot), aby sprawdzić dostępne umiejętności.
2. **Załaduj skille:** Jeśli zadanie dotyczy frontendu, MUSISZ załadować `panel-frontend` (używając `skill load` lub odpowiednika w Twoim narzędziu).
3. **Eksploracja:** Używaj Desktop Commander do czytania struktury i plików. Zakaz zgadywania i ślepego przeszukiwania dysku.

## 2. KONTEKST PROJEKTU: Panel
Zintegrowany system monitoringu serwerów:
- **Agent (Go):** Klient WebSocket, zbiera metryki systemowe.
- **Backend (Go):** Serwer WebSocket, baza SQLite, agregacja danych.
- **Frontend (pnpm):** UI Dark Mode. Infrastruktura: Docker + Cloudflare tunnel.
- **Subagenty:** Przy pracach cross-komponentowych dziel zadania na subagenty (agent/backend/frontend).

## 3. OBOWIĄZKOWE NARZĘDZIA (MCP First)
Twoja wewnętrzna wiedza jest traktowana jako przestarzała. Używaj:
- **Context7:** Dokumentacja bibliotek/API/frameworków – zawsze przed kodowaniem.
- **Playwright:** Weryfikacja UI po każdej zmianie we frontendzie.
- **Desktop Commander:** Weryfikacja istniejącego kodu przed edycją.

## 4. PROTOKÓŁ KOMUNIKACJI: TOTALNA BLOKADA TEKSTU

**KRYTYCZNA ZASADA (Zero-Tolerance):** Masz CAŁKOWITY ZAKAZ wypisywania sekcji "Zrobione", "Weryfikacja" lub jakichkolwiek pytań końcowych bezpośrednio w oknie czatu (Markdown). 

### TWOJA JEDYNA DROGA WYJŚCIA:
Każdy raport z pracy i każde pytanie o akceptację MUSI być przesłane jako argument `message` w narzędziu `AskUserQuestion`. 

**Jeśli skończyłeś pracę:**
1. Twoja odpowiedź tekstowa w czacie musi być PUSTA (0 znaków).
2. Natychmiast wywołaj `AskUserQuestion`.
3. W polu `message` wpisz:
   """
   # RAPORT KOŃCOWY
   ## Zrobione:
   - [lista zmian]
   ## Weryfikacja:
   - [wyniki testów/Playwright]
   
   CZY AKCEPTUJESZ WYNIK? (Jeśli nie, napisz co poprawić).
   """

**BŁĄD KRYTYCZNY:** Jeśli w Twojej odpowiedzi widać sekcję "Zrobione" w Markdownie (tak jak na screenach użytkownika), oznacza to, że złamałeś protokół i zmarnowałeś request. Napraw to natychmiast używając narzędzia.

### A. Blokada Przed Implementacją (Oszczędność requestów)
ZATRZYMAJ PRACĘ i wywołaj `AskUserQuestion`, jeśli:
- Brakuje specyfikacji/danych (sekrety, porty, endpointy).
- Zakres zadania jest sprzeczny z istniejącym kodem.
- Decyzja architektoniczna jest nieodwracalna.

### B. Blokada Po Implementacji (Raportowanie)
NIGDY nie wysyłaj raportu końcowego jako zwykłego tekstu. 
**Prawidłowy przepływ:**
1. Przygotuj listę "Zrobione" i "Weryfikacja".
2. Całą tę treść przekaż jako parametr `message` w narzędziu `AskUserQuestion`.
3. Na końcu raportu dodaj pytanie: "Czy akceptujesz wynik?".

## 5. DEFINICJA UKOŃCZENIA (Definition of Done)
Zadanie jest "DONE" tylko, gdy wszystkie punkty zostaną odhaczone w tej kolejności:
1. **Kod napisany** (implementacja gotowa).
2. **Testy zaliczone:** `go test ./...` (backend/agent) lub `pnpm test` (frontend).
3. **Weryfikacja wizualna:** Potwierdzona przez Playwright (dla frontendu).
4. **Techniczne zamknięcie:** Wywołanie `AskUserQuestion` z pełnym raportem zmian i testów.

**ZAKAZ:** Wysyłania raportu w Markdown i czekania na odpowiedź bez wywołania narzędzia.