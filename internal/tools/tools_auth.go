package tools

import (
	"context"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"github.com/kondratjev/fatsecret-mcp-go/internal/auth"
	"github.com/kondratjev/fatsecret-mcp-go/internal/config"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type setupCredentialsInput struct {
	ClientID       string `json:"client_id" jsonschema:"FatSecret Client ID (from platform.fatsecret.com → API Keys)"`
	ClientSecret   string `json:"client_secret" jsonschema:"FatSecret Client Secret (OAuth 2.0)"`
	ConsumerSecret string `json:"consumer_secret" jsonschema:"FatSecret Consumer Secret (OAuth 1.0 — different from Client Secret, found on the same API Keys page)"`
}

type completeAuthInput struct {
	Verifier string `json:"verifier" jsonschema:"OAuth verifier code from the authorization page"`
}

func handleCheckAuthStatus(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in emptyInput) (*mcp.CallToolResult, any, error) {
	hasCreds := s.HasAPI()
	hasTokens := s.HasProfileTokens()
	path, _ := config.Path()
	if !hasCreds {
		return api.ToolResult(map[string]any{
			"credentials_configured": false,
			"profile_authenticated":  false,
			"config_path":            path,
			"message": "API credentials are not configured. Use setup_credentials to provide your FatSecret API keys. " +
				"Get them at https://platform.fatsecret.com/ → My Account → API Keys. " +
				"You need: Client ID, Client Secret (OAuth 2.0), and Consumer Secret (OAuth 1.0 — different from Client Secret).",
		}, false)
	}
	msg := "API credentials configured (public tools work). Profile not authenticated — use start_auth to authorize profile access."
	if hasTokens {
		msg = "Fully configured. API credentials and profile authentication are ready. All tools are available."
	}
	return api.ToolResult(map[string]any{
		"credentials_configured": true,
		"profile_authenticated":  hasTokens,
		"config_path":            path,
		"message":                msg,
	}, false)
}

func handleSetupCredentials(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in setupCredentialsInput) (*mcp.CallToolResult, any, error) {
	s.SetCredentials(in.ClientID, in.ClientSecret, in.ConsumerSecret)
	if err := config.Save(map[string]string{
		"clientId": in.ClientID, "clientSecret": in.ClientSecret,
		"consumerSecret": in.ConsumerSecret, "accessToken": "", "accessTokenSecret": "",
	}); err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	path, _ := config.Path()
	return api.ToolResult(map[string]any{
		"message":     "Credentials saved! Public API tools (food search, recipes) are now available. For profile tools (food diary, weight, etc.), use start_auth to authorize your FatSecret account.",
		"config_path": path,
	}, false)
}

func handleStartAuth(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in emptyInput) (*mcp.CallToolResult, any, error) {
	if err := s.EnsureAPI(); err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	key, secret := s.ConsumerCredentials()
	creds := auth.OAuth1Credentials{ConsumerKey: key, ConsumerSecret: secret}
	token, secret, authURL, err := auth.RequestToken(creds)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	s.SetPending(token, secret)
	return api.ToolResult(map[string]any{
		"message": "Visit the URL below to authorize the app, then use complete_auth with the verifier code.",
		// NB: must not end in _url — ToolResult drops *_url keys to save tokens.
		"authorization_link": authURL,
	}, false)
}

func handleCompleteAuth(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in completeAuthInput) (*mcp.CallToolResult, any, error) {
	if err := s.CompleteAuth(in.Verifier); err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	path, _ := config.Path()
	return api.ToolResult(map[string]any{
		"message":     "Authentication successful! Profile tools are now available.",
		"config_path": path,
	}, false)
}

func registerAuthTools(s *api.Server) {
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "check_auth_status", Description: "Check if API credentials and profile authentication are configured. Call this first to understand what setup is needed.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}}, bind(s, handleCheckAuthStatus))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "setup_credentials", Description: "Configure FatSecret API credentials. Get them at https://platform.fatsecret.com/ → My Account → API Keys. Saves to persistent config file.", Annotations: writeTools(true)}, bind(s, handleSetupCredentials))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "start_auth", Description: "Start the OAuth 1.0 authorization flow for profile access. Returns an authorization URL the user must visit. Requires API credentials (setup_credentials first).", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: false}}, bind(s, handleStartAuth))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "complete_auth", Description: "Complete the OAuth 1.0 flow with the verifier code from the authorization page.", Annotations: writeTools(false)}, bind(s, handleCompleteAuth))
}
