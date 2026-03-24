package main

import (
	"log/slog"
	"os"

	"github.com/testsabirweb/plateful/internal/config"
)

func main() {
	_ = config.Load()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))
	slog.Info("worker starting")
}
