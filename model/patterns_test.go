package model

import (
	"testing"
	"time"
)

func TestComputeHourlyPatterns(t *testing.T) {
	day1 := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	entries := []Nightscoutdb{
		{Sgv: 100, Ns_datetime: day1.Add(7 * time.Hour)},
		{Sgv: 140, Ns_datetime: day2.Add(7 * time.Hour)},
		{Sgv: 90, Ns_datetime: day1.Add(13 * time.Hour)},
	}

	patterns := ComputeHourlyPatterns(entries)

	if len(patterns) != 2 {
		t.Fatalf("len(patterns) = %d, want 2 (only hours 7 and 13 have readings)", len(patterns))
	}

	byHour := make(map[int]HourlyPattern, len(patterns))
	for _, p := range patterns {
		byHour[p.Hour] = p
	}

	hour7, ok := byHour[7]
	if !ok {
		t.Fatalf("expected a bucket for hour 7")
	}
	if hour7.Count != 2 {
		t.Errorf("hour7.Count = %d, want 2", hour7.Count)
	}
	if hour7.AverageSgv != 120 {
		t.Errorf("hour7.AverageSgv = %v, want 120", hour7.AverageSgv)
	}
	if hour7.Min != 100 || hour7.Max != 140 {
		t.Errorf("hour7.Min/Max = %d/%d, want 100/140", hour7.Min, hour7.Max)
	}

	hour13, ok := byHour[13]
	if !ok {
		t.Fatalf("expected a bucket for hour 13")
	}
	if hour13.Count != 1 {
		t.Errorf("hour13.Count = %d, want 1", hour13.Count)
	}
	if hour13.Median != 90 {
		t.Errorf("hour13.Median = %v, want 90", hour13.Median)
	}

	if _, ok := byHour[0]; ok {
		t.Errorf("hour 0 has no readings and should be omitted")
	}
}

func TestComputeHourlyPatternsEmpty(t *testing.T) {
	patterns := ComputeHourlyPatterns(nil)
	if len(patterns) != 0 {
		t.Errorf("len(patterns) = %d, want 0", len(patterns))
	}
}

func TestComputeHourlyPatternsOrderedByHour(t *testing.T) {
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	// Insert entries with hours out of order to verify output is ascending by hour.
	entries := []Nightscoutdb{
		{Sgv: 100, Ns_datetime: base.Add(20 * time.Hour)},
		{Sgv: 100, Ns_datetime: base.Add(2 * time.Hour)},
		{Sgv: 100, Ns_datetime: base.Add(10 * time.Hour)},
	}

	patterns := ComputeHourlyPatterns(entries)
	if len(patterns) != 3 {
		t.Fatalf("len(patterns) = %d, want 3", len(patterns))
	}
	if patterns[0].Hour != 2 || patterns[1].Hour != 10 || patterns[2].Hour != 20 {
		t.Errorf("hours = %d, %d, %d, want ascending 2, 10, 20", patterns[0].Hour, patterns[1].Hour, patterns[2].Hour)
	}
}
