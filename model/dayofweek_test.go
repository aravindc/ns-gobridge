package model

import (
	"testing"
	"time"
)

func TestComputeDayOfWeekPatterns(t *testing.T) {
	// 2026-07-18 is a Saturday, 2026-07-19 a Sunday, 2026-07-20 a Monday.
	saturday := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	sunday := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	entries := []Nightscoutdb{
		{Sgv: 100, Ns_datetime: saturday},
		{Sgv: 140, Ns_datetime: saturday.Add(24 * time.Hour)}, // next Sunday
		{Sgv: 90, Ns_datetime: sunday.Add(-7 * 24 * time.Hour)},
	}

	patterns := ComputeDayOfWeekPatterns(entries)

	if len(patterns) != 2 {
		t.Fatalf("len(patterns) = %d, want 2 (only Saturday and Sunday have readings)", len(patterns))
	}

	byWeekday := make(map[time.Weekday]DayOfWeekPattern, len(patterns))
	for _, p := range patterns {
		byWeekday[p.Weekday] = p
	}

	sat, ok := byWeekday[time.Saturday]
	if !ok {
		t.Fatalf("expected a bucket for Saturday")
	}
	if sat.Count != 1 || sat.AverageSgv != 100 {
		t.Errorf("Saturday Count/AverageSgv = %d/%v, want 1/100", sat.Count, sat.AverageSgv)
	}

	sun, ok := byWeekday[time.Sunday]
	if !ok {
		t.Fatalf("expected a bucket for Sunday")
	}
	if sun.Count != 2 {
		t.Errorf("Sunday Count = %d, want 2", sun.Count)
	}
	if sun.AverageSgv != 115 {
		t.Errorf("Sunday AverageSgv = %v, want 115", sun.AverageSgv)
	}

	if _, ok := byWeekday[time.Monday]; ok {
		t.Errorf("Monday has no readings and should be omitted")
	}
}

func TestComputeDayOfWeekPatternsEmpty(t *testing.T) {
	patterns := ComputeDayOfWeekPatterns(nil)
	if len(patterns) != 0 {
		t.Errorf("len(patterns) = %d, want 0", len(patterns))
	}
}

func TestComputeDayOfWeekPatternsOrderedByWeekday(t *testing.T) {
	// 2026-07-18 is a Saturday; verify output is ordered Sunday..Saturday
	// (time.Sunday == 0) regardless of insertion order.
	saturday := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	wednesday := saturday.Add(-3 * 24 * time.Hour)
	entries := []Nightscoutdb{
		{Sgv: 100, Ns_datetime: saturday},
		{Sgv: 100, Ns_datetime: wednesday},
	}

	patterns := ComputeDayOfWeekPatterns(entries)
	if len(patterns) != 2 {
		t.Fatalf("len(patterns) = %d, want 2", len(patterns))
	}
	if patterns[0].Weekday != time.Wednesday || patterns[1].Weekday != time.Saturday {
		t.Errorf("weekdays = %v, %v, want Wednesday, Saturday (ascending)", patterns[0].Weekday, patterns[1].Weekday)
	}
}
