package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	os.Clearenv()
	cfg := Load()
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("HTTPAddr: got %q want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("DatabaseURL: want empty when unset")
	}
}

func TestLoad_Override(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":3000")
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	cfg := Load()
	if cfg.HTTPAddr != ":3000" {
		t.Fatalf("HTTPAddr: got %q", cfg.HTTPAddr)
	}
	if cfg.DatabaseURL != "postgres://localhost/test" {
		t.Fatalf("DatabaseURL: got %q", cfg.DatabaseURL)
	}
}
