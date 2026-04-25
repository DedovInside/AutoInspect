package service

import (
	"math"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	analysisv1 "github.com/DedovInside/AutoInspect/backend/internal/proto/gen/go/analysis/v1"
)

func DomainToProtoRequest(job *domain.AnalysisJob, model *domain.CarModel) *analysisv1.AnalysisRequest {
	if job == nil {
		return nil
	}
	if model == nil {
		return nil
	}

	return &analysisv1.AnalysisRequest{
		CorrelationId: job.CorrelationID.String(),
		UserId:        job.UserID.String(),
		CarInfo: &analysisv1.CarInfo{
			Make:       job.CarMake,
			Model:      job.CarModel,
			Generation: job.CarGeneration,
			Year:       carYearToProto(job.CarYear),
		},
		ImageS3Keys:       safeStringSlice(job.ImageKeys),
		ModelS3Key:        model.ModelS3Key,
		PartsCatalogS3Key: model.PartsCatalogS3Key,
	}
}

func carYearToProto(year int) int32 {
	if year < math.MinInt32 || year > math.MaxInt32 {
		return 0
	}
	// #nosec G115 -- bounds are checked above; car years are also request-validated.
	return int32(year)
}

func ProtoToDomainResult(protoResult *analysisv1.AnalysisResult) *domain.AnalysisResult {
	if protoResult == nil {
		return nil
	}

	return &domain.AnalysisResult{
		ModelID:      protoResult.ModelId,
		ModelVersion: protoResult.ModelVersion,
		BatchID:      protoResult.BatchId,
		Results:      protoImageResults(protoResult.Results),
	}
}

func protoImageResults(results []*analysisv1.ImageAnalysisResult) []domain.ImageAnalysisResult {
	out := make([]domain.ImageAnalysisResult, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue
		}
		out = append(out, domain.ImageAnalysisResult{
			ImageID:         result.ImageId,
			ImageURI:        result.ImageUri,
			ImageInfo:       protoImageMeta(result.Image),
			DamageInstances: protoDamages(result.DamageInstances),
			PartsSummary:    protoPartsSummary(result.PartsSummary),
		})
	}
	return out
}

func protoImageMeta(info *analysisv1.ImageInfo) domain.ImageMeta {
	if info == nil {
		return domain.ImageMeta{}
	}
	return domain.ImageMeta{
		Width:  int(info.Width),
		Height: int(info.Height),
	}
}

func protoDamages(damages []*analysisv1.DamageInstance) []domain.DamageInstance {
	out := make([]domain.DamageInstance, 0, len(damages))
	for _, d := range damages {
		if d == nil {
			continue
		}
		out = append(out, protoDamage(d))
	}
	return out
}

func protoDamage(d *analysisv1.DamageInstance) domain.DamageInstance {
	return domain.DamageInstance{
		ID:         d.Id,
		DamageType: d.DamageType,
		Confidence: float64(d.Confidence),
		BBox:       protoBBox(d.Bbox),
		Polygon:    protoPolygon(d.Polygon),
		Parts:      protoPartAssociations(d.Parts),
	}
}

func protoBBox(bbox *analysisv1.BBox) []int {
	if bbox == nil {
		return nil
	}
	return []int{
		int(bbox.XMin), int(bbox.YMin),
		int(bbox.XMax), int(bbox.YMax),
	}
}

func protoPolygon(points []*analysisv1.Point) [][]int {
	out := make([][]int, 0, len(points))
	for _, p := range points {
		if p != nil {
			out = append(out, []int{int(p.X), int(p.Y)})
		}
	}
	return out
}

func protoPartAssociations(parts []*analysisv1.PartAssociation) []domain.PartAssociation {
	out := make([]domain.PartAssociation, 0, len(parts))
	for _, p := range parts {
		if p != nil {
			out = append(out, domain.PartAssociation{
				Name:       p.Name,
				Side:       p.Side,
				Confidence: float64(p.Confidence),
			})
		}
	}
	return out
}

func protoPartsSummary(summary []*analysisv1.PartSummary) []domain.PartSummary {
	out := make([]domain.PartSummary, 0, len(summary))
	for _, ps := range summary {
		if ps == nil {
			continue
		}
		out = append(out, protoPartSummary(ps))
	}
	return out
}

func protoPartSummary(summary *analysisv1.PartSummary) domain.PartSummary {
	return domain.PartSummary{
		Name:        summary.Name,
		Side:        summary.Side,
		DamageCount: int(summary.DamageCount),
		DamageTypes: protoDamageTypeCounts(summary.DamageTypes),
	}
}

func protoDamageTypeCounts(counts map[string]int32) map[string]int {
	out := make(map[string]int, len(counts))
	for k, v := range counts {
		out[k] = int(v)
	}
	return out
}

func safeStringSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
