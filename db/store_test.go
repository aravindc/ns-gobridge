package db

import (
	"testing"
)

func TestDbClient(t *testing.T) {
	client := DbClient("localhost", "5432", "postgres", "secret", "health", "disable")
	if client == nil {
		t.Errorf("DbClient() returned nil")
	}
	defer client.Close()
}
