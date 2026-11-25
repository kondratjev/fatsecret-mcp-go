package httpx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestResilientFetch(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if r.URL.Path == "/flaky" && n < 2 {
			w.WriteHeader(500)
			return
		}
		if r.URL.Path == "/down" {
			w.WriteHeader(500)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	get := func(path string) *http.Request {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+path, nil)
		return req
	}
	resp, err := ResilientFetch(get("/flaky"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "ok" || hits.Load() != 2 {
		t.Fatalf("GET should retry 500: hits=%d body=%q", hits.Load(), body)
	}
	before := hits.Load()
	resp, err = ResilientFetch(get("/down"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 500 || hits.Load() != before+3 {
		t.Fatalf("GET should try 3 times: hits=%d", hits.Load()-before)
	}
	before = hits.Load()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/down", nil)
	resp, err = ResilientFetch(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if hits.Load() != before+1 {
		t.Fatal("POST must never retry")
	}
	if d := delayFor(99); d != 2*time.Second {
		t.Fatalf("delays should cap at last value: %v", d)
	}
}
