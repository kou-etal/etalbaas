package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kou-etal/etalbaas/pkg/auth"
	"github.com/kou-etal/etalbaas/pkg/metadb"
	"github.com/kou-etal/etalbaas/pkg/observability"
	"github.com/kou-etal/etalbaas/pkg/server"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/storage/v1/storagev1connect"
	"github.com/kou-etal/etalbaas/services/storage/internal"
	"github.com/kou-etal/etalbaas/services/storage/internal/handler"
	resthandler "github.com/kou-etal/etalbaas/services/storage/internal/handler/rest"
	"github.com/kou-etal/etalbaas/services/storage/internal/metastore"
	"github.com/kou-etal/etalbaas/services/storage/internal/pool"
	"github.com/kou-etal/etalbaas/services/storage/internal/service"
	"github.com/kou-etal/etalbaas/storage-providers/s3compat"
)

func main() {
	ctx := context.Background()

	cfg, err := internal.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	shutdown, err := observability.Init(ctx, observability.Config{
		ServiceName:    "storage",
		ServiceVersion: "0.1.0",
		OTELEndpoint:   cfg.OTELEndpoint,
		Environment:    cfg.Environment,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			slog.Error("observability shutdown failed", "error", err)
		}
	}()

	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Meta DB pool (for project ownership + API key validation).
	metaPool, err := metadb.NewPool(initCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer metaPool.Close()

	// K8s client (in-cluster, for reading tenant DB secrets).
	k8sCfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal("k8s in-cluster config: ", err)
	}
	k8sClient, err := kubernetes.NewForConfig(k8sCfg)
	if err != nil {
		log.Fatal("k8s client: ", err)
	}

	// S3-compatible storage provider.
	s3Provider, err := s3compat.New(s3compat.Config{
		Endpoint:       cfg.S3Endpoint,
		BucketName:     cfg.S3Bucket,
		AccessKeyID:    cfg.S3AccessKey,
		SecretAccessKey: cfg.S3SecretKey,
		Region:         cfg.S3Region,
		UseSSL:         cfg.S3UseSSL,
	})
	if err != nil {
		log.Fatal("s3 provider: ", err)
	}
	if err := s3Provider.EnsureBucket(initCtx); err != nil {
		log.Fatal("ensure s3 bucket: ", err)
	}

	// Pool Manager (per-project tenant DB connections).
	poolMgr := pool.NewManager(k8sClient)
	defer poolMgr.Close()

	// Services.
	metaQ := metastore.New(metaPool)
	bucketSvc := service.NewBucketService(metaQ, poolMgr)
	objectSvc := service.NewObjectService(poolMgr, s3Provider)

	// gRPC handler (bucket CRUD).
	bucketHandler := handler.NewBucketHandler(bucketSvc)
	interceptors := server.DefaultInterceptors(slog.Default())
	grpcPath, grpcHnd := storagev1connect.NewStorageServiceHandler(bucketHandler, interceptors)

	// REST handler (file operations) with API key middleware.
	apiKeyValidator := auth.NewAPIKeyValidator(metaPool)
	restRouter := resthandler.NewRouter(objectSvc)
	restHnd := apiKeyValidator.Middleware(restRouter)

	// Public routes (no auth required) — must be registered separately.
	publicRouter := resthandler.NewPublicRouter(objectSvc)

	srv := server.New(
		server.Config{Port: cfg.Port, MetricsPort: cfg.MetricsPort},
		server.Handler{Pattern: grpcPath, Handler: grpcHnd},
		server.Handler{Pattern: "/storage/v1/public/", Handler: publicRouter},
		server.Handler{Pattern: "/storage/", Handler: http.StripPrefix("", restHnd)},
	)
	srv.RegisterHealthChecker(func(ctx context.Context) error {
		return metadb.HealthCheck(ctx, metaPool)
	})
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
