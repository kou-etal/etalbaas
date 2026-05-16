package rest

import (
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

type PublicHandler struct {
	svc *service.ObjectService
}

// ServeHTTP handles public (no auth) downloads.
// The projectID must be provided via query parameter since there's no API key.
func (h *PublicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id query parameter is required")
		return
	}

	bucket := r.PathValue("bucket")
	objectPath := r.PathValue("path")
	if bucket == "" || objectPath == "" {
		writeError(w, http.StatusBadRequest, "bucket and path are required")
		return
	}

	result, err := h.svc.PublicDownload(r.Context(), projectID, bucket, objectPath)
	if err != nil {
		writeAppError(w, err)
		return
	}
	defer result.Body.Close()

	w.Header().Set("Content-Type", result.ContentType)
	if result.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", result.Size))
	}
	if result.ETag != "" {
		w.Header().Set("ETag", `"`+result.ETag+`"`)
	}
	filename := path.Base(objectPath)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "public, max-age=3600")

	w.WriteHeader(http.StatusOK)
	io.Copy(w, result.Body) //nolint:errcheck
}
