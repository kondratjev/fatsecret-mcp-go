package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kondratjev/fatsecret-mcp-go/internal/auth"
	"github.com/kondratjev/fatsecret-mcp-go/internal/config"
	"github.com/kondratjev/fatsecret-mcp-go/internal/httpx"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/singleflight"
)

const baseURL = "https://platform.fatsecret.com/rest"

// Server holds auth state: OAuth2 token cache (scope known after first
// request), OAuth1 credentials, and the pending 3-legged flow.
type Server struct {
	mcp *mcp.Server

	mu           sync.Mutex
	clientID     string
	clientSecret string

	oauth2Access string
	oauth2Expiry time.Time
	oauth2Scope  string
	oauth2Known  bool
	oauth2Group  singleflight.Group

	oauth1  auth.OAuth1Credentials
	pending *oauthPending
}

type oauthPending struct {
	token  string
	secret string
}

func New(mcpServer *mcp.Server) *Server {
	cfg := config.Resolve()
	return &Server{
		mcp:          mcpServer,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		oauth1: auth.OAuth1Credentials{
			ConsumerKey:       cfg.ClientID,
			ConsumerSecret:    cfg.ConsumerSecret,
			AccessToken:       cfg.AccessToken,
			AccessTokenSecret: cfg.AccessTokenSecret,
		},
	}
}

func (s *Server) HasAPI() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clientID != "" && s.clientSecret != "" && s.oauth1.ConsumerSecret != ""
}

func (s *Server) EnsureAPI() error {
	if !s.HasAPI() {
		return fmt.Errorf("API credentials not configured. Use setup_credentials tool first. " +
			"Get your credentials at https://platform.fatsecret.com/ → My Account → API Keys")
	}
	return nil
}

func (s *Server) GetOAuth2Token(ctx context.Context) (string, error) {
	if err := s.EnsureAPI(); err != nil {
		return "", err
	}
	s.mu.Lock()
	tok, exp := s.oauth2Access, s.oauth2Expiry
	s.mu.Unlock()
	if tok != "" && time.Now().Before(exp) {
		return tok, nil
	}
	v, err, _ := s.oauth2Group.Do("token", func() (any, error) {
		s.mu.Lock()
		known, scope := s.oauth2Known, s.oauth2Scope
		clientID, clientSecret := s.clientID, s.clientSecret
		s.mu.Unlock()
		var scopes []string
		if known {
			scopes = []string{scope}
		} else {
			scopes = nil // detectOAuth2Token probes
		}
		var t *auth.OAuth2Token
		var err error
		if scopes == nil {
			t, err = auth.DetectOAuth2Token(clientID, clientSecret)
		} else {
			t, err = auth.FetchOAuth2Token(clientID, clientSecret, scopes)
		}
		if err != nil {
			return "", err
		}
		s.mu.Lock()
		s.oauth2Access = t.AccessToken
		s.oauth2Expiry = time.Now().Add(time.Duration(t.ExpiresIn-60) * time.Second)
		s.oauth2Scope = t.Scope
		s.oauth2Known = true
		s.mu.Unlock()
		return t.AccessToken, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (s *Server) ensureProfileAuth() error {
	if err := s.EnsureAPI(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.oauth1.AccessToken == "" || s.oauth1.AccessTokenSecret == "" {
		return fmt.Errorf("not authenticated for profile access. Use check_auth_status to check, then start_auth and complete_auth to authorize")
	}
	return nil
}

// toQueryMap converts a struct (json tags = API param names) to query params,
// skipping nil pointers and empty strings.
func ToQueryMap(v any) map[string]string {
	out := map[string]string{}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return out
	}
	rt := rv.Type()
	for i := range rt.NumField() {
		name, _, _ := strings.Cut(rt.Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		fv := rv.Field(i)
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}
		switch fv.Kind() {
		case reflect.String:
			if fv.String() != "" {
				out[name] = fv.String()
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			out[name] = strconv.FormatInt(fv.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			out[name] = strconv.FormatUint(fv.Uint(), 10)
		case reflect.Float32, reflect.Float64:
			out[name] = strconv.FormatFloat(fv.Float(), 'f', -1, 64)
		case reflect.Bool:
			out[name] = strconv.FormatBool(fv.Bool())
		default:
			if b, err := json.Marshal(fv.Interface()); err == nil {
				out[name] = string(b)
			}
		}
	}
	return out
}

func encodeQuery(q map[string]string) string {
	vals := url.Values{}
	for k, v := range q {
		vals.Set(k, v)
	}
	return vals.Encode()
}

// parseAPIResponse decodes the body JSON, or wraps transport failures and
// non-JSON bodies as {error:{status,message}}.
func parseAPIResponse(resp *http.Response) (any, bool) {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var data any
	if json.Unmarshal(body, &data) != nil || data == nil {
		data = map[string]any{"error": map[string]any{"status": resp.StatusCode, "message": string(body)}}
		return data, true
	}
	return data, false
}

// publicGET: OAuth2 GET call with format=json.
func (s *Server) PublicGET(ctx context.Context, path string, query map[string]string) (any, bool) {
	token, err := s.GetOAuth2Token(ctx)
	if err != nil {
		return map[string]any{"error": map[string]any{"message": err.Error()}}, true
	}
	if query == nil {
		query = map[string]string{}
	}
	query["format"] = "json"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path+"?"+encodeQuery(query), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpx.ResilientFetch(req)
	if err != nil {
		return map[string]any{"error": map[string]any{"message": err.Error()}}, true
	}
	return parseAPIResponse(resp)
}

// postPublicJSON: POST a JSON body with the OAuth2 token (NLP, image recognition).
func (s *Server) PostPublicJSON(ctx context.Context, path string, body any) (any, bool) {
	token, err := s.GetOAuth2Token(ctx)
	if err != nil {
		return map[string]any{"error": map[string]any{"message": err.Error()}}, true
	}
	payload, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpx.ResilientFetch(req)
	if err != nil {
		return map[string]any{"error": map[string]any{"message": err.Error()}}, true
	}
	return parseAPIResponse(resp)
}

// profileRequest: OAuth1-signed call, params in the query string.
func (s *Server) ProfileRequest(ctx context.Context, method, path string, query map[string]string) (any, bool) {
	if err := s.ensureProfileAuth(); err != nil {
		return map[string]any{"error": map[string]any{"message": err.Error()}}, true
	}
	if query == nil {
		query = map[string]string{}
	}
	query["format"] = "json"
	s.mu.Lock()
	creds := s.oauth1
	s.mu.Unlock()
	signed := auth.BuildOAuth1Params(method, baseURL+path, creds, query)
	req, _ := http.NewRequestWithContext(ctx, method, baseURL+path+"?"+encodeQuery(signed), nil)
	resp, err := httpx.ResilientFetch(req)
	if err != nil {
		return map[string]any{"error": map[string]any{"message": err.Error()}}, true
	}
	return parseAPIResponse(resp)
}

// ensureScope: add-ons are separate OAuth2 scopes; say so up front.
func (s *Server) EnsureScope(ctx context.Context, scope, feature string) error {
	if _, err := s.GetOAuth2Token(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	granted := s.oauth2Scope
	s.mu.Unlock()
	if !auth.HasScope(granted, scope) {
		return fmt.Errorf("%s is not enabled for this FatSecret key (granted scopes: %s). "+
			"It is a paid add-on: enable it at https://platform.fatsecret.com/ and restart the server", feature, granted)
	}
	return nil
}

// MCP exposes the underlying MCP server for tool registration and serving.
func (s *Server) MCP() *mcp.Server { return s.mcp }

// HasProfileTokens reports whether OAuth1 user tokens are present.
func (s *Server) HasProfileTokens() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.oauth1.AccessToken != "" && s.oauth1.AccessTokenSecret != ""
}

// HasPremierScope reports whether the cached OAuth2 scope includes premier.
func (s *Server) HasPremierScope() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return auth.HasPremierScope(s.oauth2Scope)
}

// SetCredentials replaces API keys and clears cached auth state.
func (s *Server) SetCredentials(clientID, clientSecret, consumerSecret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clientID = clientID
	s.clientSecret = clientSecret
	s.oauth1.ConsumerKey = clientID
	s.oauth1.ConsumerSecret = consumerSecret
	s.oauth2Access = ""
	s.oauth2Known = false
	s.oauth1.AccessToken = ""
	s.oauth1.AccessTokenSecret = ""
	s.pending = nil
}

// ConsumerCredentials returns the OAuth1 consumer key/secret for the 3-legged flow.
func (s *Server) ConsumerCredentials() (key, secret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.oauth1.ConsumerKey, s.oauth1.ConsumerSecret
}

// SetPending stores the request-token pair awaiting user authorization.
func (s *Server) SetPending(token, secret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = &oauthPending{token: token, secret: secret}
}

// CompleteAuth exchanges the verifier for access tokens and persists them.
func (s *Server) CompleteAuth(verifier string) error {
	s.mu.Lock()
	pending := s.pending
	creds := s.oauth1
	s.mu.Unlock()
	if pending == nil {
		return fmt.Errorf("No pending OAuth flow. Call start_auth first.")
	}
	access, secret, err := auth.AccessToken(creds, pending.token, pending.secret, verifier)
	if err != nil {
		return err
	}
	if err := config.Save(map[string]string{"accessToken": access, "accessTokenSecret": secret}); err != nil {
		return err
	}
	s.mu.Lock()
	s.oauth1.AccessToken = access
	s.oauth1.AccessTokenSecret = secret
	s.pending = nil
	s.mu.Unlock()
	return nil
}
