package cmd

import (
	"os"
	"testing"
)

// keepEnv saves and restores env vars used by ensureAPIKey.
func keepEnv(t *testing.T) func() {
	t.Helper()
	orig := map[string]string{
		"ELEVENLABS_API_KEY": os.Getenv("ELEVENLABS_API_KEY"),
		"SAG_API_KEY":        os.Getenv("SAG_API_KEY"),
	}
	return func() {
		_ = os.Setenv("ELEVENLABS_API_KEY", orig["ELEVENLABS_API_KEY"])
		_ = os.Setenv("SAG_API_KEY", orig["SAG_API_KEY"])
	}
}

func TestEnsureAPIKeyPrefersCLIValue(t *testing.T) {
	defer keepEnv(t)()
	cfg.APIKey = "cli-key"
	_ = os.Unsetenv("ELEVENLABS_API_KEY")
	_ = os.Unsetenv("SAG_API_KEY")

	if err := ensureAPIKey(); err != nil {
		t.Fatalf("ensureAPIKey error: %v", err)
	}
	if cfg.APIKey != "cli-key" {
		t.Fatalf("expected CLI cfg API key to win, got %q", cfg.APIKey)
	}
}

func TestEnsureAPIKeyFallsBackToEnvOrder(t *testing.T) {
	defer keepEnv(t)()
	cfg.APIKey = ""
	_ = os.Setenv("ELEVENLABS_API_KEY", "env-key")
	_ = os.Setenv("SAG_API_KEY", "sag-key")

	if err := ensureAPIKey(); err != nil {
		t.Fatalf("ensureAPIKey error: %v", err)
	}
	if cfg.APIKey != "env-key" {
		t.Fatalf("expected ELEVENLABS_API_KEY to be used, got %q", cfg.APIKey)
	}

	// Clear primary env to ensure SAG_API_KEY is used next.
	cfg.APIKey = ""
	_ = os.Unsetenv("ELEVENLABS_API_KEY")
	if err := ensureAPIKey(); err != nil {
		t.Fatalf("ensureAPIKey error: %v", err)
	}
	if cfg.APIKey != "sag-key" {
		t.Fatalf("expected SAG_API_KEY to be used, got %q", cfg.APIKey)
	}
}

func TestEnsureAPIKeyMissing(t *testing.T) {
	defer keepEnv(t)()
	cfg.APIKey = ""
	_ = os.Unsetenv("ELEVENLABS_API_KEY")
	_ = os.Unsetenv("SAG_API_KEY")

	if err := ensureAPIKey(); err == nil {
		t.Fatal("expected error when API key missing")
	}
}
