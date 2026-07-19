package web

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"ns-gobridge/common"
	"ns-gobridge/db"
	"ns-gobridge/model"

	"github.com/gin-gonic/gin"
)

const (
	latestEntryCacheKey = "latest-entry"
	statsCacheKey       = "stats:default"
	cacheTTL            = 10 * time.Second
)

// periods maps supported ?period= values to their duration, looking back
// from now. 1mth/3mths use fixed 30/90-day approximations rather than
// calendar months, to keep the lookup a simple duration table.
var periods = map[string]time.Duration{
	"24h":   24 * time.Hour,
	"1wk":   7 * 24 * time.Hour,
	"1mth":  30 * 24 * time.Hour,
	"3mths": 90 * 24 * time.Hour,
}

// requireAPIKey checks the X-API-Key header against API_KEY. If API_KEY is
// unset, auth is disabled (e.g. for local development).
func requireAPIKey() gin.HandlerFunc {
	apiKey := os.Getenv("API_KEY")
	return func(c *gin.Context) {
		if apiKey == "" {
			c.Next()
			return
		}
		if c.GetHeader("X-API-Key") != apiKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing X-API-Key header"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func parseRange(c *gin.Context) (time.Time, time.Time, error) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now

	if v := c.Query("from"); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return from, to, err
		}
		from = parsed
	}
	if v := c.Query("to"); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return from, to, err
		}
		to = parsed
	}
	return from, to, nil
}

// parsePeriod resolves the ?period= query param (one of "24h", "1wk",
// "1mth", "3mths") to a from/to range ending now, and the normalized period
// string. Defaults to defaultPeriod when absent; returns an error for any
// other value.
func parsePeriod(c *gin.Context, defaultPeriod string) (time.Time, time.Time, string, error) {
	period := c.Query("period")
	if period == "" {
		period = defaultPeriod
	}
	dur, ok := periods[period]
	if !ok {
		return time.Time{}, time.Time{}, period, fmt.Errorf("invalid period %q", period)
	}
	to := time.Now()
	from := to.Add(-dur)
	return from, to, period, nil
}

// getLatestEntry fetches the latest reading, using the cache to avoid
// re-querying Postgres on every request when many clients poll concurrently.
func getLatestEntry(db_client *sql.DB, cache *ttlCache) (model.Nightscoutdb, error) {
	if cached, ok := cache.get(latestEntryCacheKey); ok {
		return cached.(model.Nightscoutdb), nil
	}
	entry, err := db.SelectLatestEntry(db_client)
	if err != nil {
		return entry, err
	}
	cache.set(latestEntryCacheKey, entry)
	return entry, nil
}

func currentHandler(db_client *sql.DB, cache *ttlCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		entry, err := getLatestEntry(db_client, cache)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		units := resolveUnits(c)
		c.JSON(http.StatusOK, gin.H{
			"sgv":       sgvForUnits(entry.Sgv, units),
			"units":     units,
			"trend":     entry.Trend,
			"direction": common.DirectionToArrow(entry.Trend),
			"datetime":  entry.Ns_datetime,
		})
	}
}

// deviceCurrentHandler returns a minimal, flat payload for constrained IoT
// clients (e.g. M5Stack): short keys, no nesting, cheap to parse on-device.
func deviceCurrentHandler(db_client *sql.DB, cache *ttlCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		entry, err := getLatestEntry(db_client, cache)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		units := resolveUnits(c)
		c.JSON(http.StatusOK, gin.H{
			"sgv":      sgvForUnits(entry.Sgv, units),
			"units":    units,
			"dir":      common.DirectionToArrow(entry.Trend),
			"mins_ago": int(time.Since(entry.Ns_datetime).Minutes()),
		})
	}
}

func entriesHandler(db_client *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from/to must be RFC3339 timestamps"})
			return
		}
		entries, err := db.SelectEntriesBetween(db_client, from.UnixMilli(), to.UnixMilli())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		units := resolveUnits(c)
		out := make([]gin.H, len(entries))
		for i, e := range entries {
			out[i] = gin.H{
				"sgv":       sgvForUnits(e.Sgv, units),
				"units":     units,
				"trend":     e.Trend,
				"direction": common.DirectionToArrow(e.Trend),
				"datetime":  e.Ns_datetime,
			}
		}
		c.JSON(http.StatusOK, out)
	}
}

// statsResponse formats a model.Stats for display in the given units.
// TimeInRange/episode figures are unit-independent and passed through as-is.
func statsResponse(stats model.Stats, units string) gin.H {
	return gin.H{
		"from":              stats.From,
		"to":                stats.To,
		"units":             units,
		"count":             stats.Count,
		"averageSgv":        sgvForUnits(int(math.Round(stats.AverageSgv)), units),
		"minSgv":            sgvForUnits(stats.MinSgv, units),
		"maxSgv":            sgvForUnits(stats.MaxSgv, units),
		"estimatedHba1c":    stats.EstimatedHba1c,
		"gmi":               stats.Gmi,
		"timeInRangePct":    stats.TimeInRangePct,
		"timeBelowRangePct": stats.TimeBelowRangePct,
		"timeAboveRangePct": stats.TimeAboveRangePct,
		"lowEpisodes":       stats.LowEpisodes,
		"highEpisodes":      stats.HighEpisodes,
	}
}

func statsHandler(db_client *sql.DB, cache *ttlCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		units := resolveUnits(c)

		// Only cache the default (no from/to) range, which is what repeat
		// pollers hit; explicit ranges bypass the cache and hit Postgres.
		// The cache stores the raw mg/dL stats; unit formatting always
		// happens at response time so both units share one cache entry.
		isDefaultRange := c.Query("from") == "" && c.Query("to") == ""
		if isDefaultRange {
			if cached, ok := cache.get(statsCacheKey); ok {
				c.JSON(http.StatusOK, statsResponse(cached.(model.Stats), units))
				return
			}
		}

		from, to, err := parseRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from/to must be RFC3339 timestamps"})
			return
		}
		entries, err := db.SelectEntriesBetween(db_client, from.UnixMilli(), to.UnixMilli())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		stats := model.ComputeStats(entries, from, to)
		if isDefaultRange {
			cache.set(statsCacheKey, stats)
		}
		c.JSON(http.StatusOK, statsResponse(stats, units))
	}
}

// quartilesResponse formats a model.Quartiles for display in the given units.
func quartilesResponse(period string, q model.Quartiles, units string) gin.H {
	return gin.H{
		"period": period,
		"from":   q.From,
		"to":     q.To,
		"units":  units,
		"count":  q.Count,
		"min":    sgvForUnits(q.Min, units),
		"q1":     sgvForUnits(int(math.Round(q.Q1)), units),
		"median": sgvForUnits(int(math.Round(q.Median)), units),
		"q3":     sgvForUnits(int(math.Round(q.Q3)), units),
		"max":    sgvForUnits(q.Max, units),
	}
}

// quartilesHandler returns glucose quartiles (Q1/median/Q3, plus min/max)
// for a given lookback period: ?period=24h|1wk|1mth|3mths (default 24h).
func quartilesHandler(db_client *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, period, err := parsePeriod(c, "24h")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "period must be one of: 24h, 1wk, 1mth, 3mths"})
			return
		}
		entries, err := db.SelectEntriesBetween(db_client, from.UnixMilli(), to.UnixMilli())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		units := resolveUnits(c)
		quartiles := model.ComputeQuartiles(entries, from, to)
		c.JSON(http.StatusOK, quartilesResponse(period, quartiles, units))
	}
}

// hourlyPatternsResponse formats []model.HourlyPattern for display in the given units.
func hourlyPatternsResponse(period string, from, to time.Time, patterns []model.HourlyPattern, units string) gin.H {
	out := make([]gin.H, len(patterns))
	for i, p := range patterns {
		out[i] = gin.H{
			"hour":       p.Hour,
			"count":      p.Count,
			"averageSgv": sgvForUnits(int(math.Round(p.AverageSgv)), units),
			"min":        sgvForUnits(p.Min, units),
			"q1":         sgvForUnits(int(math.Round(p.Q1)), units),
			"median":     sgvForUnits(int(math.Round(p.Median)), units),
			"q3":         sgvForUnits(int(math.Round(p.Q3)), units),
			"max":        sgvForUnits(p.Max, units),
		}
	}
	return gin.H{
		"period": period,
		"from":   from,
		"to":     to,
		"units":  units,
		"hourly": out,
	}
}

// hourlyPatternsHandler returns glucose statistics bucketed by hour-of-day
// (0-23) over a given lookback period: ?period=24h|1wk|1mth|3mths (default
// 1mth, since recurring daily patterns need more than one day of data per
// hour bucket to be meaningful). Hours with no readings in the period are
// omitted from the response.
func hourlyPatternsHandler(db_client *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, period, err := parsePeriod(c, "1mth")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "period must be one of: 24h, 1wk, 1mth, 3mths"})
			return
		}
		entries, err := db.SelectEntriesBetween(db_client, from.UnixMilli(), to.UnixMilli())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		units := resolveUnits(c)
		patterns := model.ComputeHourlyPatterns(entries)
		c.JSON(http.StatusOK, hourlyPatternsResponse(period, from, to, patterns, units))
	}
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Router builds the REST API for querying CGM readings and derived insights.
func Router(db_client *sql.DB) *gin.Engine {
	r := gin.Default()
	cache := newTTLCache(cacheTTL)

	r.GET("/api/health", healthHandler)

	api := r.Group("/api")
	api.Use(requireAPIKey())
	{
		api.GET("/current", currentHandler(db_client, cache))
		api.GET("/entries", entriesHandler(db_client))
		api.GET("/stats", statsHandler(db_client, cache))
		api.GET("/quartiles", quartilesHandler(db_client))
		api.GET("/patterns/hourly", hourlyPatternsHandler(db_client))
		api.GET("/device/current", deviceCurrentHandler(db_client, cache))
	}

	return r
}

// Serve starts the REST API on the given port (e.g. "8080").
func Serve(db_client *sql.DB, port string) error {
	return Router(db_client).Run(":" + port)
}
