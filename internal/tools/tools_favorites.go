package tools

import (
	"context"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type foodIDInput struct {
	FoodID int `json:"food_id" jsonschema:"Food ID"`
}

type recipeIDInput struct {
	RecipeID int `json:"recipe_id" jsonschema:"Recipe ID"`
}

type mealFilterInput struct {
	Meal *string `json:"meal,omitempty" jsonschema:"Meal type: breakfast, lunch, dinner or other"`
}

func handleGetFavoriteFoods(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in emptyInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "GET", "/food/favorites/v2", nil)
	return api.ToolResult(data, failed)
}

func handleAddFavoriteFood(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in foodIDInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "POST", "/food/favorite/v1", map[string]string{"food_id": strconv.Itoa(in.FoodID)})
	return api.ToolResult(data, failed)
}

func handleDeleteFavoriteFood(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in foodIDInput) (*mcp.CallToolResult, any, error) {
	// food.delete_favorite is DELETE; the Postman collection lists POST, which adds the food instead.
	data, failed := s.ProfileRequest(ctx, "DELETE", "/food/favorite/v1", map[string]string{"food_id": strconv.Itoa(in.FoodID)})
	return api.ToolResult(data, failed)
}

func handleGetMostEatenFoods(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in mealFilterInput) (*mcp.CallToolResult, any, error) {
	q := map[string]string{}
	if in.Meal != nil {
		q["meal"] = *in.Meal
	}
	data, failed := s.ProfileRequest(ctx, "GET", "/food/most-eaten/v2", q)
	return api.ToolResult(data, failed)
}

func handleGetRecentlyEatenFoods(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in mealFilterInput) (*mcp.CallToolResult, any, error) {
	q := map[string]string{}
	if in.Meal != nil {
		q["meal"] = *in.Meal
	}
	data, failed := s.ProfileRequest(ctx, "GET", "/food/recently-eaten/v2", q)
	return api.ToolResult(data, failed)
}

func handleGetFavoriteRecipes(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in emptyInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "GET", "/recipe/favorites/v2", nil)
	return api.ToolResult(data, failed)
}

func handleAddFavoriteRecipe(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in recipeIDInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "POST", "/recipe/favorites/v1", map[string]string{"recipe_id": strconv.Itoa(in.RecipeID)})
	return api.ToolResult(data, failed)
}

func handleDeleteFavoriteRecipe(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in recipeIDInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "DELETE", "/recipe/favorites/v1", map[string]string{"recipe_id": strconv.Itoa(in.RecipeID)})
	return api.ToolResult(data, failed)
}

func registerFavoriteTools(s *api.Server) {
	ro := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	const req = " Requires profile auth (check_auth_status first)."
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_favorite_foods", Description: "Get the user's favorite foods." + req, Annotations: ro}, bind(s, handleGetFavoriteFoods))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "add_favorite_food", Description: "Add a food to the user's favorites." + req, Annotations: writeTools(true)}, bindWrite(s, "add_favorite_food", handleAddFavoriteFood, nil))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "delete_favorite_food", Description: "Remove a food from the user's favorites." + req, Annotations: destructiveTools()}, bindWrite(s, "delete_favorite_food", handleDeleteFavoriteFood, nil))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_most_eaten_foods", Description: "Get the user's most eaten foods, optionally filtered by meal." + req, Annotations: ro}, bind(s, handleGetMostEatenFoods))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_recently_eaten_foods", Description: "Get the user's recently eaten foods, optionally filtered by meal." + req, Annotations: ro}, bind(s, handleGetRecentlyEatenFoods))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_favorite_recipes", Description: "Get the user's favorite recipes." + req, Annotations: ro}, bind(s, handleGetFavoriteRecipes))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "add_favorite_recipe", Description: "Add a recipe to the user's favorites." + req, Annotations: writeTools(true)}, bindWrite(s, "add_favorite_recipe", handleAddFavoriteRecipe, nil))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "delete_favorite_recipe", Description: "Remove a recipe from the user's favorites." + req, Annotations: destructiveTools()}, bindWrite(s, "delete_favorite_recipe", handleDeleteFavoriteRecipe, nil))
}
