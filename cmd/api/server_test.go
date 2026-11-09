package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestOccupiedPortDoesNotStartWorkersOrLogListening(t *testing.T) {
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer listener.Close()
	var logs bytes.Buffer
	started := false
	e = serveAPI(context.Background(), &http.Server{Addr: listener.Addr().String()}, slog.New(slog.NewTextHandler(&logs, nil)), func(context.Context) func() { started = true; return func() {} })
	if !errors.Is(e, syscall.EADDRINUSE) {
		t.Fatalf("expected address-in-use error, got %v", e)
	}
	if started {
		t.Fatal("background workers started before the port was bound")
	}
	if strings.Contains(logs.String(), "API listening") {
		t.Fatal("reported listening after a bind failure")
	}
}
func TestShutdownCancelsAndWaitsForWorkers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stopped := make(chan struct{})
	waited := false
	e := serveAPI(ctx, &http.Server{Addr: "127.0.0.1:0", Handler: http.NotFoundHandler()}, slog.Default(), func(workerCtx context.Context) func() {
		go func() { <-workerCtx.Done(); close(stopped) }()
		cancel()
		return func() { <-stopped; waited = true }
	})
	if e != nil {
		t.Fatal(e)
	}
	if !waited {
		t.Fatal("server returned without waiting for worker shutdown")
	}
}
