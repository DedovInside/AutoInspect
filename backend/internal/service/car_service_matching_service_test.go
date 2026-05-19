package service

import (
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestMatchCriteriaFromAnalysisJobUsesPartSummaryParentAndDeduplicates(t *testing.T) {
	t.Parallel()

	job := &domain.AnalysisJob{
		Status: domain.StatusCompleted,
		Result: &domain.AnalysisResult{
			Results: []domain.ImageAnalysisResult{
				{
					PartsSummary: []domain.PartSummary{
						{
							Name:       "front-fender",
							ParentName: " fender ",
							DamageTypes: []domain.DamageTypeSummary{
								{Code: "Dent", Count: 1},
								{Code: " Scratch ", Count: 2},
							},
						},
						{
							Name:       "back-fender",
							ParentName: "fender",
							DamageTypes: []domain.DamageTypeSummary{
								{Code: "dent", Count: 1},
							},
						},
					},
				},
				{
					PartsSummary: []domain.PartSummary{
						{
							Name: "hood",
							DamageTypes: []domain.DamageTypeSummary{
								{Code: "Dent", Count: 1},
								{Code: "", Count: 1},
							},
						},
					},
				},
			},
		},
	}

	got, err := matchCriteriaFromAnalysisJob(job)

	require.NoError(t, err)
	require.ElementsMatch(t, []domain.CarServiceMatchCriterion{
		{DamageTypeCode: "dent", PartCategoryCode: "fender"},
		{DamageTypeCode: "scratch", PartCategoryCode: "fender"},
		{DamageTypeCode: "dent", PartCategoryCode: "hood"},
	}, got)
}

func TestMatchCriteriaFromAnalysisJobRejectsNotReadyJobs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		job  *domain.AnalysisJob
		want error
	}{
		{name: "nil", job: nil, want: domain.ErrJobNotFound},
		{name: "pending", job: &domain.AnalysisJob{Status: domain.StatusPending}, want: domain.ErrJobNotReady},
		{name: "failed", job: &domain.AnalysisJob{Status: domain.StatusFailed}, want: domain.ErrJobFailed},
		{name: "completed without result", job: &domain.AnalysisJob{Status: domain.StatusCompleted}, want: domain.ErrJobNotReady},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := matchCriteriaFromAnalysisJob(tt.job)

			require.Nil(t, got)
			require.ErrorIs(t, err, tt.want)
		})
	}
}

func TestMatchPartCategoryCodeFallsBackToPartName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "hood", matchPartCategoryCode(&domain.PartSummary{Name: " hood "}))
	require.Equal(t, "fender", matchPartCategoryCode(&domain.PartSummary{Name: "front-fender", ParentName: " fender "}))
	require.Empty(t, matchPartCategoryCode(nil))
}
