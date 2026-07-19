package model

import "sort"

type HourlyPattern struct {
	Hour       int     `json:"hour"`
	Count      int     `json:"count"`
	AverageSgv float64 `json:"averageSgv"`
	Min        int     `json:"min"`
	Q1         float64 `json:"q1"`
	Median     float64 `json:"median"`
	Q3         float64 `json:"q3"`
	Max        int     `json:"max"`
}

// ComputeHourlyPatterns buckets readings by hour-of-day (0-23, in the
// reading's own Ns_datetime location) across the full span of entries, and
// derives average/quartiles per bucket. Useful for spotting recurring
// patterns like the dawn phenomenon or post-meal spikes that a single
// whole-period average or quartile set would smooth over. Hours with no
// readings are omitted rather than returned as zeroed buckets.
func ComputeHourlyPatterns(entries []Nightscoutdb) []HourlyPattern {
	byHour := make(map[int][]int, 24)
	for _, e := range entries {
		hour := e.Ns_datetime.Hour()
		byHour[hour] = append(byHour[hour], e.Sgv)
	}

	patterns := make([]HourlyPattern, 0, len(byHour))
	for hour := 0; hour < 24; hour++ {
		sgvs, ok := byHour[hour]
		if !ok {
			continue
		}
		sort.Ints(sgvs)

		sum := 0
		for _, v := range sgvs {
			sum += v
		}

		patterns = append(patterns, HourlyPattern{
			Hour:       hour,
			Count:      len(sgvs),
			AverageSgv: float64(sum) / float64(len(sgvs)),
			Min:        sgvs[0],
			Q1:         percentile(sgvs, 0.25),
			Median:     percentile(sgvs, 0.5),
			Q3:         percentile(sgvs, 0.75),
			Max:        sgvs[len(sgvs)-1],
		})
	}

	return patterns
}
