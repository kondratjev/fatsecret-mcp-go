package tools

import (
	"context"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"github.com/kondratjev/fatsecret-mcp-go/internal/platform"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type updateWeightInput struct {
	CurrentWeightKg float64  `json:"current_weight_kg" jsonschema:"Current weight in kg"`
	Date            *string  `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format (default today)"`
	GoalWeightKg    *float64 `json:"goal_weight_kg,omitempty" jsonschema:"Goal weight in kg (needed on first weigh-in)"`
	CurrentHeightCm *float64 `json:"current_height_cm,omitempty" jsonschema:"Current height in cm (needed on first weigh-in)"`
	Comment         *string  `json:"comment,omitempty" jsonschema:"Comment"`
}

func handleUpdateWeight(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in updateWeightInput) (*mcp.CallToolResult, any, error) {
	date := ""
	if in.Date != nil {
		date = *in.Date
	}
	days, err := platform.OptionalDateToDays(date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	q := api.ToQueryMap(in)
	delete(q, "date")
	q["date"] = strconv.FormatInt(days, 10)
	data, failed := s.ProfileRequest(ctx, "POST", "/weight/v1", q)
	return api.ToolResult(data, failed)
}

func handleGetWeightMonth(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in monthDateInput) (*mcp.CallToolResult, any, error) {
	date := ""
	if in.Date != nil {
		date = *in.Date
	}
	days, err := platform.OptionalDateToDays(date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	data, failed := s.ProfileRequest(ctx, "GET", "/weight/month/v2", map[string]string{"date": strconv.FormatInt(days, 10)})
	return api.ToolResult(data, failed)
}

func registerWeightTools(s *api.Server) {
	const req = " Requires profile auth (check_auth_status first)."
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "update_weight", Description: "Record the user's weight for a date. First weigh-in requires goal_weight_kg and current_height_cm." + req, Annotations: writeTools(true)}, bind(s, handleUpdateWeight))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_weight_month", Description: "Get the user's weight entries for a month." + req, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}}, bind(s, handleGetWeightMonth))
}
