package web

import (
	"math"
	"os"

	"github.com/gin-gonic/gin"
)

const mgdlPerMmol = 18.0182

// resolveUnits determines the glucose unit for a request: the "units" query
// param if present ("mg/dl" or "mmol"), otherwise the UNITS env var,
// otherwise "mg/dl".
func resolveUnits(c *gin.Context) string {
	if v := c.Query("units"); v == "mg/dl" || v == "mmol" {
		return v
	}
	if v := os.Getenv("UNITS"); v == "mg/dl" || v == "mmol" {
		return v
	}
	return "mg/dl"
}

// sgvForUnits converts an mg/dL glucose value for display in the given
// units, rounding mmol/L to one decimal place as is conventional.
func sgvForUnits(mgdl int, units string) float64 {
	if units == "mmol" {
		return math.Round(float64(mgdl)/mgdlPerMmol*10) / 10
	}
	return float64(mgdl)
}
