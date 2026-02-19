package buffer

import (
	"sync"
)

// maxPendingPerContainer caps the retry queue for a single agent/container pair.
// If the database is unavailable for longer than this many seconds, the oldest
// unaggregated points are dropped to prevent unbounded memory growth.
const maxPendingPerContainer = 1200 // 20 minutes of 1s resolution

type BufferManager struct {
	Buffers     map[string]map[string]*RingBuffer
	HostBuffers map[string]*HostRingBuffer
	pending     map[string]map[string][]MetricPoint
	pendingHost map[string][]HostMetricPoint
	mu          sync.RWMutex
}

func NewBufferManager() *BufferManager {
	return &BufferManager{
		Buffers:     make(map[string]map[string]*RingBuffer),
		HostBuffers: make(map[string]*HostRingBuffer),
		pending:     make(map[string]map[string][]MetricPoint),
		pendingHost: make(map[string][]HostMetricPoint),
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
	rb := bm.GetOrCreate(agentUUID, containerID)
	evicted := rb.Add(point)
	if evicted == nil {
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, ok := bm.pending[agentUUID]; !ok {
		bm.pending[agentUUID] = make(map[string][]MetricPoint)
	}
	bm.pending[agentUUID][containerID] = append(bm.pending[agentUUID][containerID], *evicted)
}

func (bm *BufferManager) AddHostMetric(agentUUID string, point HostMetricPoint) {
	bm.mu.Lock()
	rb, ok := bm.HostBuffers[agentUUID]
	if !ok {
		rb = NewHostRingBuffer(60)
		bm.HostBuffers[agentUUID] = rb
	}
	bm.mu.Unlock()

	evicted := rb.Add(point)
	if evicted == nil {
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.pendingHost[agentUUID] = append(bm.pendingHost[agentUUID], *evicted)
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

func (bm *BufferManager) GetContainerPoints(agentUUID, containerID string) []MetricPoint {
	bm.mu.RLock()
	containers, ok := bm.Buffers[agentUUID]
	if !ok {
		bm.mu.RUnlock()
		return nil
	}

	rb, ok := containers[containerID]
	bm.mu.RUnlock()
	if !ok {
		return nil
	}

	return rb.GetAll()
}

func (bm *BufferManager) GetHostPoints(agentUUID string) []HostMetricPoint {
	bm.mu.RLock()
	rb, ok := bm.HostBuffers[agentUUID]
	bm.mu.RUnlock()
	if !ok {
		return nil
	}

	return rb.GetAll()
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

// RemoveAgentBuffers removes only the live ring buffers (Buffers and HostBuffers) for a given agent.
// Call this when an agent disconnects so stale ring-buffer allocations are freed.
// The pending queues are NOT affected - they will be flushed to DB by the BulkInserter.
// The 60-second live cache is NOT affected for other agents; when this agent reconnects,
// GetOrCreate will allocate a fresh buffer.
func (bm *BufferManager) RemoveAgentBuffers(agentUUID string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	delete(bm.Buffers, agentUUID)
	delete(bm.HostBuffers, agentUUID)
	// NOTE: pending and pendingHost are intentionally NOT deleted here.
	// They will be flushed to DB by the BulkInserter on its next cycle (up to 10s delay).
	// This prevents data loss during agent disconnect/reconnect scenarios.
}

func (bm *BufferManager) DrainPendingMetrics() map[string]map[string][]MetricPoint {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	snapshot := make(map[string]map[string][]MetricPoint, len(bm.pending))
	for agentID, containers := range bm.pending {
		snapshot[agentID] = make(map[string][]MetricPoint, len(containers))
		for containerID, points := range containers {
			copied := make([]MetricPoint, len(points))
			copy(copied, points)
			snapshot[agentID][containerID] = copied
		}
	}

	bm.pending = make(map[string]map[string][]MetricPoint)
	return snapshot
}

func (bm *BufferManager) RequeuePendingMetrics(failed map[string]map[string][]MetricPoint) {
	if len(failed) == 0 {
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	for agentID, containers := range failed {
		if _, ok := bm.pending[agentID]; !ok {
			bm.pending[agentID] = make(map[string][]MetricPoint)
		}

		for containerID, points := range containers {
			if len(points) == 0 {
				continue
			}

			existing := bm.pending[agentID][containerID]
			requeued := make([]MetricPoint, 0, len(points)+len(existing))
			requeued = append(requeued, points...)
			requeued = append(requeued, existing...)
			// Cap to avoid unbounded growth when DB is unavailable.
			if len(requeued) > maxPendingPerContainer {
				requeued = requeued[len(requeued)-maxPendingPerContainer:]
			}
			bm.pending[agentID][containerID] = requeued
		}
	}
}

func (bm *BufferManager) DrainPendingHostMetrics() map[string][]HostMetricPoint {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	snapshot := make(map[string][]HostMetricPoint, len(bm.pendingHost))
	for agentID, points := range bm.pendingHost {
		copied := make([]HostMetricPoint, len(points))
		copy(copied, points)
		snapshot[agentID] = copied
	}

	bm.pendingHost = make(map[string][]HostMetricPoint)
	return snapshot
}

func (bm *BufferManager) RequeuePendingHostMetrics(failed map[string][]HostMetricPoint) {
	if len(failed) == 0 {
		return
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	for agentID, points := range failed {
		if len(points) == 0 {
			continue
		}

		existing := bm.pendingHost[agentID]
		requeued := make([]HostMetricPoint, 0, len(points)+len(existing))
		requeued = append(requeued, points...)
		requeued = append(requeued, existing...)
		// Cap to avoid unbounded growth when DB is unavailable.
		if len(requeued) > maxPendingPerContainer {
			requeued = requeued[len(requeued)-maxPendingPerContainer:]
		}
		bm.pendingHost[agentID] = requeued
	}
}
