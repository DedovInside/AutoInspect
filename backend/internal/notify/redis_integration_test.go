//go:build integration

package notify

import (
	"context"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/testsupport"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRedisNotifierPublishesAndSubscribesJobEvent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	address := redisAddress(t, ctx)
	client := goredis.NewClient(&goredis.Options{Addr: address})
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	const channel = "notify:test:analysis"
	notifier, err := NewRedisNotifier(client, channel)
	require.NoError(t, err)

	subscribeCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)
	events := make(chan JobEvent, 1)
	go func() {
		_ = notifier.Subscribe(subscribeCtx, func(_ context.Context, event JobEvent) {
			events <- event
		})
	}()

	time.Sleep(150 * time.Millisecond)
	jobID := uuid.New()
	err = notifier.NotifyJobEvent(ctx, &JobEvent{
		JobID:  jobID,
		Type:   EventAnalysisCompleted,
		Status: "completed",
	})
	require.NoError(t, err)

	select {
	case event := <-events:
		require.Equal(t, jobID, event.JobID)
		require.Equal(t, EventAnalysisCompleted, event.Type)
		require.Equal(t, "completed", event.Status)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for redis notification")
	}
}

func redisAddress(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		testsupport.SkipIfDockerUnavailable(t, err)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	address, err := container.PortEndpoint(ctx, "6379/tcp", "")
	require.NoError(t, err)
	return address
}
