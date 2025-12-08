package analysis

import (
	"backend-go/internal/models"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CSVService struct{}

func NewCSVService() *CSVService {
	return &CSVService{}
}

// AnalyzeData performs analysis on generic data (from CSV or DB)
func (s *CSVService) AnalyzeData(data []map[string]interface{}, columns []string) (models.DataAnalysisResult, error) {
	result := models.DataAnalysisResult{
		ColumnNames:      columns,
		ColumnTypes:      make(map[string]string),
		PotentialIDs:     []string{},
		PotentialDates:   []string{},
		PotentialAmounts: []string{},
		NumRows:          len(data),
		NumColumns:       len(columns),
	}

	// Infer types
	for _, colName := range columns {
		colType := "string" // default

		// Check first non-nil value to guess type
		// For robustness we should check a sample, but simplistic for now
		foundType := false
		for _, row := range data {
			val := row[colName]
			if val == nil {
				continue
			}

			// If it's a string, we try to infer underlying type
			if strVal, ok := val.(string); ok {
				if strVal == "" {
					continue
				}
				// Use the string inference logic
				// We need a helper that takes a single string, or we reuse inferColumnType logic?
				// Let's create a specialized helper for mixed inputs
				colType = inferTypeFromValue(val)
			} else {
				// It's already typed (from DB)
				switch val.(type) {
				case int, int32, int64, float32, float64:
					colType = "float" // Treat all numbers as float for simplicity in analysis or separate?
					// Project Euler logic distinguished int/float.
					// Let's refine.
					if reflect.TypeOf(val).Kind() == reflect.Int || reflect.TypeOf(val).Kind() == reflect.Int64 {
						colType = "int"
					} else {
						colType = "float"
					}
				case time.Time:
					colType = "date"
				default:
					colType = "string"
				}
			}
			foundType = true
			break
		}

		if !foundType {
			colType = "string" // All nulls or empty
		}

		result.ColumnTypes[colName] = colType
		colLower := strings.ToLower(colName)

		if colType == "int" || colType == "float" {
			result.HasNumeric = true
			if containsAny(colLower, []string{"id", "number", "code", "key"}) {
				result.PotentialIDs = append(result.PotentialIDs, colName)
			}
			if containsAny(colLower, []string{"amount", "price", "cost", "revenue", "salary"}) {
				result.PotentialAmounts = append(result.PotentialAmounts, colName)
			}
		} else if colType == "date" {
			result.HasDates = true
			result.PotentialDates = append(result.PotentialDates, colName)
		} else {
			result.HasText = true
			// Check if name implies date even if data didn't parse easily
			if containsAny(colLower, []string{"date", "time", "timestamp"}) {
				result.PotentialDates = append(result.PotentialDates, colName)
				result.HasDates = true
			}
		}
	}

	return result, nil
}

// AnalyzeFile reads a CSV file and returns analysis results
func (s *CSVService) AnalyzeFile(filePath string) (models.DataAnalysisResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return models.DataAnalysisResult{}, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header
	headers, err := reader.Read()
	if err != nil {
		return models.DataAnalysisResult{}, err
	}

	// Read all rows and convert to map
	var data []map[string]interface{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return models.DataAnalysisResult{}, err
		}

		rowMap := make(map[string]interface{})
		for i, val := range record {
			if i < len(headers) {
				rowMap[headers[i]] = val
			}
		}
		data = append(data, rowMap)
	}

	return s.AnalyzeData(data, headers)
}

func inferTypeFromValue(v interface{}) string {
	strVal, ok := v.(string)
	if !ok {
		return "string"
	}

	if _, err := strconv.Atoi(strVal); err == nil {
		return "int"
	}
	if _, err := strconv.ParseFloat(strVal, 64); err == nil {
		return "float"
	}
	if isDateString(strVal) {
		return "date"
	}
	return "string"
}

func inferColumnType(rows [][]string, colIndex int) string {
	// Check a sample of rows
	sampleSize := 20
	if len(rows) < sampleSize {
		sampleSize = len(rows)
	}

	isInt := true
	isFloat := true
	isDate := true

	for i := 0; i < sampleSize; i++ {
		val := rows[i][colIndex]
		if val == "" {
			continue // Skip empties
		}

		if _, err := strconv.Atoi(val); err != nil {
			isInt = false
		}
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			isFloat = false
		}
		if !isDateString(val) {
			isDate = false
		}
	}

	if isInt {
		return "int"
	}
	if isFloat {
		return "float"
	}
	if isDate {
		return "date"
	}
	return "string"
}

func isDateString(val string) bool {
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
		"2006/01/02",
	}
	for _, f := range formats {
		if _, err := time.Parse(f, val); err == nil {
			return true
		}
	}
	return false
}

func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// CalculateStats computes basic stats for a numeric column
func CalculateStats(rows [][]string, colIndex int) (min, max, mean, median float64, err error) {
	values := []float64{}
	for _, row := range rows {
		if colIndex >= len(row) {
			continue
		}
		valStr := row[colIndex]
		if val, err := strconv.ParseFloat(valStr, 64); err == nil {
			values = append(values, val)
		}
	}

	if len(values) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("no numeric values")
	}

	sort.Float64s(values)
	min = values[0]
	max = values[len(values)-1]

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	if len(values)%2 == 0 {
		median = (values[len(values)/2-1] + values[len(values)/2]) / 2
	} else {
		median = values[len(values)/2]
	}

	return
}
