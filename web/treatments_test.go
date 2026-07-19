package web

import (
	"testing"
	"time"
)

func TestTreatmentFromRequest(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	req := createTreatmentRequest{
		Carbs:           45,
		Insulin:         4.5,
		MealType:        "lunch",
		FoodDescription: "pasta with garlic bread",
	}

	treatment, err := treatmentFromRequest(req, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if treatment.Carbs != 45 {
		t.Errorf("Carbs = %d, want 45", treatment.Carbs)
	}
	if treatment.Insulin != 4.5 {
		t.Errorf("Insulin = %v, want 4.5", treatment.Insulin)
	}
	if treatment.MealType != "lunch" {
		t.Errorf("MealType = %q, want lunch", treatment.MealType)
	}
	if treatment.FoodDescription != "pasta with garlic bread" {
		t.Errorf("FoodDescription = %q, want %q", treatment.FoodDescription, "pasta with garlic bread")
	}
	// No explicit Datetime given: should default to now.
	if !treatment.TreatmentDatetime.Equal(now) {
		t.Errorf("TreatmentDatetime = %v, want %v", treatment.TreatmentDatetime, now)
	}
	if treatment.TreatmentTime != now.UnixMilli() {
		t.Errorf("TreatmentTime = %d, want %d", treatment.TreatmentTime, now.UnixMilli())
	}
}

func TestTreatmentFromRequestExplicitDatetime(t *testing.T) {
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	explicit := time.Date(2026, 7, 18, 8, 30, 0, 0, time.UTC)
	req := createTreatmentRequest{
		Carbs:    20,
		Datetime: explicit.Format(time.RFC3339),
	}

	treatment, err := treatmentFromRequest(req, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !treatment.TreatmentDatetime.Equal(explicit) {
		t.Errorf("TreatmentDatetime = %v, want %v", treatment.TreatmentDatetime, explicit)
	}
	// Systime should still reflect when the request was processed, not the
	// logged treatment time.
	if !treatment.Systime.Equal(now) {
		t.Errorf("Systime = %v, want %v", treatment.Systime, now)
	}
}

func TestTreatmentFromRequestInvalidDatetime(t *testing.T) {
	req := createTreatmentRequest{Carbs: 10, Datetime: "not-a-timestamp"}
	_, err := treatmentFromRequest(req, time.Now())
	if err == nil {
		t.Fatal("expected an error for an invalid datetime, got nil")
	}
}

func TestTreatmentFromRequestInvalidMealType(t *testing.T) {
	req := createTreatmentRequest{Carbs: 10, MealType: "brunch"}
	_, err := treatmentFromRequest(req, time.Now())
	if err == nil {
		t.Fatal("expected an error for an invalid mealType, got nil")
	}
}

func TestTreatmentFromRequestEmptyMealTypeAllowed(t *testing.T) {
	req := createTreatmentRequest{Insulin: 2.0}
	_, err := treatmentFromRequest(req, time.Now())
	if err != nil {
		t.Fatalf("expected no error for an omitted mealType (correction bolus with no food), got: %v", err)
	}
}

func TestTreatmentFromRequestValidMealTypes(t *testing.T) {
	for _, mt := range []string{"breakfast", "lunch", "dinner", "snack"} {
		t.Run(mt, func(t *testing.T) {
			req := createTreatmentRequest{Carbs: 10, MealType: mt}
			if _, err := treatmentFromRequest(req, time.Now()); err != nil {
				t.Errorf("mealType %q should be valid, got error: %v", mt, err)
			}
		})
	}
}
