package config_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/config"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/domain"
)

func TestValidateRejectsUnsafeStrategyLimits(t *testing.T) {
	base := validConfig()

	tests := []struct {
		name   string
		mutate func(*config.Config)
	}{
		{"zero notional", func(c *config.Config) { c.PaperTradeUSD = domain.USD{} }},
		{"too many open positions", func(c *config.Config) { c.MaxOpenPositions = 4 }},
		{"too many daily trades", func(c *config.Config) { c.MaxDailyTrades = 31 }},
		{"zero take profit", func(c *config.Config) { c.TakeProfitBPS = 0 }},
		{"zero stop loss", func(c *config.Config) { c.StopLossBPS = 0 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() accepted unsafe strategy configuration")
			}
		})
	}
}

func TestProviderReadinessIsScopedAndConfigurationIsRedacted(t *testing.T) {
	cfg := validConfig()
	cfg.TwitterAPIKey = "twitter-secret-value"
	cfg.HermesAPIKey = "hermes-secret-value"
	cfg.JupiterAPIKey = "jupiter-secret-value"

	cfg.TwitterAPIKey = ""
	if err := cfg.ProviderReady(config.ProviderTwitterAPI); err == nil {
		t.Fatal("Twitter provider unexpectedly reported ready")
	}
	if err := cfg.ProviderReady(config.ProviderJupiter); err != nil {
		t.Fatalf("Jupiter readiness was affected by missing Twitter key: %v", err)
	}

	for _, rendered := range []string{fmt.Sprint(cfg), cfg.SafeHealth().String()} {
		for _, secret := range []string{"hermes-secret-value", "jupiter-secret-value"} {
			if contains(rendered, secret) {
				t.Fatalf("safe configuration rendering leaked %q", secret)
			}
		}
	}
}

func validConfig() config.Config {
	return config.Config{
		HTTPAddr:             ":8080",
		DatabaseURL:          "postgres://redacted",
		PaperTradeUSD:        domain.USD{Micros: 10_000_000},
		MaxOpenPositions:     3,
		MaxDailyTrades:       30,
		TakeProfitBPS:        3000,
		StopLossBPS:          1500,
		MaxHoldDuration:      time.Hour,
		DiscoveryInterval:    time.Minute,
		PositionMarkInterval: 30 * time.Second,
	}
}

func contains(value, fragment string) bool {
	return len(fragment) > 0 && len(value) >= len(fragment) && stringContains(value, fragment)
}

func stringContains(value, fragment string) bool {
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
