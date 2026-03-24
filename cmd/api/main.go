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
	"github.com/testsabirweb/plateful/internal/store"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{Store: store.New(pool)},
	}))

	http.Handle("/", playground.Handler("Plateful API", "/query"))
	http.Handle("/query", srv)

	slog.Info("api listening", "addr", cfg.HTTPAddr, "playground", "/", "graphql", "/query")
	if err := http.ListenAndServe(cfg.HTTPAddr, nil); err != nil {
		slog.Error("http server", "err", err)
		os.Exit(1)
	}
}
