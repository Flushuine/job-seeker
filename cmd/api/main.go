package main

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/bootstrap"
	transport "github.com/aqil/ai-job-hunter/internal/transport/http"
	"github.com/aqil/ai-job-hunter/internal/worker"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if e := run(log); e != nil {
		log.Error("API stopped", "error", e)
		os.Exit(1)
	}
}
func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c, store, s, e := bootstrap.Start(ctx, log)
	if e != nil {
		return e
	}
	defer store.Close()
	server := &http.Server{Addr: c.Address, Handler: transport.New(store, s, c, log), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	return serveAPI(ctx, server, log, func(workerCtx context.Context) func() {
		var workers sync.WaitGroup
		workers.Add(2)
		go func() { defer workers.Done(); worker.Queue(workerCtx, store, s, log) }()
		go func() { defer workers.Done(); worker.Run(workerCtx, s, log) }()
		return workers.Wait
	})
}
