package model

import (
	"sort"
	"time"
)

type DayOfWeekPattern struct {
	Weekday    time.Weekday `json:"weekday"`
	Count      int          `json:"count"`
	AverageSgv float64      `json:"averageSgv"`
	Min        int          `json:"min"`
	Q1         float64      `json:"q1"`
	Median     float64      `json:"median"`
	Q3         float64      `json:"q3"`
	Max        int          `json:"max"`
}

// ComputeDayOfWeekPatterns buckets readings by day of week (Sunday-Saturday,
// in the reading's own Ns_datetime location) and derives average/quartiles
// per bucket. Useful for spotting weekday-vs-weekend differences in control
// (e.g. diet or activity changes) that a whole-period average would smooth
// over. Days with no readings are omitted rather than returned as zeroed
// buckets.
func ComputeDayOfWeekPatterns(entries []Nightscoutdb) []DayOfWeekPattern {
	byWeekday := make(map[time.Weekday][]int)
	for _, e := range entries {
		weekday := e.Ns_datetime.Weekday()
		byWeekday[weekday] = append(byWeekday[weekday], e.Sgv)
	}

	patterns := make([]DayOfWeekPattern, 0, len(byWeekday))
	for weekday := time.Sunday; weekday <= time.Saturday; weekday++ {
		sgvs, ok := byWeekday[weekday]
		if !ok {
			continue
		}
		sort.Ints(sgvs)

		sum := 0
		for _, v := range sgvs {
			sum += v
		}

		patterns = append(patterns, DayOfWeekPattern{
			Weekday:    weekday,
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
