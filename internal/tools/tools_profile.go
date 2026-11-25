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

func registerProfileTools(s *api.Server) {
	const req = " Requires profile auth (check_auth_status first)."
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_profile", Description: "Get profile status information for the authenticated user." + req, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}}, bind(s, handleGetProfile))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "create_food", Description: "Create a custom food with nutrition info. Premier exclusive." + req, Annotations: writeTools(false)}, bind(s, handleCreateFood))
}
