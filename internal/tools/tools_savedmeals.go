package tools

import (
	"context"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type getSavedMealsInput struct {
	Meal *string `json:"meal,omitempty" jsonschema:"Filter by meal type"`
}

type createSavedMealInput struct {
	SavedMealName        string  `json:"saved_meal_name" jsonschema:"Meal name"`
	SavedMealDescription *string `json:"saved_meal_description,omitempty" jsonschema:"Meal description"`
	Meal                 *string `json:"meal,omitempty" jsonschema:"Associated meals"`
}

type editSavedMealInput struct {
	SavedMealID          int     `json:"saved_meal_id" jsonschema:"Saved meal ID"`
	SavedMealName        *string `json:"saved_meal_name,omitempty" jsonschema:"Meal name"`
	SavedMealDescription *string `json:"saved_meal_description,omitempty" jsonschema:"Meal description"`
	Meal                 *string `json:"meal,omitempty" jsonschema:"Associated meals"`
}

type deleteSavedMealInput struct {
	SavedMealID int `json:"saved_meal_id" jsonschema:"Saved meal ID to delete"`
}

type getSavedMealItemsInput struct {
	SavedMealID int `json:"saved_meal_id" jsonschema:"Saved meal ID"`
}

type addSavedMealItemInput struct {
	SavedMealID       int     `json:"saved_meal_id" jsonschema:"Saved meal ID"`
	FoodID            int     `json:"food_id" jsonschema:"Food ID to add"`
	SavedMealItemName string  `json:"saved_meal_item_name" jsonschema:"Item name"`
	ServingID         int     `json:"serving_id" jsonschema:"Serving ID"`
	NumberOfUnits     float64 `json:"number_of_units" jsonschema:"Number of serving units"`
}

type editSavedMealItemInput struct {
	SavedMealItemID   int      `json:"saved_meal_item_id" jsonschema:"Saved meal item ID"`
	SavedMealItemName *string  `json:"saved_meal_item_name,omitempty" jsonschema:"Item name"`
	NumberOfUnits     *float64 `json:"number_of_units,omitempty" jsonschema:"Number of serving units"`
}

type deleteSavedMealItemInput struct {
	SavedMealItemID int `json:"saved_meal_item_id" jsonschema:"Saved meal item ID to delete"`
}

func handleGetSavedMeals(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in getSavedMealsInput) (*mcp.CallToolResult, any, error) {
	q := map[string]string{}
	if in.Meal != nil {
		q["meal"] = *in.Meal
	}
	data, failed := s.ProfileRequest(ctx, "GET", "/saved-meals/v2", q)
	return api.ToolResult(data, failed)
}

func handleCreateSavedMeal(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in createSavedMealInput) (*mcp.CallToolResult, any, error) {
	q := map[string]string{"saved_meal_name": in.SavedMealName}
	if in.SavedMealDescription != nil {
		q["saved_meal_description"] = *in.SavedMealDescription
	}
	if in.Meal != nil {
		q["meal"] = *in.Meal
	}
	data, failed := s.ProfileRequest(ctx, "POST", "/saved-meals/v1", q)
	return api.ToolResult(data, failed)
}

func handleEditSavedMeal(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in editSavedMealInput) (*mcp.CallToolResult, any, error) {
	q := map[string]string{"saved_meal_id": strconv.Itoa(in.SavedMealID)}
	if in.SavedMealName != nil {
		q["saved_meal_name"] = *in.SavedMealName
	}
	if in.SavedMealDescription != nil {
		q["saved_meal_description"] = *in.SavedMealDescription
	}
	if in.Meal != nil {
		q["meal"] = *in.Meal
	}
	data, failed := s.ProfileRequest(ctx, "PUT", "/saved-meals/v1", q)
	return api.ToolResult(data, failed)
}

func handleDeleteSavedMeal(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in deleteSavedMealInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "DELETE", "/saved-meals/v1", map[string]string{"saved_meal_id": strconv.Itoa(in.SavedMealID)})
	return api.ToolResult(data, failed)
}

func handleGetSavedMealItems(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in getSavedMealItemsInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "GET", "/saved-meals/item/v2", map[string]string{"saved_meal_id": strconv.Itoa(in.SavedMealID)})
	return api.ToolResult(data, failed)
}

func handleAddSavedMealItem(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in addSavedMealItemInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "POST", "/saved-meals/item/v1", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func handleEditSavedMealItem(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in editSavedMealItemInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "PUT", "/saved-meals/item/v1", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func handleDeleteSavedMealItem(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in deleteSavedMealItemInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "DELETE", "/saved-meals/item/v1", api.ToQueryMap(in))
	return api.ToolResult(data, failed)
}

func registerSavedMealTools(s *api.Server) {
	ro := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	const req = " Requires profile auth (check_auth_status first)."
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_saved_meals", Description: "Get the user's saved meals, optionally filtered by meal type." + req, Annotations: ro}, bind(s, handleGetSavedMeals))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "create_saved_meal", Description: "Create a new saved meal." + req, Annotations: writeTools(false)}, bind(s, handleCreateSavedMeal))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "edit_saved_meal", Description: "Edit a saved meal name, description, or associated meals." + req, Annotations: writeTools(true)}, bind(s, handleEditSavedMeal))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "delete_saved_meal", Description: "Delete a saved meal." + req, Annotations: destructiveTools()}, bind(s, handleDeleteSavedMeal))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_saved_meal_items", Description: "Get all food items in a saved meal." + req, Annotations: ro}, bind(s, handleGetSavedMealItems))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "add_saved_meal_item", Description: "Add a food item to a saved meal." + req, Annotations: writeTools(false)}, bind(s, handleAddSavedMealItem))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "edit_saved_meal_item", Description: "Edit a food item in a saved meal (name or units). Cannot change serving_id." + req, Annotations: writeTools(true)}, bind(s, handleEditSavedMealItem))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "delete_saved_meal_item", Description: "Remove a food item from a saved meal." + req, Annotations: destructiveTools()}, bind(s, handleDeleteSavedMealItem))
}
