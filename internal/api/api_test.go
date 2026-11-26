package api

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type queryFixture struct {
	Name   string  `json:"search_expression"`
	Page   *int    `json:"page_number,omitempty"`
	Max    int     `json:"max_results"`
	Images *bool   `json:"include_food_images,omitempty"`
	Skip   string  `json:"-"`
	Note   *string `json:"note,omitempty"`
}

func TestToQueryMap(t *testing.T) {
	page := 2
	empty := ""
	q := ToQueryMap(queryFixture{Name: "apple", Page: &page, Max: 20, Note: &empty})
	if q["search_expression"] != "apple" || q["page_number"] != "2" || q["max_results"] != "20" {
		t.Fatalf("bad map: %v", q)
	}
	for _, k := range []string{"include_food_images", "note", "Skip"} {
		if _, ok := q[k]; ok {
			t.Fatalf("key %q should be skipped: %v", k, q)
		}
	}
	if got := ToQueryMap("nope"); len(got) != 0 {
		t.Fatalf("non-struct should give empty map: %v", got)
	}
}

func resp(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func TestParseAPIResponse(t *testing.T) {
	data, failed := parseAPIResponse(resp(200, `{"x":1}`))
	if failed {
		t.Fatal("valid JSON flagged failed")
	}
	if data.(map[string]any)["x"] != 1.0 {
		t.Fatalf("bad data: %v", data)
	}
	data, _ = parseAPIResponse(resp(502, "bad gateway"))
	errMap := data.(map[string]any)["error"].(map[string]any)
	if errMap["status"] != 502 {
		t.Fatalf("non-JSON not wrapped: %v", data)
	}
	_, failed = parseAPIResponse(resp(500, "null"))
	if !failed {
		t.Fatal("null body with 500 should fail")
	}
}

func TestEnsureAPI(t *testing.T) {
	s := &Server{}
	if s.HasAPI() {
		t.Fatal("empty server should lack API creds")
	}
	if err := s.EnsureAPI(); err == nil {
		t.Fatal("EnsureAPI should fail without creds")
	}
	if err := s.CompleteAuth(context.Background(), "v"); err == nil {
		t.Fatal("CompleteAuth should fail without pending flow")
	}
	if _, err := s.GetOAuth2Token(context.Background()); err == nil {
		t.Fatal("GetOAuth2Token should fail without creds")
	}
	if err := s.EnsureScope(context.Background(), "premier", "x"); err == nil {
		t.Fatal("EnsureScope should fail without creds")
	}
	s.SetCredentials("id", "secret", "consumer")
	if !s.HasAPI() {
		t.Fatal("server should have API creds after SetCredentials")
	}
}

func TestGetOAuth2TokenCacheHit(t *testing.T) {
	s := &Server{}
	s.SetCredentials("id", "secret", "consumer")
	s.oauth2Access = "cached"
	s.oauth2Expiry = time.Now().Add(time.Hour)
	tok, err := s.GetOAuth2Token(context.Background())
	if err != nil || tok != "cached" {
		t.Fatalf("cache hit failed: %q %v", tok, err)
	}
	if s.HasPremierScope() {
		t.Fatal("empty scope should not claim premier")
	}
	s.oauth2Scope = "basic premier"
	if !s.HasPremierScope() {
		t.Fatal("basic premier should claim premier")
	}
}

func TestMultiUserPendingIsolation(t *testing.T) {
	s := &Server{}
	s.SetCredentials("id", "secret", "consumer")
	alice := WithUser(context.Background(), "alice")
	bob := WithUser(context.Background(), "bob")
	s.SetPending(alice, "tok-a", "sec-a")
	if err := s.CompleteAuth(bob, "v"); err == nil {
		t.Fatal("bob must not see alice's pending flow")
	}
	if s.HasProfileTokens(alice) || s.HasProfileTokens(bob) {
		t.Fatal("no user should have tokens yet")
	}
	// Default user is unaffected by named users.
	if UserID(context.Background()) != "" || UserID(alice) != "alice" {
		t.Fatal("UserID extraction broken")
	}
}
