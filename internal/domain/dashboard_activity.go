package domain

import (
	"errors"
	"strings"
	"time"
)

const MaxDashboardItems = 10

type AutomationJob string

const (
	AutomationJobScan    AutomationJob = "SCAN"
	AutomationJobMonitor AutomationJob = "MONITOR"
)

type AutomationOutcome string

const (
	AutomationOutcomeStarted   AutomationOutcome = "STARTED"
	AutomationOutcomeCompleted AutomationOutcome = "COMPLETED"
	AutomationOutcomeFailed    AutomationOutcome = "FAILED"
)

type AutomationActivity struct {
	OccurredAt time.Time
	Job        AutomationJob
	Outcome    AutomationOutcome
	Category   string
	Stage      string
}

func (activity AutomationActivity) Validate() error {
	if activity.OccurredAt.IsZero() || activity.OccurredAt.Location() != time.UTC {
		return errors.New("automation activity time must be UTC")
	}
	if activity.Job != AutomationJobScan && activity.Job != AutomationJobMonitor {
		return errors.New("automation activity job is unsupported")
	}
	if activity.Outcome != AutomationOutcomeStarted && activity.Outcome != AutomationOutcomeCompleted && activity.Outcome != AutomationOutcomeFailed {
		return errors.New("automation activity outcome is unsupported")
	}
	if activity.Outcome == AutomationOutcomeFailed {
		if !isAutomationErrorCategory(activity.Category) || !isAutomationFailureStage(activity.Stage) {
			return errors.New("failed automation activity requires safe category and stage")
		}
		return nil
	}
	if activity.Category != "" || activity.Stage != "" {
		return errors.New("non-failed automation activity must not include failure fields")
	}
	return nil
}

func ValidateDashboardActivity(activities []AutomationActivity) error {
	if len(activities) > MaxDashboardItems {
		return errors.New("automation activity exceeds dashboard limit")
	}
	for index, activity := range activities {
		if err := activity.Validate(); err != nil {
			return err
		}
		if index > 0 && activity.OccurredAt.After(activities[index-1].OccurredAt) {
			return errors.New("automation activity must be newest first")
		}
	}
	return nil
}

func isAutomationErrorCategory(value string) bool {
	switch value {
	case "context_canceled", "context_deadline_exceeded", "provider_rate_limited", "provider_response_too_large", "provider_upstream_error", "internal_error":
		return true
	default:
		return false
	}
}

func isAutomationFailureStage(value string) bool {
	switch value {
	case "discovery", "evaluation", "social_evidence", "hermes_verdict", "paper_admission", "paper_open", "position_monitor", "daily_reconcile", "daily_results_load", "daily_report_write", "daily_report", "internal":
		return true
	default:
		return false
	}
}

type ReviewedCandidateOutcome string

const ReviewedCandidateNotTraded ReviewedCandidateOutcome = "NOT_TRADED"

type ReviewedCandidateReason string

const (
	ReviewReasonDeterministicPoolAge          ReviewedCandidateReason = "deterministic_pool_age"
	ReviewReasonDeterministicLiquidity        ReviewedCandidateReason = "deterministic_liquidity"
	ReviewReasonDeterministicFiveMinuteVolume ReviewedCandidateReason = "deterministic_five_minute_activity"
	ReviewReasonDeterministicBuyShare         ReviewedCandidateReason = "deterministic_five_minute_buy_share"
	ReviewReasonDeterministicTurnover         ReviewedCandidateReason = "deterministic_five_minute_turnover"
	ReviewReasonDeterministicPriceChange      ReviewedCandidateReason = "deterministic_five_minute_price_change"
	ReviewReasonMarketEvidenceUnavailable     ReviewedCandidateReason = "market_evidence_unavailable"
	ReviewReasonSocialEvidenceUnavailable     ReviewedCandidateReason = "social_evidence_unavailable"
	ReviewReasonDeterministicSocialQuality    ReviewedCandidateReason = "deterministic_social_quality"
	ReviewReasonHermesVerdictThreshold        ReviewedCandidateReason = "hermes_verdict_threshold"

	ReviewReasonDeterministicEntryPriceImpact ReviewedCandidateReason = "deterministic_entry_price_impact"
	ReviewReasonDeterministicEntryRoute       ReviewedCandidateReason = "deterministic_entry_route"
	ReviewReasonDeterministicExitRoute        ReviewedCandidateReason = "deterministic_exit_route"
	ReviewReasonDeterministicMintAuthority    ReviewedCandidateReason = "deterministic_mint_authority_revoked"
	ReviewReasonDeterministicFreezeAuthority  ReviewedCandidateReason = "deterministic_freeze_authority_revoked"
	ReviewReasonDeterministicTokenProgram     ReviewedCandidateReason = "deterministic_token_program_extensions"
	ReviewReasonAdmissionNotAdmitted          ReviewedCandidateReason = "admission_not_admitted"
	ReviewReasonLegacyUnavailable             ReviewedCandidateReason = "legacy_reason_unavailable"
)

type ReviewedCandidate struct {
	MintAddress string
	CheckedAt   time.Time
	Outcome     ReviewedCandidateOutcome
	Reason      ReviewedCandidateReason
}

func (candidate ReviewedCandidate) Validate() error {
	if strings.TrimSpace(candidate.MintAddress) == "" {
		return errors.New("reviewed candidate mint address is required")
	}
	if candidate.CheckedAt.IsZero() || candidate.CheckedAt.Location() != time.UTC {
		return errors.New("reviewed candidate time must be UTC")
	}
	if candidate.Outcome != ReviewedCandidateNotTraded {
		return errors.New("reviewed candidate outcome is unsupported")
	}
	if !isReviewedCandidateReason(candidate.Reason) {
		return errors.New("reviewed candidate reason is unsupported")
	}
	return nil
}

func isReviewedCandidateReason(value ReviewedCandidateReason) bool {
	switch value {
	case ReviewReasonDeterministicPoolAge, ReviewReasonDeterministicLiquidity, ReviewReasonDeterministicFiveMinuteVolume, ReviewReasonDeterministicBuyShare, ReviewReasonDeterministicTurnover, ReviewReasonDeterministicPriceChange, ReviewReasonMarketEvidenceUnavailable, ReviewReasonSocialEvidenceUnavailable, ReviewReasonDeterministicSocialQuality, ReviewReasonHermesVerdictThreshold, ReviewReasonDeterministicEntryPriceImpact, ReviewReasonDeterministicEntryRoute, ReviewReasonDeterministicExitRoute, ReviewReasonDeterministicMintAuthority, ReviewReasonDeterministicFreezeAuthority, ReviewReasonDeterministicTokenProgram, ReviewReasonAdmissionNotAdmitted, ReviewReasonLegacyUnavailable:
		return true
	default:
		return false
	}
}

func ValidateReviewedCandidates(candidates []ReviewedCandidate) error {
	if len(candidates) > MaxDashboardItems {
		return errors.New("reviewed candidates exceed dashboard limit")
	}
	for index, candidate := range candidates {
		if err := candidate.Validate(); err != nil {
			return err
		}
		if index > 0 && candidate.CheckedAt.After(candidates[index-1].CheckedAt) {
			return errors.New("reviewed candidates must be newest first")
		}
	}
	return nil
}

type TwitterAnalytics struct {
	MintAddress       string
	SearchedAt        time.Time
	SearchCount       int
	Score             int
	Posts             int
	UniqueAuthors     int
	ExactMintMentions int
	WarningPosts      int
}

func (analytics TwitterAnalytics) Validate() error {
	if strings.TrimSpace(analytics.MintAddress) == "" {
		return errors.New("Twitter analytics mint address is required")
	}
	if analytics.SearchedAt.IsZero() || analytics.SearchedAt.Location() != time.UTC {
		return errors.New("Twitter analytics time must be UTC")
	}
	if analytics.SearchCount != 1 {
		return errors.New("Twitter analytics must represent one completed search")
	}
	if analytics.Score < 0 || analytics.Score > 100 {
		return errors.New("Twitter analytics score must be between zero and one hundred")
	}
	if analytics.Posts < 0 || analytics.UniqueAuthors < 0 || analytics.ExactMintMentions < 0 || analytics.WarningPosts < 0 {
		return errors.New("Twitter analytics aggregates cannot be negative")
	}
	return nil
}

func ValidateTwitterAnalytics(analytics []TwitterAnalytics) error {
	if len(analytics) > MaxDashboardItems {
		return errors.New("Twitter analytics exceed dashboard limit")
	}
	for index, item := range analytics {
		if err := item.Validate(); err != nil {
			return err
		}
		if index > 0 && item.SearchedAt.After(analytics[index-1].SearchedAt) {
			return errors.New("Twitter analytics must be newest first")
		}
	}
	return nil
}
