package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/jackc/pgx/v5/pgxpool"
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

	ctx := context.Background()

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

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			Store: store.New(pool),
			Queue: pub,
		},
	}))

	log := slog.Default()
	mux := http.NewServeMux()
	mux.Handle("/metrics", observability.MetricsHandler())
	mux.Handle("/", observability.HTTPMiddleware(log, playground.Handler("Plateful API", "/query")))
	mux.Handle("/query", observability.HTTPMiddleware(log, srv))

	slog.Info("api listening", "addr", cfg.HTTPAddr, "playground", "/", "graphql", "/query", "metrics", "/metrics", "sqs", cfg.SQSEnabled())
	if err := http.ListenAndServe(cfg.HTTPAddr, mux); err != nil {
		slog.Error("http server", "err", err)
		os.Exit(1)
	}
}
