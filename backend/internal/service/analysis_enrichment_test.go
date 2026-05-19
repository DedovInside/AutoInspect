package service

import (
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

const (
	testFenderNameRU = "переднее крыло"
	testSideLeftRU   = "слева"
	testSideRightRU  = "справа"
)

func TestEnrichAnalysisResultAddsRussianNamesAndParentCategories(t *testing.T) {
	t.Parallel()

	result := &domain.AnalysisResult{
		Results: []domain.ImageAnalysisResult{
			{
				DamageInstances: []domain.DamageInstance{
					{
						ID:         "damage-1",
						DamageType: " Dent ",
						Parts: []domain.PartAssociation{
							{Name: " front-fender ", Side: " LEFT ", Confidence: 0.91},
							{Name: "unknown-part", Side: "right", Confidence: 0.5},
						},
					},
				},
				PartsSummary: []domain.PartSummary{
					{
						Name:        "front-fender",
						Side:        "right",
						DamageCount: 2,
						DamageTypes: []domain.DamageTypeSummary{
							{Code: "Dent", Count: 2},
						},
					},
				},
			},
		},
	}
	catalog := &partCatalogLookup{
		parts: map[string]partsCatalogItem{
			"fender": {
				NameEN: "fender",
				NameRU: "крыло",
				IsPair: true,
			},
			"front-fender": {
				NameEN:     "front-fender",
				NameRU:     testFenderNameRU,
				IsPair:     true,
				ParentName: "fender",
			},
		},
	}
	damageTypes := damageTypeLookup{
		"dent": "вмятина",
	}

	enrichAnalysisResult(result, catalog, damageTypes)

	damage := result.Results[0].DamageInstances[0]
	require.Equal(t, "вмятина", damage.DamageNameRU)
	require.Len(t, damage.Parts, 2)

	part := damage.Parts[0]
	require.Equal(t, "front-fender", part.Name)
	require.Equal(t, testFenderNameRU, part.NameRU)
	require.Equal(t, "fender", part.ParentName)
	require.Equal(t, "крыло", part.ParentNameRU)
	require.Equal(t, "left", part.Side)
	require.Equal(t, testSideLeftRU, part.SideRU)
	require.True(t, part.IsPair)

	unknown := damage.Parts[1]
	require.Equal(t, "unknown-part", unknown.Name)
	require.Empty(t, unknown.NameRU)
	require.Equal(t, "right", unknown.Side)
	require.Equal(t, testSideRightRU, unknown.SideRU)

	summary := result.Results[0].PartsSummary[0]
	require.Equal(t, "front-fender", summary.Name)
	require.Equal(t, testFenderNameRU, summary.NameRU)
	require.Equal(t, "fender", summary.ParentName)
	require.Equal(t, "крыло", summary.ParentNameRU)
	require.Equal(t, "right", summary.Side)
	require.Equal(t, testSideRightRU, summary.SideRU)
	require.True(t, summary.IsPair)
	require.Equal(t, []domain.DamageTypeSummary{{Code: "Dent", NameRU: "вмятина", Count: 2}}, summary.DamageTypes)
}

func TestPartCatalogParentFallsBackToCurrentPart(t *testing.T) {
	t.Parallel()

	catalog := &partCatalogLookup{
		parts: map[string]partsCatalogItem{
			"hood": {NameEN: "hood", NameRU: "капот"},
		},
	}

	parentName, parentNameRU := catalog.parent(partsCatalogItem{NameEN: "hood", NameRU: "капот"})

	require.Equal(t, "hood", parentName)
	require.Equal(t, "капот", parentNameRU)
}

func TestDamageTypeLookupNormalizesCodes(t *testing.T) {
	t.Parallel()

	lookup := damageTypeLookup{"scratch": "царапина"}

	require.Equal(t, "царапина", lookup.nameRU(" Scratch "))
	require.Empty(t, lookup.nameRU("unknown"))
}

func TestSideNameRU(t *testing.T) {
	t.Parallel()

	require.Equal(t, testSideLeftRU, sideNameRU(" LEFT "))
	require.Equal(t, testSideRightRU, sideNameRU("right"))
	require.Empty(t, sideNameRU("center"))
}
