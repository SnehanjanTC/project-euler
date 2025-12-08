package service

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const patternLearningFile = "./data/pattern_learning.json"

// PatternRule represents a learned pattern transformation
type PatternRule struct {
	Pattern1     string    `json:"pattern1"`      // Pattern from file1 (e.g., "*_id")
	Pattern2     string    `json:"pattern2"`      // Pattern from file2 (e.g., "*_identifier")
	Confidence   float64   `json:"confidence"`    // Learned confidence
	SuccessCount int       `json:"success_count"` // Times confirmed correct
	FailCount    int       `json:"fail_count"`    // Times marked incorrect
	LastUpdated  time.Time `json:"last_updated"`
}

// TokenMapping represents learned token equivalences
type TokenMapping struct {
	Token1      string  `json:"token1"`
	Token2      string  `json:"token2"`
	Score       float64 `json:"score"` // Learned similarity score
	Occurrences int     `json:"occurrences"`
}

// PatternLearner learns column naming patterns from feedback
type PatternLearner struct {
	patterns      []PatternRule
	tokenMappings map[string]TokenMapping // key: "token1|token2"
	mutex         sync.RWMutex
}

var (
	patternLearner     *PatternLearner
	patternLearnerOnce sync.Once
)

// GetPatternLearner returns the singleton pattern learner
func GetPatternLearner() *PatternLearner {
	patternLearnerOnce.Do(func() {
		patternLearner = &PatternLearner{
			patterns:      []PatternRule{},
			tokenMappings: make(map[string]TokenMapping),
		}
		patternLearner.load()
	})
	return patternLearner
}

// load loads patterns from file
func (p *PatternLearner) load() {
	dir := filepath.Dir(patternLearningFile)
	os.MkdirAll(dir, 0755)

	data, err := os.ReadFile(patternLearningFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[PatternLearner] Error loading patterns: %v", err)
		}
		return
	}

	var saved struct {
		Patterns      []PatternRule            `json:"patterns"`
		TokenMappings map[string]TokenMapping `json:"token_mappings"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		log.Printf("[PatternLearner] Error parsing patterns: %v", err)
		return
	}

	p.mutex.Lock()
	p.patterns = saved.Patterns
	if saved.TokenMappings != nil {
		p.tokenMappings = saved.TokenMappings
	}
	p.mutex.Unlock()

	log.Printf("[PatternLearner] Loaded %d patterns and %d token mappings",
		len(p.patterns), len(p.tokenMappings))
}

// save persists patterns to file
func (p *PatternLearner) save() error {
	p.mutex.RLock()
	data, err := json.MarshalIndent(map[string]interface{}{
		"patterns":       p.patterns,
		"token_mappings": p.tokenMappings,
	}, "", "  ")
	p.mutex.RUnlock()

	if err != nil {
		return err
	}

	dir := filepath.Dir(patternLearningFile)
	os.MkdirAll(dir, 0755)

	return os.WriteFile(patternLearningFile, data, 0644)
}

// LearnFromPositive learns from a confirmed correct match
func (p *PatternLearner) LearnFromPositive(col1, col2 string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Extract patterns
	pattern1 := extractPattern(col1)
	pattern2 := extractPattern(col2)

	// Update or create pattern rule
	found := false
	for i := range p.patterns {
		if p.patterns[i].Pattern1 == pattern1 && p.patterns[i].Pattern2 == pattern2 {
			p.patterns[i].SuccessCount++
			p.patterns[i].Confidence = calculatePatternConfidence(
				p.patterns[i].SuccessCount, p.patterns[i].FailCount)
			p.patterns[i].LastUpdated = time.Now()
			found = true
			break
		}
	}

	if !found && pattern1 != "" && pattern2 != "" {
		p.patterns = append(p.patterns, PatternRule{
			Pattern1:     pattern1,
			Pattern2:     pattern2,
			Confidence:   0.7, // Initial confidence
			SuccessCount: 1,
			FailCount:    0,
			LastUpdated:  time.Now(),
		})
	}

	// Learn token mappings
	tokens1 := tokenizeColumn(col1)
	tokens2 := tokenizeColumn(col2)
	for _, t1 := range tokens1 {
		for _, t2 := range tokens2 {
			if t1 != "" && t2 != "" {
				key := t1 + "|" + t2
				mapping, exists := p.tokenMappings[key]
				if exists {
					mapping.Occurrences++
					mapping.Score = 0.5 + (0.5 * float64(mapping.Occurrences) / float64(mapping.Occurrences+5))
				} else {
					mapping = TokenMapping{
						Token1:      t1,
						Token2:      t2,
						Score:       0.6,
						Occurrences: 1,
					}
				}
				p.tokenMappings[key] = mapping
			}
		}
	}

	go p.save()

	log.Printf("[PatternLearner] Learned positive: %s ↔ %s (pattern: %s ↔ %s)",
		col1, col2, pattern1, pattern2)
}

// LearnFromNegative learns from a confirmed incorrect match
func (p *PatternLearner) LearnFromNegative(col1, col2 string, nameSim, dataSim float64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	pattern1 := extractPattern(col1)
	pattern2 := extractPattern(col2)

	// Update pattern rule if exists
	for i := range p.patterns {
		if p.patterns[i].Pattern1 == pattern1 && p.patterns[i].Pattern2 == pattern2 {
			p.patterns[i].FailCount++
			p.patterns[i].Confidence = calculatePatternConfidence(
				p.patterns[i].SuccessCount, p.patterns[i].FailCount)
			p.patterns[i].LastUpdated = time.Now()
			break
		}
	}

	// Reduce token mapping scores
	tokens1 := tokenizeColumn(col1)
	tokens2 := tokenizeColumn(col2)
	for _, t1 := range tokens1 {
		for _, t2 := range tokens2 {
			key := t1 + "|" + t2
			if mapping, exists := p.tokenMappings[key]; exists {
				mapping.Score *= 0.8 // Reduce by 20%
				if mapping.Score < 0.1 {
					mapping.Score = 0.1
				}
				p.tokenMappings[key] = mapping
			}
		}
	}

	go p.save()

	log.Printf("[PatternLearner] Learned negative: %s ↔ %s (nameSim=%.2f, dataSim=%.2f)",
		col1, col2, nameSim, dataSim)
}

// GetPatternBoost returns a confidence boost based on learned patterns
func (p *PatternLearner) GetPatternBoost(col1, col2 string) float64 {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	pattern1 := extractPattern(col1)
	pattern2 := extractPattern(col2)

	// Check for matching pattern rule
	for _, rule := range p.patterns {
		if rule.Pattern1 == pattern1 && rule.Pattern2 == pattern2 {
			// Return boost based on confidence (can be negative for low confidence)
			if rule.Confidence > 0.7 {
				return (rule.Confidence - 0.5) * 0.4 // Up to +0.2 boost
			} else if rule.Confidence < 0.3 {
				return (rule.Confidence - 0.5) * 0.4 // Up to -0.2 penalty
			}
		}
	}

	// Check token mappings
	tokens1 := tokenizeColumn(col1)
	tokens2 := tokenizeColumn(col2)
	totalScore := 0.0
	count := 0
	for _, t1 := range tokens1 {
		for _, t2 := range tokens2 {
			key := t1 + "|" + t2
			if mapping, exists := p.tokenMappings[key]; exists {
				totalScore += mapping.Score - 0.5 // Centered around 0
				count++
			}
		}
	}

	if count > 0 {
		return (totalScore / float64(count)) * 0.2 // Up to +/- 0.1 boost
	}

	return 0.0
}

// GetPatterns returns all learned patterns
func (p *PatternLearner) GetPatterns() []PatternRule {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]PatternRule, len(p.patterns))
	copy(result, p.patterns)
	return result
}

// extractPattern extracts a generalized pattern from a column name
func extractPattern(col string) string {
	col = strings.ToLower(col)

	// Common pattern extractions
	patterns := []struct {
		suffix  string
		pattern string
	}{
		{"_id", "*_id"},
		{"_identifier", "*_identifier"},
		{"_code", "*_code"},
		{"_name", "*_name"},
		{"_date", "*_date"},
		{"_time", "*_time"},
		{"_at", "*_at"},
		{"_type", "*_type"},
		{"_status", "*_status"},
		{"_amount", "*_amount"},
		{"_price", "*_price"},
		{"_count", "*_count"},
		{"_num", "*_num"},
		{"_number", "*_number"},
	}

	for _, p := range patterns {
		if strings.HasSuffix(col, p.suffix) {
			return p.pattern
		}
	}

	// Check for prefixes
	prefixes := []struct {
		prefix  string
		pattern string
	}{
		{"is_", "is_*"},
		{"has_", "has_*"},
		{"date_", "date_*"},
		{"num_", "num_*"},
	}

	for _, p := range prefixes {
		if strings.HasPrefix(col, p.prefix) {
			return p.pattern
		}
	}

	return ""
}

// tokenizeColumn splits a column name into tokens
func tokenizeColumn(col string) []string {
	col = strings.ToLower(col)
	col = strings.ReplaceAll(col, "_", " ")
	col = strings.ReplaceAll(col, "-", " ")
	
	tokens := strings.Fields(col)
	result := []string{}
	for _, t := range tokens {
		if len(t) > 1 {
			result = append(result, t)
		}
	}
	return result
}

// calculatePatternConfidence calculates confidence from success/fail counts
func calculatePatternConfidence(success, fail int) float64 {
	total := success + fail
	if total == 0 {
		return 0.5
	}
	// Wilson score lower bound (simplified)
	// Add a small prior to avoid extreme values with few samples
	return (float64(success) + 2) / (float64(total) + 4)
}

