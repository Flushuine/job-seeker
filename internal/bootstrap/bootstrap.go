package bootstrap

import (
	"context"
	"github.com/aqil/ai-job-hunter/internal/config"
	"github.com/aqil/ai-job-hunter/internal/domain"
	"github.com/aqil/ai-job-hunter/internal/provider/greenhouse"
	"github.com/aqil/ai-job-hunter/internal/provider/himalayas"
	"github.com/aqil/ai-job-hunter/internal/provider/httpclient"
	"github.com/aqil/ai-job-hunter/internal/provider/lever"
	"github.com/aqil/ai-job-hunter/internal/provider/ollama"
	"github.com/aqil/ai-job-hunter/internal/provider/research"
	"github.com/aqil/ai-job-hunter/internal/repository/postgres"
	"github.com/aqil/ai-job-hunter/internal/usecase"
	"log/slog"
	"net/http"
	"time"
)

func Start(ctx context.Context, log *slog.Logger) (config.Config, *postgres.Store, *usecase.Service, error) {
	c, e := config.Load()
	if e != nil {
		return c, nil, nil, e
	}
	store, e := postgres.Open(ctx, c.DatabaseURL)
	if e != nil {
		return c, nil, nil, e
	}
	if e = store.Migrate(ctx); e != nil {
		store.Close()
		return c, nil, nil, e
	}
	client := httpclient.New(c.HTTPTimeout)
	lev := &lever.Provider{Client: httpclient.New(c.HTTPTimeout)}
	broader := httpclient.New(c.HTTPTimeout)
	broader.Interval = time.Second
	s := &usecase.Service{Repo: store, Providers: map[string]domain.JobProvider{"greenhouse": &greenhouse.Provider{Client: client}, "lever": lev, "lever-eu": lev, "himalayas": &himalayas.Provider{Client: broader}}, AI: &ollama.Provider{Client: localClient(), BaseURL: c.OllamaURL, Model: c.Model}, Researcher: research.New(c.ResearchHosts), AIEnabled: c.AIEnabled, Log: log}
	return c, store, s, nil
}

func localClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.Proxy = nil
	return &http.Client{Timeout: 180 * time.Second, Transport: tr, CheckRedirect: func(r *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
}
