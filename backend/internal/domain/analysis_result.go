package domain

type AnalysisResult struct {
	ImageInfo       ImageMeta        `json:"image"`
	ModelVersion    string           `json:"model_version"`
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
	PartName   string  `json:"part_name"`
	Confidence float64 `json:"confidence"`
}

type PartSummary struct {
	PartName    string         `json:"part_name"`
	DamageCount int            `json:"damage_count"`
	DamageTypes map[string]int `json:"damage_types"` // "dent": 1, "scratch": 2
}
