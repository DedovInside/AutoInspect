package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidReviewRating(t *testing.T) {
	t.Parallel()

	require.False(t, IsValidReviewRating(0))
	require.True(t, IsValidReviewRating(1))
	require.True(t, IsValidReviewRating(5))
	require.False(t, IsValidReviewRating(6))
}
