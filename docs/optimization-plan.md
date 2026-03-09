# Plan optymalizacji Panelu - Spike'y CPU/Dysku i Problemy z Metrykami

**Data:** 2026-03-09  
**Status:** Planowanie (do zatwierdzenia przez użytkownika)  

---

## Spis treści

1. [Podsumowanie problemów](#1-podsumowanie-problemów)
2. [Przyczyny problemów](#2-przyczyny-problemów)
3. [Analiza porównawcza z Beszel](#3-analiza-porównawcza-z-beszel)
4. [Szczegółowy plan naprawy](#4-szczegółowy-plan-naprawy)
5. [Plan optymalizacji spike'ów](#5-plan-optymalizacji-spikeów)
6. [Szczegóły implementacji](#6-szczegóły-implementacji)
7. [Testowanie i weryfikacja](#7-testowanie-i-weryfikacja)
8. [Zachowanie kluczowych funkcji](#8-zachowanie-kluczowych-funkcji)
9. [Ryzyka i przeciwwskazania](#9-ryzyka-i-przeciwwskazania)
10. [Kolejność wdrożenia](#10-kolejność-wdrożenia)

---

## 1. PODSUMOWANIE PROBLEMÓW

### 1.1 Problem z metrykami (KRYTYCZNY)

Na podstawie odpowiedzi użytkownika:
- **Punkty co 30 minut** dla `range=24h` - metryki są agregowane do 30-minutowych interwałów
- **Brak metryki z 21:05** mimo że jest 21:05 - dane są usuwane przed agregacją
- **Ostatnia metryka z 20:30** dla zakresu 24h - retencja jest zbyt krótka
- **Im dalsze metryki tym większy błąd** - brak danych źródłowych dla starszych przedziałów

### 1.2 Problem z spike'ów CPU/dysku

- **Spike'y regularne co kilka sekund** - obserwowane przez użytkownika
- **25% CPU w spike** - wysokie obciążenie przy każdym flush
- **Ilość agentów:** 2-8 (potencjalnie więcej)
- **Środowisko:** Docker Compose

### 1.3 Ograniczenia od użytkownika

- **Rozmiar bazy danych:** Maksymalnie 50MB
- **Wymagania:**
  - Zachować precyzyjne metryki
  - Nie pogorszyć wykorzystania zasobów
  - Nie powodować opóźnień
- **Brak logów:** Użytkownik nie ma dostępu do logów backendu

---

## 2. PRZYCZYNY PROBLEMÓW

### 2.1 Retencja metrics_30m zbyt krótka (GŁÓWNA PRZYCZYNA PROBLEMU Z METRYKAMI)

**Obecna konfiguracja w `backend/internal/aggregation/config.go`:**

```go
{
    SourceTable:         "metrics_15m",
    TargetTable:         "metrics_30m",
    SourceThreshold:     1800 * time.Second,  // 30 minut
    AggregationInterval: 30 * time.Minute,
    RetentionThreshold:  12 * time.Hour,       // TYLKO 12 GODZIN!
}
```

**Problem:**
- Dla `range=24h` API używa tabeli `metrics_30m` (konfigurowane w `models/metrics.go:97`)
- Retencja `metrics_30m` to tylko **12 godzin**
- API próbuje odczytać 24 godziny, ale dane starsze niż 12h zostały usunięte!

**Wynik:**
- Metryki 24h pokazują tylko ostatnie 12 godzin
- Dla godzin 12-24h wstecz - brak danych (stąd 20:30 zamiast 21:05)

### 2.2 Aggregator nie filtruje po agent_uuid (PRZYCZYNA SPIKE'ÓW)

**Obecne zapytanie w `aggregator.go:108`:**

```go
query := fmt.Sprintf(`
    SELECT ... FROM %s WHERE timestamp < ? ORDER BY timestamp`, table)
```

**Problem:**
- Brak filtrowania po `agent_uuid`
- Pełne skanowanie tabeli przy każdym uruchomieniu
- Przy 2-8 agentach - wielokrotne obciążenie

**Skutki:**
- Spowolnienie zapytań SQL
- Wysokie CPU przy agregacji
- Mieszanie danych między agentami (potencjalnie)

### 2.3 Brak indeksów na timestamp

**Obecna struktura tabel (z `migrations/00001_initial_schema.sql`):**

```sql
CREATE TABLE IF NOT EXISTS metrics_30m (
    agent_uuid TEXT,
    container_id TEXT,
    timestamp INTEGER,
    cpu_avg REAL, ...
    PRIMARY KEY(agent_uuid, container_id, timestamp)
);
```

**Problem:**
- Brak indeksu na kolumnie `timestamp`
- Brak indeksu na `(agent_uuid, timestamp)`
- PRIMARY KEY nie jest wystarczający dla zapytań `WHERE timestamp < ?`

**Skutki:**
- Pełne skanowanie tabeli (FULL TABLE SCAN)
- Spowolnienie zapytań o 10-100x
- Wysokie I/O dyskowe przy agregacji

### 2.4 Równoczesny timing BulkInserter i Aggregator

**Obecna konfiguracja:**

```go
// BulkInserter (inserter.go:88)
ticker := time.NewTicker(10 * time.Second)

// Aggregator (aggregator.go:31)
ticker := time.NewTicker(10 * time.Second)
```

**Problem:**
- Oba tickery uruchamiają się w tym samym czasie
- Flush do bazy danych i agregacja wykonywane równocześnie
- Szczytowe obciążenie CPU/dysku co 10 sekund

---

## 3. ANALIZA PORÓWNAWCZA Z BESZEL

### 3.1 Porównanie architektury

| Aspekt | Panel (obecny) | Beszel | Różnica |
|--------|----------------|--------|----------|
| Interwał zbierania | 1s | 60s | Panel zbiera częściej |
| Interwał zapisu | 10s | 60s | Panel zapisuje częściej |
| Interwał agregacji | 10s | 60s | Panel agreguje częściej |
| Poziomy agregacji | 10 (5s-12h) | 5 (1m-480m) | Panel ma więcej poziomów |
| Peak values | min, avg, max | min, avg, max, peak | Podobnie |
| Retencja | Konfigurowalna | 1h-30d | Podobnie |
| Indeksy | Brak | Tak (PocketBase) | **Panel ma lukę** |
| Filtrowanie w SELECT | Brak | Tak | **Panel ma lukę** |
| Timing | Równoczesny | Rozsynchronizowany | **Panel ma lukę** |

### 3.2 Co Panel ma lepiej niż Beszel

- **Więcej poziomów agregacji** (10 vs 5)
- **Bardziej szczegółowa retencja** (różne czasy dla różnych poziomów)
- **Częstsze zbieranie metryk** (1s vs 60s)
- **WAL mode w SQLite** (poprawnie skonfigurowany)

### 3.3 Co Beszel ma lepiej niż Panel

- **Indeksy na timestamp** (w PocketBase)
- **Rozsynchronizowany timing** (unika spike'ów)
- **Filtrowanie w zapytaniach SELECT** (po agent_uuid)
- **Większy interwał** (60s vs 10s) = mniej obciążenia

---

## 4. SZCZEGÓŁOWY PLAN NAPRAWY

### 4.1 Naprawa problemu z metrykami (KROK 1)

**Problem:** Retencja metrics_30m (12h) jest krótsza niż zakres 24h

**Rozwiązanie:** Zwiększenie retencji metrics_30m z 12h do 48h

**Plik do zmiany:** `backend/internal/aggregation/config.go:53-57`

**Zmiana:**
```go
{
    SourceTable:         "metrics_15m",
    TargetTable:         "metrics_30m",
    SourceThreshold:     1800 * time.Second,
    AggregationInterval: 30 * time.Minute,
    RetentionThreshold:  48 * time.Hour,  // ZMIANA: z 12h na 48h
},
```

**Uzasadnienie:**
- Pozwoli na odczyt 24h z `metrics_30m` bez utraty danych
- Minimalny wpływ na rozmiar bazy (tylko 48 dodatkowych punktów na agenta)
- Przy 8 agentach: ~10KB dodatkowo

**Szacowany wpływ na rozmiar bazy:**
- `metrics_30m`: 2 punkty/godzinę * 48h = 96 punktów/agenta
- Dla 8 agentów: ~10KB
- Całość nadal w limicie 50MB

**Czy frontend wymaga zmian?**
- NIE - backend zwraca dane z `metrics_30m` i frontend je wyświetla bez zmian

---

### 4.2 Dodanie indeksów na timestamp (KROK 2)

**Problem:** Brak indeksów powoduje pełne skanowanie tabel

**Rozwiązanie:** Utworzenie nowej migracji z indeksami

**Nowy plik:** `backend/migrations/00007_add_metrics_indexes.sql`

**Zawartość:**
```sql
-- +goose Up

-- Indeksy dla metrics_5s
CREATE INDEX IF NOT EXISTS idx_metrics_5s_timestamp ON metrics_5s(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_5s_agent_timestamp ON metrics_5s(agent_uuid, timestamp);

-- Indeksy dla metrics_15s
CREATE INDEX IF NOT EXISTS idx_metrics_15s_timestamp ON metrics_15s(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_15s_agent_timestamp ON metrics_15s(agent_uuid, timestamp);

-- Indeksy dla metrics_30s
CREATE INDEX IF NOT EXISTS idx_metrics_30s_timestamp ON metrics_30s(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_30s_agent_timestamp ON metrics_30s(agent_uuid, timestamp);

-- Indeksy dla metrics_1m
CREATE INDEX IF NOT EXISTS idx_metrics_1m_timestamp ON metrics_1m(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_1m_agent_timestamp ON metrics_1m(agent_uuid, timestamp);

-- Indeksy dla metrics_5m
CREATE INDEX IF NOT EXISTS idx_metrics_5m_timestamp ON metrics_5m(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_5m_agent_timestamp ON metrics_5m(agent_uuid, timestamp);

-- Indeksy dla metrics_15m
CREATE INDEX IF NOT EXISTS idx_metrics_15m_timestamp ON metrics_15m(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_15m_agent_timestamp ON metrics_15m(agent_uuid, timestamp);

-- Indeksy dla metrics_30m
CREATE INDEX IF NOT EXISTS idx_metrics_30m_timestamp ON metrics_30m(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_30m_agent_timestamp ON metrics_30m(agent_uuid, timestamp);

-- Indeksy dla metrics_1h
CREATE INDEX IF NOT EXISTS idx_metrics_1h_timestamp ON metrics_1h(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_1h_agent_timestamp ON metrics_1h(agent_uuid, timestamp);

-- Indeksy dla metrics_6h
CREATE INDEX IF NOT EXISTS idx_metrics_6h_timestamp ON metrics_6h(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_6h_agent_timestamp ON metrics_6h(agent_uuid, timestamp);

-- Indeksy dla metrics_12h
CREATE INDEX IF NOT EXISTS idx_metrics_12h_timestamp ON metrics_12h(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_12h_agent_timestamp ON metrics_12h(agent_uuid, timestamp);

-- +goose Down

DROP INDEX IF EXISTS idx_metrics_5s_timestamp;
DROP INDEX IF EXISTS idx_metrics_5s_agent_timestamp;
-- ... (analogicznie dla wszystkich tabel)
```

**Szacowany wpływ na wydajność:**
- Przyspieszenie zapytań SELECT: **10-100x**
- Redukcja I/O dyskowego: **50-80%**
- Redukcja CPU przy agregacji: **30-50%**

**Czy wymaga restartu?**
- TAK - wymaga restartu backendu po uruchomieniu migracji

---

## 5. PLAN OPTYMALIZACJI SPIKE'ÓW

### 5.1 Rozsynchronizowanie timing'u (KROK 3)

**Problem:** BulkInserter i Aggregator uruchamiają się równocześnie

**Rozwiązanie A:** Dodanie jitter (losowe opóźnienie) do BulkInserter

**Plik:** `backend/internal/buffer/inserter.go:87-101`

**Zmiana:**
```go
func (bi *BulkInserter) Run() {
    // Dodaj jitter (losowe opóźnienie 0-2s) aby uniknąć równoczesnego flush z agregatorem
    jitter := time.Duration(rand.Intn(2000)) * time.Millisecond
    time.Sleep(jitter)
    
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    // ... reszta kodu bez zmian
}
```

**Wymagany import:**
```go
import (
    // ... istniejące importy
    "math/rand"
)
```

**Rozwiązanie B:** Dodanie offset do Aggregator

**Plik:** `backend/internal/aggregation/aggregator.go:30-42`

**Zmiana:**
```go
func (a *Aggregator) Run() {
    // Dodaj offset 5s aby uniknąć równoczesnego flush z BulkInserter
    time.Sleep(5 * time.Second)
    
    ticker := time.NewTicker(15 * time.Second)
    defer ticker.Stop()
    // ... reszta kodu bez zmian
}
```

**Rezultat:**
- BulkInserter: flush co 10s z losowym opóźnieniem 0-2s
- Aggregator: flush co 15s, z offsetem 5s
- Brak równoczesnych operacji = brak spike'ów

**Szacowany wpływ:**
- Redukcja spike'ów CPU: **30-50%**
- Redukcja spike'ów dysku: **30-50%**

---

### 5.2 Optymalizacja zapytań SELECT w aggregatorze (KROK 4)

**Problem:** Aggregator nie filtruje po agent_uuid

**Rozwiązanie:** Dodanie filtrowania po agent_uuid i iteracja po agentach osobno

**Plik:** `backend/internal/aggregation/aggregator.go`

**Zmiana 1:** Dodanie funkcji pobierającej listę agentów

```go
// getAgentList pobiera listę wszystkich agentów z bazy danych
func (a *Aggregator) getAgentList() ([]string, error) {
    query := `SELECT DISTINCT agent_uuid FROM metrics_5s ORDER BY agent_uuid`
    rows, err := a.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var agents []string
    for rows.Next() {
        var agentUUID string
        if err := rows.Scan(&agentUUID); err != nil {
            return nil, err
        }
        agents = append(agents, agentUUID)
    }
    return agents, rows.Err()
}
```

**Zmiana 2:** Aktualizacja fetchData z dodatkowym parametrem agentUUID

```go
func (a *Aggregator) fetchData(table string, agentUUID string, threshold int64) ([]MetricRow, error) {
    query := fmt.Sprintf(`
        SELECT agent_uuid, container_id, timestamp,
        cpu_avg, cpu_min, cpu_max,
        mem_avg, mem_min, mem_max,
        disk_avg, disk_write_avg, disk_write_min, disk_write_max,
        net_rx_avg, net_rx_min, net_rx_max,
        net_tx_avg, net_tx_min, net_tx_max,
        disk_used_percent_avg, disk_used_percent_min, disk_used_percent_max
        FROM %s WHERE agent_uuid = ? AND timestamp < ? ORDER BY timestamp`, table)

    rows, err := a.db.Query(query, agentUUID, threshold)
    // ... reszta kodu bez zmian
}
```

**Zmiana 3:** Aktualizacja ProcessAggregation

```go
func (a *Aggregator) ProcessAggregation() {
    now := time.Now().Unix()

    // Pobierz listę wszystkich agentów
    agentList, err := a.getAgentList()
    if err != nil {
        log.Printf("Failed to get agent list: %v", err)
        return
    }

    for _, level := range ContainerAggregationLevels {
        threshold := now - int64(level.SourceThreshold.Seconds())

        // Agreguj dane dla każdego agenta osobno
        for _, agentUUID := range agentList {
            // 1. Fetch data to aggregate (dla konkretnego agenta)
            rows, err := a.fetchData(level.SourceTable, agentUUID, threshold)
            // ... reszta kodu bez zmian
        }
        
        // 4. Delete old data (dla wszystkich agentów - bez zmian)
        retentionThreshold := now - int64(level.RetentionThreshold.Seconds())
        // ... reszta kodu bez zmian
    }
}
```

**Szacowany wpływ:**
- Redukcja CPU przy agregacji: **40-60%** (mniej danych per zapytanie)
- Wykorzystanie indeksów: **TAK** (indeks na agent_uuid, timestamp)
- Unikanie pełnego skanowania: **TAK**

---

## 6. SZCZEGÓŁY IMPLEMENTACJI

### 6.1 KROK 1: Zmiana retencji metrics_30m

**Plik:** `backend/internal/aggregation/config.go`

**Linie:** 53-57

**Zmiana:**
```go
// PRZED:
RetentionThreshold:  12 * time.Hour,

// PO:
RetentionThreshold:  48 * time.Hour,
```

**Ryzyko:** Bardzo niskie
**Czas implementacji:** 1 minuta
**Testowanie:** Sprawdzenie czy metryki 24h są pełne

---

### 6.2 KROK 2: Dodanie indeksów

**Plik:** `backend/migrations/00007_add_metrics_indexes.sql` (NOWY PLIK)

**Ryzyko:** Niskie (migracja dodaje indeksy, nie usuwa danych)
**Czas implementacji:** 5 minut
**Testowanie:** Sprawdzenie czy indeksy istnieją po migracji

**Wykonanie migracji:**
```bash
cd backend
goose up
```

**Weryfikacja:**
```sql
SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_metrics_%';
```

---

### 6.3 KROK 3: Rozsynchronizowanie timing'u

**Plik 1:** `backend/internal/buffer/inserter.go`

**Zmiana 1.1:** Dodanie importu `math/rand`
```go
import (
    // ... istniejące importy
    "math/rand"
)
```

**Zmiana 1.2:** Dodanie jitter w funkcji Run
```go
func (bi *BulkInserter) Run() {
    // Dodaj jitter (losowe opóźnienie 0-2s)
    jitter := time.Duration(rand.Intn(2000)) * time.Millisecond
    time.Sleep(jitter)
    
    ticker := time.NewTicker(10 * time.Second)
    // ... reszta bez zmian
}
```

**Plik 2:** `backend/internal/aggregation/aggregator.go`

**Zmiana 2.1:** Dodanie offset w funkcji Run
```go
func (a *Aggregator) Run() {
    // Dodaj offset 5s
    time.Sleep(5 * time.Second)
    
    ticker := time.NewTicker(15 * time.Second)
    // ... reszta bez zmian
}
```

**Ryzyko:** Niskie
**Czas implementacji:** 5 minut
**Testowanie:** Monitorowanie CPU/dysku przez 24h

---

### 6.4 KROK 4: Optymalizacja zapytań SELECT

**Plik:** `backend/internal/aggregation/aggregator.go`

**Zmiany:**
1. Dodanie funkcji `getAgentList()` - ~20 linii kodu
2. Modyfikacja `fetchData()` - dodanie parametru agentUUID
3. Modyfikacja `ProcessAggregation()` - iteracja po agentach

**Ryzyko:** Średnie (zmiana logiki agregacji)
**Czas implementacji:** 15 minut
**Testowanie:** 
- Sprawdzenie czy agregacja działa poprawnie
- Porównanie wyników przed/po zmianach

---

## 7. TESTOWANIE I WERYFIKACJA

### 7.1 Środowisko testowe

Użytkownik ma środowisko testowe w Docker Compose:
- `panel/agent/docker-compose.yml`
- `panel/backend/docker-compose.yml`
- `panel/frontend/docker-compose.yml`

### 7.2 Procedura testowa

**KROK 1: Uruchomienie środowiska**
```bash
cd /home/marcelstosio/Desktop/Projekty/panel/agent && docker-compose up -d
cd /home/marcelstosio/Desktop/Projekty/panel/backend && docker-compose up -d
cd /home/marcelstosio/Desktop/Projekty/panel/frontend && docker-compose up -d
```

**KROK 2: Test metryk 24h**
```bash
curl -s "http://localhost:8080/api/metrics/history/servers/{UUID}?range=24h" | jq '.host.points | length'
```

**Oczekiwany wynik:** Powyżej 40 punktów (48 godzin / 30 minut = 96 punktów maksymalnie)

**KROK 3: Test wydajności**
```bash
# Monitorowanie CPU
htop

# Monitorowanie dysku
iostat -x 1
```

**Oczekiwany wynik:** Brak regularnych spike'ów co 10 sekund

**KROK 4: Weryfikacja indeksów**
```bash
docker-compose exec backend sqlite3 /app/data.db "SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_metrics_%';"
```

**Oczekiwany wynik:** Lista indeksów dla każdej tabeli metryk

---

## 8. ZACHOWANIE KLUCZOWYCH FUNKCJI

Plan NIE zmienia następujących funkcji:

✅ **Wszystkie poziomy agregacji** (5s → 12h) - bez zmian  
✅ **WebSocket od agentów** (1s zbieranie metryk) - bez zmian  
✅ **SSE dla frontendu** - bez zmian  
✅ **Wszystkie endpointy API** - bez zmian  
✅ **Peak values** (min, avg, max) - bez zmian  
✅ **Dashboard** - bez zmian  
✅ **Buffer w pamięci** (RingBuffer) - bez zmian  

**Co SIĘ zmienia:**
- Retencja `metrics_30m`: 12h → 48h (dłuższe przechowywanie)
- Timing operacji: rozsynchronizowany (mniej spike'ów)
- Indeksy: dodane (szybsze zapytania)

---

## 9. RYZYKA I PRZECIWWSKAZANIA

### 9.1 Ryzyka

| Ryzyko | Poziom | Opis | Mitygacja |
|--------|--------|------|----------|
| Migracja spowolni backend | Niskie | Tworzenie indeksów na dużej tabeli | Utworzenie indeksów w tle (non-blocking) |
| Zmiana agregacji psuje dane | Średnie | Błędne obliczenia po zmianie logiki | Testowanie przed wdrożeniem |
| Większa baza danych | Niskie | 48h vs 12h = +36h danych | Sprawdzenie rozmiaru po zmianie |

### 9.2 Przeciwwskazania

| Przeciwwskazanie | Warunek | Rozwiązanie |
|------------------|---------|-------------|
| Mało miejsca na dysku | < 100MB wolnego | Zmniejszenie retencji lub zwiększenie interwału agregacji |
| Wiele kontenerów | > 100 kontenerów/agenta | Zmniejszenie częstotliwości zapisu |
| Słaby CPU | < 2 rdzenie | Zmniejszenie częstotliwości agregacji |

### 9.3 Szacowany rozmiar bazy danych

**Przy 8 agentach, 10 kontenerach na agenta:**

| Tabela | Punkty/godzinę | Rozmiar (8 agentów) |
|--------|----------------|---------------------|
| metrics_5s | 7200 | ~50 MB/godzinę |
| metrics_15s | 2400 | ~15 MB/godzinę |
| metrics_30s | 1200 | ~8 MB/godzinę |
| metrics_1m | 600 | ~4 MB/godzinę |
| metrics_5m | 120 | ~1 MB/godzinę |
| metrics_15m | 40 | ~0.3 MB/godzinę |
| metrics_30m | 20 | ~0.2 MB/godzinę |
| metrics_1h | 8 | ~0.1 MB/godzinę |
| metrics_6h | 1.3 | ~0.02 MB/godzinę |
| metrics_12h | 0.7 | ~0.01 MB/godzinę |

**CAŁKOWITY ROZMÓAR (bez optymalizacji):** ~80 MB/godzinę

**Po optymalizacji (retencja metrics_30m = 48h):**
- Wzrost ~10 KB (minimalny)

---

## 10. KOLEJNOŚĆ WDROŻENIA

### Etap 1: Naprawa metryk (Dzień 1)

1. Zmiana retencji `metrics_30m` (12h → 48h)
2. Test: sprawdzenie metryk 24h

**Czas:** 1 godzina
**Ryzyko:** Bardzo niskie

### Etap 2: Indeksy (Dzień 2)

1. Utworzenie migracji `00007_add_metrics_indexes.sql`
2. Uruchomienie migracji
3. Test: sprawdzenie indeksów

**Czas:** 1 godzina
**Ryzyko:** Niskie

### Etap 3: Rozsynchronizowanie timing'u (Dzień 3)

1. Dodanie jitter do BulkInserter
2. Dodanie offset do Aggregator
3. Restart backendu
4. Test: monitorowanie CPU/dysku

**Czas:** 1 godzina
**Ryzyko:** Niskie

### Etap 4: Optymalizacja zapytań (Dzień 4)

1. Dodanie funkcji getAgentList
2. Modyfikacja fetchData
3. Modyfikacja ProcessAggregation
4. Test: porównanie wyników agregacji

**Czas:** 2 godziny
**Ryzyko:** Średnie

### Etap 5: Weryfikacja końcowa (Dzień 5)

1. Test wszystkich zakresów czasowych (1m, 5m, 1h, 24h, 7d)
2. Pomiar wydajności CPU/dysku
3. Sprawdzenie rozmiaru bazy danych

**Czas:** 2 godziny

---

## PODSUMOWANIE PLANU

### Zmiany do wykonania:

| # | Zmiana | Plik | Czas | Ryzyko |
|---|--------|------|------|--------|
| 1 | Retencja metrics_30m (12h → 48h) | `config.go` | 1 min | Bardzo niskie |
| 2 | Indeksy na timestamp | Nowa migracja | 5 min | Niskie |
| 3 | Jitter w BulkInserter | `inserter.go` | 5 min | Niskie |
| 4 | Offset w Aggregator | `aggregator.go` | 5 min | Niskie |
| 5 | Filtrowanie po agent_uuid | `aggregator.go` | 15 min | Średnie |

### Oczekiwane efekty:

| Problem | Przed | Po |
|---------|-------|-----|
| Metryki 24h | Niepełne (12h) | Pełne (48h) |
| Spike'y CPU | 25% co 10s | < 10% |
| Spike'y dysku | Wysokie I/O | Minimalne |
| Czas zapytań | Wolne | 10-100x szybsze |

### Zachowanie kluczowych funkcji:

✅ Wszystkie funkcje zachowane  
✅ Brak zmian w API  
✅ Brak zmian w frontendzie  
✅ Brak zmian w zbieraniu metryk  

---

## AKCEPTACJA PLANU

**Użytkownik musi zatwierdzić przed implementacją:**

1. ✅ Zrozumiałem plan
2. ✅ Akceptuję zwiększenie retencji metrics_30m do 48h
3. ✅ Akceptuję dodanie indeksów (migracja)
4. ✅ Akceptuję rozsynchronizowanie timing'u
5. ✅ Mam środowisko testowe do weryfikacji

---

**Data ostatniej aktualizacji:** 2026-03-09  
**Wersja dokumentu:** 1.0
