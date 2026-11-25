package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// start_auth's link must survive the *_url trimming.
func TestToolResultKeepsAuthorizationLink(t *testing.T) {
	res, _, err := ToolResult(map[string]any{
		"message":            "visit",
		"authorization_link": "https://example.com/auth",
		"food_url":           "https://example.com/food",
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatal(err)
	}
	if data["authorization_link"] != "https://example.com/auth" {
		t.Fatalf("link stripped: %s", text)
	}
	if _, ok := data["food_url"]; ok {
		t.Fatalf("food_url not trimmed: %s", text)
	}
	if !strings.Contains(text, "visit") {
		t.Fatalf("message lost: %s", text)
	}
}
