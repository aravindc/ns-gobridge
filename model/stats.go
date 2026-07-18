package model

import "time"

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
