package model

import (
	"testing"
	"time"
)

func TestComputeRateOfChange(t *testing.T) {
	base := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	entries := []Nightscoutdb{
		{Sgv: 100, Trend: 4, Ns_datetime: base},                       // flat
		{Sgv: 110, Trend: 2, Ns_datetime: base.Add(5 * time.Minute)},  // +10 over 5min = +2 mg/dl/min (rapid rise)
		{Sgv: 112, Trend: 4, Ns_datetime: base.Add(10 * time.Minute)}, // +2 over 5min = +0.4 mg/dl/min
		{Sgv: 90, Trend: 6, Ns_datetime: base.Add(15 * time.Minute)},  // -22 over 5min = -4.4 mg/dl/min (rapid fall)
		{Sgv: 88, Trend: 4, Ns_datetime: base.Add(20 * time.Minute)},  // -2 over 5min = -0.4 mg/dl/min
	}

	r := ComputeRateOfChange(entries, base, base.Add(20*time.Minute))

	if r.Count != 5 {
		t.Errorf("Count = %d, want 5", r.Count)
	}
	if r.RocSamples != 4 {
		t.Errorf("RocSamples = %d, want 4", r.RocSamples)
	}

	trendCounts := make(map[int]TrendCount, len(r.TrendCounts))
	for _, tc := range r.TrendCounts {
		trendCounts[tc.Trend] = tc
	}
	if trendCounts[4].Count != 3 {
		t.Errorf("trend 4 (flat) count = %d, want 3", trendCounts[4].Count)
	}
	if trendCounts[2].Count != 1 {
		t.Errorf("trend 2 (SingleUp) count = %d, want 1", trendCounts[2].Count)
	}
	if trendCounts[6].Count != 1 {
		t.Errorf("trend 6 (SingleDown) count = %d, want 1", trendCounts[6].Count)
	}

	if r.MaxRoc != 2.0 {
		t.Errorf("MaxRoc = %v, want 2.0", r.MaxRoc)
	}
	if r.MinRoc != -4.4 {
		t.Errorf("MinRoc = %v, want -4.4", r.MinRoc)
	}
	// One rapid rise (+2 mg/dl/min, at the >= 2.0 threshold) and one rapid fall (-4.4 mg/dl/min).
	if r.RapidRiseEpisodes != 1 {
		t.Errorf("RapidRiseEpisodes = %d, want 1", r.RapidRiseEpisodes)
	}
	if r.RapidFallEpisodes != 1 {
		t.Errorf("RapidFallEpisodes = %d, want 1", r.RapidFallEpisodes)
	}
}

func TestComputeRateOfChangeEmpty(t *testing.T) {
	from := time.Now()
	to := from.Add(time.Hour)
	r := ComputeRateOfChange(nil, from, to)
	if r.Count != 0 {
		t.Errorf("Count = %d, want 0", r.Count)
	}
}

func TestComputeRateOfChangeSingleEntry(t *testing.T) {
	base := time.Now()
	entries := []Nightscoutdb{{Sgv: 100, Trend: 4, Ns_datetime: base}}
	r := ComputeRateOfChange(entries, base, base)
	if r.Count != 1 {
		t.Errorf("Count = %d, want 1", r.Count)
	}
	if r.RocSamples != 0 {
		t.Errorf("RocSamples = %d, want 0 (need at least 2 entries)", r.RocSamples)
	}
}

func TestComputeRateOfChangeSkipsLargeGaps(t *testing.T) {
	base := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	entries := []Nightscoutdb{
		{Sgv: 100, Trend: 4, Ns_datetime: base},
		// A 30-minute gap (e.g. sensor dropout) should be skipped, not
		// treated as a real ~0.3 mg/dl/min rate over a huge span.
		{Sgv: 190, Trend: 2, Ns_datetime: base.Add(30 * time.Minute)},
		{Sgv: 192, Trend: 4, Ns_datetime: base.Add(35 * time.Minute)},
	}
	r := ComputeRateOfChange(entries, base, base.Add(35*time.Minute))
	if r.RocSamples != 1 {
		t.Errorf("RocSamples = %d, want 1 (the 30-min gap pair should be skipped)", r.RocSamples)
	}
}
