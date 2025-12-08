package service

import (
	"backend-go/internal/models"
	"fmt"
	"strings"
	"time"
)

type ContextService struct {
	File1Context  *models.Context
	File2Context  *models.Context
	File1Analysis *models.DataAnalysisResult
	File2Analysis *models.DataAnalysisResult
}

func NewContextService() *ContextService {
	return &ContextService{}
}

func (s *ContextService) ValidateContext(ctx *models.Context) bool {
	if ctx == nil {
		return false
	}
	return ctx.DatasetPurpose != "" && ctx.BusinessDomain != ""
}

func (s *ContextService) MergeContext(existing *models.Context, newCtx *models.Context) *models.Context {
	if existing == nil {
		return newCtx
	}
	// Copy simple fields if new ones are present
	if newCtx.DatasetPurpose != "" {
		existing.DatasetPurpose = newCtx.DatasetPurpose
	}
	if newCtx.BusinessDomain != "" {
		existing.BusinessDomain = newCtx.BusinessDomain
	}
	if newCtx.TemporalContext != "" {
		existing.TemporalContext = newCtx.TemporalContext
	}

	// Merge slices and maps
	if len(newCtx.KeyEntities) > 0 {
		existing.KeyEntities = append(existing.KeyEntities, newCtx.KeyEntities...)
		existing.KeyEntities = uniqueStrings(existing.KeyEntities)
	}
	// Note: For maps, just taking the new keys. A deeper merge strategy could be applied if needed.
	for k, v := range newCtx.ColumnDescriptions {
		existing.ColumnDescriptions[k] = v
	}
	if len(newCtx.Relationships) > 0 {
		existing.Relationships = append(existing.Relationships, newCtx.Relationships...)
		existing.Relationships = uniqueStrings(existing.Relationships)
	}
	for k, v := range newCtx.CustomMappings {
		existing.CustomMappings[k] = v
	}
	if len(newCtx.Exclusions) > 0 {
		existing.Exclusions = append(existing.Exclusions, newCtx.Exclusions...)
		existing.Exclusions = uniqueStrings(existing.Exclusions)
	}

	existing.UpdatedAt = time.Now().Format(time.RFC3339)
	return existing
}

func (s *ContextService) BuildContextPrompt() string {
	if s.File1Context == nil && s.File2Context == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Consider the following context:\n")

	if s.File1Context != nil {
		sb.WriteString("File 1 Context:\n")
		sb.WriteString(fmt.Sprintf("  - Purpose: %s\n", s.File1Context.DatasetPurpose))
		sb.WriteString(fmt.Sprintf("  - Domain: %s\n", s.File1Context.BusinessDomain))
		if len(s.File1Context.KeyEntities) > 0 {
			sb.WriteString(fmt.Sprintf("  - Key Entities: %s\n", strings.Join(s.File1Context.KeyEntities, ", ")))
		}
		sb.WriteString("\n")
	}

	if s.File2Context != nil {
		sb.WriteString("File 2 Context:\n")
		sb.WriteString(fmt.Sprintf("  - Purpose: %s\n", s.File2Context.DatasetPurpose))
		sb.WriteString(fmt.Sprintf("  - Domain: %s\n", s.File2Context.BusinessDomain))
		if len(s.File2Context.KeyEntities) > 0 {
			sb.WriteString(fmt.Sprintf("  - Key Entities: %s\n", strings.Join(s.File2Context.KeyEntities, ", ")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// StoreContext updates the in-memory state
func (s *ContextService) StoreContext(fileIndex int, ctx *models.Context) error {
	if !s.ValidateContext(ctx) {
		return fmt.Errorf("invalid context data: missing required fields")
	}

	if fileIndex == 1 {
		s.File1Context = s.MergeContext(s.File1Context, ctx)
	} else if fileIndex == 2 {
		s.File2Context = s.MergeContext(s.File2Context, ctx)
	} else {
		return fmt.Errorf("invalid file_index: must be 1 or 2")
	}
	return nil
}

// GetContext retrieves context
func (s *ContextService) GetContext(fileIndex int) *models.Context {
	if fileIndex == 1 {
		return s.File1Context
	} else if fileIndex == 2 {
		return s.File2Context
	}
	return nil
}

// uniqueStrings helper
func uniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// StoreAnalysis updates the in-memory analysis state
func (s *ContextService) StoreAnalysis(fileIndex int, analysis *models.DataAnalysisResult) error {
	if fileIndex == 1 {
		s.File1Analysis = analysis
	} else if fileIndex == 2 {
		s.File2Analysis = analysis
	} else {
		return fmt.Errorf("invalid file_index: must be 1 or 2")
	}
	return nil
}

// GetAnalysis retrieves analysis
func (s *ContextService) GetAnalysis(fileIndex int) *models.DataAnalysisResult {
	if fileIndex == 1 {
		return s.File1Analysis
	} else if fileIndex == 2 {
		return s.File2Analysis
	}
	return nil
}
