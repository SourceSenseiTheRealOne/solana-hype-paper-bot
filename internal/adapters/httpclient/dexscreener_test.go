package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/adapters/httpclient"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/domain"
)

func TestDexScreenerFetchLatestSolanaTokenHintsFiltersOtherNetworks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got, want := request.URL.Path, "/token-profiles/latest/v1"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		_, _ = writer.Write([]byte(`[
			{"chainId":"solana","tokenAddress":"solana-mint","url":"https://dexscreener.com/solana/solana-mint"},
			{"chainId":"ethereum","tokenAddress":"eth-mint","url":"https://dexscreener.com/ethereum/eth-mint"}
		]`))
	}))
	defer server.Close()

	hints, err := httpclient.NewDexScreener(newBoundedClient(t, server.URL)).FetchLatestSolanaTokenHints(context.Background())
	if err != nil {
		t.Fatalf("FetchLatestSolanaTokenHints() error = %v", err)
	}
	if got, want := len(hints), 1; got != want {
		t.Fatalf("hint count = %d, want %d", got, want)
	}
	if got, want := hints[0].MintAddress, "solana-mint"; got != want {
		t.Fatalf("mint = %q, want %q", got, want)
	}
}

func TestDexScreenerCapsLatestSolanaTokenHints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		var payload strings.Builder
		payload.WriteByte('[')
		for index := 0; index < 21; index++ {
			if index > 0 {
				payload.WriteByte(',')
			}
			_, _ = fmt.Fprintf(&payload, `{"chainId":"solana","tokenAddress":"mint-%02d"}`, index)
		}
		payload.WriteByte(']')
		_, _ = writer.Write([]byte(payload.String()))
	}))
	defer server.Close()

	hints, err := httpclient.NewDexScreener(newBoundedClient(t, server.URL)).FetchLatestSolanaTokenHints(context.Background())
	if err != nil {
		t.Fatalf("FetchLatestSolanaTokenHints() error = %v", err)
	}
	if got, want := len(hints), 20; got != want {
		t.Fatalf("hint count = %d, want bounded %d", got, want)
	}
}

func TestDexScreenerFetchNewPoolsResolvesNewestSolanaPairForLatestProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/token-profiles/latest/v1":
			_, _ = writer.Write([]byte(`[{"chainId":"solana","tokenAddress":"solana-mint"}]`))
		case "/token-pairs/v1/solana/solana-mint":
			_, _ = writer.Write([]byte(`[
				{"chainId":"solana","pairAddress":"older-pool","pairCreatedAt":1787227200000},
				{"chainId":"solana","pairAddress":"newest-pool","pairCreatedAt":1787227260000},
				{"chainId":"ethereum","pairAddress":"wrong-network","pairCreatedAt":1787227320000}
			]`))
		default:
			t.Errorf("unexpected path %q", request.URL.Path)
		}
	}))
	defer server.Close()

	page, err := httpclient.NewDexScreener(newBoundedClient(t, server.URL)).FetchNewPools(context.Background(), 1)
	if err != nil {
		t.Fatalf("FetchNewPools() error = %v", err)
	}
	if got, want := len(page.Pools), 1; got != want {
		t.Fatalf("pool count = %d, want %d", got, want)
	}
	pool := page.Pools[0]
	if pool.Source != domain.SourceDexScreener || pool.Network != domain.NetworkSolana || pool.MintAddress != "solana-mint" || pool.PoolAddress != "newest-pool" || !pool.CreatedAt.Equal(time.UnixMilli(1787227260000).UTC()) {
		t.Fatalf("pool = %#v, want newest normalized DexScreener Solana pair", pool)
	}
}

func TestDexScreenerRejectsProfileWithoutTokenAddress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`[{"chainId":"solana"}]`))
	}))
	defer server.Close()

	_, err := httpclient.NewDexScreener(newBoundedClient(t, server.URL)).FetchLatestSolanaTokenHints(context.Background())
	if err == nil {
		t.Fatal("FetchLatestSolanaTokenHints() accepted a profile without tokenAddress")
	}
}

func TestDexScreenerFetchesNormalizedSolanaPoolMarketEvidence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got, want := request.URL.Path, "/latest/dex/pairs/solana/pool-address"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		_, _ = writer.Write([]byte(`{"pairs":[{"chainId":"solana","pairAddress":"pool-address","liquidity":{"usd":20000.25},"txns":{"m5":{"buys":13,"sells":7}},"volume":{"m5":3000.0375},"priceChange":{"m5":2.345}}]}`))
	}))
	defer server.Close()
	pool := domain.DiscoveredPool{Source: domain.SourceDexScreener, Network: domain.NetworkSolana, MintAddress: "mint-address", PoolAddress: "pool-address", CreatedAt: time.Now().UTC().Add(-5 * time.Minute)}

	snapshot, err := httpclient.NewDexScreener(newBoundedClient(t, server.URL)).Fetch(context.Background(), pool)
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if got, want := snapshot.LiquidityUSD.Micros, int64(20_000_250_000); got != want {
		t.Fatalf("liquidity micros = %d, want %d", got, want)
	}
	if got, want := snapshot.FiveMinuteTransactions, 20; got != want {
		t.Fatalf("five minute transactions = %d, want %d", got, want)
	}
	if snapshot.FiveMinuteBuys != 13 || snapshot.FiveMinuteSells != 7 || snapshot.FiveMinuteVolumeUSD.Micros != 3_000_037_500 || snapshot.FiveMinutePriceChangeBPS != 235 {
		t.Fatalf("five-minute market evidence = %#v", snapshot)
	}
	if snapshot.ObservedAt.IsZero() {
		t.Fatal("observed at is zero")
	}
}

func TestDexScreenerRejectsIncompleteFiveMinuteMarketEvidence(t *testing.T) {
	tests := []struct{ name, market string }{
		{name: "missing buys", market: `"txns":{"m5":{"sells":7}},"volume":{"m5":750},"priceChange":{"m5":2}`},
		{name: "missing sells", market: `"txns":{"m5":{"buys":13}},"volume":{"m5":750},"priceChange":{"m5":2}`},
		{name: "missing volume", market: `"txns":{"m5":{"buys":13,"sells":7}},"volume":{},"priceChange":{"m5":2}`},
		{name: "missing price change", market: `"txns":{"m5":{"buys":13,"sells":7}},"volume":{"m5":750},"priceChange":{}`},
		{name: "negative buys", market: `"txns":{"m5":{"buys":-1,"sells":7}},"volume":{"m5":750},"priceChange":{"m5":2}`},
		{name: "negative volume", market: `"txns":{"m5":{"buys":13,"sells":7}},"volume":{"m5":-1},"priceChange":{"m5":2}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = fmt.Fprintf(writer, `{"pairs":[{"chainId":"solana","pairAddress":"pool-address","liquidity":{"usd":5000},%s}]}`, test.market)
			}))
			defer server.Close()
			pool := domain.DiscoveredPool{Source: domain.SourceDexScreener, Network: domain.NetworkSolana, MintAddress: "mint", PoolAddress: "pool-address", CreatedAt: time.Now().UTC()}
			if _, err := httpclient.NewDexScreener(newBoundedClient(t, server.URL)).Fetch(context.Background(), pool); err == nil {
				t.Fatal("Fetch() accepted incomplete five-minute evidence")
			}
		})
	}
}
