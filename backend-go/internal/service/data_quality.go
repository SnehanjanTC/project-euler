package service

import (
	"backend-go/internal/state"
	"math"
)

// DataQualityProfile holds quality metrics for a column
type DataQualityProfile struct {
	ColumnName      string  `json:"column_name"`
	TotalRows       int     `json:"total_rows"`
	NonNullRows     int     `json:"non_null_rows"`
	NullRate        float64 `json:"null_rate"`
	DistinctCount   int     `json:"distinct_count"`
	UniquenessRatio float64 `json:"uniqueness_ratio"`
	Entropy         float64 `json:"entropy"`
	IsPrimaryKey    bool    `json:"is_primary_key"`
	QualityScore    float64 `json:"quality_score"` // 0-1
}

// DataQualityProfiler analyzes data quality metrics
type DataQualityProfiler struct{}

// NewDataQualityProfiler creates a new profiler
func NewDataQualityProfiler() *DataQualityProfiler {
	return &DataQualityProfiler{}
}

// ProfileColumn analyzes quality metrics for a single column
func (dqp *DataQualityProfiler) ProfileColumn(df *state.DataFrame, colIdx int) DataQualityProfile {
	profile := DataQualityProfile{
		ColumnName: df.Headers[colIdx],
		TotalRows:  len(df.Rows),
	}

	// Track unique values and null count
	uniqueValues := make(map[string]int)
	nonNullCount := 0

	for _, row := range df.Rows {
		if colIdx >= len(row) {
			continue
		}

		value := row[colIdx]
		if value == "" || value == "null" || value == "NULL" || value == "None" {
			continue
		}

		nonNullCount++
		uniqueValues[value]++
	}

	profile.NonNullRows = nonNullCount
	profile.DistinctCount = len(uniqueValues)

	// Calculate null rate
	if profile.TotalRows > 0 {
		profile.NullRate = float64(profile.TotalRows-nonNullCount) / float64(profile.TotalRows)
	}

	// Calculate uniqueness ratio
	if nonNullCount > 0 {
		profile.UniquenessRatio = float64(profile.DistinctCount) / float64(nonNullCount)
	}

	// Calculate entropy (measure of randomness/diversity)
	profile.Entropy = dqp.calculateEntropy(uniqueValues, nonNullCount)

	// Detect if this is likely a primary key
	// High uniqueness (>95%) and low null rate (<5%)
	profile.IsPrimaryKey = profile.UniquenessRatio > 0.95 && profile.NullRate < 0.05

	// Calculate overall quality score
	profile.QualityScore = dqp.calculateQualityScore(profile)

	return profile
}

// ProfileAllColumns profiles all columns in a dataframe
func (dqp *DataQualityProfiler) ProfileAllColumns(df *state.DataFrame) []DataQualityProfile {
	profiles := make([]DataQualityProfile, len(df.Headers))
	for i := range df.Headers {
		profiles[i] = dqp.ProfileColumn(df, i)
	}
	return profiles
}

// calculateEntropy computes Shannon entropy
func (dqp *DataQualityProfiler) calculateEntropy(valueCounts map[string]int, total int) float64 {
	if total == 0 {
		return 0
	}

	entropy := 0.0
	for _, count := range valueCounts {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

// calculateQualityScore computes overall quality (0-1)
func (dqp *DataQualityProfiler) calculateQualityScore(profile DataQualityProfile) float64 {
	score := 1.0

	// Penalize high null rates
	score *= (1.0 - profile.NullRate)

	// Reward moderate entropy (not too uniform, not too random)
	// Ideal entropy is around 3-5 bits
	idealEntropy := 4.0
	entropyPenalty := math.Abs(profile.Entropy-idealEntropy) / 10.0
	score *= math.Max(0.5, 1.0-entropyPenalty)

	// Ensure score is between 0 and 1
	return math.Max(0, math.Min(1, score))
}

// CompareQuality compares two column profiles for matching
func (dqp *DataQualityProfiler) CompareQuality(profile1, profile2 DataQualityProfile) float64 {
	// Both should have good quality
	if profile1.QualityScore < 0.3 || profile2.QualityScore < 0.3 {
		return 0.5 // Penalize low quality
	}

	// Similar uniqueness ratios suggest similar column types
	uniquenessDiff := math.Abs(profile1.UniquenessRatio - profile2.UniquenessRatio)
	uniquenessSim := 1.0 - uniquenessDiff

	// Similar entropy suggests similar data distributions
	entropyDiff := math.Abs(profile1.Entropy - profile2.Entropy)
	entropySim := math.Max(0, 1.0-(entropyDiff/10.0))

	// Both primary keys is a strong signal
	bothPrimaryKeys := 0.0
	if profile1.IsPrimaryKey && profile2.IsPrimaryKey {
		bothPrimaryKeys = 0.3
	}

	// Weighted combination
	similarity := (uniquenessSim * 0.4) + (entropySim * 0.3) + bothPrimaryKeys + 0.3

	return math.Max(0, math.Min(1, similarity))
}
