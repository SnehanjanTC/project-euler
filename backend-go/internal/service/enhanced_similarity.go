package service

import (
	"backend-go/internal/models"
	"backend-go/internal/state"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// EnhancedSimilarityService provides advanced column matching capabilities
type EnhancedSimilarityService struct {
	contextService    *ContextService
	synonyms          map[string][]string
	patterns          map[string]*regexp.Regexp
	normalizedMatcher *NormalizedValueMatcher
	qualityProfiler   *DataQualityProfiler
}

// NewEnhancedSimilarityService creates a new enhanced similarity service
func NewEnhancedSimilarityService(ctx *ContextService) *EnhancedSimilarityService {
	svc := &EnhancedSimilarityService{
		contextService:    ctx,
		synonyms:          buildSynonymMap(),
		patterns:          buildPatternMap(),
		normalizedMatcher: NewNormalizedValueMatcher(),
		qualityProfiler:   NewDataQualityProfiler(),
	}
	return svc
}

// buildSynonymMap creates a map of common synonyms for column names
func buildSynonymMap() map[string][]string {
	return map[string][]string{
		// Price/Money
		"price":   {"cost", "amount", "value", "fee", "charge", "rate", "total"},
		"cost":    {"price", "amount", "value", "fee", "charge", "expense"},
		"amount":  {"price", "cost", "value", "total", "sum", "quantity"},
		"revenue": {"income", "sales", "earnings", "profit"},

		// Identity
		"id":         {"identifier", "key", "code", "number", "num", "no"},
		"identifier": {"id", "key", "code", "number"},
		"code":       {"id", "identifier", "key", "number"},
		"number":     {"id", "num", "no", "code"},

		// Person
		"name":      {"fullname", "title", "label", "description"},
		"firstname": {"first", "fname", "givenname", "given"},
		"lastname":  {"last", "lname", "surname", "family"},
		"email":     {"mail", "emailaddress", "emailid"},
		"phone":     {"mobile", "cell", "telephone", "tel", "contact"},

		// Address
		"address": {"location", "addr", "street"},
		"city":    {"town", "municipality", "place"},
		"state":   {"province", "region", "territory"},
		"country": {"nation", "countrycode"},
		"zip":     {"zipcode", "postal", "postalcode", "pincode"},

		// Date/Time
		"date":      {"datetime", "timestamp", "time", "dt"},
		"created":   {"createdat", "createddate", "creationdate", "created_at"},
		"updated":   {"updatedat", "updateddate", "modifieddate", "modified", "updated_at"},
		"startdate": {"start", "begin", "from", "fromdate"},
		"enddate":   {"end", "finish", "to", "todate"},

		// Status
		"status": {"state", "condition", "flag", "active", "enabled"},
		"type":   {"category", "kind", "class", "classification"},

		// Quantity
		"quantity": {"qty", "count", "num", "amount", "units"},
		"count":    {"quantity", "qty", "total", "num"},

		// Description
		"description": {"desc", "details", "info", "notes", "comment", "remarks"},
		"notes":       {"description", "comments", "remarks", "memo"},
	}
}

// buildPatternMap creates regex patterns for common data formats
func buildPatternMap() map[string]*regexp.Regexp {
	return map[string]*regexp.Regexp{
		"email":    regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
		"phone":    regexp.MustCompile(`^[\+]?[(]?[0-9]{1,4}[)]?[-\s\./0-9]*$`),
		"uuid":     regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`),
		"date_iso": regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
		"date_us":  regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`),
		"ip":       regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`),
		"url":      regexp.MustCompile(`^https?://`),
		"currency": regexp.MustCompile(`^[\$€£¥₹]?\s*\d+([,.]\d{2})?$`),
		"zipcode":  regexp.MustCompile(`^\d{5}(-\d{4})?$`),
	}
}

// SimilarityResult holds the comprehensive similarity analysis
type SimilarityResult struct {
	File1Column            string  `json:"file1_column"`
	File2Column            string  `json:"file2_column"`
	Similarity             float64 `json:"similarity"`
	Confidence             float64 `json:"confidence"`
	Type                   string  `json:"type"`
	DataSimilarity         float64 `json:"data_similarity"`
	NameSimilarity         float64 `json:"name_similarity"`
	DistributionSimilarity float64 `json:"distribution_similarity"`
	JSONConfidence         float64 `json:"json_confidence"`
	LLMSemanticScore       float64 `json:"llm_semantic_score"`
	Reason                 string  `json:"reason,omitempty"`

	// Enhanced metrics
	TokenSimilarity float64 `json:"token_similarity"`
	SynonymMatch    bool    `json:"synonym_match"`
	PatternMatch    string  `json:"pattern_match,omitempty"`
	ValueOverlap    float64 `json:"value_overlap"`
}

// CalculateEnhancedSimilarity performs comprehensive similarity analysis
func (s *EnhancedSimilarityService) CalculateEnhancedSimilarity(
	df1, df2 *state.DataFrame,
	ctx1, ctx2 *models.Context,
) []SimilarityResult {
	results := []SimilarityResult{}

	for col1Idx, col1 := range df1.Headers {
		for col2Idx, col2 := range df2.Headers {
			result := s.compareColumns(df1, df2, col1Idx, col2Idx, col1, col2, ctx1, ctx2)

			// Only include if has meaningful similarity
			if result.Confidence > 10 {
				results = append(results, result)
			}
		}
	}

	// Sort by confidence
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	return results
}

// compareColumns performs detailed comparison between two columns
func (s *EnhancedSimilarityService) compareColumns(
	df1, df2 *state.DataFrame,
	col1Idx, col2Idx int,
	col1, col2 string,
	ctx1, ctx2 *models.Context,
) SimilarityResult {
	result := SimilarityResult{
		File1Column: col1,
		File2Column: col2,
	}

	// 1. Tokenized Name Similarity
	tokenSim, isSynonym := s.calculateTokenSimilarity(col1, col2)
	result.TokenSimilarity = tokenSim
	result.SynonymMatch = isSynonym
	result.NameSimilarity = tokenSim

	// 2. Pattern Detection
	pattern1 := s.detectPattern(df1, col1Idx)
	pattern2 := s.detectPattern(df2, col2Idx)
	patternScore := 0.0
	if pattern1 != "" && pattern1 == pattern2 {
		patternScore = 0.9
		result.PatternMatch = pattern1
	}
	result.JSONConfidence = patternScore

	// 3. Data Quality Profiling (NEW)
	profile1 := s.qualityProfiler.ProfileColumn(df1, col1Idx)
	profile2 := s.qualityProfiler.ProfileColumn(df2, col2Idx)
	qualityMatch := s.qualityProfiler.CompareQuality(profile1, profile2)

	// 4. Cardinality Analysis (NEW)
	cardinalityMatch := s.normalizedMatcher.CalculateCardinalityMatch(profile1, profile2)

	// 5. Format Normalization & Value Matching (NEW)
	normalizedMatch := s.normalizedMatcher.CalculateNormalizedMatch(df1, df2, col1Idx, col2Idx)
	formatTransform, formatType := s.normalizedMatcher.DetectFormatTransformation(df1, df2, col1Idx, col2Idx)

	// 6. Traditional Value Overlap (for categorical) or Distribution (for numeric)
	numericCols1 := df1.GetNumericColumnIndices()
	numericCols2 := df2.GetNumericColumnIndices()
	isNum1, isNum2 := numericCols1[col1Idx], numericCols2[col2Idx]

	if isNum1 && isNum2 {
		// Numeric: distribution similarity
		result.DistributionSimilarity = s.calculateDistributionSimilarity(df1, df2, col1Idx, col2Idx)
		result.DataSimilarity = result.DistributionSimilarity
	} else if !isNum1 && !isNum2 {
		// Categorical: use normalized match if better than raw overlap
		rawOverlap := s.calculateValueOverlap(df1, df2, col1Idx, col2Idx)
		result.ValueOverlap = math.Max(rawOverlap, normalizedMatch)
		result.DataSimilarity = result.ValueOverlap
	}

	// 7. Get adaptive weights
	adaptiveLearner := GetAdaptiveLearner()
	weights := adaptiveLearner.GetWeights()

	// 8. Calculate Final Confidence using ENHANCED weights
	// Include new signals: quality, cardinality, normalized matching
	result.Confidence = (result.NameSimilarity * weights.Name * 100) +
		(result.DataSimilarity * weights.Data * 100) +
		(patternScore * weights.Pattern * 100) +
		(result.LLMSemanticScore * weights.LLM * 100) +
		(qualityMatch * 10) + // NEW: Quality boost up to 10%
		(cardinalityMatch * 15) + // NEW: Cardinality boost up to 15%
		(normalizedMatch * 10) // NEW: Normalized match boost up to 10%

	// 9. Format transformation bonus (NEW)
	if formatTransform {
		result.Confidence = math.Min(100, result.Confidence*1.25) // 25% boost for format matches
		result.PatternMatch = formatType + "_transform"
	}

	// 10. Apply learned boosts from feedback
	feedbackSystem := GetFeedbackSystem()
	feedbackBoost := feedbackSystem.GetLearnedBoost(col1, col2)
	result.Confidence += feedbackBoost * 100

	// 11. Apply pattern learning boost
	patternLearner := GetPatternLearner()
	patternBoost := patternLearner.GetPatternBoost(col1, col2)
	result.Confidence += patternBoost * 100

	// 12. Boost for synonym matches
	if isSynonym {
		result.Confidence = math.Min(100, result.Confidence*1.2)
	}

	// 13. Primary key matching bonus (NEW)
	if profile1.IsPrimaryKey && profile2.IsPrimaryKey && normalizedMatch > 0.5 {
		result.Confidence = math.Min(100, result.Confidence*1.3) // Strong boost for matching PKs
	}

	// 14. Context boost
	if ctx1 != nil && ctx2 != nil {
		result.Confidence = s.applyContextBoost(result.Confidence, col1, col2, ctx1, ctx2)
	}

	// 15. Apply confidence calibration
	calibrator := GetConfidenceCalibrator()
	result.Confidence = calibrator.Calibrate(result.Confidence)

	// Clamp to 0-100
	if result.Confidence < 0 {
		result.Confidence = 0
	}
	if result.Confidence > 100 {
		result.Confidence = 100
	}

	// Determine similarity type
	result.Type = s.determineType(result)
	result.Similarity = result.Confidence / 100

	// Build ENHANCED reason string (NEW)
	result.Reason = s.normalizedMatcher.ExplainMatch(
		col1, col2,
		result.NameSimilarity,
		result.DataSimilarity,
		normalizedMatch,
		cardinalityMatch,
		profile1, profile2,
		formatTransform,
		formatType,
	)

	return result
}

// calculateTokenSimilarity compares tokenized column names with synonym matching
func (s *EnhancedSimilarityService) calculateTokenSimilarity(col1, col2 string) (float64, bool) {
	// Normalize and tokenize
	tokens1 := tokenize(col1)
	tokens2 := tokenize(col2)

	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0, false
	}

	// Exact match
	if strings.EqualFold(normalize(col1), normalize(col2)) {
		return 1.0, false
	}

	// Token set comparison
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	for _, t := range tokens1 {
		set1[t] = true
	}
	for _, t := range tokens2 {
		set2[t] = true
	}

	// Direct token overlap
	intersection := 0
	for t := range set1 {
		if set2[t] {
			intersection++
		}
	}

	// Synonym matching
	synonymMatch := false
	for t1 := range set1 {
		if synonyms, ok := s.synonyms[t1]; ok {
			for _, syn := range synonyms {
				if set2[syn] {
					intersection++
					synonymMatch = true
					break
				}
			}
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0, false
	}

	// Jaccard similarity of tokens
	jaccardSim := float64(intersection) / float64(union)

	// Also consider Levenshtein for partial matches
	levenSim := LevenshteinRatio(col1, col2)

	// Combine both
	finalSim := math.Max(jaccardSim, levenSim)

	return finalSim, synonymMatch
}

// tokenize splits a column name into normalized tokens
func tokenize(name string) []string {
	// Convert to lowercase
	name = strings.ToLower(name)

	// Split by common separators
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, ".", " ")

	// Split camelCase
	var result []rune
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, ' ')
		}
		result = append(result, unicode.ToLower(r))
	}
	name = string(result)

	// Split and filter
	tokens := strings.Fields(name)
	filtered := []string{}
	for _, t := range tokens {
		if len(t) > 1 { // Skip single chars
			filtered = append(filtered, t)
		}
	}

	return filtered
}

// normalize removes common prefixes/suffixes and normalizes
func normalize(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, " ", "")
	return name
}

// detectPattern analyzes sample data to detect format patterns
func (s *EnhancedSimilarityService) detectPattern(df *state.DataFrame, colIdx int) string {
	if len(df.Rows) == 0 {
		return ""
	}

	// Sample up to 50 rows
	sampleSize := 50
	if len(df.Rows) < sampleSize {
		sampleSize = len(df.Rows)
	}

	// Count pattern matches
	patternCounts := make(map[string]int)
	for i := 0; i < sampleSize; i++ {
		if colIdx >= len(df.Rows[i]) {
			continue
		}
		val := df.Rows[i][colIdx]
		if val == "" {
			continue
		}

		for name, pattern := range s.patterns {
			if pattern.MatchString(val) {
				patternCounts[name]++
				break // One pattern per value
			}
		}
	}

	// Find dominant pattern (must match at least 60% of samples)
	threshold := int(float64(sampleSize) * 0.6)
	for name, count := range patternCounts {
		if count >= threshold {
			return name
		}
	}

	return ""
}

// calculateValueOverlap computes Jaccard similarity of unique values
func (s *EnhancedSimilarityService) calculateValueOverlap(df1, df2 *state.DataFrame, col1Idx, col2Idx int) float64 {
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	// Sample up to 500 rows
	limit1 := 500
	if len(df1.Rows) < limit1 {
		limit1 = len(df1.Rows)
	}
	for i := 0; i < limit1; i++ {
		if col1Idx < len(df1.Rows[i]) && df1.Rows[i][col1Idx] != "" {
			set1[strings.ToLower(df1.Rows[i][col1Idx])] = true
		}
	}

	limit2 := 500
	if len(df2.Rows) < limit2 {
		limit2 = len(df2.Rows)
	}
	for i := 0; i < limit2; i++ {
		if col2Idx < len(df2.Rows[i]) && df2.Rows[i][col2Idx] != "" {
			set2[strings.ToLower(df2.Rows[i][col2Idx])] = true
		}
	}

	if len(set1) == 0 || len(set2) == 0 {
		return 0
	}

	// Jaccard similarity
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

// calculateDistributionSimilarity compares statistical distributions
func (s *EnhancedSimilarityService) calculateDistributionSimilarity(df1, df2 *state.DataFrame, col1Idx, col2Idx int) float64 {
	vals1 := getFloatValues(df1, col1Idx)
	vals2 := getFloatValues(df2, col2Idx)

	if len(vals1) < 5 || len(vals2) < 5 {
		return 0
	}

	// Calculate stats for both columns
	mean1, std1 := meanAndStd(vals1)
	mean2, std2 := meanAndStd(vals2)

	// Coefficient of Variation similarity
	cv1 := 0.0
	cv2 := 0.0
	if mean1 != 0 {
		cv1 = std1 / math.Abs(mean1)
	}
	if mean2 != 0 {
		cv2 = std2 / math.Abs(mean2)
	}

	cvDiff := math.Abs(cv1 - cv2)
	cvSim := math.Max(0, 1-cvDiff)

	// Range similarity (normalized)
	min1, max1 := minMax(vals1)
	min2, max2 := minMax(vals2)
	range1 := max1 - min1
	range2 := max2 - min2

	rangeSim := 0.0
	if range1 > 0 && range2 > 0 {
		rangeRatio := math.Min(range1, range2) / math.Max(range1, range2)
		rangeSim = rangeRatio
	}

	// Combine metrics
	return (cvSim * 0.6) + (rangeSim * 0.4)
}

// getFloatValues extracts numeric values from a column
func getFloatValues(df *state.DataFrame, colIdx int) []float64 {
	values := []float64{}
	for _, row := range df.Rows {
		if colIdx >= len(row) {
			continue
		}
		if val, err := strconv.ParseFloat(row[colIdx], 64); err == nil {
			values = append(values, val)
		}
	}
	return values
}

// meanAndStd calculates mean and standard deviation
func meanAndStd(vals []float64) (float64, float64) {
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

// minMax returns min and max values
func minMax(vals []float64) (float64, float64) {
	if len(vals) == 0 {
		return 0, 0
	}
	min, max := vals[0], vals[0]
	for _, v := range vals {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

// applyContextBoost adjusts confidence based on context
func (s *EnhancedSimilarityService) applyContextBoost(confidence float64, col1, col2 string, ctx1, ctx2 *models.Context) float64 {
	boost := 1.0

	// Custom mapping check
	if target, ok := ctx1.CustomMappings[col1]; ok && target == col2 {
		return 95.0 // High confidence for explicit mappings
	}

	// Same business domain boost
	if ctx1.BusinessDomain != "" && ctx1.BusinessDomain == ctx2.BusinessDomain {
		boost *= 1.1
	}

	// Key entity boost
	col1Lower := strings.ToLower(col1)
	col2Lower := strings.ToLower(col2)
	for _, entity := range ctx1.KeyEntities {
		entityLower := strings.ToLower(entity)
		if strings.Contains(col1Lower, entityLower) && strings.Contains(col2Lower, entityLower) {
			boost *= 1.15
			break
		}
	}

	return math.Min(100, confidence*boost)
}

// determineType categorizes the match type
func (s *EnhancedSimilarityService) determineType(r SimilarityResult) string {
	if r.PatternMatch != "" {
		return "pattern"
	}
	if r.ValueOverlap > 0.5 {
		return "value_overlap"
	}
	if r.DistributionSimilarity > 0.5 {
		return "distribution"
	}
	if r.SynonymMatch {
		return "synonym"
	}
	if r.TokenSimilarity > 0.5 {
		return "token"
	}
	if r.NameSimilarity > 0.3 {
		return "name"
	}
	return "weak"
}

// buildReason creates an explanation string
func (s *EnhancedSimilarityService) buildReason(r SimilarityResult, synonym bool) string {
	parts := []string{}

	if r.TokenSimilarity > 0 {
		parts = append(parts, "Name:"+formatPercent(r.TokenSimilarity))
	}
	if synonym {
		parts = append(parts, "Synonym✓")
	}
	if r.DataSimilarity > 0 {
		parts = append(parts, "Data:"+formatPercent(r.DataSimilarity))
	}
	if r.PatternMatch != "" {
		parts = append(parts, "Pattern:"+r.PatternMatch)
	}
	if r.ValueOverlap > 0 {
		parts = append(parts, "Overlap:"+formatPercent(r.ValueOverlap))
	}

	return strings.Join(parts, ", ")
}

func formatPercent(v float64) string {
	return strconv.FormatFloat(v*100, 'f', 0, 64) + "%"
}
