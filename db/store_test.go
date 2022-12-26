package db

import (
	"ns-gobridge/model"
	"reflect"
	"testing"

	"github.com/supabase/postgrest-go"
)

func TestDbClient(t *testing.T) {
	type args struct {
		supabase_connection_string string
	}
	tests := []struct {
		name string
		args args
		want *postgrest.Client
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DbClient(tt.args.supabase_connection_string); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DbClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSelectEntries(t *testing.T) {
	type args struct {
		db_client *postgrest.Client
	}
	tests := []struct {
		name string
		args args
		want []model.NsBgEntry
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SelectEntries(tt.args.db_client); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SelectEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntriesExist(t *testing.T) {
	type args struct {
		db_client *postgrest.Client
		nsBgEntry model.NsBgEntry
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EntriesExist(tt.args.db_client, tt.args.nsBgEntry); got != tt.want {
				t.Errorf("EntriesExist() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInsertEntries(t *testing.T) {
	type args struct {
		db_client *postgrest.Client
		nsItem    model.NsBgEntry
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InsertEntries(tt.args.db_client, tt.args.nsItem)
		})
	}
}
