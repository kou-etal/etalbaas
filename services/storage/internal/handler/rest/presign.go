package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

type PresignHandler struct {
	svc *service.ObjectService
}

type presignRequest struct {
	BucketName string `json:"bucket_name"`
	ObjectPath string `json:"object_path"`
	ExpirySec  int    `json:"expiry_seconds"`
}

func (h *PresignHandler) Upload(w http.ResponseWriter, r *http.Request) {
	projectID, claims, err := claimsFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req presignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.BucketName == "" || req.ObjectPath == "" {
		writeError(w, http.StatusBadRequest, "bucket_name and object_path are required")
		return
	}

	expiry := time.Duration(req.ExpirySec) * time.Second

	result, err := h.svc.PresignUpload(r.Context(), service.PresignUploadParams{
		ProjectID:  projectID,
		BucketName: req.BucketName,
		ObjectPath: req.ObjectPath,
		Expiry:     expiry,
		Claims:     claims,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": result.URL})
}

func (h *PresignHandler) Download(w http.ResponseWriter, r *http.Request) {
	projectID, claims, err := claimsFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req presignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.BucketName == "" || req.ObjectPath == "" {
		writeError(w, http.StatusBadRequest, "bucket_name and object_path are required")
		return
	}

	expiry := time.Duration(req.ExpirySec) * time.Second

	result, err := h.svc.PresignDownload(r.Context(), service.PresignDownloadParams{
		ProjectID:  projectID,
		BucketName: req.BucketName,
		ObjectPath: req.ObjectPath,
		Expiry:     expiry,
		Claims:     claims,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": result.URL})
}
