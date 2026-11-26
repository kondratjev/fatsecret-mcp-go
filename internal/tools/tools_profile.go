package tools

import (
	"context"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type createFoodInput struct {
	BrandName    string  `json:"brand_name" jsonschema:"Brand name"`
	FoodName     string  `json:"food_name" jsonschema:"Food name"`
	ServingSize  string  `json:"serving_size" jsonschema:"Serving size description (e.g. 1 slice)"`
	Calories     float64 `json:"calories" jsonschema:"Calories (kcal)"`
	Fat          float64 `json:"fat" jsonschema:"Total fat (g)"`
	Carbohydrate float64 `json:"carbohydrate" jsonschema:"Total carbohydrate (g)"`
	Protein      float64 `json:"protein" jsonschema:"Protein (g)"`
}

func handleGetProfile(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in emptyInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "GET", "/profile/v1", nil)
	return api.ToolResult(data, failed)
}

func handleCreateFood(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in createFoodInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "POST", "/food/v2", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

type createProfileInput struct {
	UserID string `json:"user_id" jsonschema:"Your own ID for the new detached profile"`
}

type profileAuthInput struct {
	UserID *string `json:"user_id,omitempty" jsonschema:"Your own ID of an existing detached profile"`
}

func handleCreateProfile(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in createProfileInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "POST", "/profile/v1", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func handleGetProfileAuth(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in profileAuthInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "GET", "/profile/auth/v1", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func registerProfileTools(s *api.Server) {
	const req = " Requires profile auth (check_auth_status first)."
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_profile", Description: "Get profile status information for the authenticated user." + req, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}}, bind(s, handleGetProfile))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "create_food", Description: "Create a custom food with nutrition info. Premier exclusive." + req, Annotations: writeTools(false)}, bindWrite(s, "create_food", handleCreateFood, nil))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "create_profile", Description: "Create a detached profile for your own user ID. Returns auth_token/auth_secret to store — it does NOT switch this server to the new profile." + req, Annotations: writeTools(false)}, bindWrite(s, "create_profile", handleCreateProfile, nil))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_profile_auth", Description: "Get auth_token/auth_secret for a detached profile by your own user ID." + req, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}}, bind(s, handleGetProfileAuth))
}
