package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kou-etal/etalbaas/pkg/auth"
	"github.com/kou-etal/etalbaas/pkg/metadb"
	"github.com/kou-etal/etalbaas/pkg/observability"
	"github.com/kou-etal/etalbaas/pkg/server"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/storage/v1/storagev1connect"
	"github.com/kou-etal/etalbaas/services/storage/internal"
	"github.com/kou-etal/etalbaas/services/storage/internal/event"
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

	obs, err := observability.Init(ctx, observability.Config{
		ServiceName:    "storage",
		ServiceVersion: "0.1.0",
		OTELEndpoint:   cfg.OTELEndpoint,
		Environment:    cfg.Environment,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := obs.Shutdown(ctx); err != nil {
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

	// Pool provider (per-project tenant DB connections).
	var poolProvider pool.Provider
	if cfg.K8sEnabled {
		k8sCfg, err := rest.InClusterConfig()
		if err != nil {
			log.Fatal("k8s in-cluster config: ", err)
		}
		k8sClient, err := kubernetes.NewForConfig(k8sCfg)
		if err != nil {
			log.Fatal("k8s client: ", err)
		}
		poolProvider = pool.NewManager(k8sClient)
	} else {
		poolProvider = pool.NewStaticProvider(metaPool)
		slog.Warn("K8S_ENABLED=false: using static DB pool (no per-project routing)")
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

	defer poolProvider.Close()

	// NATS JetStream (optional: event publishing for storage triggers).
	var pub event.Publisher
	if cfg.NATSURL != "" {
		nc, err := nats.Connect(cfg.NATSURL,
			nats.Name("storage-ms"),
			nats.MaxReconnects(-1),
			nats.ReconnectWait(2*time.Second),
		)
		if err != nil {
			log.Fatal("nats connect: ", err)
		}
		defer nc.Close()

		js, err := nc.JetStream()
		if err != nil {
			log.Fatal("nats jetstream: ", err)
		}
		pub = event.NewNATSPublisher(js, cfg.NATSSubjectPrefix)
		slog.Info("NATS event publishing enabled", "url", cfg.NATSURL, "prefix", cfg.NATSSubjectPrefix)
	} else {
		slog.Info("NATS_URL not set, storage event publishing disabled")
	}

	// Services.
	metaQ := metastore.New(metaPool)
	bucketSvc := service.NewBucketService(metaQ, poolProvider)
	objectSvc := service.NewObjectService(poolProvider, s3Provider, pub)

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

	// Dashboard routes (JWT auth via gateway, no API key middleware).
	dashboardHandler := resthandler.NewDashboardHandler(objectSvc, metaQ)
	dashboardRouter := resthandler.NewDashboardRouter(dashboardHandler)

	srv := server.New(
		server.Config{Port: cfg.Port, MetricsPort: cfg.MetricsPort},
		server.Handler{Pattern: grpcPath, Handler: grpcHnd},
		server.Handler{Pattern: "/storage/v1/dashboard/", Handler: dashboardRouter},
		server.Handler{Pattern: "/storage/v1/public/", Handler: publicRouter},
		server.Handler{Pattern: "/storage/", Handler: http.StripPrefix("", restHnd)},
	)
	srv.SetMetricsHandler(obs.MetricsHandler)
	srv.RegisterHealthChecker(func(ctx context.Context) error {
		return metadb.HealthCheck(ctx, metaPool)
	})
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
