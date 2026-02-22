package buffer

import "sync"

type HostMetricPoint struct {
	Timestamp int64
	CPU       float64

	MemUsed    uint64
	MemPercent float64

	DiskReadBytesPerSec  uint64
	DiskWriteBytesPerSec uint64
	NetRxBytesPerSec     uint64
	NetTxBytesPerSec     uint64
	DiskUsedPercent      float64
}

type HostRingBuffer struct {
	Size     int
	Data     []HostMetricPoint
	WritePos int
	Count    int
	mu       sync.Mutex
}

func NewHostRingBuffer(size int) *HostRingBuffer {
	return &HostRingBuffer{
		Size: size,
		Data: make([]HostMetricPoint, size),
	}
}

func (rb *HostRingBuffer) Add(point HostMetricPoint) *HostMetricPoint {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	var evicted *HostMetricPoint
	if rb.Count == rb.Size {
		oldest := rb.Data[rb.WritePos]
		evicted = &oldest
	}

	rb.Data[rb.WritePos] = point
	rb.WritePos = (rb.WritePos + 1) % rb.Size
	if rb.Count < rb.Size {
		rb.Count++
	}

	return evicted
}

func (rb *HostRingBuffer) GetAll() []HostMetricPoint {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.getAllUnsafe()
}

func (rb *HostRingBuffer) GetPointsSince(ts int64) []HostMetricPoint {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	all := rb.getAllUnsafe()
	var newPoints []HostMetricPoint
	for _, p := range all {
		if p.Timestamp > ts {
			newPoints = append(newPoints, p)
		}
	}
	return newPoints
}

func (rb *HostRingBuffer) getAllUnsafe() []HostMetricPoint {
	result := make([]HostMetricPoint, 0, rb.Count)
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
