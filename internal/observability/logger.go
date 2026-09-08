package observability

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LoggerOptions struct {
	Dir           string
	Now           func() time.Time
	RetentionDays int
}

type DailyLogger struct {
	*slog.Logger
	file *os.File
}

func (logger *DailyLogger) Close() error {
	return logger.file.Close()
}

func NewDailyJSONLogger(options LoggerOptions) (*DailyLogger, error) {
	if options.Dir == "" {
		return nil, fmt.Errorf("log directory is required")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.RetentionDays < 1 {
		options.RetentionDays = 30
	}
	if err := os.MkdirAll(options.Dir, 0o750); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	if err := prune(options.Dir, options.Now().UTC(), options.RetentionDays); err != nil {
		return nil, err
	}
	path := filepath.Join(options.Dir, options.Now().UTC().Format("2006-01-02")+".jsonl")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open daily log file: %w", err)
	}
	return &DailyLogger{
		Logger: slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{ReplaceAttr: allowlistedAttr})),
		file:   file,
	}, nil
}

func allowlistedAttr(_ []string, attr slog.Attr) slog.Attr {
	switch attr.Key {
	case slog.TimeKey, slog.LevelKey, slog.MessageKey,
		"provider", "source", "event", "request_id", "operation", "status",
		"duration_ms", "count", "attempt", "page", "reason_code":
		return attr
	default:
		return slog.Attr{}
	}
}

func prune(dir string, now time.Time, retentionDays int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read log directory: %w", err)
	}
	utcNow := now.UTC()
	dayStart := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
	cutoff := dayStart.AddDate(0, 0, 1-retentionDays)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		date, err := time.Parse("2006-01-02.jsonl", entry.Name())
		if err != nil || !date.Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("remove retained log: %w", err)
		}
	}
	return nil
}
