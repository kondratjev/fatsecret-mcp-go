package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubToken serves OAuth2 token responses per scope: "ok" grants, anything
// else is invalid_scope, except scopes containing "boom" which 500.
func stubToken(t *testing.T, granted map[string]bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		scope := r.Form.Get("scope")
		if strings.Contains(scope, "boom") {
			w.WriteHeader(500)
			return
		}
		ok := true
		for _, s := range strings.Fields(scope) {
			if s != "basic" && !granted[s] {
				ok = false
			}
		}
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"invalid_scope"}`))
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"tok","expires_in":86400}`))
	}))
	old := OAuth2TokenURL
	OAuth2TokenURL = srv.URL
	t.Cleanup(func() { OAuth2TokenURL = old })
	return srv
}

func TestFetchOAuth2Token(t *testing.T) {
	stubToken(t, map[string]bool{"premier": true})
	tok, err := FetchOAuth2Token("id", "secret", []string{"basic nothing", "basic premier"})
	if err != nil || tok.AccessToken != "tok" || tok.Scope != "basic premier" {
		t.Fatalf("should skip invalid scope: %+v %v", tok, err)
	}
	stubToken(t, map[string]bool{})
	if _, err := FetchOAuth2Token("id", "secret", []string{"basic nothing"}); err == nil {
		t.Fatal("all-invalid should error")
	}
	stubToken(t, map[string]bool{"premier": true})
	if _, err := FetchOAuth2Token("id", "secret", []string{"basic boom"}); err == nil {
		t.Fatal("500 should error, not skip")
	}
}

func TestDetectOAuth2Token(t *testing.T) {
	stubToken(t, map[string]bool{"premier": true, "barcode": true, "localization": true, "nlp": true, "image-recognition": true, "feedback": true})
	tok, err := DetectOAuth2Token("id", "secret")
	if err != nil || !HasPremierScope(tok.Scope) {
		t.Fatalf("full grant should keep all scopes: %+v %v", tok, err)
	}
	stubToken(t, map[string]bool{"premier": true})
	tok, err = DetectOAuth2Token("id", "secret")
	if err != nil || tok.Scope != "basic premier" {
		t.Fatalf("should probe down to granted scopes: %+v %v", tok, err)
	}
}

func TestHasScope(t *testing.T) {
	if !HasScope("basic premier", "premier") || HasScope("basic premierx", "premier") {
		t.Fatal("scope matching wrong")
	}
	if !HasPremierScope("basic premier") || HasPremierScope("basic") {
		t.Fatal("premier matching wrong")
	}
	if !isInvalidScopeError(400, `{"error":"Invalid_Scope"}`) || isInvalidScopeError(401, "invalid_scope") {
		t.Fatal("invalid_scope detection wrong")
	}
}
