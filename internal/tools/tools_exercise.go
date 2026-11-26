package tools

import (
	"context"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"github.com/kondratjev/fatsecret-mcp-go/internal/platform"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type editExerciseEntriesInput struct {
	ShiftToID   int     `json:"shift_to_id" jsonschema:"Exercise ID to shift time TO"`
	ShiftFromID int     `json:"shift_from_id" jsonschema:"Exercise ID to shift time FROM"`
	Minutes     int     `json:"minutes" jsonschema:"Minutes to shift"`
	Date        *string `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format (default today)"`
}

type saveExerciseTemplateInput struct {
	Days int     `json:"days" jsonschema:"Days of week as bit flags (Sun=bit 1, Sat=bit 7), converted to int"`
	Date *string `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format (default today)"`
}

func monthDays(date *string) (string, error) {
	d := ""
	if date != nil {
		d = *date
	}
	days, err := platform.OptionalDateToDays(d)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(days, 10), nil
}

func handleGetExerciseEntries(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in monthDateInput) (*mcp.CallToolResult, any, error) {
	days, err := monthDays(in.Date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	data, failed := s.ProfileRequest(ctx, "GET", "/exercise-entries/v2", map[string]string{"date": days})
	return api.ToolResult(data, failed)
}

func handleGetExercises(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in emptyInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "GET", "/exercises/v2", nil)
	return api.ToolResult(data, failed)
}

func handleEditExerciseEntries(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in editExerciseEntriesInput) (*mcp.CallToolResult, any, error) {
	days, err := monthDays(in.Date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	q := api.ToQueryMap(in)
	delete(q, "date")
	q["date"] = days
	data, failed := s.ProfileRequest(ctx, "PUT", "/exercise-entries/v1", q)
	return api.ToolResult(data, failed)
}

func handleGetExerciseEntriesMonth(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in monthDateInput) (*mcp.CallToolResult, any, error) {
	days, err := monthDays(in.Date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	data, failed := s.ProfileRequest(ctx, "GET", "/exercise-entries/month/v2", map[string]string{"date": days})
	return api.ToolResult(data, failed)
}

func handleSaveExerciseTemplate(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in saveExerciseTemplateInput) (*mcp.CallToolResult, any, error) {
	days, err := monthDays(in.Date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	data, failed := s.ProfileRequest(ctx, "POST", "/exercise-entries/day/v1", map[string]string{
		"days": strconv.Itoa(in.Days), "date": days,
	})
	return api.ToolResult(data, failed)
}

func handleCommitExerciseDay(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in monthDateInput) (*mcp.CallToolResult, any, error) {
	days, err := monthDays(in.Date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	data, failed := s.ProfileRequest(ctx, "POST", "/exercise-entries/day/v1", map[string]string{"date": days})
	return api.ToolResult(data, failed)
}

func registerExerciseTools(s *api.Server) {
	ro := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	const req = " Requires profile auth (check_auth_status first)."
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_exercise_entries", Description: "Get the user's exercise diary entries for a date." + req, Annotations: ro}, bind(s, handleGetExerciseEntries))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_exercises", Description: "Get the full list of supported exercise types and their IDs." + req, Annotations: ro}, bind(s, handleGetExercises))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "edit_exercise_entries", Description: "Shift exercise time between activities for a date. Moves minutes from one exercise to another." + req, Annotations: writeTools(false)}, bind(s, handleEditExerciseEntries))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_exercise_entries_month", Description: "Get daily calories expended from exercise for a month." + req, Annotations: ro}, bind(s, handleGetExerciseEntriesMonth))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "save_exercise_template", Description: "Save the current day's exercise entries as a template for specified days of the week." + req, Annotations: writeTools(true)}, bind(s, handleSaveExerciseTemplate))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "commit_exercise_day", Description: "Commit the day's template exercise entries to the diary (acknowledge the activities as done)." + req, Annotations: writeTools(true)}, bind(s, handleCommitExerciseDay))
}
