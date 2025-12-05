package models

import "time"

// Context represents the business context for a dataset
type Context struct {
	DatasetPurpose     string            `json:"dataset_purpose"`
	BusinessDomain     string            `json:"business_domain"`
	KeyEntities        []string          `json:"key_entities"`
	TemporalContext    string            `json:"temporal_context,omitempty"`
	ColumnDescriptions map[string]string `json:"column_descriptions"`
	Relationships      []string          `json:"relationships"`
	CustomMappings     map[string]string `json:"custom_mappings"`
	Exclusions         []string          `json:"exclusions"`
	CreatedAt          string            `json:"created_at"`
	UpdatedAt          string            `json:"updated_at"`
}

// NewContext creates a new empty Context with initialized maps/slices
func NewContext() *Context {
	now := time.Now().Format(time.RFC3339)
	return &Context{
		KeyEntities:        []string{},
		ColumnDescriptions: make(map[string]string),
		Relationships:      []string{},
		CustomMappings:     make(map[string]string),
		Exclusions:         []string{},
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}
