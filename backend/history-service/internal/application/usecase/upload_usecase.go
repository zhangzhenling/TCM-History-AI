package usecase

import (
	"context"
	"io"
	"path"
	"path/filepath"
	"strings"
	"time"

	"tcm-history-ai/backend/pkg/errno"
	"tcm-history-ai/backend/pkg/idgen"
)

// StorageClient is the application-level port for the MinIO adapter.
type StorageClient interface {
	Upload(ctx context.Context, objectKey string, reader io.Reader, contentType string) (string, error)
	PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	Delete(ctx context.Context, objectKey string) error
}

// UploadUseCase handles file uploads for person portraits and book PDFs.
// The UseCase is intentionally generic: callers pass a "purpose" prefix
// (e.g. "portraits" or "books") and a filename; the usecase generates a
// unique object key, uploads via the StorageClient, and returns the object URL.
type UploadUseCase struct {
	storage StorageClient
}

// NewUploadUseCase constructs an UploadUseCase.
func NewUploadUseCase(storage StorageClient) *UploadUseCase {
	return &UploadUseCase{storage: storage}
}

// UploadResult is the response payload returned by Upload.
type UploadResult struct {
	ObjectKey string `json:"object_key"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
}

// Upload persists a file to object storage. The filename is sanitised and
// embedded into the object key under the given purpose prefix.
func (uc *UploadUseCase) Upload(ctx context.Context, purpose, filename, contentType string, size int64, body io.Reader) (*UploadResult, error) {
	if uc.storage == nil {
		return nil, errno.New(errno.DependencyUnavailable, "storage client not configured")
	}
	if strings.TrimSpace(filename) == "" {
		return nil, errno.New(errno.InvalidParams, "filename is required")
	}
	purpose = strings.Trim(strings.TrimSpace(purpose), "/")
	if purpose == "" {
		purpose = "misc"
	}
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	base = sanitize(base)
	if base == "" {
		base = "file"
	}
	objectKey := path.Join(purpose, time.Now().UTC().Format("2006/01/02"), base+"-"+idText()+ext)

	url, err := uc.storage.Upload(ctx, objectKey, body, contentType)
	if err != nil {
		return nil, err
	}
	return &UploadResult{
		ObjectKey: objectKey,
		URL:       url,
		Size:      size,
		MimeType:  contentType,
	}, nil
}

// sanitize strips characters that are unsafe in object keys.
func sanitize(s string) string {
	r := strings.NewReplacer(" ", "_", "/", "_", "\\", "_", ":", "_", "?", "_")
	return r.Replace(s)
}

// idText returns a string form of a fresh snowflake id for object key uniqueness.
func idText() string {
	const digits = "0123456789abcdefghijklmnopqrstuvwxyz"
	v := idgen.Next()
	if v == 0 {
		return "0"
	}
	buf := make([]byte, 0, 16)
	for v > 0 {
		buf = append([]byte{digits[v%36]}, buf...)
		v /= 36
	}
	return string(buf)
}
