package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	SourceDexScreener   = "dexscreener"
	SourceGeckoTerminal = "geckoterminal"
	NetworkSolana       = "solana"
)

// DiscoveredPool is normalized public pool evidence before any risk filtering.
type DiscoveredPool struct {
	Source      string
	Network     string
	MintAddress string
	PoolAddress string
	CreatedAt   time.Time
}

func (pool DiscoveredPool) Validate() error {
	if strings.TrimSpace(pool.Source) == "" {
		return errors.New("pool source is required")
	}
	if strings.TrimSpace(pool.Network) == "" {
		return errors.New("pool network is required")
	}
	if strings.TrimSpace(pool.MintAddress) == "" {
		return errors.New("pool mint address is required")
	}
	if strings.TrimSpace(pool.PoolAddress) == "" {
		return errors.New("pool address is required")
	}
	if pool.CreatedAt.IsZero() {
		return errors.New("pool created time is required")
	}
	return nil
}

func (pool DiscoveredPool) Identity() string {
	return fmt.Sprintf("%s:%s:%s", pool.Network, pool.MintAddress, pool.PoolAddress)
}

// TokenHint is a bounded public signal from DexScreener, not a trade candidate.
type TokenHint struct {
	Source      string
	Network     string
	MintAddress string
	URL         string
}

func (hint TokenHint) Validate() error {
	if strings.TrimSpace(hint.Source) == "" || strings.TrimSpace(hint.Network) == "" || strings.TrimSpace(hint.MintAddress) == "" {
		return errors.New("token hint source, network, and mint address are required")
	}
	return nil
}

// Watermark records the oldest persisted pool from a complete scan.
type Watermark struct {
	CreatedAt   time.Time
	PoolAddress string
}

func (watermark Watermark) IsZero() bool {
	return watermark.CreatedAt.IsZero() || strings.TrimSpace(watermark.PoolAddress) == ""
}

func (watermark Watermark) Matches(pool DiscoveredPool) bool {
	return watermark.CreatedAt.Equal(pool.CreatedAt) && watermark.PoolAddress == pool.PoolAddress
}
