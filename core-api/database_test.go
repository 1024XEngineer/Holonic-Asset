package main

import (
	"context"
	"testing"

	"github.com/1024XEngineer/Holonic-Asset/config"
)

func TestInitDBRejectsEmptyDSN(t *testing.T) {
	if _, err := InitDB(context.Background(), &config.DBConfig{}, nil); err == nil {
		t.Fatal("expected empty database DSN to be rejected")
	}
}

func TestInitDBRejectsNilConfig(t *testing.T) {
	if _, err := InitDB(context.Background(), nil, nil); err == nil {
		t.Fatal("expected nil database config to be rejected")
	}
}
