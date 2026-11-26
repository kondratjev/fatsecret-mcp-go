package tools

import (
	"context"
	"fmt"
	"github.com/kondratjev/fatsecret-mcp-go/internal/api"
	"github.com/kondratjev/fatsecret-mcp-go/internal/platform"
	"strconv"
	"strings"

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
	items, err := resolveLogMealItems(s, ctx, in.Items)
	if err != nil {
		return api.ToolResult(map[string]any{"error": map[string]any{"message": err.Error()}}, true)
	}
	var created []any
	var failed []any
	for _, r := range items {
		if r.failed != "" {
			failed = append(failed, map[string]any{"food_id": r.foodID, "error": r.failed})
			continue
		}
		q := map[string]string{
			"food_id":         strconv.Itoa(r.foodID),
			"food_entry_name": r.name,
			"serving_id":      strconv.Itoa(r.servingID),
			"number_of_units": strconv.FormatFloat(r.units, 'f', -1, 64),
			"meal":            in.Meal,
			"date":            daysStr,
		}
		data, reqFailed := s.ProfileRequest(ctx, "POST", "/food-entries/v1", q)
		entryID := foodEntryID(data)
		if reqFailed || entryID == 0 {
			failed = append(failed, map[string]any{"food_id": r.foodID, "error": fmt.Sprint(data)})
			continue
		}
		created = append(created, map[string]any{
			"food_entry_id": entryID, "food_entry_name": r.name,
			"serving_id": r.servingID, "number_of_units": r.units,
		})
	}
	day, _ := s.ProfileRequest(ctx, "GET", "/food-entries/v2", map[string]string{"date": daysStr})
	return api.ToolResult(map[string]any{
		"created":    nonNilSlice(created),
		"failed":     nonNilSlice(failed),
		"day_totals": platform.DayTotals(day),
	}, len(failed) > 0)
}

// resolvedMealItem is one log_meal item with servings resolved (reads only,
// no writes) so previews and execution share the same code.
type resolvedMealItem struct {
	foodID    int
	name      string
	servingID int
	units     float64
	desc      string
	kcal      float64
	protein   float64
	fat       float64
	carbs     float64
	failed    string
}

// eachRawServing walks the single-object-or-array serving shape of a raw
// /food/v6 response, which platform.Serving drops nutrition from.
func eachRawServing(data any, fn func(map[string]any)) {
	m, ok := data.(map[string]any)
	if !ok {
		return
	}
	fm, ok := m["food"].(map[string]any)
	if !ok {
		return
	}
	sv, ok := fm["servings"].(map[string]any)
	if !ok {
		return
	}
	switch t := sv["serving"].(type) {
	case map[string]any:
		fn(t)
	case []any:
		for _, item := range t {
			if sm, ok := item.(map[string]any); ok {
				fn(sm)
			}
		}
	}
}

// servingNutrition pulls one serving's description and scaled nutrition out
// of a raw /food/v6 response.
func servingNutrition(data any, servingID int, units float64) (desc string, kcal, protein, fat, carbs float64) {
	eachRawServing(data, func(sm map[string]any) {
		if desc != "" || int(platform.Num(sm["serving_id"])) != servingID {
			return
		}
		desc, _ = sm["serving_description"].(string)
		kcal = platform.Num(sm["calories"]) * units
		protein = platform.Num(sm["protein"]) * units
		fat = platform.Num(sm["fat"]) * units
		carbs = platform.Num(sm["carbohydrate"]) * units
	})
	return desc, kcal, protein, fat, carbs
}

// resolveLogMealItems resolves servings for every item. Reads (food.get) run
// in parallel under a small cap.
func resolveLogMealItems(s *api.Server, ctx context.Context, items []logMealItem) ([]resolvedMealItem, error) {
	type resolution struct {
		food *foodInfo
		raw  any
		err  string
	}
	resolved := make([]resolution, len(items))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(4)
	for i, item := range items {
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
			resolved[i] = resolution{food: food, raw: data}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	out := make([]resolvedMealItem, len(items))
	for i, item := range items {
		r := resolvedMealItem{foodID: item.FoodID}
		if item.ServingID != nil && item.FoodEntryName != nil {
			r.name = *item.FoodEntryName
			r.servingID = *item.ServingID
			r.units = 1
			if item.NumberOfUnits != nil {
				r.units = *item.NumberOfUnits
			}
		} else {
			res := resolved[i]
			if res.err != "" {
				r.failed = res.err
				out[i] = r
				continue
			}
			if item.FoodEntryName != nil {
				r.name = *item.FoodEntryName
			} else {
				r.name = res.food.name
			}
			var sid *int
			var nou *float64
			if item.ServingID != nil {
				sid = item.ServingID
			}
			if item.NumberOfUnits != nil {
				nou = item.NumberOfUnits
			}
			servingID, units, err := platform.ChooseServing(res.food.servings, sid, nou, item.Grams)
			if err != nil {
				r.failed = err.Error()
				out[i] = r
				continue
			}
			r.servingID, r.units = servingID, units
			r.desc, r.kcal, r.protein, r.fat, r.carbs = servingNutrition(res.raw, servingID, units)
		}
		out[i] = r
	}
	return out, nil
}

func trimNum(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// dayTotalsLine fetches current day totals for preview text.
func dayTotalsLine(s *api.Server, ctx context.Context, label, daysStr string) string {
	day, _ := s.ProfileRequest(ctx, "GET", "/food-entries/v2", map[string]string{"date": daysStr})
	total := map[string]any{}
	if t, ok := platform.DayTotals(day)["total"].(map[string]any); ok {
		total = t
	}
	return fmt.Sprintf("Day %s now: %.0f kcal (P %.0f, F %.0f, C %.0f).", label,
		platform.Num(total["calories"]), platform.Num(total["protein"]),
		platform.Num(total["fat"]), platform.Num(total["carbohydrate"]))
}

func previewLogMeal(s *api.Server, ctx context.Context, in logMealInput) (string, error) {
	date := "today"
	if in.Date != nil {
		date = *in.Date
	}
	days, err := platform.OptionalDateToDays(firstVal(in.Date))
	if err != nil {
		return "", err
	}
	daysStr := strconv.FormatInt(days, 10)
	items, err := resolveLogMealItems(s, ctx, in.Items)
	if err != nil {
		return "", err
	}
	lines := make([]string, 0, len(items))
	var kcal float64
	failed := 0
	for _, r := range items {
		if r.failed != "" {
			failed++
			lines = append(lines, fmt.Sprintf("- food_id %d: FAILED (%s)", r.foodID, r.failed))
			continue
		}
		kcal += r.kcal
		what := r.desc
		if what == "" {
			what = fmt.Sprintf("serving %d", r.servingID)
		}
		line := fmt.Sprintf("- %s: %s x %s", r.name, trimNum(r.units), what)
		if r.kcal > 0 {
			line += fmt.Sprintf(" (~%.0f kcal, P%.0f F%.0f C%.0f)", r.kcal, r.protein, r.fat, r.carbs)
		}
		lines = append(lines, line)
	}
	head := fmt.Sprintf("Will log %s on %s (%d items, ~%.0f kcal):", in.Meal, date, len(items), kcal)
	if failed > 0 {
		head += fmt.Sprintf(" %d item(s) would FAIL and be skipped.", failed)
	}
	return head + "\n" + strings.Join(lines, "\n") + "\n" + dayTotalsLine(s, ctx, date, daysStr), nil
}

func previewCreateFoodEntry(s *api.Server, ctx context.Context, in createFoodEntryInput) (string, error) {
	date := "today"
	if in.Date != nil {
		date = *in.Date
	}
	days, err := platform.OptionalDateToDays(firstVal(in.Date))
	if err != nil {
		return "", err
	}
	daysStr := strconv.FormatInt(days, 10)
	data, failed := s.PublicGET(ctx, "/food/v6", map[string]string{"food_id": strconv.Itoa(in.FoodID)})
	if failed {
		return "", fmt.Errorf("food %d not found", in.FoodID)
	}
	desc, kcal, protein, fat, carbs := servingNutrition(data, in.ServingID, in.NumberOfUnits)
	what := desc
	if what == "" {
		what = fmt.Sprintf("serving %d", in.ServingID)
	}
	return fmt.Sprintf("Will add %q (%s x %s) to %s on %s: ~%.0f kcal (P%.0f F%.0f C%.0f).\n%s",
		in.FoodEntryName, trimNum(in.NumberOfUnits), what, in.Meal, date,
		kcal, protein, fat, carbs, dayTotalsLine(s, ctx, date, daysStr)), nil
}

func firstVal(p *string) string {
	if p != nil {
		return *p
	}
	return ""
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
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "create_food_entry", Description: "Add a food diary entry. Requires food_id, serving_id, and meal type. Requires profile auth (check_auth_status first).", Annotations: writeTools(false)}, bindWrite(s, "create_food_entry", handleCreateFoodEntry, previewCreateFoodEntry))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "log_meal", Description: "Log a whole meal in one call: each item is a food_id plus grams (or serving_id + number_of_units); servings are resolved automatically. Returns the created entries and the day totals. Prefer this over several create_food_entry calls. Requires profile auth (check_auth_status first).", Annotations: writeTools(false)}, bindWrite(s, "log_meal", handleLogMeal, previewLogMeal))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "edit_food_entry", Description: "Edit an existing food diary entry. Cannot change the date. Requires profile auth (check_auth_status first).", Annotations: writeTools(true)}, bindWrite(s, "edit_food_entry", handleEditFoodEntry, nil))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "delete_food_entry", Description: "Delete a food diary entry by ID. Requires profile auth (check_auth_status first).", Annotations: destructiveTools()}, bindWrite(s, "delete_food_entry", handleDeleteFoodEntry, nil))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "copy_food_entries", Description: "Copy food entries from one date to another, optionally filtered by meal. Requires profile auth (check_auth_status first).", Annotations: writeTools(false)}, bindWrite(s, "copy_food_entries", handleCopyFoodEntries, nil))
	mcp.AddTool(s.MCP(), &mcp.Tool{Name: "copy_saved_meal_entries", Description: "Copy entries from a saved meal to a meal on a specific date. Requires profile auth (check_auth_status first).", Annotations: writeTools(false)}, bindWrite(s, "copy_saved_meal_entries", handleCopySavedMealEntries, nil))
}
