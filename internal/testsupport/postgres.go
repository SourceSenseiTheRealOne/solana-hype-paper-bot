// Package testsupport provides isolated PostgreSQL fixtures for database contracts.
package testsupport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

var fixtures = struct {
	sync.Mutex
	urls map[*testing.T]string
}{urls: make(map[*testing.T]string)}

// DatabaseURL creates a private schema using the real project migrations. Calls
// within one test reuse it, including restart tests; no pre-existing rows are deleted.
func DatabaseURL(t *testing.T) string {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL is required for isolated PostgreSQL contracts")
	}
	fixtures.Lock()
	defer fixtures.Unlock()
	if existing, ok := fixtures.urls[t]; ok {
		return existing
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must be a PostgreSQL URL")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal("generate isolated schema name")
	}
	schema := "paper_test_" + hex.EncodeToString(nonce[:])
	identifier := pgx.Identifier{schema}.Sanitize()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connection, err := pgx.Connect(ctx, base)
	if err != nil {
		t.Fatal("connect to PostgreSQL fixture server")
	}
	defer func() { _ = connection.Close(context.Background()) }()
	if _, err := connection.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal("create isolated fixture schema")
	}
	t.Cleanup(func() {
		fixtures.Lock()
		delete(fixtures.urls, t)
		fixtures.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cleanup, err := pgx.Connect(ctx, base)
		if err != nil {
			t.Error("connect for fixture schema cleanup")
			return
		}
		defer func() { _ = cleanup.Close(context.Background()) }()
		if _, err := cleanup.Exec(ctx, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Error("remove this test's isolated schema")
		}
	})
	if _, err := connection.Exec(ctx, "SELECT set_config('search_path', $1, false)", schema); err != nil {
		t.Fatal("select isolated fixture schema")
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate project migrations")
	}
	paths, err := filepath.Glob(filepath.Join(filepath.Dir(filename), "..", "..", "supabase", "migrations", "*.sql"))
	if err != nil || len(paths) == 0 {
		t.Fatal("project migration files are required")
	}
	sort.Strings(paths)
	for _, path := range paths {
		statement, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s", filepath.Base(path))
		}
		if _, err := connection.Exec(ctx, string(statement)); err != nil {
			t.Fatalf("apply migration %s", filepath.Base(path))
		}
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	fixtures.urls[t] = parsed.String()
	return fixtures.urls[t]
}
