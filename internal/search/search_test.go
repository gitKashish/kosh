package search

import (
	"math"
	"testing"
	"time"

	"git.plutolab.org/plutolab/kosh/internal/constants"
	"git.plutolab.org/plutolab/kosh/internal/model"
)

func TestDamerauLevenshtein(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		// Empty Values
		{"empty both", "", "", 0},
		{"empty a", "", "abc", 3},
		{"empty b", "abc", "", 3},

		// Trivial Operations
		{"identical a and b", "abc", "abc", 0},
		{"one deletion", "abcd", "abc", 1},
		{"one addition", "abc", "abcd", 1},
		{"one substitution", "abd", "abc", 1},

		// Full Replace
		{"full replace same length", "abc", "xyz", 3},
		{"full replace diff length", "abcd", "xy", 4},

		// Structural
		{"symmetry", "kitten", "sitting", 3},
		{"case sensitive", "ABC", "abc", 3},

		// Adjacent Swap
		{"adjacent characters mistyped", "catr", "cart", 1},
		{"spaced apart swapped characters", "crta", "cart", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := damerauLevenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf(
					"levenshtein(%s, %s) = %d, want %d",
					tt.a, tt.b, got, tt.want,
				)
			}
		})
	}
}

func TestStringScore(t *testing.T) {
	t.Run("exact match returns 1.0", func(t *testing.T) {
		got := stringScore("abc", "abc")
		if got != 1.0 {
			t.Errorf("exact match got %f, want 1.0", got)
		}
	})

	t.Run("empty query should return 0", func(t *testing.T) {
		got := stringScore("", "abc")
		if got != 0.0 {
			t.Errorf("empty query got %f, want 0.0", got)
		}
	})

	t.Run("score should be casing agnostic", func(t *testing.T) {
		lowerCase := stringScore("git", "github")
		upperCase := stringScore("GIT", "github")
		if lowerCase != upperCase {
			t.Errorf("lower case score (%f) not equal to upper case score (%f), want them same", lowerCase, upperCase)
		}
	})

	t.Run("a query with just whitespaces should return 0", func(t *testing.T) {
		got := stringScore("   ", "accessToken")
		if got != 0.0 {
			t.Errorf("whitespace only query got %f, want 0", got)
		}
	})

	t.Run("leading and trailing whitespaces should be ignore", func(t *testing.T) {
		got := stringScore("  git ", "github    ")
		want := stringScore("git", "github")
		if got != want {
			t.Errorf("leading & trailing space got %f, want %f", got, want)
		}
	})

	t.Run("completely mismatched strings should not exceed min threshold", func(t *testing.T) {
		got := stringScore("xyz", "abc")
		if got >= MIN_SCORE_THRESHOLD {
			t.Errorf("mismatched string has score %f, must be less than %f", got, MIN_SCORE_THRESHOLD)
		}
	})

	t.Run("query with prefix always wins", func(t *testing.T) {
		prefixScore := stringScore("git", "github")
		fuzzyScore := stringScore("git", "digit")
		if fuzzyScore >= prefixScore {
			t.Errorf(
				"prefix score (%f) must be > fuzzy score (%f)",
				prefixScore, fuzzyScore,
			)
		}
	})

	t.Run("scoring order: exact > prefix > substring > fuzzy", func(t *testing.T) {
		// Keeping levenshteinDistance same (except for exact match)
		// so that only boosts affect the final result
		exact := stringScore("git", "git")        // lev distance = 0
		prefix := stringScore("git", "gitX")      // lev distance = 1
		substring := stringScore("git", "Xgit")   // lev distance = 1
		subsequence := stringScore("git", "gXit") // lev distance = 1
		fuzzy := stringScore("git", "gXt")        // leve distance = 1

		if !(exact > prefix &&
			prefix > substring &&
			substring > subsequence &&
			subsequence > fuzzy) {
			t.Errorf(
				"ordering violated, exact=%f prefix=%f substr=%f subsequence=%f fuzzy=%f",
				exact, prefix, substring, subsequence, fuzzy,
			)
		}
	})

	t.Run("abbrevation should match", func(t *testing.T) {
		score := stringScore("sslv", "saas_slave_database")
		if score < MIN_SCORE_THRESHOLD {
			t.Errorf(
				"abbrevation score (%f), must be greater than %f",
				score, MIN_SCORE_THRESHOLD,
			)
		}
	})

	t.Run("out-of-order abbrevation should score less than in-order", func(t *testing.T) {
		inOrder := stringScore("ssslv", "saas_slave_database")
		outOfOrder := stringScore("slvss", "saas_slave_database")
		if outOfOrder >= inOrder {
			t.Errorf(
				"out-of-order abbrevation score (%f), must be less than in-order abbrevation score (%f)",
				outOfOrder, inOrder,
			)
		}
	})
}

func TestScoreQuery_UserOnly(t *testing.T) {
	now := time.Now()
	last := time.Now().Add(-1 * time.Hour)

	score := ScoreQuery("", "alice", "github", "alice", 0, last, now)

	if score == 0 {
		t.Fatal("expected non-zero score for matching user")
	}
}

func TestScoreQuery_LabelAndUserBoost(t *testing.T) {
	now := time.Now()
	last := time.Now().Add(-1 * time.Hour)

	labelOnly := ScoreQuery("git", "", "github", "alice", 10, last, now)
	both := ScoreQuery("git", "alice", "github", "alice", 10, last, now)

	if both <= labelOnly {
		t.Fatal("combined label+user query should score higher")
	}
}

func TestSimilarityScore_EmptyStrings(t *testing.T) {
	if got := similarityScore("", ""); got != 1.0 {
		t.Fatalf(
			"empty strings should be perfectly similar, got %f, want %f",
			got, 1.0,
		)
	}
}

func TestRecencyScore(t *testing.T) {
	t.Run("zero last access should return 0", func(t *testing.T) {
		got := recencyScore(
			time.Time{},
			time.Now(),
		)

		if got != 0.0 {
			t.Errorf("zero time should have %f score, got %f", 0.0, got)
		}
	})

	t.Run("now before last access should clamp and return 1.0", func(t *testing.T) {
		got := recencyScore(
			time.Now(),
			time.Now().Add(-1*time.Minute),
		)

		if got != 1.0 {
			t.Errorf("current before last access has score %f, must be 1.0", got)
		}
	})
	t.Run("recently used credential scores higher than older credential", func(t *testing.T) {
		recent := recencyScore(time.Now().Add(-1*time.Hour), time.Now())
		older := recencyScore(time.Now().Add(-2*time.Hour), time.Now())
		if recent <= older {
			t.Errorf("recent scored %f, must be more than %f", recent, older)
		}
	})

	t.Run("score at half-life is approximately 0.5", func(t *testing.T) {
		last := time.Now().Add(-12 * time.Hour)
		got := recencyScore(last, time.Now())
		if math.Abs(got-0.5) > 0.01 {
			t.Errorf("half-life score got %f, want ~0.5", got)
		}
	})
}

func TestFrequencyScore(t *testing.T) {
	t.Run("credential with zero usage should score zero", func(t *testing.T) {
		got := frequencyScore(0)
		if got != 0.0 {
			t.Errorf("credential with zero usage got %f, want 0.0", got)
		}
	})

	t.Run("credential with a negative usage must score zero", func(t *testing.T) {
		got := frequencyScore(-1)
		if got != 0.0 {
			t.Errorf("credential with negative usage got %f, want 0.0", got)
		}
	})

	t.Run("credential with higher usage should score more than less used one", func(t *testing.T) {
		higherUsage := frequencyScore(250)
		lowerUsage := frequencyScore(80)
		if lowerUsage >= higherUsage {
			t.Errorf("less used cred's score (%f) >= more used cred's score (%f), must be less than more used",
				lowerUsage, higherUsage,
			)
		}
	})

	t.Run("cred score on usage reset threshold must not cross 1.0", func(t *testing.T) {
		got := frequencyScore(constants.AccessCountResetThreshold)
		if got > 1.0 {
			t.Errorf("cred with max usage before reset has score %f, must be less than 1.0", got)
		}
	})
}

func TestSearch_Ranking(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		query    string
		creds    []model.Credential
		expected string
	}{
		{
			name:  "higher frequency wins when string quality is similar",
			query: "git",
			creds: []model.Credential{
				{Label: "github", AccessCount: 10, AccessedAt: now},
				{Label: "gitlab", AccessCount: 50, AccessedAt: now},
			},
			expected: "gitlab",
		},
		{
			name:  "more recent usage wins",
			query: "git",
			creds: []model.Credential{
				{Label: "github", AccessCount: 10, AccessedAt: now.Add(-1 * time.Hour)},
				{Label: "gitlab", AccessCount: 50, AccessedAt: now.Add(-7 * 24 * time.Hour)},
			},
			expected: "github",
		},
		{
			name:  "prefix match beats fuzzy match",
			query: "git",
			creds: []model.Credential{
				{Label: "github", AccessCount: 1, AccessedAt: now},
				{Label: "digit", AccessCount: 1000, AccessedAt: now},
			},
			expected: "github",
		},
		{
			name:  "lexical order of label wins when score is same",
			query: "git",
			creds: []model.Credential{
				{Label: "github", AccessCount: 10, AccessedAt: time.Now()},
				{Label: "gitlab", AccessCount: 10, AccessedAt: time.Now()},
			},
			expected: "github",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := BestMatches(tt.query, "", tt.creds, now)

			if len(res) == 0 {
				t.Fatal("expected results")
			}

			if res[0].Credential.Label != tt.expected {
				t.Fatalf("expected %s first, got %s", tt.expected, res[0].Credential.Label)
			}
		})
	}
}

func TestSearch_ThresholdFiltering(t *testing.T) {
	now := time.Now()
	creds := []model.Credential{
		{Label: "abc", AccessCount: 10, AccessedAt: time.Now()},
		{Label: "github", AccessCount: 10, AccessedAt: time.Now()},
	}

	res := search("git", "", creds, MIN_SCORE_THRESHOLD, now)

	if len(res) != 1 || res[0].Credential.Label != "github" {
		t.Fatalf("unexpected results: %+v", res)
	}
}
