package web

import (
	"database/sql"
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
		api.GET("/device/current", deviceCurrentHandler(db_client, cache))
	}

	return r
}

// Serve starts the REST API on the given port (e.g. "8080").
func Serve(db_client *sql.DB, port string) error {
	return Router(db_client).Run(":" + port)
}
