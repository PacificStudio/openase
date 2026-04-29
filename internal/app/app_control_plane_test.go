package app

import (
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
)

func TestAgentPlatformAPIURLPrefersOpenASEBaseURL(t *testing.T) {
	t.Setenv("OPENASE_BASE_URL", "https://control.example.com")
	app := &App{config: config.Config{Server: config.ServerConfig{Host: "127.0.0.1", Port: 19836}}}
	if got := app.agentPlatformAPIURL(); got != "https://control.example.com/api/v1/platform" {
		t.Fatalf("agentPlatformAPIURL() = %q", got)
	}
}
