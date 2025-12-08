package service

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const feedbackFile = "./data/matching_feedback.json"

// FeedbackEntry represents a single feedback submission
type FeedbackEntry struct {
	File1Column    string    `json:"file1_column"`
	File2Column    string    `json:"file2_column"`
	IsCorrect      bool      `json:"is_correct"`
	CorrectMatch   string    `json:"correct_match,omitempty"`
	UserNote       string    `json:"user_note,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	NameSimilarity float64   `json:"name_similarity"`
	DataSimilarity float64   `json:"data_similarity"`
	PatternScore   float64   `json:"pattern_score"`
	Confidence     float64   `json:"confidence"`
}

// Correction represents a learned correction
type Correction struct {
	Suggested string `json:"suggested"`
	Correct   string `json:"correct"`
	Count     int    `json:"count"`
}

// FeedbackData holds all feedback data
type FeedbackData struct {
	Matches     []FeedbackEntry       `json:"matches"`
	Corrections map[string]Correction `json:"corrections"`
}

// FeedbackLearningSystem manages feedback-based learning
type FeedbackLearningSystem struct {
	data   *FeedbackData
	mutex  sync.RWMutex
	dirty  bool
}

var (
	feedbackSystem *FeedbackLearningSystem
	feedbackOnce   sync.Once
)

// GetFeedbackSystem returns the singleton feedback system
func GetFeedbackSystem() *FeedbackLearningSystem {
	feedbackOnce.Do(func() {
		feedbackSystem = &FeedbackLearningSystem{
			data: &FeedbackData{
				Matches:     []FeedbackEntry{},
				Corrections: make(map[string]Correction),
			},
		}
		feedbackSystem.load()
	})
	return feedbackSystem
}

// load loads feedback from file
func (f *FeedbackLearningSystem) load() {
	// Ensure directory exists
	dir := filepath.Dir(feedbackFile)
	os.MkdirAll(dir, 0755)

	data, err := os.ReadFile(feedbackFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[Feedback] Error loading feedback: %v", err)
		}
		return
	}

	var loaded FeedbackData
	if err := json.Unmarshal(data, &loaded); err != nil {
		log.Printf("[Feedback] Error parsing feedback: %v", err)
		return
	}

	f.mutex.Lock()
	f.data = &loaded
	if f.data.Corrections == nil {
		f.data.Corrections = make(map[string]Correction)
	}
	f.mutex.Unlock()

	log.Printf("[Feedback] Loaded %d feedback entries", len(f.data.Matches))
}

// save persists feedback to file
func (f *FeedbackLearningSystem) save() error {
	f.mutex.RLock()
	data, err := json.MarshalIndent(f.data, "", "  ")
	f.mutex.RUnlock()

	if err != nil {
		return err
	}

	dir := filepath.Dir(feedbackFile)
	os.MkdirAll(dir, 0755)

	return os.WriteFile(feedbackFile, data, 0644)
}

// AddFeedback records user feedback on a column match
func (f *FeedbackLearningSystem) AddFeedback(entry FeedbackEntry) (*FeedbackEntry, error) {
	entry.Timestamp = time.Now()

	f.mutex.Lock()
	f.data.Matches = append(f.data.Matches, entry)

	// Store corrections for learning
	if !entry.IsCorrect && entry.CorrectMatch != "" {
		key := entry.File1Column + "|" + entry.File2Column
		existing, ok := f.data.Corrections[key]
		count := 1
		if ok {
			count = existing.Count + 1
		}
		f.data.Corrections[key] = Correction{
			Suggested: entry.File2Column,
			Correct:   entry.CorrectMatch,
			Count:     count,
		}
	}
	
	// Get recent feedback for batch learning
	recentFeedback := f.data.Matches
	if len(recentFeedback) > 10 {
		recentFeedback = recentFeedback[len(recentFeedback)-10:]
	}
	f.mutex.Unlock()

	// Save to file
	if err := f.save(); err != nil {
		log.Printf("[Feedback] Error saving feedback: %v", err)
	}

	// Trigger ML learning systems asynchronously
	go f.triggerMLLearning(entry, recentFeedback)

	log.Printf("[Feedback] Recorded: %s ↔ %s (correct: %v)", 
		entry.File1Column, entry.File2Column, entry.IsCorrect)

	return &entry, nil
}

// triggerMLLearning triggers all ML learning systems
func (f *FeedbackLearningSystem) triggerMLLearning(feedback FeedbackEntry, recentBatch []FeedbackEntry) {
	// 1. Update confidence calibration
	calibrator := GetConfidenceCalibrator()
	calibrator.Update(feedback.Confidence, feedback.IsCorrect)

	// 2. Update pattern learning
	patternLearner := GetPatternLearner()
	if feedback.IsCorrect {
		patternLearner.LearnFromPositive(feedback.File1Column, feedback.File2Column)
	} else {
		patternLearner.LearnFromNegative(
			feedback.File1Column,
			feedback.File2Column,
			feedback.NameSimilarity,
			feedback.DataSimilarity,
		)
	}

	// 3. Update adaptive weights (batch update every 10 feedbacks)
	if len(recentBatch) >= 10 {
		adaptiveLearner := GetAdaptiveLearner()
		adaptiveLearner.UpdateWeights(recentBatch)
	}

	log.Printf("[ML Learning] Triggered for: %s ↔ %s", feedback.File1Column, feedback.File2Column)
}


// GetLearnedBoost returns a confidence adjustment based on historical feedback
// Returns a value between -0.3 and +0.3
func (f *FeedbackLearningSystem) GetLearnedBoost(file1Col, file2Col string) float64 {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// Check if this exact match has feedback
	for _, match := range f.data.Matches {
		if match.File1Column == file1Col && match.File2Column == file2Col {
			if match.IsCorrect {
				return 0.2 // Boost by 20%
			}
			return -0.3 // Penalize by 30%
		}
	}

	// Check corrections
	key := file1Col + "|" + file2Col
	if _, ok := f.data.Corrections[key]; ok {
		return -0.25 // Penalize known incorrect matches
	}

	// Check if file2_col was previously suggested incorrectly
	for _, correction := range f.data.Corrections {
		if correction.Suggested == file2Col {
			return -0.15
		}
	}

	return 0.0
}

// GetSuggestedMatch returns the learned correct match for a column
func (f *FeedbackLearningSystem) GetSuggestedMatch(file1Col string) string {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// Check for confirmed correct matches
	for _, match := range f.data.Matches {
		if match.File1Column == file1Col && match.IsCorrect {
			return match.File2Column
		}
	}

	// Check corrections
	for key, correction := range f.data.Corrections {
		if len(key) > len(file1Col)+1 && key[:len(file1Col)+1] == file1Col+"|" {
			return correction.Correct
		}
	}

	return ""
}

// GetStats returns feedback statistics
func (f *FeedbackLearningSystem) GetStats() map[string]interface{} {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	totalFeedback := len(f.data.Matches)
	correctMatches := 0
	for _, m := range f.data.Matches {
		if m.IsCorrect {
			correctMatches++
		}
	}
	incorrectMatches := totalFeedback - correctMatches

	accuracy := 0.0
	if totalFeedback > 0 {
		accuracy = float64(correctMatches) / float64(totalFeedback) * 100
	}

	return map[string]interface{}{
		"total_feedback":    totalFeedback,
		"correct_matches":   correctMatches,
		"incorrect_matches": incorrectMatches,
		"accuracy":          accuracy,
		"total_corrections": len(f.data.Corrections),
	}
}

// GetRecentFeedback returns the most recent N feedback entries
func (f *FeedbackLearningSystem) GetRecentFeedback(n int) []FeedbackEntry {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	if len(f.data.Matches) <= n {
		return f.data.Matches
	}
	return f.data.Matches[len(f.data.Matches)-n:]
}

// HasPositiveFeedback checks if a column pair has positive feedback
func (f *FeedbackLearningSystem) HasPositiveFeedback(file1Col, file2Col string) bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	for _, match := range f.data.Matches {
		if match.File1Column == file1Col && match.File2Column == file2Col && match.IsCorrect {
			return true
		}
	}
	return false
}

// ClearFeedback clears all feedback (for testing)
func (f *FeedbackLearningSystem) ClearFeedback() {
	f.mutex.Lock()
	f.data = &FeedbackData{
		Matches:     []FeedbackEntry{},
		Corrections: make(map[string]Correction),
	}
	f.mutex.Unlock()
	f.save()
}
