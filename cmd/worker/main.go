package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/testsabirweb/plateful/internal/config"
	"github.com/testsabirweb/plateful/internal/queue"
)

func main() {
	cfg := config.Load()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if !cfg.SQSEnabled() {
		slog.Error("SQS_ENDPOINT is required for the worker")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client, err := queue.NewSQS(ctx, queue.SQSConfig{
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

	var processed atomic.Uint64
	slog.Info("worker consuming", "queue_url", client.QueueURL())

	err = client.ReceiveLoop(ctx, func(ctx context.Context, e queue.Event) error {
		if e.Type != queue.EventTypeOrderCreated {
			slog.Warn("unknown event type", "type", e.Type)
			return nil
		}
		n := processed.Add(1)
		slog.Info("simulated notification", "order_id", e.OrderID, "events_processed", n)
		return nil
	})
	if err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("worker stopped", "err", err)
		os.Exit(1)
	}
	slog.Info("worker shutdown")
}
