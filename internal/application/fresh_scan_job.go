package application

import (
	"context"
	"errors"
	"time"
)

// FreshJob creates a new job graph per scheduler tick so every time-dependent
// evidence check shares one current UTC timestamp.
type FreshJobOptions struct {
	Now   func() time.Time
	Build func(time.Time) (ScheduledJob, error)
}

type FreshJob struct {
	options FreshJobOptions
}

func NewFreshJob(options FreshJobOptions) *FreshJob {
	return &FreshJob{options: options}
}

func (job *FreshJob) RunOnce(ctx context.Context) error {
	if job == nil || job.options.Now == nil || job.options.Build == nil {
		return errors.New("fresh scan job is not completely configured")
	}
	now := job.options.Now().UTC()
	if now.IsZero() {
		return errors.New("fresh scan job clock returned zero time")
	}
	built, err := job.options.Build(now)
	if err != nil {
		return err
	}
	if built == nil {
		return errors.New("fresh scan job builder returned no job")
	}
	return built.RunOnce(ctx)
}

// FreshScanJob keeps the scan-oriented API used by existing callers while
// FreshJob can also construct a current position-monitor graph per tick.
type FreshScanJob = FreshJob
type FreshScanJobOptions = FreshJobOptions

func NewFreshScanJob(options FreshScanJobOptions) *FreshScanJob {
	return NewFreshJob(options)
}
