package service

import (
	"backend-go/internal/llm"
	"backend-go/internal/models"
	"backend-go/internal/state"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AISemanticMatcher uses LLM for intelligent column matching
type AISemanticMatcher struct {
	llmService     *llm.Service
	contextService *ContextService
	cache          map[string]*SemanticMatch
	cacheMutex     sync.RWMutex
	cacheExpiry    time.Duration
}

// SemanticMatch represents an AI-determined match
type SemanticMatch struct {
	File1Column    string    `json:"file1_column"`
	File2Column    string    `json:"file2_column"`
	Confidence     float64   `json:"confidence"`
	Reason         string    `json:"reason"`
	MatchType      string    `json:"match_type"`
	AIExplanation  string    `json:"ai_explanation,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	
	// Enhanced metrics
	NameSimilarity         float64 `json:"name_similarity"`
	DataSimilarity         float64 `json:"data_similarity"`
	SemanticScore          float64 `json:"semantic_score"`
	DistributionSimilarity float64 `json:"distribution_similarity"`
	ValueOverlap           float64 `json:"value_overlap"`
}

// NewAISemanticMatcher creates a new AI-powered matcher
func NewAISemanticMatcher(llmSvc *llm.Service, ctxSvc *ContextService) *AISemanticMatcher {
	return &AISemanticMatcher{
		llmService:     llmSvc,
		contextService: ctxSvc,
		cache:          make(map[string]*SemanticMatch),
		cacheExpiry:    30 * time.Minute,
	}
}

// MatchColumns performs AI-powered column matching
func (m *AISemanticMatcher) MatchColumns(
	df1, df2 *state.DataFrame,
	ctx1, ctx2 *models.Context,
) []SemanticMatch {
	results := []SemanticMatch{}

	// Step 1: Quick heuristic pre-filtering
	candidates := m.preFilterCandidates(df1, df2)
	log.Printf("[AI Matcher] Found %d candidate pairs from heuristics", len(candidates))

	// Step 2: Use LLM for semantic matching on column names
	llmMatches, err := m.getLLMSemanticMatches(df1.Headers, df2.Headers)
	if err != nil {
		log.Printf("[AI Matcher] LLM matching failed, falling back to heuristics: %v", err)
	} else {
		log.Printf("[AI Matcher] LLM found %d semantic matches", len(llmMatches))
		// Merge LLM matches with candidates
		for _, match := range llmMatches {
			candidates[match.File1Column+"||"+match.File2Column] = &match
		}
	}

	// Step 3: Enhance each candidate with data analysis
	for key, match := range candidates {
		parts := strings.Split(key, "||")
		if len(parts) != 2 {
			continue
		}
		col1, col2 := parts[0], parts[1]

		// Get column indices
		col1Idx := getColIndex(df1.Headers, col1)
		col2Idx := getColIndex(df2.Headers, col2)

		if col1Idx < 0 || col2Idx < 0 {
			continue
		}

		// Enhance with data analysis
		enhanced := m.enhanceWithDataAnalysis(df1, df2, col1Idx, col2Idx, match)

		// Apply context boost if available
		if ctx1 != nil && ctx2 != nil {
			enhanced = m.applyContextBoost(enhanced, ctx1, ctx2)
		}

		// Only include meaningful matches
		if enhanced.Confidence > 15 {
			results = append(results, *enhanced)
		}
	}

	// Sort by confidence
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	return results
}

// preFilterCandidates uses quick heuristics to identify potential matches
func (m *AISemanticMatcher) preFilterCandidates(df1, df2 *state.DataFrame) map[string]*SemanticMatch {
	candidates := make(map[string]*SemanticMatch)

	for _, col1 := range df1.Headers {
		for _, col2 := range df2.Headers {
			// Quick name similarity check
			nameSim := calculateNameSimilarity(col1, col2)
			
			if nameSim > 0.3 {
				key := col1 + "||" + col2
				candidates[key] = &SemanticMatch{
					File1Column:    col1,
					File2Column:    col2,
					NameSimilarity: nameSim,
					Confidence:     nameSim * 50, // Initial confidence based on name
					MatchType:      "heuristic",
					Reason:         fmt.Sprintf("Name similarity: %.0f%%", nameSim*100),
				}
			}
		}
	}

	return candidates
}

// getLLMSemanticMatches uses the LLM for semantic matching
func (m *AISemanticMatcher) getLLMSemanticMatches(cols1, cols2 []string) ([]SemanticMatch, error) {
	if m.llmService == nil {
		return nil, fmt.Errorf("LLM service not configured")
	}

	// Check cache for recent matches
	cacheKey := strings.Join(cols1, ",") + "||" + strings.Join(cols2, ",")
	m.cacheMutex.RLock()
	if cached, ok := m.cache[cacheKey]; ok {
		if time.Since(cached.Timestamp) < m.cacheExpiry {
			m.cacheMutex.RUnlock()
			return []SemanticMatch{*cached}, nil
		}
	}
	m.cacheMutex.RUnlock()

	// Call LLM
	matches, err := m.llmService.GetSemanticMatches(cols1, cols2)
	if err != nil {
		return nil, err
	}

	// Convert to SemanticMatch
	results := []SemanticMatch{}
	for _, match := range matches {
		sm := SemanticMatch{
			File1Column:   match.ColA,
			File2Column:   match.ColB,
			Confidence:    match.Confidence * 100, // Convert to percentage
			Reason:        match.Reason,
			AIExplanation: match.Reason,
			MatchType:     "ai_semantic",
			SemanticScore: match.Confidence,
			Timestamp:     time.Now(),
		}
		results = append(results, sm)
	}

	return results, nil
}

// enhanceWithDataAnalysis adds data-level similarity metrics
func (m *AISemanticMatcher) enhanceWithDataAnalysis(
	df1, df2 *state.DataFrame,
	col1Idx, col2Idx int,
	match *SemanticMatch,
) *SemanticMatch {
	if match == nil {
		match = &SemanticMatch{}
	}

	result := *match // Copy

	// Check if numeric
	numericCols1 := df1.GetNumericColumnIndices()
	numericCols2 := df2.GetNumericColumnIndices()

	isNum1 := numericCols1[col1Idx]
	isNum2 := numericCols2[col2Idx]

	if isNum1 && isNum2 {
		// Numeric: distribution similarity
		result.DistributionSimilarity = calculateDistributionSim(df1, df2, col1Idx, col2Idx)
		result.DataSimilarity = result.DistributionSimilarity
	} else if !isNum1 && !isNum2 {
		// Categorical: value overlap
		result.ValueOverlap = calculateValueOverlapSim(df1, df2, col1Idx, col2Idx)
		result.DataSimilarity = result.ValueOverlap
	}

	// Recalculate confidence with data
	// Weights: Name 30%, Semantic 40%, Data 30%
	result.Confidence = (result.NameSimilarity * 30) +
		(result.SemanticScore * 40) +
		(result.DataSimilarity * 30)

	// Boost for high semantic matches from AI
	if result.MatchType == "ai_semantic" && result.SemanticScore > 0.7 {
		result.Confidence = math.Min(100, result.Confidence*1.2)
	}

	// Update match type
	if result.SemanticScore > 0.5 {
		result.MatchType = "ai_semantic"
	} else if result.DataSimilarity > 0.5 {
		result.MatchType = "data_match"
	} else if result.NameSimilarity > 0.5 {
		result.MatchType = "name_match"
	}

	return &result
}

// applyContextBoost applies context-aware adjustments
func (m *AISemanticMatcher) applyContextBoost(match *SemanticMatch, ctx1, ctx2 *models.Context) *SemanticMatch {
	result := *match

	// Custom mapping override
	if target, ok := ctx1.CustomMappings[match.File1Column]; ok {
		if target == match.File2Column {
			result.Confidence = 98
			result.MatchType = "custom_mapping"
			result.Reason = "User-defined custom mapping"
			return &result
		}
	}

	// Business domain boost
	if ctx1.BusinessDomain != "" && ctx1.BusinessDomain == ctx2.BusinessDomain {
		result.Confidence = math.Min(100, result.Confidence*1.1)
	}

	// Key entity boost
	col1Lower := strings.ToLower(match.File1Column)
	col2Lower := strings.ToLower(match.File2Column)
	for _, entity := range ctx1.KeyEntities {
		entityLower := strings.ToLower(entity)
		if strings.Contains(col1Lower, entityLower) || strings.Contains(col2Lower, entityLower) {
			result.Confidence = math.Min(100, result.Confidence*1.15)
			break
		}
	}

	return &result
}

// AskAIForMatch asks the LLM about a specific column pair
func (m *AISemanticMatcher) AskAIForMatch(col1, col2 string, sampleData1, sampleData2 []string) (*SemanticMatch, error) {
	if m.llmService == nil {
		return nil, fmt.Errorf("LLM service not configured")
	}

	prompt := fmt.Sprintf(`You are a data integration expert. Analyze if these two columns likely represent the same concept.

Column 1 name: "%s"
Sample values: %v

Column 2 name: "%s"  
Sample values: %v

Respond with JSON only:
{
  "is_match": true/false,
  "confidence": 0.0-1.0,
  "reason": "explanation why they match or don't match",
  "match_type": "exact|semantic|partial|none"
}`, col1, sampleData1[:minInt(5, len(sampleData1))], col2, sampleData2[:minInt(5, len(sampleData2))])

	response, err := m.llmService.CallOllama(prompt)
	if err != nil {
		return nil, err
	}

	// Parse response (simplified - in production use proper JSON extraction)
	match := &SemanticMatch{
		File1Column:   col1,
		File2Column:   col2,
		AIExplanation: response,
		MatchType:     "ai_analyzed",
		Timestamp:     time.Now(),
	}

	// Extract confidence (basic parsing)
	if strings.Contains(strings.ToLower(response), "\"is_match\": true") ||
		strings.Contains(strings.ToLower(response), "\"is_match\":true") {
		match.Confidence = 70
		match.SemanticScore = 0.7
	}

	return match, nil
}

// Helper functions

func getColIndex(headers []string, col string) int {
	for i, h := range headers {
		if h == col {
			return i
		}
	}
	return -1
}

func calculateNameSimilarity(col1, col2 string) float64 {
	// Normalize
	n1 := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(col1, "_", ""), "-", ""))
	n2 := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(col2, "_", ""), "-", ""))

	// Exact match
	if n1 == n2 {
		return 1.0
	}

	// Containment
	if strings.Contains(n1, n2) || strings.Contains(n2, n1) {
		return 0.8
	}

	// Levenshtein
	return LevenshteinRatio(col1, col2)
}

func calculateDistributionSim(df1, df2 *state.DataFrame, col1Idx, col2Idx int) float64 {
	vals1 := getFloatVals(df1, col1Idx)
	vals2 := getFloatVals(df2, col2Idx)

	if len(vals1) < 5 || len(vals2) < 5 {
		return 0
	}

	mean1, std1 := calcMeanStd(vals1)
	mean2, std2 := calcMeanStd(vals2)

	// CV similarity
	cv1, cv2 := 0.0, 0.0
	if mean1 != 0 {
		cv1 = std1 / math.Abs(mean1)
	}
	if mean2 != 0 {
		cv2 = std2 / math.Abs(mean2)
	}

	cvSim := math.Max(0, 1-math.Abs(cv1-cv2))
	return cvSim
}

func calculateValueOverlapSim(df1, df2 *state.DataFrame, col1Idx, col2Idx int) float64 {
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	limit := 200
	for i := 0; i < minInt(limit, len(df1.Rows)); i++ {
		if col1Idx < len(df1.Rows[i]) && df1.Rows[i][col1Idx] != "" {
			set1[strings.ToLower(df1.Rows[i][col1Idx])] = true
		}
	}
	for i := 0; i < minInt(limit, len(df2.Rows)); i++ {
		if col2Idx < len(df2.Rows[i]) && df2.Rows[i][col2Idx] != "" {
			set2[strings.ToLower(df2.Rows[i][col2Idx])] = true
		}
	}

	if len(set1) == 0 || len(set2) == 0 {
		return 0
	}

	intersection := 0
	for k := range set1 {
		if set2[k] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

func getFloatVals(df *state.DataFrame, colIdx int) []float64 {
	vals := []float64{}
	for _, row := range df.Rows {
		if colIdx < len(row) {
			if v, err := strconv.ParseFloat(row[colIdx], 64); err == nil {
				vals = append(vals, v)
			}
		}
	}
	return vals
}

func calcMeanStd(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	mean := sum / float64(len(vals))

	sumSq := 0.0
	for _, v := range vals {
		sumSq += (v - mean) * (v - mean)
	}
	std := math.Sqrt(sumSq / float64(len(vals)))

	return mean, std
}

// minInt returns the smaller of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

