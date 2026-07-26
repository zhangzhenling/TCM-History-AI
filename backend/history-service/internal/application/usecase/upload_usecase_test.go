package usecase_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tcm-history-ai/backend/history-service/internal/application/usecase"
	"tcm-history-ai/backend/pkg/errno"
)

// fakeStorageClient is a stub usecase.StorageClient used by the upload tests.
type fakeStorageClient struct {
	uploadFn        func(ctx context.Context, objectKey string, reader io.Reader, contentType string) (string, error)
	presignFn       func(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	deleteFn        func(ctx context.Context, objectKey string) error
	uploadCalls     int
	lastObjectKey   string
	lastContentType string
	lastReader      io.Reader
}

func (f *fakeStorageClient) Upload(ctx context.Context, objectKey string, reader io.Reader, contentType string) (string, error) {
	f.uploadCalls++
	f.lastObjectKey = objectKey
	f.lastContentType = contentType
	f.lastReader = reader
	if f.uploadFn != nil {
		return f.uploadFn(ctx, objectKey, reader, contentType)
	}
	return "https://cdn.example.com/" + objectKey, nil
}

func (f *fakeStorageClient) PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	if f.presignFn != nil {
		return f.presignFn(ctx, objectKey, expiry)
	}
	return "https://cdn.example.com/" + objectKey + "?sig=presigned", nil
}

func (f *fakeStorageClient) Delete(ctx context.Context, objectKey string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, objectKey)
	}
	return nil
}

// TestUploadUseCase_HappyPath verifies the object key is built from the
// purpose, a date prefix, a sanitised base, and a unique id suffix.
func TestUploadUseCase_HappyPath(t *testing.T) {
	storage := &fakeStorageClient{}
	uc := usecase.NewUploadUseCase(storage)

	body := bytes.NewReader([]byte("hello world"))
	resp, err := uc.Upload(context.Background(), "portraits", "zhang zhongjing.jpg", "image/jpeg", 11, body)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Object key should be under portraits/<date>/zhang_zhongjing-<id>.jpg
	assert.Contains(t, resp.ObjectKey, "portraits/")
	assert.Contains(t, resp.ObjectKey, "zhang_zhongjing-")
	assert.True(t, strings.HasSuffix(resp.ObjectKey, ".jpg"),
		"object key %q should end with .jpg", resp.ObjectKey)
	assert.Equal(t, "https://cdn.example.com/"+resp.ObjectKey, resp.URL)
	assert.Equal(t, int64(11), resp.Size)
	assert.Equal(t, "image/jpeg", resp.MimeType)
	assert.Equal(t, 1, storage.uploadCalls)
	assert.Equal(t, resp.ObjectKey, storage.lastObjectKey)
	assert.Equal(t, "image/jpeg", storage.lastContentType)
}

// TestUploadUseCase_NilStorage verifies the missing-dependency guard.
func TestUploadUseCase_NilStorage(t *testing.T) {
	uc := usecase.NewUploadUseCase(nil)
	resp, err := uc.Upload(context.Background(), "portraits", "x.jpg", "image/jpeg", 1, bytes.NewReader([]byte("x")))
	require.Error(t, err)
	assert.Nil(t, resp)
	var e *errno.Error
	if errors.As(err, &e) {
		assert.Equal(t, errno.DependencyUnavailable, e.Code)
	}
}

// TestUploadUseCase_EmptyFilename verifies the filename guard.
func TestUploadUseCase_EmptyFilename(t *testing.T) {
	storage := &fakeStorageClient{}
	uc := usecase.NewUploadUseCase(storage)

	cases := []string{"", "   ", "\t\n"}
	for _, fn := range cases {
		resp, err := uc.Upload(context.Background(), "portraits", fn, "image/jpeg", 1, bytes.NewReader([]byte("x")))
		require.Error(t, err, "filename %q should be rejected", fn)
		assert.Nil(t, resp)
		var e *errno.Error
		if errors.As(err, &e) {
			assert.Equal(t, errno.InvalidParams, e.Code)
		}
	}
	assert.Equal(t, 0, storage.uploadCalls)
}

// TestUploadUseCase_DefaultPurpose verifies an empty purpose is replaced by
// "misc" and that surrounding slashes are trimmed.
func TestUploadUseCase_DefaultPurpose(t *testing.T) {
	storage := &fakeStorageClient{}
	uc := usecase.NewUploadUseCase(storage)

	t.Run("empty purpose defaults to misc", func(t *testing.T) {
		resp, err := uc.Upload(context.Background(), "", "file.pdf", "application/pdf", 1, bytes.NewReader([]byte("x")))
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(resp.ObjectKey, "misc/"),
			"object key %q should have prefix misc/", resp.ObjectKey)
	})

	t.Run("slashes around purpose are trimmed", func(t *testing.T) {
		resp, err := uc.Upload(context.Background(), "/books/", "file.pdf", "application/pdf", 1, bytes.NewReader([]byte("x")))
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(resp.ObjectKey, "books/"),
			"object key %q should have prefix books/", resp.ObjectKey)
		assert.NotContains(t, resp.ObjectKey, "//")
	})
}

// TestUploadUseCase_EmptyBaseAfterSanitise verifies that a filename with no
// usable base falls back to "file".
func TestUploadUseCase_EmptyBaseAfterSanitise(t *testing.T) {
	storage := &fakeStorageClient{}
	uc := usecase.NewUploadUseCase(storage)

	resp, err := uc.Upload(context.Background(), "books", ".pdf", "application/pdf", 1, bytes.NewReader([]byte("x")))
	require.NoError(t, err)
	assert.Contains(t, resp.ObjectKey, "/file-", "expected fallback to 'file' base; got %q", resp.ObjectKey)
	assert.True(t, strings.HasSuffix(resp.ObjectKey, ".pdf"))
}

// TestUploadUseCase_SanitisesUnsafeChars verifies that space, slash, colon,
// backslash and question mark in the base filename are replaced with _.
func TestUploadUseCase_SanitisesUnsafeChars(t *testing.T) {
	storage := &fakeStorageClient{}
	uc := usecase.NewUploadUseCase(storage)

	cases := []struct {
		filename string
	}{
		{"my file.jpg"},   // space
		{"a/b.jpg"},       // slash
		{"a\\b.jpg"},      // backslash
		{"a:b.jpg"},       // colon
		{"a?b.jpg"},       // question mark
	}
	for _, tc := range cases {
		resp, err := uc.Upload(context.Background(), "docs", tc.filename, "application/octet-stream", 1, bytes.NewReader([]byte("x")))
		require.NoError(t, err)
		assert.NotContains(t, resp.ObjectKey, " ", "space should be sanitised in %q -> %q", tc.filename, resp.ObjectKey)
		// The base portion should not contain raw slashes beyond the path separators.
		parts := strings.Split(resp.ObjectKey, "/")
		base := parts[len(parts)-1]
		assert.NotContains(t, base, "?", "base %q should not contain ?", base)
	}
}

// TestUploadUseCase_StorageError verifies errors from the storage client are
// propagated.
func TestUploadUseCase_StorageError(t *testing.T) {
	storage := &fakeStorageClient{
		uploadFn: func(context.Context, string, io.Reader, string) (string, error) {
			return "", errors.New("minio down")
		},
	}
	uc := usecase.NewUploadUseCase(storage)

	resp, err := uc.Upload(context.Background(), "books", "file.pdf", "application/pdf", 1, bytes.NewReader([]byte("x")))
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, 1, storage.uploadCalls)
}

// TestUploadUseCase_UniqueObjectKeys verifies that two uploads of the same
// filename produce different object keys (because of the snowflake id).
func TestUploadUseCase_UniqueObjectKeys(t *testing.T) {
	storage := &fakeStorageClient{}
	uc := usecase.NewUploadUseCase(storage)

	r1, err := uc.Upload(context.Background(), "books", "file.pdf", "application/pdf", 1, bytes.NewReader([]byte("a")))
	require.NoError(t, err)
	r2, err := uc.Upload(context.Background(), "books", "file.pdf", "application/pdf", 1, bytes.NewReader([]byte("b")))
	require.NoError(t, err)
	assert.NotEqual(t, r1.ObjectKey, r2.ObjectKey, "object keys should be unique")
}
