package api

import (
	"encoding/json"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// isAPIError: FatSecret reports most failures as HTTP 200 with {"error": ...}.
func isAPIError(data any) bool {
	m, ok := data.(map[string]any)
	return ok && m["error"] != nil
}

func dropURLs(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if strings.HasSuffix(k, "_url") {
				continue
			}
			out[k] = dropURLs(val)
		}
		return out
	case []any:
		for i, item := range t {
			t[i] = dropURLs(item)
		}
		return t
	default:
		return v
	}
}

// ToolResult: compact JSON without *_url links (~55% fewer tokens); flags
// transport failures and {"error": ...} payloads via IsError.
func ToolResult(data any, failed bool) (*mcp.CallToolResult, any, error) {
	if data == nil {
		data = map[string]any{}
	}
	text, err := json.Marshal(dropURLs(data))
	if err != nil {
		text = []byte(`{"error":{"message":"failed to encode response"}}`)
		failed = true
	}
	isErr := failed || isAPIError(data)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(text)}},
		IsError: isErr,
	}, nil, nil
}
