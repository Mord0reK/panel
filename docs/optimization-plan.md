# Plan optymalizacji Panelu - Usuwanie spike'ów CPU/I/O

**Data:** 2026-03-10  
**Status:** Planowanie (do zatwierdzenia przez użytkownika)  

---

## Spis treści

1. [Podsumowanie problemu](#1-podsumowanie-problemu)
   1.1 [Problem z metrykami 24h](#11-problem-z-metrykami-24h)
2. [Cel techniczny](#2-cel-techniczny)
3. [Zakres plików do zmiany](#3-zakres-plików-do-zmiany)
4. [Plan wdrożenia – 4 etapy](#4-plan-wdrożenia--4-etapy)
5. [Szczegółowe zmiany per plik](#5-szczegółowe-zmiany-per-plik)
6. [Proponowana kolejność commitów](#6-proponowana-kolejność-commitów)
7. [Testowanie po każdym etapie](#7-testowanie-po-każdym-etapie)
8. [Co zrobić, jeśli nadal będą spike'i](#8-co-zrobić-jeśli-nadal-będą-spikei)
9. [Rekomendacja końcowa](#9-rekomendacja-końcowa)

---

## 1. Podsumowanie problemu

Backend wykonuje kosztowną pracę w skumulowanych burstach co 10 sekund:
- flush pending metrics do SQLite
- pełny przebieg agregacji przez wszystkie poziomy
- delete starych danych

To powoduje widoczne spike'y CPU i I/O.

## 1.1 Problem z metrykami 24h

**Problem:** Retencja metrics_30m (12h) jest krótsza niż zakres 24h, co powoduje, że metryki 24h pokazują tylko ostatnie 12 godzin.

**Rozwiązanie:** Zwiększenie retencji metrics_30m z 12h do 48h.

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

## 2. Cel techniczny

1. **Spłaszczenie obciążenia w czasie**
2. **Mniejsze transakcje**
3. **Rzadsze uruchamianie ciężkich poziomów agregacji**
4. **Oddzielenie cleanupu od agregacji**
5. **Wykorzystanie indeksów, żeby backend mniej skanował bazę**

## 3. Zakres plików do zmiany

### Na pewno:
- `backend/internal/buffer/inserter.go`
- `backend/internal/aggregation/aggregator.go`
- `backend/internal/aggregation/config.go`
- `backend/migrations/0000X_add_metrics_indexes.sql` — nowa migracja
- opcjonalnie: dokumentacja w `docs/Backend.md` i `docs/Backend-API.md`

### Prawdopodobnie bez zmian:
- agent
- frontend
- websockety
- modele API

---

## 4. Plan wdrożenia – 4 etapy

### ETAP 0 – Naprawa metryk 24h
**Cel:** Zapewnić, że metryki 24h są pełne (retencja metrics_30m 48h).

**Zmiana:** W `backend/internal/aggregation/config.go`:
- Zmiana `RetentionThreshold` dla `metrics_30m` z 12h na 48h (linie 53-57).

**Dlaczego:** Obecna retencja 12h jest krótsza niż zakres 24h, co powoduje brak danych.

**Ryzyko:** Bardzo niskie.

### ETAP 1 – Wygładzenie flush do DB
**Cel:** Zmniejszyć rozmiar burstu co 10s przez częstszy, lżejszy flush.

**Zmiana:** W `BulkInserter.Run()`:
- ticker z `10s` na **2s**
- ewentualny mały startowy jitter 0–300ms

**Dlaczego:** Zamiast jednego dużego zapisu co 10s: 5 mniejszych zapisów, mniejsze zużycie CPU "na raz", mniejsze I/O burst.

**Ryzyko:** Niskie.

### ETAP 2 – Rozdzielenie harmonogramu agregacji od flush
**Cel:** Przestać uruchamiać agregację równocześnie z insertem.

**Zmiana:** W `Aggregator.Run()`:
- offset startowy 5s
- główny ticker agregacji z `10s` na **30s**

**Dlaczego:** Agregacja nie musi odpalać co 10s. Jej koszt jest duży, a wartość biznesowa z tak częstego odpalania mała.

**Ryzyko:** Niskie do średniego.

### ETAP 3 – Rozdzielenie agregacji od cleanupu
**Cel:** Usunąć kosztowne `DELETE` z gorącej ścieżki agregacji.

**Zmiana architektury:**
- `ProcessAggregation()` – tylko agregacja + insert
- `ProcessCleanup()` – tylko delete old data
- osobny ticker cleanupu: **10 minut**

**Dlaczego:** Najcięższa operacja kasowania nie będzie już doklejona do każdej agregacji.

**Ryzyko:** Niskie.

### ETAP 4 – Dodać indeksy pod metryki
**Cel:** Zmniejszyć koszt SELECT i DELETE na tabelach metryk.

**Zmiana:** Nowa migracja z indeksami na:
- `timestamp`
- `agent_uuid, timestamp`

**Dlaczego:** Przyspieszy zapytania, zmniejszy skanowanie.

**Ryzyko:** Niskie funkcjonalnie, średnie operacyjnie podczas tworzenia.

---

## 5. Szczegółowe zmiany per plik

### 1. `backend/internal/buffer/inserter.go`

**Zmiana A:** W `Run()`:
- `ticker := time.NewTicker(2 * time.Second)`

**Zmiana B (opcjonalna):** Dodać niewielki start delay (250ms) by startup nie był zsynchronizowany.

**Zmiana C (opcjonalna):** Dodać log/debug metryk czasu flush.

### 2. `backend/internal/aggregation/aggregator.go`

**Zmiana A:** Podzielić `Run()` na:
- `RunAggregation()`
- `RunCleanup()`

**Zmiana B:** `RunAggregation()`:
- offset startowy `5s`
- ticker `30s`

**Zmiana C:** `RunCleanup()`:
- ticker `10m`

**Zmiana D:** `ProcessAggregation()`:
- usunąć delete old data
- zostawić tylko fetch → aggregate → insert

**Zmiana E:** Dodać `ProcessCleanup()`:
- iteracja po poziomach
- delete wg `RetentionThreshold`
- cleanup `metrics_12h`

**Zmiana F (opcjonalna):** Dodać pomiary czasu.

### 3. `backend/internal/aggregation/config.go`

**Zmiana A (retencja):** Zmiana `RetentionThreshold` dla `metrics_30m` z 12h na 48h (linie 53-57).

**Wersja minimum:** Bez innych zmian.

**Wersja lepsza:** Dodać pole `RunEvery` (ale jeszcze nie używać) – przygotowanie pod przyszły etap.

### 4. `backend/migrations/0000X_add_metrics_indexes.sql`

**Zmiana:** Nowa migracja z indeksami (wzór w planie użytkownika).

### 5. Dokumentacja

Po wdrożeniu zaktualizować `docs/Backend.md` i `docs/Backend-API.md` o zmiany w harmonogramach.

---

## 6. Proponowana kolejność commitów

1. **`backend: increase retention of metrics_30m to 48h`**
   - tylko `backend/internal/aggregation/config.go`

2. **`backend: reduce write burst by flushing metrics every 2s`**
   - tylko `inserter.go`

3. **`backend: decouple aggregation from flush and run aggregation every 30s`**
   - `aggregator.go`

4. **`backend: move retention cleanup to dedicated 10m scheduler`**
   - `aggregator.go`

5. **`backend: add SQLite indexes for metrics timestamp queries`**
   - nowa migracja

6. (Opcjonalnie) **`backend: add per-level aggregation scheduling`**
   - `config.go`, `aggregator.go`

---

## 7. Testowanie po każdym etapie

### Po ETAPIE 0
- Sprawdzić, czy metryki 24h są pełne (powinny mieć dane z ostatnich 48h).
- Sprawdzić, czy rozmiar bazy danych nie wzrósł znacząco.

### Po ETAPIE 1
- Sprawdzić, czy zniknął duży pik co 10s.
- Czy nie pojawiły się mniejsze, ale akceptowalne piki co 2s.
- Czy CPU max spadło.

### Po ETAPIE 2
- Sprawdzić, czy nadal widać korelację pików.
- Czy wykres CPU jest wyraźnie gładszy.

### Po ETAPIE 3
- Sprawdzić, czy przestały się pojawiać okresowe I/O spike związane z delete.
- Czy cleanup nie generuje jednorazowego dużego piku co 10 min.

### Po ETAPIE 4
- Sprawdzić czas odpowiedzi API historii.
- Sprawdzić czas agregacji, iowait, zużycie CPU.

---

## 8. Co zrobić, jeśli nadal będą spike'i

Jeśli po tych zmianach nadal będą spike'i, kolejne kroki:

**A. Batchowany cleanup:**
- Zamiast `DELETE FROM table WHERE timestamp < ?` robić partiami z `LIMIT N`.

**B. Harmonogram per poziom agregacji:**
- Nie dotykać wszystkich tabel na każdym przebiegu.

**C. Agregacja inkrementalna:**
- Pamiętać ostatnio przetworzony watermark/max timestamp per level.

---

## 9. Rekomendacja końcowa

### Faza 1 – koniecznie
1. Retencja metrics_30m 48h (naprawa metryk 24h)
2. `BulkInserter` 2s
3. `Aggregator` 30s + offset 5s
4. Cleanup osobno co 10m
5. Indeksy metryk

### Faza 2 – jeśli nadal coś widać
5. Batchowany cleanup
6. Harmonogram per poziom agregacji

### Faza 3 – tylko jeśli dalej problem istnieje
7. Inkrementalna agregacja zamiast pełnych przebiegów

---

**Data ostatniej aktualizacji:** 2026-03-10  
**Wersja dokumentu:** 2.1 (dodano retencję metrics_30m 48h)