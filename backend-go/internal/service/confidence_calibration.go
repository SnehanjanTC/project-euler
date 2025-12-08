package service

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const confidenceCalibrationFile = "./data/confidence_calibration.json"

// CalibrationBucket represents a confidence range bucket
type CalibrationBucket struct {
	RangeMin      float64 `json:"range_min"`
	RangeMax      float64 `json:"range_max"`
	TotalCount    int     `json:"total_count"`
	CorrectCount  int     `json:"correct_count"`
	ActualAccuracy float64 `json:"actual_accuracy"`
	CalibrationFactor float64 `json:"calibration_factor"`
}

// CalibrationHistory records calibration updates
type CalibrationHistory struct {
	Timestamp       time.Time           `json:"timestamp"`
	PredictedConf   float64             `json:"predicted_confidence"`
	ActualCorrect   bool                `json:"actual_correct"`
	CalibratedConf  float64             `json:"calibrated_confidence"`
}

// ConfidenceCalibrator adjusts confidence scores based on historical accuracy
type ConfidenceCalibrator struct {
	buckets  []CalibrationBucket
	history  []CalibrationHistory
	mutex    sync.RWMutex
}

var (
	confidenceCalibrator     *ConfidenceCalibrator
	confidenceCalibratorOnce sync.Once
)

// GetConfidenceCalibrator returns the singleton calibrator
func GetConfidenceCalibrator() *ConfidenceCalibrator {
	confidenceCalibratorOnce.Do(func() {
		confidenceCalibrator = &ConfidenceCalibrator{
			buckets: initializeBuckets(),
			history: []CalibrationHistory{},
		}
		confidenceCalibrator.load()
	})
	return confidenceCalibrator
}

// initializeBuckets creates 10 buckets for confidence ranges 0-10, 10-20, ..., 90-100
func initializeBuckets() []CalibrationBucket {
	buckets := make([]CalibrationBucket, 10)
	for i := 0; i < 10; i++ {
		buckets[i] = CalibrationBucket{
			RangeMin:          float64(i * 10),
			RangeMax:          float64((i + 1) * 10),
			TotalCount:        0,
			CorrectCount:      0,
			ActualAccuracy:    float64(i*10 + 5) / 100, // Initial estimate based on range midpoint
			CalibrationFactor: 1.0,
		}
	}
	return buckets
}

// load loads calibration data from file
func (c *ConfidenceCalibrator) load() {
	dir := filepath.Dir(confidenceCalibrationFile)
	os.MkdirAll(dir, 0755)

	data, err := os.ReadFile(confidenceCalibrationFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[Calibrator] Error loading calibration: %v", err)
		}
		return
	}

	var saved struct {
		Buckets []CalibrationBucket  `json:"buckets"`
		History []CalibrationHistory `json:"history"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		log.Printf("[Calibrator] Error parsing calibration: %v", err)
		return
	}

	c.mutex.Lock()
	if len(saved.Buckets) == 10 {
		c.buckets = saved.Buckets
	}
	c.history = saved.History
	c.mutex.Unlock()

	log.Printf("[Calibrator] Loaded calibration data")
}

// save persists calibration data to file
func (c *ConfidenceCalibrator) save() error {
	c.mutex.RLock()
	data, err := json.MarshalIndent(map[string]interface{}{
		"buckets": c.buckets,
		"history": c.history,
	}, "", "  ")
	c.mutex.RUnlock()

	if err != nil {
		return err
	}

	dir := filepath.Dir(confidenceCalibrationFile)
	os.MkdirAll(dir, 0755)

	return os.WriteFile(confidenceCalibrationFile, data, 0644)
}

// Update records a new prediction outcome and updates calibration
func (c *ConfidenceCalibrator) Update(predictedConfidence float64, actualCorrect bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Find the bucket
	bucketIdx := int(predictedConfidence / 10)
	if bucketIdx >= 10 {
		bucketIdx = 9
	}
	if bucketIdx < 0 {
		bucketIdx = 0
	}

	// Update bucket
	c.buckets[bucketIdx].TotalCount++
	if actualCorrect {
		c.buckets[bucketIdx].CorrectCount++
	}

	// Recalculate actual accuracy
	if c.buckets[bucketIdx].TotalCount > 0 {
		c.buckets[bucketIdx].ActualAccuracy = float64(c.buckets[bucketIdx].CorrectCount) / float64(c.buckets[bucketIdx].TotalCount)
	}

	// Calculate calibration factor
	// If predicted is 80% but actual is 60%, factor = 0.75 (reduce confidence)
	expectedAccuracy := (c.buckets[bucketIdx].RangeMin + c.buckets[bucketIdx].RangeMax) / 200.0
	if expectedAccuracy > 0 {
		c.buckets[bucketIdx].CalibrationFactor = c.buckets[bucketIdx].ActualAccuracy / expectedAccuracy
	}

	// Record history
	c.history = append(c.history, CalibrationHistory{
		Timestamp:      time.Now(),
		PredictedConf:  predictedConfidence,
		ActualCorrect:  actualCorrect,
		CalibratedConf: c.calibrateInternal(predictedConfidence),
	})

	// Keep last 500 entries
	if len(c.history) > 500 {
		c.history = c.history[len(c.history)-500:]
	}

	// Save async
	go c.save()

	log.Printf("[Calibrator] Updated bucket %d: count=%d, accuracy=%.2f, factor=%.2f",
		bucketIdx, c.buckets[bucketIdx].TotalCount, c.buckets[bucketIdx].ActualAccuracy, c.buckets[bucketIdx].CalibrationFactor)
}

// Calibrate returns a calibrated confidence score
func (c *ConfidenceCalibrator) Calibrate(predictedConfidence float64) float64 {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.calibrateInternal(predictedConfidence)
}

// calibrateInternal (must hold lock)
func (c *ConfidenceCalibrator) calibrateInternal(predictedConfidence float64) float64 {
	bucketIdx := int(predictedConfidence / 10)
	if bucketIdx >= 10 {
		bucketIdx = 9
	}
	if bucketIdx < 0 {
		bucketIdx = 0
	}

	bucket := c.buckets[bucketIdx]
	
	// Only apply calibration if we have enough data
	if bucket.TotalCount < 5 {
		return predictedConfidence
	}

	// Apply calibration factor
	calibrated := predictedConfidence * bucket.CalibrationFactor

	// Clamp to 0-100
	if calibrated < 0 {
		calibrated = 0
	}
	if calibrated > 100 {
		calibrated = 100
	}

	return calibrated
}

// GetBuckets returns current bucket statistics
func (c *ConfidenceCalibrator) GetBuckets() []CalibrationBucket {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	result := make([]CalibrationBucket, len(c.buckets))
	copy(result, c.buckets)
	return result
}

// GetCalibrationStats returns summary statistics
func (c *ConfidenceCalibrator) GetCalibrationStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	totalSamples := 0
	totalCorrect := 0
	for _, b := range c.buckets {
		totalSamples += b.TotalCount
		totalCorrect += b.CorrectCount
	}

	overallAccuracy := 0.0
	if totalSamples > 0 {
		overallAccuracy = float64(totalCorrect) / float64(totalSamples) * 100
	}

	return map[string]interface{}{
		"total_samples":    totalSamples,
		"total_correct":    totalCorrect,
		"overall_accuracy": overallAccuracy,
		"buckets":          c.buckets,
	}
}
