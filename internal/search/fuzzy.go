package search

import (
	"sort"
	"strings"
	"unicode"
)

// SearchResult represents a matched item with its score
type SearchResult struct {
	ID          string
	Type        string // "finding", "unknown", "dead_end"
	Text        string // Primary text (finding/unknown/approach)
	SecondaryText string // Secondary text (why_failed for dead ends)
	Scope       string
	Score       float64
	Highlights  []int // Indices of matching characters (for UI highlighting)
}

// SearchItem represents an item to be searched
type SearchItem struct {
	ID            string
	Type          string
	Text          string
	SecondaryText string
	Scope         string
}

// FuzzySearch performs fuzzy matching on a list of items
// Returns results sorted by score (highest first)
func FuzzySearch(query string, items []SearchItem, threshold float64) []SearchResult {
	if query == "" {
		return nil
	}

	query = strings.ToLower(strings.TrimSpace(query))
	queryTokens := tokenize(query)

	var results []SearchResult

	for _, item := range items {
		score, highlights := scoreItem(queryTokens, item)
		if score >= threshold {
			results = append(results, SearchResult{
				ID:            item.ID,
				Type:          item.Type,
				Text:          item.Text,
				SecondaryText: item.SecondaryText,
				Scope:         item.Scope,
				Score:         score,
				Highlights:    highlights,
			})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results
}

// tokenize splits a query into searchable tokens
func tokenize(s string) []string {
	s = strings.ToLower(s)
	// Split on whitespace and common separators
	var tokens []string
	var current strings.Builder

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// scoreItem calculates how well an item matches the query tokens
func scoreItem(queryTokens []string, item SearchItem) (float64, []int) {
	if len(queryTokens) == 0 {
		return 0, nil
	}

	textLower := strings.ToLower(item.Text)
	secondaryLower := strings.ToLower(item.SecondaryText)
	scopeLower := strings.ToLower(item.Scope)

	var totalScore float64
	var allHighlights []int
	matchedTokens := 0

	for _, token := range queryTokens {
		tokenScore, highlights := scoreToken(token, textLower, secondaryLower, scopeLower)
		if tokenScore > 0 {
			matchedTokens++
			totalScore += tokenScore
			allHighlights = append(allHighlights, highlights...)
		}
	}

	// Require all tokens to match for good results
	if matchedTokens < len(queryTokens) {
		// Penalize partial token matches significantly
		totalScore *= float64(matchedTokens) / float64(len(queryTokens)) * 0.5
	}

	// Normalize by number of tokens
	if len(queryTokens) > 0 {
		totalScore /= float64(len(queryTokens))
	}

	return totalScore, allHighlights
}

// scoreToken calculates score for a single token against text fields
func scoreToken(token, text, secondary, scope string) (float64, []int) {
	var score float64
	var highlights []int

	// Exact word match (highest score)
	if containsWord(text, token) {
		score = 1.0
		if idx := strings.Index(text, token); idx >= 0 {
			for i := idx; i < idx+len(token); i++ {
				highlights = append(highlights, i)
			}
		}
	} else if strings.Contains(text, token) {
		// Substring match (good score)
		score = 0.7
		if idx := strings.Index(text, token); idx >= 0 {
			for i := idx; i < idx+len(token); i++ {
				highlights = append(highlights, i)
			}
		}
	} else if fuzzyContains(text, token) {
		// Fuzzy substring match (moderate score)
		score = 0.4
	}

	// Check secondary text (lower weight)
	if secondary != "" {
		if containsWord(secondary, token) {
			score = max(score, 0.6)
		} else if strings.Contains(secondary, token) {
			score = max(score, 0.4)
		} else if fuzzyContains(secondary, token) {
			score = max(score, 0.2)
		}
	}

	// Check scope (even lower weight, but helpful)
	if scope != "" {
		if strings.Contains(scope, token) {
			score = max(score, 0.3)
		}
	}

	return score, highlights
}

// containsWord checks if text contains token as a whole word
func containsWord(text, word string) bool {
	idx := strings.Index(text, word)
	if idx == -1 {
		return false
	}

	// Check word boundary before
	if idx > 0 {
		r := rune(text[idx-1])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
	}

	// Check word boundary after
	endIdx := idx + len(word)
	if endIdx < len(text) {
		r := rune(text[endIdx])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
	}

	return true
}

// fuzzyContains checks if text contains characters of pattern in order
// with limited gaps (allows for typos and abbreviations)
func fuzzyContains(text, pattern string) bool {
	if len(pattern) == 0 {
		return true
	}
	if len(text) == 0 {
		return false
	}

	patternIdx := 0
	gaps := 0
	maxGaps := len(pattern) // Allow gaps proportional to pattern length

	for i := 0; i < len(text) && patternIdx < len(pattern); i++ {
		if text[i] == pattern[patternIdx] {
			patternIdx++
			gaps = 0 // Reset gap counter on match
		} else if patternIdx > 0 {
			gaps++
			if gaps > maxGaps {
				return false
			}
		}
	}

	return patternIdx == len(pattern)
}

// max returns the larger of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
