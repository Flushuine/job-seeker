package worker

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/usecase"
	"log/slog"
	"time"
)

func Run(ctx context.Context, s *usecase.Service, log *slog.Logger) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	last := time.Time{}
	for ctx.Err() == nil {
		runs, e := s.Repo.Runs(ctx)
		if ctx.Err() != nil {
			return
		}
		if e != nil {
			log.Error("scheduler history", "error", e)
		} else {
			for _, r := range runs {
				if r.FinishedAt != nil && r.CreatedAt.After(last) {
					last = r.CreatedAt
					break
				}
			}
		}
		pref, e := s.Repo.Preferences(ctx)
		if ctx.Err() != nil {
			return
		}
		if e != nil {
			log.Error("scheduler preferences", "error", e)
		} else if time.Since(last) >= time.Duration(pref.DiscoveryMinutes)*time.Minute {
			runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
			r, e := s.Discover(runCtx)
			if ctx.Err() != nil {
				cancel()
				return
			}
			if e != nil {
				log.Warn("scheduled discovery", "error", e)
			} else {
				log.Info("discovery complete", "status", r.Status, "seen", r.JobsSeen)
				s.Automate(runCtx)
			}
			cancel()
			last = time.Now()
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
