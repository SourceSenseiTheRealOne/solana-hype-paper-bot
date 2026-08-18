package ports

import (
	"context"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/domain"
)

type PoolPage struct {
	Pools    []domain.DiscoveredPool
	NextPage int
}

// PoolDiscovery returns public pools ordered newest-first by the provider.
type PoolDiscovery interface {
	FetchNewPools(ctx context.Context, page int) (PoolPage, error)
}

// TokenHintDiscovery returns bounded public token hints, not executable trade data.
type TokenHintDiscovery interface {
	FetchLatestSolanaTokenHints(ctx context.Context) ([]domain.TokenHint, error)
}
