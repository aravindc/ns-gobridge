package db

import (
	"context"
	"database/sql"
	"ns-gobridge/model"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func InsertTreatment(db_client *sql.DB, treatment model.Treatment) (model.Treatment, error) {
	db := bun.NewDB(db_client, pgdialect.New())
	ctx := context.Background()
	newTreatment := &model.Treatment{
		Carbs:             treatment.Carbs,
		Insulin:           treatment.Insulin,
		MealType:          treatment.MealType,
		FoodDescription:   treatment.FoodDescription,
		TreatmentTime:     treatment.TreatmentTime,
		TreatmentDatetime: treatment.TreatmentDatetime,
		Systime:           treatment.Systime,
	}
	_, err := db.NewInsert().Model(newTreatment).Returning("id").Exec(ctx)
	return *newTreatment, err
}

func SelectTreatmentsBetween(db_client *sql.DB, from int64, to int64) ([]model.Treatment, error) {
	db := bun.NewDB(db_client, pgdialect.New())
	ctx := context.Background()
	var treatments []model.Treatment
	err := db.NewSelect().
		Model(&treatments).
		Where("treatment_time >= ?", from).
		Where("treatment_time <= ?", to).
		OrderExpr("treatment_time ASC").
		Scan(ctx)
	return treatments, err
}
