package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
	"unicode"
)

type SocialPost struct {
	ID              string
	AuthorID        string
	CreatedAt       time.Time
	AuthorCreatedAt time.Time
	Text            string
	Likes           int64
	Replies         int64
	RepostCount     int64
	QuoteCount      int64
	Followers       int64
	IsReply         bool
	IsRepost        bool
}

type SocialWindow struct {
	MintAddress string
	StartsAt    time.Time
	EndsAt      time.Time
}

type SocialMetrics struct {
	Posts                         int
	UniqueAuthors                 int
	ExactMintMentions             int
	OriginalPosts                 int
	Reposts                       int
	Replies                       int
	PostsPerMinuteBPS             int64
	ExactMintMentionBPS           int64
	RepeatedTextHashes            int
	PostTextHashes                map[string]string
	FollowerAdjustedEngagementBPS int64
	WarningPosts                  int
	SimultaneousPosts             int
}

func AnalyzeSocial(posts []SocialPost, window SocialWindow) SocialMetrics {
	metrics := SocialMetrics{PostTextHashes: make(map[string]string)}
	if strings.TrimSpace(window.MintAddress) == "" || window.StartsAt.IsZero() || window.EndsAt.Before(window.StartsAt) {
		return metrics
	}
	authors := make(map[string]struct{})
	hashes := make(map[string]int)
	minutes := make(map[time.Time]int)
	engagementBPS := int64(0)
	for _, post := range posts {
		if strings.TrimSpace(post.ID) == "" || strings.TrimSpace(post.AuthorID) == "" || post.CreatedAt.Before(window.StartsAt) || post.CreatedAt.After(window.EndsAt) {
			continue
		}
		metrics.Posts++
		authors[post.AuthorID] = struct{}{}
		if containsExactTerm(post.Text, window.MintAddress) {
			metrics.ExactMintMentions++
		}
		if post.IsRepost {
			metrics.Reposts++
		} else if post.IsReply {
			metrics.Replies++
		} else {
			metrics.OriginalPosts++
		}
		normalizedHash := textHash(post.Text)
		hashes[normalizedHash]++
		metrics.PostTextHashes[post.ID] = normalizedHash
		minute := post.CreatedAt.UTC().Truncate(time.Minute)
		minutes[minute]++
		followers := post.Followers
		if followers < 1 {
			followers = 1
		}
		engagement := nonNegative(post.Likes) + nonNegative(post.Replies) + nonNegative(post.RepostCount) + nonNegative(post.QuoteCount)
		engagementBPS += engagement * 10_000 / followers
		if containsWarningTerm(post.Text) {
			metrics.WarningPosts++
		}
	}
	metrics.UniqueAuthors = len(authors)
	for _, count := range hashes {
		if count > 1 {
			metrics.RepeatedTextHashes++
		}
	}
	for _, count := range minutes {
		if count > 1 {
			metrics.SimultaneousPosts += count
		}
	}
	if metrics.Posts == 0 {
		return metrics
	}
	windowMinutes := int64(window.EndsAt.Sub(window.StartsAt) / time.Minute)
	if windowMinutes < 1 {
		windowMinutes = 1
	}
	metrics.PostsPerMinuteBPS = int64(metrics.Posts) * 10_000 / windowMinutes
	metrics.ExactMintMentionBPS = int64(metrics.ExactMintMentions) * 10_000 / int64(metrics.Posts)
	metrics.FollowerAdjustedEngagementBPS = engagementBPS / int64(metrics.Posts)
	return metrics
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
func textHash(value string) string {
	digest := sha256.Sum256([]byte(normalizeSocialText(value)))
	return hex.EncodeToString(digest[:])
}
func normalizeSocialText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}
func containsExactTerm(text, term string) bool {
	for _, token := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if strings.EqualFold(token, term) {
			return true
		}
	}
	return false
}
func containsWarningTerm(value string) bool {
	text := normalizeSocialText(value)
	for _, term := range []string{"scam", "rug", "honeypot", "cannot sell"} {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

// ScoreSocial gives bounded, reproducible weight to quality evidence and manipulation signals.
func ScoreSocial(metrics SocialMetrics) int {
	score := metrics.UniqueAuthors*10 + int(metrics.ExactMintMentionBPS/1_000) + min(20, int(metrics.FollowerAdjustedEngagementBPS/100))
	score -= metrics.RepeatedTextHashes*10 + metrics.WarningPosts*25 + metrics.SimultaneousPosts*2
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}
