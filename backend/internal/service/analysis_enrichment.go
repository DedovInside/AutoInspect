package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
)

const (
	sideLeft  = "left"
	sideRight = "right"
)

type partsCatalogFile struct {
	ModelID string             `json:"model_id"`
	Parts   []partsCatalogItem `json:"parts"`
}

type partsCatalogItem struct {
	Name       string `json:"name"`
	NameEN     string `json:"name_en"`
	NameRU     string `json:"name_ru"`
	IsPair     bool   `json:"is_pair"`
	ParentName string `json:"parent_name"`
}

type partCatalogLookup struct {
	parts map[string]partsCatalogItem
}

type damageTypeLookup map[string]string

func (s *AnalysisService) enrichAnalysisResult(
	ctx context.Context,
	result *domain.AnalysisResult,
	model *domain.CarModel,
) error {
	if result == nil || model == nil {
		return nil
	}

	catalog, err := s.loadPartsCatalog(ctx, model.PartsCatalogS3Key)
	if err != nil {
		return err
	}

	damageTypes, err := s.loadDamageTypes(ctx)
	if err != nil {
		return err
	}

	enrichAnalysisResult(result, catalog, damageTypes)

	return nil
}

func (s *AnalysisService) loadPartsCatalog(ctx context.Context, objectKey string) (*partCatalogLookup, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return nil, domain.ErrInvalidModel
	}

	reader, err := s.fileRepo.Download(ctx, s.s3Cfg.BucketModels, objectKey)
	if err != nil {
		return nil, fmt.Errorf("download parts catalog %q: %w", objectKey, err)
	}

	defer func() {
		_ = reader.Close()
	}()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read parts catalog %q: %w", objectKey, err)
	}

	var catalog partsCatalogFile
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("parse parts catalog %q: %w", objectKey, err)
	}

	lookup := &partCatalogLookup{parts: make(map[string]partsCatalogItem, len(catalog.Parts))}
	for _, part := range catalog.Parts {
		code := partCode(part)
		if code == "" {
			continue
		}
		lookup.parts[code] = part
	}

	return lookup, nil
}

func (s *AnalysisService) loadDamageTypes(ctx context.Context) (damageTypeLookup, error) {
	if s.damageTypeRepo == nil {
		return damageTypeLookup{}, nil
	}

	damageTypes, err := s.damageTypeRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("load damage types: %w", err)
	}

	lookup := make(damageTypeLookup, len(damageTypes))
	for _, damageType := range damageTypes {
		if damageType == nil {
			continue
		}
		code := normalizeDamageTypeCode(damageType.Code)
		if code == "" {
			continue
		}
		lookup[code] = strings.TrimSpace(damageType.NameRU)
	}

	return lookup, nil
}

func enrichAnalysisResult(result *domain.AnalysisResult, catalog *partCatalogLookup, damageTypes damageTypeLookup) {
	if result == nil {
		return
	}

	for resultIdx := range result.Results {
		imageResult := &result.Results[resultIdx]

		for damageIdx := range imageResult.DamageInstances {
			damage := &imageResult.DamageInstances[damageIdx]
			damage.DamageNameRU = damageTypes.nameRU(damage.DamageType)

			for partIdx := range damage.Parts {
				enrichPartAssociation(&damage.Parts[partIdx], catalog)
			}
		}

		for summaryIdx := range imageResult.PartsSummary {
			enrichPartSummary(&imageResult.PartsSummary[summaryIdx], catalog, damageTypes)
		}
	}
}

func enrichPartAssociation(part *domain.PartAssociation, catalog *partCatalogLookup) {
	if part == nil {
		return
	}

	part.Name = normalizePartCode(part.Name)
	part.Side = normalizeSide(part.Side)
	part.SideRU = sideNameRU(part.Side)

	meta, ok := catalog.find(part.Name)
	if !ok {
		return
	}

	parentName, parentNameRU := catalog.parent(meta)
	part.NameRU = strings.TrimSpace(meta.NameRU)
	part.ParentName = parentName
	part.ParentNameRU = parentNameRU
	part.IsPair = meta.IsPair
}

func enrichPartSummary(summary *domain.PartSummary, catalog *partCatalogLookup, damageTypes damageTypeLookup) {
	if summary == nil {
		return
	}

	summary.Name = normalizePartCode(summary.Name)
	summary.Side = normalizeSide(summary.Side)
	summary.SideRU = sideNameRU(summary.Side)

	meta, ok := catalog.find(summary.Name)
	if ok {
		parentName, parentNameRU := catalog.parent(meta)
		summary.NameRU = strings.TrimSpace(meta.NameRU)
		summary.ParentName = parentName
		summary.ParentNameRU = parentNameRU
		summary.IsPair = meta.IsPair
	}

	for idx := range summary.DamageTypes {
		summary.DamageTypes[idx].NameRU = damageTypes.nameRU(summary.DamageTypes[idx].Code)
	}
}

func (c *partCatalogLookup) find(code string) (partsCatalogItem, bool) {
	if c == nil {
		return partsCatalogItem{}, false
	}
	part, ok := c.parts[normalizePartCode(code)]
	return part, ok
}

func (c *partCatalogLookup) parent(part partsCatalogItem) (parentName, parentNameRU string) {
	code := partCode(part)
	parentName = normalizePartCode(part.ParentName)
	if parentName == "" {
		parentName = code
	}

	if parentName == code {
		parentNameRU = strings.TrimSpace(part.NameRU)
	} else if parent, ok := c.find(parentName); ok && strings.TrimSpace(parent.NameRU) != "" {
		parentNameRU = strings.TrimSpace(parent.NameRU)
	}

	return parentName, parentNameRU
}

func partCode(part partsCatalogItem) string {
	if code := normalizePartCode(part.NameEN); code != "" {
		return code
	}
	return normalizePartCode(part.Name)
}

func normalizePartCode(code string) string {
	return strings.TrimSpace(code)
}

func normalizeSide(side string) string {
	return strings.ToLower(strings.TrimSpace(side))
}

func sideNameRU(side string) string {
	switch normalizeSide(side) {
	case sideLeft:
		return "слева"
	case sideRight:
		return "справа"
	default:
		return ""
	}
}

func (l damageTypeLookup) nameRU(code string) string {
	name, ok := l[normalizeDamageTypeCode(code)]
	if !ok {
		return ""
	}

	return name
}

func normalizeDamageTypeCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}
