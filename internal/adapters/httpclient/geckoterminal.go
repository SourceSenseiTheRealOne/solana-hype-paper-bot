package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/domain"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/ports"
)

type GeckoTerminal struct {
	client *Client
}

func NewGeckoTerminal(client *Client) *GeckoTerminal {
	return &GeckoTerminal{client: client}
}

func (provider *GeckoTerminal) FetchNewPools(ctx context.Context, page int) (ports.PoolPage, error) {
	if page < 1 {
		return ports.PoolPage{}, errors.New("GeckoTerminal page must be at least one")
	}
	var response geckoPoolsResponse
	if err := provider.client.GetJSON(ctx, "/api/v2/networks/solana/new_pools", url.Values{"page": []string{strconv.Itoa(page)}}, &response); err != nil {
		return ports.PoolPage{}, fmt.Errorf("fetch GeckoTerminal new pools: %w", err)
	}

	tokens := make(map[string]string, len(response.Included))
	for _, token := range response.Included {
		if token.Type != "token" || strings.TrimSpace(token.ID) == "" || strings.TrimSpace(token.Attributes.Address) == "" {
			continue
		}
		tokens[token.ID] = token.Attributes.Address
	}

	seen := make(map[string]struct{}, len(response.Data))
	pools := make([]domain.DiscoveredPool, 0, len(response.Data))
	for _, item := range response.Data {
		pool, err := parseGeckoPool(item, tokens)
		if err != nil {
			return ports.PoolPage{}, err
		}
		if _, exists := seen[pool.PoolAddress]; exists {
			continue
		}
		seen[pool.PoolAddress] = struct{}{}
		pools = append(pools, pool)
	}
	sort.Slice(pools, func(left, right int) bool {
		if pools[left].CreatedAt.Equal(pools[right].CreatedAt) {
			return pools[left].PoolAddress < pools[right].PoolAddress
		}
		return pools[left].CreatedAt.After(pools[right].CreatedAt)
	})

	nextPage, err := parseNextPage(response.Links.Next)
	if err != nil {
		return ports.PoolPage{}, err
	}
	return ports.PoolPage{Pools: pools, NextPage: nextPage}, nil
}

func parseGeckoPool(item geckoPool, tokens map[string]string) (domain.DiscoveredPool, error) {
	if item.Type != "pool" {
		return domain.DiscoveredPool{}, errors.New("GeckoTerminal response contains a non-pool data item")
	}
	mint, ok := tokens[item.Relationships.BaseToken.Data.ID]
	if !ok {
		return domain.DiscoveredPool{}, errors.New("GeckoTerminal pool base-token relationship was not included")
	}
	createdAt, err := time.Parse(time.RFC3339, item.Attributes.PoolCreatedAt)
	if err != nil {
		return domain.DiscoveredPool{}, fmt.Errorf("parse GeckoTerminal pool creation time: %w", err)
	}
	pool := domain.DiscoveredPool{Source: domain.SourceGeckoTerminal, Network: domain.NetworkSolana, MintAddress: mint, PoolAddress: item.Attributes.Address, CreatedAt: createdAt.UTC()}
	if err := pool.Validate(); err != nil {
		return domain.DiscoveredPool{}, fmt.Errorf("validate GeckoTerminal pool: %w", err)
	}
	return pool, nil
}

func parseNextPage(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return 0, fmt.Errorf("parse GeckoTerminal next link: %w", err)
	}
	page, err := strconv.Atoi(parsed.Query().Get("page"))
	if err != nil || page < 1 {
		return 0, errors.New("GeckoTerminal next link has no valid page")
	}
	return page, nil
}

type geckoPoolsResponse struct {
	Data     []geckoPool    `json:"data"`
	Included []geckoToken   `json:"included"`
	Links    geckoPageLinks `json:"links"`
}

type geckoPool struct {
	Type       string `json:"type"`
	Attributes struct {
		Address       string `json:"address"`
		PoolCreatedAt string `json:"pool_created_at"`
	} `json:"attributes"`
	Relationships struct {
		BaseToken struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		} `json:"base_token"`
	} `json:"relationships"`
}

type geckoToken struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes struct {
		Address string `json:"address"`
	} `json:"attributes"`
}

type geckoPageLinks struct {
	Next string `json:"next"`
}

var _ ports.PoolDiscovery = (*GeckoTerminal)(nil)
