package tools

import (
	"context"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type searchRecipesInput struct {
	SearchExpression *string  `json:"search_expression,omitempty" jsonschema:"Search query for recipes"`
	RecipeType       *string  `json:"recipe_type,omitempty" jsonschema:"Recipe type filter"`
	MaxResults       *int     `json:"max_results,omitempty" jsonschema:"Max results"`
	PageNumber       *int     `json:"page_number,omitempty" jsonschema:"Page number"`
	MinCalories      *float64 `json:"min_calories,omitempty" jsonschema:"Minimum calories"`
	MaxCalories      *float64 `json:"max_calories,omitempty" jsonschema:"Maximum calories"`
	MinProtein       *float64 `json:"min_protein,omitempty" jsonschema:"Minimum protein (g)"`
	MaxProtein       *float64 `json:"max_protein,omitempty" jsonschema:"Maximum protein (g)"`
	MinFat           *float64 `json:"min_fat,omitempty" jsonschema:"Minimum fat (g)"`
	MaxFat           *float64 `json:"max_fat,omitempty" jsonschema:"Maximum fat (g)"`
	MinCarbohydrate  *float64 `json:"min_carbohydrate,omitempty" jsonschema:"Minimum carbs (g)"`
	MaxCarbohydrate  *float64 `json:"max_carbohydrate,omitempty" jsonschema:"Maximum carbs (g)"`
	MaxPrepTime      *int     `json:"max_prep_time_mins,omitempty" jsonschema:"Maximum prep time in minutes"`
}

type getRecipeInput struct {
	RecipeID int `json:"recipe_id" jsonschema:"Recipe ID"`
}

func handleSearchRecipes(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in searchRecipesInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.PublicGET(ctx, "/recipes/search/v3", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func handleGetRecipe(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in getRecipeInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.PublicGET(ctx, "/recipe/v2", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func registerRecipeTools(s *api.Server) {
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "search_recipes", Description: "Search recipes with optional filters for calories, macros, prep time, and recipe types.", Annotations: roTools()}, bind(s, handleSearchRecipes))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_recipe", Description: "Get detailed recipe information by ID including ingredients, directions, and nutrition.", Annotations: roTools()}, bind(s, handleGetRecipe))
}
