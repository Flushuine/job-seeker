package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Client struct {
	HTTP     *http.Client
	mu       sync.Mutex
	next     time.Time
	Interval time.Duration
}

func New(timeout time.Duration) *Client {
	return &Client{HTTP: &http.Client{Timeout: timeout}, Interval: 300 * time.Millisecond}
}
func (c *Client) wait(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
func (c *Client) Get(ctx context.Context, url string, out any) error {
	for attempt := 0; attempt < 3; attempt++ {
		c.mu.Lock()
		delay := time.Until(c.next)
		if delay < 0 {
			delay = 0
		}
		c.next = time.Now().Add(delay + c.Interval)
		c.mu.Unlock()
		if e := c.wait(ctx, delay); e != nil {
			return e
		}
		req, e := http.NewRequestWithContext(ctx, "GET", url, nil)
		if e != nil {
			return e
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "AIJobHunter/1.0 (personal job discovery)")
		res, e := c.HTTP.Do(req)
		if e != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if attempt == 2 {
				return e
			}
			if e = c.wait(ctx, time.Second<<attempt); e != nil {
				return e
			}
			continue
		}
		b, e := io.ReadAll(io.LimitReader(res.Body, 16<<20+1))
		res.Body.Close()
		if e != nil {
			return e
		}
		if len(b) > 16<<20 {
			return fmt.Errorf("provider response too large")
		}
		if res.StatusCode == 429 || res.StatusCode >= 500 {
			if attempt == 2 {
				return fmt.Errorf("provider HTTP %d", res.StatusCode)
			}
			delay = time.Second << attempt
			if v, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil && v > 0 {
				delay = time.Duration(v) * time.Second
				if delay > 30*time.Second {
					delay = 30 * time.Second
				}
			}
			if e = c.wait(ctx, delay); e != nil {
				return e
			}
			continue
		}
		if res.StatusCode != 200 {
			return fmt.Errorf("provider HTTP %d", res.StatusCode)
		}
		if e = json.Unmarshal(b, out); e != nil {
			return fmt.Errorf("invalid provider JSON: %w", e)
		}
		return nil
	}
	return fmt.Errorf("provider retries exhausted")
}
