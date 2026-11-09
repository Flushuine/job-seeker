package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryTransient(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(503)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer s.Close()
	c := New(time.Second)
	c.Interval = 0
	var v map[string]bool
	if e := c.Get(context.Background(), s.URL, &v); e != nil || !v["ok"] || calls.Load() != 2 {
		t.Fatal(v, e, calls.Load())
	}
}
func TestPermanentFailure(t *testing.T) {
	var calls int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(404) }))
	defer s.Close()
	c := New(time.Second)
	var v any
	if e := c.Get(context.Background(), s.URL, &v); e == nil || calls != 1 {
		t.Fatal("permanent failure should not retry", e, calls)
	}
}
