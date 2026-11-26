package httpx

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	httpTimeout  = 15 * time.Second
	httpMaxAttem = 3
)

var httpRetryDelays = []time.Duration{500 * time.Millisecond, 2000 * time.Millisecond}

var httpClient = &http.Client{Timeout: httpTimeout}

// resilientFetch: timeout on every attempt, retries for safe (GET) requests only.
// Writes are never retried: a timed-out write may have landed server-side.
func ResilientFetch(req *http.Request) (*http.Response, error) {
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}
	target := method + " " + req.URL.String()
	for attempt := 1; ; attempt++ {
		resp, err := httpClient.Do(cloneRequest(req))
		if err == nil {
			if method == http.MethodGet && resp.StatusCode >= 500 && attempt < httpMaxAttem {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				time.Sleep(delayFor(attempt))
				continue
			}
			return resp, nil
		}
		timedOut := isTimeout(err)
		if method == http.MethodGet && attempt < httpMaxAttem && (timedOut || isTransient(err)) {
			time.Sleep(delayFor(attempt))
			continue
		}
		if timedOut {
			return nil, fmt.Errorf("FatSecret API request timed out after %s (%s, attempt %d/%d)", httpTimeout, target, attempt, httpMaxAttem)
		}
		return nil, err
	}
}

func delayFor(attempt int) time.Duration {
	i := attempt - 1
	if i >= len(httpRetryDelays) {
		i = len(httpRetryDelays) - 1
	}
	return httpRetryDelays[i]
}

func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// Stalls and refused connections surface as net.Errors (timeouts included)
// — safe to retry for GET.
func isTransient(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	var dns *net.DNSError
	return errors.As(err, &dns)
}

func cloneRequest(r *http.Request) *http.Request {
	c := r.Clone(r.Context())
	if r.GetBody != nil {
		if body, err := r.GetBody(); err == nil {
			c.Body = body
		}
	}
	return c
}
