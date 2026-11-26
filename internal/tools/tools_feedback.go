package tools

import (
	"context"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ── Public API – Food feedback (paid add-on, "feedback" scope) ──

type foodFeedbackInput struct {
	IssueTypeID        int     `json:"issue_type_id" jsonschema:"1 Wrong Name/Brand, 2 Wrong Nutrition, 3 Missing Serving, 4 Barcode not found, 99 Other"`
	ExternalID         string  `json:"external_id" jsonschema:"Your own tracking identifier for this report"`
	Barcode            *string `json:"barcode,omitempty" jsonschema:"Barcode tied to the issue (keep leading zeros)"`
	IssueType          *string `json:"issue_type,omitempty" jsonschema:"Free-text description, required when issue_type_id is 99"`
	Notes              *string `json:"notes,omitempty" jsonschema:"Custom notes"`
	FoodID             *int    `json:"food_id,omitempty" jsonschema:"Related food ID (required for types 1, 2, 3)"`
	ServingID          *int    `json:"serving_id,omitempty" jsonschema:"Related serving ID (required for types 2, 3)"`
	ImageFileExtension *string `json:"image_file_extension,omitempty" jsonschema:"jpg, jpeg, png or tiff for the signed upload URLs"`
	Region             *string `json:"region,omitempty" jsonschema:"Country code, e.g. GB"`
	Language           *string `json:"language,omitempty" jsonschema:"Language code, e.g. en"`
}

func handleCreateFoodFeedback(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in foodFeedbackInput) (*mcp.CallToolResult, any, error) {
	if err := s.EnsureScope(ctx, "feedback", "Food feedback"); err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	body := map[string]any{
		"issue_type_id": in.IssueTypeID,
		"external_id":   in.ExternalID,
	}
	if in.Barcode != nil {
		body["barcode"] = *in.Barcode
	}
	if in.IssueType != nil {
		body["issue_type"] = *in.IssueType
	}
	if in.Notes != nil {
		body["notes"] = *in.Notes
	}
	if in.FoodID != nil {
		returned := map[string]any{"food_id": *in.FoodID}
		if in.ServingID != nil {
			returned["serving_id"] = *in.ServingID
		}
		body["returned_food"] = returned
	}
	if in.ImageFileExtension != nil {
		body["image_file_extension"] = *in.ImageFileExtension
	}
	if in.Region != nil {
		body["region"] = *in.Region
	}
	if in.Language != nil {
		body["language"] = *in.Language
	}
	data, failed := s.PostPublicJSON(ctx, "/feedback/v1", body)
	return api.ToolResult(data, failed)
}

func registerFeedbackTools(s *api.Server) {
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "create_food_feedback", Description: "Report wrong food/brand names, nutrition facts, missing servings or unknown barcodes. Returns signed upload URLs for evidence photos. Paid feedback add-on.", Annotations: writeTools(true)}, bindWrite(s, "create_food_feedback", handleCreateFoodFeedback, nil))
}
