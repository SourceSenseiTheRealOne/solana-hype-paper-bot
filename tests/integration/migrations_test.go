package integration_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationContract(t *testing.T) {
	t.Parallel()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; run this contract against a reset local Supabase database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect migration test database: %v", err)
	}
	defer pool.Close()

	for _, table := range []string{
		"candidates",
		"candidate_snapshots",
		"social_snapshots",
		"verdicts",
		"trade_decisions",
		"paper_positions",
		"position_marks",
		"position_events",
		"daily_results",
		"bot_states",
		"api_usages",
	} {
		assertTableExists(t, ctx, pool, table)
	}

	assertUniqueIndexWithColumns(t, ctx, pool, "candidates", "network", "mint_address", "pool_address")
	assertUniqueIndexWithColumns(t, ctx, pool, "trade_decisions", "idempotency_key")
	assertUniqueIndexWithColumns(t, ctx, pool, "daily_results", "utc_date", "strategy_version")
	assertCheckContains(t, ctx, pool, "daily_results", "daily_admitted_count", ">=", "0")
	assertCheckContains(t, ctx, pool, "daily_results", "daily_admitted_count", "<=", "30")
	assertCheckContains(t, ctx, pool, "paper_positions", "notional_micros", ">", "0")
	assertCheckContains(
		t,
		ctx,
		pool,
		"paper_positions",
		"PENDING",
		"OPEN",
		"CLOSING",
		"CLOSED",
		"UNSELLABLE",
	)
}

func assertTableExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string) {
	t.Helper()

	var exists bool
	if err := pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", "public."+table).Scan(&exists); err != nil {
		t.Fatalf("look up table %q: %v", table, err)
	}
	if !exists {
		t.Errorf("required table %q does not exist", table)
	}
}

func assertUniqueIndexWithColumns(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	table string,
	columns ...string,
) {
	t.Helper()

	rows, err := pool.Query(
		ctx,
		`SELECT indexdef
		 FROM pg_indexes
		 WHERE schemaname = 'public' AND tablename = $1 AND indexdef LIKE 'CREATE UNIQUE INDEX%'
		 ORDER BY indexname`,
		table,
	)
	if err != nil {
		t.Fatalf("read unique indexes for %q: %v", table, err)
		return
	}
	defer rows.Close()

	var definitions []string
	for rows.Next() {
		var definition string
		if err := rows.Scan(&definition); err != nil {
			t.Fatalf("scan unique index for %q: %v", table, err)
		}
		definitions = append(definitions, definition)

		matches := true
		for _, column := range columns {
			if !strings.Contains(definition, column) {
				matches = false
				break
			}
		}
		if matches {
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate unique indexes for %q: %v", table, err)
	}
	t.Errorf("no unique index for %q contains %q: %s", table, strings.Join(columns, ", "), strings.Join(definitions, "; "))
}

func assertCheckContains(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	table string,
	tokens ...string,
) {
	t.Helper()

	rows, err := pool.Query(
		ctx,
		`SELECT pg_get_constraintdef(oid)
		 FROM pg_constraint
		 WHERE conrelid = $1::regclass AND contype = 'c'`,
		"public."+table,
	)
	if err != nil {
		t.Fatalf("read check constraints for %q: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var definition string
		if err := rows.Scan(&definition); err != nil {
			t.Fatalf("scan check constraint for %q: %v", table, err)
		}

		matches := true
		for _, token := range tokens {
			if !strings.Contains(definition, token) {
				matches = false
				break
			}
		}
		if matches {
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate check constraints for %q: %v", table, err)
	}
	t.Errorf("no check constraint for %q contains %q", table, strings.Join(tokens, ", "))
}
