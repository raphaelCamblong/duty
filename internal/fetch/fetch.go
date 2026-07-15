// Package fetch is duty's network port: a Fetcher that GETs a URL, plus an
// HTTP adapter over the real network with a short timeout. Any failure is the
// caller's cue to fall back to an embedded copy, so callers never surface it
// and tests run offline against a fake.
package fetch

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

// Fetcher retrieves the bytes at a URL. duty's one network touch goes through
// it, so callers stay testable without dialing.
type Fetcher interface {
	// Fetch GETs url and returns its body, or an error on any failure.
	Fetch(url string) ([]byte, error)
}

// DefaultTimeout bounds a real fetch when HTTP.Timeout is zero, so a slow
// network never delays the embedded fallback for long.
const DefaultTimeout = 2 * time.Second

// HTTP is a Fetcher over the real network. A zero HTTP uses DefaultTimeout.
type HTTP struct {
	// Timeout bounds the whole request; zero means DefaultTimeout.
	Timeout time.Duration
}

// Fetch GETs url with HTTP's timeout and returns its body; a non-200 status or
// any transport error is returned as an error.
func (h HTTP) Fetch(url string) ([]byte, error) {
	timeout := h.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
