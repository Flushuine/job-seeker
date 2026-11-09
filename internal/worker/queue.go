package worker

import (
	"context"
	"fmt"
	"github.com/aqil/ai-job-hunter/internal/repository/postgres"
	"github.com/aqil/ai-job-hunter/internal/usecase"
	"log/slog"
	"time"
)

// A process-wide database lock guarantees a single inference consumer, including
// when the optional standalone worker and the API run simultaneously.
func Queue(ctx context.Context, store *postgres.Store, s *usecase.Service, log *slog.Logger) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		e := store.WithLock(ctx, 735017, func() error {
			if e := store.RecoverTasks(ctx); e != nil {
				return e
			}
			for {
				task, e := store.NextTask(ctx)
				if e != nil {
					return e
				}
				if task == nil {
					return nil
				}
				taskCtx, cancel := context.WithTimeout(ctx, 200*time.Second)
				switch task.Kind {
				case "job":
					_, e = s.Analyse(taskCtx, task.ResourceID)
				case "company":
					e = s.InterpretCompany(taskCtx, task.ResourceID)
				case "cv":
					e = s.AnalyseCV(taskCtx, task.ResourceID)
				default:
					e = fmt.Errorf("unknown analysis task kind: %s", task.Kind)
				}
				cancel()
				if e != nil {
					log.Warn("local analysis failed", "task", task.ID, "error", e)
				}
				finishCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				saveErr := store.FinishTask(finishCtx, task.ID, e)
				stop()
				if saveErr != nil {
					return saveErr
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
			}
		})
		if e != nil && e != postgres.ErrBusy && ctx.Err() == nil {
			log.Error("analysis queue", "error", e)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
