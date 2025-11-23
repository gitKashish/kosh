package search

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gitKashish/kosh/src/internals/logger"
	"github.com/gitKashish/kosh/src/internals/model"
)

const (
	// feature weights
	LABEL_WEIGHT     = 0.60
	USER_WEIGHT      = 0.20
	RECENCY_WEIGHT   = 0.12
	FREQUENCY_WEIGHT = 0.05

	// string scoring
	PREFIX_BOOST    = 1.0
	SUBSTRING_BOOST = 0.5

	// limits
	MAX_STRING_SCORE    = 5.0
	MIN_SCORE_THRESHOLD = 0.2
)

type SearchResult struct {
	Credential model.Credential
	Score      float64
}

func BestMatch(queryLabel, queryUser string, credentials []model.Credential, now time.Time) *SearchResult {
	res := Search(queryLabel, queryUser, credentials, MIN_SCORE_THRESHOLD, now)
	if len(res) == 0 {
		return nil
	}
	return &res[0]
}

func Search(queryLabel, queryUser string, credentials []model.Credential, threshold float64, now time.Time) []SearchResult {
	timeSearchStart := time.Now()
	results := make([]SearchResult, 0, len(credentials))

	logger.Debug("query %s %s", queryLabel, queryUser)
	for _, c := range credentials {
		score := ScoreQuery(
			strings.ToLower(queryLabel),
			strings.ToLower(queryUser),
			strings.ToLower(c.Label),
			strings.ToLower(c.User),
			c.AccessCount,
			c.AccessedAt,
			now,
		)

		// consider record only if it satisfies a minimum threshold
		if score >= threshold {
			results = append(results, SearchResult{c, score})
		}
	}

	// sort records based on score, with access count and label as tie-breakers
	sort.Slice(results, func(i, j int) bool {
		prev := results[i]
		curr := results[j]

		if prev.Score == curr.Score {
			// same score tie-breaker
			if prev.Credential.AccessCount == curr.Credential.AccessCount {
				// same access count tie-breaker
				return prev.Credential.Label < curr.Credential.Label
			}
			return prev.Credential.AccessCount > curr.Credential.AccessCount
		}
		return prev.Score > curr.Score
	})

	time.Sleep(time.Nanosecond)
	timeSearchElapsed := time.Since(timeSearchStart)
	logger.Debug("time for search %s", timeSearchElapsed.String())

	return results
}

// ScoreQuery provides the overall score of an individual credential query based on following - label and/or user
// string match, last used date-time, and frequency of usage.
func ScoreQuery(queryLabel, queryUser, label, user string, count int, last time.Time, now time.Time) float64 {
	labelScore := 0.0
	userScore := 0.0

	freqScore := frequencyScore(count) * FREQUENCY_WEIGHT
	recScore := recencyScore(last, now) * RECENCY_WEIGHT

	if queryLabel != "" {
		labelScore = stringScore(queryLabel, label) * LABEL_WEIGHT
	}

	if queryUser != "" {
		userScore = stringScore(queryUser, user) * USER_WEIGHT
	}

	score := labelScore + userScore + recScore + freqScore
	return score
}

// stringScore provides a score for query and target match based on levenshtein distance (normalized) with bias
// towards prefix and substring matching. In case of an exact match a MAX_STRING_SCORE is returned.
func stringScore(query, target string) float64 {
	// On exact match return max score
	if query == target {
		return MAX_STRING_SCORE
	}

	simScore := similarityScore(query, target)

	if strings.HasPrefix(target, query) {
		simScore += PREFIX_BOOST
	} else if strings.Contains(target, query) {
		simScore += SUBSTRING_BOOST
	}

	// clamp the final score
	if simScore > MAX_STRING_SCORE {
		return MAX_STRING_SCORE
	}

	return simScore
}

// similarityScore provides a normalized levenshtein distance between source and target strings
func similarityScore(source, target string) float64 {
	distance := Levenshtein(source, target)
	maxLen := max(len(source), len(target))
	similarity := 1.0 - (float64(distance) / float64(maxLen))
	return similarity
}

// recencyScore provides a normalized score with a quick decay score based on last access date-time
func recencyScore(last time.Time, now time.Time) float64 {
	if last.IsZero() {
		return 0.0
	}
	if now.Before(last) {
		now = last
	}
	hours := now.Sub(last).Hours()
	// Quick decay: half-life ~12h (tunable)
	return 1.0 / (1.0 + hours/12.0)
}

// frequencyStore provides a normalized, logarithmic score based on frequency of usage of a record
func frequencyScore(count int) float64 {
	if count <= 0 {
		return 0
	}
	return math.Log(float64(count)+1.0) / 5.0
}

// helper functions

func Levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	if la > lb {
		a, b = b, a
		la, lb = lb, la
	}

	prev := make([]int, la+1)
	curr := make([]int, la+1)

	for i := 0; i <= la; i++ {
		prev[i] = i
	}

	for j := 1; j <= lb; j++ {
		curr[0] = j
		bj := b[j-1]

		for i := 1; i <= la; i++ {
			cost := 0
			if a[i-1] != bj {
				cost = 1
			}

			deletion := prev[i] + 1
			insertion := curr[i-1] + 1
			substitution := prev[i-1] + cost

			curr[i] = min(deletion, min(insertion, substitution))
		}

		prev, curr = curr, prev
	}

	return prev[la]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
