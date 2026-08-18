package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/domain"
	"github.com/caarlos0/env/v11"
)

type Provider string

const (
	ProviderTwitterAPI Provider = "twitterapiio"
	ProviderHermes     Provider = "hermes"
	ProviderJupiter    Provider = "jupiter"
)

type Config struct {
	AppEnv               string        `env:"APP_ENV" envDefault:"local"`
	HTTPAddr             string        `env:"HTTP_ADDR" envDefault:":8080"`
	DatabaseURL          string        `env:"DATABASE_URL"`
	SolanaRPCURL         string        `env:"SOLANA_RPC_URL" envDefault:"https://api.mainnet-beta.solana.com"`
	TwitterAPIKey        string        `env:"TWITTERAPIIO_API_KEY"`
	HermesAPIURL         string        `env:"HERMES_API_URL" envDefault:"http://host.docker.internal:8642/v1"`
	HermesAPIKey         string        `env:"HERMES_API_KEY"`
	JupiterAPIURL        string        `env:"JUPITER_API_URL" envDefault:"https://api.jup.ag"`
	JupiterAPIKey        string        `env:"JUPITER_API_KEY"`
	DiscoveryInterval    time.Duration `env:"DISCOVERY_INTERVAL" envDefault:"60s"`
	PositionMarkInterval time.Duration `env:"POSITION_MARK_INTERVAL" envDefault:"30s"`
	PaperTradeUSD        domain.USD    `env:"PAPER_TRADE_USD" envDefault:"10"`
	MaxOpenPositions     int           `env:"MAX_OPEN_POSITIONS" envDefault:"3"`
	MaxDailyTrades       int           `env:"MAX_DAILY_TRADES" envDefault:"30"`
	TakeProfitBPS        int64         `env:"TAKE_PROFIT_BPS" envDefault:"3000"`
	StopLossBPS          int64         `env:"STOP_LOSS_BPS" envDefault:"1500"`
	MaxHoldDuration      time.Duration `env:"MAX_HOLD_DURATION" envDefault:"60m"`
	LogDir               string        `env:"LOG_DIR" envDefault:"var/log"`
	ReportDir            string        `env:"REPORT_DIR" envDefault:"var/reports"`
}

type Health struct{ ReadyProviders []Provider }

func (Health) String() string { return "configuration health (redacted)" }
func (Config) String() string { return "configuration (redacted)" }

func Load() (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse configuration: %w", err)
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if c.PaperTradeUSD.Micros <= 0 {
		return errors.New("PAPER_TRADE_USD must be positive")
	}
	if c.MaxOpenPositions < 1 || c.MaxOpenPositions > 3 {
		return errors.New("MAX_OPEN_POSITIONS must be between 1 and 3")
	}
	if c.MaxDailyTrades < 1 || c.MaxDailyTrades > 30 {
		return errors.New("MAX_DAILY_TRADES must be between 1 and 30")
	}
	if c.TakeProfitBPS <= 0 || c.StopLossBPS <= 0 {
		return errors.New("take-profit and stop-loss basis points must be positive")
	}
	if c.MaxHoldDuration <= 0 || c.DiscoveryInterval <= 0 || c.PositionMarkInterval <= 0 {
		return errors.New("durations must be positive")
	}
	return nil
}

func (c Config) ProviderReady(provider Provider) error {
	var key string
	switch provider {
	case ProviderTwitterAPI:
		key = c.TwitterAPIKey
	case ProviderHermes:
		key = c.HermesAPIKey
	case ProviderJupiter:
		key = c.JupiterAPIKey
	default:
		return fmt.Errorf("unknown provider %q", provider)
	}
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("%s provider is not configured", provider)
	}
	return nil
}

func (c Config) SafeHealth() Health {
	providers := make([]Provider, 0, 3)
	for _, provider := range []Provider{ProviderTwitterAPI, ProviderHermes, ProviderJupiter} {
		if c.ProviderReady(provider) == nil {
			providers = append(providers, provider)
		}
	}
	return Health{ReadyProviders: providers}
}
