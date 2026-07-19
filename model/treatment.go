package model

import (
	"time"

	"github.com/uptrace/bun"
)

// MealTypes are the allowed values for Treatment.MealType.
var MealTypes = map[string]bool{
	"breakfast": true,
	"lunch":     true,
	"dinner":    true,
	"snack":     true,
}

type Treatment struct {
	bun.BaseModel `bun:"table:treatments,alias:tr"`

	Id                int64     `bun:"id,pk,autoincrement" json:"id"`
	Carbs             int       `bun:"carbs" json:"carbs"`
	Insulin           float64   `bun:"insulin" json:"insulin"`
	MealType          string    `bun:"meal_type" json:"mealType"`
	FoodDescription   string    `bun:"food_description" json:"foodDescription"`
	TreatmentTime     int64     `bun:"treatment_time,type:bigint" json:"treatmentTime"`
	TreatmentDatetime time.Time `bun:"treatment_datetime,type:timestampz" json:"treatmentDatetime"`
	Systime           time.Time `bun:"systime,type:timestampz" json:"-"`
}
