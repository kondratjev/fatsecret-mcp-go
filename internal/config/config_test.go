package config

import (
	"os"
	"testing"
)

func TestSaveAndResolve(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("FATSECRET_CLIENT_ID", "")
	t.Setenv("FATSECRET_CLIENT_SECRET", "")
	t.Setenv("FATSECRET_CONSUMER_SECRET", "")

	if err := Save(map[string]string{"clientId": "file-id", "clientSecret": "file-secret"}); err != nil {
		t.Fatal(err)
	}
	cfg := Resolve()
	if cfg.ClientID != "file-id" || cfg.ClientSecret != "file-secret" {
		t.Fatalf("file values not loaded: %+v", cfg)
	}
	t.Setenv("FATSECRET_CLIENT_ID", "env-id")
	if cfg := Resolve(); cfg.ClientID != "env-id" || cfg.ClientSecret != "file-secret" {
		t.Fatalf("env should win over file: %+v", cfg)
	}
	if err := Save(map[string]string{"clientSecret": ""}); err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("FATSECRET_CLIENT_ID")
	if cfg := Resolve(); cfg.ClientSecret != "" || cfg.ClientID != "file-id" {
		t.Fatalf("empty save should delete key: %+v", cfg)
	}
}
