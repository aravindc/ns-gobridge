package model

import (
	"math"
	"time"
)

// rollingBucketDuration is the fixed bucket size used to slice a lookback
// period into successive windows for trend-over-time reporting.
const rollingBucketDuration = 7 * 24 * time.Hour

type RollingWeek struct {
	From                      time.Time `json:"from"`
	To                        time.Time `json:"to"`
	Count                     int       `json:"count"`
	AverageSgv                float64   `json:"averageSgv"`
	TimeInRangePct            float64   `json:"timeInRangePct"`
	CoefficientOfVariationPct float64   `json:"coefficientOfVariationPct"`
}

// ComputeRollingTrend slices [from, to) into successive 7-day buckets and
// computes average glucose, time-in-range, and coefficient of variation for
// each, so that whether control is improving or worsening over time can be
// read off directly instead of inferred from a single whole-period stat.
// The final bucket is shortened to end at "to" rather than padded with an
// empty tail. Entries do not need to be pre-sorted.
func ComputeRollingTrend(entries []Nightscoutdb, from time.Time, to time.Time) []RollingWeek {
	if !to.After(from) {
		return nil
	}

	weeks := make([]RollingWeek, 0)
	for bucketStart := from; bucketStart.Before(to); bucketStart = bucketStart.Add(rollingBucketDuration) {
		bucketEnd := bucketStart.Add(rollingBucketDuration)
		if bucketEnd.After(to) {
			bucketEnd = to
		}

		var bucketEntries []Nightscoutdb
		for _, e := range entries {
			if !e.Ns_datetime.Before(bucketStart) && e.Ns_datetime.Before(bucketEnd) {
				bucketEntries = append(bucketEntries, e)
			}
		}

		weeks = append(weeks, computeRollingWeek(bucketEntries, bucketStart, bucketEnd))
	}

	return weeks
}

func computeRollingWeek(entries []Nightscoutdb, from, to time.Time) RollingWeek {
	w := RollingWeek{From: from, To: to}
	if len(entries) == 0 {
		return w
	}
	w.Count = len(entries)

	sum := 0
	inRange := 0
	for _, e := range entries {
		sum += e.Sgv
		if e.Sgv >= LowThreshold && e.Sgv <= HighThreshold {
			inRange++
		}
	}
	w.AverageSgv = float64(sum) / float64(len(entries))
	w.TimeInRangePct = 100 * float64(inRange) / float64(len(entries))

	var sumSquaredDiff float64
	for _, e := range entries {
		diff := float64(e.Sgv) - w.AverageSgv
		sumSquaredDiff += diff * diff
	}
	sd := math.Sqrt(sumSquaredDiff / float64(len(entries)))
	if w.AverageSgv != 0 {
		w.CoefficientOfVariationPct = 100 * sd / w.AverageSgv
	}

	return w
}
