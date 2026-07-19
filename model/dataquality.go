package model

import (
	"sort"
	"time"
)

const (
	// expectedReadingInterval is the nominal Dexcom Share polling cadence
	// (main.go polls every minute, but the sensor itself reports roughly
	// every 5 minutes; readings can also be deduped/missed).
	expectedReadingInterval = 5 * time.Minute

	// gapThreshold is how large a delay between consecutive readings must
	// be before it's counted as a gap (missing data) rather than normal
	// jitter around the expected interval.
	gapThreshold = 2 * expectedReadingInterval
)

type Gap struct {
	From     time.Time     `json:"from"`
	To       time.Time     `json:"to"`
	Duration time.Duration `json:"-"`
}

type DataQuality struct {
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	Count int       `json:"count"`
	// ExpectedCount is roughly how many readings should exist across
	// [From, To) at the expected cadence, used to derive CoveragePct.
	ExpectedCount int     `json:"expectedCount"`
	CoveragePct   float64 `json:"coveragePct"`
	Gaps          []Gap   `json:"gaps"`
	// LargestGap is the single longest gap in the period, zero-valued if
	// there were no gaps.
	LargestGap Gap `json:"largestGap"`
}

// ComputeDataQuality detects gaps (missing data, e.g. sensor dropouts) in a
// set of readings and derives a coverage percentage, so that TIR/HbA1c/GMI
// figures computed elsewhere can be understood in context: a period with
// large gaps (e.g. an overnight dropout) may not be representative.
// Entries do not need to be pre-sorted.
func ComputeDataQuality(entries []Nightscoutdb, from time.Time, to time.Time) DataQuality {
	dq := DataQuality{From: from, To: to}
	if to.After(from) {
		dq.ExpectedCount = int(to.Sub(from) / expectedReadingInterval)
	}
	if len(entries) == 0 {
		return dq
	}
	dq.Count = len(entries)
	if dq.ExpectedCount > 0 {
		dq.CoveragePct = 100 * float64(dq.Count) / float64(dq.ExpectedCount)
		if dq.CoveragePct > 100 {
			dq.CoveragePct = 100
		}
	}

	sorted := make([]Nightscoutdb, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Ns_datetime.Before(sorted[j].Ns_datetime)
	})

	for i := 1; i < len(sorted); i++ {
		prev, cur := sorted[i-1], sorted[i]
		gapDuration := cur.Ns_datetime.Sub(prev.Ns_datetime)
		if gapDuration <= gapThreshold {
			continue
		}
		gap := Gap{From: prev.Ns_datetime, To: cur.Ns_datetime, Duration: gapDuration}
		dq.Gaps = append(dq.Gaps, gap)
		if gap.Duration > dq.LargestGap.Duration {
			dq.LargestGap = gap
		}
	}

	return dq
}
