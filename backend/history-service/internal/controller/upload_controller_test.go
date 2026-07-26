package controller_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"tcm-history-ai/backend/history-service/internal/application/usecase"
	"tcm-history-ai/backend/history-service/internal/controller"
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
}

func (f *fakeStorageClient) Upload(ctx context.Context, objectKey string, reader io.Reader, contentType string) (string, error) {
	f.uploadCalls++
	f.lastObjectKey = objectKey
	f.lastContentType = contentType
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

func newUploadController(storage *fakeStorageClient) *controller.UploadController {
	uc := usecase.NewUploadUseCase(storage)
	return controller.NewUploadController(uc)
}

// TestNewUploadController_NotNil verifies the constructor returns a non-nil
// controller.
func TestNewUploadController_NotNil(t *testing.T) {
	uc := usecase.NewUploadUseCase(&fakeStorageClient{})
	ctrl := controller.NewUploadController(uc)
	assert.NotNil(t, ctrl)
}

// multipartBody builds a multipart/form-data body with a single "file" part
// (and an optional "purpose" field) and returns the body bytes plus the
// Content-Type header value (including the boundary).
func multipartBody(t *testing.T, filename, contentType, purpose, content string) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteString("--boundary123456\r\n")
	if purpose != "" {
		buf.WriteString("Content-Disposition: form-data; name=\"purpose\"\r\n\r\n")
		buf.WriteString(purpose + "\r\n")
		buf.WriteString("--boundary123456\r\n")
	}
	buf.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"" + filename + "\"\r\n")
	buf.WriteString("Content-Type: " + contentType + "\r\n\r\n")
	buf.WriteString(content + "\r\n")
	buf.WriteString("--boundary123456--\r\n")
	return buf.Bytes(), "multipart/form-data; boundary=boundary123456"
}

// TestUploadController_Upload covers the multipart upload endpoint.
func TestUploadController_Upload(t *testing.T) {
	t.Run("happy path returns 200 with object key and URL", func(t *testing.T) {
		storage := &fakeStorageClient{}
		ctrl := newUploadController(storage)

		body, ct := multipartBody(t, "portrait.jpg", "image/jpeg", "portraits", "fake-image-bytes")
		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/upload")
		rc.Request.Header.Set("Content-Type", ct)
		rc.Request.SetBody(body)

		ctrl.Upload(ctx(), rc)

		bodyResp := assertStatusCode(t, rc, http.StatusOK)
		assert.Equal(t, int(errno.OK), bodyResp.Code)
		assert.Equal(t, 1, storage.uploadCalls, "storage Upload should be called once")
		assert.Contains(t, storage.lastObjectKey, "portraits/")
	})

	t.Run("missing file field returns 400", func(t *testing.T) {
		storage := &fakeStorageClient{}
		ctrl := newUploadController(storage)

		// Submit a multipart body without the "file" field.
		var buf bytes.Buffer
		buf.WriteString("--boundary123456\r\n")
		buf.WriteString("Content-Disposition: form-data; name=\"purpose\"\r\n\r\n")
		buf.WriteString("portraits\r\n")
		buf.WriteString("--boundary123456--\r\n")

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/upload")
		rc.Request.Header.Set("Content-Type", "multipart/form-data; boundary=boundary123456")
		rc.Request.SetBody(buf.Bytes())

		ctrl.Upload(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
		assert.Equal(t, 0, storage.uploadCalls, "storage Upload should not be called when file is missing")
	})

	t.Run("non-multipart body returns 400", func(t *testing.T) {
		storage := &fakeStorageClient{}
		ctrl := newUploadController(storage)

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/upload")
		rc.Request.Header.Set("Content-Type", "application/json")
		rc.Request.SetBody([]byte(`{"hello":"world"}`))

		ctrl.Upload(ctx(), rc)
		assertStatusCode(t, rc, http.StatusBadRequest)
		assert.Equal(t, 0, storage.uploadCalls)
	})

	t.Run("storage error returns 500", func(t *testing.T) {
		storage := &fakeStorageClient{
			uploadFn: func(context.Context, string, io.Reader, string) (string, error) {
				return "", assertError("minio down")
			},
		}
		ctrl := newUploadController(storage)

		body, ct := multipartBody(t, "file.pdf", "application/pdf", "books", "x")
		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/upload")
		rc.Request.Header.Set("Content-Type", ct)
		rc.Request.SetBody(body)

		ctrl.Upload(ctx(), rc)
		assert.Equal(t, http.StatusInternalServerError, rc.Response.StatusCode())
		assert.Equal(t, 1, storage.uploadCalls)
	})

	t.Run("empty purpose defaults to misc", func(t *testing.T) {
		storage := &fakeStorageClient{}
		ctrl := newUploadController(storage)

		// No purpose field in the multipart body.
		var buf bytes.Buffer
		buf.WriteString("--boundary123456\r\n")
		buf.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"file.pdf\"\r\n")
		buf.WriteString("Content-Type: application/pdf\r\n\r\n")
		buf.WriteString("x\r\n")
		buf.WriteString("--boundary123456--\r\n")

		rc := newRC()
		rc.Request.SetMethod("POST")
		rc.Request.SetRequestURI("/api/v1/history/upload")
		rc.Request.Header.Set("Content-Type", "multipart/form-data; boundary=boundary123456")
		rc.Request.SetBody(buf.Bytes())

		ctrl.Upload(ctx(), rc)
		assertStatusCode(t, rc, http.StatusOK)
		assert.Contains(t, storage.lastObjectKey, "misc/")
	})
}
