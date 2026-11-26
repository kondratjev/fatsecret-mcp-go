package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kondratjev/fatsecret-mcp-go/internal/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func testEchoHandler(_ *api.Server, _ context.Context, _ *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, any, error) {
	return api.ToolResult(map[string]any{"echo": in["x"]}, false)
}

func TestDraftRoundTrip(t *testing.T) {
	addWrite("test_write", testEchoHandler, nil)
	def := writeDefs["test_write"]
	raw, _ := json.Marshal(map[string]any{"x": 1})
	preview, exec, err := def.prepare(nil, context.Background(), raw)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !strings.Contains(preview, "test_write") {
		t.Fatalf("default preview should name the tool, got %q", preview)
	}
	res, out, err := exec(context.Background(), nil)
	if err != nil || res.IsError {
		t.Fatalf("exec: %v %+v", err, out)
	}
}

func TestDraftSingleUseAndExpiry(t *testing.T) {
	st := newDraftStore()
	id := st.stage("test_write", "p", nil)
	if _, ok := st.take(id); !ok {
		t.Fatal("first take should succeed")
	}
	if _, ok := st.take(id); ok {
		t.Fatal("second take should fail: drafts are single-use")
	}
	if _, ok := st.take("nope"); ok {
		t.Fatal("unknown id should fail")
	}
	st.mu.Lock()
	st.drafts["old"] = draft{tool: "t", created: time.Now().Add(-time.Hour)}
	st.mu.Unlock()
	if _, ok := st.take("old"); ok {
		t.Fatal("expired draft should fail")
	}
}

func TestPrepareUnknownTool(t *testing.T) {
	res, _, _ := handlePrepareCall(nil, context.Background(), nil, prepareCallInput{Tool: "get_food"})
	if !res.IsError {
		t.Fatal("reads must not be stageable")
	}
}

func TestRequireConfirmGate(t *testing.T) {
	old := requireConfirm
	requireConfirm = true
	defer func() { requireConfirm = old }()
	h := bindWrite(nil, "test_gated", testEchoHandler, nil)
	res, _, _ := h(context.Background(), nil, map[string]any{})
	if !res.IsError {
		t.Fatal("gated write should refuse direct execution")
	}
	text, _ := json.Marshal(res.Content)
	if !strings.Contains(string(text), "prepare_call") {
		t.Fatalf("refusal should point at prepare_call, got %s", text)
	}
}
