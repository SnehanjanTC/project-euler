package models

type SimilarityGraph struct {
	Nodes              []Node        `json:"nodes"`
	Edges              []Edge        `json:"edges"`
	Similarities       []Similarity  `json:"similarities"`
	TotalRelationships int           `json:"total_relationships"`
	Correlations       []Correlation `json:"correlations"`
}

type Node struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Group string `json:"group"`
}

type Edge struct {
	Source     string  `json:"source"`
	Target     string  `json:"target"`
	Value      float64 `json:"value"`
	Similarity float64 `json:"similarity"`
	Type       string  `json:"type"`
}

type Similarity struct {
	File1Column            string  `json:"file1_column"`
	File2Column            string  `json:"file2_column"`
	Similarity             float64 `json:"similarity"`
	Confidence             float64 `json:"confidence"`
	Type                   string  `json:"type"`
	DataSimilarity         float64 `json:"data_similarity"`
	NameSimilarity         float64 `json:"name_similarity"`
	DistributionSimilarity float64 `json:"distribution_similarity"`
	JSONConfidence         float64 `json:"json_confidence"` // Pattern score
	LLMSemanticScore       float64 `json:"llm_semantic_score"`
	Reason                 string  `json:"reason,omitempty"`
}

type Correlation struct {
	File1Column         string  `json:"file1_column"`
	File2Column         string  `json:"file2_column"`
	PearsonCorrelation  float64 `json:"pearson_correlation"`
	SpearmanCorrelation float64 `json:"spearman_correlation"`
	Strength            string  `json:"strength"`
	SampleSize          int     `json:"sample_size"`
}
