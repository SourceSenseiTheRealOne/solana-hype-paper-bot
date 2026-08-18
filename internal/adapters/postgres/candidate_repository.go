package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/ent"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/ent/botstate"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/ent/candidate"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/domain"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/ports"
)

const discoveryStatePrefix = "discovery:"

type CandidateRepository struct {
	client *ent.Client
}

func NewCandidateRepository(client *ent.Client) (*CandidateRepository, error) {
	if client == nil {
		return nil, errors.New("candidate repository requires an Ent client")
	}
	return &CandidateRepository{client: client}, nil
}

func (repository *CandidateRepository) InsertDiscovered(ctx context.Context, pools []domain.DiscoveredPool) (int, error) {
	unique, err := validatedUniquePools(pools)
	if err != nil {
		return 0, err
	}
	if len(unique) == 0 {
		return 0, nil
	}

	tx, err := repository.client.Tx(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin candidate transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	inserted := 0
	for _, pool := range unique {
		exists, err := tx.Candidate.Query().Where(
			candidate.NetworkEQ(pool.Network),
			candidate.MintAddressEQ(pool.MintAddress),
			candidate.PoolAddressEQ(pool.PoolAddress),
		).Exist(ctx)
		if err != nil {
			return 0, fmt.Errorf("check candidate identity: %w", err)
		}
		if exists {
			continue
		}

		if err := tx.Candidate.Create().
			SetNetwork(pool.Network).
			SetMintAddress(pool.MintAddress).
			SetPoolAddress(pool.PoolAddress).
			SetDiscoveredAt(pool.CreatedAt).
			OnConflictColumns(candidate.FieldNetwork, candidate.FieldMintAddress, candidate.FieldPoolAddress).
			DoNothing().
			Exec(ctx); err != nil {
			return 0, fmt.Errorf("insert discovered candidate: %w", err)
		}
		inserted++
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit candidate transaction: %w", err)
	}
	return inserted, nil
}

func (repository *CandidateRepository) LoadWatermark(ctx context.Context, source string) (domain.Watermark, error) {
	key, err := discoveryStateKey(source, "watermark")
	if err != nil {
		return domain.Watermark{}, err
	}
	state, err := repository.client.BotState.Query().Where(botstate.StateKeyEQ(key)).Only(ctx)
	if ent.IsNotFound(err) {
		return domain.Watermark{}, nil
	}
	if err != nil {
		return domain.Watermark{}, fmt.Errorf("load discovery watermark: %w", err)
	}

	mark, err := watermarkFromState(state.Value)
	if err != nil {
		return domain.Watermark{}, fmt.Errorf("decode discovery watermark: %w", err)
	}
	return mark, nil
}

func (repository *CandidateRepository) SaveWatermark(ctx context.Context, source string, mark domain.Watermark) error {
	key, err := discoveryStateKey(source, "watermark")
	if err != nil {
		return err
	}
	if mark.IsZero() {
		return errors.New("discovery watermark requires pool address and creation time")
	}
	value := map[string]any{
		"created_at":   mark.CreatedAt.UTC().Format(time.RFC3339Nano),
		"pool_address": mark.PoolAddress,
	}
	if err := repository.client.BotState.Create().
		SetStateKey(key).
		SetValue(value).
		OnConflictColumns(botstate.FieldStateKey).
		UpdateNewValues().
		Exec(ctx); err != nil {
		return fmt.Errorf("save discovery watermark: %w", err)
	}
	return nil
}

func (repository *CandidateRepository) RecordCoverageGap(ctx context.Context, source string, page int, reason string) error {
	key, err := discoveryStateKey(source, "coverage-gap")
	if err != nil {
		return err
	}
	if page < 1 || strings.TrimSpace(reason) == "" {
		return errors.New("coverage gap requires a positive page and reason")
	}
	value := map[string]any{
		"page":        page,
		"reason":      strings.TrimSpace(reason),
		"recorded_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := repository.client.BotState.Create().
		SetStateKey(key).
		SetValue(value).
		OnConflictColumns(botstate.FieldStateKey).
		UpdateNewValues().
		Exec(ctx); err != nil {
		return fmt.Errorf("save discovery coverage gap: %w", err)
	}
	return nil
}

func validatedUniquePools(pools []domain.DiscoveredPool) ([]domain.DiscoveredPool, error) {
	unique := make([]domain.DiscoveredPool, 0, len(pools))
	seen := make(map[string]struct{}, len(pools))
	for _, pool := range pools {
		if err := pool.Validate(); err != nil {
			return nil, fmt.Errorf("validate discovered pool: %w", err)
		}
		identity := pool.Identity()
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		unique = append(unique, pool)
	}
	return unique, nil
}

func discoveryStateKey(source, suffix string) (string, error) {
	source = strings.TrimSpace(source)
	if source == "" || strings.ContainsAny(source, ":\t\n\r") {
		return "", errors.New("discovery source must be a non-empty key segment")
	}
	return discoveryStatePrefix + source + ":" + suffix, nil
}

func watermarkFromState(value map[string]any) (domain.Watermark, error) {
	createdAtRaw, ok := value["created_at"].(string)
	if !ok {
		return domain.Watermark{}, errors.New("watermark created_at is missing")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return domain.Watermark{}, fmt.Errorf("parse watermark created_at: %w", err)
	}
	poolAddress, ok := value["pool_address"].(string)
	if !ok || strings.TrimSpace(poolAddress) == "" {
		return domain.Watermark{}, errors.New("watermark pool_address is missing")
	}
	return domain.Watermark{CreatedAt: createdAt.UTC(), PoolAddress: poolAddress}, nil
}

var _ ports.CandidateRepository = (*CandidateRepository)(nil)
