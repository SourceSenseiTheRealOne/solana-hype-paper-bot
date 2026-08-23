package application

import (
	"context"
	"errors"
	"time"
)

// ScheduledJob is an idempotent paper-only unit of work. It has no authority
// beyond the injected dependencies used by its RunOnce implementation.
type ScheduledJob interface {
	RunOnce(context.Context) error
}

type SplitCadenceOptions struct {
	Scan            ScheduledJob
	Monitor         ScheduledJob
	ScanInterval    time.Duration
	MonitorInterval time.Duration
	ScanTimeout     time.Duration
	MonitorTimeout  time.Duration
	OnError         func(error)
}

// SplitCadence runs scan and monitoring work serially. A slow scan therefore
// delays, rather than overlaps, a monitor tick; this prevents duplicate
// provider work and competing paper-position transitions.
type SplitCadence struct{ options SplitCadenceOptions }

func NewSplitCadence(options SplitCadenceOptions) *SplitCadence {
	return &SplitCadence{options: options}
}

func (cadence *SplitCadence) Run(ctx context.Context) error {
	if cadence == nil || cadence.options.Scan == nil || cadence.options.Monitor == nil || cadence.options.ScanInterval <= 0 || cadence.options.MonitorInterval <= 0 || cadence.options.ScanTimeout <= 0 || cadence.options.MonitorTimeout <= 0 {
		return errors.New("split cadence is not completely configured")
	}
	if ctx == nil {
		return errors.New("split cadence requires a context")
	}

	cadence.run(ctx, cadence.options.Scan, cadence.options.ScanTimeout)
	cadence.run(ctx, cadence.options.Monitor, cadence.options.MonitorTimeout)

	scanTicker := time.NewTicker(cadence.options.ScanInterval)
	defer scanTicker.Stop()
	monitorTicker := time.NewTicker(cadence.options.MonitorInterval)
	defer monitorTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-scanTicker.C:
			cadence.run(ctx, cadence.options.Scan, cadence.options.ScanTimeout)
		case <-monitorTicker.C:
			cadence.run(ctx, cadence.options.Monitor, cadence.options.MonitorTimeout)
		}
	}
}

func (cadence *SplitCadence) run(parent context.Context, job ScheduledJob, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	if err := job.RunOnce(ctx); err != nil && cadence.options.OnError != nil {
		cadence.options.OnError(err)
	}
}
