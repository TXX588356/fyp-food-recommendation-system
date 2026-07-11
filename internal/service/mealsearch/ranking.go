package mealsearch

import (
	"fyp/food-rs/internal/interfaces"
	"math"
	"sort"
	"strings"
)

const (
	// Exact canonical name is the safest lexical match.
	// Example: query "Nasi Lemak" and food name "Nasi Lemak".
	exactNameScore = 1.00

	// Exact alias is almost as strong as exact canonical name.
	// Example: query "Wonton Mee" and food alias "Wantan Mee".
	exactAliasScore = 0.98

	// Fuzzy matches are plausible, but not authoritative.
	// Example: query typo "Wanton Mee" and candidate "Wantan Mee".
	fuzzyBaseScore = 0.85

	// Partial matches are weaker because they may only share a phrase.
	// Example: query "Chicken Sausage" and candidate "Chicken Breakfast Sausage".
	partialBaseScore = 0.70
)

// RankedMatch is the best lexical match classification for one food name
// against all generated meal search terms.
type RankedMatch struct {
	Kind        interfaces.FoodMatchKind
	MatchedTerm string
	Score       float64
}

// RankFoodName scores one canonical food name and optional aliases against the
// generated meal name / search terms. It returns false when no term is plausible.
// Ranking rules:
// - exact canonical name: 1.00
// - exact alias / search term: 0.98
// - fuzzy full-query match: 0.85 minus a small length penalty
// - partial phrase match: 0.70 minus an extra-token penalty
func RankFoodName(name string, aliases, queries []string) (RankedMatch, bool) {
	// Remove empty/duplicate query terms before ranking
	terms := uniqueNonEmptyTerms(queries)

	if normalizeMealName(name) == "" || len(terms) == 0 {
		return RankedMatch{}, false
	}

	var best RankedMatch
	found := false

	// Compare this candidate food name against every generated meal term.
	// Example queries may be:
	//   ["Wantan Mee", "Wonton Mee", "Wan Tan Mee"]
	for _, query := range terms {
		match, ok := rankAgainstQuery(name, aliases, query)
		if !ok {
			continue
		}

		// Keep only the strongest match across all generated terms
		if !found || betterRankedMatch(match, best) {
			best = match
			found = true
		}
	}

	return best, found

}

// SortFoodMatchCandidates orders candidates by highest score first, then by
// match strength, normalized food name, and stable food ID.
func SortFoodMatchCandidates(candidates []interfaces.FoodMatchCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		return CompareFoodMatchCandidates(candidates[i], candidates[j]) < 0
	})
}

// CompareFoodMatchCandidates compares two candidates using the canonical
// recommendation ordering. It returns -1 when a should sort before b, 1 when b
// should sort before a, and 0 when they are equivalent for ordering.
func CompareFoodMatchCandidates(a, b interfaces.FoodMatchCandidate) int {
	// Higher score wins
	if a.Score > b.Score {
		return -1
	}
	if a.Score < b.Score {
		return 1
	}

	// If score tie, prefer stronger match type:
	// exact name > exact alias > fuzzy > partial
	aKind := matchKindPriority(a.MatchKind)
	bKind := matchKindPriority(b.MatchKind)
	if aKind < bKind {
		return -1
	}
	if aKind > bKind {
		return 1
	}

	// If score and kind still tie, sort by normalized name so results are stable
	// and not dependent on database/map iteration order.
	aName := normalizeMealName(a.Food.Name)
	bName := normalizeMealName(b.Food.Name)
	if aName < bName {
		return -1
	}
	if aName > bName {
		return 1
	}

	// Final tie-breaker: stable ID.
	if a.Food.ID < b.Food.ID {
		return -1
	}
	if a.Food.ID > b.Food.ID {
		return 1
	}

	return 0
}

// rankAgainstQuery scores one food name and its aliases against one generated
// search query, preferring exact matches before fuzzy and partial matches.
func rankAgainstQuery(name string, aliases []string, query string) (RankedMatch, bool) {
	normalizedName := normalizeMealName(name)
	normalizedQuery := normalizeMealName(query)

	// Empty strings should never produce a match.
	if normalizedName == "" || normalizedQuery == "" {
		return RankedMatch{}, false
	}

	// Best case: generated query exactly equals the canonical food name.
	// This is safe enough for the recommendation service's exact fast path,
	// as long as it is the only exact candidate in the group.
	if normalizedName == normalizedQuery {
		return RankedMatch{
			Kind:        interfaces.FoodMatchExactName,
			MatchedTerm: query,
			Score:       exactNameScore,
		}, true
	}

	// Second-best case: generated query exactly equals one of the food aliases.
	// This is also considered exact, but slightly lower than canonical name.
	for _, alias := range aliases {
		if normalizeMealName(alias) == normalizedQuery {
			return RankedMatch{
				Kind:        interfaces.FoodMatchExactAlias,
				MatchedTerm: query,
				Score:       exactAliasScore,
			}, true
		}
	}

	// Fuzzy and partial matching should check both canonical name and aliases.
	searchNames := make([]string, 0, 1+len(aliases))
	searchNames = append(searchNames, name)
	searchNames = append(searchNames, aliases...)

	var best RankedMatch
	found := false

	for _, candidateName := range searchNames {
		normalizedCandidate := normalizeMealName(candidateName)
		if normalizedCandidate == "" {
			continue
		}

		// Fuzzy match means all query words can be matched to candidate words.
		// This handles small typos but still requires the whole query to be
		// represented in the candidate.
		if fuzzyMealNameMatch(normalizedCandidate, normalizedQuery) {
			// Penalize length difference so a much longer candidate does not rank
			// the same as a tight candidate.
			score := fuzzyBaseScore - lengthPenalty(normalizedCandidate, normalizedQuery)

			match := RankedMatch{
				Kind:        interfaces.FoodMatchFuzzy,
				MatchedTerm: query,
				Score:       clampScore(score),
			}

			if !found || betterRankedMatch(match, best) {
				best = match
				found = true
			}
		}

		// Partial match means one phrase contains the other.
		// This keeps "Chicken Breakfast Sausage" eligible for "Chicken Sausage",
		// but ranks it lower than exact/fuzzy full-query matches.
		if partialPhraseMatch(normalizedCandidate, normalizedQuery) {
			// Penalize extra words more strongly for partial matches because
			// shared words alone are not enough to prove identity.
			score := partialBaseScore - extraTokenPenalty(normalizedCandidate, normalizedQuery)

			match := RankedMatch{
				Kind:        interfaces.FoodMatchPartial,
				MatchedTerm: query,
				Score:       clampScore(score),
			}

			if !found || betterRankedMatch(match, best) {
				best = match
				found = true
			}
		}
	}

	return best, found
}

// betterRankedMatch reports whether candidate a is a stronger lexical match
// than candidate b.
func betterRankedMatch(a, b RankedMatch) bool {
	// Score is the primary comparison.
	if a.Score != b.Score {
		return a.Score > b.Score
	}

	// Match kind is the secondary comparison.
	aPriority := matchKindPriority(a.Kind)
	bPriority := matchKindPriority(b.Kind)
	if aPriority != bPriority {
		return aPriority < bPriority
	}

	// Last tie-breaker for matches from different query terms.
	return normalizeMealName(a.MatchedTerm) < normalizeMealName(b.MatchedTerm)
}

// matchKindPriority returns a lower number for stronger match kinds so sorting
// can break score ties deterministically.
func matchKindPriority(kind interfaces.FoodMatchKind) int {
	switch kind {
	case interfaces.FoodMatchExactName:
		return 0
	case interfaces.FoodMatchExactAlias:
		return 1
	case interfaces.FoodMatchFuzzy:
		return 2
	case interfaces.FoodMatchPartial:
		return 3
	default:
		return 4
	}
}

// uniqueNonEmptyTerms normalizes generated meal search terms for deduplication
// while preserving the first original spelling for MatchedTerm.
func uniqueNonEmptyTerms(values []string) []string {
	terms := make([]string, 0, len(values))
	seen := make(map[string]bool)

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		normalized := normalizeMealName(trimmed)

		// Skip empty strings and duplicate terms such as :
		// "Nasi Lemak", "nasi   lemak", "NASI LEMAK"
		if normalized == "" || seen[normalized] {
			continue
		}

		seen[normalized] = true
		terms = append(terms, trimmed)
	}

	return terms
}

// normalizeMealName lowercases a meal name and collapses surrounding/repeated
// whitespace so lexical comparisons are stable.
func normalizeMealName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)

	// strings.Fields splits on any whitespace and removes repeated spaces/
	// strings.Join put it back with a single space.
	value = strings.Join(strings.Fields(value), " ")

	return value
}

// fuzzyMealNameMatch reports whether every query word can be matched to a
// distinct candidate word using exact or small edit-distance matching.
func fuzzyMealNameMatch(candidate, query string) bool {
	candidateWords := strings.Fields(normalizeMealName(candidate))
	queryWords := strings.Fields(normalizeMealName(query))

	if len(candidateWords) == 0 || len(queryWords) == 0 {
		return false
	}

	// Prevent one candidate word from satisfying multiple query words.
	// E.g. query "chicken chicken" should not match candidate "chicken".
	usedCandidateWords := make([]bool, len(candidateWords))
	for _, queryWord := range queryWords {
		matched := false

		for index, candidateWord := range candidateWords {
			if usedCandidateWords[index] {
				continue
			}

			if mealWordFuzzyMatch(candidateWord, queryWord) {
				usedCandidateWords[index] = true
				matched = true
				break
			}
		}

		// Every query word must be represented
		if !matched {
			return false
		}
	}
	return true
}

// mealWordFuzzyMatch reports whether two words are equal or close enough by
// edit distance to count as the same meal-search token.
func mealWordFuzzyMatch(candidateWord, queryWord string) bool {
	if candidateWord == queryWord {
		return true
	}

	// Very short words are risky to fuzzy-match
	// E.g. "mee" vs "bee" has distance 1 but means different thing.
	if len(queryWord) < 4 || len(candidateWord) < 4 {
		return false
	}

	allowedDistance := 1

	// Longer words can tolerate slightly more typos.
	// E.g. "spagheti" vs "spaghetti"
	if len(queryWord) >= 8 {
		allowedDistance = 2
	}

	return levenshteinDistance(candidateWord, queryWord) <= allowedDistance
}

// partialPhraseMatch reports whether one normalized phrase contains the
// other and both sides are specific enough to avoid one-word broad matches.
func partialPhraseMatch(candidate, query string) bool {
	normalizedCandidate := normalizeMealName(candidate)
	normalizedQuery := normalizeMealName(query)

	if normalizedCandidate == "" || normalizedQuery == "" {
		return false
	}

	candidateWordCount := wordCount(normalizedCandidate)
	queryWordCount := wordCount(normalizedQuery)

	// Query appears inside candidate.
	// E.g. query "chicken sausage", candidate "chicken breakfast sausage"
	if strings.Contains(normalizedCandidate, normalizedQuery) && queryWordCount >= 2 {
		return true
	}

	// Candidate appears inside query.
	// E.g. query "fried rice chicken", candidate "fried rice"
	if strings.Contains(normalizedQuery, normalizedCandidate) && candidateWordCount >= 2 {
		return true
	}

	return false
}

// lengthPenalty slightly lowers fuzzy scores when candidate and query word counts differ,
// without removing plausible ambiguous candidates.
func lengthPenalty(candidate, query string) float64 {
	diff := math.Abs(float64(wordCount(candidate) - wordCount(query)))

	// Each extra/missing word costs 0.02, capped at 0.10
	return math.Min(diff*0.02, 0.10)
}

// extraTokenPenalty lowers partial-match scores when one phrase has extra words,
// making exact and fuzzy full-query matches rank higher.
func extraTokenPenalty(candidate, query string) float64 {
	diff := math.Abs(float64(wordCount(candidate) - wordCount(query)))

	// Partial matches are weaker, so extra words cost more here.
	return math.Min(diff*0.04, 0.20)
}

func wordCount(value string) int {
	normalized := normalizeMealName(value)

	if normalized == "" {
		return 0
	}

	return len(strings.Fields(normalized))
}

// clampScore keeps computed match scores inside the expected 0.0 to 1.0 range.
func clampScore(score float64) float64 {
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}

	return score
}

// levenshteinDistance returns the edit distance between two strings.
func levenshteinDistance(a, b string) int {
	if a == b {
		return 0
	}

	if len(a) == 0 {
		return len(b)
	}

	if len(b) == 0 {
		return len(a)
	}

	// previous and current hold two rows of the dynamic-programming table.
	// We only need the previous row to calculate the current row, so this avoids
	// allocating a full len(a) * len(b) matrix.
	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)

	// Transforming an empty string into b[:j] costs j insertions.
	for j := range previous {
		previous[j] = j
	}

	for i := 1; i <= len(a); i++ {
		// Transforming a[:i] into an empty string costs i deletions.
		current[0] = i

		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}

			// Minimum of:
			// - delete from a
			// - insert into a
			// - substitute one character
			current[j] = minInt(
				previous[j]+1,
				current[j-1]+1,
				previous[j-1]+cost,
			)
		}

		// Reuse slices by swapping rows.
		previous, current = current, previous
	}

	return previous[len(b)]
}

// minInt returns the smallest integer from a non-empty list.
func minInt(values ...int) int {
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}
