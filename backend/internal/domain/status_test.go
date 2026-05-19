package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoleIsValid(t *testing.T) {
	t.Parallel()

	require.True(t, RoleUser.IsValid())
	require.True(t, RoleCarService.IsValid())
	require.True(t, RoleAdmin.IsValid())
	require.False(t, Role("owner").IsValid())
}

func TestModelTrainingRequestStatusHelpers(t *testing.T) {
	t.Parallel()

	require.True(t, ModelTrainingRequestStatusPending.IsValid())
	require.True(t, ModelTrainingRequestStatusApproved.IsActive())
	require.True(t, ModelTrainingRequestStatusInProgress.IsActive())
	require.False(t, ModelTrainingRequestStatusCompleted.IsActive())
	require.True(t, ModelTrainingRequestStatusRejected.IsTerminal())
	require.True(t, ModelTrainingRequestStatusCompleted.IsTerminal())
	require.False(t, ModelTrainingRequestStatus("queued").IsValid())
}

func TestRepairRequestStatusHelpers(t *testing.T) {
	t.Parallel()

	require.True(t, RepairRequestStatusPending.IsValid())
	require.False(t, RepairRequestStatusPending.IsTerminal())
	require.True(t, RepairRequestStatusAccepted.IsTerminal())
	require.True(t, RepairRequestStatusRejected.IsTerminal())
	require.True(t, RepairRequestStatusCanceled.IsTerminal())
	require.False(t, RepairRequestStatus("done").IsValid())
}
