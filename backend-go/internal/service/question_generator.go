package service

import (
	"backend-go/internal/llm"
	"backend-go/internal/models"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type QuestionGenerator struct {
	llmService *llm.Service
}

func NewQuestionGenerator(llmService *llm.Service) *QuestionGenerator {
	return &QuestionGenerator{
		llmService: llmService,
	}
}

var DomainOptions = []string{
	"Sales & Marketing",
	"Finance & Accounting",
	"Human Resources",
	"Operations & Supply Chain",
	"Customer Service",
	"Healthcare",
	"E-commerce",
	"Manufacturing",
	"Technology & IT",
	"Education",
	"Other",
}

// GenerateQuestions generates context questions for a dataset
func (s *QuestionGenerator) GenerateQuestions(analysis models.DataAnalysisResult, fileIndex int) []models.Question {
	questions := []models.Question{}

	// Q1: Dataset Purpose
	questions = append(questions, models.Question{
		ID:       fmt.Sprintf("f%d_purpose", fileIndex),
		Type:     models.QuestionTypeDatasetPurpose,
		Text:     fmt.Sprintf("What is the primary purpose of this dataset (File %d)?", fileIndex),
		Options:  []string{},
		Required: true,
		Metadata: map[string]interface{}{"placeholder": "e.g., Customer transaction records, Employee performance data, etc."},
	})

	// Q2: Business Domain
	questions = append(questions, models.Question{
		ID:       fmt.Sprintf("f%d_domain", fileIndex),
		Type:     models.QuestionTypeBusinessDomain,
		Text:     "Which business domain does this dataset belong to?",
		Options:  DomainOptions,
		Required: true,
		Metadata: map[string]interface{}{},
	})

	// Try AI questions
	aiQuestions := s.generateAIQuestions(analysis, fileIndex)
	if len(aiQuestions) > 0 {
		questions = append(questions, aiQuestions...)
	} else {
		// Fallback heuristics
		questions = append(questions, models.Question{
			ID:       fmt.Sprintf("f%d_entities", fileIndex),
			Type:     models.QuestionTypeKeyEntities,
			Text:     "What are the main entities or subjects in this dataset?",
			Options:  []string{},
			Required: true,
			Metadata: map[string]interface{}{
				"placeholder": "e.g., Customer, Product, Order",
				"input_type":  "tags",
				"hint":        "Enter multiple entities separated by commas",
			},
		})

		if analysis.HasDates {
			dateCols := strings.Join(takeFirst(analysis.PotentialDates, 3), ", ")
			questions = append(questions, models.Question{
				ID:       fmt.Sprintf("f%d_temporal", fileIndex),
				Type:     models.QuestionTypeTemporalContext,
				Text:     fmt.Sprintf("What time period does this data cover? (Found date columns: %s)", dateCols),
				Options:  []string{},
				Required: false,
				Metadata: map[string]interface{}{"placeholder": "e.g., Q1 2024, Last 12 months"},
			})
		}

		ambiguous := s.findAmbiguousColumns(analysis.ColumnNames)
		if len(ambiguous) > 0 {
			colList := strings.Join(takeFirst(ambiguous, 5), ", ")
			questions = append(questions, models.Question{
				ID:       fmt.Sprintf("f%d_column_semantics", fileIndex),
				Type:     models.QuestionTypeColumnSemantic,
				Text:     fmt.Sprintf("Can you briefly describe what these columns represent: %s?", colList),
				Options:  []string{},
				Required: false,
				Metadata: map[string]interface{}{
					"columns":    takeFirst(ambiguous, 5),
					"input_type": "column_descriptions",
				},
			})
		}
	}

	// Exclusions
	questions = append(questions, models.Question{
		ID:       fmt.Sprintf("f%d_exclusions", fileIndex),
		Type:     models.QuestionTypeExclusions,
		Text:     "Are there any columns that should be excluded from correlation analysis?",
		Options:  analysis.ColumnNames,
		Required: false,
		Metadata: map[string]interface{}{
			"input_type": "multi_select",
			"hint":       "Select columns like temporary fields, debug data, or irrelevant information",
		},
	})

	return questions
}

func (s *QuestionGenerator) generateAIQuestions(analysis models.DataAnalysisResult, fileIndex int) []models.Question {
	prompt := fmt.Sprintf(`
Analyze this dataset summary and generate 3 specific questions to understand its business context.

Dataset Summary:
- Columns: %s
- Row Count: %d
- Date Columns: %s
- ID Columns: %s

Generate 3 questions that would help clarify:
1. The specific business process this data represents
2. The meaning of any ambiguous columns
3. The time granularity or scope

Return a JSON object with a 'questions' array. Each question should have:
- 'text': The question text
- 'type': One of ['text', 'select', 'multi_select']
- 'options': Array of strings (only for select/multi_select)
- 'id_suffix': A unique suffix for the ID (e.g., 'process_type')

Example JSON:
{
	"questions": [
		{
			"text": "What type of transactions does this represent?",
			"type": "select",
			"options": ["Online Sales", "In-store POS"],
			"id_suffix": "trans_type"
		}
	]
}

Return ONLY the JSON.
`, strings.Join(takeFirst(analysis.ColumnNames, 20), ", "), analysis.NumRows, strings.Join(analysis.PotentialDates, ", "), strings.Join(analysis.PotentialIDs, ", "))

	response, err := s.llmService.CallOllama(prompt)
	if err != nil || response == "" {
		return nil
	}

	// Extract JSON
	jsonRegex := regexp.MustCompile(`\{[\s\S]*\}`)
	jsonStr := jsonRegex.FindString(response)
	if jsonStr == "" {
		return nil
	}

	var data struct {
		Questions []struct {
			Text     string   `json:"text"`
			Type     string   `json:"type"`
			Options  []string `json:"options"`
			IdSuffix string   `json:"id_suffix"`
		} `json:"questions"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil
	}

	aiQuestions := []models.Question{}
	for i, q := range data.Questions {
		qID := fmt.Sprintf("f%d_ai_%s", fileIndex, q.IdSuffix)
		if q.IdSuffix == "" {
			qID = fmt.Sprintf("f%d_ai_%d", fileIndex, i)
		}

		qType := models.QuestionTypeColumnSemantic
		textLower := strings.ToLower(q.Text)
		if strings.Contains(textLower, "entity") {
			qType = models.QuestionTypeKeyEntities
		} else if strings.Contains(textLower, "time") || strings.Contains(textLower, "date") {
			qType = models.QuestionTypeTemporalContext
		}

		aiQuestions = append(aiQuestions, models.Question{
			ID:       qID,
			Type:     qType,
			Text:     q.Text,
			Options:  q.Options,
			Required: false,
			Metadata: map[string]interface{}{"generated_by": "ai"},
		})
	}
	return aiQuestions
}

func (s *QuestionGenerator) findAmbiguousColumns(cols []string) []string {
	ambiguous := []string{}
	for _, col := range cols {
		colLower := strings.ToLower(col)
		clear := false
		for _, keyword := range []string{"id", "name", "email", "phone", "address", "date", "time", "amount", "price", "quantity", "status", "type", "category"} {
			if strings.Contains(colLower, keyword) {
				clear = true
				break
			}
		}
		if clear {
			continue
		}

		if len(col) <= 3 || (!strings.Contains(col, "_") && len(strings.Fields(col)) == 1) {
			ambiguous = append(ambiguous, col)
		}
	}
	return ambiguous
}

func takeFirst(s []string, n int) []string {
	if len(s) < n {
		return s
	}
	return s[:n]
}
