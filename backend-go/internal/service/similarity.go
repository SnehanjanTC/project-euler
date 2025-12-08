package service

import (
	"backend-go/internal/models"
	"fmt"
	"strings"
)

type SimilarityService struct {
	ContextService *ContextService
}

func NewSimilarityService(ctxService *ContextService) *SimilarityService {
	return &SimilarityService{
		ContextService: ctxService,
	}
}

// GenerateGraph creates the similarity graph
func (s *SimilarityService) GenerateGraph(fileIndex1, fileIndex2 int) (*models.SimilarityGraph, error) {
	analysis1 := s.ContextService.GetAnalysis(fileIndex1)
	analysis2 := s.ContextService.GetAnalysis(fileIndex2)

	if analysis1 == nil || analysis2 == nil {
		return nil, fmt.Errorf("analysis not found for one or both files")
	}

	ctx1 := s.ContextService.GetContext(fileIndex1)
	ctx2 := s.ContextService.GetContext(fileIndex2)

	graph := &models.SimilarityGraph{
		Nodes:        []models.Node{},
		Edges:        []models.Edge{},
		Similarities: []models.Similarity{},
		Correlations: []models.Correlation{}, // Numeric only
	}

	// Create Nodes
	for _, col := range analysis1.ColumnNames {
		graph.Nodes = append(graph.Nodes, models.Node{ID: "f1_" + col, Label: col, Group: "File 1"})
	}
	for _, col := range analysis2.ColumnNames {
		graph.Nodes = append(graph.Nodes, models.Node{ID: "f2_" + col, Label: col, Group: "File 2"})
	}

	// Create Edges (Compare all vs all)
	for _, col1 := range analysis1.ColumnNames {
		for _, col2 := range analysis2.ColumnNames {
			simScore, details := s.calculateDetailedSimilarity(col1, col2, analysis1.ColumnTypes[col1], analysis2.ColumnTypes[col2], ctx1, ctx2)

			if simScore >= 30.0 { // Threshold
				// Add Similarity
				simEntry := models.Similarity{
					File1Column:    col1,
					File2Column:    col2,
					Similarity:     simScore / 100.0,
					Confidence:     simScore,
					Type:           details.Type,
					NameSimilarity: details.NameSim,
					DataSimilarity: details.DataSim,
					Reason:         details.Reason,
				}
				graph.Similarities = append(graph.Similarities, simEntry)

				// Add Edge
				edge := models.Edge{
					Source:     "f1_" + col1,
					Target:     "f2_" + col2,
					Value:      simScore / 10.0, // Weight for graph vis
					Similarity: simScore,
					Type:       details.Type,
				}
				graph.Edges = append(graph.Edges, edge)
			}
		}
	}

	graph.TotalRelationships = len(graph.Similarities)
	return graph, nil
}

type simDetails struct {
	Type    string
	NameSim float64
	DataSim float64
	Reason  string
}

func (s *SimilarityService) calculateDetailedSimilarity(col1, col2, type1, type2 string, ctx1, ctx2 *models.Context) (float64, simDetails) {
	nameSim := LevenshteinRatio(col1, col2) * 100
	dataSim := 0.0
	matchType := "unknown"

	// Simple Data Similarity based on Type
	if type1 == type2 {
		dataSim = 50.0 // Base score for matching type
		matchType = type1 + "_match"
		if type1 == "int" || type1 == "float" {
			dataSim = 80.0
		} else if type1 == "date" {
			dataSim = 90.0
		}
	} else {
		// Mismatch penalty
		dataSim = 0.0
		if (type1 == "int" && type2 == "float") || (type1 == "float" && type2 == "int") {
			dataSim = 60.0 // Numeric compatibility
			matchType = "numeric_compatible"
		}
	}

	// Weighted Score
	totalScore := (nameSim * 0.6) + (dataSim * 0.4)

	// Context Overrides
	if ctx1 != nil && ctx1.CustomMappings[col1] == col2 {
		totalScore = 95.0
		matchType = "custom_mapping"
	}

	return totalScore, simDetails{
		Type:    matchType,
		NameSim: nameSim,
		DataSim: dataSim,
	}
}

// LevenshteinRatio calculates similarity ratio (0-1)
func LevenshteinRatio(s1, s2 string) float64 {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)
	distance := levenshtein(s1, s2)
	maxLen := float64(max(len(s1), len(s2)))
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - (float64(distance) / maxLen)
}

func levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)

	row := make([]int, len2+1)
	for i := 0; i <= len2; i++ {
		row[i] = i
	}

	for i := 1; i <= len1; i++ {
		prev := i
		for j := 1; j <= len2; j++ {
			val := row[j]
			if r1[i-1] == r2[j-1] {
				val = row[j-1]
			} else {
				val = min(min(row[j-1]+1, prev+1), row[j]+1)
			}
			row[j-1] = prev
			prev = val
		}
		row[len2] = prev
	}
	return row[len2]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
