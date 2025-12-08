package cmd

import (
	"fmt"
	"os"
)

func ensureAPIKey() error {
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("ELEVENLABS_API_KEY")
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("SAG_API_KEY")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("missing ElevenLabs API key (set --api-key or ELEVENLABS_API_KEY)")
	}
	return nil
}
