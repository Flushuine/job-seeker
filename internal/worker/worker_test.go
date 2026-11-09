package worker

import (
	"bytes"
	"context"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/usecase"
	"log/slog"
	"testing"
)

type cancelHistoryRepo struct {
	usecase.Repository
	cancel context.CancelFunc
}

func (r cancelHistoryRepo) Runs(context.Context) ([]domain.DiscoveryRun, error) {
	r.cancel()
	return nil, context.Canceled
}
func TestSchedulerCancellationIsQuiet(t *testing.T) {
	for _, duringRead := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		var logs bytes.Buffer
		s := &usecase.Service{}
		if duringRead {
			s.Repo = cancelHistoryRepo{cancel: cancel}
		} else {
			cancel()
		}
		Run(ctx, s, slog.New(slog.NewTextHandler(&logs, nil)))
		cancel()
		if logs.Len() > 0 {
			t.Fatalf("normal cancellation logged as a scheduler error: %s", logs.String())
		}
	}
}
