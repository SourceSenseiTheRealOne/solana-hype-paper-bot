package httpclient

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/domain"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/ports"
)

type DexScreener struct {
	client *Client
}

func NewDexScreener(client *Client) *DexScreener {
	return &DexScreener{client: client}
}

func (provider *DexScreener) FetchLatestSolanaTokenHints(ctx context.Context) ([]domain.TokenHint, error) {
	var profiles []dexScreenerTokenProfile
	if err := provider.client.GetJSON(ctx, "/token-profiles/latest/v1", nil, &profiles); err != nil {
		return nil, fmt.Errorf("fetch DexScreener token profiles: %w", err)
	}

	seen := make(map[string]struct{}, len(profiles))
	hints := make([]domain.TokenHint, 0, len(profiles))
	for _, profile := range profiles {
		if profile.ChainID != domain.NetworkSolana {
			continue
		}
		hint := domain.TokenHint{Source: domain.SourceDexScreener, Network: domain.NetworkSolana, MintAddress: profile.TokenAddress, URL: profile.URL}
		if err := hint.Validate(); err != nil {
			return nil, fmt.Errorf("validate DexScreener token profile: %w", err)
		}
		if _, exists := seen[hint.MintAddress]; exists {
			continue
		}
		seen[hint.MintAddress] = struct{}{}
		hints = append(hints, hint)
	}
	sort.Slice(hints, func(left, right int) bool { return hints[left].MintAddress < hints[right].MintAddress })
	return hints, nil
}

type dexScreenerTokenProfile struct {
	ChainID      string `json:"chainId"`
	TokenAddress string `json:"tokenAddress"`
	URL          string `json:"url"`
}

var _ ports.TokenHintDiscovery = (*DexScreener)(nil)

func isSolana(chainID string) bool {
	return strings.EqualFold(strings.TrimSpace(chainID), domain.NetworkSolana)
}
