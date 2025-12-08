package models

// QuestionType constants
const (
	QuestionTypeDatasetPurpose  = "dataset_purpose"
	QuestionTypeBusinessDomain  = "business_domain"
	QuestionTypeKeyEntities     = "key_entities"
	QuestionTypeTemporalContext = "temporal_context"
	QuestionTypeColumnSemantic  = "column_semantic"
	QuestionTypeRelationships   = "relationships"
	QuestionTypeCustomMappings  = "custom_mappings"
	QuestionTypeExclusions      = "exclusions"
)

// Question represents a context collection question
type Question struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Text     string                 `json:"text"`
	Options  []string               `json:"options"`
	Required bool                   `json:"required"`
	Metadata map[string]interface{} `json:"metadata"`
}

// DataAnalysisResult holds analysis of a dataframe for question generation
type DataAnalysisResult struct {
	NumRows          int               `json:"rows"`
	NumColumns       int               `json:"columns"`
	ColumnNames      []string          `json:"column_names"`
	ColumnTypes      map[string]string `json:"column_types"`
	HasDates         bool              `json:"has_dates"`
	HasNumeric       bool              `json:"has_numeric"`
	HasText          bool              `json:"has_text"`
	PotentialIDs     []string          `json:"potential_ids"`
	PotentialDates   []string          `json:"potential_dates"`
	PotentialAmounts []string          `json:"potential_amounts"`
}
