package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// image_b64 field limit from the /image-recognition/v2 docs.
const maxImageB64Length = 999_982

var imageExtensions = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}

// resolveImageB64: reads image_path (images only) or takes image_b64, enforces the size limit.
func resolveImageB64(imagePath, imageB64 string) (string, error) {
	var b64 string
	if imagePath != "" {
		if !imageExtensions[strings.ToLower(filepath.Ext(imagePath))] {
			return "", fmt.Errorf("image_path must be a .jpg, .jpeg, .png or .webp file: %s", imagePath)
		}
		raw, err := os.ReadFile(imagePath)
		if err != nil {
			return "", err
		}
		b64 = base64.StdEncoding.EncodeToString(raw)
	} else if imageB64 != "" {
		if i := strings.Index(imageB64, ";base64,"); i >= 0 {
			imageB64 = imageB64[i+len(";base64,"):]
		}
		b64 = imageB64
	} else {
		return "", fmt.Errorf("provide image_path or image_b64")
	}
	if len(b64) > maxImageB64Length {
		return "", fmt.Errorf("image too large for FatSecret (%d base64 chars, max %d). Shrink it first, e.g. `sips -Z 512 photo.jpg --out small.jpg` on macOS", len(b64), maxImageB64Length)
	}
	return b64, nil
}

type eatenFood struct {
	FoodID             int      `json:"food_id" jsonschema:"Food ID"`
	FoodName           string   `json:"food_name" jsonschema:"Food name"`
	FoodBrand          *string  `json:"food_brand,omitempty" jsonschema:"Brand name"`
	ServingDescription *string  `json:"serving_description,omitempty" jsonschema:"Serving description, e.g. whole"`
	ServingSize        *float64 `json:"serving_size,omitempty" jsonschema:"Serving size"`
}

type parseFoodTextInput struct {
	UserInput       string      `json:"user_input" jsonschema:"Free-text meal description, e.g. 2 eggs, a slice of toast and a cappuccino"`
	IncludeFoodData *bool       `json:"include_food_data,omitempty" jsonschema:"Include full food and serving data for each match (default true)"`
	EatenFoods      []eatenFood `json:"eaten_foods,omitempty" jsonschema:"Foods the user usually eats, used to improve matching"`
	Region          *string     `json:"region,omitempty" jsonschema:"Country code, e.g. GB"`
	Language        *string     `json:"language,omitempty" jsonschema:"Language code, e.g. en"`
}

type recognizeFoodImageInput struct {
	ImagePath       *string     `json:"image_path,omitempty" jsonschema:"Local path to a .jpg/.jpeg/.png/.webp photo, at most ~730 KB"`
	ImageB64        *string     `json:"image_b64,omitempty" jsonschema:"Base64-encoded image, used when image_path is not given"`
	IncludeFoodData *bool       `json:"include_food_data,omitempty" jsonschema:"Include full food and serving data for each match (default true)"`
	EatenFoods      []eatenFood `json:"eaten_foods,omitempty" jsonschema:"Foods the user usually eats, used to improve matching"`
	Region          *string     `json:"region,omitempty" jsonschema:"Country code, e.g. GB"`
	Language        *string     `json:"language,omitempty" jsonschema:"Language code, e.g. en"`
}

func strOr(s *string, def string) string {
	if s == nil {
		return def
	}
	return *s
}

func handleParseFoodText(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in parseFoodTextInput) (*mcp.CallToolResult, any, error) {
	if err := s.EnsureScope(ctx, "nlp", "Natural language processing"); err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	include := true
	if in.IncludeFoodData != nil {
		include = *in.IncludeFoodData
	}
	body := map[string]any{"user_input": in.UserInput, "include_food_data": include}
	if in.EatenFoods != nil {
		body["eaten_foods"] = in.EatenFoods
	}
	if in.Region != nil {
		body["region"] = *in.Region
	}
	if in.Language != nil {
		body["language"] = *in.Language
	}
	data, failed := s.PostPublicJSON(ctx, "/natural-language-processing/v1", body)
	return api.ToolResult(data, failed)
}

func handleRecognizeFoodImage(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in recognizeFoodImageInput) (*mcp.CallToolResult, any, error) {
	if err := s.EnsureScope(ctx, "image-recognition", "Image recognition"); err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	b64, err := resolveImageB64(strOr(in.ImagePath, ""), strOr(in.ImageB64, ""))
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	include := true
	if in.IncludeFoodData != nil {
		include = *in.IncludeFoodData
	}
	body := map[string]any{"image_b64": b64, "include_food_data": include}
	if in.EatenFoods != nil {
		body["eaten_foods"] = in.EatenFoods
	}
	if in.Region != nil {
		body["region"] = *in.Region
	}
	if in.Language != nil {
		body["language"] = *in.Language
	}
	data, failed := s.PostPublicJSON(ctx, "/image-recognition/v2", body)
	return api.ToolResult(data, failed)
}

func registerRecognitionTools(s *api.Server) {
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "parse_food_text", Description: "Turn a free-text meal description (2 eggs and a slice of toast) into matched FatSecret foods with suggested servings and nutrition, ready for create_food_entry. Paid NLP add-on (nlp scope).", Annotations: roTools()}, bind(s, handleParseFoodText))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "recognize_food_image", Description: "Recognize foods in a meal photo and match them to FatSecret foods with suggested servings and nutrition, ready for create_food_entry. Pass image_path (local file) or image_b64. Paid image recognition add-on (image-recognition scope).", Annotations: roTools()}, bind(s, handleRecognizeFoodImage))
}
