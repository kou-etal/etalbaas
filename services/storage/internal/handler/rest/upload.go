package rest

import (
	"net/http"

	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

type UploadHandler struct {
	svc *service.ObjectService
}

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectID, claims, err := claimsFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	bucket := r.PathValue("bucket")
	objectPath := r.PathValue("path")
	if bucket == "" || objectPath == "" {
		writeError(w, http.StatusBadRequest, "bucket and path are required")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	result, err := h.svc.Upload(r.Context(), service.UploadParams{
		ProjectID:   projectID,
		BucketName:  bucket,
		ObjectPath:  objectPath,
		ContentType: contentType,
		Size:        r.ContentLength,
		Body:        r.Body,
		Claims:      claims,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":           result.Object.ID.String(),
		"bucket_id":    result.Object.BucketID,
		"name":         result.Object.Name,
		"size":         result.Object.Size,
		"mime_type":    result.Object.MimeType,
		"etag":         result.Object.Etag,
		"key":          result.Key,
		"created_at":   result.Object.CreatedAt,
		"updated_at":   result.Object.UpdatedAt,
	})
}
