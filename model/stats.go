package model

import (
	"math"
	"sort"
	"time"
)

const (
	LowThreshold  = 70
	HighThreshold = 180

	// Episode boundaries: a run of consecutive out-of-range readings
	// separated by less than this gap counts as the same episode.
	episodeGapMinutes = 15
)

type Stats struct {
	From              time.Time `json:"from"`
	To                time.Time `json:"to"`
	Count             int       `json:"count"`
	AverageSgv        float64   `json:"averageSgv"`
	MinSgv            int       `json:"minSgv"`
	MaxSgv            int       `json:"maxSgv"`
	EstimatedHba1c    float64   `json:"estimatedHba1c"`
	Gmi               float64   `json:"gmi"`
	TimeInRangePct    float64   `json:"timeInRangePct"`
	TimeBelowRangePct float64   `json:"timeBelowRangePct"`
	TimeAboveRangePct float64   `json:"timeAboveRangePct"`
	LowEpisodes       int       `json:"lowEpisodes"`
	HighEpisodes      int       `json:"highEpisodes"`
}

// ComputeStats derives glucose insights from a set of readings, expected to
// be ordered ascending by Ns_datetime.
func ComputeStats(entries []Nightscoutdb, from time.Time, to time.Time) Stats {
	stats := Stats{From: from, To: to}
	if len(entries) == 0 {
		return stats
	}

	stats.Count = len(entries)
	sum := 0
	inRange, below, above := 0, 0, 0
	minSgv, maxSgv := entries[0].Sgv, entries[0].Sgv

	var lastLowAt, lastHighAt time.Time
	for _, e := range entries {
		sum += e.Sgv
		if e.Sgv < minSgv {
			minSgv = e.Sgv
		}
		if e.Sgv > maxSgv {
			maxSgv = e.Sgv
		}

		switch {
		case e.Sgv < LowThreshold:
			below++
			if lastLowAt.IsZero() || e.Ns_datetime.Sub(lastLowAt) > episodeGapMinutes*time.Minute {
				stats.LowEpisodes++
			}
			lastLowAt = e.Ns_datetime
		case e.Sgv > HighThreshold:
			above++
			if lastHighAt.IsZero() || e.Ns_datetime.Sub(lastHighAt) > episodeGapMinutes*time.Minute {
				stats.HighEpisodes++
			}
			lastHighAt = e.Ns_datetime
		default:
			inRange++
		}
	}

	stats.MinSgv = minSgv
	stats.MaxSgv = maxSgv
	stats.AverageSgv = float64(sum) / float64(len(entries))
	// Standard estimated HbA1c formula (NGSP), derived from average glucose in mg/dL.
	stats.EstimatedHba1c = (stats.AverageSgv + 46.7) / 28.7
	// Glucose Management Indicator (Bergenstal et al. 2018), also derived from
	// average glucose in mg/dL. Distinct from (and generally lower than) the
	// NGSP-based estimated HbA1c above.
	stats.Gmi = 3.31 + 0.02392*stats.AverageSgv
	stats.TimeInRangePct = 100 * float64(inRange) / float64(len(entries))
	stats.TimeBelowRangePct = 100 * float64(below) / float64(len(entries))
	stats.TimeAboveRangePct = 100 * float64(above) / float64(len(entries))

	return stats
}

type Quartiles struct {
	From   time.Time `json:"from"`
	To     time.Time `json:"to"`
	Count  int       `json:"count"`
	Min    int       `json:"min"`
	Q1     float64   `json:"q1"`
	Median float64   `json:"median"`
	Q3     float64   `json:"q3"`
	Max    int       `json:"max"`
}

// percentile returns the p-th percentile (0<=p<=1) of a slice already
// sorted ascending, using linear interpolation between closest ranks.
func percentile(sorted []int, p float64) float64 {
	if len(sorted) == 1 {
		return float64(sorted[0])
	}
	rank := p * float64(len(sorted)-1)
	lo := int(rank)
	hi := lo + 1
	if hi >= len(sorted) {
		return float64(sorted[lo])
	}
	frac := rank - float64(lo)
	return float64(sorted[lo]) + frac*float64(sorted[hi]-sorted[lo])
}

// ComputeQuartiles derives glucose quartiles (Q1/median/Q3) plus min/max
// from a set of readings. Entries do not need to be pre-sorted; a sorted
// copy of the Sgv values is used for percentile computation.
func ComputeQuartiles(entries []Nightscoutdb, from time.Time, to time.Time) Quartiles {
	q := Quartiles{From: from, To: to}
	if len(entries) == 0 {
		return q
	}

	sgvs := make([]int, len(entries))
	for i, e := range entries {
		sgvs[i] = e.Sgv
	}
	sort.Ints(sgvs)

	q.Count = len(sgvs)
	q.Min = sgvs[0]
	q.Max = sgvs[len(sgvs)-1]
	q.Q1 = percentile(sgvs, 0.25)
	q.Median = percentile(sgvs, 0.5)
	q.Q3 = percentile(sgvs, 0.75)

	return q
}

type Variability struct {
	From              time.Time `json:"from"`
	To                time.Time `json:"to"`
	Count             int       `json:"count"`
	AverageSgv        float64   `json:"averageSgv"`
	StandardDeviation float64   `json:"standardDeviation"`
	// CoefficientOfVariationPct is SD/mean as a percentage. The ADA/ATTD
	// consensus (Battelino et al. 2019) treats <=36% as an indicator of
	// stable glycemic control; higher values suggest unstable control
	// independent of how much time is spent in range.
	CoefficientOfVariationPct float64 `json:"coefficientOfVariationPct"`
}

// ComputeVariability derives glycemic variability (population standard
// deviation and coefficient of variation) from a set of readings.
func ComputeVariability(entries []Nightscoutdb, from time.Time, to time.Time) Variability {
	v := Variability{From: from, To: to}
	if len(entries) == 0 {
		return v
	}

	v.Count = len(entries)
	sum := 0
	for _, e := range entries {
		sum += e.Sgv
	}
	v.AverageSgv = float64(sum) / float64(len(entries))

	var sumSquaredDiff float64
	for _, e := range entries {
		diff := float64(e.Sgv) - v.AverageSgv
		sumSquaredDiff += diff * diff
	}
	v.StandardDeviation = math.Sqrt(sumSquaredDiff / float64(len(entries)))

	if v.AverageSgv != 0 {
		v.CoefficientOfVariationPct = 100 * v.StandardDeviation / v.AverageSgv
	}

	return v
}
