package web

import "testing"

func TestSgvForUnits(t *testing.T) {
	tests := []struct {
		name  string
		mgdl  int
		units string
		want  float64
	}{
		{name: "mg/dl passthrough", mgdl: 120, units: "mg/dl", want: 120},
		{name: "mmol conversion", mgdl: 126, units: "mmol", want: 7.0},
		{name: "mmol conversion rounds to 1 decimal", mgdl: 100, units: "mmol", want: 5.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sgvForUnits(tt.mgdl, tt.units); got != tt.want {
				t.Errorf("sgvForUnits(%d, %q) = %v, want %v", tt.mgdl, tt.units, got, tt.want)
			}
		})
	}
}

func TestRateForUnits(t *testing.T) {
	tests := []struct {
		name       string
		mgdlPerMin float64
		units      string
		want       float64
	}{
		{name: "mg/dl passthrough", mgdlPerMin: 2.0, units: "mg/dl", want: 2.0},
		{name: "mg/dl rounds to 2 decimals", mgdlPerMin: 2.006, units: "mg/dl", want: 2.01},
		{name: "mmol conversion", mgdlPerMin: 18.0182, units: "mmol", want: 1.0},
		{name: "negative rate", mgdlPerMin: -4.4, units: "mg/dl", want: -4.4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rateForUnits(tt.mgdlPerMin, tt.units); got != tt.want {
				t.Errorf("rateForUnits(%v, %q) = %v, want %v", tt.mgdlPerMin, tt.units, got, tt.want)
			}
		})
	}
}
