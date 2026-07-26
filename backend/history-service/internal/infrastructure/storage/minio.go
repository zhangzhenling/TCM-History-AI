// Package storage wraps minio-go/v7 to provide object upload, download,
// delete, and presigned URL operations.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"tcm-history-ai/backend/pkg/errno"
)

// MinIOClient wraps the official minio-go client.
type MinIOClient struct {
	client   *minio.Client
	bucket   string
	endpoint string
	useSSL   bool
}

// NewMinIOClient constructs a MinIOClient. It does not connect eagerly.
func NewMinIOClient(endpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinIOClient, error) {
	if endpoint == "" {
		return nil, errors.New("minio endpoint is required")
	}
	if bucket == "" {
		return nil, errors.New("minio bucket is required")
	}
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}
	return &MinIOClient{
		client:   cli,
		bucket:   bucket,
		endpoint: endpoint,
		useSSL:   useSSL,
	}, nil
}

// EnsureBucket creates the bucket if it does not yet exist.
func (c *MinIOClient) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "minio bucket exists", err)
	}
	if !exists {
		if err := c.client.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
			return errno.Wrap(errno.DependencyUnavailable, "minio make bucket", err)
		}
	}
	return nil
}

// Upload stores an object and returns its public URL.
func (c *MinIOClient) Upload(ctx context.Context, objectKey string, reader io.Reader, contentType string) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	_, err := c.client.PutObject(ctx, c.bucket, objectKey, reader, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", errno.Wrap(errno.DependencyUnavailable, "minio put object", err)
	}
	return c.objectURL(objectKey), nil
}

// Download returns a reader for the object. The caller must close it.
func (c *MinIOClient) Download(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	obj, err := c.client.GetObject(ctx, c.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, errno.Wrap(errno.DependencyUnavailable, "minio get object", err)
	}
	return obj, nil
}

// PresignedGetURL returns a time-limited URL that allows anonymous GET access.
func (c *MinIOClient) PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}
	u, err := c.client.PresignedGetObject(ctx, c.bucket, objectKey, expiry, url.Values{})
	if err != nil {
		return "", errno.Wrap(errno.DependencyUnavailable, "minio presigned get", err)
	}
	return u.String(), nil
}

// Delete removes an object.
func (c *MinIOClient) Delete(ctx context.Context, objectKey string) error {
	if err := c.client.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return errno.Wrap(errno.DependencyUnavailable, "minio remove object", err)
	}
	return nil
}

// objectURL builds the canonical URL for an object key under this client's
// endpoint. Used as the persistent file_url stored in the database.
func (c *MinIOClient) objectURL(objectKey string) string {
	scheme := "http"
	if c.useSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, c.endpoint, c.bucket, objectKey)
}
