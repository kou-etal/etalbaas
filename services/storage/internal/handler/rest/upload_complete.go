package rest

import (
	"encoding/json"
	"net/http"

	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

type UploadCompleteHandler struct {
	svc *service.ObjectService
}

type uploadCompleteRequest struct {
	BucketName string `json:"bucket_name"`
	ObjectPath string `json:"object_path"`
}

func (h *UploadCompleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectID, claims, err := claimsFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req uploadCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.BucketName == "" || req.ObjectPath == "" {
		writeError(w, http.StatusBadRequest, "bucket_name and object_path are required")
		return
	}

	obj, err := h.svc.UploadComplete(r.Context(), service.UploadCompleteParams{
		ProjectID:  projectID,
		BucketName: req.BucketName,
		ObjectPath: req.ObjectPath,
		Claims:     claims,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         obj.ID.String(),
		"bucket_id":  obj.BucketID,
		"name":       obj.Name,
		"size":       obj.Size,
		"mime_type":  obj.MimeType,
		"etag":       obj.Etag,
		"created_at": obj.CreatedAt,
		"updated_at": obj.UpdatedAt,
	})
}
