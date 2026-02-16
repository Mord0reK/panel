package buffer

import (
	"sync"
)

type MetricPoint struct {
	Timestamp   int64
	CPU         float64
	MemUsed     uint64
	MemPercent  float64
	DiskUsed    uint64
	DiskPercent float64
	NetRx       uint64
	NetTx       uint64
}

type RingBuffer struct {
	Size     int
	Data     []MetricPoint
	WritePos int
	Count    int // How many items are actually in buffer
	mu       sync.Mutex
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		Size: size,
		Data: make([]MetricPoint, size),
	}
}

func (rb *RingBuffer) Add(point MetricPoint) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.Data[rb.WritePos] = point
	rb.WritePos = (rb.WritePos + 1) % rb.Size
	if rb.Count < rb.Size {
		rb.Count++
	}
}

// GetAll returns a copy of all valid points in the buffer
func (rb *RingBuffer) GetAll() []MetricPoint {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	result := make([]MetricPoint, 0, rb.Count)
	
	if rb.Count < rb.Size {
		// Buffer not full, points are from 0 to Count-1
		for i := 0; i < rb.Count; i++ {
			result = append(result, rb.Data[i])
		}
	} else {
		// Buffer full, start from WritePos (oldest) to end, then 0 to WritePos-1
		idx := rb.WritePos
		for i := 0; i < rb.Size; i++ {
			result = append(result, rb.Data[idx])
			idx = (idx + 1) % rb.Size
		}
	}
	return result
}

func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.WritePos = 0
	rb.Count = 0
}

// ClearOldest clears n oldest points (simulate "consuming" them if we were using a queue, but here we just want to reset if we bulk inserted everything)
// Actually, plan says: "Clear buffer (zachowaj ostatnie 60s)".
// My GetAll returns everything.
// If I bulk insert everything, I should clear everything?
// Plan says: "Clear buffer (zachowaj ostatnie 60s)". This suggests the buffer might hold MORE than what we insert?
// Or maybe we insert everything and then clear.
// If the buffer size is 60 (1 minute), and we insert every 10s, we have 10 points.
// If we insert them, we might want to keep them for SSE/Live view?
// Ah, ETAP 9: SSE "Pobierz dane z RAM bufferów (ostatnie punkty)".
// So we should NOT clear points that are needed for live view?
// But BulkInserter says "Clear buffer (zachowaj ostatnie 60s)".
// If buffer size IS 60, then "keeping last 60s" means keeping everything?
// No, maybe we have a buffer of 120s?
// Plan says: "Metoda GetOrCreate... (60 punktów capacity)".
// If capacity is 60, and we insert every 10s, we insert 10 points.
// If we clear, we lose them from RAM.
// If we want to support Live view from RAM, we must keep them.
// But we also want to avoid inserting duplicates to DB?
// Usually, we track "last inserted timestamp" or we just insert what's new.
// But simpler approach: Buffer acts as a cache.
// BulkInserter reads ALL points, filters those NOT yet in DB? No, that's expensive.
// Better: RingBuffer holds recent data for Live View.
// BulkInserter needs to know what to insert.
// Maybe a separate queue for DB insertion? Or `LastInsertedTimestamp`?
// The plan says: "Clear buffer (zachowaj ostatnie 60s)".
// This is slightly ambiguous.
// Interpretation: Buffer holds e.g. 60 points.
// BulkInserter runs every 10s. It takes points that are NEW?
// If I clear the buffer, Live View loses data.
// If I don't clear, BulkInserter inserts duplicates?
// Let's look at `BulkInserter` logic:
// "Jeśli punktów >= 10: Przygotuj bulk INSERT... Clear buffer (zachowaj ostatnie 60s)".
// This implies the buffer might grow larger than 60? Or maybe we keep 60s in RAM for live view, but we only insert "new" ones.
// But if we clear, we lose data.
// Revised plan interpretation:
// The buffer is primarily for batching inserts AND live view.
// If we insert into DB, we verify we don't insert duplicates (PRIMARY KEY constraint).
// But we want to avoid DB errors.
// Maybe `RingBuffer` is just a circular buffer of size 60.
// We add points.
// BulkInserter grabs ALL points?
// If we grab all 60 points and try to insert them every 10s, we insert 50 duplicates.
// We should track `LastFlushedTimestamp`.
// `GetAllUnflushed(lastFlushed int64)`?
// Or: `GetNewPoints()` which returns points > last check.
// I will implement `GetPointsSince(timestamp int64) []MetricPoint`.
// And `BulkInserter` will keep track of last inserted timestamp per buffer?
// Or `RingBuffer` keeps track of `LastFlushed`.
// Let's add `LastFlushedTime int64` to `RingBuffer`.
// And `GetNewPoints()` updates it? No, `Get` should be idempotent. `MarkFlushed(ts)`?
// Plan says "Clear buffer (zachowaj ostatnie 60s)".
// Maybe the buffer is NOT a ring buffer in the strict sense for DB, but a list?
// "Struct RingBuffer (Size int...)" implies fixed size.
// If I use `RingBuffer` for live view (last 60s), I should NEVER clear it fully.
// The `BulkInserter` should just read "new" items.
// I will add `GetPointsSince(ts int64)` to `RingBuffer`.
// And `BulkInserter` will store the last timestamp it successfully inserted for that buffer (in memory map).

func (rb *RingBuffer) GetPointsSince(ts int64) []MetricPoint {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	// Iterate and find points > ts
	// Since it's a ring buffer, order is tricky.
	// But `GetAll()` returns sorted by time (if added in order).
	// Let's reuse `GetAll` logic inside.
	
	all := rb.getAllUnsafe()
	var newPoints []MetricPoint
	for _, p := range all {
		if p.Timestamp > ts {
			newPoints = append(newPoints, p)
		}
	}
	return newPoints
}

func (rb *RingBuffer) getAllUnsafe() []MetricPoint {
	result := make([]MetricPoint, 0, rb.Count)
	if rb.Count < rb.Size {
		for i := 0; i < rb.Count; i++ {
			result = append(result, rb.Data[i])
		}
	} else {
		idx := rb.WritePos
		for i := 0; i < rb.Size; i++ {
			result = append(result, rb.Data[idx])
			idx = (idx + 1) % rb.Size
		}
	}
	return result
}
