package common

import (
	"reflect"
	"testing"
)

func TestTrendToDirection(t *testing.T) {
	type args struct {
		trend string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "Flat with right value",
			args: args{trend: "Flat"},
			want: 4,
		},
		{
			name: "Unknown value",
			args: args{trend: "UnknownValue"},
			want: 99,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TrendToDirection(tt.args.trend); got != tt.want {
				t.Errorf("TrendToDirection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTernaryIf(t *testing.T) {
	X := "123"
	type args struct {
		cond   bool
		vtrue  any
		vfalse any
	}
	tests := []struct {
		name string
		args args
		want any
	}{
		{name: "Test if X = X", args: args{cond: (X == "123"), vtrue: true, vfalse: false}, want: true},
		{name: "Test if X = Y", args: args{cond: (X == "Y"), vtrue: true, vfalse: false}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TernaryIf(tt.args.cond, tt.args.vtrue, tt.args.vfalse); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TernaryIf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCleanString(t *testing.T) {
	type args struct {
		input_string string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "String with additional quote", args: args{input_string: "\"12345\""}, want: "12345"},
		{name: "String without additional quote", args: args{input_string: "12345"}, want: "12345"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanString(tt.args.input_string); got != tt.want {
				t.Errorf("CleanString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCleanDateString(t *testing.T) {
	type args struct {
		input_string string
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{name: "Strip Date() from string", args: args{input_string: "Date(1672008331)"}, want: 1672008331},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanDateString(tt.args.input_string); got != tt.want {
				t.Errorf("CleanDateString() = %v, want %v", got, tt.want)
			}
		})
	}
}
