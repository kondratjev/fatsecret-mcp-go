package platform

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Serving as returned by /food/v6 (all numbers come back as strings).
type Serving struct {
	ServingID              string `json:"serving_id"`
	ServingDescription     string `json:"serving_description"`
	MeasurementDescription string `json:"measurement_description"`
	NumberOfUnits          string `json:"number_of_units"`
	MetricServingAmount    string `json:"metric_serving_amount"`
	MetricServingUnit      string `json:"metric_serving_unit"`
	IsDefault              string `json:"is_default"`
}

func round3(n float64) float64 { return math.Round(n*1000) / 1000 }

func parseNum(s string) float64 {
	n, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return n
}

// Num coerces FatSecret's string-or-number (or {value}-wrapped) shapes to float64.
func Num(v any) float64 {
	switch t := v.(type) {
	case map[string]any:
		return Num(t["value"])
	case float64:
		return t
	case string:
		return parseNum(t)
	}
	return 0
}

// toServings normalizes the FatSecret single-object-or-array serving shape.
func ToServings(v any) []Serving {
	var out []Serving
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			if s, ok := servingFromMap(item); ok {
				out = append(out, s)
			}
		}
	case map[string]any:
		if s, ok := servingFromMap(t); ok {
			out = append(out, s)
		}
	}
	return out
}

func servingFromMap(m any) (Serving, bool) {
	mm, ok := m.(map[string]any)
	if !ok {
		return Serving{}, false
	}
	str := func(k string) string {
		if v, ok := mm[k].(string); ok {
			return v
		}
		return ""
	}
	return Serving{
		ServingID:              str("serving_id"),
		ServingDescription:     str("serving_description"),
		MeasurementDescription: str("measurement_description"),
		NumberOfUnits:          str("number_of_units"),
		MetricServingAmount:    str("metric_serving_amount"),
		MetricServingUnit:      str("metric_serving_unit"),
		IsDefault:              str("is_default"),
	}, true
}

// chooseServing picks serving_id + number_of_units: explicit serving_id wins;
// grams use a gram-based serving; otherwise the default serving.
func ChooseServing(servings []Serving, servingID *int, numberOfUnits *float64, grams *float64) (int, float64, error) {
	if servingID != nil {
		n := 1.0
		if numberOfUnits != nil {
			n = *numberOfUnits
		}
		return *servingID, n, nil
	}
	if grams != nil {
		var metric []Serving
		for _, s := range servings {
			if s.MetricServingUnit == "g" && parseNum(s.MetricServingAmount) > 0 {
				metric = append(metric, s)
			}
		}
		var pick *Serving
		for i := range metric {
			if metric[i].MeasurementDescription == "g" {
				pick = &metric[i]
				break
			}
		}
		if pick == nil && len(metric) > 0 {
			pick = &metric[0]
		}
		if pick == nil {
			return 0, 0, fmt.Errorf("no gram-based serving for this food; pass serving_id instead")
		}
		units := (*grams / parseNum(pick.MetricServingAmount)) * max1(parseNum(pick.NumberOfUnits))
		id, _ := strconv.Atoi(pick.ServingID)
		return id, round3(units), nil
	}
	var pick *Serving
	for i := range servings {
		if servings[i].IsDefault == "1" {
			pick = &servings[i]
			break
		}
	}
	if pick == nil && len(servings) > 0 {
		pick = &servings[0]
	}
	if pick == nil {
		return 0, 0, fmt.Errorf("food has no servings")
	}
	id, _ := strconv.Atoi(pick.ServingID)
	n := 1.0
	if numberOfUnits != nil {
		n = *numberOfUnits
	}
	return id, n, nil
}

func max1(n float64) float64 {
	if n == 0 {
		return 1
	}
	return n
}

// dayTotals sums calories and macros of a /food-entries/v2 response, overall and per meal.
func DayTotals(response any) map[string]any {
	var entries []map[string]any
	if m, ok := response.(map[string]any); ok {
		if fe, ok := m["food_entries"].(map[string]any); ok {
			switch e := fe["food_entry"].(type) {
			case []any:
				for _, item := range e {
					if em, ok := item.(map[string]any); ok {
						entries = append(entries, em)
					}
				}
			case map[string]any:
				entries = append(entries, e)
			}
		}
	}
	nutrients := []string{"calories", "protein", "fat", "carbohydrate"}
	empty := func() map[string]any {
		m := map[string]any{}
		for _, n := range nutrients {
			m[n] = 0.0
		}
		return m
	}
	total := empty()
	byMeal := map[string]any{}
	for _, e := range entries {
		meal, _ := e["meal"].(string)
		meal = strings.ToLower(meal)
		if meal == "" {
			meal = "other"
		}
		bm, ok := byMeal[meal].(map[string]any)
		if !ok {
			bm = empty()
			byMeal[meal] = bm
		}
		for _, n := range nutrients {
			var v float64
			switch t := e[n].(type) {
			case string:
				v = parseNum(t)
			case float64:
				v = t
			}
			total[n] = round3(total[n].(float64) + v)
			bm[n] = round3(bm[n].(float64) + v)
		}
	}
	return map[string]any{"entries": len(entries), "total": total, "by_meal": byMeal}
}
