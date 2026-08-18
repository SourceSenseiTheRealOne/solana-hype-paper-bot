package httpclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/adapters/httpclient"
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
