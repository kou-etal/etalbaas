package rest

import (
	"net/http"

	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

// NewRouter creates a new HTTP mux for the authenticated Storage REST API.
// All routes are prefixed with /storage/v1/.
// The apiKeyMiddleware is expected to be applied externally by the caller.
func NewRouter(objSvc *service.ObjectService) http.Handler {
	mux := http.NewServeMux()

	upload := &UploadHandler{svc: objSvc}
	download := &DownloadHandler{svc: objSvc}
	del := &DeleteHandler{svc: objSvc}
	list := &ListHandler{svc: objSvc}
	presign := &PresignHandler{svc: objSvc}
	complete := &UploadCompleteHandler{svc: objSvc}

	mux.HandleFunc("POST /storage/v1/object/{bucket}/{path...}", upload.ServeHTTP)
	mux.HandleFunc("GET /storage/v1/object/{bucket}/{path...}", download.ServeHTTP)
	mux.HandleFunc("DELETE /storage/v1/object/{bucket}/{path...}", del.ServeHTTP)
	mux.HandleFunc("GET /storage/v1/object/list/{bucket}", list.ServeHTTP)
	mux.HandleFunc("POST /storage/v1/sign/upload", presign.Upload)
	mux.HandleFunc("POST /storage/v1/sign/download", presign.Download)
	mux.HandleFunc("POST /storage/v1/upload-complete", complete.ServeHTTP)

	return mux
}

// NewPublicRouter creates a new HTTP mux for unauthenticated public routes.
// These routes must NOT be wrapped by API key middleware.
func NewPublicRouter(objSvc *service.ObjectService) http.Handler {
	mux := http.NewServeMux()
	pub := &PublicHandler{svc: objSvc}
	mux.HandleFunc("GET /storage/v1/public/{bucket}/{path...}", pub.ServeHTTP)
	return mux
}
