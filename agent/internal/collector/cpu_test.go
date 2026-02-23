package collector

import (
	"context"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
)

func TestCollectCPUTimes(t *testing.T) {
	ctx := context.Background()

	times, err := cpu.TimesWithContext(ctx, false)
	if err != nil {
		t.Fatalf("cpu.TimesWithContext failed: %v", err)
	}

	if len(times) == 0 {
		t.Fatal("no CPU times returned")
	}

	t.Logf("CPU Times: %+v", times[0])
}

func TestCPUDeltaCalculation(t *testing.T) {
	ctx := context.Background()

	times1, err := cpu.TimesWithContext(ctx, false)
	if err != nil {
		t.Fatalf("first cpu.TimesWithContext failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	times2, err := cpu.TimesWithContext(ctx, false)
	if err != nil {
		t.Fatalf("second cpu.TimesWithContext failed: %v", err)
	}

	totalDelta := (times2[0].User - times1[0].User) +
		(times2[0].System - times1[0].System) +
		(times2[0].Idle - times1[0].Idle)

	if totalDelta <= 0 {
		t.Fatal("totalDelta should be positive")
	}

	cpuPercent := ((times2[0].User - times1[0].User) + (times2[0].System - times1[0].System)) / totalDelta * 100

	t.Logf("CPU Percent (delta): %.2f%%", cpuPercent)

	if cpuPercent < 0 || cpuPercent > 100 {
		t.Errorf("CPU percent out of range: %.2f", cpuPercent)
	}
}

func TestCPUPercentNonBlocking(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()

	times, err := cpu.TimesWithContext(ctx, false)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("cpu.TimesWithContext failed: %v", err)
	}

	if elapsed > 500*time.Millisecond {
		t.Errorf("cpu.TimesWithContext took too long: %v", elapsed)
	}

	if len(times) == 0 {
		t.Fatal("no CPU times returned")
	}

	t.Logf("cpu.TimesWithContext completed in %v", elapsed)
}
