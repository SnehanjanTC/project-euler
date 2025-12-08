package state

import (
	"backend-go/internal/models"
	"sync"
)

// DataFrame represents a loaded CSV file with its data
type DataFrame struct {
	Headers  []string
	Rows     [][]string
	FilePath string
	FileName string
}

// AppState holds the global application state
type AppState struct {
	mu sync.RWMutex

	// Loaded DataFrames
	DF1 *DataFrame
	DF2 *DataFrame

	// Context
	File1Context *models.Context
	File2Context *models.Context

	// Ollama Config
	OllamaBaseURL string
	OllamaModel   string
}

// Global state instance
var State = &AppState{
	OllamaBaseURL: "http://localhost:11434",
	OllamaModel:   "qwen3-vl:2b",
}

// SetDataFrame sets the dataframe for the given file index (1 or 2)
func (s *AppState) SetDataFrame(fileIndex int, df *DataFrame) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fileIndex == 1 {
		s.DF1 = df
	} else if fileIndex == 2 {
		s.DF2 = df
	}
}

// GetDataFrame retrieves the dataframe for the given file index
func (s *AppState) GetDataFrame(fileIndex int) *DataFrame {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if fileIndex == 1 {
		return s.DF1
	} else if fileIndex == 2 {
		return s.DF2
	}
	return nil
}

// SetContext sets context for the given file index
func (s *AppState) SetContext(fileIndex int, ctx *models.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fileIndex == 1 {
		s.File1Context = ctx
	} else if fileIndex == 2 {
		s.File2Context = ctx
	}
}

// GetContext retrieves context for the given file index
func (s *AppState) GetContext(fileIndex int) *models.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if fileIndex == 1 {
		return s.File1Context
	} else if fileIndex == 2 {
		return s.File2Context
	}
	return nil
}

// ClearContext clears context for a file or all files
func (s *AppState) ClearContext(fileIndex *int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fileIndex == nil {
		s.File1Context = nil
		s.File2Context = nil
	} else if *fileIndex == 1 {
		s.File1Context = nil
	} else if *fileIndex == 2 {
		s.File2Context = nil
	}
}

// GetNumericColumnIndices returns indices of numeric columns
func (df *DataFrame) GetNumericColumnIndices() map[int]bool {
	if len(df.Rows) == 0 {
		return nil
	}

	numericCols := make(map[int]bool)
	for colIdx := range df.Headers {
		isNumeric := true
		// Check first 20 rows (or all if fewer)
		checkRows := 20
		if len(df.Rows) < checkRows {
			checkRows = len(df.Rows)
		}
		for i := 0; i < checkRows; i++ {
			if colIdx >= len(df.Rows[i]) {
				continue
			}
			val := df.Rows[i][colIdx]
			if val == "" {
				continue
			}
			if !isNumericString(val) {
				isNumeric = false
				break
			}
		}
		if isNumeric {
			numericCols[colIdx] = true
		}
	}
	return numericCols
}

func isNumericString(s string) bool {
	if s == "" {
		return false
	}
	dotCount := 0
	for i, c := range s {
		if c == '-' && i == 0 {
			continue
		}
		if c == '.' {
			dotCount++
			if dotCount > 1 {
				return false
			}
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
