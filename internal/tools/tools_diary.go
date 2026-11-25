package tools

import (
	"context"
	"fmt"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"github.com/kondratjev/fatsecret-mcp-go/internal/platform"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
)

const mealDesc = "Meal type: breakfast, lunch, dinner or other"

// ── Profile API – Food Diary ──

type getFoodEntriesInput struct {
	FoodEntryID *int    `json:"food_entry_id,omitempty" jsonschema:"Food entry ID (fetches the entry directly)"`
	Date        *string `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD (required if food_entry_id not specified)"`
}

type monthDateInput struct {
	Date *string `json:"date,omitempty" jsonschema:"Any date within the target month (YYYY-MM-DD)"`
}

type createFoodEntryInput struct {
	FoodID        int     `json:"food_id" jsonschema:"Food ID"`
	FoodEntryName string  `json:"food_entry_name" jsonschema:"Name for the food entry"`
	ServingID     int     `json:"serving_id" jsonschema:"Serving size ID"`
	NumberOfUnits float64 `json:"number_of_units" jsonschema:"Number of serving units"`
	Meal          string  `json:"meal" jsonschema:"Meal type: breakfast, lunch, dinner or other"`
	Date          *string `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format (default today)"`
}

type logMealItem struct {
	FoodID        int      `json:"food_id" jsonschema:"Food ID"`
	Grams         *float64 `json:"grams,omitempty" jsonschema:"Amount eaten in grams; a gram-based serving is picked automatically"`
	ServingID     *int     `json:"serving_id,omitempty" jsonschema:"Serving ID, instead of grams"`
	NumberOfUnits *float64 `json:"number_of_units,omitempty" jsonschema:"Units of serving_id (default 1)"`
	FoodEntryName *string  `json:"food_entry_name,omitempty" jsonschema:"Diary name (default: the food name)"`
}

type logMealInput struct {
	Meal  string        `json:"meal" jsonschema:"Meal type: breakfast, lunch, dinner or other"`
	Date  *string       `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format (default today)"`
	Items []logMealItem `json:"items" jsonschema:"Foods eaten in this meal"`
}

type editFoodEntryInput struct {
	FoodEntryID   int      `json:"food_entry_id" jsonschema:"Food entry ID to edit"`
	FoodEntryName *string  `json:"food_entry_name,omitempty" jsonschema:"Name for the food entry"`
	ServingID     *int     `json:"serving_id,omitempty" jsonschema:"Serving size ID"`
	NumberOfUnits *float64 `json:"number_of_units,omitempty" jsonschema:"Number of serving units"`
	Meal          *string  `json:"meal,omitempty" jsonschema:"Meal type: breakfast, lunch, dinner or other"`
}

type deleteFoodEntryInput struct {
	FoodEntryID int `json:"food_entry_id" jsonschema:"Food entry ID to delete"`
}

type copyFoodEntriesInput struct {
	FromDate string  `json:"from_date" jsonschema:"Source date YYYY-MM-DD"`
	ToDate   string  `json:"to_date" jsonschema:"Target date YYYY-MM-DD"`
	Meal     *string `json:"meal,omitempty" jsonschema:"Meal type: breakfast, lunch, dinner or other"`
}

type copySavedMealEntriesInput struct {
	SavedMealID int     `json:"saved_meal_id" jsonschema:"Saved meal ID to copy"`
	Meal        string  `json:"meal" jsonschema:"Meal type: breakfast, lunch, dinner or other"`
	Date        *string `json:"date,omitempty" jsonschema:"Date in YYYY-MM-DD format (default today)"`
}

func handleGetFoodEntries(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in getFoodEntriesInput) (*mcp.CallToolResult, any, error) {
	q := map[string]string{}
	if in.FoodEntryID != nil && *in.FoodEntryID > 0 {
		q["food_entry_id"] = strconv.Itoa(*in.FoodEntryID)
		if in.Date != nil {
			days, err := platform.DateToDays(*in.Date)
			if err != nil {
				return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
			}
			q["date"] = strconv.FormatInt(days, 10)
		}
	} else {
		date := ""
		if in.Date != nil {
			date = *in.Date
		}
		days, err := platform.OptionalDateToDays(date)
		if err != nil {
			return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
		}
		q["date"] = strconv.FormatInt(days, 10)
	}
	data, failed := s.ProfileRequest(ctx, "GET", "/food-entries/v2", q)
	return api.ToolResult(data, failed)
}

func handleGetFoodEntriesMonth(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in monthDateInput) (*mcp.CallToolResult, any, error) {
	date := ""
	if in.Date != nil {
		date = *in.Date
	}
	days, err := platform.OptionalDateToDays(date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	data, failed := s.ProfileRequest(ctx, "GET", "/food-entries/month/v2", map[string]string{"date": strconv.FormatInt(days, 10)})
	return api.ToolResult(data, failed)
}

func handleCreateFoodEntry(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in createFoodEntryInput) (*mcp.CallToolResult, any, error) {
	date := ""
	if in.Date != nil {
		date = *in.Date
	}
	days, err := platform.OptionalDateToDays(date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	q := map[string]string{
		"food_id":         strconv.Itoa(in.FoodID),
		"food_entry_name": in.FoodEntryName,
		"serving_id":      strconv.Itoa(in.ServingID),
		"number_of_units": strconv.FormatFloat(in.NumberOfUnits, 'f', -1, 64),
		"meal":            in.Meal,
		"date":            strconv.FormatInt(days, 10),
	}
	data, failed := s.ProfileRequest(ctx, "POST", "/food-entries/v1", q)
	return api.ToolResult(data, failed)
}

func handleLogMeal(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in logMealInput) (*mcp.CallToolResult, any, error) {
	date := ""
	if in.Date != nil {
		date = *in.Date
	}
	days, err := platform.OptionalDateToDays(date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	daysStr := strconv.FormatInt(days, 10)
	// Reads (food.get) run in parallel under a small cap; writes stay
	// sequential — bursts trip the API call limits.
	type resolution struct {
		food *foodInfo
		err  string
	}
	resolved := make([]resolution, len(in.Items))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(4)
	for i, item := range in.Items {
		if item.ServingID != nil && item.FoodEntryName != nil {
			continue // fast path: no read needed
		}
		g.Go(func() error {
			data, _ := s.PublicGET(gctx, "/food/v6", map[string]string{"food_id": strconv.Itoa(item.FoodID)})
			food := foodFromGet(data)
			if food == nil {
				resolved[i] = resolution{err: "food not found"}
				return nil
			}
			resolved[i] = resolution{food: food}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	var created []any
	var failed []any
	for i, item := range in.Items {
		func() {
			var name string
			var servingID int
			var units float64
			if item.ServingID != nil && item.FoodEntryName != nil {
				name = *item.FoodEntryName
				servingID = *item.ServingID
				units = 1
				if item.NumberOfUnits != nil {
					units = *item.NumberOfUnits
				}
			} else {
				res := resolved[i]
				if res.err != "" {
					failed = append(failed, map[string]any{"food_id": item.FoodID, "error": res.err})
					return
				}
				food := res.food
				if item.FoodEntryName != nil {
					name = *item.FoodEntryName
				} else {
					name = food.name
				}
				var sid *int
				var nou *float64
				if item.ServingID != nil {
					sid = item.ServingID
				}
				if item.NumberOfUnits != nil {
					nou = item.NumberOfUnits
				}
				var err error
				servingID, units, err = platform.ChooseServing(food.servings, sid, nou, item.Grams)
				if err != nil {
					failed = append(failed, map[string]any{"food_id": item.FoodID, "error": err.Error()})
					return
				}
			}
			q := map[string]string{
				"food_id":         strconv.Itoa(item.FoodID),
				"food_entry_name": name,
				"serving_id":      strconv.Itoa(servingID),
				"number_of_units": strconv.FormatFloat(units, 'f', -1, 64),
				"meal":            in.Meal,
				"date":            daysStr,
			}
			data, reqFailed := s.ProfileRequest(ctx, "POST", "/food-entries/v1", q)
			entryID := foodEntryID(data)
			if reqFailed || entryID == 0 {
				msg := fmt.Sprint(data)
				failed = append(failed, map[string]any{"food_id": item.FoodID, "error": msg})
				return
			}
			created = append(created, map[string]any{
				"food_entry_id": entryID, "food_entry_name": name,
				"serving_id": servingID, "number_of_units": units,
			})
		}()
	}
	day, _ := s.ProfileRequest(ctx, "GET", "/food-entries/v2", map[string]string{"date": daysStr})
	return api.ToolResult(map[string]any{
		"created":    nonNilSlice(created),
		"failed":     nonNilSlice(failed),
		"day_totals": platform.DayTotals(day),
	}, len(failed) > 0)
}

func nonNilSlice(v []any) any {
	if v == nil {
		return []any{}
	}
	return v
}

type foodInfo struct {
	name     string
	servings []platform.Serving
}

func foodFromGet(data any) *foodInfo {
	m, ok := data.(map[string]any)
	if !ok {
		return nil
	}
	fm, ok := m["food"].(map[string]any)
	if !ok {
		return nil
	}
	name, _ := fm["food_name"].(string)
	if name == "" {
		return nil
	}
	var servings []platform.Serving
	if sv, ok := fm["servings"].(map[string]any); ok {
		servings = platform.ToServings(sv["serving"])
	}
	return &foodInfo{name: name, servings: servings}
}

func foodEntryID(data any) int {
	m, ok := data.(map[string]any)
	if !ok {
		return 0
	}
	return int(platform.Num(m["food_entry_id"]))
}

func handleEditFoodEntry(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in editFoodEntryInput) (*mcp.CallToolResult, any, error) {
	q := map[string]string{"food_entry_id": strconv.Itoa(in.FoodEntryID)}
	if in.FoodEntryName != nil {
		q["food_entry_name"] = *in.FoodEntryName
	}
	if in.ServingID != nil {
		q["serving_id"] = strconv.Itoa(*in.ServingID)
	}
	if in.NumberOfUnits != nil {
		q["number_of_units"] = strconv.FormatFloat(*in.NumberOfUnits, 'f', -1, 64)
	}
	if in.Meal != nil {
		q["meal"] = *in.Meal
	}
	data, failed := s.ProfileRequest(ctx, "PUT", "/food-entries/v1", q)
	return api.ToolResult(data, failed)
}

func handleDeleteFoodEntry(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in deleteFoodEntryInput) (*mcp.CallToolResult, any, error) {
	data, failed := s.ProfileRequest(ctx, "DELETE", "/food-entries/v1", map[string]string{"food_entry_id": strconv.Itoa(in.FoodEntryID)})
	return api.ToolResult(data, failed)
}

func handleCopyFoodEntries(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in copyFoodEntriesInput) (*mcp.CallToolResult, any, error) {
	from, err := platform.DateToDays(in.FromDate)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	to, err := platform.DateToDays(in.ToDate)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	q := map[string]string{"from_date": strconv.FormatInt(from, 10), "to_date": strconv.FormatInt(to, 10)}
	if in.Meal != nil {
		q["meal"] = *in.Meal
	}
	data, failed := s.ProfileRequest(ctx, "POST", "/food-entries/copy/v1", q)
	return api.ToolResult(data, failed)
}

func handleCopySavedMealEntries(s *api.Server, ctx context.Context, req *mcp.CallToolRequest, in copySavedMealEntriesInput) (*mcp.CallToolResult, any, error) {
	date := ""
	if in.Date != nil {
		date = *in.Date
	}
	days, err := platform.OptionalDateToDays(date)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	data, failed := s.ProfileRequest(ctx, "POST", "/food-entries/copy/saved-meal/v1", map[string]string{
		"saved_meal_id": strconv.Itoa(in.SavedMealID),
		"meal":          in.Meal,
		"date":          strconv.FormatInt(days, 10),
	})
	return api.ToolResult(data, failed)
}

func registerFoodDiaryTools(s *api.Server) {
	ro := &mcp.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true}
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_food_entries", Description: "Get food diary entries for a date or a specific entry by ID. Requires profile auth (check_auth_status first).", Annotations: ro}, bind(s, handleGetFoodEntries))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "get_food_entries_month", Description: "Get daily nutrition summary for a month. Returns calories and macros per day. Requires profile auth (check_auth_status first).", Annotations: ro}, bind(s, handleGetFoodEntriesMonth))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "create_food_entry", Description: "Add a food diary entry. Requires food_id, serving_id, and meal type. Requires profile auth (check_auth_status first).", Annotations: writeTools(false)}, bind(s, handleCreateFoodEntry))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "log_meal", Description: "Log a whole meal in one call: each item is a food_id plus grams (or serving_id + number_of_units); servings are resolved automatically. Returns the created entries and the day totals. Prefer this over several create_food_entry calls. Requires profile auth (check_auth_status first).", Annotations: writeTools(false)}, bind(s, handleLogMeal))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "edit_food_entry", Description: "Edit an existing food diary entry. Cannot change the date. Requires profile auth (check_auth_status first).", Annotations: writeTools(true)}, bind(s, handleEditFoodEntry))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "delete_food_entry", Description: "Delete a food diary entry by ID. Requires profile auth (check_auth_status first).", Annotations: destructiveTools()}, bind(s, handleDeleteFoodEntry))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "copy_food_entries", Description: "Copy food entries from one date to another, optionally filtered by meal. Requires profile auth (check_auth_status first).", Annotations: writeTools(false)}, bind(s, handleCopyFoodEntries))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "copy_saved_meal_entries", Description: "Copy entries from a saved meal to a meal on a specific date. Requires profile auth (check_auth_status first).", Annotations: writeTools(false)}, bind(s, handleCopySavedMealEntries))
}
