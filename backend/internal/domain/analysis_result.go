package domain

type AnalysisResult struct {
	ModelID      string                `json:"model_id"`
	ModelVersion string                `json:"model_version"`
	BatchID      string                `json:"batch_id"`
	Results      []ImageAnalysisResult `json:"results"`
}

type ImageAnalysisResult struct {
	ImageID         string           `json:"image_id"`
	ImageURI        string           `json:"image_uri"`
	ImageInfo       ImageMeta        `json:"image"`
	DamageInstances []DamageInstance `json:"damage_instances"`
	PartsSummary    []PartSummary    `json:"parts_summary"`
}

type ImageMeta struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type DamageInstance struct {
	ID         string            `json:"id"`
	DamageType string            `json:"damage_type"`
	Polygon    [][]int           `json:"polygon"`
	BBox       []int             `json:"bbox"`
	Confidence float64           `json:"confidence"`
	Parts      []PartAssociation `json:"parts"`
}

type PartAssociation struct {
	Name       string  `json:"name"`
	Side       string  `json:"side,omitempty"`
	Confidence float64 `json:"confidence"`
}

type PartSummary struct {
	Name        string         `json:"name"`
	Side        string         `json:"side,omitempty"`
	DamageCount int            `json:"damage_count"`
	DamageTypes map[string]int `json:"damage_types"`
}
