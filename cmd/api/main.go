package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/testsabirweb/plateful/internal/config"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	slog.Info("api starting", "addr", cfg.HTTPAddr)
	fmt.Println("plateful api — next: GraphQL + DB")
}
