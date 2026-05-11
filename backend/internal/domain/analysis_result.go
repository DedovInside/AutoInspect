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
	ID           string            `json:"id"`
	DamageType   string            `json:"damage_type"`
	DamageNameRU string            `json:"damage_name_ru,omitempty"`
	Polygon      [][]int           `json:"polygon"`
	BBox         []int             `json:"bbox"`
	Confidence   float64           `json:"confidence"`
	Parts        []PartAssociation `json:"parts"`
}

type PartAssociation struct {
	Name         string  `json:"name"`
	NameRU       string  `json:"name_ru,omitempty"`
	ParentName   string  `json:"parent_name,omitempty"`
	ParentNameRU string  `json:"parent_name_ru,omitempty"`
	IsPair       bool    `json:"is_pair"`
	Side         string  `json:"side,omitempty"`
	SideRU       string  `json:"side_ru,omitempty"`
	Confidence   float64 `json:"confidence"`
}

type PartSummary struct {
	Name         string              `json:"name"`
	NameRU       string              `json:"name_ru,omitempty"`
	ParentName   string              `json:"parent_name,omitempty"`
	ParentNameRU string              `json:"parent_name_ru,omitempty"`
	IsPair       bool                `json:"is_pair"`
	Side         string              `json:"side,omitempty"`
	SideRU       string              `json:"side_ru,omitempty"`
	DamageCount  int                 `json:"damage_count"`
	DamageTypes  []DamageTypeSummary `json:"damage_types"`
}

type DamageTypeSummary struct {
	Code   string `json:"code"`
	NameRU string `json:"name_ru,omitempty"`
	Count  int    `json:"count"`
}
