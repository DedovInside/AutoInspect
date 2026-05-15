package s3

import (
	"context"
	"fmt"

	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	cfg           *config.S3Config
}

func New(ctx context.Context, cfg *config.S3Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("s3 config is nil")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("load aws sdk config: %w", err)
	}

	s3ClientOpts := clientOptions(cfg.Endpoint, cfg.UsePathStyle)

	s3Client := s3.NewFromConfig(awsCfg, s3ClientOpts)

	presignClient := s3.NewPresignClient(s3Client)
	if cfg.PublicEndpoint != "" {
		presignClient = s3.NewPresignClient(
			s3.NewFromConfig(awsCfg, clientOptions(cfg.PublicEndpoint, cfg.UsePathStyle)),
		)
	}

	return &Client{
		client:        s3Client,
		presignClient: presignClient,
		cfg:           cfg,
	}, nil
}

func clientOptions(endpoint string, usePathStyle bool) func(*s3.Options) {
	return func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		o.UsePathStyle = usePathStyle
	}
}

func (c *Client) HealthCheck(ctx context.Context) error {
	buckets := []string{c.cfg.BucketUploads, c.cfg.BucketModels, c.cfg.BucketResults}
	seen := make(map[string]struct{}, len(buckets))

	for _, bucket := range buckets {
		if bucket == "" {
			continue
		}

		if _, ok := seen[bucket]; ok {
			continue
		}
		seen[bucket] = struct{}{}

		_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})

		if err != nil {
			return fmt.Errorf("s3 health check failed for bucket %q: %w", bucket, err)
		}
	}

	return nil
}
