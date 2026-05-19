//go:build integration

package s3

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/testsupport"
	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testMinIOUser     = "minioadmin"
	testMinIOPassword = "minioadmin"
	testUploadsBucket = "autoinspect-uploads"
	testModelsBucket  = "autoinspect-models"
	testResultsBucket = "autoinspect-results"
)

func TestS3ClientObjectLifecycle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	endpoint := minIOEndpoint(t, ctx)
	client, err := New(ctx, &config.S3Config{
		Endpoint:        endpoint,
		PublicEndpoint:  endpoint,
		AccessKey:       testMinIOUser,
		SecretKey:       testMinIOPassword,
		Region:          "us-east-1",
		UsePathStyle:    true,
		BucketUploads:   testUploadsBucket,
		BucketModels:    testModelsBucket,
		BucketResults:   testResultsBucket,
		PresignedURLTTL: time.Minute,
	})
	require.NoError(t, err)
	createTestBuckets(t, ctx, client)

	const objectKey = "uploads/user/car.jpg"
	payload := []byte("fake-image")

	require.NoError(t, client.Upload(ctx, testUploadsBucket, objectKey, bytes.NewReader(payload), "image/jpeg", int64(len(payload))))
	exists, err := client.Exists(ctx, testUploadsBucket, objectKey)
	require.NoError(t, err)
	require.True(t, exists)

	reader, err := client.Download(ctx, testUploadsBucket, objectKey)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()
	downloaded, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, payload, downloaded)

	presignedURL, err := client.GetPresignedURL(ctx, testUploadsBucket, objectKey, time.Minute)
	require.NoError(t, err)
	parsed, err := url.Parse(presignedURL)
	require.NoError(t, err)
	require.Contains(t, parsed.Path, objectKey)

	require.NoError(t, client.Delete(ctx, testUploadsBucket, objectKey))
	exists, err = client.Exists(ctx, testUploadsBucket, objectKey)
	require.NoError(t, err)
	require.False(t, exists)
}

func minIOEndpoint(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "minio/minio:latest",
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER":     testMinIOUser,
				"MINIO_ROOT_PASSWORD": testMinIOPassword,
			},
			Cmd:        []string{"server", "/data"},
			WaitingFor: wait.ForHTTP("/minio/health/ready").WithPort("9000/tcp").WithStartupTimeout(60 * time.Second),
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

	endpoint, err := container.PortEndpoint(ctx, "9000/tcp", "http")
	require.NoError(t, err)
	return endpoint
}

func createTestBuckets(t *testing.T, ctx context.Context, client *Client) {
	t.Helper()

	for _, bucket := range []string{testUploadsBucket, testModelsBucket, testResultsBucket} {
		_, err := client.client.CreateBucket(ctx, &awss3.CreateBucketInput{Bucket: aws.String(bucket)})
		require.NoError(t, err)
	}
}
