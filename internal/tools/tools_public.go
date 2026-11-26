package tools

import (
	"context"
	"strings"

	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"github.com/kondratjev/fatsecret-mcp-go/internal/platform"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ── Public API – Foods ──

type searchFoodsInput struct {
	SearchExpression              string  `json:"search_expression" jsonschema:"Search query for foods"`
	PageNumber                    *int    `json:"page_number,omitempty" jsonschema:"Page number"`
	MaxResults                    *int    `json:"max_results,omitempty" jsonschema:"Max results (up to 50)"`
	FlagDefaultServing            *bool   `json:"flag_default_serving,omitempty" jsonschema:"Flag the default serving"`
	FoodType                      *string `json:"food_type,omitempty" jsonschema:"generic, brand or none"`
	Region                        *string `json:"region,omitempty" jsonschema:"Country code, e.g. GB"`
	Language                      *string `json:"language,omitempty" jsonschema:"Language code, e.g. en"`
	IncludeFoodImages             *bool   `json:"include_food_images,omitempty" jsonschema:"Include food images (Premier)"`
	IncludeFoodAttributes         *bool   `json:"include_food_attributes,omitempty" jsonschema:"Include allergens and preferences (Premier)"`
	IncludeFoodClassificationData *bool   `json:"include_food_classification_data,omitempty" jsonschema:"Include food classification (Premier)"`
	IncludeNutrientContentTags    *bool   `json:"include_nutrient_content_tags,omitempty" jsonschema:"Include nutrient content tags (Premier)"`
}

type getFoodInput struct {
	FoodID                        int     `json:"food_id" jsonschema:"Food ID"`
	FlagDefaultServing            *bool   `json:"flag_default_serving,omitempty" jsonschema:"Flag the default serving"`
	Region                        *string `json:"region,omitempty" jsonschema:"Country code, e.g. GB"`
	Language                      *string `json:"language,omitempty" jsonschema:"Language code, e.g. en"`
	IncludeFoodImages             *bool   `json:"include_food_images,omitempty" jsonschema:"Include food images (Premier)"`
	IncludeFoodAttributes         *bool   `json:"include_food_attributes,omitempty" jsonschema:"Include allergens and preferences (Premier)"`
	IncludeFoodClassificationData *bool   `json:"include_food_classification_data,omitempty" jsonschema:"Include food classification (Premier)"`
	IncludeNutrientContentTags    *bool   `json:"include_nutrient_content_tags,omitempty" jsonschema:"Include nutrient content tags (Premier)"`
}

type findFoodByBarcodeInput struct {
	Barcode string `json:"barcode" jsonschema:"GTIN-13 barcode number"`
}

type autocompleteFoodsInput struct {
	Expression string  `json:"expression" jsonschema:"Partial search expression"`
	MaxResults *int    `json:"max_results,omitempty" jsonschema:"Max results"`
	Region     *string `json:"region,omitempty" jsonschema:"Country code, e.g. GB"`
	Language   *string `json:"language,omitempty" jsonschema:"Language code, e.g. en"`
}

var v1OnlyParams = []string{"flag_default_serving", "food_type", "region", "language", "include_food_images",
	"include_food_attributes", "include_food_classification_data", "include_nutrient_content_tags"}

func handleSearchFoods(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in searchFoodsInput) (*mcp.CallToolResult, any, error) {
	if _, err := s.GetOAuth2Token(ctx); err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	premier := s.HasPremierScope()
	if premier {
		data, failed := s.PublicGET(ctx, "/foods/search/v6", api.ToQueryMap(in))
		if !platform.IsMissingScopeError(data) {
			if !failed {
				data = platform.TrimSearchServings(data)
			}
			return api.ToolResult(data, failed)
		}
	}
	v1 := map[string]string{}
	if in.SearchExpression != "" {
		v1["search_expression"] = in.SearchExpression
	}
	full := api.ToQueryMap(in)
	if v, ok := full["page_number"]; ok {
		v1["page_number"] = v
	}
	if v, ok := full["max_results"]; ok {
		v1["max_results"] = v
	}
	data, failed := s.PublicGET(ctx, "/foods/search/v1", v1)
	if failed {
		return api.ToolResult(data, true)
	}
	var ignored []string
	for _, k := range v1OnlyParams {
		if _, ok := full[k]; ok {
			ignored = append(ignored, k)
		}
	}
	if len(ignored) > 0 {
		if m, ok := data.(map[string]any); ok {
			m["note"] = "Basic plan: ignored: " + strings.Join(ignored, ", ")
			data = m
		}
	}
	return api.ToolResult(data, false)
}

func handleGetFood(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in getFoodInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.PublicGET(ctx, "/food/v6", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func handleFindFoodByBarcode(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in findFoodByBarcodeInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.PublicGET(ctx, "/food/barcode/find-by-id/v3", map[string]string{"barcode": in.Barcode})
	return api.ToolResult(data, failed)
}

func handleAutocompleteFoods(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in autocompleteFoodsInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.PublicGET(ctx, "/food/autocomplete/v2", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func roTools() *mcp.ToolAnnotations {
	t := true
	return &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: &t}
}

func roRefTools() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
}

func writeTools(idempotent bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: false, IdempotentHint: idempotent}
}

func destructiveTools() *mcp.ToolAnnotations {
	t := true
	return &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &t, IdempotentHint: true}
}

func registerPublicFoodTools(s *api.Server) {
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "search_foods", Description: "Search the FatSecret food database. Returns food names and servings with calories and macros (use get_food for micronutrients). On the Basic plan only search_expression, page_number and max_results (max 50) are used.", Annotations: roTools()}, bind(s, handleSearchFoods))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_food", Description: "Get detailed nutritional information for a specific food by ID. Returns servings, calories, macros, and micronutrients.", Annotations: roTools()}, bind(s, handleGetFood))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "find_food_by_barcode", Description: "Find food by barcode (GTIN-13). UPC-A, EAN-13 and EAN-8 supported. Needs the barcode scope (Premier).", Annotations: roTools()}, bind(s, handleFindFoodByBarcode))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "autocomplete_foods", Description: "Get autocomplete suggestions for a partial food search expression. Premier exclusive.", Annotations: roTools()}, bind(s, handleAutocompleteFoods))
}
