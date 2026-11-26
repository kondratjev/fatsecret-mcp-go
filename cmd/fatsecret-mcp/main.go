package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"github.com/kondratjev/fatsecret-mcp-go/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var version = "dev" // stamped by GoReleaser via -ldflags -X main.version=…

const instructions = `FatSecret MCP server provides two levels of access:

1. SETUP: If API credentials are not configured, call check_auth_status first.
   It will tell you if credentials are missing and guide through setup_credentials.
   Get credentials at https://platform.fatsecret.com/ → My Account → API Keys.

2. PUBLIC API (works after setup): Food search, recipes, brands, categories.
   These tools use OAuth 2.0 with the configured Client ID and Client Secret.

3. PROFILE API (requires user authorization): Food diary, saved meals, favorites, weight, exercises, profile.
   These tools require OAuth 1.0 user authorization. Before using any profile tool,
   call check_auth_status to see if the user is authenticated.
   If not, guide them through: start_auth → user visits URL and authorizes → complete_auth with verifier PIN.
   All credentials and tokens persist across sessions in ~/.fatsecret-mcp/config.json.
   Household: over HTTP, the X-MCP-User header selects the member (each authorizes
   separately); stdio always uses the default member.`

func buildServer() *api.Server {
	ms := mcp.NewServer(&mcp.Implementation{Name: "fatsecret-mcp", Version: version}, &mcp.ServerOptions{
		Instructions: instructions,
	})
	s := api.New(ms)
	tools.RegisterAll(s)
	return s
}

func serveStdio(s *api.Server) error {
	return s.MCP().Run(context.Background(), &mcp.StdioTransport{})
}

func serveHTTP(s *api.Server, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	// Stateless handler bound to our single server instance. The X-MCP-User
	// header selects the household member (each authorizes separately via
	// start_auth/complete_auth); stdio clients always use the default user.
	stateless := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return s.MCP() }, &mcp.StreamableHTTPOptions{Stateless: true})
	mux.Handle("/mcp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := r.Header.Get("X-MCP-User"); user != "" {
			r = r.WithContext(api.WithUser(r.Context(), user))
		}
		stateless.ServeHTTP(w, r)
	}))
	log.Printf("FatSecret MCP server listening on :%d", port)
	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       time.Minute,
	}
	return srv.ListenAndServe()
}

func main() {
	log.SetOutput(os.Stderr)
	s := buildServer()
	// Warm the OAuth2 token cache in the background: first use probes scopes
	// (up to 8 token calls), so pay it at startup, not on the first tool call.
	// Without credentials this fails fast and is a no-op.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if _, err := s.GetOAuth2Token(ctx); err != nil {
			log.Printf("token warmup skipped: %v", err)
		}
	}()
	if portStr := os.Getenv("PORT"); portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("bad PORT %q: %v", portStr, err)
		}
		if err := serveHTTP(s, port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
		return
	}
	if err := serveStdio(s); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
