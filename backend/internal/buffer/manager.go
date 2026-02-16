package buffer

import (
	"sync"
)

type BufferManager struct {
	Buffers     map[string]map[string]*RingBuffer
	HostBuffers map[string]*HostRingBuffer
	mu          sync.RWMutex
}

func NewBufferManager() *BufferManager {
	return &BufferManager{
		Buffers:     make(map[string]map[string]*RingBuffer),
		HostBuffers: make(map[string]*HostRingBuffer),
	}
}

func (bm *BufferManager) GetOrCreate(agentUUID, containerID string) *RingBuffer {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, ok := bm.Buffers[agentUUID]; !ok {
		bm.Buffers[agentUUID] = make(map[string]*RingBuffer)
	}

	if _, ok := bm.Buffers[agentUUID][containerID]; !ok {
		// Capacity 60 points (1 minute of 1s resolution)
		bm.Buffers[agentUUID][containerID] = NewRingBuffer(60)
	}

	return bm.Buffers[agentUUID][containerID]
}

func (bm *BufferManager) AddMetric(agentUUID, containerID string, point MetricPoint) {
	// GetOrCreate handles lock internally, but we might want to optimize?
	// It's fine for now.
	rb := bm.GetOrCreate(agentUUID, containerID)
	rb.Add(point)
}

func (bm *BufferManager) AddHostMetric(agentUUID string, point HostMetricPoint) {
	bm.mu.Lock()
	rb, ok := bm.HostBuffers[agentUUID]
	if !ok {
		rb = NewHostRingBuffer(60)
		bm.HostBuffers[agentUUID] = rb
	}
	bm.mu.Unlock()
	rb.Add(point)
}

func (bm *BufferManager) GetLatestForServer(agentUUID string) map[string]MetricPoint {
	bm.mu.RLock()
	containers, ok := bm.Buffers[agentUUID]
	bm.mu.RUnlock()

	if !ok {
		return nil
	}

	latest := make(map[string]MetricPoint)
	for containerID, rb := range containers {
		points := rb.GetAll()
		if len(points) > 0 {
			latest[containerID] = points[len(points)-1]
		}
	}
	return latest
}

func (bm *BufferManager) GetLatestForServerAtTimestamp(agentUUID string, timestamp int64) map[string]MetricPoint {
	bm.mu.RLock()
	containers, ok := bm.Buffers[agentUUID]
	bm.mu.RUnlock()

	if !ok {
		return nil
	}

	latest := make(map[string]MetricPoint)
	for containerID, rb := range containers {
		points := rb.GetAll()
		if len(points) == 0 {
			continue
		}
		last := points[len(points)-1]
		if last.Timestamp == timestamp {
			latest[containerID] = last
		}
	}
	return latest
}

func (bm *BufferManager) GetLatestHostForServer(agentUUID string) *HostMetricPoint {
	bm.mu.RLock()
	rb, ok := bm.HostBuffers[agentUUID]
	bm.mu.RUnlock()
	if !ok {
		return nil
	}

	points := rb.GetAll()
	if len(points) == 0 {
		return nil
	}
	p := points[len(points)-1]
	return &p
}

func (bm *BufferManager) GetLatestForContainer(agentUUID, containerID string) *MetricPoint {
	bm.mu.RLock()
	containers, ok := bm.Buffers[agentUUID]
	bm.mu.RUnlock()

	if !ok {
		return nil
	}

	rb, ok := containers[containerID]
	if !ok {
		return nil
	}

	points := rb.GetAll()
	if len(points) > 0 {
		p := points[len(points)-1]
		return &p
	}
	return nil
}

// GetAllBuffers returns a snapshot of the buffer map structure.
// Note: It returns the POINTERS to RingBuffers. Thread safety of accessing those buffers is handled by RingBuffer methods.
// But the map structure itself is protected.
func (bm *BufferManager) GetAllBuffers() map[string]map[string]*RingBuffer {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	// Deep copy of the map structure, but pointers to buffers are shared
	snapshot := make(map[string]map[string]*RingBuffer)
	for agentID, containers := range bm.Buffers {
		snapshot[agentID] = make(map[string]*RingBuffer)
		for containerID, buf := range containers {
			snapshot[agentID][containerID] = buf
		}
	}
	return snapshot
}

func (bm *BufferManager) GetAllHostBuffers() map[string]*HostRingBuffer {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	snapshot := make(map[string]*HostRingBuffer)
	for agentID, buf := range bm.HostBuffers {
		snapshot[agentID] = buf
	}
	return snapshot
}
