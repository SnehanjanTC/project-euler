package service

import (
	"encoding/json"
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const adaptiveWeightsFile = "./data/adaptive_weights.json"

// AdaptiveWeights represents the learned weights for different similarity factors
type AdaptiveWeights struct {
	Name    float64 `json:"name"`
	Data    float64 `json:"data"`
	Pattern float64 `json:"pattern"`
	LLM     float64 `json:"llm"`
}

// TrainingHistoryEntry records a weight update
type TrainingHistoryEntry struct {
	Timestamp time.Time       `json:"timestamp"`
	Loss      float64         `json:"loss"`
	Weights   AdaptiveWeights `json:"weights"`
	BatchSize int             `json:"batch_size"`
}

// AdaptiveWeightLearner uses gradient descent to learn optimal weights
type AdaptiveWeightLearner struct {
	weights         AdaptiveWeights
	learningRate    float64
	trainingHistory []TrainingHistoryEntry
	mutex           sync.RWMutex
}

var (
	adaptiveLearner     *AdaptiveWeightLearner
	adaptiveLearnerOnce sync.Once
)

// GetAdaptiveLearner returns the singleton adaptive learner
func GetAdaptiveLearner() *AdaptiveWeightLearner {
	adaptiveLearnerOnce.Do(func() {
		adaptiveLearner = &AdaptiveWeightLearner{
			weights: AdaptiveWeights{
				Name:    0.35, // Default weights
				Data:    0.30,
				Pattern: 0.20,
				LLM:     0.15,
			},
			learningRate:    0.01,
			trainingHistory: []TrainingHistoryEntry{},
		}
		adaptiveLearner.load()
	})
	return adaptiveLearner
}

// load loads weights from file
func (a *AdaptiveWeightLearner) load() {
	dir := filepath.Dir(adaptiveWeightsFile)
	os.MkdirAll(dir, 0755)

	data, err := os.ReadFile(adaptiveWeightsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[AdaptiveLearner] Error loading weights: %v", err)
		}
		return
	}

	var saved struct {
		Weights  AdaptiveWeights        `json:"weights"`
		History  []TrainingHistoryEntry `json:"history"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		log.Printf("[AdaptiveLearner] Error parsing weights: %v", err)
		return
	}

	a.mutex.Lock()
	a.weights = saved.Weights
	a.trainingHistory = saved.History
	a.mutex.Unlock()

	log.Printf("[AdaptiveLearner] Loaded weights: Name=%.2f, Data=%.2f, Pattern=%.2f, LLM=%.2f",
		a.weights.Name, a.weights.Data, a.weights.Pattern, a.weights.LLM)
}

// save persists weights to file
func (a *AdaptiveWeightLearner) save() error {
	a.mutex.RLock()
	data, err := json.MarshalIndent(map[string]interface{}{
		"weights": a.weights,
		"history": a.trainingHistory,
	}, "", "  ")
	a.mutex.RUnlock()

	if err != nil {
		return err
	}

	dir := filepath.Dir(adaptiveWeightsFile)
	os.MkdirAll(dir, 0755)

	return os.WriteFile(adaptiveWeightsFile, data, 0644)
}

// GetWeights returns the current weights
func (a *AdaptiveWeightLearner) GetWeights() AdaptiveWeights {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.weights
}

// CalculateScore calculates weighted score using learned weights
func (a *AdaptiveWeightLearner) CalculateScore(nameSim, dataSim, patternScore, llmScore float64) float64 {
	a.mutex.RLock()
	w := a.weights
	a.mutex.RUnlock()

	return (nameSim * w.Name) + (dataSim * w.Data) + (patternScore * w.Pattern) + (llmScore * w.LLM)
}

// UpdateWeights performs gradient descent on a batch of feedback
func (a *AdaptiveWeightLearner) UpdateWeights(feedbackBatch []FeedbackEntry) {
	if len(feedbackBatch) == 0 {
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Accumulate gradients
	gradients := AdaptiveWeights{}
	totalLoss := 0.0
	n := float64(len(feedbackBatch))

	for _, fb := range feedbackBatch {
		// Current prediction using weights
		predicted := (fb.NameSimilarity * a.weights.Name) +
			(fb.DataSimilarity * a.weights.Data) +
			(fb.PatternScore * a.weights.Pattern)

		// Target: 1.0 if correct, 0.0 if incorrect
		target := 0.0
		if fb.IsCorrect {
			target = 1.0
		}

		// Error
		error := predicted - target
		totalLoss += error * error

		// Compute gradients (partial derivatives)
		gradients.Name += error * fb.NameSimilarity
		gradients.Data += error * fb.DataSimilarity
		gradients.Pattern += error * fb.PatternScore
	}

	// Average gradients
	gradients.Name /= n
	gradients.Data /= n
	gradients.Pattern /= n

	// Update weights using gradient descent
	a.weights.Name -= a.learningRate * gradients.Name
	a.weights.Data -= a.learningRate * gradients.Data
	a.weights.Pattern -= a.learningRate * gradients.Pattern

	// Ensure weights stay positive
	a.weights.Name = math.Max(0.05, a.weights.Name)
	a.weights.Data = math.Max(0.05, a.weights.Data)
	a.weights.Pattern = math.Max(0.05, a.weights.Pattern)
	a.weights.LLM = math.Max(0.05, a.weights.LLM)

	// Normalize weights to sum to 1.0
	total := a.weights.Name + a.weights.Data + a.weights.Pattern + a.weights.LLM
	if total > 0 {
		a.weights.Name /= total
		a.weights.Data /= total
		a.weights.Pattern /= total
		a.weights.LLM /= total
	}

	// Record training history
	avgLoss := totalLoss / n
	a.trainingHistory = append(a.trainingHistory, TrainingHistoryEntry{
		Timestamp: time.Now(),
		Loss:      avgLoss,
		Weights:   a.weights,
		BatchSize: len(feedbackBatch),
	})

	// Keep last 100 entries
	if len(a.trainingHistory) > 100 {
		a.trainingHistory = a.trainingHistory[len(a.trainingHistory)-100:]
	}

	// Save updated weights
	go a.save()

	log.Printf("[AdaptiveLearner] Weights updated: Name=%.3f, Data=%.3f, Pattern=%.3f, LLM=%.3f (Loss=%.4f)",
		a.weights.Name, a.weights.Data, a.weights.Pattern, a.weights.LLM, avgLoss)
}

// GetTrainingHistory returns recent training history
func (a *AdaptiveWeightLearner) GetTrainingHistory() []TrainingHistoryEntry {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.trainingHistory
}
