package httpclient_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/adapters/httpclient"
)

func TestClientRejectsResponseLargerThanConfiguredLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("{}" + strings.Repeat(" ", 32)))
	}))
	defer server.Close()

	client, err := httpclient.New(httpclient.Options{
		BaseURL:      server.URL,
		Timeout:      time.Second,
		MaxBodyBytes: 2,
		MaxAttempts:  1,
		UserAgent:    "solana-hype-paper-bot/test",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	var response map[string]any
	err = client.GetJSON(context.Background(), "/response", nil, &response)
	if !errors.Is(err, httpclient.ErrResponseTooLarge) {
		t.Fatalf("GetJSON() error = %v, want ErrResponseTooLarge", err)
	}
}
