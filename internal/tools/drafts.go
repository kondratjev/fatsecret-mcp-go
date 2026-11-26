package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kondratjev/fatsecret-mcp-go/internal/api"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Draft → confirm ceremony for writes. prepare_call validates arguments,
// builds a human-readable preview and stages the call as a single-use draft;
// confirm_draft executes it. Nothing is written before confirmation.
//
// Direct write tools keep working unless FATSECRET_REQUIRE_CONFIRM=1, in
// which case they refuse and point at prepare_call.

const draftTTL = 30 * time.Minute
const maxDrafts = 100

// requireConfirm is read once at startup: direct writes are disabled and
// every mutation must go through prepare_call + confirm_draft.
var requireConfirm = os.Getenv("FATSECRET_REQUIRE_CONFIRM") == "1"

// writeDef is a registered write tool: decode args, build a preview (reads
// only), and return the deferred execution.
type writeDef struct {
	prepare func(s *api.Server, ctx context.Context, raw json.RawMessage) (string, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, any, error), error)
}

var writeDefs = map[string]writeDef{}

// addWrite registers a write tool for the draft ceremony. pv may be nil, in
// which case the preview is the compacted arguments.
func addWrite[In any, Out any](name string, h func(*api.Server, context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error), pv func(*api.Server, context.Context, In) (string, error)) {
	writeDefs[name] = writeDef{
		prepare: func(s *api.Server, ctx context.Context, raw json.RawMessage) (string, func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, any, error), error) {
			var in In
			if err := json.Unmarshal(raw, &in); err != nil {
				return "", nil, fmt.Errorf("invalid arguments for %s: %w", name, err)
			}
			var preview string
			if pv != nil {
				p, err := pv(s, ctx, in)
				if err != nil {
					return "", nil, err
				}
				preview = p
			} else {
				var v any
				if err := json.Unmarshal(raw, &v); err != nil {
					return "", nil, err
				}
				compact, _ := json.Marshal(v)
				preview = fmt.Sprintf("Will call %s with: %s", name, compact)
			}
			exec := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, any, error) {
				res, out, err := h(s, ctx, req, in)
				return res, out, err
			}
			return preview, exec, nil
		},
	}
}

// bindWrite is bind for write tools: it registers the tool in the draft
// ceremony and, when FATSECRET_REQUIRE_CONFIRM=1, refuses direct execution.
func bindWrite[In any, Out any](s *api.Server, name string, h func(*api.Server, context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error), pv func(*api.Server, context.Context, In) (string, error)) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	addWrite(name, h, pv)
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		if requireConfirm {
			var zero Out
			res, _, _ := api.ToolResult(map[string]any{"error": map[string]any{"message": "writes require confirmation: stage this call with prepare_call first (FATSECRET_REQUIRE_CONFIRM=1)"}}, true)
			return res, zero, nil
		}
		return h(s, ctx, req, in)
	}
}

type draft struct {
	tool    string
	preview string
	created time.Time
	exec    func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, any, error)
}

type draftStore struct {
	mu     sync.Mutex
	drafts map[string]draft
}

func newDraftStore() *draftStore { return &draftStore{drafts: map[string]draft{}} }

var drafts = newDraftStore()

func draftID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// stage validates and stores a draft, evicting expired entries first and the
// oldest one when the store is full.
func (st *draftStore) stage(tool, preview string, exec func(context.Context, *mcp.CallToolRequest) (*mcp.CallToolResult, any, error)) string {
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	for id, d := range st.drafts {
		if now.Sub(d.created) > draftTTL {
			delete(st.drafts, id)
		}
	}
	if len(st.drafts) >= maxDrafts {
		oldest := ""
		var oldestTime time.Time
		for id, d := range st.drafts {
			if oldest == "" || d.created.Before(oldestTime) {
				oldest, oldestTime = id, d.created
			}
		}
		delete(st.drafts, oldest)
	}
	id := draftID()
	st.drafts[id] = draft{tool: tool, preview: preview, created: now, exec: exec}
	return id
}

// take removes and returns a draft. Single-use: the second confirm fails.
func (st *draftStore) take(id string) (draft, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, ok := st.drafts[id]
	if !ok {
		return draft{}, false
	}
	delete(st.drafts, id)
	if time.Since(d.created) > draftTTL {
		return draft{}, false
	}
	return d, true
}

type prepareCallInput struct {
	Tool      string         `json:"tool" jsonschema:"Write tool to stage, e.g. log_meal or delete_food_entry"`
	Arguments map[string]any `json:"arguments" jsonschema:"Arguments for the tool, as for a direct call"`
}

type confirmDraftInput struct {
	DraftID string `json:"draft_id" jsonschema:"Draft ID returned by prepare_call"`
}

func handlePrepareCall(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in prepareCallInput) (*mcp.CallToolResult, any, error) {
	def, ok := writeDefs[in.Tool]
	if !ok {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": "unknown write tool " + in.Tool + ": reads never need drafts"}}, true)
	}
	raw, err := json.Marshal(in.Arguments)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	preview, exec, err := def.prepare(s, ctx, raw)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	id := drafts.stage(in.Tool, preview, exec)
	return api.ToolResult(map[string]any{
		"draft_id":   id,
		"tool":       in.Tool,
		"preview":    preview,
		"expires_at": time.Now().Add(draftTTL).UTC().Format(time.RFC3339),
	}, false)
}

func handleConfirmDraft(_ *api.Server, ctx context.Context, req *mcp.CallToolRequest, in confirmDraftInput) (*mcp.CallToolResult, any, error) {
	d, ok := drafts.take(in.DraftID)
	if !ok {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": "unknown or expired draft: stage it again with prepare_call"}}, true)
	}
	return d.exec(ctx, req)
}

func registerDraftTools(s *api.Server) {
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "prepare_call", Description: "Stage a write (any diary, favorites, saved meal, weight, exercise, profile or feedback tool) as a single-use draft: arguments are validated and a human-readable preview is returned. Nothing is written. Drafts expire after 30 minutes. Use before risky writes so the user can review first.", Annotations: writeTools(false)}, bind(s, handlePrepareCall))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "confirm_draft", Description: "Execute a draft staged by prepare_call. Single-use: a draft_id works once.", Annotations: writeTools(false)}, bind(s, handleConfirmDraft))
}
