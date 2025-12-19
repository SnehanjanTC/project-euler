package service

import (
	"backend-go/internal/state"
	"math"
	"strings"
)

// NormalizedValueMatcher compares columns using normalized values
type NormalizedValueMatcher struct {
	normalizer *FormatNormalizer
	profiler   *DataQualityProfiler
}

// NewNormalizedValueMatcher creates a new matcher
func NewNormalizedValueMatcher() *NormalizedValueMatcher {
	return &NormalizedValueMatcher{
		normalizer: NewFormatNormalizer(),
		profiler:   NewDataQualityProfiler(),
	}
}

// CalculateNormalizedMatch compares columns using normalized values
func (nvm *NormalizedValueMatcher) CalculateNormalizedMatch(
	df1, df2 *state.DataFrame,
	col1Idx, col2Idx int,
) float64 {
	// Sample up to 200 rows from each
	sampleSize := 200
	if len(df1.Rows) < sampleSize {
		sampleSize = len(df1.Rows)
	}
	if len(df2.Rows) < sampleSize {
		sampleSize = len(df2.Rows)
	}

	// Normalize values from both columns
	normalized1 := make(map[string]bool)
	normalized2 := make(map[string]bool)

	for i := 0; i < sampleSize; i++ {
		if col1Idx < len(df1.Rows[i]) {
			val := df1.Rows[i][col1Idx]
			if val != "" {
				normalized := nvm.normalizer.NormalizeValue(val)
				if normalized != "" {
					normalized1[normalized] = true
				}
			}
		}
	}

	for i := 0; i < sampleSize; i++ {
		if col2Idx < len(df2.Rows[i]) {
			val := df2.Rows[i][col2Idx]
			if val != "" {
				normalized := nvm.normalizer.NormalizeValue(val)
				if normalized != "" {
					normalized2[normalized] = true
				}
			}
		}
	}

	if len(normalized1) == 0 || len(normalized2) == 0 {
		return 0
	}

	// Calculate Jaccard similarity of normalized values
	intersection := 0
	for val := range normalized1 {
		if normalized2[val] {
			intersection++
		}
	}

	union := len(normalized1) + len(normalized2) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// DetectFormatTransformation checks if columns have same data in different formats
func (nvm *NormalizedValueMatcher) DetectFormatTransformation(
	df1, df2 *state.DataFrame,
	col1Idx, col2Idx int,
) (bool, string) {
	// Sample a few values
	sampleSize := 10
	if len(df1.Rows) < sampleSize {
		sampleSize = len(df1.Rows)
	}

	format1 := ""
	format2 := ""

	// Detect format of first column
	for i := 0; i < sampleSize; i++ {
		if col1Idx < len(df1.Rows[i]) && df1.Rows[i][col1Idx] != "" {
			format1 = nvm.normalizer.DetectFormat(df1.Rows[i][col1Idx])
			if format1 != "text" {
				break
			}
		}
	}

	// Detect format of second column
	for i := 0; i < sampleSize; i++ {
		if col2Idx < len(df2.Rows[i]) && df2.Rows[i][col2Idx] != "" {
			format2 = nvm.normalizer.DetectFormat(df2.Rows[i][col2Idx])
			if format2 != "text" {
				break
			}
		}
	}

	// If both have the same non-text format, check if values match when normalized
	if format1 != "" && format1 == format2 && format1 != "text" {
		matchScore := nvm.CalculateNormalizedMatch(df1, df2, col1Idx, col2Idx)
		if matchScore > 0.5 {
			return true, format1
		}
	}

	return false, ""
}

// CalculateCardinalityMatch compares cardinality patterns
func (nvm *NormalizedValueMatcher) CalculateCardinalityMatch(
	profile1, profile2 DataQualityProfile,
) float64 {
	// If both are primary keys, strong match
	if profile1.IsPrimaryKey && profile2.IsPrimaryKey {
		return 0.9
	}

	// If one is primary key and other is not, weak match
	if profile1.IsPrimaryKey != profile2.IsPrimaryKey {
		return 0.2
	}

	// Compare uniqueness ratios
	uniquenessDiff := math.Abs(profile1.UniquenessRatio - profile2.UniquenessRatio)

	// Similar uniqueness suggests similar cardinality
	if uniquenessDiff < 0.1 {
		return 0.8
	} else if uniquenessDiff < 0.3 {
		return 0.6
	} else {
		return 0.3
	}
}

// ExplainMatch generates a detailed explanation of why columns match
func (nvm *NormalizedValueMatcher) ExplainMatch(
	col1, col2 string,
	nameSim, dataSim, normalizedSim, cardinalitySim float64,
	profile1, profile2 DataQualityProfile,
	formatMatch bool,
	formatType string,
) string {
	explanations := []string{}

	// Name similarity
	if nameSim > 0.7 {
		explanations = append(explanations, "column names are very similar")
	} else if nameSim > 0.4 {
		explanations = append(explanations, "column names are somewhat similar")
	}

	// Format transformation
	if formatMatch {
		explanations = append(explanations, "same "+formatType+" data in different formats")
	}

	// Normalized value match
	if normalizedSim > 0.7 {
		explanations = append(explanations, "high value overlap when normalized")
	} else if normalizedSim > 0.4 {
		explanations = append(explanations, "moderate value overlap")
	}

	// Cardinality
	if profile1.IsPrimaryKey && profile2.IsPrimaryKey {
		explanations = append(explanations, "both are unique identifiers")
	} else if cardinalitySim > 0.7 {
		explanations = append(explanations, "similar cardinality patterns")
	}

	// Data quality
	if profile1.QualityScore > 0.7 && profile2.QualityScore > 0.7 {
		explanations = append(explanations, "both have high data quality")
	}

	if len(explanations) == 0 {
		return "weak match based on basic similarity"
	}

	return strings.Join(explanations, ", ")
}
