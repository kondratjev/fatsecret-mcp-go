package auth

import (
	"encoding/json"
	"fmt"
	"github.com/kondratjev/fatsecret-mcp-go/internal/httpx"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const OAuth2TokenURL = "https://oauth.fatsecret.com/connect/token"

// OptionalScopes a key may hold on top of "basic". localization and feedback
// localization and feedback are requested on top of premier/barcode/nlp/image-recognition.
var optionalScopes = []string{"premier", "barcode", "localization", "nlp", "image-recognition", "feedback"}

type OAuth2Token struct {
	AccessToken string
	ExpiresIn   int64
	Scope       string
}

// isInvalidScopeError: RFC 6749 §5.2 — a scope the client isn't entitled to is 400 invalid_scope.
func isInvalidScopeError(status int, body string) bool {
	return status == 400 && strings.Contains(strings.ToLower(body), "invalid_scope")
}

func HasScope(scope, name string) bool {
	for _, s := range strings.Fields(scope) {
		if s == name {
			return true
		}
	}
	return false
}

func HasPremierScope(scope string) bool { return HasScope(scope, "premier") }

// requestOAuth2Token: one token request; nil means invalid_scope, other errors throw.
func requestOAuth2Token(clientID, clientSecret, scope string) (*OAuth2Token, error) {
	form := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"scope":         {scope},
	}
	req, _ := http.NewRequest(http.MethodPost, OAuth2TokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := httpx.ResilientFetch(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var data struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int64  `json:"expires_in"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("OAuth2 token: bad JSON: %s", string(body))
		}
		return &OAuth2Token{AccessToken: data.AccessToken, ExpiresIn: data.ExpiresIn, Scope: scope}, nil
	}
	if isInvalidScopeError(resp.StatusCode, string(body)) {
		return nil, nil
	}
	return nil, fmt.Errorf("OAuth2 token request failed: %d %s", resp.StatusCode, string(body))
}

func FetchOAuth2Token(clientID, clientSecret string, scopes []string) (*OAuth2Token, error) {
	var tried []string
	for _, scope := range scopes {
		tried = append(tried, scope)
		tok, err := requestOAuth2Token(clientID, clientSecret, scope)
		if err != nil {
			return nil, err
		}
		if tok != nil {
			return tok, nil
		}
	}
	return nil, fmt.Errorf("OAuth2 token request failed: 400 invalid_scope (tried: %s)", strings.Join(tried, " | "))
}

// detectOAuth2Token: first token for a key — all optional scopes at once, then
// probe each alone on invalid_scope. Reuse the returned scope for refreshes.
func DetectOAuth2Token(clientID, clientSecret string) (*OAuth2Token, error) {
	all := "basic " + strings.Join(optionalScopes, " ")
	if tok, err := requestOAuth2Token(clientID, clientSecret, all); err != nil {
		return nil, err
	} else if tok != nil {
		return tok, nil
	}
	var granted []string
	for _, scope := range optionalScopes {
		if tok, err := requestOAuth2Token(clientID, clientSecret, "basic "+scope); err != nil {
			return nil, err
		} else if tok != nil {
			granted = append(granted, scope)
		}
	}
	return FetchOAuth2Token(clientID, clientSecret, []string{strings.Join(append([]string{"basic"}, granted...), " ")})
}
