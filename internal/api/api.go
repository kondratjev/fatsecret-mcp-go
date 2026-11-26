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

// Server holds auth state: shared OAuth2 token cache (scope known after first
// request), app-level OAuth1 consumer credentials, and per-user OAuth1 tokens
// with their pending 3-legged flows. The user comes from the request context
// (X-MCP-User header over HTTP, default user over stdio).
type Server struct {
	mcp *mcp.Server

	mu             sync.Mutex
	clientID       string
	clientSecret   string
	consumerKey    string
	consumerSecret string
	users          map[string]*userAuth
	oauth2Access   string
	oauth2Expiry   time.Time
	oauth2Scope    string
	oauth2Known    bool
	oauth2Group    singleflight.Group
}

type userAuth struct {
	accessToken       string
	accessTokenSecret string
	pending           *oauthPending
}

type oauthPending struct {
	token  string
	secret string
}

// ctxUserKey carries the household member id. Empty = default user.
type ctxUserKey struct{}

// WithUser returns a context bound to the given household member.
func WithUser(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxUserKey{}, id)
}

// userID extracts the household member id from the context ("", if none).
func userID(ctx context.Context) string {
	return UserID(ctx)
}

// UserID extracts the household member id from the context ("", if none).
func UserID(ctx context.Context) string {
	if id, _ := ctx.Value(ctxUserKey{}).(string); id != "" {
		return id
	}
	return ""
}

func New(mcpServer *mcp.Server) *Server {
	cfg := config.Resolve()
	s := &Server{
		mcp:            mcpServer,
		clientID:       cfg.ClientID,
		clientSecret:   cfg.ClientSecret,
		consumerKey:    cfg.ClientID,
		consumerSecret: cfg.ConsumerSecret,
		users:          map[string]*userAuth{},
	}
	s.users[""] = &userAuth{accessToken: cfg.AccessToken, accessTokenSecret: cfg.AccessTokenSecret}
	for name, p := range cfg.Profiles {
		s.users[name] = &userAuth{accessToken: p.AccessToken, accessTokenSecret: p.AccessTokenSecret}
	}
	return s
}

// user returns the state for the context's household member, creating it
// lazily. Callers must not hold s.mu while using the result.
func (s *Server) user(ctx context.Context) *userAuth {
	id := userID(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[id]
	if !ok {
		u = &userAuth{}
		if s.users == nil {
			s.users = map[string]*userAuth{}
		}
		s.users[id] = u
	}
	return u
}

func (s *Server) HasAPI() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clientID != "" && s.clientSecret != "" && s.consumerSecret != ""
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

func (s *Server) ensureProfileAuth(ctx context.Context) error {
	if err := s.EnsureAPI(); err != nil {
		return err
	}
	u := s.user(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if u.accessToken == "" || u.accessTokenSecret == "" {
		if id := userID(ctx); id != "" {
			return fmt.Errorf("user %q is not authenticated for profile access. Use start_auth and complete_auth to authorize this user (each household member authorizes separately)", id)
		}
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
	if err := s.ensureProfileAuth(ctx); err != nil {
		return map[string]any{"error": map[string]any{"message": err.Error()}}, true
	}
	if query == nil {
		query = map[string]string{}
	}
	query["format"] = "json"
	u := s.user(ctx)
	s.mu.Lock()
	creds := auth.OAuth1Credentials{
		ConsumerKey:       s.consumerKey,
		ConsumerSecret:    s.consumerSecret,
		AccessToken:       u.accessToken,
		AccessTokenSecret: u.accessTokenSecret,
	}
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

// HasProfileTokens reports whether the context's user has OAuth1 user tokens.
func (s *Server) HasProfileTokens(ctx context.Context) bool {
	u := s.user(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	return u.accessToken != "" && u.accessTokenSecret != ""
}

// HasPremierScope reports whether the cached OAuth2 scope includes premier.
func (s *Server) HasPremierScope() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return auth.HasPremierScope(s.oauth2Scope)
}

// SetCredentials replaces API keys and clears cached auth state for all users.
func (s *Server) SetCredentials(clientID, clientSecret, consumerSecret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clientID = clientID
	s.clientSecret = clientSecret
	s.consumerKey = clientID
	s.consumerSecret = consumerSecret
	s.oauth2Access = ""
	s.oauth2Known = false
	s.users = map[string]*userAuth{}
}

// ConsumerCredentials returns the OAuth1 consumer key/secret for the 3-legged flow.
func (s *Server) ConsumerCredentials() (key, secret string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.consumerKey, s.consumerSecret
}

// SetPending stores the request-token pair awaiting user authorization.
func (s *Server) SetPending(ctx context.Context, token, secret string) {
	u := s.user(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	u.pending = &oauthPending{token: token, secret: secret}
}

// CompleteAuth exchanges the verifier for access tokens and persists them
// for the context's user.
func (s *Server) CompleteAuth(ctx context.Context, verifier string) error {
	u := s.user(ctx)
	s.mu.Lock()
	pending := u.pending
	creds := auth.OAuth1Credentials{
		ConsumerKey:    s.consumerKey,
		ConsumerSecret: s.consumerSecret,
	}
	s.mu.Unlock()
	if pending == nil {
		return fmt.Errorf("No pending OAuth flow. Call start_auth first.")
	}
	access, secret, err := auth.AccessToken(creds, pending.token, pending.secret, verifier)
	if err != nil {
		return err
	}
	if err := config.SaveProfileTokens(userID(ctx), config.ProfileTokens{AccessToken: access, AccessTokenSecret: secret}); err != nil {
		return err
	}
	s.mu.Lock()
	u.accessToken = access
	u.accessTokenSecret = secret
	u.pending = nil
	s.mu.Unlock()
	return nil
}
