package rest

import (
	"net/http"

	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

type DeleteHandler struct {
	svc *service.ObjectService
}

func (h *DeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	if err := h.svc.DeleteObject(r.Context(), service.DeleteObjectParams{
		ProjectID:  projectID,
		BucketName: bucket,
		ObjectPath: objectPath,
		Claims:     claims,
	}); err != nil {
		writeAppError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
