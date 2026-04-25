package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

func (c *Client) Upload(ctx context.Context, bucket, objectKey string, data io.Reader, contentType string, size int64) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(objectKey),
		Body:        data,
		ContentType: aws.String(contentType),
	}

	if size >= 0 {
		input.ContentLength = aws.Int64(size)
	}

	_, err := c.client.PutObject(ctx, input)

	if err != nil {
		return fmt.Errorf("s3 put object: %w", err)
	}

	return nil
}

func (c *Client) Download(ctx context.Context, bucket, objectKey string) (io.ReadCloser, error) {
	resp, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return nil, fmt.Errorf("s3 get object: %w", err)
	}

	return resp.Body, nil
}

func (c *Client) Exists(ctx context.Context, bucket, objectKey string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("s3 head object: %w", err)
	}

	return true, nil
}

func (c *Client) Delete(ctx context.Context, bucket, objectKey string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return fmt.Errorf("s3 delete object: %w", err)
	}

	return nil
}

func (c *Client) GetPresignedURL(ctx context.Context, bucket, objectKey string, expires time.Duration) (string, error) {
	req, err := c.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expires
	})

	if err != nil {
		return "", fmt.Errorf("s3 presign get object: %w", err)
	}

	return req.URL, nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr smithy.APIError

	if errors.As(err, &apiErr) {
		switch strings.TrimSpace(apiErr.ErrorCode()) {
		case "NotFound", "NoSuchKey", "NoSuchBucket":
			return true
		}
	}

	var respErr *smithyhttp.ResponseError

	if errors.As(err, &respErr) {
		return respErr.HTTPStatusCode() == 404
	}

	return false
}
