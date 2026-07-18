package model

import (
	"testing"
	"time"
)

func TestComputeStats(t *testing.T) {
	base := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	entries := []Nightscoutdb{
		{Sgv: 100, Ns_datetime: base},
		{Sgv: 60, Ns_datetime: base.Add(5 * time.Minute)},
		{Sgv: 65, Ns_datetime: base.Add(10 * time.Minute)},
		{Sgv: 190, Ns_datetime: base.Add(60 * time.Minute)},
		{Sgv: 120, Ns_datetime: base.Add(65 * time.Minute)},
	}

	stats := ComputeStats(entries, base, base.Add(70*time.Minute))

	if stats.Count != 5 {
		t.Errorf("Count = %d, want 5", stats.Count)
	}
	if stats.MinSgv != 60 {
		t.Errorf("MinSgv = %d, want 60", stats.MinSgv)
	}
	if stats.MaxSgv != 190 {
		t.Errorf("MaxSgv = %d, want 190", stats.MaxSgv)
	}
	// Two consecutive low readings (60, 65) within the episode gap count as one episode.
	if stats.LowEpisodes != 1 {
		t.Errorf("LowEpisodes = %d, want 1", stats.LowEpisodes)
	}
	if stats.HighEpisodes != 1 {
		t.Errorf("HighEpisodes = %d, want 1", stats.HighEpisodes)
	}
	wantInRange := 100.0 * 2 / 5
	if stats.TimeInRangePct != wantInRange {
		t.Errorf("TimeInRangePct = %v, want %v", stats.TimeInRangePct, wantInRange)
	}
	// average = (100+60+65+190+120)/5 = 107; GMI = 3.31 + 0.02392*107
	wantGmi := 3.31 + 0.02392*107.0
	if stats.Gmi != wantGmi {
		t.Errorf("Gmi = %v, want %v", stats.Gmi, wantGmi)
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	from := time.Now()
	to := from.Add(time.Hour)
	stats := ComputeStats(nil, from, to)
	if stats.Count != 0 {
		t.Errorf("Count = %d, want 0", stats.Count)
	}
}
