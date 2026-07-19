package model

import (
	"testing"
	"time"
)

func TestComputeDataQuality(t *testing.T) {
	base := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	from := base
	to := base.Add(30 * time.Minute)

	// Readings intentionally out of order to verify sorting happens
	// independently of entry order. Normal 5-min cadence except one
	// 20-minute gap between 00:05 and 00:25.
	entries := []Nightscoutdb{
		{Sgv: 100, Ns_datetime: base.Add(25 * time.Minute)},
		{Sgv: 100, Ns_datetime: base},
		{Sgv: 100, Ns_datetime: base.Add(5 * time.Minute)},
		{Sgv: 100, Ns_datetime: base.Add(30 * time.Minute)},
	}

	dq := ComputeDataQuality(entries, from, to)

	if dq.Count != 4 {
		t.Errorf("Count = %d, want 4", dq.Count)
	}
	// ExpectedCount = 30min / 5min = 6
	if dq.ExpectedCount != 6 {
		t.Errorf("ExpectedCount = %d, want 6", dq.ExpectedCount)
	}
	if len(dq.Gaps) != 1 {
		t.Fatalf("len(Gaps) = %d, want 1", len(dq.Gaps))
	}
	gap := dq.Gaps[0]
	if !gap.From.Equal(base.Add(5*time.Minute)) || !gap.To.Equal(base.Add(25*time.Minute)) {
		t.Errorf("gap = %v -> %v, want %v -> %v", gap.From, gap.To, base.Add(5*time.Minute), base.Add(25*time.Minute))
	}
	if gap.Duration != 20*time.Minute {
		t.Errorf("gap.Duration = %v, want 20m", gap.Duration)
	}
	if dq.LargestGap.Duration != 20*time.Minute {
		t.Errorf("LargestGap.Duration = %v, want 20m", dq.LargestGap.Duration)
	}
}

func TestComputeDataQualityNoGaps(t *testing.T) {
	base := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	entries := []Nightscoutdb{
		{Sgv: 100, Ns_datetime: base},
		{Sgv: 100, Ns_datetime: base.Add(5 * time.Minute)},
		{Sgv: 100, Ns_datetime: base.Add(10 * time.Minute)},
	}
	dq := ComputeDataQuality(entries, base, base.Add(10*time.Minute))
	if len(dq.Gaps) != 0 {
		t.Errorf("len(Gaps) = %d, want 0", len(dq.Gaps))
	}
	if dq.LargestGap.Duration != 0 {
		t.Errorf("LargestGap.Duration = %v, want 0", dq.LargestGap.Duration)
	}
}

func TestComputeDataQualityCoveragePctCapsAt100(t *testing.T) {
	base := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	// More readings than the expected cadence would predict (e.g. a denser
	// polling interval); coverage should cap at 100, not exceed it.
	entries := make([]Nightscoutdb, 0, 20)
	for i := 0; i < 20; i++ {
		entries = append(entries, Nightscoutdb{Sgv: 100, Ns_datetime: base.Add(time.Duration(i) * time.Minute)})
	}
	dq := ComputeDataQuality(entries, base, base.Add(10*time.Minute))
	if dq.CoveragePct != 100 {
		t.Errorf("CoveragePct = %v, want 100", dq.CoveragePct)
	}
}

func TestComputeDataQualityEmpty(t *testing.T) {
	from := time.Now()
	to := from.Add(time.Hour)
	dq := ComputeDataQuality(nil, from, to)
	if dq.Count != 0 {
		t.Errorf("Count = %d, want 0", dq.Count)
	}
	if dq.CoveragePct != 0 {
		t.Errorf("CoveragePct = %v, want 0", dq.CoveragePct)
	}
	if len(dq.Gaps) != 0 {
		t.Errorf("len(Gaps) = %d, want 0", len(dq.Gaps))
	}
}

func TestComputeDataQualityMultipleGapsLargestGapPicksBiggest(t *testing.T) {
	base := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	entries := []Nightscoutdb{
		{Sgv: 100, Ns_datetime: base},
		{Sgv: 100, Ns_datetime: base.Add(15 * time.Minute)}, // 15-min gap
		{Sgv: 100, Ns_datetime: base.Add(20 * time.Minute)}, // back to normal cadence
		{Sgv: 100, Ns_datetime: base.Add(90 * time.Minute)}, // 70-min gap (largest)
	}
	dq := ComputeDataQuality(entries, base, base.Add(90*time.Minute))
	if len(dq.Gaps) != 2 {
		t.Fatalf("len(Gaps) = %d, want 2", len(dq.Gaps))
	}
	if dq.LargestGap.Duration != 70*time.Minute {
		t.Errorf("LargestGap.Duration = %v, want 70m", dq.LargestGap.Duration)
	}
}
