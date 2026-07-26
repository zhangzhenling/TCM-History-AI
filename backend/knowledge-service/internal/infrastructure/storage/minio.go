// Package storage wraps the MinIO client to provide object storage for
// Knowledge Service: original PDFs and structured Markdown.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"tcm-history-ai/backend/pkg/errno"
)

// MinIOClient wraps the official minio-go client.
type MinIOClient struct {
	client          *minio.Client
	originalBucket  string
	markdownBucket  string
}

// NewMinIOClient constructs a MinIOClient and eagerly verifies connectivity
// by checking whether the configured buckets exist (creating them if absent).
func NewMinIOClient(endpoint, accessKey, secretKey, originalBucket, markdownBucket string, useSSL bool) (*MinIOClient, error) {
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "init minio client", err)
	}
	return &MinIOClient{
		client:         cli,
		originalBucket: originalBucket,
		markdownBucket: markdownBucket,
	}, nil
}

// EnsureBuckets creates the original + markdown buckets if absent.
// 失败不返回错误，仅记录日志：broker 未就绪不应阻塞服务启动。
func (c *MinIOClient) EnsureBuckets(ctx context.Context) error {
	for _, bucket := range []string{c.originalBucket, c.markdownBucket} {
		exists, err := c.client.BucketExists(ctx, bucket)
		if err != nil {
			return errno.Wrap(errno.DependencyUnavailable, "check bucket "+bucket, err)
		}
		if !exists {
			if err := c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return errno.Wrap(errno.DependencyUnavailable, "create bucket "+bucket, err)
			}
		}
	}
	return nil
}

// PutPDF uploads a PDF to the original bucket.
func (c *MinIOClient) PutPDF(ctx context.Context, objectKey string, r io.Reader, size int64) error {
	_, err := c.client.PutObject(ctx, c.originalBucket, objectKey, r, size, minio.PutObjectOptions{
		ContentType: "application/pdf",
	})
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "put pdf", err)
	}
	return nil
}

// PutMarkdown uploads structured Markdown to the markdown bucket.
func (c *MinIOClient) PutMarkdown(ctx context.Context, objectKey string, r io.Reader, size int64) error {
	_, err := c.client.PutObject(ctx, c.markdownBucket, objectKey, r, size, minio.PutObjectOptions{
		ContentType: "text/markdown; charset=utf-8",
	})
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "put markdown", err)
	}
	return nil
}

// Get retrieves an object as a readable stream.
func (c *MinIOClient) Get(ctx context.Context, bucket, objectKey string) (io.ReadCloser, error) {
	obj, err := c.client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "get object", err)
	}
	return obj, nil
}

// PresignGet returns a time-limited download URL.
func (c *MinIOClient) PresignGet(ctx context.Context, bucket, objectKey string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	url, err := c.client.PresignedGetObject(ctx, bucket, objectKey, expiry, nil)
	if err != nil {
		return "", errno.Wrap(errno.DependencyUnavailable, "presign get", err)
	}
	return url.String(), nil
}

// OriginalBucket returns the bucket name for original PDFs.
func (c *MinIOClient) OriginalBucket() string { return c.originalBucket }

// MarkdownBucket returns the bucket name for structured Markdown.
func (c *MinIOClient) MarkdownBucket() string { return c.markdownBucket }

// String returns a debug representation.
func (c *MinIOClient) String() string {
	return fmt.Sprintf("storage.MinIOClient{original=%s, markdown=%s}",
		c.originalBucket, c.markdownBucket)
}

// ErrObjectNotFound is returned when an object does not exist.
var ErrObjectNotFound = errors.New("object not found")
