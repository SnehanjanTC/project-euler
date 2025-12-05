package analysis

import (
	"backend-go/internal/models"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CSVService struct{}

func NewCSVService() *CSVService {
	return &CSVService{}
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

	// Initialize result
	result := models.DataAnalysisResult{
		ColumnNames:      headers,
		ColumnTypes:      make(map[string]string),
		PotentialIDs:     []string{},
		PotentialDates:   []string{},
		PotentialAmounts: []string{},
	}

	// Read rows to infer types (sample first 100 rows or all)
	// For simplicity, we'll read all but be mindful of memory.
	// In a real heavy app, we'd stream or sample.
	rows := [][]string{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return models.DataAnalysisResult{}, err
		}
		rows = append(rows, record)
	}
	result.NumRows = len(rows)
	result.NumColumns = len(headers)

	// Infer types for each column
	for i, colName := range headers {
		colType := inferColumnType(rows, i)
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
