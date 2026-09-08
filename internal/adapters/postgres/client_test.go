package postgres

import (
	"context"
	"testing"
)

func TestOpenRejectsEmptyDatabaseURL(t *testing.T) {
	if _, err := Open(context.Background(), ""); err == nil {
		t.Fatal("Open() accepted an empty database URL")
	}
}
