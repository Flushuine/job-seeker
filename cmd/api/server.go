package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Bind before starting workers so a second API instance exits without launching
// discovery or inference, or reporting a server that never actually listened.
func serveAPI(ctx context.Context, server *http.Server, log *slog.Logger, startWorkers func(context.Context) func()) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("cannot start API on %s: %w", server.Addr, err)
	}
	defer listener.Close()
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	server.BaseContext = func(net.Listener) context.Context { return workerCtx }
	waitWorkers := startWorkers(workerCtx)
	defer func() { cancelWorkers(); waitWorkers() }()
	done := make(chan error, 1)
	log.Info("API listening", "address", listener.Addr().String())
	go func() { done <- server.Serve(listener) }()
	select {
	case err = <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
	return nil
}
