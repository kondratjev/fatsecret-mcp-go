package platform

import "testing"

func TestIsMissingScopeError(t *testing.T) {
	strCode := map[string]any{"error": map[string]any{"code": "14", "message": "Missing scope"}}
	numCode := map[string]any{"error": map[string]any{"code": 14.0}}
	other := map[string]any{"error": map[string]any{"code": "4"}}
	ok := map[string]any{"foods_search": map[string]any{}}
	if !IsMissingScopeError(strCode) || !IsMissingScopeError(numCode) {
		t.Fatal("should detect code 14 in both forms")
	}
	if IsMissingScopeError(other) || IsMissingScopeError(ok) || IsMissingScopeError(nil) {
		t.Fatal("false positive")
	}
}

func TestTrimSearchServings(t *testing.T) {
	data := map[string]any{"foods_search": map[string]any{"results": map[string]any{"food": []any{
		map[string]any{"servings": map[string]any{"serving": map[string]any{
			"serving_id": "1", "calories": "100", "vitamin_a": "5", "food_url": "x",
		}}},
	}}}}
	TrimSearchServings(data)
	s := data["foods_search"].(map[string]any)["results"].(map[string]any)["food"].([]any)[0].(map[string]any)["servings"].(map[string]any)["serving"].(map[string]any)
	if s["calories"] != "100" || s["serving_id"] != "1" {
		t.Fatalf("kept fields missing: %v", s)
	}
	if _, ok := s["vitamin_a"]; ok {
		t.Fatalf("micronutrient not trimmed: %v", s)
	}
}
