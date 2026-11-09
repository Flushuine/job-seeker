package main

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/bootstrap"
	"github.com/aqil/ai-job-hunter/internal/worker"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	_, store, s, e := bootstrap.Start(ctx, log)
	if e != nil {
		log.Error("worker startup", "error", e)
		os.Exit(1)
	}
	defer store.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); worker.Queue(ctx, store, s, log) }()
	worker.Run(ctx, s, log)
	wg.Wait()
}
