package main

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/bootstrap"
	"log/slog"
	"os"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	_, s, _, e := bootstrap.Start(context.Background(), log)
	if e != nil {
		log.Error("migration failed", "error", e)
		os.Exit(1)
	}
	defer s.Close()
	log.Info("migrations applied")
}
