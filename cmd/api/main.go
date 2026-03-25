package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/time/rate"
	"github.com/testsabirweb/plateful/internal/config"
	"github.com/testsabirweb/plateful/internal/graph"
	"github.com/testsabirweb/plateful/internal/observability"
	"github.com/testsabirweb/plateful/internal/queue"
	"github.com/testsabirweb/plateful/internal/store"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	var pub queue.Publisher = queue.NoOpPublisher{}
	if cfg.SQSEnabled() {
		sqsClient, err := queue.NewSQS(ctx, queue.SQSConfig{
			Endpoint:  cfg.SQSEndpoint,
			Region:    cfg.AWSRegion,
			QueueName: cfg.SQSQueueName,
			AccessKey: cfg.AWSAccessKeyID,
			SecretKey: cfg.AWSSecretAccessKey,
		})
		if err != nil {
			slog.Error("sqs client", "err", err)
			os.Exit(1)
		}
		pub = sqsClient
		slog.Info("sqs publisher ready", "queue_url", sqsClient.QueueURL())
	}

	st := store.New(pool)
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			Store: st,
			Queue: pub,
		},
	}))

	log := slog.Default()
	mux := http.NewServeMux()
	mux.Handle("/metrics", observability.MetricsHandler())
	mux.Handle("/", observability.HTTPMiddleware(log, playground.Handler("Plateful API", "/query")))
	mux.Handle("/query", observability.HTTPMiddleware(log, graph.DataloaderMiddleware(st, srv)))

	// 10 requests/sec per IP, burst of 30.
	rl := observability.NewRateLimiter(rate.Limit(10), 30)
	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: rl.Middleware(mux)}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down api server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			slog.Error("http shutdown", "err", err)
		}
	}()

	slog.Info("api listening", "addr", cfg.HTTPAddr, "playground", "/", "graphql", "/query", "metrics", "/metrics", "sqs", cfg.SQSEnabled())
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http server", "err", err)
		os.Exit(1)
	}
}
