package model

import (
	"math"
	"time"
)

const (
	// rapidChangeMgdlPerMin is the commonly used clinical threshold above
	// which a glucose rate of change is considered "rapid" (e.g. Klonoff et
	// al. 2016, consensus on CGM rate-of-change alerts).
	rapidChangeMgdlPerMin = 2.0

	// maxRocGapMinutes bounds how far apart two consecutive readings can be
	// while still being used to compute a rate of change; a larger gap (e.g.
	// a sensor dropout) would produce a rate that isn't a real physiological
	// signal.
	maxRocGapMinutes = 15
)

type TrendCount struct {
	Trend int     `json:"trend"`
	Count int     `json:"count"`
	Pct   float64 `json:"pct"`
}

type RateOfChange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
	// Count is the number of readings in the period; TrendCounts summarize
	// the Dexcom-reported trend code (coarse, 1-7) for all of them.
	Count       int          `json:"count"`
	TrendCounts []TrendCount `json:"trendCounts"`

	// The following are derived from consecutive-reading Sgv/time deltas
	// (mg/dL per minute), a finer-grained signal than the Dexcom trend
	// code. RocSamples is the number of consecutive pairs used (Count-1,
	// less any pairs skipped for exceeding maxRocGapMinutes).
	RocSamples        int     `json:"rocSamples"`
	AverageAbsRoc     float64 `json:"averageAbsRoc"`
	MaxRoc            float64 `json:"maxRoc"`
	MinRoc            float64 `json:"minRoc"`
	RapidRiseEpisodes int     `json:"rapidRiseEpisodes"`
	RapidFallEpisodes int     `json:"rapidFallEpisodes"`
}

// ComputeRateOfChange derives trend-code distribution and rate-of-change
// statistics from a set of readings, expected to be ordered ascending by
// Ns_datetime (as returned by db.SelectEntriesBetween).
func ComputeRateOfChange(entries []Nightscoutdb, from time.Time, to time.Time) RateOfChange {
	r := RateOfChange{From: from, To: to}
	if len(entries) == 0 {
		return r
	}
	r.Count = len(entries)

	byTrend := make(map[int]int)
	for _, e := range entries {
		byTrend[e.Trend]++
	}
	r.TrendCounts = make([]TrendCount, 0, len(byTrend))
	for trend := 0; trend <= 9; trend++ {
		count, ok := byTrend[trend]
		if !ok {
			continue
		}
		r.TrendCounts = append(r.TrendCounts, TrendCount{
			Trend: trend,
			Count: count,
			Pct:   100 * float64(count) / float64(r.Count),
		})
	}

	if len(entries) < 2 {
		return r
	}

	var sumAbsRoc float64
	minRoc, maxRoc := math.Inf(1), math.Inf(-1)
	var lastRapidRiseAt, lastRapidFallAt time.Time

	for i := 1; i < len(entries); i++ {
		prev, cur := entries[i-1], entries[i]
		minutes := cur.Ns_datetime.Sub(prev.Ns_datetime).Minutes()
		if minutes <= 0 || minutes > maxRocGapMinutes {
			continue
		}

		roc := float64(cur.Sgv-prev.Sgv) / minutes
		r.RocSamples++
		sumAbsRoc += math.Abs(roc)
		if roc < minRoc {
			minRoc = roc
		}
		if roc > maxRoc {
			maxRoc = roc
		}

		switch {
		case roc >= rapidChangeMgdlPerMin:
			if lastRapidRiseAt.IsZero() || cur.Ns_datetime.Sub(lastRapidRiseAt) > episodeGapMinutes*time.Minute {
				r.RapidRiseEpisodes++
			}
			lastRapidRiseAt = cur.Ns_datetime
		case roc <= -rapidChangeMgdlPerMin:
			if lastRapidFallAt.IsZero() || cur.Ns_datetime.Sub(lastRapidFallAt) > episodeGapMinutes*time.Minute {
				r.RapidFallEpisodes++
			}
			lastRapidFallAt = cur.Ns_datetime
		}
	}

	if r.RocSamples > 0 {
		r.AverageAbsRoc = sumAbsRoc / float64(r.RocSamples)
		r.MinRoc = minRoc
		r.MaxRoc = maxRoc
	}

	return r
}
