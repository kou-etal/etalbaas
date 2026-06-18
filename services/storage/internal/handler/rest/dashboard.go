package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kou-etal/etalbaas/services/storage/internal/metastore"
	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

// DashboardHandler provides REST endpoints for dashboard file operations.
// Unlike the SDK/end-user endpoints (API key auth), these use JWT auth
// via the gateway's SecurityPolicy (X-User-ID header from JWT sub claim).
type DashboardHandler struct {
	objSvc *service.ObjectService
	meta   metastore.Querier
}

func NewDashboardHandler(objSvc *service.ObjectService, meta metastore.Querier) *DashboardHandler {
	return &DashboardHandler{objSvc: objSvc, meta: meta}
}

// NewDashboardRouter creates routes for dashboard-authenticated file operations.
// These routes should NOT be wrapped by API key middleware.
func NewDashboardRouter(h *DashboardHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /storage/v1/dashboard/objects", h.ListObjects)
	mux.HandleFunc("POST /storage/v1/dashboard/upload", h.Upload)
	mux.HandleFunc("GET /storage/v1/dashboard/download", h.Download)
	mux.HandleFunc("DELETE /storage/v1/dashboard/delete", h.Delete)
	return mux
}

// dashboardAuth extracts user identity from gateway-set headers and verifies
// that the user owns the specified project. Returns projectID and service_role claims.
func (h *DashboardHandler) dashboardAuth(r *http.Request) (projectID string, claims json.RawMessage, err error) {
	userIDStr := r.Header.Get("X-User-ID")
	if userIDStr == "" {
		return "", nil, fmt.Errorf("missing X-User-ID header")
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return "", nil, fmt.Errorf("invalid X-User-ID format")
	}

	projectID = r.URL.Query().Get("project_id")
	if projectID == "" {
		return "", nil, fmt.Errorf("project_id query parameter is required")
	}

	// Verify project belongs to user/tenant.
	if _, err := h.meta.GetProjectByIDAndTenantID(r.Context(), metastore.GetProjectByIDAndTenantIDParams{
		ID:       projectID,
		TenantID: userID,
	}); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil, fmt.Errorf("project not found or not owned by user")
		}
		return "", nil, fmt.Errorf("verify project ownership: %w", err)
	}

	// Dashboard operates with service_role privileges.
	claims = json.RawMessage(fmt.Sprintf(`{"role":"service_role","sub":"%s"}`, userID.String()))
	return projectID, claims, nil
}

func (h *DashboardHandler) ListObjects(w http.ResponseWriter, r *http.Request) {
	projectID, claims, err := h.dashboardAuth(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	bucket := r.URL.Query().Get("bucket")
	if bucket == "" {
		writeError(w, http.StatusBadRequest, "bucket query parameter is required")
		return
	}

	limit := int32(100)
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = int32(n)
	}

	var offset int32
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = int32(n)
	}

	objects, err := h.objSvc.ListObjects(r.Context(), service.ListObjectsParams{
		ProjectID:  projectID,
		BucketName: bucket,
		Limit:      limit,
		Offset:     offset,
		Claims:     claims,
	})
	if err != nil {
		writeAppError(w, err)
		return
	}

	type objectItem struct {
		ID        string  `json:"id"`
		BucketID  string  `json:"bucket_id"`
		Name      string  `json:"name"`
		Size      *int64  `json:"size"`
		MimeType  *string `json:"mime_type"`
		ETag      *string `json:"etag"`
		CreatedAt string  `json:"created_at"`
		UpdatedAt string  `json:"updated_at"`
	}

	items := make([]objectItem, len(objects))
	for i, obj := range objects {
		items[i] = objectItem{
			ID:        obj.ID.String(),
			BucketID:  obj.BucketID,
			Name:      obj.Name,
			Size:      obj.Size,
			MimeType:  obj.MimeType,
			ETag:      obj.Etag,
			CreatedAt: obj.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: obj.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *DashboardHandler) Upload(w http.ResponseWriter, r *http.Request) {
	projectID, claims, err := h.dashboardAuth(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	bucket := r.URL.Query().Get("bucket")
	objectPath := r.URL.Query().Get("path")
	if bucket == "" || objectPath == "" {
		writeError(w, http.StatusBadRequest, "bucket and path query parameters are required")
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	result, err := h.objSvc.Upload(r.Context(), service.UploadParams{
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
		"id":         result.Object.ID.String(),
		"bucket_id":  result.Object.BucketID,
		"name":       result.Object.Name,
		"size":       result.Object.Size,
		"mime_type":  result.Object.MimeType,
		"etag":       result.Object.Etag,
		"key":        result.Key,
		"created_at": result.Object.CreatedAt,
		"updated_at": result.Object.UpdatedAt,
	})
}

func (h *DashboardHandler) Download(w http.ResponseWriter, r *http.Request) {
	projectID, claims, err := h.dashboardAuth(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	bucket := r.URL.Query().Get("bucket")
	objectPath := r.URL.Query().Get("path")
	if bucket == "" || objectPath == "" {
		writeError(w, http.StatusBadRequest, "bucket and path query parameters are required")
		return
	}

	result, err := h.objSvc.Download(r.Context(), service.DownloadParams{
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
	filename := sanitizeFilename(path.Base(objectPath))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filename))

	w.WriteHeader(http.StatusOK)
	io.Copy(w, result.Body) //nolint:errcheck
}

func (h *DashboardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	projectID, claims, err := h.dashboardAuth(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	bucket := r.URL.Query().Get("bucket")
	objectPath := r.URL.Query().Get("path")
	if bucket == "" || objectPath == "" {
		writeError(w, http.StatusBadRequest, "bucket and path query parameters are required")
		return
	}

	if err := h.objSvc.DeleteObject(r.Context(), service.DeleteObjectParams{
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
