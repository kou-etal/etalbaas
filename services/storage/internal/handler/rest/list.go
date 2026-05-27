package rest

import (
	"net/http"
	"strconv"

	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

type ListHandler struct {
	svc *service.ObjectService
}

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectID, claims, err := claimsFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	bucket := r.PathValue("bucket")
	if bucket == "" {
		writeError(w, http.StatusBadRequest, "bucket is required")
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

	objects, err := h.svc.ListObjects(r.Context(), service.ListObjectsParams{
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
