package web

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"ns-gobridge/db"
	"ns-gobridge/model"

	"github.com/gin-gonic/gin"
)

// createTreatmentRequest is the POST /api/treatments request body. Carbs
// and Insulin are both optional (either can be zero) so a correction bolus
// with no food, or carbs logged without a dose, can each be recorded with
// the same endpoint.
type createTreatmentRequest struct {
	Carbs           int     `json:"carbs" binding:"min=0"`
	Insulin         float64 `json:"insulin" binding:"min=0"`
	MealType        string  `json:"mealType"`
	FoodDescription string  `json:"foodDescription"`
	Datetime        string  `json:"datetime"`
}

// treatmentResponse formats a model.Treatment for display.
func treatmentResponse(t model.Treatment) gin.H {
	return gin.H{
		"id":                t.Id,
		"carbs":             t.Carbs,
		"insulin":           t.Insulin,
		"mealType":          t.MealType,
		"foodDescription":   t.FoodDescription,
		"treatmentDatetime": t.TreatmentDatetime,
	}
}

// treatmentFromRequest validates a createTreatmentRequest and converts it to
// a model.Treatment ready for insertion. now is passed in so callers (and
// tests) control the default Datetime/Systime rather than relying on the
// wall clock.
func treatmentFromRequest(req createTreatmentRequest, now time.Time) (model.Treatment, error) {
	if req.MealType != "" && !model.MealTypes[req.MealType] {
		return model.Treatment{}, fmt.Errorf("mealType must be one of: breakfast, lunch, dinner, snack")
	}

	datetime := now
	if req.Datetime != "" {
		parsed, err := time.Parse(time.RFC3339, req.Datetime)
		if err != nil {
			return model.Treatment{}, fmt.Errorf("datetime must be an RFC3339 timestamp")
		}
		datetime = parsed
	}

	return model.Treatment{
		Carbs:             req.Carbs,
		Insulin:           req.Insulin,
		MealType:          req.MealType,
		FoodDescription:   req.FoodDescription,
		TreatmentTime:     datetime.UnixMilli(),
		TreatmentDatetime: datetime,
		Systime:           now,
	}, nil
}

// createTreatmentHandler logs a carbs/insulin treatment. datetime (RFC3339)
// is optional and defaults to now; mealType, if given, must be one of
// breakfast/lunch/dinner/snack.
func createTreatmentHandler(db_client *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createTreatmentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body (carbs and insulin must be non-negative numbers): " + err.Error()})
			return
		}

		treatment, err := treatmentFromRequest(req, time.Now())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		saved, err := db.InsertTreatment(db_client, treatment)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, treatmentResponse(saved))
	}
}

// treatmentsHandler lists logged carbs/insulin treatments in a time range:
// ?from=/?to= (RFC3339), defaulting to the last 24h, matching entriesHandler.
func treatmentsHandler(db_client *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		from, to, err := parseRange(c)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from/to must be RFC3339 timestamps"})
			return
		}
		treatments, err := db.SelectTreatmentsBetween(db_client, from.UnixMilli(), to.UnixMilli())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out := make([]gin.H, len(treatments))
		for i, t := range treatments {
			out[i] = treatmentResponse(t)
		}
		c.JSON(http.StatusOK, out)
	}
}
