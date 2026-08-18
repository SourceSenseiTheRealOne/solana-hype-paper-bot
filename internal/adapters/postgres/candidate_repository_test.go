package postgres_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/ent"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/ent/botstate"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/ent/candidate"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/adapters/postgres"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/domain"
)

func TestCandidateRepositoryPersistsIdempotentlyAndRoundTripsDiscoveryState(t *testing.T) {
	client := openTestEntClient(t)
	defer func() { _ = client.Close() }()

	repository, err := postgres.NewCandidateRepository(client)
	if err != nil {
		t.Fatalf("NewCandidateRepository() error = %v", err)
	}

	unique := fmt.Sprintf("candidate-repository-%d", time.Now().UTC().UnixNano())
	pool := domain.DiscoveredPool{
		Source:      domain.SourceGeckoTerminal,
		Network:     domain.NetworkSolana,
		MintAddress: unique + "-mint",
		PoolAddress: unique + "-pool",
		CreatedAt:   time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	}

	inserted, err := repository.InsertDiscovered(context.Background(), []domain.DiscoveredPool{pool})
	if err != nil {
		t.Fatalf("first InsertDiscovered() error = %v", err)
	}
	if got, want := inserted, 1; got != want {
		t.Fatalf("first inserted = %d, want %d", got, want)
	}

	inserted, err = repository.InsertDiscovered(context.Background(), []domain.DiscoveredPool{pool})
	if err != nil {
		t.Fatalf("second InsertDiscovered() error = %v", err)
	}
	if got, want := inserted, 0; got != want {
		t.Fatalf("second inserted = %d, want %d", got, want)
	}

	stored, err := client.Candidate.Query().Where(candidate.PoolAddressEQ(pool.PoolAddress)).Only(context.Background())
	if err != nil {
		t.Fatalf("query stored candidate: %v", err)
	}
	if got, want := stored.DiscoveredAt, pool.CreatedAt; !got.Equal(want) {
		t.Fatalf("stored discovered_at = %s, want %s", got, want)
	}

	mark := domain.Watermark{CreatedAt: pool.CreatedAt, PoolAddress: pool.PoolAddress}
	if err := repository.SaveWatermark(context.Background(), unique, mark); err != nil {
		t.Fatalf("SaveWatermark() error = %v", err)
	}
	loaded, err := repository.LoadWatermark(context.Background(), unique)
	if err != nil {
		t.Fatalf("LoadWatermark() error = %v", err)
	}
	if loaded != mark {
		t.Fatalf("loaded watermark = %#v, want %#v", loaded, mark)
	}

	if err := repository.RecordCoverageGap(context.Background(), unique, 3, "page budget exhausted"); err != nil {
		t.Fatalf("RecordCoverageGap() error = %v", err)
	}
	coverage, err := client.BotState.Query().Where(botstate.StateKeyEQ("discovery:" + unique + ":coverage-gap")).Only(context.Background())
	if err != nil {
		t.Fatalf("query coverage gap: %v", err)
	}
	if got, want := coverage.Value["page"], float64(3); got != want {
		t.Fatalf("coverage page = %#v, want %#v", got, want)
	}
	if got, want := coverage.Value["reason"], "page budget exhausted"; got != want {
		t.Fatalf("coverage reason = %#v, want %#v", got, want)
	}
}

func TestCandidateRepositoryRejectsMixedInvalidBatchWithoutPersisting(t *testing.T) {
	client := openTestEntClient(t)
	defer func() { _ = client.Close() }()

	repository, err := postgres.NewCandidateRepository(client)
	if err != nil {
		t.Fatalf("NewCandidateRepository() error = %v", err)
	}

	unique := fmt.Sprintf("candidate-repository-invalid-%d", time.Now().UTC().UnixNano())
	valid := domain.DiscoveredPool{
		Source:      domain.SourceGeckoTerminal,
		Network:     domain.NetworkSolana,
		MintAddress: unique + "-mint",
		PoolAddress: unique + "-pool",
		CreatedAt:   time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	}
	invalid := valid
	invalid.MintAddress = ""

	if _, err := repository.InsertDiscovered(context.Background(), []domain.DiscoveredPool{valid, invalid}); err == nil {
		t.Fatal("InsertDiscovered() accepted a mixed invalid batch")
	}
	count, err := client.Candidate.Query().Where(candidate.PoolAddressEQ(valid.PoolAddress)).Count(context.Background())
	if err != nil {
		t.Fatalf("count valid candidate after rejected batch: %v", err)
	}
	if got, want := count, 0; got != want {
		t.Fatalf("stored candidates after rejected batch = %d, want %d", got, want)
	}
}

func openTestEntClient(t *testing.T) *ent.Client {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; run this contract against a reset local Supabase database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := postgres.OpenEnt(ctx, databaseURL)
	if err != nil {
		t.Fatalf("OpenEnt() error = %v", err)
	}
	return client
}
