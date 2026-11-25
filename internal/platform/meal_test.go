package platform

import "testing"

func TestChooseServingGrams(t *testing.T) {
	servings := []Serving{
		{ServingID: "1", ServingDescription: "cup", MetricServingAmount: "240", MetricServingUnit: "g", NumberOfUnits: "1"},
		{ServingID: "2", ServingDescription: "g", MeasurementDescription: "g", MetricServingAmount: "100", MetricServingUnit: "g", NumberOfUnits: "1"},
	}
	g := 150.0
	id, units, err := ChooseServing(servings, nil, nil, &g)
	if err != nil {
		t.Fatal(err)
	}
	if id != 2 || units != 1.5 {
		t.Fatalf("got id=%d units=%v", id, units)
	}
}

func TestChooseServingDefault(t *testing.T) {
	servings := []Serving{
		{ServingID: "1", IsDefault: "0"},
		{ServingID: "5", IsDefault: "1"},
	}
	id, units, err := ChooseServing(servings, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != 5 || units != 1 {
		t.Fatalf("got id=%d units=%v", id, units)
	}
}

func TestToServingsShapes(t *testing.T) {
	single := map[string]any{"serving_id": "7"}
	if got := ToServings(single); len(got) != 1 || got[0].ServingID != "7" {
		t.Fatalf("single object: %+v", got)
	}
	arr := []any{map[string]any{"serving_id": "7"}, map[string]any{"serving_id": "8"}}
	if got := ToServings(arr); len(got) != 2 {
		t.Fatalf("array: %+v", got)
	}
	if got := ToServings(nil); len(got) != 0 {
		t.Fatalf("nil: %+v", got)
	}
}

func TestDayTotals(t *testing.T) {
	resp := map[string]any{"food_entries": map[string]any{"food_entry": []any{
		map[string]any{"meal": "Breakfast", "calories": "100", "protein": "5", "fat": "1", "carbohydrate": "20"},
		map[string]any{"meal": "Lunch", "calories": "200", "protein": "10", "fat": "2", "carbohydrate": "30"},
	}}}
	tot := DayTotals(resp)
	if tot["entries"] != 2 {
		t.Fatalf("entries: %v", tot["entries"])
	}
	total := tot["total"].(map[string]any)
	if total["calories"] != 300.0 || total["protein"] != 15.0 {
		t.Fatalf("total: %v", total)
	}
}
