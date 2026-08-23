package application_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/application"
	"github.com/SourceSenseiTheRealOne/solana-hype-paper-bot/internal/domain"
)

func TestProductionScanJobRecordsPublicNonAdmissionForIneligibleCandidate(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	discovery := &productionDiscovery{result: application.DiscoveryResult{Pools: []domain.DiscoveredPool{productionPool(now.Add(-5 * time.Minute))}}}
	evaluator := &productionEvaluator{evaluation: domain.CandidateEvaluation{Eligible: false, Rules: []domain.RuleResult{{Code: domain.RuleLiquidity, Passed: false}}}}
	recorder := &productionReviewRecorder{}
	options := validProductionScanOptions(now, discovery, evaluator)
	options.Reviewed = recorder

	if err := application.NewProductionScanJob(options).RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if recorder.calls != 1 || recorder.candidate.MintAddress != "mint" || !recorder.candidate.CheckedAt.Equal(now) || recorder.candidate.Outcome != domain.ReviewedCandidateNotTraded || recorder.candidate.Reason != domain.ReviewReasonDeterministicLiquidity {
		t.Fatalf("reviewed candidate = %#v calls=%d, want one safe non-admission", recorder.candidate, recorder.calls)
	}
}

func TestProductionScanJobRecordsSafeScanLifecycle(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	discovery := &productionDiscovery{result: application.DiscoveryResult{}}
	evaluator := &productionEvaluator{}
	activity := &productionActivityRecorder{}
	options := validProductionScanOptions(now, discovery, evaluator)
	options.Activity = activity

	if err := application.NewProductionScanJob(options).RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if len(activity.activities) != 2 || activity.activities[0].Job != domain.AutomationJobScan || activity.activities[0].Outcome != domain.AutomationOutcomeStarted || activity.activities[1].Outcome != domain.AutomationOutcomeCompleted {
		t.Fatalf("activity = %#v, want safe SCAN STARTED then COMPLETED", activity.activities)
	}
}

func TestProductionScanJobRecordsMarketEvidenceUnavailableAndContinues(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	first := productionPool(now.Add(-5 * time.Minute))
	first.MintAddress = "market-unavailable-mint"
	first.PoolAddress = "market-unavailable-pool"
	second := productionPool(now.Add(-4 * time.Minute))
	second.MintAddress = "liquidity-mint"
	second.PoolAddress = "liquidity-pool"
	discovery := &productionDiscovery{result: application.DiscoveryResult{Pools: []domain.DiscoveredPool{first, second}}}
	recorder := &productionReviewSequenceRecorder{}
	options := validProductionScanOptions(now, discovery, &productionEvaluator{})
	options.Evaluator = &marketFailureThenIneligibleEvaluator{}
	options.Reviewed = recorder

	if err := application.NewProductionScanJob(options).RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if got, want := len(recorder.candidates), 2; got != want {
		t.Fatalf("reviewed candidate count = %d, want %d", got, want)
	}
	if got, want := recorder.candidates[0].Reason, domain.ReviewedCandidateReason("market_evidence_unavailable"); got != want {
		t.Fatalf("first reason = %q, want %q", got, want)
	}
	if got, want := recorder.candidates[1].Reason, domain.ReviewReasonDeterministicLiquidity; got != want {
		t.Fatalf("second reason = %q, want %q", got, want)
	}
}

func TestProductionScanJobRecordsSocialEvidenceUnavailableWithoutRequestingHermes(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	discovery := &productionDiscovery{result: application.DiscoveryResult{Pools: []domain.DiscoveredPool{productionPool(now.Add(-5 * time.Minute))}}}
	verdicts := &productionVerdicts{record: validProductionVerdictRecord()}
	recorder := &productionReviewSequenceRecorder{}
	options := validProductionScanOptions(now, discovery, &productionEvaluator{evaluation: eligibleProductionEvaluation()})
	options.Social = &productionSocial{analysis: application.SocialAnalysis{}}
	options.Verdicts = verdicts
	options.Reviewed = recorder

	if err := application.NewProductionScanJob(options).RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if verdicts.calls != 0 {
		t.Fatalf("Hermes verdict calls = %d, want 0 for empty social evidence", verdicts.calls)
	}
	if got, want := len(recorder.candidates), 1; got != want {
		t.Fatalf("reviewed candidate count = %d, want %d", got, want)
	}
	if got, want := recorder.candidates[0].Reason, domain.ReviewedCandidateReason("social_evidence_unavailable"); got != want {
		t.Fatalf("reviewed reason = %q, want %q", got, want)
	}
}

func TestProductionScanJobMapsBoldMomentumRulesToFixedReasons(t *testing.T) {
	now := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		rule string
		want domain.ReviewedCandidateReason
	}{
		{name: "buy share", rule: domain.RuleFiveMinuteBuyShare, want: domain.ReviewReasonDeterministicBuyShare},
		{name: "turnover", rule: domain.RuleFiveMinuteTurnover, want: domain.ReviewReasonDeterministicTurnover},
		{name: "price change", rule: domain.RuleFiveMinutePriceChange, want: domain.ReviewReasonDeterministicPriceChange},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			discovery := &productionDiscovery{result: application.DiscoveryResult{Pools: []domain.DiscoveredPool{productionPool(now.Add(-5 * time.Minute))}}}
			evaluator := &productionEvaluator{evaluation: domain.CandidateEvaluation{Eligible: false, Rules: []domain.RuleResult{{Code: test.rule, Passed: false}}}}
			recorder := &productionReviewRecorder{}
			options := validProductionScanOptions(now, discovery, evaluator)
			options.Reviewed = recorder
			if err := application.NewProductionScanJob(options).RunOnce(context.Background()); err != nil {
				t.Fatalf("RunOnce() error = %v", err)
			}
			if recorder.candidate.Reason != test.want {
				t.Fatalf("review reason = %q, want %q", recorder.candidate.Reason, test.want)
			}
		})
	}
}

type productionReviewRecorder struct {
	calls     int
	candidate domain.ReviewedCandidate
}

func (recorder *productionReviewRecorder) AppendReviewed(_ context.Context, candidate domain.ReviewedCandidate) error {
	recorder.calls++
	recorder.candidate = candidate
	return nil
}

type productionActivityRecorder struct{ activities []domain.AutomationActivity }

func (recorder *productionActivityRecorder) Append(_ context.Context, activity domain.AutomationActivity) error {
	recorder.activities = append(recorder.activities, activity)
	return nil
}

type marketEvidenceUnavailableError struct{}

func (marketEvidenceUnavailableError) Error() string              { return "market evidence unavailable" }
func (marketEvidenceUnavailableError) MarketEvidenceUnavailable() {}

type marketFailureThenIneligibleEvaluator struct{ calls int }

func (evaluator *marketFailureThenIneligibleEvaluator) Evaluate(context.Context, domain.DiscoveredPool) (domain.CandidateEvaluation, error) {
	evaluator.calls++
	if evaluator.calls == 1 {
		return domain.CandidateEvaluation{}, fmt.Errorf("fetch market evidence: %w", marketEvidenceUnavailableError{})
	}
	return domain.CandidateEvaluation{Eligible: false, Rules: []domain.RuleResult{{Code: domain.RuleLiquidity, Passed: false}}}, nil
}

type productionReviewSequenceRecorder struct{ candidates []domain.ReviewedCandidate }

func (recorder *productionReviewSequenceRecorder) AppendReviewed(_ context.Context, candidate domain.ReviewedCandidate) error {
	recorder.candidates = append(recorder.candidates, candidate)
	return nil
}
