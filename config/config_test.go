package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigReadsDotenv(t *testing.T) {
	dir := t.TempDir()
	env := "BOT_TOKEN=test_token_123\n" +
		"DATABASE_URL=postgres://u:p@localhost:5432/db?sslmode=disable\n" +
		"ADMIN_IDS=11,22\n" +
		"DEVELOPER_NAME=tester\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	// Clear keys so the test is deterministic regardless of the host shell env.
	for _, k := range []string{"BOT_TOKEN", "DATABASE_URL", "ADMIN_IDS", "DEVELOPER_NAME"} {
		if v, ok := os.LookupEnv(k); ok {
			t.Setenv(k, v) // restore on cleanup
		}
		os.Unsetenv(k)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.BotToken != "test_token_123" {
		t.Errorf("BotToken = %q; want test_token_123", cfg.BotToken)
	}
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/db?sslmode=disable" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if len(cfg.AdminIDs) != 2 || cfg.AdminIDs[0] != 11 || cfg.AdminIDs[1] != 22 {
		t.Errorf("AdminIDs = %v; want [11 22]", cfg.AdminIDs)
	}
	if cfg.DeveloperName != "tester" {
		t.Errorf("DeveloperName = %q; want tester", cfg.DeveloperName)
	}
}

func TestLoadConfigEnvOverridesDotenv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("BOT_TOKEN=from_file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	t.Setenv("BOT_TOKEN", "from_env")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.BotToken != "from_env" {
		t.Errorf("BotToken = %q; want from_env (real env must win)", cfg.BotToken)
	}
}
