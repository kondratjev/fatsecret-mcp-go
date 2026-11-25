package tools

import (
	"context"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type foodCategoryID struct {
	FoodCategoryID *int `json:"food_category_id,omitempty" jsonschema:"Food category ID"`
}

type getFoodSubCategoriesInput struct {
	FoodCategoryID int `json:"food_category_id" jsonschema:"Food category ID"`
}

type getBrandsInput struct {
	StartsWith *string `json:"starts_with,omitempty" jsonschema:"Filter brands by starting letter"`
	BrandType  *string `json:"brand_type,omitempty" jsonschema:"Brand type filter"`
	PageNumber *int    `json:"page_number,omitempty" jsonschema:"Page number"`
	MaxResults *int    `json:"max_results,omitempty" jsonschema:"Max results"`
}

type emptyInput struct{}

func handleGetFoodCategories(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in emptyInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.PublicGET(ctx, "/food-categories/v2", nil)
	return api.ToolResult(data, failed)
}

func handleGetFoodSubCategories(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in getFoodSubCategoriesInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.PublicGET(ctx, "/food-sub-categories/v2", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func handleGetBrands(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in getBrandsInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.PublicGET(ctx, "/brands/v2", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func handleGetRecipeTypes(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in emptyInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.PublicGET(ctx, "/recipe-types/v2", nil)
	return api.ToolResult(data, failed)
}

func registerReferenceTools(s *api.Server) {
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_food_categories", Description: "Get the full list of food categories. Premier exclusive.", Annotations: roRefTools()}, bind(s, handleGetFoodCategories))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_food_sub_categories", Description: "Get food sub categories for a given food category. Premier exclusive.", Annotations: roRefTools()}, bind(s, handleGetFoodSubCategories))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_brands", Description: "Get the list of food brands, optionally filtered by starting letter and type. Premier exclusive.", Annotations: roRefTools()}, bind(s, handleGetBrands))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_recipe_types", Description: "Get the full list of supported recipe type names.", Annotations: roRefTools()}, bind(s, handleGetRecipeTypes))
}
