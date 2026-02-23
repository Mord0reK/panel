package metrics

import (
	"context"
	"testing"
	"time"

	"agent/internal/collector"
)

func TestSnapshotCollectorCPUDelta(t *testing.T) {
	ctx := context.Background()

	snapshotCollector := NewSnapshotCollector()

	first, err := snapshotCollector.Collect(ctx, nil)
	if err != nil {
		t.Fatalf("First Collect failed: %v", err)
	}

	t.Logf("First CPU: %.2f%%", first.Host.CPU)

	time.Sleep(200 * time.Millisecond)

	second, err := snapshotCollector.Collect(ctx, nil)
	if err != nil {
		t.Fatalf("Second Collect failed: %v", err)
	}

	t.Logf("Second CPU: %.2f%%", second.Host.CPU)

	if second.Host.CPU <= 0 {
		t.Errorf("Expected CPU > 0, got %.2f", second.Host.CPU)
	}
}

func TestCollectorCPUWithFallback(t *testing.T) {
	ctx := context.Background()

	metrics, err := collector.CollectSystemMetrics(ctx)
	if err != nil {
		t.Fatalf("CollectSystemMetrics failed: %v", err)
	}

	t.Logf("CPU Percent from cpu.Percent(100ms): %.2f%%", metrics.CPU.Percent)
	t.Logf("CPU Times: User=%.2f, System=%.2f, Idle=%.2f",
		metrics.CPU.User, metrics.CPU.System, metrics.CPU.Idle)

	if metrics.CPU.Percent < 0 || metrics.CPU.Percent > 100 {
		t.Errorf("CPU percent out of range: %.2f", metrics.CPU.Percent)
	}
}
