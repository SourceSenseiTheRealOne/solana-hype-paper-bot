package domain

import "errors"

type SocialPolicy struct {
	MinScore             int
	MinUniqueAuthors     int
	MinOriginalPosts     int
	MinExactMintMentions int
	MaxWarningPosts      int
}

type SocialPolicyEvaluation struct {
	Eligible bool
}

func (policy SocialPolicy) Validate() error {
	if policy.MinScore < 1 || policy.MinScore > 100 {
		return errors.New("minimum social score must be between 1 and 100")
	}
	if policy.MinUniqueAuthors < 1 || policy.MinOriginalPosts < 1 || policy.MinExactMintMentions < 1 {
		return errors.New("minimum social aggregate thresholds must be positive")
	}
	if policy.MaxWarningPosts < 0 {
		return errors.New("maximum warning posts must be non-negative")
	}
	return nil
}

func (policy SocialPolicy) Evaluate(metrics SocialMetrics, score int) SocialPolicyEvaluation {
	if policy.Validate() != nil || !validSocialMetrics(metrics) || score < 0 || score > 100 {
		return SocialPolicyEvaluation{}
	}
	return SocialPolicyEvaluation{Eligible: score >= policy.MinScore &&
		metrics.UniqueAuthors >= policy.MinUniqueAuthors &&
		metrics.OriginalPosts >= policy.MinOriginalPosts &&
		metrics.ExactMintMentions >= policy.MinExactMintMentions &&
		metrics.WarningPosts <= policy.MaxWarningPosts}
}

func validSocialMetrics(metrics SocialMetrics) bool {
	if metrics.Posts <= 0 || metrics.UniqueAuthors < 0 || metrics.OriginalPosts < 0 || metrics.Reposts < 0 || metrics.Replies < 0 || metrics.ExactMintMentions < 0 || metrics.WarningPosts < 0 {
		return false
	}
	if metrics.UniqueAuthors > metrics.Posts || metrics.ExactMintMentions > metrics.Posts || metrics.WarningPosts > metrics.Posts {
		return false
	}
	return metrics.OriginalPosts+metrics.Reposts+metrics.Replies == metrics.Posts
}

type VerdictPolicy struct {
	MinimumConfidence              int
	MinimumHypeQuality             int
	MaximumManipulationProbability int
}

func (policy VerdictPolicy) Validate() error {
	if policy.MinimumConfidence < 1 || policy.MinimumConfidence > 100 {
		return errors.New("minimum verdict confidence must be between 1 and 100")
	}
	if policy.MinimumHypeQuality < 1 || policy.MinimumHypeQuality > 100 {
		return errors.New("minimum hype quality must be between 1 and 100")
	}
	if policy.MaximumManipulationProbability < 0 || policy.MaximumManipulationProbability > 100 {
		return errors.New("maximum manipulation probability must be between 0 and 100")
	}
	return nil
}

func (policy VerdictPolicy) Accepts(verdict Verdict) bool {
	if policy.Validate() != nil || verdict.Confidence < 0 || verdict.Confidence > 100 || verdict.HypeQualityScore < 0 || verdict.HypeQualityScore > 100 || verdict.ManipulationProbability < 0 || verdict.ManipulationProbability > 100 {
		return false
	}
	return verdict.Outcome == VerdictBuy && verdict.Confidence >= policy.MinimumConfidence &&
		verdict.HypeQualityScore >= policy.MinimumHypeQuality &&
		verdict.ManipulationProbability <= policy.MaximumManipulationProbability
}
