package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/application"
)

func TestSplitCadenceRunsInitialScanAndMonitorSerially(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := make([]string, 0, 2)
	scan := scheduledJobFake{run: func(context.Context) error {
		calls = append(calls, "scan")
		return nil
	}}
	monitor := scheduledJobFake{run: func(context.Context) error {
		calls = append(calls, "monitor")
		cancel()
		return nil
	}}

	err := application.NewSplitCadence(application.SplitCadenceOptions{
		Scan:            scan,
		Monitor:         monitor,
		ScanInterval:    5 * time.Minute,
		MonitorInterval: 30 * time.Second,
		ScanTimeout:     20 * time.Second,
		MonitorTimeout:  10 * time.Second,
	}).Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := calls, []string{"scan", "monitor"}; !sameStrings(got, want) {
		t.Fatalf("scheduled calls = %v, want serial initial %v", got, want)
	}
}

type scheduledJobFake struct{ run func(context.Context) error }

func (fake scheduledJobFake) RunOnce(ctx context.Context) error { return fake.run(ctx) }
