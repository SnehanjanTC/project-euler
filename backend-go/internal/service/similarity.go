package service

import (
	"backend-go/internal/models"
	"strings"
)

type SimilarityService struct {
	ContextService *ContextService
}

func NewSimilarityService(ctxService *ContextService) *SimilarityService {
	return &SimilarityService{
		ContextService: ctxService,
	}
}

// LevenshteinRatio calculates similarity ratio (0-1)
func LevenshteinRatio(s1, s2 string) float64 {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)
	distance := levenshtein(s1, s2)
	maxLen := float64(max(len(s1), len(s2)))
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - (float64(distance) / maxLen)
}

func levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	row := make([]int, len2+1)
	for i := 0; i <= len2; i++ {
		row[i] = i
	}

	for i := 1; i <= len1; i++ {
		prev := i
		for j := 1; j <= len2; j++ {
			val := row[j]
			if r1[i-1] == r2[j-1] {
				val = row[j-1]
			} else {
				val = min(min(row[j-1]+1, prev+1), row[j]+1)
			}
			row[j-1] = prev
			prev = val
		}
		row[len2] = prev
	}
	return row[len2]
}

// Function to calculate confidence score
func (s *SimilarityService) CalculateConfidence(col1, col2 string, ctx1, ctx2 *models.Context) float64 {
	// Base score: Levenshtein
	score := LevenshteinRatio(col1, col2) * 100

	// Boost based on Jaccard of tokens (simple tokenization)
	tokens1 := strings.Fields(strings.ReplaceAll(col1, "_", " "))
	tokens2 := strings.Fields(strings.ReplaceAll(col2, "_", " "))
	jaccard := jaccardSimilarity(tokens1, tokens2)

	score = (score * 0.7) + (jaccard * 100 * 0.3)

	// Context Enhancements
	if ctx1 != nil && ctx2 != nil {
		// Custom mapping check
		if target, ok := ctx1.CustomMappings[col1]; ok && target == col2 {
			return 95.0
		}

		// Domain boost
		if ctx1.BusinessDomain != "" && ctx1.BusinessDomain == ctx2.BusinessDomain {
			score *= 1.1
		}
	}

	if score > 100 {
		score = 100
	}
	return score
}

func jaccardSimilarity(s1, s2 []string) float64 {
	map1 := make(map[string]bool)
	for _, s := range s1 {
		map1[strings.ToLower(s)] = true
	}

	intersection := 0
	union := len(map1)

	for _, s := range s2 {
		lower := strings.ToLower(s)
		if map1[lower] {
			intersection++
		} else {
			union++
		}
	}

	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
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
