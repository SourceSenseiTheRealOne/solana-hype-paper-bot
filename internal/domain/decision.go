package domain

import (
	"errors"
	"fmt"
	"strings"
)

const (
	maxVerdictItems = 8
	maxVerdictText  = 280
)

type VerdictOutcome string

const (
	VerdictBuy    VerdictOutcome = "BUY"
	VerdictWatch  VerdictOutcome = "WATCH"
	VerdictReject VerdictOutcome = "REJECT"
)

type Verdict struct {
	Outcome                 VerdictOutcome `json:"verdict"`
	Confidence              int            `json:"confidence"`
	HypeQualityScore        int            `json:"hype_quality_score"`
	ManipulationProbability int            `json:"manipulation_probability"`
	Reasons                 []string       `json:"reasons"`
	RiskFlags               []string       `json:"risk_flags"`
	InvalidationConditions  []string       `json:"invalidation_conditions"`
	EvidencePostIDs         []string       `json:"evidence_post_ids"`
}

func (verdict Verdict) Validate(allowedEvidencePostIDs map[string]struct{}) error {
	switch verdict.Outcome {
	case VerdictBuy, VerdictWatch, VerdictReject:
	default:
		return fmt.Errorf("unsupported verdict %q", verdict.Outcome)
	}
	for name, score := range map[string]int{"confidence": verdict.Confidence, "hype quality score": verdict.HypeQualityScore, "manipulation probability": verdict.ManipulationProbability} {
		if score < 0 || score > 100 {
			return fmt.Errorf("%s must be between 0 and 100", name)
		}
	}
	for name, values := range map[string][]string{"reasons": verdict.Reasons, "risk flags": verdict.RiskFlags, "invalidation conditions": verdict.InvalidationConditions} {
		if err := validateVerdictTexts(name, values); err != nil {
			return err
		}
	}
	if len(verdict.EvidencePostIDs) > maxVerdictItems {
		return errors.New("evidence post IDs exceed limit")
	}
	seen := make(map[string]struct{}, len(verdict.EvidencePostIDs))
	for _, postID := range verdict.EvidencePostIDs {
		if _, exists := allowedEvidencePostIDs[postID]; !exists {
			return fmt.Errorf("verdict references unknown evidence post %q", postID)
		}
		if _, duplicate := seen[postID]; duplicate {
			return fmt.Errorf("verdict references duplicate evidence post %q", postID)
		}
		seen[postID] = struct{}{}
	}
	return nil
}

func validateVerdictTexts(name string, values []string) error {
	if len(values) > maxVerdictItems {
		return fmt.Errorf("%s exceed limit", name)
	}
	for _, value := range values {
		if len([]rune(strings.TrimSpace(value))) == 0 || len([]rune(value)) > maxVerdictText {
			return fmt.Errorf("%s contain invalid text", name)
		}
	}
	return nil
}
