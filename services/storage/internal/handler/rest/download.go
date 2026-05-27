package rest

import (
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

type DownloadHandler struct {
	svc *service.ObjectService
}

func (h *DownloadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	result, err := h.svc.Download(r.Context(), service.DownloadParams{
		ProjectID:  projectID,
		BucketName: bucket,
		ObjectPath: objectPath,
		Claims:     claims,
	})
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

	w.WriteHeader(http.StatusOK)
	io.Copy(w, result.Body) //nolint:errcheck
}
