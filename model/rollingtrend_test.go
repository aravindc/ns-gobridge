package model

import (
	"math"
	"testing"
	"time"
)

func TestComputeRollingTrend(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(14 * 24 * time.Hour) // exactly 2 full weeks

	entries := []Nightscoutdb{
		// Week 1: in range, stable.
		{Sgv: 100, Ns_datetime: from.Add(1 * time.Hour)},
		{Sgv: 105, Ns_datetime: from.Add(2 * time.Hour)},
		// Week 2: elevated, out of range.
		{Sgv: 200, Ns_datetime: from.Add(8 * 24 * time.Hour)},
		{Sgv: 210, Ns_datetime: from.Add(8*24*time.Hour + time.Hour)},
	}

	weeks := ComputeRollingTrend(entries, from, to)

	if len(weeks) != 2 {
		t.Fatalf("len(weeks) = %d, want 2", len(weeks))
	}

	w1 := weeks[0]
	if w1.Count != 2 {
		t.Errorf("week1.Count = %d, want 2", w1.Count)
	}
	if w1.AverageSgv != 102.5 {
		t.Errorf("week1.AverageSgv = %v, want 102.5", w1.AverageSgv)
	}
	if w1.TimeInRangePct != 100 {
		t.Errorf("week1.TimeInRangePct = %v, want 100", w1.TimeInRangePct)
	}

	w2 := weeks[1]
	if w2.Count != 2 {
		t.Errorf("week2.Count = %d, want 2", w2.Count)
	}
	if w2.AverageSgv != 205 {
		t.Errorf("week2.AverageSgv = %v, want 205", w2.AverageSgv)
	}
	if w2.TimeInRangePct != 0 {
		t.Errorf("week2.TimeInRangePct = %v, want 0 (both readings > HighThreshold)", w2.TimeInRangePct)
	}

	// Control is worsening: TIR drops from 100% to 0%, average rises.
	if !(w2.AverageSgv > w1.AverageSgv && w2.TimeInRangePct < w1.TimeInRangePct) {
		t.Errorf("expected worsening trend from week1 to week2, got %+v -> %+v", w1, w2)
	}
}

func TestComputeRollingTrendBucketBoundaries(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(7 * 24 * time.Hour)

	entries := []Nightscoutdb{
		{Sgv: 100, Ns_datetime: from},                     // exactly at bucket start: included
		{Sgv: 100, Ns_datetime: to.Add(-time.Nanosecond)}, // just before bucket end: included
		{Sgv: 999, Ns_datetime: to},                       // exactly at "to": excluded (next bucket, but there is none)
	}

	weeks := ComputeRollingTrend(entries, from, to)
	if len(weeks) != 1 {
		t.Fatalf("len(weeks) = %d, want 1", len(weeks))
	}
	if weeks[0].Count != 2 {
		t.Errorf("Count = %d, want 2 (boundary reading at exactly \"to\" should be excluded)", weeks[0].Count)
	}
}

func TestComputeRollingTrendPartialFinalBucket(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// 10 days: one full week bucket plus a 3-day partial bucket, not padded to a full week.
	to := from.Add(10 * 24 * time.Hour)

	weeks := ComputeRollingTrend(nil, from, to)
	if len(weeks) != 2 {
		t.Fatalf("len(weeks) = %d, want 2", len(weeks))
	}
	if !weeks[1].To.Equal(to) {
		t.Errorf("weeks[1].To = %v, want %v (final bucket should end at \"to\", not be padded)", weeks[1].To, to)
	}
	gotDuration := weeks[1].To.Sub(weeks[1].From)
	wantDuration := 3 * 24 * time.Hour
	if gotDuration != wantDuration {
		t.Errorf("final bucket duration = %v, want %v", gotDuration, wantDuration)
	}
}

func TestComputeRollingTrendEmptyRange(t *testing.T) {
	from := time.Now()
	weeks := ComputeRollingTrend(nil, from, from)
	if weeks != nil {
		t.Errorf("weeks = %v, want nil for a zero-length range", weeks)
	}
}

func TestComputeRollingTrendEmptyBucket(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(7 * 24 * time.Hour)
	weeks := ComputeRollingTrend(nil, from, to)
	if len(weeks) != 1 {
		t.Fatalf("len(weeks) = %d, want 1", len(weeks))
	}
	if weeks[0].Count != 0 {
		t.Errorf("Count = %d, want 0", weeks[0].Count)
	}
	if weeks[0].AverageSgv != 0 || weeks[0].TimeInRangePct != 0 || weeks[0].CoefficientOfVariationPct != 0 {
		t.Errorf("expected all-zero stats for an empty bucket, got %+v", weeks[0])
	}
}

func TestComputeRollingTrendCV(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(7 * 24 * time.Hour)
	entries := []Nightscoutdb{
		{Sgv: 90, Ns_datetime: from.Add(time.Hour)},
		{Sgv: 100, Ns_datetime: from.Add(2 * time.Hour)},
		{Sgv: 110, Ns_datetime: from.Add(3 * time.Hour)},
	}
	weeks := ComputeRollingTrend(entries, from, to)
	if len(weeks) != 1 {
		t.Fatalf("len(weeks) = %d, want 1", len(weeks))
	}
	wantSD := math.Sqrt(200.0 / 3.0)
	wantCV := 100 * wantSD / 100
	if math.Abs(weeks[0].CoefficientOfVariationPct-wantCV) > 1e-9 {
		t.Errorf("CoefficientOfVariationPct = %v, want %v", weeks[0].CoefficientOfVariationPct, wantCV)
	}
}
